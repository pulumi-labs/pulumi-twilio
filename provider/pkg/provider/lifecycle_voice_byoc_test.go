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

func byocInputs() property.Map {
	return pv(map[string]property.Value{
		"friendlyName": sv("carrier"), "voiceUrl": sv("https://example.com/voice"), "voiceMethod": sv("GET"),
		"voiceFallbackUrl": sv("https://example.com/fallback"), "voiceFallbackMethod": sv("GET"),
		"statusCallbackUrl": sv("https://example.com/status"), "statusCallbackMethod": sv("GET"),
		"cnamLookupEnabled": bv(true), "connectionPolicySid": sv("NYpolicy"), "fromDomainSid": sv("SDdomain"),
	})
}

func TestByocTrunkLC_Lifecycle(t *testing.T) {
	l := newTL(t, "ByocTrunk", "b")
	in := byocInputs()
	c := l.create(in)
	sid := c.ID
	require.True(t, strings.HasPrefix(sid, "BY"), sid)
	require.Equal(t, sid, getS(t, c.Properties, "sid"))
	require.Equal(t, testAccount, getS(t, c.Properties, "accountSid"))
	form := l.f.callsMatching("POST", "/voice/v1/ByocTrunks")[0].Form
	require.Equal(t, "https://example.com/voice", form.Get("VoiceUrl"))
	require.Equal(t, "GET", form.Get("VoiceMethod"))
	require.Equal(t, "true", form.Get("CnamLookupEnabled"))
	require.Equal(t, "NYpolicy", form.Get("ConnectionPolicySid"))
	require.Equal(t, "SDdomain", form.Get("FromDomainSid"))
	requireNoChanges(t, l.diff(sid, c.Properties, in, in))

	// A refresh changes nothing; an out-of-band change is drift on the inputs the program set.
	rd := l.read(sid, c.Properties, in)
	requireNoChanges(t, l.diff(sid, rd.Properties, in, rd.Inputs))
	l.f.mutate("byoc", testAccount, sid, func(o map[string]any) { o["cnam_lookup_enabled"] = false; o["voice_url"] = nil })
	rd = l.read(sid, c.Properties, in)
	require.False(t, rd.Inputs.Get("cnamLookupEnabled").AsBool())
	d := l.diff(sid, rd.Properties, in, rd.Inputs)
	require.Equal(t, pgp.Update, d.DetailedDiff["cnamLookupEnabled"].Kind)
	require.Equal(t, pgp.Update, d.DetailedDiff["voiceUrl"].Kind)
	require.Len(t, d.DetailedDiff, 2)

	// Updating applies in place; nothing replaces.
	next := with(in, map[string]property.Value{"friendlyName": sv("carrier v2"), "voiceMethod": sv("POST"), "connectionPolicySid": sv("NYother")})
	d = l.diff(sid, c.Properties, next, in)
	require.Len(t, d.DetailedDiff, 3)
	for k, v := range d.DetailedDiff {
		require.Equal(t, pgp.Update, v.Kind, k)
	}
	l.f.resetCalls()
	u := l.update(sid, c.Properties, next, in)
	require.Equal(t, "carrier v2", l.f.get("byoc", testAccount, sid)["friendly_name"])
	require.Equal(t, "NYother", l.f.get("byoc", testAccount, sid)["connection_policy_sid"])
	require.Equal(t, "2026-11-07T00:00:00Z", getS(t, u.Properties, "dateUpdated"))
	requireNoChanges(t, l.diff(sid, u.Properties, next, next))

	// Removing a URL or SID clears it; removing a plain setting leaves it alone.
	gone := without(next, "voiceFallbackUrl", "fromDomainSid", "voiceMethod")
	d = l.diff(sid, u.Properties, gone, next)
	require.Len(t, d.DetailedDiff, 2)
	require.Equal(t, pgp.Update, d.DetailedDiff["voiceFallbackUrl"].Kind)
	require.Equal(t, pgp.Update, d.DetailedDiff["fromDomainSid"].Kind)
	l.f.resetCalls()
	l.update(sid, u.Properties, gone, next)
	posted := l.f.callsMatching("POST", "/voice/v1/ByocTrunks/"+sid)[0].Form
	require.Equal(t, []string{""}, posted["VoiceFallbackUrl"])
	require.Equal(t, []string{""}, posted["FromDomainSid"])
	require.NotContains(t, posted, "VoiceMethod")

	// Gone out of band: a refresh drops it, and deleting twice is fine.
	l.f.remove("byoc", testAccount, sid)
	require.Empty(t, l.read(sid, u.Properties, gone).ID)
	l.del(sid, u.Properties)
	c = l.create(in)
	l.del(c.ID, c.Properties)
	require.Nil(t, l.f.get("byoc", testAccount, c.ID))
}

func TestByocTrunkLC_EmptyDefaultsAndImport(t *testing.T) {
	l := newTL(t, "ByocTrunk", "b")
	c := l.create(empty())
	requireNoChanges(t, l.diff(c.ID, c.Properties, empty(), empty()))
	// Stating a Twilio default is not a change.
	stated := pv(map[string]property.Value{"voiceMethod": sv("POST"), "voiceFallbackMethod": sv("POST"), "statusCallbackMethod": sv("POST"), "cnamLookupEnabled": bv(false)})
	d := l.diff(c.ID, c.Properties, stated, empty())
	require.Len(t, d.DetailedDiff, 0, "defaults equal nothing recorded except the cnam flag, which is already false")

	seed := l.other("ByocTrunk", "seed").create(pv(map[string]property.Value{
		"friendlyName": sv("imported"), "voiceUrl": sv("https://example.com/v"), "cnamLookupEnabled": bv(true), "connectionPolicySid": sv("NY1"),
	}))
	rd := l.read(seed.ID, empty(), empty())
	require.Equal(t, seed.ID, rd.ID)
	got := map[string]bool{}
	for k := range rd.Inputs.AsMap() {
		got[string(k)] = true
	}
	require.Equal(t, map[string]bool{"friendlyName": true, "voiceUrl": true, "cnamLookupEnabled": true, "connectionPolicySid": true}, got)
	prog := pv(map[string]property.Value{
		"friendlyName": sv("imported"), "voiceUrl": sv("https://example.com/v"), "voiceMethod": sv("POST"),
		"cnamLookupEnabled": bv(true), "connectionPolicySid": sv("NY1"),
	})
	requireNoChanges(t, l.diff(seed.ID, rd.Properties, prog, rd.Inputs))
	d = l.diff(seed.ID, rd.Properties, with(prog, map[string]property.Value{"voiceMethod": sv("GET")}), rd.Inputs)
	require.Len(t, d.DetailedDiff, 1)
	require.Equal(t, pgp.Update, d.DetailedDiff["voiceMethod"].Kind)
}

func TestByocTrunkLC_Check(t *testing.T) {
	l := newTL(t, "ByocTrunk", "b")
	for name, in := range map[string]property.Map{
		"friendlyName":         pv(map[string]property.Value{"friendlyName": sv("")}),
		"voiceMethod":          pv(map[string]property.Value{"voiceMethod": sv("PUT")}),
		"voiceFallbackMethod":  pv(map[string]property.Value{"voiceFallbackMethod": sv("DELETE")}),
		"statusCallbackMethod": pv(map[string]property.Value{"statusCallbackMethod": sv("HEAD")}),
		"connectionPolicySid":  pv(map[string]property.Value{"connectionPolicySid": sv("SD1")}),
		"fromDomainSid":        pv(map[string]property.Value{"fromDomainSid": sv("NY1")}),
	} {
		chk := l.check(in)
		require.Len(t, chk.Failures, 1, name)
		require.Equal(t, name, chk.Failures[0].Property)
	}
	require.Empty(t, l.check(pv(map[string]property.Value{"voiceMethod": computed(), "connectionPolicySid": computed()})).Failures)
}

func TestConnectionPolicyLC_Lifecycle(t *testing.T) {
	l := newTL(t, "ConnectionPolicy", "p")
	in := pv(map[string]property.Value{"friendlyName": sv("primary")})
	c := l.create(in)
	require.True(t, strings.HasPrefix(c.ID, "NY"), c.ID)
	require.Equal(t, "primary", l.f.callsMatching("POST", "/voice/v1/ConnectionPolicies")[0].Form.Get("FriendlyName"))
	requireNoChanges(t, l.diff(c.ID, c.Properties, in, in))
	rd := l.read(c.ID, c.Properties, in)
	requireNoChanges(t, l.diff(c.ID, rd.Properties, in, rd.Inputs))

	l.f.mutate("connpol", testAccount, c.ID, func(o map[string]any) { o["friendly_name"] = "renamed" })
	rd = l.read(c.ID, c.Properties, in)
	d := l.diff(c.ID, rd.Properties, in, rd.Inputs)
	require.Equal(t, pgp.Update, d.DetailedDiff["friendlyName"].Kind)

	next := pv(map[string]property.Value{"friendlyName": sv("secondary")})
	require.Equal(t, pgp.Update, l.diff(c.ID, c.Properties, next, in).DetailedDiff["friendlyName"].Kind)
	u := l.update(c.ID, c.Properties, next, in)
	require.Equal(t, "secondary", l.f.get("connpol", testAccount, c.ID)["friendly_name"])
	require.Equal(t, "2026-11-07T00:00:00Z", getS(t, u.Properties, "dateUpdated"))
	requireNoChanges(t, l.diff(c.ID, u.Properties, next, next))
	requireNoChanges(t, l.diff(c.ID, u.Properties, empty(), next))

	// Import adopts the name.
	imp := l.read(c.ID, empty(), empty())
	require.Equal(t, "secondary", getS(t, imp.Inputs, "friendlyName"))

	l.f.remove("connpol", testAccount, c.ID)
	require.Empty(t, l.read(c.ID, u.Properties, next).ID)
	l.del(c.ID, u.Properties)
	require.Equal(t, "friendlyName", l.check(pv(map[string]property.Value{"friendlyName": sv("")})).Failures[0].Property)

	// A policy created without a name is fine.
	c = l.create(empty())
	requireNoChanges(t, l.diff(c.ID, c.Properties, empty(), empty()))
}

func targetInputs(policy string) property.Map {
	return pv(map[string]property.Value{"connectionPolicySid": sv(policy), "target": sv("sip:carrier.example.com:5060"),
		"friendlyName": sv("primary carrier"), "priority": nv(5), "weight": nv(20), "enabled": bv(false)})
}

func TestConnectionPolicyTargetLC_Lifecycle(t *testing.T) {
	l := newTL(t, "ConnectionPolicyTarget", "t")
	pol := l.other("ConnectionPolicy", "p").create(empty())
	in := targetInputs(pol.ID)
	c := l.create(in)
	sid := getS(t, c.Properties, "sid")
	require.True(t, strings.HasPrefix(sid, "NE"), sid)
	require.Equal(t, pol.ID+"/"+sid, c.ID)
	form := l.f.callsMatching("POST", "/Targets")[0].Form
	require.Equal(t, "sip:carrier.example.com:5060", form.Get("Target"))
	require.Equal(t, "5", form.Get("Priority"))
	require.Equal(t, "20", form.Get("Weight"))
	require.Equal(t, "false", form.Get("Enabled"))
	requireNoChanges(t, l.diff(c.ID, c.Properties, in, in))

	rd := l.read(c.ID, c.Properties, in)
	requireNoChanges(t, l.diff(c.ID, rd.Properties, in, rd.Inputs))
	l.f.mutate("polytarget", testAccount, sid, func(o map[string]any) { o["weight"] = float64(99) })
	rd = l.read(c.ID, c.Properties, in)
	d := l.diff(c.ID, rd.Properties, in, rd.Inputs)
	require.Equal(t, pgp.Update, d.DetailedDiff["weight"].Kind)
	require.Len(t, d.DetailedDiff, 1)

	// Everything but the policy updates in place.
	next := with(in, map[string]property.Value{"target": sv("sip:other.example.com"), "friendlyName": sv("backup"), "priority": nv(1), "weight": nv(30), "enabled": bv(true)})
	d = l.diff(c.ID, c.Properties, next, in)
	require.Len(t, d.DetailedDiff, 5)
	for k, v := range d.DetailedDiff {
		require.Equal(t, pgp.Update, v.Kind, k)
	}
	u := l.update(c.ID, c.Properties, next, in)
	require.Equal(t, "sip:other.example.com", l.f.get("polytarget", testAccount, sid)["target"])
	require.Equal(t, true, l.f.get("polytarget", testAccount, sid)["enabled"])
	requireNoChanges(t, l.diff(c.ID, u.Properties, next, next))
	other := l.other("ConnectionPolicy", "p2").create(empty())
	d = l.diff(c.ID, u.Properties, with(next, map[string]property.Value{"connectionPolicySid": sv(other.ID)}), next)
	require.Equal(t, pgp.UpdateReplace, d.DetailedDiff["connectionPolicySid"].Kind)

	// Removing priority, weight and enabled restores Twilio's defaults on the next update.
	min := pv(map[string]property.Value{"connectionPolicySid": sv(pol.ID), "target": sv("sip:other.example.com")})
	d = l.diff(c.ID, u.Properties, min, next)
	require.Len(t, d.DetailedDiff, 2, "priority and weight differ from the defaults; enabled is already true")
	l.f.resetCalls()
	l.update(c.ID, u.Properties, min, next)
	require.Equal(t, float64(10), l.f.get("polytarget", testAccount, sid)["priority"])
	require.Equal(t, float64(10), l.f.get("polytarget", testAccount, sid)["weight"])

	// Gone out of band, or with its policy.
	l.f.remove("polytarget", testAccount, sid)
	require.Empty(t, l.read(c.ID, u.Properties, min).ID)
	l.del(c.ID, u.Properties)
	c = l.create(in)
	l.other("ConnectionPolicy", "p").del(pol.ID, pol.Properties)
	require.Empty(t, l.read(c.ID, c.Properties, in).ID)
	_, err := l.s.Read(pgp.ReadRequest{ID: "bad", Urn: l.urn, Properties: empty(), Inputs: empty()})
	require.ErrorContains(t, err, "expected <connectionPolicySid>/<sid>")
}

func TestConnectionPolicyTargetLC_DefaultsImportAndCheck(t *testing.T) {
	l := newTL(t, "ConnectionPolicyTarget", "t")
	pol := l.other("ConnectionPolicy", "p").create(empty())
	min := pv(map[string]property.Value{"connectionPolicySid": sv(pol.ID), "target": sv("sip:carrier.example.com")})
	c := l.create(min)
	form := l.f.callsMatching("POST", "/Targets")[0].Form
	require.Equal(t, "10", form.Get("Priority"))
	require.Equal(t, "10", form.Get("Weight"))
	require.Equal(t, "true", form.Get("Enabled"))

	rd := l.read(c.ID, empty(), empty())
	require.Equal(t, pol.ID, getS(t, rd.Inputs, "connectionPolicySid"))
	require.Equal(t, "sip:carrier.example.com", getS(t, rd.Inputs, "target"))
	require.False(t, hasProp(rd.Inputs, "priority"), "defaults are not adopted on import")
	require.False(t, hasProp(rd.Inputs, "weight"))
	require.False(t, hasProp(rd.Inputs, "enabled"))
	stated := with(min, map[string]property.Value{"weight": nv(10), "priority": nv(10), "enabled": bv(true)})
	requireNoChanges(t, l.diff(c.ID, rd.Properties, stated, rd.Inputs))
	requireNoChanges(t, l.diff(c.ID, rd.Properties, min, rd.Inputs))
	d := l.diff(c.ID, rd.Properties, with(stated, map[string]property.Value{"weight": nv(11)}), rd.Inputs)
	require.Len(t, d.DetailedDiff, 1)
	require.Equal(t, pgp.Update, d.DetailedDiff["weight"].Kind)

	// A non-default target is adopted by an import.
	full := l.other("ConnectionPolicyTarget", "full").create(targetInputs(pol.ID))
	rd = l.read(full.ID, empty(), empty())
	require.Equal(t, float64(5), rd.Inputs.Get("priority").AsNumber())
	require.False(t, rd.Inputs.Get("enabled").AsBool())
	require.Equal(t, "primary carrier", getS(t, rd.Inputs, "friendlyName"))

	for name, in := range map[string]property.Map{
		"target":              with(min, map[string]property.Value{"target": sv("sips:carrier.example.com")}),
		"friendlyName":        with(min, map[string]property.Value{"friendlyName": sv("")}),
		"weight":              with(min, map[string]property.Value{"weight": nv(0)}),
		"priority":            with(min, map[string]property.Value{"priority": nv(70000)}),
		"connectionPolicySid": with(min, map[string]property.Value{"connectionPolicySid": sv("IL1")}),
	} {
		chk := l.check(in)
		require.Len(t, chk.Failures, 1, name)
		require.Equal(t, name, chk.Failures[0].Property)
	}
	require.Empty(t, l.check(with(min, map[string]property.Value{"target": computed(), "weight": computed()})).Failures)
}

func TestConnectionPolicyLC_DeleteRemovesTargets(t *testing.T) {
	l := newTL(t, "ConnectionPolicy", "p")
	pol := l.create(empty())
	tg := l.other("ConnectionPolicyTarget", "t").create(targetInputs(pol.ID))
	l.del(pol.ID, pol.Properties)
	require.Empty(t, l.other("ConnectionPolicyTarget", "t").read(tg.ID, tg.Properties, targetInputs(pol.ID)).ID)
}

func ipRecordInputs() property.Map {
	return pv(map[string]property.Value{"ipAddress": sv("203.0.113.10"), "cidrPrefixLength": nv(24), "friendlyName": sv("carrier edge")})
}

func TestIpRecordLC_Lifecycle(t *testing.T) {
	l := newTL(t, "IpRecord", "r")
	in := ipRecordInputs()
	c := l.create(in)
	require.True(t, strings.HasPrefix(c.ID, "IL"), c.ID)
	form := l.f.callsMatching("POST", "/voice/v1/IpRecords")[0].Form
	require.Equal(t, "203.0.113.10", form.Get("IpAddress"))
	require.Equal(t, "24", form.Get("CidrPrefixLength"))
	require.Equal(t, "carrier edge", form.Get("FriendlyName"))
	requireNoChanges(t, l.diff(c.ID, c.Properties, in, in))
	rd := l.read(c.ID, c.Properties, in)
	requireNoChanges(t, l.diff(c.ID, rd.Properties, in, rd.Inputs))

	// Only the name updates in place; the address and prefix replace.
	next := with(in, map[string]property.Value{"friendlyName": sv("renamed")})
	d := l.diff(c.ID, c.Properties, next, in)
	require.Equal(t, pgp.Update, d.DetailedDiff["friendlyName"].Kind)
	require.Len(t, d.DetailedDiff, 1)
	l.f.resetCalls()
	u := l.update(c.ID, c.Properties, next, in)
	require.Equal(t, "renamed", l.f.get("iprec", testAccount, c.ID)["friendly_name"])
	posted := l.f.callsMatching("POST", "/voice/v1/IpRecords/"+c.ID)[0].Form
	require.NotContains(t, posted, "IpAddress")
	requireNoChanges(t, l.diff(c.ID, u.Properties, next, next))
	d = l.diff(c.ID, u.Properties, with(next, map[string]property.Value{"ipAddress": sv("203.0.113.11")}), next)
	require.Equal(t, pgp.UpdateReplace, d.DetailedDiff["ipAddress"].Kind)
	d = l.diff(c.ID, u.Properties, with(next, map[string]property.Value{"cidrPrefixLength": nv(32)}), next)
	require.Equal(t, pgp.UpdateReplace, d.DetailedDiff["cidrPrefixLength"].Kind)

	l.f.remove("iprec", testAccount, c.ID)
	require.Empty(t, l.read(c.ID, u.Properties, next).ID)
	l.del(c.ID, u.Properties)
}

func TestIpRecordLC_DefaultsImportAndCheck(t *testing.T) {
	l := newTL(t, "IpRecord", "r")
	min := pv(map[string]property.Value{"ipAddress": sv("198.51.100.7")})
	c := l.create(min)
	require.NotContains(t, l.f.callsMatching("POST", "/voice/v1/IpRecords")[0].Form, "CidrPrefixLength")
	rd := l.read(c.ID, empty(), empty())
	require.Equal(t, "198.51.100.7", getS(t, rd.Inputs, "ipAddress"))
	require.False(t, hasProp(rd.Inputs, "cidrPrefixLength"), "the /32 default is not adopted")
	require.False(t, hasProp(rd.Inputs, "friendlyName"))
	stated := with(min, map[string]property.Value{"cidrPrefixLength": nv(32)})
	requireNoChanges(t, l.diff(c.ID, rd.Properties, stated, rd.Inputs))
	requireNoChanges(t, l.diff(c.ID, rd.Properties, min, rd.Inputs))

	full := l.other("IpRecord", "full").create(ipRecordInputs())
	rd = l.read(full.ID, empty(), empty())
	require.Equal(t, float64(24), rd.Inputs.Get("cidrPrefixLength").AsNumber())
	require.Equal(t, "carrier edge", getS(t, rd.Inputs, "friendlyName"))

	for name, in := range map[string]property.Map{
		"ipAddress":        pv(map[string]property.Value{"ipAddress": sv("2001:db8::1")}),
		"cidrPrefixLength": with(min, map[string]property.Value{"cidrPrefixLength": nv(33)}),
		"friendlyName":     with(min, map[string]property.Value{"friendlyName": sv("")}),
	} {
		chk := l.check(in)
		require.Len(t, chk.Failures, 1, name)
		require.Equal(t, name, chk.Failures[0].Property)
	}
	for _, bad := range []string{"not-an-ip", "", "203.0.113.256", "::ffff:203.0.113.1"} {
		chk := l.check(pv(map[string]property.Value{"ipAddress": sv(bad)}))
		require.Len(t, chk.Failures, 1, bad)
	}
	require.Empty(t, l.check(with(min, map[string]property.Value{"ipAddress": computed(), "cidrPrefixLength": computed()})).Failures)
	chk := l.check(with(min, map[string]property.Value{"cidrPrefixLength": nv(0)}))
	require.Len(t, chk.Failures, 1)
}

func TestSourceIpMappingLC_Lifecycle(t *testing.T) {
	l := newTL(t, "SourceIpMapping", "m")
	rec := l.other("IpRecord", "r").create(ipRecordInputs())
	in := pv(map[string]property.Value{"ipRecordSid": sv(rec.ID), "sipDomainSid": sv("SDone")})
	c := l.create(in)
	require.True(t, strings.HasPrefix(c.ID, "IB"), c.ID)
	form := l.f.callsMatching("POST", "/voice/v1/SourceIpMappings")[0].Form
	require.Equal(t, rec.ID, form.Get("IpRecordSid"))
	require.Equal(t, "SDone", form.Get("SipDomainSid"))
	requireNoChanges(t, l.diff(c.ID, c.Properties, in, in))

	// Import reads both references.
	imp := l.read(c.ID, empty(), empty())
	require.Equal(t, rec.ID, getS(t, imp.Inputs, "ipRecordSid"))
	require.Equal(t, "SDone", getS(t, imp.Inputs, "sipDomainSid"))
	requireNoChanges(t, l.diff(c.ID, imp.Properties, in, imp.Inputs))

	l.f.mutate("srcmap", testAccount, c.ID, func(o map[string]any) { o["sip_domain_sid"] = "SDdrift" })
	rd := l.read(c.ID, c.Properties, in)
	d := l.diff(c.ID, rd.Properties, in, rd.Inputs)
	require.Equal(t, pgp.Update, d.DetailedDiff["sipDomainSid"].Kind)
	require.Len(t, d.DetailedDiff, 1)

	// The domain updates in place; the record replaces.
	next := with(in, map[string]property.Value{"sipDomainSid": sv("SDtwo")})
	require.Equal(t, pgp.Update, l.diff(c.ID, c.Properties, next, in).DetailedDiff["sipDomainSid"].Kind)
	l.f.resetCalls()
	u := l.update(c.ID, c.Properties, next, in)
	require.Equal(t, "SDtwo", l.f.get("srcmap", testAccount, c.ID)["sip_domain_sid"])
	posted := l.f.callsMatching("POST", "/voice/v1/SourceIpMappings/"+c.ID)[0].Form
	require.NotContains(t, posted, "IpRecordSid")
	requireNoChanges(t, l.diff(c.ID, u.Properties, next, next))
	d = l.diff(c.ID, u.Properties, with(next, map[string]property.Value{"ipRecordSid": sv("ILother")}), next)
	require.Equal(t, pgp.UpdateReplace, d.DetailedDiff["ipRecordSid"].Kind)

	l.f.remove("srcmap", testAccount, c.ID)
	require.Empty(t, l.read(c.ID, u.Properties, next).ID)
	l.del(c.ID, u.Properties)

	for name, bad := range map[string]property.Map{
		"ipRecordSid":  pv(map[string]property.Value{"ipRecordSid": sv("SD1"), "sipDomainSid": sv("SD1")}),
		"sipDomainSid": pv(map[string]property.Value{"ipRecordSid": sv("IL1"), "sipDomainSid": sv("IL1")}),
	} {
		chk := l.check(bad)
		require.Len(t, chk.Failures, 1, name)
		require.Equal(t, name, chk.Failures[0].Property)
	}
}

func TestLookup_Byoc(t *testing.T) {
	l := newTL(t, "ByocTrunk", "b")
	notFound := func(fn, arg string) {
		_, err := invoke(t, l, fn, map[string]property.Value{arg: sv("XXnope")})
		require.ErrorContains(t, err, "was not found", fn)
		_, err = invoke(t, l, fn, map[string]property.Value{arg: sv("")})
		require.ErrorContains(t, err, "must be set", fn)
	}

	tr := l.create(byocInputs())
	r, err := invoke(t, l, "getByocTrunk", map[string]property.Value{"byocTrunkSid": sv(tr.ID)})
	require.NoError(t, err)
	require.Equal(t, tr.ID, getS(t, r, "sid"))
	require.Equal(t, "carrier", getS(t, r, "friendlyName"))
	require.Equal(t, "https://example.com/voice", getS(t, r, "voiceUrl"))
	require.Equal(t, "GET", getS(t, r, "voiceMethod"))
	require.Equal(t, "NYpolicy", getS(t, r, "connectionPolicySid"))
	require.True(t, r.Get("cnamLookupEnabled").AsBool())
	require.Equal(t, testAccount, getS(t, r, "accountSid"))
	notFound("getByocTrunk", "byocTrunkSid")

	bare := l.other("ByocTrunk", "bare").create(empty())
	r, err = invoke(t, l, "getByocTrunk", map[string]property.Value{"byocTrunkSid": sv(bare.ID)})
	require.NoError(t, err)
	require.Equal(t, "", getS(t, r, "voiceUrl"), "unset values read as empty")
	require.Equal(t, "POST", getS(t, r, "voiceMethod"))

	pol := l.other("ConnectionPolicy", "p").create(pv(map[string]property.Value{"friendlyName": sv("primary")}))
	r, err = invoke(t, l, "getConnectionPolicy", map[string]property.Value{"connectionPolicySid": sv(pol.ID)})
	require.NoError(t, err)
	require.Equal(t, "primary", getS(t, r, "friendlyName"))
	require.Equal(t, pol.ID, getS(t, r, "sid"))
	notFound("getConnectionPolicy", "connectionPolicySid")

	tg := l.other("ConnectionPolicyTarget", "t").create(targetInputs(pol.ID))
	r, err = invoke(t, l, "getConnectionPolicyTarget", map[string]property.Value{"connectionPolicySid": sv(pol.ID), "targetSid": sv(getS(t, tg.Properties, "sid"))})
	require.NoError(t, err)
	require.Equal(t, "sip:carrier.example.com:5060", getS(t, r, "target"))
	require.Equal(t, float64(5), r.Get("priority").AsNumber())
	require.Equal(t, float64(20), r.Get("weight").AsNumber())
	require.False(t, r.Get("enabled").AsBool())
	_, err = invoke(t, l, "getConnectionPolicyTarget", map[string]property.Value{"connectionPolicySid": sv(pol.ID), "targetSid": sv("NEnope")})
	require.ErrorContains(t, err, "was not found")
	_, err = invoke(t, l, "getConnectionPolicyTarget", map[string]property.Value{"connectionPolicySid": sv(""), "targetSid": sv("NE1")})
	require.ErrorContains(t, err, "must be set")

	rec := l.other("IpRecord", "r").create(ipRecordInputs())
	r, err = invoke(t, l, "getIpRecord", map[string]property.Value{"ipRecordSid": sv(rec.ID)})
	require.NoError(t, err)
	require.Equal(t, "203.0.113.10", getS(t, r, "ipAddress"))
	require.Equal(t, float64(24), r.Get("cidrPrefixLength").AsNumber())
	notFound("getIpRecord", "ipRecordSid")

	m := l.other("SourceIpMapping", "m").create(pv(map[string]property.Value{"ipRecordSid": sv(rec.ID), "sipDomainSid": sv("SDone")}))
	r, err = invoke(t, l, "getSourceIpMapping", map[string]property.Value{"sourceIpMappingSid": sv(m.ID)})
	require.NoError(t, err)
	require.Equal(t, rec.ID, getS(t, r, "ipRecordSid"))
	require.Equal(t, "SDone", getS(t, r, "sipDomainSid"))
	notFound("getSourceIpMapping", "sourceIpMappingSid")
}
