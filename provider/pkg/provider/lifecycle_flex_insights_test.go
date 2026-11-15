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

const flexInsightsBase = "/flex-api/v1/Insights/QualityManagement/"

func insightsCategory(t *testing.T, l *tl, name string) string {
	t.Helper()
	return l.other("FlexInsightsCategory", "c-"+name).create(pv(map[string]property.Value{"name": sv(name)})).ID
}

func insightsQuestion(t *testing.T, l *tl, cat, name string) string {
	t.Helper()
	return l.other("FlexInsightsQuestion", "q-"+name).create(pv(map[string]property.Value{
		"categorySid": sv(cat), "question": sv(name), "answerSetId": sv("yes-no"), "allowNa": bv(false)})).ID
}

func TestFlexInsightsCategoryLC_Lifecycle(t *testing.T) {
	l := newTL(t, "FlexInsightsCategory", "c")
	in := pv(map[string]property.Value{"name": sv("Greeting")})
	c := l.create(in)
	require.True(t, strings.HasPrefix(c.ID, "FY"))
	require.Equal(t, c.ID, getS(t, c.Properties, "sid"))
	require.Equal(t, "Greeting", l.f.get("flex", testAccount, "Insights/Categories/"+c.ID)["name"])
	requireNoChanges(t, l.diff(c.ID, c.Properties, in, in))

	next := pv(map[string]property.Value{"name": sv("Opening")})
	require.Equal(t, pgp.Update, l.diff(c.ID, c.Properties, next, in).DetailedDiff["name"].Kind)
	u := l.update(c.ID, c.Properties, next, in)
	require.Equal(t, "Opening", l.f.get("flex", testAccount, "Insights/Categories/"+c.ID)["name"])
	require.Len(t, l.f.callsMatching("POST", flexInsightsBase+"Categories/"+c.ID), 1)
	requireNoChanges(t, l.diff(c.ID, u.Properties, next, next))

	// Drift on the name is read back through the list.
	l.f.mutate("flex", testAccount, "Insights/Categories/"+c.ID, func(o map[string]any) { o["name"] = "Renamed" })
	rd := l.read(c.ID, u.Properties, next)
	require.Equal(t, "Renamed", getS(t, rd.Inputs, "name"))
	require.Equal(t, pgp.Update, l.diff(c.ID, rd.Properties, next, rd.Inputs).DetailedDiff["name"].Kind)

	// Import by SID alone, found by scanning the list past the first page.
	for _, n := range []string{"b", "c", "d"} {
		insightsCategory(t, l, n)
	}
	imp := l.read(c.ID, empty(), empty())
	require.Equal(t, c.ID, imp.ID)
	require.Equal(t, "Renamed", getS(t, imp.Inputs, "name"))
	requireNoChanges(t, l.diff(imp.ID, imp.Properties, imp.Inputs, imp.Inputs))
	l.requireImportFails("FY" + strings.Repeat("9", 32))

	// Deleted outside: a refresh drops it.
	l.f.remove("flex", testAccount, "Insights/Categories/"+c.ID)
	gone := l.read(c.ID, u.Properties, next)
	require.Empty(t, gone.ID)

	// Destroy tolerates a category that is already gone.
	l.del(c.ID, u.Properties)
	c2 := l.other("FlexInsightsCategory", "c2").create(pv(map[string]property.Value{"name": sv("x")}))
	l.other("FlexInsightsCategory", "c2").del(c2.ID, c2.Properties)
	require.Nil(t, l.f.get("flex", testAccount, "Insights/Categories/"+c2.ID))

	require.Len(t, l.check(pv(map[string]property.Value{"name": sv("")})).Failures, 1)
}

func TestFlexInsightsQuestionLC_Lifecycle(t *testing.T) {
	l := newTL(t, "FlexInsightsQuestion", "q")
	cat := insightsCategory(t, l, "Greeting")
	cat2 := insightsCategory(t, l, "Closing")
	ids := map[string]property.Value{"categorySid": sv(cat), "question": sv("Did the agent greet?"), "answerSetId": sv("yes-no"), "allowNa": bv(true)}
	in := pv(ids)
	c := l.create(in)
	require.True(t, strings.HasPrefix(c.ID, "GE"))
	form := l.f.callsMatching("POST", flexInsightsBase+"Questions")[0].Form
	require.Equal(t, cat, form.Get("CategorySid"))
	require.Equal(t, "true", form.Get("AllowNa"))
	require.Empty(t, form["Description"])
	require.Equal(t, float64(0), c.Properties.Get("usage").AsNumber())
	requireNoChanges(t, l.diff(c.ID, c.Properties, in, in))

	// A false AllowNa is still sent: Twilio requires it on update.
	l.f.resetCalls()
	next := pv(with2(ids, map[string]property.Value{"allowNa": bv(false), "description": sv("Greeting check"), "categorySid": sv(cat2)}))
	d := l.diff(c.ID, c.Properties, next, in).DetailedDiff
	require.Equal(t, pgp.Update, d["allowNa"].Kind)
	require.Equal(t, pgp.Update, d["categorySid"].Kind)
	u := l.update(c.ID, c.Properties, next, in)
	uf := l.f.callsMatching("POST", flexInsightsBase+"Questions/"+c.ID)[0].Form
	require.Equal(t, "false", uf.Get("AllowNa"))
	require.Equal(t, cat2, uf.Get("CategorySid"))
	stored := l.f.get("flex", testAccount, "Insights/Questions/"+c.ID)
	require.Equal(t, "Greeting check", stored["description"])
	require.Equal(t, false, stored["allow_na"])
	requireNoChanges(t, l.diff(c.ID, u.Properties, next, next))
	// Removing the description is not a change.
	requireNoChanges(t, l.diff(c.ID, u.Properties, without(next, "description"), next))

	// Drift is read back; the question is found with or without a known category.
	l.f.mutate("flex", testAccount, "Insights/Questions/"+c.ID, func(o map[string]any) { o["question"] = "Changed?" })
	rd := l.read(c.ID, u.Properties, next)
	require.Equal(t, "Changed?", getS(t, rd.Inputs, "question"))
	require.Equal(t, pgp.Update, l.diff(c.ID, rd.Properties, next, rd.Inputs).DetailedDiff["question"].Kind)
	l.f.mutate("flex", testAccount, "Insights/Questions/"+c.ID, func(o map[string]any) {
		o["category"] = map[string]any{"category_sid": cat, "name": "Greeting"}
	})
	moved := l.read(c.ID, u.Properties, next)
	require.Equal(t, cat, getS(t, moved.Inputs, "categorySid"), "a question moved to another category is still found")

	// Import by SID alone adopts the required fields and what differs from the default.
	imp := l.read(c.ID, empty(), empty())
	require.Equal(t, cat, getS(t, imp.Inputs, "categorySid"))
	require.Equal(t, "yes-no", getS(t, imp.Inputs, "answerSetId"))
	require.False(t, imp.Inputs.Get("allowNa").AsBool())
	require.Equal(t, "Greeting check", getS(t, imp.Inputs, "description"))
	requireNoChanges(t, l.diff(imp.ID, imp.Properties, imp.Inputs, imp.Inputs))
	l.requireImportFails("GE" + strings.Repeat("8", 32))

	// Writing an empty description (the default) plans no change on a question that has none.
	bare := l.other("FlexInsightsQuestion", "bare")
	bin := pv(with2(ids, map[string]property.Value{"categorySid": sv(cat)}))
	bc := bare.create(bin)
	bimp := bare.read(bc.ID, empty(), empty())
	require.False(t, hasProp(bimp.Inputs, "description"))
	requireNoChanges(t, bare.diff(bc.ID, bimp.Properties, bimp.Inputs, bimp.Inputs))

	// Deleted outside, then destroy.
	l.f.remove("flex", testAccount, "Insights/Questions/"+c.ID)
	require.Empty(t, l.read(c.ID, u.Properties, next).ID)
	l.del(c.ID, u.Properties)
	del := l.other("FlexInsightsQuestion", "del")
	dc := del.create(bin)
	del.del(dc.ID, dc.Properties)
	require.Nil(t, l.f.get("flex", testAccount, "Insights/Questions/"+dc.ID))

	require.Len(t, l.check(pv(with2(ids, map[string]property.Value{"categorySid": sv("nope")}))).Failures, 1)
	require.Len(t, l.check(pv(with2(ids, map[string]property.Value{"question": sv(""), "answerSetId": sv("")}))).Failures, 2)
}

func TestFlexInsightsQuestionnaireLC_Lifecycle(t *testing.T) {
	l := newTL(t, "FlexInsightsQuestionnaire", "qn")
	cat := insightsCategory(t, l, "Greeting")
	q1, q2 := insightsQuestion(t, l, cat, "one"), insightsQuestion(t, l, cat, "two")
	ids := map[string]property.Value{"name": sv("Weekly QA")}
	in := pv(with2(ids, map[string]property.Value{"description": sv("Weekly"), "active": bv(true), "questionSids": strs(q1)}))
	c := l.create(in)
	form := l.f.callsMatching("POST", flexInsightsBase+"Questionnaires")[0].Form
	require.Equal(t, "Weekly QA", form.Get("Name"))
	require.Equal(t, "true", form.Get("Active"))
	require.Equal(t, []string{q1}, form["QuestionSids"])
	requireNoChanges(t, l.diff(c.ID, c.Properties, in, in))

	// Update replaces the questions (order is not significant) and sends Active.
	l.f.resetCalls()
	next := pv(with2(ids, map[string]property.Value{"description": sv("Weekly"), "active": bv(false), "questionSids": strs(q2, q1)}))
	d := l.diff(c.ID, c.Properties, next, in).DetailedDiff
	require.Equal(t, pgp.Update, d["active"].Kind)
	require.Equal(t, pgp.Update, d["questionSids"].Kind)
	u := l.update(c.ID, c.Properties, next, in)
	uf := l.f.callsMatching("POST", flexInsightsBase+"Questionnaires/"+c.ID)[0].Form
	require.Equal(t, "false", uf.Get("Active"))
	require.ElementsMatch(t, []string{q1, q2}, uf["QuestionSids"])
	requireNoChanges(t, l.diff(c.ID, u.Properties, next, next))
	reordered := pv(with2(ids, map[string]property.Value{"description": sv("Weekly"), "active": bv(false), "questionSids": strs(q1, q2)}))
	requireNoChanges(t, l.diff(c.ID, u.Properties, reordered, next))

	// A program that does not state Active still sends the current value: Twilio requires it.
	l.f.resetCalls()
	quiet := pv(with2(ids, map[string]property.Value{"description": sv("Monthly"), "questionSids": strs(q1, q2)}))
	l.update(c.ID, u.Properties, quiet, next)
	qf := l.f.callsMatching("POST", flexInsightsBase+"Questionnaires/"+c.ID)[0].Form
	require.Equal(t, "false", qf.Get("Active"), "the current Active is sent back unchanged")
	require.Equal(t, false, l.f.get("flex", testAccount, "Insights/Questionnaires/"+c.ID)["active"])
	requireNoChanges(t, l.diff(c.ID, u.Properties, without(next, "active"), next))

	// Drift is read back.
	l.f.mutate("flex", testAccount, "Insights/Questionnaires/"+c.ID, func(o map[string]any) { o["name"] = "Renamed"; o["active"] = true })
	rd := l.read(c.ID, u.Properties, next)
	require.Equal(t, "Renamed", getS(t, rd.Inputs, "name"))
	require.True(t, rd.Inputs.Get("active").AsBool())
	require.Equal(t, pgp.Update, l.diff(c.ID, rd.Properties, next, rd.Inputs).DetailedDiff["active"].Kind)

	// Import by SID alone: name, description and questions; Active is not adopted.
	imp := l.read(c.ID, empty(), empty())
	require.Equal(t, c.ID, imp.ID)
	require.Equal(t, "Renamed", getS(t, imp.Inputs, "name"))
	require.False(t, hasProp(imp.Inputs, "active"))
	require.Len(t, imp.Inputs.Get("questionSids").AsArray().AsSlice(), 2)
	requireNoChanges(t, l.diff(imp.ID, imp.Properties, imp.Inputs, imp.Inputs))
	l.requireImportFails("QQ" + strings.Repeat("7", 32))

	// A bare questionnaire imports with no description or questions and plans no change.
	bare := l.other("FlexInsightsQuestionnaire", "bare")
	bc := bare.create(pv(ids))
	bimp := bare.read(bc.ID, empty(), empty())
	require.False(t, hasProp(bimp.Inputs, "description"))
	require.False(t, hasProp(bimp.Inputs, "questionSids"))
	requireNoChanges(t, bare.diff(bc.ID, bimp.Properties, bimp.Inputs, bimp.Inputs))

	// Deleted outside, then destroy.
	l.f.remove("flex", testAccount, "Insights/Questionnaires/"+c.ID)
	require.Empty(t, l.read(c.ID, u.Properties, next).ID)
	l.del(c.ID, u.Properties)
	bare.del(bc.ID, bc.Properties)
	require.Nil(t, l.f.get("flex", testAccount, "Insights/Questionnaires/"+bc.ID))

	require.Len(t, l.check(pv(map[string]property.Value{"name": sv("")})).Failures, 1)
	require.Len(t, l.check(pv(with2(ids, map[string]property.Value{"questionSids": strs("nope")}))).Failures, 1)
}

func TestLookup_FlexInsights(t *testing.T) {
	l := newTL(t, "FlexInsightsQuestionnaire", "qn")
	cat := insightsCategory(t, l, "Greeting")
	insightsCategory(t, l, "Closing")
	insightsCategory(t, l, "Billing")
	q1 := insightsQuestion(t, l, cat, "one")
	qn := l.create(pv(map[string]property.Value{"name": sv("Weekly"), "active": bv(false), "questionSids": strs(q1)}))
	l.other("FlexInsightsQuestionnaire", "on").create(pv(map[string]property.Value{"name": sv("On"), "active": bv(true)}))

	c, err := invoke(t, l, "getFlexInsightsCategory", map[string]property.Value{"sid": sv(cat)})
	require.NoError(t, err)
	require.Equal(t, "Greeting", getS(t, c, "name"))
	_, err = invoke(t, l, "getFlexInsightsCategory", map[string]property.Value{"sid": sv("FYmissing")})
	require.ErrorContains(t, err, "was not found")
	_, err = invoke(t, l, "getFlexInsightsCategory", map[string]property.Value{"sid": sv("")})
	require.ErrorContains(t, err, "must be set")
	cs, err := invoke(t, l, "getFlexInsightsCategories", nil)
	require.NoError(t, err)
	require.Len(t, cs.Get("categories").AsArray().AsSlice(), 3, "the list follows its pages")

	q, err := invoke(t, l, "getFlexInsightsQuestion", map[string]property.Value{"sid": sv(q1)})
	require.NoError(t, err)
	require.Equal(t, cat, getS(t, q, "categorySid"))
	require.Equal(t, "yes-no", getS(t, q, "answerSetId"))
	require.False(t, q.Get("allowNa").AsBool())
	q, err = invoke(t, l, "getFlexInsightsQuestion", map[string]property.Value{"sid": sv(q1), "categorySid": sv(cat)})
	require.NoError(t, err)
	require.Equal(t, "one", getS(t, q, "question"))
	_, err = invoke(t, l, "getFlexInsightsQuestion", map[string]property.Value{"sid": sv("GEmissing")})
	require.ErrorContains(t, err, "was not found")
	qs, err := invoke(t, l, "getFlexInsightsQuestions", map[string]property.Value{"categorySid": sv(cat)})
	require.NoError(t, err)
	require.Len(t, qs.Get("questions").AsArray().AsSlice(), 1)
	qs, err = invoke(t, l, "getFlexInsightsQuestions", map[string]property.Value{"categorySid": sv("FY" + strings.Repeat("5", 32))})
	require.NoError(t, err)
	require.Empty(t, qs.Get("questions").AsArray().AsSlice())

	n, err := invoke(t, l, "getFlexInsightsQuestionnaire", map[string]property.Value{"sid": sv(qn.ID)})
	require.NoError(t, err)
	require.Equal(t, "Weekly", getS(t, n, "name"))
	require.False(t, n.Get("active").AsBool())
	require.Len(t, n.Get("questionSids").AsArray().AsSlice(), 1)
	_, err = invoke(t, l, "getFlexInsightsQuestionnaire", map[string]property.Value{"sid": sv("QQmissing")})
	require.ErrorContains(t, err, "was not found")
	ns, err := invoke(t, l, "getFlexInsightsQuestionnaires", nil)
	require.NoError(t, err)
	require.Len(t, ns.Get("questionnaires").AsArray().AsSlice(), 1, "inactive questionnaires are left out by default")
	ns, err = invoke(t, l, "getFlexInsightsQuestionnaires", map[string]property.Value{"includeInactive": bv(true)})
	require.NoError(t, err)
	require.Len(t, ns.Get("questionnaires").AsArray().AsSlice(), 2)

	as, err := invoke(t, l, "getFlexInsightsAnswerSets", nil)
	require.NoError(t, err)
	require.Contains(t, getS(t, as, "answerSets"), "yes-no")
	require.Contains(t, getS(t, as, "notApplicable"), "N/A")
	require.Contains(t, getS(t, as, "answerSetCategories"), "binary")
	ct, err := invoke(t, l, "getFlexInsightsCommentTags", nil)
	require.NoError(t, err)
	require.Contains(t, getS(t, ct, "comments"), "good")
}
