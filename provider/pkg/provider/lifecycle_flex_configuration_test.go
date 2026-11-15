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
	"testing"

	pgp "github.com/pulumi/pulumi-go-provider"
	"github.com/pulumi/pulumi/sdk/v3/go/property"
	"github.com/stretchr/testify/require"
)

func TestFlexConfigurationLC_Lifecycle(t *testing.T) {
	l := newTL(t, "FlexConfiguration", "cfg")
	in := pv(map[string]property.Value{
		"uiAttributes":         sv(`{"colorTheme":{"baseName":"GreyDark"},"language":"en-US"}`),
		"uiLanguage":           sv("en-US"),
		"callRecordingEnabled": bv(true),
	})
	c := l.create(in)
	require.Equal(t, "account", c.ID)
	calls := l.f.callsMatching("POST", "/flex-api/v1/Configuration")
	require.Len(t, calls, 1)
	var body map[string]any
	require.NoError(t, json.Unmarshal([]byte(calls[0].Body), &body))
	require.Equal(t, testAccount, body["account_sid"])
	require.Equal(t, "en-US", body["ui_language"])
	require.Equal(t, true, body["call_recording_enabled"])
	require.Equal(t, map[string]any{"colorTheme": map[string]any{"baseName": "GreyDark"}, "language": "en-US"}, body["ui_attributes"])
	require.NotContains(t, body, "crm_enabled")
	require.Equal(t, "ok", getS(t, c.Properties, "status"))
	require.Equal(t, "2.0.0", getS(t, c.Properties, "uiVersion"))
	requireNoChanges(t, l.diff(c.ID, c.Properties, in, in))

	// Key order and whitespace of a JSON object are not a change.
	reordered := with(in, map[string]property.Value{"uiAttributes": sv(`{ "language": "en-US", "colorTheme": {"baseName": "GreyDark"} }`)})
	requireNoChanges(t, l.diff(c.ID, c.Properties, reordered, in))
	rd := l.read(c.ID, c.Properties, in)
	requireNoChanges(t, l.diff(c.ID, rd.Properties, in, rd.Inputs))
	require.Equal(t, `{"colorTheme":{"baseName":"GreyDark"},"language":"en-US"}`, getS(t, rd.Inputs, "uiAttributes"))

	// Drift on Twilio shows up on the settings the program manages, and only those.
	l.f.mutate("flex", testAccount, "Configuration", func(o map[string]any) {
		o["ui_language"] = "de-DE"
		o["crm_type"] = "custom"
	})
	rd = l.read(c.ID, c.Properties, in)
	require.Equal(t, "de-DE", getS(t, rd.Inputs, "uiLanguage"))
	require.False(t, hasProp(rd.Inputs, "crmType"))
	require.Equal(t, pgp.Update, l.diff(c.ID, rd.Properties, in, rd.Inputs).DetailedDiff["uiLanguage"].Kind)
	// Removing an input leaves the setting alone.
	requireNoChanges(t, l.diff(c.ID, c.Properties, without(in, "uiLanguage"), in))

	next := with(in, map[string]property.Value{"crmEnabled": bv(true), "serverlessServiceSids": property.New([]property.Value{sv("ZS" + "a1b2")})})
	require.Equal(t, pgp.Update, l.diff(c.ID, c.Properties, next, in).DetailedDiff["crmEnabled"].Kind)
	l.f.resetCalls()
	u := l.update(c.ID, c.Properties, next, in)
	calls = l.f.callsMatching("POST", "/flex-api/v1/Configuration")
	require.Len(t, calls, 1)
	require.NoError(t, json.Unmarshal([]byte(calls[0].Body), &body))
	require.Equal(t, true, body["crm_enabled"])
	require.Equal(t, []any{"ZSa1b2"}, body["serverless_service_sids"])
	requireNoChanges(t, l.diff(c.ID, u.Properties, next, next))

	// An import adopts nothing: Twilio documents no defaults.
	imp := l.read("account", empty(), empty())
	require.Equal(t, "account", imp.ID)
	require.False(t, hasProp(imp.Inputs, "uiAttributes"))
	require.False(t, hasProp(imp.Inputs, "callRecordingEnabled"))
	require.Equal(t, "GO"+"11111111111111111111111111111111", getS(t, imp.Properties, "flexInstanceSid"))
	// A program that states a setting after the import plans one harmless update, then settles.
	stated := pv(map[string]property.Value{"uiLanguage": sv("de-DE")})
	require.Equal(t, pgp.Update, l.diff(imp.ID, imp.Properties, stated, imp.Inputs).DetailedDiff["uiLanguage"].Kind)
	settled := l.update(imp.ID, imp.Properties, stated, imp.Inputs)
	requireNoChanges(t, l.diff(imp.ID, settled.Properties, stated, stated))
	requireNoChanges(t, l.diff(imp.ID, imp.Properties, imp.Inputs, imp.Inputs))
	_, err := l.s.Read(pgp.ReadRequest{ID: "other", Urn: l.urn, Properties: empty(), Inputs: empty()})
	require.ErrorContains(t, err, "import ID")

	l.f.resetCalls()
	l.del(c.ID, u.Properties)
	require.Empty(t, l.f.callsMatching("DELETE", "/"))
	require.Equal(t, true, l.f.get("flex", testAccount, "Configuration")["crm_enabled"])

	require.Len(t, l.check(pv(map[string]property.Value{"uiAttributes": sv("[1]"), "crmAttributes": sv("nope")})).Failures, 2)
	require.Empty(t, l.check(in).Failures)
}

func TestFlexConfigurationLC_EmptyProgramOnlyReads(t *testing.T) {
	l := newTL(t, "FlexConfiguration", "cfg")
	c := l.create(empty())
	require.Equal(t, "account", c.ID)
	require.Empty(t, l.f.callsMatching("POST", "/flex-api/v1/Configuration"))
	require.Equal(t, "ok", getS(t, c.Properties, "status"))
}

func TestLookup_FlexConfiguration(t *testing.T) {
	l := newTL(t, "FlexConfiguration", "cfg")
	l.create(pv(map[string]property.Value{"uiAttributes": sv(`{"a":1}`), "crmEnabled": bv(true)}))
	r, err := invoke(t, l, "getFlexConfiguration", map[string]property.Value{"uiVersion": sv("2.1.0")})
	require.NoError(t, err)
	require.Equal(t, `{"a":1}`, getS(t, r, "uiAttributes"))
	require.True(t, r.Get("crmEnabled").AsBool())
	require.Equal(t, "ok", getS(t, r, "status"))
	require.Equal(t, "https://flex.twilio.com", getS(t, r, "flexUrl"))
	require.False(t, hasProp(r, "crmType"))
	q := l.f.callsMatching("GET", "/flex-api/v1/Configuration")
	require.Equal(t, "2.1.0", q[len(q)-1].Query.Get("UiVersion"))
}
