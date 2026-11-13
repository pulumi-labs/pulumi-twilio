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
	"strings"
)

// The Accounts v1 routes, served for the Accounts tests:
//
//	/accounts/v1/Credentials/AWS[/{sid}], /accounts/v1/Credentials/PublicKeys[/{sid}],
//	/accounts/v1/SafeList/Numbers (number in the query), /accounts/v1/Messaging/GeoPermissions
//
// The shapes follow twilio-oai spec/json/twilio_accounts_v1.json. Credentials are stored as kind
// "acr" keyed <AWS|PublicKeys>/<sid>; the secret is checked on create and never returned.

func (f *fakeTwilio) handleAccounts(w http.ResponseWriter, r *http.Request, form url.Values, rest string) {
	notFound := func() { ferr(w, 404, "The requested resource "+r.URL.Path+" was not found") }
	parts := strings.Split(rest, "/")
	switch {
	case parts[0] == "Credentials" && len(parts) >= 2 && len(parts) <= 3 && (parts[1] == "AWS" || parts[1] == "PublicKeys"):
		sid := ""
		if len(parts) == 3 {
			sid = parts[2]
		}
		f.accountsCredential(w, r, form, parts[1], sid, notFound)
	case rest == "SafeList/Numbers":
		f.accountsSafeList(w, r, form, notFound)
	case rest == "Messaging/GeoPermissions" && r.Method == "GET":
		all := []any{
			map[string]any{"country_code": "IN", "type": "country", "enabled": false, "prefix": "+91", "message": ""},
			map[string]any{"country_code": "PK", "type": "country", "enabled": true, "prefix": "+92", "message": "High Risk Country"},
			map[string]any{"country_code": "US", "type": "country", "enabled": true, "prefix": "+1", "message": ""},
		}
		want := r.URL.Query().Get("CountryCode")
		out := []any{}
		for _, p := range all {
			cc := p.(map[string]any)["country_code"].(string)
			if want == "" || strings.Contains(","+want+",", ","+cc+",") {
				out = append(out, p)
			}
		}
		fwrite(w, 200, map[string]any{"permissions": out})
	default:
		notFound()
	}
}

func (f *fakeTwilio) accountsCredential(w http.ResponseWriter, r *http.Request, form url.Values, kind, sid string, notFound func()) {
	secretParam := map[string]string{"AWS": "Credentials", "PublicKeys": "PublicKey"}[kind]
	fields := []fakeField{{"FriendlyName", "friendly_name", "s"}}
	if sid == "" {
		if r.Method != "POST" {
			ferr(w, 405, "method not allowed")
			return
		}
		if form.Get(secretParam) == "" {
			ferr(w, 400, "Missing required parameter "+secretParam)
			return
		}
		id := f.sid("CR")
		o := map[string]any{"sid": id, "account_sid": testAccount, "friendly_name": nil, "date_created": cvStamp, "date_updated": cvStamp,
			"url": "https://accounts.twilio.com/v1/Credentials/" + kind + "/" + id}
		if a := form.Get("AccountSid"); a != "" {
			o["account_sid"] = a
		}
		if msg := applyForm(o, form, fields); msg != "" {
			ferr(w, 400, msg)
			return
		}
		f.put("acr", testAccount, kind+"/"+id, o)
		fwrite(w, 201, o)
		return
	}
	k := kind + "/" + sid
	o, ok := f.objs[key("acr", testAccount, k)]
	if !ok {
		notFound()
		return
	}
	switch r.Method {
	case "GET":
		fwrite(w, 200, o)
	case "POST":
		if form.Get(secretParam) != "" {
			ferr(w, 400, "the "+secretParam+" cannot be updated")
			return
		}
		if msg := applyForm(o, form, fields); msg != "" {
			ferr(w, 400, msg)
			return
		}
		o["date_updated"] = cvUpdatedStamp
		fwrite(w, 200, o)
	case "DELETE":
		f.del("acr", testAccount, k)
		w.WriteHeader(204)
	default:
		ferr(w, 405, "method not allowed")
	}
}

func (f *fakeTwilio) accountsSafeList(w http.ResponseWriter, r *http.Request, form url.Values, notFound func()) {
	if r.Method == "POST" {
		n := form.Get("PhoneNumber")
		if n == "" {
			ferr(w, 400, "Missing required parameter PhoneNumber")
			return
		}
		o := map[string]any{"sid": f.sid("GN"), "phone_number": n}
		f.put("asl", testAccount, n, o)
		fwrite(w, 201, o)
		return
	}
	n := r.URL.Query().Get("PhoneNumber")
	if !strings.Contains(r.URL.RawQuery, "%2B") {
		ferr(w, 400, "the plus sign of the phone number must be escaped as %2B: "+r.URL.RawQuery)
		return
	}
	o, ok := f.objs[key("asl", testAccount, n)]
	if !ok {
		notFound()
		return
	}
	switch r.Method {
	case "GET":
		fwrite(w, 200, o)
	case "DELETE":
		f.del("asl", testAccount, n)
		w.WriteHeader(204)
	default:
		ferr(w, 405, "method not allowed")
	}
}
