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
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strconv"
	"strings"
	"sync"
	"testing"
)

// fakeTwilio is an in-memory fake of the Twilio REST surface the provider uses
// (api 2010-04-01 accounts, applications, incoming phone numbers and classic
// keys; iam v1 keys; messaging v1 services and senders; verify v2 services).
// It follows the real API's semantics where the provider's assumptions differ
// from a naive mock:
//
//   - responses use the real shapes: unset URLs are null, booleans and integers
//     are JSON booleans and numbers, defaulted methods read back as "POST",
//     the policy of a restricted key is a JSON object;
//   - HTTP-method enums reject the empty string, and KeyType accepts only
//     "restricted" (omit it for a standard key);
//   - account-scoped objects (applications, numbers, classic keys) live under
//     their owning account and are 404 under any other account;
//   - the IAM key route has no account in its path and, as the worst case the
//     provider must survive, only sees keys of the authenticated account;
//   - a number is moved between accounts by posting AccountSid to the account
//     that currently owns it, and a top-level account cannot be closed.
type fakeTwilio struct {
	t      *testing.T
	server *httptest.Server

	mu      sync.Mutex
	objs    map[string]map[string]any // "kind/account/sid" -> object; ordered by seq
	order   []string
	next    int
	calls   []fakeCall
	failure map[string]int // "METHOD substring" -> status to return once
}

type fakeCall struct {
	Method, Path string
	Form         url.Values
}

type fakeField struct{ param, key, typ string } // typ: s string, u url, m method enum, b bool, i int

const (
	fakeUser = testAccount
	fakePass = testToken
	fakeSub  = "ACsub0000000000000000000000000000"
)

var (
	appFields = []fakeField{
		{"FriendlyName", "friendly_name", "s"},
		{"VoiceUrl", "voice_url", "u"}, {"VoiceMethod", "voice_method", "m"},
		{"VoiceFallbackUrl", "voice_fallback_url", "u"}, {"VoiceFallbackMethod", "voice_fallback_method", "m"},
		{"StatusCallback", "status_callback", "u"}, {"StatusCallbackMethod", "status_callback_method", "m"},
		{"SmsUrl", "sms_url", "u"}, {"SmsMethod", "sms_method", "m"},
		{"SmsFallbackUrl", "sms_fallback_url", "u"}, {"SmsFallbackMethod", "sms_fallback_method", "m"},
		{"SmsStatusCallback", "sms_status_callback", "u"}, {"MessageStatusCallback", "message_status_callback", "u"},
		{"VoiceCallerIdLookup", "voice_caller_id_lookup", "b"},
	}
	numberFields = []fakeField{
		{"FriendlyName", "friendly_name", "s"},
		{"VoiceUrl", "voice_url", "u"}, {"VoiceMethod", "voice_method", "m"},
		{"VoiceFallbackUrl", "voice_fallback_url", "u"}, {"VoiceFallbackMethod", "voice_fallback_method", "m"},
		{"StatusCallback", "status_callback", "u"}, {"StatusCallbackMethod", "status_callback_method", "m"},
		{"SmsUrl", "sms_url", "u"}, {"SmsMethod", "sms_method", "m"},
		{"SmsFallbackUrl", "sms_fallback_url", "u"}, {"SmsFallbackMethod", "sms_fallback_method", "m"},
		{"VoiceApplicationSid", "voice_application_sid", "u"}, {"SmsApplicationSid", "sms_application_sid", "u"},
		{"VoiceCallerIdLookup", "voice_caller_id_lookup", "b"},
	}
	msgFields = []fakeField{
		{"FriendlyName", "friendly_name", "s"},
		{"InboundRequestUrl", "inbound_request_url", "u"}, {"InboundMethod", "inbound_method", "m"},
		{"FallbackUrl", "fallback_url", "u"}, {"FallbackMethod", "fallback_method", "m"},
		{"StatusCallback", "status_callback", "u"}, {"ScanMessageContent", "scan_message_content", "s"},
		{"StickySender", "sticky_sender", "b"}, {"MmsConverter", "mms_converter", "b"},
		{"SmartEncoding", "smart_encoding", "b"}, {"FallbackToLongCode", "fallback_to_long_code", "b"},
		{"AreaCodeGeomatch", "area_code_geomatch", "b"}, {"SynchronousValidation", "synchronous_validation", "b"},
		{"UseInboundWebhookOnNumber", "use_inbound_webhook_on_number", "b"}, {"ValidityPeriod", "validity_period", "i"},
	}
	verifyFields = []fakeField{
		{"FriendlyName", "friendly_name", "s"}, {"CodeLength", "code_length", "i"},
		{"LookupEnabled", "lookup_enabled", "b"}, {"SkipSmsToLandlines", "skip_sms_to_landlines", "b"},
		{"DtmfInputRequired", "dtmf_input_required", "b"}, {"TtsName", "tts_name", "s"},
		{"Psd2Enabled", "psd2_enabled", "b"}, {"DoNotShareWarningEnabled", "do_not_share_warning_enabled", "b"},
		{"CustomCodeEnabled", "custom_code_enabled", "b"},
	}
)

func startFakeTwilio(t *testing.T) *fakeTwilio {
	t.Helper()
	f := &fakeTwilio{t: t, objs: map[string]map[string]any{}, failure: map[string]int{}}
	f.server = httptest.NewServer(http.HandlerFunc(f.handle))
	t.Cleanup(f.server.Close)
	return f
}

func fwrite(w http.ResponseWriter, code int, v any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(code)
	_ = json.NewEncoder(w).Encode(v)
}

func ferr(w http.ResponseWriter, code int, msg string) {
	fwrite(w, code, map[string]any{"code": 20000 + code, "message": msg, "status": code})
}

func key(kind, acct, sid string) string { return kind + "/" + acct + "/" + sid }

func (f *fakeTwilio) put(kind, acct, sid string, o map[string]any) {
	k := key(kind, acct, sid)
	if _, ok := f.objs[k]; !ok {
		f.order = append(f.order, k)
	}
	f.objs[k] = o
}

func (f *fakeTwilio) del(kind, acct, sid string) { delete(f.objs, key(kind, acct, sid)) }

func (f *fakeTwilio) sid(prefix string) string {
	f.next++
	return fmt.Sprintf("%s%032d", prefix, f.next)[:34]
}

func (f *fakeTwilio) accountKnown(acct string) bool {
	if acct == testAccount {
		return true
	}
	o, ok := f.objs[key("account", testAccount, acct)]
	return ok && o["owner_account_sid"] == testAccount
}

// applyForm applies the posted fields; it returns an API error message on a
// value the real API rejects.
func applyForm(o map[string]any, form url.Values, fields []fakeField) string {
	for _, fl := range fields {
		vals, ok := form[fl.param]
		if !ok {
			continue
		}
		v := vals[0]
		switch fl.typ {
		case "s", "u":
			if v == "" && fl.typ == "u" {
				o[fl.key] = nil
			} else {
				o[fl.key] = v
			}
		case "m":
			if v != "GET" && v != "POST" {
				return fmt.Sprintf("Invalid value for %s: %q (must be GET or POST)", fl.param, v)
			}
			o[fl.key] = v
		case "b":
			b, err := strconv.ParseBool(v)
			if err != nil {
				return "Invalid boolean for " + fl.param
			}
			o[fl.key] = b
		case "i":
			n, err := strconv.Atoi(v)
			if err != nil {
				return "Invalid integer for " + fl.param
			}
			o[fl.key] = float64(n)
		}
	}
	return ""
}

func (f *fakeTwilio) handle(w http.ResponseWriter, r *http.Request) {
	f.mu.Lock()
	defer f.mu.Unlock()
	if u, p, ok := r.BasicAuth(); !ok || u != fakeUser || p != fakePass {
		ferr(w, 401, "Authenticate")
		return
	}
	body, _ := io.ReadAll(r.Body)
	form, _ := url.ParseQuery(string(body))
	f.calls = append(f.calls, fakeCall{Method: r.Method, Path: r.URL.Path, Form: form})
	for k, status := range f.failure {
		parts := strings.SplitN(k, " ", 2)
		if parts[0] == r.Method && strings.Contains(r.URL.Path, parts[1]) {
			delete(f.failure, k)
			ferr(w, status, "injected failure")
			return
		}
	}
	path := r.URL.Path
	switch {
	case strings.HasPrefix(path, "/api/2010-04-01/"):
		f.handleAPI(w, r, form, strings.Split(strings.TrimPrefix(path, "/api/2010-04-01/"), "/"))
	case strings.HasPrefix(path, "/iam/v1/Keys"):
		f.handleIAMKeys(w, r, form, strings.Trim(strings.TrimPrefix(path, "/iam/v1/Keys"), "/"))
	case strings.HasPrefix(path, "/messaging/v1/Services"):
		f.handleMessaging(w, r, form, strings.Split(strings.Trim(strings.TrimPrefix(path, "/messaging/v1/Services"), "/"), "/"))
	case strings.HasPrefix(path, "/verify/v2/Services"):
		f.handleVerify(w, r, form, strings.Trim(strings.TrimPrefix(path, "/verify/v2/Services"), "/"))
	default:
		ferr(w, 404, "no route "+path)
	}
}

func clone(o map[string]any) map[string]any {
	c := make(map[string]any, len(o))
	for k, v := range o {
		c[k] = v
	}
	return c
}

func trimJSON(s string) string { return strings.TrimSuffix(s, ".json") }

func (f *fakeTwilio) handleAPI(w http.ResponseWriter, r *http.Request, form url.Values, parts []string) {
	if parts[0] != "Accounts.json" && parts[0] != "Accounts" {
		ferr(w, 404, "not found")
		return
	}
	switch len(parts) {
	case 1: // /Accounts.json
		if r.Method != "POST" {
			ferr(w, 405, "method not allowed")
			return
		}
		sid := f.sid("AC")
		o := map[string]any{"sid": sid, "friendly_name": form.Get("FriendlyName"), "status": "active",
			"auth_token": "tok-" + sid, "owner_account_sid": testAccount, "type": "Full",
			"date_created": "Mon, 02 Nov 2026 00:00:00 +0000"}
		f.put("account", testAccount, sid, o)
		fwrite(w, 201, o)
	case 2: // /Accounts/{sid}.json
		sid := trimJSON(parts[1])
		if sid == testAccount {
			main := map[string]any{"sid": testAccount, "friendly_name": "main", "status": "active", "type": "Full",
				"owner_account_sid": testAccount, "date_created": "Mon, 01 Jun 2026 00:00:00 +0000"}
			switch r.Method {
			case "GET":
				fwrite(w, 200, main)
			case "POST":
				if form.Get("Status") == "closed" {
					ferr(w, 403, "A top-level account cannot be closed")
					return
				}
				fwrite(w, 200, main)
			default:
				ferr(w, 405, "method not allowed")
			}
			return
		}
		o, ok := f.objs[key("account", testAccount, sid)]
		if !ok {
			ferr(w, 404, "The requested resource /Accounts/"+sid+".json was not found")
			return
		}
		switch r.Method {
		case "GET":
			fwrite(w, 200, o)
		case "POST":
			if v, ok := form["FriendlyName"]; ok {
				o["friendly_name"] = v[0]
			}
			if v, ok := form["Status"]; ok {
				if v[0] != "active" && v[0] != "suspended" && v[0] != "closed" {
					ferr(w, 400, "Invalid status")
					return
				}
				o["status"] = v[0]
			}
			fwrite(w, 200, o)
		default:
			ferr(w, 405, "method not allowed")
		}
	case 3, 4:
		f.handleScoped(w, r, form, parts)
	default:
		ferr(w, 404, "not found")
	}
}

func (f *fakeTwilio) handleScoped(w http.ResponseWriter, r *http.Request, form url.Values, parts []string) {
	acct := parts[1]
	coll := trimJSON(parts[2])
	if coll == "Balance" && len(parts) == 3 && r.Method == "GET" {
		if !f.accountKnown(acct) {
			ferr(w, 404, "not found")
			return
		}
		fwrite(w, 200, map[string]any{"account_sid": acct, "balance": "12.3400", "currency": "USD"})
		return
	}
	kinds := map[string]struct {
		kind   string
		prefix string
		fields []fakeField
	}{
		"Applications":         {"app", "AP", appFields},
		"IncomingPhoneNumbers": {"number", "PN", numberFields},
		"Keys":                 {"key", "SK", []fakeField{{"FriendlyName", "friendly_name", "s"}}},
	}
	k, ok := kinds[coll]
	if !ok || !f.accountKnown(acct) {
		ferr(w, 404, "The requested resource "+r.URL.Path+" was not found")
		return
	}
	if len(parts) == 3 { // collection
		switch {
		case r.Method == "POST" && coll == "Applications":
			o := map[string]any{"sid": f.sid("AP"), "account_sid": acct, "friendly_name": nil, "voice_url": nil,
				"voice_method": "POST", "voice_fallback_url": nil, "voice_fallback_method": "POST", "status_callback": nil,
				"status_callback_method": "POST", "sms_url": nil, "sms_method": "POST", "sms_fallback_url": nil,
				"sms_fallback_method": "POST", "sms_status_callback": nil, "message_status_callback": nil,
				"voice_caller_id_lookup": false, "date_created": "Mon, 02 Nov 2026 00:00:00 +0000"}
			if msg := applyForm(o, form, k.fields); msg != "" {
				ferr(w, 400, msg)
				return
			}
			f.put("app", acct, o["sid"].(string), o)
			fwrite(w, 201, o)
		case r.Method == "GET" && coll == "IncomingPhoneNumbers":
			want := r.URL.Query().Get("PhoneNumber")
			list := []any{}
			for _, ok := range f.order {
				o, live := f.objs[ok]
				if live && strings.HasPrefix(ok, "number/"+acct+"/") && (want == "" || o["phone_number"] == want) {
					list = append(list, o)
				}
			}
			fwrite(w, 200, map[string]any{"incoming_phone_numbers": list})
		default:
			ferr(w, 405, "method not allowed")
		}
		return
	}
	sid := trimJSON(parts[3])
	o, ok := f.objs[key(k.kind, acct, sid)]
	if !ok {
		ferr(w, 404, "The requested resource "+r.URL.Path+" was not found")
		return
	}
	switch r.Method {
	case "GET":
		fwrite(w, 200, o)
	case "DELETE":
		if k.kind == "number" {
			ferr(w, 405, "numbers are released through a separate flow")
			return
		}
		f.del(k.kind, acct, sid)
		w.WriteHeader(204)
	case "POST":
		if msg := applyForm(o, form, k.fields); msg != "" {
			ferr(w, 400, msg)
			return
		}
		if k.kind == "number" {
			if target := form.Get("AccountSid"); target != "" && target != acct {
				if !f.accountKnown(target) {
					ferr(w, 400, "Account "+target+" is not a subaccount of "+testAccount)
					return
				}
				f.del("number", acct, sid)
				o["account_sid"] = target
				f.put("number", target, sid, o)
			}
		}
		fwrite(w, 200, o)
	default:
		ferr(w, 405, "method not allowed")
	}
}

func (f *fakeTwilio) handleIAMKeys(w http.ResponseWriter, r *http.Request, form url.Values, rest string) {
	if rest == "" {
		if r.Method != "POST" {
			ferr(w, 405, "method not allowed")
			return
		}
		acct := form.Get("AccountSid")
		if acct == "" {
			acct = testAccount
		}
		if !f.accountKnown(acct) {
			ferr(w, 400, "Invalid AccountSid")
			return
		}
		kt := form.Get("KeyType")
		if kt != "" && kt != "restricted" {
			ferr(w, 400, fmt.Sprintf("KeyType %q is not valid: omit it for a standard key or send restricted", kt))
			return
		}
		var pol any
		if kt == "restricted" {
			var p map[string]any
			if json.Unmarshal([]byte(form.Get("Policy")), &p) != nil || len(p) == 0 {
				ferr(w, 400, "A restricted key needs a JSON Policy")
				return
			}
			pol = p
		} else if form.Get("Policy") != "" {
			ferr(w, 400, "Policy is only valid for a restricted key")
			return
		}
		sid := f.sid("SK")
		o := map[string]any{"sid": sid, "account_sid": acct, "friendly_name": form.Get("FriendlyName"), "policy": pol,
			"secret": "secret-" + sid, "date_created": "Mon, 02 Nov 2026 00:00:00 +0000"}
		f.put("key", acct, sid, o)
		fwrite(w, 201, o)
		return
	}
	// The IAM instance route has no account in its path: only the authenticated
	// account's keys are visible through it.
	o, ok := f.objs[key("key", testAccount, rest)]
	if !ok {
		ferr(w, 404, "The requested resource /v1/Keys/"+rest+" was not found")
		return
	}
	switch r.Method {
	case "GET":
		c := clone(o)
		delete(c, "secret")
		fwrite(w, 200, c)
	case "POST":
		if v, ok := form["FriendlyName"]; ok {
			o["friendly_name"] = v[0]
		}
		if v, ok := form["Policy"]; ok {
			var p map[string]any
			if o["policy"] == nil || json.Unmarshal([]byte(v[0]), &p) != nil {
				ferr(w, 400, "Invalid policy update")
				return
			}
			o["policy"] = p
		}
		c := clone(o)
		delete(c, "secret")
		fwrite(w, 200, c)
	case "DELETE":
		f.del("key", testAccount, rest)
		w.WriteHeader(204)
	default:
		ferr(w, 405, "method not allowed")
	}
}

func (f *fakeTwilio) handleMessaging(w http.ResponseWriter, r *http.Request, form url.Values, parts []string) {
	if len(parts) == 1 && parts[0] == "" {
		if r.Method != "POST" {
			ferr(w, 405, "method not allowed")
			return
		}
		if len(form.Get("FriendlyName")) > 64 || form.Get("FriendlyName") == "" {
			ferr(w, 400, "FriendlyName must be 1-64 characters")
			return
		}
		o := map[string]any{"sid": f.sid("MG"), "account_sid": testAccount, "friendly_name": nil,
			"inbound_request_url": nil, "inbound_method": "POST", "fallback_url": nil, "fallback_method": "POST",
			"status_callback": nil, "scan_message_content": "inherit", "sticky_sender": true, "mms_converter": true,
			"smart_encoding": false, "fallback_to_long_code": true, "area_code_geomatch": true,
			"synchronous_validation": false, "validity_period": float64(36000), "use_inbound_webhook_on_number": true,
			"date_created": "Mon, 02 Nov 2026 00:00:00 +0000"}
		if msg := f.applyMessaging(o, form); msg != "" {
			ferr(w, 400, msg)
			return
		}
		f.put("msg", testAccount, o["sid"].(string), o)
		fwrite(w, 201, o)
		return
	}
	svc := parts[0]
	o, ok := f.objs[key("msg", testAccount, svc)]
	if !ok {
		ferr(w, 404, "The requested resource /v1/Services/"+svc+" was not found")
		return
	}
	if len(parts) == 1 {
		switch r.Method {
		case "GET":
			fwrite(w, 200, o)
		case "POST":
			if msg := f.applyMessaging(o, form); msg != "" {
				ferr(w, 400, msg)
				return
			}
			fwrite(w, 200, o)
		case "DELETE":
			f.del("msg", testAccount, svc)
			w.WriteHeader(204)
		default:
			ferr(w, 405, "method not allowed")
		}
		return
	}
	if parts[1] != "PhoneNumbers" {
		ferr(w, 404, "not found")
		return
	}
	if len(parts) == 2 {
		if r.Method != "POST" {
			ferr(w, 405, "method not allowed")
			return
		}
		pn := form.Get("PhoneNumberSid")
		num := f.findNumberBySid(pn)
		if num == nil {
			ferr(w, 404, "Phone number "+pn+" was not found")
			return
		}
		if _, dup := f.objs[key("sender", svc, pn)]; dup {
			ferr(w, 409, "Phone Number is already in the Messaging Service")
			return
		}
		s := map[string]any{"sid": pn, "service_sid": svc, "phone_number": num["phone_number"],
			"capabilities": []any{"SMS", "MMS"}, "country_code": "US"}
		f.put("sender", svc, pn, s)
		fwrite(w, 201, s)
		return
	}
	pn := parts[2]
	s, ok := f.objs[key("sender", svc, pn)]
	if !ok {
		ferr(w, 404, "The requested resource was not found")
		return
	}
	switch r.Method {
	case "GET":
		fwrite(w, 200, s)
	case "DELETE":
		f.del("sender", svc, pn)
		w.WriteHeader(204)
	default:
		ferr(w, 405, "method not allowed")
	}
}

func (f *fakeTwilio) applyMessaging(o map[string]any, form url.Values) string {
	if msg := applyForm(o, form, msgFields); msg != "" {
		return msg
	}
	if v, _ := o["validity_period"].(float64); v < 1 || v > 36000 {
		return "ValidityPeriod must be between 1 and 36000"
	}
	if s, _ := o["scan_message_content"].(string); s != "inherit" && s != "enable" && s != "disable" {
		return "Invalid ScanMessageContent"
	}
	return ""
}

func (f *fakeTwilio) findNumberBySid(sid string) map[string]any {
	for k, o := range f.objs {
		if strings.HasPrefix(k, "number/") && o["sid"] == sid {
			return o
		}
	}
	return nil
}

func (f *fakeTwilio) handleVerify(w http.ResponseWriter, r *http.Request, form url.Values, svc string) {
	if svc == "" {
		if r.Method != "POST" {
			ferr(w, 405, "method not allowed")
			return
		}
		o := map[string]any{"sid": f.sid("VA"), "account_sid": testAccount, "friendly_name": nil, "code_length": float64(6),
			"lookup_enabled": false, "skip_sms_to_landlines": false, "dtmf_input_required": false, "tts_name": nil,
			"psd2_enabled": false, "do_not_share_warning_enabled": false, "custom_code_enabled": true,
			"date_created": "Mon, 02 Nov 2026 00:00:00 +0000"}
		if msg := f.applyVerify(o, form); msg != "" {
			ferr(w, 400, msg)
			return
		}
		f.put("verify", testAccount, o["sid"].(string), o)
		fwrite(w, 201, o)
		return
	}
	o, ok := f.objs[key("verify", testAccount, svc)]
	if !ok {
		ferr(w, 404, "The requested resource /v2/Services/"+svc+" was not found")
		return
	}
	switch r.Method {
	case "GET":
		fwrite(w, 200, o)
	case "POST":
		if msg := f.applyVerify(o, form); msg != "" {
			ferr(w, 400, msg)
			return
		}
		fwrite(w, 200, o)
	case "DELETE":
		f.del("verify", testAccount, svc)
		w.WriteHeader(204)
	default:
		ferr(w, 405, "method not allowed")
	}
}

func (f *fakeTwilio) applyVerify(o map[string]any, form url.Values) string {
	if msg := applyForm(o, form, verifyFields); msg != "" {
		return msg
	}
	if n, _ := o["friendly_name"].(string); n == "" || len(n) > 32 {
		return "FriendlyName must be 1-32 characters"
	}
	if c, _ := o["code_length"].(float64); c < 4 || c > 10 {
		return "CodeLength must be between 4 and 10"
	}
	return ""
}

// ---- test-side accessors ----

func (f *fakeTwilio) get(kind, acct, sid string) map[string]any {
	f.mu.Lock()
	defer f.mu.Unlock()
	return f.objs[key(kind, acct, sid)]
}

func (f *fakeTwilio) mutate(kind, acct, sid string, fn func(map[string]any)) {
	f.mu.Lock()
	defer f.mu.Unlock()
	o, ok := f.objs[key(kind, acct, sid)]
	if !ok {
		f.t.Fatalf("fake: no %s/%s/%s", kind, acct, sid)
	}
	fn(o)
}

func (f *fakeTwilio) remove(kind, acct, sid string) {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.del(kind, acct, sid)
}

// addNumber seeds an owned phone number with Twilio's default demo webhooks.
func (f *fakeTwilio) addNumber(acct, sid, e164 string) {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.put("number", acct, sid, map[string]any{"sid": sid, "account_sid": acct, "phone_number": e164,
		"friendly_name": "(415) 555-0100", "voice_url": "https://demo.twilio.com/welcome/voice/", "voice_method": "POST",
		"voice_fallback_url": nil, "voice_fallback_method": "POST", "status_callback": nil, "status_callback_method": "POST",
		"sms_url": nil, "sms_method": "POST", "sms_fallback_url": nil, "sms_fallback_method": "POST",
		"voice_application_sid": nil, "sms_application_sid": nil, "voice_caller_id_lookup": false,
		"capabilities": map[string]any{"voice": true, "sms": true, "mms": false}})
}

// addSubaccount seeds a subaccount of the main account.
func (f *fakeTwilio) addSubaccount(sid string) {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.put("account", testAccount, sid, map[string]any{"sid": sid, "friendly_name": "sub", "status": "active",
		"auth_token": "tok-" + sid, "owner_account_sid": testAccount, "type": "Full",
		"date_created": "Mon, 02 Nov 2026 00:00:00 +0000"})
}

// addKey seeds a key (as created through the IAM route) owned by acct.
func (f *fakeTwilio) addKey(acct, sid, name string, policy map[string]any) {
	f.mu.Lock()
	defer f.mu.Unlock()
	var pol any
	if policy != nil {
		pol = policy
	}
	f.put("key", acct, sid, map[string]any{"sid": sid, "account_sid": acct, "friendly_name": name, "policy": pol,
		"secret": "s", "date_created": "Mon, 02 Nov 2026 00:00:00 +0000"})
}

// failNext makes the next request whose method and path match answer with status.
func (f *fakeTwilio) failNext(method, pathPart string, status int) {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.failure[method+" "+pathPart] = status
}

func (f *fakeTwilio) callsMatching(method, pathPart string) []fakeCall {
	f.mu.Lock()
	defer f.mu.Unlock()
	var out []fakeCall
	for _, c := range f.calls {
		if c.Method == method && strings.Contains(c.Path, pathPart) {
			out = append(out, c)
		}
	}
	return out
}

func (f *fakeTwilio) resetCalls() {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.calls = nil
}

func (f *fakeTwilio) callCount() int {
	f.mu.Lock()
	defer f.mu.Unlock()
	return len(f.calls)
}
