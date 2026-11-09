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

// OutgoingCallerId adopts a phone number the account has already verified as a caller ID and
// manages its label. It never starts a verification and never removes the caller ID.
type OutgoingCallerId struct{}

// OutgoingCallerIdArgs are the inputs of an OutgoingCallerId.
type OutgoingCallerIdArgs struct {
	PhoneNumber         *string `pulumi:"phoneNumber,optional" provider:"replaceOnChanges"`
	OutgoingCallerIdSid *string `pulumi:"outgoingCallerIdSid,optional" provider:"replaceOnChanges"`
	AccountSid          *string `pulumi:"accountSid,optional" provider:"replaceOnChanges"`
	FriendlyName        *string `pulumi:"friendlyName,optional"`
}

func (a *OutgoingCallerIdArgs) props() props {
	return props{strs: []strProp{{"FriendlyName", "friendly_name", &a.FriendlyName}}}
}

func (a *OutgoingCallerIdArgs) Annotate(an infer.Annotator) {
	an.Describe(&a.PhoneNumber, "The verified number in E.164 format, for example +14155550100. Set exactly one of phoneNumber or "+
		"outgoingCallerIdSid. Changing it replaces the resource (a different caller ID is adopted).")
	an.Describe(&a.OutgoingCallerIdSid, "SID (PN...) of the verified caller ID. Set exactly one of phoneNumber or outgoingCallerIdSid. "+
		"Changing it replaces the resource.")
	an.Describe(&a.AccountSid, "SID of the account or subaccount that verified the number. Defaults to the provider account. "+
		"Changing it replaces the resource. To import a caller ID of a subaccount use the ID <accountSid>/<outgoingCallerIdSid>.")
	an.Describe(&a.FriendlyName, "Human-readable label for the caller ID. Removing it leaves the label unchanged.")
}

// OutgoingCallerIdState is the persisted state of an OutgoingCallerId.
type OutgoingCallerIdState struct {
	OutgoingCallerIdArgs

	// Sid is the caller ID SID (PN...) and the resource ID.
	Sid string `pulumi:"sid"`
	// Number is the verified number in E.164 format.
	Number string `pulumi:"number"`
	// OwnerAccountSid is the account that verified the number.
	OwnerAccountSid string `pulumi:"ownerAccountSid"`
	// Effective is the configuration Twilio reports, including fields the program did not set.
	Effective *OutgoingCallerIdEffective `pulumi:"effective,optional"`
}

func (s *OutgoingCallerIdState) Annotate(a infer.Annotator) {
	a.Describe(&s.Sid, "The caller ID SID (PN...).")
	a.Describe(&s.Number, "The verified number in E.164 format.")
	a.Describe(&s.OwnerAccountSid, "SID of the account that verified the number.")
	a.Describe(&s.Effective, "The configuration Twilio currently reports for the caller ID, including a label changed "+
		"outside Pulumi. Inputs the program leaves unset are not overwritten by these values.")
}

func (r *OutgoingCallerId) Annotate(a infer.Annotator) {
	a.Describe(&r, "A phone number the account has already verified as an outgoing caller ID. The resource adopts the caller ID; "+
		"it never starts a verification (Twilio phones the number and a person must enter a code), and destroying it does NOT "+
		"remove the caller ID (re-verifying would need a person), it only stops managing it.")
}

var _ infer.CustomResource[OutgoingCallerIdArgs, OutgoingCallerIdState] = (*OutgoingCallerId)(nil)
var _ infer.CustomCheck[OutgoingCallerIdArgs] = (*OutgoingCallerId)(nil)
var _ infer.CustomRead[OutgoingCallerIdArgs, OutgoingCallerIdState] = (*OutgoingCallerId)(nil)
var _ infer.CustomUpdate[OutgoingCallerIdArgs, OutgoingCallerIdState] = (*OutgoingCallerId)(nil)
var _ infer.CustomDelete[OutgoingCallerIdState] = (*OutgoingCallerId)(nil)

func (r *OutgoingCallerId) Check(ctx context.Context, req infer.CheckRequest) (infer.CheckResponse[OutgoingCallerIdArgs], error) {
	args, failures, err := infer.DefaultCheck[OutgoingCallerIdArgs](ctx, req.NewInputs)
	if err != nil || len(failures) > 0 {
		return infer.CheckResponse[OutgoingCallerIdArgs]{Inputs: args, Failures: failures}, err
	}
	unk := unknowns(req.NewInputs)
	hasNum := unk("phoneNumber") || deref(args.PhoneNumber) != ""
	hasSid := unk("outgoingCallerIdSid") || deref(args.OutgoingCallerIdSid) != ""
	if hasNum == hasSid {
		failures = append(failures, p.CheckFailure{Property: "phoneNumber", Reason: "set exactly one of phoneNumber or outgoingCallerIdSid"})
	}
	return infer.CheckResponse[OutgoingCallerIdArgs]{Inputs: args, Failures: failures}, nil
}

// findCallerID resolves the verified caller ID named by a SID or by the E.164 number.
func findCallerID(ctx context.Context, c *Client, acct, sid, e164 string) (obj, error) {
	if sid != "" {
		return fetchAdopted(ctx, c, acct, callerIDColl, sid)
	}
	return findAdopted(ctx, c, acct, callerIDColl, "PhoneNumber", "phone_number", e164)
}

// OutgoingCallerIdEffective is the configuration Twilio reports for an OutgoingCallerId.
type OutgoingCallerIdEffective OutgoingCallerIdArgs

// Annotate reuses the input descriptions.
func (e *OutgoingCallerIdEffective) Annotate(a infer.Annotator) {
	(*OutgoingCallerIdArgs)(e).Annotate(a)
}

func callerIDEffective(o obj) *OutgoingCallerIdEffective {
	e := OutgoingCallerIdArgs{}
	e.props().fillAll(o)
	v := OutgoingCallerIdEffective(e)
	return &v
}

func (r *OutgoingCallerId) Create(ctx context.Context, req infer.CreateRequest[OutgoingCallerIdArgs]) (infer.CreateResponse[OutgoingCallerIdState], error) {
	a := req.Inputs
	if req.DryRun {
		return infer.CreateResponse[OutgoingCallerIdState]{ID: "", Output: OutgoingCallerIdState{OutgoingCallerIdArgs: a}}, nil
	}
	c, err := clientFor(ctx)
	if err != nil {
		return infer.CreateResponse[OutgoingCallerIdState]{}, err
	}
	acct := c.acct(a.AccountSid)
	cur, err := findCallerID(ctx, c, acct, deref(a.OutgoingCallerIdSid), deref(a.PhoneNumber))
	if err != nil {
		return infer.CreateResponse[OutgoingCallerIdState]{}, fmt.Errorf("adopt caller ID: %w", err)
	}
	sid := cur.str("sid")
	st := OutgoingCallerIdState{OutgoingCallerIdArgs: a, Sid: sid, Number: cur.str("phone_number"), OwnerAccountSid: acct, Effective: callerIDEffective(cur)}
	form := url.Values{}
	a.props().encode(form, nil)
	if len(form) > 0 {
		o, err := c.do(ctx, "POST", "api", acctPath(acct, callerIDColl.coll, sid), nil, form)
		if err != nil {
			// The caller ID is adopted but not configured: save it so the next update retries.
			return infer.CreateResponse[OutgoingCallerIdState]{ID: scopedID(c, acct, sid), Output: st},
				infer.ResourceInitFailedError{Reasons: []string{fmt.Sprintf("configure caller ID %s: %v", sid, err)}}
		}
		st.Effective = callerIDEffective(o)
	}
	return infer.CreateResponse[OutgoingCallerIdState]{ID: scopedID(c, acct, sid), Output: st}, nil
}

func (r *OutgoingCallerId) Read(ctx context.Context, req infer.ReadRequest[OutgoingCallerIdArgs, OutgoingCallerIdState]) (infer.ReadResponse[OutgoingCallerIdArgs, OutgoingCallerIdState], error) {
	c, err := clientFor(ctx)
	if err != nil {
		return infer.ReadResponse[OutgoingCallerIdArgs, OutgoingCallerIdState]{}, err
	}
	acct, sid := resolveScope(c, req.ID, req.State.OwnerAccountSid)
	o, err := c.do(ctx, "GET", "api", acctPath(acct, callerIDColl.coll, sid), nil, nil)
	if err != nil {
		if isNotFound(err) {
			if err := notFoundOnImport("outgoing caller id", req.ID, req.Inputs, req.State); err != nil {
				return infer.ReadResponse[OutgoingCallerIdArgs, OutgoingCallerIdState]{}, err
			}
			return infer.ReadResponse[OutgoingCallerIdArgs, OutgoingCallerIdState]{}, nil
		}
		return infer.ReadResponse[OutgoingCallerIdArgs, OutgoingCallerIdState]{}, fmt.Errorf("read caller ID %s: %w", req.ID, err)
	}
	in := req.Inputs
	if in.PhoneNumber == nil && in.OutgoingCallerIdSid == nil { // an import
		s := sid
		in.OutgoingCallerIdSid = &s
		if acct != c.accountSid {
			in.AccountSid = &acct
		}
		in.props().adopt(o)
	} else {
		in.props().refreshSet(o)
	}
	st := OutgoingCallerIdState{OutgoingCallerIdArgs: in, Sid: sid, Number: o.str("phone_number"),
		OwnerAccountSid: derefOr(nonEmpty(o.optStr("account_sid")), acct), Effective: callerIDEffective(o)}
	return infer.ReadResponse[OutgoingCallerIdArgs, OutgoingCallerIdState]{ID: req.ID, Inputs: in, State: st}, nil
}

func (r *OutgoingCallerId) Update(ctx context.Context, req infer.UpdateRequest[OutgoingCallerIdArgs, OutgoingCallerIdState]) (infer.UpdateResponse[OutgoingCallerIdState], error) {
	a := req.Inputs
	st := req.State
	st.OutgoingCallerIdArgs = a
	if req.DryRun { // a preview needs no credentials
		return infer.UpdateResponse[OutgoingCallerIdState]{Output: st}, nil
	}
	c, err := clientFor(ctx)
	if err != nil {
		return infer.UpdateResponse[OutgoingCallerIdState]{}, err
	}
	acct, sid := resolveScope(c, req.ID, req.State.OwnerAccountSid)
	form := url.Values{}
	prev := req.State.OutgoingCallerIdArgs.props()
	a.props().encode(form, &prev)
	if len(form) > 0 {
		o, err := c.do(ctx, "POST", "api", acctPath(acct, callerIDColl.coll, sid), nil, form)
		if err != nil {
			return infer.UpdateResponse[OutgoingCallerIdState]{}, fmt.Errorf("update caller ID %s: %w", req.ID, err)
		}
		st.Effective = callerIDEffective(o)
	}
	return infer.UpdateResponse[OutgoingCallerIdState]{Output: st}, nil
}

// Delete only stops managing the caller ID: it is never removed, because verifying it again needs a person.
func (r *OutgoingCallerId) Delete(ctx context.Context, req infer.DeleteRequest[OutgoingCallerIdState]) (infer.DeleteResponse, error) {
	return infer.DeleteResponse{}, nil
}
