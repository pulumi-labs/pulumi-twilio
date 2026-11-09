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

	p "github.com/pulumi/pulumi-go-provider"
	"github.com/pulumi/pulumi-go-provider/infer"
)

// Twilio authenticates SIP calls to a domain against the credential lists and IP access control
// lists mapped to it, and SIP registrations against mapped credential lists. Each mapping is a
// create-and-delete object (the API has no update), and its SID is the SID of the mapped list.
// The same mappings are also served under the legacy Domains/{sid}/CredentialListMappings and
// IpAccessControlListMappings paths, which are aliases of the "calls" mappings and are not used here.

const (
	authScopeCalls         = "calls"
	authScopeRegistrations = "registrations"
)

// authScopePath is the path segment under Auth for a scope.
func authScopePath(scope string) string {
	if scope == authScopeRegistrations {
		return "Registrations"
	}
	return "Calls"
}

func validAuthScope(s string) bool { return s == authScopeCalls || s == authScopeRegistrations }

// sipMappingPath is the collection (sid empty) or instance path of a mapping of a SIP domain.
// kind is CredentialListMappings or IpAccessControlListMappings; scope is "Calls" or "Registrations".
func sipMappingPath(account, domain, scope, kind, sid string) string {
	return sipPath(account, "Domains/"+url.PathEscape(domain)+"/Auth/"+scope+"/"+kind, sid)
}

// ------------------------------------------------ SipDomainCredentialListMapping

// SipDomainCredentialListMapping maps a SIP credential list to a SIP domain so the domain
// authenticates calls or registrations against it.
type SipDomainCredentialListMapping struct{}

// SipDomainCredentialListMappingArgs are the inputs of a SipDomainCredentialListMapping.
type SipDomainCredentialListMappingArgs struct {
	DomainSid         string  `pulumi:"domainSid" provider:"replaceOnChanges"`
	CredentialListSid string  `pulumi:"credentialListSid" provider:"replaceOnChanges"`
	AuthScope         *string `pulumi:"authScope,optional" provider:"replaceOnChanges"`
	AccountSid        *string `pulumi:"accountSid,optional" provider:"replaceOnChanges"`
}

func (a *SipDomainCredentialListMappingArgs) Annotate(an infer.Annotator) {
	an.Describe(&a.DomainSid, "SID (SD...) of the SIP domain. Changing it replaces the resource.")
	an.Describe(&a.CredentialListSid, "SID (CL...) of the credential list to map. Changing it replaces the resource.")
	an.Describe(&a.AuthScope, "What the credential list authenticates: \"calls\" (incoming calls, the default) or \"registrations\" (SIP registrations). "+
		"A list can be mapped to both scopes with two resources. Changing it replaces the resource.")
	an.Describe(&a.AccountSid, "SID of the account or subaccount that owns the domain. Defaults to the provider account. "+
		"Changing it replaces the resource. To import a mapping owned by a subaccount use the ID <accountSid>/<domainSid>/<authScope>/<sid>.")
	an.SetDefault(&a.AuthScope, authScopeCalls)
}

// SipDomainCredentialListMappingState is the persisted state of a SipDomainCredentialListMapping.
type SipDomainCredentialListMappingState struct {
	SipDomainCredentialListMappingArgs

	// Sid is the mapping SID, which equals the SID of the mapped credential list.
	Sid string `pulumi:"sid"`
	// OwnerAccountSid is the account that owns the domain.
	OwnerAccountSid string `pulumi:"ownerAccountSid"`
	// FriendlyName is the name of the mapped credential list.
	FriendlyName string `pulumi:"friendlyName"`
	// DateCreated is the RFC 2822 creation time.
	DateCreated string `pulumi:"dateCreated"`
	// DateUpdated is the RFC 2822 time of the last update.
	DateUpdated string `pulumi:"dateUpdated"`
}

func (s *SipDomainCredentialListMappingState) Annotate(a infer.Annotator) {
	a.Describe(&s.Sid, "The mapping SID, which is the SID of the mapped credential list (CL...).")
	a.Describe(&s.OwnerAccountSid, "SID of the account that owns the domain.")
	a.Describe(&s.FriendlyName, "Name of the mapped credential list.")
	a.Describe(&s.DateCreated, "When the mapping was created (RFC 2822).")
	a.Describe(&s.DateUpdated, "When the mapping was last updated (RFC 2822).")
}

func (r *SipDomainCredentialListMapping) Annotate(a infer.Annotator) {
	a.Describe(&r, "A credential list mapped to a Twilio SIP domain for call or registration authentication. Twilio has no update call, so every input replaces the mapping. "+
		"The resource ID is <domainSid>/<authScope>/<sid> (<accountSid>/<domainSid>/<authScope>/<sid> for a subaccount).")
}

var _ infer.CustomResource[SipDomainCredentialListMappingArgs, SipDomainCredentialListMappingState] = (*SipDomainCredentialListMapping)(nil)
var _ infer.CustomCheck[SipDomainCredentialListMappingArgs] = (*SipDomainCredentialListMapping)(nil)
var _ infer.CustomRead[SipDomainCredentialListMappingArgs, SipDomainCredentialListMappingState] = (*SipDomainCredentialListMapping)(nil)
var _ infer.CustomDelete[SipDomainCredentialListMappingState] = (*SipDomainCredentialListMapping)(nil)

func (a SipDomainCredentialListMappingArgs) scope() string {
	if a.AuthScope == nil || *a.AuthScope == "" {
		return authScopeCalls
	}
	return *a.AuthScope
}

func (r *SipDomainCredentialListMapping) Check(ctx context.Context, req infer.CheckRequest) (infer.CheckResponse[SipDomainCredentialListMappingArgs], error) {
	args, failures, err := infer.DefaultCheck[SipDomainCredentialListMappingArgs](ctx, req.NewInputs)
	if err != nil || len(failures) > 0 {
		return infer.CheckResponse[SipDomainCredentialListMappingArgs]{Inputs: args, Failures: failures}, err
	}
	unk := unknowns(req.NewInputs)
	if !unk("domainSid") && args.DomainSid == "" {
		failures = append(failures, p.CheckFailure{Property: "domainSid", Reason: "domainSid must be set"})
	}
	if !unk("credentialListSid") && args.CredentialListSid == "" {
		failures = append(failures, p.CheckFailure{Property: "credentialListSid", Reason: "credentialListSid must be set"})
	}
	if !unk("authScope") {
		if args.AuthScope != nil && !validAuthScope(*args.AuthScope) {
			failures = append(failures, p.CheckFailure{Property: "authScope", Reason: `authScope must be "calls" or "registrations"`})
		}
	}
	return infer.CheckResponse[SipDomainCredentialListMappingArgs]{Inputs: args, Failures: failures}, nil
}

// credMappingID is <domainSid>/<authScope>/<sid>, prefixed with the account for a subaccount.
func credMappingID(c *Client, acct, domain, scope, sid string) string {
	id := domain + "/" + scope + "/" + sid
	if acct == "" || acct == c.accountSid {
		return id
	}
	return acct + "/" + id
}

// splitCredMappingID parses a credential list mapping ID into its owning account (empty when the
// ID does not name one), domain, scope and SID.
func splitCredMappingID(id string) (acct, domain, scope, sid string, err error) {
	parts := strings.Split(id, "/")
	if len(parts) == 4 {
		acct, parts = parts[0], parts[1:]
	}
	if len(parts) == 3 && validAuthScope(parts[1]) && parts[0] != "" && parts[2] != "" {
		return acct, parts[0], parts[1], parts[2], nil
	}
	return "", "", "", "", fmt.Errorf("invalid ID %q: expected <domainSid>/<calls|registrations>/<sid> or <accountSid>/<domainSid>/<calls|registrations>/<sid>", id)
}

func (r *SipDomainCredentialListMapping) Create(ctx context.Context, req infer.CreateRequest[SipDomainCredentialListMappingArgs]) (infer.CreateResponse[SipDomainCredentialListMappingState], error) {
	a := req.Inputs
	if req.DryRun {
		return infer.CreateResponse[SipDomainCredentialListMappingState]{ID: "", Output: SipDomainCredentialListMappingState{SipDomainCredentialListMappingArgs: a}}, nil
	}
	c, err := clientFor(ctx)
	if err != nil {
		return infer.CreateResponse[SipDomainCredentialListMappingState]{}, err
	}
	acct, scope := c.acct(a.AccountSid), a.scope()
	o, err := c.do(ctx, "POST", "api", sipMappingPath(acct, a.DomainSid, authScopePath(scope), "CredentialListMappings", ""), nil,
		url.Values{"CredentialListSid": {a.CredentialListSid}})
	if err != nil {
		return infer.CreateResponse[SipDomainCredentialListMappingState]{}, fmt.Errorf("create SIP domain credential list mapping: %w", err)
	}
	sid := o.str("sid")
	return infer.CreateResponse[SipDomainCredentialListMappingState]{ID: credMappingID(c, acct, a.DomainSid, scope, sid), Output: SipDomainCredentialListMappingState{
		SipDomainCredentialListMappingArgs: a, Sid: sid, OwnerAccountSid: acct, FriendlyName: o.str("friendly_name"),
		DateCreated: o.str("date_created"), DateUpdated: o.str("date_updated"),
	}}, nil
}

func (r *SipDomainCredentialListMapping) Read(ctx context.Context, req infer.ReadRequest[SipDomainCredentialListMappingArgs, SipDomainCredentialListMappingState]) (infer.ReadResponse[SipDomainCredentialListMappingArgs, SipDomainCredentialListMappingState], error) {
	type resp = infer.ReadResponse[SipDomainCredentialListMappingArgs, SipDomainCredentialListMappingState]
	c, err := clientFor(ctx)
	if err != nil {
		return resp{}, err
	}
	idAcct, domain, scope, sid, err := splitCredMappingID(req.ID)
	if err != nil {
		return resp{}, err
	}
	acct := c.accountSid
	switch {
	case req.State.OwnerAccountSid != "":
		acct = req.State.OwnerAccountSid
	case idAcct != "":
		acct = idAcct
	}
	o, err := c.do(ctx, "GET", "api", sipMappingPath(acct, domain, authScopePath(scope), "CredentialListMappings", sid), nil, nil)
	if err != nil {
		if isNotFound(err) {
			if err := notFoundOnImport("sip domain credential list mapping", req.ID, req.Inputs, req.State); err != nil {
				return resp{}, err
			}
			return resp{}, nil
		}
		return resp{}, fmt.Errorf("read SIP domain credential list mapping %s: %w", req.ID, err)
	}
	in := req.Inputs
	if in.DomainSid == "" { // import: the ID is all there is
		in.DomainSid, in.CredentialListSid = domain, sid
		s := scope
		in.AuthScope = &s
		if acct != c.accountSid {
			in.AccountSid = &acct
		}
	}
	st := SipDomainCredentialListMappingState{
		SipDomainCredentialListMappingArgs: in, Sid: sid, OwnerAccountSid: acct, FriendlyName: o.str("friendly_name"),
		DateCreated: o.str("date_created"), DateUpdated: o.str("date_updated"),
	}
	return resp{ID: req.ID, Inputs: in, State: st}, nil
}

func (r *SipDomainCredentialListMapping) Delete(ctx context.Context, req infer.DeleteRequest[SipDomainCredentialListMappingState]) (infer.DeleteResponse, error) {
	c, err := clientFor(ctx)
	if err != nil {
		return infer.DeleteResponse{}, err
	}
	idAcct, domain, scope, sid, err := splitCredMappingID(req.ID)
	if err != nil {
		return infer.DeleteResponse{}, err
	}
	acct := req.State.OwnerAccountSid
	if acct == "" {
		acct = idAcct
	}
	if acct == "" {
		acct = c.accountSid
	}
	if _, err := c.do(ctx, "DELETE", "api", sipMappingPath(acct, domain, authScopePath(scope), "CredentialListMappings", sid), nil, nil); err != nil && !isNotFound(err) {
		return infer.DeleteResponse{}, fmt.Errorf("delete SIP domain credential list mapping %s: %w", req.ID, err)
	}
	return infer.DeleteResponse{}, nil
}

// ------------------------------------------ SipDomainIpAccessControlListMapping

// SipDomainIpAccessControlListMapping maps a SIP IP access control list to a SIP domain so the
// domain accepts calls from the addresses in the list.
type SipDomainIpAccessControlListMapping struct{}

// SipDomainIpAccessControlListMappingArgs are the inputs of a SipDomainIpAccessControlListMapping.
type SipDomainIpAccessControlListMappingArgs struct {
	DomainSid              string  `pulumi:"domainSid" provider:"replaceOnChanges"`
	IpAccessControlListSid string  `pulumi:"ipAccessControlListSid" provider:"replaceOnChanges"`
	AccountSid             *string `pulumi:"accountSid,optional" provider:"replaceOnChanges"`
}

func (a *SipDomainIpAccessControlListMappingArgs) Annotate(an infer.Annotator) {
	an.Describe(&a.DomainSid, "SID (SD...) of the SIP domain. Changing it replaces the resource.")
	an.Describe(&a.IpAccessControlListSid, "SID (AL...) of the IP access control list to map. Changing it replaces the resource.")
	an.Describe(&a.AccountSid, "SID of the account or subaccount that owns the domain. Defaults to the provider account. "+
		"Changing it replaces the resource. To import a mapping owned by a subaccount use the ID <accountSid>/<domainSid>/<sid>.")
}

// SipDomainIpAccessControlListMappingState is the persisted state of a SipDomainIpAccessControlListMapping.
type SipDomainIpAccessControlListMappingState struct {
	SipDomainIpAccessControlListMappingArgs

	// Sid is the mapping SID, which equals the SID of the mapped access control list.
	Sid string `pulumi:"sid"`
	// OwnerAccountSid is the account that owns the domain.
	OwnerAccountSid string `pulumi:"ownerAccountSid"`
	// FriendlyName is the name of the mapped access control list.
	FriendlyName string `pulumi:"friendlyName"`
	// DateCreated is the RFC 2822 creation time.
	DateCreated string `pulumi:"dateCreated"`
	// DateUpdated is the RFC 2822 time of the last update.
	DateUpdated string `pulumi:"dateUpdated"`
}

func (s *SipDomainIpAccessControlListMappingState) Annotate(a infer.Annotator) {
	a.Describe(&s.Sid, "The mapping SID, which is the SID of the mapped IP access control list (AL...).")
	a.Describe(&s.OwnerAccountSid, "SID of the account that owns the domain.")
	a.Describe(&s.FriendlyName, "Name of the mapped IP access control list.")
	a.Describe(&s.DateCreated, "When the mapping was created (RFC 2822).")
	a.Describe(&s.DateUpdated, "When the mapping was last updated (RFC 2822).")
}

func (r *SipDomainIpAccessControlListMapping) Annotate(a infer.Annotator) {
	a.Describe(&r, "An IP access control list mapped to a Twilio SIP domain for call authentication. Twilio has no update call, so every input replaces the mapping. "+
		"The resource ID is <domainSid>/<sid> (<accountSid>/<domainSid>/<sid> for a subaccount).")
}

var _ infer.CustomResource[SipDomainIpAccessControlListMappingArgs, SipDomainIpAccessControlListMappingState] = (*SipDomainIpAccessControlListMapping)(nil)
var _ infer.CustomCheck[SipDomainIpAccessControlListMappingArgs] = (*SipDomainIpAccessControlListMapping)(nil)
var _ infer.CustomRead[SipDomainIpAccessControlListMappingArgs, SipDomainIpAccessControlListMappingState] = (*SipDomainIpAccessControlListMapping)(nil)
var _ infer.CustomDelete[SipDomainIpAccessControlListMappingState] = (*SipDomainIpAccessControlListMapping)(nil)

func (r *SipDomainIpAccessControlListMapping) Check(ctx context.Context, req infer.CheckRequest) (infer.CheckResponse[SipDomainIpAccessControlListMappingArgs], error) {
	args, failures, err := infer.DefaultCheck[SipDomainIpAccessControlListMappingArgs](ctx, req.NewInputs)
	if err != nil || len(failures) > 0 {
		return infer.CheckResponse[SipDomainIpAccessControlListMappingArgs]{Inputs: args, Failures: failures}, err
	}
	unk := unknowns(req.NewInputs)
	if !unk("domainSid") && args.DomainSid == "" {
		failures = append(failures, p.CheckFailure{Property: "domainSid", Reason: "domainSid must be set"})
	}
	if !unk("ipAccessControlListSid") && args.IpAccessControlListSid == "" {
		failures = append(failures, p.CheckFailure{Property: "ipAccessControlListSid", Reason: "ipAccessControlListSid must be set"})
	}
	return infer.CheckResponse[SipDomainIpAccessControlListMappingArgs]{Inputs: args, Failures: failures}, nil
}

func (r *SipDomainIpAccessControlListMapping) Create(ctx context.Context, req infer.CreateRequest[SipDomainIpAccessControlListMappingArgs]) (infer.CreateResponse[SipDomainIpAccessControlListMappingState], error) {
	a := req.Inputs
	if req.DryRun {
		return infer.CreateResponse[SipDomainIpAccessControlListMappingState]{ID: "", Output: SipDomainIpAccessControlListMappingState{SipDomainIpAccessControlListMappingArgs: a}}, nil
	}
	c, err := clientFor(ctx)
	if err != nil {
		return infer.CreateResponse[SipDomainIpAccessControlListMappingState]{}, err
	}
	acct := c.acct(a.AccountSid)
	o, err := c.do(ctx, "POST", "api", sipMappingPath(acct, a.DomainSid, "Calls", "IpAccessControlListMappings", ""), nil,
		url.Values{"IpAccessControlListSid": {a.IpAccessControlListSid}})
	if err != nil {
		return infer.CreateResponse[SipDomainIpAccessControlListMappingState]{}, fmt.Errorf("create SIP domain IP access control list mapping: %w", err)
	}
	sid := o.str("sid")
	return infer.CreateResponse[SipDomainIpAccessControlListMappingState]{ID: childID(c, acct, a.DomainSid, sid), Output: SipDomainIpAccessControlListMappingState{
		SipDomainIpAccessControlListMappingArgs: a, Sid: sid, OwnerAccountSid: acct, FriendlyName: o.str("friendly_name"),
		DateCreated: o.str("date_created"), DateUpdated: o.str("date_updated"),
	}}, nil
}

func (r *SipDomainIpAccessControlListMapping) Read(ctx context.Context, req infer.ReadRequest[SipDomainIpAccessControlListMappingArgs, SipDomainIpAccessControlListMappingState]) (infer.ReadResponse[SipDomainIpAccessControlListMappingArgs, SipDomainIpAccessControlListMappingState], error) {
	type resp = infer.ReadResponse[SipDomainIpAccessControlListMappingArgs, SipDomainIpAccessControlListMappingState]
	c, err := clientFor(ctx)
	if err != nil {
		return resp{}, err
	}
	acct, domain, sid, err := resolveChildScope(c, req.ID, req.State.OwnerAccountSid)
	if err != nil {
		return resp{}, err
	}
	o, err := c.do(ctx, "GET", "api", sipMappingPath(acct, domain, "Calls", "IpAccessControlListMappings", sid), nil, nil)
	if err != nil {
		if isNotFound(err) {
			if err := notFoundOnImport("sip domain ip access control list mapping", req.ID, req.Inputs, req.State); err != nil {
				return resp{}, err
			}
			return resp{}, nil
		}
		return resp{}, fmt.Errorf("read SIP domain IP access control list mapping %s: %w", req.ID, err)
	}
	in := req.Inputs
	if in.DomainSid == "" { // import
		in.DomainSid, in.IpAccessControlListSid = domain, sid
		if acct != c.accountSid {
			in.AccountSid = &acct
		}
	}
	st := SipDomainIpAccessControlListMappingState{
		SipDomainIpAccessControlListMappingArgs: in, Sid: sid, OwnerAccountSid: acct, FriendlyName: o.str("friendly_name"),
		DateCreated: o.str("date_created"), DateUpdated: o.str("date_updated"),
	}
	return resp{ID: req.ID, Inputs: in, State: st}, nil
}

func (r *SipDomainIpAccessControlListMapping) Delete(ctx context.Context, req infer.DeleteRequest[SipDomainIpAccessControlListMappingState]) (infer.DeleteResponse, error) {
	c, err := clientFor(ctx)
	if err != nil {
		return infer.DeleteResponse{}, err
	}
	acct, domain, sid, err := resolveChildScope(c, req.ID, req.State.OwnerAccountSid)
	if err != nil {
		return infer.DeleteResponse{}, err
	}
	if _, err := c.do(ctx, "DELETE", "api", sipMappingPath(acct, domain, "Calls", "IpAccessControlListMappings", sid), nil, nil); err != nil && !isNotFound(err) {
		return infer.DeleteResponse{}, fmt.Errorf("delete SIP domain IP access control list mapping %s: %w", req.ID, err)
	}
	return infer.DeleteResponse{}, nil
}

// WireDependencies: the SID, owner and creation time are fixed at creation; the list name and the
// update time are what Twilio reports for the mapped list.
func (r *SipDomainCredentialListMapping) WireDependencies(f infer.FieldSelector, args *SipDomainCredentialListMappingArgs, state *SipDomainCredentialListMappingState) {
	wireEchoes(f, args, state)
	for _, o := range []any{&state.FriendlyName, &state.DateUpdated} {
		f.OutputField(o).DependsOn(f.InputField(args).Computed())
	}
}

// WireDependencies: the SID, owner and creation time are fixed at creation; the list name and the
// update time are what Twilio reports for the mapped list.
func (r *SipDomainIpAccessControlListMapping) WireDependencies(f infer.FieldSelector, args *SipDomainIpAccessControlListMappingArgs, state *SipDomainIpAccessControlListMappingState) {
	wireEchoes(f, args, state)
	for _, o := range []any{&state.FriendlyName, &state.DateUpdated} {
		f.OutputField(o).DependsOn(f.InputField(args).Computed())
	}
}
