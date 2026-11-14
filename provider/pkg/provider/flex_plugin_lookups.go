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
	"net/url"

	"github.com/pulumi/pulumi-go-provider/infer"
)

// Lookups of the Flex Plugins service (flex-api.twilio.com /v1/PluginService). A list returns
// what Twilio returns, archived objects included; each item says whether it is archived.

// ------------------------------------------------------------ plugins

// GetFlexPlugin looks up a Flex Plugin by SID.
type GetFlexPlugin struct{}

// GetFlexPluginArgs are the lookup arguments.
type GetFlexPluginArgs struct {
	Sid string `pulumi:"sid"`
}

func (a *GetFlexPluginArgs) Annotate(an infer.Annotator) {
	an.Describe(&a.Sid, "SID (FP...) of the Flex Plugin.")
}

// FlexPluginInfo is one Flex Plugin as Twilio reports it.
type FlexPluginInfo struct {
	Sid          string  `pulumi:"sid"`
	UniqueName   string  `pulumi:"uniqueName"`
	FriendlyName *string `pulumi:"friendlyName,optional"`
	Description  *string `pulumi:"description,optional"`
	Archived     bool    `pulumi:"archived"`
	DateCreated  string  `pulumi:"dateCreated"`
	DateUpdated  string  `pulumi:"dateUpdated"`
}

func (a *FlexPluginInfo) Annotate(an infer.Annotator) {
	an.Describe(&a.Sid, "The Flex Plugin SID (FP...).")
	an.Describe(&a.UniqueName, "The unique name of the Flex Plugin.")
	an.Describe(&a.FriendlyName, "The descriptive name of the Flex Plugin.")
	an.Describe(&a.Description, "The description of the Flex Plugin.")
	an.Describe(&a.Archived, "Whether the Flex Plugin is archived.")
	an.Describe(&a.DateCreated, "When the Flex Plugin was created (ISO 8601).")
	an.Describe(&a.DateUpdated, "When the Flex Plugin was last updated (ISO 8601).")
}

func flexPluginInfo(o obj) FlexPluginInfo {
	sid, created, updated := trSid(o)
	return FlexPluginInfo{Sid: sid, UniqueName: o.str("unique_name"), FriendlyName: nonEmpty(o.optStr("friendly_name")),
		Description: nonEmpty(o.optStr("description")), Archived: flexArchived(o), DateCreated: created, DateUpdated: updated}
}

func flexArchived(o obj) bool {
	b := o.optBool("archived")
	return b != nil && *b
}

func (f *GetFlexPlugin) Annotate(a infer.Annotator) {
	a.Describe(&f, "Look up a Flex Plugin by SID (GET /v1/PluginService/Plugins/{Sid} on flex-api.twilio.com).")
}

func (f *GetFlexPlugin) Invoke(ctx context.Context, req infer.FunctionRequest[GetFlexPluginArgs]) (infer.FunctionResponse[FlexPluginInfo], error) {
	const fn = "getFlexPlugin"
	type resp = infer.FunctionResponse[FlexPluginInfo]
	if err := requireArg(fn, "sid", req.Input.Sid); err != nil {
		return resp{}, err
	}
	c, err := clientFor(ctx)
	if err != nil {
		return resp{}, err
	}
	o, err := c.do(ctx, "GET", flexProduct, flexPluginPath("Plugins", req.Input.Sid), nil, nil)
	if err != nil {
		return resp{}, lookupFailure(fn, "Flex Plugin "+req.Input.Sid, err)
	}
	return resp{Output: flexPluginInfo(o)}, nil
}

// GetFlexPlugins lists the Flex Plugins of the account.
type GetFlexPlugins struct{}

// GetFlexPluginsArgs are the lookup arguments (none).
type GetFlexPluginsArgs struct{}

// GetFlexPluginsResult is the list.
type GetFlexPluginsResult struct {
	FlexPlugins []FlexPluginInfo `pulumi:"flexPlugins"`
}

func (a *GetFlexPluginsResult) Annotate(an infer.Annotator) {
	an.Describe(&a.FlexPlugins, "The Flex Plugins, in the order Twilio returns them.")
}

func (f *GetFlexPlugins) Annotate(a infer.Annotator) {
	a.Describe(&f, "List the Flex Plugins of the account (GET /v1/PluginService/Plugins on flex-api.twilio.com).")
}

func (f *GetFlexPlugins) Invoke(ctx context.Context, _ infer.FunctionRequest[GetFlexPluginsArgs]) (infer.FunctionResponse[GetFlexPluginsResult], error) {
	const fn = "getFlexPlugins"
	none := infer.FunctionResponse[GetFlexPluginsResult]{}
	c, err := clientFor(ctx)
	if err != nil {
		return none, err
	}
	out := []FlexPluginInfo{}
	err = listPages(ctx, c, fn, flexProduct, flexPluginPath("Plugins"), url.Values{"PageSize": {"1000"}}, "plugins", func(o obj) bool {
		out = append(out, flexPluginInfo(o))
		return true
	})
	if err != nil {
		return none, lookupFailure(fn, "Flex Plugins", err)
	}
	return infer.FunctionResponse[GetFlexPluginsResult]{Output: GetFlexPluginsResult{FlexPlugins: out}}, nil
}

// ------------------------------------------------------------ plugin versions

// GetFlexPluginVersion looks up a Flex Plugin Version.
type GetFlexPluginVersion struct{}

// GetFlexPluginVersionArgs are the lookup arguments.
type GetFlexPluginVersionArgs struct {
	PluginSid string `pulumi:"pluginSid"`
	Sid       string `pulumi:"sid"`
}

func (a *GetFlexPluginVersionArgs) Annotate(an infer.Annotator) {
	an.Describe(&a.PluginSid, "SID (FP...) of the Flex Plugin.")
	an.Describe(&a.Sid, "SID (FV...) of the Flex Plugin Version.")
}

// FlexPluginVersionInfo is one Flex Plugin Version as Twilio reports it.
type FlexPluginVersionInfo struct {
	Sid           string  `pulumi:"sid"`
	PluginSid     string  `pulumi:"pluginSid"`
	PluginVersion string  `pulumi:"pluginVersion"`
	PluginUrl     string  `pulumi:"pluginUrl"`
	Changelog     *string `pulumi:"changelog,optional"`
	Private       *bool   `pulumi:"private,optional"`
	Validated     *bool   `pulumi:"validated,optional"`
	Archived      bool    `pulumi:"archived"`
	DateCreated   string  `pulumi:"dateCreated"`
}

func (a *FlexPluginVersionInfo) Annotate(an infer.Annotator) {
	an.Describe(&a.Sid, "The Flex Plugin Version SID (FV...).")
	an.Describe(&a.PluginSid, "The SID (FP...) of the Flex Plugin.")
	an.Describe(&a.PluginVersion, "The version label of the bundle.")
	an.Describe(&a.PluginUrl, "The URL of the plugin bundle.")
	an.Describe(&a.Changelog, "The changelog of the version.")
	an.Describe(&a.Private, "Whether the bundle requires authorization to load.")
	an.Describe(&a.Validated, "Whether Twilio has validated the bundle.")
	an.Describe(&a.Archived, "Whether the version is archived.")
	an.Describe(&a.DateCreated, "When the version was created (ISO 8601).")
}

func flexPluginVersionInfo(o obj) FlexPluginVersionInfo {
	return FlexPluginVersionInfo{Sid: o.str("sid"), PluginSid: o.str("plugin_sid"), PluginVersion: o.str("version"), PluginUrl: o.str("plugin_url"),
		Changelog: nonEmpty(o.optStr("changelog")), Private: o.optBool("private"), Validated: o.optBool("validated"),
		Archived: flexArchived(o), DateCreated: o.str("date_created")}
}

func (f *GetFlexPluginVersion) Annotate(a infer.Annotator) {
	a.Describe(&f, "Look up a Flex Plugin Version (GET /v1/PluginService/Plugins/{PluginSid}/Versions/{Sid} on flex-api.twilio.com).")
}

func (f *GetFlexPluginVersion) Invoke(ctx context.Context, req infer.FunctionRequest[GetFlexPluginVersionArgs]) (infer.FunctionResponse[FlexPluginVersionInfo], error) {
	const fn = "getFlexPluginVersion"
	type resp = infer.FunctionResponse[FlexPluginVersionInfo]
	if err := requireArgs(fn, [2]string{"pluginSid", req.Input.PluginSid}, [2]string{"sid", req.Input.Sid}); err != nil {
		return resp{}, err
	}
	c, err := clientFor(ctx)
	if err != nil {
		return resp{}, err
	}
	o, err := c.do(ctx, "GET", flexProduct, flexPluginPath("Plugins", req.Input.PluginSid, "Versions", req.Input.Sid), nil, nil)
	if err != nil {
		return resp{}, lookupFailure(fn, "Flex Plugin Version "+req.Input.Sid, err)
	}
	return resp{Output: flexPluginVersionInfo(o)}, nil
}

// GetFlexPluginVersions lists the versions of a Flex Plugin.
type GetFlexPluginVersions struct{}

// GetFlexPluginVersionsArgs are the lookup arguments.
type GetFlexPluginVersionsArgs struct {
	PluginSid string `pulumi:"pluginSid"`
}

func (a *GetFlexPluginVersionsArgs) Annotate(an infer.Annotator) {
	an.Describe(&a.PluginSid, "SID (FP...) of the Flex Plugin.")
}

// GetFlexPluginVersionsResult is the list.
type GetFlexPluginVersionsResult struct {
	FlexPluginVersions []FlexPluginVersionInfo `pulumi:"flexPluginVersions"`
}

func (a *GetFlexPluginVersionsResult) Annotate(an infer.Annotator) {
	an.Describe(&a.FlexPluginVersions, "The versions of the Flex Plugin, in the order Twilio returns them.")
}

func (f *GetFlexPluginVersions) Annotate(a infer.Annotator) {
	a.Describe(&f, "List the versions of a Flex Plugin (GET /v1/PluginService/Plugins/{PluginSid}/Versions on flex-api.twilio.com).")
}

func (f *GetFlexPluginVersions) Invoke(ctx context.Context, req infer.FunctionRequest[GetFlexPluginVersionsArgs]) (infer.FunctionResponse[GetFlexPluginVersionsResult], error) {
	const fn = "getFlexPluginVersions"
	none := infer.FunctionResponse[GetFlexPluginVersionsResult]{}
	if err := requireArg(fn, "pluginSid", req.Input.PluginSid); err != nil {
		return none, err
	}
	c, err := clientFor(ctx)
	if err != nil {
		return none, err
	}
	out := []FlexPluginVersionInfo{}
	err = listPages(ctx, c, fn, flexProduct, flexPluginPath("Plugins", req.Input.PluginSid, "Versions"), url.Values{"PageSize": {"1000"}}, "plugin_versions", func(o obj) bool {
		out = append(out, flexPluginVersionInfo(o))
		return true
	})
	if err != nil {
		return none, lookupFailure(fn, "Flex Plugin Versions of "+req.Input.PluginSid, err)
	}
	return infer.FunctionResponse[GetFlexPluginVersionsResult]{Output: GetFlexPluginVersionsResult{FlexPluginVersions: out}}, nil
}

// ------------------------------------------------------------ configurations

// GetFlexPluginConfiguration looks up a Flex Plugin Configuration with its plugin versions.
type GetFlexPluginConfiguration struct{}

// GetFlexPluginConfigurationArgs are the lookup arguments.
type GetFlexPluginConfigurationArgs struct {
	Sid string `pulumi:"sid"`
}

func (a *GetFlexPluginConfigurationArgs) Annotate(an infer.Annotator) {
	an.Describe(&a.Sid, "SID (FJ...) of the Flex Plugin Configuration.")
}

// FlexPluginConfigurationInfo is one Flex Plugin Configuration as Twilio reports it.
type FlexPluginConfigurationInfo struct {
	Sid               string    `pulumi:"sid"`
	Name              string    `pulumi:"name"`
	Description       *string   `pulumi:"description,optional"`
	Archived          bool      `pulumi:"archived"`
	DateCreated       string    `pulumi:"dateCreated"`
	PluginVersionSids *[]string `pulumi:"pluginVersionSids,optional"`
}

func (a *FlexPluginConfigurationInfo) Annotate(an infer.Annotator) {
	an.Describe(&a.Sid, "The Configuration SID (FJ...).")
	an.Describe(&a.Name, "The name of the Configuration.")
	an.Describe(&a.Description, "The description of the Configuration.")
	an.Describe(&a.Archived, "Whether the Configuration is archived.")
	an.Describe(&a.DateCreated, "When the Configuration was created (ISO 8601).")
	an.Describe(&a.PluginVersionSids, "The SIDs (FV...) of the Plugin Versions the Configuration loads, in order. Only the single lookup fills it.")
}

func flexPluginConfigurationInfo(o obj) FlexPluginConfigurationInfo {
	return FlexPluginConfigurationInfo{Sid: o.str("sid"), Name: o.str("name"), Description: nonEmpty(o.optStr("description")),
		Archived: flexArchived(o), DateCreated: o.str("date_created")}
}

func (f *GetFlexPluginConfiguration) Annotate(a infer.Annotator) {
	a.Describe(&f, "Look up a Flex Plugin Configuration and the Plugin Versions it loads (GET /v1/PluginService/Configurations/{Sid} and its /Plugins on flex-api.twilio.com).")
}

func (f *GetFlexPluginConfiguration) Invoke(ctx context.Context, req infer.FunctionRequest[GetFlexPluginConfigurationArgs]) (infer.FunctionResponse[FlexPluginConfigurationInfo], error) {
	const fn = "getFlexPluginConfiguration"
	type resp = infer.FunctionResponse[FlexPluginConfigurationInfo]
	if err := requireArg(fn, "sid", req.Input.Sid); err != nil {
		return resp{}, err
	}
	c, err := clientFor(ctx)
	if err != nil {
		return resp{}, err
	}
	o, err := c.do(ctx, "GET", flexProduct, flexPluginPath("Configurations", req.Input.Sid), nil, nil)
	if err != nil {
		return resp{}, lookupFailure(fn, "Flex Plugin Configuration "+req.Input.Sid, err)
	}
	sids, err := flexConfiguredVersions(ctx, c, fn, req.Input.Sid)
	if err != nil {
		return resp{}, lookupFailure(fn, "Flex Plugin Configuration "+req.Input.Sid, err)
	}
	out := flexPluginConfigurationInfo(o)
	out.PluginVersionSids = &sids
	return resp{Output: out}, nil
}

// GetFlexPluginConfigurations lists the Flex Plugin Configurations of the account.
type GetFlexPluginConfigurations struct{}

// GetFlexPluginConfigurationsArgs are the lookup arguments (none).
type GetFlexPluginConfigurationsArgs struct{}

// GetFlexPluginConfigurationsResult is the list.
type GetFlexPluginConfigurationsResult struct {
	FlexPluginConfigurations []FlexPluginConfigurationInfo `pulumi:"flexPluginConfigurations"`
}

func (a *GetFlexPluginConfigurationsResult) Annotate(an infer.Annotator) {
	an.Describe(&a.FlexPluginConfigurations, "The Configurations, in the order Twilio returns them. Their pluginVersionSids are not filled; use getFlexPluginConfiguration for one.")
}

func (f *GetFlexPluginConfigurations) Annotate(a infer.Annotator) {
	a.Describe(&f, "List the Flex Plugin Configurations of the account (GET /v1/PluginService/Configurations on flex-api.twilio.com).")
}

func (f *GetFlexPluginConfigurations) Invoke(ctx context.Context, _ infer.FunctionRequest[GetFlexPluginConfigurationsArgs]) (infer.FunctionResponse[GetFlexPluginConfigurationsResult], error) {
	const fn = "getFlexPluginConfigurations"
	none := infer.FunctionResponse[GetFlexPluginConfigurationsResult]{}
	c, err := clientFor(ctx)
	if err != nil {
		return none, err
	}
	out := []FlexPluginConfigurationInfo{}
	err = listPages(ctx, c, fn, flexProduct, flexPluginPath("Configurations"), url.Values{"PageSize": {"1000"}}, "configurations", func(o obj) bool {
		out = append(out, flexPluginConfigurationInfo(o))
		return true
	})
	if err != nil {
		return none, lookupFailure(fn, "Flex Plugin Configurations", err)
	}
	return infer.FunctionResponse[GetFlexPluginConfigurationsResult]{Output: GetFlexPluginConfigurationsResult{FlexPluginConfigurations: out}}, nil
}

// ------------------------------------------------------------ releases

// GetFlexPluginRelease looks up a Flex Plugin Release.
type GetFlexPluginRelease struct{}

// GetFlexPluginReleaseArgs are the lookup arguments.
type GetFlexPluginReleaseArgs struct {
	Sid string `pulumi:"sid"`
}

func (a *GetFlexPluginReleaseArgs) Annotate(an infer.Annotator) {
	an.Describe(&a.Sid, "SID (FK...) of the Flex Plugin Release.")
}

// FlexPluginReleaseInfo is one Flex Plugin Release as Twilio reports it.
type FlexPluginReleaseInfo struct {
	Sid              string `pulumi:"sid"`
	ConfigurationSid string `pulumi:"configurationSid"`
	DateCreated      string `pulumi:"dateCreated"`
}

func (a *FlexPluginReleaseInfo) Annotate(an infer.Annotator) {
	an.Describe(&a.Sid, "The Release SID (FK...).")
	an.Describe(&a.ConfigurationSid, "The SID (FJ...) of the Configuration the Release made live.")
	an.Describe(&a.DateCreated, "When the Release was created (ISO 8601).")
}

func flexPluginReleaseInfo(o obj) FlexPluginReleaseInfo {
	return FlexPluginReleaseInfo{Sid: o.str("sid"), ConfigurationSid: o.str("configuration_sid"), DateCreated: o.str("date_created")}
}

func (f *GetFlexPluginRelease) Annotate(a infer.Annotator) {
	a.Describe(&f, "Look up a Flex Plugin Release by SID (GET /v1/PluginService/Releases/{Sid} on flex-api.twilio.com).")
}

func (f *GetFlexPluginRelease) Invoke(ctx context.Context, req infer.FunctionRequest[GetFlexPluginReleaseArgs]) (infer.FunctionResponse[FlexPluginReleaseInfo], error) {
	const fn = "getFlexPluginRelease"
	type resp = infer.FunctionResponse[FlexPluginReleaseInfo]
	if err := requireArg(fn, "sid", req.Input.Sid); err != nil {
		return resp{}, err
	}
	c, err := clientFor(ctx)
	if err != nil {
		return resp{}, err
	}
	o, err := c.do(ctx, "GET", flexProduct, flexPluginPath("Releases", req.Input.Sid), nil, nil)
	if err != nil {
		return resp{}, lookupFailure(fn, "Flex Plugin Release "+req.Input.Sid, err)
	}
	return resp{Output: flexPluginReleaseInfo(o)}, nil
}

// GetFlexPluginReleases lists the Flex Plugin Releases of the account.
type GetFlexPluginReleases struct{}

// GetFlexPluginReleasesArgs are the lookup arguments (none).
type GetFlexPluginReleasesArgs struct{}

// GetFlexPluginReleasesResult is the list.
type GetFlexPluginReleasesResult struct {
	FlexPluginReleases []FlexPluginReleaseInfo `pulumi:"flexPluginReleases"`
}

func (a *GetFlexPluginReleasesResult) Annotate(an infer.Annotator) {
	an.Describe(&a.FlexPluginReleases, "The Releases, in the order Twilio returns them.")
}

func (f *GetFlexPluginReleases) Annotate(a infer.Annotator) {
	a.Describe(&f, "List the Flex Plugin Releases of the account (GET /v1/PluginService/Releases on flex-api.twilio.com).")
}

func (f *GetFlexPluginReleases) Invoke(ctx context.Context, _ infer.FunctionRequest[GetFlexPluginReleasesArgs]) (infer.FunctionResponse[GetFlexPluginReleasesResult], error) {
	const fn = "getFlexPluginReleases"
	none := infer.FunctionResponse[GetFlexPluginReleasesResult]{}
	c, err := clientFor(ctx)
	if err != nil {
		return none, err
	}
	out := []FlexPluginReleaseInfo{}
	err = listPages(ctx, c, fn, flexProduct, flexPluginPath("Releases"), url.Values{"PageSize": {"1000"}}, "releases", func(o obj) bool {
		out = append(out, flexPluginReleaseInfo(o))
		return true
	})
	if err != nil {
		return none, lookupFailure(fn, "Flex Plugin Releases", err)
	}
	return infer.FunctionResponse[GetFlexPluginReleasesResult]{Output: GetFlexPluginReleasesResult{FlexPluginReleases: out}}, nil
}
