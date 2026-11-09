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
	"net"
	"net/url"
	"strconv"
	"strings"

	p "github.com/pulumi/pulumi-go-provider"
	"github.com/pulumi/pulumi-go-provider/infer"
)

const (
	targetDefaultPriority = 10
	targetDefaultWeight   = 10
	ipRecordDefaultCidr   = 32
)

func ipRecordPath(sid string) string {
	path := "/v1/IpRecords"
	if sid != "" {
		path += "/" + url.PathEscape(sid)
	}
	return path
}

func sourceIpMappingPath(sid string) string {
	path := "/v1/SourceIpMappings"
	if sid != "" {
		path += "/" + url.PathEscape(sid)
	}
	return path
}

// policyTargetPath is the path of a collection (sid empty) or instance of a target of a connection policy.
func policyTargetPath(policy, sid string) string {
	path := connectionPolicyPath(policy) + "/Targets"
	if sid != "" {
		path += "/" + url.PathEscape(sid)
	}
	return path
}

// splitPolicyTargetID splits <connectionPolicySid>/<sid>.
func splitPolicyTargetID(id string) (policy, sid string, err error) {
	parts := strings.Split(id, "/")
	if len(parts) != 2 || parts[0] == "" || parts[1] == "" {
		return "", "", fmt.Errorf("invalid ID %q: expected <connectionPolicySid>/<sid>", id)
	}
	return parts[0], parts[1], nil
}

// ------------------------------------------------------ ConnectionPolicyTarget

// ConnectionPolicyTarget manages a carrier SIP target of a ConnectionPolicy: where a BYOC trunk sends
// outbound calls, with a priority and a weight that spread the load.
type ConnectionPolicyTarget struct{}

// ConnectionPolicyTargetArgs are the inputs of a ConnectionPolicyTarget.
type ConnectionPolicyTargetArgs struct {
	ConnectionPolicySid string  `pulumi:"connectionPolicySid" provider:"replaceOnChanges"`
	Target              string  `pulumi:"target"`
	FriendlyName        *string `pulumi:"friendlyName,optional"`
	Priority            *int    `pulumi:"priority,optional"`
	Weight              *int    `pulumi:"weight,optional"`
	Enabled             *bool   `pulumi:"enabled,optional"`
}

func (a *ConnectionPolicyTargetArgs) Annotate(an infer.Annotator) {
	an.Describe(&a.ConnectionPolicySid, "SID (NY...) of the connection policy. Changing it replaces the resource.")
	an.Describe(&a.Target, "The SIP address of the carrier, for example sip:carrier.example.com:5060. It must use the sip: scheme (sips: is not supported).")
	an.Describe(&a.FriendlyName, "Human-readable name of the target. Removing it leaves the name unchanged on Twilio.")
	an.Describe(&a.Priority, "Relative importance of the target, 0 to 65535 (default 10). The lowest number is the most important.")
	an.Describe(&a.Weight, "Relative share of the load a target receives among targets of the same priority, 1 to 65535 (default 10).")
	an.Describe(&a.Enabled, "Whether the target is enabled (default true).")
}

// ConnectionPolicyTargetState is the persisted state of a ConnectionPolicyTarget.
type ConnectionPolicyTargetState struct {
	ConnectionPolicyTargetArgs

	// Sid is the target SID (NE...).
	Sid string `pulumi:"sid"`
	// AccountSid is the account that owns the target.
	AccountSid string `pulumi:"accountSid"`
	// DateCreated and DateUpdated are ISO 8601 timestamps.
	DateCreated string `pulumi:"dateCreated"`
	DateUpdated string `pulumi:"dateUpdated"`
}

func (s *ConnectionPolicyTargetState) Annotate(a infer.Annotator) {
	a.Describe(&s.Sid, "The target SID (NE...).")
	a.Describe(&s.AccountSid, "SID of the account that owns the target.")
	a.Describe(&s.DateCreated, "When the target was created (ISO 8601).")
	a.Describe(&s.DateUpdated, "When the target was last updated (ISO 8601).")
}

func (r *ConnectionPolicyTarget) Annotate(a infer.Annotator) {
	a.Describe(&r, "A carrier target of a BYOC connection policy. The resource ID is <connectionPolicySid>/<sid>. Twilio accepts only a sip: URI.")
}

var _ infer.CustomResource[ConnectionPolicyTargetArgs, ConnectionPolicyTargetState] = (*ConnectionPolicyTarget)(nil)
var _ infer.CustomCheck[ConnectionPolicyTargetArgs] = (*ConnectionPolicyTarget)(nil)
var _ infer.CustomDiff[ConnectionPolicyTargetArgs, ConnectionPolicyTargetState] = (*ConnectionPolicyTarget)(nil)
var _ infer.CustomRead[ConnectionPolicyTargetArgs, ConnectionPolicyTargetState] = (*ConnectionPolicyTarget)(nil)
var _ infer.CustomUpdate[ConnectionPolicyTargetArgs, ConnectionPolicyTargetState] = (*ConnectionPolicyTarget)(nil)
var _ infer.CustomDelete[ConnectionPolicyTargetState] = (*ConnectionPolicyTarget)(nil)

func (a ConnectionPolicyTargetArgs) form() url.Values {
	v := url.Values{"Target": {a.Target}}
	setS(v, "FriendlyName", a.FriendlyName)
	// The defaults are sent explicitly so that removing an input restores Twilio's default.
	v.Set("Priority", strconv.Itoa(intOr(a.Priority, targetDefaultPriority)))
	v.Set("Weight", strconv.Itoa(intOr(a.Weight, targetDefaultWeight)))
	v.Set("Enabled", strconv.FormatBool(a.Enabled == nil || *a.Enabled))
	return v
}

func (r *ConnectionPolicyTarget) Check(ctx context.Context, req infer.CheckRequest) (infer.CheckResponse[ConnectionPolicyTargetArgs], error) {
	args, failures, err := infer.DefaultCheck[ConnectionPolicyTargetArgs](ctx, req.NewInputs)
	if err != nil || len(failures) > 0 {
		return infer.CheckResponse[ConnectionPolicyTargetArgs]{Inputs: args, Failures: failures}, err
	}
	unk := unknowns(req.NewInputs)
	fail := func(prop, reason string) { failures = append(failures, p.CheckFailure{Property: prop, Reason: reason}) }
	if !unk("target") && !strings.HasPrefix(strings.ToLower(args.Target), "sip:") {
		fail("target", "target must start with sip: (sips: is not supported)")
	}
	if args.FriendlyName != nil && !unk("friendlyName") && *args.FriendlyName == "" {
		fail("friendlyName", "friendlyName must not be empty")
	}
	if args.Weight != nil && !unk("weight") && (*args.Weight < 1 || *args.Weight > 65535) {
		fail("weight", "weight must be between 1 and 65535")
	}
	if args.Priority != nil && !unk("priority") && (*args.Priority < 0 || *args.Priority > 65535) {
		fail("priority", "priority must be between 0 and 65535")
	}
	failures = checkSidPrefix(failures, unk, "connectionPolicySid", &args.ConnectionPolicySid, "NY", "a ConnectionPolicy")
	return infer.CheckResponse[ConnectionPolicyTargetArgs]{Inputs: args, Failures: failures}, nil
}

// Diff treats an unset priority, weight or enabled flag as Twilio's default, so an imported target and
// a program that omits or restates a default agree.
func (r *ConnectionPolicyTarget) Diff(_ context.Context, req infer.DiffRequest[ConnectionPolicyTargetArgs, ConnectionPolicyTargetState]) (infer.DiffResponse, error) {
	o, n := req.State.ConnectionPolicyTargetArgs, req.Inputs
	detail := map[string]p.PropertyDiff{}
	if o.ConnectionPolicySid != n.ConnectionPolicySid {
		detail["connectionPolicySid"] = p.PropertyDiff{Kind: p.UpdateReplace}
	}
	if o.Target != n.Target {
		detail["target"] = p.PropertyDiff{Kind: p.Update}
	}
	if n.FriendlyName != nil && deref(o.FriendlyName) != *n.FriendlyName {
		detail["friendlyName"] = p.PropertyDiff{Kind: p.Update}
	}
	if intOr(o.Priority, targetDefaultPriority) != intOr(n.Priority, targetDefaultPriority) {
		detail["priority"] = p.PropertyDiff{Kind: p.Update}
	}
	if intOr(o.Weight, targetDefaultWeight) != intOr(n.Weight, targetDefaultWeight) {
		detail["weight"] = p.PropertyDiff{Kind: p.Update}
	}
	if (o.Enabled == nil || *o.Enabled) != (n.Enabled == nil || *n.Enabled) {
		detail["enabled"] = p.PropertyDiff{Kind: p.Update}
	}
	return infer.DiffResponse{HasChanges: len(detail) > 0, DetailedDiff: detail}, nil
}

func policyTargetOutputs(st *ConnectionPolicyTargetState, o obj) {
	st.Sid, st.AccountSid = o.str("sid"), o.str("account_sid")
	st.DateCreated, st.DateUpdated = o.str("date_created"), o.str("date_updated")
}

func (r *ConnectionPolicyTarget) Create(ctx context.Context, req infer.CreateRequest[ConnectionPolicyTargetArgs]) (infer.CreateResponse[ConnectionPolicyTargetState], error) {
	a := req.Inputs
	if req.DryRun {
		return infer.CreateResponse[ConnectionPolicyTargetState]{Output: ConnectionPolicyTargetState{ConnectionPolicyTargetArgs: a}}, nil
	}
	c, err := clientFor(ctx)
	if err != nil {
		return infer.CreateResponse[ConnectionPolicyTargetState]{}, err
	}
	o, err := c.do(ctx, "POST", voiceProduct, policyTargetPath(a.ConnectionPolicySid, ""), nil, a.form())
	if err != nil {
		return infer.CreateResponse[ConnectionPolicyTargetState]{}, fmt.Errorf("create target on connection policy %s: %w", a.ConnectionPolicySid, err)
	}
	st := ConnectionPolicyTargetState{ConnectionPolicyTargetArgs: a}
	policyTargetOutputs(&st, o)
	return infer.CreateResponse[ConnectionPolicyTargetState]{ID: a.ConnectionPolicySid + "/" + st.Sid, Output: st}, nil
}

func (r *ConnectionPolicyTarget) Read(ctx context.Context, req infer.ReadRequest[ConnectionPolicyTargetArgs, ConnectionPolicyTargetState]) (infer.ReadResponse[ConnectionPolicyTargetArgs, ConnectionPolicyTargetState], error) {
	type resp = infer.ReadResponse[ConnectionPolicyTargetArgs, ConnectionPolicyTargetState]
	policy, sid, err := splitPolicyTargetID(req.ID)
	if err != nil {
		return resp{}, err
	}
	c, err := clientFor(ctx)
	if err != nil {
		return resp{}, err
	}
	o, err := c.do(ctx, "GET", voiceProduct, policyTargetPath(policy, sid), nil, nil)
	if err != nil {
		if isNotFound(err) {
			return resp{}, nil
		}
		return resp{}, fmt.Errorf("read connection policy target %s: %w", req.ID, err)
	}
	in := req.Inputs
	imported := in.ConnectionPolicySid == ""
	if imported {
		in.ConnectionPolicySid = policy
	}
	in.Target = o.str("target")
	if got := o.optStr("friendly_name"); got != nil && (in.FriendlyName != nil || (imported && *got != "")) {
		in.FriendlyName = got
	}
	if got := o.optInt("priority"); got != nil && (in.Priority != nil || (imported && *got != targetDefaultPriority)) {
		in.Priority = got
	}
	if got := o.optInt("weight"); got != nil && (in.Weight != nil || (imported && *got != targetDefaultWeight)) {
		in.Weight = got
	}
	if got := o.optBool("enabled"); got != nil && (in.Enabled != nil || (imported && !*got)) {
		in.Enabled = got
	}
	st := ConnectionPolicyTargetState{ConnectionPolicyTargetArgs: in}
	policyTargetOutputs(&st, o)
	return resp{ID: req.ID, Inputs: in, State: st}, nil
}

func (r *ConnectionPolicyTarget) Update(ctx context.Context, req infer.UpdateRequest[ConnectionPolicyTargetArgs, ConnectionPolicyTargetState]) (infer.UpdateResponse[ConnectionPolicyTargetState], error) {
	a := req.Inputs
	st := req.State
	st.ConnectionPolicyTargetArgs = a
	if req.DryRun {
		return infer.UpdateResponse[ConnectionPolicyTargetState]{Output: st}, nil
	}
	policy, sid, err := splitPolicyTargetID(req.ID)
	if err != nil {
		return infer.UpdateResponse[ConnectionPolicyTargetState]{}, err
	}
	c, err := clientFor(ctx)
	if err != nil {
		return infer.UpdateResponse[ConnectionPolicyTargetState]{}, err
	}
	o, err := c.do(ctx, "POST", voiceProduct, policyTargetPath(policy, sid), nil, a.form())
	if err != nil {
		return infer.UpdateResponse[ConnectionPolicyTargetState]{}, fmt.Errorf("update connection policy target %s: %w", req.ID, err)
	}
	policyTargetOutputs(&st, o)
	return infer.UpdateResponse[ConnectionPolicyTargetState]{Output: st}, nil
}

func (r *ConnectionPolicyTarget) Delete(ctx context.Context, req infer.DeleteRequest[ConnectionPolicyTargetState]) (infer.DeleteResponse, error) {
	policy, sid, err := splitPolicyTargetID(req.ID)
	if err != nil {
		return infer.DeleteResponse{}, err
	}
	c, err := clientFor(ctx)
	if err != nil {
		return infer.DeleteResponse{}, err
	}
	if _, err := c.do(ctx, "DELETE", voiceProduct, policyTargetPath(policy, sid), nil, nil); err != nil && !isNotFound(err) {
		return infer.DeleteResponse{}, fmt.Errorf("delete connection policy target %s: %w", req.ID, err)
	}
	return infer.DeleteResponse{}, nil
}

// -------------------------------------------------------------------- IpRecord

// IpRecord manages a Twilio BYOC IP record: an IPv4 address or CIDR block of your carrier. Map it to
// a SIP domain with SourceIpMapping so Twilio routes the carrier's inbound calls.
type IpRecord struct{}

// IpRecordArgs are the inputs of an IpRecord.
type IpRecordArgs struct {
	IpAddress        string  `pulumi:"ipAddress" provider:"replaceOnChanges"`
	CidrPrefixLength *int    `pulumi:"cidrPrefixLength,optional" provider:"replaceOnChanges"`
	FriendlyName     *string `pulumi:"friendlyName,optional"`
}

func (a *IpRecordArgs) Annotate(an infer.Annotator) {
	an.Describe(&a.IpAddress, "The IPv4 address of the record (IPv6 is not supported). Changing it replaces the resource.")
	an.Describe(&a.CidrPrefixLength, "CIDR prefix length of the record, 1 to 32 (default 32, a single address). Changing it replaces the resource.")
	an.Describe(&a.FriendlyName, "Human-readable name of the record. Removing it leaves the name unchanged on Twilio.")
}

// IpRecordState is the persisted state of an IpRecord.
type IpRecordState struct {
	IpRecordArgs

	// Sid is the IP record SID (IL...) and the resource ID.
	Sid string `pulumi:"sid"`
	// AccountSid is the account that owns the record.
	AccountSid string `pulumi:"accountSid"`
	// DateCreated and DateUpdated are ISO 8601 timestamps.
	DateCreated string `pulumi:"dateCreated"`
	DateUpdated string `pulumi:"dateUpdated"`
}

func (s *IpRecordState) Annotate(a infer.Annotator) {
	a.Describe(&s.Sid, "The IP record SID (IL...).")
	a.Describe(&s.AccountSid, "SID of the account that owns the record.")
	a.Describe(&s.DateCreated, "When the record was created (ISO 8601).")
	a.Describe(&s.DateUpdated, "When the record was last updated (ISO 8601).")
}

func (r *IpRecord) Annotate(a infer.Annotator) {
	a.Describe(&r, "A BYOC IP record: an IPv4 address or CIDR block of your carrier, for use in a SourceIpMapping.")
}

var _ infer.CustomResource[IpRecordArgs, IpRecordState] = (*IpRecord)(nil)
var _ infer.CustomCheck[IpRecordArgs] = (*IpRecord)(nil)
var _ infer.CustomDiff[IpRecordArgs, IpRecordState] = (*IpRecord)(nil)
var _ infer.CustomRead[IpRecordArgs, IpRecordState] = (*IpRecord)(nil)
var _ infer.CustomUpdate[IpRecordArgs, IpRecordState] = (*IpRecord)(nil)
var _ infer.CustomDelete[IpRecordState] = (*IpRecord)(nil)

func (r *IpRecord) Check(ctx context.Context, req infer.CheckRequest) (infer.CheckResponse[IpRecordArgs], error) {
	args, failures, err := infer.DefaultCheck[IpRecordArgs](ctx, req.NewInputs)
	if err != nil || len(failures) > 0 {
		return infer.CheckResponse[IpRecordArgs]{Inputs: args, Failures: failures}, err
	}
	unk := unknowns(req.NewInputs)
	fail := func(prop, reason string) { failures = append(failures, p.CheckFailure{Property: prop, Reason: reason}) }
	if !unk("ipAddress") {
		if ip := net.ParseIP(args.IpAddress); ip == nil || ip.To4() == nil || strings.Contains(args.IpAddress, ":") {
			fail("ipAddress", "ipAddress must be an IPv4 address (IPv6 is not supported)")
		}
	}
	if args.CidrPrefixLength != nil && !unk("cidrPrefixLength") && (*args.CidrPrefixLength < 1 || *args.CidrPrefixLength > 32) {
		fail("cidrPrefixLength", "cidrPrefixLength must be between 1 and 32")
	}
	if args.FriendlyName != nil && !unk("friendlyName") && *args.FriendlyName == "" {
		fail("friendlyName", "friendlyName must not be empty")
	}
	return infer.CheckResponse[IpRecordArgs]{Inputs: args, Failures: failures}, nil
}

// Diff treats an unset prefix length as Twilio's default of 32.
func (r *IpRecord) Diff(_ context.Context, req infer.DiffRequest[IpRecordArgs, IpRecordState]) (infer.DiffResponse, error) {
	o, n := req.State.IpRecordArgs, req.Inputs
	detail := map[string]p.PropertyDiff{}
	if o.IpAddress != n.IpAddress {
		detail["ipAddress"] = p.PropertyDiff{Kind: p.UpdateReplace}
	}
	if intOr(o.CidrPrefixLength, ipRecordDefaultCidr) != intOr(n.CidrPrefixLength, ipRecordDefaultCidr) {
		detail["cidrPrefixLength"] = p.PropertyDiff{Kind: p.UpdateReplace}
	}
	if n.FriendlyName != nil && deref(o.FriendlyName) != *n.FriendlyName {
		detail["friendlyName"] = p.PropertyDiff{Kind: p.Update}
	}
	return infer.DiffResponse{HasChanges: len(detail) > 0, DetailedDiff: detail}, nil
}

func ipRecordOutputs(st *IpRecordState, o obj) {
	st.Sid, st.AccountSid = o.str("sid"), o.str("account_sid")
	st.DateCreated, st.DateUpdated = o.str("date_created"), o.str("date_updated")
}

func (r *IpRecord) Create(ctx context.Context, req infer.CreateRequest[IpRecordArgs]) (infer.CreateResponse[IpRecordState], error) {
	a := req.Inputs
	if req.DryRun {
		return infer.CreateResponse[IpRecordState]{Output: IpRecordState{IpRecordArgs: a}}, nil
	}
	c, err := clientFor(ctx)
	if err != nil {
		return infer.CreateResponse[IpRecordState]{}, err
	}
	v := url.Values{"IpAddress": {a.IpAddress}}
	setS(v, "FriendlyName", a.FriendlyName)
	setI(v, "CidrPrefixLength", a.CidrPrefixLength)
	o, err := c.do(ctx, "POST", voiceProduct, ipRecordPath(""), nil, v)
	if err != nil {
		return infer.CreateResponse[IpRecordState]{}, fmt.Errorf("create IP record: %w", err)
	}
	st := IpRecordState{IpRecordArgs: a}
	ipRecordOutputs(&st, o)
	return infer.CreateResponse[IpRecordState]{ID: st.Sid, Output: st}, nil
}

func (r *IpRecord) Read(ctx context.Context, req infer.ReadRequest[IpRecordArgs, IpRecordState]) (infer.ReadResponse[IpRecordArgs, IpRecordState], error) {
	type resp = infer.ReadResponse[IpRecordArgs, IpRecordState]
	c, err := clientFor(ctx)
	if err != nil {
		return resp{}, err
	}
	o, err := c.do(ctx, "GET", voiceProduct, ipRecordPath(req.ID), nil, nil)
	if err != nil {
		if isNotFound(err) {
			return resp{}, nil
		}
		return resp{}, fmt.Errorf("read IP record %s: %w", req.ID, err)
	}
	in := req.Inputs
	imported := in.IpAddress == ""
	in.IpAddress = o.str("ip_address")
	if got := o.optInt("cidr_prefix_length"); got != nil && (in.CidrPrefixLength != nil || (imported && *got != ipRecordDefaultCidr)) {
		in.CidrPrefixLength = got
	}
	if got := o.optStr("friendly_name"); got != nil && (in.FriendlyName != nil || (imported && *got != "")) {
		in.FriendlyName = got
	}
	st := IpRecordState{IpRecordArgs: in}
	ipRecordOutputs(&st, o)
	return resp{ID: req.ID, Inputs: in, State: st}, nil
}

func (r *IpRecord) Update(ctx context.Context, req infer.UpdateRequest[IpRecordArgs, IpRecordState]) (infer.UpdateResponse[IpRecordState], error) {
	a := req.Inputs
	st := req.State
	st.IpRecordArgs = a
	if req.DryRun {
		return infer.UpdateResponse[IpRecordState]{Output: st}, nil
	}
	c, err := clientFor(ctx)
	if err != nil {
		return infer.UpdateResponse[IpRecordState]{}, err
	}
	// Only the friendly name updates in place; the address and prefix replace.
	v := url.Values{}
	setS(v, "FriendlyName", a.FriendlyName)
	method := "POST"
	if len(v) == 0 {
		method = "GET"
	}
	o, err := c.do(ctx, method, voiceProduct, ipRecordPath(req.ID), nil, v)
	if err != nil {
		return infer.UpdateResponse[IpRecordState]{}, fmt.Errorf("update IP record %s: %w", req.ID, err)
	}
	ipRecordOutputs(&st, o)
	return infer.UpdateResponse[IpRecordState]{Output: st}, nil
}

func (r *IpRecord) Delete(ctx context.Context, req infer.DeleteRequest[IpRecordState]) (infer.DeleteResponse, error) {
	c, err := clientFor(ctx)
	if err != nil {
		return infer.DeleteResponse{}, err
	}
	if _, err := c.do(ctx, "DELETE", voiceProduct, ipRecordPath(req.ID), nil, nil); err != nil && !isNotFound(err) {
		return infer.DeleteResponse{}, fmt.Errorf("delete IP record %s: %w", req.ID, err)
	}
	return infer.DeleteResponse{}, nil
}

// ------------------------------------------------------------ SourceIpMapping

// SourceIpMapping maps an IpRecord to a SipDomain: calls Twilio receives from the record's address
// are routed to the domain's voice configuration.
type SourceIpMapping struct{}

// SourceIpMappingArgs are the inputs of a SourceIpMapping.
type SourceIpMappingArgs struct {
	IpRecordSid  string `pulumi:"ipRecordSid" provider:"replaceOnChanges"`
	SipDomainSid string `pulumi:"sipDomainSid"`
}

func (a *SourceIpMappingArgs) Annotate(an infer.Annotator) {
	an.Describe(&a.IpRecordSid, "SID (IL...) of the IpRecord. Changing it replaces the resource.")
	an.Describe(&a.SipDomainSid, "SID (SD...) of the SipDomain that handles calls from the record. Changing it updates the mapping in place.")
}

// SourceIpMappingState is the persisted state of a SourceIpMapping.
type SourceIpMappingState struct {
	SourceIpMappingArgs

	// Sid is the mapping SID (IB...) and the resource ID.
	Sid string `pulumi:"sid"`
	// DateCreated and DateUpdated are ISO 8601 timestamps.
	DateCreated string `pulumi:"dateCreated"`
	DateUpdated string `pulumi:"dateUpdated"`
}

func (s *SourceIpMappingState) Annotate(a infer.Annotator) {
	a.Describe(&s.Sid, "The source IP mapping SID (IB...).")
	a.Describe(&s.DateCreated, "When the mapping was created (ISO 8601).")
	a.Describe(&s.DateUpdated, "When the mapping was last updated (ISO 8601).")
}

func (r *SourceIpMapping) Annotate(a infer.Annotator) {
	a.Describe(&r, "A BYOC source IP mapping: routes calls arriving from an IpRecord to a SipDomain.")
}

var _ infer.CustomResource[SourceIpMappingArgs, SourceIpMappingState] = (*SourceIpMapping)(nil)
var _ infer.CustomCheck[SourceIpMappingArgs] = (*SourceIpMapping)(nil)
var _ infer.CustomDiff[SourceIpMappingArgs, SourceIpMappingState] = (*SourceIpMapping)(nil)
var _ infer.CustomRead[SourceIpMappingArgs, SourceIpMappingState] = (*SourceIpMapping)(nil)
var _ infer.CustomUpdate[SourceIpMappingArgs, SourceIpMappingState] = (*SourceIpMapping)(nil)
var _ infer.CustomDelete[SourceIpMappingState] = (*SourceIpMapping)(nil)

func (r *SourceIpMapping) Check(ctx context.Context, req infer.CheckRequest) (infer.CheckResponse[SourceIpMappingArgs], error) {
	args, failures, err := infer.DefaultCheck[SourceIpMappingArgs](ctx, req.NewInputs)
	if err != nil || len(failures) > 0 {
		return infer.CheckResponse[SourceIpMappingArgs]{Inputs: args, Failures: failures}, err
	}
	unk := unknowns(req.NewInputs)
	failures = checkSidPrefix(failures, unk, "ipRecordSid", &args.IpRecordSid, "IL", "an IpRecord")
	failures = checkSidPrefix(failures, unk, "sipDomainSid", &args.SipDomainSid, "SD", "a SipDomain")
	return infer.CheckResponse[SourceIpMappingArgs]{Inputs: args, Failures: failures}, nil
}

func (r *SourceIpMapping) Diff(_ context.Context, req infer.DiffRequest[SourceIpMappingArgs, SourceIpMappingState]) (infer.DiffResponse, error) {
	o, n := req.State.SourceIpMappingArgs, req.Inputs
	detail := map[string]p.PropertyDiff{}
	if o.IpRecordSid != n.IpRecordSid {
		detail["ipRecordSid"] = p.PropertyDiff{Kind: p.UpdateReplace}
	}
	if o.SipDomainSid != n.SipDomainSid {
		detail["sipDomainSid"] = p.PropertyDiff{Kind: p.Update}
	}
	return infer.DiffResponse{HasChanges: len(detail) > 0, DetailedDiff: detail}, nil
}

func sourceIpMappingOutputs(st *SourceIpMappingState, o obj) {
	st.Sid, st.DateCreated, st.DateUpdated = o.str("sid"), o.str("date_created"), o.str("date_updated")
}

func (a SourceIpMappingArgs) form() url.Values {
	return url.Values{"IpRecordSid": {a.IpRecordSid}, "SipDomainSid": {a.SipDomainSid}}
}

func (r *SourceIpMapping) Create(ctx context.Context, req infer.CreateRequest[SourceIpMappingArgs]) (infer.CreateResponse[SourceIpMappingState], error) {
	a := req.Inputs
	if req.DryRun {
		return infer.CreateResponse[SourceIpMappingState]{Output: SourceIpMappingState{SourceIpMappingArgs: a}}, nil
	}
	c, err := clientFor(ctx)
	if err != nil {
		return infer.CreateResponse[SourceIpMappingState]{}, err
	}
	o, err := c.do(ctx, "POST", voiceProduct, sourceIpMappingPath(""), nil, a.form())
	if err != nil {
		return infer.CreateResponse[SourceIpMappingState]{}, fmt.Errorf("create source IP mapping: %w", err)
	}
	st := SourceIpMappingState{SourceIpMappingArgs: a}
	sourceIpMappingOutputs(&st, o)
	return infer.CreateResponse[SourceIpMappingState]{ID: st.Sid, Output: st}, nil
}

func (r *SourceIpMapping) Read(ctx context.Context, req infer.ReadRequest[SourceIpMappingArgs, SourceIpMappingState]) (infer.ReadResponse[SourceIpMappingArgs, SourceIpMappingState], error) {
	type resp = infer.ReadResponse[SourceIpMappingArgs, SourceIpMappingState]
	c, err := clientFor(ctx)
	if err != nil {
		return resp{}, err
	}
	o, err := c.do(ctx, "GET", voiceProduct, sourceIpMappingPath(req.ID), nil, nil)
	if err != nil {
		if isNotFound(err) {
			return resp{}, nil
		}
		return resp{}, fmt.Errorf("read source IP mapping %s: %w", req.ID, err)
	}
	in := SourceIpMappingArgs{IpRecordSid: o.str("ip_record_sid"), SipDomainSid: o.str("sip_domain_sid")}
	st := SourceIpMappingState{SourceIpMappingArgs: in}
	sourceIpMappingOutputs(&st, o)
	return resp{ID: req.ID, Inputs: in, State: st}, nil
}

func (r *SourceIpMapping) Update(ctx context.Context, req infer.UpdateRequest[SourceIpMappingArgs, SourceIpMappingState]) (infer.UpdateResponse[SourceIpMappingState], error) {
	a := req.Inputs
	st := req.State
	st.SourceIpMappingArgs = a
	if req.DryRun {
		return infer.UpdateResponse[SourceIpMappingState]{Output: st}, nil
	}
	c, err := clientFor(ctx)
	if err != nil {
		return infer.UpdateResponse[SourceIpMappingState]{}, err
	}
	o, err := c.do(ctx, "POST", voiceProduct, sourceIpMappingPath(req.ID), nil, url.Values{"SipDomainSid": {a.SipDomainSid}})
	if err != nil {
		return infer.UpdateResponse[SourceIpMappingState]{}, fmt.Errorf("update source IP mapping %s: %w", req.ID, err)
	}
	sourceIpMappingOutputs(&st, o)
	return infer.UpdateResponse[SourceIpMappingState]{Output: st}, nil
}

func (r *SourceIpMapping) Delete(ctx context.Context, req infer.DeleteRequest[SourceIpMappingState]) (infer.DeleteResponse, error) {
	c, err := clientFor(ctx)
	if err != nil {
		return infer.DeleteResponse{}, err
	}
	if _, err := c.do(ctx, "DELETE", voiceProduct, sourceIpMappingPath(req.ID), nil, nil); err != nil && !isNotFound(err) {
		return infer.DeleteResponse{}, fmt.Errorf("delete source IP mapping %s: %w", req.ID, err)
	}
	return infer.DeleteResponse{}, nil
}
