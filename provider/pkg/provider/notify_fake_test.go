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

// The Notify v1 routes, served for the Notify tests:
//
//	/notify/v1/Services[/{sid}], /notify/v1/Credentials[/{sid}]
//
// The shapes follow twilio-oai spec/json/twilio_notify_v1.json. Objects are stored as kind "nt"
// keyed <collection>/<sid>.

type notifySpec struct {
	prefix   string
	required []string
	create   []fakeField
	update   []fakeField
}

var notifyServiceFields = []fakeField{
	{"FriendlyName", "friendly_name", "s"}, {"ApnCredentialSid", "apn_credential_sid", "u"}, {"GcmCredentialSid", "gcm_credential_sid", "u"},
	{"FcmCredentialSid", "fcm_credential_sid", "u"}, {"MessagingServiceSid", "messaging_service_sid", "u"},
	{"DefaultApnNotificationProtocolVersion", "default_apn_notification_protocol_version", "u"},
	{"DefaultGcmNotificationProtocolVersion", "default_gcm_notification_protocol_version", "u"},
	{"DefaultFcmNotificationProtocolVersion", "default_fcm_notification_protocol_version", "u"},
	{"LogEnabled", "log_enabled", "b"}, {"DeliveryCallbackUrl", "delivery_callback_url", "u"}, {"DeliveryCallbackEnabled", "delivery_callback_enabled", "b"},
}

var notifySpecs = map[string]notifySpec{
	"Services": {"IS", nil, notifyServiceFields, nil},
	"Credentials": {"CR", []string{"Type"},
		[]fakeField{{"Type", "type", "s"}, {"FriendlyName", "friendly_name", "s"}, {"Sandbox", "sandbox", "b"}},
		[]fakeField{{"FriendlyName", "friendly_name", "s"}, {"Sandbox", "sandbox", "b"}}},
}

func (f *fakeTwilio) handleNotify(w http.ResponseWriter, r *http.Request, form url.Values, rest string) {
	parts := strings.Split(rest, "/")
	spec, ok := notifySpecs[parts[0]]
	if !ok || len(parts) > 2 {
		ferr(w, 404, "The requested resource "+r.URL.Path+" was not found")
		return
	}
	coll := parts[0]
	if len(parts) == 1 {
		if r.Method != "POST" {
			ferr(w, 405, "method not allowed")
			return
		}
		for _, req := range spec.required {
			if form.Get(req) == "" {
				ferr(w, 400, "Missing required parameter "+req)
				return
			}
		}
		id := f.sid(spec.prefix)
		o := map[string]any{"sid": id, "account_sid": testAccount, "date_created": cvStamp, "date_updated": cvStamp, "friendly_name": nil}
		fields := spec.create
		if coll == "Services" {
			for _, k := range []string{"apn_credential_sid", "gcm_credential_sid", "fcm_credential_sid", "messaging_service_sid",
				"default_apn_notification_protocol_version", "default_gcm_notification_protocol_version", "default_fcm_notification_protocol_version", "delivery_callback_url"} {
				o[k] = nil
			}
			o["log_enabled"], o["delivery_callback_enabled"] = true, false
		} else {
			o["sandbox"] = "false"
		}
		if msg := applyForm(o, form, fields); msg != "" {
			ferr(w, 400, msg)
			return
		}
		f.put("nt", testAccount, coll+"/"+id, o)
		fwrite(w, 201, o)
		return
	}
	sid := parts[1]
	k := coll + "/" + sid
	o, ok := f.objs[key("nt", testAccount, k)]
	if !ok || !strings.HasPrefix(sid, spec.prefix) {
		ferr(w, 404, "The requested resource "+r.URL.Path+" was not found")
		return
	}
	switch r.Method {
	case "GET":
		fwrite(w, 200, o)
	case "POST":
		fields := spec.update
		if fields == nil {
			fields = spec.create
		}
		if msg := applyForm(o, form, fields); msg != "" {
			ferr(w, 400, msg)
			return
		}
		o["date_updated"] = cvUpdatedStamp
		fwrite(w, 200, o)
	case "DELETE":
		f.del("nt", testAccount, k)
		w.WriteHeader(204)
	default:
		ferr(w, 405, "method not allowed")
	}
}
