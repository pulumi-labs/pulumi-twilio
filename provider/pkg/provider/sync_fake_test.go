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
)

// The Sync v1 routes, served for the Sync tests:
//
//	/sync/v1/Services[/{sid}]
//
// The shapes follow twilio-oai spec/json/twilio_sync_v1.json. Services are stored as kind "sync".

var syncServiceFields = []fakeField{
	{"FriendlyName", "friendly_name", "s"}, {"WebhookUrl", "webhook_url", "u"},
	{"ReachabilityWebhooksEnabled", "reachability_webhooks_enabled", "b"}, {"AclEnabled", "acl_enabled", "b"},
	{"ReachabilityDebouncingEnabled", "reachability_debouncing_enabled", "b"},
	{"ReachabilityDebouncingWindow", "reachability_debouncing_window", "i"},
	{"WebhooksFromRestEnabled", "webhooks_from_rest_enabled", "b"},
}

func (f *fakeTwilio) handleSync(w http.ResponseWriter, r *http.Request, form url.Values, sid string) {
	if sid == "" {
		if r.Method != "POST" {
			ferr(w, 405, "method not allowed")
			return
		}
		id := f.sid("IS")
		o := map[string]any{"sid": id, "account_sid": testAccount, "date_created": cvStamp, "date_updated": cvStamp,
			"friendly_name": nil, "webhook_url": nil, "reachability_webhooks_enabled": false, "acl_enabled": false,
			"reachability_debouncing_enabled": false, "reachability_debouncing_window": 5000, "webhooks_from_rest_enabled": false}
		if msg := applyForm(o, form, syncServiceFields); msg != "" {
			ferr(w, 400, msg)
			return
		}
		f.put("sync", testAccount, id, o)
		fwrite(w, 201, o)
		return
	}
	o, ok := f.objs[key("sync", testAccount, sid)]
	if !ok {
		ferr(w, 404, "The requested resource "+r.URL.Path+" was not found")
		return
	}
	switch r.Method {
	case "GET":
		fwrite(w, 200, o)
	case "POST":
		if msg := applyForm(o, form, syncServiceFields); msg != "" {
			ferr(w, 400, msg)
			return
		}
		o["date_updated"] = cvUpdatedStamp
		fwrite(w, 200, o)
	case "DELETE":
		f.del("sync", testAccount, sid)
		w.WriteHeader(204)
	default:
		ferr(w, 405, "method not allowed")
	}
}
