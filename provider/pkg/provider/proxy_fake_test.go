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

// The Proxy v1 routes, served for the Proxy tests:
//
//	/proxy/v1/Services[/{sid}], /proxy/v1/Services/{sid}/PhoneNumbers[/{sid}]
//
// The shapes follow twilio-oai spec/json/twilio_proxy_v1.json. Objects are stored as kind "px"
// keyed Services/<sid> and Services/<serviceSid>/PhoneNumbers/<sid>.

var proxyServiceFakeFields = []fakeField{
	{"UniqueName", "unique_name", "s"}, {"DefaultTtl", "default_ttl", "i"}, {"CallbackUrl", "callback_url", "u"},
	{"GeoMatchLevel", "geo_match_level", "s"}, {"NumberSelectionBehavior", "number_selection_behavior", "s"},
	{"InterceptCallbackUrl", "intercept_callback_url", "u"}, {"OutOfSessionCallbackUrl", "out_of_session_callback_url", "u"},
	{"ChatInstanceSid", "chat_instance_sid", "u"},
}

func (f *fakeTwilio) handleProxy(w http.ResponseWriter, r *http.Request, form url.Values, rest string) {
	parts := strings.Split(rest, "/")
	notFound := func() { ferr(w, 404, "The requested resource "+r.URL.Path+" was not found") }
	if parts[0] != "Services" || len(parts) > 4 || (len(parts) > 2 && parts[2] != "PhoneNumbers") {
		notFound()
		return
	}
	switch len(parts) {
	case 1:
		f.proxyServiceCreate(w, r, form)
	case 2:
		f.proxyServiceItem(w, r, form, parts[1], notFound)
	default:
		if _, ok := f.objs[key("px", testAccount, "Services/"+parts[1])]; !ok {
			notFound()
			return
		}
		sid := ""
		if len(parts) == 4 {
			sid = parts[3]
		}
		f.proxyNumber(w, r, form, parts[1], sid, notFound)
	}
}

func (f *fakeTwilio) proxyServiceCreate(w http.ResponseWriter, r *http.Request, form url.Values) {
	if r.Method != "POST" {
		ferr(w, 405, "method not allowed")
		return
	}
	if form.Get("UniqueName") == "" {
		ferr(w, 400, "Missing required parameter UniqueName")
		return
	}
	id := f.sid("KS")
	o := map[string]any{"sid": id, "account_sid": testAccount, "date_created": cvStamp, "date_updated": cvStamp, "unique_name": nil,
		"chat_instance_sid": nil, "callback_url": nil, "default_ttl": float64(0), "number_selection_behavior": "prefer-sticky",
		"geo_match_level": "country", "intercept_callback_url": nil, "out_of_session_callback_url": nil}
	if msg := applyForm(o, form, proxyServiceFakeFields); msg != "" {
		ferr(w, 400, msg)
		return
	}
	f.put("px", testAccount, "Services/"+id, o)
	fwrite(w, 201, o)
}

func (f *fakeTwilio) proxyServiceItem(w http.ResponseWriter, r *http.Request, form url.Values, sid string, notFound func()) {
	k := "Services/" + sid
	o, ok := f.objs[key("px", testAccount, k)]
	if !ok || !strings.HasPrefix(sid, "KS") {
		notFound()
		return
	}
	switch r.Method {
	case "GET":
		fwrite(w, 200, o)
	case "POST":
		if msg := applyForm(o, form, proxyServiceFakeFields); msg != "" {
			ferr(w, 400, msg)
			return
		}
		o["date_updated"] = cvUpdatedStamp
		fwrite(w, 200, o)
	case "DELETE":
		f.del("px", testAccount, k)
		w.WriteHeader(204)
	default:
		ferr(w, 405, "method not allowed")
	}
}

func (f *fakeTwilio) proxyNumber(w http.ResponseWriter, r *http.Request, form url.Values, svc, sid string, notFound func()) {
	if sid == "" {
		if r.Method != "POST" {
			ferr(w, 405, "method not allowed")
			return
		}
		pnSid, number := form.Get("Sid"), form.Get("PhoneNumber")
		if (pnSid == "") == (number == "") {
			ferr(w, 400, "Exactly one of Sid and PhoneNumber is required")
			return
		}
		if pnSid == "" {
			pnSid = f.sid("PN")
		} else {
			number = "+1415555" + pnSid[len(pnSid)-4:]
		}
		o := map[string]any{"sid": pnSid, "account_sid": testAccount, "service_sid": svc, "date_created": cvStamp, "date_updated": cvStamp,
			"phone_number": number, "friendly_name": number, "iso_country": "US", "is_reserved": false, "in_use": float64(0)}
		if msg := applyForm(o, form, []fakeField{{"IsReserved", "is_reserved", "b"}}); msg != "" {
			ferr(w, 400, msg)
			return
		}
		f.put("px", testAccount, "Services/"+svc+"/PhoneNumbers/"+pnSid, o)
		fwrite(w, 201, o)
		return
	}
	k := "Services/" + svc + "/PhoneNumbers/" + sid
	o, ok := f.objs[key("px", testAccount, k)]
	if !ok {
		notFound()
		return
	}
	switch r.Method {
	case "GET":
		fwrite(w, 200, o)
	case "POST":
		if msg := applyForm(o, form, []fakeField{{"IsReserved", "is_reserved", "b"}}); msg != "" {
			ferr(w, 400, msg)
			return
		}
		o["date_updated"] = cvUpdatedStamp
		fwrite(w, 200, o)
	case "DELETE":
		f.del("px", testAccount, k)
		w.WriteHeader(204)
	default:
		ferr(w, 405, "method not allowed")
	}
}
