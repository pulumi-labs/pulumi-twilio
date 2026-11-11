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

func accountDefaultInputs() property.Map {
	return pv(map[string]property.Value{
		"description":       sv("Account default"),
		"compositionPolicy": objv(map[string]property.Value{"channels": sv("mono"), "track": sv("both")}),
		"callRecordingStatusCallback": objv(map[string]property.Value{
			"url": sv("https://example.com/rec"), "events": arrv(sv("completed"))}),
	})
}

func TestVoiceAccountDefaultRecordingLC_Lifecycle(t *testing.T) {
	l := newTL(t, "VoiceAccountDefaultRecordingConfiguration", "default")
	in := accountDefaultInputs()
	c := l.create(in)
	require.Equal(t, testAccount, c.ID, "the singleton ID is the account SID")
	require.Equal(t, testAccount, getS(t, c.Properties, "accountSid"))

	// The body is the shared recording body without a unique name.
	body := lastBody(t, l, "POST", "/voice/v2/AccountDefaultConfiguration/Recording")
	require.NotContains(t, body, "unique_name")
	require.Equal(t, "Account default", body["description"])
	cfg := body["configuration"].(map[string]any)
	require.Equal(t, "Recording", cfg["configurationType"])
	require.Equal(t, "mono", cfg["compositionPolicy"].(map[string]any)["channels"])
	requireNoChanges(t, l.diff(c.ID, c.Properties, in, in))

	// A refresh sees no drift, then drift made outside Pulumi.
	rd := l.read(c.ID, c.Properties, in)
	requireNoChanges(t, l.diff(c.ID, rd.Properties, in, rd.Inputs))
	l.f.mutate("voiceaccdef", testAccount, "Recording", func(o map[string]any) {
		o["configuration"].(map[string]any)["compositionPolicy"].(map[string]any)["channels"] = "dual"
	})
	rd = l.read(c.ID, c.Properties, in)
	d := l.diff(c.ID, rd.Properties, in, rd.Inputs)
	require.True(t, d.HasChanges)

	// Update replaces the whole configuration and puts back the drifted value.
	in2 := pv(map[string]property.Value{
		"description":       sv("Account default v2"),
		"compositionPolicy": objv(map[string]property.Value{"channels": sv("dual")}),
	})
	u := l.update(c.ID, rd.Properties, in2, rd.Inputs)
	put := lastBody(t, l, "PUT", "/voice/v2/AccountDefaultConfiguration/Recording")
	require.Equal(t, "Account default v2", put["description"])
	require.NotContains(t, put["configuration"], "callRecordingStatusCallback")
	rd = l.read(c.ID, u.Properties, in2)
	requireNoChanges(t, l.diff(c.ID, rd.Properties, in2, rd.Inputs))

	// Import by account SID adopts the account default.
	imp := l.read(testAccount, empty(), empty())
	require.Equal(t, testAccount, imp.ID)
	require.Equal(t, "Account default v2", getS(t, imp.Inputs, "description"))
	requireNoChanges(t, l.diff(imp.ID, imp.Properties, imp.Inputs, imp.Inputs))

	// Delete removes it; a refresh then drops it and a second delete is quiet.
	l.del(c.ID, u.Properties)
	require.Empty(t, l.read(c.ID, u.Properties, in2).ID)
	l.requireImportFails(testAccount)
	l.del(c.ID, u.Properties)
}

func TestVoiceAccountDefaultRecordingLC_CreateAdoptsAnExistingDefault(t *testing.T) {
	l := newTL(t, "VoiceAccountDefaultRecordingConfiguration", "default")
	first := l.create(accountDefaultInputs())
	// A second program creating the resource meets the existing default (409) and replaces it.
	l2 := l.other("VoiceAccountDefaultRecordingConfiguration", "takeover")
	_ = first
	in2 := pv(map[string]property.Value{"description": sv("takeover")})
	c := l2.create(in2)
	require.Equal(t, testAccount, c.ID)
	require.Equal(t, "takeover", l.f.objs[key("voiceaccdef", testAccount, "Recording")]["description"])
	require.Len(t, l.f.callsMatching("POST", "/voice/v2/AccountDefaultConfiguration/Recording"), 2)
	require.Len(t, l.f.callsMatching("PUT", "/voice/v2/AccountDefaultConfiguration/Recording"), 1)
}

func TestVoiceAccountDefaultRecordingLC_CheckAndImportID(t *testing.T) {
	l := newTL(t, "VoiceAccountDefaultRecordingConfiguration", "default")
	chk := l.check(pv(map[string]property.Value{"compositionPolicy": objv(map[string]property.Value{"channels": sv("stereo")})}))
	require.NotEmpty(t, chk.Failures)
	_, err := l.s.Read(pgp.ReadRequest{Urn: l.urn, ID: "ACsomeoneelse", Inputs: empty(), Properties: empty()})
	require.ErrorContains(t, err, "invalid VoiceAccountDefaultRecordingConfiguration ID")
}

func TestGetVoiceAccountDefaultRecordingConfiguration(t *testing.T) {
	l := newTL(t, "VoiceAccountDefaultRecordingConfiguration", "default")
	_, err := invoke(t, l, "getVoiceAccountDefaultRecordingConfiguration", map[string]property.Value{})
	require.ErrorContains(t, err, "no default recording configuration")
	l.create(accountDefaultInputs())
	got, err := invoke(t, l, "getVoiceAccountDefaultRecordingConfiguration", map[string]property.Value{})
	require.NoError(t, err)
	require.Equal(t, testAccount, getS(t, got, "accountSid"))
	require.Equal(t, "Account default", getS(t, got, "description"))
	require.Equal(t, "mono", got.Get("compositionPolicy").AsMap().Get("channels").AsString())
}
