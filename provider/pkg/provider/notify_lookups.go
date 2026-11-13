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

// Lookups of the Notify objects. Each reads one object and returns what Twilio reports, null where
// Twilio reports nothing. Secrets (push credential keys) are never returned.

// notifyLookup reads one object for a lookup function.
func notifyLookup(ctx context.Context, fn, what, path string) (obj, error) {
	c, err := clientFor(ctx)
	if err != nil {
		return nil, err
	}
	o, err := c.do(ctx, "GET", notifyProduct, path, nil, nil)
	if err != nil {
		return nil, lookupFailure(fn, what, err)
	}
	return o, nil
}

// ------------------------------------------------------------ getNotifyService

// GetNotifyService looks up a Notify service by SID.
type GetNotifyService struct{}

// GetNotifyServiceArgs are the lookup arguments.
type GetNotifyServiceArgs struct {
	Sid string `pulumi:"sid"`
}

func (a *GetNotifyServiceArgs) Annotate(an infer.Annotator) {
	an.Describe(&a.Sid, "SID (IS...) of the Notify service.")
}

// GetNotifyServiceResult is the looked-up service.
type GetNotifyServiceResult struct {
	Sid                                   string  `pulumi:"sid"`
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
	DateCreated                           string  `pulumi:"dateCreated"`
	DateUpdated                           string  `pulumi:"dateUpdated"`
}

func (f *GetNotifyService) Annotate(a infer.Annotator) {
	a.Describe(&f, "Look up a Notify service by SID.")
}

func (f *GetNotifyService) Invoke(ctx context.Context, req infer.FunctionRequest[GetNotifyServiceArgs]) (infer.FunctionResponse[GetNotifyServiceResult], error) {
	const fn = "getNotifyService"
	type resp = infer.FunctionResponse[GetNotifyServiceResult]
	if err := requireArg(fn, "sid", req.Input.Sid); err != nil {
		return resp{}, err
	}
	o, err := notifyLookup(ctx, fn, "Notify service "+req.Input.Sid, notifyPath("Services", req.Input.Sid))
	if err != nil {
		return resp{}, err
	}
	sid, created, updated := trSid(o)
	return resp{Output: GetNotifyServiceResult{
		Sid:                                   sid,
		FriendlyName:                          nonEmpty(o.optStr("friendly_name")),
		ApnCredentialSid:                      nonEmpty(o.optStr("apn_credential_sid")),
		GcmCredentialSid:                      nonEmpty(o.optStr("gcm_credential_sid")),
		FcmCredentialSid:                      nonEmpty(o.optStr("fcm_credential_sid")),
		MessagingServiceSid:                   nonEmpty(o.optStr("messaging_service_sid")),
		DefaultApnNotificationProtocolVersion: nonEmpty(o.optStr("default_apn_notification_protocol_version")),
		DefaultGcmNotificationProtocolVersion: nonEmpty(o.optStr("default_gcm_notification_protocol_version")),
		DefaultFcmNotificationProtocolVersion: nonEmpty(o.optStr("default_fcm_notification_protocol_version")),
		LogEnabled:                            o.optBool("log_enabled"),
		DeliveryCallbackUrl:                   nonEmpty(o.optStr("delivery_callback_url")),
		DeliveryCallbackEnabled:               o.optBool("delivery_callback_enabled"),
		DateCreated:                           created,
		DateUpdated:                           updated,
	}}, nil
}

// ------------------------------------------------------------ getNotifyCredential

// GetNotifyCredential looks up a Notify push credential by SID.
type GetNotifyCredential struct{}

// GetNotifyCredentialArgs are the lookup arguments.
type GetNotifyCredentialArgs struct {
	Sid string `pulumi:"sid"`
}

func (a *GetNotifyCredentialArgs) Annotate(an infer.Annotator) {
	an.Describe(&a.Sid, "SID (CR...) of the credential.")
}

// GetNotifyCredentialResult is the looked-up credential; the certificate, keys and secret are never returned.
type GetNotifyCredentialResult struct {
	Sid            string  `pulumi:"sid"`
	CredentialType string  `pulumi:"credentialType"`
	FriendlyName   *string `pulumi:"friendlyName,optional"`
	Sandbox        *bool   `pulumi:"sandbox,optional"`
	DateCreated    string  `pulumi:"dateCreated"`
	DateUpdated    string  `pulumi:"dateUpdated"`
}

func (f *GetNotifyCredential) Annotate(a infer.Annotator) {
	a.Describe(&f, "Look up a Notify push credential by SID. The certificate, keys and secret are never returned.")
}

func (f *GetNotifyCredential) Invoke(ctx context.Context, req infer.FunctionRequest[GetNotifyCredentialArgs]) (infer.FunctionResponse[GetNotifyCredentialResult], error) {
	const fn = "getNotifyCredential"
	type resp = infer.FunctionResponse[GetNotifyCredentialResult]
	if err := requireArg(fn, "sid", req.Input.Sid); err != nil {
		return resp{}, err
	}
	o, err := notifyLookup(ctx, fn, "Notify credential "+req.Input.Sid, notifyPath("Credentials", req.Input.Sid))
	if err != nil {
		return resp{}, err
	}
	sid, created, updated := trSid(o)
	return resp{Output: GetNotifyCredentialResult{
		Sid: sid, CredentialType: o.str("type"), FriendlyName: nonEmpty(o.optStr("friendly_name")),
		Sandbox: o.optBool("sandbox"), DateCreated: created, DateUpdated: updated,
	}}, nil
}
