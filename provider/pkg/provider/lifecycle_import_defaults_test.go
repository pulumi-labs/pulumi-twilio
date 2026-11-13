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

	"github.com/pulumi/pulumi/sdk/v3/go/property"
	"github.com/stretchr/testify/require"
)

// An import leaves an input unset when Twilio's value is the default, so a program that writes the
// default explicitly must plan no change, for every Conversations, Notify and Sync resource that
// adopts defaults this way (#1888, the same defect #1880 fixed for TaskRouter). A non-default
// value must still plan a change.
func TestImportThenExplicitDefaultsPlansNothing(t *testing.T) {
	l := newTL(t, "ConversationsService", "svc")
	svc := cvSvc(l)
	m := func(kv ...any) map[string]property.Value {
		out := map[string]property.Value{}
		for i := 0; i < len(kv); i += 2 {
			switch v := kv[i+1].(type) {
			case string:
				out[kv[i].(string)] = sv(v)
			case bool:
				out[kv[i].(string)] = bv(v)
			case int:
				out[kv[i].(string)] = nv(v)
			}
		}
		return out
	}
	join := func(a, b map[string]property.Value) map[string]property.Value {
		out := map[string]property.Value{}
		for k, v := range a {
			out[k] = v
		}
		for k, v := range b {
			out[k] = v
		}
		return out
	}

	type tc struct {
		typ      string
		minimal  map[string]property.Value // creates the object with Twilio defaults
		defaults map[string]property.Value // the same program with every default written out
		changed  map[string]property.Value // a non-default value that must still plan a change
	}
	apn := m("credentialType", "apn", "certificate", "CERT", "privateKey", "KEY")
	cases := []tc{
		{"NotifyService", m("friendlyName", "push"),
			m("logEnabled", true, "deliveryCallbackEnabled", false, "apnCredentialSid", "", "messagingServiceSid", "", "deliveryCallbackUrl", ""),
			m("logEnabled", false)},
		{"NotifyCredential", apn, m("sandbox", false), m("sandbox", true)},
		{"SyncService", m("friendlyName", "rt"),
			m("reachabilityDebouncingWindow", 5000, "reachabilityWebhooksEnabled", false, "aclEnabled", false, "reachabilityDebouncingEnabled", false,
				"webhooksFromRestEnabled", false, "webhookUrl", ""),
			m("reachabilityDebouncingWindow", 2000)},
		{"ConversationsCredential", apn, m("sandbox", false), m("sandbox", true)},
		{"ConversationsConfiguration", m("defaultChatServiceSid", svc),
			m("defaultMessagingServiceSid", "", "defaultInactiveTimer", "", "defaultClosedTimer", ""),
			m("defaultClosedTimer", "P1D")},
		{"ConversationsServiceConfiguration", m("chatServiceSid", svc),
			m("reachabilityEnabled", false, "defaultChatServiceRoleSid", ""), m("reachabilityEnabled", true)},
		{"ConversationsServiceNotificationConfiguration", m("chatServiceSid", svc),
			m("logEnabled", false), m("logEnabled", true)},
		{"ConversationsWebhookConfiguration", m("postWebhookUrl", "https://example.com/post"),
			m("preWebhookUrl", ""), m("method", "get")},
		{"ConversationsAddressConfiguration", m("addressType", "whatsapp", "address", "whatsapp:+15551234567"),
			m("friendlyName", "", "addressCountry", ""), m("friendlyName", "support")},
	}
	for _, c := range cases {
		lc := l.other(c.typ, "x")
		created := lc.create(pv(c.minimal))
		imp := lc.read(created.ID, empty(), empty())
		require.NotEmpty(t, imp.ID, c.typ)
		state, old := imp.Properties, imp.Inputs
		if c.typ == "NotifyCredential" || c.typ == "ConversationsCredential" {
			// Twilio never returns the certificate or key, so an import cannot know them.
			secrets := map[string]property.Value{"certificate": sv("CERT"), "privateKey": sv("KEY")}
			state, old = with(state, secrets), with(old, secrets)
		}
		requireNoChanges(t, lc.diff(imp.ID, state, pv(join(c.minimal, c.defaults)), old))
		require.True(t, lc.diff(imp.ID, imp.Properties, pv(join(c.minimal, c.changed)), old).HasChanges, "%s: a non-default value still plans a change", c.typ)
	}

	// Notification settings: the nested objects an import leaves unset are disabled with empty text.
	nl := l.other("ConversationsServiceNotificationConfiguration", "nested")
	ncreated := nl.create(pv(m("chatServiceSid", svc)))
	nimp := nl.read(ncreated.ID, empty(), empty())
	obj := func(kv map[string]property.Value) property.Value { return property.New(property.NewMap(kv)) }
	nDefaults := with(nimp.Inputs, map[string]property.Value{
		"newMessage": obj(map[string]property.Value{"enabled": bv(false), "template": sv(""), "badgeCountEnabled": bv(false),
			"withMedia": obj(map[string]property.Value{"enabled": bv(false), "template": sv("")})}),
		"addedToConversation":     obj(map[string]property.Value{"enabled": bv(false), "template": sv(""), "sound": sv("")}),
		"removedFromConversation": obj(map[string]property.Value{"enabled": bv(false)}),
	})
	requireNoChanges(t, nl.diff(nimp.ID, nimp.Properties, nDefaults, nimp.Inputs))
	for _, key := range []string{"newMessage", "addedToConversation", "removedFromConversation"} {
		on := with(nimp.Inputs, map[string]property.Value{key: obj(map[string]property.Value{"enabled": bv(true)})})
		require.True(t, nl.diff(nimp.ID, nimp.Properties, on, nimp.Inputs).HasChanges, key)
	}
	badge := with(nimp.Inputs, map[string]property.Value{"newMessage": obj(map[string]property.Value{"withMedia": obj(map[string]property.Value{"enabled": bv(true)})})})
	require.True(t, nl.diff(nimp.ID, nimp.Properties, badge, nimp.Inputs).HasChanges, "withMedia")

	// An address configuration's auto-creation: an unset old value is the default (disabled, empty, 0).
	al := l.other("ConversationsAddressConfiguration", "auto")
	created := al.create(pv(m("addressType", "whatsapp", "address", "whatsapp:+15551234568")))
	imp := al.read(created.ID, empty(), empty())
	off := property.New(property.NewMap(map[string]property.Value{"enabled": bv(false), "webhookUrl": sv(""), "studioRetryCount": nv(0)}))
	requireNoChanges(t, al.diff(imp.ID, imp.Properties, with(imp.Inputs, map[string]property.Value{"autoCreation": off}), imp.Inputs))
	on := property.New(property.NewMap(map[string]property.Value{"enabled": bv(true)}))
	require.True(t, al.diff(imp.ID, imp.Properties, with(imp.Inputs, map[string]property.Value{"autoCreation": on}), imp.Inputs).HasChanges)
}
