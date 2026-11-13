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

func TestNotifyCredentialLC_Lifecycle(t *testing.T) {
	l := newTL(t, "NotifyCredential", "cred")
	in := pv(map[string]property.Value{"credentialType": sv("apn"), "friendlyName": sv("ios"), "certificate": sv("CERT"),
		"privateKey": sv("KEY"), "sandbox": bv(true)})
	c := l.create(in)
	require.True(t, strings.HasPrefix(c.ID, "CR"), c.ID)
	form := l.f.callsMatching("POST", "/Credentials")[0].Form
	require.Equal(t, "apn", form.Get("Type"))
	require.Equal(t, "KEY", form.Get("PrivateKey"))
	require.Equal(t, "true", form.Get("Sandbox"))
	requireNoChanges(t, l.diff(c.ID, c.Properties, in, in))

	// The fake, like Twilio, never returns the certificate or key, and reports the flag as a string.
	rd := l.read(c.ID, c.Properties, in)
	requireNoChanges(t, l.diff(c.ID, rd.Properties, in, rd.Inputs))
	require.Equal(t, "CERT", getS(t, rd.Inputs, "certificate"))
	l.f.mutate("nt", testAccount, "Credentials/"+c.ID, func(o map[string]any) { o["sandbox"] = "false" })
	rd = l.read(c.ID, c.Properties, in)
	require.Equal(t, pgp.Update, l.diff(c.ID, rd.Properties, in, rd.Inputs).DetailedDiff["sandbox"].Kind)

	next := with(in, map[string]property.Value{"friendlyName": sv("ios 2"), "privateKey": sv("KEY2")})
	d := l.diff(c.ID, c.Properties, next, in)
	require.Equal(t, pgp.Update, d.DetailedDiff["friendlyName"].Kind)
	require.Equal(t, pgp.Update, d.DetailedDiff["privateKey"].Kind)
	l.f.resetCalls()
	u := l.update(c.ID, c.Properties, next, in)
	uf := l.f.callsMatching("POST", "/Credentials/")[0].Form
	require.Equal(t, "KEY2", uf.Get("PrivateKey"))
	require.Empty(t, uf["Type"])
	require.Equal(t, "ios 2", l.f.get("nt", testAccount, "Credentials/"+c.ID)["friendly_name"])
	requireNoChanges(t, l.diff(c.ID, u.Properties, next, next))
	require.Equal(t, pgp.UpdateReplace, l.diff(c.ID, u.Properties, with(next, map[string]property.Value{"credentialType": sv("gcm")}), next).DetailedDiff["credentialType"].Kind)

	imp := l.read(c.ID, empty(), empty())
	require.Equal(t, "apn", getS(t, imp.Inputs, "credentialType"))
	require.Equal(t, "ios 2", getS(t, imp.Inputs, "friendlyName"))
	require.False(t, hasProp(imp.Inputs, "privateKey"))

	l.f.remove("nt", testAccount, "Credentials/"+c.ID)
	require.Empty(t, l.read(c.ID, u.Properties, next).ID)
	l.del(c.ID, u.Properties)
	l.requireImportFails("CR" + strings.Repeat("0", 32))

	// An FCM credential takes a secret; a key for another service is refused.
	fcm := l.create(pv(map[string]property.Value{"credentialType": sv("fcm"), "secret": sv("SRV")}))
	require.Equal(t, "SRV", l.f.callsMatching("POST", "/Credentials")[1].Form.Get("Secret"))
	l.del(fcm.ID, fcm.Properties)
	bad := l.check(pv(map[string]property.Value{"credentialType": sv("fcm"), "apiKey": sv("k"), "certificate": sv("c")}))
	require.Len(t, bad.Failures, 2)
	require.Len(t, l.check(pv(map[string]property.Value{"credentialType": sv("sms")})).Failures, 1)
}

func TestNotifyServiceLC_Lifecycle(t *testing.T) {
	l := newTL(t, "NotifyService", "svc")
	cred := l.other("NotifyCredential", "cred").create(pv(map[string]property.Value{"credentialType": sv("gcm"), "apiKey": sv("AIza")})).ID
	in := pv(map[string]property.Value{"friendlyName": sv("push"), "gcmCredentialSid": sv(cred), "logEnabled": bv(false)})
	c := l.create(in)
	require.True(t, strings.HasPrefix(c.ID, "IS"), c.ID)
	require.Equal(t, c.ID, getS(t, c.Properties, "sid"))
	form := l.f.callsMatching("POST", "/notify/v1/Services")[0].Form
	require.Equal(t, cred, form.Get("GcmCredentialSid"))
	require.Equal(t, "false", form.Get("LogEnabled"))
	require.Empty(t, form["ApnCredentialSid"])
	requireNoChanges(t, l.diff(c.ID, c.Properties, in, in))
	rd := l.read(c.ID, c.Properties, in)
	requireNoChanges(t, l.diff(c.ID, rd.Properties, in, rd.Inputs))

	// Drift in a setting the program states shows; a setting it leaves out does not.
	l.f.mutate("nt", testAccount, "Services/"+c.ID, func(o map[string]any) { o["log_enabled"] = true; o["delivery_callback_enabled"] = true })
	rd = l.read(c.ID, c.Properties, in)
	require.True(t, rd.Inputs.Get("logEnabled").AsBool())
	require.Equal(t, pgp.Update, l.diff(c.ID, rd.Properties, in, rd.Inputs).DetailedDiff["logEnabled"].Kind)
	require.False(t, hasProp(rd.Inputs, "deliveryCallbackEnabled"))

	next := with(in, map[string]property.Value{"friendlyName": sv("push-2"), "messagingServiceSid": sv("MG" + strings.Repeat("2", 32)), "deliveryCallbackUrl": sv("https://example.com/dlr")})
	d := l.diff(c.ID, c.Properties, next, in)
	require.Equal(t, pgp.Update, d.DetailedDiff["friendlyName"].Kind)
	require.Equal(t, pgp.Update, d.DetailedDiff["messagingServiceSid"].Kind)
	l.f.resetCalls()
	u := l.update(c.ID, c.Properties, next, in)
	uf := l.f.callsMatching("POST", "/notify/v1/Services/"+c.ID)[0].Form
	require.Equal(t, "push-2", uf.Get("FriendlyName"))
	require.Equal(t, "https://example.com/dlr", l.f.get("nt", testAccount, "Services/"+c.ID)["delivery_callback_url"])
	requireNoChanges(t, l.diff(c.ID, u.Properties, next, next))

	// Import adopts what differs from the defaults.
	imp := l.read(c.ID, empty(), empty())
	require.Equal(t, "push-2", getS(t, imp.Inputs, "friendlyName"))
	require.Equal(t, cred, getS(t, imp.Inputs, "gcmCredentialSid"))
	require.True(t, imp.Inputs.Get("deliveryCallbackEnabled").AsBool())
	require.False(t, imp.Inputs.Get("logEnabled").AsBool())
	require.False(t, hasProp(imp.Inputs, "apnCredentialSid"))
	l.requireImportFails("IS" + strings.Repeat("0", 32))

	l.f.remove("nt", testAccount, "Services/"+c.ID)
	require.Empty(t, l.read(c.ID, c.Properties, in).ID)
	l.del(c.ID, c.Properties)
	c = l.create(in)
	l.del(c.ID, c.Properties)
	require.Nil(t, l.f.get("nt", testAccount, "Services/"+c.ID))

	bad := l.check(pv(map[string]property.Value{"gcmCredentialSid": sv("nope"), "messagingServiceSid": sv("nope"), "friendlyName": sv(strings.Repeat("x", 65))}))
	require.Len(t, bad.Failures, 3)
}

func TestLookup_Notify(t *testing.T) {
	l := newTL(t, "NotifyService", "svc")
	cred := l.other("NotifyCredential", "cred").create(pv(map[string]property.Value{"credentialType": sv("gcm"), "apiKey": sv("AIzaSECRET")}))
	svc := l.create(pv(map[string]property.Value{"friendlyName": sv("push"), "gcmCredentialSid": sv(cred.ID)})).ID
	r, err := invoke(t, l, "getNotifyService", map[string]property.Value{"sid": sv(svc)})
	require.NoError(t, err)
	require.Equal(t, "push", getS(t, r, "friendlyName"))
	require.Equal(t, cred.ID, getS(t, r, "gcmCredentialSid"))
	require.True(t, r.Get("logEnabled").AsBool())
	require.False(t, hasProp(r, "apnCredentialSid"))
	_, err = invoke(t, l, "getNotifyService", map[string]property.Value{"sid": sv("ISmissing")})
	require.ErrorContains(t, err, "was not found")
	_, err = invoke(t, l, "getNotifyService", map[string]property.Value{"sid": sv("")})
	require.ErrorContains(t, err, "must be set")

	cr, err := invoke(t, l, "getNotifyCredential", map[string]property.Value{"sid": sv(cred.ID)})
	require.NoError(t, err)
	require.Equal(t, "gcm", getS(t, cr, "credentialType"))
	require.False(t, hasProp(cr, "apiKey"))
	_, err = invoke(t, l, "getNotifyCredential", map[string]property.Value{"sid": sv("CRmissing")})
	require.ErrorContains(t, err, "was not found")
	_, err = invoke(t, l, "getNotifyCredential", map[string]property.Value{"sid": sv("")})
	require.ErrorContains(t, err, "must be set")
}
