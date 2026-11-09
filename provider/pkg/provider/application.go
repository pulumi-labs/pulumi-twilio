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

// Application manages a Twilio (TwiML) application: a named bundle of voice and
// SMS webhooks that phone numbers can point at.
type Application struct{}

// ApplicationArgs are the inputs of an Application.
type ApplicationArgs struct {
	FriendlyName          string  `pulumi:"friendlyName"`
	AccountSid            *string `pulumi:"accountSid,optional" provider:"replaceOnChanges"`
	VoiceUrl              *string `pulumi:"voiceUrl,optional"`
	VoiceMethod           *string `pulumi:"voiceMethod,optional"`
	VoiceFallbackUrl      *string `pulumi:"voiceFallbackUrl,optional"`
	VoiceFallbackMethod   *string `pulumi:"voiceFallbackMethod,optional"`
	StatusCallback        *string `pulumi:"statusCallback,optional"`
	StatusCallbackMethod  *string `pulumi:"statusCallbackMethod,optional"`
	SmsUrl                *string `pulumi:"smsUrl,optional"`
	SmsMethod             *string `pulumi:"smsMethod,optional"`
	SmsFallbackUrl        *string `pulumi:"smsFallbackUrl,optional"`
	SmsFallbackMethod     *string `pulumi:"smsFallbackMethod,optional"`
	SmsStatusCallback     *string `pulumi:"smsStatusCallback,optional"`
	MessageStatusCallback *string `pulumi:"messageStatusCallback,optional"`
	VoiceCallerIdLookup   *bool   `pulumi:"voiceCallerIdLookup,optional"`

	ApiVersion                      *string `pulumi:"apiVersion,optional"`
	PublicApplicationConnectEnabled *bool   `pulumi:"publicApplicationConnectEnabled,optional"`
}

func (a *ApplicationArgs) props() props {
	return props{
		strs: []strProp{
			{"VoiceUrl", "voice_url", &a.VoiceUrl}, {"VoiceMethod", "voice_method", &a.VoiceMethod},
			{"VoiceFallbackUrl", "voice_fallback_url", &a.VoiceFallbackUrl},
			{"VoiceFallbackMethod", "voice_fallback_method", &a.VoiceFallbackMethod},
			{"StatusCallback", "status_callback", &a.StatusCallback},
			{"StatusCallbackMethod", "status_callback_method", &a.StatusCallbackMethod},
			{"SmsUrl", "sms_url", &a.SmsUrl}, {"SmsMethod", "sms_method", &a.SmsMethod},
			{"SmsFallbackUrl", "sms_fallback_url", &a.SmsFallbackUrl},
			{"SmsFallbackMethod", "sms_fallback_method", &a.SmsFallbackMethod},
			{"SmsStatusCallback", "sms_status_callback", &a.SmsStatusCallback},
			{"MessageStatusCallback", "message_status_callback", &a.MessageStatusCallback},
			{"ApiVersion", "api_version", &a.ApiVersion},
		},
		bools: []boolProp{
			{"VoiceCallerIdLookup", "voice_caller_id_lookup", &a.VoiceCallerIdLookup},
			{"PublicApplicationConnectEnabled", "public_application_connect_enabled", &a.PublicApplicationConnectEnabled},
		},
	}
}

func (a *ApplicationArgs) Annotate(an infer.Annotator) {
	an.Describe(&a.FriendlyName, "Human-readable name of the application (up to 64 characters).")
	an.Describe(&a.AccountSid, "SID of the account or subaccount that owns the application. Defaults to the provider "+
		"account; set it to manage an application in a subaccount with the parent's credentials. Changing it replaces the resource.")
	an.Describe(&a.VoiceUrl, "URL Twilio requests for incoming voice calls.")
	an.Describe(&a.VoiceMethod, "HTTP method for voiceUrl: GET or POST (Twilio default POST).")
	an.Describe(&a.VoiceFallbackUrl, "URL requested if voiceUrl fails.")
	an.Describe(&a.VoiceFallbackMethod, "HTTP method for voiceFallbackUrl: GET or POST.")
	an.Describe(&a.StatusCallback, "URL that receives call status events.")
	an.Describe(&a.StatusCallbackMethod, "HTTP method for statusCallback: GET or POST.")
	an.Describe(&a.SmsUrl, "URL Twilio requests for incoming SMS.")
	an.Describe(&a.SmsMethod, "HTTP method for smsUrl: GET or POST.")
	an.Describe(&a.SmsFallbackUrl, "URL requested if smsUrl fails.")
	an.Describe(&a.SmsFallbackMethod, "HTTP method for smsFallbackUrl: GET or POST.")
	an.Describe(&a.SmsStatusCallback, "URL that receives status events for outbound SMS sent with this application.")
	an.Describe(&a.MessageStatusCallback, "URL that receives status events for messages sent by this application.")
	an.Describe(&a.VoiceCallerIdLookup, "Whether to look up the caller's name on incoming calls.")
	an.Describe(&a.ApiVersion, "Twilio API version used for requests to the application's webhooks, for example 2010-04-01. "+
		"Removing it leaves the version on Twilio unchanged.")
	an.Describe(&a.PublicApplicationConnectEnabled, "Whether the application can be used as a public Twilio Connect application "+
		"(Twilio does not document a default, so removing the input leaves the setting unchanged).")
}

// ApplicationState is the persisted state of an Application.
type ApplicationState struct {
	ApplicationArgs

	// Sid is the application SID (AP...) and the resource ID.
	Sid string `pulumi:"sid"`
	// OwnerAccountSid is the account that owns the application.
	OwnerAccountSid string `pulumi:"ownerAccountSid"`
	// Effective is the configuration Twilio reports, including fields the program did not set.
	Effective *ApplicationEffective `pulumi:"effective,optional"`
}

func (s *ApplicationState) Annotate(a infer.Annotator) {
	a.Describe(&s.Sid, "The application SID (AP...), usable as voiceApplicationSid or smsApplicationSid on a phone number.")
	a.Describe(&s.OwnerAccountSid, "SID of the account that owns the application.")
	a.Describe(&s.Effective, "The configuration Twilio currently reports for the application, including webhooks "+
		"changed outside Pulumi and fields the program does not set. Inputs the program leaves unset are not "+
		"overwritten by these values.")
}

func (r *Application) Annotate(a infer.Annotator) {
	a.Describe(&r, "A Twilio TwiML application: voice and SMS webhooks shared by phone numbers.")
}

var _ infer.CustomResource[ApplicationArgs, ApplicationState] = (*Application)(nil)
var _ infer.CustomCheck[ApplicationArgs] = (*Application)(nil)
var _ infer.CustomRead[ApplicationArgs, ApplicationState] = (*Application)(nil)
var _ infer.CustomUpdate[ApplicationArgs, ApplicationState] = (*Application)(nil)
var _ infer.CustomDelete[ApplicationState] = (*Application)(nil)

var httpMethods = map[string]bool{"GET": true, "POST": true}

func checkMethods(failures []p.CheckFailure, unk func(...string) bool, m map[string]*string) []p.CheckFailure {
	for prop, v := range m {
		if v != nil && !unk(prop) && !httpMethods[*v] {
			failures = append(failures, p.CheckFailure{Property: prop, Reason: prop + " must be GET or POST"})
		}
	}
	return failures
}

func (r *Application) Check(ctx context.Context, req infer.CheckRequest) (infer.CheckResponse[ApplicationArgs], error) {
	args, failures, err := infer.DefaultCheck[ApplicationArgs](ctx, req.NewInputs)
	if err != nil || len(failures) > 0 {
		return infer.CheckResponse[ApplicationArgs]{Inputs: args, Failures: failures}, err
	}
	unk := unknowns(req.NewInputs)
	if !unk("friendlyName") && (args.FriendlyName == "" || len(args.FriendlyName) > 64) {
		failures = append(failures, p.CheckFailure{Property: "friendlyName", Reason: "friendlyName must be 1-64 characters"})
	}
	failures = checkMethods(failures, unk, map[string]*string{
		"voiceMethod": args.VoiceMethod, "voiceFallbackMethod": args.VoiceFallbackMethod,
		"statusCallbackMethod": args.StatusCallbackMethod, "smsMethod": args.SmsMethod,
		"smsFallbackMethod": args.SmsFallbackMethod,
	})
	return infer.CheckResponse[ApplicationArgs]{Inputs: args, Failures: failures}, nil
}

func appPath(account, sid string) string {
	if sid == "" {
		return api2010(account) + "/Applications.json"
	}
	return api2010(account) + "/Applications/" + url.PathEscape(sid) + ".json"
}

// ApplicationEffective is the configuration Twilio reports for an Application: every field of the inputs,
// filled from the API response.
type ApplicationEffective ApplicationArgs

// Annotate reuses the input descriptions.
func (e *ApplicationEffective) Annotate(a infer.Annotator) { (*ApplicationArgs)(e).Annotate(a) }

func applicationEffective(o obj) *ApplicationEffective {
	e := ApplicationArgs{FriendlyName: o.str("friendly_name")}
	e.props().fillAll(o)
	v := ApplicationEffective(e)
	return &v
}

func (r *Application) Create(ctx context.Context, req infer.CreateRequest[ApplicationArgs]) (infer.CreateResponse[ApplicationState], error) {
	a := req.Inputs
	if req.DryRun {
		return infer.CreateResponse[ApplicationState]{ID: "", Output: ApplicationState{ApplicationArgs: a}}, nil
	}
	c, err := clientFor(ctx)
	if err != nil {
		return infer.CreateResponse[ApplicationState]{}, err
	}
	acct := c.acct(a.AccountSid)
	form := url.Values{"FriendlyName": {a.FriendlyName}}
	a.props().encode(form, nil)
	o, err := c.do(ctx, "POST", "api", appPath(acct, ""), nil, form)
	if err != nil {
		return infer.CreateResponse[ApplicationState]{}, fmt.Errorf("create application: %w", err)
	}
	sid := o.str("sid")
	return infer.CreateResponse[ApplicationState]{ID: scopedID(c, acct, sid),
		Output: ApplicationState{ApplicationArgs: a, Sid: sid, OwnerAccountSid: acct, Effective: applicationEffective(o)}}, nil
}

func (r *Application) Read(ctx context.Context, req infer.ReadRequest[ApplicationArgs, ApplicationState]) (infer.ReadResponse[ApplicationArgs, ApplicationState], error) {
	c, err := clientFor(ctx)
	if err != nil {
		return infer.ReadResponse[ApplicationArgs, ApplicationState]{}, err
	}
	acct, sid := resolveScope(c, req.ID, req.State.OwnerAccountSid)
	o, err := c.do(ctx, "GET", "api", appPath(acct, sid), nil, nil)
	if err != nil {
		if isNotFound(err) {
			if err := notFoundOnImport("application", req.ID, req.Inputs, req.State); err != nil {
				return infer.ReadResponse[ApplicationArgs, ApplicationState]{}, err
			}
			return infer.ReadResponse[ApplicationArgs, ApplicationState]{}, nil
		}
		return infer.ReadResponse[ApplicationArgs, ApplicationState]{}, fmt.Errorf("read application %s: %w", req.ID, err)
	}
	in := req.Inputs
	imported := in.FriendlyName == ""
	in.FriendlyName = o.str("friendly_name")
	if imported {
		if acct != c.accountSid {
			in.AccountSid = &acct
		}
		in.props().adopt(o)
	} else {
		in.props().refreshSet(o)
	}
	st := ApplicationState{ApplicationArgs: in, Sid: sid, OwnerAccountSid: acct, Effective: applicationEffective(o)}
	return infer.ReadResponse[ApplicationArgs, ApplicationState]{ID: req.ID, Inputs: in, State: st}, nil
}

func (r *Application) Update(ctx context.Context, req infer.UpdateRequest[ApplicationArgs, ApplicationState]) (infer.UpdateResponse[ApplicationState], error) {
	a := req.Inputs
	st := req.State
	st.ApplicationArgs = a
	if req.DryRun {
		return infer.UpdateResponse[ApplicationState]{Output: st}, nil
	}
	c, err := clientFor(ctx)
	if err != nil {
		return infer.UpdateResponse[ApplicationState]{}, err
	}
	acct, sid := resolveScope(c, req.ID, req.State.OwnerAccountSid)
	form := url.Values{"FriendlyName": {a.FriendlyName}}
	prev := req.State.ApplicationArgs.props()
	a.props().encode(form, &prev)
	o, err := c.do(ctx, "POST", "api", appPath(acct, sid), nil, form)
	if err != nil {
		return infer.UpdateResponse[ApplicationState]{}, fmt.Errorf("update application %s: %w", req.ID, err)
	}
	st.Effective = applicationEffective(o)
	return infer.UpdateResponse[ApplicationState]{Output: st}, nil
}

func (r *Application) Delete(ctx context.Context, req infer.DeleteRequest[ApplicationState]) (infer.DeleteResponse, error) {
	c, err := clientFor(ctx)
	if err != nil {
		return infer.DeleteResponse{}, err
	}
	acct, sid := resolveScope(c, req.ID, req.State.OwnerAccountSid)
	if _, err := c.do(ctx, "DELETE", "api", appPath(acct, sid), nil, nil); err != nil && !isNotFound(err) {
		return infer.DeleteResponse{}, fmt.Errorf("delete application %s: %w", req.ID, err)
	}
	return infer.DeleteResponse{}, nil
}
