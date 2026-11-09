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
	"strings"

	p "github.com/pulumi/pulumi-go-provider"
	"github.com/pulumi/pulumi-go-provider/infer"
)

// EventSink manages a Twilio Event Streams sink: the destination (a webhook, an Amazon Kinesis
// stream or Segment) that an EventSubscription delivers events to.
type EventSink struct{}

// EventSinkArgs are the inputs of an EventSink.
type EventSinkArgs struct {
	Description string `pulumi:"description"`
	SinkType    string `pulumi:"sinkType" provider:"replaceOnChanges"`

	WebhookDestination *string `pulumi:"webhookDestination,optional" provider:"replaceOnChanges"`
	WebhookMethod      *string `pulumi:"webhookMethod,optional" provider:"replaceOnChanges"`
	WebhookBatchEvents *bool   `pulumi:"webhookBatchEvents,optional" provider:"replaceOnChanges"`

	KinesisArn        *string `pulumi:"kinesisArn,optional" provider:"replaceOnChanges"`
	KinesisRoleArn    *string `pulumi:"kinesisRoleArn,optional" provider:"replaceOnChanges"`
	KinesisExternalId *string `pulumi:"kinesisExternalId,optional" provider:"replaceOnChanges,secret"`

	SegmentWriteKey *string `pulumi:"segmentWriteKey,optional" provider:"replaceOnChanges,secret"`
}

func (a *EventSinkArgs) Annotate(an infer.Annotator) {
	an.Describe(&a.Description, "Human-readable description of the sink. Twilio asks that it contains no personal data. Updates in place.")
	an.Describe(&a.SinkType, "Kind of destination: webhook, kinesis or segment. Changing it replaces the sink.")
	an.Describe(&a.WebhookDestination, "Webhook sinks: the URL Twilio delivers events to. Required for webhook sinks. Twilio cannot change a sink's configuration, so changing it replaces the sink.")
	an.Describe(&a.WebhookMethod, "Webhook sinks: the HTTP method, GET or POST. Twilio's default is POST. Changing it replaces the sink.")
	an.Describe(&a.WebhookBatchEvents, "Webhook sinks: whether events are delivered in batches. Changing it replaces the sink. Twilio accepts true on create but its API reports false afterwards, so the provider keeps the value you set (it never reads it back) and logs a warning at create if the API does not echo it; an import adopts only true.")
	an.Describe(&a.KinesisArn, "Kinesis sinks: ARN of the Amazon Kinesis Data Stream. Required for kinesis sinks. Changing it replaces the sink.")
	an.Describe(&a.KinesisRoleArn, "Kinesis sinks: ARN of the IAM role Twilio assumes to write to the stream. Required for kinesis sinks. Changing it replaces the sink.")
	an.Describe(&a.KinesisExternalId, "Kinesis sinks: the external ID Twilio presents when assuming the role. Treated as a secret. Changing it replaces the sink.")
	an.Describe(&a.SegmentWriteKey, "Segment sinks: the Segment source write key. Required for segment sinks. A secret; Twilio does not return it, so drift in it is not detected. Changing it replaces the sink.")
}

// EventSinkState is the persisted state of an EventSink.
type EventSinkState struct {
	EventSinkArgs

	// Sid is the sink SID (DG...) and the resource ID.
	Sid string `pulumi:"sid"`
	// Status is the validation status Twilio reports.
	Status string `pulumi:"status"`
	// DateCreated and DateUpdated are ISO 8601 timestamps.
	DateCreated string `pulumi:"dateCreated"`
	DateUpdated string `pulumi:"dateUpdated"`
}

func (s *EventSinkState) Annotate(a infer.Annotator) {
	a.Describe(&s.Sid, "The sink SID (DG...).")
	a.Describe(&s.Status, "Sink status reported by Twilio: initialized, validating, active or failed. A subscription can only be created for an active sink; a new webhook sink must first be validated with Twilio's sink test and validate calls.")
	a.Describe(&s.DateCreated, "When the sink was created (ISO 8601).")
	a.Describe(&s.DateUpdated, "When the sink was last updated (ISO 8601).")
}

func (r *EventSink) Annotate(a infer.Annotator) {
	a.Describe(&r, "A Twilio Event Streams sink: where subscribed events are delivered. Twilio only lets the description of a sink change; "+
		"any change to its type or connection settings replaces the sink. Events API (events.twilio.com) credentials are the provider's account credentials.")
}

var _ infer.CustomResource[EventSinkArgs, EventSinkState] = (*EventSink)(nil)
var _ infer.CustomCheck[EventSinkArgs] = (*EventSink)(nil)
var _ infer.CustomDiff[EventSinkArgs, EventSinkState] = (*EventSink)(nil)
var _ infer.CustomRead[EventSinkArgs, EventSinkState] = (*EventSink)(nil)
var _ infer.CustomUpdate[EventSinkArgs, EventSinkState] = (*EventSink)(nil)
var _ infer.CustomDelete[EventSinkState] = (*EventSink)(nil)

const eventSinksPath = "/v1/Sinks"

func eventSinkPath(sid string) string { return eventSinksPath + "/" + url.PathEscape(sid) }

// config builds the SinkConfiguration JSON of the sink's type.
func (a *EventSinkArgs) config() map[string]any {
	cfg := map[string]any{}
	switch a.SinkType {
	case "webhook":
		cfg["destination"] = deref(a.WebhookDestination)
		if a.WebhookMethod != nil {
			cfg["method"] = *a.WebhookMethod
		}
		if a.WebhookBatchEvents != nil {
			cfg["batch_events"] = *a.WebhookBatchEvents
		}
	case "kinesis":
		cfg["arn"] = deref(a.KinesisArn)
		cfg["role_arn"] = deref(a.KinesisRoleArn)
		if a.KinesisExternalId != nil {
			cfg["external_id"] = *a.KinesisExternalId
		}
	case "segment":
		cfg["write_key"] = deref(a.SegmentWriteKey)
	}
	return cfg
}

var eventSinkTypes = map[string]bool{"webhook": true, "kinesis": true, "segment": true}

func (r *EventSink) Check(ctx context.Context, req infer.CheckRequest) (infer.CheckResponse[EventSinkArgs], error) {
	args, failures, err := infer.DefaultCheck[EventSinkArgs](ctx, req.NewInputs)
	if err != nil || len(failures) > 0 {
		return infer.CheckResponse[EventSinkArgs]{Inputs: args, Failures: failures}, err
	}
	unk := unknowns(req.NewInputs)
	fail := func(prop, reason string) { failures = append(failures, p.CheckFailure{Property: prop, Reason: reason}) }
	if !unk("description") && args.Description == "" {
		fail("description", "description must not be empty")
	}
	if unk("sinkType") {
		return infer.CheckResponse[EventSinkArgs]{Inputs: args, Failures: failures}, nil
	}
	if !eventSinkTypes[args.SinkType] {
		fail("sinkType", "sinkType must be one of webhook, kinesis or segment")
		return infer.CheckResponse[EventSinkArgs]{Inputs: args, Failures: failures}, nil
	}
	// Each type needs its own settings and rejects the others', so a typo cannot silently send an empty configuration.
	fields := []struct {
		name, sinkType string
		set, required  bool
	}{
		{"webhookDestination", "webhook", args.WebhookDestination != nil, true},
		{"webhookMethod", "webhook", args.WebhookMethod != nil, false},
		{"webhookBatchEvents", "webhook", args.WebhookBatchEvents != nil, false},
		{"kinesisArn", "kinesis", args.KinesisArn != nil, true},
		{"kinesisRoleArn", "kinesis", args.KinesisRoleArn != nil, true},
		{"kinesisExternalId", "kinesis", args.KinesisExternalId != nil, false},
		{"segmentWriteKey", "segment", args.SegmentWriteKey != nil, true},
	}
	for _, f := range fields {
		switch {
		case f.sinkType != args.SinkType && f.set:
			fail(f.name, fmt.Sprintf("%s applies to %s sinks, not %s", f.name, f.sinkType, args.SinkType))
		case f.sinkType == args.SinkType && f.required && !f.set && !unk(f.name):
			fail(f.name, fmt.Sprintf("%s is required for %s sinks", f.name, f.sinkType))
		}
	}
	failures = checkMethods(failures, unk, map[string]*string{"webhookMethod": args.WebhookMethod})
	return infer.CheckResponse[EventSinkArgs]{Inputs: args, Failures: failures}, nil
}

// Diff replaces the sink when a connection setting changes, but never for a setting an import cannot
// know: Twilio does not return the Segment write key (and may omit the Kinesis external ID), so an
// imported sink has none recorded and a program that sets one is not a change; an unset webhook
// method is POST and an unset batch flag is false, so stating the default is not a change either.
// Replacing would destroy the sink and leave its subscriptions on an initialized, unusable one.
func (r *EventSink) Diff(_ context.Context, req infer.DiffRequest[EventSinkArgs, EventSinkState]) (infer.DiffResponse, error) {
	o, n := req.State.EventSinkArgs, req.Inputs
	detail := map[string]p.PropertyDiff{}
	replace := func(prop string, changed bool) {
		if changed {
			detail[prop] = p.PropertyDiff{Kind: p.UpdateReplace}
		}
	}
	if o.Description != n.Description {
		detail["description"] = p.PropertyDiff{Kind: p.Update}
	}
	replace("sinkType", o.SinkType != n.SinkType)
	replace("webhookDestination", deref(o.WebhookDestination) != deref(n.WebhookDestination))
	replace("webhookMethod", !strings.EqualFold(derefOr(o.WebhookMethod, "POST"), derefOr(n.WebhookMethod, "POST")))
	replace("webhookBatchEvents", derefBool(o.WebhookBatchEvents) != derefBool(n.WebhookBatchEvents))
	replace("kinesisArn", deref(o.KinesisArn) != deref(n.KinesisArn))
	replace("kinesisRoleArn", deref(o.KinesisRoleArn) != deref(n.KinesisRoleArn))
	replace("kinesisExternalId", o.KinesisExternalId != nil && n.KinesisExternalId != nil && *o.KinesisExternalId != *n.KinesisExternalId)
	replace("segmentWriteKey", o.SegmentWriteKey != nil && n.SegmentWriteKey != nil && *o.SegmentWriteKey != *n.SegmentWriteKey)
	return infer.DiffResponse{HasChanges: len(detail) > 0, DetailedDiff: detail}, nil
}

func derefBool(b *bool) bool { return b != nil && *b }

func eventSinkOutputs(st *EventSinkState, o obj) {
	st.Status = o.str("status")
	st.DateCreated = o.str("date_created")
	st.DateUpdated = o.str("date_updated")
}

// refresh copies what Twilio reports into the inputs the program set (drift) and, on an import, adopts
// every returned setting. The Segment write key and the Kinesis external ID are only read when returned.
func (a *EventSinkArgs) refresh(o obj, imported bool) {
	a.Description = o.str("description")
	if imported {
		a.SinkType = o.str("sink_type")
	}
	str := []struct {
		dst **string
		key string
	}{
		{&a.WebhookDestination, "sink_configuration.destination"}, {&a.WebhookMethod, "sink_configuration.method"},
		{&a.KinesisArn, "sink_configuration.arn"}, {&a.KinesisRoleArn, "sink_configuration.role_arn"},
		{&a.KinesisExternalId, "sink_configuration.external_id"}, {&a.SegmentWriteKey, "sink_configuration.write_key"},
	}
	for _, f := range str {
		v := nonEmpty(o.optStr(f.key))
		if v != nil && f.key == "sink_configuration.method" {
			// Twilio returns the method in lower case ("post") whatever was sent.
			up := strings.ToUpper(*v)
			v = &up
		}
		switch {
		case imported && v != nil && !(f.key == "sink_configuration.method" && strings.EqualFold(*v, "POST")):
			*f.dst = v
		case !imported && *f.dst != nil && v != nil:
			*f.dst = v
		}
	}
	// Twilio accepts batch_events=true on create but reports false afterwards, so a refresh never overwrites
	// the recorded input (that would plan a replace on every run); an import adopts only an explicit true.
	if b := o.optBool("sink_configuration.batch_events"); b != nil && imported && *b {
		a.WebhookBatchEvents = b
	}
}

func (r *EventSink) Create(ctx context.Context, req infer.CreateRequest[EventSinkArgs]) (infer.CreateResponse[EventSinkState], error) {
	a := req.Inputs
	if req.DryRun {
		return infer.CreateResponse[EventSinkState]{Output: EventSinkState{EventSinkArgs: a}}, nil
	}
	c, err := clientFor(ctx)
	if err != nil {
		return infer.CreateResponse[EventSinkState]{}, err
	}
	cfg, _ := json.Marshal(a.config())
	form := url.Values{"Description": {a.Description}, "SinkType": {a.SinkType}, "SinkConfiguration": {string(cfg)}}
	o, err := c.do(ctx, "POST", "events", eventSinksPath, nil, form)
	if err != nil {
		return infer.CreateResponse[EventSinkState]{}, fmt.Errorf("create event sink: %w", err)
	}
	if want := derefBool(a.WebhookBatchEvents); want && !derefBool(o.optBool("sink_configuration.batch_events")) {
		p.GetLogger(ctx).Warningf("Twilio did not report webhookBatchEvents=true for event sink %s: the API accepts the flag but returns false, so batching may not be applied. The program's value is kept so later runs do not plan a replacement.", o.str("sid"))
	}
	st := EventSinkState{EventSinkArgs: a, Sid: o.str("sid")}
	eventSinkOutputs(&st, o)
	return infer.CreateResponse[EventSinkState]{ID: st.Sid, Output: st}, nil
}

func (r *EventSink) Read(ctx context.Context, req infer.ReadRequest[EventSinkArgs, EventSinkState]) (infer.ReadResponse[EventSinkArgs, EventSinkState], error) {
	c, err := clientFor(ctx)
	if err != nil {
		return infer.ReadResponse[EventSinkArgs, EventSinkState]{}, err
	}
	o, err := c.do(ctx, "GET", "events", eventSinkPath(req.ID), nil, nil)
	if err != nil {
		if isNotFound(err) {
			return infer.ReadResponse[EventSinkArgs, EventSinkState]{}, nil
		}
		return infer.ReadResponse[EventSinkArgs, EventSinkState]{}, fmt.Errorf("read event sink %s: %w", req.ID, err)
	}
	in := req.Inputs
	in.refresh(o, in.SinkType == "")
	st := EventSinkState{EventSinkArgs: in, Sid: req.ID}
	eventSinkOutputs(&st, o)
	return infer.ReadResponse[EventSinkArgs, EventSinkState]{ID: req.ID, Inputs: in, State: st}, nil
}

func (r *EventSink) Update(ctx context.Context, req infer.UpdateRequest[EventSinkArgs, EventSinkState]) (infer.UpdateResponse[EventSinkState], error) {
	st := req.State
	st.EventSinkArgs = req.Inputs
	if req.DryRun {
		return infer.UpdateResponse[EventSinkState]{Output: st}, nil
	}
	c, err := clientFor(ctx)
	if err != nil {
		return infer.UpdateResponse[EventSinkState]{}, err
	}
	o, err := c.do(ctx, "POST", "events", eventSinkPath(req.ID), nil, url.Values{"Description": {req.Inputs.Description}})
	if err != nil {
		return infer.UpdateResponse[EventSinkState]{}, fmt.Errorf("update event sink %s: %w", req.ID, err)
	}
	eventSinkOutputs(&st, o)
	return infer.UpdateResponse[EventSinkState]{Output: st}, nil
}

func (r *EventSink) Delete(ctx context.Context, req infer.DeleteRequest[EventSinkState]) (infer.DeleteResponse, error) {
	c, err := clientFor(ctx)
	if err != nil {
		return infer.DeleteResponse{}, err
	}
	if _, err := c.do(ctx, "DELETE", "events", eventSinkPath(req.ID), nil, nil); err != nil && !isNotFound(err) {
		return infer.DeleteResponse{}, fmt.Errorf("delete event sink %s: %w", req.ID, err)
	}
	return infer.DeleteResponse{}, nil
}
