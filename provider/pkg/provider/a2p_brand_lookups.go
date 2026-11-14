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

// Lookups of A2P 10DLC brand registrations (messaging.twilio.com v1 /a2p/BrandRegistrations).

// BrandRegistrationInfo is one Brand Registration as Twilio reports it.
type BrandRegistrationInfo struct {
	Sid                      string   `pulumi:"sid"`
	CustomerProfileBundleSid string   `pulumi:"customerProfileBundleSid"`
	A2PProfileBundleSid      string   `pulumi:"a2pProfileBundleSid"`
	BrandType                string   `pulumi:"brandType"`
	Status                   string   `pulumi:"status"`
	IdentityStatus           string   `pulumi:"identityStatus"`
	TcrId                    string   `pulumi:"tcrId"`
	FailureReason            string   `pulumi:"failureReason"`
	Errors                   []string `pulumi:"errors"`
	BrandScore               int      `pulumi:"brandScore"`
	Russell3000              bool     `pulumi:"russell3000"`
	GovernmentEntity         bool     `pulumi:"governmentEntity"`
	TaxExemptStatus          string   `pulumi:"taxExemptStatus"`
	Mock                     bool     `pulumi:"mock"`
	SkipAutomaticSecVet      bool     `pulumi:"skipAutomaticSecVet"`
	DateCreated              string   `pulumi:"dateCreated"`
	DateUpdated              string   `pulumi:"dateUpdated"`
}

func (a *BrandRegistrationInfo) Annotate(an infer.Annotator) {
	an.Describe(&a.Sid, "The Brand Registration SID (BN...).")
	an.Describe(&a.CustomerProfileBundleSid, "SID (BU...) of the Customer Profile bundle the brand was registered with.")
	an.Describe(&a.A2PProfileBundleSid, "SID (BU...) of the A2P Messaging Profile bundle the brand was registered with.")
	an.Describe(&a.BrandType, "The brand type: STANDARD or SOLE_PROPRIETOR (empty when Twilio does not say).")
	an.Describe(&a.Status, "The registration status: PENDING, APPROVED, FAILED, IN_REVIEW, DELETION_PENDING, DELETION_FAILED or SUSPENDED.")
	an.Describe(&a.IdentityStatus, "The identity verification status: SELF_DECLARED, UNVERIFIED, VERIFIED or VETTED_VERIFIED (empty when none).")
	an.Describe(&a.TcrId, "The Campaign Registry (TCR) brand ID (empty until assigned).")
	an.Describe(&a.FailureReason, "Deprecated Twilio field: why the registration failed.")
	an.Describe(&a.Errors, "The errors of the registration process, each as a JSON document.")
	an.Describe(&a.BrandScore, "The secondary vetting score, else the brand score (0 when there is none).")
	an.Describe(&a.Russell3000, "Whether the brand is a publicly traded company in the Russell 3000 index.")
	an.Describe(&a.GovernmentEntity, "Whether the brand was identified as a government entity.")
	an.Describe(&a.TaxExemptStatus, "The nonprofit tax-exempt status of the brand (empty when none).")
	an.Describe(&a.Mock, "Whether the brand is a mock brand.")
	an.Describe(&a.SkipAutomaticSecVet, "Whether automatic secondary vetting was disabled for the brand.")
	an.Describe(&a.DateCreated, "When the Brand Registration was created (ISO 8601).")
	an.Describe(&a.DateUpdated, "When the Brand Registration was last updated (ISO 8601).")
}

func brandInfo(o obj) BrandRegistrationInfo {
	return BrandRegistrationInfo{Sid: o.str("sid"), CustomerProfileBundleSid: o.str("customer_profile_bundle_sid"),
		A2PProfileBundleSid: o.str("a2p_profile_bundle_sid"), BrandType: o.str("brand_type"), Status: o.str("status"),
		IdentityStatus: o.str("identity_status"), TcrId: o.str("tcr_id"), FailureReason: o.str("failure_reason"),
		Errors: brandErrors(o), BrandScore: brandInt(o, "brand_score"), Russell3000: brandBool(o, "russell_3000"),
		GovernmentEntity: brandBool(o, "government_entity"), TaxExemptStatus: o.str("tax_exempt_status"),
		Mock: brandBool(o, "mock"), SkipAutomaticSecVet: brandBool(o, "skip_automatic_sec_vet"),
		DateCreated: o.str("date_created"), DateUpdated: o.str("date_updated")}
}

// GetBrandRegistration looks up a Brand Registration by SID.
type GetBrandRegistration struct{}

// GetBrandRegistrationArgs are the lookup arguments.
type GetBrandRegistrationArgs struct {
	Sid string `pulumi:"sid"`
}

func (a *GetBrandRegistrationArgs) Annotate(an infer.Annotator) {
	an.Describe(&a.Sid, "SID (BN...) of the Brand Registration.")
}

func (f *GetBrandRegistration) Annotate(a infer.Annotator) {
	a.Describe(&f, "Look up an A2P 10DLC Brand Registration by SID (GET /v1/a2p/BrandRegistrations/{Sid} on messaging.twilio.com).")
}

func (f *GetBrandRegistration) Invoke(ctx context.Context, req infer.FunctionRequest[GetBrandRegistrationArgs]) (infer.FunctionResponse[BrandRegistrationInfo], error) {
	const fn = "getBrandRegistration"
	type resp = infer.FunctionResponse[BrandRegistrationInfo]
	if err := requireArg(fn, "sid", req.Input.Sid); err != nil {
		return resp{}, err
	}
	c, err := clientFor(ctx)
	if err != nil {
		return resp{}, err
	}
	o, err := c.do(ctx, "GET", messagingProduct, brandPath(req.Input.Sid), nil, nil)
	if err != nil {
		return resp{}, lookupFailure(fn, "Brand Registration "+req.Input.Sid, err)
	}
	return resp{Output: brandInfo(o)}, nil
}

// GetBrandRegistrations lists the Brand Registrations of the account.
type GetBrandRegistrations struct{}

// GetBrandRegistrationsArgs are the lookup arguments (none).
type GetBrandRegistrationsArgs struct{}

// GetBrandRegistrationsResult is the list.
type GetBrandRegistrationsResult struct {
	BrandRegistrations []BrandRegistrationInfo `pulumi:"brandRegistrations"`
}

func (a *GetBrandRegistrationsResult) Annotate(an infer.Annotator) {
	an.Describe(&a.BrandRegistrations, "The Brand Registrations, in the order Twilio returns them.")
}

func (f *GetBrandRegistrations) Annotate(a infer.Annotator) {
	a.Describe(&f, "List the A2P 10DLC Brand Registrations of the account (GET /v1/a2p/BrandRegistrations on messaging.twilio.com).")
}

func (f *GetBrandRegistrations) Invoke(ctx context.Context, _ infer.FunctionRequest[GetBrandRegistrationsArgs]) (infer.FunctionResponse[GetBrandRegistrationsResult], error) {
	const fn = "getBrandRegistrations"
	none := infer.FunctionResponse[GetBrandRegistrationsResult]{}
	c, err := clientFor(ctx)
	if err != nil {
		return none, err
	}
	out := []BrandRegistrationInfo{}
	err = listPages(ctx, c, fn, messagingProduct, brandPath(""), url.Values{"PageSize": {"1000"}}, "data", func(o obj) bool {
		out = append(out, brandInfo(o))
		return true
	})
	if err != nil {
		return none, lookupFailure(fn, "Brand Registrations", err)
	}
	return infer.FunctionResponse[GetBrandRegistrationsResult]{Output: GetBrandRegistrationsResult{BrandRegistrations: out}}, nil
}
