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
	"fmt"
	"net/http"
	"net/url"
	"strconv"
	"strings"
)

// Read-only data served for the lookup tests: AvailablePhoneNumbers, Usage/Records,
// AuthorizedConnectApps on the 2010 API and the Events Types/Schemas catalogue. The
// shapes follow twilio-oai spec/json (twilio_api_v2010.json, twilio_events_v1.json).

// fakeAvailable is the pool of numbers the fake can sell, per country and kind.
func fakeAvailable(country, kind string) []map[string]any {
	var out []map[string]any
	for i := 0; i < 5; i++ {
		prefix := map[string]string{"Local": "+1415555010", "TollFree": "+1800555010", "Mobile": "+4479005550"}[kind]
		out = append(out, map[string]any{
			"phone_number": prefix + strconv.Itoa(i), "friendly_name": "num " + strconv.Itoa(i), "iso_country": country,
			"locality": "San Francisco", "region": "CA", "postal_code": "94105", "rate_center": "SNFC", "lata": "722",
			"latitude": "37.78", "longitude": -122.4, "address_requirements": "none", "beta": i == 4,
			"capabilities": map[string]any{"voice": true, "SMS": i%2 == 0, "MMS": false, "fax": false},
		})
	}
	return out
}

// fakeUsagePages builds usage records; a test sets usagePageCap to force paging.
var usagePageCap = 100

func fakeUsage(acct, period string, total int) []map[string]any {
	var out []map[string]any
	for i := 0; i < total; i++ {
		out = append(out, map[string]any{
			"account_sid": acct, "category": "cat" + strconv.Itoa(i), "description": "usage " + period, "count": "3", "count_unit": "calls",
			"usage": "7", "usage_unit": "minutes", "price": "0.5000", "price_unit": "usd", "start_date": "2026-11-01", "end_date": "2026-11-06",
			"as_of": "Fri, 06 Nov 2026 00:00:00 +0000",
		})
	}
	return out
}

var fakeUsageTotal = 3

// pageJSON serves one page of items with next_page_uri (2010 style).
func pageJSON(w http.ResponseWriter, r *http.Request, key string, items []map[string]any, cap int) {
	q := r.URL.Query()
	size := cap
	if v, err := strconv.Atoi(q.Get("PageSize")); err == nil && v < size {
		size = v
	}
	page, _ := strconv.Atoi(q.Get("Page"))
	lo := min(page*size, len(items))
	hi := min(lo+size, len(items))
	resp := map[string]any{key: items[lo:hi], "page": page, "page_size": size, "next_page_uri": nil}
	if hi < len(items) {
		nq := url.Values{"PageSize": {strconv.Itoa(size)}, "Page": {strconv.Itoa(page + 1)}}
		for k, v := range q {
			if k != "Page" && k != "PageSize" {
				nq[k] = v
			}
		}
		resp["next_page_uri"] = strings.TrimPrefix(r.URL.Path, "/api") + "?" + nq.Encode()
	}
	fwrite(w, 200, resp)
}

func (f *fakeTwilio) handleDataAPI(w http.ResponseWriter, r *http.Request, parts []string) bool {
	if r.Method != "GET" {
		return false
	}
	acct := parts[1]
	switch trimJSON(parts[2]) {
	case "AvailablePhoneNumbers":
		if len(parts) != 5 {
			return false
		}
		kind := trimJSON(parts[4])
		if kind != "Local" && kind != "TollFree" && kind != "Mobile" {
			ferr(w, 404, "The requested resource "+r.URL.Path+" was not found")
			return true
		}
		items := fakeAvailable(parts[3], kind)
		q := r.URL.Query()
		if c := q.Get("Contains"); c != "" {
			var kept []map[string]any
			for _, it := range items {
				if strings.HasSuffix(it["phone_number"].(string), c) {
					kept = append(kept, it)
				}
			}
			items = kept
		}
		if q.Get("SmsEnabled") == "true" {
			var kept []map[string]any
			for _, it := range items {
				if it["capabilities"].(map[string]any)["SMS"] == true {
					kept = append(kept, it)
				}
			}
			items = kept
		}
		pageJSON(w, r, "available_phone_numbers", items, 100)
		return true
	case "Usage":
		if len(parts) < 4 || trimJSON(parts[3]) != "Records" {
			return false
		}
		period := "Default"
		if len(parts) == 5 {
			period = trimJSON(parts[4])
		}
		if len(parts) > 5 || (period != "Default" && !strings.Contains("AllTime Daily Monthly Yearly Today Yesterday ThisMonth LastMonth", period)) {
			return false
		}
		items := fakeUsage(acct, period, fakeUsageTotal)
		if c := r.URL.Query().Get("Category"); c != "" {
			var kept []map[string]any
			for _, it := range items {
				if it["category"] == c {
					kept = append(kept, it)
				}
			}
			items = kept
		}
		pageJSON(w, r, "usage_records", items, usagePageCap)
		return true
	case "AuthorizedConnectApps":
		apps := []map[string]any{}
		for i := 0; i < 2; i++ {
			apps = append(apps, map[string]any{
				"account_sid": acct, "connect_app_sid": fmt.Sprintf("CN%032d", i), "connect_app_friendly_name": "app" + strconv.Itoa(i),
				"connect_app_company_name": "Acme", "connect_app_description": "d", "connect_app_homepage_url": "https://example.com",
				"permissions": []any{"get-all"},
			})
		}
		if len(parts) == 4 {
			for _, a := range apps {
				if a["connect_app_sid"] == trimJSON(parts[3]) {
					fwrite(w, 200, a)
					return true
				}
			}
			ferr(w, 404, "The requested resource "+r.URL.Path+" was not found")
			return true
		}
		pageJSON(w, r, "authorized_connect_apps", apps, 100)
		return true
	}
	return false
}

// eventCatalogPageCap forces the Events catalogue to page in a test.
var eventCatalogPageCap = 100

func (f *fakeTwilio) handleEventCatalog(w http.ResponseWriter, r *http.Request, parts []string, notFound func()) {
	if r.Method != "GET" {
		ferr(w, 405, "method not allowed")
		return
	}
	types := []map[string]any{
		{"type": "com.twilio.messaging.message.sent", "schema_id": "Messaging.MessageStatus", "description": "sent", "status": "available", "documentation_url": "https://example.com/sent", "date_created": eventsStamp, "date_updated": eventsStamp},
		{"type": "com.twilio.messaging.message.delivered", "schema_id": "Messaging.MessageStatus", "description": "delivered", "status": "available", "documentation_url": "https://example.com/delivered", "date_created": eventsStamp, "date_updated": eventsStamp},
		{"type": "com.twilio.voice.insights.call-summary.complete", "schema_id": "Voice.CallSummary", "description": "summary", "status": "deprecated", "documentation_url": "", "date_created": eventsStamp, "date_updated": eventsStamp},
	}
	switch {
	case parts[0] == "Types" && len(parts) == 1:
		var kept []map[string]any
		for _, t := range types {
			if s := r.URL.Query().Get("SchemaId"); s == "" || t["schema_id"] == s {
				kept = append(kept, t)
			}
		}
		metaPage(w, r, "types", kept, eventCatalogPageCap)
	case parts[0] == "Types" && len(parts) == 2:
		for _, t := range types {
			if t["type"] == parts[1] {
				fwrite(w, 200, t)
				return
			}
		}
		notFound()
	case parts[0] == "Schemas" && len(parts) >= 2 && parts[1] == "Messaging.MessageStatus":
		versions := []map[string]any{
			{"id": parts[1], "schema_version": 1, "date_created": eventsStamp, "raw": "https://events-schemas.twilio.com/Messaging.MessageStatus/1"},
			{"id": parts[1], "schema_version": 2, "date_created": eventsStamp, "raw": "https://events-schemas.twilio.com/Messaging.MessageStatus/2"},
		}
		switch len(parts) {
		case 2:
			fwrite(w, 200, map[string]any{"id": parts[1], "latest_version": 2, "latest_version_date_created": eventsStamp})
		case 3:
			if parts[2] != "Versions" {
				notFound()
				return
			}
			metaPage(w, r, "schema_versions", versions, eventCatalogPageCap)
		default:
			notFound()
		}
	default:
		notFound()
	}
}

// metaPage serves one page of items with meta.next_page_url (newer API style).
func metaPage(w http.ResponseWriter, r *http.Request, key string, items []map[string]any, cap int) {
	q := r.URL.Query()
	size := cap
	if v, err := strconv.Atoi(q.Get("PageSize")); err == nil && v < size {
		size = min(size, v)
	}
	size = min(size, cap)
	page, _ := strconv.Atoi(q.Get("Page"))
	lo := min(page*size, len(items))
	hi := min(lo+size, len(items))
	meta := map[string]any{"key": key, "page": page, "page_size": size, "next_page_url": nil}
	if hi < len(items) {
		nq := url.Values{"PageSize": {strconv.Itoa(size)}, "Page": {strconv.Itoa(page + 1)}}
		for k, v := range q {
			if k != "Page" && k != "PageSize" {
				nq[k] = v
			}
		}
		meta["next_page_url"] = "https://events.twilio.com" + r.URL.Path + "?" + nq.Encode()
	}
	fwrite(w, 200, map[string]any{key: items[lo:hi], "meta": meta})
}
