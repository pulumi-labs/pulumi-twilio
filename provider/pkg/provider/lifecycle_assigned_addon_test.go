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
	"strconv"
	"strings"
	"testing"

	pgp "github.com/pulumi/pulumi-go-provider"
	"github.com/pulumi/pulumi/sdk/v3/go/property"
	"github.com/stretchr/testify/require"
)

const (
	testNumberSid = "PN00000000000000000000000000000001"
	testAddOnSid  = "XE0000000000000000000000000000000a"
	testAddOnSid2 = "XE0000000000000000000000000000000b"
)

func addonInputs(number, installed string) property.Map {
	return pv(map[string]property.Value{"phoneNumberSid": sv(number), "installedAddOnSid": sv(installed)})
}

func addonTL(t *testing.T) *tl {
	l := newTL(t, "IncomingPhoneNumberAssignedAddOn", "a")
	l.f.addNumber(testAccount, testNumberSid, "+14155550100")
	return l
}

func TestAssignedAddOnLC_Lifecycle(t *testing.T) {
	l := addonTL(t)
	in := addonInputs(testNumberSid, testAddOnSid)
	c := l.create(in)
	require.True(t, strings.HasPrefix(c.ID, testNumberSid+"/XE"), c.ID)
	_, _, sid, err := splitChildID(c.ID)
	require.NoError(t, err)
	require.Equal(t, sid, getS(t, c.Properties, "sid"))
	require.Equal(t, testAccount, getS(t, c.Properties, "ownerAccountSid"))
	require.Equal(t, "Spam Filter", getS(t, c.Properties, "friendlyName"))
	require.JSONEq(t, `{"threshold":"high"}`, getS(t, c.Properties, "configuration"))
	require.Equal(t, testAddOnSid, l.f.get("addonasg", testAccount, sid)["_installed"])
	require.Equal(t, []string{testAddOnSid}, l.f.callsMatching("POST", "/AssignedAddOns.json")[0].Form["InstalledAddOnSid"])
	requireNoChanges(t, l.diff(c.ID, c.Properties, in, in))

	// A refresh keeps the installation SID Twilio never returns and plans no change.
	rd := l.read(c.ID, c.Properties, in)
	require.Equal(t, testAddOnSid, getS(t, rd.Inputs, "installedAddOnSid"))
	require.Equal(t, testNumberSid, getS(t, rd.Inputs, "phoneNumberSid"))
	require.False(t, hasProp(rd.Inputs, "accountSid"))
	requireNoChanges(t, l.diff(c.ID, rd.Properties, in, rd.Inputs))

	// Twilio has no update: every input replaces.
	for k, v := range map[string]property.Value{"installedAddOnSid": sv(testAddOnSid2), "phoneNumberSid": sv("PN0000"), "accountSid": sv(fakeSub)} {
		d := l.diff(c.ID, rd.Properties, with(in, map[string]property.Value{k: v}), in)
		require.True(t, d.HasChanges, k)
		require.Contains(t, []pgp.DiffKind{pgp.UpdateReplace, pgp.AddReplace}, d.DetailedDiff[k].Kind, k)
	}

	// Gone out of band: a refresh drops it, and deleting twice is fine.
	l.f.remove("addonasg", testAccount, sid)
	require.Empty(t, l.read(c.ID, rd.Properties, in).ID)
	l.del(c.ID, rd.Properties)
	c = l.create(in)
	l.del(c.ID, c.Properties)
	_, _, sid, _ = splitChildID(c.ID)
	require.Nil(t, l.f.get("addonasg", testAccount, sid))
	require.Len(t, l.f.callsMatching("DELETE", "/AssignedAddOns/"+sid+".json"), 1)
}

func TestAssignedAddOnLC_ImportPreviewAndChecks(t *testing.T) {
	l := addonTL(t)
	l.f.addSubaccount(fakeSub)
	l.f.addNumber(fakeSub, "PN00000000000000000000000000000002", "+14155550101")
	in := with(addonInputs("PN00000000000000000000000000000002", testAddOnSid), map[string]property.Value{"accountSid": sv(fakeSub)})
	c := l.create(in)
	require.True(t, strings.HasPrefix(c.ID, fakeSub+"/PN00000000000000000000000000000002/XE"), c.ID)

	// Import by the three-part ID: the number and account are recovered, the installation is not.
	imp := l.read(c.ID, empty(), empty())
	require.Equal(t, fakeSub, getS(t, imp.Inputs, "accountSid"))
	require.Equal(t, "PN00000000000000000000000000000002", getS(t, imp.Inputs, "phoneNumberSid"))
	require.Equal(t, "", getS(t, imp.Inputs, "installedAddOnSid"))
	// The engine runs Check on the imported inputs: the empty installation SID is not a failure...
	require.Empty(t, l.check(imp.Inputs).Failures, "importing must not warn that the imported inputs failed to validate")
	// ...the program's value is adopted without replacing the assignment...
	d := l.diff(c.ID, imp.Properties, in, imp.Inputs)
	require.True(t, d.HasChanges)
	require.Equal(t, pgp.Update, d.DetailedDiff["installedAddOnSid"].Kind)
	require.Len(t, d.DetailedDiff, 1)
	// ...by an Update that calls no API and records it, so a later change is a replacement even before a refresh.
	l.f.resetCalls()
	u := l.update(c.ID, imp.Properties, in, imp.Inputs)
	require.Zero(t, l.f.callCount(), "adopting the installation SID makes no API call")
	require.Equal(t, testAddOnSid, getS(t, u.Properties, "installedAddOnSid"))
	require.Equal(t, getS(t, imp.Properties, "sid"), getS(t, u.Properties, "sid"))
	requireNoChanges(t, l.diff(c.ID, u.Properties, in, in))
	d = l.diff(c.ID, u.Properties, with(in, map[string]property.Value{"installedAddOnSid": sv(testAddOnSid2)}), in)
	require.Equal(t, pgp.UpdateReplace, d.DetailedDiff["installedAddOnSid"].Kind, "XE2 replaces against the unrefreshed state")
	// ...but Create refuses an empty one, and a preview does not.
	_, err := l.s.Create(pgp.CreateRequest{Urn: l.urn, Properties: imp.Inputs})
	require.ErrorContains(t, err, "installedAddOnSid must be set")
	_, err = l.s.Create(pgp.CreateRequest{Urn: l.urn, Properties: imp.Inputs, DryRun: true})
	require.NoError(t, err)

	// Import by <phoneNumberSid>/<sid> in the provider account.
	own := l.create(addonInputs(testNumberSid, testAddOnSid))
	imp = l.read(own.ID, empty(), empty())
	require.False(t, hasProp(imp.Inputs, "accountSid"))
	require.Equal(t, testNumberSid, getS(t, imp.Inputs, "phoneNumberSid"))
	_, err = l.s.Read(pgp.ReadRequest{ID: "bare-sid", Urn: l.urn, Properties: empty(), Inputs: empty()})
	require.ErrorContains(t, err, "invalid ID")

	// A preview creates nothing and invents no ID.
	before := len(l.f.callsMatching("POST", "/AssignedAddOns.json"))
	pr, err := l.s.Create(pgp.CreateRequest{Urn: l.urn, Properties: addonInputs(testNumberSid, testAddOnSid2), DryRun: true})
	require.NoError(t, err)
	require.Empty(t, pr.ID)
	require.Len(t, l.f.callsMatching("POST", "/AssignedAddOns.json"), before)

	// The API refuses the same installation twice; the error names the number.
	_, err = l.s.Create(pgp.CreateRequest{Urn: l.urn, Properties: addonInputs(testNumberSid, testAddOnSid)})
	require.ErrorContains(t, err, "already assigned")
	require.ErrorContains(t, err, testNumberSid)

	require.NotEmpty(t, l.check(addonInputs("", testAddOnSid)).Failures)
	require.NotEmpty(t, l.check(addonInputs(testNumberSid, "XE123")).Failures)
	require.NotEmpty(t, l.check(addonInputs(testNumberSid, "KS0000000000000000000000000000000a")).Failures)
	require.Empty(t, l.check(pv(map[string]property.Value{"phoneNumberSid": computed(), "installedAddOnSid": computed(), "accountSid": computed()})).Failures)

	// The number gone out of band takes its assignment with it.
	l.f.remove("number", testAccount, testNumberSid)
	require.Empty(t, l.read(own.ID, own.Properties, addonInputs(testNumberSid, testAddOnSid)).ID)
}

func TestLookup_AssignedAddOn(t *testing.T) {
	l := addonTL(t)
	c := l.create(addonInputs(testNumberSid, testAddOnSid))
	_, _, sid, err := splitChildID(c.ID)
	require.NoError(t, err)
	args := map[string]property.Value{"phoneNumberSid": sv(testNumberSid), "assignedAddOnSid": sv(sid)}

	r, err := invoke(t, l, "getIncomingPhoneNumberAssignedAddOn", args)
	require.NoError(t, err)
	require.Equal(t, sid, getS(t, r, "sid"))
	require.Equal(t, testAccount, getS(t, r, "ownerAccountSid"))
	require.Equal(t, "Spam Filter", getS(t, r, "friendlyName"))
	require.Equal(t, "spam_filter", getS(t, r, "uniqueName"))
	require.Equal(t, 0, listLen(t, r, "extensions"))

	// Extensions follow the list's paging.
	var exts []map[string]any
	for i := 0; i < 5; i++ {
		exts = append(exts, map[string]any{"sid": "XF" + strconv.Itoa(i), "friendly_name": "ext" + strconv.Itoa(i), "product_name": "Spam Filter",
			"unique_name": "spam_filter.ext" + strconv.Itoa(i), "enabled": i%2 == 0})
	}
	l.f.mutate("addonasg", testAccount, sid, func(o map[string]any) { o["_extensions"] = exts })
	old := assignedAddOnPageCap
	assignedAddOnPageCap = 2
	t.Cleanup(func() { assignedAddOnPageCap = old })
	l.f.resetCalls()
	r, err = invoke(t, l, "getIncomingPhoneNumberAssignedAddOn", args)
	require.NoError(t, err)
	require.Equal(t, 5, listLen(t, r, "extensions"))
	require.Len(t, l.f.callsMatching("GET", "/Extensions.json"), 3, "three pages of two, two and one")
	last := r.Get("extensions").AsArray().Get(4).AsMap()
	require.Equal(t, "XF4", getS(t, last, "sid"))
	require.True(t, last.Get("enabled").AsBool())
	require.False(t, r.Get("extensions").AsArray().Get(1).AsMap().Get("enabled").AsBool())

	_, err = invoke(t, l, "getIncomingPhoneNumberAssignedAddOn", map[string]property.Value{"phoneNumberSid": sv(testNumberSid), "assignedAddOnSid": sv("XE0000")})
	require.ErrorContains(t, err, "was not found")
	_, err = invoke(t, l, "getIncomingPhoneNumberAssignedAddOn", map[string]property.Value{"phoneNumberSid": sv(""), "assignedAddOnSid": sv(sid)})
	require.ErrorContains(t, err, "must be set")
	_, err = invoke(t, l, "getIncomingPhoneNumberAssignedAddOn", map[string]property.Value{"phoneNumberSid": sv(testNumberSid), "assignedAddOnSid": sv("")})
	require.ErrorContains(t, err, "must be set")
}
