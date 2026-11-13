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
	"regexp"

	"github.com/pulumi/pulumi-go-provider/infer"
)

// Accounts v1 (accounts.twilio.com): the AWS and public key credentials of the account, and the
// account's Global SafeList. The messaging geo permissions are read-only and have a lookup only.

const accountsProduct = "accounts"

var (
	accountsCredentialSidRE = regexp.MustCompile(`^CR[0-9a-fA-F]{32}$`)
	accountsAccountSidRE    = regexp.MustCompile(`^AC[0-9a-fA-F]{32}$`)
)

// accountsCredentialPath is /v1/Credentials/<kind>[/sid]; kind is AWS or PublicKeys.
func accountsCredentialPath(kind, sid string) string {
	if sid == "" {
		return "/v1/Credentials/" + kind
	}
	return "/v1/Credentials/" + kind + "/" + url.PathEscape(sid)
}

// accountsSafeListPath is /v1/SafeList/Numbers. The number travels in the query string, where the
// plus sign is escaped as %2B.
const accountsSafeListPath = "/v1/SafeList/Numbers"

// accountsCredentialDefaults are the Twilio defaults shared by Read (import adoption) and Diff.
var accountsCredentialDefaults = trDefaults{
	"friendlyName": "",
}

func accountsCheckCommon(c *checker, friendlyName, accountSid *string) {
	if friendlyName != nil && !c.unk("friendlyName") {
		c.maxLen("friendlyName", *friendlyName, 64)
	}
	if accountSid != nil {
		c.sid("accountSid", *accountSid, accountsAccountSidRE, "subaccount (AC followed by 32 hex characters)")
	}
}

// accountsReplaceIfChanged records a replacement when an optional input that has a stored value is
// changed. A stored value that is unset (an imported resource, where Twilio never returns the
// value) is adopted: stating the value is no change, because nothing can tell the two apart.
func accountsReplaceIfChanged(d trDiffSet, name string, o, n *string) {
	if o != nil && n != nil && *o != *n {
		d.replace(name, *o, *n)
	}
}

// accountsReplaceAccount records a replacement when the program names a subaccount that is not the
// one the credential has. Removing the input is not a change.
func accountsReplaceAccount(d trDiffSet, o, n *string) {
	if n != nil && (o == nil || *o != *n) {
		d.replace("accountSid", derefOr(o, ""), *n)
	}
}

// accountsAdoptAccount returns the account an import adopts: the one Twilio reports for the credential,
// the provider's own account included, so that a program stating either is no change. A refresh keeps
// the program's setting.
func accountsAdoptAccount(in *string, o obj, imported bool) *string {
	if in != nil || !imported {
		return in
	}
	return nonEmpty(o.optStr("account_sid"))
}

// ------------------------------------------------------------------ AwsCredential

// AwsCredential stores AWS access keys with Twilio, for the products that write to the account's S3
// bucket (Video recordings and compositions, Voice Intelligence).
type AwsCredential struct{}

// AwsCredentialArgs are the inputs of an AwsCredential.
type AwsCredentialArgs struct {
	Credentials  *string `pulumi:"credentials,optional" provider:"secret"`
	FriendlyName *string `pulumi:"friendlyName,optional"`
	AccountSid   *string `pulumi:"accountSid,optional"`
}

func (a *AwsCredentialArgs) Annotate(an infer.Annotator) {
	an.Describe(&a.Credentials, "The AWS access keys as <AWS_ACCESS_KEY_ID>:<AWS_SECRET_ACCESS_KEY>. Required to create the credential. Twilio never returns it, so a change made outside Pulumi is not detected, and a credential imported by SID has none until the program states it (stating it after an import is no change). Changing it replaces the credential, because Twilio cannot update the keys.")
	an.Describe(&a.FriendlyName, "Human-readable name of the credential, up to 64 characters. Removing it leaves the name unchanged.")
	an.Describe(&a.AccountSid, "SID (AC...) of a subaccount of the provider's account to associate the credential with. Changing it replaces the credential. An import adopts the account Twilio reports.")
}

// AwsCredentialState is the persisted state of an AwsCredential.
type AwsCredentialState struct {
	AwsCredentialArgs

	Sid         string `pulumi:"sid"`
	DateCreated string `pulumi:"dateCreated"`
	DateUpdated string `pulumi:"dateUpdated"`
}

func (s *AwsCredentialState) Annotate(a infer.Annotator) {
	a.Describe(&s.Sid, "The credential SID (CR...).")
	a.Describe(&s.DateCreated, "When the credential was created (ISO 8601).")
	a.Describe(&s.DateUpdated, "When the credential was last updated (ISO 8601).")
}

func (r *AwsCredential) Annotate(a infer.Annotator) {
	a.Describe(&r, "AWS access keys stored with Twilio (accounts.twilio.com /v1/Credentials/AWS). The resource ID is the credential SID. Only the friendly name updates in place; the keys and the subaccount replace the credential.")
}

var _ infer.CustomResource[AwsCredentialArgs, AwsCredentialState] = (*AwsCredential)(nil)
var _ infer.CustomCheck[AwsCredentialArgs] = (*AwsCredential)(nil)
var _ infer.CustomDiff[AwsCredentialArgs, AwsCredentialState] = (*AwsCredential)(nil)
var _ infer.CustomRead[AwsCredentialArgs, AwsCredentialState] = (*AwsCredential)(nil)
var _ infer.CustomUpdate[AwsCredentialArgs, AwsCredentialState] = (*AwsCredential)(nil)
var _ infer.CustomDelete[AwsCredentialState] = (*AwsCredential)(nil)

func (r *AwsCredential) Check(ctx context.Context, req infer.CheckRequest) (infer.CheckResponse[AwsCredentialArgs], error) {
	args, failures, err := infer.DefaultCheck[AwsCredentialArgs](ctx, req.NewInputs)
	if err != nil || len(failures) > 0 {
		return infer.CheckResponse[AwsCredentialArgs]{Inputs: args, Failures: failures}, err
	}
	c := &checker{unk: unknowns(req.NewInputs)}
	accountsCheckCommon(c, args.FriendlyName, args.AccountSid)
	return infer.CheckResponse[AwsCredentialArgs]{Inputs: args, Failures: c.failures}, nil
}

func (r *AwsCredential) Diff(_ context.Context, req infer.DiffRequest[AwsCredentialArgs, AwsCredentialState]) (infer.DiffResponse, error) {
	o, n := req.State.AwsCredentialArgs, req.Inputs
	d := trDiffSet{}
	accountsReplaceIfChanged(d, "credentials", o.Credentials, n.Credentials)
	accountsReplaceAccount(d, o.AccountSid, n.AccountSid)
	d.strDef(accountsCredentialDefaults, "friendlyName", o.FriendlyName, n.FriendlyName)
	return d.response(), nil
}

func (r *AwsCredential) Create(ctx context.Context, req infer.CreateRequest[AwsCredentialArgs]) (infer.CreateResponse[AwsCredentialState], error) {
	type resp = infer.CreateResponse[AwsCredentialState]
	a := req.Inputs
	if req.DryRun {
		return resp{Output: AwsCredentialState{AwsCredentialArgs: a}}, nil
	}
	if a.Credentials == nil || *a.Credentials == "" {
		return resp{}, fmt.Errorf("create AWS credential: credentials is required (<AWS_ACCESS_KEY_ID>:<AWS_SECRET_ACCESS_KEY>)")
	}
	c, err := clientFor(ctx)
	if err != nil {
		return resp{}, err
	}
	form := url.Values{"Credentials": {*a.Credentials}}
	setS(form, "FriendlyName", a.FriendlyName)
	setS(form, "AccountSid", a.AccountSid)
	o, err := c.do(ctx, "POST", accountsProduct, accountsCredentialPath("AWS", ""), nil, form)
	if err != nil {
		return resp{}, fmt.Errorf("create AWS credential: %w", err)
	}
	st := AwsCredentialState{AwsCredentialArgs: a}
	st.Sid, st.DateCreated, st.DateUpdated = trSid(o)
	return resp{ID: st.Sid, Output: st}, nil
}

func (r *AwsCredential) Read(ctx context.Context, req infer.ReadRequest[AwsCredentialArgs, AwsCredentialState]) (infer.ReadResponse[AwsCredentialArgs, AwsCredentialState], error) {
	type resp = infer.ReadResponse[AwsCredentialArgs, AwsCredentialState]
	c, err := clientFor(ctx)
	if err != nil {
		return resp{}, err
	}
	o, err := c.do(ctx, "GET", accountsProduct, accountsCredentialPath("AWS", req.ID), nil, nil)
	if err != nil {
		if isNotFound(err) {
			if err := notFoundOnImport("AWS credential", req.ID, req.Inputs, req.State); err != nil {
				return resp{}, err
			}
			return resp{}, nil
		}
		return resp{}, fmt.Errorf("read AWS credential %s: %w", req.ID, err)
	}
	in := req.Inputs
	imported := req.State.Sid == ""
	in.FriendlyName = trStr(in.FriendlyName, o.optStr("friendly_name"), accountsCredentialDefaults.str("friendlyName"), imported)
	in.AccountSid = accountsAdoptAccount(in.AccountSid, o, imported)
	// The keys are never returned: they stay as the program or the prior state has them.
	st := AwsCredentialState{AwsCredentialArgs: in}
	st.Sid, st.DateCreated, st.DateUpdated = trSid(o)
	return resp{ID: req.ID, Inputs: in, State: st}, nil
}

func (r *AwsCredential) Update(ctx context.Context, req infer.UpdateRequest[AwsCredentialArgs, AwsCredentialState]) (infer.UpdateResponse[AwsCredentialState], error) {
	type resp = infer.UpdateResponse[AwsCredentialState]
	st := req.State
	st.AwsCredentialArgs = req.Inputs
	if req.DryRun {
		return resp{Output: st}, nil
	}
	c, err := clientFor(ctx)
	if err != nil {
		return resp{}, err
	}
	form := url.Values{}
	setS(form, "FriendlyName", req.Inputs.FriendlyName)
	o, err := c.do(ctx, "POST", accountsProduct, accountsCredentialPath("AWS", req.ID), nil, form)
	if err != nil {
		return resp{}, fmt.Errorf("update AWS credential %s: %w", req.ID, err)
	}
	st.DateUpdated = o.str("date_updated")
	return resp{Output: st}, nil
}

func (r *AwsCredential) Delete(ctx context.Context, req infer.DeleteRequest[AwsCredentialState]) (infer.DeleteResponse, error) {
	c, err := clientFor(ctx)
	if err != nil {
		return infer.DeleteResponse{}, err
	}
	if _, err := c.do(ctx, "DELETE", accountsProduct, accountsCredentialPath("AWS", req.ID), nil, nil); err != nil && !isNotFound(err) {
		return infer.DeleteResponse{}, fmt.Errorf("delete AWS credential %s: %w", req.ID, err)
	}
	return infer.DeleteResponse{}, nil
}

// ------------------------------------------------------------ PublicKeyCredential

// PublicKeyCredential stores an RSA public key with Twilio, used to encrypt Voice call recordings
// (and other products that take a public key credential).
type PublicKeyCredential struct{}

// PublicKeyCredentialArgs are the inputs of a PublicKeyCredential.
type PublicKeyCredentialArgs struct {
	PublicKey    *string `pulumi:"publicKey,optional"`
	FriendlyName *string `pulumi:"friendlyName,optional"`
	AccountSid   *string `pulumi:"accountSid,optional"`
}

func (a *PublicKeyCredentialArgs) Annotate(an infer.Annotator) {
	an.Describe(&a.PublicKey, "The PEM encoded public key (-----BEGIN PUBLIC KEY-----). Required to create the credential. Twilio never returns it, so a change made outside Pulumi is not detected, and a credential imported by SID has none until the program states it (stating it after an import is no change). Changing it replaces the credential, because Twilio cannot update the key.")
	an.Describe(&a.FriendlyName, "Human-readable name of the credential, up to 64 characters. Removing it leaves the name unchanged.")
	an.Describe(&a.AccountSid, "SID (AC...) of a subaccount of the provider's account to associate the credential with. Changing it replaces the credential. An import adopts the account Twilio reports.")
}

// PublicKeyCredentialState is the persisted state of a PublicKeyCredential.
type PublicKeyCredentialState struct {
	PublicKeyCredentialArgs

	Sid         string `pulumi:"sid"`
	DateCreated string `pulumi:"dateCreated"`
	DateUpdated string `pulumi:"dateUpdated"`
}

func (s *PublicKeyCredentialState) Annotate(a infer.Annotator) {
	a.Describe(&s.Sid, "The credential SID (CR...).")
	a.Describe(&s.DateCreated, "When the credential was created (ISO 8601).")
	a.Describe(&s.DateUpdated, "When the credential was last updated (ISO 8601).")
}

func (r *PublicKeyCredential) Annotate(a infer.Annotator) {
	a.Describe(&r, "A public key stored with Twilio (accounts.twilio.com /v1/Credentials/PublicKeys). The resource ID is the credential SID. Only the friendly name updates in place; the key and the subaccount replace the credential.")
}

var _ infer.CustomResource[PublicKeyCredentialArgs, PublicKeyCredentialState] = (*PublicKeyCredential)(nil)
var _ infer.CustomCheck[PublicKeyCredentialArgs] = (*PublicKeyCredential)(nil)
var _ infer.CustomDiff[PublicKeyCredentialArgs, PublicKeyCredentialState] = (*PublicKeyCredential)(nil)
var _ infer.CustomRead[PublicKeyCredentialArgs, PublicKeyCredentialState] = (*PublicKeyCredential)(nil)
var _ infer.CustomUpdate[PublicKeyCredentialArgs, PublicKeyCredentialState] = (*PublicKeyCredential)(nil)
var _ infer.CustomDelete[PublicKeyCredentialState] = (*PublicKeyCredential)(nil)

func (r *PublicKeyCredential) Check(ctx context.Context, req infer.CheckRequest) (infer.CheckResponse[PublicKeyCredentialArgs], error) {
	args, failures, err := infer.DefaultCheck[PublicKeyCredentialArgs](ctx, req.NewInputs)
	if err != nil || len(failures) > 0 {
		return infer.CheckResponse[PublicKeyCredentialArgs]{Inputs: args, Failures: failures}, err
	}
	c := &checker{unk: unknowns(req.NewInputs)}
	accountsCheckCommon(c, args.FriendlyName, args.AccountSid)
	return infer.CheckResponse[PublicKeyCredentialArgs]{Inputs: args, Failures: c.failures}, nil
}

func (r *PublicKeyCredential) Diff(_ context.Context, req infer.DiffRequest[PublicKeyCredentialArgs, PublicKeyCredentialState]) (infer.DiffResponse, error) {
	o, n := req.State.PublicKeyCredentialArgs, req.Inputs
	d := trDiffSet{}
	accountsReplaceIfChanged(d, "publicKey", o.PublicKey, n.PublicKey)
	accountsReplaceAccount(d, o.AccountSid, n.AccountSid)
	d.strDef(publicKeyCredentialDefaults, "friendlyName", o.FriendlyName, n.FriendlyName)
	return d.response(), nil
}

func (r *PublicKeyCredential) Create(ctx context.Context, req infer.CreateRequest[PublicKeyCredentialArgs]) (infer.CreateResponse[PublicKeyCredentialState], error) {
	type resp = infer.CreateResponse[PublicKeyCredentialState]
	a := req.Inputs
	if req.DryRun {
		return resp{Output: PublicKeyCredentialState{PublicKeyCredentialArgs: a}}, nil
	}
	if a.PublicKey == nil || *a.PublicKey == "" {
		return resp{}, fmt.Errorf("create public key credential: publicKey is required")
	}
	c, err := clientFor(ctx)
	if err != nil {
		return resp{}, err
	}
	form := url.Values{"PublicKey": {*a.PublicKey}}
	setS(form, "FriendlyName", a.FriendlyName)
	setS(form, "AccountSid", a.AccountSid)
	o, err := c.do(ctx, "POST", accountsProduct, accountsCredentialPath("PublicKeys", ""), nil, form)
	if err != nil {
		return resp{}, fmt.Errorf("create public key credential: %w", err)
	}
	st := PublicKeyCredentialState{PublicKeyCredentialArgs: a}
	st.Sid, st.DateCreated, st.DateUpdated = trSid(o)
	return resp{ID: st.Sid, Output: st}, nil
}

func (r *PublicKeyCredential) Read(ctx context.Context, req infer.ReadRequest[PublicKeyCredentialArgs, PublicKeyCredentialState]) (infer.ReadResponse[PublicKeyCredentialArgs, PublicKeyCredentialState], error) {
	type resp = infer.ReadResponse[PublicKeyCredentialArgs, PublicKeyCredentialState]
	c, err := clientFor(ctx)
	if err != nil {
		return resp{}, err
	}
	o, err := c.do(ctx, "GET", accountsProduct, accountsCredentialPath("PublicKeys", req.ID), nil, nil)
	if err != nil {
		if isNotFound(err) {
			if err := notFoundOnImport("public key credential", req.ID, req.Inputs, req.State); err != nil {
				return resp{}, err
			}
			return resp{}, nil
		}
		return resp{}, fmt.Errorf("read public key credential %s: %w", req.ID, err)
	}
	in := req.Inputs
	imported := req.State.Sid == ""
	in.FriendlyName = trStr(in.FriendlyName, o.optStr("friendly_name"), publicKeyCredentialDefaults.str("friendlyName"), imported)
	in.AccountSid = accountsAdoptAccount(in.AccountSid, o, imported)
	st := PublicKeyCredentialState{PublicKeyCredentialArgs: in}
	st.Sid, st.DateCreated, st.DateUpdated = trSid(o)
	return resp{ID: req.ID, Inputs: in, State: st}, nil
}

func (r *PublicKeyCredential) Update(ctx context.Context, req infer.UpdateRequest[PublicKeyCredentialArgs, PublicKeyCredentialState]) (infer.UpdateResponse[PublicKeyCredentialState], error) {
	type resp = infer.UpdateResponse[PublicKeyCredentialState]
	st := req.State
	st.PublicKeyCredentialArgs = req.Inputs
	if req.DryRun {
		return resp{Output: st}, nil
	}
	c, err := clientFor(ctx)
	if err != nil {
		return resp{}, err
	}
	form := url.Values{}
	setS(form, "FriendlyName", req.Inputs.FriendlyName)
	o, err := c.do(ctx, "POST", accountsProduct, accountsCredentialPath("PublicKeys", req.ID), nil, form)
	if err != nil {
		return resp{}, fmt.Errorf("update public key credential %s: %w", req.ID, err)
	}
	st.DateUpdated = o.str("date_updated")
	return resp{Output: st}, nil
}

func (r *PublicKeyCredential) Delete(ctx context.Context, req infer.DeleteRequest[PublicKeyCredentialState]) (infer.DeleteResponse, error) {
	c, err := clientFor(ctx)
	if err != nil {
		return infer.DeleteResponse{}, err
	}
	if _, err := c.do(ctx, "DELETE", accountsProduct, accountsCredentialPath("PublicKeys", req.ID), nil, nil); err != nil && !isNotFound(err) {
		return infer.DeleteResponse{}, fmt.Errorf("delete public key credential %s: %w", req.ID, err)
	}
	return infer.DeleteResponse{}, nil
}

// publicKeyCredentialDefaults mirrors accountsCredentialDefaults for the public key resource.
var publicKeyCredentialDefaults = trDefaults{
	"friendlyName": "",
}

// ------------------------------------------------------------ AccountSafeListNumber

// AccountSafeListNumber puts a phone number (or a phone number prefix) on the account's Global
// SafeList (accounts.twilio.com), which exempts it from Twilio's SMS fraud blocks. It is a different
// API from the Verify SafeList (VerifySafeListNumber, verify.twilio.com).
type AccountSafeListNumber struct{}

// AccountSafeListNumberArgs are the inputs of an AccountSafeListNumber.
type AccountSafeListNumberArgs struct {
	PhoneNumber string `pulumi:"phoneNumber" provider:"replaceOnChanges"`
}

func (a *AccountSafeListNumberArgs) Annotate(an infer.Annotator) {
	an.Describe(&a.PhoneNumber, "The phone number, or the leading digits of a range of numbers, to add to the Global SafeList, in E.164 format (+14155550100). Changing it replaces the resource.")
}

// AccountSafeListNumberState is the persisted state of an AccountSafeListNumber.
type AccountSafeListNumberState struct {
	AccountSafeListNumberArgs

	// Sid is the SafeList entry SID (GN...).
	Sid string `pulumi:"sid"`
}

func (s *AccountSafeListNumberState) Annotate(a infer.Annotator) {
	a.Describe(&s.Sid, "The SafeList entry SID, as Twilio reports it.")
}

func (r *AccountSafeListNumber) Annotate(a infer.Annotator) {
	a.Describe(&r, "A phone number on the account's Global SafeList (accounts.twilio.com /v1/SafeList/Numbers). The resource ID is the phone number in E.164 format. Twilio has no update call, so changing the number replaces the entry. This is not the Verify SafeList (see VerifySafeListNumber).")
}

var _ infer.CustomResource[AccountSafeListNumberArgs, AccountSafeListNumberState] = (*AccountSafeListNumber)(nil)
var _ infer.CustomCheck[AccountSafeListNumberArgs] = (*AccountSafeListNumber)(nil)
var _ infer.CustomRead[AccountSafeListNumberArgs, AccountSafeListNumberState] = (*AccountSafeListNumber)(nil)
var _ infer.CustomDelete[AccountSafeListNumberState] = (*AccountSafeListNumber)(nil)

func (r *AccountSafeListNumber) Check(ctx context.Context, req infer.CheckRequest) (infer.CheckResponse[AccountSafeListNumberArgs], error) {
	args, failures, err := infer.DefaultCheck[AccountSafeListNumberArgs](ctx, req.NewInputs)
	if err != nil || len(failures) > 0 {
		return infer.CheckResponse[AccountSafeListNumberArgs]{Inputs: args, Failures: failures}, err
	}
	c := &checker{unk: unknowns(req.NewInputs)}
	if !c.unk("phoneNumber") && !e164RE.MatchString(args.PhoneNumber) {
		c.fail("phoneNumber", "phoneNumber must be in E.164 format, for example +14155550100")
	}
	return infer.CheckResponse[AccountSafeListNumberArgs]{Inputs: args, Failures: c.failures}, nil
}

func (r *AccountSafeListNumber) Create(ctx context.Context, req infer.CreateRequest[AccountSafeListNumberArgs]) (infer.CreateResponse[AccountSafeListNumberState], error) {
	type resp = infer.CreateResponse[AccountSafeListNumberState]
	a := req.Inputs
	if req.DryRun {
		return resp{Output: AccountSafeListNumberState{AccountSafeListNumberArgs: a}}, nil
	}
	c, err := clientFor(ctx)
	if err != nil {
		return resp{}, err
	}
	o, err := c.do(ctx, "POST", accountsProduct, accountsSafeListPath, nil, url.Values{"PhoneNumber": {a.PhoneNumber}})
	if err != nil {
		return resp{}, fmt.Errorf("add %s to the Global SafeList: %w", a.PhoneNumber, err)
	}
	return resp{ID: a.PhoneNumber, Output: AccountSafeListNumberState{AccountSafeListNumberArgs: a, Sid: o.str("sid")}}, nil
}

func (r *AccountSafeListNumber) Read(ctx context.Context, req infer.ReadRequest[AccountSafeListNumberArgs, AccountSafeListNumberState]) (infer.ReadResponse[AccountSafeListNumberArgs, AccountSafeListNumberState], error) {
	type resp = infer.ReadResponse[AccountSafeListNumberArgs, AccountSafeListNumberState]
	c, err := clientFor(ctx)
	if err != nil {
		return resp{}, err
	}
	o, err := c.do(ctx, "GET", accountsProduct, accountsSafeListPath, url.Values{"PhoneNumber": {req.ID}}, nil)
	if err == nil && o.str("phone_number") == "" {
		// A reply that names no number is not an entry.
		err = &apiError{Status: 404, Message: "the number is not on the Global SafeList"}
	}
	if err != nil {
		if isNotFound(err) {
			if err := notFoundOnImport("Global SafeList number", req.ID, req.Inputs, req.State); err != nil {
				return resp{}, err
			}
			return resp{}, nil
		}
		return resp{}, fmt.Errorf("read Global SafeList number %s: %w", req.ID, err)
	}
	in := req.Inputs
	if in.PhoneNumber == "" {
		in.PhoneNumber = req.ID
	}
	return resp{ID: req.ID, Inputs: in, State: AccountSafeListNumberState{AccountSafeListNumberArgs: in, Sid: o.str("sid")}}, nil
}

func (r *AccountSafeListNumber) Delete(ctx context.Context, req infer.DeleteRequest[AccountSafeListNumberState]) (infer.DeleteResponse, error) {
	c, err := clientFor(ctx)
	if err != nil {
		return infer.DeleteResponse{}, err
	}
	if _, err := c.do(ctx, "DELETE", accountsProduct, accountsSafeListPath, url.Values{"PhoneNumber": {req.ID}}, nil); err != nil && !isNotFound(err) {
		return infer.DeleteResponse{}, fmt.Errorf("remove %s from the Global SafeList: %w", req.ID, err)
	}
	return infer.DeleteResponse{}, nil
}
