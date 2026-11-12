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

	p "github.com/pulumi/pulumi-go-provider"
	"github.com/pulumi/pulumi-go-provider/infer"
)

// Function and Asset containers share one shape: a service child with a friendly name. What goes inside
// them (versions) is uploaded to serverless-upload.twilio.com, which the OpenAPI document does not
// describe, so these resources manage the container only.

// ------------------------------------------------------------ ServerlessFunction

// ServerlessFunction manages the function container of a Serverless service. Function code is uploaded as immutable versions through Twilio's upload host, which this provider does not manage (use the Serverless Toolkit or the Twilio CLI to deploy code).
type ServerlessFunction struct{}

// ServerlessFunctionArgs are the inputs of a ServerlessFunction.
type ServerlessFunctionArgs struct {
	ServiceSid   string `pulumi:"serviceSid" provider:"replaceOnChanges"`
	FriendlyName string `pulumi:"friendlyName"`
}

func (a *ServerlessFunctionArgs) Annotate(an infer.Annotator) {
	an.Describe(&a.ServiceSid, "SID (ZS...) of the Serverless service. Changing it replaces the resource.")
	an.Describe(&a.FriendlyName, "Human-readable name of the function, at most 255 characters. It is the only property Twilio lets you change.")
}

// ServerlessFunctionState is the persisted state of a ServerlessFunction.
type ServerlessFunctionState struct {
	ServerlessFunctionArgs

	// Sid is the function SID (ZH...).
	Sid         string `pulumi:"sid"`
	AccountSid  string `pulumi:"accountSid"`
	Url         string `pulumi:"url"`
	DateCreated string `pulumi:"dateCreated"`
	DateUpdated string `pulumi:"dateUpdated"`
}

func (s *ServerlessFunctionState) Annotate(a infer.Annotator) {
	a.Describe(&s.Sid, "The function SID (ZH...).")
	a.Describe(&s.AccountSid, "SID of the account that owns the function.")
	a.Describe(&s.Url, "The absolute URL of the function resource.")
	a.Describe(&s.DateCreated, "When the function was created (ISO 8601).")
	a.Describe(&s.DateUpdated, "When the function was last updated (ISO 8601).")
}

func (r *ServerlessFunction) Annotate(a infer.Annotator) {
	a.Describe(&r, "The function container of a Twilio Serverless service. Function code is uploaded as immutable versions through Twilio's upload host, which this provider does not manage (use the Serverless Toolkit or the Twilio CLI to deploy code). Twilio refuses to delete a function whose version a build holds, so destroying such a function only removes it from Pulumi state with a warning; it stays on Twilio until its service is deleted. The resource ID is <serviceSid>/<sid>.")
}

var _ infer.CustomResource[ServerlessFunctionArgs, ServerlessFunctionState] = (*ServerlessFunction)(nil)
var _ infer.CustomCheck[ServerlessFunctionArgs] = (*ServerlessFunction)(nil)
var _ infer.CustomDiff[ServerlessFunctionArgs, ServerlessFunctionState] = (*ServerlessFunction)(nil)
var _ infer.CustomRead[ServerlessFunctionArgs, ServerlessFunctionState] = (*ServerlessFunction)(nil)
var _ infer.CustomUpdate[ServerlessFunctionArgs, ServerlessFunctionState] = (*ServerlessFunction)(nil)
var _ infer.CustomDelete[ServerlessFunctionState] = (*ServerlessFunction)(nil)

func (r *ServerlessFunction) Check(ctx context.Context, req infer.CheckRequest) (infer.CheckResponse[ServerlessFunctionArgs], error) {
	args, failures, err := infer.DefaultCheck[ServerlessFunctionArgs](ctx, req.NewInputs)
	if err != nil || len(failures) > 0 {
		return infer.CheckResponse[ServerlessFunctionArgs]{Inputs: args, Failures: failures}, err
	}
	c := &checker{unk: unknowns(req.NewInputs)}
	c.sid("serviceSid", args.ServiceSid, serverlessServiceSidRE, "Serverless service (ZS followed by 32 hex characters)")
	c.required("friendlyName", args.FriendlyName)
	c.maxLen("friendlyName", args.FriendlyName, 255)
	return infer.CheckResponse[ServerlessFunctionArgs]{Inputs: args, Failures: c.failures}, nil
}

func (r *ServerlessFunction) Diff(_ context.Context, req infer.DiffRequest[ServerlessFunctionArgs, ServerlessFunctionState]) (infer.DiffResponse, error) {
	o, n := req.State.ServerlessFunctionArgs, req.Inputs
	detail := map[string]p.PropertyDiff{}
	if o.ServiceSid != n.ServiceSid {
		detail["serviceSid"] = p.PropertyDiff{Kind: p.UpdateReplace}
	}
	if o.FriendlyName != n.FriendlyName {
		detail["friendlyName"] = p.PropertyDiff{Kind: p.Update}
	}
	return infer.DiffResponse{HasChanges: len(detail) > 0, DetailedDiff: detail}, nil
}

func serverlessFunctionOutputs(st *ServerlessFunctionState, o obj) {
	st.Sid, st.AccountSid, st.Url = o.str("sid"), o.str("account_sid"), o.str("url")
	st.DateCreated, st.DateUpdated = o.str("date_created"), o.str("date_updated")
}

func (r *ServerlessFunction) Create(ctx context.Context, req infer.CreateRequest[ServerlessFunctionArgs]) (infer.CreateResponse[ServerlessFunctionState], error) {
	a := req.Inputs
	if req.DryRun {
		return infer.CreateResponse[ServerlessFunctionState]{Output: ServerlessFunctionState{ServerlessFunctionArgs: a}}, nil
	}
	c, err := clientFor(ctx)
	if err != nil {
		return infer.CreateResponse[ServerlessFunctionState]{}, err
	}
	o, err := c.do(ctx, "POST", "serverless", serverlessSvcPath(a.ServiceSid, "Functions", ""), nil, url.Values{"FriendlyName": {a.FriendlyName}})
	if err != nil {
		return infer.CreateResponse[ServerlessFunctionState]{}, fmt.Errorf("create function %q on serverless service %s: %w", a.FriendlyName, a.ServiceSid, err)
	}
	st := ServerlessFunctionState{ServerlessFunctionArgs: a}
	serverlessFunctionOutputs(&st, o)
	return infer.CreateResponse[ServerlessFunctionState]{ID: a.ServiceSid + "/" + st.Sid, Output: st}, nil
}

func (r *ServerlessFunction) Read(ctx context.Context, req infer.ReadRequest[ServerlessFunctionArgs, ServerlessFunctionState]) (infer.ReadResponse[ServerlessFunctionArgs, ServerlessFunctionState], error) {
	type resp = infer.ReadResponse[ServerlessFunctionArgs, ServerlessFunctionState]
	ids, err := splitVerifyID(req.ID, 2, "<serviceSid>/<sid>")
	if err != nil {
		return resp{}, err
	}
	c, err := clientFor(ctx)
	if err != nil {
		return resp{}, err
	}
	o, err := c.do(ctx, "GET", "serverless", serverlessSvcPath(ids[0], "Functions", ids[1]), nil, nil)
	if err != nil {
		if isNotFound(err) {
			if err := notFoundOnImport("serverless function", req.ID, req.Inputs, req.State); err != nil {
				return resp{}, err
			}
			return resp{}, nil
		}
		return resp{}, fmt.Errorf("read serverless function %s: %w", req.ID, err)
	}
	in := req.Inputs
	if in.ServiceSid == "" {
		in = ServerlessFunctionArgs{ServiceSid: ids[0]}
	}
	if v := o.str("friendly_name"); v != "" {
		in.FriendlyName = v
	}
	st := ServerlessFunctionState{ServerlessFunctionArgs: in}
	serverlessFunctionOutputs(&st, o)
	return resp{ID: req.ID, Inputs: in, State: st}, nil
}

func (r *ServerlessFunction) Update(ctx context.Context, req infer.UpdateRequest[ServerlessFunctionArgs, ServerlessFunctionState]) (infer.UpdateResponse[ServerlessFunctionState], error) {
	a := req.Inputs
	st := req.State
	st.ServerlessFunctionArgs = a
	if req.DryRun {
		return infer.UpdateResponse[ServerlessFunctionState]{Output: st}, nil
	}
	ids, err := splitVerifyID(req.ID, 2, "<serviceSid>/<sid>")
	if err != nil {
		return infer.UpdateResponse[ServerlessFunctionState]{}, err
	}
	c, err := clientFor(ctx)
	if err != nil {
		return infer.UpdateResponse[ServerlessFunctionState]{}, err
	}
	o, err := c.do(ctx, "POST", "serverless", serverlessSvcPath(ids[0], "Functions", ids[1]), nil, url.Values{"FriendlyName": {a.FriendlyName}})
	if err != nil {
		return infer.UpdateResponse[ServerlessFunctionState]{}, fmt.Errorf("update serverless function %s: %w", req.ID, err)
	}
	serverlessFunctionOutputs(&st, o)
	return infer.UpdateResponse[ServerlessFunctionState]{Output: st}, nil
}

func (r *ServerlessFunction) Delete(ctx context.Context, req infer.DeleteRequest[ServerlessFunctionState]) (infer.DeleteResponse, error) {
	ids, err := splitVerifyID(req.ID, 2, "<serviceSid>/<sid>")
	if err != nil {
		return infer.DeleteResponse{}, err
	}
	c, err := clientFor(ctx)
	if err != nil {
		return infer.DeleteResponse{}, err
	}
	if _, err := c.do(ctx, "DELETE", "serverless", serverlessSvcPath(ids[0], "Functions", ids[1]), nil, nil); err != nil && !isNotFound(err) {
		// A version referenced by a Build cannot be deleted, and a Build that was deployed cannot be
		// deleted either (HTTP 400, error 20001): forget the function, deleting the Service removes it (#1856).
		if isRefusal(err, "referenced in a build") {
			p.GetLogger(ctx).Warningf("serverless function %s has a version referenced in a build, which Twilio cannot delete: it is removed from the state "+
				"and stays on Twilio until its service is deleted", req.ID)
			return infer.DeleteResponse{}, nil
		}
		return infer.DeleteResponse{}, fmt.Errorf("delete serverless function %s: %w", req.ID, err)
	}
	return infer.DeleteResponse{}, nil
}

// ------------------------------------------------------------ ServerlessAsset

// ServerlessAsset manages the asset container of a Serverless service. Asset files are uploaded as immutable versions through Twilio's upload host, which this provider does not manage (use the Serverless Toolkit or the Twilio CLI to upload files).
type ServerlessAsset struct{}

// ServerlessAssetArgs are the inputs of a ServerlessAsset.
type ServerlessAssetArgs struct {
	ServiceSid   string `pulumi:"serviceSid" provider:"replaceOnChanges"`
	FriendlyName string `pulumi:"friendlyName"`
}

func (a *ServerlessAssetArgs) Annotate(an infer.Annotator) {
	an.Describe(&a.ServiceSid, "SID (ZS...) of the Serverless service. Changing it replaces the resource.")
	an.Describe(&a.FriendlyName, "Human-readable name of the asset, at most 255 characters. It is the only property Twilio lets you change.")
}

// ServerlessAssetState is the persisted state of a ServerlessAsset.
type ServerlessAssetState struct {
	ServerlessAssetArgs

	// Sid is the asset SID (ZH...).
	Sid         string `pulumi:"sid"`
	AccountSid  string `pulumi:"accountSid"`
	Url         string `pulumi:"url"`
	DateCreated string `pulumi:"dateCreated"`
	DateUpdated string `pulumi:"dateUpdated"`
}

func (s *ServerlessAssetState) Annotate(a infer.Annotator) {
	a.Describe(&s.Sid, "The asset SID (ZH...).")
	a.Describe(&s.AccountSid, "SID of the account that owns the asset.")
	a.Describe(&s.Url, "The absolute URL of the asset resource.")
	a.Describe(&s.DateCreated, "When the asset was created (ISO 8601).")
	a.Describe(&s.DateUpdated, "When the asset was last updated (ISO 8601).")
}

func (r *ServerlessAsset) Annotate(a infer.Annotator) {
	a.Describe(&r, "The asset container of a Twilio Serverless service. Asset files are uploaded as immutable versions through Twilio's upload host, which this provider does not manage (use the Serverless Toolkit or the Twilio CLI to upload files). Twilio refuses to delete an asset whose version a build holds, so destroying such an asset only removes it from Pulumi state with a warning; it stays on Twilio until its service is deleted. The resource ID is <serviceSid>/<sid>.")
}

var _ infer.CustomResource[ServerlessAssetArgs, ServerlessAssetState] = (*ServerlessAsset)(nil)
var _ infer.CustomCheck[ServerlessAssetArgs] = (*ServerlessAsset)(nil)
var _ infer.CustomDiff[ServerlessAssetArgs, ServerlessAssetState] = (*ServerlessAsset)(nil)
var _ infer.CustomRead[ServerlessAssetArgs, ServerlessAssetState] = (*ServerlessAsset)(nil)
var _ infer.CustomUpdate[ServerlessAssetArgs, ServerlessAssetState] = (*ServerlessAsset)(nil)
var _ infer.CustomDelete[ServerlessAssetState] = (*ServerlessAsset)(nil)

func (r *ServerlessAsset) Check(ctx context.Context, req infer.CheckRequest) (infer.CheckResponse[ServerlessAssetArgs], error) {
	args, failures, err := infer.DefaultCheck[ServerlessAssetArgs](ctx, req.NewInputs)
	if err != nil || len(failures) > 0 {
		return infer.CheckResponse[ServerlessAssetArgs]{Inputs: args, Failures: failures}, err
	}
	c := &checker{unk: unknowns(req.NewInputs)}
	c.sid("serviceSid", args.ServiceSid, serverlessServiceSidRE, "Serverless service (ZS followed by 32 hex characters)")
	c.required("friendlyName", args.FriendlyName)
	c.maxLen("friendlyName", args.FriendlyName, 255)
	return infer.CheckResponse[ServerlessAssetArgs]{Inputs: args, Failures: c.failures}, nil
}

func (r *ServerlessAsset) Diff(_ context.Context, req infer.DiffRequest[ServerlessAssetArgs, ServerlessAssetState]) (infer.DiffResponse, error) {
	o, n := req.State.ServerlessAssetArgs, req.Inputs
	detail := map[string]p.PropertyDiff{}
	if o.ServiceSid != n.ServiceSid {
		detail["serviceSid"] = p.PropertyDiff{Kind: p.UpdateReplace}
	}
	if o.FriendlyName != n.FriendlyName {
		detail["friendlyName"] = p.PropertyDiff{Kind: p.Update}
	}
	return infer.DiffResponse{HasChanges: len(detail) > 0, DetailedDiff: detail}, nil
}

func serverlessAssetOutputs(st *ServerlessAssetState, o obj) {
	st.Sid, st.AccountSid, st.Url = o.str("sid"), o.str("account_sid"), o.str("url")
	st.DateCreated, st.DateUpdated = o.str("date_created"), o.str("date_updated")
}

func (r *ServerlessAsset) Create(ctx context.Context, req infer.CreateRequest[ServerlessAssetArgs]) (infer.CreateResponse[ServerlessAssetState], error) {
	a := req.Inputs
	if req.DryRun {
		return infer.CreateResponse[ServerlessAssetState]{Output: ServerlessAssetState{ServerlessAssetArgs: a}}, nil
	}
	c, err := clientFor(ctx)
	if err != nil {
		return infer.CreateResponse[ServerlessAssetState]{}, err
	}
	o, err := c.do(ctx, "POST", "serverless", serverlessSvcPath(a.ServiceSid, "Assets", ""), nil, url.Values{"FriendlyName": {a.FriendlyName}})
	if err != nil {
		return infer.CreateResponse[ServerlessAssetState]{}, fmt.Errorf("create asset %q on serverless service %s: %w", a.FriendlyName, a.ServiceSid, err)
	}
	st := ServerlessAssetState{ServerlessAssetArgs: a}
	serverlessAssetOutputs(&st, o)
	return infer.CreateResponse[ServerlessAssetState]{ID: a.ServiceSid + "/" + st.Sid, Output: st}, nil
}

func (r *ServerlessAsset) Read(ctx context.Context, req infer.ReadRequest[ServerlessAssetArgs, ServerlessAssetState]) (infer.ReadResponse[ServerlessAssetArgs, ServerlessAssetState], error) {
	type resp = infer.ReadResponse[ServerlessAssetArgs, ServerlessAssetState]
	ids, err := splitVerifyID(req.ID, 2, "<serviceSid>/<sid>")
	if err != nil {
		return resp{}, err
	}
	c, err := clientFor(ctx)
	if err != nil {
		return resp{}, err
	}
	o, err := c.do(ctx, "GET", "serverless", serverlessSvcPath(ids[0], "Assets", ids[1]), nil, nil)
	if err != nil {
		if isNotFound(err) {
			if err := notFoundOnImport("serverless asset", req.ID, req.Inputs, req.State); err != nil {
				return resp{}, err
			}
			return resp{}, nil
		}
		return resp{}, fmt.Errorf("read serverless asset %s: %w", req.ID, err)
	}
	in := req.Inputs
	if in.ServiceSid == "" {
		in = ServerlessAssetArgs{ServiceSid: ids[0]}
	}
	if v := o.str("friendly_name"); v != "" {
		in.FriendlyName = v
	}
	st := ServerlessAssetState{ServerlessAssetArgs: in}
	serverlessAssetOutputs(&st, o)
	return resp{ID: req.ID, Inputs: in, State: st}, nil
}

func (r *ServerlessAsset) Update(ctx context.Context, req infer.UpdateRequest[ServerlessAssetArgs, ServerlessAssetState]) (infer.UpdateResponse[ServerlessAssetState], error) {
	a := req.Inputs
	st := req.State
	st.ServerlessAssetArgs = a
	if req.DryRun {
		return infer.UpdateResponse[ServerlessAssetState]{Output: st}, nil
	}
	ids, err := splitVerifyID(req.ID, 2, "<serviceSid>/<sid>")
	if err != nil {
		return infer.UpdateResponse[ServerlessAssetState]{}, err
	}
	c, err := clientFor(ctx)
	if err != nil {
		return infer.UpdateResponse[ServerlessAssetState]{}, err
	}
	o, err := c.do(ctx, "POST", "serverless", serverlessSvcPath(ids[0], "Assets", ids[1]), nil, url.Values{"FriendlyName": {a.FriendlyName}})
	if err != nil {
		return infer.UpdateResponse[ServerlessAssetState]{}, fmt.Errorf("update serverless asset %s: %w", req.ID, err)
	}
	serverlessAssetOutputs(&st, o)
	return infer.UpdateResponse[ServerlessAssetState]{Output: st}, nil
}

func (r *ServerlessAsset) Delete(ctx context.Context, req infer.DeleteRequest[ServerlessAssetState]) (infer.DeleteResponse, error) {
	ids, err := splitVerifyID(req.ID, 2, "<serviceSid>/<sid>")
	if err != nil {
		return infer.DeleteResponse{}, err
	}
	c, err := clientFor(ctx)
	if err != nil {
		return infer.DeleteResponse{}, err
	}
	if _, err := c.do(ctx, "DELETE", "serverless", serverlessSvcPath(ids[0], "Assets", ids[1]), nil, nil); err != nil && !isNotFound(err) {
		// A version referenced by a Build cannot be deleted, and a Build that was deployed cannot be
		// deleted either (HTTP 400, error 20001): forget the asset, deleting the Service removes it (#1856).
		if isRefusal(err, "referenced in a build") {
			p.GetLogger(ctx).Warningf("serverless asset %s has a version referenced in a build, which Twilio cannot delete: it is removed from the state "+
				"and stays on Twilio until its service is deleted", req.ID)
			return infer.DeleteResponse{}, nil
		}
		return infer.DeleteResponse{}, fmt.Errorf("delete serverless asset %s: %w", req.ID, err)
	}
	return infer.DeleteResponse{}, nil
}
