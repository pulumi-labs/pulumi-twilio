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

// The toll-free verification routes of the fake: /messaging/v1/Tollfree/Verifications[/{sid}].
// Verifications are stored as kind "tfv". An edit is only accepted while edit_allowed is true. The
// shapes follow twilio-oai spec/json/twilio_messaging_v1.json (1.0.0).

var tollfreeFields = []fakeField{{"BusinessName", "business_name", "s"}, {"BusinessWebsite", "business_website", "s"},
	{"NotificationEmail", "notification_email", "s"}, {"UseCaseSummary", "use_case_summary", "s"}, {"ProductionMessageSample", "production_message_sample", "s"},
	{"OptInType", "opt_in_type", "s"}, {"MessageVolume", "message_volume", "s"}, {"CustomerProfileSid", "customer_profile_sid", "s"},
	{"ExternalReferenceId", "external_reference_id", "s"}, {"BusinessStreetAddress", "business_street_address", "s"},
	{"BusinessCity", "business_city", "s"}, {"BusinessCountry", "business_country", "s"}, {"BusinessContactEmail", "business_contact_email", "s"},
	{"BusinessType", "business_type", "s"}, {"BusinessRegistrationNumber", "business_registration_number", "s"},
	{"BusinessRegistrationAuthority", "business_registration_authority", "s"}, {"PrivacyPolicyUrl", "privacy_policy_url", "s"},
	{"VettingProvider", "vetting_provider", "s"}, {"AgeGatedContent", "age_gated_content", "b"}}

func (f *fakeTwilio) handleTollfree(w http.ResponseWriter, r *http.Request, form url.Values, parts []string) {
	notFound := func() { ferr(w, 404, "The requested resource "+r.URL.Path+" was not found") }
	if parts[0] != "Verifications" || len(parts) > 2 {
		notFound()
		return
	}
	lists := map[string]string{"UseCaseCategories": "use_case_categories", "OptInImageUrls": "opt_in_image_urls", "OptInKeywords": "opt_in_keywords"}
	apply := func(o map[string]any) string {
		if msg := applyForm(o, form, tollfreeFields); msg != "" {
			return msg
		}
		for param, k := range lists {
			if v, ok := form[param]; ok {
				l := []any{}
				for _, e := range v {
					l = append(l, e)
				}
				o[k] = l
			}
		}
		return ""
	}
	switch {
	case len(parts) == 1 && r.Method == "POST":
		for _, n := range []string{"BusinessName", "BusinessWebsite", "NotificationEmail", "UseCaseSummary", "ProductionMessageSample", "OptInType", "MessageVolume", "TollfreePhoneNumberSid"} {
			if form.Get(n) == "" {
				ferr(w, 400, "Missing required parameter "+n)
				return
			}
		}
		if len(form["UseCaseCategories"]) == 0 || len(form["OptInImageUrls"]) == 0 {
			ferr(w, 400, "Missing required parameter UseCaseCategories or OptInImageUrls")
			return
		}
		id := f.sid("HH")
		o := map[string]any{"sid": id, "account_sid": testAccount, "customer_profile_sid": nil, "trust_product_sid": nil, "regulated_item_sid": nil,
			"tollfree_phone_number_sid": form.Get("TollfreePhoneNumberSid"), "tollfree_phone_number": "+18005550100", "status": "PENDING_REVIEW",
			"external_reference_id": nil, "business_street_address": nil, "business_city": nil, "business_country": nil, "business_contact_email": nil,
			"business_type": nil, "business_registration_number": nil, "business_registration_authority": nil, "privacy_policy_url": nil,
			"vetting_provider": nil, "vetting_id": nil, "vetting_id_expiration": nil, "age_gated_content": false, "opt_in_keywords": nil,
			"rejection_reason": nil, "rejection_reasons": nil, "error_code": nil, "edit_allowed": false, "edit_expiration": nil, "resource_links": nil,
			"date_created": cvStamp, "date_updated": cvStamp, "url": "https://messaging.twilio.com/v1/Tollfree/Verifications/" + id}
		if msg := apply(o); msg != "" {
			ferr(w, 400, msg)
			return
		}
		f.put("tfv", testAccount, id, o)
		fwrite(w, 201, o)
	case len(parts) == 1 && r.Method == "GET":
		items := []map[string]any{}
		q := r.URL.Query()
		for _, fk := range f.order {
			o, ok := f.objs[fk]
			if !ok || !strings.HasPrefix(fk, "tfv/"+testAccount+"/") {
				continue
			}
			if v := q.Get("Status"); v != "" && o["status"] != v {
				continue
			}
			if v := q.Get("TollfreePhoneNumberSid"); v != "" && o["tollfree_phone_number_sid"] != v {
				continue
			}
			items = append(items, o)
		}
		metaPage(w, r, "verifications", items, 2)
	case len(parts) == 2:
		o, ok := f.objs[key("tfv", testAccount, parts[1])]
		if !ok {
			notFound()
			return
		}
		switch r.Method {
		case "GET":
			fwrite(w, 200, o)
		case "POST":
			if allowed, _ := o["edit_allowed"].(bool); !allowed {
				ferr(w, 400, "The verification cannot be edited")
				return
			}
			for _, n := range []string{"TollfreePhoneNumberSid", "CustomerProfileSid", "ExternalReferenceId"} {
				if _, ok := form[n]; ok {
					ferr(w, 400, "Unknown parameter "+n)
					return
				}
			}
			if msg := apply(o); msg != "" {
				ferr(w, 400, msg)
				return
			}
			o["status"], o["date_updated"] = "PENDING_REVIEW", cvStamp
			fwrite(w, 200, o)
		case "DELETE":
			f.del("tfv", testAccount, parts[1])
			w.WriteHeader(204)
		default:
			ferr(w, 405, "method not allowed")
		}
	default:
		ferr(w, 405, "method not allowed")
	}
}
