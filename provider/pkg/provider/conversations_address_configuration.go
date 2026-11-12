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
	"strconv"

	p "github.com/pulumi/pulumi-go-provider"
	"github.com/pulumi/pulumi-go-provider/infer"
)

// ConversationsAddressConfiguration manages an address configuration of Conversations: how a
// phone number, WhatsApp sender or other address is wired to conversations, in particular whether
// an inbound message to it creates a conversation automatically.
type ConversationsAddressConfiguration struct{}

// ConversationsAutoCreation is the auto-creation setting of an address configuration.
type ConversationsAutoCreation struct {
	Enabled                bool      `pulumi:"enabled"`
	Type                   *string   `pulumi:"type,optional"`
	ConversationServiceSid *string   `pulumi:"conversationServiceSid,optional"`
	WebhookUrl             *string   `pulumi:"webhookUrl,optional"`
	WebhookMethod          *string   `pulumi:"webhookMethod,optional"`
	WebhookFilters         *[]string `pulumi:"webhookFilters,optional"`
	StudioFlowSid          *string   `pulumi:"studioFlowSid,optional"`
	StudioRetryCount       *int      `pulumi:"studioRetryCount,optional"`
}

func (a *ConversationsAutoCreation) Annotate(an infer.Annotator) {
	an.Describe(&a.Enabled, "Whether an inbound message to the address creates a conversation automatically.")
	an.Describe(&a.Type, "How the auto-created conversation is handled: webhook, studio or default.")
	an.Describe(&a.ConversationServiceSid, "SID (IS...) of the service the conversation is created in; the default service when not set.")
	an.Describe(&a.WebhookUrl, "For type webhook, the URL the webhook request is sent to.")
	an.Describe(&a.WebhookMethod, "For type webhook, the HTTP method, GET or POST (any case).")
	an.Describe(&a.WebhookFilters, "For type webhook, the events that fire the webhook, for example onMessageAdded; the order is not significant.")
	an.Describe(&a.StudioFlowSid, "For type studio, the SID (FW...) of the Studio flow the webhook is sent to.")
	an.Describe(&a.StudioRetryCount, "For type studio, how many times to retry the webhook request.")
}

// ConversationsAddressConfigurationArgs are the inputs of a ConversationsAddressConfiguration.
type ConversationsAddressConfigurationArgs struct {
	AddressType    string                     `pulumi:"addressType" provider:"replaceOnChanges"`
	Address        string                     `pulumi:"address" provider:"replaceOnChanges"`
	FriendlyName   *string                    `pulumi:"friendlyName,optional"`
	AddressCountry *string                    `pulumi:"addressCountry,optional" provider:"replaceOnChanges"`
	AutoCreation   *ConversationsAutoCreation `pulumi:"autoCreation,optional"`
}

var cvAddressTypes = []string{"sms", "whatsapp", "messenger", "gbm", "email", "rcs", "apple", "chat"}

func (a *ConversationsAddressConfigurationArgs) Annotate(an infer.Annotator) {
	an.Describe(&a.AddressType, "Channel of the address: sms, whatsapp, messenger, gbm, email, rcs, apple or chat. Changing it replaces the configuration.")
	an.Describe(&a.Address, "The address to configure: a phone number, a WhatsApp address such as whatsapp:+15551234567, or another channel address. Changing it replaces the configuration.")
	an.Describe(&a.FriendlyName, "Human-readable name of the configuration, up to 256 characters. Removing it leaves the name unchanged.")
	an.Describe(&a.AddressCountry, "ISO 3166-1 alpha-2 country code of the address; only applies to short codes. Changing it replaces the configuration.")
	an.Describe(&a.AutoCreation, "Whether and how a conversation is created for an inbound message to the address. Removing it leaves the setting unchanged.")
}

// ConversationsAddressConfigurationState is the persisted state of a ConversationsAddressConfiguration.
type ConversationsAddressConfigurationState struct {
	ConversationsAddressConfigurationArgs

	Sid         string `pulumi:"sid"`
	DateCreated string `pulumi:"dateCreated"`
	DateUpdated string `pulumi:"dateUpdated"`
}

func (s *ConversationsAddressConfigurationState) Annotate(a infer.Annotator) {
	a.Describe(&s.Sid, "The address configuration SID (IG...).")
	a.Describe(&s.DateCreated, "When the configuration was created (ISO 8601).")
	a.Describe(&s.DateUpdated, "When the configuration was last updated (ISO 8601).")
}

func (r *ConversationsAddressConfiguration) Annotate(a infer.Annotator) {
	a.Describe(&r, "An address configuration of Conversations. The resource ID is the configuration SID.")
}

var _ infer.CustomResource[ConversationsAddressConfigurationArgs, ConversationsAddressConfigurationState] = (*ConversationsAddressConfiguration)(nil)
var _ infer.CustomCheck[ConversationsAddressConfigurationArgs] = (*ConversationsAddressConfiguration)(nil)
var _ infer.CustomDiff[ConversationsAddressConfigurationArgs, ConversationsAddressConfigurationState] = (*ConversationsAddressConfiguration)(nil)
var _ infer.CustomRead[ConversationsAddressConfigurationArgs, ConversationsAddressConfigurationState] = (*ConversationsAddressConfiguration)(nil)
var _ infer.CustomUpdate[ConversationsAddressConfigurationArgs, ConversationsAddressConfigurationState] = (*ConversationsAddressConfiguration)(nil)
var _ infer.CustomDelete[ConversationsAddressConfigurationState] = (*ConversationsAddressConfiguration)(nil)

func (r *ConversationsAddressConfiguration) Check(ctx context.Context, req infer.CheckRequest) (infer.CheckResponse[ConversationsAddressConfigurationArgs], error) {
	args, failures, err := infer.DefaultCheck[ConversationsAddressConfigurationArgs](ctx, req.NewInputs)
	if err != nil || len(failures) > 0 {
		return infer.CheckResponse[ConversationsAddressConfigurationArgs]{Inputs: args, Failures: failures}, err
	}
	c := &checker{unk: unknowns(req.NewInputs)}
	c.oneOf("addressType", args.AddressType, cvAddressTypes)
	c.required("address", args.Address)
	if v := args.FriendlyName; v != nil {
		c.maxLen("friendlyName", *v, 256)
	}
	if ac := args.AutoCreation; ac != nil && !c.unk("autoCreation") {
		if ac.Type != nil {
			c.oneOf("autoCreation", *ac.Type, []string{"webhook", "studio", "default"})
		}
		cvCheckOptionalService(c, "autoCreation", ac.ConversationServiceSid)
		cvUpper(c, "autoCreation", ac.WebhookMethod)
	}
	return infer.CheckResponse[ConversationsAddressConfigurationArgs]{Inputs: args, Failures: c.failures}, nil
}

// sameAutoCreation reports whether every field the program sets in n is already set so in o.
func sameAutoCreation(o, n *ConversationsAutoCreation) bool {
	if n == nil {
		return true
	}
	if o == nil || o.Enabled != n.Enabled {
		return false
	}
	same := func(a, b *string) bool { return b == nil || (a != nil && *a == *b) }
	if !same(o.Type, n.Type) || !same(o.ConversationServiceSid, n.ConversationServiceSid) || !same(o.WebhookUrl, n.WebhookUrl) ||
		!same(o.WebhookMethod, n.WebhookMethod) || !same(o.StudioFlowSid, n.StudioFlowSid) {
		return false
	}
	if n.WebhookFilters != nil && (o.WebhookFilters == nil || !sameSet(*o.WebhookFilters, *n.WebhookFilters)) {
		return false
	}
	return n.StudioRetryCount == nil || (o.StudioRetryCount != nil && *o.StudioRetryCount == *n.StudioRetryCount)
}

func (r *ConversationsAddressConfiguration) Diff(_ context.Context, req infer.DiffRequest[ConversationsAddressConfigurationArgs, ConversationsAddressConfigurationState]) (infer.DiffResponse, error) {
	o, n := req.State.ConversationsAddressConfigurationArgs, req.Inputs
	d := trDiffSet{}
	d.replace("addressType", o.AddressType, n.AddressType)
	d.replace("address", o.Address, n.Address)
	if n.AddressCountry != nil && !reflect.DeepEqual(o.AddressCountry, n.AddressCountry) {
		d["addressCountry"] = p.PropertyDiff{Kind: p.UpdateReplace}
	}
	d.str("friendlyName", o.FriendlyName, n.FriendlyName)
	if !sameAutoCreation(o.AutoCreation, n.AutoCreation) {
		d["autoCreation"] = p.PropertyDiff{Kind: p.Update}
	}
	return d.response(), nil
}

// form is the update form; create adds the type, address and country.
func (a ConversationsAddressConfigurationArgs) form() url.Values {
	form := url.Values{}
	setS(form, "FriendlyName", a.FriendlyName)
	if ac := a.AutoCreation; ac != nil {
		form.Set("AutoCreation.Enabled", strconv.FormatBool(ac.Enabled))
		setS(form, "AutoCreation.Type", ac.Type)
		setS(form, "AutoCreation.ConversationServiceSid", ac.ConversationServiceSid)
		setS(form, "AutoCreation.WebhookUrl", ac.WebhookUrl)
		setS(form, "AutoCreation.WebhookMethod", ac.WebhookMethod)
		cvAddList(form, "AutoCreation.WebhookFilters", ac.WebhookFilters)
		setS(form, "AutoCreation.StudioFlowSid", ac.StudioFlowSid)
		setI(form, "AutoCreation.StudioRetryCount", ac.StudioRetryCount)
	}
	return form
}

func (r *ConversationsAddressConfiguration) Create(ctx context.Context, req infer.CreateRequest[ConversationsAddressConfigurationArgs]) (infer.CreateResponse[ConversationsAddressConfigurationState], error) {
	type resp = infer.CreateResponse[ConversationsAddressConfigurationState]
	a := req.Inputs
	if req.DryRun {
		return resp{Output: ConversationsAddressConfigurationState{ConversationsAddressConfigurationArgs: a}}, nil
	}
	c, err := clientFor(ctx)
	if err != nil {
		return resp{}, err
	}
	form := a.form()
	form.Set("Type", a.AddressType)
	form.Set("Address", a.Address)
	setS(form, "AddressCountry", a.AddressCountry)
	o, err := c.do(ctx, "POST", cvProduct, cvPath("", "Configuration/Addresses", ""), nil, form)
	if err != nil {
		return resp{}, fmt.Errorf("create Conversations address configuration for %s: %w", a.Address, err)
	}
	st := ConversationsAddressConfigurationState{ConversationsAddressConfigurationArgs: a}
	st.Sid, st.DateCreated, st.DateUpdated = trSid(o)
	return resp{ID: st.Sid, Output: st}, nil
}

// cvAutoCreation reads the auto-creation setting back: the fields the program states follow Twilio,
// the others stay unset. On import it adopts what Twilio reports when auto-creation is enabled.
func cvAutoCreation(in *ConversationsAutoCreation, o obj, imported bool) *ConversationsAutoCreation {
	enabled := o.optBool("auto_creation.enabled")
	if imported {
		if enabled == nil || !*enabled {
			return nil
		}
		in = &ConversationsAutoCreation{}
	}
	if in == nil {
		return nil
	}
	out := *in
	if enabled != nil {
		out.Enabled = *enabled
	}
	out.Type = trStr(in.Type, o.optStr("auto_creation.type"), "", imported)
	out.ConversationServiceSid = trStr(in.ConversationServiceSid, o.optStr("auto_creation.conversation_service_sid"), "", imported)
	out.WebhookUrl = trStr(in.WebhookUrl, o.optStr("auto_creation.webhook_url"), "", imported)
	out.WebhookMethod = trStr(in.WebhookMethod, o.optStr("auto_creation.webhook_method"), "", imported)
	out.WebhookFilters = cvList(in.WebhookFilters, o.optList("auto_creation.webhook_filters"), imported)
	out.StudioFlowSid = trStr(in.StudioFlowSid, o.optStr("auto_creation.studio_flow_sid"), "", imported)
	out.StudioRetryCount = trInt(in.StudioRetryCount, o.optInt("auto_creation.studio_retry_count"), 0, imported)
	return &out
}

func (r *ConversationsAddressConfiguration) Read(ctx context.Context, req infer.ReadRequest[ConversationsAddressConfigurationArgs, ConversationsAddressConfigurationState]) (infer.ReadResponse[ConversationsAddressConfigurationArgs, ConversationsAddressConfigurationState], error) {
	type resp = infer.ReadResponse[ConversationsAddressConfigurationArgs, ConversationsAddressConfigurationState]
	c, err := clientFor(ctx)
	if err != nil {
		return resp{}, err
	}
	o, err := c.do(ctx, "GET", cvProduct, cvPath("", "Configuration/Addresses", req.ID), nil, nil)
	if err != nil {
		if isNotFound(err) {
			if err := notFoundOnImport("Conversations address configuration", req.ID, req.Inputs, req.State); err != nil {
				return resp{}, err
			}
			return resp{}, nil
		}
		return resp{}, fmt.Errorf("read Conversations address configuration %s: %w", req.ID, err)
	}
	in := req.Inputs
	imported := in.Address == ""
	in.AddressType = o.str("type")
	in.Address = o.str("address")
	in.FriendlyName = trStr(in.FriendlyName, o.optStr("friendly_name"), "", imported)
	in.AddressCountry = trStr(in.AddressCountry, o.optStr("address_country"), "", imported)
	in.AutoCreation = cvAutoCreation(in.AutoCreation, o, imported)
	st := ConversationsAddressConfigurationState{ConversationsAddressConfigurationArgs: in}
	st.Sid, st.DateCreated, st.DateUpdated = trSid(o)
	return resp{ID: req.ID, Inputs: in, State: st}, nil
}

func (r *ConversationsAddressConfiguration) Update(ctx context.Context, req infer.UpdateRequest[ConversationsAddressConfigurationArgs, ConversationsAddressConfigurationState]) (infer.UpdateResponse[ConversationsAddressConfigurationState], error) {
	type resp = infer.UpdateResponse[ConversationsAddressConfigurationState]
	st := req.State
	st.ConversationsAddressConfigurationArgs = req.Inputs
	if req.DryRun {
		return resp{Output: st}, nil
	}
	c, err := clientFor(ctx)
	if err != nil {
		return resp{}, err
	}
	o, err := c.do(ctx, "POST", cvProduct, cvPath("", "Configuration/Addresses", req.ID), nil, req.Inputs.form())
	if err != nil {
		return resp{}, fmt.Errorf("update Conversations address configuration %s: %w", req.ID, err)
	}
	st.DateUpdated = o.str("date_updated")
	return resp{Output: st}, nil
}

func (r *ConversationsAddressConfiguration) Delete(ctx context.Context, req infer.DeleteRequest[ConversationsAddressConfigurationState]) (infer.DeleteResponse, error) {
	c, err := clientFor(ctx)
	if err != nil {
		return infer.DeleteResponse{}, err
	}
	if _, err := c.do(ctx, "DELETE", cvProduct, cvPath("", "Configuration/Addresses", req.ID), nil, nil); err != nil && !isNotFound(err) {
		return infer.DeleteResponse{}, fmt.Errorf("delete Conversations address configuration %s: %w", req.ID, err)
	}
	return infer.DeleteResponse{}, nil
}
