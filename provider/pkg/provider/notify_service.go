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

	"github.com/pulumi/pulumi-go-provider/infer"
)

var (
	notifyCredentialSidRE = regexp.MustCompile(`^CR[0-9a-fA-F]{32}$`)
	notifyMessagingSidRE  = regexp.MustCompile(`^MG[0-9a-fA-F]{32}$`)
)

// NotifyService manages a Notify service: the push and SMS settings that bindings and
// notifications are sent through.
type NotifyService struct{}

// NotifyServiceArgs are the inputs of a NotifyService.
type NotifyServiceArgs struct {
	FriendlyName                          *string `pulumi:"friendlyName,optional"`
	ApnCredentialSid                      *string `pulumi:"apnCredentialSid,optional"`
	GcmCredentialSid                      *string `pulumi:"gcmCredentialSid,optional"`
	FcmCredentialSid                      *string `pulumi:"fcmCredentialSid,optional"`
	MessagingServiceSid                   *string `pulumi:"messagingServiceSid,optional"`
	DefaultApnNotificationProtocolVersion *string `pulumi:"defaultApnNotificationProtocolVersion,optional"`
	DefaultGcmNotificationProtocolVersion *string `pulumi:"defaultGcmNotificationProtocolVersion,optional"`
	DefaultFcmNotificationProtocolVersion *string `pulumi:"defaultFcmNotificationProtocolVersion,optional"`
	LogEnabled                            *bool   `pulumi:"logEnabled,optional"`
	DeliveryCallbackUrl                   *string `pulumi:"deliveryCallbackUrl,optional"`
	DeliveryCallbackEnabled               *bool   `pulumi:"deliveryCallbackEnabled,optional"`
}

func (a *NotifyServiceArgs) Annotate(an infer.Annotator) {
	an.Describe(&a.FriendlyName, "Human-readable name of the service, up to 64 characters. Removing it leaves the name unchanged.")
	an.Describe(&a.ApnCredentialSid, "SID (CR...) of the NotifyCredential used for APNs bindings. Removing it leaves the setting unchanged.")
	an.Describe(&a.GcmCredentialSid, "SID (CR...) of the NotifyCredential used for GCM bindings. Removing it leaves the setting unchanged.")
	an.Describe(&a.FcmCredentialSid, "SID (CR...) of the NotifyCredential used for FCM bindings. Removing it leaves the setting unchanged.")
	an.Describe(&a.MessagingServiceSid, "SID (MG...) of the Messaging service used for SMS bindings; it must be set to send SMS notifications. Removing it leaves the setting unchanged.")
	an.Describe(&a.DefaultApnNotificationProtocolVersion, "Protocol version used for APNs notifications unless a binding overrides it. Removing it leaves the setting unchanged.")
	an.Describe(&a.DefaultGcmNotificationProtocolVersion, "Protocol version used for GCM notifications unless a binding overrides it. Removing it leaves the setting unchanged.")
	an.Describe(&a.DefaultFcmNotificationProtocolVersion, "Protocol version used for FCM notifications unless a binding overrides it. Removing it leaves the setting unchanged.")
	an.Describe(&a.LogEnabled, "Whether Twilio logs notifications (Twilio default true).")
	an.Describe(&a.DeliveryCallbackUrl, "Absolute URL Twilio calls with delivery receipts. Removing it leaves the setting unchanged.")
	an.Describe(&a.DeliveryCallbackEnabled, "Whether Twilio calls deliveryCallbackUrl (Twilio default false).")
}

// NotifyServiceState is the persisted state of a NotifyService.
type NotifyServiceState struct {
	NotifyServiceArgs

	Sid         string `pulumi:"sid"`
	DateCreated string `pulumi:"dateCreated"`
	DateUpdated string `pulumi:"dateUpdated"`
}

func (s *NotifyServiceState) Annotate(a infer.Annotator) {
	a.Describe(&s.Sid, "The service SID (IS...).")
	a.Describe(&s.DateCreated, "When the service was created (ISO 8601).")
	a.Describe(&s.DateUpdated, "When the service was last updated (ISO 8601).")
}

func (r *NotifyService) Annotate(a infer.Annotator) {
	a.Describe(&r, "A Notify service. The resource ID is the service SID. Optional settings that are set are sent and kept in step with Twilio; one that is removed leaves the setting as it is on Twilio.")
}

var _ infer.CustomResource[NotifyServiceArgs, NotifyServiceState] = (*NotifyService)(nil)
var _ infer.CustomCheck[NotifyServiceArgs] = (*NotifyService)(nil)
var _ infer.CustomDiff[NotifyServiceArgs, NotifyServiceState] = (*NotifyService)(nil)
var _ infer.CustomRead[NotifyServiceArgs, NotifyServiceState] = (*NotifyService)(nil)
var _ infer.CustomUpdate[NotifyServiceArgs, NotifyServiceState] = (*NotifyService)(nil)
var _ infer.CustomDelete[NotifyServiceState] = (*NotifyService)(nil)

func (r *NotifyService) Check(ctx context.Context, req infer.CheckRequest) (infer.CheckResponse[NotifyServiceArgs], error) {
	args, failures, err := infer.DefaultCheck[NotifyServiceArgs](ctx, req.NewInputs)
	if err != nil || len(failures) > 0 {
		return infer.CheckResponse[NotifyServiceArgs]{Inputs: args, Failures: failures}, err
	}
	c := &checker{unk: unknowns(req.NewInputs)}
	if v := args.FriendlyName; v != nil && !c.unk("friendlyName") && len(*v) > 64 {
		c.fail("friendlyName", "friendlyName must be at most 64 characters")
	}
	for prop, v := range map[string]*string{"apnCredentialSid": args.ApnCredentialSid, "gcmCredentialSid": args.GcmCredentialSid, "fcmCredentialSid": args.FcmCredentialSid} {
		if v != nil {
			c.sid(prop, *v, notifyCredentialSidRE, "Notify credential (CR followed by 32 hex characters)")
		}
	}
	if v := args.MessagingServiceSid; v != nil {
		c.sid("messagingServiceSid", *v, notifyMessagingSidRE, "Messaging service (MG followed by 32 hex characters)")
	}
	return infer.CheckResponse[NotifyServiceArgs]{Inputs: args, Failures: c.failures}, nil
}

func (r *NotifyService) Diff(_ context.Context, req infer.DiffRequest[NotifyServiceArgs, NotifyServiceState]) (infer.DiffResponse, error) {
	o, n := req.State.NotifyServiceArgs, req.Inputs
	d := trDiffSet{}
	d.strDef(notifyServiceDefaults, "friendlyName", o.FriendlyName, n.FriendlyName)
	d.strDef(notifyServiceDefaults, "apnCredentialSid", o.ApnCredentialSid, n.ApnCredentialSid)
	d.strDef(notifyServiceDefaults, "gcmCredentialSid", o.GcmCredentialSid, n.GcmCredentialSid)
	d.strDef(notifyServiceDefaults, "fcmCredentialSid", o.FcmCredentialSid, n.FcmCredentialSid)
	d.strDef(notifyServiceDefaults, "messagingServiceSid", o.MessagingServiceSid, n.MessagingServiceSid)
	d.strDef(notifyServiceDefaults, "defaultApnNotificationProtocolVersion", o.DefaultApnNotificationProtocolVersion, n.DefaultApnNotificationProtocolVersion)
	d.strDef(notifyServiceDefaults, "defaultGcmNotificationProtocolVersion", o.DefaultGcmNotificationProtocolVersion, n.DefaultGcmNotificationProtocolVersion)
	d.strDef(notifyServiceDefaults, "defaultFcmNotificationProtocolVersion", o.DefaultFcmNotificationProtocolVersion, n.DefaultFcmNotificationProtocolVersion)
	d.boolDef(notifyServiceDefaults, "logEnabled", o.LogEnabled, n.LogEnabled)
	d.strDef(notifyServiceDefaults, "deliveryCallbackUrl", o.DeliveryCallbackUrl, n.DeliveryCallbackUrl)
	d.boolDef(notifyServiceDefaults, "deliveryCallbackEnabled", o.DeliveryCallbackEnabled, n.DeliveryCallbackEnabled)
	return d.response(), nil
}

func (a NotifyServiceArgs) form() url.Values {
	form := url.Values{}
	setS(form, "FriendlyName", a.FriendlyName)
	setS(form, "ApnCredentialSid", a.ApnCredentialSid)
	setS(form, "GcmCredentialSid", a.GcmCredentialSid)
	setS(form, "FcmCredentialSid", a.FcmCredentialSid)
	setS(form, "MessagingServiceSid", a.MessagingServiceSid)
	setS(form, "DefaultApnNotificationProtocolVersion", a.DefaultApnNotificationProtocolVersion)
	setS(form, "DefaultGcmNotificationProtocolVersion", a.DefaultGcmNotificationProtocolVersion)
	setS(form, "DefaultFcmNotificationProtocolVersion", a.DefaultFcmNotificationProtocolVersion)
	setB(form, "LogEnabled", a.LogEnabled)
	setS(form, "DeliveryCallbackUrl", a.DeliveryCallbackUrl)
	setB(form, "DeliveryCallbackEnabled", a.DeliveryCallbackEnabled)
	return form
}

func (r *NotifyService) Create(ctx context.Context, req infer.CreateRequest[NotifyServiceArgs]) (infer.CreateResponse[NotifyServiceState], error) {
	type resp = infer.CreateResponse[NotifyServiceState]
	a := req.Inputs
	if req.DryRun {
		return resp{Output: NotifyServiceState{NotifyServiceArgs: a}}, nil
	}
	c, err := clientFor(ctx)
	if err != nil {
		return resp{}, err
	}
	o, err := c.do(ctx, "POST", notifyProduct, notifyPath("Services", ""), nil, a.form())
	if err != nil {
		return resp{}, fmt.Errorf("create Notify service: %w", err)
	}
	st := NotifyServiceState{NotifyServiceArgs: a}
	st.Sid, st.DateCreated, st.DateUpdated = trSid(o)
	return resp{ID: st.Sid, Output: st}, nil
}

func (r *NotifyService) Read(ctx context.Context, req infer.ReadRequest[NotifyServiceArgs, NotifyServiceState]) (infer.ReadResponse[NotifyServiceArgs, NotifyServiceState], error) {
	type resp = infer.ReadResponse[NotifyServiceArgs, NotifyServiceState]
	c, err := clientFor(ctx)
	if err != nil {
		return resp{}, err
	}
	o, err := c.do(ctx, "GET", notifyProduct, notifyPath("Services", req.ID), nil, nil)
	if err != nil {
		if isNotFound(err) {
			if err := notFoundOnImport("Notify service", req.ID, req.Inputs, req.State); err != nil {
				return resp{}, err
			}
			return resp{}, nil
		}
		return resp{}, fmt.Errorf("read Notify service %s: %w", req.ID, err)
	}
	in := req.Inputs
	// An import arrives with no prior state; the program's own settings are the only record of what it manages.
	imported := req.State.Sid == ""
	in.FriendlyName = trStr(in.FriendlyName, o.optStr("friendly_name"), notifyServiceDefaults.str("friendlyName"), imported)
	in.ApnCredentialSid = trStr(in.ApnCredentialSid, o.optStr("apn_credential_sid"), notifyServiceDefaults.str("apnCredentialSid"), imported)
	in.GcmCredentialSid = trStr(in.GcmCredentialSid, o.optStr("gcm_credential_sid"), notifyServiceDefaults.str("gcmCredentialSid"), imported)
	in.FcmCredentialSid = trStr(in.FcmCredentialSid, o.optStr("fcm_credential_sid"), notifyServiceDefaults.str("fcmCredentialSid"), imported)
	in.MessagingServiceSid = trStr(in.MessagingServiceSid, o.optStr("messaging_service_sid"), notifyServiceDefaults.str("messagingServiceSid"), imported)
	in.DefaultApnNotificationProtocolVersion = trStr(in.DefaultApnNotificationProtocolVersion, o.optStr("default_apn_notification_protocol_version"), notifyServiceDefaults.str("defaultApnNotificationProtocolVersion"), imported)
	in.DefaultGcmNotificationProtocolVersion = trStr(in.DefaultGcmNotificationProtocolVersion, o.optStr("default_gcm_notification_protocol_version"), notifyServiceDefaults.str("defaultGcmNotificationProtocolVersion"), imported)
	in.DefaultFcmNotificationProtocolVersion = trStr(in.DefaultFcmNotificationProtocolVersion, o.optStr("default_fcm_notification_protocol_version"), notifyServiceDefaults.str("defaultFcmNotificationProtocolVersion"), imported)
	in.LogEnabled = trBool(in.LogEnabled, o.optBool("log_enabled"), notifyServiceDefaults.boolean("logEnabled"), imported)
	in.DeliveryCallbackUrl = trStr(in.DeliveryCallbackUrl, o.optStr("delivery_callback_url"), notifyServiceDefaults.str("deliveryCallbackUrl"), imported)
	in.DeliveryCallbackEnabled = trBool(in.DeliveryCallbackEnabled, o.optBool("delivery_callback_enabled"), notifyServiceDefaults.boolean("deliveryCallbackEnabled"), imported)
	st := NotifyServiceState{NotifyServiceArgs: in}
	st.Sid, st.DateCreated, st.DateUpdated = trSid(o)
	return resp{ID: req.ID, Inputs: in, State: st}, nil
}

func (r *NotifyService) Update(ctx context.Context, req infer.UpdateRequest[NotifyServiceArgs, NotifyServiceState]) (infer.UpdateResponse[NotifyServiceState], error) {
	type resp = infer.UpdateResponse[NotifyServiceState]
	st := req.State
	st.NotifyServiceArgs = req.Inputs
	if req.DryRun {
		return resp{Output: st}, nil
	}
	c, err := clientFor(ctx)
	if err != nil {
		return resp{}, err
	}
	o, err := c.do(ctx, "POST", notifyProduct, notifyPath("Services", req.ID), nil, req.Inputs.form())
	if err != nil {
		return resp{}, fmt.Errorf("update Notify service %s: %w", req.ID, err)
	}
	st.DateUpdated = o.str("date_updated")
	return resp{Output: st}, nil
}

func (r *NotifyService) Delete(ctx context.Context, req infer.DeleteRequest[NotifyServiceState]) (infer.DeleteResponse, error) {
	c, err := clientFor(ctx)
	if err != nil {
		return infer.DeleteResponse{}, err
	}
	if _, err := c.do(ctx, "DELETE", notifyProduct, notifyPath("Services", req.ID), nil, nil); err != nil && !isNotFound(err) {
		return infer.DeleteResponse{}, fmt.Errorf("delete Notify service %s: %w", req.ID, err)
	}
	return infer.DeleteResponse{}, nil
}

// notifyServiceDefaults are the Twilio defaults of this resource, shared by Read (import adoption) and Diff.
var notifyServiceDefaults = trDefaults{
	"friendlyName":                          "",
	"apnCredentialSid":                      "",
	"gcmCredentialSid":                      "",
	"fcmCredentialSid":                      "",
	"messagingServiceSid":                   "",
	"defaultApnNotificationProtocolVersion": "",
	"defaultGcmNotificationProtocolVersion": "",
	"defaultFcmNotificationProtocolVersion": "",
	"logEnabled":                            true,
	"deliveryCallbackUrl":                   "",
	"deliveryCallbackEnabled":               false,
}
