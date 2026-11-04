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

// GetApiKey looks up an API key. The secret is never returned (Twilio shows it once, at creation).
type GetApiKey struct{}

// GetApiKeyArgs are the lookup arguments.
type GetApiKeyArgs struct {
	KeySid     string  `pulumi:"keySid"`
	AccountSid *string `pulumi:"accountSid,optional"`
}

func (a *GetApiKeyArgs) Annotate(an infer.Annotator) {
	an.Describe(&a.KeySid, "SID (SK...) of the API key.")
	an.Describe(&a.AccountSid, "SID of the account that owns the key. Defaults to the provider account.")
}

// GetApiKeyResult is the looked-up key.
type GetApiKeyResult struct {
	Sid          string `pulumi:"sid"`
	FriendlyName string `pulumi:"friendlyName"`
	KeyType      string `pulumi:"keyType"`
	Policy       string `pulumi:"policy"`
	DateCreated  string `pulumi:"dateCreated"`
}

func (a *GetApiKeyResult) Annotate(an infer.Annotator) {
	an.Describe(&a.Sid, "The key SID (SK...).")
	an.Describe(&a.FriendlyName, "Name of the key.")
	an.Describe(&a.KeyType, "restricted when the key has a policy, otherwise standard.")
	an.Describe(&a.Policy, "The policy of a restricted key as JSON (empty for a standard key, or when only the account-explicit route could read the key).")
	an.Describe(&a.DateCreated, "When the key was created (RFC 2822).")
}

func (f *GetApiKey) Annotate(a infer.Annotator) {
	a.Describe(&f, "Look up a Twilio API key by SID. The secret is never returned.")
}

func (f *GetApiKey) Invoke(ctx context.Context, req infer.FunctionRequest[GetApiKeyArgs]) (infer.FunctionResponse[GetApiKeyResult], error) {
	const fn = "getApiKey"
	if err := requireArg(fn, "keySid", req.Input.KeySid); err != nil {
		return infer.FunctionResponse[GetApiKeyResult]{}, err
	}
	c, err := clientFor(ctx)
	if err != nil {
		return infer.FunctionResponse[GetApiKeyResult]{}, err
	}
	acct := c.acct(req.Input.AccountSid)
	o, _, err := fetchKey(ctx, c, acct, req.Input.KeySid)
	if err != nil {
		return infer.FunctionResponse[GetApiKeyResult]{}, lookupFailure(fn, "API key "+req.Input.KeySid, err)
	}
	pol := policyJSON(o)
	kt := "standard"
	if pol != "" {
		kt = "restricted"
	}
	return infer.FunctionResponse[GetApiKeyResult]{Output: GetApiKeyResult{
		Sid: o.str("sid"), FriendlyName: o.str("friendly_name"), KeyType: kt, Policy: pol, DateCreated: o.str("date_created"),
	}}, nil
}
