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
	"strings"
	"testing"

	pgp "github.com/pulumi/pulumi-go-provider"
	"github.com/pulumi/pulumi/sdk/v3/go/property"
	"github.com/stretchr/testify/require"
)

const sipPassword = "Sup3rSecretPass1"

// ------------------------------------------------- lists (credential and IP ACL)

func TestSipListsLC_Lifecycle(t *testing.T) {
	for _, tc := range []struct{ typ, kind, prefix, coll string }{
		{"SipCredentialList", "siplist", "CL", "/SIP/CredentialLists"},
		{"SipIpAccessControlList", "sipacl", "AL", "/SIP/IpAccessControlLists"},
	} {
		t.Run(tc.typ, func(t *testing.T) {
			l := newTL(t, tc.typ, "l")
			in := pv(map[string]property.Value{"friendlyName": sv("office")})
			c := l.create(in)
			sid := c.ID
			require.True(t, strings.HasPrefix(sid, tc.prefix), sid)
			require.Equal(t, sid, getS(t, c.Properties, "sid"))
			require.Equal(t, testAccount, getS(t, c.Properties, "ownerAccountSid"))
			require.Equal(t, "office", l.f.get(tc.kind, testAccount, sid)["friendly_name"])
			requireNoChanges(t, l.diff(sid, c.Properties, in, in))

			// A refresh changes nothing, and an out-of-band rename is drift on the input.
			rd := l.read(sid, c.Properties, in)
			requireNoChanges(t, l.diff(sid, rd.Properties, in, rd.Inputs))
			l.f.mutate(tc.kind, testAccount, sid, func(o map[string]any) { o["friendly_name"] = "renamed" })
			rd = l.read(sid, c.Properties, in)
			require.Equal(t, "renamed", getS(t, rd.Inputs, "friendlyName"))
			l.f.mutate(tc.kind, testAccount, sid, func(o map[string]any) { o["friendly_name"] = "office" })

			// The name updates in place; the account replaces.
			next := pv(map[string]property.Value{"friendlyName": sv("hq")})
			d := l.diff(sid, rd.Properties, next, in)
			require.True(t, d.HasChanges)
			require.False(t, d.DeleteBeforeReplace)
			require.Equal(t, pgp.Update, d.DetailedDiff["friendlyName"].Kind)
			u := l.update(sid, rd.Properties, next, in)
			require.Equal(t, "hq", l.f.get(tc.kind, testAccount, sid)["friendly_name"])
			require.Len(t, l.f.callsMatching("POST", tc.coll+"/"+sid+".json"), 1)
			requireNoChanges(t, l.diff(sid, u.Properties, next, next))
			d = l.diff(sid, u.Properties, with(next, map[string]property.Value{"accountSid": sv(fakeSub)}), next)
			require.Equal(t, pgp.AddReplace, d.DetailedDiff["accountSid"].Kind)

			// Gone out of band: a refresh drops it, and deleting twice is fine.
			l.f.remove(tc.kind, testAccount, sid)
			require.Empty(t, l.read(sid, u.Properties, next).ID)
			l.del(sid, u.Properties)
			c = l.create(in)
			l.del(c.ID, c.Properties)
			require.Nil(t, l.f.get(tc.kind, testAccount, c.ID))
		})
	}
}

func TestSipListsLC_ImportPreviewAndChecks(t *testing.T) {
	for _, tc := range []struct{ typ, kind string }{{"SipCredentialList", "siplist"}, {"SipIpAccessControlList", "sipacl"}} {
		t.Run(tc.typ, func(t *testing.T) {
			l := newTL(t, tc.typ, "l")
			l.f.addSubaccount(fakeSub)
			in := pv(map[string]property.Value{"friendlyName": sv("branch"), "accountSid": sv(fakeSub)})
			c := l.create(in)
			_, sid := splitScoped(c.ID)
			require.Equal(t, fakeSub+"/"+sid, c.ID)

			// Import by the composite ID, then by the bare SID of a provider-account list.
			imp := l.read(c.ID, empty(), empty())
			require.Equal(t, fakeSub, getS(t, imp.Inputs, "accountSid"))
			require.Equal(t, "branch", getS(t, imp.Inputs, "friendlyName"))
			requireNoChanges(t, l.diff(imp.ID, imp.Properties, in, imp.Inputs))
			own := l.create(pv(map[string]property.Value{"friendlyName": sv("own")}))
			imp = l.read(own.ID, empty(), empty())
			require.False(t, hasProp(imp.Inputs, "accountSid"))
			require.Equal(t, "own", getS(t, imp.Inputs, "friendlyName"))
			requireNoChanges(t, l.diff(own.ID, imp.Properties, pv(map[string]property.Value{"friendlyName": sv("own")}), imp.Inputs))

			// A preview creates nothing and invents no ID.
			before := l.f.sipCount(tc.kind)
			pr, err := l.s.Create(pgp.CreateRequest{Urn: l.urn, Properties: pv(map[string]property.Value{"friendlyName": sv("x")}), DryRun: true})
			require.NoError(t, err)
			require.Empty(t, pr.ID)
			require.Equal(t, before, l.f.sipCount(tc.kind))

			require.NotEmpty(t, l.check(pv(map[string]property.Value{"friendlyName": sv("")})).Failures)
			require.NotEmpty(t, l.check(pv(map[string]property.Value{"friendlyName": sv(strings.Repeat("x", 256))})).Failures)
			require.Empty(t, l.check(pv(map[string]property.Value{"friendlyName": computed(), "accountSid": computed()})).Failures)
		})
	}
}

// ------------------------------------------------------------- SipCredential

func credInputs(list string) property.Map {
	return pv(map[string]property.Value{"credentialListSid": sv(list), "username": sv("alice"), "password": sv(sipPassword).WithSecret(true)})
}

func TestSipCredentialLC_Lifecycle(t *testing.T) {
	lists := newTL(t, "SipCredentialList", "l")
	list := lists.create(pv(map[string]property.Value{"friendlyName": sv("users")}))
	l := lists.other("SipCredential", "c")
	in := credInputs(list.ID)
	c := l.create(in)
	require.True(t, strings.HasPrefix(c.ID, list.ID+"/CR"), c.ID)
	_, _, sid, err := splitChildID(c.ID)
	require.NoError(t, err)
	require.Equal(t, sid, getS(t, c.Properties, "sid"))
	require.Equal(t, testAccount, getS(t, c.Properties, "ownerAccountSid"))
	require.Equal(t, "alice", getS(t, c.Properties, "username"))
	require.True(t, c.Properties.Get("password").Secret(), "the password is a secret in state")
	require.Equal(t, sipPassword, c.Properties.Get("password").WithSecret(false).AsString())
	require.Equal(t, sipPassword, l.f.get("sipcred", testAccount, sid)["_password"])
	requireNoChanges(t, l.diff(c.ID, c.Properties, in, in))

	// A refresh keeps the password the API never returns and plans no change.
	rd := l.read(c.ID, c.Properties, in)
	require.Equal(t, sipPassword, rd.Properties.Get("password").WithSecret(false).AsString())
	requireNoChanges(t, l.diff(c.ID, rd.Properties, in, rd.Inputs))
	_, hasPw := l.f.get("sipcred", testAccount, sid)["password"]
	require.False(t, hasPw, "the fake never returns the password")

	// The password updates in place and only the password is sent.
	next := with(in, map[string]property.Value{"password": sv("An0therSecretPass").WithSecret(true)})
	d := l.diff(c.ID, rd.Properties, next, in)
	require.True(t, d.HasChanges)
	require.Equal(t, pgp.Update, d.DetailedDiff["password"].Kind)
	u := l.update(c.ID, rd.Properties, next, in)
	require.Equal(t, "An0therSecretPass", l.f.get("sipcred", testAccount, sid)["_password"])
	upd := l.f.callsMatching("POST", "/Credentials/"+sid+".json")
	require.Len(t, upd, 1)
	require.Equal(t, []string{"Password"}, keysOf(upd[0].Form))
	require.True(t, u.Properties.Get("password").Secret())
	requireNoChanges(t, l.diff(c.ID, u.Properties, next, next))

	// The username, list and account replace the credential.
	for k, v := range map[string]property.Value{"username": sv("bob"), "credentialListSid": sv("CL0000"), "accountSid": sv(fakeSub)} {
		d = l.diff(c.ID, u.Properties, with(next, map[string]property.Value{k: v}), next)
		require.Contains(t, []pgp.DiffKind{pgp.UpdateReplace, pgp.AddReplace}, d.DetailedDiff[k].Kind, k)
	}

	// Gone out of band, and deleted twice.
	l.f.remove("sipcred", testAccount, sid)
	require.Empty(t, l.read(c.ID, u.Properties, next).ID)
	l.del(c.ID, u.Properties)
	c = l.create(in)
	l.del(c.ID, c.Properties)
	_, _, sid, _ = splitChildID(c.ID)
	require.Nil(t, l.f.get("sipcred", testAccount, sid))
}

func keysOf(v map[string][]string) []string {
	out := []string{}
	for k := range v {
		out = append(out, k)
	}
	return out
}

func TestSipCredentialLC_ImportPreviewAndChecks(t *testing.T) {
	lists := newTL(t, "SipCredentialList", "l")
	lists.f.addSubaccount(fakeSub)
	list := lists.create(pv(map[string]property.Value{"friendlyName": sv("users"), "accountSid": sv(fakeSub)}))
	_, listSid := splitScoped(list.ID)
	l := lists.other("SipCredential", "c")
	in := with(credInputs(listSid), map[string]property.Value{"accountSid": sv(fakeSub)})
	c := l.create(in)
	_, _, sid, err := splitChildID(c.ID)
	require.NoError(t, err)
	require.Equal(t, fakeSub+"/"+listSid+"/"+sid, c.ID)

	// Import by the three-part ID: no password comes back, the list and account are recovered.
	imp := l.read(c.ID, empty(), empty())
	require.Equal(t, fakeSub, getS(t, imp.Inputs, "accountSid"))
	require.Equal(t, listSid, getS(t, imp.Inputs, "credentialListSid"))
	require.Equal(t, "alice", getS(t, imp.Inputs, "username"))
	require.Equal(t, listSid, getS(t, imp.Properties, "credentialListSid"))

	// Import by <credentialListSid>/<sid> in the provider account; the password is recorded by the program's next update.
	own := lists.other("SipCredentialList", "own").create(pv(map[string]property.Value{"friendlyName": sv("own")}))
	oc := l.create(credInputs(own.ID))
	imp = l.read(oc.ID, empty(), empty())
	require.Equal(t, own.ID, getS(t, imp.Inputs, "credentialListSid"))
	require.False(t, hasProp(imp.Inputs, "accountSid"))
	d := l.diff(oc.ID, imp.Properties, credInputs(own.ID), imp.Inputs)
	require.True(t, d.HasChanges, "an imported credential has no password until the program sets one")
	require.Equal(t, pgp.Update, d.DetailedDiff["password"].Kind)
	_, err = l.s.Read(pgp.ReadRequest{ID: "bare-sid", Urn: l.urn, Properties: empty(), Inputs: empty()})
	require.ErrorContains(t, err, "invalid ID")

	// A preview creates nothing and invents no ID.
	before := l.f.sipCount("sipcred")
	pr, err := l.s.Create(pgp.CreateRequest{Urn: l.urn, Properties: credInputs(own.ID), DryRun: true})
	require.NoError(t, err)
	require.Empty(t, pr.ID)
	require.Equal(t, before, l.f.sipCount("sipcred"))

	require.NotEmpty(t, l.check(with(credInputs("CL1"), map[string]property.Value{"username": sv("")})).Failures)
	require.NotEmpty(t, l.check(with(credInputs("CL1"), map[string]property.Value{"username": sv(strings.Repeat("u", 33))})).Failures)
	require.NotEmpty(t, l.check(with(credInputs("CL1"), map[string]property.Value{"password": sv("short1A")})).Failures)
	require.NotEmpty(t, l.check(credInputs("")).Failures)
	require.Empty(t, l.check(pv(map[string]property.Value{"credentialListSid": computed(), "username": computed(), "password": computed(), "accountSid": computed()})).Failures)

	// Deleting a list's parent out of band makes the child gone as well.
	lists.f.remove("siplist", testAccount, own.ID)
	require.Empty(t, l.read(oc.ID, oc.Properties, credInputs(own.ID)).ID)
}

// ---------------------------------------------------------------- SipIpAddress

func ipInputs(acl string) property.Map {
	return pv(map[string]property.Value{"ipAccessControlListSid": sv(acl), "friendlyName": sv("office"), "ipAddress": sv("203.0.113.7")})
}

func TestSipIpAddressLC_Lifecycle(t *testing.T) {
	acls := newTL(t, "SipIpAccessControlList", "a")
	acl := acls.create(pv(map[string]property.Value{"friendlyName": sv("offices")}))
	l := acls.other("SipIpAddress", "ip")
	in := ipInputs(acl.ID)
	c := l.create(in)
	require.True(t, strings.HasPrefix(c.ID, acl.ID+"/IP"), c.ID)
	_, _, sid, err := splitChildID(c.ID)
	require.NoError(t, err)
	require.Equal(t, sid, getS(t, c.Properties, "sid"))
	stored := l.f.get("sipip", testAccount, sid)
	require.Equal(t, "203.0.113.7", stored["ip_address"])
	require.Equal(t, 32.0, stored["cidr_prefix_length"], "the server default")
	require.Empty(t, l.f.callsMatching("POST", "/IpAddresses.json")[0].Form.Get("CidrPrefixLength"), "an unset prefix is not sent")
	requireNoChanges(t, l.diff(c.ID, c.Properties, in, in))

	// A refresh with the default prefix plans no change; drift on the address is detected.
	rd := l.read(c.ID, c.Properties, in)
	require.False(t, hasProp(rd.Inputs, "cidrPrefixLength"))
	requireNoChanges(t, l.diff(c.ID, rd.Properties, in, rd.Inputs))
	l.f.mutate("sipip", testAccount, sid, func(o map[string]any) { o["ip_address"] = "198.51.100.1" })
	rd = l.read(c.ID, c.Properties, in)
	require.Equal(t, "198.51.100.1", getS(t, rd.Inputs, "ipAddress"))
	l.f.mutate("sipip", testAccount, sid, func(o map[string]any) { o["ip_address"] = "203.0.113.7" })

	// Name, address and prefix all update in place.
	next := with(in, map[string]property.Value{"friendlyName": sv("dc"), "ipAddress": sv("203.0.113.0"), "cidrPrefixLength": nv(24)})
	d := l.diff(c.ID, c.Properties, next, in)
	require.True(t, d.HasChanges)
	require.False(t, d.DeleteBeforeReplace)
	for _, k := range []string{"friendlyName", "ipAddress"} {
		require.Equal(t, pgp.Update, d.DetailedDiff[k].Kind, k)
	}
	require.Equal(t, pgp.Add, d.DetailedDiff["cidrPrefixLength"].Kind)
	u := l.update(c.ID, c.Properties, next, in)
	stored = l.f.get("sipip", testAccount, sid)
	require.Equal(t, "dc", stored["friendly_name"])
	require.Equal(t, "203.0.113.0", stored["ip_address"])
	require.Equal(t, 24.0, stored["cidr_prefix_length"])
	requireNoChanges(t, l.diff(c.ID, u.Properties, next, next))

	// Removing the prefix resets it to 32.
	u2 := l.update(c.ID, u.Properties, in, next)
	require.Equal(t, 32.0, l.f.get("sipip", testAccount, sid)["cidr_prefix_length"])
	requireNoChanges(t, l.diff(c.ID, u2.Properties, in, in))

	// The list and account replace.
	for k, v := range map[string]property.Value{"ipAccessControlListSid": sv("AL0000"), "accountSid": sv(fakeSub)} {
		d = l.diff(c.ID, u2.Properties, with(in, map[string]property.Value{k: v}), in)
		require.Contains(t, []pgp.DiffKind{pgp.UpdateReplace, pgp.AddReplace}, d.DetailedDiff[k].Kind, k)
	}

	l.f.remove("sipip", testAccount, sid)
	require.Empty(t, l.read(c.ID, u2.Properties, in).ID)
	l.del(c.ID, u2.Properties)
	c = l.create(in)
	l.del(c.ID, c.Properties)
}

func TestSipIpAddressLC_ImportPreviewAndChecks(t *testing.T) {
	acls := newTL(t, "SipIpAccessControlList", "a")
	acls.f.addSubaccount(fakeSub)
	acl := acls.create(pv(map[string]property.Value{"friendlyName": sv("offices"), "accountSid": sv(fakeSub)}))
	_, aclSid := splitScoped(acl.ID)
	l := acls.other("SipIpAddress", "ip")
	in := with(ipInputs(aclSid), map[string]property.Value{"accountSid": sv(fakeSub), "cidrPrefixLength": nv(28)})
	c := l.create(in)
	_, _, sid, _ := splitChildID(c.ID)
	require.Equal(t, fakeSub+"/"+aclSid+"/"+sid, c.ID)
	imp := l.read(c.ID, empty(), empty())
	require.Equal(t, fakeSub, getS(t, imp.Inputs, "accountSid"))
	require.Equal(t, aclSid, getS(t, imp.Inputs, "ipAccessControlListSid"))
	require.Equal(t, 28.0, imp.Inputs.Get("cidrPrefixLength").AsNumber())
	requireNoChanges(t, l.diff(imp.ID, imp.Properties, in, imp.Inputs))

	// Import by <ipAccessControlListSid>/<sid> in the provider account; a /32 is not adopted.
	own := acls.other("SipIpAccessControlList", "own").create(pv(map[string]property.Value{"friendlyName": sv("own")}))
	oc := l.create(ipInputs(own.ID))
	imp = l.read(oc.ID, empty(), empty())
	require.False(t, hasProp(imp.Inputs, "accountSid"))
	require.False(t, hasProp(imp.Inputs, "cidrPrefixLength"))
	requireNoChanges(t, l.diff(oc.ID, imp.Properties, ipInputs(own.ID), imp.Inputs))

	before := l.f.sipCount("sipip")
	pr, err := l.s.Create(pgp.CreateRequest{Urn: l.urn, Properties: ipInputs(own.ID), DryRun: true})
	require.NoError(t, err)
	require.Empty(t, pr.ID)
	require.Equal(t, before, l.f.sipCount("sipip"))

	set := func(k string, v property.Value) property.Map {
		return with(ipInputs("AL1"), map[string]property.Value{k: v})
	}
	require.NotEmpty(t, l.check(set("friendlyName", sv(""))).Failures)
	require.NotEmpty(t, l.check(set("ipAddress", sv(""))).Failures)
	require.NotEmpty(t, l.check(set("cidrPrefixLength", nv(33))).Failures)
	require.NotEmpty(t, l.check(set("cidrPrefixLength", nv(-1))).Failures)
	require.NotEmpty(t, l.check(ipInputs("")).Failures)
	require.Empty(t, l.check(pv(map[string]property.Value{"ipAccessControlListSid": computed(), "friendlyName": computed(),
		"ipAddress": computed(), "cidrPrefixLength": computed(), "accountSid": computed()})).Failures)
}

// ------------------------------------------------------------------- SipDomain

func domainInputs() property.Map {
	return pv(map[string]property.Value{"domainName": sv("acme.sip.twilio.com"), "friendlyName": sv("acme"),
		"voiceUrl": sv("https://example.com/voice"), "voiceMethod": sv("GET"), "secure": bv(true)})
}

func TestSipDomainLC_Lifecycle(t *testing.T) {
	l := newTL(t, "SipDomain", "d")
	in := domainInputs()
	c := l.create(in)
	sid := c.ID
	require.True(t, strings.HasPrefix(sid, "SD"), sid)
	require.Equal(t, sid, getS(t, c.Properties, "sid"))
	require.Equal(t, testAccount, getS(t, c.Properties, "ownerAccountSid"))
	require.Equal(t, "2010-04-01", getS(t, c.Properties, "apiVersion"))
	stored := l.f.get("sipdomain", testAccount, sid)
	require.Equal(t, "acme.sip.twilio.com", stored["domain_name"])
	require.Equal(t, true, stored["secure"])
	require.Equal(t, "GET", stored["voice_method"])
	requireNoChanges(t, l.diff(sid, c.Properties, in, in))

	rd := l.read(sid, c.Properties, in)
	requireNoChanges(t, l.diff(sid, rd.Properties, in, rd.Inputs))
	// Drift in a managed setting is detected; an unmanaged one is not an input.
	l.f.mutate("sipdomain", testAccount, sid, func(o map[string]any) { o["secure"] = false; o["sip_registration"] = true; o["auth_type"] = "IP_ACL" })
	rd = l.read(sid, c.Properties, in)
	require.Equal(t, false, rd.Inputs.Get("secure").AsBool())
	require.False(t, hasProp(rd.Inputs, "sipRegistration"))
	require.Equal(t, "IP_ACL", getS(t, rd.Properties, "authType"))
	l.f.mutate("sipdomain", testAccount, sid, func(o map[string]any) { o["secure"] = true })
	rd = l.read(sid, c.Properties, in)

	// Everything updates in place, including the domain name.
	next := with(in, map[string]property.Value{
		"domainName": sv("acme-eu.sip.twilio.com"), "friendlyName": sv("acme eu"), "voiceUrl": sv("https://example.com/v2"),
		"voiceFallbackUrl": sv("https://example.com/fb"), "voiceFallbackMethod": sv("GET"),
		"voiceStatusCallbackUrl": sv("https://example.com/status"), "voiceStatusCallbackMethod": sv("POST"),
		"sipRegistration": bv(true), "emergencyCallingEnabled": bv(true), "secure": bv(false),
		"byocTrunkSid": sv("BY00000000000000000000000000000001"), "emergencyCallerSid": sv("PN00000000000000000000000000000001"),
	})
	d := l.diff(sid, rd.Properties, next, in)
	require.True(t, d.HasChanges)
	require.False(t, d.DeleteBeforeReplace)
	require.Equal(t, pgp.Update, d.DetailedDiff["domainName"].Kind)
	u := l.update(sid, rd.Properties, next, in)
	stored = l.f.get("sipdomain", testAccount, sid)
	for k, want := range map[string]any{
		"domain_name": "acme-eu.sip.twilio.com", "friendly_name": "acme eu", "voice_url": "https://example.com/v2",
		"voice_fallback_url": "https://example.com/fb", "voice_fallback_method": "GET", "voice_status_callback_url": "https://example.com/status",
		"voice_status_callback_method": "POST", "sip_registration": true, "emergency_calling_enabled": true, "secure": false,
		"byoc_trunk_sid": "BY00000000000000000000000000000001", "emergency_caller_sid": "PN00000000000000000000000000000001",
	} {
		require.Equal(t, want, stored[k], k)
	}
	requireNoChanges(t, l.diff(sid, u.Properties, next, next))

	// Removing inputs clears URLs and associations and resets methods to POST; the name is left alone.
	min := pv(map[string]property.Value{"domainName": sv("acme-eu.sip.twilio.com")})
	u2 := l.update(sid, u.Properties, min, next)
	stored = l.f.get("sipdomain", testAccount, sid)
	require.Nil(t, stored["voice_url"])
	require.Nil(t, stored["voice_fallback_url"])
	require.Nil(t, stored["byoc_trunk_sid"])
	require.Nil(t, stored["emergency_caller_sid"])
	require.Equal(t, "POST", stored["voice_fallback_method"])
	require.Equal(t, "acme eu", stored["friendly_name"])
	requireNoChanges(t, l.diff(sid, u2.Properties, min, min))

	// The account replaces.
	d = l.diff(sid, u2.Properties, with(min, map[string]property.Value{"accountSid": sv(fakeSub)}), min)
	require.Equal(t, pgp.AddReplace, d.DetailedDiff["accountSid"].Kind)

	l.f.remove("sipdomain", testAccount, sid)
	require.Empty(t, l.read(sid, u2.Properties, min).ID)
	l.del(sid, u2.Properties)
	c = l.create(in)
	l.del(c.ID, c.Properties)
	require.Nil(t, l.f.get("sipdomain", testAccount, c.ID))
}

func TestSipDomainLC_ImportPreviewAndChecks(t *testing.T) {
	l := newTL(t, "SipDomain", "d")
	l.f.addSubaccount(fakeSub)
	in := with(domainInputs(), map[string]property.Value{"accountSid": sv(fakeSub)})
	c := l.create(in)
	_, sid := splitScoped(c.ID)
	require.Equal(t, fakeSub+"/"+sid, c.ID)

	imp := l.read(c.ID, empty(), empty())
	require.Equal(t, fakeSub, getS(t, imp.Inputs, "accountSid"))
	require.Equal(t, "acme.sip.twilio.com", getS(t, imp.Inputs, "domainName"))
	require.Equal(t, "acme", getS(t, imp.Inputs, "friendlyName"))
	require.Equal(t, "GET", getS(t, imp.Inputs, "voiceMethod"))
	require.False(t, hasProp(imp.Inputs, "voiceFallbackMethod"), "the default POST is not adopted")
	own := l.other("SipDomain", "own").create(pv(map[string]property.Value{"domainName": sv("own.sip.twilio.com")}))
	imp = l.read(own.ID, empty(), empty())
	require.False(t, hasProp(imp.Inputs, "accountSid"))
	requireNoChanges(t, l.diff(own.ID, imp.Properties, pv(map[string]property.Value{"domainName": sv("own.sip.twilio.com")}), imp.Inputs))

	// Twilio treats the name case-insensitively: a differently cased program value is not drift.
	l.f.mutate("sipdomain", testAccount, own.ID, func(o map[string]any) { o["domain_name"] = "OWN.sip.twilio.com" })
	up := pv(map[string]property.Value{"domainName": sv("own.sip.twilio.com")})
	rd := l.read(own.ID, own.Properties, up)
	require.Equal(t, "own.sip.twilio.com", getS(t, rd.Inputs, "domainName"))

	// The API rejects a taken name.
	_, err := l.s.Create(pgp.CreateRequest{Urn: l.urn, Properties: l.checked(pv(map[string]property.Value{"domainName": sv("acme.sip.twilio.com")}))})
	require.ErrorContains(t, err, "already in use")

	before := l.f.sipCount("sipdomain")
	pr, err := l.s.Create(pgp.CreateRequest{Urn: l.urn, Properties: domainInputs(), DryRun: true})
	require.NoError(t, err)
	require.Empty(t, pr.ID)
	require.Equal(t, before, l.f.sipCount("sipdomain"))

	set := func(k string, v property.Value) property.Map {
		return with(domainInputs(), map[string]property.Value{k: v})
	}
	require.NotEmpty(t, l.check(set("domainName", sv("acme.example.com"))).Failures)
	require.NotEmpty(t, l.check(set("domainName", sv(".sip.twilio.com"))).Failures)
	require.NotEmpty(t, l.check(set("friendlyName", sv(strings.Repeat("n", 65)))).Failures)
	require.NotEmpty(t, l.check(set("voiceMethod", sv("PUT"))).Failures)
	require.NotEmpty(t, l.check(set("voiceFallbackMethod", sv("PUT"))).Failures)
	require.NotEmpty(t, l.check(set("voiceStatusCallbackMethod", sv("PUT"))).Failures)
	require.Empty(t, l.check(pv(map[string]property.Value{"domainName": computed(), "friendlyName": computed(), "voiceMethod": computed(),
		"voiceFallbackMethod": computed(), "voiceStatusCallbackMethod": computed(), "accountSid": computed()})).Failures)
}

// --------------------------------------------------------------------- lookups

func TestLookup_Sip(t *testing.T) {
	l := newTL(t, "SipCredentialList", "l")
	list := l.create(pv(map[string]property.Value{"friendlyName": sv("users")}))
	r, err := invoke(t, l, "getSipCredentialList", map[string]property.Value{"credentialListSid": sv(list.ID)})
	require.NoError(t, err)
	require.Equal(t, "users", getS(t, r, "friendlyName"))
	require.Equal(t, testAccount, getS(t, r, "ownerAccountSid"))

	cl := l.other("SipCredential", "c")
	cc := cl.create(credInputs(list.ID))
	_, _, csid, _ := splitChildID(cc.ID)
	r, err = invoke(t, l, "getSipCredential", map[string]property.Value{"credentialListSid": sv(list.ID), "credentialSid": sv(csid)})
	require.NoError(t, err)
	require.Equal(t, "alice", getS(t, r, "username"))
	require.Equal(t, list.ID, getS(t, r, "credentialListSid"))
	require.False(t, hasProp(r, "password"), "the password must not be returned")

	al := l.other("SipIpAccessControlList", "a")
	acl := al.create(pv(map[string]property.Value{"friendlyName": sv("offices")}))
	r, err = invoke(t, l, "getSipIpAccessControlList", map[string]property.Value{"ipAccessControlListSid": sv(acl.ID)})
	require.NoError(t, err)
	require.Equal(t, "offices", getS(t, r, "friendlyName"))

	ip := l.other("SipIpAddress", "ip")
	ic := ip.create(with(ipInputs(acl.ID), map[string]property.Value{"cidrPrefixLength": nv(30)}))
	_, _, isid, _ := splitChildID(ic.ID)
	r, err = invoke(t, l, "getSipIpAddress", map[string]property.Value{"ipAccessControlListSid": sv(acl.ID), "ipAddressSid": sv(isid)})
	require.NoError(t, err)
	require.Equal(t, "203.0.113.7", getS(t, r, "ipAddress"))
	require.Equal(t, 30.0, r.Get("cidrPrefixLength").AsNumber())

	dm := l.other("SipDomain", "d")
	dc := dm.create(domainInputs())
	r, err = invoke(t, l, "getSipDomain", map[string]property.Value{"domainSid": sv(dc.ID)})
	require.NoError(t, err)
	require.Equal(t, "acme.sip.twilio.com", getS(t, r, "domainName"))
	require.Equal(t, "GET", getS(t, r, "voiceMethod"))
	require.True(t, r.Get("secure").AsBool())
	require.Equal(t, testAccount, getS(t, r, "ownerAccountSid"))

	// Not found, and missing arguments.
	cl.del(cc.ID, cc.Properties)
	_, err = invoke(t, l, "getSipCredential", map[string]property.Value{"credentialListSid": sv(list.ID), "credentialSid": sv(csid)})
	require.ErrorContains(t, err, "was not found")
	ip.del(ic.ID, ic.Properties)
	_, err = invoke(t, l, "getSipIpAddress", map[string]property.Value{"ipAccessControlListSid": sv(acl.ID), "ipAddressSid": sv(isid)})
	require.ErrorContains(t, err, "was not found")
	al.del(acl.ID, acl.Properties)
	_, err = invoke(t, l, "getSipIpAccessControlList", map[string]property.Value{"ipAccessControlListSid": sv(acl.ID)})
	require.ErrorContains(t, err, "was not found")
	l.del(list.ID, list.Properties)
	_, err = invoke(t, l, "getSipCredentialList", map[string]property.Value{"credentialListSid": sv(list.ID)})
	require.ErrorContains(t, err, "was not found")
	dm.del(dc.ID, dc.Properties)
	_, err = invoke(t, l, "getSipDomain", map[string]property.Value{"domainSid": sv(dc.ID)})
	require.ErrorContains(t, err, "was not found")

	for tok, args := range map[string]map[string]property.Value{
		"getSipCredentialList":      {},
		"getSipCredential":          {"credentialListSid": sv("CL1")},
		"getSipIpAccessControlList": {},
		"getSipIpAddress":           {"ipAccessControlListSid": sv("AL1")},
		"getSipDomain":              {},
	} {
		_, err = invoke(t, l, tok, args)
		require.Error(t, err, tok)
	}
}

func TestSipChildIDs(t *testing.T) {
	a, p, s, err := splitChildID("CL1/CR2")
	require.NoError(t, err)
	require.Equal(t, []string{"", "CL1", "CR2"}, []string{a, p, s})
	a, p, s, err = splitChildID("AC9/CL1/CR2")
	require.NoError(t, err)
	require.Equal(t, []string{"AC9", "CL1", "CR2"}, []string{a, p, s})
	for _, bad := range []string{"CR2", "a/b/c/d", ""} {
		_, _, _, err = splitChildID(bad)
		require.Error(t, err, bad)
	}
}
