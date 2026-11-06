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

// SigningKey manages a Twilio signing key (legacy SK key used to sign Access Tokens and Client
// capability tokens).
type SigningKey struct{}

// SigningKeyArgs are the inputs of a SigningKey.
type SigningKeyArgs struct {
	FriendlyName string  `pulumi:"friendlyName"`
	AccountSid   *string `pulumi:"accountSid,optional" provider:"replaceOnChanges"`
}

func (a *SigningKeyArgs) Annotate(an infer.Annotator) {
	an.Describe(&a.FriendlyName, "Human-readable name of the key (up to 64 characters).")
	an.Describe(&a.AccountSid, "SID of the account or subaccount that owns the key. Defaults to the provider account. "+
		"Changing it replaces the resource. To import a key owned by a subaccount use the ID <accountSid>/<keySid>.")
}

// SigningKeyState is the persisted state of a SigningKey.
type SigningKeyState struct {
	SigningKeyArgs

	// Sid is the key SID (SK...) and the resource ID.
	Sid string `pulumi:"sid"`
	// OwnerAccountSid is the account that owns the key.
	OwnerAccountSid string `pulumi:"ownerAccountSid"`
	// Secret is returned by Twilio exactly once, at creation.
	Secret string `pulumi:"secret" provider:"secret"`
	// DateCreated is the RFC 2822 creation time.
	DateCreated string `pulumi:"dateCreated"`
}

func (s *SigningKeyState) Annotate(a infer.Annotator) {
	a.Describe(&s.Sid, "The signing key SID (SK...); use it as the issuer when signing tokens.")
	a.Describe(&s.OwnerAccountSid, "SID of the account that owns the key.")
	a.Describe(&s.Secret, "The signing key secret. Twilio returns it only once, at creation, so it cannot be recovered "+
		"later or after import.")
	a.Describe(&s.DateCreated, "When the key was created (RFC 2822).")
}

func (r *SigningKey) Annotate(a infer.Annotator) {
	a.Describe(&r, "A Twilio signing key (account-scoped SK key for signing Access Tokens). The secret is available only "+
		"in the creation response and is stored as a Pulumi secret.")
}

var _ infer.CustomResource[SigningKeyArgs, SigningKeyState] = (*SigningKey)(nil)
var _ infer.CustomCheck[SigningKeyArgs] = (*SigningKey)(nil)
var _ infer.CustomRead[SigningKeyArgs, SigningKeyState] = (*SigningKey)(nil)
var _ infer.CustomUpdate[SigningKeyArgs, SigningKeyState] = (*SigningKey)(nil)
var _ infer.CustomDelete[SigningKeyState] = (*SigningKey)(nil)

func (r *SigningKey) Check(ctx context.Context, req infer.CheckRequest) (infer.CheckResponse[SigningKeyArgs], error) {
	args, failures, err := infer.DefaultCheck[SigningKeyArgs](ctx, req.NewInputs)
	if err != nil || len(failures) > 0 {
		return infer.CheckResponse[SigningKeyArgs]{Inputs: args, Failures: failures}, err
	}
	if !unknowns(req.NewInputs)("friendlyName") && (args.FriendlyName == "" || len(args.FriendlyName) > 64) {
		failures = append(failures, p.CheckFailure{Property: "friendlyName", Reason: "friendlyName must be 1-64 characters"})
	}
	return infer.CheckResponse[SigningKeyArgs]{Inputs: args, Failures: failures}, nil
}

func (r *SigningKey) Create(ctx context.Context, req infer.CreateRequest[SigningKeyArgs]) (infer.CreateResponse[SigningKeyState], error) {
	a := req.Inputs
	if req.DryRun {
		return infer.CreateResponse[SigningKeyState]{ID: "", Output: SigningKeyState{SigningKeyArgs: a}}, nil
	}
	c, err := clientFor(ctx)
	if err != nil {
		return infer.CreateResponse[SigningKeyState]{}, err
	}
	acct := c.acct(a.AccountSid)
	o, err := c.do(ctx, "POST", "api", acctPath(acct, "SigningKeys", ""), nil, url.Values{"FriendlyName": {a.FriendlyName}})
	if err != nil {
		return infer.CreateResponse[SigningKeyState]{}, fmt.Errorf("create signing key: %w", err)
	}
	sid := o.str("sid")
	return infer.CreateResponse[SigningKeyState]{ID: scopedID(c, acct, sid), Output: SigningKeyState{
		SigningKeyArgs: a, Sid: sid, OwnerAccountSid: acct, Secret: o.str("secret"), DateCreated: o.str("date_created"),
	}}, nil
}

func (r *SigningKey) Read(ctx context.Context, req infer.ReadRequest[SigningKeyArgs, SigningKeyState]) (infer.ReadResponse[SigningKeyArgs, SigningKeyState], error) {
	c, err := clientFor(ctx)
	if err != nil {
		return infer.ReadResponse[SigningKeyArgs, SigningKeyState]{}, err
	}
	acct, sid := resolveScope(c, req.ID, req.State.OwnerAccountSid)
	o, err := c.do(ctx, "GET", "api", acctPath(acct, "SigningKeys", sid), nil, nil)
	if err != nil {
		if isNotFound(err) {
			return infer.ReadResponse[SigningKeyArgs, SigningKeyState]{}, nil
		}
		return infer.ReadResponse[SigningKeyArgs, SigningKeyState]{}, fmt.Errorf("read signing key %s: %w", req.ID, err)
	}
	in := req.Inputs
	st := req.State                                                           // keeps the secret recorded at creation
	if req.State.Sid == "" && in.FriendlyName == "" && acct != c.accountSid { // import
		in.AccountSid = &acct
	}
	in.FriendlyName = o.str("friendly_name")
	st.SigningKeyArgs, st.Sid, st.OwnerAccountSid, st.DateCreated = in, sid, acct, o.str("date_created")
	return infer.ReadResponse[SigningKeyArgs, SigningKeyState]{ID: req.ID, Inputs: in, State: st}, nil
}

func (r *SigningKey) Update(ctx context.Context, req infer.UpdateRequest[SigningKeyArgs, SigningKeyState]) (infer.UpdateResponse[SigningKeyState], error) {
	st := req.State
	st.SigningKeyArgs = req.Inputs
	if req.DryRun {
		return infer.UpdateResponse[SigningKeyState]{Output: st}, nil
	}
	c, err := clientFor(ctx)
	if err != nil {
		return infer.UpdateResponse[SigningKeyState]{}, err
	}
	acct, sid := resolveScope(c, req.ID, req.State.OwnerAccountSid)
	if _, err := c.do(ctx, "POST", "api", acctPath(acct, "SigningKeys", sid), nil, url.Values{"FriendlyName": {req.Inputs.FriendlyName}}); err != nil {
		return infer.UpdateResponse[SigningKeyState]{}, fmt.Errorf("update signing key %s: %w", req.ID, err)
	}
	return infer.UpdateResponse[SigningKeyState]{Output: st}, nil
}

func (r *SigningKey) Delete(ctx context.Context, req infer.DeleteRequest[SigningKeyState]) (infer.DeleteResponse, error) {
	c, err := clientFor(ctx)
	if err != nil {
		return infer.DeleteResponse{}, err
	}
	acct, sid := resolveScope(c, req.ID, req.State.OwnerAccountSid)
	if _, err := c.do(ctx, "DELETE", "api", acctPath(acct, "SigningKeys", sid), nil, nil); err != nil && !isNotFound(err) {
		return infer.DeleteResponse{}, fmt.Errorf("delete signing key %s: %w", req.ID, err)
	}
	return infer.DeleteResponse{}, nil
}
