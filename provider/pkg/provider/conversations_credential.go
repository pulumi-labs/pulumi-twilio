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

	"github.com/pulumi/pulumi-go-provider/infer"
)

// ConversationsCredential manages a push credential of Conversations: an APNs certificate, a GCM
// API key or an FCM server key that Twilio uses to send push notifications.
type ConversationsCredential struct{}

// ConversationsCredentialArgs are the inputs of a ConversationsCredential.
type ConversationsCredentialArgs struct {
	CredentialType string  `pulumi:"credentialType" provider:"replaceOnChanges"`
	FriendlyName   *string `pulumi:"friendlyName,optional"`
	Certificate    *string `pulumi:"certificate,optional"`
	PrivateKey     *string `pulumi:"privateKey,optional" provider:"secret"`
	Sandbox        *bool   `pulumi:"sandbox,optional"`
	ApiKey         *string `pulumi:"apiKey,optional" provider:"secret"`
	Secret         *string `pulumi:"secret,optional" provider:"secret"`
}

func (a *ConversationsCredentialArgs) Annotate(an infer.Annotator) {
	an.Describe(&a.CredentialType, "Push service the credential is for: apn, gcm or fcm. Changing it replaces the credential.")
	an.Describe(&a.FriendlyName, "Human-readable name of the credential, up to 64 characters. Removing it leaves the name unchanged.")
	an.Describe(&a.Certificate, "APNs only: the PEM certificate. Twilio never returns it, so a change made outside Pulumi is not detected.")
	an.Describe(&a.PrivateKey, "APNs only: the PEM private key. Twilio never returns it, so a change made outside Pulumi is not detected.")
	an.Describe(&a.Sandbox, "APNs only: whether to send to the sandbox APNs. Removing it leaves the setting unchanged.")
	an.Describe(&a.ApiKey, "GCM only: the API key of the Google project. Twilio never returns it, so a change made outside Pulumi is not detected.")
	an.Describe(&a.Secret, "FCM only: the server key of the Firebase project. Twilio never returns it, so a change made outside Pulumi is not detected.")
}

// ConversationsCredentialState is the persisted state of a ConversationsCredential.
type ConversationsCredentialState struct {
	ConversationsCredentialArgs

	Sid         string `pulumi:"sid"`
	DateCreated string `pulumi:"dateCreated"`
	DateUpdated string `pulumi:"dateUpdated"`
}

func (s *ConversationsCredentialState) Annotate(a infer.Annotator) {
	a.Describe(&s.Sid, "The credential SID (CR...).")
	a.Describe(&s.DateCreated, "When the credential was created (ISO 8601).")
	a.Describe(&s.DateUpdated, "When the credential was last updated (ISO 8601).")
}

func (r *ConversationsCredential) Annotate(a infer.Annotator) {
	a.Describe(&r, "A push credential of Conversations. The resource ID is the credential SID.")
}

var _ infer.CustomResource[ConversationsCredentialArgs, ConversationsCredentialState] = (*ConversationsCredential)(nil)
var _ infer.CustomCheck[ConversationsCredentialArgs] = (*ConversationsCredential)(nil)
var _ infer.CustomDiff[ConversationsCredentialArgs, ConversationsCredentialState] = (*ConversationsCredential)(nil)
var _ infer.CustomRead[ConversationsCredentialArgs, ConversationsCredentialState] = (*ConversationsCredential)(nil)
var _ infer.CustomUpdate[ConversationsCredentialArgs, ConversationsCredentialState] = (*ConversationsCredential)(nil)
var _ infer.CustomDelete[ConversationsCredentialState] = (*ConversationsCredential)(nil)

func (r *ConversationsCredential) Check(ctx context.Context, req infer.CheckRequest) (infer.CheckResponse[ConversationsCredentialArgs], error) {
	args, failures, err := infer.DefaultCheck[ConversationsCredentialArgs](ctx, req.NewInputs)
	if err != nil || len(failures) > 0 {
		return infer.CheckResponse[ConversationsCredentialArgs]{Inputs: args, Failures: failures}, err
	}
	c := &checker{unk: unknowns(req.NewInputs)}
	c.oneOf("credentialType", args.CredentialType, []string{"apn", "gcm", "fcm"})
	if v := args.FriendlyName; v != nil && !c.unk("friendlyName") && len(*v) > 64 {
		c.fail("friendlyName", "friendlyName must be at most 64 characters")
	}
	if !c.unk("credentialType") {
		only := func(prop string, set bool, want string) {
			if set && args.CredentialType != want && !c.unk(prop) {
				c.fail(prop, prop+" applies to "+want+" credentials only")
			}
		}
		only("certificate", args.Certificate != nil, "apn")
		only("privateKey", args.PrivateKey != nil, "apn")
		only("sandbox", args.Sandbox != nil, "apn")
		only("apiKey", args.ApiKey != nil, "gcm")
		only("secret", args.Secret != nil, "fcm")
	}
	return infer.CheckResponse[ConversationsCredentialArgs]{Inputs: args, Failures: c.failures}, nil
}

func (r *ConversationsCredential) Diff(_ context.Context, req infer.DiffRequest[ConversationsCredentialArgs, ConversationsCredentialState]) (infer.DiffResponse, error) {
	o, n := req.State.ConversationsCredentialArgs, req.Inputs
	d := trDiffSet{}
	d.replace("credentialType", o.CredentialType, n.CredentialType)
	d.strDef(conversationsCredentialDefaults, "friendlyName", o.FriendlyName, n.FriendlyName)
	d.str("certificate", o.Certificate, n.Certificate)
	d.str("privateKey", o.PrivateKey, n.PrivateKey)
	d.boolDef(conversationsCredentialDefaults, "sandbox", o.Sandbox, n.Sandbox)
	d.str("apiKey", o.ApiKey, n.ApiKey)
	d.str("secret", o.Secret, n.Secret)
	return d.response(), nil
}

func (a ConversationsCredentialArgs) form() url.Values {
	form := url.Values{}
	setS(form, "FriendlyName", a.FriendlyName)
	setS(form, "Certificate", a.Certificate)
	setS(form, "PrivateKey", a.PrivateKey)
	setB(form, "Sandbox", a.Sandbox)
	setS(form, "ApiKey", a.ApiKey)
	setS(form, "Secret", a.Secret)
	return form
}

func (r *ConversationsCredential) Create(ctx context.Context, req infer.CreateRequest[ConversationsCredentialArgs]) (infer.CreateResponse[ConversationsCredentialState], error) {
	type resp = infer.CreateResponse[ConversationsCredentialState]
	a := req.Inputs
	if req.DryRun {
		return resp{Output: ConversationsCredentialState{ConversationsCredentialArgs: a}}, nil
	}
	c, err := clientFor(ctx)
	if err != nil {
		return resp{}, err
	}
	form := a.form()
	form.Set("Type", a.CredentialType)
	o, err := c.do(ctx, "POST", cvProduct, cvPath("", "Credentials", ""), nil, form)
	if err != nil {
		return resp{}, fmt.Errorf("create Conversations credential: %w", err)
	}
	st := ConversationsCredentialState{ConversationsCredentialArgs: a}
	st.Sid, st.DateCreated, st.DateUpdated = trSid(o)
	return resp{ID: st.Sid, Output: st}, nil
}

func (r *ConversationsCredential) Read(ctx context.Context, req infer.ReadRequest[ConversationsCredentialArgs, ConversationsCredentialState]) (infer.ReadResponse[ConversationsCredentialArgs, ConversationsCredentialState], error) {
	type resp = infer.ReadResponse[ConversationsCredentialArgs, ConversationsCredentialState]
	c, err := clientFor(ctx)
	if err != nil {
		return resp{}, err
	}
	o, err := c.do(ctx, "GET", cvProduct, cvPath("", "Credentials", req.ID), nil, nil)
	if err != nil {
		if isNotFound(err) {
			if err := notFoundOnImport("Conversations credential", req.ID, req.Inputs, req.State); err != nil {
				return resp{}, err
			}
			return resp{}, nil
		}
		return resp{}, fmt.Errorf("read Conversations credential %s: %w", req.ID, err)
	}
	in := req.Inputs
	imported := in.CredentialType == ""
	in.CredentialType = o.str("type")
	in.FriendlyName = trStr(in.FriendlyName, o.optStr("friendly_name"), conversationsCredentialDefaults.str("friendlyName"), imported)
	// The certificate, keys and secret are never returned; Twilio reports the sandbox flag for APNs only.
	if in.CredentialType == "apn" {
		in.Sandbox = trBool(in.Sandbox, o.optBool("sandbox"), conversationsCredentialDefaults.boolean("sandbox"), imported)
	}
	st := ConversationsCredentialState{ConversationsCredentialArgs: in}
	st.Sid, st.DateCreated, st.DateUpdated = trSid(o)
	return resp{ID: req.ID, Inputs: in, State: st}, nil
}

func (r *ConversationsCredential) Update(ctx context.Context, req infer.UpdateRequest[ConversationsCredentialArgs, ConversationsCredentialState]) (infer.UpdateResponse[ConversationsCredentialState], error) {
	type resp = infer.UpdateResponse[ConversationsCredentialState]
	st := req.State
	st.ConversationsCredentialArgs = req.Inputs
	if req.DryRun {
		return resp{Output: st}, nil
	}
	c, err := clientFor(ctx)
	if err != nil {
		return resp{}, err
	}
	o, err := c.do(ctx, "POST", cvProduct, cvPath("", "Credentials", req.ID), nil, req.Inputs.form())
	if err != nil {
		return resp{}, fmt.Errorf("update Conversations credential %s: %w", req.ID, err)
	}
	st.DateUpdated = o.str("date_updated")
	return resp{Output: st}, nil
}

func (r *ConversationsCredential) Delete(ctx context.Context, req infer.DeleteRequest[ConversationsCredentialState]) (infer.DeleteResponse, error) {
	c, err := clientFor(ctx)
	if err != nil {
		return infer.DeleteResponse{}, err
	}
	if _, err := c.do(ctx, "DELETE", cvProduct, cvPath("", "Credentials", req.ID), nil, nil); err != nil && !isNotFound(err) {
		return infer.DeleteResponse{}, fmt.Errorf("delete Conversations credential %s: %w", req.ID, err)
	}
	return infer.DeleteResponse{}, nil
}

// conversationsCredentialDefaults are the Twilio defaults of this resource, shared by Read (import adoption) and Diff.
var conversationsCredentialDefaults = trDefaults{
	"friendlyName": "",
	"sandbox":      false,
}
