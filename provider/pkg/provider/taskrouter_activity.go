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

// TaskRouterActivity manages an activity of a TaskRouter workspace: a state a worker can be in.
type TaskRouterActivity struct{}

// TaskRouterActivityArgs are the inputs of a TaskRouterActivity.
type TaskRouterActivityArgs struct {
	WorkspaceSid string `pulumi:"workspaceSid" provider:"replaceOnChanges"`
	FriendlyName string `pulumi:"friendlyName"`
	Available    *bool  `pulumi:"available,optional" provider:"replaceOnChanges"`
}

func (a *TaskRouterActivityArgs) Annotate(an infer.Annotator) {
	an.Describe(&a.WorkspaceSid, "SID (WS...) of the workspace. Changing it replaces the resource.")
	an.Describe(&a.FriendlyName, "Human-readable name of the activity, unique in the workspace.")
	an.Describe(&a.Available, "Whether workers in this activity can receive tasks (Twilio default true). It cannot be changed after creation, so changing it replaces the resource.")
}

// TaskRouterActivityState is the persisted state of a TaskRouterActivity.
type TaskRouterActivityState struct {
	TaskRouterActivityArgs

	Sid         string `pulumi:"sid"`
	DateCreated string `pulumi:"dateCreated"`
	DateUpdated string `pulumi:"dateUpdated"`
}

func (s *TaskRouterActivityState) Annotate(a infer.Annotator) {
	a.Describe(&s.Sid, "The activity SID (WA...).")
	a.Describe(&s.DateCreated, "When the activity was created (ISO 8601).")
	a.Describe(&s.DateUpdated, "When the activity was last updated (ISO 8601).")
}

func (r *TaskRouterActivity) Annotate(a infer.Annotator) {
	a.Describe(&r, "An activity of a TaskRouter workspace. The resource ID is <workspaceSid>/<sid>.")
}

var _ infer.CustomResource[TaskRouterActivityArgs, TaskRouterActivityState] = (*TaskRouterActivity)(nil)
var _ infer.CustomCheck[TaskRouterActivityArgs] = (*TaskRouterActivity)(nil)
var _ infer.CustomDiff[TaskRouterActivityArgs, TaskRouterActivityState] = (*TaskRouterActivity)(nil)
var _ infer.CustomRead[TaskRouterActivityArgs, TaskRouterActivityState] = (*TaskRouterActivity)(nil)
var _ infer.CustomUpdate[TaskRouterActivityArgs, TaskRouterActivityState] = (*TaskRouterActivity)(nil)
var _ infer.CustomDelete[TaskRouterActivityState] = (*TaskRouterActivity)(nil)

func (r *TaskRouterActivity) Check(ctx context.Context, req infer.CheckRequest) (infer.CheckResponse[TaskRouterActivityArgs], error) {
	args, failures, err := infer.DefaultCheck[TaskRouterActivityArgs](ctx, req.NewInputs)
	if err != nil || len(failures) > 0 {
		return infer.CheckResponse[TaskRouterActivityArgs]{Inputs: args, Failures: failures}, err
	}
	c := &checker{unk: unknowns(req.NewInputs)}
	trCheckWorkspace(c, args.WorkspaceSid)
	if !c.unk("friendlyName") && args.FriendlyName == "" {
		c.fail("friendlyName", "friendlyName must not be empty")
	}
	return infer.CheckResponse[TaskRouterActivityArgs]{Inputs: args, Failures: c.failures}, nil
}

func (r *TaskRouterActivity) Diff(_ context.Context, req infer.DiffRequest[TaskRouterActivityArgs, TaskRouterActivityState]) (infer.DiffResponse, error) {
	o, n := req.State.TaskRouterActivityArgs, req.Inputs
	d := trDiffSet{}
	d.replace("workspaceSid", o.WorkspaceSid, n.WorkspaceSid)
	d.requiredStr("friendlyName", o.FriendlyName, n.FriendlyName)
	if n.Available != nil && (o.Available == nil || *o.Available != *n.Available) {
		d["available"] = p.PropertyDiff{Kind: p.UpdateReplace}
	}
	return d.response(), nil
}

func (r *TaskRouterActivity) Create(ctx context.Context, req infer.CreateRequest[TaskRouterActivityArgs]) (infer.CreateResponse[TaskRouterActivityState], error) {
	type resp = infer.CreateResponse[TaskRouterActivityState]
	a := req.Inputs
	if req.DryRun {
		return resp{Output: TaskRouterActivityState{TaskRouterActivityArgs: a}}, nil
	}
	c, err := clientFor(ctx)
	if err != nil {
		return resp{}, err
	}
	form := url.Values{"FriendlyName": {a.FriendlyName}}
	setB(form, "Available", a.Available)
	o, err := c.do(ctx, "POST", trProduct, trPath(a.WorkspaceSid, "Activities", ""), nil, form)
	if err != nil {
		return resp{}, fmt.Errorf("create activity in TaskRouter workspace %s: %w", a.WorkspaceSid, err)
	}
	st := TaskRouterActivityState{TaskRouterActivityArgs: a}
	st.Sid, st.DateCreated, st.DateUpdated = trSid(o)
	return resp{ID: a.WorkspaceSid + "/" + st.Sid, Output: st}, nil
}

func (r *TaskRouterActivity) Read(ctx context.Context, req infer.ReadRequest[TaskRouterActivityArgs, TaskRouterActivityState]) (infer.ReadResponse[TaskRouterActivityArgs, TaskRouterActivityState], error) {
	type resp = infer.ReadResponse[TaskRouterActivityArgs, TaskRouterActivityState]
	ws, sid, err := trSplitID(req.ID)
	if err != nil {
		return resp{}, err
	}
	c, err := clientFor(ctx)
	if err != nil {
		return resp{}, err
	}
	o, err := c.do(ctx, "GET", trProduct, trPath(ws, "Activities", sid), nil, nil)
	if err != nil {
		if isNotFound(err) {
			if err := notFoundOnImport("TaskRouter activity", req.ID, req.Inputs, req.State); err != nil {
				return resp{}, err
			}
			return resp{}, nil
		}
		return resp{}, fmt.Errorf("read TaskRouter activity %s: %w", req.ID, err)
	}
	in := req.Inputs
	imported := in.WorkspaceSid == ""
	in.WorkspaceSid = ws
	in.FriendlyName = o.str("friendly_name")
	in.Available = trBool(in.Available, o.optBool("available"), true, imported)
	st := TaskRouterActivityState{TaskRouterActivityArgs: in}
	st.Sid, st.DateCreated, st.DateUpdated = trSid(o)
	return resp{ID: req.ID, Inputs: in, State: st}, nil
}

func (r *TaskRouterActivity) Update(ctx context.Context, req infer.UpdateRequest[TaskRouterActivityArgs, TaskRouterActivityState]) (infer.UpdateResponse[TaskRouterActivityState], error) {
	type resp = infer.UpdateResponse[TaskRouterActivityState]
	st := req.State
	st.TaskRouterActivityArgs = req.Inputs
	if req.DryRun {
		return resp{Output: st}, nil
	}
	ws, sid, err := trSplitID(req.ID)
	if err != nil {
		return resp{}, err
	}
	c, err := clientFor(ctx)
	if err != nil {
		return resp{}, err
	}
	o, err := c.do(ctx, "POST", trProduct, trPath(ws, "Activities", sid), nil, url.Values{"FriendlyName": {req.Inputs.FriendlyName}})
	if err != nil {
		return resp{}, fmt.Errorf("update TaskRouter activity %s: %w", req.ID, err)
	}
	st.DateUpdated = o.str("date_updated")
	return resp{Output: st}, nil
}

func (r *TaskRouterActivity) Delete(ctx context.Context, req infer.DeleteRequest[TaskRouterActivityState]) (infer.DeleteResponse, error) {
	return infer.DeleteResponse{}, trDelete(ctx, "activity", "Activities", req.ID)
}
