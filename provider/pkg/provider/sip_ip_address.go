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

// SipIpAddress manages one IP address (or range) in a SIP IP access control list.
type SipIpAddress struct{}

// SipIpAddressArgs are the inputs of a SipIpAddress.
type SipIpAddressArgs struct {
	IpAccessControlListSid string  `pulumi:"ipAccessControlListSid" provider:"replaceOnChanges"`
	FriendlyName           string  `pulumi:"friendlyName"`
	IpAddress              string  `pulumi:"ipAddress"`
	AccountSid             *string `pulumi:"accountSid,optional" provider:"replaceOnChanges"`
	CidrPrefixLength       *int    `pulumi:"cidrPrefixLength,optional"`
}

func (a *SipIpAddressArgs) props() props {
	return props{ints: []intProp{{"CidrPrefixLength", "cidr_prefix_length", &a.CidrPrefixLength}}}
}

func (a *SipIpAddressArgs) Annotate(an infer.Annotator) {
	an.Describe(&a.IpAccessControlListSid, "SID (AL...) of the SIP IP access control list the address belongs to. Changing it replaces the resource.")
	an.Describe(&a.FriendlyName, "Human-readable name of the address (up to 255 characters).")
	an.Describe(&a.IpAddress, "IPv4 address in dotted decimal notation from which SIP traffic is accepted. IPv6 is not supported by Twilio.")
	an.Describe(&a.AccountSid, "SID of the account or subaccount that owns the list. Defaults to the provider account. "+
		"Changing it replaces the resource. To import an address owned by a subaccount use the ID <accountSid>/<ipAccessControlListSid>/<sid>.")
	an.Describe(&a.CidrPrefixLength, "Length of the CIDR prefix applied to the address, 0 to 32. By default the entire address is used (32). Removing the input resets it to 32.")
}

// SipIpAddressState is the persisted state of a SipIpAddress.
type SipIpAddressState struct {
	SipIpAddressArgs

	// Sid is the address SID (IP...).
	Sid string `pulumi:"sid"`
	// OwnerAccountSid is the account that owns the address.
	OwnerAccountSid string `pulumi:"ownerAccountSid"`
	// DateCreated is the RFC 2822 creation time.
	DateCreated string `pulumi:"dateCreated"`
	// DateUpdated is the RFC 2822 time of the last update.
	DateUpdated string `pulumi:"dateUpdated"`
}

func (s *SipIpAddressState) Annotate(a infer.Annotator) {
	a.Describe(&s.Sid, "The IP address SID (IP...).")
	a.Describe(&s.OwnerAccountSid, "SID of the account that owns the address.")
	a.Describe(&s.DateCreated, "When the address was created (RFC 2822).")
	a.Describe(&s.DateUpdated, "When the address was last updated (RFC 2822).")
}

func (r *SipIpAddress) Annotate(a infer.Annotator) {
	a.Describe(&r, "An IP address in a Twilio SIP IP access control list. The resource ID is <ipAccessControlListSid>/<sid> "+
		"(<accountSid>/<ipAccessControlListSid>/<sid> for a subaccount).")
}

var _ infer.CustomResource[SipIpAddressArgs, SipIpAddressState] = (*SipIpAddress)(nil)
var _ infer.CustomCheck[SipIpAddressArgs] = (*SipIpAddress)(nil)
var _ infer.CustomRead[SipIpAddressArgs, SipIpAddressState] = (*SipIpAddress)(nil)
var _ infer.CustomUpdate[SipIpAddressArgs, SipIpAddressState] = (*SipIpAddress)(nil)
var _ infer.CustomDelete[SipIpAddressState] = (*SipIpAddress)(nil)

func (r *SipIpAddress) Check(ctx context.Context, req infer.CheckRequest) (infer.CheckResponse[SipIpAddressArgs], error) {
	args, failures, err := infer.DefaultCheck[SipIpAddressArgs](ctx, req.NewInputs)
	if err != nil || len(failures) > 0 {
		return infer.CheckResponse[SipIpAddressArgs]{Inputs: args, Failures: failures}, err
	}
	unk := unknowns(req.NewInputs)
	if !unk("ipAccessControlListSid") && args.IpAccessControlListSid == "" {
		failures = append(failures, p.CheckFailure{Property: "ipAccessControlListSid", Reason: "ipAccessControlListSid must be set"})
	}
	if !unk("friendlyName") && (args.FriendlyName == "" || len(args.FriendlyName) > 255) {
		failures = append(failures, p.CheckFailure{Property: "friendlyName", Reason: "friendlyName must be 1-255 characters"})
	}
	if !unk("ipAddress") && args.IpAddress == "" {
		failures = append(failures, p.CheckFailure{Property: "ipAddress", Reason: "ipAddress must be set"})
	}
	if args.CidrPrefixLength != nil && !unk("cidrPrefixLength") && (*args.CidrPrefixLength < 0 || *args.CidrPrefixLength > 32) {
		failures = append(failures, p.CheckFailure{Property: "cidrPrefixLength", Reason: "cidrPrefixLength must be between 0 and 32"})
	}
	return infer.CheckResponse[SipIpAddressArgs]{Inputs: args, Failures: failures}, nil
}

func (a *SipIpAddressArgs) form() url.Values {
	return url.Values{"FriendlyName": {a.FriendlyName}, "IpAddress": {a.IpAddress}}
}

func (r *SipIpAddress) Create(ctx context.Context, req infer.CreateRequest[SipIpAddressArgs]) (infer.CreateResponse[SipIpAddressState], error) {
	a := req.Inputs
	if req.DryRun {
		return infer.CreateResponse[SipIpAddressState]{ID: "", Output: SipIpAddressState{SipIpAddressArgs: a}}, nil
	}
	c, err := clientFor(ctx)
	if err != nil {
		return infer.CreateResponse[SipIpAddressState]{}, err
	}
	acct := c.acct(a.AccountSid)
	form := a.form()
	a.props().encode(form, nil)
	o, err := c.do(ctx, "POST", "api", sipChildPath(acct, "IpAccessControlLists", a.IpAccessControlListSid, "IpAddresses", ""), nil, form)
	if err != nil {
		return infer.CreateResponse[SipIpAddressState]{}, fmt.Errorf("create SIP IP address: %w", err)
	}
	sid := o.str("sid")
	return infer.CreateResponse[SipIpAddressState]{ID: childID(c, acct, a.IpAccessControlListSid, sid), Output: SipIpAddressState{
		SipIpAddressArgs: a, Sid: sid, OwnerAccountSid: acct, DateCreated: o.str("date_created"), DateUpdated: o.str("date_updated"),
	}}, nil
}

func (r *SipIpAddress) Read(ctx context.Context, req infer.ReadRequest[SipIpAddressArgs, SipIpAddressState]) (infer.ReadResponse[SipIpAddressArgs, SipIpAddressState], error) {
	c, err := clientFor(ctx)
	if err != nil {
		return infer.ReadResponse[SipIpAddressArgs, SipIpAddressState]{}, err
	}
	acct, list, sid, err := resolveChildScope(c, req.ID, req.State.OwnerAccountSid)
	if err != nil {
		return infer.ReadResponse[SipIpAddressArgs, SipIpAddressState]{}, err
	}
	o, err := c.do(ctx, "GET", "api", sipChildPath(acct, "IpAccessControlLists", list, "IpAddresses", sid), nil, nil)
	if err != nil {
		if isNotFound(err) {
			if err := notFoundOnImport("sip ip address", req.ID, req.Inputs, req.State); err != nil {
				return infer.ReadResponse[SipIpAddressArgs, SipIpAddressState]{}, err
			}
			return infer.ReadResponse[SipIpAddressArgs, SipIpAddressState]{}, nil
		}
		return infer.ReadResponse[SipIpAddressArgs, SipIpAddressState]{}, fmt.Errorf("read SIP IP address %s: %w", req.ID, err)
	}
	in := req.Inputs
	imported := in.IpAddress == ""
	in.IpAccessControlListSid, in.FriendlyName, in.IpAddress = list, o.str("friendly_name"), o.str("ip_address")
	if imported {
		if acct != c.accountSid {
			in.AccountSid = &acct
		}
		in.props().adopt(o)
	} else {
		in.props().refreshSet(o)
	}
	st := SipIpAddressState{SipIpAddressArgs: in, Sid: sid, OwnerAccountSid: acct, DateCreated: o.str("date_created"), DateUpdated: o.str("date_updated")}
	return infer.ReadResponse[SipIpAddressArgs, SipIpAddressState]{ID: req.ID, Inputs: in, State: st}, nil
}

func (r *SipIpAddress) Update(ctx context.Context, req infer.UpdateRequest[SipIpAddressArgs, SipIpAddressState]) (infer.UpdateResponse[SipIpAddressState], error) {
	a := req.Inputs
	st := req.State
	st.SipIpAddressArgs = a
	if req.DryRun {
		return infer.UpdateResponse[SipIpAddressState]{Output: st}, nil
	}
	c, err := clientFor(ctx)
	if err != nil {
		return infer.UpdateResponse[SipIpAddressState]{}, err
	}
	acct, list, sid, err := resolveChildScope(c, req.ID, req.State.OwnerAccountSid)
	if err != nil {
		return infer.UpdateResponse[SipIpAddressState]{}, err
	}
	form := a.form()
	prev := req.State.SipIpAddressArgs.props()
	a.props().encode(form, &prev)
	o, err := c.do(ctx, "POST", "api", sipChildPath(acct, "IpAccessControlLists", list, "IpAddresses", sid), nil, form)
	if err != nil {
		return infer.UpdateResponse[SipIpAddressState]{}, fmt.Errorf("update SIP IP address %s: %w", req.ID, err)
	}
	st.DateUpdated = o.str("date_updated")
	return infer.UpdateResponse[SipIpAddressState]{Output: st}, nil
}

func (r *SipIpAddress) Delete(ctx context.Context, req infer.DeleteRequest[SipIpAddressState]) (infer.DeleteResponse, error) {
	c, err := clientFor(ctx)
	if err != nil {
		return infer.DeleteResponse{}, err
	}
	acct, list, sid, err := resolveChildScope(c, req.ID, req.State.OwnerAccountSid)
	if err != nil {
		return infer.DeleteResponse{}, err
	}
	if _, err := c.do(ctx, "DELETE", "api", sipChildPath(acct, "IpAccessControlLists", list, "IpAddresses", sid), nil, nil); err != nil && !isNotFound(err) {
		return infer.DeleteResponse{}, fmt.Errorf("delete SIP IP address %s: %w", req.ID, err)
	}
	return infer.DeleteResponse{}, nil
}
