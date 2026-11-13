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

func TestSyncServiceLC_Lifecycle(t *testing.T) {
	l := newTL(t, "SyncService", "sync")
	in := pv(map[string]property.Value{"friendlyName": sv("realtime"), "aclEnabled": bv(true), "reachabilityDebouncingWindow": nv(2000)})
	c := l.create(in)
	require.True(t, strings.HasPrefix(c.ID, "IS"), c.ID)
	require.Equal(t, c.ID, getS(t, c.Properties, "sid"))
	form := l.f.callsMatching("POST", "/sync/v1/Services")[0].Form
	require.Equal(t, "realtime", form.Get("FriendlyName"))
	require.Equal(t, "true", form.Get("AclEnabled"))
	require.Equal(t, "2000", form.Get("ReachabilityDebouncingWindow"))
	require.Empty(t, form["WebhookUrl"])
	requireNoChanges(t, l.diff(c.ID, c.Properties, in, in))
	rd := l.read(c.ID, c.Properties, in)
	requireNoChanges(t, l.diff(c.ID, rd.Properties, in, rd.Inputs))

	// Drift in a setting the program states shows; a setting it leaves out does not.
	l.f.mutate("sync", testAccount, c.ID, func(o map[string]any) { o["acl_enabled"] = false; o["webhooks_from_rest_enabled"] = true })
	rd = l.read(c.ID, c.Properties, in)
	require.False(t, rd.Inputs.Get("aclEnabled").AsBool())
	require.Equal(t, pgp.Update, l.diff(c.ID, rd.Properties, in, rd.Inputs).DetailedDiff["aclEnabled"].Kind)
	require.False(t, hasProp(rd.Inputs, "webhooksFromRestEnabled"))

	// Every change is an update in place.
	next := with(in, map[string]property.Value{"friendlyName": sv("realtime-2"), "webhookUrl": sv("https://example.com/sync"), "aclEnabled": bv(false)})
	d := l.diff(c.ID, c.Properties, next, in)
	require.Equal(t, pgp.Update, d.DetailedDiff["friendlyName"].Kind)
	require.Equal(t, pgp.Update, d.DetailedDiff["webhookUrl"].Kind)
	l.f.resetCalls()
	u := l.update(c.ID, c.Properties, next, in)
	require.Equal(t, "realtime-2", l.f.callsMatching("POST", "/sync/v1/Services/"+c.ID)[0].Form.Get("FriendlyName"))
	require.Equal(t, "https://example.com/sync", l.f.get("sync", testAccount, c.ID)["webhook_url"])
	requireNoChanges(t, l.diff(c.ID, u.Properties, next, next))

	// Import adopts what differs from the defaults.
	imp := l.read(c.ID, empty(), empty())
	require.Equal(t, "realtime-2", getS(t, imp.Inputs, "friendlyName"))
	require.Equal(t, "https://example.com/sync", getS(t, imp.Inputs, "webhookUrl"))
	require.True(t, imp.Inputs.Get("reachabilityDebouncingWindow").AsNumber() == 2000)
	require.True(t, imp.Inputs.Get("webhooksFromRestEnabled").AsBool())
	require.False(t, hasProp(imp.Inputs, "aclEnabled"))
	l.requireImportFails("IS" + strings.Repeat("0", 32))

	// Deleted outside Pulumi.
	l.f.remove("sync", testAccount, c.ID)
	require.Empty(t, l.read(c.ID, c.Properties, in).ID)
	l.del(c.ID, c.Properties)
	c = l.create(in)
	l.del(c.ID, c.Properties)
	require.Nil(t, l.f.get("sync", testAccount, c.ID))

	chk := l.check(pv(map[string]property.Value{"reachabilityDebouncingWindow": nv(10)}))
	require.Len(t, chk.Failures, 1)
}

func TestLookup_Sync(t *testing.T) {
	l := newTL(t, "SyncService", "sync")
	svc := l.create(pv(map[string]property.Value{"friendlyName": sv("realtime"), "aclEnabled": bv(true)})).ID
	r, err := invoke(t, l, "getSyncService", map[string]property.Value{"sid": sv(svc)})
	require.NoError(t, err)
	require.Equal(t, "realtime", getS(t, r, "friendlyName"))
	require.True(t, r.Get("aclEnabled").AsBool())
	require.False(t, hasProp(r, "webhookUrl"))
	_, err = invoke(t, l, "getSyncService", map[string]property.Value{"sid": sv("ISmissing")})
	require.ErrorContains(t, err, "was not found")
	_, err = invoke(t, l, "getSyncService", map[string]property.Value{"sid": sv("")})
	require.ErrorContains(t, err, "must be set")
}
