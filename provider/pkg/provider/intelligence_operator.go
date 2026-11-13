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

// IntelligenceCustomOperator manages a custom Voice Intelligence operator: a prompt-style analysis
// that runs over transcripts, built from an operator type.
type IntelligenceCustomOperator struct{}

// IntelligenceCustomOperatorArgs are the inputs of an IntelligenceCustomOperator.
type IntelligenceCustomOperatorArgs struct {
	FriendlyName string `pulumi:"friendlyName"`
	OperatorType string `pulumi:"operatorType"`
	Config       string `pulumi:"config"`
}

func (a *IntelligenceCustomOperatorArgs) Annotate(an infer.Annotator) {
	an.Describe(&a.FriendlyName, "Human-readable name of the operator, up to 64 characters.")
	an.Describe(&a.OperatorType, "The operator type the operator is built from (an existing Twilio operator type). Twilio does not allow changing it, so changing it replaces the operator.")
	an.Describe(&a.Config, "The operator configuration as a JSON object, following the schema of the operator type. Key order and formatting are not significant.")
}

// IntelligenceCustomOperatorState is the persisted state of an IntelligenceCustomOperator.
type IntelligenceCustomOperatorState struct {
	IntelligenceCustomOperatorArgs

	Sid          string  `pulumi:"sid"`
	Description  *string `pulumi:"description,optional"`
	Availability *string `pulumi:"availability,optional"`
	DateCreated  string  `pulumi:"dateCreated"`
	DateUpdated  string  `pulumi:"dateUpdated"`
}

func (s *IntelligenceCustomOperatorState) Annotate(a infer.Annotator) {
	a.Describe(&s.Sid, "The operator SID (LY...).")
	a.Describe(&s.Description, "Description Twilio reports for the operator.")
	a.Describe(&s.Availability, "Availability Twilio reports for the operator (internal, beta, public, retired, general-availability or deprecated).")
	a.Describe(&s.DateCreated, "When the operator was created (ISO 8601).")
	a.Describe(&s.DateUpdated, "When the operator was last updated (ISO 8601).")
}

func (r *IntelligenceCustomOperator) Annotate(a infer.Annotator) {
	a.Describe(&r, "A custom Voice Intelligence operator. The resource ID is the operator SID. Updating changes the name and configuration.")
}

var _ infer.CustomResource[IntelligenceCustomOperatorArgs, IntelligenceCustomOperatorState] = (*IntelligenceCustomOperator)(nil)
var _ infer.CustomCheck[IntelligenceCustomOperatorArgs] = (*IntelligenceCustomOperator)(nil)
var _ infer.CustomDiff[IntelligenceCustomOperatorArgs, IntelligenceCustomOperatorState] = (*IntelligenceCustomOperator)(nil)
var _ infer.CustomRead[IntelligenceCustomOperatorArgs, IntelligenceCustomOperatorState] = (*IntelligenceCustomOperator)(nil)
var _ infer.CustomUpdate[IntelligenceCustomOperatorArgs, IntelligenceCustomOperatorState] = (*IntelligenceCustomOperator)(nil)
var _ infer.CustomDelete[IntelligenceCustomOperatorState] = (*IntelligenceCustomOperator)(nil)

func (r *IntelligenceCustomOperator) Check(ctx context.Context, req infer.CheckRequest) (infer.CheckResponse[IntelligenceCustomOperatorArgs], error) {
	args, failures, err := infer.DefaultCheck[IntelligenceCustomOperatorArgs](ctx, req.NewInputs)
	if err != nil || len(failures) > 0 {
		return infer.CheckResponse[IntelligenceCustomOperatorArgs]{Inputs: args, Failures: failures}, err
	}
	c := &checker{unk: unknowns(req.NewInputs)}
	c.required("friendlyName", args.FriendlyName)
	c.maxLen("friendlyName", args.FriendlyName, 64)
	c.required("operatorType", args.OperatorType)
	if !c.unk("config") && !jsonObject(args.Config) {
		c.fail("config", "config must be a JSON object")
	}
	return infer.CheckResponse[IntelligenceCustomOperatorArgs]{Inputs: args, Failures: c.failures}, nil
}

func (r *IntelligenceCustomOperator) Diff(_ context.Context, req infer.DiffRequest[IntelligenceCustomOperatorArgs, IntelligenceCustomOperatorState]) (infer.DiffResponse, error) {
	o, n := req.State.IntelligenceCustomOperatorArgs, req.Inputs
	d := trDiffSet{}
	d.requiredStr("friendlyName", o.FriendlyName, n.FriendlyName)
	d.replace("operatorType", o.OperatorType, n.OperatorType)
	if !jsonSame(o.Config, n.Config) {
		d["config"] = p.PropertyDiff{Kind: p.Update}
	}
	return d.response(), nil
}

func (a IntelligenceCustomOperatorArgs) form(create bool) url.Values {
	form := url.Values{}
	form.Set("FriendlyName", a.FriendlyName)
	form.Set("Config", a.Config)
	if create {
		form.Set("OperatorType", a.OperatorType)
	}
	return form
}

func (r *IntelligenceCustomOperator) Create(ctx context.Context, req infer.CreateRequest[IntelligenceCustomOperatorArgs]) (infer.CreateResponse[IntelligenceCustomOperatorState], error) {
	type resp = infer.CreateResponse[IntelligenceCustomOperatorState]
	a := req.Inputs
	if req.DryRun {
		return resp{Output: IntelligenceCustomOperatorState{IntelligenceCustomOperatorArgs: a}}, nil
	}
	c, err := clientFor(ctx)
	if err != nil {
		return resp{}, err
	}
	o, err := c.do(ctx, "POST", intelligenceProduct, intelligencePath("Operators/Custom", ""), nil, a.form(true))
	if err != nil {
		return resp{}, fmt.Errorf("create Intelligence custom operator: %w", err)
	}
	st := IntelligenceCustomOperatorState{IntelligenceCustomOperatorArgs: a, Description: nonEmpty(o.optStr("description")), Availability: nonEmpty(o.optStr("availability"))}
	st.Sid, st.DateCreated, st.DateUpdated = trSid(o)
	return resp{ID: st.Sid, Output: st}, nil
}

func (r *IntelligenceCustomOperator) Read(ctx context.Context, req infer.ReadRequest[IntelligenceCustomOperatorArgs, IntelligenceCustomOperatorState]) (infer.ReadResponse[IntelligenceCustomOperatorArgs, IntelligenceCustomOperatorState], error) {
	type resp = infer.ReadResponse[IntelligenceCustomOperatorArgs, IntelligenceCustomOperatorState]
	c, err := clientFor(ctx)
	if err != nil {
		return resp{}, err
	}
	o, err := c.do(ctx, "GET", intelligenceProduct, intelligencePath("Operators/Custom", req.ID), nil, nil)
	if err != nil {
		if isNotFound(err) {
			if err := notFoundOnImport("Intelligence custom operator", req.ID, req.Inputs, req.State); err != nil {
				return resp{}, err
			}
			return resp{}, nil
		}
		return resp{}, fmt.Errorf("read Intelligence custom operator %s: %w", req.ID, err)
	}
	in := req.Inputs
	imported := req.State.Sid == ""
	in.FriendlyName = o.str("friendly_name")
	// Twilio may report the operator type by another name than the program used, so it is read
	// only on import; a later change of the program's value still replaces the operator.
	if imported {
		in.OperatorType = o.str("operator_type")
	}
	if cfg := objJSON(o, "config"); cfg != nil {
		if imported || !jsonSame(in.Config, *cfg) {
			in.Config = *cfg
		}
	}
	st := IntelligenceCustomOperatorState{IntelligenceCustomOperatorArgs: in, Description: nonEmpty(o.optStr("description")), Availability: nonEmpty(o.optStr("availability"))}
	st.Sid, st.DateCreated, st.DateUpdated = trSid(o)
	return resp{ID: req.ID, Inputs: in, State: st}, nil
}

func (r *IntelligenceCustomOperator) Update(ctx context.Context, req infer.UpdateRequest[IntelligenceCustomOperatorArgs, IntelligenceCustomOperatorState]) (infer.UpdateResponse[IntelligenceCustomOperatorState], error) {
	type resp = infer.UpdateResponse[IntelligenceCustomOperatorState]
	st := req.State
	st.IntelligenceCustomOperatorArgs = req.Inputs
	if req.DryRun {
		return resp{Output: st}, nil
	}
	c, err := clientFor(ctx)
	if err != nil {
		return resp{}, err
	}
	o, err := c.do(ctx, "POST", intelligenceProduct, intelligencePath("Operators/Custom", req.ID), nil, req.Inputs.form(false))
	if err != nil {
		return resp{}, fmt.Errorf("update Intelligence custom operator %s: %w", req.ID, err)
	}
	st.DateUpdated = o.str("date_updated")
	st.Description = nonEmpty(o.optStr("description"))
	st.Availability = nonEmpty(o.optStr("availability"))
	return resp{Output: st}, nil
}

func (r *IntelligenceCustomOperator) Delete(ctx context.Context, req infer.DeleteRequest[IntelligenceCustomOperatorState]) (infer.DeleteResponse, error) {
	c, err := clientFor(ctx)
	if err != nil {
		return infer.DeleteResponse{}, err
	}
	if _, err := c.do(ctx, "DELETE", intelligenceProduct, intelligencePath("Operators/Custom", req.ID), nil, nil); err != nil && !isNotFound(err) {
		return infer.DeleteResponse{}, fmt.Errorf("delete Intelligence custom operator %s: %w", req.ID, err)
	}
	return infer.DeleteResponse{}, nil
}
