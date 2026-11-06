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

func trunkInputs() property.Map {
	return pv(map[string]property.Value{
		"friendlyName": sv("primary"), "domainName": sv("acme-prod.pstn.twilio.com"),
		"disasterRecoveryUrl": sv("https://example.com/dr"), "disasterRecoveryMethod": sv("GET"),
		"transferMode": sv("sip-only"), "transferCallerId": sv("from-transferor"),
		"secure": bv(true), "cnamLookupEnabled": bv(true),
		"recordingMode": sv("record-from-answer"), "recordingTrim": sv("trim-silence"),
	})
}

func TestTrunkLC_Lifecycle(t *testing.T) {
	l := newTL(t, "Trunk", "t")
	in := trunkInputs()
	c := l.create(in)
	sid := c.ID
	require.True(t, strings.HasPrefix(sid, "TK"), sid)
	require.Equal(t, sid, getS(t, c.Properties, "sid"))
	require.Equal(t, testAccount, getS(t, c.Properties, "accountSid"))
	form := l.f.callsMatching("POST", "/trunking/v1/Trunks")[0].Form
	require.Equal(t, "acme-prod.pstn.twilio.com", form.Get("DomainName"))
	require.Equal(t, "sip-only", form.Get("TransferMode"))
	require.Equal(t, "true", form.Get("Secure"))
	rec := l.f.callsMatching("POST", "/Recording")
	require.Len(t, rec, 1)
	require.Equal(t, "record-from-answer", rec[0].Form.Get("Mode"))
	require.Equal(t, "trim-silence", rec[0].Form.Get("Trim"))
	eff := effective(t, c.Properties)
	require.Equal(t, "record-from-answer", getS(t, eff, "recordingMode"))
	require.Equal(t, "acme-prod.pstn.twilio.com", getS(t, eff, "domainName"))
	requireNoChanges(t, l.diff(sid, c.Properties, in, in))

	// A refresh changes nothing; an out-of-band change is drift on the inputs the program set.
	rd := l.read(sid, c.Properties, in)
	requireNoChanges(t, l.diff(sid, rd.Properties, in, rd.Inputs))
	l.f.mutate("trunk", testAccount, sid, func(o map[string]any) { o["secure"] = false; o["_rec_mode"] = "do-not-record" })
	rd = l.read(sid, c.Properties, in)
	require.False(t, rd.Inputs.Get("secure").AsBool())
	require.Equal(t, "do-not-record", getS(t, rd.Inputs, "recordingMode"))
	d := l.diff(sid, rd.Properties, in, rd.Inputs)
	require.True(t, d.HasChanges)
	require.Equal(t, pgp.Update, d.DetailedDiff["secure"].Kind)
	require.Equal(t, pgp.Update, d.DetailedDiff["recordingMode"].Kind)
	require.Len(t, d.DetailedDiff, 2)

	// Updating applies the trunk fields and the recording; none of it replaces.
	next := with(in, map[string]property.Value{"friendlyName": sv("primary v2"), "recordingMode": sv("record-from-ringing")})
	d = l.diff(sid, c.Properties, next, in)
	require.True(t, d.HasChanges)
	require.Equal(t, pgp.Update, d.DetailedDiff["friendlyName"].Kind)
	require.Equal(t, pgp.Update, d.DetailedDiff["recordingMode"].Kind)
	require.Len(t, d.DetailedDiff, 2)
	l.f.resetCalls()
	u := l.update(sid, c.Properties, next, in)
	require.Equal(t, "primary v2", l.f.get("trunk", testAccount, sid)["friendly_name"])
	require.Equal(t, "record-from-ringing", l.f.get("trunk", testAccount, sid)["_rec_mode"])
	require.Equal(t, "2026-11-07T00:00:00Z", getS(t, u.Properties, "dateUpdated"))
	requireNoChanges(t, l.diff(sid, u.Properties, next, next))

	// Removing the recovery URL clears it; removing a plain setting leaves it alone.
	gone := without(next, "disasterRecoveryUrl", "transferMode")
	d = l.diff(sid, u.Properties, gone, next)
	require.Equal(t, pgp.Update, d.DetailedDiff["disasterRecoveryUrl"].Kind)
	require.Len(t, d.DetailedDiff, 1)
	l.f.resetCalls()
	u = l.update(sid, u.Properties, gone, next)
	require.Equal(t, "", l.f.get("trunk", testAccount, sid)["disaster_recovery_url"])
	require.Equal(t, "sip-only", l.f.get("trunk", testAccount, sid)["transfer_mode"])
	require.Equal(t, []string{""}, l.f.callsMatching("POST", "/trunking/v1/Trunks/"+sid)[0].Form["DisasterRecoveryUrl"])

	// Gone out of band: a refresh drops it, and deleting twice is fine.
	l.f.remove("trunk", testAccount, sid)
	require.Empty(t, l.read(sid, u.Properties, gone).ID)
	l.del(sid, u.Properties)
	c = l.create(in)
	l.del(c.ID, c.Properties)
	require.Nil(t, l.f.get("trunk", testAccount, c.ID))
}

func TestTrunkLC_EmptyInputsAndDefaults(t *testing.T) {
	l := newTL(t, "Trunk", "t")
	c := l.create(empty())
	require.Empty(t, l.f.callsMatching("POST", "/Recording"), "no recording call without recording inputs")
	require.Equal(t, "disable-all", getS(t, effective(t, c.Properties), "transferMode"))
	require.Equal(t, "do-not-record", getS(t, effective(t, c.Properties), "recordingMode"))
	requireNoChanges(t, l.diff(c.ID, c.Properties, empty(), empty()))
	// Stating a Twilio default is not a change.
	stated := pv(map[string]property.Value{"transferMode": sv("disable-all"), "secure": bv(false), "recordingMode": sv("do-not-record"),
		"recordingTrim": sv("do-not-trim"), "disasterRecoveryMethod": sv("POST")})
	requireNoChanges(t, l.diff(c.ID, c.Properties, stated, empty()))
	l.update(c.ID, c.Properties, stated, empty())
}

func TestTrunkLC_ImportThenDiff(t *testing.T) {
	l := newTL(t, "Trunk", "t")
	// Build the trunk behind the provider's back with explicit values and one default left alone.
	seed := l.other("Trunk", "seed").create(pv(map[string]property.Value{
		"friendlyName": sv("imported"), "domainName": sv("Legacy.pstn.twilio.com"), "disasterRecoveryUrl": sv("https://example.com/dr"),
		"secure": bv(true), "recordingMode": sv("record-from-answer"),
	}))
	rd := l.read(seed.ID, empty(), empty())
	require.Equal(t, seed.ID, rd.ID)
	// Exactly the non-default settings are adopted.
	got := map[string]bool{}
	for k := range rd.Inputs.AsMap() {
		got[k] = true
	}
	require.Equal(t, map[string]bool{"friendlyName": true, "domainName": true, "disasterRecoveryUrl": true, "secure": true, "recordingMode": true}, got)
	require.True(t, rd.Inputs.Get("secure").AsBool())
	require.Equal(t, "record-from-answer", getS(t, rd.Inputs, "recordingMode"))

	// A program that spells out the same values, defaults included, sees no change.
	prog := pv(map[string]property.Value{
		"friendlyName": sv("imported"), "domainName": sv("legacy.pstn.twilio.com"), "disasterRecoveryUrl": sv("https://example.com/dr"),
		"disasterRecoveryMethod": sv("POST"), "transferMode": sv("disable-all"), "transferCallerId": sv("from-transferee"),
		"secure": bv(true), "cnamLookupEnabled": bv(false), "recordingMode": sv("record-from-answer"), "recordingTrim": sv("do-not-trim"),
	})
	requireNoChanges(t, l.diff(seed.ID, rd.Properties, prog, rd.Inputs))
	// A program with a different value is a real, in-place change on exactly that key.
	d := l.diff(seed.ID, rd.Properties, with(prog, map[string]property.Value{"cnamLookupEnabled": bv(true), "recordingTrim": sv("trim-silence")}), rd.Inputs)
	require.True(t, d.HasChanges)
	require.Len(t, d.DetailedDiff, 2)
	require.Equal(t, pgp.Update, d.DetailedDiff["cnamLookupEnabled"].Kind)
	require.Equal(t, pgp.Update, d.DetailedDiff["recordingTrim"].Kind)
}

func TestTrunkLC_PartialFailuresAreRetrySafe(t *testing.T) {
	l := newTL(t, "Trunk", "t")
	// The trunk exists but its recording could not be set: the partial state is saved.
	l.f.failNext("POST", "/Recording", 500)
	resp, err := l.s.Create(pgp.CreateRequest{Urn: l.urn, Properties: l.checked(trunkInputs())})
	require.Error(t, err)
	require.NotNil(t, resp.PartialState)
	require.True(t, strings.HasPrefix(resp.ID, "TK"))
	require.Contains(t, resp.PartialState.Reasons[0], "set recording on trunk")

	// An update that fails on the recording leaves the old state, and repeating it succeeds.
	c := l.create(trunkInputs())
	next := with(trunkInputs(), map[string]property.Value{"friendlyName": sv("renamed"), "recordingMode": sv("do-not-record")})
	l.f.failNext("POST", "/Recording", 500)
	_, err = l.s.Update(pgp.UpdateRequest{ID: c.ID, Urn: l.urn, State: c.Properties, Inputs: l.checked(next), OldInputs: trunkInputs()})
	require.Error(t, err)
	u := l.update(c.ID, c.Properties, next, trunkInputs())
	require.Equal(t, "renamed", l.f.get("trunk", testAccount, c.ID)["friendly_name"])
	require.Equal(t, "do-not-record", l.f.get("trunk", testAccount, c.ID)["_rec_mode"])
	requireNoChanges(t, l.diff(c.ID, u.Properties, next, next))
}

func TestTrunkLC_Check(t *testing.T) {
	l := newTL(t, "Trunk", "t")
	for name, in := range map[string]property.Map{
		"domainName":             pv(map[string]property.Value{"domainName": sv("acme.sip.twilio.com")}),
		"friendlyName":           pv(map[string]property.Value{"friendlyName": sv("")}),
		"disasterRecoveryMethod": pv(map[string]property.Value{"disasterRecoveryMethod": sv("PUT")}),
		"transferMode":           pv(map[string]property.Value{"transferMode": sv("everything")}),
		"transferCallerId":       pv(map[string]property.Value{"transferCallerId": sv("nobody")}),
		"recordingMode":          pv(map[string]property.Value{"recordingMode": sv("always")}),
		"recordingTrim":          pv(map[string]property.Value{"recordingTrim": sv("shave")}),
	} {
		chk := l.check(in)
		require.Len(t, chk.Failures, 1, name)
		require.Equal(t, name, chk.Failures[0].Property)
	}
	// Preview-time unknowns are never validated.
	require.Empty(t, l.check(pv(map[string]property.Value{"domainName": computed(), "transferMode": computed(), "recordingMode": computed()})).Failures)
}

func originationInputs(trunk string) property.Map {
	return pv(map[string]property.Value{"trunkSid": sv(trunk), "friendlyName": sv("pbx"), "sipUrl": sv("sip:pbx.example.com:5060;transport=tls"),
		"weight": nv(20), "priority": nv(5), "enabled": bv(false)})
}

func TestTrunkOriginationUrlLC_Lifecycle(t *testing.T) {
	l := newTL(t, "TrunkOriginationUrl", "ou")
	tr := l.other("Trunk", "t").create(empty())
	in := originationInputs(tr.ID)
	c := l.create(in)
	sid := getS(t, c.Properties, "sid")
	require.True(t, strings.HasPrefix(sid, "OU"), sid)
	require.Equal(t, tr.ID+"/"+sid, c.ID)
	form := l.f.callsMatching("POST", "/OriginationUrls")[0].Form
	require.Equal(t, "20", form.Get("Weight"))
	require.Equal(t, "5", form.Get("Priority"))
	require.Equal(t, "false", form.Get("Enabled"))
	requireNoChanges(t, l.diff(c.ID, c.Properties, in, in))

	rd := l.read(c.ID, c.Properties, in)
	requireNoChanges(t, l.diff(c.ID, rd.Properties, in, rd.Inputs))
	l.f.mutate("trunkou", testAccount, sid, func(o map[string]any) { o["weight"] = float64(99) })
	rd = l.read(c.ID, c.Properties, in)
	require.Equal(t, float64(99), rd.Inputs.Get("weight").AsNumber())
	d := l.diff(c.ID, rd.Properties, in, rd.Inputs)
	require.Equal(t, pgp.Update, d.DetailedDiff["weight"].Kind)
	require.Len(t, d.DetailedDiff, 1)

	// Everything but the trunk updates in place.
	next := with(in, map[string]property.Value{"friendlyName": sv("pbx2"), "sipUrl": sv("sip:other.example.com"), "weight": nv(30), "priority": nv(1), "enabled": bv(true)})
	d = l.diff(c.ID, c.Properties, next, in)
	require.Len(t, d.DetailedDiff, 5)
	for k, v := range d.DetailedDiff {
		require.Equal(t, pgp.Update, v.Kind, k)
	}
	u := l.update(c.ID, c.Properties, next, in)
	require.Equal(t, "sip:other.example.com", l.f.get("trunkou", testAccount, sid)["sip_url"])
	require.Equal(t, true, l.f.get("trunkou", testAccount, sid)["enabled"])
	requireNoChanges(t, l.diff(c.ID, u.Properties, next, next))
	other := l.other("Trunk", "t2").create(empty())
	d = l.diff(c.ID, u.Properties, with(next, map[string]property.Value{"trunkSid": sv(other.ID)}), next)
	require.Equal(t, pgp.UpdateReplace, d.DetailedDiff["trunkSid"].Kind)

	// Gone out of band, or with its trunk.
	l.f.remove("trunkou", testAccount, sid)
	require.Empty(t, l.read(c.ID, u.Properties, next).ID)
	l.del(c.ID, u.Properties)
	c = l.create(in)
	l.other("Trunk", "t").del(tr.ID, tr.Properties)
	require.Empty(t, l.read(c.ID, c.Properties, in).ID)
	_, err := l.s.Read(pgp.ReadRequest{ID: "bad", Urn: l.urn, Properties: empty(), Inputs: empty()})
	require.ErrorContains(t, err, "expected <trunkSid>/<sid>")
}

func TestTrunkOriginationUrlLC_DefaultsAndImport(t *testing.T) {
	l := newTL(t, "TrunkOriginationUrl", "ou")
	tr := l.other("Trunk", "t").create(empty())
	min := pv(map[string]property.Value{"trunkSid": sv(tr.ID), "friendlyName": sv("pbx"), "sipUrl": sv("sip:pbx.example.com")})
	c := l.create(min)
	form := l.f.callsMatching("POST", "/OriginationUrls")[0].Form
	require.Equal(t, "10", form.Get("Weight"), "Twilio requires every field, so defaults are sent")
	require.Equal(t, "10", form.Get("Priority"))
	require.Equal(t, "true", form.Get("Enabled"))

	// Import adopts what Twilio reports; stating the defaults afterwards is no change.
	rd := l.read(c.ID, empty(), empty())
	require.Equal(t, tr.ID, getS(t, rd.Inputs, "trunkSid"))
	require.Equal(t, "sip:pbx.example.com", getS(t, rd.Inputs, "sipUrl"))
	stated := with(min, map[string]property.Value{"weight": nv(10), "priority": nv(10), "enabled": bv(true)})
	requireNoChanges(t, l.diff(c.ID, rd.Properties, stated, rd.Inputs))
	requireNoChanges(t, l.diff(c.ID, rd.Properties, min, rd.Inputs))
	explicit := with(stated, map[string]property.Value{"weight": nv(11)})
	d := l.diff(c.ID, rd.Properties, explicit, rd.Inputs)
	require.True(t, d.HasChanges)
	require.Len(t, d.DetailedDiff, 1)
	require.Equal(t, pgp.Update, d.DetailedDiff["weight"].Kind)

	for name, in := range map[string]property.Map{
		"sipUrl":       with(min, map[string]property.Value{"sipUrl": sv("sips:pbx.example.com")}),
		"friendlyName": with(min, map[string]property.Value{"friendlyName": sv("")}),
		"weight":       with(min, map[string]property.Value{"weight": nv(0)}),
		"priority":     with(min, map[string]property.Value{"priority": nv(70000)}),
	} {
		chk := l.check(in)
		require.Len(t, chk.Failures, 1, name)
		require.Equal(t, name, chk.Failures[0].Property)
	}
	require.Empty(t, l.check(with(min, map[string]property.Value{"sipUrl": computed(), "weight": computed()})).Failures)
}

func TestTrunkAttachmentsLC(t *testing.T) {
	for _, tc := range []struct{ typ, param, listType, listInputKey, kind, auth string }{
		{"TrunkCredentialList", "credentialListSid", "SipCredentialList", "friendlyName", "trunkcl", "CREDENTIAL_LIST"},
		{"TrunkIpAccessControlList", "ipAccessControlListSid", "SipIpAccessControlList", "friendlyName", "trunkacl", "IP_ACL"},
	} {
		t.Run(tc.typ, func(t *testing.T) {
			l := newTL(t, tc.typ, "a")
			tr := l.other("Trunk", "t").create(empty())
			list := l.other(tc.listType, "list").create(pv(map[string]property.Value{"friendlyName": sv("lan")}))
			in := pv(map[string]property.Value{"trunkSid": sv(tr.ID), tc.param: sv(list.ID)})
			c := l.create(in)
			require.Equal(t, tr.ID+"/"+list.ID, c.ID)
			require.Equal(t, list.ID, getS(t, c.Properties, "sid"))
			require.Equal(t, "lan", getS(t, c.Properties, "friendlyName"))
			require.Equal(t, tc.auth, l.f.get("trunk", testAccount, tr.ID)["auth_type"])
			requireNoChanges(t, l.diff(c.ID, c.Properties, in, in))
			require.Equal(t, tr.ID, getS(t, l.read(c.ID, empty(), empty()).Inputs, "trunkSid"), "import reads the IDs")
			rd := l.read(c.ID, c.Properties, in)
			require.Equal(t, list.ID, getS(t, rd.Inputs, tc.param))
			requireNoChanges(t, l.diff(c.ID, rd.Properties, in, rd.Inputs))

			// Every input replaces.
			other := l.other(tc.listType, "other").create(pv(map[string]property.Value{"friendlyName": sv("wan")}))
			d := l.diff(c.ID, c.Properties, with(in, map[string]property.Value{tc.param: sv(other.ID)}), in)
			require.Equal(t, pgp.UpdateReplace, d.DetailedDiff[tc.param].Kind)
			d = l.diff(c.ID, c.Properties, with(in, map[string]property.Value{"trunkSid": sv("TKother")}), in)
			require.Equal(t, pgp.UpdateReplace, d.DetailedDiff["trunkSid"].Kind)

			// Detaching leaves the list; a missing attachment is dropped on refresh.
			l.del(c.ID, c.Properties)
			require.Equal(t, "", l.f.get("trunk", testAccount, tr.ID)["auth_type"])
			require.NotNil(t, l.f.get(map[string]string{"trunkcl": "siplist", "trunkacl": "sipacl"}[tc.kind], testAccount, list.ID))
			require.Empty(t, l.read(c.ID, c.Properties, in).ID)
			l.del(c.ID, c.Properties)
			_, err := l.s.Create(pgp.CreateRequest{Urn: l.urn, Properties: l.checked(with(in, map[string]property.Value{tc.param: sv("XXmissing")}))})
			require.Error(t, err)
		})
	}
}

func TestLookup_Trunk(t *testing.T) {
	l := newTL(t, "Trunk", "t")
	tr := l.create(trunkInputs())
	r, err := invoke(t, l, "getTrunk", map[string]property.Value{"trunkSid": sv(tr.ID)})
	require.NoError(t, err)
	require.Equal(t, tr.ID, getS(t, r, "sid"))
	require.Equal(t, "acme-prod.pstn.twilio.com", getS(t, r, "domainName"))
	require.Equal(t, "record-from-answer", getS(t, r, "recordingMode"))
	require.Equal(t, "sip-only", getS(t, r, "transferMode"))
	require.True(t, r.Get("secure").AsBool())
	require.Equal(t, testAccount, getS(t, r, "accountSid"))
	_, err = invoke(t, l, "getTrunk", map[string]property.Value{"trunkSid": sv("TKmissing")})
	require.ErrorContains(t, err, "was not found")
	_, err = invoke(t, l, "getTrunk", map[string]property.Value{"trunkSid": sv("")})
	require.ErrorContains(t, err, "must be set")

	ou := l.other("TrunkOriginationUrl", "ou").create(originationInputs(tr.ID))
	r, err = invoke(t, l, "getTrunkOriginationUrl", map[string]property.Value{"trunkSid": sv(tr.ID), "originationUrlSid": sv(getS(t, ou.Properties, "sid"))})
	require.NoError(t, err)
	require.Equal(t, "sip:pbx.example.com:5060;transport=tls", getS(t, r, "sipUrl"))
	require.Equal(t, float64(20), r.Get("weight").AsNumber())
	require.False(t, r.Get("enabled").AsBool())
	_, err = invoke(t, l, "getTrunkOriginationUrl", map[string]property.Value{"trunkSid": sv(tr.ID), "originationUrlSid": sv("OUnope")})
	require.ErrorContains(t, err, "was not found")

	cl := l.other("SipCredentialList", "cl").create(pv(map[string]property.Value{"friendlyName": sv("users")}))
	acl := l.other("SipIpAccessControlList", "acl").create(pv(map[string]property.Value{"friendlyName": sv("offices")}))
	l.other("TrunkCredentialList", "a").create(pv(map[string]property.Value{"trunkSid": sv(tr.ID), "credentialListSid": sv(cl.ID)}))
	l.other("TrunkIpAccessControlList", "b").create(pv(map[string]property.Value{"trunkSid": sv(tr.ID), "ipAccessControlListSid": sv(acl.ID)}))
	r, err = invoke(t, l, "getTrunkCredentialList", map[string]property.Value{"trunkSid": sv(tr.ID), "credentialListSid": sv(cl.ID)})
	require.NoError(t, err)
	require.Equal(t, "users", getS(t, r, "friendlyName"))
	r, err = invoke(t, l, "getTrunkIpAccessControlList", map[string]property.Value{"trunkSid": sv(tr.ID), "ipAccessControlListSid": sv(acl.ID)})
	require.NoError(t, err)
	require.Equal(t, "offices", getS(t, r, "friendlyName"))
	_, err = invoke(t, l, "getTrunkIpAccessControlList", map[string]property.Value{"trunkSid": sv(tr.ID), "ipAccessControlListSid": sv("ALnope")})
	require.ErrorContains(t, err, "was not found")
	r, err = invoke(t, l, "getTrunk", map[string]property.Value{"trunkSid": sv(tr.ID)})
	require.NoError(t, err)
	require.Equal(t, "IP_ACL,CREDENTIAL_LIST", getS(t, r, "authType"))
}
