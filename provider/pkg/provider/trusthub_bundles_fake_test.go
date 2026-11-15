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

// The TrustHub regulatory bundles of the trusthub.twilio.com v1 fake: /v1/CustomerProfiles and
// /v1/TrustProducts with their /EntityAssignments, from twilio-oai spec/json/twilio_trusthub_v1.json.
// A bundle starts as a draft; a POST with Status=pending-review submits it, which Twilio only accepts
// once at least one entity is assigned (the fake's stand-in for the policy evaluation), and a bundle
// that is no longer a draft cannot be deleted.

var bundleKinds = map[string]struct{ sidKey, assignmentKind string }{
	"CustomerProfiles": {"customer_profile_sid", "th:CustomerProfiles:assign"},
	"TrustProducts":    {"trust_product_sid", "th:TrustProducts:assign"},
	"Bundles":          {"bundle_sid", "nr:Bundles:assign"}, // Numbers regulatory bundles
}

func (f *fakeTwilio) handleBundles(w http.ResponseWriter, r *http.Request, form url.Values, parts []string) {
	coll := parts[0]
	k := bundleKinds[coll]
	notFound := func() { ferr(w, 404, "The requested resource "+r.URL.Path+" was not found") }
	kind := "th:" + coll
	if len(parts) == 1 {
		switch r.Method {
		case "POST":
			required := []string{"FriendlyName", "Email", "PolicySid"}
			if coll == "Bundles" {
				required = []string{"FriendlyName", "Email"}
			}
			for _, req := range required {
				if form.Get(req) == "" {
					ferr(w, 400, req+" is required")
					return
				}
			}
			sid := f.sid("BU")
			o := map[string]any{"sid": sid, "account_sid": testAccount, "policy_sid": form.Get("PolicySid"), "friendly_name": form.Get("FriendlyName"),
				"email": form.Get("Email"), "status": "draft", "valid_until": nil, "errors": nil, "date_created": cvStamp, "date_updated": cvStamp,
				"url": "https://trusthub.twilio.com/v1/" + coll + "/" + sid}
			if cb := form.Get("StatusCallback"); cb != "" {
				o["status_callback"] = cb
			}
			if coll == "Bundles" {
				// Numbers bundles are created for a regulation, given or resolved from the country, type and end user type.
				delete(o, "policy_sid")
				if form.Get("RegulationSid") == "" && (form.Get("IsoCountry") == "" || form.Get("NumberType") == "" || form.Get("EndUserType") == "") {
					ferr(w, 400, "RegulationSid, or IsoCountry, NumberType and EndUserType, is required")
					return
				}
				o["regulation_sid"] = firstNonEmpty(form.Get("RegulationSid"), "RN00000000000000000000000000000077")
			}
			f.put(kind, testAccount, sid, o)
			fwrite(w, 201, o)
		default:
			ferr(w, 405, "method not allowed")
		}
		return
	}
	sid := parts[1]
	o, ok := f.objs[key(kind, testAccount, sid)]
	if !ok {
		notFound()
		return
	}
	if len(parts) == 2 {
		switch r.Method {
		case "GET":
			fwrite(w, 200, o)
		case "POST":
			if v := form.Get("Status"); v != "" {
				if v != "pending-review" || o["status"] != "draft" {
					ferr(w, 400, "Status cannot change from "+o["status"].(string)+" to "+v)
					return
				}
				if f.bundleAssignments(k.assignmentKind, sid) == 0 {
					ferr(w, 400, "The bundle is not compliant with the policy: no entities assigned")
					return
				}
				o["status"] = "pending-review"
			}
			for fld, field := range map[string]string{"FriendlyName": "friendly_name", "Email": "email", "StatusCallback": "status_callback"} {
				if v, ok := form[fld]; ok {
					o[field] = v[0]
				}
			}
			o["date_updated"] = "2026-11-14T02:00:00Z"
			fwrite(w, 200, o)
		case "DELETE":
			if o["status"] != "draft" {
				ferr(w, 409, "Only a draft bundle can be deleted")
				return
			}
			f.del(kind, testAccount, sid)
			w.WriteHeader(204)
		default:
			ferr(w, 405, "method not allowed")
		}
		return
	}
	seg := "EntityAssignments"
	if coll == "Bundles" {
		seg = "ItemAssignments"
	}
	if parts[2] == "ChannelEndpointAssignments" && coll != "Bundles" && len(parts) <= 4 {
		f.handleChannelEndpointAssignments(w, r, form, k.sidKey, coll, parts, o, notFound)
		return
	}
	if parts[2] != seg || len(parts) > 4 {
		notFound()
		return
	}
	if len(parts) == 3 {
		if r.Method != "POST" {
			ferr(w, 405, "method not allowed")
			return
		}
		obj := form.Get("ObjectSid")
		if obj == "" {
			ferr(w, 400, "ObjectSid is required")
			return
		}
		if !strings.HasPrefix(obj, "IT") && !strings.HasPrefix(obj, "RD") {
			ferr(w, 400, "ObjectSid must be an end user or a supporting document")
			return
		}
		a := map[string]any{"sid": f.sid("BV"), "account_sid": testAccount, "object_sid": obj, k.sidKey: sid, "date_created": cvStamp}
		f.put(k.assignmentKind, testAccount, sid+"/"+a["sid"].(string), a)
		fwrite(w, 201, a)
		return
	}
	a, ok := f.objs[key(k.assignmentKind, testAccount, sid+"/"+parts[3])]
	if !ok {
		notFound()
		return
	}
	switch r.Method {
	case "GET":
		fwrite(w, 200, a)
	case "DELETE":
		if o["status"] != "draft" {
			ferr(w, 409, "Entities cannot be removed from a bundle that is in review")
			return
		}
		f.del(k.assignmentKind, testAccount, sid+"/"+parts[3])
		w.WriteHeader(204)
	default:
		ferr(w, 405, "method not allowed")
	}
}

func firstNonEmpty(vals ...string) string {
	for _, v := range vals {
		if v != "" {
			return v
		}
	}
	return ""
}

// handleNumbersReg serves numbers.twilio.com /v2/RegulatoryCompliance: end users, supporting documents and their types
// like TrustHub, bundles like the TrustHub bundles with ItemAssignments, and the read-only regulations.
func (f *fakeTwilio) handleNumbersReg(w http.ResponseWriter, r *http.Request, form url.Values, parts []string) {
	switch parts[0] {
	case "Regulations":
		var items []map[string]any
		for _, k := range f.order {
			o, ok := f.objs[k]
			if !ok || !strings.HasPrefix(k, "nr:Regulations/") {
				continue
			}
			if c := r.URL.Query().Get("IsoCountry"); c != "" && o["iso_country"] != c {
				continue
			}
			if len(parts) == 2 && o["sid"] != parts[1] {
				continue
			}
			items = append(items, o)
		}
		if len(parts) == 2 {
			if len(items) == 0 {
				ferr(w, 404, "The requested resource "+r.URL.Path+" was not found")
				return
			}
			fwrite(w, 200, items[0])
			return
		}
		metaPage(w, r, "results", items, thFakePageSize)
	case "Bundles":
		f.handleBundles(w, r, form, parts)
	default:
		f.handleTrusthub(w, r, form, parts)
	}
}

func (f *fakeTwilio) bundleAssignments(kind, bundle string) int {
	n := 0
	for k := range f.objs {
		if strings.HasPrefix(k, kind+"/"+testAccount+"/"+bundle+"/") {
			n++
		}
	}
	return n
}

// handleChannelEndpointAssignments serves /v1/{CustomerProfiles,TrustProducts}/{sid}/ChannelEndpointAssignments: create,
// fetch and delete, no update. A second assignment of the same endpoint to the bundle is a 409, and a bundle that is
// no longer a draft refuses to lose an assignment.
func (f *fakeTwilio) handleChannelEndpointAssignments(w http.ResponseWriter, r *http.Request, form url.Values, sidKey, coll string, parts []string, bundle map[string]any, notFound func()) {
	sid := parts[1]
	kind := "th:" + coll + ":endpoint"
	if len(parts) == 3 {
		if r.Method != "POST" {
			ferr(w, 405, "method not allowed")
			return
		}
		typ, ep := form.Get("ChannelEndpointType"), form.Get("ChannelEndpointSid")
		if typ == "" || ep == "" {
			ferr(w, 400, "ChannelEndpointType and ChannelEndpointSid are required")
			return
		}
		if len(ep) != 34 {
			ferr(w, 400, "ChannelEndpointSid must be 34 characters")
			return
		}
		for k, a := range f.objs {
			if strings.HasPrefix(k, kind+"/") && a[sidKey] == sid && a["channel_endpoint_sid"] == ep {
				ferr(w, 409, "The channel endpoint is already assigned")
				return
			}
		}
		a := map[string]any{"sid": f.sid("RA"), "account_sid": testAccount, "channel_endpoint_type": typ, "channel_endpoint_sid": ep, sidKey: sid, "date_created": cvStamp,
			"url": "https://trusthub.twilio.com/v1/" + coll + "/" + sid + "/ChannelEndpointAssignments/x"}
		f.put(kind, testAccount, sid+"/"+a["sid"].(string), a)
		fwrite(w, 201, a)
		return
	}
	a, ok := f.objs[key(kind, testAccount, sid+"/"+parts[3])]
	if !ok {
		notFound()
		return
	}
	switch r.Method {
	case "GET":
		fwrite(w, 200, a)
	case "DELETE":
		if bundle["status"] != "draft" {
			ferr(w, 409, "Channel endpoints cannot be removed from a bundle that is in review")
			return
		}
		f.del(kind, testAccount, sid+"/"+parts[3])
		w.WriteHeader(204)
	default:
		ferr(w, 405, "method not allowed")
	}
}
