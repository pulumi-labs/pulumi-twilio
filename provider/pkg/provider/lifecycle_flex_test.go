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

const (
	flexChat = "IS00000000000000000000000000000001"
	flexWS   = "WS00000000000000000000000000000001"
	flexWF   = "WW00000000000000000000000000000001"
)

func TestFlexFlowLC_Lifecycle(t *testing.T) {
	l := newTL(t, "FlexFlow", "flow")
	in := pv(map[string]property.Value{"friendlyName": sv("sms to agents"), "chatServiceSid": sv(flexChat), "channelType": sv("sms"),
		"contactIdentity": sv("+15555550100"), "integrationType": sv("task"), "integrationWorkspaceSid": sv(flexWS),
		"integrationWorkflowSid": sv(flexWF), "integrationPriority": nv(5), "longLived": bv(true)})
	c := l.create(in)
	require.True(t, strings.HasPrefix(c.ID, "FO"), c.ID)
	form := l.f.callsMatching("POST", "/flex-api/v1/FlexFlows")[0].Form
	require.Equal(t, flexWS, form.Get("Integration.WorkspaceSid"))
	require.Equal(t, "5", form.Get("Integration.Priority"))
	require.Equal(t, "task", form.Get("IntegrationType"))
	require.Equal(t, "true", form.Get("LongLived"))
	require.Empty(t, form["Enabled"])
	require.Empty(t, form["Integration.Timeout"])
	requireNoChanges(t, l.diff(c.ID, c.Properties, in, in))

	// Stating the defaults Twilio filled in is no change, and a refresh does not turn them into inputs.
	explicit := with(in, map[string]property.Value{"integrationChannel": sv("default"), "integrationTimeout": nv(86400), "janitorEnabled": bv(false)})
	requireNoChanges(t, l.diff(c.ID, c.Properties, explicit, in))
	rd := l.read(c.ID, c.Properties, in)
	requireNoChanges(t, l.diff(c.ID, rd.Properties, in, rd.Inputs))
	require.False(t, hasProp(rd.Inputs, "integrationChannel"))
	require.False(t, hasProp(rd.Inputs, "integrationTimeout"))
	require.False(t, hasProp(rd.Inputs, "enabled"))

	// Drift in a setting the program states shows; one it leaves out does not.
	l.f.mutate("flex", testAccount, "FlexFlows/"+c.ID, func(o map[string]any) {
		o["friendly_name"] = "renamed"
		o["integration"].(map[string]any)["priority"] = float64(9)
		o["integration"].(map[string]any)["timeout"] = float64(60)
		o["enabled"] = false
	})
	rd = l.read(c.ID, c.Properties, in)
	require.Equal(t, "renamed", getS(t, rd.Inputs, "friendlyName"))
	require.Equal(t, 9.0, rd.Inputs.Get("integrationPriority").AsNumber())
	d := l.diff(c.ID, rd.Properties, in, rd.Inputs)
	require.Equal(t, pgp.Update, d.DetailedDiff["friendlyName"].Kind)
	require.Equal(t, pgp.Update, d.DetailedDiff["integrationPriority"].Kind)
	require.False(t, hasProp(rd.Inputs, "enabled"))
	require.False(t, hasProp(rd.Inputs, "integrationTimeout"))

	// Everything updates in place, and the update sends the whole integration.
	next := with(in, map[string]property.Value{"friendlyName": sv("sms v2"), "channelType": sv("whatsapp"), "enabled": bv(false),
		"integrationPriority": nv(1), "integrationTimeout": nv(120), "integrationCreationOnMessage": bv(true), "janitorEnabled": bv(true)})
	d = l.diff(c.ID, c.Properties, next, in)
	for _, k := range []string{"friendlyName", "channelType", "enabled", "integrationPriority", "integrationTimeout", "integrationCreationOnMessage", "janitorEnabled"} {
		require.Equal(t, pgp.Update, d.DetailedDiff[k].Kind, k)
	}
	l.f.resetCalls()
	u := l.update(c.ID, c.Properties, next, in)
	uf := l.f.callsMatching("POST", "/flex-api/v1/FlexFlows/"+c.ID)[0].Form
	require.Equal(t, "120", uf.Get("Integration.Timeout"))
	require.Equal(t, "true", uf.Get("Integration.CreationOnMessage"))
	require.Equal(t, flexWS, uf.Get("Integration.WorkspaceSid"))
	require.Equal(t, "whatsapp", l.f.get("flex", testAccount, "FlexFlows/"+c.ID)["channel_type"])
	requireNoChanges(t, l.diff(c.ID, u.Properties, next, next))
	rd = l.read(c.ID, u.Properties, next)
	requireNoChanges(t, l.diff(c.ID, rd.Properties, next, rd.Inputs))

	// A studio flow with the retry count Twilio defaults to.
	studio := pv(map[string]property.Value{"friendlyName": sv("studio"), "chatServiceSid": sv(flexChat), "channelType": sv("web"),
		"integrationType": sv("studio"), "integrationFlowSid": sv("FW00000000000000000000000000000001")})
	sc := l.create(studio)
	sr := l.read(sc.ID, sc.Properties, studio)
	require.False(t, hasProp(sr.Inputs, "integrationRetryCount"))
	requireNoChanges(t, l.diff(sc.ID, sr.Properties, with(studio, map[string]property.Value{"integrationRetryCount": nv(3)}), sr.Inputs))

	// Import adopts what is not a default and leaves the undocumented bools alone.
	imp := l.read(c.ID, empty(), empty())
	require.Empty(t, l.check(imp.Inputs).Failures)
	require.Equal(t, "sms v2", getS(t, imp.Inputs, "friendlyName"))
	require.Equal(t, flexChat, getS(t, imp.Inputs, "chatServiceSid"))
	require.Equal(t, "task", getS(t, imp.Inputs, "integrationType"))
	require.Equal(t, 120.0, imp.Inputs.Get("integrationTimeout").AsNumber())
	require.True(t, imp.Inputs.Get("janitorEnabled").AsBool())
	require.False(t, hasProp(imp.Inputs, "enabled"))
	require.False(t, hasProp(imp.Inputs, "integrationCreationOnMessage"))
	l.requireImportFails("FO" + strings.Repeat("0", 32))

	// Destroy deletes the Flex Flow; one already gone is not an error.
	l.del(sc.ID, sc.Properties)
	require.Nil(t, l.f.get("flex", testAccount, "FlexFlows/"+sc.ID))
	l.f.remove("flex", testAccount, "FlexFlows/"+c.ID)
	require.Empty(t, l.read(c.ID, u.Properties, next).ID)
	l.del(c.ID, u.Properties)
	require.Len(t, l.f.callsMatching("DELETE", "/flex-api/v1/FlexFlows/"+sc.ID), 1)

	ok := map[string]property.Value{"friendlyName": sv("f"), "chatServiceSid": sv(flexChat), "channelType": sv("sms")}
	bad := l.check(pv(map[string]property.Value{"friendlyName": sv(""), "chatServiceSid": sv("x"), "channelType": sv("fax"), "integrationType": sv("mail"), "integrationRetryCount": nv(4)}))
	require.Len(t, bad.Failures, 5)
	for _, tc := range []struct {
		typ  string
		want int
	}{{"studio", 1}, {"external", 1}, {"task", 2}} {
		require.Len(t, l.check(with(pv(ok), map[string]property.Value{"integrationType": sv(tc.typ)})).Failures, tc.want, tc.typ)
	}
	require.Empty(t, l.check(with(pv(ok), map[string]property.Value{"integrationType": sv("external"), "integrationUrl": sv("https://example.com/hook"), "integrationRetryCount": nv(0)})).Failures)
	// Values wired from other resources are unknown at preview.
	require.Empty(t, l.check(pv(map[string]property.Value{"friendlyName": sv("f"), "chatServiceSid": computed(), "channelType": sv("sms"),
		"integrationType": sv("studio"), "integrationFlowSid": computed()})).Failures)
}

func TestLookup_FlexFlow(t *testing.T) {
	l := newTL(t, "FlexFlow", "flow")
	mk := func(name, typ string) string {
		return l.create(pv(map[string]property.Value{"friendlyName": sv(name), "chatServiceSid": sv(flexChat), "channelType": sv(typ),
			"integrationType": sv("external"), "integrationUrl": sv("https://example.com/" + name)})).ID
	}
	a, b, _ := mk("a", "sms"), mk("b", "web"), mk("c", "line")

	r, err := invoke(t, l, "getFlexFlow", map[string]property.Value{"sid": sv(a)})
	require.NoError(t, err)
	require.Equal(t, "a", getS(t, r, "friendlyName"))
	require.Equal(t, flexChat, getS(t, r, "chatServiceSid"))
	require.Equal(t, "https://example.com/a", getS(t, r, "integrationUrl"))
	require.Equal(t, 3.0, r.Get("integrationRetryCount").AsNumber())
	require.True(t, r.Get("enabled").AsBool())
	require.False(t, hasProp(r, "contactIdentity"))
	require.False(t, hasProp(r, "integrationFlowSid"))
	_, err = invoke(t, l, "getFlexFlow", map[string]property.Value{"sid": sv("FOmissing")})
	require.ErrorContains(t, err, "was not found")
	_, err = invoke(t, l, "getFlexFlow", map[string]property.Value{"sid": sv("")})
	require.ErrorContains(t, err, "must be set")

	// The list pages (two a page in the fake) and the name filter narrows it.
	r, err = invoke(t, l, "getFlexFlows", map[string]property.Value{})
	require.NoError(t, err)
	require.Len(t, r.Get("flexFlows").AsArray().AsSlice(), 3)
	r, err = invoke(t, l, "getFlexFlows", map[string]property.Value{"friendlyName": sv("b")})
	require.NoError(t, err)
	got := r.Get("flexFlows").AsArray().AsSlice()
	require.Len(t, got, 1)
	require.Equal(t, b, getS(t, got[0].AsMap(), "sid"))
}
