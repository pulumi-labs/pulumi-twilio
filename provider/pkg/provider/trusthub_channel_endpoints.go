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
	"net/url"
	"regexp"
	"strings"

	"github.com/pulumi/pulumi-go-provider/infer"
)

// Channel endpoint assignments attach a phone number (or another channel endpoint) to a TrustHub customer
// profile or trust product (trusthub.twilio.com /v1/{CustomerProfiles,TrustProducts}/{sid}/ChannelEndpointAssignments),
// from twilio-oai spec/json/twilio_trusthub_v1.json (release 2.11.0). Twilio has create, fetch, list and delete, no update,
// so every input replaces. Nothing here has been exercised against a live account yet.

var channelEndpointSidRE = regexp.MustCompile(`^[a-zA-Z]{2}[0-9a-fA-F]{32}$`)

func channelAssignmentPath(coll, bundle, sid string) string {
	p := thPath(coll, bundle) + "/ChannelEndpointAssignments"
	if sid != "" {
		p += "/" + url.PathEscape(sid)
	}
	return p
}

func splitChannelAssignmentID(id string) (bundle, sid string, err error) {
	bundle, sid, ok := strings.Cut(id, "/")
	if !ok || bundle == "" || sid == "" || strings.Contains(sid, "/") {
		return "", "", fmt.Errorf("invalid channel endpoint assignment ID %q: expected <bundleSid>/<assignmentSid>", id)
	}
	return bundle, sid, nil
}

func channelAssignmentCreate(ctx context.Context, coll, bundle, endpointType, endpointSid string) (obj, error) {
	c, err := clientFor(ctx)
	if err != nil {
		return nil, err
	}
	o, err := c.do(ctx, "POST", trusthubProduct, channelAssignmentPath(coll, bundle, ""), nil,
		url.Values{"ChannelEndpointType": {endpointType}, "ChannelEndpointSid": {endpointSid}})
	if err != nil {
		return nil, fmt.Errorf("assign %s %s to TrustHub bundle %s: %w", endpointType, endpointSid, bundle, err)
	}
	if o.str("sid") == "" {
		return nil, fmt.Errorf("assign %s %s to TrustHub bundle %s: Twilio returned no SID", endpointType, endpointSid, bundle)
	}
	return o, nil
}

func channelAssignmentFetch(ctx context.Context, coll, bundle, sid string) (obj, bool, error) {
	c, err := clientFor(ctx)
	if err != nil {
		return nil, false, err
	}
	o, err := c.do(ctx, "GET", trusthubProduct, channelAssignmentPath(coll, bundle, sid), nil, nil)
	if err != nil {
		if isNotFound(err) {
			return nil, false, nil
		}
		return nil, false, fmt.Errorf("read TrustHub channel endpoint assignment %s/%s: %w", bundle, sid, err)
	}
	return o, true, nil
}

func channelAssignmentDelete(ctx context.Context, coll, bundle, sid string) error {
	c, err := clientFor(ctx)
	if err != nil {
		return err
	}
	if _, err := c.do(ctx, "DELETE", trusthubProduct, channelAssignmentPath(coll, bundle, sid), nil, nil); err != nil && !isNotFound(err) {
		return fmt.Errorf("delete TrustHub channel endpoint assignment %s/%s: %w", bundle, sid, err)
	}
	return nil
}

// ------------------------------------------------------------ CustomerProfileChannelEndpointAssignment

// CustomerProfileChannelEndpointAssignment assigns a channel endpoint, such as a phone number, to a customer profile.
type CustomerProfileChannelEndpointAssignment struct{}

// CustomerProfileChannelEndpointAssignmentArgs are the inputs of a CustomerProfileChannelEndpointAssignment.
type CustomerProfileChannelEndpointAssignmentArgs struct {
	CustomerProfileSid  string `pulumi:"customerProfileSid" provider:"replaceOnChanges"`
	ChannelEndpointType string `pulumi:"channelEndpointType" provider:"replaceOnChanges"`
	ChannelEndpointSid  string `pulumi:"channelEndpointSid" provider:"replaceOnChanges"`
}

func (a *CustomerProfileChannelEndpointAssignmentArgs) Annotate(an infer.Annotator) {
	an.Describe(&a.CustomerProfileSid, "SID (BU...) of the customer profile. Changing it replaces the assignment.")
	an.Describe(&a.ChannelEndpointType, "Type of the channel endpoint, for example phone-number. Changing it replaces the assignment.")
	an.Describe(&a.ChannelEndpointSid, "SID of the channel endpoint, for example the PN... SID of an incoming phone number. Changing it replaces the assignment.")
}

// CustomerProfileChannelEndpointAssignmentState is the persisted state of a CustomerProfileChannelEndpointAssignment.
type CustomerProfileChannelEndpointAssignmentState struct {
	CustomerProfileChannelEndpointAssignmentArgs

	Sid         string `pulumi:"sid"`
	AccountSid  string `pulumi:"accountSid"`
	DateCreated string `pulumi:"dateCreated"`
}

func (s *CustomerProfileChannelEndpointAssignmentState) Annotate(a infer.Annotator) {
	a.Describe(&s.Sid, "The assignment SID (RA...). The resource ID is the customer profile SID and this SID joined by a slash.")
	a.Describe(&s.AccountSid, "SID (AC...) of the account that owns the assignment.")
	a.Describe(&s.DateCreated, "When the assignment was created (ISO 8601).")
}

func (r *CustomerProfileChannelEndpointAssignment) Annotate(a infer.Annotator) {
	a.Describe(&r, "Assigns a channel endpoint (a phone number) to a TrustHub customer profile (trusthub.twilio.com /v1/CustomerProfiles/{sid}/ChannelEndpointAssignments). Every input replaces, because Twilio has no update call. "+
		"Twilio only accepts the assignment while the customer profile is a draft or approved as it documents, and may refuse to remove one once the customer profile is in review. Import with the ID <customerProfileSid>/<assignmentSid>. Experimental, not verified against a live account.")
}

var _ infer.CustomResource[CustomerProfileChannelEndpointAssignmentArgs, CustomerProfileChannelEndpointAssignmentState] = (*CustomerProfileChannelEndpointAssignment)(nil)
var _ infer.CustomCheck[CustomerProfileChannelEndpointAssignmentArgs] = (*CustomerProfileChannelEndpointAssignment)(nil)
var _ infer.CustomDiff[CustomerProfileChannelEndpointAssignmentArgs, CustomerProfileChannelEndpointAssignmentState] = (*CustomerProfileChannelEndpointAssignment)(nil)
var _ infer.CustomRead[CustomerProfileChannelEndpointAssignmentArgs, CustomerProfileChannelEndpointAssignmentState] = (*CustomerProfileChannelEndpointAssignment)(nil)
var _ infer.CustomDelete[CustomerProfileChannelEndpointAssignmentState] = (*CustomerProfileChannelEndpointAssignment)(nil)

func (r *CustomerProfileChannelEndpointAssignment) Check(ctx context.Context, req infer.CheckRequest) (infer.CheckResponse[CustomerProfileChannelEndpointAssignmentArgs], error) {
	args, failures, err := infer.DefaultCheck[CustomerProfileChannelEndpointAssignmentArgs](ctx, req.NewInputs)
	if err != nil || len(failures) > 0 {
		return infer.CheckResponse[CustomerProfileChannelEndpointAssignmentArgs]{Inputs: args, Failures: failures}, err
	}
	c := &checker{unk: unknowns(req.NewInputs)}
	c.sid("customerProfileSid", args.CustomerProfileSid, bundleSidRE, "customer profile (BU...)")
	c.sid("channelEndpointSid", args.ChannelEndpointSid, channelEndpointSidRE, "channel endpoint, two letters and 32 hex digits (for example PN...)")
	c.required("channelEndpointType", args.ChannelEndpointType)
	return infer.CheckResponse[CustomerProfileChannelEndpointAssignmentArgs]{Inputs: args, Failures: c.failures}, nil
}

func (r *CustomerProfileChannelEndpointAssignment) Diff(_ context.Context, req infer.DiffRequest[CustomerProfileChannelEndpointAssignmentArgs, CustomerProfileChannelEndpointAssignmentState]) (infer.DiffResponse, error) {
	o, n := req.State.CustomerProfileChannelEndpointAssignmentArgs, req.Inputs
	d := trDiffSet{}
	d.replace("customerProfileSid", o.CustomerProfileSid, n.CustomerProfileSid)
	d.replace("channelEndpointType", o.ChannelEndpointType, n.ChannelEndpointType)
	d.replace("channelEndpointSid", o.ChannelEndpointSid, n.ChannelEndpointSid)
	return d.response(), nil
}

func (r *CustomerProfileChannelEndpointAssignment) Create(ctx context.Context, req infer.CreateRequest[CustomerProfileChannelEndpointAssignmentArgs]) (infer.CreateResponse[CustomerProfileChannelEndpointAssignmentState], error) {
	type resp = infer.CreateResponse[CustomerProfileChannelEndpointAssignmentState]
	a := req.Inputs
	st := CustomerProfileChannelEndpointAssignmentState{CustomerProfileChannelEndpointAssignmentArgs: a}
	if req.DryRun {
		return resp{Output: st}, nil
	}
	o, err := channelAssignmentCreate(ctx, "CustomerProfiles", a.CustomerProfileSid, a.ChannelEndpointType, a.ChannelEndpointSid)
	if err != nil {
		return resp{}, err
	}
	st.Sid, st.AccountSid, st.DateCreated = o.str("sid"), o.str("account_sid"), o.str("date_created")
	return resp{ID: a.CustomerProfileSid + "/" + st.Sid, Output: st}, nil
}

func (r *CustomerProfileChannelEndpointAssignment) Read(ctx context.Context, req infer.ReadRequest[CustomerProfileChannelEndpointAssignmentArgs, CustomerProfileChannelEndpointAssignmentState]) (infer.ReadResponse[CustomerProfileChannelEndpointAssignmentArgs, CustomerProfileChannelEndpointAssignmentState], error) {
	type resp = infer.ReadResponse[CustomerProfileChannelEndpointAssignmentArgs, CustomerProfileChannelEndpointAssignmentState]
	bundle, sid, err := splitChannelAssignmentID(req.ID)
	if err != nil {
		return resp{}, err
	}
	o, found, err := channelAssignmentFetch(ctx, "CustomerProfiles", bundle, sid)
	if err != nil {
		return resp{}, err
	}
	if !found {
		if err := notFoundOnImport("customer profile channel endpoint assignment", req.ID, req.Inputs, req.State); err != nil {
			return resp{}, err
		}
		return resp{}, nil
	}
	in := req.Inputs
	in.CustomerProfileSid, in.ChannelEndpointType, in.ChannelEndpointSid = bundle, o.str("channel_endpoint_type"), o.str("channel_endpoint_sid")
	st := CustomerProfileChannelEndpointAssignmentState{CustomerProfileChannelEndpointAssignmentArgs: in, Sid: sid, AccountSid: o.str("account_sid"), DateCreated: o.str("date_created")}
	return resp{ID: req.ID, Inputs: in, State: st}, nil
}

func (r *CustomerProfileChannelEndpointAssignment) Delete(ctx context.Context, req infer.DeleteRequest[CustomerProfileChannelEndpointAssignmentState]) (infer.DeleteResponse, error) {
	bundle, sid, err := splitChannelAssignmentID(req.ID)
	if err != nil {
		return infer.DeleteResponse{}, err
	}
	return infer.DeleteResponse{}, channelAssignmentDelete(ctx, "CustomerProfiles", bundle, sid)
}

// ------------------------------------------------------------ TrustProductChannelEndpointAssignment

// TrustProductChannelEndpointAssignment assigns a channel endpoint, such as a phone number, to a trust product.
type TrustProductChannelEndpointAssignment struct{}

// TrustProductChannelEndpointAssignmentArgs are the inputs of a TrustProductChannelEndpointAssignment.
type TrustProductChannelEndpointAssignmentArgs struct {
	TrustProductSid     string `pulumi:"trustProductSid" provider:"replaceOnChanges"`
	ChannelEndpointType string `pulumi:"channelEndpointType" provider:"replaceOnChanges"`
	ChannelEndpointSid  string `pulumi:"channelEndpointSid" provider:"replaceOnChanges"`
}

func (a *TrustProductChannelEndpointAssignmentArgs) Annotate(an infer.Annotator) {
	an.Describe(&a.TrustProductSid, "SID (BU...) of the trust product. Changing it replaces the assignment.")
	an.Describe(&a.ChannelEndpointType, "Type of the channel endpoint, for example phone-number. Changing it replaces the assignment.")
	an.Describe(&a.ChannelEndpointSid, "SID of the channel endpoint, for example the PN... SID of an incoming phone number. Changing it replaces the assignment.")
}

// TrustProductChannelEndpointAssignmentState is the persisted state of a TrustProductChannelEndpointAssignment.
type TrustProductChannelEndpointAssignmentState struct {
	TrustProductChannelEndpointAssignmentArgs

	Sid         string `pulumi:"sid"`
	AccountSid  string `pulumi:"accountSid"`
	DateCreated string `pulumi:"dateCreated"`
}

func (s *TrustProductChannelEndpointAssignmentState) Annotate(a infer.Annotator) {
	a.Describe(&s.Sid, "The assignment SID (RA...). The resource ID is the trust product SID and this SID joined by a slash.")
	a.Describe(&s.AccountSid, "SID (AC...) of the account that owns the assignment.")
	a.Describe(&s.DateCreated, "When the assignment was created (ISO 8601).")
}

func (r *TrustProductChannelEndpointAssignment) Annotate(a infer.Annotator) {
	a.Describe(&r, "Assigns a channel endpoint (a phone number) to a TrustHub trust product (trusthub.twilio.com /v1/TrustProducts/{sid}/ChannelEndpointAssignments). Every input replaces, because Twilio has no update call. "+
		"Twilio only accepts the assignment while the trust product is a draft or approved as it documents, and may refuse to remove one once the trust product is in review. Import with the ID <trustProductSid>/<assignmentSid>. Experimental, not verified against a live account.")
}

var _ infer.CustomResource[TrustProductChannelEndpointAssignmentArgs, TrustProductChannelEndpointAssignmentState] = (*TrustProductChannelEndpointAssignment)(nil)
var _ infer.CustomCheck[TrustProductChannelEndpointAssignmentArgs] = (*TrustProductChannelEndpointAssignment)(nil)
var _ infer.CustomDiff[TrustProductChannelEndpointAssignmentArgs, TrustProductChannelEndpointAssignmentState] = (*TrustProductChannelEndpointAssignment)(nil)
var _ infer.CustomRead[TrustProductChannelEndpointAssignmentArgs, TrustProductChannelEndpointAssignmentState] = (*TrustProductChannelEndpointAssignment)(nil)
var _ infer.CustomDelete[TrustProductChannelEndpointAssignmentState] = (*TrustProductChannelEndpointAssignment)(nil)

func (r *TrustProductChannelEndpointAssignment) Check(ctx context.Context, req infer.CheckRequest) (infer.CheckResponse[TrustProductChannelEndpointAssignmentArgs], error) {
	args, failures, err := infer.DefaultCheck[TrustProductChannelEndpointAssignmentArgs](ctx, req.NewInputs)
	if err != nil || len(failures) > 0 {
		return infer.CheckResponse[TrustProductChannelEndpointAssignmentArgs]{Inputs: args, Failures: failures}, err
	}
	c := &checker{unk: unknowns(req.NewInputs)}
	c.sid("trustProductSid", args.TrustProductSid, bundleSidRE, "trust product (BU...)")
	c.sid("channelEndpointSid", args.ChannelEndpointSid, channelEndpointSidRE, "channel endpoint, two letters and 32 hex digits (for example PN...)")
	c.required("channelEndpointType", args.ChannelEndpointType)
	return infer.CheckResponse[TrustProductChannelEndpointAssignmentArgs]{Inputs: args, Failures: c.failures}, nil
}

func (r *TrustProductChannelEndpointAssignment) Diff(_ context.Context, req infer.DiffRequest[TrustProductChannelEndpointAssignmentArgs, TrustProductChannelEndpointAssignmentState]) (infer.DiffResponse, error) {
	o, n := req.State.TrustProductChannelEndpointAssignmentArgs, req.Inputs
	d := trDiffSet{}
	d.replace("trustProductSid", o.TrustProductSid, n.TrustProductSid)
	d.replace("channelEndpointType", o.ChannelEndpointType, n.ChannelEndpointType)
	d.replace("channelEndpointSid", o.ChannelEndpointSid, n.ChannelEndpointSid)
	return d.response(), nil
}

func (r *TrustProductChannelEndpointAssignment) Create(ctx context.Context, req infer.CreateRequest[TrustProductChannelEndpointAssignmentArgs]) (infer.CreateResponse[TrustProductChannelEndpointAssignmentState], error) {
	type resp = infer.CreateResponse[TrustProductChannelEndpointAssignmentState]
	a := req.Inputs
	st := TrustProductChannelEndpointAssignmentState{TrustProductChannelEndpointAssignmentArgs: a}
	if req.DryRun {
		return resp{Output: st}, nil
	}
	o, err := channelAssignmentCreate(ctx, "TrustProducts", a.TrustProductSid, a.ChannelEndpointType, a.ChannelEndpointSid)
	if err != nil {
		return resp{}, err
	}
	st.Sid, st.AccountSid, st.DateCreated = o.str("sid"), o.str("account_sid"), o.str("date_created")
	return resp{ID: a.TrustProductSid + "/" + st.Sid, Output: st}, nil
}

func (r *TrustProductChannelEndpointAssignment) Read(ctx context.Context, req infer.ReadRequest[TrustProductChannelEndpointAssignmentArgs, TrustProductChannelEndpointAssignmentState]) (infer.ReadResponse[TrustProductChannelEndpointAssignmentArgs, TrustProductChannelEndpointAssignmentState], error) {
	type resp = infer.ReadResponse[TrustProductChannelEndpointAssignmentArgs, TrustProductChannelEndpointAssignmentState]
	bundle, sid, err := splitChannelAssignmentID(req.ID)
	if err != nil {
		return resp{}, err
	}
	o, found, err := channelAssignmentFetch(ctx, "TrustProducts", bundle, sid)
	if err != nil {
		return resp{}, err
	}
	if !found {
		if err := notFoundOnImport("trust product channel endpoint assignment", req.ID, req.Inputs, req.State); err != nil {
			return resp{}, err
		}
		return resp{}, nil
	}
	in := req.Inputs
	in.TrustProductSid, in.ChannelEndpointType, in.ChannelEndpointSid = bundle, o.str("channel_endpoint_type"), o.str("channel_endpoint_sid")
	st := TrustProductChannelEndpointAssignmentState{TrustProductChannelEndpointAssignmentArgs: in, Sid: sid, AccountSid: o.str("account_sid"), DateCreated: o.str("date_created")}
	return resp{ID: req.ID, Inputs: in, State: st}, nil
}

func (r *TrustProductChannelEndpointAssignment) Delete(ctx context.Context, req infer.DeleteRequest[TrustProductChannelEndpointAssignmentState]) (infer.DeleteResponse, error) {
	bundle, sid, err := splitChannelAssignmentID(req.ID)
	if err != nil {
		return infer.DeleteResponse{}, err
	}
	return infer.DeleteResponse{}, channelAssignmentDelete(ctx, "TrustProducts", bundle, sid)
}
