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

	"github.com/pulumi/pulumi-go-provider/infer"
)

// Lookups for the deploy plane of a Serverless service: versions, builds and deployments.

// ------------------------------------------------------------ getServerlessFunctionVersion

// GetServerlessFunctionVersion looks up one version of a Function.
type GetServerlessFunctionVersion struct{}

// GetServerlessFunctionVersionArgs are the lookup arguments.
type GetServerlessFunctionVersionArgs struct {
	ServiceSid  string `pulumi:"serviceSid"`
	FunctionSid string `pulumi:"functionSid"`
	VersionSid  string `pulumi:"versionSid"`
}

func (a *GetServerlessFunctionVersionArgs) Annotate(an infer.Annotator) {
	an.Describe(&a.ServiceSid, "SID (ZS...) of the Serverless service.")
	an.Describe(&a.FunctionSid, "SID (ZH...) of the Function.")
	an.Describe(&a.VersionSid, "SID (ZN...) of the version.")
}

// GetServerlessFunctionVersionResult is the looked-up version.
type GetServerlessFunctionVersionResult struct {
	ServiceSid  string `pulumi:"serviceSid"`
	FunctionSid string `pulumi:"functionSid"`
	Sid         string `pulumi:"sid"`
	AccountSid  string `pulumi:"accountSid"`
	Path        string `pulumi:"path"`
	Visibility  string `pulumi:"visibility"`
	Url         string `pulumi:"url"`
	DateCreated string `pulumi:"dateCreated"`
}

func (a *GetServerlessFunctionVersionResult) Annotate(an infer.Annotator) {
	an.Describe(&a.ServiceSid, "SID of the Serverless service.")
	an.Describe(&a.FunctionSid, "SID of the Function the version belongs to.")
	an.Describe(&a.Sid, "The version SID (ZN...).")
	an.Describe(&a.AccountSid, "SID of the account that owns the version.")
	an.Describe(&a.Path, "URL path the function answers on.")
	an.Describe(&a.Visibility, "public, protected or private.")
	an.Describe(&a.Url, "The absolute URL of the version resource.")
	an.Describe(&a.DateCreated, "When the version was created (ISO 8601).")
}

func (f *GetServerlessFunctionVersion) Annotate(a infer.Annotator) {
	a.Describe(&f, "Look up one version of a Serverless Function by SID.")
}

func (f *GetServerlessFunctionVersion) Invoke(ctx context.Context, req infer.FunctionRequest[GetServerlessFunctionVersionArgs]) (infer.FunctionResponse[GetServerlessFunctionVersionResult], error) {
	const fn = "getServerlessFunctionVersion"
	in := req.Input
	if err := requireArgs(fn, [2]string{"serviceSid", in.ServiceSid}, [2]string{"functionSid", in.FunctionSid}, [2]string{"versionSid", in.VersionSid}); err != nil {
		return infer.FunctionResponse[GetServerlessFunctionVersionResult]{}, err
	}
	c, err := clientFor(ctx)
	if err != nil {
		return infer.FunctionResponse[GetServerlessFunctionVersionResult]{}, err
	}
	o, err := c.do(ctx, "GET", "serverless", serverlessPath(in.ServiceSid, "Functions", in.FunctionSid, "Versions", in.VersionSid), nil, nil)
	if err != nil {
		return infer.FunctionResponse[GetServerlessFunctionVersionResult]{}, lookupFailure(fn, "version "+in.VersionSid+" of function "+in.FunctionSid, err)
	}
	return infer.FunctionResponse[GetServerlessFunctionVersionResult]{Output: GetServerlessFunctionVersionResult{
		ServiceSid: in.ServiceSid, FunctionSid: in.FunctionSid, Sid: o.str("sid"), AccountSid: o.str("account_sid"),
		Path: o.str("path"), Visibility: o.str("visibility"), Url: o.str("url"), DateCreated: o.str("date_created"),
	}}, nil
}

// ------------------------------------------------------------ getServerlessAssetVersion

// GetServerlessAssetVersion looks up one version of an Asset.
type GetServerlessAssetVersion struct{}

// GetServerlessAssetVersionArgs are the lookup arguments.
type GetServerlessAssetVersionArgs struct {
	ServiceSid string `pulumi:"serviceSid"`
	AssetSid   string `pulumi:"assetSid"`
	VersionSid string `pulumi:"versionSid"`
}

func (a *GetServerlessAssetVersionArgs) Annotate(an infer.Annotator) {
	an.Describe(&a.ServiceSid, "SID (ZS...) of the Serverless service.")
	an.Describe(&a.AssetSid, "SID (ZH...) of the Asset.")
	an.Describe(&a.VersionSid, "SID (ZN...) of the version.")
}

// GetServerlessAssetVersionResult is the looked-up version.
type GetServerlessAssetVersionResult struct {
	ServiceSid  string `pulumi:"serviceSid"`
	AssetSid    string `pulumi:"assetSid"`
	Sid         string `pulumi:"sid"`
	AccountSid  string `pulumi:"accountSid"`
	Path        string `pulumi:"path"`
	Visibility  string `pulumi:"visibility"`
	Url         string `pulumi:"url"`
	DateCreated string `pulumi:"dateCreated"`
}

func (a *GetServerlessAssetVersionResult) Annotate(an infer.Annotator) {
	an.Describe(&a.ServiceSid, "SID of the Serverless service.")
	an.Describe(&a.AssetSid, "SID of the Asset the version belongs to.")
	an.Describe(&a.Sid, "The version SID (ZN...).")
	an.Describe(&a.AccountSid, "SID of the account that owns the version.")
	an.Describe(&a.Path, "URL path the asset is served on.")
	an.Describe(&a.Visibility, "public, protected or private.")
	an.Describe(&a.Url, "The absolute URL of the version resource.")
	an.Describe(&a.DateCreated, "When the version was created (ISO 8601).")
}

func (f *GetServerlessAssetVersion) Annotate(a infer.Annotator) {
	a.Describe(&f, "Look up one version of a Serverless Asset by SID.")
}

func (f *GetServerlessAssetVersion) Invoke(ctx context.Context, req infer.FunctionRequest[GetServerlessAssetVersionArgs]) (infer.FunctionResponse[GetServerlessAssetVersionResult], error) {
	const fn = "getServerlessAssetVersion"
	in := req.Input
	if err := requireArgs(fn, [2]string{"serviceSid", in.ServiceSid}, [2]string{"assetSid", in.AssetSid}, [2]string{"versionSid", in.VersionSid}); err != nil {
		return infer.FunctionResponse[GetServerlessAssetVersionResult]{}, err
	}
	c, err := clientFor(ctx)
	if err != nil {
		return infer.FunctionResponse[GetServerlessAssetVersionResult]{}, err
	}
	o, err := c.do(ctx, "GET", "serverless", serverlessPath(in.ServiceSid, "Assets", in.AssetSid, "Versions", in.VersionSid), nil, nil)
	if err != nil {
		return infer.FunctionResponse[GetServerlessAssetVersionResult]{}, lookupFailure(fn, "version "+in.VersionSid+" of asset "+in.AssetSid, err)
	}
	return infer.FunctionResponse[GetServerlessAssetVersionResult]{Output: GetServerlessAssetVersionResult{
		ServiceSid: in.ServiceSid, AssetSid: in.AssetSid, Sid: o.str("sid"), AccountSid: o.str("account_sid"),
		Path: o.str("path"), Visibility: o.str("visibility"), Url: o.str("url"), DateCreated: o.str("date_created"),
	}}, nil
}

// ------------------------------------------------------------ getServerlessBuild

// GetServerlessBuild looks up a build of a Serverless service.
type GetServerlessBuild struct{}

// GetServerlessBuildArgs are the lookup arguments.
type GetServerlessBuildArgs struct {
	ServiceSid string `pulumi:"serviceSid"`
	BuildSid   string `pulumi:"buildSid"`
}

func (a *GetServerlessBuildArgs) Annotate(an infer.Annotator) {
	an.Describe(&a.ServiceSid, "SID (ZS...) of the Serverless service.")
	an.Describe(&a.BuildSid, "SID (ZB...) of the build.")
}

// GetServerlessBuildResult is the looked-up build.
type GetServerlessBuildResult struct {
	ServiceSid          string                      `pulumi:"serviceSid"`
	Sid                 string                      `pulumi:"sid"`
	AccountSid          string                      `pulumi:"accountSid"`
	Status              string                      `pulumi:"status"`
	Runtime             *string                     `pulumi:"runtime,optional"`
	FunctionVersionSids []string                    `pulumi:"functionVersionSids"`
	AssetVersionSids    []string                    `pulumi:"assetVersionSids"`
	Dependencies        []ServerlessBuildDependency `pulumi:"dependencies"`
	Url                 string                      `pulumi:"url"`
	DateCreated         string                      `pulumi:"dateCreated"`
	DateUpdated         string                      `pulumi:"dateUpdated"`
}

func (a *GetServerlessBuildResult) Annotate(an infer.Annotator) {
	an.Describe(&a.ServiceSid, "SID of the Serverless service.")
	an.Describe(&a.Sid, "The build SID (ZB...).")
	an.Describe(&a.AccountSid, "SID of the account that owns the build.")
	an.Describe(&a.Status, "building, completed or failed.")
	an.Describe(&a.Runtime, "Node.js runtime of the build.")
	an.Describe(&a.FunctionVersionSids, "SIDs of the Function versions in the build.")
	an.Describe(&a.AssetVersionSids, "SIDs of the Asset versions in the build.")
	an.Describe(&a.Dependencies, "npm packages installed into the build.")
	an.Describe(&a.Url, "The absolute URL of the build resource.")
	an.Describe(&a.DateCreated, "When the build was created (ISO 8601).")
	an.Describe(&a.DateUpdated, "When the build was last updated (ISO 8601).")
}

func (f *GetServerlessBuild) Annotate(a infer.Annotator) {
	a.Describe(&f, "Look up a Serverless build by SID.")
}

func (f *GetServerlessBuild) Invoke(ctx context.Context, req infer.FunctionRequest[GetServerlessBuildArgs]) (infer.FunctionResponse[GetServerlessBuildResult], error) {
	const fn = "getServerlessBuild"
	in := req.Input
	if err := requireArgs(fn, [2]string{"serviceSid", in.ServiceSid}, [2]string{"buildSid", in.BuildSid}); err != nil {
		return infer.FunctionResponse[GetServerlessBuildResult]{}, err
	}
	c, err := clientFor(ctx)
	if err != nil {
		return infer.FunctionResponse[GetServerlessBuildResult]{}, err
	}
	o, err := c.do(ctx, "GET", "serverless", serverlessSvcPath(in.ServiceSid, "Builds", in.BuildSid), nil, nil)
	if err != nil {
		return infer.FunctionResponse[GetServerlessBuildResult]{}, lookupFailure(fn, "build "+in.BuildSid+" of service "+in.ServiceSid, err)
	}
	res := GetServerlessBuildResult{
		ServiceSid: in.ServiceSid, Sid: o.str("sid"), AccountSid: o.str("account_sid"), Status: o.str("status"), Runtime: o.optStr("runtime"),
		FunctionVersionSids: []string{}, AssetVersionSids: []string{}, Dependencies: []ServerlessBuildDependency{},
		Url: o.str("url"), DateCreated: o.str("date_created"), DateUpdated: o.str("date_updated"),
	}
	if l := versionList(o, "function_versions"); l != nil {
		res.FunctionVersionSids = *l
	}
	if l := versionList(o, "asset_versions"); l != nil {
		res.AssetVersionSids = *l
	}
	if d := buildDeps(o); d != nil {
		res.Dependencies = *d
	}
	return infer.FunctionResponse[GetServerlessBuildResult]{Output: res}, nil
}

// ------------------------------------------------------------ getServerlessDeployment

// GetServerlessDeployment looks up a deployment of an environment.
type GetServerlessDeployment struct{}

// GetServerlessDeploymentArgs are the lookup arguments.
type GetServerlessDeploymentArgs struct {
	ServiceSid     string `pulumi:"serviceSid"`
	EnvironmentSid string `pulumi:"environmentSid"`
	DeploymentSid  string `pulumi:"deploymentSid"`
}

func (a *GetServerlessDeploymentArgs) Annotate(an infer.Annotator) {
	an.Describe(&a.ServiceSid, "SID (ZS...) of the Serverless service.")
	an.Describe(&a.EnvironmentSid, "SID (ZE...) of the environment.")
	an.Describe(&a.DeploymentSid, "SID (ZD...) of the deployment.")
}

// GetServerlessDeploymentResult is the looked-up deployment.
type GetServerlessDeploymentResult struct {
	ServiceSid     string `pulumi:"serviceSid"`
	EnvironmentSid string `pulumi:"environmentSid"`
	Sid            string `pulumi:"sid"`
	AccountSid     string `pulumi:"accountSid"`
	BuildSid       string `pulumi:"buildSid"`
	Url            string `pulumi:"url"`
	DateCreated    string `pulumi:"dateCreated"`
	DateUpdated    string `pulumi:"dateUpdated"`
}

func (a *GetServerlessDeploymentResult) Annotate(an infer.Annotator) {
	an.Describe(&a.ServiceSid, "SID of the Serverless service.")
	an.Describe(&a.EnvironmentSid, "SID of the environment.")
	an.Describe(&a.Sid, "The deployment SID (ZD...).")
	an.Describe(&a.AccountSid, "SID of the account that owns the deployment.")
	an.Describe(&a.BuildSid, "SID (ZB...) of the build that was deployed.")
	an.Describe(&a.Url, "The absolute URL of the deployment resource.")
	an.Describe(&a.DateCreated, "When the deployment was created (ISO 8601).")
	an.Describe(&a.DateUpdated, "When the deployment was last updated (ISO 8601).")
}

func (f *GetServerlessDeployment) Annotate(a infer.Annotator) {
	a.Describe(&f, "Look up a deployment of a Serverless environment by SID.")
}

func (f *GetServerlessDeployment) Invoke(ctx context.Context, req infer.FunctionRequest[GetServerlessDeploymentArgs]) (infer.FunctionResponse[GetServerlessDeploymentResult], error) {
	const fn = "getServerlessDeployment"
	in := req.Input
	if err := requireArgs(fn, [2]string{"serviceSid", in.ServiceSid}, [2]string{"environmentSid", in.EnvironmentSid}, [2]string{"deploymentSid", in.DeploymentSid}); err != nil {
		return infer.FunctionResponse[GetServerlessDeploymentResult]{}, err
	}
	c, err := clientFor(ctx)
	if err != nil {
		return infer.FunctionResponse[GetServerlessDeploymentResult]{}, err
	}
	o, err := c.do(ctx, "GET", "serverless", deploymentPath(in.ServiceSid, in.EnvironmentSid, in.DeploymentSid), nil, nil)
	if err != nil {
		return infer.FunctionResponse[GetServerlessDeploymentResult]{}, lookupFailure(fn, "deployment "+in.DeploymentSid+" of environment "+in.EnvironmentSid, err)
	}
	return infer.FunctionResponse[GetServerlessDeploymentResult]{Output: GetServerlessDeploymentResult{
		ServiceSid: in.ServiceSid, EnvironmentSid: in.EnvironmentSid, Sid: o.str("sid"), AccountSid: o.str("account_sid"),
		BuildSid: o.str("build_sid"), Url: o.str("url"), DateCreated: o.str("date_created"), DateUpdated: o.str("date_updated"),
	}}, nil
}
