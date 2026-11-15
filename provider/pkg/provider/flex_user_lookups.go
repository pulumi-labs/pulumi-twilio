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

// ------------------------------------------------------------ getFlexUser

// GetFlexUser looks up a Flex user.
type GetFlexUser struct{}

// GetFlexUserArgs are the lookup arguments.
type GetFlexUserArgs struct {
	InstanceSid string `pulumi:"instanceSid"`
	FlexUserSid string `pulumi:"flexUserSid"`
}

func (a *GetFlexUserArgs) Annotate(an infer.Annotator) {
	an.Describe(&a.InstanceSid, "SID of the Flex instance (GO...).")
	an.Describe(&a.FlexUserSid, "SID of the Flex user.")
}

// FlexUserInfo is one Flex user as Twilio reports it.
type FlexUserInfo struct {
	InstanceSid  string   `pulumi:"instanceSid"`
	FlexUserSid  string   `pulumi:"flexUserSid"`
	AccountSid   string   `pulumi:"accountSid"`
	UserSid      string   `pulumi:"userSid"`
	WorkerSid    string   `pulumi:"workerSid"`
	WorkspaceSid string   `pulumi:"workspaceSid"`
	FlexTeamSid  string   `pulumi:"flexTeamSid"`
	Username     string   `pulumi:"username"`
	Email        *string  `pulumi:"email,optional"`
	Locale       *string  `pulumi:"locale,optional"`
	Roles        []string `pulumi:"roles"`
	CreatedDate  string   `pulumi:"createdDate"`
	UpdatedDate  string   `pulumi:"updatedDate"`
	UserVersion  int      `pulumi:"userVersion"`
	Url          string   `pulumi:"url"`
}

func (a *FlexUserInfo) Annotate(an infer.Annotator) {
	an.Describe(&a.InstanceSid, "The SID of the Flex instance.")
	an.Describe(&a.FlexUserSid, "The SID of the Flex user.")
	an.Describe(&a.AccountSid, "The SID (AC...) of the account.")
	an.Describe(&a.UserSid, "The SID (US...) of the Twilio Unified User.")
	an.Describe(&a.WorkerSid, "The SID of the TaskRouter worker of the user.")
	an.Describe(&a.WorkspaceSid, "The SID (WS...) of the TaskRouter workspace.")
	an.Describe(&a.FlexTeamSid, "The SID of the Flex team of the user.")
	an.Describe(&a.Username, "The username of the user.")
	an.Describe(&a.Email, "The email of the user.")
	an.Describe(&a.Locale, "The locale preference of the user.")
	an.Describe(&a.Roles, "The roles of the user.")
	an.Describe(&a.CreatedDate, "When the user was created (ISO 8601).")
	an.Describe(&a.UpdatedDate, "When the user was last updated (ISO 8601).")
	an.Describe(&a.UserVersion, "The current version of the user.")
	an.Describe(&a.Url, "The URL of the user resource.")
}

func (f *GetFlexUser) Annotate(a infer.Annotator) {
	a.Describe(&f, "Look up a Flex user (GET /v2/Instances/{InstanceSid}/Users/{FlexUserSid} on flex-api.twilio.com).")
}

func (f *GetFlexUser) Invoke(ctx context.Context, req infer.FunctionRequest[GetFlexUserArgs]) (infer.FunctionResponse[FlexUserInfo], error) {
	const fn = "getFlexUser"
	type resp = infer.FunctionResponse[FlexUserInfo]
	in := req.Input
	if err := requireArgs(fn, [2]string{"instanceSid", in.InstanceSid}, [2]string{"flexUserSid", in.FlexUserSid}); err != nil {
		return resp{}, err
	}
	c, err := clientFor(ctx)
	if err != nil {
		return resp{}, err
	}
	o, err := c.do(ctx, "GET", flexProduct, flexUserPath(in.InstanceSid, in.FlexUserSid), nil, nil)
	if err != nil {
		return resp{}, lookupFailure(fn, "Flex user "+in.FlexUserSid, err)
	}
	st := FlexUserState{}
	st.fill(o)
	return resp{Output: FlexUserInfo{
		InstanceSid: in.InstanceSid, FlexUserSid: in.FlexUserSid, AccountSid: st.AccountSid, UserSid: o.str("user_sid"),
		WorkerSid: st.WorkerSid, WorkspaceSid: st.WorkspaceSid, FlexTeamSid: st.FlexTeamSid, Username: st.Username,
		Email: nonEmpty(o.optStr("email")), Locale: nonEmpty(o.optStr("locale")), Roles: st.Roles,
		CreatedDate: st.CreatedDate, UpdatedDate: st.UpdatedDate, UserVersion: st.UserVersion, Url: st.Url,
	}}, nil
}

// ------------------------------------------------------------ getFrontlineUser

// GetFrontlineUser looks up a Frontline user by SID.
type GetFrontlineUser struct{}

// GetFrontlineUserArgs are the lookup arguments.
type GetFrontlineUserArgs struct {
	Sid string `pulumi:"sid"`
}

func (a *GetFrontlineUserArgs) Annotate(an infer.Annotator) {
	an.Describe(&a.Sid, "SID of the Frontline user.")
}

// FrontlineUserInfo is one Frontline user as Twilio reports it.
type FrontlineUserInfo struct {
	Sid          string  `pulumi:"sid"`
	Identity     string  `pulumi:"identity"`
	FriendlyName *string `pulumi:"friendlyName,optional"`
	Avatar       *string `pulumi:"avatar,optional"`
	State        *string `pulumi:"state,optional"`
	IsAvailable  *bool   `pulumi:"isAvailable,optional"`
	Url          string  `pulumi:"url"`
}

func (a *FrontlineUserInfo) Annotate(an infer.Annotator) {
	an.Describe(&a.Sid, "The SID of the Frontline user.")
	an.Describe(&a.Identity, "The application-defined string that identifies the user.")
	an.Describe(&a.FriendlyName, "The name of the user.")
	an.Describe(&a.Avatar, "The URL of the avatar shown in the Frontline app.")
	an.Describe(&a.State, "The state of the user: active or deactivated.")
	an.Describe(&a.IsAvailable, "Whether the user is available for new conversations.")
	an.Describe(&a.Url, "The URL of the user resource.")
}

func (f *GetFrontlineUser) Annotate(a infer.Annotator) {
	a.Describe(&f, "Look up a Frontline user by SID (GET /v1/Users/{Sid} on frontline-api.twilio.com).")
}

func (f *GetFrontlineUser) Invoke(ctx context.Context, req infer.FunctionRequest[GetFrontlineUserArgs]) (infer.FunctionResponse[FrontlineUserInfo], error) {
	const fn = "getFrontlineUser"
	type resp = infer.FunctionResponse[FrontlineUserInfo]
	if err := requireArg(fn, "sid", req.Input.Sid); err != nil {
		return resp{}, err
	}
	c, err := clientFor(ctx)
	if err != nil {
		return resp{}, err
	}
	o, err := c.do(ctx, "GET", frontlineProduct, frontlineUserPath(req.Input.Sid), nil, nil)
	if err != nil {
		return resp{}, lookupFailure(fn, "Frontline user "+req.Input.Sid, err)
	}
	return resp{Output: FrontlineUserInfo{
		Sid: req.Input.Sid, Identity: o.str("identity"), FriendlyName: nonEmpty(o.optStr("friendly_name")), Avatar: nonEmpty(o.optStr("avatar")),
		State: nonEmpty(o.optStr("state")), IsAvailable: o.optBool("is_available"), Url: o.str("url"),
	}}, nil
}
