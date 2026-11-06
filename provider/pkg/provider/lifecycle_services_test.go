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

// ----------------------------------------------------------- MessagingService

func msgInputs() property.Map {
	return pv(map[string]property.Value{
		"friendlyName": sv("notify"), "inboundRequestUrl": sv("https://example.com/in"), "inboundMethod": sv("GET"),
		"statusCallback": sv("https://example.com/status"), "scanMessageContent": sv("enable"),
		"stickySender": bv(false), "validityPeriod": nv(600), "smartEncoding": bv(true),
	})
}

func TestMessagingServiceLC_Lifecycle(t *testing.T) {
	l := newTL(t, "MessagingService", "svc")
	in := msgInputs()
	c := l.create(in)
	sid := c.ID
	require.Equal(t, sid, getS(t, c.Properties, "sid"))
	stored := l.f.get("msg", testAccount, sid)
	require.Equal(t, false, stored["sticky_sender"])
	require.Equal(t, float64(600), stored["validity_period"])
	require.Equal(t, "enable", stored["scan_message_content"])
	form := l.f.callsMatching("POST", "/v1/Services")[0].Form
	require.NotContains(t, form, "MmsConverter", "unset fields are not sent on create")

	// The server's view (including defaults the program did not set) is an output only.
	eff := effective(t, c.Properties)
	require.True(t, eff.Get("mmsConverter").AsBool())
	require.Equal(t, "POST", getS(t, eff, "fallbackMethod"))
	require.False(t, hasProp(c.Properties, "mmsConverter"))

	requireNoChanges(t, l.diff(sid, c.Properties, in, in))
	rd := l.read(sid, c.Properties, in)
	requireNoChanges(t, l.diff(sid, rd.Properties, in, rd.Inputs))

	next := pv(map[string]property.Value{
		"friendlyName": sv("notify-2"), "inboundRequestUrl": sv("https://example.com/in2"), "inboundMethod": sv("POST"),
		"fallbackUrl": sv("https://example.com/fb"), "fallbackMethod": sv("GET"),
		"statusCallback": sv("https://example.com/status2"), "scanMessageContent": sv("disable"),
		"stickySender": bv(true), "mmsConverter": bv(false), "smartEncoding": bv(false), "fallbackToLongCode": bv(false),
		"areaCodeGeomatch": bv(false), "synchronousValidation": bv(true), "useInboundWebhookOnNumber": bv(false),
		"validityPeriod": nv(1200),
	})
	require.True(t, l.diff(sid, rd.Properties, next, in).HasChanges)
	u := l.update(sid, rd.Properties, next, in)
	stored = l.f.get("msg", testAccount, sid)
	for k, v := range map[string]any{
		"friendly_name": "notify-2", "inbound_request_url": "https://example.com/in2", "inbound_method": "POST",
		"fallback_url": "https://example.com/fb", "fallback_method": "GET", "status_callback": "https://example.com/status2",
		"scan_message_content": "disable", "sticky_sender": true, "mms_converter": false, "smart_encoding": false,
		"fallback_to_long_code": false, "area_code_geomatch": false, "synchronous_validation": true,
		"use_inbound_webhook_on_number": false, "validity_period": float64(1200),
	} {
		require.Equal(t, v, stored[k], k)
	}
	requireNoChanges(t, l.diff(sid, u.Properties, next, next))

	// B2: removing bool and int inputs resets them on Twilio (previously they were skipped, so the
	// change never reached the API and the next refresh could not see it).
	min := pv(map[string]property.Value{"friendlyName": sv("notify-2")})
	u2 := l.update(sid, u.Properties, min, next)
	stored = l.f.get("msg", testAccount, sid)
	for k, v := range map[string]any{
		"validity_period": float64(36000), "scan_message_content": "inherit", "inbound_method": "POST", "fallback_method": "POST",
	} {
		require.Equal(t, v, stored[k], k)
	}
	// Unverified defaults are not sent: the flags keep the values of the previous update, and the
	// removal form carries only the field with a default stated in Twilio's OpenAPI definition.
	for k, v := range map[string]any{
		"sticky_sender": true, "mms_converter": false, "smart_encoding": false, "fallback_to_long_code": false,
		"area_code_geomatch": false, "synchronous_validation": true, "use_inbound_webhook_on_number": false,
	} {
		require.Equal(t, v, stored[k], k)
	}
	calls := l.f.callsMatching("POST", "/v1/Services/")
	last := calls[len(calls)-1].Form
	require.Equal(t, "36000", last.Get("ValidityPeriod"), "a verified default is sent")
	for _, k := range []string{"StickySender", "MmsConverter", "SmartEncoding", "FallbackToLongCode",
		"AreaCodeGeomatch", "SynchronousValidation", "UseInboundWebhookOnNumber"} {
		require.NotContains(t, last, k, "an unverified default must not be sent")
	}
	for _, k := range []string{"inbound_request_url", "fallback_url", "status_callback"} {
		require.Nil(t, stored[k], k)
	}
	requireNoChanges(t, l.diff(sid, u2.Properties, min, min))

	l.del(sid, u2.Properties)
	require.Nil(t, l.f.get("msg", testAccount, sid))
	l.del(sid, u2.Properties)
}

func TestMessagingServiceLC_RefreshDriftAndImport(t *testing.T) {
	l := newTL(t, "MessagingService", "svc")
	in := msgInputs()
	c := l.create(in)
	sid := c.ID

	l.f.mutate("msg", testAccount, sid, func(o map[string]any) {
		o["sticky_sender"] = true
		o["validity_period"] = float64(900)
		o["mms_converter"] = false // not in the program
	})
	rd := l.read(sid, c.Properties, in)
	require.True(t, rd.Inputs.Get("stickySender").AsBool())
	require.Equal(t, float64(900), rd.Inputs.Get("validityPeriod").AsNumber())
	require.False(t, hasProp(rd.Inputs, "mmsConverter"))
	require.False(t, effective(t, rd.Properties).Get("mmsConverter").AsBool())
	d := l.diff(sid, rd.Properties, in, rd.Inputs)
	require.Equal(t, pgp.Update, d.DetailedDiff["stickySender"].Kind)
	require.Equal(t, pgp.Update, d.DetailedDiff["validityPeriod"].Kind)
	require.NotContains(t, d.DetailedDiff, "mmsConverter")

	// Import adopts what differs from the defaults.
	imp := l.read(sid, empty(), empty())
	require.Equal(t, sid, imp.ID)
	require.Equal(t, "notify", getS(t, imp.Inputs, "friendlyName"))
	require.False(t, imp.Inputs.Get("mmsConverter").AsBool(), "a flag that differs from the import-only default is adopted (#1595)")
	require.False(t, hasProp(imp.Inputs, "fallbackMethod"), "default methods are not adopted")
	require.False(t, hasProp(imp.Inputs, "areaCodeGeomatch"), "a flag without a verified default is not adopted (#1572)")
	require.NotNil(t, effective(t, imp.Properties).Get("areaCodeGeomatch"), "the effective output still reports it")
	requireNoChanges(t, l.diff(imp.ID, imp.Properties, l.checked(imp.Inputs), imp.Inputs))
}

func TestMessagingServiceLC_DeletedOutsidePulumi(t *testing.T) {
	l := newTL(t, "MessagingService", "svc")
	in := msgInputs()
	c := l.create(in)
	l.f.remove("msg", testAccount, c.ID)
	require.Empty(t, l.read(c.ID, c.Properties, in).ID)
	l.del(c.ID, c.Properties)
}

func TestMessagingServiceLC_PreviewAndUnknowns(t *testing.T) {
	l := newTL(t, "MessagingService", "svc")
	_, err := l.s.Create(pgp.CreateRequest{Urn: l.urn, Properties: msgInputs(), DryRun: true})
	require.NoError(t, err)
	require.Zero(t, l.f.callCount())

	chk := l.check(pv(map[string]property.Value{"friendlyName": computed(), "validityPeriod": computed(), "inboundMethod": computed()}))
	require.Empty(t, chk.Failures)
	require.True(t, chk.Inputs.Get("friendlyName").IsComputed())

	for name, in := range map[string]map[string]property.Value{
		"empty name": {"friendlyName": sv("")}, "bad method": {"friendlyName": sv("x"), "fallbackMethod": sv("PUT")},
		"bad scan": {"friendlyName": sv("x"), "scanMessageContent": sv("maybe")}, "bad validity": {"friendlyName": sv("x"), "validityPeriod": nv(0)},
	} {
		require.NotEmpty(t, l.check(pv(in)).Failures, name)
	}
}

// ------------------------------------------------- MessagingServicePhoneNumber

func newSenderLC(t *testing.T) (*tl, string) {
	l := newTL(t, "MessagingServicePhoneNumber", "sender")
	l.f.addNumber(testAccount, numSid, numE)
	svc := l.other("MessagingService", "svc").create(pv(map[string]property.Value{"friendlyName": sv("pool")}))
	return l, svc.ID
}

func TestSenderLC_Lifecycle(t *testing.T) {
	l, svc := newSenderLC(t)
	in := pv(map[string]property.Value{"serviceSid": sv(svc), "phoneNumberSid": sv(numSid)})
	c := l.create(in)
	require.Equal(t, svc+"/"+numSid, c.ID)
	require.Equal(t, numE, getS(t, c.Properties, "phoneNumber"))
	require.Equal(t, []property.Value{sv("SMS"), sv("MMS")}, c.Properties.Get("capabilities").AsArray().AsSlice())
	require.NotNil(t, l.f.get("sender", svc, numSid))

	requireNoChanges(t, l.diff(c.ID, c.Properties, in, in))
	rd := l.read(c.ID, c.Properties, in)
	require.Equal(t, c.ID, rd.ID)
	requireNoChanges(t, l.diff(c.ID, rd.Properties, in, rd.Inputs))

	// Drift in the reported capabilities refreshes into the output.
	l.f.mutate("sender", svc, numSid, func(o map[string]any) { o["capabilities"] = []any{"SMS"} })
	rd = l.read(c.ID, c.Properties, in)
	require.Len(t, rd.Properties.Get("capabilities").AsArray().AsSlice(), 1)

	// Every input identifies the attachment: changing one replaces it.
	for k, v := range map[string]property.Value{"serviceSid": sv("MGother"), "phoneNumberSid": sv("PNother")} {
		d := l.diff(c.ID, rd.Properties, with(in, map[string]property.Value{k: v}), in)
		require.Equal(t, pgp.UpdateReplace, d.DetailedDiff[k].Kind, k)
	}

	// Import by ID alone, then no change.
	imp := l.read(c.ID, empty(), empty())
	require.Equal(t, c.ID, imp.ID)
	require.Equal(t, svc, getS(t, imp.Inputs, "serviceSid"))
	require.Equal(t, numSid, getS(t, imp.Inputs, "phoneNumberSid"))
	requireNoChanges(t, l.diff(imp.ID, imp.Properties, imp.Inputs, imp.Inputs))
	requireNoChanges(t, l.diff(imp.ID, imp.Properties, in, imp.Inputs))

	// Delete detaches without releasing the number, and is idempotent.
	l.del(c.ID, c.Properties)
	require.Nil(t, l.f.get("sender", svc, numSid))
	require.NotNil(t, l.f.get("number", testAccount, numSid))
	l.del(c.ID, c.Properties)
}

func TestSenderLC_DeletedOutsideAndErrors(t *testing.T) {
	l, svc := newSenderLC(t)
	in := pv(map[string]property.Value{"serviceSid": sv(svc), "phoneNumberSid": sv(numSid)})
	c := l.create(in)

	// The same number twice is a 409, with nothing left half-created.
	_, err := l.s.Create(pgp.CreateRequest{Urn: l.urn, Properties: in})
	require.Error(t, err)

	l.f.remove("sender", svc, numSid)
	require.Empty(t, l.read(c.ID, c.Properties, in).ID)
	l.del(c.ID, c.Properties)

	_, err = l.s.Read(pgp.ReadRequest{ID: "no-slash", Urn: l.urn, Properties: empty(), Inputs: empty()})
	require.ErrorContains(t, err, "invalid ID")

	_, err = l.s.Create(pgp.CreateRequest{Urn: l.urn, Properties: pv(map[string]property.Value{"serviceSid": sv(svc), "phoneNumberSid": sv("PNnone")})})
	require.Error(t, err, "an unknown number")
	_, err = l.s.Create(pgp.CreateRequest{Urn: l.urn, Properties: in, DryRun: true})
	require.NoError(t, err)
}

// --------------------------------------------------------------- VerifyService

func verifyInputs() property.Map {
	return pv(map[string]property.Value{
		"friendlyName": sv("login"), "codeLength": nv(8), "lookupEnabled": bv(true), "skipSmsToLandlines": bv(true),
		"dtmfInputRequired": bv(true), "ttsName": sv("alt"), "psd2Enabled": bv(true),
		"doNotShareWarningEnabled": bv(true), "customCodeEnabled": bv(false),
	})
}

func TestVerifyServiceLC_Lifecycle(t *testing.T) {
	l := newTL(t, "VerifyService", "verify")
	in := verifyInputs()
	c := l.create(in)
	sid := c.ID
	require.Equal(t, sid, getS(t, c.Properties, "sid"))
	stored := l.f.get("verify", testAccount, sid)
	require.Equal(t, float64(8), stored["code_length"])
	require.Equal(t, true, stored["psd2_enabled"])
	require.Equal(t, false, stored["custom_code_enabled"])
	require.Equal(t, "alt", stored["tts_name"])

	requireNoChanges(t, l.diff(sid, c.Properties, in, in))
	rd := l.read(sid, c.Properties, in)
	requireNoChanges(t, l.diff(sid, rd.Properties, in, rd.Inputs))

	next := pv(map[string]property.Value{
		"friendlyName": sv("login-2"), "codeLength": nv(5), "lookupEnabled": bv(false), "skipSmsToLandlines": bv(false),
		"dtmfInputRequired": bv(false), "ttsName": sv("other"), "psd2Enabled": bv(false),
		"doNotShareWarningEnabled": bv(false), "customCodeEnabled": bv(true),
	})
	require.True(t, l.diff(sid, rd.Properties, next, in).HasChanges)
	u := l.update(sid, rd.Properties, next, in)
	stored = l.f.get("verify", testAccount, sid)
	for k, v := range map[string]any{
		"friendly_name": "login-2", "code_length": float64(5), "lookup_enabled": false, "skip_sms_to_landlines": false,
		"dtmf_input_required": false, "tts_name": "other", "psd2_enabled": false, "do_not_share_warning_enabled": false,
		"custom_code_enabled": true,
	} {
		require.Equal(t, v, stored[k], k)
	}
	requireNoChanges(t, l.diff(sid, u.Properties, next, next))

	// B2: removing a bool/int input resets it only when Twilio documents its default.
	prev := verifyInputs()
	u = l.update(sid, u.Properties, prev, next)
	min := pv(map[string]property.Value{"friendlyName": sv("login-2")})
	u2 := l.update(sid, u.Properties, min, prev)
	stored = l.f.get("verify", testAccount, sid)
	// Only doNotShareWarningEnabled has a default stated in Twilio's OpenAPI definition, so it is
	// reset; the other flags and codeLength keep the values of the previous update (not sent).
	for k, v := range map[string]any{
		"code_length": float64(8), "lookup_enabled": true, "skip_sms_to_landlines": true, "dtmf_input_required": true,
		"psd2_enabled": true, "do_not_share_warning_enabled": false, "custom_code_enabled": false,
	} {
		require.Equal(t, v, stored[k], k)
	}
	calls := l.f.callsMatching("POST", "/v2/Services/")
	last := calls[len(calls)-1].Form
	require.Equal(t, "false", last.Get("DoNotShareWarningEnabled"))
	for _, k := range []string{"CodeLength", "LookupEnabled", "SkipSmsToLandlines", "DtmfInputRequired", "Psd2Enabled", "CustomCodeEnabled"} {
		require.NotContains(t, last, k, "an unverified default must not be sent")
	}
	require.Empty(t, stored["tts_name"])
	requireNoChanges(t, l.diff(sid, u2.Properties, min, min))

	l.del(sid, u2.Properties)
	require.Nil(t, l.f.get("verify", testAccount, sid))
	l.del(sid, u2.Properties)
}

func TestVerifyServiceLC_RefreshDriftAndImport(t *testing.T) {
	l := newTL(t, "VerifyService", "verify")
	in := verifyInputs()
	c := l.create(in)
	sid := c.ID

	l.f.mutate("verify", testAccount, sid, func(o map[string]any) {
		o["psd2_enabled"] = false
		o["code_length"] = float64(4)
	})
	rd := l.read(sid, c.Properties, in)
	require.False(t, rd.Inputs.Get("psd2Enabled").AsBool())
	require.Equal(t, float64(4), rd.Inputs.Get("codeLength").AsNumber())
	d := l.diff(sid, rd.Properties, in, rd.Inputs)
	require.Equal(t, pgp.Update, d.DetailedDiff["psd2Enabled"].Kind)
	require.Equal(t, pgp.Update, d.DetailedDiff["codeLength"].Kind)

	// A field the program does not manage drifts into the output only.
	min := pv(map[string]property.Value{"friendlyName": sv("login")})
	c2 := l.other("VerifyService", "plain").create(min)
	l.f.mutate("verify", testAccount, c2.ID, func(o map[string]any) { o["lookup_enabled"] = true })
	rd2 := l.other("VerifyService", "plain").read(c2.ID, c2.Properties, min)
	require.True(t, effective(t, rd2.Properties).Get("lookupEnabled").AsBool())
	require.False(t, hasProp(rd2.Inputs, "lookupEnabled"))
	requireNoChanges(t, l.diff(c2.ID, rd2.Properties, min, rd2.Inputs))

	imp := l.read(sid, empty(), empty())
	require.Equal(t, sid, imp.ID)
	require.Equal(t, "login", getS(t, imp.Inputs, "friendlyName"))
	require.Equal(t, float64(4), imp.Inputs.Get("codeLength").AsNumber(), "a number that differs from the import-only default is adopted (#1595)")
	require.False(t, hasProp(imp.Inputs, "psd2Enabled"), "a flag without a verified default is not adopted (#1572)")
	requireNoChanges(t, l.diff(imp.ID, imp.Properties, l.checked(imp.Inputs), imp.Inputs))
}

func TestVerifyServiceLC_DeletedOutsideAndChecks(t *testing.T) {
	l := newTL(t, "VerifyService", "verify")
	in := verifyInputs()
	c := l.create(in)
	l.f.remove("verify", testAccount, c.ID)
	require.Empty(t, l.read(c.ID, c.Properties, in).ID)
	l.del(c.ID, c.Properties)

	_, err := l.s.Create(pgp.CreateRequest{Urn: l.urn, Properties: in, DryRun: true})
	require.NoError(t, err)

	chk := l.check(pv(map[string]property.Value{"friendlyName": computed(), "codeLength": computed()}))
	require.Empty(t, chk.Failures)
	require.True(t, chk.Inputs.Get("friendlyName").IsComputed())
	require.NotEmpty(t, l.check(pv(map[string]property.Value{"friendlyName": sv("")})).Failures)
	require.NotEmpty(t, l.check(pv(map[string]property.Value{"friendlyName": sv("x"), "codeLength": nv(3)})).Failures)
}
