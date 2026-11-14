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

func trWorkspaceInputs() property.Map {
	return pv(map[string]property.Value{
		"friendlyName": sv("support"), "eventCallbackUrl": sv("https://example.com/events"),
		"eventsFilter": sv("task.created,worker.activity.update"), "prioritizeQueueOrder": sv("LIFO"),
		"multiTaskEnabled": property.New(true),
	})
}

// trWS creates a workspace on the fake and returns its SID.
func trWS(l *tl) string { return l.other("TaskRouterWorkspace", "ws").create(trWorkspaceInputs()).ID }

func TestTaskRouterWorkspaceLC_Lifecycle(t *testing.T) {
	l := newTL(t, "TaskRouterWorkspace", "ws")
	in := trWorkspaceInputs()
	c := l.create(in)
	require.True(t, strings.HasPrefix(c.ID, "WS"), c.ID)
	require.Equal(t, c.ID, getS(t, c.Properties, "sid"))
	require.True(t, strings.HasPrefix(getS(t, c.Properties, "defaultActivitySid"), "WA"))
	form := l.f.callsMatching("POST", "/taskrouter/v1/Workspaces")[0].Form
	require.Equal(t, "support", form.Get("FriendlyName"))
	require.Equal(t, "LIFO", form.Get("PrioritizeQueueOrder"))
	require.Equal(t, "true", form.Get("MultiTaskEnabled"))
	requireNoChanges(t, l.diff(c.ID, c.Properties, in, in))

	// A refresh changes nothing; an out-of-band change is drift.
	rd := l.read(c.ID, c.Properties, in)
	requireNoChanges(t, l.diff(c.ID, rd.Properties, in, rd.Inputs))
	l.f.mutate("trws", testAccount, c.ID, func(o map[string]any) { o["event_callback_url"] = "https://example.com/other" })
	rd = l.read(c.ID, c.Properties, in)
	require.Equal(t, "https://example.com/other", getS(t, rd.Inputs, "eventCallbackUrl"))
	d := l.diff(c.ID, rd.Properties, in, rd.Inputs)
	require.Equal(t, pgp.Update, d.DetailedDiff["eventCallbackUrl"].Kind)
	require.Len(t, d.DetailedDiff, 1)

	// In-place update.
	next := with(in, map[string]property.Value{"friendlyName": sv("support-2")})
	require.Equal(t, pgp.Update, l.diff(c.ID, c.Properties, next, in).DetailedDiff["friendlyName"].Kind)
	l.f.resetCalls()
	u := l.update(c.ID, c.Properties, next, in)
	require.Equal(t, "support-2", l.f.get("trws", testAccount, c.ID)["friendly_name"])
	require.Equal(t, trUpdatedStamp, getS(t, u.Properties, "dateUpdated"))
	requireNoChanges(t, l.diff(c.ID, u.Properties, next, next))

	// Removing an optional input is not a change and is not sent.
	gone := without(next, "eventsFilter")
	requireNoChanges(t, l.diff(c.ID, u.Properties, gone, next))

	// Gone out of band: a refresh drops it, and deleting twice is fine.
	l.f.remove("trws", testAccount, c.ID)
	require.Empty(t, l.read(c.ID, u.Properties, next).ID)
	l.del(c.ID, u.Properties)
	c = l.create(in)
	l.del(c.ID, c.Properties)
	require.Nil(t, l.f.get("trws", testAccount, c.ID))
}

func TestTaskRouterWorkspaceLC_Import(t *testing.T) {
	l := newTL(t, "TaskRouterWorkspace", "ws")
	minimal := pv(map[string]property.Value{"friendlyName": sv("bare")})
	c := l.create(minimal)
	require.Empty(t, l.f.callsMatching("POST", "/Workspaces")[0].Form["EventCallbackUrl"])
	rd := l.read(c.ID, empty(), empty())
	require.Equal(t, "bare", getS(t, rd.Inputs, "friendlyName"))
	for _, k := range []string{"eventCallbackUrl", "eventsFilter", "multiTaskEnabled", "prioritizeQueueOrder"} {
		require.False(t, hasProp(rd.Inputs, k), k)
	}
	requireNoChanges(t, l.diff(c.ID, rd.Properties, minimal, rd.Inputs))

	full := l.other("TaskRouterWorkspace", "full").create(trWorkspaceInputs())
	rd = l.read(full.ID, empty(), empty())
	require.Equal(t, "LIFO", getS(t, rd.Inputs, "prioritizeQueueOrder"))
	require.True(t, rd.Inputs.Get("multiTaskEnabled").AsBool())
	requireNoChanges(t, l.diff(full.ID, rd.Properties, trWorkspaceInputs(), rd.Inputs))
	l.requireImportFails("WS" + strings.Repeat("0", 32))
}

func TestTaskRouterWorkspaceLC_Check(t *testing.T) {
	l := newTL(t, "TaskRouterWorkspace", "ws")
	chk := l.check(pv(map[string]property.Value{"friendlyName": sv("x"), "prioritizeQueueOrder": sv("random")}))
	require.Len(t, chk.Failures, 1)
	require.Equal(t, "prioritizeQueueOrder", chk.Failures[0].Property)
}

func TestTaskRouterActivityLC_Lifecycle(t *testing.T) {
	l := newTL(t, "TaskRouterActivity", "act")
	ws := trWS(l)
	in := pv(map[string]property.Value{"workspaceSid": sv(ws), "friendlyName": sv("Busy"), "available": property.New(false)})
	c := l.create(in)
	sid := getS(t, c.Properties, "sid")
	require.True(t, strings.HasPrefix(sid, "WA"))
	require.Equal(t, ws+"/"+sid, c.ID)
	require.Equal(t, "false", l.f.callsMatching("POST", "/Activities")[0].Form.Get("Available"))
	requireNoChanges(t, l.diff(c.ID, c.Properties, in, in))
	rd := l.read(c.ID, c.Properties, in)
	requireNoChanges(t, l.diff(c.ID, rd.Properties, in, rd.Inputs))

	// The name updates in place; available and the workspace replace.
	next := with(in, map[string]property.Value{"friendlyName": sv("Away")})
	require.Equal(t, pgp.Update, l.diff(c.ID, c.Properties, next, in).DetailedDiff["friendlyName"].Kind)
	u := l.update(c.ID, c.Properties, next, in)
	require.Equal(t, "Away", l.f.get("trchild", testAccount, c.ID)["friendly_name"])
	requireNoChanges(t, l.diff(c.ID, u.Properties, next, next))
	d := l.diff(c.ID, u.Properties, with(next, map[string]property.Value{"available": property.New(true)}), next)
	require.Equal(t, pgp.UpdateReplace, d.DetailedDiff["available"].Kind)
	d = l.diff(c.ID, u.Properties, with(next, map[string]property.Value{"workspaceSid": sv("WS" + strings.Repeat("1", 32))}), next)
	require.Equal(t, pgp.UpdateReplace, d.DetailedDiff["workspaceSid"].Kind)

	// Import adopts the workspace and name; available only when it is not the default true.
	imp := l.read(c.ID, empty(), empty())
	require.Equal(t, ws, getS(t, imp.Inputs, "workspaceSid"))
	require.False(t, imp.Inputs.Get("available").AsBool())
	l.f.mutate("trchild", testAccount, c.ID, func(o map[string]any) { o["available"] = true })
	imp = l.read(c.ID, empty(), empty())
	require.False(t, hasProp(imp.Inputs, "available"))

	l.f.remove("trchild", testAccount, c.ID)
	require.Empty(t, l.read(c.ID, u.Properties, next).ID)
	l.del(c.ID, u.Properties)
	l.requireImportFails(c.ID)
}

func TestTaskRouterTaskChannelLC_Lifecycle(t *testing.T) {
	l := newTL(t, "TaskRouterTaskChannel", "ch")
	ws := trWS(l)
	in := pv(map[string]property.Value{"workspaceSid": sv(ws), "uniqueName": sv("chat"), "friendlyName": sv("Chat"), "channelOptimizedRouting": property.New(true)})
	c := l.create(in)
	form := l.f.callsMatching("POST", "/TaskChannels")[0].Form
	require.Equal(t, "chat", form.Get("UniqueName"))
	require.Equal(t, "true", form.Get("ChannelOptimizedRouting"))
	requireNoChanges(t, l.diff(c.ID, c.Properties, in, in))
	rd := l.read(c.ID, c.Properties, in)
	requireNoChanges(t, l.diff(c.ID, rd.Properties, in, rd.Inputs))
	l.f.mutate("trchild", testAccount, c.ID, func(o map[string]any) { o["channel_optimized_routing"] = false })
	rd = l.read(c.ID, c.Properties, in)
	require.Equal(t, pgp.Update, l.diff(c.ID, rd.Properties, in, rd.Inputs).DetailedDiff["channelOptimizedRouting"].Kind)

	next := with(in, map[string]property.Value{"friendlyName": sv("Chat 2")})
	u := l.update(c.ID, c.Properties, next, in)
	require.Equal(t, "Chat 2", l.f.get("trchild", testAccount, c.ID)["friendly_name"])
	requireNoChanges(t, l.diff(c.ID, u.Properties, next, next))
	d := l.diff(c.ID, u.Properties, with(next, map[string]property.Value{"uniqueName": sv("other")}), next)
	require.Equal(t, pgp.UpdateReplace, d.DetailedDiff["uniqueName"].Kind)

	imp := l.read(c.ID, empty(), empty())
	require.Equal(t, "chat", getS(t, imp.Inputs, "uniqueName"))
	l.del(c.ID, u.Properties)
	l.del(c.ID, u.Properties)
	require.Empty(t, l.read(c.ID, u.Properties, next).ID)
}

func TestTaskRouterTaskQueueLC_Lifecycle(t *testing.T) {
	l := newTL(t, "TaskRouterTaskQueue", "q")
	ws := trWS(l)
	act := l.other("TaskRouterActivity", "act").create(pv(map[string]property.Value{"workspaceSid": sv(ws), "friendlyName": sv("Reserved")}))
	actSid := getS(t, act.Properties, "sid")
	in := pv(map[string]property.Value{
		"workspaceSid": sv(ws), "friendlyName": sv("support"), "targetWorkers": sv(`skills HAS "support"`),
		"maxReservedWorkers": property.New(3.0), "taskOrder": sv("LIFO"), "reservationActivitySid": sv(actSid),
	})
	c := l.create(in)
	require.True(t, strings.HasPrefix(getS(t, c.Properties, "sid"), "WQ"))
	form := l.f.callsMatching("POST", "/TaskQueues")[0].Form
	require.Equal(t, "3", form.Get("MaxReservedWorkers"))
	require.Equal(t, `skills HAS "support"`, form.Get("TargetWorkers"))
	require.Empty(t, form["AssignmentActivitySid"])
	requireNoChanges(t, l.diff(c.ID, c.Properties, in, in))
	rd := l.read(c.ID, c.Properties, in)
	requireNoChanges(t, l.diff(c.ID, rd.Properties, in, rd.Inputs))

	l.f.mutate("trchild", testAccount, c.ID, func(o map[string]any) { o["max_reserved_workers"] = float64(5) })
	rd = l.read(c.ID, c.Properties, in)
	d := l.diff(c.ID, rd.Properties, in, rd.Inputs)
	require.Equal(t, pgp.Update, d.DetailedDiff["maxReservedWorkers"].Kind)
	require.Len(t, d.DetailedDiff, 1)

	next := with(in, map[string]property.Value{"taskOrder": sv("FIFO")})
	u := l.update(c.ID, c.Properties, next, in)
	require.Equal(t, "FIFO", l.f.get("trchild", testAccount, c.ID)["task_order"])
	requireNoChanges(t, l.diff(c.ID, u.Properties, next, next))
	requireNoChanges(t, l.diff(c.ID, u.Properties, without(next, "targetWorkers"), next))

	// Import adopts only values that differ from Twilio's defaults.
	imp := l.read(c.ID, empty(), empty())
	require.Equal(t, `skills HAS "support"`, getS(t, imp.Inputs, "targetWorkers"))
	require.False(t, hasProp(imp.Inputs, "taskOrder"))
	require.Equal(t, actSid, getS(t, imp.Inputs, "reservationActivitySid"))
	require.False(t, hasProp(imp.Inputs, "assignmentActivitySid"))

	bad := l.check(with(in, map[string]property.Value{"maxReservedWorkers": property.New(99.0), "reservationActivitySid": sv("nope")}))
	require.Len(t, bad.Failures, 2)

	l.f.remove("trchild", testAccount, c.ID)
	require.Empty(t, l.read(c.ID, u.Properties, next).ID)
	l.del(c.ID, u.Properties)
}

const (
	trConfigA = `{"task_routing":{"default_filter":{"queue":"WQ00000000000000000000000000000001"}}}`
	trConfigB = `{"task_routing":{"default_filter":{"queue":"WQ00000000000000000000000000000002"}}}`
)

func TestTaskRouterWorkflowLC_Lifecycle(t *testing.T) {
	l := newTL(t, "TaskRouterWorkflow", "wf")
	ws := trWS(l)
	in := pv(map[string]property.Value{"workspaceSid": sv(ws), "friendlyName": sv("route"), "configuration": sv(trConfigA),
		"assignmentCallbackUrl": sv("https://example.com/assign"), "taskReservationTimeout": property.New(60.0)})
	c := l.create(in)
	require.True(t, strings.HasPrefix(getS(t, c.Properties, "sid"), "WW"))
	form := l.f.callsMatching("POST", "/Workflows")[0].Form
	require.Equal(t, trConfigA, form.Get("Configuration"))
	require.Equal(t, "60", form.Get("TaskReservationTimeout"))
	requireNoChanges(t, l.diff(c.ID, c.Properties, in, in))

	// The same document formatted differently is not a change, nor drift.
	spaced := with(in, map[string]property.Value{"configuration": sv(`{ "task_routing": { "default_filter": { "queue": "WQ00000000000000000000000000000001" } } }`)})
	requireNoChanges(t, l.diff(c.ID, c.Properties, spaced, in))
	l.f.mutate("trchild", testAccount, c.ID, func(o map[string]any) {
		o["configuration"] = `{"task_routing": {"default_filter": {"queue": "WQ00000000000000000000000000000001"}}}`
	})
	rd := l.read(c.ID, c.Properties, in)
	require.Equal(t, trConfigA, getS(t, rd.Inputs, "configuration"), "a reformatted document is kept as written")
	requireNoChanges(t, l.diff(c.ID, rd.Properties, in, rd.Inputs))

	// A different document updates in place.
	next := with(in, map[string]property.Value{"configuration": sv(trConfigB)})
	require.Equal(t, pgp.Update, l.diff(c.ID, c.Properties, next, in).DetailedDiff["configuration"].Kind)
	u := l.update(c.ID, c.Properties, next, in)
	require.Equal(t, trConfigB, l.f.get("trchild", testAccount, c.ID)["configuration"])
	requireNoChanges(t, l.diff(c.ID, u.Properties, next, next))
	l.f.mutate("trchild", testAccount, c.ID, func(o map[string]any) { o["configuration"] = trConfigA })
	rd = l.read(c.ID, u.Properties, next)
	require.Equal(t, pgp.Update, l.diff(c.ID, rd.Properties, next, rd.Inputs).DetailedDiff["configuration"].Kind)

	// Import adopts the document and the settings that differ from the defaults.
	imp := l.read(c.ID, empty(), empty())
	require.Equal(t, trConfigA, getS(t, imp.Inputs, "configuration"))
	require.Equal(t, 60.0, imp.Inputs.Get("taskReservationTimeout").AsNumber())
	require.False(t, hasProp(imp.Inputs, "fallbackAssignmentCallbackUrl"))

	require.Len(t, l.check(with(in, map[string]property.Value{"configuration": sv("{nope")})).Failures, 1)
	l.f.remove("trchild", testAccount, c.ID)
	require.Empty(t, l.read(c.ID, u.Properties, next).ID)
	l.del(c.ID, u.Properties)
}

func TestTaskRouterWorkerLC_Lifecycle(t *testing.T) {
	l := newTL(t, "TaskRouterWorker", "wk")
	ws := trWS(l)
	act := l.other("TaskRouterActivity", "act").create(pv(map[string]property.Value{"workspaceSid": sv(ws), "friendlyName": sv("Idle")}))
	actSid := getS(t, act.Properties, "sid")
	in := pv(map[string]property.Value{"workspaceSid": sv(ws), "friendlyName": sv("alice"), "attributes": sv(`{"skills":["support"]}`), "activitySid": sv(actSid)})
	c := l.create(in)
	require.True(t, strings.HasPrefix(getS(t, c.Properties, "sid"), "WK"))
	form := l.f.callsMatching("POST", "/Workers")[0].Form
	require.Equal(t, `{"skills":["support"]}`, form.Get("Attributes"))
	require.Equal(t, actSid, form.Get("ActivitySid"))
	requireNoChanges(t, l.diff(c.ID, c.Properties, in, in))

	// A worker that changed activity on its own is not drift; changed attributes are.
	l.f.mutate("trchild", testAccount, c.ID, func(o map[string]any) { o["activity_sid"] = "WA" + strings.Repeat("9", 32) })
	rd := l.read(c.ID, c.Properties, in)
	requireNoChanges(t, l.diff(c.ID, rd.Properties, in, rd.Inputs))
	l.f.mutate("trchild", testAccount, c.ID, func(o map[string]any) { o["attributes"] = `{"skills":["billing"]}` })
	rd = l.read(c.ID, c.Properties, in)
	require.Equal(t, pgp.Update, l.diff(c.ID, rd.Properties, in, rd.Inputs).DetailedDiff["attributes"].Kind)

	// An update sends the activity only when the program changed it.
	next := with(in, map[string]property.Value{"friendlyName": sv("alice b")})
	l.f.resetCalls()
	u := l.update(c.ID, c.Properties, next, in)
	require.Empty(t, l.f.callsMatching("POST", "/Workers/")[0].Form["ActivitySid"])
	moved := with(next, map[string]property.Value{"activitySid": sv("WA" + strings.Repeat("a", 32))})
	require.Equal(t, pgp.Update, l.diff(c.ID, u.Properties, moved, next).DetailedDiff["activitySid"].Kind)
	l.f.resetCalls()
	u = l.update(c.ID, u.Properties, moved, next)
	require.Equal(t, "WA"+strings.Repeat("a", 32), l.f.callsMatching("POST", "/Workers/")[0].Form.Get("ActivitySid"))

	// Import adopts attributes unless they are the default empty object, and the activity (#1981), so
	// a program that states the activity the worker is in plans no change after the import.
	imp := l.read(c.ID, empty(), empty())
	require.Equal(t, `{"skills":["support"]}`, getS(t, imp.Inputs, "attributes"))
	require.Equal(t, "WA"+strings.Repeat("a", 32), getS(t, imp.Inputs, "activitySid"))
	requireNoChanges(t, l.diff(c.ID, imp.Properties, moved, imp.Inputs))
	// A later refresh keeps the activity the program stated even when the worker moved.
	l.f.mutate("trchild", testAccount, c.ID, func(o map[string]any) { o["activity_sid"] = "WA" + strings.Repeat("7", 32) })
	rf := l.read(c.ID, imp.Properties, imp.Inputs)
	require.Equal(t, "WA"+strings.Repeat("a", 32), getS(t, rf.Inputs, "activitySid"))
	l.f.mutate("trchild", testAccount, c.ID, func(o map[string]any) { o["attributes"] = "{}" })
	require.False(t, hasProp(l.read(c.ID, empty(), empty()).Inputs, "attributes"))

	l.f.remove("trchild", testAccount, c.ID)
	require.Empty(t, l.read(c.ID, u.Properties, moved).ID)
	l.del(c.ID, u.Properties)
}

func TestTaskRouterDeleteWorkspaceRemovesChildren(t *testing.T) {
	l := newTL(t, "TaskRouterWorkspace", "ws")
	c := l.create(trWorkspaceInputs())
	ch := l.other("TaskRouterTaskChannel", "ch").create(pv(map[string]property.Value{"workspaceSid": sv(c.ID), "uniqueName": sv("voice"), "friendlyName": sv("Voice")}))
	l.del(c.ID, c.Properties)
	// Destroying the child after its workspace is fine.
	l.other("TaskRouterTaskChannel", "ch").del(ch.ID, ch.Properties)
}

func TestLookup_TaskRouter(t *testing.T) {
	l := newTL(t, "TaskRouterWorkspace", "ws")
	ws := l.create(trWorkspaceInputs()).ID
	r, err := invoke(t, l, "getTaskRouterWorkspace", map[string]property.Value{"workspaceSid": sv(ws)})
	require.NoError(t, err)
	require.Equal(t, ws, getS(t, r, "sid"))
	require.Equal(t, "support", getS(t, r, "friendlyName"))
	require.Equal(t, "LIFO", getS(t, r, "prioritizeQueueOrder"))
	_, err = invoke(t, l, "getTaskRouterWorkspace", map[string]property.Value{"workspaceSid": sv("WSmissing")})
	require.ErrorContains(t, err, "was not found")
	_, err = invoke(t, l, "getTaskRouterWorkspace", map[string]property.Value{"workspaceSid": sv("")})
	require.ErrorContains(t, err, "must be set")

	mk := func(typ string, in map[string]property.Value) string {
		in["workspaceSid"] = sv(ws)
		return strings.SplitN(l.other(typ, "x").create(pv(in)).ID, "/", 2)[1]
	}
	look := func(fn, sid string) property.Map {
		out, err := invoke(t, l, fn, map[string]property.Value{"workspaceSid": sv(ws), "sid": sv(sid)})
		require.NoError(t, err, fn)
		_, err = invoke(t, l, fn, map[string]property.Value{"workspaceSid": sv(ws), "sid": sv("XXmissing")})
		require.ErrorContains(t, err, "was not found", fn)
		_, err = invoke(t, l, fn, map[string]property.Value{"workspaceSid": sv(ws), "sid": sv("")})
		require.ErrorContains(t, err, "must be set", fn)
		return out
	}
	sid := mk("TaskRouterActivity", map[string]property.Value{"friendlyName": sv("Busy")})
	out := look("getTaskRouterActivity", sid)
	require.Equal(t, "Busy", getS(t, out, "friendlyName"))
	require.True(t, out.Get("available").AsBool())

	sid = mk("TaskRouterTaskChannel", map[string]property.Value{"uniqueName": sv("sms"), "friendlyName": sv("SMS")})
	require.Equal(t, "sms", getS(t, look("getTaskRouterTaskChannel", sid), "uniqueName"))

	sid = mk("TaskRouterTaskQueue", map[string]property.Value{"friendlyName": sv("q")})
	out = look("getTaskRouterTaskQueue", sid)
	require.Equal(t, "1==1", getS(t, out, "targetWorkers"))
	require.Equal(t, float64(1), out.Get("maxReservedWorkers").AsNumber())
	require.False(t, hasProp(out, "reservationActivitySid"))

	sid = mk("TaskRouterWorkflow", map[string]property.Value{"friendlyName": sv("w"), "configuration": sv(trConfigA)})
	require.Equal(t, trConfigA, getS(t, look("getTaskRouterWorkflow", sid), "configuration"))

	sid = mk("TaskRouterWorker", map[string]property.Value{"friendlyName": sv("alice")})
	out = look("getTaskRouterWorker", sid)
	require.Equal(t, "{}", getS(t, out, "attributes"))
	require.True(t, strings.HasPrefix(getS(t, out, "activitySid"), "WA"))
}

// An import leaves an input unset when Twilio's value is the default, so a program that sets the
// default explicitly must plan no change, for every TaskRouter resource (#1880). Activity's
// `available` replaces, so a spurious diff there would destroy the activity.
func TestTaskRouterImportThenExplicitDefaultsPlansNothing(t *testing.T) {
	l := newTL(t, "TaskRouterWorkspace", "ws")
	ws := trWS(l)
	boolV := func(b bool) property.Value { return property.New(b) }
	num := func(n float64) property.Value { return property.New(n) }

	type tc struct {
		typ      string
		minimal  map[string]property.Value // creates the object with Twilio defaults
		explicit map[string]property.Value // the same program with every default written out
		changed  map[string]property.Value // a non-default value that must still plan a change
	}
	cases := []tc{
		{"TaskRouterActivity",
			map[string]property.Value{"workspaceSid": sv(ws), "friendlyName": sv("Idle")},
			map[string]property.Value{"workspaceSid": sv(ws), "friendlyName": sv("Idle"), "available": boolV(true)},
			map[string]property.Value{"workspaceSid": sv(ws), "friendlyName": sv("Idle"), "available": boolV(false)}},
		{"TaskRouterTaskChannel",
			map[string]property.Value{"workspaceSid": sv(ws), "friendlyName": sv("Voice"), "uniqueName": sv("voice")},
			map[string]property.Value{"workspaceSid": sv(ws), "friendlyName": sv("Voice"), "uniqueName": sv("voice"), "channelOptimizedRouting": boolV(false)},
			map[string]property.Value{"workspaceSid": sv(ws), "friendlyName": sv("Voice"), "uniqueName": sv("voice"), "channelOptimizedRouting": boolV(true)}},
		{"TaskRouterTaskQueue",
			map[string]property.Value{"workspaceSid": sv(ws), "friendlyName": sv("q")},
			map[string]property.Value{"workspaceSid": sv(ws), "friendlyName": sv("q"), "targetWorkers": sv("1==1"), "maxReservedWorkers": num(1), "taskOrder": sv("FIFO"),
				"reservationActivitySid": sv(""), "assignmentActivitySid": sv("")},
			map[string]property.Value{"workspaceSid": sv(ws), "friendlyName": sv("q"), "taskOrder": sv("LIFO")}},
		{"TaskRouterWorkflow",
			map[string]property.Value{"workspaceSid": sv(ws), "friendlyName": sv("w"), "configuration": sv(trConfigA)},
			map[string]property.Value{"workspaceSid": sv(ws), "friendlyName": sv("w"), "configuration": sv(trConfigA), "taskReservationTimeout": num(120),
				"assignmentCallbackUrl": sv(""), "fallbackAssignmentCallbackUrl": sv("")},
			map[string]property.Value{"workspaceSid": sv(ws), "friendlyName": sv("w"), "configuration": sv(trConfigA), "taskReservationTimeout": num(60)}},
		{"TaskRouterWorker",
			map[string]property.Value{"workspaceSid": sv(ws), "friendlyName": sv("alice")},
			map[string]property.Value{"workspaceSid": sv(ws), "friendlyName": sv("alice"), "attributes": sv("{}")},
			map[string]property.Value{"workspaceSid": sv(ws), "friendlyName": sv("alice"), "attributes": sv(`{"a":1}`)}},
	}
	for _, c := range cases {
		lc := l.other(c.typ, "x")
		created := lc.create(pv(c.minimal))
		imp := lc.read(created.ID, empty(), empty())
		require.NotEmpty(t, imp.ID, c.typ)
		requireNoChanges(t, lc.diff(imp.ID, imp.Properties, pv(c.explicit), imp.Inputs))
		require.True(t, lc.diff(imp.ID, imp.Properties, pv(c.changed), imp.Inputs).HasChanges, "%s: a non-default value still plans a change", c.typ)
	}

	// The workspace itself.
	wl := l.other("TaskRouterWorkspace", "ws2")
	created := wl.create(pv(map[string]property.Value{"friendlyName": sv("plain")}))
	imp := wl.read(created.ID, empty(), empty())
	explicit := pv(map[string]property.Value{"friendlyName": sv("plain"), "eventCallbackUrl": sv(""), "eventsFilter": sv(""),
		"multiTaskEnabled": boolV(false), "prioritizeQueueOrder": sv("FIFO")})
	requireNoChanges(t, wl.diff(imp.ID, imp.Properties, explicit, imp.Inputs))
	require.True(t, wl.diff(imp.ID, imp.Properties, with(explicit, map[string]property.Value{"multiTaskEnabled": boolV(true)}), imp.Inputs).HasChanges)
}

// #1982: Twilio refuses (20001) to delete a worker in an available activity; Delete moves it to
// the Offline activity (an unavailable one) first and then deletes it.
func TestTaskRouterWorkerLC_DeleteMovesAvailableWorkerOffline(t *testing.T) {
	l := newTL(t, "TaskRouterWorker", "wk")
	ws := trWS(l)
	acts := l.other("TaskRouterActivity", "act")
	on := acts.create(pv(map[string]property.Value{"workspaceSid": sv(ws), "friendlyName": sv("Idle"), "available": bv(true)}))
	acts.create(pv(map[string]property.Value{"workspaceSid": sv(ws), "friendlyName": sv("Break"), "available": bv(false)}))
	off := acts.create(pv(map[string]property.Value{"workspaceSid": sv(ws), "friendlyName": sv("Offline"), "available": bv(false)}))
	in := pv(map[string]property.Value{"workspaceSid": sv(ws), "friendlyName": sv("alice"), "activitySid": sv(getS(t, on.Properties, "sid"))})
	c := l.create(in)

	l.f.resetCalls()
	l.del(c.ID, c.Properties)
	workerPath := "/Workers/" + strings.SplitN(c.ID, "/", 2)[1]
	var seq []string
	for _, call := range l.f.calls {
		if strings.Contains(call.Path, workerPath) {
			seq = append(seq, call.Method)
		}
	}
	require.Equal(t, []string{"DELETE", "POST", "DELETE"}, seq, "refused delete, move, delete")
	post := l.f.callsMatching("POST", workerPath)[0].Form
	require.Equal(t, getS(t, off.Properties, "sid"), post.Get("ActivitySid"), "moved to Offline, not the first unavailable activity")
	require.Empty(t, l.read(c.ID, c.Properties, in).ID, "the worker is gone")
}

// With no unavailable activity the error names the problem instead of a bare 20001.
func TestTaskRouterWorkerLC_DeleteNoUnavailableActivity(t *testing.T) {
	l := newTL(t, "TaskRouterWorker", "wk")
	ws := trWS(l)
	on := l.other("TaskRouterActivity", "act").create(pv(map[string]property.Value{"workspaceSid": sv(ws), "friendlyName": sv("Idle"), "available": bv(true)}))
	c := l.create(pv(map[string]property.Value{"workspaceSid": sv(ws), "friendlyName": sv("bob"), "activitySid": sv(getS(t, on.Properties, "sid"))}))
	err := l.s.Delete(pgp.DeleteRequest{ID: c.ID, Urn: l.urn, Properties: c.Properties})
	require.ErrorContains(t, err, "no activity with available false")
}
