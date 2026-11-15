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

const (
	testFlexInstance = "GO11111111111111111111111111111111"
	testFlexUser     = "FO22222222222222222222222222222222"
	testFrontlineSid = "US33333333333333333333333333333333"
)

func TestFlexUserLC_Lifecycle(t *testing.T) {
	l := newTL(t, "FlexUser", "u")
	ids := map[string]property.Value{"instanceSid": sv(testFlexInstance), "flexUserSid": sv(testFlexUser)}
	id := testFlexInstance + "/" + testFlexUser

	// Adopting a user that does not exist is an error: there is no create.
	_, err := l.s.Create(pgp.CreateRequest{Urn: l.urn, Properties: pv(ids)})
	require.ErrorContains(t, err, "cannot create one")

	l.f.seedFlexUser(testFlexInstance, testFlexUser)
	in := pv(with2(ids, map[string]property.Value{"locale": sv("de-DE")}))
	c := l.create(in)
	require.Equal(t, id, c.ID)
	form := l.f.callsMatching("POST", "/flex-api/v2/Instances/"+testFlexInstance+"/Users/"+testFlexUser)[0].Form
	require.Equal(t, "de-DE", form.Get("Locale"))
	require.Empty(t, form["Email"])
	require.Equal(t, "ada", getS(t, c.Properties, "username"))
	requireNoChanges(t, l.diff(c.ID, c.Properties, in, in))

	// Only an adopted user with nothing stated is read, not written.
	l.f.resetCalls()
	bare := l.other("FlexUser", "bare")
	bare.create(pv(ids))
	require.Empty(t, l.f.callsMatching("POST", "/Users/"))

	next := pv(with2(ids, map[string]property.Value{"locale": sv("de-DE"), "email": sv("a@b.example"), "userSid": sv("US" + strings.Repeat("b", 32))}))
	require.Equal(t, pgp.Update, l.diff(c.ID, c.Properties, next, in).DetailedDiff["email"].Kind)
	u := l.update(c.ID, c.Properties, next, in)
	require.Equal(t, "a@b.example", l.f.get("flexuser", testAccount, id)["email"])
	require.Equal(t, "US"+strings.Repeat("b", 32), l.f.get("flexuser", testAccount, id)["user_sid"])
	requireNoChanges(t, l.diff(c.ID, u.Properties, next, next))
	requireNoChanges(t, l.diff(c.ID, u.Properties, without(next, "email"), next))

	// A different user replaces.
	other := pv(with2(map[string]property.Value{"instanceSid": sv(testFlexInstance), "flexUserSid": sv("FO" + strings.Repeat("9", 32))}, nil))
	require.Equal(t, pgp.UpdateReplace, l.diff(c.ID, u.Properties, other, next).DetailedDiff["flexUserSid"].Kind)

	// Drift on a managed setting, none on the rest.
	l.f.mutate("flexuser", testAccount, id, func(o map[string]any) { o["locale"] = "fr-FR"; o["email"] = "x@y.example" })
	rd := l.read(c.ID, u.Properties, next)
	require.Equal(t, "fr-FR", getS(t, rd.Inputs, "locale"))
	require.Equal(t, pgp.Update, l.diff(c.ID, rd.Properties, next, rd.Inputs).DetailedDiff["locale"].Kind)

	// An import adopts none of the settings.
	imp := l.read(id, empty(), empty())
	require.Equal(t, id, imp.ID)
	require.Equal(t, testFlexUser, getS(t, imp.Inputs, "flexUserSid"))
	require.False(t, hasProp(imp.Inputs, "email"))
	require.False(t, hasProp(imp.Inputs, "locale"))
	requireNoChanges(t, l.diff(imp.ID, imp.Properties, imp.Inputs, imp.Inputs))
	_, err = l.s.Read(pgp.ReadRequest{ID: "nope", Urn: l.urn, Properties: empty(), Inputs: empty()})
	require.ErrorContains(t, err, "invalid ID")
	_, err = l.s.Read(pgp.ReadRequest{ID: testFlexInstance + "/FO" + strings.Repeat("8", 32), Urn: l.urn, Properties: empty(), Inputs: empty()})
	require.ErrorContains(t, err, "does not exist")

	// Destroy forgets the user and calls nothing.
	l.f.resetCalls()
	l.del(c.ID, u.Properties)
	require.Empty(t, l.f.callsMatching("DELETE", "/"))
	require.NotNil(t, l.f.get("flexuser", testAccount, id))

	require.Len(t, l.check(pv(with2(ids, map[string]property.Value{"userSid": sv("nope")}))).Failures, 1)
	require.Len(t, l.check(pv(map[string]property.Value{"instanceSid": sv(""), "flexUserSid": sv("")})).Failures, 2)
}

func TestFrontlineUserLC_Lifecycle(t *testing.T) {
	l := newTL(t, "FrontlineUser", "u")
	ids := map[string]property.Value{"sid": sv(testFrontlineSid)}
	_, err := l.s.Create(pgp.CreateRequest{Urn: l.urn, Properties: pv(ids)})
	require.ErrorContains(t, err, "cannot create one")

	l.f.seedFrontlineUser(testFrontlineSid)
	in := pv(with2(ids, map[string]property.Value{"friendlyName": sv("Ada L"), "isAvailable": bv(true)}))
	c := l.create(in)
	require.Equal(t, testFrontlineSid, c.ID)
	form := l.f.callsMatching("POST", "/frontline-api/v1/Users/"+testFrontlineSid)[0].Form
	require.Equal(t, "Ada L", form.Get("FriendlyName"))
	require.Equal(t, "true", form.Get("IsAvailable"))
	require.Empty(t, form["State"])
	require.Equal(t, "ada@example.com", getS(t, c.Properties, "identity"))
	requireNoChanges(t, l.diff(c.ID, c.Properties, in, in))

	// Writing the Twilio defaults changes nothing.
	defaults := pv(with2(ids, map[string]property.Value{"friendlyName": sv("Ada L"), "isAvailable": bv(true), "state": sv("active")}))
	requireNoChanges(t, l.diff(c.ID, c.Properties, defaults, in))
	bare := l.other("FrontlineUser", "bare")
	bc := bare.create(pv(ids))
	requireNoChanges(t, bare.diff(bc.ID, bc.Properties, pv(with2(ids, map[string]property.Value{"state": sv("active"), "isAvailable": bv(false)})), pv(ids)))

	next := pv(with2(ids, map[string]property.Value{"friendlyName": sv("Ada L"), "isAvailable": bv(true), "state": sv("deactivated"), "avatar": sv("https://example.com/a.png")}))
	require.Equal(t, pgp.Update, l.diff(c.ID, c.Properties, next, in).DetailedDiff["state"].Kind)
	u := l.update(c.ID, c.Properties, next, in)
	require.Equal(t, "deactivated", l.f.get("frontlineuser", testAccount, testFrontlineSid)["state"])
	requireNoChanges(t, l.diff(c.ID, u.Properties, next, next))

	l.f.mutate("frontlineuser", testAccount, testFrontlineSid, func(o map[string]any) { o["friendly_name"] = "Renamed" })
	rd := l.read(c.ID, u.Properties, next)
	require.Equal(t, "Renamed", getS(t, rd.Inputs, "friendlyName"))
	require.Equal(t, pgp.Update, l.diff(c.ID, rd.Properties, next, rd.Inputs).DetailedDiff["friendlyName"].Kind)

	// An import adopts what differs from the default and nothing else.
	imp := l.read(testFrontlineSid, empty(), empty())
	require.Equal(t, "deactivated", getS(t, imp.Inputs, "state"))
	require.True(t, imp.Inputs.Get("isAvailable").AsBool())
	require.False(t, hasProp(imp.Inputs, "friendlyName"))
	require.False(t, hasProp(imp.Inputs, "avatar"))
	requireNoChanges(t, l.diff(imp.ID, imp.Properties, imp.Inputs, imp.Inputs))
	_, err = l.s.Read(pgp.ReadRequest{ID: "US" + strings.Repeat("7", 32), Urn: l.urn, Properties: empty(), Inputs: empty()})
	require.ErrorContains(t, err, "does not exist")

	require.Equal(t, pgp.UpdateReplace, l.diff(c.ID, u.Properties, pv(map[string]property.Value{"sid": sv("US" + strings.Repeat("6", 32))}), next).DetailedDiff["sid"].Kind)
	l.f.resetCalls()
	l.del(c.ID, u.Properties)
	require.Empty(t, l.f.callsMatching("DELETE", "/"))
	require.Len(t, l.check(pv(with2(ids, map[string]property.Value{"state": sv("gone")}))).Failures, 1)
}

func TestLookup_FlexAndFrontlineUsers(t *testing.T) {
	l := newTL(t, "FlexUser", "u")
	l.f.seedFlexUser(testFlexInstance, testFlexUser)
	l.f.seedFrontlineUser(testFrontlineSid)
	r, err := invoke(t, l, "getFlexUser", map[string]property.Value{"instanceSid": sv(testFlexInstance), "flexUserSid": sv(testFlexUser)})
	require.NoError(t, err)
	require.Equal(t, "ada", getS(t, r, "username"))
	require.Equal(t, "ada@example.com", getS(t, r, "email"))
	require.Equal(t, "US"+strings.Repeat("a", 32), getS(t, r, "userSid"))
	require.Equal(t, float64(1), r.Get("userVersion").AsNumber())
	_, err = invoke(t, l, "getFlexUser", map[string]property.Value{"instanceSid": sv(testFlexInstance), "flexUserSid": sv("FOmissing")})
	require.ErrorContains(t, err, "was not found")
	_, err = invoke(t, l, "getFlexUser", map[string]property.Value{"instanceSid": sv(""), "flexUserSid": sv("x")})
	require.ErrorContains(t, err, "must be set")

	f, err := invoke(t, l, "getFrontlineUser", map[string]property.Value{"sid": sv(testFrontlineSid)})
	require.NoError(t, err)
	require.Equal(t, "ada@example.com", getS(t, f, "identity"))
	require.Equal(t, "active", getS(t, f, "state"))
	require.False(t, f.Get("isAvailable").AsBool())
	require.False(t, hasProp(f, "avatar"))
	_, err = invoke(t, l, "getFrontlineUser", map[string]property.Value{"sid": sv("USmissing")})
	require.ErrorContains(t, err, "was not found")
}

// with2 merges extra into a copy of base.
func with2(base, extra map[string]property.Value) map[string]property.Value {
	out := map[string]property.Value{}
	for k, v := range base {
		out[k] = v
	}
	for k, v := range extra {
		out[k] = v
	}
	return out
}
