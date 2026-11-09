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

// Trunk manages a Twilio Elastic SIP Trunk (trunking.twilio.com): the SIP connection between a
// SIP infrastructure (a PBX or SBC) and Twilio. Trunks belong to the account of the provider
// credentials; Elastic SIP Trunking has no subaccount path.
type Trunk struct{}

// TrunkArgs are the inputs of a Trunk. Every input is optional and updates in place.
type TrunkArgs struct {
	FriendlyName           *string `pulumi:"friendlyName,optional"`
	DomainName             *string `pulumi:"domainName,optional"`
	DisasterRecoveryUrl    *string `pulumi:"disasterRecoveryUrl,optional"`
	DisasterRecoveryMethod *string `pulumi:"disasterRecoveryMethod,optional"`
	TransferMode           *string `pulumi:"transferMode,optional"`
	TransferCallerId       *string `pulumi:"transferCallerId,optional"`
	Secure                 *bool   `pulumi:"secure,optional"`
	CnamLookupEnabled      *bool   `pulumi:"cnamLookupEnabled,optional"`
	RecordingMode          *string `pulumi:"recordingMode,optional"`
	RecordingTrim          *string `pulumi:"recordingTrim,optional"`
}

func (a *TrunkArgs) Annotate(an infer.Annotator) {
	an.Describe(&a.FriendlyName, "Human-readable name of the trunk (up to 64 characters). Removing it leaves the name unchanged on Twilio.")
	an.Describe(&a.DomainName, "The termination SIP domain of the trunk, <name>.pstn.twilio.com: the address your SIP infrastructure sends calls to. "+
		"Letters, digits and \"-\" only; it is unique across Twilio. Changing it updates the trunk in place. Removing it leaves the domain unchanged on Twilio.")
	an.Describe(&a.DisasterRecoveryUrl, "URL Twilio requests (with disasterRecoveryMethod) when it cannot reach your origination URLs. Removing it clears the URL.")
	an.Describe(&a.DisasterRecoveryMethod, "HTTP method used to request disasterRecoveryUrl, GET or POST (Twilio default POST).")
	an.Describe(&a.TransferMode, "Call transfer setting: disable-all (Twilio default), enable-all or sip-only. Removing the input leaves the setting unchanged on Twilio.")
	an.Describe(&a.TransferCallerId, "Caller ID of the transfer target: from-transferee (Twilio default) or from-transferor. Removing the input leaves the setting unchanged on Twilio.")
	an.Describe(&a.Secure, "Whether Secure Trunking is enabled: SRTP for media and TLS for signalling on every call through the trunk (default false). Removing the input leaves the setting unchanged on Twilio.")
	an.Describe(&a.CnamLookupEnabled, "Whether Caller ID Name (CNAM) lookup is enabled for inbound calls (default false). Removing the input leaves the setting unchanged on Twilio.")
	an.Describe(&a.RecordingMode, "Call recording mode: do-not-record (Twilio default), record-from-ringing, record-from-answer, record-from-ringing-dual or record-from-answer-dual. "+
		"Twilio changes it on the trunk's Recording sub-resource; the provider sets it there. Removing the input leaves the setting unchanged on Twilio.")
	an.Describe(&a.RecordingTrim, "Whether silence is trimmed from recordings: do-not-trim (Twilio default) or trim-silence. Removing the input leaves the setting unchanged on Twilio.")
}

// TrunkEffective is the configuration Twilio reports for a Trunk, including fields the program did not set.
type TrunkEffective TrunkArgs

// Annotate reuses the input descriptions.
func (e *TrunkEffective) Annotate(a infer.Annotator) { (*TrunkArgs)(e).Annotate(a) }

// TrunkState is the persisted state of a Trunk.
type TrunkState struct {
	TrunkArgs

	// Sid is the trunk SID (TK...) and the resource ID.
	Sid string `pulumi:"sid"`
	// AccountSid is the account that owns the trunk.
	AccountSid string `pulumi:"accountSid"`
	// AuthType is the authentication method(s) of the trunk.
	AuthType string `pulumi:"authType"`
	// DateCreated and DateUpdated are RFC 2822 timestamps.
	DateCreated string `pulumi:"dateCreated"`
	DateUpdated string `pulumi:"dateUpdated"`
	// Effective is the configuration Twilio reports, including fields the program did not set.
	Effective *TrunkEffective `pulumi:"effective,optional"`
}

func (s *TrunkState) Annotate(a infer.Annotator) {
	a.Describe(&s.Sid, "The trunk SID (TK...).")
	a.Describe(&s.AccountSid, "SID of the account that owns the trunk.")
	a.Describe(&s.AuthType, "The authentication types of the trunk, IP_ACL and CREDENTIAL_LIST (comma-delimited when both). Set by TrunkIpAccessControlList and TrunkCredentialList.")
	a.Describe(&s.DateCreated, "When the trunk was created (RFC 2822).")
	a.Describe(&s.DateUpdated, "When the trunk was last updated (RFC 2822).")
	a.Describe(&s.Effective, "The configuration Twilio currently reports for the trunk, including the domain name Twilio assigned and settings the program did not set.")
}

func (r *Trunk) Annotate(a infer.Annotator) {
	a.Describe(&r, "A Twilio Elastic SIP Trunk. Attach origination URLs with TrunkOriginationUrl, "+
		"credential lists with TrunkCredentialList and IP access control lists with TrunkIpAccessControlList. "+
		"Attach phone numbers with the trunkSid input of IncomingPhoneNumber.")
}

var _ infer.CustomResource[TrunkArgs, TrunkState] = (*Trunk)(nil)
var _ infer.CustomCheck[TrunkArgs] = (*Trunk)(nil)
var _ infer.CustomDiff[TrunkArgs, TrunkState] = (*Trunk)(nil)
var _ infer.CustomRead[TrunkArgs, TrunkState] = (*Trunk)(nil)
var _ infer.CustomUpdate[TrunkArgs, TrunkState] = (*Trunk)(nil)
var _ infer.CustomDelete[TrunkState] = (*Trunk)(nil)

const trunkingProduct = "trunking"

func trunkPath(sid string) string { return "/v1/Trunks/" + url.PathEscape(sid) }

// trunkChildPath is the path of a collection (sid empty) or instance under a trunk, for example OriginationUrls.
func trunkChildPath(trunk, coll, sid string) string {
	path := trunkPath(trunk) + "/" + coll
	if sid != "" {
		path += "/" + url.PathEscape(sid)
	}
	return path
}

var (
	trunkTransferModes = map[string]bool{"disable-all": true, "enable-all": true, "sip-only": true}
	trunkTransferIDs   = map[string]bool{"from-transferee": true, "from-transferor": true}
	trunkRecordModes   = map[string]bool{"do-not-record": true, "record-from-ringing": true, "record-from-answer": true,
		"record-from-ringing-dual": true, "record-from-answer-dual": true}
	trunkRecordTrims = map[string]bool{"do-not-trim": true, "trim-silence": true}
)

func (r *Trunk) Check(ctx context.Context, req infer.CheckRequest) (infer.CheckResponse[TrunkArgs], error) {
	args, failures, err := infer.DefaultCheck[TrunkArgs](ctx, req.NewInputs)
	if err != nil || len(failures) > 0 {
		return infer.CheckResponse[TrunkArgs]{Inputs: args, Failures: failures}, err
	}
	unk := unknowns(req.NewInputs)
	fail := func(prop, reason string) { failures = append(failures, p.CheckFailure{Property: prop, Reason: reason}) }
	if args.FriendlyName != nil && !unk("friendlyName") && (*args.FriendlyName == "" || len(*args.FriendlyName) > 64) {
		fail("friendlyName", "friendlyName must be 1-64 characters")
	}
	if args.DomainName != nil && !unk("domainName") {
		n := strings.ToLower(*args.DomainName)
		if !strings.HasSuffix(n, ".pstn.twilio.com") || len(n) == len(".pstn.twilio.com") {
			fail("domainName", "domainName must be <name>.pstn.twilio.com")
		}
	}
	failures = checkMethods(failures, unk, map[string]*string{"disasterRecoveryMethod": args.DisasterRecoveryMethod})
	for _, e := range []struct {
		name string
		v    *string
		ok   map[string]bool
		list string
	}{
		{"transferMode", args.TransferMode, trunkTransferModes, "disable-all, enable-all or sip-only"},
		{"transferCallerId", args.TransferCallerId, trunkTransferIDs, "from-transferee or from-transferor"},
		{"recordingMode", args.RecordingMode, trunkRecordModes, "do-not-record, record-from-ringing, record-from-answer, record-from-ringing-dual or record-from-answer-dual"},
		{"recordingTrim", args.RecordingTrim, trunkRecordTrims, "do-not-trim or trim-silence"},
	} {
		if e.v != nil && !unk(e.name) && !e.ok[*e.v] {
			fail(e.name, e.name+" must be one of "+e.list)
		}
	}
	return infer.CheckResponse[TrunkArgs]{Inputs: args, Failures: failures}, nil
}

// Diff compares inputs with the recorded inputs, treating an unset input as "leave it alone" and a
// stated Twilio default as equal to nothing recorded, so an imported trunk (which adopts only
// non-default values) and a program that spells out the defaults never disagree.
func (r *Trunk) Diff(_ context.Context, req infer.DiffRequest[TrunkArgs, TrunkState]) (infer.DiffResponse, error) {
	o, n := req.State.TrunkArgs, req.Inputs
	detail := map[string]p.PropertyDiff{}
	str := func(prop string, old, new *string, def string, fold bool) {
		if new == nil {
			return
		}
		a, b := derefOr(old, def), *new
		if a != b && !(fold && strings.EqualFold(a, b)) {
			detail[prop] = p.PropertyDiff{Kind: p.Update}
		}
	}
	flag := func(prop string, old, new *bool) {
		if new != nil && derefBool(old) != *new {
			detail[prop] = p.PropertyDiff{Kind: p.Update}
		}
	}
	str("friendlyName", o.FriendlyName, n.FriendlyName, "", false)
	str("domainName", o.DomainName, n.DomainName, "", true)
	// Removing the recovery URL clears it on Twilio.
	if (n.DisasterRecoveryUrl == nil && deref(o.DisasterRecoveryUrl) != "") || (n.DisasterRecoveryUrl != nil && deref(o.DisasterRecoveryUrl) != *n.DisasterRecoveryUrl) {
		detail["disasterRecoveryUrl"] = p.PropertyDiff{Kind: p.Update}
	}
	str("disasterRecoveryMethod", o.DisasterRecoveryMethod, n.DisasterRecoveryMethod, "POST", false)
	str("transferMode", o.TransferMode, n.TransferMode, "disable-all", false)
	str("transferCallerId", o.TransferCallerId, n.TransferCallerId, "from-transferee", false)
	str("recordingMode", o.RecordingMode, n.RecordingMode, "do-not-record", false)
	str("recordingTrim", o.RecordingTrim, n.RecordingTrim, "do-not-trim", false)
	flag("secure", o.Secure, n.Secure)
	flag("cnamLookupEnabled", o.CnamLookupEnabled, n.CnamLookupEnabled)
	return infer.DiffResponse{HasChanges: len(detail) > 0, DetailedDiff: detail}, nil
}

// trunkForm encodes the trunk settings to send; prev is the recorded inputs on an update.
func (a *TrunkArgs) trunkForm(prev *TrunkArgs) url.Values {
	v := url.Values{}
	setS(v, "FriendlyName", a.FriendlyName)
	setS(v, "DomainName", a.DomainName)
	if a.DisasterRecoveryUrl != nil {
		v.Set("DisasterRecoveryUrl", *a.DisasterRecoveryUrl)
	} else if prev != nil && deref(prev.DisasterRecoveryUrl) != "" {
		v.Set("DisasterRecoveryUrl", "")
	}
	setS(v, "DisasterRecoveryMethod", a.DisasterRecoveryMethod)
	setS(v, "TransferMode", a.TransferMode)
	setS(v, "TransferCallerId", a.TransferCallerId)
	setB(v, "Secure", a.Secure)
	setB(v, "CnamLookupEnabled", a.CnamLookupEnabled)
	return v
}

func (a *TrunkArgs) recordingForm() url.Values {
	v := url.Values{}
	setS(v, "Mode", a.RecordingMode)
	setS(v, "Trim", a.RecordingTrim)
	return v
}

func (a *TrunkArgs) isEmpty() bool { return *a == TrunkArgs{} }

// trunkEffective builds the configuration Twilio reports from the trunk and its Recording sub-resource.
func trunkEffective(o, rec obj) *TrunkEffective {
	e := TrunkArgs{
		FriendlyName: nonEmpty(o.optStr("friendly_name")), DomainName: nonEmpty(o.optStr("domain_name")),
		DisasterRecoveryUrl: nonEmpty(o.optStr("disaster_recovery_url")), DisasterRecoveryMethod: nonEmpty(o.optStr("disaster_recovery_method")),
		TransferMode: nonEmpty(o.optStr("transfer_mode")), TransferCallerId: nonEmpty(o.optStr("transfer_caller_id")),
		Secure: o.optBool("secure"), CnamLookupEnabled: o.optBool("cnam_lookup_enabled"),
		RecordingMode: nonEmpty(rec.optStr("mode")), RecordingTrim: nonEmpty(rec.optStr("trim")),
	}
	v := TrunkEffective(e)
	return &v
}

func trunkOutputs(st *TrunkState, o, rec obj) {
	st.Sid, st.AccountSid, st.AuthType = o.str("sid"), o.str("account_sid"), o.str("auth_type")
	st.DateCreated, st.DateUpdated = o.str("date_created"), o.str("date_updated")
	st.Effective = trunkEffective(o, rec)
}

// refresh copies what Twilio reports into the inputs the program set (drift detection). On an import
// (no inputs at all) it adopts every returned setting that is not a Twilio default, so a program that
// leaves a setting out, or states its default, shows no change.
func (a *TrunkArgs) refresh(e *TrunkEffective, imported bool) {
	strs := []struct {
		dst  **string
		got  *string
		def  string
		fold bool
	}{
		{&a.FriendlyName, e.FriendlyName, "", false}, {&a.DomainName, e.DomainName, "", true},
		{&a.DisasterRecoveryUrl, e.DisasterRecoveryUrl, "", false},
		{&a.DisasterRecoveryMethod, e.DisasterRecoveryMethod, "POST", false},
		{&a.TransferMode, e.TransferMode, "disable-all", false}, {&a.TransferCallerId, e.TransferCallerId, "from-transferee", false},
		{&a.RecordingMode, e.RecordingMode, "do-not-record", false}, {&a.RecordingTrim, e.RecordingTrim, "do-not-trim", false},
	}
	for _, f := range strs {
		switch {
		case f.got == nil:
		case imported:
			if *f.got != f.def {
				*f.dst = f.got
			}
		case *f.dst != nil && !(f.fold && strings.EqualFold(**f.dst, *f.got)):
			*f.dst = f.got
		}
	}
	// A recovery method only matters with a recovery URL.
	if imported && a.DisasterRecoveryUrl == nil {
		a.DisasterRecoveryMethod = nil
	}
	for _, f := range []struct{ dst, got **bool }{{&a.Secure, &e.Secure}, {&a.CnamLookupEnabled, &e.CnamLookupEnabled}} {
		switch {
		case *f.got == nil:
		case imported:
			if **f.got {
				*f.dst = *f.got
			}
		case *f.dst != nil:
			*f.dst = *f.got
		}
	}
}

func (r *Trunk) Create(ctx context.Context, req infer.CreateRequest[TrunkArgs]) (infer.CreateResponse[TrunkState], error) {
	a := req.Inputs
	if req.DryRun {
		return infer.CreateResponse[TrunkState]{Output: TrunkState{TrunkArgs: a}}, nil
	}
	c, err := clientFor(ctx)
	if err != nil {
		return infer.CreateResponse[TrunkState]{}, err
	}
	o, err := c.do(ctx, "POST", trunkingProduct, "/v1/Trunks", nil, a.trunkForm(nil))
	if err != nil {
		return infer.CreateResponse[TrunkState]{}, fmt.Errorf("create trunk: %w", err)
	}
	sid := o.str("sid")
	st := TrunkState{TrunkArgs: a}
	rec := obj{}
	if rf := a.recordingForm(); len(rf) > 0 {
		rec, err = c.do(ctx, "POST", trunkingProduct, trunkChildPath(sid, "Recording", ""), nil, rf)
		if err != nil {
			// The trunk exists but its recording settings are not applied: save it so the next update retries.
			trunkOutputs(&st, o, obj{})
			return infer.CreateResponse[TrunkState]{ID: sid, Output: st},
				infer.ResourceInitFailedError{Reasons: []string{fmt.Sprintf("set recording on trunk %s: %v", sid, err)}}
		}
	} else if rec, err = c.do(ctx, "GET", trunkingProduct, trunkChildPath(sid, "Recording", ""), nil, nil); err != nil {
		rec = obj{}
	}
	trunkOutputs(&st, o, rec)
	return infer.CreateResponse[TrunkState]{ID: sid, Output: st}, nil
}

func (r *Trunk) Read(ctx context.Context, req infer.ReadRequest[TrunkArgs, TrunkState]) (infer.ReadResponse[TrunkArgs, TrunkState], error) {
	c, err := clientFor(ctx)
	if err != nil {
		return infer.ReadResponse[TrunkArgs, TrunkState]{}, err
	}
	o, err := c.do(ctx, "GET", trunkingProduct, trunkPath(req.ID), nil, nil)
	if err != nil {
		if isNotFound(err) {
			if err := notFoundOnImport("trunk", req.ID, req.Inputs, req.State); err != nil {
				return infer.ReadResponse[TrunkArgs, TrunkState]{}, err
			}
			return infer.ReadResponse[TrunkArgs, TrunkState]{}, nil
		}
		return infer.ReadResponse[TrunkArgs, TrunkState]{}, fmt.Errorf("read trunk %s: %w", req.ID, err)
	}
	rec, err := c.do(ctx, "GET", trunkingProduct, trunkChildPath(req.ID, "Recording", ""), nil, nil)
	if err != nil {
		if isNotFound(err) {
			if err := notFoundOnImport("trunk", req.ID, req.Inputs, req.State); err != nil {
				return infer.ReadResponse[TrunkArgs, TrunkState]{}, err
			}
			return infer.ReadResponse[TrunkArgs, TrunkState]{}, nil
		}
		return infer.ReadResponse[TrunkArgs, TrunkState]{}, fmt.Errorf("read trunk %s recording: %w", req.ID, err)
	}
	in := req.Inputs
	st := TrunkState{}
	trunkOutputs(&st, o, rec)
	in.refresh(st.Effective, in.isEmpty())
	st.TrunkArgs = in
	return infer.ReadResponse[TrunkArgs, TrunkState]{ID: req.ID, Inputs: in, State: st}, nil
}

func (r *Trunk) Update(ctx context.Context, req infer.UpdateRequest[TrunkArgs, TrunkState]) (infer.UpdateResponse[TrunkState], error) {
	a := req.Inputs
	st := req.State
	st.TrunkArgs = a
	if req.DryRun {
		return infer.UpdateResponse[TrunkState]{Output: st}, nil
	}
	c, err := clientFor(ctx)
	if err != nil {
		return infer.UpdateResponse[TrunkState]{}, err
	}
	prev := req.State.TrunkArgs
	o := obj{}
	if form := a.trunkForm(&prev); len(form) > 0 {
		if o, err = c.do(ctx, "POST", trunkingProduct, trunkPath(req.ID), nil, form); err != nil {
			return infer.UpdateResponse[TrunkState]{}, fmt.Errorf("update trunk %s: %w", req.ID, err)
		}
	} else if o, err = c.do(ctx, "GET", trunkingProduct, trunkPath(req.ID), nil, nil); err != nil {
		return infer.UpdateResponse[TrunkState]{}, fmt.Errorf("read trunk %s: %w", req.ID, err)
	}
	var rec obj
	if rf := a.recordingForm(); len(rf) > 0 {
		rec, err = c.do(ctx, "POST", trunkingProduct, trunkChildPath(req.ID, "Recording", ""), nil, rf)
	} else {
		rec, err = c.do(ctx, "GET", trunkingProduct, trunkChildPath(req.ID, "Recording", ""), nil, nil)
	}
	if err != nil {
		// Both calls are idempotent, so a failed update is simply repeated by the next one.
		return infer.UpdateResponse[TrunkState]{}, fmt.Errorf("update trunk %s recording: %w", req.ID, err)
	}
	trunkOutputs(&st, o, rec)
	return infer.UpdateResponse[TrunkState]{Output: st}, nil
}

func (r *Trunk) Delete(ctx context.Context, req infer.DeleteRequest[TrunkState]) (infer.DeleteResponse, error) {
	c, err := clientFor(ctx)
	if err != nil {
		return infer.DeleteResponse{}, err
	}
	if _, err := c.do(ctx, "DELETE", trunkingProduct, trunkPath(req.ID), nil, nil); err != nil && !isNotFound(err) {
		return infer.DeleteResponse{}, fmt.Errorf("delete trunk %s: %w", req.ID, err)
	}
	return infer.DeleteResponse{}, nil
}
