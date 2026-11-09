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

// IncomingPhoneNumber adopts a phone number the account already owns and
// manages its voice and SMS routing. It never purchases or releases a number.
type IncomingPhoneNumber struct{}

// IncomingPhoneNumberArgs are the inputs of an IncomingPhoneNumber.
type IncomingPhoneNumberArgs struct {
	PhoneNumber          *string `pulumi:"phoneNumber,optional" provider:"replaceOnChanges"`
	PhoneNumberSid       *string `pulumi:"phoneNumberSid,optional" provider:"replaceOnChanges"`
	AccountSid           *string `pulumi:"accountSid,optional"`
	FriendlyName         *string `pulumi:"friendlyName,optional"`
	VoiceUrl             *string `pulumi:"voiceUrl,optional"`
	VoiceMethod          *string `pulumi:"voiceMethod,optional"`
	VoiceFallbackUrl     *string `pulumi:"voiceFallbackUrl,optional"`
	VoiceFallbackMethod  *string `pulumi:"voiceFallbackMethod,optional"`
	StatusCallback       *string `pulumi:"statusCallback,optional"`
	StatusCallbackMethod *string `pulumi:"statusCallbackMethod,optional"`
	SmsUrl               *string `pulumi:"smsUrl,optional"`
	SmsMethod            *string `pulumi:"smsMethod,optional"`
	SmsFallbackUrl       *string `pulumi:"smsFallbackUrl,optional"`
	SmsFallbackMethod    *string `pulumi:"smsFallbackMethod,optional"`
	VoiceApplicationSid  *string `pulumi:"voiceApplicationSid,optional"`
	SmsApplicationSid    *string `pulumi:"smsApplicationSid,optional"`
	VoiceCallerIdLookup  *bool   `pulumi:"voiceCallerIdLookup,optional"`

	TrunkSid            *string `pulumi:"trunkSid,optional"`
	VoiceReceiveMode    *string `pulumi:"voiceReceiveMode,optional"`
	AddressSid          *string `pulumi:"addressSid,optional"`
	IdentitySid         *string `pulumi:"identitySid,optional"`
	BundleSid           *string `pulumi:"bundleSid,optional"`
	EmergencyStatus     *string `pulumi:"emergencyStatus,optional"`
	EmergencyAddressSid *string `pulumi:"emergencyAddressSid,optional"`
	ApiVersion          *string `pulumi:"apiVersion,optional"`
}

func (a *IncomingPhoneNumberArgs) props() props {
	return props{
		strs: []strProp{
			{"FriendlyName", "friendly_name", &a.FriendlyName},
			{"VoiceUrl", "voice_url", &a.VoiceUrl}, {"VoiceMethod", "voice_method", &a.VoiceMethod},
			{"VoiceFallbackUrl", "voice_fallback_url", &a.VoiceFallbackUrl},
			{"VoiceFallbackMethod", "voice_fallback_method", &a.VoiceFallbackMethod},
			{"StatusCallback", "status_callback", &a.StatusCallback},
			{"StatusCallbackMethod", "status_callback_method", &a.StatusCallbackMethod},
			{"SmsUrl", "sms_url", &a.SmsUrl}, {"SmsMethod", "sms_method", &a.SmsMethod},
			{"SmsFallbackUrl", "sms_fallback_url", &a.SmsFallbackUrl},
			{"SmsFallbackMethod", "sms_fallback_method", &a.SmsFallbackMethod},
			{"VoiceApplicationSid", "voice_application_sid", &a.VoiceApplicationSid},
			{"SmsApplicationSid", "sms_application_sid", &a.SmsApplicationSid},
			{"TrunkSid", "trunk_sid", &a.TrunkSid}, {"VoiceReceiveMode", "voice_receive_mode", &a.VoiceReceiveMode},
			{"AddressSid", "address_sid", &a.AddressSid}, {"IdentitySid", "identity_sid", &a.IdentitySid},
			{"BundleSid", "bundle_sid", &a.BundleSid}, {"EmergencyStatus", "emergency_status", &a.EmergencyStatus},
			{"EmergencyAddressSid", "emergency_address_sid", &a.EmergencyAddressSid},
			{"ApiVersion", "api_version", &a.ApiVersion},
		},
		bools: []boolProp{{"VoiceCallerIdLookup", "voice_caller_id_lookup", &a.VoiceCallerIdLookup}},
	}
}

func (a *IncomingPhoneNumberArgs) Annotate(an infer.Annotator) {
	an.Describe(&a.PhoneNumber, "The owned number in E.164 format, for example +14155550100. Set exactly one of phoneNumber "+
		"or phoneNumberSid. Changing it replaces the resource (a different number is adopted).")
	an.Describe(&a.PhoneNumberSid, "SID (PN...) of the owned number. Set exactly one of phoneNumber or phoneNumberSid. "+
		"Changing it replaces the resource.")
	an.Describe(&a.AccountSid, "SID of the account or subaccount that owns the number. Defaults to the provider account. "+
		"Changing it transfers the number to that account in place (Twilio's AccountSid update parameter); the "+
		"provider's credentials must own the number or its parent account. To import a number owned by a subaccount "+
		"use the ID <accountSid>/<numberSid>. Removing accountSid later sends the number back to the provider account, "+
		"so protect the resource (the protect resource option) to stop an edit from moving a live number.")
	an.Describe(&a.FriendlyName, "Human-readable label for the number.")
	an.Describe(&a.VoiceUrl, "URL Twilio requests for incoming voice calls. Mutually exclusive with voiceApplicationSid on Twilio's side.")
	an.Describe(&a.VoiceMethod, "HTTP method for voiceUrl: GET or POST.")
	an.Describe(&a.VoiceFallbackUrl, "URL requested if voiceUrl fails.")
	an.Describe(&a.VoiceFallbackMethod, "HTTP method for voiceFallbackUrl: GET or POST.")
	an.Describe(&a.StatusCallback, "URL that receives call status events.")
	an.Describe(&a.StatusCallbackMethod, "HTTP method for statusCallback: GET or POST.")
	an.Describe(&a.SmsUrl, "URL Twilio requests for incoming SMS. Mutually exclusive with smsApplicationSid on Twilio's side.")
	an.Describe(&a.SmsMethod, "HTTP method for smsUrl: GET or POST.")
	an.Describe(&a.SmsFallbackUrl, "URL requested if smsUrl fails.")
	an.Describe(&a.SmsFallbackMethod, "HTTP method for smsFallbackUrl: GET or POST.")
	an.Describe(&a.VoiceApplicationSid, "SID (AP...) of an Application that handles incoming voice calls instead of voiceUrl.")
	an.Describe(&a.SmsApplicationSid, "SID (AP...) of an Application that handles incoming SMS instead of smsUrl.")
	an.Describe(&a.VoiceCallerIdLookup, "Whether to look up the caller's name on incoming calls.")
	an.Describe(&a.TrunkSid, "SID (TK...) of the SIP trunk that handles calls to the number instead of a webhook. Removing it unassigns the trunk.")
	an.Describe(&a.VoiceReceiveMode, "Whether the number receives voice or fax calls: voice or fax. Removing it leaves the mode unchanged.")
	an.Describe(&a.AddressSid, "SID (AD...) of the Address the number is registered to, for regulatory compliance. Removing it unassigns the address.")
	an.Describe(&a.IdentitySid, "SID (RI...) of the regulatory Identity the number is registered to. Removing it unassigns the identity.")
	an.Describe(&a.BundleSid, "SID (BU...) of the regulatory Bundle the number is registered under. Removing it unassigns the bundle.")
	an.Describe(&a.EmergencyStatus, "Whether emergency calling is enabled for the number: Active or Inactive. Removing it leaves the status unchanged.")
	an.Describe(&a.EmergencyAddressSid, "SID (AD...) of the Address used for emergency calls from the number. Removing it unassigns the address.")
	an.Describe(&a.ApiVersion, "Twilio API version used for requests to the number's webhooks, for example 2010-04-01. "+
		"Removing it leaves the version on Twilio unchanged.")
}

// IncomingPhoneNumberState is the persisted state of an IncomingPhoneNumber.
type IncomingPhoneNumberState struct {
	IncomingPhoneNumberArgs

	// Sid is the number SID (PN...) and the resource ID.
	Sid string `pulumi:"sid"`
	// Number is the number in E.164 format.
	Number string `pulumi:"number"`
	// OwnerAccountSid is the account that owns the number.
	OwnerAccountSid string `pulumi:"ownerAccountSid"`
	// Effective is the configuration Twilio reports, including fields the program did not set.
	Effective *IncomingPhoneNumberEffective `pulumi:"effective,optional"`
}

func (s *IncomingPhoneNumberState) Annotate(a infer.Annotator) {
	a.Describe(&s.Sid, "The phone number SID (PN...).")
	a.Describe(&s.Number, "The phone number in E.164 format.")
	a.Describe(&s.OwnerAccountSid, "SID of the account that owns the number.")
	a.Describe(&s.Effective, "The configuration Twilio currently reports for the number, including webhooks changed "+
		"outside Pulumi and fields the program does not set. Inputs the program leaves unset are not overwritten by these values.")
}

func (r *IncomingPhoneNumber) Annotate(a infer.Annotator) {
	a.Describe(&r, "Routing configuration for a Twilio phone number the account already owns. The resource adopts the "+
		"number; it never buys one, and destroying it does NOT release the number (it only stops managing it, leaving the "+
		"current webhooks in place).")
}

var _ infer.CustomResource[IncomingPhoneNumberArgs, IncomingPhoneNumberState] = (*IncomingPhoneNumber)(nil)
var _ infer.CustomCheck[IncomingPhoneNumberArgs] = (*IncomingPhoneNumber)(nil)
var _ infer.CustomRead[IncomingPhoneNumberArgs, IncomingPhoneNumberState] = (*IncomingPhoneNumber)(nil)
var _ infer.CustomUpdate[IncomingPhoneNumberArgs, IncomingPhoneNumberState] = (*IncomingPhoneNumber)(nil)
var _ infer.CustomDelete[IncomingPhoneNumberState] = (*IncomingPhoneNumber)(nil)

func (r *IncomingPhoneNumber) Check(ctx context.Context, req infer.CheckRequest) (infer.CheckResponse[IncomingPhoneNumberArgs], error) {
	args, failures, err := infer.DefaultCheck[IncomingPhoneNumberArgs](ctx, req.NewInputs)
	if err != nil || len(failures) > 0 {
		return infer.CheckResponse[IncomingPhoneNumberArgs]{Inputs: args, Failures: failures}, err
	}
	unk := unknowns(req.NewInputs)
	// A computed identity counts as set: its value is only known at apply time.
	hasNum := unk("phoneNumber") || (args.PhoneNumber != nil && *args.PhoneNumber != "")
	hasSid := unk("phoneNumberSid") || (args.PhoneNumberSid != nil && *args.PhoneNumberSid != "")
	if hasNum == hasSid {
		failures = append(failures, p.CheckFailure{Property: "phoneNumber", Reason: "set exactly one of phoneNumber or phoneNumberSid"})
	}
	if v := args.VoiceReceiveMode; v != nil && !unk("voiceReceiveMode") && *v != "voice" && *v != "fax" {
		failures = append(failures, p.CheckFailure{Property: "voiceReceiveMode", Reason: "voiceReceiveMode must be voice or fax"})
	}
	if v := args.EmergencyStatus; v != nil && !unk("emergencyStatus") && *v != "Active" && *v != "Inactive" {
		failures = append(failures, p.CheckFailure{Property: "emergencyStatus", Reason: "emergencyStatus must be Active or Inactive"})
	}
	failures = checkMethods(failures, unk, map[string]*string{
		"voiceMethod": args.VoiceMethod, "voiceFallbackMethod": args.VoiceFallbackMethod,
		"statusCallbackMethod": args.StatusCallbackMethod, "smsMethod": args.SmsMethod,
		"smsFallbackMethod": args.SmsFallbackMethod,
	})
	return infer.CheckResponse[IncomingPhoneNumberArgs]{Inputs: args, Failures: failures}, nil
}

func numberPath(account, sid string) string {
	if sid == "" {
		return api2010(account) + "/IncomingPhoneNumbers.json"
	}
	return api2010(account) + "/IncomingPhoneNumbers/" + url.PathEscape(sid) + ".json"
}

// findNumber resolves the owned number named by the args.
func findNumber(ctx context.Context, c *Client, acct string, a IncomingPhoneNumberArgs) (obj, error) {
	if sid := deref(a.PhoneNumberSid); sid != "" {
		o, err := c.do(ctx, "GET", "api", numberPath(acct, sid), nil, nil)
		if err != nil {
			if isNotFound(err) {
				return nil, fmt.Errorf("account %s does not own a phone number with SID %s", acct, sid)
			}
			return nil, err
		}
		return o, nil
	}
	return findNumberByE164(ctx, c, acct, deref(a.PhoneNumber))
}

func findNumberByE164(ctx context.Context, c *Client, acct, e164 string) (obj, error) {
	o, err := c.do(ctx, "GET", "api", numberPath(acct, ""), url.Values{"PhoneNumber": {e164}}, nil)
	if err != nil {
		return nil, err
	}
	list, _ := o["incoming_phone_numbers"].([]any)
	for _, it := range list {
		if n, ok := it.(map[string]any); ok && obj(n).str("phone_number") == e164 {
			return obj(n), nil
		}
	}
	return nil, fmt.Errorf("account %s does not own the phone number %s (this resource adopts numbers, it does not purchase them)", acct, e164)
}

// IncomingPhoneNumberEffective is the configuration Twilio reports for an IncomingPhoneNumber: every field of the inputs,
// filled from the API response.
type IncomingPhoneNumberEffective IncomingPhoneNumberArgs

// Annotate reuses the input descriptions.
func (e *IncomingPhoneNumberEffective) Annotate(a infer.Annotator) {
	(*IncomingPhoneNumberArgs)(e).Annotate(a)
}

func numberEffective(o obj) *IncomingPhoneNumberEffective {
	e := IncomingPhoneNumberArgs{}
	e.props().fillAll(o)
	v := IncomingPhoneNumberEffective(e)
	return &v
}

func (r *IncomingPhoneNumber) Create(ctx context.Context, req infer.CreateRequest[IncomingPhoneNumberArgs]) (infer.CreateResponse[IncomingPhoneNumberState], error) {
	a := req.Inputs
	if req.DryRun {
		return infer.CreateResponse[IncomingPhoneNumberState]{ID: "", Output: IncomingPhoneNumberState{IncomingPhoneNumberArgs: a}}, nil
	}
	c, err := clientFor(ctx)
	if err != nil {
		return infer.CreateResponse[IncomingPhoneNumberState]{}, err
	}
	acct := c.acct(a.AccountSid)
	cur, err := findNumber(ctx, c, acct, a)
	if err != nil {
		return infer.CreateResponse[IncomingPhoneNumberState]{}, fmt.Errorf("adopt phone number: %w", err)
	}
	sid := cur.str("sid")
	st := IncomingPhoneNumberState{IncomingPhoneNumberArgs: a, Sid: sid, Number: cur.str("phone_number"),
		OwnerAccountSid: acct, Effective: numberEffective(cur)}
	form := url.Values{}
	a.props().encode(form, nil)
	if len(form) > 0 {
		o, err := c.do(ctx, "POST", "api", numberPath(acct, sid), nil, form)
		if err != nil {
			// The number is adopted but not configured: save it so the next update retries.
			return infer.CreateResponse[IncomingPhoneNumberState]{ID: scopedID(c, acct, sid), Output: st},
				infer.ResourceInitFailedError{Reasons: []string{fmt.Sprintf("configure phone number %s: %v", sid, err)}}
		}
		st.Effective = numberEffective(o)
	}
	return infer.CreateResponse[IncomingPhoneNumberState]{ID: scopedID(c, acct, sid), Output: st}, nil
}

func (r *IncomingPhoneNumber) Read(ctx context.Context, req infer.ReadRequest[IncomingPhoneNumberArgs, IncomingPhoneNumberState]) (infer.ReadResponse[IncomingPhoneNumberArgs, IncomingPhoneNumberState], error) {
	c, err := clientFor(ctx)
	if err != nil {
		return infer.ReadResponse[IncomingPhoneNumberArgs, IncomingPhoneNumberState]{}, err
	}
	acct, sid := resolveScope(c, req.ID, req.State.OwnerAccountSid)
	o, err := c.do(ctx, "GET", "api", numberPath(acct, sid), nil, nil)
	if err != nil {
		if isNotFound(err) {
			if err := notFoundOnImport("incoming phone number", req.ID, req.Inputs, req.State); err != nil {
				return infer.ReadResponse[IncomingPhoneNumberArgs, IncomingPhoneNumberState]{}, err
			}
			return infer.ReadResponse[IncomingPhoneNumberArgs, IncomingPhoneNumberState]{}, nil
		}
		return infer.ReadResponse[IncomingPhoneNumberArgs, IncomingPhoneNumberState]{}, fmt.Errorf("read phone number %s: %w", req.ID, err)
	}
	in := req.Inputs
	imported := in.PhoneNumber == nil && in.PhoneNumberSid == nil
	if imported {
		s := sid
		in.PhoneNumberSid = &s
		if acct != c.accountSid {
			in.AccountSid = &acct
		}
		in.props().adopt(o)
	} else {
		in.props().refreshSet(o)
	}
	st := IncomingPhoneNumberState{IncomingPhoneNumberArgs: in, Sid: sid, Number: o.str("phone_number"),
		OwnerAccountSid: derefOr(nonEmpty(o.optStr("account_sid")), acct), Effective: numberEffective(o)}
	return infer.ReadResponse[IncomingPhoneNumberArgs, IncomingPhoneNumberState]{ID: req.ID, Inputs: in, State: st}, nil
}

func (r *IncomingPhoneNumber) Update(ctx context.Context, req infer.UpdateRequest[IncomingPhoneNumberArgs, IncomingPhoneNumberState]) (infer.UpdateResponse[IncomingPhoneNumberState], error) {
	a := req.Inputs
	st := req.State
	st.IncomingPhoneNumberArgs = a
	if req.DryRun { // a preview needs no credentials
		if a.AccountSid != nil && *a.AccountSid != "" {
			st.OwnerAccountSid = *a.AccountSid
		}
		return infer.UpdateResponse[IncomingPhoneNumberState]{Output: st}, nil
	}
	c, err := clientFor(ctx)
	if err != nil {
		return infer.UpdateResponse[IncomingPhoneNumberState]{}, err
	}
	acct, sid := resolveScope(c, req.ID, req.State.OwnerAccountSid)
	target := c.acct(a.AccountSid)
	form := url.Values{}
	prev := req.State.IncomingPhoneNumberArgs.props()
	a.props().encode(form, &prev)
	if target != acct {
		// Twilio moves a number between accounts with the AccountSid update parameter,
		// posted to the account that currently owns it.
		form.Set("AccountSid", target)
	}
	if len(form) > 0 {
		o, err := c.do(ctx, "POST", "api", numberPath(acct, sid), nil, form)
		if err != nil {
			return infer.UpdateResponse[IncomingPhoneNumberState]{}, fmt.Errorf("update phone number %s: %w", req.ID, err)
		}
		st.Effective = numberEffective(o)
		st.OwnerAccountSid = derefOr(nonEmpty(o.optStr("account_sid")), target)
	}
	return infer.UpdateResponse[IncomingPhoneNumberState]{Output: st}, nil
}

// Delete only stops managing the number: it is never released.
func (r *IncomingPhoneNumber) Delete(ctx context.Context, req infer.DeleteRequest[IncomingPhoneNumberState]) (infer.DeleteResponse, error) {
	return infer.DeleteResponse{}, nil
}
