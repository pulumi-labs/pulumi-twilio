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

// The Routes API (routes.twilio.com), served for the inbound processing region tests:
//
//	/routes/v2/SipDomains/{SipDomain}, /routes/v2/Trunks/{SipTrunkDomain}   (PascalCase form, snake_case JSON)
//	/routes/v3/PhoneNumbers/{phoneNumber}, /routes/v3/ShortCodes/{iso}/{code} (camelCase form and JSON)
//
// Only objects seeded with seedRoute exist: POST updates the settings it names and answers the
// object, GET reads it, both answer 404 for an unknown object, and there is no DELETE. Shapes
// follow twilio-oai spec/json/twilio_routes_v2.json and twilio_routes_v3.json (version 1.0.0).

// seedRoute makes the object at path (for example v2/SipDomains/a.sip.twilio.com) exist with
// Twilio's default region.
func (f *fakeTwilio) seedRoute(path string) {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.put("routes", testAccount, path, map[string]any{"voice": "us1", "messaging": "us1", "friendly": "", "created": "2026-11-11T00:00:00Z", "updated": "2026-11-11T00:00:00Z"})
}

func (f *fakeTwilio) routeObject(path string) map[string]any {
	return f.objs[key("routes", testAccount, path)]
}

func (f *fakeTwilio) handleRoutes(w http.ResponseWriter, r *http.Request, form url.Values) {
	path := strings.TrimPrefix(r.URL.Path, "/routes/")
	o := f.routeObject(path)
	if o == nil {
		ferr(w, 404, "The requested resource "+r.URL.Path+" was not found")
		return
	}
	v2 := strings.HasPrefix(path, "v2/")
	name := func(camel string) string {
		if v2 {
			return strings.ToUpper(camel[:1]) + camel[1:]
		}
		return camel
	}
	switch r.Method {
	case http.MethodPost:
		for form_, field := range map[string]string{"voiceRegion": "voice", "messagingRegion": "messaging", "friendlyName": "friendly"} {
			if form.Has(name(form_)) {
				o[field] = form.Get(name(form_))
			}
		}
		o["updated"] = "2026-11-11T01:00:00Z"
	case http.MethodGet:
	default:
		ferr(w, 405, "no such method")
		return
	}
	segs := strings.Split(strings.TrimPrefix(strings.TrimPrefix(path, "v2/"), "v3/"), "/")
	id := strings.Join(segs[1:], "/")
	var out map[string]any
	if v2 {
		out = map[string]any{"sid": "QQ" + strings.Repeat("0", 32), "account_sid": testAccount, "url": "https://routes.twilio.com/" + path,
			"voice_region": o["voice"], "friendly_name": o["friendly"], "date_created": o["created"], "date_updated": o["updated"]}
		if segs[0] == "SipDomains" {
			out["sip_domain"] = id
		} else {
			out["sip_trunk_domain"] = id
		}
	} else {
		out = map[string]any{"accountSid": testAccount, "url": "https://routes.twilio.com/" + path,
			"messagingRegion": o["messaging"], "friendlyName": o["friendly"], "dateCreated": o["created"], "dateUpdated": o["updated"]}
		if segs[0] == "PhoneNumbers" {
			out["phoneNumber"] = id
			out["voiceRegion"] = o["voice"]
		}
	}
	fwrite(w, 200, out)
}
