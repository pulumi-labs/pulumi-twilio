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

// ---------------------------------------------------------------- Subaccount

func TestSubaccountLC_Lifecycle(t *testing.T) {
	l := newTL(t, "Subaccount", "sub")
	in := l.checked(pv(map[string]property.Value{"friendlyName": sv("tenant-a")}))
	require.Equal(t, "active", getS(t, in, "status"), "Check applies the active default")

	c := l.create(in)
	sid := c.ID
	require.Equal(t, sid, getS(t, c.Properties, "sid"))
	require.Equal(t, testAccount, getS(t, c.Properties, "ownerAccountSid"))
	require.Equal(t, "tenant-a", l.f.get("account", testAccount, sid)["friendly_name"])

	// An unchanged program plans nothing, before and after a refresh.
	requireNoChanges(t, l.diff(sid, c.Properties, in, in))
	rd := l.read(sid, c.Properties, in)
	require.Equal(t, sid, rd.ID)
	requireNoChanges(t, l.diff(sid, rd.Properties, in, rd.Inputs))

	// Rename and suspend both reach the API.
	next := with(in, map[string]property.Value{"friendlyName": sv("tenant-b"), "status": sv("suspended")})
	d := l.diff(sid, rd.Properties, next, in)
	require.Equal(t, pgp.Update, d.DetailedDiff["friendlyName"].Kind)
	require.Equal(t, pgp.Update, d.DetailedDiff["status"].Kind)
	u := l.update(sid, rd.Properties, next, in)
	require.Equal(t, "tenant-b", l.f.get("account", testAccount, sid)["friendly_name"])
	require.Equal(t, "suspended", l.f.get("account", testAccount, sid)["status"])
	requireNoChanges(t, l.diff(sid, u.Properties, next, next))

	// Drift: a rename in the console.
	l.f.mutate("account", testAccount, sid, func(o map[string]any) { o["friendly_name"] = "console" })
	rd = l.read(sid, u.Properties, next)
	require.Equal(t, "console", getS(t, rd.Inputs, "friendlyName"))
	require.Equal(t, pgp.Update, l.diff(sid, rd.Properties, next, rd.Inputs).DetailedDiff["friendlyName"].Kind)

	// Closing outside Pulumi drops the resource.
	l.f.mutate("account", testAccount, sid, func(o map[string]any) { o["status"] = "closed" })
	require.Empty(t, l.read(sid, u.Properties, next).ID)

	// Delete closes (Twilio cannot delete) and is idempotent.
	l.del(sid, u.Properties)
	require.Equal(t, "closed", l.f.get("account", testAccount, sid)["status"])
	l.del(sid, u.Properties)
}

func TestSubaccountLC_ImportThenNoChange(t *testing.T) {
	l := newTL(t, "Subaccount", "sub")
	l.f.addSubaccount(fakeSub)
	imp := l.read(fakeSub, empty(), empty())
	require.Equal(t, fakeSub, imp.ID)
	require.Equal(t, "sub", getS(t, imp.Inputs, "friendlyName"))
	require.Equal(t, "active", getS(t, imp.Inputs, "status"))
	require.Equal(t, "tok-"+fakeSub, getS(t, imp.Properties, "authToken"))
	requireNoChanges(t, l.diff(imp.ID, imp.Properties, l.checked(imp.Inputs), imp.Inputs))
}

func TestSubaccountLC_DeletedOutsidePulumi(t *testing.T) {
	l := newTL(t, "Subaccount", "sub")
	in := pv(map[string]property.Value{"friendlyName": sv("x")})
	c := l.create(in)
	l.f.remove("account", testAccount, c.ID)
	require.Empty(t, l.read(c.ID, c.Properties, l.checked(in)).ID)
	l.del(c.ID, c.Properties) // the 404 is success
}

// B9: the top-level account must not be adoptable as a subaccount, because
// destroying the resource would close it.
func TestSubaccountLC_TopLevelAccountIsRefused(t *testing.T) {
	l := newTL(t, "Subaccount", "sub")
	_, err := l.s.Read(pgp.ReadRequest{ID: testAccount, Urn: l.urn, Properties: empty(), Inputs: empty()})
	require.ErrorContains(t, err, "top-level account")
	err = l.s.Delete(pgp.DeleteRequest{ID: testAccount, Urn: l.urn, Properties: pv(map[string]property.Value{
		"sid": sv(testAccount), "friendlyName": sv("main"), "status": sv("active"), "authToken": sv("t"),
		"ownerAccountSid": sv(testAccount), "dateCreated": sv("d")})})
	require.ErrorContains(t, err, "refusing to close")
	require.ErrorContains(t, err, "pulumi state delete")
	require.Empty(t, l.f.callsMatching("POST", "/Accounts/"+testAccount), "no request may reach the main account")
}

// A failure after the subaccount exists keeps it in state instead of leaking it.
func TestSubaccountLC_SuspendFailureKeepsPartialState(t *testing.T) {
	l := newTL(t, "Subaccount", "sub")
	l.f.failNext("POST", "/Accounts/AC", 500)
	resp, err := l.s.Create(pgp.CreateRequest{Urn: l.urn, Properties: l.checked(pv(map[string]property.Value{
		"friendlyName": sv("x"), "status": sv("suspended")}))})
	require.Error(t, err)
	require.NotNil(t, resp.PartialState)
	require.NotEmpty(t, resp.ID)
	require.Contains(t, resp.PartialState.Reasons[0], "suspend new subaccount")
	require.NotNil(t, l.f.get("account", testAccount, resp.ID), "the subaccount exists")
	require.Equal(t, "active", getS(t, resp.Properties, "status"), "state shows what Twilio really has")
	require.Equal(t, resp.ID, getS(t, resp.Properties, "sid"))
	require.True(t, hasProp(resp.Properties, "authToken"))
}

func TestSubaccountLC_PreviewAndUnknowns(t *testing.T) {
	l := newTL(t, "Subaccount", "sub")
	c, err := l.s.Create(pgp.CreateRequest{Urn: l.urn, Properties: l.checked(pv(map[string]property.Value{"friendlyName": sv("x")})), DryRun: true})
	require.NoError(t, err)
	require.Empty(t, l.f.callCount(), "preview must not call the API")
	require.NotEmpty(t, c.ID)

	// B1: friendlyName and status computed at preview.
	chk := l.check(pv(map[string]property.Value{"friendlyName": computed(), "status": computed()}))
	require.Empty(t, chk.Failures)
	require.True(t, chk.Inputs.Get("friendlyName").IsComputed(), "the unknown must be passed through, not replaced by a zero value")
	require.True(t, chk.Inputs.Get("status").IsComputed())

	// Known bad values still fail.
	require.NotEmpty(t, l.check(pv(map[string]property.Value{"friendlyName": sv(" ")})).Failures)
	require.NotEmpty(t, l.check(pv(map[string]property.Value{"friendlyName": sv("x"), "status": sv("closed")})).Failures)
}

// -------------------------------------------------------------- Application

func appInputs() property.Map {
	return pv(map[string]property.Value{
		"friendlyName": sv("ivr"), "voiceUrl": sv("https://example.com/voice"), "voiceMethod": sv("GET"),
		"smsUrl": sv("https://example.com/sms"), "voiceCallerIdLookup": bv(true),
	})
}

func TestApplicationLC_Lifecycle(t *testing.T) {
	l := newTL(t, "Application", "app")
	in := appInputs()
	c := l.create(in)
	sid := c.ID
	require.Equal(t, sid, getS(t, c.Properties, "sid"))
	require.Equal(t, testAccount, getS(t, c.Properties, "ownerAccountSid"))
	stored := l.f.get("app", testAccount, sid)
	require.Equal(t, "GET", stored["voice_method"])
	require.Equal(t, true, stored["voice_caller_id_lookup"])
	form := l.f.callsMatching("POST", "/Applications.json")[0].Form
	require.Equal(t, "GET", form.Get("VoiceMethod"))
	require.NotContains(t, form, "SmsMethod", "unset fields are not sent on create")

	// B10: the server's view is an output, the input stays as the program wrote it.
	eff := effective(t, c.Properties)
	require.Equal(t, "POST", getS(t, eff, "smsMethod"))
	require.Equal(t, "https://example.com/sms", getS(t, eff, "smsUrl"))
	require.False(t, hasProp(c.Properties, "smsMethod"), "a server default must not overwrite an unset input")

	requireNoChanges(t, l.diff(sid, c.Properties, in, in))
	rd := l.read(sid, c.Properties, in)
	requireNoChanges(t, l.diff(sid, rd.Properties, in, rd.Inputs))
	require.False(t, hasProp(rd.Inputs, "smsMethod"))

	// Every updatable property reaches the API.
	next := pv(map[string]property.Value{
		"friendlyName": sv("ivr-2"), "voiceUrl": sv("https://example.com/v2"), "voiceMethod": sv("POST"),
		"voiceFallbackUrl": sv("https://example.com/vf"), "voiceFallbackMethod": sv("GET"),
		"statusCallback": sv("https://example.com/sc"), "statusCallbackMethod": sv("GET"),
		"smsUrl": sv("https://example.com/s2"), "smsMethod": sv("GET"),
		"smsFallbackUrl": sv("https://example.com/sf"), "smsFallbackMethod": sv("GET"),
		"smsStatusCallback": sv("https://example.com/ss"), "messageStatusCallback": sv("https://example.com/ms"),
		"voiceCallerIdLookup": bv(false),
	})
	d := l.diff(sid, rd.Properties, next, in)
	require.True(t, d.HasChanges)
	require.False(t, d.DeleteBeforeReplace)
	u := l.update(sid, rd.Properties, next, in)
	stored = l.f.get("app", testAccount, sid)
	for k, v := range map[string]any{
		"friendly_name": "ivr-2", "voice_url": "https://example.com/v2", "voice_method": "POST",
		"voice_fallback_url": "https://example.com/vf", "voice_fallback_method": "GET",
		"status_callback": "https://example.com/sc", "status_callback_method": "GET",
		"sms_url": "https://example.com/s2", "sms_method": "GET", "sms_fallback_url": "https://example.com/sf",
		"sms_fallback_method": "GET", "sms_status_callback": "https://example.com/ss",
		"message_status_callback": "https://example.com/ms", "voice_caller_id_lookup": false,
	} {
		require.Equal(t, v, stored[k], k)
	}
	requireNoChanges(t, l.diff(sid, u.Properties, next, next))
	require.Equal(t, "ivr-2", getS(t, effective(t, u.Properties), "friendlyName"))

	// B2/B7: removing inputs resets them on Twilio. Methods go back to POST (the API rejects ""),
	// URLs are cleared and the bool returns to its default.
	min := pv(map[string]property.Value{"friendlyName": sv("ivr-2")})
	u2 := l.update(sid, u.Properties, min, next)
	stored = l.f.get("app", testAccount, sid)
	for _, k := range []string{"voice_url", "voice_fallback_url", "status_callback", "sms_url", "sms_fallback_url", "sms_status_callback", "message_status_callback"} {
		require.Nil(t, stored[k], k)
	}
	for _, k := range []string{"voice_method", "voice_fallback_method", "status_callback_method", "sms_method", "sms_fallback_method"} {
		require.Equal(t, "POST", stored[k], k)
	}
	require.Equal(t, false, stored["voice_caller_id_lookup"], "removed bool without a verified default keeps its value")
	require.NotContains(t, l.f.callsMatching("POST", "/Applications/")[len(l.f.callsMatching("POST", "/Applications/"))-1].Form, "VoiceCallerIdLookup")
	requireNoChanges(t, l.diff(sid, u2.Properties, min, min))

	// accountSid cannot change in place.
	d = l.diff(sid, u2.Properties, with(min, map[string]property.Value{"accountSid": sv(fakeSub)}), min)
	require.Equal(t, pgp.AddReplace, d.DetailedDiff["accountSid"].Kind)

	l.del(sid, u2.Properties)
	require.Nil(t, l.f.get("app", testAccount, sid))
	l.del(sid, u2.Properties) // 404 is success
}

func TestApplicationLC_RefreshDriftAndImport(t *testing.T) {
	l := newTL(t, "Application", "app")
	in := appInputs()
	c := l.create(in)
	sid := c.ID

	// Drift on a field the program set is an input diff.
	l.f.mutate("app", testAccount, sid, func(o map[string]any) { o["voice_url"] = "https://drift.example.com" })
	rd := l.read(sid, c.Properties, in)
	require.Equal(t, "https://drift.example.com", getS(t, rd.Inputs, "voiceUrl"))
	require.Equal(t, pgp.Update, l.diff(sid, rd.Properties, in, rd.Inputs).DetailedDiff["voiceUrl"].Kind)

	// B10: a webhook added in the console to a field the program did not set shows in the
	// output without turning into a diff.
	l.f.mutate("app", testAccount, sid, func(o map[string]any) {
		o["voice_url"] = "https://example.com/voice"
		o["status_callback"] = "https://console.example.com/status"
	})
	rd = l.read(sid, c.Properties, in)
	require.Equal(t, "https://console.example.com/status", getS(t, effective(t, rd.Properties), "statusCallback"))
	require.False(t, hasProp(rd.Inputs, "statusCallback"))
	requireNoChanges(t, l.diff(sid, rd.Properties, in, rd.Inputs))

	// Import by ID alone adopts what differs from the defaults; the program that created it plans nothing.
	imp := l.read(sid, empty(), empty())
	require.Equal(t, sid, imp.ID)
	require.Equal(t, "ivr", getS(t, imp.Inputs, "friendlyName"))
	require.Equal(t, "GET", getS(t, imp.Inputs, "voiceMethod"))
	require.False(t, hasProp(imp.Inputs, "smsMethod"), "default methods are not adopted")
	require.False(t, hasProp(imp.Inputs, "accountSid"))
	require.Equal(t, testAccount, getS(t, imp.Properties, "ownerAccountSid"))
	requireNoChanges(t, l.diff(imp.ID, imp.Properties, l.checked(imp.Inputs), imp.Inputs))
	prog := with(in, map[string]property.Value{"statusCallback": sv("https://console.example.com/status")})
	requireNoChanges(t, l.diff(imp.ID, imp.Properties, prog, imp.Inputs))
}

func TestApplicationLC_DeletedOutsidePulumi(t *testing.T) {
	l := newTL(t, "Application", "app")
	in := appInputs()
	c := l.create(in)
	l.f.remove("app", testAccount, c.ID)
	require.Empty(t, l.read(c.ID, c.Properties, in).ID)
	l.del(c.ID, c.Properties)
}

// B3: an application in a subaccount is importable by <accountSid>/<sid>, and state written by
// earlier versions (a bare SID plus ownerAccountSid) keeps working.
func TestApplicationLC_SubaccountIDAndImport(t *testing.T) {
	l := newTL(t, "Application", "app")
	l.f.addSubaccount(fakeSub)
	in := with(appInputs(), map[string]property.Value{"accountSid": sv(fakeSub)})
	c := l.create(in)
	_, sid := splitScoped(c.ID)
	require.Equal(t, fakeSub+"/"+sid, c.ID)
	require.Equal(t, fakeSub, getS(t, c.Properties, "ownerAccountSid"))
	require.NotNil(t, l.f.get("app", fakeSub, sid))
	require.Nil(t, l.f.get("app", testAccount, sid))
	requireNoChanges(t, l.diff(c.ID, c.Properties, in, in))

	rd := l.read(c.ID, c.Properties, in)
	require.Equal(t, c.ID, rd.ID)
	requireNoChanges(t, l.diff(c.ID, rd.Properties, in, rd.Inputs))

	imp := l.read(c.ID, empty(), empty())
	require.Equal(t, c.ID, imp.ID)
	require.Equal(t, fakeSub, getS(t, imp.Inputs, "accountSid"), "a program with accountSid plans nothing after import")
	requireNoChanges(t, l.diff(imp.ID, imp.Properties, l.checked(imp.Inputs), imp.Inputs))
	requireNoChanges(t, l.diff(imp.ID, imp.Properties, in, imp.Inputs))

	// A bare SID cannot name a subaccount's object (it is looked up in the provider's account).
	require.Empty(t, l.read(sid, empty(), empty()).ID)

	// State from before the ID format changed: bare SID, owner in state.
	old := l.read(sid, c.Properties, in)
	require.Equal(t, sid, old.ID)

	u := l.update(c.ID, c.Properties, with(in, map[string]property.Value{"friendlyName": sv("renamed")}), in)
	require.Equal(t, "renamed", l.f.get("app", fakeSub, sid)["friendly_name"])
	l.del(c.ID, u.Properties)
	require.Nil(t, l.f.get("app", fakeSub, sid))
	l.del(c.ID, u.Properties)
}

func TestApplicationLC_PreviewAndUnknowns(t *testing.T) {
	l := newTL(t, "Application", "app")
	_, err := l.s.Create(pgp.CreateRequest{Urn: l.urn, Properties: appInputs(), DryRun: true})
	require.NoError(t, err)
	require.Zero(t, l.f.callCount())

	chk := l.check(pv(map[string]property.Value{"friendlyName": computed(), "voiceMethod": computed()}))
	require.Empty(t, chk.Failures)
	require.True(t, chk.Inputs.Get("friendlyName").IsComputed())

	require.NotEmpty(t, l.check(pv(map[string]property.Value{"friendlyName": sv("")})).Failures)
	require.NotEmpty(t, l.check(pv(map[string]property.Value{"friendlyName": sv("x"), "voiceMethod": sv("PUT")})).Failures)
}
