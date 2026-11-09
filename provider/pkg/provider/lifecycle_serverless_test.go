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

func serverlessServiceInputs() property.Map {
	return pv(map[string]property.Value{"uniqueName": sv("owl-api"), "friendlyName": sv("Owl API")})
}

// serverlessSvc creates a Serverless service on the fake and returns its SID.
func serverlessSvc(l *tl) string {
	return l.other("ServerlessService", "svc").create(serverlessServiceInputs()).ID
}

func serverlessEnv(l *tl, svc string) string {
	c := l.other("ServerlessEnvironment", "env").create(pv(map[string]property.Value{"serviceSid": sv(svc), "uniqueName": sv("stage")}))
	return getS(l.t, c.Properties, "sid")
}

func TestServerlessServiceLC_Lifecycle(t *testing.T) {
	l := newTL(t, "ServerlessService", "svc")
	in := serverlessServiceInputs()
	c := l.create(in)
	require.True(t, strings.HasPrefix(c.ID, "ZS"), c.ID)
	require.Equal(t, c.ID, getS(t, c.Properties, "sid"))
	require.Equal(t, testAccount, getS(t, c.Properties, "accountSid"))
	require.NotEmpty(t, getS(t, c.Properties, "domainBase"))
	require.Equal(t, "2026-11-06T00:00:00Z", getS(t, c.Properties, "dateCreated"))
	form := l.f.callsMatching("POST", "/serverless/v1/Services")[0].Form
	require.Equal(t, "owl-api", form.Get("UniqueName"))
	require.Equal(t, "Owl API", form.Get("FriendlyName"))
	require.Empty(t, form["IncludeCredentials"], "an unset flag is not sent")
	require.Empty(t, form["UiEditable"], "an unset flag is not sent")
	requireNoChanges(t, l.diff(c.ID, c.Properties, in, in))

	// A refresh changes nothing; an out-of-band rename is drift.
	rd := l.read(c.ID, c.Properties, in)
	requireNoChanges(t, l.diff(c.ID, rd.Properties, in, rd.Inputs))
	require.False(t, hasProp(rd.Inputs, "includeCredentials"), "a flag the program left unset stays unset")
	l.f.mutate("zs", testAccount, c.ID, func(o map[string]any) { o["friendly_name"] = "edited" })
	rd = l.read(c.ID, c.Properties, in)
	require.Equal(t, "edited", getS(t, rd.Inputs, "friendlyName"))
	d := l.diff(c.ID, rd.Properties, in, rd.Inputs)
	require.Equal(t, pgp.Update, d.DetailedDiff["friendlyName"].Kind)
	require.Len(t, d.DetailedDiff, 1)

	// The name and both flags update in place; the unique name replaces.
	next := with(in, map[string]property.Value{"friendlyName": sv("Owl API v2"), "includeCredentials": bv(false), "uiEditable": bv(true)})
	d = l.diff(c.ID, c.Properties, next, in)
	require.Equal(t, pgp.Update, d.DetailedDiff["friendlyName"].Kind)
	require.Equal(t, pgp.Update, d.DetailedDiff["includeCredentials"].Kind)
	require.Equal(t, pgp.Update, d.DetailedDiff["uiEditable"].Kind)
	l.f.resetCalls()
	u := l.update(c.ID, c.Properties, next, in)
	require.Equal(t, "Owl API v2", l.f.get("zs", testAccount, c.ID)["friendly_name"])
	require.Equal(t, false, l.f.get("zs", testAccount, c.ID)["include_credentials"])
	require.Equal(t, true, l.f.get("zs", testAccount, c.ID)["ui_editable"])
	require.Equal(t, "2026-11-07T00:00:00Z", getS(t, u.Properties, "dateUpdated"))
	requireNoChanges(t, l.diff(c.ID, u.Properties, next, next))
	// Dropping a flag from the program keeps what Twilio has.
	requireNoChanges(t, l.diff(c.ID, u.Properties, without(next, "includeCredentials", "uiEditable"), next))
	d = l.diff(c.ID, u.Properties, with(next, map[string]property.Value{"uniqueName": sv("other")}), next)
	require.Equal(t, pgp.UpdateReplace, d.DetailedDiff["uniqueName"].Kind)

	// Gone out of band: a refresh drops it, and deleting twice is fine.
	l.f.remove("zs", testAccount, c.ID)
	require.Empty(t, l.read(c.ID, u.Properties, next).ID)
	l.del(c.ID, u.Properties)
	c = l.create(in)
	l.del(c.ID, c.Properties)
	require.Nil(t, l.f.get("zs", testAccount, c.ID))
}

func TestServerlessServiceLC_ImportAndCheck(t *testing.T) {
	l := newTL(t, "ServerlessService", "svc")
	in := serverlessServiceInputs()
	c := l.create(in)

	// Import adopts the names and the reported flags; a program that omits the flags plans nothing.
	rd := l.read(c.ID, empty(), empty())
	require.Equal(t, c.ID, rd.ID)
	require.Equal(t, "owl-api", getS(t, rd.Inputs, "uniqueName"))
	require.Equal(t, "Owl API", getS(t, rd.Inputs, "friendlyName"))
	requireNoChanges(t, l.diff(c.ID, rd.Properties, in, rd.Inputs))
	requireNoChanges(t, l.diff(c.ID, rd.Properties, rd.Inputs, rd.Inputs))

	for name, bad := range map[string]property.Map{
		"uniqueName":   with(in, map[string]property.Value{"uniqueName": sv(strings.Repeat("a", 51))}),
		"friendlyName": with(in, map[string]property.Value{"friendlyName": sv("")}),
	} {
		chk := l.check(bad)
		require.Len(t, chk.Failures, 1, name)
		require.Equal(t, name, chk.Failures[0].Property)
	}
	// Unknown values are not judged at preview.
	require.Empty(t, l.check(with(in, map[string]property.Value{"uniqueName": computed()})).Failures)

	// A preview creates nothing.
	l.f.resetCalls()
	_, err := l.s.Create(pgp.CreateRequest{Urn: l.urn, Properties: in, DryRun: true})
	require.NoError(t, err)
	require.Empty(t, l.f.callsMatching("POST", "/serverless/v1/Services"))
}

func TestServerlessEnvironmentLC_Lifecycle(t *testing.T) {
	l := newTL(t, "ServerlessEnvironment", "env")
	svc := serverlessSvc(l)
	in := pv(map[string]property.Value{"serviceSid": sv(svc), "uniqueName": sv("stage"), "domainSuffix": sv("stg")})
	c := l.create(in)
	sid := getS(t, c.Properties, "sid")
	require.True(t, strings.HasPrefix(sid, "ZE"), sid)
	require.Equal(t, svc+"/"+sid, c.ID)
	require.True(t, strings.HasSuffix(getS(t, c.Properties, "domainName"), "-stg.twil.io"), getS(t, c.Properties, "domainName"))
	require.False(t, hasProp(c.Properties, "buildSid"), "no build is deployed yet")
	form := l.f.callsMatching("POST", "/Environments")[0].Form
	require.Equal(t, "stage", form.Get("UniqueName"))
	require.Equal(t, "stg", form.Get("DomainSuffix"))
	requireNoChanges(t, l.diff(c.ID, c.Properties, in, in))

	// Twilio has no environment update: every input replaces.
	for prop, v := range map[string]string{"uniqueName": "prod", "domainSuffix": "prd", "serviceSid": "ZS" + strings.Repeat("0", 32)} {
		d := l.diff(c.ID, c.Properties, with(in, map[string]property.Value{prop: sv(v)}), in)
		require.Equal(t, pgp.UpdateReplace, d.DetailedDiff[prop].Kind, prop)
	}

	// A deployed build is reported on refresh.
	l.f.mutate("ze", testAccount, c.ID, func(o map[string]any) { o["build_sid"] = "ZB" + strings.Repeat("1", 32) })
	rd := l.read(c.ID, c.Properties, in)
	require.Equal(t, "ZB"+strings.Repeat("1", 32), getS(t, rd.Properties, "buildSid"))
	requireNoChanges(t, l.diff(c.ID, rd.Properties, in, rd.Inputs))

	l.f.remove("ze", testAccount, c.ID)
	require.Empty(t, l.read(c.ID, rd.Properties, in).ID)
	l.del(c.ID, rd.Properties)
}

func TestServerlessEnvironmentLC_ImportAndCheck(t *testing.T) {
	l := newTL(t, "ServerlessEnvironment", "env")
	svc := serverlessSvc(l)
	minimal := pv(map[string]property.Value{"serviceSid": sv(svc), "uniqueName": sv("dev")})
	c := l.create(minimal)
	require.Empty(t, l.f.callsMatching("POST", "/Environments")[0].Form["DomainSuffix"], "no suffix is sent when none is set")

	rd := l.read(c.ID, empty(), empty())
	require.Equal(t, svc, getS(t, rd.Inputs, "serviceSid"))
	require.Equal(t, "dev", getS(t, rd.Inputs, "uniqueName"))
	require.False(t, hasProp(rd.Inputs, "domainSuffix"))
	requireNoChanges(t, l.diff(c.ID, rd.Properties, minimal, rd.Inputs))

	_, err := l.s.Read(pgp.ReadRequest{ID: "nonsense", Urn: l.urn, Properties: empty(), Inputs: empty()})
	require.ErrorContains(t, err, "<serviceSid>/<sid>")
	for name, bad := range map[string]property.Map{
		"serviceSid":   with(minimal, map[string]property.Value{"serviceSid": sv("ZSshort")}),
		"uniqueName":   with(minimal, map[string]property.Value{"uniqueName": sv("")}),
		"domainSuffix": with(minimal, map[string]property.Value{"domainSuffix": sv(strings.Repeat("s", 17))}),
	} {
		chk := l.check(bad)
		require.Len(t, chk.Failures, 1, name)
		require.Equal(t, name, chk.Failures[0].Property)
	}
}

func serverlessVariableInputs(svc, env string) property.Map {
	return pv(map[string]property.Value{"serviceSid": sv(svc), "environmentSid": sv(env), "key": sv("API_URL"), "value": sv("https://example.com")})
}

func TestServerlessVariableLC_Lifecycle(t *testing.T) {
	l := newTL(t, "ServerlessVariable", "var")
	svc := serverlessSvc(l)
	env := serverlessEnv(l, svc)
	in := serverlessVariableInputs(svc, env)
	c := l.create(in)
	sid := getS(t, c.Properties, "sid")
	require.True(t, strings.HasPrefix(sid, "ZV"), sid)
	require.Equal(t, svc+"/"+env+"/"+sid, c.ID)
	v, _ := c.Properties.GetOk("value")
	require.True(t, v.Secret(), "the value is a Pulumi secret")
	form := l.f.callsMatching("POST", "/Variables")[0].Form
	require.Equal(t, "API_URL", form.Get("Key"))
	require.Equal(t, "https://example.com", form.Get("Value"))
	requireNoChanges(t, l.diff(c.ID, c.Properties, in, in))

	// Drift in the value is seen on refresh.
	rd := l.read(c.ID, c.Properties, in)
	requireNoChanges(t, l.diff(c.ID, rd.Properties, in, rd.Inputs))
	l.f.mutate("zv", testAccount, c.ID, func(o map[string]any) { o["value"] = "tampered" })
	rd = l.read(c.ID, c.Properties, in)
	require.Equal(t, "tampered", getS(t, rd.Inputs, "value"))
	d := l.diff(c.ID, rd.Properties, in, rd.Inputs)
	require.Equal(t, pgp.Update, d.DetailedDiff["value"].Kind)
	require.Len(t, d.DetailedDiff, 1)

	// Key and value update in place; the parents replace.
	next := with(in, map[string]property.Value{"key": sv("API_BASE"), "value": sv("https://example.org")})
	d = l.diff(c.ID, c.Properties, next, in)
	require.Equal(t, pgp.Update, d.DetailedDiff["key"].Kind)
	require.Equal(t, pgp.Update, d.DetailedDiff["value"].Kind)
	l.f.resetCalls()
	u := l.update(c.ID, c.Properties, next, in)
	require.Equal(t, "API_BASE", l.f.get("zv", testAccount, c.ID)["key"])
	require.Equal(t, "https://example.org", l.f.get("zv", testAccount, c.ID)["value"])
	require.Equal(t, "2026-11-07T00:00:00Z", getS(t, u.Properties, "dateUpdated"))
	require.Len(t, l.f.callsMatching("POST", "/Variables/"+sid), 1)
	requireNoChanges(t, l.diff(c.ID, u.Properties, next, next))
	for prop, val := range map[string]string{"serviceSid": "ZS" + strings.Repeat("0", 32), "environmentSid": "ZE" + strings.Repeat("0", 32)} {
		d = l.diff(c.ID, u.Properties, with(next, map[string]property.Value{prop: sv(val)}), next)
		require.Equal(t, pgp.UpdateReplace, d.DetailedDiff[prop].Kind, prop)
	}

	l.f.remove("zv", testAccount, c.ID)
	require.Empty(t, l.read(c.ID, u.Properties, next).ID)
	l.del(c.ID, u.Properties)
}

func TestServerlessVariableLC_ImportAndCheck(t *testing.T) {
	l := newTL(t, "ServerlessVariable", "var")
	svc := serverlessSvc(l)
	env := serverlessEnv(l, svc)
	in := serverlessVariableInputs(svc, env)
	c := l.create(in)
	rd := l.read(c.ID, empty(), empty())
	require.Equal(t, svc, getS(t, rd.Inputs, "serviceSid"))
	require.Equal(t, env, getS(t, rd.Inputs, "environmentSid"))
	require.Equal(t, "API_URL", getS(t, rd.Inputs, "key"))
	require.Equal(t, "https://example.com", getS(t, rd.Inputs, "value"))
	requireNoChanges(t, l.diff(c.ID, rd.Properties, in, rd.Inputs))

	_, err := l.s.Read(pgp.ReadRequest{ID: svc + "/" + env, Urn: l.urn, Properties: empty(), Inputs: empty()})
	require.ErrorContains(t, err, "<serviceSid>/<environmentSid>/<sid>")
	for name, bad := range map[string]property.Map{
		"serviceSid":     with(in, map[string]property.Value{"serviceSid": sv("nope")}),
		"environmentSid": with(in, map[string]property.Value{"environmentSid": sv("ZSwrongkind")}),
		"key":            with(in, map[string]property.Value{"key": sv("")}),
	} {
		chk := l.check(bad)
		require.Len(t, chk.Failures, 1, name)
		require.Equal(t, name, chk.Failures[0].Property)
	}
}

// Twilio may omit a variable's value on read: a refresh keeps the recorded value instead of blanking it.
func TestServerlessVariableLC_ReadKeepsValueWhenOmitted(t *testing.T) {
	l := newTL(t, "ServerlessVariable", "var")
	svc := serverlessSvc(l)
	env := serverlessEnv(l, svc)
	in := serverlessVariableInputs(svc, env)
	c := l.create(in)
	sid := getS(t, c.Properties, "sid")
	l.f.mutate("zv", testAccount, svc+"/"+env+"/"+sid, func(o map[string]any) { delete(o, "value") })
	rd := l.read(c.ID, c.Properties, in)
	require.Equal(t, "https://example.com", getS(t, rd.Inputs, "value"))
	requireNoChanges(t, l.diff(c.ID, rd.Properties, in, rd.Inputs))
}

// A Function and an Asset are the same shape: a container with a friendly name.
func TestServerlessContainerLC_Lifecycle(t *testing.T) {
	for _, tc := range []struct{ typ, kind, coll, noun string }{
		{"ServerlessFunction", "zf", "/Functions", "Function"},
		{"ServerlessAsset", "za", "/Assets", "Asset"},
	} {
		t.Run(tc.typ, func(t *testing.T) {
			l := newTL(t, tc.typ, "c")
			svc := serverlessSvc(l)
			in := pv(map[string]property.Value{"serviceSid": sv(svc), "friendlyName": sv("hello")})
			c := l.create(in)
			sid := getS(t, c.Properties, "sid")
			require.True(t, strings.HasPrefix(sid, "ZH"), sid)
			require.Equal(t, svc+"/"+sid, c.ID)
			require.Equal(t, "hello", l.f.callsMatching("POST", tc.coll)[0].Form.Get("FriendlyName"))
			requireNoChanges(t, l.diff(c.ID, c.Properties, in, in))

			l.f.mutate(tc.kind, testAccount, c.ID, func(o map[string]any) { o["friendly_name"] = "edited" })
			rd := l.read(c.ID, c.Properties, in)
			require.Equal(t, "edited", getS(t, rd.Inputs, "friendlyName"))
			d := l.diff(c.ID, rd.Properties, in, rd.Inputs)
			require.Equal(t, pgp.Update, d.DetailedDiff["friendlyName"].Kind)

			next := with(in, map[string]property.Value{"friendlyName": sv("renamed")})
			u := l.update(c.ID, c.Properties, next, in)
			require.Equal(t, "renamed", l.f.get(tc.kind, testAccount, c.ID)["friendly_name"])
			require.Equal(t, "2026-11-07T00:00:00Z", getS(t, u.Properties, "dateUpdated"))
			requireNoChanges(t, l.diff(c.ID, u.Properties, next, next))
			d = l.diff(c.ID, u.Properties, with(next, map[string]property.Value{"serviceSid": sv("ZS" + strings.Repeat("0", 32))}), next)
			require.Equal(t, pgp.UpdateReplace, d.DetailedDiff["serviceSid"].Kind)

			// Import adopts the service and name.
			imp := l.read(c.ID, empty(), empty())
			require.Equal(t, svc, getS(t, imp.Inputs, "serviceSid"))
			require.Equal(t, "renamed", getS(t, imp.Inputs, "friendlyName"))
			requireNoChanges(t, l.diff(c.ID, imp.Properties, next, imp.Inputs))
			_, err := l.s.Read(pgp.ReadRequest{ID: "nonsense", Urn: l.urn, Properties: empty(), Inputs: empty()})
			require.ErrorContains(t, err, "<serviceSid>/<sid>")

			for name, bad := range map[string]property.Map{
				"serviceSid":   with(in, map[string]property.Value{"serviceSid": sv("ZSshort")}),
				"friendlyName": with(in, map[string]property.Value{"friendlyName": sv("")}),
			} {
				chk := l.check(bad)
				require.Len(t, chk.Failures, 1, name)
				require.Equal(t, name, chk.Failures[0].Property)
			}

			l.f.remove(tc.kind, testAccount, c.ID)
			require.Empty(t, l.read(c.ID, u.Properties, next).ID)
			l.del(c.ID, u.Properties)
		})
	}
}

// Deleting a service takes its children with it, so a later refresh of a child finds nothing and its
// own delete is a no-op.
func TestServerlessServiceLC_DeleteCascades(t *testing.T) {
	l := newTL(t, "ServerlessService", "svc")
	sc := l.create(serverlessServiceInputs())
	svc := sc.ID
	env := l.other("ServerlessEnvironment", "env").create(pv(map[string]property.Value{"serviceSid": sv(svc), "uniqueName": sv("stage")}))
	l.del(svc, sc.Properties)
	require.Empty(t, l.other("ServerlessEnvironment", "env").read(env.ID, env.Properties, empty()).ID)
	l.other("ServerlessEnvironment", "env").del(env.ID, env.Properties)
}

func TestLookup_Serverless(t *testing.T) {
	l := newTL(t, "ServerlessService", "svc")
	svc := serverlessSvc(l)
	env := serverlessEnv(l, svc)
	r, err := invoke(t, l, "getServerlessService", map[string]property.Value{"serviceSid": sv(svc)})
	require.NoError(t, err)
	require.Equal(t, svc, getS(t, r, "sid"))
	require.Equal(t, "owl-api", getS(t, r, "uniqueName"))
	require.Equal(t, "Owl API", getS(t, r, "friendlyName"))
	require.NotEmpty(t, getS(t, r, "domainBase"))
	require.True(t, r.Get("includeCredentials").AsBool())
	_, err = invoke(t, l, "getServerlessService", map[string]property.Value{"serviceSid": sv("ZSmissing")})
	require.ErrorContains(t, err, "was not found")
	_, err = invoke(t, l, "getServerlessService", map[string]property.Value{"serviceSid": sv("")})
	require.ErrorContains(t, err, "serviceSid must be set")

	r, err = invoke(t, l, "getServerlessEnvironment", map[string]property.Value{"serviceSid": sv(svc), "environmentSid": sv(env)})
	require.NoError(t, err)
	require.Equal(t, "stage", getS(t, r, "uniqueName"))
	require.True(t, strings.HasSuffix(getS(t, r, "domainName"), ".twil.io"))
	_, err = invoke(t, l, "getServerlessEnvironment", map[string]property.Value{"serviceSid": sv(svc), "environmentSid": sv("ZEmissing")})
	require.ErrorContains(t, err, "was not found")

	v := l.other("ServerlessVariable", "var").create(serverlessVariableInputs(svc, env))
	r, err = invoke(t, l, "getServerlessVariable", map[string]property.Value{"serviceSid": sv(svc), "environmentSid": sv(env), "variableSid": sv(getS(t, v.Properties, "sid"))})
	require.NoError(t, err)
	require.Equal(t, "API_URL", getS(t, r, "key"))
	require.Equal(t, "https://example.com", getS(t, r, "value"))
	_, err = invoke(t, l, "getServerlessVariable", map[string]property.Value{"serviceSid": sv(svc), "environmentSid": sv(env), "variableSid": sv("ZVmissing")})
	require.ErrorContains(t, err, "was not found")

	fn := l.other("ServerlessFunction", "fn").create(pv(map[string]property.Value{"serviceSid": sv(svc), "friendlyName": sv("hello")}))
	r, err = invoke(t, l, "getServerlessFunction", map[string]property.Value{"serviceSid": sv(svc), "functionSid": sv(getS(t, fn.Properties, "sid"))})
	require.NoError(t, err)
	require.Equal(t, "hello", getS(t, r, "friendlyName"))
	_, err = invoke(t, l, "getServerlessFunction", map[string]property.Value{"serviceSid": sv(svc), "functionSid": sv("ZHmissing")})
	require.ErrorContains(t, err, "was not found")

	as := l.other("ServerlessAsset", "as").create(pv(map[string]property.Value{"serviceSid": sv(svc), "friendlyName": sv("logo")}))
	r, err = invoke(t, l, "getServerlessAsset", map[string]property.Value{"serviceSid": sv(svc), "assetSid": sv(getS(t, as.Properties, "sid"))})
	require.NoError(t, err)
	require.Equal(t, "logo", getS(t, r, "friendlyName"))
	_, err = invoke(t, l, "getServerlessAsset", map[string]property.Value{"serviceSid": sv(svc), "assetSid": sv("")})
	require.ErrorContains(t, err, "assetSid must be set")
}

// The identifiers fixed at creation stay known in an update preview; what Twilio reports stays unknown.
func TestWiring_ServerlessPreviewUpdate(t *testing.T) {
	l := newTL(t, "ServerlessService", "svc")
	in := serverlessServiceInputs()
	c := l.create(in)
	u := previewUpdate(t, l, c.ID, c.Properties, in, with(in, map[string]property.Value{"friendlyName": sv("x")}))
	requireKnown(t, u.Properties, "sid", "domainBase", "dateCreated")
	requireComputed(t, u.Properties, "dateUpdated")

	svc := c.ID
	vl := l.other("ServerlessVariable", "var")
	env := serverlessEnv(l, svc)
	vIn := serverlessVariableInputs(svc, env)
	v := vl.create(vIn)
	u = previewUpdate(t, vl, v.ID, v.Properties, vIn, with(vIn, map[string]property.Value{"value": sv("y")}))
	requireKnown(t, u.Properties, "sid", "dateCreated")
	requireComputed(t, u.Properties, "dateUpdated")

	fl := l.other("ServerlessFunction", "fn")
	fIn := pv(map[string]property.Value{"serviceSid": sv(svc), "friendlyName": sv("hello")})
	f := fl.create(fIn)
	u = previewUpdate(t, fl, f.ID, f.Properties, fIn, with(fIn, map[string]property.Value{"friendlyName": sv("z")}))
	requireKnown(t, u.Properties, "sid", "dateCreated")
	requireComputed(t, u.Properties, "dateUpdated")
}
