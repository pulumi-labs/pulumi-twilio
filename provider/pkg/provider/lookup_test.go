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
	"fmt"
	"testing"

	pgp "github.com/pulumi/pulumi-go-provider"
	"github.com/pulumi/pulumi/sdk/v3/go/common/tokens"
	"github.com/pulumi/pulumi/sdk/v3/go/property"
	"github.com/stretchr/testify/require"
)

func invoke(t *testing.T, l *tl, token string, args map[string]property.Value) (property.Map, error) {
	t.Helper()
	r, err := l.s.Invoke(pgp.InvokeRequest{Token: tokens.Type("twilio:index:" + token), Args: pv(args)})
	if err == nil && len(r.Failures) > 0 {
		err = fmt.Errorf("invalid arguments: %+v", r.Failures)
	}
	return r.Return, err
}

func TestLookup_Subaccount(t *testing.T) {
	l := newTL(t, "Subaccount", "sub")
	l.f.addSubaccount(fakeSub)
	r, err := invoke(t, l, "getSubaccount", map[string]property.Value{"subaccountSid": sv(fakeSub)})
	require.NoError(t, err)
	require.Equal(t, fakeSub, getS(t, r, "sid"))
	require.Equal(t, "sub", getS(t, r, "friendlyName"))
	require.Equal(t, testAccount, getS(t, r, "ownerAccountSid"))
	require.False(t, hasProp(r, "authToken"), "the auth token must not be returned")

	_, err = invoke(t, l, "getSubaccount", map[string]property.Value{"subaccountSid": sv("ACmissing")})
	require.ErrorContains(t, err, "was not found")
	_, err = invoke(t, l, "getSubaccount", map[string]property.Value{"subaccountSid": sv("")})
	require.ErrorContains(t, err, "must be set")
}

func TestLookup_ApiKey(t *testing.T) {
	l := newTL(t, "ApiKey", "k")
	l.f.addKey(testAccount, "SKstd", "plain", nil)
	l.f.addKey(testAccount, "SKres", "limited", map[string]any{"allow": []any{"/Accounts/*"}})
	r, err := invoke(t, l, "getApiKey", map[string]property.Value{"keySid": sv("SKstd")})
	require.NoError(t, err)
	require.Equal(t, "standard", getS(t, r, "keyType"))
	require.Equal(t, "", getS(t, r, "policy"))
	require.False(t, hasProp(r, "secret"), "the secret must not be returned")
	r, err = invoke(t, l, "getApiKey", map[string]property.Value{"keySid": sv("SKres")})
	require.NoError(t, err)
	require.Equal(t, "restricted", getS(t, r, "keyType"))
	require.Contains(t, getS(t, r, "policy"), "/Accounts/*")
	require.Equal(t, "limited", getS(t, r, "friendlyName"))

	_, err = invoke(t, l, "getApiKey", map[string]property.Value{"keySid": sv("SKnope")})
	require.ErrorContains(t, err, "was not found")
}

func TestLookup_Application(t *testing.T) {
	l := newTL(t, "Application", "app")
	c := l.create(pv(map[string]property.Value{"friendlyName": sv("voice-app"), "voiceUrl": sv("https://example.com/v")}))
	r, err := invoke(t, l, "getApplication", map[string]property.Value{"applicationSid": sv(c.ID)})
	require.NoError(t, err)
	require.Equal(t, c.ID, getS(t, r, "sid"))
	require.Equal(t, "voice-app", getS(t, r, "friendlyName"))
	require.Equal(t, "https://example.com/v", getS(t, r, "voiceUrl"))
	require.Equal(t, testAccount, getS(t, r, "ownerAccountSid"))

	l.del(c.ID, c.Properties)
	_, err = invoke(t, l, "getApplication", map[string]property.Value{"applicationSid": sv(c.ID)})
	require.ErrorContains(t, err, "was not found")
}

func TestLookup_MessagingServiceAndSender(t *testing.T) {
	l := newTL(t, "MessagingService", "svc")
	c := l.create(msgInputs())
	r, err := invoke(t, l, "getMessagingService", map[string]property.Value{"serviceSid": sv(c.ID)})
	require.NoError(t, err)
	require.Equal(t, c.ID, getS(t, r, "sid"))
	require.Equal(t, "notify", getS(t, r, "friendlyName"))
	require.Equal(t, "https://example.com/in", getS(t, r, "inboundRequestUrl"))
	require.Equal(t, 600.0, r.Get("validityPeriod").AsNumber())

	l.f.addNumber(testAccount, "PNlookup", "+14155550111")
	s := l.other("MessagingServicePhoneNumber", "sender")
	sc := s.create(pv(map[string]property.Value{"serviceSid": sv(c.ID), "phoneNumberSid": sv("PNlookup")}))
	sr, err := invoke(t, l, "getMessagingServicePhoneNumber", map[string]property.Value{"serviceSid": sv(c.ID), "phoneNumberSid": sv("PNlookup")})
	require.NoError(t, err)
	require.Equal(t, "+14155550111", getS(t, sr, "phoneNumber"))
	require.Equal(t, "PNlookup", getS(t, sr, "phoneNumberSid"))

	s.del(sc.ID, sc.Properties)
	_, err = invoke(t, l, "getMessagingServicePhoneNumber", map[string]property.Value{"serviceSid": sv(c.ID), "phoneNumberSid": sv("PNlookup")})
	require.ErrorContains(t, err, "was not found")
	_, err = invoke(t, l, "getMessagingServicePhoneNumber", map[string]property.Value{"serviceSid": sv(c.ID)})
	require.Error(t, err)

	l.del(c.ID, c.Properties)
	_, err = invoke(t, l, "getMessagingService", map[string]property.Value{"serviceSid": sv(c.ID)})
	require.ErrorContains(t, err, "was not found")
}

func TestLookup_VerifyService(t *testing.T) {
	l := newTL(t, "VerifyService", "v")
	c := l.create(pv(map[string]property.Value{"friendlyName": sv("login"), "codeLength": nv(7)}))
	r, err := invoke(t, l, "getVerifyService", map[string]property.Value{"serviceSid": sv(c.ID)})
	require.NoError(t, err)
	require.Equal(t, c.ID, getS(t, r, "sid"))
	require.Equal(t, "login", getS(t, r, "friendlyName"))
	require.Equal(t, 7.0, r.Get("codeLength").AsNumber())

	l.del(c.ID, c.Properties)
	_, err = invoke(t, l, "getVerifyService", map[string]property.Value{"serviceSid": sv(c.ID)})
	require.ErrorContains(t, err, "was not found")
}

func TestLookup_Balance(t *testing.T) {
	l := newTL(t, "Subaccount", "sub")
	r, err := invoke(t, l, "getBalance", map[string]property.Value{})
	require.NoError(t, err)
	require.Equal(t, "12.3400", getS(t, r, "balance"))
	require.Equal(t, "USD", getS(t, r, "currency"))
	require.Equal(t, testAccount, getS(t, r, "accountSid"))
	_, err = invoke(t, l, "getBalance", map[string]property.Value{"accountSid": sv("ACmissing")})
	require.ErrorContains(t, err, "was not found")
}
