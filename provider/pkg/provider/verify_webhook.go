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
	"fmt"
	"net/url"

	p "github.com/pulumi/pulumi-go-provider"
	"github.com/pulumi/pulumi-go-provider/infer"
)

// VerifyWebhook: the webhooks of a Verify service (verify.twilio.com v2).

// ---------------------------------------------------------------- VerifyWebhook

// VerifyWebhook manages a webhook of a Verify service: Twilio posts the listed factor and challenge events to the URL.
type VerifyWebhook struct{}

// VerifyWebhookArgs are the inputs of a VerifyWebhook. The webhook version is called webhookVersion
// because version is reserved in the Pulumi resource model.
type VerifyWebhookArgs struct {
	ServiceSid     string   `pulumi:"serviceSid" provider:"replaceOnChanges"`
	FriendlyName   string   `pulumi:"friendlyName"`
	EventTypes     []string `pulumi:"eventTypes"`
	WebhookUrl     string   `pulumi:"webhookUrl"`
	Status         *string  `pulumi:"status,optional"`
	WebhookVersion *string  `pulumi:"webhookVersion,optional"`
}

func (a *VerifyWebhookArgs) Annotate(an infer.Annotator) {
	an.Describe(&a.ServiceSid, "SID (VA...) of the Verify service. Changing it replaces the resource.")
	an.Describe(&a.FriendlyName, "Name of the webhook. It must not contain PII.")
	an.Describe(&a.EventTypes, "Events the webhook is subscribed to: * for all, or factor.deleted, factor.created, factor.verified, challenge.approved, challenge.denied. The order does not matter.")
	an.Describe(&a.WebhookUrl, "URL Twilio posts the events to.")
	an.Describe(&a.Status, "Webhook status: enabled (Twilio's default) or disabled. Removing it from the program resets it to enabled.")
	an.Describe(&a.WebhookVersion, "Webhook payload version: v2 (Twilio's default, with all the latest fields) or v1 (legacy, may be removed). Removing it from the program resets it to v2.")
}

// VerifyWebhookState is the persisted state of a VerifyWebhook.
type VerifyWebhookState struct {
	VerifyWebhookArgs

	// Sid is the webhook SID (YW...).
	Sid string `pulumi:"sid"`
	// WebhookMethod is the HTTP method Twilio uses to call the webhook URL.
	WebhookMethod string `pulumi:"webhookMethod"`
	DateCreated   string `pulumi:"dateCreated"`
	DateUpdated   string `pulumi:"dateUpdated"`
}

func (s *VerifyWebhookState) Annotate(a infer.Annotator) {
	a.Describe(&s.Sid, "The webhook SID (YW...).")
	a.Describe(&s.WebhookMethod, "The HTTP method Twilio uses to call the webhook URL (GET or POST), as Twilio reports it.")
	a.Describe(&s.DateCreated, "When the webhook was created (ISO 8601).")
	a.Describe(&s.DateUpdated, "When the webhook was last updated (ISO 8601).")
}

func (r *VerifyWebhook) Annotate(a infer.Annotator) {
	a.Describe(&r, "A webhook of a Verify service that receives factor and challenge events. The resource ID is <serviceSid>/<sid>.")
}

var _ infer.CustomResource[VerifyWebhookArgs, VerifyWebhookState] = (*VerifyWebhook)(nil)
var _ infer.CustomCheck[VerifyWebhookArgs] = (*VerifyWebhook)(nil)
var _ infer.CustomDiff[VerifyWebhookArgs, VerifyWebhookState] = (*VerifyWebhook)(nil)
var _ infer.CustomRead[VerifyWebhookArgs, VerifyWebhookState] = (*VerifyWebhook)(nil)
var _ infer.CustomUpdate[VerifyWebhookArgs, VerifyWebhookState] = (*VerifyWebhook)(nil)
var _ infer.CustomDelete[VerifyWebhookState] = (*VerifyWebhook)(nil)

func (r *VerifyWebhook) Check(ctx context.Context, req infer.CheckRequest) (infer.CheckResponse[VerifyWebhookArgs], error) {
	args, failures, err := infer.DefaultCheck[VerifyWebhookArgs](ctx, req.NewInputs)
	if err != nil || len(failures) > 0 {
		return infer.CheckResponse[VerifyWebhookArgs]{Inputs: args, Failures: failures}, err
	}
	c := &checker{unk: unknowns(req.NewInputs)}
	c.sid("serviceSid", args.ServiceSid, verifyServiceSidRE, "Verify service (VA followed by 32 hex characters)")
	if !c.unk("friendlyName") && args.FriendlyName == "" {
		c.fail("friendlyName", "friendlyName must not be empty")
	}
	if !c.unk("webhookUrl") && args.WebhookUrl == "" {
		c.fail("webhookUrl", "webhookUrl must not be empty")
	}
	if !c.unk("eventTypes") {
		if len(args.EventTypes) == 0 {
			c.fail("eventTypes", "eventTypes must list at least one event type")
		}
		for _, e := range args.EventTypes {
			if e == "" {
				c.fail("eventTypes", "eventTypes must not contain an empty event type")
				break
			}
		}
	}
	if v := args.Status; v != nil && !c.unk("status") && *v != "enabled" && *v != "disabled" {
		c.fail("status", "status must be enabled or disabled")
	}
	if v := args.WebhookVersion; v != nil && !c.unk("webhookVersion") && *v != "v1" && *v != "v2" {
		c.fail("webhookVersion", "webhookVersion must be v1 or v2")
	}
	return infer.CheckResponse[VerifyWebhookArgs]{Inputs: args, Failures: c.failures}, nil
}

// Diff treats an unset status or version as Twilio's default and ignores the order of the event
// types, so an imported webhook and a program that restates the defaults agree.
func (r *VerifyWebhook) Diff(_ context.Context, req infer.DiffRequest[VerifyWebhookArgs, VerifyWebhookState]) (infer.DiffResponse, error) {
	o, n := req.State.VerifyWebhookArgs, req.Inputs
	detail := map[string]p.PropertyDiff{}
	if o.ServiceSid != n.ServiceSid {
		detail["serviceSid"] = p.PropertyDiff{Kind: p.UpdateReplace}
	}
	if o.FriendlyName != n.FriendlyName {
		detail["friendlyName"] = p.PropertyDiff{Kind: p.Update}
	}
	if o.WebhookUrl != n.WebhookUrl {
		detail["webhookUrl"] = p.PropertyDiff{Kind: p.Update}
	}
	if !sameStrings(o.EventTypes, n.EventTypes) {
		detail["eventTypes"] = p.PropertyDiff{Kind: p.Update}
	}
	if strOrDefault(o.Status, verifyWebhookDefaultStatus) != strOrDefault(n.Status, verifyWebhookDefaultStatus) {
		detail["status"] = p.PropertyDiff{Kind: p.Update}
	}
	if strOrDefault(o.WebhookVersion, verifyWebhookDefaultVersion) != strOrDefault(n.WebhookVersion, verifyWebhookDefaultVersion) {
		detail["webhookVersion"] = p.PropertyDiff{Kind: p.Update}
	}
	return infer.DiffResponse{HasChanges: len(detail) > 0, DetailedDiff: detail}, nil
}

func verifyWebhookOutputs(st *VerifyWebhookState, o obj) {
	st.Sid, st.WebhookMethod = o.str("sid"), o.str("webhook_method")
	st.DateCreated, st.DateUpdated = o.str("date_created"), o.str("date_updated")
}

// form builds the posted fields. prev is the previous state of an update: a status or version the
// program removed is sent as Twilio's default, so removal resets it.
func (a *VerifyWebhookArgs) form(prev *VerifyWebhookArgs) url.Values {
	v := url.Values{"FriendlyName": {a.FriendlyName}, "WebhookUrl": {a.WebhookUrl}, "EventTypes": append([]string(nil), a.EventTypes...)}
	switch {
	case a.Status != nil:
		v.Set("Status", *a.Status)
	case prev != nil && prev.Status != nil:
		v.Set("Status", verifyWebhookDefaultStatus)
	}
	switch {
	case a.WebhookVersion != nil:
		v.Set("Version", *a.WebhookVersion)
	case prev != nil && prev.WebhookVersion != nil:
		v.Set("Version", verifyWebhookDefaultVersion)
	}
	return v
}

func (r *VerifyWebhook) Create(ctx context.Context, req infer.CreateRequest[VerifyWebhookArgs]) (infer.CreateResponse[VerifyWebhookState], error) {
	a := req.Inputs
	if req.DryRun {
		return infer.CreateResponse[VerifyWebhookState]{Output: VerifyWebhookState{VerifyWebhookArgs: a}}, nil
	}
	c, err := clientFor(ctx)
	if err != nil {
		return infer.CreateResponse[VerifyWebhookState]{}, err
	}
	o, err := c.do(ctx, "POST", "verify", verifyChildPath(a.ServiceSid, "Webhooks", ""), nil, a.form(nil))
	if err != nil {
		return infer.CreateResponse[VerifyWebhookState]{}, fmt.Errorf("create webhook on Verify service %s: %w", a.ServiceSid, err)
	}
	st := VerifyWebhookState{VerifyWebhookArgs: a}
	verifyWebhookOutputs(&st, o)
	return infer.CreateResponse[VerifyWebhookState]{ID: a.ServiceSid + "/" + st.Sid, Output: st}, nil
}

func (r *VerifyWebhook) Read(ctx context.Context, req infer.ReadRequest[VerifyWebhookArgs, VerifyWebhookState]) (infer.ReadResponse[VerifyWebhookArgs, VerifyWebhookState], error) {
	type resp = infer.ReadResponse[VerifyWebhookArgs, VerifyWebhookState]
	ids, err := splitVerifyID(req.ID, 2, "<serviceSid>/<sid>")
	if err != nil {
		return resp{}, err
	}
	c, err := clientFor(ctx)
	if err != nil {
		return resp{}, err
	}
	o, err := c.do(ctx, "GET", "verify", verifyChildPath(ids[0], "Webhooks", ids[1]), nil, nil)
	if err != nil {
		if isNotFound(err) {
			return resp{}, nil
		}
		return resp{}, fmt.Errorf("read webhook %s: %w", req.ID, err)
	}
	in := req.Inputs
	imported := in.ServiceSid == ""
	if imported {
		in = VerifyWebhookArgs{ServiceSid: ids[0]}
	}
	in.FriendlyName, in.WebhookUrl = o.str("friendly_name"), o.str("webhook_url")
	if have := o.optList("event_types"); have != nil && !sameStrings(*have, in.EventTypes) {
		in.EventTypes = *have
	}
	// Status and version are adopted on import only when they differ from Twilio's defaults,
	// and refreshed afterwards only when the program manages them.
	if s := o.str("status"); imported && s != "" && s != verifyWebhookDefaultStatus || !imported && in.Status != nil && s != "" {
		in.Status = &s
	}
	if v := o.str("version"); imported && v != "" && v != verifyWebhookDefaultVersion || !imported && in.WebhookVersion != nil && v != "" {
		in.WebhookVersion = &v
	}
	st := VerifyWebhookState{VerifyWebhookArgs: in}
	verifyWebhookOutputs(&st, o)
	return resp{ID: req.ID, Inputs: in, State: st}, nil
}

func (r *VerifyWebhook) Update(ctx context.Context, req infer.UpdateRequest[VerifyWebhookArgs, VerifyWebhookState]) (infer.UpdateResponse[VerifyWebhookState], error) {
	a := req.Inputs
	st := req.State
	prev := st.VerifyWebhookArgs
	st.VerifyWebhookArgs = a
	if req.DryRun {
		return infer.UpdateResponse[VerifyWebhookState]{Output: st}, nil
	}
	ids, err := splitVerifyID(req.ID, 2, "<serviceSid>/<sid>")
	if err != nil {
		return infer.UpdateResponse[VerifyWebhookState]{}, err
	}
	c, err := clientFor(ctx)
	if err != nil {
		return infer.UpdateResponse[VerifyWebhookState]{}, err
	}
	o, err := c.do(ctx, "POST", "verify", verifyChildPath(ids[0], "Webhooks", ids[1]), nil, a.form(&prev))
	if err != nil {
		return infer.UpdateResponse[VerifyWebhookState]{}, fmt.Errorf("update webhook %s: %w", req.ID, err)
	}
	verifyWebhookOutputs(&st, o)
	return infer.UpdateResponse[VerifyWebhookState]{Output: st}, nil
}

func (r *VerifyWebhook) Delete(ctx context.Context, req infer.DeleteRequest[VerifyWebhookState]) (infer.DeleteResponse, error) {
	ids, err := splitVerifyID(req.ID, 2, "<serviceSid>/<sid>")
	if err != nil {
		return infer.DeleteResponse{}, err
	}
	c, err := clientFor(ctx)
	if err != nil {
		return infer.DeleteResponse{}, err
	}
	if _, err := c.do(ctx, "DELETE", "verify", verifyChildPath(ids[0], "Webhooks", ids[1]), nil, nil); err != nil && !isNotFound(err) {
		return infer.DeleteResponse{}, fmt.Errorf("delete webhook %s: %w", req.ID, err)
	}
	return infer.DeleteResponse{}, nil
}
