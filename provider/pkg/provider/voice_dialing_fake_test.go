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
)

// The Voice v1 dialing permissions routes, read-only like the real API (changes go through the
// bulk update, which is not modelled):
//
//	/voice/v1/DialingPermissions/Countries
//	/voice/v1/DialingPermissions/Countries/{IsoCode}
//	/voice/v1/DialingPermissions/Countries/{IsoCode}/HighRiskSpecialPrefixes
//
// Lists answer {"content": [...], "meta": {...}} and page through meta.next_page_url. Shapes
// follow twilio-oai spec/json/twilio_voice_v1.json (spec version 1.0.0).

// seedDialingCountry stores one country; prefixes are its high-risk special prefixes.
func (f *fakeTwilio) seedDialingCountry(iso, name, continent string, codes []any, low, special, toll bool, prefixes ...string) {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.put("dialingcountry", testAccount, iso, map[string]any{
		"iso_code": iso, "name": name, "continent": continent, "country_codes": codes,
		"low_risk_numbers_enabled": low, "high_risk_special_numbers_enabled": special, "high_risk_tollfraud_numbers_enabled": toll,
		"url": "https://voice.twilio.com/v1/DialingPermissions/Countries/" + iso,
	})
	ps := []any{}
	for _, p := range prefixes {
		ps = append(ps, map[string]any{"prefix": p})
	}
	f.put("dialingprefixes", testAccount, iso, map[string]any{"items": ps})
}

func (f *fakeTwilio) dialingPage(w http.ResponseWriter, r *http.Request, items []any) {
	q := r.URL.Query()
	size, _ := strconv.Atoi(q.Get("PageSize"))
	if size < 1 {
		size = 50
	}
	if f.dialingMaxPage > 0 && size > f.dialingMaxPage {
		size = f.dialingMaxPage
	}
	page, _ := strconv.Atoi(q.Get("Page"))
	lo := page * size
	if lo > len(items) {
		lo = len(items)
	}
	hi := lo + size
	if hi > len(items) {
		hi = len(items)
	}
	meta := map[string]any{"page": page, "page_size": size, "key": "content", "next_page_url": nil}
	if hi < len(items) {
		nq := url.Values{}
		for k, v := range q {
			nq[k] = v
		}
		nq.Set("Page", strconv.Itoa(page+1))
		meta["next_page_url"] = f.server.URL + r.URL.Path + "?" + nq.Encode()
	}
	fwrite(w, 200, map[string]any{"content": items[lo:hi], "meta": meta})
}

func (f *fakeTwilio) handleDialingPermissions(w http.ResponseWriter, r *http.Request, parts []string, notFound func()) {
	if r.Method != http.MethodGet || len(parts) == 0 || parts[0] != "Countries" || len(parts) > 3 {
		notFound()
		return
	}
	switch len(parts) {
	case 1:
		q := r.URL.Query()
		var items []any
		for _, k := range f.order {
			o, ok := f.objs[k]
			if !ok || o["iso_code"] == nil {
				continue
			}
			if v := q.Get("IsoCode"); v != "" && o["iso_code"] != v {
				continue
			}
			if v := q.Get("Continent"); v != "" && o["continent"] != v {
				continue
			}
			if v := q.Get("CountryCode"); v != "" {
				match := false
				for _, c := range o["country_codes"].([]any) {
					if fmt.Sprint(c) == v {
						match = true
					}
				}
				if !match {
					continue
				}
			}
			skip := false
			for param, field := range map[string]string{"LowRiskNumbersEnabled": "low_risk_numbers_enabled",
				"HighRiskSpecialNumbersEnabled": "high_risk_special_numbers_enabled", "HighRiskTollfraudNumbersEnabled": "high_risk_tollfraud_numbers_enabled"} {
				if v := q.Get(param); v != "" && strconv.FormatBool(o[field] == true) != v {
					skip = true
				}
			}
			if !skip {
				items = append(items, o)
			}
		}
		if items == nil {
			items = []any{}
		}
		f.dialingPage(w, r, items)
	case 2:
		o, ok := f.objs[key("dialingcountry", testAccount, parts[1])]
		if !ok {
			notFound()
			return
		}
		fwrite(w, 200, o)
	case 3:
		o, ok := f.objs[key("dialingprefixes", testAccount, parts[1])]
		if !ok || parts[2] != "HighRiskSpecialPrefixes" {
			notFound()
			return
		}
		f.dialingPage(w, r, o["items"].([]any))
	}
}
