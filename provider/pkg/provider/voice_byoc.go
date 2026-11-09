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
	"strings"

	p "github.com/pulumi/pulumi-go-provider"
	"github.com/pulumi/pulumi-go-provider/infer"
)

// Bring Your Own Carrier (BYOC) lives on voice.twilio.com (Voice API v1). Nothing in its paths names
// an account: the resources belong to the account of the provider credentials.

const voiceProduct = "voice"

func connectionPolicyPath(sid string) string {
	path := "/v1/ConnectionPolicies"
	if sid != "" {
		path += "/" + url.PathEscape(sid)
	}
	return path
}

func byocTrunkPath(sid string) string {
	path := "/v1/ByocTrunks"
	if sid != "" {
		path += "/" + url.PathEscape(sid)
	}
	return path
}

// checkSidPrefix rejects a SID that does not start with the prefix of its resource type.
func checkSidPrefix(failures []p.CheckFailure, unk func(...string) bool, prop string, v *string, prefix, what string) []p.CheckFailure {
	if v != nil && !unk(prop) && !strings.HasPrefix(*v, prefix) {
		failures = append(failures, p.CheckFailure{Property: prop, Reason: prop + " must be the SID (" + prefix + "...) of " + what})
	}
	return failures
}

// ------------------------------------------------------------ ConnectionPolicy

// ConnectionPolicy manages a Twilio BYOC connection policy: the list of carrier SIP targets that a
// BYOC trunk sends outbound calls to. Add the targets with ConnectionPolicyTarget.
type ConnectionPolicy struct{}

// ConnectionPolicyArgs are the inputs of a ConnectionPolicy.
type ConnectionPolicyArgs struct {
	FriendlyName *string `pulumi:"friendlyName,optional"`
}

func (a *ConnectionPolicyArgs) Annotate(an infer.Annotator) {
	an.Describe(&a.FriendlyName, "Human-readable name of the connection policy. Removing it leaves the name unchanged on Twilio.")
}

// ConnectionPolicyState is the persisted state of a ConnectionPolicy.
type ConnectionPolicyState struct {
	ConnectionPolicyArgs

	// Sid is the connection policy SID (NY...) and the resource ID.
	Sid string `pulumi:"sid"`
	// AccountSid is the account that owns the policy.
	AccountSid string `pulumi:"accountSid"`
	// DateCreated and DateUpdated are ISO 8601 timestamps.
	DateCreated string `pulumi:"dateCreated"`
	DateUpdated string `pulumi:"dateUpdated"`
}

func (s *ConnectionPolicyState) Annotate(a infer.Annotator) {
	a.Describe(&s.Sid, "The connection policy SID (NY...).")
	a.Describe(&s.AccountSid, "SID of the account that owns the connection policy.")
	a.Describe(&s.DateCreated, "When the policy was created (ISO 8601).")
	a.Describe(&s.DateUpdated, "When the policy was last updated (ISO 8601).")
}

func (r *ConnectionPolicy) Annotate(a infer.Annotator) {
	a.Describe(&r, "A Twilio BYOC (Bring Your Own Carrier) connection policy: the carrier targets a ByocTrunk sends outbound calls to. "+
		"Add targets with ConnectionPolicyTarget and reference the policy from ByocTrunk.connectionPolicySid.")
}

var _ infer.CustomResource[ConnectionPolicyArgs, ConnectionPolicyState] = (*ConnectionPolicy)(nil)
var _ infer.CustomCheck[ConnectionPolicyArgs] = (*ConnectionPolicy)(nil)
var _ infer.CustomDiff[ConnectionPolicyArgs, ConnectionPolicyState] = (*ConnectionPolicy)(nil)
var _ infer.CustomRead[ConnectionPolicyArgs, ConnectionPolicyState] = (*ConnectionPolicy)(nil)
var _ infer.CustomUpdate[ConnectionPolicyArgs, ConnectionPolicyState] = (*ConnectionPolicy)(nil)
var _ infer.CustomDelete[ConnectionPolicyState] = (*ConnectionPolicy)(nil)

func (r *ConnectionPolicy) Check(ctx context.Context, req infer.CheckRequest) (infer.CheckResponse[ConnectionPolicyArgs], error) {
	args, failures, err := infer.DefaultCheck[ConnectionPolicyArgs](ctx, req.NewInputs)
	if err != nil || len(failures) > 0 {
		return infer.CheckResponse[ConnectionPolicyArgs]{Inputs: args, Failures: failures}, err
	}
	if args.FriendlyName != nil && !unknowns(req.NewInputs)("friendlyName") && *args.FriendlyName == "" {
		failures = append(failures, p.CheckFailure{Property: "friendlyName", Reason: "friendlyName must not be empty"})
	}
	return infer.CheckResponse[ConnectionPolicyArgs]{Inputs: args, Failures: failures}, nil
}

func (r *ConnectionPolicy) Diff(_ context.Context, req infer.DiffRequest[ConnectionPolicyArgs, ConnectionPolicyState]) (infer.DiffResponse, error) {
	detail := map[string]p.PropertyDiff{}
	if n := req.Inputs.FriendlyName; n != nil && deref(req.State.FriendlyName) != *n {
		detail["friendlyName"] = p.PropertyDiff{Kind: p.Update}
	}
	return infer.DiffResponse{HasChanges: len(detail) > 0, DetailedDiff: detail}, nil
}

func connectionPolicyOutputs(st *ConnectionPolicyState, o obj) {
	st.Sid, st.AccountSid = o.str("sid"), o.str("account_sid")
	st.DateCreated, st.DateUpdated = o.str("date_created"), o.str("date_updated")
}

func (a ConnectionPolicyArgs) form() url.Values {
	v := url.Values{}
	setS(v, "FriendlyName", a.FriendlyName)
	return v
}

func (r *ConnectionPolicy) Create(ctx context.Context, req infer.CreateRequest[ConnectionPolicyArgs]) (infer.CreateResponse[ConnectionPolicyState], error) {
	a := req.Inputs
	if req.DryRun {
		return infer.CreateResponse[ConnectionPolicyState]{Output: ConnectionPolicyState{ConnectionPolicyArgs: a}}, nil
	}
	c, err := clientFor(ctx)
	if err != nil {
		return infer.CreateResponse[ConnectionPolicyState]{}, err
	}
	o, err := c.do(ctx, "POST", voiceProduct, connectionPolicyPath(""), nil, a.form())
	if err != nil {
		return infer.CreateResponse[ConnectionPolicyState]{}, fmt.Errorf("create connection policy: %w", err)
	}
	st := ConnectionPolicyState{ConnectionPolicyArgs: a}
	connectionPolicyOutputs(&st, o)
	return infer.CreateResponse[ConnectionPolicyState]{ID: st.Sid, Output: st}, nil
}

func (r *ConnectionPolicy) Read(ctx context.Context, req infer.ReadRequest[ConnectionPolicyArgs, ConnectionPolicyState]) (infer.ReadResponse[ConnectionPolicyArgs, ConnectionPolicyState], error) {
	type resp = infer.ReadResponse[ConnectionPolicyArgs, ConnectionPolicyState]
	c, err := clientFor(ctx)
	if err != nil {
		return resp{}, err
	}
	o, err := c.do(ctx, "GET", voiceProduct, connectionPolicyPath(req.ID), nil, nil)
	if err != nil {
		if isNotFound(err) {
			if err := notFoundOnImport("connection policy", req.ID, req.Inputs, req.State); err != nil {
				return resp{}, err
			}
			return resp{}, nil
		}
		return resp{}, fmt.Errorf("read connection policy %s: %w", req.ID, err)
	}
	in := req.Inputs
	if got := o.optStr("friendly_name"); got != nil && (in.FriendlyName != nil || *got != "") {
		in.FriendlyName = got
	}
	st := ConnectionPolicyState{ConnectionPolicyArgs: in}
	connectionPolicyOutputs(&st, o)
	return resp{ID: req.ID, Inputs: in, State: st}, nil
}

func (r *ConnectionPolicy) Update(ctx context.Context, req infer.UpdateRequest[ConnectionPolicyArgs, ConnectionPolicyState]) (infer.UpdateResponse[ConnectionPolicyState], error) {
	a := req.Inputs
	st := req.State
	st.ConnectionPolicyArgs = a
	if req.DryRun {
		return infer.UpdateResponse[ConnectionPolicyState]{Output: st}, nil
	}
	c, err := clientFor(ctx)
	if err != nil {
		return infer.UpdateResponse[ConnectionPolicyState]{}, err
	}
	o, err := c.do(ctx, "POST", voiceProduct, connectionPolicyPath(req.ID), nil, a.form())
	if err != nil {
		return infer.UpdateResponse[ConnectionPolicyState]{}, fmt.Errorf("update connection policy %s: %w", req.ID, err)
	}
	connectionPolicyOutputs(&st, o)
	return infer.UpdateResponse[ConnectionPolicyState]{Output: st}, nil
}

func (r *ConnectionPolicy) Delete(ctx context.Context, req infer.DeleteRequest[ConnectionPolicyState]) (infer.DeleteResponse, error) {
	c, err := clientFor(ctx)
	if err != nil {
		return infer.DeleteResponse{}, err
	}
	if _, err := c.do(ctx, "DELETE", voiceProduct, connectionPolicyPath(req.ID), nil, nil); err != nil && !isNotFound(err) {
		return infer.DeleteResponse{}, fmt.Errorf("delete connection policy %s: %w", req.ID, err)
	}
	return infer.DeleteResponse{}, nil
}

// ------------------------------------------------------------------ ByocTrunk

// ByocTrunk manages a Twilio BYOC trunk: the connection between Twilio Programmable Voice and your
// own carrier. Outbound calls go to the targets of its connection policy; inbound calls from the
// carrier's IP addresses are mapped to it through SourceIpMapping.
type ByocTrunk struct{}

// ByocTrunkArgs are the inputs of a ByocTrunk. Every input is optional and updates in place.
type ByocTrunkArgs struct {
	FriendlyName         *string `pulumi:"friendlyName,optional"`
	VoiceUrl             *string `pulumi:"voiceUrl,optional"`
	VoiceMethod          *string `pulumi:"voiceMethod,optional"`
	VoiceFallbackUrl     *string `pulumi:"voiceFallbackUrl,optional"`
	VoiceFallbackMethod  *string `pulumi:"voiceFallbackMethod,optional"`
	StatusCallbackUrl    *string `pulumi:"statusCallbackUrl,optional"`
	StatusCallbackMethod *string `pulumi:"statusCallbackMethod,optional"`
	CnamLookupEnabled    *bool   `pulumi:"cnamLookupEnabled,optional"`
	ConnectionPolicySid  *string `pulumi:"connectionPolicySid,optional"`
	FromDomainSid        *string `pulumi:"fromDomainSid,optional"`
}

func (a *ByocTrunkArgs) Annotate(an infer.Annotator) {
	an.Describe(&a.FriendlyName, "Human-readable name of the BYOC trunk. Removing it leaves the name unchanged on Twilio.")
	an.Describe(&a.VoiceUrl, "URL Twilio requests (with voiceMethod) when a call arrives from the carrier and needs TwiML. Removing it clears the URL.")
	an.Describe(&a.VoiceMethod, "HTTP method used to request voiceUrl, GET or POST (Twilio default POST).")
	an.Describe(&a.VoiceFallbackUrl, "URL Twilio requests when voiceUrl fails or returns invalid TwiML. Removing it clears the URL.")
	an.Describe(&a.VoiceFallbackMethod, "HTTP method used to request voiceFallbackUrl, GET or POST (Twilio default POST).")
	an.Describe(&a.StatusCallbackUrl, "URL Twilio posts call status events to. Removing it clears the URL.")
	an.Describe(&a.StatusCallbackMethod, "HTTP method used to request statusCallbackUrl, GET or POST (Twilio default POST).")
	an.Describe(&a.CnamLookupEnabled, "Whether Caller ID Name (CNAM) lookup is enabled for inbound calls (default false). Removing the input leaves the setting unchanged on Twilio.")
	an.Describe(&a.ConnectionPolicySid, "SID (NY...) of the ConnectionPolicy whose targets receive the trunk's outbound calls. Removing it detaches the policy.")
	an.Describe(&a.FromDomainSid, "SID (SD...) of the SipDomain whose domain name is used as the From host of outbound calls. Removing it clears the setting.")
}

// ByocTrunkState is the persisted state of a ByocTrunk.
type ByocTrunkState struct {
	ByocTrunkArgs

	// Sid is the BYOC trunk SID (BY...) and the resource ID.
	Sid string `pulumi:"sid"`
	// AccountSid is the account that owns the trunk.
	AccountSid string `pulumi:"accountSid"`
	// DateCreated and DateUpdated are ISO 8601 timestamps.
	DateCreated string `pulumi:"dateCreated"`
	DateUpdated string `pulumi:"dateUpdated"`
}

func (s *ByocTrunkState) Annotate(a infer.Annotator) {
	a.Describe(&s.Sid, "The BYOC trunk SID (BY...).")
	a.Describe(&s.AccountSid, "SID of the account that owns the BYOC trunk.")
	a.Describe(&s.DateCreated, "When the trunk was created (ISO 8601).")
	a.Describe(&s.DateUpdated, "When the trunk was last updated (ISO 8601).")
}

func (r *ByocTrunk) Annotate(a infer.Annotator) {
	a.Describe(&r, "A Twilio BYOC (Bring Your Own Carrier) trunk. Point connectionPolicySid at a ConnectionPolicy for outbound calls "+
		"and map the carrier's IP addresses to it with IpRecord and SourceIpMapping for inbound calls.")
}

var _ infer.CustomResource[ByocTrunkArgs, ByocTrunkState] = (*ByocTrunk)(nil)
var _ infer.CustomCheck[ByocTrunkArgs] = (*ByocTrunk)(nil)
var _ infer.CustomDiff[ByocTrunkArgs, ByocTrunkState] = (*ByocTrunk)(nil)
var _ infer.CustomRead[ByocTrunkArgs, ByocTrunkState] = (*ByocTrunk)(nil)
var _ infer.CustomUpdate[ByocTrunkArgs, ByocTrunkState] = (*ByocTrunk)(nil)
var _ infer.CustomDelete[ByocTrunkState] = (*ByocTrunk)(nil)

// byocStr describes one string input: its property, form parameter and response key, the value
// Twilio uses when it is unset (treated as equal to nothing recorded), and whether removing the
// input clears it on Twilio.
type byocStr struct {
	prop, param, key, def string
	clear                 bool
	p                     **string
}

func (a *ByocTrunkArgs) strs() []byocStr {
	return []byocStr{
		{"friendlyName", "FriendlyName", "friendly_name", "", false, &a.FriendlyName},
		{"voiceUrl", "VoiceUrl", "voice_url", "", true, &a.VoiceUrl},
		{"voiceMethod", "VoiceMethod", "voice_method", "POST", false, &a.VoiceMethod},
		{"voiceFallbackUrl", "VoiceFallbackUrl", "voice_fallback_url", "", true, &a.VoiceFallbackUrl},
		{"voiceFallbackMethod", "VoiceFallbackMethod", "voice_fallback_method", "POST", false, &a.VoiceFallbackMethod},
		{"statusCallbackUrl", "StatusCallbackUrl", "status_callback_url", "", true, &a.StatusCallbackUrl},
		{"statusCallbackMethod", "StatusCallbackMethod", "status_callback_method", "POST", false, &a.StatusCallbackMethod},
		{"connectionPolicySid", "ConnectionPolicySid", "connection_policy_sid", "", true, &a.ConnectionPolicySid},
		{"fromDomainSid", "FromDomainSid", "from_domain_sid", "", true, &a.FromDomainSid},
	}
}

func (r *ByocTrunk) Check(ctx context.Context, req infer.CheckRequest) (infer.CheckResponse[ByocTrunkArgs], error) {
	args, failures, err := infer.DefaultCheck[ByocTrunkArgs](ctx, req.NewInputs)
	if err != nil || len(failures) > 0 {
		return infer.CheckResponse[ByocTrunkArgs]{Inputs: args, Failures: failures}, err
	}
	unk := unknowns(req.NewInputs)
	if args.FriendlyName != nil && !unk("friendlyName") && *args.FriendlyName == "" {
		failures = append(failures, p.CheckFailure{Property: "friendlyName", Reason: "friendlyName must not be empty"})
	}
	failures = checkMethods(failures, unk, map[string]*string{
		"voiceMethod": args.VoiceMethod, "voiceFallbackMethod": args.VoiceFallbackMethod, "statusCallbackMethod": args.StatusCallbackMethod,
	})
	failures = checkSidPrefix(failures, unk, "connectionPolicySid", args.ConnectionPolicySid, "NY", "a ConnectionPolicy")
	failures = checkSidPrefix(failures, unk, "fromDomainSid", args.FromDomainSid, "SD", "a SipDomain")
	return infer.CheckResponse[ByocTrunkArgs]{Inputs: args, Failures: failures}, nil
}

// Diff compares inputs with the recorded inputs. An unset plain input means "leave it alone", an unset
// URL or SID input means "cleared", and a stated Twilio default equals nothing recorded, so an imported
// trunk (which adopts only non-default values) and a program that spells out the defaults agree.
func (r *ByocTrunk) Diff(_ context.Context, req infer.DiffRequest[ByocTrunkArgs, ByocTrunkState]) (infer.DiffResponse, error) {
	o, n := req.State.ByocTrunkArgs, req.Inputs
	detail := map[string]p.PropertyDiff{}
	olds := o.strs()
	for i, f := range n.strs() {
		old, nw := *olds[i].p, *f.p
		switch {
		case nw == nil:
			if f.clear && deref(old) != "" {
				detail[f.prop] = p.PropertyDiff{Kind: p.Update}
			}
		case derefOr(old, f.def) != *nw:
			detail[f.prop] = p.PropertyDiff{Kind: p.Update}
		}
	}
	if n.CnamLookupEnabled != nil && derefBool(o.CnamLookupEnabled) != *n.CnamLookupEnabled {
		detail["cnamLookupEnabled"] = p.PropertyDiff{Kind: p.Update}
	}
	return infer.DiffResponse{HasChanges: len(detail) > 0, DetailedDiff: detail}, nil
}

// form encodes the settings to send; prev is the recorded inputs on an update.
func (a *ByocTrunkArgs) form(prev *ByocTrunkArgs) url.Values {
	v := url.Values{}
	var olds []byocStr
	if prev != nil {
		olds = prev.strs()
	}
	for i, f := range a.strs() {
		switch {
		case *f.p != nil:
			v.Set(f.param, **f.p)
		case prev != nil && f.clear && deref(*olds[i].p) != "":
			v.Set(f.param, "")
		}
	}
	setB(v, "CnamLookupEnabled", a.CnamLookupEnabled)
	return v
}

// refresh folds what Twilio reports into the inputs. An import adopts only non-default values; an
// existing resource refreshes the inputs the program set, so drift shows up as a diff.
func (a *ByocTrunkArgs) refresh(o obj, imported bool) {
	for _, f := range a.strs() {
		got := o.optStr(f.key)
		switch {
		case got == nil:
			if !imported && f.clear && *f.p != nil {
				cleared := ""
				*f.p = &cleared
			}
		case imported:
			if *got != f.def {
				*f.p = got
			}
		case *f.p != nil:
			*f.p = got
		}
	}
	if imported {
		// A method only matters with its URL.
		if a.VoiceUrl == nil {
			a.VoiceMethod = nil
		}
		if a.VoiceFallbackUrl == nil {
			a.VoiceFallbackMethod = nil
		}
		if a.StatusCallbackUrl == nil {
			a.StatusCallbackMethod = nil
		}
	}
	if got := o.optBool("cnam_lookup_enabled"); got != nil {
		if (imported && *got) || (!imported && a.CnamLookupEnabled != nil) {
			a.CnamLookupEnabled = got
		}
	}
}

func byocTrunkOutputs(st *ByocTrunkState, o obj) {
	st.Sid, st.AccountSid = o.str("sid"), o.str("account_sid")
	st.DateCreated, st.DateUpdated = o.str("date_created"), o.str("date_updated")
}

func (r *ByocTrunk) Create(ctx context.Context, req infer.CreateRequest[ByocTrunkArgs]) (infer.CreateResponse[ByocTrunkState], error) {
	a := req.Inputs
	if req.DryRun {
		return infer.CreateResponse[ByocTrunkState]{Output: ByocTrunkState{ByocTrunkArgs: a}}, nil
	}
	c, err := clientFor(ctx)
	if err != nil {
		return infer.CreateResponse[ByocTrunkState]{}, err
	}
	o, err := c.do(ctx, "POST", voiceProduct, byocTrunkPath(""), nil, a.form(nil))
	if err != nil {
		return infer.CreateResponse[ByocTrunkState]{}, fmt.Errorf("create BYOC trunk: %w", err)
	}
	st := ByocTrunkState{ByocTrunkArgs: a}
	byocTrunkOutputs(&st, o)
	return infer.CreateResponse[ByocTrunkState]{ID: st.Sid, Output: st}, nil
}

func (r *ByocTrunk) Read(ctx context.Context, req infer.ReadRequest[ByocTrunkArgs, ByocTrunkState]) (infer.ReadResponse[ByocTrunkArgs, ByocTrunkState], error) {
	type resp = infer.ReadResponse[ByocTrunkArgs, ByocTrunkState]
	c, err := clientFor(ctx)
	if err != nil {
		return resp{}, err
	}
	o, err := c.do(ctx, "GET", voiceProduct, byocTrunkPath(req.ID), nil, nil)
	if err != nil {
		if isNotFound(err) {
			if err := notFoundOnImport("byoc trunk", req.ID, req.Inputs, req.State); err != nil {
				return resp{}, err
			}
			return resp{}, nil
		}
		return resp{}, fmt.Errorf("read BYOC trunk %s: %w", req.ID, err)
	}
	in := req.Inputs
	in.refresh(o, in == (ByocTrunkArgs{}))
	st := ByocTrunkState{ByocTrunkArgs: in}
	byocTrunkOutputs(&st, o)
	return resp{ID: req.ID, Inputs: in, State: st}, nil
}

func (r *ByocTrunk) Update(ctx context.Context, req infer.UpdateRequest[ByocTrunkArgs, ByocTrunkState]) (infer.UpdateResponse[ByocTrunkState], error) {
	a := req.Inputs
	st := req.State
	st.ByocTrunkArgs = a
	if req.DryRun {
		return infer.UpdateResponse[ByocTrunkState]{Output: st}, nil
	}
	c, err := clientFor(ctx)
	if err != nil {
		return infer.UpdateResponse[ByocTrunkState]{}, err
	}
	prev := req.State.ByocTrunkArgs
	var o obj
	if form := a.form(&prev); len(form) > 0 {
		o, err = c.do(ctx, "POST", voiceProduct, byocTrunkPath(req.ID), nil, form)
	} else {
		o, err = c.do(ctx, "GET", voiceProduct, byocTrunkPath(req.ID), nil, nil)
	}
	if err != nil {
		return infer.UpdateResponse[ByocTrunkState]{}, fmt.Errorf("update BYOC trunk %s: %w", req.ID, err)
	}
	byocTrunkOutputs(&st, o)
	return infer.UpdateResponse[ByocTrunkState]{Output: st}, nil
}

func (r *ByocTrunk) Delete(ctx context.Context, req infer.DeleteRequest[ByocTrunkState]) (infer.DeleteResponse, error) {
	c, err := clientFor(ctx)
	if err != nil {
		return infer.DeleteResponse{}, err
	}
	if _, err := c.do(ctx, "DELETE", voiceProduct, byocTrunkPath(req.ID), nil, nil); err != nil && !isNotFound(err) {
		return infer.DeleteResponse{}, fmt.Errorf("delete BYOC trunk %s: %w", req.ID, err)
	}
	return infer.DeleteResponse{}, nil
}
