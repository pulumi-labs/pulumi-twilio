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

	"github.com/pulumi/pulumi-go-provider/infer"
)

// TrustHub end users and supporting documents (trusthub.twilio.com v1, twilio-oai
// spec/json/twilio_trusthub_v1.json 1.0.0, read 2026-11-14; not checked against a live account).
// Both are plain CRUD objects with an immutable type and updatable friendlyName and attributes.
// They are the building blocks of regulatory bundles (Customer Profiles, Trust Products), which are
// not modelled here: the bundles carry the review workflow and assignments.

const trusthubProduct = "trusthub"

func thPath(coll, sid string) string {
	p := "/v1/" + coll
	if sid != "" {
		p += "/" + url.PathEscape(sid)
	}
	return p
}

// thCheck validates the inputs both resources share.
func thCheck(c *checker, friendlyName, typ string, attrs *string) {
	if !c.unk("friendlyName") && friendlyName == "" {
		c.fail("friendlyName", "friendlyName must not be empty")
	}
	if !c.unk("type") && typ == "" {
		c.fail("type", "type must not be empty")
	}
	if attrs != nil && !c.unk("attributes") {
		var m map[string]any
		if json.Unmarshal([]byte(*attrs), &m) != nil || m == nil {
			c.fail("attributes", "attributes must be a JSON object")
		}
	}
}

func thCreate(ctx context.Context, kind, coll, friendlyName, typ string, attrs *string) (obj, error) {
	c, err := clientFor(ctx)
	if err != nil {
		return nil, err
	}
	form := url.Values{"FriendlyName": {friendlyName}, "Type": {typ}}
	setS(form, "Attributes", attrs)
	o, err := c.do(ctx, "POST", trusthubProduct, thPath(coll, ""), nil, form)
	if err != nil {
		return nil, fmt.Errorf("create TrustHub %s: %w", kind, err)
	}
	if o.str("sid") == "" {
		return nil, fmt.Errorf("create TrustHub %s: Twilio returned no SID", kind)
	}
	return o, nil
}

func thUpdate(ctx context.Context, kind, coll, id, friendlyName string, attrs *string) (obj, error) {
	c, err := clientFor(ctx)
	if err != nil {
		return nil, err
	}
	form := url.Values{"FriendlyName": {friendlyName}}
	setS(form, "Attributes", attrs)
	o, err := c.do(ctx, "POST", trusthubProduct, thPath(coll, id), nil, form)
	if err != nil {
		return nil, fmt.Errorf("update TrustHub %s %s: %w", kind, id, err)
	}
	return o, nil
}

func thDelete(ctx context.Context, kind, coll, id string) error {
	c, err := clientFor(ctx)
	if err != nil {
		return err
	}
	if _, err := c.do(ctx, "DELETE", trusthubProduct, thPath(coll, id), nil, nil); err != nil && !isNotFound(err) {
		return fmt.Errorf("delete TrustHub %s %s: %w", kind, id, err)
	}
	return nil
}

// thFetch reads one object; found is false when it does not exist.
func thFetch(ctx context.Context, kind, coll, id string) (o obj, found bool, err error) {
	c, err := clientFor(ctx)
	if err != nil {
		return nil, false, err
	}
	o, err = c.do(ctx, "GET", trusthubProduct, thPath(coll, id), nil, nil)
	if err != nil {
		if isNotFound(err) {
			return nil, false, nil
		}
		return nil, false, fmt.Errorf("read TrustHub %s %s: %w", kind, id, err)
	}
	return o, true, nil
}

// thAttributes returns the attributes input after a refresh: the program's own formatting when it
// means the same document, Twilio's document otherwise, and nothing on an import of an object
// without attributes (Twilio reports null or {} for it).
func thAttributes(in *string, o obj, imported bool) *string {
	var api *string
	if s := a2pJSON(o.get("attributes")); s != "" {
		api = &s
	}
	out := trJSON(in, api, imported)
	if imported && out != nil && jsonSame(*out, "{}") {
		return nil
	}
	return out
}

// thAttributesDiff compares attributes; an unset stored value means Twilio's default {} as far as a
// program that sets {} is concerned.
func thAttributesDiff(d trDiffSet, o, n *string) {
	d.jsonDoc("attributes", trAttributesOrDefault(o, n), n)
}

const thAttributesDoc = "The attributes of the object as a JSON object. Which keys are valid depends on the type (see getTrustHubEndUserTypes and getTrustHubSupportingDocumentTypes). Formatting and key order do not matter. The attributes often hold personal data: Twilio redacts nothing, and the value is stored in Pulumi state as written, so treat state accordingly. Removing the input leaves the attributes unchanged."

// ------------------------------------------------------------ TrustHubEndUser

// TrustHubEndUser is an end user (a business, an individual or an authorized representative) of a regulatory bundle.
type TrustHubEndUser struct{}

// TrustHubEndUserArgs are the inputs of a TrustHubEndUser.
type TrustHubEndUserArgs struct {
	FriendlyName string  `pulumi:"friendlyName"`
	Type         string  `pulumi:"type" provider:"replaceOnChanges"`
	Attributes   *string `pulumi:"attributes,optional"`
}

func (a *TrustHubEndUserArgs) Annotate(an infer.Annotator) {
	an.Describe(&a.FriendlyName, "Human-readable name of the end user.")
	an.Describe(&a.Type, "Machine name of the end user type, for example customer_profile_business_information or authorized_representative_1 (see getTrustHubEndUserTypes). Changing it replaces the end user.")
	an.Describe(&a.Attributes, thAttributesDoc)
}

// TrustHubEndUserState is the persisted state of a TrustHubEndUser.
type TrustHubEndUserState struct {
	TrustHubEndUserArgs

	Sid         string `pulumi:"sid"`
	AccountSid  string `pulumi:"accountSid"`
	DateCreated string `pulumi:"dateCreated"`
	DateUpdated string `pulumi:"dateUpdated"`
}

func (s *TrustHubEndUserState) Annotate(a infer.Annotator) {
	a.Describe(&s.Sid, "The end user SID (IT...).")
	a.Describe(&s.AccountSid, "SID (AC...) of the account that owns the end user.")
	a.Describe(&s.DateCreated, "When the end user was created (ISO 8601).")
	a.Describe(&s.DateUpdated, "When the end user was last updated (ISO 8601).")
}

func (s *TrustHubEndUserState) fill(o obj) {
	s.Sid, s.AccountSid, s.DateCreated, s.DateUpdated = o.str("sid"), o.str("account_sid"), o.str("date_created"), o.str("date_updated")
}

func (r *TrustHubEndUser) Annotate(a infer.Annotator) {
	a.Describe(&r, "A Twilio TrustHub end user (trusthub.twilio.com /v1/EndUsers): the business, individual or authorized representative information that regulatory bundles are built from. The resource ID is the end user SID (IT...). Creating one does not submit anything for review; assign it to a Customer Profile or Trust Product to do that (not yet modelled).")
}

var _ infer.CustomResource[TrustHubEndUserArgs, TrustHubEndUserState] = (*TrustHubEndUser)(nil)
var _ infer.CustomCheck[TrustHubEndUserArgs] = (*TrustHubEndUser)(nil)
var _ infer.CustomDiff[TrustHubEndUserArgs, TrustHubEndUserState] = (*TrustHubEndUser)(nil)
var _ infer.CustomRead[TrustHubEndUserArgs, TrustHubEndUserState] = (*TrustHubEndUser)(nil)
var _ infer.CustomUpdate[TrustHubEndUserArgs, TrustHubEndUserState] = (*TrustHubEndUser)(nil)
var _ infer.CustomDelete[TrustHubEndUserState] = (*TrustHubEndUser)(nil)

func (r *TrustHubEndUser) Check(ctx context.Context, req infer.CheckRequest) (infer.CheckResponse[TrustHubEndUserArgs], error) {
	args, failures, err := infer.DefaultCheck[TrustHubEndUserArgs](ctx, req.NewInputs)
	if err != nil || len(failures) > 0 {
		return infer.CheckResponse[TrustHubEndUserArgs]{Inputs: args, Failures: failures}, err
	}
	c := &checker{unk: unknowns(req.NewInputs)}
	thCheck(c, args.FriendlyName, args.Type, args.Attributes)
	return infer.CheckResponse[TrustHubEndUserArgs]{Inputs: args, Failures: c.failures}, nil
}

func (r *TrustHubEndUser) Diff(_ context.Context, req infer.DiffRequest[TrustHubEndUserArgs, TrustHubEndUserState]) (infer.DiffResponse, error) {
	o, n := req.State.TrustHubEndUserArgs, req.Inputs
	d := trDiffSet{}
	d.replace("type", o.Type, n.Type)
	d.requiredStr("friendlyName", o.FriendlyName, n.FriendlyName)
	thAttributesDiff(d, o.Attributes, n.Attributes)
	return d.response(), nil
}

func (r *TrustHubEndUser) Create(ctx context.Context, req infer.CreateRequest[TrustHubEndUserArgs]) (infer.CreateResponse[TrustHubEndUserState], error) {
	type resp = infer.CreateResponse[TrustHubEndUserState]
	a := req.Inputs
	st := TrustHubEndUserState{TrustHubEndUserArgs: a}
	if req.DryRun {
		return resp{Output: st}, nil
	}
	o, err := thCreate(ctx, "end user", "EndUsers", a.FriendlyName, a.Type, a.Attributes)
	if err != nil {
		return resp{}, err
	}
	st.fill(o)
	return resp{ID: st.Sid, Output: st}, nil
}

func (r *TrustHubEndUser) Read(ctx context.Context, req infer.ReadRequest[TrustHubEndUserArgs, TrustHubEndUserState]) (infer.ReadResponse[TrustHubEndUserArgs, TrustHubEndUserState], error) {
	type resp = infer.ReadResponse[TrustHubEndUserArgs, TrustHubEndUserState]
	o, found, err := thFetch(ctx, "end user", "EndUsers", req.ID)
	if err != nil {
		return resp{}, err
	}
	if !found {
		if err := notFoundOnImport("TrustHub end user", req.ID, req.Inputs, req.State); err != nil {
			return resp{}, err
		}
		return resp{}, nil
	}
	in := req.Inputs
	imported := in.Type == ""
	in.FriendlyName, in.Type = o.str("friendly_name"), o.str("type")
	in.Attributes = thAttributes(in.Attributes, o, imported)
	st := TrustHubEndUserState{TrustHubEndUserArgs: in}
	st.fill(o)
	return resp{ID: req.ID, Inputs: in, State: st}, nil
}

func (r *TrustHubEndUser) Update(ctx context.Context, req infer.UpdateRequest[TrustHubEndUserArgs, TrustHubEndUserState]) (infer.UpdateResponse[TrustHubEndUserState], error) {
	type resp = infer.UpdateResponse[TrustHubEndUserState]
	st := req.State
	st.TrustHubEndUserArgs = req.Inputs
	if req.DryRun {
		return resp{Output: st}, nil
	}
	o, err := thUpdate(ctx, "end user", "EndUsers", req.ID, req.Inputs.FriendlyName, req.Inputs.Attributes)
	if err != nil {
		return resp{}, err
	}
	st.DateUpdated = o.str("date_updated")
	return resp{Output: st}, nil
}

func (r *TrustHubEndUser) Delete(ctx context.Context, req infer.DeleteRequest[TrustHubEndUserState]) (infer.DeleteResponse, error) {
	return infer.DeleteResponse{}, thDelete(ctx, "end user", "EndUsers", req.ID)
}

// ------------------------------------------------------------ TrustHubSupportingDocument

// TrustHubSupportingDocument is a supporting document (an address or other evidence) of a regulatory bundle.
type TrustHubSupportingDocument struct{}

// TrustHubSupportingDocumentArgs are the inputs of a TrustHubSupportingDocument.
type TrustHubSupportingDocumentArgs struct {
	FriendlyName string  `pulumi:"friendlyName"`
	Type         string  `pulumi:"type" provider:"replaceOnChanges"`
	Attributes   *string `pulumi:"attributes,optional"`
}

func (a *TrustHubSupportingDocumentArgs) Annotate(an infer.Annotator) {
	an.Describe(&a.FriendlyName, "Human-readable name of the supporting document.")
	an.Describe(&a.Type, "Machine name of the supporting document type, for example customer_profile_address (see getTrustHubSupportingDocumentTypes). Changing it replaces the document.")
	an.Describe(&a.Attributes, thAttributesDoc)
}

// TrustHubSupportingDocumentState is the persisted state of a TrustHubSupportingDocument.
type TrustHubSupportingDocumentState struct {
	TrustHubSupportingDocumentArgs

	Sid         string `pulumi:"sid"`
	AccountSid  string `pulumi:"accountSid"`
	MimeType    string `pulumi:"mimeType"`
	Status      string `pulumi:"status"`
	DateCreated string `pulumi:"dateCreated"`
	DateUpdated string `pulumi:"dateUpdated"`
}

func (s *TrustHubSupportingDocumentState) Annotate(a infer.Annotator) {
	a.Describe(&s.Sid, "The supporting document SID (RD...).")
	a.Describe(&s.AccountSid, "SID (AC...) of the account that owns the document.")
	a.Describe(&s.MimeType, "The MIME type of the document, empty when Twilio reports none.")
	a.Describe(&s.Status, "The verification status: DRAFT, PENDING_REVIEW, REJECTED, APPROVED, EXPIRED or PROVISIONALLY_APPROVED, as Twilio last reported it. It changes when a bundle containing the document is reviewed, not through this resource.")
	a.Describe(&s.DateCreated, "When the document was created (ISO 8601).")
	a.Describe(&s.DateUpdated, "When the document was last updated (ISO 8601).")
}

func (s *TrustHubSupportingDocumentState) fill(o obj) {
	s.Sid, s.AccountSid, s.MimeType, s.Status = o.str("sid"), o.str("account_sid"), o.str("mime_type"), o.str("status")
	s.DateCreated, s.DateUpdated = o.str("date_created"), o.str("date_updated")
}

func (r *TrustHubSupportingDocument) Annotate(a infer.Annotator) {
	a.Describe(&r, "A Twilio TrustHub supporting document (trusthub.twilio.com /v1/SupportingDocuments), such as a business address, used as evidence in a regulatory bundle. The resource ID is the document SID (RD...). A new document is in the DRAFT status and has no cost. Submitting it for review happens through a Customer Profile or Trust Product (not yet modelled). Twilio may refuse to delete a document that a bundle still uses.")
}

var _ infer.CustomResource[TrustHubSupportingDocumentArgs, TrustHubSupportingDocumentState] = (*TrustHubSupportingDocument)(nil)
var _ infer.CustomCheck[TrustHubSupportingDocumentArgs] = (*TrustHubSupportingDocument)(nil)
var _ infer.CustomDiff[TrustHubSupportingDocumentArgs, TrustHubSupportingDocumentState] = (*TrustHubSupportingDocument)(nil)
var _ infer.CustomRead[TrustHubSupportingDocumentArgs, TrustHubSupportingDocumentState] = (*TrustHubSupportingDocument)(nil)
var _ infer.CustomUpdate[TrustHubSupportingDocumentArgs, TrustHubSupportingDocumentState] = (*TrustHubSupportingDocument)(nil)
var _ infer.CustomDelete[TrustHubSupportingDocumentState] = (*TrustHubSupportingDocument)(nil)

func (r *TrustHubSupportingDocument) Check(ctx context.Context, req infer.CheckRequest) (infer.CheckResponse[TrustHubSupportingDocumentArgs], error) {
	args, failures, err := infer.DefaultCheck[TrustHubSupportingDocumentArgs](ctx, req.NewInputs)
	if err != nil || len(failures) > 0 {
		return infer.CheckResponse[TrustHubSupportingDocumentArgs]{Inputs: args, Failures: failures}, err
	}
	c := &checker{unk: unknowns(req.NewInputs)}
	thCheck(c, args.FriendlyName, args.Type, args.Attributes)
	return infer.CheckResponse[TrustHubSupportingDocumentArgs]{Inputs: args, Failures: c.failures}, nil
}

func (r *TrustHubSupportingDocument) Diff(_ context.Context, req infer.DiffRequest[TrustHubSupportingDocumentArgs, TrustHubSupportingDocumentState]) (infer.DiffResponse, error) {
	o, n := req.State.TrustHubSupportingDocumentArgs, req.Inputs
	d := trDiffSet{}
	d.replace("type", o.Type, n.Type)
	d.requiredStr("friendlyName", o.FriendlyName, n.FriendlyName)
	thAttributesDiff(d, o.Attributes, n.Attributes)
	return d.response(), nil
}

func (r *TrustHubSupportingDocument) Create(ctx context.Context, req infer.CreateRequest[TrustHubSupportingDocumentArgs]) (infer.CreateResponse[TrustHubSupportingDocumentState], error) {
	type resp = infer.CreateResponse[TrustHubSupportingDocumentState]
	a := req.Inputs
	st := TrustHubSupportingDocumentState{TrustHubSupportingDocumentArgs: a}
	if req.DryRun {
		return resp{Output: st}, nil
	}
	o, err := thCreate(ctx, "supporting document", "SupportingDocuments", a.FriendlyName, a.Type, a.Attributes)
	if err != nil {
		return resp{}, err
	}
	st.fill(o)
	return resp{ID: st.Sid, Output: st}, nil
}

func (r *TrustHubSupportingDocument) Read(ctx context.Context, req infer.ReadRequest[TrustHubSupportingDocumentArgs, TrustHubSupportingDocumentState]) (infer.ReadResponse[TrustHubSupportingDocumentArgs, TrustHubSupportingDocumentState], error) {
	type resp = infer.ReadResponse[TrustHubSupportingDocumentArgs, TrustHubSupportingDocumentState]
	o, found, err := thFetch(ctx, "supporting document", "SupportingDocuments", req.ID)
	if err != nil {
		return resp{}, err
	}
	if !found {
		if err := notFoundOnImport("TrustHub supporting document", req.ID, req.Inputs, req.State); err != nil {
			return resp{}, err
		}
		return resp{}, nil
	}
	in := req.Inputs
	imported := in.Type == ""
	in.FriendlyName, in.Type = o.str("friendly_name"), o.str("type")
	in.Attributes = thAttributes(in.Attributes, o, imported)
	st := TrustHubSupportingDocumentState{TrustHubSupportingDocumentArgs: in}
	st.fill(o)
	return resp{ID: req.ID, Inputs: in, State: st}, nil
}

func (r *TrustHubSupportingDocument) Update(ctx context.Context, req infer.UpdateRequest[TrustHubSupportingDocumentArgs, TrustHubSupportingDocumentState]) (infer.UpdateResponse[TrustHubSupportingDocumentState], error) {
	type resp = infer.UpdateResponse[TrustHubSupportingDocumentState]
	st := req.State
	st.TrustHubSupportingDocumentArgs = req.Inputs
	if req.DryRun {
		return resp{Output: st}, nil
	}
	o, err := thUpdate(ctx, "supporting document", "SupportingDocuments", req.ID, req.Inputs.FriendlyName, req.Inputs.Attributes)
	if err != nil {
		return resp{}, err
	}
	st.DateUpdated = o.str("date_updated")
	st.Status, st.MimeType = o.str("status"), o.str("mime_type")
	return resp{Output: st}, nil
}

func (r *TrustHubSupportingDocument) Delete(ctx context.Context, req infer.DeleteRequest[TrustHubSupportingDocumentState]) (infer.DeleteResponse, error) {
	return infer.DeleteResponse{}, thDelete(ctx, "supporting document", "SupportingDocuments", req.ID)
}
