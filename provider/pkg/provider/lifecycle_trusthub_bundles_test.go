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

const (
	bundlePolicy = "RN00000000000000000000000000000001"
	bundleObject = "IT00000000000000000000000000000001"
)

var bundleCases = []struct{ typ, assign, coll, param string }{
	{"CustomerProfile", "CustomerProfileEntityAssignment", "CustomerProfiles", "customerProfileSid"},
	{"TrustProduct", "TrustProductEntityAssignment", "TrustProducts", "trustProductSid"},
}

func bundleInputs() property.Map {
	return pv(map[string]property.Value{"friendlyName": sv("Acme profile"), "email": sv("compliance@example.com"), "policySid": sv(bundlePolicy),
		"statusCallback": sv("https://hooks.example.com/status")})
}

func TestBundleLC_Lifecycle(t *testing.T) {
	for _, tc := range bundleCases {
		t.Run(tc.typ, func(t *testing.T) {
			l := newTL(t, tc.typ, "b")
			base := "/trusthub/v1/" + tc.coll
			in := bundleInputs()
			c := l.create(in)
			require.True(t, strings.HasPrefix(c.ID, "BU"), c.ID)
			require.Equal(t, "draft", getS(t, c.Properties, "status"))
			form := l.f.callsMatching("POST", base)[0].Form
			require.Equal(t, "Acme profile", form.Get("FriendlyName"))
			require.Equal(t, "compliance@example.com", form.Get("Email"))
			require.Equal(t, bundlePolicy, form.Get("PolicySid"))
			require.Equal(t, "https://hooks.example.com/status", form.Get("StatusCallback"))
			require.Empty(t, form.Get("Status"), "creating does not submit")
			requireNoChanges(t, l.diff(c.ID, c.Properties, in, in))

			rd := l.read(c.ID, c.Properties, in)
			requireNoChanges(t, l.diff(c.ID, rd.Properties, in, rd.Inputs))
			l.f.mutate("th:"+tc.coll, testAccount, c.ID, func(o map[string]any) { o["email"] = "other@example.com" })
			require.Equal(t, "other@example.com", getS(t, l.read(c.ID, c.Properties, in).Inputs, "email"))
			l.f.mutate("th:"+tc.coll, testAccount, c.ID, func(o map[string]any) { o["email"] = "compliance@example.com" })

			// Name, email and callback update in place; the policy replaces.
			next := with(in, map[string]property.Value{"friendlyName": sv("Renamed"), "email": sv("new@example.com"), "statusCallback": sv("https://hooks.example.com/v2")})
			d := l.diff(c.ID, c.Properties, next, in)
			for _, k := range []string{"friendlyName", "email", "statusCallback"} {
				require.Equal(t, pgp.Update, d.DetailedDiff[k].Kind, k)
			}
			l.f.resetCalls()
			u := l.update(c.ID, c.Properties, next, in)
			require.Equal(t, "new@example.com", l.f.get("th:"+tc.coll, testAccount, c.ID)["email"])
			require.Empty(t, l.f.callsMatching("POST", base+"/"+c.ID)[0].Form.Get("Status"))
			require.Equal(t, pgp.UpdateReplace, l.diff(c.ID, u.Properties, with(next, map[string]property.Value{"policySid": sv("RN00000000000000000000000000000002")}), next).DetailedDiff["policySid"].Kind)
			// Removing the callback is not a change.
			requireNoChanges(t, l.diff(c.ID, u.Properties, without(next, "statusCallback"), next))

			l.del(c.ID, u.Properties)
			require.Nil(t, l.f.get("th:"+tc.coll, testAccount, c.ID))
			l.del(c.ID, u.Properties)
			require.Empty(t, l.read(c.ID, u.Properties, next).ID)
		})
	}
}

func TestBundleLC_SubmitIsOneWay(t *testing.T) {
	for _, tc := range bundleCases {
		t.Run(tc.typ, func(t *testing.T) {
			l := newTL(t, tc.typ, "b")
			al := l.other(tc.assign, "a")
			base := "/trusthub/v1/" + tc.coll
			in := with(bundleInputs(), map[string]property.Value{"submit": bv(true)})

			// Twilio refuses a bundle that is not compliant, so nothing is left behind.
			_, err := l.s.Create(pgp.CreateRequest{Urn: l.urn, Properties: l.checked(in)})
			require.ErrorContains(t, err, "not compliant")
			require.ErrorContains(t, err, "the draft was deleted")
			for k := range l.f.objs {
				require.False(t, strings.HasPrefix(k, "th:"+tc.coll+"/"), "the draft is gone: %s", k)
			}

			// Draft, assign, then submit by updating.
			draft := with(in, map[string]property.Value{"submit": bv(false)})
			c := l.create(draft)
			asg := al.create(pv(map[string]property.Value{tc.param: sv(c.ID), "objectSid": sv(bundleObject)}))
			require.Equal(t, c.ID+"/"+getS(t, asg.Properties, "sid"), asg.ID)
			d := l.diff(c.ID, c.Properties, in, draft)
			require.Equal(t, pgp.Update, d.DetailedDiff["submit"].Kind)
			l.f.resetCalls()
			u := l.update(c.ID, c.Properties, in, draft)
			require.Equal(t, "pending-review", l.f.callsMatching("POST", base+"/"+c.ID)[0].Form.Get("Status"))
			require.Equal(t, "pending-review", getS(t, u.Properties, "status"))

			// Submitting cannot be undone: turning it off or removing it is no change, and a later update does not submit again.
			requireNoChanges(t, l.diff(c.ID, u.Properties, draft, in))
			requireNoChanges(t, l.diff(c.ID, u.Properties, without(in, "submit"), in))
			l.f.resetCalls()
			l.update(c.ID, u.Properties, with(in, map[string]property.Value{"friendlyName": sv("again")}), in)
			require.Empty(t, l.f.callsMatching("POST", base+"/"+c.ID)[0].Form.Get("Status"))

			// A bundle in review cannot be deleted, and Twilio's refusal surfaces.
			require.Error(t, l.s.Delete(pgp.DeleteRequest{ID: c.ID, Urn: l.urn, Properties: u.Properties}))
		})
	}
}

func TestBundleLC_ImportAndChecks(t *testing.T) {
	for _, tc := range bundleCases {
		t.Run(tc.typ, func(t *testing.T) {
			l := newTL(t, tc.typ, "b")
			in := bundleInputs()
			c := l.create(in)
			imp := l.read(c.ID, empty(), empty())
			require.Equal(t, "Acme profile", getS(t, imp.Inputs, "friendlyName"))
			require.Equal(t, bundlePolicy, getS(t, imp.Inputs, "policySid"))
			requireNoChanges(t, l.diff(c.ID, imp.Properties, in, imp.Inputs))
			l.requireImportFails("BU00000000000000000000000000000099")

			for name, bad := range map[string]property.Map{
				"friendlyName":   with(in, map[string]property.Value{"friendlyName": sv("")}),
				"email":          with(in, map[string]property.Value{"email": sv("not-an-email")}),
				"policySid":      with(in, map[string]property.Value{"policySid": sv("BU1")}),
				"statusCallback": with(in, map[string]property.Value{"statusCallback": sv("ftp://x")}),
			} {
				f := l.check(bad).Failures
				require.NotEmpty(t, f, name)
				require.Equal(t, name, f[0].Property)
			}
		})
	}
}

func TestBundleEntityAssignmentLC(t *testing.T) {
	for _, tc := range bundleCases {
		t.Run(tc.assign, func(t *testing.T) {
			bl := newTL(t, tc.typ, "b")
			l := bl.other(tc.assign, "a")
			b := bl.create(bundleInputs())
			in := pv(map[string]property.Value{tc.param: sv(b.ID), "objectSid": sv(bundleObject)})
			c := l.create(in)
			sid := getS(t, c.Properties, "sid")
			require.True(t, strings.HasPrefix(sid, "BV"), sid)
			require.Equal(t, b.ID+"/"+sid, c.ID)
			require.Equal(t, bundleObject, l.f.callsMatching("POST", "/trusthub/v1/"+tc.coll+"/"+b.ID+"/EntityAssignments")[0].Form.Get("ObjectSid"))
			requireNoChanges(t, l.diff(c.ID, c.Properties, in, in))
			rd := l.read(c.ID, c.Properties, in)
			requireNoChanges(t, l.diff(c.ID, rd.Properties, in, rd.Inputs))

			// Every input replaces.
			for k, v := range map[string]property.Value{tc.param: sv("BU00000000000000000000000000000002"), "objectSid": sv("RD00000000000000000000000000000001")} {
				require.Equal(t, pgp.UpdateReplace, l.diff(c.ID, c.Properties, with(in, map[string]property.Value{k: v}), in).DetailedDiff[k].Kind, k)
			}

			imp := l.read(c.ID, empty(), empty())
			require.Equal(t, b.ID, getS(t, imp.Inputs, tc.param))
			require.Equal(t, bundleObject, getS(t, imp.Inputs, "objectSid"))
			requireNoChanges(t, l.diff(c.ID, imp.Properties, in, imp.Inputs))
			l.requireImportFails(b.ID + "/BV00000000000000000000000000000099")
			_, err := l.s.Read(pgp.ReadRequest{ID: "no-slash", Urn: l.urn, Properties: empty(), Inputs: empty()})
			require.ErrorContains(t, err, "expected <bundleSid>/<assignmentSid>")

			require.NotEmpty(t, l.check(with(in, map[string]property.Value{"objectSid": sv("PN1")})).Failures)
			require.NotEmpty(t, l.check(with(in, map[string]property.Value{tc.param: sv("x")})).Failures)

			l.del(c.ID, c.Properties)
			require.Empty(t, l.read(c.ID, c.Properties, in).ID)
			l.del(c.ID, c.Properties)
		})
	}
}

func TestLookup_BundlesAndAssignments(t *testing.T) {
	for _, tc := range bundleCases {
		t.Run(tc.typ, func(t *testing.T) {
			bl := newTL(t, tc.typ, "b")
			b := bl.create(bundleInputs())
			a := bl.other(tc.assign, "a").create(pv(map[string]property.Value{tc.param: sv(b.ID), "objectSid": sv(bundleObject)}))
			r, err := invoke(t, bl, "get"+tc.typ, map[string]property.Value{"sid": sv(b.ID)})
			require.NoError(t, err)
			require.Equal(t, "Acme profile", getS(t, r, "friendlyName"))
			require.Equal(t, "draft", getS(t, r, "status"))
			require.Equal(t, bundlePolicy, getS(t, r, "policySid"))
			_, err = invoke(t, bl, "get"+tc.typ, map[string]property.Value{"sid": sv("BU00000000000000000000000000000099")})
			require.Error(t, err)
			_, err = invoke(t, bl, "get"+tc.typ, map[string]property.Value{"sid": sv("")})
			require.Error(t, err)

			r, err = invoke(t, bl, "get"+tc.assign, map[string]property.Value{tc.param: sv(b.ID), "sid": sv(getS(t, a.Properties, "sid"))})
			require.NoError(t, err)
			require.Equal(t, bundleObject, getS(t, r, "objectSid"))
			_, err = invoke(t, bl, "get"+tc.assign, map[string]property.Value{tc.param: sv(b.ID), "sid": sv("BV00000000000000000000000000000099")})
			require.Error(t, err)
		})
	}
}
