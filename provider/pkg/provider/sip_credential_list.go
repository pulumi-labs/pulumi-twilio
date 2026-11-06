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

	p "github.com/pulumi/pulumi-go-provider"
	"github.com/pulumi/pulumi-go-provider/infer"
)

// SipCredentialList manages a SIP credential list.
type SipCredentialList struct{}

// SipCredentialListArgs are the inputs of a SipCredentialList.
type SipCredentialListArgs struct {
	FriendlyName string  `pulumi:"friendlyName"`
	AccountSid   *string `pulumi:"accountSid,optional" provider:"replaceOnChanges"`
}

func (a *SipCredentialListArgs) Annotate(an infer.Annotator) {
	an.Describe(&a.FriendlyName, "Human-readable name of the list (up to 64 characters).")
	an.Describe(&a.AccountSid, "SID of the account or subaccount that owns the list. Defaults to the provider account. "+
		"Changing it replaces the resource. To import a list owned by a subaccount use the ID <accountSid>/<sid>.")
}

// SipCredentialListState is the persisted state of a SipCredentialList.
type SipCredentialListState struct {
	SipCredentialListArgs

	// Sid is the list SID (CL...) and the resource ID.
	Sid string `pulumi:"sid"`
	// OwnerAccountSid is the account that owns the list.
	OwnerAccountSid string `pulumi:"ownerAccountSid"`
	// DateCreated is the RFC 2822 creation time.
	DateCreated string `pulumi:"dateCreated"`
	// DateUpdated is the RFC 2822 time of the last update.
	DateUpdated string `pulumi:"dateUpdated"`
}

func (s *SipCredentialListState) Annotate(a infer.Annotator) {
	a.Describe(&s.Sid, "The credential list SID (CL...) and the resource ID.")
	a.Describe(&s.OwnerAccountSid, "SID of the account that owns the list.")
	a.Describe(&s.DateCreated, "When the list was created (RFC 2822).")
	a.Describe(&s.DateUpdated, "When the list was last updated (RFC 2822).")
}

func (r *SipCredentialList) Annotate(a infer.Annotator) {
	a.Describe(&r, "A Twilio SIP credential list: a named set of username and password credentials that SIP domains and trunks can use to authenticate SIP requests. Add credentials with the SipCredential resource. Mapping a list to a SIP domain is not managed yet.")
}

var _ infer.CustomResource[SipCredentialListArgs, SipCredentialListState] = (*SipCredentialList)(nil)
var _ infer.CustomCheck[SipCredentialListArgs] = (*SipCredentialList)(nil)
var _ infer.CustomRead[SipCredentialListArgs, SipCredentialListState] = (*SipCredentialList)(nil)
var _ infer.CustomUpdate[SipCredentialListArgs, SipCredentialListState] = (*SipCredentialList)(nil)
var _ infer.CustomDelete[SipCredentialListState] = (*SipCredentialList)(nil)

func (r *SipCredentialList) Check(ctx context.Context, req infer.CheckRequest) (infer.CheckResponse[SipCredentialListArgs], error) {
	args, failures, err := infer.DefaultCheck[SipCredentialListArgs](ctx, req.NewInputs)
	if err != nil || len(failures) > 0 {
		return infer.CheckResponse[SipCredentialListArgs]{Inputs: args, Failures: failures}, err
	}
	if !unknowns(req.NewInputs)("friendlyName") && (args.FriendlyName == "" || len(args.FriendlyName) > 64) {
		failures = append(failures, p.CheckFailure{Property: "friendlyName", Reason: "friendlyName must be 1-64 characters"})
	}
	return infer.CheckResponse[SipCredentialListArgs]{Inputs: args, Failures: failures}, nil
}

func sipCredentialListOutputs(st *SipCredentialListState, o obj) {
	st.DateCreated, st.DateUpdated = o.str("date_created"), o.str("date_updated")
}

func (r *SipCredentialList) Create(ctx context.Context, req infer.CreateRequest[SipCredentialListArgs]) (infer.CreateResponse[SipCredentialListState], error) {
	a := req.Inputs
	if req.DryRun {
		return infer.CreateResponse[SipCredentialListState]{ID: "", Output: SipCredentialListState{SipCredentialListArgs: a}}, nil
	}
	c, err := clientFor(ctx)
	if err != nil {
		return infer.CreateResponse[SipCredentialListState]{}, err
	}
	acct := c.acct(a.AccountSid)
	o, err := c.do(ctx, "POST", "api", sipPath(acct, "CredentialLists", ""), nil, url.Values{"FriendlyName": {a.FriendlyName}})
	if err != nil {
		return infer.CreateResponse[SipCredentialListState]{}, fmt.Errorf("create SIP credential list: %w", err)
	}
	sid := o.str("sid")
	st := SipCredentialListState{SipCredentialListArgs: a, Sid: sid, OwnerAccountSid: acct}
	sipCredentialListOutputs(&st, o)
	return infer.CreateResponse[SipCredentialListState]{ID: scopedID(c, acct, sid), Output: st}, nil
}

func (r *SipCredentialList) Read(ctx context.Context, req infer.ReadRequest[SipCredentialListArgs, SipCredentialListState]) (infer.ReadResponse[SipCredentialListArgs, SipCredentialListState], error) {
	c, err := clientFor(ctx)
	if err != nil {
		return infer.ReadResponse[SipCredentialListArgs, SipCredentialListState]{}, err
	}
	acct, sid := resolveScope(c, req.ID, req.State.OwnerAccountSid)
	o, err := c.do(ctx, "GET", "api", sipPath(acct, "CredentialLists", sid), nil, nil)
	if err != nil {
		if isNotFound(err) {
			return infer.ReadResponse[SipCredentialListArgs, SipCredentialListState]{}, nil
		}
		return infer.ReadResponse[SipCredentialListArgs, SipCredentialListState]{}, fmt.Errorf("read SIP credential list %s: %w", req.ID, err)
	}
	in := req.Inputs
	if in.FriendlyName == "" && acct != c.accountSid { // import
		in.AccountSid = &acct
	}
	in.FriendlyName = o.str("friendly_name")
	st := SipCredentialListState{SipCredentialListArgs: in, Sid: sid, OwnerAccountSid: acct}
	sipCredentialListOutputs(&st, o)
	return infer.ReadResponse[SipCredentialListArgs, SipCredentialListState]{ID: req.ID, Inputs: in, State: st}, nil
}

func (r *SipCredentialList) Update(ctx context.Context, req infer.UpdateRequest[SipCredentialListArgs, SipCredentialListState]) (infer.UpdateResponse[SipCredentialListState], error) {
	st := req.State
	st.SipCredentialListArgs = req.Inputs
	if req.DryRun {
		return infer.UpdateResponse[SipCredentialListState]{Output: st}, nil
	}
	c, err := clientFor(ctx)
	if err != nil {
		return infer.UpdateResponse[SipCredentialListState]{}, err
	}
	acct, sid := resolveScope(c, req.ID, req.State.OwnerAccountSid)
	o, err := c.do(ctx, "POST", "api", sipPath(acct, "CredentialLists", sid), nil, url.Values{"FriendlyName": {req.Inputs.FriendlyName}})
	if err != nil {
		return infer.UpdateResponse[SipCredentialListState]{}, fmt.Errorf("update SIP credential list %s: %w", req.ID, err)
	}
	sipCredentialListOutputs(&st, o)
	return infer.UpdateResponse[SipCredentialListState]{Output: st}, nil
}

func (r *SipCredentialList) Delete(ctx context.Context, req infer.DeleteRequest[SipCredentialListState]) (infer.DeleteResponse, error) {
	c, err := clientFor(ctx)
	if err != nil {
		return infer.DeleteResponse{}, err
	}
	acct, sid := resolveScope(c, req.ID, req.State.OwnerAccountSid)
	if _, err := c.do(ctx, "DELETE", "api", sipPath(acct, "CredentialLists", sid), nil, nil); err != nil && !isNotFound(err) {
		return infer.DeleteResponse{}, fmt.Errorf("delete SIP credential list %s: %w", req.ID, err)
	}
	return infer.DeleteResponse{}, nil
}
