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
	"strings"

	"github.com/pulumi/pulumi-go-provider/infer"
)

// ConversationsRole manages a role of Conversations: a named set of permissions that is either a
// service role or a conversation role. The role is account-level, or belongs to one service when
// chatServiceSid is set.
type ConversationsRole struct{}

// ConversationsRoleArgs are the inputs of a ConversationsRole.
type ConversationsRoleArgs struct {
	ChatServiceSid *string  `pulumi:"chatServiceSid,optional" provider:"replaceOnChanges"`
	FriendlyName   string   `pulumi:"friendlyName" provider:"replaceOnChanges"`
	RoleType       string   `pulumi:"roleType" provider:"replaceOnChanges"`
	Permissions    []string `pulumi:"permissions"`
}

func (a *ConversationsRoleArgs) Annotate(an infer.Annotator) {
	an.Describe(&a.ChatServiceSid, "SID (IS...) of the Conversations service the role belongs to. Leave it out for an account-level role. Changing it replaces the role.")
	an.Describe(&a.FriendlyName, "Human-readable name of the role, up to 64 characters. It cannot be changed after creation, so changing it replaces the role.")
	an.Describe(&a.RoleType, "What the role applies to: conversation or service. It cannot be changed after creation, so changing it replaces the role.")
	an.Describe(&a.Permissions, "The permissions the role grants, for example sendMessage or editOwnMessage; the order is not significant. The full list is in the Twilio Conversations roles documentation.")
}

// ConversationsRoleState is the persisted state of a ConversationsRole.
type ConversationsRoleState struct {
	ConversationsRoleArgs

	Sid         string `pulumi:"sid"`
	DateCreated string `pulumi:"dateCreated"`
	DateUpdated string `pulumi:"dateUpdated"`
}

func (s *ConversationsRoleState) Annotate(a infer.Annotator) {
	a.Describe(&s.Sid, "The role SID (RL...).")
	a.Describe(&s.DateCreated, "When the role was created (ISO 8601).")
	a.Describe(&s.DateUpdated, "When the role was last updated (ISO 8601).")
}

func (r *ConversationsRole) Annotate(a infer.Annotator) {
	a.Describe(&r, "A Conversations role. The resource ID is <sid>, or <chatServiceSid>/<sid> for a role of one service.")
}

var _ infer.CustomResource[ConversationsRoleArgs, ConversationsRoleState] = (*ConversationsRole)(nil)
var _ infer.CustomCheck[ConversationsRoleArgs] = (*ConversationsRole)(nil)
var _ infer.CustomDiff[ConversationsRoleArgs, ConversationsRoleState] = (*ConversationsRole)(nil)
var _ infer.CustomRead[ConversationsRoleArgs, ConversationsRoleState] = (*ConversationsRole)(nil)
var _ infer.CustomUpdate[ConversationsRoleArgs, ConversationsRoleState] = (*ConversationsRole)(nil)
var _ infer.CustomDelete[ConversationsRoleState] = (*ConversationsRole)(nil)

// cvSplitRoleID splits <sid> or <chatServiceSid>/<sid>.
func cvSplitRoleID(id string) (service, sid string, err error) {
	switch parts := strings.Split(id, "/"); len(parts) {
	case 1:
		return "", parts[0], nil
	case 2:
		return parts[0], parts[1], nil
	}
	return "", "", fmt.Errorf("invalid Conversations role ID %q: want <sid> or <chatServiceSid>/<sid>", id)
}

func (r *ConversationsRole) Check(ctx context.Context, req infer.CheckRequest) (infer.CheckResponse[ConversationsRoleArgs], error) {
	args, failures, err := infer.DefaultCheck[ConversationsRoleArgs](ctx, req.NewInputs)
	if err != nil || len(failures) > 0 {
		return infer.CheckResponse[ConversationsRoleArgs]{Inputs: args, Failures: failures}, err
	}
	c := &checker{unk: unknowns(req.NewInputs)}
	cvCheckOptionalService(c, "chatServiceSid", args.ChatServiceSid)
	if !c.unk("friendlyName") && (args.FriendlyName == "" || len(args.FriendlyName) > 64) {
		c.fail("friendlyName", "friendlyName must be 1-64 characters")
	}
	c.oneOf("roleType", args.RoleType, []string{"conversation", "service"})
	if !c.unk("permissions") && len(args.Permissions) == 0 {
		c.fail("permissions", "permissions must list at least one permission")
	}
	return infer.CheckResponse[ConversationsRoleArgs]{Inputs: args, Failures: c.failures}, nil
}

func (r *ConversationsRole) Diff(_ context.Context, req infer.DiffRequest[ConversationsRoleArgs, ConversationsRoleState]) (infer.DiffResponse, error) {
	o, n := req.State.ConversationsRoleArgs, req.Inputs
	d := trDiffSet{}
	d.replace("chatServiceSid", derefOr(o.ChatServiceSid, ""), derefOr(n.ChatServiceSid, ""))
	d.replace("friendlyName", o.FriendlyName, n.FriendlyName)
	d.replace("roleType", o.RoleType, n.RoleType)
	d.strList("permissions", &o.Permissions, &n.Permissions)
	return d.response(), nil
}

func (r *ConversationsRole) Create(ctx context.Context, req infer.CreateRequest[ConversationsRoleArgs]) (infer.CreateResponse[ConversationsRoleState], error) {
	type resp = infer.CreateResponse[ConversationsRoleState]
	a := req.Inputs
	if req.DryRun {
		return resp{Output: ConversationsRoleState{ConversationsRoleArgs: a}}, nil
	}
	c, err := clientFor(ctx)
	if err != nil {
		return resp{}, err
	}
	form := url.Values{"FriendlyName": {a.FriendlyName}, "Type": {a.RoleType}}
	cvAddList(form, "Permission", &a.Permissions)
	service := derefOr(a.ChatServiceSid, "")
	o, err := c.do(ctx, "POST", cvProduct, cvPath(service, "Roles", ""), nil, form)
	if err != nil {
		return resp{}, fmt.Errorf("create Conversations role: %w", err)
	}
	st := ConversationsRoleState{ConversationsRoleArgs: a}
	st.Sid, st.DateCreated, st.DateUpdated = trSid(o)
	id := st.Sid
	if service != "" {
		id = service + "/" + st.Sid
	}
	return resp{ID: id, Output: st}, nil
}

func (r *ConversationsRole) Read(ctx context.Context, req infer.ReadRequest[ConversationsRoleArgs, ConversationsRoleState]) (infer.ReadResponse[ConversationsRoleArgs, ConversationsRoleState], error) {
	type resp = infer.ReadResponse[ConversationsRoleArgs, ConversationsRoleState]
	service, sid, err := cvSplitRoleID(req.ID)
	if err != nil {
		return resp{}, err
	}
	c, err := clientFor(ctx)
	if err != nil {
		return resp{}, err
	}
	o, err := c.do(ctx, "GET", cvProduct, cvPath(service, "Roles", sid), nil, nil)
	if err != nil {
		if isNotFound(err) {
			if err := notFoundOnImport("Conversations role", req.ID, req.Inputs, req.State); err != nil {
				return resp{}, err
			}
			return resp{}, nil
		}
		return resp{}, fmt.Errorf("read Conversations role %s: %w", req.ID, err)
	}
	in := req.Inputs
	in.ChatServiceSid = nil
	if service != "" {
		in.ChatServiceSid = &service
	}
	in.FriendlyName = o.str("friendly_name")
	in.RoleType = o.str("type")
	if api := o.optList("permissions"); api != nil && !sameSet(in.Permissions, *api) {
		in.Permissions = *api
	}
	st := ConversationsRoleState{ConversationsRoleArgs: in}
	st.Sid, st.DateCreated, st.DateUpdated = trSid(o)
	return resp{ID: req.ID, Inputs: in, State: st}, nil
}

func (r *ConversationsRole) Update(ctx context.Context, req infer.UpdateRequest[ConversationsRoleArgs, ConversationsRoleState]) (infer.UpdateResponse[ConversationsRoleState], error) {
	type resp = infer.UpdateResponse[ConversationsRoleState]
	st := req.State
	st.ConversationsRoleArgs = req.Inputs
	if req.DryRun {
		return resp{Output: st}, nil
	}
	service, sid, err := cvSplitRoleID(req.ID)
	if err != nil {
		return resp{}, err
	}
	c, err := clientFor(ctx)
	if err != nil {
		return resp{}, err
	}
	form := url.Values{}
	cvAddList(form, "Permission", &req.Inputs.Permissions)
	o, err := c.do(ctx, "POST", cvProduct, cvPath(service, "Roles", sid), nil, form)
	if err != nil {
		return resp{}, fmt.Errorf("update Conversations role %s: %w", req.ID, err)
	}
	st.DateUpdated = o.str("date_updated")
	return resp{Output: st}, nil
}

func (r *ConversationsRole) Delete(ctx context.Context, req infer.DeleteRequest[ConversationsRoleState]) (infer.DeleteResponse, error) {
	service, sid, err := cvSplitRoleID(req.ID)
	if err != nil {
		return infer.DeleteResponse{}, err
	}
	c, err := clientFor(ctx)
	if err != nil {
		return infer.DeleteResponse{}, err
	}
	if _, err := c.do(ctx, "DELETE", cvProduct, cvPath(service, "Roles", sid), nil, nil); err != nil && !isNotFound(err) {
		return infer.DeleteResponse{}, fmt.Errorf("delete Conversations role %s: %w", req.ID, err)
	}
	return infer.DeleteResponse{}, nil
}
