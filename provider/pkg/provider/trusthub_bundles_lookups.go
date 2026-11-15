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

// ------------------------------------------------------------ CustomerProfile lookups

// GetCustomerProfile looks up a TrustHub customer profile by SID.
type GetCustomerProfile struct{}

// GetCustomerProfileArgs are the lookup arguments.
type GetCustomerProfileArgs struct {
	Sid string `pulumi:"sid"`
}

func (a *GetCustomerProfileArgs) Annotate(an infer.Annotator) {
	an.Describe(&a.Sid, "SID (BU...) of the customer profile.")
}

// GetCustomerProfileResult is the looked-up customer profile.
type GetCustomerProfileResult struct {
	Sid            string  `pulumi:"sid"`
	FriendlyName   string  `pulumi:"friendlyName"`
	Email          string  `pulumi:"email"`
	PolicySid      string  `pulumi:"policySid"`
	Status         string  `pulumi:"status"`
	StatusCallback *string `pulumi:"statusCallback,optional"`
	ValidUntil     string  `pulumi:"validUntil"`
	Errors         string  `pulumi:"errors"`
	DateCreated    string  `pulumi:"dateCreated"`
	DateUpdated    string  `pulumi:"dateUpdated"`
}

func (a *GetCustomerProfileResult) Annotate(an infer.Annotator) {
	an.Describe(&a.Sid, "The customer profile SID.")
	an.Describe(&a.FriendlyName, "Name of the customer profile.")
	an.Describe(&a.Email, "Email address that receives status updates.")
	an.Describe(&a.PolicySid, "SID of the policy the customer profile is evaluated against.")
	an.Describe(&a.Status, "Review status: draft, pending-review, in-review, twilio-rejected or twilio-approved.")
	an.Describe(&a.StatusCallback, "URL Twilio calls when the status changes, if any.")
	an.Describe(&a.ValidUntil, "When an approved customer profile expires (ISO 8601), empty until then.")
	an.Describe(&a.Errors, "The reasons of a rejection as a JSON document, empty when there are none.")
	an.Describe(&a.DateCreated, "When the customer profile was created (ISO 8601).")
	an.Describe(&a.DateUpdated, "When the customer profile was last updated (ISO 8601).")
}

func (f *GetCustomerProfile) Annotate(a infer.Annotator) {
	a.Describe(&f, "Look up a TrustHub customer profile by SID.")
}

func (f *GetCustomerProfile) Invoke(ctx context.Context, req infer.FunctionRequest[GetCustomerProfileArgs]) (infer.FunctionResponse[GetCustomerProfileResult], error) {
	const fn = "getCustomerProfile"
	if err := requireArg(fn, "sid", req.Input.Sid); err != nil {
		return infer.FunctionResponse[GetCustomerProfileResult]{}, err
	}
	o, found, err := thFetch(ctx, trustHubAPI, "customer profile", "CustomerProfiles", req.Input.Sid)
	if err != nil {
		return infer.FunctionResponse[GetCustomerProfileResult]{}, lookupFailure(fn, "customer profile "+req.Input.Sid, err)
	}
	if !found {
		return infer.FunctionResponse[GetCustomerProfileResult]{}, lookupFailure(fn, "customer profile "+req.Input.Sid, &apiError{Status: 404, Message: "not found"})
	}
	return infer.FunctionResponse[GetCustomerProfileResult]{Output: GetCustomerProfileResult{
		Sid: o.str("sid"), FriendlyName: o.str("friendly_name"), Email: o.str("email"), PolicySid: o.str("policy_sid"),
		Status: o.str("status"), StatusCallback: nonEmpty(o.optStr("status_callback")), ValidUntil: o.str("valid_until"),
		Errors: a2pJSON(o.get("errors")), DateCreated: o.str("date_created"), DateUpdated: o.str("date_updated"),
	}}, nil
}

// GetCustomerProfileEntityAssignment looks up one entity assignment of a customer profile.
type GetCustomerProfileEntityAssignment struct{}

// GetCustomerProfileEntityAssignmentArgs are the lookup arguments.
type GetCustomerProfileEntityAssignmentArgs struct {
	CustomerProfileSid string `pulumi:"customerProfileSid"`
	Sid                string `pulumi:"sid"`
}

func (a *GetCustomerProfileEntityAssignmentArgs) Annotate(an infer.Annotator) {
	an.Describe(&a.CustomerProfileSid, "SID (BU...) of the customer profile.")
	an.Describe(&a.Sid, "SID (BV...) of the assignment.")
}

// GetCustomerProfileEntityAssignmentResult is the looked-up assignment.
type GetCustomerProfileEntityAssignmentResult struct {
	CustomerProfileSid string `pulumi:"customerProfileSid"`
	Sid                string `pulumi:"sid"`
	ObjectSid          string `pulumi:"objectSid"`
	AccountSid         string `pulumi:"accountSid"`
	DateCreated        string `pulumi:"dateCreated"`
}

func (a *GetCustomerProfileEntityAssignmentResult) Annotate(an infer.Annotator) {
	an.Describe(&a.CustomerProfileSid, "SID of the customer profile.")
	an.Describe(&a.Sid, "SID of the assignment.")
	an.Describe(&a.ObjectSid, "SID of the assigned end user or supporting document.")
	an.Describe(&a.AccountSid, "SID of the account that owns the assignment.")
	an.Describe(&a.DateCreated, "When the assignment was created (ISO 8601).")
}

func (f *GetCustomerProfileEntityAssignment) Annotate(a infer.Annotator) {
	a.Describe(&f, "Look up an end user or supporting document assignment of a TrustHub customer profile.")
}

func (f *GetCustomerProfileEntityAssignment) Invoke(ctx context.Context, req infer.FunctionRequest[GetCustomerProfileEntityAssignmentArgs]) (infer.FunctionResponse[GetCustomerProfileEntityAssignmentResult], error) {
	const fn = "getCustomerProfileEntityAssignment"
	in := req.Input
	if err := requireArgs(fn, [2]string{"customerProfileSid", in.CustomerProfileSid}, [2]string{"sid", in.Sid}); err != nil {
		return infer.FunctionResponse[GetCustomerProfileEntityAssignmentResult]{}, err
	}
	o, found, err := assignmentFetch(ctx, "CustomerProfiles", in.CustomerProfileSid, in.Sid)
	if err != nil {
		return infer.FunctionResponse[GetCustomerProfileEntityAssignmentResult]{}, lookupFailure(fn, "entity assignment "+in.Sid, err)
	}
	if !found {
		return infer.FunctionResponse[GetCustomerProfileEntityAssignmentResult]{}, lookupFailure(fn, "entity assignment "+in.Sid, &apiError{Status: 404, Message: "not found"})
	}
	return infer.FunctionResponse[GetCustomerProfileEntityAssignmentResult]{Output: GetCustomerProfileEntityAssignmentResult{
		CustomerProfileSid: in.CustomerProfileSid, Sid: in.Sid, ObjectSid: o.str("object_sid"), AccountSid: o.str("account_sid"), DateCreated: o.str("date_created"),
	}}, nil
}

// ------------------------------------------------------------ TrustProduct lookups

// GetTrustProduct looks up a TrustHub trust product by SID.
type GetTrustProduct struct{}

// GetTrustProductArgs are the lookup arguments.
type GetTrustProductArgs struct {
	Sid string `pulumi:"sid"`
}

func (a *GetTrustProductArgs) Annotate(an infer.Annotator) {
	an.Describe(&a.Sid, "SID (BU...) of the trust product.")
}

// GetTrustProductResult is the looked-up trust product.
type GetTrustProductResult struct {
	Sid            string  `pulumi:"sid"`
	FriendlyName   string  `pulumi:"friendlyName"`
	Email          string  `pulumi:"email"`
	PolicySid      string  `pulumi:"policySid"`
	Status         string  `pulumi:"status"`
	StatusCallback *string `pulumi:"statusCallback,optional"`
	ValidUntil     string  `pulumi:"validUntil"`
	Errors         string  `pulumi:"errors"`
	DateCreated    string  `pulumi:"dateCreated"`
	DateUpdated    string  `pulumi:"dateUpdated"`
}

func (a *GetTrustProductResult) Annotate(an infer.Annotator) {
	an.Describe(&a.Sid, "The trust product SID.")
	an.Describe(&a.FriendlyName, "Name of the trust product.")
	an.Describe(&a.Email, "Email address that receives status updates.")
	an.Describe(&a.PolicySid, "SID of the policy the trust product is evaluated against.")
	an.Describe(&a.Status, "Review status: draft, pending-review, in-review, twilio-rejected or twilio-approved.")
	an.Describe(&a.StatusCallback, "URL Twilio calls when the status changes, if any.")
	an.Describe(&a.ValidUntil, "When an approved trust product expires (ISO 8601), empty until then.")
	an.Describe(&a.Errors, "The reasons of a rejection as a JSON document, empty when there are none.")
	an.Describe(&a.DateCreated, "When the trust product was created (ISO 8601).")
	an.Describe(&a.DateUpdated, "When the trust product was last updated (ISO 8601).")
}

func (f *GetTrustProduct) Annotate(a infer.Annotator) {
	a.Describe(&f, "Look up a TrustHub trust product by SID.")
}

func (f *GetTrustProduct) Invoke(ctx context.Context, req infer.FunctionRequest[GetTrustProductArgs]) (infer.FunctionResponse[GetTrustProductResult], error) {
	const fn = "getTrustProduct"
	if err := requireArg(fn, "sid", req.Input.Sid); err != nil {
		return infer.FunctionResponse[GetTrustProductResult]{}, err
	}
	o, found, err := thFetch(ctx, trustHubAPI, "trust product", "TrustProducts", req.Input.Sid)
	if err != nil {
		return infer.FunctionResponse[GetTrustProductResult]{}, lookupFailure(fn, "trust product "+req.Input.Sid, err)
	}
	if !found {
		return infer.FunctionResponse[GetTrustProductResult]{}, lookupFailure(fn, "trust product "+req.Input.Sid, &apiError{Status: 404, Message: "not found"})
	}
	return infer.FunctionResponse[GetTrustProductResult]{Output: GetTrustProductResult{
		Sid: o.str("sid"), FriendlyName: o.str("friendly_name"), Email: o.str("email"), PolicySid: o.str("policy_sid"),
		Status: o.str("status"), StatusCallback: nonEmpty(o.optStr("status_callback")), ValidUntil: o.str("valid_until"),
		Errors: a2pJSON(o.get("errors")), DateCreated: o.str("date_created"), DateUpdated: o.str("date_updated"),
	}}, nil
}

// GetTrustProductEntityAssignment looks up one entity assignment of a trust product.
type GetTrustProductEntityAssignment struct{}

// GetTrustProductEntityAssignmentArgs are the lookup arguments.
type GetTrustProductEntityAssignmentArgs struct {
	TrustProductSid string `pulumi:"trustProductSid"`
	Sid             string `pulumi:"sid"`
}

func (a *GetTrustProductEntityAssignmentArgs) Annotate(an infer.Annotator) {
	an.Describe(&a.TrustProductSid, "SID (BU...) of the trust product.")
	an.Describe(&a.Sid, "SID (BV...) of the assignment.")
}

// GetTrustProductEntityAssignmentResult is the looked-up assignment.
type GetTrustProductEntityAssignmentResult struct {
	TrustProductSid string `pulumi:"trustProductSid"`
	Sid             string `pulumi:"sid"`
	ObjectSid       string `pulumi:"objectSid"`
	AccountSid      string `pulumi:"accountSid"`
	DateCreated     string `pulumi:"dateCreated"`
}

func (a *GetTrustProductEntityAssignmentResult) Annotate(an infer.Annotator) {
	an.Describe(&a.TrustProductSid, "SID of the trust product.")
	an.Describe(&a.Sid, "SID of the assignment.")
	an.Describe(&a.ObjectSid, "SID of the assigned end user or supporting document.")
	an.Describe(&a.AccountSid, "SID of the account that owns the assignment.")
	an.Describe(&a.DateCreated, "When the assignment was created (ISO 8601).")
}

func (f *GetTrustProductEntityAssignment) Annotate(a infer.Annotator) {
	a.Describe(&f, "Look up an end user or supporting document assignment of a TrustHub trust product.")
}

func (f *GetTrustProductEntityAssignment) Invoke(ctx context.Context, req infer.FunctionRequest[GetTrustProductEntityAssignmentArgs]) (infer.FunctionResponse[GetTrustProductEntityAssignmentResult], error) {
	const fn = "getTrustProductEntityAssignment"
	in := req.Input
	if err := requireArgs(fn, [2]string{"trustProductSid", in.TrustProductSid}, [2]string{"sid", in.Sid}); err != nil {
		return infer.FunctionResponse[GetTrustProductEntityAssignmentResult]{}, err
	}
	o, found, err := assignmentFetch(ctx, "TrustProducts", in.TrustProductSid, in.Sid)
	if err != nil {
		return infer.FunctionResponse[GetTrustProductEntityAssignmentResult]{}, lookupFailure(fn, "entity assignment "+in.Sid, err)
	}
	if !found {
		return infer.FunctionResponse[GetTrustProductEntityAssignmentResult]{}, lookupFailure(fn, "entity assignment "+in.Sid, &apiError{Status: 404, Message: "not found"})
	}
	return infer.FunctionResponse[GetTrustProductEntityAssignmentResult]{Output: GetTrustProductEntityAssignmentResult{
		TrustProductSid: in.TrustProductSid, Sid: in.Sid, ObjectSid: o.str("object_sid"), AccountSid: o.str("account_sid"), DateCreated: o.str("date_created"),
	}}, nil
}
