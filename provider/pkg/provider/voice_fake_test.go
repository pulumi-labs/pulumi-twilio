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

// The Voice v1 BYOC surface, served for the BYOC tests:
//
//	/voice/v1/ByocTrunks[/{sid}]
//	/voice/v1/ConnectionPolicies[/{sid}]
//	/voice/v1/ConnectionPolicies/{sid}/Targets[/{sid}]
//	/voice/v1/IpRecords[/{sid}]
//	/voice/v1/SourceIpMappings[/{sid}]
//
// The shapes follow twilio-oai spec/json/twilio_voice_v1.json (spec version 1.0.0). Deleting a
// connection policy removes its targets, as Twilio does.

const (
	voiceStamp        = "2026-11-06T00:00:00Z"
	voiceUpdatedStamp = "2026-11-07T00:00:00Z"
)

// voiceKind describes one top-level Voice v1 collection.
type voiceKind struct {
	kind, prefix string
	required     []string
	fields       []fakeField // param, key, typ
	init         map[string]any
	validate     func(o map[string]any, form url.Values, create bool) string
}

var voiceKinds = map[string]voiceKind{
	"ByocTrunks": {kind: "byoc", prefix: "BY",
		fields: []fakeField{{"FriendlyName", "friendly_name", "s"}, {"VoiceUrl", "voice_url", "s"}, {"VoiceMethod", "voice_method", "m"},
			{"VoiceFallbackUrl", "voice_fallback_url", "s"}, {"VoiceFallbackMethod", "voice_fallback_method", "m"},
			{"StatusCallbackUrl", "status_callback_url", "s"}, {"StatusCallbackMethod", "status_callback_method", "m"},
			{"CnamLookupEnabled", "cnam_lookup_enabled", "b"}, {"ConnectionPolicySid", "connection_policy_sid", "s"}, {"FromDomainSid", "from_domain_sid", "s"}},
		init: map[string]any{"friendly_name": nil, "voice_url": nil, "voice_method": "POST", "voice_fallback_url": nil, "voice_fallback_method": "POST",
			"status_callback_url": nil, "status_callback_method": "POST", "cnam_lookup_enabled": false, "connection_policy_sid": nil, "from_domain_sid": nil}},
	"ConnectionPolicies": {kind: "connpol", prefix: "NY",
		fields: []fakeField{{"FriendlyName", "friendly_name", "s"}}, init: map[string]any{"friendly_name": nil}},
	"IpRecords": {kind: "iprec", prefix: "IL", required: []string{"IpAddress"},
		fields: []fakeField{{"IpAddress", "ip_address", "s"}, {"CidrPrefixLength", "cidr_prefix_length", "i"}, {"FriendlyName", "friendly_name", "s"}},
		init:   map[string]any{"friendly_name": nil, "cidr_prefix_length": float64(32)},
		validate: func(o map[string]any, form url.Values, create bool) string {
			if !create {
				for _, p := range []string{"IpAddress", "CidrPrefixLength"} {
					if form.Get(p) != "" {
						return p + " cannot be updated"
					}
				}
				return ""
			}
			if strings.Contains(form.Get("IpAddress"), ":") {
				return "IPv6 is not supported"
			}
			return ""
		}},
	"SourceIpMappings": {kind: "srcmap", prefix: "IB", required: []string{"IpRecordSid", "SipDomainSid"},
		fields: []fakeField{{"IpRecordSid", "ip_record_sid", "s"}, {"SipDomainSid", "sip_domain_sid", "s"}},
		validate: func(o map[string]any, form url.Values, create bool) string {
			if !create && form.Get("IpRecordSid") != "" {
				return "IpRecordSid cannot be updated"
			}
			return ""
		}},
}

func (f *fakeTwilio) handleVoice(w http.ResponseWriter, r *http.Request, form url.Values, parts []string) {
	notFound := func() { ferr(w, 404, "The requested resource "+r.URL.Path+" was not found") }
	if parts[0] == "DialingPermissions" {
		f.handleDialingPermissions(w, r, parts[1:], notFound)
		return
	}
	k, ok := voiceKinds[parts[0]]
	if !ok {
		notFound()
		return
	}
	switch {
	case len(parts) == 1:
		f.voiceCreate(w, r, form, k)
	case len(parts) == 2:
		f.voiceInstance(w, r, form, k, parts[1], notFound)
	case len(parts) <= 4 && parts[0] == "ConnectionPolicies" && parts[2] == "Targets":
		if _, ok := f.objs[key("connpol", testAccount, parts[1])]; !ok {
			notFound()
			return
		}
		sid := ""
		if len(parts) == 4 {
			sid = parts[3]
		}
		f.voiceTarget(w, r, form, parts[1], sid, notFound)
	default:
		notFound()
	}
}

// voiceApply applies the posted fields and reports an API error for a value Twilio rejects.
func voiceApply(o map[string]any, form url.Values, fields []fakeField) string {
	for _, fld := range fields {
		v, ok := form[fld.param]
		if !ok {
			continue
		}
		switch fld.typ {
		case "m":
			if v[0] != "GET" && v[0] != "POST" {
				return "Invalid " + fld.param + " " + v[0]
			}
			o[fld.key] = v[0]
		case "b":
			b, err := strconv.ParseBool(v[0])
			if err != nil {
				return "Invalid " + fld.param
			}
			o[fld.key] = b
		case "i":
			n, err := strconv.Atoi(v[0])
			if err != nil {
				return fld.param + " must be an integer"
			}
			o[fld.key] = float64(n)
		default:
			o[fld.key] = v[0]
		}
	}
	return ""
}

func (f *fakeTwilio) voiceCreate(w http.ResponseWriter, r *http.Request, form url.Values, k voiceKind) {
	if r.Method != "POST" {
		ferr(w, 405, "method not allowed")
		return
	}
	for _, req := range k.required {
		if form.Get(req) == "" {
			ferr(w, 400, "Missing required parameter "+req)
			return
		}
	}
	o := map[string]any{"sid": f.sid(k.prefix), "account_sid": testAccount, "date_created": voiceStamp, "date_updated": voiceStamp}
	for key, v := range k.init {
		o[key] = v
	}
	if k.validate != nil {
		if msg := k.validate(o, form, true); msg != "" {
			ferr(w, 400, msg)
			return
		}
	}
	if msg := voiceApply(o, form, k.fields); msg != "" {
		ferr(w, 400, msg)
		return
	}
	f.put(k.kind, testAccount, o["sid"].(string), o)
	fwrite(w, 201, o)
}

func (f *fakeTwilio) voiceInstance(w http.ResponseWriter, r *http.Request, form url.Values, k voiceKind, sid string, notFound func()) {
	o, ok := f.objs[key(k.kind, testAccount, sid)]
	if !ok {
		notFound()
		return
	}
	switch r.Method {
	case "GET":
		fwrite(w, 200, o)
	case "POST":
		if k.validate != nil {
			if msg := k.validate(o, form, false); msg != "" {
				ferr(w, 400, msg)
				return
			}
		}
		if msg := voiceApply(o, form, k.fields); msg != "" {
			ferr(w, 400, msg)
			return
		}
		o["date_updated"] = voiceUpdatedStamp
		fwrite(w, 200, o)
	case "DELETE":
		if k.kind == "connpol" {
			for key, c := range f.objs {
				if c["connection_policy_sid"] == sid && strings.HasPrefix(key, "polytarget/") {
					delete(f.objs, key)
				}
			}
		}
		f.del(k.kind, testAccount, sid)
		w.WriteHeader(204)
	default:
		ferr(w, 405, "method not allowed")
	}
}

func (f *fakeTwilio) voiceTarget(w http.ResponseWriter, r *http.Request, form url.Values, policy, sid string, notFound func()) {
	fields := []fakeField{{"Target", "target", "s"}, {"FriendlyName", "friendly_name", "s"}, {"Priority", "priority", "i"},
		{"Weight", "weight", "i"}, {"Enabled", "enabled", "b"}}
	apply := func(o map[string]any) string {
		if t := form.Get("Target"); t != "" && !strings.HasPrefix(t, "sip:") {
			return "Target must use the sip: scheme"
		}
		return voiceApply(o, form, fields)
	}
	if sid == "" {
		if r.Method != "POST" {
			ferr(w, 405, "method not allowed")
			return
		}
		if form.Get("Target") == "" {
			ferr(w, 400, "Missing required parameter Target")
			return
		}
		o := map[string]any{"sid": f.sid("NE"), "account_sid": testAccount, "connection_policy_sid": policy, "friendly_name": nil,
			"priority": float64(10), "weight": float64(10), "enabled": true, "date_created": voiceStamp, "date_updated": voiceStamp}
		if msg := apply(o); msg != "" {
			ferr(w, 400, msg)
			return
		}
		f.put("polytarget", testAccount, o["sid"].(string), o)
		fwrite(w, 201, o)
		return
	}
	o, ok := f.objs[key("polytarget", testAccount, sid)]
	if !ok || o["connection_policy_sid"] != policy {
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
		o["date_updated"] = voiceUpdatedStamp
		fwrite(w, 200, o)
	case "DELETE":
		f.del("polytarget", testAccount, sid)
		w.WriteHeader(204)
	default:
		ferr(w, 405, "method not allowed")
	}
}
