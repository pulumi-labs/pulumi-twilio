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
	"reflect"
	"regexp"

	"github.com/pulumi/pulumi-go-provider/infer"
)

// Flex v2 users (flex-api.twilio.com /v2/Instances/{InstanceSid}/Users/{FlexUserSid}) and Frontline
// users (frontline-api.twilio.com /v1/Users/{Sid}) have a fetch and an update and no create or
// delete in twilio-oai 2.11.0 (spec/json/twilio_flex_v2.json, twilio_frontline_v1.json). Twilio
// creates both when a person is invited in the Console, so the resources adopt an existing user,
// manage the settings the spec lets an update change, and never delete it on destroy.

const frontlineProduct = "frontline-api"

var flexUserSidRE = regexp.MustCompile(`^US[0-9a-fA-F]{32}$`)

func flexUserPath(instance, user string) string {
	return "/v2/Instances/" + url.PathEscape(instance) + "/Users/" + url.PathEscape(user)
}

func frontlineUserPath(sid string) string { return "/v1/Users/" + url.PathEscape(sid) }

// ------------------------------------------------------------ FlexUser

// FlexUser adopts a Flex user and manages its email, unified user link and locale. It never
// creates or deletes a Flex user.
type FlexUser struct{}

// FlexUserArgs are the inputs of a FlexUser.
type FlexUserArgs struct {
	InstanceSid string  `pulumi:"instanceSid"`
	FlexUserSid string  `pulumi:"flexUserSid"`
	Email       *string `pulumi:"email,optional"`
	UserSid     *string `pulumi:"userSid,optional"`
	Locale      *string `pulumi:"locale,optional"`
}

func (a *FlexUserArgs) Annotate(an infer.Annotator) {
	an.Describe(&a.InstanceSid, "The SID of the Flex instance (GO...) the user belongs to. Changing it adopts a different user.")
	an.Describe(&a.FlexUserSid, "The SID of the Flex user. Twilio creates Flex users when a person is invited, so this resource adopts an existing user and cannot create one. Changing it adopts a different user.")
	an.Describe(&a.Email, "The email of the user. Twilio documents no default, so an import does not adopt it. Removing it leaves the setting unchanged.")
	an.Describe(&a.UserSid, "The SID (US...) of the Twilio Unified User the Flex user is linked to. Twilio documents no default, so an import does not adopt it. Removing it leaves the setting unchanged.")
	an.Describe(&a.Locale, "The locale preference of the user, for example en-US. Twilio documents no default, so an import does not adopt it. Removing it leaves the setting unchanged.")
}

// FlexUserState is the persisted state of a FlexUser.
type FlexUserState struct {
	FlexUserArgs

	AccountSid   string   `pulumi:"accountSid"`
	WorkerSid    string   `pulumi:"workerSid"`
	WorkspaceSid string   `pulumi:"workspaceSid"`
	FlexTeamSid  string   `pulumi:"flexTeamSid"`
	Username     string   `pulumi:"username"`
	Roles        []string `pulumi:"roles"`
	CreatedDate  string   `pulumi:"createdDate"`
	UpdatedDate  string   `pulumi:"updatedDate"`
	UserVersion  int      `pulumi:"userVersion"`
	Url          string   `pulumi:"url"`
}

func (s *FlexUserState) Annotate(a infer.Annotator) {
	a.Describe(&s.AccountSid, "The SID (AC...) of the account.")
	a.Describe(&s.WorkerSid, "The SID of the TaskRouter worker of the user.")
	a.Describe(&s.WorkspaceSid, "The SID (WS...) of the TaskRouter workspace.")
	a.Describe(&s.FlexTeamSid, "The SID of the Flex team of the user.")
	a.Describe(&s.Username, "The username of the user.")
	a.Describe(&s.Roles, "The roles of the user.")
	a.Describe(&s.CreatedDate, "When the user was created (ISO 8601).")
	a.Describe(&s.UpdatedDate, "When the user was last updated (ISO 8601).")
	a.Describe(&s.UserVersion, "The current version of the user, as Twilio's version field.")
	a.Describe(&s.Url, "The URL of the user resource.")
}

func (r *FlexUser) Annotate(a infer.Annotator) {
	a.Describe(&r, "Adopts an existing Flex user (the resource ID is <instanceSid>/<flexUserSid>) and manages its email, userSid and locale. Twilio offers no create or delete, so destroying it forgets it and leaves the user as it is. Unverified against a live account.")
}

var _ infer.CustomResource[FlexUserArgs, FlexUserState] = (*FlexUser)(nil)
var _ infer.CustomCheck[FlexUserArgs] = (*FlexUser)(nil)
var _ infer.CustomDiff[FlexUserArgs, FlexUserState] = (*FlexUser)(nil)
var _ infer.CustomRead[FlexUserArgs, FlexUserState] = (*FlexUser)(nil)
var _ infer.CustomUpdate[FlexUserArgs, FlexUserState] = (*FlexUser)(nil)
var _ infer.CustomDelete[FlexUserState] = (*FlexUser)(nil)

func (r *FlexUser) Check(ctx context.Context, req infer.CheckRequest) (infer.CheckResponse[FlexUserArgs], error) {
	args, failures, err := infer.DefaultCheck[FlexUserArgs](ctx, req.NewInputs)
	if err != nil || len(failures) > 0 {
		return infer.CheckResponse[FlexUserArgs]{Inputs: args, Failures: failures}, err
	}
	c := &checker{unk: unknowns(req.NewInputs)}
	c.required("instanceSid", args.InstanceSid)
	c.required("flexUserSid", args.FlexUserSid)
	if v := args.UserSid; v != nil {
		c.sid("userSid", *v, flexUserSidRE, "Twilio Unified User (US followed by 32 hex characters)")
	}
	return infer.CheckResponse[FlexUserArgs]{Inputs: args, Failures: c.failures}, nil
}

func (r *FlexUser) Diff(_ context.Context, req infer.DiffRequest[FlexUserArgs, FlexUserState]) (infer.DiffResponse, error) {
	o, n := req.State.FlexUserArgs, req.Inputs
	d := trDiffSet{}
	d.replace("instanceSid", o.InstanceSid, n.InstanceSid)
	d.replace("flexUserSid", o.FlexUserSid, n.FlexUserSid)
	d.str("email", o.Email, n.Email)
	d.str("userSid", o.UserSid, n.UserSid)
	d.str("locale", o.Locale, n.Locale)
	return d.response(), nil
}

func (a FlexUserArgs) form() url.Values {
	form := url.Values{}
	setS(form, "Email", a.Email)
	setS(form, "UserSid", a.UserSid)
	setS(form, "Locale", a.Locale)
	return form
}

func (s *FlexUserState) fill(o obj) {
	s.AccountSid = o.str("account_sid")
	s.WorkerSid = o.str("worker_sid")
	s.WorkspaceSid = o.str("workspace_sid")
	s.FlexTeamSid = o.str("flex_team_sid")
	s.Username = o.str("username")
	s.Roles = []string{}
	if l := o.optList("roles"); l != nil {
		s.Roles = *l
	}
	s.CreatedDate = o.str("created_date")
	s.UpdatedDate = o.str("updated_date")
	if v := o.optInt("version"); v != nil {
		s.UserVersion = *v
	}
	s.Url = o.str("url")
}

// apply reads the user (so a missing one is a clear error), writes the stated settings and returns it.
func (a FlexUserArgs) apply(ctx context.Context, verb string) (obj, error) {
	c, err := clientFor(ctx)
	if err != nil {
		return nil, err
	}
	path := flexUserPath(a.InstanceSid, a.FlexUserSid)
	o, err := c.do(ctx, "GET", flexProduct, path, nil, nil)
	if err != nil {
		if isNotFound(err) {
			return nil, fmt.Errorf("%s Flex user: instance %s has no Flex user %s (this resource adopts an existing user, it cannot create one)", verb, a.InstanceSid, a.FlexUserSid)
		}
		return nil, fmt.Errorf("%s Flex user: %w", verb, err)
	}
	if form := a.form(); len(form) > 0 {
		if o, err = c.do(ctx, "POST", flexProduct, path, nil, form); err != nil {
			return nil, fmt.Errorf("%s Flex user: %w", verb, err)
		}
	}
	return o, nil
}

func (r *FlexUser) Create(ctx context.Context, req infer.CreateRequest[FlexUserArgs]) (infer.CreateResponse[FlexUserState], error) {
	type resp = infer.CreateResponse[FlexUserState]
	st := FlexUserState{FlexUserArgs: req.Inputs}
	if req.DryRun {
		return resp{Output: st}, nil
	}
	o, err := req.Inputs.apply(ctx, "adopt")
	if err != nil {
		return resp{}, err
	}
	st.fill(o)
	return resp{ID: req.Inputs.InstanceSid + "/" + req.Inputs.FlexUserSid, Output: st}, nil
}

func (r *FlexUser) Read(ctx context.Context, req infer.ReadRequest[FlexUserArgs, FlexUserState]) (infer.ReadResponse[FlexUserArgs, FlexUserState], error) {
	type resp = infer.ReadResponse[FlexUserArgs, FlexUserState]
	parts, err := splitVerifyID(req.ID, 2, "<instanceSid>/<flexUserSid>")
	if err != nil {
		return resp{}, err
	}
	c, err := clientFor(ctx)
	if err != nil {
		return resp{}, err
	}
	o, err := c.do(ctx, "GET", flexProduct, flexUserPath(parts[0], parts[1]), nil, nil)
	if err != nil {
		if isNotFound(err) {
			if err := notFoundOnImport("Flex user", req.ID, req.Inputs, req.State); err != nil {
				return resp{}, err
			}
			return resp{}, nil
		}
		return resp{}, fmt.Errorf("read Flex user: %w", err)
	}
	in := req.Inputs
	in.InstanceSid, in.FlexUserSid = parts[0], parts[1]
	imported := reflect.ValueOf(req.State).IsZero()
	in.Email = flexKeepStr(in.Email, o.optStr("email"), imported)
	in.UserSid = flexKeepStr(in.UserSid, o.optStr("user_sid"), imported)
	in.Locale = flexKeepStr(in.Locale, o.optStr("locale"), imported)
	st := FlexUserState{FlexUserArgs: in}
	st.fill(o)
	return resp{ID: req.ID, Inputs: in, State: st}, nil
}

func (r *FlexUser) Update(ctx context.Context, req infer.UpdateRequest[FlexUserArgs, FlexUserState]) (infer.UpdateResponse[FlexUserState], error) {
	type resp = infer.UpdateResponse[FlexUserState]
	st := req.State
	st.FlexUserArgs = req.Inputs
	if req.DryRun {
		return resp{Output: st}, nil
	}
	o, err := req.Inputs.apply(ctx, "update")
	if err != nil {
		return resp{}, err
	}
	st.fill(o)
	return resp{Output: st}, nil
}

// Delete forgets the user: the spec offers no way to remove one.
func (r *FlexUser) Delete(_ context.Context, _ infer.DeleteRequest[FlexUserState]) (infer.DeleteResponse, error) {
	return infer.DeleteResponse{}, nil
}

// ------------------------------------------------------------ FrontlineUser

// FrontlineUser adopts a Frontline user and manages its name, avatar, state and availability. It
// never creates or deletes a Frontline user.
type FrontlineUser struct{}

var frontlineUserStates = []string{"active", "deactivated"}

// FrontlineUserArgs are the inputs of a FrontlineUser.
type FrontlineUserArgs struct {
	Sid          string  `pulumi:"sid"`
	FriendlyName *string `pulumi:"friendlyName,optional"`
	Avatar       *string `pulumi:"avatar,optional"`
	State        *string `pulumi:"state,optional"`
	IsAvailable  *bool   `pulumi:"isAvailable,optional"`
}

func (a *FrontlineUserArgs) Annotate(an infer.Annotator) {
	an.Describe(&a.Sid, "The SID of the Frontline user. Twilio creates Frontline users when a person is invited, so this resource adopts an existing user and cannot create one. Changing it adopts a different user.")
	an.Describe(&a.FriendlyName, "A descriptive name for the user. Removing it leaves the name unchanged.")
	an.Describe(&a.Avatar, "The URL of the avatar shown in the Frontline app. Removing it leaves the avatar unchanged.")
	an.Describe(&a.State, "The state of the user: active (Twilio default) or deactivated. Removing it leaves the state unchanged.")
	an.Describe(&a.IsAvailable, "Whether the user is available for new conversations (Twilio default false for new users). Set it to false to stop new inbound conversations when Pool Routing is used. Removing it leaves the setting unchanged.")
}

// FrontlineUserState is the persisted state of a FrontlineUser.
type FrontlineUserState struct {
	FrontlineUserArgs

	Identity string `pulumi:"identity"`
	Url      string `pulumi:"url"`
}

func (s *FrontlineUserState) Annotate(a infer.Annotator) {
	a.Describe(&s.Identity, "The application-defined string that identifies the user, often a username or an email address (case-sensitive).")
	a.Describe(&s.Url, "The URL of the user resource.")
}

func (r *FrontlineUser) Annotate(a infer.Annotator) {
	a.Describe(&r, "Adopts an existing Frontline user (the resource ID is its SID) and manages its name, avatar, state and availability. Twilio offers no create or delete, so destroying it forgets it and leaves the user as it is. Unverified against a live account.")
}

var _ infer.CustomResource[FrontlineUserArgs, FrontlineUserState] = (*FrontlineUser)(nil)
var _ infer.CustomCheck[FrontlineUserArgs] = (*FrontlineUser)(nil)
var _ infer.CustomDiff[FrontlineUserArgs, FrontlineUserState] = (*FrontlineUser)(nil)
var _ infer.CustomRead[FrontlineUserArgs, FrontlineUserState] = (*FrontlineUser)(nil)
var _ infer.CustomUpdate[FrontlineUserArgs, FrontlineUserState] = (*FrontlineUser)(nil)
var _ infer.CustomDelete[FrontlineUserState] = (*FrontlineUser)(nil)

// frontlineUserDefaults are the Twilio defaults of this resource, shared by Read (import adoption) and Diff.
var frontlineUserDefaults = trDefaults{
	"state":       "active",
	"isAvailable": false,
}

func (r *FrontlineUser) Check(ctx context.Context, req infer.CheckRequest) (infer.CheckResponse[FrontlineUserArgs], error) {
	args, failures, err := infer.DefaultCheck[FrontlineUserArgs](ctx, req.NewInputs)
	if err != nil || len(failures) > 0 {
		return infer.CheckResponse[FrontlineUserArgs]{Inputs: args, Failures: failures}, err
	}
	c := &checker{unk: unknowns(req.NewInputs)}
	c.required("sid", args.Sid)
	if v := args.State; v != nil && !c.unk("state") {
		c.oneOf("state", *v, frontlineUserStates)
	}
	return infer.CheckResponse[FrontlineUserArgs]{Inputs: args, Failures: c.failures}, nil
}

func (r *FrontlineUser) Diff(_ context.Context, req infer.DiffRequest[FrontlineUserArgs, FrontlineUserState]) (infer.DiffResponse, error) {
	o, n := req.State.FrontlineUserArgs, req.Inputs
	d := trDiffSet{}
	d.replace("sid", o.Sid, n.Sid)
	d.str("friendlyName", o.FriendlyName, n.FriendlyName)
	d.str("avatar", o.Avatar, n.Avatar)
	d.strDef(frontlineUserDefaults, "state", o.State, n.State)
	d.boolDef(frontlineUserDefaults, "isAvailable", o.IsAvailable, n.IsAvailable)
	return d.response(), nil
}

func (a FrontlineUserArgs) form() url.Values {
	form := url.Values{}
	setS(form, "FriendlyName", a.FriendlyName)
	setS(form, "Avatar", a.Avatar)
	setS(form, "State", a.State)
	setB(form, "IsAvailable", a.IsAvailable)
	return form
}

func (a FrontlineUserArgs) apply(ctx context.Context, verb string) (obj, error) {
	c, err := clientFor(ctx)
	if err != nil {
		return nil, err
	}
	path := frontlineUserPath(a.Sid)
	o, err := c.do(ctx, "GET", frontlineProduct, path, nil, nil)
	if err != nil {
		if isNotFound(err) {
			return nil, fmt.Errorf("%s Frontline user: there is no Frontline user %s (this resource adopts an existing user, it cannot create one)", verb, a.Sid)
		}
		return nil, fmt.Errorf("%s Frontline user: %w", verb, err)
	}
	if form := a.form(); len(form) > 0 {
		if o, err = c.do(ctx, "POST", frontlineProduct, path, nil, form); err != nil {
			return nil, fmt.Errorf("%s Frontline user: %w", verb, err)
		}
	}
	return o, nil
}

func (r *FrontlineUser) Create(ctx context.Context, req infer.CreateRequest[FrontlineUserArgs]) (infer.CreateResponse[FrontlineUserState], error) {
	type resp = infer.CreateResponse[FrontlineUserState]
	st := FrontlineUserState{FrontlineUserArgs: req.Inputs}
	if req.DryRun {
		return resp{Output: st}, nil
	}
	o, err := req.Inputs.apply(ctx, "adopt")
	if err != nil {
		return resp{}, err
	}
	st.Identity, st.Url = o.str("identity"), o.str("url")
	return resp{ID: req.Inputs.Sid, Output: st}, nil
}

func (r *FrontlineUser) Read(ctx context.Context, req infer.ReadRequest[FrontlineUserArgs, FrontlineUserState]) (infer.ReadResponse[FrontlineUserArgs, FrontlineUserState], error) {
	type resp = infer.ReadResponse[FrontlineUserArgs, FrontlineUserState]
	c, err := clientFor(ctx)
	if err != nil {
		return resp{}, err
	}
	o, err := c.do(ctx, "GET", frontlineProduct, frontlineUserPath(req.ID), nil, nil)
	if err != nil {
		if isNotFound(err) {
			if err := notFoundOnImport("Frontline user", req.ID, req.Inputs, req.State); err != nil {
				return resp{}, err
			}
			return resp{}, nil
		}
		return resp{}, fmt.Errorf("read Frontline user: %w", err)
	}
	in := req.Inputs
	in.Sid = req.ID
	imported := reflect.ValueOf(req.State).IsZero()
	in.FriendlyName = flexKeepStr(in.FriendlyName, o.optStr("friendly_name"), imported)
	in.Avatar = flexKeepStr(in.Avatar, o.optStr("avatar"), imported)
	in.State = trStr(in.State, o.optStr("state"), frontlineUserDefaults.str("state"), imported)
	in.IsAvailable = trBool(in.IsAvailable, o.optBool("is_available"), frontlineUserDefaults.boolean("isAvailable"), imported)
	st := FrontlineUserState{FrontlineUserArgs: in, Identity: o.str("identity"), Url: o.str("url")}
	return resp{ID: req.ID, Inputs: in, State: st}, nil
}

func (r *FrontlineUser) Update(ctx context.Context, req infer.UpdateRequest[FrontlineUserArgs, FrontlineUserState]) (infer.UpdateResponse[FrontlineUserState], error) {
	type resp = infer.UpdateResponse[FrontlineUserState]
	st := req.State
	st.FrontlineUserArgs = req.Inputs
	if req.DryRun {
		return resp{Output: st}, nil
	}
	o, err := req.Inputs.apply(ctx, "update")
	if err != nil {
		return resp{}, err
	}
	st.Identity, st.Url = o.str("identity"), o.str("url")
	return resp{Output: st}, nil
}

// Delete forgets the user: the spec offers no way to remove one.
func (r *FrontlineUser) Delete(_ context.Context, _ infer.DeleteRequest[FrontlineUserState]) (infer.DeleteResponse, error) {
	return infer.DeleteResponse{}, nil
}
