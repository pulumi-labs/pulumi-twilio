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

// Lookups of A2P 10DLC campaigns (messaging.twilio.com v1 /Services/{MessagingServiceSid}/Compliance/Usa2p).

// UsAppToPersonInfo is one A2P campaign as Twilio reports it.
type UsAppToPersonInfo struct {
	Sid                    string   `pulumi:"sid"`
	MessagingServiceSid    string   `pulumi:"messagingServiceSid"`
	BrandRegistrationSid   string   `pulumi:"brandRegistrationSid"`
	UsAppToPersonUsecase   string   `pulumi:"usAppToPersonUsecase"`
	Description            string   `pulumi:"description"`
	MessageFlow            string   `pulumi:"messageFlow"`
	MessageSamples         []string `pulumi:"messageSamples"`
	HasEmbeddedLinks       bool     `pulumi:"hasEmbeddedLinks"`
	HasEmbeddedPhone       bool     `pulumi:"hasEmbeddedPhone"`
	OptInMessage           string   `pulumi:"optInMessage"`
	OptOutMessage          string   `pulumi:"optOutMessage"`
	HelpMessage            string   `pulumi:"helpMessage"`
	OptInKeywords          []string `pulumi:"optInKeywords"`
	OptOutKeywords         []string `pulumi:"optOutKeywords"`
	HelpKeywords           []string `pulumi:"helpKeywords"`
	SubscriberOptIn        bool     `pulumi:"subscriberOptIn"`
	AgeGated               bool     `pulumi:"ageGated"`
	DirectLending          bool     `pulumi:"directLending"`
	PrivacyPolicyUrl       string   `pulumi:"privacyPolicyUrl"`
	TermsAndConditionsUrl  string   `pulumi:"termsAndConditionsUrl"`
	CampaignStatus         string   `pulumi:"campaignStatus"`
	CampaignId             string   `pulumi:"campaignId"`
	IsExternallyRegistered bool     `pulumi:"isExternallyRegistered"`
	RateLimits             string   `pulumi:"rateLimits"`
	Mock                   bool     `pulumi:"mock"`
	Errors                 []string `pulumi:"errors"`
	DateCreated            string   `pulumi:"dateCreated"`
	DateUpdated            string   `pulumi:"dateUpdated"`
}

func (a *UsAppToPersonInfo) Annotate(an infer.Annotator) {
	an.Describe(&a.Sid, "The campaign SID (QE...).")
	an.Describe(&a.MessagingServiceSid, "SID (MG...) of the Messaging Service the campaign belongs to.")
	an.Describe(&a.BrandRegistrationSid, "SID (BN...) of the Brand Registration the campaign is registered under.")
	an.Describe(&a.UsAppToPersonUsecase, "The A2P campaign use case.")
	an.Describe(&a.Description, "What the campaign does.")
	an.Describe(&a.MessageFlow, "How a consumer opts in to the campaign.")
	an.Describe(&a.MessageSamples, "The sample messages of the campaign.")
	an.Describe(&a.HasEmbeddedLinks, "Whether the campaign's messages contain links.")
	an.Describe(&a.HasEmbeddedPhone, "Whether the campaign's messages contain phone numbers.")
	an.Describe(&a.OptInMessage, "The opt-in auto-reply (empty when none).")
	an.Describe(&a.OptOutMessage, "The opt-out auto-reply (empty when none).")
	an.Describe(&a.HelpMessage, "The help auto-reply (empty when none).")
	an.Describe(&a.OptInKeywords, "The opt-in keywords.")
	an.Describe(&a.OptOutKeywords, "The opt-out keywords.")
	an.Describe(&a.HelpKeywords, "The help keywords.")
	an.Describe(&a.SubscriberOptIn, "Whether the campaign has subscriber opt-in.")
	an.Describe(&a.AgeGated, "Whether the campaign requires an age gate.")
	an.Describe(&a.DirectLending, "Whether the campaign allows direct lending.")
	an.Describe(&a.PrivacyPolicyUrl, "The privacy policy URL (empty when none).")
	an.Describe(&a.TermsAndConditionsUrl, "The terms and conditions URL (empty when none).")
	an.Describe(&a.CampaignStatus, "The campaign status, for example IN_PROGRESS, VERIFIED or FAILED.")
	an.Describe(&a.CampaignId, "The Campaign Registry (TCR) campaign ID (empty until assigned).")
	an.Describe(&a.IsExternallyRegistered, "Whether the campaign was registered outside Twilio.")
	an.Describe(&a.RateLimits, "The per carrier rate limits as a JSON document (empty when none).")
	an.Describe(&a.Mock, "Whether the campaign is a mock campaign.")
	an.Describe(&a.Errors, "Why the registration failed, each error as a JSON document.")
	an.Describe(&a.DateCreated, "When the campaign was created (ISO 8601).")
	an.Describe(&a.DateUpdated, "When the campaign was last updated (ISO 8601).")
}

func usAppToPersonInfo(svc string, o obj) UsAppToPersonInfo {
	ms := o.str("messaging_service_sid")
	if ms == "" {
		ms = svc
	}
	return UsAppToPersonInfo{Sid: o.str("sid"), MessagingServiceSid: ms, BrandRegistrationSid: o.str("brand_registration_sid"),
		UsAppToPersonUsecase: o.str("us_app_to_person_usecase"), Description: o.str("description"), MessageFlow: o.str("message_flow"),
		MessageSamples: brandList(o, "message_samples"), HasEmbeddedLinks: brandBool(o, "has_embedded_links"),
		HasEmbeddedPhone: brandBool(o, "has_embedded_phone"), OptInMessage: o.str("opt_in_message"), OptOutMessage: o.str("opt_out_message"),
		HelpMessage: o.str("help_message"), OptInKeywords: brandList(o, "opt_in_keywords"), OptOutKeywords: brandList(o, "opt_out_keywords"),
		HelpKeywords: brandList(o, "help_keywords"), SubscriberOptIn: brandBool(o, "subscriber_opt_in"), AgeGated: brandBool(o, "age_gated"),
		DirectLending: brandBool(o, "direct_lending"), PrivacyPolicyUrl: o.str("privacy_policy_url"),
		TermsAndConditionsUrl: o.str("terms_and_conditions_url"), CampaignStatus: o.str("campaign_status"), CampaignId: o.str("campaign_id"),
		IsExternallyRegistered: brandBool(o, "is_externally_registered"), RateLimits: a2pJSON(o.get("rate_limits")),
		Mock: brandBool(o, "mock"), Errors: brandErrors(o), DateCreated: o.str("date_created"), DateUpdated: o.str("date_updated")}
}

// GetUsAppToPerson looks up an A2P campaign.
type GetUsAppToPerson struct{}

// GetUsAppToPersonArgs are the lookup arguments.
type GetUsAppToPersonArgs struct {
	MessagingServiceSid string `pulumi:"messagingServiceSid"`
	Sid                 string `pulumi:"sid"`
}

func (a *GetUsAppToPersonArgs) Annotate(an infer.Annotator) {
	an.Describe(&a.MessagingServiceSid, "SID (MG...) of the Messaging Service.")
	an.Describe(&a.Sid, "SID (QE...) of the campaign.")
}

func (f *GetUsAppToPerson) Annotate(a infer.Annotator) {
	a.Describe(&f, "Look up an A2P 10DLC campaign (GET /v1/Services/{MessagingServiceSid}/Compliance/Usa2p/{Sid} on messaging.twilio.com).")
}

func (f *GetUsAppToPerson) Invoke(ctx context.Context, req infer.FunctionRequest[GetUsAppToPersonArgs]) (infer.FunctionResponse[UsAppToPersonInfo], error) {
	const fn = "getUsAppToPerson"
	type resp = infer.FunctionResponse[UsAppToPersonInfo]
	if err := requireArgs(fn, [2]string{"messagingServiceSid", req.Input.MessagingServiceSid}, [2]string{"sid", req.Input.Sid}); err != nil {
		return resp{}, err
	}
	c, err := clientFor(ctx)
	if err != nil {
		return resp{}, err
	}
	o, err := c.do(ctx, "GET", messagingProduct, a2pCampaignPath(req.Input.MessagingServiceSid, req.Input.Sid), nil, nil)
	if err != nil {
		return resp{}, lookupFailure(fn, "A2P campaign "+req.Input.Sid, err)
	}
	return resp{Output: usAppToPersonInfo(req.Input.MessagingServiceSid, o)}, nil
}

// GetUsAppToPersons lists the A2P campaigns of a Messaging Service.
type GetUsAppToPersons struct{}

// GetUsAppToPersonsArgs are the lookup arguments.
type GetUsAppToPersonsArgs struct {
	MessagingServiceSid string `pulumi:"messagingServiceSid"`
}

func (a *GetUsAppToPersonsArgs) Annotate(an infer.Annotator) {
	an.Describe(&a.MessagingServiceSid, "SID (MG...) of the Messaging Service.")
}

// GetUsAppToPersonsResult is the list.
type GetUsAppToPersonsResult struct {
	UsAppToPersons []UsAppToPersonInfo `pulumi:"usAppToPersons"`
}

func (a *GetUsAppToPersonsResult) Annotate(an infer.Annotator) {
	an.Describe(&a.UsAppToPersons, "The campaigns, in the order Twilio returns them.")
}

func (f *GetUsAppToPersons) Annotate(a infer.Annotator) {
	a.Describe(&f, "List the A2P 10DLC campaigns of a Messaging Service (GET /v1/Services/{MessagingServiceSid}/Compliance/Usa2p on messaging.twilio.com).")
}

func (f *GetUsAppToPersons) Invoke(ctx context.Context, req infer.FunctionRequest[GetUsAppToPersonsArgs]) (infer.FunctionResponse[GetUsAppToPersonsResult], error) {
	const fn = "getUsAppToPersons"
	none := infer.FunctionResponse[GetUsAppToPersonsResult]{}
	if err := requireArg(fn, "messagingServiceSid", req.Input.MessagingServiceSid); err != nil {
		return none, err
	}
	c, err := clientFor(ctx)
	if err != nil {
		return none, err
	}
	svc := req.Input.MessagingServiceSid
	out := []UsAppToPersonInfo{}
	err = listPages(ctx, c, fn, messagingProduct, a2pCampaignPath(svc), url.Values{"PageSize": {"1000"}}, "compliance", func(o obj) bool {
		out = append(out, usAppToPersonInfo(svc, o))
		return true
	})
	if err != nil {
		return none, lookupFailure(fn, "A2P campaigns of "+svc, err)
	}
	return infer.FunctionResponse[GetUsAppToPersonsResult]{Output: GetUsAppToPersonsResult{UsAppToPersons: out}}, nil
}

// GetUsAppToPersonUsecases lists the A2P campaign use cases a Messaging Service (and brand) may register.
type GetUsAppToPersonUsecases struct{}

// GetUsAppToPersonUsecasesArgs are the lookup arguments.
type GetUsAppToPersonUsecasesArgs struct {
	MessagingServiceSid  string  `pulumi:"messagingServiceSid"`
	BrandRegistrationSid *string `pulumi:"brandRegistrationSid,optional"`
}

func (a *GetUsAppToPersonUsecasesArgs) Annotate(an infer.Annotator) {
	an.Describe(&a.MessagingServiceSid, "SID (MG...) of the Messaging Service.")
	an.Describe(&a.BrandRegistrationSid, "SID (BN...) of a Brand Registration: when set, Twilio returns the use cases available to that brand.")
}

// UsAppToPersonUsecaseInfo is one A2P campaign use case.
type UsAppToPersonUsecaseInfo struct {
	Code                 string `pulumi:"code"`
	Name                 string `pulumi:"name"`
	Description          string `pulumi:"description"`
	PostApprovalRequired bool   `pulumi:"postApprovalRequired"`
}

func (a *UsAppToPersonUsecaseInfo) Annotate(an infer.Annotator) {
	an.Describe(&a.Code, "The use case code to pass as usAppToPersonUsecase, for example 2FA.")
	an.Describe(&a.Name, "The human readable name of the use case.")
	an.Describe(&a.Description, "What the use case is for.")
	an.Describe(&a.PostApprovalRequired, "Whether the use case needs a post approval step.")
}

// GetUsAppToPersonUsecasesResult is the list.
type GetUsAppToPersonUsecasesResult struct {
	Usecases []UsAppToPersonUsecaseInfo `pulumi:"usecases"`
}

func (a *GetUsAppToPersonUsecasesResult) Annotate(an infer.Annotator) {
	an.Describe(&a.Usecases, "The use cases, in the order Twilio returns them.")
}

func (f *GetUsAppToPersonUsecases) Annotate(a infer.Annotator) {
	a.Describe(&f, "List the A2P campaign use cases (GET /v1/Services/{MessagingServiceSid}/Compliance/Usa2p/Usecases on messaging.twilio.com).")
}

func (f *GetUsAppToPersonUsecases) Invoke(ctx context.Context, req infer.FunctionRequest[GetUsAppToPersonUsecasesArgs]) (infer.FunctionResponse[GetUsAppToPersonUsecasesResult], error) {
	const fn = "getUsAppToPersonUsecases"
	none := infer.FunctionResponse[GetUsAppToPersonUsecasesResult]{}
	if err := requireArg(fn, "messagingServiceSid", req.Input.MessagingServiceSid); err != nil {
		return none, err
	}
	c, err := clientFor(ctx)
	if err != nil {
		return none, err
	}
	q := url.Values{}
	setS(q, "BrandRegistrationSid", req.Input.BrandRegistrationSid)
	o, err := c.do(ctx, "GET", messagingProduct, a2pCampaignPath(req.Input.MessagingServiceSid, "Usecases"), q, nil)
	if err != nil {
		return none, lookupFailure(fn, "A2P use cases of "+req.Input.MessagingServiceSid, err)
	}
	out := []UsAppToPersonUsecaseInfo{}
	items, _ := o.get("us_app_to_person_usecases").([]any)
	for _, it := range items {
		m, ok := it.(map[string]any)
		if !ok {
			continue
		}
		u := obj(m)
		out = append(out, UsAppToPersonUsecaseInfo{Code: u.str("code"), Name: u.str("name"), Description: u.str("description"),
			PostApprovalRequired: brandBool(u, "post_approval_required")})
	}
	return infer.FunctionResponse[GetUsAppToPersonUsecasesResult]{Output: GetUsAppToPersonUsecasesResult{Usecases: out}}, nil
}
