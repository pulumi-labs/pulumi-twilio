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

	"github.com/pulumi/pulumi-go-provider/infer"
)

// Lookups of the TaskRouter objects. Each reads one object by SID and returns what Twilio reports,
// null where Twilio reports nothing.

// trLookup reads one object for a lookup function.
func trLookup(ctx context.Context, fn, what, path string) (obj, error) {
	c, err := clientFor(ctx)
	if err != nil {
		return nil, err
	}
	o, err := c.do(ctx, "GET", trProduct, path, nil, nil)
	if err != nil {
		return nil, lookupFailure(fn, what, err)
	}
	return o, nil
}

// ------------------------------------------------------------ getTaskRouterWorkspace

// GetTaskRouterWorkspace looks up a TaskRouter workspace by SID.
type GetTaskRouterWorkspace struct{}

// GetTaskRouterWorkspaceArgs are the lookup arguments.
type GetTaskRouterWorkspaceArgs struct {
	WorkspaceSid string `pulumi:"workspaceSid"`
}

func (a *GetTaskRouterWorkspaceArgs) Annotate(an infer.Annotator) {
	an.Describe(&a.WorkspaceSid, "SID (WS...) of the TaskRouter workspace.")
}

// GetTaskRouterWorkspaceResult is the looked-up workspace.
type GetTaskRouterWorkspaceResult struct {
	Sid                  string  `pulumi:"sid"`
	FriendlyName         string  `pulumi:"friendlyName"`
	EventCallbackUrl     *string `pulumi:"eventCallbackUrl,optional"`
	EventsFilter         *string `pulumi:"eventsFilter,optional"`
	MultiTaskEnabled     *bool   `pulumi:"multiTaskEnabled,optional"`
	PrioritizeQueueOrder *string `pulumi:"prioritizeQueueOrder,optional"`
	DefaultActivitySid   *string `pulumi:"defaultActivitySid,optional"`
	TimeoutActivitySid   *string `pulumi:"timeoutActivitySid,optional"`
	DateCreated          string  `pulumi:"dateCreated"`
	DateUpdated          string  `pulumi:"dateUpdated"`
}

func (f *GetTaskRouterWorkspace) Annotate(a infer.Annotator) {
	a.Describe(&f, "Look up a TaskRouter workspace by SID.")
}

func (f *GetTaskRouterWorkspace) Invoke(ctx context.Context, req infer.FunctionRequest[GetTaskRouterWorkspaceArgs]) (infer.FunctionResponse[GetTaskRouterWorkspaceResult], error) {
	const fn = "getTaskRouterWorkspace"
	type resp = infer.FunctionResponse[GetTaskRouterWorkspaceResult]
	if err := requireArg(fn, "workspaceSid", req.Input.WorkspaceSid); err != nil {
		return resp{}, err
	}
	o, err := trLookup(ctx, fn, "TaskRouter workspace "+req.Input.WorkspaceSid, trPath(req.Input.WorkspaceSid, "", ""))
	if err != nil {
		return resp{}, err
	}
	sid, created, updated := trSid(o)
	return resp{Output: GetTaskRouterWorkspaceResult{
		Sid: sid, FriendlyName: o.str("friendly_name"),
		EventCallbackUrl: nonEmpty(o.optStr("event_callback_url")), EventsFilter: nonEmpty(o.optStr("events_filter")),
		MultiTaskEnabled: o.optBool("multi_task_enabled"), PrioritizeQueueOrder: nonEmpty(o.optStr("prioritize_queue_order")),
		DefaultActivitySid: nonEmpty(o.optStr("default_activity_sid")), TimeoutActivitySid: nonEmpty(o.optStr("timeout_activity_sid")),
		DateCreated: created, DateUpdated: updated,
	}}, nil
}

// ------------------------------------------------------------ getTaskRouterActivity

// GetTaskRouterActivity looks up an activity of a TaskRouter workspace by SID.
type GetTaskRouterActivity struct{}

// GetTaskRouterActivityArgs are the lookup arguments.
type GetTaskRouterActivityArgs struct {
	WorkspaceSid string `pulumi:"workspaceSid"`
	Sid          string `pulumi:"sid"`
}

func (a *GetTaskRouterActivityArgs) Annotate(an infer.Annotator) {
	an.Describe(&a.WorkspaceSid, "SID (WS...) of the TaskRouter workspace.")
	an.Describe(&a.Sid, "SID (WA...) of the activity.")
}

// GetTaskRouterActivityResult is the looked-up activity.
type GetTaskRouterActivityResult struct {
	Sid          string `pulumi:"sid"`
	WorkspaceSid string `pulumi:"workspaceSid"`
	FriendlyName string `pulumi:"friendlyName"`
	Available    *bool  `pulumi:"available,optional"`
	DateCreated  string `pulumi:"dateCreated"`
	DateUpdated  string `pulumi:"dateUpdated"`
}

func (f *GetTaskRouterActivity) Annotate(a infer.Annotator) {
	a.Describe(&f, "Look up an activity of a TaskRouter workspace by SID.")
}

func (f *GetTaskRouterActivity) Invoke(ctx context.Context, req infer.FunctionRequest[GetTaskRouterActivityArgs]) (infer.FunctionResponse[GetTaskRouterActivityResult], error) {
	const fn = "getTaskRouterActivity"
	type resp = infer.FunctionResponse[GetTaskRouterActivityResult]
	in := req.Input
	if err := requireArgs(fn, [2]string{"workspaceSid", in.WorkspaceSid}, [2]string{"sid", in.Sid}); err != nil {
		return resp{}, err
	}
	o, err := trLookup(ctx, fn, "TaskRouter activity "+in.Sid, trPath(in.WorkspaceSid, "Activities", in.Sid))
	if err != nil {
		return resp{}, err
	}
	sid, created, updated := trSid(o)
	return resp{Output: GetTaskRouterActivityResult{
		Sid: sid, WorkspaceSid: in.WorkspaceSid, FriendlyName: o.str("friendly_name"), Available: o.optBool("available"),
		DateCreated: created, DateUpdated: updated,
	}}, nil
}

// ------------------------------------------------------------ getTaskRouterTaskChannel

// GetTaskRouterTaskChannel looks up a task channel of a TaskRouter workspace by SID or unique name.
type GetTaskRouterTaskChannel struct{}

// GetTaskRouterTaskChannelArgs are the lookup arguments.
type GetTaskRouterTaskChannelArgs struct {
	WorkspaceSid string `pulumi:"workspaceSid"`
	Sid          string `pulumi:"sid"`
}

func (a *GetTaskRouterTaskChannelArgs) Annotate(an infer.Annotator) {
	an.Describe(&a.WorkspaceSid, "SID (WS...) of the TaskRouter workspace.")
	an.Describe(&a.Sid, "SID (TC...) or unique name of the task channel.")
}

// GetTaskRouterTaskChannelResult is the looked-up task channel.
type GetTaskRouterTaskChannelResult struct {
	Sid                     string `pulumi:"sid"`
	WorkspaceSid            string `pulumi:"workspaceSid"`
	UniqueName              string `pulumi:"uniqueName"`
	FriendlyName            string `pulumi:"friendlyName"`
	ChannelOptimizedRouting *bool  `pulumi:"channelOptimizedRouting,optional"`
	DateCreated             string `pulumi:"dateCreated"`
	DateUpdated             string `pulumi:"dateUpdated"`
}

func (f *GetTaskRouterTaskChannel) Annotate(a infer.Annotator) {
	a.Describe(&f, "Look up a task channel of a TaskRouter workspace by SID or unique name.")
}

func (f *GetTaskRouterTaskChannel) Invoke(ctx context.Context, req infer.FunctionRequest[GetTaskRouterTaskChannelArgs]) (infer.FunctionResponse[GetTaskRouterTaskChannelResult], error) {
	const fn = "getTaskRouterTaskChannel"
	type resp = infer.FunctionResponse[GetTaskRouterTaskChannelResult]
	in := req.Input
	if err := requireArgs(fn, [2]string{"workspaceSid", in.WorkspaceSid}, [2]string{"sid", in.Sid}); err != nil {
		return resp{}, err
	}
	o, err := trLookup(ctx, fn, "TaskRouter task channel "+in.Sid, trPath(in.WorkspaceSid, "TaskChannels", in.Sid))
	if err != nil {
		return resp{}, err
	}
	sid, created, updated := trSid(o)
	return resp{Output: GetTaskRouterTaskChannelResult{
		Sid: sid, WorkspaceSid: in.WorkspaceSid, UniqueName: o.str("unique_name"), FriendlyName: o.str("friendly_name"),
		ChannelOptimizedRouting: o.optBool("channel_optimized_routing"), DateCreated: created, DateUpdated: updated,
	}}, nil
}

// ------------------------------------------------------------ getTaskRouterTaskQueue

// GetTaskRouterTaskQueue looks up a task queue of a TaskRouter workspace by SID.
type GetTaskRouterTaskQueue struct{}

// GetTaskRouterTaskQueueArgs are the lookup arguments.
type GetTaskRouterTaskQueueArgs struct {
	WorkspaceSid string `pulumi:"workspaceSid"`
	Sid          string `pulumi:"sid"`
}

func (a *GetTaskRouterTaskQueueArgs) Annotate(an infer.Annotator) {
	an.Describe(&a.WorkspaceSid, "SID (WS...) of the TaskRouter workspace.")
	an.Describe(&a.Sid, "SID (WQ...) of the task queue.")
}

// GetTaskRouterTaskQueueResult is the looked-up task queue.
type GetTaskRouterTaskQueueResult struct {
	Sid                    string  `pulumi:"sid"`
	WorkspaceSid           string  `pulumi:"workspaceSid"`
	FriendlyName           string  `pulumi:"friendlyName"`
	TargetWorkers          *string `pulumi:"targetWorkers,optional"`
	MaxReservedWorkers     *int    `pulumi:"maxReservedWorkers,optional"`
	TaskOrder              *string `pulumi:"taskOrder,optional"`
	ReservationActivitySid *string `pulumi:"reservationActivitySid,optional"`
	AssignmentActivitySid  *string `pulumi:"assignmentActivitySid,optional"`
	DateCreated            string  `pulumi:"dateCreated"`
	DateUpdated            string  `pulumi:"dateUpdated"`
}

func (f *GetTaskRouterTaskQueue) Annotate(a infer.Annotator) {
	a.Describe(&f, "Look up a task queue of a TaskRouter workspace by SID.")
}

func (f *GetTaskRouterTaskQueue) Invoke(ctx context.Context, req infer.FunctionRequest[GetTaskRouterTaskQueueArgs]) (infer.FunctionResponse[GetTaskRouterTaskQueueResult], error) {
	const fn = "getTaskRouterTaskQueue"
	type resp = infer.FunctionResponse[GetTaskRouterTaskQueueResult]
	in := req.Input
	if err := requireArgs(fn, [2]string{"workspaceSid", in.WorkspaceSid}, [2]string{"sid", in.Sid}); err != nil {
		return resp{}, err
	}
	o, err := trLookup(ctx, fn, "TaskRouter task queue "+in.Sid, trPath(in.WorkspaceSid, "TaskQueues", in.Sid))
	if err != nil {
		return resp{}, err
	}
	sid, created, updated := trSid(o)
	return resp{Output: GetTaskRouterTaskQueueResult{
		Sid: sid, WorkspaceSid: in.WorkspaceSid, FriendlyName: o.str("friendly_name"),
		TargetWorkers: nonEmpty(o.optStr("target_workers")), MaxReservedWorkers: o.optInt("max_reserved_workers"),
		TaskOrder:              nonEmpty(o.optStr("task_order")),
		ReservationActivitySid: nonEmpty(o.optStr("reservation_activity_sid")), AssignmentActivitySid: nonEmpty(o.optStr("assignment_activity_sid")),
		DateCreated: created, DateUpdated: updated,
	}}, nil
}

// ------------------------------------------------------------ getTaskRouterWorkflow

// GetTaskRouterWorkflow looks up a workflow of a TaskRouter workspace by SID.
type GetTaskRouterWorkflow struct{}

// GetTaskRouterWorkflowArgs are the lookup arguments.
type GetTaskRouterWorkflowArgs struct {
	WorkspaceSid string `pulumi:"workspaceSid"`
	Sid          string `pulumi:"sid"`
}

func (a *GetTaskRouterWorkflowArgs) Annotate(an infer.Annotator) {
	an.Describe(&a.WorkspaceSid, "SID (WS...) of the TaskRouter workspace.")
	an.Describe(&a.Sid, "SID (WW...) of the workflow.")
}

// GetTaskRouterWorkflowResult is the looked-up workflow.
type GetTaskRouterWorkflowResult struct {
	Sid                           string  `pulumi:"sid"`
	WorkspaceSid                  string  `pulumi:"workspaceSid"`
	FriendlyName                  string  `pulumi:"friendlyName"`
	Configuration                 string  `pulumi:"configuration"`
	AssignmentCallbackUrl         *string `pulumi:"assignmentCallbackUrl,optional"`
	FallbackAssignmentCallbackUrl *string `pulumi:"fallbackAssignmentCallbackUrl,optional"`
	TaskReservationTimeout        *int    `pulumi:"taskReservationTimeout,optional"`
	DateCreated                   string  `pulumi:"dateCreated"`
	DateUpdated                   string  `pulumi:"dateUpdated"`
}

func (f *GetTaskRouterWorkflow) Annotate(a infer.Annotator) {
	a.Describe(&f, "Look up a workflow of a TaskRouter workspace by SID.")
}

func (f *GetTaskRouterWorkflow) Invoke(ctx context.Context, req infer.FunctionRequest[GetTaskRouterWorkflowArgs]) (infer.FunctionResponse[GetTaskRouterWorkflowResult], error) {
	const fn = "getTaskRouterWorkflow"
	type resp = infer.FunctionResponse[GetTaskRouterWorkflowResult]
	in := req.Input
	if err := requireArgs(fn, [2]string{"workspaceSid", in.WorkspaceSid}, [2]string{"sid", in.Sid}); err != nil {
		return resp{}, err
	}
	o, err := trLookup(ctx, fn, "TaskRouter workflow "+in.Sid, trPath(in.WorkspaceSid, "Workflows", in.Sid))
	if err != nil {
		return resp{}, err
	}
	sid, created, updated := trSid(o)
	return resp{Output: GetTaskRouterWorkflowResult{
		Sid: sid, WorkspaceSid: in.WorkspaceSid, FriendlyName: o.str("friendly_name"), Configuration: o.str("configuration"),
		AssignmentCallbackUrl:         nonEmpty(o.optStr("assignment_callback_url")),
		FallbackAssignmentCallbackUrl: nonEmpty(o.optStr("fallback_assignment_callback_url")),
		TaskReservationTimeout:        o.optInt("task_reservation_timeout"), DateCreated: created, DateUpdated: updated,
	}}, nil
}

// ------------------------------------------------------------ getTaskRouterWorker

// GetTaskRouterWorker looks up a worker of a TaskRouter workspace by SID.
type GetTaskRouterWorker struct{}

// GetTaskRouterWorkerArgs are the lookup arguments.
type GetTaskRouterWorkerArgs struct {
	WorkspaceSid string `pulumi:"workspaceSid"`
	Sid          string `pulumi:"sid"`
}

func (a *GetTaskRouterWorkerArgs) Annotate(an infer.Annotator) {
	an.Describe(&a.WorkspaceSid, "SID (WS...) of the TaskRouter workspace.")
	an.Describe(&a.Sid, "SID (WK...) of the worker.")
}

// GetTaskRouterWorkerResult is the looked-up worker.
type GetTaskRouterWorkerResult struct {
	Sid          string  `pulumi:"sid"`
	WorkspaceSid string  `pulumi:"workspaceSid"`
	FriendlyName string  `pulumi:"friendlyName"`
	Attributes   *string `pulumi:"attributes,optional"`
	ActivitySid  *string `pulumi:"activitySid,optional"`
	ActivityName *string `pulumi:"activityName,optional"`
	Available    *bool   `pulumi:"available,optional"`
	DateCreated  string  `pulumi:"dateCreated"`
	DateUpdated  string  `pulumi:"dateUpdated"`
}

func (f *GetTaskRouterWorker) Annotate(a infer.Annotator) {
	a.Describe(&f, "Look up a worker of a TaskRouter workspace by SID. The activity is the worker's current one at the time of the lookup.")
}

func (f *GetTaskRouterWorker) Invoke(ctx context.Context, req infer.FunctionRequest[GetTaskRouterWorkerArgs]) (infer.FunctionResponse[GetTaskRouterWorkerResult], error) {
	const fn = "getTaskRouterWorker"
	type resp = infer.FunctionResponse[GetTaskRouterWorkerResult]
	in := req.Input
	if err := requireArgs(fn, [2]string{"workspaceSid", in.WorkspaceSid}, [2]string{"sid", in.Sid}); err != nil {
		return resp{}, err
	}
	o, err := trLookup(ctx, fn, "TaskRouter worker "+in.Sid, trPath(in.WorkspaceSid, "Workers", in.Sid))
	if err != nil {
		return resp{}, err
	}
	sid, created, updated := trSid(o)
	return resp{Output: GetTaskRouterWorkerResult{
		Sid: sid, WorkspaceSid: in.WorkspaceSid, FriendlyName: o.str("friendly_name"), Attributes: nonEmpty(o.optStr("attributes")),
		ActivitySid: nonEmpty(o.optStr("activity_sid")), ActivityName: nonEmpty(o.optStr("activity_name")), Available: o.optBool("available"),
		DateCreated: created, DateUpdated: updated,
	}}, nil
}
