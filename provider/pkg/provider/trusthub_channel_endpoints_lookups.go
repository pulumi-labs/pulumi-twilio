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

// ------------------------------------------------------------ CustomerProfile channel endpoint assignment lookup

// GetCustomerProfileChannelEndpointAssignment looks up one channel endpoint assignment of a customer profile.
type GetCustomerProfileChannelEndpointAssignment struct{}

// GetCustomerProfileChannelEndpointAssignmentArgs are the lookup arguments.
type GetCustomerProfileChannelEndpointAssignmentArgs struct {
	CustomerProfileSid string `pulumi:"customerProfileSid"`
	Sid                string `pulumi:"sid"`
}

func (a *GetCustomerProfileChannelEndpointAssignmentArgs) Annotate(an infer.Annotator) {
	an.Describe(&a.CustomerProfileSid, "SID (BU...) of the customer profile.")
	an.Describe(&a.Sid, "SID (RA...) of the assignment.")
}

// GetCustomerProfileChannelEndpointAssignmentResult is the looked-up assignment.
type GetCustomerProfileChannelEndpointAssignmentResult struct {
	CustomerProfileSid  string `pulumi:"customerProfileSid"`
	Sid                 string `pulumi:"sid"`
	ChannelEndpointType string `pulumi:"channelEndpointType"`
	ChannelEndpointSid  string `pulumi:"channelEndpointSid"`
	AccountSid          string `pulumi:"accountSid"`
	DateCreated         string `pulumi:"dateCreated"`
}

func (a *GetCustomerProfileChannelEndpointAssignmentResult) Annotate(an infer.Annotator) {
	an.Describe(&a.CustomerProfileSid, "SID of the customer profile.")
	an.Describe(&a.Sid, "SID of the assignment.")
	an.Describe(&a.ChannelEndpointType, "Type of the channel endpoint, for example phone-number.")
	an.Describe(&a.ChannelEndpointSid, "SID of the assigned channel endpoint.")
	an.Describe(&a.AccountSid, "SID of the account that owns the assignment.")
	an.Describe(&a.DateCreated, "When the assignment was created (ISO 8601).")
}

func (f *GetCustomerProfileChannelEndpointAssignment) Annotate(a infer.Annotator) {
	a.Describe(&f, "Look up a channel endpoint (phone number) assignment of a TrustHub customer profile. Experimental, unverified against a live account.")
}

func (f *GetCustomerProfileChannelEndpointAssignment) Invoke(ctx context.Context, req infer.FunctionRequest[GetCustomerProfileChannelEndpointAssignmentArgs]) (infer.FunctionResponse[GetCustomerProfileChannelEndpointAssignmentResult], error) {
	const fn = "getCustomerProfileChannelEndpointAssignment"
	in := req.Input
	if err := requireArgs(fn, [2]string{"customerProfileSid", in.CustomerProfileSid}, [2]string{"sid", in.Sid}); err != nil {
		return infer.FunctionResponse[GetCustomerProfileChannelEndpointAssignmentResult]{}, err
	}
	o, found, err := channelAssignmentFetch(ctx, "CustomerProfiles", in.CustomerProfileSid, in.Sid)
	if err != nil {
		return infer.FunctionResponse[GetCustomerProfileChannelEndpointAssignmentResult]{}, lookupFailure(fn, "channel endpoint assignment "+in.Sid, err)
	}
	if !found {
		return infer.FunctionResponse[GetCustomerProfileChannelEndpointAssignmentResult]{}, lookupFailure(fn, "channel endpoint assignment "+in.Sid, &apiError{Status: 404, Message: "not found"})
	}
	return infer.FunctionResponse[GetCustomerProfileChannelEndpointAssignmentResult]{Output: GetCustomerProfileChannelEndpointAssignmentResult{
		CustomerProfileSid: in.CustomerProfileSid, Sid: in.Sid, ChannelEndpointType: o.str("channel_endpoint_type"), ChannelEndpointSid: o.str("channel_endpoint_sid"),
		AccountSid: o.str("account_sid"), DateCreated: o.str("date_created"),
	}}, nil
}

// ------------------------------------------------------------ TrustProduct channel endpoint assignment lookup

// GetTrustProductChannelEndpointAssignment looks up one channel endpoint assignment of a trust product.
type GetTrustProductChannelEndpointAssignment struct{}

// GetTrustProductChannelEndpointAssignmentArgs are the lookup arguments.
type GetTrustProductChannelEndpointAssignmentArgs struct {
	TrustProductSid string `pulumi:"trustProductSid"`
	Sid             string `pulumi:"sid"`
}

func (a *GetTrustProductChannelEndpointAssignmentArgs) Annotate(an infer.Annotator) {
	an.Describe(&a.TrustProductSid, "SID (BU...) of the trust product.")
	an.Describe(&a.Sid, "SID (RA...) of the assignment.")
}

// GetTrustProductChannelEndpointAssignmentResult is the looked-up assignment.
type GetTrustProductChannelEndpointAssignmentResult struct {
	TrustProductSid     string `pulumi:"trustProductSid"`
	Sid                 string `pulumi:"sid"`
	ChannelEndpointType string `pulumi:"channelEndpointType"`
	ChannelEndpointSid  string `pulumi:"channelEndpointSid"`
	AccountSid          string `pulumi:"accountSid"`
	DateCreated         string `pulumi:"dateCreated"`
}

func (a *GetTrustProductChannelEndpointAssignmentResult) Annotate(an infer.Annotator) {
	an.Describe(&a.TrustProductSid, "SID of the trust product.")
	an.Describe(&a.Sid, "SID of the assignment.")
	an.Describe(&a.ChannelEndpointType, "Type of the channel endpoint, for example phone-number.")
	an.Describe(&a.ChannelEndpointSid, "SID of the assigned channel endpoint.")
	an.Describe(&a.AccountSid, "SID of the account that owns the assignment.")
	an.Describe(&a.DateCreated, "When the assignment was created (ISO 8601).")
}

func (f *GetTrustProductChannelEndpointAssignment) Annotate(a infer.Annotator) {
	a.Describe(&f, "Look up a channel endpoint (phone number) assignment of a TrustHub trust product. Experimental, unverified against a live account.")
}

func (f *GetTrustProductChannelEndpointAssignment) Invoke(ctx context.Context, req infer.FunctionRequest[GetTrustProductChannelEndpointAssignmentArgs]) (infer.FunctionResponse[GetTrustProductChannelEndpointAssignmentResult], error) {
	const fn = "getTrustProductChannelEndpointAssignment"
	in := req.Input
	if err := requireArgs(fn, [2]string{"trustProductSid", in.TrustProductSid}, [2]string{"sid", in.Sid}); err != nil {
		return infer.FunctionResponse[GetTrustProductChannelEndpointAssignmentResult]{}, err
	}
	o, found, err := channelAssignmentFetch(ctx, "TrustProducts", in.TrustProductSid, in.Sid)
	if err != nil {
		return infer.FunctionResponse[GetTrustProductChannelEndpointAssignmentResult]{}, lookupFailure(fn, "channel endpoint assignment "+in.Sid, err)
	}
	if !found {
		return infer.FunctionResponse[GetTrustProductChannelEndpointAssignmentResult]{}, lookupFailure(fn, "channel endpoint assignment "+in.Sid, &apiError{Status: 404, Message: "not found"})
	}
	return infer.FunctionResponse[GetTrustProductChannelEndpointAssignmentResult]{Output: GetTrustProductChannelEndpointAssignmentResult{
		TrustProductSid: in.TrustProductSid, Sid: in.Sid, ChannelEndpointType: o.str("channel_endpoint_type"), ChannelEndpointSid: o.str("channel_endpoint_sid"),
		AccountSid: o.str("account_sid"), DateCreated: o.str("date_created"),
	}}, nil
}
