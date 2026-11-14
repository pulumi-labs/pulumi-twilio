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
	"encoding/json"
	"fmt"
	"net/url"
	"regexp"

	p "github.com/pulumi/pulumi-go-provider"
	"github.com/pulumi/pulumi-go-provider/infer"
)

// A2P 10DLC campaigns (messaging.twilio.com v1 /Services/{MessagingServiceSid}/Compliance/Usa2p). A
// campaign (Twilio calls the resource UsAppToPerson) registers one use case of a Brand Registration
// for a Messaging Service. The shapes follow twilio-oai spec/json/twilio_messaging_v1.json (1.0.0).

var brandSidRE = regexp.MustCompile(`^BN[0-9a-fA-F]{32}$`)

func a2pCampaignPath(svc string, rest ...string) string {
	path := serviceBase(svc) + "/Compliance/Usa2p"
	for _, r := range rest {
		path += "/" + url.PathEscape(r)
	}
	return path
}

// UsAppToPerson registers an A2P 10DLC campaign for a Messaging Service.
type UsAppToPerson struct{}

// UsAppToPersonArgs are the inputs of a UsAppToPerson. The description, message flow, message
// samples, link and phone flags, age gate, direct lending flag and the two policy URLs update in
// place; everything else is fixed once the campaign is registered, so changing it replaces the campaign.
type UsAppToPersonArgs struct {
	MessagingServiceSid   string    `pulumi:"messagingServiceSid"`
	BrandRegistrationSid  string    `pulumi:"brandRegistrationSid"`
	UsAppToPersonUsecase  string    `pulumi:"usAppToPersonUsecase"`
	Description           string    `pulumi:"description"`
	MessageFlow           string    `pulumi:"messageFlow"`
	MessageSamples        []string  `pulumi:"messageSamples"`
	HasEmbeddedLinks      bool      `pulumi:"hasEmbeddedLinks"`
	HasEmbeddedPhone      bool      `pulumi:"hasEmbeddedPhone"`
	OptInMessage          *string   `pulumi:"optInMessage,optional"`
	OptOutMessage         *string   `pulumi:"optOutMessage,optional"`
	HelpMessage           *string   `pulumi:"helpMessage,optional"`
	OptInKeywords         *[]string `pulumi:"optInKeywords,optional"`
	OptOutKeywords        *[]string `pulumi:"optOutKeywords,optional"`
	HelpKeywords          *[]string `pulumi:"helpKeywords,optional"`
	SubscriberOptIn       *bool     `pulumi:"subscriberOptIn,optional"`
	AgeGated              *bool     `pulumi:"ageGated,optional"`
	DirectLending         *bool     `pulumi:"directLending,optional"`
	PrivacyPolicyUrl      *string   `pulumi:"privacyPolicyUrl,optional"`
	TermsAndConditionsUrl *string   `pulumi:"termsAndConditionsUrl,optional"`
}

func (a *UsAppToPersonArgs) Annotate(an infer.Annotator) {
	an.Describe(&a.MessagingServiceSid, "SID (MG...) of the Messaging Service the campaign belongs to. Changing it replaces the campaign.")
	an.Describe(&a.BrandRegistrationSid, "SID (BN...) of the approved BrandRegistration the campaign is registered under. Changing it replaces the campaign.")
	an.Describe(&a.UsAppToPersonUsecase, "The A2P campaign use case, for example 2FA, MARKETING, LOW_VOLUME or SOLE_PROPRIETOR; getUsAppToPersonUsecases lists the ones the brand may use. Changing it replaces the campaign.")
	an.Describe(&a.Description, "What the campaign does (Twilio asks for 40 to 4096 characters). Updates in place.")
	an.Describe(&a.MessageFlow, "How a consumer opts in to the campaign and so consents to its messages. Updates in place.")
	an.Describe(&a.MessageSamples, "Sample messages of the campaign (Twilio asks for two to five of 20 to 1024 characters). Order is not significant. Updates in place.")
	an.Describe(&a.HasEmbeddedLinks, "Whether the campaign's messages contain links. Updates in place.")
	an.Describe(&a.HasEmbeddedPhone, "Whether the campaign's messages contain phone numbers. Updates in place.")
	an.Describe(&a.OptInMessage, "The auto-reply sent when an end user texts an opt-in keyword. Changing it replaces the campaign. Once Twilio has reported a value, a program that omits the input keeps it.")
	an.Describe(&a.OptOutMessage, "The auto-reply sent when an end user texts an opt-out keyword. Changing it replaces the campaign. Once Twilio has reported a value, a program that omits the input keeps it.")
	an.Describe(&a.HelpMessage, "The auto-reply sent when an end user texts a help keyword. Changing it replaces the campaign. Once Twilio has reported a value, a program that omits the input keeps it.")
	an.Describe(&a.OptInKeywords, "Keywords an end user texts to opt in. Order is not significant. Changing them replaces the campaign.")
	an.Describe(&a.OptOutKeywords, "Keywords an end user texts to opt out. Order is not significant. Changing them replaces the campaign.")
	an.Describe(&a.HelpKeywords, "Keywords an end user texts to ask for help. Order is not significant. Changing them replaces the campaign.")
	an.Describe(&a.SubscriberOptIn, "Whether the campaign has subscriber opt-in. Changing it replaces the campaign.")
	an.Describe(&a.AgeGated, "Whether the campaign requires an age gate for federally legal content. Updates in place; Twilio treats an unset value as false.")
	an.Describe(&a.DirectLending, "Whether the campaign allows direct lending. Updates in place; Twilio treats an unset value as false.")
	an.Describe(&a.PrivacyPolicyUrl, "The URL of the campaign's privacy policy. Updates in place. Removing it leaves the URL unchanged.")
	an.Describe(&a.TermsAndConditionsUrl, "The URL of the campaign's terms and conditions. Updates in place. Removing it leaves the URL unchanged.")
}

// UsAppToPersonState is the persisted state of a UsAppToPerson.
type UsAppToPersonState struct {
	UsAppToPersonArgs

	Sid                    string   `pulumi:"sid"`
	AccountSid             string   `pulumi:"accountSid"`
	CampaignStatus         string   `pulumi:"campaignStatus"`
	CampaignId             string   `pulumi:"campaignId"`
	IsExternallyRegistered bool     `pulumi:"isExternallyRegistered"`
	RateLimits             string   `pulumi:"rateLimits"`
	Mock                   bool     `pulumi:"mock"`
	Errors                 []string `pulumi:"errors"`
	Url                    string   `pulumi:"url"`
	DateCreated            string   `pulumi:"dateCreated"`
	DateUpdated            string   `pulumi:"dateUpdated"`
}

func (s *UsAppToPersonState) Annotate(a infer.Annotator) {
	a.Describe(&s.Sid, "The campaign SID (QE...).")
	a.Describe(&s.AccountSid, "The SID of the account that owns the campaign.")
	a.Describe(&s.CampaignStatus, "The campaign status, for example IN_PROGRESS, VERIFIED or FAILED. It changes as The Campaign Registry reviews the campaign, so a refresh can show a new value.")
	a.Describe(&s.CampaignId, "The Campaign Registry (TCR) campaign ID (empty until assigned).")
	a.Describe(&s.IsExternallyRegistered, "Whether the campaign was registered outside Twilio.")
	a.Describe(&s.RateLimits, "The per carrier rate limits and classification, as a JSON document (empty until the campaign is verified).")
	a.Describe(&s.Mock, "Whether the campaign is a mock campaign, which Twilio verifies automatically.")
	a.Describe(&s.Errors, "Why the registration failed, each error as a JSON document (empty when there are none).")
	a.Describe(&s.Url, "The absolute URL of the campaign.")
	a.Describe(&s.DateCreated, "When the campaign was created (ISO 8601).")
	a.Describe(&s.DateUpdated, "When the campaign was last updated (ISO 8601).")
}

func (r *UsAppToPerson) Annotate(a infer.Annotator) {
	a.Describe(&r, "An A2P 10DLC campaign (US App to Person) of a Messaging Service, registered under an approved BrandRegistration. The resource ID is <messagingServiceSid>/<sid>. The description, message flow, message samples, link and phone flags, age gate, direct lending flag and policy URLs update in place; anything else replaces the campaign. A campaign is reviewed by The Campaign Registry and billed by Twilio and the registry: its status changes after creation. Destroying the resource deletes the campaign.")
}

var _ infer.CustomResource[UsAppToPersonArgs, UsAppToPersonState] = (*UsAppToPerson)(nil)
var _ infer.CustomCheck[UsAppToPersonArgs] = (*UsAppToPerson)(nil)
var _ infer.CustomDiff[UsAppToPersonArgs, UsAppToPersonState] = (*UsAppToPerson)(nil)
var _ infer.CustomRead[UsAppToPersonArgs, UsAppToPersonState] = (*UsAppToPerson)(nil)
var _ infer.CustomUpdate[UsAppToPersonArgs, UsAppToPersonState] = (*UsAppToPerson)(nil)
var _ infer.CustomDelete[UsAppToPersonState] = (*UsAppToPerson)(nil)

func (r *UsAppToPerson) Check(ctx context.Context, req infer.CheckRequest) (infer.CheckResponse[UsAppToPersonArgs], error) {
	args, failures, err := infer.DefaultCheck[UsAppToPersonArgs](ctx, req.NewInputs)
	if err != nil || len(failures) > 0 {
		return infer.CheckResponse[UsAppToPersonArgs]{Inputs: args, Failures: failures}, err
	}
	c := &checker{unk: unknowns(req.NewInputs)}
	c.sid("messagingServiceSid", args.MessagingServiceSid, verifyMessagingSidRE, "Messaging Service (MG followed by 32 hex characters)")
	c.sid("brandRegistrationSid", args.BrandRegistrationSid, brandSidRE, "Brand Registration (BN followed by 32 hex characters)")
	c.required("usAppToPersonUsecase", args.UsAppToPersonUsecase)
	c.required("description", args.Description)
	c.required("messageFlow", args.MessageFlow)
	if len(args.MessageSamples) == 0 && !c.unk("messageSamples") {
		c.fail("messageSamples", "messageSamples must list at least one sample message")
	}
	return infer.CheckResponse[UsAppToPersonArgs]{Inputs: args, Failures: c.failures}, nil
}

func (r *UsAppToPerson) Diff(_ context.Context, req infer.DiffRequest[UsAppToPersonArgs, UsAppToPersonState]) (infer.DiffResponse, error) {
	o, n := req.State.UsAppToPersonArgs, req.Inputs
	d := trDiffSet{}
	d.replace("messagingServiceSid", o.MessagingServiceSid, n.MessagingServiceSid)
	d.replace("brandRegistrationSid", o.BrandRegistrationSid, n.BrandRegistrationSid)
	d.replace("usAppToPersonUsecase", o.UsAppToPersonUsecase, n.UsAppToPersonUsecase)
	d.replaceStr("optInMessage", o.OptInMessage, n.OptInMessage)
	d.replaceStr("optOutMessage", o.OptOutMessage, n.OptOutMessage)
	d.replaceStr("helpMessage", o.HelpMessage, n.HelpMessage)
	d.replaceList("optInKeywords", o.OptInKeywords, n.OptInKeywords)
	d.replaceList("optOutKeywords", o.OptOutKeywords, n.OptOutKeywords)
	d.replaceList("helpKeywords", o.HelpKeywords, n.HelpKeywords)
	d.replaceBool("subscriberOptIn", o.SubscriberOptIn, n.SubscriberOptIn)
	d.requiredStr("description", o.Description, n.Description)
	d.requiredStr("messageFlow", o.MessageFlow, n.MessageFlow)
	if !sameSet(o.MessageSamples, n.MessageSamples) {
		d["messageSamples"] = p.PropertyDiff{Kind: p.Update}
	}
	if o.HasEmbeddedLinks != n.HasEmbeddedLinks {
		d["hasEmbeddedLinks"] = p.PropertyDiff{Kind: p.Update}
	}
	if o.HasEmbeddedPhone != n.HasEmbeddedPhone {
		d["hasEmbeddedPhone"] = p.PropertyDiff{Kind: p.Update}
	}
	d.boolean("ageGated", flexDefBool(o.AgeGated), n.AgeGated)
	d.boolean("directLending", flexDefBool(o.DirectLending), n.DirectLending)
	d.str("privacyPolicyUrl", o.PrivacyPolicyUrl, n.PrivacyPolicyUrl)
	d.str("termsAndConditionsUrl", o.TermsAndConditionsUrl, n.TermsAndConditionsUrl)
	return d.response(), nil
}

func (a UsAppToPersonArgs) createForm() url.Values {
	form := url.Values{}
	form.Set("BrandRegistrationSid", a.BrandRegistrationSid)
	form.Set("UsAppToPersonUsecase", a.UsAppToPersonUsecase)
	a.commonForm(form)
	setS(form, "OptInMessage", a.OptInMessage)
	setS(form, "OptOutMessage", a.OptOutMessage)
	setS(form, "HelpMessage", a.HelpMessage)
	cvAddList(form, "OptInKeywords", a.OptInKeywords)
	cvAddList(form, "OptOutKeywords", a.OptOutKeywords)
	cvAddList(form, "HelpKeywords", a.HelpKeywords)
	setB(form, "SubscriberOptIn", a.SubscriberOptIn)
	setB(form, "AgeGated", a.AgeGated)
	setB(form, "DirectLending", a.DirectLending)
	return form
}

// commonForm sets what the create and the update call both take.
func (a UsAppToPersonArgs) commonForm(form url.Values) {
	form.Set("Description", a.Description)
	form.Set("MessageFlow", a.MessageFlow)
	for _, m := range a.MessageSamples {
		form.Add("MessageSamples", m)
	}
	setB(form, "HasEmbeddedLinks", &a.HasEmbeddedLinks)
	setB(form, "HasEmbeddedPhone", &a.HasEmbeddedPhone)
	setS(form, "PrivacyPolicyUrl", a.PrivacyPolicyUrl)
	setS(form, "TermsAndConditionsUrl", a.TermsAndConditionsUrl)
}

// updateForm is the update call: Twilio requires the age gate and the direct lending flag on it too.
func (a UsAppToPersonArgs) updateForm() url.Values {
	form := url.Values{}
	a.commonForm(form)
	form.Set("AgeGated", fmt.Sprint(*flexDefBool(a.AgeGated)))
	form.Set("DirectLending", fmt.Sprint(*flexDefBool(a.DirectLending)))
	return form
}

// a2pJSON renders a free-form value as a compact JSON document, empty for null.
func a2pJSON(v any) string {
	if v == nil {
		return ""
	}
	b, err := json.Marshal(v)
	if err != nil {
		return ""
	}
	return string(b)
}

func (s *UsAppToPersonState) fill(o obj) {
	s.Sid, s.AccountSid = o.str("sid"), o.str("account_sid")
	s.CampaignStatus, s.CampaignId = o.str("campaign_status"), o.str("campaign_id")
	s.IsExternallyRegistered, s.Mock = brandBool(o, "is_externally_registered"), brandBool(o, "mock")
	s.RateLimits, s.Errors = a2pJSON(o.get("rate_limits")), brandErrors(o)
	s.Url, s.DateCreated, s.DateUpdated = o.str("url"), o.str("date_created"), o.str("date_updated")
}

func (r *UsAppToPerson) Create(ctx context.Context, req infer.CreateRequest[UsAppToPersonArgs]) (infer.CreateResponse[UsAppToPersonState], error) {
	type resp = infer.CreateResponse[UsAppToPersonState]
	a := req.Inputs
	if req.DryRun {
		return resp{Output: UsAppToPersonState{UsAppToPersonArgs: a, Errors: []string{}}}, nil
	}
	c, err := clientFor(ctx)
	if err != nil {
		return resp{}, err
	}
	o, err := c.do(ctx, "POST", messagingProduct, a2pCampaignPath(a.MessagingServiceSid), nil, a.createForm())
	if err != nil {
		return resp{}, fmt.Errorf("create A2P campaign of Messaging Service %s: %w", a.MessagingServiceSid, err)
	}
	st := UsAppToPersonState{UsAppToPersonArgs: a}
	st.fill(o)
	return resp{ID: a.MessagingServiceSid + "/" + st.Sid, Output: st}, nil
}

// a2pAdoptList is the refreshed value of a list Twilio cannot change after creation: what Twilio
// reports, kept in the program's order when it is the same set, else what the program stated.
func a2pAdoptList(in, api *[]string) *[]string {
	api = nonEmptyList(api)
	switch {
	case api == nil:
		return in
	case in != nil && sameSet(*in, *api):
		return in
	}
	return api
}

func (r *UsAppToPerson) Read(ctx context.Context, req infer.ReadRequest[UsAppToPersonArgs, UsAppToPersonState]) (infer.ReadResponse[UsAppToPersonArgs, UsAppToPersonState], error) {
	type resp = infer.ReadResponse[UsAppToPersonArgs, UsAppToPersonState]
	parts, err := splitVerifyID(req.ID, 2, "<messagingServiceSid>/<sid>")
	if err != nil {
		return resp{}, err
	}
	c, err := clientFor(ctx)
	if err != nil {
		return resp{}, err
	}
	o, err := c.do(ctx, "GET", messagingProduct, a2pCampaignPath(parts[0], parts[1]), nil, nil)
	if err != nil {
		if isNotFound(err) {
			if err := notFoundOnImport("A2P campaign", req.ID, req.Inputs, req.State); err != nil {
				return resp{}, err
			}
			return resp{}, nil
		}
		return resp{}, fmt.Errorf("read A2P campaign %s: %w", req.ID, err)
	}
	in := req.Inputs
	imported := req.State.Sid == ""
	in.MessagingServiceSid = parts[0]
	if v := o.str("brand_registration_sid"); v != "" {
		in.BrandRegistrationSid = v
	}
	if v := o.str("us_app_to_person_usecase"); v != "" {
		in.UsAppToPersonUsecase = v
	}
	if v := o.str("description"); v != "" {
		in.Description = v
	}
	if v := o.str("message_flow"); v != "" {
		in.MessageFlow = v
	}
	if l := o.optList("message_samples"); l != nil && len(*l) > 0 && !sameSet(*l, in.MessageSamples) {
		in.MessageSamples = *l
	}
	if b := o.optBool("has_embedded_links"); b != nil {
		in.HasEmbeddedLinks = *b
	}
	if b := o.optBool("has_embedded_phone"); b != nil {
		in.HasEmbeddedPhone = *b
	}
	in.OptInMessage = adoptAPI(in.OptInMessage, nonEmpty(o.optStr("opt_in_message")))
	in.OptOutMessage = adoptAPI(in.OptOutMessage, nonEmpty(o.optStr("opt_out_message")))
	in.HelpMessage = adoptAPI(in.HelpMessage, nonEmpty(o.optStr("help_message")))
	in.OptInKeywords = a2pAdoptList(in.OptInKeywords, o.optList("opt_in_keywords"))
	in.OptOutKeywords = a2pAdoptList(in.OptOutKeywords, o.optList("opt_out_keywords"))
	in.HelpKeywords = a2pAdoptList(in.HelpKeywords, o.optList("help_keywords"))
	in.SubscriberOptIn = adoptAPI(in.SubscriberOptIn, o.optBool("subscriber_opt_in"))
	in.AgeGated = trBool(in.AgeGated, o.optBool("age_gated"), false, imported)
	in.DirectLending = trBool(in.DirectLending, o.optBool("direct_lending"), false, imported)
	in.PrivacyPolicyUrl = trStr(in.PrivacyPolicyUrl, o.optStr("privacy_policy_url"), "", imported)
	in.TermsAndConditionsUrl = trStr(in.TermsAndConditionsUrl, o.optStr("terms_and_conditions_url"), "", imported)
	st := UsAppToPersonState{UsAppToPersonArgs: in}
	st.fill(o)
	return resp{ID: req.ID, Inputs: in, State: st}, nil
}

func (r *UsAppToPerson) Update(ctx context.Context, req infer.UpdateRequest[UsAppToPersonArgs, UsAppToPersonState]) (infer.UpdateResponse[UsAppToPersonState], error) {
	type resp = infer.UpdateResponse[UsAppToPersonState]
	st := req.State
	st.UsAppToPersonArgs = req.Inputs
	if req.DryRun {
		return resp{Output: st}, nil
	}
	parts, err := splitVerifyID(req.ID, 2, "<messagingServiceSid>/<sid>")
	if err != nil {
		return resp{}, err
	}
	c, err := clientFor(ctx)
	if err != nil {
		return resp{}, err
	}
	o, err := c.do(ctx, "POST", messagingProduct, a2pCampaignPath(parts[0], parts[1]), nil, req.Inputs.updateForm())
	if err != nil {
		return resp{}, fmt.Errorf("update A2P campaign %s: %w", req.ID, err)
	}
	st.fill(o)
	return resp{Output: st}, nil
}

func (r *UsAppToPerson) Delete(ctx context.Context, req infer.DeleteRequest[UsAppToPersonState]) (infer.DeleteResponse, error) {
	parts, err := splitVerifyID(req.ID, 2, "<messagingServiceSid>/<sid>")
	if err != nil {
		return infer.DeleteResponse{}, err
	}
	c, err := clientFor(ctx)
	if err != nil {
		return infer.DeleteResponse{}, err
	}
	if _, err := c.do(ctx, "DELETE", messagingProduct, a2pCampaignPath(parts[0], parts[1]), nil, nil); err != nil && !isNotFound(err) {
		return infer.DeleteResponse{}, fmt.Errorf("delete A2P campaign %s: %w", req.ID, err)
	}
	return infer.DeleteResponse{}, nil
}
