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
	"fmt"
	"net/url"
	"unicode/utf8"

	p "github.com/pulumi/pulumi-go-provider"
	"github.com/pulumi/pulumi-go-provider/infer"
)

// SipCredential manages one username and password credential in a SIP credential list.
type SipCredential struct{}

// SipCredentialArgs are the inputs of a SipCredential.
type SipCredentialArgs struct {
	CredentialListSid string  `pulumi:"credentialListSid" provider:"replaceOnChanges"`
	Username          string  `pulumi:"username" provider:"replaceOnChanges"`
	Password          string  `pulumi:"password" provider:"secret"`
	AccountSid        *string `pulumi:"accountSid,optional" provider:"replaceOnChanges"`
}

func (a *SipCredentialArgs) Annotate(an infer.Annotator) {
	an.Describe(&a.CredentialListSid, "SID (CL...) of the SIP credential list the credential belongs to. Changing it replaces the resource.")
	an.Describe(&a.Username, "Username SIP clients send when they authenticate (up to 32 characters). Twilio cannot rename a credential, so changing it replaces the resource.")
	an.Describe(&a.Password, "Password for the username: at least 12 characters, with at least one digit and mixed case. "+
		"Twilio never returns it, so drift is not detected and an imported credential has no password until the program sets one. Changing it updates the credential in place.")
	an.Describe(&a.AccountSid, "SID of the account or subaccount that owns the credential list. Defaults to the provider account. "+
		"Changing it replaces the resource. To import a credential owned by a subaccount use the ID <accountSid>/<credentialListSid>/<sid>.")
}

// SipCredentialState is the persisted state of a SipCredential.
type SipCredentialState struct {
	SipCredentialArgs

	// Sid is the credential SID (CR...).
	Sid string `pulumi:"sid"`
	// OwnerAccountSid is the account that owns the credential.
	OwnerAccountSid string `pulumi:"ownerAccountSid"`
	// DateCreated is the RFC 2822 creation time.
	DateCreated string `pulumi:"dateCreated"`
	// DateUpdated is the RFC 2822 time of the last update.
	DateUpdated string `pulumi:"dateUpdated"`
}

func (s *SipCredentialState) Annotate(a infer.Annotator) {
	a.Describe(&s.Sid, "The credential SID (CR...).")
	a.Describe(&s.OwnerAccountSid, "SID of the account that owns the credential.")
	a.Describe(&s.DateCreated, "When the credential was created (RFC 2822).")
	a.Describe(&s.DateUpdated, "When the credential was last updated (RFC 2822).")
}

func (r *SipCredential) Annotate(a infer.Annotator) {
	a.Describe(&r, "A username and password in a Twilio SIP credential list. The resource ID is <credentialListSid>/<sid> "+
		"(<accountSid>/<credentialListSid>/<sid> for a subaccount). The password is a Pulumi secret and is write-only: Twilio never returns it.")
}

var _ infer.CustomResource[SipCredentialArgs, SipCredentialState] = (*SipCredential)(nil)
var _ infer.CustomCheck[SipCredentialArgs] = (*SipCredential)(nil)
var _ infer.CustomRead[SipCredentialArgs, SipCredentialState] = (*SipCredential)(nil)
var _ infer.CustomUpdate[SipCredentialArgs, SipCredentialState] = (*SipCredential)(nil)
var _ infer.CustomDelete[SipCredentialState] = (*SipCredential)(nil)

func (r *SipCredential) Check(ctx context.Context, req infer.CheckRequest) (infer.CheckResponse[SipCredentialArgs], error) {
	args, failures, err := infer.DefaultCheck[SipCredentialArgs](ctx, req.NewInputs)
	if err != nil || len(failures) > 0 {
		return infer.CheckResponse[SipCredentialArgs]{Inputs: args, Failures: failures}, err
	}
	unk := unknowns(req.NewInputs)
	if !unk("credentialListSid") && args.CredentialListSid == "" {
		failures = append(failures, p.CheckFailure{Property: "credentialListSid", Reason: "credentialListSid must be set"})
	}
	if !unk("username") && (args.Username == "" || utf8.RuneCountInString(args.Username) > 32) {
		failures = append(failures, p.CheckFailure{Property: "username", Reason: "username must be 1-32 characters"})
	}
	// An imported credential has an empty password (Twilio never returns it), and the engine runs Check on
	// the imported inputs, so an empty password is not a failure here; Create and Update refuse it (#1635).
	if !unk("password") && args.Password != "" && utf8.RuneCountInString(args.Password) < 12 {
		failures = append(failures, p.CheckFailure{Property: "password", Reason: "password must be at least 12 characters"})
	}
	return infer.CheckResponse[SipCredentialArgs]{Inputs: args, Failures: failures}, nil
}

// errPasswordUnset is returned by Create and Update when the password is empty: only an import leaves it
// empty, and Twilio needs one to create or change a credential.
var errPasswordUnset = errors.New("password must be set: Twilio requires at least 12 characters, with a digit and mixed case")

func (r *SipCredential) Create(ctx context.Context, req infer.CreateRequest[SipCredentialArgs]) (infer.CreateResponse[SipCredentialState], error) {
	a := req.Inputs
	if req.DryRun { // a preview may see a computed password as empty: only a real write refuses it
		return infer.CreateResponse[SipCredentialState]{ID: "", Output: SipCredentialState{SipCredentialArgs: a}}, nil
	}
	if a.Password == "" {
		return infer.CreateResponse[SipCredentialState]{}, errPasswordUnset
	}
	c, err := clientFor(ctx)
	if err != nil {
		return infer.CreateResponse[SipCredentialState]{}, err
	}
	acct := c.acct(a.AccountSid)
	form := url.Values{"Username": {a.Username}, "Password": {a.Password}}
	o, err := c.do(ctx, "POST", "api", sipChildPath(acct, "CredentialLists", a.CredentialListSid, "Credentials", ""), nil, form)
	if err != nil {
		return infer.CreateResponse[SipCredentialState]{}, fmt.Errorf("create SIP credential: %w", err)
	}
	sid := o.str("sid")
	return infer.CreateResponse[SipCredentialState]{ID: childID(c, acct, a.CredentialListSid, sid), Output: SipCredentialState{
		SipCredentialArgs: a, Sid: sid, OwnerAccountSid: acct, DateCreated: o.str("date_created"), DateUpdated: o.str("date_updated"),
	}}, nil
}

func (r *SipCredential) Read(ctx context.Context, req infer.ReadRequest[SipCredentialArgs, SipCredentialState]) (infer.ReadResponse[SipCredentialArgs, SipCredentialState], error) {
	c, err := clientFor(ctx)
	if err != nil {
		return infer.ReadResponse[SipCredentialArgs, SipCredentialState]{}, err
	}
	acct, list, sid, err := resolveChildScope(c, req.ID, req.State.OwnerAccountSid)
	if err != nil {
		return infer.ReadResponse[SipCredentialArgs, SipCredentialState]{}, err
	}
	o, err := c.do(ctx, "GET", "api", sipChildPath(acct, "CredentialLists", list, "Credentials", sid), nil, nil)
	if err != nil {
		if isNotFound(err) {
			if err := notFoundOnImport("sip credential", req.ID, req.Inputs, req.State); err != nil {
				return infer.ReadResponse[SipCredentialArgs, SipCredentialState]{}, err
			}
			return infer.ReadResponse[SipCredentialArgs, SipCredentialState]{}, nil
		}
		return infer.ReadResponse[SipCredentialArgs, SipCredentialState]{}, fmt.Errorf("read SIP credential %s: %w", req.ID, err)
	}
	in := req.Inputs
	st := req.State                                // keeps the password the program set: Twilio never returns it
	if in.Username == "" && acct != c.accountSid { // import
		in.AccountSid = &acct
	}
	in.CredentialListSid, in.Username = list, o.str("username")
	if in.Password == "" {
		in.Password = st.Password
	}
	st.SipCredentialArgs, st.Sid, st.OwnerAccountSid = in, sid, acct
	st.DateCreated, st.DateUpdated = o.str("date_created"), o.str("date_updated")
	return infer.ReadResponse[SipCredentialArgs, SipCredentialState]{ID: req.ID, Inputs: in, State: st}, nil
}

func (r *SipCredential) Update(ctx context.Context, req infer.UpdateRequest[SipCredentialArgs, SipCredentialState]) (infer.UpdateResponse[SipCredentialState], error) {
	st := req.State
	st.SipCredentialArgs = req.Inputs
	if req.DryRun {
		return infer.UpdateResponse[SipCredentialState]{Output: st}, nil
	}
	if req.Inputs.Password == "" {
		return infer.UpdateResponse[SipCredentialState]{}, errPasswordUnset
	}
	c, err := clientFor(ctx)
	if err != nil {
		return infer.UpdateResponse[SipCredentialState]{}, err
	}
	acct, list, sid, err := resolveChildScope(c, req.ID, req.State.OwnerAccountSid)
	if err != nil {
		return infer.UpdateResponse[SipCredentialState]{}, err
	}
	o, err := c.do(ctx, "POST", "api", sipChildPath(acct, "CredentialLists", list, "Credentials", sid), nil, url.Values{"Password": {req.Inputs.Password}})
	if err != nil {
		return infer.UpdateResponse[SipCredentialState]{}, fmt.Errorf("update SIP credential %s: %w", req.ID, err)
	}
	st.DateUpdated = o.str("date_updated")
	return infer.UpdateResponse[SipCredentialState]{Output: st}, nil
}

func (r *SipCredential) Delete(ctx context.Context, req infer.DeleteRequest[SipCredentialState]) (infer.DeleteResponse, error) {
	c, err := clientFor(ctx)
	if err != nil {
		return infer.DeleteResponse{}, err
	}
	acct, list, sid, err := resolveChildScope(c, req.ID, req.State.OwnerAccountSid)
	if err != nil {
		return infer.DeleteResponse{}, err
	}
	if _, err := c.do(ctx, "DELETE", "api", sipChildPath(acct, "CredentialLists", list, "Credentials", sid), nil, nil); err != nil && !isNotFound(err) {
		return infer.DeleteResponse{}, fmt.Errorf("delete SIP credential %s: %w", req.ID, err)
	}
	return infer.DeleteResponse{}, nil
}
