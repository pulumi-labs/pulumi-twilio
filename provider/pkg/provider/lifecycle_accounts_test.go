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

const awsKeys = "AKIAEXAMPLE:wJalrXUtnFEMI/K7MDENG/bPxRfiCYEXAMPLEKEY"

func TestAwsCredentialLC_Lifecycle(t *testing.T) {
	l := newTL(t, "AwsCredential", "aws")
	in := pv(map[string]property.Value{"credentials": sv(awsKeys), "friendlyName": sv("recordings")})
	c := l.create(in)
	require.True(t, strings.HasPrefix(c.ID, "CR"), c.ID)
	require.Equal(t, c.ID, getS(t, c.Properties, "sid"))
	form := l.f.callsMatching("POST", "/accounts/v1/Credentials/AWS")[0].Form
	require.Equal(t, awsKeys, form.Get("Credentials"))
	require.Equal(t, "recordings", form.Get("FriendlyName"))
	require.Empty(t, form["AccountSid"])
	requireNoChanges(t, l.diff(c.ID, c.Properties, in, in))
	rd := l.read(c.ID, c.Properties, in)
	requireNoChanges(t, l.diff(c.ID, rd.Properties, in, rd.Inputs))
	// The keys are never returned: the stored value stays.
	require.Equal(t, awsKeys, getS(t, rd.Inputs, "credentials"))

	// Drift in the name shows.
	l.f.mutate("acr", testAccount, "AWS/"+c.ID, func(o map[string]any) { o["friendly_name"] = "renamed" })
	rd = l.read(c.ID, c.Properties, in)
	require.Equal(t, "renamed", getS(t, rd.Inputs, "friendlyName"))
	require.Equal(t, pgp.Update, l.diff(c.ID, rd.Properties, in, rd.Inputs).DetailedDiff["friendlyName"].Kind)

	// Only the name updates in place; the keys and the subaccount replace.
	next := with(in, map[string]property.Value{"friendlyName": sv("recordings 2")})
	require.Equal(t, pgp.Update, l.diff(c.ID, c.Properties, next, in).DetailedDiff["friendlyName"].Kind)
	l.f.resetCalls()
	u := l.update(c.ID, c.Properties, next, in)
	uf := l.f.callsMatching("POST", "/accounts/v1/Credentials/AWS/"+c.ID)[0].Form
	require.Equal(t, "recordings 2", uf.Get("FriendlyName"))
	require.Empty(t, uf["Credentials"])
	requireNoChanges(t, l.diff(c.ID, u.Properties, next, next))
	require.Equal(t, pgp.UpdateReplace, l.diff(c.ID, u.Properties, with(next, map[string]property.Value{"credentials": sv("AKIAOTHER:secret")}), next).DetailedDiff["credentials"].Kind)
	sub := "AC" + strings.Repeat("b", 32)
	require.Equal(t, pgp.UpdateReplace, l.diff(c.ID, u.Properties, with(next, map[string]property.Value{"accountSid": sv(sub)}), next).DetailedDiff["accountSid"].Kind)

	// The account Twilio reports is adopted, the provider's own included, so stating it is no change.
	own := "AC" + strings.Repeat("a", 32)
	l.f.mutate("acr", testAccount, "AWS/"+c.ID, func(o map[string]any) { o["account_sid"] = own })
	// Import by SID: the keys are unknown, so stating them afterwards is no change.
	imp := l.read(c.ID, empty(), empty())
	require.Empty(t, l.check(imp.Inputs).Failures)
	require.Equal(t, "recordings 2", getS(t, imp.Inputs, "friendlyName"))
	require.False(t, hasProp(imp.Inputs, "credentials"))
	require.Equal(t, own, getS(t, imp.Inputs, "accountSid"))
	requireNoChanges(t, l.diff(c.ID, imp.Properties, with(imp.Inputs, map[string]property.Value{"accountSid": sv(own)}), imp.Inputs))
	requireNoChanges(t, l.diff(c.ID, imp.Properties, with(imp.Inputs, map[string]property.Value{"credentials": sv(awsKeys)}), imp.Inputs))
	l.requireImportFails("CR" + strings.Repeat("0", 32))

	l.f.remove("acr", testAccount, "AWS/"+c.ID)
	require.Empty(t, l.read(c.ID, u.Properties, next).ID)
	l.del(c.ID, u.Properties)
	c = l.create(pv(map[string]property.Value{"credentials": sv(awsKeys), "accountSid": sv(sub)}))
	require.Equal(t, sub, l.f.callsMatching("POST", "/accounts/v1/Credentials/AWS")[1].Form.Get("AccountSid"))
	// An import adopts the subaccount a credential belongs to, and the program stating it is no change.
	imp = l.read(c.ID, empty(), empty())
	require.Equal(t, sub, getS(t, imp.Inputs, "accountSid"))
	require.Empty(t, l.check(imp.Inputs).Failures)
	l.del(c.ID, c.Properties)
	require.Nil(t, l.f.get("acr", testAccount, "AWS/"+c.ID))

	// Preview accepts a missing key (an import has none); creating without one fails with a clear error.
	require.Empty(t, l.check(pv(map[string]property.Value{"friendlyName": sv("x")})).Failures)
	_, err := l.s.Create(pgp.CreateRequest{Urn: l.urn, Properties: l.checked(pv(map[string]property.Value{"friendlyName": sv("x")}))})
	require.ErrorContains(t, err, "credentials is required")
	bad := l.check(pv(map[string]property.Value{"credentials": sv(awsKeys), "friendlyName": sv(strings.Repeat("x", 65)), "accountSid": sv("nope")}))
	require.Len(t, bad.Failures, 2)
}

func TestPublicKeyCredentialLC_Lifecycle(t *testing.T) {
	l := newTL(t, "PublicKeyCredential", "pk")
	key := "-----BEGIN PUBLIC KEY-----MIIBIjANBgkqhkiG9w0BAQEFAAOCAQ8A-----END PUBLIC KEY-----"
	in := pv(map[string]property.Value{"publicKey": sv(key), "friendlyName": sv("call recordings")})
	c := l.create(in)
	require.True(t, strings.HasPrefix(c.ID, "CR"), c.ID)
	form := l.f.callsMatching("POST", "/accounts/v1/Credentials/PublicKeys")[0].Form
	require.Equal(t, key, form.Get("PublicKey"))
	requireNoChanges(t, l.diff(c.ID, c.Properties, in, in))
	rd := l.read(c.ID, c.Properties, in)
	requireNoChanges(t, l.diff(c.ID, rd.Properties, in, rd.Inputs))
	require.Equal(t, key, getS(t, rd.Inputs, "publicKey"))

	next := with(in, map[string]property.Value{"friendlyName": sv("call recordings 2")})
	require.Equal(t, pgp.Update, l.diff(c.ID, c.Properties, next, in).DetailedDiff["friendlyName"].Kind)
	u := l.update(c.ID, c.Properties, next, in)
	require.Equal(t, "call recordings 2", l.f.get("acr", testAccount, "PublicKeys/"+c.ID)["friendly_name"])
	require.Equal(t, pgp.UpdateReplace, l.diff(c.ID, u.Properties, with(next, map[string]property.Value{"publicKey": sv("other")}), next).DetailedDiff["publicKey"].Kind)

	l.f.mutate("acr", testAccount, "PublicKeys/"+c.ID, func(o map[string]any) { o["account_sid"] = "AC" + strings.Repeat("a", 32) })
	imp := l.read(c.ID, empty(), empty())
	require.Empty(t, l.check(imp.Inputs).Failures)
	require.False(t, hasProp(imp.Inputs, "publicKey"))
	requireNoChanges(t, l.diff(c.ID, imp.Properties, with(imp.Inputs, map[string]property.Value{"publicKey": sv(key)}), imp.Inputs))
	l.requireImportFails("CR" + strings.Repeat("0", 32))

	l.f.remove("acr", testAccount, "PublicKeys/"+c.ID)
	require.Empty(t, l.read(c.ID, u.Properties, next).ID)
	l.del(c.ID, u.Properties)
	c = l.create(in)
	l.del(c.ID, c.Properties)
	require.Nil(t, l.f.get("acr", testAccount, "PublicKeys/"+c.ID))
	_, err := l.s.Create(pgp.CreateRequest{Urn: l.urn, Properties: l.checked(pv(map[string]property.Value{"friendlyName": sv("x")}))})
	require.ErrorContains(t, err, "publicKey is required")
}

func TestAccountSafeListNumberLC_Lifecycle(t *testing.T) {
	l := newTL(t, "AccountSafeListNumber", "sl")
	in := pv(map[string]property.Value{"phoneNumber": sv("+14155550100")})
	c := l.create(in)
	require.Equal(t, "+14155550100", c.ID)
	require.True(t, strings.HasPrefix(getS(t, c.Properties, "sid"), "GN"))
	require.Equal(t, "+14155550100", l.f.callsMatching("POST", "/accounts/v1/SafeList/Numbers")[0].Form.Get("PhoneNumber"))
	requireNoChanges(t, l.diff(c.ID, c.Properties, in, in))

	// The fake insists the plus sign is escaped as %2B in the query: a read succeeds only then.
	rd := l.read(c.ID, c.Properties, in)
	require.Equal(t, c.ID, rd.ID)
	requireNoChanges(t, l.diff(c.ID, rd.Properties, in, rd.Inputs))
	imp := l.read(c.ID, empty(), empty())
	require.Equal(t, "+14155550100", getS(t, imp.Inputs, "phoneNumber"))
	require.Empty(t, l.check(imp.Inputs).Failures)
	requireNoChanges(t, l.diff(c.ID, imp.Properties, in, imp.Inputs))
	d := l.diff(c.ID, c.Properties, pv(map[string]property.Value{"phoneNumber": sv("+14155550101")}), in)
	require.Equal(t, pgp.UpdateReplace, d.DetailedDiff["phoneNumber"].Kind)
	l.requireImportFails("+14155550199")

	// A reply that names no number is not an entry.
	l.f.mutate("asl", testAccount, c.ID, func(o map[string]any) { o["phone_number"] = "" })
	require.Empty(t, l.read(c.ID, c.Properties, in).ID)
	l.f.remove("asl", testAccount, c.ID)
	require.Empty(t, l.read(c.ID, c.Properties, in).ID)
	l.del(c.ID, c.Properties)
	c = l.create(in)
	l.del(c.ID, c.Properties)
	require.Nil(t, l.f.get("asl", testAccount, "+14155550100"))
	require.NotEmpty(t, l.f.callsMatching("DELETE", "/SafeList/Numbers"))

	for _, bad := range []string{"", "4155550100", "+0123", "tel:+14155550100"} {
		chk := l.check(pv(map[string]property.Value{"phoneNumber": sv(bad)}))
		require.Len(t, chk.Failures, 1, bad)
	}
	require.Empty(t, l.check(pv(map[string]property.Value{"phoneNumber": computed()})).Failures)
}

func TestLookup_Accounts(t *testing.T) {
	l := newTL(t, "AwsCredential", "aws")
	aws := l.create(pv(map[string]property.Value{"credentials": sv(awsKeys), "friendlyName": sv("recordings")})).ID
	r, err := invoke(t, l, "getAwsCredential", map[string]property.Value{"sid": sv(aws)})
	require.NoError(t, err)
	require.Equal(t, "recordings", getS(t, r, "friendlyName"))
	require.Equal(t, testAccount, getS(t, r, "accountSid"))
	require.False(t, hasProp(r, "credentials"))
	_, err = invoke(t, l, "getAwsCredential", map[string]property.Value{"sid": sv("CRmissing")})
	require.ErrorContains(t, err, "was not found")
	_, err = invoke(t, l, "getAwsCredential", map[string]property.Value{"sid": sv("")})
	require.ErrorContains(t, err, "must be set")

	pk := l.other("PublicKeyCredential", "pk").create(pv(map[string]property.Value{"publicKey": sv("key")})).ID
	r, err = invoke(t, l, "getPublicKeyCredential", map[string]property.Value{"sid": sv(pk)})
	require.NoError(t, err)
	require.Equal(t, pk, getS(t, r, "sid"))
	require.False(t, hasProp(r, "friendlyName"))
	_, err = invoke(t, l, "getPublicKeyCredential", map[string]property.Value{"sid": sv("CRmissing")})
	require.ErrorContains(t, err, "was not found")

	l.other("AccountSafeListNumber", "sl").create(pv(map[string]property.Value{"phoneNumber": sv("+14155550100")}))
	r, err = invoke(t, l, "getAccountSafeListNumber", map[string]property.Value{"phoneNumber": sv("+14155550100")})
	require.NoError(t, err)
	require.Equal(t, "+14155550100", getS(t, r, "phoneNumber"))
	require.True(t, strings.HasPrefix(getS(t, r, "sid"), "GN"))
	_, err = invoke(t, l, "getAccountSafeListNumber", map[string]property.Value{"phoneNumber": sv("+14155550199")})
	require.ErrorContains(t, err, "was not found")

	r, err = invoke(t, l, "getMessagingGeoPermissions", map[string]property.Value{})
	require.NoError(t, err)
	require.Len(t, r.Get("permissions").AsArray().AsSlice(), 3)
	r, err = invoke(t, l, "getMessagingGeoPermissions", map[string]property.Value{"countryCodes": sv("IN,PK")})
	require.NoError(t, err)
	perms := r.Get("permissions").AsArray().AsSlice()
	require.Len(t, perms, 2)
	require.Equal(t, "IN", perms[0].AsMap().Get("countryCode").AsString())
	require.False(t, perms[0].AsMap().Get("enabled").AsBool())
	require.Equal(t, "High Risk Country", perms[1].AsMap().Get("message").AsString())
	require.False(t, hasProp(perms[0].AsMap(), "message"))
}
