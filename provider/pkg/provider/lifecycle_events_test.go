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

func webhookSinkInputs() property.Map {
	return pv(map[string]property.Value{
		"description": sv("delivery events"), "sinkType": sv("webhook"),
		"webhookDestination": sv("https://hooks.example.com/twilio"), "webhookMethod": sv("POST"), "webhookBatchEvents": bv(true),
	})
}

func typesValue(ts ...map[string]property.Value) property.Value {
	vs := make([]property.Value, 0, len(ts))
	for _, t := range ts {
		vs = append(vs, property.New(pv(t)))
	}
	return property.New(property.NewArray(vs))
}

func evType(name string) map[string]property.Value {
	return map[string]property.Value{"type": sv(name)}
}

func evTypeV(name string, v int) map[string]property.Value {
	return map[string]property.Value{"type": sv(name), "schemaVersion": nv(v)}
}

func subInputs(sink string, ts ...map[string]property.Value) property.Map {
	return pv(map[string]property.Value{"description": sv("deliveries"), "sinkSid": sv(sink), "types": typesValue(ts...)})
}

func TestEventSinkLC_WebhookLifecycle(t *testing.T) {
	l := newTL(t, "EventSink", "sink")
	in := webhookSinkInputs()
	c := l.create(in)
	sid := c.ID
	require.True(t, strings.HasPrefix(sid, "DG"), sid)
	require.Equal(t, sid, getS(t, c.Properties, "sid"))
	require.Equal(t, "initialized", getS(t, c.Properties, "status"))
	form := l.f.callsMatching("POST", "/events/v1/Sinks")[0].Form
	require.Equal(t, "webhook", form.Get("SinkType"))
	require.JSONEq(t, `{"destination":"https://hooks.example.com/twilio","method":"POST","batch_events":true}`, form.Get("SinkConfiguration"))
	requireNoChanges(t, l.diff(sid, c.Properties, in, in))

	// A refresh changes nothing; an out-of-band rename is drift on the input.
	rd := l.read(sid, c.Properties, in)
	requireNoChanges(t, l.diff(sid, rd.Properties, in, rd.Inputs))
	l.f.mutate("evsink", testAccount, sid, func(o map[string]any) { o["description"] = "renamed" })
	rd = l.read(sid, c.Properties, in)
	require.Equal(t, "renamed", getS(t, rd.Inputs, "description"))
	l.f.mutate("evsink", testAccount, sid, func(o map[string]any) { o["description"] = "delivery events" })

	// Only the description updates in place.
	next := with(in, map[string]property.Value{"description": sv("delivery events v2")})
	d := l.diff(sid, c.Properties, next, in)
	require.Equal(t, pgp.Update, d.DetailedDiff["description"].Kind)
	require.False(t, d.DeleteBeforeReplace)
	l.f.resetCalls()
	u := l.update(sid, c.Properties, next, in)
	require.Equal(t, "delivery events v2", l.f.get("evsink", testAccount, sid)["description"])
	require.Len(t, l.f.callsMatching("POST", "/events/v1/Sinks/"+sid), 1)
	require.Equal(t, "2026-11-03T00:00:00Z", getS(t, u.Properties, "dateUpdated"))
	requireNoChanges(t, l.diff(sid, u.Properties, next, next))

	// Every connection setting and the type replace the sink.
	for k, v := range map[string]property.Value{
		"webhookDestination": sv("https://hooks.example.com/other"), "webhookMethod": sv("GET"), "webhookBatchEvents": bv(false),
	} {
		d = l.diff(sid, u.Properties, with(next, map[string]property.Value{k: v}), next)
		require.Equal(t, pgp.UpdateReplace, d.DetailedDiff[k].Kind, k)
	}
	d = l.diff(sid, u.Properties, with(next, map[string]property.Value{"sinkType": sv("segment")}), next)
	require.Equal(t, pgp.UpdateReplace, d.DetailedDiff["sinkType"].Kind)

	// Gone out of band: a refresh drops it, and deleting twice is fine.
	l.f.remove("evsink", testAccount, sid)
	require.Empty(t, l.read(sid, u.Properties, next).ID)
	l.del(sid, u.Properties)
	c = l.create(in)
	l.del(c.ID, c.Properties)
	require.Nil(t, l.f.get("evsink", testAccount, c.ID))
}

func TestEventSinkLC_KinesisAndSegmentConfiguration(t *testing.T) {
	l := newTL(t, "EventSink", "sink")
	k := l.create(pv(map[string]property.Value{"description": sv("stream"), "sinkType": sv("kinesis"),
		"kinesisArn": sv("arn:aws:kinesis:us-east-1:123456789012:stream/events"), "kinesisRoleArn": sv("arn:aws:iam::123456789012:role/twilio"),
		"kinesisExternalId": sv("ext-1")}))
	require.JSONEq(t, `{"arn":"arn:aws:kinesis:us-east-1:123456789012:stream/events","role_arn":"arn:aws:iam::123456789012:role/twilio","external_id":"ext-1"}`,
		l.f.callsMatching("POST", "/events/v1/Sinks")[0].Form.Get("SinkConfiguration"))
	require.True(t, k.Properties.Get("kinesisExternalId").Secret(), "the external ID is a secret")

	l.f.resetCalls()
	in := pv(map[string]property.Value{"description": sv("cdp"), "sinkType": sv("segment"), "segmentWriteKey": property.New("wk-123").WithSecret(true)})
	s := l.create(in)
	require.JSONEq(t, `{"write_key":"wk-123"}`, l.f.callsMatching("POST", "/events/v1/Sinks")[0].Form.Get("SinkConfiguration"))
	require.True(t, s.Properties.Get("segmentWriteKey").Secret(), "the write key is a secret in the state")

	// Twilio never returns the write key: a refresh keeps the one in the inputs and plans no change.
	rd := l.read(s.ID, s.Properties, in)
	require.Equal(t, "wk-123", rd.Inputs.Get("segmentWriteKey").WithSecret(false).AsString())
	requireNoChanges(t, l.diff(s.ID, rd.Properties, in, rd.Inputs))
	d := l.diff(s.ID, rd.Properties, with(in, map[string]property.Value{"segmentWriteKey": property.New("wk-456").WithSecret(true)}), rd.Inputs)
	require.Equal(t, pgp.UpdateReplace, d.DetailedDiff["segmentWriteKey"].Kind)
}

func TestEventSinkLC_ImportPreviewAndChecks(t *testing.T) {
	l := newTL(t, "EventSink", "sink")
	in := with(webhookSinkInputs(), nil)
	c := l.create(in)

	// An import adopts the type and the connection settings; a default method is left out.
	imp := l.read(c.ID, empty(), empty())
	require.Equal(t, "webhook", getS(t, imp.Inputs, "sinkType"))
	require.Equal(t, "https://hooks.example.com/twilio", getS(t, imp.Inputs, "webhookDestination"))
	require.Equal(t, "delivery events", getS(t, imp.Inputs, "description"))
	require.False(t, hasProp(imp.Inputs, "webhookMethod"), "POST is Twilio's default")
	requireNoChanges(t, l.diff(c.ID, imp.Properties, without(in, "webhookMethod"), imp.Inputs))
	// Stating the defaults is not a change either.
	requireNoChanges(t, l.diff(c.ID, imp.Properties, in, imp.Inputs))

	// Twilio stores the method in lower case ("post"): the default is still recognised on import,
	// a refresh reports no drift, and a non-default method is adopted in upper case.
	require.Equal(t, "post", l.f.get("evsink", testAccount, c.ID)["sink_configuration"].(map[string]any)["method"])
	rf := l.read(c.ID, c.Properties, in)
	require.Equal(t, "POST", getS(t, rf.Inputs, "webhookMethod"))
	requireNoChanges(t, l.diff(c.ID, rf.Properties, in, rf.Inputs))
	get := l.create(with(in, map[string]property.Value{"webhookMethod": sv("GET")}))
	require.Equal(t, "get", l.f.get("evsink", testAccount, get.ID)["sink_configuration"].(map[string]any)["method"])
	gi := l.read(get.ID, empty(), empty())
	require.Equal(t, "GET", getS(t, gi.Inputs, "webhookMethod"))
	gin := with(in, map[string]property.Value{"webhookMethod": sv("GET")})
	requireNoChanges(t, l.diff(get.ID, gi.Properties, gin, gi.Inputs))
	// The sink type's own keys are the only ones sent: the fake rejects any other (the segment key is write_key).
	seg := l.read(l.create(pv(map[string]property.Value{"description": sv("cdp"), "sinkType": sv("segment"), "segmentWriteKey": sv("wk")})).ID, empty(), empty())
	require.Equal(t, "segment", getS(t, seg.Inputs, "sinkType"))

	// A preview creates nothing and invents no ID.
	before := len(l.f.callsMatching("POST", "/events/v1/Sinks"))
	pr, err := l.s.Create(pgp.CreateRequest{Urn: l.urn, Properties: in, DryRun: true})
	require.NoError(t, err)
	require.Empty(t, pr.ID)
	require.Len(t, l.f.callsMatching("POST", "/events/v1/Sinks"), before)

	bad := func(m map[string]property.Value, prop string) {
		t.Helper()
		fails := l.check(m2(m)).Failures
		require.NotEmpty(t, fails)
		require.Equal(t, prop, fails[0].Property)
	}
	bad(map[string]property.Value{"description": sv("d"), "sinkType": sv("pigeon")}, "sinkType")
	bad(map[string]property.Value{"description": sv(""), "sinkType": sv("segment"), "segmentWriteKey": sv("k")}, "description")
	bad(map[string]property.Value{"description": sv("d"), "sinkType": sv("webhook")}, "webhookDestination")
	bad(map[string]property.Value{"description": sv("d"), "sinkType": sv("webhook"), "webhookDestination": sv("https://x"), "webhookMethod": sv("PUT")}, "webhookMethod")
	bad(map[string]property.Value{"description": sv("d"), "sinkType": sv("kinesis"), "kinesisArn": sv("arn")}, "kinesisRoleArn")
	bad(map[string]property.Value{"description": sv("d"), "sinkType": sv("segment")}, "segmentWriteKey")
	bad(map[string]property.Value{"description": sv("d"), "sinkType": sv("segment"), "segmentWriteKey": sv("k"), "webhookDestination": sv("https://x")}, "webhookDestination")
	require.Empty(t, l.check(pv(map[string]property.Value{"description": computed(), "sinkType": computed(), "webhookDestination": computed()})).Failures)
	require.Empty(t, l.check(pv(map[string]property.Value{"description": sv("d"), "sinkType": sv("webhook"), "webhookDestination": computed()})).Failures)
}

func m2(m map[string]property.Value) property.Map { return pv(m) }

func activeSink(t *testing.T, l *tl) string {
	t.Helper()
	sl := l.other("EventSink", "sink")
	c := sl.create(webhookSinkInputs())
	l.f.mutate("evsink", testAccount, c.ID, func(o map[string]any) { o["status"] = "active" })
	return c.ID
}

func TestEventSubscriptionLC_Lifecycle(t *testing.T) {
	l := newTL(t, "EventSubscription", "sub")
	sink := activeSink(t, l)
	in := subInputs(sink, evType("com.twilio.messaging.message.delivered"), evTypeV("com.twilio.messaging.message.failed", 1))
	c := l.create(in)
	sid := c.ID
	require.True(t, strings.HasPrefix(sid, "DF"), sid)
	require.Equal(t, testAccount, getS(t, c.Properties, "accountSid"))
	require.Equal(t, map[string]float64{"com.twilio.messaging.message.delivered": 1, "com.twilio.messaging.message.failed": 1}, l.f.eventsTypes(sid))
	require.JSONEq(t, `[{"type":"com.twilio.messaging.message.delivered"},{"type":"com.twilio.messaging.message.failed","schema_version":1}]`,
		l.f.callsMatching("POST", "/events/v1/Subscriptions")[0].Form.Get("Types"))
	requireNoChanges(t, l.diff(sid, c.Properties, in, in))

	// A refresh changes nothing: the version of an unpinned type is not turned into an input.
	rd := l.read(sid, c.Properties, in)
	requireNoChanges(t, l.diff(sid, rd.Properties, in, rd.Inputs))
	require.False(t, hasProp(rd.Inputs.Get("types").AsArray().Get(0).AsMap(), "schemaVersion"))

	// A type added and one re-versioned outside Pulumi are drift on the inputs.
	l.f.eventsSetTypes(sid, map[string]float64{"com.twilio.messaging.message.delivered": 1, "com.twilio.messaging.message.failed": 2, "com.twilio.voice.call.completed": 1})
	rd = l.read(sid, c.Properties, in)
	require.Equal(t, 3, rd.Inputs.Get("types").AsArray().Len())
	d := l.diff(sid, rd.Properties, in, rd.Inputs)
	require.True(t, d.HasChanges)
	require.False(t, d.DeleteBeforeReplace)

	// Update: description, one type re-versioned, one dropped, one added.
	next := pv(map[string]property.Value{"description": sv("deliveries v2"), "sinkSid": sv(sink),
		"types": typesValue(evTypeV("com.twilio.messaging.message.failed", 3), evType("com.twilio.voice.call.completed"), evType("com.twilio.messaging.message.read"))})
	l.f.resetCalls()
	u := l.update(sid, rd.Properties, next, rd.Inputs)
	require.Equal(t, map[string]float64{"com.twilio.messaging.message.failed": 3, "com.twilio.voice.call.completed": 1, "com.twilio.messaging.message.read": 1}, l.f.eventsTypes(sid))
	require.Equal(t, "deliveries v2", l.f.get("evsub", testAccount, sid)["description"])
	require.Len(t, l.f.callsMatching("DELETE", "SubscribedEvents/com.twilio.messaging.message.delivered"), 1)
	require.Empty(t, l.f.callsMatching("POST", "SubscribedEvents/com.twilio.voice.call.completed"), "an unchanged type is not touched")
	requireNoChanges(t, l.diff(sid, u.Properties, next, next))

	// An idempotent update sends no type calls.
	l.f.resetCalls()
	l.update(sid, u.Properties, next, next)
	require.Empty(t, l.f.callsMatching("DELETE", "SubscribedEvents"))
	require.Empty(t, l.f.callsMatching("POST", "SubscribedEvents"))

	// The sink replaces the subscription.
	d = l.diff(sid, u.Properties, with(next, map[string]property.Value{"sinkSid": sv("DG00000000000000000000000000000099")}), next)
	require.Equal(t, pgp.UpdateReplace, d.DetailedDiff["sinkSid"].Kind)

	// Gone out of band: a refresh drops it, and deleting twice is fine.
	l.f.remove("evsub", testAccount, sid)
	require.Empty(t, l.read(sid, u.Properties, next).ID)
	l.del(sid, u.Properties)
	c = l.create(in)
	l.del(c.ID, c.Properties)
	require.Nil(t, l.f.get("evsub", testAccount, c.ID))
}

func TestEventSubscriptionLC_ImportPaginationPreviewAndChecks(t *testing.T) {
	l := newTL(t, "EventSubscription", "sub")
	sink := activeSink(t, l)
	in := subInputs(sink, evType("a.one"), evType("b.two"), evType("c.three"))
	c := l.create(in)

	// Reading follows the pagination.
	defer func(old int) { eventsPageCap = old }(eventsPageCap)
	eventsPageCap = 2
	l.f.resetCalls()
	rd := l.read(c.ID, c.Properties, in)
	require.Equal(t, 3, rd.Inputs.Get("types").AsArray().Len())
	require.Len(t, l.f.callsMatching("GET", "/SubscribedEvents"), 2)
	requireNoChanges(t, l.diff(c.ID, rd.Properties, in, rd.Inputs))

	// An import adopts the sink and every type with its version.
	imp := l.read(c.ID, empty(), empty())
	require.Equal(t, sink, getS(t, imp.Inputs, "sinkSid"))
	require.Equal(t, "deliveries", getS(t, imp.Inputs, "description"))
	first := imp.Inputs.Get("types").AsArray().Get(0).AsMap()
	require.Equal(t, "a.one", getS(t, first, "type"))
	require.Equal(t, float64(1), first.Get("schemaVersion").AsNumber())
	require.Equal(t, 3, imp.Inputs.Get("types").AsArray().Len())

	// A preview creates nothing and invents no ID.
	before := len(l.f.callsMatching("POST", "/events/v1/Subscriptions"))
	pr, err := l.s.Create(pgp.CreateRequest{Urn: l.urn, Properties: in, DryRun: true})
	require.NoError(t, err)
	require.Empty(t, pr.ID)
	require.Len(t, l.f.callsMatching("POST", "/events/v1/Subscriptions"), before)

	// Twilio only accepts an active sink; the API error reaches the user.
	inactive := l.other("EventSink", "s2").create(webhookSinkInputs()).ID
	_, err = l.s.Create(pgp.CreateRequest{Urn: l.urn, Properties: l.checked(subInputs(inactive, evType("a.one")))})
	require.ErrorContains(t, err, "must be active")

	bad := func(m property.Map, prop string) {
		t.Helper()
		fails := l.check(m).Failures
		require.NotEmpty(t, fails)
		require.Equal(t, prop, fails[0].Property)
	}
	bad(subInputs(sink), "types")
	bad(subInputs(sink, evType("a.one"), evType("a.one")), "types[1].type")
	bad(subInputs(sink, evType("")), "types[0].type")
	bad(subInputs(sink, evTypeV("a.one", 0)), "types[0].schemaVersion")
	bad(with(subInputs(sink, evType("a.one")), map[string]property.Value{"description": sv("")}), "description")
	require.Empty(t, l.check(pv(map[string]property.Value{"description": sv("d"), "sinkSid": computed(), "types": computed()})).Failures)
}

func TestLookup_EventSinkAndSubscription(t *testing.T) {
	l := newTL(t, "EventSubscription", "sub")
	sink := activeSink(t, l)
	r, err := invoke(t, l, "getEventSink", map[string]property.Value{"sid": sv(sink)})
	require.NoError(t, err)
	require.Equal(t, "webhook", getS(t, r, "sinkType"))
	require.Equal(t, "active", getS(t, r, "status"))
	require.Equal(t, "https://hooks.example.com/twilio", getS(t, r, "webhookDestination"))
	require.False(t, hasProp(r, "kinesisArn"))

	c := l.create(subInputs(sink, evTypeV("a.one", 2), evType("b.two")))
	r, err = invoke(t, l, "getEventSubscription", map[string]property.Value{"sid": sv(c.ID)})
	require.NoError(t, err)
	require.Equal(t, sink, getS(t, r, "sinkSid"))
	require.Equal(t, 2, r.Get("types").AsArray().Len())
	require.Equal(t, float64(2), r.Get("types").AsArray().Get(0).AsMap().Get("schemaVersion").AsNumber())

	l.f.remove("evsub", testAccount, c.ID)
	_, err = invoke(t, l, "getEventSubscription", map[string]property.Value{"sid": sv(c.ID)})
	require.ErrorContains(t, err, "was not found")
	_, err = invoke(t, l, "getEventSink", map[string]property.Value{"sid": sv("DG00000000000000000000000000000000")})
	require.ErrorContains(t, err, "was not found")
	_, err = invoke(t, l, "getEventSink", map[string]property.Value{"sid": sv("")})
	require.Error(t, err)
}

// An import cannot know the Segment write key or the Kinesis external ID (Twilio does not return the write key),
// a default webhook method or a false batch flag; a program that sets them must not replace the live sink.
func TestEventSinkLC_ImportThenProgramInputsDoNotReplace(t *testing.T) {
	l := newTL(t, "EventSink", "sink")

	seg := pv(map[string]property.Value{"description": sv("cdp"), "sinkType": sv("segment"), "segmentWriteKey": property.New("wk-123").WithSecret(true)})
	c := l.create(seg)
	imp := l.read(c.ID, empty(), empty())
	require.False(t, hasProp(imp.Inputs, "segmentWriteKey"))
	requireNoChanges(t, l.diff(c.ID, imp.Properties, seg, imp.Inputs))
	require.Equal(t, pgp.Update, l.diff(c.ID, imp.Properties, with(seg, map[string]property.Value{"description": sv("cdp2")}), imp.Inputs).DetailedDiff["description"].Kind)

	kin := pv(map[string]property.Value{"description": sv("stream"), "sinkType": sv("kinesis"),
		"kinesisArn": sv("arn:aws:kinesis:us-east-1:123456789012:stream/events"), "kinesisRoleArn": sv("arn:aws:iam::123456789012:role/twilio"),
		"kinesisExternalId": property.New("ext-1").WithSecret(true)})
	k := l.create(kin)
	l.f.mutate("evsink", testAccount, k.ID, func(o map[string]any) { delete(o["sink_configuration"].(map[string]any), "external_id") })
	imp = l.read(k.ID, empty(), empty())
	require.False(t, hasProp(imp.Inputs, "kinesisExternalId"))
	requireNoChanges(t, l.diff(k.ID, imp.Properties, kin, imp.Inputs))

	off := with(webhookSinkInputs(), map[string]property.Value{"webhookBatchEvents": bv(false), "webhookMethod": sv("POST")})
	w := l.create(off)
	imp = l.read(w.ID, empty(), empty())
	requireNoChanges(t, l.diff(w.ID, imp.Properties, off, imp.Inputs))
	requireNoChanges(t, l.diff(w.ID, imp.Properties, without(off, "webhookBatchEvents", "webhookMethod"), imp.Inputs))

	// A real change still replaces.
	df := l.diff(w.ID, imp.Properties, with(off, map[string]property.Value{"webhookMethod": sv("GET")}), imp.Inputs)
	require.Equal(t, pgp.UpdateReplace, df.DetailedDiff["webhookMethod"].Kind)
	df = l.diff(w.ID, imp.Properties, with(off, map[string]property.Value{"webhookBatchEvents": bv(true)}), imp.Inputs)
	require.Equal(t, pgp.UpdateReplace, df.DetailedDiff["webhookBatchEvents"].Kind)
}

// A types call that fails part way leaves the state as it was; the retry finishes the same diff.
func TestEventSubscriptionLC_UpdateRetryAfterPartialFailure(t *testing.T) {
	l := newTL(t, "EventSubscription", "sub")
	sink := activeSink(t, l)
	in := subInputs(sink, evType("a.one"), evType("b.two"))
	c := l.create(in)
	next := subInputs(sink, evType("b.two"), evType("c.three"), evType("d.four"))
	next = with(next, map[string]property.Value{"description": sv("deliveries v2")})

	l.f.failNext("POST", "SubscribedEvents", 500)
	_, err := l.s.Update(pgp.UpdateRequest{ID: c.ID, Urn: l.urn, State: c.Properties, Inputs: l.checked(next), OldInputs: in})
	require.Error(t, err)
	require.Equal(t, "deliveries v2", l.f.get("evsub", testAccount, c.ID)["description"])

	u := l.update(c.ID, c.Properties, next, in)
	require.Equal(t, map[string]float64{"b.two": 1, "c.three": 1, "d.four": 1}, l.f.eventsTypes(c.ID))
	requireNoChanges(t, l.diff(c.ID, u.Properties, next, next))
}
