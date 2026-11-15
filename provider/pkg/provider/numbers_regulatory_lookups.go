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

// Lookups of the Numbers regulatory compliance objects (numbers.twilio.com /v2/RegulatoryCompliance):
// end users, supporting documents, their types, bundles and regulations.

// GetRegulatoryArgs are the lookup arguments of the lookups by SID.
type GetRegulatoryArgs struct {
	Sid string `pulumi:"sid"`
}

func (a *GetRegulatoryArgs) Annotate(an infer.Annotator) {
	an.Describe(&a.Sid, "The SID of the object: IT... for an end user, RD... for a supporting document, BU... for a bundle, RN... for a regulation, or the SID of the type.")
}

// ------------------------------------------------------------ end users

// RegulatoryEndUserInfo is an end user as the lookups return it.
type RegulatoryEndUserInfo struct {
	Sid          string `pulumi:"sid"`
	AccountSid   string `pulumi:"accountSid"`
	FriendlyName string `pulumi:"friendlyName"`
	Type         string `pulumi:"type"`
	Attributes   string `pulumi:"attributes"`
	DateCreated  string `pulumi:"dateCreated"`
	DateUpdated  string `pulumi:"dateUpdated"`
}

func (a *RegulatoryEndUserInfo) Annotate(an infer.Annotator) {
	an.Describe(&a.Sid, "The end user SID (IT...).")
	an.Describe(&a.AccountSid, "SID (AC...) of the owning account.")
	an.Describe(&a.FriendlyName, "Human-readable name of the end user.")
	an.Describe(&a.Type, "Machine name of the end user type.")
	an.Describe(&a.Attributes, "The attributes as a JSON object (empty when Twilio reports none).")
	an.Describe(&a.DateCreated, "When the end user was created (ISO 8601).")
	an.Describe(&a.DateUpdated, "When the end user was last updated (ISO 8601).")
}

func nrEndUserInfo(o obj) RegulatoryEndUserInfo {
	return RegulatoryEndUserInfo{Sid: o.str("sid"), AccountSid: o.str("account_sid"), FriendlyName: o.str("friendly_name"), Type: o.str("type"),
		Attributes: a2pJSON(o.get("attributes")), DateCreated: o.str("date_created"), DateUpdated: o.str("date_updated")}
}

// GetRegulatoryEndUser looks up an end user by SID.
type GetRegulatoryEndUser struct{}

func (f *GetRegulatoryEndUser) Annotate(a infer.Annotator) {
	a.Describe(&f, "Look up a Numbers regulatory end user by SID (GET /v2/RegulatoryCompliance/EndUsers/{Sid} on numbers.twilio.com).")
}

func (f *GetRegulatoryEndUser) Invoke(ctx context.Context, req infer.FunctionRequest[GetRegulatoryArgs]) (infer.FunctionResponse[RegulatoryEndUserInfo], error) {
	o, err := thGet(ctx, numbersRegAPI, "getRegulatoryEndUser", "Numbers regulatory end user", "EndUsers", req.Input.Sid)
	if err != nil {
		return infer.FunctionResponse[RegulatoryEndUserInfo]{}, err
	}
	return infer.FunctionResponse[RegulatoryEndUserInfo]{Output: nrEndUserInfo(o)}, nil
}

// GetRegulatoryEndUsersResult is the list of end users.
type GetRegulatoryEndUsersResult struct {
	EndUsers []RegulatoryEndUserInfo `pulumi:"endUsers"`
}

func (a *GetRegulatoryEndUsersResult) Annotate(an infer.Annotator) {
	an.Describe(&a.EndUsers, "The end users, in the order Twilio returns them.")
}

// GetRegulatoryEndUsers lists the end users of the account.
type GetRegulatoryEndUsers struct{}

func (f *GetRegulatoryEndUsers) Annotate(a infer.Annotator) {
	a.Describe(&f, "List the Numbers regulatory end users of the account (GET /v2/RegulatoryCompliance/EndUsers on numbers.twilio.com).")
}

func (f *GetRegulatoryEndUsers) Invoke(ctx context.Context, _ infer.FunctionRequest[struct{}]) (infer.FunctionResponse[GetRegulatoryEndUsersResult], error) {
	l, err := thList(ctx, numbersRegAPI, "getRegulatoryEndUsers", "Numbers regulatory end users", "EndUsers", "results", nrEndUserInfo)
	return infer.FunctionResponse[GetRegulatoryEndUsersResult]{Output: GetRegulatoryEndUsersResult{EndUsers: l}}, err
}

// ------------------------------------------------------------ supporting documents

// RegulatorySupportingDocumentInfo is a supporting document as the lookups return it.
type RegulatorySupportingDocumentInfo struct {
	Sid           string `pulumi:"sid"`
	AccountSid    string `pulumi:"accountSid"`
	FriendlyName  string `pulumi:"friendlyName"`
	Type          string `pulumi:"type"`
	MimeType      string `pulumi:"mimeType"`
	Status        string `pulumi:"status"`
	FailureReason string `pulumi:"failureReason"`
	Errors        string `pulumi:"errors"`
	Attributes    string `pulumi:"attributes"`
	DateCreated   string `pulumi:"dateCreated"`
	DateUpdated   string `pulumi:"dateUpdated"`
}

func (a *RegulatorySupportingDocumentInfo) Annotate(an infer.Annotator) {
	an.Describe(&a.Sid, "The supporting document SID (RD...).")
	an.Describe(&a.AccountSid, "SID (AC...) of the owning account.")
	an.Describe(&a.FriendlyName, "Human-readable name of the document.")
	an.Describe(&a.Type, "Machine name of the supporting document type.")
	an.Describe(&a.MimeType, "The MIME type of the document (empty when none).")
	an.Describe(&a.Status, "The verification status: draft, pending-review, rejected, approved, expired or provisionally-approved.")
	an.Describe(&a.FailureReason, "Why Twilio rejected the document, empty when it was not rejected.")
	an.Describe(&a.Errors, "The errors of a rejected document as a JSON document (empty when there are none).")
	an.Describe(&a.Attributes, "The attributes as a JSON object (empty when Twilio reports none).")
	an.Describe(&a.DateCreated, "When the document was created (ISO 8601).")
	an.Describe(&a.DateUpdated, "When the document was last updated (ISO 8601).")
}

func nrDocInfo(o obj) RegulatorySupportingDocumentInfo {
	return RegulatorySupportingDocumentInfo{Sid: o.str("sid"), AccountSid: o.str("account_sid"), FriendlyName: o.str("friendly_name"), Type: o.str("type"),
		MimeType: o.str("mime_type"), Status: o.str("status"), FailureReason: o.str("failure_reason"), Errors: a2pJSON(o.get("errors")), Attributes: a2pJSON(o.get("attributes")), DateCreated: o.str("date_created"), DateUpdated: o.str("date_updated")}
}

// GetRegulatorySupportingDocument looks up a supporting document by SID.
type GetRegulatorySupportingDocument struct{}

func (f *GetRegulatorySupportingDocument) Annotate(a infer.Annotator) {
	a.Describe(&f, "Look up a Numbers regulatory supporting document by SID (GET /v2/RegulatoryCompliance/SupportingDocuments/{Sid} on numbers.twilio.com).")
}

func (f *GetRegulatorySupportingDocument) Invoke(ctx context.Context, req infer.FunctionRequest[GetRegulatoryArgs]) (infer.FunctionResponse[RegulatorySupportingDocumentInfo], error) {
	o, err := thGet(ctx, numbersRegAPI, "getRegulatorySupportingDocument", "Numbers regulatory supporting document", "SupportingDocuments", req.Input.Sid)
	if err != nil {
		return infer.FunctionResponse[RegulatorySupportingDocumentInfo]{}, err
	}
	return infer.FunctionResponse[RegulatorySupportingDocumentInfo]{Output: nrDocInfo(o)}, nil
}

// GetRegulatorySupportingDocumentsResult is the list of supporting documents.
type GetRegulatorySupportingDocumentsResult struct {
	SupportingDocuments []RegulatorySupportingDocumentInfo `pulumi:"supportingDocuments"`
}

func (a *GetRegulatorySupportingDocumentsResult) Annotate(an infer.Annotator) {
	an.Describe(&a.SupportingDocuments, "The supporting documents, in the order Twilio returns them.")
}

// GetRegulatorySupportingDocuments lists the supporting documents of the account.
type GetRegulatorySupportingDocuments struct{}

func (f *GetRegulatorySupportingDocuments) Annotate(a infer.Annotator) {
	a.Describe(&f, "List the Numbers regulatory supporting documents of the account (GET /v2/RegulatoryCompliance/SupportingDocuments on numbers.twilio.com).")
}

func (f *GetRegulatorySupportingDocuments) Invoke(ctx context.Context, _ infer.FunctionRequest[struct{}]) (infer.FunctionResponse[GetRegulatorySupportingDocumentsResult], error) {
	l, err := thList(ctx, numbersRegAPI, "getRegulatorySupportingDocuments", "Numbers regulatory supporting documents", "SupportingDocuments", "results", nrDocInfo)
	return infer.FunctionResponse[GetRegulatorySupportingDocumentsResult]{Output: GetRegulatorySupportingDocumentsResult{SupportingDocuments: l}}, err
}

// ------------------------------------------------------------ types (end user and supporting document)

// RegulatoryTypeInfo is an end user type or a supporting document type: the machine name to use as the type of a
// resource and the fields its attributes take.
type RegulatoryTypeInfo struct {
	Sid          string `pulumi:"sid"`
	FriendlyName string `pulumi:"friendlyName"`
	MachineName  string `pulumi:"machineName"`
	Fields       string `pulumi:"fields"`
}

func (a *RegulatoryTypeInfo) Annotate(an infer.Annotator) {
	an.Describe(&a.Sid, "The SID of the type.")
	an.Describe(&a.FriendlyName, "Human-readable name of the type.")
	an.Describe(&a.MachineName, "The machine name, which is what the type input of the resource takes.")
	an.Describe(&a.Fields, "The fields the attributes of this type take, as a JSON array (empty when Twilio reports none). Which fields are required changes with the regulations.")
}

func nrTypeInfo(o obj) RegulatoryTypeInfo {
	return RegulatoryTypeInfo{Sid: o.str("sid"), FriendlyName: o.str("friendly_name"), MachineName: o.str("machine_name"), Fields: a2pJSON(o.get("fields"))}
}

// GetRegulatoryEndUserType looks up an end user type by SID.
type GetRegulatoryEndUserType struct{}

func (f *GetRegulatoryEndUserType) Annotate(a infer.Annotator) {
	a.Describe(&f, "Look up a Numbers regulatory end user type (GET /v2/RegulatoryCompliance/EndUserTypes/{Sid} on numbers.twilio.com). The sid is the type SID (OY...).")
}

func (f *GetRegulatoryEndUserType) Invoke(ctx context.Context, req infer.FunctionRequest[GetRegulatoryArgs]) (infer.FunctionResponse[RegulatoryTypeInfo], error) {
	o, err := thGet(ctx, numbersRegAPI, "getRegulatoryEndUserType", "Numbers regulatory end user type", "EndUserTypes", req.Input.Sid)
	if err != nil {
		return infer.FunctionResponse[RegulatoryTypeInfo]{}, err
	}
	return infer.FunctionResponse[RegulatoryTypeInfo]{Output: nrTypeInfo(o)}, nil
}

// GetRegulatoryTypesResult is a list of types.
type GetRegulatoryTypesResult struct {
	Types []RegulatoryTypeInfo `pulumi:"types"`
}

func (a *GetRegulatoryTypesResult) Annotate(an infer.Annotator) {
	an.Describe(&a.Types, "The types, in the order Twilio returns them.")
}

// GetRegulatoryEndUserTypes lists the end user types.
type GetRegulatoryEndUserTypes struct{}

func (f *GetRegulatoryEndUserTypes) Annotate(a infer.Annotator) {
	a.Describe(&f, "List the Numbers regulatory end user types (GET /v2/RegulatoryCompliance/EndUserTypes on numbers.twilio.com).")
}

func (f *GetRegulatoryEndUserTypes) Invoke(ctx context.Context, _ infer.FunctionRequest[struct{}]) (infer.FunctionResponse[GetRegulatoryTypesResult], error) {
	l, err := thList(ctx, numbersRegAPI, "getRegulatoryEndUserTypes", "Numbers regulatory end user types", "EndUserTypes", "end_user_types", nrTypeInfo)
	return infer.FunctionResponse[GetRegulatoryTypesResult]{Output: GetRegulatoryTypesResult{Types: l}}, err
}

// GetRegulatorySupportingDocumentType looks up a supporting document type by SID.
type GetRegulatorySupportingDocumentType struct{}

func (f *GetRegulatorySupportingDocumentType) Annotate(a infer.Annotator) {
	a.Describe(&f, "Look up a Numbers regulatory supporting document type (GET /v2/RegulatoryCompliance/SupportingDocumentTypes/{Sid} on numbers.twilio.com). The sid is the type SID (OY...).")
}

func (f *GetRegulatorySupportingDocumentType) Invoke(ctx context.Context, req infer.FunctionRequest[GetRegulatoryArgs]) (infer.FunctionResponse[RegulatoryTypeInfo], error) {
	o, err := thGet(ctx, numbersRegAPI, "getRegulatorySupportingDocumentType", "Numbers regulatory supporting document type", "SupportingDocumentTypes", req.Input.Sid)
	if err != nil {
		return infer.FunctionResponse[RegulatoryTypeInfo]{}, err
	}
	return infer.FunctionResponse[RegulatoryTypeInfo]{Output: nrTypeInfo(o)}, nil
}

// GetRegulatorySupportingDocumentTypes lists the supporting document types.
type GetRegulatorySupportingDocumentTypes struct{}

func (f *GetRegulatorySupportingDocumentTypes) Annotate(a infer.Annotator) {
	a.Describe(&f, "List the Numbers regulatory supporting document types (GET /v2/RegulatoryCompliance/SupportingDocumentTypes on numbers.twilio.com).")
}

func (f *GetRegulatorySupportingDocumentTypes) Invoke(ctx context.Context, _ infer.FunctionRequest[struct{}]) (infer.FunctionResponse[GetRegulatoryTypesResult], error) {
	l, err := thList(ctx, numbersRegAPI, "getRegulatorySupportingDocumentTypes", "Numbers regulatory supporting document types", "SupportingDocumentTypes", "supporting_document_types", nrTypeInfo)
	return infer.FunctionResponse[GetRegulatoryTypesResult]{Output: GetRegulatoryTypesResult{Types: l}}, err
}

// ------------------------------------------------------------ bundles

// RegulatoryBundleInfo is a regulatory bundle as the lookups return it.
type RegulatoryBundleInfo struct {
	Sid            string  `pulumi:"sid"`
	AccountSid     string  `pulumi:"accountSid"`
	RegulationSid  string  `pulumi:"regulationSid"`
	FriendlyName   string  `pulumi:"friendlyName"`
	Email          string  `pulumi:"email"`
	Status         string  `pulumi:"status"`
	StatusCallback *string `pulumi:"statusCallback,optional"`
	ValidUntil     string  `pulumi:"validUntil"`
	DateCreated    string  `pulumi:"dateCreated"`
	DateUpdated    string  `pulumi:"dateUpdated"`
}

func (a *RegulatoryBundleInfo) Annotate(an infer.Annotator) {
	an.Describe(&a.Sid, "The bundle SID (BU...).")
	an.Describe(&a.AccountSid, "SID (AC...) of the owning account.")
	an.Describe(&a.RegulationSid, "SID (RN...) of the regulation the bundle is for.")
	an.Describe(&a.FriendlyName, "Name of the bundle.")
	an.Describe(&a.Email, "Email address that receives status updates.")
	an.Describe(&a.Status, "Review status: draft, pending-review, in-review, twilio-rejected, twilio-approved or provisionally-approved.")
	an.Describe(&a.StatusCallback, "URL Twilio calls when the status changes, if any.")
	an.Describe(&a.ValidUntil, "When an approved bundle expires (ISO 8601), empty until then.")
	an.Describe(&a.DateCreated, "When the bundle was created (ISO 8601).")
	an.Describe(&a.DateUpdated, "When the bundle was last updated (ISO 8601).")
}

func nrBundleInfo(o obj) RegulatoryBundleInfo {
	return RegulatoryBundleInfo{Sid: o.str("sid"), AccountSid: o.str("account_sid"), RegulationSid: o.str("regulation_sid"), FriendlyName: o.str("friendly_name"),
		Email: o.str("email"), Status: o.str("status"), StatusCallback: nonEmpty(o.optStr("status_callback")), ValidUntil: o.str("valid_until"),
		DateCreated: o.str("date_created"), DateUpdated: o.str("date_updated")}
}

// GetRegulatoryBundle looks up a regulatory bundle by SID.
type GetRegulatoryBundle struct{}

func (f *GetRegulatoryBundle) Annotate(a infer.Annotator) {
	a.Describe(&f, "Look up a Numbers regulatory bundle by SID (GET /v2/RegulatoryCompliance/Bundles/{Sid} on numbers.twilio.com).")
}

func (f *GetRegulatoryBundle) Invoke(ctx context.Context, req infer.FunctionRequest[GetRegulatoryArgs]) (infer.FunctionResponse[RegulatoryBundleInfo], error) {
	o, err := thGet(ctx, numbersRegAPI, "getRegulatoryBundle", "Numbers regulatory bundle", "Bundles", req.Input.Sid)
	if err != nil {
		return infer.FunctionResponse[RegulatoryBundleInfo]{}, err
	}
	return infer.FunctionResponse[RegulatoryBundleInfo]{Output: nrBundleInfo(o)}, nil
}

// GetRegulatoryBundleItemAssignment looks up one assignment of a regulatory bundle.
type GetRegulatoryBundleItemAssignment struct{}

// GetRegulatoryBundleItemAssignmentArgs are the lookup arguments.
type GetRegulatoryBundleItemAssignmentArgs struct {
	BundleSid string `pulumi:"bundleSid"`
	Sid       string `pulumi:"sid"`
}

func (a *GetRegulatoryBundleItemAssignmentArgs) Annotate(an infer.Annotator) {
	an.Describe(&a.BundleSid, "SID (BU...) of the bundle.")
	an.Describe(&a.Sid, "SID (BV...) of the assignment.")
}

// GetRegulatoryBundleItemAssignmentResult is the looked-up assignment.
type GetRegulatoryBundleItemAssignmentResult struct {
	BundleSid   string `pulumi:"bundleSid"`
	Sid         string `pulumi:"sid"`
	ObjectSid   string `pulumi:"objectSid"`
	AccountSid  string `pulumi:"accountSid"`
	DateCreated string `pulumi:"dateCreated"`
}

func (a *GetRegulatoryBundleItemAssignmentResult) Annotate(an infer.Annotator) {
	an.Describe(&a.BundleSid, "SID of the bundle.")
	an.Describe(&a.Sid, "SID of the assignment.")
	an.Describe(&a.ObjectSid, "SID of the assigned end user or supporting document.")
	an.Describe(&a.AccountSid, "SID of the account that owns the assignment.")
	an.Describe(&a.DateCreated, "When the assignment was created (ISO 8601).")
}

func (f *GetRegulatoryBundleItemAssignment) Annotate(a infer.Annotator) {
	a.Describe(&f, "Look up an end user or supporting document assignment of a Numbers regulatory bundle.")
}

func (f *GetRegulatoryBundleItemAssignment) Invoke(ctx context.Context, req infer.FunctionRequest[GetRegulatoryBundleItemAssignmentArgs]) (infer.FunctionResponse[GetRegulatoryBundleItemAssignmentResult], error) {
	const fn = "getRegulatoryBundleItemAssignment"
	in := req.Input
	if err := requireArgs(fn, [2]string{"bundleSid", in.BundleSid}, [2]string{"sid", in.Sid}); err != nil {
		return infer.FunctionResponse[GetRegulatoryBundleItemAssignmentResult]{}, err
	}
	o, found, err := nrFetchItem(ctx, in.BundleSid, in.Sid)
	if err != nil {
		return infer.FunctionResponse[GetRegulatoryBundleItemAssignmentResult]{}, lookupFailure(fn, "item assignment "+in.Sid, err)
	}
	if !found {
		return infer.FunctionResponse[GetRegulatoryBundleItemAssignmentResult]{}, lookupFailure(fn, "item assignment "+in.Sid, &apiError{Status: 404, Message: "not found"})
	}
	return infer.FunctionResponse[GetRegulatoryBundleItemAssignmentResult]{Output: GetRegulatoryBundleItemAssignmentResult{
		BundleSid: in.BundleSid, Sid: in.Sid, ObjectSid: o.str("object_sid"), AccountSid: o.str("account_sid"), DateCreated: o.str("date_created"),
	}}, nil
}

// ------------------------------------------------------------ regulations

// RegulatoryRegulationInfo is a regulation as the lookups return it.
type RegulatoryRegulationInfo struct {
	Sid          string `pulumi:"sid"`
	FriendlyName string `pulumi:"friendlyName"`
	IsoCountry   string `pulumi:"isoCountry"`
	NumberType   string `pulumi:"numberType"`
	EndUserType  string `pulumi:"endUserType"`
	Requirements string `pulumi:"requirements"`
}

func (a *RegulatoryRegulationInfo) Annotate(an infer.Annotator) {
	an.Describe(&a.Sid, "The regulation SID (RN...), which is what RegulatoryBundle.regulationSid takes.")
	an.Describe(&a.FriendlyName, "Human-readable name of the regulation.")
	an.Describe(&a.IsoCountry, "ISO country code of the phone numbers the regulation covers.")
	an.Describe(&a.NumberType, "Type of the phone numbers the regulation covers.")
	an.Describe(&a.EndUserType, "Who the regulation applies to: individual or business.")
	an.Describe(&a.Requirements, "The end user and supporting document requirements as a JSON document (empty when Twilio reports none).")
}

func nrRegulationInfo(o obj) RegulatoryRegulationInfo {
	return RegulatoryRegulationInfo{Sid: o.str("sid"), FriendlyName: o.str("friendly_name"), IsoCountry: o.str("iso_country"), NumberType: o.str("number_type"),
		EndUserType: o.str("end_user_type"), Requirements: a2pJSON(o.get("requirements"))}
}

// GetRegulatoryRegulation looks up a regulation by SID.
type GetRegulatoryRegulation struct{}

func (f *GetRegulatoryRegulation) Annotate(a infer.Annotator) {
	a.Describe(&f, "Look up a Numbers regulation by SID (GET /v2/RegulatoryCompliance/Regulations/{Sid} on numbers.twilio.com).")
}

func (f *GetRegulatoryRegulation) Invoke(ctx context.Context, req infer.FunctionRequest[GetRegulatoryArgs]) (infer.FunctionResponse[RegulatoryRegulationInfo], error) {
	o, err := thGet(ctx, numbersRegAPI, "getRegulatoryRegulation", "Numbers regulation", "Regulations", req.Input.Sid)
	if err != nil {
		return infer.FunctionResponse[RegulatoryRegulationInfo]{}, err
	}
	return infer.FunctionResponse[RegulatoryRegulationInfo]{Output: nrRegulationInfo(o)}, nil
}

// GetRegulatoryRegulationsArgs filters the regulations.
type GetRegulatoryRegulationsArgs struct {
	IsoCountry  *string `pulumi:"isoCountry,optional"`
	NumberType  *string `pulumi:"numberType,optional"`
	EndUserType *string `pulumi:"endUserType,optional"`
}

func (a *GetRegulatoryRegulationsArgs) Annotate(an infer.Annotator) {
	an.Describe(&a.IsoCountry, "Only the regulations of this ISO country code.")
	an.Describe(&a.NumberType, "Only the regulations of this phone number type.")
	an.Describe(&a.EndUserType, "Only the regulations for this end user type: individual or business.")
}

// GetRegulatoryRegulationsResult is the list of regulations.
type GetRegulatoryRegulationsResult struct {
	Regulations []RegulatoryRegulationInfo `pulumi:"regulations"`
}

func (a *GetRegulatoryRegulationsResult) Annotate(an infer.Annotator) {
	an.Describe(&a.Regulations, "The regulations, in the order Twilio returns them.")
}

// GetRegulatoryRegulations lists the regulations, optionally filtered.
type GetRegulatoryRegulations struct{}

func (f *GetRegulatoryRegulations) Annotate(a infer.Annotator) {
	a.Describe(&f, "List the Numbers regulations (GET /v2/RegulatoryCompliance/Regulations on numbers.twilio.com), optionally for one country, number type or end user type. Use a returned sid as RegulatoryBundle.regulationSid.")
}

func (f *GetRegulatoryRegulations) Invoke(ctx context.Context, req infer.FunctionRequest[GetRegulatoryRegulationsArgs]) (infer.FunctionResponse[GetRegulatoryRegulationsResult], error) {
	const fn = "getRegulatoryRegulations"
	c, err := clientFor(ctx)
	if err != nil {
		return infer.FunctionResponse[GetRegulatoryRegulationsResult]{}, err
	}
	q := url.Values{"PageSize": {"50"}}
	setS(q, "IsoCountry", req.Input.IsoCountry)
	setS(q, "NumberType", req.Input.NumberType)
	setS(q, "EndUserType", req.Input.EndUserType)
	out := []RegulatoryRegulationInfo{}
	err = listPages(ctx, c, fn, numbersProduct, numbersRegAPI.path("Regulations", ""), q, "results", func(o obj) bool {
		out = append(out, nrRegulationInfo(o))
		return true
	})
	if err != nil {
		return infer.FunctionResponse[GetRegulatoryRegulationsResult]{}, lookupFailure(fn, "Numbers regulations", err)
	}
	return infer.FunctionResponse[GetRegulatoryRegulationsResult]{Output: GetRegulatoryRegulationsResult{Regulations: out}}, nil
}
