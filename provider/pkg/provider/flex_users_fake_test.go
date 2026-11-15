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

// The user routes, served for the Flex and Frontline user tests. Neither API has a create or a
// delete (twilio-oai 2.11.0), so the tests seed users with seedFlexUser and seedFrontlineUser.
//
//	/flex-api/v2/Instances/{instanceSid}/Users/{flexUserSid}      GET, POST (Email, UserSid, Locale)
//	/frontline-api/v1/Users/{sid}                                 GET, POST (FriendlyName, Avatar, State, IsAvailable)

var flexUserFields = []fakeField{{"Email", "email", "s"}, {"UserSid", "user_sid", "s"}, {"Locale", "locale", "s"}}

var frontlineUserFields = []fakeField{{"FriendlyName", "friendly_name", "s"}, {"Avatar", "avatar", "s"}, {"State", "state", "s"}, {"IsAvailable", "is_available", "b"}}

func (f *fakeTwilio) seedFlexUser(instance, sid string) {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.put("flexuser", testAccount, instance+"/"+sid, map[string]any{
		"account_sid": testAccount, "instance_sid": instance, "flex_user_sid": sid, "user_sid": "US" + strings.Repeat("a", 32),
		"worker_sid": "WK" + strings.Repeat("1", 32), "workspace_sid": "WS" + strings.Repeat("2", 32), "flex_team_sid": "FT" + strings.Repeat("3", 32),
		"username": "ada", "email": "ada@example.com", "locale": "en-US", "roles": []any{"agent"}, "created_date": cvStamp, "updated_date": cvStamp,
		"version": float64(1), "url": "https://flex-api.twilio.com/v2/Instances/" + instance + "/Users/" + sid,
	})
}

func (f *fakeTwilio) seedFrontlineUser(sid string) {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.put("frontlineuser", testAccount, sid, map[string]any{
		"sid": sid, "identity": "ada@example.com", "friendly_name": "Ada", "avatar": nil, "state": "active", "is_available": false,
		"url": "https://frontline-api.twilio.com/v1/Users/" + sid,
	})
}

// handleUsers serves both user APIs; the caller holds the lock.
func (f *fakeTwilio) handleUsers(w http.ResponseWriter, r *http.Request, form url.Values, _ []byte) {
	parts := strings.Split(strings.Trim(r.URL.Path, "/"), "/")
	var o map[string]any
	var fields []fakeField
	var ok bool
	if parts[0] == "frontline-api" {
		if len(parts) != 4 {
			ferr(w, 404, "The requested resource "+r.URL.Path+" was not found")
			return
		}
		o, ok = f.objs[key("frontlineuser", testAccount, parts[3])]
		fields = frontlineUserFields
	} else {
		if len(parts) != 6 || parts[4] != "Users" {
			ferr(w, 404, "The requested resource "+r.URL.Path+" was not found")
			return
		}
		o, ok = f.objs[key("flexuser", testAccount, parts[3]+"/"+parts[5])]
		fields = flexUserFields
	}
	if !ok {
		ferr(w, 404, "The requested resource "+r.URL.Path+" was not found")
		return
	}
	switch r.Method {
	case "GET":
		fwrite(w, 200, o)
	case "POST":
		if v := form.Get("State"); v != "" && v != "active" && v != "deactivated" {
			ferr(w, 400, "Invalid State "+v)
			return
		}
		if msg := applyForm(o, form, fields); msg != "" {
			ferr(w, 400, msg)
			return
		}
		if parts[0] == "flex-api" {
			o["updated_date"] = cvUpdatedStamp
			o["version"] = o["version"].(float64) + 1
		}
		fwrite(w, 200, o)
	default:
		ferr(w, 405, "method not allowed")
	}
}
