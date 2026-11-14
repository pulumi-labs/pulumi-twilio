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

// The link shortening routes of the fake: /messaging/v1/LinkShortening/...
//
//	Domains/{d}/Config (GET, POST), Domains/{d}/Certificate (GET, POST, DELETE), Domains/{d}/ValidateDns (GET),
//	Domains/{d}/MessagingServices/{svc} (POST, DELETE), MessagingServices/{svc}/Domain (GET),
//	MessagingService/{svc}/DomainConfig (GET)
//
// Domains are created out of band by addLinkDomain (Twilio sets them up); they are stored as kind "lsdom".
// Associations are stored as kind "lsassoc" keyed by the Messaging Service SID. The shapes follow
// twilio-oai spec/json/twilio_messaging_v1.json (1.0.0).

func (f *fakeTwilio) addLinkDomain(sid string) {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.put("lsdom", testAccount, sid, map[string]any{"domain_sid": sid, "config_sid": "ZK" + strings.TrimPrefix(sid, "ZN"),
		"fallback_url": nil, "callback_url": nil, "continue_on_failure": false, "disable_https": false,
		"date_created": cvStamp, "date_updated": cvStamp, "url": "https://messaging.twilio.com/v1/LinkShortening/Domains/" + sid + "/Config"})
}

func (f *fakeTwilio) handleLinkShortening(w http.ResponseWriter, r *http.Request, form url.Values, parts []string) {
	notFound := func() { ferr(w, 404, "The requested resource "+r.URL.Path+" was not found") }
	base := "https://messaging.twilio.com/v1/LinkShortening/"
	configOf := func(dom map[string]any, svc string) map[string]any {
		o := clone(dom)
		if svc != "" {
			o["messaging_service_sid"] = svc
		}
		return o
	}
	switch {
	case len(parts) == 3 && parts[0] == "MessagingServices" && parts[2] == "Domain" && r.Method == "GET":
		a, ok := f.objs[key("lsassoc", testAccount, parts[1])]
		if !ok {
			notFound()
			return
		}
		fwrite(w, 200, a)
	case len(parts) == 3 && parts[0] == "MessagingService" && parts[2] == "DomainConfig" && r.Method == "GET":
		a, ok := f.objs[key("lsassoc", testAccount, parts[1])]
		if !ok {
			notFound()
			return
		}
		fwrite(w, 200, configOf(f.objs[key("lsdom", testAccount, a["domain_sid"].(string))], parts[1]))
	case len(parts) >= 3 && parts[0] == "Domains":
		dom, ok := f.objs[key("lsdom", testAccount, parts[1])]
		if !ok {
			notFound()
			return
		}
		switch {
		case parts[2] == "Config" && len(parts) == 3 && r.Method == "GET":
			fwrite(w, 200, configOf(dom, ""))
		case parts[2] == "Config" && len(parts) == 3 && r.Method == "POST":
			if msg := applyForm(dom, form, []fakeField{{"FallbackUrl", "fallback_url", "s"}, {"CallbackUrl", "callback_url", "s"},
				{"ContinueOnFailure", "continue_on_failure", "b"}, {"DisableHttps", "disable_https", "b"}}); msg != "" {
				ferr(w, 400, msg)
				return
			}
			dom["date_updated"] = cvStamp
			fwrite(w, 200, configOf(dom, ""))
		case parts[2] == "Certificate" && len(parts) == 3:
			certKey := key("lscert", testAccount, parts[1])
			switch r.Method {
			case "POST":
				if !strings.Contains(form.Get("TlsCert"), "BEGIN CERTIFICATE") {
					ferr(w, 400, "TlsCert must be a PEM certificate")
					return
				}
				c, had := f.objs[certKey]
				if !had {
					c = map[string]any{"domain_sid": parts[1], "certificate_sid": "CR" + strings.TrimPrefix(parts[1], "ZN"), "domain_name": "go.acme.example.com",
						"date_created": cvStamp, "url": base + "Domains/" + parts[1] + "/Certificate"}
				}
				c["date_expires"], c["date_updated"], c["cert_in_validation"] = "2027-11-13T00:00:00Z", cvStamp, nil
				if had {
					c["cert_in_validation"] = map[string]any{"status": "pending"}
				}
				f.put("lscert", testAccount, parts[1], c)
				fwrite(w, 200, c)
			case "GET":
				c, ok := f.objs[certKey]
				if !ok {
					notFound()
					return
				}
				fwrite(w, 200, c)
			case "DELETE":
				if _, ok := f.objs[certKey]; !ok {
					notFound()
					return
				}
				f.del("lscert", testAccount, parts[1])
				w.WriteHeader(204)
			default:
				ferr(w, 405, "method not allowed")
			}
		case parts[2] == "ValidateDns" && r.Method == "GET":
			fwrite(w, 200, map[string]any{"domain_sid": parts[1], "is_valid": false, "reason": "CNAME record missing", "url": base + "Domains/" + parts[1] + "/ValidateDns"})
		case parts[2] == "MessagingServices" && len(parts) == 4:
			svc := parts[3]
			if _, ok := f.objs[key("msg", testAccount, svc)]; !ok {
				notFound()
				return
			}
			switch r.Method {
			case "POST":
				a := map[string]any{"domain_sid": parts[1], "messaging_service_sid": svc, "url": base + "MessagingServices/" + svc + "/Domain"}
				f.put("lsassoc", testAccount, svc, a)
				fwrite(w, 201, a)
			case "DELETE":
				if a, ok := f.objs[key("lsassoc", testAccount, svc)]; !ok || a["domain_sid"] != parts[1] {
					notFound()
					return
				}
				f.del("lsassoc", testAccount, svc)
				w.WriteHeader(204)
			default:
				ferr(w, 405, "method not allowed")
			}
		default:
			notFound()
		}
	default:
		notFound()
	}
}
