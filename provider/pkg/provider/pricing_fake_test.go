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
	"strconv"
	"strings"
)

// The Pricing routes (pricing.twilio.com), read-only like the real API:
//
//	/pricing/v1/{Messaging,PhoneNumbers}/Countries[/{IsoCountry}]
//	/pricing/v2/{Voice,Trunking}/Countries[/{IsoCountry}]
//	/pricing/v2/{Voice,Trunking}/Numbers/{DestinationNumber}[?OriginationNumber=]
//
// Lists answer {"countries": [...], "meta": {...}} and page through meta.next_page_url. Shapes
// follow twilio-oai spec/json/twilio_pricing_v1.json and twilio_pricing_v2.json (version 1.0.0).

// seedPricing stores the object served at /pricing/<path> (for example v2/Voice/Countries/DE).
func (f *fakeTwilio) seedPricing(path string, o map[string]any) {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.put("pricing", testAccount, path, o)
}

// seedPricingCountry stores a country instance under a family (v1/Messaging, v2/Voice ...).
func (f *fakeTwilio) seedPricingCountry(family, iso, country string, extra map[string]any) {
	o := map[string]any{"iso_country": iso, "country": country, "price_unit": "usd",
		"url": "https://pricing.twilio.com/" + family + "/Countries/" + iso}
	for k, v := range extra {
		o[k] = v
	}
	f.seedPricing(family+"/Countries/"+iso, o)
}

func (f *fakeTwilio) handlePricing(w http.ResponseWriter, r *http.Request) {
	path := strings.TrimPrefix(r.URL.Path, "/pricing/")
	if r.Method != http.MethodGet {
		ferr(w, 405, "read-only")
		return
	}
	if strings.HasSuffix(path, "/Countries") {
		prefix := key("pricing", testAccount, path+"/")
		var items []any
		for _, k := range f.order {
			o, ok := f.objs[k]
			if !ok || !strings.HasPrefix(k, prefix) {
				continue
			}
			items = append(items, map[string]any{"iso_country": o["iso_country"], "country": o["country"], "url": o["url"]})
		}
		if items == nil {
			items = []any{}
		}
		q := r.URL.Query()
		size, _ := strconv.Atoi(q.Get("PageSize"))
		if size < 1 {
			size = 50
		}
		if f.pricingMaxPage > 0 && size > f.pricingMaxPage {
			size = f.pricingMaxPage
		}
		page, _ := strconv.Atoi(q.Get("Page"))
		lo := min(page*size, len(items))
		hi := min(lo+size, len(items))
		meta := map[string]any{"page": page, "page_size": size, "key": "countries", "next_page_url": nil}
		if hi < len(items) {
			nq := url.Values{}
			for k, v := range q {
				nq[k] = v
			}
			nq.Set("Page", strconv.Itoa(page+1))
			meta["next_page_url"] = f.server.URL + r.URL.Path + "?" + nq.Encode()
		}
		fwrite(w, 200, map[string]any{"countries": items[lo:hi], "meta": meta})
		return
	}
	o, ok := f.objs[key("pricing", testAccount, path)]
	if !ok {
		ferr(w, 404, "The requested resource /"+path+" was not found")
		return
	}
	o = clone(o)
	if strings.Contains(path, "/Numbers/") {
		if orig := r.URL.Query().Get("OriginationNumber"); orig != "" {
			o["origination_number"] = orig
		}
	}
	fwrite(w, 200, o)
}
