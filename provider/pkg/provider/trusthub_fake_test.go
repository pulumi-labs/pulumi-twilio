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

// The TrustHub routes of the fake: /trusthub/v1/{EndUsers|SupportingDocuments}[/{sid}] (create, fetch,
// update, delete) and the read-only EndUserTypes, SupportingDocumentTypes and Policies. Shapes follow
// twilio-oai spec/json/twilio_trusthub_v1.json (1.0.0). Objects are stored as kind "th:<collection>".
// Lists are served two items per page so paging is exercised. Types and policies are seeded with addTrusthubItem.

const thFakePageSize = 2

func (f *fakeTwilio) addTrusthubItem(coll string, o map[string]any) {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.put("th:"+coll, testAccount, o["sid"].(string), o)
}

func (f *fakeTwilio) handleTrusthub(w http.ResponseWriter, r *http.Request, form url.Values, parts []string) {
	coll, sid := parts[0], ""
	if len(parts) > 1 {
		sid = parts[1]
	}
	itemsKey := map[string]string{"EndUsers": "results", "SupportingDocuments": "results", "Policies": "results",
		"EndUserTypes": "end_user_types", "SupportingDocumentTypes": "supporting_document_types"}[coll]
	if itemsKey == "" {
		ferr(w, 404, "The requested resource "+r.URL.Path+" was not found")
		return
	}
	writable := coll == "EndUsers" || coll == "SupportingDocuments"
	kind := "th:" + coll
	if sid == "" {
		switch {
		case r.Method == "GET":
			var items []map[string]any
			for _, k := range f.order {
				if o, ok := f.objs[k]; ok && strings.HasPrefix(k, kind+"/") {
					items = append(items, o)
				}
			}
			metaPage(w, r, itemsKey, items, thFakePageSize)
		case r.Method == "POST" && writable:
			if form.Get("FriendlyName") == "" || form.Get("Type") == "" {
				ferr(w, 400, "FriendlyName and Type are required")
				return
			}
			prefix := "IT"
			o := map[string]any{"account_sid": testAccount, "friendly_name": form.Get("FriendlyName"), "type": form.Get("Type"),
				"attributes": map[string]any{}, "date_created": cvStamp, "date_updated": cvStamp}
			if coll == "SupportingDocuments" {
				prefix = "RD"
				o["mime_type"], o["status"] = nil, "DRAFT"
			}
			if msg := thApplyAttributes(o, form); msg != "" {
				ferr(w, 400, msg)
				return
			}
			o["sid"] = f.sid(prefix)
			o["url"] = "https://trusthub.twilio.com/v1/" + coll + "/" + o["sid"].(string)
			f.put(kind, testAccount, o["sid"].(string), o)
			fwrite(w, 201, o)
		default:
			ferr(w, 405, "method not allowed")
		}
		return
	}
	o, ok := f.objs[key(kind, testAccount, sid)]
	if !ok {
		ferr(w, 404, "The requested resource "+r.URL.Path+" was not found")
		return
	}
	switch {
	case r.Method == "GET":
		fwrite(w, 200, o)
	case r.Method == "POST" && writable:
		if v, ok := form["FriendlyName"]; ok {
			o["friendly_name"] = v[0]
		}
		if msg := thApplyAttributes(o, form); msg != "" {
			ferr(w, 400, msg)
			return
		}
		o["date_updated"] = "2026-11-14T01:00:00Z"
		fwrite(w, 200, o)
	case r.Method == "DELETE" && writable:
		f.del(kind, testAccount, sid)
		w.WriteHeader(204)
	default:
		ferr(w, 405, "method not allowed")
	}
}

// thApplyAttributes stores the posted Attributes JSON document as an object, as Twilio returns it.
func thApplyAttributes(o map[string]any, form url.Values) string {
	v, ok := form["Attributes"]
	if !ok {
		return ""
	}
	var m map[string]any
	if json.Unmarshal([]byte(v[0]), &m) != nil || m == nil {
		return "Attributes must be a JSON object"
	}
	o["attributes"] = m
	return ""
}
