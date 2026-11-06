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
	"net/url"
	"testing"

	"github.com/pulumi/pulumi/sdk/v3/go/property"
	"github.com/stretchr/testify/require"
)

const verifyTemplate = "HJ0123456789abcdef0123456789abcdef"

func TestObjDottedLookup(t *testing.T) {
	o := obj{
		"literal.key": "wins",
		"totp":        map[string]any{"issuer": "acme", "skew": float64(2), "enabled": true},
		"passkeys": map[string]any{"relying_party": map[string]any{
			"id": "example.com", "origins": []any{"https://a", "https://b"}, "csv": "https://c, https://d"}},
	}
	require.Equal(t, "wins", o.str("literal.key"))
	require.Equal(t, "acme", o.str("totp.issuer"))
	require.Equal(t, 2, *o.optInt("totp.skew"))
	require.True(t, *o.optBool("totp.enabled"))
	require.Equal(t, "example.com", o.str("passkeys.relying_party.id"))
	require.Nil(t, o.optStr("totp.missing"))
	require.Nil(t, o.optStr("push.apn_credential_sid"), "a missing parent object reads as unset")
	require.Nil(t, o.optStr("totp.issuer.deeper"))
	require.Equal(t, []string{"https://a", "https://b"}, *o.optList("passkeys.relying_party.origins"))
	require.Equal(t, []string{"https://c", "https://d"}, *o.optList("passkeys.relying_party.csv"))
	require.Nil(t, o.optList("passkeys.relying_party.none"))
}

func TestListPropEncodeResetAdoptRefresh(t *testing.T) {
	set := []string{"https://a", "https://b"}
	a := VerifyServiceArgs{PasskeysRelyingPartyOrigins: &set}
	form := url.Values{}
	a.props().encode(form, nil)
	require.Equal(t, "https://a,https://b", form.Get("Passkeys.RelyingParty.Origins"))

	// Removing the list resets it; a list that was never set sends nothing.
	prev := a.props()
	form = url.Values{}
	(&VerifyServiceArgs{}).props().encode(form, &prev)
	v, ok := form["Passkeys.RelyingParty.Origins"]
	require.True(t, ok)
	require.Equal(t, []string{""}, v)
	form = url.Values{}
	empty := VerifyServiceArgs{}
	ep := empty.props()
	empty.props().encode(form, &ep)
	require.NotContains(t, form, "Passkeys.RelyingParty.Origins")

	resp := obj{"passkeys": map[string]any{"relying_party": map[string]any{"origins": []any{"https://z"}}}}
	imp := VerifyServiceArgs{}
	imp.props().adopt(resp)
	require.Equal(t, []string{"https://z"}, *imp.PasskeysRelyingPartyOrigins)
	a.props().refreshSet(resp)
	require.Equal(t, []string{"https://z"}, *a.PasskeysRelyingPartyOrigins)
	a.props().refreshSet(obj{})
	require.Nil(t, a.PasskeysRelyingPartyOrigins, "a list cleared outside Pulumi shows as drift")
}

func TestApplicationLC_FieldGaps(t *testing.T) {
	l := newTL(t, "Application", "app")
	in := with(appInputs(), map[string]property.Value{
		"apiVersion": sv("2010-04-01"), "publicApplicationConnectEnabled": bv(true),
	})
	c := l.create(in)
	stored := l.f.get("app", testAccount, c.ID)
	require.Equal(t, "2010-04-01", stored["api_version"])
	require.Equal(t, true, stored["public_application_connect_enabled"])
	require.True(t, effective(t, c.Properties).Get("publicApplicationConnectEnabled").AsBool())
	requireNoChanges(t, l.diff(c.ID, c.Properties, in, in))

	// Drift on a set field is detected.
	l.f.mutate("app", testAccount, c.ID, func(o map[string]any) { o["public_application_connect_enabled"] = false })
	rd := l.read(c.ID, c.Properties, in)
	require.False(t, rd.Inputs.Get("publicApplicationConnectEnabled").AsBool())
	require.True(t, l.diff(c.ID, rd.Properties, in, rd.Inputs).HasChanges)

	// Update in place, then removal leaves the unreset fields untouched.
	next := with(in, map[string]property.Value{"apiVersion": sv("2010-04-01"), "publicApplicationConnectEnabled": bv(false)})
	u := l.update(c.ID, rd.Properties, next, in)
	require.Equal(t, false, l.f.get("app", testAccount, c.ID)["public_application_connect_enabled"])
	min := pv(map[string]property.Value{"friendlyName": sv("ivr")})
	u = l.update(c.ID, u.Properties, min, next)
	stored = l.f.get("app", testAccount, c.ID)
	require.Equal(t, "2010-04-01", stored["api_version"], "apiVersion has no valid empty value and is not reset")
	last := l.f.callsMatching("POST", "/Applications/")
	require.NotContains(t, last[len(last)-1].Form, "ApiVersion")
	require.NotContains(t, last[len(last)-1].Form, "PublicApplicationConnectEnabled")
	requireNoChanges(t, l.diff(c.ID, u.Properties, min, min))

	// An import does not adopt the API version Twilio always returns.
	imp := l.read(c.ID, empty(), empty())
	require.False(t, hasProp(imp.Inputs, "apiVersion"))
	require.False(t, hasProp(imp.Inputs, "publicApplicationConnectEnabled"), "a bool without a verified default is not adopted (#1572)")
}

func TestNumberLC_FieldGaps(t *testing.T) {
	l := newNumberLC(t)
	in := with(numberInputs(), map[string]property.Value{
		"trunkSid": sv("TK1"), "voiceReceiveMode": sv("fax"), "addressSid": sv("AD1"), "identitySid": sv("RI1"),
		"bundleSid": sv("BU1"), "emergencyStatus": sv("Active"), "emergencyAddressSid": sv("AD2"), "apiVersion": sv("2010-04-01"),
	})
	c := l.create(in)
	stored := l.f.get("number", testAccount, numSid)
	for k, v := range map[string]any{
		"trunk_sid": "TK1", "voice_receive_mode": "fax", "address_sid": "AD1", "identity_sid": "RI1", "bundle_sid": "BU1",
		"emergency_status": "Active", "emergency_address_sid": "AD2", "api_version": "2010-04-01",
	} {
		require.Equal(t, v, stored[k], k)
	}
	require.Equal(t, "TK1", getS(t, effective(t, c.Properties), "trunkSid"))
	requireNoChanges(t, l.diff(c.ID, c.Properties, in, in))

	l.f.mutate("number", testAccount, numSid, func(o map[string]any) { o["emergency_status"] = "Inactive" })
	rd := l.read(c.ID, c.Properties, in)
	require.Equal(t, "Inactive", getS(t, rd.Inputs, "emergencyStatus"))
	require.True(t, l.diff(c.ID, rd.Properties, in, rd.Inputs).HasChanges)

	next := with(in, map[string]property.Value{"trunkSid": sv("TK2"), "voiceReceiveMode": sv("voice"), "emergencyStatus": sv("Active")})
	u := l.update(c.ID, rd.Properties, next, in)
	stored = l.f.get("number", testAccount, numSid)
	require.Equal(t, "TK2", stored["trunk_sid"])
	require.Equal(t, "voice", stored["voice_receive_mode"])
	requireNoChanges(t, l.diff(c.ID, u.Properties, next, next))

	// Removal: SID references are unassigned; enums and the API version are left alone.
	min := pv(map[string]property.Value{"phoneNumber": sv(numE)})
	u = l.update(c.ID, u.Properties, min, next)
	stored = l.f.get("number", testAccount, numSid)
	for _, k := range []string{"trunk_sid", "address_sid", "identity_sid", "bundle_sid", "emergency_address_sid"} {
		require.Nil(t, stored[k], k)
	}
	require.Equal(t, "voice", stored["voice_receive_mode"])
	require.Equal(t, "Active", stored["emergency_status"])
	require.Equal(t, "2010-04-01", stored["api_version"])
	last := l.f.callsMatching("POST", "/IncomingPhoneNumbers/")
	form := last[len(last)-1].Form
	require.Equal(t, []string{""}, form["TrunkSid"])
	for _, k := range []string{"VoiceReceiveMode", "EmergencyStatus", "ApiVersion"} {
		require.NotContains(t, form, k)
	}
	requireNoChanges(t, l.diff(c.ID, u.Properties, min, min))

	// Import: always-present server values (the API version, default enums) are not inputs.
	l.f.mutate("number", testAccount, numSid, func(o map[string]any) {
		o["voice_receive_mode"] = "voice"
		o["emergency_status"] = "Inactive"
		o["trunk_sid"] = "TK9"
	})
	imp := l.read(c.ID, empty(), empty())
	require.Equal(t, "TK9", getS(t, imp.Inputs, "trunkSid"))
	for _, k := range []string{"apiVersion", "voiceReceiveMode", "emergencyStatus"} {
		require.False(t, hasProp(imp.Inputs, k), k)
	}

	chk := l.check(with(numberInputs(), map[string]property.Value{"voiceReceiveMode": sv("sms"), "emergencyStatus": sv("on")}))
	require.Len(t, chk.Failures, 2)
	require.Empty(t, l.check(with(numberInputs(), map[string]property.Value{"voiceReceiveMode": computed()})).Failures)
}

func TestMessagingServiceLC_Usecase(t *testing.T) {
	l := newTL(t, "MessagingService", "svc")
	in := with(msgInputs(), map[string]property.Value{"usecase": sv("marketing")})
	c := l.create(in)
	require.Equal(t, "marketing", l.f.get("msg", testAccount, c.ID)["usecase"])
	requireNoChanges(t, l.diff(c.ID, c.Properties, in, in))

	l.f.mutate("msg", testAccount, c.ID, func(o map[string]any) { o["usecase"] = "notifications" })
	rd := l.read(c.ID, c.Properties, in)
	require.Equal(t, "notifications", getS(t, rd.Inputs, "usecase"))
	require.True(t, l.diff(c.ID, rd.Properties, in, rd.Inputs).HasChanges)

	next := with(in, map[string]property.Value{"usecase": sv("verification")})
	u := l.update(c.ID, rd.Properties, next, in)
	require.Equal(t, "verification", l.f.get("msg", testAccount, c.ID)["usecase"])

	min := pv(map[string]property.Value{"friendlyName": sv("notify")})
	u = l.update(c.ID, u.Properties, min, next)
	require.Equal(t, "verification", l.f.get("msg", testAccount, c.ID)["usecase"], "usecase is not reset")
	last := l.f.callsMatching("POST", "/v1/Services/")
	require.NotContains(t, last[len(last)-1].Form, "Usecase")
	requireNoChanges(t, l.diff(c.ID, u.Properties, min, min))

	imp := l.read(c.ID, empty(), empty())
	require.Equal(t, "verification", getS(t, imp.Inputs, "usecase"))
}

func verifyGapInputs() property.Map {
	return with(verifyInputs(), map[string]property.Value{
		"defaultTemplateSid": sv(verifyTemplate), "verifyEventSubscriptionEnabled": bv(true),
		"pushApnCredentialSid": sv("CR1"), "pushFcmCredentialSid": sv("CR2"), "pushIncludeDate": bv(true),
		"totpIssuer": sv("acme"), "totpCodeLength": nv(7), "totpTimeStep": nv(60), "totpSkew": nv(2),
		"whatsappMsgServiceSid": sv("MG1"), "whatsappFrom": sv("+14155550100"),
		"passkeysRelyingPartyId": sv("example.com"), "passkeysRelyingPartyName": sv("Example"),
		"passkeysRelyingPartyOrigins":     property.New(property.NewArray([]property.Value{sv("https://a.example.com"), sv("https://b.example.com")})),
		"passkeysAuthenticatorAttachment": sv("platform"), "passkeysDiscoverableCredentials": sv("preferred"),
		"passkeysUserVerification": sv("required"),
	})
}

func TestVerifyServiceLC_FieldGaps(t *testing.T) {
	l := newTL(t, "VerifyService", "verify")
	in := verifyGapInputs()
	c := l.create(in)
	sid := c.ID
	posts := l.f.callsMatching("POST", "/v2/Services")
	form := posts[0].Form
	for k, v := range map[string]string{
		"DefaultTemplateSid": verifyTemplate, "VerifyEventSubscriptionEnabled": "true", "Push.ApnCredentialSid": "CR1",
		"Push.FcmCredentialSid": "CR2", "Push.IncludeDate": "true", "Totp.Issuer": "acme", "Totp.CodeLength": "7",
		"Totp.TimeStep": "60", "Totp.Skew": "2", "Whatsapp.MsgServiceSid": "MG1", "Whatsapp.From": "+14155550100",
		"Passkeys.RelyingParty.Id": "example.com", "Passkeys.RelyingParty.Name": "Example",
		"Passkeys.RelyingParty.Origins":    "https://a.example.com,https://b.example.com",
		"Passkeys.AuthenticatorAttachment": "platform", "Passkeys.DiscoverableCredentials": "preferred",
		"Passkeys.UserVerification": "required",
	} {
		require.Equal(t, v, form.Get(k), k)
	}
	stored := l.f.get("verify", testAccount, sid)
	require.Equal(t, "acme", stored["totp"].(map[string]any)["issuer"])
	require.Equal(t, "CR1", stored["push"].(map[string]any)["apn_credential_sid"])
	rp := stored["passkeys"].(map[string]any)["relying_party"].(map[string]any)
	require.Equal(t, []any{"https://a.example.com", "https://b.example.com"}, rp["origins"])
	require.Equal(t, 7, int(effective(t, c.Properties).Get("totpCodeLength").AsNumber()))

	requireNoChanges(t, l.diff(sid, c.Properties, in, in))
	rd := l.read(sid, c.Properties, in)
	requireNoChanges(t, l.diff(sid, rd.Properties, in, rd.Inputs))

	// Drift inside nested objects is detected on the set fields.
	l.f.mutate("verify", testAccount, sid, func(o map[string]any) {
		o["totp"].(map[string]any)["skew"] = float64(1)
		o["whatsapp"].(map[string]any)["from"] = "+14155550199"
		o["passkeys"].(map[string]any)["relying_party"].(map[string]any)["origins"] = []any{"https://c.example.com"}
	})
	rd = l.read(sid, c.Properties, in)
	d := l.diff(sid, rd.Properties, in, rd.Inputs)
	for _, k := range []string{"totpSkew", "whatsappFrom", "passkeysRelyingPartyOrigins[0]"} {
		require.Contains(t, d.DetailedDiff, k)
	}

	next := with(in, map[string]property.Value{"totpIssuer": sv("beta"), "pushIncludeDate": bv(false), "passkeysUserVerification": sv("discouraged")})
	u := l.update(sid, rd.Properties, next, in)
	stored = l.f.get("verify", testAccount, sid)
	require.Equal(t, "beta", stored["totp"].(map[string]any)["issuer"])
	require.Equal(t, false, stored["push"].(map[string]any)["include_date"])
	require.Equal(t, "discouraged", stored["passkeys"].(map[string]any)["user_verification"])
	requireNoChanges(t, l.diff(sid, u.Properties, next, next))

	// Removal resets references and strings, never the template or unverified bools/ints.
	min := pv(map[string]property.Value{"friendlyName": sv("login")})
	u = l.update(sid, u.Properties, min, next)
	stored = l.f.get("verify", testAccount, sid)
	require.Nil(t, stored["push"].(map[string]any)["apn_credential_sid"])
	require.Nil(t, stored["whatsapp"].(map[string]any)["msg_service_sid"])
	require.Equal(t, "beta", stored["totp"].(map[string]any)["issuer"], "totpIssuer is not reset: Twilio rejects an empty issuer (#1633)")
	require.NotContains(t, l.f.callsMatching("POST", "/v2/Services/")[len(l.f.callsMatching("POST", "/v2/Services/"))-1].Form, "Totp.Issuer")
	require.Equal(t, []any{}, stored["passkeys"].(map[string]any)["relying_party"].(map[string]any)["origins"])
	require.Equal(t, verifyTemplate, stored["default_template_sid"])
	require.Equal(t, float64(2), stored["totp"].(map[string]any)["skew"])
	posts = l.f.callsMatching("POST", "/v2/Services/")
	last := posts[len(posts)-1].Form
	for _, k := range []string{"DefaultTemplateSid", "Totp.Skew", "Totp.CodeLength", "Totp.TimeStep", "Push.IncludeDate", "VerifyEventSubscriptionEnabled"} {
		require.NotContains(t, last, k)
	}
	requireNoChanges(t, l.diff(sid, u.Properties, min, min))

	// Import adopts the nested settings.
	imp := l.read(sid, empty(), empty())
	require.Equal(t, verifyTemplate, getS(t, imp.Inputs, "defaultTemplateSid"))
	require.False(t, hasProp(imp.Inputs, "totpSkew"), "an int without a verified default is not adopted (#1572)")
	origins := imp.Inputs.Get("passkeysRelyingPartyOrigins")
	require.True(t, origins.IsNull() || origins.AsArray().Len() == 0)
	requireNoChanges(t, l.diff(imp.ID, imp.Properties, l.checked(imp.Inputs), imp.Inputs))
}

func TestVerifyServiceLC_FieldGapChecks(t *testing.T) {
	l := newTL(t, "VerifyService", "verify")
	bad := with(verifyInputs(), map[string]property.Value{"defaultTemplateSid": sv("HJ123"), "totpCodeLength": nv(2)})
	require.Len(t, l.check(bad).Failures, 2)
	require.Empty(t, l.check(with(verifyInputs(), map[string]property.Value{"defaultTemplateSid": computed()})).Failures)
	require.Empty(t, l.check(with(verifyInputs(), map[string]property.Value{"defaultTemplateSid": sv(verifyTemplate)})).Failures)
}
