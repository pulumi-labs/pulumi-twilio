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
	"net/url"
	"strings"
)

// The Intelligence v2 routes, served for the Intelligence tests:
//
//	/intelligence/v2/Services[/{sid}], /intelligence/v2/Operators/Custom[/{sid}]
//
// The shapes follow twilio-oai spec/json/twilio_intelligence_v2.json. Objects are stored as kind
// "iq" keyed <collection>/<sid>. Like Twilio, a service never accepts a language on update.

var intelligenceServiceFakeFields = []fakeField{
	{"UniqueName", "unique_name", "s"}, {"FriendlyName", "friendly_name", "s"}, {"LanguageCode", "language_code", "s"},
	{"AutoTranscribe", "auto_transcribe", "b"}, {"AutoRedaction", "auto_redaction", "b"}, {"MediaRedaction", "media_redaction", "b"},
	{"WebhookUrl", "webhook_url", "u"}, {"WebhookHttpMethod", "webhook_http_method", "m"}, {"EncryptionCredentialSid", "encryption_credential_sid", "u"},
}

func (f *fakeTwilio) handleIntelligence(w http.ResponseWriter, r *http.Request, form url.Values, rest string) {
	var coll, sid string
	switch {
	case rest == "Services" || strings.HasPrefix(rest, "Services/"):
		coll, sid = "Services", strings.Trim(strings.TrimPrefix(rest, "Services"), "/")
	case rest == "Operators/Custom" || strings.HasPrefix(rest, "Operators/Custom/"):
		coll, sid = "Operators/Custom", strings.Trim(strings.TrimPrefix(rest, "Operators/Custom"), "/")
	default:
		ferr(w, 404, "The requested resource "+r.URL.Path+" was not found")
		return
	}
	prefix := "GA"
	if coll != "Services" {
		prefix = "LY"
	}
	if sid == "" {
		if r.Method != "POST" {
			ferr(w, 405, "method not allowed")
			return
		}
		id := f.sid(prefix)
		o := map[string]any{"sid": id, "account_sid": testAccount, "date_created": cvStamp, "date_updated": cvStamp}
		if coll == "Services" {
			if form.Get("UniqueName") == "" {
				ferr(w, 400, "Missing required parameter UniqueName")
				return
			}
			for _, k := range []string{"friendly_name", "webhook_url", "webhook_http_method", "encryption_credential_sid"} {
				o[k] = nil
			}
			o["language_code"] = "en-US"
			o["auto_transcribe"], o["auto_redaction"], o["media_redaction"], o["data_logging"] = false, false, false, true
			if msg := applyForm(o, form, intelligenceServiceFakeFields); msg != "" {
				ferr(w, 400, msg)
				return
			}
		} else {
			for _, req := range []string{"FriendlyName", "OperatorType", "Config"} {
				if form.Get(req) == "" {
					ferr(w, 400, "Missing required parameter "+req)
					return
				}
			}
			var cfg map[string]any
			if json.Unmarshal([]byte(form.Get("Config")), &cfg) != nil {
				ferr(w, 400, "Config must be a JSON object")
				return
			}
			o["friendly_name"], o["operator_type"], o["config"] = form.Get("FriendlyName"), form.Get("OperatorType"), cfg
			o["description"], o["author"], o["availability"] = "", testAccount, "internal"
		}
		f.put("iq", testAccount, coll+"/"+id, o)
		fwrite(w, 201, o)
		return
	}
	k := coll + "/" + sid
	o, ok := f.objs[key("iq", testAccount, k)]
	if !ok || !strings.HasPrefix(sid, prefix) {
		ferr(w, 404, "The requested resource "+r.URL.Path+" was not found")
		return
	}
	switch r.Method {
	case "GET":
		fwrite(w, 200, o)
	case "POST":
		if coll == "Services" {
			if _, sent := form["LanguageCode"]; sent {
				ferr(w, 400, "LanguageCode cannot be updated")
				return
			}
			if msg := applyForm(o, form, intelligenceServiceFakeFields); msg != "" {
				ferr(w, 400, msg)
				return
			}
		} else {
			if _, sent := form["OperatorType"]; sent {
				ferr(w, 400, "OperatorType cannot be updated")
				return
			}
			var cfg map[string]any
			if json.Unmarshal([]byte(form.Get("Config")), &cfg) != nil {
				ferr(w, 400, "Config must be a JSON object")
				return
			}
			o["friendly_name"], o["config"] = form.Get("FriendlyName"), cfg
		}
		o["date_updated"] = cvUpdatedStamp
		fwrite(w, 200, o)
	case "DELETE":
		f.del("iq", testAccount, k)
		w.WriteHeader(204)
	default:
		ferr(w, 405, "method not allowed")
	}
}
