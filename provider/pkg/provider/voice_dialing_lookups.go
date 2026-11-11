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
	"strconv"

	"github.com/pulumi/pulumi-go-provider/infer"
)

// The Voice dialing permissions are account-wide geo permissions: which countries the account
// may dial, split into low-risk numbers, high-risk special services numbers and high-risk
// toll fraud numbers. Twilio exposes only reads per country (changes go through the bulk update
// endpoint, whose published wire format is inconsistent and not modelled, see #1665), so these
// are lookups only.

func dialingCountriesPath() string { return "/v1/DialingPermissions/Countries" }

func dialingCountryPath(iso string) string {
	return dialingCountriesPath() + "/" + url.PathEscape(iso)
}

// DialingCountry is the dialing permission of one country.
type DialingCountry struct {
	IsoCode                         string   `pulumi:"isoCode"`
	Name                            string   `pulumi:"name"`
	Continent                       string   `pulumi:"continent"`
	CountryCodes                    []string `pulumi:"countryCodes"`
	LowRiskNumbersEnabled           bool     `pulumi:"lowRiskNumbersEnabled"`
	HighRiskSpecialNumbersEnabled   bool     `pulumi:"highRiskSpecialNumbersEnabled"`
	HighRiskTollfraudNumbersEnabled bool     `pulumi:"highRiskTollfraudNumbersEnabled"`
}

func (a *DialingCountry) Annotate(an infer.Annotator) {
	an.Describe(&a.IsoCode, "The ISO 3166-1 alpha-2 country code.")
	an.Describe(&a.Name, "The name of the country.")
	an.Describe(&a.Continent, "The continent the country is in.")
	an.Describe(&a.CountryCodes, "The E.164 country calling codes assigned to the country (without the plus sign); empty when Twilio reports none.")
	an.Describe(&a.LowRiskNumbersEnabled, "Whether dialing low-risk numbers in the country is enabled; false when Twilio reports none.")
	an.Describe(&a.HighRiskSpecialNumbersEnabled, "Whether dialing high-risk special services numbers (premium, shared cost and similar ranges) in the country is enabled; false when Twilio reports none.")
	an.Describe(&a.HighRiskTollfraudNumbersEnabled, "Whether dialing high-risk toll fraud (IRSF) number ranges in the country is enabled; false when Twilio reports none.")
}

func dialingCountry(o obj) DialingCountry {
	codes := []string{}
	if l, ok := o.get("country_codes").([]any); ok {
		for _, c := range l {
			switch v := c.(type) {
			case string:
				codes = append(codes, v)
			case float64:
				codes = append(codes, strconv.FormatFloat(v, 'f', -1, 64))
			}
		}
	}
	b := func(k string) bool {
		if p := o.optBool(k); p != nil {
			return *p
		}
		return false
	}
	return DialingCountry{
		IsoCode: o.str("iso_code"), Name: o.str("name"), Continent: o.str("continent"), CountryCodes: codes,
		LowRiskNumbersEnabled: b("low_risk_numbers_enabled"), HighRiskSpecialNumbersEnabled: b("high_risk_special_numbers_enabled"),
		HighRiskTollfraudNumbersEnabled: b("high_risk_tollfraud_numbers_enabled"),
	}
}

// GetDialingPermissionsCountry looks up the Voice dialing permission of one country.
type GetDialingPermissionsCountry struct{}

// GetDialingPermissionsCountryArgs are the lookup arguments.
type GetDialingPermissionsCountryArgs struct {
	IsoCode string `pulumi:"isoCode"`
}

func (a *GetDialingPermissionsCountryArgs) Annotate(an infer.Annotator) {
	an.Describe(&a.IsoCode, "The ISO 3166-1 alpha-2 country code, for example `DE`.")
}

func (f *GetDialingPermissionsCountry) Annotate(a infer.Annotator) {
	a.Describe(&f, "Look up the Voice dialing permission of one country: whether the account may dial low-risk, high-risk special services and high-risk toll fraud numbers there (GET /v1/DialingPermissions/Countries/{IsoCode} on voice.twilio.com). Read-only: Twilio changes these through a bulk update that this provider does not model.")
}

func (f *GetDialingPermissionsCountry) Invoke(ctx context.Context, req infer.FunctionRequest[GetDialingPermissionsCountryArgs]) (infer.FunctionResponse[DialingCountry], error) {
	const fn = "getDialingPermissionsCountry"
	none := infer.FunctionResponse[DialingCountry]{}
	if err := requireArg(fn, "isoCode", req.Input.IsoCode); err != nil {
		return none, err
	}
	c, err := clientFor(ctx)
	if err != nil {
		return none, err
	}
	o, err := c.doJSON(ctx, "GET", voiceProduct, dialingCountryPath(req.Input.IsoCode), nil, nil)
	if err != nil {
		return none, lookupFailure(fn, "dialing permission for country "+req.Input.IsoCode, err)
	}
	return infer.FunctionResponse[DialingCountry]{Output: dialingCountry(o)}, nil
}

// GetDialingPermissionsCountries lists the Voice dialing permissions of the countries.
type GetDialingPermissionsCountries struct{}

// GetDialingPermissionsCountriesArgs are the lookup arguments; every one narrows the list.
type GetDialingPermissionsCountriesArgs struct {
	IsoCode                         *string `pulumi:"isoCode,optional"`
	Continent                       *string `pulumi:"continent,optional"`
	CountryCode                     *string `pulumi:"countryCode,optional"`
	LowRiskNumbersEnabled           *bool   `pulumi:"lowRiskNumbersEnabled,optional"`
	HighRiskSpecialNumbersEnabled   *bool   `pulumi:"highRiskSpecialNumbersEnabled,optional"`
	HighRiskTollfraudNumbersEnabled *bool   `pulumi:"highRiskTollfraudNumbersEnabled,optional"`
}

func (a *GetDialingPermissionsCountriesArgs) Annotate(an infer.Annotator) {
	an.Describe(&a.IsoCode, "Only this ISO 3166-1 alpha-2 country code.")
	an.Describe(&a.Continent, "Only countries on this continent.")
	an.Describe(&a.CountryCode, "Only countries with this E.164 calling code.")
	an.Describe(&a.LowRiskNumbersEnabled, "Only countries where dialing low-risk numbers is (or is not) enabled.")
	an.Describe(&a.HighRiskSpecialNumbersEnabled, "Only countries where dialing high-risk special services numbers is (or is not) enabled.")
	an.Describe(&a.HighRiskTollfraudNumbersEnabled, "Only countries where dialing high-risk toll fraud numbers is (or is not) enabled.")
}

// GetDialingPermissionsCountriesResult is the list.
type GetDialingPermissionsCountriesResult struct {
	Countries []DialingCountry `pulumi:"countries"`
}

func (a *GetDialingPermissionsCountriesResult) Annotate(an infer.Annotator) {
	an.Describe(&a.Countries, "The matching countries, in the order Twilio returns them.")
}

func (f *GetDialingPermissionsCountries) Annotate(a infer.Annotator) {
	a.Describe(&f, "List the Voice dialing permissions per country, optionally narrowed by country, continent, calling code or permission (GET /v1/DialingPermissions/Countries on voice.twilio.com). Read-only.")
}

func (f *GetDialingPermissionsCountries) Invoke(ctx context.Context, req infer.FunctionRequest[GetDialingPermissionsCountriesArgs]) (infer.FunctionResponse[GetDialingPermissionsCountriesResult], error) {
	const fn = "getDialingPermissionsCountries"
	none := infer.FunctionResponse[GetDialingPermissionsCountriesResult]{}
	c, err := clientFor(ctx)
	if err != nil {
		return none, err
	}
	in := req.Input
	q := url.Values{"PageSize": {"1000"}}
	for k, v := range map[string]*string{"IsoCode": in.IsoCode, "Continent": in.Continent, "CountryCode": in.CountryCode} {
		if v != nil && *v != "" {
			q.Set(k, *v)
		}
	}
	for k, v := range map[string]*bool{"LowRiskNumbersEnabled": in.LowRiskNumbersEnabled, "HighRiskSpecialNumbersEnabled": in.HighRiskSpecialNumbersEnabled,
		"HighRiskTollfraudNumbersEnabled": in.HighRiskTollfraudNumbersEnabled} {
		if v != nil {
			q.Set(k, strconv.FormatBool(*v))
		}
	}
	out := []DialingCountry{}
	err = listPages(ctx, c, fn, voiceProduct, dialingCountriesPath(), q, "content", func(o obj) bool {
		out = append(out, dialingCountry(o))
		return true
	})
	if err != nil {
		return none, lookupFailure(fn, "dialing permissions", err)
	}
	return infer.FunctionResponse[GetDialingPermissionsCountriesResult]{Output: GetDialingPermissionsCountriesResult{Countries: out}}, nil
}

// GetDialingPermissionsHighRiskSpecialPrefixes lists the high-risk special services number
// prefixes of a country.
type GetDialingPermissionsHighRiskSpecialPrefixes struct{}

// GetDialingPermissionsHighRiskSpecialPrefixesArgs are the lookup arguments.
type GetDialingPermissionsHighRiskSpecialPrefixesArgs struct {
	IsoCode string `pulumi:"isoCode"`
}

func (a *GetDialingPermissionsHighRiskSpecialPrefixesArgs) Annotate(an infer.Annotator) {
	an.Describe(&a.IsoCode, "The ISO 3166-1 alpha-2 country code, for example `DE`.")
}

// GetDialingPermissionsHighRiskSpecialPrefixesResult is the prefix list.
type GetDialingPermissionsHighRiskSpecialPrefixesResult struct {
	IsoCode  string   `pulumi:"isoCode"`
	Prefixes []string `pulumi:"prefixes"`
}

func (a *GetDialingPermissionsHighRiskSpecialPrefixesResult) Annotate(an infer.Annotator) {
	an.Describe(&a.IsoCode, "The country code that was looked up.")
	an.Describe(&a.Prefixes, "The E.164 number prefixes (with the country code, for example `+49900`) that count as high-risk special services numbers in the country, in the order Twilio returns them.")
}

func (f *GetDialingPermissionsHighRiskSpecialPrefixes) Annotate(a infer.Annotator) {
	a.Describe(&f, "List the number prefixes Twilio treats as high-risk special services numbers in a country (GET /v1/DialingPermissions/Countries/{IsoCode}/HighRiskSpecialPrefixes). Read-only.")
}

func (f *GetDialingPermissionsHighRiskSpecialPrefixes) Invoke(ctx context.Context, req infer.FunctionRequest[GetDialingPermissionsHighRiskSpecialPrefixesArgs]) (infer.FunctionResponse[GetDialingPermissionsHighRiskSpecialPrefixesResult], error) {
	const fn = "getDialingPermissionsHighRiskSpecialPrefixes"
	none := infer.FunctionResponse[GetDialingPermissionsHighRiskSpecialPrefixesResult]{}
	if err := requireArg(fn, "isoCode", req.Input.IsoCode); err != nil {
		return none, err
	}
	c, err := clientFor(ctx)
	if err != nil {
		return none, err
	}
	out := []string{}
	err = listPages(ctx, c, fn, voiceProduct, dialingCountryPath(req.Input.IsoCode)+"/HighRiskSpecialPrefixes", url.Values{"PageSize": {"1000"}}, "content", func(o obj) bool {
		out = append(out, o.str("prefix"))
		return true
	})
	if err != nil {
		return none, lookupFailure(fn, "high-risk special prefixes for country "+req.Input.IsoCode, err)
	}
	return infer.FunctionResponse[GetDialingPermissionsHighRiskSpecialPrefixesResult]{
		Output: GetDialingPermissionsHighRiskSpecialPrefixesResult{IsoCode: req.Input.IsoCode, Prefixes: out}}, nil
}
