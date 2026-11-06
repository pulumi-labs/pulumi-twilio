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

	"github.com/pulumi/pulumi-go-provider/infer"
)

// ------------------------------------------- getSipDomainCredentialListMapping

// GetSipDomainCredentialListMapping looks up the mapping of a credential list to a SIP domain.
type GetSipDomainCredentialListMapping struct{}

// GetSipDomainCredentialListMappingArgs are the lookup arguments.
type GetSipDomainCredentialListMappingArgs struct {
	DomainSid         string  `pulumi:"domainSid"`
	CredentialListSid string  `pulumi:"credentialListSid"`
	AuthScope         *string `pulumi:"authScope,optional"`
	AccountSid        *string `pulumi:"accountSid,optional"`
}

func (a *GetSipDomainCredentialListMappingArgs) Annotate(an infer.Annotator) {
	an.Describe(&a.DomainSid, "SID (SD...) of the SIP domain.")
	an.Describe(&a.CredentialListSid, "SID (CL...) of the mapped credential list.")
	an.Describe(&a.AuthScope, "\"calls\" (the default) or \"registrations\".")
	an.Describe(&a.AccountSid, "SID of the account that owns the domain. Defaults to the provider account.")
}

// GetSipDomainCredentialListMappingResult is the looked-up mapping.
type GetSipDomainCredentialListMappingResult struct {
	Sid             string `pulumi:"sid"`
	OwnerAccountSid string `pulumi:"ownerAccountSid"`
	AuthScope       string `pulumi:"authScope"`
	FriendlyName    string `pulumi:"friendlyName"`
	DateCreated     string `pulumi:"dateCreated"`
	DateUpdated     string `pulumi:"dateUpdated"`
}

func (a *GetSipDomainCredentialListMappingResult) Annotate(an infer.Annotator) {
	an.Describe(&a.Sid, "The mapping SID, which is the SID of the mapped credential list.")
	an.Describe(&a.OwnerAccountSid, "SID of the account that owns the domain.")
	an.Describe(&a.AuthScope, "\"calls\" or \"registrations\".")
	an.Describe(&a.FriendlyName, "Name of the mapped credential list.")
	an.Describe(&a.DateCreated, "When the mapping was created (RFC 2822).")
	an.Describe(&a.DateUpdated, "When the mapping was last updated (RFC 2822).")
}

func (f *GetSipDomainCredentialListMapping) Annotate(a infer.Annotator) {
	a.Describe(&f, "Look up whether a credential list is mapped to a Twilio SIP domain, for calls or registrations. Fails when the mapping does not exist.")
}

func (f *GetSipDomainCredentialListMapping) Invoke(ctx context.Context, req infer.FunctionRequest[GetSipDomainCredentialListMappingArgs]) (infer.FunctionResponse[GetSipDomainCredentialListMappingResult], error) {
	const fn = "getSipDomainCredentialListMapping"
	type resp = infer.FunctionResponse[GetSipDomainCredentialListMappingResult]
	if err := requireArg(fn, "domainSid", req.Input.DomainSid); err != nil {
		return resp{}, err
	}
	if err := requireArg(fn, "credentialListSid", req.Input.CredentialListSid); err != nil {
		return resp{}, err
	}
	scope := authScopeCalls
	if req.Input.AuthScope != nil && *req.Input.AuthScope != "" {
		scope = *req.Input.AuthScope
	}
	if !validAuthScope(scope) {
		return resp{}, fmt.Errorf(`%s: authScope must be "calls" or "registrations", got %q`, fn, scope)
	}
	c, err := clientFor(ctx)
	if err != nil {
		return resp{}, err
	}
	acct := c.acct(req.Input.AccountSid)
	o, err := c.do(ctx, "GET", "api", sipMappingPath(acct, req.Input.DomainSid, authScopePath(scope), "CredentialListMappings", req.Input.CredentialListSid), nil, nil)
	if err != nil {
		return resp{}, lookupFailure(fn, "credential list mapping "+req.Input.CredentialListSid+" of SIP domain "+req.Input.DomainSid, err)
	}
	return resp{Output: GetSipDomainCredentialListMappingResult{
		Sid: o.str("sid"), OwnerAccountSid: acct, AuthScope: scope, FriendlyName: o.str("friendly_name"),
		DateCreated: o.str("date_created"), DateUpdated: o.str("date_updated"),
	}}, nil
}

// ---------------------------------------- getSipDomainIpAccessControlListMapping

// GetSipDomainIpAccessControlListMapping looks up the mapping of an IP access control list to a SIP domain.
type GetSipDomainIpAccessControlListMapping struct{}

// GetSipDomainIpAccessControlListMappingArgs are the lookup arguments.
type GetSipDomainIpAccessControlListMappingArgs struct {
	DomainSid              string  `pulumi:"domainSid"`
	IpAccessControlListSid string  `pulumi:"ipAccessControlListSid"`
	AccountSid             *string `pulumi:"accountSid,optional"`
}

func (a *GetSipDomainIpAccessControlListMappingArgs) Annotate(an infer.Annotator) {
	an.Describe(&a.DomainSid, "SID (SD...) of the SIP domain.")
	an.Describe(&a.IpAccessControlListSid, "SID (AL...) of the mapped IP access control list.")
	an.Describe(&a.AccountSid, "SID of the account that owns the domain. Defaults to the provider account.")
}

// GetSipDomainIpAccessControlListMappingResult is the looked-up mapping.
type GetSipDomainIpAccessControlListMappingResult struct {
	Sid             string `pulumi:"sid"`
	OwnerAccountSid string `pulumi:"ownerAccountSid"`
	FriendlyName    string `pulumi:"friendlyName"`
	DateCreated     string `pulumi:"dateCreated"`
	DateUpdated     string `pulumi:"dateUpdated"`
}

func (a *GetSipDomainIpAccessControlListMappingResult) Annotate(an infer.Annotator) {
	an.Describe(&a.Sid, "The mapping SID, which is the SID of the mapped IP access control list.")
	an.Describe(&a.OwnerAccountSid, "SID of the account that owns the domain.")
	an.Describe(&a.FriendlyName, "Name of the mapped IP access control list.")
	an.Describe(&a.DateCreated, "When the mapping was created (RFC 2822).")
	an.Describe(&a.DateUpdated, "When the mapping was last updated (RFC 2822).")
}

func (f *GetSipDomainIpAccessControlListMapping) Annotate(a infer.Annotator) {
	a.Describe(&f, "Look up whether an IP access control list is mapped to a Twilio SIP domain. Fails when the mapping does not exist.")
}

func (f *GetSipDomainIpAccessControlListMapping) Invoke(ctx context.Context, req infer.FunctionRequest[GetSipDomainIpAccessControlListMappingArgs]) (infer.FunctionResponse[GetSipDomainIpAccessControlListMappingResult], error) {
	const fn = "getSipDomainIpAccessControlListMapping"
	type resp = infer.FunctionResponse[GetSipDomainIpAccessControlListMappingResult]
	if err := requireArg(fn, "domainSid", req.Input.DomainSid); err != nil {
		return resp{}, err
	}
	if err := requireArg(fn, "ipAccessControlListSid", req.Input.IpAccessControlListSid); err != nil {
		return resp{}, err
	}
	c, err := clientFor(ctx)
	if err != nil {
		return resp{}, err
	}
	acct := c.acct(req.Input.AccountSid)
	o, err := c.do(ctx, "GET", "api", sipMappingPath(acct, req.Input.DomainSid, "Calls", "IpAccessControlListMappings", req.Input.IpAccessControlListSid), nil, nil)
	if err != nil {
		return resp{}, lookupFailure(fn, "IP access control list mapping "+req.Input.IpAccessControlListSid+" of SIP domain "+req.Input.DomainSid, err)
	}
	return resp{Output: GetSipDomainIpAccessControlListMappingResult{
		Sid: o.str("sid"), OwnerAccountSid: acct, FriendlyName: o.str("friendly_name"),
		DateCreated: o.str("date_created"), DateUpdated: o.str("date_updated"),
	}}, nil
}
