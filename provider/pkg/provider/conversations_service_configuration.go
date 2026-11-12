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

// ConversationsServiceConfiguration manages the configuration of a Conversations service: the
// default roles and the reachability indicator. A service has exactly one configuration, so this
// adopts it; destroying the resource forgets it and leaves the settings as they are.
type ConversationsServiceConfiguration struct{}

// ConversationsServiceConfigurationArgs are the inputs of a ConversationsServiceConfiguration.
type ConversationsServiceConfigurationArgs struct {
	ChatServiceSid                 string  `pulumi:"chatServiceSid" provider:"replaceOnChanges"`
	DefaultConversationCreatorRole *string `pulumi:"defaultConversationCreatorRoleSid,optional"`
	DefaultConversationRole        *string `pulumi:"defaultConversationRoleSid,optional"`
	DefaultChatServiceRole         *string `pulumi:"defaultChatServiceRoleSid,optional"`
	ReachabilityEnabled            *bool   `pulumi:"reachabilityEnabled,optional"`
}

func (a *ConversationsServiceConfigurationArgs) Annotate(an infer.Annotator) {
	an.Describe(&a.ChatServiceSid, "SID (IS...) of the Conversations service. Changing it replaces the resource.")
	an.Describe(&a.DefaultConversationCreatorRole, "SID (RL...) of the conversation role given to the creator of a conversation. Removing it leaves the setting unchanged.")
	an.Describe(&a.DefaultConversationRole, "SID (RL...) of the conversation role given to users added to a conversation. Removing it leaves the setting unchanged.")
	an.Describe(&a.DefaultChatServiceRole, "SID (RL...) of the service role given to users added to the service. Removing it leaves the setting unchanged.")
	an.Describe(&a.ReachabilityEnabled, "Whether the Reachability Indicator is enabled for the service (Twilio default false). Removing it leaves the setting unchanged.")
}

// ConversationsServiceConfigurationState is the persisted state of a ConversationsServiceConfiguration.
type ConversationsServiceConfigurationState struct {
	ConversationsServiceConfigurationArgs
}

func (r *ConversationsServiceConfiguration) Annotate(a infer.Annotator) {
	a.Describe(&r, "The configuration of a Conversations service (default roles, reachability). The resource ID is the service SID. Destroying it forgets it and leaves the settings on Twilio as they are.")
}

var _ infer.CustomResource[ConversationsServiceConfigurationArgs, ConversationsServiceConfigurationState] = (*ConversationsServiceConfiguration)(nil)
var _ infer.CustomCheck[ConversationsServiceConfigurationArgs] = (*ConversationsServiceConfiguration)(nil)
var _ infer.CustomDiff[ConversationsServiceConfigurationArgs, ConversationsServiceConfigurationState] = (*ConversationsServiceConfiguration)(nil)
var _ infer.CustomRead[ConversationsServiceConfigurationArgs, ConversationsServiceConfigurationState] = (*ConversationsServiceConfiguration)(nil)
var _ infer.CustomUpdate[ConversationsServiceConfigurationArgs, ConversationsServiceConfigurationState] = (*ConversationsServiceConfiguration)(nil)
var _ infer.CustomDelete[ConversationsServiceConfigurationState] = (*ConversationsServiceConfiguration)(nil)

func (r *ConversationsServiceConfiguration) Check(ctx context.Context, req infer.CheckRequest) (infer.CheckResponse[ConversationsServiceConfigurationArgs], error) {
	args, failures, err := infer.DefaultCheck[ConversationsServiceConfigurationArgs](ctx, req.NewInputs)
	if err != nil || len(failures) > 0 {
		return infer.CheckResponse[ConversationsServiceConfigurationArgs]{Inputs: args, Failures: failures}, err
	}
	c := &checker{unk: unknowns(req.NewInputs)}
	c.sid("chatServiceSid", args.ChatServiceSid, cvServiceSidRE, "Conversations service (IS followed by 32 hex characters)")
	for prop, v := range map[string]*string{
		"defaultConversationCreatorRoleSid": args.DefaultConversationCreatorRole,
		"defaultConversationRoleSid":        args.DefaultConversationRole,
		"defaultChatServiceRoleSid":         args.DefaultChatServiceRole,
	} {
		if v != nil {
			c.sid(prop, *v, cvRoleSidRE, "Conversations role (RL followed by 32 hex characters)")
		}
	}
	return infer.CheckResponse[ConversationsServiceConfigurationArgs]{Inputs: args, Failures: c.failures}, nil
}

func (r *ConversationsServiceConfiguration) Diff(_ context.Context, req infer.DiffRequest[ConversationsServiceConfigurationArgs, ConversationsServiceConfigurationState]) (infer.DiffResponse, error) {
	o, n := req.State.ConversationsServiceConfigurationArgs, req.Inputs
	d := trDiffSet{}
	d.replace("chatServiceSid", o.ChatServiceSid, n.ChatServiceSid)
	d.str("defaultConversationCreatorRoleSid", o.DefaultConversationCreatorRole, n.DefaultConversationCreatorRole)
	d.str("defaultConversationRoleSid", o.DefaultConversationRole, n.DefaultConversationRole)
	d.str("defaultChatServiceRoleSid", o.DefaultChatServiceRole, n.DefaultChatServiceRole)
	d.boolean("reachabilityEnabled", o.ReachabilityEnabled, n.ReachabilityEnabled)
	return d.response(), nil
}

func (a ConversationsServiceConfigurationArgs) form() url.Values {
	form := url.Values{}
	setS(form, "DefaultConversationCreatorRoleSid", a.DefaultConversationCreatorRole)
	setS(form, "DefaultConversationRoleSid", a.DefaultConversationRole)
	setS(form, "DefaultChatServiceRoleSid", a.DefaultChatServiceRole)
	setB(form, "ReachabilityEnabled", a.ReachabilityEnabled)
	return form
}

// apply writes the settings the program states; with none it only checks that the service exists.
func (a ConversationsServiceConfigurationArgs) apply(ctx context.Context, verb string) error {
	c, err := clientFor(ctx)
	if err != nil {
		return err
	}
	form, method := a.form(), "POST"
	if len(form) == 0 {
		method, form = "GET", nil
	}
	if _, err := c.do(ctx, method, cvProduct, cvPath(a.ChatServiceSid, "Configuration", ""), nil, form); err != nil {
		return fmt.Errorf("%s Conversations service configuration %s: %w", verb, a.ChatServiceSid, err)
	}
	return nil
}

func (r *ConversationsServiceConfiguration) Create(ctx context.Context, req infer.CreateRequest[ConversationsServiceConfigurationArgs]) (infer.CreateResponse[ConversationsServiceConfigurationState], error) {
	type resp = infer.CreateResponse[ConversationsServiceConfigurationState]
	a := req.Inputs
	st := ConversationsServiceConfigurationState{ConversationsServiceConfigurationArgs: a}
	if req.DryRun {
		return resp{Output: st}, nil
	}
	if err := a.apply(ctx, "configure"); err != nil {
		return resp{}, err
	}
	return resp{ID: a.ChatServiceSid, Output: st}, nil
}

func (r *ConversationsServiceConfiguration) Read(ctx context.Context, req infer.ReadRequest[ConversationsServiceConfigurationArgs, ConversationsServiceConfigurationState]) (infer.ReadResponse[ConversationsServiceConfigurationArgs, ConversationsServiceConfigurationState], error) {
	type resp = infer.ReadResponse[ConversationsServiceConfigurationArgs, ConversationsServiceConfigurationState]
	c, err := clientFor(ctx)
	if err != nil {
		return resp{}, err
	}
	o, err := c.do(ctx, "GET", cvProduct, cvPath(req.ID, "Configuration", ""), nil, nil)
	if err != nil {
		if isNotFound(err) {
			if err := notFoundOnImport("Conversations service configuration", req.ID, req.Inputs, req.State); err != nil {
				return resp{}, err
			}
			return resp{}, nil
		}
		return resp{}, fmt.Errorf("read Conversations service configuration %s: %w", req.ID, err)
	}
	in := req.Inputs
	imported := in.ChatServiceSid == ""
	in.ChatServiceSid = req.ID
	in.DefaultConversationCreatorRole = trStr(in.DefaultConversationCreatorRole, o.optStr("default_conversation_creator_role_sid"), "", imported)
	in.DefaultConversationRole = trStr(in.DefaultConversationRole, o.optStr("default_conversation_role_sid"), "", imported)
	in.DefaultChatServiceRole = trStr(in.DefaultChatServiceRole, o.optStr("default_chat_service_role_sid"), "", imported)
	in.ReachabilityEnabled = trBool(in.ReachabilityEnabled, o.optBool("reachability_enabled"), false, imported)
	return resp{ID: req.ID, Inputs: in, State: ConversationsServiceConfigurationState{ConversationsServiceConfigurationArgs: in}}, nil
}

func (r *ConversationsServiceConfiguration) Update(ctx context.Context, req infer.UpdateRequest[ConversationsServiceConfigurationArgs, ConversationsServiceConfigurationState]) (infer.UpdateResponse[ConversationsServiceConfigurationState], error) {
	st := ConversationsServiceConfigurationState{ConversationsServiceConfigurationArgs: req.Inputs}
	if req.DryRun {
		return infer.UpdateResponse[ConversationsServiceConfigurationState]{Output: st}, nil
	}
	if err := req.Inputs.apply(ctx, "update"); err != nil {
		return infer.UpdateResponse[ConversationsServiceConfigurationState]{}, err
	}
	return infer.UpdateResponse[ConversationsServiceConfigurationState]{Output: st}, nil
}

// Delete forgets the configuration: a service always has one and the spec offers no way to remove it.
func (r *ConversationsServiceConfiguration) Delete(_ context.Context, _ infer.DeleteRequest[ConversationsServiceConfigurationState]) (infer.DeleteResponse, error) {
	return infer.DeleteResponse{}, nil
}
