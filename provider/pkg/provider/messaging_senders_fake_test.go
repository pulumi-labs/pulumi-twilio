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

// The Messaging Service pool member routes, served for the sender tests:
//
//	/messaging/v1/Services/{sid}/{ShortCodes|ChannelSenders|AlphaSenders|DestinationAlphaSenders}[/{sid}]
//
// The shapes follow twilio-oai spec/json/twilio_messaging_v1.json. Members are stored as kind "msnd"
// keyed by the service SID and <collection>/<sid>.

func (f *fakeTwilio) messagingPoolMember(w http.ResponseWriter, r *http.Request, form url.Values, svc, coll, sid string) {
	if sid == "" {
		if r.Method != "POST" {
			ferr(w, 405, "method not allowed")
			return
		}
		o := map[string]any{"account_sid": testAccount, "service_sid": svc, "date_created": cvStamp, "date_updated": cvStamp}
		switch coll {
		case "ShortCodes":
			id := form.Get("ShortCodeSid")
			if !strings.HasPrefix(id, "SC") {
				ferr(w, 400, "Missing or invalid ShortCodeSid")
				return
			}
			o["sid"], o["short_code"], o["country_code"], o["capabilities"] = id, "+12345", "US", []any{"SMS", "MMS"}
		case "ChannelSenders":
			id := form.Get("Sid")
			if !strings.HasPrefix(id, "XE") {
				ferr(w, 400, "Missing or invalid Sid")
				return
			}
			o["messaging_service_sid"] = svc
			o["sid"], o["sender"], o["sender_type"], o["country_code"] = id, "whatsapp:+14155550100", "whatsapp", "US"
		default:
			n := form.Get("AlphaSender")
			if n == "" || len(n) > 11 {
				ferr(w, 400, "AlphaSender must be 1 to 11 characters")
				return
			}
			o["sid"], o["alpha_sender"], o["capabilities"] = f.sid("AI"), n, []any{"SMS"}
			if coll == "DestinationAlphaSenders" {
				o["iso_country_code"] = nil
				if cc := form.Get("IsoCountryCode"); cc != "" {
					o["iso_country_code"] = strings.ToUpper(cc)
				}
			}
			for k, v := range f.objs {
				if strings.HasPrefix(k, "msnd/"+svc+"/"+coll+"/") && v["alpha_sender"] == n && v["iso_country_code"] == o["iso_country_code"] {
					ferr(w, 409, "The Alphanumeric Sender ID is already in the Messaging Service")
					return
				}
			}
		}
		id := o["sid"].(string)
		if _, dup := f.objs[key("msnd", svc, coll+"/"+id)]; dup {
			ferr(w, 409, "already in the Messaging Service")
			return
		}
		f.put("msnd", svc, coll+"/"+id, o)
		fwrite(w, 201, o)
		return
	}
	k := coll + "/" + sid
	o, ok := f.objs[key("msnd", svc, k)]
	if !ok {
		ferr(w, 404, "The requested resource "+r.URL.Path+" was not found")
		return
	}
	switch r.Method {
	case "GET":
		fwrite(w, 200, o)
	case "DELETE":
		f.del("msnd", svc, k)
		w.WriteHeader(204)
	default:
		ferr(w, 405, "method not allowed")
	}
}
