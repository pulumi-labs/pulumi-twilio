// Copyright 2026, Pulumi Corporation.
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package provider

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"reflect"
	"strconv"
	"strings"
	"time"
)

// retryBaseDelay is the first backoff step for throttled or failing requests; tests shrink it.
var retryBaseDelay = 500 * time.Millisecond

// Client is a minimal Twilio REST client: form-encoded requests, JSON responses,
// HTTP Basic auth. It covers the handful of endpoints the provider uses.
type Client struct {
	http       *http.Client
	accountSid string
	user, pass string
	edge       string
	region     string
	baseURL    string
}

func newClient(accountSid, user, pass, edge, region, baseURL string) *Client {
	return &Client{
		http:       &http.Client{Timeout: 60 * time.Second},
		accountSid: accountSid, user: user, pass: pass, edge: edge, region: region,
		baseURL: strings.TrimRight(baseURL, "/"),
	}
}

// obj is a decoded JSON response object.
type obj map[string]any

// get returns the value at key k. A dotted key (push.apn_credential_sid) walks
// nested objects; a key that exists literally wins over a dotted reading.
func (o obj) get(k string) any {
	if v, ok := o[k]; ok || !strings.Contains(k, ".") {
		return v
	}
	var cur any = map[string]any(o)
	for _, part := range strings.Split(k, ".") {
		m, ok := cur.(map[string]any)
		if !ok {
			return nil
		}
		cur = m[part]
	}
	return cur
}

func (o obj) str(k string) string {
	switch v := o.get(k).(type) {
	case string:
		return v
	case float64:
		return strconv.FormatFloat(v, 'f', -1, 64)
	}
	return ""
}

func (o obj) optStr(k string) *string {
	if v := o.get(k); v != nil {
		s := o.str(k)
		return &s
	}
	return nil
}

func (o obj) optBool(k string) *bool {
	switch v := o.get(k).(type) {
	case bool:
		return &v
	case string:
		if b, err := strconv.ParseBool(v); err == nil {
			return &b
		}
	}
	return nil
}

func (o obj) optInt(k string) *int {
	switch v := o.get(k).(type) {
	case float64:
		i := int(v)
		return &i
	case string:
		if i, err := strconv.Atoi(v); err == nil {
			return &i
		}
	}
	return nil
}

// optList reads a list of strings, given either as a JSON array or as a
// comma-separated string.
func (o obj) optList(k string) *[]string {
	switch v := o.get(k).(type) {
	case []any:
		out := []string{}
		for _, it := range v {
			if s, ok := it.(string); ok {
				out = append(out, s)
			}
		}
		return &out
	case string:
		out := []string{}
		for _, s := range strings.Split(v, ",") {
			if s = strings.TrimSpace(s); s != "" {
				out = append(out, s)
			}
		}
		return &out
	}
	return nil
}

// apiError is a non-2xx Twilio response.
type apiError struct {
	Status  int
	Code    int
	Message string
}

func (e *apiError) Error() string {
	if e.Code != 0 {
		return fmt.Sprintf("twilio: HTTP %d (error %d): %s", e.Status, e.Code, e.Message)
	}
	return fmt.Sprintf("twilio: HTTP %d: %s", e.Status, e.Message)
}

func isNotFound(err error) bool {
	ae, ok := err.(*apiError)
	return ok && ae.Status == http.StatusNotFound
}

// isRefusal reports whether err is a Twilio 400 with error code 20001 (a generic code) whose
// message holds every one of the given lower case fragments. Matching the text keeps an unrelated
// 20001 a real failure.
func isRefusal(err error, fragments ...string) bool {
	ae, ok := err.(*apiError)
	if !ok || ae.Status != http.StatusBadRequest || ae.Code != 20001 {
		return false
	}
	msg := strings.ToLower(ae.Message)
	for _, fr := range fragments {
		if !strings.Contains(msg, fr) {
			return false
		}
	}
	return true
}

// host returns the base URL for a Twilio product (api, messaging, verify, iam, events, trunking, serverless).
func (c *Client) host(product string) string {
	if c.baseURL != "" {
		return c.baseURL + "/" + product
	}
	parts := []string{product}
	if c.edge != "" {
		parts = append(parts, c.edge)
	}
	if c.region != "" {
		parts = append(parts, c.region)
	}
	return "https://" + strings.Join(parts, ".") + ".twilio.com"
}

// acct returns override when set, else the provider account SID.
func (c *Client) acct(override *string) string {
	return derefOr(override, c.accountSid)
}

// do performs one request and decodes a JSON object response (nil for 204).
// Throttled (429) requests are retried for every method; transient 5xx only
// for GET.
func (c *Client) do(ctx context.Context, method, product, path string, query, form url.Values) (obj, error) {
	if form == nil {
		return c.send(ctx, method, product, path, query, nil, "")
	}
	return c.send(ctx, method, product, path, query, []byte(form.Encode()), "application/x-www-form-urlencoded")
}

// doJSON is do for the Twilio APIs that take a JSON body (Voice v2 configurations). A nil
// body sends none.
func (c *Client) doJSON(ctx context.Context, method, product, path string, query url.Values, body any) (obj, error) {
	if body == nil {
		return c.send(ctx, method, product, path, query, nil, "")
	}
	raw, err := json.Marshal(body)
	if err != nil {
		return nil, fmt.Errorf("twilio: encode %s %s body: %w", method, path, err)
	}
	return c.send(ctx, method, product, path, query, raw, "application/json")
}

// send performs the request with an optional body of the given content type.
func (c *Client) send(ctx context.Context, method, product, path string, query url.Values, payload []byte, contentType string) (obj, error) {
	u := c.host(product) + path
	if len(query) > 0 {
		u += "?" + query.Encode()
	}
	var lastErr error
	for attempt := 0; attempt < 4; attempt++ {
		if attempt > 0 {
			select {
			case <-ctx.Done():
				return nil, ctx.Err()
			case <-time.After(retryBaseDelay << (attempt - 1)):
			}
		}
		var body io.Reader
		if contentType != "" {
			body = strings.NewReader(string(payload))
		}
		req, err := http.NewRequestWithContext(ctx, method, u, body)
		if err != nil {
			return nil, err
		}
		req.SetBasicAuth(c.user, c.pass)
		req.Header.Set("Accept", "application/json")
		if contentType != "" {
			req.Header.Set("Content-Type", contentType)
		}
		resp, err := c.http.Do(req)
		if err != nil {
			lastErr = err
			if method == http.MethodGet {
				continue
			}
			return nil, err
		}
		raw, _ := io.ReadAll(io.LimitReader(resp.Body, 8<<20))
		_ = resp.Body.Close()
		if resp.StatusCode >= 200 && resp.StatusCode < 300 {
			if len(raw) == 0 {
				return nil, nil
			}
			var o obj
			if err := json.Unmarshal(raw, &o); err != nil {
				return nil, fmt.Errorf("twilio: decode %s %s response: %w", method, path, err)
			}
			return o, nil
		}
		ae := &apiError{Status: resp.StatusCode, Message: strings.TrimSpace(string(raw))}
		var eb obj
		if json.Unmarshal(raw, &eb) == nil {
			if m := eb.str("message"); m != "" {
				ae.Message = m
			}
			if cd := eb.optInt("code"); cd != nil {
				ae.Code = *cd
			}
		}
		lastErr = ae
		if resp.StatusCode == http.StatusTooManyRequests || (method == http.MethodGet && resp.StatusCode >= 500) {
			continue
		}
		return nil, ae
	}
	return nil, lastErr
}

// form helpers: only set values the caller supplied.

func setS(v url.Values, k string, p *string) {
	if p != nil {
		v.Set(k, *p)
	}
}

func setB(v url.Values, k string, p *bool) {
	if p != nil {
		v.Set(k, strconv.FormatBool(*p))
	}
}

func setI(v url.Values, k string, p *int) {
	if p != nil {
		v.Set(k, strconv.Itoa(*p))
	}
}

// api2010 returns the account-scoped path prefix of the 2010-04-01 API.
func api2010(account string) string {
	return "/2010-04-01/Accounts/" + url.PathEscape(account)
}

// notFoundOnImport reports a missing remote object during an import.
//
// The engine treats an empty ID from Read as "deleted" only on refresh; on import the
// framework accepts the empty response and an empty resource lands in state with exit
// code 0. An import is recognisable by Read arriving with neither prior inputs nor prior
// state, so there a missing object is an error. On refresh it returns nil and the caller
// returns the empty response that drops the resource from state.
func notFoundOnImport[A, S any](kind, id string, inputs A, state S) error {
	if reflect.ValueOf(inputs).IsZero() && reflect.ValueOf(state).IsZero() {
		return fmt.Errorf("cannot import %s %q: it does not exist (or the API key cannot see it); check the import ID", kind, id)
	}
	return nil
}

// redirectTarget performs a GET that Twilio answers with a redirect and returns the target without
// following it: the Location header of a 3xx, else bodyKey of a JSON body (a 2xx). The target of
// some redirects is an authenticated URL of a file, so it must reach the caller, not be fetched.
// Throttled (429) and failing (5xx) requests are retried like any GET.
func (c *Client) redirectTarget(ctx context.Context, product, path string, query url.Values, bodyKey string) (string, error) {
	u := c.host(product) + path
	if len(query) > 0 {
		u += "?" + query.Encode()
	}
	hc := *c.http
	hc.CheckRedirect = func(*http.Request, []*http.Request) error { return http.ErrUseLastResponse }
	var lastErr error
	for attempt := 0; attempt < 4; attempt++ {
		if attempt > 0 {
			select {
			case <-ctx.Done():
				return "", ctx.Err()
			case <-time.After(retryBaseDelay << (attempt - 1)):
			}
		}
		req, err := http.NewRequestWithContext(ctx, http.MethodGet, u, nil)
		if err != nil {
			return "", err
		}
		req.SetBasicAuth(c.user, c.pass)
		req.Header.Set("Accept", "application/json")
		resp, err := hc.Do(req)
		if err != nil {
			lastErr = err
			continue
		}
		raw, _ := io.ReadAll(io.LimitReader(resp.Body, 8<<20))
		_ = resp.Body.Close()
		var eb obj
		_ = json.Unmarshal(raw, &eb)
		switch {
		case resp.StatusCode >= 300 && resp.StatusCode < 400:
			if loc := resp.Header.Get("Location"); loc != "" {
				return loc, nil
			}
			if t := eb.str(bodyKey); t != "" {
				return t, nil
			}
			return "", fmt.Errorf("twilio: HTTP %d without a redirect location", resp.StatusCode)
		case resp.StatusCode >= 200 && resp.StatusCode < 300:
			if t := eb.str(bodyKey); t != "" {
				return t, nil
			}
			return "", fmt.Errorf("twilio: HTTP %d without a redirect location", resp.StatusCode)
		}
		ae := &apiError{Status: resp.StatusCode, Message: strings.TrimSpace(string(raw))}
		if m := eb.str("message"); m != "" {
			ae.Message = m
		}
		if cd := eb.optInt("code"); cd != nil {
			ae.Code = *cd
		}
		lastErr = ae
		if resp.StatusCode == http.StatusTooManyRequests || resp.StatusCode >= 500 {
			continue
		}
		return "", ae
	}
	return "", lastErr
}
