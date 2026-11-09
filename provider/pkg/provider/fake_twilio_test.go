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
	"regexp"
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

	mu               sync.Mutex
	endlessKeyPages  bool                      // the account-explicit key list never ends (a truncation probe)
	templatePageSize int                       // when set, /v2/Templates pages hold this many items
	objs             map[string]map[string]any // "kind/account/sid" -> object; ordered by seq
	order            []string
	next             int
	calls            []fakeCall
	failure          map[string]int // "METHOD substring" -> status to return once
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
		{"ApiVersion", "api_version", "s"}, {"PublicApplicationConnectEnabled", "public_application_connect_enabled", "b"},
	}
	addressFields = []fakeField{
		{"CustomerName", "customer_name", "s"}, {"Street", "street", "s"}, {"StreetSecondary", "street_secondary", "u"},
		{"City", "city", "s"}, {"Region", "region", "s"}, {"PostalCode", "postal_code", "s"},
		{"IsoCountry", "iso_country", "s"}, {"FriendlyName", "friendly_name", "u"}, {"EmergencyEnabled", "emergency_enabled", "b"},
	}
	addressUpdateFields = []fakeField{addressFields[0], addressFields[1], addressFields[2], addressFields[3], addressFields[4], addressFields[5], addressFields[7], addressFields[8]} // no IsoCountry on update
	queueFields         = []fakeField{{"FriendlyName", "friendly_name", "s"}, {"MaxSize", "max_size", "i"}}
	triggerFields       = []fakeField{
		{"CallbackUrl", "callback_url", "u"}, {"CallbackMethod", "callback_method", "m"}, {"FriendlyName", "friendly_name", "s"},
		{"Recurring", "recurring", "u"}, {"TriggerBy", "trigger_by", "u"}, {"UsageCategory", "usage_category", "s"},
	}
	triggerUpdateFields = triggerFields[:3:3] // callback and name only
	numberFields        = []fakeField{
		{"FriendlyName", "friendly_name", "s"},
		{"VoiceUrl", "voice_url", "u"}, {"VoiceMethod", "voice_method", "m"},
		{"VoiceFallbackUrl", "voice_fallback_url", "u"}, {"VoiceFallbackMethod", "voice_fallback_method", "m"},
		{"StatusCallback", "status_callback", "u"}, {"StatusCallbackMethod", "status_callback_method", "m"},
		{"SmsUrl", "sms_url", "u"}, {"SmsMethod", "sms_method", "m"},
		{"SmsFallbackUrl", "sms_fallback_url", "u"}, {"SmsFallbackMethod", "sms_fallback_method", "m"},
		{"VoiceApplicationSid", "voice_application_sid", "u"}, {"SmsApplicationSid", "sms_application_sid", "u"},
		{"VoiceCallerIdLookup", "voice_caller_id_lookup", "b"},
		{"TrunkSid", "trunk_sid", "u"}, {"VoiceReceiveMode", "voice_receive_mode", "s"},
		{"AddressSid", "address_sid", "u"}, {"IdentitySid", "identity_sid", "u"}, {"BundleSid", "bundle_sid", "u"},
		{"EmergencyStatus", "emergency_status", "s"}, {"EmergencyAddressSid", "emergency_address_sid", "u"},
		{"ApiVersion", "api_version", "s"},
	}
	shortCodeFields = []fakeField{
		{"FriendlyName", "friendly_name", "s"}, {"ApiVersion", "api_version", "s"},
		{"SmsUrl", "sms_url", "u"}, {"SmsMethod", "sms_method", "m"},
		{"SmsFallbackUrl", "sms_fallback_url", "u"}, {"SmsFallbackMethod", "sms_fallback_method", "m"},
	}
	connectAppFields = []fakeField{
		{"FriendlyName", "friendly_name", "s"}, {"AuthorizeRedirectUrl", "authorize_redirect_url", "u"},
		{"CompanyName", "company_name", "s"}, {"DeauthorizeCallbackUrl", "deauthorize_callback_url", "u"},
		{"DeauthorizeCallbackMethod", "deauthorize_callback_method", "m"}, {"Description", "description", "s"},
		{"HomepageUrl", "homepage_url", "u"}, {"Permissions", "permissions", "l"},
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
		{"Usecase", "usecase", "s"},
	}
	verifyFields = []fakeField{
		{"FriendlyName", "friendly_name", "s"}, {"CodeLength", "code_length", "i"},
		{"LookupEnabled", "lookup_enabled", "b"}, {"SkipSmsToLandlines", "skip_sms_to_landlines", "b"},
		{"DtmfInputRequired", "dtmf_input_required", "b"}, {"TtsName", "tts_name", "s"},
		{"Psd2Enabled", "psd2_enabled", "b"}, {"DoNotShareWarningEnabled", "do_not_share_warning_enabled", "b"},
		{"CustomCodeEnabled", "custom_code_enabled", "b"},
		{"DefaultTemplateSid", "default_template_sid", "u"}, {"VerifyEventSubscriptionEnabled", "verify_event_subscription_enabled", "b"},
		{"Push.ApnCredentialSid", "push.apn_credential_sid", "u"}, {"Push.FcmCredentialSid", "push.fcm_credential_sid", "u"},
		{"Push.IncludeDate", "push.include_date", "b"}, {"Totp.Issuer", "totp.issuer", "s"},
		{"Totp.CodeLength", "totp.code_length", "i"}, {"Totp.TimeStep", "totp.time_step", "i"}, {"Totp.Skew", "totp.skew", "i"},
		{"Whatsapp.MsgServiceSid", "whatsapp.msg_service_sid", "u"}, {"Whatsapp.From", "whatsapp.from", "s"},
		{"Passkeys.RelyingParty.Id", "passkeys.relying_party.id", "s"}, {"Passkeys.RelyingParty.Name", "passkeys.relying_party.name", "s"},
		{"Passkeys.RelyingParty.Origins", "passkeys.relying_party.origins", "l"},
		{"Passkeys.AuthenticatorAttachment", "passkeys.authenticator_attachment", "s"},
		{"Passkeys.DiscoverableCredentials", "passkeys.discoverable_credentials", "s"},
		{"Passkeys.UserVerification", "passkeys.user_verification", "s"},
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
// setPath stores v at key; a dotted key writes into nested objects, as the
// real API returns push.*, totp.*, whatsapp.* and passkeys.* settings.
func setPath(o map[string]any, key string, v any) {
	parts := strings.Split(key, ".")
	for _, p := range parts[:len(parts)-1] {
		next, ok := o[p].(map[string]any)
		if !ok {
			next = map[string]any{}
			o[p] = next
		}
		o = next
	}
	o[parts[len(parts)-1]] = v
}

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
				setPath(o, fl.key, nil)
			} else {
				setPath(o, fl.key, v)
			}
		case "m":
			if v != "GET" && v != "POST" {
				return fmt.Sprintf("Invalid value for %s: %q (must be GET or POST)", fl.param, v)
			}
			setPath(o, fl.key, v)
		case "b":
			b, err := strconv.ParseBool(v)
			if err != nil {
				return "Invalid boolean for " + fl.param
			}
			setPath(o, fl.key, b)
		case "l":
			list := []any{}
			for _, e := range strings.Split(v, ",") {
				if e != "" {
					list = append(list, e)
				}
			}
			setPath(o, fl.key, list)
		case "i":
			n, err := strconv.Atoi(v)
			if err != nil {
				return "Invalid integer for " + fl.param
			}
			setPath(o, fl.key, float64(n))
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
		rest := strings.Trim(strings.TrimPrefix(path, "/verify/v2/Services"), "/")
		if strings.Contains(rest, "/") {
			f.handleVerifyChild(w, r, form, strings.Split(rest, "/"))
			return
		}
		f.handleVerify(w, r, form, rest)
	case strings.HasPrefix(path, "/verify/v2/SafeList/Numbers"):
		f.handleVerifySafeList(w, r, form, strings.Trim(strings.TrimPrefix(path, "/verify/v2/SafeList/Numbers"), "/"))
	case path == "/verify/v2/Templates":
		f.handleVerifyTemplates(w, r)
	case strings.HasPrefix(path, "/serverless/v1/Services"):
		rest := strings.Trim(strings.TrimPrefix(path, "/serverless/v1/Services"), "/")
		if rest == "" {
			f.handleServerless(w, r, form, nil)
			return
		}
		f.handleServerless(w, r, form, strings.Split(rest, "/"))
	case strings.HasPrefix(path, "/events/v1/"):
		f.handleEvents(w, r, form, strings.Split(strings.Trim(strings.TrimPrefix(path, "/events/v1/"), "/"), "/"))
	case strings.HasPrefix(path, "/voice/v1/"):
		f.handleVoice(w, r, form, strings.Split(strings.Trim(strings.TrimPrefix(path, "/voice/v1/"), "/"), "/"))
	case strings.HasPrefix(path, "/trunking/v1/"):
		f.handleTrunking(w, r, form, strings.Split(strings.Trim(strings.TrimPrefix(path, "/trunking/v1/"), "/"), "/"))
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
	if len(parts) >= 4 && parts[2] == "Usage" && trimJSON(parts[3]) == "Triggers" { // fold Usage/Triggers into one segment
		parts = append([]string{parts[0], parts[1], "Usage/Triggers"}, parts[4:]...)
	}
	if len(parts) >= 4 && parts[2] == "SMS" && trimJSON(parts[3]) == "ShortCodes" { // fold SMS/ShortCodes into one segment
		parts = append([]string{parts[0], parts[1], "SMS/ShortCodes"}, parts[4:]...)
	}
	if len(parts) >= 4 && parts[2] == "SIP" {
		f.handleSIP(w, r, form, parts)
		return
	}
	if f.handleAssignedAddOn(w, r, form, parts) {
		return
	}
	if len(parts) >= 3 && f.handleDataAPI(w, r, parts) {
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
		update []fakeField // fields an update accepts; nil means all of fields
	}{
		"Addresses":            {"address", "AD", addressFields, addressUpdateFields},
		"SigningKeys":          {"signingkey", "SK", []fakeField{{"FriendlyName", "friendly_name", "s"}}, nil},
		"Queues":               {"queue", "QU", queueFields, nil},
		"Usage/Triggers":       {"trigger", "UT", triggerFields, triggerUpdateFields},
		"Applications":         {"app", "AP", appFields, nil},
		"IncomingPhoneNumbers": {"number", "PN", numberFields, nil},
		"Keys":                 {"key", "SK", []fakeField{{"FriendlyName", "friendly_name", "s"}}, nil},
		"SMS/ShortCodes":       {"shortcode", "SC", shortCodeFields, nil},
		"ConnectApps":          {"connectapp", "CN", connectAppFields, nil},
		"OutgoingCallerIds":    {"callerid", "PN", []fakeField{{"FriendlyName", "friendly_name", "s"}}, nil},
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
		case r.Method == "POST" && (coll == "Addresses" || coll == "SigningKeys" || coll == "Queues" || coll == "Usage/Triggers"):
			f.createAccountResource(w, coll, acct, form, k.kind, k.prefix, k.fields)
		case r.Method == "GET" && coll == "Keys": // the account-explicit key list; no secrets
			list := []any{}
			for _, ok := range f.order {
				o, live := f.objs[ok]
				if live && strings.HasPrefix(ok, "key/"+acct+"/") {
					c := clone(o)
					delete(c, "secret")
					list = append(list, c)
				}
			}
			resp := map[string]any{"keys": list}
			if f.endlessKeyPages { // an account whose key list never ends
				resp["next_page_uri"] = strings.TrimPrefix(r.URL.Path, "/api") + "?Page=next" // as Twilio: no host prefix
			}
			fwrite(w, 200, resp)
		case r.Method == "GET" && (coll == "SMS/ShortCodes" || coll == "ConnectApps" || coll == "OutgoingCallerIds"):
			listKey := map[string]string{"SMS/ShortCodes": "short_codes", "ConnectApps": "connect_apps", "OutgoingCallerIds": "outgoing_caller_ids"}[coll]
			filters := map[string]string{"ShortCode": "short_code", "PhoneNumber": "phone_number"}
			list := []any{}
			for _, ok := range f.order {
				o, live := f.objs[ok]
				if !live || !strings.HasPrefix(ok, k.kind+"/"+acct+"/") {
					continue
				}
				match := true
				for param, field := range filters {
					if want := r.URL.Query().Get(param); want != "" && o[field] != want {
						match = false
					}
				}
				if match {
					list = append(list, o)
				}
			}
			fwrite(w, 200, map[string]any{listKey: list})
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
		if o["in_use"] == true {
			ferr(w, 400, "Address is in use and cannot be deleted")
			return
		}
		if k.kind == "number" || k.kind == "shortcode" {
			ferr(w, 405, "numbers are released through a separate flow")
			return
		}
		f.del(k.kind, acct, sid)
		w.WriteHeader(204)
	case "POST":
		upd := k.fields
		if k.update != nil {
			upd = k.update
		}
		if msg := applyForm(o, form, upd); msg != "" {
			ferr(w, 400, msg)
			return
		}
		if n, ok := o["max_size"].(float64); ok && k.kind == "queue" && (n < 1 || n > 5000) {
			ferr(w, 400, "MaxSize must be between 1 and 5000")
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
		// Like the live API (#1574): an assertion outside /twilio/<product>/<resource>/<action>
		// fails the call with error 70002 AFTER the key was created, and the error has no SID.
		if bad := badAssertion(pol); bad != "" {
			fwrite(w, 400, map[string]any{"code": 70002, "message": "assertion '" + bad + "' is invalid", "status": 400})
			return
		}
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
			"synchronous_validation": false, "validity_period": float64(36000), "use_inbound_webhook_on_number": true, "usecase": "undeclared",
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
		// Twilio sets the TOTP issuer to the friendly name at creation.
		if _, ok := form["Totp.Issuer"]; !ok {
			setPath(o, "totp.issuer", o["friendly_name"])
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
	if v, ok := form["Totp.Issuer"]; ok && v[0] == "" {
		return "Invalid parameter: Totp.Issuer must not be empty"
	}
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

// addShortCode seeds a leased short code with no webhooks.
func (f *fakeTwilio) addShortCode(acct, sid, code string) {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.put("shortcode", acct, sid, map[string]any{"sid": sid, "account_sid": acct, "short_code": code,
		"friendly_name": "Short code " + code, "api_version": "2010-04-01", "sms_url": nil, "sms_method": "POST",
		"sms_fallback_url": nil, "sms_fallback_method": "POST"})
}

// addConnectApp seeds a Connect app as the Console creates it.
func (f *fakeTwilio) addConnectApp(acct, sid string) {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.put("connectapp", acct, sid, map[string]any{"sid": sid, "account_sid": acct, "friendly_name": "console app",
		"authorize_redirect_url": "https://example.com/authorize", "company_name": "Acme", "description": "An app",
		"homepage_url": "https://example.com", "deauthorize_callback_url": nil, "deauthorize_callback_method": "POST",
		"permissions": []any{"get-all"}})
}

// addCallerID seeds a verified outgoing caller ID.
func (f *fakeTwilio) addCallerID(acct, sid, e164 string) {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.put("callerid", acct, sid, map[string]any{"sid": sid, "account_sid": acct, "phone_number": e164,
		"friendly_name": e164, "date_created": "Mon, 02 Nov 2026 00:00:00 +0000"})
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

// createAccountResource creates an address, signing key, queue or usage trigger with the
// defaults and quirks of the real API: addresses are auto-corrected unless AutoCorrectAddress
// is false, the signing key secret is returned only on create, queues default to 1000 slots and
// trigger values come back as normalized decimals (an offset becomes the absolute value).
func (f *fakeTwilio) createAccountResource(w http.ResponseWriter, coll, acct string, form url.Values, kind, prefix string, fields []fakeField) {
	now := "Mon, 02 Nov 2026 00:00:00 +0000"
	o := map[string]any{"sid": f.sid(prefix), "account_sid": acct, "date_created": now, "date_updated": now}
	var required []string
	switch coll {
	case "Addresses":
		required = []string{"CustomerName", "Street", "City", "Region", "PostalCode", "IsoCountry"}
		o["validated"], o["verified"], o["emergency_enabled"] = true, false, false
		o["friendly_name"], o["street_secondary"] = nil, nil
	case "SigningKeys":
		o["friendly_name"] = nil
	case "Queues":
		required = []string{"FriendlyName"}
		o["max_size"], o["current_size"], o["average_wait_time"] = float64(1000), float64(0), float64(0)
	case "Usage/Triggers":
		required = []string{"CallbackUrl", "TriggerValue", "UsageCategory"}
		o["callback_method"], o["recurring"], o["trigger_by"] = "POST", nil, "usage"
		o["current_value"], o["date_fired"], o["api_version"] = "0", nil, "2010-04-01"
		o["usage_record_uri"] = "/2010-04-01/Accounts/" + acct + "/Usage/Records.json"
	}
	for _, r := range required {
		if form.Get(r) == "" {
			ferr(w, 400, "Missing required parameter "+r)
			return
		}
	}
	if msg := applyForm(o, form, fields); msg != "" {
		ferr(w, 400, msg)
		return
	}
	switch coll {
	case "Addresses":
		if form.Get("AutoCorrectAddress") != "false" {
			o["street"] = strings.ToUpper(o["street"].(string)) // Twilio normalizes the stored address
		}
	case "Queues":
		if n := o["max_size"].(float64); n < 1 || n > 5000 {
			ferr(w, 400, "MaxSize must be between 1 and 5000")
			return
		}
	case "Usage/Triggers":
		v := strings.TrimPrefix(form.Get("TriggerValue"), "+")
		n, err := strconv.ParseFloat(v, 64)
		if err != nil {
			ferr(w, 400, "Invalid TriggerValue")
			return
		}
		o["trigger_value"] = strconv.FormatFloat(n, 'f', 1, 64)
	}
	f.put(kind, acct, o["sid"].(string), o)
	out := clone(o)
	if coll == "SigningKeys" {
		out["secret"] = "secret-" + o["sid"].(string)
	}
	fwrite(w, 201, out)
}

var keyAssertion = regexp.MustCompile(`^/twilio/[^/]+/[^/]+/[^/]+$`)

// badAssertion returns the first allow or deny entry of a key policy that Twilio would reject.
func badAssertion(pol any) string {
	m, _ := pol.(map[string]any)
	for _, k := range []string{"allow", "deny"} {
		items, _ := m[k].([]any)
		for _, it := range items {
			if a, ok := it.(string); ok && !keyAssertion.MatchString(a) {
				return a
			}
		}
	}
	return ""
}
