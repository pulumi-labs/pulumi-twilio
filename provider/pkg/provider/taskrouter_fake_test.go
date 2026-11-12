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
	"net/http"
	"net/url"
	"strings"
)

// The TaskRouter v1 routes, served for the TaskRouter tests:
//
//	/taskrouter/v1/Workspaces[/{ws}]
//	/taskrouter/v1/Workspaces/{ws}/{Activities|TaskChannels|TaskQueues|Workflows|Workers}[/{sid}]
//
// The shapes follow twilio-oai spec/json/twilio_taskrouter_v1.json (spec version 1.0.0). Workspaces are
// stored as kind "trws" and children as kind "trchild" keyed <ws>/<sid>.

const (
	trStamp        = "2026-11-12T00:00:00Z"
	trUpdatedStamp = "2026-11-12T01:00:00Z"
)

type trSpec struct {
	prefix   string
	required []string
	create   []fakeField
	update   []fakeField
}

var (
	trWorkspaceFields = []fakeField{
		{"FriendlyName", "friendly_name", "s"}, {"EventCallbackUrl", "event_callback_url", "u"},
		{"EventsFilter", "events_filter", "u"}, {"MultiTaskEnabled", "multi_task_enabled", "b"},
		{"PrioritizeQueueOrder", "prioritize_queue_order", "s"},
	}
	trSpecs = map[string]trSpec{
		"Activities": {"WA", []string{"FriendlyName"},
			[]fakeField{{"FriendlyName", "friendly_name", "s"}, {"Available", "available", "b"}},
			[]fakeField{{"FriendlyName", "friendly_name", "s"}}},
		"TaskChannels": {"TC", []string{"FriendlyName", "UniqueName"},
			[]fakeField{{"FriendlyName", "friendly_name", "s"}, {"UniqueName", "unique_name", "s"}, {"ChannelOptimizedRouting", "channel_optimized_routing", "b"}},
			[]fakeField{{"FriendlyName", "friendly_name", "s"}, {"ChannelOptimizedRouting", "channel_optimized_routing", "b"}}},
		"TaskQueues": {"WQ", []string{"FriendlyName"},
			[]fakeField{{"FriendlyName", "friendly_name", "s"}, {"TargetWorkers", "target_workers", "s"}, {"MaxReservedWorkers", "max_reserved_workers", "i"},
				{"TaskOrder", "task_order", "s"}, {"ReservationActivitySid", "reservation_activity_sid", "u"}, {"AssignmentActivitySid", "assignment_activity_sid", "u"}},
			nil},
		"Workflows": {"WW", []string{"FriendlyName", "Configuration"},
			[]fakeField{{"FriendlyName", "friendly_name", "s"}, {"Configuration", "configuration", "s"}, {"AssignmentCallbackUrl", "assignment_callback_url", "u"},
				{"FallbackAssignmentCallbackUrl", "fallback_assignment_callback_url", "u"}, {"TaskReservationTimeout", "task_reservation_timeout", "i"}},
			nil},
		"Workers": {"WK", []string{"FriendlyName"},
			[]fakeField{{"FriendlyName", "friendly_name", "s"}, {"Attributes", "attributes", "s"}, {"ActivitySid", "activity_sid", "u"}},
			nil},
	}
)

func (f *fakeTwilio) handleTaskRouter(w http.ResponseWriter, r *http.Request, form url.Values, rest string) {
	notFound := func() { ferr(w, 404, "The requested resource "+r.URL.Path+" was not found") }
	var parts []string
	if rest != "" {
		parts = strings.Split(rest, "/")
	}
	switch len(parts) {
	case 0, 1:
		sid := ""
		if len(parts) == 1 {
			sid = parts[0]
		}
		f.trWorkspace(w, r, form, sid, notFound)
		return
	case 2, 3:
		if _, ok := f.objs[key("trws", testAccount, parts[0])]; !ok {
			notFound()
			return
		}
		spec, ok := trSpecs[parts[1]]
		if !ok {
			notFound()
			return
		}
		sid := ""
		if len(parts) == 3 {
			sid = parts[2]
		}
		f.trChild(w, r, form, parts[0], parts[1], spec, sid, notFound)
		return
	}
	notFound()
}

func (f *fakeTwilio) trWorkspace(w http.ResponseWriter, r *http.Request, form url.Values, sid string, notFound func()) {
	if sid == "" {
		if r.Method != "POST" {
			ferr(w, 405, "method not allowed")
			return
		}
		if form.Get("FriendlyName") == "" {
			ferr(w, 400, "Missing required parameter FriendlyName")
			return
		}
		id := f.sid("WS")
		o := map[string]any{"sid": id, "account_sid": testAccount, "friendly_name": nil, "event_callback_url": nil, "events_filter": nil,
			"multi_task_enabled": false, "prioritize_queue_order": "FIFO",
			"default_activity_sid": f.sid("WA"), "timeout_activity_sid": f.sid("WA"),
			"date_created": trStamp, "date_updated": trStamp}
		if msg := applyForm(o, form, trWorkspaceFields); msg != "" {
			ferr(w, 400, msg)
			return
		}
		f.put("trws", testAccount, id, o)
		fwrite(w, 201, o)
		return
	}
	o, ok := f.objs[key("trws", testAccount, sid)]
	if !ok {
		notFound()
		return
	}
	switch r.Method {
	case "GET":
		fwrite(w, 200, o)
	case "POST":
		if msg := applyForm(o, form, trWorkspaceFields); msg != "" {
			ferr(w, 400, msg)
			return
		}
		o["date_updated"] = trUpdatedStamp
		fwrite(w, 200, o)
	case "DELETE":
		for k, c := range f.objs {
			if c["workspace_sid"] == sid {
				delete(f.objs, k)
			}
		}
		f.del("trws", testAccount, sid)
		w.WriteHeader(204)
	default:
		ferr(w, 405, "method not allowed")
	}
}

func (f *fakeTwilio) trChild(w http.ResponseWriter, r *http.Request, form url.Values, ws, coll string, spec trSpec, sid string, notFound func()) {
	if sid == "" {
		if r.Method != "POST" {
			ferr(w, 405, "method not allowed")
			return
		}
		for _, req := range spec.required {
			if form.Get(req) == "" {
				ferr(w, 400, "Missing required parameter "+req)
				return
			}
		}
		id := f.sid(spec.prefix)
		o := map[string]any{"sid": id, "workspace_sid": ws, "account_sid": testAccount, "date_created": trStamp, "date_updated": trStamp}
		switch coll {
		case "Activities":
			o["available"] = true
		case "TaskChannels":
			o["channel_optimized_routing"] = false
		case "TaskQueues":
			o["target_workers"], o["max_reserved_workers"], o["task_order"] = "1==1", float64(1), "FIFO"
			o["reservation_activity_sid"], o["assignment_activity_sid"] = nil, nil
		case "Workflows":
			o["task_reservation_timeout"] = float64(120)
			o["assignment_callback_url"], o["fallback_assignment_callback_url"] = nil, nil
		case "Workers":
			o["attributes"] = "{}"
			o["activity_sid"] = f.objs[key("trws", testAccount, ws)]["default_activity_sid"]
		}
		if msg := applyForm(o, form, spec.create); msg != "" {
			ferr(w, 400, msg)
			return
		}
		f.put("trchild", testAccount, ws+"/"+id, o)
		fwrite(w, 201, o)
		return
	}
	o, ok := f.objs[key("trchild", testAccount, ws+"/"+sid)]
	if !ok || !strings.HasPrefix(sid, spec.prefix) {
		notFound()
		return
	}
	switch r.Method {
	case "GET":
		fwrite(w, 200, o)
	case "POST":
		fields := spec.update
		if fields == nil {
			fields = spec.create
		}
		if msg := applyForm(o, form, fields); msg != "" {
			ferr(w, 400, msg)
			return
		}
		o["date_updated"] = trUpdatedStamp
		fwrite(w, 200, o)
	case "DELETE":
		f.del("trchild", testAccount, ws+"/"+sid)
		w.WriteHeader(204)
	default:
		ferr(w, 405, "method not allowed")
	}
}
