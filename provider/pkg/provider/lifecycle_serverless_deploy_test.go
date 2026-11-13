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
	"encoding/base64"
	"fmt"
	"strings"
	"testing"
	"time"

	pgp "github.com/pulumi/pulumi-go-provider"
	"github.com/pulumi/pulumi/sdk/v3/go/property"
	"github.com/stretchr/testify/require"
)

func serverlessFn(l *tl, svc string) string {
	c := l.other("ServerlessFunction", "fn").create(pv(map[string]property.Value{"serviceSid": sv(svc), "friendlyName": sv("hello")}))
	return getS(l.t, c.Properties, "sid")
}

func serverlessAsset(l *tl, svc string) string {
	c := l.other("ServerlessAsset", "as").create(pv(map[string]property.Value{"serviceSid": sv(svc), "friendlyName": sv("page")}))
	return getS(l.t, c.Properties, "sid")
}

func TestServerlessFunctionVersionLC_Lifecycle(t *testing.T) {
	l := newTL(t, "ServerlessFunctionVersion", "v")
	svc := serverlessSvc(l)
	fn := serverlessFn(l, svc)
	in := pv(map[string]property.Value{"serviceSid": sv(svc), "functionSid": sv(fn), "path": sv("/hello"), "visibility": sv("public"),
		"content": sv("exports.handler = (c, e, cb) => cb(null, 'hi');"), "contentType": sv("application/javascript")})
	c := l.create(in)
	sid := getS(t, c.Properties, "sid")
	require.True(t, strings.HasPrefix(sid, "ZN"), sid)
	require.Equal(t, svc+"/"+fn+"/"+sid, c.ID)
	require.Equal(t, "2026-11-06T00:00:00Z", getS(t, c.Properties, "dateCreated"))

	// The code goes to the upload host as a multipart file named after the path.
	require.Len(t, l.f.callsMatching("POST", "/serverless-upload/v1/Services/"+svc+"/Functions/"+fn+"/Versions"), 1)
	up := l.f.uploads[sid]
	require.Equal(t, "hello", up.Filename)
	require.Equal(t, "application/javascript", up.ContentType)
	require.Equal(t, "/hello", up.Path)
	require.Equal(t, "public", up.Visibility)
	require.Equal(t, "exports.handler = (c, e, cb) => cb(null, 'hi');", string(up.Content))
	requireNoChanges(t, l.diff(c.ID, c.Properties, in, in))

	// A refresh changes nothing and keeps the code.
	rd := l.read(c.ID, c.Properties, in)
	requireNoChanges(t, l.diff(c.ID, rd.Properties, in, rd.Inputs))
	require.Equal(t, "/hello", getS(t, rd.Inputs, "path"))

	// Every input replaces: versions are immutable.
	for prop, v := range map[string]property.Value{
		"content": sv("// v2"), "path": sv("/hello2"), "visibility": sv("protected"),
		"functionSid": sv("ZH" + strings.Repeat("0", 32)), "serviceSid": sv("ZS" + strings.Repeat("0", 32)), "contentType": sv("text/plain"),
	} {
		d := l.diff(c.ID, c.Properties, with(in, map[string]property.Value{prop: v}), in)
		require.Equal(t, pgp.UpdateReplace, d.DetailedDiff[prop].Kind, prop)
	}

	// An import adopts the version and reads the code back, so the program plans nothing.
	imp := l.read(c.ID, empty(), empty())
	require.Equal(t, svc, getS(t, imp.Inputs, "serviceSid"))
	require.Equal(t, fn, getS(t, imp.Inputs, "functionSid"))
	require.Equal(t, "/hello", getS(t, imp.Inputs, "path"))
	require.Equal(t, "exports.handler = (c, e, cb) => cb(null, 'hi');", getS(t, imp.Inputs, "content"))
	requireNoChanges(t, l.diff(c.ID, imp.Properties, in, imp.Inputs))
	_, err := l.s.Read(pgp.ReadRequest{ID: "nonsense", Urn: l.urn, Properties: empty(), Inputs: empty()})
	require.ErrorContains(t, err, "<serviceSid>/<functionSid>/<sid>")
	l.requireImportFails(svc + "/" + fn + "/ZN" + strings.Repeat("9", 32))

	for name, bad := range map[string]property.Map{
		"serviceSid":  with(in, map[string]property.Value{"serviceSid": sv("ZSshort")}),
		"functionSid": with(in, map[string]property.Value{"functionSid": sv("nope")}),
		"path":        with(in, map[string]property.Value{"path": sv("hello")}),
		"visibility":  with(in, map[string]property.Value{"visibility": sv("secret")}),
		"content":     with(in, map[string]property.Value{"content": sv("")}),
	} {
		chk := l.check(bad)
		require.Len(t, chk.Failures, 1, name)
		require.Equal(t, name, chk.Failures[0].Property)
	}

	// Deleted outside Pulumi: the refresh drops it. Destroy forgets and calls nothing.
	l.f.remove("zvf", testAccount, svc+"/"+fn+"/"+sid)
	require.Empty(t, l.read(c.ID, c.Properties, in).ID)
	l.f.resetCalls()
	l.del(c.ID, c.Properties)
	require.Empty(t, l.f.callsMatching("DELETE", "/"))
}

func TestServerlessFunctionVersionLC_UploadFailureAndMissingFunction(t *testing.T) {
	l := newTL(t, "ServerlessFunctionVersion", "v")
	svc := serverlessSvc(l)
	in := pv(map[string]property.Value{"serviceSid": sv(svc), "functionSid": sv("ZH" + strings.Repeat("1", 32)), "path": sv("/hello"), "visibility": sv("private"), "content": sv("x")})
	_, err := l.s.Create(pgp.CreateRequest{Urn: l.urn, Properties: l.checked(in)})
	require.ErrorContains(t, err, "upload version of function")
	require.ErrorContains(t, err, "was not found")
}

func TestServerlessAssetVersionLC_Lifecycle(t *testing.T) {
	l := newTL(t, "ServerlessAssetVersion", "v")
	svc := serverlessSvc(l)
	as := serverlessAsset(l, svc)
	in := pv(map[string]property.Value{"serviceSid": sv(svc), "assetSid": sv(as), "path": sv("/index.html"), "visibility": sv("public"),
		"content": sv("<h1>hi</h1>"), "contentType": sv("text/html")})
	c := l.create(in)
	sid := getS(t, c.Properties, "sid")
	require.Equal(t, svc+"/"+as+"/"+sid, c.ID)
	up := l.f.uploads[sid]
	require.Equal(t, "index.html", up.Filename)
	require.Equal(t, "text/html", up.ContentType)
	require.Equal(t, "<h1>hi</h1>", string(up.Content))
	requireNoChanges(t, l.diff(c.ID, c.Properties, in, in))

	for prop, v := range map[string]property.Value{"content": sv("<h1>yo</h1>"), "path": sv("/b.html"), "visibility": sv("private"), "contentType": sv("text/plain")} {
		d := l.diff(c.ID, c.Properties, with(in, map[string]property.Value{prop: v}), in)
		require.Equal(t, pgp.UpdateReplace, d.DetailedDiff[prop].Kind, prop)
	}

	// Import: Twilio does not return asset bytes, so a program that sets content plans nothing.
	imp := l.read(c.ID, empty(), empty())
	require.Equal(t, as, getS(t, imp.Inputs, "assetSid"))
	require.False(t, hasProp(imp.Inputs, "content"))
	requireNoChanges(t, l.diff(c.ID, imp.Properties, in, imp.Inputs))
	// The imported inputs (neither content nor contentBase64) pass Check, so `pulumi import`
	// no longer warns (#1879); a new version with no bytes is refused at create.
	require.Empty(t, l.check(imp.Inputs).Failures)
	_, cerr := l.s.Create(pgp.CreateRequest{Urn: l.urn, Properties: imp.Inputs})
	require.ErrorContains(t, cerr, "exactly one")
	l.requireImportFails(svc + "/" + as + "/ZN" + strings.Repeat("9", 32))

	// Binary assets go in as base64.
	raw := []byte{0x89, 'P', 'N', 'G', 0, 1, 2}
	bin := pv(map[string]property.Value{"serviceSid": sv(svc), "assetSid": sv(as), "path": sv("/logo.png"), "visibility": sv("public"),
		"contentBase64": sv(base64.StdEncoding.EncodeToString(raw)), "contentType": sv("image/png")})
	b := l.other("ServerlessAssetVersion", "bin").create(bin)
	require.Equal(t, raw, l.f.uploads[getS(t, b.Properties, "sid")].Content)

	// The base64 must decode.
	badB64 := l.check(with(bin, map[string]property.Value{"contentBase64": sv("!!!")}))
	require.Len(t, badB64.Failures, 1)
	require.Equal(t, "contentBase64", badB64.Failures[0].Property)
	// Setting both content and contentBase64 still fails.
	both := l.check(with(in, map[string]property.Value{"contentBase64": sv("aGk=")}))
	require.Len(t, both.Failures, 1)

	l.f.remove("zva", testAccount, svc+"/"+as+"/"+sid)
	require.Empty(t, l.read(c.ID, c.Properties, in).ID)
	l.del(c.ID, c.Properties)
}

func serverlessVersionSid(l *tl, svc, fn string, n string) string {
	in := pv(map[string]property.Value{"serviceSid": sv(svc), "functionSid": sv(fn), "path": sv("/" + n), "visibility": sv("public"), "content": sv("// " + n)})
	return getS(l.t, l.other("ServerlessFunctionVersion", n).create(in).Properties, "sid")
}

func TestServerlessBuildLC_Lifecycle(t *testing.T) {
	buildPollInterval = time.Millisecond
	l := newTL(t, "ServerlessBuild", "b")
	l.f.buildBuildingPolls = 2
	svc := serverlessSvc(l)
	fn := serverlessFn(l, svc)
	v1 := serverlessVersionSid(l, svc, fn, "one")
	in := pv(map[string]property.Value{"serviceSid": sv(svc), "functionVersionSids": strList(v1), "runtime": sv("node20"),
		"dependencies": property.New(property.NewArray([]property.Value{property.New(property.NewMap(map[string]property.Value{"name": sv("twilio"), "version": sv("5.3.0")}))}))})
	c := l.create(in)
	sid := getS(t, c.Properties, "sid")
	require.True(t, strings.HasPrefix(sid, "ZB"), sid)
	require.Equal(t, svc+"/"+sid, c.ID)
	require.Equal(t, "completed", getS(t, c.Properties, "status"))
	require.Equal(t, 3, l.f.buildPolls[sid], "polled until the build completed")

	form := l.f.callsMatching("POST", "/serverless/v1/Services/"+svc+"/Builds")[0].Form
	require.Equal(t, []string{v1}, form["FunctionVersions"])
	require.Empty(t, form["AssetVersions"])
	require.Equal(t, "node20", form.Get("Runtime"))
	require.JSONEq(t, `[{"name":"twilio","version":"5.3.0"}]`, form.Get("Dependencies"))
	requireNoChanges(t, l.diff(c.ID, c.Properties, in, in))

	rd := l.read(c.ID, c.Properties, in)
	require.Equal(t, "completed", getS(t, rd.Properties, "status"))
	requireNoChanges(t, l.diff(c.ID, rd.Properties, in, rd.Inputs))

	// Builds are immutable: every input replaces; an unset runtime keeps what Twilio chose.
	v2 := serverlessVersionSid(l, svc, fn, "two")
	for prop, v := range map[string]property.Value{"functionVersionSids": strList(v1, v2), "assetVersionSids": strList(v2), "runtime": sv("node22"),
		"serviceSid": sv("ZS" + strings.Repeat("0", 32)), "dependencies": property.New(property.NewArray(nil))} {
		d := l.diff(c.ID, c.Properties, with(in, map[string]property.Value{prop: v}), in)
		require.Equal(t, pgp.UpdateReplace, d.DetailedDiff[prop].Kind, prop)
	}
	requireNoChanges(t, l.diff(c.ID, c.Properties, without(in, "runtime"), in))

	// Import adopts the versions, runtime and dependencies.
	imp := l.read(c.ID, empty(), empty())
	require.Equal(t, svc, getS(t, imp.Inputs, "serviceSid"))
	require.Equal(t, "node20", getS(t, imp.Inputs, "runtime"))
	require.Equal(t, v1, imp.Inputs.Get("functionVersionSids").AsArray().Get(0).AsString())
	require.Equal(t, "twilio", imp.Inputs.Get("dependencies").AsArray().Get(0).AsMap().Get("name").AsString())
	requireNoChanges(t, l.diff(c.ID, imp.Properties, in, imp.Inputs))
	_, err := l.s.Read(pgp.ReadRequest{ID: "nonsense", Urn: l.urn, Properties: empty(), Inputs: empty()})
	require.ErrorContains(t, err, "<serviceSid>/<sid>")
	l.requireImportFails(svc + "/ZB" + strings.Repeat("9", 32))

	for name, bad := range map[string]property.Map{
		"serviceSid":          with(in, map[string]property.Value{"serviceSid": sv("nope")}),
		"functionVersionSids": with(in, map[string]property.Value{"functionVersionSids": strList("ZH" + strings.Repeat("0", 32))}),
		"runtime":             with(in, map[string]property.Value{"runtime": sv("python")}),
		"dependencies":        with(in, map[string]property.Value{"dependencies": property.New(property.NewArray([]property.Value{property.New(property.NewMap(map[string]property.Value{"name": sv("x"), "version": sv("")}))}))}),
	} {
		chk := l.check(bad)
		require.Len(t, chk.Failures, 1, name)
		require.Equal(t, name, chk.Failures[0].Property)
	}

	// Delete removes the build; one deleted outside Pulumi is gone on refresh and delete is a no-op.
	l.del(c.ID, c.Properties)
	require.Nil(t, l.f.get("zb", testAccount, svc+"/"+sid))
	require.Empty(t, l.read(c.ID, c.Properties, in).ID)
	l.del(c.ID, c.Properties)
}

func TestServerlessBuildLC_FailedBuildIsAnErrorAndLeavesNothing(t *testing.T) {
	buildPollInterval = time.Millisecond
	l := newTL(t, "ServerlessBuild", "b")
	l.f.buildFinalStatus = "failed"
	svc := serverlessSvc(l)
	fn := serverlessFn(l, svc)
	v1 := serverlessVersionSid(l, svc, fn, "one")
	in := pv(map[string]property.Value{"serviceSid": sv(svc), "functionVersionSids": strList(v1)})
	_, err := l.s.Create(pgp.CreateRequest{Urn: l.urn, Properties: l.checked(in)})
	require.ErrorContains(t, err, "failed")
	require.Len(t, l.f.callsMatching("DELETE", "/Builds/"), 1, "the failed build is removed")
}

// A failing read-back after a completed build must not orphan the build: Create still returns its ID.
func TestServerlessBuildLC_ReadBackFailureKeepsID(t *testing.T) {
	buildPollInterval = time.Millisecond
	l := newTL(t, "ServerlessBuild", "b")
	svc := serverlessSvc(l)
	l.f.buildReadFails = true
	c := l.create(pv(map[string]property.Value{"serviceSid": sv(svc)}))
	require.Equal(t, "completed", getS(t, c.Properties, "status"))
	require.True(t, strings.HasPrefix(c.ID, svc+"/ZB"), c.ID)
	require.Equal(t, c.ID, svc+"/"+getS(t, c.Properties, "sid"))
}

func TestServerlessBuildLC_PollTimeout(t *testing.T) {
	buildPollInterval = time.Millisecond
	old := buildPollTimeout
	buildPollTimeout = 30 * time.Millisecond
	t.Cleanup(func() { buildPollTimeout = old })
	l := newTL(t, "ServerlessBuild", "b")
	l.f.buildBuildingPolls = 1 << 30
	svc := serverlessSvc(l)
	_, err := l.s.Create(pgp.CreateRequest{Urn: l.urn, Properties: l.checked(pv(map[string]property.Value{"serviceSid": sv(svc)}))})
	require.ErrorContains(t, err, "did not finish")
}

func TestServerlessDeploymentLC_Lifecycle(t *testing.T) {
	buildPollInterval = time.Millisecond
	l := newTL(t, "ServerlessDeployment", "d")
	svc := serverlessSvc(l)
	env := serverlessEnv(l, svc)
	build := getS(t, l.other("ServerlessBuild", "b").create(pv(map[string]property.Value{"serviceSid": sv(svc)})).Properties, "sid")
	in := pv(map[string]property.Value{"serviceSid": sv(svc), "environmentSid": sv(env), "buildSid": sv(build)})
	c := l.create(in)
	sid := getS(t, c.Properties, "sid")
	require.True(t, strings.HasPrefix(sid, "ZD"), sid)
	require.Equal(t, svc+"/"+env+"/"+sid, c.ID)
	require.Equal(t, build, l.f.callsMatching("POST", "/Deployments")[0].Form.Get("BuildSid"))
	require.Empty(t, l.f.callsMatching("POST", "/Deployments")[0].Form["IsPlugin"], "an unset flag is not sent")
	require.Equal(t, build, l.f.get("ze", testAccount, svc+"/"+env)["build_sid"])
	requireNoChanges(t, l.diff(c.ID, c.Properties, in, in))

	rd := l.read(c.ID, c.Properties, in)
	requireNoChanges(t, l.diff(c.ID, rd.Properties, in, rd.Inputs))

	other := getS(t, l.other("ServerlessBuild", "b2").create(pv(map[string]property.Value{"serviceSid": sv(svc)})).Properties, "sid")
	for prop, v := range map[string]property.Value{"buildSid": sv(other), "isPlugin": bv(true),
		"environmentSid": sv("ZE" + strings.Repeat("0", 32)), "serviceSid": sv("ZS" + strings.Repeat("0", 32))} {
		d := l.diff(c.ID, c.Properties, with(in, map[string]property.Value{prop: v}), in)
		require.Equal(t, pgp.UpdateReplace, d.DetailedDiff[prop].Kind, prop)
	}

	imp := l.read(c.ID, empty(), empty())
	require.Equal(t, env, getS(t, imp.Inputs, "environmentSid"))
	require.Equal(t, build, getS(t, imp.Inputs, "buildSid"))
	requireNoChanges(t, l.diff(c.ID, imp.Properties, in, imp.Inputs))
	_, err := l.s.Read(pgp.ReadRequest{ID: "nonsense", Urn: l.urn, Properties: empty(), Inputs: empty()})
	require.ErrorContains(t, err, "<serviceSid>/<environmentSid>/<sid>")
	l.requireImportFails(svc + "/" + env + "/ZD" + strings.Repeat("9", 32))

	for name, bad := range map[string]property.Map{
		"serviceSid":     with(in, map[string]property.Value{"serviceSid": sv("x")}),
		"environmentSid": with(in, map[string]property.Value{"environmentSid": sv("x")}),
		"buildSid":       with(in, map[string]property.Value{"buildSid": sv("ZH" + strings.Repeat("0", 32))}),
	} {
		chk := l.check(bad)
		require.Len(t, chk.Failures, 1, name)
		require.Equal(t, name, chk.Failures[0].Property)
	}

	// A deployment that Twilio no longer has drops from state; destroy forgets and calls nothing.
	l.f.remove("zd", testAccount, svc+"/"+env+"/"+sid)
	require.Empty(t, l.read(c.ID, c.Properties, in).ID)
	l.f.resetCalls()
	l.del(c.ID, c.Properties)
	require.Empty(t, l.f.callsMatching("DELETE", "/"))

	// Deploying a build that does not exist is an error naming the build.
	_, err = l.s.Create(pgp.CreateRequest{Urn: l.urn, Properties: l.checked(with(in, map[string]property.Value{"buildSid": sv("ZB" + strings.Repeat("7", 32))}))})
	require.ErrorContains(t, err, "deploy build")
}

func TestLookup_ServerlessDeployPlane(t *testing.T) {
	buildPollInterval = time.Millisecond
	l := newTL(t, "ServerlessService", "svc")
	svc := serverlessSvc(l)
	env := serverlessEnv(l, svc)
	fn := serverlessFn(l, svc)
	as := serverlessAsset(l, svc)
	v := l.other("ServerlessFunctionVersion", "v").create(pv(map[string]property.Value{"serviceSid": sv(svc), "functionSid": sv(fn), "path": sv("/hello"), "visibility": sv("protected"), "content": sv("//")}))
	av := l.other("ServerlessAssetVersion", "av").create(pv(map[string]property.Value{"serviceSid": sv(svc), "assetSid": sv(as), "path": sv("/a.txt"), "visibility": sv("public"), "content": sv("a")}))
	vsid, avsid := getS(t, v.Properties, "sid"), getS(t, av.Properties, "sid")
	b := l.other("ServerlessBuild", "b").create(pv(map[string]property.Value{"serviceSid": sv(svc), "functionVersionSids": strList(vsid), "assetVersionSids": strList(avsid)}))
	bsid := getS(t, b.Properties, "sid")
	d := l.other("ServerlessDeployment", "d").create(pv(map[string]property.Value{"serviceSid": sv(svc), "environmentSid": sv(env), "buildSid": sv(bsid)}))
	dsid := getS(t, d.Properties, "sid")

	r, err := invoke(t, l, "getServerlessFunctionVersion", map[string]property.Value{"serviceSid": sv(svc), "functionSid": sv(fn), "versionSid": sv(vsid)})
	require.NoError(t, err)
	require.Equal(t, "/hello", getS(t, r, "path"))
	require.Equal(t, "protected", getS(t, r, "visibility"))
	_, err = invoke(t, l, "getServerlessFunctionVersion", map[string]property.Value{"serviceSid": sv(svc), "functionSid": sv(fn), "versionSid": sv("ZNmissing")})
	require.ErrorContains(t, err, "was not found")
	_, err = invoke(t, l, "getServerlessFunctionVersion", map[string]property.Value{"serviceSid": sv(svc), "functionSid": sv(fn), "versionSid": sv("")})
	require.ErrorContains(t, err, "versionSid must be set")

	r, err = invoke(t, l, "getServerlessAssetVersion", map[string]property.Value{"serviceSid": sv(svc), "assetSid": sv(as), "versionSid": sv(avsid)})
	require.NoError(t, err)
	require.Equal(t, "/a.txt", getS(t, r, "path"))
	_, err = invoke(t, l, "getServerlessAssetVersion", map[string]property.Value{"serviceSid": sv(svc), "assetSid": sv(as), "versionSid": sv("ZNmissing")})
	require.ErrorContains(t, err, "was not found")

	r, err = invoke(t, l, "getServerlessBuild", map[string]property.Value{"serviceSid": sv(svc), "buildSid": sv(bsid)})
	require.NoError(t, err)
	require.Equal(t, "completed", getS(t, r, "status"))
	require.Equal(t, vsid, r.Get("functionVersionSids").AsArray().Get(0).AsString())
	require.Equal(t, avsid, r.Get("assetVersionSids").AsArray().Get(0).AsString())
	require.Equal(t, 0, r.Get("dependencies").AsArray().Len())
	_, err = invoke(t, l, "getServerlessBuild", map[string]property.Value{"serviceSid": sv(svc), "buildSid": sv("ZBmissing")})
	require.ErrorContains(t, err, "was not found")

	r, err = invoke(t, l, "getServerlessDeployment", map[string]property.Value{"serviceSid": sv(svc), "environmentSid": sv(env), "deploymentSid": sv(dsid)})
	require.NoError(t, err)
	require.Equal(t, bsid, getS(t, r, "buildSid"))
	_, err = invoke(t, l, "getServerlessDeployment", map[string]property.Value{"serviceSid": sv(svc), "environmentSid": sv(env), "deploymentSid": sv("ZDmissing")})
	require.ErrorContains(t, err, "was not found")
}

// Twilio refuses to delete a Build that a Deployment points at (400/20001) and a Function or Asset
// whose version a Build holds (400/20001), and has no delete for deployments or versions, so a
// destroy of a deployed Function must not fail on them (#1856). The fake models both refusals.
func TestServerlessDeployPlane_DestroyEverything(t *testing.T) {
	buildPollInterval = time.Millisecond
	l := newTL(t, "ServerlessService", "svc")
	svcC := l.create(serverlessServiceInputs())
	svc := svcC.ID
	envC := l.other("ServerlessEnvironment", "env").create(pv(map[string]property.Value{"serviceSid": sv(svc), "uniqueName": sv("stage")}))
	env := getS(t, envC.Properties, "sid")
	fnC := l.other("ServerlessFunction", "fn").create(pv(map[string]property.Value{"serviceSid": sv(svc), "friendlyName": sv("hello")}))
	asC := l.other("ServerlessAsset", "as").create(pv(map[string]property.Value{"serviceSid": sv(svc), "friendlyName": sv("page")}))
	fvC := l.other("ServerlessFunctionVersion", "fv").create(pv(map[string]property.Value{"serviceSid": sv(svc), "functionSid": sv(fnC.ID[len(svc)+1:]),
		"path": sv("/hello"), "visibility": sv("public"), "content": sv("// hi")}))
	avC := l.other("ServerlessAssetVersion", "av").create(pv(map[string]property.Value{"serviceSid": sv(svc), "assetSid": sv(asC.ID[len(svc)+1:]),
		"path": sv("/a.txt"), "visibility": sv("public"), "content": sv("hi")}))
	bC := l.other("ServerlessBuild", "b").create(pv(map[string]property.Value{"serviceSid": sv(svc),
		"functionVersionSids": strList(getS(t, fvC.Properties, "sid")), "assetVersionSids": strList(getS(t, avC.Properties, "sid"))}))
	dC := l.other("ServerlessDeployment", "d").create(pv(map[string]property.Value{"serviceSid": sv(svc), "environmentSid": sv(env), "buildSid": sv(getS(t, bC.Properties, "sid"))}))

	// Destroy in dependency order: every step succeeds although Twilio refuses some deletes.
	l.other("ServerlessDeployment", "d").del(dC.ID, dC.Properties)
	l.other("ServerlessBuild", "b").del(bC.ID, bC.Properties)
	require.NotNil(t, l.f.get("zb", testAccount, bC.ID), "the deployed build stays on Twilio until its service goes")
	l.other("ServerlessAssetVersion", "av").del(avC.ID, avC.Properties)
	l.other("ServerlessFunctionVersion", "fv").del(fvC.ID, fvC.Properties)
	l.other("ServerlessAsset", "as").del(asC.ID, asC.Properties)
	l.other("ServerlessFunction", "fn").del(fnC.ID, fnC.Properties)
	l.other("ServerlessEnvironment", "env").del(envC.ID, envC.Properties)
	l.del(svcC.ID, svcC.Properties)
	require.Nil(t, l.f.get("zb", testAccount, bC.ID), "deleting the service removes what was left")

	// Without a deployment the build is deleted for real, and an unrelated 400 is still an error.
	l2 := newTL(t, "ServerlessService", "svc2")
	svc2 := l2.create(serverlessServiceInputs()).ID
	b2 := l2.other("ServerlessBuild", "b2").create(pv(map[string]property.Value{"serviceSid": sv(svc2)}))
	l2.other("ServerlessBuild", "b2").del(b2.ID, b2.Properties)
	require.Nil(t, l2.f.get("zb", testAccount, b2.ID))
}

// A 400/20001 that is not the deployment refusal is a real failure: delete must return it.
func TestServerlessBuildLC_DeleteUnrelated400IsAnError(t *testing.T) {
	buildPollInterval = time.Millisecond
	l := newTL(t, "ServerlessBuild", "b")
	svc := serverlessSvc(l)
	c := l.create(pv(map[string]property.Value{"serviceSid": sv(svc)}))
	l.f.buildDeleteUnrelated = true
	err := l.s.Delete(pgp.DeleteRequest{ID: c.ID, Urn: l.urn, Properties: c.Properties})
	require.ErrorContains(t, err, "Something else is wrong")
}

func TestIsRefusal(t *testing.T) {
	e := &apiError{Status: 400, Code: 20001, Message: "Cannot delete Build because it's part of an active Deployment."}
	require.True(t, isRefusal(e, "deployment"))
	require.False(t, isRefusal(e, "referenced in a build"), "another 20001 is a real failure")
	require.False(t, isRefusal(&apiError{Status: 400, Code: 20002, Message: e.Message}, "deployment"))
	require.False(t, isRefusal(&apiError{Status: 500, Code: 20001, Message: e.Message}, "deployment"))
	require.False(t, isRefusal(fmt.Errorf("x"), "deployment"))
}

// A known list that holds a computed element (functionVersionSids=[fnVersion.sid]) decodes that
// element to "" at preview: Check must skip it and still reject a known malformed one (#1855).
func TestServerlessBuildLC_CheckUnknownListElement(t *testing.T) {
	l := newTL(t, "ServerlessBuild", "b")
	svc := "ZS" + strings.Repeat("a", 32)
	good := "ZN" + strings.Repeat("b", 32)
	chk := l.check(pv(map[string]property.Value{"serviceSid": sv(svc),
		"functionVersionSids": arrv(computed()), "assetVersionSids": arrv(computed(), sv(good))}))
	require.Empty(t, chk.Failures, "a computed element must not be validated")
	chk = l.check(pv(map[string]property.Value{"serviceSid": sv(svc),
		"functionVersionSids": arrv(computed(), sv("not-a-sid"))}))
	require.Len(t, chk.Failures, 1)
	require.Equal(t, "functionVersionSids", chk.Failures[0].Property)
	chk = l.check(pv(map[string]property.Value{"serviceSid": sv(svc),
		"functionVersionSids": computed(), "dependencies": arrv(objv(map[string]property.Value{"name": computed(), "version": sv("1.0.0")}))}))
	require.Empty(t, chk.Failures)
}

func TestConnectAppLC_CheckUnknownPermissionElement(t *testing.T) {
	l := newTL(t, "ConnectApp", "ca")
	chk := l.check(pv(map[string]property.Value{"connectAppSid": sv("CN" + strings.Repeat("a", 32)), "permissions": arrv(computed())}))
	require.Empty(t, chk.Failures)
}

// An imported build lists the versions as objects and carries Twilio's default
// dependencies; importing and then running the original program must plan no
// change (#1878).
func TestServerlessBuildLC_ImportThenDiffIsClean(t *testing.T) {
	buildPollInterval = time.Millisecond
	l := newTL(t, "ServerlessService", "svc")
	svc := serverlessSvc(l)
	fn := serverlessFn(l, svc)
	v := l.other("ServerlessFunctionVersion", "v").create(pv(map[string]property.Value{"serviceSid": sv(svc), "functionSid": sv(fn), "path": sv("/hello"), "visibility": sv("protected"), "content": sv("//")}))
	vsid := getS(t, v.Properties, "sid")
	bl := l.other("ServerlessBuild", "b")
	in := pv(map[string]property.Value{"serviceSid": sv(svc), "functionVersionSids": strList(vsid),
		"dependencies": property.New(property.NewArray([]property.Value{
			property.New(property.NewMap(map[string]property.Value{"name": sv("twilio"), "version": sv("5.3.0")})),
		}))})
	b := bl.create(in)
	bsid := getS(t, b.Properties, "sid")

	// Twilio adds its defaults, and lists them in its own order.
	o := l.f.get("zb", testAccount, svc+"/"+bsid)
	o["dependencies"] = []any{
		map[string]any{"name": "util", "version": "0.12.5"},
		map[string]any{"name": "twilio", "version": "5.3.0"},
		map[string]any{"name": "xmldom", "version": "0.6.0"},
		map[string]any{"name": "lodash", "version": "4.17.21"},
		map[string]any{"name": "@twilio/runtime-handler", "version": "2.1.2"},
	}

	imp := bl.read(svc+"/"+bsid, empty(), empty())
	require.Equal(t, vsid, imp.Inputs.Get("functionVersionSids").AsArray().Get(0).AsString(), "the version SID is read from the object")
	requireNoChanges(t, bl.diff(imp.ID, imp.Properties, bl.checked(in), imp.Inputs))

	// A real change to a listed dependency still plans a replacement.
	changed := with(in, map[string]property.Value{"dependencies": property.New(property.NewArray([]property.Value{
		property.New(property.NewMap(map[string]property.Value{"name": sv("twilio"), "version": sv("5.4.0")})),
	}))})
	require.True(t, bl.diff(imp.ID, imp.Properties, bl.checked(changed), imp.Inputs).HasChanges)
}
