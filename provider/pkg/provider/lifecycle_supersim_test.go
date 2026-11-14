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

func seedSuperSimNetworks(l *tl) {
	l.f.addSuperSimNetwork(ssNetA, "Telco A", "US", [2]string{"310", "260"}, [2]string{"310", "410"})
	l.f.addSuperSimNetwork(ssNetB, "Telco B", "DE", [2]string{"262", "01"})
	l.f.addSuperSimNetwork(ssNetC, "Telco C", "US", [2]string{"311", "480"})
}

func profileNetworks(l *tl, id string) []string {
	out := []string{}
	for _, n := range []string{ssNetA, ssNetB, ssNetC} {
		if l.f.get("ss", testAccount, "NetworkAccessProfiles/"+id+"/Networks/"+n) != nil {
			out = append(out, n)
		}
	}
	return out
}

func TestSuperSimNetworkAccessProfileLC_Lifecycle(t *testing.T) {
	l := newTL(t, "SuperSimNetworkAccessProfile", "profile")
	seedSuperSimNetworks(l)
	in := pv(map[string]property.Value{"uniqueName": sv("iot-eu"), "networks": strList(ssNetA, ssNetB)})
	c := l.create(in)
	require.True(t, strings.HasPrefix(c.ID, "HA"), c.ID)
	require.Equal(t, c.ID, getS(t, c.Properties, "sid"))
	form := l.f.callsMatching("POST", "/supersim/v1/NetworkAccessProfiles")[0].Form
	require.Equal(t, "iot-eu", form.Get("UniqueName"))
	require.Equal(t, []string{ssNetA, ssNetB}, form["Networks"])
	require.Equal(t, []string{ssNetA, ssNetB}, profileNetworks(l, c.ID))
	requireNoChanges(t, l.diff(c.ID, c.Properties, in, in))

	// Twilio lists the networks in its own order; that is not drift. A network removed outside Pulumi is.
	rd := l.read(c.ID, c.Properties, in)
	requireNoChanges(t, l.diff(c.ID, rd.Properties, in, rd.Inputs))
	l.f.remove("ss", testAccount, "NetworkAccessProfiles/"+c.ID+"/Networks/"+ssNetB)
	rd = l.read(c.ID, c.Properties, in)
	require.Len(t, rd.Inputs.Get("networks").AsArray().AsSlice(), 1)
	require.Equal(t, pgp.Update, l.diff(c.ID, rd.Properties, in, rd.Inputs).DetailedDiff["networks"].Kind)

	// An update reconciles against what Twilio holds: it adds the missing and removes the extra ones.
	next := with(in, map[string]property.Value{"uniqueName": sv("iot-world"), "networks": strList(ssNetB, ssNetC)})
	d := l.diff(c.ID, c.Properties, next, in)
	require.Equal(t, pgp.Update, d.DetailedDiff["uniqueName"].Kind)
	require.Equal(t, pgp.Update, d.DetailedDiff["networks"].Kind)
	l.f.resetCalls()
	u := l.update(c.ID, rd.Properties, next, in)
	require.Equal(t, "iot-world", l.f.get("ss", testAccount, "NetworkAccessProfiles/"+c.ID)["unique_name"])
	require.Equal(t, []string{ssNetB, ssNetC}, profileNetworks(l, c.ID))
	require.Len(t, l.f.callsMatching("POST", "/Networks"), 2)
	require.Len(t, l.f.callsMatching("DELETE", "/Networks/"+ssNetA), 1)
	require.Len(t, l.f.callsMatching("DELETE", "/Networks/"+ssNetB), 0, "B was already gone from Twilio and is wanted")
	requireNoChanges(t, l.diff(c.ID, u.Properties, next, next))

	// Omitting networks leaves them alone; an empty list removes every network.
	noNets := without(next, "networks")
	requireNoChanges(t, l.diff(c.ID, u.Properties, noNets, next))
	cleared := with(next, map[string]property.Value{"networks": strList()})
	require.Equal(t, pgp.Update, l.diff(c.ID, u.Properties, cleared, next).DetailedDiff["networks"].Kind)
	u = l.update(c.ID, u.Properties, cleared, next)
	require.Empty(t, profileNetworks(l, c.ID))
	requireNoChanges(t, l.diff(c.ID, u.Properties, cleared, cleared))

	// Import adopts the name and the networks; the result passes Check.
	l.f.put("ss", testAccount, "NetworkAccessProfiles/"+c.ID+"/Networks/"+ssNetA, l.f.ssProfileNetwork(c.ID, l.f.get("ss", testAccount, "Networks/"+ssNetA)))
	imp := l.read(c.ID, empty(), empty())
	require.Empty(t, l.check(imp.Inputs).Failures)
	require.Equal(t, "iot-world", getS(t, imp.Inputs, "uniqueName"))
	require.Len(t, imp.Inputs.Get("networks").AsArray().AsSlice(), 1)
	l.requireImportFails("HA" + strings.Repeat("0", 32))

	// Twilio has no DELETE: destroy forgets the profile and does not call it.
	l.f.resetCalls()
	l.del(c.ID, u.Properties)
	require.Empty(t, l.f.callsMatching("DELETE", "NetworkAccessProfiles"))
	require.NotNil(t, l.f.get("ss", testAccount, "NetworkAccessProfiles/"+c.ID))

	l.f.remove("ss", testAccount, "NetworkAccessProfiles/"+c.ID)
	require.Empty(t, l.read(c.ID, u.Properties, cleared).ID)

	bad := l.check(pv(map[string]property.Value{"networks": strList("HW123", "nope")}))
	require.Len(t, bad.Failures, 1)
	require.Empty(t, l.check(pv(map[string]property.Value{"networks": strList(ssNetA)})).Failures)
	require.Empty(t, l.check(empty()).Failures)
}

func TestSuperSimFleetLC_Lifecycle(t *testing.T) {
	l := newTL(t, "SuperSimFleet", "fleet")
	prof := l.other("SuperSimNetworkAccessProfile", "profile")
	p1 := prof.create(pv(map[string]property.Value{"uniqueName": sv("p1")}))
	p2 := prof.create(pv(map[string]property.Value{"uniqueName": sv("p2")}))

	in := pv(map[string]property.Value{"networkAccessProfile": sv(p1.ID), "uniqueName": sv("meters"), "dataLimit": nv(500),
		"dataEnabled": bv(true), "smsCommandsUrl": sv("https://example.com/sms"), "ipCommandsMethod": sv("GET")})
	c := l.create(in)
	require.True(t, strings.HasPrefix(c.ID, "HF"), c.ID)
	require.Equal(t, p1.ID, getS(t, c.Properties, "networkAccessProfileSid"))
	require.Equal(t, "payg", getS(t, c.Properties, "dataMetering"))
	form := l.f.callsMatching("POST", "/supersim/v1/Fleets")[0].Form
	require.Equal(t, p1.ID, form.Get("NetworkAccessProfile"))
	require.Equal(t, "500", form.Get("DataLimit"))
	require.Equal(t, "true", form.Get("DataEnabled"))
	require.Equal(t, "GET", form.Get("IpCommandsMethod"))
	require.Empty(t, form["SmsCommandsEnabled"])
	requireNoChanges(t, l.diff(c.ID, c.Properties, in, in))
	// smsCommandsEnabled was left out: the state holds what Twilio chose, and stating it now is no change.
	require.True(t, c.Properties.Get("smsCommandsEnabled").AsBool())
	requireNoChanges(t, l.diff(c.ID, c.Properties, with(in, map[string]property.Value{"smsCommandsEnabled": bv(true)}), in))

	rd := l.read(c.ID, c.Properties, in)
	requireNoChanges(t, l.diff(c.ID, rd.Properties, in, rd.Inputs))

	// Drift in a setting the program states shows; a setting it leaves out does not.
	l.f.mutate("ss", testAccount, "Fleets/"+c.ID, func(o map[string]any) {
		o["data_limit"] = float64(900)
		o["ip_commands_url"] = "https://example.com/ip"
	})
	rd = l.read(c.ID, c.Properties, in)
	require.Equal(t, 900.0, rd.Inputs.Get("dataLimit").AsNumber())
	require.Equal(t, pgp.Update, l.diff(c.ID, rd.Properties, in, rd.Inputs).DetailedDiff["dataLimit"].Kind)
	require.False(t, hasProp(rd.Inputs, "ipCommandsUrl"))

	// A profile swapped outside Pulumi is noticed by its SID, whether the program names it by SID or by unique name.
	byName := with(in, map[string]property.Value{"networkAccessProfile": sv("p1"), "uniqueName": sv("meters-n")})
	cn := l.create(byName)
	requireNoChanges(t, l.diff(cn.ID, cn.Properties, byName, byName))
	rn := l.read(cn.ID, cn.Properties, byName)
	require.Equal(t, "p1", getS(t, rn.Inputs, "networkAccessProfile"))
	l.f.mutate("ss", testAccount, "Fleets/"+cn.ID, func(o map[string]any) { o["network_access_profile_sid"] = p2.ID })
	rn = l.read(cn.ID, cn.Properties, byName)
	require.Equal(t, p2.ID, getS(t, rn.Inputs, "networkAccessProfile"))
	require.Equal(t, pgp.Update, l.diff(cn.ID, rn.Properties, byName, rn.Inputs).DetailedDiff["networkAccessProfile"].Kind)
	l.del(cn.ID, cn.Properties)

	// Everything but dataEnabled and smsCommandsEnabled updates in place, and an update sends neither of them.
	next := with(in, map[string]property.Value{"networkAccessProfile": sv(p2.ID), "uniqueName": sv("meters-2"), "dataLimit": nv(2000),
		"smsCommandsUrl": sv("https://example.com/sms2"), "smsCommandsMethod": sv("GET"), "ipCommandsUrl": sv("https://example.com/ip2")})
	d := l.diff(c.ID, c.Properties, next, in)
	for _, k := range []string{"networkAccessProfile", "uniqueName", "dataLimit", "smsCommandsUrl", "smsCommandsMethod", "ipCommandsUrl"} {
		require.Equal(t, pgp.Update, d.DetailedDiff[k].Kind, k)
	}
	require.Len(t, d.DetailedDiff, 6)
	l.f.resetCalls()
	u := l.update(c.ID, c.Properties, next, in)
	uf := l.f.callsMatching("POST", "/supersim/v1/Fleets/"+c.ID)[0].Form
	require.Empty(t, uf["DataEnabled"])
	require.Empty(t, uf["SmsCommandsEnabled"])
	require.Equal(t, p2.ID, uf.Get("NetworkAccessProfile"))
	require.Equal(t, p2.ID, l.f.get("ss", testAccount, "Fleets/"+c.ID)["network_access_profile_sid"])
	require.Equal(t, p2.ID, getS(t, u.Properties, "networkAccessProfileSid"))
	requireNoChanges(t, l.diff(c.ID, u.Properties, next, next))

	// The two create-only flags replace the Fleet.
	for _, k := range []string{"dataEnabled", "smsCommandsEnabled"} {
		rep := with(next, map[string]property.Value{k: bv(false)})
		require.Equal(t, pgp.UpdateReplace, l.diff(c.ID, u.Properties, rep, next).DetailedDiff[k].Kind, k)
	}
	rep := l.create(with(next, map[string]property.Value{"uniqueName": sv("meters-3"), "dataEnabled": bv(false), "smsCommandsEnabled": bv(false)}))
	require.False(t, l.f.get("ss", testAccount, "Fleets/"+rep.ID)["data_enabled"].(bool))

	// Import adopts what differs from the defaults; the defaults stated explicitly are no change.
	l.f.mutate("ss", testAccount, "Fleets/"+c.ID, func(o map[string]any) {
		o["data_limit"] = float64(1000)
		o["ip_commands_method"] = "POST"
		o["sms_commands_url"] = nil
		o["ip_commands_url"] = nil
	})
	imp := l.read(c.ID, empty(), empty())
	require.Empty(t, l.check(imp.Inputs).Failures)
	require.Equal(t, p2.ID, getS(t, imp.Inputs, "networkAccessProfile"))
	require.Equal(t, "meters-2", getS(t, imp.Inputs, "uniqueName"))
	require.Equal(t, "GET", getS(t, imp.Inputs, "smsCommandsMethod"))
	for _, k := range []string{"dataLimit", "ipCommandsMethod", "smsCommandsUrl", "ipCommandsUrl"} {
		require.False(t, hasProp(imp.Inputs, k), k)
	}
	require.True(t, imp.Inputs.Get("dataEnabled").AsBool())
	explicit := with(imp.Inputs, map[string]property.Value{"dataLimit": nv(1000), "ipCommandsMethod": sv("POST")})
	requireNoChanges(t, l.diff(c.ID, imp.Properties, explicit, imp.Inputs))
	l.requireImportFails("HF" + strings.Repeat("0", 32))

	// Twilio has no DELETE: destroy forgets the Fleet and does not call it.
	l.f.resetCalls()
	l.del(c.ID, u.Properties)
	require.Empty(t, l.f.callsMatching("DELETE", "Fleets"))
	require.NotNil(t, l.f.get("ss", testAccount, "Fleets/"+c.ID))
	l.f.remove("ss", testAccount, "Fleets/"+c.ID)
	require.Empty(t, l.read(c.ID, u.Properties, next).ID)

	bad := l.check(pv(map[string]property.Value{"networkAccessProfile": sv(""), "dataLimit": nv(0), "ipCommandsMethod": sv("PUT"), "smsCommandsMethod": sv("PUT")}))
	require.Len(t, bad.Failures, 4)
	require.Len(t, l.check(pv(map[string]property.Value{"networkAccessProfile": sv("x"), "dataLimit": nv(2000001)})).Failures, 1)
	require.Empty(t, l.check(pv(map[string]property.Value{"networkAccessProfile": sv("x"), "dataLimit": nv(2000000)})).Failures)
	// A profile wired from another resource is unknown at preview.
	require.Empty(t, l.check(pv(map[string]property.Value{"networkAccessProfile": computed()})).Failures)
}

func TestWirelessRatePlanLC_Lifecycle(t *testing.T) {
	l := newTL(t, "WirelessRatePlan", "plan")
	in := pv(map[string]property.Value{"uniqueName": sv("meter"), "friendlyName": sv("Meter plan"), "dataLimit": nv(250), "dataMetering": sv("quota-10"),
		"internationalRoaming": strList("data", "messaging"), "dataLimitStrategy": sv("throttle")})
	c := l.create(in)
	require.True(t, strings.HasPrefix(c.ID, "WP"), c.ID)
	form := l.f.callsMatching("POST", "/wireless/v1/RatePlans")[0].Form
	require.Equal(t, "250", form.Get("DataLimit"))
	require.Equal(t, "quota-10", form.Get("DataMetering"))
	require.Equal(t, []string{"data", "messaging"}, form["InternationalRoaming"])
	require.Equal(t, "throttle", form.Get("DataLimitStrategy"))
	require.Empty(t, form["VoiceEnabled"])
	requireNoChanges(t, l.diff(c.ID, c.Properties, in, in))
	// Inputs left out hold what Twilio chose; stating the same value later is no change, a different one replaces.
	require.True(t, c.Properties.Get("messagingEnabled").AsBool())
	requireNoChanges(t, l.diff(c.ID, c.Properties, with(in, map[string]property.Value{"messagingEnabled": bv(true), "dataEnabled": bv(true)}), in))
	require.Equal(t, pgp.UpdateReplace, l.diff(c.ID, c.Properties, with(in, map[string]property.Value{"messagingEnabled": bv(false)}), in).DetailedDiff["messagingEnabled"].Kind)

	rd := l.read(c.ID, c.Properties, in)
	requireNoChanges(t, l.diff(c.ID, rd.Properties, in, rd.Inputs))
	// The roaming services in another order are not drift.
	l.f.mutate("ss", testAccount, "RatePlans/"+c.ID, func(o map[string]any) { o["international_roaming"] = []any{"messaging", "data"} })
	rd = l.read(c.ID, c.Properties, in)
	requireNoChanges(t, l.diff(c.ID, rd.Properties, in, rd.Inputs))

	// Only the two names update in place.
	next := with(in, map[string]property.Value{"uniqueName": sv("meter-2"), "friendlyName": sv("Meter plan 2")})
	d := l.diff(c.ID, c.Properties, next, in)
	require.Len(t, d.DetailedDiff, 2)
	require.Equal(t, pgp.Update, d.DetailedDiff["uniqueName"].Kind)
	require.Equal(t, pgp.Update, d.DetailedDiff["friendlyName"].Kind)
	l.f.resetCalls()
	u := l.update(c.ID, c.Properties, next, in)
	uf := l.f.callsMatching("POST", "/wireless/v1/RatePlans/"+c.ID)[0].Form
	require.Len(t, uf, 2)
	require.Equal(t, "meter-2", l.f.get("ss", testAccount, "RatePlans/"+c.ID)["unique_name"])
	require.Equal(t, 250.0, u.Properties.Get("dataLimit").AsNumber())
	requireNoChanges(t, l.diff(c.ID, u.Properties, next, next))

	// Every other input replaces the plan.
	for k, v := range map[string]property.Value{"dataEnabled": bv(false), "dataLimit": nv(300), "dataMetering": sv("payg"), "messagingEnabled": bv(false),
		"voiceEnabled": bv(true), "nationalRoamingEnabled": bv(false), "nationalRoamingDataLimit": nv(5), "internationalRoaming": strList("data"),
		"internationalRoamingDataLimit": nv(7), "dataLimitStrategy": sv("block")} {
		require.Equal(t, pgp.UpdateReplace, l.diff(c.ID, u.Properties, with(next, map[string]property.Value{k: v}), next).DetailedDiff[k].Kind, k)
	}

	// Import adopts everything Twilio reports (not dataLimitStrategy, which it does not); the result passes Check.
	imp := l.read(c.ID, empty(), empty())
	require.Empty(t, l.check(imp.Inputs).Failures)
	require.Equal(t, "meter-2", getS(t, imp.Inputs, "uniqueName"))
	require.Equal(t, 250.0, imp.Inputs.Get("dataLimit").AsNumber())
	require.False(t, hasProp(imp.Inputs, "dataLimitStrategy"))
	requireNoChanges(t, l.diff(c.ID, imp.Properties, without(next, "dataLimitStrategy"), imp.Inputs))
	l.requireImportFails("WP" + strings.Repeat("0", 32))

	l.del(c.ID, u.Properties)
	require.Nil(t, l.f.get("ss", testAccount, "RatePlans/"+c.ID))
	require.Len(t, l.f.callsMatching("DELETE", "/wireless/v1/RatePlans/"+c.ID), 1)
	l.del(c.ID, u.Properties) // already gone is fine
	require.Empty(t, l.read(c.ID, u.Properties, next).ID)

	bad := l.check(pv(map[string]property.Value{"dataLimit": nv(2000001), "nationalRoamingDataLimit": nv(-1), "internationalRoamingDataLimit": nv(-1),
		"dataMetering": sv("quota-5"), "dataLimitStrategy": sv("drop"), "internationalRoaming": strList("voice")}))
	require.Len(t, bad.Failures, 6)
	require.Empty(t, l.check(empty()).Failures)
}

func TestLookup_SuperSimAndWireless(t *testing.T) {
	l := newTL(t, "SuperSimNetworkAccessProfile", "profile")
	seedSuperSimNetworks(l)
	p := l.create(pv(map[string]property.Value{"uniqueName": sv("eu"), "networks": strList(ssNetA, ssNetB)}))
	f := l.other("SuperSimFleet", "fleet").create(pv(map[string]property.Value{"networkAccessProfile": sv("eu"), "uniqueName": sv("meters"), "dataLimit": nv(300)}))

	r, err := invoke(t, l, "getSuperSimFleet", map[string]property.Value{"sid": sv(f.ID)})
	require.NoError(t, err)
	require.Equal(t, p.ID, getS(t, r, "networkAccessProfileSid"))
	require.Equal(t, "meters", getS(t, r, "uniqueName"))
	require.Equal(t, 300.0, r.Get("dataLimit").AsNumber())
	require.Equal(t, "payg", getS(t, r, "dataMetering"))
	require.Equal(t, "POST", getS(t, r, "ipCommandsMethod"))
	require.False(t, hasProp(r, "ipCommandsUrl"))
	r, err = invoke(t, l, "getSuperSimFleet", map[string]property.Value{"sid": sv("meters")})
	require.NoError(t, err)
	require.Equal(t, f.ID, getS(t, r, "sid"))
	_, err = invoke(t, l, "getSuperSimFleet", map[string]property.Value{"sid": sv("HFmissing")})
	require.ErrorContains(t, err, "was not found")
	_, err = invoke(t, l, "getSuperSimFleet", map[string]property.Value{"sid": sv("")})
	require.ErrorContains(t, err, "must be set")

	r, err = invoke(t, l, "getSuperSimNetworkAccessProfile", map[string]property.Value{"sid": sv(p.ID)})
	require.NoError(t, err)
	require.Equal(t, "eu", getS(t, r, "uniqueName"))
	nets := r.Get("networks").AsArray().AsSlice()
	require.Len(t, nets, 2)
	require.Equal(t, ssNetA, getS(t, nets[0].AsMap(), "sid"))
	require.Len(t, nets[0].AsMap().Get("identifiers").AsArray().AsSlice(), 2)
	_, err = invoke(t, l, "getSuperSimNetworkAccessProfile", map[string]property.Value{"sid": sv("HAmissing")})
	require.ErrorContains(t, err, "was not found")

	r, err = invoke(t, l, "getSuperSimNetwork", map[string]property.Value{"sid": sv(ssNetB)})
	require.NoError(t, err)
	require.Equal(t, "Telco B", getS(t, r, "friendlyName"))
	require.Equal(t, "DE", getS(t, r, "isoCountry"))
	require.Equal(t, "262", getS(t, r.Get("identifiers").AsArray().AsSlice()[0].AsMap(), "mcc"))
	_, err = invoke(t, l, "getSuperSimNetwork", map[string]property.Value{"sid": sv("HWmissing")})
	require.ErrorContains(t, err, "was not found")

	// The catalog list pages (two a page in the fake) and every filter narrows it.
	r, err = invoke(t, l, "getSuperSimNetworks", map[string]property.Value{})
	require.NoError(t, err)
	require.Len(t, r.Get("networks").AsArray().AsSlice(), 3)
	for filter, want := range map[string][2]string{"isoCountry": {"US", "2"}, "mcc": {"262", "1"}, "mnc": {"480", "1"}} {
		r, err = invoke(t, l, "getSuperSimNetworks", map[string]property.Value{filter: sv(want[0])})
		require.NoError(t, err)
		require.Len(t, r.Get("networks").AsArray().AsSlice(), map[string]int{"2": 2, "1": 1}[want[1]], filter)
	}
	r, err = invoke(t, l, "getSuperSimNetworks", map[string]property.Value{"isoCountry": sv("FR")})
	require.NoError(t, err)
	require.Empty(t, r.Get("networks").AsArray().AsSlice())

	plan := l.other("WirelessRatePlan", "plan").create(pv(map[string]property.Value{"uniqueName": sv("meter"), "dataLimit": nv(250), "internationalRoaming": strList("data")}))
	r, err = invoke(t, l, "getWirelessRatePlan", map[string]property.Value{"sid": sv(plan.ID)})
	require.NoError(t, err)
	require.Equal(t, "meter", getS(t, r, "uniqueName"))
	require.Equal(t, 250.0, r.Get("dataLimit").AsNumber())
	require.Equal(t, "payg", getS(t, r, "dataMetering"))
	require.Len(t, r.Get("internationalRoaming").AsArray().AsSlice(), 1)
	require.False(t, hasProp(r, "friendlyName"))
	_, err = invoke(t, l, "getWirelessRatePlan", map[string]property.Value{"sid": sv("WPmissing")})
	require.ErrorContains(t, err, "was not found")
}
