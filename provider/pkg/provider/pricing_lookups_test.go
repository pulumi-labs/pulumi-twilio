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
	"testing"

	"github.com/pulumi/pulumi/sdk/v3/go/property"
	"github.com/stretchr/testify/require"

	pgp "github.com/pulumi/pulumi-go-provider"
)

func pv0() map[string]property.Value { return map[string]property.Value{} }

func pricingL(t *testing.T) *tl {
	l := newTL(t, "VoiceRecordingConfiguration", "unused")
	f := l.f
	f.seedPricingCountry("v1/Messaging", "DE", "Germany", map[string]any{
		"outbound_sms_prices": []any{map[string]any{"carrier": "Telekom", "mcc": "262", "mnc": "01",
			"prices": []any{map[string]any{"number_type": "mobile", "base_price": "0.0830", "current_price": "0.0790"},
				map[string]any{"number_type": "local", "base_price": "0.0900", "current_price": "0.0900"}}}},
		"inbound_sms_prices": []any{map[string]any{"number_type": "local", "base_price": "0.0075", "current_price": "0.0070"}}})
	f.seedPricingCountry("v1/Messaging", "US", "United States", map[string]any{"outbound_sms_prices": nil, "inbound_sms_prices": nil})
	f.seedPricingCountry("v1/PhoneNumbers", "GB", "United Kingdom", map[string]any{
		"phone_number_prices": []any{map[string]any{"number_type": "mobile", "base_price": "1.00", "current_price": "1.00"},
			map[string]any{"number_type": "toll free", "base_price": "25.00", "current_price": "22.50"}}})
	f.seedPricingCountry("v2/Voice", "DE", "Germany", map[string]any{
		"outbound_prefix_prices": []any{map[string]any{"friendly_name": "Germany - Mobile", "origination_prefixes": []any{"ALL"},
			"destination_prefixes": []any{"4915", "4916"}, "base_price": "0.2000", "current_price": "0.1900"}},
		"inbound_call_prices": []any{map[string]any{"number_type": "local", "base_price": "0.0100", "current_price": "0.0090"}}})
	f.seedPricingCountry("v2/Trunking", "FR", "France", map[string]any{
		"terminating_prefix_prices": []any{map[string]any{"friendly_name": "France", "origination_prefixes": []any{"33"},
			"destination_prefixes": []any{"33"}, "base_price": "0.0150", "current_price": "0.0140"}},
		"originating_call_prices": []any{map[string]any{"number_type": "toll free", "base_price": "0.0300", "current_price": "0.0300"}}})
	f.seedPricing("v2/Voice/Numbers/+4915123456789", map[string]any{
		"destination_number": "+4915123456789", "origination_number": nil, "country": "Germany", "iso_country": "DE", "price_unit": "usd",
		"outbound_call_prices": []any{map[string]any{"origination_prefixes": []any{"ALL"}, "base_price": "0.2000", "current_price": "0.1900"}},
		"inbound_call_price":   nil, "url": "https://pricing.twilio.com/v2/Voice/Numbers/+4915123456789"})
	f.seedPricing("v2/Voice/Numbers/+14155550100", map[string]any{
		"destination_number": "+14155550100", "origination_number": nil, "country": "United States", "iso_country": "US", "price_unit": "usd",
		"outbound_call_prices": []any{}, "inbound_call_price": map[string]any{"number_type": "local", "base_price": "0.0085", "current_price": "0.0080"}})
	f.seedPricing("v2/Trunking/Numbers/+33142685300", map[string]any{
		"destination_number": "+33142685300", "origination_number": nil, "country": "France", "iso_country": "FR", "price_unit": "usd",
		"terminating_prefix_prices": []any{map[string]any{"friendly_name": "France", "origination_prefixes": []any{"1"}, "destination_prefixes": []any{"331"}, "base_price": "0.0150", "current_price": "0.0140"}},
		"originating_call_price":    map[string]any{"number_type": "local", "base_price": "0.0100", "current_price": "0.0100"}})
	return l
}

func countryIsos(r property.Map) []string {
	var out []string
	for _, c := range r.Get("countries").AsArray().AsSlice() {
		out = append(out, c.AsMap().Get("isoCountry").AsString())
	}
	return out
}

// #1667: Pricing is a read-only catalogue, offered as lookups.
func TestLookup_PricingMessaging(t *testing.T) {
	l := pricingL(t)
	r, err := invoke(t, l, "getPricingMessagingCountry", map[string]property.Value{"isoCountry": sv("DE")})
	require.NoError(t, err)
	require.Equal(t, "DE", getS(t, r, "isoCountry"))
	require.Equal(t, "Germany", getS(t, r, "country"))
	require.Equal(t, "usd", getS(t, r, "priceUnit"))
	out := r.Get("outboundSmsPrices").AsArray()
	require.Equal(t, 1, len(out.AsSlice()))
	c := out.Get(0).AsMap()
	require.Equal(t, "Telekom", c.Get("carrier").AsString())
	require.Equal(t, "262", c.Get("mcc").AsString())
	require.Equal(t, "01", c.Get("mnc").AsString())
	p := c.Get("prices").AsArray()
	require.Equal(t, 2, len(p.AsSlice()))
	require.Equal(t, "mobile", p.Get(0).AsMap().Get("numberType").AsString())
	require.Equal(t, "0.0830", p.Get(0).AsMap().Get("basePrice").AsString(), "prices stay decimal strings")
	require.Equal(t, "0.0790", p.Get(0).AsMap().Get("currentPrice").AsString())
	in := r.Get("inboundSmsPrices").AsArray()
	require.Equal(t, "0.0070", in.Get(0).AsMap().Get("currentPrice").AsString())

	// Null price lists read as empty lists.
	r, err = invoke(t, l, "getPricingMessagingCountry", map[string]property.Value{"isoCountry": sv("US")})
	require.NoError(t, err)
	require.Empty(t, r.Get("outboundSmsPrices").AsArray().AsSlice())
	require.Empty(t, r.Get("inboundSmsPrices").AsArray().AsSlice())

	list, err := invoke(t, l, "getPricingMessagingCountries", pv0())
	require.NoError(t, err)
	require.Equal(t, []string{"DE", "US"}, countryIsos(list))
	require.Equal(t, "Germany", list.Get("countries").AsArray().Get(0).AsMap().Get("country").AsString())

	_, err = invoke(t, l, "getPricingMessagingCountry", map[string]property.Value{"isoCountry": sv("ZZ")})
	require.ErrorContains(t, err, "was not found")
	_, err = invoke(t, l, "getPricingMessagingCountry", map[string]property.Value{"isoCountry": sv("")})
	require.ErrorContains(t, err, "isoCountry must be set")
	require.Len(t, l.f.callsMatching("GET", "/pricing/v1/Messaging/Countries/DE"), 1)
}

func TestLookup_PricingPhoneNumbers(t *testing.T) {
	l := pricingL(t)
	r, err := invoke(t, l, "getPricingPhoneNumberCountry", map[string]property.Value{"isoCountry": sv("GB")})
	require.NoError(t, err)
	require.Equal(t, "United Kingdom", getS(t, r, "country"))
	p := r.Get("phoneNumberPrices").AsArray()
	require.Equal(t, 2, len(p.AsSlice()))
	require.Equal(t, "toll free", p.Get(1).AsMap().Get("numberType").AsString())
	require.Equal(t, "25.00", p.Get(1).AsMap().Get("basePrice").AsString())
	require.Equal(t, "22.50", p.Get(1).AsMap().Get("currentPrice").AsString())

	list, err := invoke(t, l, "getPricingPhoneNumberCountries", pv0())
	require.NoError(t, err)
	require.Equal(t, []string{"GB"}, countryIsos(list))
	require.Len(t, l.f.callsMatching("GET", "/pricing/v1/PhoneNumbers/Countries"), 2)

	_, err = invoke(t, l, "getPricingPhoneNumberCountry", map[string]property.Value{"isoCountry": sv("ZZ")})
	require.ErrorContains(t, err, "was not found")
	_, err = invoke(t, l, "getPricingPhoneNumberCountry", map[string]property.Value{"isoCountry": sv("")})
	require.ErrorContains(t, err, "isoCountry must be set")
}

func TestLookup_PricingVoice(t *testing.T) {
	l := pricingL(t)
	r, err := invoke(t, l, "getPricingVoiceCountry", map[string]property.Value{"isoCountry": sv("DE")})
	require.NoError(t, err)
	require.Equal(t, "Germany", getS(t, r, "country"))
	o := r.Get("outboundPrefixPrices").AsArray().Get(0).AsMap()
	require.Equal(t, "Germany - Mobile", o.Get("friendlyName").AsString())
	require.Equal(t, "ALL", o.Get("originationPrefixes").AsArray().Get(0).AsString())
	require.Equal(t, "4916", o.Get("destinationPrefixes").AsArray().Get(1).AsString())
	require.Equal(t, "0.1900", o.Get("currentPrice").AsString())
	require.Equal(t, "0.0090", r.Get("inboundCallPrices").AsArray().Get(0).AsMap().Get("currentPrice").AsString())

	list, err := invoke(t, l, "getPricingVoiceCountries", pv0())
	require.NoError(t, err)
	require.Equal(t, []string{"DE"}, countryIsos(list))
	_, err = invoke(t, l, "getPricingVoiceCountry", map[string]property.Value{"isoCountry": sv("ZZ")})
	require.ErrorContains(t, err, "was not found")
	_, err = invoke(t, l, "getPricingVoiceCountry", map[string]property.Value{"isoCountry": sv("")})
	require.ErrorContains(t, err, "isoCountry must be set")

	// A number that is not yours has no inbound price: the output leaves it unset.
	n, err := invoke(t, l, "getPricingVoiceNumber", map[string]property.Value{"destinationNumber": sv("+4915123456789")})
	require.NoError(t, err)
	require.Equal(t, "+4915123456789", getS(t, n, "destinationNumber"))
	require.Equal(t, "", getS(t, n, "originationNumber"))
	require.Equal(t, "DE", getS(t, n, "isoCountry"))
	require.Equal(t, "0.1900", n.Get("outboundCallPrices").AsArray().Get(0).AsMap().Get("currentPrice").AsString())
	require.False(t, hasProp(n, "inboundCallPrice"))
	calls := l.f.callsMatching("GET", "/pricing/v2/Voice/Numbers/")
	require.Len(t, calls, 1)
	require.Empty(t, calls[0].Query.Get("OriginationNumber"), "no origination query unless asked")

	// The origination number goes out as the OriginationNumber query parameter.
	l.f.resetCalls()
	n, err = invoke(t, l, "getPricingVoiceNumber", map[string]property.Value{"destinationNumber": sv("+14155550100"), "originationNumber": sv("+4930123")})
	require.NoError(t, err)
	require.Equal(t, "+4930123", getS(t, n, "originationNumber"))
	require.Equal(t, "local", n.Get("inboundCallPrice").AsMap().Get("numberType").AsString())
	require.Equal(t, "0.0080", n.Get("inboundCallPrice").AsMap().Get("currentPrice").AsString())
	require.Empty(t, n.Get("outboundCallPrices").AsArray().AsSlice())
	calls = l.f.callsMatching("GET", "/pricing/v2/Voice/Numbers/")
	require.Len(t, calls, 1)
	require.Equal(t, "+4930123", calls[0].Query.Get("OriginationNumber"))

	_, err = invoke(t, l, "getPricingVoiceNumber", map[string]property.Value{"destinationNumber": sv("+1000")})
	require.ErrorContains(t, err, "was not found")
	_, err = invoke(t, l, "getPricingVoiceNumber", map[string]property.Value{"destinationNumber": sv("")})
	require.ErrorContains(t, err, "destinationNumber must be set")
}

func TestLookup_PricingTrunking(t *testing.T) {
	l := pricingL(t)
	r, err := invoke(t, l, "getPricingTrunkingCountry", map[string]property.Value{"isoCountry": sv("FR")})
	require.NoError(t, err)
	require.Equal(t, "France", getS(t, r, "country"))
	tp := r.Get("terminatingPrefixPrices").AsArray().Get(0).AsMap()
	require.Equal(t, "33", tp.Get("originationPrefixes").AsArray().Get(0).AsString())
	require.Equal(t, "0.0140", tp.Get("currentPrice").AsString())
	require.Equal(t, "toll free", r.Get("originatingCallPrices").AsArray().Get(0).AsMap().Get("numberType").AsString())

	list, err := invoke(t, l, "getPricingTrunkingCountries", pv0())
	require.NoError(t, err)
	require.Equal(t, []string{"FR"}, countryIsos(list))
	_, err = invoke(t, l, "getPricingTrunkingCountry", map[string]property.Value{"isoCountry": sv("ZZ")})
	require.ErrorContains(t, err, "was not found")
	_, err = invoke(t, l, "getPricingTrunkingCountry", map[string]property.Value{"isoCountry": sv("")})
	require.ErrorContains(t, err, "isoCountry must be set")

	l.f.resetCalls()
	n, err := invoke(t, l, "getPricingTrunkingNumber", map[string]property.Value{"destinationNumber": sv("+33142685300"), "originationNumber": sv("+1415")})
	require.NoError(t, err)
	require.Equal(t, "+1415", getS(t, n, "originationNumber"))
	require.Equal(t, "FR", getS(t, n, "isoCountry"))
	require.Equal(t, "331", n.Get("terminatingPrefixPrices").AsArray().Get(0).AsMap().Get("destinationPrefixes").AsArray().Get(0).AsString())
	require.Equal(t, "0.0100", n.Get("originatingCallPrice").AsMap().Get("currentPrice").AsString())
	calls := l.f.callsMatching("GET", "/pricing/v2/Trunking/Numbers/")
	require.Len(t, calls, 1)
	require.Equal(t, "+1415", calls[0].Query.Get("OriginationNumber"))

	_, err = invoke(t, l, "getPricingTrunkingNumber", map[string]property.Value{"destinationNumber": sv("+1000")})
	require.ErrorContains(t, err, "was not found")
	_, err = invoke(t, l, "getPricingTrunkingNumber", map[string]property.Value{"destinationNumber": sv("")})
	require.ErrorContains(t, err, "destinationNumber must be set")
}

// Country lists spanning several pages are read to the end through meta.next_page_url.
func TestLookup_PricingCountriesPaginate(t *testing.T) {
	l := newTL(t, "VoiceRecordingConfiguration", "unused")
	for _, iso := range []string{"AA", "AB", "AC", "AD", "AE"} {
		l.f.seedPricingCountry("v2/Voice", iso, "Country "+iso, nil)
	}
	l.f.pricingMaxPage = 2
	all, err := invoke(t, l, "getPricingVoiceCountries", pv0())
	require.NoError(t, err)
	require.Equal(t, []string{"AA", "AB", "AC", "AD", "AE"}, countryIsos(all))
	calls := l.f.callsMatching("GET", "/pricing/v2/Voice/Countries")
	require.Len(t, calls, 3, "three pages of two")
	require.Equal(t, "1000", calls[0].Query.Get("PageSize"), "the first request asks for the largest page")

	// A list that is empty is an empty list, not an error.
	empty, err := invoke(t, l, "getPricingTrunkingCountries", pv0())
	require.NoError(t, err)
	require.Empty(t, empty.Get("countries").AsArray().AsSlice())
}

func TestPricingLookups_SchemaTokens(t *testing.T) {
	prov, err := New()
	require.NoError(t, err)
	spec, err := pgp.GetSchema(t.Context(), "twilio", "0.1.0", prov)
	require.NoError(t, err)
	for _, fn := range []string{"getPricingMessagingCountries", "getPricingMessagingCountry", "getPricingPhoneNumberCountries", "getPricingPhoneNumberCountry",
		"getPricingVoiceCountries", "getPricingVoiceCountry", "getPricingVoiceNumber", "getPricingTrunkingCountries", "getPricingTrunkingCountry", "getPricingTrunkingNumber"} {
		_, ok := spec.Functions["twilio:index:"+fn]
		require.True(t, ok, fn)
	}
}
