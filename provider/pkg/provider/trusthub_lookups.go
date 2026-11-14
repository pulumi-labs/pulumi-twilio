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

// Lookups of TrustHub end users, supporting documents, their types and the policies
// (trusthub.twilio.com v1).

// thGet reads one object of a collection for a lookup.
func thGet(ctx context.Context, fn, what, coll, sid string) (obj, error) {
	if err := requireArg(fn, "sid", sid); err != nil {
		return nil, err
	}
	c, err := clientFor(ctx)
	if err != nil {
		return nil, err
	}
	o, err := c.do(ctx, "GET", trusthubProduct, thPath(coll, sid), nil, nil)
	if err != nil {
		return nil, lookupFailure(fn, what+" "+sid, err)
	}
	return o, nil
}

// thList reads every page of a collection for a lookup; itemsKey is the array of the page.
func thList[T any](ctx context.Context, fn, what, coll, itemsKey string, conv func(obj) T) ([]T, error) {
	c, err := clientFor(ctx)
	if err != nil {
		return nil, err
	}
	out := []T{}
	// The specification states no maximum PageSize; 50 is the cap of the other Twilio list APIs
	// that state one, and listPages follows the pages either way.
	err = listPages(ctx, c, fn, trusthubProduct, thPath(coll, ""), url.Values{"PageSize": {"50"}}, itemsKey, func(o obj) bool {
		out = append(out, conv(o))
		return true
	})
	if err != nil {
		return nil, lookupFailure(fn, what, err)
	}
	return out, nil
}

// GetTrustHubArgs are the lookup arguments of the lookups by SID.
type GetTrustHubArgs struct {
	Sid string `pulumi:"sid"`
}

func (a *GetTrustHubArgs) Annotate(an infer.Annotator) {
	an.Describe(&a.Sid, "The SID of the object: IT... for an end user, RD... for a supporting document, or the SID of the type or policy.")
}

// ------------------------------------------------------------ end users

// TrustHubEndUserInfo is an end user as the lookups return it.
type TrustHubEndUserInfo struct {
	Sid          string `pulumi:"sid"`
	AccountSid   string `pulumi:"accountSid"`
	FriendlyName string `pulumi:"friendlyName"`
	Type         string `pulumi:"type"`
	Attributes   string `pulumi:"attributes"`
	DateCreated  string `pulumi:"dateCreated"`
	DateUpdated  string `pulumi:"dateUpdated"`
}

func (a *TrustHubEndUserInfo) Annotate(an infer.Annotator) {
	an.Describe(&a.Sid, "The end user SID (IT...).")
	an.Describe(&a.AccountSid, "SID (AC...) of the owning account.")
	an.Describe(&a.FriendlyName, "Human-readable name of the end user.")
	an.Describe(&a.Type, "Machine name of the end user type.")
	an.Describe(&a.Attributes, "The attributes as a JSON object (empty when Twilio reports none).")
	an.Describe(&a.DateCreated, "When the end user was created (ISO 8601).")
	an.Describe(&a.DateUpdated, "When the end user was last updated (ISO 8601).")
}

func thEndUserInfo(o obj) TrustHubEndUserInfo {
	return TrustHubEndUserInfo{Sid: o.str("sid"), AccountSid: o.str("account_sid"), FriendlyName: o.str("friendly_name"), Type: o.str("type"),
		Attributes: a2pJSON(o.get("attributes")), DateCreated: o.str("date_created"), DateUpdated: o.str("date_updated")}
}

// GetTrustHubEndUser looks up an end user by SID.
type GetTrustHubEndUser struct{}

func (f *GetTrustHubEndUser) Annotate(a infer.Annotator) {
	a.Describe(&f, "Look up a TrustHub end user by SID (GET /v1/EndUsers/{Sid} on trusthub.twilio.com).")
}

func (f *GetTrustHubEndUser) Invoke(ctx context.Context, req infer.FunctionRequest[GetTrustHubArgs]) (infer.FunctionResponse[TrustHubEndUserInfo], error) {
	o, err := thGet(ctx, "getTrustHubEndUser", "TrustHub end user", "EndUsers", req.Input.Sid)
	if err != nil {
		return infer.FunctionResponse[TrustHubEndUserInfo]{}, err
	}
	return infer.FunctionResponse[TrustHubEndUserInfo]{Output: thEndUserInfo(o)}, nil
}

// GetTrustHubEndUsersResult is the list of end users.
type GetTrustHubEndUsersResult struct {
	EndUsers []TrustHubEndUserInfo `pulumi:"endUsers"`
}

func (a *GetTrustHubEndUsersResult) Annotate(an infer.Annotator) {
	an.Describe(&a.EndUsers, "The end users, in the order Twilio returns them.")
}

// GetTrustHubEndUsers lists the end users of the account.
type GetTrustHubEndUsers struct{}

func (f *GetTrustHubEndUsers) Annotate(a infer.Annotator) {
	a.Describe(&f, "List the TrustHub end users of the account (GET /v1/EndUsers on trusthub.twilio.com).")
}

func (f *GetTrustHubEndUsers) Invoke(ctx context.Context, _ infer.FunctionRequest[struct{}]) (infer.FunctionResponse[GetTrustHubEndUsersResult], error) {
	l, err := thList(ctx, "getTrustHubEndUsers", "TrustHub end users", "EndUsers", "results", thEndUserInfo)
	return infer.FunctionResponse[GetTrustHubEndUsersResult]{Output: GetTrustHubEndUsersResult{EndUsers: l}}, err
}

// ------------------------------------------------------------ supporting documents

// TrustHubSupportingDocumentInfo is a supporting document as the lookups return it.
type TrustHubSupportingDocumentInfo struct {
	Sid          string `pulumi:"sid"`
	AccountSid   string `pulumi:"accountSid"`
	FriendlyName string `pulumi:"friendlyName"`
	Type         string `pulumi:"type"`
	MimeType     string `pulumi:"mimeType"`
	Status       string `pulumi:"status"`
	Attributes   string `pulumi:"attributes"`
	DateCreated  string `pulumi:"dateCreated"`
	DateUpdated  string `pulumi:"dateUpdated"`
}

func (a *TrustHubSupportingDocumentInfo) Annotate(an infer.Annotator) {
	an.Describe(&a.Sid, "The supporting document SID (RD...).")
	an.Describe(&a.AccountSid, "SID (AC...) of the owning account.")
	an.Describe(&a.FriendlyName, "Human-readable name of the document.")
	an.Describe(&a.Type, "Machine name of the supporting document type.")
	an.Describe(&a.MimeType, "The MIME type of the document (empty when none).")
	an.Describe(&a.Status, "The verification status: DRAFT, PENDING_REVIEW, REJECTED, APPROVED, EXPIRED or PROVISIONALLY_APPROVED.")
	an.Describe(&a.Attributes, "The attributes as a JSON object (empty when Twilio reports none).")
	an.Describe(&a.DateCreated, "When the document was created (ISO 8601).")
	an.Describe(&a.DateUpdated, "When the document was last updated (ISO 8601).")
}

func thDocInfo(o obj) TrustHubSupportingDocumentInfo {
	return TrustHubSupportingDocumentInfo{Sid: o.str("sid"), AccountSid: o.str("account_sid"), FriendlyName: o.str("friendly_name"), Type: o.str("type"),
		MimeType: o.str("mime_type"), Status: o.str("status"), Attributes: a2pJSON(o.get("attributes")), DateCreated: o.str("date_created"), DateUpdated: o.str("date_updated")}
}

// GetTrustHubSupportingDocument looks up a supporting document by SID.
type GetTrustHubSupportingDocument struct{}

func (f *GetTrustHubSupportingDocument) Annotate(a infer.Annotator) {
	a.Describe(&f, "Look up a TrustHub supporting document by SID (GET /v1/SupportingDocuments/{Sid} on trusthub.twilio.com).")
}

func (f *GetTrustHubSupportingDocument) Invoke(ctx context.Context, req infer.FunctionRequest[GetTrustHubArgs]) (infer.FunctionResponse[TrustHubSupportingDocumentInfo], error) {
	o, err := thGet(ctx, "getTrustHubSupportingDocument", "TrustHub supporting document", "SupportingDocuments", req.Input.Sid)
	if err != nil {
		return infer.FunctionResponse[TrustHubSupportingDocumentInfo]{}, err
	}
	return infer.FunctionResponse[TrustHubSupportingDocumentInfo]{Output: thDocInfo(o)}, nil
}

// GetTrustHubSupportingDocumentsResult is the list of supporting documents.
type GetTrustHubSupportingDocumentsResult struct {
	SupportingDocuments []TrustHubSupportingDocumentInfo `pulumi:"supportingDocuments"`
}

func (a *GetTrustHubSupportingDocumentsResult) Annotate(an infer.Annotator) {
	an.Describe(&a.SupportingDocuments, "The supporting documents, in the order Twilio returns them.")
}

// GetTrustHubSupportingDocuments lists the supporting documents of the account.
type GetTrustHubSupportingDocuments struct{}

func (f *GetTrustHubSupportingDocuments) Annotate(a infer.Annotator) {
	a.Describe(&f, "List the TrustHub supporting documents of the account (GET /v1/SupportingDocuments on trusthub.twilio.com).")
}

func (f *GetTrustHubSupportingDocuments) Invoke(ctx context.Context, _ infer.FunctionRequest[struct{}]) (infer.FunctionResponse[GetTrustHubSupportingDocumentsResult], error) {
	l, err := thList(ctx, "getTrustHubSupportingDocuments", "TrustHub supporting documents", "SupportingDocuments", "results", thDocInfo)
	return infer.FunctionResponse[GetTrustHubSupportingDocumentsResult]{Output: GetTrustHubSupportingDocumentsResult{SupportingDocuments: l}}, err
}

// ------------------------------------------------------------ types (end user and supporting document)

// TrustHubTypeInfo is an end user type or a supporting document type: the machine name to use as the type of a
// resource and the fields its attributes take.
type TrustHubTypeInfo struct {
	Sid          string `pulumi:"sid"`
	FriendlyName string `pulumi:"friendlyName"`
	MachineName  string `pulumi:"machineName"`
	Fields       string `pulumi:"fields"`
}

func (a *TrustHubTypeInfo) Annotate(an infer.Annotator) {
	an.Describe(&a.Sid, "The SID of the type.")
	an.Describe(&a.FriendlyName, "Human-readable name of the type.")
	an.Describe(&a.MachineName, "The machine name, which is what the type input of the resource takes.")
	an.Describe(&a.Fields, "The fields the attributes of this type take, as a JSON array (empty when Twilio reports none). Which fields are required changes with the regulations.")
}

func thTypeInfo(o obj) TrustHubTypeInfo {
	return TrustHubTypeInfo{Sid: o.str("sid"), FriendlyName: o.str("friendly_name"), MachineName: o.str("machine_name"), Fields: a2pJSON(o.get("fields"))}
}

// GetTrustHubEndUserType looks up an end user type by SID.
type GetTrustHubEndUserType struct{}

func (f *GetTrustHubEndUserType) Annotate(a infer.Annotator) {
	a.Describe(&f, "Look up a TrustHub end user type (GET /v1/EndUserTypes/{Sid} on trusthub.twilio.com). The sid is the type SID (OY...).")
}

func (f *GetTrustHubEndUserType) Invoke(ctx context.Context, req infer.FunctionRequest[GetTrustHubArgs]) (infer.FunctionResponse[TrustHubTypeInfo], error) {
	o, err := thGet(ctx, "getTrustHubEndUserType", "TrustHub end user type", "EndUserTypes", req.Input.Sid)
	if err != nil {
		return infer.FunctionResponse[TrustHubTypeInfo]{}, err
	}
	return infer.FunctionResponse[TrustHubTypeInfo]{Output: thTypeInfo(o)}, nil
}

// GetTrustHubTypesResult is a list of types.
type GetTrustHubTypesResult struct {
	Types []TrustHubTypeInfo `pulumi:"types"`
}

func (a *GetTrustHubTypesResult) Annotate(an infer.Annotator) {
	an.Describe(&a.Types, "The types, in the order Twilio returns them.")
}

// GetTrustHubEndUserTypes lists the end user types.
type GetTrustHubEndUserTypes struct{}

func (f *GetTrustHubEndUserTypes) Annotate(a infer.Annotator) {
	a.Describe(&f, "List the TrustHub end user types (GET /v1/EndUserTypes on trusthub.twilio.com).")
}

func (f *GetTrustHubEndUserTypes) Invoke(ctx context.Context, _ infer.FunctionRequest[struct{}]) (infer.FunctionResponse[GetTrustHubTypesResult], error) {
	l, err := thList(ctx, "getTrustHubEndUserTypes", "TrustHub end user types", "EndUserTypes", "end_user_types", thTypeInfo)
	return infer.FunctionResponse[GetTrustHubTypesResult]{Output: GetTrustHubTypesResult{Types: l}}, err
}

// GetTrustHubSupportingDocumentType looks up a supporting document type by SID.
type GetTrustHubSupportingDocumentType struct{}

func (f *GetTrustHubSupportingDocumentType) Annotate(a infer.Annotator) {
	a.Describe(&f, "Look up a TrustHub supporting document type (GET /v1/SupportingDocumentTypes/{Sid} on trusthub.twilio.com). The sid is the type SID (OY...).")
}

func (f *GetTrustHubSupportingDocumentType) Invoke(ctx context.Context, req infer.FunctionRequest[GetTrustHubArgs]) (infer.FunctionResponse[TrustHubTypeInfo], error) {
	o, err := thGet(ctx, "getTrustHubSupportingDocumentType", "TrustHub supporting document type", "SupportingDocumentTypes", req.Input.Sid)
	if err != nil {
		return infer.FunctionResponse[TrustHubTypeInfo]{}, err
	}
	return infer.FunctionResponse[TrustHubTypeInfo]{Output: thTypeInfo(o)}, nil
}

// GetTrustHubSupportingDocumentTypes lists the supporting document types.
type GetTrustHubSupportingDocumentTypes struct{}

func (f *GetTrustHubSupportingDocumentTypes) Annotate(a infer.Annotator) {
	a.Describe(&f, "List the TrustHub supporting document types (GET /v1/SupportingDocumentTypes on trusthub.twilio.com).")
}

func (f *GetTrustHubSupportingDocumentTypes) Invoke(ctx context.Context, _ infer.FunctionRequest[struct{}]) (infer.FunctionResponse[GetTrustHubTypesResult], error) {
	l, err := thList(ctx, "getTrustHubSupportingDocumentTypes", "TrustHub supporting document types", "SupportingDocumentTypes", "supporting_document_types", thTypeInfo)
	return infer.FunctionResponse[GetTrustHubTypesResult]{Output: GetTrustHubTypesResult{Types: l}}, err
}

// ------------------------------------------------------------ policies

// TrustHubPolicyInfo is a regulatory policy: the requirements a bundle must meet.
type TrustHubPolicyInfo struct {
	Sid          string `pulumi:"sid"`
	FriendlyName string `pulumi:"friendlyName"`
	Requirements string `pulumi:"requirements"`
}

func (a *TrustHubPolicyInfo) Annotate(an infer.Annotator) {
	an.Describe(&a.Sid, "The policy SID (RN...).")
	an.Describe(&a.FriendlyName, "Human-readable name of the policy.")
	an.Describe(&a.Requirements, "The requirements of the policy as a JSON document: the end user and supporting document types a bundle needs (empty when Twilio reports none).")
}

func thPolicyInfo(o obj) TrustHubPolicyInfo {
	return TrustHubPolicyInfo{Sid: o.str("sid"), FriendlyName: o.str("friendly_name"), Requirements: a2pJSON(o.get("requirements"))}
}

// GetTrustHubPolicy looks up a policy by SID.
type GetTrustHubPolicy struct{}

func (f *GetTrustHubPolicy) Annotate(a infer.Annotator) {
	a.Describe(&f, "Look up a TrustHub policy by SID (GET /v1/Policies/{Sid} on trusthub.twilio.com).")
}

func (f *GetTrustHubPolicy) Invoke(ctx context.Context, req infer.FunctionRequest[GetTrustHubArgs]) (infer.FunctionResponse[TrustHubPolicyInfo], error) {
	o, err := thGet(ctx, "getTrustHubPolicy", "TrustHub policy", "Policies", req.Input.Sid)
	if err != nil {
		return infer.FunctionResponse[TrustHubPolicyInfo]{}, err
	}
	return infer.FunctionResponse[TrustHubPolicyInfo]{Output: thPolicyInfo(o)}, nil
}

// GetTrustHubPoliciesResult is the list of policies.
type GetTrustHubPoliciesResult struct {
	Policies []TrustHubPolicyInfo `pulumi:"policies"`
}

func (a *GetTrustHubPoliciesResult) Annotate(an infer.Annotator) {
	an.Describe(&a.Policies, "The policies, in the order Twilio returns them.")
}

// GetTrustHubPolicies lists the policies.
type GetTrustHubPolicies struct{}

func (f *GetTrustHubPolicies) Annotate(a infer.Annotator) {
	a.Describe(&f, "List the TrustHub policies (GET /v1/Policies on trusthub.twilio.com).")
}

func (f *GetTrustHubPolicies) Invoke(ctx context.Context, _ infer.FunctionRequest[struct{}]) (infer.FunctionResponse[GetTrustHubPoliciesResult], error) {
	l, err := thList(ctx, "getTrustHubPolicies", "TrustHub policies", "Policies", "results", thPolicyInfo)
	return infer.FunctionResponse[GetTrustHubPoliciesResult]{Output: GetTrustHubPoliciesResult{Policies: l}}, err
}
