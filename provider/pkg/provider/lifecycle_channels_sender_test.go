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
	"encoding/json"
	"strings"
	"testing"

	pgp "github.com/pulumi/pulumi-go-provider"
	"github.com/pulumi/pulumi/sdk/v3/go/property"
	"github.com/stretchr/testify/require"
)

const csNumber = "whatsapp:+15017122661"

func channelsSenderInputs() property.Map {
	return pv(map[string]property.Value{
		"senderId":     sv(csNumber),
		"friendlyName": sv("support line"),
		"configuration": obj2(map[string]property.Value{
			"wabaId": sv("123456789"),
		}),
		"webhook": obj2(map[string]property.Value{
			"callbackUrl":    sv("https://example.com/in"),
			"callbackMethod": sv("POST"),
		}),
		"profile": obj2(map[string]property.Value{
			"name":  sv("Acme Support"),
			"about": sv("We help"),
			"websites": arrv(obj2(map[string]property.Value{
				"website": sv("https://example.com"), "label": sv("Home")})),
		}),
	})
}

func csProfile(in property.Map) property.Map { return in.Get("profile").AsMap() }

func withProfile(in property.Map, kv map[string]property.Value) property.Map {
	return with(in, map[string]property.Value{"profile": obj2(func() map[string]property.Value {
		m := map[string]property.Value{}
		for k, v := range csProfile(in).AsMap() {
			m[k] = v
		}
		for k, v := range kv {
			m[k] = v
		}
		return m
	}())})
}

func TestChannelsSenderLC_Lifecycle(t *testing.T) {
	l := newTL(t, "ChannelsSender", "wa")
	in := channelsSenderInputs()
	c := l.create(in)
	require.True(t, strings.HasPrefix(c.ID, "XE"), c.ID)
	require.Equal(t, c.ID, getS(t, c.Properties, "sid"))
	require.Equal(t, "CREATING", getS(t, c.Properties, "status"))

	// The body is JSON in the spec's snake_case, with the sender ID only on create.
	calls := l.f.callsMatching("POST", "/messaging/v2/Channels/Senders")
	require.Len(t, calls, 1)
	var sent map[string]any
	require.NoError(t, json.Unmarshal([]byte(calls[0].Body), &sent))
	require.Equal(t, csNumber, sent["sender_id"])
	require.Equal(t, "support line", sent["friendly_name"])
	require.Equal(t, "123456789", sent["configuration"].(map[string]any)["waba_id"])
	require.Equal(t, "POST", sent["webhook"].(map[string]any)["callback_method"])
	prof := sent["profile"].(map[string]any)
	require.Equal(t, "Acme Support", prof["name"])
	require.Equal(t, "Home", prof["websites"].([]any)[0].(map[string]any)["label"])

	requireNoChanges(t, l.diff(c.ID, c.Properties, in, in))
	rd := l.read(c.ID, c.Properties, in)
	requireNoChanges(t, l.diff(c.ID, rd.Properties, in, rd.Inputs))

	// senderId replaces; everything else updates in place.
	other := with(in, map[string]property.Value{"senderId": sv("whatsapp:+15017122662")})
	require.Equal(t, pgp.UpdateReplace, l.diff(c.ID, c.Properties, other, in).DetailedDiff["senderId"].Kind)
	next := with(in, map[string]property.Value{"friendlyName": sv("renamed"), "webhook": obj2(map[string]property.Value{
		"callbackUrl": sv("https://example.com/in2"), "callbackMethod": sv("PUT")})})
	d := l.diff(c.ID, c.Properties, next, in)
	require.Equal(t, pgp.Update, d.DetailedDiff["friendlyName"].Kind)
	require.Equal(t, pgp.Update, d.DetailedDiff["webhook.callbackUrl"].Kind)
	l.f.resetCalls()
	u := l.update(c.ID, c.Properties, next, in)
	post := l.f.callsMatching("POST", "/messaging/v2/Channels/Senders/"+c.ID)
	require.Len(t, post, 1)
	require.NotContains(t, post[0].Body, "sender_id")
	require.Equal(t, "renamed", l.f.get("cs", testAccount, c.ID)["friendly_name"])
	require.Equal(t, "https://example.com/in2", l.f.get("cs", testAccount, c.ID)["webhook"].(map[string]any)["callback_url"])
	requireNoChanges(t, l.diff(c.ID, u.Properties, next, next))
	rd = l.read(c.ID, u.Properties, next)
	requireNoChanges(t, l.diff(c.ID, rd.Properties, next, rd.Inputs))

	// Drift in the webhook shows.
	l.f.mutate("cs", testAccount, c.ID, func(o map[string]any) { o["webhook"].(map[string]any)["callback_url"] = "https://elsewhere.example" })
	rd = l.read(c.ID, u.Properties, next)
	require.Equal(t, pgp.Update, l.diff(c.ID, rd.Properties, next, rd.Inputs).DetailedDiff["webhook.callbackUrl"].Kind)

	// An import adopts everything Twilio reports.
	l.f.mutate("cs", testAccount, c.ID, func(o map[string]any) {
		o["status"] = "ONLINE"
		o["properties"] = map[string]any{"quality_rating": "GREEN", "messaging_limit": "TIER_1K"}
		o["offline_reasons"] = []any{map[string]any{"code": "63016", "message": "x", "more_info": "https://twilio.com/docs/errors/63016"}}
	})
	imp := l.read(c.ID, empty(), empty())
	require.Empty(t, l.check(imp.Inputs).Failures)
	require.Equal(t, csNumber, getS(t, imp.Inputs, "senderId"))
	require.Equal(t, "renamed", getS(t, imp.Inputs, "friendlyName"))
	require.Equal(t, "Acme Support", getS(t, csProfile(imp.Inputs), "name"))
	require.Equal(t, "123456789", getS(t, imp.Inputs.Get("configuration").AsMap(), "wabaId"))
	require.Equal(t, "sms", getS(t, imp.Inputs.Get("configuration").AsMap(), "verificationMethod"))
	require.Equal(t, "ONLINE", getS(t, imp.Properties, "status"))
	require.Equal(t, "GREEN", getS(t, imp.Properties.Get("properties").AsMap(), "qualityRating"))
	require.Equal(t, "63016", getS(t, imp.Properties.Get("offlineReasons").AsArray().Get(0).AsMap(), "code"))
	l.requireImportFails("XE" + strings.Repeat("0", 32))

	// Deleted outside Pulumi: Read returns an empty ID, and Delete tolerates the absence.
	l.f.remove("cs", testAccount, c.ID)
	require.Empty(t, l.read(c.ID, u.Properties, next).ID)
	l.del(c.ID, u.Properties)
	c = l.create(in)
	l.del(c.ID, c.Properties)
	require.Nil(t, l.f.get("cs", testAccount, c.ID))
}

// A program that writes the value Twilio defaults to (verification method sms) has no diff, and
// neither has one that leaves it out although Twilio reports it.
func TestChannelsSenderLC_DefaultWrittenNoDiff(t *testing.T) {
	l := newTL(t, "ChannelsSender", "wa")
	in := with(channelsSenderInputs(), map[string]property.Value{"configuration": obj2(map[string]property.Value{
		"wabaId": sv("123456789"), "verificationMethod": sv("sms")})})
	c := l.create(in)
	rd := l.read(c.ID, c.Properties, in)
	requireNoChanges(t, l.diff(c.ID, rd.Properties, in, rd.Inputs))
	omitted := channelsSenderInputs()
	c2 := l.create(omitted)
	rd2 := l.read(c2.ID, c2.Properties, omitted)
	requireNoChanges(t, l.diff(c2.ID, rd2.Properties, omitted, rd2.Inputs))
	require.False(t, hasProp(rd2.Inputs.Get("configuration").AsMap(), "verificationMethod"))
}

// A new display name stays pending at Meta while Twilio reports the old one: no perpetual diff
// while the review runs, the active name once it failed (so the next update re-submits), and no
// diff once it is applied.
func TestChannelsSenderLC_PendingDisplayName(t *testing.T) {
	l := newTL(t, "ChannelsSender", "wa")
	in := channelsSenderInputs()
	c := l.create(in)
	next := withProfile(in, map[string]property.Value{"name": sv("Acme Care")})
	d := l.diff(c.ID, c.Properties, next, in)
	require.Equal(t, pgp.Update, d.DetailedDiff["profile.name"].Kind)
	u := l.update(c.ID, c.Properties, next, in)
	stored := l.f.get("cs", testAccount, c.ID)
	require.Equal(t, "Acme Support", stored["profile"].(map[string]any)["name"], "Twilio keeps the active name")
	require.Equal(t, "PENDING_REVIEW", getS(t, u.Properties, "pendingDisplayNameStatus"))
	require.Equal(t, "Acme Care", getS(t, u.Properties, "pendingDisplayName"))
	requireNoChanges(t, l.diff(c.ID, u.Properties, next, next))

	rd := l.read(c.ID, u.Properties, next)
	require.Equal(t, "Acme Care", getS(t, csProfile(rd.Inputs), "name"), "pending review keeps the desired name")
	requireNoChanges(t, l.diff(c.ID, rd.Properties, next, rd.Inputs))

	// Approved and waiting to be applied: still no diff.
	l.f.mutate("cs", testAccount, c.ID, func(o map[string]any) { o["pending_display_name_status"] = "APPROVED" })
	rd = l.read(c.ID, u.Properties, next)
	requireNoChanges(t, l.diff(c.ID, rd.Properties, next, rd.Inputs))

	// Failed: the active name is reported, so the program shows a diff and an update re-submits.
	for _, failed := range channelsSenderFailedNames {
		l.f.mutate("cs", testAccount, c.ID, func(o map[string]any) { o["pending_display_name_status"] = failed })
		rd = l.read(c.ID, u.Properties, next)
		require.Equal(t, "Acme Support", getS(t, csProfile(rd.Inputs), "name"), failed)
		require.Equal(t, pgp.Update, l.diff(c.ID, rd.Properties, next, rd.Inputs).DetailedDiff["profile.name"].Kind, failed)
	}
	l.f.resetCalls()
	u = l.update(c.ID, rd.Properties, next, rd.Inputs)
	require.Contains(t, l.f.callsMatching("POST", "/messaging/v2/Channels/Senders/"+c.ID)[0].Body, `"Acme Care"`)
	require.Equal(t, "PENDING_REVIEW", getS(t, u.Properties, "pendingDisplayNameStatus"))

	// A pending name that is not the program's (changed elsewhere) is not kept.
	l.f.mutate("cs", testAccount, c.ID, func(o map[string]any) { o["pending_display_name"] = "Someone Else" })
	rd = l.read(c.ID, u.Properties, next)
	require.Equal(t, "Acme Support", getS(t, csProfile(rd.Inputs), "name"))

	// Completed: Twilio reports the new name as the active one.
	l.f.mutate("cs", testAccount, c.ID, func(o map[string]any) {
		o["profile"].(map[string]any)["name"] = "Acme Care"
		o["pending_display_name"] = "Acme Care"
		o["pending_display_name_status"] = "COMPLETED"
	})
	rd = l.read(c.ID, u.Properties, next)
	require.Equal(t, "Acme Care", getS(t, csProfile(rd.Inputs), "name"))
	requireNoChanges(t, l.diff(c.ID, rd.Properties, next, rd.Inputs))
}

// The verification code is write-only: sent when it is new, never read back, kept secret.
func TestChannelsSenderLC_VerificationCodeIsWriteOnly(t *testing.T) {
	l := newTL(t, "ChannelsSender", "wa")
	in := channelsSenderInputs()
	c := l.create(in)
	require.NotContains(t, l.f.callsMatching("POST", "/messaging/v2/Channels/Senders")[0].Body, "verification_code")

	coded := with(in, map[string]property.Value{"configuration": obj2(map[string]property.Value{
		"wabaId": sv("123456789"), "verificationCode": property.New("123456").WithSecret(true)})})
	l.f.resetCalls()
	u := l.update(c.ID, c.Properties, coded, in)
	require.Contains(t, l.f.callsMatching("POST", "/messaging/v2/Channels/Senders/"+c.ID)[0].Body, `"verification_code":"123456"`)
	require.Equal(t, "ONLINE", getS(t, u.Properties, "status"))
	code := u.Properties.Get("configuration").AsMap().Get("verificationCode")
	require.Equal(t, "123456", code.WithSecret(false).AsString())
	require.True(t, code.Secret(), "the verification code must be a secret")

	// Refresh keeps it and shows no drift, although Twilio never returns it.
	rd := l.read(c.ID, u.Properties, coded)
	requireNoChanges(t, l.diff(c.ID, rd.Properties, coded, rd.Inputs))
	require.Equal(t, "123456", rd.Inputs.Get("configuration").AsMap().Get("verificationCode").WithSecret(false).AsString())

	// A later update does not send the spent code again.
	next := with(coded, map[string]property.Value{"friendlyName": sv("again")})
	l.f.resetCalls()
	l.update(c.ID, u.Properties, next, coded)
	require.NotContains(t, l.f.callsMatching("POST", "/messaging/v2/Channels/Senders/"+c.ID)[0].Body, "verification_code")

	// A wrong code is Twilio's refusal, surfaced.
	bad := with(next, map[string]property.Value{"configuration": obj2(map[string]property.Value{
		"wabaId": sv("123456789"), "verificationCode": property.New("000000").WithSecret(true)})})
	_, err := l.s.Update(pgp.UpdateRequest{ID: c.ID, Urn: l.urn, State: u.Properties, Inputs: l.checked(bad), OldInputs: next})
	require.ErrorContains(t, err, "invalid verification code")
}

func TestChannelsSenderLC_Check(t *testing.T) {
	l := newTL(t, "ChannelsSender", "wa")
	in := channelsSenderInputs()
	fails := func(m property.Map, prop string) {
		t.Helper()
		got := []string{}
		for _, f := range l.check(m).Failures {
			got = append(got, f.Property)
		}
		require.Contains(t, got, prop)
	}
	fails(with(in, map[string]property.Value{"senderId": sv("rcs:brand_agent")}), "senderId")
	fails(with(in, map[string]property.Value{"senderId": sv("whatsapp:15017122661")}), "senderId")
	fails(withProfile(in, map[string]property.Value{"name": sv("")}), "profile.name")
	fails(withProfile(in, map[string]property.Value{"name": sv(strings.Repeat("x", 256))}), "profile.name")
	fails(with(in, map[string]property.Value{"webhook": obj2(map[string]property.Value{"callbackMethod": sv("GET")})}), "webhook.callbackMethod")
	fails(with(in, map[string]property.Value{"configuration": obj2(map[string]property.Value{"verificationMethod": sv("email")})}), "configuration.verificationMethod")
	fails(with(in, map[string]property.Value{"configuration": obj2(map[string]property.Value{"accountType": sv("DIRECT")})}), "configuration.accountType")
	fails(with(in, map[string]property.Value{"friendlyName": sv("")}), "friendlyName")
	// Unknown values are not validated at preview.
	require.Empty(t, l.check(with(in, map[string]property.Value{"senderId": computed()})).Failures)
	require.Empty(t, l.check(in).Failures)
}

func TestLookup_ChannelsSender(t *testing.T) {
	l := newTL(t, "ChannelsSender", "wa")
	c := l.create(channelsSenderInputs())
	l.f.mutate("cs", testAccount, c.ID, func(o map[string]any) {
		o["status"] = "ONLINE"
		o["pending_display_name"] = "Acme Care"
		o["pending_display_name_status"] = "PENDING_REVIEW"
	})
	r, err := invoke(t, l, "getChannelsSender", map[string]property.Value{"sid": sv(c.ID)})
	require.NoError(t, err)
	require.Equal(t, csNumber, getS(t, r, "senderId"))
	require.Equal(t, "ONLINE", getS(t, r, "status"))
	require.Equal(t, "Acme Support", getS(t, r.Get("profile").AsMap(), "name"))
	require.Equal(t, "Acme Care", getS(t, r, "pendingDisplayName"))
	require.Equal(t, "PENDING_REVIEW", getS(t, r, "pendingDisplayNameStatus"))
	require.Equal(t, "sms", getS(t, r.Get("configuration").AsMap(), "verificationMethod"))
	require.False(t, hasProp(r.Get("configuration").AsMap(), "verificationCode"))

	_, err = invoke(t, l, "getChannelsSender", map[string]property.Value{"sid": sv("XE" + strings.Repeat("0", 32))})
	require.ErrorContains(t, err, "was not found")
	_, err = invoke(t, l, "getChannelsSender", map[string]property.Value{"sid": sv("")})
	require.ErrorContains(t, err, "must be set")
	_, err = invoke(t, l, "getChannelsSender", map[string]property.Value{"sid": sv("CHnotasender")})
	require.ErrorContains(t, err, "sid must be the SID of a sender")
}

func TestLookup_ChannelsSenders(t *testing.T) {
	l := newTL(t, "ChannelsSender", "wa")
	// Empty account: an empty list, not an error.
	r, err := invoke(t, l, "getChannelsSenders", map[string]property.Value{"channel": sv("whatsapp")})
	require.NoError(t, err)
	require.Empty(t, r.Get("senders").AsArray().AsSlice())

	for _, n := range []string{"+15017122661", "+15017122662", "+15017122663"} {
		in := with(channelsSenderInputs(), map[string]property.Value{"senderId": sv("whatsapp:" + n)})
		l.create(in)
	}
	l.f.put("cs", testAccount, "XE"+strings.Repeat("a", 32), map[string]any{"sid": "XE" + strings.Repeat("a", 32), "status": "ONLINE",
		"sender_id": "rcs:brand_agent", "profile": map[string]any{"name": "Brand", "use_case": "OTP"},
		"compliance": map[string]any{"registration_sid": "CR123", "countries": []any{map[string]any{
			"country": "US", "status": "ONLINE", "carriers": []any{map[string]any{"name": "Verizon", "status": "APPROVED"}}}}}})
	l.f.resetCalls()
	r, err = invoke(t, l, "getChannelsSenders", map[string]property.Value{"channel": sv("whatsapp")})
	require.NoError(t, err)
	require.Len(t, r.Get("senders").AsArray().AsSlice(), 3, "both pages are read")
	calls := l.f.callsMatching("GET", "/messaging/v2/Channels/Senders")
	require.Len(t, calls, 2)
	for _, c := range calls {
		require.Empty(t, c.Query.Get("PageSize"), "no page size is sent")
		require.Equal(t, "whatsapp", c.Query.Get("Channel"))
	}

	r, err = invoke(t, l, "getChannelsSenders", map[string]property.Value{"channel": sv("rcs")})
	require.NoError(t, err)
	rcs := r.Get("senders").AsArray().Get(0).AsMap()
	require.Equal(t, "rcs:brand_agent", getS(t, rcs, "senderId"))
	require.Equal(t, "OTP", getS(t, rcs, "profileUseCase"))
	country := rcs.Get("compliance").AsMap().Get("countries").AsArray().Get(0).AsMap()
	require.Equal(t, "US", getS(t, country, "country"))
	require.Equal(t, "APPROVED", getS(t, country.Get("carriers").AsArray().Get(0).AsMap(), "status"))

	// The channel is required and checked before any request.
	l.f.resetCalls()
	_, err = invoke(t, l, "getChannelsSenders", map[string]property.Value{"channel": sv("")})
	require.ErrorContains(t, err, "must be set")
	_, err = invoke(t, l, "getChannelsSenders", map[string]property.Value{"channel": sv("sms")})
	require.ErrorContains(t, err, "channel must be whatsapp or rcs")
	require.Zero(t, l.f.callCount())
}
