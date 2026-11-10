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
	"net/http"
	"net/url"
	"sort"
	"strconv"
	"strings"
)

// The Verify v2 extras, served for the rate limit, messaging configuration, webhook, SafeList and
// template tests:
//
//	/verify/v2/Services/{svc}/RateLimits[/{sid}]
//	/verify/v2/Services/{svc}/RateLimits/{sid}/Buckets[/{sid}]
//	/verify/v2/Services/{svc}/MessagingConfigurations[/{country}]
//	/verify/v2/Services/{svc}/Webhooks[/{sid}]
//	/verify/v2/SafeList/Numbers[/{phone}]
//	/verify/v2/Templates
//
// The shapes follow twilio-oai spec/json/twilio_verify_v2.json (spec version 1.0.0).

const (
	verifyStamp        = "2026-11-06T00:00:00Z"
	verifyUpdatedStamp = "2026-11-07T00:00:00Z"
)

func (f *fakeTwilio) handleVerifyChild(w http.ResponseWriter, r *http.Request, form url.Values, parts []string) {
	notFound := func() { ferr(w, 404, "The requested resource "+r.URL.Path+" was not found") }
	svc := parts[0]
	if _, ok := f.objs[key("verify", testAccount, svc)]; !ok {
		notFound()
		return
	}
	sid := ""
	switch {
	case len(parts) == 3:
		sid = parts[2]
	case len(parts) == 2:
	case len(parts) == 5 && parts[1] == "RateLimits" && parts[3] == "Buckets":
		sid = parts[4]
	case len(parts) == 4 && parts[1] == "RateLimits" && parts[3] == "Buckets":
	default:
		notFound()
		return
	}
	switch {
	case parts[1] == "RateLimits" && len(parts) <= 3:
		f.verifyRateLimit(w, r, form, svc, sid, notFound)
	case parts[1] == "RateLimits":
		if _, ok := f.objs[key("vrl", testAccount, svc+"/"+parts[2])]; !ok {
			notFound()
			return
		}
		f.verifyBucket(w, r, form, svc, parts[2], sid, notFound)
	case parts[1] == "MessagingConfigurations" && len(parts) <= 3:
		f.verifyMessagingConfig(w, r, form, svc, sid, notFound)
	case parts[1] == "Webhooks" && len(parts) <= 3:
		f.verifyWebhook(w, r, form, svc, sid, notFound)
	default:
		notFound()
	}
}

func (f *fakeTwilio) verifyRateLimit(w http.ResponseWriter, r *http.Request, form url.Values, svc, sid string, notFound func()) {
	if sid == "" {
		if r.Method != "POST" {
			ferr(w, 405, "method not allowed")
			return
		}
		name := form.Get("UniqueName")
		if name == "" {
			ferr(w, 400, "Missing required parameter UniqueName")
			return
		}
		for _, o := range f.objs {
			if o["service_sid"] == svc && o["unique_name"] == name && strings.HasPrefix(o["sid"].(string), "RK") {
				ferr(w, 409, "A rate limit with this unique name already exists")
				return
			}
		}
		o := map[string]any{"sid": f.sid("RK"), "service_sid": svc, "account_sid": testAccount, "unique_name": name,
			"description": nil, "date_created": verifyStamp, "date_updated": verifyStamp}
		if d, ok := form["Description"]; ok {
			o["description"] = d[0]
		}
		f.put("vrl", testAccount, svc+"/"+o["sid"].(string), o)
		fwrite(w, 201, o)
		return
	}
	o, ok := f.objs[key("vrl", testAccount, svc+"/"+sid)]
	if !ok {
		notFound()
		return
	}
	switch r.Method {
	case "GET":
		fwrite(w, 200, o)
	case "POST":
		if d, ok := form["Description"]; ok {
			o["description"] = d[0]
		}
		o["date_updated"] = verifyUpdatedStamp
		fwrite(w, 200, o)
	case "DELETE":
		for k, c := range f.objs {
			if c["rate_limit_sid"] == sid {
				delete(f.objs, k)
			}
		}
		f.del("vrl", testAccount, svc+"/"+sid)
		w.WriteHeader(204)
	default:
		ferr(w, 405, "method not allowed")
	}
}

func (f *fakeTwilio) verifyBucket(w http.ResponseWriter, r *http.Request, form url.Values, svc, rl, sid string, notFound func()) {
	apply := func(o map[string]any) string {
		for param, k := range map[string]string{"Max": "max", "Interval": "interval"} {
			if v, ok := form[param]; ok {
				n, err := strconv.Atoi(v[0])
				if err != nil {
					return param + " must be an integer"
				}
				o[k] = float64(n)
			}
		}
		return ""
	}
	if sid == "" {
		if r.Method != "POST" {
			ferr(w, 405, "method not allowed")
			return
		}
		for _, req := range []string{"Max", "Interval"} {
			if form.Get(req) == "" {
				ferr(w, 400, "Missing required parameter "+req)
				return
			}
		}
		o := map[string]any{"sid": f.sid("BL"), "rate_limit_sid": rl, "service_sid": svc, "account_sid": testAccount,
			"date_created": verifyStamp, "date_updated": verifyStamp}
		if msg := apply(o); msg != "" {
			ferr(w, 400, msg)
			return
		}
		f.put("vbk", testAccount, svc+"/"+rl+"/"+o["sid"].(string), o)
		fwrite(w, 201, o)
		return
	}
	o, ok := f.objs[key("vbk", testAccount, svc+"/"+rl+"/"+sid)]
	if !ok {
		notFound()
		return
	}
	switch r.Method {
	case "GET":
		fwrite(w, 200, o)
	case "POST":
		if msg := apply(o); msg != "" {
			ferr(w, 400, msg)
			return
		}
		o["date_updated"] = verifyUpdatedStamp
		fwrite(w, 200, o)
	case "DELETE":
		f.del("vbk", testAccount, svc+"/"+rl+"/"+sid)
		w.WriteHeader(204)
	default:
		ferr(w, 405, "method not allowed")
	}
}

func (f *fakeTwilio) verifyMessagingConfig(w http.ResponseWriter, r *http.Request, form url.Values, svc, country string, notFound func()) {
	if country == "" {
		if r.Method != "POST" {
			ferr(w, 405, "method not allowed")
			return
		}
		for _, req := range []string{"Country", "MessagingServiceSid"} {
			if form.Get(req) == "" {
				ferr(w, 400, "Missing required parameter "+req)
				return
			}
		}
		c := form.Get("Country")
		if f.verifyMsgCfgPhantomCreate {
			fwrite(w, 200, map[string]any{"service_sid": svc, "account_sid": testAccount, "country": "", "messaging_service_sid": "MG" + strings.Repeat("f", 32),
				"date_created": verifyStamp, "date_updated": verifyStamp})
			return
		}
		if _, dup := f.objs[key("vmc", testAccount, svc+"/"+c)]; dup {
			ferr(w, 409, "A configuration for this country already exists")
			return
		}
		o := map[string]any{"service_sid": svc, "account_sid": testAccount, "country": c, "messaging_service_sid": form.Get("MessagingServiceSid"),
			"date_created": verifyStamp, "date_updated": verifyStamp}
		f.put("vmc", testAccount, svc+"/"+c, o)
		fwrite(w, 201, o)
		return
	}
	o, ok := f.objs[key("vmc", testAccount, svc+"/"+country)]
	if !ok && r.Method == "GET" && f.verifyMsgCfgFake200 {
		fwrite(w, 200, map[string]any{"service_sid": svc, "account_sid": testAccount, "country": "", "messaging_service_sid": "MG" + strings.Repeat("f", 32),
			"date_created": verifyStamp, "date_updated": verifyStamp})
		return
	}
	if !ok {
		notFound()
		return
	}
	switch r.Method {
	case "GET":
		fwrite(w, 200, o)
	case "POST":
		m := form.Get("MessagingServiceSid")
		if m == "" {
			ferr(w, 400, "Missing required parameter MessagingServiceSid")
			return
		}
		o["messaging_service_sid"] = m
		o["date_updated"] = verifyUpdatedStamp
		fwrite(w, 200, o)
	case "DELETE":
		f.del("vmc", testAccount, svc+"/"+country)
		w.WriteHeader(204)
	default:
		ferr(w, 405, "method not allowed")
	}
}

func (f *fakeTwilio) verifyWebhook(w http.ResponseWriter, r *http.Request, form url.Values, svc, sid string, notFound func()) {
	apply := func(o map[string]any) string {
		if v, ok := form["FriendlyName"]; ok {
			o["friendly_name"] = v[0]
		}
		if v, ok := form["WebhookUrl"]; ok {
			o["webhook_url"] = v[0]
		}
		if v, ok := form["EventTypes"]; ok {
			// Twilio does not promise to return the event types in the order they were posted.
			t := append([]string(nil), v...)
			sort.Strings(t)
			l := make([]any, len(t))
			for i, s := range t {
				l[i] = s
			}
			o["event_types"] = l
		}
		if v, ok := form["Status"]; ok {
			if v[0] != "enabled" && v[0] != "disabled" {
				return "Invalid Status " + v[0]
			}
			o["status"] = v[0]
		}
		if v, ok := form["Version"]; ok {
			if v[0] != "v1" && v[0] != "v2" {
				return "Invalid Version " + v[0]
			}
			o["version"] = v[0]
		}
		return ""
	}
	if sid == "" {
		if r.Method != "POST" {
			ferr(w, 405, "method not allowed")
			return
		}
		for _, req := range []string{"FriendlyName", "EventTypes", "WebhookUrl"} {
			if form.Get(req) == "" {
				ferr(w, 400, "Missing required parameter "+req)
				return
			}
		}
		o := map[string]any{"sid": f.sid("YW"), "service_sid": svc, "account_sid": testAccount, "status": "enabled", "version": "v2",
			"webhook_method": "POST", "date_created": verifyStamp, "date_updated": verifyStamp}
		if msg := apply(o); msg != "" {
			ferr(w, 400, msg)
			return
		}
		f.put("vwh", testAccount, svc+"/"+o["sid"].(string), o)
		fwrite(w, 201, o)
		return
	}
	o, ok := f.objs[key("vwh", testAccount, svc+"/"+sid)]
	if !ok {
		notFound()
		return
	}
	switch r.Method {
	case "GET":
		fwrite(w, 200, o)
	case "POST":
		if msg := apply(o); msg != "" {
			ferr(w, 400, msg)
			return
		}
		o["date_updated"] = verifyUpdatedStamp
		fwrite(w, 200, o)
	case "DELETE":
		f.del("vwh", testAccount, svc+"/"+sid)
		w.WriteHeader(204)
	default:
		ferr(w, 405, "method not allowed")
	}
}

// handleVerifySafeList serves /v2/SafeList/Numbers. Twilio documents the plus sign of the number as
// %2B in the path, so the fake insists on it.
func (f *fakeTwilio) handleVerifySafeList(w http.ResponseWriter, r *http.Request, form url.Values, number string) {
	if number == "" {
		if r.Method != "POST" {
			ferr(w, 405, "method not allowed")
			return
		}
		n := form.Get("PhoneNumber")
		if n == "" {
			ferr(w, 400, "Missing required parameter PhoneNumber")
			return
		}
		if _, dup := f.objs[key("vsl", testAccount, n)]; dup {
			ferr(w, 409, "The number is already on the SafeList")
			return
		}
		o := map[string]any{"sid": f.sid("GN"), "phone_number": n}
		f.put("vsl", testAccount, n, o)
		fwrite(w, 201, o)
		return
	}
	if !strings.Contains(r.URL.EscapedPath(), "%2B") {
		ferr(w, 400, "the plus sign of the phone number must be escaped as %2B: "+r.URL.EscapedPath())
		return
	}
	o, ok := f.objs[key("vsl", testAccount, number)]
	if !ok {
		ferr(w, 404, "The requested resource "+r.URL.Path+" was not found")
		return
	}
	switch r.Method {
	case "GET":
		fwrite(w, 200, o)
	case "DELETE":
		f.del("vsl", testAccount, number)
		w.WriteHeader(204)
	default:
		ferr(w, 405, "method not allowed")
	}
}

// addVerifyTemplate seeds a Verify template.
func (f *fakeTwilio) addVerifyTemplate(sid, name string, channels ...string) {
	l := make([]any, len(channels))
	for i, c := range channels {
		l[i] = c
	}
	f.put("vtpl", testAccount, sid, map[string]any{"sid": sid, "account_sid": testAccount, "friendly_name": name, "channels": l,
		"translations": map[string]any{}})
}

// handleVerifyTemplates serves GET /v2/Templates with the FriendlyName filter and pagination.
func (f *fakeTwilio) handleVerifyTemplates(w http.ResponseWriter, r *http.Request) {
	if r.Method != "GET" {
		ferr(w, 405, "method not allowed")
		return
	}
	q := r.URL.Query()
	var all []map[string]any
	for k, o := range f.objs {
		if strings.HasPrefix(k, "vtpl/") && (q.Get("FriendlyName") == "" || o["friendly_name"] == q.Get("FriendlyName")) {
			all = append(all, o)
		}
	}
	sort.Slice(all, func(i, j int) bool { return all[i]["sid"].(string) < all[j]["sid"].(string) })
	size, page := 50, 0
	if v, err := strconv.Atoi(q.Get("PageSize")); err == nil && v > 0 {
		size = v
	}
	if f.templatePageSize > 0 {
		size = f.templatePageSize
	}
	if v, err := strconv.Atoi(q.Get("Page")); err == nil && v > 0 {
		page = v
	}
	lo, hi := page*size, (page+1)*size
	if lo > len(all) {
		lo = len(all)
	}
	if hi > len(all) {
		hi = len(all)
	}
	items := make([]any, 0, hi-lo)
	for _, o := range all[lo:hi] {
		items = append(items, o)
	}
	meta := map[string]any{"page": float64(page), "page_size": float64(size), "key": "templates", "next_page_url": nil}
	if hi < len(all) {
		nq := url.Values{"PageSize": {strconv.Itoa(size)}, "Page": {strconv.Itoa(page + 1)}}
		if n := q.Get("FriendlyName"); n != "" {
			nq.Set("FriendlyName", n)
		}
		meta["next_page_url"] = "https://verify.twilio.com/v2/Templates?" + nq.Encode()
	}
	fwrite(w, 200, map[string]any{"templates": items, "meta": meta})
}
