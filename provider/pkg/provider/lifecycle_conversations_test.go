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

func strs(vs ...string) property.Value {
	l := make([]property.Value, len(vs))
	for i, v := range vs {
		l[i] = sv(v)
	}
	return property.New(property.NewArray(l))
}

func cvSvc(l *tl) string {
	return l.other("ConversationsService", "svc").create(pv(map[string]property.Value{"friendlyName": sv("support")})).ID
}

func TestConversationsServiceLC_Lifecycle(t *testing.T) {
	l := newTL(t, "ConversationsService", "svc")
	in := pv(map[string]property.Value{"friendlyName": sv("support")})
	c := l.create(in)
	require.True(t, strings.HasPrefix(c.ID, "IS"), c.ID)
	require.Equal(t, c.ID, getS(t, c.Properties, "sid"))
	require.Equal(t, "support", l.f.callsMatching("POST", "/conversations/v1/Services")[0].Form.Get("FriendlyName"))
	requireNoChanges(t, l.diff(c.ID, c.Properties, in, in))
	rd := l.read(c.ID, c.Properties, in)
	requireNoChanges(t, l.diff(c.ID, rd.Properties, in, rd.Inputs))

	// Twilio cannot rename a service, so the name replaces it.
	next := with(in, map[string]property.Value{"friendlyName": sv("support-2")})
	require.Equal(t, pgp.UpdateReplace, l.diff(c.ID, c.Properties, next, in).DetailedDiff["friendlyName"].Kind)

	imp := l.read(c.ID, empty(), empty())
	require.Equal(t, "support", getS(t, imp.Inputs, "friendlyName"))
	l.requireImportFails("IS" + strings.Repeat("0", 32))

	l.f.remove("cv", testAccount, "Services//"+c.ID)
	require.Empty(t, l.read(c.ID, c.Properties, in).ID)
	l.del(c.ID, c.Properties)
	c = l.create(in)
	l.del(c.ID, c.Properties)
	require.Nil(t, l.f.get("cv", testAccount, "Services//"+c.ID))

	chk := l.check(pv(map[string]property.Value{"friendlyName": sv("")}))
	require.Len(t, chk.Failures, 1)
}

func TestConversationsServiceConfigurationLC_Lifecycle(t *testing.T) {
	l := newTL(t, "ConversationsServiceConfiguration", "cfg")
	svc := cvSvc(l)
	role := l.other("ConversationsRole", "role").create(pv(map[string]property.Value{
		"friendlyName": sv("admin"), "roleType": sv("service"), "permissions": strs("createConversation"), "chatServiceSid": sv(svc)})).Properties
	roleSid := getS(t, role, "sid")
	in := pv(map[string]property.Value{"chatServiceSid": sv(svc), "defaultChatServiceRoleSid": sv(roleSid), "reachabilityEnabled": bv(true)})
	c := l.create(in)
	require.Equal(t, svc, c.ID)
	form := l.f.callsMatching("POST", "/Configuration")[0].Form
	require.Equal(t, roleSid, form.Get("DefaultChatServiceRoleSid"))
	require.Equal(t, "true", form.Get("ReachabilityEnabled"))
	require.Empty(t, form["DefaultConversationRoleSid"])
	requireNoChanges(t, l.diff(c.ID, c.Properties, in, in))
	rd := l.read(c.ID, c.Properties, in)
	requireNoChanges(t, l.diff(c.ID, rd.Properties, in, rd.Inputs))

	// Drift in a setting the program states shows; a setting it leaves out does not.
	l.f.mutate("cvone", testAccount, "svccfg/"+svc, func(o map[string]any) { o["reachability_enabled"] = false })
	rd = l.read(c.ID, c.Properties, in)
	require.False(t, rd.Inputs.Get("reachabilityEnabled").AsBool())
	require.Equal(t, pgp.Update, l.diff(c.ID, rd.Properties, in, rd.Inputs).DetailedDiff["reachabilityEnabled"].Kind)
	requireNoChanges(t, l.diff(c.ID, c.Properties, without(in, "reachabilityEnabled"), in))

	next := with(in, map[string]property.Value{"reachabilityEnabled": bv(false)})
	l.f.resetCalls()
	u := l.update(c.ID, c.Properties, next, in)
	require.Equal(t, "false", l.f.callsMatching("POST", "/Configuration")[0].Form.Get("ReachabilityEnabled"))
	requireNoChanges(t, l.diff(c.ID, u.Properties, next, next))
	require.Equal(t, pgp.UpdateReplace, l.diff(c.ID, u.Properties, with(next, map[string]property.Value{"chatServiceSid": sv("IS" + strings.Repeat("1", 32))}), next).DetailedDiff["chatServiceSid"].Kind)

	// Import adopts what Twilio reports.
	imp := l.read(svc, empty(), empty())
	require.Equal(t, svc, getS(t, imp.Inputs, "chatServiceSid"))
	require.Equal(t, roleSid, getS(t, imp.Inputs, "defaultChatServiceRoleSid"))
	l.requireImportFails("IS" + strings.Repeat("0", 32))

	// Destroying forgets it: no request, the settings stay.
	l.f.resetCalls()
	l.del(c.ID, u.Properties)
	require.Empty(t, l.f.callsMatching("DELETE", "/"))
	require.Equal(t, roleSid, l.f.get("cvone", testAccount, "svccfg/"+svc)["default_chat_service_role_sid"])

	// With nothing to set it only checks that the service exists.
	l.f.resetCalls()
	l.create(pv(map[string]property.Value{"chatServiceSid": sv(svc)}))
	require.Empty(t, l.f.callsMatching("POST", "/Configuration"))
	_, err := l.s.Create(pgp.CreateRequest{Urn: l.urn, Properties: pv(map[string]property.Value{"chatServiceSid": sv("IS" + strings.Repeat("9", 32))})})
	require.Error(t, err)

	bad := l.check(pv(map[string]property.Value{"chatServiceSid": sv("nope"), "defaultConversationRoleSid": sv("nope")}))
	require.Len(t, bad.Failures, 2)
}

func TestConversationsRoleLC_Lifecycle(t *testing.T) {
	l := newTL(t, "ConversationsRole", "role")
	for _, scoped := range []bool{false, true} {
		m := map[string]property.Value{"friendlyName": sv("agent"), "roleType": sv("conversation"), "permissions": strs("sendMessage", "editOwnMessage")}
		wantID, scope := "", ""
		if scoped {
			scope = cvSvc(l)
			m["chatServiceSid"] = sv(scope)
		}
		in := pv(m)
		c := l.create(in)
		sid := getS(t, c.Properties, "sid")
		require.True(t, strings.HasPrefix(sid, "RL"))
		wantID = sid
		if scoped {
			wantID = scope + "/" + sid
		}
		require.Equal(t, wantID, c.ID)
		form := l.f.callsMatching("POST", "/Roles")[len(l.f.callsMatching("POST", "/Roles"))-1].Form
		require.Equal(t, []string{"sendMessage", "editOwnMessage"}, form["Permission"])
		require.Equal(t, "conversation", form.Get("Type"))
		requireNoChanges(t, l.diff(c.ID, c.Properties, in, in))
		rd := l.read(c.ID, c.Properties, in)
		requireNoChanges(t, l.diff(c.ID, rd.Properties, in, rd.Inputs))

		// The order of the permissions is not a change, nor drift; a different set is.
		reordered := with(in, map[string]property.Value{"permissions": strs("editOwnMessage", "sendMessage")})
		requireNoChanges(t, l.diff(c.ID, c.Properties, reordered, in))
		l.f.mutate("cv", testAccount, "Roles/"+scope+"/"+sid, func(o map[string]any) { o["permissions"] = []any{"editOwnMessage", "sendMessage"} })
		rd = l.read(c.ID, c.Properties, in)
		requireNoChanges(t, l.diff(c.ID, rd.Properties, in, rd.Inputs))
		l.f.mutate("cv", testAccount, "Roles/"+scope+"/"+sid, func(o map[string]any) { o["permissions"] = []any{"sendMessage"} })
		rd = l.read(c.ID, c.Properties, in)
		require.Equal(t, pgp.Update, l.diff(c.ID, rd.Properties, in, rd.Inputs).DetailedDiff["permissions"].Kind)

		next := with(in, map[string]property.Value{"permissions": strs("sendMessage", "deleteOwnMessage")})
		require.Equal(t, pgp.Update, l.diff(c.ID, c.Properties, next, in).DetailedDiff["permissions"].Kind)
		l.f.resetCalls()
		u := l.update(c.ID, c.Properties, next, in)
		require.Equal(t, []string{"sendMessage", "deleteOwnMessage"}, l.f.callsMatching("POST", "/Roles/")[0].Form["Permission"])
		require.Empty(t, l.f.callsMatching("POST", "/Roles/")[0].Form["FriendlyName"])
		requireNoChanges(t, l.diff(c.ID, u.Properties, next, next))
		for _, k := range []string{"friendlyName", "roleType"} {
			d := l.diff(c.ID, u.Properties, with(next, map[string]property.Value{k: sv("other")}), next)
			require.Equal(t, pgp.UpdateReplace, d.DetailedDiff[k].Kind, k)
		}

		imp := l.read(c.ID, empty(), empty())
		require.Equal(t, "agent", getS(t, imp.Inputs, "friendlyName"))
		require.Equal(t, scoped, hasProp(imp.Inputs, "chatServiceSid"))
		requireNoChanges(t, l.diff(c.ID, imp.Properties, next, imp.Inputs))

		l.f.remove("cv", testAccount, "Roles/"+scope+"/"+sid)
		require.Empty(t, l.read(c.ID, u.Properties, next).ID)
		l.del(c.ID, u.Properties)
	}
	l.requireImportFails("RL" + strings.Repeat("0", 32))
	_, err := l.s.Read(pgp.ReadRequest{ID: "a/b/c", Urn: l.urn, Properties: empty(), Inputs: empty()})
	require.ErrorContains(t, err, "invalid Conversations role ID")
	bad := l.check(pv(map[string]property.Value{"friendlyName": sv("x"), "roleType": sv("user"), "permissions": strs()}))
	require.Len(t, bad.Failures, 2)
}

func TestConversationsCredentialLC_Lifecycle(t *testing.T) {
	l := newTL(t, "ConversationsCredential", "cred")
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
	l.f.mutate("cv", testAccount, "Credentials//"+c.ID, func(o map[string]any) { o["sandbox"] = "false" })
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
	require.Equal(t, "ios 2", l.f.get("cv", testAccount, "Credentials//"+c.ID)["friendly_name"])
	requireNoChanges(t, l.diff(c.ID, u.Properties, next, next))
	require.Equal(t, pgp.UpdateReplace, l.diff(c.ID, u.Properties, with(next, map[string]property.Value{"credentialType": sv("gcm")}), next).DetailedDiff["credentialType"].Kind)

	imp := l.read(c.ID, empty(), empty())
	require.Equal(t, "apn", getS(t, imp.Inputs, "credentialType"))
	require.Equal(t, "ios 2", getS(t, imp.Inputs, "friendlyName"))
	require.False(t, hasProp(imp.Inputs, "privateKey"))

	l.f.remove("cv", testAccount, "Credentials//"+c.ID)
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

func TestConversationsAddressConfigurationLC_Lifecycle(t *testing.T) {
	l := newTL(t, "ConversationsAddressConfiguration", "addr")
	svc := cvSvc(l)
	auto := func(m map[string]property.Value) property.Value { return property.New(property.NewMap(m)) }
	in := pv(map[string]property.Value{"addressType": sv("whatsapp"), "address": sv("whatsapp:+15551234567"), "friendlyName": sv("support line"),
		"autoCreation": auto(map[string]property.Value{"enabled": bv(true), "type": sv("webhook"), "webhookUrl": sv("https://example.com/hook"),
			"webhookMethod": sv("post"), "webhookFilters": strs("onMessageAdded", "onConversationAdded"), "conversationServiceSid": sv(svc)})})
	c := l.create(in)
	require.True(t, strings.HasPrefix(c.ID, "IG"), c.ID)
	form := l.f.callsMatching("POST", "/Configuration/Addresses")[0].Form
	require.Equal(t, "whatsapp", form.Get("Type"))
	require.Equal(t, "true", form.Get("AutoCreation.Enabled"))
	require.Equal(t, "POST", form.Get("AutoCreation.WebhookMethod"), "the method is sent in upper case")
	require.Equal(t, []string{"onMessageAdded", "onConversationAdded"}, form["AutoCreation.WebhookFilters"])
	require.Empty(t, form["AddressCountry"])
	checked := l.checked(in)
	requireNoChanges(t, l.diff(c.ID, c.Properties, checked, checked))
	rd := l.read(c.ID, c.Properties, checked)
	requireNoChanges(t, l.diff(c.ID, rd.Properties, checked, rd.Inputs))

	// Drift in a nested setting shows; an unset nested setting Twilio fills in does not.
	l.f.mutate("cv", testAccount, "Configuration/Addresses//"+c.ID, func(o map[string]any) {
		o["auto_creation"].(map[string]any)["webhook_url"] = "https://example.com/other"
		o["auto_creation"].(map[string]any)["studio_retry_count"] = float64(3)
	})
	rd = l.read(c.ID, c.Properties, checked)
	require.Equal(t, pgp.Update, l.diff(c.ID, rd.Properties, checked, rd.Inputs).DetailedDiff["autoCreation"].Kind)
	require.False(t, hasProp(rd.Inputs.Get("autoCreation").AsMap(), "studioRetryCount"))

	// Name and auto-creation update in place with one POST that leaves the address alone.
	next := with(checked, map[string]property.Value{"friendlyName": sv("support"),
		"autoCreation": auto(map[string]property.Value{"enabled": bv(false)})})
	require.Equal(t, pgp.Update, l.diff(c.ID, c.Properties, next, checked).DetailedDiff["friendlyName"].Kind)
	l.f.resetCalls()
	u := l.update(c.ID, c.Properties, next, checked)
	uf := l.f.callsMatching("POST", "/Configuration/Addresses/")[0].Form
	require.Equal(t, "false", uf.Get("AutoCreation.Enabled"))
	require.Empty(t, uf["Address"])
	require.Empty(t, uf["Type"])
	requireNoChanges(t, l.diff(c.ID, u.Properties, next, next))
	for k, v := range map[string]string{"address": "whatsapp:+15557654321", "addressType": "sms", "addressCountry": "US"} {
		d := l.diff(c.ID, u.Properties, with(next, map[string]property.Value{k: sv(v)}), next)
		require.Equal(t, pgp.UpdateReplace, d.DetailedDiff[k].Kind, k)
	}
	// Removing autoCreation leaves the setting alone.
	requireNoChanges(t, l.diff(c.ID, u.Properties, without(next, "autoCreation"), next))

	// Import adopts the auto-creation only when it is enabled.
	imp := l.read(c.ID, empty(), empty())
	require.Equal(t, "whatsapp", getS(t, imp.Inputs, "addressType"))
	require.False(t, hasProp(imp.Inputs, "autoCreation"))
	l.f.mutate("cv", testAccount, "Configuration/Addresses//"+c.ID, func(o map[string]any) {
		o["auto_creation"].(map[string]any)["enabled"] = true
		o["auto_creation"].(map[string]any)["type"] = "studio"
		o["auto_creation"].(map[string]any)["studio_flow_sid"] = "FW" + strings.Repeat("1", 32)
	})
	imp = l.read(c.ID, empty(), empty())
	ac := imp.Inputs.Get("autoCreation").AsMap()
	require.Equal(t, "studio", getS(t, ac, "type"))
	require.Equal(t, "FW"+strings.Repeat("1", 32), getS(t, ac, "studioFlowSid"))
	require.Equal(t, "https://example.com/other", getS(t, ac, "webhookUrl"), "a value Twilio still holds is adopted on import")
	require.Equal(t, "POST", getS(t, ac, "webhookMethod"))

	l.f.remove("cv", testAccount, "Configuration/Addresses//"+c.ID)
	require.Empty(t, l.read(c.ID, u.Properties, next).ID)
	l.del(c.ID, u.Properties)
	l.requireImportFails("IG" + strings.Repeat("0", 32))

	bad := l.check(pv(map[string]property.Value{"addressType": sv("fax"), "address": sv("x"),
		"autoCreation": auto(map[string]property.Value{"enabled": bv(true), "type": sv("magic")})}))
	require.Len(t, bad.Failures, 2)
}

func TestConversationsWebhookConfigurationLC_Lifecycle(t *testing.T) {
	l := newTL(t, "ConversationsWebhookConfiguration", "wh")
	svc := cvSvc(l)

	// Account level.
	in := pv(map[string]property.Value{"method": sv("get"), "filters": strs("onMessageAdded", "onMessageUpdated"),
		"postWebhookUrl": sv("https://example.com/post"), "target": sv("flex")})
	c := l.create(in)
	require.Equal(t, "account", c.ID)
	form := l.f.callsMatching("POST", "/conversations/v1/Configuration/Webhooks")[0].Form
	require.Equal(t, "GET", form.Get("Method"))
	require.Equal(t, []string{"onMessageAdded", "onMessageUpdated"}, form["Filters"])
	require.Equal(t, "flex", form.Get("Target"))
	checked := l.checked(in)
	requireNoChanges(t, l.diff(c.ID, c.Properties, checked, checked))
	rd := l.read(c.ID, c.Properties, checked)
	requireNoChanges(t, l.diff(c.ID, rd.Properties, checked, rd.Inputs))
	require.False(t, hasProp(rd.Inputs, "preWebhookUrl"))

	l.f.mutate("cvone", testAccount, "acctwh/", func(o map[string]any) { o["filters"] = []any{"onMessageAdded"} })
	rd = l.read(c.ID, c.Properties, checked)
	require.Equal(t, pgp.Update, l.diff(c.ID, rd.Properties, checked, rd.Inputs).DetailedDiff["filters"].Kind)
	// The order of the filters is not drift.
	l.f.mutate("cvone", testAccount, "acctwh/", func(o map[string]any) { o["filters"] = []any{"onMessageUpdated", "onMessageAdded"} })
	rd = l.read(c.ID, c.Properties, checked)
	requireNoChanges(t, l.diff(c.ID, rd.Properties, checked, rd.Inputs))

	next := with(checked, map[string]property.Value{"preWebhookUrl": sv("https://example.com/pre")})
	require.Equal(t, pgp.Update, l.diff(c.ID, c.Properties, next, checked).DetailedDiff["preWebhookUrl"].Kind)
	l.f.resetCalls()
	u := l.update(c.ID, c.Properties, next, checked)
	require.Equal(t, "https://example.com/pre", l.f.callsMatching("POST", "/Configuration/Webhooks")[0].Form.Get("PreWebhookUrl"))
	requireNoChanges(t, l.diff(c.ID, u.Properties, next, next))
	requireNoChanges(t, l.diff(c.ID, u.Properties, without(next, "filters"), next))

	imp := l.read("account", empty(), empty())
	require.False(t, hasProp(imp.Inputs, "chatServiceSid"))
	require.Equal(t, "flex", getS(t, imp.Inputs, "target"))
	require.Equal(t, "https://example.com/pre", getS(t, imp.Inputs, "preWebhookUrl"))

	// Destroying forgets it.
	l.f.resetCalls()
	l.del(c.ID, u.Properties)
	require.Empty(t, l.f.callsMatching("DELETE", "/"))

	// Service level.
	sin := pv(map[string]property.Value{"chatServiceSid": sv(svc), "method": sv("POST"), "preWebhookUrl": sv("https://example.com/pre")})
	sc := l.create(sin)
	require.Equal(t, svc, sc.ID)
	require.Equal(t, "https://example.com/pre", l.f.callsMatching("POST", "/Services/"+svc+"/Configuration/Webhooks")[0].Form.Get("PreWebhookUrl"))
	requireNoChanges(t, l.diff(sc.ID, sc.Properties, sin, sin))
	require.Equal(t, pgp.UpdateReplace, l.diff(sc.ID, sc.Properties, c0(sin), sin).DetailedDiff["chatServiceSid"].Kind)
	simp := l.read(svc, empty(), empty())
	require.Equal(t, svc, getS(t, simp.Inputs, "chatServiceSid"))
	require.False(t, hasProp(simp.Inputs, "target"))
	l.requireImportFails("IS" + strings.Repeat("0", 32))

	require.Len(t, l.check(pv(map[string]property.Value{"chatServiceSid": sv(svc), "target": sv("flex")})).Failures, 1)
	require.Len(t, l.check(pv(map[string]property.Value{"method": sv("PUT"), "target": sv("sms")})).Failures, 2)
}

// c0 is the same inputs for the account-level configuration.
func c0(m property.Map) property.Map { return without(m, "chatServiceSid") }

func TestLookup_Conversations(t *testing.T) {
	l := newTL(t, "ConversationsService", "svc")
	svc := cvSvc(l)
	r, err := invoke(t, l, "getConversationsService", map[string]property.Value{"serviceSid": sv(svc)})
	require.NoError(t, err)
	require.Equal(t, "support", getS(t, r, "friendlyName"))
	_, err = invoke(t, l, "getConversationsService", map[string]property.Value{"serviceSid": sv("ISmissing")})
	require.ErrorContains(t, err, "was not found")
	_, err = invoke(t, l, "getConversationsService", map[string]property.Value{"serviceSid": sv("")})
	require.ErrorContains(t, err, "must be set")

	cfg, err := invoke(t, l, "getConversationsServiceConfiguration", map[string]property.Value{"chatServiceSid": sv(svc)})
	require.NoError(t, err)
	require.False(t, cfg.Get("reachabilityEnabled").AsBool())
	require.False(t, hasProp(cfg, "defaultChatServiceRoleSid"))
	_, err = invoke(t, l, "getConversationsServiceConfiguration", map[string]property.Value{"chatServiceSid": sv("ISmissing")})
	require.ErrorContains(t, err, "was not found")

	role := l.other("ConversationsRole", "role").create(pv(map[string]property.Value{"friendlyName": sv("agent"), "roleType": sv("conversation"),
		"permissions": strs("sendMessage")}))
	r, err = invoke(t, l, "getConversationsRole", map[string]property.Value{"sid": sv(role.ID)})
	require.NoError(t, err)
	require.Equal(t, "conversation", getS(t, r, "roleType"))
	require.Equal(t, 1, r.Get("permissions").AsArray().Len())
	_, err = invoke(t, l, "getConversationsRole", map[string]property.Value{"sid": sv(role.ID), "chatServiceSid": sv(svc)})
	require.ErrorContains(t, err, "was not found", "an account-level role is not in a service")
	_, err = invoke(t, l, "getConversationsRole", map[string]property.Value{"sid": sv("")})
	require.ErrorContains(t, err, "must be set")

	cred := l.other("ConversationsCredential", "cred").create(pv(map[string]property.Value{"credentialType": sv("gcm"), "apiKey": sv("AIza"), "friendlyName": sv("android")}))
	r, err = invoke(t, l, "getConversationsCredential", map[string]property.Value{"sid": sv(cred.ID)})
	require.NoError(t, err)
	require.Equal(t, "gcm", getS(t, r, "credentialType"))
	require.Equal(t, "android", getS(t, r, "friendlyName"))
	require.False(t, hasProp(r, "apiKey"))
	_, err = invoke(t, l, "getConversationsCredential", map[string]property.Value{"sid": sv("CRmissing")})
	require.ErrorContains(t, err, "was not found")

	addr := l.other("ConversationsAddressConfiguration", "addr").create(pv(map[string]property.Value{"addressType": sv("sms"), "address": sv("+15551234567"),
		"autoCreation": property.New(property.NewMap(map[string]property.Value{"enabled": bv(true), "type": sv("default")}))}))
	r, err = invoke(t, l, "getConversationsAddressConfiguration", map[string]property.Value{"sid": sv(addr.ID)})
	require.NoError(t, err)
	require.Equal(t, "+15551234567", getS(t, r, "address"))
	require.Equal(t, "default", getS(t, r.Get("autoCreation").AsMap(), "type"))
	_, err = invoke(t, l, "getConversationsAddressConfiguration", map[string]property.Value{"sid": sv("IGmissing")})
	require.ErrorContains(t, err, "was not found")

	wh, err := invoke(t, l, "getConversationsWebhookConfiguration", map[string]property.Value{})
	require.NoError(t, err)
	require.Equal(t, "webhook", getS(t, wh, "target"))
	swh, err := invoke(t, l, "getConversationsWebhookConfiguration", map[string]property.Value{"chatServiceSid": sv(svc)})
	require.NoError(t, err)
	require.False(t, hasProp(swh, "target"))
	_, err = invoke(t, l, "getConversationsWebhookConfiguration", map[string]property.Value{"chatServiceSid": sv("ISmissing")})
	require.ErrorContains(t, err, "was not found")
}
