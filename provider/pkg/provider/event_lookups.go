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

	"github.com/pulumi/pulumi-go-provider/infer"
)

// GetEventSink looks up an Event Streams sink by SID.
type GetEventSink struct{}

// GetEventSinkArgs are the lookup arguments.
type GetEventSinkArgs struct {
	Sid string `pulumi:"sid"`
}

func (a *GetEventSinkArgs) Annotate(an infer.Annotator) {
	an.Describe(&a.Sid, "SID (DG...) of the sink.")
}

// GetEventSinkResult is the looked-up sink. Secret settings (the Segment write key, the Kinesis external ID) are never returned.
type GetEventSinkResult struct {
	Sid                string  `pulumi:"sid"`
	Description        string  `pulumi:"description"`
	SinkType           string  `pulumi:"sinkType"`
	Status             string  `pulumi:"status"`
	DateCreated        string  `pulumi:"dateCreated"`
	DateUpdated        string  `pulumi:"dateUpdated"`
	WebhookDestination *string `pulumi:"webhookDestination,optional"`
	WebhookMethod      *string `pulumi:"webhookMethod,optional"`
	WebhookBatchEvents *bool   `pulumi:"webhookBatchEvents,optional"`
	KinesisArn         *string `pulumi:"kinesisArn,optional"`
	KinesisRoleArn     *string `pulumi:"kinesisRoleArn,optional"`
}

func (a *GetEventSinkResult) Annotate(an infer.Annotator) {
	an.Describe(&a.Sid, "The sink SID (DG...).")
	an.Describe(&a.Description, "Description of the sink.")
	an.Describe(&a.SinkType, "Kind of destination: webhook, kinesis, segment or email.")
	an.Describe(&a.Status, "Sink status: initialized, validating, active or failed.")
	an.Describe(&a.DateCreated, "When the sink was created (ISO 8601).")
	an.Describe(&a.DateUpdated, "When the sink was last updated (ISO 8601).")
	an.Describe(&a.WebhookDestination, "Webhook sinks: the URL events are delivered to.")
	an.Describe(&a.WebhookMethod, "Webhook sinks: the HTTP method.")
	an.Describe(&a.WebhookBatchEvents, "Webhook sinks: whether events are batched.")
	an.Describe(&a.KinesisArn, "Kinesis sinks: ARN of the stream.")
	an.Describe(&a.KinesisRoleArn, "Kinesis sinks: ARN of the role Twilio assumes.")
}

func (f *GetEventSink) Annotate(a infer.Annotator) {
	a.Describe(&f, "Look up a Twilio Event Streams sink by SID.")
}

func (f *GetEventSink) Invoke(ctx context.Context, req infer.FunctionRequest[GetEventSinkArgs]) (infer.FunctionResponse[GetEventSinkResult], error) {
	const fn = "getEventSink"
	if err := requireArg(fn, "sid", req.Input.Sid); err != nil {
		return infer.FunctionResponse[GetEventSinkResult]{}, err
	}
	c, err := clientFor(ctx)
	if err != nil {
		return infer.FunctionResponse[GetEventSinkResult]{}, err
	}
	o, err := c.do(ctx, "GET", "events", eventSinkPath(req.Input.Sid), nil, nil)
	if err != nil {
		return infer.FunctionResponse[GetEventSinkResult]{}, lookupFailure(fn, "event sink "+req.Input.Sid, err)
	}
	return infer.FunctionResponse[GetEventSinkResult]{Output: GetEventSinkResult{
		Sid: o.str("sid"), Description: o.str("description"), SinkType: o.str("sink_type"), Status: o.str("status"),
		DateCreated: o.str("date_created"), DateUpdated: o.str("date_updated"),
		WebhookDestination: nonEmpty(o.optStr("sink_configuration.destination")), WebhookMethod: nonEmpty(o.optStr("sink_configuration.method")),
		WebhookBatchEvents: o.optBool("sink_configuration.batch_events"),
		KinesisArn:         nonEmpty(o.optStr("sink_configuration.arn")), KinesisRoleArn: nonEmpty(o.optStr("sink_configuration.role_arn")),
	}}, nil
}

// GetEventSubscription looks up an Event Streams subscription by SID.
type GetEventSubscription struct{}

// GetEventSubscriptionArgs are the lookup arguments.
type GetEventSubscriptionArgs struct {
	Sid string `pulumi:"sid"`
}

func (a *GetEventSubscriptionArgs) Annotate(an infer.Annotator) {
	an.Describe(&a.Sid, "SID (DF...) of the subscription.")
}

// GetEventSubscriptionResult is the looked-up subscription with its event types.
type GetEventSubscriptionResult struct {
	Sid         string                  `pulumi:"sid"`
	AccountSid  string                  `pulumi:"accountSid"`
	Description string                  `pulumi:"description"`
	SinkSid     string                  `pulumi:"sinkSid"`
	DateCreated string                  `pulumi:"dateCreated"`
	DateUpdated string                  `pulumi:"dateUpdated"`
	Types       []EventSubscriptionType `pulumi:"types"`
}

func (a *GetEventSubscriptionResult) Annotate(an infer.Annotator) {
	an.Describe(&a.Sid, "The subscription SID (DF...).")
	an.Describe(&a.AccountSid, "SID of the account that owns the subscription.")
	an.Describe(&a.Description, "Description of the subscription.")
	an.Describe(&a.SinkSid, "SID of the sink the events are sent to.")
	an.Describe(&a.DateCreated, "When the subscription was created (ISO 8601).")
	an.Describe(&a.DateUpdated, "When the subscription was last updated (ISO 8601).")
	an.Describe(&a.Types, "The subscribed event types with the schema version Twilio reports for each.")
}

func (f *GetEventSubscription) Annotate(a infer.Annotator) {
	a.Describe(&f, "Look up a Twilio Event Streams subscription by SID.")
}

func (f *GetEventSubscription) Invoke(ctx context.Context, req infer.FunctionRequest[GetEventSubscriptionArgs]) (infer.FunctionResponse[GetEventSubscriptionResult], error) {
	const fn = "getEventSubscription"
	if err := requireArg(fn, "sid", req.Input.Sid); err != nil {
		return infer.FunctionResponse[GetEventSubscriptionResult]{}, err
	}
	c, err := clientFor(ctx)
	if err != nil {
		return infer.FunctionResponse[GetEventSubscriptionResult]{}, err
	}
	o, err := c.do(ctx, "GET", "events", eventSubscriptionPath(req.Input.Sid), nil, nil)
	if err != nil {
		return infer.FunctionResponse[GetEventSubscriptionResult]{}, lookupFailure(fn, "event subscription "+req.Input.Sid, err)
	}
	have, err := listSubscribedEvents(ctx, c, req.Input.Sid)
	if err != nil {
		return infer.FunctionResponse[GetEventSubscriptionResult]{}, lookupFailure(fn, "event types of subscription "+req.Input.Sid, err)
	}
	return infer.FunctionResponse[GetEventSubscriptionResult]{Output: GetEventSubscriptionResult{
		Sid: o.str("sid"), AccountSid: o.str("account_sid"), Description: o.str("description"), SinkSid: o.str("sink_sid"),
		DateCreated: o.str("date_created"), DateUpdated: o.str("date_updated"), Types: refreshTypes(nil, have, true),
	}}, nil
}
