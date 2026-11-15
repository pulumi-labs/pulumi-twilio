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
	"reflect"

	"github.com/pulumi/pulumi-go-provider/infer"
)

// flexConfigurationID is the resource ID of the Flex configuration, of which there is one per account.
const flexConfigurationID = "account"

// FlexConfiguration manages the account-level Flex settings (flex-api.twilio.com v1 /Configuration):
// the UI attributes and language, call recording, CRM and plugin service switches. An account has
// exactly one, so this adopts it; only the settings the program states are written and compared, and
// destroying the resource forgets it and leaves the settings as they are.
type FlexConfiguration struct{}

// FlexConfigurationArgs are the inputs of a FlexConfiguration. Twilio documents no defaults for
// them, so an import adopts none and a program only ever manages the settings it states.
type FlexConfigurationArgs struct {
	Attributes              *string   `pulumi:"attributes,optional"`
	UiAttributes            *string   `pulumi:"uiAttributes,optional"`
	UiDependencies          *string   `pulumi:"uiDependencies,optional"`
	UiLanguage              *string   `pulumi:"uiLanguage,optional"`
	PublicAttributes        *string   `pulumi:"publicAttributes,optional"`
	CallRecordingEnabled    *bool     `pulumi:"callRecordingEnabled,optional"`
	CallRecordingWebhookUrl *string   `pulumi:"callRecordingWebhookUrl,optional"`
	CrmEnabled              *bool     `pulumi:"crmEnabled,optional"`
	CrmType                 *string   `pulumi:"crmType,optional"`
	CrmCallbackUrl          *string   `pulumi:"crmCallbackUrl,optional"`
	CrmFallbackUrl          *string   `pulumi:"crmFallbackUrl,optional"`
	CrmAttributes           *string   `pulumi:"crmAttributes,optional"`
	PluginServiceEnabled    *bool     `pulumi:"pluginServiceEnabled,optional"`
	PluginServiceAttributes *string   `pulumi:"pluginServiceAttributes,optional"`
	ServerlessServiceSids   *[]string `pulumi:"serverlessServiceSids,optional"`
}

func (a *FlexConfigurationArgs) Annotate(an infer.Annotator) {
	const keep = " Removing it leaves the setting unchanged. Twilio does not document a default, so an import does not adopt it."
	an.Describe(&a.Attributes, "Free-form attributes of the Flex configuration as a JSON object. Formatting and key order do not matter."+keep)
	an.Describe(&a.UiAttributes, "The Flex UI attributes as a JSON object (branding, feature switches and so on). Formatting and key order do not matter."+keep)
	an.Describe(&a.UiDependencies, "The Flex UI dependencies as a JSON object. Formatting and key order do not matter."+keep)
	an.Describe(&a.UiLanguage, "The primary language of the Flex UI, for example en-US."+keep)
	an.Describe(&a.PublicAttributes, "Attributes of the Flex configuration that are readable without authentication, as a JSON object. Formatting and key order do not matter."+keep)
	an.Describe(&a.CallRecordingEnabled, "Whether calls are recorded."+keep)
	an.Describe(&a.CallRecordingWebhookUrl, "The URL Twilio calls when a call recording is available."+keep)
	an.Describe(&a.CrmEnabled, "Whether the CRM integration is enabled."+keep)
	an.Describe(&a.CrmType, "The CRM type, for example custom."+keep)
	an.Describe(&a.CrmCallbackUrl, "The URL of the CRM that Flex calls to look up a contact."+keep)
	an.Describe(&a.CrmFallbackUrl, "The URL Flex calls when the CRM callback URL fails."+keep)
	an.Describe(&a.CrmAttributes, "Attributes of the CRM integration as a JSON object. Formatting and key order do not matter."+keep)
	an.Describe(&a.PluginServiceEnabled, "Whether the Flex plugin service is enabled."+keep)
	an.Describe(&a.PluginServiceAttributes, "Attributes of the Flex plugin service as a JSON object. Formatting and key order do not matter."+keep)
	an.Describe(&a.ServerlessServiceSids, "The SIDs (ZS...) of the Serverless services Flex uses."+keep)
}

// FlexConfigurationState is the persisted state of a FlexConfiguration.
type FlexConfigurationState struct {
	FlexConfigurationArgs

	AccountSid                  string `pulumi:"accountSid"`
	Status                      string `pulumi:"status"`
	FlexInstanceSid             string `pulumi:"flexInstanceSid"`
	FlexServiceInstanceSid      string `pulumi:"flexServiceInstanceSid"`
	ChatServiceInstanceSid      string `pulumi:"chatServiceInstanceSid"`
	MessagingServiceInstanceSid string `pulumi:"messagingServiceInstanceSid"`
	TaskrouterWorkspaceSid      string `pulumi:"taskrouterWorkspaceSid"`
	RuntimeDomain               string `pulumi:"runtimeDomain"`
	FlexUrl                     string `pulumi:"flexUrl"`
	UiVersion                   string `pulumi:"uiVersion"`
	ServiceVersion              string `pulumi:"serviceVersion"`
	Url                         string `pulumi:"url"`
	DateUpdated                 string `pulumi:"dateUpdated"`
}

func (s *FlexConfigurationState) Annotate(a infer.Annotator) {
	a.Describe(&s.AccountSid, "The SID (AC...) of the account.")
	a.Describe(&s.Status, "The status of the Flex configuration, for example ok, inprogress, notstarted or failed.")
	a.Describe(&s.FlexInstanceSid, "The SID (GO...) of the Flex instance.")
	a.Describe(&s.FlexServiceInstanceSid, "The SID of the Flex service instance.")
	a.Describe(&s.ChatServiceInstanceSid, "The SID (IS...) of the Chat service instance Flex uses.")
	a.Describe(&s.MessagingServiceInstanceSid, "The SID (MG...) of the messaging service instance Flex uses.")
	a.Describe(&s.TaskrouterWorkspaceSid, "The SID (WS...) of the TaskRouter workspace Flex uses.")
	a.Describe(&s.RuntimeDomain, "The domain of the Flex runtime.")
	a.Describe(&s.FlexUrl, "The URL of the Flex UI.")
	a.Describe(&s.UiVersion, "The version of the Flex UI that is served.")
	a.Describe(&s.ServiceVersion, "The version of the Flex service.")
	a.Describe(&s.Url, "The URL of the Flex configuration resource.")
	a.Describe(&s.DateUpdated, "When the Flex configuration was last updated (ISO 8601).")
}

func (r *FlexConfiguration) Annotate(a infer.Annotator) {
	a.Describe(&r, "The account-level Flex settings. The resource ID is account. Only the settings the program states are written and compared; destroying it forgets it and leaves the settings on Twilio as they are. The update request is a JSON body whose exact shape is not in Twilio's OpenAPI spec (unverified against a live account).")
}

var _ infer.CustomResource[FlexConfigurationArgs, FlexConfigurationState] = (*FlexConfiguration)(nil)
var _ infer.CustomCheck[FlexConfigurationArgs] = (*FlexConfiguration)(nil)
var _ infer.CustomDiff[FlexConfigurationArgs, FlexConfigurationState] = (*FlexConfiguration)(nil)
var _ infer.CustomRead[FlexConfigurationArgs, FlexConfigurationState] = (*FlexConfiguration)(nil)
var _ infer.CustomUpdate[FlexConfigurationArgs, FlexConfigurationState] = (*FlexConfiguration)(nil)
var _ infer.CustomDelete[FlexConfigurationState] = (*FlexConfiguration)(nil)

func (r *FlexConfiguration) Check(ctx context.Context, req infer.CheckRequest) (infer.CheckResponse[FlexConfigurationArgs], error) {
	args, failures, err := infer.DefaultCheck[FlexConfigurationArgs](ctx, req.NewInputs)
	if err != nil || len(failures) > 0 {
		return infer.CheckResponse[FlexConfigurationArgs]{Inputs: args, Failures: failures}, err
	}
	c := &checker{unk: unknowns(req.NewInputs)}
	for _, j := range []struct {
		prop string
		v    *string
	}{{"attributes", args.Attributes}, {"uiAttributes", args.UiAttributes}, {"uiDependencies", args.UiDependencies},
		{"publicAttributes", args.PublicAttributes}, {"crmAttributes", args.CrmAttributes}, {"pluginServiceAttributes", args.PluginServiceAttributes}} {
		if j.v == nil || c.unk(j.prop) {
			continue
		}
		var m map[string]any
		if json.Unmarshal([]byte(*j.v), &m) != nil || m == nil {
			c.fail(j.prop, j.prop+" must be a JSON object")
		}
	}
	return infer.CheckResponse[FlexConfigurationArgs]{Inputs: args, Failures: c.failures}, nil
}

func (r *FlexConfiguration) Diff(_ context.Context, req infer.DiffRequest[FlexConfigurationArgs, FlexConfigurationState]) (infer.DiffResponse, error) {
	o, n := req.State.FlexConfigurationArgs, req.Inputs
	d := trDiffSet{}
	d.jsonDoc("attributes", o.Attributes, n.Attributes)
	d.jsonDoc("uiAttributes", o.UiAttributes, n.UiAttributes)
	d.jsonDoc("uiDependencies", o.UiDependencies, n.UiDependencies)
	d.str("uiLanguage", o.UiLanguage, n.UiLanguage)
	d.jsonDoc("publicAttributes", o.PublicAttributes, n.PublicAttributes)
	d.boolean("callRecordingEnabled", o.CallRecordingEnabled, n.CallRecordingEnabled)
	d.str("callRecordingWebhookUrl", o.CallRecordingWebhookUrl, n.CallRecordingWebhookUrl)
	d.boolean("crmEnabled", o.CrmEnabled, n.CrmEnabled)
	d.str("crmType", o.CrmType, n.CrmType)
	d.str("crmCallbackUrl", o.CrmCallbackUrl, n.CrmCallbackUrl)
	d.str("crmFallbackUrl", o.CrmFallbackUrl, n.CrmFallbackUrl)
	d.jsonDoc("crmAttributes", o.CrmAttributes, n.CrmAttributes)
	d.boolean("pluginServiceEnabled", o.PluginServiceEnabled, n.PluginServiceEnabled)
	d.jsonDoc("pluginServiceAttributes", o.PluginServiceAttributes, n.PluginServiceAttributes)
	d.strList("serverlessServiceSids", o.ServerlessServiceSids, n.ServerlessServiceSids)
	return d.response(), nil
}

// body is the JSON update request: account_sid plus the settings the program states.
func (a FlexConfigurationArgs) body(account string) (map[string]any, int) {
	b := map[string]any{"account_sid": account}
	n := 0
	str := func(k string, v *string) {
		if v != nil {
			b[k] = *v
			n++
		}
	}
	boolean := func(k string, v *bool) {
		if v != nil {
			b[k] = *v
			n++
		}
	}
	doc := func(k string, v *string) {
		if v == nil {
			return
		}
		var m any
		if json.Unmarshal([]byte(*v), &m) == nil {
			b[k] = m
			n++
		}
	}
	doc("attributes", a.Attributes)
	doc("ui_attributes", a.UiAttributes)
	doc("ui_dependencies", a.UiDependencies)
	str("ui_language", a.UiLanguage)
	doc("public_attributes", a.PublicAttributes)
	boolean("call_recording_enabled", a.CallRecordingEnabled)
	str("call_recording_webhook_url", a.CallRecordingWebhookUrl)
	boolean("crm_enabled", a.CrmEnabled)
	str("crm_type", a.CrmType)
	str("crm_callback_url", a.CrmCallbackUrl)
	str("crm_fallback_url", a.CrmFallbackUrl)
	doc("crm_attributes", a.CrmAttributes)
	boolean("plugin_service_enabled", a.PluginServiceEnabled)
	doc("plugin_service_attributes", a.PluginServiceAttributes)
	if a.ServerlessServiceSids != nil {
		b["serverless_service_sids"] = *a.ServerlessServiceSids
		n++
	}
	return b, n
}

// flexConfigJSON returns the JSON object at key as a compact document, or nil when absent.
func flexConfigJSON(o obj, key string) *string {
	v, ok := o.get(key).(map[string]any)
	if !ok {
		return nil
	}
	raw, err := json.Marshal(v)
	if err != nil {
		return nil
	}
	s := string(raw)
	return &s
}

func (s *FlexConfigurationState) fill(o obj) {
	s.AccountSid = o.str("account_sid")
	s.Status = o.str("status")
	s.FlexInstanceSid = o.str("flex_instance_sid")
	s.FlexServiceInstanceSid = o.str("flex_service_instance_sid")
	s.ChatServiceInstanceSid = o.str("chat_service_instance_sid")
	s.MessagingServiceInstanceSid = o.str("messaging_service_instance_sid")
	s.TaskrouterWorkspaceSid = o.str("taskrouter_workspace_sid")
	s.RuntimeDomain = o.str("runtime_domain")
	s.FlexUrl = o.str("flex_url")
	s.UiVersion = o.str("ui_version")
	s.ServiceVersion = o.str("service_version")
	s.Url = o.str("url")
	s.DateUpdated = o.str("date_updated")
}

// apply writes the settings the program states and returns the configuration; with none it only reads it.
func (a FlexConfigurationArgs) apply(ctx context.Context, verb string) (obj, error) {
	c, err := clientFor(ctx)
	if err != nil {
		return nil, err
	}
	body, n := a.body(c.accountSid)
	path := flexPath("Configuration")
	if n > 0 {
		if _, err := c.doJSON(ctx, "POST", flexProduct, path, nil, body); err != nil {
			return nil, fmt.Errorf("%s Flex configuration: %w", verb, err)
		}
	}
	o, err := c.do(ctx, "GET", flexProduct, path, nil, nil)
	if err != nil {
		return nil, fmt.Errorf("%s Flex configuration: %w", verb, err)
	}
	return o, nil
}

func (r *FlexConfiguration) Create(ctx context.Context, req infer.CreateRequest[FlexConfigurationArgs]) (infer.CreateResponse[FlexConfigurationState], error) {
	type resp = infer.CreateResponse[FlexConfigurationState]
	st := FlexConfigurationState{FlexConfigurationArgs: req.Inputs}
	if req.DryRun {
		return resp{Output: st}, nil
	}
	o, err := req.Inputs.apply(ctx, "configure")
	if err != nil {
		return resp{}, err
	}
	st.fill(o)
	return resp{ID: flexConfigurationID, Output: st}, nil
}

func (r *FlexConfiguration) Read(ctx context.Context, req infer.ReadRequest[FlexConfigurationArgs, FlexConfigurationState]) (infer.ReadResponse[FlexConfigurationArgs, FlexConfigurationState], error) {
	type resp = infer.ReadResponse[FlexConfigurationArgs, FlexConfigurationState]
	if req.ID != flexConfigurationID {
		return resp{}, fmt.Errorf("cannot import Flex configuration %q: the import ID is %q", req.ID, flexConfigurationID)
	}
	c, err := clientFor(ctx)
	if err != nil {
		return resp{}, err
	}
	o, err := c.do(ctx, "GET", flexProduct, flexPath("Configuration"), nil, nil)
	if err != nil {
		if isNotFound(err) {
			if err := notFoundOnImport("Flex configuration", req.ID, req.Inputs, req.State); err != nil {
				return resp{}, err
			}
			return resp{}, nil
		}
		return resp{}, fmt.Errorf("read Flex configuration: %w", err)
	}
	in := req.Inputs
	// An import arrives with no prior state; Twilio documents no defaults, so it adopts nothing and
	// the program's own settings are the only record of what it manages.
	imported := reflect.ValueOf(req.State).IsZero()
	in.Attributes = flexKeepJSON(in.Attributes, flexConfigJSON(o, "attributes"), imported)
	in.UiAttributes = flexKeepJSON(in.UiAttributes, flexConfigJSON(o, "ui_attributes"), imported)
	in.UiDependencies = flexKeepJSON(in.UiDependencies, flexConfigJSON(o, "ui_dependencies"), imported)
	in.PublicAttributes = flexKeepJSON(in.PublicAttributes, flexConfigJSON(o, "public_attributes"), imported)
	in.CrmAttributes = flexKeepJSON(in.CrmAttributes, flexConfigJSON(o, "crm_attributes"), imported)
	in.PluginServiceAttributes = flexKeepJSON(in.PluginServiceAttributes, flexConfigJSON(o, "plugin_service_attributes"), imported)
	in.UiLanguage = flexKeepStr(in.UiLanguage, o.optStr("ui_language"), imported)
	in.CallRecordingEnabled = flexKeep(in.CallRecordingEnabled, o.optBool("call_recording_enabled"), imported)
	in.CallRecordingWebhookUrl = flexKeepStr(in.CallRecordingWebhookUrl, o.optStr("call_recording_webhook_url"), imported)
	in.CrmEnabled = flexKeep(in.CrmEnabled, o.optBool("crm_enabled"), imported)
	in.CrmType = flexKeepStr(in.CrmType, o.optStr("crm_type"), imported)
	in.CrmCallbackUrl = flexKeepStr(in.CrmCallbackUrl, o.optStr("crm_callback_url"), imported)
	in.CrmFallbackUrl = flexKeepStr(in.CrmFallbackUrl, o.optStr("crm_fallback_url"), imported)
	in.PluginServiceEnabled = flexKeep(in.PluginServiceEnabled, o.optBool("plugin_service_enabled"), imported)
	if imported || in.ServerlessServiceSids == nil {
		in.ServerlessServiceSids = nil
	} else if l := o.optList("serverless_service_sids"); l != nil {
		in.ServerlessServiceSids = l
	} else {
		in.ServerlessServiceSids = &[]string{}
	}
	st := FlexConfigurationState{FlexConfigurationArgs: in}
	st.fill(o)
	return resp{ID: req.ID, Inputs: in, State: st}, nil
}

// flexKeepJSON is trJSON for a setting without a documented default: an import never adopts it.
func flexKeepJSON(in, api *string, imported bool) *string {
	if imported {
		return nil
	}
	return trJSON(in, api, false)
}

// flexKeepStr is trStr for a setting without a documented default: an import never adopts it.
func flexKeepStr(in, api *string, imported bool) *string {
	if imported {
		return nil
	}
	return trStr(in, api, "", false)
}

func (r *FlexConfiguration) Update(ctx context.Context, req infer.UpdateRequest[FlexConfigurationArgs, FlexConfigurationState]) (infer.UpdateResponse[FlexConfigurationState], error) {
	type resp = infer.UpdateResponse[FlexConfigurationState]
	st := FlexConfigurationState{FlexConfigurationArgs: req.Inputs}
	if req.DryRun {
		st = req.State
		st.FlexConfigurationArgs = req.Inputs
		return resp{Output: st}, nil
	}
	o, err := req.Inputs.apply(ctx, "update")
	if err != nil {
		return resp{}, err
	}
	st.fill(o)
	return resp{Output: st}, nil
}

// Delete forgets the configuration: an account always has one and the spec offers no way to remove it.
func (r *FlexConfiguration) Delete(_ context.Context, _ infer.DeleteRequest[FlexConfigurationState]) (infer.DeleteResponse, error) {
	return infer.DeleteResponse{}, nil
}

// ------------------------------------------------------------ getFlexConfiguration

// GetFlexConfiguration looks up the Flex configuration of the account.
type GetFlexConfiguration struct{}

// GetFlexConfigurationArgs are the lookup arguments.
type GetFlexConfigurationArgs struct {
	UiVersion *string `pulumi:"uiVersion,optional"`
}

func (a *GetFlexConfigurationArgs) Annotate(an infer.Annotator) {
	an.Describe(&a.UiVersion, "A Flex UI version to evaluate the configuration for, for example 2.0.0. Leave it out for the version that is served.")
}

// FlexConfigurationInfo is the Flex configuration as Twilio reports it.
type FlexConfigurationInfo struct {
	AccountSid                  string    `pulumi:"accountSid"`
	Status                      string    `pulumi:"status"`
	FlexInstanceSid             string    `pulumi:"flexInstanceSid"`
	FlexServiceInstanceSid      string    `pulumi:"flexServiceInstanceSid"`
	ChatServiceInstanceSid      string    `pulumi:"chatServiceInstanceSid"`
	MessagingServiceInstanceSid string    `pulumi:"messagingServiceInstanceSid"`
	TaskrouterWorkspaceSid      string    `pulumi:"taskrouterWorkspaceSid"`
	TaskrouterTargetWorkflowSid string    `pulumi:"taskrouterTargetWorkflowSid"`
	TaskrouterTargetTaskqueue   string    `pulumi:"taskrouterTargetTaskqueueSid"`
	TaskrouterOfflineActivity   string    `pulumi:"taskrouterOfflineActivitySid"`
	RuntimeDomain               string    `pulumi:"runtimeDomain"`
	FlexUrl                     string    `pulumi:"flexUrl"`
	UiVersion                   string    `pulumi:"uiVersion"`
	ServiceVersion              string    `pulumi:"serviceVersion"`
	UiLanguage                  *string   `pulumi:"uiLanguage,optional"`
	Attributes                  *string   `pulumi:"attributes,optional"`
	UiAttributes                *string   `pulumi:"uiAttributes,optional"`
	UiDependencies              *string   `pulumi:"uiDependencies,optional"`
	PublicAttributes            *string   `pulumi:"publicAttributes,optional"`
	CallRecordingEnabled        *bool     `pulumi:"callRecordingEnabled,optional"`
	CallRecordingWebhookUrl     *string   `pulumi:"callRecordingWebhookUrl,optional"`
	CrmEnabled                  *bool     `pulumi:"crmEnabled,optional"`
	CrmType                     *string   `pulumi:"crmType,optional"`
	CrmCallbackUrl              *string   `pulumi:"crmCallbackUrl,optional"`
	CrmFallbackUrl              *string   `pulumi:"crmFallbackUrl,optional"`
	CrmAttributes               *string   `pulumi:"crmAttributes,optional"`
	PluginServiceEnabled        *bool     `pulumi:"pluginServiceEnabled,optional"`
	PluginServiceAttributes     *string   `pulumi:"pluginServiceAttributes,optional"`
	ServerlessServiceSids       *[]string `pulumi:"serverlessServiceSids,optional"`
	Url                         string    `pulumi:"url"`
	DateUpdated                 string    `pulumi:"dateUpdated"`
}

func (a *FlexConfigurationInfo) Annotate(an infer.Annotator) {
	an.Describe(&a.AccountSid, "The SID (AC...) of the account.")
	an.Describe(&a.Status, "The status of the Flex configuration.")
	an.Describe(&a.FlexInstanceSid, "The SID (GO...) of the Flex instance.")
	an.Describe(&a.FlexServiceInstanceSid, "The SID of the Flex service instance.")
	an.Describe(&a.ChatServiceInstanceSid, "The SID (IS...) of the Chat service instance Flex uses.")
	an.Describe(&a.MessagingServiceInstanceSid, "The SID (MG...) of the messaging service instance Flex uses.")
	an.Describe(&a.TaskrouterWorkspaceSid, "The SID (WS...) of the TaskRouter workspace Flex uses.")
	an.Describe(&a.TaskrouterTargetWorkflowSid, "The SID (WW...) of the TaskRouter workflow Flex routes to.")
	an.Describe(&a.TaskrouterTargetTaskqueue, "The SID (WQ...) of the TaskRouter task queue Flex routes to.")
	an.Describe(&a.TaskrouterOfflineActivity, "The SID (WA...) of the TaskRouter activity that means offline.")
	an.Describe(&a.RuntimeDomain, "The domain of the Flex runtime.")
	an.Describe(&a.FlexUrl, "The URL of the Flex UI.")
	an.Describe(&a.UiVersion, "The version of the Flex UI that is served.")
	an.Describe(&a.ServiceVersion, "The version of the Flex service.")
	an.Describe(&a.UiLanguage, "The primary language of the Flex UI.")
	an.Describe(&a.Attributes, "The free-form attributes as a JSON object.")
	an.Describe(&a.UiAttributes, "The Flex UI attributes as a JSON object.")
	an.Describe(&a.UiDependencies, "The Flex UI dependencies as a JSON object.")
	an.Describe(&a.PublicAttributes, "The publicly readable attributes as a JSON object.")
	an.Describe(&a.CallRecordingEnabled, "Whether calls are recorded.")
	an.Describe(&a.CallRecordingWebhookUrl, "The URL Twilio calls when a call recording is available.")
	an.Describe(&a.CrmEnabled, "Whether the CRM integration is enabled.")
	an.Describe(&a.CrmType, "The CRM type.")
	an.Describe(&a.CrmCallbackUrl, "The URL of the CRM that Flex calls to look up a contact.")
	an.Describe(&a.CrmFallbackUrl, "The URL Flex calls when the CRM callback URL fails.")
	an.Describe(&a.CrmAttributes, "The CRM integration attributes as a JSON object.")
	an.Describe(&a.PluginServiceEnabled, "Whether the Flex plugin service is enabled.")
	an.Describe(&a.PluginServiceAttributes, "The Flex plugin service attributes as a JSON object.")
	an.Describe(&a.ServerlessServiceSids, "The SIDs (ZS...) of the Serverless services Flex uses.")
	an.Describe(&a.Url, "The URL of the Flex configuration resource.")
	an.Describe(&a.DateUpdated, "When the Flex configuration was last updated (ISO 8601).")
}

func flexConfigurationInfo(o obj) FlexConfigurationInfo {
	return FlexConfigurationInfo{
		AccountSid:                  o.str("account_sid"),
		Status:                      o.str("status"),
		FlexInstanceSid:             o.str("flex_instance_sid"),
		FlexServiceInstanceSid:      o.str("flex_service_instance_sid"),
		ChatServiceInstanceSid:      o.str("chat_service_instance_sid"),
		MessagingServiceInstanceSid: o.str("messaging_service_instance_sid"),
		TaskrouterWorkspaceSid:      o.str("taskrouter_workspace_sid"),
		TaskrouterTargetWorkflowSid: o.str("taskrouter_target_workflow_sid"),
		TaskrouterTargetTaskqueue:   o.str("taskrouter_target_taskqueue_sid"),
		TaskrouterOfflineActivity:   o.str("taskrouter_offline_activity_sid"),
		RuntimeDomain:               o.str("runtime_domain"),
		FlexUrl:                     o.str("flex_url"),
		UiVersion:                   o.str("ui_version"),
		ServiceVersion:              o.str("service_version"),
		UiLanguage:                  nonEmpty(o.optStr("ui_language")),
		Attributes:                  flexConfigJSON(o, "attributes"),
		UiAttributes:                flexConfigJSON(o, "ui_attributes"),
		UiDependencies:              flexConfigJSON(o, "ui_dependencies"),
		PublicAttributes:            flexConfigJSON(o, "public_attributes"),
		CallRecordingEnabled:        o.optBool("call_recording_enabled"),
		CallRecordingWebhookUrl:     nonEmpty(o.optStr("call_recording_webhook_url")),
		CrmEnabled:                  o.optBool("crm_enabled"),
		CrmType:                     nonEmpty(o.optStr("crm_type")),
		CrmCallbackUrl:              nonEmpty(o.optStr("crm_callback_url")),
		CrmFallbackUrl:              nonEmpty(o.optStr("crm_fallback_url")),
		CrmAttributes:               flexConfigJSON(o, "crm_attributes"),
		PluginServiceEnabled:        o.optBool("plugin_service_enabled"),
		PluginServiceAttributes:     flexConfigJSON(o, "plugin_service_attributes"),
		ServerlessServiceSids:       nonEmptyList(o.optList("serverless_service_sids")),
		Url:                         o.str("url"),
		DateUpdated:                 o.str("date_updated"),
	}
}

func (f *GetFlexConfiguration) Annotate(a infer.Annotator) {
	a.Describe(&f, "Look up the Flex configuration of the account (GET /v1/Configuration on flex-api.twilio.com).")
}

func (f *GetFlexConfiguration) Invoke(ctx context.Context, req infer.FunctionRequest[GetFlexConfigurationArgs]) (infer.FunctionResponse[FlexConfigurationInfo], error) {
	const fn = "getFlexConfiguration"
	type resp = infer.FunctionResponse[FlexConfigurationInfo]
	c, err := clientFor(ctx)
	if err != nil {
		return resp{}, err
	}
	var q url.Values
	if v := req.Input.UiVersion; v != nil && *v != "" {
		q = url.Values{"UiVersion": {*v}}
	}
	o, err := c.do(ctx, "GET", flexProduct, flexPath("Configuration"), q, nil)
	if err != nil {
		return resp{}, lookupFailure(fn, "the Flex configuration", err)
	}
	return resp{Output: flexConfigurationInfo(o)}, nil
}
