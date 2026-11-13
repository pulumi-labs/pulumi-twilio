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

	"github.com/pulumi/pulumi-go-provider/infer"
)

// SyncService manages a Sync service: the container of Sync documents, lists, maps and streams.
type SyncService struct{}

// SyncServiceArgs are the inputs of a SyncService.
type SyncServiceArgs struct {
	FriendlyName                  *string `pulumi:"friendlyName,optional"`
	WebhookUrl                    *string `pulumi:"webhookUrl,optional"`
	ReachabilityWebhooksEnabled   *bool   `pulumi:"reachabilityWebhooksEnabled,optional"`
	AclEnabled                    *bool   `pulumi:"aclEnabled,optional"`
	ReachabilityDebouncingEnabled *bool   `pulumi:"reachabilityDebouncingEnabled,optional"`
	ReachabilityDebouncingWindow  *int    `pulumi:"reachabilityDebouncingWindow,optional"`
	WebhooksFromRestEnabled       *bool   `pulumi:"webhooksFromRestEnabled,optional"`
}

func (a *SyncServiceArgs) Annotate(an infer.Annotator) {
	an.Describe(&a.FriendlyName, "Human-readable name of the service. Removing it leaves the name unchanged.")
	an.Describe(&a.WebhookUrl, "Absolute URL Twilio calls when Sync objects are manipulated. Removing it leaves the setting unchanged.")
	an.Describe(&a.ReachabilityWebhooksEnabled, "Whether Twilio calls webhookUrl when client endpoints connect to Sync (Twilio default false).")
	an.Describe(&a.AclEnabled, "Whether token identities must be granted access to Sync objects through permissions (Twilio default false).")
	an.Describe(&a.ReachabilityDebouncingEnabled, "Whether every endpoint_disconnected event occurs after a configurable delay (Twilio default false).")
	an.Describe(&a.ReachabilityDebouncingWindow, "Delay of the reachability events in milliseconds when reachabilityDebouncingEnabled is true, 1000 to 30000 (Twilio default 5000).")
	an.Describe(&a.WebhooksFromRestEnabled, "Whether Twilio calls webhookUrl when the REST API updates Sync objects (Twilio default false).")
}

// SyncServiceState is the persisted state of a SyncService.
type SyncServiceState struct {
	SyncServiceArgs

	Sid         string `pulumi:"sid"`
	DateCreated string `pulumi:"dateCreated"`
	DateUpdated string `pulumi:"dateUpdated"`
}

func (s *SyncServiceState) Annotate(a infer.Annotator) {
	a.Describe(&s.Sid, "The service SID (IS...).")
	a.Describe(&s.DateCreated, "When the service was created (ISO 8601).")
	a.Describe(&s.DateUpdated, "When the service was last updated (ISO 8601).")
}

func (r *SyncService) Annotate(a infer.Annotator) {
	a.Describe(&r, "A Sync service. The resource ID is the service SID. Optional settings that are set are sent and kept in step with Twilio; one that is removed leaves the setting as it is on Twilio.")
}

var _ infer.CustomResource[SyncServiceArgs, SyncServiceState] = (*SyncService)(nil)
var _ infer.CustomCheck[SyncServiceArgs] = (*SyncService)(nil)
var _ infer.CustomDiff[SyncServiceArgs, SyncServiceState] = (*SyncService)(nil)
var _ infer.CustomRead[SyncServiceArgs, SyncServiceState] = (*SyncService)(nil)
var _ infer.CustomUpdate[SyncServiceArgs, SyncServiceState] = (*SyncService)(nil)
var _ infer.CustomDelete[SyncServiceState] = (*SyncService)(nil)

func (r *SyncService) Check(ctx context.Context, req infer.CheckRequest) (infer.CheckResponse[SyncServiceArgs], error) {
	args, failures, err := infer.DefaultCheck[SyncServiceArgs](ctx, req.NewInputs)
	if err != nil || len(failures) > 0 {
		return infer.CheckResponse[SyncServiceArgs]{Inputs: args, Failures: failures}, err
	}
	c := &checker{unk: unknowns(req.NewInputs)}
	if v := args.ReachabilityDebouncingWindow; v != nil && !c.unk("reachabilityDebouncingWindow") && (*v < 1000 || *v > 30000) {
		c.fail("reachabilityDebouncingWindow", "reachabilityDebouncingWindow must be between 1000 and 30000 milliseconds")
	}
	return infer.CheckResponse[SyncServiceArgs]{Inputs: args, Failures: c.failures}, nil
}

func (r *SyncService) Diff(_ context.Context, req infer.DiffRequest[SyncServiceArgs, SyncServiceState]) (infer.DiffResponse, error) {
	o, n := req.State.SyncServiceArgs, req.Inputs
	d := trDiffSet{}
	d.strDef(syncServiceDefaults, "friendlyName", o.FriendlyName, n.FriendlyName)
	d.strDef(syncServiceDefaults, "webhookUrl", o.WebhookUrl, n.WebhookUrl)
	d.boolDef(syncServiceDefaults, "reachabilityWebhooksEnabled", o.ReachabilityWebhooksEnabled, n.ReachabilityWebhooksEnabled)
	d.boolDef(syncServiceDefaults, "aclEnabled", o.AclEnabled, n.AclEnabled)
	d.boolDef(syncServiceDefaults, "reachabilityDebouncingEnabled", o.ReachabilityDebouncingEnabled, n.ReachabilityDebouncingEnabled)
	d.intDef(syncServiceDefaults, "reachabilityDebouncingWindow", o.ReachabilityDebouncingWindow, n.ReachabilityDebouncingWindow)
	d.boolDef(syncServiceDefaults, "webhooksFromRestEnabled", o.WebhooksFromRestEnabled, n.WebhooksFromRestEnabled)
	return d.response(), nil
}

func (a SyncServiceArgs) form() url.Values {
	form := url.Values{}
	setS(form, "FriendlyName", a.FriendlyName)
	setS(form, "WebhookUrl", a.WebhookUrl)
	setB(form, "ReachabilityWebhooksEnabled", a.ReachabilityWebhooksEnabled)
	setB(form, "AclEnabled", a.AclEnabled)
	setB(form, "ReachabilityDebouncingEnabled", a.ReachabilityDebouncingEnabled)
	setI(form, "ReachabilityDebouncingWindow", a.ReachabilityDebouncingWindow)
	setB(form, "WebhooksFromRestEnabled", a.WebhooksFromRestEnabled)
	return form
}

func (r *SyncService) Create(ctx context.Context, req infer.CreateRequest[SyncServiceArgs]) (infer.CreateResponse[SyncServiceState], error) {
	type resp = infer.CreateResponse[SyncServiceState]
	a := req.Inputs
	if req.DryRun {
		return resp{Output: SyncServiceState{SyncServiceArgs: a}}, nil
	}
	c, err := clientFor(ctx)
	if err != nil {
		return resp{}, err
	}
	o, err := c.do(ctx, "POST", syncProduct, syncPath(""), nil, a.form())
	if err != nil {
		return resp{}, fmt.Errorf("create Sync service: %w", err)
	}
	st := SyncServiceState{SyncServiceArgs: a}
	st.Sid, st.DateCreated, st.DateUpdated = trSid(o)
	return resp{ID: st.Sid, Output: st}, nil
}

func (r *SyncService) Read(ctx context.Context, req infer.ReadRequest[SyncServiceArgs, SyncServiceState]) (infer.ReadResponse[SyncServiceArgs, SyncServiceState], error) {
	type resp = infer.ReadResponse[SyncServiceArgs, SyncServiceState]
	c, err := clientFor(ctx)
	if err != nil {
		return resp{}, err
	}
	o, err := c.do(ctx, "GET", syncProduct, syncPath(req.ID), nil, nil)
	if err != nil {
		if isNotFound(err) {
			if err := notFoundOnImport("Sync service", req.ID, req.Inputs, req.State); err != nil {
				return resp{}, err
			}
			return resp{}, nil
		}
		return resp{}, fmt.Errorf("read Sync service %s: %w", req.ID, err)
	}
	in := req.Inputs
	// An import arrives with no prior state; the program's own settings are the only record of what it manages.
	imported := req.State.Sid == ""
	in.FriendlyName = trStr(in.FriendlyName, o.optStr("friendly_name"), syncServiceDefaults.str("friendlyName"), imported)
	in.WebhookUrl = trStr(in.WebhookUrl, o.optStr("webhook_url"), syncServiceDefaults.str("webhookUrl"), imported)
	in.ReachabilityWebhooksEnabled = trBool(in.ReachabilityWebhooksEnabled, o.optBool("reachability_webhooks_enabled"), syncServiceDefaults.boolean("reachabilityWebhooksEnabled"), imported)
	in.AclEnabled = trBool(in.AclEnabled, o.optBool("acl_enabled"), syncServiceDefaults.boolean("aclEnabled"), imported)
	in.ReachabilityDebouncingEnabled = trBool(in.ReachabilityDebouncingEnabled, o.optBool("reachability_debouncing_enabled"), syncServiceDefaults.boolean("reachabilityDebouncingEnabled"), imported)
	in.ReachabilityDebouncingWindow = trInt(in.ReachabilityDebouncingWindow, o.optInt("reachability_debouncing_window"), syncServiceDefaults.integer("reachabilityDebouncingWindow"), imported)
	in.WebhooksFromRestEnabled = trBool(in.WebhooksFromRestEnabled, o.optBool("webhooks_from_rest_enabled"), syncServiceDefaults.boolean("webhooksFromRestEnabled"), imported)
	st := SyncServiceState{SyncServiceArgs: in}
	st.Sid, st.DateCreated, st.DateUpdated = trSid(o)
	return resp{ID: req.ID, Inputs: in, State: st}, nil
}

func (r *SyncService) Update(ctx context.Context, req infer.UpdateRequest[SyncServiceArgs, SyncServiceState]) (infer.UpdateResponse[SyncServiceState], error) {
	type resp = infer.UpdateResponse[SyncServiceState]
	st := req.State
	st.SyncServiceArgs = req.Inputs
	if req.DryRun {
		return resp{Output: st}, nil
	}
	c, err := clientFor(ctx)
	if err != nil {
		return resp{}, err
	}
	o, err := c.do(ctx, "POST", syncProduct, syncPath(req.ID), nil, req.Inputs.form())
	if err != nil {
		return resp{}, fmt.Errorf("update Sync service %s: %w", req.ID, err)
	}
	st.DateUpdated = o.str("date_updated")
	return resp{Output: st}, nil
}

func (r *SyncService) Delete(ctx context.Context, req infer.DeleteRequest[SyncServiceState]) (infer.DeleteResponse, error) {
	c, err := clientFor(ctx)
	if err != nil {
		return infer.DeleteResponse{}, err
	}
	if _, err := c.do(ctx, "DELETE", syncProduct, syncPath(req.ID), nil, nil); err != nil && !isNotFound(err) {
		return infer.DeleteResponse{}, fmt.Errorf("delete Sync service %s: %w", req.ID, err)
	}
	return infer.DeleteResponse{}, nil
}

// syncServiceDefaults are the Twilio defaults of this resource, shared by Read (import adoption) and Diff.
var syncServiceDefaults = trDefaults{
	"friendlyName":                  "",
	"webhookUrl":                    "",
	"reachabilityWebhooksEnabled":   false,
	"aclEnabled":                    false,
	"reachabilityDebouncingEnabled": false,
	"reachabilityDebouncingWindow":  5000,
	"webhooksFromRestEnabled":       false,
}
