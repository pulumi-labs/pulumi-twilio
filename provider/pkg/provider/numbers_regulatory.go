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

	"github.com/pulumi/pulumi-go-provider/infer"
)

// Numbers regulatory compliance end users and supporting documents (numbers.twilio.com
// /v2/RegulatoryCompliance, twilio-oai spec/json/twilio_numbers_v2.json 2.11.0; not checked against a
// live account). They have the same shape as the TrustHub ones and share their helpers (thAPI); the
// bundles they are assigned to are in numbers_regulatory_bundles.go.

// ------------------------------------------------------------ RegulatoryEndUser

// RegulatoryEndUser is an end user (a business, an individual or an authorized representative) of a regulatory bundle.
type RegulatoryEndUser struct{}

// RegulatoryEndUserArgs are the inputs of a RegulatoryEndUser.
type RegulatoryEndUserArgs struct {
	FriendlyName string  `pulumi:"friendlyName"`
	Type         string  `pulumi:"type" provider:"replaceOnChanges"`
	Attributes   *string `pulumi:"attributes,optional"`
}

func (a *RegulatoryEndUserArgs) Annotate(an infer.Annotator) {
	an.Describe(&a.FriendlyName, "Human-readable name of the end user.")
	an.Describe(&a.Type, "Machine name of the end user type, for example customer_profile_business_information or authorized_representative_1 (see getRegulatoryEndUserTypes). Changing it replaces the end user.")
	an.Describe(&a.Attributes, thAttributesDoc)
}

// RegulatoryEndUserState is the persisted state of a RegulatoryEndUser.
type RegulatoryEndUserState struct {
	RegulatoryEndUserArgs

	Sid         string `pulumi:"sid"`
	AccountSid  string `pulumi:"accountSid"`
	DateCreated string `pulumi:"dateCreated"`
	DateUpdated string `pulumi:"dateUpdated"`
}

func (s *RegulatoryEndUserState) Annotate(a infer.Annotator) {
	a.Describe(&s.Sid, "The end user SID (IT...).")
	a.Describe(&s.AccountSid, "SID (AC...) of the account that owns the end user.")
	a.Describe(&s.DateCreated, "When the end user was created (ISO 8601).")
	a.Describe(&s.DateUpdated, "When the end user was last updated (ISO 8601).")
}

func (s *RegulatoryEndUserState) fill(o obj) {
	s.Sid, s.AccountSid, s.DateCreated, s.DateUpdated = o.str("sid"), o.str("account_sid"), o.str("date_created"), o.str("date_updated")
}

func (r *RegulatoryEndUser) Annotate(a infer.Annotator) {
	a.Describe(&r, "A Twilio Numbers regulatory end user (numbers.twilio.com /v2/RegulatoryCompliance/EndUsers): the business, individual or authorized representative information that regulatory bundles are built from. The resource ID is the end user SID (IT...). Creating one does not submit anything for review; assign it to a RegulatoryBundle (RegulatoryBundleItemAssignment) to do that.")
}

var _ infer.CustomResource[RegulatoryEndUserArgs, RegulatoryEndUserState] = (*RegulatoryEndUser)(nil)
var _ infer.CustomCheck[RegulatoryEndUserArgs] = (*RegulatoryEndUser)(nil)
var _ infer.CustomDiff[RegulatoryEndUserArgs, RegulatoryEndUserState] = (*RegulatoryEndUser)(nil)
var _ infer.CustomRead[RegulatoryEndUserArgs, RegulatoryEndUserState] = (*RegulatoryEndUser)(nil)
var _ infer.CustomUpdate[RegulatoryEndUserArgs, RegulatoryEndUserState] = (*RegulatoryEndUser)(nil)
var _ infer.CustomDelete[RegulatoryEndUserState] = (*RegulatoryEndUser)(nil)

func (r *RegulatoryEndUser) Check(ctx context.Context, req infer.CheckRequest) (infer.CheckResponse[RegulatoryEndUserArgs], error) {
	args, failures, err := infer.DefaultCheck[RegulatoryEndUserArgs](ctx, req.NewInputs)
	if err != nil || len(failures) > 0 {
		return infer.CheckResponse[RegulatoryEndUserArgs]{Inputs: args, Failures: failures}, err
	}
	c := &checker{unk: unknowns(req.NewInputs)}
	thCheck(c, args.FriendlyName, args.Type, args.Attributes)
	return infer.CheckResponse[RegulatoryEndUserArgs]{Inputs: args, Failures: c.failures}, nil
}

func (r *RegulatoryEndUser) Diff(_ context.Context, req infer.DiffRequest[RegulatoryEndUserArgs, RegulatoryEndUserState]) (infer.DiffResponse, error) {
	o, n := req.State.RegulatoryEndUserArgs, req.Inputs
	d := trDiffSet{}
	d.replace("type", o.Type, n.Type)
	d.requiredStr("friendlyName", o.FriendlyName, n.FriendlyName)
	thAttributesDiff(d, o.Attributes, n.Attributes)
	return d.response(), nil
}

func (r *RegulatoryEndUser) Create(ctx context.Context, req infer.CreateRequest[RegulatoryEndUserArgs]) (infer.CreateResponse[RegulatoryEndUserState], error) {
	type resp = infer.CreateResponse[RegulatoryEndUserState]
	a := req.Inputs
	st := RegulatoryEndUserState{RegulatoryEndUserArgs: a}
	if req.DryRun {
		return resp{Output: st}, nil
	}
	o, err := thCreate(ctx, numbersRegAPI, "end user", "EndUsers", a.FriendlyName, a.Type, a.Attributes)
	if err != nil {
		return resp{}, err
	}
	st.fill(o)
	return resp{ID: st.Sid, Output: st}, nil
}

func (r *RegulatoryEndUser) Read(ctx context.Context, req infer.ReadRequest[RegulatoryEndUserArgs, RegulatoryEndUserState]) (infer.ReadResponse[RegulatoryEndUserArgs, RegulatoryEndUserState], error) {
	type resp = infer.ReadResponse[RegulatoryEndUserArgs, RegulatoryEndUserState]
	o, found, err := thFetch(ctx, numbersRegAPI, "end user", "EndUsers", req.ID)
	if err != nil {
		return resp{}, err
	}
	if !found {
		if err := notFoundOnImport("Numbers regulatory end user", req.ID, req.Inputs, req.State); err != nil {
			return resp{}, err
		}
		return resp{}, nil
	}
	in := req.Inputs
	imported := in.Type == ""
	in.FriendlyName, in.Type = o.str("friendly_name"), o.str("type")
	in.Attributes = thAttributes(in.Attributes, o, imported)
	st := RegulatoryEndUserState{RegulatoryEndUserArgs: in}
	st.fill(o)
	return resp{ID: req.ID, Inputs: in, State: st}, nil
}

func (r *RegulatoryEndUser) Update(ctx context.Context, req infer.UpdateRequest[RegulatoryEndUserArgs, RegulatoryEndUserState]) (infer.UpdateResponse[RegulatoryEndUserState], error) {
	type resp = infer.UpdateResponse[RegulatoryEndUserState]
	st := req.State
	st.RegulatoryEndUserArgs = req.Inputs
	if req.DryRun {
		return resp{Output: st}, nil
	}
	o, err := thUpdate(ctx, numbersRegAPI, "end user", "EndUsers", req.ID, req.Inputs.FriendlyName, req.Inputs.Attributes)
	if err != nil {
		return resp{}, err
	}
	st.DateUpdated = o.str("date_updated")
	return resp{Output: st}, nil
}

func (r *RegulatoryEndUser) Delete(ctx context.Context, req infer.DeleteRequest[RegulatoryEndUserState]) (infer.DeleteResponse, error) {
	return infer.DeleteResponse{}, thDelete(ctx, numbersRegAPI, "end user", "EndUsers", req.ID)
}

// ------------------------------------------------------------ RegulatorySupportingDocument

// RegulatorySupportingDocument is a supporting document (an address or other evidence) of a regulatory bundle.
type RegulatorySupportingDocument struct{}

// RegulatorySupportingDocumentArgs are the inputs of a RegulatorySupportingDocument.
type RegulatorySupportingDocumentArgs struct {
	FriendlyName string  `pulumi:"friendlyName"`
	Type         string  `pulumi:"type" provider:"replaceOnChanges"`
	Attributes   *string `pulumi:"attributes,optional"`
}

func (a *RegulatorySupportingDocumentArgs) Annotate(an infer.Annotator) {
	an.Describe(&a.FriendlyName, "Human-readable name of the supporting document.")
	an.Describe(&a.Type, "Machine name of the supporting document type, for example customer_profile_address (see getRegulatorySupportingDocumentTypes). Changing it replaces the document.")
	an.Describe(&a.Attributes, thAttributesDoc)
}

// RegulatorySupportingDocumentState is the persisted state of a RegulatorySupportingDocument.
type RegulatorySupportingDocumentState struct {
	RegulatorySupportingDocumentArgs

	Sid           string `pulumi:"sid"`
	AccountSid    string `pulumi:"accountSid"`
	MimeType      string `pulumi:"mimeType"`
	Status        string `pulumi:"status"`
	FailureReason string `pulumi:"failureReason"`
	Errors        string `pulumi:"errors"`
	DateCreated   string `pulumi:"dateCreated"`
	DateUpdated   string `pulumi:"dateUpdated"`
}

func (s *RegulatorySupportingDocumentState) Annotate(a infer.Annotator) {
	a.Describe(&s.Sid, "The supporting document SID (RD...).")
	a.Describe(&s.AccountSid, "SID (AC...) of the account that owns the document.")
	a.Describe(&s.MimeType, "The MIME type of the document, empty when Twilio reports none.")
	a.Describe(&s.Status, "The verification status: draft, pending-review, rejected, approved, expired or provisionally-approved, as Twilio last reported it. It changes when a bundle containing the document is reviewed, not through this resource.")
	a.Describe(&s.FailureReason, "Why Twilio rejected the document, empty when it was not rejected.")
	a.Describe(&s.Errors, "The errors of a rejected document as a JSON document, empty when there are none.")
	a.Describe(&s.DateCreated, "When the document was created (ISO 8601).")
	a.Describe(&s.DateUpdated, "When the document was last updated (ISO 8601).")
}

func (s *RegulatorySupportingDocumentState) fill(o obj) {
	s.Sid, s.AccountSid, s.MimeType, s.Status = o.str("sid"), o.str("account_sid"), o.str("mime_type"), o.str("status")
	s.FailureReason, s.Errors = o.str("failure_reason"), a2pJSON(o.get("errors"))
	s.DateCreated, s.DateUpdated = o.str("date_created"), o.str("date_updated")
}

func (r *RegulatorySupportingDocument) Annotate(a infer.Annotator) {
	a.Describe(&r, "A Twilio Numbers regulatory supporting document (numbers.twilio.com /v2/RegulatoryCompliance/SupportingDocuments), such as a business address, used as evidence in a regulatory bundle. The resource ID is the document SID (RD...). A new document is in the draft status and has no cost. Submitting it for review happens through a RegulatoryBundle that it is assigned to. Twilio may refuse to delete a document that a bundle still uses.")
}

var _ infer.CustomResource[RegulatorySupportingDocumentArgs, RegulatorySupportingDocumentState] = (*RegulatorySupportingDocument)(nil)
var _ infer.CustomCheck[RegulatorySupportingDocumentArgs] = (*RegulatorySupportingDocument)(nil)
var _ infer.CustomDiff[RegulatorySupportingDocumentArgs, RegulatorySupportingDocumentState] = (*RegulatorySupportingDocument)(nil)
var _ infer.CustomRead[RegulatorySupportingDocumentArgs, RegulatorySupportingDocumentState] = (*RegulatorySupportingDocument)(nil)
var _ infer.CustomUpdate[RegulatorySupportingDocumentArgs, RegulatorySupportingDocumentState] = (*RegulatorySupportingDocument)(nil)
var _ infer.CustomDelete[RegulatorySupportingDocumentState] = (*RegulatorySupportingDocument)(nil)

func (r *RegulatorySupportingDocument) Check(ctx context.Context, req infer.CheckRequest) (infer.CheckResponse[RegulatorySupportingDocumentArgs], error) {
	args, failures, err := infer.DefaultCheck[RegulatorySupportingDocumentArgs](ctx, req.NewInputs)
	if err != nil || len(failures) > 0 {
		return infer.CheckResponse[RegulatorySupportingDocumentArgs]{Inputs: args, Failures: failures}, err
	}
	c := &checker{unk: unknowns(req.NewInputs)}
	thCheck(c, args.FriendlyName, args.Type, args.Attributes)
	return infer.CheckResponse[RegulatorySupportingDocumentArgs]{Inputs: args, Failures: c.failures}, nil
}

func (r *RegulatorySupportingDocument) Diff(_ context.Context, req infer.DiffRequest[RegulatorySupportingDocumentArgs, RegulatorySupportingDocumentState]) (infer.DiffResponse, error) {
	o, n := req.State.RegulatorySupportingDocumentArgs, req.Inputs
	d := trDiffSet{}
	d.replace("type", o.Type, n.Type)
	d.requiredStr("friendlyName", o.FriendlyName, n.FriendlyName)
	thAttributesDiff(d, o.Attributes, n.Attributes)
	return d.response(), nil
}

func (r *RegulatorySupportingDocument) Create(ctx context.Context, req infer.CreateRequest[RegulatorySupportingDocumentArgs]) (infer.CreateResponse[RegulatorySupportingDocumentState], error) {
	type resp = infer.CreateResponse[RegulatorySupportingDocumentState]
	a := req.Inputs
	st := RegulatorySupportingDocumentState{RegulatorySupportingDocumentArgs: a}
	if req.DryRun {
		return resp{Output: st}, nil
	}
	o, err := thCreate(ctx, numbersRegAPI, "supporting document", "SupportingDocuments", a.FriendlyName, a.Type, a.Attributes)
	if err != nil {
		return resp{}, err
	}
	st.fill(o)
	return resp{ID: st.Sid, Output: st}, nil
}

func (r *RegulatorySupportingDocument) Read(ctx context.Context, req infer.ReadRequest[RegulatorySupportingDocumentArgs, RegulatorySupportingDocumentState]) (infer.ReadResponse[RegulatorySupportingDocumentArgs, RegulatorySupportingDocumentState], error) {
	type resp = infer.ReadResponse[RegulatorySupportingDocumentArgs, RegulatorySupportingDocumentState]
	o, found, err := thFetch(ctx, numbersRegAPI, "supporting document", "SupportingDocuments", req.ID)
	if err != nil {
		return resp{}, err
	}
	if !found {
		if err := notFoundOnImport("Numbers regulatory supporting document", req.ID, req.Inputs, req.State); err != nil {
			return resp{}, err
		}
		return resp{}, nil
	}
	in := req.Inputs
	imported := in.Type == ""
	in.FriendlyName, in.Type = o.str("friendly_name"), o.str("type")
	in.Attributes = thAttributes(in.Attributes, o, imported)
	st := RegulatorySupportingDocumentState{RegulatorySupportingDocumentArgs: in}
	st.fill(o)
	return resp{ID: req.ID, Inputs: in, State: st}, nil
}

func (r *RegulatorySupportingDocument) Update(ctx context.Context, req infer.UpdateRequest[RegulatorySupportingDocumentArgs, RegulatorySupportingDocumentState]) (infer.UpdateResponse[RegulatorySupportingDocumentState], error) {
	type resp = infer.UpdateResponse[RegulatorySupportingDocumentState]
	st := req.State
	st.RegulatorySupportingDocumentArgs = req.Inputs
	if req.DryRun {
		return resp{Output: st}, nil
	}
	o, err := thUpdate(ctx, numbersRegAPI, "supporting document", "SupportingDocuments", req.ID, req.Inputs.FriendlyName, req.Inputs.Attributes)
	if err != nil {
		return resp{}, err
	}
	st.DateUpdated = o.str("date_updated")
	st.Status, st.MimeType = o.str("status"), o.str("mime_type")
	st.FailureReason, st.Errors = o.str("failure_reason"), a2pJSON(o.get("errors"))
	return resp{Output: st}, nil
}

func (r *RegulatorySupportingDocument) Delete(ctx context.Context, req infer.DeleteRequest[RegulatorySupportingDocumentState]) (infer.DeleteResponse, error) {
	return infer.DeleteResponse{}, thDelete(ctx, numbersRegAPI, "supporting document", "SupportingDocuments", req.ID)
}
