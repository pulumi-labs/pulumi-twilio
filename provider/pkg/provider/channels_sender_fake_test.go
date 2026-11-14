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
	"fmt"
	"net/http"
	"sort"
	"strings"
)

// The Messaging v2 Channels Senders routes, served for the ChannelsSender tests:
//
//	/messaging/v2/Channels/Senders[/{sid}]
//
// The shapes follow twilio-oai spec/json/twilio_messaging_v2.json. They take JSON bodies. Senders
// are stored as kind "cs" keyed by SID. A new profile.name on update is not applied: like Meta's
// review it stays pending (pending_display_name, PENDING_REVIEW) while profile.name keeps the
// active name; a test completes or fails the review by mutating the stored object.

// channelsSenderPageSize is how many senders the fake returns per list page.
const channelsSenderPageSize = 2

func (f *fakeTwilio) handleChannelsSenders(w http.ResponseWriter, r *http.Request, body []byte, rest string) {
	if rest == "" {
		switch r.Method {
		case "POST":
			f.createChannelsSender(w, r, body)
		case "GET":
			f.listChannelsSenders(w, r)
		default:
			ferr(w, 405, "method not allowed")
		}
		return
	}
	o, ok := f.objs[key("cs", testAccount, rest)]
	if !ok || !strings.HasPrefix(rest, "XE") {
		ferr(w, 404, "The requested resource "+r.URL.Path+" was not found")
		return
	}
	switch r.Method {
	case "GET":
		fwrite(w, 200, o)
	case "POST":
		var req map[string]any
		if err := json.Unmarshal(body, &req); err != nil {
			ferr(w, 400, "invalid JSON body")
			return
		}
		if _, ok := req["sender_id"]; ok {
			ferr(w, 400, "sender_id cannot be updated")
			return
		}
		if msg := applyChannelsSender(o, req); msg != "" {
			ferr(w, 400, msg)
			return
		}
		out := clone(o)
		out["display_name_status"] = "no_change"
		delete(out, "pending_display_name")
		delete(out, "pending_display_name_status")
		delete(out, "pending_display_name_status_date")
		fwrite(w, 202, out)
	case "DELETE":
		if !strings.HasPrefix(fmt.Sprint(o["sender_id"]), "whatsapp:") {
			ferr(w, 400, "only WhatsApp senders can be deleted")
			return
		}
		f.del("cs", testAccount, rest)
		w.WriteHeader(204)
	default:
		ferr(w, 405, "method not allowed")
	}
}

func (f *fakeTwilio) createChannelsSender(w http.ResponseWriter, r *http.Request, body []byte) {
	var req map[string]any
	if err := json.Unmarshal(body, &req); err != nil {
		ferr(w, 400, "invalid JSON body")
		return
	}
	id, _ := req["sender_id"].(string)
	if id == "" {
		ferr(w, 400, "sender_id is required")
		return
	}
	sid := f.sid("XE")
	o := map[string]any{"sid": sid, "status": "CREATING", "sender_id": id, "friendly_name": nil,
		"configuration": map[string]any{"verification_method": "sms"}, "webhook": map[string]any{},
		"profile": map[string]any{}, "properties": map[string]any{}, "offline_reasons": []any{},
		"url": "https://messaging.twilio.com/v2/Channels/Senders/" + sid}
	delete(req, "sender_id")
	if msg := applyChannelsSender(o, req); msg != "" {
		ferr(w, 400, msg)
		return
	}
	if name, _ := o["profile"].(map[string]any)["name"].(string); name == "" {
		ferr(w, 400, "profile.name is required for WhatsApp senders")
		return
	}
	f.put("cs", testAccount, sid, o)
	fwrite(w, 202, o)
}

// applyChannelsSender merges an update or create body into a stored sender.
func applyChannelsSender(o map[string]any, req map[string]any) string {
	for k, v := range req {
		switch k {
		case "friendly_name":
			o[k] = v
		case "configuration", "webhook":
			m, _ := v.(map[string]any)
			dst := o[k].(map[string]any)
			for ck, cv := range m {
				switch {
				case k == "configuration" && ck == "verification_code":
					if cv == "123456" {
						o["status"] = "ONLINE"
					} else {
						return "invalid verification code"
					}
				case k == "configuration" && !isOneOf(ck, "waba_id", "verification_method", "voice_application_sid", "account_type"):
					return "unknown configuration field " + ck
				case k == "webhook" && !isOneOf(ck, "callback_url", "callback_method", "fallback_url", "fallback_method", "status_callback_url", "status_callback_method"):
					return "unknown webhook field " + ck
				default:
					dst[ck] = cv
				}
			}
		case "profile":
			m, _ := v.(map[string]any)
			dst := o["profile"].(map[string]any)
			for pk, pv := range m {
				if pk != "name" {
					dst[pk] = pv
					continue
				}
				active, _ := dst["name"].(string)
				if active == "" {
					dst["name"] = pv // first registration
				} else if pv != active {
					o["pending_display_name"] = pv
					o["pending_display_name_status"] = "PENDING_REVIEW"
					o["pending_display_name_status_date"] = cvStamp
				}
			}
		default:
			return "unknown field " + k
		}
	}
	return ""
}

func isOneOf(s string, l ...string) bool {
	for _, x := range l {
		if x == s {
			return true
		}
	}
	return false
}

func (f *fakeTwilio) listChannelsSenders(w http.ResponseWriter, r *http.Request) {
	q := r.URL.Query()
	ch := q.Get("Channel")
	if ch != "whatsapp" && ch != "rcs" {
		ferr(w, 400, "Channel is required (whatsapp or rcs)")
		return
	}
	var sids []string
	for k, o := range f.objs {
		if !strings.HasPrefix(k, "cs/"+testAccount+"/") {
			continue
		}
		if strings.HasPrefix(fmt.Sprint(o["sender_id"]), "whatsapp:") == (ch == "whatsapp") {
			sids = append(sids, fmt.Sprint(o["sid"]))
		}
	}
	sort.Strings(sids)
	page := 0
	fmt.Sscanf(q.Get("PageToken"), "%d", &page)
	items := []any{}
	for i := page * channelsSenderPageSize; i < len(sids) && i < (page+1)*channelsSenderPageSize; i++ {
		items = append(items, f.objs[key("cs", testAccount, sids[i])])
	}
	meta := map[string]any{"page": page, "page_size": channelsSenderPageSize, "key": "senders", "next_page_url": nil}
	if (page+1)*channelsSenderPageSize < len(sids) {
		meta["next_page_url"] = fmt.Sprintf("https://messaging.twilio.com/v2/Channels/Senders?Channel=%s&PageToken=%d", ch, page+1)
	}
	fwrite(w, 200, map[string]any{"senders": items, "meta": meta})
}
