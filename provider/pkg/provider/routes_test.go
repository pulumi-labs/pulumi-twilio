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
	"testing"

	pgp "github.com/pulumi/pulumi-go-provider"
	"github.com/pulumi/pulumi/sdk/v3/go/property"
	"github.com/stretchr/testify/require"
)

func routeIn(kv map[string]string) property.Map {
	m := map[string]property.Value{}
	for k, v := range kv {
		m[k] = sv(v)
	}
	return pv(m)
}

// routesCase drives the shared lifecycle of one Routes resource.
type routesCase struct {
	typ, lookup, id, path string
	keys                  map[string]string // the key inputs
	create, update        map[string]string // settings (field -> value) for create and update
	form                  map[string]string // the form fields create must send
	updateForm            map[string]string
	v2                    bool
}

func (c routesCase) inputs(settings map[string]string) property.Map {
	kv := map[string]string{}
	for k, v := range c.keys {
		kv[k] = v
	}
	for k, v := range settings {
		kv[k] = v
	}
	return routeIn(kv)
}

func requireForm(t *testing.T, l *tl, path string, want map[string]string) {
	t.Helper()
	calls := l.f.callsMatching("POST", path)
	require.NotEmpty(t, calls, "POST %s", path)
	got := map[string]string{}
	for k, v := range calls[len(calls)-1].Form {
		got[k] = v[0]
	}
	require.Equal(t, want, got)
}

func runRoutesLifecycle(t *testing.T, c routesCase) {
	l := newTL(t, c.typ, "r")
	full := "/routes/" + c.path

	// An object Twilio does not know cannot be configured, and a create with nothing to set fails.
	_, err := l.s.Create(pgp.CreateRequest{Urn: l.urn, Properties: c.inputs(c.create)})
	require.ErrorContains(t, err, "was not found")
	l.f.seedRoute(c.path)
	_, err = l.s.Create(pgp.CreateRequest{Urn: l.urn, Properties: c.inputs(nil)})
	require.ErrorContains(t, err, "set at least one")

	in := c.inputs(c.create)
	cr := l.create(in)
	require.Equal(t, c.id, cr.ID)
	require.Equal(t, testAccount, getS(t, cr.Properties, "accountSid"))
	require.NotEmpty(t, getS(t, cr.Properties, "url"))
	require.NotEmpty(t, getS(t, cr.Properties, "dateUpdated"))
	requireForm(t, l, full, c.form)
	requireNoChanges(t, l.diff(cr.ID, cr.Properties, in, in))

	// Refresh sees no drift, then drift made outside Pulumi.
	rd := l.read(cr.ID, cr.Properties, in)
	requireNoChanges(t, l.diff(cr.ID, rd.Properties, in, rd.Inputs))
	var drift string
	for field := range c.create {
		drift = field
	}
	l.f.mutate("routes", testAccount, c.path, func(o map[string]any) { o[routesFakeField(drift)] = "outside" })
	rd = l.read(cr.ID, cr.Properties, in)
	require.Equal(t, "outside", getS(t, rd.Inputs, drift))
	require.True(t, l.diff(cr.ID, rd.Properties, in, rd.Inputs).HasChanges)

	// Update puts the program's settings back, and a changed setting is sent.
	u := l.update(cr.ID, rd.Properties, in, rd.Inputs)
	requireForm(t, l, full, c.form)
	in2 := c.inputs(merge(c.create, c.update))
	d := l.diff(cr.ID, u.Properties, in2, in)
	require.True(t, d.HasChanges)
	require.False(t, d.DeleteBeforeReplace)
	u2 := l.update(cr.ID, u.Properties, in2, in)
	requireForm(t, l, full, c.updateForm)
	for field, v := range c.update {
		require.Equal(t, v, getS(t, u2.Properties, field))
	}

	// A different key replaces.
	for k := range c.keys {
		other := map[string]string{}
		for kk, vv := range c.keys {
			other[kk] = vv
		}
		other[k] = other[k] + "9"
		od := l.diff(cr.ID, u2.Properties, routeIn(merge(other, merge(c.create, c.update))), in2)
		require.True(t, od.HasChanges)
		require.NotEmpty(t, od.DetailedDiff)
		for _, pd := range od.DetailedDiff {
			require.Contains(t, []pgp.DiffKind{pgp.UpdateReplace, pgp.AddReplace, pgp.DeleteReplace}, pd.Kind)
		}
	}

	// Import by ID adopts every setting Twilio reports.
	imp := l.read(c.id, empty(), empty())
	require.Equal(t, c.id, imp.ID)
	for field, v := range c.update {
		require.Equal(t, v, getS(t, imp.Inputs, field))
	}
	for k, v := range c.keys {
		require.Equal(t, v, getS(t, imp.Inputs, k))
	}
	requireNoChanges(t, l.diff(imp.ID, imp.Properties, imp.Inputs, imp.Inputs))
	_, err = l.s.Read(pgp.ReadRequest{ID: "not/a/valid/id/x", Urn: l.urn, Inputs: empty(), Properties: empty()})
	require.ErrorContains(t, err, "invalid")

	// Delete only forgets the setting: no call removes it.
	l.f.resetCalls()
	l.del(cr.ID, u2.Properties)
	require.Zero(t, l.f.callCount())
	require.NotNil(t, l.f.routeObject(c.path))

	// An object that is gone refreshes away and cannot be imported.
	l.f.del("routes", testAccount, c.path)
	require.Empty(t, l.read(cr.ID, u2.Properties, in2).ID)
	l.requireImportFails(c.id)
}

func routesFakeField(field string) string {
	switch field {
	case "voiceRegion":
		return "voice"
	case "messagingRegion":
		return "messaging"
	}
	return "friendly"
}

func merge(a, b map[string]string) map[string]string {
	m := map[string]string{}
	for k, v := range a {
		m[k] = v
	}
	for k, v := range b {
		m[k] = v
	}
	return m
}

func TestRoutesSipDomainLC_Lifecycle(t *testing.T) {
	runRoutesLifecycle(t, routesCase{
		typ: "RoutesSipDomain", id: "example.sip.twilio.com", path: "v2/SipDomains/example.sip.twilio.com", v2: true,
		keys:   map[string]string{"sipDomain": "example.sip.twilio.com"},
		create: map[string]string{"voiceRegion": "ie1"}, update: map[string]string{"friendlyName": "Dublin"},
		form: map[string]string{"VoiceRegion": "ie1"}, updateForm: map[string]string{"VoiceRegion": "ie1", "FriendlyName": "Dublin"},
	})
}

func TestRoutesTrunkLC_Lifecycle(t *testing.T) {
	runRoutesLifecycle(t, routesCase{
		typ: "RoutesTrunk", id: "example.pstn.twilio.com", path: "v2/Trunks/example.pstn.twilio.com", v2: true,
		keys:   map[string]string{"sipTrunkDomain": "example.pstn.twilio.com"},
		create: map[string]string{"voiceRegion": "au1"}, update: map[string]string{"voiceRegion": "sg1"},
		form: map[string]string{"VoiceRegion": "au1"}, updateForm: map[string]string{"VoiceRegion": "sg1"},
	})
}

func TestRoutesPhoneNumberLC_Lifecycle(t *testing.T) {
	runRoutesLifecycle(t, routesCase{
		typ: "RoutesPhoneNumber", id: "+14155550100", path: "v3/PhoneNumbers/+14155550100",
		keys:   map[string]string{"phoneNumber": "+14155550100"},
		create: map[string]string{"voiceRegion": "ie1", "messagingRegion": "ie1"}, update: map[string]string{"messagingRegion": "au1"},
		form: map[string]string{"voiceRegion": "ie1", "messagingRegion": "ie1"}, updateForm: map[string]string{"voiceRegion": "ie1", "messagingRegion": "au1"},
	})
}

func TestRoutesShortCodeLC_Lifecycle(t *testing.T) {
	runRoutesLifecycle(t, routesCase{
		typ: "RoutesShortCode", id: "US/12345", path: "v3/ShortCodes/US/12345",
		keys:   map[string]string{"isoCountryCode": "US", "shortCode": "12345"},
		create: map[string]string{"messagingRegion": "ie1"}, update: map[string]string{"friendlyName": "Promo"},
		form: map[string]string{"messagingRegion": "ie1"}, updateForm: map[string]string{"messagingRegion": "ie1", "friendlyName": "Promo"},
	})
}

// A setting the program leaves to Twilio is not adopted on refresh, so it never becomes a diff.
func TestRoutesSipDomainLC_UnsetSettingsStayUnadopted(t *testing.T) {
	l := newTL(t, "RoutesSipDomain", "r")
	l.f.seedRoute("v2/SipDomains/a.sip.twilio.com")
	in := routeIn(map[string]string{"sipDomain": "a.sip.twilio.com", "voiceRegion": "ie1"})
	cr := l.create(in)
	rd := l.read(cr.ID, cr.Properties, in)
	require.False(t, hasProp(rd.Inputs, "friendlyName"))
	require.False(t, hasProp(rd.Inputs, "messagingRegion"))
	requireNoChanges(t, l.diff(cr.ID, rd.Properties, in, rd.Inputs))
}

func TestRoutesLookups(t *testing.T) {
	l := newTL(t, "RoutesSipDomain", "r")
	for _, p := range []string{"v2/SipDomains/a.sip.twilio.com", "v2/Trunks/a.pstn.twilio.com", "v3/PhoneNumbers/+14155550100", "v3/ShortCodes/US/12345"} {
		l.f.seedRoute(p)
	}
	l.f.mutate("routes", testAccount, "v3/ShortCodes/US/12345", func(o map[string]any) { o["messaging"] = "ie1"; o["friendly"] = "Promo" })

	sd, err := invoke(t, l, "getRoutesSipDomain", map[string]property.Value{"sipDomain": sv("a.sip.twilio.com")})
	require.NoError(t, err)
	require.Equal(t, "us1", getS(t, sd, "voiceRegion"))
	require.Equal(t, testAccount, getS(t, sd, "accountSid"))
	require.NotEmpty(t, getS(t, sd, "sid"))
	tr, err := invoke(t, l, "getRoutesTrunk", map[string]property.Value{"sipTrunkDomain": sv("a.pstn.twilio.com")})
	require.NoError(t, err)
	require.Equal(t, "us1", getS(t, tr, "voiceRegion"))
	pn, err := invoke(t, l, "getRoutesPhoneNumber", map[string]property.Value{"phoneNumber": sv("+14155550100")})
	require.NoError(t, err)
	require.Equal(t, "us1", getS(t, pn, "messagingRegion"))
	require.Equal(t, "us1", getS(t, pn, "voiceRegion"))
	sc, err := invoke(t, l, "getRoutesShortCode", map[string]property.Value{"isoCountryCode": sv("US"), "shortCode": sv("12345")})
	require.NoError(t, err)
	require.Equal(t, "ie1", getS(t, sc, "messagingRegion"))
	require.Equal(t, "Promo", getS(t, sc, "friendlyName"))

	_, err = invoke(t, l, "getRoutesSipDomain", map[string]property.Value{"sipDomain": sv("missing.sip.twilio.com")})
	require.ErrorContains(t, err, "was not found")
	_, err = invoke(t, l, "getRoutesShortCode", map[string]property.Value{"isoCountryCode": sv("US"), "shortCode": sv("")})
	require.ErrorContains(t, err, "must be set")
}
