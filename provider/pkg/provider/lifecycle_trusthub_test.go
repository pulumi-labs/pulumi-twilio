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

// The two resources share their behaviour, so one table drives both.
var thCases = []struct{ typ, coll, prefix, apiType string }{
	{"TrustHubEndUser", "EndUsers", "IT", "customer_profile_business_information"},
	{"TrustHubSupportingDocument", "SupportingDocuments", "RD", "customer_profile_address"},
}

func TestTrustHubLC_Lifecycle(t *testing.T) {
	for _, tc := range thCases {
		t.Run(tc.typ, func(t *testing.T) {
			l := newTL(t, tc.typ, "x")
			base := "/trusthub/v1/" + tc.coll
			in := pv(map[string]property.Value{"friendlyName": sv("acme"), "type": sv(tc.apiType),
				"attributes": sv(`{"business_name":"Acme","business_type":"LLC"}`)})
			c := l.create(in)
			require.True(t, strings.HasPrefix(c.ID, tc.prefix), c.ID)
			require.Equal(t, c.ID, getS(t, c.Properties, "sid"))
			require.Equal(t, testAccount, getS(t, c.Properties, "accountSid"))
			form := l.f.callsMatching("POST", base)[0].Form
			require.Equal(t, "acme", form.Get("FriendlyName"))
			require.Equal(t, tc.apiType, form.Get("Type"))
			require.JSONEq(t, `{"business_name":"Acme","business_type":"LLC"}`, form.Get("Attributes"))
			if tc.coll == "SupportingDocuments" {
				require.Equal(t, "DRAFT", getS(t, c.Properties, "status"))
			}
			requireNoChanges(t, l.diff(c.ID, c.Properties, in, in))

			// A refresh keeps the program's formatting when Twilio holds the same document, even with another key order.
			spaced := with(in, map[string]property.Value{"attributes": sv(`{ "business_type": "LLC", "business_name": "Acme" }`)})
			rd := l.read(c.ID, c.Properties, spaced)
			require.Equal(t, getS(t, spaced, "attributes"), getS(t, rd.Inputs, "attributes"))
			requireNoChanges(t, l.diff(c.ID, rd.Properties, spaced, rd.Inputs))

			// Drift shows for what the program states.
			l.f.mutate("th:"+tc.coll, testAccount, c.ID, func(o map[string]any) {
				o["friendly_name"] = "renamed"
				o["attributes"] = map[string]any{"business_name": "Other"}
			})
			rd = l.read(c.ID, c.Properties, in)
			require.Equal(t, "renamed", getS(t, rd.Inputs, "friendlyName"))
			d := l.diff(c.ID, rd.Properties, in, rd.Inputs)
			require.Equal(t, pgp.Update, d.DetailedDiff["friendlyName"].Kind)
			require.Equal(t, pgp.Update, d.DetailedDiff["attributes"].Kind)

			// friendlyName and attributes update in place; the type replaces.
			next := with(in, map[string]property.Value{"friendlyName": sv("acme two"), "attributes": sv(`{"business_name":"Acme Two"}`)})
			l.f.resetCalls()
			u := l.update(c.ID, rd.Properties, next, rd.Inputs)
			uf := l.f.callsMatching("POST", base+"/"+c.ID)[0].Form
			require.Equal(t, "acme two", uf.Get("FriendlyName"))
			require.JSONEq(t, `{"business_name":"Acme Two"}`, uf.Get("Attributes"))
			require.Empty(t, uf["Type"])
			rd = l.read(c.ID, u.Properties, next)
			requireNoChanges(t, l.diff(c.ID, rd.Properties, next, rd.Inputs))
			rep := l.diff(c.ID, rd.Properties, with(next, map[string]property.Value{"type": sv("other_type")}), rd.Inputs)
			require.Equal(t, pgp.UpdateReplace, rep.DetailedDiff["type"].Kind)

			// Removing the attributes input is no change.
			requireNoChanges(t, l.diff(c.ID, rd.Properties, without(next, "attributes"), rd.Inputs))

			// Import adopts everything Twilio holds and plans no change for the same program.
			imp := l.read(c.ID, empty(), empty())
			require.Empty(t, l.check(imp.Inputs).Failures)
			require.Equal(t, tc.apiType, getS(t, imp.Inputs, "type"))
			require.Equal(t, "acme two", getS(t, imp.Inputs, "friendlyName"))
			require.JSONEq(t, `{"business_name":"Acme Two"}`, getS(t, imp.Inputs, "attributes"))
			requireNoChanges(t, l.diff(c.ID, imp.Properties, imp.Inputs, imp.Inputs))
			l.requireImportFails(tc.prefix + strings.Repeat("0", 32))

			// Delete; a second delete and a refresh of a deleted object are fine.
			l.f.resetCalls()
			l.del(c.ID, u.Properties)
			require.Len(t, l.f.callsMatching("DELETE", base+"/"+c.ID), 1)
			require.Empty(t, l.read(c.ID, u.Properties, next).ID)
			l.del(c.ID, u.Properties)
		})
	}
}

func TestTrustHubLC_DefaultsAndCheck(t *testing.T) {
	for _, tc := range thCases {
		t.Run(tc.typ, func(t *testing.T) {
			l := newTL(t, tc.typ, "x")
			// No attributes: Twilio reports {}; a program that does not state them plans nothing, and one that states {} plans nothing either.
			bare := pv(map[string]property.Value{"friendlyName": sv("n"), "type": sv(tc.apiType)})
			c := l.create(bare)
			require.Empty(t, l.f.callsMatching("POST", "/trusthub/v1/"+tc.coll)[0].Form["Attributes"])
			rd := l.read(c.ID, c.Properties, bare)
			require.False(t, hasProp(rd.Inputs, "attributes"))
			requireNoChanges(t, l.diff(c.ID, rd.Properties, bare, rd.Inputs))
			requireNoChanges(t, l.diff(c.ID, rd.Properties, with(bare, map[string]property.Value{"attributes": sv("{}")}), rd.Inputs))
			imp := l.read(c.ID, empty(), empty())
			require.False(t, hasProp(imp.Inputs, "attributes"), "an import adopts no {} attributes")

			require.Len(t, l.check(with(bare, map[string]property.Value{"friendlyName": sv("")})).Failures, 1)
			require.Len(t, l.check(with(bare, map[string]property.Value{"type": sv("")})).Failures, 1)
			require.Len(t, l.check(with(bare, map[string]property.Value{"attributes": sv("[1]")})).Failures, 1)
			require.Len(t, l.check(with(bare, map[string]property.Value{"attributes": sv("not json")})).Failures, 1)
			require.Empty(t, l.check(with(bare, map[string]property.Value{"attributes": computed()})).Failures)
		})
	}
}

func TestTrustHubLC_CreateError(t *testing.T) {
	l := newTL(t, "TrustHubEndUser", "x")
	l.f.failNext("POST", "/trusthub/v1/EndUsers", 400)
	_, err := l.s.Create(pgp.CreateRequest{Urn: l.urn, Properties: l.checked(pv(map[string]property.Value{"friendlyName": sv("n"), "type": sv("t")}))})
	require.ErrorContains(t, err, "create TrustHub end user")
}

func TestLookup_TrustHub(t *testing.T) {
	l := newTL(t, "TrustHubEndUser", "x")
	eu := l.create(pv(map[string]property.Value{"friendlyName": sv("rep"), "type": sv("authorized_representative_1"), "attributes": sv(`{"first_name":"Ada"}`)}))
	doc := l.other("TrustHubSupportingDocument", "d").create(pv(map[string]property.Value{"friendlyName": sv("addr"), "type": sv("customer_profile_address")}))
	r, err := invoke(t, l, "getTrustHubEndUser", map[string]property.Value{"sid": sv(eu.ID)})
	require.NoError(t, err)
	require.Equal(t, "authorized_representative_1", getS(t, r, "type"))
	require.JSONEq(t, `{"first_name":"Ada"}`, getS(t, r, "attributes"))
	r, err = invoke(t, l, "getTrustHubSupportingDocument", map[string]property.Value{"sid": sv(doc.ID)})
	require.NoError(t, err)
	require.Equal(t, "DRAFT", getS(t, r, "status"))
	require.Equal(t, "customer_profile_address", getS(t, r, "type"))
	_, err = invoke(t, l, "getTrustHubEndUser", map[string]property.Value{"sid": sv("IT" + strings.Repeat("0", 32))})
	require.ErrorContains(t, err, "was not found")
	_, err = invoke(t, l, "getTrustHubSupportingDocument", map[string]property.Value{"sid": sv("")})
	require.ErrorContains(t, err, "must be set")

	// Lists follow the pages (two per page in the fake).
	for i := 0; i < 2; i++ {
		l.create(pv(map[string]property.Value{"friendlyName": sv("more"), "type": sv("t")}))
	}
	r, err = invoke(t, l, "getTrustHubEndUsers", map[string]property.Value{})
	require.NoError(t, err)
	require.Len(t, r.Get("endUsers").AsArray().AsSlice(), 3)
	r, err = invoke(t, l, "getTrustHubSupportingDocuments", map[string]property.Value{})
	require.NoError(t, err)
	require.Len(t, r.Get("supportingDocuments").AsArray().AsSlice(), 1)

	// Types and policies are read only.
	typ := func(i string) map[string]any {
		return map[string]any{"sid": "OY" + strings.Repeat(i, 32), "friendly_name": "Business " + i, "machine_name": "business_" + i,
			"fields": []any{map[string]any{"machine_name": "business_name", "type": "string", "required": true}}}
	}
	for _, i := range []string{"1", "2", "3"} {
		l.f.addTrusthubItem("EndUserTypes", typ(i))
	}
	l.f.addTrusthubItem("SupportingDocumentTypes", typ("4"))
	l.f.addTrusthubItem("Policies", map[string]any{"sid": "RN" + strings.Repeat("5", 32), "friendly_name": "Primary", "requirements": map[string]any{"end_user": []any{}}})
	r, err = invoke(t, l, "getTrustHubEndUserTypes", map[string]property.Value{})
	require.NoError(t, err)
	require.Len(t, r.Get("types").AsArray().AsSlice(), 3)
	r, err = invoke(t, l, "getTrustHubEndUserType", map[string]property.Value{"sid": sv("OY" + strings.Repeat("2", 32))})
	require.NoError(t, err)
	require.Equal(t, "business_2", getS(t, r, "machineName"))
	require.Contains(t, getS(t, r, "fields"), "business_name")
	r, err = invoke(t, l, "getTrustHubSupportingDocumentTypes", map[string]property.Value{})
	require.NoError(t, err)
	require.Len(t, r.Get("types").AsArray().AsSlice(), 1)
	r, err = invoke(t, l, "getTrustHubSupportingDocumentType", map[string]property.Value{"sid": sv("OY" + strings.Repeat("4", 32))})
	require.NoError(t, err)
	require.Equal(t, "business_4", getS(t, r, "machineName"))
	r, err = invoke(t, l, "getTrustHubPolicies", map[string]property.Value{})
	require.NoError(t, err)
	require.Len(t, r.Get("policies").AsArray().AsSlice(), 1)
	r, err = invoke(t, l, "getTrustHubPolicy", map[string]property.Value{"sid": sv("RN" + strings.Repeat("5", 32))})
	require.NoError(t, err)
	require.Equal(t, "Primary", getS(t, r, "friendlyName"))
	require.Contains(t, getS(t, r, "requirements"), "end_user")
	_, err = invoke(t, l, "getTrustHubPolicy", map[string]property.Value{"sid": sv("RN" + strings.Repeat("0", 32))})
	require.ErrorContains(t, err, "was not found")
}
