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

// The SIP resources of the 2010-04-01 API, served for the SIP tests:
//
//	/Accounts/{acct}/SIP/CredentialLists[/{sid}]
//	/Accounts/{acct}/SIP/CredentialLists/{list}/Credentials[/{sid}]
//	/Accounts/{acct}/SIP/IpAccessControlLists[/{sid}]
//	/Accounts/{acct}/SIP/IpAccessControlLists/{list}/IpAddresses[/{sid}]
//	/Accounts/{acct}/SIP/Domains[/{sid}]
//	/Accounts/{acct}/SIP/Domains/{domain}/Auth/{Calls|Registrations}/{CredentialListMappings|IpAccessControlListMappings}[/{sid}]
//
// The field names follow twilio-oai spec/json/twilio_api_v2010.json. A credential's password
// is accepted but never returned (the fake keeps it under "_password" for assertions).

var (
	sipListFields   = []fakeField{{"FriendlyName", "friendly_name", "s"}}
	sipIPFields     = []fakeField{{"FriendlyName", "friendly_name", "s"}, {"IpAddress", "ip_address", "s"}, {"CidrPrefixLength", "cidr_prefix_length", "i"}}
	sipCredFields   = []fakeField{{"Username", "username", "s"}, {"Password", "_password", "s"}}
	sipCredUpdate   = []fakeField{{"Password", "_password", "s"}}
	sipDomainFields = []fakeField{
		{"DomainName", "domain_name", "s"}, {"FriendlyName", "friendly_name", "s"},
		{"VoiceUrl", "voice_url", "u"}, {"VoiceMethod", "voice_method", "m"},
		{"VoiceFallbackUrl", "voice_fallback_url", "u"}, {"VoiceFallbackMethod", "voice_fallback_method", "m"},
		{"VoiceStatusCallbackUrl", "voice_status_callback_url", "u"}, {"VoiceStatusCallbackMethod", "voice_status_callback_method", "m"},
		{"SipRegistration", "sip_registration", "b"}, {"EmergencyCallingEnabled", "emergency_calling_enabled", "b"},
		{"Secure", "secure", "b"}, {"ByocTrunkSid", "byoc_trunk_sid", "u"}, {"EmergencyCallerSid", "emergency_caller_sid", "u"},
	}
)

type sipKind struct {
	kind, prefix string
	fields       []fakeField
	update       []fakeField // nil means all of fields
	required     []string
	parentKind   string // kind of the list a child lives in
}

var sipKinds = map[string]sipKind{
	"CredentialLists":      {kind: "siplist", prefix: "CL", fields: sipListFields, required: []string{"FriendlyName"}},
	"IpAccessControlLists": {kind: "sipacl", prefix: "AL", fields: sipListFields, required: []string{"FriendlyName"}},
	"Domains":              {kind: "sipdomain", prefix: "SD", fields: sipDomainFields, required: []string{"DomainName"}},
	"Credentials":          {kind: "sipcred", prefix: "CR", fields: sipCredFields, update: sipCredUpdate, required: []string{"Username", "Password"}, parentKind: "siplist"},
	"IpAddresses":          {kind: "sipip", prefix: "IP", fields: sipIPFields, required: []string{"FriendlyName", "IpAddress"}, parentKind: "sipacl"},
}

// sipOut is the JSON a client sees: the password is never part of it.
func sipOut(o map[string]any) map[string]any {
	c := clone(o)
	delete(c, "_password")
	return c
}

func (f *fakeTwilio) handleSIP(w http.ResponseWriter, r *http.Request, form url.Values, parts []string) {
	// parts: Accounts, {acct}, SIP, {coll}[, {sid}] or Accounts, {acct}, SIP, {parentColl}, {parent}, {coll}[, {sid}]
	acct := parts[1]
	rest := parts[3:]
	notFound := func() { ferr(w, 404, "The requested resource "+r.URL.Path+" was not found") }
	if !f.accountKnown(acct) {
		notFound()
		return
	}
	if len(rest) >= 5 && rest[0] == "Domains" && rest[2] == "Auth" {
		f.handleSipMapping(w, r, form, acct, rest)
		return
	}
	var coll, parent, sid string
	switch len(rest) {
	case 1:
		coll = trimJSON(rest[0])
	case 2:
		coll, sid = trimJSON(rest[0]), trimJSON(rest[1])
	case 3:
		coll, parent = trimJSON(rest[2]), rest[1]
	case 4:
		coll, parent, sid = trimJSON(rest[2]), rest[1], trimJSON(rest[3])
	default:
		notFound()
		return
	}
	k, ok := sipKinds[coll]
	if !ok || (k.parentKind != "") != (parent != "") {
		notFound()
		return
	}
	if parent != "" {
		if _, live := f.objs[key(k.parentKind, acct, parent)]; !live {
			notFound()
			return
		}
	}
	if sid == "" { // collection
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
		o := map[string]any{"sid": f.sid(k.prefix), "account_sid": acct, "date_created": "Mon, 02 Nov 2026 00:00:00 +0000", "date_updated": "Mon, 02 Nov 2026 00:00:00 +0000"}
		switch k.kind {
		case "sipdomain":
			for kk, v := range map[string]any{"friendly_name": nil, "voice_url": nil, "voice_method": "POST", "voice_fallback_url": nil, "voice_fallback_method": "POST",
				"voice_status_callback_url": nil, "voice_status_callback_method": "POST", "sip_registration": false, "emergency_calling_enabled": false,
				"secure": false, "byoc_trunk_sid": nil, "emergency_caller_sid": nil, "auth_type": "", "api_version": "2010-04-01"} {
				o[kk] = v
			}
			if msg := f.sipDomainName(form.Get("DomainName"), ""); msg != "" {
				ferr(w, 400, msg)
				return
			}
		case "sipip":
			o["ip_access_control_list_sid"], o["cidr_prefix_length"] = parent, float64(32)
		case "sipcred":
			o["credential_list_sid"] = parent
			if len(form.Get("Password")) < 12 {
				ferr(w, 400, "Password must be at least 12 characters")
				return
			}
			for _, oo := range f.objs {
				if oo["credential_list_sid"] == parent && oo["username"] == form.Get("Username") {
					ferr(w, 400, "Username already exists in the credential list")
					return
				}
			}
		}
		if msg := applyForm(o, form, k.fields); msg != "" {
			ferr(w, 400, msg)
			return
		}
		f.put(k.kind, acct, o["sid"].(string), o)
		fwrite(w, 201, sipOut(o))
		return
	}
	o, live := f.objs[key(k.kind, acct, sid)]
	if !live || (parent != "" && o[k.parentField()] != parent) {
		notFound()
		return
	}
	switch r.Method {
	case "GET":
		fwrite(w, 200, sipOut(o))
	case "DELETE":
		f.del(k.kind, acct, sid)
		w.WriteHeader(204)
	case "POST":
		upd := k.fields
		if k.update != nil {
			upd = k.update
		}
		if k.kind == "sipdomain" {
			if msg := f.sipDomainName(form.Get("DomainName"), sid); form.Get("DomainName") != "" && msg != "" {
				ferr(w, 400, msg)
				return
			}
		}
		if k.kind == "sipcred" && form.Has("Password") && len(form.Get("Password")) < 12 {
			ferr(w, 400, "Password must be at least 12 characters")
			return
		}
		for _, req := range k.required {
			if v, sent := form[req]; sent && v[0] == "" && k.kind != "sipcred" {
				ferr(w, 400, "Missing required parameter "+req)
				return
			}
		}
		if msg := applyForm(o, form, upd); msg != "" {
			ferr(w, 400, msg)
			return
		}
		o["date_updated"] = "Tue, 03 Nov 2026 00:00:00 +0000"
		fwrite(w, 200, sipOut(o))
	default:
		ferr(w, 405, "method not allowed")
	}
}

func (k sipKind) parentField() string {
	if k.kind == "sipcred" {
		return "credential_list_sid"
	}
	return "ip_access_control_list_sid"
}

// sipDomainName validates a domain name as Twilio does (it must end with sip.twilio.com and be
// unique); self is the SID of the domain being renamed.
func (f *fakeTwilio) sipDomainName(name, self string) string {
	if !strings.HasSuffix(name, "sip.twilio.com") {
		return "Domain name must end with sip.twilio.com"
	}
	for k, o := range f.objs {
		if strings.HasPrefix(k, "sipdomain/") && o["domain_name"] == name && o["sid"] != self {
			return "Domain name is already in use"
		}
	}
	return ""
}

// sipCount is the number of live objects of a kind.
func (f *fakeTwilio) sipCount(kind string) int {
	n := 0
	for k := range f.objs {
		if strings.HasPrefix(k, kind+"/") {
			n++
		}
	}
	return n
}

// handleSipMapping serves the domain auth mappings. A mapping's SID is the SID of the mapped list
// and the domain's authType follows the mappings. rest: Domains, {domain}, Auth, {scope}, {kind}[, {sid}].
func (f *fakeTwilio) handleSipMapping(w http.ResponseWriter, r *http.Request, form url.Values, acct string, rest []string) {
	notFound := func() { ferr(w, 404, "The requested resource "+r.URL.Path+" was not found") }
	domain, scope, coll := rest[1], rest[3], trimJSON(rest[4])
	sid := ""
	if len(rest) == 6 {
		sid = trimJSON(rest[5])
	} else if len(rest) != 5 {
		notFound()
		return
	}
	var kind, listKind, listParam string
	switch {
	case coll == "CredentialListMappings" && (scope == "Calls" || scope == "Registrations"):
		kind, listKind, listParam = "sipmapcl", "siplist", "CredentialListSid"
	case coll == "IpAccessControlListMappings" && scope == "Calls":
		kind, listKind, listParam = "sipmapip", "sipacl", "IpAccessControlListSid"
	default:
		notFound()
		return
	}
	if _, live := f.objs[key("sipdomain", acct, domain)]; !live {
		notFound()
		return
	}
	mk := func(sid string) string { return sipMapKey(domain, scope, sid) }
	if sid == "" {
		if r.Method != "POST" {
			ferr(w, 405, "method not allowed")
			return
		}
		list := form.Get(listParam)
		if list == "" {
			ferr(w, 400, "Missing required parameter "+listParam)
			return
		}
		l, live := f.objs[key(listKind, acct, list)]
		if !live {
			notFound()
			return
		}
		if _, dup := f.objs[key(kind, acct, mk(list))]; dup {
			ferr(w, 400, "The list is already mapped to the domain")
			return
		}
		o := map[string]any{"sid": list, "account_sid": acct, "friendly_name": l["friendly_name"],
			"date_created": "Mon, 02 Nov 2026 00:00:00 +0000", "date_updated": "Mon, 02 Nov 2026 00:00:00 +0000"}
		f.put(kind, acct, mk(list), o)
		f.syncAuthType(acct, domain)
		fwrite(w, 201, o)
		return
	}
	o, live := f.objs[key(kind, acct, mk(sid))]
	if !live {
		notFound()
		return
	}
	switch r.Method {
	case "GET":
		fwrite(w, 200, o)
	case "DELETE":
		f.del(kind, acct, mk(sid))
		f.syncAuthType(acct, domain)
		w.WriteHeader(204)
	default:
		ferr(w, 405, "method not allowed")
	}
}

// sipMapKey is the fake's key of a mapping: the domain, the auth scope and the mapped list.
func sipMapKey(domain, scope, sid string) string { return domain + "|" + scope + "|" + sid }

// syncAuthType sets the domain's auth_type from its call mappings, as Twilio does.
func (f *fakeTwilio) syncAuthType(acct, domain string) {
	d, ok := f.objs[key("sipdomain", acct, domain)]
	if !ok {
		return
	}
	var types []string
	for _, kv := range []struct{ kind, t string }{{"sipmapip", "IP_ACL"}, {"sipmapcl", "CREDENTIAL_LIST"}} {
		for k := range f.objs {
			if strings.HasPrefix(k, kv.kind+"/"+acct+"/"+domain+"|Calls|") {
				types = append(types, kv.t)
				break
			}
		}
	}
	d["auth_type"] = strings.Join(types, ",")
}
