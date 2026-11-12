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

// Lookups of the Conversations objects. Each reads one object and returns what Twilio reports,
// null where Twilio reports nothing. Secrets (push credential keys) are never returned.

// cvLookup reads one object for a lookup function.
func cvLookup(ctx context.Context, fn, what, path string) (obj, error) {
	c, err := clientFor(ctx)
	if err != nil {
		return nil, err
	}
	o, err := c.do(ctx, "GET", cvProduct, path, nil, nil)
	if err != nil {
		return nil, lookupFailure(fn, what, err)
	}
	return o, nil
}

// ------------------------------------------------------------ getConversationsService

// GetConversationsService looks up a Conversations service by SID.
type GetConversationsService struct{}

// GetConversationsServiceArgs are the lookup arguments.
type GetConversationsServiceArgs struct {
	ServiceSid string `pulumi:"serviceSid"`
}

func (a *GetConversationsServiceArgs) Annotate(an infer.Annotator) {
	an.Describe(&a.ServiceSid, "SID (IS...) of the Conversations service.")
}

// GetConversationsServiceResult is the looked-up service.
type GetConversationsServiceResult struct {
	Sid          string `pulumi:"sid"`
	FriendlyName string `pulumi:"friendlyName"`
	DateCreated  string `pulumi:"dateCreated"`
	DateUpdated  string `pulumi:"dateUpdated"`
}

func (f *GetConversationsService) Annotate(a infer.Annotator) {
	a.Describe(&f, "Look up a Conversations service by SID.")
}

func (f *GetConversationsService) Invoke(ctx context.Context, req infer.FunctionRequest[GetConversationsServiceArgs]) (infer.FunctionResponse[GetConversationsServiceResult], error) {
	const fn = "getConversationsService"
	type resp = infer.FunctionResponse[GetConversationsServiceResult]
	if err := requireArg(fn, "serviceSid", req.Input.ServiceSid); err != nil {
		return resp{}, err
	}
	o, err := cvLookup(ctx, fn, "Conversations service "+req.Input.ServiceSid, cvPath("", "Services", req.Input.ServiceSid))
	if err != nil {
		return resp{}, err
	}
	sid, created, updated := trSid(o)
	return resp{Output: GetConversationsServiceResult{Sid: sid, FriendlyName: o.str("friendly_name"), DateCreated: created, DateUpdated: updated}}, nil
}

// ------------------------------------------------------------ getConversationsServiceConfiguration

// GetConversationsServiceConfiguration looks up the configuration of a Conversations service.
type GetConversationsServiceConfiguration struct{}

// GetConversationsServiceConfigurationArgs are the lookup arguments.
type GetConversationsServiceConfigurationArgs struct {
	ChatServiceSid string `pulumi:"chatServiceSid"`
}

func (a *GetConversationsServiceConfigurationArgs) Annotate(an infer.Annotator) {
	an.Describe(&a.ChatServiceSid, "SID (IS...) of the Conversations service.")
}

// GetConversationsServiceConfigurationResult is the looked-up configuration.
type GetConversationsServiceConfigurationResult struct {
	ChatServiceSid                    string  `pulumi:"chatServiceSid"`
	DefaultConversationCreatorRoleSid *string `pulumi:"defaultConversationCreatorRoleSid,optional"`
	DefaultConversationRoleSid        *string `pulumi:"defaultConversationRoleSid,optional"`
	DefaultChatServiceRoleSid         *string `pulumi:"defaultChatServiceRoleSid,optional"`
	ReachabilityEnabled               *bool   `pulumi:"reachabilityEnabled,optional"`
}

func (f *GetConversationsServiceConfiguration) Annotate(a infer.Annotator) {
	a.Describe(&f, "Look up the configuration (default roles, reachability) of a Conversations service.")
}

func (f *GetConversationsServiceConfiguration) Invoke(ctx context.Context, req infer.FunctionRequest[GetConversationsServiceConfigurationArgs]) (infer.FunctionResponse[GetConversationsServiceConfigurationResult], error) {
	const fn = "getConversationsServiceConfiguration"
	type resp = infer.FunctionResponse[GetConversationsServiceConfigurationResult]
	if err := requireArg(fn, "chatServiceSid", req.Input.ChatServiceSid); err != nil {
		return resp{}, err
	}
	o, err := cvLookup(ctx, fn, "the configuration of Conversations service "+req.Input.ChatServiceSid, cvPath(req.Input.ChatServiceSid, "Configuration", ""))
	if err != nil {
		return resp{}, err
	}
	return resp{Output: GetConversationsServiceConfigurationResult{
		ChatServiceSid:                    req.Input.ChatServiceSid,
		DefaultConversationCreatorRoleSid: nonEmpty(o.optStr("default_conversation_creator_role_sid")),
		DefaultConversationRoleSid:        nonEmpty(o.optStr("default_conversation_role_sid")),
		DefaultChatServiceRoleSid:         nonEmpty(o.optStr("default_chat_service_role_sid")),
		ReachabilityEnabled:               o.optBool("reachability_enabled"),
	}}, nil
}

// ------------------------------------------------------------ getConversationsRole

// GetConversationsRole looks up a Conversations role by SID.
type GetConversationsRole struct{}

// GetConversationsRoleArgs are the lookup arguments.
type GetConversationsRoleArgs struct {
	Sid            string  `pulumi:"sid"`
	ChatServiceSid *string `pulumi:"chatServiceSid,optional"`
}

func (a *GetConversationsRoleArgs) Annotate(an infer.Annotator) {
	an.Describe(&a.Sid, "SID (RL...) of the role.")
	an.Describe(&a.ChatServiceSid, "SID (IS...) of the service the role belongs to; leave it out for an account-level role.")
}

// GetConversationsRoleResult is the looked-up role.
type GetConversationsRoleResult struct {
	Sid            string   `pulumi:"sid"`
	ChatServiceSid *string  `pulumi:"chatServiceSid,optional"`
	FriendlyName   string   `pulumi:"friendlyName"`
	RoleType       string   `pulumi:"roleType"`
	Permissions    []string `pulumi:"permissions"`
	DateCreated    string   `pulumi:"dateCreated"`
	DateUpdated    string   `pulumi:"dateUpdated"`
}

func (f *GetConversationsRole) Annotate(a infer.Annotator) {
	a.Describe(&f, "Look up a Conversations role by SID.")
}

func (f *GetConversationsRole) Invoke(ctx context.Context, req infer.FunctionRequest[GetConversationsRoleArgs]) (infer.FunctionResponse[GetConversationsRoleResult], error) {
	const fn = "getConversationsRole"
	type resp = infer.FunctionResponse[GetConversationsRoleResult]
	in := req.Input
	if err := requireArg(fn, "sid", in.Sid); err != nil {
		return resp{}, err
	}
	o, err := cvLookup(ctx, fn, "Conversations role "+in.Sid, cvPath(derefOr(in.ChatServiceSid, ""), "Roles", in.Sid))
	if err != nil {
		return resp{}, err
	}
	sid, created, updated := trSid(o)
	perms := []string{}
	if l := o.optList("permissions"); l != nil {
		perms = *l
	}
	return resp{Output: GetConversationsRoleResult{
		Sid: sid, ChatServiceSid: nonEmpty(o.optStr("chat_service_sid")), FriendlyName: o.str("friendly_name"),
		RoleType: o.str("type"), Permissions: perms, DateCreated: created, DateUpdated: updated,
	}}, nil
}

// ------------------------------------------------------------ getConversationsCredential

// GetConversationsCredential looks up a Conversations push credential by SID.
type GetConversationsCredential struct{}

// GetConversationsCredentialArgs are the lookup arguments.
type GetConversationsCredentialArgs struct {
	Sid string `pulumi:"sid"`
}

func (a *GetConversationsCredentialArgs) Annotate(an infer.Annotator) {
	an.Describe(&a.Sid, "SID (CR...) of the credential.")
}

// GetConversationsCredentialResult is the looked-up credential; the certificate, keys and secret are never returned.
type GetConversationsCredentialResult struct {
	Sid            string  `pulumi:"sid"`
	CredentialType string  `pulumi:"credentialType"`
	FriendlyName   *string `pulumi:"friendlyName,optional"`
	Sandbox        *bool   `pulumi:"sandbox,optional"`
	DateCreated    string  `pulumi:"dateCreated"`
	DateUpdated    string  `pulumi:"dateUpdated"`
}

func (f *GetConversationsCredential) Annotate(a infer.Annotator) {
	a.Describe(&f, "Look up a Conversations push credential by SID. The certificate, keys and secret are never returned.")
}

func (f *GetConversationsCredential) Invoke(ctx context.Context, req infer.FunctionRequest[GetConversationsCredentialArgs]) (infer.FunctionResponse[GetConversationsCredentialResult], error) {
	const fn = "getConversationsCredential"
	type resp = infer.FunctionResponse[GetConversationsCredentialResult]
	if err := requireArg(fn, "sid", req.Input.Sid); err != nil {
		return resp{}, err
	}
	o, err := cvLookup(ctx, fn, "Conversations credential "+req.Input.Sid, cvPath("", "Credentials", req.Input.Sid))
	if err != nil {
		return resp{}, err
	}
	sid, created, updated := trSid(o)
	return resp{Output: GetConversationsCredentialResult{
		Sid: sid, CredentialType: o.str("type"), FriendlyName: nonEmpty(o.optStr("friendly_name")),
		Sandbox: o.optBool("sandbox"), DateCreated: created, DateUpdated: updated,
	}}, nil
}

// ------------------------------------------------------------ getConversationsAddressConfiguration

// GetConversationsAddressConfiguration looks up a Conversations address configuration by SID.
type GetConversationsAddressConfiguration struct{}

// GetConversationsAddressConfigurationArgs are the lookup arguments.
type GetConversationsAddressConfigurationArgs struct {
	Sid string `pulumi:"sid"`
}

func (a *GetConversationsAddressConfigurationArgs) Annotate(an infer.Annotator) {
	an.Describe(&a.Sid, "SID (IG...) of the address configuration.")
}

// GetConversationsAddressConfigurationResult is the looked-up address configuration.
type GetConversationsAddressConfigurationResult struct {
	Sid            string                     `pulumi:"sid"`
	AddressType    string                     `pulumi:"addressType"`
	Address        string                     `pulumi:"address"`
	FriendlyName   *string                    `pulumi:"friendlyName,optional"`
	AddressCountry *string                    `pulumi:"addressCountry,optional"`
	AutoCreation   *ConversationsAutoCreation `pulumi:"autoCreation,optional"`
	DateCreated    string                     `pulumi:"dateCreated"`
	DateUpdated    string                     `pulumi:"dateUpdated"`
}

func (f *GetConversationsAddressConfiguration) Annotate(a infer.Annotator) {
	a.Describe(&f, "Look up a Conversations address configuration by SID.")
}

func (f *GetConversationsAddressConfiguration) Invoke(ctx context.Context, req infer.FunctionRequest[GetConversationsAddressConfigurationArgs]) (infer.FunctionResponse[GetConversationsAddressConfigurationResult], error) {
	const fn = "getConversationsAddressConfiguration"
	type resp = infer.FunctionResponse[GetConversationsAddressConfigurationResult]
	if err := requireArg(fn, "sid", req.Input.Sid); err != nil {
		return resp{}, err
	}
	o, err := cvLookup(ctx, fn, "Conversations address configuration "+req.Input.Sid, cvPath("", "Configuration/Addresses", req.Input.Sid))
	if err != nil {
		return resp{}, err
	}
	sid, created, updated := trSid(o)
	return resp{Output: GetConversationsAddressConfigurationResult{
		Sid: sid, AddressType: o.str("type"), Address: o.str("address"), FriendlyName: nonEmpty(o.optStr("friendly_name")),
		AddressCountry: nonEmpty(o.optStr("address_country")), AutoCreation: cvAutoCreation(nil, o, true),
		DateCreated: created, DateUpdated: updated,
	}}, nil
}

// ------------------------------------------------------------ getConversationsWebhookConfiguration

// GetConversationsWebhookConfiguration looks up the webhook configuration of Conversations.
type GetConversationsWebhookConfiguration struct{}

// GetConversationsWebhookConfigurationArgs are the lookup arguments.
type GetConversationsWebhookConfigurationArgs struct {
	ChatServiceSid *string `pulumi:"chatServiceSid,optional"`
}

func (a *GetConversationsWebhookConfigurationArgs) Annotate(an infer.Annotator) {
	an.Describe(&a.ChatServiceSid, "SID (IS...) of the Conversations service; leave it out for the account-level configuration.")
}

// GetConversationsWebhookConfigurationResult is the looked-up webhook configuration.
type GetConversationsWebhookConfigurationResult struct {
	ChatServiceSid *string  `pulumi:"chatServiceSid,optional"`
	Method         *string  `pulumi:"method,optional"`
	Filters        []string `pulumi:"filters"`
	PreWebhookUrl  *string  `pulumi:"preWebhookUrl,optional"`
	PostWebhookUrl *string  `pulumi:"postWebhookUrl,optional"`
	Target         *string  `pulumi:"target,optional"`
}

func (f *GetConversationsWebhookConfiguration) Annotate(a infer.Annotator) {
	a.Describe(&f, "Look up the webhook configuration of Conversations, account-level or of one service.")
}

func (f *GetConversationsWebhookConfiguration) Invoke(ctx context.Context, req infer.FunctionRequest[GetConversationsWebhookConfigurationArgs]) (infer.FunctionResponse[GetConversationsWebhookConfigurationResult], error) {
	const fn = "getConversationsWebhookConfiguration"
	type resp = infer.FunctionResponse[GetConversationsWebhookConfigurationResult]
	id, what := cvAccountWebhookID, "the account-level Conversations webhook configuration"
	if s := derefOr(req.Input.ChatServiceSid, ""); s != "" {
		id, what = s, "the webhook configuration of Conversations service "+s
	}
	o, err := cvLookup(ctx, fn, what, cvWebhookPath(id))
	if err != nil {
		return resp{}, err
	}
	filters := []string{}
	if l := o.optList("filters"); l != nil {
		filters = *l
	}
	return resp{Output: GetConversationsWebhookConfigurationResult{
		ChatServiceSid: nonEmpty(req.Input.ChatServiceSid), Method: nonEmpty(o.optStr("method")), Filters: filters,
		PreWebhookUrl: nonEmpty(o.optStr("pre_webhook_url")), PostWebhookUrl: nonEmpty(o.optStr("post_webhook_url")),
		Target: nonEmpty(o.optStr("target")),
	}}, nil
}
