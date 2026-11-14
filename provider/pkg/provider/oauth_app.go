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
	"fmt"
	"net/url"
	"regexp"
	"slices"

	p "github.com/pulumi/pulumi-go-provider"
	"github.com/pulumi/pulumi-go-provider/infer"
)

// OAuth apps of the IAM API (iam.twilio.com, twilio-oai twilio_iam_v1 1.0.0): POST /v1/Account/OAuthApps
// creates one and DELETE /v1/Account/OAuthApps/{sid} removes it. The API has no fetch and no update,
// so every input replaces the app, a refresh keeps the stored state, and an import cannot read the app.

const (
	iamProduct = "iam"

	// oauthAppNoReadMessage is the one message of every operation that would need to read an OAuth app.
	oauthAppNoReadMessage = "the Twilio API does not support reading OAuth apps"
)

var (
	oauthAppOwnerSidRE  = regexp.MustCompile(`^[A-Z]{2}[0-9a-fA-F]{32}$`)
	oauthAppClientSidRE = regexp.MustCompile(`^OQ[0-9a-fA-F]{32}$`)
)

// oauthAppGrantType is the only type Twilio's create call accepts for an account OAuth app.
const oauthAppGrantType = "Client Credential"

// OAuthApp is an OAuth application of the Twilio account. The client secret is returned once, at creation.
type OAuthApp struct{}

// OAuthAppPolicy is the set of permissions the app is explicitly allowed or denied.
type OAuthAppPolicy struct {
	Allow *[]string `pulumi:"allow,optional"`
	Deny  *[]string `pulumi:"deny,optional"`
}

func (a *OAuthAppPolicy) Annotate(an infer.Annotator) {
	an.Describe(&a.Allow, "Permissions explicitly allowed, for example /twilio/iam/api-keys/create.")
	an.Describe(&a.Deny, "Permissions explicitly denied.")
}

// OAuthAppArgs are the inputs of an OAuthApp.
type OAuthAppArgs struct {
	FriendlyName            string          `pulumi:"friendlyName"`
	Description             *string         `pulumi:"description,optional"`
	Type                    *string         `pulumi:"type,optional"`
	OwnerSid                *string         `pulumi:"ownerSid,optional"`
	ClientSid               *string         `pulumi:"clientSid,optional"`
	TokenEndpointAuthMethod *string         `pulumi:"tokenEndpointAuthMethod,optional"`
	Policy                  *OAuthAppPolicy `pulumi:"policy,optional"`
	AccessTokenTtl          *int            `pulumi:"accessTokenTtl,optional"`
}

func (a *OAuthAppArgs) Annotate(an infer.Annotator) {
	an.Describe(&a.FriendlyName, "Human-readable name of the OAuth app. Changing it replaces the app, because Twilio has no update call for OAuth apps.")
	an.Describe(&a.Description, "A description of the OAuth app. Changing it replaces the app.")
	an.Describe(&a.Type, "The OAuth grant type of the app. Twilio's only accepted value is \"Client Credential\" (the API specification does not list it, the Twilio docs do), which is the default; any other value, such as confidential or m2m, is refused by Twilio. Changing it replaces the app.")
	an.Describe(&a.OwnerSid, "SID (two capital letters and 32 hex characters) of the account that owns the app. Twilio requires it to match the account the provider authenticates with, which is the default. Changing it replaces the app.")
	an.Describe(&a.ClientSid, "SID (OQ...) to give the app as its client SID. Changing it replaces the app.")
	an.Describe(&a.TokenEndpointAuthMethod, "How the client authenticates at the token endpoint. Account OAuth apps support only client_secret_basic. Changing it replaces the app.")
	an.Describe(&a.Policy, "The permissions the app is allowed and denied. Changing it replaces the app.")
	an.Describe(&a.AccessTokenTtl, "Lifetime of the access tokens the app issues, as Twilio takes it (the API specification does not state the unit). Changing it replaces the app.")
}

// OAuthAppState is the persisted state of an OAuthApp.
type OAuthAppState struct {
	OAuthAppArgs

	Sid         string `pulumi:"sid"`
	Secret      string `pulumi:"secret" provider:"secret"`
	Status      string `pulumi:"status"`
	CreatedBy   string `pulumi:"createdBy"`
	CreatorSid  string `pulumi:"creatorSid"`
	DateCreated string `pulumi:"dateCreated"`
}

func (s *OAuthAppState) Annotate(a infer.Annotator) {
	a.Describe(&s.Sid, "The OAuth app SID (OQ...), which is also its client ID.")
	a.Describe(&s.Secret, "The client secret. Twilio returns it only once, at creation, so it cannot be recovered later. It is stored as a Pulumi secret.")
	a.Describe(&s.Status, "The status of the app, as Twilio reported it at creation.")
	a.Describe(&s.CreatedBy, "Who created the app, as Twilio reported it at creation.")
	a.Describe(&s.CreatorSid, "SID (US...) of the user who created the app.")
	a.Describe(&s.DateCreated, "When the app was created (ISO 8601).")
}

func (r *OAuthApp) Annotate(a infer.Annotator) {
	a.Describe(&r, "A Twilio OAuth app (iam.twilio.com /v1/Account/OAuthApps). The resource ID is the app SID (OQ...). The Twilio API can create and delete OAuth apps but not read or update them: every input replaces the app, a refresh keeps the stored state (a deletion outside Pulumi is noticed only when the app is destroyed or replaced), and an app cannot be imported. The client secret is returned once, at creation, and is stored as a Pulumi secret.")
}

var _ infer.CustomResource[OAuthAppArgs, OAuthAppState] = (*OAuthApp)(nil)
var _ infer.CustomCheck[OAuthAppArgs] = (*OAuthApp)(nil)
var _ infer.CustomDiff[OAuthAppArgs, OAuthAppState] = (*OAuthApp)(nil)
var _ infer.CustomRead[OAuthAppArgs, OAuthAppState] = (*OAuthApp)(nil)
var _ infer.CustomDelete[OAuthAppState] = (*OAuthApp)(nil)

func (r *OAuthApp) Check(ctx context.Context, req infer.CheckRequest) (infer.CheckResponse[OAuthAppArgs], error) {
	args, failures, err := infer.DefaultCheck[OAuthAppArgs](ctx, req.NewInputs)
	if err != nil || len(failures) > 0 {
		return infer.CheckResponse[OAuthAppArgs]{Inputs: args, Failures: failures}, err
	}
	c := &checker{unk: unknowns(req.NewInputs)}
	c.required("friendlyName", args.FriendlyName)
	if args.OwnerSid != nil {
		c.sid("ownerSid", *args.OwnerSid, oauthAppOwnerSidRE, "Twilio resource (two capital letters followed by 32 hex characters)")
	}
	if args.ClientSid != nil {
		c.sid("clientSid", *args.ClientSid, oauthAppClientSidRE, "OAuth app (OQ followed by 32 hex characters)")
	}
	if args.TokenEndpointAuthMethod != nil && !c.unk("tokenEndpointAuthMethod") {
		c.oneOf("tokenEndpointAuthMethod", *args.TokenEndpointAuthMethod, []string{"client_secret_basic"})
	}
	if args.AccessTokenTtl != nil && !c.unk("accessTokenTtl") && *args.AccessTokenTtl < 1 {
		c.fail("accessTokenTtl", "accessTokenTtl must be greater than 0")
	}
	return infer.CheckResponse[OAuthAppArgs]{Inputs: args, Failures: c.failures}, nil
}

// oauthAppPolicySame compares two policies; a missing list and an empty one are the same.
func oauthAppPolicySame(o, n *OAuthAppPolicy) bool {
	list := func(p *OAuthAppPolicy, deny bool) []string {
		if p == nil {
			return nil
		}
		l := p.Allow
		if deny {
			l = p.Deny
		}
		if l == nil {
			return nil
		}
		return *l
	}
	return sameSet(list(o, false), list(n, false)) && sameSet(list(o, true), list(n, true))
}

// Diff: Twilio can neither read nor update an OAuth app, so a changed input replaces it. Removing an
// optional input is no change, because the stored value is what the program last stated.
func (r *OAuthApp) Diff(_ context.Context, req infer.DiffRequest[OAuthAppArgs, OAuthAppState]) (infer.DiffResponse, error) {
	o, n := req.State.OAuthAppArgs, req.Inputs
	d := trDiffSet{}
	d.replace("friendlyName", o.FriendlyName, n.FriendlyName)
	d.replaceStr("description", o.Description, n.Description)
	d.replaceStr("type", o.Type, n.Type)
	d.replaceStr("ownerSid", o.OwnerSid, n.OwnerSid)
	d.replaceStr("clientSid", o.ClientSid, n.ClientSid)
	d.replaceStr("tokenEndpointAuthMethod", o.TokenEndpointAuthMethod, n.TokenEndpointAuthMethod)
	d.replaceInt("accessTokenTtl", o.AccessTokenTtl, n.AccessTokenTtl)
	if n.Policy != nil && !oauthAppPolicySame(o.Policy, n.Policy) {
		d["policy"] = p.PropertyDiff{Kind: p.UpdateReplace}
	}
	return d.response(), nil
}

func (r *OAuthApp) Create(ctx context.Context, req infer.CreateRequest[OAuthAppArgs]) (infer.CreateResponse[OAuthAppState], error) {
	type resp = infer.CreateResponse[OAuthAppState]
	a := req.Inputs
	st := OAuthAppState{OAuthAppArgs: a}
	if req.DryRun {
		return resp{Output: st}, nil
	}
	c, err := clientFor(ctx)
	if err != nil {
		return resp{}, err
	}
	body := map[string]any{"friendly_name": a.FriendlyName}
	if a.Description != nil {
		body["description"] = *a.Description
	}
	// Twilio refuses a create without a type ("type cannot be null") and accepts only "Client Credential".
	grant := oauthAppGrantType
	if a.Type != nil {
		grant = *a.Type
	}
	body["type"] = grant
	st.Type = &grant
	if a.OwnerSid != nil {
		body["owner_sid"] = *a.OwnerSid
	} else if owner := c.acct(nil); owner != "" {
		body["owner_sid"] = owner
	}
	if a.ClientSid != nil {
		body["client_sid"] = *a.ClientSid
	}
	if a.TokenEndpointAuthMethod != nil {
		body["token_endpoint_auth_method"] = *a.TokenEndpointAuthMethod
	}
	if a.AccessTokenTtl != nil {
		body["access_token_ttl"] = *a.AccessTokenTtl
	}
	if a.Policy != nil {
		pol := map[string]any{}
		if a.Policy.Allow != nil {
			pol["allow"] = slices.Clone(*a.Policy.Allow)
		}
		if a.Policy.Deny != nil {
			pol["deny"] = slices.Clone(*a.Policy.Deny)
		}
		body["policy"] = pol
	}
	o, err := c.doJSON(ctx, "POST", iamProduct, "/v1/Account/OAuthApps", nil, body)
	if err != nil {
		return resp{}, fmt.Errorf("create OAuth app: %w", err)
	}
	st.Sid, st.Secret, st.Status = o.str("sid"), o.str("secret"), o.str("status")
	st.CreatedBy, st.CreatorSid, st.DateCreated = o.str("created_by"), o.str("creator_sid"), o.str("date_created")
	if st.Sid == "" {
		return resp{}, fmt.Errorf("create OAuth app: Twilio returned no SID")
	}
	return resp{ID: st.Sid, Output: st}, nil
}

// Read keeps the stored state: there is nothing to fetch. An import has no stored state, so it fails.
func (r *OAuthApp) Read(_ context.Context, req infer.ReadRequest[OAuthAppArgs, OAuthAppState]) (infer.ReadResponse[OAuthAppArgs, OAuthAppState], error) {
	type resp = infer.ReadResponse[OAuthAppArgs, OAuthAppState]
	// An import has neither inputs nor state (notFoundOnImport recognises it); an app that has no
	// stored SID cannot be refreshed either, because Twilio has no call that returns it.
	if notFoundOnImport("OAuth app", req.ID, req.Inputs, req.State) != nil || req.State.Sid == "" {
		return resp{}, fmt.Errorf("cannot import OAuth app %q: %s", req.ID, oauthAppNoReadMessage)
	}
	return resp{ID: req.ID, Inputs: req.Inputs, State: req.State}, nil
}

func (r *OAuthApp) Delete(ctx context.Context, req infer.DeleteRequest[OAuthAppState]) (infer.DeleteResponse, error) {
	c, err := clientFor(ctx)
	if err != nil {
		return infer.DeleteResponse{}, err
	}
	if _, err := c.do(ctx, "DELETE", iamProduct, "/v1/Account/OAuthApps/"+url.PathEscape(req.ID), nil, nil); err != nil && !isNotFound(err) {
		return infer.DeleteResponse{}, fmt.Errorf("delete OAuth app %s: %w", req.ID, err)
	}
	return infer.DeleteResponse{}, nil
}
