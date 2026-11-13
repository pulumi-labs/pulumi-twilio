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
	"strings"
)

// The Content v1 routes, served for the Content tests:
//
//	/content/v1/Content[/{sid}[/ApprovalRequests]]
//
// The shapes follow twilio-oai spec/json/twilio_content_v1.json. They take JSON bodies. Objects are
// stored as kind "ct" keyed by SID; the approval state of a template lives in its "approval" entry.

// contentMangleCreate lets a test corrupt the answer to a create.
var contentMangleCreate func(map[string]any)

func (f *fakeTwilio) handleContent(w http.ResponseWriter, r *http.Request, body []byte, rest string) {
	parts := strings.Split(rest, "/")
	if parts[0] != "Content" || len(parts) > 3 || (len(parts) == 3 && parts[2] != "ApprovalRequests") {
		ferr(w, 404, "no route "+r.URL.Path)
		return
	}
	read := func(o map[string]any, needTypes bool) string {
		var req map[string]any
		if err := json.Unmarshal(body, &req); err != nil {
			return "invalid JSON body"
		}
		types, _ := req["types"].(map[string]any)
		if needTypes && len(types) == 0 {
			return "types is required"
		}
		if _, ok := req["language"].(string); !ok && o["language"] == nil {
			return "language is required"
		}
		for k, v := range req {
			switch k {
			case "friendly_name", "variables", "language", "types":
				o[k] = v
			default:
				return "unknown field " + k
			}
		}
		return ""
	}
	if len(parts) == 1 {
		if r.Method != "POST" {
			ferr(w, 405, "method not allowed")
			return
		}
		id := f.sid("HX")
		o := map[string]any{"sid": id, "account_sid": testAccount, "friendly_name": nil, "language": nil, "variables": map[string]any{},
			"types": map[string]any{}, "date_created": cvStamp, "date_updated": cvStamp, "url": "https://content.twilio.com/v1/Content/" + id,
			"links": map[string]any{"approval_create": "https://content.twilio.com/v1/Content/" + id + "/ApprovalRequests/whatsapp"}}
		if msg := read(o, true); msg != "" {
			ferr(w, 400, msg)
			return
		}
		f.put("ct", testAccount, id, o)
		if contentMangleCreate != nil {
			out := clone(o)
			contentMangleCreate(out)
			fwrite(w, 201, out)
			return
		}
		fwrite(w, 201, o)
		return
	}
	o, ok := f.objs[key("ct", testAccount, parts[1])]
	if !ok || !strings.HasPrefix(parts[1], "HX") {
		ferr(w, 404, "The requested resource "+r.URL.Path+" was not found")
		return
	}
	if len(parts) == 3 {
		if r.Method != "GET" {
			ferr(w, 405, "method not allowed")
			return
		}
		fwrite(w, 200, map[string]any{"sid": parts[1], "account_sid": testAccount, "whatsapp": o["approval"],
			"url": "https://content.twilio.com/v1/Content/" + parts[1] + "/ApprovalRequests"})
		return
	}
	switch r.Method {
	case "GET":
		fwrite(w, 200, o)
	case "PUT":
		if msg := read(o, true); msg != "" {
			ferr(w, 400, msg)
			return
		}
		o["date_updated"] = cvUpdatedStamp
		fwrite(w, 200, o)
	case "DELETE":
		f.del("ct", testAccount, parts[1])
		w.WriteHeader(204)
	default:
		ferr(w, 405, "method not allowed")
	}
}
