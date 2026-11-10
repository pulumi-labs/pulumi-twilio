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
	"encoding/json"
	"strings"
	"testing"

	pgp "github.com/pulumi/pulumi-go-provider"
	"github.com/pulumi/pulumi/sdk/v3/go/property"
	"github.com/stretchr/testify/require"
)

func objv(kv map[string]property.Value) property.Value { return property.New(property.NewMap(kv)) }

func arrv(vs ...property.Value) property.Value { return property.New(property.NewArray(vs)) }

func recordingInputs() property.Map {
	return pv(map[string]property.Value{
		"uniqueName":        sv("support-calls"),
		"description":       sv("Support line"),
		"compositionPolicy": objv(map[string]property.Value{"channels": sv("dual"), "trim": sv("trim-silence"), "track": sv("both")}),
		"callRecordingStatusCallback": objv(map[string]property.Value{
			"url": sv("https://example.com/rec"), "method": sv("POST"), "events": arrv(sv("completed"), sv("failed"))}),
		"features": arrv(objv(map[string]property.Value{"type": sv("transcriptionConfiguration"), "featureId": sv("VXabc")})),
	})
}

func lastBody(t *testing.T, l *tl, method, path string) map[string]any {
	t.Helper()
	calls := l.f.callsMatching(method, path)
	require.NotEmpty(t, calls, "%s %s", method, path)
	var m map[string]any
	require.NoError(t, json.Unmarshal([]byte(calls[len(calls)-1].Body), &m))
	return m
}

func TestVoiceRecordingConfigurationLC_Lifecycle(t *testing.T) {
	l := newTL(t, "VoiceRecordingConfiguration", "rec")
	in := recordingInputs()
	c := l.create(in)
	id := c.ID
	require.True(t, strings.HasPrefix(id, "VJ"), id)
	require.Equal(t, id, getS(t, c.Properties, "configurationId"))
	require.Equal(t, testAccount, getS(t, c.Properties, "accountSid"))

	// The body is JSON with the configuration discriminator and camelCase nested fields.
	body := lastBody(t, l, "POST", "/voice/v2/Configurations/Recording")
	require.Equal(t, "support-calls", body["unique_name"])
	cfg := body["configuration"].(map[string]any)
	require.Equal(t, "Recording", cfg["configurationType"])
	require.Equal(t, "dual", cfg["compositionPolicy"].(map[string]any)["channels"])
	require.Equal(t, []any{"completed", "failed"}, cfg["callRecordingStatusCallback"].(map[string]any)["events"])
	require.Equal(t, "VXabc", cfg["features"].([]any)[0].(map[string]any)["featureId"])
	require.NotContains(t, cfg, "conferenceRecordingStatusCallback")
	requireNoChanges(t, l.diff(id, c.Properties, in, in))

	// A refresh changes nothing, even though the fake reports the same data in another event order.
	l.f.mutate("voicerec", testAccount, id, func(o map[string]any) {
		o["configuration"].(map[string]any)["callRecordingStatusCallback"].(map[string]any)["events"] = []any{"failed", "completed"}
	})
	rd := l.read(id, c.Properties, in)
	requireNoChanges(t, l.diff(id, rd.Properties, in, rd.Inputs))

	// Out-of-band drift shows on the inputs the program set.
	l.f.mutate("voicerec", testAccount, id, func(o map[string]any) {
		o["configuration"].(map[string]any)["compositionPolicy"].(map[string]any)["channels"] = "mono"
	})
	rd = l.read(id, c.Properties, in)
	d := l.diff(id, rd.Properties, in, rd.Inputs)
	require.Equal(t, pgp.Update, d.DetailedDiff["compositionPolicy.channels"].Kind)

	// Everything updates in place with PUT; the whole configuration is sent.
	next := with(in, map[string]property.Value{
		"description":                       sv("Support line v2"),
		"conferenceRecordingStatusCallback": objv(map[string]property.Value{"url": sv("https://example.com/conf")}),
	})
	require.False(t, l.diff(id, c.Properties, next, in).DetailedDiff["description"].Kind == pgp.UpdateReplace)
	l.f.resetCalls()
	u := l.update(id, c.Properties, next, in)
	body = lastBody(t, l, "PUT", "/voice/v2/Configurations/Recording/"+id)
	require.Equal(t, "Support line v2", body["description"])
	require.Equal(t, "https://example.com/conf", body["configuration"].(map[string]any)["conferenceRecordingStatusCallback"].(map[string]any)["url"])
	require.Equal(t, "2026-11-07T00:00:00Z", getS(t, u.Properties, "dateUpdated"))
	requireNoChanges(t, l.diff(id, u.Properties, next, next))

	// Twilio defaults the method of a callback to POST: a program that omits it plans no change.
	rd = l.read(id, u.Properties, next)
	require.Equal(t, "POST", l.f.get("voicerec", testAccount, id)["configuration"].(map[string]any)["conferenceRecordingStatusCallback"].(map[string]any)["method"])
	requireNoChanges(t, l.diff(id, rd.Properties, next, rd.Inputs))

	// Removing the features and the policy clears them (they are left out of the PUT).
	gone := without(next, "features", "compositionPolicy", "conferenceRecordingStatusCallback")
	l.f.resetCalls()
	u = l.update(id, u.Properties, gone, next)
	cfg = lastBody(t, l, "PUT", "/voice/v2/Configurations/Recording/"+id)["configuration"].(map[string]any)
	require.NotContains(t, cfg, "features")
	require.NotContains(t, cfg, "compositionPolicy")

	// Gone out of band: a refresh drops it, and deleting twice is fine.
	l.f.remove("voicerec", testAccount, id)
	require.Empty(t, l.read(id, u.Properties, gone).ID)
	l.del(id, u.Properties)
	c = l.create(in)
	l.del(c.ID, c.Properties)
	require.Nil(t, l.f.get("voicerec", testAccount, c.ID))
}

func TestVoiceRecordingConfigurationLC_ImportAndLookup(t *testing.T) {
	l := newTL(t, "VoiceRecordingConfiguration", "rec")
	seed := l.other("VoiceRecordingConfiguration", "seed").create(recordingInputs())
	id := seed.ID

	rd := l.read(id, empty(), empty())
	require.Equal(t, id, rd.ID)
	require.Equal(t, "support-calls", getS(t, rd.Inputs, "uniqueName"))
	require.Equal(t, "both", rd.Inputs.Get("compositionPolicy").AsMap().Get("track").AsString())
	require.Equal(t, "transcriptionConfiguration", rd.Inputs.Get("features").AsArray().Get(0).AsMap().Get("type").AsString())
	requireNoChanges(t, l.diff(id, rd.Properties, recordingInputs(), rd.Inputs))
	l.requireImportFails("VJnosuch")

	// A configuration with nothing optional set imports without invented empty objects.
	bare := l.other("VoiceRecordingConfiguration", "bare").create(empty())
	rd = l.read(bare.ID, empty(), empty())
	_, hasPolicy := rd.Inputs.GetOk("compositionPolicy")
	require.False(t, hasPolicy)
	_, hasFeatures := rd.Inputs.GetOk("features")
	require.False(t, hasFeatures)
	requireNoChanges(t, l.diff(bare.ID, rd.Properties, empty(), rd.Inputs))

	// The lookup takes the ID or the unique name.
	for _, ref := range []string{id, "support-calls"} {
		out := invokeLookup(t, l, "getVoiceRecordingConfiguration", map[string]property.Value{"idOrUniqueName": sv(ref)})
		require.Equal(t, id, getS(t, out, "configurationId"))
		require.Equal(t, "dual", out.Get("compositionPolicy").AsMap().Get("channels").AsString())
		require.Len(t, out.Get("features").AsArray().AsSlice(), 1)
	}
	_, err := invokeLookupErr(l, "getVoiceRecordingConfiguration", map[string]property.Value{"idOrUniqueName": sv("nope")})
	require.ErrorContains(t, err, "was not found")
}

func TestVoiceRecordingConfigurationLC_Check(t *testing.T) {
	l := newTL(t, "VoiceRecordingConfiguration", "rec")
	for name, in := range map[string]property.Map{
		"bad channels": pv(map[string]property.Value{"compositionPolicy": objv(map[string]property.Value{"channels": sv("stereo")})}),
		"bad trim":     pv(map[string]property.Value{"compositionPolicy": objv(map[string]property.Value{"trim": sv("yes")})}),
		"bad method":   pv(map[string]property.Value{"callRecordingStatusCallback": objv(map[string]property.Value{"method": sv("PUT")})}),
		"bad event":    pv(map[string]property.Value{"conferenceRecordingStatusCallback": objv(map[string]property.Value{"events": arrv(sv("started"))})}),
		"empty name":   pv(map[string]property.Value{"uniqueName": sv("")}),
		"bad feature":  pv(map[string]property.Value{"features": arrv(objv(map[string]property.Value{"type": sv("other"), "featureId": sv("x")}))}),
		"no featureId": pv(map[string]property.Value{"features": arrv(objv(map[string]property.Value{"type": sv("transcriptionConfiguration"), "featureId": sv("")}))}),
		"four features": pv(map[string]property.Value{"features": arrv(
			feat("a"), feat("b"), feat("c"), feat("d"))}),
	} {
		require.NotEmpty(t, l.check(in).Failures, name)
	}
	require.Empty(t, l.check(recordingInputs()).Failures)
	require.Empty(t, l.check(empty()).Failures)
}

func feat(id string) property.Value {
	return objv(map[string]property.Value{"type": sv("transcriptionConfiguration"), "featureId": sv(id)})
}

func transcriptionInputs() property.Map {
	return pv(map[string]property.Value{
		"uniqueName":                  sv("support-transcripts"),
		"transcriptionEngine":         sv("deepgram"),
		"speechModel":                 sv("nova-3"),
		"language":                    sv("en-US"),
		"transcriptionStatusCallback": objv(map[string]property.Value{"url": sv("https://example.com/tx")}),
		"participantDefaults": arrv(
			objv(map[string]property.Value{"audioChannelIndex": property.New(float64(1)), "type": sv("CUSTOMER")}),
			objv(map[string]property.Value{"audioChannelIndex": property.New(float64(2)), "type": sv("HUMAN_AGENT")})),
	})
}

func TestVoiceTranscriptionConfigurationLC_Lifecycle(t *testing.T) {
	l := newTL(t, "VoiceTranscriptionConfiguration", "tx")
	in := transcriptionInputs()
	c := l.create(in)
	id := c.ID
	require.True(t, strings.HasPrefix(id, "VX"), id)
	cfg := lastBody(t, l, "POST", "/voice/v2/Configurations/Transcription")["configuration"].(map[string]any)
	require.Equal(t, "Transcription", cfg["configurationType"])
	require.Equal(t, "deepgram", cfg["transcriptionEngine"])
	require.Equal(t, "nova-3", cfg["speechModel"])
	require.Len(t, cfg["participantDefaults"], 2)
	require.EqualValues(t, 2, cfg["participantDefaults"].([]any)[1].(map[string]any)["audioChannelIndex"])
	require.NotContains(t, cfg, "conversationConfigurationId")

	// The fake reports the default POST method; a program that omits it still plans no change.
	rd := l.read(id, c.Properties, in)
	require.Equal(t, "POST", l.f.get("voicetx", testAccount, id)["configuration"].(map[string]any)["transcriptionStatusCallback"].(map[string]any)["method"])
	requireNoChanges(t, l.diff(id, rd.Properties, in, rd.Inputs))

	// Drift on a set input shows; updates apply in place with PUT.
	l.f.mutate("voicetx", testAccount, id, func(o map[string]any) { o["configuration"].(map[string]any)["language"] = "de-DE" })
	rd = l.read(id, c.Properties, in)
	require.Equal(t, pgp.Update, l.diff(id, rd.Properties, in, rd.Inputs).DetailedDiff["language"].Kind)

	next := with(in, map[string]property.Value{"language": sv("fr-FR"), "conversationConfigurationId": sv("conv-1"), "description": sv("French")})
	l.f.resetCalls()
	u := l.update(id, c.Properties, next, in)
	body := lastBody(t, l, "PUT", "/voice/v2/Configurations/Transcription/"+id)
	require.Equal(t, "French", body["description"])
	require.Equal(t, "fr-FR", body["configuration"].(map[string]any)["language"])
	require.Equal(t, "conv-1", body["configuration"].(map[string]any)["conversationConfigurationId"])
	require.Equal(t, "2026-11-07T00:00:00Z", getS(t, u.Properties, "dateUpdated"))
	rd = l.read(id, u.Properties, next)
	requireNoChanges(t, l.diff(id, rd.Properties, next, rd.Inputs))

	// Import by ID adopts everything Twilio reports.
	imp := l.read(id, empty(), empty())
	require.Equal(t, "fr-FR", getS(t, imp.Inputs, "language"))
	require.Equal(t, "conv-1", getS(t, imp.Inputs, "conversationConfigurationId"))
	l.requireImportFails("VXnosuch")

	// The lookup takes the unique name.
	out := invokeLookup(t, l, "getVoiceTranscriptionConfiguration", map[string]property.Value{"idOrUniqueName": sv("support-transcripts")})
	require.Equal(t, id, getS(t, out, "configurationId"))
	require.Equal(t, "nova-3", getS(t, out, "speechModel"))
	require.Len(t, out.Get("participantDefaults").AsArray().AsSlice(), 2)

	l.f.remove("voicetx", testAccount, id)
	require.Empty(t, l.read(id, u.Properties, next).ID)
	l.del(id, u.Properties)
}

func TestVoiceTranscriptionConfigurationLC_ServerRulesAndCheck(t *testing.T) {
	l := newTL(t, "VoiceTranscriptionConfiguration", "tx")
	// Twilio needs a status callback or a conversation configuration: its 400 surfaces on create.
	noTarget := without(transcriptionInputs(), "transcriptionStatusCallback")
	_, err := l.s.Create(pgp.CreateRequest{Urn: l.urn, Properties: l.checked(noTarget)})
	require.ErrorContains(t, err, "either transcriptionStatusCallback or conversationConfigurationId")

	for name, in := range map[string]property.Map{
		"empty engine":   with(transcriptionInputs(), map[string]property.Value{"transcriptionEngine": sv("")}),
		"empty language": with(transcriptionInputs(), map[string]property.Value{"language": sv("")}),
		"bad method": with(transcriptionInputs(), map[string]property.Value{
			"transcriptionStatusCallback": objv(map[string]property.Value{"url": sv("https://x"), "method": sv("GET")})}),
		"bad role": with(transcriptionInputs(), map[string]property.Value{"participantDefaults": arrv(
			objv(map[string]property.Value{"audioChannelIndex": property.New(float64(1)), "type": sv("ROBOT")}))}),
		"zero channel": with(transcriptionInputs(), map[string]property.Value{"participantDefaults": arrv(
			objv(map[string]property.Value{"audioChannelIndex": property.New(float64(0)), "type": sv("CUSTOMER")}))}),
	} {
		require.NotEmpty(t, l.check(in).Failures, name)
	}
	require.Empty(t, l.check(transcriptionInputs()).Failures)
}

func TestVoiceV2Configurations_SchemaTokens(t *testing.T) {
	prov, err := New()
	require.NoError(t, err)
	spec, err := pgp.GetSchema(context.Background(), "twilio", "0.1.0", prov)
	require.NoError(t, err)
	for _, tok := range []string{"twilio:index:VoiceRecordingConfiguration", "twilio:index:VoiceTranscriptionConfiguration"} {
		_, ok := spec.Resources[tok]
		require.True(t, ok, tok)
	}
	for _, tok := range []string{"twilio:index:getVoiceRecordingConfiguration", "twilio:index:getVoiceTranscriptionConfiguration"} {
		_, ok := spec.Functions[tok]
		require.True(t, ok, tok)
	}
}

func invokeLookup(t *testing.T, l *tl, token string, args map[string]property.Value) property.Map {
	t.Helper()
	out, err := invoke(t, l, token, args)
	require.NoError(t, err)
	return out
}

func invokeLookupErr(l *tl, token string, args map[string]property.Value) (property.Map, error) {
	return invoke(l.t, l, token, args)
}

// A recording configuration wired to a transcription configuration has an unknown featureId at
// preview (#1742). Check must let it through, and must still reject a known empty one.
func TestVoiceRecordingConfigurationLC_CheckUnknownFeatureId(t *testing.T) {
	l := newTL(t, "VoiceRecordingConfiguration", "rec")
	wired := pv(map[string]property.Value{"features": arrv(objv(map[string]property.Value{
		"type": sv("transcriptionConfiguration"), "featureId": computed()}))})
	chk := l.check(wired)
	require.Empty(t, chk.Failures, "an unknown featureId must not be validated")
	require.True(t, chk.Inputs.Get("features").AsArray().Get(0).AsMap().Get("featureId").IsComputed(),
		"the unknown must be passed through Check, not replaced by an empty string")

	// An unknown type is skipped too, and a known empty featureId next to it still fails.
	require.Empty(t, l.check(pv(map[string]property.Value{"features": arrv(objv(map[string]property.Value{
		"type": computed(), "featureId": sv("VXabc")}))})).Failures)
	require.NotEmpty(t, l.check(pv(map[string]property.Value{"features": arrv(
		objv(map[string]property.Value{"type": sv("transcriptionConfiguration"), "featureId": computed()}),
		objv(map[string]property.Value{"type": sv("transcriptionConfiguration"), "featureId": sv("")}))})).Failures,
		"a known empty featureId on another entry must still fail")
}

// The same bug class in the transcription configuration's participantDefaults: an unknown
// audioChannelIndex or type decodes to its zero value and used to fail Check.
func TestVoiceTranscriptionConfigurationLC_CheckUnknownParticipantDefaults(t *testing.T) {
	l := newTL(t, "VoiceTranscriptionConfiguration", "tx")
	for name, d := range map[string]property.Value{
		"unknown index": objv(map[string]property.Value{"audioChannelIndex": computed(), "type": sv("CUSTOMER")}),
		"unknown type":  objv(map[string]property.Value{"audioChannelIndex": property.New(float64(1)), "type": computed()}),
		"both unknown":  objv(map[string]property.Value{"audioChannelIndex": computed(), "type": computed()}),
	} {
		in := with(transcriptionInputs(), map[string]property.Value{"participantDefaults": arrv(d)})
		require.Empty(t, l.check(in).Failures, name)
	}
}

func TestUnknownAt(t *testing.T) {
	m := pv(map[string]property.Value{
		"top":  computed(),
		"list": arrv(objv(map[string]property.Value{"a": computed(), "b": sv("x")}), sv("known")),
		"obj":  objv(map[string]property.Value{"c": computed(), "d": objv(map[string]property.Value{"e": computed()})}),
	})
	for name, tc := range map[string]struct {
		path []any
		want bool
	}{
		"top level unknown":     {[]any{"top"}, true},
		"element field unknown": {[]any{"list", 0, "a"}, true},
		"element field known":   {[]any{"list", 0, "b"}, false},
		"other element":         {[]any{"list", 1}, false},
		"object field unknown":  {[]any{"obj", "c"}, true},
		"nested object unknown": {[]any{"obj", "d", "e"}, true},
		"below an unknown":      {[]any{"top", "anything"}, true},
		"missing key":           {[]any{"nope"}, false},
		"index out of range":    {[]any{"list", 5, "a"}, false},
		"index on a non array":  {[]any{"obj", 0}, false},
		"key on a non object":   {[]any{"list", 1, "x"}, false},
		"negative index":        {[]any{"list", -1}, false},
	} {
		require.Equal(t, tc.want, unknownAt(m, tc.path...), name)
	}
}
