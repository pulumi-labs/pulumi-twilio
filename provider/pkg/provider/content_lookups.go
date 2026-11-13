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

	"github.com/pulumi/pulumi-go-provider/infer"
)

// Lookups of the Content API. Each reads one object and returns what Twilio reports, null where
// Twilio reports nothing.

// contentLookup reads one object for a lookup function.
func contentLookup(ctx context.Context, fn, what, path string) (obj, error) {
	c, err := clientFor(ctx)
	if err != nil {
		return nil, err
	}
	o, err := c.doJSON(ctx, "GET", contentProduct, path, nil, nil)
	if err != nil {
		return nil, lookupFailure(fn, what, err)
	}
	return o, nil
}

// ------------------------------------------------------------ getContentTemplate

// GetContentTemplate looks up a Content template by SID.
type GetContentTemplate struct{}

// GetContentTemplateArgs are the lookup arguments.
type GetContentTemplateArgs struct {
	Sid string `pulumi:"sid"`
}

func (a *GetContentTemplateArgs) Annotate(an infer.Annotator) {
	an.Describe(&a.Sid, "SID (HX...) of the Content template.")
}

// GetContentTemplateResult is the looked-up template.
type GetContentTemplateResult struct {
	Sid          string            `pulumi:"sid"`
	AccountSid   string            `pulumi:"accountSid"`
	FriendlyName *string           `pulumi:"friendlyName,optional"`
	Language     string            `pulumi:"language"`
	Variables    map[string]string `pulumi:"variables,optional"`
	Types        ContentTypes      `pulumi:"types"`
	DateCreated  string            `pulumi:"dateCreated"`
	DateUpdated  string            `pulumi:"dateUpdated"`
	Url          string            `pulumi:"url"`
}

func (f *GetContentTemplate) Annotate(a infer.Annotator) {
	a.Describe(&f, "Look up a Content template by SID.")
}

func (f *GetContentTemplate) Invoke(ctx context.Context, req infer.FunctionRequest[GetContentTemplateArgs]) (infer.FunctionResponse[GetContentTemplateResult], error) {
	const fn = "getContentTemplate"
	type resp = infer.FunctionResponse[GetContentTemplateResult]
	if err := requireArg(fn, "sid", req.Input.Sid); err != nil {
		return resp{}, err
	}
	o, err := contentLookup(ctx, fn, "Content template "+req.Input.Sid, contentPath(req.Input.Sid))
	if err != nil {
		return resp{}, err
	}
	got, err := decodeContent(o)
	if err != nil {
		return resp{}, err
	}
	return resp{Output: GetContentTemplateResult{Sid: got.Sid, AccountSid: got.AccountSid, FriendlyName: got.FriendlyName,
		Language: got.Language, Variables: got.Variables, Types: got.Types,
		DateCreated: got.DateCreated, DateUpdated: got.DateUpdated, Url: got.Url}}, nil
}

// ------------------------------------------------------------ getContentApprovalStatus

// GetContentApprovalStatus reads the WhatsApp approval status of a Content template.
type GetContentApprovalStatus struct{}

// GetContentApprovalStatusArgs are the lookup arguments.
type GetContentApprovalStatusArgs struct {
	Sid string `pulumi:"sid"`
}

func (a *GetContentApprovalStatusArgs) Annotate(an infer.Annotator) {
	an.Describe(&a.Sid, "SID (HX...) of the Content template.")
}

// GetContentApprovalStatusResult is the approval state of a template.
type GetContentApprovalStatusResult struct {
	Sid        string `pulumi:"sid"`
	AccountSid string `pulumi:"accountSid"`
	// WhatsappStatus, WhatsappRejectionReason and WhatsappCategory are read from the whatsapp
	// object, which the twilio-oai spec leaves untyped; they are null when Twilio omits them.
	WhatsappStatus          *string `pulumi:"whatsappStatus,optional"`
	WhatsappRejectionReason *string `pulumi:"whatsappRejectionReason,optional"`
	WhatsappCategory        *string `pulumi:"whatsappCategory,optional"`
	// WhatsappJson is the whole whatsapp object as JSON, null when the template was never submitted.
	WhatsappJson *string `pulumi:"whatsappJson,optional"`
	Url          string  `pulumi:"url"`
}

func (a *GetContentApprovalStatusResult) Annotate(an infer.Annotator) {
	an.Describe(&a.Sid, "SID of the Content template.")
	an.Describe(&a.AccountSid, "SID of the account that owns the template.")
	an.Describe(&a.WhatsappStatus, "WhatsApp approval status, for example unsubmitted, pending, approved or rejected. Null when Twilio reports none.")
	an.Describe(&a.WhatsappRejectionReason, "Why WhatsApp rejected the template. Null when Twilio reports none.")
	an.Describe(&a.WhatsappCategory, "The template category WhatsApp assigned. Null when Twilio reports none.")
	an.Describe(&a.WhatsappJson, "The whole whatsapp approval object as JSON. Null when the template was never submitted to WhatsApp.")
	an.Describe(&a.Url, "The URL of the approval status resource, relative to https://content.twilio.com.")
}

func (f *GetContentApprovalStatus) Annotate(a infer.Annotator) {
	a.Describe(&f, "Read the WhatsApp approval status of a Content template. Submitting the template for approval is not managed by this provider.")
}

func (f *GetContentApprovalStatus) Invoke(ctx context.Context, req infer.FunctionRequest[GetContentApprovalStatusArgs]) (infer.FunctionResponse[GetContentApprovalStatusResult], error) {
	const fn = "getContentApprovalStatus"
	type resp = infer.FunctionResponse[GetContentApprovalStatusResult]
	if err := requireArg(fn, "sid", req.Input.Sid); err != nil {
		return resp{}, err
	}
	o, err := contentLookup(ctx, fn, "Content template "+req.Input.Sid, contentPath(req.Input.Sid)+"/ApprovalRequests")
	if err != nil {
		return resp{}, err
	}
	out := GetContentApprovalStatusResult{Sid: o.str("sid"), AccountSid: o.str("account_sid"), Url: o.str("url")}
	if wa, ok := o["whatsapp"].(map[string]any); ok && wa != nil {
		w := obj(wa)
		out.WhatsappStatus = nonEmpty(w.optStr("status"))
		out.WhatsappRejectionReason = nonEmpty(w.optStr("rejection_reason"))
		out.WhatsappCategory = nonEmpty(w.optStr("category"))
		if raw, err := json.Marshal(wa); err == nil {
			s := string(raw)
			out.WhatsappJson = &s
		}
	}
	return resp{Output: out}, nil
}
