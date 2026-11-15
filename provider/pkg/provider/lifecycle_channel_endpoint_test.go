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

const bundleEndpoint = "PN00000000000000000000000000000001"

func TestChannelEndpointAssignmentLC(t *testing.T) {
	for _, tc := range bundleCases {
		typ := tc.typ + "ChannelEndpointAssignment"
		t.Run(typ, func(t *testing.T) {
			bl := newTL(t, tc.typ, "b")
			l := bl.other(typ, "a")
			b := bl.create(bundleInputs())
			in := pv(map[string]property.Value{tc.param: sv(b.ID), "channelEndpointType": sv("phone-number"), "channelEndpointSid": sv(bundleEndpoint)})
			require.Empty(t, l.check(in).Failures)
			c := l.create(in)
			sid := getS(t, c.Properties, "sid")
			require.True(t, strings.HasPrefix(sid, "RA"), sid)
			require.Equal(t, b.ID+"/"+sid, c.ID)
			form := l.f.callsMatching("POST", "/trusthub/v1/"+tc.coll+"/"+b.ID+"/ChannelEndpointAssignments")[0].Form
			require.Equal(t, "phone-number", form.Get("ChannelEndpointType"))
			require.Equal(t, bundleEndpoint, form.Get("ChannelEndpointSid"))
			requireNoChanges(t, l.diff(c.ID, c.Properties, in, in))
			rd := l.read(c.ID, c.Properties, in)
			requireNoChanges(t, l.diff(c.ID, rd.Properties, in, rd.Inputs))

			// Twilio has no update call, so every input replaces.
			for k, v := range map[string]property.Value{tc.param: sv("BU00000000000000000000000000000002"), "channelEndpointType": sv("other"), "channelEndpointSid": sv("PN00000000000000000000000000000002")} {
				require.Equal(t, pgp.UpdateReplace, l.diff(c.ID, c.Properties, with(in, map[string]property.Value{k: v}), in).DetailedDiff[k].Kind, k)
			}

			// The same endpoint cannot be assigned twice; Twilio's reason surfaces.
			_, err := l.s.Create(pgp.CreateRequest{Urn: l.urn, Properties: l.checked(in)})
			require.ErrorContains(t, err, "already assigned")

			// Import adopts the whole assignment.
			imp := l.read(c.ID, empty(), empty())
			require.Equal(t, b.ID, getS(t, imp.Inputs, tc.param))
			require.Equal(t, "phone-number", getS(t, imp.Inputs, "channelEndpointType"))
			require.Equal(t, bundleEndpoint, getS(t, imp.Inputs, "channelEndpointSid"))
			requireNoChanges(t, l.diff(c.ID, imp.Properties, in, imp.Inputs))
			l.requireImportFails(b.ID + "/RA00000000000000000000000000000099")
			_, err = l.s.Read(pgp.ReadRequest{ID: "no-slash", Urn: l.urn, Properties: empty(), Inputs: empty()})
			require.ErrorContains(t, err, "expected <bundleSid>/<assignmentSid>")

			// Check.
			for name, bad := range map[string]property.Map{
				"bundle sid":   with(in, map[string]property.Value{tc.param: sv("x")}),
				"endpoint sid": with(in, map[string]property.Value{"channelEndpointSid": sv("PN1")}),
				"empty type":   with(in, map[string]property.Value{"channelEndpointType": sv("")}),
				"missing type": without(in, "channelEndpointType"),
			} {
				require.NotEmpty(t, l.check(bad).Failures, name)
			}

			// A bundle in review refuses to lose the assignment: the error says so and the state is kept.
			l.f.mutate("th:"+tc.coll, testAccount, b.ID, func(o map[string]any) { o["status"] = "in-review" })
			err = l.s.Delete(pgp.DeleteRequest{ID: c.ID, Urn: l.urn, Properties: c.Properties})
			require.ErrorContains(t, err, "in review")
			l.f.mutate("th:"+tc.coll, testAccount, b.ID, func(o map[string]any) { o["status"] = "draft" })

			l.del(c.ID, c.Properties)
			require.Empty(t, l.read(c.ID, c.Properties, in).ID)
			l.del(c.ID, c.Properties)
		})
	}
}

func TestLookup_ChannelEndpointAssignments(t *testing.T) {
	for _, tc := range bundleCases {
		typ := tc.typ + "ChannelEndpointAssignment"
		t.Run(typ, func(t *testing.T) {
			bl := newTL(t, tc.typ, "b")
			b := bl.create(bundleInputs())
			a := bl.other(typ, "a").create(pv(map[string]property.Value{tc.param: sv(b.ID), "channelEndpointType": sv("phone-number"), "channelEndpointSid": sv(bundleEndpoint)}))
			r, err := invoke(t, bl, "get"+typ, map[string]property.Value{tc.param: sv(b.ID), "sid": sv(getS(t, a.Properties, "sid"))})
			require.NoError(t, err)
			require.Equal(t, "phone-number", getS(t, r, "channelEndpointType"))
			require.Equal(t, bundleEndpoint, getS(t, r, "channelEndpointSid"))
			_, err = invoke(t, bl, "get"+typ, map[string]property.Value{tc.param: sv(b.ID), "sid": sv("RA00000000000000000000000000000099")})
			require.Error(t, err)
			_, err = invoke(t, bl, "get"+typ, map[string]property.Value{tc.param: sv(b.ID), "sid": sv("")})
			require.Error(t, err)
		})
	}
}
