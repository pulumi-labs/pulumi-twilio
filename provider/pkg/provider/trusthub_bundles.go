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
	"strings"

	"github.com/pulumi/pulumi-go-provider/infer"
)

// TrustHub regulatory bundles (trusthub.twilio.com v1, twilio-oai spec/json/twilio_trusthub_v1.json
// 2.11.0; not checked against a live account): Customer Profiles and Trust Products are the same shape
// (a draft that end users and supporting documents are assigned to, then submitted for review), and
// each has entity assignments. Evaluations (a dry-run compliance check), provisional copies and channel
// endpoint assignments are not modelled.

var (
	objectSidRE = regexp.MustCompile(`^(IT|RD)[0-9a-fA-F]{32}$`)
	policySidRE = regexp.MustCompile(`^RN[0-9a-fA-F]{32}$`)
)

// bundleCheck validates the inputs both bundle resources share.
func bundleCheck(c *checker, friendlyName, email, policySid string, callback *string) {
	c.required("friendlyName", friendlyName)
	c.required("email", email)
	if !c.unk("email") && email != "" && !strings.Contains(email, "@") {
		c.fail("email", "email must be an email address")
	}
	c.sid("policySid", policySid, policySidRE, "policy (RN...)")
	if callback != nil && !c.unk("statusCallback") {
		if u, err := url.Parse(*callback); err != nil || (u.Scheme != "http" && u.Scheme != "https") || u.Host == "" {
			c.fail("statusCallback", "statusCallback must be an http or https URL")
		}
	}
}

// bundleDiff records the changes of a bundle. Submitting is one way: only going from not submitting to
// submitting is a change.
func bundleDiff(d trDiffSet, oName, oEmail, oPolicy string, oCb *string, oSubmit *bool, nName, nEmail, nPolicy string, nCb *string, nSubmit *bool) {
	d.replace("policySid", oPolicy, nPolicy)
	d.requiredStr("friendlyName", oName, nName)
	d.requiredStr("email", oEmail, nEmail)
	d.str("statusCallback", oCb, nCb)
	if derefBool(nSubmit) && !derefBool(oSubmit) {
		d.requiredStr("submit", "", "true")
	}
}

func bundleCreate(ctx context.Context, kind, coll, friendlyName, email, policySid string, callback *string, submit bool) (obj, error) {
	c, err := clientFor(ctx)
	if err != nil {
		return nil, err
	}
	form := url.Values{"FriendlyName": {friendlyName}, "Email": {email}, "PolicySid": {policySid}}
	setS(form, "StatusCallback", callback)
	o, err := c.do(ctx, "POST", trusthubProduct, thPath(coll, ""), nil, form)
	if err != nil {
		return nil, fmt.Errorf("create TrustHub %s: %w", kind, err)
	}
	sid := o.str("sid")
	if sid == "" {
		return nil, fmt.Errorf("create TrustHub %s: Twilio returned no SID", kind)
	}
	if submit {
		sub, err := c.do(ctx, "POST", trusthubProduct, thPath(coll, sid), nil, url.Values{"Status": {"pending-review"}})
		if err != nil {
			// Do not leave a draft behind that no state tracks.
			_, _ = c.do(context.WithoutCancel(ctx), "DELETE", trusthubProduct, thPath(coll, sid), nil, nil)
			return nil, fmt.Errorf("submit TrustHub %s %s for review (the draft was deleted): %w", kind, sid, err)
		}
		o = sub
	}
	return o, nil
}

func bundleUpdate(ctx context.Context, kind, coll, id, friendlyName, email string, callback *string, submit bool) (obj, error) {
	c, err := clientFor(ctx)
	if err != nil {
		return nil, err
	}
	form := url.Values{"FriendlyName": {friendlyName}, "Email": {email}}
	setS(form, "StatusCallback", callback)
	if submit {
		form.Set("Status", "pending-review")
	}
	o, err := c.do(ctx, "POST", trusthubProduct, thPath(coll, id), nil, form)
	if err != nil {
		return nil, fmt.Errorf("update TrustHub %s %s: %w", kind, id, err)
	}
	return o, nil
}

func assignmentPath(coll, bundle, sid string) string {
	p := thPath(coll, bundle) + "/EntityAssignments"
	if sid != "" {
		p += "/" + url.PathEscape(sid)
	}
	return p
}

func splitAssignmentID(id string) (bundle, sid string, err error) {
	bundle, sid, ok := strings.Cut(id, "/")
	if !ok || bundle == "" || sid == "" || strings.Contains(sid, "/") {
		return "", "", fmt.Errorf("invalid entity assignment ID %q: expected <bundleSid>/<assignmentSid>", id)
	}
	return bundle, sid, nil
}

func assignmentCreate(ctx context.Context, coll, bundle, objectSid string) (obj, error) {
	c, err := clientFor(ctx)
	if err != nil {
		return nil, err
	}
	o, err := c.do(ctx, "POST", trusthubProduct, assignmentPath(coll, bundle, ""), nil, url.Values{"ObjectSid": {objectSid}})
	if err != nil {
		return nil, fmt.Errorf("assign %s to TrustHub bundle %s: %w", objectSid, bundle, err)
	}
	if o.str("sid") == "" {
		return nil, fmt.Errorf("assign %s to TrustHub bundle %s: Twilio returned no SID", objectSid, bundle)
	}
	return o, nil
}

func assignmentFetch(ctx context.Context, coll, bundle, sid string) (obj, bool, error) {
	c, err := clientFor(ctx)
	if err != nil {
		return nil, false, err
	}
	o, err := c.do(ctx, "GET", trusthubProduct, assignmentPath(coll, bundle, sid), nil, nil)
	if err != nil {
		if isNotFound(err) {
			return nil, false, nil
		}
		return nil, false, fmt.Errorf("read TrustHub entity assignment %s/%s: %w", bundle, sid, err)
	}
	return o, true, nil
}

func assignmentDelete(ctx context.Context, coll, bundle, sid string) error {
	c, err := clientFor(ctx)
	if err != nil {
		return err
	}
	if _, err := c.do(ctx, "DELETE", trusthubProduct, assignmentPath(coll, bundle, sid), nil, nil); err != nil && !isNotFound(err) {
		return fmt.Errorf("delete TrustHub entity assignment %s/%s: %w", bundle, sid, err)
	}
	return nil
}

// ------------------------------------------------------------ CustomerProfile

// CustomerProfile is a customer profile regulatory bundle: the object that groups end users and supporting documents and is
// submitted to Twilio for review.
type CustomerProfile struct{}

// CustomerProfileArgs are the inputs of a CustomerProfile.
type CustomerProfileArgs struct {
	FriendlyName   string  `pulumi:"friendlyName"`
	Email          string  `pulumi:"email"`
	PolicySid      string  `pulumi:"policySid" provider:"replaceOnChanges"`
	StatusCallback *string `pulumi:"statusCallback,optional"`
	Submit         *bool   `pulumi:"submit,optional"`
}

func (a *CustomerProfileArgs) Annotate(an infer.Annotator) {
	an.Describe(&a.FriendlyName, "Human-readable name of the customer profile.")
	an.Describe(&a.Email, "Email address that receives updates when the customer profile changes status.")
	an.Describe(&a.PolicySid, "SID (RN...) of the regulatory policy the customer profile is evaluated against (see getTrustHubPolicies). Changing it replaces the customer profile.")
	an.Describe(&a.StatusCallback, "URL Twilio calls when the status changes. Removing the input leaves the URL unchanged.")
	an.Describe(&a.Submit, "Set to true to submit the customer profile for Twilio's review (status pending-review) once its end users and documents are assigned. Submitting cannot be undone from the provider: setting it back to false or removing it changes nothing, and neither does it when the status is no longer draft. Twilio rejects the submission when the bundle is not compliant with the policy; the create then fails and the draft is deleted. Not read back, so it is not drift.")
}

// CustomerProfileState is the persisted state of a CustomerProfile.
type CustomerProfileState struct {
	CustomerProfileArgs

	Sid         string `pulumi:"sid"`
	AccountSid  string `pulumi:"accountSid"`
	Status      string `pulumi:"status"`
	ValidUntil  string `pulumi:"validUntil"`
	Errors      string `pulumi:"errors"`
	DateCreated string `pulumi:"dateCreated"`
	DateUpdated string `pulumi:"dateUpdated"`
}

func (s *CustomerProfileState) Annotate(a infer.Annotator) {
	a.Describe(&s.Sid, "The customer profile SID (BU...).")
	a.Describe(&s.AccountSid, "SID (AC...) of the account that owns the customer profile.")
	a.Describe(&s.Status, "Review status as Twilio last reported it: draft, pending-review, in-review, twilio-rejected or twilio-approved. It changes through Twilio's review, not through this resource, except that submit moves a draft to pending-review.")
	a.Describe(&s.ValidUntil, "When an approved customer profile expires (ISO 8601), empty until then.")
	a.Describe(&s.Errors, "The reasons of a rejection as a JSON document, empty when there are none.")
	a.Describe(&s.DateCreated, "When the customer profile was created (ISO 8601).")
	a.Describe(&s.DateUpdated, "When the customer profile was last updated (ISO 8601).")
}

func (s *CustomerProfileState) fill(o obj) {
	s.Sid, s.AccountSid, s.Status = o.str("sid"), o.str("account_sid"), o.str("status")
	s.ValidUntil, s.Errors = o.str("valid_until"), a2pJSON(o.get("errors"))
	s.DateCreated, s.DateUpdated = o.str("date_created"), o.str("date_updated")
}

func (r *CustomerProfile) Annotate(a infer.Annotator) {
	a.Describe(&r, "A Twilio TrustHub customer profile (trusthub.twilio.com /v1/CustomerProfiles): the regulatory bundle that end users and supporting documents are assigned to and that Twilio reviews. "+
		"A new one is a draft and has no cost. Set submit to true to send it for review; assignments are separate resources. Twilio may refuse to delete a customer profile that is in review or approved. The resource ID is the SID (BU...). Not verified against a live account.")
}

var _ infer.CustomResource[CustomerProfileArgs, CustomerProfileState] = (*CustomerProfile)(nil)
var _ infer.CustomCheck[CustomerProfileArgs] = (*CustomerProfile)(nil)
var _ infer.CustomDiff[CustomerProfileArgs, CustomerProfileState] = (*CustomerProfile)(nil)
var _ infer.CustomRead[CustomerProfileArgs, CustomerProfileState] = (*CustomerProfile)(nil)
var _ infer.CustomUpdate[CustomerProfileArgs, CustomerProfileState] = (*CustomerProfile)(nil)
var _ infer.CustomDelete[CustomerProfileState] = (*CustomerProfile)(nil)

func (r *CustomerProfile) Check(ctx context.Context, req infer.CheckRequest) (infer.CheckResponse[CustomerProfileArgs], error) {
	args, failures, err := infer.DefaultCheck[CustomerProfileArgs](ctx, req.NewInputs)
	if err != nil || len(failures) > 0 {
		return infer.CheckResponse[CustomerProfileArgs]{Inputs: args, Failures: failures}, err
	}
	c := &checker{unk: unknowns(req.NewInputs)}
	bundleCheck(c, args.FriendlyName, args.Email, args.PolicySid, args.StatusCallback)
	return infer.CheckResponse[CustomerProfileArgs]{Inputs: args, Failures: c.failures}, nil
}

func (r *CustomerProfile) Diff(_ context.Context, req infer.DiffRequest[CustomerProfileArgs, CustomerProfileState]) (infer.DiffResponse, error) {
	o, n := req.State.CustomerProfileArgs, req.Inputs
	d := trDiffSet{}
	bundleDiff(d, o.FriendlyName, o.Email, o.PolicySid, o.StatusCallback, o.Submit, n.FriendlyName, n.Email, n.PolicySid, n.StatusCallback, n.Submit)
	return d.response(), nil
}

func (r *CustomerProfile) Create(ctx context.Context, req infer.CreateRequest[CustomerProfileArgs]) (infer.CreateResponse[CustomerProfileState], error) {
	type resp = infer.CreateResponse[CustomerProfileState]
	a := req.Inputs
	st := CustomerProfileState{CustomerProfileArgs: a}
	if req.DryRun {
		return resp{Output: st}, nil
	}
	o, err := bundleCreate(ctx, "customer profile", "CustomerProfiles", a.FriendlyName, a.Email, a.PolicySid, a.StatusCallback, derefBool(a.Submit))
	if err != nil {
		return resp{}, err
	}
	st.fill(o)
	return resp{ID: st.Sid, Output: st}, nil
}

func (r *CustomerProfile) Read(ctx context.Context, req infer.ReadRequest[CustomerProfileArgs, CustomerProfileState]) (infer.ReadResponse[CustomerProfileArgs, CustomerProfileState], error) {
	type resp = infer.ReadResponse[CustomerProfileArgs, CustomerProfileState]
	o, found, err := thFetch(ctx, trustHubAPI, "customer profile", "CustomerProfiles", req.ID)
	if err != nil {
		return resp{}, err
	}
	if !found {
		if err := notFoundOnImport("TrustHub customer profile", req.ID, req.Inputs, req.State); err != nil {
			return resp{}, err
		}
		return resp{}, nil
	}
	in := req.Inputs
	in.FriendlyName, in.Email, in.PolicySid = o.str("friendly_name"), o.str("email"), o.str("policy_sid")
	if v := nonEmpty(o.optStr("status_callback")); v != nil || in.StatusCallback != nil {
		in.StatusCallback = v
	}
	st := CustomerProfileState{CustomerProfileArgs: in}
	st.fill(o)
	return resp{ID: req.ID, Inputs: in, State: st}, nil
}

func (r *CustomerProfile) Update(ctx context.Context, req infer.UpdateRequest[CustomerProfileArgs, CustomerProfileState]) (infer.UpdateResponse[CustomerProfileState], error) {
	type resp = infer.UpdateResponse[CustomerProfileState]
	st := req.State
	st.CustomerProfileArgs = req.Inputs
	if req.DryRun {
		return resp{Output: st}, nil
	}
	in := req.Inputs
	submit := derefBool(in.Submit) && st.Status == "draft"
	o, err := bundleUpdate(ctx, "customer profile", "CustomerProfiles", req.ID, in.FriendlyName, in.Email, in.StatusCallback, submit)
	if err != nil {
		return resp{}, err
	}
	st.Status, st.ValidUntil, st.Errors, st.DateUpdated = o.str("status"), o.str("valid_until"), a2pJSON(o.get("errors")), o.str("date_updated")
	return resp{Output: st}, nil
}

func (r *CustomerProfile) Delete(ctx context.Context, req infer.DeleteRequest[CustomerProfileState]) (infer.DeleteResponse, error) {
	return infer.DeleteResponse{}, thDelete(ctx, trustHubAPI, "customer profile", "CustomerProfiles", req.ID)
}

// ------------------------------------------------------------ CustomerProfileEntityAssignment

// CustomerProfileEntityAssignment assigns an end user or a supporting document to a customer profile.
type CustomerProfileEntityAssignment struct{}

// CustomerProfileEntityAssignmentArgs are the inputs of a CustomerProfileEntityAssignment.
type CustomerProfileEntityAssignmentArgs struct {
	CustomerProfileSid string `pulumi:"customerProfileSid" provider:"replaceOnChanges"`
	ObjectSid          string `pulumi:"objectSid" provider:"replaceOnChanges"`
}

func (a *CustomerProfileEntityAssignmentArgs) Annotate(an infer.Annotator) {
	an.Describe(&a.CustomerProfileSid, "SID (BU...) of the customer profile. Changing it replaces the assignment.")
	an.Describe(&a.ObjectSid, "SID of the end user (IT...) or supporting document (RD...) to assign. Changing it replaces the assignment.")
}

// CustomerProfileEntityAssignmentState is the persisted state of a CustomerProfileEntityAssignment.
type CustomerProfileEntityAssignmentState struct {
	CustomerProfileEntityAssignmentArgs

	Sid         string `pulumi:"sid"`
	AccountSid  string `pulumi:"accountSid"`
	DateCreated string `pulumi:"dateCreated"`
}

func (s *CustomerProfileEntityAssignmentState) Annotate(a infer.Annotator) {
	a.Describe(&s.Sid, "The assignment SID (BV...). The resource ID is the customer profile SID and this SID joined by a slash.")
	a.Describe(&s.AccountSid, "SID (AC...) of the account that owns the assignment.")
	a.Describe(&s.DateCreated, "When the assignment was created (ISO 8601).")
}

func (r *CustomerProfileEntityAssignment) Annotate(a infer.Annotator) {
	a.Describe(&r, "Assigns a TrustHub end user or supporting document to a customer profile (trusthub.twilio.com /v1/CustomerProfiles/{sid}/EntityAssignments). Every input replaces, because Twilio has no update call. "+
		"Twilio only accepts the assignment while the customer profile is a draft and refuses to remove one once it is in review. Import with the ID <customerProfileSid>/<assignmentSid>. Not verified against a live account.")
}

var _ infer.CustomResource[CustomerProfileEntityAssignmentArgs, CustomerProfileEntityAssignmentState] = (*CustomerProfileEntityAssignment)(nil)
var _ infer.CustomCheck[CustomerProfileEntityAssignmentArgs] = (*CustomerProfileEntityAssignment)(nil)
var _ infer.CustomDiff[CustomerProfileEntityAssignmentArgs, CustomerProfileEntityAssignmentState] = (*CustomerProfileEntityAssignment)(nil)
var _ infer.CustomRead[CustomerProfileEntityAssignmentArgs, CustomerProfileEntityAssignmentState] = (*CustomerProfileEntityAssignment)(nil)
var _ infer.CustomDelete[CustomerProfileEntityAssignmentState] = (*CustomerProfileEntityAssignment)(nil)

func (r *CustomerProfileEntityAssignment) Check(ctx context.Context, req infer.CheckRequest) (infer.CheckResponse[CustomerProfileEntityAssignmentArgs], error) {
	args, failures, err := infer.DefaultCheck[CustomerProfileEntityAssignmentArgs](ctx, req.NewInputs)
	if err != nil || len(failures) > 0 {
		return infer.CheckResponse[CustomerProfileEntityAssignmentArgs]{Inputs: args, Failures: failures}, err
	}
	c := &checker{unk: unknowns(req.NewInputs)}
	c.sid("customerProfileSid", args.CustomerProfileSid, bundleSidRE, "customer profile (BU...)")
	c.sid("objectSid", args.ObjectSid, objectSidRE, "end user (IT...) or supporting document (RD...)")
	return infer.CheckResponse[CustomerProfileEntityAssignmentArgs]{Inputs: args, Failures: c.failures}, nil
}

func (r *CustomerProfileEntityAssignment) Diff(_ context.Context, req infer.DiffRequest[CustomerProfileEntityAssignmentArgs, CustomerProfileEntityAssignmentState]) (infer.DiffResponse, error) {
	o, n := req.State.CustomerProfileEntityAssignmentArgs, req.Inputs
	d := trDiffSet{}
	d.replace("customerProfileSid", o.CustomerProfileSid, n.CustomerProfileSid)
	d.replace("objectSid", o.ObjectSid, n.ObjectSid)
	return d.response(), nil
}

func (r *CustomerProfileEntityAssignment) Create(ctx context.Context, req infer.CreateRequest[CustomerProfileEntityAssignmentArgs]) (infer.CreateResponse[CustomerProfileEntityAssignmentState], error) {
	type resp = infer.CreateResponse[CustomerProfileEntityAssignmentState]
	a := req.Inputs
	st := CustomerProfileEntityAssignmentState{CustomerProfileEntityAssignmentArgs: a}
	if req.DryRun {
		return resp{Output: st}, nil
	}
	o, err := assignmentCreate(ctx, "CustomerProfiles", a.CustomerProfileSid, a.ObjectSid)
	if err != nil {
		return resp{}, err
	}
	st.Sid, st.AccountSid, st.DateCreated = o.str("sid"), o.str("account_sid"), o.str("date_created")
	return resp{ID: a.CustomerProfileSid + "/" + st.Sid, Output: st}, nil
}

func (r *CustomerProfileEntityAssignment) Read(ctx context.Context, req infer.ReadRequest[CustomerProfileEntityAssignmentArgs, CustomerProfileEntityAssignmentState]) (infer.ReadResponse[CustomerProfileEntityAssignmentArgs, CustomerProfileEntityAssignmentState], error) {
	type resp = infer.ReadResponse[CustomerProfileEntityAssignmentArgs, CustomerProfileEntityAssignmentState]
	bundle, sid, err := splitAssignmentID(req.ID)
	if err != nil {
		return resp{}, err
	}
	o, found, err := assignmentFetch(ctx, "CustomerProfiles", bundle, sid)
	if err != nil {
		return resp{}, err
	}
	if !found {
		if err := notFoundOnImport("customer profile entity assignment", req.ID, req.Inputs, req.State); err != nil {
			return resp{}, err
		}
		return resp{}, nil
	}
	in := req.Inputs
	in.CustomerProfileSid, in.ObjectSid = bundle, o.str("object_sid")
	st := CustomerProfileEntityAssignmentState{CustomerProfileEntityAssignmentArgs: in, Sid: sid, AccountSid: o.str("account_sid"), DateCreated: o.str("date_created")}
	return resp{ID: req.ID, Inputs: in, State: st}, nil
}

func (r *CustomerProfileEntityAssignment) Delete(ctx context.Context, req infer.DeleteRequest[CustomerProfileEntityAssignmentState]) (infer.DeleteResponse, error) {
	bundle, sid, err := splitAssignmentID(req.ID)
	if err != nil {
		return infer.DeleteResponse{}, err
	}
	return infer.DeleteResponse{}, assignmentDelete(ctx, "CustomerProfiles", bundle, sid)
}

// ------------------------------------------------------------ TrustProduct

// TrustProduct is a trust product regulatory bundle: the object that groups end users and supporting documents and is
// submitted to Twilio for review.
type TrustProduct struct{}

// TrustProductArgs are the inputs of a TrustProduct.
type TrustProductArgs struct {
	FriendlyName   string  `pulumi:"friendlyName"`
	Email          string  `pulumi:"email"`
	PolicySid      string  `pulumi:"policySid" provider:"replaceOnChanges"`
	StatusCallback *string `pulumi:"statusCallback,optional"`
	Submit         *bool   `pulumi:"submit,optional"`
}

func (a *TrustProductArgs) Annotate(an infer.Annotator) {
	an.Describe(&a.FriendlyName, "Human-readable name of the trust product.")
	an.Describe(&a.Email, "Email address that receives updates when the trust product changes status.")
	an.Describe(&a.PolicySid, "SID (RN...) of the regulatory policy the trust product is evaluated against (see getTrustHubPolicies). Changing it replaces the trust product.")
	an.Describe(&a.StatusCallback, "URL Twilio calls when the status changes. Removing the input leaves the URL unchanged.")
	an.Describe(&a.Submit, "Set to true to submit the trust product for Twilio's review (status pending-review) once its end users and documents are assigned. Submitting cannot be undone from the provider: setting it back to false or removing it changes nothing, and neither does it when the status is no longer draft. Twilio rejects the submission when the bundle is not compliant with the policy; the create then fails and the draft is deleted. Not read back, so it is not drift.")
}

// TrustProductState is the persisted state of a TrustProduct.
type TrustProductState struct {
	TrustProductArgs

	Sid         string `pulumi:"sid"`
	AccountSid  string `pulumi:"accountSid"`
	Status      string `pulumi:"status"`
	ValidUntil  string `pulumi:"validUntil"`
	Errors      string `pulumi:"errors"`
	DateCreated string `pulumi:"dateCreated"`
	DateUpdated string `pulumi:"dateUpdated"`
}

func (s *TrustProductState) Annotate(a infer.Annotator) {
	a.Describe(&s.Sid, "The trust product SID (BU...).")
	a.Describe(&s.AccountSid, "SID (AC...) of the account that owns the trust product.")
	a.Describe(&s.Status, "Review status as Twilio last reported it: draft, pending-review, in-review, twilio-rejected or twilio-approved. It changes through Twilio's review, not through this resource, except that submit moves a draft to pending-review.")
	a.Describe(&s.ValidUntil, "When an approved trust product expires (ISO 8601), empty until then.")
	a.Describe(&s.Errors, "The reasons of a rejection as a JSON document, empty when there are none.")
	a.Describe(&s.DateCreated, "When the trust product was created (ISO 8601).")
	a.Describe(&s.DateUpdated, "When the trust product was last updated (ISO 8601).")
}

func (s *TrustProductState) fill(o obj) {
	s.Sid, s.AccountSid, s.Status = o.str("sid"), o.str("account_sid"), o.str("status")
	s.ValidUntil, s.Errors = o.str("valid_until"), a2pJSON(o.get("errors"))
	s.DateCreated, s.DateUpdated = o.str("date_created"), o.str("date_updated")
}

func (r *TrustProduct) Annotate(a infer.Annotator) {
	a.Describe(&r, "A Twilio TrustHub trust product (trusthub.twilio.com /v1/TrustProducts): the regulatory bundle that end users and supporting documents are assigned to and that Twilio reviews. "+
		"A new one is a draft and has no cost. Set submit to true to send it for review; assignments are separate resources. Twilio may refuse to delete a trust product that is in review or approved. The resource ID is the SID (BU...). Not verified against a live account.")
}

var _ infer.CustomResource[TrustProductArgs, TrustProductState] = (*TrustProduct)(nil)
var _ infer.CustomCheck[TrustProductArgs] = (*TrustProduct)(nil)
var _ infer.CustomDiff[TrustProductArgs, TrustProductState] = (*TrustProduct)(nil)
var _ infer.CustomRead[TrustProductArgs, TrustProductState] = (*TrustProduct)(nil)
var _ infer.CustomUpdate[TrustProductArgs, TrustProductState] = (*TrustProduct)(nil)
var _ infer.CustomDelete[TrustProductState] = (*TrustProduct)(nil)

func (r *TrustProduct) Check(ctx context.Context, req infer.CheckRequest) (infer.CheckResponse[TrustProductArgs], error) {
	args, failures, err := infer.DefaultCheck[TrustProductArgs](ctx, req.NewInputs)
	if err != nil || len(failures) > 0 {
		return infer.CheckResponse[TrustProductArgs]{Inputs: args, Failures: failures}, err
	}
	c := &checker{unk: unknowns(req.NewInputs)}
	bundleCheck(c, args.FriendlyName, args.Email, args.PolicySid, args.StatusCallback)
	return infer.CheckResponse[TrustProductArgs]{Inputs: args, Failures: c.failures}, nil
}

func (r *TrustProduct) Diff(_ context.Context, req infer.DiffRequest[TrustProductArgs, TrustProductState]) (infer.DiffResponse, error) {
	o, n := req.State.TrustProductArgs, req.Inputs
	d := trDiffSet{}
	bundleDiff(d, o.FriendlyName, o.Email, o.PolicySid, o.StatusCallback, o.Submit, n.FriendlyName, n.Email, n.PolicySid, n.StatusCallback, n.Submit)
	return d.response(), nil
}

func (r *TrustProduct) Create(ctx context.Context, req infer.CreateRequest[TrustProductArgs]) (infer.CreateResponse[TrustProductState], error) {
	type resp = infer.CreateResponse[TrustProductState]
	a := req.Inputs
	st := TrustProductState{TrustProductArgs: a}
	if req.DryRun {
		return resp{Output: st}, nil
	}
	o, err := bundleCreate(ctx, "trust product", "TrustProducts", a.FriendlyName, a.Email, a.PolicySid, a.StatusCallback, derefBool(a.Submit))
	if err != nil {
		return resp{}, err
	}
	st.fill(o)
	return resp{ID: st.Sid, Output: st}, nil
}

func (r *TrustProduct) Read(ctx context.Context, req infer.ReadRequest[TrustProductArgs, TrustProductState]) (infer.ReadResponse[TrustProductArgs, TrustProductState], error) {
	type resp = infer.ReadResponse[TrustProductArgs, TrustProductState]
	o, found, err := thFetch(ctx, trustHubAPI, "trust product", "TrustProducts", req.ID)
	if err != nil {
		return resp{}, err
	}
	if !found {
		if err := notFoundOnImport("TrustHub trust product", req.ID, req.Inputs, req.State); err != nil {
			return resp{}, err
		}
		return resp{}, nil
	}
	in := req.Inputs
	in.FriendlyName, in.Email, in.PolicySid = o.str("friendly_name"), o.str("email"), o.str("policy_sid")
	if v := nonEmpty(o.optStr("status_callback")); v != nil || in.StatusCallback != nil {
		in.StatusCallback = v
	}
	st := TrustProductState{TrustProductArgs: in}
	st.fill(o)
	return resp{ID: req.ID, Inputs: in, State: st}, nil
}

func (r *TrustProduct) Update(ctx context.Context, req infer.UpdateRequest[TrustProductArgs, TrustProductState]) (infer.UpdateResponse[TrustProductState], error) {
	type resp = infer.UpdateResponse[TrustProductState]
	st := req.State
	st.TrustProductArgs = req.Inputs
	if req.DryRun {
		return resp{Output: st}, nil
	}
	in := req.Inputs
	submit := derefBool(in.Submit) && st.Status == "draft"
	o, err := bundleUpdate(ctx, "trust product", "TrustProducts", req.ID, in.FriendlyName, in.Email, in.StatusCallback, submit)
	if err != nil {
		return resp{}, err
	}
	st.Status, st.ValidUntil, st.Errors, st.DateUpdated = o.str("status"), o.str("valid_until"), a2pJSON(o.get("errors")), o.str("date_updated")
	return resp{Output: st}, nil
}

func (r *TrustProduct) Delete(ctx context.Context, req infer.DeleteRequest[TrustProductState]) (infer.DeleteResponse, error) {
	return infer.DeleteResponse{}, thDelete(ctx, trustHubAPI, "trust product", "TrustProducts", req.ID)
}

// ------------------------------------------------------------ TrustProductEntityAssignment

// TrustProductEntityAssignment assigns an end user or a supporting document to a trust product.
type TrustProductEntityAssignment struct{}

// TrustProductEntityAssignmentArgs are the inputs of a TrustProductEntityAssignment.
type TrustProductEntityAssignmentArgs struct {
	TrustProductSid string `pulumi:"trustProductSid" provider:"replaceOnChanges"`
	ObjectSid       string `pulumi:"objectSid" provider:"replaceOnChanges"`
}

func (a *TrustProductEntityAssignmentArgs) Annotate(an infer.Annotator) {
	an.Describe(&a.TrustProductSid, "SID (BU...) of the trust product. Changing it replaces the assignment.")
	an.Describe(&a.ObjectSid, "SID of the end user (IT...) or supporting document (RD...) to assign. Changing it replaces the assignment.")
}

// TrustProductEntityAssignmentState is the persisted state of a TrustProductEntityAssignment.
type TrustProductEntityAssignmentState struct {
	TrustProductEntityAssignmentArgs

	Sid         string `pulumi:"sid"`
	AccountSid  string `pulumi:"accountSid"`
	DateCreated string `pulumi:"dateCreated"`
}

func (s *TrustProductEntityAssignmentState) Annotate(a infer.Annotator) {
	a.Describe(&s.Sid, "The assignment SID (BV...). The resource ID is the trust product SID and this SID joined by a slash.")
	a.Describe(&s.AccountSid, "SID (AC...) of the account that owns the assignment.")
	a.Describe(&s.DateCreated, "When the assignment was created (ISO 8601).")
}

func (r *TrustProductEntityAssignment) Annotate(a infer.Annotator) {
	a.Describe(&r, "Assigns a TrustHub end user or supporting document to a trust product (trusthub.twilio.com /v1/TrustProducts/{sid}/EntityAssignments). Every input replaces, because Twilio has no update call. "+
		"Twilio only accepts the assignment while the trust product is a draft and refuses to remove one once it is in review. Import with the ID <trustProductSid>/<assignmentSid>. Not verified against a live account.")
}

var _ infer.CustomResource[TrustProductEntityAssignmentArgs, TrustProductEntityAssignmentState] = (*TrustProductEntityAssignment)(nil)
var _ infer.CustomCheck[TrustProductEntityAssignmentArgs] = (*TrustProductEntityAssignment)(nil)
var _ infer.CustomDiff[TrustProductEntityAssignmentArgs, TrustProductEntityAssignmentState] = (*TrustProductEntityAssignment)(nil)
var _ infer.CustomRead[TrustProductEntityAssignmentArgs, TrustProductEntityAssignmentState] = (*TrustProductEntityAssignment)(nil)
var _ infer.CustomDelete[TrustProductEntityAssignmentState] = (*TrustProductEntityAssignment)(nil)

func (r *TrustProductEntityAssignment) Check(ctx context.Context, req infer.CheckRequest) (infer.CheckResponse[TrustProductEntityAssignmentArgs], error) {
	args, failures, err := infer.DefaultCheck[TrustProductEntityAssignmentArgs](ctx, req.NewInputs)
	if err != nil || len(failures) > 0 {
		return infer.CheckResponse[TrustProductEntityAssignmentArgs]{Inputs: args, Failures: failures}, err
	}
	c := &checker{unk: unknowns(req.NewInputs)}
	c.sid("trustProductSid", args.TrustProductSid, bundleSidRE, "trust product (BU...)")
	c.sid("objectSid", args.ObjectSid, objectSidRE, "end user (IT...) or supporting document (RD...)")
	return infer.CheckResponse[TrustProductEntityAssignmentArgs]{Inputs: args, Failures: c.failures}, nil
}

func (r *TrustProductEntityAssignment) Diff(_ context.Context, req infer.DiffRequest[TrustProductEntityAssignmentArgs, TrustProductEntityAssignmentState]) (infer.DiffResponse, error) {
	o, n := req.State.TrustProductEntityAssignmentArgs, req.Inputs
	d := trDiffSet{}
	d.replace("trustProductSid", o.TrustProductSid, n.TrustProductSid)
	d.replace("objectSid", o.ObjectSid, n.ObjectSid)
	return d.response(), nil
}

func (r *TrustProductEntityAssignment) Create(ctx context.Context, req infer.CreateRequest[TrustProductEntityAssignmentArgs]) (infer.CreateResponse[TrustProductEntityAssignmentState], error) {
	type resp = infer.CreateResponse[TrustProductEntityAssignmentState]
	a := req.Inputs
	st := TrustProductEntityAssignmentState{TrustProductEntityAssignmentArgs: a}
	if req.DryRun {
		return resp{Output: st}, nil
	}
	o, err := assignmentCreate(ctx, "TrustProducts", a.TrustProductSid, a.ObjectSid)
	if err != nil {
		return resp{}, err
	}
	st.Sid, st.AccountSid, st.DateCreated = o.str("sid"), o.str("account_sid"), o.str("date_created")
	return resp{ID: a.TrustProductSid + "/" + st.Sid, Output: st}, nil
}

func (r *TrustProductEntityAssignment) Read(ctx context.Context, req infer.ReadRequest[TrustProductEntityAssignmentArgs, TrustProductEntityAssignmentState]) (infer.ReadResponse[TrustProductEntityAssignmentArgs, TrustProductEntityAssignmentState], error) {
	type resp = infer.ReadResponse[TrustProductEntityAssignmentArgs, TrustProductEntityAssignmentState]
	bundle, sid, err := splitAssignmentID(req.ID)
	if err != nil {
		return resp{}, err
	}
	o, found, err := assignmentFetch(ctx, "TrustProducts", bundle, sid)
	if err != nil {
		return resp{}, err
	}
	if !found {
		if err := notFoundOnImport("trust product entity assignment", req.ID, req.Inputs, req.State); err != nil {
			return resp{}, err
		}
		return resp{}, nil
	}
	in := req.Inputs
	in.TrustProductSid, in.ObjectSid = bundle, o.str("object_sid")
	st := TrustProductEntityAssignmentState{TrustProductEntityAssignmentArgs: in, Sid: sid, AccountSid: o.str("account_sid"), DateCreated: o.str("date_created")}
	return resp{ID: req.ID, Inputs: in, State: st}, nil
}

func (r *TrustProductEntityAssignment) Delete(ctx context.Context, req infer.DeleteRequest[TrustProductEntityAssignmentState]) (infer.DeleteResponse, error) {
	bundle, sid, err := splitAssignmentID(req.ID)
	if err != nil {
		return infer.DeleteResponse{}, err
	}
	return infer.DeleteResponse{}, assignmentDelete(ctx, "TrustProducts", bundle, sid)
}
