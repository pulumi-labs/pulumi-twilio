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

	p "github.com/pulumi/pulumi-go-provider"
	"github.com/pulumi/pulumi-go-provider/infer"
)

// VerifyService manages a Twilio Verify service (one-time-passcode verification configuration).
type VerifyService struct{}

// VerifyServiceArgs are the inputs of a VerifyService.
type VerifyServiceArgs struct {
	FriendlyName             string  `pulumi:"friendlyName"`
	CodeLength               *int    `pulumi:"codeLength,optional"`
	LookupEnabled            *bool   `pulumi:"lookupEnabled,optional"`
	SkipSmsToLandlines       *bool   `pulumi:"skipSmsToLandlines,optional"`
	DtmfInputRequired        *bool   `pulumi:"dtmfInputRequired,optional"`
	TtsName                  *string `pulumi:"ttsName,optional"`
	Psd2Enabled              *bool   `pulumi:"psd2Enabled,optional"`
	DoNotShareWarningEnabled *bool   `pulumi:"doNotShareWarningEnabled,optional"`
	CustomCodeEnabled        *bool   `pulumi:"customCodeEnabled,optional"`

	DefaultTemplateSid             *string `pulumi:"defaultTemplateSid,optional"`
	VerifyEventSubscriptionEnabled *bool   `pulumi:"verifyEventSubscriptionEnabled,optional"`

	// The push, totp, whatsapp and passkeys settings are flat, prefixed fields
	// that map to Twilio's dotted form parameters (Push.ApnCredentialSid) and
	// nested response objects.
	PushApnCredentialSid  *string `pulumi:"pushApnCredentialSid,optional"`
	PushFcmCredentialSid  *string `pulumi:"pushFcmCredentialSid,optional"`
	PushIncludeDate       *bool   `pulumi:"pushIncludeDate,optional"`
	TotpIssuer            *string `pulumi:"totpIssuer,optional"`
	TotpCodeLength        *int    `pulumi:"totpCodeLength,optional"`
	TotpTimeStep          *int    `pulumi:"totpTimeStep,optional"`
	TotpSkew              *int    `pulumi:"totpSkew,optional"`
	WhatsappMsgServiceSid *string `pulumi:"whatsappMsgServiceSid,optional"`
	WhatsappFrom          *string `pulumi:"whatsappFrom,optional"`

	PasskeysRelyingPartyId          *string   `pulumi:"passkeysRelyingPartyId,optional"`
	PasskeysRelyingPartyName        *string   `pulumi:"passkeysRelyingPartyName,optional"`
	PasskeysRelyingPartyOrigins     *[]string `pulumi:"passkeysRelyingPartyOrigins,optional"`
	PasskeysAuthenticatorAttachment *string   `pulumi:"passkeysAuthenticatorAttachment,optional"`
	PasskeysDiscoverableCredentials *string   `pulumi:"passkeysDiscoverableCredentials,optional"`
	PasskeysUserVerification        *string   `pulumi:"passkeysUserVerification,optional"`
}

func (a *VerifyServiceArgs) props() props {
	return props{
		strs: []strProp{
			{"TtsName", "tts_name", &a.TtsName},
			{"DefaultTemplateSid", "default_template_sid", &a.DefaultTemplateSid},
			{"Push.ApnCredentialSid", "push.apn_credential_sid", &a.PushApnCredentialSid},
			{"Push.FcmCredentialSid", "push.fcm_credential_sid", &a.PushFcmCredentialSid},
			{"Totp.Issuer", "totp.issuer", &a.TotpIssuer},
			{"Whatsapp.MsgServiceSid", "whatsapp.msg_service_sid", &a.WhatsappMsgServiceSid},
			{"Whatsapp.From", "whatsapp.from", &a.WhatsappFrom},
			{"Passkeys.RelyingParty.Id", "passkeys.relying_party.id", &a.PasskeysRelyingPartyId},
			{"Passkeys.RelyingParty.Name", "passkeys.relying_party.name", &a.PasskeysRelyingPartyName},
			{"Passkeys.AuthenticatorAttachment", "passkeys.authenticator_attachment", &a.PasskeysAuthenticatorAttachment},
			{"Passkeys.DiscoverableCredentials", "passkeys.discoverable_credentials", &a.PasskeysDiscoverableCredentials},
			{"Passkeys.UserVerification", "passkeys.user_verification", &a.PasskeysUserVerification},
		},
		lists: []listProp{{"Passkeys.RelyingParty.Origins", "passkeys.relying_party.origins", &a.PasskeysRelyingPartyOrigins}},
		bools: []boolProp{
			{"LookupEnabled", "lookup_enabled", &a.LookupEnabled},
			{"SkipSmsToLandlines", "skip_sms_to_landlines", &a.SkipSmsToLandlines},
			{"DtmfInputRequired", "dtmf_input_required", &a.DtmfInputRequired},
			{"Psd2Enabled", "psd2_enabled", &a.Psd2Enabled},
			{"DoNotShareWarningEnabled", "do_not_share_warning_enabled", &a.DoNotShareWarningEnabled},
			{"CustomCodeEnabled", "custom_code_enabled", &a.CustomCodeEnabled},
			{"VerifyEventSubscriptionEnabled", "verify_event_subscription_enabled", &a.VerifyEventSubscriptionEnabled},
			{"Push.IncludeDate", "push.include_date", &a.PushIncludeDate},
		},
		ints: []intProp{
			{"CodeLength", "code_length", &a.CodeLength},
			{"Totp.CodeLength", "totp.code_length", &a.TotpCodeLength},
			{"Totp.TimeStep", "totp.time_step", &a.TotpTimeStep},
			{"Totp.Skew", "totp.skew", &a.TotpSkew},
		},
	}
}

func (a *VerifyServiceArgs) Annotate(an infer.Annotator) {
	an.Describe(&a.FriendlyName, "Name of the service, shown in verification messages (up to 30 characters).")
	an.Describe(&a.CodeLength, "Length of the verification code, 4 to 10 digits (Twilio default 6).")
	an.Describe(&a.LookupEnabled, "Whether to look up the destination phone number before sending.")
	an.Describe(&a.SkipSmsToLandlines, "Whether to skip SMS to landline numbers.")
	an.Describe(&a.DtmfInputRequired, "Whether the recipient must press a key to hear a voice code.")
	an.Describe(&a.TtsName, "Name spoken in voice verifications instead of friendlyName.")
	an.Describe(&a.Psd2Enabled, "Whether PSD2 dynamic-linking (payee and amount) is required.")
	an.Describe(&a.DoNotShareWarningEnabled, "Whether to add a do-not-share warning to the message.")
	an.Describe(&a.CustomCodeEnabled, "Whether you may supply your own verification code (requires Twilio approval).")
	an.Describe(&a.DefaultTemplateSid, "SID (HJ...) of the default message template used for verifications. Removing it leaves the template unchanged.")
	an.Describe(&a.VerifyEventSubscriptionEnabled, "Whether Verify events are sent to the service's event subscriptions.")
	an.Describe(&a.PushApnCredentialSid, "SID (CR...) of the Apple Push Notification credential used by the push channel. Removing it unassigns the credential.")
	an.Describe(&a.PushFcmCredentialSid, "SID (CR...) of the Firebase Cloud Messaging credential used by the push channel. Removing it unassigns the credential.")
	an.Describe(&a.PushIncludeDate, "Whether push challenges include the date they were created.")
	an.Describe(&a.TotpIssuer, "Issuer shown in authenticator apps for TOTP factors. Twilio sets it to the friendlyName when the service is created and rejects an empty issuer, so removing it leaves the issuer unchanged. An import does not adopt the issuer; set it in the program to change it.")
	an.Describe(&a.TotpCodeLength, "Number of digits of a TOTP code (3 to 8).")
	an.Describe(&a.TotpTimeStep, "Seconds a TOTP code is valid (Twilio's totp.time_step).")
	an.Describe(&a.TotpSkew, "Number of time steps before and after the current one in which a TOTP code is accepted.")
	an.Describe(&a.WhatsappMsgServiceSid, "SID (MG...) of the Messaging Service used to send WhatsApp verifications. Removing it unassigns the service.")
	an.Describe(&a.WhatsappFrom, "WhatsApp sender (E.164 number) verifications are sent from. Removing it clears the sender.")
	an.Describe(&a.PasskeysRelyingPartyId, "Passkeys relying party ID (the domain). Removing it clears the value.")
	an.Describe(&a.PasskeysRelyingPartyName, "Passkeys relying party display name. Removing it clears the value.")
	an.Describe(&a.PasskeysRelyingPartyOrigins, "Origins allowed to use the passkeys relying party. Removing the list clears it.")
	an.Describe(&a.PasskeysAuthenticatorAttachment, "Authenticator attachment for passkeys: platform, cross-platform or any.")
	an.Describe(&a.PasskeysDiscoverableCredentials, "Whether passkey credentials are discoverable: required, preferred or discouraged.")
	an.Describe(&a.PasskeysUserVerification, "User verification for passkeys: required, preferred or discouraged.")
}

// VerifyServiceState is the persisted state of a VerifyService.
type VerifyServiceState struct {
	VerifyServiceArgs

	// Sid is the service SID (VA...) and the resource ID.
	Sid string `pulumi:"sid"`
	// Effective is the configuration Twilio reports, including fields the program did not set.
	Effective *VerifyServiceEffective `pulumi:"effective,optional"`
}

func (s *VerifyServiceState) Annotate(a infer.Annotator) {
	a.Describe(&s.Sid, "The Verify service SID (VA...).")
	a.Describe(&s.Effective, "The configuration Twilio currently reports for the service, including fields the program "+
		"does not set. Inputs the program leaves unset are not overwritten by these values.")
}

func (r *VerifyService) Annotate(a infer.Annotator) {
	a.Describe(&r, "A Twilio Verify service: configuration for sending and checking one-time passcodes.")
}

var _ infer.CustomResource[VerifyServiceArgs, VerifyServiceState] = (*VerifyService)(nil)
var _ infer.CustomCheck[VerifyServiceArgs] = (*VerifyService)(nil)
var _ infer.CustomRead[VerifyServiceArgs, VerifyServiceState] = (*VerifyService)(nil)
var _ infer.CustomUpdate[VerifyServiceArgs, VerifyServiceState] = (*VerifyService)(nil)
var _ infer.CustomDelete[VerifyServiceState] = (*VerifyService)(nil)

func (r *VerifyService) Check(ctx context.Context, req infer.CheckRequest) (infer.CheckResponse[VerifyServiceArgs], error) {
	args, failures, err := infer.DefaultCheck[VerifyServiceArgs](ctx, req.NewInputs)
	if err != nil || len(failures) > 0 {
		return infer.CheckResponse[VerifyServiceArgs]{Inputs: args, Failures: failures}, err
	}
	unk := unknowns(req.NewInputs)
	if !unk("friendlyName") && (args.FriendlyName == "" || len(args.FriendlyName) > 30) {
		failures = append(failures, p.CheckFailure{Property: "friendlyName", Reason: "friendlyName must be 1-30 characters"})
	}
	if v := args.CodeLength; !unk("codeLength") && v != nil && (*v < 4 || *v > 10) {
		failures = append(failures, p.CheckFailure{Property: "codeLength", Reason: "codeLength must be between 4 and 10"})
	}
	if v := args.DefaultTemplateSid; v != nil && !unk("defaultTemplateSid") && !verifyTemplateSid.MatchString(*v) {
		failures = append(failures, p.CheckFailure{Property: "defaultTemplateSid", Reason: "defaultTemplateSid must be a template SID (HJ followed by 32 hex characters)"})
	}
	if v := args.TotpCodeLength; !unk("totpCodeLength") && v != nil && (*v < 3 || *v > 8) {
		failures = append(failures, p.CheckFailure{Property: "totpCodeLength", Reason: "totpCodeLength must be between 3 and 8"})
	}
	return infer.CheckResponse[VerifyServiceArgs]{Inputs: args, Failures: failures}, nil
}

var verifyTemplateSid = regexp.MustCompile(`^HJ[0-9a-fA-F]{32}$`)

func verifyPath(sid string) string {
	if sid == "" {
		return "/v2/Services"
	}
	return "/v2/Services/" + url.PathEscape(sid)
}

// VerifyServiceEffective is the configuration Twilio reports for a VerifyService: every field of the inputs,
// filled from the API response.
type VerifyServiceEffective VerifyServiceArgs

// Annotate reuses the input descriptions.
func (e *VerifyServiceEffective) Annotate(a infer.Annotator) { (*VerifyServiceArgs)(e).Annotate(a) }

func verifyEffective(o obj) *VerifyServiceEffective {
	e := VerifyServiceArgs{FriendlyName: o.str("friendly_name")}
	e.props().fillAll(o)
	v := VerifyServiceEffective(e)
	return &v
}

func (r *VerifyService) Create(ctx context.Context, req infer.CreateRequest[VerifyServiceArgs]) (infer.CreateResponse[VerifyServiceState], error) {
	a := req.Inputs
	if req.DryRun {
		return infer.CreateResponse[VerifyServiceState]{ID: "", Output: VerifyServiceState{VerifyServiceArgs: a}}, nil
	}
	c, err := clientFor(ctx)
	if err != nil {
		return infer.CreateResponse[VerifyServiceState]{}, err
	}
	form := url.Values{"FriendlyName": {a.FriendlyName}}
	a.props().encode(form, nil)
	o, err := c.do(ctx, "POST", "verify", verifyPath(""), nil, form)
	if err != nil {
		return infer.CreateResponse[VerifyServiceState]{}, fmt.Errorf("create verify service: %w", err)
	}
	return infer.CreateResponse[VerifyServiceState]{ID: o.str("sid"),
		Output: VerifyServiceState{VerifyServiceArgs: a, Sid: o.str("sid"), Effective: verifyEffective(o)}}, nil
}

func (r *VerifyService) Read(ctx context.Context, req infer.ReadRequest[VerifyServiceArgs, VerifyServiceState]) (infer.ReadResponse[VerifyServiceArgs, VerifyServiceState], error) {
	c, err := clientFor(ctx)
	if err != nil {
		return infer.ReadResponse[VerifyServiceArgs, VerifyServiceState]{}, err
	}
	o, err := c.do(ctx, "GET", "verify", verifyPath(req.ID), nil, nil)
	if err != nil {
		if isNotFound(err) {
			return infer.ReadResponse[VerifyServiceArgs, VerifyServiceState]{}, nil
		}
		return infer.ReadResponse[VerifyServiceArgs, VerifyServiceState]{}, fmt.Errorf("read verify service %s: %w", req.ID, err)
	}
	in := req.Inputs
	imported := in.FriendlyName == ""
	in.FriendlyName = o.str("friendly_name")
	if imported {
		in.props().adopt(o)
	} else {
		in.props().refreshSet(o)
	}
	return infer.ReadResponse[VerifyServiceArgs, VerifyServiceState]{ID: req.ID, Inputs: in,
		State: VerifyServiceState{VerifyServiceArgs: in, Sid: req.ID, Effective: verifyEffective(o)}}, nil
}

func (r *VerifyService) Update(ctx context.Context, req infer.UpdateRequest[VerifyServiceArgs, VerifyServiceState]) (infer.UpdateResponse[VerifyServiceState], error) {
	a := req.Inputs
	st := req.State
	st.VerifyServiceArgs = a
	if req.DryRun {
		return infer.UpdateResponse[VerifyServiceState]{Output: st}, nil
	}
	c, err := clientFor(ctx)
	if err != nil {
		return infer.UpdateResponse[VerifyServiceState]{}, err
	}
	form := url.Values{"FriendlyName": {a.FriendlyName}}
	prev := req.State.VerifyServiceArgs.props()
	a.props().encode(form, &prev)
	o, err := c.do(ctx, "POST", "verify", verifyPath(req.ID), nil, form)
	if err != nil {
		return infer.UpdateResponse[VerifyServiceState]{}, fmt.Errorf("update verify service %s: %w", req.ID, err)
	}
	st.Effective = verifyEffective(o)
	return infer.UpdateResponse[VerifyServiceState]{Output: st}, nil
}

func (r *VerifyService) Delete(ctx context.Context, req infer.DeleteRequest[VerifyServiceState]) (infer.DeleteResponse, error) {
	c, err := clientFor(ctx)
	if err != nil {
		return infer.DeleteResponse{}, err
	}
	if _, err := c.do(ctx, "DELETE", "verify", verifyPath(req.ID), nil, nil); err != nil && !isNotFound(err) {
		return infer.DeleteResponse{}, fmt.Errorf("delete verify service %s: %w", req.ID, err)
	}
	return infer.DeleteResponse{}, nil
}
