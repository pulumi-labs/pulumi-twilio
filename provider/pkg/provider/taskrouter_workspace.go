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

// TaskRouterWorkspace manages a TaskRouter workspace, the container of workers, queues, workflows and tasks.
type TaskRouterWorkspace struct{}

// TaskRouterWorkspaceArgs are the inputs of a TaskRouterWorkspace.
type TaskRouterWorkspaceArgs struct {
	FriendlyName         string  `pulumi:"friendlyName"`
	EventCallbackUrl     *string `pulumi:"eventCallbackUrl,optional"`
	EventsFilter         *string `pulumi:"eventsFilter,optional"`
	MultiTaskEnabled     *bool   `pulumi:"multiTaskEnabled,optional"`
	PrioritizeQueueOrder *string `pulumi:"prioritizeQueueOrder,optional"`
}

func (a *TaskRouterWorkspaceArgs) Annotate(an infer.Annotator) {
	an.Describe(&a.FriendlyName, "Human-readable name of the workspace.")
	an.Describe(&a.EventCallbackUrl, "URL Twilio posts workspace events to. Removing it leaves the setting unchanged.")
	an.Describe(&a.EventsFilter, "Comma-separated list of event types sent to eventCallbackUrl, for example task.created,worker.activity.update. Removing it leaves the setting unchanged.")
	an.Describe(&a.MultiTaskEnabled, "Whether workers may handle several tasks at once (Twilio default false). Twilio does not allow turning it back off once it is on. Removing it leaves the setting unchanged.")
	an.Describe(&a.PrioritizeQueueOrder, "Which queue order is used first when a task is routed through several queues: FIFO or LIFO (Twilio default FIFO). Removing it leaves the setting unchanged.")
}

// TaskRouterWorkspaceState is the persisted state of a TaskRouterWorkspace.
type TaskRouterWorkspaceState struct {
	TaskRouterWorkspaceArgs

	Sid                string `pulumi:"sid"`
	DateCreated        string `pulumi:"dateCreated"`
	DateUpdated        string `pulumi:"dateUpdated"`
	DefaultActivitySid string `pulumi:"defaultActivitySid"`
	TimeoutActivitySid string `pulumi:"timeoutActivitySid"`
}

func (s *TaskRouterWorkspaceState) Annotate(a infer.Annotator) {
	a.Describe(&s.Sid, "The workspace SID (WS...).")
	a.Describe(&s.DateCreated, "When the workspace was created (ISO 8601).")
	a.Describe(&s.DateUpdated, "When the workspace was last updated (ISO 8601).")
	a.Describe(&s.DefaultActivitySid, "SID (WA...) of the activity new workers start in. Twilio creates it with the workspace.")
	a.Describe(&s.TimeoutActivitySid, "SID (WA...) of the activity a worker moves to when it does not answer a reservation in time. Twilio creates it with the workspace.")
}

func (r *TaskRouterWorkspace) Annotate(a infer.Annotator) {
	a.Describe(&r, "A TaskRouter workspace. Twilio creates a set of default activities with it (see defaultActivitySid and timeoutActivitySid); add TaskRouterActivity, TaskRouterTaskChannel, TaskRouterTaskQueue, TaskRouterWorkflow and TaskRouterWorker resources to configure it. Destroying a workspace deletes everything in it. The resource ID is the workspace SID.")
}

var _ infer.CustomResource[TaskRouterWorkspaceArgs, TaskRouterWorkspaceState] = (*TaskRouterWorkspace)(nil)
var _ infer.CustomCheck[TaskRouterWorkspaceArgs] = (*TaskRouterWorkspace)(nil)
var _ infer.CustomDiff[TaskRouterWorkspaceArgs, TaskRouterWorkspaceState] = (*TaskRouterWorkspace)(nil)
var _ infer.CustomRead[TaskRouterWorkspaceArgs, TaskRouterWorkspaceState] = (*TaskRouterWorkspace)(nil)
var _ infer.CustomUpdate[TaskRouterWorkspaceArgs, TaskRouterWorkspaceState] = (*TaskRouterWorkspace)(nil)
var _ infer.CustomDelete[TaskRouterWorkspaceState] = (*TaskRouterWorkspace)(nil)

func (r *TaskRouterWorkspace) Check(ctx context.Context, req infer.CheckRequest) (infer.CheckResponse[TaskRouterWorkspaceArgs], error) {
	args, failures, err := infer.DefaultCheck[TaskRouterWorkspaceArgs](ctx, req.NewInputs)
	if err != nil || len(failures) > 0 {
		return infer.CheckResponse[TaskRouterWorkspaceArgs]{Inputs: args, Failures: failures}, err
	}
	c := &checker{unk: unknowns(req.NewInputs)}
	if !c.unk("friendlyName") && args.FriendlyName == "" {
		c.fail("friendlyName", "friendlyName must not be empty")
	}
	if v := args.PrioritizeQueueOrder; v != nil && !c.unk("prioritizeQueueOrder") && *v != "FIFO" && *v != "LIFO" {
		c.fail("prioritizeQueueOrder", "prioritizeQueueOrder must be FIFO or LIFO")
	}
	return infer.CheckResponse[TaskRouterWorkspaceArgs]{Inputs: args, Failures: c.failures}, nil
}

func (r *TaskRouterWorkspace) Diff(_ context.Context, req infer.DiffRequest[TaskRouterWorkspaceArgs, TaskRouterWorkspaceState]) (infer.DiffResponse, error) {
	o, n := req.State.TaskRouterWorkspaceArgs, req.Inputs
	d := trDiffSet{}
	d.requiredStr("friendlyName", o.FriendlyName, n.FriendlyName)
	d.str("eventCallbackUrl", o.EventCallbackUrl, n.EventCallbackUrl)
	d.str("eventsFilter", o.EventsFilter, n.EventsFilter)
	d.boolean("multiTaskEnabled", o.MultiTaskEnabled, n.MultiTaskEnabled)
	d.str("prioritizeQueueOrder", o.PrioritizeQueueOrder, n.PrioritizeQueueOrder)
	return d.response(), nil
}

func (a TaskRouterWorkspaceArgs) form() url.Values {
	form := url.Values{"FriendlyName": {a.FriendlyName}}
	setS(form, "EventCallbackUrl", a.EventCallbackUrl)
	setS(form, "EventsFilter", a.EventsFilter)
	setB(form, "MultiTaskEnabled", a.MultiTaskEnabled)
	setS(form, "PrioritizeQueueOrder", a.PrioritizeQueueOrder)
	return form
}

func trWorkspaceOutputs(st *TaskRouterWorkspaceState, o obj) {
	st.Sid, st.DateCreated, st.DateUpdated = trSid(o)
	st.DefaultActivitySid, st.TimeoutActivitySid = o.str("default_activity_sid"), o.str("timeout_activity_sid")
}

func (r *TaskRouterWorkspace) Create(ctx context.Context, req infer.CreateRequest[TaskRouterWorkspaceArgs]) (infer.CreateResponse[TaskRouterWorkspaceState], error) {
	type resp = infer.CreateResponse[TaskRouterWorkspaceState]
	a := req.Inputs
	if req.DryRun {
		return resp{Output: TaskRouterWorkspaceState{TaskRouterWorkspaceArgs: a}}, nil
	}
	c, err := clientFor(ctx)
	if err != nil {
		return resp{}, err
	}
	o, err := c.do(ctx, "POST", trProduct, trPath("", "", ""), nil, a.form())
	if err != nil {
		return resp{}, fmt.Errorf("create TaskRouter workspace: %w", err)
	}
	st := TaskRouterWorkspaceState{TaskRouterWorkspaceArgs: a}
	trWorkspaceOutputs(&st, o)
	return resp{ID: st.Sid, Output: st}, nil
}

func (r *TaskRouterWorkspace) Read(ctx context.Context, req infer.ReadRequest[TaskRouterWorkspaceArgs, TaskRouterWorkspaceState]) (infer.ReadResponse[TaskRouterWorkspaceArgs, TaskRouterWorkspaceState], error) {
	type resp = infer.ReadResponse[TaskRouterWorkspaceArgs, TaskRouterWorkspaceState]
	c, err := clientFor(ctx)
	if err != nil {
		return resp{}, err
	}
	o, err := c.do(ctx, "GET", trProduct, trPath(req.ID, "", ""), nil, nil)
	if err != nil {
		if isNotFound(err) {
			if err := notFoundOnImport("TaskRouter workspace", req.ID, req.Inputs, req.State); err != nil {
				return resp{}, err
			}
			return resp{}, nil
		}
		return resp{}, fmt.Errorf("read TaskRouter workspace %s: %w", req.ID, err)
	}
	in := req.Inputs
	imported := in.FriendlyName == ""
	in.FriendlyName = o.str("friendly_name")
	in.EventCallbackUrl = trStr(in.EventCallbackUrl, o.optStr("event_callback_url"), "", imported)
	in.EventsFilter = trStr(in.EventsFilter, o.optStr("events_filter"), "", imported)
	in.MultiTaskEnabled = trBool(in.MultiTaskEnabled, o.optBool("multi_task_enabled"), false, imported)
	in.PrioritizeQueueOrder = trStr(in.PrioritizeQueueOrder, o.optStr("prioritize_queue_order"), "FIFO", imported)
	st := TaskRouterWorkspaceState{TaskRouterWorkspaceArgs: in}
	trWorkspaceOutputs(&st, o)
	return resp{ID: req.ID, Inputs: in, State: st}, nil
}

func (r *TaskRouterWorkspace) Update(ctx context.Context, req infer.UpdateRequest[TaskRouterWorkspaceArgs, TaskRouterWorkspaceState]) (infer.UpdateResponse[TaskRouterWorkspaceState], error) {
	type resp = infer.UpdateResponse[TaskRouterWorkspaceState]
	a := req.Inputs
	st := req.State
	st.TaskRouterWorkspaceArgs = a
	if req.DryRun {
		return resp{Output: st}, nil
	}
	c, err := clientFor(ctx)
	if err != nil {
		return resp{}, err
	}
	o, err := c.do(ctx, "POST", trProduct, trPath(req.ID, "", ""), nil, a.form())
	if err != nil {
		return resp{}, fmt.Errorf("update TaskRouter workspace %s: %w", req.ID, err)
	}
	trWorkspaceOutputs(&st, o)
	return resp{Output: st}, nil
}

func (r *TaskRouterWorkspace) Delete(ctx context.Context, req infer.DeleteRequest[TaskRouterWorkspaceState]) (infer.DeleteResponse, error) {
	c, err := clientFor(ctx)
	if err != nil {
		return infer.DeleteResponse{}, err
	}
	if _, err := c.do(ctx, "DELETE", trProduct, trPath(req.ID, "", ""), nil, nil); err != nil && !isNotFound(err) {
		return infer.DeleteResponse{}, fmt.Errorf("delete TaskRouter workspace %s: %w", req.ID, err)
	}
	return infer.DeleteResponse{}, nil
}
