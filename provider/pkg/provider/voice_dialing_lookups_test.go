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

func seedDialing(l *tl) {
	l.f.seedDialingCountry("DE", "Germany", "Europe", []any{"49"}, true, false, false, "+49900", "+49137")
	l.f.seedDialingCountry("US", "United States", "North America", []any{"1"}, true, true, false)
	l.f.seedDialingCountry("CA", "Canada", "North America", []any{float64(1)}, true, true, true)
	l.f.seedDialingCountry("GB", "United Kingdom", "Europe", []any{"44"}, true, false, true, "+44909")
}

// #1665: the dialing permissions are readable per country, in lists and as prefix lists.
func TestLookup_DialingPermissionsCountry(t *testing.T) {
	l := newTL(t, "VoiceRecordingConfiguration", "unused")
	seedDialing(l)
	r, err := invoke(t, l, "getDialingPermissionsCountry", map[string]property.Value{"isoCode": sv("DE")})
	require.NoError(t, err)
	require.Equal(t, "DE", getS(t, r, "isoCode"))
	require.Equal(t, "Germany", getS(t, r, "name"))
	require.Equal(t, "Europe", getS(t, r, "continent"))
	require.Equal(t, true, r.Get("lowRiskNumbersEnabled").AsBool())
	require.Equal(t, false, r.Get("highRiskSpecialNumbersEnabled").AsBool())
	require.Equal(t, "49", r.Get("countryCodes").AsArray().Get(0).AsString())

	// A numeric calling code is rendered as text.
	r, err = invoke(t, l, "getDialingPermissionsCountry", map[string]property.Value{"isoCode": sv("CA")})
	require.NoError(t, err)
	require.Equal(t, "1", r.Get("countryCodes").AsArray().Get(0).AsString())
	require.Equal(t, true, r.Get("highRiskTollfraudNumbersEnabled").AsBool())

	_, err = invoke(t, l, "getDialingPermissionsCountry", map[string]property.Value{"isoCode": sv("ZZ")})
	require.ErrorContains(t, err, "was not found")
	_, err = invoke(t, l, "getDialingPermissionsCountry", map[string]property.Value{"isoCode": sv("")})
	require.ErrorContains(t, err, "isoCode must be set")
}

func TestLookup_DialingPermissionsCountries(t *testing.T) {
	l := newTL(t, "VoiceRecordingConfiguration", "unused")
	seedDialing(l)
	isos := func(r property.Map) []string {
		var out []string
		for _, c := range r.Get("countries").AsArray().AsSlice() {
			out = append(out, c.AsMap().Get("isoCode").AsString())
		}
		return out
	}
	all, err := invoke(t, l, "getDialingPermissionsCountries", map[string]property.Value{})
	require.NoError(t, err)
	require.Equal(t, []string{"DE", "US", "CA", "GB"}, isos(all))

	eu, err := invoke(t, l, "getDialingPermissionsCountries", map[string]property.Value{"continent": sv("Europe")})
	require.NoError(t, err)
	require.Equal(t, []string{"DE", "GB"}, isos(eu))

	noToll, err := invoke(t, l, "getDialingPermissionsCountries", map[string]property.Value{"highRiskTollfraudNumbersEnabled": property.New(false)})
	require.NoError(t, err)
	require.Equal(t, []string{"DE", "US"}, isos(noToll))

	special, err := invoke(t, l, "getDialingPermissionsCountries", map[string]property.Value{"highRiskSpecialNumbersEnabled": property.New(true), "countryCode": sv("1")})
	require.NoError(t, err)
	require.Equal(t, []string{"US", "CA"}, isos(special))

	// The filters go out under the names of the API spec.
	l.f.resetCalls()
	_, err = invoke(t, l, "getDialingPermissionsCountries", map[string]property.Value{
		"isoCode": sv("DE"), "continent": sv("Europe"), "countryCode": sv("49"),
		"lowRiskNumbersEnabled": property.New(true), "highRiskSpecialNumbersEnabled": property.New(false), "highRiskTollfraudNumbersEnabled": property.New(false)})
	require.NoError(t, err)
	calls := l.f.callsMatching("GET", "/DialingPermissions/Countries")
	require.Len(t, calls, 1)
	require.Equal(t, "DE", calls[0].Query.Get("IsoCode"))
	require.Equal(t, "Europe", calls[0].Query.Get("Continent"))
	require.Equal(t, "49", calls[0].Query.Get("CountryCode"))
	require.Equal(t, "true", calls[0].Query.Get("LowRiskNumbersEnabled"))
	require.Equal(t, "false", calls[0].Query.Get("HighRiskSpecialNumbersEnabled"))
	require.Equal(t, "false", calls[0].Query.Get("HighRiskTollfraudNumbersEnabled"))

	none, err := invoke(t, l, "getDialingPermissionsCountries", map[string]property.Value{"isoCode": sv("ZZ")})
	require.NoError(t, err)
	require.Empty(t, isos(none))
}

// A list that spans several pages is read to the end through meta.next_page_url, and a
// prefix list for a country with none is empty.
func TestLookup_DialingPermissionsPagesAndPrefixes(t *testing.T) {
	l := newTL(t, "VoiceRecordingConfiguration", "unused")
	for _, iso := range []string{"AA", "AB", "AC", "AD", "AE"} {
		l.f.seedDialingCountry(iso, "Country "+iso, "Europe", []any{"9"}, true, false, false)
	}
	l.f.seedDialingCountry("XX", "Xland", "Europe", []any{"7"}, true, true, false, "+7001", "+7002", "+7003")
	l.f.dialingMaxPage = 2

	all, err := invoke(t, l, "getDialingPermissionsCountries", map[string]property.Value{})
	require.NoError(t, err)
	require.Len(t, all.Get("countries").AsArray().AsSlice(), 6, "every page is read")
	require.Len(t, l.f.callsMatching("GET", "/DialingPermissions/Countries"), 3, "three pages of two")

	r, err := invoke(t, l, "getDialingPermissionsHighRiskSpecialPrefixes", map[string]property.Value{"isoCode": sv("XX")})
	require.NoError(t, err)
	require.Equal(t, "XX", getS(t, r, "isoCode"))
	require.Equal(t, 3, len(r.Get("prefixes").AsArray().AsSlice()))
	require.Equal(t, "+7002", r.Get("prefixes").AsArray().Get(1).AsString())

	empty, err := invoke(t, l, "getDialingPermissionsHighRiskSpecialPrefixes", map[string]property.Value{"isoCode": sv("AA")})
	require.NoError(t, err)
	require.Empty(t, empty.Get("prefixes").AsArray().AsSlice())
	_, err = invoke(t, l, "getDialingPermissionsHighRiskSpecialPrefixes", map[string]property.Value{"isoCode": sv("ZZ")})
	require.ErrorContains(t, err, "was not found")
}

func TestDialingPermissions_SchemaTokens(t *testing.T) {
	prov, err := New()
	require.NoError(t, err)
	spec, err := pgp.GetSchema(t.Context(), "twilio", "0.1.0", prov)
	require.NoError(t, err)
	for _, fn := range []string{"getDialingPermissionsCountry", "getDialingPermissionsCountries", "getDialingPermissionsHighRiskSpecialPrefixes"} {
		_, ok := spec.Functions["twilio:index:"+fn]
		require.True(t, ok, fn)
	}
}
