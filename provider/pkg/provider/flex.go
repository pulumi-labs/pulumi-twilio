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
	"regexp"

	"github.com/pulumi/pulumi-go-provider/infer"
)

// The resources in this file family configure Flex (flex-api.twilio.com v1): the Flex Flow that
// routes a messaging channel into a Studio flow, a webhook or a TaskRouter task. The shapes follow
// twilio-oai spec/json/twilio_flex_v1.json. Chat sessions, interactions and web channels are
// runtime data and are not managed.

const flexProduct = "flex-api"

var flexChatServiceSidRE = regexp.MustCompile(`^IS[0-9a-fA-F]{32}$`)

var (
	flexChannelTypes     = []string{"web", "sms", "facebook", "whatsapp", "line", "custom"}
	flexIntegrationTypes = []string{"studio", "external", "task"}
)

// flexPath is /v1/<coll>[/sid[/sub[/subSid]]].
func flexPath(coll string, rest ...string) string {
	path := "/v1/" + coll
	for _, r := range rest {
		if r != "" {
			path += "/" + url.PathEscape(r)
		}
	}
	return path
}

// flexKeep is the refreshed value of a bool whose Twilio default is not documented. An import never
// adopts it (so a program that states it plans one harmless update at most, instead of the first
// preview removing a value Twilio chose), a later refresh follows Twilio when the program states it.
func flexKeep(in, api *bool, imported bool) *bool {
	if imported {
		return nil
	}
	return trBool(in, api, false, false)
}

// ------------------------------------------------------------ FlexFlow

// FlexFlow manages a Flex Flow: the routing of one messaging channel (SMS, WhatsApp, web chat, ...)
// into a Studio flow, an external webhook or a TaskRouter task.
type FlexFlow struct{}

// FlexFlowArgs are the inputs of a FlexFlow.
type FlexFlowArgs struct {
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
}

func (a *FlexFlowArgs) Annotate(an infer.Annotator) {
	an.Describe(&a.FriendlyName, "A descriptive name for the Flex Flow.")
	an.Describe(&a.ChatServiceSid, "The SID (IS...) of the Chat service the Flex Flow creates its chat channels in.")
	an.Describe(&a.ChannelType, "The messaging channel the Flex Flow serves: web, sms, facebook, whatsapp, line or custom.")
	an.Describe(&a.ContactIdentity, "The identity of the channel contact, for example the phone number or the WhatsApp sender the Flex Flow listens on. Removing it leaves the identity unchanged.")
	an.Describe(&a.Enabled, "Whether the Flex Flow is enabled. Twilio does not document its default, so an import does not adopt it. Removing it leaves the setting unchanged.")
	an.Describe(&a.IntegrationType, "Where inbound messages go: studio (a Studio flow), external (a webhook) or task (a TaskRouter task). Removing it leaves the integration type unchanged.")
	an.Describe(&a.IntegrationFlowSid, "The SID (FW...) of the Studio flow. Required when integrationType is studio.")
	an.Describe(&a.IntegrationUrl, "The URL of the external webhook. Required when integrationType is external.")
	an.Describe(&a.IntegrationWorkspaceSid, "The SID (WS...) of the TaskRouter workspace the Task is created in. Required when integrationType is task.")
	an.Describe(&a.IntegrationWorkflowSid, "The SID (WW...) of the TaskRouter workflow that routes the Task. Required when integrationType is task.")
	an.Describe(&a.IntegrationChannel, "The Task Channel SID (TC...) or unique name (for example sms) of the Task that is created (Twilio default default). Applies when integrationType is task.")
	an.Describe(&a.IntegrationTimeout, "The Task timeout in seconds (Twilio default 86400, 24 hours). Applies when integrationType is task.")
	an.Describe(&a.IntegrationPriority, "The priority of the Task that is created (Twilio default 0). Applies when integrationType is task.")
	an.Describe(&a.IntegrationCreationOnMessage, "For outbound messaging, false creates the Task immediately (reserving the conversation to the current agent) and true delays it until the customer first replies. Twilio does not document its default, so an import does not adopt it. Applies when integrationType is task.")
	an.Describe(&a.IntegrationRetryCount, "How many times to retry the Studio flow or webhook on failure, 0 to 3 (Twilio default 3). Applies when integrationType is studio or external.")
	an.Describe(&a.LongLived, "Whether Flex keeps the chat channel active for later interactions with the same contact identity (Twilio default false).")
	an.Describe(&a.JanitorEnabled, "Whether the Messaging Channel Janitor removes active Proxy sessions when the associated Task is deleted outside the Flex UI (Twilio default false).")
}

// FlexFlowState is the persisted state of a FlexFlow.
type FlexFlowState struct {
	FlexFlowArgs

	Sid         string `pulumi:"sid"`
	DateCreated string `pulumi:"dateCreated"`
	DateUpdated string `pulumi:"dateUpdated"`
}

func (s *FlexFlowState) Annotate(a infer.Annotator) {
	a.Describe(&s.Sid, "The Flex Flow SID (FO...).")
	a.Describe(&s.DateCreated, "When the Flex Flow was created (ISO 8601).")
	a.Describe(&s.DateUpdated, "When the Flex Flow was last updated (ISO 8601).")
}

func (r *FlexFlow) Annotate(a infer.Annotator) {
	a.Describe(&r, "A Flex Flow. The resource ID is the Flex Flow SID. Optional settings that are set are sent and kept in step with Twilio; one that is removed leaves the setting as it is on Twilio. Destroying the resource deletes the Flex Flow.")
}

var _ infer.CustomResource[FlexFlowArgs, FlexFlowState] = (*FlexFlow)(nil)
var _ infer.CustomCheck[FlexFlowArgs] = (*FlexFlow)(nil)
var _ infer.CustomDiff[FlexFlowArgs, FlexFlowState] = (*FlexFlow)(nil)
var _ infer.CustomRead[FlexFlowArgs, FlexFlowState] = (*FlexFlow)(nil)
var _ infer.CustomUpdate[FlexFlowArgs, FlexFlowState] = (*FlexFlow)(nil)
var _ infer.CustomDelete[FlexFlowState] = (*FlexFlow)(nil)

// flexFlowDefaults are the Twilio defaults of this resource, shared by Read (import adoption) and Diff.
var flexFlowDefaults = trDefaults{
	"contactIdentity":       "",
	"integrationChannel":    "default",
	"integrationTimeout":    86400,
	"integrationPriority":   0,
	"integrationRetryCount": 3,
	"longLived":             false,
	"janitorEnabled":        false,
}

func (r *FlexFlow) Check(ctx context.Context, req infer.CheckRequest) (infer.CheckResponse[FlexFlowArgs], error) {
	args, failures, err := infer.DefaultCheck[FlexFlowArgs](ctx, req.NewInputs)
	if err != nil || len(failures) > 0 {
		return infer.CheckResponse[FlexFlowArgs]{Inputs: args, Failures: failures}, err
	}
	c := &checker{unk: unknowns(req.NewInputs)}
	c.required("friendlyName", args.FriendlyName)
	c.sid("chatServiceSid", args.ChatServiceSid, flexChatServiceSidRE, "Chat service (IS followed by 32 hex characters)")
	c.oneOf("channelType", args.ChannelType, flexChannelTypes)
	if v := args.IntegrationType; v != nil {
		c.oneOf("integrationType", *v, flexIntegrationTypes)
		if !c.unk("integrationType") {
			need := func(prop string, v *string) {
				if !c.unk(prop) && (v == nil || *v == "") {
					c.fail(prop, prop+" is required when integrationType is "+*args.IntegrationType)
				}
			}
			switch *args.IntegrationType {
			case "studio":
				need("integrationFlowSid", args.IntegrationFlowSid)
			case "external":
				need("integrationUrl", args.IntegrationUrl)
			case "task":
				need("integrationWorkspaceSid", args.IntegrationWorkspaceSid)
				need("integrationWorkflowSid", args.IntegrationWorkflowSid)
			}
		}
	}
	if v := args.IntegrationRetryCount; v != nil && !c.unk("integrationRetryCount") && (*v < 0 || *v > 3) {
		c.fail("integrationRetryCount", "integrationRetryCount must be between 0 and 3")
	}
	if v := args.IntegrationTimeout; v != nil && !c.unk("integrationTimeout") && *v < 1 {
		c.fail("integrationTimeout", "integrationTimeout must be at least 1 second")
	}
	return infer.CheckResponse[FlexFlowArgs]{Inputs: args, Failures: c.failures}, nil
}

func (r *FlexFlow) Diff(_ context.Context, req infer.DiffRequest[FlexFlowArgs, FlexFlowState]) (infer.DiffResponse, error) {
	o, n := req.State.FlexFlowArgs, req.Inputs
	d := trDiffSet{}
	d.requiredStr("friendlyName", o.FriendlyName, n.FriendlyName)
	d.requiredStr("chatServiceSid", o.ChatServiceSid, n.ChatServiceSid)
	d.requiredStr("channelType", o.ChannelType, n.ChannelType)
	d.strDef(flexFlowDefaults, "contactIdentity", o.ContactIdentity, n.ContactIdentity)
	d.boolean("enabled", o.Enabled, n.Enabled)
	d.str("integrationType", o.IntegrationType, n.IntegrationType)
	d.str("integrationFlowSid", o.IntegrationFlowSid, n.IntegrationFlowSid)
	d.str("integrationUrl", o.IntegrationUrl, n.IntegrationUrl)
	d.str("integrationWorkspaceSid", o.IntegrationWorkspaceSid, n.IntegrationWorkspaceSid)
	d.str("integrationWorkflowSid", o.IntegrationWorkflowSid, n.IntegrationWorkflowSid)
	d.strDef(flexFlowDefaults, "integrationChannel", o.IntegrationChannel, n.IntegrationChannel)
	d.intDef(flexFlowDefaults, "integrationTimeout", o.IntegrationTimeout, n.IntegrationTimeout)
	d.intDef(flexFlowDefaults, "integrationPriority", o.IntegrationPriority, n.IntegrationPriority)
	d.boolean("integrationCreationOnMessage", o.IntegrationCreationOnMessage, n.IntegrationCreationOnMessage)
	d.intDef(flexFlowDefaults, "integrationRetryCount", o.IntegrationRetryCount, n.IntegrationRetryCount)
	d.boolDef(flexFlowDefaults, "longLived", o.LongLived, n.LongLived)
	d.boolDef(flexFlowDefaults, "janitorEnabled", o.JanitorEnabled, n.JanitorEnabled)
	return d.response(), nil
}

// form is the body of the create and the update call: every input that is set.
func (a FlexFlowArgs) form() url.Values {
	form := url.Values{}
	form.Set("FriendlyName", a.FriendlyName)
	form.Set("ChatServiceSid", a.ChatServiceSid)
	form.Set("ChannelType", a.ChannelType)
	setS(form, "ContactIdentity", a.ContactIdentity)
	setB(form, "Enabled", a.Enabled)
	setS(form, "IntegrationType", a.IntegrationType)
	setS(form, "Integration.FlowSid", a.IntegrationFlowSid)
	setS(form, "Integration.Url", a.IntegrationUrl)
	setS(form, "Integration.WorkspaceSid", a.IntegrationWorkspaceSid)
	setS(form, "Integration.WorkflowSid", a.IntegrationWorkflowSid)
	setS(form, "Integration.Channel", a.IntegrationChannel)
	setI(form, "Integration.Timeout", a.IntegrationTimeout)
	setI(form, "Integration.Priority", a.IntegrationPriority)
	setB(form, "Integration.CreationOnMessage", a.IntegrationCreationOnMessage)
	setI(form, "Integration.RetryCount", a.IntegrationRetryCount)
	setB(form, "LongLived", a.LongLived)
	setB(form, "JanitorEnabled", a.JanitorEnabled)
	return form
}

// fill sets the values Twilio reports.
func (s *FlexFlowState) fill(o obj) {
	s.Sid, s.DateCreated, s.DateUpdated = trSid(o)
}

func (r *FlexFlow) Create(ctx context.Context, req infer.CreateRequest[FlexFlowArgs]) (infer.CreateResponse[FlexFlowState], error) {
	type resp = infer.CreateResponse[FlexFlowState]
	a := req.Inputs
	if req.DryRun {
		return resp{Output: FlexFlowState{FlexFlowArgs: a}}, nil
	}
	c, err := clientFor(ctx)
	if err != nil {
		return resp{}, err
	}
	o, err := c.do(ctx, "POST", flexProduct, flexPath("FlexFlows"), nil, a.form())
	if err != nil {
		return resp{}, fmt.Errorf("create Flex Flow: %w", err)
	}
	st := FlexFlowState{FlexFlowArgs: a}
	st.fill(o)
	return resp{ID: st.Sid, Output: st}, nil
}

func (r *FlexFlow) Read(ctx context.Context, req infer.ReadRequest[FlexFlowArgs, FlexFlowState]) (infer.ReadResponse[FlexFlowArgs, FlexFlowState], error) {
	type resp = infer.ReadResponse[FlexFlowArgs, FlexFlowState]
	c, err := clientFor(ctx)
	if err != nil {
		return resp{}, err
	}
	o, err := c.do(ctx, "GET", flexProduct, flexPath("FlexFlows", req.ID), nil, nil)
	if err != nil {
		if isNotFound(err) {
			if err := notFoundOnImport("Flex Flow", req.ID, req.Inputs, req.State); err != nil {
				return resp{}, err
			}
			return resp{}, nil
		}
		return resp{}, fmt.Errorf("read Flex Flow %s: %w", req.ID, err)
	}
	in := req.Inputs
	imported := req.State.Sid == ""
	if v := o.str("friendly_name"); v != "" {
		in.FriendlyName = v
	}
	if v := o.str("chat_service_sid"); v != "" {
		in.ChatServiceSid = v
	}
	if v := o.str("channel_type"); v != "" {
		in.ChannelType = v
	}
	d := flexFlowDefaults
	in.ContactIdentity = trStr(in.ContactIdentity, o.optStr("contact_identity"), d.str("contactIdentity"), imported)
	in.Enabled = flexKeep(in.Enabled, o.optBool("enabled"), imported)
	in.IntegrationType = trStr(in.IntegrationType, o.optStr("integration_type"), "", imported)
	in.IntegrationFlowSid = trStr(in.IntegrationFlowSid, o.optStr("integration.flow_sid"), "", imported)
	in.IntegrationUrl = trStr(in.IntegrationUrl, o.optStr("integration.url"), "", imported)
	in.IntegrationWorkspaceSid = trStr(in.IntegrationWorkspaceSid, o.optStr("integration.workspace_sid"), "", imported)
	in.IntegrationWorkflowSid = trStr(in.IntegrationWorkflowSid, o.optStr("integration.workflow_sid"), "", imported)
	in.IntegrationChannel = trStr(in.IntegrationChannel, o.optStr("integration.channel"), d.str("integrationChannel"), imported)
	in.IntegrationTimeout = trInt(in.IntegrationTimeout, o.optInt("integration.timeout"), d.integer("integrationTimeout"), imported)
	in.IntegrationPriority = trInt(in.IntegrationPriority, o.optInt("integration.priority"), d.integer("integrationPriority"), imported)
	in.IntegrationCreationOnMessage = flexKeep(in.IntegrationCreationOnMessage, o.optBool("integration.creation_on_message"), imported)
	in.IntegrationRetryCount = trInt(in.IntegrationRetryCount, o.optInt("integration.retry_count"), d.integer("integrationRetryCount"), imported)
	in.LongLived = trBool(in.LongLived, o.optBool("long_lived"), d.boolean("longLived"), imported)
	in.JanitorEnabled = trBool(in.JanitorEnabled, o.optBool("janitor_enabled"), d.boolean("janitorEnabled"), imported)
	st := FlexFlowState{FlexFlowArgs: in}
	st.fill(o)
	return resp{ID: req.ID, Inputs: in, State: st}, nil
}

func (r *FlexFlow) Update(ctx context.Context, req infer.UpdateRequest[FlexFlowArgs, FlexFlowState]) (infer.UpdateResponse[FlexFlowState], error) {
	type resp = infer.UpdateResponse[FlexFlowState]
	st := req.State
	st.FlexFlowArgs = req.Inputs
	if req.DryRun {
		return resp{Output: st}, nil
	}
	c, err := clientFor(ctx)
	if err != nil {
		return resp{}, err
	}
	o, err := c.do(ctx, "POST", flexProduct, flexPath("FlexFlows", req.ID), nil, req.Inputs.form())
	if err != nil {
		return resp{}, fmt.Errorf("update Flex Flow %s: %w", req.ID, err)
	}
	st.fill(o)
	return resp{Output: st}, nil
}

func (r *FlexFlow) Delete(ctx context.Context, req infer.DeleteRequest[FlexFlowState]) (infer.DeleteResponse, error) {
	c, err := clientFor(ctx)
	if err != nil {
		return infer.DeleteResponse{}, err
	}
	if _, err := c.do(ctx, "DELETE", flexProduct, flexPath("FlexFlows", req.ID), nil, nil); err != nil && !isNotFound(err) {
		return infer.DeleteResponse{}, fmt.Errorf("delete Flex Flow %s: %w", req.ID, err)
	}
	return infer.DeleteResponse{}, nil
}
