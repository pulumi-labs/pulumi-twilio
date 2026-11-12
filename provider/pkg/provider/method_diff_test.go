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

	pgp "github.com/pulumi/pulumi-go-provider"
	"github.com/pulumi/pulumi/sdk/v3/go/property"
	"github.com/stretchr/testify/require"
)

// #1743: an imported Application has no voiceMethod input (a returned POST is the default and is
// not adopted, #1572), so a program that states voiceMethod: "POST" must plan nothing.
func TestApplicationLC_ImportThenProgramStatesDefaultMethod(t *testing.T) {
	l := newTL(t, "Application", "app")
	c := l.create(pv(map[string]property.Value{"friendlyName": sv("ivr"), "voiceUrl": sv("https://example.com/voice")}))
	imp := l.read(c.ID, empty(), empty())
	require.False(t, hasProp(imp.Inputs, "voiceMethod"), "the default method is not adopted")

	prog := with(imp.Inputs, map[string]property.Value{"voiceMethod": sv("POST"), "smsMethod": sv("post")})
	requireNoChanges(t, l.diff(imp.ID, imp.Properties, prog, imp.Inputs))
	// A program that omits the methods plans nothing either (#1572).
	requireNoChanges(t, l.diff(imp.ID, imp.Properties, imp.Inputs, imp.Inputs))
}

// The default is only equal to POST: a real change to or from another method still plans an update,
// and the update is the one that resets a removed method.
func TestApplicationLC_MethodChangesStillDiff(t *testing.T) {
	l := newTL(t, "Application", "app")
	base := pv(map[string]property.Value{"friendlyName": sv("ivr"), "voiceUrl": sv("https://example.com/voice")})
	get := with(base, map[string]property.Value{"voiceMethod": sv("GET")})
	c := l.create(get)

	// GET -> unset resets to POST.
	d := l.diff(c.ID, c.Properties, base, get)
	require.True(t, d.HasChanges)
	require.Equal(t, pgp.Delete, d.DetailedDiff["voiceMethod"].Kind)
	// GET -> POST.
	d = l.diff(c.ID, c.Properties, with(base, map[string]property.Value{"voiceMethod": sv("POST")}), get)
	require.True(t, d.HasChanges)
	require.Equal(t, pgp.Update, d.DetailedDiff["voiceMethod"].Kind)
	// unset -> GET.
	post := l.create(base)
	d = l.diff(post.ID, post.Properties, get, base)
	require.True(t, d.HasChanges)
	require.Equal(t, pgp.Add, d.DetailedDiff["voiceMethod"].Kind)
	// POST dropped alongside a real change elsewhere keeps the real change only.
	d = l.diff(post.ID, post.Properties,
		with(base, map[string]property.Value{"voiceMethod": sv("POST"), "friendlyName": sv("renamed")}), base)
	require.True(t, d.HasChanges)
	require.Contains(t, d.DetailedDiff, "friendlyName")
	require.NotContains(t, d.DetailedDiff, "voiceMethod")
}

// The wrapper is provider-wide: another resource with a method field behaves the same.
func TestNumberLC_ImportThenProgramStatesDefaultMethod(t *testing.T) {
	l := newNumberLC(t)
	c := l.create(with(numberInputs(), map[string]property.Value{"smsMethod": sv("POST")}))
	imp := l.read(c.ID, empty(), empty())
	require.False(t, hasProp(imp.Inputs, "smsMethod"))
	in := with(l.checked(imp.Inputs), map[string]property.Value{"smsMethod": sv("POST")})
	requireNoChanges(t, l.diff(imp.ID, imp.Properties, in, imp.Inputs))
	// A computed method is left to infer.
	d := l.diff(imp.ID, imp.Properties, with(in, map[string]property.Value{"smsMethod": computed()}), imp.Inputs)
	require.True(t, d.HasChanges)
}

// Twilio takes GET and POST in either case (#1857): Check accepts a lower case method, normalizes
// it to upper case so the request carries the canonical form, and still rejects a real typo.
func TestMethodInputsAcceptAnyCase(t *testing.T) {
	l := newTL(t, "Application", "app")
	in := pv(map[string]property.Value{"friendlyName": sv("a"), "voiceMethod": sv("post"), "smsMethod": sv("Get")})
	chk := l.check(in)
	require.Empty(t, chk.Failures)
	require.Equal(t, "POST", getS(t, chk.Inputs, "voiceMethod"))
	require.Equal(t, "GET", getS(t, chk.Inputs, "smsMethod"))

	c := l.create(in)
	form := l.f.callsMatching("POST", "/Applications.json")[0].Form
	require.Equal(t, "POST", form.Get("VoiceMethod"))
	require.Equal(t, "GET", form.Get("SmsMethod"))
	require.Equal(t, "POST", l.f.get("app", testAccount, c.ID)["voice_method"])
	// Twilio reports "POST"; the program wrote "post": nothing to change.
	requireNoChanges(t, l.diff(c.ID, c.Properties, in, l.checked(in)))

	bad := l.check(pv(map[string]property.Value{"friendlyName": sv("a"), "voiceMethod": sv("PUT")}))
	require.Len(t, bad.Failures, 1)
	require.Equal(t, "voiceMethod", bad.Failures[0].Property)

	d := newTL(t, "SipDomain", "d")
	dc := d.check(pv(map[string]property.Value{"domainName": sv("x.sip.twilio.com"), "voiceMethod": sv("get")}))
	require.Empty(t, dc.Failures)
	require.Equal(t, "GET", getS(t, dc.Inputs, "voiceMethod"))
}
