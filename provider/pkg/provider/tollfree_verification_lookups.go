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
	"strconv"

	"github.com/pulumi/pulumi-go-provider/infer"
)

// Lookups of toll-free verifications (messaging.twilio.com v1 /Tollfree/Verifications).

// TollfreeVerificationInfo is one toll-free verification as Twilio reports it.
type TollfreeVerificationInfo struct {
	Sid                             string   `pulumi:"sid"`
	AccountSid                      string   `pulumi:"accountSid"`
	Status                          string   `pulumi:"status"`
	TollfreePhoneNumberSid          string   `pulumi:"tollfreePhoneNumberSid"`
	TollfreePhoneNumber             string   `pulumi:"tollfreePhoneNumber"`
	CustomerProfileSid              string   `pulumi:"customerProfileSid"`
	TrustProductSid                 string   `pulumi:"trustProductSid"`
	RegulatedItemSid                string   `pulumi:"regulatedItemSid"`
	ExternalReferenceId             string   `pulumi:"externalReferenceId"`
	BusinessName                    string   `pulumi:"businessName"`
	BusinessWebsite                 string   `pulumi:"businessWebsite"`
	BusinessStreetAddress           string   `pulumi:"businessStreetAddress"`
	BusinessStreetAddress2          string   `pulumi:"businessStreetAddress2"`
	BusinessCity                    string   `pulumi:"businessCity"`
	BusinessStateProvinceRegion     string   `pulumi:"businessStateProvinceRegion"`
	BusinessPostalCode              string   `pulumi:"businessPostalCode"`
	BusinessCountry                 string   `pulumi:"businessCountry"`
	BusinessContactFirstName        string   `pulumi:"businessContactFirstName"`
	BusinessContactLastName         string   `pulumi:"businessContactLastName"`
	BusinessContactEmail            string   `pulumi:"businessContactEmail"`
	BusinessContactPhone            string   `pulumi:"businessContactPhone"`
	BusinessRegistrationNumber      string   `pulumi:"businessRegistrationNumber"`
	BusinessRegistrationAuthority   string   `pulumi:"businessRegistrationAuthority"`
	BusinessRegistrationCountry     string   `pulumi:"businessRegistrationCountry"`
	BusinessType                    string   `pulumi:"businessType"`
	BusinessRegistrationPhoneNumber string   `pulumi:"businessRegistrationPhoneNumber"`
	DoingBusinessAs                 string   `pulumi:"doingBusinessAs"`
	NotificationEmail               string   `pulumi:"notificationEmail"`
	UseCaseSummary                  string   `pulumi:"useCaseSummary"`
	ProductionMessageSample         string   `pulumi:"productionMessageSample"`
	OptInType                       string   `pulumi:"optInType"`
	MessageVolume                   string   `pulumi:"messageVolume"`
	AdditionalInformation           string   `pulumi:"additionalInformation"`
	OptInConfirmationMessage        string   `pulumi:"optInConfirmationMessage"`
	HelpMessageSample               string   `pulumi:"helpMessageSample"`
	PrivacyPolicyUrl                string   `pulumi:"privacyPolicyUrl"`
	TermsAndConditionsUrl           string   `pulumi:"termsAndConditionsUrl"`
	RejectionReason                 string   `pulumi:"rejectionReason"`
	EditExpiration                  string   `pulumi:"editExpiration"`
	VettingProvider                 string   `pulumi:"vettingProvider"`
	VettingId                       string   `pulumi:"vettingId"`
	VettingIdExpiration             string   `pulumi:"vettingIdExpiration"`
	ResourceLinks                   string   `pulumi:"resourceLinks"`
	Url                             string   `pulumi:"url"`
	DateCreated                     string   `pulumi:"dateCreated"`
	DateUpdated                     string   `pulumi:"dateUpdated"`
	UseCaseCategories               []string `pulumi:"useCaseCategories"`
	OptInImageUrls                  []string `pulumi:"optInImageUrls"`
	OptInKeywords                   []string `pulumi:"optInKeywords"`
	RejectionReasons                []string `pulumi:"rejectionReasons"`
	ErrorCode                       int      `pulumi:"errorCode"`
	EditAllowed                     bool     `pulumi:"editAllowed"`
	AgeGatedContent                 bool     `pulumi:"ageGatedContent"`
}

func (a *TollfreeVerificationInfo) Annotate(an infer.Annotator) {
	an.Describe(&a.Sid, "The verification SID (HH...).")
	an.Describe(&a.AccountSid, "The SID of the account that created the verification.")
	an.Describe(&a.Status, "The review status: PENDING_REVIEW, IN_REVIEW, TWILIO_APPROVED or TWILIO_REJECTED.")
	an.Describe(&a.TollfreePhoneNumberSid, "SID (PN...) of the toll-free phone number.")
	an.Describe(&a.TollfreePhoneNumber, "The E.164 toll-free number.")
	an.Describe(&a.CustomerProfileSid, "SID (BU...) of the Customer Profile bundle (empty when none).")
	an.Describe(&a.TrustProductSid, "SID of the toll-free Trust Product bundle (empty when none).")
	an.Describe(&a.RegulatedItemSid, "SID of the regulated item (empty when none).")
	an.Describe(&a.ExternalReferenceId, "Your external reference ID (empty when none).")
	an.Describe(&a.BusinessName, "The business name.")
	an.Describe(&a.BusinessWebsite, "The business website.")
	an.Describe(&a.BusinessStreetAddress, "The business street address (empty when none).")
	an.Describe(&a.BusinessStreetAddress2, "The second address line (empty when none).")
	an.Describe(&a.BusinessCity, "The business city (empty when none).")
	an.Describe(&a.BusinessStateProvinceRegion, "The business state, province or region (empty when none).")
	an.Describe(&a.BusinessPostalCode, "The business postal code (empty when none).")
	an.Describe(&a.BusinessCountry, "The business country (empty when none).")
	an.Describe(&a.BusinessContactFirstName, "The first name of the business contact (empty when none).")
	an.Describe(&a.BusinessContactLastName, "The last name of the business contact (empty when none).")
	an.Describe(&a.BusinessContactEmail, "The email address of the business contact (empty when none).")
	an.Describe(&a.BusinessContactPhone, "The phone number of the business contact (empty when none).")
	an.Describe(&a.BusinessRegistrationNumber, "The business registration number (empty when none).")
	an.Describe(&a.BusinessRegistrationAuthority, "The registration authority (empty when none).")
	an.Describe(&a.BusinessRegistrationCountry, "The registration country (empty when none).")
	an.Describe(&a.BusinessType, "The business type (empty when none).")
	an.Describe(&a.BusinessRegistrationPhoneNumber, "The phone number registered with the business (empty when none).")
	an.Describe(&a.DoingBusinessAs, "The trade name (empty when none).")
	an.Describe(&a.NotificationEmail, "The email address that receives the verification result.")
	an.Describe(&a.UseCaseSummary, "The use case summary.")
	an.Describe(&a.ProductionMessageSample, "The production message sample.")
	an.Describe(&a.OptInType, "How end users opt in.")
	an.Describe(&a.MessageVolume, "The estimated monthly message volume.")
	an.Describe(&a.AdditionalInformation, "Additional information for the reviewer (empty when none).")
	an.Describe(&a.OptInConfirmationMessage, "The opt-in confirmation message (empty when none).")
	an.Describe(&a.HelpMessageSample, "A sample help message (empty when none).")
	an.Describe(&a.PrivacyPolicyUrl, "The privacy policy URL (empty when none).")
	an.Describe(&a.TermsAndConditionsUrl, "The terms and conditions URL (empty when none).")
	an.Describe(&a.RejectionReason, "Why the verification was rejected (empty unless it was).")
	an.Describe(&a.EditExpiration, "When the ability to edit a rejected verification ends (ISO 8601, empty when none).")
	an.Describe(&a.VettingProvider, "The third party vetting provider (empty when none).")
	an.Describe(&a.VettingId, "The vetting ID (empty when none).")
	an.Describe(&a.VettingIdExpiration, "When the vetting ID expires (ISO 8601, empty when none).")
	an.Describe(&a.ResourceLinks, "The document URLs of the verification as a JSON document (empty when none).")
	an.Describe(&a.Url, "The absolute URL of the verification.")
	an.Describe(&a.DateCreated, "When the verification was created (ISO 8601).")
	an.Describe(&a.DateUpdated, "When the verification was last updated (ISO 8601).")
	an.Describe(&a.UseCaseCategories, "The use case categories.")
	an.Describe(&a.OptInImageUrls, "The opt-in workflow image URLs.")
	an.Describe(&a.OptInKeywords, "The opt-in keywords.")
	an.Describe(&a.RejectionReasons, "The rejection reasons and codes, each as a JSON document.")
	an.Describe(&a.ErrorCode, "The error code of a rejection (0 when none).")
	an.Describe(&a.EditAllowed, "Whether a rejected verification can be edited and resubmitted.")
	an.Describe(&a.AgeGatedContent, "Whether the content is age gated.")
}

func tollfreeVerificationInfo(o obj) TollfreeVerificationInfo {
	return TollfreeVerificationInfo{
		Sid:                             o.str("sid"),
		AccountSid:                      o.str("account_sid"),
		Status:                          o.str("status"),
		TollfreePhoneNumberSid:          o.str("tollfree_phone_number_sid"),
		TollfreePhoneNumber:             o.str("tollfree_phone_number"),
		CustomerProfileSid:              o.str("customer_profile_sid"),
		TrustProductSid:                 o.str("trust_product_sid"),
		RegulatedItemSid:                o.str("regulated_item_sid"),
		ExternalReferenceId:             o.str("external_reference_id"),
		BusinessName:                    o.str("business_name"),
		BusinessWebsite:                 o.str("business_website"),
		BusinessStreetAddress:           o.str("business_street_address"),
		BusinessStreetAddress2:          o.str("business_street_address2"),
		BusinessCity:                    o.str("business_city"),
		BusinessStateProvinceRegion:     o.str("business_state_province_region"),
		BusinessPostalCode:              o.str("business_postal_code"),
		BusinessCountry:                 o.str("business_country"),
		BusinessContactFirstName:        o.str("business_contact_first_name"),
		BusinessContactLastName:         o.str("business_contact_last_name"),
		BusinessContactEmail:            o.str("business_contact_email"),
		BusinessContactPhone:            o.str("business_contact_phone"),
		BusinessRegistrationNumber:      o.str("business_registration_number"),
		BusinessRegistrationAuthority:   o.str("business_registration_authority"),
		BusinessRegistrationCountry:     o.str("business_registration_country"),
		BusinessType:                    o.str("business_type"),
		BusinessRegistrationPhoneNumber: o.str("business_registration_phone_number"),
		DoingBusinessAs:                 o.str("doing_business_as"),
		NotificationEmail:               o.str("notification_email"),
		UseCaseSummary:                  o.str("use_case_summary"),
		ProductionMessageSample:         o.str("production_message_sample"),
		OptInType:                       o.str("opt_in_type"),
		MessageVolume:                   o.str("message_volume"),
		AdditionalInformation:           o.str("additional_information"),
		OptInConfirmationMessage:        o.str("opt_in_confirmation_message"),
		HelpMessageSample:               o.str("help_message_sample"),
		PrivacyPolicyUrl:                o.str("privacy_policy_url"),
		TermsAndConditionsUrl:           o.str("terms_and_conditions_url"),
		RejectionReason:                 o.str("rejection_reason"),
		EditExpiration:                  o.str("edit_expiration"),
		VettingProvider:                 o.str("vetting_provider"),
		VettingId:                       o.str("vetting_id"),
		VettingIdExpiration:             o.str("vetting_id_expiration"),
		ResourceLinks:                   a2pJSON(o.get("resource_links")),
		Url:                             o.str("url"),
		DateCreated:                     o.str("date_created"),
		DateUpdated:                     o.str("date_updated"),
		UseCaseCategories:               brandList(o, "use_case_categories"),
		OptInImageUrls:                  brandList(o, "opt_in_image_urls"),
		OptInKeywords:                   brandList(o, "opt_in_keywords"),
		RejectionReasons:                brandJSONItems(o.get("rejection_reasons")),
		ErrorCode:                       brandInt(o, "error_code"), EditAllowed: brandBool(o, "edit_allowed"), AgeGatedContent: brandBool(o, "age_gated_content"),
	}
}

// GetTollfreeVerification looks up a toll-free verification by SID.
type GetTollfreeVerification struct{}

// GetTollfreeVerificationArgs are the lookup arguments.
type GetTollfreeVerificationArgs struct {
	Sid string `pulumi:"sid"`
}

func (a *GetTollfreeVerificationArgs) Annotate(an infer.Annotator) {
	an.Describe(&a.Sid, "SID (HH...) of the toll-free verification.")
}

func (f *GetTollfreeVerification) Annotate(a infer.Annotator) {
	a.Describe(&f, "Look up a toll-free verification by SID (GET /v1/Tollfree/Verifications/{Sid} on messaging.twilio.com).")
}

func (f *GetTollfreeVerification) Invoke(ctx context.Context, req infer.FunctionRequest[GetTollfreeVerificationArgs]) (infer.FunctionResponse[TollfreeVerificationInfo], error) {
	const fn = "getTollfreeVerification"
	type resp = infer.FunctionResponse[TollfreeVerificationInfo]
	if err := requireArg(fn, "sid", req.Input.Sid); err != nil {
		return resp{}, err
	}
	c, err := clientFor(ctx)
	if err != nil {
		return resp{}, err
	}
	o, err := c.do(ctx, "GET", messagingProduct, tollfreePath(req.Input.Sid), nil, nil)
	if err != nil {
		return resp{}, lookupFailure(fn, "toll-free verification "+req.Input.Sid, err)
	}
	return resp{Output: tollfreeVerificationInfo(o)}, nil
}

// GetTollfreeVerifications lists toll-free verifications, optionally filtered.
type GetTollfreeVerifications struct{}

// GetTollfreeVerificationsArgs are the optional filters of the list.
type GetTollfreeVerificationsArgs struct {
	TollfreePhoneNumberSid *string `pulumi:"tollfreePhoneNumberSid,optional"`
	Status                 *string `pulumi:"status,optional"`
	ExternalReferenceId    *string `pulumi:"externalReferenceId,optional"`
	TrustProductSid        *string `pulumi:"trustProductSid,optional"`
	IncludeSubAccounts     *bool   `pulumi:"includeSubAccounts,optional"`
}

func (a *GetTollfreeVerificationsArgs) Annotate(an infer.Annotator) {
	an.Describe(&a.TollfreePhoneNumberSid, "Only the verifications of this toll-free number SID (PN...).")
	an.Describe(&a.Status, "Only the verifications in this status: PENDING_REVIEW, IN_REVIEW, TWILIO_APPROVED or TWILIO_REJECTED.")
	an.Describe(&a.ExternalReferenceId, "Only the verifications with this external reference ID.")
	an.Describe(&a.TrustProductSid, "Only the verifications of this Trust Product bundle.")
	an.Describe(&a.IncludeSubAccounts, "Whether to include the verifications of the subaccounts as well.")
}

// GetTollfreeVerificationsResult is the list.
type GetTollfreeVerificationsResult struct {
	Verifications []TollfreeVerificationInfo `pulumi:"verifications"`
}

func (a *GetTollfreeVerificationsResult) Annotate(an infer.Annotator) {
	an.Describe(&a.Verifications, "The verifications, in the order Twilio returns them.")
}

func (f *GetTollfreeVerifications) Annotate(a infer.Annotator) {
	a.Describe(&f, "List toll-free verifications (GET /v1/Tollfree/Verifications on messaging.twilio.com).")
}

func (f *GetTollfreeVerifications) Invoke(ctx context.Context, req infer.FunctionRequest[GetTollfreeVerificationsArgs]) (infer.FunctionResponse[GetTollfreeVerificationsResult], error) {
	const fn = "getTollfreeVerifications"
	none := infer.FunctionResponse[GetTollfreeVerificationsResult]{}
	c, err := clientFor(ctx)
	if err != nil {
		return none, err
	}
	// The API caps PageSize at 50 for this resource (a larger value is a 400, #1979); listPages follows the pages.
	q := url.Values{"PageSize": {"50"}}
	in := req.Input
	setS(q, "TollfreePhoneNumberSid", in.TollfreePhoneNumberSid)
	setS(q, "Status", in.Status)
	setS(q, "ExternalReferenceId", in.ExternalReferenceId)
	setS(q, "TrustProductSid", in.TrustProductSid)
	if in.IncludeSubAccounts != nil {
		q.Set("IncludeSubAccounts", strconv.FormatBool(*in.IncludeSubAccounts))
	}
	out := []TollfreeVerificationInfo{}
	err = listPages(ctx, c, fn, messagingProduct, tollfreePath(""), q, "verifications", func(o obj) bool {
		out = append(out, tollfreeVerificationInfo(o))
		return true
	})
	if err != nil {
		return none, lookupFailure(fn, "toll-free verifications", err)
	}
	return infer.FunctionResponse[GetTollfreeVerificationsResult]{Output: GetTollfreeVerificationsResult{Verifications: out}}, nil
}
