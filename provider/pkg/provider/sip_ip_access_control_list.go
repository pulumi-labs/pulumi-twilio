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

// SipIpAccessControlList manages a SIP IP access control list.
type SipIpAccessControlList struct{}

// SipIpAccessControlListArgs are the inputs of a SipIpAccessControlList.
type SipIpAccessControlListArgs struct {
	FriendlyName string  `pulumi:"friendlyName"`
	AccountSid   *string `pulumi:"accountSid,optional" provider:"replaceOnChanges"`
}

func (a *SipIpAccessControlListArgs) Annotate(an infer.Annotator) {
	an.Describe(&a.FriendlyName, "Human-readable name of the list (up to 255 characters).")
	an.Describe(&a.AccountSid, "SID of the account or subaccount that owns the list. Defaults to the provider account. "+
		"Changing it replaces the resource. To import a list owned by a subaccount use the ID <accountSid>/<sid>.")
}

// SipIpAccessControlListState is the persisted state of a SipIpAccessControlList.
type SipIpAccessControlListState struct {
	SipIpAccessControlListArgs

	// Sid is the list SID (AL...) and the resource ID.
	Sid string `pulumi:"sid"`
	// OwnerAccountSid is the account that owns the list.
	OwnerAccountSid string `pulumi:"ownerAccountSid"`
	// DateCreated is the RFC 2822 creation time.
	DateCreated string `pulumi:"dateCreated"`
	// DateUpdated is the RFC 2822 time of the last update.
	DateUpdated string `pulumi:"dateUpdated"`
}

func (s *SipIpAccessControlListState) Annotate(a infer.Annotator) {
	a.Describe(&s.Sid, "The IP access control list SID (AL...) and the resource ID.")
	a.Describe(&s.OwnerAccountSid, "SID of the account that owns the list.")
	a.Describe(&s.DateCreated, "When the list was created (RFC 2822).")
	a.Describe(&s.DateUpdated, "When the list was last updated (RFC 2822).")
}

func (r *SipIpAccessControlList) Annotate(a infer.Annotator) {
	a.Describe(&r, "A Twilio SIP IP access control list: a named set of IP addresses and ranges from which SIP traffic is accepted. Add addresses with the SipIpAddress resource. Mapping a list to a SIP domain is not managed yet.")
}

var _ infer.CustomResource[SipIpAccessControlListArgs, SipIpAccessControlListState] = (*SipIpAccessControlList)(nil)
var _ infer.CustomCheck[SipIpAccessControlListArgs] = (*SipIpAccessControlList)(nil)
var _ infer.CustomRead[SipIpAccessControlListArgs, SipIpAccessControlListState] = (*SipIpAccessControlList)(nil)
var _ infer.CustomUpdate[SipIpAccessControlListArgs, SipIpAccessControlListState] = (*SipIpAccessControlList)(nil)
var _ infer.CustomDelete[SipIpAccessControlListState] = (*SipIpAccessControlList)(nil)

func (r *SipIpAccessControlList) Check(ctx context.Context, req infer.CheckRequest) (infer.CheckResponse[SipIpAccessControlListArgs], error) {
	args, failures, err := infer.DefaultCheck[SipIpAccessControlListArgs](ctx, req.NewInputs)
	if err != nil || len(failures) > 0 {
		return infer.CheckResponse[SipIpAccessControlListArgs]{Inputs: args, Failures: failures}, err
	}
	if !unknowns(req.NewInputs)("friendlyName") && (args.FriendlyName == "" || len(args.FriendlyName) > 255) {
		failures = append(failures, p.CheckFailure{Property: "friendlyName", Reason: "friendlyName must be 1-255 characters"})
	}
	return infer.CheckResponse[SipIpAccessControlListArgs]{Inputs: args, Failures: failures}, nil
}

func sipIpAccessControlListOutputs(st *SipIpAccessControlListState, o obj) {
	st.DateCreated, st.DateUpdated = o.str("date_created"), o.str("date_updated")
}

func (r *SipIpAccessControlList) Create(ctx context.Context, req infer.CreateRequest[SipIpAccessControlListArgs]) (infer.CreateResponse[SipIpAccessControlListState], error) {
	a := req.Inputs
	if req.DryRun {
		return infer.CreateResponse[SipIpAccessControlListState]{ID: "", Output: SipIpAccessControlListState{SipIpAccessControlListArgs: a}}, nil
	}
	c, err := clientFor(ctx)
	if err != nil {
		return infer.CreateResponse[SipIpAccessControlListState]{}, err
	}
	acct := c.acct(a.AccountSid)
	o, err := c.do(ctx, "POST", "api", sipPath(acct, "IpAccessControlLists", ""), nil, url.Values{"FriendlyName": {a.FriendlyName}})
	if err != nil {
		return infer.CreateResponse[SipIpAccessControlListState]{}, fmt.Errorf("create SIP IP access control list: %w", err)
	}
	sid := o.str("sid")
	st := SipIpAccessControlListState{SipIpAccessControlListArgs: a, Sid: sid, OwnerAccountSid: acct}
	sipIpAccessControlListOutputs(&st, o)
	return infer.CreateResponse[SipIpAccessControlListState]{ID: scopedID(c, acct, sid), Output: st}, nil
}

func (r *SipIpAccessControlList) Read(ctx context.Context, req infer.ReadRequest[SipIpAccessControlListArgs, SipIpAccessControlListState]) (infer.ReadResponse[SipIpAccessControlListArgs, SipIpAccessControlListState], error) {
	c, err := clientFor(ctx)
	if err != nil {
		return infer.ReadResponse[SipIpAccessControlListArgs, SipIpAccessControlListState]{}, err
	}
	acct, sid := resolveScope(c, req.ID, req.State.OwnerAccountSid)
	o, err := c.do(ctx, "GET", "api", sipPath(acct, "IpAccessControlLists", sid), nil, nil)
	if err != nil {
		if isNotFound(err) {
			if err := notFoundOnImport("sip ip access control list", req.ID, req.Inputs, req.State); err != nil {
				return infer.ReadResponse[SipIpAccessControlListArgs, SipIpAccessControlListState]{}, err
			}
			return infer.ReadResponse[SipIpAccessControlListArgs, SipIpAccessControlListState]{}, nil
		}
		return infer.ReadResponse[SipIpAccessControlListArgs, SipIpAccessControlListState]{}, fmt.Errorf("read SIP IP access control list %s: %w", req.ID, err)
	}
	in := req.Inputs
	if in.FriendlyName == "" && acct != c.accountSid { // import
		in.AccountSid = &acct
	}
	in.FriendlyName = o.str("friendly_name")
	st := SipIpAccessControlListState{SipIpAccessControlListArgs: in, Sid: sid, OwnerAccountSid: acct}
	sipIpAccessControlListOutputs(&st, o)
	return infer.ReadResponse[SipIpAccessControlListArgs, SipIpAccessControlListState]{ID: req.ID, Inputs: in, State: st}, nil
}

func (r *SipIpAccessControlList) Update(ctx context.Context, req infer.UpdateRequest[SipIpAccessControlListArgs, SipIpAccessControlListState]) (infer.UpdateResponse[SipIpAccessControlListState], error) {
	st := req.State
	st.SipIpAccessControlListArgs = req.Inputs
	if req.DryRun {
		return infer.UpdateResponse[SipIpAccessControlListState]{Output: st}, nil
	}
	c, err := clientFor(ctx)
	if err != nil {
		return infer.UpdateResponse[SipIpAccessControlListState]{}, err
	}
	acct, sid := resolveScope(c, req.ID, req.State.OwnerAccountSid)
	o, err := c.do(ctx, "POST", "api", sipPath(acct, "IpAccessControlLists", sid), nil, url.Values{"FriendlyName": {req.Inputs.FriendlyName}})
	if err != nil {
		return infer.UpdateResponse[SipIpAccessControlListState]{}, fmt.Errorf("update SIP IP access control list %s: %w", req.ID, err)
	}
	sipIpAccessControlListOutputs(&st, o)
	return infer.UpdateResponse[SipIpAccessControlListState]{Output: st}, nil
}

func (r *SipIpAccessControlList) Delete(ctx context.Context, req infer.DeleteRequest[SipIpAccessControlListState]) (infer.DeleteResponse, error) {
	c, err := clientFor(ctx)
	if err != nil {
		return infer.DeleteResponse{}, err
	}
	acct, sid := resolveScope(c, req.ID, req.State.OwnerAccountSid)
	if _, err := c.do(ctx, "DELETE", "api", sipPath(acct, "IpAccessControlLists", sid), nil, nil); err != nil && !isNotFound(err) {
		return infer.DeleteResponse{}, fmt.Errorf("delete SIP IP access control list %s: %w", req.ID, err)
	}
	return infer.DeleteResponse{}, nil
}
