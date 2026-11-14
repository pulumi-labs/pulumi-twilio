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

func TestVideoCompositionHookLC_Lifecycle(t *testing.T) {
	l := newTL(t, "VideoCompositionHook", "hook")
	in := pv(map[string]property.Value{"friendlyName": sv("recap"), "enabled": bv(true), "audioSources": strList("student*", "teacher"),
		"videoLayout": sv(`{"grid":{"video_sources":["*"]}}`), "resolution": sv("1280x720"), "format": sv("webm"), "trim": bv(false)})
	c := l.create(in)
	require.True(t, strings.HasPrefix(c.ID, "HK"), c.ID)
	require.Equal(t, c.ID, getS(t, c.Properties, "sid"))
	form := l.f.callsMatching("POST", "/video/v1/CompositionHooks")[0].Form
	require.Equal(t, "recap", form.Get("FriendlyName"))
	require.Equal(t, []string{"student*", "teacher"}, form["AudioSources"])
	require.Equal(t, "1280x720", form.Get("Resolution"))
	require.Equal(t, "false", form.Get("Trim"))
	require.Empty(t, form["StatusCallback"])
	require.Empty(t, form["AudioSourcesExcluded"])
	requireNoChanges(t, l.diff(c.ID, c.Properties, in, in))

	// Twilio reports the layout as an object and the track names in its own order; neither is drift.
	l.f.mutate("vd", testAccount, "CompositionHooks/"+c.ID, func(o map[string]any) { o["audio_sources"] = []any{"teacher", "student*"} })
	rd := l.read(c.ID, c.Properties, in)
	requireNoChanges(t, l.diff(c.ID, rd.Properties, in, rd.Inputs))
	require.JSONEq(t, `{"grid":{"video_sources":["*"]}}`, getS(t, rd.Inputs, "videoLayout"))

	// Drift in a setting the program states shows; a setting it leaves out does not.
	l.f.mutate("vd", testAccount, "CompositionHooks/"+c.ID, func(o map[string]any) { o["trim"] = true; o["status_callback"] = "https://example.com/x" })
	rd = l.read(c.ID, c.Properties, in)
	require.True(t, rd.Inputs.Get("trim").AsBool())
	require.Equal(t, pgp.Update, l.diff(c.ID, rd.Properties, in, rd.Inputs).DetailedDiff["trim"].Kind)
	require.False(t, hasProp(rd.Inputs, "statusCallback"))

	next := with(in, map[string]property.Value{"friendlyName": sv("recap 2"), "enabled": bv(false), "audioSources": strList("*"),
		"audioSourcesExcluded": strList("host"), "statusCallback": sv("https://example.com/cb"), "statusCallbackMethod": sv("GET")})
	d := l.diff(c.ID, c.Properties, next, in)
	for _, k := range []string{"friendlyName", "enabled", "audioSources", "audioSourcesExcluded", "statusCallback", "statusCallbackMethod"} {
		require.Equal(t, pgp.Update, d.DetailedDiff[k].Kind, k)
	}
	l.f.resetCalls()
	u := l.update(c.ID, c.Properties, next, in)
	uf := l.f.callsMatching("POST", "/video/v1/CompositionHooks/"+c.ID)[0].Form
	require.Equal(t, []string{"host"}, uf["AudioSourcesExcluded"])
	require.Equal(t, "GET", l.f.get("vd", testAccount, "CompositionHooks/"+c.ID)["status_callback_method"])
	requireNoChanges(t, l.diff(c.ID, u.Properties, next, next))

	// An empty list is sent as one empty value and clears the setting.
	cleared := with(next, map[string]property.Value{"audioSourcesExcluded": strList()})
	require.Equal(t, pgp.Update, l.diff(c.ID, u.Properties, cleared, next).DetailedDiff["audioSourcesExcluded"].Kind)
	u = l.update(c.ID, u.Properties, cleared, next)
	require.Empty(t, l.f.get("vd", testAccount, "CompositionHooks/"+c.ID)["audio_sources_excluded"])
	requireNoChanges(t, l.diff(c.ID, u.Properties, cleared, cleared))

	// Import adopts what differs from the defaults; the defaults stated explicitly are no change.
	l.f.mutate("vd", testAccount, "CompositionHooks/"+c.ID, func(o map[string]any) {
		o["resolution"] = "640x480"
		o["trim"] = true
		o["status_callback_method"] = "POST"
	})
	imp := l.read(c.ID, empty(), empty())
	require.Empty(t, l.check(imp.Inputs).Failures)
	require.Equal(t, "recap 2", getS(t, imp.Inputs, "friendlyName"))
	require.Equal(t, "webm", getS(t, imp.Inputs, "format"))
	require.False(t, imp.Inputs.Get("enabled").AsBool())
	require.False(t, hasProp(imp.Inputs, "resolution"))
	require.False(t, hasProp(imp.Inputs, "trim"))
	require.False(t, hasProp(imp.Inputs, "statusCallbackMethod"))
	require.False(t, hasProp(imp.Inputs, "audioSourcesExcluded"))
	explicit := with(imp.Inputs, map[string]property.Value{"resolution": sv("640x480"), "trim": bv(true), "statusCallbackMethod": sv("POST"), "audioSourcesExcluded": strList()})
	requireNoChanges(t, l.diff(c.ID, imp.Properties, explicit, imp.Inputs))
	l.requireImportFails("HK" + strings.Repeat("0", 32))

	l.f.remove("vd", testAccount, "CompositionHooks/"+c.ID)
	require.Empty(t, l.read(c.ID, u.Properties, cleared).ID)
	l.del(c.ID, u.Properties)
	c = l.create(in)
	l.del(c.ID, c.Properties)
	require.Nil(t, l.f.get("vd", testAccount, "CompositionHooks/"+c.ID))

	bad := l.check(pv(map[string]property.Value{"friendlyName": sv(strings.Repeat("x", 101)), "videoLayout": sv("[1]"), "resolution": sv("2000x2000"),
		"format": sv("avi"), "statusCallbackMethod": sv("PUT")}))
	require.Len(t, bad.Failures, 5)
	require.Len(t, l.check(pv(map[string]property.Value{"friendlyName": sv("x"), "resolution": sv("1280x1280")})).Failures, 1)
	require.Empty(t, l.check(pv(map[string]property.Value{"friendlyName": sv("x"), "resolution": sv("1280x720")})).Failures)
}

func TestLookup_Video(t *testing.T) {
	l := newTL(t, "VideoCompositionHook", "hook")
	id := l.create(pv(map[string]property.Value{"friendlyName": sv("recap"), "audioSources": strList("*")})).ID
	r, err := invoke(t, l, "getVideoCompositionHook", map[string]property.Value{"sid": sv(id)})
	require.NoError(t, err)
	require.Equal(t, "recap", getS(t, r, "friendlyName"))
	require.Equal(t, "640x480", getS(t, r, "resolution"))
	require.True(t, r.Get("trim").AsBool())
	require.False(t, hasProp(r, "videoLayout"))
	require.False(t, hasProp(r, "statusCallback"))
	require.Len(t, r.Get("audioSources").AsArray().AsSlice(), 1)
	_, err = invoke(t, l, "getVideoCompositionHook", map[string]property.Value{"sid": sv("HKmissing")})
	require.ErrorContains(t, err, "was not found")
	_, err = invoke(t, l, "getVideoCompositionHook", map[string]property.Value{"sid": sv("")})
	require.ErrorContains(t, err, "must be set")
}

func TestVideoSettingsLC_Lifecycle(t *testing.T) {
	for _, tc := range []struct{ typ, name, path, lookup string }{
		{"VideoCompositionSettings", "CompositionSettings", "/video/v1/CompositionSettings/Default", "getVideoCompositionSettings"},
		{"VideoRecordingSettings", "RecordingSettings", "/video/v1/RecordingSettings/Default", "getVideoRecordingSettings"},
	} {
		t.Run(tc.typ, func(t *testing.T) {
			l := newTL(t, tc.typ, "settings")
			cred, bucket := "CR"+strings.Repeat("a", 32), "https://my-bucket.s3.amazonaws.com/videos"
			in := pv(map[string]property.Value{"friendlyName": sv("storage"), "awsCredentialsSid": sv(cred), "awsS3Url": sv(bucket), "awsStorageEnabled": bv(true)})
			c := l.create(in)
			require.Equal(t, "default", c.ID)
			form := l.f.callsMatching("POST", tc.path)[0].Form
			require.Equal(t, "storage", form.Get("FriendlyName"))
			require.Equal(t, cred, form.Get("AwsCredentialsSid"))
			require.Equal(t, "true", form.Get("AwsStorageEnabled"))
			require.Empty(t, form["EncryptionKeySid"])
			requireNoChanges(t, l.diff(c.ID, c.Properties, in, in))
			rd := l.read(c.ID, c.Properties, in)
			requireNoChanges(t, l.diff(c.ID, rd.Properties, in, rd.Inputs))

			// Drift in a setting the program states shows; a setting it leaves out does not.
			l.f.mutate("vd", testAccount, tc.name+"/Default", func(o map[string]any) { o["aws_storage_enabled"] = false; o["encryption_enabled"] = true })
			rd = l.read(c.ID, c.Properties, in)
			require.False(t, rd.Inputs.Get("awsStorageEnabled").AsBool())
			require.Equal(t, pgp.Update, l.diff(c.ID, rd.Properties, in, rd.Inputs).DetailedDiff["awsStorageEnabled"].Kind)
			require.False(t, hasProp(rd.Inputs, "encryptionEnabled"))
			requireNoChanges(t, l.diff(c.ID, c.Properties, without(in, "awsS3Url"), in))

			key := "CR" + strings.Repeat("b", 32)
			next := with(in, map[string]property.Value{"friendlyName": sv("storage 2"), "encryptionKeySid": sv(key), "encryptionEnabled": bv(true)})
			d := l.diff(c.ID, c.Properties, next, in)
			require.Equal(t, pgp.Update, d.DetailedDiff["friendlyName"].Kind)
			require.Equal(t, pgp.Update, d.DetailedDiff["encryptionKeySid"].Kind)
			l.f.resetCalls()
			u := l.update(c.ID, c.Properties, next, in)
			require.Equal(t, key, l.f.callsMatching("POST", tc.path)[0].Form.Get("EncryptionKeySid"))
			requireNoChanges(t, l.diff(c.ID, u.Properties, next, next))

			// Import adopts what differs from the defaults; the defaults stated explicitly are no change.
			imp := l.read("default", empty(), empty())
			require.Empty(t, l.check(imp.Inputs).Failures)
			require.Equal(t, "storage 2", getS(t, imp.Inputs, "friendlyName"))
			require.Equal(t, bucket, getS(t, imp.Inputs, "awsS3Url"))
			require.True(t, imp.Inputs.Get("encryptionEnabled").AsBool())
			l.f.mutate("vd", testAccount, tc.name+"/Default", func(o map[string]any) { o["aws_storage_enabled"] = false })
			imp = l.read("default", empty(), empty())
			require.False(t, hasProp(imp.Inputs, "awsStorageEnabled"))
			requireNoChanges(t, l.diff("default", imp.Properties, with(imp.Inputs, map[string]property.Value{"awsStorageEnabled": bv(false)}), imp.Inputs))
			_, err := l.s.Read(pgp.ReadRequest{ID: "other", Urn: l.urn, Properties: empty(), Inputs: empty()})
			require.ErrorContains(t, err, "import ID")

			// Destroy forgets the settings and leaves them on Twilio.
			l.f.resetCalls()
			l.del(c.ID, u.Properties)
			require.Empty(t, l.f.callsMatching("DELETE", "/"))
			require.Equal(t, "storage 2", l.f.get("vd", testAccount, tc.name+"/Default")["friendly_name"])

			bad := l.check(pv(map[string]property.Value{"friendlyName": sv(""), "awsCredentialsSid": sv("nope"), "encryptionKeySid": sv("nope"), "awsS3Url": sv("bucket/path")}))
			require.Len(t, bad.Failures, 4)

			r, err := invoke(t, l, tc.lookup, map[string]property.Value{})
			require.NoError(t, err)
			require.Equal(t, "storage 2", getS(t, r, "friendlyName"))
			require.Equal(t, cred, getS(t, r, "awsCredentialsSid"))
			require.Equal(t, key, getS(t, r, "encryptionKeySid"))
			require.True(t, r.Get("encryptionEnabled").AsBool())
		})
	}
}

// Twilio fills in its defaults for every region of a video layout it reports (#1937); a layout the
// program wrote without them must not read as drift, while a real change must.
func TestVideoCompositionHookLC_LayoutDefaultsAreNotDrift(t *testing.T) {
	expanded := func(o map[string]any) {
		o["video_layout"] = map[string]any{"grid": map[string]any{
			"video_sources": []any{"*"}, "cells_excluded": []any{}, "reuse": "show_oldest",
			"video_sources_excluded": []any{}, "x_pos": float64(0), "y_pos": float64(0), "z_pos": float64(0),
			"height": nil, "width": nil, "max_rows": nil, "max_columns": nil,
		}}
	}
	l := newTL(t, "VideoCompositionHook", "hook")
	in := pv(map[string]property.Value{"friendlyName": sv("recap"), "enabled": bv(false), "audioSources": strList("*"),
		"videoLayout": sv(`{"grid":{"video_sources":["*"]}}`), "resolution": sv("1280x720"), "format": sv("mp4")})
	c := l.create(in)

	l.f.mutate("vd", testAccount, "CompositionHooks/"+c.ID, expanded)
	rd := l.read(c.ID, c.Properties, in)
	// The state keeps the program's form, so the next preview plans nothing.
	require.JSONEq(t, `{"grid":{"video_sources":["*"]}}`, getS(t, rd.Inputs, "videoLayout"))
	requireNoChanges(t, l.diff(c.ID, rd.Properties, in, rd.Inputs))

	// State written by an earlier release already holds Twilio's expanded layout; the program is still no change.
	old := with(c.Properties, map[string]property.Value{"videoLayout": sv(`{"grid":{"video_sources":["*"],"cells_excluded":[],"reuse":"show_oldest","x_pos":0,"y_pos":0,"z_pos":0,"video_sources_excluded":[],"height":null}}`)})
	requireNoChanges(t, l.diff(c.ID, old, in, old))

	// A program that states the defaults itself is no change against Twilio's report.
	explicit := with(in, map[string]property.Value{"videoLayout": sv(`{"grid":{"video_sources":["*"],"reuse":"show_oldest","x_pos":0}}`)})
	requireNoChanges(t, l.diff(c.ID, rd.Properties, explicit, rd.Inputs))

	// Non-default values are real settings: a change shows, in the diff and on refresh.
	changed := with(in, map[string]property.Value{"videoLayout": sv(`{"grid":{"video_sources":["*"],"reuse":"show_newest"}}`)})
	require.Equal(t, pgp.Update, l.diff(c.ID, rd.Properties, changed, rd.Inputs).DetailedDiff["videoLayout"].Kind)
	l.f.mutate("vd", testAccount, "CompositionHooks/"+c.ID, func(o map[string]any) {
		expanded(o)
		o["video_layout"].(map[string]any)["grid"].(map[string]any)["x_pos"] = float64(5)
	})
	drift := l.read(c.ID, c.Properties, in)
	require.Contains(t, getS(t, drift.Inputs, "videoLayout"), `"x_pos":5`)
	require.Equal(t, pgp.Update, l.diff(c.ID, drift.Properties, in, drift.Inputs).DetailedDiff["videoLayout"].Kind)

	// An import adopts the layout without Twilio's defaults.
	l.f.mutate("vd", testAccount, "CompositionHooks/"+c.ID, expanded)
	imp := l.read(c.ID, empty(), empty())
	require.JSONEq(t, `{"grid":{"video_sources":["*"]}}`, getS(t, imp.Inputs, "videoLayout"))
	requireNoChanges(t, l.diff(c.ID, imp.Properties, in, imp.Inputs))
}

func TestVideoLayoutSame(t *testing.T) {
	for _, tc := range []struct {
		name, a, b string
		same       bool
	}{
		{"identical", `{"grid":{"video_sources":["*"]}}`, `{"grid":{"video_sources":["*"]}}`, true},
		{"key order", `{"a":{"x_pos":1,"y_pos":2}}`, `{"a":{"y_pos":2,"x_pos":1}}`, true},
		{"defaults and nulls", `{"grid":{"video_sources":["*"]}}`, `{"grid":{"video_sources":["*"],"cells_excluded":[],"reuse":"show_oldest","z_pos":0,"width":null}}`, true},
		{"non-default reuse", `{"grid":{"video_sources":["*"]}}`, `{"grid":{"video_sources":["*"],"reuse":"show_newest"}}`, false},
		{"non-default position", `{"grid":{}}`, `{"grid":{"x_pos":3}}`, false},
		{"non-empty exclusions", `{"grid":{}}`, `{"grid":{"cells_excluded":[1]}}`, false},
		{"different sources", `{"grid":{"video_sources":["a"]}}`, `{"grid":{"video_sources":["b"]}}`, false},
		{"different region", `{"grid":{"video_sources":["*"]}}`, `{"pip":{"video_sources":["*"]}}`, false},
		{"not an object", `[1]`, `[1,2]`, false},
	} {
		t.Run(tc.name, func(t *testing.T) { require.Equal(t, tc.same, videoLayoutSame(tc.a, tc.b)) })
	}
}
