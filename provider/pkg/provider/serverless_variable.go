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

// ------------------------------------------------------------ ServerlessVariable

// ServerlessVariable manages an environment variable of a Serverless environment.
type ServerlessVariable struct{}

// ServerlessVariableArgs are the inputs of a ServerlessVariable.
type ServerlessVariableArgs struct {
	ServiceSid     string `pulumi:"serviceSid" provider:"replaceOnChanges"`
	EnvironmentSid string `pulumi:"environmentSid" provider:"replaceOnChanges"`
	Key            string `pulumi:"key"`
	Value          string `pulumi:"value" provider:"secret"`
}

func (a *ServerlessVariableArgs) Annotate(an infer.Annotator) {
	an.Describe(&a.ServiceSid, "SID (ZS...) of the Serverless service. Changing it replaces the resource.")
	an.Describe(&a.EnvironmentSid, "SID (ZE...) of the environment. Changing it replaces the resource.")
	an.Describe(&a.Key, "Name of the variable, as the Function reads it from context. Changing it updates the variable in place.")
	an.Describe(&a.Value, "Value of the variable. It is a Pulumi secret, because variables commonly carry credentials.")
}

// ServerlessVariableState is the persisted state of a ServerlessVariable.
type ServerlessVariableState struct {
	ServerlessVariableArgs

	// Sid is the variable SID (ZV...).
	Sid         string `pulumi:"sid"`
	DateCreated string `pulumi:"dateCreated"`
	DateUpdated string `pulumi:"dateUpdated"`
}

func (s *ServerlessVariableState) Annotate(a infer.Annotator) {
	a.Describe(&s.Sid, "The variable SID (ZV...).")
	a.Describe(&s.DateCreated, "When the variable was created (ISO 8601).")
	a.Describe(&s.DateUpdated, "When the variable was last updated (ISO 8601).")
}

func (r *ServerlessVariable) Annotate(a infer.Annotator) {
	a.Describe(&r, "An environment variable of a Twilio Serverless environment. The value is a Pulumi secret. The resource ID is <serviceSid>/<environmentSid>/<sid>.")
}

var _ infer.CustomResource[ServerlessVariableArgs, ServerlessVariableState] = (*ServerlessVariable)(nil)
var _ infer.CustomCheck[ServerlessVariableArgs] = (*ServerlessVariable)(nil)
var _ infer.CustomDiff[ServerlessVariableArgs, ServerlessVariableState] = (*ServerlessVariable)(nil)
var _ infer.CustomRead[ServerlessVariableArgs, ServerlessVariableState] = (*ServerlessVariable)(nil)
var _ infer.CustomUpdate[ServerlessVariableArgs, ServerlessVariableState] = (*ServerlessVariable)(nil)
var _ infer.CustomDelete[ServerlessVariableState] = (*ServerlessVariable)(nil)

func (r *ServerlessVariable) Check(ctx context.Context, req infer.CheckRequest) (infer.CheckResponse[ServerlessVariableArgs], error) {
	args, failures, err := infer.DefaultCheck[ServerlessVariableArgs](ctx, req.NewInputs)
	if err != nil || len(failures) > 0 {
		return infer.CheckResponse[ServerlessVariableArgs]{Inputs: args, Failures: failures}, err
	}
	c := &checker{unk: unknowns(req.NewInputs)}
	c.sid("serviceSid", args.ServiceSid, serverlessServiceSidRE, "Serverless service (ZS followed by 32 hex characters)")
	c.sid("environmentSid", args.EnvironmentSid, serverlessEnvironmentSidRE, "Serverless environment (ZE followed by 32 hex characters)")
	c.required("key", args.Key)
	return infer.CheckResponse[ServerlessVariableArgs]{Inputs: args, Failures: c.failures}, nil
}

func (r *ServerlessVariable) Diff(_ context.Context, req infer.DiffRequest[ServerlessVariableArgs, ServerlessVariableState]) (infer.DiffResponse, error) {
	o, n := req.State.ServerlessVariableArgs, req.Inputs
	detail := map[string]p.PropertyDiff{}
	if o.ServiceSid != n.ServiceSid {
		detail["serviceSid"] = p.PropertyDiff{Kind: p.UpdateReplace}
	}
	if o.EnvironmentSid != n.EnvironmentSid {
		detail["environmentSid"] = p.PropertyDiff{Kind: p.UpdateReplace}
	}
	if o.Key != n.Key {
		detail["key"] = p.PropertyDiff{Kind: p.Update}
	}
	if o.Value != n.Value {
		detail["value"] = p.PropertyDiff{Kind: p.Update}
	}
	return infer.DiffResponse{HasChanges: len(detail) > 0, DetailedDiff: detail}, nil
}

func serverlessVariableOutputs(st *ServerlessVariableState, o obj) {
	st.Sid, st.DateCreated, st.DateUpdated = o.str("sid"), o.str("date_created"), o.str("date_updated")
}

func (r *ServerlessVariable) Create(ctx context.Context, req infer.CreateRequest[ServerlessVariableArgs]) (infer.CreateResponse[ServerlessVariableState], error) {
	a := req.Inputs
	if req.DryRun {
		return infer.CreateResponse[ServerlessVariableState]{Output: ServerlessVariableState{ServerlessVariableArgs: a}}, nil
	}
	c, err := clientFor(ctx)
	if err != nil {
		return infer.CreateResponse[ServerlessVariableState]{}, err
	}
	form := url.Values{"Key": {a.Key}, "Value": {a.Value}}
	o, err := c.do(ctx, "POST", "serverless", serverlessEnvPath(a.ServiceSid, a.EnvironmentSid, ""), nil, form)
	if err != nil {
		return infer.CreateResponse[ServerlessVariableState]{}, fmt.Errorf("create variable %q on serverless environment %s: %w", a.Key, a.EnvironmentSid, err)
	}
	st := ServerlessVariableState{ServerlessVariableArgs: a}
	serverlessVariableOutputs(&st, o)
	return infer.CreateResponse[ServerlessVariableState]{ID: a.ServiceSid + "/" + a.EnvironmentSid + "/" + st.Sid, Output: st}, nil
}

func (r *ServerlessVariable) Read(ctx context.Context, req infer.ReadRequest[ServerlessVariableArgs, ServerlessVariableState]) (infer.ReadResponse[ServerlessVariableArgs, ServerlessVariableState], error) {
	type resp = infer.ReadResponse[ServerlessVariableArgs, ServerlessVariableState]
	ids, err := splitVerifyID(req.ID, 3, "<serviceSid>/<environmentSid>/<sid>")
	if err != nil {
		return resp{}, err
	}
	c, err := clientFor(ctx)
	if err != nil {
		return resp{}, err
	}
	o, err := c.do(ctx, "GET", "serverless", serverlessEnvPath(ids[0], ids[1], ids[2]), nil, nil)
	if err != nil {
		if isNotFound(err) {
			if err := notFoundOnImport("serverless variable", req.ID, req.Inputs, req.State); err != nil {
				return resp{}, err
			}
			return resp{}, nil
		}
		return resp{}, fmt.Errorf("read serverless variable %s: %w", req.ID, err)
	}
	in := req.Inputs
	if in.ServiceSid == "" {
		in = ServerlessVariableArgs{ServiceSid: ids[0], EnvironmentSid: ids[1]}
	}
	in.Key = o.str("key")
	// Keep the recorded value when Twilio omits it, rather than turning it into an empty string.
	if v := o.optStr("value"); v != nil {
		in.Value = *v
	}
	st := ServerlessVariableState{ServerlessVariableArgs: in}
	serverlessVariableOutputs(&st, o)
	return resp{ID: req.ID, Inputs: in, State: st}, nil
}

func (r *ServerlessVariable) Update(ctx context.Context, req infer.UpdateRequest[ServerlessVariableArgs, ServerlessVariableState]) (infer.UpdateResponse[ServerlessVariableState], error) {
	a := req.Inputs
	st := req.State
	st.ServerlessVariableArgs = a
	if req.DryRun {
		return infer.UpdateResponse[ServerlessVariableState]{Output: st}, nil
	}
	ids, err := splitVerifyID(req.ID, 3, "<serviceSid>/<environmentSid>/<sid>")
	if err != nil {
		return infer.UpdateResponse[ServerlessVariableState]{}, err
	}
	c, err := clientFor(ctx)
	if err != nil {
		return infer.UpdateResponse[ServerlessVariableState]{}, err
	}
	o, err := c.do(ctx, "POST", "serverless", serverlessEnvPath(ids[0], ids[1], ids[2]), nil, url.Values{"Key": {a.Key}, "Value": {a.Value}})
	if err != nil {
		return infer.UpdateResponse[ServerlessVariableState]{}, fmt.Errorf("update serverless variable %s: %w", req.ID, err)
	}
	serverlessVariableOutputs(&st, o)
	return infer.UpdateResponse[ServerlessVariableState]{Output: st}, nil
}

func (r *ServerlessVariable) Delete(ctx context.Context, req infer.DeleteRequest[ServerlessVariableState]) (infer.DeleteResponse, error) {
	ids, err := splitVerifyID(req.ID, 3, "<serviceSid>/<environmentSid>/<sid>")
	if err != nil {
		return infer.DeleteResponse{}, err
	}
	c, err := clientFor(ctx)
	if err != nil {
		return infer.DeleteResponse{}, err
	}
	if _, err := c.do(ctx, "DELETE", "serverless", serverlessEnvPath(ids[0], ids[1], ids[2]), nil, nil); err != nil && !isNotFound(err) {
		return infer.DeleteResponse{}, fmt.Errorf("delete serverless variable %s: %w", req.ID, err)
	}
	return infer.DeleteResponse{}, nil
}
