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

	"github.com/pulumi/pulumi-go-provider/infer"
)

// GetShortCode looks up an SMS short code the account leases.
type GetShortCode struct{}

// GetShortCodeArgs are the lookup arguments; set exactly one of code or sid.
type GetShortCodeArgs struct {
	Code       *string `pulumi:"code,optional"`
	Sid        *string `pulumi:"sid,optional"`
	AccountSid *string `pulumi:"accountSid,optional"`
}

func (a *GetShortCodeArgs) Annotate(an infer.Annotator) {
	an.Describe(&a.Code, "The short code, for example 894546. Exactly one of code or sid must be set.")
	an.Describe(&a.Sid, "The short code SID (SC...). Exactly one of code or sid must be set.")
	an.Describe(&a.AccountSid, "SID of the account that owns the short code. Defaults to the provider account.")
}

// GetShortCodeResult is the looked-up short code.
type GetShortCodeResult struct {
	Sid               string `pulumi:"sid"`
	Code              string `pulumi:"code"`
	FriendlyName      string `pulumi:"friendlyName"`
	AccountSid        string `pulumi:"accountSid"`
	ApiVersion        string `pulumi:"apiVersion"`
	SmsUrl            string `pulumi:"smsUrl"`
	SmsMethod         string `pulumi:"smsMethod"`
	SmsFallbackUrl    string `pulumi:"smsFallbackUrl"`
	SmsFallbackMethod string `pulumi:"smsFallbackMethod"`
}

func (a *GetShortCodeResult) Annotate(an infer.Annotator) {
	an.Describe(&a.Sid, "The short code SID (SC...).")
	an.Describe(&a.Code, "The short code.")
	an.Describe(&a.FriendlyName, "Label of the short code.")
	an.Describe(&a.AccountSid, "SID of the account that owns the short code.")
	an.Describe(&a.ApiVersion, "Twilio API version used for the short code's webhooks.")
	an.Describe(&a.SmsUrl, "Current SMS webhook URL (empty when none).")
	an.Describe(&a.SmsMethod, "HTTP method for the SMS webhook.")
	an.Describe(&a.SmsFallbackUrl, "Current SMS fallback URL (empty when none).")
	an.Describe(&a.SmsFallbackMethod, "HTTP method for the SMS fallback URL.")
}

func (f *GetShortCode) Annotate(a infer.Annotator) {
	a.Describe(&f, "Look up an SMS short code leased by the account, by code or SID.")
}

func (f *GetShortCode) Invoke(ctx context.Context, req infer.FunctionRequest[GetShortCodeArgs]) (infer.FunctionResponse[GetShortCodeResult], error) {
	in := req.Input
	if (deref(in.Code) != "") == (deref(in.Sid) != "") {
		return infer.FunctionResponse[GetShortCodeResult]{}, fmt.Errorf("getShortCode: set exactly one of code or sid")
	}
	c, err := clientFor(ctx)
	if err != nil {
		return infer.FunctionResponse[GetShortCodeResult]{}, err
	}
	acct := c.acct(in.AccountSid)
	o, err := findShortCode(ctx, c, acct, deref(in.Sid), deref(in.Code))
	if err != nil {
		return infer.FunctionResponse[GetShortCodeResult]{}, fmt.Errorf("getShortCode: %w", err)
	}
	return infer.FunctionResponse[GetShortCodeResult]{Output: GetShortCodeResult{
		Sid: o.str("sid"), Code: o.str("short_code"), FriendlyName: o.str("friendly_name"),
		AccountSid: derefOr(nonEmpty(o.optStr("account_sid")), acct), ApiVersion: o.str("api_version"),
		SmsUrl: o.str("sms_url"), SmsMethod: o.str("sms_method"),
		SmsFallbackUrl: o.str("sms_fallback_url"), SmsFallbackMethod: o.str("sms_fallback_method"),
	}}, nil
}

// GetConnectApp looks up a Twilio Connect app.
type GetConnectApp struct{}

// GetConnectAppArgs are the lookup arguments.
type GetConnectAppArgs struct {
	Sid        string  `pulumi:"sid"`
	AccountSid *string `pulumi:"accountSid,optional"`
}

func (a *GetConnectAppArgs) Annotate(an infer.Annotator) {
	an.Describe(&a.Sid, "The Connect app SID (CN...).")
	an.Describe(&a.AccountSid, "SID of the account that owns the app. Defaults to the provider account.")
}

// GetConnectAppResult is the looked-up Connect app.
type GetConnectAppResult struct {
	Sid                       string   `pulumi:"sid"`
	AccountSid                string   `pulumi:"accountSid"`
	FriendlyName              string   `pulumi:"friendlyName"`
	AuthorizeRedirectUrl      string   `pulumi:"authorizeRedirectUrl"`
	CompanyName               string   `pulumi:"companyName"`
	DeauthorizeCallbackUrl    string   `pulumi:"deauthorizeCallbackUrl"`
	DeauthorizeCallbackMethod string   `pulumi:"deauthorizeCallbackMethod"`
	Description               string   `pulumi:"description"`
	HomepageUrl               string   `pulumi:"homepageUrl"`
	Permissions               []string `pulumi:"permissions"`
}

func (a *GetConnectAppResult) Annotate(an infer.Annotator) {
	an.Describe(&a.Sid, "The Connect app SID (CN...).")
	an.Describe(&a.AccountSid, "SID of the account that owns the app.")
	an.Describe(&a.FriendlyName, "Name of the app.")
	an.Describe(&a.AuthorizeRedirectUrl, "URL users are redirected to after authorizing the app.")
	an.Describe(&a.CompanyName, "Company name shown on the authorization page.")
	an.Describe(&a.DeauthorizeCallbackUrl, "URL requested when a user deauthorizes the app (empty when none).")
	an.Describe(&a.DeauthorizeCallbackMethod, "HTTP method for the deauthorize callback.")
	an.Describe(&a.Description, "Description shown on the authorization page.")
	an.Describe(&a.HomepageUrl, "Public homepage URL.")
	an.Describe(&a.Permissions, "Permissions the app requests: get-all and post-all.")
}

func (f *GetConnectApp) Annotate(a infer.Annotator) {
	a.Describe(&f, "Look up a Twilio Connect app by SID.")
}

func (f *GetConnectApp) Invoke(ctx context.Context, req infer.FunctionRequest[GetConnectAppArgs]) (infer.FunctionResponse[GetConnectAppResult], error) {
	in := req.Input
	if err := requireArg("getConnectApp", "sid", in.Sid); err != nil {
		return infer.FunctionResponse[GetConnectAppResult]{}, err
	}
	c, err := clientFor(ctx)
	if err != nil {
		return infer.FunctionResponse[GetConnectAppResult]{}, err
	}
	acct := c.acct(in.AccountSid)
	o, err := fetchAdopted(ctx, c, acct, connectAppColl, in.Sid)
	if err != nil {
		return infer.FunctionResponse[GetConnectAppResult]{}, fmt.Errorf("getConnectApp: %w", err)
	}
	perms := []string{}
	if l := o.optList("permissions"); l != nil {
		perms = *l
	}
	return infer.FunctionResponse[GetConnectAppResult]{Output: GetConnectAppResult{
		Sid: o.str("sid"), AccountSid: derefOr(nonEmpty(o.optStr("account_sid")), acct), FriendlyName: o.str("friendly_name"),
		AuthorizeRedirectUrl: o.str("authorize_redirect_url"), CompanyName: o.str("company_name"),
		DeauthorizeCallbackUrl: o.str("deauthorize_callback_url"), DeauthorizeCallbackMethod: o.str("deauthorize_callback_method"),
		Description: o.str("description"), HomepageUrl: o.str("homepage_url"), Permissions: perms,
	}}, nil
}

// GetOutgoingCallerId looks up a verified outgoing caller ID.
type GetOutgoingCallerId struct{}

// GetOutgoingCallerIdArgs are the lookup arguments; set exactly one of phoneNumber or sid.
type GetOutgoingCallerIdArgs struct {
	PhoneNumber *string `pulumi:"phoneNumber,optional"`
	Sid         *string `pulumi:"sid,optional"`
	AccountSid  *string `pulumi:"accountSid,optional"`
}

func (a *GetOutgoingCallerIdArgs) Annotate(an infer.Annotator) {
	an.Describe(&a.PhoneNumber, "The verified number in E.164 format. Exactly one of phoneNumber or sid must be set.")
	an.Describe(&a.Sid, "The caller ID SID (PN...). Exactly one of phoneNumber or sid must be set.")
	an.Describe(&a.AccountSid, "SID of the account that verified the number. Defaults to the provider account.")
}

// GetOutgoingCallerIdResult is the looked-up caller ID.
type GetOutgoingCallerIdResult struct {
	Sid          string `pulumi:"sid"`
	PhoneNumber  string `pulumi:"phoneNumber"`
	FriendlyName string `pulumi:"friendlyName"`
	AccountSid   string `pulumi:"accountSid"`
	DateCreated  string `pulumi:"dateCreated"`
}

func (a *GetOutgoingCallerIdResult) Annotate(an infer.Annotator) {
	an.Describe(&a.Sid, "The caller ID SID (PN...).")
	an.Describe(&a.PhoneNumber, "The verified number in E.164 format.")
	an.Describe(&a.FriendlyName, "Label of the caller ID.")
	an.Describe(&a.AccountSid, "SID of the account that verified the number.")
	an.Describe(&a.DateCreated, "When the number was verified, as Twilio reports it.")
}

func (f *GetOutgoingCallerId) Annotate(a infer.Annotator) {
	a.Describe(&f, "Look up a verified outgoing caller ID by E.164 number or SID.")
}

func (f *GetOutgoingCallerId) Invoke(ctx context.Context, req infer.FunctionRequest[GetOutgoingCallerIdArgs]) (infer.FunctionResponse[GetOutgoingCallerIdResult], error) {
	in := req.Input
	if (deref(in.PhoneNumber) != "") == (deref(in.Sid) != "") {
		return infer.FunctionResponse[GetOutgoingCallerIdResult]{}, fmt.Errorf("getOutgoingCallerId: set exactly one of phoneNumber or sid")
	}
	c, err := clientFor(ctx)
	if err != nil {
		return infer.FunctionResponse[GetOutgoingCallerIdResult]{}, err
	}
	acct := c.acct(in.AccountSid)
	o, err := findCallerID(ctx, c, acct, deref(in.Sid), deref(in.PhoneNumber))
	if err != nil {
		return infer.FunctionResponse[GetOutgoingCallerIdResult]{}, fmt.Errorf("getOutgoingCallerId: %w", err)
	}
	return infer.FunctionResponse[GetOutgoingCallerIdResult]{Output: GetOutgoingCallerIdResult{
		Sid: o.str("sid"), PhoneNumber: o.str("phone_number"), FriendlyName: o.str("friendly_name"),
		AccountSid: derefOr(nonEmpty(o.optStr("account_sid")), acct), DateCreated: o.str("date_created"),
	}}, nil
}
