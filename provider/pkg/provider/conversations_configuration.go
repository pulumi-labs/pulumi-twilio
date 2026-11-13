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
	"reflect"
	"regexp"

	"github.com/pulumi/pulumi-go-provider/infer"
)

// cvAccountConfigurationID is the resource ID of the account-level configuration, of which there is one per account.
const cvAccountConfigurationID = "account"

var cvMessagingServiceSidRE = regexp.MustCompile(`^MG[0-9a-fA-F]{32}$`)

// ConversationsConfiguration manages the account-level defaults of Conversations: the service and
// messaging service used when a conversation is created and the timers that move a conversation
// to inactive and closed. An account has exactly one, so this adopts it; destroying the resource
// forgets it and leaves the settings as they are.
type ConversationsConfiguration struct{}

// ConversationsConfigurationArgs are the inputs of a ConversationsConfiguration.
type ConversationsConfigurationArgs struct {
	DefaultChatServiceSid      *string `pulumi:"defaultChatServiceSid,optional"`
	DefaultMessagingServiceSid *string `pulumi:"defaultMessagingServiceSid,optional"`
	DefaultInactiveTimer       *string `pulumi:"defaultInactiveTimer,optional"`
	DefaultClosedTimer         *string `pulumi:"defaultClosedTimer,optional"`
}

func (a *ConversationsConfigurationArgs) Annotate(an infer.Annotator) {
	an.Describe(&a.DefaultChatServiceSid, "SID (IS...) of the Conversations service a conversation is created in when none is given. Removing it leaves the setting unchanged.")
	an.Describe(&a.DefaultMessagingServiceSid, "SID (MG...) of the messaging service used when a conversation is created. Removing it leaves the setting unchanged.")
	an.Describe(&a.DefaultInactiveTimer, "ISO 8601 duration after which a conversation becomes inactive, at least one minute, for example PT10M. Removing it leaves the setting unchanged.")
	an.Describe(&a.DefaultClosedTimer, "ISO 8601 duration after which a conversation is closed, at least ten minutes, for example P1D. Removing it leaves the setting unchanged.")
}

// ConversationsConfigurationState is the persisted state of a ConversationsConfiguration.
type ConversationsConfigurationState struct {
	ConversationsConfigurationArgs
}

func (r *ConversationsConfiguration) Annotate(a infer.Annotator) {
	a.Describe(&r, "The account-level defaults of Conversations. The resource ID is account. Destroying it forgets it and leaves the settings on Twilio as they are.")
}

var _ infer.CustomResource[ConversationsConfigurationArgs, ConversationsConfigurationState] = (*ConversationsConfiguration)(nil)
var _ infer.CustomCheck[ConversationsConfigurationArgs] = (*ConversationsConfiguration)(nil)
var _ infer.CustomDiff[ConversationsConfigurationArgs, ConversationsConfigurationState] = (*ConversationsConfiguration)(nil)
var _ infer.CustomRead[ConversationsConfigurationArgs, ConversationsConfigurationState] = (*ConversationsConfiguration)(nil)
var _ infer.CustomUpdate[ConversationsConfigurationArgs, ConversationsConfigurationState] = (*ConversationsConfiguration)(nil)
var _ infer.CustomDelete[ConversationsConfigurationState] = (*ConversationsConfiguration)(nil)

func (r *ConversationsConfiguration) Check(ctx context.Context, req infer.CheckRequest) (infer.CheckResponse[ConversationsConfigurationArgs], error) {
	args, failures, err := infer.DefaultCheck[ConversationsConfigurationArgs](ctx, req.NewInputs)
	if err != nil || len(failures) > 0 {
		return infer.CheckResponse[ConversationsConfigurationArgs]{Inputs: args, Failures: failures}, err
	}
	c := &checker{unk: unknowns(req.NewInputs)}
	cvCheckOptionalService(c, "defaultChatServiceSid", args.DefaultChatServiceSid)
	if v := args.DefaultMessagingServiceSid; v != nil {
		c.sid("defaultMessagingServiceSid", *v, cvMessagingServiceSidRE, "messaging service (MG followed by 32 hex characters)")
	}
	return infer.CheckResponse[ConversationsConfigurationArgs]{Inputs: args, Failures: c.failures}, nil
}

func (r *ConversationsConfiguration) Diff(_ context.Context, req infer.DiffRequest[ConversationsConfigurationArgs, ConversationsConfigurationState]) (infer.DiffResponse, error) {
	o, n := req.State.ConversationsConfigurationArgs, req.Inputs
	d := trDiffSet{}
	d.strDef(conversationsConfigurationDefaults, "defaultChatServiceSid", o.DefaultChatServiceSid, n.DefaultChatServiceSid)
	d.strDef(conversationsConfigurationDefaults, "defaultMessagingServiceSid", o.DefaultMessagingServiceSid, n.DefaultMessagingServiceSid)
	d.strDef(conversationsConfigurationDefaults, "defaultInactiveTimer", o.DefaultInactiveTimer, n.DefaultInactiveTimer)
	d.strDef(conversationsConfigurationDefaults, "defaultClosedTimer", o.DefaultClosedTimer, n.DefaultClosedTimer)
	return d.response(), nil
}

func (a ConversationsConfigurationArgs) form() url.Values {
	form := url.Values{}
	setS(form, "DefaultChatServiceSid", a.DefaultChatServiceSid)
	setS(form, "DefaultMessagingServiceSid", a.DefaultMessagingServiceSid)
	setS(form, "DefaultInactiveTimer", a.DefaultInactiveTimer)
	setS(form, "DefaultClosedTimer", a.DefaultClosedTimer)
	return form
}

// apply writes the settings the program states; with none it only checks that the configuration can be read.
func (a ConversationsConfigurationArgs) apply(ctx context.Context, verb string) error {
	c, err := clientFor(ctx)
	if err != nil {
		return err
	}
	form, method := a.form(), "POST"
	if len(form) == 0 {
		method, form = "GET", nil
	}
	if _, err := c.do(ctx, method, cvProduct, cvPath("", "Configuration", ""), nil, form); err != nil {
		return fmt.Errorf("%s Conversations configuration: %w", verb, err)
	}
	return nil
}

func (r *ConversationsConfiguration) Create(ctx context.Context, req infer.CreateRequest[ConversationsConfigurationArgs]) (infer.CreateResponse[ConversationsConfigurationState], error) {
	type resp = infer.CreateResponse[ConversationsConfigurationState]
	st := ConversationsConfigurationState{ConversationsConfigurationArgs: req.Inputs}
	if req.DryRun {
		return resp{Output: st}, nil
	}
	if err := req.Inputs.apply(ctx, "configure"); err != nil {
		return resp{}, err
	}
	return resp{ID: cvAccountConfigurationID, Output: st}, nil
}

func (r *ConversationsConfiguration) Read(ctx context.Context, req infer.ReadRequest[ConversationsConfigurationArgs, ConversationsConfigurationState]) (infer.ReadResponse[ConversationsConfigurationArgs, ConversationsConfigurationState], error) {
	type resp = infer.ReadResponse[ConversationsConfigurationArgs, ConversationsConfigurationState]
	if req.ID != cvAccountConfigurationID {
		return resp{}, fmt.Errorf("cannot import Conversations configuration %q: the import ID is %q", req.ID, cvAccountConfigurationID)
	}
	c, err := clientFor(ctx)
	if err != nil {
		return resp{}, err
	}
	o, err := c.do(ctx, "GET", cvProduct, cvPath("", "Configuration", ""), nil, nil)
	if err != nil {
		if isNotFound(err) {
			if err := notFoundOnImport("Conversations configuration", req.ID, req.Inputs, req.State); err != nil {
				return resp{}, err
			}
			return resp{}, nil
		}
		return resp{}, fmt.Errorf("read Conversations configuration: %w", err)
	}
	in := req.Inputs
	// An import arrives with no prior state; the program's own settings are the only record of what it manages.
	imported := reflect.ValueOf(req.State).IsZero()
	in.DefaultChatServiceSid = trStr(in.DefaultChatServiceSid, o.optStr("default_chat_service_sid"), conversationsConfigurationDefaults.str("defaultChatServiceSid"), imported)
	in.DefaultMessagingServiceSid = trStr(in.DefaultMessagingServiceSid, o.optStr("default_messaging_service_sid"), conversationsConfigurationDefaults.str("defaultMessagingServiceSid"), imported)
	in.DefaultInactiveTimer = trStr(in.DefaultInactiveTimer, o.optStr("default_inactive_timer"), conversationsConfigurationDefaults.str("defaultInactiveTimer"), imported)
	in.DefaultClosedTimer = trStr(in.DefaultClosedTimer, o.optStr("default_closed_timer"), conversationsConfigurationDefaults.str("defaultClosedTimer"), imported)
	return resp{ID: req.ID, Inputs: in, State: ConversationsConfigurationState{ConversationsConfigurationArgs: in}}, nil
}

func (r *ConversationsConfiguration) Update(ctx context.Context, req infer.UpdateRequest[ConversationsConfigurationArgs, ConversationsConfigurationState]) (infer.UpdateResponse[ConversationsConfigurationState], error) {
	st := ConversationsConfigurationState{ConversationsConfigurationArgs: req.Inputs}
	if req.DryRun {
		return infer.UpdateResponse[ConversationsConfigurationState]{Output: st}, nil
	}
	if err := req.Inputs.apply(ctx, "update"); err != nil {
		return infer.UpdateResponse[ConversationsConfigurationState]{}, err
	}
	return infer.UpdateResponse[ConversationsConfigurationState]{Output: st}, nil
}

// Delete forgets the configuration: an account always has one and the spec offers no way to remove it.
func (r *ConversationsConfiguration) Delete(_ context.Context, _ infer.DeleteRequest[ConversationsConfigurationState]) (infer.DeleteResponse, error) {
	return infer.DeleteResponse{}, nil
}

// conversationsConfigurationDefaults are the Twilio defaults of this resource, shared by Read (import adoption) and Diff.
var conversationsConfigurationDefaults = trDefaults{
	"defaultChatServiceSid":      "",
	"defaultMessagingServiceSid": "",
	"defaultInactiveTimer":       "",
	"defaultClosedTimer":         "",
}
