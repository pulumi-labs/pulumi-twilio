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

	"github.com/pulumi/pulumi-go-provider/infer"
)

// A2P 10DLC brand registration (messaging.twilio.com v1 /a2p/BrandRegistrations). A Brand
// Registration submits a Customer Profile bundle and an A2P Messaging Profile bundle to The Campaign
// Registry. The shapes follow twilio-oai spec/json/twilio_messaging_v1.json (1.0.0).

const messagingProduct = "messaging"

var bundleSidRE = regexp.MustCompile(`^BU[0-9a-fA-F]{32}$`)

func brandPath(sid string) string {
	if sid == "" {
		return "/v1/a2p/BrandRegistrations"
	}
	return "/v1/a2p/BrandRegistrations/" + url.PathEscape(sid)
}

// BrandRegistration registers a brand for A2P 10DLC messaging in the United States.
type BrandRegistration struct{}

// BrandRegistrationArgs are the inputs of a BrandRegistration. Twilio cannot change a Brand
// Registration, so every input replaces it.
type BrandRegistrationArgs struct {
	CustomerProfileBundleSid string  `pulumi:"customerProfileBundleSid"`
	A2PProfileBundleSid      string  `pulumi:"a2pProfileBundleSid"`
	BrandType                *string `pulumi:"brandType,optional"`
	Mock                     *bool   `pulumi:"mock,optional"`
	SkipAutomaticSecVet      *bool   `pulumi:"skipAutomaticSecVet,optional"`
}

func (a *BrandRegistrationArgs) Annotate(an infer.Annotator) {
	an.Describe(&a.CustomerProfileBundleSid, "SID (BU...) of the approved Customer Profile bundle (the business's primary profile). Changing it replaces the Brand Registration.")
	an.Describe(&a.A2PProfileBundleSid, "SID (BU...) of the approved A2P Messaging Profile bundle. Changing it replaces the Brand Registration.")
	an.Describe(&a.BrandType, "The type of brand: STANDARD (the default) or SOLE_PROPRIETOR (low volume, sole proprietor use cases). Changing it replaces the Brand Registration.")
	an.Describe(&a.Mock, "Whether to register a mock brand, which is not sent to The Campaign Registry and costs nothing (Twilio default false). Changing it replaces the Brand Registration.")
	an.Describe(&a.SkipAutomaticSecVet, "Whether to disable the automatic secondary vetting Twilio would otherwise run for the brand (Twilio default false). Changing it replaces the Brand Registration.")
}

// BrandRegistrationState is the persisted state of a BrandRegistration.
type BrandRegistrationState struct {
	BrandRegistrationArgs

	Sid              string   `pulumi:"sid"`
	AccountSid       string   `pulumi:"accountSid"`
	Status           string   `pulumi:"status"`
	IdentityStatus   string   `pulumi:"identityStatus"`
	TcrId            string   `pulumi:"tcrId"`
	FailureReason    string   `pulumi:"failureReason"`
	Errors           []string `pulumi:"errors"`
	BrandScore       int      `pulumi:"brandScore"`
	BrandFeedback    []string `pulumi:"brandFeedback"`
	Russell3000      bool     `pulumi:"russell3000"`
	GovernmentEntity bool     `pulumi:"governmentEntity"`
	TaxExemptStatus  string   `pulumi:"taxExemptStatus"`
	Url              string   `pulumi:"url"`
	DateCreated      string   `pulumi:"dateCreated"`
	DateUpdated      string   `pulumi:"dateUpdated"`
}

func (s *BrandRegistrationState) Annotate(a infer.Annotator) {
	a.Describe(&s.Sid, "The Brand Registration SID (BN...).")
	a.Describe(&s.AccountSid, "The SID of the account that created the Brand Registration.")
	a.Describe(&s.Status, "The registration status: PENDING, APPROVED, FAILED, IN_REVIEW, DELETION_PENDING, DELETION_FAILED or SUSPENDED. It changes as The Campaign Registry reviews the brand, so a refresh can show a new value.")
	a.Describe(&s.IdentityStatus, "The identity verification status: SELF_DECLARED, UNVERIFIED, VERIFIED or VETTED_VERIFIED (empty until Twilio reports it).")
	a.Describe(&s.TcrId, "The Campaign Registry (TCR) brand ID, assigned once the brand is registered (empty until then).")
	a.Describe(&s.FailureReason, "Deprecated Twilio field: why the registration failed, when status is FAILED. See errors.")
	a.Describe(&s.Errors, "The errors of the registration process, each one as a JSON document (empty when there are none).")
	a.Describe(&s.BrandScore, "The secondary vetting score, else the brand score The Campaign Registry returned (0 when there is none yet).")
	a.Describe(&s.BrandFeedback, "Deprecated Twilio field: feedback on how to improve the brand score (TAX_ID, STOCK_SYMBOL, NONPROFIT, GOVERNMENT_ENTITY, OTHERS).")
	a.Describe(&s.Russell3000, "Whether the brand was identified as a publicly traded company in the Russell 3000 index.")
	a.Describe(&s.GovernmentEntity, "Whether the brand was identified as a government entity.")
	a.Describe(&s.TaxExemptStatus, "The nonprofit tax-exempt status of the brand (section 501 of the US tax code), empty when none.")
	a.Describe(&s.Url, "The absolute URL of the Brand Registration.")
	a.Describe(&s.DateCreated, "When the Brand Registration was created (ISO 8601).")
	a.Describe(&s.DateUpdated, "When the Brand Registration was last updated (ISO 8601).")
}

func (r *BrandRegistration) Annotate(a infer.Annotator) {
	a.Describe(&r, "An A2P 10DLC brand registration with The Campaign Registry. The resource ID is the Brand Registration SID (BN...). Twilio cannot change a Brand Registration and has no API to delete one, so every input replaces it and destroying the resource only removes it from Pulumi state: the brand stays in the account (contact Twilio support to remove it). A non-mock registration is billed by Twilio and by The Campaign Registry; use mock: true to try the flow for free.")
}

var _ infer.CustomResource[BrandRegistrationArgs, BrandRegistrationState] = (*BrandRegistration)(nil)
var _ infer.CustomCheck[BrandRegistrationArgs] = (*BrandRegistration)(nil)
var _ infer.CustomDiff[BrandRegistrationArgs, BrandRegistrationState] = (*BrandRegistration)(nil)
var _ infer.CustomRead[BrandRegistrationArgs, BrandRegistrationState] = (*BrandRegistration)(nil)
var _ infer.CustomDelete[BrandRegistrationState] = (*BrandRegistration)(nil)

func (r *BrandRegistration) Check(ctx context.Context, req infer.CheckRequest) (infer.CheckResponse[BrandRegistrationArgs], error) {
	args, failures, err := infer.DefaultCheck[BrandRegistrationArgs](ctx, req.NewInputs)
	if err != nil || len(failures) > 0 {
		return infer.CheckResponse[BrandRegistrationArgs]{Inputs: args, Failures: failures}, err
	}
	c := &checker{unk: unknowns(req.NewInputs)}
	c.sid("customerProfileBundleSid", args.CustomerProfileBundleSid, bundleSidRE, "Customer Profile bundle (BU followed by 32 hex characters)")
	c.sid("a2pProfileBundleSid", args.A2PProfileBundleSid, bundleSidRE, "A2P Messaging Profile bundle (BU followed by 32 hex characters)")
	return infer.CheckResponse[BrandRegistrationArgs]{Inputs: args, Failures: c.failures}, nil
}

func (r *BrandRegistration) Diff(_ context.Context, req infer.DiffRequest[BrandRegistrationArgs, BrandRegistrationState]) (infer.DiffResponse, error) {
	o, n := req.State.BrandRegistrationArgs, req.Inputs
	d := trDiffSet{}
	d.replace("customerProfileBundleSid", o.CustomerProfileBundleSid, n.CustomerProfileBundleSid)
	d.replace("a2pProfileBundleSid", o.A2PProfileBundleSid, n.A2PProfileBundleSid)
	d.replaceStr("brandType", o.BrandType, n.BrandType)
	d.replaceBool("mock", flexDefBool(o.Mock), flexDefBool(n.Mock))
	d.replaceBool("skipAutomaticSecVet", flexDefBool(o.SkipAutomaticSecVet), flexDefBool(n.SkipAutomaticSecVet))
	return d.response(), nil
}

func (a BrandRegistrationArgs) form() url.Values {
	form := url.Values{}
	form.Set("CustomerProfileBundleSid", a.CustomerProfileBundleSid)
	form.Set("A2PProfileBundleSid", a.A2PProfileBundleSid)
	setS(form, "BrandType", a.BrandType)
	setB(form, "Mock", a.Mock)
	setB(form, "SkipAutomaticSecVet", a.SkipAutomaticSecVet)
	return form
}

// brandJSONItems renders a free-form list as JSON documents, one per entry (strings stay as they are).
func brandJSONItems(v any) []string {
	out := []string{}
	items, _ := v.([]any)
	for _, it := range items {
		if s, ok := it.(string); ok {
			out = append(out, s)
			continue
		}
		if b, err := json.Marshal(it); err == nil {
			out = append(out, string(b))
		}
	}
	return out
}

// brandErrors renders the free-form errors list as JSON documents.
func brandErrors(o obj) []string {
	return brandJSONItems(o.get("errors"))
}

func brandInt(o obj, k string) int {
	if v := o.optInt(k); v != nil {
		return *v
	}
	return 0
}

func brandBool(o obj, k string) bool {
	b := o.optBool(k)
	return b != nil && *b
}

func brandList(o obj, k string) []string {
	if l := o.optList(k); l != nil {
		return *l
	}
	return []string{}
}

func (s *BrandRegistrationState) fill(o obj) {
	s.Sid, s.AccountSid = o.str("sid"), o.str("account_sid")
	s.Status, s.IdentityStatus = o.str("status"), o.str("identity_status")
	s.TcrId, s.FailureReason = o.str("tcr_id"), o.str("failure_reason")
	s.Errors, s.BrandScore, s.BrandFeedback = brandErrors(o), brandInt(o, "brand_score"), brandList(o, "brand_feedback")
	s.Russell3000, s.GovernmentEntity = brandBool(o, "russell_3000"), brandBool(o, "government_entity")
	s.TaxExemptStatus, s.Url = o.str("tax_exempt_status"), o.str("url")
	s.DateCreated, s.DateUpdated = o.str("date_created"), o.str("date_updated")
}

func (r *BrandRegistration) Create(ctx context.Context, req infer.CreateRequest[BrandRegistrationArgs]) (infer.CreateResponse[BrandRegistrationState], error) {
	type resp = infer.CreateResponse[BrandRegistrationState]
	a := req.Inputs
	if req.DryRun {
		return resp{Output: BrandRegistrationState{BrandRegistrationArgs: a, Errors: []string{}, BrandFeedback: []string{}}}, nil
	}
	c, err := clientFor(ctx)
	if err != nil {
		return resp{}, err
	}
	o, err := c.do(ctx, "POST", messagingProduct, brandPath(""), nil, a.form())
	if err != nil {
		return resp{}, fmt.Errorf("create Brand Registration: %w", err)
	}
	st := BrandRegistrationState{BrandRegistrationArgs: a}
	st.fill(o)
	return resp{ID: st.Sid, Output: st}, nil
}

func (r *BrandRegistration) Read(ctx context.Context, req infer.ReadRequest[BrandRegistrationArgs, BrandRegistrationState]) (infer.ReadResponse[BrandRegistrationArgs, BrandRegistrationState], error) {
	type resp = infer.ReadResponse[BrandRegistrationArgs, BrandRegistrationState]
	c, err := clientFor(ctx)
	if err != nil {
		return resp{}, err
	}
	o, err := c.do(ctx, "GET", messagingProduct, brandPath(req.ID), nil, nil)
	if err != nil {
		if isNotFound(err) {
			if err := notFoundOnImport("Brand Registration", req.ID, req.Inputs, req.State); err != nil {
				return resp{}, err
			}
			return resp{}, nil
		}
		return resp{}, fmt.Errorf("read Brand Registration %s: %w", req.ID, err)
	}
	in := req.Inputs
	imported := req.State.Sid == ""
	if v := o.str("customer_profile_bundle_sid"); v != "" {
		in.CustomerProfileBundleSid = v
	}
	if v := o.str("a2p_profile_bundle_sid"); v != "" {
		in.A2PProfileBundleSid = v
	}
	in.BrandType = trStr(in.BrandType, o.optStr("brand_type"), "STANDARD", imported)
	in.Mock = trBool(in.Mock, o.optBool("mock"), false, imported)
	in.SkipAutomaticSecVet = trBool(in.SkipAutomaticSecVet, o.optBool("skip_automatic_sec_vet"), false, imported)
	st := BrandRegistrationState{BrandRegistrationArgs: in}
	st.fill(o)
	return resp{ID: req.ID, Inputs: in, State: st}, nil
}

// Delete only forgets the Brand Registration: Twilio has no API to delete one.
func (r *BrandRegistration) Delete(_ context.Context, _ infer.DeleteRequest[BrandRegistrationState]) (infer.DeleteResponse, error) {
	return infer.DeleteResponse{}, nil
}
