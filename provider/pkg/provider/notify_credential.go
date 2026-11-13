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

// NotifyCredential manages a push credential of Notify: an APNs certificate, a GCM
// API key or an FCM server key that Twilio uses to send push notifications.
type NotifyCredential struct{}

// NotifyCredentialArgs are the inputs of a NotifyCredential.
type NotifyCredentialArgs struct {
	CredentialType string  `pulumi:"credentialType" provider:"replaceOnChanges"`
	FriendlyName   *string `pulumi:"friendlyName,optional"`
	Certificate    *string `pulumi:"certificate,optional"`
	PrivateKey     *string `pulumi:"privateKey,optional" provider:"secret"`
	Sandbox        *bool   `pulumi:"sandbox,optional"`
	ApiKey         *string `pulumi:"apiKey,optional" provider:"secret"`
	Secret         *string `pulumi:"secret,optional" provider:"secret"`
}

func (a *NotifyCredentialArgs) Annotate(an infer.Annotator) {
	an.Describe(&a.CredentialType, "Push service the credential is for: apn, gcm or fcm. Changing it replaces the credential.")
	an.Describe(&a.FriendlyName, "Human-readable name of the credential, up to 64 characters. Removing it leaves the name unchanged.")
	an.Describe(&a.Certificate, "APNs only: the PEM certificate. Twilio never returns it, so a change made outside Pulumi is not detected.")
	an.Describe(&a.PrivateKey, "APNs only: the PEM private key. Twilio never returns it, so a change made outside Pulumi is not detected.")
	an.Describe(&a.Sandbox, "APNs only: whether to send to the sandbox APNs. Removing it leaves the setting unchanged.")
	an.Describe(&a.ApiKey, "GCM only: the API key of the Google project. Twilio never returns it, so a change made outside Pulumi is not detected.")
	an.Describe(&a.Secret, "FCM only: the server key of the Firebase project. Twilio never returns it, so a change made outside Pulumi is not detected.")
}

// NotifyCredentialState is the persisted state of a NotifyCredential.
type NotifyCredentialState struct {
	NotifyCredentialArgs

	Sid         string `pulumi:"sid"`
	DateCreated string `pulumi:"dateCreated"`
	DateUpdated string `pulumi:"dateUpdated"`
}

func (s *NotifyCredentialState) Annotate(a infer.Annotator) {
	a.Describe(&s.Sid, "The credential SID (CR...).")
	a.Describe(&s.DateCreated, "When the credential was created (ISO 8601).")
	a.Describe(&s.DateUpdated, "When the credential was last updated (ISO 8601).")
}

func (r *NotifyCredential) Annotate(a infer.Annotator) {
	a.Describe(&r, "A push credential of Notify. The resource ID is the credential SID.")
}

var _ infer.CustomResource[NotifyCredentialArgs, NotifyCredentialState] = (*NotifyCredential)(nil)
var _ infer.CustomCheck[NotifyCredentialArgs] = (*NotifyCredential)(nil)
var _ infer.CustomDiff[NotifyCredentialArgs, NotifyCredentialState] = (*NotifyCredential)(nil)
var _ infer.CustomRead[NotifyCredentialArgs, NotifyCredentialState] = (*NotifyCredential)(nil)
var _ infer.CustomUpdate[NotifyCredentialArgs, NotifyCredentialState] = (*NotifyCredential)(nil)
var _ infer.CustomDelete[NotifyCredentialState] = (*NotifyCredential)(nil)

func (r *NotifyCredential) Check(ctx context.Context, req infer.CheckRequest) (infer.CheckResponse[NotifyCredentialArgs], error) {
	args, failures, err := infer.DefaultCheck[NotifyCredentialArgs](ctx, req.NewInputs)
	if err != nil || len(failures) > 0 {
		return infer.CheckResponse[NotifyCredentialArgs]{Inputs: args, Failures: failures}, err
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
	return infer.CheckResponse[NotifyCredentialArgs]{Inputs: args, Failures: c.failures}, nil
}

func (r *NotifyCredential) Diff(_ context.Context, req infer.DiffRequest[NotifyCredentialArgs, NotifyCredentialState]) (infer.DiffResponse, error) {
	o, n := req.State.NotifyCredentialArgs, req.Inputs
	d := trDiffSet{}
	d.replace("credentialType", o.CredentialType, n.CredentialType)
	d.strDef(notifyCredentialDefaults, "friendlyName", o.FriendlyName, n.FriendlyName)
	d.str("certificate", o.Certificate, n.Certificate)
	d.str("privateKey", o.PrivateKey, n.PrivateKey)
	d.boolDef(notifyCredentialDefaults, "sandbox", o.Sandbox, n.Sandbox)
	d.str("apiKey", o.ApiKey, n.ApiKey)
	d.str("secret", o.Secret, n.Secret)
	return d.response(), nil
}

func (a NotifyCredentialArgs) form() url.Values {
	form := url.Values{}
	setS(form, "FriendlyName", a.FriendlyName)
	setS(form, "Certificate", a.Certificate)
	setS(form, "PrivateKey", a.PrivateKey)
	setB(form, "Sandbox", a.Sandbox)
	setS(form, "ApiKey", a.ApiKey)
	setS(form, "Secret", a.Secret)
	return form
}

func (r *NotifyCredential) Create(ctx context.Context, req infer.CreateRequest[NotifyCredentialArgs]) (infer.CreateResponse[NotifyCredentialState], error) {
	type resp = infer.CreateResponse[NotifyCredentialState]
	a := req.Inputs
	if req.DryRun {
		return resp{Output: NotifyCredentialState{NotifyCredentialArgs: a}}, nil
	}
	c, err := clientFor(ctx)
	if err != nil {
		return resp{}, err
	}
	form := a.form()
	form.Set("Type", a.CredentialType)
	o, err := c.do(ctx, "POST", notifyProduct, notifyPath("Credentials", ""), nil, form)
	if err != nil {
		return resp{}, fmt.Errorf("create Notify credential: %w", err)
	}
	st := NotifyCredentialState{NotifyCredentialArgs: a}
	st.Sid, st.DateCreated, st.DateUpdated = trSid(o)
	return resp{ID: st.Sid, Output: st}, nil
}

func (r *NotifyCredential) Read(ctx context.Context, req infer.ReadRequest[NotifyCredentialArgs, NotifyCredentialState]) (infer.ReadResponse[NotifyCredentialArgs, NotifyCredentialState], error) {
	type resp = infer.ReadResponse[NotifyCredentialArgs, NotifyCredentialState]
	c, err := clientFor(ctx)
	if err != nil {
		return resp{}, err
	}
	o, err := c.do(ctx, "GET", notifyProduct, notifyPath("Credentials", req.ID), nil, nil)
	if err != nil {
		if isNotFound(err) {
			if err := notFoundOnImport("Notify credential", req.ID, req.Inputs, req.State); err != nil {
				return resp{}, err
			}
			return resp{}, nil
		}
		return resp{}, fmt.Errorf("read Notify credential %s: %w", req.ID, err)
	}
	in := req.Inputs
	imported := in.CredentialType == ""
	in.CredentialType = o.str("type")
	in.FriendlyName = trStr(in.FriendlyName, o.optStr("friendly_name"), notifyCredentialDefaults.str("friendlyName"), imported)
	// The certificate, keys and secret are never returned; Twilio reports the sandbox flag for APNs only.
	if in.CredentialType == "apn" {
		in.Sandbox = trBool(in.Sandbox, o.optBool("sandbox"), notifyCredentialDefaults.boolean("sandbox"), imported)
	}
	st := NotifyCredentialState{NotifyCredentialArgs: in}
	st.Sid, st.DateCreated, st.DateUpdated = trSid(o)
	return resp{ID: req.ID, Inputs: in, State: st}, nil
}

func (r *NotifyCredential) Update(ctx context.Context, req infer.UpdateRequest[NotifyCredentialArgs, NotifyCredentialState]) (infer.UpdateResponse[NotifyCredentialState], error) {
	type resp = infer.UpdateResponse[NotifyCredentialState]
	st := req.State
	st.NotifyCredentialArgs = req.Inputs
	if req.DryRun {
		return resp{Output: st}, nil
	}
	c, err := clientFor(ctx)
	if err != nil {
		return resp{}, err
	}
	o, err := c.do(ctx, "POST", notifyProduct, notifyPath("Credentials", req.ID), nil, req.Inputs.form())
	if err != nil {
		return resp{}, fmt.Errorf("update Notify credential %s: %w", req.ID, err)
	}
	st.DateUpdated = o.str("date_updated")
	return resp{Output: st}, nil
}

func (r *NotifyCredential) Delete(ctx context.Context, req infer.DeleteRequest[NotifyCredentialState]) (infer.DeleteResponse, error) {
	c, err := clientFor(ctx)
	if err != nil {
		return infer.DeleteResponse{}, err
	}
	if _, err := c.do(ctx, "DELETE", notifyProduct, notifyPath("Credentials", req.ID), nil, nil); err != nil && !isNotFound(err) {
		return infer.DeleteResponse{}, fmt.Errorf("delete Notify credential %s: %w", req.ID, err)
	}
	return infer.DeleteResponse{}, nil
}

// notifyCredentialDefaults are the Twilio defaults of this resource, shared by Read (import adoption) and Diff.
var notifyCredentialDefaults = trDefaults{
	"friendlyName": "",
	"sandbox":      false,
}
