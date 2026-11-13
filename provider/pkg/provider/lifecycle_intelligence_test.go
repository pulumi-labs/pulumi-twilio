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

func TestIntelligenceServiceLC_Lifecycle(t *testing.T) {
	l := newTL(t, "IntelligenceService", "svc")
	in := pv(map[string]property.Value{"uniqueName": sv("calls"), "friendlyName": sv("Calls"), "languageCode": sv("de-DE"), "autoTranscribe": bv(true)})
	c := l.create(in)
	require.True(t, strings.HasPrefix(c.ID, "GA"), c.ID)
	require.Equal(t, c.ID, getS(t, c.Properties, "sid"))
	form := l.f.callsMatching("POST", "/intelligence/v2/Services")[0].Form
	require.Equal(t, "calls", form.Get("UniqueName"))
	require.Equal(t, "de-DE", form.Get("LanguageCode"))
	require.Equal(t, "true", form.Get("AutoTranscribe"))
	require.Empty(t, form["WebhookUrl"])
	require.True(t, c.Properties.Get("dataLogging").AsBool())
	requireNoChanges(t, l.diff(c.ID, c.Properties, in, in))
	rd := l.read(c.ID, c.Properties, in)
	requireNoChanges(t, l.diff(c.ID, rd.Properties, in, rd.Inputs))

	// Drift in a setting the program states shows; a setting it leaves out does not.
	l.f.mutate("iq", testAccount, "Services/"+c.ID, func(o map[string]any) { o["auto_transcribe"] = false; o["auto_redaction"] = true })
	rd = l.read(c.ID, c.Properties, in)
	require.False(t, rd.Inputs.Get("autoTranscribe").AsBool())
	require.Equal(t, pgp.Update, l.diff(c.ID, rd.Properties, in, rd.Inputs).DetailedDiff["autoTranscribe"].Kind)
	require.False(t, hasProp(rd.Inputs, "autoRedaction"))

	next := with(in, map[string]property.Value{"friendlyName": sv("Calls 2"), "webhookUrl": sv("https://example.com/done"), "webhookHttpMethod": sv("POST")})
	d := l.diff(c.ID, c.Properties, next, in)
	require.Equal(t, pgp.Update, d.DetailedDiff["friendlyName"].Kind)
	require.Equal(t, pgp.Update, d.DetailedDiff["webhookUrl"].Kind)
	l.f.resetCalls()
	u := l.update(c.ID, c.Properties, next, in)
	uc := l.f.callsMatching("POST", "/intelligence/v2/Services/"+c.ID)[0].Form
	require.Equal(t, "Calls 2", uc.Get("FriendlyName"))
	require.Empty(t, uc["LanguageCode"])
	require.Equal(t, "POST", l.f.get("iq", testAccount, "Services/"+c.ID)["webhook_http_method"])
	requireNoChanges(t, l.diff(c.ID, u.Properties, next, next))

	// The language cannot change in place.
	require.Equal(t, pgp.UpdateReplace, l.diff(c.ID, u.Properties, with(next, map[string]property.Value{"languageCode": sv("fr-FR")}), next).DetailedDiff["languageCode"].Kind)

	// Import adopts what differs from the defaults; the default language stated explicitly is no change.
	imp := l.read(c.ID, empty(), empty())
	require.Empty(t, l.check(imp.Inputs).Failures)
	require.Equal(t, "calls", getS(t, imp.Inputs, "uniqueName"))
	require.Equal(t, "de-DE", getS(t, imp.Inputs, "languageCode"))
	require.False(t, hasProp(imp.Inputs, "mediaRedaction"))
	require.True(t, imp.Inputs.Get("autoRedaction").AsBool())
	l.f.mutate("iq", testAccount, "Services/"+c.ID, func(o map[string]any) { o["language_code"] = "en-US"; o["auto_transcribe"] = false })
	imp = l.read(c.ID, empty(), empty())
	require.False(t, hasProp(imp.Inputs, "languageCode"))
	explicit := pv(map[string]property.Value{"uniqueName": sv("calls"), "languageCode": sv("en-US"), "autoTranscribe": bv(false), "mediaRedaction": bv(false)})
	requireNoChanges(t, l.diff(c.ID, imp.Properties, explicit, imp.Inputs))
	l.requireImportFails("GA" + strings.Repeat("0", 32))

	l.f.remove("iq", testAccount, "Services/"+c.ID)
	require.Empty(t, l.read(c.ID, c.Properties, in).ID)
	l.del(c.ID, c.Properties)
	c = l.create(in)
	l.del(c.ID, c.Properties)
	require.Nil(t, l.f.get("iq", testAccount, "Services/"+c.ID))

	bad := l.check(pv(map[string]property.Value{"uniqueName": sv(""), "friendlyName": sv(strings.Repeat("x", 65)), "webhookHttpMethod": sv("PUT"), "encryptionCredentialSid": sv("nope")}))
	require.Len(t, bad.Failures, 4)
}

func TestIntelligenceCustomOperatorLC_Lifecycle(t *testing.T) {
	l := newTL(t, "IntelligenceCustomOperator", "op")
	in := pv(map[string]property.Value{"friendlyName": sv("Sentiment"), "operatorType": sv("Extract"), "config": sv(`{"prompt":"a","labels":["x","y"]}`)})
	c := l.create(in)
	require.True(t, strings.HasPrefix(c.ID, "LY"), c.ID)
	form := l.f.callsMatching("POST", "/intelligence/v2/Operators/Custom")[0].Form
	require.Equal(t, "Extract", form.Get("OperatorType"))
	require.JSONEq(t, `{"prompt":"a","labels":["x","y"]}`, form.Get("Config"))
	require.Equal(t, "internal", getS(t, c.Properties, "availability"))
	requireNoChanges(t, l.diff(c.ID, c.Properties, in, in))

	// Twilio returns the configuration as an object with its own key order; that is not drift.
	rd := l.read(c.ID, c.Properties, in)
	require.Equal(t, `{"prompt":"a","labels":["x","y"]}`, getS(t, rd.Inputs, "config"))
	requireNoChanges(t, l.diff(c.ID, rd.Properties, in, rd.Inputs))
	reordered := with(in, map[string]property.Value{"config": sv("{ \"labels\": [\"x\", \"y\"], \"prompt\": \"a\" }")})
	requireNoChanges(t, l.diff(c.ID, c.Properties, reordered, in))
	l.f.mutate("iq", testAccount, "Operators/Custom/"+c.ID, func(o map[string]any) { o["config"] = map[string]any{"prompt": "drifted"} })
	rd = l.read(c.ID, c.Properties, in)
	require.JSONEq(t, `{"prompt":"drifted"}`, getS(t, rd.Inputs, "config"))
	require.Equal(t, pgp.Update, l.diff(c.ID, rd.Properties, in, rd.Inputs).DetailedDiff["config"].Kind)

	next := with(in, map[string]property.Value{"friendlyName": sv("Sentiment 2"), "config": sv(`{"prompt":"b"}`)})
	d := l.diff(c.ID, c.Properties, next, in)
	require.Equal(t, pgp.Update, d.DetailedDiff["friendlyName"].Kind)
	require.Equal(t, pgp.Update, d.DetailedDiff["config"].Kind)
	l.f.resetCalls()
	u := l.update(c.ID, c.Properties, next, in)
	uf := l.f.callsMatching("POST", "/intelligence/v2/Operators/Custom/"+c.ID)[0].Form
	require.Equal(t, "Sentiment 2", uf.Get("FriendlyName"))
	require.Empty(t, uf["OperatorType"])
	requireNoChanges(t, l.diff(c.ID, u.Properties, next, next))
	require.Equal(t, pgp.UpdateReplace, l.diff(c.ID, u.Properties, with(next, map[string]property.Value{"operatorType": sv("Classify")}), next).DetailedDiff["operatorType"].Kind)

	imp := l.read(c.ID, empty(), empty())
	require.Empty(t, l.check(imp.Inputs).Failures)
	require.Equal(t, "Sentiment 2", getS(t, imp.Inputs, "friendlyName"))
	require.Equal(t, "Extract", getS(t, imp.Inputs, "operatorType"))
	require.JSONEq(t, `{"prompt":"b"}`, getS(t, imp.Inputs, "config"))
	l.requireImportFails("LY" + strings.Repeat("0", 32))

	l.f.remove("iq", testAccount, "Operators/Custom/"+c.ID)
	require.Empty(t, l.read(c.ID, u.Properties, next).ID)
	l.del(c.ID, u.Properties)
	c = l.create(in)
	l.del(c.ID, c.Properties)
	require.Nil(t, l.f.get("iq", testAccount, "Operators/Custom/"+c.ID))

	bad := l.check(pv(map[string]property.Value{"friendlyName": sv(strings.Repeat("x", 65)), "operatorType": sv(""), "config": sv("[1]")}))
	require.Len(t, bad.Failures, 3)
}

func TestLookup_Intelligence(t *testing.T) {
	l := newTL(t, "IntelligenceService", "svc")
	svc := l.create(pv(map[string]property.Value{"uniqueName": sv("calls"), "friendlyName": sv("Calls")})).ID
	r, err := invoke(t, l, "getIntelligenceService", map[string]property.Value{"sid": sv(svc)})
	require.NoError(t, err)
	require.Equal(t, "calls", getS(t, r, "uniqueName"))
	require.Equal(t, "en-US", getS(t, r, "languageCode"))
	require.False(t, r.Get("autoTranscribe").AsBool())
	require.False(t, hasProp(r, "webhookUrl"))
	_, err = invoke(t, l, "getIntelligenceService", map[string]property.Value{"sid": sv("GAmissing")})
	require.ErrorContains(t, err, "was not found")
	_, err = invoke(t, l, "getIntelligenceService", map[string]property.Value{"sid": sv("")})
	require.ErrorContains(t, err, "must be set")

	op := l.other("IntelligenceCustomOperator", "op").create(pv(map[string]property.Value{"friendlyName": sv("Sentiment"), "operatorType": sv("Extract"), "config": sv(`{"prompt":"a"}`)})).ID
	r, err = invoke(t, l, "getIntelligenceCustomOperator", map[string]property.Value{"sid": sv(op)})
	require.NoError(t, err)
	require.Equal(t, "Extract", getS(t, r, "operatorType"))
	require.JSONEq(t, `{"prompt":"a"}`, getS(t, r, "config"))
	_, err = invoke(t, l, "getIntelligenceCustomOperator", map[string]property.Value{"sid": sv("LYmissing")})
	require.ErrorContains(t, err, "was not found")
	_, err = invoke(t, l, "getIntelligenceCustomOperator", map[string]property.Value{"sid": sv("")})
	require.ErrorContains(t, err, "must be set")
}
