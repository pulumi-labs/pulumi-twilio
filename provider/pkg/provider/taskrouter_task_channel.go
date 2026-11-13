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

// TaskRouterTaskChannel manages a task channel of a TaskRouter workspace: a type of task such as voice or sms.
type TaskRouterTaskChannel struct{}

// TaskRouterTaskChannelArgs are the inputs of a TaskRouterTaskChannel.
type TaskRouterTaskChannelArgs struct {
	WorkspaceSid            string `pulumi:"workspaceSid" provider:"replaceOnChanges"`
	UniqueName              string `pulumi:"uniqueName" provider:"replaceOnChanges"`
	FriendlyName            string `pulumi:"friendlyName"`
	ChannelOptimizedRouting *bool  `pulumi:"channelOptimizedRouting,optional"`
}

func (a *TaskRouterTaskChannelArgs) Annotate(an infer.Annotator) {
	an.Describe(&a.WorkspaceSid, "SID (WS...) of the workspace. Changing it replaces the resource.")
	an.Describe(&a.UniqueName, "Unique name of the channel, such as voice, sms or email. It cannot be changed after creation, so changing it replaces the resource.")
	an.Describe(&a.FriendlyName, "Human-readable name of the channel.")
	an.Describe(&a.ChannelOptimizedRouting, "Whether TaskRouter prefers the worker that has been available longest in the channel (Twilio default false). Removing it leaves the setting unchanged.")
}

// TaskRouterTaskChannelState is the persisted state of a TaskRouterTaskChannel.
type TaskRouterTaskChannelState struct {
	TaskRouterTaskChannelArgs

	Sid         string `pulumi:"sid"`
	DateCreated string `pulumi:"dateCreated"`
	DateUpdated string `pulumi:"dateUpdated"`
}

func (s *TaskRouterTaskChannelState) Annotate(a infer.Annotator) {
	a.Describe(&s.Sid, "The task channel SID (TC...).")
	a.Describe(&s.DateCreated, "When the channel was created (ISO 8601).")
	a.Describe(&s.DateUpdated, "When the channel was last updated (ISO 8601).")
}

func (r *TaskRouterTaskChannel) Annotate(a infer.Annotator) {
	a.Describe(&r, "A task channel of a TaskRouter workspace. The resource ID is <workspaceSid>/<sid>.")
}

var _ infer.CustomResource[TaskRouterTaskChannelArgs, TaskRouterTaskChannelState] = (*TaskRouterTaskChannel)(nil)
var _ infer.CustomCheck[TaskRouterTaskChannelArgs] = (*TaskRouterTaskChannel)(nil)
var _ infer.CustomDiff[TaskRouterTaskChannelArgs, TaskRouterTaskChannelState] = (*TaskRouterTaskChannel)(nil)
var _ infer.CustomRead[TaskRouterTaskChannelArgs, TaskRouterTaskChannelState] = (*TaskRouterTaskChannel)(nil)
var _ infer.CustomUpdate[TaskRouterTaskChannelArgs, TaskRouterTaskChannelState] = (*TaskRouterTaskChannel)(nil)
var _ infer.CustomDelete[TaskRouterTaskChannelState] = (*TaskRouterTaskChannel)(nil)

func (r *TaskRouterTaskChannel) Check(ctx context.Context, req infer.CheckRequest) (infer.CheckResponse[TaskRouterTaskChannelArgs], error) {
	args, failures, err := infer.DefaultCheck[TaskRouterTaskChannelArgs](ctx, req.NewInputs)
	if err != nil || len(failures) > 0 {
		return infer.CheckResponse[TaskRouterTaskChannelArgs]{Inputs: args, Failures: failures}, err
	}
	c := &checker{unk: unknowns(req.NewInputs)}
	trCheckWorkspace(c, args.WorkspaceSid)
	if !c.unk("uniqueName") && args.UniqueName == "" {
		c.fail("uniqueName", "uniqueName must not be empty")
	}
	if !c.unk("friendlyName") && args.FriendlyName == "" {
		c.fail("friendlyName", "friendlyName must not be empty")
	}
	return infer.CheckResponse[TaskRouterTaskChannelArgs]{Inputs: args, Failures: c.failures}, nil
}

func (r *TaskRouterTaskChannel) Diff(_ context.Context, req infer.DiffRequest[TaskRouterTaskChannelArgs, TaskRouterTaskChannelState]) (infer.DiffResponse, error) {
	o, n := req.State.TaskRouterTaskChannelArgs, req.Inputs
	d := trDiffSet{}
	d.replace("workspaceSid", o.WorkspaceSid, n.WorkspaceSid)
	d.replace("uniqueName", o.UniqueName, n.UniqueName)
	d.requiredStr("friendlyName", o.FriendlyName, n.FriendlyName)
	d.boolDef(taskrouterTaskChannelDefaults, "channelOptimizedRouting", o.ChannelOptimizedRouting, n.ChannelOptimizedRouting)
	return d.response(), nil
}

func (r *TaskRouterTaskChannel) Create(ctx context.Context, req infer.CreateRequest[TaskRouterTaskChannelArgs]) (infer.CreateResponse[TaskRouterTaskChannelState], error) {
	type resp = infer.CreateResponse[TaskRouterTaskChannelState]
	a := req.Inputs
	if req.DryRun {
		return resp{Output: TaskRouterTaskChannelState{TaskRouterTaskChannelArgs: a}}, nil
	}
	c, err := clientFor(ctx)
	if err != nil {
		return resp{}, err
	}
	form := url.Values{"FriendlyName": {a.FriendlyName}, "UniqueName": {a.UniqueName}}
	setB(form, "ChannelOptimizedRouting", a.ChannelOptimizedRouting)
	o, err := c.do(ctx, "POST", trProduct, trPath(a.WorkspaceSid, "TaskChannels", ""), nil, form)
	if err != nil {
		return resp{}, fmt.Errorf("create task channel in TaskRouter workspace %s: %w", a.WorkspaceSid, err)
	}
	st := TaskRouterTaskChannelState{TaskRouterTaskChannelArgs: a}
	st.Sid, st.DateCreated, st.DateUpdated = trSid(o)
	return resp{ID: a.WorkspaceSid + "/" + st.Sid, Output: st}, nil
}

func (r *TaskRouterTaskChannel) Read(ctx context.Context, req infer.ReadRequest[TaskRouterTaskChannelArgs, TaskRouterTaskChannelState]) (infer.ReadResponse[TaskRouterTaskChannelArgs, TaskRouterTaskChannelState], error) {
	type resp = infer.ReadResponse[TaskRouterTaskChannelArgs, TaskRouterTaskChannelState]
	ws, sid, err := trSplitID(req.ID)
	if err != nil {
		return resp{}, err
	}
	c, err := clientFor(ctx)
	if err != nil {
		return resp{}, err
	}
	o, err := c.do(ctx, "GET", trProduct, trPath(ws, "TaskChannels", sid), nil, nil)
	if err != nil {
		if isNotFound(err) {
			if err := notFoundOnImport("TaskRouter task channel", req.ID, req.Inputs, req.State); err != nil {
				return resp{}, err
			}
			return resp{}, nil
		}
		return resp{}, fmt.Errorf("read TaskRouter task channel %s: %w", req.ID, err)
	}
	in := req.Inputs
	imported := in.WorkspaceSid == ""
	in.WorkspaceSid = ws
	in.UniqueName = o.str("unique_name")
	in.FriendlyName = o.str("friendly_name")
	in.ChannelOptimizedRouting = trBool(in.ChannelOptimizedRouting, o.optBool("channel_optimized_routing"), taskrouterTaskChannelDefaults.boolean("channelOptimizedRouting"), imported)
	st := TaskRouterTaskChannelState{TaskRouterTaskChannelArgs: in}
	st.Sid, st.DateCreated, st.DateUpdated = trSid(o)
	return resp{ID: req.ID, Inputs: in, State: st}, nil
}

func (r *TaskRouterTaskChannel) Update(ctx context.Context, req infer.UpdateRequest[TaskRouterTaskChannelArgs, TaskRouterTaskChannelState]) (infer.UpdateResponse[TaskRouterTaskChannelState], error) {
	type resp = infer.UpdateResponse[TaskRouterTaskChannelState]
	a := req.Inputs
	st := req.State
	st.TaskRouterTaskChannelArgs = a
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
	setB(form, "ChannelOptimizedRouting", a.ChannelOptimizedRouting)
	o, err := c.do(ctx, "POST", trProduct, trPath(ws, "TaskChannels", sid), nil, form)
	if err != nil {
		return resp{}, fmt.Errorf("update TaskRouter task channel %s: %w", req.ID, err)
	}
	st.DateUpdated = o.str("date_updated")
	return resp{Output: st}, nil
}

func (r *TaskRouterTaskChannel) Delete(ctx context.Context, req infer.DeleteRequest[TaskRouterTaskChannelState]) (infer.DeleteResponse, error) {
	return infer.DeleteResponse{}, trDelete(ctx, "task channel", "TaskChannels", req.ID)
}

// taskrouterTaskChannelDefaults are the Twilio defaults of this resource, shared by Read (import adoption) and Diff.
var taskrouterTaskChannelDefaults = trDefaults{
	"channelOptimizedRouting": false,
}
