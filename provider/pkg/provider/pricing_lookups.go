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

// Twilio Pricing (pricing.twilio.com) is a read-only price catalogue: per-country price lists for
// Messaging, Phone Numbers, Voice and Elastic SIP Trunking, and per-number Voice and Trunking
// prices. There is nothing to provision, so these are lookups only. Prices are returned as the
// decimal strings Twilio sends (the API spec types them as strings) so no precision is lost.
// The Voice v1 pricing endpoints (/v1/Voice/...) are superseded by /v2/Voice (which adds the
// origination prefixes) and are not offered.

const pricingProduct = "pricing"

func pricingPath(version, family string, rest ...string) string {
	p := "/" + version + "/" + family
	for _, r := range rest {
		p += "/" + url.PathEscape(r)
	}
	return p
}

// PricingPrice is one base and current price, per number type where Twilio splits it.
type PricingPrice struct {
	NumberType   string `pulumi:"numberType"`
	BasePrice    string `pulumi:"basePrice"`
	CurrentPrice string `pulumi:"currentPrice"`
}

func (a *PricingPrice) Annotate(an infer.Annotator) {
	an.Describe(&a.NumberType, "The kind of Twilio number the price applies to, for example `local`, `mobile`, `national` or `toll free`; empty when the price is not split by number type.")
	an.Describe(&a.BasePrice, "The list price per unit as a decimal string, in the response's price unit.")
	an.Describe(&a.CurrentPrice, "The price per unit that applies to your account (list price with your discounts) as a decimal string.")
}

// PricingPrefixPrice is the price for a set of number prefixes.
type PricingPrefixPrice struct {
	FriendlyName        string   `pulumi:"friendlyName"`
	OriginationPrefixes []string `pulumi:"originationPrefixes"`
	DestinationPrefixes []string `pulumi:"destinationPrefixes"`
	BasePrice           string   `pulumi:"basePrice"`
	CurrentPrice        string   `pulumi:"currentPrice"`
}

func (a *PricingPrefixPrice) Annotate(an infer.Annotator) {
	an.Describe(&a.FriendlyName, "The name of the destination the prefixes belong to, for example `Germany - Mobile`.")
	an.Describe(&a.OriginationPrefixes, "The calling-number prefixes the price applies to; `ALL` means any origination.")
	an.Describe(&a.DestinationPrefixes, "The destination number prefixes the price applies to.")
	an.Describe(&a.BasePrice, "The list price per minute as a decimal string, in the response's price unit.")
	an.Describe(&a.CurrentPrice, "The price per minute that applies to your account as a decimal string.")
}

// PricingSmsPrice is the price of sending an SMS through one carrier.
type PricingSmsPrice struct {
	Carrier string         `pulumi:"carrier"`
	Mcc     string         `pulumi:"mcc"`
	Mnc     string         `pulumi:"mnc"`
	Prices  []PricingPrice `pulumi:"prices"`
}

func (a *PricingSmsPrice) Annotate(an infer.Annotator) {
	an.Describe(&a.Carrier, "The name of the carrier.")
	an.Describe(&a.Mcc, "The mobile country code of the carrier.")
	an.Describe(&a.Mnc, "The mobile network code of the carrier.")
	an.Describe(&a.Prices, "The price per message for each number type used to send.")
}

// PricingOutboundCallPrice is the price of calling a destination number from an origination prefix.
type PricingOutboundCallPrice struct {
	OriginationPrefixes []string `pulumi:"originationPrefixes"`
	BasePrice           string   `pulumi:"basePrice"`
	CurrentPrice        string   `pulumi:"currentPrice"`
}

func (a *PricingOutboundCallPrice) Annotate(an infer.Annotator) {
	an.Describe(&a.OriginationPrefixes, "The calling-number prefixes the price applies to; `ALL` means any origination.")
	an.Describe(&a.BasePrice, "The list price per minute as a decimal string.")
	an.Describe(&a.CurrentPrice, "The price per minute that applies to your account as a decimal string.")
}

// PricingCountrySummary is one entry of a country list.
type PricingCountrySummary struct {
	IsoCountry string `pulumi:"isoCountry"`
	Country    string `pulumi:"country"`
}

func (a *PricingCountrySummary) Annotate(an infer.Annotator) {
	an.Describe(&a.IsoCountry, "The ISO 3166-1 alpha-2 country code.")
	an.Describe(&a.Country, "The name of the country.")
}

// PricingCountriesResult is the result of every country list lookup.
type PricingCountriesResult struct {
	Countries []PricingCountrySummary `pulumi:"countries"`
}

func (a *PricingCountriesResult) Annotate(an infer.Annotator) {
	an.Describe(&a.Countries, "The countries Twilio publishes prices for, in the order Twilio returns them (every page is read). Look one up with the matching single-country function for its prices.")
}

// PricingCountryArgs is the argument of every single-country price lookup.
type PricingCountryArgs struct {
	IsoCountry string `pulumi:"isoCountry"`
}

func (a *PricingCountryArgs) Annotate(an infer.Annotator) {
	an.Describe(&a.IsoCountry, "The ISO 3166-1 alpha-2 country code, for example `DE`.")
}

// PricingNumberArgs is the argument of the per-number price lookups.
type PricingNumberArgs struct {
	DestinationNumber string  `pulumi:"destinationNumber"`
	OriginationNumber *string `pulumi:"originationNumber,optional"`
}

func (a *PricingNumberArgs) Annotate(an infer.Annotator) {
	an.Describe(&a.DestinationNumber, "The destination phone number in E.164 format, for example `+14155550100`.")
	an.Describe(&a.OriginationNumber, "The calling phone number in E.164 format; when set, Twilio returns the price for that origination only. Unset returns the prices for every origination.")
}

func priceOf(o obj) PricingPrice {
	return PricingPrice{NumberType: o.str("number_type"), BasePrice: o.str("base_price"), CurrentPrice: o.str("current_price")}
}

func objList(o obj, k string) []obj {
	out := []obj{}
	l, _ := o.get(k).([]any)
	for _, it := range l {
		if m, ok := it.(map[string]any); ok {
			out = append(out, obj(m))
		}
	}
	return out
}

func pricingStrings(o obj, k string) []string {
	if l := o.optList(k); l != nil {
		return *l
	}
	return []string{}
}

func prices(o obj, k string) []PricingPrice {
	out := []PricingPrice{}
	for _, p := range objList(o, k) {
		out = append(out, priceOf(p))
	}
	return out
}

// optPrice reads a nullable single price; nil when Twilio sends null (for example an inbound
// price for a number that is not yours).
func optPrice(o obj, k string) *PricingPrice {
	m, ok := o.get(k).(map[string]any)
	if !ok {
		return nil
	}
	p := priceOf(obj(m))
	return &p
}

func prefixPrices(o obj, k string) []PricingPrefixPrice {
	out := []PricingPrefixPrice{}
	for _, p := range objList(o, k) {
		out = append(out, PricingPrefixPrice{
			FriendlyName: p.str("friendly_name"), OriginationPrefixes: pricingStrings(p, "origination_prefixes"),
			DestinationPrefixes: pricingStrings(p, "destination_prefixes"), BasePrice: p.str("base_price"), CurrentPrice: p.str("current_price"),
		})
	}
	return out
}

// pricingCountryList reads every page of a country list.
func pricingCountryList(ctx context.Context, fn, what, path string) (infer.FunctionResponse[PricingCountriesResult], error) {
	none := infer.FunctionResponse[PricingCountriesResult]{}
	c, err := clientFor(ctx)
	if err != nil {
		return none, err
	}
	out := []PricingCountrySummary{}
	err = listPages(ctx, c, fn, pricingProduct, path, url.Values{"PageSize": {"1000"}}, "countries", func(o obj) bool {
		out = append(out, PricingCountrySummary{IsoCountry: o.str("iso_country"), Country: o.str("country")})
		return true
	})
	if err != nil {
		return none, lookupFailure(fn, what, err)
	}
	return infer.FunctionResponse[PricingCountriesResult]{Output: PricingCountriesResult{Countries: out}}, nil
}

// pricingGet reads one price object; query may be nil.
func pricingGet(ctx context.Context, fn, what, path string, query url.Values) (obj, error) {
	c, err := clientFor(ctx)
	if err != nil {
		return nil, err
	}
	o, err := c.do(ctx, "GET", pricingProduct, path, query, nil)
	if err != nil {
		return nil, lookupFailure(fn, what, err)
	}
	return o, nil
}

func pricingNumberQuery(a PricingNumberArgs) url.Values {
	if a.OriginationNumber != nil && *a.OriginationNumber != "" {
		return url.Values{"OriginationNumber": {*a.OriginationNumber}}
	}
	return nil
}

// ---- Messaging ----

// GetPricingMessagingCountries lists the countries with Messaging prices.
type GetPricingMessagingCountries struct{}

func (f *GetPricingMessagingCountries) Annotate(a infer.Annotator) {
	a.Describe(&f, "List the countries Twilio publishes Messaging (SMS) prices for (GET /v1/Messaging/Countries on pricing.twilio.com). Read-only; every page is read.")
}

func (f *GetPricingMessagingCountries) Invoke(ctx context.Context, _ infer.FunctionRequest[struct{}]) (infer.FunctionResponse[PricingCountriesResult], error) {
	return pricingCountryList(ctx, "getPricingMessagingCountries", "messaging pricing countries", pricingPath("v1", "Messaging", "Countries"))
}

// GetPricingMessagingCountry looks up the Messaging prices of one country.
type GetPricingMessagingCountry struct{}

// PricingMessagingCountry is the Messaging price list of a country.
type PricingMessagingCountry struct {
	IsoCountry        string            `pulumi:"isoCountry"`
	Country           string            `pulumi:"country"`
	PriceUnit         string            `pulumi:"priceUnit"`
	OutboundSmsPrices []PricingSmsPrice `pulumi:"outboundSmsPrices"`
	InboundSmsPrices  []PricingPrice    `pulumi:"inboundSmsPrices"`
}

func (a *PricingMessagingCountry) Annotate(an infer.Annotator) {
	an.Describe(&a.IsoCountry, "The ISO 3166-1 alpha-2 country code.")
	an.Describe(&a.Country, "The name of the country.")
	an.Describe(&a.PriceUnit, "The currency of every price, an ISO 4217 code such as `USD`.")
	an.Describe(&a.OutboundSmsPrices, "The price to send a message per carrier (MCC/MNC) and number type.")
	an.Describe(&a.InboundSmsPrices, "The price to receive a message per Twilio number type.")
}

func (f *GetPricingMessagingCountry) Annotate(a infer.Annotator) {
	a.Describe(&f, "Look up the Messaging (SMS) prices of one country: outbound per carrier and number type, inbound per number type (GET /v1/Messaging/Countries/{IsoCountry} on pricing.twilio.com). Read-only.")
}

func (f *GetPricingMessagingCountry) Invoke(ctx context.Context, req infer.FunctionRequest[PricingCountryArgs]) (infer.FunctionResponse[PricingMessagingCountry], error) {
	const fn = "getPricingMessagingCountry"
	none := infer.FunctionResponse[PricingMessagingCountry]{}
	if err := requireArg(fn, "isoCountry", req.Input.IsoCountry); err != nil {
		return none, err
	}
	o, err := pricingGet(ctx, fn, "messaging prices for country "+req.Input.IsoCountry, pricingPath("v1", "Messaging", "Countries", req.Input.IsoCountry), nil)
	if err != nil {
		return none, err
	}
	out := []PricingSmsPrice{}
	for _, p := range objList(o, "outbound_sms_prices") {
		out = append(out, PricingSmsPrice{Carrier: p.str("carrier"), Mcc: p.str("mcc"), Mnc: p.str("mnc"), Prices: prices(p, "prices")})
	}
	return infer.FunctionResponse[PricingMessagingCountry]{Output: PricingMessagingCountry{
		IsoCountry: o.str("iso_country"), Country: o.str("country"), PriceUnit: o.str("price_unit"),
		OutboundSmsPrices: out, InboundSmsPrices: prices(o, "inbound_sms_prices"),
	}}, nil
}

// ---- Phone Numbers ----

// GetPricingPhoneNumberCountries lists the countries with phone number prices.
type GetPricingPhoneNumberCountries struct{}

func (f *GetPricingPhoneNumberCountries) Annotate(a infer.Annotator) {
	a.Describe(&f, "List the countries Twilio publishes phone number prices for (GET /v1/PhoneNumbers/Countries on pricing.twilio.com). Read-only; every page is read.")
}

func (f *GetPricingPhoneNumberCountries) Invoke(ctx context.Context, _ infer.FunctionRequest[struct{}]) (infer.FunctionResponse[PricingCountriesResult], error) {
	return pricingCountryList(ctx, "getPricingPhoneNumberCountries", "phone number pricing countries", pricingPath("v1", "PhoneNumbers", "Countries"))
}

// GetPricingPhoneNumberCountry looks up the phone number prices of one country.
type GetPricingPhoneNumberCountry struct{}

// PricingPhoneNumberCountry is the phone number price list of a country.
type PricingPhoneNumberCountry struct {
	IsoCountry        string         `pulumi:"isoCountry"`
	Country           string         `pulumi:"country"`
	PriceUnit         string         `pulumi:"priceUnit"`
	PhoneNumberPrices []PricingPrice `pulumi:"phoneNumberPrices"`
}

func (a *PricingPhoneNumberCountry) Annotate(an infer.Annotator) {
	an.Describe(&a.IsoCountry, "The ISO 3166-1 alpha-2 country code.")
	an.Describe(&a.Country, "The name of the country.")
	an.Describe(&a.PriceUnit, "The currency of every price, an ISO 4217 code such as `USD`.")
	an.Describe(&a.PhoneNumberPrices, "The monthly price of a phone number of each type (`local`, `mobile`, `national`, `toll free`).")
}

func (f *GetPricingPhoneNumberCountry) Annotate(a infer.Annotator) {
	a.Describe(&f, "Look up the monthly phone number prices of one country per number type (GET /v1/PhoneNumbers/Countries/{IsoCountry} on pricing.twilio.com). Read-only.")
}

func (f *GetPricingPhoneNumberCountry) Invoke(ctx context.Context, req infer.FunctionRequest[PricingCountryArgs]) (infer.FunctionResponse[PricingPhoneNumberCountry], error) {
	const fn = "getPricingPhoneNumberCountry"
	none := infer.FunctionResponse[PricingPhoneNumberCountry]{}
	if err := requireArg(fn, "isoCountry", req.Input.IsoCountry); err != nil {
		return none, err
	}
	o, err := pricingGet(ctx, fn, "phone number prices for country "+req.Input.IsoCountry, pricingPath("v1", "PhoneNumbers", "Countries", req.Input.IsoCountry), nil)
	if err != nil {
		return none, err
	}
	return infer.FunctionResponse[PricingPhoneNumberCountry]{Output: PricingPhoneNumberCountry{
		IsoCountry: o.str("iso_country"), Country: o.str("country"), PriceUnit: o.str("price_unit"),
		PhoneNumberPrices: prices(o, "phone_number_prices"),
	}}, nil
}

// ---- Voice (v2) ----

// GetPricingVoiceCountries lists the countries with Voice prices.
type GetPricingVoiceCountries struct{}

func (f *GetPricingVoiceCountries) Annotate(a infer.Annotator) {
	a.Describe(&f, "List the countries Twilio publishes Voice prices for (GET /v2/Voice/Countries on pricing.twilio.com). Read-only; every page is read.")
}

func (f *GetPricingVoiceCountries) Invoke(ctx context.Context, _ infer.FunctionRequest[struct{}]) (infer.FunctionResponse[PricingCountriesResult], error) {
	return pricingCountryList(ctx, "getPricingVoiceCountries", "voice pricing countries", pricingPath("v2", "Voice", "Countries"))
}

// GetPricingVoiceCountry looks up the Voice prices of one country.
type GetPricingVoiceCountry struct{}

// PricingVoiceCountry is the Voice price list of a country.
type PricingVoiceCountry struct {
	IsoCountry           string               `pulumi:"isoCountry"`
	Country              string               `pulumi:"country"`
	PriceUnit            string               `pulumi:"priceUnit"`
	OutboundPrefixPrices []PricingPrefixPrice `pulumi:"outboundPrefixPrices"`
	InboundCallPrices    []PricingPrice       `pulumi:"inboundCallPrices"`
}

func (a *PricingVoiceCountry) Annotate(an infer.Annotator) {
	an.Describe(&a.IsoCountry, "The ISO 3166-1 alpha-2 country code.")
	an.Describe(&a.Country, "The name of the country.")
	an.Describe(&a.PriceUnit, "The currency of every price, an ISO 4217 code such as `USD`.")
	an.Describe(&a.OutboundPrefixPrices, "The per-minute price to call each group of destination prefixes, by origination prefix.")
	an.Describe(&a.InboundCallPrices, "The per-minute price to receive a call on each Twilio number type.")
}

func (f *GetPricingVoiceCountry) Annotate(a infer.Annotator) {
	a.Describe(&f, "Look up the Voice prices of one country: outbound per destination prefix and origination, inbound per number type (GET /v2/Voice/Countries/{IsoCountry} on pricing.twilio.com). Read-only.")
}

func (f *GetPricingVoiceCountry) Invoke(ctx context.Context, req infer.FunctionRequest[PricingCountryArgs]) (infer.FunctionResponse[PricingVoiceCountry], error) {
	const fn = "getPricingVoiceCountry"
	none := infer.FunctionResponse[PricingVoiceCountry]{}
	if err := requireArg(fn, "isoCountry", req.Input.IsoCountry); err != nil {
		return none, err
	}
	o, err := pricingGet(ctx, fn, "voice prices for country "+req.Input.IsoCountry, pricingPath("v2", "Voice", "Countries", req.Input.IsoCountry), nil)
	if err != nil {
		return none, err
	}
	return infer.FunctionResponse[PricingVoiceCountry]{Output: PricingVoiceCountry{
		IsoCountry: o.str("iso_country"), Country: o.str("country"), PriceUnit: o.str("price_unit"),
		OutboundPrefixPrices: prefixPrices(o, "outbound_prefix_prices"), InboundCallPrices: prices(o, "inbound_call_prices"),
	}}, nil
}

// GetPricingVoiceNumber looks up the Voice prices of one destination number.
type GetPricingVoiceNumber struct{}

// PricingVoiceNumber is the Voice price of a number.
type PricingVoiceNumber struct {
	DestinationNumber  string                     `pulumi:"destinationNumber"`
	OriginationNumber  string                     `pulumi:"originationNumber"`
	IsoCountry         string                     `pulumi:"isoCountry"`
	Country            string                     `pulumi:"country"`
	PriceUnit          string                     `pulumi:"priceUnit"`
	OutboundCallPrices []PricingOutboundCallPrice `pulumi:"outboundCallPrices"`
	InboundCallPrice   *PricingPrice              `pulumi:"inboundCallPrice,optional"`
}

func (a *PricingVoiceNumber) Annotate(an infer.Annotator) {
	an.Describe(&a.DestinationNumber, "The destination number that was looked up.")
	an.Describe(&a.OriginationNumber, "The origination number that was looked up; empty when none was given.")
	an.Describe(&a.IsoCountry, "The ISO 3166-1 alpha-2 country code of the destination.")
	an.Describe(&a.Country, "The name of the destination country.")
	an.Describe(&a.PriceUnit, "The currency of every price, an ISO 4217 code such as `USD`.")
	an.Describe(&a.OutboundCallPrices, "The per-minute price to call the number, per origination prefix.")
	an.Describe(&a.InboundCallPrice, "The per-minute price to receive a call on the number; unset when the number is not a Twilio number of this account.")
}

func (f *GetPricingVoiceNumber) Annotate(a infer.Annotator) {
	a.Describe(&f, "Look up the Voice prices of one phone number, optionally for one origination number (GET /v2/Voice/Numbers/{DestinationNumber} on pricing.twilio.com). Read-only.")
}

func (f *GetPricingVoiceNumber) Invoke(ctx context.Context, req infer.FunctionRequest[PricingNumberArgs]) (infer.FunctionResponse[PricingVoiceNumber], error) {
	const fn = "getPricingVoiceNumber"
	none := infer.FunctionResponse[PricingVoiceNumber]{}
	in := req.Input
	if err := requireArg(fn, "destinationNumber", in.DestinationNumber); err != nil {
		return none, err
	}
	o, err := pricingGet(ctx, fn, "voice prices for number "+in.DestinationNumber, pricingPath("v2", "Voice", "Numbers", in.DestinationNumber), pricingNumberQuery(in))
	if err != nil {
		return none, err
	}
	out := []PricingOutboundCallPrice{}
	for _, p := range objList(o, "outbound_call_prices") {
		out = append(out, PricingOutboundCallPrice{OriginationPrefixes: pricingStrings(p, "origination_prefixes"), BasePrice: p.str("base_price"), CurrentPrice: p.str("current_price")})
	}
	return infer.FunctionResponse[PricingVoiceNumber]{Output: PricingVoiceNumber{
		DestinationNumber: o.str("destination_number"), OriginationNumber: o.str("origination_number"),
		IsoCountry: o.str("iso_country"), Country: o.str("country"), PriceUnit: o.str("price_unit"),
		OutboundCallPrices: out, InboundCallPrice: optPrice(o, "inbound_call_price"),
	}}, nil
}

// ---- Elastic SIP Trunking ----

// GetPricingTrunkingCountries lists the countries with Elastic SIP Trunking prices.
type GetPricingTrunkingCountries struct{}

func (f *GetPricingTrunkingCountries) Annotate(a infer.Annotator) {
	a.Describe(&f, "List the countries Twilio publishes Elastic SIP Trunking prices for (GET /v2/Trunking/Countries on pricing.twilio.com). Read-only; every page is read.")
}

func (f *GetPricingTrunkingCountries) Invoke(ctx context.Context, _ infer.FunctionRequest[struct{}]) (infer.FunctionResponse[PricingCountriesResult], error) {
	return pricingCountryList(ctx, "getPricingTrunkingCountries", "trunking pricing countries", pricingPath("v2", "Trunking", "Countries"))
}

// GetPricingTrunkingCountry looks up the Elastic SIP Trunking prices of one country.
type GetPricingTrunkingCountry struct{}

// PricingTrunkingCountry is the Elastic SIP Trunking price list of a country.
type PricingTrunkingCountry struct {
	IsoCountry              string               `pulumi:"isoCountry"`
	Country                 string               `pulumi:"country"`
	PriceUnit               string               `pulumi:"priceUnit"`
	TerminatingPrefixPrices []PricingPrefixPrice `pulumi:"terminatingPrefixPrices"`
	OriginatingCallPrices   []PricingPrice       `pulumi:"originatingCallPrices"`
}

func (a *PricingTrunkingCountry) Annotate(an infer.Annotator) {
	an.Describe(&a.IsoCountry, "The ISO 3166-1 alpha-2 country code.")
	an.Describe(&a.Country, "The name of the country.")
	an.Describe(&a.PriceUnit, "The currency of every price, an ISO 4217 code such as `USD`.")
	an.Describe(&a.TerminatingPrefixPrices, "The per-minute price of terminating calls (calls from your trunk to the PSTN) per destination prefix and origination.")
	an.Describe(&a.OriginatingCallPrices, "The per-minute price of originating calls (calls from the PSTN to your trunk) per Twilio number type.")
}

func (f *GetPricingTrunkingCountry) Annotate(a infer.Annotator) {
	a.Describe(&f, "Look up the Elastic SIP Trunking prices of one country: terminating per destination prefix, originating per number type (GET /v2/Trunking/Countries/{IsoCountry} on pricing.twilio.com). Read-only.")
}

func (f *GetPricingTrunkingCountry) Invoke(ctx context.Context, req infer.FunctionRequest[PricingCountryArgs]) (infer.FunctionResponse[PricingTrunkingCountry], error) {
	const fn = "getPricingTrunkingCountry"
	none := infer.FunctionResponse[PricingTrunkingCountry]{}
	if err := requireArg(fn, "isoCountry", req.Input.IsoCountry); err != nil {
		return none, err
	}
	o, err := pricingGet(ctx, fn, "trunking prices for country "+req.Input.IsoCountry, pricingPath("v2", "Trunking", "Countries", req.Input.IsoCountry), nil)
	if err != nil {
		return none, err
	}
	return infer.FunctionResponse[PricingTrunkingCountry]{Output: PricingTrunkingCountry{
		IsoCountry: o.str("iso_country"), Country: o.str("country"), PriceUnit: o.str("price_unit"),
		TerminatingPrefixPrices: prefixPrices(o, "terminating_prefix_prices"), OriginatingCallPrices: prices(o, "originating_call_prices"),
	}}, nil
}

// GetPricingTrunkingNumber looks up the Elastic SIP Trunking prices of one destination number.
type GetPricingTrunkingNumber struct{}

// PricingTrunkingNumber is the Elastic SIP Trunking price of a number.
type PricingTrunkingNumber struct {
	DestinationNumber       string               `pulumi:"destinationNumber"`
	OriginationNumber       string               `pulumi:"originationNumber"`
	IsoCountry              string               `pulumi:"isoCountry"`
	Country                 string               `pulumi:"country"`
	PriceUnit               string               `pulumi:"priceUnit"`
	TerminatingPrefixPrices []PricingPrefixPrice `pulumi:"terminatingPrefixPrices"`
	OriginatingCallPrice    *PricingPrice        `pulumi:"originatingCallPrice,optional"`
}

func (a *PricingTrunkingNumber) Annotate(an infer.Annotator) {
	an.Describe(&a.DestinationNumber, "The destination number that was looked up.")
	an.Describe(&a.OriginationNumber, "The origination number that was looked up; empty when none was given.")
	an.Describe(&a.IsoCountry, "The ISO 3166-1 alpha-2 country code of the destination.")
	an.Describe(&a.Country, "The name of the destination country.")
	an.Describe(&a.PriceUnit, "The currency of every price, an ISO 4217 code such as `USD`.")
	an.Describe(&a.TerminatingPrefixPrices, "The per-minute price of terminating a call to the number, per prefix and origination.")
	an.Describe(&a.OriginatingCallPrice, "The per-minute price of an originating call to the number; unset when Twilio reports none.")
}

func (f *GetPricingTrunkingNumber) Annotate(a infer.Annotator) {
	a.Describe(&f, "Look up the Elastic SIP Trunking prices of one phone number, optionally for one origination number (GET /v2/Trunking/Numbers/{DestinationNumber} on pricing.twilio.com). Read-only.")
}

func (f *GetPricingTrunkingNumber) Invoke(ctx context.Context, req infer.FunctionRequest[PricingNumberArgs]) (infer.FunctionResponse[PricingTrunkingNumber], error) {
	const fn = "getPricingTrunkingNumber"
	none := infer.FunctionResponse[PricingTrunkingNumber]{}
	in := req.Input
	if err := requireArg(fn, "destinationNumber", in.DestinationNumber); err != nil {
		return none, err
	}
	o, err := pricingGet(ctx, fn, "trunking prices for number "+in.DestinationNumber, pricingPath("v2", "Trunking", "Numbers", in.DestinationNumber), pricingNumberQuery(in))
	if err != nil {
		return none, err
	}
	return infer.FunctionResponse[PricingTrunkingNumber]{Output: PricingTrunkingNumber{
		DestinationNumber: o.str("destination_number"), OriginationNumber: o.str("origination_number"),
		IsoCountry: o.str("iso_country"), Country: o.str("country"), PriceUnit: o.str("price_unit"),
		TerminatingPrefixPrices: prefixPrices(o, "terminating_prefix_prices"), OriginatingCallPrice: optPrice(o, "originating_call_price"),
	}}, nil
}
