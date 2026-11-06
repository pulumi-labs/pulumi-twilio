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

// ------------------------------------------------------ getSipCredentialList

// GetSipCredentialList looks up a SIP credential list by SID.
type GetSipCredentialList struct{}

// GetSipCredentialListArgs are the lookup arguments.
type GetSipCredentialListArgs struct {
	CredentialListSid string  `pulumi:"credentialListSid"`
	AccountSid        *string `pulumi:"accountSid,optional"`
}

func (a *GetSipCredentialListArgs) Annotate(an infer.Annotator) {
	an.Describe(&a.CredentialListSid, "SID (CL...) of the SIP credential list.")
	an.Describe(&a.AccountSid, "SID of the account that owns the list. Defaults to the provider account.")
}

// GetSipCredentialListResult is the looked-up credential list.
type GetSipCredentialListResult struct {
	Sid             string `pulumi:"sid"`
	OwnerAccountSid string `pulumi:"ownerAccountSid"`
	FriendlyName    string `pulumi:"friendlyName"`
	DateCreated     string `pulumi:"dateCreated"`
	DateUpdated     string `pulumi:"dateUpdated"`
}

func (a *GetSipCredentialListResult) Annotate(an infer.Annotator) {
	an.Describe(&a.Sid, "The credential list SID (CL...).")
	an.Describe(&a.OwnerAccountSid, "SID of the account that owns the list.")
	an.Describe(&a.FriendlyName, "Name of the list.")
	an.Describe(&a.DateCreated, "When the list was created (RFC 2822).")
	an.Describe(&a.DateUpdated, "When the list was last updated (RFC 2822).")
}

func (f *GetSipCredentialList) Annotate(a infer.Annotator) {
	a.Describe(&f, "Look up a Twilio SIP credential list by SID.")
}

func (f *GetSipCredentialList) Invoke(ctx context.Context, req infer.FunctionRequest[GetSipCredentialListArgs]) (infer.FunctionResponse[GetSipCredentialListResult], error) {
	const fn = "getSipCredentialList"
	if err := requireArg(fn, "credentialListSid", req.Input.CredentialListSid); err != nil {
		return infer.FunctionResponse[GetSipCredentialListResult]{}, err
	}
	c, err := clientFor(ctx)
	if err != nil {
		return infer.FunctionResponse[GetSipCredentialListResult]{}, err
	}
	acct := c.acct(req.Input.AccountSid)
	o, err := c.do(ctx, "GET", "api", sipPath(acct, "CredentialLists", req.Input.CredentialListSid), nil, nil)
	if err != nil {
		return infer.FunctionResponse[GetSipCredentialListResult]{}, lookupFailure(fn, "SIP credential list "+req.Input.CredentialListSid, err)
	}
	return infer.FunctionResponse[GetSipCredentialListResult]{Output: GetSipCredentialListResult{
		Sid: o.str("sid"), OwnerAccountSid: acct, FriendlyName: o.str("friendly_name"),
		DateCreated: o.str("date_created"), DateUpdated: o.str("date_updated"),
	}}, nil
}

// ------------------------------------------------------------ getSipCredential

// GetSipCredential looks up a credential in a SIP credential list. The password is never returned.
type GetSipCredential struct{}

// GetSipCredentialArgs are the lookup arguments.
type GetSipCredentialArgs struct {
	CredentialListSid string  `pulumi:"credentialListSid"`
	CredentialSid     string  `pulumi:"credentialSid"`
	AccountSid        *string `pulumi:"accountSid,optional"`
}

func (a *GetSipCredentialArgs) Annotate(an infer.Annotator) {
	an.Describe(&a.CredentialListSid, "SID (CL...) of the credential list the credential belongs to.")
	an.Describe(&a.CredentialSid, "SID (CR...) of the credential.")
	an.Describe(&a.AccountSid, "SID of the account that owns the list. Defaults to the provider account.")
}

// GetSipCredentialResult is the looked-up credential.
type GetSipCredentialResult struct {
	Sid               string `pulumi:"sid"`
	OwnerAccountSid   string `pulumi:"ownerAccountSid"`
	CredentialListSid string `pulumi:"credentialListSid"`
	Username          string `pulumi:"username"`
	DateCreated       string `pulumi:"dateCreated"`
	DateUpdated       string `pulumi:"dateUpdated"`
}

func (a *GetSipCredentialResult) Annotate(an infer.Annotator) {
	an.Describe(&a.Sid, "The credential SID (CR...).")
	an.Describe(&a.OwnerAccountSid, "SID of the account that owns the credential.")
	an.Describe(&a.CredentialListSid, "SID of the credential list the credential belongs to.")
	an.Describe(&a.Username, "The username of the credential.")
	an.Describe(&a.DateCreated, "When the credential was created (RFC 2822).")
	an.Describe(&a.DateUpdated, "When the credential was last updated (RFC 2822).")
}

func (f *GetSipCredential) Annotate(a infer.Annotator) {
	a.Describe(&f, "Look up a Twilio SIP credential by list SID and credential SID. The password is never returned.")
}

func (f *GetSipCredential) Invoke(ctx context.Context, req infer.FunctionRequest[GetSipCredentialArgs]) (infer.FunctionResponse[GetSipCredentialResult], error) {
	const fn = "getSipCredential"
	if err := requireArg(fn, "credentialListSid", req.Input.CredentialListSid); err != nil {
		return infer.FunctionResponse[GetSipCredentialResult]{}, err
	}
	if err := requireArg(fn, "credentialSid", req.Input.CredentialSid); err != nil {
		return infer.FunctionResponse[GetSipCredentialResult]{}, err
	}
	c, err := clientFor(ctx)
	if err != nil {
		return infer.FunctionResponse[GetSipCredentialResult]{}, err
	}
	acct := c.acct(req.Input.AccountSid)
	o, err := c.do(ctx, "GET", "api", sipChildPath(acct, "CredentialLists", req.Input.CredentialListSid, "Credentials", req.Input.CredentialSid), nil, nil)
	if err != nil {
		return infer.FunctionResponse[GetSipCredentialResult]{}, lookupFailure(fn, "SIP credential "+req.Input.CredentialSid, err)
	}
	return infer.FunctionResponse[GetSipCredentialResult]{Output: GetSipCredentialResult{
		Sid: o.str("sid"), OwnerAccountSid: acct, CredentialListSid: req.Input.CredentialListSid, Username: o.str("username"),
		DateCreated: o.str("date_created"), DateUpdated: o.str("date_updated"),
	}}, nil
}

// ------------------------------------------------- getSipIpAccessControlList

// GetSipIpAccessControlList looks up a SIP IP access control list by SID.
type GetSipIpAccessControlList struct{}

// GetSipIpAccessControlListArgs are the lookup arguments.
type GetSipIpAccessControlListArgs struct {
	IpAccessControlListSid string  `pulumi:"ipAccessControlListSid"`
	AccountSid             *string `pulumi:"accountSid,optional"`
}

func (a *GetSipIpAccessControlListArgs) Annotate(an infer.Annotator) {
	an.Describe(&a.IpAccessControlListSid, "SID (AL...) of the SIP IP access control list.")
	an.Describe(&a.AccountSid, "SID of the account that owns the list. Defaults to the provider account.")
}

// GetSipIpAccessControlListResult is the looked-up IP access control list.
type GetSipIpAccessControlListResult struct {
	Sid             string `pulumi:"sid"`
	OwnerAccountSid string `pulumi:"ownerAccountSid"`
	FriendlyName    string `pulumi:"friendlyName"`
	DateCreated     string `pulumi:"dateCreated"`
	DateUpdated     string `pulumi:"dateUpdated"`
}

func (a *GetSipIpAccessControlListResult) Annotate(an infer.Annotator) {
	an.Describe(&a.Sid, "The IP access control list SID (AL...).")
	an.Describe(&a.OwnerAccountSid, "SID of the account that owns the list.")
	an.Describe(&a.FriendlyName, "Name of the list.")
	an.Describe(&a.DateCreated, "When the list was created (RFC 2822).")
	an.Describe(&a.DateUpdated, "When the list was last updated (RFC 2822).")
}

func (f *GetSipIpAccessControlList) Annotate(a infer.Annotator) {
	a.Describe(&f, "Look up a Twilio SIP IP access control list by SID.")
}

func (f *GetSipIpAccessControlList) Invoke(ctx context.Context, req infer.FunctionRequest[GetSipIpAccessControlListArgs]) (infer.FunctionResponse[GetSipIpAccessControlListResult], error) {
	const fn = "getSipIpAccessControlList"
	if err := requireArg(fn, "ipAccessControlListSid", req.Input.IpAccessControlListSid); err != nil {
		return infer.FunctionResponse[GetSipIpAccessControlListResult]{}, err
	}
	c, err := clientFor(ctx)
	if err != nil {
		return infer.FunctionResponse[GetSipIpAccessControlListResult]{}, err
	}
	acct := c.acct(req.Input.AccountSid)
	o, err := c.do(ctx, "GET", "api", sipPath(acct, "IpAccessControlLists", req.Input.IpAccessControlListSid), nil, nil)
	if err != nil {
		return infer.FunctionResponse[GetSipIpAccessControlListResult]{}, lookupFailure(fn, "SIP IP access control list "+req.Input.IpAccessControlListSid, err)
	}
	return infer.FunctionResponse[GetSipIpAccessControlListResult]{Output: GetSipIpAccessControlListResult{
		Sid: o.str("sid"), OwnerAccountSid: acct, FriendlyName: o.str("friendly_name"),
		DateCreated: o.str("date_created"), DateUpdated: o.str("date_updated"),
	}}, nil
}

// ------------------------------------------------------------- getSipIpAddress

// GetSipIpAddress looks up an address in a SIP IP access control list.
type GetSipIpAddress struct{}

// GetSipIpAddressArgs are the lookup arguments.
type GetSipIpAddressArgs struct {
	IpAccessControlListSid string  `pulumi:"ipAccessControlListSid"`
	IpAddressSid           string  `pulumi:"ipAddressSid"`
	AccountSid             *string `pulumi:"accountSid,optional"`
}

func (a *GetSipIpAddressArgs) Annotate(an infer.Annotator) {
	an.Describe(&a.IpAccessControlListSid, "SID (AL...) of the IP access control list the address belongs to.")
	an.Describe(&a.IpAddressSid, "SID (IP...) of the address.")
	an.Describe(&a.AccountSid, "SID of the account that owns the list. Defaults to the provider account.")
}

// GetSipIpAddressResult is the looked-up address.
type GetSipIpAddressResult struct {
	Sid                    string `pulumi:"sid"`
	OwnerAccountSid        string `pulumi:"ownerAccountSid"`
	IpAccessControlListSid string `pulumi:"ipAccessControlListSid"`
	FriendlyName           string `pulumi:"friendlyName"`
	IpAddress              string `pulumi:"ipAddress"`
	CidrPrefixLength       int    `pulumi:"cidrPrefixLength"`
	DateCreated            string `pulumi:"dateCreated"`
	DateUpdated            string `pulumi:"dateUpdated"`
}

func (a *GetSipIpAddressResult) Annotate(an infer.Annotator) {
	an.Describe(&a.Sid, "The address SID (IP...).")
	an.Describe(&a.OwnerAccountSid, "SID of the account that owns the address.")
	an.Describe(&a.IpAccessControlListSid, "SID of the IP access control list the address belongs to.")
	an.Describe(&a.FriendlyName, "Name of the address.")
	an.Describe(&a.IpAddress, "The IPv4 address.")
	an.Describe(&a.CidrPrefixLength, "Length of the CIDR prefix applied to the address.")
	an.Describe(&a.DateCreated, "When the address was created (RFC 2822).")
	an.Describe(&a.DateUpdated, "When the address was last updated (RFC 2822).")
}

func (f *GetSipIpAddress) Annotate(a infer.Annotator) {
	a.Describe(&f, "Look up a Twilio SIP IP address by access control list SID and address SID.")
}

func (f *GetSipIpAddress) Invoke(ctx context.Context, req infer.FunctionRequest[GetSipIpAddressArgs]) (infer.FunctionResponse[GetSipIpAddressResult], error) {
	const fn = "getSipIpAddress"
	if err := requireArg(fn, "ipAccessControlListSid", req.Input.IpAccessControlListSid); err != nil {
		return infer.FunctionResponse[GetSipIpAddressResult]{}, err
	}
	if err := requireArg(fn, "ipAddressSid", req.Input.IpAddressSid); err != nil {
		return infer.FunctionResponse[GetSipIpAddressResult]{}, err
	}
	c, err := clientFor(ctx)
	if err != nil {
		return infer.FunctionResponse[GetSipIpAddressResult]{}, err
	}
	acct := c.acct(req.Input.AccountSid)
	o, err := c.do(ctx, "GET", "api", sipChildPath(acct, "IpAccessControlLists", req.Input.IpAccessControlListSid, "IpAddresses", req.Input.IpAddressSid), nil, nil)
	if err != nil {
		return infer.FunctionResponse[GetSipIpAddressResult]{}, lookupFailure(fn, "SIP IP address "+req.Input.IpAddressSid, err)
	}
	return infer.FunctionResponse[GetSipIpAddressResult]{Output: GetSipIpAddressResult{
		Sid: o.str("sid"), OwnerAccountSid: acct, IpAccessControlListSid: req.Input.IpAccessControlListSid,
		FriendlyName: o.str("friendly_name"), IpAddress: o.str("ip_address"), CidrPrefixLength: intOf(o.optInt("cidr_prefix_length")),
		DateCreated: o.str("date_created"), DateUpdated: o.str("date_updated"),
	}}, nil
}

// ---------------------------------------------------------------- getSipDomain

// GetSipDomain looks up a SIP domain by SID.
type GetSipDomain struct{}

// GetSipDomainArgs are the lookup arguments.
type GetSipDomainArgs struct {
	DomainSid  string  `pulumi:"domainSid"`
	AccountSid *string `pulumi:"accountSid,optional"`
}

func (a *GetSipDomainArgs) Annotate(an infer.Annotator) {
	an.Describe(&a.DomainSid, "SID (SD...) of the SIP domain.")
	an.Describe(&a.AccountSid, "SID of the account that owns the domain. Defaults to the provider account.")
}

// GetSipDomainResult is the looked-up SIP domain.
type GetSipDomainResult struct {
	Sid                       string `pulumi:"sid"`
	OwnerAccountSid           string `pulumi:"ownerAccountSid"`
	DomainName                string `pulumi:"domainName"`
	FriendlyName              string `pulumi:"friendlyName"`
	AuthType                  string `pulumi:"authType"`
	VoiceUrl                  string `pulumi:"voiceUrl"`
	VoiceMethod               string `pulumi:"voiceMethod"`
	VoiceFallbackUrl          string `pulumi:"voiceFallbackUrl"`
	VoiceFallbackMethod       string `pulumi:"voiceFallbackMethod"`
	VoiceStatusCallbackUrl    string `pulumi:"voiceStatusCallbackUrl"`
	VoiceStatusCallbackMethod string `pulumi:"voiceStatusCallbackMethod"`
	SipRegistration           bool   `pulumi:"sipRegistration"`
	EmergencyCallingEnabled   bool   `pulumi:"emergencyCallingEnabled"`
	Secure                    bool   `pulumi:"secure"`
	ByocTrunkSid              string `pulumi:"byocTrunkSid"`
	EmergencyCallerSid        string `pulumi:"emergencyCallerSid"`
	ApiVersion                string `pulumi:"apiVersion"`
	DateCreated               string `pulumi:"dateCreated"`
	DateUpdated               string `pulumi:"dateUpdated"`
}

func (a *GetSipDomainResult) Annotate(an infer.Annotator) {
	an.Describe(&a.Sid, "The domain SID (SD...).")
	an.Describe(&a.OwnerAccountSid, "SID of the account that owns the domain.")
	an.Describe(&a.DomainName, "The SIP domain name, <name>.sip.twilio.com.")
	an.Describe(&a.FriendlyName, "Human-readable name (empty if none).")
	an.Describe(&a.AuthType, "Authentication types mapped to the domain (IP_ACL, CREDENTIAL_LIST; empty if none).")
	an.Describe(&a.VoiceUrl, "URL requested for TwiML when the domain receives a call (empty if none).")
	an.Describe(&a.VoiceMethod, "HTTP method used for voiceUrl.")
	an.Describe(&a.VoiceFallbackUrl, "URL requested when voiceUrl fails (empty if none).")
	an.Describe(&a.VoiceFallbackMethod, "HTTP method used for voiceFallbackUrl.")
	an.Describe(&a.VoiceStatusCallbackUrl, "URL that receives call status parameters (empty if none).")
	an.Describe(&a.VoiceStatusCallbackMethod, "HTTP method used for voiceStatusCallbackUrl.")
	an.Describe(&a.SipRegistration, "Whether SIP endpoints may register with the domain.")
	an.Describe(&a.EmergencyCallingEnabled, "Whether emergency calling is enabled for the domain.")
	an.Describe(&a.Secure, "Whether secure SIP (TLS and SRTP) is enforced.")
	an.Describe(&a.ByocTrunkSid, "SID of the associated BYOC Trunk (empty if none).")
	an.Describe(&a.EmergencyCallerSid, "SID of the emergency callback phone number (empty if none).")
	an.Describe(&a.ApiVersion, "The Twilio API version used to process the domain's calls.")
	an.Describe(&a.DateCreated, "When the domain was created (RFC 2822).")
	an.Describe(&a.DateUpdated, "When the domain was last updated (RFC 2822).")
}

func (f *GetSipDomain) Annotate(a infer.Annotator) {
	a.Describe(&f, "Look up a Twilio SIP domain by SID.")
}

func (f *GetSipDomain) Invoke(ctx context.Context, req infer.FunctionRequest[GetSipDomainArgs]) (infer.FunctionResponse[GetSipDomainResult], error) {
	const fn = "getSipDomain"
	if err := requireArg(fn, "domainSid", req.Input.DomainSid); err != nil {
		return infer.FunctionResponse[GetSipDomainResult]{}, err
	}
	c, err := clientFor(ctx)
	if err != nil {
		return infer.FunctionResponse[GetSipDomainResult]{}, err
	}
	acct := c.acct(req.Input.AccountSid)
	o, err := c.do(ctx, "GET", "api", sipPath(acct, "Domains", req.Input.DomainSid), nil, nil)
	if err != nil {
		return infer.FunctionResponse[GetSipDomainResult]{}, lookupFailure(fn, "SIP domain "+req.Input.DomainSid, err)
	}
	return infer.FunctionResponse[GetSipDomainResult]{Output: GetSipDomainResult{
		Sid: o.str("sid"), OwnerAccountSid: acct, DomainName: o.str("domain_name"), FriendlyName: o.str("friendly_name"),
		AuthType: o.str("auth_type"), VoiceUrl: o.str("voice_url"), VoiceMethod: o.str("voice_method"),
		VoiceFallbackUrl: o.str("voice_fallback_url"), VoiceFallbackMethod: o.str("voice_fallback_method"),
		VoiceStatusCallbackUrl: o.str("voice_status_callback_url"), VoiceStatusCallbackMethod: o.str("voice_status_callback_method"),
		SipRegistration: isTrue(o.optBool("sip_registration")), EmergencyCallingEnabled: isTrue(o.optBool("emergency_calling_enabled")),
		Secure: isTrue(o.optBool("secure")), ByocTrunkSid: o.str("byoc_trunk_sid"), EmergencyCallerSid: o.str("emergency_caller_sid"),
		ApiVersion: o.str("api_version"), DateCreated: o.str("date_created"), DateUpdated: o.str("date_updated"),
	}}, nil
}
