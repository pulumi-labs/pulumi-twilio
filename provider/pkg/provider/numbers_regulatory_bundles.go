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

	p "github.com/pulumi/pulumi-go-provider"
	"github.com/pulumi/pulumi-go-provider/infer"
)

// Numbers regulatory bundles (numbers.twilio.com /v2/RegulatoryCompliance/Bundles, twilio-oai
// spec/json/twilio_numbers_v2.json 2.11.0; not checked against a live account): the bundle that
// proves who owns the phone numbers of a country, built from end users and supporting documents
// (numbers_regulatory.go) assigned to it and submitted for Twilio's review. Same workflow as the
// TrustHub bundles, but a bundle is created for a regulation. Clones, copies, evaluations and
// item replacement are not modelled.

const numbersProduct = "numbers"

var isoCountryRE = regexp.MustCompile(`^[A-Za-z]{2}$`)

func nrBundlePath(sid string) string { return numbersRegAPI.path("Bundles", sid) }

func nrItemPath(bundle, sid string) string {
	p := nrBundlePath(bundle) + "/ItemAssignments"
	if sid != "" {
		p += "/" + url.PathEscape(sid)
	}
	return p
}

// RegulatoryBundle is a regulatory bundle for the phone numbers of one country.
type RegulatoryBundle struct{}

// RegulatoryBundleArgs are the inputs of a RegulatoryBundle.
type RegulatoryBundleArgs struct {
	FriendlyName   string  `pulumi:"friendlyName"`
	Email          string  `pulumi:"email"`
	StatusCallback *string `pulumi:"statusCallback,optional"`
	Submit         *bool   `pulumi:"submit,optional"`

	RegulationSid *string `pulumi:"regulationSid,optional"`
	IsoCountry    *string `pulumi:"isoCountry,optional"`
	NumberType    *string `pulumi:"numberType,optional"`
	EndUserType   *string `pulumi:"endUserType,optional"`
	IsTest        *bool   `pulumi:"isTest,optional"`
}

func (a *RegulatoryBundleArgs) Annotate(an infer.Annotator) {
	an.Describe(&a.FriendlyName, "Human-readable name of the bundle.")
	an.Describe(&a.Email, "Email address that receives updates when the bundle changes status.")
	an.Describe(&a.StatusCallback, "URL Twilio calls when the status changes. Removing the input leaves the URL unchanged.")
	an.Describe(&a.Submit, "Set to true to submit the bundle for Twilio's review (status pending-review) once its end users and documents are assigned. Submitting cannot be undone from the provider: setting it back to false or removing it changes nothing, and neither does it when the status is no longer draft. Twilio rejects the submission when the bundle is not compliant with its regulation; the create then fails and the draft is deleted. Not read back, so it is not drift.")
	an.Describe(&a.RegulationSid, "SID (RN...) of the regulation the bundle is for (see getRegulatoryRegulations). Give this, or isoCountry, numberType and endUserType, which Twilio resolves to a regulation (reported as effectiveRegulationSid). When the parts are given, Twilio does not return them, so drift in them is not detected; an import adopts the resolved regulation as regulationSid. Changing it replaces the bundle.")
	an.Describe(&a.IsoCountry, "ISO 3166-1 alpha-2 code of the country of the phone numbers the bundle covers. Not read back; changing it replaces the bundle.")
	an.Describe(&a.NumberType, "Type of the phone numbers: local, mobile, national or toll-free. Not read back; changing it replaces the bundle.")
	an.Describe(&a.EndUserType, "Who owns the numbers: individual or business. Not read back; changing it replaces the bundle.")
	an.Describe(&a.IsTest, "Marks a test bundle, which Twilio rejects automatically after submission. Not read back; changing it replaces the bundle.")
}

// RegulatoryBundleState is the persisted state of a RegulatoryBundle.
type RegulatoryBundleState struct {
	RegulatoryBundleArgs

	Sid                    string `pulumi:"sid"`
	AccountSid             string `pulumi:"accountSid"`
	EffectiveRegulationSid string `pulumi:"effectiveRegulationSid"`
	Status                 string `pulumi:"status"`
	ValidUntil             string `pulumi:"validUntil"`
	DateCreated            string `pulumi:"dateCreated"`
	DateUpdated            string `pulumi:"dateUpdated"`
}

func (s *RegulatoryBundleState) Annotate(a infer.Annotator) {
	a.Describe(&s.Sid, "The bundle SID (BU...).")
	a.Describe(&s.AccountSid, "SID (AC...) of the account that owns the bundle.")
	a.Describe(&s.EffectiveRegulationSid, "SID (RN...) of the regulation Twilio applies to the bundle, as Twilio reports it.")
	a.Describe(&s.Status, "Review status as Twilio last reported it: draft, pending-review, in-review, twilio-rejected, twilio-approved or provisionally-approved. It changes through Twilio's review, except that submit moves a draft to pending-review.")
	a.Describe(&s.ValidUntil, "When an approved bundle expires (ISO 8601), empty until then.")
	a.Describe(&s.DateCreated, "When the bundle was created (ISO 8601).")
	a.Describe(&s.DateUpdated, "When the bundle was last updated (ISO 8601).")
}

func (s *RegulatoryBundleState) fill(o obj) {
	s.Sid, s.AccountSid, s.EffectiveRegulationSid, s.Status = o.str("sid"), o.str("account_sid"), o.str("regulation_sid"), o.str("status")
	s.ValidUntil, s.DateCreated, s.DateUpdated = o.str("valid_until"), o.str("date_created"), o.str("date_updated")
}

func (r *RegulatoryBundle) Annotate(a infer.Annotator) {
	a.Describe(&r, "A Twilio Numbers regulatory bundle (numbers.twilio.com /v2/RegulatoryCompliance/Bundles): the draft that end users and supporting documents are assigned to, and that Twilio reviews, before it provides phone numbers of a regulated country. "+
		"A new one is a draft and has no cost. Set submit to true to send it for review; assignments are separate resources. Twilio may refuse to delete a bundle that is in review or approved. The resource ID is the SID (BU...). Not verified against a live account.")
}

var _ infer.CustomResource[RegulatoryBundleArgs, RegulatoryBundleState] = (*RegulatoryBundle)(nil)
var _ infer.CustomCheck[RegulatoryBundleArgs] = (*RegulatoryBundle)(nil)
var _ infer.CustomDiff[RegulatoryBundleArgs, RegulatoryBundleState] = (*RegulatoryBundle)(nil)
var _ infer.CustomRead[RegulatoryBundleArgs, RegulatoryBundleState] = (*RegulatoryBundle)(nil)
var _ infer.CustomUpdate[RegulatoryBundleArgs, RegulatoryBundleState] = (*RegulatoryBundle)(nil)
var _ infer.CustomDelete[RegulatoryBundleState] = (*RegulatoryBundle)(nil)

func (r *RegulatoryBundle) Check(ctx context.Context, req infer.CheckRequest) (infer.CheckResponse[RegulatoryBundleArgs], error) {
	args, failures, err := infer.DefaultCheck[RegulatoryBundleArgs](ctx, req.NewInputs)
	if err != nil || len(failures) > 0 {
		return infer.CheckResponse[RegulatoryBundleArgs]{Inputs: args, Failures: failures}, err
	}
	c := &checker{unk: unknowns(req.NewInputs)}
	c.required("friendlyName", args.FriendlyName)
	c.required("email", args.Email)
	if !c.unk("email") && args.Email != "" && !strings.Contains(args.Email, "@") {
		c.fail("email", "email must be an email address")
	}
	if args.StatusCallback != nil && !c.unk("statusCallback") {
		if u, err := url.Parse(*args.StatusCallback); err != nil || (u.Scheme != "http" && u.Scheme != "https") || u.Host == "" {
			c.fail("statusCallback", "statusCallback must be an http or https URL")
		}
	}
	if args.RegulationSid != nil {
		c.sid("regulationSid", *args.RegulationSid, regulationSidRE, "regulation (RN...)")
	}
	if args.IsoCountry != nil && !c.unk("isoCountry") && !isoCountryRE.MatchString(*args.IsoCountry) {
		c.fail("isoCountry", "isoCountry must be a two-letter ISO 3166-1 country code")
	}
	if args.EndUserType != nil {
		c.oneOf("endUserType", *args.EndUserType, []string{"individual", "business"})
	}
	if args.NumberType != nil && !c.unk("numberType") && *args.NumberType == "" {
		c.fail("numberType", "numberType must not be empty")
	}
	if !c.unk("regulationSid", "isoCountry", "numberType", "endUserType") {
		byParts := args.IsoCountry != nil && args.NumberType != nil && args.EndUserType != nil
		anyPart := args.IsoCountry != nil || args.NumberType != nil || args.EndUserType != nil
		switch {
		case args.RegulationSid != nil && anyPart:
			c.fail("regulationSid", "give either regulationSid or isoCountry, numberType and endUserType, not both")
		case args.RegulationSid == nil && !byParts:
			c.fail("regulationSid", "give regulationSid, or all of isoCountry, numberType and endUserType")
		}
	}
	return infer.CheckResponse[RegulatoryBundleArgs]{Inputs: args, Failures: c.failures}, nil
}

var regulationSidRE = regexp.MustCompile(`^RN[0-9a-fA-F]{32}$`)

// replaceKnown records a replacement of a create-only input Twilio does not return: only when both the
// recorded and the new value are set and differ, since an imported bundle has none recorded.
func replaceKnown[T comparable](d trDiffSet, name string, o, n *T) {
	if o != nil && n != nil && *o != *n {
		d[name] = p.PropertyDiff{Kind: p.UpdateReplace}
	}
}

func (r *RegulatoryBundle) Diff(_ context.Context, req infer.DiffRequest[RegulatoryBundleArgs, RegulatoryBundleState]) (infer.DiffResponse, error) {
	o, n := req.State.RegulatoryBundleArgs, req.Inputs
	d := trDiffSet{}
	d.requiredStr("friendlyName", o.FriendlyName, n.FriendlyName)
	d.requiredStr("email", o.Email, n.Email)
	d.str("statusCallback", o.StatusCallback, n.StatusCallback)
	if derefBool(n.Submit) && !derefBool(o.Submit) {
		d.requiredStr("submit", "", "true")
	}
	replaceKnown(d, "regulationSid", o.RegulationSid, n.RegulationSid)
	replaceKnown(d, "isoCountry", o.IsoCountry, n.IsoCountry)
	replaceKnown(d, "numberType", o.NumberType, n.NumberType)
	replaceKnown(d, "endUserType", o.EndUserType, n.EndUserType)
	replaceKnown(d, "isTest", o.IsTest, n.IsTest)
	return d.response(), nil
}

func (r *RegulatoryBundle) Create(ctx context.Context, req infer.CreateRequest[RegulatoryBundleArgs]) (infer.CreateResponse[RegulatoryBundleState], error) {
	type resp = infer.CreateResponse[RegulatoryBundleState]
	a := req.Inputs
	st := RegulatoryBundleState{RegulatoryBundleArgs: a}
	if req.DryRun {
		return resp{Output: st}, nil
	}
	c, err := clientFor(ctx)
	if err != nil {
		return resp{}, err
	}
	form := url.Values{"FriendlyName": {a.FriendlyName}, "Email": {a.Email}}
	setS(form, "StatusCallback", a.StatusCallback)
	setS(form, "RegulationSid", a.RegulationSid)
	setS(form, "IsoCountry", a.IsoCountry)
	setS(form, "NumberType", a.NumberType)
	setS(form, "EndUserType", a.EndUserType)
	setB(form, "IsTest", a.IsTest)
	o, err := c.do(ctx, "POST", numbersProduct, nrBundlePath(""), nil, form)
	if err != nil {
		return resp{}, fmt.Errorf("create regulatory bundle: %w", err)
	}
	sid := o.str("sid")
	if sid == "" {
		return resp{}, fmt.Errorf("create regulatory bundle: Twilio returned no SID")
	}
	if derefBool(a.Submit) {
		sub, err := c.do(ctx, "POST", numbersProduct, nrBundlePath(sid), nil, url.Values{"Status": {"pending-review"}})
		if err != nil {
			// Do not leave a draft behind that no state tracks.
			_, _ = c.do(context.WithoutCancel(ctx), "DELETE", numbersProduct, nrBundlePath(sid), nil, nil)
			return resp{}, fmt.Errorf("submit regulatory bundle %s for review (the draft was deleted): %w", sid, err)
		}
		o = sub
	}
	st.fill(o)
	return resp{ID: sid, Output: st}, nil
}

func (r *RegulatoryBundle) Read(ctx context.Context, req infer.ReadRequest[RegulatoryBundleArgs, RegulatoryBundleState]) (infer.ReadResponse[RegulatoryBundleArgs, RegulatoryBundleState], error) {
	type resp = infer.ReadResponse[RegulatoryBundleArgs, RegulatoryBundleState]
	o, found, err := thFetch(ctx, numbersRegAPI, "bundle", "Bundles", req.ID)
	if err != nil {
		return resp{}, err
	}
	if !found {
		if err := notFoundOnImport("regulatory bundle", req.ID, req.Inputs, req.State); err != nil {
			return resp{}, err
		}
		return resp{}, nil
	}
	in := req.Inputs
	in.FriendlyName, in.Email = o.str("friendly_name"), o.str("email")
	if v := nonEmpty(o.optStr("status_callback")); v != nil || in.StatusCallback != nil {
		in.StatusCallback = v
	}
	// An import has no regulation input at all, and a bundle without one fails Check. Twilio does
	// return the regulation it resolved, so adopt it as regulationSid; a program that states
	// isoCountry, numberType and endUserType instead stays clean (Diff only replaces when both
	// sides hold a regulationSid and they differ). A refresh keeps the inputs the program wrote.
	if in.RegulationSid == nil && in.IsoCountry == nil && in.NumberType == nil && in.EndUserType == nil {
		in.RegulationSid = nonEmpty(o.optStr("regulation_sid"))
	}
	st := RegulatoryBundleState{RegulatoryBundleArgs: in}
	st.fill(o)
	return resp{ID: req.ID, Inputs: in, State: st}, nil
}

func (r *RegulatoryBundle) Update(ctx context.Context, req infer.UpdateRequest[RegulatoryBundleArgs, RegulatoryBundleState]) (infer.UpdateResponse[RegulatoryBundleState], error) {
	type resp = infer.UpdateResponse[RegulatoryBundleState]
	st := req.State
	st.RegulatoryBundleArgs = req.Inputs
	if req.DryRun {
		return resp{Output: st}, nil
	}
	c, err := clientFor(ctx)
	if err != nil {
		return resp{}, err
	}
	in := req.Inputs
	form := url.Values{"FriendlyName": {in.FriendlyName}, "Email": {in.Email}}
	setS(form, "StatusCallback", in.StatusCallback)
	if derefBool(in.Submit) && st.Status == "draft" {
		form.Set("Status", "pending-review")
	}
	o, err := c.do(ctx, "POST", numbersProduct, nrBundlePath(req.ID), nil, form)
	if err != nil {
		return resp{}, fmt.Errorf("update regulatory bundle %s: %w", req.ID, err)
	}
	st.Status, st.ValidUntil, st.DateUpdated = o.str("status"), o.str("valid_until"), o.str("date_updated")
	return resp{Output: st}, nil
}

func (r *RegulatoryBundle) Delete(ctx context.Context, req infer.DeleteRequest[RegulatoryBundleState]) (infer.DeleteResponse, error) {
	return infer.DeleteResponse{}, thDelete(ctx, numbersRegAPI, "bundle", "Bundles", req.ID)
}

// ------------------------------------------------------------ RegulatoryBundleItemAssignment

// RegulatoryBundleItemAssignment assigns an end user or a supporting document to a regulatory bundle.
type RegulatoryBundleItemAssignment struct{}

// RegulatoryBundleItemAssignmentArgs are the inputs of a RegulatoryBundleItemAssignment.
type RegulatoryBundleItemAssignmentArgs struct {
	BundleSid string `pulumi:"bundleSid" provider:"replaceOnChanges"`
	ObjectSid string `pulumi:"objectSid" provider:"replaceOnChanges"`
}

func (a *RegulatoryBundleItemAssignmentArgs) Annotate(an infer.Annotator) {
	an.Describe(&a.BundleSid, "SID (BU...) of the bundle. Changing it replaces the assignment.")
	an.Describe(&a.ObjectSid, "SID of the end user (IT...) or supporting document (RD...) to assign. Changing it replaces the assignment.")
}

// RegulatoryBundleItemAssignmentState is the persisted state of a RegulatoryBundleItemAssignment.
type RegulatoryBundleItemAssignmentState struct {
	RegulatoryBundleItemAssignmentArgs

	Sid         string `pulumi:"sid"`
	AccountSid  string `pulumi:"accountSid"`
	DateCreated string `pulumi:"dateCreated"`
}

func (s *RegulatoryBundleItemAssignmentState) Annotate(a infer.Annotator) {
	a.Describe(&s.Sid, "The assignment SID (BV...). The resource ID is the bundle SID and this SID joined by a slash.")
	a.Describe(&s.AccountSid, "SID (AC...) of the account that owns the assignment.")
	a.Describe(&s.DateCreated, "When the assignment was created (ISO 8601).")
}

func (r *RegulatoryBundleItemAssignment) Annotate(a infer.Annotator) {
	a.Describe(&r, "Assigns a Numbers regulatory end user or supporting document to a regulatory bundle (numbers.twilio.com /v2/RegulatoryCompliance/Bundles/{sid}/ItemAssignments). Every input replaces, because Twilio has no update call. "+
		"Twilio only accepts the assignment while the bundle is a draft and refuses to remove one once it is in review. Import with the ID <bundleSid>/<assignmentSid>. Not verified against a live account.")
}

var _ infer.CustomResource[RegulatoryBundleItemAssignmentArgs, RegulatoryBundleItemAssignmentState] = (*RegulatoryBundleItemAssignment)(nil)
var _ infer.CustomCheck[RegulatoryBundleItemAssignmentArgs] = (*RegulatoryBundleItemAssignment)(nil)
var _ infer.CustomDiff[RegulatoryBundleItemAssignmentArgs, RegulatoryBundleItemAssignmentState] = (*RegulatoryBundleItemAssignment)(nil)
var _ infer.CustomRead[RegulatoryBundleItemAssignmentArgs, RegulatoryBundleItemAssignmentState] = (*RegulatoryBundleItemAssignment)(nil)
var _ infer.CustomDelete[RegulatoryBundleItemAssignmentState] = (*RegulatoryBundleItemAssignment)(nil)

func (r *RegulatoryBundleItemAssignment) Check(ctx context.Context, req infer.CheckRequest) (infer.CheckResponse[RegulatoryBundleItemAssignmentArgs], error) {
	args, failures, err := infer.DefaultCheck[RegulatoryBundleItemAssignmentArgs](ctx, req.NewInputs)
	if err != nil || len(failures) > 0 {
		return infer.CheckResponse[RegulatoryBundleItemAssignmentArgs]{Inputs: args, Failures: failures}, err
	}
	c := &checker{unk: unknowns(req.NewInputs)}
	c.sid("bundleSid", args.BundleSid, bundleSidRE, "bundle (BU...)")
	c.sid("objectSid", args.ObjectSid, objectSidRE, "end user (IT...) or supporting document (RD...)")
	return infer.CheckResponse[RegulatoryBundleItemAssignmentArgs]{Inputs: args, Failures: c.failures}, nil
}

func (r *RegulatoryBundleItemAssignment) Diff(_ context.Context, req infer.DiffRequest[RegulatoryBundleItemAssignmentArgs, RegulatoryBundleItemAssignmentState]) (infer.DiffResponse, error) {
	o, n := req.State.RegulatoryBundleItemAssignmentArgs, req.Inputs
	d := trDiffSet{}
	d.replace("bundleSid", o.BundleSid, n.BundleSid)
	d.replace("objectSid", o.ObjectSid, n.ObjectSid)
	return d.response(), nil
}

func (r *RegulatoryBundleItemAssignment) Create(ctx context.Context, req infer.CreateRequest[RegulatoryBundleItemAssignmentArgs]) (infer.CreateResponse[RegulatoryBundleItemAssignmentState], error) {
	type resp = infer.CreateResponse[RegulatoryBundleItemAssignmentState]
	a := req.Inputs
	st := RegulatoryBundleItemAssignmentState{RegulatoryBundleItemAssignmentArgs: a}
	if req.DryRun {
		return resp{Output: st}, nil
	}
	c, err := clientFor(ctx)
	if err != nil {
		return resp{}, err
	}
	o, err := c.do(ctx, "POST", numbersProduct, nrItemPath(a.BundleSid, ""), nil, url.Values{"ObjectSid": {a.ObjectSid}})
	if err != nil {
		return resp{}, fmt.Errorf("assign %s to regulatory bundle %s: %w", a.ObjectSid, a.BundleSid, err)
	}
	if o.str("sid") == "" {
		return resp{}, fmt.Errorf("assign %s to regulatory bundle %s: Twilio returned no SID", a.ObjectSid, a.BundleSid)
	}
	st.Sid, st.AccountSid, st.DateCreated = o.str("sid"), o.str("account_sid"), o.str("date_created")
	return resp{ID: a.BundleSid + "/" + st.Sid, Output: st}, nil
}

func nrFetchItem(ctx context.Context, bundle, sid string) (obj, bool, error) {
	c, err := clientFor(ctx)
	if err != nil {
		return nil, false, err
	}
	o, err := c.do(ctx, "GET", numbersProduct, nrItemPath(bundle, sid), nil, nil)
	if err != nil {
		if isNotFound(err) {
			return nil, false, nil
		}
		return nil, false, fmt.Errorf("read regulatory bundle item assignment %s/%s: %w", bundle, sid, err)
	}
	return o, true, nil
}

func (r *RegulatoryBundleItemAssignment) Read(ctx context.Context, req infer.ReadRequest[RegulatoryBundleItemAssignmentArgs, RegulatoryBundleItemAssignmentState]) (infer.ReadResponse[RegulatoryBundleItemAssignmentArgs, RegulatoryBundleItemAssignmentState], error) {
	type resp = infer.ReadResponse[RegulatoryBundleItemAssignmentArgs, RegulatoryBundleItemAssignmentState]
	bundle, sid, err := splitAssignmentID(req.ID)
	if err != nil {
		return resp{}, err
	}
	o, found, err := nrFetchItem(ctx, bundle, sid)
	if err != nil {
		return resp{}, err
	}
	if !found {
		if err := notFoundOnImport("regulatory bundle item assignment", req.ID, req.Inputs, req.State); err != nil {
			return resp{}, err
		}
		return resp{}, nil
	}
	in := req.Inputs
	in.BundleSid, in.ObjectSid = bundle, o.str("object_sid")
	st := RegulatoryBundleItemAssignmentState{RegulatoryBundleItemAssignmentArgs: in, Sid: sid, AccountSid: o.str("account_sid"), DateCreated: o.str("date_created")}
	return resp{ID: req.ID, Inputs: in, State: st}, nil
}

func (r *RegulatoryBundleItemAssignment) Delete(ctx context.Context, req infer.DeleteRequest[RegulatoryBundleItemAssignmentState]) (infer.DeleteResponse, error) {
	bundle, sid, err := splitAssignmentID(req.ID)
	if err != nil {
		return infer.DeleteResponse{}, err
	}
	c, err := clientFor(ctx)
	if err != nil {
		return infer.DeleteResponse{}, err
	}
	if _, err := c.do(ctx, "DELETE", numbersProduct, nrItemPath(bundle, sid), nil, nil); err != nil && !isNotFound(err) {
		return infer.DeleteResponse{}, fmt.Errorf("delete regulatory bundle item assignment %s/%s: %w", bundle, sid, err)
	}
	return infer.DeleteResponse{}, nil
}
