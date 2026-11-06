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
	"encoding/json"
	"net/http"
	"net/url"
	"sort"
	"strconv"
)

// The Events v1 API (events.twilio.com), served for the Event Streams tests:
//
//	/events/v1/Sinks[/{sid}]
//	/events/v1/Subscriptions[/{sid}]
//	/events/v1/Subscriptions/{sid}/SubscribedEvents[/{type}]
//
// The shapes follow twilio-oai spec/json/twilio_events_v1.json. A sink only updates its description,
// a subscription can only be created for an active sink, and the Segment write key is never returned.

// eventsPageCap is how many items a list page holds; a test lowers it to exercise pagination.
var eventsPageCap = 100

const eventsStamp = "2026-11-02T00:00:00Z"

func (f *fakeTwilio) handleEvents(w http.ResponseWriter, r *http.Request, form url.Values, parts []string) {
	notFound := func() { ferr(w, 404, "The requested resource "+r.URL.Path+" was not found") }
	switch {
	case len(parts) == 1 && parts[0] == "Sinks":
		f.eventsCreateSink(w, r, form)
	case len(parts) == 2 && parts[0] == "Sinks":
		f.eventsSink(w, r, form, parts[1], notFound)
	case len(parts) == 1 && parts[0] == "Subscriptions":
		f.eventsCreateSubscription(w, r, form)
	case len(parts) == 2 && parts[0] == "Subscriptions":
		f.eventsSubscription(w, r, form, parts[1], notFound)
	case len(parts) >= 3 && parts[0] == "Subscriptions" && parts[2] == "SubscribedEvents" && len(parts) <= 4:
		f.eventsSubscribed(w, r, form, parts, notFound)
	default:
		notFound()
	}
}

func (f *fakeTwilio) eventsCreateSink(w http.ResponseWriter, r *http.Request, form url.Values) {
	if r.Method != "POST" {
		ferr(w, 405, "method not allowed")
		return
	}
	for _, req := range []string{"Description", "SinkType", "SinkConfiguration"} {
		if form.Get(req) == "" {
			ferr(w, 400, "Missing required parameter "+req)
			return
		}
	}
	typ := form.Get("SinkType")
	if typ != "kinesis" && typ != "webhook" && typ != "segment" && typ != "email" {
		ferr(w, 400, "Invalid SinkType "+typ)
		return
	}
	var cfg map[string]any
	if err := json.Unmarshal([]byte(form.Get("SinkConfiguration")), &cfg); err != nil {
		ferr(w, 400, "SinkConfiguration is not valid JSON")
		return
	}
	o := map[string]any{"sid": f.sid("DG"), "description": form.Get("Description"), "sink_type": typ, "sink_configuration": cfg,
		"status": "initialized", "date_created": eventsStamp, "date_updated": eventsStamp}
	f.put("evsink", testAccount, o["sid"].(string), o)
	fwrite(w, 201, eventsSinkOut(o))
}

// eventsSinkOut hides the Segment write key, which Twilio never returns.
func eventsSinkOut(o map[string]any) map[string]any {
	c := clone(o)
	if cfg, ok := o["sink_configuration"].(map[string]any); ok {
		cc := clone(cfg)
		delete(cc, "key")
		c["sink_configuration"] = cc
	}
	return c
}

func (f *fakeTwilio) eventsSink(w http.ResponseWriter, r *http.Request, form url.Values, sid string, notFound func()) {
	o, ok := f.objs[key("evsink", testAccount, sid)]
	if !ok {
		notFound()
		return
	}
	switch r.Method {
	case "GET":
		fwrite(w, 200, eventsSinkOut(o))
	case "POST":
		if form.Get("Description") == "" {
			ferr(w, 400, "Missing required parameter Description")
			return
		}
		o["description"] = form.Get("Description")
		o["date_updated"] = "2026-11-03T00:00:00Z"
		fwrite(w, 200, eventsSinkOut(o))
	case "DELETE":
		for _, s := range f.objs {
			if s["sink_sid"] == sid {
				ferr(w, 409, "Sink is in use by a subscription")
				return
			}
		}
		f.del("evsink", testAccount, sid)
		w.WriteHeader(204)
	default:
		ferr(w, 405, "method not allowed")
	}
}

func (f *fakeTwilio) eventsCreateSubscription(w http.ResponseWriter, r *http.Request, form url.Values) {
	if r.Method != "POST" {
		ferr(w, 405, "method not allowed")
		return
	}
	for _, req := range []string{"Description", "SinkSid", "Types"} {
		if form.Get(req) == "" {
			ferr(w, 400, "Missing required parameter "+req)
			return
		}
	}
	sink, ok := f.objs[key("evsink", testAccount, form.Get("SinkSid"))]
	if !ok {
		ferr(w, 404, "Sink "+form.Get("SinkSid")+" was not found")
		return
	}
	if sink["status"] != "active" {
		ferr(w, 400, "Sink must be active to create a subscription")
		return
	}
	var types []map[string]any
	if err := json.Unmarshal([]byte(form.Get("Types")), &types); err != nil || len(types) == 0 {
		ferr(w, 400, "Types must be a non-empty JSON array")
		return
	}
	events := map[string]any{}
	for _, t := range types {
		name, _ := t["type"].(string)
		if name == "" {
			ferr(w, 400, "every type needs a type")
			return
		}
		v := float64(1)
		if sv, ok := t["schema_version"].(float64); ok {
			v = sv
		}
		events[name] = v
	}
	o := map[string]any{"sid": f.sid("DF"), "account_sid": testAccount, "description": form.Get("Description"), "sink_sid": form.Get("SinkSid"),
		"date_created": eventsStamp, "date_updated": eventsStamp, "_events": events}
	f.put("evsub", testAccount, o["sid"].(string), o)
	fwrite(w, 201, eventsSubOut(o))
}

func eventsSubOut(o map[string]any) map[string]any {
	c := clone(o)
	delete(c, "_events")
	return c
}

func (f *fakeTwilio) eventsSubscription(w http.ResponseWriter, r *http.Request, form url.Values, sid string, notFound func()) {
	o, ok := f.objs[key("evsub", testAccount, sid)]
	if !ok {
		notFound()
		return
	}
	switch r.Method {
	case "GET":
		fwrite(w, 200, eventsSubOut(o))
	case "POST":
		if d := form.Get("Description"); d != "" {
			o["description"] = d
		}
		o["date_updated"] = "2026-11-03T00:00:00Z"
		fwrite(w, 200, eventsSubOut(o))
	case "DELETE":
		f.del("evsub", testAccount, sid)
		w.WriteHeader(204)
	default:
		ferr(w, 405, "method not allowed")
	}
}

func (f *fakeTwilio) eventsSubscribed(w http.ResponseWriter, r *http.Request, form url.Values, parts []string, notFound func()) {
	o, ok := f.objs[key("evsub", testAccount, parts[1])]
	if !ok {
		notFound()
		return
	}
	events := o["_events"].(map[string]any)
	item := func(name string) map[string]any {
		return map[string]any{"account_sid": testAccount, "type": name, "schema_version": events[name], "subscription_sid": parts[1]}
	}
	if len(parts) == 3 {
		switch r.Method {
		case "GET":
			names := make([]string, 0, len(events))
			for n := range events {
				names = append(names, n)
			}
			sort.Strings(names)
			start, _ := strconv.Atoi(r.URL.Query().Get("PageToken"))
			end := start + eventsPageCap
			if end > len(names) {
				end = len(names)
			}
			items := []any{}
			for _, n := range names[start:end] {
				items = append(items, item(n))
			}
			meta := map[string]any{"key": "types", "next_page_url": nil}
			if end < len(names) {
				meta["next_page_url"] = "https://events.twilio.com" + r.URL.Path + "?PageSize=100&PageToken=" + strconv.Itoa(end)
			}
			fwrite(w, 200, map[string]any{"types": items, "meta": meta})
		case "POST":
			name := form.Get("Type")
			if name == "" {
				ferr(w, 400, "Missing required parameter Type")
				return
			}
			if _, dup := events[name]; dup {
				ferr(w, 409, "Type already subscribed")
				return
			}
			v := float64(1)
			if n, err := strconv.Atoi(form.Get("SchemaVersion")); err == nil {
				v = float64(n)
			}
			events[name] = v
			fwrite(w, 201, item(name))
		default:
			ferr(w, 405, "method not allowed")
		}
		return
	}
	name := parts[3]
	if _, ok := events[name]; !ok {
		notFound()
		return
	}
	switch r.Method {
	case "GET":
		fwrite(w, 200, item(name))
	case "POST":
		n, err := strconv.Atoi(form.Get("SchemaVersion"))
		if err != nil {
			ferr(w, 400, "SchemaVersion must be an integer")
			return
		}
		events[name] = float64(n)
		fwrite(w, 200, item(name))
	case "DELETE":
		if len(events) == 1 {
			ferr(w, 400, "A subscription needs at least one type")
			return
		}
		delete(events, name)
		w.WriteHeader(204)
	default:
		ferr(w, 405, "method not allowed")
	}
}

// eventsSetTypes replaces the event types of a subscription behind the provider's back.
func (f *fakeTwilio) eventsSetTypes(sid string, types map[string]float64) {
	f.mutate("evsub", testAccount, sid, func(o map[string]any) {
		m := map[string]any{}
		for k, v := range types {
			m[k] = v
		}
		o["_events"] = m
	})
}

func (f *fakeTwilio) eventsTypes(sid string) map[string]float64 {
	out := map[string]float64{}
	for k, v := range f.get("evsub", testAccount, sid)["_events"].(map[string]any) {
		out[k] = v.(float64)
	}
	return out
}
