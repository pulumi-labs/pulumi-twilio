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

// Lookups of the Proxy objects. Each reads one object and returns what Twilio reports, null where
// Twilio reports nothing.

// proxyLookup reads one object for a lookup function.
func proxyLookup(ctx context.Context, fn, what, path string) (obj, error) {
	c, err := clientFor(ctx)
	if err != nil {
		return nil, err
	}
	o, err := c.do(ctx, "GET", proxyProduct, path, nil, nil)
	if err != nil {
		return nil, lookupFailure(fn, what, err)
	}
	return o, nil
}

// ------------------------------------------------------------ getProxyService

// GetProxyService looks up a Proxy service by SID.
type GetProxyService struct{}

// GetProxyServiceArgs are the lookup arguments.
type GetProxyServiceArgs struct {
	Sid string `pulumi:"sid"`
}

func (a *GetProxyServiceArgs) Annotate(an infer.Annotator) {
	an.Describe(&a.Sid, "SID (KS...) of the Proxy service.")
}

// GetProxyServiceResult is the looked-up service.
type GetProxyServiceResult struct {
	Sid                     string  `pulumi:"sid"`
	UniqueName              *string `pulumi:"uniqueName,optional"`
	DefaultTtl              *int    `pulumi:"defaultTtl,optional"`
	CallbackUrl             *string `pulumi:"callbackUrl,optional"`
	GeoMatchLevel           *string `pulumi:"geoMatchLevel,optional"`
	NumberSelectionBehavior *string `pulumi:"numberSelectionBehavior,optional"`
	InterceptCallbackUrl    *string `pulumi:"interceptCallbackUrl,optional"`
	OutOfSessionCallbackUrl *string `pulumi:"outOfSessionCallbackUrl,optional"`
	ChatInstanceSid         *string `pulumi:"chatInstanceSid,optional"`
	DateCreated             string  `pulumi:"dateCreated"`
	DateUpdated             string  `pulumi:"dateUpdated"`
}

func (f *GetProxyService) Annotate(a infer.Annotator) {
	a.Describe(&f, "Look up a Proxy service by SID.")
}

func (f *GetProxyService) Invoke(ctx context.Context, req infer.FunctionRequest[GetProxyServiceArgs]) (infer.FunctionResponse[GetProxyServiceResult], error) {
	const fn = "getProxyService"
	type resp = infer.FunctionResponse[GetProxyServiceResult]
	if err := requireArg(fn, "sid", req.Input.Sid); err != nil {
		return resp{}, err
	}
	o, err := proxyLookup(ctx, fn, "Proxy service "+req.Input.Sid, proxyPath(req.Input.Sid))
	if err != nil {
		return resp{}, err
	}
	sid, created, updated := trSid(o)
	return resp{Output: GetProxyServiceResult{
		Sid:                     sid,
		UniqueName:              nonEmpty(o.optStr("unique_name")),
		DefaultTtl:              o.optInt("default_ttl"),
		CallbackUrl:             nonEmpty(o.optStr("callback_url")),
		GeoMatchLevel:           nonEmpty(o.optStr("geo_match_level")),
		NumberSelectionBehavior: nonEmpty(o.optStr("number_selection_behavior")),
		InterceptCallbackUrl:    nonEmpty(o.optStr("intercept_callback_url")),
		OutOfSessionCallbackUrl: nonEmpty(o.optStr("out_of_session_callback_url")),
		ChatInstanceSid:         nonEmpty(o.optStr("chat_instance_sid")),
		DateCreated:             created,
		DateUpdated:             updated,
	}}, nil
}

// ------------------------------------------------------------ getProxyPhoneNumber

// GetProxyPhoneNumber looks up a phone number in the pool of a Proxy service.
type GetProxyPhoneNumber struct{}

// GetProxyPhoneNumberArgs are the lookup arguments.
type GetProxyPhoneNumberArgs struct {
	ServiceSid string `pulumi:"serviceSid"`
	Sid        string `pulumi:"sid"`
}

func (a *GetProxyPhoneNumberArgs) Annotate(an infer.Annotator) {
	an.Describe(&a.ServiceSid, "SID (KS...) of the Proxy service.")
	an.Describe(&a.Sid, "SID (PN...) of the phone number.")
}

// GetProxyPhoneNumberResult is the looked-up phone number.
type GetProxyPhoneNumberResult struct {
	Sid          string  `pulumi:"sid"`
	ServiceSid   string  `pulumi:"serviceSid"`
	PhoneNumber  *string `pulumi:"phoneNumber,optional"`
	FriendlyName *string `pulumi:"friendlyName,optional"`
	IsoCountry   *string `pulumi:"isoCountry,optional"`
	IsReserved   *bool   `pulumi:"isReserved,optional"`
	InUse        *int    `pulumi:"inUse,optional"`
	DateCreated  string  `pulumi:"dateCreated"`
	DateUpdated  string  `pulumi:"dateUpdated"`
}

func (f *GetProxyPhoneNumber) Annotate(a infer.Annotator) {
	a.Describe(&f, "Look up a phone number in the pool of a Proxy service.")
}

func (f *GetProxyPhoneNumber) Invoke(ctx context.Context, req infer.FunctionRequest[GetProxyPhoneNumberArgs]) (infer.FunctionResponse[GetProxyPhoneNumberResult], error) {
	const fn = "getProxyPhoneNumber"
	type resp = infer.FunctionResponse[GetProxyPhoneNumberResult]
	if err := requireArgs(fn, [2]string{"serviceSid", req.Input.ServiceSid}, [2]string{"sid", req.Input.Sid}); err != nil {
		return resp{}, err
	}
	o, err := proxyLookup(ctx, fn, "Proxy phone number "+req.Input.Sid, proxyPath(req.Input.ServiceSid, "PhoneNumbers", req.Input.Sid))
	if err != nil {
		return resp{}, err
	}
	sid, created, updated := trSid(o)
	return resp{Output: GetProxyPhoneNumberResult{
		Sid:          sid,
		ServiceSid:   req.Input.ServiceSid,
		PhoneNumber:  nonEmpty(o.optStr("phone_number")),
		FriendlyName: nonEmpty(o.optStr("friendly_name")),
		IsoCountry:   nonEmpty(o.optStr("iso_country")),
		IsReserved:   o.optBool("is_reserved"),
		InUse:        o.optInt("in_use"),
		DateCreated:  created,
		DateUpdated:  updated,
	}}, nil
}
