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

func recInputs(name string) property.Map {
	return pv(map[string]property.Value{"uniqueName": sv(name), "compositionPolicy": objv(map[string]property.Value{"channels": sv("mono")})})
}

func txInputs(name string) property.Map {
	return pv(map[string]property.Value{
		"uniqueName": sv(name), "transcriptionEngine": sv("google"), "language": sv("en-US"),
		"conversationConfigurationId": sv("conv-1"),
		"participantDefaults": arrv(
			objv(map[string]property.Value{"audioChannelIndex": property.New(float64(1)), "type": sv("CUSTOMER")}),
			objv(map[string]property.Value{"audioChannelIndex": property.New(float64(2)), "type": sv("HUMAN_AGENT")})),
	})
}

func TestVoiceDefaultRecordingLC_Lifecycle(t *testing.T) {
	base := newTL(t, "VoiceRecordingConfiguration", "a")
	a := base.create(recInputs("first"))
	b := base.other("VoiceRecordingConfiguration", "b").create(recInputs("second"))
	idA, idB := getS(t, a.Properties, "configurationId"), getS(t, b.Properties, "configurationId")

	l := base.other("VoiceDefaultRecordingConfiguration", "default")
	in := pv(map[string]property.Value{"configurationId": sv(idA)})
	c := l.create(in)
	require.Equal(t, testAccount, c.ID, "the singleton ID is the account SID")
	require.Equal(t, testAccount, getS(t, c.Properties, "accountSid"))
	post := lastBody(t, l, "POST", "/voice/v2/Configurations/Recording/Default")
	require.Equal(t, map[string]any{"configuration_id": idA}, post)
	requireNoChanges(t, l.diff(c.ID, c.Properties, in, in))

	// Refresh sees no drift, then drift made outside Pulumi.
	rd := l.read(c.ID, c.Properties, in)
	requireNoChanges(t, l.diff(c.ID, rd.Properties, in, rd.Inputs))
	l.f.put("voicedefault", testAccount, "Recording", map[string]any{"id": idB})
	rd = l.read(c.ID, c.Properties, in)
	require.Equal(t, idB, getS(t, rd.Inputs, "configurationId"))
	require.True(t, l.diff(c.ID, rd.Properties, in, rd.Inputs).HasChanges)

	// Update puts the program's selection back.
	u := l.update(c.ID, rd.Properties, in, rd.Inputs)
	require.Equal(t, idA, getS(t, u.Properties, "configurationId"))
	require.Equal(t, idA, l.f.objs[key("voicedefault", testAccount, "Recording")]["id"])

	// Import by account SID adopts the default.
	imp := l.read(testAccount, empty(), empty())
	require.Equal(t, idA, getS(t, imp.Inputs, "configurationId"))
	requireNoChanges(t, l.diff(imp.ID, imp.Properties, imp.Inputs, imp.Inputs))

	// Delete only forgets the selection: no unset call exists.
	l.del(c.ID, u.Properties)
	require.Empty(t, l.f.callsMatching("DELETE", "/voice/v2/Configurations/Recording/Default"))
	require.Equal(t, idA, l.f.objs[key("voicedefault", testAccount, "Recording")]["id"])

	// A default that is gone refreshes away and cannot be imported.
	l.f.del("voicedefault", testAccount, "Recording")
	require.Empty(t, l.read(c.ID, u.Properties, in).ID)
	l.requireImportFails(testAccount)
}

func TestVoiceDefaultRecordingLC_UnknownConfigurationFails(t *testing.T) {
	l := newTL(t, "VoiceDefaultRecordingConfiguration", "default")
	_, err := l.s.Create(pgp.CreateRequest{Urn: l.urn, Properties: pv(map[string]property.Value{"configurationId": sv("VJnone")})})
	require.Error(t, err)
	_, err = l.s.Read(pgp.ReadRequest{Urn: l.urn, ID: "ACsomeoneelse", Inputs: empty(), Properties: empty()})
	require.ErrorContains(t, err, "invalid VoiceDefaultRecordingConfiguration ID")
}

func TestVoiceDefaultTranscriptionLC_LifecycleAndLookup(t *testing.T) {
	base := newTL(t, "VoiceTranscriptionConfiguration", "a")
	a := base.create(txInputs("tx-first"))
	id := getS(t, a.Properties, "configurationId")
	l := base.other("VoiceDefaultTranscriptionConfiguration", "default")

	_, err := invoke(t, l, "getVoiceDefaultTranscriptionConfiguration", map[string]property.Value{})
	require.ErrorContains(t, err, "no default transcription configuration")

	in := pv(map[string]property.Value{"configurationId": sv(id)})
	c := l.create(in)
	require.Equal(t, testAccount, c.ID)
	require.Equal(t, map[string]any{"configuration_id": id}, lastBody(t, l, "POST", "/voice/v2/Configurations/Transcription/Default"))
	requireNoChanges(t, l.diff(c.ID, c.Properties, in, in))

	got, err := invoke(t, l, "getVoiceDefaultTranscriptionConfiguration", map[string]property.Value{})
	require.NoError(t, err)
	require.Equal(t, id, getS(t, got, "configurationId"))
	require.Equal(t, "tx-first", getS(t, got, "uniqueName"))
	require.Equal(t, testAccount, getS(t, got, "accountSid"))

	l.del(c.ID, c.Properties)
}

func TestGetVoiceDefaultRecordingConfiguration(t *testing.T) {
	base := newTL(t, "VoiceRecordingConfiguration", "a")
	_, err := invoke(t, base, "getVoiceDefaultRecordingConfiguration", map[string]property.Value{})
	require.ErrorContains(t, err, "no default recording configuration")
	a := base.create(recInputs("only"))
	base.other("VoiceDefaultRecordingConfiguration", "d").create(pv(map[string]property.Value{"configurationId": sv(getS(t, a.Properties, "configurationId"))}))
	got, err := invoke(t, base, "getVoiceDefaultRecordingConfiguration", map[string]property.Value{})
	require.NoError(t, err)
	require.Equal(t, "only", getS(t, got, "uniqueName"))
}
