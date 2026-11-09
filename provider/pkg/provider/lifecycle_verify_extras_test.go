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
	verifyMsgSid  = "MG" + "0123456789abcdef0123456789abcdef"
	verifyMsgSid2 = "MG" + "fedcba9876543210fedcba9876543210"
)

func verifyStrList(vs ...string) property.Value {
	out := make([]property.Value, len(vs))
	for i, s := range vs {
		out[i] = sv(s)
	}
	return property.New(property.NewArray(out))
}

// verifySvc creates a Verify service on the fake and returns its SID.
func verifySvc(l *tl) string {
	return l.other("VerifyService", "svc").create(verifyInputs()).ID
}

func rateLimitInputs(svc string) property.Map {
	return pv(map[string]property.Value{"serviceSid": sv(svc), "uniqueName": sv("signup-by-ip"), "description": sv("limits signups per IP")})
}

func TestVerifyRateLimitLC_Lifecycle(t *testing.T) {
	l := newTL(t, "VerifyRateLimit", "rl")
	svc := verifySvc(l)
	in := rateLimitInputs(svc)
	c := l.create(in)
	rlSid := getS(t, c.Properties, "sid")
	require.True(t, strings.HasPrefix(rlSid, "RK"), rlSid)
	require.Equal(t, svc+"/"+rlSid, c.ID)
	require.Equal(t, "2026-11-06T00:00:00Z", getS(t, c.Properties, "dateCreated"))
	form := l.f.callsMatching("POST", "/RateLimits")[0].Form
	require.Equal(t, "signup-by-ip", form.Get("UniqueName"))
	require.Equal(t, "limits signups per IP", form.Get("Description"))
	requireNoChanges(t, l.diff(c.ID, c.Properties, in, in))

	// A refresh changes nothing; an out-of-band description change is drift.
	rd := l.read(c.ID, c.Properties, in)
	requireNoChanges(t, l.diff(c.ID, rd.Properties, in, rd.Inputs))
	l.f.mutate("vrl", testAccount, svc+"/"+rlSid, func(o map[string]any) { o["description"] = "edited" })
	rd = l.read(c.ID, c.Properties, in)
	require.Equal(t, "edited", getS(t, rd.Inputs, "description"))
	d := l.diff(c.ID, rd.Properties, in, rd.Inputs)
	require.Equal(t, pgp.Update, d.DetailedDiff["description"].Kind)
	require.Len(t, d.DetailedDiff, 1)

	// The description updates in place; the unique name and the service replace.
	next := with(in, map[string]property.Value{"description": sv("new text")})
	d = l.diff(c.ID, c.Properties, next, in)
	require.Equal(t, pgp.Update, d.DetailedDiff["description"].Kind)
	l.f.resetCalls()
	u := l.update(c.ID, c.Properties, next, in)
	require.Equal(t, "new text", l.f.get("vrl", testAccount, svc+"/"+rlSid)["description"])
	require.Equal(t, "2026-11-07T00:00:00Z", getS(t, u.Properties, "dateUpdated"))
	requireNoChanges(t, l.diff(c.ID, u.Properties, next, next))
	d = l.diff(c.ID, u.Properties, with(next, map[string]property.Value{"uniqueName": sv("other")}), next)
	require.Equal(t, pgp.UpdateReplace, d.DetailedDiff["uniqueName"].Kind)
	d = l.diff(c.ID, u.Properties, with(next, map[string]property.Value{"serviceSid": sv("VA" + strings.Repeat("0", 32))}), next)
	require.Equal(t, pgp.UpdateReplace, d.DetailedDiff["serviceSid"].Kind)

	// Removing the description clears it.
	gone := without(next, "description")
	d = l.diff(c.ID, u.Properties, gone, next)
	require.Equal(t, pgp.Update, d.DetailedDiff["description"].Kind)
	l.f.resetCalls()
	u = l.update(c.ID, u.Properties, gone, next)
	require.Equal(t, []string{""}, l.f.callsMatching("POST", "/RateLimits/"+rlSid)[0].Form["Description"])
	require.Equal(t, "", l.f.get("vrl", testAccount, svc+"/"+rlSid)["description"])
	// An empty description and none are the same thing.
	requireNoChanges(t, l.diff(c.ID, u.Properties, with(gone, map[string]property.Value{"description": sv("")}), gone))

	// Gone out of band: a refresh drops it, and deleting twice is fine.
	l.f.remove("vrl", testAccount, svc+"/"+rlSid)
	require.Empty(t, l.read(c.ID, u.Properties, gone).ID)
	l.del(c.ID, u.Properties)
	c = l.create(in)
	l.del(c.ID, c.Properties)
	require.Nil(t, l.f.get("vrl", testAccount, c.ID))
}

func TestVerifyRateLimitLC_NoDescriptionAndImport(t *testing.T) {
	l := newTL(t, "VerifyRateLimit", "rl")
	svc := verifySvc(l)
	minimal := pv(map[string]property.Value{"serviceSid": sv(svc), "uniqueName": sv("bare")})
	c := l.create(minimal)
	require.Empty(t, l.f.callsMatching("POST", "/RateLimits")[0].Form["Description"], "no description is sent when none is set")
	requireNoChanges(t, l.diff(c.ID, c.Properties, minimal, minimal))

	// Import adopts the service, the unique name and no description.
	rd := l.read(c.ID, empty(), empty())
	require.Equal(t, c.ID, rd.ID)
	require.Equal(t, svc, getS(t, rd.Inputs, "serviceSid"))
	require.Equal(t, "bare", getS(t, rd.Inputs, "uniqueName"))
	require.False(t, hasProp(rd.Inputs, "description"))
	requireNoChanges(t, l.diff(c.ID, rd.Properties, minimal, rd.Inputs))

	// A description set out of band is adopted by an import.
	l.f.mutate("vrl", testAccount, c.ID, func(o map[string]any) { o["description"] = "set elsewhere" })
	rd = l.read(c.ID, empty(), empty())
	require.Equal(t, "set elsewhere", getS(t, rd.Inputs, "description"))
	requireNoChanges(t, l.diff(c.ID, rd.Properties, rd.Inputs, rd.Inputs))

	_, err := l.s.Read(pgp.ReadRequest{ID: "nonsense", Urn: l.urn, Properties: empty(), Inputs: empty()})
	require.ErrorContains(t, err, "<serviceSid>/<sid>")
}

func TestVerifyRateLimitLC_CheckAndPreview(t *testing.T) {
	l := newTL(t, "VerifyRateLimit", "rl")
	for name, in := range map[string]property.Map{
		"serviceSid": pv(map[string]property.Value{"serviceSid": sv("VAshort"), "uniqueName": sv("x")}),
		"uniqueName": pv(map[string]property.Value{"serviceSid": sv("VA" + strings.Repeat("a", 32)), "uniqueName": sv("")}),
	} {
		chk := l.check(in)
		require.Len(t, chk.Failures, 1, name)
		require.Equal(t, name, chk.Failures[0].Property)
	}
	require.Empty(t, l.check(pv(map[string]property.Value{"serviceSid": computed(), "uniqueName": computed()})).Failures)
	// A preview Create has no ID and calls nothing.
	in := l.checked(rateLimitInputs("VA" + strings.Repeat("a", 32)))
	c, err := l.s.Create(pgp.CreateRequest{Urn: l.urn, Properties: in, DryRun: true})
	require.NoError(t, err)
	require.Empty(t, c.ID)
	require.Zero(t, l.f.callCount())
}

func bucketInputs(svc, rl string) property.Map {
	return pv(map[string]property.Value{"serviceSid": sv(svc), "rateLimitSid": sv(rl), "max": nv(5), "interval": nv(60)})
}

func TestVerifyRateLimitBucketLC_Lifecycle(t *testing.T) {
	l := newTL(t, "VerifyRateLimitBucket", "b")
	svc := verifySvc(l)
	rl := l.other("VerifyRateLimit", "rl").create(rateLimitInputs(svc))
	rlSid := getS(t, rl.Properties, "sid")
	in := bucketInputs(svc, rlSid)
	c := l.create(in)
	sid := getS(t, c.Properties, "sid")
	require.True(t, strings.HasPrefix(sid, "BL"), sid)
	require.Equal(t, svc+"/"+rlSid+"/"+sid, c.ID)
	form := l.f.callsMatching("POST", "/Buckets")[0].Form
	require.Equal(t, "5", form.Get("Max"))
	require.Equal(t, "60", form.Get("Interval"))
	requireNoChanges(t, l.diff(c.ID, c.Properties, in, in))

	// Refresh and drift.
	rd := l.read(c.ID, c.Properties, in)
	requireNoChanges(t, l.diff(c.ID, rd.Properties, in, rd.Inputs))
	l.f.mutate("vbk", testAccount, svc+"/"+rlSid+"/"+sid, func(o map[string]any) { o["max"] = float64(9) })
	rd = l.read(c.ID, c.Properties, in)
	require.Equal(t, float64(9), rd.Inputs.Get("max").AsNumber())
	d := l.diff(c.ID, rd.Properties, in, rd.Inputs)
	require.Equal(t, pgp.Update, d.DetailedDiff["max"].Kind)
	require.Len(t, d.DetailedDiff, 1)

	// Max and interval update in place; the parents replace.
	next := with(in, map[string]property.Value{"max": nv(10), "interval": nv(120)})
	d = l.diff(c.ID, c.Properties, next, in)
	require.Equal(t, pgp.Update, d.DetailedDiff["max"].Kind)
	require.Equal(t, pgp.Update, d.DetailedDiff["interval"].Kind)
	require.Len(t, d.DetailedDiff, 2)
	l.f.resetCalls()
	u := l.update(c.ID, c.Properties, next, in)
	o := l.f.get("vbk", testAccount, svc+"/"+rlSid+"/"+sid)
	require.Equal(t, float64(10), o["max"])
	require.Equal(t, float64(120), o["interval"])
	require.Equal(t, "2026-11-07T00:00:00Z", getS(t, u.Properties, "dateUpdated"))
	requireNoChanges(t, l.diff(c.ID, u.Properties, next, next))
	d = l.diff(c.ID, u.Properties, with(next, map[string]property.Value{"rateLimitSid": sv("RK" + strings.Repeat("1", 32))}), next)
	require.Equal(t, pgp.UpdateReplace, d.DetailedDiff["rateLimitSid"].Kind)

	// Import adopts everything from the ID and the response.
	imp := l.read(c.ID, empty(), empty())
	require.Equal(t, svc, getS(t, imp.Inputs, "serviceSid"))
	require.Equal(t, rlSid, getS(t, imp.Inputs, "rateLimitSid"))
	require.Equal(t, float64(10), imp.Inputs.Get("max").AsNumber())
	requireNoChanges(t, l.diff(c.ID, imp.Properties, next, imp.Inputs))

	// Gone out of band, including when the parent rate limit is deleted.
	l.other("VerifyRateLimit", "rl").del(rl.ID, rl.Properties)
	require.Empty(t, l.read(c.ID, u.Properties, next).ID)
	l.del(c.ID, u.Properties)
	_, err := l.s.Read(pgp.ReadRequest{ID: svc + "/" + rlSid, Urn: l.urn, Properties: empty(), Inputs: empty()})
	require.ErrorContains(t, err, "<serviceSid>/<rateLimitSid>/<sid>")
}

func TestVerifyRateLimitBucketLC_DeletedWithParentAndCheck(t *testing.T) {
	l := newTL(t, "VerifyRateLimitBucket", "b")
	chk := l.check(pv(map[string]property.Value{"serviceSid": sv("nope"), "rateLimitSid": sv("RKnope"), "max": nv(1), "interval": nv(1)}))
	require.Len(t, chk.Failures, 2)
	require.Empty(t, l.check(pv(map[string]property.Value{"serviceSid": computed(), "rateLimitSid": computed(), "max": computed(), "interval": computed()})).Failures)
	svc := verifySvc(l)
	_, err := l.s.Create(pgp.CreateRequest{Urn: l.urn, Properties: l.checked(bucketInputs(svc, "RK"+strings.Repeat("2", 32)))})
	require.ErrorContains(t, err, "create bucket")
	c, err := l.s.Create(pgp.CreateRequest{Urn: l.urn, Properties: l.checked(bucketInputs(svc, "RK"+strings.Repeat("2", 32))), DryRun: true})
	require.NoError(t, err)
	require.Empty(t, c.ID)
}

func TestVerifyRateLimitBucketLC_CheckBounds(t *testing.T) {
	l := newTL(t, "VerifyRateLimitBucket", "b")
	svc, rl := "VA"+strings.Repeat("a", 32), "RK"+strings.Repeat("b", 32)
	for _, tc := range []struct {
		max, interval int
		bad           string
	}{{0, 60, "max"}, {-1, 60, "max"}, {5, 0, "interval"}, {5, -30, "interval"}} {
		chk := l.check(with(bucketInputs(svc, rl), map[string]property.Value{"max": nv(tc.max), "interval": nv(tc.interval)}))
		require.Len(t, chk.Failures, 1, "%+v", tc)
		require.Equal(t, tc.bad, chk.Failures[0].Property)
	}
	require.Empty(t, l.check(bucketInputs(svc, rl)).Failures)
	require.Empty(t, l.check(with(bucketInputs(svc, rl), map[string]property.Value{"max": nv(1), "interval": nv(1)})).Failures)
	require.Empty(t, l.check(with(bucketInputs(svc, rl), map[string]property.Value{"max": computed(), "interval": computed()})).Failures, "an unknown value is not a failure")
}

func configInputs(svc, country, msg string) property.Map {
	return pv(map[string]property.Value{"serviceSid": sv(svc), "country": sv(country), "messagingServiceSid": sv(msg)})
}

func TestVerifyMessagingConfigurationLC_Lifecycle(t *testing.T) {
	l := newTL(t, "VerifyMessagingConfiguration", "mc")
	svc := verifySvc(l)
	in := configInputs(svc, "DE", verifyMsgSid)
	c := l.create(in)
	require.Equal(t, svc+"/DE", c.ID)
	form := l.f.callsMatching("POST", "/MessagingConfigurations")[0].Form
	require.Equal(t, "DE", form.Get("Country"))
	require.Equal(t, verifyMsgSid, form.Get("MessagingServiceSid"))
	requireNoChanges(t, l.diff(c.ID, c.Properties, in, in))

	// Refresh and drift.
	rd := l.read(c.ID, c.Properties, in)
	requireNoChanges(t, l.diff(c.ID, rd.Properties, in, rd.Inputs))
	l.f.mutate("vmc", testAccount, svc+"/DE", func(o map[string]any) { o["messaging_service_sid"] = verifyMsgSid2 })
	rd = l.read(c.ID, c.Properties, in)
	require.Equal(t, verifyMsgSid2, getS(t, rd.Inputs, "messagingServiceSid"))
	d := l.diff(c.ID, rd.Properties, in, rd.Inputs)
	require.Equal(t, pgp.Update, d.DetailedDiff["messagingServiceSid"].Kind)
	require.Len(t, d.DetailedDiff, 1)

	// The Messaging Service updates in place; the country and the service replace.
	next := with(in, map[string]property.Value{"messagingServiceSid": sv(verifyMsgSid2)})
	d = l.diff(c.ID, c.Properties, next, in)
	require.Equal(t, pgp.Update, d.DetailedDiff["messagingServiceSid"].Kind)
	l.f.resetCalls()
	u := l.update(c.ID, c.Properties, next, in)
	require.Equal(t, verifyMsgSid2, l.f.get("vmc", testAccount, svc+"/DE")["messaging_service_sid"])
	require.Equal(t, "2026-11-07T00:00:00Z", getS(t, u.Properties, "dateUpdated"))
	requireNoChanges(t, l.diff(c.ID, u.Properties, next, next))
	d = l.diff(c.ID, u.Properties, with(next, map[string]property.Value{"country": sv("FR")}), next)
	require.Equal(t, pgp.UpdateReplace, d.DetailedDiff["country"].Kind)

	// The global configuration is country all.
	g := l.other("VerifyMessagingConfiguration", "all").create(configInputs(svc, "all", verifyMsgSid))
	require.Equal(t, svc+"/all", g.ID)

	// Import and deleted-outside.
	imp := l.read(c.ID, empty(), empty())
	require.Equal(t, "DE", getS(t, imp.Inputs, "country"))
	require.Equal(t, svc, getS(t, imp.Inputs, "serviceSid"))
	requireNoChanges(t, l.diff(c.ID, imp.Properties, next, imp.Inputs))
	l.f.remove("vmc", testAccount, svc+"/DE")
	require.Empty(t, l.read(c.ID, u.Properties, next).ID)
	l.del(c.ID, u.Properties)
	l.other("VerifyMessagingConfiguration", "all").del(g.ID, g.Properties)
	require.Nil(t, l.f.get("vmc", testAccount, svc+"/all"))
}

// Regression for pulumi-labs/pulumi-twilio#17: Twilio answers a GET for a country without a
// configuration with HTTP 200, country "" and a made-up Messaging Service SID.
func TestVerifyMessagingConfigurationLC_FakeOKIsNotFound(t *testing.T) {
	l := newTL(t, "VerifyMessagingConfiguration", "mc")
	svc := verifySvc(l)
	in := configInputs(svc, "DE", verifyMsgSid)
	c := l.create(in)
	l.f.verifyMsgCfgFake200 = true

	// A refresh right after create is a no-op, and a lower-case recorded country still matches.
	rd := l.read(c.ID, c.Properties, in)
	require.Equal(t, c.ID, rd.ID)
	requireNoChanges(t, l.diff(c.ID, rd.Properties, in, rd.Inputs))

	// Deleted outside: the refresh drops it from state instead of adopting the made-up configuration.
	l.f.remove("vmc", testAccount, svc+"/DE")
	require.Empty(t, l.read(c.ID, c.Properties, in).ID)

	// Importing a configuration that does not exist fails.
	_, err := l.s.Read(pgp.ReadRequest{ID: svc + "/FR", Urn: l.urn, Properties: empty(), Inputs: empty()})
	require.ErrorContains(t, err, "does not exist")

	// The lookup fails too, for a missing country and after the configuration was destroyed.
	for _, country := range []string{"FR", "DE"} {
		_, err = invoke(t, l, "getVerifyMessagingConfiguration", map[string]property.Value{"serviceSid": sv(svc), "country": sv(country)})
		require.ErrorContains(t, err, "was not found", country)
	}
}

func TestVerifyMessagingConfigurationLC_CheckAndErrors(t *testing.T) {
	l := newTL(t, "VerifyMessagingConfiguration", "mc")
	for name, in := range map[string]property.Map{
		"serviceSid":          configInputs("VAbad", "DE", verifyMsgSid),
		"messagingServiceSid": configInputs("VA"+strings.Repeat("a", 32), "DE", "MGbad"),
		"country":             configInputs("VA"+strings.Repeat("a", 32), "", verifyMsgSid),
	} {
		chk := l.check(in)
		require.Len(t, chk.Failures, 1, name)
		require.Equal(t, name, chk.Failures[0].Property)
	}
}

func TestVerifyMessagingConfigurationLC_DuplicateAndPreview(t *testing.T) {
	l := newTL(t, "VerifyMessagingConfiguration", "mc")
	svc := verifySvc(l)
	in := configInputs(svc, "US", verifyMsgSid)
	l.create(in)
	_, err := l.s.Create(pgp.CreateRequest{Urn: l.urn, Properties: l.checked(in)})
	require.ErrorContains(t, err, "already exists")
	c, err := l.s.Create(pgp.CreateRequest{Urn: l.urn, Properties: l.checked(in), DryRun: true})
	require.NoError(t, err)
	require.Empty(t, c.ID)
}

func webhookInputs(svc string) property.Map {
	return pv(map[string]property.Value{"serviceSid": sv(svc), "friendlyName": sv("owl bank"),
		"eventTypes": verifyStrList("factor.verified", "factor.created"), "webhookUrl": sv("https://example.com/hook")})
}

func TestVerifyWebhookLC_Lifecycle(t *testing.T) {
	l := newTL(t, "VerifyWebhook", "wh")
	svc := verifySvc(l)
	in := webhookInputs(svc)
	c := l.create(in)
	sid := getS(t, c.Properties, "sid")
	require.True(t, strings.HasPrefix(sid, "YW"), sid)
	require.Equal(t, svc+"/"+sid, c.ID)
	require.Equal(t, "POST", getS(t, c.Properties, "webhookMethod"))
	form := l.f.callsMatching("POST", "/Webhooks")[0].Form
	require.Equal(t, []string{"factor.verified", "factor.created"}, form["EventTypes"])
	require.Equal(t, "https://example.com/hook", form.Get("WebhookUrl"))
	_, hasStatus := form["Status"]
	_, hasVersion := form["Version"]
	require.False(t, hasStatus || hasVersion, "unset status and version are not sent")
	requireNoChanges(t, l.diff(c.ID, c.Properties, in, in))

	// Twilio returns the event types sorted; a refresh keeps the program's order and shows no change.
	require.Equal(t, "factor.created", l.f.get("vwh", testAccount, svc+"/"+sid)["event_types"].([]any)[0])
	rd := l.read(c.ID, c.Properties, in)
	require.Equal(t, "factor.verified", rd.Inputs.Get("eventTypes").AsArray().Get(0).AsString())
	requireNoChanges(t, l.diff(c.ID, rd.Properties, in, rd.Inputs))
	// Reordering or restating the defaults is not a change either.
	reordered := with(in, map[string]property.Value{"eventTypes": verifyStrList("factor.created", "factor.verified"), "status": sv("enabled"), "webhookVersion": sv("v2")})
	requireNoChanges(t, l.diff(c.ID, c.Properties, reordered, in))

	// Drift on the URL and the event types.
	l.f.mutate("vwh", testAccount, svc+"/"+sid, func(o map[string]any) {
		o["webhook_url"] = "https://example.com/other"
		o["event_types"] = []any{"challenge.approved"}
	})
	rd = l.read(c.ID, c.Properties, in)
	require.Equal(t, "https://example.com/other", getS(t, rd.Inputs, "webhookUrl"))
	d := l.diff(c.ID, rd.Properties, in, rd.Inputs)
	require.Equal(t, pgp.Update, d.DetailedDiff["webhookUrl"].Kind)
	require.Equal(t, pgp.Update, d.DetailedDiff["eventTypes"].Kind)
	require.Len(t, d.DetailedDiff, 2)
	l.update(c.ID, rd.Properties, in, rd.Inputs)
	require.Equal(t, "https://example.com/hook", l.f.get("vwh", testAccount, svc+"/"+sid)["webhook_url"])

	// Update every field in place: the name, URL, events, status and version.
	next := with(in, map[string]property.Value{"friendlyName": sv("owl bank 2"), "webhookUrl": sv("https://example.com/v2"),
		"eventTypes": verifyStrList("*"), "status": sv("disabled"), "webhookVersion": sv("v1")})
	d = l.diff(c.ID, c.Properties, next, in)
	require.Len(t, d.DetailedDiff, 5)
	for _, k := range []string{"friendlyName", "webhookUrl", "eventTypes", "status", "webhookVersion"} {
		require.Equal(t, pgp.Update, d.DetailedDiff[k].Kind, k)
	}
	l.f.resetCalls()
	u := l.update(c.ID, c.Properties, next, in)
	o := l.f.get("vwh", testAccount, svc+"/"+sid)
	require.Equal(t, "disabled", o["status"])
	require.Equal(t, "v1", o["version"])
	require.Equal(t, "owl bank 2", o["friendly_name"])
	require.Equal(t, "2026-11-07T00:00:00Z", getS(t, u.Properties, "dateUpdated"))
	requireNoChanges(t, l.diff(c.ID, u.Properties, next, next))
	rd = l.read(c.ID, u.Properties, next)
	requireNoChanges(t, l.diff(c.ID, rd.Properties, next, rd.Inputs))
	l.f.mutate("vwh", testAccount, svc+"/"+sid, func(o map[string]any) { o["status"] = "enabled" })
	rd = l.read(c.ID, u.Properties, next)
	require.Equal(t, "enabled", getS(t, rd.Inputs, "status"))
	require.Equal(t, pgp.Update, l.diff(c.ID, rd.Properties, next, rd.Inputs).DetailedDiff["status"].Kind)
	l.f.mutate("vwh", testAccount, svc+"/"+sid, func(o map[string]any) { o["status"] = "disabled" })

	// Removing the status and version resets them to Twilio's defaults.
	gone := without(next, "status", "webhookVersion")
	d = l.diff(c.ID, u.Properties, gone, next)
	require.Equal(t, pgp.Update, d.DetailedDiff["status"].Kind)
	require.Equal(t, pgp.Update, d.DetailedDiff["webhookVersion"].Kind)
	l.f.resetCalls()
	u = l.update(c.ID, u.Properties, gone, next)
	o = l.f.get("vwh", testAccount, svc+"/"+sid)
	require.Equal(t, "enabled", o["status"])
	require.Equal(t, "v2", o["version"])
	requireNoChanges(t, l.diff(c.ID, u.Properties, gone, gone))

	// The service replaces.
	d = l.diff(c.ID, u.Properties, with(gone, map[string]property.Value{"serviceSid": sv("VA" + strings.Repeat("0", 32))}), gone)
	require.Equal(t, pgp.UpdateReplace, d.DetailedDiff["serviceSid"].Kind)

	// Gone out of band; deleting twice is fine.
	l.f.remove("vwh", testAccount, svc+"/"+sid)
	require.Empty(t, l.read(c.ID, u.Properties, gone).ID)
	l.del(c.ID, u.Properties)
}

func TestVerifyWebhookLC_ImportAdoptsOnlyNonDefaults(t *testing.T) {
	l := newTL(t, "VerifyWebhook", "wh")
	svc := verifySvc(l)
	seed := l.other("VerifyWebhook", "seed").create(webhookInputs(svc))

	rd := l.read(seed.ID, empty(), empty())
	got := map[string]bool{}
	for k := range rd.Inputs.AsMap() {
		got[k] = true
	}
	require.Equal(t, map[string]bool{"serviceSid": true, "friendlyName": true, "eventTypes": true, "webhookUrl": true}, got)
	// A program that spells out the defaults sees no change.
	prog := with(webhookInputs(svc), map[string]property.Value{"status": sv("enabled"), "webhookVersion": sv("v2")})
	requireNoChanges(t, l.diff(seed.ID, rd.Properties, prog, rd.Inputs))
	requireNoChanges(t, l.diff(seed.ID, rd.Properties, webhookInputs(svc), rd.Inputs))

	// Non-default values are adopted.
	l.f.mutate("vwh", testAccount, svc+"/"+getS(t, seed.Properties, "sid"), func(o map[string]any) { o["status"] = "disabled"; o["version"] = "v1" })
	rd = l.read(seed.ID, empty(), empty())
	require.Equal(t, "disabled", getS(t, rd.Inputs, "status"))
	require.Equal(t, "v1", getS(t, rd.Inputs, "webhookVersion"))
	requireNoChanges(t, l.diff(seed.ID, rd.Properties, rd.Inputs, rd.Inputs))
}

func TestVerifyWebhookLC_CheckAndPreview(t *testing.T) {
	l := newTL(t, "VerifyWebhook", "wh")
	svc := "VA" + strings.Repeat("a", 32)
	for name, over := range map[string]map[string]property.Value{
		"serviceSid":     {"serviceSid": sv("VAbad")},
		"friendlyName":   {"friendlyName": sv("")},
		"webhookUrl":     {"webhookUrl": sv("")},
		"eventTypes":     {"eventTypes": verifyStrList()},
		"status":         {"status": sv("paused")},
		"webhookVersion": {"webhookVersion": sv("v3")},
	} {
		chk := l.check(with(webhookInputs(svc), over))
		require.Len(t, chk.Failures, 1, name)
		require.Equal(t, name, chk.Failures[0].Property)
	}
	chk := l.check(with(webhookInputs(svc), map[string]property.Value{"eventTypes": verifyStrList("factor.created", "")}))
	require.Len(t, chk.Failures, 1)
	require.Empty(t, l.check(pv(map[string]property.Value{"serviceSid": computed(), "friendlyName": computed(), "eventTypes": computed(),
		"webhookUrl": computed(), "status": computed(), "webhookVersion": computed()})).Failures)
	c, err := l.s.Create(pgp.CreateRequest{Urn: l.urn, Properties: l.checked(webhookInputs(svc)), DryRun: true})
	require.NoError(t, err)
	require.Empty(t, c.ID)
	require.Zero(t, l.f.callCount())
	// The API rejects an unknown service.
	_, err = l.s.Create(pgp.CreateRequest{Urn: l.urn, Properties: l.checked(webhookInputs(svc))})
	require.ErrorContains(t, err, "create webhook")
}

func TestVerifySafeListNumberLC_Lifecycle(t *testing.T) {
	l := newTL(t, "VerifySafeListNumber", "sl")
	in := pv(map[string]property.Value{"phoneNumber": sv("+14155550100")})
	c := l.create(in)
	require.Equal(t, "+14155550100", c.ID)
	require.True(t, strings.HasPrefix(getS(t, c.Properties, "sid"), "GN"))
	require.Equal(t, "+14155550100", l.f.callsMatching("POST", "/SafeList/Numbers")[0].Form.Get("PhoneNumber"))
	requireNoChanges(t, l.diff(c.ID, c.Properties, in, in))

	// The fake insists the plus sign is escaped as %2B in the path: a read succeeds only then.
	rd := l.read(c.ID, c.Properties, in)
	require.Equal(t, c.ID, rd.ID)
	requireNoChanges(t, l.diff(c.ID, rd.Properties, in, rd.Inputs))

	// Import adopts the number from the ID.
	imp := l.read(c.ID, empty(), empty())
	require.Equal(t, "+14155550100", getS(t, imp.Inputs, "phoneNumber"))
	requireNoChanges(t, l.diff(c.ID, imp.Properties, in, imp.Inputs))

	// A different number replaces.
	d := l.diff(c.ID, c.Properties, pv(map[string]property.Value{"phoneNumber": sv("+14155550101")}), in)
	require.Equal(t, pgp.UpdateReplace, d.DetailedDiff["phoneNumber"].Kind)

	// Gone out of band; deleting twice is fine; delete removes it.
	l.f.remove("vsl", testAccount, c.ID)
	require.Empty(t, l.read(c.ID, c.Properties, in).ID)
	l.del(c.ID, c.Properties)
	c = l.create(in)
	l.del(c.ID, c.Properties)
	require.Nil(t, l.f.get("vsl", testAccount, "+14155550100"))
	require.NotEmpty(t, l.f.callsMatching("DELETE", "/SafeList/Numbers/+14155550100"))
}

func TestVerifySafeListNumberLC_CheckAndPreview(t *testing.T) {
	l := newTL(t, "VerifySafeListNumber", "sl")
	for _, bad := range []string{"", "4155550100", "+0123", "+1 415 555 0100", "tel:+14155550100"} {
		chk := l.check(pv(map[string]property.Value{"phoneNumber": sv(bad)}))
		require.Len(t, chk.Failures, 1, bad)
		require.Equal(t, "phoneNumber", chk.Failures[0].Property)
	}
	require.Empty(t, l.check(pv(map[string]property.Value{"phoneNumber": computed()})).Failures)
	c, err := l.s.Create(pgp.CreateRequest{Urn: l.urn, Properties: l.checked(pv(map[string]property.Value{"phoneNumber": sv("+14155550100")})), DryRun: true})
	require.NoError(t, err)
	require.Empty(t, c.ID)
	require.Zero(t, l.f.callCount())
}

func TestLookup_VerifyExtras(t *testing.T) {
	l := newTL(t, "VerifyRateLimit", "rl")
	svc := verifySvc(l)
	rl := l.create(rateLimitInputs(svc))
	rlSid := getS(t, rl.Properties, "sid")
	r, err := invoke(t, l, "getVerifyRateLimit", map[string]property.Value{"serviceSid": sv(svc), "sid": sv(rlSid)})
	require.NoError(t, err)
	require.Equal(t, rlSid, getS(t, r, "sid"))
	require.Equal(t, "signup-by-ip", getS(t, r, "uniqueName"))
	require.Equal(t, "limits signups per IP", getS(t, r, "description"))
	require.Equal(t, "2026-11-06T00:00:00Z", getS(t, r, "dateCreated"))
	_, err = invoke(t, l, "getVerifyRateLimit", map[string]property.Value{"serviceSid": sv(svc), "sid": sv("RKmissing")})
	require.ErrorContains(t, err, "was not found")
	_, err = invoke(t, l, "getVerifyRateLimit", map[string]property.Value{"serviceSid": sv(svc), "sid": sv("")})
	require.ErrorContains(t, err, "must be set")

	b := l.other("VerifyRateLimitBucket", "b").create(bucketInputs(svc, rlSid))
	r, err = invoke(t, l, "getVerifyRateLimitBucket", map[string]property.Value{"serviceSid": sv(svc), "rateLimitSid": sv(rlSid), "sid": sv(getS(t, b.Properties, "sid"))})
	require.NoError(t, err)
	require.Equal(t, float64(5), r.Get("max").AsNumber())
	require.Equal(t, float64(60), r.Get("interval").AsNumber())
	_, err = invoke(t, l, "getVerifyRateLimitBucket", map[string]property.Value{"serviceSid": sv(svc), "rateLimitSid": sv(rlSid), "sid": sv("BLmissing")})
	require.ErrorContains(t, err, "was not found")

	l.other("VerifyMessagingConfiguration", "mc").create(configInputs(svc, "DE", verifyMsgSid))
	r, err = invoke(t, l, "getVerifyMessagingConfiguration", map[string]property.Value{"serviceSid": sv(svc), "country": sv("DE")})
	require.NoError(t, err)
	require.Equal(t, verifyMsgSid, getS(t, r, "messagingServiceSid"))
	require.Equal(t, "DE", getS(t, r, "country"))
	_, err = invoke(t, l, "getVerifyMessagingConfiguration", map[string]property.Value{"serviceSid": sv(svc), "country": sv("FR")})
	require.ErrorContains(t, err, "was not found")

	wh := l.other("VerifyWebhook", "wh").create(webhookInputs(svc))
	r, err = invoke(t, l, "getVerifyWebhook", map[string]property.Value{"serviceSid": sv(svc), "sid": sv(getS(t, wh.Properties, "sid"))})
	require.NoError(t, err)
	require.Equal(t, "owl bank", getS(t, r, "friendlyName"))
	require.Equal(t, "enabled", getS(t, r, "status"))
	require.Equal(t, "v2", getS(t, r, "webhookVersion"))
	require.Equal(t, "POST", getS(t, r, "webhookMethod"))
	require.Equal(t, 2, r.Get("eventTypes").AsArray().Len())
	_, err = invoke(t, l, "getVerifyWebhook", map[string]property.Value{"serviceSid": sv(svc), "sid": sv("YWmissing")})
	require.ErrorContains(t, err, "was not found")

	l.other("VerifySafeListNumber", "sl").create(pv(map[string]property.Value{"phoneNumber": sv("+14155550100")}))
	r, err = invoke(t, l, "getVerifySafeListNumber", map[string]property.Value{"phoneNumber": sv("+14155550100")})
	require.NoError(t, err)
	require.Equal(t, "+14155550100", getS(t, r, "phoneNumber"))
	require.True(t, strings.HasPrefix(getS(t, r, "sid"), "GN"))
	_, err = invoke(t, l, "getVerifySafeListNumber", map[string]property.Value{"phoneNumber": sv("+14155550199")})
	require.ErrorContains(t, err, "was not found")
}

func TestLookup_VerifyTemplates(t *testing.T) {
	l := newTL(t, "VerifyService", "svc")
	r, err := invoke(t, l, "getVerifyTemplates", map[string]property.Value{})
	require.NoError(t, err)
	require.Equal(t, 0, r.Get("templates").AsArray().Len())

	l.f.addVerifyTemplate("HJ"+strings.Repeat("1", 32), "Default", "sms", "voice")
	l.f.addVerifyTemplate("HJ"+strings.Repeat("2", 32), "Bank", "sms")
	l.f.addVerifyTemplate("HJ"+strings.Repeat("3", 32), "Shop", "sms")
	l.f.templatePageSize = 2 // three templates over two pages
	r, err = invoke(t, l, "getVerifyTemplates", map[string]property.Value{})
	require.NoError(t, err)
	ts := r.Get("templates").AsArray()
	require.Equal(t, 3, ts.Len())
	first := ts.Get(0).AsMap()
	require.Equal(t, "HJ"+strings.Repeat("1", 32), getS(t, first, "sid"))
	require.Equal(t, "Default", getS(t, first, "friendlyName"))
	require.Equal(t, 2, first.Get("channels").AsArray().Len())

	r, err = invoke(t, l, "getVerifyTemplates", map[string]property.Value{"friendlyName": sv("Bank")})
	require.NoError(t, err)
	require.Equal(t, 1, r.Get("templates").AsArray().Len())
	require.Equal(t, "Bank", getS(t, r.Get("templates").AsArray().Get(0).AsMap(), "friendlyName"))
}

// The identifiers fixed at creation stay known in an update preview; what Twilio reports stays unknown.
func TestWiring_VerifyExtrasPreviewUpdate(t *testing.T) {
	l := newTL(t, "VerifyRateLimit", "rl")
	svc := verifySvc(l)
	rlIn := rateLimitInputs(svc)
	rl := l.create(rlIn)
	rlSid := getS(t, rl.Properties, "sid")
	u := previewUpdate(t, l, rl.ID, rl.Properties, rlIn, with(rlIn, map[string]property.Value{"description": sv("x")}))
	requireKnown(t, u.Properties, "sid", "dateCreated")
	requireComputed(t, u.Properties, "dateUpdated")

	bl := l.other("VerifyRateLimitBucket", "b")
	bIn := bucketInputs(svc, rlSid)
	b := bl.create(bIn)
	u = previewUpdate(t, bl, b.ID, b.Properties, bIn, with(bIn, map[string]property.Value{"max": nv(7)}))
	requireKnown(t, u.Properties, "sid", "dateCreated")
	requireComputed(t, u.Properties, "dateUpdated")

	ml := l.other("VerifyMessagingConfiguration", "mc")
	mIn := configInputs(svc, "DE", verifyMsgSid)
	m := ml.create(mIn)
	u = previewUpdate(t, ml, m.ID, m.Properties, mIn, with(mIn, map[string]property.Value{"messagingServiceSid": sv(verifyMsgSid2)}))
	requireKnown(t, u.Properties, "dateCreated", "country")
	requireComputed(t, u.Properties, "dateUpdated")

	wl := l.other("VerifyWebhook", "wh")
	wIn := webhookInputs(svc)
	w := wl.create(wIn)
	u = previewUpdate(t, wl, w.ID, w.Properties, wIn, with(wIn, map[string]property.Value{"friendlyName": sv("renamed")}))
	requireKnown(t, u.Properties, "sid", "dateCreated")
	requireComputed(t, u.Properties, "webhookMethod", "dateUpdated")
}
