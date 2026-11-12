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
	"bytes"
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"mime/multipart"
	"net/textproto"
	"net/url"
	"path"
	"regexp"
	"strings"
	"time"

	p "github.com/pulumi/pulumi-go-provider"
	"github.com/pulumi/pulumi-go-provider/infer"
)

// The deploy plane of a Serverless service: code versions, builds and deployments.
//
// Versions are the one place Twilio takes a file: they are uploaded as a multipart POST to
// serverless-upload.twilio.com, which the OpenAPI document does not describe (the Twilio CLI and the
// Serverless Toolkit use the same endpoint). A Build is then assembled from version SIDs and a
// Deployment points an environment at a finished Build. Everything here is immutable on Twilio's side,
// so every input replaces, and the API has no delete for versions or deployments: destroying those
// resources only forgets them.

var (
	serverlessVersionSidRE    = regexp.MustCompile(`^ZN[0-9a-fA-F]{32}$`)
	serverlessBuildSidRE      = regexp.MustCompile(`^ZB[0-9a-fA-F]{32}$`)
	serverlessDeploymentSidRE = regexp.MustCompile(`^ZD[0-9a-fA-F]{32}$`)
	serverlessVisibilities    = []string{"public", "protected", "private"}
	serverlessRuntimes        = []string{"node8", "node10", "node12", "node14", "node16", "node18", "node20", "node22", "node24"}
)

const serverlessUploadProduct = "serverless-upload"

// buildPollInterval and buildPollTimeout pace the wait for a Build to finish; tests shrink the interval.
var (
	buildPollInterval = 3 * time.Second
	buildPollTimeout  = 20 * time.Minute
)

func (c *checker) oneOf(prop, v string, allowed []string) {
	if c.unk(prop) {
		return
	}
	for _, a := range allowed {
		if v == a {
			return
		}
	}
	c.fail(prop, fmt.Sprintf("%s must be one of %s", prop, strings.Join(allowed, ", ")))
}

func (c *checker) pathProp(prop, v string) {
	if c.unk(prop) {
		return
	}
	if !strings.HasPrefix(v, "/") || len(v) > 255 {
		c.fail(prop, prop+" must start with / and be at most 255 characters, for example /hello")
	}
}

// uploadVersion posts one file to the upload host and returns the new Version resource.
func uploadVersion(ctx context.Context, c *Client, svc, coll, parent, vpath, visibility, contentType string, content []byte) (obj, error) {
	var buf bytes.Buffer
	mw := multipart.NewWriter(&buf)
	name := path.Base(vpath)
	if name == "" || name == "/" || name == "." {
		name = "file"
	}
	h := textproto.MIMEHeader{}
	h.Set("Content-Disposition", fmt.Sprintf(`form-data; name="Content"; filename=%q`, name))
	h.Set("Content-Type", contentType)
	part, err := mw.CreatePart(h)
	if err != nil {
		return nil, err
	}
	if _, err := part.Write(content); err != nil {
		return nil, err
	}
	if err := mw.WriteField("Path", vpath); err != nil {
		return nil, err
	}
	if err := mw.WriteField("Visibility", visibility); err != nil {
		return nil, err
	}
	if err := mw.Close(); err != nil {
		return nil, err
	}
	return c.send(ctx, "POST", serverlessUploadProduct, serverlessPath(svc, coll, parent, "Versions"), nil, buf.Bytes(), mw.FormDataContentType())
}

// ------------------------------------------------------------ ServerlessFunctionVersion

// ServerlessFunctionVersion uploads one immutable version of a Function: the JavaScript source of a Twilio Function. Every input replaces the version; destroying it only forgets it, because Twilio has no delete for versions.
type ServerlessFunctionVersion struct{}

// ServerlessFunctionVersionArgs are the inputs of a ServerlessFunctionVersion.
type ServerlessFunctionVersionArgs struct {
	ServiceSid  string  `pulumi:"serviceSid"`
	FunctionSid string  `pulumi:"functionSid"`
	Path        string  `pulumi:"path"`
	Visibility  string  `pulumi:"visibility"`
	Content     string  `pulumi:"content"`
	ContentType *string `pulumi:"contentType,optional"`
}

func (a *ServerlessFunctionVersionArgs) Annotate(an infer.Annotator) {
	an.Describe(&a.ServiceSid, "SID (ZS...) of the Serverless service. Changing it replaces the resource.")
	an.Describe(&a.FunctionSid, "SID (ZH...) of the Function (a ServerlessFunction) the version belongs to. Changing it replaces the resource.")
	an.Describe(&a.Path, "URL path the function answers on inside its service, starting with /, at most 255 characters, for example /hello. Changing it replaces the resource.")
	an.Describe(&a.Visibility, "Who may call the function: public (anyone), protected (requests signed by Twilio) or private (only other Functions of the service). Changing it replaces the resource.")
	an.Describe(&a.Content, "The source code of the function, for example the text of a .js file. Changing it uploads a new version and replaces the resource. After an import the code is read back from Twilio.")
	an.Describe(&a.ContentType, "Content type of the uploaded file. Defaults to application/javascript.")
}

// ServerlessFunctionVersionState is the persisted state of a ServerlessFunctionVersion.
type ServerlessFunctionVersionState struct {
	ServerlessFunctionVersionArgs

	// Sid is the version SID (ZN...).
	Sid         string `pulumi:"sid"`
	AccountSid  string `pulumi:"accountSid"`
	Url         string `pulumi:"url"`
	DateCreated string `pulumi:"dateCreated"`
}

func (s *ServerlessFunctionVersionState) Annotate(a infer.Annotator) {
	a.Describe(&s.Sid, "The version SID (ZN...). Pass it to ServerlessBuild.functionVersionSids.")
	a.Describe(&s.AccountSid, "SID of the account that owns the version.")
	a.Describe(&s.Url, "The absolute URL of the version resource.")
	a.Describe(&s.DateCreated, "When the version was created (ISO 8601).")
}

func (r *ServerlessFunctionVersion) Annotate(a infer.Annotator) {
	a.Describe(&r, "One immutable version of a Twilio Function: the source code uploaded to a ServerlessFunction. The provider uploads the code itself (multipart POST to serverless-upload.twilio.com), so a version created here can go straight into a ServerlessBuild. Every input replaces the version. Twilio has no delete for versions, so destroying the resource only removes it from Pulumi state. The resource ID is <serviceSid>/<functionSid>/<sid>.")
}

var _ infer.CustomResource[ServerlessFunctionVersionArgs, ServerlessFunctionVersionState] = (*ServerlessFunctionVersion)(nil)
var _ infer.CustomCheck[ServerlessFunctionVersionArgs] = (*ServerlessFunctionVersion)(nil)
var _ infer.CustomDiff[ServerlessFunctionVersionArgs, ServerlessFunctionVersionState] = (*ServerlessFunctionVersion)(nil)
var _ infer.CustomRead[ServerlessFunctionVersionArgs, ServerlessFunctionVersionState] = (*ServerlessFunctionVersion)(nil)
var _ infer.CustomDelete[ServerlessFunctionVersionState] = (*ServerlessFunctionVersion)(nil)

func (r *ServerlessFunctionVersion) Check(ctx context.Context, req infer.CheckRequest) (infer.CheckResponse[ServerlessFunctionVersionArgs], error) {
	args, failures, err := infer.DefaultCheck[ServerlessFunctionVersionArgs](ctx, req.NewInputs)
	if err != nil || len(failures) > 0 {
		return infer.CheckResponse[ServerlessFunctionVersionArgs]{Inputs: args, Failures: failures}, err
	}
	c := &checker{unk: unknowns(req.NewInputs)}
	c.sid("serviceSid", args.ServiceSid, serverlessServiceSidRE, "Serverless service (ZS followed by 32 hex characters)")
	c.sid("functionSid", args.FunctionSid, serverlessContainerSidRE, "Function (ZH followed by 32 hex characters)")
	c.pathProp("path", args.Path)
	c.oneOf("visibility", args.Visibility, serverlessVisibilities)
	c.required("content", args.Content)
	return infer.CheckResponse[ServerlessFunctionVersionArgs]{Inputs: args, Failures: c.failures}, nil
}

// versionReplace builds the diff of a version: every input replaces. Content is compared only when
// the state holds it, so an imported asset version (whose bytes Twilio will not return) plans nothing.
func versionReplace(changed map[string]bool) p.DiffResponse {
	detail := map[string]p.PropertyDiff{}
	for k, v := range changed {
		if v {
			detail[k] = p.PropertyDiff{Kind: p.UpdateReplace}
		}
	}
	return p.DiffResponse{HasChanges: len(detail) > 0, DeleteBeforeReplace: false, DetailedDiff: detail}
}

func strPtrChanged(old, next *string, def string) bool {
	return derefOr(old, def) != derefOr(next, def)
}

func (r *ServerlessFunctionVersion) Diff(_ context.Context, req infer.DiffRequest[ServerlessFunctionVersionArgs, ServerlessFunctionVersionState]) (infer.DiffResponse, error) {
	o, n := req.State.ServerlessFunctionVersionArgs, req.Inputs
	d := versionReplace(map[string]bool{
		"serviceSid":  o.ServiceSid != n.ServiceSid,
		"functionSid": o.FunctionSid != n.FunctionSid,
		"path":        o.Path != n.Path,
		"visibility":  o.Visibility != n.Visibility,
		"content":     o.Content != "" && o.Content != n.Content,
		"contentType": o.ContentType != nil && strPtrChanged(o.ContentType, n.ContentType, "application/javascript"),
	})
	return infer.DiffResponse{HasChanges: d.HasChanges, DetailedDiff: d.DetailedDiff}, nil
}

func versionIdentity(sid string, o obj) (account, u, created string) {
	return o.str("account_sid"), o.str("url"), o.str("date_created")
}

func (r *ServerlessFunctionVersion) Create(ctx context.Context, req infer.CreateRequest[ServerlessFunctionVersionArgs]) (infer.CreateResponse[ServerlessFunctionVersionState], error) {
	a := req.Inputs
	if req.DryRun {
		return infer.CreateResponse[ServerlessFunctionVersionState]{Output: ServerlessFunctionVersionState{ServerlessFunctionVersionArgs: a}}, nil
	}
	c, err := clientFor(ctx)
	if err != nil {
		return infer.CreateResponse[ServerlessFunctionVersionState]{}, err
	}
	o, err := uploadVersion(ctx, c, a.ServiceSid, "Functions", a.FunctionSid, a.Path, a.Visibility, derefOr(a.ContentType, "application/javascript"), []byte(a.Content))
	if err != nil {
		return infer.CreateResponse[ServerlessFunctionVersionState]{}, fmt.Errorf("upload version of function %s on serverless service %s: %w", a.FunctionSid, a.ServiceSid, err)
	}
	st := ServerlessFunctionVersionState{ServerlessFunctionVersionArgs: a, Sid: o.str("sid")}
	st.AccountSid, st.Url, st.DateCreated = versionIdentity(st.Sid, o)
	return infer.CreateResponse[ServerlessFunctionVersionState]{ID: a.ServiceSid + "/" + a.FunctionSid + "/" + st.Sid, Output: st}, nil
}

func (r *ServerlessFunctionVersion) Read(ctx context.Context, req infer.ReadRequest[ServerlessFunctionVersionArgs, ServerlessFunctionVersionState]) (infer.ReadResponse[ServerlessFunctionVersionArgs, ServerlessFunctionVersionState], error) {
	type resp = infer.ReadResponse[ServerlessFunctionVersionArgs, ServerlessFunctionVersionState]
	ids, err := splitVerifyID(req.ID, 3, "<serviceSid>/<functionSid>/<sid>")
	if err != nil {
		return resp{}, err
	}
	c, err := clientFor(ctx)
	if err != nil {
		return resp{}, err
	}
	o, err := c.do(ctx, "GET", "serverless", serverlessPath(ids[0], "Functions", ids[1], "Versions", ids[2]), nil, nil)
	if err != nil {
		if isNotFound(err) {
			if err := notFoundOnImport("serverless function version", req.ID, req.Inputs, req.State); err != nil {
				return resp{}, err
			}
			return resp{}, nil
		}
		return resp{}, fmt.Errorf("read serverless function version %s: %w", req.ID, err)
	}
	in := req.Inputs
	if in.ServiceSid == "" {
		in = ServerlessFunctionVersionArgs{ServiceSid: ids[0], FunctionSid: ids[1]}
	}
	if v := o.str("path"); v != "" {
		in.Path = v
	}
	if v := o.str("visibility"); v != "" {
		in.Visibility = v
	}
	if in.Content == "" {
		// An import: function code can be read back.
		if cb, err := c.do(ctx, "GET", "serverless", serverlessPath(ids[0], "Functions", ids[1], "Versions", ids[2], "Content"), nil, nil); err == nil {
			in.Content = cb.str("content")
		}
	}
	st := ServerlessFunctionVersionState{ServerlessFunctionVersionArgs: in, Sid: o.str("sid")}
	st.AccountSid, st.Url, st.DateCreated = versionIdentity(st.Sid, o)
	return resp{ID: req.ID, Inputs: in, State: st}, nil
}

func (r *ServerlessFunctionVersion) Delete(_ context.Context, _ infer.DeleteRequest[ServerlessFunctionVersionState]) (infer.DeleteResponse, error) {
	// Twilio has no delete for Function Versions; they stay in the service, unreferenced once no Build uses them.
	return infer.DeleteResponse{}, nil
}

// ------------------------------------------------------------ ServerlessAssetVersion

// ServerlessAssetVersion uploads one immutable version of an Asset: a static file served from the service's domain.
type ServerlessAssetVersion struct{}

// ServerlessAssetVersionArgs are the inputs of a ServerlessAssetVersion.
type ServerlessAssetVersionArgs struct {
	ServiceSid    string  `pulumi:"serviceSid"`
	AssetSid      string  `pulumi:"assetSid"`
	Path          string  `pulumi:"path"`
	Visibility    string  `pulumi:"visibility"`
	Content       *string `pulumi:"content,optional"`
	ContentBase64 *string `pulumi:"contentBase64,optional"`
	ContentType   *string `pulumi:"contentType,optional"`
}

func (a *ServerlessAssetVersionArgs) Annotate(an infer.Annotator) {
	an.Describe(&a.ServiceSid, "SID (ZS...) of the Serverless service. Changing it replaces the resource.")
	an.Describe(&a.AssetSid, "SID (ZH...) of the Asset (a ServerlessAsset) the version belongs to. Changing it replaces the resource.")
	an.Describe(&a.Path, "URL path the asset is served on, starting with /, at most 255 characters, for example /logo.svg. Changing it replaces the resource.")
	an.Describe(&a.Visibility, "Who may fetch the asset: public (anyone), protected (requests signed by Twilio) or private (only Functions of the service). Changing it replaces the resource.")
	an.Describe(&a.Content, "The file as text, for HTML, CSS, JSON or SVG. Set exactly one of content and contentBase64. Twilio does not return asset bytes, so after an import this stays unset and a program that sets it plans no change.")
	an.Describe(&a.ContentBase64, "The file as base64, for binary content such as images. Set exactly one of content and contentBase64.")
	an.Describe(&a.ContentType, "Content type of the uploaded file. Defaults to application/octet-stream; set it (for example text/html) so Twilio serves the asset with the right type.")
}

// ServerlessAssetVersionState is the persisted state of a ServerlessAssetVersion.
type ServerlessAssetVersionState struct {
	ServerlessAssetVersionArgs

	// Sid is the version SID (ZN...).
	Sid         string `pulumi:"sid"`
	AccountSid  string `pulumi:"accountSid"`
	Url         string `pulumi:"url"`
	DateCreated string `pulumi:"dateCreated"`
}

func (s *ServerlessAssetVersionState) Annotate(a infer.Annotator) {
	a.Describe(&s.Sid, "The version SID (ZN...). Pass it to ServerlessBuild.assetVersionSids.")
	a.Describe(&s.AccountSid, "SID of the account that owns the version.")
	a.Describe(&s.Url, "The absolute URL of the version resource.")
	a.Describe(&s.DateCreated, "When the version was created (ISO 8601).")
}

func (r *ServerlessAssetVersion) Annotate(a infer.Annotator) {
	a.Describe(&r, "One immutable version of a Twilio Asset: a static file uploaded to a ServerlessAsset. The provider uploads the file itself (multipart POST to serverless-upload.twilio.com), so a version created here can go straight into a ServerlessBuild. Every input replaces the version. Twilio has no delete for versions, so destroying the resource only removes it from Pulumi state. The resource ID is <serviceSid>/<assetSid>/<sid>.")
}

var _ infer.CustomResource[ServerlessAssetVersionArgs, ServerlessAssetVersionState] = (*ServerlessAssetVersion)(nil)
var _ infer.CustomCheck[ServerlessAssetVersionArgs] = (*ServerlessAssetVersion)(nil)
var _ infer.CustomDiff[ServerlessAssetVersionArgs, ServerlessAssetVersionState] = (*ServerlessAssetVersion)(nil)
var _ infer.CustomRead[ServerlessAssetVersionArgs, ServerlessAssetVersionState] = (*ServerlessAssetVersion)(nil)
var _ infer.CustomDelete[ServerlessAssetVersionState] = (*ServerlessAssetVersion)(nil)

func (r *ServerlessAssetVersion) Check(ctx context.Context, req infer.CheckRequest) (infer.CheckResponse[ServerlessAssetVersionArgs], error) {
	args, failures, err := infer.DefaultCheck[ServerlessAssetVersionArgs](ctx, req.NewInputs)
	if err != nil || len(failures) > 0 {
		return infer.CheckResponse[ServerlessAssetVersionArgs]{Inputs: args, Failures: failures}, err
	}
	c := &checker{unk: unknowns(req.NewInputs)}
	c.sid("serviceSid", args.ServiceSid, serverlessServiceSidRE, "Serverless service (ZS followed by 32 hex characters)")
	c.sid("assetSid", args.AssetSid, serverlessContainerSidRE, "Asset (ZH followed by 32 hex characters)")
	c.pathProp("path", args.Path)
	c.oneOf("visibility", args.Visibility, serverlessVisibilities)
	if !c.unk("content", "contentBase64") {
		if (args.Content == nil) == (args.ContentBase64 == nil) {
			c.fail("content", "set exactly one of content and contentBase64")
		} else if args.ContentBase64 != nil {
			if _, err := base64.StdEncoding.DecodeString(*args.ContentBase64); err != nil {
				c.fail("contentBase64", "contentBase64 must be valid base64")
			}
		}
	}
	return infer.CheckResponse[ServerlessAssetVersionArgs]{Inputs: args, Failures: c.failures}, nil
}

func (r *ServerlessAssetVersion) Diff(_ context.Context, req infer.DiffRequest[ServerlessAssetVersionArgs, ServerlessAssetVersionState]) (infer.DiffResponse, error) {
	o, n := req.State.ServerlessAssetVersionArgs, req.Inputs
	haveBytes := o.Content != nil || o.ContentBase64 != nil
	d := versionReplace(map[string]bool{
		"serviceSid":    o.ServiceSid != n.ServiceSid,
		"assetSid":      o.AssetSid != n.AssetSid,
		"path":          o.Path != n.Path,
		"visibility":    o.Visibility != n.Visibility,
		"content":       haveBytes && strPtrChanged(o.Content, n.Content, ""),
		"contentBase64": haveBytes && strPtrChanged(o.ContentBase64, n.ContentBase64, ""),
		"contentType":   o.ContentType != nil && strPtrChanged(o.ContentType, n.ContentType, "application/octet-stream"),
	})
	return infer.DiffResponse{HasChanges: d.HasChanges, DetailedDiff: d.DetailedDiff}, nil
}

func (r *ServerlessAssetVersion) Create(ctx context.Context, req infer.CreateRequest[ServerlessAssetVersionArgs]) (infer.CreateResponse[ServerlessAssetVersionState], error) {
	a := req.Inputs
	if req.DryRun {
		return infer.CreateResponse[ServerlessAssetVersionState]{Output: ServerlessAssetVersionState{ServerlessAssetVersionArgs: a}}, nil
	}
	var content []byte
	if a.ContentBase64 != nil {
		b, err := base64.StdEncoding.DecodeString(*a.ContentBase64)
		if err != nil {
			return infer.CreateResponse[ServerlessAssetVersionState]{}, fmt.Errorf("contentBase64 is not valid base64: %w", err)
		}
		content = b
	} else {
		content = []byte(derefOr(a.Content, ""))
	}
	c, err := clientFor(ctx)
	if err != nil {
		return infer.CreateResponse[ServerlessAssetVersionState]{}, err
	}
	o, err := uploadVersion(ctx, c, a.ServiceSid, "Assets", a.AssetSid, a.Path, a.Visibility, derefOr(a.ContentType, "application/octet-stream"), content)
	if err != nil {
		return infer.CreateResponse[ServerlessAssetVersionState]{}, fmt.Errorf("upload version of asset %s on serverless service %s: %w", a.AssetSid, a.ServiceSid, err)
	}
	st := ServerlessAssetVersionState{ServerlessAssetVersionArgs: a, Sid: o.str("sid")}
	st.AccountSid, st.Url, st.DateCreated = versionIdentity(st.Sid, o)
	return infer.CreateResponse[ServerlessAssetVersionState]{ID: a.ServiceSid + "/" + a.AssetSid + "/" + st.Sid, Output: st}, nil
}

func (r *ServerlessAssetVersion) Read(ctx context.Context, req infer.ReadRequest[ServerlessAssetVersionArgs, ServerlessAssetVersionState]) (infer.ReadResponse[ServerlessAssetVersionArgs, ServerlessAssetVersionState], error) {
	type resp = infer.ReadResponse[ServerlessAssetVersionArgs, ServerlessAssetVersionState]
	ids, err := splitVerifyID(req.ID, 3, "<serviceSid>/<assetSid>/<sid>")
	if err != nil {
		return resp{}, err
	}
	c, err := clientFor(ctx)
	if err != nil {
		return resp{}, err
	}
	o, err := c.do(ctx, "GET", "serverless", serverlessPath(ids[0], "Assets", ids[1], "Versions", ids[2]), nil, nil)
	if err != nil {
		if isNotFound(err) {
			if err := notFoundOnImport("serverless asset version", req.ID, req.Inputs, req.State); err != nil {
				return resp{}, err
			}
			return resp{}, nil
		}
		return resp{}, fmt.Errorf("read serverless asset version %s: %w", req.ID, err)
	}
	in := req.Inputs
	if in.ServiceSid == "" {
		in = ServerlessAssetVersionArgs{ServiceSid: ids[0], AssetSid: ids[1]}
	}
	if v := o.str("path"); v != "" {
		in.Path = v
	}
	if v := o.str("visibility"); v != "" {
		in.Visibility = v
	}
	st := ServerlessAssetVersionState{ServerlessAssetVersionArgs: in, Sid: o.str("sid")}
	st.AccountSid, st.Url, st.DateCreated = versionIdentity(st.Sid, o)
	return resp{ID: req.ID, Inputs: in, State: st}, nil
}

func (r *ServerlessAssetVersion) Delete(_ context.Context, _ infer.DeleteRequest[ServerlessAssetVersionState]) (infer.DeleteResponse, error) {
	// Twilio has no delete for Asset Versions.
	return infer.DeleteResponse{}, nil
}

// ------------------------------------------------------------ ServerlessBuild

// ServerlessBuild assembles function and asset versions plus npm dependencies into a Build and waits for Twilio to finish building it.
type ServerlessBuild struct{}

// ServerlessBuildDependency is one npm package installed into a Build.
type ServerlessBuildDependency struct {
	Name    string `pulumi:"name" json:"name"`
	Version string `pulumi:"version" json:"version"`
}

func (d *ServerlessBuildDependency) Annotate(an infer.Annotator) {
	an.Describe(&d.Name, "npm package name, for example twilio.")
	an.Describe(&d.Version, "Exact version or range of the package, for example 5.3.0 or ^5.3.0.")
}

// ServerlessBuildArgs are the inputs of a ServerlessBuild.
type ServerlessBuildArgs struct {
	ServiceSid          string                       `pulumi:"serviceSid"`
	FunctionVersionSids *[]string                    `pulumi:"functionVersionSids,optional"`
	AssetVersionSids    *[]string                    `pulumi:"assetVersionSids,optional"`
	Dependencies        *[]ServerlessBuildDependency `pulumi:"dependencies,optional"`
	Runtime             *string                      `pulumi:"runtime,optional"`
}

func (a *ServerlessBuildArgs) Annotate(an infer.Annotator) {
	an.Describe(&a.ServiceSid, "SID (ZS...) of the Serverless service. Changing it replaces the resource.")
	an.Describe(&a.FunctionVersionSids, "SIDs (ZN...) of the Function versions to include, typically ServerlessFunctionVersion.sid. Versions uploaded outside Pulumi (Twilio CLI, Serverless Toolkit) work too. Changing the list replaces the build.")
	an.Describe(&a.AssetVersionSids, "SIDs (ZN...) of the Asset versions to include, typically ServerlessAssetVersion.sid. Changing the list replaces the build.")
	an.Describe(&a.Dependencies, "npm packages to install into the build. Changing them replaces the build.")
	an.Describe(&a.Runtime, "Node.js runtime the build runs on: node8 to node24. Defaults to what Twilio picks when unset. Changing it replaces the build.")
}

// ServerlessBuildState is the persisted state of a ServerlessBuild.
type ServerlessBuildState struct {
	ServerlessBuildArgs

	// Sid is the build SID (ZB...).
	Sid         string `pulumi:"sid"`
	AccountSid  string `pulumi:"accountSid"`
	Status      string `pulumi:"status"`
	Url         string `pulumi:"url"`
	DateCreated string `pulumi:"dateCreated"`
	DateUpdated string `pulumi:"dateUpdated"`
}

func (s *ServerlessBuildState) Annotate(a infer.Annotator) {
	a.Describe(&s.Sid, "The build SID (ZB...). Pass it to ServerlessDeployment.buildSid.")
	a.Describe(&s.AccountSid, "SID of the account that owns the build.")
	a.Describe(&s.Status, "Build status: building, completed or failed. The resource is only created once the build is completed.")
	a.Describe(&s.Url, "The absolute URL of the build resource.")
	a.Describe(&s.DateCreated, "When the build was created (ISO 8601).")
	a.Describe(&s.DateUpdated, "When the build was last updated (ISO 8601).")
}

func (r *ServerlessBuild) Annotate(a infer.Annotator) {
	a.Describe(&r, "A Twilio Serverless build: function versions, asset versions and npm dependencies assembled into one deployable unit. Creating it waits until Twilio reports the build completed and fails if the build fails. Builds are immutable, so every input replaces the build. Deploy one with a ServerlessDeployment. Twilio refuses to delete a build that was deployed (and has no delete for deployments), so destroying such a build only removes it from Pulumi state with a warning; it stays on Twilio until its service is deleted. A build that was never deployed is deleted. The resource ID is <serviceSid>/<sid>.")
}

var _ infer.CustomResource[ServerlessBuildArgs, ServerlessBuildState] = (*ServerlessBuild)(nil)
var _ infer.CustomCheck[ServerlessBuildArgs] = (*ServerlessBuild)(nil)
var _ infer.CustomDiff[ServerlessBuildArgs, ServerlessBuildState] = (*ServerlessBuild)(nil)
var _ infer.CustomRead[ServerlessBuildArgs, ServerlessBuildState] = (*ServerlessBuild)(nil)
var _ infer.CustomDelete[ServerlessBuildState] = (*ServerlessBuild)(nil)

func (r *ServerlessBuild) Check(ctx context.Context, req infer.CheckRequest) (infer.CheckResponse[ServerlessBuildArgs], error) {
	args, failures, err := infer.DefaultCheck[ServerlessBuildArgs](ctx, req.NewInputs)
	if err != nil || len(failures) > 0 {
		return infer.CheckResponse[ServerlessBuildArgs]{Inputs: args, Failures: failures}, err
	}
	c := &checker{unk: unknowns(req.NewInputs)}
	c.sid("serviceSid", args.ServiceSid, serverlessServiceSidRE, "Serverless service (ZS followed by 32 hex characters)")
	for prop, list := range map[string]*[]string{"functionVersionSids": args.FunctionVersionSids, "assetVersionSids": args.AssetVersionSids} {
		if list == nil || c.unk(prop) {
			continue
		}
		for i, s := range *list {
			// A known list can hold a computed element (functionVersionSids=[fnVersion.sid]); infer decodes it to "".
			if unknownAt(req.NewInputs, prop, i) {
				continue
			}
			if !serverlessVersionSidRE.MatchString(s) {
				c.fail(prop, prop+" must hold version SIDs (ZN followed by 32 hex characters)")
				break
			}
		}
	}
	if args.Runtime != nil {
		c.oneOf("runtime", *args.Runtime, serverlessRuntimes)
	}
	if args.Dependencies != nil && !c.unk("dependencies") {
		for i, d := range *args.Dependencies {
			if unknownAt(req.NewInputs, "dependencies", i) {
				continue
			}
			if (d.Name == "" && !unknownAt(req.NewInputs, "dependencies", i, "name")) || (d.Version == "" && !unknownAt(req.NewInputs, "dependencies", i, "version")) {
				c.fail("dependencies", "every dependency needs a name and a version")
				break
			}
		}
	}
	return infer.CheckResponse[ServerlessBuildArgs]{Inputs: args, Failures: c.failures}, nil
}

func sameList(a, b *[]string) bool {
	var x, y []string
	if a != nil {
		x = *a
	}
	if b != nil {
		y = *b
	}
	if len(x) != len(y) {
		return false
	}
	for i := range x {
		if x[i] != y[i] {
			return false
		}
	}
	return true
}

func sameDeps(a, b *[]ServerlessBuildDependency) bool {
	var x, y []ServerlessBuildDependency
	if a != nil {
		x = *a
	}
	if b != nil {
		y = *b
	}
	if len(x) != len(y) {
		return false
	}
	for i := range x {
		if x[i] != y[i] {
			return false
		}
	}
	return true
}

func (r *ServerlessBuild) Diff(_ context.Context, req infer.DiffRequest[ServerlessBuildArgs, ServerlessBuildState]) (infer.DiffResponse, error) {
	o, n := req.State.ServerlessBuildArgs, req.Inputs
	d := versionReplace(map[string]bool{
		"serviceSid":          o.ServiceSid != n.ServiceSid,
		"functionVersionSids": !sameList(o.FunctionVersionSids, n.FunctionVersionSids),
		"assetVersionSids":    !sameList(o.AssetVersionSids, n.AssetVersionSids),
		"dependencies":        !sameDeps(o.Dependencies, n.Dependencies),
		// An unset runtime keeps whatever Twilio chose.
		"runtime": n.Runtime != nil && strPtrChanged(o.Runtime, n.Runtime, ""),
	})
	return infer.DiffResponse{HasChanges: d.HasChanges, DetailedDiff: d.DetailedDiff}, nil
}

func buildOutputs(st *ServerlessBuildState, o obj) {
	st.Sid, st.AccountSid, st.Status, st.Url = o.str("sid"), o.str("account_sid"), o.str("status"), o.str("url")
	st.DateCreated, st.DateUpdated = o.str("date_created"), o.str("date_updated")
}

// waitForBuild polls the build status until it completes, fails, or the deadline passes.
func waitForBuild(ctx context.Context, c *Client, svc, sid string) (string, error) {
	ctx, cancel := context.WithTimeout(ctx, buildPollTimeout)
	defer cancel()
	for {
		o, err := c.do(ctx, "GET", "serverless", serverlessPath(svc, "Builds", sid, "Status"), nil, nil)
		if err != nil {
			if ctx.Err() != nil {
				return "", fmt.Errorf("build %s did not finish within %s: %w", sid, buildPollTimeout, ctx.Err())
			}
			return "", err
		}
		switch s := o.str("status"); s {
		case "completed", "failed":
			return s, nil
		}
		select {
		case <-ctx.Done():
			return "", fmt.Errorf("build %s did not finish within %s: %w", sid, buildPollTimeout, ctx.Err())
		case <-time.After(buildPollInterval):
		}
	}
}

func (r *ServerlessBuild) Create(ctx context.Context, req infer.CreateRequest[ServerlessBuildArgs]) (infer.CreateResponse[ServerlessBuildState], error) {
	a := req.Inputs
	if req.DryRun {
		return infer.CreateResponse[ServerlessBuildState]{Output: ServerlessBuildState{ServerlessBuildArgs: a}}, nil
	}
	c, err := clientFor(ctx)
	if err != nil {
		return infer.CreateResponse[ServerlessBuildState]{}, err
	}
	form := url.Values{}
	if a.FunctionVersionSids != nil {
		form["FunctionVersions"] = append([]string{}, *a.FunctionVersionSids...)
	}
	if a.AssetVersionSids != nil {
		form["AssetVersions"] = append([]string{}, *a.AssetVersionSids...)
	}
	if a.Dependencies != nil && len(*a.Dependencies) > 0 {
		raw, err := json.Marshal(*a.Dependencies)
		if err != nil {
			return infer.CreateResponse[ServerlessBuildState]{}, err
		}
		form.Set("Dependencies", string(raw))
	}
	setS(form, "Runtime", a.Runtime)
	o, err := c.do(ctx, "POST", "serverless", serverlessSvcPath(a.ServiceSid, "Builds", ""), nil, form)
	if err != nil {
		return infer.CreateResponse[ServerlessBuildState]{}, fmt.Errorf("create build on serverless service %s: %w", a.ServiceSid, err)
	}
	sid := o.str("sid")
	status, err := waitForBuild(ctx, c, a.ServiceSid, sid)
	if err == nil && status == "failed" {
		err = fmt.Errorf("build %s failed; check the build logs in the Twilio Console", sid)
	}
	if err != nil {
		// Do not leave a build nobody tracks behind.
		_, _ = c.do(context.WithoutCancel(ctx), "DELETE", "serverless", serverlessSvcPath(a.ServiceSid, "Builds", sid), nil, nil)
		return infer.CreateResponse[ServerlessBuildState]{}, fmt.Errorf("serverless service %s: %w", a.ServiceSid, err)
	}
	// The build exists and is complete. If the read-back fails, keep what the create call returned
	// rather than failing: an error here would leave a real build that no state tracks.
	created := o
	if got, gerr := c.do(ctx, "GET", "serverless", serverlessSvcPath(a.ServiceSid, "Builds", sid), nil, nil); gerr == nil {
		o = got
	} else {
		o = created
		o["status"] = "completed"
	}
	st := ServerlessBuildState{ServerlessBuildArgs: a}
	buildOutputs(&st, o)
	return infer.CreateResponse[ServerlessBuildState]{ID: a.ServiceSid + "/" + sid, Output: st}, nil
}

func versionList(o obj, k string) *[]string { return o.optList(k) }

func buildDeps(o obj) *[]ServerlessBuildDependency {
	raw, ok := o.get("dependencies").([]any)
	if !ok {
		return nil
	}
	out := []ServerlessBuildDependency{}
	for _, it := range raw {
		if m, ok := it.(map[string]any); ok {
			out = append(out, ServerlessBuildDependency{Name: obj(m).str("name"), Version: obj(m).str("version")})
		}
	}
	return &out
}

func (r *ServerlessBuild) Read(ctx context.Context, req infer.ReadRequest[ServerlessBuildArgs, ServerlessBuildState]) (infer.ReadResponse[ServerlessBuildArgs, ServerlessBuildState], error) {
	type resp = infer.ReadResponse[ServerlessBuildArgs, ServerlessBuildState]
	ids, err := splitVerifyID(req.ID, 2, "<serviceSid>/<sid>")
	if err != nil {
		return resp{}, err
	}
	c, err := clientFor(ctx)
	if err != nil {
		return resp{}, err
	}
	o, err := c.do(ctx, "GET", "serverless", serverlessSvcPath(ids[0], "Builds", ids[1]), nil, nil)
	if err != nil {
		if isNotFound(err) {
			if err := notFoundOnImport("serverless build", req.ID, req.Inputs, req.State); err != nil {
				return resp{}, err
			}
			return resp{}, nil
		}
		return resp{}, fmt.Errorf("read serverless build %s: %w", req.ID, err)
	}
	in := req.Inputs
	if in.ServiceSid == "" {
		// An import adopts everything the build holds; builds are immutable, so a refresh never needs to.
		in = ServerlessBuildArgs{ServiceSid: ids[0], FunctionVersionSids: versionList(o, "function_versions"), AssetVersionSids: versionList(o, "asset_versions"), Dependencies: buildDeps(o), Runtime: o.optStr("runtime")}
	}
	st := ServerlessBuildState{ServerlessBuildArgs: in}
	buildOutputs(&st, o)
	return resp{ID: req.ID, Inputs: in, State: st}, nil
}

func (r *ServerlessBuild) Delete(ctx context.Context, req infer.DeleteRequest[ServerlessBuildState]) (infer.DeleteResponse, error) {
	ids, err := splitVerifyID(req.ID, 2, "<serviceSid>/<sid>")
	if err != nil {
		return infer.DeleteResponse{}, err
	}
	c, err := clientFor(ctx)
	if err != nil {
		return infer.DeleteResponse{}, err
	}
	if _, err := c.do(ctx, "DELETE", "serverless", serverlessSvcPath(ids[0], "Builds", ids[1]), nil, nil); err != nil && !isNotFound(err) {
		// Twilio has no delete for a Deployment, so a Build that was ever deployed cannot be deleted
		// ("Cannot delete Build because it's part of an active Deployment", HTTP 400, error 20001).
		// Forget it: deleting the Service (or its Environment) removes it, and failing here would make
		// every destroy of a deployed Function fail (#1856).
		if isRefusal(err, "deployment") {
			p.GetLogger(ctx).Warningf("serverless build %s is part of a deployment, which Twilio cannot delete: it is removed from the state "+
				"and stays on Twilio until its service is deleted", req.ID)
			return infer.DeleteResponse{}, nil
		}
		return infer.DeleteResponse{}, fmt.Errorf("delete serverless build %s: %w", req.ID, err)
	}
	return infer.DeleteResponse{}, nil
}

// ------------------------------------------------------------ ServerlessDeployment

// ServerlessDeployment deploys a finished Build to an environment of a Serverless service.
type ServerlessDeployment struct{}

// ServerlessDeploymentArgs are the inputs of a ServerlessDeployment.
type ServerlessDeploymentArgs struct {
	ServiceSid     string `pulumi:"serviceSid"`
	EnvironmentSid string `pulumi:"environmentSid"`
	BuildSid       string `pulumi:"buildSid"`
	IsPlugin       *bool  `pulumi:"isPlugin,optional"`
}

func (a *ServerlessDeploymentArgs) Annotate(an infer.Annotator) {
	an.Describe(&a.ServiceSid, "SID (ZS...) of the Serverless service. Changing it replaces the resource.")
	an.Describe(&a.EnvironmentSid, "SID (ZE...) of the environment to deploy to (a ServerlessEnvironment). Changing it replaces the resource.")
	an.Describe(&a.BuildSid, "SID (ZB...) of the build to deploy (a ServerlessBuild). Changing it deploys the new build and replaces the resource.")
	an.Describe(&a.IsPlugin, "Whether the deployment is a Flex plugin deployment. Twilio does not return it, so it is kept as given. Changing it replaces the resource.")
}

// ServerlessDeploymentState is the persisted state of a ServerlessDeployment.
type ServerlessDeploymentState struct {
	ServerlessDeploymentArgs

	// Sid is the deployment SID (ZD...).
	Sid         string `pulumi:"sid"`
	AccountSid  string `pulumi:"accountSid"`
	Url         string `pulumi:"url"`
	DateCreated string `pulumi:"dateCreated"`
	DateUpdated string `pulumi:"dateUpdated"`
}

func (s *ServerlessDeploymentState) Annotate(a infer.Annotator) {
	a.Describe(&s.Sid, "The deployment SID (ZD...).")
	a.Describe(&s.AccountSid, "SID of the account that owns the deployment.")
	a.Describe(&s.Url, "The absolute URL of the deployment resource.")
	a.Describe(&s.DateCreated, "When the deployment was created (ISO 8601).")
	a.Describe(&s.DateUpdated, "When the deployment was last updated (ISO 8601).")
}

func (r *ServerlessDeployment) Annotate(a infer.Annotator) {
	a.Describe(&r, "Deploys a ServerlessBuild to a ServerlessEnvironment, which makes the build's functions and assets live on the environment's domain. Deployments are immutable, so every input replaces the resource, which deploys again. Twilio has no delete for deployments: destroying the resource only removes it from Pulumi state and the environment keeps serving the build. The resource ID is <serviceSid>/<environmentSid>/<sid>.")
}

var _ infer.CustomResource[ServerlessDeploymentArgs, ServerlessDeploymentState] = (*ServerlessDeployment)(nil)
var _ infer.CustomCheck[ServerlessDeploymentArgs] = (*ServerlessDeployment)(nil)
var _ infer.CustomDiff[ServerlessDeploymentArgs, ServerlessDeploymentState] = (*ServerlessDeployment)(nil)
var _ infer.CustomRead[ServerlessDeploymentArgs, ServerlessDeploymentState] = (*ServerlessDeployment)(nil)
var _ infer.CustomDelete[ServerlessDeploymentState] = (*ServerlessDeployment)(nil)

func (r *ServerlessDeployment) Check(ctx context.Context, req infer.CheckRequest) (infer.CheckResponse[ServerlessDeploymentArgs], error) {
	args, failures, err := infer.DefaultCheck[ServerlessDeploymentArgs](ctx, req.NewInputs)
	if err != nil || len(failures) > 0 {
		return infer.CheckResponse[ServerlessDeploymentArgs]{Inputs: args, Failures: failures}, err
	}
	c := &checker{unk: unknowns(req.NewInputs)}
	c.sid("serviceSid", args.ServiceSid, serverlessServiceSidRE, "Serverless service (ZS followed by 32 hex characters)")
	c.sid("environmentSid", args.EnvironmentSid, serverlessEnvironmentSidRE, "Serverless environment (ZE followed by 32 hex characters)")
	c.sid("buildSid", args.BuildSid, serverlessBuildSidRE, "Serverless build (ZB followed by 32 hex characters)")
	return infer.CheckResponse[ServerlessDeploymentArgs]{Inputs: args, Failures: c.failures}, nil
}

func (r *ServerlessDeployment) Diff(_ context.Context, req infer.DiffRequest[ServerlessDeploymentArgs, ServerlessDeploymentState]) (infer.DiffResponse, error) {
	o, n := req.State.ServerlessDeploymentArgs, req.Inputs
	d := versionReplace(map[string]bool{
		"serviceSid":     o.ServiceSid != n.ServiceSid,
		"environmentSid": o.EnvironmentSid != n.EnvironmentSid,
		"buildSid":       o.BuildSid != n.BuildSid,
		"isPlugin":       n.IsPlugin != nil && (o.IsPlugin == nil || *o.IsPlugin != *n.IsPlugin),
	})
	return infer.DiffResponse{HasChanges: d.HasChanges, DetailedDiff: d.DetailedDiff}, nil
}

func deploymentOutputs(st *ServerlessDeploymentState, o obj) {
	st.Sid, st.AccountSid, st.Url = o.str("sid"), o.str("account_sid"), o.str("url")
	st.DateCreated, st.DateUpdated = o.str("date_created"), o.str("date_updated")
}

func deploymentPath(svc, env, sid string) string {
	return serverlessPath(svc, "Environments", env, "Deployments", sid)
}

func (r *ServerlessDeployment) Create(ctx context.Context, req infer.CreateRequest[ServerlessDeploymentArgs]) (infer.CreateResponse[ServerlessDeploymentState], error) {
	a := req.Inputs
	if req.DryRun {
		return infer.CreateResponse[ServerlessDeploymentState]{Output: ServerlessDeploymentState{ServerlessDeploymentArgs: a}}, nil
	}
	c, err := clientFor(ctx)
	if err != nil {
		return infer.CreateResponse[ServerlessDeploymentState]{}, err
	}
	form := url.Values{"BuildSid": {a.BuildSid}}
	setB(form, "IsPlugin", a.IsPlugin)
	o, err := c.do(ctx, "POST", "serverless", deploymentPath(a.ServiceSid, a.EnvironmentSid, ""), nil, form)
	if err != nil {
		return infer.CreateResponse[ServerlessDeploymentState]{}, fmt.Errorf("deploy build %s to environment %s of serverless service %s: %w", a.BuildSid, a.EnvironmentSid, a.ServiceSid, err)
	}
	st := ServerlessDeploymentState{ServerlessDeploymentArgs: a}
	deploymentOutputs(&st, o)
	return infer.CreateResponse[ServerlessDeploymentState]{ID: a.ServiceSid + "/" + a.EnvironmentSid + "/" + st.Sid, Output: st}, nil
}

func (r *ServerlessDeployment) Read(ctx context.Context, req infer.ReadRequest[ServerlessDeploymentArgs, ServerlessDeploymentState]) (infer.ReadResponse[ServerlessDeploymentArgs, ServerlessDeploymentState], error) {
	type resp = infer.ReadResponse[ServerlessDeploymentArgs, ServerlessDeploymentState]
	ids, err := splitVerifyID(req.ID, 3, "<serviceSid>/<environmentSid>/<sid>")
	if err != nil {
		return resp{}, err
	}
	c, err := clientFor(ctx)
	if err != nil {
		return resp{}, err
	}
	o, err := c.do(ctx, "GET", "serverless", deploymentPath(ids[0], ids[1], ids[2]), nil, nil)
	if err != nil {
		if isNotFound(err) {
			if err := notFoundOnImport("serverless deployment", req.ID, req.Inputs, req.State); err != nil {
				return resp{}, err
			}
			return resp{}, nil
		}
		return resp{}, fmt.Errorf("read serverless deployment %s: %w", req.ID, err)
	}
	in := req.Inputs
	if in.ServiceSid == "" {
		in = ServerlessDeploymentArgs{ServiceSid: ids[0], EnvironmentSid: ids[1]}
	}
	if v := o.str("build_sid"); v != "" {
		in.BuildSid = v
	}
	st := ServerlessDeploymentState{ServerlessDeploymentArgs: in}
	deploymentOutputs(&st, o)
	return resp{ID: req.ID, Inputs: in, State: st}, nil
}

func (r *ServerlessDeployment) Delete(_ context.Context, _ infer.DeleteRequest[ServerlessDeploymentState]) (infer.DeleteResponse, error) {
	// Twilio has no delete for deployments: the environment keeps serving the build until another deployment replaces it.
	return infer.DeleteResponse{}, nil
}
