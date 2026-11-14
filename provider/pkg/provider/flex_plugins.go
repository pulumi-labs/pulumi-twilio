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
	"context"
	"encoding/json"
	"fmt"
	"net/url"
	"regexp"

	p "github.com/pulumi/pulumi-go-provider"
	"github.com/pulumi/pulumi-go-provider/infer"
)

// The Flex Plugins service (flex-api.twilio.com v1 /PluginService): a Plugin is a named container,
// a Plugin Version is one immutable bundle of it, a Configuration is an immutable set of Plugin
// Versions and a Release makes one Configuration the one Flex loads. The shapes follow twilio-oai
// spec/json/twilio_flex_v1.json. The optional Flex-Metadata request header is not sent.

var (
	flexPluginSidRE        = regexp.MustCompile(`^FP[0-9a-fA-F]{32}$`)
	flexPluginVersionSidRE = regexp.MustCompile(`^FV[0-9a-fA-F]{32}$`)
)

func flexPluginPath(rest ...string) string {
	return flexPath("PluginService", rest...)
}

// flexArchive archives a Plugin, Plugin Version or Configuration: the only way Twilio retires them.
// One that is already gone is not an error.
func flexArchive(ctx context.Context, kind, id string, path ...string) error {
	c, err := clientFor(ctx)
	if err != nil {
		return err
	}
	path = append(path, "Archive")
	if _, err := c.do(ctx, "POST", flexProduct, flexPluginPath(path...), nil, url.Values{}); err != nil && !isNotFound(err) {
		return fmt.Errorf("archive %s %s: %w", kind, id, err)
	}
	return nil
}

// flexGone reads an object and reports whether it is missing or archived. An archived object is
// treated as deleted: it can no longer be used or changed.
func flexGone(ctx context.Context, path ...string) (obj, bool, error) {
	c, err := clientFor(ctx)
	if err != nil {
		return nil, false, err
	}
	o, err := c.do(ctx, "GET", flexProduct, flexPluginPath(path...), nil, nil)
	if err != nil {
		if isNotFound(err) {
			return nil, true, nil
		}
		return nil, false, err
	}
	if b := o.optBool("archived"); b != nil && *b {
		return nil, true, nil
	}
	return o, false, nil
}

func sameOrdered(a, b []string) bool {
	if len(a) != len(b) {
		return false
	}
	for i := range a {
		if a[i] != b[i] {
			return false
		}
	}
	return true
}

// ------------------------------------------------------------ FlexPlugin

// FlexPlugin manages a Flex Plugin, the named container of the versions of one Flex UI plugin.
type FlexPlugin struct{}

// FlexPluginArgs are the inputs of a FlexPlugin.
type FlexPluginArgs struct {
	UniqueName   string  `pulumi:"uniqueName"`
	FriendlyName *string `pulumi:"friendlyName,optional"`
	Description  *string `pulumi:"description,optional"`
}

func (a *FlexPluginArgs) Annotate(an infer.Annotator) {
	an.Describe(&a.UniqueName, "The unique name of the Flex Plugin, for example the name the Flex Plugins CLI uses. Changing it replaces the Flex Plugin.")
	an.Describe(&a.FriendlyName, "A descriptive name for the Flex Plugin. Removing it leaves the name unchanged.")
	an.Describe(&a.Description, "A description of the Flex Plugin. Removing it leaves the description unchanged.")
}

// FlexPluginState is the persisted state of a FlexPlugin.
type FlexPluginState struct {
	FlexPluginArgs

	Sid         string `pulumi:"sid"`
	DateCreated string `pulumi:"dateCreated"`
	DateUpdated string `pulumi:"dateUpdated"`
}

func (s *FlexPluginState) Annotate(a infer.Annotator) {
	a.Describe(&s.Sid, "The Flex Plugin SID (FP...).")
	a.Describe(&s.DateCreated, "When the Flex Plugin was created (ISO 8601).")
	a.Describe(&s.DateUpdated, "When the Flex Plugin was last updated (ISO 8601).")
}

func (r *FlexPlugin) Annotate(a infer.Annotator) {
	a.Describe(&r, "A Flex Plugin. The resource ID is the Flex Plugin SID. Destroying the resource archives the Flex Plugin (Twilio has no delete); an archived Flex Plugin is treated as deleted, but its unique name stays taken.")
}

var _ infer.CustomResource[FlexPluginArgs, FlexPluginState] = (*FlexPlugin)(nil)
var _ infer.CustomCheck[FlexPluginArgs] = (*FlexPlugin)(nil)
var _ infer.CustomDiff[FlexPluginArgs, FlexPluginState] = (*FlexPlugin)(nil)
var _ infer.CustomRead[FlexPluginArgs, FlexPluginState] = (*FlexPlugin)(nil)
var _ infer.CustomUpdate[FlexPluginArgs, FlexPluginState] = (*FlexPlugin)(nil)
var _ infer.CustomDelete[FlexPluginState] = (*FlexPlugin)(nil)

func (r *FlexPlugin) Check(ctx context.Context, req infer.CheckRequest) (infer.CheckResponse[FlexPluginArgs], error) {
	args, failures, err := infer.DefaultCheck[FlexPluginArgs](ctx, req.NewInputs)
	if err != nil || len(failures) > 0 {
		return infer.CheckResponse[FlexPluginArgs]{Inputs: args, Failures: failures}, err
	}
	c := &checker{unk: unknowns(req.NewInputs)}
	c.required("uniqueName", args.UniqueName)
	return infer.CheckResponse[FlexPluginArgs]{Inputs: args, Failures: c.failures}, nil
}

func (r *FlexPlugin) Diff(_ context.Context, req infer.DiffRequest[FlexPluginArgs, FlexPluginState]) (infer.DiffResponse, error) {
	o, n := req.State.FlexPluginArgs, req.Inputs
	d := trDiffSet{}
	d.replace("uniqueName", o.UniqueName, n.UniqueName)
	d.str("friendlyName", o.FriendlyName, n.FriendlyName)
	d.str("description", o.Description, n.Description)
	return d.response(), nil
}

func (a FlexPluginArgs) form(create bool) url.Values {
	form := url.Values{}
	if create {
		form.Set("UniqueName", a.UniqueName)
	}
	setS(form, "FriendlyName", a.FriendlyName)
	setS(form, "Description", a.Description)
	return form
}

func (s *FlexPluginState) fill(o obj) {
	s.Sid, s.DateCreated, s.DateUpdated = trSid(o)
}

func (r *FlexPlugin) Create(ctx context.Context, req infer.CreateRequest[FlexPluginArgs]) (infer.CreateResponse[FlexPluginState], error) {
	type resp = infer.CreateResponse[FlexPluginState]
	a := req.Inputs
	if req.DryRun {
		return resp{Output: FlexPluginState{FlexPluginArgs: a}}, nil
	}
	c, err := clientFor(ctx)
	if err != nil {
		return resp{}, err
	}
	o, err := c.do(ctx, "POST", flexProduct, flexPluginPath("Plugins"), nil, a.form(true))
	if err != nil {
		return resp{}, fmt.Errorf("create Flex Plugin: %w", err)
	}
	st := FlexPluginState{FlexPluginArgs: a}
	st.fill(o)
	return resp{ID: st.Sid, Output: st}, nil
}

func (r *FlexPlugin) Read(ctx context.Context, req infer.ReadRequest[FlexPluginArgs, FlexPluginState]) (infer.ReadResponse[FlexPluginArgs, FlexPluginState], error) {
	type resp = infer.ReadResponse[FlexPluginArgs, FlexPluginState]
	o, gone, err := flexGone(ctx, "Plugins", req.ID)
	if err != nil {
		return resp{}, fmt.Errorf("read Flex Plugin %s: %w", req.ID, err)
	}
	if gone {
		if err := notFoundOnImport("Flex Plugin", req.ID, req.Inputs, req.State); err != nil {
			return resp{}, err
		}
		return resp{}, nil
	}
	in := req.Inputs
	imported := req.State.Sid == ""
	if v := o.str("unique_name"); v != "" {
		in.UniqueName = v
	}
	in.FriendlyName = trStr(in.FriendlyName, o.optStr("friendly_name"), "", imported)
	in.Description = trStr(in.Description, o.optStr("description"), "", imported)
	st := FlexPluginState{FlexPluginArgs: in}
	st.fill(o)
	return resp{ID: req.ID, Inputs: in, State: st}, nil
}

func (r *FlexPlugin) Update(ctx context.Context, req infer.UpdateRequest[FlexPluginArgs, FlexPluginState]) (infer.UpdateResponse[FlexPluginState], error) {
	type resp = infer.UpdateResponse[FlexPluginState]
	st := req.State
	st.FlexPluginArgs = req.Inputs
	if req.DryRun {
		return resp{Output: st}, nil
	}
	c, err := clientFor(ctx)
	if err != nil {
		return resp{}, err
	}
	o, err := c.do(ctx, "POST", flexProduct, flexPluginPath("Plugins", req.ID), nil, req.Inputs.form(false))
	if err != nil {
		return resp{}, fmt.Errorf("update Flex Plugin %s: %w", req.ID, err)
	}
	st.fill(o)
	return resp{Output: st}, nil
}

func (r *FlexPlugin) Delete(ctx context.Context, req infer.DeleteRequest[FlexPluginState]) (infer.DeleteResponse, error) {
	return infer.DeleteResponse{}, flexArchive(ctx, "Flex Plugin", req.ID, "Plugins", req.ID)
}

// ------------------------------------------------------------ FlexPluginVersion

// FlexPluginVersion manages one Flex Plugin Version: an immutable bundle of a Flex Plugin.
type FlexPluginVersion struct{}

// FlexPluginVersionArgs are the inputs of a FlexPluginVersion. Twilio cannot change a Plugin
// Version, so every input replaces it.
type FlexPluginVersionArgs struct {
	PluginSid      string  `pulumi:"pluginSid"`
	PluginVersion  string  `pulumi:"pluginVersion"`
	PluginUrl      string  `pulumi:"pluginUrl"`
	Changelog      *string `pulumi:"changelog,optional"`
	Private        *bool   `pulumi:"private,optional"`
	CliVersion     *string `pulumi:"cliVersion,optional"`
	ValidateStatus *string `pulumi:"validateStatus,optional"`
}

func (a *FlexPluginVersionArgs) Annotate(an infer.Annotator) {
	an.Describe(&a.PluginSid, "The SID (FP...) of the Flex Plugin the version belongs to. Changing it replaces the version.")
	an.Describe(&a.PluginVersion, "The version label of this bundle, for example 1.0.0 (Twilio's Version parameter). Changing it replaces the version.")
	an.Describe(&a.PluginUrl, "The URL of the plugin bundle, typically a Twilio Asset URL. Changing it replaces the version.")
	an.Describe(&a.Changelog, "The changelog of this version. Changing it replaces the version.")
	an.Describe(&a.Private, "Whether the bundle requires authorization to load (Twilio default false). Changing it replaces the version.")
	an.Describe(&a.CliVersion, "The version of the Flex Plugins CLI that built the bundle. Twilio does not return it, so it is never refreshed. Changing it replaces the version.")
	an.Describe(&a.ValidateStatus, "The validation status of the bundle as the Flex Plugins CLI reports it. Twilio does not return it, so it is never refreshed. Changing it replaces the version.")
}

// FlexPluginVersionState is the persisted state of a FlexPluginVersion.
type FlexPluginVersionState struct {
	FlexPluginVersionArgs

	Sid         string `pulumi:"sid"`
	Validated   bool   `pulumi:"validated"`
	DateCreated string `pulumi:"dateCreated"`
}

func (s *FlexPluginVersionState) Annotate(a infer.Annotator) {
	a.Describe(&s.Sid, "The Flex Plugin Version SID (FV...).")
	a.Describe(&s.Validated, "Whether Twilio has validated the bundle.")
	a.Describe(&s.DateCreated, "When the version was created (ISO 8601).")
}

func (r *FlexPluginVersion) Annotate(a infer.Annotator) {
	a.Describe(&r, "A Flex Plugin Version. The resource ID is <pluginSid>/<sid>. Twilio cannot change a version, so any change replaces it. Destroying the resource archives the version (Twilio has no delete).")
}

var _ infer.CustomResource[FlexPluginVersionArgs, FlexPluginVersionState] = (*FlexPluginVersion)(nil)
var _ infer.CustomCheck[FlexPluginVersionArgs] = (*FlexPluginVersion)(nil)
var _ infer.CustomDiff[FlexPluginVersionArgs, FlexPluginVersionState] = (*FlexPluginVersion)(nil)
var _ infer.CustomRead[FlexPluginVersionArgs, FlexPluginVersionState] = (*FlexPluginVersion)(nil)
var _ infer.CustomDelete[FlexPluginVersionState] = (*FlexPluginVersion)(nil)

func (r *FlexPluginVersion) Check(ctx context.Context, req infer.CheckRequest) (infer.CheckResponse[FlexPluginVersionArgs], error) {
	args, failures, err := infer.DefaultCheck[FlexPluginVersionArgs](ctx, req.NewInputs)
	if err != nil || len(failures) > 0 {
		return infer.CheckResponse[FlexPluginVersionArgs]{Inputs: args, Failures: failures}, err
	}
	c := &checker{unk: unknowns(req.NewInputs)}
	c.sid("pluginSid", args.PluginSid, flexPluginSidRE, "Flex Plugin (FP followed by 32 hex characters)")
	c.required("pluginVersion", args.PluginVersion)
	c.required("pluginUrl", args.PluginUrl)
	return infer.CheckResponse[FlexPluginVersionArgs]{Inputs: args, Failures: c.failures}, nil
}

func (r *FlexPluginVersion) Diff(_ context.Context, req infer.DiffRequest[FlexPluginVersionArgs, FlexPluginVersionState]) (infer.DiffResponse, error) {
	o, n := req.State.FlexPluginVersionArgs, req.Inputs
	d := trDiffSet{}
	d.replace("pluginSid", o.PluginSid, n.PluginSid)
	d.replace("pluginVersion", o.PluginVersion, n.PluginVersion)
	d.replace("pluginUrl", o.PluginUrl, n.PluginUrl)
	d.replaceStr("changelog", o.Changelog, n.Changelog)
	d.replaceBool("private", flexDefBool(o.Private), flexDefBool(n.Private))
	d.replaceStr("cliVersion", o.CliVersion, n.CliVersion)
	d.replaceStr("validateStatus", o.ValidateStatus, n.ValidateStatus)
	return d.response(), nil
}

// flexDefBool treats an unset bool as Twilio's default, false, so stating the default is no change.
func flexDefBool(b *bool) *bool {
	if b == nil {
		f := false
		return &f
	}
	return b
}

func (s *FlexPluginVersionState) fill(o obj) {
	s.Sid = o.str("sid")
	s.DateCreated = o.str("date_created")
	s.Validated = o.optBool("validated") != nil && *o.optBool("validated")
}

func (r *FlexPluginVersion) Create(ctx context.Context, req infer.CreateRequest[FlexPluginVersionArgs]) (infer.CreateResponse[FlexPluginVersionState], error) {
	type resp = infer.CreateResponse[FlexPluginVersionState]
	a := req.Inputs
	if req.DryRun {
		return resp{Output: FlexPluginVersionState{FlexPluginVersionArgs: a}}, nil
	}
	c, err := clientFor(ctx)
	if err != nil {
		return resp{}, err
	}
	form := url.Values{}
	form.Set("Version", a.PluginVersion)
	form.Set("PluginUrl", a.PluginUrl)
	setS(form, "Changelog", a.Changelog)
	setB(form, "Private", a.Private)
	setS(form, "CliVersion", a.CliVersion)
	setS(form, "ValidateStatus", a.ValidateStatus)
	o, err := c.do(ctx, "POST", flexProduct, flexPluginPath("Plugins", a.PluginSid, "Versions"), nil, form)
	if err != nil {
		return resp{}, fmt.Errorf("create Flex Plugin Version of %s: %w", a.PluginSid, err)
	}
	st := FlexPluginVersionState{FlexPluginVersionArgs: a}
	st.fill(o)
	return resp{ID: a.PluginSid + "/" + st.Sid, Output: st}, nil
}

func (r *FlexPluginVersion) Read(ctx context.Context, req infer.ReadRequest[FlexPluginVersionArgs, FlexPluginVersionState]) (infer.ReadResponse[FlexPluginVersionArgs, FlexPluginVersionState], error) {
	type resp = infer.ReadResponse[FlexPluginVersionArgs, FlexPluginVersionState]
	parts, err := splitVerifyID(req.ID, 2, "<pluginSid>/<sid>")
	if err != nil {
		return resp{}, err
	}
	o, gone, err := flexGone(ctx, "Plugins", parts[0], "Versions", parts[1])
	if err != nil {
		return resp{}, fmt.Errorf("read Flex Plugin Version %s: %w", req.ID, err)
	}
	if gone {
		if err := notFoundOnImport("Flex Plugin Version", req.ID, req.Inputs, req.State); err != nil {
			return resp{}, err
		}
		return resp{}, nil
	}
	in := req.Inputs
	imported := req.State.Sid == ""
	in.PluginSid = parts[0]
	if v := o.str("version"); v != "" {
		in.PluginVersion = v
	}
	if v := o.str("plugin_url"); v != "" {
		in.PluginUrl = v
	}
	in.Changelog = trStr(in.Changelog, o.optStr("changelog"), "", imported)
	in.Private = trBool(in.Private, o.optBool("private"), false, imported)
	// cliVersion and validateStatus are write-only: Twilio never returns them.
	st := FlexPluginVersionState{FlexPluginVersionArgs: in}
	st.fill(o)
	return resp{ID: req.ID, Inputs: in, State: st}, nil
}

func (r *FlexPluginVersion) Delete(ctx context.Context, req infer.DeleteRequest[FlexPluginVersionState]) (infer.DeleteResponse, error) {
	parts, err := splitVerifyID(req.ID, 2, "<pluginSid>/<sid>")
	if err != nil {
		return infer.DeleteResponse{}, err
	}
	return infer.DeleteResponse{}, flexArchive(ctx, "Flex Plugin Version", req.ID, "Plugins", parts[0], "Versions", parts[1])
}

// ------------------------------------------------------------ FlexPluginConfiguration

// FlexPluginConfiguration manages a Flex Plugin Configuration: an immutable, named set of Flex
// Plugin Versions that a Flex Plugin Release can make live.
type FlexPluginConfiguration struct{}

// FlexPluginConfigurationArgs are the inputs of a FlexPluginConfiguration. Twilio cannot change
// a Configuration, so every input replaces it.
type FlexPluginConfigurationArgs struct {
	Name              string    `pulumi:"name"`
	Description       *string   `pulumi:"description,optional"`
	PluginVersionSids *[]string `pulumi:"pluginVersionSids,optional"`
}

func (a *FlexPluginConfigurationArgs) Annotate(an infer.Annotator) {
	an.Describe(&a.Name, "The name of the Flex Plugin Configuration. Changing it replaces the Configuration.")
	an.Describe(&a.Description, "A description of the Configuration. Changing it replaces the Configuration.")
	an.Describe(&a.PluginVersionSids, "The SIDs (FV...) of the Flex Plugin Versions the Configuration loads, in order. Changing the list replaces the Configuration.")
}

// FlexPluginConfigurationState is the persisted state of a FlexPluginConfiguration.
type FlexPluginConfigurationState struct {
	FlexPluginConfigurationArgs

	Sid         string `pulumi:"sid"`
	DateCreated string `pulumi:"dateCreated"`
}

func (s *FlexPluginConfigurationState) Annotate(a infer.Annotator) {
	a.Describe(&s.Sid, "The Flex Plugin Configuration SID (FJ...).")
	a.Describe(&s.DateCreated, "When the Configuration was created (ISO 8601).")
}

func (r *FlexPluginConfiguration) Annotate(a infer.Annotator) {
	a.Describe(&r, "A Flex Plugin Configuration. The resource ID is the Configuration SID. Twilio cannot change a Configuration, so any change replaces it. Destroying the resource archives the Configuration (Twilio has no delete); Twilio may refuse to archive the Configuration that is currently released.")
}

var _ infer.CustomResource[FlexPluginConfigurationArgs, FlexPluginConfigurationState] = (*FlexPluginConfiguration)(nil)
var _ infer.CustomCheck[FlexPluginConfigurationArgs] = (*FlexPluginConfiguration)(nil)
var _ infer.CustomDiff[FlexPluginConfigurationArgs, FlexPluginConfigurationState] = (*FlexPluginConfiguration)(nil)
var _ infer.CustomRead[FlexPluginConfigurationArgs, FlexPluginConfigurationState] = (*FlexPluginConfiguration)(nil)
var _ infer.CustomDelete[FlexPluginConfigurationState] = (*FlexPluginConfiguration)(nil)

func (r *FlexPluginConfiguration) Check(ctx context.Context, req infer.CheckRequest) (infer.CheckResponse[FlexPluginConfigurationArgs], error) {
	args, failures, err := infer.DefaultCheck[FlexPluginConfigurationArgs](ctx, req.NewInputs)
	if err != nil || len(failures) > 0 {
		return infer.CheckResponse[FlexPluginConfigurationArgs]{Inputs: args, Failures: failures}, err
	}
	c := &checker{unk: unknowns(req.NewInputs)}
	c.required("name", args.Name)
	if args.PluginVersionSids != nil && !c.unk("pluginVersionSids") {
		for _, s := range *args.PluginVersionSids {
			c.sid("pluginVersionSids", s, flexPluginVersionSidRE, "Flex Plugin Version (FV followed by 32 hex characters)")
		}
	}
	return infer.CheckResponse[FlexPluginConfigurationArgs]{Inputs: args, Failures: c.failures}, nil
}

func (r *FlexPluginConfiguration) Diff(_ context.Context, req infer.DiffRequest[FlexPluginConfigurationArgs, FlexPluginConfigurationState]) (infer.DiffResponse, error) {
	o, n := req.State.FlexPluginConfigurationArgs, req.Inputs
	d := trDiffSet{}
	d.replace("name", o.Name, n.Name)
	d.replaceStr("description", o.Description, n.Description)
	if n.PluginVersionSids != nil {
		var old []string
		if o.PluginVersionSids != nil {
			old = *o.PluginVersionSids
		}
		if !sameOrdered(old, *n.PluginVersionSids) {
			d["pluginVersionSids"] = p.PropertyDiff{Kind: p.UpdateReplace}
		}
	}
	return d.response(), nil
}

// flexPluginsParam encodes the plugin list the way Twilio takes it: one Plugins parameter per
// version, each a JSON object that names the Plugin Version SID.
func flexPluginsParam(sids []string) []string {
	out := make([]string, 0, len(sids))
	for _, s := range sids {
		b, _ := json.Marshal(map[string]string{"plugin_version": s})
		out = append(out, string(b))
	}
	return out
}

func (s *FlexPluginConfigurationState) fill(o obj) {
	s.Sid = o.str("sid")
	s.DateCreated = o.str("date_created")
}

func (r *FlexPluginConfiguration) Create(ctx context.Context, req infer.CreateRequest[FlexPluginConfigurationArgs]) (infer.CreateResponse[FlexPluginConfigurationState], error) {
	type resp = infer.CreateResponse[FlexPluginConfigurationState]
	a := req.Inputs
	if req.DryRun {
		return resp{Output: FlexPluginConfigurationState{FlexPluginConfigurationArgs: a}}, nil
	}
	c, err := clientFor(ctx)
	if err != nil {
		return resp{}, err
	}
	form := url.Values{}
	form.Set("Name", a.Name)
	setS(form, "Description", a.Description)
	if a.PluginVersionSids != nil {
		for _, v := range flexPluginsParam(*a.PluginVersionSids) {
			form.Add("Plugins", v)
		}
	}
	o, err := c.do(ctx, "POST", flexProduct, flexPluginPath("Configurations"), nil, form)
	if err != nil {
		return resp{}, fmt.Errorf("create Flex Plugin Configuration: %w", err)
	}
	st := FlexPluginConfigurationState{FlexPluginConfigurationArgs: a}
	st.fill(o)
	return resp{ID: st.Sid, Output: st}, nil
}

func (r *FlexPluginConfiguration) Read(ctx context.Context, req infer.ReadRequest[FlexPluginConfigurationArgs, FlexPluginConfigurationState]) (infer.ReadResponse[FlexPluginConfigurationArgs, FlexPluginConfigurationState], error) {
	type resp = infer.ReadResponse[FlexPluginConfigurationArgs, FlexPluginConfigurationState]
	o, gone, err := flexGone(ctx, "Configurations", req.ID)
	if err != nil {
		return resp{}, fmt.Errorf("read Flex Plugin Configuration %s: %w", req.ID, err)
	}
	if gone {
		if err := notFoundOnImport("Flex Plugin Configuration", req.ID, req.Inputs, req.State); err != nil {
			return resp{}, err
		}
		return resp{}, nil
	}
	c, err := clientFor(ctx)
	if err != nil {
		return resp{}, err
	}
	sids, err := flexConfiguredVersions(ctx, c, "read Flex Plugin Configuration", req.ID)
	if err != nil {
		return resp{}, err
	}
	in := req.Inputs
	imported := req.State.Sid == ""
	if v := o.str("name"); v != "" {
		in.Name = v
	}
	in.Description = trStr(in.Description, o.optStr("description"), "", imported)
	if imported || in.PluginVersionSids != nil {
		in.PluginVersionSids = nonEmptyList(&sids)
		if in.PluginVersionSids == nil && !imported {
			empty := []string{}
			in.PluginVersionSids = &empty
		}
	}
	st := FlexPluginConfigurationState{FlexPluginConfigurationArgs: in}
	st.fill(o)
	return resp{ID: req.ID, Inputs: in, State: st}, nil
}

// flexConfiguredVersions lists the Plugin Version SIDs of a Configuration in Twilio's order.
func flexConfiguredVersions(ctx context.Context, c *Client, fn, id string) ([]string, error) {
	sids := []string{}
	err := listPages(ctx, c, fn, flexProduct, flexPluginPath("Configurations", id, "Plugins"), url.Values{"PageSize": {"1000"}}, "plugins", func(o obj) bool {
		sids = append(sids, o.str("plugin_version_sid"))
		return true
	})
	if err != nil {
		return nil, fmt.Errorf("%s %s: list its plugins: %w", fn, id, err)
	}
	return sids, nil
}

func (r *FlexPluginConfiguration) Delete(ctx context.Context, req infer.DeleteRequest[FlexPluginConfigurationState]) (infer.DeleteResponse, error) {
	return infer.DeleteResponse{}, flexArchive(ctx, "Flex Plugin Configuration", req.ID, "Configurations", req.ID)
}

// ------------------------------------------------------------ FlexPluginRelease

// FlexPluginRelease manages a Flex Plugin Release: the act of making one Flex Plugin
// Configuration the one Flex loads.
type FlexPluginRelease struct{}

// FlexPluginReleaseArgs are the inputs of a FlexPluginRelease.
type FlexPluginReleaseArgs struct {
	ConfigurationId string `pulumi:"configurationId"`
}

func (a *FlexPluginReleaseArgs) Annotate(an infer.Annotator) {
	an.Describe(&a.ConfigurationId, "The SID (FJ...) of the Flex Plugin Configuration to release. Changing it releases the new Configuration (a new Release).")
}

// FlexPluginReleaseState is the persisted state of a FlexPluginRelease.
type FlexPluginReleaseState struct {
	FlexPluginReleaseArgs

	Sid              string `pulumi:"sid"`
	ConfigurationSid string `pulumi:"configurationSid"`
	DateCreated      string `pulumi:"dateCreated"`
}

func (s *FlexPluginReleaseState) Annotate(a infer.Annotator) {
	a.Describe(&s.Sid, "The Flex Plugin Release SID (FK...).")
	a.Describe(&s.ConfigurationSid, "The SID (FJ...) of the Configuration the Release made live.")
	a.Describe(&s.DateCreated, "When the Release was created (ISO 8601).")
}

func (r *FlexPluginRelease) Annotate(a infer.Annotator) {
	a.Describe(&r, "A Flex Plugin Release, which makes a Flex Plugin Configuration the one every Flex user loads. The resource ID is the Release SID. A Release is a record: Twilio can neither change nor delete it, so destroying the resource only forgets it and the Configuration stays live until another Release replaces it.")
}

var _ infer.CustomResource[FlexPluginReleaseArgs, FlexPluginReleaseState] = (*FlexPluginRelease)(nil)
var _ infer.CustomCheck[FlexPluginReleaseArgs] = (*FlexPluginRelease)(nil)
var _ infer.CustomDiff[FlexPluginReleaseArgs, FlexPluginReleaseState] = (*FlexPluginRelease)(nil)
var _ infer.CustomRead[FlexPluginReleaseArgs, FlexPluginReleaseState] = (*FlexPluginRelease)(nil)
var _ infer.CustomDelete[FlexPluginReleaseState] = (*FlexPluginRelease)(nil)

func (r *FlexPluginRelease) Check(ctx context.Context, req infer.CheckRequest) (infer.CheckResponse[FlexPluginReleaseArgs], error) {
	args, failures, err := infer.DefaultCheck[FlexPluginReleaseArgs](ctx, req.NewInputs)
	if err != nil || len(failures) > 0 {
		return infer.CheckResponse[FlexPluginReleaseArgs]{Inputs: args, Failures: failures}, err
	}
	c := &checker{unk: unknowns(req.NewInputs)}
	c.required("configurationId", args.ConfigurationId)
	return infer.CheckResponse[FlexPluginReleaseArgs]{Inputs: args, Failures: c.failures}, nil
}

func (r *FlexPluginRelease) Diff(_ context.Context, req infer.DiffRequest[FlexPluginReleaseArgs, FlexPluginReleaseState]) (infer.DiffResponse, error) {
	n := req.Inputs.ConfigurationId
	d := trDiffSet{}
	// Twilio also takes a Configuration version, so the SID Twilio reports is as good as the input.
	if n != req.State.ConfigurationId && n != req.State.ConfigurationSid {
		d["configurationId"] = p.PropertyDiff{Kind: p.UpdateReplace}
	}
	return d.response(), nil
}

func (s *FlexPluginReleaseState) fill(o obj) {
	s.Sid = o.str("sid")
	s.ConfigurationSid = o.str("configuration_sid")
	s.DateCreated = o.str("date_created")
}

func (r *FlexPluginRelease) Create(ctx context.Context, req infer.CreateRequest[FlexPluginReleaseArgs]) (infer.CreateResponse[FlexPluginReleaseState], error) {
	type resp = infer.CreateResponse[FlexPluginReleaseState]
	a := req.Inputs
	if req.DryRun {
		return resp{Output: FlexPluginReleaseState{FlexPluginReleaseArgs: a}}, nil
	}
	c, err := clientFor(ctx)
	if err != nil {
		return resp{}, err
	}
	o, err := c.do(ctx, "POST", flexProduct, flexPluginPath("Releases"), nil, url.Values{"ConfigurationId": {a.ConfigurationId}})
	if err != nil {
		return resp{}, fmt.Errorf("release Flex Plugin Configuration %s: %w", a.ConfigurationId, err)
	}
	st := FlexPluginReleaseState{FlexPluginReleaseArgs: a}
	st.fill(o)
	return resp{ID: st.Sid, Output: st}, nil
}

func (r *FlexPluginRelease) Read(ctx context.Context, req infer.ReadRequest[FlexPluginReleaseArgs, FlexPluginReleaseState]) (infer.ReadResponse[FlexPluginReleaseArgs, FlexPluginReleaseState], error) {
	type resp = infer.ReadResponse[FlexPluginReleaseArgs, FlexPluginReleaseState]
	c, err := clientFor(ctx)
	if err != nil {
		return resp{}, err
	}
	o, err := c.do(ctx, "GET", flexProduct, flexPluginPath("Releases", req.ID), nil, nil)
	if err != nil {
		if isNotFound(err) {
			if err := notFoundOnImport("Flex Plugin Release", req.ID, req.Inputs, req.State); err != nil {
				return resp{}, err
			}
			return resp{}, nil
		}
		return resp{}, fmt.Errorf("read Flex Plugin Release %s: %w", req.ID, err)
	}
	in := req.Inputs
	st := FlexPluginReleaseState{FlexPluginReleaseArgs: in}
	st.fill(o)
	if in.ConfigurationId == "" {
		in.ConfigurationId = st.ConfigurationSid
		st.ConfigurationId = st.ConfigurationSid
	}
	return resp{ID: req.ID, Inputs: in, State: st}, nil
}

// Delete forgets the Release: Twilio has no DELETE for one.
func (r *FlexPluginRelease) Delete(_ context.Context, _ infer.DeleteRequest[FlexPluginReleaseState]) (infer.DeleteResponse, error) {
	return infer.DeleteResponse{}, nil
}
