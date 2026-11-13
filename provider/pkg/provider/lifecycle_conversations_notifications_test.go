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

func cvObj(m map[string]property.Value) property.Value { return property.New(property.NewMap(m)) }

func TestConversationsServiceNotificationConfigurationLC_Lifecycle(t *testing.T) {
	l := newTL(t, "ConversationsServiceNotificationConfiguration", "notif")
	svc := cvSvc(l)
	in := pv(map[string]property.Value{
		"chatServiceSid": sv(svc),
		"logEnabled":     bv(true),
		"newMessage": cvObj(map[string]property.Value{
			"enabled": bv(true), "template": sv("New message"), "badgeCountEnabled": bv(true),
			"withMedia": cvObj(map[string]property.Value{"enabled": bv(true), "template": sv("New media")}),
		}),
		"addedToConversation": cvObj(map[string]property.Value{"enabled": bv(true), "sound": sv("chime")}),
	})
	c := l.create(in)
	require.Equal(t, svc, c.ID)
	form := l.f.callsMatching("POST", "/Configuration/Notifications")[0].Form
	require.Equal(t, "true", form.Get("LogEnabled"))
	require.Equal(t, "New message", form.Get("NewMessage.Template"))
	require.Equal(t, "true", form.Get("NewMessage.BadgeCountEnabled"))
	require.Equal(t, "New media", form.Get("NewMessage.WithMedia.Template"))
	require.Equal(t, "chime", form.Get("AddedToConversation.Sound"))
	require.Empty(t, form["RemovedFromConversation.Enabled"])
	requireNoChanges(t, l.diff(c.ID, c.Properties, in, in))
	rd := l.read(c.ID, c.Properties, in)
	requireNoChanges(t, l.diff(c.ID, rd.Properties, in, rd.Inputs))
	require.False(t, hasProp(rd.Inputs, "removedFromConversation"))

	// Drift in a nested setting the program states shows; one it leaves out does not.
	l.f.mutate("cvone", testAccount, "svcnotif/"+svc, func(o map[string]any) {
		o["new_message"].(map[string]any)["with_media"].(map[string]any)["enabled"] = false
		o["removed_from_conversation"].(map[string]any)["enabled"] = true
	})
	rd = l.read(c.ID, c.Properties, in)
	require.False(t, rd.Inputs.Get("newMessage").AsMap().Get("withMedia").AsMap().Get("enabled").AsBool())
	require.Equal(t, pgp.Update, l.diff(c.ID, rd.Properties, in, rd.Inputs).DetailedDiff["newMessage"].Kind)
	require.False(t, hasProp(rd.Inputs, "removedFromConversation"))

	next := with(in, map[string]property.Value{"addedToConversation": cvObj(map[string]property.Value{"enabled": bv(false)})})
	require.Equal(t, pgp.Update, l.diff(c.ID, c.Properties, next, in).DetailedDiff["addedToConversation"].Kind)
	l.f.resetCalls()
	u := l.update(c.ID, c.Properties, next, in)
	require.Equal(t, "false", l.f.callsMatching("POST", "/Configuration/Notifications")[0].Form.Get("AddedToConversation.Enabled"))
	requireNoChanges(t, l.diff(c.ID, u.Properties, next, next))
	requireNoChanges(t, l.diff(c.ID, c.Properties, without(in, "logEnabled"), in))
	require.Equal(t, pgp.UpdateReplace, l.diff(c.ID, u.Properties, with(next, map[string]property.Value{"chatServiceSid": sv("IS" + strings.Repeat("1", 32))}), next).DetailedDiff["chatServiceSid"].Kind)

	// Import adopts what is not at its default.
	imp := l.read(svc, empty(), empty())
	require.Equal(t, svc, getS(t, imp.Inputs, "chatServiceSid"))
	require.True(t, imp.Inputs.Get("logEnabled").AsBool())
	require.Equal(t, "New message", getS(t, imp.Inputs.Get("newMessage").AsMap(), "template"))
	require.Equal(t, "New media", getS(t, imp.Inputs.Get("newMessage").AsMap().Get("withMedia").AsMap(), "template"))
	require.True(t, imp.Inputs.Get("removedFromConversation").AsMap().Get("enabled").AsBool())
	l.requireImportFails("IS" + strings.Repeat("0", 32))

	// Destroying forgets it: no request, the settings stay.
	l.f.resetCalls()
	l.del(c.ID, u.Properties)
	require.Empty(t, l.f.callsMatching("DELETE", "/"))
	require.Equal(t, true, l.f.get("cvone", testAccount, "svcnotif/"+svc)["log_enabled"])

	// A service removed outside Pulumi takes its settings with it.
	l.f.remove("cv", testAccount, "Services//"+svc)
	require.Empty(t, l.read(c.ID, c.Properties, in).ID)

	// With nothing to set it only checks that the service exists.
	l.f.resetCalls()
	svc2 := cvSvc(l)
	l.create(pv(map[string]property.Value{"chatServiceSid": sv(svc2)}))
	require.Empty(t, l.f.callsMatching("POST", "/Configuration/Notifications"))
	_, err := l.s.Create(pgp.CreateRequest{Urn: l.urn, Properties: pv(map[string]property.Value{"chatServiceSid": sv("IS" + strings.Repeat("9", 32))})})
	require.Error(t, err)

	require.Len(t, l.check(pv(map[string]property.Value{"chatServiceSid": sv("nope")})).Failures, 1)
}

func TestConversationsConfigurationLC_Lifecycle(t *testing.T) {
	l := newTL(t, "ConversationsConfiguration", "acct")
	svc := cvSvc(l)
	in := pv(map[string]property.Value{"defaultChatServiceSid": sv(svc), "defaultInactiveTimer": sv("PT30M"), "defaultClosedTimer": sv("P1D")})
	c := l.create(in)
	require.Equal(t, "account", c.ID)
	form := l.f.callsMatching("POST", "/conversations/v1/Configuration")[0].Form
	require.Equal(t, svc, form.Get("DefaultChatServiceSid"))
	require.Equal(t, "PT30M", form.Get("DefaultInactiveTimer"))
	require.Empty(t, form["DefaultMessagingServiceSid"])
	requireNoChanges(t, l.diff(c.ID, c.Properties, in, in))
	rd := l.read(c.ID, c.Properties, in)
	requireNoChanges(t, l.diff(c.ID, rd.Properties, in, rd.Inputs))

	l.f.mutate("cvone", testAccount, "acctcfg/", func(o map[string]any) { o["default_closed_timer"] = "P2D" })
	rd = l.read(c.ID, c.Properties, in)
	require.Equal(t, "P2D", getS(t, rd.Inputs, "defaultClosedTimer"))
	require.Equal(t, pgp.Update, l.diff(c.ID, rd.Properties, in, rd.Inputs).DetailedDiff["defaultClosedTimer"].Kind)
	requireNoChanges(t, l.diff(c.ID, c.Properties, without(in, "defaultClosedTimer"), in))

	mg := "MG" + strings.Repeat("a", 32)
	next := with(in, map[string]property.Value{"defaultMessagingServiceSid": sv(mg)})
	l.f.resetCalls()
	u := l.update(c.ID, c.Properties, next, in)
	require.Equal(t, mg, l.f.callsMatching("POST", "/conversations/v1/Configuration")[0].Form.Get("DefaultMessagingServiceSid"))
	requireNoChanges(t, l.diff(c.ID, u.Properties, next, next))

	imp := l.read("account", empty(), empty())
	require.Equal(t, mg, getS(t, imp.Inputs, "defaultMessagingServiceSid"))
	require.Equal(t, svc, getS(t, imp.Inputs, "defaultChatServiceSid"))
	_, err := l.s.Read(pgp.ReadRequest{ID: "other", Urn: l.urn, Properties: empty(), Inputs: empty()})
	require.ErrorContains(t, err, "import ID")

	l.f.resetCalls()
	l.del(c.ID, u.Properties)
	require.Empty(t, l.f.callsMatching("DELETE", "/"))
	require.Equal(t, mg, l.f.get("cvone", testAccount, "acctcfg/")["default_messaging_service_sid"])

	require.Len(t, l.check(pv(map[string]property.Value{"defaultChatServiceSid": sv("nope"), "defaultMessagingServiceSid": sv("nope")})).Failures, 2)
}

func TestLookup_ConversationsNotificationsAndConfiguration(t *testing.T) {
	l := newTL(t, "ConversationsServiceNotificationConfiguration", "notif")
	svc := cvSvc(l)
	r, err := invoke(t, l, "getConversationsServiceNotificationConfiguration", map[string]property.Value{"chatServiceSid": sv(svc)})
	require.NoError(t, err)
	require.False(t, r.Get("logEnabled").AsBool())
	require.False(t, r.Get("newMessage").AsMap().Get("enabled").AsBool())
	require.False(t, hasProp(r.Get("newMessage").AsMap(), "template"))
	_, err = invoke(t, l, "getConversationsServiceNotificationConfiguration", map[string]property.Value{"chatServiceSid": sv("ISmissing")})
	require.ErrorContains(t, err, "was not found")
	_, err = invoke(t, l, "getConversationsServiceNotificationConfiguration", map[string]property.Value{"chatServiceSid": sv("")})
	require.ErrorContains(t, err, "must be set")

	l.other("ConversationsConfiguration", "acct").create(pv(map[string]property.Value{"defaultChatServiceSid": sv(svc), "defaultClosedTimer": sv("P1D")}))
	cfg, err := invoke(t, l, "getConversationsConfiguration", map[string]property.Value{})
	require.NoError(t, err)
	require.Equal(t, svc, getS(t, cfg, "defaultChatServiceSid"))
	require.Equal(t, "P1D", getS(t, cfg, "defaultClosedTimer"))
	require.False(t, hasProp(cfg, "defaultMessagingServiceSid"))
}
