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
	"errors"
	"net/url"
	"regexp"

	"github.com/pulumi/pulumi-go-provider/infer"
)

// Lookups of the IAM v1 API that have no resource: OAuth apps cannot be read, and the permissions of a
// role are read-only.

// ------------------------------------------------------------ getOAuthApp

// GetOAuthApp exists so that a program asking for an OAuth app gets a precise answer.
type GetOAuthApp struct{}

// GetOAuthAppArgs are the lookup arguments.
type GetOAuthAppArgs struct {
	Sid string `pulumi:"sid"`
}

func (a *GetOAuthAppArgs) Annotate(an infer.Annotator) {
	an.Describe(&a.Sid, "SID (OQ...) of the OAuth app.")
}

// GetOAuthAppResult is never returned: the Twilio API cannot read an OAuth app.
type GetOAuthAppResult struct {
	Sid string `pulumi:"sid"`
}

func (a *GetOAuthAppResult) Annotate(an infer.Annotator) {
	an.Describe(&a.Sid, "SID (OQ...) of the OAuth app.")
}

func (f *GetOAuthApp) Annotate(a infer.Annotator) {
	a.Describe(&f, "Always fails with \"the Twilio API does not support reading OAuth apps\": iam.twilio.com can create and delete OAuth apps but has no call that returns one. Manage an app with the OAuthApp resource, whose outputs hold its SID.")
}

func (f *GetOAuthApp) Invoke(_ context.Context, _ infer.FunctionRequest[GetOAuthAppArgs]) (infer.FunctionResponse[GetOAuthAppResult], error) {
	return infer.FunctionResponse[GetOAuthAppResult]{}, errors.New(oauthAppNoReadMessage)
}

// ------------------------------------------------------------ getRolePermissions

var iamRoleSidRE = regexp.MustCompile(`^IX[0-9a-fA-F]{32}$`)

// GetRolePermissions lists the resolved permissions of an IAM role.
type GetRolePermissions struct{}

// GetRolePermissionsArgs are the lookup arguments.
type GetRolePermissionsArgs struct {
	RoleSid string `pulumi:"roleSid"`
}

func (a *GetRolePermissionsArgs) Annotate(an infer.Annotator) {
	an.Describe(&a.RoleSid, "SID (IX...) of the role.")
}

// RolePermission is one permission of a role.
type RolePermission struct {
	Sid                 *string `pulumi:"sid,optional"`
	Namespace           *string `pulumi:"namespace,optional"`
	Product             *string `pulumi:"product,optional"`
	Resource            *string `pulumi:"resource,optional"`
	Action              *string `pulumi:"action,optional"`
	ExternalDescription *string `pulumi:"externalDescription,optional"`
}

func (a *RolePermission) Annotate(an infer.Annotator) {
	an.Describe(&a.Sid, "SID (IZ...) of the permission.")
	an.Describe(&a.Namespace, "The namespace of the permission, for example twilio.")
	an.Describe(&a.Product, "The product the permission belongs to, for example iam.")
	an.Describe(&a.Resource, "The resource the permission applies to, for example roles.")
	an.Describe(&a.Action, "The action the permission grants, for example read.")
	an.Describe(&a.ExternalDescription, "The external description of the permission.")
}

// GetRolePermissionsResult is the looked-up permissions.
type GetRolePermissionsResult struct {
	RoleSid     string           `pulumi:"roleSid"`
	Permissions []RolePermission `pulumi:"permissions"`
}

func (a *GetRolePermissionsResult) Annotate(an infer.Annotator) {
	an.Describe(&a.RoleSid, "SID (IX...) of the role.")
	an.Describe(&a.Permissions, "The resolved permissions of the role.")
}

func (f *GetRolePermissions) Annotate(a infer.Annotator) {
	a.Describe(&f, "List the resolved permissions of an IAM role (iam.twilio.com /v1/Roles/{RoleSid}/Permissions). Read-only; every page is read.")
}

func (f *GetRolePermissions) Invoke(ctx context.Context, req infer.FunctionRequest[GetRolePermissionsArgs]) (infer.FunctionResponse[GetRolePermissionsResult], error) {
	const fn = "getRolePermissions"
	type resp = infer.FunctionResponse[GetRolePermissionsResult]
	sid := req.Input.RoleSid
	if err := requireArg(fn, "roleSid", sid); err != nil {
		return resp{}, err
	}
	if !iamRoleSidRE.MatchString(sid) {
		return resp{}, lookupFailure(fn, "roleSid "+sid, &apiError{Status: 400, Message: "roleSid must be the SID of a role (IX followed by 32 hex characters)"})
	}
	c, err := clientFor(ctx)
	if err != nil {
		return resp{}, err
	}
	out := GetRolePermissionsResult{RoleSid: sid, Permissions: []RolePermission{}}
	err = listPages(ctx, c, fn, iamProduct, "/v1/Roles/"+url.PathEscape(sid)+"/Permissions", url.Values{"PageSize": {"100"}}, "permissions", func(o obj) bool {
		out.Permissions = append(out.Permissions, RolePermission{
			Sid: nonEmpty(o.optStr("sid")), Namespace: nonEmpty(o.optStr("namespace")), Product: nonEmpty(o.optStr("product")),
			Resource: nonEmpty(o.optStr("resource")), Action: nonEmpty(o.optStr("action")),
			ExternalDescription: nonEmpty(firstStr(o, "externalDescription", "external_description")),
		})
		return true
	})
	if err != nil {
		return resp{}, lookupFailure(fn, "permissions of role "+sid, err)
	}
	return resp{Output: out}, nil
}

// firstStr returns the first of the keys the object has.
func firstStr(o obj, keys ...string) *string {
	for _, k := range keys {
		if v := o.optStr(k); v != nil {
			return v
		}
	}
	return nil
}
