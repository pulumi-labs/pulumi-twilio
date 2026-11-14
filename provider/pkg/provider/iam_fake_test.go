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
	"net/http"
	"strings"
)

// The IAM routes of OAuth apps and role permissions, and the account-wide messaging routes, served for
// the OAuthApp and lookup tests:
//
//	POST /iam/v1/Account/OAuthApps, DELETE /iam/v1/Account/OAuthApps/{sid} (no fetch, no update)
//	GET /iam/v1/Roles/{sid}/Permissions (two pages), GET /messaging/v1/Services/Usecases,
//	GET /messaging/v1/Deactivations?Date= (307 with Location)
//
// The shapes follow twilio-oai spec/json/twilio_iam_v1.json and twilio_messaging_v1.json (1.0.0).
// OAuth apps are stored as kind "oapp"; the client secret is returned on create only.

func (f *fakeTwilio) handleIAMOther(w http.ResponseWriter, r *http.Request, body []byte, rest string) {
	parts := strings.Split(rest, "/")
	switch {
	case rest == "Account/OAuthApps" && r.Method == "POST":
		if !strings.HasPrefix(r.Header.Get("Content-Type"), "application/json") {
			ferr(w, 400, "the body must be JSON")
			return
		}
		var in map[string]any
		if err := json.Unmarshal(body, &in); err != nil {
			ferr(w, 400, "invalid JSON")
			return
		}
		if in["friendly_name"] == nil || in["friendly_name"] == "" {
			ferr(w, 400, "friendly_name is required")
			return
		}
		if m, ok := in["token_endpoint_auth_method"]; ok && m != "client_secret_basic" {
			ferr(w, 400, "unsupported token_endpoint_auth_method")
			return
		}
		sid := f.sid("OQ")
		o := map[string]any{"sid": sid, "friendly_name": in["friendly_name"], "description": in["description"], "type": in["type"],
			"date_created": cvStamp, "created_by": "api", "creator_sid": "US" + strings.Repeat("1", 32), "secret": "oauth-secret-" + sid,
			"status": "active", "policy": in["policy"], "access_token_ttl": in["access_token_ttl"]}
		f.put("oapp", testAccount, sid, o)
		f.oauthBodies = append(f.oauthBodies, in)
		fwrite(w, 201, o)
	case len(parts) == 3 && parts[0] == "Account" && parts[1] == "OAuthApps" && r.Method == "DELETE":
		if _, ok := f.objs[key("oapp", testAccount, parts[2])]; !ok {
			ferr(w, 404, "The requested resource "+r.URL.Path+" was not found")
			return
		}
		f.del("oapp", testAccount, parts[2])
		w.WriteHeader(204)
	case len(parts) == 3 && parts[0] == "Roles" && parts[2] == "Permissions" && r.Method == "GET":
		if parts[1] != "IX"+strings.Repeat("a", 32) {
			ferr(w, 404, "The requested resource "+r.URL.Path+" was not found")
			return
		}
		perm := func(sid, action string) map[string]any {
			return map[string]any{"sid": sid, "namespace": "twilio", "product": "iam", "resource": "roles", "action": action, "externalDescription": "Can " + action + " roles"}
		}
		if r.URL.Query().Get("PageToken") == "2" {
			fwrite(w, 200, map[string]any{"permissions": []any{perm("IZ"+strings.Repeat("2", 32), "write")}, "meta": map[string]any{"key": "permissions", "next_page_url": nil}})
			return
		}
		fwrite(w, 200, map[string]any{"permissions": []any{perm("IZ"+strings.Repeat("1", 32), "read")},
			"meta": map[string]any{"key": "permissions", "next_page_url": "https://iam.twilio.com" + r.URL.Path + "?PageSize=100&PageToken=2"}})
	default:
		ferr(w, 404, "The requested resource "+r.URL.Path+" was not found")
	}
}

func (f *fakeTwilio) handleMessagingAccount(w http.ResponseWriter, r *http.Request, path string) {
	if r.Method != "GET" {
		ferr(w, 405, "method not allowed")
		return
	}
	if path == "/messaging/v1/Services/Usecases" {
		fwrite(w, 200, map[string]any{"usecases": []any{
			map[string]any{"usecase": "notifications", "description": "Notifications", "purpose": "Order and account notifications"},
			map[string]any{"usecase": "marketing", "description": "Marketing", "purpose": "Promotions"},
		}})
		return
	}
	d := r.URL.Query().Get("Date")
	if d == "" {
		ferr(w, 400, "Date is required")
		return
	}
	target := "https://api.example.invalid/deactivated/" + d + ".csv?token=abc"
	w.Header().Set("Location", target)
	fwrite(w, 307, map[string]any{"redirect_to": target})
}
