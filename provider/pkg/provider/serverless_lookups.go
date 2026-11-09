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

// ------------------------------------------------------------ getServerlessService

// GetServerlessService looks up a Serverless service.
type GetServerlessService struct{}

// GetServerlessServiceArgs are the lookup arguments.
type GetServerlessServiceArgs struct {
	ServiceSid string `pulumi:"serviceSid"`
}

func (a *GetServerlessServiceArgs) Annotate(an infer.Annotator) {
	an.Describe(&a.ServiceSid, "SID (ZS...) of the Serverless service.")
}

// GetServerlessServiceResult is the looked-up service.
type GetServerlessServiceResult struct {
	Sid                string `pulumi:"sid"`
	AccountSid         string `pulumi:"accountSid"`
	UniqueName         string `pulumi:"uniqueName"`
	FriendlyName       string `pulumi:"friendlyName"`
	IncludeCredentials *bool  `pulumi:"includeCredentials,optional"`
	UiEditable         *bool  `pulumi:"uiEditable,optional"`
	DomainBase         string `pulumi:"domainBase"`
	Url                string `pulumi:"url"`
	DateCreated        string `pulumi:"dateCreated"`
	DateUpdated        string `pulumi:"dateUpdated"`
}

func (a *GetServerlessServiceResult) Annotate(an infer.Annotator) {
	an.Describe(&a.Sid, "The service SID (ZS...).")
	an.Describe(&a.AccountSid, "SID of the account that owns the service.")
	an.Describe(&a.UniqueName, "Unique name of the service.")
	an.Describe(&a.FriendlyName, "Human-readable name of the service.")
	an.Describe(&a.IncludeCredentials, "Whether Twilio injects the account credentials into the Functions of the service.")
	an.Describe(&a.UiEditable, "Whether the Functions and Assets of the service may be edited in the Twilio Console.")
	an.Describe(&a.DomainBase, "The base domain name of the service's environments.")
	an.Describe(&a.Url, "The absolute URL of the service resource.")
	an.Describe(&a.DateCreated, "When the service was created (ISO 8601).")
	an.Describe(&a.DateUpdated, "When the service was last updated (ISO 8601).")
}

func (f *GetServerlessService) Annotate(a infer.Annotator) {
	a.Describe(&f, "Look up a Serverless service by SID.")
}

func (f *GetServerlessService) Invoke(ctx context.Context, req infer.FunctionRequest[GetServerlessServiceArgs]) (infer.FunctionResponse[GetServerlessServiceResult], error) {
	const fn = "getServerlessService"
	in := req.Input
	if err := requireArgs(fn, [2]string{"serviceSid", in.ServiceSid}); err != nil {
		return infer.FunctionResponse[GetServerlessServiceResult]{}, err
	}
	c, err := clientFor(ctx)
	if err != nil {
		return infer.FunctionResponse[GetServerlessServiceResult]{}, err
	}
	o, err := c.do(ctx, "GET", "serverless", serverlessPath(in.ServiceSid), nil, nil)
	if err != nil {
		return infer.FunctionResponse[GetServerlessServiceResult]{}, lookupFailure(fn, "service "+in.ServiceSid, err)
	}
	return infer.FunctionResponse[GetServerlessServiceResult]{Output: GetServerlessServiceResult{
		Sid: o.str("sid"), AccountSid: o.str("account_sid"), UniqueName: o.str("unique_name"), FriendlyName: o.str("friendly_name"),
		IncludeCredentials: o.optBool("include_credentials"), UiEditable: o.optBool("ui_editable"),
		DomainBase: o.str("domain_base"), Url: o.str("url"), DateCreated: o.str("date_created"), DateUpdated: o.str("date_updated"),
	}}, nil
}

// ------------------------------------------------------------ getServerlessEnvironment

// GetServerlessEnvironment looks up an environment of a Serverless service.
type GetServerlessEnvironment struct{}

// GetServerlessEnvironmentArgs are the lookup arguments.
type GetServerlessEnvironmentArgs struct {
	ServiceSid     string `pulumi:"serviceSid"`
	EnvironmentSid string `pulumi:"environmentSid"`
}

func (a *GetServerlessEnvironmentArgs) Annotate(an infer.Annotator) {
	an.Describe(&a.ServiceSid, "SID (ZS...) of the Serverless service.")
	an.Describe(&a.EnvironmentSid, "SID (ZE...) of the environment.")
}

// GetServerlessEnvironmentResult is the looked-up environment.
type GetServerlessEnvironmentResult struct {
	ServiceSid  string  `pulumi:"serviceSid"`
	Sid         string  `pulumi:"sid"`
	AccountSid  string  `pulumi:"accountSid"`
	UniqueName  string  `pulumi:"uniqueName"`
	BuildSid    *string `pulumi:"buildSid,optional"`
	DomainName  string  `pulumi:"domainName"`
	Url         string  `pulumi:"url"`
	DateCreated string  `pulumi:"dateCreated"`
	DateUpdated string  `pulumi:"dateUpdated"`
}

func (a *GetServerlessEnvironmentResult) Annotate(an infer.Annotator) {
	an.Describe(&a.ServiceSid, "SID of the Serverless service.")
	an.Describe(&a.Sid, "The environment SID (ZE...).")
	an.Describe(&a.AccountSid, "SID of the account that owns the environment.")
	an.Describe(&a.UniqueName, "Unique name of the environment.")
	an.Describe(&a.BuildSid, "SID (ZB...) of the build deployed to the environment, when there is one.")
	an.Describe(&a.DomainName, "The domain name of the environment.")
	an.Describe(&a.Url, "The absolute URL of the environment resource.")
	an.Describe(&a.DateCreated, "When the environment was created (ISO 8601).")
	an.Describe(&a.DateUpdated, "When the environment was last updated (ISO 8601).")
}

func (f *GetServerlessEnvironment) Annotate(a infer.Annotator) {
	a.Describe(&f, "Look up an environment of a Serverless service by SID.")
}

func (f *GetServerlessEnvironment) Invoke(ctx context.Context, req infer.FunctionRequest[GetServerlessEnvironmentArgs]) (infer.FunctionResponse[GetServerlessEnvironmentResult], error) {
	const fn = "getServerlessEnvironment"
	in := req.Input
	if err := requireArgs(fn, [2]string{"serviceSid", in.ServiceSid}, [2]string{"environmentSid", in.EnvironmentSid}); err != nil {
		return infer.FunctionResponse[GetServerlessEnvironmentResult]{}, err
	}
	c, err := clientFor(ctx)
	if err != nil {
		return infer.FunctionResponse[GetServerlessEnvironmentResult]{}, err
	}
	o, err := c.do(ctx, "GET", "serverless", serverlessSvcPath(in.ServiceSid, "Environments", in.EnvironmentSid), nil, nil)
	if err != nil {
		return infer.FunctionResponse[GetServerlessEnvironmentResult]{}, lookupFailure(fn, "environment "+in.EnvironmentSid+" of service "+in.ServiceSid, err)
	}
	return infer.FunctionResponse[GetServerlessEnvironmentResult]{Output: GetServerlessEnvironmentResult{
		ServiceSid: in.ServiceSid, Sid: o.str("sid"), AccountSid: o.str("account_sid"), UniqueName: o.str("unique_name"),
		BuildSid: nonEmpty(o.optStr("build_sid")), DomainName: o.str("domain_name"), Url: o.str("url"),
		DateCreated: o.str("date_created"), DateUpdated: o.str("date_updated"),
	}}, nil
}

// ------------------------------------------------------------ getServerlessVariable

// GetServerlessVariable looks up an environment variable of a Serverless environment.
type GetServerlessVariable struct{}

// GetServerlessVariableArgs are the lookup arguments.
type GetServerlessVariableArgs struct {
	ServiceSid     string `pulumi:"serviceSid"`
	EnvironmentSid string `pulumi:"environmentSid"`
	VariableSid    string `pulumi:"variableSid"`
}

func (a *GetServerlessVariableArgs) Annotate(an infer.Annotator) {
	an.Describe(&a.ServiceSid, "SID (ZS...) of the Serverless service.")
	an.Describe(&a.EnvironmentSid, "SID (ZE...) of the environment.")
	an.Describe(&a.VariableSid, "SID (ZV...) of the variable.")
}

// GetServerlessVariableResult is the looked-up variable.
type GetServerlessVariableResult struct {
	ServiceSid     string `pulumi:"serviceSid"`
	EnvironmentSid string `pulumi:"environmentSid"`
	Sid            string `pulumi:"sid"`
	Key            string `pulumi:"key"`
	Value          string `pulumi:"value" provider:"secret"`
	DateCreated    string `pulumi:"dateCreated"`
	DateUpdated    string `pulumi:"dateUpdated"`
}

func (a *GetServerlessVariableResult) Annotate(an infer.Annotator) {
	an.Describe(&a.ServiceSid, "SID of the Serverless service.")
	an.Describe(&a.EnvironmentSid, "SID of the environment.")
	an.Describe(&a.Sid, "The variable SID (ZV...).")
	an.Describe(&a.Key, "Name of the variable.")
	an.Describe(&a.Value, "Value of the variable, returned as a Pulumi secret.")
	an.Describe(&a.DateCreated, "When the variable was created (ISO 8601).")
	an.Describe(&a.DateUpdated, "When the variable was last updated (ISO 8601).")
}

func (f *GetServerlessVariable) Annotate(a infer.Annotator) {
	a.Describe(&f, "Look up an environment variable of a Serverless environment by SID. The value is a Pulumi secret.")
}

func (f *GetServerlessVariable) Invoke(ctx context.Context, req infer.FunctionRequest[GetServerlessVariableArgs]) (infer.FunctionResponse[GetServerlessVariableResult], error) {
	const fn = "getServerlessVariable"
	in := req.Input
	if err := requireArgs(fn, [2]string{"serviceSid", in.ServiceSid}, [2]string{"environmentSid", in.EnvironmentSid}, [2]string{"variableSid", in.VariableSid}); err != nil {
		return infer.FunctionResponse[GetServerlessVariableResult]{}, err
	}
	c, err := clientFor(ctx)
	if err != nil {
		return infer.FunctionResponse[GetServerlessVariableResult]{}, err
	}
	o, err := c.do(ctx, "GET", "serverless", serverlessEnvPath(in.ServiceSid, in.EnvironmentSid, in.VariableSid), nil, nil)
	if err != nil {
		return infer.FunctionResponse[GetServerlessVariableResult]{}, lookupFailure(fn, "variable "+in.VariableSid+" of environment "+in.EnvironmentSid, err)
	}
	return infer.FunctionResponse[GetServerlessVariableResult]{Output: GetServerlessVariableResult{
		ServiceSid: in.ServiceSid, EnvironmentSid: in.EnvironmentSid, Sid: o.str("sid"), Key: o.str("key"), Value: o.str("value"),
		DateCreated: o.str("date_created"), DateUpdated: o.str("date_updated"),
	}}, nil
}

// ------------------------------------------------------------ getServerlessFunction

// GetServerlessFunction looks up the function container of a Serverless service.
type GetServerlessFunction struct{}

// GetServerlessFunctionArgs are the lookup arguments.
type GetServerlessFunctionArgs struct {
	ServiceSid  string `pulumi:"serviceSid"`
	FunctionSid string `pulumi:"functionSid"`
}

func (a *GetServerlessFunctionArgs) Annotate(an infer.Annotator) {
	an.Describe(&a.ServiceSid, "SID (ZS...) of the Serverless service.")
	an.Describe(&a.FunctionSid, "SID (ZH...) of the function.")
}

// GetServerlessFunctionResult is the looked-up function.
type GetServerlessFunctionResult struct {
	ServiceSid   string `pulumi:"serviceSid"`
	Sid          string `pulumi:"sid"`
	AccountSid   string `pulumi:"accountSid"`
	FriendlyName string `pulumi:"friendlyName"`
	Url          string `pulumi:"url"`
	DateCreated  string `pulumi:"dateCreated"`
	DateUpdated  string `pulumi:"dateUpdated"`
}

func (a *GetServerlessFunctionResult) Annotate(an infer.Annotator) {
	an.Describe(&a.ServiceSid, "SID of the Serverless service.")
	an.Describe(&a.Sid, "The function SID (ZH...).")
	an.Describe(&a.AccountSid, "SID of the account that owns the function.")
	an.Describe(&a.FriendlyName, "Human-readable name of the function.")
	an.Describe(&a.Url, "The absolute URL of the function resource.")
	an.Describe(&a.DateCreated, "When the function was created (ISO 8601).")
	an.Describe(&a.DateUpdated, "When the function was last updated (ISO 8601).")
}

func (f *GetServerlessFunction) Annotate(a infer.Annotator) {
	a.Describe(&f, "Look up the function container of a Serverless service by SID.")
}

func (f *GetServerlessFunction) Invoke(ctx context.Context, req infer.FunctionRequest[GetServerlessFunctionArgs]) (infer.FunctionResponse[GetServerlessFunctionResult], error) {
	const fn = "getServerlessFunction"
	in := req.Input
	if err := requireArgs(fn, [2]string{"serviceSid", in.ServiceSid}, [2]string{"functionSid", in.FunctionSid}); err != nil {
		return infer.FunctionResponse[GetServerlessFunctionResult]{}, err
	}
	c, err := clientFor(ctx)
	if err != nil {
		return infer.FunctionResponse[GetServerlessFunctionResult]{}, err
	}
	o, err := c.do(ctx, "GET", "serverless", serverlessSvcPath(in.ServiceSid, "Functions", in.FunctionSid), nil, nil)
	if err != nil {
		return infer.FunctionResponse[GetServerlessFunctionResult]{}, lookupFailure(fn, "function "+in.FunctionSid+" of service "+in.ServiceSid, err)
	}
	return infer.FunctionResponse[GetServerlessFunctionResult]{Output: GetServerlessFunctionResult{
		ServiceSid: in.ServiceSid, Sid: o.str("sid"), AccountSid: o.str("account_sid"), FriendlyName: o.str("friendly_name"),
		Url: o.str("url"), DateCreated: o.str("date_created"), DateUpdated: o.str("date_updated"),
	}}, nil
}

// ------------------------------------------------------------ getServerlessAsset

// GetServerlessAsset looks up the asset container of a Serverless service.
type GetServerlessAsset struct{}

// GetServerlessAssetArgs are the lookup arguments.
type GetServerlessAssetArgs struct {
	ServiceSid string `pulumi:"serviceSid"`
	AssetSid   string `pulumi:"assetSid"`
}

func (a *GetServerlessAssetArgs) Annotate(an infer.Annotator) {
	an.Describe(&a.ServiceSid, "SID (ZS...) of the Serverless service.")
	an.Describe(&a.AssetSid, "SID (ZH...) of the asset.")
}

// GetServerlessAssetResult is the looked-up asset.
type GetServerlessAssetResult struct {
	ServiceSid   string `pulumi:"serviceSid"`
	Sid          string `pulumi:"sid"`
	AccountSid   string `pulumi:"accountSid"`
	FriendlyName string `pulumi:"friendlyName"`
	Url          string `pulumi:"url"`
	DateCreated  string `pulumi:"dateCreated"`
	DateUpdated  string `pulumi:"dateUpdated"`
}

func (a *GetServerlessAssetResult) Annotate(an infer.Annotator) {
	an.Describe(&a.ServiceSid, "SID of the Serverless service.")
	an.Describe(&a.Sid, "The asset SID (ZH...).")
	an.Describe(&a.AccountSid, "SID of the account that owns the asset.")
	an.Describe(&a.FriendlyName, "Human-readable name of the asset.")
	an.Describe(&a.Url, "The absolute URL of the asset resource.")
	an.Describe(&a.DateCreated, "When the asset was created (ISO 8601).")
	an.Describe(&a.DateUpdated, "When the asset was last updated (ISO 8601).")
}

func (f *GetServerlessAsset) Annotate(a infer.Annotator) {
	a.Describe(&f, "Look up the asset container of a Serverless service by SID.")
}

func (f *GetServerlessAsset) Invoke(ctx context.Context, req infer.FunctionRequest[GetServerlessAssetArgs]) (infer.FunctionResponse[GetServerlessAssetResult], error) {
	const fn = "getServerlessAsset"
	in := req.Input
	if err := requireArgs(fn, [2]string{"serviceSid", in.ServiceSid}, [2]string{"assetSid", in.AssetSid}); err != nil {
		return infer.FunctionResponse[GetServerlessAssetResult]{}, err
	}
	c, err := clientFor(ctx)
	if err != nil {
		return infer.FunctionResponse[GetServerlessAssetResult]{}, err
	}
	o, err := c.do(ctx, "GET", "serverless", serverlessSvcPath(in.ServiceSid, "Assets", in.AssetSid), nil, nil)
	if err != nil {
		return infer.FunctionResponse[GetServerlessAssetResult]{}, lookupFailure(fn, "asset "+in.AssetSid+" of service "+in.ServiceSid, err)
	}
	return infer.FunctionResponse[GetServerlessAssetResult]{Output: GetServerlessAssetResult{
		ServiceSid: in.ServiceSid, Sid: o.str("sid"), AccountSid: o.str("account_sid"), FriendlyName: o.str("friendly_name"),
		Url: o.str("url"), DateCreated: o.str("date_created"), DateUpdated: o.str("date_updated"),
	}}, nil
}
