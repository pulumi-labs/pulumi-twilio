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

// SipDomain manages a Twilio programmable SIP domain: a unique <name>.sip.twilio.com address to
// which SIP traffic is routed.
type SipDomain struct{}

// SipDomainArgs are the inputs of a SipDomain.
type SipDomainArgs struct {
	DomainName                string  `pulumi:"domainName"`
	AccountSid                *string `pulumi:"accountSid,optional" provider:"replaceOnChanges"`
	FriendlyName              *string `pulumi:"friendlyName,optional"`
	VoiceUrl                  *string `pulumi:"voiceUrl,optional"`
	VoiceMethod               *string `pulumi:"voiceMethod,optional"`
	VoiceFallbackUrl          *string `pulumi:"voiceFallbackUrl,optional"`
	VoiceFallbackMethod       *string `pulumi:"voiceFallbackMethod,optional"`
	VoiceStatusCallbackUrl    *string `pulumi:"voiceStatusCallbackUrl,optional"`
	VoiceStatusCallbackMethod *string `pulumi:"voiceStatusCallbackMethod,optional"`
	SipRegistration           *bool   `pulumi:"sipRegistration,optional"`
	EmergencyCallingEnabled   *bool   `pulumi:"emergencyCallingEnabled,optional"`
	Secure                    *bool   `pulumi:"secure,optional"`
	ByocTrunkSid              *string `pulumi:"byocTrunkSid,optional"`
	EmergencyCallerSid        *string `pulumi:"emergencyCallerSid,optional"`
}

func (a *SipDomainArgs) props() props {
	return props{
		strs: []strProp{
			{"FriendlyName", "friendly_name", &a.FriendlyName},
			{"VoiceUrl", "voice_url", &a.VoiceUrl}, {"VoiceMethod", "voice_method", &a.VoiceMethod},
			{"VoiceFallbackUrl", "voice_fallback_url", &a.VoiceFallbackUrl}, {"VoiceFallbackMethod", "voice_fallback_method", &a.VoiceFallbackMethod},
			{"VoiceStatusCallbackUrl", "voice_status_callback_url", &a.VoiceStatusCallbackUrl},
			{"VoiceStatusCallbackMethod", "voice_status_callback_method", &a.VoiceStatusCallbackMethod},
			{"ByocTrunkSid", "byoc_trunk_sid", &a.ByocTrunkSid}, {"EmergencyCallerSid", "emergency_caller_sid", &a.EmergencyCallerSid},
		},
		bools: []boolProp{
			{"SipRegistration", "sip_registration", &a.SipRegistration},
			{"EmergencyCallingEnabled", "emergency_calling_enabled", &a.EmergencyCallingEnabled},
			{"Secure", "secure", &a.Secure},
		},
	}
}

func (a *SipDomainArgs) Annotate(an infer.Annotator) {
	an.Describe(&a.DomainName, "The unique SIP address you reserve on Twilio, for example example.sip.twilio.com. It can contain letters, digits and \"-\" "+
		"and must end with sip.twilio.com. Changing it updates the domain in place.")
	an.Describe(&a.AccountSid, "SID of the account or subaccount that owns the domain. Defaults to the provider account. "+
		"Changing it replaces the resource. To import a domain owned by a subaccount use the ID <accountSid>/<sid>.")
	an.Describe(&a.FriendlyName, "Human-readable name of the domain (up to 64 characters). Removing it leaves the name unchanged on Twilio.")
	an.Describe(&a.VoiceUrl, "URL Twilio requests for TwiML when the domain receives a call. Removing it clears the URL.")
	an.Describe(&a.VoiceMethod, "HTTP method used to request voiceUrl, GET or POST (Twilio default POST).")
	an.Describe(&a.VoiceFallbackUrl, "URL Twilio requests when retrieving or executing the TwiML from voiceUrl fails. Removing it clears the URL.")
	an.Describe(&a.VoiceFallbackMethod, "HTTP method used to request voiceFallbackUrl, GET or POST (Twilio default POST).")
	an.Describe(&a.VoiceStatusCallbackUrl, "URL Twilio sends call status parameters (such as call ended) to. Removing it clears the URL.")
	an.Describe(&a.VoiceStatusCallbackMethod, "HTTP method used to request voiceStatusCallbackUrl, GET or POST (Twilio default POST).")
	an.Describe(&a.SipRegistration, "Whether SIP endpoints may register with the domain to receive calls. Removing the input leaves the setting unchanged on Twilio.")
	an.Describe(&a.EmergencyCallingEnabled, "Whether emergency calling is enabled for the domain (needs phone numbers with validated addresses). Removing the input leaves the setting unchanged on Twilio.")
	an.Describe(&a.Secure, "Whether secure SIP is enforced: TLS is required and SRTP is negotiated on all incoming calls. Removing the input leaves the setting unchanged on Twilio.")
	an.Describe(&a.ByocTrunkSid, "SID (BY...) of the BYOC Trunk the domain is associated with. Removing it clears the association.")
	an.Describe(&a.EmergencyCallerSid, "SID (PN...) of the phone number used as the callback for emergency calls. Removing it clears the setting.")
}

// SipDomainState is the persisted state of a SipDomain.
type SipDomainState struct {
	SipDomainArgs

	// Sid is the domain SID (SD...) and the resource ID.
	Sid string `pulumi:"sid"`
	// OwnerAccountSid is the account that owns the domain.
	OwnerAccountSid string `pulumi:"ownerAccountSid"`
	// AuthType lists the authentication types mapped to the domain.
	AuthType string `pulumi:"authType"`
	// ApiVersion is the Twilio API version used to process the domain's calls.
	ApiVersion string `pulumi:"apiVersion"`
	// DateCreated is the RFC 2822 creation time.
	DateCreated string `pulumi:"dateCreated"`
	// DateUpdated is the RFC 2822 time of the last update.
	DateUpdated string `pulumi:"dateUpdated"`
}

func (s *SipDomainState) Annotate(a infer.Annotator) {
	a.Describe(&s.Sid, "The domain SID (SD...).")
	a.Describe(&s.OwnerAccountSid, "SID of the account that owns the domain.")
	a.Describe(&s.AuthType, "The authentication types mapped to the domain, IP_ACL and CREDENTIAL_LIST (comma-delimited when both). "+
		"Empty until a list is mapped to the domain; with none the domain cannot receive traffic. Set by the SipDomainCredentialListMapping and SipDomainIpAccessControlListMapping resources.")
	a.Describe(&s.ApiVersion, "The Twilio API version used to process the domain's calls.")
	a.Describe(&s.DateCreated, "When the domain was created (RFC 2822).")
	a.Describe(&s.DateUpdated, "When the domain was last updated (RFC 2822).")
}

func (r *SipDomain) Annotate(a infer.Annotator) {
	a.Describe(&r, "A Twilio programmable SIP domain. Map credential lists and IP access control lists to the domain (which sets authType) "+
		"with SipDomainCredentialListMapping and SipDomainIpAccessControlListMapping.")
}

var _ infer.CustomResource[SipDomainArgs, SipDomainState] = (*SipDomain)(nil)
var _ infer.CustomCheck[SipDomainArgs] = (*SipDomain)(nil)
var _ infer.CustomRead[SipDomainArgs, SipDomainState] = (*SipDomain)(nil)
var _ infer.CustomUpdate[SipDomainArgs, SipDomainState] = (*SipDomain)(nil)
var _ infer.CustomDelete[SipDomainState] = (*SipDomain)(nil)

func (r *SipDomain) Check(ctx context.Context, req infer.CheckRequest) (infer.CheckResponse[SipDomainArgs], error) {
	args, failures, err := infer.DefaultCheck[SipDomainArgs](ctx, req.NewInputs)
	if err != nil || len(failures) > 0 {
		return infer.CheckResponse[SipDomainArgs]{Inputs: args, Failures: failures}, err
	}
	unk := unknowns(req.NewInputs)
	if !unk("domainName") {
		n := strings.ToLower(args.DomainName)
		if !strings.HasSuffix(n, ".sip.twilio.com") || len(n) == len(".sip.twilio.com") {
			failures = append(failures, p.CheckFailure{Property: "domainName", Reason: "domainName must be <name>.sip.twilio.com"})
		}
	}
	if args.FriendlyName != nil && !unk("friendlyName") && (*args.FriendlyName == "" || len(*args.FriendlyName) > 64) {
		failures = append(failures, p.CheckFailure{Property: "friendlyName", Reason: "friendlyName must be 1-64 characters"})
	}
	for _, m := range []struct {
		name string
		v    *string
	}{{"voiceMethod", args.VoiceMethod}, {"voiceFallbackMethod", args.VoiceFallbackMethod}, {"voiceStatusCallbackMethod", args.VoiceStatusCallbackMethod}} {
		if m.v != nil && !unk(m.name) && *m.v != "GET" && *m.v != "POST" {
			failures = append(failures, p.CheckFailure{Property: m.name, Reason: m.name + " must be GET or POST"})
		}
	}
	return infer.CheckResponse[SipDomainArgs]{Inputs: args, Failures: failures}, nil
}

func sipDomainOutputs(st *SipDomainState, o obj) {
	st.AuthType, st.ApiVersion = o.str("auth_type"), o.str("api_version")
	st.DateCreated, st.DateUpdated = o.str("date_created"), o.str("date_updated")
}

// sameDomain reports whether two domain names are the same address; DNS names are case-insensitive.
func sameDomain(a, b string) bool { return strings.EqualFold(a, b) }

func (r *SipDomain) Create(ctx context.Context, req infer.CreateRequest[SipDomainArgs]) (infer.CreateResponse[SipDomainState], error) {
	a := req.Inputs
	if req.DryRun {
		return infer.CreateResponse[SipDomainState]{ID: "", Output: SipDomainState{SipDomainArgs: a}}, nil
	}
	c, err := clientFor(ctx)
	if err != nil {
		return infer.CreateResponse[SipDomainState]{}, err
	}
	acct := c.acct(a.AccountSid)
	form := url.Values{"DomainName": {a.DomainName}}
	a.props().encode(form, nil)
	o, err := c.do(ctx, "POST", "api", sipPath(acct, "Domains", ""), nil, form)
	if err != nil {
		return infer.CreateResponse[SipDomainState]{}, fmt.Errorf("create SIP domain: %w", err)
	}
	sid := o.str("sid")
	st := SipDomainState{SipDomainArgs: a, Sid: sid, OwnerAccountSid: acct}
	sipDomainOutputs(&st, o)
	return infer.CreateResponse[SipDomainState]{ID: scopedID(c, acct, sid), Output: st}, nil
}

func (r *SipDomain) Read(ctx context.Context, req infer.ReadRequest[SipDomainArgs, SipDomainState]) (infer.ReadResponse[SipDomainArgs, SipDomainState], error) {
	c, err := clientFor(ctx)
	if err != nil {
		return infer.ReadResponse[SipDomainArgs, SipDomainState]{}, err
	}
	acct, sid := resolveScope(c, req.ID, req.State.OwnerAccountSid)
	o, err := c.do(ctx, "GET", "api", sipPath(acct, "Domains", sid), nil, nil)
	if err != nil {
		if isNotFound(err) {
			if err := notFoundOnImport("sip domain", req.ID, req.Inputs, req.State); err != nil {
				return infer.ReadResponse[SipDomainArgs, SipDomainState]{}, err
			}
			return infer.ReadResponse[SipDomainArgs, SipDomainState]{}, nil
		}
		return infer.ReadResponse[SipDomainArgs, SipDomainState]{}, fmt.Errorf("read SIP domain %s: %w", req.ID, err)
	}
	in := req.Inputs
	imported := in.DomainName == ""
	if got := o.str("domain_name"); imported || !sameDomain(in.DomainName, got) {
		in.DomainName = got
	}
	if imported {
		if acct != c.accountSid {
			in.AccountSid = &acct
		}
		in.props().adopt(o)
	} else {
		in.props().refreshSet(o)
	}
	st := SipDomainState{SipDomainArgs: in, Sid: sid, OwnerAccountSid: acct}
	sipDomainOutputs(&st, o)
	return infer.ReadResponse[SipDomainArgs, SipDomainState]{ID: req.ID, Inputs: in, State: st}, nil
}

func (r *SipDomain) Update(ctx context.Context, req infer.UpdateRequest[SipDomainArgs, SipDomainState]) (infer.UpdateResponse[SipDomainState], error) {
	a := req.Inputs
	st := req.State
	st.SipDomainArgs = a
	if req.DryRun {
		return infer.UpdateResponse[SipDomainState]{Output: st}, nil
	}
	c, err := clientFor(ctx)
	if err != nil {
		return infer.UpdateResponse[SipDomainState]{}, err
	}
	acct, sid := resolveScope(c, req.ID, req.State.OwnerAccountSid)
	form := url.Values{"DomainName": {a.DomainName}}
	prev := req.State.SipDomainArgs.props()
	a.props().encode(form, &prev)
	o, err := c.do(ctx, "POST", "api", sipPath(acct, "Domains", sid), nil, form)
	if err != nil {
		return infer.UpdateResponse[SipDomainState]{}, fmt.Errorf("update SIP domain %s: %w", req.ID, err)
	}
	sipDomainOutputs(&st, o)
	return infer.UpdateResponse[SipDomainState]{Output: st}, nil
}

func (r *SipDomain) Delete(ctx context.Context, req infer.DeleteRequest[SipDomainState]) (infer.DeleteResponse, error) {
	c, err := clientFor(ctx)
	if err != nil {
		return infer.DeleteResponse{}, err
	}
	acct, sid := resolveScope(c, req.ID, req.State.OwnerAccountSid)
	if _, err := c.do(ctx, "DELETE", "api", sipPath(acct, "Domains", sid), nil, nil); err != nil && !isNotFound(err) {
		return infer.DeleteResponse{}, fmt.Errorf("delete SIP domain %s: %w", req.ID, err)
	}
	return infer.DeleteResponse{}, nil
}
