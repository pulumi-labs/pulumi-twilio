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
)

func listLen(t *testing.T, m property.Map, key string) int {
	t.Helper()
	return m.Get(key).AsArray().Len()
}

func TestLookup_AvailablePhoneNumbers(t *testing.T) {
	l := newTL(t, "Subaccount", "sub")
	r, err := invoke(t, l, "getAvailablePhoneNumbers", map[string]property.Value{"countryCode": sv("US")})
	require.NoError(t, err)
	require.Equal(t, 5, listLen(t, r, "numbers"))
	require.Equal(t, "local", getS(t, r, "numberType"))
	first := r.Get("numbers").AsArray().Get(0).AsMap()
	require.Equal(t, "+14155550100", getS(t, first, "phoneNumber"))
	require.Equal(t, "CA", getS(t, first, "region"))
	require.Equal(t, 37.78, first.Get("latitude").AsNumber(), "a latitude sent as a string is parsed")
	require.Equal(t, -122.4, first.Get("longitude").AsNumber())
	require.True(t, first.Get("capabilities").AsMap().Get("sms").AsBool(), "SMS is read from the upper-case key")
	require.True(t, first.Get("capabilities").AsMap().Get("voice").AsBool())
	require.True(t, r.Get("numbers").AsArray().Get(4).AsMap().Get("beta").AsBool())

	// Filters are sent as Twilio's query parameters, the type picks the path.
	l.f.resetCalls()
	r, err = invoke(t, l, "getAvailablePhoneNumbers", map[string]property.Value{
		"countryCode": sv("US"), "numberType": sv("tollFree"), "smsEnabled": bv(true), "contains": sv("102"), "areaCode": nv(800), "limit": nv(10),
	})
	require.NoError(t, err)
	require.Equal(t, 1, listLen(t, r, "numbers"))
	require.Equal(t, "tollFree", getS(t, r, "numberType"))
	require.Len(t, l.f.callsMatching("GET", "/AvailablePhoneNumbers/US/TollFree.json"), 1)

	// limit stops the walk; paging is followed otherwise.
	r, err = invoke(t, l, "getAvailablePhoneNumbers", map[string]property.Value{"countryCode": sv("GB"), "numberType": sv("mobile"), "limit": nv(2)})
	require.NoError(t, err)
	require.Equal(t, 2, listLen(t, r, "numbers"))
	l.f.resetCalls()
	r, err = invoke(t, l, "getAvailablePhoneNumbers", map[string]property.Value{"countryCode": sv("US"), "limit": nv(3)})
	require.NoError(t, err)
	require.Equal(t, 3, listLen(t, r, "numbers"))

	_, err = invoke(t, l, "getAvailablePhoneNumbers", map[string]property.Value{})
	require.Error(t, err)
	_, err = invoke(t, l, "getAvailablePhoneNumbers", map[string]property.Value{"countryCode": sv("")})
	require.ErrorContains(t, err, "must be set")
	_, err = invoke(t, l, "getAvailablePhoneNumbers", map[string]property.Value{"countryCode": sv("US"), "numberType": sv("voip")})
	require.ErrorContains(t, err, "numberType must be")
	_, err = invoke(t, l, "getAvailablePhoneNumbers", map[string]property.Value{"countryCode": sv("US"), "limit": nv(0)})
	require.ErrorContains(t, err, "limit must be between")
	l.f.failNext("GET", "/AvailablePhoneNumbers/ZZ/Local.json", 404)
	_, err = invoke(t, l, "getAvailablePhoneNumbers", map[string]property.Value{"countryCode": sv("ZZ")})
	require.ErrorContains(t, err, "was not found")
}

func TestLookup_UsageRecords(t *testing.T) {
	l := newTL(t, "Subaccount", "sub")
	r, err := invoke(t, l, "getUsageRecords", map[string]property.Value{})
	require.NoError(t, err)
	require.Equal(t, 3, listLen(t, r, "records"))
	require.Equal(t, "allTime", getS(t, r, "period"))
	rec := r.Get("records").AsArray().Get(0).AsMap()
	require.Equal(t, "cat0", getS(t, rec, "category"))
	require.Equal(t, "0.5000", getS(t, rec, "price"), "the price stays the decimal string Twilio sent")
	require.Equal(t, "minutes", getS(t, rec, "usageUnit"))

	l.f.resetCalls()
	r, err = invoke(t, l, "getUsageRecords", map[string]property.Value{"period": sv("daily"), "category": sv("cat1"), "startDate": sv("2026-11-01")})
	require.NoError(t, err)
	require.Equal(t, 1, listLen(t, r, "records"))
	calls := l.f.callsMatching("GET", "/Usage/Records/Daily.json")
	require.Len(t, calls, 1)

	// Pagination follows next_page_uri, limit cuts the walk short.
	old := usagePageCap
	usagePageCap = 2
	t.Cleanup(func() { usagePageCap = old })
	r, err = invoke(t, l, "getUsageRecords", map[string]property.Value{"period": sv("thisMonth")})
	require.NoError(t, err)
	require.Equal(t, 3, listLen(t, r, "records"))
	r, err = invoke(t, l, "getUsageRecords", map[string]property.Value{"limit": nv(1)})
	require.NoError(t, err)
	require.Equal(t, 1, listLen(t, r, "records"))

	// A list that never ends fails loudly rather than returning a truncated result.
	oldTotal := fakeUsageTotal
	fakeUsageTotal = maxListPages*2 + 5
	usagePageCap = 1
	t.Cleanup(func() { fakeUsageTotal = oldTotal })
	_, err = invoke(t, l, "getUsageRecords", map[string]property.Value{"limit": nv(5000)})
	require.ErrorContains(t, err, "more than 100 pages")

	_, err = invoke(t, l, "getUsageRecords", map[string]property.Value{"period": sv("weekly")})
	require.ErrorContains(t, err, "period must be")
}

// A limit smaller than the list stops the walk: the pages after the one that reaches it are never requested.
func TestLookup_LimitStopsTheWalkEarly(t *testing.T) {
	l := newTL(t, "Subaccount", "sub")
	oldCap, oldTotal := usagePageCap, fakeUsageTotal
	usagePageCap, fakeUsageTotal = 2, 7
	t.Cleanup(func() { usagePageCap, fakeUsageTotal = oldCap, oldTotal })

	l.f.resetCalls()
	r, err := invoke(t, l, "getUsageRecords", map[string]property.Value{"limit": nv(3)})
	require.NoError(t, err)
	require.Equal(t, 3, listLen(t, r, "records"))
	require.Len(t, l.f.callsMatching("GET", "/Usage/Records/AllTime.json"), 2, "pages of 2 reach 3 records on the second page; the third and fourth are not read")

	l.f.resetCalls()
	r, err = invoke(t, l, "getUsageRecords", map[string]property.Value{"limit": nv(7)})
	require.NoError(t, err)
	require.Equal(t, 7, listLen(t, r, "records"))
	require.Len(t, l.f.callsMatching("GET", "/Usage/Records/AllTime.json"), 4, "without a smaller limit the whole list is walked")
}

func TestLookup_AuthorizedConnectApps(t *testing.T) {
	l := newTL(t, "Subaccount", "sub")
	r, err := invoke(t, l, "getAuthorizedConnectApps", map[string]property.Value{})
	require.NoError(t, err)
	require.Equal(t, 2, listLen(t, r, "apps"))
	app := r.Get("apps").AsArray().Get(1).AsMap()
	require.Equal(t, "app1", getS(t, app, "friendlyName"))
	require.Equal(t, "Acme", getS(t, app, "companyName"))
	require.Equal(t, "get-all", app.Get("permissions").AsArray().Get(0).AsString())

	sid := getS(t, app, "connectAppSid")
	one, err := invoke(t, l, "getAuthorizedConnectApp", map[string]property.Value{"connectAppSid": sv(sid)})
	require.NoError(t, err)
	require.Equal(t, sid, getS(t, one, "connectAppSid"))
	require.Equal(t, "https://example.com", getS(t, one, "homepageUrl"))

	_, err = invoke(t, l, "getAuthorizedConnectApp", map[string]property.Value{"connectAppSid": sv("CNnope")})
	require.ErrorContains(t, err, "was not found")
	_, err = invoke(t, l, "getAuthorizedConnectApp", map[string]property.Value{"connectAppSid": sv("")})
	require.ErrorContains(t, err, "must be set")
}

func TestLookup_EventCatalog(t *testing.T) {
	l := newTL(t, "Subaccount", "sub")
	r, err := invoke(t, l, "getEventTypes", map[string]property.Value{})
	require.NoError(t, err)
	require.Equal(t, 3, listLen(t, r, "types"))
	r, err = invoke(t, l, "getEventTypes", map[string]property.Value{"schemaId": sv("Messaging.MessageStatus")})
	require.NoError(t, err)
	require.Equal(t, 2, listLen(t, r, "types"))

	// meta.next_page_url paging is followed.
	old := eventCatalogPageCap
	eventCatalogPageCap = 1
	t.Cleanup(func() { eventCatalogPageCap = old })
	r, err = invoke(t, l, "getEventTypes", map[string]property.Value{})
	require.NoError(t, err)
	require.Equal(t, 3, listLen(t, r, "types"))

	one, err := invoke(t, l, "getEventType", map[string]property.Value{"type": sv("com.twilio.voice.insights.call-summary.complete")})
	require.NoError(t, err)
	require.Equal(t, "Voice.CallSummary", getS(t, one, "schemaId"))
	require.Equal(t, "deprecated", getS(t, one, "status"))
	_, err = invoke(t, l, "getEventType", map[string]property.Value{"type": sv("com.twilio.nope")})
	require.ErrorContains(t, err, "was not found")
	_, err = invoke(t, l, "getEventType", map[string]property.Value{"type": sv("")})
	require.ErrorContains(t, err, "must be set")

	s, err := invoke(t, l, "getEventSchema", map[string]property.Value{"schemaId": sv("Messaging.MessageStatus")})
	require.NoError(t, err)
	require.Equal(t, "Messaging.MessageStatus", getS(t, s, "schemaId"))
	require.Equal(t, 2.0, s.Get("latestVersion").AsNumber())
	require.Equal(t, 2, listLen(t, s, "versions"))
	require.Equal(t, 2.0, s.Get("versions").AsArray().Get(1).AsMap().Get("schemaVersion").AsNumber())
	_, err = invoke(t, l, "getEventSchema", map[string]property.Value{"schemaId": sv("Nope")})
	require.ErrorContains(t, err, "was not found")
	_, err = invoke(t, l, "getEventSchema", map[string]property.Value{"schemaId": sv("")})
	require.ErrorContains(t, err, "must be set")
}
