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

func (o obj) str(k string) string {
	switch v := o[k].(type) {
	case string:
		return v
	case float64:
		return strconv.FormatFloat(v, 'f', -1, 64)
	}
	return ""
}

func (o obj) optStr(k string) *string {
	if v, ok := o[k]; ok && v != nil {
		s := o.str(k)
		return &s
	}
	return nil
}

func (o obj) optBool(k string) *bool {
	switch v := o[k].(type) {
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
	switch v := o[k].(type) {
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

// host returns the base URL for a Twilio product (api, messaging, verify, iam).
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
		if form != nil {
			body = strings.NewReader(form.Encode())
		}
		req, err := http.NewRequestWithContext(ctx, method, u, body)
		if err != nil {
			return nil, err
		}
		req.SetBasicAuth(c.user, c.pass)
		req.Header.Set("Accept", "application/json")
		if form != nil {
			req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
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
