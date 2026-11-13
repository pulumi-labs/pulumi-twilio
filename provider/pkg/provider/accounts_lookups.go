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
	"net/url"

	"github.com/pulumi/pulumi-go-provider/infer"
)

// Lookups of the Accounts v1 objects.

func accountsLookup(ctx context.Context, fn, what, path string, query map[string]string) (obj, error) {
	c, err := clientFor(ctx)
	if err != nil {
		return nil, err
	}
	var q url.Values
	if len(query) > 0 {
		q = url.Values{}
		for k, v := range query {
			q.Set(k, v)
		}
	}
	o, err := c.do(ctx, "GET", accountsProduct, path, q, nil)
	if err != nil {
		return nil, lookupFailure(fn, what, err)
	}
	return o, nil
}

// ------------------------------------------------------------ getAwsCredential

// GetAwsCredential looks up an AWS credential by SID.
type GetAwsCredential struct{}

// GetAwsCredentialArgs are the lookup arguments.
type GetAwsCredentialArgs struct {
	Sid string `pulumi:"sid"`
}

func (a *GetAwsCredentialArgs) Annotate(an infer.Annotator) {
	an.Describe(&a.Sid, "SID (CR...) of the AWS credential.")
}

// GetAwsCredentialResult is the looked-up credential. The keys are never returned by Twilio.
type GetAwsCredentialResult struct {
	Sid          string  `pulumi:"sid"`
	FriendlyName *string `pulumi:"friendlyName,optional"`
	AccountSid   *string `pulumi:"accountSid,optional"`
	DateCreated  string  `pulumi:"dateCreated"`
	DateUpdated  string  `pulumi:"dateUpdated"`
}

func (f *GetAwsCredential) Annotate(a infer.Annotator) {
	a.Describe(&f, "Look up an AWS credential by SID. Twilio never returns the keys.")
}

func (f *GetAwsCredential) Invoke(ctx context.Context, req infer.FunctionRequest[GetAwsCredentialArgs]) (infer.FunctionResponse[GetAwsCredentialResult], error) {
	const fn = "getAwsCredential"
	type resp = infer.FunctionResponse[GetAwsCredentialResult]
	if err := requireArg(fn, "sid", req.Input.Sid); err != nil {
		return resp{}, err
	}
	o, err := accountsLookup(ctx, fn, "AWS credential "+req.Input.Sid, accountsCredentialPath("AWS", req.Input.Sid), nil)
	if err != nil {
		return resp{}, err
	}
	sid, created, updated := trSid(o)
	return resp{Output: GetAwsCredentialResult{Sid: sid, FriendlyName: nonEmpty(o.optStr("friendly_name")),
		AccountSid: nonEmpty(o.optStr("account_sid")), DateCreated: created, DateUpdated: updated}}, nil
}

// ------------------------------------------------------------ getPublicKeyCredential

// GetPublicKeyCredential looks up a public key credential by SID.
type GetPublicKeyCredential struct{}

// GetPublicKeyCredentialArgs are the lookup arguments.
type GetPublicKeyCredentialArgs struct {
	Sid string `pulumi:"sid"`
}

func (a *GetPublicKeyCredentialArgs) Annotate(an infer.Annotator) {
	an.Describe(&a.Sid, "SID (CR...) of the public key credential.")
}

// GetPublicKeyCredentialResult is the looked-up credential. The key is never returned by Twilio.
type GetPublicKeyCredentialResult struct {
	Sid          string  `pulumi:"sid"`
	FriendlyName *string `pulumi:"friendlyName,optional"`
	AccountSid   *string `pulumi:"accountSid,optional"`
	DateCreated  string  `pulumi:"dateCreated"`
	DateUpdated  string  `pulumi:"dateUpdated"`
}

func (f *GetPublicKeyCredential) Annotate(a infer.Annotator) {
	a.Describe(&f, "Look up a public key credential by SID. Twilio never returns the key.")
}

func (f *GetPublicKeyCredential) Invoke(ctx context.Context, req infer.FunctionRequest[GetPublicKeyCredentialArgs]) (infer.FunctionResponse[GetPublicKeyCredentialResult], error) {
	const fn = "getPublicKeyCredential"
	type resp = infer.FunctionResponse[GetPublicKeyCredentialResult]
	if err := requireArg(fn, "sid", req.Input.Sid); err != nil {
		return resp{}, err
	}
	o, err := accountsLookup(ctx, fn, "public key credential "+req.Input.Sid, accountsCredentialPath("PublicKeys", req.Input.Sid), nil)
	if err != nil {
		return resp{}, err
	}
	sid, created, updated := trSid(o)
	return resp{Output: GetPublicKeyCredentialResult{Sid: sid, FriendlyName: nonEmpty(o.optStr("friendly_name")),
		AccountSid: nonEmpty(o.optStr("account_sid")), DateCreated: created, DateUpdated: updated}}, nil
}

// ------------------------------------------------------------ getAccountSafeListNumber

// GetAccountSafeListNumber looks up a phone number on the Global SafeList.
type GetAccountSafeListNumber struct{}

// GetAccountSafeListNumberArgs are the lookup arguments.
type GetAccountSafeListNumberArgs struct {
	PhoneNumber string `pulumi:"phoneNumber"`
}

func (a *GetAccountSafeListNumberArgs) Annotate(an infer.Annotator) {
	an.Describe(&a.PhoneNumber, "The phone number (or number prefix) in E.164 format, for example +14155550100.")
}

// GetAccountSafeListNumberResult is the looked-up SafeList entry.
type GetAccountSafeListNumberResult struct {
	PhoneNumber string  `pulumi:"phoneNumber"`
	Sid         *string `pulumi:"sid,optional"`
}

func (f *GetAccountSafeListNumber) Annotate(a infer.Annotator) {
	a.Describe(&f, "Look up a phone number on the account's Global SafeList (accounts.twilio.com). Fails when the number is not on the list. Not the Verify SafeList (see getVerifySafeListNumber).")
}

func (f *GetAccountSafeListNumber) Invoke(ctx context.Context, req infer.FunctionRequest[GetAccountSafeListNumberArgs]) (infer.FunctionResponse[GetAccountSafeListNumberResult], error) {
	const fn = "getAccountSafeListNumber"
	type resp = infer.FunctionResponse[GetAccountSafeListNumberResult]
	if err := requireArg(fn, "phoneNumber", req.Input.PhoneNumber); err != nil {
		return resp{}, err
	}
	o, err := accountsLookup(ctx, fn, "Global SafeList number "+req.Input.PhoneNumber, accountsSafeListPath, map[string]string{"PhoneNumber": req.Input.PhoneNumber})
	if err != nil {
		return resp{}, err
	}
	number := o.str("phone_number")
	if number == "" {
		return resp{}, lookupFailure(fn, "Global SafeList number "+req.Input.PhoneNumber, &apiError{Status: 404, Message: "the number is not on the Global SafeList"})
	}
	return resp{Output: GetAccountSafeListNumberResult{PhoneNumber: number, Sid: nonEmpty(o.optStr("sid"))}}, nil
}

// ------------------------------------------------------------ getMessagingGeoPermissions

// GetMessagingGeoPermissions reads the messaging geo permissions of the account.
type GetMessagingGeoPermissions struct{}

// GetMessagingGeoPermissionsArgs are the lookup arguments.
type GetMessagingGeoPermissionsArgs struct {
	CountryCodes *string `pulumi:"countryCodes,optional"`
}

func (a *GetMessagingGeoPermissionsArgs) Annotate(an infer.Annotator) {
	an.Describe(&a.CountryCodes, "Comma separated ISO country codes to report, for example IN,PK,US. Omit it for every country.")
}

// MessagingGeoPermission is the permission of one country.
type MessagingGeoPermission struct {
	CountryCode string  `pulumi:"countryCode"`
	Type        *string `pulumi:"type,optional"`
	Enabled     *bool   `pulumi:"enabled,optional"`
	Prefix      *string `pulumi:"prefix,optional"`
	Message     *string `pulumi:"message,optional"`
}

func (a *MessagingGeoPermission) Annotate(an infer.Annotator) {
	an.Describe(&a.CountryCode, "The ISO country code.")
	an.Describe(&a.Type, "The kind of permission, for example country.")
	an.Describe(&a.Enabled, "Whether the account may send messages to the country.")
	an.Describe(&a.Prefix, "The international dialing prefix of the country.")
	an.Describe(&a.Message, "A note Twilio attaches to the country, for example a risk warning.")
}

// GetMessagingGeoPermissionsResult is the looked-up permissions.
type GetMessagingGeoPermissionsResult struct {
	Permissions []MessagingGeoPermission `pulumi:"permissions"`
}

func (a *GetMessagingGeoPermissionsResult) Annotate(an infer.Annotator) {
	an.Describe(&a.Permissions, "The permission of each country.")
}

func (f *GetMessagingGeoPermissions) Annotate(a infer.Annotator) {
	a.Describe(&f, "Read the messaging geo permissions of the account (accounts.twilio.com /v1/Messaging/GeoPermissions). They can be changed in the Twilio Console only; the API is read-only.")
}

func (f *GetMessagingGeoPermissions) Invoke(ctx context.Context, req infer.FunctionRequest[GetMessagingGeoPermissionsArgs]) (infer.FunctionResponse[GetMessagingGeoPermissionsResult], error) {
	const fn = "getMessagingGeoPermissions"
	type resp = infer.FunctionResponse[GetMessagingGeoPermissionsResult]
	var query map[string]string
	if v := req.Input.CountryCodes; v != nil && *v != "" {
		query = map[string]string{"CountryCode": *v}
	}
	o, err := accountsLookup(ctx, fn, "messaging geo permissions", "/v1/Messaging/GeoPermissions", query)
	if err != nil {
		return resp{}, err
	}
	out := GetMessagingGeoPermissionsResult{Permissions: []MessagingGeoPermission{}}
	if list, ok := o.get("permissions").([]any); ok {
		for _, e := range list {
			m, ok := e.(map[string]any)
			if !ok {
				continue
			}
			p := obj(m)
			out.Permissions = append(out.Permissions, MessagingGeoPermission{CountryCode: p.str("country_code"), Type: nonEmpty(p.optStr("type")),
				Enabled: p.optBool("enabled"), Prefix: nonEmpty(p.optStr("prefix")), Message: nonEmpty(p.optStr("message"))})
		}
	}
	return resp{Output: out}, nil
}
