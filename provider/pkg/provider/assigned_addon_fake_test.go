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

// The add-ons assigned to an incoming phone number (twilio_api_v2010.json):
//
//	/Accounts/{acct}/IncomingPhoneNumbers/{pn}/AssignedAddOns[/{sid}]
//	/Accounts/{acct}/IncomingPhoneNumbers/{pn}/AssignedAddOns/{sid}/Extensions[/{xid}]   (read-only)
//
// Create takes InstalledAddOnSid only; the response does not echo it (the spec's resource has no such field),
// so the fake keeps it under "_installed" for assertions. There is no update call.

// assignedAddOnPageCap is how many extensions one page of the fake holds; a test lowers it to force paging.
var assignedAddOnPageCap = 100

func assignedAddOnOut(o map[string]any) map[string]any {
	c := clone(o)
	delete(c, "_installed")
	delete(c, "_extensions")
	return c
}

// handleAssignedAddOn serves the routes above and reports whether parts named one of them.
func (f *fakeTwilio) handleAssignedAddOn(w http.ResponseWriter, r *http.Request, form url.Values, parts []string) bool {
	if len(parts) < 5 || trimJSON(parts[2]) != "IncomingPhoneNumbers" || trimJSON(parts[4]) != "AssignedAddOns" {
		return false
	}
	acct, number := parts[1], parts[3]
	notFound := func() { ferr(w, 404, "The requested resource "+r.URL.Path+" was not found") }
	if !f.accountKnown(acct) {
		notFound()
		return true
	}
	if _, ok := f.objs[key("number", acct, number)]; !ok {
		notFound()
		return true
	}
	switch len(parts) {
	case 5: // collection
		if r.Method != "POST" {
			ferr(w, 405, "method not allowed")
			return true
		}
		inst := form.Get("InstalledAddOnSid")
		if inst == "" {
			ferr(w, 400, "Missing required parameter InstalledAddOnSid")
			return true
		}
		if !installedAddOnSidRE.MatchString(inst) {
			ferr(w, 400, "InstalledAddOnSid is not a valid installed add-on SID")
			return true
		}
		for k, o := range f.objs {
			if len(k) > 9 && k[:9] == "addonasg/" && o["resource_sid"] == number && o["_installed"] == inst {
				ferr(w, 400, "The add-on is already assigned to this phone number")
				return true
			}
		}
		sid := f.sid("XE")
		o := map[string]any{"sid": sid, "account_sid": acct, "resource_sid": number, "friendly_name": "Spam Filter", "description": "Flags spam callers",
			"configuration": map[string]any{"threshold": "high"}, "unique_name": "spam_filter", "uri": "/x", "_installed": inst,
			"date_created": "Mon, 02 Nov 2026 00:00:00 +0000", "date_updated": "Mon, 02 Nov 2026 00:00:00 +0000"}
		f.put("addonasg", acct, sid, o)
		fwrite(w, 201, assignedAddOnOut(o))
	case 6, 7: // instance, or its extension list
		sid := trimJSON(parts[5])
		o, live := f.objs[key("addonasg", acct, sid)]
		if !live || o["resource_sid"] != number {
			notFound()
			return true
		}
		if len(parts) == 7 {
			if trimJSON(parts[6]) != "Extensions" || r.Method != "GET" {
				notFound()
				return true
			}
			exts, _ := o["_extensions"].([]map[string]any)
			pageJSON(w, r, "extensions", exts, assignedAddOnPageCap)
			return true
		}
		switch r.Method {
		case "GET":
			fwrite(w, 200, assignedAddOnOut(o))
		case "DELETE":
			f.del("addonasg", acct, sid)
			w.WriteHeader(204)
		default:
			ferr(w, 405, "method not allowed")
		}
	default:
		notFound()
	}
	return true
}
