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

// The A2P 10DLC routes of the fake: /messaging/v1/a2p/BrandRegistrations[/{sid}]. Brand Registrations
// have no delete. The shapes follow twilio-oai spec/json/twilio_messaging_v1.json (1.0.0).

func (f *fakeTwilio) a2pList(coll string) []map[string]any {
	out := []map[string]any{}
	prefix := "a2p/" + testAccount + "/" + coll + "/"
	for _, fk := range f.order {
		if o, ok := f.objs[fk]; ok && strings.HasPrefix(fk, prefix) {
			out = append(out, o)
		}
	}
	return out
}

func (f *fakeTwilio) handleA2P(w http.ResponseWriter, r *http.Request, form url.Values, parts []string) {
	if parts[0] != "BrandRegistrations" {
		ferr(w, 404, "The requested resource "+r.URL.Path+" was not found")
		return
	}
	const base = "https://messaging.twilio.com/v1/a2p/BrandRegistrations/"
	switch {
	case len(parts) == 1 && r.Method == "POST":
		for _, n := range []string{"CustomerProfileBundleSid", "A2PProfileBundleSid"} {
			if !strings.HasPrefix(form.Get(n), "BU") {
				ferr(w, 400, "Missing or invalid "+n)
				return
			}
		}
		id := f.sid("BN")
		o := map[string]any{"sid": id, "account_sid": testAccount, "customer_profile_bundle_sid": form.Get("CustomerProfileBundleSid"),
			"a2p_profile_bundle_sid": form.Get("A2PProfileBundleSid"), "brand_type": "STANDARD", "status": "PENDING",
			"tcr_id": nil, "failure_reason": nil, "errors": nil, "url": base + id, "brand_score": nil, "brand_feedback": nil,
			"identity_status": nil, "russell_3000": nil, "government_entity": nil, "tax_exempt_status": nil,
			"skip_automatic_sec_vet": false, "mock": false, "date_created": cvStamp, "date_updated": cvStamp}
		if msg := applyForm(o, form, []fakeField{{"BrandType", "brand_type", "s"}, {"Mock", "mock", "b"}, {"SkipAutomaticSecVet", "skip_automatic_sec_vet", "b"}}); msg != "" {
			ferr(w, 400, msg)
			return
		}
		f.put("a2p", testAccount, "BrandRegistrations/"+id, o)
		fwrite(w, 201, o)
	case len(parts) == 1 && r.Method == "GET":
		metaPage(w, r, "data", f.a2pList("BrandRegistrations"), 2)
	case len(parts) == 2 && r.Method == "GET":
		o, ok := f.objs[key("a2p", testAccount, "BrandRegistrations/"+parts[1])]
		if !ok {
			ferr(w, 404, "The requested resource "+r.URL.Path+" was not found")
			return
		}
		fwrite(w, 200, o)
	default:
		ferr(w, 405, "method not allowed")
	}
}

// messagingCompliance serves /messaging/v1/Services/{sid}/Compliance/Usa2p[/Usecases|/{sid}]: A2P
// campaigns, stored as kind "usa2p" keyed by the service SID and the campaign SID.
func (f *fakeTwilio) messagingCompliance(w http.ResponseWriter, r *http.Request, form url.Values, svc string, parts []string) {
	notFound := func() { ferr(w, 404, "The requested resource "+r.URL.Path+" was not found") }
	if len(parts) == 0 || parts[0] != "Usa2p" {
		notFound()
		return
	}
	list := func(k string) []any {
		out := []any{}
		for _, v := range form[k] {
			out = append(out, v)
		}
		return out
	}
	lists := map[string]string{"MessageSamples": "message_samples", "OptInKeywords": "opt_in_keywords", "OptOutKeywords": "opt_out_keywords", "HelpKeywords": "help_keywords"}
	fields := []fakeField{{"Description", "description", "s"}, {"MessageFlow", "message_flow", "s"}, {"HasEmbeddedLinks", "has_embedded_links", "b"},
		{"HasEmbeddedPhone", "has_embedded_phone", "b"}, {"OptInMessage", "opt_in_message", "s"}, {"OptOutMessage", "opt_out_message", "s"},
		{"HelpMessage", "help_message", "s"}, {"SubscriberOptIn", "subscriber_opt_in", "b"}, {"AgeGated", "age_gated", "b"},
		{"DirectLending", "direct_lending", "b"}, {"PrivacyPolicyUrl", "privacy_policy_url", "s"}, {"TermsAndConditionsUrl", "terms_and_conditions_url", "s"}}
	apply := func(o map[string]any) string {
		if msg := applyForm(o, form, fields); msg != "" {
			return msg
		}
		for param, k := range lists {
			if _, ok := form[param]; ok {
				o[k] = list(param)
			}
		}
		return ""
	}
	prefix := "usa2p/" + svc + "/"
	switch {
	case len(parts) == 1 && r.Method == "POST":
		for _, n := range []string{"BrandRegistrationSid", "Description", "MessageFlow", "UsAppToPersonUsecase", "HasEmbeddedLinks", "HasEmbeddedPhone"} {
			if form.Get(n) == "" {
				ferr(w, 400, "Missing required parameter "+n)
				return
			}
		}
		if len(form["MessageSamples"]) == 0 {
			ferr(w, 400, "Missing required parameter MessageSamples")
			return
		}
		if _, ok := f.objs[key("a2p", testAccount, "BrandRegistrations/"+form.Get("BrandRegistrationSid"))]; !ok {
			ferr(w, 404, "Brand "+form.Get("BrandRegistrationSid")+" was not found")
			return
		}
		id := f.sid("QE")
		o := map[string]any{"sid": id, "account_sid": testAccount, "brand_registration_sid": form.Get("BrandRegistrationSid"), "messaging_service_sid": svc,
			"us_app_to_person_usecase": form.Get("UsAppToPersonUsecase"), "message_samples": []any{}, "opt_in_message": nil, "opt_out_message": nil,
			"help_message": nil, "opt_in_keywords": nil, "opt_out_keywords": nil, "help_keywords": nil, "subscriber_opt_in": false, "age_gated": false,
			"direct_lending": false, "privacy_policy_url": nil, "terms_and_conditions_url": nil, "campaign_status": "IN_PROGRESS", "campaign_id": nil,
			"is_externally_registered": false, "rate_limits": nil, "mock": false, "errors": nil, "date_created": cvStamp, "date_updated": cvStamp,
			"url": "https://messaging.twilio.com/v1/Services/" + svc + "/Compliance/Usa2p/" + id}
		if msg := apply(o); msg != "" {
			ferr(w, 400, msg)
			return
		}
		f.put("usa2p", svc, id, o)
		fwrite(w, 201, o)
	case len(parts) == 1 && r.Method == "GET":
		items := []map[string]any{}
		for _, fk := range f.order {
			if o, ok := f.objs[fk]; ok && strings.HasPrefix(fk, prefix) {
				items = append(items, o)
			}
		}
		metaPage(w, r, "compliance", items, 2)
	case len(parts) == 2 && parts[1] == "Usecases" && r.Method == "GET":
		if bn := r.URL.Query().Get("BrandRegistrationSid"); bn != "" && !strings.HasPrefix(bn, "BN") {
			ferr(w, 400, "Invalid BrandRegistrationSid")
			return
		}
		fwrite(w, 200, map[string]any{"us_app_to_person_usecases": []any{
			map[string]any{"code": "2FA", "name": "Two-Factor Authentication", "description": "One-time passcodes", "post_approval_required": false},
			map[string]any{"code": "MARKETING", "name": "Marketing", "description": "Promotions", "post_approval_required": true}}})
	case len(parts) == 2:
		o, ok := f.objs[key("usa2p", svc, parts[1])]
		if !ok {
			notFound()
			return
		}
		switch r.Method {
		case "GET":
			fwrite(w, 200, o)
		case "POST":
			for _, n := range []string{"HasEmbeddedLinks", "HasEmbeddedPhone", "MessageFlow", "Description", "AgeGated", "DirectLending"} {
				if form.Get(n) == "" {
					ferr(w, 400, "Missing required parameter "+n)
					return
				}
			}
			if len(form["MessageSamples"]) == 0 {
				ferr(w, 400, "Missing required parameter MessageSamples")
				return
			}
			if msg := apply(o); msg != "" {
				ferr(w, 400, msg)
				return
			}
			o["date_updated"] = cvStamp
			fwrite(w, 200, o)
		case "DELETE":
			f.del("usa2p", svc, parts[1])
			w.WriteHeader(204)
		default:
			ferr(w, 405, "method not allowed")
		}
	default:
		notFound()
	}
}
