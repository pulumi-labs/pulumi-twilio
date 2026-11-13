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

// TaskRouterTaskQueue manages a task queue of a TaskRouter workspace: a set of tasks waiting for workers that match its expression.
type TaskRouterTaskQueue struct{}

// TaskRouterTaskQueueArgs are the inputs of a TaskRouterTaskQueue.
type TaskRouterTaskQueueArgs struct {
	WorkspaceSid           string  `pulumi:"workspaceSid" provider:"replaceOnChanges"`
	FriendlyName           string  `pulumi:"friendlyName"`
	TargetWorkers          *string `pulumi:"targetWorkers,optional"`
	MaxReservedWorkers     *int    `pulumi:"maxReservedWorkers,optional"`
	TaskOrder              *string `pulumi:"taskOrder,optional"`
	ReservationActivitySid *string `pulumi:"reservationActivitySid,optional"`
	AssignmentActivitySid  *string `pulumi:"assignmentActivitySid,optional"`
}

func (a *TaskRouterTaskQueueArgs) Annotate(an infer.Annotator) {
	an.Describe(&a.WorkspaceSid, "SID (WS...) of the workspace. Changing it replaces the resource.")
	an.Describe(&a.FriendlyName, "Human-readable name of the queue.")
	an.Describe(&a.TargetWorkers, "Worker selection expression, for example skills HAS \"support\" (Twilio default 1==1, every worker). Removing it leaves the setting unchanged.")
	an.Describe(&a.MaxReservedWorkers, "Maximum number of workers reserved for a task at once, 1 to 50 (Twilio default 1). Removing it leaves the setting unchanged.")
	an.Describe(&a.TaskOrder, "Order tasks are assigned from the queue: FIFO or LIFO (Twilio default FIFO). Removing it leaves the setting unchanged.")
	an.Describe(&a.ReservationActivitySid, "SID (WA...) of the activity a worker moves to when a task is reserved for it. Removing it leaves the setting unchanged.")
	an.Describe(&a.AssignmentActivitySid, "SID (WA...) of the activity a worker moves to when it accepts a task. Removing it leaves the setting unchanged.")
}

// TaskRouterTaskQueueState is the persisted state of a TaskRouterTaskQueue.
type TaskRouterTaskQueueState struct {
	TaskRouterTaskQueueArgs

	Sid         string `pulumi:"sid"`
	DateCreated string `pulumi:"dateCreated"`
	DateUpdated string `pulumi:"dateUpdated"`
}

func (s *TaskRouterTaskQueueState) Annotate(a infer.Annotator) {
	a.Describe(&s.Sid, "The task queue SID (WQ...).")
	a.Describe(&s.DateCreated, "When the queue was created (ISO 8601).")
	a.Describe(&s.DateUpdated, "When the queue was last updated (ISO 8601).")
}

func (r *TaskRouterTaskQueue) Annotate(a infer.Annotator) {
	a.Describe(&r, "A task queue of a TaskRouter workspace. The resource ID is <workspaceSid>/<sid>.")
}

var _ infer.CustomResource[TaskRouterTaskQueueArgs, TaskRouterTaskQueueState] = (*TaskRouterTaskQueue)(nil)
var _ infer.CustomCheck[TaskRouterTaskQueueArgs] = (*TaskRouterTaskQueue)(nil)
var _ infer.CustomDiff[TaskRouterTaskQueueArgs, TaskRouterTaskQueueState] = (*TaskRouterTaskQueue)(nil)
var _ infer.CustomRead[TaskRouterTaskQueueArgs, TaskRouterTaskQueueState] = (*TaskRouterTaskQueue)(nil)
var _ infer.CustomUpdate[TaskRouterTaskQueueArgs, TaskRouterTaskQueueState] = (*TaskRouterTaskQueue)(nil)
var _ infer.CustomDelete[TaskRouterTaskQueueState] = (*TaskRouterTaskQueue)(nil)

func (r *TaskRouterTaskQueue) Check(ctx context.Context, req infer.CheckRequest) (infer.CheckResponse[TaskRouterTaskQueueArgs], error) {
	args, failures, err := infer.DefaultCheck[TaskRouterTaskQueueArgs](ctx, req.NewInputs)
	if err != nil || len(failures) > 0 {
		return infer.CheckResponse[TaskRouterTaskQueueArgs]{Inputs: args, Failures: failures}, err
	}
	c := &checker{unk: unknowns(req.NewInputs)}
	trCheckWorkspace(c, args.WorkspaceSid)
	if !c.unk("friendlyName") && args.FriendlyName == "" {
		c.fail("friendlyName", "friendlyName must not be empty")
	}
	if v := args.TaskOrder; v != nil && !c.unk("taskOrder") && *v != "FIFO" && *v != "LIFO" {
		c.fail("taskOrder", "taskOrder must be FIFO or LIFO")
	}
	if v := args.MaxReservedWorkers; v != nil && !c.unk("maxReservedWorkers") && (*v < 1 || *v > 50) {
		c.fail("maxReservedWorkers", "maxReservedWorkers must be between 1 and 50")
	}
	trCheckOptionalActivity(c, "reservationActivitySid", args.ReservationActivitySid)
	trCheckOptionalActivity(c, "assignmentActivitySid", args.AssignmentActivitySid)
	return infer.CheckResponse[TaskRouterTaskQueueArgs]{Inputs: args, Failures: c.failures}, nil
}

func (r *TaskRouterTaskQueue) Diff(_ context.Context, req infer.DiffRequest[TaskRouterTaskQueueArgs, TaskRouterTaskQueueState]) (infer.DiffResponse, error) {
	o, n := req.State.TaskRouterTaskQueueArgs, req.Inputs
	d := trDiffSet{}
	d.replace("workspaceSid", o.WorkspaceSid, n.WorkspaceSid)
	d.requiredStr("friendlyName", o.FriendlyName, n.FriendlyName)
	d.strDef(taskrouterTaskQueueDefaults, "targetWorkers", o.TargetWorkers, n.TargetWorkers)
	d.intDef(taskrouterTaskQueueDefaults, "maxReservedWorkers", o.MaxReservedWorkers, n.MaxReservedWorkers)
	d.strDef(taskrouterTaskQueueDefaults, "taskOrder", o.TaskOrder, n.TaskOrder)
	d.strDef(taskrouterTaskQueueDefaults, "reservationActivitySid", o.ReservationActivitySid, n.ReservationActivitySid)
	d.strDef(taskrouterTaskQueueDefaults, "assignmentActivitySid", o.AssignmentActivitySid, n.AssignmentActivitySid)
	return d.response(), nil
}

func (a TaskRouterTaskQueueArgs) form() url.Values {
	form := url.Values{"FriendlyName": {a.FriendlyName}}
	setS(form, "TargetWorkers", a.TargetWorkers)
	setI(form, "MaxReservedWorkers", a.MaxReservedWorkers)
	setS(form, "TaskOrder", a.TaskOrder)
	setS(form, "ReservationActivitySid", a.ReservationActivitySid)
	setS(form, "AssignmentActivitySid", a.AssignmentActivitySid)
	return form
}

func (r *TaskRouterTaskQueue) Create(ctx context.Context, req infer.CreateRequest[TaskRouterTaskQueueArgs]) (infer.CreateResponse[TaskRouterTaskQueueState], error) {
	type resp = infer.CreateResponse[TaskRouterTaskQueueState]
	a := req.Inputs
	if req.DryRun {
		return resp{Output: TaskRouterTaskQueueState{TaskRouterTaskQueueArgs: a}}, nil
	}
	c, err := clientFor(ctx)
	if err != nil {
		return resp{}, err
	}
	o, err := c.do(ctx, "POST", trProduct, trPath(a.WorkspaceSid, "TaskQueues", ""), nil, a.form())
	if err != nil {
		return resp{}, fmt.Errorf("create task queue in TaskRouter workspace %s: %w", a.WorkspaceSid, err)
	}
	st := TaskRouterTaskQueueState{TaskRouterTaskQueueArgs: a}
	st.Sid, st.DateCreated, st.DateUpdated = trSid(o)
	return resp{ID: a.WorkspaceSid + "/" + st.Sid, Output: st}, nil
}

func (r *TaskRouterTaskQueue) Read(ctx context.Context, req infer.ReadRequest[TaskRouterTaskQueueArgs, TaskRouterTaskQueueState]) (infer.ReadResponse[TaskRouterTaskQueueArgs, TaskRouterTaskQueueState], error) {
	type resp = infer.ReadResponse[TaskRouterTaskQueueArgs, TaskRouterTaskQueueState]
	ws, sid, err := trSplitID(req.ID)
	if err != nil {
		return resp{}, err
	}
	c, err := clientFor(ctx)
	if err != nil {
		return resp{}, err
	}
	o, err := c.do(ctx, "GET", trProduct, trPath(ws, "TaskQueues", sid), nil, nil)
	if err != nil {
		if isNotFound(err) {
			if err := notFoundOnImport("TaskRouter task queue", req.ID, req.Inputs, req.State); err != nil {
				return resp{}, err
			}
			return resp{}, nil
		}
		return resp{}, fmt.Errorf("read TaskRouter task queue %s: %w", req.ID, err)
	}
	in := req.Inputs
	imported := in.WorkspaceSid == ""
	in.WorkspaceSid = ws
	in.FriendlyName = o.str("friendly_name")
	in.TargetWorkers = trStr(in.TargetWorkers, o.optStr("target_workers"), taskrouterTaskQueueDefaults.str("targetWorkers"), imported)
	in.MaxReservedWorkers = trInt(in.MaxReservedWorkers, o.optInt("max_reserved_workers"), taskrouterTaskQueueDefaults.integer("maxReservedWorkers"), imported)
	in.TaskOrder = trStr(in.TaskOrder, o.optStr("task_order"), taskrouterTaskQueueDefaults.str("taskOrder"), imported)
	in.ReservationActivitySid = trStr(in.ReservationActivitySid, o.optStr("reservation_activity_sid"), taskrouterTaskQueueDefaults.str("reservationActivitySid"), imported)
	in.AssignmentActivitySid = trStr(in.AssignmentActivitySid, o.optStr("assignment_activity_sid"), taskrouterTaskQueueDefaults.str("assignmentActivitySid"), imported)
	st := TaskRouterTaskQueueState{TaskRouterTaskQueueArgs: in}
	st.Sid, st.DateCreated, st.DateUpdated = trSid(o)
	return resp{ID: req.ID, Inputs: in, State: st}, nil
}

func (r *TaskRouterTaskQueue) Update(ctx context.Context, req infer.UpdateRequest[TaskRouterTaskQueueArgs, TaskRouterTaskQueueState]) (infer.UpdateResponse[TaskRouterTaskQueueState], error) {
	type resp = infer.UpdateResponse[TaskRouterTaskQueueState]
	a := req.Inputs
	st := req.State
	st.TaskRouterTaskQueueArgs = a
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
	o, err := c.do(ctx, "POST", trProduct, trPath(ws, "TaskQueues", sid), nil, a.form())
	if err != nil {
		return resp{}, fmt.Errorf("update TaskRouter task queue %s: %w", req.ID, err)
	}
	st.DateUpdated = o.str("date_updated")
	return resp{Output: st}, nil
}

func (r *TaskRouterTaskQueue) Delete(ctx context.Context, req infer.DeleteRequest[TaskRouterTaskQueueState]) (infer.DeleteResponse, error) {
	return infer.DeleteResponse{}, trDelete(ctx, "task queue", "TaskQueues", req.ID)
}

// taskrouterTaskQueueDefaults are the Twilio defaults of this resource, shared by Read (import adoption) and Diff.
var taskrouterTaskQueueDefaults = trDefaults{
	"targetWorkers":          "1==1",
	"maxReservedWorkers":     1,
	"taskOrder":              "FIFO",
	"reservationActivitySid": "",
	"assignmentActivitySid":  "",
}
