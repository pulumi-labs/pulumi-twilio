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

	"github.com/blang/semver"
	pgp "github.com/pulumi/pulumi-go-provider"
	"github.com/pulumi/pulumi-go-provider/integration"
	"github.com/pulumi/pulumi/sdk/v3/go/property"
	"github.com/stretchr/testify/require"
)

// ----------------------------------------------------- IncomingPhoneNumber

const (
	numSid = "PN900"
	numE   = "+14155550100"
)

func newNumberLC(t *testing.T) *tl {
	l := newTL(t, "IncomingPhoneNumber", "num")
	l.f.addNumber(testAccount, numSid, numE)
	return l
}

func numberInputs() property.Map {
	return pv(map[string]property.Value{
		"phoneNumber": sv(numE), "friendlyName": sv("support"), "voiceUrl": sv("https://example.com/voice"),
		"smsUrl": sv("https://example.com/sms"), "smsMethod": sv("GET"), "voiceCallerIdLookup": bv(true),
	})
}

func TestNumberLC_Lifecycle(t *testing.T) {
	l := newNumberLC(t)
	in := numberInputs()
	c := l.create(in)
	require.Equal(t, numSid, c.ID)
	require.Equal(t, numE, getS(t, c.Properties, "number"))
	require.Equal(t, testAccount, getS(t, c.Properties, "ownerAccountSid"))
	stored := l.f.get("number", testAccount, numSid)
	require.Equal(t, "https://example.com/voice", stored["voice_url"])
	require.Equal(t, "GET", stored["sms_method"])
	require.Equal(t, true, stored["voice_caller_id_lookup"])
	require.Equal(t, "support", stored["friendly_name"])

	// The adopted number's demo webhook is visible as an output, not an input.
	require.Equal(t, "https://example.com/voice", getS(t, effective(t, c.Properties), "voiceUrl"))
	requireNoChanges(t, l.diff(c.ID, c.Properties, in, in))
	rd := l.read(c.ID, c.Properties, in)
	requireNoChanges(t, l.diff(c.ID, rd.Properties, in, rd.Inputs))

	next := pv(map[string]property.Value{
		"phoneNumber": sv(numE), "friendlyName": sv("billing"), "voiceUrl": sv("https://example.com/v2"),
		"voiceMethod": sv("GET"), "voiceFallbackUrl": sv("https://example.com/vf"), "voiceFallbackMethod": sv("GET"),
		"statusCallback": sv("https://example.com/sc"), "statusCallbackMethod": sv("GET"),
		"smsUrl": sv("https://example.com/s2"), "smsMethod": sv("POST"), "smsFallbackUrl": sv("https://example.com/sf"),
		"smsFallbackMethod": sv("GET"), "voiceApplicationSid": sv("AP1"), "smsApplicationSid": sv("AP2"),
		"voiceCallerIdLookup": bv(false),
	})
	d := l.diff(c.ID, rd.Properties, next, in)
	require.True(t, d.HasChanges)
	require.False(t, d.DeleteBeforeReplace)
	u := l.update(c.ID, rd.Properties, next, in)
	stored = l.f.get("number", testAccount, numSid)
	for k, v := range map[string]any{
		"friendly_name": "billing", "voice_url": "https://example.com/v2", "voice_method": "GET",
		"voice_fallback_url": "https://example.com/vf", "voice_fallback_method": "GET",
		"status_callback": "https://example.com/sc", "status_callback_method": "GET",
		"sms_url": "https://example.com/s2", "sms_method": "POST", "sms_fallback_url": "https://example.com/sf",
		"sms_fallback_method": "GET", "voice_application_sid": "AP1", "sms_application_sid": "AP2",
		"voice_caller_id_lookup": false,
	} {
		require.Equal(t, v, stored[k], k)
	}
	requireNoChanges(t, l.diff(c.ID, u.Properties, next, next))

	// B2/B7: removed inputs are reset (methods to POST, URLs cleared, the bool to its default). The
	// friendly name has no default, so it is left alone instead of being sent as "".
	min := pv(map[string]property.Value{"phoneNumber": sv(numE)})
	l.f.resetCalls()
	u2 := l.update(c.ID, u.Properties, min, next)
	stored = l.f.get("number", testAccount, numSid)
	for _, k := range []string{"voice_url", "voice_fallback_url", "status_callback", "sms_url", "sms_fallback_url", "voice_application_sid", "sms_application_sid"} {
		require.Nil(t, stored[k], k)
	}
	for _, k := range []string{"voice_method", "voice_fallback_method", "status_callback_method", "sms_method", "sms_fallback_method"} {
		require.Equal(t, "POST", stored[k], k)
	}
	require.Equal(t, "billing", stored["friendly_name"])
	require.NotContains(t, l.f.callsMatching("POST", "IncomingPhoneNumbers/")[0].Form, "FriendlyName")
	requireNoChanges(t, l.diff(c.ID, u2.Properties, min, min))

	// The identity is replace-on-change.
	for k, v := range map[string]property.Value{"phoneNumber": sv("+14155550199")} {
		d = l.diff(c.ID, u2.Properties, with(min, map[string]property.Value{k: v}), min)
		require.Equal(t, pgp.UpdateReplace, d.DetailedDiff[k].Kind, k)
	}
	d = l.diff(c.ID, u2.Properties, with(min, map[string]property.Value{"phoneNumberSid": sv("PN1")}), min)
	require.Equal(t, pgp.AddReplace, d.DetailedDiff["phoneNumberSid"].Kind)

	// Delete never releases the number and makes no API call.
	l.f.resetCalls()
	l.del(c.ID, u2.Properties)
	require.Zero(t, l.f.callCount())
	require.NotNil(t, l.f.get("number", testAccount, numSid))
}

func TestNumberLC_RefreshDriftAndImport(t *testing.T) {
	l := newNumberLC(t)
	in := numberInputs()
	c := l.create(in)

	l.f.mutate("number", testAccount, numSid, func(o map[string]any) {
		o["voice_url"] = "https://drift.example.com"
		o["status_callback"] = "https://console.example.com/status" // not in the program
	})
	rd := l.read(c.ID, c.Properties, in)
	require.Equal(t, "https://drift.example.com", getS(t, rd.Inputs, "voiceUrl"))
	require.Equal(t, "https://console.example.com/status", getS(t, effective(t, rd.Properties), "statusCallback"))
	require.False(t, hasProp(rd.Inputs, "statusCallback"))
	require.Equal(t, pgp.Update, l.diff(c.ID, rd.Properties, in, rd.Inputs).DetailedDiff["voiceUrl"].Kind)

	imp := l.read(numSid, empty(), empty())
	require.Equal(t, numSid, imp.ID)
	require.Equal(t, numSid, getS(t, imp.Inputs, "phoneNumberSid"))
	require.Equal(t, "https://drift.example.com", getS(t, imp.Inputs, "voiceUrl"))
	require.Equal(t, "GET", getS(t, imp.Inputs, "smsMethod"))
	require.False(t, hasProp(imp.Inputs, "voiceMethod"), "default methods are not adopted")
	require.False(t, hasProp(imp.Inputs, "accountSid"))
	requireNoChanges(t, l.diff(imp.ID, imp.Properties, l.checked(imp.Inputs), imp.Inputs))
}

func TestNumberLC_DeletedOutsidePulumi(t *testing.T) {
	l := newNumberLC(t)
	in := numberInputs()
	c := l.create(in)
	l.f.remove("number", testAccount, numSid)
	require.Empty(t, l.read(c.ID, c.Properties, in).ID)
	l.del(c.ID, c.Properties)
}

// B5: a number moves between accounts in place with the AccountSid update parameter,
// posted to the account that currently owns it.
func TestNumberLC_AccountTransferIsInPlace(t *testing.T) {
	l := newNumberLC(t)
	l.f.addSubaccount(fakeSub)
	in := pv(map[string]property.Value{"phoneNumberSid": sv(numSid)})
	c := l.create(in)

	toSub := with(in, map[string]property.Value{"accountSid": sv(fakeSub)})
	d := l.diff(c.ID, c.Properties, toSub, in)
	require.Equal(t, pgp.Add, d.DetailedDiff["accountSid"].Kind, "an update, not a replace")
	require.False(t, d.DeleteBeforeReplace)
	l.f.resetCalls()
	u := l.update(c.ID, c.Properties, toSub, in)
	require.Nil(t, l.f.get("number", testAccount, numSid))
	require.NotNil(t, l.f.get("number", fakeSub, numSid))
	posts := l.f.callsMatching("POST", "/Accounts/"+testAccount+"/IncomingPhoneNumbers/"+numSid)
	require.Len(t, posts, 1)
	require.Equal(t, fakeSub, posts[0].Form.Get("AccountSid"))
	require.Equal(t, fakeSub, getS(t, u.Properties, "ownerAccountSid"))
	requireNoChanges(t, l.diff(c.ID, u.Properties, toSub, toSub))

	// The unchanged ID still finds it, and deletion does not need the old account.
	rd := l.read(c.ID, u.Properties, toSub)
	require.Equal(t, c.ID, rd.ID)
	requireNoChanges(t, l.diff(c.ID, rd.Properties, toSub, rd.Inputs))

	// And back to the provider account when accountSid is removed.
	l.f.resetCalls()
	back := l.update(c.ID, u.Properties, in, toSub)
	require.NotNil(t, l.f.get("number", testAccount, numSid))
	require.Equal(t, testAccount, getS(t, back.Properties, "ownerAccountSid"))
	require.Len(t, l.f.callsMatching("POST", "/Accounts/"+fakeSub+"/IncomingPhoneNumbers/"+numSid), 1)
}

// B3: a number owned by a subaccount is importable by <accountSid>/<sid>.
func TestNumberLC_SubaccountIDAndImport(t *testing.T) {
	l := newTL(t, "IncomingPhoneNumber", "num")
	l.f.addSubaccount(fakeSub)
	l.f.addNumber(fakeSub, "PN77", "+14155550177")

	// Creating one in the subaccount yields a scoped ID.
	c := l.create(pv(map[string]property.Value{"phoneNumber": sv("+14155550177"), "accountSid": sv(fakeSub)}))
	require.Equal(t, fakeSub+"/PN77", c.ID)
	require.Equal(t, "PN77", getS(t, c.Properties, "sid"))

	imp := l.read(fakeSub+"/PN77", empty(), empty())
	require.Equal(t, fakeSub+"/PN77", imp.ID)
	require.Equal(t, "PN77", getS(t, imp.Inputs, "phoneNumberSid"))
	require.Equal(t, fakeSub, getS(t, imp.Inputs, "accountSid"))
	require.Equal(t, fakeSub, getS(t, imp.Properties, "ownerAccountSid"))
	requireNoChanges(t, l.diff(imp.ID, imp.Properties, l.checked(imp.Inputs), imp.Inputs))
	require.Empty(t, l.read("PN77", empty(), empty()).ID, "a bare SID is looked up in the provider's account")
}

func TestNumberLC_ConfigureFailureKeepsPartialState(t *testing.T) {
	l := newNumberLC(t)
	l.f.failNext("POST", "IncomingPhoneNumbers/"+numSid, 500)
	resp, err := l.s.Create(pgp.CreateRequest{Urn: l.urn, Properties: l.checked(numberInputs())})
	require.Error(t, err)
	require.NotNil(t, resp.PartialState)
	require.Equal(t, numSid, resp.ID)
	require.Contains(t, resp.PartialState.Reasons[0], "configure phone number")
	require.Equal(t, numSid, getS(t, resp.Properties, "sid"))
}

func TestNumberLC_UnownedAndPreviewAndUnknowns(t *testing.T) {
	l := newNumberLC(t)
	_, err := l.s.Create(pgp.CreateRequest{Urn: l.urn, Properties: l.checked(pv(map[string]property.Value{"phoneNumber": sv("+19995550000")}))})
	require.ErrorContains(t, err, "does not own")
	_, err = l.s.Create(pgp.CreateRequest{Urn: l.urn, Properties: numberInputs(), DryRun: true})
	require.NoError(t, err)

	// B1: a computed identity is only known at apply time; both set or neither set still fail.
	require.Empty(t, l.check(pv(map[string]property.Value{"phoneNumber": computed()})).Failures)
	require.Empty(t, l.check(pv(map[string]property.Value{"phoneNumberSid": computed()})).Failures)
	require.NotEmpty(t, l.check(pv(map[string]property.Value{"phoneNumber": sv(numE), "phoneNumberSid": computed()})).Failures)
	require.NotEmpty(t, l.check(pv(map[string]property.Value{})).Failures)
	require.NotEmpty(t, l.check(pv(map[string]property.Value{"phoneNumber": sv(numE), "voiceMethod": sv("PUT")})).Failures)
	require.Empty(t, l.check(pv(map[string]property.Value{"phoneNumber": sv(numE), "voiceMethod": computed()})).Failures)
}

// --------------------------------------------------------------------- ApiKey

func TestApiKeyLC_StandardLifecycle(t *testing.T) {
	l := newTL(t, "ApiKey", "k")
	in := l.checked(pv(map[string]property.Value{"friendlyName": sv("ci")}))
	require.Equal(t, "standard", getS(t, in, "keyType"))
	c := l.create(in)
	require.Contains(t, c.ID, "SK")
	// B8: a standard key is created by omitting KeyType.
	require.NotContains(t, l.f.callsMatching("POST", "/iam/v1/Keys")[0].Form, "KeyType")
	sec := c.Properties.Get("secret")
	require.Equal(t, "secret-"+c.ID, sec.WithSecret(false).AsString())

	requireNoChanges(t, l.diff(c.ID, c.Properties, in, in))
	rd := l.read(c.ID, c.Properties, in)
	require.Equal(t, "secret-"+c.ID, rd.Properties.Get("secret").WithSecret(false).AsString(), "the secret survives a refresh")
	requireNoChanges(t, l.diff(c.ID, rd.Properties, in, rd.Inputs))

	next := with(in, map[string]property.Value{"friendlyName": sv("ci-2")})
	d := l.diff(c.ID, rd.Properties, next, in)
	require.Equal(t, pgp.Update, d.DetailedDiff["friendlyName"].Kind)
	u := l.update(c.ID, rd.Properties, next, in)
	require.Equal(t, "ci-2", l.f.get("key", testAccount, c.ID)["friendly_name"])

	for k, v := range map[string]property.Value{"keyType": sv("restricted"), "accountSid": sv(fakeSub)} {
		d = l.diff(c.ID, u.Properties, with(next, map[string]property.Value{k: v}), next)
		require.Contains(t, []pgp.DiffKind{pgp.UpdateReplace, pgp.AddReplace}, d.DetailedDiff[k].Kind, k)
	}

	// Drift and deletion outside Pulumi.
	l.f.mutate("key", testAccount, c.ID, func(o map[string]any) { o["friendly_name"] = "console" })
	require.Equal(t, "console", getS(t, l.read(c.ID, u.Properties, next).Inputs, "friendlyName"))
	l.f.remove("key", testAccount, c.ID)
	require.Empty(t, l.read(c.ID, u.Properties, next).ID)
	l.del(c.ID, u.Properties)
}

const (
	pol1 = `{"allow":["/twilio/iam/api-keys/create"]}`
	pol2 = `{"allow":["/twilio/iam/api-keys/create","/twilio/iam/api-keys/delete"]}`
)

func TestApiKeyLC_RestrictedPolicyLifecycleAndDrift(t *testing.T) {
	l := newTL(t, "ApiKey", "k")
	in := pv(map[string]property.Value{"friendlyName": sv("svc"), "keyType": sv("restricted"), "policy": sv(pol1)})
	c := l.create(in)
	require.Equal(t, "restricted", l.f.callsMatching("POST", "/iam/v1/Keys")[0].Form.Get("KeyType"))
	require.Equal(t, map[string]any{"allow": []any{"/twilio/iam/api-keys/create"}}, l.f.get("key", testAccount, c.ID)["policy"])
	requireNoChanges(t, l.diff(c.ID, c.Properties, in, in))

	// The same policy rendered differently by the API is not drift.
	rd := l.read(c.ID, c.Properties, in)
	require.Equal(t, pol1, getS(t, rd.Inputs, "policy"))
	requireNoChanges(t, l.diff(c.ID, rd.Properties, in, rd.Inputs))
	spaced := with(in, map[string]property.Value{"policy": sv(`{ "allow": [ "/twilio/iam/api-keys/create" ] }`)})
	rd = l.read(c.ID, c.Properties, spaced)
	require.Equal(t, `{ "allow": [ "/twilio/iam/api-keys/create" ] }`, getS(t, rd.Inputs, "policy"))

	// B6: a policy widened in the console is detected and reverted by the next update.
	l.f.mutate("key", testAccount, c.ID, func(o map[string]any) {
		o["policy"] = map[string]any{"allow": []any{"/twilio/iam/api-keys/create", "/twilio/iam/api-keys/delete"}}
	})
	rd = l.read(c.ID, c.Properties, in)
	require.JSONEq(t, pol2, getS(t, rd.Inputs, "policy"))
	require.JSONEq(t, pol2, getS(t, rd.Properties, "policy"))
	d := l.diff(c.ID, rd.Properties, in, rd.Inputs)
	require.Equal(t, pgp.Update, d.DetailedDiff["policy"].Kind)
	u := l.update(c.ID, rd.Properties, in, rd.Inputs)
	require.Equal(t, map[string]any{"allow": []any{"/twilio/iam/api-keys/create"}}, l.f.get("key", testAccount, c.ID)["policy"])

	// An intended policy change reaches the API.
	wide := with(in, map[string]property.Value{"policy": sv(pol2)})
	u = l.update(c.ID, u.Properties, wide, in)
	require.Equal(t, map[string]any{"allow": []any{"/twilio/iam/api-keys/create", "/twilio/iam/api-keys/delete"}}, l.f.get("key", testAccount, c.ID)["policy"])
	requireNoChanges(t, l.diff(c.ID, u.Properties, wide, wide))

	// Import infers the key type and policy.
	imp := l.read(c.ID, empty(), empty())
	require.Equal(t, "restricted", getS(t, imp.Inputs, "keyType"))
	require.JSONEq(t, pol2, getS(t, imp.Inputs, "policy"))
	requireNoChanges(t, l.diff(imp.ID, imp.Properties, l.checked(imp.Inputs), imp.Inputs))

	l.del(c.ID, u.Properties)
	require.Nil(t, l.f.get("key", testAccount, c.ID))
	l.del(c.ID, u.Properties)
}

func TestApiKeyLC_ImportStandardKey(t *testing.T) {
	l := newTL(t, "ApiKey", "k")
	l.f.addKey(testAccount, "SK1", "external", nil)
	imp := l.read("SK1", empty(), empty())
	require.Equal(t, "SK1", imp.ID)
	require.Equal(t, "external", getS(t, imp.Inputs, "friendlyName"))
	require.Equal(t, "standard", getS(t, imp.Inputs, "keyType"))
	require.False(t, hasProp(imp.Inputs, "policy"))
	require.False(t, hasProp(imp.Inputs, "accountSid"))
	require.Equal(t, "SK1", getS(t, imp.Properties, "sid"))
	requireNoChanges(t, l.diff(imp.ID, imp.Properties, l.checked(imp.Inputs), imp.Inputs))
}

// B3/B4: a key owned by a subaccount. The IAM route only sees the authenticated account's keys,
// so refresh, update and delete must use the account-explicit route; otherwise refresh would
// drop the key from state and delete would leave a live credential behind.
func TestApiKeyLC_SubaccountKey(t *testing.T) {
	l := newTL(t, "ApiKey", "k")
	l.f.addSubaccount(fakeSub)
	in := l.checked(pv(map[string]property.Value{"friendlyName": sv("sub-key"), "accountSid": sv(fakeSub)}))
	c := l.create(in)
	_, sid := splitScoped(c.ID)
	require.Equal(t, fakeSub+"/"+sid, c.ID)
	require.Equal(t, sid, getS(t, c.Properties, "sid"))
	require.NotNil(t, l.f.get("key", fakeSub, sid))
	require.Equal(t, fakeSub, l.f.callsMatching("POST", "/iam/v1/Keys")[0].Form.Get("AccountSid"))

	rd := l.read(c.ID, c.Properties, in)
	require.Equal(t, c.ID, rd.ID, "refresh must find the subaccount's key, not drop it")
	requireNoChanges(t, l.diff(c.ID, rd.Properties, in, rd.Inputs))

	next := with(in, map[string]property.Value{"friendlyName": sv("renamed")})
	u := l.update(c.ID, rd.Properties, next, in)
	require.Equal(t, "renamed", l.f.get("key", fakeSub, sid)["friendly_name"])

	// A policy change on a subaccount key cannot be done through the classic route.
	l.f.mutate("key", fakeSub, sid, func(o map[string]any) { o["policy"] = nil })
	_, err := l.s.Update(pgp.UpdateRequest{ID: c.ID, Urn: l.urn, State: u.Properties,
		Inputs: with(next, map[string]property.Value{"policy": sv(pol1)}), OldInputs: next})
	require.Error(t, err)

	imp := l.read(c.ID, empty(), empty())
	require.Equal(t, c.ID, imp.ID)
	require.Equal(t, fakeSub, getS(t, imp.Inputs, "accountSid"), "a program with accountSid plans nothing after import")
	requireNoChanges(t, l.diff(imp.ID, imp.Properties, l.checked(imp.Inputs), imp.Inputs))
	requireNoChanges(t, l.diff(imp.ID, imp.Properties, next, imp.Inputs))
	require.Empty(t, l.read(sid, empty(), empty()).ID, "a bare SID is looked up in the provider's account")

	l.del(c.ID, u.Properties)
	require.Nil(t, l.f.get("key", fakeSub, sid), "the credential must really be deleted")
	l.del(c.ID, u.Properties)

	// Deleted outside Pulumi.
	c2 := l.create(in)
	_, sid2 := splitScoped(c2.ID)
	l.f.remove("key", fakeSub, sid2)
	require.Empty(t, l.read(c2.ID, c2.Properties, in).ID)
	l.del(c2.ID, c2.Properties)
}

func TestApiKeyLC_CheckUnknownsAndRules(t *testing.T) {
	l := newTL(t, "ApiKey", "k")
	// B1: a policy that references a service created in the same stack is unknown at preview.
	chk := l.check(pv(map[string]property.Value{"friendlyName": sv("svc"), "keyType": sv("restricted"), "policy": computed()}))
	require.Empty(t, chk.Failures)
	require.True(t, chk.Inputs.Get("policy").IsComputed())
	require.Empty(t, l.check(pv(map[string]property.Value{"friendlyName": computed()})).Failures)
	require.Empty(t, l.check(pv(map[string]property.Value{"friendlyName": sv("x"), "keyType": computed(), "policy": sv(pol1)})).Failures)
	require.NotEmpty(t, l.check(pv(map[string]property.Value{"friendlyName": sv("x"), "keyType": computed(), "policy": sv("{")})).Failures)

	// Known values are still validated.
	require.NotEmpty(t, l.check(pv(map[string]property.Value{"friendlyName": sv("x"), "policy": computed()})).Failures, "a standard key has no policy")
	require.NotEmpty(t, l.check(pv(map[string]property.Value{"friendlyName": sv("x"), "keyType": sv("restricted")})).Failures)
	require.NotEmpty(t, l.check(pv(map[string]property.Value{"friendlyName": sv("x"), "keyType": sv("restricted"), "policy": sv("{")})).Failures)
	require.NotEmpty(t, l.check(pv(map[string]property.Value{"friendlyName": sv("x"), "keyType": sv("weird")})).Failures)

	_, err := l.s.Create(pgp.CreateRequest{Urn: l.urn, Properties: l.checked(pv(map[string]property.Value{"friendlyName": sv("x")})), DryRun: true})
	require.NoError(t, err)
	require.Zero(t, l.f.callCount())
}

// Twilio may echo fields the program never wrote; only the supplied keys are compared.
func TestApiKeyLC_PolicyEchoWithExtraFieldIsNotDrift(t *testing.T) {
	l := newTL(t, "ApiKey", "k")
	in := pv(map[string]property.Value{"friendlyName": sv("svc"), "keyType": sv("restricted"), "policy": sv(pol1)})
	c := l.create(in)
	l.f.mutate("key", testAccount, c.ID, func(o map[string]any) {
		o["policy"] = map[string]any{"allow": []any{"/twilio/iam/api-keys/create"}, "version": "2"}
	})
	rd := l.read(c.ID, c.Properties, in)
	require.Equal(t, pol1, getS(t, rd.Inputs, "policy"), "an extra echoed field is not drift")
	requireNoChanges(t, l.diff(c.ID, rd.Properties, in, rd.Inputs))

	// A supplied key that changed is still drift.
	l.f.mutate("key", testAccount, c.ID, func(o map[string]any) {
		o["policy"] = map[string]any{"allow": []any{"/twilio/iam/api-keys/create", "/twilio/iam/api-keys/delete"}, "version": "2"}
	})
	rd = l.read(c.ID, c.Properties, in)
	require.JSONEq(t, `{"allow":["/twilio/iam/api-keys/create","/twilio/iam/api-keys/delete"],"version":"2"}`, getS(t, rd.Inputs, "policy"))
	require.Equal(t, pgp.Update, l.diff(c.ID, rd.Properties, in, rd.Inputs).DetailedDiff["policy"].Kind)
}

// The subaccount route cannot change a policy, so Update must only refuse a real change: the same
// policy written with another key order is not one.
func TestApiKeyLC_SubaccountPolicyKeyOrderIsNotAChange(t *testing.T) {
	l := newTL(t, "ApiKey", "k")
	l.f.addSubaccount(fakeSub)
	l.f.addKey(fakeSub, "SK777", "sub-restricted", map[string]any{"allow": []any{"/twilio/iam/api-keys/create"}, "deny": []any{}})
	id := fakeSub + "/SK777"
	imp := l.read(id, empty(), empty())
	require.Equal(t, id, imp.ID)
	reordered := pv(map[string]property.Value{"friendlyName": sv("renamed"), "accountSid": sv(fakeSub), "keyType": sv("restricted"),
		"policy": sv(`{ "deny": [], "allow": ["/twilio/iam/api-keys/create"] }`)})
	l.update(id, imp.Properties, reordered, imp.Inputs)
	require.Equal(t, "renamed", l.f.get("key", fakeSub, "SK777")["friendly_name"])

	changed := with(reordered, map[string]property.Value{"policy": sv(`{"allow":["/twilio/iam/api-keys/delete"],"deny":[]}`)})
	_, err := l.s.Update(pgp.UpdateRequest{ID: id, Urn: l.urn, State: imp.Properties, Inputs: l.checked(changed), OldInputs: imp.Inputs})
	require.ErrorContains(t, err, "policy of a subaccount key")
}

// A preview must not need credentials.
func TestNumberLC_UpdatePreviewNeedsNoCredentials(t *testing.T) {
	for _, k := range []string{"TWILIO_ACCOUNT_SID", "TWILIO_AUTH_TOKEN", "TWILIO_API_KEY", "TWILIO_API_SECRET", "TWILIO_BASE_URL"} {
		t.Setenv(k, "")
	}
	l := newNumberLC(t)
	c := l.create(numberInputs())
	// A server that was never configured has no credentials at all.
	prov, err := New()
	require.NoError(t, err)
	bare, err := integration.NewServer(t.Context(), "twilio", semver.MustParse("0.0.1"), integration.WithProvider(prov))
	require.NoError(t, err)
	next := with(numberInputs(), map[string]property.Value{"friendlyName": sv("support-2"), "accountSid": sv(fakeSub)})
	before := l.f.callCount()
	u, err := bare.Update(pgp.UpdateRequest{ID: c.ID, Urn: l.urn, State: c.Properties, Inputs: l.checked(next),
		OldInputs: numberInputs(), DryRun: true})
	require.NoError(t, err)
	require.Equal(t, "support-2", getS(t, u.Properties, "friendlyName"))
	require.Equal(t, before, l.f.callCount(), "a preview sends no request")
}
