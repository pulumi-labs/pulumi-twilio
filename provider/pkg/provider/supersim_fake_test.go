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

// The Super SIM and Wireless v1 routes, served for the Super SIM and Wireless tests:
//
//	/supersim/v1/Fleets[/{sid}]
//	/supersim/v1/NetworkAccessProfiles[/{sid}[/Networks[/{networkSid}]]]
//	/supersim/v1/Networks[/{sid}]                    (read-only catalog, seeded with addSuperSimNetwork)
//	/wireless/v1/RatePlans[/{sid}]
//
// The shapes follow twilio-oai spec/json/twilio_supersim_v1.json and twilio_wireless_v1.json.
// Objects are stored as kind "ss" keyed <collection>/<sid>; the networks of a profile are stored as
// NetworkAccessProfiles/<profile>/Networks/<network>. Neither a Fleet nor a profile has a DELETE.

const (
	ssNetA = "HW00000000000000000000000000000001"
	ssNetB = "HW00000000000000000000000000000002"
	ssNetC = "HW00000000000000000000000000000003"
)

// addSuperSimNetwork seeds one network of the catalog.
func (f *fakeTwilio) addSuperSimNetwork(sid, name, country string, ids ...[2]string) {
	f.mu.Lock()
	defer f.mu.Unlock()
	list := []any{}
	for _, id := range ids {
		list = append(list, map[string]any{"mcc": id[0], "mnc": id[1]})
	}
	f.put("ss", testAccount, "Networks/"+sid, map[string]any{"sid": sid, "friendly_name": name, "iso_country": country,
		"identifiers": list, "url": "https://supersim.twilio.com/v1/Networks/" + sid})
}

// ssByKey finds a stored object by SID or unique name.
func (f *fakeTwilio) ssByKey(coll, k string) (string, map[string]any) {
	if o, ok := f.objs[key("ss", testAccount, coll+"/"+k)]; ok {
		return k, o
	}
	for _, fk := range f.order {
		o, ok := f.objs[fk]
		if ok && strings.HasPrefix(fk, "ss/"+testAccount+"/"+coll+"/") && o["unique_name"] == k {
			return o["sid"].(string), o
		}
	}
	return "", nil
}

// ssUniqueTaken reports whether another object of the collection has the unique name.
func (f *fakeTwilio) ssUniqueTaken(coll, name, self string) bool {
	if name == "" {
		return false
	}
	for _, fk := range f.order {
		o, ok := f.objs[fk]
		if ok && strings.HasPrefix(fk, "ss/"+testAccount+"/"+coll+"/") && strings.Count(fk, "/") == 3 && o["unique_name"] == name && o["sid"] != self {
			return true
		}
	}
	return false
}

// ssList returns the stored objects directly under coll (no deeper keys), in creation order.
func (f *fakeTwilio) ssList(coll string, keep func(map[string]any) bool) []map[string]any {
	out := []map[string]any{}
	prefix := "ss/" + testAccount + "/" + coll + "/"
	seen := map[string]bool{} // put appends a key again when it is re-added after a delete
	for _, fk := range f.order {
		o, ok := f.objs[fk]
		if !ok || seen[fk] || !strings.HasPrefix(fk, prefix) || strings.Contains(strings.TrimPrefix(fk, prefix), "/") {
			continue
		}
		seen[fk] = true
		if keep == nil || keep(o) {
			out = append(out, o)
		}
	}
	return out
}

var ssFleetFields = []fakeField{
	{"UniqueName", "unique_name", "s"}, {"IpCommandsUrl", "ip_commands_url", "u"}, {"IpCommandsMethod", "ip_commands_method", "m"},
	{"SmsCommandsUrl", "sms_commands_url", "u"}, {"SmsCommandsMethod", "sms_commands_method", "m"}, {"DataLimit", "data_limit", "i"},
}

var ssFleetCreateOnly = []fakeField{{"DataEnabled", "data_enabled", "b"}, {"SmsCommandsEnabled", "sms_commands_enabled", "b"}}

func (f *fakeTwilio) handleSuperSim(w http.ResponseWriter, r *http.Request, form url.Values, rest string) {
	parts := strings.Split(rest, "/")
	notFound := func() { ferr(w, 404, "The requested resource "+r.URL.Path+" was not found") }
	switch parts[0] {
	case "Fleets":
		if len(parts) > 2 {
			notFound()
			return
		}
		f.ssFleets(w, r, form, parts)
	case "NetworkAccessProfiles":
		f.ssProfiles(w, r, form, parts)
	case "Networks":
		if r.Method != "GET" || len(parts) > 2 {
			ferr(w, 405, "method not allowed")
			return
		}
		if len(parts) == 2 {
			o, ok := f.objs[key("ss", testAccount, "Networks/"+parts[1])]
			if !ok {
				notFound()
				return
			}
			fwrite(w, 200, o)
			return
		}
		q := r.URL.Query()
		items := f.ssList("Networks", func(o map[string]any) bool {
			if c := q.Get("IsoCountry"); c != "" && o["iso_country"] != c {
				return false
			}
			for _, k := range [][2]string{{"Mcc", "mcc"}, {"Mnc", "mnc"}} {
				v := q.Get(k[0])
				if v == "" {
					continue
				}
				found := false
				for _, id := range o["identifiers"].([]any) {
					if id.(map[string]any)[k[1]] == v {
						found = true
					}
				}
				if !found {
					return false
				}
			}
			return true
		})
		metaPage(w, r, "networks", items, 2)
	default:
		notFound()
	}
}

func (f *fakeTwilio) ssFleets(w http.ResponseWriter, r *http.Request, form url.Values, parts []string) {
	apply := func(o map[string]any, create bool) string {
		if msg := applyForm(o, form, ssFleetFields); msg != "" {
			return msg
		}
		if create {
			if msg := applyForm(o, form, ssFleetCreateOnly); msg != "" {
				return msg
			}
		} else {
			for _, fl := range ssFleetCreateOnly {
				if _, ok := form[fl.param]; ok {
					return "Unknown parameter " + fl.param + " on update"
				}
			}
		}
		if v, ok := form["NetworkAccessProfile"]; ok {
			sid, _ := f.ssByKey("NetworkAccessProfiles", v[0])
			if sid == "" {
				return "Network Access Profile " + v[0] + " was not found"
			}
			o["network_access_profile_sid"] = sid
		}
		if n, ok := o["data_limit"].(float64); ok && (n < 1 || n > 2000000) {
			return "DataLimit must be between 1 and 2000000"
		}
		return ""
	}
	if len(parts) == 1 {
		switch r.Method {
		case "POST":
			if form.Get("NetworkAccessProfile") == "" {
				ferr(w, 400, "Missing required parameter NetworkAccessProfile")
				return
			}
			id := f.sid("HF")
			o := map[string]any{"sid": id, "account_sid": testAccount, "unique_name": nil, "date_created": cvStamp, "date_updated": cvStamp,
				"url": "https://supersim.twilio.com/v1/Fleets/" + id, "data_enabled": true, "data_limit": float64(1000), "data_metering": "payg",
				"sms_commands_enabled": true, "sms_commands_url": nil, "sms_commands_method": "POST", "network_access_profile_sid": nil,
				"ip_commands_url": nil, "ip_commands_method": "POST"}
			if msg := apply(o, true); msg != "" {
				ferr(w, 400, msg)
				return
			}
			if name, _ := o["unique_name"].(string); f.ssUniqueTaken("Fleets", name, id) {
				ferr(w, 409, "Fleet unique name "+name+" is already in use")
				return
			}
			f.put("ss", testAccount, "Fleets/"+id, o)
			fwrite(w, 201, o)
		case "GET":
			metaPage(w, r, "fleets", f.ssList("Fleets", nil), 50)
		default:
			ferr(w, 405, "method not allowed")
		}
		return
	}
	sid, o := f.ssByKey("Fleets", parts[1])
	if o == nil {
		ferr(w, 404, "The requested resource "+r.URL.Path+" was not found")
		return
	}
	switch r.Method {
	case "GET":
		fwrite(w, 200, o)
	case "POST":
		if msg := apply(o, false); msg != "" {
			ferr(w, 400, msg)
			return
		}
		o["date_updated"] = cvUpdatedStamp
		fwrite(w, 200, o)
	default:
		_ = sid
		ferr(w, 405, "method not allowed")
	}
}

func (f *fakeTwilio) ssProfiles(w http.ResponseWriter, r *http.Request, form url.Values, parts []string) {
	notFound := func() { ferr(w, 404, "The requested resource "+r.URL.Path+" was not found") }
	if len(parts) == 1 {
		switch r.Method {
		case "POST":
			id := f.sid("HA")
			name := form.Get("UniqueName")
			if f.ssUniqueTaken("NetworkAccessProfiles", name, id) {
				ferr(w, 409, "Network Access Profile unique name "+name+" is already in use")
				return
			}
			o := map[string]any{"sid": id, "account_sid": testAccount, "unique_name": nil, "date_created": cvStamp, "date_updated": cvStamp,
				"url": "https://supersim.twilio.com/v1/NetworkAccessProfiles/" + id, "links": map[string]any{}}
			if name != "" {
				o["unique_name"] = name
			}
			for _, n := range form["Networks"] {
				net, ok := f.objs[key("ss", testAccount, "Networks/"+n)]
				if !ok {
					ferr(w, 400, "Network "+n+" does not exist")
					return
				}
				f.put("ss", testAccount, "NetworkAccessProfiles/"+id+"/Networks/"+n, f.ssProfileNetwork(id, net))
			}
			f.put("ss", testAccount, "NetworkAccessProfiles/"+id, o)
			fwrite(w, 201, o)
		case "GET":
			metaPage(w, r, "network_access_profiles", f.ssList("NetworkAccessProfiles", nil), 50)
		default:
			ferr(w, 405, "method not allowed")
		}
		return
	}
	sid, o := f.ssByKey("NetworkAccessProfiles", parts[1])
	if o == nil {
		notFound()
		return
	}
	switch len(parts) {
	case 2:
		switch r.Method {
		case "GET":
			fwrite(w, 200, o)
		case "POST":
			if v, ok := form["UniqueName"]; ok {
				if f.ssUniqueTaken("NetworkAccessProfiles", v[0], sid) {
					ferr(w, 409, "Network Access Profile unique name "+v[0]+" is already in use")
					return
				}
				o["unique_name"] = v[0]
			}
			o["date_updated"] = cvUpdatedStamp
			fwrite(w, 200, o)
		default:
			ferr(w, 405, "method not allowed")
		}
	case 3, 4:
		if parts[2] != "Networks" {
			notFound()
			return
		}
		base := "NetworkAccessProfiles/" + sid + "/Networks"
		if len(parts) == 3 {
			switch r.Method {
			case "GET":
				metaPage(w, r, "networks", f.ssList(base, nil), 2)
			case "POST":
				n := form.Get("Network")
				net, ok := f.objs[key("ss", testAccount, "Networks/"+n)]
				if !ok {
					ferr(w, 400, "Network "+n+" does not exist")
					return
				}
				po := f.ssProfileNetwork(sid, net)
				f.put("ss", testAccount, base+"/"+n, po)
				fwrite(w, 201, po)
			default:
				ferr(w, 405, "method not allowed")
			}
			return
		}
		po, ok := f.objs[key("ss", testAccount, base+"/"+parts[3])]
		if !ok {
			notFound()
			return
		}
		switch r.Method {
		case "GET":
			fwrite(w, 200, po)
		case "DELETE":
			f.del("ss", testAccount, base+"/"+parts[3])
			w.WriteHeader(204)
		default:
			ferr(w, 405, "method not allowed")
		}
	default:
		notFound()
	}
}

func (f *fakeTwilio) ssProfileNetwork(profile string, net map[string]any) map[string]any {
	return map[string]any{"sid": net["sid"], "network_access_profile_sid": profile, "friendly_name": net["friendly_name"],
		"iso_country": net["iso_country"], "identifiers": net["identifiers"], "url": "https://supersim.twilio.com/v1/NetworkAccessProfiles/" + profile + "/Networks/" + net["sid"].(string)}
}

var wirelessPlanFields = []fakeField{
	{"UniqueName", "unique_name", "s"}, {"FriendlyName", "friendly_name", "s"}, {"DataEnabled", "data_enabled", "b"}, {"DataLimit", "data_limit", "i"},
	{"DataMetering", "data_metering", "s"}, {"MessagingEnabled", "messaging_enabled", "b"}, {"VoiceEnabled", "voice_enabled", "b"},
	{"NationalRoamingEnabled", "national_roaming_enabled", "b"}, {"NationalRoamingDataLimit", "national_roaming_data_limit", "i"},
	{"InternationalRoamingDataLimit", "international_roaming_data_limit", "i"},
}

func (f *fakeTwilio) handleWireless(w http.ResponseWriter, r *http.Request, form url.Values, rest string) {
	parts := strings.Split(rest, "/")
	if parts[0] != "RatePlans" || len(parts) > 2 {
		ferr(w, 404, "The requested resource "+r.URL.Path+" was not found")
		return
	}
	if len(parts) == 1 {
		switch r.Method {
		case "POST":
			id := f.sid("WP")
			o := map[string]any{"sid": id, "account_sid": testAccount, "unique_name": nil, "friendly_name": nil, "data_enabled": true, "data_metering": "payg",
				"data_limit": float64(1000), "messaging_enabled": true, "voice_enabled": false, "national_roaming_enabled": true,
				"national_roaming_data_limit": float64(1000), "international_roaming": []any{}, "international_roaming_data_limit": float64(1000),
				"date_created": cvStamp, "date_updated": cvStamp, "url": "https://wireless.twilio.com/v1/RatePlans/" + id}
			if msg := applyForm(o, form, wirelessPlanFields); msg != "" {
				ferr(w, 400, msg)
				return
			}
			if v, ok := form["InternationalRoaming"]; ok {
				list := []any{}
				for _, s := range v {
					list = append(list, s)
				}
				o["international_roaming"] = list
			}
			if form.Get("DataLimitStrategy") != "" {
				if s := form.Get("DataLimitStrategy"); s != "block" && s != "throttle" {
					ferr(w, 400, "Invalid DataLimitStrategy "+s)
					return
				}
			}
			f.put("ss", testAccount, "RatePlans/"+id, o)
			fwrite(w, 201, o)
		case "GET":
			metaPage(w, r, "rate_plans", f.ssList("RatePlans", nil), 50)
		default:
			ferr(w, 405, "method not allowed")
		}
		return
	}
	sid, o := f.ssByKey("RatePlans", parts[1])
	if o == nil {
		ferr(w, 404, "The requested resource "+r.URL.Path+" was not found")
		return
	}
	switch r.Method {
	case "GET":
		fwrite(w, 200, o)
	case "POST":
		for k := range form {
			if k != "UniqueName" && k != "FriendlyName" {
				ferr(w, 400, "Unknown parameter "+k+" on update")
				return
			}
		}
		if msg := applyForm(o, form, wirelessPlanFields[:2]); msg != "" {
			ferr(w, 400, msg)
			return
		}
		o["date_updated"] = cvUpdatedStamp
		fwrite(w, 200, o)
	case "DELETE":
		f.del("ss", testAccount, "RatePlans/"+sid)
		w.WriteHeader(204)
	default:
		ferr(w, 405, "method not allowed")
	}
}
