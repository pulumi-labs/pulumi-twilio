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

	p "github.com/pulumi/pulumi-go-provider"
	"github.com/pulumi/pulumi-go-provider/infer"
)

// Toll-free verification (messaging.twilio.com v1 /Tollfree/Verifications): the review that lets a
// toll-free number send messages in the United States and Canada. The shapes follow twilio-oai
// spec/json/twilio_messaging_v1.json (1.0.0).

var tollfreeNumberSidRE = regexp.MustCompile(`^PN[0-9a-fA-F]{32}$`)

func tollfreePath(sid string) string {
	if sid == "" {
		return "/v1/Tollfree/Verifications"
	}
	return "/v1/Tollfree/Verifications/" + url.PathEscape(sid)
}

// TollfreeVerification submits a toll-free number for verification.
type TollfreeVerification struct{}

// TollfreeVerificationArgs are the inputs of a TollfreeVerification. Twilio lets a rejected
// verification be edited and resubmitted, so most inputs update in place; the number, the Customer
// Profile and the external reference are fixed at creation and replace it.
type TollfreeVerificationArgs struct {
	TollfreePhoneNumberSid          string    `pulumi:"tollfreePhoneNumberSid"`
	BusinessName                    string    `pulumi:"businessName"`
	BusinessWebsite                 string    `pulumi:"businessWebsite"`
	NotificationEmail               string    `pulumi:"notificationEmail"`
	UseCaseCategories               []string  `pulumi:"useCaseCategories"`
	UseCaseSummary                  string    `pulumi:"useCaseSummary"`
	ProductionMessageSample         string    `pulumi:"productionMessageSample"`
	OptInImageUrls                  []string  `pulumi:"optInImageUrls"`
	OptInType                       string    `pulumi:"optInType"`
	MessageVolume                   string    `pulumi:"messageVolume"`
	CustomerProfileSid              *string   `pulumi:"customerProfileSid,optional"`
	ExternalReferenceId             *string   `pulumi:"externalReferenceId,optional"`
	BusinessStreetAddress           *string   `pulumi:"businessStreetAddress,optional"`
	BusinessStreetAddress2          *string   `pulumi:"businessStreetAddress2,optional"`
	BusinessCity                    *string   `pulumi:"businessCity,optional"`
	BusinessStateProvinceRegion     *string   `pulumi:"businessStateProvinceRegion,optional"`
	BusinessPostalCode              *string   `pulumi:"businessPostalCode,optional"`
	BusinessCountry                 *string   `pulumi:"businessCountry,optional"`
	AdditionalInformation           *string   `pulumi:"additionalInformation,optional"`
	BusinessContactFirstName        *string   `pulumi:"businessContactFirstName,optional"`
	BusinessContactLastName         *string   `pulumi:"businessContactLastName,optional"`
	BusinessContactEmail            *string   `pulumi:"businessContactEmail,optional"`
	BusinessContactPhone            *string   `pulumi:"businessContactPhone,optional"`
	BusinessRegistrationNumber      *string   `pulumi:"businessRegistrationNumber,optional"`
	BusinessRegistrationAuthority   *string   `pulumi:"businessRegistrationAuthority,optional"`
	BusinessRegistrationCountry     *string   `pulumi:"businessRegistrationCountry,optional"`
	BusinessType                    *string   `pulumi:"businessType,optional"`
	BusinessRegistrationPhoneNumber *string   `pulumi:"businessRegistrationPhoneNumber,optional"`
	DoingBusinessAs                 *string   `pulumi:"doingBusinessAs,optional"`
	OptInConfirmationMessage        *string   `pulumi:"optInConfirmationMessage,optional"`
	HelpMessageSample               *string   `pulumi:"helpMessageSample,optional"`
	PrivacyPolicyUrl                *string   `pulumi:"privacyPolicyUrl,optional"`
	TermsAndConditionsUrl           *string   `pulumi:"termsAndConditionsUrl,optional"`
	VettingProvider                 *string   `pulumi:"vettingProvider,optional"`
	VettingId                       *string   `pulumi:"vettingId,optional"`
	AgeGatedContent                 *bool     `pulumi:"ageGatedContent,optional"`
	OptInKeywords                   *[]string `pulumi:"optInKeywords,optional"`
	EditReason                      *string   `pulumi:"editReason,optional"`
}

// tfField describes one string input: its property, the Twilio form parameter and response key,
// and whether it is fixed at creation.
type tfField struct {
	prop, param, key, desc string
	createOnly             bool
	field                  func(a *TollfreeVerificationArgs) **string
}

// tfRequired describes one required string input; all of them update in place.
type tfRequired struct {
	prop, param, key, desc string
	field                  func(a *TollfreeVerificationArgs) *string
}

var tfRequiredFields = []tfRequired{
	{"businessName", "BusinessName", "business_name", "The name of the business or organization that uses the toll-free number.", func(a *TollfreeVerificationArgs) *string { return &a.BusinessName }},
	{"businessWebsite", "BusinessWebsite", "business_website", "The website of the business or organization.", func(a *TollfreeVerificationArgs) *string { return &a.BusinessWebsite }},
	{"notificationEmail", "NotificationEmail", "notification_email", "The email address that receives the verification result.", func(a *TollfreeVerificationArgs) *string { return &a.NotificationEmail }},
	{"useCaseSummary", "UseCaseSummary", "use_case_summary", "A summary of how the business uses messaging.", func(a *TollfreeVerificationArgs) *string { return &a.UseCaseSummary }},
	{"productionMessageSample", "ProductionMessageSample", "production_message_sample", "An example of a production message.", func(a *TollfreeVerificationArgs) *string { return &a.ProductionMessageSample }},
	{"optInType", "OptInType", "opt_in_type", "How end users opt in: VERBAL, WEB_FORM, PAPER_FORM, VIA_TEXT, MOBILE_QR_CODE or IMPORT.", func(a *TollfreeVerificationArgs) *string { return &a.OptInType }},
	{"messageVolume", "MessageVolume", "message_volume", "The estimated monthly message volume of the number, for example 1,000 (Twilio takes a volume bracket as text).", func(a *TollfreeVerificationArgs) *string { return &a.MessageVolume }},
}

var tfOptionalFields = []tfField{
	{"customerProfileSid", "CustomerProfileSid", "customer_profile_sid", "SID (BU...) of the Customer Profile bundle of the business. Changing it replaces the verification.", true, func(a *TollfreeVerificationArgs) **string { return &a.CustomerProfileSid }},
	{"externalReferenceId", "ExternalReferenceId", "external_reference_id", "Your own reference ID, echoed back by Twilio. Changing it replaces the verification.", true, func(a *TollfreeVerificationArgs) **string { return &a.ExternalReferenceId }},
	{"businessStreetAddress", "BusinessStreetAddress", "business_street_address", "The street address of the business.", false, func(a *TollfreeVerificationArgs) **string { return &a.BusinessStreetAddress }},
	{"businessStreetAddress2", "BusinessStreetAddress2", "business_street_address2", "The second address line of the business.", false, func(a *TollfreeVerificationArgs) **string { return &a.BusinessStreetAddress2 }},
	{"businessCity", "BusinessCity", "business_city", "The city of the business.", false, func(a *TollfreeVerificationArgs) **string { return &a.BusinessCity }},
	{"businessStateProvinceRegion", "BusinessStateProvinceRegion", "business_state_province_region", "The state, province or region of the business.", false, func(a *TollfreeVerificationArgs) **string { return &a.BusinessStateProvinceRegion }},
	{"businessPostalCode", "BusinessPostalCode", "business_postal_code", "The postal code of the business.", false, func(a *TollfreeVerificationArgs) **string { return &a.BusinessPostalCode }},
	{"businessCountry", "BusinessCountry", "business_country", "The country of the business.", false, func(a *TollfreeVerificationArgs) **string { return &a.BusinessCountry }},
	{"additionalInformation", "AdditionalInformation", "additional_information", "Additional information for the reviewer.", false, func(a *TollfreeVerificationArgs) **string { return &a.AdditionalInformation }},
	{"businessContactFirstName", "BusinessContactFirstName", "business_contact_first_name", "The first name of the business contact.", false, func(a *TollfreeVerificationArgs) **string { return &a.BusinessContactFirstName }},
	{"businessContactLastName", "BusinessContactLastName", "business_contact_last_name", "The last name of the business contact.", false, func(a *TollfreeVerificationArgs) **string { return &a.BusinessContactLastName }},
	{"businessContactEmail", "BusinessContactEmail", "business_contact_email", "The email address of the business contact.", false, func(a *TollfreeVerificationArgs) **string { return &a.BusinessContactEmail }},
	{"businessContactPhone", "BusinessContactPhone", "business_contact_phone", "The E.164 phone number of the business contact.", false, func(a *TollfreeVerificationArgs) **string { return &a.BusinessContactPhone }},
	{"businessRegistrationNumber", "BusinessRegistrationNumber", "business_registration_number", "A legally recognized business registration number (Twilio requires it for every business type except SOLE_PROPRIETOR).", false, func(a *TollfreeVerificationArgs) **string { return &a.BusinessRegistrationNumber }},
	{"businessRegistrationAuthority", "BusinessRegistrationAuthority", "business_registration_authority", "The authority that issued the registration number, for example EIN, CBN, VAT or OTHER.", false, func(a *TollfreeVerificationArgs) **string { return &a.BusinessRegistrationAuthority }},
	{"businessRegistrationCountry", "BusinessRegistrationCountry", "business_registration_country", "The country the business is registered in.", false, func(a *TollfreeVerificationArgs) **string { return &a.BusinessRegistrationCountry }},
	{"businessType", "BusinessType", "business_type", "The business type: PRIVATE_PROFIT, PUBLIC_PROFIT, SOLE_PROPRIETOR, NON_PROFIT or GOVERNMENT.", false, func(a *TollfreeVerificationArgs) **string { return &a.BusinessType }},
	{"businessRegistrationPhoneNumber", "BusinessRegistrationPhoneNumber", "business_registration_phone_number", "The E.164 phone number registered with the business.", false, func(a *TollfreeVerificationArgs) **string { return &a.BusinessRegistrationPhoneNumber }},
	{"doingBusinessAs", "DoingBusinessAs", "doing_business_as", "The trade name or downstream business name submitted for verification.", false, func(a *TollfreeVerificationArgs) **string { return &a.DoingBusinessAs }},
	{"optInConfirmationMessage", "OptInConfirmationMessage", "opt_in_confirmation_message", "The confirmation message sent to end users when they opt in.", false, func(a *TollfreeVerificationArgs) **string { return &a.OptInConfirmationMessage }},
	{"helpMessageSample", "HelpMessageSample", "help_message_sample", "A sample help message.", false, func(a *TollfreeVerificationArgs) **string { return &a.HelpMessageSample }},
	{"privacyPolicyUrl", "PrivacyPolicyUrl", "privacy_policy_url", "The URL of the privacy policy.", false, func(a *TollfreeVerificationArgs) **string { return &a.PrivacyPolicyUrl }},
	{"termsAndConditionsUrl", "TermsAndConditionsUrl", "terms_and_conditions_url", "The URL of the terms and conditions.", false, func(a *TollfreeVerificationArgs) **string { return &a.TermsAndConditionsUrl }},
	{"vettingProvider", "VettingProvider", "vetting_provider", "The third party vetting provider, CAMPAIGN_VERIFY.", false, func(a *TollfreeVerificationArgs) **string { return &a.VettingProvider }},
	{"vettingId", "VettingId", "vetting_id", "The ID of the vetting at the vetting provider.", false, func(a *TollfreeVerificationArgs) **string { return &a.VettingId }},
}

func (a *TollfreeVerificationArgs) Annotate(an infer.Annotator) {
	an.Describe(&a.TollfreePhoneNumberSid, "SID (PN...) of the toll-free phone number to verify. Changing it replaces the verification.")
	an.Describe(&a.UseCaseCategories, "The use case categories of the messaging, for example 2FA or MARKETING. Updates in place.")
	an.Describe(&a.OptInImageUrls, "Publicly hosted URLs of images that show the opt-in workflow. Updates in place.")
	an.Describe(&a.AgeGatedContent, "Whether the content is age gated (Twilio default false). Updates in place.")
	an.Describe(&a.OptInKeywords, "Keywords end users text to opt in. Order is not significant. Updates in place.")
	an.Describe(&a.EditReason, "Why a rejected verification is being edited, for example Website fixed. Sent with every update, never refreshed and never a reason to update on its own. Twilio only accepts an edit while editAllowed is true.")
	for _, f := range tfRequiredFields {
		an.Describe(f.field(a), f.desc+" Updates in place.")
	}
	for _, f := range tfOptionalFields {
		d := f.desc
		if !f.createOnly {
			d += " Updates in place. Removing it leaves the value unchanged."
		}
		an.Describe(f.field(a), d)
	}
}

// TollfreeVerificationState is the persisted state of a TollfreeVerification.
type TollfreeVerificationState struct {
	TollfreeVerificationArgs

	Sid                 string   `pulumi:"sid"`
	AccountSid          string   `pulumi:"accountSid"`
	Status              string   `pulumi:"status"`
	TollfreePhoneNumber string   `pulumi:"tollfreePhoneNumber"`
	TrustProductSid     string   `pulumi:"trustProductSid"`
	RegulatedItemSid    string   `pulumi:"regulatedItemSid"`
	RejectionReason     string   `pulumi:"rejectionReason"`
	RejectionReasons    []string `pulumi:"rejectionReasons"`
	ErrorCode           int      `pulumi:"errorCode"`
	EditAllowed         bool     `pulumi:"editAllowed"`
	EditExpiration      string   `pulumi:"editExpiration"`
	ResourceLinks       string   `pulumi:"resourceLinks"`
	VettingIdExpiration string   `pulumi:"vettingIdExpiration"`
	Url                 string   `pulumi:"url"`
	DateCreated         string   `pulumi:"dateCreated"`
	DateUpdated         string   `pulumi:"dateUpdated"`
}

func (s *TollfreeVerificationState) Annotate(a infer.Annotator) {
	a.Describe(&s.Sid, "The verification SID (HH...).")
	a.Describe(&s.AccountSid, "The SID of the account that created the verification.")
	a.Describe(&s.Status, "The review status: PENDING_REVIEW, IN_REVIEW, TWILIO_APPROVED or TWILIO_REJECTED. It changes as Twilio reviews, so a refresh can show a new value.")
	a.Describe(&s.TollfreePhoneNumber, "The E.164 toll-free number being verified.")
	a.Describe(&s.TrustProductSid, "SID of the toll-free Trust Product bundle (empty when none).")
	a.Describe(&s.RegulatedItemSid, "SID of the regulated item (empty when none).")
	a.Describe(&s.RejectionReason, "Why the verification was rejected (empty unless it was).")
	a.Describe(&s.RejectionReasons, "The rejection reasons and codes, each as a JSON document.")
	a.Describe(&s.ErrorCode, "The error code of a rejection (0 when none).")
	a.Describe(&s.EditAllowed, "Whether a rejected verification can be edited and resubmitted. Only then does an update succeed.")
	a.Describe(&s.EditExpiration, "When the ability to edit a rejected verification ends (ISO 8601, empty when none).")
	a.Describe(&s.ResourceLinks, "The URLs of the documents of the verification, as a JSON document (empty when none).")
	a.Describe(&s.VettingIdExpiration, "When the vetting ID expires (ISO 8601, empty when none).")
	a.Describe(&s.Url, "The absolute URL of the verification.")
	a.Describe(&s.DateCreated, "When the verification was created (ISO 8601).")
	a.Describe(&s.DateUpdated, "When the verification was last updated (ISO 8601).")
}

func (r *TollfreeVerification) Annotate(a infer.Annotator) {
	a.Describe(&r, "A toll-free verification: the review that lets a toll-free number send messages in the United States and Canada. The resource ID is the verification SID (HH...). The number, the Customer Profile and the external reference replace the verification; everything else is an edit, which Twilio accepts only for a verification it rejected with editAllowed true. A verification is reviewed by Twilio, so its status changes after creation. Destroying the resource deletes the verification.")
}

var _ infer.CustomResource[TollfreeVerificationArgs, TollfreeVerificationState] = (*TollfreeVerification)(nil)
var _ infer.CustomCheck[TollfreeVerificationArgs] = (*TollfreeVerification)(nil)
var _ infer.CustomDiff[TollfreeVerificationArgs, TollfreeVerificationState] = (*TollfreeVerification)(nil)
var _ infer.CustomRead[TollfreeVerificationArgs, TollfreeVerificationState] = (*TollfreeVerification)(nil)
var _ infer.CustomUpdate[TollfreeVerificationArgs, TollfreeVerificationState] = (*TollfreeVerification)(nil)
var _ infer.CustomDelete[TollfreeVerificationState] = (*TollfreeVerification)(nil)

func (r *TollfreeVerification) Check(ctx context.Context, req infer.CheckRequest) (infer.CheckResponse[TollfreeVerificationArgs], error) {
	args, failures, err := infer.DefaultCheck[TollfreeVerificationArgs](ctx, req.NewInputs)
	if err != nil || len(failures) > 0 {
		return infer.CheckResponse[TollfreeVerificationArgs]{Inputs: args, Failures: failures}, err
	}
	c := &checker{unk: unknowns(req.NewInputs)}
	c.sid("tollfreePhoneNumberSid", args.TollfreePhoneNumberSid, tollfreeNumberSidRE, "phone number (PN followed by 32 hex characters)")
	for _, f := range tfRequiredFields {
		c.required(f.prop, *f.field(&args))
	}
	if len(args.UseCaseCategories) == 0 && !c.unk("useCaseCategories") {
		c.fail("useCaseCategories", "useCaseCategories must list at least one category")
	}
	if len(args.OptInImageUrls) == 0 && !c.unk("optInImageUrls") {
		c.fail("optInImageUrls", "optInImageUrls must list at least one image URL")
	}
	return infer.CheckResponse[TollfreeVerificationArgs]{Inputs: args, Failures: c.failures}, nil
}

func (r *TollfreeVerification) Diff(_ context.Context, req infer.DiffRequest[TollfreeVerificationArgs, TollfreeVerificationState]) (infer.DiffResponse, error) {
	o, n := req.State.TollfreeVerificationArgs, req.Inputs
	d := trDiffSet{}
	d.replace("tollfreePhoneNumberSid", o.TollfreePhoneNumberSid, n.TollfreePhoneNumberSid)
	for _, f := range tfRequiredFields {
		d.requiredStr(f.prop, *f.field(&o), *f.field(&n))
	}
	if !sameSet(o.UseCaseCategories, n.UseCaseCategories) {
		d["useCaseCategories"] = p.PropertyDiff{Kind: p.Update}
	}
	if !sameSet(o.OptInImageUrls, n.OptInImageUrls) {
		d["optInImageUrls"] = p.PropertyDiff{Kind: p.Update}
	}
	for _, f := range tfOptionalFields {
		if f.createOnly {
			d.replaceStr(f.prop, *f.field(&o), *f.field(&n))
		} else {
			d.str(f.prop, *f.field(&o), *f.field(&n))
		}
	}
	d.boolean("ageGatedContent", flexDefBool(o.AgeGatedContent), n.AgeGatedContent)
	d.strList("optInKeywords", o.OptInKeywords, n.OptInKeywords)
	return d.response(), nil
}

// form builds the create form (create) or the edit form: the edit takes everything but the number,
// the Customer Profile and the external reference, plus the edit reason.
func (a TollfreeVerificationArgs) form(create bool) url.Values {
	form := url.Values{}
	if create {
		form.Set("TollfreePhoneNumberSid", a.TollfreePhoneNumberSid)
	}
	for _, f := range tfRequiredFields {
		form.Set(f.param, *f.field(&a))
	}
	for _, c := range a.UseCaseCategories {
		form.Add("UseCaseCategories", c)
	}
	for _, u := range a.OptInImageUrls {
		form.Add("OptInImageUrls", u)
	}
	for _, f := range tfOptionalFields {
		if f.createOnly && !create {
			continue
		}
		setS(form, f.param, *f.field(&a))
	}
	setB(form, "AgeGatedContent", a.AgeGatedContent)
	cvAddList(form, "OptInKeywords", a.OptInKeywords)
	if !create {
		setS(form, "EditReason", a.EditReason)
	}
	return form
}

func (s *TollfreeVerificationState) fill(o obj) {
	s.Sid, s.AccountSid, s.Status = o.str("sid"), o.str("account_sid"), o.str("status")
	s.TollfreePhoneNumber, s.TrustProductSid, s.RegulatedItemSid = o.str("tollfree_phone_number"), o.str("trust_product_sid"), o.str("regulated_item_sid")
	s.RejectionReason, s.ErrorCode, s.EditAllowed = o.str("rejection_reason"), brandInt(o, "error_code"), brandBool(o, "edit_allowed")
	s.RejectionReasons = tfJSONList(o, "rejection_reasons")
	s.EditExpiration, s.ResourceLinks = o.str("edit_expiration"), a2pJSON(o.get("resource_links"))
	s.VettingIdExpiration = o.str("vetting_id_expiration")
	s.Url, s.DateCreated, s.DateUpdated = o.str("url"), o.str("date_created"), o.str("date_updated")
}

// tfJSONList renders a free-form list as JSON documents, one per entry.
func tfJSONList(o obj, k string) []string {
	return brandJSONItems(o.get(k))
}

func (r *TollfreeVerification) Create(ctx context.Context, req infer.CreateRequest[TollfreeVerificationArgs]) (infer.CreateResponse[TollfreeVerificationState], error) {
	type resp = infer.CreateResponse[TollfreeVerificationState]
	a := req.Inputs
	if req.DryRun {
		return resp{Output: TollfreeVerificationState{TollfreeVerificationArgs: a, RejectionReasons: []string{}}}, nil
	}
	c, err := clientFor(ctx)
	if err != nil {
		return resp{}, err
	}
	o, err := c.do(ctx, "POST", messagingProduct, tollfreePath(""), nil, a.form(true))
	if err != nil {
		return resp{}, fmt.Errorf("create toll-free verification of %s: %w", a.TollfreePhoneNumberSid, err)
	}
	st := TollfreeVerificationState{TollfreeVerificationArgs: a}
	st.fill(o)
	return resp{ID: st.Sid, Output: st}, nil
}

func (r *TollfreeVerification) Read(ctx context.Context, req infer.ReadRequest[TollfreeVerificationArgs, TollfreeVerificationState]) (infer.ReadResponse[TollfreeVerificationArgs, TollfreeVerificationState], error) {
	type resp = infer.ReadResponse[TollfreeVerificationArgs, TollfreeVerificationState]
	c, err := clientFor(ctx)
	if err != nil {
		return resp{}, err
	}
	o, err := c.do(ctx, "GET", messagingProduct, tollfreePath(req.ID), nil, nil)
	if err != nil {
		if isNotFound(err) {
			if err := notFoundOnImport("toll-free verification", req.ID, req.Inputs, req.State); err != nil {
				return resp{}, err
			}
			return resp{}, nil
		}
		return resp{}, fmt.Errorf("read toll-free verification %s: %w", req.ID, err)
	}
	in := req.Inputs
	imported := req.State.Sid == ""
	if v := o.str("tollfree_phone_number_sid"); v != "" {
		in.TollfreePhoneNumberSid = v
	}
	for _, f := range tfRequiredFields {
		if v := o.str(f.key); v != "" {
			*f.field(&in) = v
		}
	}
	if l := o.optList("use_case_categories"); l != nil && len(*l) > 0 && !sameSet(*l, in.UseCaseCategories) {
		in.UseCaseCategories = *l
	}
	if l := o.optList("opt_in_image_urls"); l != nil && len(*l) > 0 && !sameSet(*l, in.OptInImageUrls) {
		in.OptInImageUrls = *l
	}
	for _, f := range tfOptionalFields {
		fp := f.field(&in)
		if f.createOnly {
			*fp = adoptAPI(*fp, nonEmpty(o.optStr(f.key)))
		} else {
			*fp = trStr(*fp, o.optStr(f.key), "", imported)
		}
	}
	in.AgeGatedContent = trBool(in.AgeGatedContent, o.optBool("age_gated_content"), false, imported)
	in.OptInKeywords = cvList(in.OptInKeywords, o.optList("opt_in_keywords"), imported)
	st := TollfreeVerificationState{TollfreeVerificationArgs: in}
	st.fill(o)
	return resp{ID: req.ID, Inputs: in, State: st}, nil
}

func (r *TollfreeVerification) Update(ctx context.Context, req infer.UpdateRequest[TollfreeVerificationArgs, TollfreeVerificationState]) (infer.UpdateResponse[TollfreeVerificationState], error) {
	type resp = infer.UpdateResponse[TollfreeVerificationState]
	st := req.State
	st.TollfreeVerificationArgs = req.Inputs
	if req.DryRun {
		return resp{Output: st}, nil
	}
	c, err := clientFor(ctx)
	if err != nil {
		return resp{}, err
	}
	o, err := c.do(ctx, "POST", messagingProduct, tollfreePath(req.ID), nil, req.Inputs.form(false))
	if err != nil {
		return resp{}, fmt.Errorf("update toll-free verification %s (Twilio accepts an edit only while editAllowed is true): %w", req.ID, err)
	}
	st.fill(o)
	return resp{Output: st}, nil
}

func (r *TollfreeVerification) Delete(ctx context.Context, req infer.DeleteRequest[TollfreeVerificationState]) (infer.DeleteResponse, error) {
	c, err := clientFor(ctx)
	if err != nil {
		return infer.DeleteResponse{}, err
	}
	if _, err := c.do(ctx, "DELETE", messagingProduct, tollfreePath(req.ID), nil, nil); err != nil && !isNotFound(err) {
		return infer.DeleteResponse{}, fmt.Errorf("delete toll-free verification %s: %w", req.ID, err)
	}
	return infer.DeleteResponse{}, nil
}
