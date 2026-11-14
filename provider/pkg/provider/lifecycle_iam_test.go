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

func oauthIn() property.Map {
	return pv(map[string]property.Value{
		"friendlyName": sv("ci app"), "description": sv("used by CI"),
		"policy":         objv(map[string]property.Value{"allow": arrv(sv("/twilio/iam/api-keys/create")), "deny": arrv(sv("/twilio/iam/roles/delete"))}),
		"accessTokenTtl": nv(3600), "tokenEndpointAuthMethod": sv("client_secret_basic"),
	})
}

func TestOAuthAppLC_Lifecycle(t *testing.T) {
	l := newTL(t, "OAuthApp", "app")
	in := oauthIn()
	c := l.create(in)
	require.True(t, strings.HasPrefix(c.ID, "OQ"), c.ID)
	require.Equal(t, c.ID, getS(t, c.Properties, "sid"))
	require.Equal(t, "active", getS(t, c.Properties, "status"))
	require.Equal(t, "oauth-secret-"+c.ID, getS(t, c.Properties, "secret"))
	_, spec := loadSecretAuditSchema(t)
	require.True(t, spec.Resources["twilio:index:OAuthApp"].Properties["secret"].Secret, "the client secret is a secret output in the schema")
	require.True(t, spec.Functions["twilio:index:getMessagingDeactivations"].Outputs.Properties["redirectUrl"].Secret)

	// The create is one JSON call with every input.
	require.Len(t, l.f.oauthBodies, 1)
	b := l.f.oauthBodies[0]
	require.Equal(t, "ci app", b["friendly_name"])
	require.Equal(t, "used by CI", b["description"])
	require.Equal(t, float64(3600), b["access_token_ttl"])
	require.Equal(t, "client_secret_basic", b["token_endpoint_auth_method"])
	require.Equal(t, map[string]any{"allow": []any{"/twilio/iam/api-keys/create"}, "deny": []any{"/twilio/iam/roles/delete"}}, b["policy"])
	// #1980: Twilio refuses a create without the one accepted type, and wants the owning account.
	require.Equal(t, "Client Credential", b["type"])
	require.Equal(t, testAccount, b["owner_sid"])
	require.NotContains(t, b, "client_sid")

	// Idempotent Diff, and a refresh keeps the stored state without calling Twilio (there is no GET).
	requireNoChanges(t, l.diff(c.ID, c.Properties, in, in))
	l.f.resetCalls()
	rd := l.read(c.ID, c.Properties, in)
	require.Equal(t, c.ID, rd.ID)
	require.Equal(t, 0, l.f.callCount())
	require.Equal(t, getS(t, c.Properties, "secret"), getS(t, rd.Properties, "secret"))
	requireNoChanges(t, l.diff(c.ID, rd.Properties, in, rd.Inputs))

	// Removing an optional input is no change; changing any input replaces, since there is no update.
	requireNoChanges(t, l.diff(c.ID, c.Properties, without(in, "description", "policy", "accessTokenTtl"), in))
	for name, v := range map[string]property.Value{
		"friendlyName": sv("renamed"), "description": sv("other"), "type": sv("account"), "accessTokenTtl": nv(60),
		"ownerSid": sv("AC" + strings.Repeat("a", 32)), "clientSid": sv("OQ" + strings.Repeat("b", 32)),
		"policy": objv(map[string]property.Value{"allow": arrv(sv("/twilio/iam/api-keys/create"))}),
	} {
		d := l.diff(c.ID, c.Properties, with(in, map[string]property.Value{name: v}), in)
		require.Equal(t, pgp.UpdateReplace, d.DetailedDiff[name].Kind, name)
		require.True(t, d.HasChanges, name)
	}
	// Policy lists compare as sets.
	swapped := with(in, map[string]property.Value{"policy": objv(map[string]property.Value{"deny": arrv(sv("/twilio/iam/roles/delete")), "allow": arrv(sv("/twilio/iam/api-keys/create"))})})
	requireNoChanges(t, l.diff(c.ID, c.Properties, swapped, in))

	// Destroy deletes the app; one deleted outside Pulumi is tolerated.
	l.del(c.ID, c.Properties)
	require.Nil(t, l.f.get("oapp", testAccount, c.ID))
	require.Len(t, l.f.callsMatching("DELETE", "/iam/v1/Account/OAuthApps/"+c.ID), 1)
	l.del(c.ID, c.Properties) // 404 from Twilio: already gone
	require.Len(t, l.f.callsMatching("DELETE", "/iam/v1/Account/OAuthApps/"+c.ID), 2)
}

func TestOAuthAppLC_DeletedOutsideAndErrors(t *testing.T) {
	l := newTL(t, "OAuthApp", "app")
	c := l.create(pv(map[string]property.Value{"friendlyName": sv("min")}))
	l.f.remove("oapp", testAccount, c.ID)
	// A refresh cannot see the deletion (Twilio cannot read the app) and keeps the resource.
	require.Equal(t, c.ID, l.read(c.ID, c.Properties, pv(map[string]property.Value{"friendlyName": sv("min")})).ID)
	l.del(c.ID, c.Properties)

	// An import cannot read the app and says why.
	_, err := l.s.Read(pgp.ReadRequest{ID: c.ID, Urn: l.urn, Properties: empty(), Inputs: empty()})
	require.ErrorContains(t, err, "the Twilio API does not support reading OAuth apps")

	// A failed create is an error without state.
	l.f.failNext("POST", "/iam/v1/Account/OAuthApps", 403)
	_, err = l.s.Create(pgp.CreateRequest{Urn: l.urn, Properties: l.checked(pv(map[string]property.Value{"friendlyName": sv("x")}))})
	require.ErrorContains(t, err, "create OAuth app")

	// Preview does not call Twilio.
	l.f.resetCalls()
	_, err = l.s.Create(pgp.CreateRequest{Urn: l.urn, Properties: l.checked(oauthIn()), DryRun: true})
	require.NoError(t, err)
	require.Equal(t, 0, l.f.callCount())
}

func TestOAuthAppLC_Check(t *testing.T) {
	l := newTL(t, "OAuthApp", "app")
	require.Empty(t, l.check(pv(map[string]property.Value{"friendlyName": computed(), "ownerSid": computed(), "tokenEndpointAuthMethod": computed(), "accessTokenTtl": computed()})).Failures)
	bad := l.check(pv(map[string]property.Value{"friendlyName": sv(""), "ownerSid": sv("nope"), "clientSid": sv("AC" + strings.Repeat("a", 32)),
		"tokenEndpointAuthMethod": sv("none"), "accessTokenTtl": nv(0)}))
	require.Len(t, bad.Failures, 5)
}

func TestLookup_IAM(t *testing.T) {
	l := newTL(t, "OAuthApp", "app")
	// getOAuthApp cannot read an app: it fails with one fixed message, whatever the argument.
	for _, sid := range []string{"OQ" + strings.Repeat("a", 32), ""} {
		_, err := invoke(t, l, "getOAuthApp", map[string]property.Value{"sid": sv(sid)})
		require.EqualError(t, err, "the Twilio API does not support reading OAuth apps")
	}
	require.Empty(t, l.f.callsMatching("GET", "OAuthApps"))

	role := "IX" + strings.Repeat("a", 32)
	r, err := invoke(t, l, "getRolePermissions", map[string]property.Value{"roleSid": sv(role)})
	require.NoError(t, err)
	require.Equal(t, role, getS(t, r, "roleSid"))
	perms := r.Get("permissions").AsArray().AsSlice() // both pages
	require.Len(t, perms, 2)
	p0 := perms[0].AsMap()
	require.Equal(t, "read", p0.Get("action").AsString())
	require.Equal(t, "iam", p0.Get("product").AsString())
	require.Equal(t, "Can read roles", p0.Get("externalDescription").AsString())
	require.Equal(t, "write", perms[1].AsMap().Get("action").AsString())
	require.Len(t, l.f.callsMatching("GET", "/Permissions"), 2)

	_, err = invoke(t, l, "getRolePermissions", map[string]property.Value{"roleSid": sv("IX" + strings.Repeat("0", 32))})
	require.ErrorContains(t, err, "was not found")
	_, err = invoke(t, l, "getRolePermissions", map[string]property.Value{"roleSid": sv("")})
	require.ErrorContains(t, err, "must be set")
	_, err = invoke(t, l, "getRolePermissions", map[string]property.Value{"roleSid": sv("nope")})
	require.ErrorContains(t, err, "roleSid must be the SID of a role")
}

func TestLookup_MessagingAccount(t *testing.T) {
	l := newTL(t, "MessagingService", "svc")
	r, err := invoke(t, l, "getMessagingUsecases", map[string]property.Value{})
	require.NoError(t, err)
	uc := r.Get("usecases").AsArray().AsSlice()
	require.Len(t, uc, 2)
	require.Equal(t, "notifications", uc[0].AsMap().Get("usecase").AsString())
	require.Equal(t, "Promotions", uc[1].AsMap().Get("purpose").AsString())

	r, err = invoke(t, l, "getMessagingDeactivations", map[string]property.Value{"date": sv("2026-11-12")})
	require.NoError(t, err)
	require.Equal(t, "2026-11-12", getS(t, r, "date"))
	require.Equal(t, "https://api.example.invalid/deactivated/2026-11-12.csv?token=abc", getS(t, r, "redirectUrl"))
	require.True(t, r.Get("redirectUrl").Secret(), "the authenticated URL is a secret output")
	// The redirect is not followed: only the one request to Twilio was made.
	require.Len(t, l.f.callsMatching("GET", "/messaging/v1/Deactivations"), 1)
	require.Equal(t, "2026-11-12", l.f.callsMatching("GET", "/messaging/v1/Deactivations")[0].Query.Get("Date"))

	for _, bad := range []string{"", "12-11-2026", "2026-13-01", "yesterday"} {
		_, err = invoke(t, l, "getMessagingDeactivations", map[string]property.Value{"date": sv(bad)})
		require.Error(t, err, bad)
	}
	l.f.failNext("GET", "/messaging/v1/Deactivations", 403)
	_, err = invoke(t, l, "getMessagingDeactivations", map[string]property.Value{"date": sv("2026-11-12")})
	require.ErrorContains(t, err, "HTTP 403")
}

// #1980: a minimal program (no type) creates the app with Twilio's one accepted type, a later
// preview plans no change, and an explicit unsupported type is refused by Twilio, not hidden.
func TestOAuthAppLC_DefaultType(t *testing.T) {
	l := newTL(t, "OAuthApp", "app")
	in := pv(map[string]property.Value{"friendlyName": sv("minimal"),
		"policy": objv(map[string]property.Value{"allow": arrv(sv("/twilio/iam/api-keys/create"))})})
	c := l.create(in)
	require.Len(t, l.f.oauthBodies, 1)
	require.Equal(t, "Client Credential", l.f.oauthBodies[0]["type"])
	require.Equal(t, "Client Credential", getS(t, c.Properties, "type"))
	requireNoChanges(t, l.diff(c.ID, c.Properties, in, in))

	bad := pv(map[string]property.Value{"friendlyName": sv("bad"), "type": sv("confidential")})
	_, err := l.s.Create(pgp.CreateRequest{Urn: l.urn, Properties: l.checked(bad)})
	require.ErrorContains(t, err, "oauth app type is invalid")
}
