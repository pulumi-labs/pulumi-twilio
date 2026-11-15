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
	"time"

	pgp "github.com/pulumi/pulumi-go-provider"
	"github.com/pulumi/pulumi/sdk/v3/go/property"
	"github.com/stretchr/testify/require"
)

func storeInputs() property.Map {
	return pv(map[string]property.Value{"displayName": sv("support-memory"), "description": sv("Customer memory")})
}

func traitValue(kv map[string]property.Value) property.Value { return property.New(pv(kv)) }

func contactTraits() property.Value {
	return property.New(pv(map[string]property.Value{
		"email": traitValue(map[string]property.Value{"dataType": sv("STRING"), "description": sv("Primary email"), "idTypePromotion": sv("email")}),
		"tier":  traitValue(map[string]property.Value{"dataType": sv("STRING"), "validationRule": sv(`{"ruleType":"STRING","maxLength":20}`)}),
	}))
}

func groupInputs() property.Map {
	return pv(map[string]property.Value{"storeId": sv(memStoreID), "displayName": sv("Contact"), "description": sv("Contact details"), "traits": contactTraits()})
}

const memStoreID = "mem_store_00000000000000000000000001"

func seedMemoryStore(l *tl) {
	l.f.put("mstore", testAccount, memStoreID, map[string]any{"id": memStoreID, "displayName": "seed", "status": "ACTIVE", "version": 1.0})
}

func TestMemoryStoreLC_Lifecycle(t *testing.T) {
	fastKnowledgePolls(t)
	l := newTL(t, "MemoryStore", "s")
	in := storeInputs()
	c := l.create(in)
	id := c.ID
	require.True(t, strings.HasPrefix(id, "mem_store_"), id)
	require.Equal(t, "ACTIVE", getS(t, c.Properties, "status"))
	require.True(t, strings.HasPrefix(getS(t, c.Properties, "intelligenceServiceId"), "intelligence_service_"))
	require.Len(t, l.f.callsMatching("GET", "/memory/v1/ControlPlane/Operations/"), 2)
	require.JSONEq(t, `{"displayName":"support-memory","description":"Customer memory"}`, l.f.callsMatching("POST", "/memory/v1/ControlPlane/Stores")[0].Body)
	requireNoChanges(t, l.diff(id, c.Properties, in, in))

	rd := l.read(id, c.Properties, in)
	requireNoChanges(t, l.diff(id, rd.Properties, in, rd.Inputs))
	l.f.mutate("mstore", testAccount, id, func(o map[string]any) { o["description"] = "console edit" })
	require.Equal(t, "console edit", getS(t, l.read(id, c.Properties, in).Inputs, "description"))
	l.f.mutate("mstore", testAccount, id, func(o map[string]any) { o["description"] = "Customer memory" })

	next := with(in, map[string]property.Value{"displayName": sv("support-memory-2"), "description": sv("v2")})
	d := l.diff(id, c.Properties, next, in)
	require.Equal(t, pgp.Update, d.DetailedDiff["displayName"].Kind)
	require.Equal(t, pgp.Update, d.DetailedDiff["description"].Kind)
	l.f.resetCalls()
	u := l.update(id, c.Properties, next, in)
	require.JSONEq(t, `{"displayName":"support-memory-2","description":"v2"}`, l.f.callsMatching("PATCH", "/memory/v1/ControlPlane/Stores/"+id)[0].Body)
	require.Equal(t, 2.0, u.Properties.Get("revision").AsNumber())
	require.Equal(t, "support-memory-2", l.f.get("mstore", testAccount, id)["displayName"])
	require.Equal(t, pgp.UpdateReplace, l.diff(id, u.Properties, without(next, "description"), next).DetailedDiff["description"].Kind)

	l.del(id, u.Properties)
	require.Nil(t, l.f.get("mstore", testAccount, id))
	l.del(id, u.Properties)
	require.Empty(t, l.read(id, u.Properties, next).ID)
}

func TestMemoryStoreLC_ImportChecksAndFailure(t *testing.T) {
	fastKnowledgePolls(t)
	l := newTL(t, "MemoryStore", "s")
	in := storeInputs()
	c := l.create(in)
	imp := l.read(c.ID, empty(), empty())
	require.Equal(t, "support-memory", getS(t, imp.Inputs, "displayName"))
	requireNoChanges(t, l.diff(c.ID, imp.Properties, in, imp.Inputs))
	l.requireImportFails("mem_store_missing")
	for name, bad := range map[string]property.Map{
		"displayName": with(in, map[string]property.Value{"displayName": sv("no spaces")}),
		"description": with(in, map[string]property.Value{"description": sv("")}),
	} {
		f := l.check(bad).Failures
		require.NotEmpty(t, f, name)
		require.Equal(t, name, f[0].Property)
	}
	memoryFailOps = "store limit reached"
	t.Cleanup(func() { memoryFailOps = "" })
	_, err := l.other("MemoryStore", "s2").s.Create(pgp.CreateRequest{Urn: l.other("MemoryStore", "s2").urn, Properties: l.checked(with(in, map[string]property.Value{"displayName": sv("second")}))})
	require.ErrorContains(t, err, "store limit reached")
}

func TestMemoryTraitGroupLC_Lifecycle(t *testing.T) {
	fastKnowledgePolls(t)
	l := newTL(t, "MemoryTraitGroup", "g")
	seedMemoryStore(l)
	in := groupInputs()
	c := l.create(in)
	require.Equal(t, memStoreID+"/Contact", c.ID)
	require.Equal(t, 1.0, c.Properties.Get("revision").AsNumber())
	require.JSONEq(t, `{"displayName":"Contact","description":"Contact details","traits":{"email":{"dataType":"STRING","description":"Primary email","idTypePromotion":"email"},"tier":{"dataType":"STRING","validationRule":{"ruleType":"STRING","maxLength":20}}}}`,
		l.f.callsMatching("POST", "/memory/v1/ControlPlane/Stores/"+memStoreID+"/TraitGroups")[0].Body)
	requireNoChanges(t, l.diff(c.ID, c.Properties, in, in))

	// A refresh plans nothing, even though Twilio returns the rule re-encoded; the rule's spelling is the program's.
	respelled := with(in, map[string]property.Value{"traits": property.New(pv(map[string]property.Value{
		"email": traitValue(map[string]property.Value{"dataType": sv("STRING"), "description": sv("Primary email"), "idTypePromotion": sv("email")}),
		"tier":  traitValue(map[string]property.Value{"dataType": sv("STRING"), "validationRule": sv("{ \"maxLength\": 20, \"ruleType\": \"STRING\" }")}),
	}))})
	rd := l.read(c.ID, c.Properties, respelled)
	requireNoChanges(t, l.diff(c.ID, rd.Properties, respelled, rd.Inputs))

	// Out-of-band trait changes are drift.
	l.f.mutate("mgroup", testAccount, memStoreID+"~Contact", func(o map[string]any) {
		o["traits"].(map[string]any)["extra"] = map[string]any{"dataType": "NUMBER"}
	})
	require.Contains(t, jsonString(l.read(c.ID, c.Properties, in).Inputs.Get("traits").AsMap().Get("extra").AsMap().Get("dataType").AsString()), "NUMBER")
	l.f.mutate("mgroup", testAccount, memStoreID+"~Contact", func(o map[string]any) { delete(o["traits"].(map[string]any), "extra") })

	// Add one trait, change one, drop one: a single PATCH with the changed definitions and a deletion marker.
	next := with(in, map[string]property.Value{"description": sv("Contacts"), "traits": property.New(pv(map[string]property.Value{
		"email": traitValue(map[string]property.Value{"dataType": sv("STRING"), "description": sv("Work email"), "idTypePromotion": sv("email")}),
		"age":   traitValue(map[string]property.Value{"dataType": sv("NUMBER")}),
	}))})
	require.Equal(t, pgp.Update, l.diff(c.ID, c.Properties, next, in).DetailedDiff["traits"].Kind)
	l.f.resetCalls()
	u := l.update(c.ID, c.Properties, next, in)
	require.JSONEq(t, `{"description":"Contacts","traits":{"email":{"dataType":"STRING","description":"Work email","idTypePromotion":"email"},"age":{"dataType":"NUMBER"},"tier":{"dataType":""}}}`,
		l.f.callsMatching("PATCH", "/memory/v1/ControlPlane/Stores/"+memStoreID+"/TraitGroups/Contact")[0].Body)
	require.Equal(t, 2.0, u.Properties.Get("revision").AsNumber())
	requireNoChanges(t, l.diff(c.ID, u.Properties, next, next))
	require.Len(t, l.f.get("mgroup", testAccount, memStoreID+"~Contact")["traits"], 2)

	// The store and the name replace the group.
	for k, v := range map[string]property.Value{"storeId": sv("mem_store_other"), "displayName": sv("Other")} {
		require.Equal(t, pgp.UpdateReplace, l.diff(c.ID, u.Properties, with(next, map[string]property.Value{k: v}), next).DetailedDiff[k].Kind, k)
	}

	// Delete waits for the group to disappear; a locked group surfaces Twilio's refusal.
	l.f.mutate("mgroup", testAccount, memStoreID+"~Contact", func(o map[string]any) { o["locked"] = true })
	require.Error(t, l.s.Delete(pgp.DeleteRequest{ID: c.ID, Urn: l.urn, Properties: u.Properties}))
	l.f.mutate("mgroup", testAccount, memStoreID+"~Contact", func(o map[string]any) { delete(o, "locked") })
	l.del(c.ID, u.Properties)
	require.Nil(t, l.f.get("mgroup", testAccount, memStoreID+"~Contact"))
	l.del(c.ID, u.Properties)
	require.Empty(t, l.read(c.ID, u.Properties, next).ID)
}

func TestMemoryTraitGroupLC_ImportAndChecks(t *testing.T) {
	fastKnowledgePolls(t)
	l := newTL(t, "MemoryTraitGroup", "g")
	seedMemoryStore(l)
	in := groupInputs()
	c := l.create(in)
	imp := l.read(c.ID, empty(), empty())
	require.Equal(t, memStoreID, getS(t, imp.Inputs, "storeId"))
	require.Equal(t, "Contact", getS(t, imp.Inputs, "displayName"))
	require.Len(t, imp.Inputs.Get("traits").AsMap().AsMap(), 2)
	requireNoChanges(t, l.diff(c.ID, imp.Properties, in, imp.Inputs))
	l.requireImportFails(memStoreID + "/Missing")
	_, err := l.s.Read(pgp.ReadRequest{ID: "no-slash", Urn: l.urn, Properties: empty(), Inputs: empty()})
	require.ErrorContains(t, err, "expected <storeId>/<displayName>")

	// No traits is allowed: the group starts empty.
	empty := l.other("MemoryTraitGroup", "e").create(pv(map[string]property.Value{"storeId": sv(memStoreID), "displayName": sv("Empty")}))
	require.Equal(t, memStoreID+"/Empty", empty.ID)

	trait := func(kv map[string]property.Value) property.Value {
		return property.New(pv(map[string]property.Value{"x": traitValue(kv)}))
	}
	for name, c := range map[string]struct {
		in   property.Map
		prop string
	}{
		"bad name":         {with(in, map[string]property.Value{"displayName": sv("has space")}), "displayName"},
		"long description": {with(in, map[string]property.Value{"description": sv(strings.Repeat("d", 1025))}), "description"},
		"bad type":         {with(in, map[string]property.Value{"traits": trait(map[string]property.Value{"dataType": sv("DATE")})}), "traits[x].dataType"},
		"deletion marker":  {with(in, map[string]property.Value{"traits": trait(map[string]property.Value{"dataType": sv("")})}), "traits[x].dataType"},
		"bad rule":         {with(in, map[string]property.Value{"traits": trait(map[string]property.Value{"dataType": sv("STRING"), "validationRule": sv("[1]")})}), "traits[x].validationRule"},
		"long promotion":   {with(in, map[string]property.Value{"traits": trait(map[string]property.Value{"dataType": sv("STRING"), "idTypePromotion": sv(strings.Repeat("p", 31))})}), "traits[x].idTypePromotion"},
	} {
		f := l.check(c.in).Failures
		require.NotEmpty(t, f, name)
		require.Equal(t, c.prop, f[0].Property, name)
	}
}

func TestLookup_MemoryStoreAndTraitGroup(t *testing.T) {
	fastKnowledgePolls(t)
	l := newTL(t, "MemoryStore", "s")
	s := l.create(storeInputs())
	r, err := invoke(t, l, "getMemoryStore", map[string]property.Value{"storeId": sv(s.ID)})
	require.NoError(t, err)
	require.Equal(t, "support-memory", getS(t, r, "displayName"))
	require.Equal(t, "ACTIVE", getS(t, r, "status"))
	_, err = invoke(t, l, "getMemoryStore", map[string]property.Value{"storeId": sv("mem_store_missing")})
	require.Error(t, err)
	_, err = invoke(t, l, "getMemoryStore", map[string]property.Value{"storeId": sv("")})
	require.Error(t, err)

	g := l.other("MemoryTraitGroup", "g").create(with(groupInputs(), map[string]property.Value{"storeId": sv(s.ID)}))
	require.NotEmpty(t, g.ID)
	r, err = invoke(t, l, "getMemoryTraitGroup", map[string]property.Value{"storeId": sv(s.ID), "displayName": sv("Contact")})
	require.NoError(t, err)
	require.Equal(t, "Contact details", getS(t, r, "description"))
	require.Len(t, r.Get("traits").AsMap().AsMap(), 2)
	_, err = invoke(t, l, "getMemoryTraitGroup", map[string]property.Value{"storeId": sv(s.ID), "displayName": sv("Nope")})
	require.Error(t, err)
}

// A trait group that never becomes readable fails the create, and the provider asks Twilio to delete it so no
// group is left that no state tracks (review note on #2016).
func TestMemoryTraitGroupLC_CreateCleansUpWhenNeverVisible(t *testing.T) {
	fastKnowledgePolls(t)
	knowledgePollTimeout = 100 * time.Millisecond
	memoryHideGroups = true
	t.Cleanup(func() { memoryHideGroups = false })
	l := newTL(t, "MemoryTraitGroup", "g")
	seedMemoryStore(l)
	_, err := l.s.Create(pgp.CreateRequest{Urn: l.urn, Properties: l.checked(groupInputs())})
	require.ErrorContains(t, err, "did not finish")
	require.NotEmpty(t, l.f.callsMatching("DELETE", "/memory/v1/ControlPlane/Stores/"+memStoreID+"/TraitGroups/Contact"))
}
