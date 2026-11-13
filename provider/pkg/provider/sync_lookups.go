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

// Lookups of the Sync objects. Each reads one object and returns what Twilio reports, null where
// Twilio reports nothing.

// ------------------------------------------------------------ getSyncService

// GetSyncService looks up a Sync service by SID.
type GetSyncService struct{}

// GetSyncServiceArgs are the lookup arguments.
type GetSyncServiceArgs struct {
	Sid string `pulumi:"sid"`
}

func (a *GetSyncServiceArgs) Annotate(an infer.Annotator) {
	an.Describe(&a.Sid, "SID (IS...) of the Sync service.")
}

// GetSyncServiceResult is the looked-up service.
type GetSyncServiceResult struct {
	Sid                           string  `pulumi:"sid"`
	FriendlyName                  *string `pulumi:"friendlyName,optional"`
	WebhookUrl                    *string `pulumi:"webhookUrl,optional"`
	ReachabilityWebhooksEnabled   *bool   `pulumi:"reachabilityWebhooksEnabled,optional"`
	AclEnabled                    *bool   `pulumi:"aclEnabled,optional"`
	ReachabilityDebouncingEnabled *bool   `pulumi:"reachabilityDebouncingEnabled,optional"`
	ReachabilityDebouncingWindow  *int    `pulumi:"reachabilityDebouncingWindow,optional"`
	WebhooksFromRestEnabled       *bool   `pulumi:"webhooksFromRestEnabled,optional"`
	DateCreated                   string  `pulumi:"dateCreated"`
	DateUpdated                   string  `pulumi:"dateUpdated"`
}

func (f *GetSyncService) Annotate(a infer.Annotator) {
	a.Describe(&f, "Look up a Sync service by SID.")
}

func (f *GetSyncService) Invoke(ctx context.Context, req infer.FunctionRequest[GetSyncServiceArgs]) (infer.FunctionResponse[GetSyncServiceResult], error) {
	const fn = "getSyncService"
	type resp = infer.FunctionResponse[GetSyncServiceResult]
	if err := requireArg(fn, "sid", req.Input.Sid); err != nil {
		return resp{}, err
	}
	c, err := clientFor(ctx)
	if err != nil {
		return resp{}, err
	}
	o, err := c.do(ctx, "GET", syncProduct, syncPath(req.Input.Sid), nil, nil)
	if err != nil {
		return resp{}, lookupFailure(fn, "Sync service "+req.Input.Sid, err)
	}
	sid, created, updated := trSid(o)
	return resp{Output: GetSyncServiceResult{
		Sid:                           sid,
		FriendlyName:                  nonEmpty(o.optStr("friendly_name")),
		WebhookUrl:                    nonEmpty(o.optStr("webhook_url")),
		ReachabilityWebhooksEnabled:   o.optBool("reachability_webhooks_enabled"),
		AclEnabled:                    o.optBool("acl_enabled"),
		ReachabilityDebouncingEnabled: o.optBool("reachability_debouncing_enabled"),
		ReachabilityDebouncingWindow:  o.optInt("reachability_debouncing_window"),
		WebhooksFromRestEnabled:       o.optBool("webhooks_from_rest_enabled"),
		DateCreated:                   created,
		DateUpdated:                   updated,
	}}, nil
}
