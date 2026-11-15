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

func traitMap(expr, group, name string) property.Value {
	return property.New(pv(map[string]property.Value{"expression": sv(expr), "traitGroup": sv(group), "traitName": sv(name)}))
}

func memStrList(vs ...string) property.Value {
	out := make([]property.Value, 0, len(vs))
	for _, v := range vs {
		out = append(out, sv(v))
	}
	return property.New(out)
}

func mappingInputs() property.Map {
	return pv(map[string]property.Value{"storeId": sv(memStoreID), "displayName": sv("crm-import"), "description": sv("From the CRM export"),
		"fromType": sv("INGRESS"), "columns": memStrList("Email", "Tier"),
		"mappings": property.New([]property.Value{traitMap("Email", "Contact", "email"), traitMap("Tier", "Contact", "tier")})})
}

const mappingDataset = "tdi_dataset_00000000000000000000000001"

const mappingsPath = "/memory/v1/ControlPlane/Stores/" + memStoreID + "/DataMappings"

func TestMemoryDataMappingLC_Lifecycle(t *testing.T) {
	fastKnowledgePolls(t)
	l := newTL(t, "MemoryDataMapping", "m")
	seedMemoryStore(l)
	in := mappingInputs()
	require.Empty(t, l.check(in).Failures)
	c := l.create(in)
	mid := getS(t, c.Properties, "dataMappingId")
	require.True(t, strings.HasPrefix(mid, "mem_datamapping_"), mid)
	require.Equal(t, memStoreID+"/"+mid, c.ID)
	require.Equal(t, 1.0, c.Properties.Get("revision").AsNumber())
	require.JSONEq(t, `{"displayName":"crm-import","description":"From the CRM export","mappingFrom":{"type":"INGRESS","columns":["Email","Tier"]},"mappingTo":{"type":"TRAITS","mappings":[{"expression":"Email","traitGroup":"Contact","traitName":"email"},{"expression":"Tier","traitGroup":"Contact","traitName":"tier"}]}}`,
		l.f.callsMatching("POST", mappingsPath)[0].Body)
	requireNoChanges(t, l.diff(c.ID, c.Properties, in, in))

	// isEnabled: Twilio's default is true, so writing it plans nothing, and a refresh keeps the program's choice.
	explicit := with(in, map[string]property.Value{"isEnabled": property.New(true)})
	requireNoChanges(t, l.diff(c.ID, c.Properties, explicit, in))
	rd := l.read(c.ID, c.Properties, in)
	requireNoChanges(t, l.diff(c.ID, rd.Properties, in, rd.Inputs))
	rd = l.read(c.ID, c.Properties, explicit)
	requireNoChanges(t, l.diff(c.ID, rd.Properties, explicit, rd.Inputs))
	requireNoChanges(t, l.diff(c.ID, rd.Properties, in, rd.Inputs))

	// Drift on Twilio is picked up by a refresh.
	l.f.mutate("mmap", testAccount, mid, func(o map[string]any) { o["isEnabled"] = false })
	drift := l.read(c.ID, c.Properties, in)
	require.False(t, drift.Inputs.Get("isEnabled").AsBool())
	require.Equal(t, pgp.Update, l.diff(c.ID, drift.Properties, in, drift.Inputs).DetailedDiff["isEnabled"].Kind)
	l.f.mutate("mmap", testAccount, mid, func(o map[string]any) { o["isEnabled"] = true })

	// An update sends only what changed; the mapping list is sent whole.
	next := with(in, map[string]property.Value{"displayName": sv("crm-import-2"), "isEnabled": property.New(false),
		"mappings": property.New([]property.Value{traitMap("Email", "Contact", "email")}), "columns": memStrList("Email", "Tier")})
	d := l.diff(c.ID, c.Properties, next, in)
	for _, k := range []string{"displayName", "isEnabled", "mappings"} {
		require.Equal(t, pgp.Update, d.DetailedDiff[k].Kind, k)
	}
	l.f.resetCalls()
	u := l.update(c.ID, c.Properties, next, in)
	require.JSONEq(t, `{"displayName":"crm-import-2","isEnabled":false,"mappingTo":{"type":"TRAITS","mappings":[{"expression":"Email","traitGroup":"Contact","traitName":"email"}]}}`,
		l.f.callsMatching("PATCH", mappingsPath+"/"+mid)[0].Body)
	require.Equal(t, 2.0, u.Properties.Get("revision").AsNumber())
	requireNoChanges(t, l.diff(c.ID, u.Properties, next, next))

	// The source cannot be changed, and a description cannot be cleared: both replace.
	for k, v := range map[string]property.Value{"storeId": sv("mem_store_other"), "fromType": sv("DATASET_CLOUDAPP"), "columns": memStrList("Email"), "datasetId": sv(mappingDataset)} {
		require.Equal(t, pgp.UpdateReplace, l.diff(c.ID, u.Properties, with(next, map[string]property.Value{k: v}), next).DetailedDiff[k].Kind, k)
	}
	require.Equal(t, pgp.UpdateReplace, l.diff(c.ID, u.Properties, without(next, "description"), next).DetailedDiff["description"].Kind)
	require.Equal(t, pgp.Update, l.diff(c.ID, u.Properties, with(next, map[string]property.Value{"description": sv("changed")}), next).DetailedDiff["description"].Kind)

	// Import adopts everything the mapping states; a program that matches plans nothing.
	imp := l.read(c.ID, empty(), empty())
	require.Equal(t, "INGRESS", getS(t, imp.Inputs, "fromType"))
	require.False(t, imp.Inputs.Get("isEnabled").AsBool(), "a disabled mapping is adopted")
	requireNoChanges(t, l.diff(c.ID, imp.Properties, next, imp.Inputs))
	l.requireImportFails(memStoreID + "/mem_datamapping_missing")
	_, err := l.s.Read(pgp.ReadRequest{ID: "no-slash", Urn: l.urn, Properties: empty(), Inputs: empty()})
	require.ErrorContains(t, err, "expected <storeId>/<dataMappingId>")

	l.del(c.ID, u.Properties)
	require.Nil(t, l.f.get("mmap", testAccount, mid))
	l.del(c.ID, u.Properties)
	require.Empty(t, l.read(c.ID, u.Properties, next).ID)
}

func TestMemoryDataMappingLC_ImportDefaultsAndDatasets(t *testing.T) {
	fastKnowledgePolls(t)
	l := newTL(t, "MemoryDataMapping", "m")
	seedMemoryStore(l)
	ds := with(mappingInputs(), map[string]property.Value{"fromType": sv("DATASET_WAREHOUSE"), "datasetId": sv(mappingDataset)})
	ds = without(ds, "columns")
	c := l.create(ds)
	require.Contains(t, l.f.callsMatching("POST", mappingsPath)[0].Body, `"mappingFrom":{"datasetId":"`+mappingDataset+`","type":"DATASET_WAREHOUSE"}`)
	imp := l.read(c.ID, empty(), empty())
	require.Equal(t, mappingDataset, getS(t, imp.Inputs, "datasetId"))
	require.False(t, imp.Inputs.Get("isEnabled").IsString(), "an enabled mapping is left to the default")
	requireNoChanges(t, l.diff(c.ID, imp.Properties, ds, imp.Inputs))
	requireNoChanges(t, l.diff(c.ID, imp.Properties, with(ds, map[string]property.Value{"isEnabled": property.New(true)}), imp.Inputs))
}

func TestMemoryDataMappingLC_FailedOperationLeavesNothing(t *testing.T) {
	fastKnowledgePolls(t)
	memoryFailOps = "boom"
	t.Cleanup(func() { memoryFailOps = "" })
	l := newTL(t, "MemoryDataMapping", "m")
	seedMemoryStore(l)
	_, err := l.s.Create(pgp.CreateRequest{Urn: l.urn, Properties: l.checked(mappingInputs())})
	require.ErrorContains(t, err, "boom")
	for k := range l.f.objs {
		require.False(t, strings.HasPrefix(k, "mmap/"), "nothing is leaked: %s", k)
	}
}

func TestMemoryDataMappingLC_Checks(t *testing.T) {
	l := newTL(t, "MemoryDataMapping", "m")
	in := mappingInputs()
	ds := without(with(in, map[string]property.Value{"fromType": sv("DATASET_CLOUDAPP"), "datasetId": sv(mappingDataset)}), "columns")
	require.Empty(t, l.check(ds).Failures)
	one := func(m property.Value) property.Value { return property.New([]property.Value{m}) }
	for name, c := range map[string]struct {
		in   property.Map
		prop string
	}{
		"bad name":          {with(in, map[string]property.Value{"displayName": sv("1-bad")}), "displayName"},
		"empty description": {with(in, map[string]property.Value{"description": sv("")}), "description"},
		"unknown type":      {with(in, map[string]property.Value{"fromType": sv("FILE")}), "fromType"},
		"ingress no cols":   {without(in, "columns"), "columns"},
		"ingress dataset":   {with(in, map[string]property.Value{"datasetId": sv(mappingDataset)}), "datasetId"},
		"dataset no id":     {without(ds, "datasetId"), "datasetId"},
		"dataset bad id":    {with(ds, map[string]property.Value{"datasetId": sv("tdi_dataset_x")}), "datasetId"},
		"dataset columns":   {with(ds, map[string]property.Value{"columns": memStrList("a")}), "columns"},
		"no mappings":       {with(in, map[string]property.Value{"mappings": property.New([]property.Value{})}), "mappings"},
		"empty expression":  {with(in, map[string]property.Value{"mappings": one(traitMap("", "Contact", "email"))}), "mappings[0].expression"},
		"bad trait group":   {with(in, map[string]property.Value{"mappings": one(traitMap("a", "bad group", "email"))}), "mappings[0].traitGroup"},
		"bad trait name":    {with(in, map[string]property.Value{"mappings": one(traitMap("a", "Contact", "Email"))}), "mappings[0].traitName"},
	} {
		f := l.check(c.in).Failures
		require.NotEmpty(t, f, name)
		require.Equal(t, c.prop, f[0].Property, name)
	}
}

func identityConfig(kv map[string]property.Value) property.Value { return property.New(pv(kv)) }

func identityInputs() property.Map {
	return pv(map[string]property.Value{"storeId": sv(memStoreID),
		"identifierConfigs": property.New([]property.Value{
			identityConfig(map[string]property.Value{"idType": sv("email"), "normalization": sv("email")}),
			identityConfig(map[string]property.Value{"idType": sv("phone"), "normalization": sv("phone"), "enforceUnique": property.New(false)}),
		}),
		"matchingRules": memStrList("email", "phone")})
}

const identityPath = "/memory/v1/ControlPlane/Stores/" + memStoreID + "/IdentityResolutionSettings"

func TestMemoryIdentityResolutionSettingsLC_Lifecycle(t *testing.T) {
	fastKnowledgePolls(t)
	l := newTL(t, "MemoryIdentityResolutionSettings", "i")
	seedMemoryStore(l)
	in := identityInputs()
	require.Empty(t, l.check(in).Failures)
	c := l.create(in)
	require.Equal(t, memStoreID, c.ID)
	require.Equal(t, 1.0, c.Properties.Get("revision").AsNumber())
	require.JSONEq(t, `{"identifierConfigs":[{"idType":"email","normalization":"email"},{"idType":"phone","normalization":"phone","enforceUnique":false}],"matchingRules":["email","phone"]}`,
		l.f.callsMatching("PUT", identityPath)[0].Body)
	requireNoChanges(t, l.diff(c.ID, c.Properties, in, in))

	// Twilio reports every setting with its default; writing those defaults, or leaving them out, is the same.
	spelled := with(in, map[string]property.Value{"identifierConfigs": property.New([]property.Value{
		identityConfig(map[string]property.Value{"idType": sv("email"), "normalization": sv("email"), "matchingAlgo": sv("exact"), "matchingThreshold": nv(75),
			"limit": nv(100), "limitPolicy": sv("fifo"), "enforceUnique": property.New(true)}),
		identityConfig(map[string]property.Value{"idType": sv("phone"), "normalization": sv("phone"), "enforceUnique": property.New(false)}),
	})})
	requireNoChanges(t, l.diff(c.ID, c.Properties, spelled, in))
	rd := l.read(c.ID, c.Properties, in)
	require.Len(t, rd.Inputs.Get("identifierConfigs").AsArray().AsSlice(), 2)
	requireNoChanges(t, l.diff(c.ID, rd.Properties, in, rd.Inputs))
	requireNoChanges(t, l.diff(c.ID, rd.Properties, spelled, rd.Inputs))
	// A refresh keeps the program's spelling when the values mean the same.
	require.False(t, l.read(c.ID, c.Properties, in).Inputs.Get("identifierConfigs").AsArray().Get(0).AsMap().Get("matchingAlgo").IsString())

	// Drift is reported.
	l.f.mutate("mident", testAccount, memStoreID, func(o map[string]any) { o["matchingRules"] = []any{"phone"} })
	drift := l.read(c.ID, c.Properties, in)
	require.Equal(t, pgp.Update, l.diff(c.ID, drift.Properties, in, drift.Inputs).DetailedDiff["matchingRules"].Kind)
	l.f.mutate("mident", testAccount, memStoreID, func(o map[string]any) { o["matchingRules"] = []any{"email", "phone"} })

	// A change writes the whole document again.
	next := with(in, map[string]property.Value{"matchingRules": memStrList("phone", "email"), "identifierConfigs": property.New([]property.Value{
		identityConfig(map[string]property.Value{"idType": sv("email"), "normalization": sv("email"), "matchingAlgo": sv("fuzzy"), "matchingThreshold": nv(80)}),
		identityConfig(map[string]property.Value{"idType": sv("phone"), "normalization": sv("phone"), "enforceUnique": property.New(false)}),
	})})
	d := l.diff(c.ID, c.Properties, next, in)
	require.Equal(t, pgp.Update, d.DetailedDiff["identifierConfigs"].Kind)
	require.Equal(t, pgp.Update, d.DetailedDiff["matchingRules"].Kind)
	l.f.resetCalls()
	u := l.update(c.ID, c.Properties, next, in)
	require.JSONEq(t, `{"identifierConfigs":[{"idType":"email","normalization":"email","matchingAlgo":"fuzzy","matchingThreshold":80},{"idType":"phone","normalization":"phone","enforceUnique":false}],"matchingRules":["phone","email"]}`,
		l.f.callsMatching("PUT", identityPath)[0].Body)
	require.Equal(t, 2.0, u.Properties.Get("revision").AsNumber())
	requireNoChanges(t, l.diff(c.ID, u.Properties, next, next))
	require.Equal(t, pgp.UpdateReplace, l.diff(c.ID, u.Properties, with(next, map[string]property.Value{"storeId": sv("mem_store_other")}), next).DetailedDiff["storeId"].Kind)

	// Import adopts the settings; destroy only forgets.
	imp := l.read(c.ID, empty(), empty())
	require.Equal(t, "fuzzy", getS(t, imp.Inputs.Get("identifierConfigs").AsArray().Get(0).AsMap(), "matchingAlgo"))
	requireNoChanges(t, l.diff(c.ID, imp.Properties, next, imp.Inputs))
	l.requireImportFails("mem_store_missing")
	l.f.resetCalls()
	l.del(c.ID, u.Properties)
	require.Empty(t, l.f.callsMatching("DELETE", "/memory/"), "Twilio has no delete for the settings")
	require.NotNil(t, l.f.get("mident", testAccount, memStoreID))
}

func TestMemoryIdentityResolutionSettingsLC_Checks(t *testing.T) {
	l := newTL(t, "MemoryIdentityResolutionSettings", "i")
	in := identityInputs()
	cfg := func(kv map[string]property.Value) property.Value {
		return property.New([]property.Value{identityConfig(kv)})
	}
	for name, c := range map[string]struct {
		in   property.Map
		prop string
	}{
		"no configs":    {with(in, map[string]property.Value{"identifierConfigs": property.New([]property.Value{})}), "identifierConfigs"},
		"short id":      {with(in, map[string]property.Value{"identifierConfigs": cfg(map[string]property.Value{"idType": sv("e")})}), "identifierConfigs[0].idType"},
		"bad algo":      {with(in, map[string]property.Value{"identifierConfigs": cfg(map[string]property.Value{"idType": sv("email"), "matchingAlgo": sv("soundex")})}), "identifierConfigs[0].matchingAlgo"},
		"threshold":     {with(in, map[string]property.Value{"identifierConfigs": cfg(map[string]property.Value{"idType": sv("email"), "matchingThreshold": nv(100)})}), "identifierConfigs[0].matchingThreshold"},
		"limit":         {with(in, map[string]property.Value{"identifierConfigs": cfg(map[string]property.Value{"idType": sv("email"), "limit": nv(0)})}), "identifierConfigs[0].limit"},
		"limit policy":  {with(in, map[string]property.Value{"identifierConfigs": cfg(map[string]property.Value{"idType": sv("email"), "limitPolicy": sv("random")})}), "identifierConfigs[0].limitPolicy"},
		"normalization": {with(in, map[string]property.Value{"identifierConfigs": cfg(map[string]property.Value{"idType": sv("email"), "normalization": sv("upper")})}), "identifierConfigs[0].normalization"},
		"duplicate id":  {with(in, map[string]property.Value{"identifierConfigs": property.New([]property.Value{identityConfig(map[string]property.Value{"idType": sv("email")}), identityConfig(map[string]property.Value{"idType": sv("email")})})}), "identifierConfigs[1].idType"},
		"no rules":      {with(in, map[string]property.Value{"matchingRules": property.New([]property.Value{})}), "matchingRules"},
		"long rule":     {with(in, map[string]property.Value{"matchingRules": memStrList(strings.Repeat("r", 31))}), "matchingRules"},
	} {
		f := l.check(c.in).Failures
		require.NotEmpty(t, f, name)
		require.Equal(t, c.prop, f[0].Property, name)
	}
}

func TestLookup_MemoryDataMappingAndIdentitySettings(t *testing.T) {
	fastKnowledgePolls(t)
	l := newTL(t, "MemoryDataMapping", "m")
	seedMemoryStore(l)
	c := l.create(mappingInputs())
	r, err := invoke(t, l, "getMemoryDataMapping", map[string]property.Value{"storeId": sv(memStoreID), "dataMappingId": sv(getS(t, c.Properties, "dataMappingId"))})
	require.NoError(t, err)
	require.Equal(t, "crm-import", getS(t, r, "displayName"))
	require.Equal(t, "INGRESS", getS(t, r, "fromType"))
	require.True(t, r.Get("isEnabled").AsBool())
	require.Len(t, r.Get("mappings").AsArray().AsSlice(), 2)
	_, err = invoke(t, l, "getMemoryDataMapping", map[string]property.Value{"storeId": sv(memStoreID), "dataMappingId": sv("mem_datamapping_missing")})
	require.Error(t, err)
	_, err = invoke(t, l, "getMemoryDataMapping", map[string]property.Value{"storeId": sv(memStoreID), "dataMappingId": sv("")})
	require.Error(t, err)

	l.other("MemoryIdentityResolutionSettings", "i").create(identityInputs())
	r, err = invoke(t, l, "getMemoryIdentityResolutionSettings", map[string]property.Value{"storeId": sv(memStoreID)})
	require.NoError(t, err)
	require.Len(t, r.Get("identifierConfigs").AsArray().AsSlice(), 2)
	require.Equal(t, 1.0, r.Get("revision").AsNumber())
	_, err = invoke(t, l, "getMemoryIdentityResolutionSettings", map[string]property.Value{"storeId": sv("mem_store_none")})
	require.Error(t, err)
}
