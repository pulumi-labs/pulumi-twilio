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

	"github.com/pulumi/pulumi-go-provider/infer"
)

// Config holds the provider-level configuration.
//
// Twilio accepts two kinds of REST credentials, both over HTTP Basic auth:
// the account SID with the account auth token, or an API key SID with its
// secret (the account SID is then still required because it appears in
// request paths). When both are set the API key is used.
type Config struct {
	// AccountSid is the SID (AC...) of the account that owns the resources.
	AccountSid *string `pulumi:"accountSid,optional"`

	// AuthToken is the account auth token.
	AuthToken *string `pulumi:"authToken,optional" provider:"secret"`

	// APIKey is an API key SID (SK...), used instead of the auth token.
	APIKey *string `pulumi:"apiKey,optional"`

	// APISecret is the secret of APIKey.
	APISecret *string `pulumi:"apiSecret,optional" provider:"secret"`

	// Region is the Twilio region for data residency, for example "au1" or "ie1".
	Region *string `pulumi:"region,optional"`

	// Edge is the Twilio edge location, for example "sydney" or "dublin".
	Edge *string `pulumi:"edge,optional"`

	// BaseURL overrides every Twilio host; intended for testing against a fake.
	BaseURL *string `pulumi:"baseUrl,optional"`
}

// Annotate describes the config fields and wires their environment variable
// fallbacks; infer reads env defaults only from SetDefault, not from struct tags.
func (c *Config) Annotate(a infer.Annotator) {
	a.SetDefault(&c.AccountSid, nil, "TWILIO_ACCOUNT_SID")
	a.SetDefault(&c.AuthToken, nil, "TWILIO_AUTH_TOKEN")
	a.SetDefault(&c.APIKey, nil, "TWILIO_API_KEY")
	a.SetDefault(&c.APISecret, nil, "TWILIO_API_SECRET")
	a.SetDefault(&c.Region, nil, "TWILIO_REGION")
	a.SetDefault(&c.Edge, nil, "TWILIO_EDGE")
	a.SetDefault(&c.BaseURL, nil, "TWILIO_BASE_URL")
	a.Describe(&c.AccountSid, "The SID (AC...) of the Twilio account that owns the resources. Required. "+
		"Falls back to the TWILIO_ACCOUNT_SID environment variable.")
	a.Describe(&c.AuthToken, "The account auth token. Use either authToken or apiKey with apiSecret. "+
		"Falls back to the TWILIO_AUTH_TOKEN environment variable.")
	a.Describe(&c.APIKey, "An API key SID (SK...) to authenticate with instead of the auth token; requires apiSecret. "+
		"Falls back to the TWILIO_API_KEY environment variable. A restricted key cannot manage Subaccount or ApiKey resources.")
	a.Describe(&c.APISecret, "The secret of apiKey. Falls back to the TWILIO_API_SECRET environment variable.")
	a.Describe(&c.Region, "Twilio region for data residency, for example au1 or ie1. Falls back to TWILIO_REGION.")
	a.Describe(&c.Edge, "Twilio edge location, for example sydney or dublin. Falls back to TWILIO_EDGE.")
	a.Describe(&c.BaseURL, "Override for every Twilio host (requests go to <baseUrl>/<product>/<path>). "+
		"Intended for testing against a mock server. Falls back to TWILIO_BASE_URL.")
}

func deref(p *string) string {
	if p == nil {
		return ""
	}
	return *p
}

func derefOr(p *string, def string) string {
	if p == nil || *p == "" {
		return def
	}
	return *p
}

// clientFor builds a Client from the provider configuration in ctx.
func clientFor(ctx context.Context) (*Client, error) {
	cfg := infer.GetConfig[Config](ctx)
	return clientFromConfig(&cfg)
}

func clientFromConfig(cfg *Config) (*Client, error) {
	sid := deref(cfg.AccountSid)
	if sid == "" {
		return nil, fmt.Errorf("twilio: accountSid is required (set the provider accountSid or TWILIO_ACCOUNT_SID)")
	}
	user, pass := sid, deref(cfg.AuthToken)
	if k := deref(cfg.APIKey); k != "" || deref(cfg.APISecret) != "" {
		if k == "" || deref(cfg.APISecret) == "" {
			return nil, fmt.Errorf("twilio: apiKey and apiSecret must be set together (TWILIO_API_KEY and TWILIO_API_SECRET)")
		}
		user, pass = k, deref(cfg.APISecret)
	}
	if pass == "" {
		return nil, fmt.Errorf("twilio: credentials are required: set authToken (TWILIO_AUTH_TOKEN) or apiKey and apiSecret (TWILIO_API_KEY and TWILIO_API_SECRET)")
	}
	if err := checkBaseURL(deref(cfg.BaseURL)); err != nil {
		return nil, fmt.Errorf("twilio: baseUrl: %w", err)
	}
	if deref(cfg.Edge) != "" && deref(cfg.Region) == "" {
		return nil, fmt.Errorf("twilio: edge requires region (set the provider region or TWILIO_REGION)")
	}
	return newClient(sid, user, pass, deref(cfg.Edge), deref(cfg.Region), deref(cfg.BaseURL)), nil
}

// checkBaseURL refuses a plain-HTTP base URL because every request carries
// credentials as HTTP Basic auth. Loopback hosts are allowed for local fakes.
func checkBaseURL(raw string) error {
	if raw == "" {
		return nil
	}
	u, err := url.Parse(raw)
	if err != nil {
		return fmt.Errorf("invalid URL %q: %w", raw, err)
	}
	if u.Scheme == "https" {
		return nil
	}
	if u.Scheme == "http" {
		switch u.Hostname() {
		case "localhost", "127.0.0.1", "::1":
			return nil
		}
	}
	return fmt.Errorf("%q must use https (credentials are sent with every request); plain http is only accepted for loopback hosts", raw)
}
