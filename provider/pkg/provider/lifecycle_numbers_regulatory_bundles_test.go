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

const regRegulation = "RN00000000000000000000000000000011"

func regBundleInputs() property.Map {
	return pv(map[string]property.Value{"friendlyName": sv("DE numbers"), "email": sv("compliance@example.com"), "regulationSid": sv(regRegulation),
		"statusCallback": sv("https://hooks.example.com/status"), "isTest": bv(false)})
}

func TestRegulatoryBundleLC_Lifecycle(t *testing.T) {
	l := newTL(t, "RegulatoryBundle", "b")
	base := "/numbers/v2/RegulatoryCompliance/Bundles"
	in := regBundleInputs()
	c := l.create(in)
	require.True(t, strings.HasPrefix(c.ID, "BU"), c.ID)
	require.Equal(t, "draft", getS(t, c.Properties, "status"))
	require.Equal(t, regRegulation, getS(t, c.Properties, "effectiveRegulationSid"))
	form := l.f.callsMatching("POST", base)[0].Form
	require.Equal(t, "DE numbers", form.Get("FriendlyName"))
	require.Equal(t, regRegulation, form.Get("RegulationSid"))
	require.Empty(t, form.Get("Status"), "creating does not submit")
	requireNoChanges(t, l.diff(c.ID, c.Properties, in, in))

	rd := l.read(c.ID, c.Properties, in)
	requireNoChanges(t, l.diff(c.ID, rd.Properties, in, rd.Inputs))
	l.f.mutate("th:Bundles", testAccount, c.ID, func(o map[string]any) { o["email"] = "other@example.com" })
	require.Equal(t, "other@example.com", getS(t, l.read(c.ID, c.Properties, in).Inputs, "email"))
	l.f.mutate("th:Bundles", testAccount, c.ID, func(o map[string]any) { o["email"] = "compliance@example.com" })

	next := with(in, map[string]property.Value{"friendlyName": sv("Renamed"), "email": sv("new@example.com"), "statusCallback": sv("https://hooks.example.com/v2")})
	d := l.diff(c.ID, c.Properties, next, in)
	for _, k := range []string{"friendlyName", "email", "statusCallback"} {
		require.Equal(t, pgp.Update, d.DetailedDiff[k].Kind, k)
	}
	u := l.update(c.ID, c.Properties, next, in)
	require.Equal(t, "new@example.com", l.f.get("th:Bundles", testAccount, c.ID)["email"])
	requireNoChanges(t, l.diff(c.ID, u.Properties, without(next, "statusCallback"), next))

	// The regulation replaces, and so does every create-only input once both sides are known.
	for k, v := range map[string]property.Value{"regulationSid": sv("RN00000000000000000000000000000012"), "isTest": bv(true)} {
		require.Equal(t, pgp.UpdateReplace, l.diff(c.ID, u.Properties, with(next, map[string]property.Value{k: v}), next).DetailedDiff[k].Kind, k)
	}

	l.del(c.ID, u.Properties)
	require.Nil(t, l.f.get("th:Bundles", testAccount, c.ID))
	l.del(c.ID, u.Properties)
	require.Empty(t, l.read(c.ID, u.Properties, next).ID)
}

// Twilio resolves a regulation from the country, number type and end user type, and returns none of them: an
// imported bundle has no create-only input recorded, so a program that states them plans no replacement.
func TestRegulatoryBundleLC_ResolvedRegulationAndImport(t *testing.T) {
	l := newTL(t, "RegulatoryBundle", "b")
	in := pv(map[string]property.Value{"friendlyName": sv("DE numbers"), "email": sv("c@example.com"), "isoCountry": sv("DE"), "numberType": sv("local"), "endUserType": sv("business")})
	c := l.create(in)
	form := l.f.callsMatching("POST", "/numbers/v2/RegulatoryCompliance/Bundles")[0].Form
	require.Equal(t, "DE", form.Get("IsoCountry"))
	require.Equal(t, "local", form.Get("NumberType"))
	require.Equal(t, "business", form.Get("EndUserType"))
	require.Empty(t, form.Get("RegulationSid"))
	require.NotEmpty(t, getS(t, c.Properties, "effectiveRegulationSid"))
	requireNoChanges(t, l.diff(c.ID, c.Properties, in, in))

	imp := l.read(c.ID, empty(), empty())
	require.Equal(t, "DE numbers", getS(t, imp.Inputs, "friendlyName"))
	for _, k := range []string{"isoCountry", "numberType", "endUserType", "regulationSid", "isTest"} {
		_, adopted := imp.Inputs.GetOk(k)
		require.False(t, adopted, k)
	}
	requireNoChanges(t, l.diff(c.ID, imp.Properties, in, imp.Inputs))
	requireNoChanges(t, l.diff(c.ID, imp.Properties, with(in, map[string]property.Value{"regulationSid": sv(regRegulation)}), imp.Inputs))
	l.requireImportFails("BU00000000000000000000000000000099")
}

func TestRegulatoryBundleLC_SubmitIsOneWay(t *testing.T) {
	l := newTL(t, "RegulatoryBundle", "b")
	al := l.other("RegulatoryBundleItemAssignment", "a")
	base := "/numbers/v2/RegulatoryCompliance/Bundles"
	in := with(regBundleInputs(), map[string]property.Value{"submit": bv(true)})

	_, err := l.s.Create(pgp.CreateRequest{Urn: l.urn, Properties: l.checked(in)})
	require.ErrorContains(t, err, "not compliant")
	require.ErrorContains(t, err, "the draft was deleted")
	for k := range l.f.objs {
		require.False(t, strings.HasPrefix(k, "th:Bundles/"), "the draft is gone: %s", k)
	}

	draft := with(in, map[string]property.Value{"submit": bv(false)})
	c := l.create(draft)
	al.create(pv(map[string]property.Value{"bundleSid": sv(c.ID), "objectSid": sv(bundleObject)}))
	require.Equal(t, pgp.Update, l.diff(c.ID, c.Properties, in, draft).DetailedDiff["submit"].Kind)
	l.f.resetCalls()
	u := l.update(c.ID, c.Properties, in, draft)
	require.Equal(t, "pending-review", l.f.callsMatching("POST", base+"/"+c.ID)[0].Form.Get("Status"))
	require.Equal(t, "pending-review", getS(t, u.Properties, "status"))
	requireNoChanges(t, l.diff(c.ID, u.Properties, draft, in))
	requireNoChanges(t, l.diff(c.ID, u.Properties, without(in, "submit"), in))
	l.f.resetCalls()
	l.update(c.ID, u.Properties, with(in, map[string]property.Value{"friendlyName": sv("again")}), in)
	require.Empty(t, l.f.callsMatching("POST", base+"/"+c.ID)[0].Form.Get("Status"))
	require.Error(t, l.s.Delete(pgp.DeleteRequest{ID: c.ID, Urn: l.urn, Properties: u.Properties}))
}

func TestRegulatoryBundleLC_Checks(t *testing.T) {
	l := newTL(t, "RegulatoryBundle", "b")
	in := regBundleInputs()
	require.Empty(t, l.check(in).Failures)
	byParts := pv(map[string]property.Value{"friendlyName": sv("x"), "email": sv("c@example.com"), "isoCountry": sv("DE"), "numberType": sv("local"), "endUserType": sv("individual")})
	require.Empty(t, l.check(byParts).Failures)
	for name, c := range map[string]struct {
		in   property.Map
		prop string
	}{
		"empty name":        {with(in, map[string]property.Value{"friendlyName": sv("")}), "friendlyName"},
		"bad email":         {with(in, map[string]property.Value{"email": sv("nope")}), "email"},
		"bad callback":      {with(in, map[string]property.Value{"statusCallback": sv("ftp://x")}), "statusCallback"},
		"bad regulation":    {with(in, map[string]property.Value{"regulationSid": sv("BU1")}), "regulationSid"},
		"neither":           {without(in, "regulationSid"), "regulationSid"},
		"both":              {with(in, map[string]property.Value{"isoCountry": sv("DE")}), "regulationSid"},
		"partial":           {without(byParts, "numberType"), "regulationSid"},
		"bad country":       {with(byParts, map[string]property.Value{"isoCountry": sv("Germany")}), "isoCountry"},
		"bad end user type": {with(byParts, map[string]property.Value{"endUserType": sv("company")}), "endUserType"},
	} {
		f := l.check(c.in).Failures
		require.NotEmpty(t, f, name)
		require.Equal(t, c.prop, f[0].Property, name)
	}
}

func TestRegulatoryBundleItemAssignmentLC(t *testing.T) {
	bl := newTL(t, "RegulatoryBundle", "b")
	l := bl.other("RegulatoryBundleItemAssignment", "a")
	b := bl.create(regBundleInputs())
	in := pv(map[string]property.Value{"bundleSid": sv(b.ID), "objectSid": sv(bundleObject)})
	c := l.create(in)
	sid := getS(t, c.Properties, "sid")
	require.True(t, strings.HasPrefix(sid, "BV"), sid)
	require.Equal(t, b.ID+"/"+sid, c.ID)
	require.Equal(t, bundleObject, l.f.callsMatching("POST", "/numbers/v2/RegulatoryCompliance/Bundles/"+b.ID+"/ItemAssignments")[0].Form.Get("ObjectSid"))
	requireNoChanges(t, l.diff(c.ID, c.Properties, in, in))
	for k, v := range map[string]property.Value{"bundleSid": sv("BU00000000000000000000000000000002"), "objectSid": sv("RD00000000000000000000000000000001")} {
		require.Equal(t, pgp.UpdateReplace, l.diff(c.ID, c.Properties, with(in, map[string]property.Value{k: v}), in).DetailedDiff[k].Kind, k)
	}
	imp := l.read(c.ID, empty(), empty())
	require.Equal(t, b.ID, getS(t, imp.Inputs, "bundleSid"))
	requireNoChanges(t, l.diff(c.ID, imp.Properties, in, imp.Inputs))
	l.requireImportFails(b.ID + "/BV00000000000000000000000000000099")
	require.NotEmpty(t, l.check(with(in, map[string]property.Value{"objectSid": sv("PN1")})).Failures)
	require.NotEmpty(t, l.check(with(in, map[string]property.Value{"bundleSid": sv("x")})).Failures)

	r, err := invoke(t, l, "getRegulatoryBundleItemAssignment", map[string]property.Value{"bundleSid": sv(b.ID), "sid": sv(sid)})
	require.NoError(t, err)
	require.Equal(t, bundleObject, getS(t, r, "objectSid"))
	_, err = invoke(t, l, "getRegulatoryBundleItemAssignment", map[string]property.Value{"bundleSid": sv(b.ID), "sid": sv("BV00000000000000000000000000000099")})
	require.Error(t, err)
	r, err = invoke(t, l, "getRegulatoryBundle", map[string]property.Value{"sid": sv(b.ID)})
	require.NoError(t, err)
	require.Equal(t, "DE numbers", getS(t, r, "friendlyName"))
	require.Equal(t, regRegulation, getS(t, r, "regulationSid"))
	_, err = invoke(t, l, "getRegulatoryBundle", map[string]property.Value{"sid": sv("BU00000000000000000000000000000099")})
	require.Error(t, err)

	l.del(c.ID, c.Properties)
	require.Empty(t, l.read(c.ID, c.Properties, in).ID)
	l.del(c.ID, c.Properties)
}
