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

	p "github.com/pulumi/pulumi-go-provider"
	"github.com/pulumi/pulumi-go-provider/infer"
)

// TaskRouterWorkflow manages a workflow of a TaskRouter workspace: the routing rules that send tasks to queues.
type TaskRouterWorkflow struct{}

// TaskRouterWorkflowArgs are the inputs of a TaskRouterWorkflow.
type TaskRouterWorkflowArgs struct {
	WorkspaceSid                  string  `pulumi:"workspaceSid" provider:"replaceOnChanges"`
	FriendlyName                  string  `pulumi:"friendlyName"`
	Configuration                 string  `pulumi:"configuration"`
	AssignmentCallbackUrl         *string `pulumi:"assignmentCallbackUrl,optional"`
	FallbackAssignmentCallbackUrl *string `pulumi:"fallbackAssignmentCallbackUrl,optional"`
	TaskReservationTimeout        *int    `pulumi:"taskReservationTimeout,optional"`
}

func (a *TaskRouterWorkflowArgs) Annotate(an infer.Annotator) {
	an.Describe(&a.WorkspaceSid, "SID (WS...) of the workspace. Changing it replaces the resource.")
	an.Describe(&a.FriendlyName, "Human-readable name of the workflow.")
	an.Describe(&a.Configuration, "The routing rules as a JSON document (task_routing with filters and targets, and an optional default_filter). Formatting and key order do not matter: a document that means the same as Twilio's is not a change.")
	an.Describe(&a.AssignmentCallbackUrl, "URL Twilio calls when a worker is reserved for a task of this workflow. Removing it leaves the setting unchanged.")
	an.Describe(&a.FallbackAssignmentCallbackUrl, "URL Twilio calls when assignmentCallbackUrl fails. Removing it leaves the setting unchanged.")
	an.Describe(&a.TaskReservationTimeout, "Seconds a worker has to accept a reservation, 1 to 86400 (Twilio default 120). Removing it leaves the setting unchanged.")
}

// TaskRouterWorkflowState is the persisted state of a TaskRouterWorkflow.
type TaskRouterWorkflowState struct {
	TaskRouterWorkflowArgs

	Sid         string `pulumi:"sid"`
	DateCreated string `pulumi:"dateCreated"`
	DateUpdated string `pulumi:"dateUpdated"`
}

func (s *TaskRouterWorkflowState) Annotate(a infer.Annotator) {
	a.Describe(&s.Sid, "The workflow SID (WW...).")
	a.Describe(&s.DateCreated, "When the workflow was created (ISO 8601).")
	a.Describe(&s.DateUpdated, "When the workflow was last updated (ISO 8601).")
}

func (r *TaskRouterWorkflow) Annotate(a infer.Annotator) {
	a.Describe(&r, "A workflow of a TaskRouter workspace. The resource ID is <workspaceSid>/<sid>.")
}

var _ infer.CustomResource[TaskRouterWorkflowArgs, TaskRouterWorkflowState] = (*TaskRouterWorkflow)(nil)
var _ infer.CustomCheck[TaskRouterWorkflowArgs] = (*TaskRouterWorkflow)(nil)
var _ infer.CustomDiff[TaskRouterWorkflowArgs, TaskRouterWorkflowState] = (*TaskRouterWorkflow)(nil)
var _ infer.CustomRead[TaskRouterWorkflowArgs, TaskRouterWorkflowState] = (*TaskRouterWorkflow)(nil)
var _ infer.CustomUpdate[TaskRouterWorkflowArgs, TaskRouterWorkflowState] = (*TaskRouterWorkflow)(nil)
var _ infer.CustomDelete[TaskRouterWorkflowState] = (*TaskRouterWorkflow)(nil)

func (r *TaskRouterWorkflow) Check(ctx context.Context, req infer.CheckRequest) (infer.CheckResponse[TaskRouterWorkflowArgs], error) {
	args, failures, err := infer.DefaultCheck[TaskRouterWorkflowArgs](ctx, req.NewInputs)
	if err != nil || len(failures) > 0 {
		return infer.CheckResponse[TaskRouterWorkflowArgs]{Inputs: args, Failures: failures}, err
	}
	c := &checker{unk: unknowns(req.NewInputs)}
	trCheckWorkspace(c, args.WorkspaceSid)
	if !c.unk("friendlyName") && args.FriendlyName == "" {
		c.fail("friendlyName", "friendlyName must not be empty")
	}
	if !c.unk("configuration") && !json.Valid([]byte(args.Configuration)) {
		c.fail("configuration", "configuration must be a valid JSON document")
	}
	if v := args.TaskReservationTimeout; v != nil && !c.unk("taskReservationTimeout") && (*v < 1 || *v > 86400) {
		c.fail("taskReservationTimeout", "taskReservationTimeout must be between 1 and 86400 seconds")
	}
	return infer.CheckResponse[TaskRouterWorkflowArgs]{Inputs: args, Failures: c.failures}, nil
}

func (r *TaskRouterWorkflow) Diff(_ context.Context, req infer.DiffRequest[TaskRouterWorkflowArgs, TaskRouterWorkflowState]) (infer.DiffResponse, error) {
	o, n := req.State.TaskRouterWorkflowArgs, req.Inputs
	d := trDiffSet{}
	d.replace("workspaceSid", o.WorkspaceSid, n.WorkspaceSid)
	d.requiredStr("friendlyName", o.FriendlyName, n.FriendlyName)
	if !jsonSame(o.Configuration, n.Configuration) {
		d["configuration"] = p.PropertyDiff{Kind: p.Update}
	}
	d.strDef(taskrouterWorkflowDefaults, "assignmentCallbackUrl", o.AssignmentCallbackUrl, n.AssignmentCallbackUrl)
	d.strDef(taskrouterWorkflowDefaults, "fallbackAssignmentCallbackUrl", o.FallbackAssignmentCallbackUrl, n.FallbackAssignmentCallbackUrl)
	d.intDef(taskrouterWorkflowDefaults, "taskReservationTimeout", o.TaskReservationTimeout, n.TaskReservationTimeout)
	return d.response(), nil
}

func (a TaskRouterWorkflowArgs) form() url.Values {
	form := url.Values{"FriendlyName": {a.FriendlyName}, "Configuration": {a.Configuration}}
	setS(form, "AssignmentCallbackUrl", a.AssignmentCallbackUrl)
	setS(form, "FallbackAssignmentCallbackUrl", a.FallbackAssignmentCallbackUrl)
	setI(form, "TaskReservationTimeout", a.TaskReservationTimeout)
	return form
}

func (r *TaskRouterWorkflow) Create(ctx context.Context, req infer.CreateRequest[TaskRouterWorkflowArgs]) (infer.CreateResponse[TaskRouterWorkflowState], error) {
	type resp = infer.CreateResponse[TaskRouterWorkflowState]
	a := req.Inputs
	if req.DryRun {
		return resp{Output: TaskRouterWorkflowState{TaskRouterWorkflowArgs: a}}, nil
	}
	c, err := clientFor(ctx)
	if err != nil {
		return resp{}, err
	}
	o, err := c.do(ctx, "POST", trProduct, trPath(a.WorkspaceSid, "Workflows", ""), nil, a.form())
	if err != nil {
		return resp{}, fmt.Errorf("create workflow in TaskRouter workspace %s: %w", a.WorkspaceSid, err)
	}
	st := TaskRouterWorkflowState{TaskRouterWorkflowArgs: a}
	st.Sid, st.DateCreated, st.DateUpdated = trSid(o)
	return resp{ID: a.WorkspaceSid + "/" + st.Sid, Output: st}, nil
}

func (r *TaskRouterWorkflow) Read(ctx context.Context, req infer.ReadRequest[TaskRouterWorkflowArgs, TaskRouterWorkflowState]) (infer.ReadResponse[TaskRouterWorkflowArgs, TaskRouterWorkflowState], error) {
	type resp = infer.ReadResponse[TaskRouterWorkflowArgs, TaskRouterWorkflowState]
	ws, sid, err := trSplitID(req.ID)
	if err != nil {
		return resp{}, err
	}
	c, err := clientFor(ctx)
	if err != nil {
		return resp{}, err
	}
	o, err := c.do(ctx, "GET", trProduct, trPath(ws, "Workflows", sid), nil, nil)
	if err != nil {
		if isNotFound(err) {
			if err := notFoundOnImport("TaskRouter workflow", req.ID, req.Inputs, req.State); err != nil {
				return resp{}, err
			}
			return resp{}, nil
		}
		return resp{}, fmt.Errorf("read TaskRouter workflow %s: %w", req.ID, err)
	}
	in := req.Inputs
	imported := in.WorkspaceSid == ""
	in.WorkspaceSid = ws
	in.FriendlyName = o.str("friendly_name")
	if cfg := trJSON(&in.Configuration, o.optStr("configuration"), false); cfg != nil {
		in.Configuration = *cfg
	}
	in.AssignmentCallbackUrl = trStr(in.AssignmentCallbackUrl, o.optStr("assignment_callback_url"), taskrouterWorkflowDefaults.str("assignmentCallbackUrl"), imported)
	in.FallbackAssignmentCallbackUrl = trStr(in.FallbackAssignmentCallbackUrl, o.optStr("fallback_assignment_callback_url"), taskrouterWorkflowDefaults.str("fallbackAssignmentCallbackUrl"), imported)
	in.TaskReservationTimeout = trInt(in.TaskReservationTimeout, o.optInt("task_reservation_timeout"), taskrouterWorkflowDefaults.integer("taskReservationTimeout"), imported)
	st := TaskRouterWorkflowState{TaskRouterWorkflowArgs: in}
	st.Sid, st.DateCreated, st.DateUpdated = trSid(o)
	return resp{ID: req.ID, Inputs: in, State: st}, nil
}

func (r *TaskRouterWorkflow) Update(ctx context.Context, req infer.UpdateRequest[TaskRouterWorkflowArgs, TaskRouterWorkflowState]) (infer.UpdateResponse[TaskRouterWorkflowState], error) {
	type resp = infer.UpdateResponse[TaskRouterWorkflowState]
	a := req.Inputs
	st := req.State
	st.TaskRouterWorkflowArgs = a
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
	o, err := c.do(ctx, "POST", trProduct, trPath(ws, "Workflows", sid), nil, a.form())
	if err != nil {
		return resp{}, fmt.Errorf("update TaskRouter workflow %s: %w", req.ID, err)
	}
	st.DateUpdated = o.str("date_updated")
	return resp{Output: st}, nil
}

func (r *TaskRouterWorkflow) Delete(ctx context.Context, req infer.DeleteRequest[TaskRouterWorkflowState]) (infer.DeleteResponse, error) {
	return infer.DeleteResponse{}, trDelete(ctx, "workflow", "Workflows", req.ID)
}

// taskrouterWorkflowDefaults are the Twilio defaults of this resource, shared by Read (import adoption) and Diff.
var taskrouterWorkflowDefaults = trDefaults{
	"assignmentCallbackUrl":         "",
	"fallbackAssignmentCallbackUrl": "",
	"taskReservationTimeout":        120,
}
