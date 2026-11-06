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

// The Elastic SIP Trunking v1 API (trunking.twilio.com), served for the trunk tests:
//
//	/trunking/v1/Trunks[/{sid}]
//	/trunking/v1/Trunks/{sid}/Recording
//	/trunking/v1/Trunks/{sid}/OriginationUrls[/{sid}]
//	/trunking/v1/Trunks/{sid}/CredentialLists[/{sid}]
//	/trunking/v1/Trunks/{sid}/IpAccessControlLists[/{sid}]
//
// The shapes follow twilio-oai spec/json/twilio_trunking_v1.json. Trunks have no account in the path,
// the recording settings are read and changed only on the Recording sub-resource, and the credential
// and IP access control list attachments cannot be updated.

const trunkStamp = "2026-11-06T00:00:00Z"

var trunkEnums = map[string][]string{
	"TransferMode":           {"disable-all", "enable-all", "sip-only"},
	"TransferCallerId":       {"from-transferee", "from-transferor"},
	"DisasterRecoveryMethod": {"GET", "POST"},
}

var trunkFormFields = []struct{ param, key string }{
	{"FriendlyName", "friendly_name"}, {"DomainName", "domain_name"}, {"DisasterRecoveryUrl", "disaster_recovery_url"},
	{"DisasterRecoveryMethod", "disaster_recovery_method"}, {"TransferMode", "transfer_mode"}, {"TransferCallerId", "transfer_caller_id"},
	{"Secure", "secure"}, {"CnamLookupEnabled", "cnam_lookup_enabled"},
}

func (f *fakeTwilio) handleTrunking(w http.ResponseWriter, r *http.Request, form url.Values, parts []string) {
	notFound := func() { ferr(w, 404, "The requested resource "+r.URL.Path+" was not found") }
	switch {
	case len(parts) == 1 && parts[0] == "Trunks":
		f.trunkCreate(w, r, form)
	case len(parts) == 2 && parts[0] == "Trunks":
		f.trunkInstance(w, r, form, parts[1], notFound)
	case len(parts) >= 3 && parts[0] == "Trunks" && len(parts) <= 4:
		if _, ok := f.objs[key("trunk", testAccount, parts[1])]; !ok {
			notFound()
			return
		}
		sid := ""
		if len(parts) == 4 {
			sid = parts[3]
		}
		switch parts[2] {
		case "Recording":
			f.trunkRecording(w, r, form, parts[1], sid, notFound)
		case "OriginationUrls":
			f.trunkOrigination(w, r, form, parts[1], sid, notFound)
		case "CredentialLists":
			f.trunkAttachment(w, r, form, parts[1], sid, "trunkcl", "siplist", "CredentialListSid", notFound)
		case "IpAccessControlLists":
			f.trunkAttachment(w, r, form, parts[1], sid, "trunkacl", "sipacl", "IpAccessControlListSid", notFound)
		default:
			notFound()
		}
	default:
		notFound()
	}
}

// trunkApply applies the posted trunk fields and reports an API error for a value Twilio rejects.
func trunkApply(o map[string]any, form url.Values) string {
	for param, allowed := range trunkEnums {
		if v, ok := form[param]; ok {
			found := false
			for _, a := range allowed {
				found = found || a == v[0]
			}
			if !found {
				return "Invalid " + param + " " + v[0]
			}
		}
	}
	if d := form.Get("DomainName"); d != "" && !strings.HasSuffix(d, ".pstn.twilio.com") {
		return "DomainName must end with pstn.twilio.com"
	}
	for _, fld := range trunkFormFields {
		v, ok := form[fld.param]
		if !ok {
			continue
		}
		switch fld.param {
		case "Secure", "CnamLookupEnabled":
			b, err := strconv.ParseBool(v[0])
			if err != nil {
				return "Invalid " + fld.param
			}
			o[fld.key] = b
		default:
			o[fld.key] = v[0]
		}
	}
	return ""
}

func trunkOut(o map[string]any) map[string]any {
	c := clone(o)
	rec := map[string]any{"mode": o["_rec_mode"], "trim": o["_rec_trim"]}
	delete(c, "_rec_mode")
	delete(c, "_rec_trim")
	c["recording"] = rec
	return c
}

func (f *fakeTwilio) trunkCreate(w http.ResponseWriter, r *http.Request, form url.Values) {
	if r.Method != "POST" {
		ferr(w, 405, "method not allowed")
		return
	}
	sid := f.sid("TK")
	o := map[string]any{"sid": sid, "account_sid": testAccount, "friendly_name": nil, "domain_name": nil,
		"disaster_recovery_url": "", "disaster_recovery_method": "POST", "transfer_mode": "disable-all", "transfer_caller_id": "from-transferee",
		"secure": false, "cnam_lookup_enabled": false, "auth_type": "", "symmetric_rtp_enabled": true,
		"date_created": trunkStamp, "date_updated": trunkStamp, "_rec_mode": "do-not-record", "_rec_trim": "do-not-trim"}
	if msg := trunkApply(o, form); msg != "" {
		ferr(w, 400, msg)
		return
	}
	f.put("trunk", testAccount, sid, o)
	fwrite(w, 201, trunkOut(o))
}

func (f *fakeTwilio) trunkInstance(w http.ResponseWriter, r *http.Request, form url.Values, sid string, notFound func()) {
	o, ok := f.objs[key("trunk", testAccount, sid)]
	if !ok {
		notFound()
		return
	}
	switch r.Method {
	case "GET":
		fwrite(w, 200, trunkOut(o))
	case "POST":
		if msg := trunkApply(o, form); msg != "" {
			ferr(w, 400, msg)
			return
		}
		o["date_updated"] = "2026-11-07T00:00:00Z"
		fwrite(w, 200, trunkOut(o))
	case "DELETE":
		for k, c := range f.objs {
			if c["trunk_sid"] == sid {
				delete(f.objs, k)
			}
		}
		f.del("trunk", testAccount, sid)
		w.WriteHeader(204)
	default:
		ferr(w, 405, "method not allowed")
	}
}

func (f *fakeTwilio) trunkRecording(w http.ResponseWriter, r *http.Request, form url.Values, trunk, sid string, notFound func()) {
	if sid != "" {
		notFound()
		return
	}
	o := f.objs[key("trunk", testAccount, trunk)]
	switch r.Method {
	case "GET":
	case "POST":
		modes := map[string]bool{"do-not-record": true, "record-from-ringing": true, "record-from-answer": true, "record-from-ringing-dual": true, "record-from-answer-dual": true}
		trims := map[string]bool{"do-not-trim": true, "trim-silence": true}
		if m := form.Get("Mode"); m != "" {
			if !modes[m] {
				ferr(w, 400, "Invalid Mode "+m)
				return
			}
			o["_rec_mode"] = m
		}
		if t := form.Get("Trim"); t != "" {
			if !trims[t] {
				ferr(w, 400, "Invalid Trim "+t)
				return
			}
			o["_rec_trim"] = t
		}
	default:
		ferr(w, 405, "method not allowed")
		return
	}
	fwrite(w, 200, map[string]any{"mode": o["_rec_mode"], "trim": o["_rec_trim"], "trunk_sid": trunk})
}

func (f *fakeTwilio) trunkOrigination(w http.ResponseWriter, r *http.Request, form url.Values, trunk, sid string, notFound func()) {
	apply := func(o map[string]any) string {
		if u := form.Get("SipUrl"); u != "" {
			if !strings.HasPrefix(u, "sip:") {
				return "SipUrl must use the sip: scheme"
			}
			o["sip_url"] = u
		}
		if v := form.Get("FriendlyName"); v != "" {
			o["friendly_name"] = v
		}
		for param, k := range map[string]string{"Weight": "weight", "Priority": "priority"} {
			if v := form.Get(param); v != "" {
				n, err := strconv.Atoi(v)
				if err != nil {
					return param + " must be an integer"
				}
				o[k] = float64(n)
			}
		}
		if v := form.Get("Enabled"); v != "" {
			b, err := strconv.ParseBool(v)
			if err != nil {
				return "Enabled must be a boolean"
			}
			o["enabled"] = b
		}
		return ""
	}
	if sid == "" {
		if r.Method != "POST" {
			ferr(w, 405, "method not allowed")
			return
		}
		for _, req := range []string{"Weight", "Priority", "Enabled", "FriendlyName", "SipUrl"} {
			if form.Get(req) == "" {
				ferr(w, 400, "Missing required parameter "+req)
				return
			}
		}
		o := map[string]any{"sid": f.sid("OU"), "account_sid": testAccount, "trunk_sid": trunk, "date_created": trunkStamp, "date_updated": trunkStamp}
		if msg := apply(o); msg != "" {
			ferr(w, 400, msg)
			return
		}
		f.put("trunkou", testAccount, o["sid"].(string), o)
		fwrite(w, 201, o)
		return
	}
	o, ok := f.objs[key("trunkou", testAccount, sid)]
	if !ok || o["trunk_sid"] != trunk {
		notFound()
		return
	}
	switch r.Method {
	case "GET":
		fwrite(w, 200, o)
	case "POST":
		if msg := apply(o); msg != "" {
			ferr(w, 400, msg)
			return
		}
		o["date_updated"] = "2026-11-07T00:00:00Z"
		fwrite(w, 200, o)
	case "DELETE":
		f.del("trunkou", testAccount, sid)
		w.WriteHeader(204)
	default:
		ferr(w, 405, "method not allowed")
	}
}

// trunkAttachment serves CredentialLists and IpAccessControlLists of a trunk: attach, fetch, list and detach.
func (f *fakeTwilio) trunkAttachment(w http.ResponseWriter, r *http.Request, form url.Values, trunk, sid, kind, listKind, param string, notFound func()) {
	if sid == "" {
		if r.Method != "POST" {
			ferr(w, 405, "method not allowed")
			return
		}
		list := form.Get(param)
		if list == "" {
			ferr(w, 400, "Missing required parameter "+param)
			return
		}
		l, ok := f.objs[key(listKind, testAccount, list)]
		if !ok {
			ferr(w, 404, "The list "+list+" was not found")
			return
		}
		if _, dup := f.objs[key(kind, testAccount, trunk+"/"+list)]; dup {
			ferr(w, 409, "The list is already attached to the trunk")
			return
		}
		o := map[string]any{"sid": list, "account_sid": testAccount, "trunk_sid": trunk, "friendly_name": l["friendly_name"],
			"date_created": trunkStamp, "date_updated": trunkStamp}
		f.put(kind, testAccount, trunk+"/"+list, o)
		f.setTrunkAuth(trunk)
		fwrite(w, 201, o)
		return
	}
	o, ok := f.objs[key(kind, testAccount, trunk+"/"+sid)]
	if !ok {
		notFound()
		return
	}
	switch r.Method {
	case "GET":
		fwrite(w, 200, o)
	case "DELETE":
		f.del(kind, testAccount, trunk+"/"+sid)
		f.setTrunkAuth(trunk)
		w.WriteHeader(204)
	default:
		ferr(w, 405, "method not allowed")
	}
}

// setTrunkAuth recomputes auth_type from the attached lists.
func (f *fakeTwilio) setTrunkAuth(trunk string) {
	var kinds []string
	for _, k := range []struct{ kind, name string }{{"trunkacl", "IP_ACL"}, {"trunkcl", "CREDENTIAL_LIST"}} {
		for _, c := range f.objs {
			if c["trunk_sid"] == trunk && strings.HasPrefix(c["sid"].(string), map[string]string{"trunkacl": "AL", "trunkcl": "CL"}[k.kind]) {
				kinds = append(kinds, k.name)
				break
			}
		}
	}
	f.objs[key("trunk", testAccount, trunk)]["auth_type"] = strings.Join(kinds, ",")
}
