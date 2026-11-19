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

	"github.com/pulumi/pulumi/sdk/v3/go/property"
	"github.com/stretchr/testify/require"
)

// Issue pulumi-labs/pulumi-twilio#51: a list element wired to another
// resource's output is a preview-time unknown inside a known list. infer
// decodes it to its zero value, so an element-wise Check must skip it rather
// than reject "". Every list whose elements Check validates is covered here;
// a known bad element must still fail.
func TestCheck_UnknownListElementsAreSkipped(t *testing.T) {
	const (
		fv = "FV00000000000000000000000000000001"
		ge = "GE00000000000000000000000000000001"
		hw = "HW00000000000000000000000000000001"
	)
	list := func(vs ...property.Value) property.Value { return property.New(vs) }
	obj := func(kv map[string]property.Value) property.Value { return property.New(pv(kv)) }

	tests := []struct {
		name, typ string
		base      property.Map
		prop      string
		// ok has a computed element beside a valid one; bad adds a known invalid one.
		ok, bad property.Value
	}{
		{"pluginVersionSids", "FlexPluginConfiguration", pv(map[string]property.Value{"name": sv("c")}), "pluginVersionSids",
			list(sv(fv), computed()), list(computed(), sv("nope"))},
		{"questionSids", "FlexInsightsQuestionnaire", pv(map[string]property.Value{"name": sv("q")}), "questionSids",
			list(sv(ge), computed()), list(computed(), sv("nope"))},
		{"networks", "SuperSimNetworkAccessProfile", pv(map[string]property.Value{"uniqueName": sv("p")}), "networks",
			list(sv(hw), computed()), list(computed(), sv("nope"))},
		{"columns", "MemoryDataMapping", without(mappingInputs(), "columns"), "columns",
			list(sv("Email"), computed()), list(computed(), sv(""))},
		{"mappings.expression", "MemoryDataMapping", without(mappingInputs(), "mappings"), "mappings",
			list(traitMap("Email", "Contact", "email"), computed()), list(computed(), traitMap("", "Contact", "email"))},
		{"mappings.fields", "MemoryDataMapping", without(mappingInputs(), "mappings"), "mappings",
			list(obj(map[string]property.Value{"expression": computed(), "traitGroup": computed(), "traitName": computed()})),
			list(obj(map[string]property.Value{"expression": sv("Email"), "traitGroup": sv("Contact"), "traitName": sv("Bad Name")}))},
		{"identifierConfigs", "MemoryIdentityResolutionSettings", without(identityInputs(), "identifierConfigs"), "identifierConfigs",
			list(identityConfig(map[string]property.Value{"idType": sv("email")}), computed(), identityConfig(map[string]property.Value{"idType": computed(), "matchingAlgo": computed(), "limit": computed()})),
			list(computed(), identityConfig(map[string]property.Value{"idType": sv("email"), "matchingAlgo": sv("bogus")}))},
		{"matchingRules", "MemoryIdentityResolutionSettings", without(identityInputs(), "matchingRules"), "matchingRules",
			list(sv("email"), computed()), list(computed(), sv(""))},
		{"traits", "MemoryTraitGroup", without(groupInputs(), "traits"), "traits",
			property.New(pv(map[string]property.Value{"a": computed(), "b": obj(map[string]property.Value{"dataType": computed()})})),
			property.New(pv(map[string]property.Value{"a": computed(), "b": obj(map[string]property.Value{"dataType": sv("WRONG")})}))},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			l := newTL(t, tc.typ, "x")
			require.Empty(t, l.check(with(tc.base, map[string]property.Value{tc.prop: tc.ok})).Failures, "computed element must not fail Check")
			require.NotEmpty(t, l.check(with(tc.base, map[string]property.Value{tc.prop: tc.bad})).Failures, "a known bad element must still fail Check")
		})
	}
}
