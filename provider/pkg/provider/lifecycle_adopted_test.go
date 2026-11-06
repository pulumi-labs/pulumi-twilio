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

const (
	scSid  = "SC500"
	scCode = "894546"
	cnSid  = "CN500"
	cidSid = "PN600"
	cidE   = "+14155550123"
)

func strList(vs ...string) property.Value {
	l := make([]property.Value, len(vs))
	for i, v := range vs {
		l[i] = property.New(v)
	}
	return property.New(l)
}

// ------------------------------------------------------------- ShortCode

func newShortCodeLC(t *testing.T) *tl {
	l := newTL(t, "ShortCode", "sc")
	l.f.addShortCode(testAccount, scSid, scCode)
	return l
}

func TestShortCodeLC_Lifecycle(t *testing.T) {
	l := newShortCodeLC(t)
	in := pv(map[string]property.Value{"code": sv(scCode), "smsUrl": sv("https://example.com/sms"), "smsMethod": sv("GET")})
	c := l.create(in)
	require.Equal(t, scSid, c.ID)
	require.Equal(t, scCode, getS(t, c.Properties, "number"))
	require.Equal(t, testAccount, getS(t, c.Properties, "ownerAccountSid"))
	stored := l.f.get("shortcode", testAccount, scSid)
	require.Equal(t, "https://example.com/sms", stored["sms_url"])
	require.Equal(t, "GET", stored["sms_method"])
	require.Equal(t, "Short code "+scCode, getS(t, effective(t, c.Properties), "friendlyName"), "the label is an output until the program sets it")
	requireNoChanges(t, l.diff(c.ID, c.Properties, in, in))
	rd := l.read(c.ID, c.Properties, in)
	requireNoChanges(t, l.diff(c.ID, rd.Properties, in, rd.Inputs))

	next := pv(map[string]property.Value{"code": sv(scCode), "friendlyName": sv("alerts"), "smsUrl": sv("https://example.com/s2"),
		"smsMethod": sv("POST"), "smsFallbackUrl": sv("https://example.com/sf"), "smsFallbackMethod": sv("GET")})
	d := l.diff(c.ID, rd.Properties, next, in)
	require.True(t, d.HasChanges)
	require.False(t, d.DeleteBeforeReplace)
	u := l.update(c.ID, rd.Properties, next, in)
	stored = l.f.get("shortcode", testAccount, scSid)
	for k, v := range map[string]any{"friendly_name": "alerts", "sms_url": "https://example.com/s2", "sms_method": "POST",
		"sms_fallback_url": "https://example.com/sf", "sms_fallback_method": "GET"} {
		require.Equal(t, v, stored[k], k)
	}
	requireNoChanges(t, l.diff(c.ID, u.Properties, next, next))

	// Removed inputs: URLs are cleared, methods go back to POST, the label is left alone.
	min := pv(map[string]property.Value{"code": sv(scCode)})
	l.f.resetCalls()
	u2 := l.update(c.ID, u.Properties, min, next)
	stored = l.f.get("shortcode", testAccount, scSid)
	require.Nil(t, stored["sms_url"])
	require.Nil(t, stored["sms_fallback_url"])
	require.Equal(t, "POST", stored["sms_fallback_method"])
	require.Equal(t, "alerts", stored["friendly_name"])
	require.NotContains(t, l.f.callsMatching("POST", "SMS/ShortCodes/")[0].Form, "FriendlyName")
	requireNoChanges(t, l.diff(c.ID, u2.Properties, min, min))

	// The identity is replace-on-change.
	d = l.diff(c.ID, u2.Properties, pv(map[string]property.Value{"code": sv("55555")}), min)
	require.Equal(t, pgp.UpdateReplace, d.DetailedDiff["code"].Kind)
	d = l.diff(c.ID, u2.Properties, with(min, map[string]property.Value{"accountSid": sv(fakeSub)}), min)
	require.Equal(t, pgp.AddReplace, d.DetailedDiff["accountSid"].Kind)

	// Delete never releases the short code and makes no API call.
	l.f.resetCalls()
	l.del(c.ID, u2.Properties)
	require.Zero(t, l.f.callCount())
	require.NotNil(t, l.f.get("shortcode", testAccount, scSid))
}

func TestShortCodeLC_RefreshImportAndDeletedOutside(t *testing.T) {
	l := newShortCodeLC(t)
	in := pv(map[string]property.Value{"shortCodeSid": sv(scSid), "smsUrl": sv("https://example.com/sms")})
	c := l.create(in)
	l.f.mutate("shortcode", testAccount, scSid, func(o map[string]any) { o["sms_url"] = "https://drift.example.com" })
	rd := l.read(c.ID, c.Properties, in)
	require.Equal(t, "https://drift.example.com", getS(t, rd.Inputs, "smsUrl"))
	require.Equal(t, pgp.Update, l.diff(c.ID, rd.Properties, in, rd.Inputs).DetailedDiff["smsUrl"].Kind)

	imp := l.read(scSid, empty(), empty())
	require.Equal(t, scSid, imp.ID)
	require.Equal(t, scSid, getS(t, imp.Inputs, "shortCodeSid"))
	require.Equal(t, "https://drift.example.com", getS(t, imp.Inputs, "smsUrl"))
	require.False(t, hasProp(imp.Inputs, "smsMethod"), "default methods are not adopted")
	require.False(t, hasProp(imp.Inputs, "apiVersion"))
	require.False(t, hasProp(imp.Inputs, "accountSid"))
	requireNoChanges(t, l.diff(imp.ID, imp.Properties, l.checked(imp.Inputs), imp.Inputs))

	l.f.remove("shortcode", testAccount, scSid)
	require.Empty(t, l.read(c.ID, c.Properties, in).ID)
	l.del(c.ID, c.Properties)
}

func TestShortCodeLC_SubaccountAndFailures(t *testing.T) {
	l := newTL(t, "ShortCode", "sc")
	l.f.addSubaccount(fakeSub)
	l.f.addShortCode(fakeSub, "SC77", "311311")
	c := l.create(pv(map[string]property.Value{"code": sv("311311"), "accountSid": sv(fakeSub)}))
	require.Equal(t, fakeSub+"/SC77", c.ID)
	require.Equal(t, fakeSub, getS(t, c.Properties, "ownerAccountSid"))
	imp := l.read(fakeSub+"/SC77", empty(), empty())
	require.Equal(t, fakeSub, getS(t, imp.Inputs, "accountSid"))
	requireNoChanges(t, l.diff(imp.ID, imp.Properties, l.checked(imp.Inputs), imp.Inputs))
	require.Empty(t, l.read("SC77", empty(), empty()).ID, "a bare SID is looked up in the provider's account")

	// Not leased by the account.
	_, err := l.s.Create(pgp.CreateRequest{Urn: l.urn, Properties: l.checked(pv(map[string]property.Value{"code": sv("1")}))})
	require.ErrorContains(t, err, "adopts an existing short code")
	_, err = l.s.Create(pgp.CreateRequest{Urn: l.urn, Properties: l.checked(pv(map[string]property.Value{"shortCodeSid": sv("SCnone")}))})
	require.ErrorContains(t, err, "has no short code with SID")
	_, err = l.s.Create(pgp.CreateRequest{Urn: l.urn, Properties: pv(map[string]property.Value{"code": sv("311311"), "accountSid": sv(fakeSub)}), DryRun: true})
	require.NoError(t, err)

	// Check: exactly one identity, computed counts as set, methods validated.
	require.Empty(t, l.check(pv(map[string]property.Value{"code": computed()})).Failures)
	require.NotEmpty(t, l.check(pv(map[string]property.Value{})).Failures)
	require.NotEmpty(t, l.check(pv(map[string]property.Value{"code": sv("1"), "shortCodeSid": sv("SC1")})).Failures)
	require.NotEmpty(t, l.check(pv(map[string]property.Value{"code": sv("1"), "smsMethod": sv("PUT")})).Failures)
	require.Empty(t, l.check(pv(map[string]property.Value{"code": sv("1"), "smsMethod": computed()})).Failures)
}

func TestShortCodeLC_ConfigureFailureKeepsPartialState(t *testing.T) {
	l := newShortCodeLC(t)
	l.f.failNext("POST", "SMS/ShortCodes/"+scSid, 500)
	resp, err := l.s.Create(pgp.CreateRequest{Urn: l.urn, Properties: l.checked(pv(map[string]property.Value{"code": sv(scCode), "smsUrl": sv("https://example.com/sms")}))})
	require.Error(t, err)
	require.NotNil(t, resp.PartialState)
	require.Equal(t, scSid, resp.ID)
	require.Contains(t, resp.PartialState.Reasons[0], "configure short code")
}

// ------------------------------------------------------------- ConnectApp

func newConnectAppLC(t *testing.T) *tl {
	l := newTL(t, "ConnectApp", "app")
	l.f.addConnectApp(testAccount, cnSid)
	return l
}

func TestConnectAppLC_Lifecycle(t *testing.T) {
	l := newConnectAppLC(t)
	in := pv(map[string]property.Value{"connectAppSid": sv(cnSid), "companyName": sv("Acme Inc"), "permissions": strList("get-all", "post-all"),
		"deauthorizeCallbackUrl": sv("https://example.com/deauth"), "deauthorizeCallbackMethod": sv("GET")})
	c := l.create(in)
	require.Equal(t, cnSid, c.ID)
	stored := l.f.get("connectapp", testAccount, cnSid)
	require.Equal(t, "Acme Inc", stored["company_name"])
	require.Equal(t, []any{"get-all", "post-all"}, stored["permissions"])
	require.Equal(t, "GET", stored["deauthorize_callback_method"])
	require.Equal(t, "console app", getS(t, effective(t, c.Properties), "friendlyName"))
	requireNoChanges(t, l.diff(c.ID, c.Properties, in, in))
	rd := l.read(c.ID, c.Properties, in)
	requireNoChanges(t, l.diff(c.ID, rd.Properties, in, rd.Inputs))

	next := pv(map[string]property.Value{"connectAppSid": sv(cnSid), "friendlyName": sv("renamed"), "companyName": sv("Acme Inc"),
		"description": sv("New text"), "homepageUrl": sv("https://acme.example.com"), "authorizeRedirectUrl": sv("https://acme.example.com/cb"),
		"permissions": strList("get-all")})
	u := l.update(c.ID, rd.Properties, next, in)
	stored = l.f.get("connectapp", testAccount, cnSid)
	for k, v := range map[string]any{"friendly_name": "renamed", "description": "New text", "homepage_url": "https://acme.example.com",
		"authorize_redirect_url": "https://acme.example.com/cb", "permissions": []any{"get-all"}, "deauthorize_callback_method": "POST"} {
		require.Equal(t, v, stored[k], k)
	}
	require.Nil(t, stored["deauthorize_callback_url"])
	requireNoChanges(t, l.diff(c.ID, u.Properties, next, next))

	// The listing details and the permissions are left alone when removed from the program.
	min := pv(map[string]property.Value{"connectAppSid": sv(cnSid)})
	l.f.resetCalls()
	u2 := l.update(c.ID, u.Properties, min, next)
	stored = l.f.get("connectapp", testAccount, cnSid)
	require.Equal(t, "New text", stored["description"])
	require.Equal(t, []any{"get-all"}, stored["permissions"])
	for _, p := range []string{"FriendlyName", "Description", "CompanyName", "HomepageUrl", "AuthorizeRedirectUrl", "Permissions"} {
		for _, call := range l.f.callsMatching("POST", "ConnectApps/") {
			require.NotContains(t, call.Form, p)
		}
	}
	requireNoChanges(t, l.diff(c.ID, u2.Properties, min, min))

	d := l.diff(c.ID, u2.Properties, pv(map[string]property.Value{"connectAppSid": sv("CN1")}), min)
	require.Equal(t, pgp.UpdateReplace, d.DetailedDiff["connectAppSid"].Kind)

	l.f.resetCalls()
	l.del(c.ID, u2.Properties)
	require.Zero(t, l.f.callCount(), "destroy must not delete the app")
	require.NotNil(t, l.f.get("connectapp", testAccount, cnSid))
}

func TestConnectAppLC_ImportCheckAndMissing(t *testing.T) {
	l := newConnectAppLC(t)
	imp := l.read(cnSid, empty(), empty())
	require.Equal(t, cnSid, getS(t, imp.Inputs, "connectAppSid"))
	require.Equal(t, "Acme", getS(t, imp.Inputs, "companyName"))
	require.Equal(t, "console app", getS(t, imp.Inputs, "friendlyName"))
	require.False(t, hasProp(imp.Inputs, "deauthorizeCallbackMethod"))
	require.True(t, hasProp(imp.Inputs, "permissions"))
	requireNoChanges(t, l.diff(imp.ID, imp.Properties, l.checked(imp.Inputs), imp.Inputs))

	l.f.remove("connectapp", testAccount, cnSid)
	require.Empty(t, l.read(cnSid, empty(), empty()).ID)
	_, err := l.s.Create(pgp.CreateRequest{Urn: l.urn, Properties: l.checked(pv(map[string]property.Value{"connectAppSid": sv("CNnone")}))})
	require.ErrorContains(t, err, "has no Connect app with SID")

	require.NotEmpty(t, l.check(pv(map[string]property.Value{})).Failures)
	require.Empty(t, l.check(pv(map[string]property.Value{"connectAppSid": computed()})).Failures)
	require.NotEmpty(t, l.check(pv(map[string]property.Value{"connectAppSid": sv(cnSid), "permissions": strList("delete-all")})).Failures)
	require.NotEmpty(t, l.check(pv(map[string]property.Value{"connectAppSid": sv(cnSid), "deauthorizeCallbackMethod": sv("PUT")})).Failures)
}

// ------------------------------------------------------------- OutgoingCallerId

func newCallerIDLC(t *testing.T) *tl {
	l := newTL(t, "OutgoingCallerId", "cid")
	l.f.addCallerID(testAccount, cidSid, cidE)
	return l
}

func TestCallerIDLC_Lifecycle(t *testing.T) {
	l := newCallerIDLC(t)
	in := pv(map[string]property.Value{"phoneNumber": sv(cidE), "friendlyName": sv("main line")})
	c := l.create(in)
	require.Equal(t, cidSid, c.ID)
	require.Equal(t, cidE, getS(t, c.Properties, "number"))
	require.Equal(t, "main line", l.f.get("callerid", testAccount, cidSid)["friendly_name"])
	requireNoChanges(t, l.diff(c.ID, c.Properties, in, in))
	rd := l.read(c.ID, c.Properties, in)
	requireNoChanges(t, l.diff(c.ID, rd.Properties, in, rd.Inputs))

	next := pv(map[string]property.Value{"phoneNumber": sv(cidE), "friendlyName": sv("billing line")})
	u := l.update(c.ID, rd.Properties, next, in)
	require.Equal(t, "billing line", l.f.get("callerid", testAccount, cidSid)["friendly_name"])
	requireNoChanges(t, l.diff(c.ID, u.Properties, next, next))

	min := pv(map[string]property.Value{"phoneNumber": sv(cidE)})
	l.f.resetCalls()
	u2 := l.update(c.ID, u.Properties, min, next)
	require.Zero(t, l.f.callCount(), "a removed label is left alone, so nothing is sent")
	requireNoChanges(t, l.diff(c.ID, u2.Properties, min, min))

	d := l.diff(c.ID, u2.Properties, pv(map[string]property.Value{"phoneNumber": sv("+14155550999")}), min)
	require.Equal(t, pgp.UpdateReplace, d.DetailedDiff["phoneNumber"].Kind)
	d = l.diff(c.ID, u2.Properties, with(min, map[string]property.Value{"outgoingCallerIdSid": sv("PN1")}), min)
	require.Equal(t, pgp.AddReplace, d.DetailedDiff["outgoingCallerIdSid"].Kind)

	l.f.resetCalls()
	l.del(c.ID, u2.Properties)
	require.Zero(t, l.f.callCount(), "destroy must not remove the verified caller ID")
	require.NotNil(t, l.f.get("callerid", testAccount, cidSid))
}

func TestCallerIDLC_ImportSubaccountAndMissing(t *testing.T) {
	l := newCallerIDLC(t)
	imp := l.read(cidSid, empty(), empty())
	require.Equal(t, cidSid, getS(t, imp.Inputs, "outgoingCallerIdSid"))
	require.Equal(t, cidE, getS(t, imp.Properties, "number"))
	requireNoChanges(t, l.diff(imp.ID, imp.Properties, l.checked(imp.Inputs), imp.Inputs))

	l.f.addSubaccount(fakeSub)
	l.f.addCallerID(fakeSub, "PN66", "+14155550166")
	c := l.create(pv(map[string]property.Value{"outgoingCallerIdSid": sv("PN66"), "accountSid": sv(fakeSub)}))
	require.Equal(t, fakeSub+"/PN66", c.ID)
	sub := l.read(c.ID, empty(), empty())
	require.Equal(t, fakeSub, getS(t, sub.Inputs, "accountSid"))

	l.f.remove("callerid", testAccount, cidSid)
	require.Empty(t, l.read(cidSid, empty(), empty()).ID)
	_, err := l.s.Create(pgp.CreateRequest{Urn: l.urn, Properties: l.checked(pv(map[string]property.Value{"phoneNumber": sv(cidE)}))})
	require.ErrorContains(t, err, "adopts an existing verified caller ID")

	require.NotEmpty(t, l.check(pv(map[string]property.Value{})).Failures)
	require.NotEmpty(t, l.check(pv(map[string]property.Value{"phoneNumber": sv(cidE), "outgoingCallerIdSid": sv("PN1")})).Failures)
	require.Empty(t, l.check(pv(map[string]property.Value{"phoneNumber": computed()})).Failures)
}

// ------------------------------------------------------------- lookups

func TestLookup_ShortCodeConnectAppAndCallerID(t *testing.T) {
	l := newTL(t, "ShortCode", "sc")
	l.f.addShortCode(testAccount, scSid, scCode)
	l.f.addConnectApp(testAccount, cnSid)
	l.f.addCallerID(testAccount, cidSid, cidE)
	l.f.addSubaccount(fakeSub)
	l.f.addShortCode(fakeSub, "SC77", "311311")

	for _, args := range []map[string]property.Value{{"code": sv(scCode)}, {"sid": sv(scSid)}} {
		r, err := invoke(t, l, "getShortCode", args)
		require.NoError(t, err)
		require.Equal(t, scSid, getS(t, r, "sid"))
		require.Equal(t, scCode, getS(t, r, "code"))
		require.Equal(t, testAccount, getS(t, r, "accountSid"))
		require.Equal(t, "POST", getS(t, r, "smsMethod"))
	}
	r, err := invoke(t, l, "getShortCode", map[string]property.Value{"code": sv("311311"), "accountSid": sv(fakeSub)})
	require.NoError(t, err)
	require.Equal(t, "SC77", getS(t, r, "sid"))
	_, err = invoke(t, l, "getShortCode", map[string]property.Value{"code": sv("311311")})
	require.ErrorContains(t, err, "has no short code")
	_, err = invoke(t, l, "getShortCode", map[string]property.Value{"code": sv(scCode), "sid": sv(scSid)})
	require.ErrorContains(t, err, "exactly one")
	_, err = invoke(t, l, "getShortCode", map[string]property.Value{})
	require.ErrorContains(t, err, "exactly one")

	r, err = invoke(t, l, "getConnectApp", map[string]property.Value{"sid": sv(cnSid)})
	require.NoError(t, err)
	require.Equal(t, "Acme", getS(t, r, "companyName"))
	require.Equal(t, "https://example.com/authorize", getS(t, r, "authorizeRedirectUrl"))
	require.Equal(t, 1, r.Get("permissions").AsArray().Len())
	_, err = invoke(t, l, "getConnectApp", map[string]property.Value{"sid": sv("CNnone")})
	require.ErrorContains(t, err, "has no Connect app")
	_, err = invoke(t, l, "getConnectApp", map[string]property.Value{"sid": sv("")})
	require.ErrorContains(t, err, "must be set")

	for _, args := range []map[string]property.Value{{"phoneNumber": sv(cidE)}, {"sid": sv(cidSid)}} {
		r, err = invoke(t, l, "getOutgoingCallerId", args)
		require.NoError(t, err)
		require.Equal(t, cidSid, getS(t, r, "sid"))
		require.Equal(t, cidE, getS(t, r, "phoneNumber"))
	}
	_, err = invoke(t, l, "getOutgoingCallerId", map[string]property.Value{"phoneNumber": sv("+19995550000")})
	require.ErrorContains(t, err, "has no verified caller ID")
	_, err = invoke(t, l, "getOutgoingCallerId", map[string]property.Value{})
	require.ErrorContains(t, err, "exactly one")
}

func TestWiring_AdoptedResourcesPreviewKeepsIdentifiersKnown(t *testing.T) {
	cases := []struct {
		typ      string
		setup    func(l *tl)
		in, next property.Map
	}{
		{"ShortCode", func(l *tl) { l.f.addShortCode(testAccount, scSid, scCode) },
			pv(map[string]property.Value{"code": sv(scCode), "smsUrl": sv("https://example.com/a")}),
			pv(map[string]property.Value{"code": sv(scCode), "smsUrl": sv("https://example.com/s")})},
		{"ConnectApp", func(l *tl) { l.f.addConnectApp(testAccount, cnSid) },
			pv(map[string]property.Value{"connectAppSid": sv(cnSid), "friendlyName": sv("w")}),
			pv(map[string]property.Value{"connectAppSid": sv(cnSid), "friendlyName": sv("x")})},
		{"OutgoingCallerId", func(l *tl) { l.f.addCallerID(testAccount, cidSid, cidE) },
			pv(map[string]property.Value{"phoneNumber": sv(cidE), "friendlyName": sv("w")}),
			pv(map[string]property.Value{"phoneNumber": sv(cidE), "friendlyName": sv("x")})},
	}
	for _, c := range cases {
		t.Run(c.typ, func(t *testing.T) {
			l := newTL(t, c.typ, "r")
			c.setup(l)
			created := l.create(c.in)
			u := previewUpdate(t, l, created.ID, created.Properties, c.in, c.next)
			requireKnown(t, u.Properties, "sid", "ownerAccountSid")
			requireComputed(t, u.Properties, "effective")
		})
	}
}
