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
	"strings"
	"testing"

	pgp "github.com/pulumi/pulumi-go-provider"
	"github.com/pulumi/pulumi/sdk/v3/go/property"
	"github.com/stretchr/testify/require"
)

func msService(t *testing.T, l *tl) string {
	t.Helper()
	return l.other("MessagingService", "svc").create(pv(map[string]property.Value{"friendlyName": sv("pool")})).ID
}

// requirePoolMemberLifecycle runs the lifecycle every pool member shares: create, idempotent diff,
// read, replace on every input, import with Check on the imported inputs, deleted outside, delete.
func requirePoolMemberLifecycle(t *testing.T, typ, coll string, in func(svc string) property.Map, changes func(svc string) map[string]property.Value, idOf func(c pgp.CreateResponse) string) {
	t.Helper()
	l := newTL(t, typ, "m")
	svc := msService(t, l)
	input := in(svc)
	c := l.create(input)
	id := idOf(c)
	require.Equal(t, svc+"/"+id, c.ID)
	require.NotNil(t, l.f.get("msnd", svc, coll+"/"+id))
	requireNoChanges(t, l.diff(c.ID, c.Properties, input, input))
	rd := l.read(c.ID, c.Properties, input)
	require.Equal(t, c.ID, rd.ID)
	requireNoChanges(t, l.diff(c.ID, rd.Properties, input, rd.Inputs))

	for k, v := range changes(svc) {
		d := l.diff(c.ID, rd.Properties, with(input, map[string]property.Value{k: v}), input)
		require.Equal(t, pgp.UpdateReplace, d.DetailedDiff[k].Kind, k)
	}

	imp := l.read(c.ID, empty(), empty())
	require.Equal(t, c.ID, imp.ID)
	require.Empty(t, l.check(imp.Inputs).Failures)
	require.Equal(t, svc, getS(t, imp.Inputs, "serviceSid"))
	requireNoChanges(t, l.diff(imp.ID, imp.Properties, input, imp.Inputs))
	l.requireImportFails(svc + "/AI" + strings.Repeat("0", 32))
	_, err := l.s.Read(pgp.ReadRequest{ID: "badid", Urn: l.urn, Properties: empty(), Inputs: empty()})
	require.ErrorContains(t, err, "<serviceSid>/<sid>")

	l.f.remove("msnd", svc, coll+"/"+id)
	require.Empty(t, l.read(c.ID, c.Properties, input).ID)
	l.del(c.ID, c.Properties)
	c = l.create(input)
	require.NotNil(t, l.f.get("msnd", svc, coll+"/"+idOf(c)))
	l.del(c.ID, c.Properties)
	require.Nil(t, l.f.get("msnd", svc, coll+"/"+idOf(c)))
	require.NotEmpty(t, l.f.callsMatching("DELETE", "/"+coll+"/"+idOf(c)))
}

func TestMessagingServiceShortCodeLC_Lifecycle(t *testing.T) {
	sc := "SC" + strings.Repeat("a", 32)
	requirePoolMemberLifecycle(t, "MessagingServiceShortCode", "ShortCodes",
		func(svc string) property.Map {
			return pv(map[string]property.Value{"serviceSid": sv(svc), "shortCodeSid": sv(sc)})
		},
		func(string) map[string]property.Value {
			return map[string]property.Value{"serviceSid": sv("MG" + strings.Repeat("9", 32)), "shortCodeSid": sv("SC" + strings.Repeat("9", 32))}
		},
		func(c pgp.CreateResponse) string { return getS(t, c.Properties, "shortCodeSid") })
	l := newTL(t, "MessagingServiceShortCode", "m")
	svc := msService(t, l)
	c := l.create(pv(map[string]property.Value{"serviceSid": sv(svc), "shortCodeSid": sv(sc)}))
	require.Equal(t, "+12345", getS(t, c.Properties, "shortCode"))
	require.Equal(t, "US", getS(t, c.Properties, "countryCode"))
	require.Len(t, c.Properties.Get("capabilities").AsArray().AsSlice(), 2)
	require.Equal(t, sc, l.f.callsMatching("POST", "/Services/"+svc+"/ShortCodes")[0].Form.Get("ShortCodeSid"))
	require.Len(t, l.check(pv(map[string]property.Value{"serviceSid": sv("nope"), "shortCodeSid": sv("nope")})).Failures, 2)
	require.Empty(t, l.check(pv(map[string]property.Value{"serviceSid": computed(), "shortCodeSid": computed()})).Failures)
}

func TestMessagingServiceChannelSenderLC_Lifecycle(t *testing.T) {
	xe := "XE" + strings.Repeat("b", 32)
	requirePoolMemberLifecycle(t, "MessagingServiceChannelSender", "ChannelSenders",
		func(svc string) property.Map {
			return pv(map[string]property.Value{"serviceSid": sv(svc), "channelSenderSid": sv(xe)})
		},
		func(string) map[string]property.Value {
			return map[string]property.Value{"serviceSid": sv("MG" + strings.Repeat("9", 32)), "channelSenderSid": sv("XE" + strings.Repeat("9", 32))}
		},
		func(c pgp.CreateResponse) string { return getS(t, c.Properties, "channelSenderSid") })
	l := newTL(t, "MessagingServiceChannelSender", "m")
	svc := msService(t, l)
	c := l.create(pv(map[string]property.Value{"serviceSid": sv(svc), "channelSenderSid": sv(xe)}))
	require.Equal(t, "whatsapp:+14155550100", getS(t, c.Properties, "sender"))
	require.Equal(t, "whatsapp", getS(t, c.Properties, "senderType"))
	require.Equal(t, xe, l.f.callsMatching("POST", "/Services/"+svc+"/ChannelSenders")[0].Form.Get("Sid"))
	require.Len(t, l.check(pv(map[string]property.Value{"serviceSid": sv(svc), "channelSenderSid": sv("SC" + strings.Repeat("a", 32))})).Failures, 1)
}

func TestMessagingServiceAlphaSenderLC_Lifecycle(t *testing.T) {
	requirePoolMemberLifecycle(t, "MessagingServiceAlphaSender", "AlphaSenders",
		func(svc string) property.Map {
			return pv(map[string]property.Value{"serviceSid": sv(svc), "alphaSender": sv("Pulumi Inc")})
		},
		func(string) map[string]property.Value {
			return map[string]property.Value{"serviceSid": sv("MG" + strings.Repeat("9", 32)), "alphaSender": sv("Other")}
		},
		func(c pgp.CreateResponse) string { return getS(t, c.Properties, "sid") })
	l := newTL(t, "MessagingServiceAlphaSender", "m")
	svc := msService(t, l)
	c := l.create(pv(map[string]property.Value{"serviceSid": sv(svc), "alphaSender": sv("Pulumi")}))
	require.True(t, strings.HasPrefix(getS(t, c.Properties, "sid"), "AI"))
	require.Equal(t, "Pulumi", l.f.callsMatching("POST", "/Services/"+svc+"/AlphaSenders")[0].Form.Get("AlphaSender"))
	// A second entry for the same name is refused by Twilio and surfaces as an error.
	_, err := l.s.Create(pgp.CreateRequest{Urn: l.urn, Properties: l.checked(pv(map[string]property.Value{"serviceSid": sv(svc), "alphaSender": sv("Pulumi")}))})
	require.ErrorContains(t, err, "already in the Messaging Service")
	for _, bad := range []string{"", "12345", "TooLongSender1", "bad!chars", "caf\u00e9"} {
		chk := l.check(pv(map[string]property.Value{"serviceSid": sv(svc), "alphaSender": sv(bad)}))
		require.Len(t, chk.Failures, 1, bad)
		require.Equal(t, "alphaSender", chk.Failures[0].Property)
	}
	require.Empty(t, l.check(pv(map[string]property.Value{"serviceSid": sv(svc), "alphaSender": computed()})).Failures)
	require.Empty(t, l.check(pv(map[string]property.Value{"serviceSid": sv(svc), "alphaSender": sv("A-b_c+d&e 1")})).Failures)
}

func TestMessagingServiceDestinationAlphaSenderLC_Lifecycle(t *testing.T) {
	requirePoolMemberLifecycle(t, "MessagingServiceDestinationAlphaSender", "DestinationAlphaSenders",
		func(svc string) property.Map {
			return pv(map[string]property.Value{"serviceSid": sv(svc), "alphaSender": sv("Pulumi"), "isoCountryCode": sv("GB")})
		},
		func(string) map[string]property.Value {
			return map[string]property.Value{"serviceSid": sv("MG" + strings.Repeat("9", 32)), "alphaSender": sv("Other"), "isoCountryCode": sv("FR")}
		},
		func(c pgp.CreateResponse) string { return getS(t, c.Properties, "sid") })
	l := newTL(t, "MessagingServiceDestinationAlphaSender", "m")
	svc := msService(t, l)
	// With no country the sender is the default for every country; the import leaves isoCountryCode unset.
	def := pv(map[string]property.Value{"serviceSid": sv(svc), "alphaSender": sv("Pulumi")})
	c := l.create(def)
	form := l.f.callsMatching("POST", "/Services/"+svc+"/DestinationAlphaSenders")[0].Form
	require.Empty(t, form["IsoCountryCode"])
	imp := l.read(c.ID, empty(), empty())
	require.False(t, hasProp(imp.Inputs, "isoCountryCode"))
	require.Empty(t, l.check(imp.Inputs).Failures)
	requireNoChanges(t, l.diff(imp.ID, imp.Properties, def, imp.Inputs))
	require.Equal(t, pgp.AddReplace, l.diff(c.ID, c.Properties, with(def, map[string]property.Value{"isoCountryCode": sv("GB")}), def).DetailedDiff["isoCountryCode"].Kind)
	gb := l.create(with(def, map[string]property.Value{"isoCountryCode": sv("gb")}))
	// Check upper-cases the code (Twilio reports it that way), so a lower case program is no change after a refresh.
	lower := with(def, map[string]property.Value{"isoCountryCode": sv("gb")})
	checked := l.check(lower)
	require.Empty(t, checked.Failures)
	require.Equal(t, "GB", getS(t, checked.Inputs, "isoCountryCode"))
	require.Equal(t, "GB", l.f.callsMatching("POST", "/Services/"+svc+"/DestinationAlphaSenders")[1].Form.Get("IsoCountryCode"))
	rd := l.read(gb.ID, gb.Properties, checked.Inputs)
	require.Equal(t, "GB", getS(t, rd.Inputs, "isoCountryCode"))
	requireNoChanges(t, l.diff(gb.ID, rd.Properties, checked.Inputs, rd.Inputs))
	require.Len(t, l.check(pv(map[string]property.Value{"serviceSid": sv(svc), "alphaSender": sv("Pulumi"), "isoCountryCode": sv("GBR")})).Failures, 1)
}

func TestLookup_MessagingServicePoolMembers(t *testing.T) {
	l := newTL(t, "MessagingServiceShortCode", "sc")
	svc := msService(t, l)
	sc := "SC" + strings.Repeat("a", 32)
	l.create(pv(map[string]property.Value{"serviceSid": sv(svc), "shortCodeSid": sv(sc)}))
	r, err := invoke(t, l, "getMessagingServiceShortCode", map[string]property.Value{"serviceSid": sv(svc), "sid": sv(sc)})
	require.NoError(t, err)
	require.Equal(t, "+12345", getS(t, r, "shortCode"))
	require.Len(t, r.Get("capabilities").AsArray().AsSlice(), 2)

	xe := "XE" + strings.Repeat("b", 32)
	l.other("MessagingServiceChannelSender", "ch").create(pv(map[string]property.Value{"serviceSid": sv(svc), "channelSenderSid": sv(xe)}))
	r, err = invoke(t, l, "getMessagingServiceChannelSender", map[string]property.Value{"serviceSid": sv(svc), "sid": sv(xe)})
	require.NoError(t, err)
	require.Equal(t, "whatsapp", getS(t, r, "senderType"))

	al := l.other("MessagingServiceAlphaSender", "al").create(pv(map[string]property.Value{"serviceSid": sv(svc), "alphaSender": sv("Pulumi")}))
	r, err = invoke(t, l, "getMessagingServiceAlphaSender", map[string]property.Value{"serviceSid": sv(svc), "sid": sv(getS(t, al.Properties, "sid"))})
	require.NoError(t, err)
	require.Equal(t, "Pulumi", getS(t, r, "alphaSender"))

	da := l.other("MessagingServiceDestinationAlphaSender", "da").create(pv(map[string]property.Value{"serviceSid": sv(svc), "alphaSender": sv("Pulumi")}))
	r, err = invoke(t, l, "getMessagingServiceDestinationAlphaSender", map[string]property.Value{"serviceSid": sv(svc), "sid": sv(getS(t, da.Properties, "sid"))})
	require.NoError(t, err)
	require.False(t, hasProp(r, "isoCountryCode"))

	for _, fn := range []string{"getMessagingServiceShortCode", "getMessagingServiceChannelSender", "getMessagingServiceAlphaSender", "getMessagingServiceDestinationAlphaSender"} {
		_, err = invoke(t, l, fn, map[string]property.Value{"serviceSid": sv(svc), "sid": sv("AImissing")})
		require.ErrorContains(t, err, "was not found", fn)
		_, err = invoke(t, l, fn, map[string]property.Value{"serviceSid": sv(""), "sid": sv("x")})
		require.ErrorContains(t, err, "must be set", fn)
	}
}
