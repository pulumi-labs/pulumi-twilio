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

	"github.com/pulumi/pulumi-go-provider/infer"
)

// GetConnectionPolicy looks up a BYOC connection policy by SID.
type GetConnectionPolicy struct{}

// GetConnectionPolicyArgs are the lookup arguments.
type GetConnectionPolicyArgs struct {
	ConnectionPolicySid string `pulumi:"connectionPolicySid"`
}

func (a *GetConnectionPolicyArgs) Annotate(an infer.Annotator) {
	an.Describe(&a.ConnectionPolicySid, "SID (NY...) of the connection policy.")
}

// GetConnectionPolicyResult is the looked-up object.
type GetConnectionPolicyResult struct {
	Sid          string `pulumi:"sid"`
	FriendlyName string `pulumi:"friendlyName"`
	AccountSid   string `pulumi:"accountSid"`
	DateCreated  string `pulumi:"dateCreated"`
	DateUpdated  string `pulumi:"dateUpdated"`
}

func (a *GetConnectionPolicyResult) Annotate(an infer.Annotator) {
	an.Describe(&a.Sid, "The connection policy SID (NY...).")
	an.Describe(&a.FriendlyName, "Human-readable name of the policy.")
	an.Describe(&a.AccountSid, "SID of the account that owns the object.")
	an.Describe(&a.DateCreated, "When the object was created (ISO 8601).")
	an.Describe(&a.DateUpdated, "When the object was last updated (ISO 8601).")
}

func (f *GetConnectionPolicy) Annotate(a infer.Annotator) {
	a.Describe(&f, "Look up a BYOC connection policy by SID.")
}

func (f *GetConnectionPolicy) Invoke(ctx context.Context, req infer.FunctionRequest[GetConnectionPolicyArgs]) (infer.FunctionResponse[GetConnectionPolicyResult], error) {
	const fn = "getConnectionPolicy"
	if err := requireArg(fn, "connectionPolicySid", req.Input.ConnectionPolicySid); err != nil {
		return infer.FunctionResponse[GetConnectionPolicyResult]{}, err
	}
	c, err := clientFor(ctx)
	if err != nil {
		return infer.FunctionResponse[GetConnectionPolicyResult]{}, err
	}
	o, err := c.do(ctx, "GET", voiceProduct, connectionPolicyPath(req.Input.ConnectionPolicySid), nil, nil)
	if err != nil {
		return infer.FunctionResponse[GetConnectionPolicyResult]{}, lookupFailure(fn, "connection policy "+req.Input.ConnectionPolicySid, err)
	}
	return infer.FunctionResponse[GetConnectionPolicyResult]{Output: GetConnectionPolicyResult{
		Sid:          o.str("sid"),
		FriendlyName: o.str("friendly_name"),
		AccountSid:   o.str("account_sid"),
		DateCreated:  o.str("date_created"),
		DateUpdated:  o.str("date_updated"),
	}}, nil
}

// GetConnectionPolicyTarget looks up a target of a BYOC connection policy.
type GetConnectionPolicyTarget struct{}

// GetConnectionPolicyTargetArgs are the lookup arguments.
type GetConnectionPolicyTargetArgs struct {
	ConnectionPolicySid string `pulumi:"connectionPolicySid"`
	TargetSid           string `pulumi:"targetSid"`
}

func (a *GetConnectionPolicyTargetArgs) Annotate(an infer.Annotator) {
	an.Describe(&a.ConnectionPolicySid, "SID (NY...) of the connection policy.")
	an.Describe(&a.TargetSid, "SID (NE...) of the target.")
}

// GetConnectionPolicyTargetResult is the looked-up object.
type GetConnectionPolicyTargetResult struct {
	ConnectionPolicySid string `pulumi:"connectionPolicySid"`
	TargetSid           string `pulumi:"targetSid"`
	Target              string `pulumi:"target"`
	FriendlyName        string `pulumi:"friendlyName"`
	Priority            int    `pulumi:"priority"`
	Weight              int    `pulumi:"weight"`
	Enabled             bool   `pulumi:"enabled"`
	AccountSid          string `pulumi:"accountSid"`
	DateCreated         string `pulumi:"dateCreated"`
	DateUpdated         string `pulumi:"dateUpdated"`
}

func (a *GetConnectionPolicyTargetResult) Annotate(an infer.Annotator) {
	an.Describe(&a.ConnectionPolicySid, "SID of the connection policy.")
	an.Describe(&a.TargetSid, "SID of the target.")
	an.Describe(&a.Target, "The SIP address of the carrier.")
	an.Describe(&a.FriendlyName, "Human-readable name of the target.")
	an.Describe(&a.Priority, "Relative importance of the target, 0 to 65535.")
	an.Describe(&a.Weight, "Relative share of the load, 1 to 65535.")
	an.Describe(&a.Enabled, "Whether the target is enabled.")
	an.Describe(&a.AccountSid, "SID of the account that owns the object.")
	an.Describe(&a.DateCreated, "When the object was created (ISO 8601).")
	an.Describe(&a.DateUpdated, "When the object was last updated (ISO 8601).")
}

func (f *GetConnectionPolicyTarget) Annotate(a infer.Annotator) {
	a.Describe(&f, "Look up a target of a BYOC connection policy.")
}

func (f *GetConnectionPolicyTarget) Invoke(ctx context.Context, req infer.FunctionRequest[GetConnectionPolicyTargetArgs]) (infer.FunctionResponse[GetConnectionPolicyTargetResult], error) {
	const fn = "getConnectionPolicyTarget"
	if err := requireArg(fn, "connectionPolicySid", req.Input.ConnectionPolicySid); err != nil {
		return infer.FunctionResponse[GetConnectionPolicyTargetResult]{}, err
	}
	if err := requireArg(fn, "targetSid", req.Input.TargetSid); err != nil {
		return infer.FunctionResponse[GetConnectionPolicyTargetResult]{}, err
	}
	c, err := clientFor(ctx)
	if err != nil {
		return infer.FunctionResponse[GetConnectionPolicyTargetResult]{}, err
	}
	o, err := c.do(ctx, "GET", voiceProduct, policyTargetPath(req.Input.ConnectionPolicySid, req.Input.TargetSid), nil, nil)
	if err != nil {
		return infer.FunctionResponse[GetConnectionPolicyTargetResult]{}, lookupFailure(fn, "target "+req.Input.TargetSid+" of connection policy "+req.Input.ConnectionPolicySid, err)
	}
	return infer.FunctionResponse[GetConnectionPolicyTargetResult]{Output: GetConnectionPolicyTargetResult{
		ConnectionPolicySid: req.Input.ConnectionPolicySid,
		TargetSid:           o.str("sid"),
		Target:              o.str("target"),
		FriendlyName:        o.str("friendly_name"),
		Priority:            intOr(o.optInt("priority"), targetDefaultPriority),
		Weight:              intOr(o.optInt("weight"), targetDefaultWeight),
		Enabled:             o.optBool("enabled") == nil || *o.optBool("enabled"),
		AccountSid:          o.str("account_sid"),
		DateCreated:         o.str("date_created"),
		DateUpdated:         o.str("date_updated"),
	}}, nil
}

// GetByocTrunk looks up a BYOC trunk by SID.
type GetByocTrunk struct{}

// GetByocTrunkArgs are the lookup arguments.
type GetByocTrunkArgs struct {
	ByocTrunkSid string `pulumi:"byocTrunkSid"`
}

func (a *GetByocTrunkArgs) Annotate(an infer.Annotator) {
	an.Describe(&a.ByocTrunkSid, "SID (BY...) of the BYOC trunk.")
}

// GetByocTrunkResult is the looked-up object.
type GetByocTrunkResult struct {
	Sid                  string `pulumi:"sid"`
	FriendlyName         string `pulumi:"friendlyName"`
	VoiceUrl             string `pulumi:"voiceUrl"`
	VoiceMethod          string `pulumi:"voiceMethod"`
	VoiceFallbackUrl     string `pulumi:"voiceFallbackUrl"`
	VoiceFallbackMethod  string `pulumi:"voiceFallbackMethod"`
	StatusCallbackUrl    string `pulumi:"statusCallbackUrl"`
	StatusCallbackMethod string `pulumi:"statusCallbackMethod"`
	CnamLookupEnabled    bool   `pulumi:"cnamLookupEnabled"`
	ConnectionPolicySid  string `pulumi:"connectionPolicySid"`
	FromDomainSid        string `pulumi:"fromDomainSid"`
	AccountSid           string `pulumi:"accountSid"`
	DateCreated          string `pulumi:"dateCreated"`
	DateUpdated          string `pulumi:"dateUpdated"`
}

func (a *GetByocTrunkResult) Annotate(an infer.Annotator) {
	an.Describe(&a.Sid, "The BYOC trunk SID (BY...).")
	an.Describe(&a.FriendlyName, "Human-readable name of the trunk.")
	an.Describe(&a.VoiceUrl, "URL Twilio requests for TwiML on calls from the carrier.")
	an.Describe(&a.VoiceMethod, "HTTP method used to request voiceUrl.")
	an.Describe(&a.VoiceFallbackUrl, "URL Twilio requests when voiceUrl fails.")
	an.Describe(&a.VoiceFallbackMethod, "HTTP method used to request voiceFallbackUrl.")
	an.Describe(&a.StatusCallbackUrl, "URL Twilio posts call status events to.")
	an.Describe(&a.StatusCallbackMethod, "HTTP method used to request statusCallbackUrl.")
	an.Describe(&a.CnamLookupEnabled, "Whether CNAM lookup is enabled for inbound calls.")
	an.Describe(&a.ConnectionPolicySid, "SID (NY...) of the connection policy, empty when none is set.")
	an.Describe(&a.FromDomainSid, "SID (SD...) of the SIP domain used as the From host, empty when none is set.")
	an.Describe(&a.AccountSid, "SID of the account that owns the object.")
	an.Describe(&a.DateCreated, "When the object was created (ISO 8601).")
	an.Describe(&a.DateUpdated, "When the object was last updated (ISO 8601).")
}

func (f *GetByocTrunk) Annotate(a infer.Annotator) {
	a.Describe(&f, "Look up a BYOC trunk by SID.")
}

func (f *GetByocTrunk) Invoke(ctx context.Context, req infer.FunctionRequest[GetByocTrunkArgs]) (infer.FunctionResponse[GetByocTrunkResult], error) {
	const fn = "getByocTrunk"
	if err := requireArg(fn, "byocTrunkSid", req.Input.ByocTrunkSid); err != nil {
		return infer.FunctionResponse[GetByocTrunkResult]{}, err
	}
	c, err := clientFor(ctx)
	if err != nil {
		return infer.FunctionResponse[GetByocTrunkResult]{}, err
	}
	o, err := c.do(ctx, "GET", voiceProduct, byocTrunkPath(req.Input.ByocTrunkSid), nil, nil)
	if err != nil {
		return infer.FunctionResponse[GetByocTrunkResult]{}, lookupFailure(fn, "BYOC trunk "+req.Input.ByocTrunkSid, err)
	}
	return infer.FunctionResponse[GetByocTrunkResult]{Output: GetByocTrunkResult{
		Sid:                  o.str("sid"),
		FriendlyName:         o.str("friendly_name"),
		VoiceUrl:             o.str("voice_url"),
		VoiceMethod:          o.str("voice_method"),
		VoiceFallbackUrl:     o.str("voice_fallback_url"),
		VoiceFallbackMethod:  o.str("voice_fallback_method"),
		StatusCallbackUrl:    o.str("status_callback_url"),
		StatusCallbackMethod: o.str("status_callback_method"),
		CnamLookupEnabled:    o.optBool("cnam_lookup_enabled") != nil && *o.optBool("cnam_lookup_enabled"),
		ConnectionPolicySid:  o.str("connection_policy_sid"),
		FromDomainSid:        o.str("from_domain_sid"),
		AccountSid:           o.str("account_sid"),
		DateCreated:          o.str("date_created"),
		DateUpdated:          o.str("date_updated"),
	}}, nil
}

// GetIpRecord looks up a BYOC IP record by SID.
type GetIpRecord struct{}

// GetIpRecordArgs are the lookup arguments.
type GetIpRecordArgs struct {
	IpRecordSid string `pulumi:"ipRecordSid"`
}

func (a *GetIpRecordArgs) Annotate(an infer.Annotator) {
	an.Describe(&a.IpRecordSid, "SID (IL...) of the IP record.")
}

// GetIpRecordResult is the looked-up object.
type GetIpRecordResult struct {
	Sid              string `pulumi:"sid"`
	IpAddress        string `pulumi:"ipAddress"`
	CidrPrefixLength int    `pulumi:"cidrPrefixLength"`
	FriendlyName     string `pulumi:"friendlyName"`
	AccountSid       string `pulumi:"accountSid"`
	DateCreated      string `pulumi:"dateCreated"`
	DateUpdated      string `pulumi:"dateUpdated"`
}

func (a *GetIpRecordResult) Annotate(an infer.Annotator) {
	an.Describe(&a.Sid, "The IP record SID (IL...).")
	an.Describe(&a.IpAddress, "The IPv4 address of the record.")
	an.Describe(&a.CidrPrefixLength, "CIDR prefix length of the record.")
	an.Describe(&a.FriendlyName, "Human-readable name of the record.")
	an.Describe(&a.AccountSid, "SID of the account that owns the object.")
	an.Describe(&a.DateCreated, "When the object was created (ISO 8601).")
	an.Describe(&a.DateUpdated, "When the object was last updated (ISO 8601).")
}

func (f *GetIpRecord) Annotate(a infer.Annotator) {
	a.Describe(&f, "Look up a BYOC IP record by SID.")
}

func (f *GetIpRecord) Invoke(ctx context.Context, req infer.FunctionRequest[GetIpRecordArgs]) (infer.FunctionResponse[GetIpRecordResult], error) {
	const fn = "getIpRecord"
	if err := requireArg(fn, "ipRecordSid", req.Input.IpRecordSid); err != nil {
		return infer.FunctionResponse[GetIpRecordResult]{}, err
	}
	c, err := clientFor(ctx)
	if err != nil {
		return infer.FunctionResponse[GetIpRecordResult]{}, err
	}
	o, err := c.do(ctx, "GET", voiceProduct, ipRecordPath(req.Input.IpRecordSid), nil, nil)
	if err != nil {
		return infer.FunctionResponse[GetIpRecordResult]{}, lookupFailure(fn, "IP record "+req.Input.IpRecordSid, err)
	}
	return infer.FunctionResponse[GetIpRecordResult]{Output: GetIpRecordResult{
		Sid:              o.str("sid"),
		IpAddress:        o.str("ip_address"),
		CidrPrefixLength: intOr(o.optInt("cidr_prefix_length"), ipRecordDefaultCidr),
		FriendlyName:     o.str("friendly_name"),
		AccountSid:       o.str("account_sid"),
		DateCreated:      o.str("date_created"),
		DateUpdated:      o.str("date_updated"),
	}}, nil
}

// GetSourceIpMapping looks up a BYOC source IP mapping by SID.
type GetSourceIpMapping struct{}

// GetSourceIpMappingArgs are the lookup arguments.
type GetSourceIpMappingArgs struct {
	SourceIpMappingSid string `pulumi:"sourceIpMappingSid"`
}

func (a *GetSourceIpMappingArgs) Annotate(an infer.Annotator) {
	an.Describe(&a.SourceIpMappingSid, "SID (IB...) of the source IP mapping.")
}

// GetSourceIpMappingResult is the looked-up object.
type GetSourceIpMappingResult struct {
	Sid          string `pulumi:"sid"`
	IpRecordSid  string `pulumi:"ipRecordSid"`
	SipDomainSid string `pulumi:"sipDomainSid"`
	DateCreated  string `pulumi:"dateCreated"`
	DateUpdated  string `pulumi:"dateUpdated"`
}

func (a *GetSourceIpMappingResult) Annotate(an infer.Annotator) {
	an.Describe(&a.Sid, "The source IP mapping SID (IB...).")
	an.Describe(&a.IpRecordSid, "SID (IL...) of the IP record.")
	an.Describe(&a.SipDomainSid, "SID (SD...) of the SIP domain.")
	an.Describe(&a.DateCreated, "When the mapping was created (ISO 8601).")
	an.Describe(&a.DateUpdated, "When the mapping was last updated (ISO 8601).")
}

func (f *GetSourceIpMapping) Annotate(a infer.Annotator) {
	a.Describe(&f, "Look up a BYOC source IP mapping by SID.")
}

func (f *GetSourceIpMapping) Invoke(ctx context.Context, req infer.FunctionRequest[GetSourceIpMappingArgs]) (infer.FunctionResponse[GetSourceIpMappingResult], error) {
	const fn = "getSourceIpMapping"
	if err := requireArg(fn, "sourceIpMappingSid", req.Input.SourceIpMappingSid); err != nil {
		return infer.FunctionResponse[GetSourceIpMappingResult]{}, err
	}
	c, err := clientFor(ctx)
	if err != nil {
		return infer.FunctionResponse[GetSourceIpMappingResult]{}, err
	}
	o, err := c.do(ctx, "GET", voiceProduct, sourceIpMappingPath(req.Input.SourceIpMappingSid), nil, nil)
	if err != nil {
		return infer.FunctionResponse[GetSourceIpMappingResult]{}, lookupFailure(fn, "source IP mapping "+req.Input.SourceIpMappingSid, err)
	}
	return infer.FunctionResponse[GetSourceIpMappingResult]{Output: GetSourceIpMappingResult{
		Sid:          o.str("sid"),
		IpRecordSid:  o.str("ip_record_sid"),
		SipDomainSid: o.str("sip_domain_sid"),
		DateCreated:  o.str("date_created"),
		DateUpdated:  o.str("date_updated"),
	}}, nil
}
