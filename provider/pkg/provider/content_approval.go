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
	"regexp"

	"github.com/pulumi/pulumi-go-provider/infer"
)

// ContentApprovalRequest submits a Content template to WhatsApp for approval.
//
// The shapes follow twilio-oai spec/json/twilio_content_v1.json (version 1.1.0) read 2026-11-14 and
// have not been checked against a live account: POST /v1/Content/{Sid}/ApprovalRequests/whatsapp
// takes {name, category, send_ttl_seconds, bid_amount}, PATCH on the same path takes only
// {bid_amount}, and GET /v1/Content/{Sid}/ApprovalRequests returns the untyped whatsapp object.
//
// The resource ID is the content SID, since a template has one WhatsApp approval request.
// Twilio offers no call to withdraw a submission, so Delete only forgets it.

var contentSidRe = regexp.MustCompile(`^HX[0-9a-fA-F]{32}$`)

func contentApprovalPath(sid string) string {
	return contentPath(sid) + "/ApprovalRequests/whatsapp"
}

// ContentApprovalRequest is the resource.
type ContentApprovalRequest struct{}

// ContentApprovalRequestArgs are the inputs. Everything but bidAmount is fixed once submitted.
type ContentApprovalRequestArgs struct {
	ContentSid     string `pulumi:"contentSid" provider:"replaceOnChanges"`
	Name           string `pulumi:"name" provider:"replaceOnChanges"`
	Category       string `pulumi:"category" provider:"replaceOnChanges"`
	SendTtlSeconds *int   `pulumi:"sendTtlSeconds,optional" provider:"replaceOnChanges"`
	BidAmount      *int   `pulumi:"bidAmount,optional"`
}

func (a *ContentApprovalRequestArgs) Annotate(an infer.Annotator) {
	an.Describe(&a.ContentSid, "SID (HX...) of the Content template to submit. A template has one WhatsApp approval request, so changing this replaces the resource.")
	an.Describe(&a.Name, "Name of the template as WhatsApp knows it: lowercase letters, digits and underscores. Cannot be changed once submitted.")
	an.Describe(&a.Category, "Category requested from WhatsApp: UTILITY, MARKETING or AUTHENTICATION. Cannot be changed once submitted. WhatsApp may assign a different category; see reviewedCategory.")
	an.Describe(&a.SendTtlSeconds, "How long, in seconds, Twilio keeps trying to deliver a message that uses the template. Cannot be changed once submitted.")
	an.Describe(&a.BidAmount, "Maximum price bid, an integer of at least 1, for a MARKETING template under Max Pricing. The only setting that updates in place. Removing it leaves the bid as it is on Twilio.")
}

// ContentApprovalRequestState is the persisted state.
type ContentApprovalRequestState struct {
	ContentApprovalRequestArgs

	Status              string  `pulumi:"status"`
	RejectionReason     *string `pulumi:"rejectionReason,optional"`
	ContentType         *string `pulumi:"contentType,optional"`
	AllowCategoryChange *bool   `pulumi:"allowCategoryChange,optional"`
	ReviewedCategory    *string `pulumi:"reviewedCategory,optional"`
}

func (s *ContentApprovalRequestState) Annotate(a infer.Annotator) {
	a.Describe(&s.Status, "Approval status as Twilio reports it, for example received, pending, approved, rejected or paused. WhatsApp review takes hours to days; the value is refreshed by pulumi refresh and is not waited for.")
	a.Describe(&s.RejectionReason, "Why WhatsApp rejected the template. Null when it did not.")
	a.Describe(&s.ContentType, "The content type of the submitted template, as Twilio reports it.")
	a.Describe(&s.AllowCategoryChange, "Whether WhatsApp may change the category of the template.")
	a.Describe(&s.ReviewedCategory, "The category WhatsApp reports for the template. It can differ from the category requested.")
}

func (r *ContentApprovalRequest) Annotate(a infer.Annotator) {
	a.Describe(&r, "Submits a Content template to WhatsApp for approval. The resource ID is the content SID. Creating it does not wait for the review, which takes hours to days: read status after a refresh. "+
		"Twilio has no call to withdraw a submission, so deleting the resource only removes it from state; delete the ContentTemplate to remove the template. "+
		"Name, category and sendTtlSeconds cannot be changed after submission, and a template WhatsApp rejected cannot be resubmitted in place: create a new ContentTemplate and approval request instead. Only bidAmount updates in place.")
}

var _ infer.CustomResource[ContentApprovalRequestArgs, ContentApprovalRequestState] = (*ContentApprovalRequest)(nil)
var _ infer.CustomCheck[ContentApprovalRequestArgs] = (*ContentApprovalRequest)(nil)
var _ infer.CustomRead[ContentApprovalRequestArgs, ContentApprovalRequestState] = (*ContentApprovalRequest)(nil)
var _ infer.CustomUpdate[ContentApprovalRequestArgs, ContentApprovalRequestState] = (*ContentApprovalRequest)(nil)
var _ infer.CustomDelete[ContentApprovalRequestState] = (*ContentApprovalRequest)(nil)

var contentApprovalCategories = []string{"UTILITY", "MARKETING", "AUTHENTICATION"}

func (r *ContentApprovalRequest) Check(ctx context.Context, req infer.CheckRequest) (infer.CheckResponse[ContentApprovalRequestArgs], error) {
	args, failures, err := infer.DefaultCheck[ContentApprovalRequestArgs](ctx, req.NewInputs)
	if err != nil || len(failures) > 0 {
		return infer.CheckResponse[ContentApprovalRequestArgs]{Inputs: args, Failures: failures}, err
	}
	c := &checker{unk: unknowns(req.NewInputs)}
	c.sid("contentSid", args.ContentSid, contentSidRe, "Content template (HX followed by 32 hex digits)")
	c.required("name", args.Name)
	c.required("category", args.Category)
	if args.Category != "" && !c.unk("category") {
		c.failures = oneOf(c.failures, "category", &args.Category, contentApprovalCategories)
	}
	if args.BidAmount != nil && !c.unk("bidAmount") && *args.BidAmount < 1 {
		c.fail("bidAmount", "bidAmount must be at least 1")
	}
	if args.SendTtlSeconds != nil && !c.unk("sendTtlSeconds") && *args.SendTtlSeconds < 0 {
		c.fail("sendTtlSeconds", "sendTtlSeconds must not be negative")
	}
	return infer.CheckResponse[ContentApprovalRequestArgs]{Inputs: args, Failures: c.failures}, nil
}

// approvalResponse is a decoded approval object.
type approvalResponse struct {
	Name                string
	Category            string
	ContentType         *string
	Status              string
	RejectionReason     *string
	AllowCategoryChange *bool
	SendTtlSeconds      *int
	BidAmount           *int
}

func decodeApproval(o obj) approvalResponse {
	return approvalResponse{Name: o.str("name"), Category: o.str("category"), ContentType: nonEmpty(o.optStr("content_type")),
		Status: o.str("status"), RejectionReason: nonEmpty(o.optStr("rejection_reason")), AllowCategoryChange: o.optBool("allow_category_change"),
		SendTtlSeconds: o.optInt("send_ttl_seconds"), BidAmount: o.optInt("bid_amount")}
}

func approvalStateFrom(in ContentApprovalRequestArgs, a approvalResponse) ContentApprovalRequestState {
	return ContentApprovalRequestState{ContentApprovalRequestArgs: in, Status: a.Status, RejectionReason: a.RejectionReason,
		ContentType: a.ContentType, AllowCategoryChange: a.AllowCategoryChange, ReviewedCategory: nonEmpty(&a.Category)}
}

func (r *ContentApprovalRequest) Create(ctx context.Context, req infer.CreateRequest[ContentApprovalRequestArgs]) (infer.CreateResponse[ContentApprovalRequestState], error) {
	type resp = infer.CreateResponse[ContentApprovalRequestState]
	a := req.Inputs
	if req.DryRun {
		return resp{Output: ContentApprovalRequestState{ContentApprovalRequestArgs: a}}, nil
	}
	c, err := clientFor(ctx)
	if err != nil {
		return resp{}, err
	}
	body := map[string]any{"name": a.Name, "category": a.Category}
	if a.SendTtlSeconds != nil {
		body["send_ttl_seconds"] = *a.SendTtlSeconds
	}
	if a.BidAmount != nil {
		body["bid_amount"] = *a.BidAmount
	}
	o, err := c.doJSON(ctx, "POST", contentProduct, contentApprovalPath(a.ContentSid), nil, body)
	if err != nil {
		return resp{}, fmt.Errorf("submit content template %s for WhatsApp approval: %w", a.ContentSid, err)
	}
	got := decodeApproval(o)
	return resp{ID: a.ContentSid, Output: approvalStateFrom(a, got)}, nil
}

func (r *ContentApprovalRequest) Read(ctx context.Context, req infer.ReadRequest[ContentApprovalRequestArgs, ContentApprovalRequestState]) (infer.ReadResponse[ContentApprovalRequestArgs, ContentApprovalRequestState], error) {
	type resp = infer.ReadResponse[ContentApprovalRequestArgs, ContentApprovalRequestState]
	c, err := clientFor(ctx)
	if err != nil {
		return resp{}, err
	}
	o, err := c.doJSON(ctx, "GET", contentProduct, contentPath(req.ID)+"/ApprovalRequests", nil, nil)
	if err != nil {
		if isNotFound(err) {
			if err := notFoundOnImport("content approval request", req.ID, req.Inputs, req.State); err != nil {
				return resp{}, err
			}
			return resp{}, nil
		}
		return resp{}, fmt.Errorf("read approval request of content template %s: %w", req.ID, err)
	}
	wa, _ := o["whatsapp"].(map[string]any)
	got := decodeApproval(obj(wa))
	if len(wa) == 0 || got.Status == "unsubmitted" || got.Name == "" {
		// Never submitted, or no longer submitted: nothing to manage.
		if err := notFoundOnImport("content approval request", req.ID, req.Inputs, req.State); err != nil {
			return resp{}, err
		}
		return resp{}, nil
	}
	in := req.Inputs
	imported := req.State.Status == "" && req.State.ContentSid == ""
	in.ContentSid = req.ID
	in.Name = got.Name
	// WhatsApp may recategorise a template; the program's category stays, reviewedCategory shows the other.
	if imported || in.Category == "" {
		in.Category = got.Category
	}
	if imported || in.SendTtlSeconds != nil {
		in.SendTtlSeconds = got.SendTtlSeconds
	}
	if imported || in.BidAmount != nil {
		in.BidAmount = got.BidAmount
	}
	return resp{ID: req.ID, Inputs: in, State: approvalStateFrom(in, got)}, nil
}

func (r *ContentApprovalRequest) Update(ctx context.Context, req infer.UpdateRequest[ContentApprovalRequestArgs, ContentApprovalRequestState]) (infer.UpdateResponse[ContentApprovalRequestState], error) {
	type resp = infer.UpdateResponse[ContentApprovalRequestState]
	st := req.State
	st.ContentApprovalRequestArgs = req.Inputs
	if req.DryRun || req.Inputs.BidAmount == nil {
		// Removing bidAmount leaves the bid on Twilio: there is nothing to send.
		return resp{Output: st}, nil
	}
	c, err := clientFor(ctx)
	if err != nil {
		return resp{}, err
	}
	o, err := c.doJSON(ctx, "PATCH", contentProduct, contentApprovalPath(req.ID), nil, map[string]any{"bid_amount": *req.Inputs.BidAmount})
	if err != nil {
		return resp{}, fmt.Errorf("update approval request of content template %s: %w", req.ID, err)
	}
	got := decodeApproval(o)
	if got.Status != "" {
		st.Status = got.Status
	}
	return resp{Output: st}, nil
}

// Delete only forgets the submission: Twilio has no call to withdraw it.
func (r *ContentApprovalRequest) Delete(ctx context.Context, req infer.DeleteRequest[ContentApprovalRequestState]) (infer.DeleteResponse, error) {
	return infer.DeleteResponse{}, nil
}

// WireDependencies: the status fields are what Twilio reports and change with the review.
func (r *ContentApprovalRequest) WireDependencies(f infer.FieldSelector, args *ContentApprovalRequestArgs, state *ContentApprovalRequestState) {
	wireEchoes(f, args, state)
	for _, o := range []any{&state.Status, &state.RejectionReason, &state.ContentType, &state.AllowCategoryChange, &state.ReviewedCategory} {
		f.OutputField(o).DependsOn(f.InputField(args).Computed())
	}
}

// ------------------------------------------------------------ getContentApprovalRequest

// GetContentApprovalRequest reads the WhatsApp approval request of a Content template.
type GetContentApprovalRequest struct{}

// GetContentApprovalRequestArgs are the lookup arguments.
type GetContentApprovalRequestArgs struct {
	ContentSid string `pulumi:"contentSid"`
}

func (a *GetContentApprovalRequestArgs) Annotate(an infer.Annotator) {
	an.Describe(&a.ContentSid, "SID (HX...) of the Content template.")
}

// GetContentApprovalRequestResult is the submitted request.
type GetContentApprovalRequestResult struct {
	ContentSid          string  `pulumi:"contentSid"`
	Name                string  `pulumi:"name"`
	Category            *string `pulumi:"category,optional"`
	Status              string  `pulumi:"status"`
	RejectionReason     *string `pulumi:"rejectionReason,optional"`
	ContentType         *string `pulumi:"contentType,optional"`
	AllowCategoryChange *bool   `pulumi:"allowCategoryChange,optional"`
	SendTtlSeconds      *int    `pulumi:"sendTtlSeconds,optional"`
	BidAmount           *int    `pulumi:"bidAmount,optional"`
}

func (a *GetContentApprovalRequestResult) Annotate(an infer.Annotator) {
	an.Describe(&a.ContentSid, "SID of the Content template.")
	an.Describe(&a.Name, "Name of the template as WhatsApp knows it.")
	an.Describe(&a.Category, "The category WhatsApp reports. Null when Twilio reports none.")
	an.Describe(&a.Status, "Approval status as Twilio reports it.")
	an.Describe(&a.RejectionReason, "Why WhatsApp rejected the template. Null when it did not.")
	an.Describe(&a.ContentType, "The content type of the submitted template. Null when Twilio reports none.")
	an.Describe(&a.AllowCategoryChange, "Whether WhatsApp may change the category. Null when Twilio reports none.")
	an.Describe(&a.SendTtlSeconds, "Delivery time to live in seconds. Null when Twilio reports none.")
	an.Describe(&a.BidAmount, "The maximum price bid. Null when Twilio reports none.")
}

func (f *GetContentApprovalRequest) Annotate(a infer.Annotator) {
	a.Describe(&f, "Read the WhatsApp approval request of a Content template. Fails when the template was never submitted; getContentApprovalStatus reports that case as nulls instead.")
}

func (f *GetContentApprovalRequest) Invoke(ctx context.Context, req infer.FunctionRequest[GetContentApprovalRequestArgs]) (infer.FunctionResponse[GetContentApprovalRequestResult], error) {
	const fn = "getContentApprovalRequest"
	type resp = infer.FunctionResponse[GetContentApprovalRequestResult]
	if err := requireArg(fn, "contentSid", req.Input.ContentSid); err != nil {
		return resp{}, err
	}
	o, err := contentLookup(ctx, fn, "Content template "+req.Input.ContentSid, contentPath(req.Input.ContentSid)+"/ApprovalRequests")
	if err != nil {
		return resp{}, err
	}
	wa, _ := o["whatsapp"].(map[string]any)
	got := decodeApproval(obj(wa))
	if len(wa) == 0 || got.Status == "unsubmitted" || got.Name == "" {
		return resp{}, fmt.Errorf("%s: content template %s has no WhatsApp approval request", fn, req.Input.ContentSid)
	}
	return resp{Output: GetContentApprovalRequestResult{ContentSid: req.Input.ContentSid, Name: got.Name, Category: nonEmpty(&got.Category),
		Status: got.Status, RejectionReason: got.RejectionReason, ContentType: got.ContentType, AllowCategoryChange: got.AllowCategoryChange,
		SendTtlSeconds: got.SendTtlSeconds, BidAmount: got.BidAmount}}, nil
}
