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

const flexPluginBase = "/flex-api/v1/PluginService"

func TestFlexPluginLC_Lifecycle(t *testing.T) {
	l := newTL(t, "FlexPlugin", "plugin")
	in := pv(map[string]property.Value{"uniqueName": sv("my-plugin"), "friendlyName": sv("My plugin")})
	c := l.create(in)
	require.True(t, strings.HasPrefix(c.ID, "FP"), c.ID)
	form := l.f.callsMatching("POST", flexPluginBase+"/Plugins")[0].Form
	require.Equal(t, "my-plugin", form.Get("UniqueName"))
	require.Equal(t, "My plugin", form.Get("FriendlyName"))
	require.Empty(t, form["Description"])
	requireNoChanges(t, l.diff(c.ID, c.Properties, in, in))
	rd := l.read(c.ID, c.Properties, in)
	requireNoChanges(t, l.diff(c.ID, rd.Properties, in, rd.Inputs))
	require.False(t, hasProp(rd.Inputs, "description"))

	// Friendly name and description update in place and the update does not resend the unique name.
	next := with(in, map[string]property.Value{"friendlyName": sv("Renamed"), "description": sv("What it does")})
	d := l.diff(c.ID, c.Properties, next, in)
	require.Equal(t, pgp.Update, d.DetailedDiff["friendlyName"].Kind)
	require.Equal(t, pgp.Update, d.DetailedDiff["description"].Kind)
	l.f.resetCalls()
	u := l.update(c.ID, c.Properties, next, in)
	uf := l.f.callsMatching("POST", flexPluginBase+"/Plugins/"+c.ID)[0].Form
	require.Equal(t, "Renamed", uf.Get("FriendlyName"))
	require.Empty(t, uf["UniqueName"])
	requireNoChanges(t, l.diff(c.ID, u.Properties, next, next))

	// Drift shows; the unique name replaces.
	l.f.mutate("flex", testAccount, "Plugins/"+c.ID, func(o map[string]any) { o["description"] = "edited" })
	rd = l.read(c.ID, u.Properties, next)
	require.Equal(t, "edited", getS(t, rd.Inputs, "description"))
	require.Equal(t, pgp.Update, l.diff(c.ID, rd.Properties, next, rd.Inputs).DetailedDiff["description"].Kind)
	d = l.diff(c.ID, u.Properties, with(next, map[string]property.Value{"uniqueName": sv("other")}), next)
	require.Equal(t, pgp.UpdateReplace, d.DetailedDiff["uniqueName"].Kind)

	// Import adopts what Twilio holds.
	imp := l.read(c.ID, empty(), empty())
	require.Empty(t, l.check(imp.Inputs).Failures)
	require.Equal(t, "my-plugin", getS(t, imp.Inputs, "uniqueName"))
	require.Equal(t, "edited", getS(t, imp.Inputs, "description"))
	l.requireImportFails("FP" + strings.Repeat("0", 32))

	// Destroy archives; an archived or missing Flex Plugin reads as deleted.
	l.del(c.ID, u.Properties)
	require.Len(t, l.f.callsMatching("POST", flexPluginBase+"/Plugins/"+c.ID+"/Archive"), 1)
	require.Empty(t, l.read(c.ID, u.Properties, next).ID)
	l.f.remove("flex", testAccount, "Plugins/"+c.ID)
	l.del(c.ID, u.Properties)

	require.Len(t, l.check(pv(map[string]property.Value{"uniqueName": sv("")})).Failures, 1)
}

func flexMakePlugin(t *testing.T, l *tl, name string) string {
	return l.other("FlexPlugin", "p-"+name).create(pv(map[string]property.Value{"uniqueName": sv(name)})).ID
}

func TestFlexPluginVersionLC_Lifecycle(t *testing.T) {
	l := newTL(t, "FlexPluginVersion", "ver")
	plugin := flexMakePlugin(t, l, "pv")
	in := pv(map[string]property.Value{"pluginSid": sv(plugin), "pluginVersion": sv("1.0.0"),
		"pluginUrl": sv("https://example.com/plugin.js"), "changelog": sv("first"), "cliVersion": sv("6.0.0")})
	c := l.create(in)
	require.True(t, strings.HasPrefix(c.ID, plugin+"/FV"), c.ID)
	sid := strings.TrimPrefix(c.ID, plugin+"/")
	form := l.f.callsMatching("POST", flexPluginBase+"/Plugins/"+plugin+"/Versions")[0].Form
	require.Equal(t, "1.0.0", form.Get("Version"))
	require.Equal(t, "https://example.com/plugin.js", form.Get("PluginUrl"))
	require.Equal(t, "6.0.0", form.Get("CliVersion"))
	require.Empty(t, form["Private"])
	requireNoChanges(t, l.diff(c.ID, c.Properties, in, in))

	// A refresh keeps the write-only cliVersion and stating the default is no change.
	rd := l.read(c.ID, c.Properties, in)
	require.Equal(t, "6.0.0", getS(t, rd.Inputs, "cliVersion"))
	require.False(t, hasProp(rd.Inputs, "private"))
	requireNoChanges(t, l.diff(c.ID, rd.Properties, in, rd.Inputs))
	requireNoChanges(t, l.diff(c.ID, c.Properties, with(in, map[string]property.Value{"private": bv(false)}), in))

	// Every input replaces the version.
	for k, v := range map[string]property.Value{"pluginVersion": sv("1.0.1"), "pluginUrl": sv("https://example.com/2.js"),
		"changelog": sv("second"), "private": bv(true), "cliVersion": sv("7"), "validateStatus": sv("ok"), "pluginSid": sv(plugin + "x")} {
		d := l.diff(c.ID, c.Properties, with(in, map[string]property.Value{k: v}), in)
		require.Equal(t, pgp.UpdateReplace, d.DetailedDiff[k].Kind, k)
	}

	// Import by <pluginSid>/<sid>.
	imp := l.read(c.ID, empty(), empty())
	require.Equal(t, plugin, getS(t, imp.Inputs, "pluginSid"))
	require.Equal(t, "1.0.0", getS(t, imp.Inputs, "pluginVersion"))
	require.Equal(t, "first", getS(t, imp.Inputs, "changelog"))
	require.False(t, hasProp(imp.Inputs, "cliVersion"))
	l.requireImportFails(plugin + "/FV" + strings.Repeat("0", 32))

	// Destroy archives; archived and missing versions read as deleted.
	l.del(c.ID, c.Properties)
	require.Len(t, l.f.callsMatching("POST", flexPluginBase+"/Plugins/"+plugin+"/Versions/"+sid+"/Archive"), 1)
	require.Empty(t, l.read(c.ID, c.Properties, in).ID)
	l.f.remove("flex", testAccount, "Versions/"+sid)
	l.del(c.ID, c.Properties)

	require.Len(t, l.check(pv(map[string]property.Value{"pluginSid": sv("x"), "pluginVersion": sv(""), "pluginUrl": sv("")})).Failures, 3)
}

func flexMakeVersion(t *testing.T, l *tl, plugin, version string) string {
	c := l.other("FlexPluginVersion", "v-"+version).create(pv(map[string]property.Value{"pluginSid": sv(plugin), "pluginVersion": sv(version),
		"pluginUrl": sv("https://example.com/" + version + ".js")}))
	return strings.TrimPrefix(c.ID, plugin+"/")
}

func TestFlexPluginConfigurationLC_Lifecycle(t *testing.T) {
	l := newTL(t, "FlexPluginConfiguration", "cfg")
	plugin := flexMakePlugin(t, l, "cfg-plugin")
	v1, v2 := flexMakeVersion(t, l, plugin, "1.0.0"), flexMakeVersion(t, l, plugin, "2.0.0")
	in := pv(map[string]property.Value{"name": sv("prod"), "description": sv("production"), "pluginVersionSids": arrv(sv(v1), sv(v2))})
	c := l.create(in)
	require.True(t, strings.HasPrefix(c.ID, "FJ"), c.ID)
	form := l.f.callsMatching("POST", flexPluginBase+"/Configurations")[0].Form
	require.Equal(t, "prod", form.Get("Name"))
	require.Equal(t, []string{`{"plugin_version":"` + v1 + `"}`, `{"plugin_version":"` + v2 + `"}`}, form["Plugins"])
	requireNoChanges(t, l.diff(c.ID, c.Properties, in, in))
	rd := l.read(c.ID, c.Properties, in)
	require.Equal(t, 2, rd.Inputs.Get("pluginVersionSids").AsArray().Len())
	requireNoChanges(t, l.diff(c.ID, rd.Properties, in, rd.Inputs))

	// Everything replaces, and the order of the versions counts.
	for k, v := range map[string]property.Value{"name": sv("staging"), "description": sv("x"), "pluginVersionSids": arrv(sv(v2), sv(v1))} {
		d := l.diff(c.ID, c.Properties, with(in, map[string]property.Value{k: v}), in)
		require.Equal(t, pgp.UpdateReplace, d.DetailedDiff[k].Kind, k)
	}
	requireNoChanges(t, l.diff(c.ID, c.Properties, without(in, "description", "pluginVersionSids"), in))

	// Drift in the plugin list shows.
	l.f.mutate("flex", testAccount, "Configurations/"+c.ID, func(o map[string]any) { o["_plugins"] = o["_plugins"].([]any)[:1] })
	rd = l.read(c.ID, c.Properties, in)
	require.Equal(t, 1, rd.Inputs.Get("pluginVersionSids").AsArray().Len())
	require.Equal(t, pgp.UpdateReplace, l.diff(c.ID, rd.Properties, in, rd.Inputs).DetailedDiff["pluginVersionSids"].Kind)

	imp := l.read(c.ID, empty(), empty())
	require.Empty(t, l.check(imp.Inputs).Failures)
	require.Equal(t, "prod", getS(t, imp.Inputs, "name"))
	require.Equal(t, "production", getS(t, imp.Inputs, "description"))
	l.requireImportFails("FJ" + strings.Repeat("0", 32))

	// Destroy archives; one Twilio refuses to archive (the live Configuration) is an error.
	rel := l.other("FlexPluginRelease", "rel").create(pv(map[string]property.Value{"configurationId": sv(c.ID)}))
	err := l.s.Delete(pgp.DeleteRequest{ID: c.ID, Urn: l.urn, Properties: c.Properties})
	require.ErrorContains(t, err, "in use by a release")
	l.f.remove("flex", testAccount, "Releases/"+rel.ID)
	l.del(c.ID, c.Properties)
	require.Empty(t, l.read(c.ID, c.Properties, in).ID)
	l.f.remove("flex", testAccount, "Configurations/"+c.ID)
	l.del(c.ID, c.Properties)

	bad := l.check(pv(map[string]property.Value{"name": sv(""), "pluginVersionSids": arrv(sv("FV1"))}))
	require.Len(t, bad.Failures, 2)
	require.Empty(t, l.check(pv(map[string]property.Value{"name": sv("n"), "pluginVersionSids": computed()})).Failures)
	require.Empty(t, l.check(pv(map[string]property.Value{"name": sv("n")})).Failures)
}

func TestFlexPluginReleaseLC_Lifecycle(t *testing.T) {
	l := newTL(t, "FlexPluginRelease", "rel")
	cfg := l.other("FlexPluginConfiguration", "c").create(pv(map[string]property.Value{"name": sv("c")})).ID
	in := pv(map[string]property.Value{"configurationId": sv(cfg)})
	c := l.create(in)
	require.True(t, strings.HasPrefix(c.ID, "FK"), c.ID)
	require.Equal(t, cfg, getS(t, c.Properties, "configurationSid"))
	require.Equal(t, cfg, l.f.callsMatching("POST", flexPluginBase+"/Releases")[0].Form.Get("ConfigurationId"))
	requireNoChanges(t, l.diff(c.ID, c.Properties, in, in))
	rd := l.read(c.ID, c.Properties, in)
	requireNoChanges(t, l.diff(c.ID, rd.Properties, in, rd.Inputs))

	// A different Configuration is a new Release.
	other := l.other("FlexPluginConfiguration", "c2").create(pv(map[string]property.Value{"name": sv("c2")})).ID
	d := l.diff(c.ID, c.Properties, pv(map[string]property.Value{"configurationId": sv(other)}), in)
	require.Equal(t, pgp.UpdateReplace, d.DetailedDiff["configurationId"].Kind)

	imp := l.read(c.ID, empty(), empty())
	require.Equal(t, cfg, getS(t, imp.Inputs, "configurationId"))
	l.requireImportFails("FK" + strings.Repeat("0", 32))

	// Destroy only forgets the Release.
	l.f.resetCalls()
	l.del(c.ID, c.Properties)
	require.Empty(t, l.f.callsMatching("POST", flexPluginBase+"/Releases/"+c.ID))
	require.NotNil(t, l.f.get("flex", testAccount, "Releases/"+c.ID))
	l.f.remove("flex", testAccount, "Releases/"+c.ID)
	require.Empty(t, l.read(c.ID, c.Properties, in).ID)

	require.Len(t, l.check(pv(map[string]property.Value{"configurationId": sv("")})).Failures, 1)
}

func TestLookup_FlexPlugins(t *testing.T) {
	l := newTL(t, "FlexPlugin", "p")
	p1, p2, p3 := flexMakePlugin(t, l, "one"), flexMakePlugin(t, l, "two"), flexMakePlugin(t, l, "three")
	v1 := flexMakeVersion(t, l, p1, "1.0.0")
	flexMakeVersion(t, l, p1, "1.1.0")
	flexMakeVersion(t, l, p2, "9.9.9")
	cfg := l.other("FlexPluginConfiguration", "c").create(pv(map[string]property.Value{"name": sv("c"), "description": sv("d"), "pluginVersionSids": arrv(sv(v1))})).ID
	rel := l.other("FlexPluginRelease", "r").create(pv(map[string]property.Value{"configurationId": sv(cfg)})).ID
	_ = p3

	r, err := invoke(t, l, "getFlexPlugin", map[string]property.Value{"sid": sv(p1)})
	require.NoError(t, err)
	require.Equal(t, "one", getS(t, r, "uniqueName"))
	require.False(t, r.Get("archived").AsBool())
	require.False(t, hasProp(r, "friendlyName"))
	_, err = invoke(t, l, "getFlexPlugin", map[string]property.Value{"sid": sv("FPmissing")})
	require.ErrorContains(t, err, "was not found")
	_, err = invoke(t, l, "getFlexPlugin", map[string]property.Value{"sid": sv("")})
	require.ErrorContains(t, err, "must be set")
	r, err = invoke(t, l, "getFlexPlugins", map[string]property.Value{})
	require.NoError(t, err)
	require.Len(t, r.Get("flexPlugins").AsArray().AsSlice(), 3)

	r, err = invoke(t, l, "getFlexPluginVersion", map[string]property.Value{"pluginSid": sv(p1), "sid": sv(v1)})
	require.NoError(t, err)
	require.Equal(t, "1.0.0", getS(t, r, "pluginVersion"))
	require.Equal(t, p1, getS(t, r, "pluginSid"))
	require.False(t, r.Get("private").AsBool())
	_, err = invoke(t, l, "getFlexPluginVersion", map[string]property.Value{"pluginSid": sv(p1), "sid": sv("")})
	require.ErrorContains(t, err, "must be set")
	r, err = invoke(t, l, "getFlexPluginVersions", map[string]property.Value{"pluginSid": sv(p1)})
	require.NoError(t, err)
	require.Len(t, r.Get("flexPluginVersions").AsArray().AsSlice(), 2)

	r, err = invoke(t, l, "getFlexPluginConfiguration", map[string]property.Value{"sid": sv(cfg)})
	require.NoError(t, err)
	require.Equal(t, "d", getS(t, r, "description"))
	require.Equal(t, v1, r.Get("pluginVersionSids").AsArray().Get(0).AsString())
	r, err = invoke(t, l, "getFlexPluginConfigurations", map[string]property.Value{})
	require.NoError(t, err)
	require.Len(t, r.Get("flexPluginConfigurations").AsArray().AsSlice(), 1)

	r, err = invoke(t, l, "getFlexPluginRelease", map[string]property.Value{"sid": sv(rel)})
	require.NoError(t, err)
	require.Equal(t, cfg, getS(t, r, "configurationSid"))
	r, err = invoke(t, l, "getFlexPluginReleases", map[string]property.Value{})
	require.NoError(t, err)
	require.Len(t, r.Get("flexPluginReleases").AsArray().AsSlice(), 1)
}
