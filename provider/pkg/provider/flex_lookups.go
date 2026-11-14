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
	"net/url"

	"github.com/pulumi/pulumi-go-provider/infer"
)

// ------------------------------------------------------------ getFlexFlow

// GetFlexFlow looks up a Flex Flow by SID.
type GetFlexFlow struct{}

// GetFlexFlowArgs are the lookup arguments.
type GetFlexFlowArgs struct {
	Sid string `pulumi:"sid"`
}

func (a *GetFlexFlowArgs) Annotate(an infer.Annotator) {
	an.Describe(&a.Sid, "SID (FO...) of the Flex Flow.")
}

// FlexFlowInfo is one Flex Flow as Twilio reports it.
type FlexFlowInfo struct {
	Sid                          string  `pulumi:"sid"`
	FriendlyName                 string  `pulumi:"friendlyName"`
	ChatServiceSid               string  `pulumi:"chatServiceSid"`
	ChannelType                  string  `pulumi:"channelType"`
	ContactIdentity              *string `pulumi:"contactIdentity,optional"`
	Enabled                      *bool   `pulumi:"enabled,optional"`
	IntegrationType              *string `pulumi:"integrationType,optional"`
	IntegrationFlowSid           *string `pulumi:"integrationFlowSid,optional"`
	IntegrationUrl               *string `pulumi:"integrationUrl,optional"`
	IntegrationWorkspaceSid      *string `pulumi:"integrationWorkspaceSid,optional"`
	IntegrationWorkflowSid       *string `pulumi:"integrationWorkflowSid,optional"`
	IntegrationChannel           *string `pulumi:"integrationChannel,optional"`
	IntegrationTimeout           *int    `pulumi:"integrationTimeout,optional"`
	IntegrationPriority          *int    `pulumi:"integrationPriority,optional"`
	IntegrationCreationOnMessage *bool   `pulumi:"integrationCreationOnMessage,optional"`
	IntegrationRetryCount        *int    `pulumi:"integrationRetryCount,optional"`
	LongLived                    *bool   `pulumi:"longLived,optional"`
	JanitorEnabled               *bool   `pulumi:"janitorEnabled,optional"`
	DateCreated                  string  `pulumi:"dateCreated"`
	DateUpdated                  string  `pulumi:"dateUpdated"`
}

func (a *FlexFlowInfo) Annotate(an infer.Annotator) {
	an.Describe(&a.Sid, "The Flex Flow SID (FO...).")
	an.Describe(&a.FriendlyName, "The name of the Flex Flow.")
	an.Describe(&a.ChatServiceSid, "The SID (IS...) of the Chat service.")
	an.Describe(&a.ChannelType, "The messaging channel: web, sms, facebook, whatsapp, line or custom.")
	an.Describe(&a.ContactIdentity, "The identity of the channel contact.")
	an.Describe(&a.Enabled, "Whether the Flex Flow is enabled.")
	an.Describe(&a.IntegrationType, "Where inbound messages go: studio, external or task.")
	an.Describe(&a.IntegrationFlowSid, "The SID of the Studio flow (studio integrations).")
	an.Describe(&a.IntegrationUrl, "The URL of the external webhook (external integrations).")
	an.Describe(&a.IntegrationWorkspaceSid, "The TaskRouter workspace SID (task integrations).")
	an.Describe(&a.IntegrationWorkflowSid, "The TaskRouter workflow SID (task integrations).")
	an.Describe(&a.IntegrationChannel, "The Task Channel SID or unique name (task integrations).")
	an.Describe(&a.IntegrationTimeout, "The Task timeout in seconds (task integrations).")
	an.Describe(&a.IntegrationPriority, "The Task priority (task integrations).")
	an.Describe(&a.IntegrationCreationOnMessage, "Whether Task creation waits for the first customer reply (task integrations, outbound messaging).")
	an.Describe(&a.IntegrationRetryCount, "How many times the Studio flow or webhook is retried on failure.")
	an.Describe(&a.LongLived, "Whether Flex keeps the chat channel active for later interactions.")
	an.Describe(&a.JanitorEnabled, "Whether the Messaging Channel Janitor is enabled.")
	an.Describe(&a.DateCreated, "When the Flex Flow was created (ISO 8601).")
	an.Describe(&a.DateUpdated, "When the Flex Flow was last updated (ISO 8601).")
}

func flexFlowInfo(o obj) FlexFlowInfo {
	sid, created, updated := trSid(o)
	return FlexFlowInfo{
		Sid:                          sid,
		FriendlyName:                 o.str("friendly_name"),
		ChatServiceSid:               o.str("chat_service_sid"),
		ChannelType:                  o.str("channel_type"),
		ContactIdentity:              nonEmpty(o.optStr("contact_identity")),
		Enabled:                      o.optBool("enabled"),
		IntegrationType:              nonEmpty(o.optStr("integration_type")),
		IntegrationFlowSid:           nonEmpty(o.optStr("integration.flow_sid")),
		IntegrationUrl:               nonEmpty(o.optStr("integration.url")),
		IntegrationWorkspaceSid:      nonEmpty(o.optStr("integration.workspace_sid")),
		IntegrationWorkflowSid:       nonEmpty(o.optStr("integration.workflow_sid")),
		IntegrationChannel:           nonEmpty(o.optStr("integration.channel")),
		IntegrationTimeout:           o.optInt("integration.timeout"),
		IntegrationPriority:          o.optInt("integration.priority"),
		IntegrationCreationOnMessage: o.optBool("integration.creation_on_message"),
		IntegrationRetryCount:        o.optInt("integration.retry_count"),
		LongLived:                    o.optBool("long_lived"),
		JanitorEnabled:               o.optBool("janitor_enabled"),
		DateCreated:                  created,
		DateUpdated:                  updated,
	}
}

func (f *GetFlexFlow) Annotate(a infer.Annotator) {
	a.Describe(&f, "Look up a Flex Flow by SID (GET /v1/FlexFlows/{Sid} on flex-api.twilio.com).")
}

func (f *GetFlexFlow) Invoke(ctx context.Context, req infer.FunctionRequest[GetFlexFlowArgs]) (infer.FunctionResponse[FlexFlowInfo], error) {
	const fn = "getFlexFlow"
	type resp = infer.FunctionResponse[FlexFlowInfo]
	if err := requireArg(fn, "sid", req.Input.Sid); err != nil {
		return resp{}, err
	}
	c, err := clientFor(ctx)
	if err != nil {
		return resp{}, err
	}
	o, err := c.do(ctx, "GET", flexProduct, flexPath("FlexFlows", req.Input.Sid), nil, nil)
	if err != nil {
		return resp{}, lookupFailure(fn, "Flex Flow "+req.Input.Sid, err)
	}
	return resp{Output: flexFlowInfo(o)}, nil
}

// ------------------------------------------------------------ getFlexFlows

// GetFlexFlows lists the Flex Flows of the account.
type GetFlexFlows struct{}

// GetFlexFlowsArgs are the lookup arguments.
type GetFlexFlowsArgs struct {
	FriendlyName *string `pulumi:"friendlyName,optional"`
}

func (a *GetFlexFlowsArgs) Annotate(an infer.Annotator) {
	an.Describe(&a.FriendlyName, "Only Flex Flows with exactly this name.")
}

// GetFlexFlowsResult is the list.
type GetFlexFlowsResult struct {
	FlexFlows []FlexFlowInfo `pulumi:"flexFlows"`
}

func (a *GetFlexFlowsResult) Annotate(an infer.Annotator) {
	an.Describe(&a.FlexFlows, "The matching Flex Flows, in the order Twilio returns them.")
}

func (f *GetFlexFlows) Annotate(a infer.Annotator) {
	a.Describe(&f, "List the Flex Flows of the account, optionally narrowed by name (GET /v1/FlexFlows on flex-api.twilio.com).")
}

func (f *GetFlexFlows) Invoke(ctx context.Context, req infer.FunctionRequest[GetFlexFlowsArgs]) (infer.FunctionResponse[GetFlexFlowsResult], error) {
	const fn = "getFlexFlows"
	none := infer.FunctionResponse[GetFlexFlowsResult]{}
	c, err := clientFor(ctx)
	if err != nil {
		return none, err
	}
	q := url.Values{"PageSize": {"1000"}}
	if v := req.Input.FriendlyName; v != nil && *v != "" {
		q.Set("FriendlyName", *v)
	}
	out := []FlexFlowInfo{}
	err = listPages(ctx, c, fn, flexProduct, flexPath("FlexFlows"), q, "flex_flows", func(o obj) bool {
		out = append(out, flexFlowInfo(o))
		return true
	})
	if err != nil {
		return none, lookupFailure(fn, "Flex Flows", err)
	}
	return infer.FunctionResponse[GetFlexFlowsResult]{Output: GetFlexFlowsResult{FlexFlows: out}}, nil
}
