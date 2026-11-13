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

	"github.com/pulumi/pulumi-go-provider/infer"
)

// cvAccountWebhookID is the resource ID of the account-level webhook configuration.
const cvAccountWebhookID = "account"

// ConversationsWebhookConfiguration manages the webhook configuration of Conversations: the
// account-level one, or that of one service when chatServiceSid is set. There is exactly one of
// each, so this adopts it; destroying the resource forgets it and leaves the settings as they are.
type ConversationsWebhookConfiguration struct{}

// ConversationsWebhookConfigurationArgs are the inputs of a ConversationsWebhookConfiguration.
type ConversationsWebhookConfigurationArgs struct {
	ChatServiceSid *string   `pulumi:"chatServiceSid,optional" provider:"replaceOnChanges"`
	Method         *string   `pulumi:"method,optional"`
	Filters        *[]string `pulumi:"filters,optional"`
	PreWebhookUrl  *string   `pulumi:"preWebhookUrl,optional"`
	PostWebhookUrl *string   `pulumi:"postWebhookUrl,optional"`
	Target         *string   `pulumi:"target,optional"`
}

func (a *ConversationsWebhookConfigurationArgs) Annotate(an infer.Annotator) {
	an.Describe(&a.ChatServiceSid, "SID (IS...) of the Conversations service whose webhook configuration this is. Leave it out for the account-level configuration. Changing it replaces the resource.")
	an.Describe(&a.Method, "HTTP method of the webhook requests, GET or POST (any case). Removing it leaves the setting unchanged.")
	an.Describe(&a.Filters, "Events that fire the webhooks, for example onMessageAdded or onConversationUpdated; the order is not significant. Removing it leaves the setting unchanged.")
	an.Describe(&a.PreWebhookUrl, "Absolute URL the pre-event webhook request is sent to. Removing it leaves the setting unchanged.")
	an.Describe(&a.PostWebhookUrl, "Absolute URL the post-event webhook request is sent to. Removing it leaves the setting unchanged.")
	an.Describe(&a.Target, "Where the webhooks are sent, webhook or flex. Account-level configuration only. Removing it leaves the setting unchanged.")
}

// ConversationsWebhookConfigurationState is the persisted state of a ConversationsWebhookConfiguration.
type ConversationsWebhookConfigurationState struct {
	ConversationsWebhookConfigurationArgs
}

func (r *ConversationsWebhookConfiguration) Annotate(a infer.Annotator) {
	a.Describe(&r, "The webhook configuration of Conversations, account-level or of one service. The resource ID is the service SID, or account for the account-level one. Destroying it forgets it and leaves the settings on Twilio as they are.")
}

var _ infer.CustomResource[ConversationsWebhookConfigurationArgs, ConversationsWebhookConfigurationState] = (*ConversationsWebhookConfiguration)(nil)
var _ infer.CustomCheck[ConversationsWebhookConfigurationArgs] = (*ConversationsWebhookConfiguration)(nil)
var _ infer.CustomDiff[ConversationsWebhookConfigurationArgs, ConversationsWebhookConfigurationState] = (*ConversationsWebhookConfiguration)(nil)
var _ infer.CustomRead[ConversationsWebhookConfigurationArgs, ConversationsWebhookConfigurationState] = (*ConversationsWebhookConfiguration)(nil)
var _ infer.CustomUpdate[ConversationsWebhookConfigurationArgs, ConversationsWebhookConfigurationState] = (*ConversationsWebhookConfiguration)(nil)
var _ infer.CustomDelete[ConversationsWebhookConfigurationState] = (*ConversationsWebhookConfiguration)(nil)

func (a ConversationsWebhookConfigurationArgs) service() string { return derefOr(a.ChatServiceSid, "") }

func (a ConversationsWebhookConfigurationArgs) id() string {
	if a.ChatServiceSid == nil {
		return cvAccountWebhookID
	}
	return *a.ChatServiceSid
}

// cvWebhookPath maps a resource ID to the API path.
func cvWebhookPath(id string) string {
	if id == cvAccountWebhookID {
		return cvPath("", "Configuration", "Webhooks")
	}
	return cvPath(id, "Configuration", "Webhooks")
}

func (r *ConversationsWebhookConfiguration) Check(ctx context.Context, req infer.CheckRequest) (infer.CheckResponse[ConversationsWebhookConfigurationArgs], error) {
	args, failures, err := infer.DefaultCheck[ConversationsWebhookConfigurationArgs](ctx, req.NewInputs)
	if err != nil || len(failures) > 0 {
		return infer.CheckResponse[ConversationsWebhookConfigurationArgs]{Inputs: args, Failures: failures}, err
	}
	c := &checker{unk: unknowns(req.NewInputs)}
	cvCheckOptionalService(c, "chatServiceSid", args.ChatServiceSid)
	cvUpper(c, "method", args.Method)
	if v := args.Target; v != nil {
		c.oneOf("target", *v, []string{"webhook", "flex"})
		if args.ChatServiceSid != nil && !c.unk("chatServiceSid", "target") {
			c.fail("target", "target applies to the account-level configuration only; remove it or remove chatServiceSid")
		}
	}
	return infer.CheckResponse[ConversationsWebhookConfigurationArgs]{Inputs: args, Failures: c.failures}, nil
}

func (r *ConversationsWebhookConfiguration) Diff(_ context.Context, req infer.DiffRequest[ConversationsWebhookConfigurationArgs, ConversationsWebhookConfigurationState]) (infer.DiffResponse, error) {
	o, n := req.State.ConversationsWebhookConfigurationArgs, req.Inputs
	d := trDiffSet{}
	if o.id() != n.id() {
		d.replace("chatServiceSid", o.id(), n.id())
	}
	d.strDef(conversationsWebhookConfigurationDefaults, "method", o.Method, n.Method)
	d.strList("filters", o.Filters, n.Filters)
	d.strDef(conversationsWebhookConfigurationDefaults, "preWebhookUrl", o.PreWebhookUrl, n.PreWebhookUrl)
	d.strDef(conversationsWebhookConfigurationDefaults, "postWebhookUrl", o.PostWebhookUrl, n.PostWebhookUrl)
	d.strDef(conversationsWebhookConfigurationDefaults, "target", o.Target, n.Target)
	return d.response(), nil
}

func (a ConversationsWebhookConfigurationArgs) form() url.Values {
	form := url.Values{}
	setS(form, "Method", a.Method)
	cvAddList(form, "Filters", a.Filters)
	setS(form, "PreWebhookUrl", a.PreWebhookUrl)
	setS(form, "PostWebhookUrl", a.PostWebhookUrl)
	setS(form, "Target", a.Target)
	return form
}

// apply writes the settings the program states; with none it only checks that the configuration exists.
func (a ConversationsWebhookConfigurationArgs) apply(ctx context.Context, verb string) error {
	c, err := clientFor(ctx)
	if err != nil {
		return err
	}
	form, method := a.form(), "POST"
	if len(form) == 0 {
		method, form = "GET", nil
	}
	if _, err := c.do(ctx, method, cvProduct, cvWebhookPath(a.id()), nil, form); err != nil {
		return fmt.Errorf("%s Conversations webhook configuration %s: %w", verb, a.id(), err)
	}
	return nil
}

func (r *ConversationsWebhookConfiguration) Create(ctx context.Context, req infer.CreateRequest[ConversationsWebhookConfigurationArgs]) (infer.CreateResponse[ConversationsWebhookConfigurationState], error) {
	type resp = infer.CreateResponse[ConversationsWebhookConfigurationState]
	a := req.Inputs
	st := ConversationsWebhookConfigurationState{ConversationsWebhookConfigurationArgs: a}
	if req.DryRun {
		return resp{Output: st}, nil
	}
	if err := a.apply(ctx, "configure"); err != nil {
		return resp{}, err
	}
	return resp{ID: a.id(), Output: st}, nil
}

func (r *ConversationsWebhookConfiguration) Read(ctx context.Context, req infer.ReadRequest[ConversationsWebhookConfigurationArgs, ConversationsWebhookConfigurationState]) (infer.ReadResponse[ConversationsWebhookConfigurationArgs, ConversationsWebhookConfigurationState], error) {
	type resp = infer.ReadResponse[ConversationsWebhookConfigurationArgs, ConversationsWebhookConfigurationState]
	c, err := clientFor(ctx)
	if err != nil {
		return resp{}, err
	}
	o, err := c.do(ctx, "GET", cvProduct, cvWebhookPath(req.ID), nil, nil)
	if err != nil {
		if isNotFound(err) {
			if err := notFoundOnImport("Conversations webhook configuration", req.ID, req.Inputs, req.State); err != nil {
				return resp{}, err
			}
			return resp{}, nil
		}
		return resp{}, fmt.Errorf("read Conversations webhook configuration %s: %w", req.ID, err)
	}
	in := req.Inputs
	// An import arrives with no prior state; the program's own settings are the only record of what it manages.
	imported := reflect.ValueOf(req.State).IsZero()
	if req.ID != cvAccountWebhookID {
		id := req.ID
		in.ChatServiceSid = &id
	}
	in.Method = trStr(in.Method, o.optStr("method"), conversationsWebhookConfigurationDefaults.str("method"), imported)
	in.Filters = cvList(in.Filters, o.optList("filters"), imported)
	in.PreWebhookUrl = trStr(in.PreWebhookUrl, o.optStr("pre_webhook_url"), conversationsWebhookConfigurationDefaults.str("preWebhookUrl"), imported)
	in.PostWebhookUrl = trStr(in.PostWebhookUrl, o.optStr("post_webhook_url"), conversationsWebhookConfigurationDefaults.str("postWebhookUrl"), imported)
	if req.ID == cvAccountWebhookID {
		in.Target = trStr(in.Target, o.optStr("target"), conversationsWebhookConfigurationDefaults.str("target"), imported)
	}
	return resp{ID: req.ID, Inputs: in, State: ConversationsWebhookConfigurationState{ConversationsWebhookConfigurationArgs: in}}, nil
}

func (r *ConversationsWebhookConfiguration) Update(ctx context.Context, req infer.UpdateRequest[ConversationsWebhookConfigurationArgs, ConversationsWebhookConfigurationState]) (infer.UpdateResponse[ConversationsWebhookConfigurationState], error) {
	st := ConversationsWebhookConfigurationState{ConversationsWebhookConfigurationArgs: req.Inputs}
	if req.DryRun {
		return infer.UpdateResponse[ConversationsWebhookConfigurationState]{Output: st}, nil
	}
	if err := req.Inputs.apply(ctx, "update"); err != nil {
		return infer.UpdateResponse[ConversationsWebhookConfigurationState]{}, err
	}
	return infer.UpdateResponse[ConversationsWebhookConfigurationState]{Output: st}, nil
}

// Delete forgets the configuration: there is always one and the spec offers no way to remove it.
func (r *ConversationsWebhookConfiguration) Delete(_ context.Context, _ infer.DeleteRequest[ConversationsWebhookConfigurationState]) (infer.DeleteResponse, error) {
	return infer.DeleteResponse{}, nil
}

// conversationsWebhookConfigurationDefaults are the Twilio defaults of this resource, shared by Read (import adoption) and Diff.
var conversationsWebhookConfigurationDefaults = trDefaults{
	"method":         "",
	"preWebhookUrl":  "",
	"postWebhookUrl": "",
	"target":         "",
}
