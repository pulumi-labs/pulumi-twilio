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

// ShortCode adopts an SMS short code the account already leases and manages its SMS
// webhooks. It never leases or releases a short code: Twilio provisions them through
// support, and the REST API offers no create or delete for them.
type ShortCode struct{}

// ShortCodeArgs are the inputs of a ShortCode.
type ShortCodeArgs struct {
	Code              *string `pulumi:"code,optional" provider:"replaceOnChanges"`
	ShortCodeSid      *string `pulumi:"shortCodeSid,optional" provider:"replaceOnChanges"`
	AccountSid        *string `pulumi:"accountSid,optional" provider:"replaceOnChanges"`
	FriendlyName      *string `pulumi:"friendlyName,optional"`
	SmsUrl            *string `pulumi:"smsUrl,optional"`
	SmsMethod         *string `pulumi:"smsMethod,optional"`
	SmsFallbackUrl    *string `pulumi:"smsFallbackUrl,optional"`
	SmsFallbackMethod *string `pulumi:"smsFallbackMethod,optional"`
	ApiVersion        *string `pulumi:"apiVersion,optional"`
}

func (a *ShortCodeArgs) props() props {
	return props{strs: []strProp{
		{"FriendlyName", "friendly_name", &a.FriendlyName},
		{"SmsUrl", "sms_url", &a.SmsUrl}, {"SmsMethod", "sms_method", &a.SmsMethod},
		{"SmsFallbackUrl", "sms_fallback_url", &a.SmsFallbackUrl},
		{"SmsFallbackMethod", "sms_fallback_method", &a.SmsFallbackMethod},
		{"ApiVersion", "api_version", &a.ApiVersion},
	}}
}

func (a *ShortCodeArgs) Annotate(an infer.Annotator) {
	an.Describe(&a.Code, "The leased short code, for example 894546. Set exactly one of code or shortCodeSid. "+
		"Changing it replaces the resource (a different short code is adopted).")
	an.Describe(&a.ShortCodeSid, "SID (SC...) of the leased short code. Set exactly one of code or shortCodeSid. "+
		"Changing it replaces the resource.")
	an.Describe(&a.AccountSid, "SID of the account or subaccount that owns the short code. Defaults to the provider account. "+
		"Twilio cannot move a short code between accounts through this API, so changing it replaces the resource (which only "+
		"adopts another account's short code; nothing is released). To import a short code owned by a subaccount use the ID "+
		"<accountSid>/<shortCodeSid>.")
	an.Describe(&a.FriendlyName, "Human-readable label for the short code. Removing it leaves the label unchanged.")
	an.Describe(&a.SmsUrl, "URL Twilio requests for incoming SMS to the short code. Removing it clears the URL.")
	an.Describe(&a.SmsMethod, "HTTP method for smsUrl: GET or POST.")
	an.Describe(&a.SmsFallbackUrl, "URL requested if smsUrl fails. Removing it clears the URL.")
	an.Describe(&a.SmsFallbackMethod, "HTTP method for smsFallbackUrl: GET or POST.")
	an.Describe(&a.ApiVersion, "Twilio API version used for requests to the short code's webhooks, for example 2010-04-01. "+
		"Removing it leaves the version on Twilio unchanged.")
}

// ShortCodeState is the persisted state of a ShortCode.
type ShortCodeState struct {
	ShortCodeArgs

	// Sid is the short code SID (SC...) and the resource ID.
	Sid string `pulumi:"sid"`
	// Number is the short code itself.
	Number string `pulumi:"number"`
	// OwnerAccountSid is the account that owns the short code.
	OwnerAccountSid string `pulumi:"ownerAccountSid"`
	// Effective is the configuration Twilio reports, including fields the program did not set.
	Effective *ShortCodeEffective `pulumi:"effective,optional"`
}

func (s *ShortCodeState) Annotate(a infer.Annotator) {
	a.Describe(&s.Sid, "The short code SID (SC...).")
	a.Describe(&s.Number, "The short code.")
	a.Describe(&s.OwnerAccountSid, "SID of the account that owns the short code.")
	a.Describe(&s.Effective, "The configuration Twilio currently reports for the short code, including webhooks changed "+
		"outside Pulumi and fields the program does not set. Inputs the program leaves unset are not overwritten by these values.")
}

func (r *ShortCode) Annotate(a infer.Annotator) {
	a.Describe(&r, "SMS webhook configuration for a short code the account already leases. The resource adopts the short code; "+
		"it never leases one, and destroying it does NOT release the short code (it only stops managing it, leaving the current "+
		"webhooks in place).")
}

var _ infer.CustomResource[ShortCodeArgs, ShortCodeState] = (*ShortCode)(nil)
var _ infer.CustomCheck[ShortCodeArgs] = (*ShortCode)(nil)
var _ infer.CustomRead[ShortCodeArgs, ShortCodeState] = (*ShortCode)(nil)
var _ infer.CustomUpdate[ShortCodeArgs, ShortCodeState] = (*ShortCode)(nil)
var _ infer.CustomDelete[ShortCodeState] = (*ShortCode)(nil)

func (r *ShortCode) Check(ctx context.Context, req infer.CheckRequest) (infer.CheckResponse[ShortCodeArgs], error) {
	args, failures, err := infer.DefaultCheck[ShortCodeArgs](ctx, req.NewInputs)
	if err != nil || len(failures) > 0 {
		return infer.CheckResponse[ShortCodeArgs]{Inputs: args, Failures: failures}, err
	}
	unk := unknowns(req.NewInputs)
	// A computed identity counts as set: its value is only known at apply time.
	hasNum := unk("code") || deref(args.Code) != ""
	hasSid := unk("shortCodeSid") || deref(args.ShortCodeSid) != ""
	if hasNum == hasSid {
		failures = append(failures, p.CheckFailure{Property: "code", Reason: "set exactly one of code or shortCodeSid"})
	}
	failures = checkMethods(failures, unk, map[string]*string{"smsMethod": args.SmsMethod, "smsFallbackMethod": args.SmsFallbackMethod})
	return infer.CheckResponse[ShortCodeArgs]{Inputs: args, Failures: failures}, nil
}

// findShortCode resolves the leased short code named by a SID or by the code itself.
func findShortCode(ctx context.Context, c *Client, acct, sid, code string) (obj, error) {
	if sid != "" {
		return fetchAdopted(ctx, c, acct, shortCodeColl, sid)
	}
	return findAdopted(ctx, c, acct, shortCodeColl, "ShortCode", "short_code", code)
}

// ShortCodeEffective is the configuration Twilio reports for a ShortCode: every field of the inputs, filled from the API response.
type ShortCodeEffective ShortCodeArgs

// Annotate reuses the input descriptions.
func (e *ShortCodeEffective) Annotate(a infer.Annotator) { (*ShortCodeArgs)(e).Annotate(a) }

func shortCodeEffective(o obj) *ShortCodeEffective {
	e := ShortCodeArgs{}
	e.props().fillAll(o)
	v := ShortCodeEffective(e)
	return &v
}

func (r *ShortCode) Create(ctx context.Context, req infer.CreateRequest[ShortCodeArgs]) (infer.CreateResponse[ShortCodeState], error) {
	a := req.Inputs
	if req.DryRun {
		return infer.CreateResponse[ShortCodeState]{ID: "", Output: ShortCodeState{ShortCodeArgs: a}}, nil
	}
	c, err := clientFor(ctx)
	if err != nil {
		return infer.CreateResponse[ShortCodeState]{}, err
	}
	acct := c.acct(a.AccountSid)
	cur, err := findShortCode(ctx, c, acct, deref(a.ShortCodeSid), deref(a.Code))
	if err != nil {
		return infer.CreateResponse[ShortCodeState]{}, fmt.Errorf("adopt short code: %w", err)
	}
	sid := cur.str("sid")
	st := ShortCodeState{ShortCodeArgs: a, Sid: sid, Number: cur.str("short_code"), OwnerAccountSid: acct, Effective: shortCodeEffective(cur)}
	form := url.Values{}
	a.props().encode(form, nil)
	if len(form) > 0 {
		o, err := c.do(ctx, "POST", "api", acctPath(acct, shortCodeColl.coll, sid), nil, form)
		if err != nil {
			// The short code is adopted but not configured: save it so the next update retries.
			return infer.CreateResponse[ShortCodeState]{ID: scopedID(c, acct, sid), Output: st},
				infer.ResourceInitFailedError{Reasons: []string{fmt.Sprintf("configure short code %s: %v", sid, err)}}
		}
		st.Effective = shortCodeEffective(o)
	}
	return infer.CreateResponse[ShortCodeState]{ID: scopedID(c, acct, sid), Output: st}, nil
}

func (r *ShortCode) Read(ctx context.Context, req infer.ReadRequest[ShortCodeArgs, ShortCodeState]) (infer.ReadResponse[ShortCodeArgs, ShortCodeState], error) {
	c, err := clientFor(ctx)
	if err != nil {
		return infer.ReadResponse[ShortCodeArgs, ShortCodeState]{}, err
	}
	acct, sid := resolveScope(c, req.ID, req.State.OwnerAccountSid)
	o, err := c.do(ctx, "GET", "api", acctPath(acct, shortCodeColl.coll, sid), nil, nil)
	if err != nil {
		if isNotFound(err) {
			if err := notFoundOnImport("short code", req.ID, req.Inputs, req.State); err != nil {
				return infer.ReadResponse[ShortCodeArgs, ShortCodeState]{}, err
			}
			return infer.ReadResponse[ShortCodeArgs, ShortCodeState]{}, nil
		}
		return infer.ReadResponse[ShortCodeArgs, ShortCodeState]{}, fmt.Errorf("read short code %s: %w", req.ID, err)
	}
	in := req.Inputs
	if in.Code == nil && in.ShortCodeSid == nil { // an import
		s := sid
		in.ShortCodeSid = &s
		if acct != c.accountSid {
			in.AccountSid = &acct
		}
		in.props().adopt(o)
	} else {
		in.props().refreshSet(o)
	}
	st := ShortCodeState{ShortCodeArgs: in, Sid: sid, Number: o.str("short_code"),
		OwnerAccountSid: derefOr(nonEmpty(o.optStr("account_sid")), acct), Effective: shortCodeEffective(o)}
	return infer.ReadResponse[ShortCodeArgs, ShortCodeState]{ID: req.ID, Inputs: in, State: st}, nil
}

func (r *ShortCode) Update(ctx context.Context, req infer.UpdateRequest[ShortCodeArgs, ShortCodeState]) (infer.UpdateResponse[ShortCodeState], error) {
	a := req.Inputs
	st := req.State
	st.ShortCodeArgs = a
	if req.DryRun { // a preview needs no credentials
		return infer.UpdateResponse[ShortCodeState]{Output: st}, nil
	}
	c, err := clientFor(ctx)
	if err != nil {
		return infer.UpdateResponse[ShortCodeState]{}, err
	}
	acct, sid := resolveScope(c, req.ID, req.State.OwnerAccountSid)
	form := url.Values{}
	prev := req.State.ShortCodeArgs.props()
	a.props().encode(form, &prev)
	if len(form) > 0 {
		o, err := c.do(ctx, "POST", "api", acctPath(acct, shortCodeColl.coll, sid), nil, form)
		if err != nil {
			return infer.UpdateResponse[ShortCodeState]{}, fmt.Errorf("update short code %s: %w", req.ID, err)
		}
		st.Effective = shortCodeEffective(o)
	}
	return infer.UpdateResponse[ShortCodeState]{Output: st}, nil
}

// Delete only stops managing the short code: it is never released.
func (r *ShortCode) Delete(ctx context.Context, req infer.DeleteRequest[ShortCodeState]) (infer.DeleteResponse, error) {
	return infer.DeleteResponse{}, nil
}
