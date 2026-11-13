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

func iv(n int) property.Value { return property.New(float64(n)) }

func TestProxyServiceLC_Lifecycle(t *testing.T) {
	l := newTL(t, "ProxyService", "svc")
	chat := "IS" + strings.Repeat("c", 32)
	in := pv(map[string]property.Value{"uniqueName": sv("masked"), "defaultTtl": iv(3600), "callbackUrl": sv("https://example.com/cb"),
		"geoMatchLevel": sv("area-code"), "chatInstanceSid": sv(chat)})
	c := l.create(in)
	require.True(t, strings.HasPrefix(c.ID, "KS"), c.ID)
	require.Equal(t, c.ID, getS(t, c.Properties, "sid"))
	form := l.f.callsMatching("POST", "/proxy/v1/Services")[0].Form
	require.Equal(t, "masked", form.Get("UniqueName"))
	require.Equal(t, "3600", form.Get("DefaultTtl"))
	require.Equal(t, chat, form.Get("ChatInstanceSid"))
	require.Empty(t, form["NumberSelectionBehavior"])
	requireNoChanges(t, l.diff(c.ID, c.Properties, in, in))
	rd := l.read(c.ID, c.Properties, in)
	requireNoChanges(t, l.diff(c.ID, rd.Properties, in, rd.Inputs))

	// Drift in a setting the program states shows; a setting it leaves out does not.
	l.f.mutate("px", testAccount, "Services/"+c.ID, func(o map[string]any) {
		o["default_ttl"] = float64(60)
		o["number_selection_behavior"] = "avoid-sticky"
	})
	rd = l.read(c.ID, c.Properties, in)
	require.Equal(t, 60, int(rd.Inputs.Get("defaultTtl").AsNumber()))
	require.Equal(t, pgp.Update, l.diff(c.ID, rd.Properties, in, rd.Inputs).DetailedDiff["defaultTtl"].Kind)
	require.False(t, hasProp(rd.Inputs, "numberSelectionBehavior"))

	next := with(in, map[string]property.Value{"uniqueName": sv("masked 2"), "numberSelectionBehavior": sv("avoid-sticky"), "interceptCallbackUrl": sv("https://example.com/i")})
	d := l.diff(c.ID, c.Properties, next, in)
	require.Equal(t, pgp.Update, d.DetailedDiff["uniqueName"].Kind)
	require.Equal(t, pgp.Update, d.DetailedDiff["numberSelectionBehavior"].Kind)
	l.f.resetCalls()
	u := l.update(c.ID, c.Properties, next, in)
	require.Equal(t, "masked 2", l.f.callsMatching("POST", "/proxy/v1/Services/"+c.ID)[0].Form.Get("UniqueName"))
	require.Equal(t, "avoid-sticky", l.f.get("px", testAccount, "Services/"+c.ID)["number_selection_behavior"])
	requireNoChanges(t, l.diff(c.ID, u.Properties, next, next))

	// Import adopts what differs from the defaults; the defaults stated explicitly are no change.
	l.f.mutate("px", testAccount, "Services/"+c.ID, func(o map[string]any) {
		o["number_selection_behavior"] = "prefer-sticky"
		o["default_ttl"] = float64(0)
	})
	imp := l.read(c.ID, empty(), empty())
	require.Empty(t, l.check(imp.Inputs).Failures)
	require.Equal(t, "masked 2", getS(t, imp.Inputs, "uniqueName"))
	require.Equal(t, "area-code", getS(t, imp.Inputs, "geoMatchLevel"))
	require.False(t, hasProp(imp.Inputs, "numberSelectionBehavior"))
	require.False(t, hasProp(imp.Inputs, "defaultTtl"))
	explicit := with(imp.Inputs, map[string]property.Value{"numberSelectionBehavior": sv("prefer-sticky"), "defaultTtl": iv(0)})
	requireNoChanges(t, l.diff(c.ID, imp.Properties, explicit, imp.Inputs))
	l.requireImportFails("KS" + strings.Repeat("0", 32))

	l.f.remove("px", testAccount, "Services/"+c.ID)
	require.Empty(t, l.read(c.ID, u.Properties, next).ID)
	l.del(c.ID, u.Properties)
	c = l.create(in)
	l.del(c.ID, c.Properties)
	require.Nil(t, l.f.get("px", testAccount, "Services/"+c.ID))

	bad := l.check(pv(map[string]property.Value{"uniqueName": sv(strings.Repeat("x", 192)), "defaultTtl": iv(-1), "geoMatchLevel": sv("city"),
		"numberSelectionBehavior": sv("random"), "chatInstanceSid": sv("nope")}))
	require.Len(t, bad.Failures, 5)
}

func TestProxyPhoneNumberLC_Lifecycle(t *testing.T) {
	l := newTL(t, "ProxyPhoneNumber", "num")
	svc := l.other("ProxyService", "svc").create(pv(map[string]property.Value{"uniqueName": sv("masked")})).ID
	in := pv(map[string]property.Value{"serviceSid": sv(svc), "phoneNumber": sv("+14155551234"), "isReserved": bv(true)})
	c := l.create(in)
	sid := getS(t, c.Properties, "sid")
	require.Equal(t, svc+"/"+sid, c.ID)
	form := l.f.callsMatching("POST", "/proxy/v1/Services/"+svc+"/PhoneNumbers")[0].Form
	require.Equal(t, "+14155551234", form.Get("PhoneNumber"))
	require.Equal(t, "true", form.Get("IsReserved"))
	require.Empty(t, form["Sid"])
	// The state names the number both ways, so a program that uses the SID instead is no change.
	require.Equal(t, sid, getS(t, c.Properties, "phoneNumberSid"))
	requireNoChanges(t, l.diff(c.ID, c.Properties, in, in))
	bySid := pv(map[string]property.Value{"serviceSid": sv(svc), "phoneNumberSid": sv(sid), "isReserved": bv(true)})
	requireNoChanges(t, l.diff(c.ID, c.Properties, bySid, in))
	rd := l.read(c.ID, c.Properties, in)
	requireNoChanges(t, l.diff(c.ID, rd.Properties, in, rd.Inputs))

	// Drift in the reservation shows.
	l.f.mutate("px", testAccount, "Services/"+svc+"/PhoneNumbers/"+sid, func(o map[string]any) { o["is_reserved"] = false })
	rd = l.read(c.ID, c.Properties, in)
	require.False(t, rd.Inputs.Get("isReserved").AsBool())
	require.Equal(t, pgp.Update, l.diff(c.ID, rd.Properties, in, rd.Inputs).DetailedDiff["isReserved"].Kind)

	// Only the reservation updates in place; the number and the service replace.
	next := with(in, map[string]property.Value{"isReserved": bv(false)})
	require.Equal(t, pgp.Update, l.diff(c.ID, c.Properties, next, in).DetailedDiff["isReserved"].Kind)
	l.f.resetCalls()
	u := l.update(c.ID, c.Properties, next, in)
	uf := l.f.callsMatching("POST", "/proxy/v1/Services/"+svc+"/PhoneNumbers/"+sid)[0].Form
	require.Equal(t, "false", uf.Get("IsReserved"))
	require.Empty(t, uf["PhoneNumber"])
	require.Equal(t, "+14155551234", getS(t, u.Properties, "phoneNumber"))
	requireNoChanges(t, l.diff(c.ID, u.Properties, next, next))
	require.Equal(t, pgp.UpdateReplace, l.diff(c.ID, u.Properties, with(next, map[string]property.Value{"phoneNumber": sv("+14155559999")}), next).DetailedDiff["phoneNumber"].Kind)
	require.Equal(t, pgp.UpdateReplace, l.diff(c.ID, u.Properties, with(next, map[string]property.Value{"serviceSid": sv("KS" + strings.Repeat("9", 32))}), next).DetailedDiff["serviceSid"].Kind)

	// Import by <serviceSid>/<sid> adopts the SID (the state keeps the number); the default reservation stated explicitly is no change.
	imp := l.read(c.ID, empty(), empty())
	require.Empty(t, l.check(imp.Inputs).Failures)
	require.Equal(t, svc, getS(t, imp.Inputs, "serviceSid"))
	require.Equal(t, sid, getS(t, imp.Inputs, "phoneNumberSid"))
	// An import adopts only the SID as the input (Check refuses both); the state keeps the number.
	require.False(t, hasProp(imp.Inputs, "phoneNumber"))
	require.Equal(t, "+14155551234", getS(t, imp.Properties, "phoneNumber"))
	require.Empty(t, l.check(imp.Inputs).Failures)
	require.False(t, hasProp(imp.Inputs, "isReserved"))
	// The program may name the number either way after an import without a change.
	byNumber := pv(map[string]property.Value{"serviceSid": sv(svc), "phoneNumber": sv("+14155551234")})
	require.Empty(t, l.check(byNumber).Failures)
	requireNoChanges(t, l.diff(c.ID, imp.Properties, byNumber, imp.Inputs))
	requireNoChanges(t, l.diff(c.ID, imp.Properties, with(imp.Inputs, map[string]property.Value{"isReserved": bv(false)}), imp.Inputs))
	l.requireImportFails(svc + "/PN" + strings.Repeat("0", 32))
	_, err := l.s.Read(pgp.ReadRequest{ID: "badid", Urn: l.urn, Properties: empty(), Inputs: empty()})
	require.ErrorContains(t, err, "<serviceSid>/<sid>")

	l.f.remove("px", testAccount, "Services/"+svc+"/PhoneNumbers/"+sid)
	require.Empty(t, l.read(c.ID, u.Properties, next).ID)
	l.del(c.ID, u.Properties)

	// A number can be added by the SID of an IncomingPhoneNumber too.
	pn := "PN" + strings.Repeat("a", 32)
	c = l.create(pv(map[string]property.Value{"serviceSid": sv(svc), "phoneNumberSid": sv(pn)}))
	require.Equal(t, svc+"/"+pn, c.ID)
	require.Equal(t, pn, l.f.callsMatching("POST", "/proxy/v1/Services/"+svc+"/PhoneNumbers")[1].Form.Get("Sid"))
	require.NotEmpty(t, getS(t, c.Properties, "phoneNumber"))
	l.del(c.ID, c.Properties)
	require.Nil(t, l.f.get("px", testAccount, "Services/"+svc+"/PhoneNumbers/"+pn))

	ok := l.check(pv(map[string]property.Value{"serviceSid": sv(svc), "phoneNumber": sv("+14155551234")}))
	require.Empty(t, ok.Failures)
	require.Len(t, l.check(pv(map[string]property.Value{"serviceSid": sv(svc)})).Failures, 1)
	require.Len(t, l.check(pv(map[string]property.Value{"serviceSid": sv(svc), "phoneNumber": sv("+14155551234"), "phoneNumberSid": sv(pn)})).Failures, 1)
	require.Len(t, l.check(pv(map[string]property.Value{"serviceSid": sv("nope"), "phoneNumber": sv("4155551234")})).Failures, 2)
	require.Len(t, l.check(pv(map[string]property.Value{"serviceSid": sv(svc), "phoneNumberSid": sv("nope")})).Failures, 1)
}

func TestLookup_Proxy(t *testing.T) {
	l := newTL(t, "ProxyService", "svc")
	svc := l.create(pv(map[string]property.Value{"uniqueName": sv("masked"), "defaultTtl": iv(10)})).ID
	r, err := invoke(t, l, "getProxyService", map[string]property.Value{"sid": sv(svc)})
	require.NoError(t, err)
	require.Equal(t, "masked", getS(t, r, "uniqueName"))
	require.Equal(t, "country", getS(t, r, "geoMatchLevel"))
	require.Equal(t, 10, int(r.Get("defaultTtl").AsNumber()))
	require.False(t, hasProp(r, "callbackUrl"))
	_, err = invoke(t, l, "getProxyService", map[string]property.Value{"sid": sv("KSmissing")})
	require.ErrorContains(t, err, "was not found")
	_, err = invoke(t, l, "getProxyService", map[string]property.Value{"sid": sv("")})
	require.ErrorContains(t, err, "must be set")

	num := l.other("ProxyPhoneNumber", "num").create(pv(map[string]property.Value{"serviceSid": sv(svc), "phoneNumber": sv("+14155551234")}))
	sid := getS(t, num.Properties, "sid")
	r, err = invoke(t, l, "getProxyPhoneNumber", map[string]property.Value{"serviceSid": sv(svc), "sid": sv(sid)})
	require.NoError(t, err)
	require.Equal(t, "+14155551234", getS(t, r, "phoneNumber"))
	require.Equal(t, "US", getS(t, r, "isoCountry"))
	require.False(t, r.Get("isReserved").AsBool())
	_, err = invoke(t, l, "getProxyPhoneNumber", map[string]property.Value{"serviceSid": sv(svc), "sid": sv("PNmissing")})
	require.ErrorContains(t, err, "was not found")
	_, err = invoke(t, l, "getProxyPhoneNumber", map[string]property.Value{"serviceSid": sv(""), "sid": sv(sid)})
	require.ErrorContains(t, err, "must be set")
}
