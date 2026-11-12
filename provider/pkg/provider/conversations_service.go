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

	"github.com/pulumi/pulumi-go-provider/infer"
)

// ConversationsService manages a Conversations service: the container of conversations, users,
// roles and the service configuration.
type ConversationsService struct{}

// ConversationsServiceArgs are the inputs of a ConversationsService.
type ConversationsServiceArgs struct {
	FriendlyName string `pulumi:"friendlyName" provider:"replaceOnChanges"`
}

func (a *ConversationsServiceArgs) Annotate(an infer.Annotator) {
	an.Describe(&a.FriendlyName, "Human-readable name of the service, up to 256 characters. Twilio offers no way to rename a service, so changing it replaces the service and everything in it.")
}

// ConversationsServiceState is the persisted state of a ConversationsService.
type ConversationsServiceState struct {
	ConversationsServiceArgs

	Sid         string `pulumi:"sid"`
	DateCreated string `pulumi:"dateCreated"`
	DateUpdated string `pulumi:"dateUpdated"`
}

func (s *ConversationsServiceState) Annotate(a infer.Annotator) {
	a.Describe(&s.Sid, "The service SID (IS...).")
	a.Describe(&s.DateCreated, "When the service was created (ISO 8601).")
	a.Describe(&s.DateUpdated, "When the service was last updated (ISO 8601).")
}

func (r *ConversationsService) Annotate(a infer.Annotator) {
	a.Describe(&r, "A Conversations service. The resource ID is the service SID.")
}

var _ infer.CustomResource[ConversationsServiceArgs, ConversationsServiceState] = (*ConversationsService)(nil)
var _ infer.CustomCheck[ConversationsServiceArgs] = (*ConversationsService)(nil)
var _ infer.CustomDiff[ConversationsServiceArgs, ConversationsServiceState] = (*ConversationsService)(nil)
var _ infer.CustomRead[ConversationsServiceArgs, ConversationsServiceState] = (*ConversationsService)(nil)
var _ infer.CustomDelete[ConversationsServiceState] = (*ConversationsService)(nil)

func (r *ConversationsService) Check(ctx context.Context, req infer.CheckRequest) (infer.CheckResponse[ConversationsServiceArgs], error) {
	args, failures, err := infer.DefaultCheck[ConversationsServiceArgs](ctx, req.NewInputs)
	if err != nil || len(failures) > 0 {
		return infer.CheckResponse[ConversationsServiceArgs]{Inputs: args, Failures: failures}, err
	}
	c := &checker{unk: unknowns(req.NewInputs)}
	if !c.unk("friendlyName") && (args.FriendlyName == "" || len(args.FriendlyName) > 256) {
		c.fail("friendlyName", "friendlyName must be 1-256 characters")
	}
	return infer.CheckResponse[ConversationsServiceArgs]{Inputs: args, Failures: c.failures}, nil
}

func (r *ConversationsService) Diff(_ context.Context, req infer.DiffRequest[ConversationsServiceArgs, ConversationsServiceState]) (infer.DiffResponse, error) {
	d := trDiffSet{}
	d.replace("friendlyName", req.State.FriendlyName, req.Inputs.FriendlyName)
	return d.response(), nil
}

func (r *ConversationsService) Create(ctx context.Context, req infer.CreateRequest[ConversationsServiceArgs]) (infer.CreateResponse[ConversationsServiceState], error) {
	type resp = infer.CreateResponse[ConversationsServiceState]
	a := req.Inputs
	if req.DryRun {
		return resp{Output: ConversationsServiceState{ConversationsServiceArgs: a}}, nil
	}
	c, err := clientFor(ctx)
	if err != nil {
		return resp{}, err
	}
	o, err := c.do(ctx, "POST", cvProduct, cvPath("", "Services", ""), nil, url.Values{"FriendlyName": {a.FriendlyName}})
	if err != nil {
		return resp{}, fmt.Errorf("create Conversations service: %w", err)
	}
	st := ConversationsServiceState{ConversationsServiceArgs: a}
	st.Sid, st.DateCreated, st.DateUpdated = trSid(o)
	return resp{ID: st.Sid, Output: st}, nil
}

func (r *ConversationsService) Read(ctx context.Context, req infer.ReadRequest[ConversationsServiceArgs, ConversationsServiceState]) (infer.ReadResponse[ConversationsServiceArgs, ConversationsServiceState], error) {
	type resp = infer.ReadResponse[ConversationsServiceArgs, ConversationsServiceState]
	c, err := clientFor(ctx)
	if err != nil {
		return resp{}, err
	}
	o, err := c.do(ctx, "GET", cvProduct, cvPath("", "Services", req.ID), nil, nil)
	if err != nil {
		if isNotFound(err) {
			if err := notFoundOnImport("Conversations service", req.ID, req.Inputs, req.State); err != nil {
				return resp{}, err
			}
			return resp{}, nil
		}
		return resp{}, fmt.Errorf("read Conversations service %s: %w", req.ID, err)
	}
	in := req.Inputs
	in.FriendlyName = o.str("friendly_name")
	st := ConversationsServiceState{ConversationsServiceArgs: in}
	st.Sid, st.DateCreated, st.DateUpdated = trSid(o)
	return resp{ID: req.ID, Inputs: in, State: st}, nil
}

func (r *ConversationsService) Delete(ctx context.Context, req infer.DeleteRequest[ConversationsServiceState]) (infer.DeleteResponse, error) {
	c, err := clientFor(ctx)
	if err != nil {
		return infer.DeleteResponse{}, err
	}
	if _, err := c.do(ctx, "DELETE", cvProduct, cvPath("", "Services", req.ID), nil, nil); err != nil && !isNotFound(err) {
		return infer.DeleteResponse{}, fmt.Errorf("delete Conversations service %s: %w", req.ID, err)
	}
	return infer.DeleteResponse{}, nil
}
