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
	"context"
	"encoding/json"
	"fmt"
	"net/url"
	"strconv"

	p "github.com/pulumi/pulumi-go-provider"
	"github.com/pulumi/pulumi-go-provider/infer"
)

// EventSubscription manages a Twilio Event Streams subscription: the set of event types delivered to a sink.
type EventSubscription struct{}

// EventSubscriptionType selects one event type, optionally pinned to a schema version.
type EventSubscriptionType struct {
	Type          string `pulumi:"type"`
	SchemaVersion *int   `pulumi:"schemaVersion,optional"`
}

func (t *EventSubscriptionType) Annotate(a infer.Annotator) {
	a.Describe(&t.Type, "Event type ID, for example com.twilio.messaging.message.delivered.")
	a.Describe(&t.SchemaVersion, "Schema version of the event type to receive. Leave it unset to take Twilio's default (the latest version).")
}

// EventSubscriptionArgs are the inputs of an EventSubscription.
type EventSubscriptionArgs struct {
	Description string                  `pulumi:"description"`
	SinkSid     string                  `pulumi:"sinkSid" provider:"replaceOnChanges"`
	Types       []EventSubscriptionType `pulumi:"types"`
}

func (a *EventSubscriptionArgs) Annotate(an infer.Annotator) {
	an.Describe(&a.Description, "Human-readable description of the subscription. Twilio asks that it contains no personal data. Updates in place.")
	an.Describe(&a.SinkSid, "SID (DG...) of the sink the events are sent to. The sink must be active when the subscription is created. "+
		"Twilio cannot move a subscription to another sink, so changing it replaces the subscription.")
	an.Describe(&a.Types, "The event types to deliver, at least one. Adding, removing or re-versioning a type updates the subscription in place.")
}

// EventSubscriptionState is the persisted state of an EventSubscription.
type EventSubscriptionState struct {
	EventSubscriptionArgs

	// Sid is the subscription SID (DF...) and the resource ID.
	Sid string `pulumi:"sid"`
	// AccountSid is the account that owns the subscription.
	AccountSid string `pulumi:"accountSid"`
	// DateCreated and DateUpdated are ISO 8601 timestamps.
	DateCreated string `pulumi:"dateCreated"`
	DateUpdated string `pulumi:"dateUpdated"`
}

func (s *EventSubscriptionState) Annotate(a infer.Annotator) {
	a.Describe(&s.Sid, "The subscription SID (DF...).")
	a.Describe(&s.AccountSid, "SID of the account that owns the subscription.")
	a.Describe(&s.DateCreated, "When the subscription was created (ISO 8601).")
	a.Describe(&s.DateUpdated, "When the subscription was last updated (ISO 8601).")
}

func (r *EventSubscription) Annotate(a infer.Annotator) {
	a.Describe(&r, "A Twilio Event Streams subscription: which event types are delivered to an EventSink. The event types are "+
		"managed one by one through Twilio's SubscribedEvents endpoints, so changing them never recreates the subscription.")
}

var _ infer.CustomResource[EventSubscriptionArgs, EventSubscriptionState] = (*EventSubscription)(nil)
var _ infer.CustomCheck[EventSubscriptionArgs] = (*EventSubscription)(nil)
var _ infer.CustomRead[EventSubscriptionArgs, EventSubscriptionState] = (*EventSubscription)(nil)
var _ infer.CustomUpdate[EventSubscriptionArgs, EventSubscriptionState] = (*EventSubscription)(nil)
var _ infer.CustomDelete[EventSubscriptionState] = (*EventSubscription)(nil)

const eventSubscriptionsPath = "/v1/Subscriptions"

func eventSubscriptionPath(sid string) string {
	return eventSubscriptionsPath + "/" + url.PathEscape(sid)
}

func subscribedEventsPath(sid string) string { return eventSubscriptionPath(sid) + "/SubscribedEvents" }

func (r *EventSubscription) Check(ctx context.Context, req infer.CheckRequest) (infer.CheckResponse[EventSubscriptionArgs], error) {
	args, failures, err := infer.DefaultCheck[EventSubscriptionArgs](ctx, req.NewInputs)
	if err != nil || len(failures) > 0 {
		return infer.CheckResponse[EventSubscriptionArgs]{Inputs: args, Failures: failures}, err
	}
	unk := unknowns(req.NewInputs)
	if !unk("description") && args.Description == "" {
		failures = append(failures, p.CheckFailure{Property: "description", Reason: "description must not be empty"})
	}
	if !unk("sinkSid") && args.SinkSid == "" {
		failures = append(failures, p.CheckFailure{Property: "sinkSid", Reason: "sinkSid must not be empty"})
	}
	if !unk("types") {
		if len(args.Types) == 0 {
			failures = append(failures, p.CheckFailure{Property: "types", Reason: "types needs at least one event type"})
		}
		seen := map[string]bool{}
		for i, t := range args.Types {
			if t.Type == "" {
				failures = append(failures, p.CheckFailure{Property: fmt.Sprintf("types[%d].type", i), Reason: "type must not be empty"})
			} else if seen[t.Type] {
				failures = append(failures, p.CheckFailure{Property: fmt.Sprintf("types[%d].type", i), Reason: "event type " + t.Type + " is listed more than once"})
			}
			seen[t.Type] = true
			if t.SchemaVersion != nil && *t.SchemaVersion < 1 {
				failures = append(failures, p.CheckFailure{Property: fmt.Sprintf("types[%d].schemaVersion", i), Reason: "schemaVersion must be 1 or greater"})
			}
		}
	}
	return infer.CheckResponse[EventSubscriptionArgs]{Inputs: args, Failures: failures}, nil
}

// subscribedEvent is one event type of a subscription as Twilio reports it.
type subscribedEvent struct {
	Type          string
	SchemaVersion *int
}

func typeSelectorJSON(ts []EventSubscriptionType) string {
	out := make([]map[string]any, 0, len(ts))
	for _, t := range ts {
		m := map[string]any{"type": t.Type}
		if t.SchemaVersion != nil {
			m["schema_version"] = *t.SchemaVersion
		}
		out = append(out, m)
	}
	b, _ := json.Marshal(out)
	return string(b)
}

// listSubscribedEvents reads every event type of a subscription, following the pagination.
func listSubscribedEvents(ctx context.Context, c *Client, sid string) ([]subscribedEvent, error) {
	var out []subscribedEvent
	query := url.Values{"PageSize": {"100"}}
	for page := 0; page < 100; page++ {
		o, err := c.do(ctx, "GET", "events", subscribedEventsPath(sid), query, nil)
		if err != nil {
			return nil, err
		}
		items, _ := o["types"].([]any)
		for _, it := range items {
			if m, ok := it.(map[string]any); ok {
				e := obj(m)
				out = append(out, subscribedEvent{Type: e.str("type"), SchemaVersion: e.optInt("schema_version")})
			}
		}
		next := o.str("meta.next_page_url")
		if next == "" {
			return out, nil
		}
		u, err := url.Parse(next)
		if err != nil {
			return nil, fmt.Errorf("parse next page URL: %w", err)
		}
		query = u.Query()
	}
	return nil, fmt.Errorf("subscribed events of %s: more than 100 pages", sid)
}

// reconcileTypes makes the subscription's event types match want: it adds new ones before removing
// old ones, so the subscription is never left without a type.
func reconcileTypes(ctx context.Context, c *Client, sid string, want []EventSubscriptionType) error {
	have, err := listSubscribedEvents(ctx, c, sid)
	if err != nil {
		return err
	}
	cur := map[string]subscribedEvent{}
	for _, h := range have {
		cur[h.Type] = h
	}
	wanted := map[string]bool{}
	for _, w := range want {
		wanted[w.Type] = true
		h, ok := cur[w.Type]
		switch {
		case !ok:
			form := url.Values{"Type": {w.Type}}
			setI(form, "SchemaVersion", w.SchemaVersion)
			if _, err := c.do(ctx, "POST", "events", subscribedEventsPath(sid), nil, form); err != nil {
				return fmt.Errorf("add event type %s: %w", w.Type, err)
			}
		case w.SchemaVersion != nil && (h.SchemaVersion == nil || *h.SchemaVersion != *w.SchemaVersion):
			form := url.Values{"SchemaVersion": {strconv.Itoa(*w.SchemaVersion)}}
			if _, err := c.do(ctx, "POST", "events", subscribedEventsPath(sid)+"/"+url.PathEscape(w.Type), nil, form); err != nil {
				return fmt.Errorf("update event type %s: %w", w.Type, err)
			}
		}
	}
	for _, h := range have {
		if !wanted[h.Type] {
			if _, err := c.do(ctx, "DELETE", "events", subscribedEventsPath(sid)+"/"+url.PathEscape(h.Type), nil, nil); err != nil && !isNotFound(err) {
				return fmt.Errorf("remove event type %s: %w", h.Type, err)
			}
		}
	}
	return nil
}

// refreshTypes returns the event types Twilio reports in the program's order, followed by types added
// outside Pulumi. A version is reported only for a type the program pinned, unless imported.
func refreshTypes(prog []EventSubscriptionType, have []subscribedEvent, imported bool) []EventSubscriptionType {
	pinned := map[string]bool{}
	for _, t := range prog {
		pinned[t.Type] = t.SchemaVersion != nil
	}
	byType := map[string]subscribedEvent{}
	for _, h := range have {
		byType[h.Type] = h
	}
	out := []EventSubscriptionType{}
	used := map[string]bool{}
	add := func(h subscribedEvent) {
		t := EventSubscriptionType{Type: h.Type}
		if imported || pinned[h.Type] {
			t.SchemaVersion = h.SchemaVersion
		}
		out = append(out, t)
		used[h.Type] = true
	}
	for _, t := range prog {
		if h, ok := byType[t.Type]; ok {
			add(h)
		}
	}
	for _, h := range have {
		if !used[h.Type] {
			add(h)
		}
	}
	return out
}

func eventSubscriptionOutputs(st *EventSubscriptionState, o obj) {
	st.AccountSid = o.str("account_sid")
	st.DateCreated = o.str("date_created")
	st.DateUpdated = o.str("date_updated")
}

func (r *EventSubscription) Create(ctx context.Context, req infer.CreateRequest[EventSubscriptionArgs]) (infer.CreateResponse[EventSubscriptionState], error) {
	a := req.Inputs
	if req.DryRun {
		return infer.CreateResponse[EventSubscriptionState]{Output: EventSubscriptionState{EventSubscriptionArgs: a}}, nil
	}
	c, err := clientFor(ctx)
	if err != nil {
		return infer.CreateResponse[EventSubscriptionState]{}, err
	}
	form := url.Values{"Description": {a.Description}, "SinkSid": {a.SinkSid}, "Types": {typeSelectorJSON(a.Types)}}
	o, err := c.do(ctx, "POST", "events", eventSubscriptionsPath, nil, form)
	if err != nil {
		return infer.CreateResponse[EventSubscriptionState]{}, fmt.Errorf("create event subscription: %w", err)
	}
	st := EventSubscriptionState{EventSubscriptionArgs: a, Sid: o.str("sid")}
	eventSubscriptionOutputs(&st, o)
	return infer.CreateResponse[EventSubscriptionState]{ID: st.Sid, Output: st}, nil
}

func (r *EventSubscription) Read(ctx context.Context, req infer.ReadRequest[EventSubscriptionArgs, EventSubscriptionState]) (infer.ReadResponse[EventSubscriptionArgs, EventSubscriptionState], error) {
	c, err := clientFor(ctx)
	if err != nil {
		return infer.ReadResponse[EventSubscriptionArgs, EventSubscriptionState]{}, err
	}
	o, err := c.do(ctx, "GET", "events", eventSubscriptionPath(req.ID), nil, nil)
	if err != nil {
		if isNotFound(err) {
			if err := notFoundOnImport("event subscription", req.ID, req.Inputs, req.State); err != nil {
				return infer.ReadResponse[EventSubscriptionArgs, EventSubscriptionState]{}, err
			}
			return infer.ReadResponse[EventSubscriptionArgs, EventSubscriptionState]{}, nil
		}
		return infer.ReadResponse[EventSubscriptionArgs, EventSubscriptionState]{}, fmt.Errorf("read event subscription %s: %w", req.ID, err)
	}
	have, err := listSubscribedEvents(ctx, c, req.ID)
	if err != nil {
		if isNotFound(err) {
			if err := notFoundOnImport("event subscription", req.ID, req.Inputs, req.State); err != nil {
				return infer.ReadResponse[EventSubscriptionArgs, EventSubscriptionState]{}, err
			}
			return infer.ReadResponse[EventSubscriptionArgs, EventSubscriptionState]{}, nil
		}
		return infer.ReadResponse[EventSubscriptionArgs, EventSubscriptionState]{}, fmt.Errorf("read event types of subscription %s: %w", req.ID, err)
	}
	in := req.Inputs
	imported := in.SinkSid == ""
	in.Description = o.str("description")
	in.SinkSid = o.str("sink_sid")
	in.Types = refreshTypes(in.Types, have, imported)
	st := EventSubscriptionState{EventSubscriptionArgs: in, Sid: req.ID}
	eventSubscriptionOutputs(&st, o)
	return infer.ReadResponse[EventSubscriptionArgs, EventSubscriptionState]{ID: req.ID, Inputs: in, State: st}, nil
}

func (r *EventSubscription) Update(ctx context.Context, req infer.UpdateRequest[EventSubscriptionArgs, EventSubscriptionState]) (infer.UpdateResponse[EventSubscriptionState], error) {
	st := req.State
	st.EventSubscriptionArgs = req.Inputs
	if req.DryRun {
		return infer.UpdateResponse[EventSubscriptionState]{Output: st}, nil
	}
	c, err := clientFor(ctx)
	if err != nil {
		return infer.UpdateResponse[EventSubscriptionState]{}, err
	}
	// The description first: it is idempotent. The types follow, and a failure part way leaves the state
	// as it was, so the next update repeats the same diff against what Twilio now holds.
	o, err := c.do(ctx, "POST", "events", eventSubscriptionPath(req.ID), nil, url.Values{"Description": {req.Inputs.Description}})
	if err != nil {
		return infer.UpdateResponse[EventSubscriptionState]{}, fmt.Errorf("update event subscription %s: %w", req.ID, err)
	}
	if err := reconcileTypes(ctx, c, req.ID, req.Inputs.Types); err != nil {
		return infer.UpdateResponse[EventSubscriptionState]{}, fmt.Errorf("update event subscription %s: %w", req.ID, err)
	}
	eventSubscriptionOutputs(&st, o)
	return infer.UpdateResponse[EventSubscriptionState]{Output: st}, nil
}

func (r *EventSubscription) Delete(ctx context.Context, req infer.DeleteRequest[EventSubscriptionState]) (infer.DeleteResponse, error) {
	c, err := clientFor(ctx)
	if err != nil {
		return infer.DeleteResponse{}, err
	}
	if _, err := c.do(ctx, "DELETE", "events", eventSubscriptionPath(req.ID), nil, nil); err != nil && !isNotFound(err) {
		return infer.DeleteResponse{}, fmt.Errorf("delete event subscription %s: %w", req.ID, err)
	}
	return infer.DeleteResponse{}, nil
}
