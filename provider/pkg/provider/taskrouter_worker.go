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
	"fmt"
	"net/url"

	"github.com/pulumi/pulumi-go-provider/infer"
)

// TaskRouterWorker manages a worker of a TaskRouter workspace: an entity, usually a person, that tasks are routed to.
type TaskRouterWorker struct{}

// TaskRouterWorkerArgs are the inputs of a TaskRouterWorker.
type TaskRouterWorkerArgs struct {
	WorkspaceSid string  `pulumi:"workspaceSid" provider:"replaceOnChanges"`
	FriendlyName string  `pulumi:"friendlyName"`
	Attributes   *string `pulumi:"attributes,optional"`
	ActivitySid  *string `pulumi:"activitySid,optional"`
}

func (a *TaskRouterWorkerArgs) Annotate(an infer.Annotator) {
	an.Describe(&a.WorkspaceSid, "SID (WS...) of the workspace. Changing it replaces the resource.")
	an.Describe(&a.FriendlyName, "Human-readable name of the worker.")
	an.Describe(&a.Attributes, "The worker's attributes as a JSON object, matched by task queue and workflow expressions (Twilio default {}). Formatting and key order do not matter. Removing it leaves the attributes unchanged.")
	an.Describe(&a.ActivitySid, "SID (WA...) of the activity the worker starts in; Twilio uses the workspace default when it is unset. It is applied on create and when it is changed in the program. It is not refreshed, because a worker changes activity while it works, so a worker that moved on its own is not drift. Removing it leaves the activity unchanged.")
}

// TaskRouterWorkerState is the persisted state of a TaskRouterWorker.
type TaskRouterWorkerState struct {
	TaskRouterWorkerArgs

	Sid         string `pulumi:"sid"`
	DateCreated string `pulumi:"dateCreated"`
	DateUpdated string `pulumi:"dateUpdated"`
}

func (s *TaskRouterWorkerState) Annotate(a infer.Annotator) {
	a.Describe(&s.Sid, "The worker SID (WK...).")
	a.Describe(&s.DateCreated, "When the worker was created (ISO 8601).")
	a.Describe(&s.DateUpdated, "When the worker was last updated (ISO 8601).")
}

func (r *TaskRouterWorker) Annotate(a infer.Annotator) {
	a.Describe(&r, "A worker of a TaskRouter workspace. The resource ID is <workspaceSid>/<sid>.")
}

var _ infer.CustomResource[TaskRouterWorkerArgs, TaskRouterWorkerState] = (*TaskRouterWorker)(nil)
var _ infer.CustomCheck[TaskRouterWorkerArgs] = (*TaskRouterWorker)(nil)
var _ infer.CustomDiff[TaskRouterWorkerArgs, TaskRouterWorkerState] = (*TaskRouterWorker)(nil)
var _ infer.CustomRead[TaskRouterWorkerArgs, TaskRouterWorkerState] = (*TaskRouterWorker)(nil)
var _ infer.CustomUpdate[TaskRouterWorkerArgs, TaskRouterWorkerState] = (*TaskRouterWorker)(nil)
var _ infer.CustomDelete[TaskRouterWorkerState] = (*TaskRouterWorker)(nil)

func (r *TaskRouterWorker) Check(ctx context.Context, req infer.CheckRequest) (infer.CheckResponse[TaskRouterWorkerArgs], error) {
	args, failures, err := infer.DefaultCheck[TaskRouterWorkerArgs](ctx, req.NewInputs)
	if err != nil || len(failures) > 0 {
		return infer.CheckResponse[TaskRouterWorkerArgs]{Inputs: args, Failures: failures}, err
	}
	c := &checker{unk: unknowns(req.NewInputs)}
	trCheckWorkspace(c, args.WorkspaceSid)
	if !c.unk("friendlyName") && args.FriendlyName == "" {
		c.fail("friendlyName", "friendlyName must not be empty")
	}
	if v := args.Attributes; v != nil && !c.unk("attributes") && !json.Valid([]byte(*v)) {
		c.fail("attributes", "attributes must be a valid JSON object")
	}
	trCheckOptionalActivity(c, "activitySid", args.ActivitySid)
	return infer.CheckResponse[TaskRouterWorkerArgs]{Inputs: args, Failures: c.failures}, nil
}

func (r *TaskRouterWorker) Diff(_ context.Context, req infer.DiffRequest[TaskRouterWorkerArgs, TaskRouterWorkerState]) (infer.DiffResponse, error) {
	o, n := req.State.TaskRouterWorkerArgs, req.Inputs
	d := trDiffSet{}
	d.replace("workspaceSid", o.WorkspaceSid, n.WorkspaceSid)
	d.requiredStr("friendlyName", o.FriendlyName, n.FriendlyName)
	o.Attributes = trAttributesOrDefault(o.Attributes, n.Attributes)
	d.jsonDoc("attributes", o.Attributes, n.Attributes)
	d.str("activitySid", o.ActivitySid, n.ActivitySid)
	return d.response(), nil
}

func (r *TaskRouterWorker) Create(ctx context.Context, req infer.CreateRequest[TaskRouterWorkerArgs]) (infer.CreateResponse[TaskRouterWorkerState], error) {
	type resp = infer.CreateResponse[TaskRouterWorkerState]
	a := req.Inputs
	if req.DryRun {
		return resp{Output: TaskRouterWorkerState{TaskRouterWorkerArgs: a}}, nil
	}
	c, err := clientFor(ctx)
	if err != nil {
		return resp{}, err
	}
	form := url.Values{"FriendlyName": {a.FriendlyName}}
	setS(form, "Attributes", a.Attributes)
	setS(form, "ActivitySid", a.ActivitySid)
	o, err := c.do(ctx, "POST", trProduct, trPath(a.WorkspaceSid, "Workers", ""), nil, form)
	if err != nil {
		return resp{}, fmt.Errorf("create worker in TaskRouter workspace %s: %w", a.WorkspaceSid, err)
	}
	st := TaskRouterWorkerState{TaskRouterWorkerArgs: a}
	st.Sid, st.DateCreated, st.DateUpdated = trSid(o)
	return resp{ID: a.WorkspaceSid + "/" + st.Sid, Output: st}, nil
}

func (r *TaskRouterWorker) Read(ctx context.Context, req infer.ReadRequest[TaskRouterWorkerArgs, TaskRouterWorkerState]) (infer.ReadResponse[TaskRouterWorkerArgs, TaskRouterWorkerState], error) {
	type resp = infer.ReadResponse[TaskRouterWorkerArgs, TaskRouterWorkerState]
	ws, sid, err := trSplitID(req.ID)
	if err != nil {
		return resp{}, err
	}
	c, err := clientFor(ctx)
	if err != nil {
		return resp{}, err
	}
	o, err := c.do(ctx, "GET", trProduct, trPath(ws, "Workers", sid), nil, nil)
	if err != nil {
		if isNotFound(err) {
			if err := notFoundOnImport("TaskRouter worker", req.ID, req.Inputs, req.State); err != nil {
				return resp{}, err
			}
			return resp{}, nil
		}
		return resp{}, fmt.Errorf("read TaskRouter worker %s: %w", req.ID, err)
	}
	in := req.Inputs
	imported := in.WorkspaceSid == ""
	in.WorkspaceSid = ws
	in.FriendlyName = o.str("friendly_name")
	in.Attributes = trJSON(in.Attributes, o.optStr("attributes"), imported)
	if imported && in.Attributes != nil && jsonSame(*in.Attributes, "{}") {
		in.Attributes = nil
	}
	// The activity is runtime state and is deliberately left as it was.
	st := TaskRouterWorkerState{TaskRouterWorkerArgs: in}
	st.Sid, st.DateCreated, st.DateUpdated = trSid(o)
	return resp{ID: req.ID, Inputs: in, State: st}, nil
}

func (r *TaskRouterWorker) Update(ctx context.Context, req infer.UpdateRequest[TaskRouterWorkerArgs, TaskRouterWorkerState]) (infer.UpdateResponse[TaskRouterWorkerState], error) {
	type resp = infer.UpdateResponse[TaskRouterWorkerState]
	a := req.Inputs
	st := req.State
	st.TaskRouterWorkerArgs = a
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
	form := url.Values{"FriendlyName": {a.FriendlyName}}
	setS(form, "Attributes", a.Attributes)
	if old := req.State.ActivitySid; a.ActivitySid != nil && (old == nil || *old != *a.ActivitySid) {
		setS(form, "ActivitySid", a.ActivitySid)
	}
	o, err := c.do(ctx, "POST", trProduct, trPath(ws, "Workers", sid), nil, form)
	if err != nil {
		return resp{}, fmt.Errorf("update TaskRouter worker %s: %w", req.ID, err)
	}
	st.DateUpdated = o.str("date_updated")
	return resp{Output: st}, nil
}

func (r *TaskRouterWorker) Delete(ctx context.Context, req infer.DeleteRequest[TaskRouterWorkerState]) (infer.DeleteResponse, error) {
	return infer.DeleteResponse{}, trDelete(ctx, "worker", "Workers", req.ID)
}

// trAttributesOrDefault returns the stored attributes, or Twilio's default {} when an import left
// them unset (#1880) and the program sets some.
func trAttributesOrDefault(o, n *string) *string {
	if o == nil && n != nil {
		def := "{}"
		return &def
	}
	return o
}
