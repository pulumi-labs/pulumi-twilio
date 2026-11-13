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

// ProxyPhoneNumber adds a Twilio phone number of the account to the pool of a Proxy service.
type ProxyPhoneNumber struct{}

// ProxyPhoneNumberArgs are the inputs of a ProxyPhoneNumber.
type ProxyPhoneNumberArgs struct {
	ServiceSid     string  `pulumi:"serviceSid"`
	PhoneNumberSid *string `pulumi:"phoneNumberSid,optional"`
	PhoneNumber    *string `pulumi:"phoneNumber,optional"`
	IsReserved     *bool   `pulumi:"isReserved,optional"`
}

func (a *ProxyPhoneNumberArgs) Annotate(an infer.Annotator) {
	an.Describe(&a.ServiceSid, "SID (KS...) of the Proxy service whose pool the number joins. Changing it replaces the resource.")
	an.Describe(&a.PhoneNumberSid, "SID (PN...) of the account's IncomingPhoneNumber to add. Set exactly one of phoneNumberSid and phoneNumber. Changing it replaces the resource.")
	an.Describe(&a.PhoneNumber, "The number to add in E.164 format, for example +14155551234. Set exactly one of phoneNumberSid and phoneNumber. Changing it replaces the resource.")
	an.Describe(&a.IsReserved, "Whether the number is reserved: a reserved number is never assigned to a participant by the pool logic (Twilio default false).")
}

// ProxyPhoneNumberState is the persisted state of a ProxyPhoneNumber. Whichever of phoneNumberSid
// and phoneNumber the program left out is filled in from what Twilio reports, so the state holds
// both while the inputs (which Check limits to one) hold only what the program or an import set.
type ProxyPhoneNumberState struct {
	ProxyPhoneNumberArgs

	Sid          string  `pulumi:"sid"`
	FriendlyName *string `pulumi:"friendlyName,optional"`
	IsoCountry   *string `pulumi:"isoCountry,optional"`
	InUse        *int    `pulumi:"inUse,optional"`
	DateCreated  string  `pulumi:"dateCreated"`
	DateUpdated  string  `pulumi:"dateUpdated"`
}

func (s *ProxyPhoneNumberState) Annotate(a infer.Annotator) {
	a.Describe(&s.Sid, "The SID (PN...) of the phone number, the same as phoneNumberSid.")
	a.Describe(&s.FriendlyName, "Friendly name Twilio reports for the number.")
	a.Describe(&s.IsoCountry, "ISO country code of the number.")
	a.Describe(&s.InUse, "How many open sessions currently use the number.")
	a.Describe(&s.DateCreated, "When the number was added to the service (ISO 8601).")
	a.Describe(&s.DateUpdated, "When the entry was last updated (ISO 8601).")
}

func (r *ProxyPhoneNumber) Annotate(a infer.Annotator) {
	a.Describe(&r, "A phone number in the pool of a Proxy service. The resource ID is <serviceSid>/<sid>. Only isReserved updates in place; the service and the number replace. Destroying it removes the number from the pool and leaves the number itself in the account.")
}

var _ infer.CustomResource[ProxyPhoneNumberArgs, ProxyPhoneNumberState] = (*ProxyPhoneNumber)(nil)
var _ infer.CustomCheck[ProxyPhoneNumberArgs] = (*ProxyPhoneNumber)(nil)
var _ infer.CustomDiff[ProxyPhoneNumberArgs, ProxyPhoneNumberState] = (*ProxyPhoneNumber)(nil)
var _ infer.CustomRead[ProxyPhoneNumberArgs, ProxyPhoneNumberState] = (*ProxyPhoneNumber)(nil)
var _ infer.CustomUpdate[ProxyPhoneNumberArgs, ProxyPhoneNumberState] = (*ProxyPhoneNumber)(nil)
var _ infer.CustomDelete[ProxyPhoneNumberState] = (*ProxyPhoneNumber)(nil)

func (r *ProxyPhoneNumber) Check(ctx context.Context, req infer.CheckRequest) (infer.CheckResponse[ProxyPhoneNumberArgs], error) {
	args, failures, err := infer.DefaultCheck[ProxyPhoneNumberArgs](ctx, req.NewInputs)
	if err != nil || len(failures) > 0 {
		return infer.CheckResponse[ProxyPhoneNumberArgs]{Inputs: args, Failures: failures}, err
	}
	c := &checker{unk: unknowns(req.NewInputs)}
	c.sid("serviceSid", args.ServiceSid, proxyServiceSidRE, "Proxy service (KS followed by 32 hex characters)")
	if !c.unk("phoneNumberSid", "phoneNumber") {
		switch {
		case args.PhoneNumberSid == nil && args.PhoneNumber == nil:
			c.fail("phoneNumber", "one of phoneNumberSid and phoneNumber must be set")
		case args.PhoneNumberSid != nil && args.PhoneNumber != nil:
			c.fail("phoneNumber", "set only one of phoneNumberSid and phoneNumber")
		}
	}
	if v := args.PhoneNumberSid; v != nil {
		c.sid("phoneNumberSid", *v, proxyNumberSidRE, "phone number (PN followed by 32 hex characters)")
	}
	if v := args.PhoneNumber; v != nil && !c.unk("phoneNumber") && !proxyE164RE.MatchString(*v) {
		c.fail("phoneNumber", "phoneNumber must be in E.164 format, for example +14155551234")
	}
	return infer.CheckResponse[ProxyPhoneNumberArgs]{Inputs: args, Failures: c.failures}, nil
}

func (r *ProxyPhoneNumber) Diff(_ context.Context, req infer.DiffRequest[ProxyPhoneNumberArgs, ProxyPhoneNumberState]) (infer.DiffResponse, error) {
	o, n := req.State.ProxyPhoneNumberArgs, req.Inputs
	d := trDiffSet{}
	d.replace("serviceSid", o.ServiceSid, n.ServiceSid)
	// The state holds both identifiers of the number, so a program that names it the other way
	// round is not a change.
	if n.PhoneNumberSid != nil && (o.PhoneNumberSid == nil || *o.PhoneNumberSid != *n.PhoneNumberSid) {
		d["phoneNumberSid"] = p.PropertyDiff{Kind: p.UpdateReplace}
	}
	if n.PhoneNumber != nil && (o.PhoneNumber == nil || *o.PhoneNumber != *n.PhoneNumber) {
		d["phoneNumber"] = p.PropertyDiff{Kind: p.UpdateReplace}
	}
	d.boolDef(proxyPhoneNumberDefaults, "isReserved", o.IsReserved, n.IsReserved)
	return d.response(), nil
}

// fill records what Twilio reports about the number in the state.
func (st *ProxyPhoneNumberState) fill(o obj) {
	st.Sid, st.DateCreated, st.DateUpdated = trSid(o)
	st.FriendlyName = nonEmpty(o.optStr("friendly_name"))
	st.IsoCountry = nonEmpty(o.optStr("iso_country"))
	st.InUse = o.optInt("in_use")
	if st.PhoneNumberSid == nil && st.Sid != "" {
		sid := st.Sid
		st.PhoneNumberSid = &sid
	}
	if st.PhoneNumber == nil {
		st.PhoneNumber = nonEmpty(o.optStr("phone_number"))
	}
}

func (r *ProxyPhoneNumber) Create(ctx context.Context, req infer.CreateRequest[ProxyPhoneNumberArgs]) (infer.CreateResponse[ProxyPhoneNumberState], error) {
	type resp = infer.CreateResponse[ProxyPhoneNumberState]
	a := req.Inputs
	if req.DryRun {
		return resp{Output: ProxyPhoneNumberState{ProxyPhoneNumberArgs: a}}, nil
	}
	c, err := clientFor(ctx)
	if err != nil {
		return resp{}, err
	}
	form := url.Values{}
	setS(form, "Sid", a.PhoneNumberSid)
	setS(form, "PhoneNumber", a.PhoneNumber)
	setB(form, "IsReserved", a.IsReserved)
	o, err := c.do(ctx, "POST", proxyProduct, proxyPath(a.ServiceSid, "PhoneNumbers"), nil, form)
	if err != nil {
		return resp{}, fmt.Errorf("add phone number to Proxy service %s: %w", a.ServiceSid, err)
	}
	st := ProxyPhoneNumberState{ProxyPhoneNumberArgs: a}
	st.fill(o)
	return resp{ID: a.ServiceSid + "/" + st.Sid, Output: st}, nil
}

func (r *ProxyPhoneNumber) Read(ctx context.Context, req infer.ReadRequest[ProxyPhoneNumberArgs, ProxyPhoneNumberState]) (infer.ReadResponse[ProxyPhoneNumberArgs, ProxyPhoneNumberState], error) {
	type resp = infer.ReadResponse[ProxyPhoneNumberArgs, ProxyPhoneNumberState]
	parts, err := splitVerifyID(req.ID, 2, "<serviceSid>/<sid>")
	if err != nil {
		return resp{}, err
	}
	c, err := clientFor(ctx)
	if err != nil {
		return resp{}, err
	}
	o, err := c.do(ctx, "GET", proxyProduct, proxyPath(parts[0], "PhoneNumbers", parts[1]), nil, nil)
	if err != nil {
		if isNotFound(err) {
			if err := notFoundOnImport("Proxy phone number", req.ID, req.Inputs, req.State); err != nil {
				return resp{}, err
			}
			return resp{}, nil
		}
		return resp{}, fmt.Errorf("read Proxy phone number %s: %w", req.ID, err)
	}
	in := req.Inputs
	// An import arrives with no prior state; the program's own settings are the only record of what it manages.
	imported := req.State.Sid == ""
	in.ServiceSid = parts[0]
	in.IsReserved = trBool(in.IsReserved, o.optBool("is_reserved"), proxyPhoneNumberDefaults.boolean("isReserved"), imported)
	st := ProxyPhoneNumberState{ProxyPhoneNumberArgs: in}
	st.fill(o)
	if imported {
		// Check refuses a program that sets both identifiers, so an import adopts only the SID as
		// the input; the number itself is kept in the state, where Diff finds it if the program
		// names the number that way instead.
		in.PhoneNumberSid = st.PhoneNumberSid
		st.ProxyPhoneNumberArgs.PhoneNumberSid = st.PhoneNumberSid
	}
	return resp{ID: req.ID, Inputs: in, State: st}, nil
}

func (r *ProxyPhoneNumber) Update(ctx context.Context, req infer.UpdateRequest[ProxyPhoneNumberArgs, ProxyPhoneNumberState]) (infer.UpdateResponse[ProxyPhoneNumberState], error) {
	type resp = infer.UpdateResponse[ProxyPhoneNumberState]
	st := req.State
	st.ProxyPhoneNumberArgs = req.Inputs
	// The identifiers cannot change in place, so keep the pair the state already holds.
	st.PhoneNumberSid, st.PhoneNumber = req.State.PhoneNumberSid, req.State.PhoneNumber
	if req.DryRun {
		return resp{Output: st}, nil
	}
	parts, err := splitVerifyID(req.ID, 2, "<serviceSid>/<sid>")
	if err != nil {
		return resp{}, err
	}
	c, err := clientFor(ctx)
	if err != nil {
		return resp{}, err
	}
	form := url.Values{}
	setB(form, "IsReserved", req.Inputs.IsReserved)
	o, err := c.do(ctx, "POST", proxyProduct, proxyPath(parts[0], "PhoneNumbers", parts[1]), nil, form)
	if err != nil {
		return resp{}, fmt.Errorf("update Proxy phone number %s: %w", req.ID, err)
	}
	st.DateUpdated = o.str("date_updated")
	st.InUse = o.optInt("in_use")
	return resp{Output: st}, nil
}

func (r *ProxyPhoneNumber) Delete(ctx context.Context, req infer.DeleteRequest[ProxyPhoneNumberState]) (infer.DeleteResponse, error) {
	parts, err := splitVerifyID(req.ID, 2, "<serviceSid>/<sid>")
	if err != nil {
		return infer.DeleteResponse{}, err
	}
	c, err := clientFor(ctx)
	if err != nil {
		return infer.DeleteResponse{}, err
	}
	if _, err := c.do(ctx, "DELETE", proxyProduct, proxyPath(parts[0], "PhoneNumbers", parts[1]), nil, nil); err != nil && !isNotFound(err) {
		return infer.DeleteResponse{}, fmt.Errorf("remove phone number %s from its Proxy service: %w", req.ID, err)
	}
	return infer.DeleteResponse{}, nil
}

// proxyPhoneNumberDefaults are the Twilio defaults of this resource, shared by Read (import adoption) and Diff.
var proxyPhoneNumberDefaults = trDefaults{
	"isReserved": false,
}
