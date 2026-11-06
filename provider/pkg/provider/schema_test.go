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

package provider_test

import (
	"context"
	"encoding/json"
	"testing"

	"github.com/blang/semver"
	pgp "github.com/pulumi/pulumi-go-provider"
	"github.com/pulumi/pulumi-go-provider/integration"
	"github.com/pulumi/pulumi/pkg/v3/codegen/schema"
	"github.com/pulumi/pulumi/sdk/v3/go/common/resource"
	"github.com/pulumi/pulumi/sdk/v3/go/property"
	"github.com/stretchr/testify/require"

	"github.com/pulumi-labs/pulumi-twilio/provider/pkg/provider"
)

func mustSchema(t *testing.T) schema.PackageSpec {
	t.Helper()
	prov, err := provider.New()
	require.NoError(t, err)
	spec, err := pgp.GetSchema(context.Background(), "twilio", "0.1.0", prov)
	require.NoError(t, err)
	return spec
}

func TestSchema_Tokens(t *testing.T) {
	spec := mustSchema(t)
	for _, r := range []string{"Subaccount", "ApiKey", "IncomingPhoneNumber", "Application", "MessagingService", "MessagingServicePhoneNumber", "VerifyService"} {
		require.Contains(t, spec.Resources, "twilio:index:"+r)
	}
	for _, f := range []string{"getAccount", "getIncomingPhoneNumber"} {
		require.Contains(t, spec.Functions, "twilio:index:"+f)
	}
}

func TestSchema_ReplaceOnChangesForIdentityInputs(t *testing.T) {
	spec := mustSchema(t)
	want := map[string][]string{
		"twilio:index:ApiKey":                      {"keyType", "accountSid"},
		"twilio:index:IncomingPhoneNumber":         {"phoneNumber", "phoneNumberSid"},
		"twilio:index:Application":                 {"accountSid"},
		"twilio:index:MessagingServicePhoneNumber": {"serviceSid", "phoneNumberSid"},
	}
	for tok, props := range want {
		for _, p := range props {
			require.True(t, spec.Resources[tok].InputProperties[p].ReplaceOnChanges, "%s.%s must be replaceOnChanges", tok, p)
		}
	}
	// Renames and webhook edits are in-place updates.
	for tok, props := range map[string][]string{
		"twilio:index:Subaccount":       {"friendlyName", "status"},
		"twilio:index:Application":      {"friendlyName", "voiceUrl"},
		"twilio:index:MessagingService": {"friendlyName", "inboundRequestUrl"},
		"twilio:index:ApiKey":           {"friendlyName", "policy"},
		// B5: Twilio moves a number between accounts in place (AccountSid update parameter).
		"twilio:index:IncomingPhoneNumber": {"accountSid", "friendlyName", "voiceUrl"},
	} {
		for _, p := range props {
			require.False(t, spec.Resources[tok].InputProperties[p].ReplaceOnChanges, "%s.%s must not be replaceOnChanges", tok, p)
		}
	}
}

func TestSchema_SecretsAreMarked(t *testing.T) {
	spec := mustSchema(t)
	require.True(t, spec.Resources["twilio:index:Subaccount"].Properties["authToken"].Secret)
	require.True(t, spec.Resources["twilio:index:ApiKey"].Properties["secret"].Secret)
	for _, k := range []string{"authToken", "apiSecret"} {
		require.True(t, spec.Config.Variables[k].Secret, "config %s must be secret", k)
	}
}

func TestSchema_Metadata(t *testing.T) {
	spec := mustSchema(t)
	require.Equal(t, "https://github.com/pulumi-labs/pulumi-twilio", spec.Repository)
	require.Equal(t, "Twilio", spec.DisplayName)
	require.NotEmpty(t, spec.Homepage)
	require.Equal(t, "pulumi", spec.Keywords[0])
}

// ---------------------------------------------------------------------------
// CheckConfig: environment variable fallbacks
// ---------------------------------------------------------------------------

// envConfigCase describes one provider config key and the environment
// variables (in precedence order) that back it.
type envConfigCase struct {
	key     string
	envVars []string
	secret  bool
}

var envConfigCases = []envConfigCase{
	{key: "accountSid", envVars: []string{"TWILIO_ACCOUNT_SID"}, secret: false},
	{key: "authToken", envVars: []string{"TWILIO_AUTH_TOKEN"}, secret: true},
	{key: "apiKey", envVars: []string{"TWILIO_API_KEY"}, secret: false},
	{key: "apiSecret", envVars: []string{"TWILIO_API_SECRET"}, secret: true},
	{key: "region", envVars: []string{"TWILIO_REGION"}, secret: false},
	{key: "edge", envVars: []string{"TWILIO_EDGE"}, secret: false},
	{key: "baseUrl", envVars: []string{"TWILIO_BASE_URL"}, secret: false},
}

// allEnvVars lists every environment variable the provider reads.
func allEnvVars() []string {
	return []string{"TWILIO_ACCOUNT_SID", "TWILIO_API_KEY", "TWILIO_API_SECRET", "TWILIO_AUTH_TOKEN", "TWILIO_BASE_URL", "TWILIO_EDGE", "TWILIO_REGION"}
}

// newEnvTestServer builds an integration server for the provider and clears
// every provider environment variable so each case starts hermetic.
func newEnvTestServer(t *testing.T) integration.Server {
	t.Helper()
	for _, v := range allEnvVars() {
		t.Setenv(v, "")
	}
	p, err := provider.New()
	require.NoError(t, err)
	s, err := integration.NewServer(t.Context(), "twilio", semver.MustParse("0.0.1"), integration.WithProvider(p))
	require.NoError(t, err)
	return s
}

// checkConfigValue runs CheckConfig with cfg and returns the plain string value of key.
func checkConfigValue(t *testing.T, s integration.Server, cfg map[string]property.Value, key string) string {
	t.Helper()
	resp, err := s.CheckConfig(pgp.CheckRequest{
		Urn:    resource.NewURN("stack", "proj", "", "pulumi:providers:twilio", "default"),
		Inputs: property.NewMap(cfg),
	})
	require.NoError(t, err)
	require.Empty(t, resp.Failures)
	got, ok := resp.Inputs.GetOk(key)
	require.True(t, ok, "%s should be populated", key)
	if got.Secret() {
		got = got.WithSecret(false)
	}
	return got.AsString()
}

// TestCheckConfig_EnvFallbacks verifies that unset provider config is filled
// from the environment, that explicit config wins, and that the first
// non-empty environment variable wins when a key has several.
func TestCheckConfig_EnvFallbacks(t *testing.T) {
	type tc struct {
		name   string
		key    string
		env    map[string]string
		config map[string]property.Value
		want   string
	}
	var tests []tc
	for _, c := range envConfigCases {
		for _, v := range c.envVars {
			tests = append(tests, tc{name: c.key + " from " + v, key: c.key, env: map[string]string{v: "from-" + v}, want: "from-" + v})
		}
		if len(c.envVars) > 1 {
			env := map[string]string{}
			for _, v := range c.envVars {
				env[v] = "from-" + v
			}
			tests = append(tests, tc{name: c.key + " first env var wins", key: c.key, env: env, want: "from-" + c.envVars[0]})
		}
		tests = append(tests, tc{
			name:   c.key + " explicit config wins over env",
			key:    c.key,
			env:    map[string]string{c.envVars[0]: "from-env"},
			config: map[string]property.Value{c.key: property.New("from-config")},
			want:   "from-config",
		})
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			s := newEnvTestServer(t)
			for k, v := range tt.env {
				t.Setenv(k, v)
			}
			require.Equal(t, tt.want, checkConfigValue(t, s, tt.config, tt.key))
		})
	}
}

// TestCheckConfig_EnvSecrets verifies that env-sourced values for secret
// config keys are returned marked secret.
func TestCheckConfig_EnvSecrets(t *testing.T) {
	for _, c := range envConfigCases {
		if !c.secret {
			continue
		}
		t.Run(c.key, func(t *testing.T) {
			s := newEnvTestServer(t)
			t.Setenv(c.envVars[0], "from-env")
			resp, err := s.CheckConfig(pgp.CheckRequest{
				Urn:    resource.NewURN("stack", "proj", "", "pulumi:providers:twilio", "default"),
				Inputs: property.NewMap(nil),
			})
			require.NoError(t, err)
			got, ok := resp.Inputs.GetOk(c.key)
			require.True(t, ok)
			require.True(t, got.Secret(), "%s sourced from env should be secret", c.key)
		})
	}
}

// TestSchema_ConfigEnvironmentDefaults verifies the generated schema advertises
// the environment variable fallbacks for every config key.
func TestSchema_ConfigEnvironmentDefaults(t *testing.T) {
	s := newEnvTestServer(t)
	resp, err := s.GetSchema(pgp.GetSchemaRequest{})
	require.NoError(t, err)

	var spec struct {
		Config struct {
			Variables map[string]struct {
				DefaultInfo struct {
					Environment []string `json:"environment"`
				} `json:"defaultInfo"`
			} `json:"variables"`
		} `json:"config"`
	}
	require.NoError(t, json.Unmarshal([]byte(resp.Schema), &spec))
	for _, c := range envConfigCases {
		v, ok := spec.Config.Variables[c.key]
		require.True(t, ok, "config.variables.%s missing from schema", c.key)
		require.Equal(t, c.envVars, v.DefaultInfo.Environment, "config.variables.%s.defaultInfo.environment", c.key)
	}
}

// pulumi-go-provider drops an input named "version", and a function argument or
// result named "id" collides with the resource ID convention.
func TestSchema_NoReservedPropertyNames(t *testing.T) {
	spec := mustSchema(t)
	for tok, r := range spec.Resources {
		require.NotContains(t, r.InputProperties, "version", "%s", tok)
		require.NotContains(t, r.InputProperties, "id", "%s", tok)
	}
	for tok, f := range spec.Functions {
		if f.Inputs != nil {
			require.NotContains(t, f.Inputs.Properties, "id", "%s args", tok)
			require.NotContains(t, f.Inputs.Properties, "version", "%s args", tok)
		}
		if f.Outputs != nil {
			require.NotContains(t, f.Outputs.Properties, "id", "%s result", tok)
		}
	}
}

// The server's view of a resource is an output next to the inputs, so defaults
// never overwrite what the program wrote.
func TestSchema_EffectiveOutputsAreOutputsOnly(t *testing.T) {
	spec := mustSchema(t)
	for _, tok := range []string{"Application", "IncomingPhoneNumber", "MessagingService", "VerifyService"} {
		r := spec.Resources["twilio:index:"+tok]
		require.Contains(t, r.Properties, "effective", tok)
		require.NotContains(t, r.InputProperties, "effective", tok)
		require.NotContains(t, r.RequiredInputs, "effective", tok)
	}
}

func TestSchema_AccountResources(t *testing.T) {
	spec := mustSchema(t)
	for _, r := range []string{"Address", "SigningKey", "Queue", "UsageTrigger"} {
		require.Contains(t, spec.Resources, "twilio:index:"+r)
	}
	for _, f := range []string{"getAddress", "getSigningKey", "getQueue", "getUsageTrigger"} {
		require.Contains(t, spec.Functions, "twilio:index:"+f)
	}
	require.True(t, spec.Resources["twilio:index:SigningKey"].Properties["secret"].Secret)
	for tok, props := range map[string][]string{
		"twilio:index:Address":      {"isoCountry", "accountSid"},
		"twilio:index:SigningKey":   {"accountSid"},
		"twilio:index:Queue":        {"accountSid"},
		"twilio:index:UsageTrigger": {"triggerValue", "usageCategory", "recurring", "triggerBy", "accountSid"},
	} {
		for _, p := range props {
			require.True(t, spec.Resources[tok].InputProperties[p].ReplaceOnChanges, "%s.%s", tok, p)
		}
	}
	for tok, props := range map[string][]string{
		"twilio:index:Address":      {"street", "friendlyName", "emergencyEnabled"},
		"twilio:index:SigningKey":   {"friendlyName"},
		"twilio:index:Queue":        {"friendlyName", "maxSize"},
		"twilio:index:UsageTrigger": {"callbackUrl", "callbackMethod", "friendlyName"},
	} {
		for _, p := range props {
			require.False(t, spec.Resources[tok].InputProperties[p].ReplaceOnChanges, "%s.%s", tok, p)
		}
	}
}

func TestSchema_SipResources(t *testing.T) {
	spec := mustSchema(t)
	for _, r := range []string{"SipCredentialList", "SipCredential", "SipIpAccessControlList", "SipIpAddress", "SipDomain"} {
		require.Contains(t, spec.Resources, "twilio:index:"+r)
	}
	for _, f := range []string{"getSipCredentialList", "getSipCredential", "getSipIpAccessControlList", "getSipIpAddress", "getSipDomain"} {
		require.Contains(t, spec.Functions, "twilio:index:"+f)
	}
	cred := spec.Resources["twilio:index:SipCredential"]
	require.True(t, cred.InputProperties["password"].Secret, "password input is secret")
	require.True(t, cred.Properties["password"].Secret, "password output is secret")
	require.Contains(t, cred.RequiredInputs, "password")
	for tok, props := range map[string][]string{
		"twilio:index:SipCredentialList":      {"accountSid"},
		"twilio:index:SipCredential":          {"credentialListSid", "username", "accountSid"},
		"twilio:index:SipIpAccessControlList": {"accountSid"},
		"twilio:index:SipIpAddress":           {"ipAccessControlListSid", "accountSid"},
		"twilio:index:SipDomain":              {"accountSid"},
	} {
		for _, p := range props {
			require.True(t, spec.Resources[tok].InputProperties[p].ReplaceOnChanges, "%s.%s", tok, p)
		}
	}
	for tok, props := range map[string][]string{
		"twilio:index:SipCredentialList":      {"friendlyName"},
		"twilio:index:SipCredential":          {"password"},
		"twilio:index:SipIpAccessControlList": {"friendlyName"},
		"twilio:index:SipIpAddress":           {"friendlyName", "ipAddress", "cidrPrefixLength"},
		"twilio:index:SipDomain":              {"domainName", "friendlyName", "voiceUrl", "secure"},
	} {
		for _, p := range props {
			require.False(t, spec.Resources[tok].InputProperties[p].ReplaceOnChanges, "%s.%s", tok, p)
		}
	}
	for _, tok := range []string{"twilio:index:SipCredential", "twilio:index:SipIpAddress"} {
		require.NotContains(t, spec.Resources[tok].RequiredInputs, "accountSid")
	}
}

func TestSchema_EventStreams(t *testing.T) {
	spec := mustSchema(t)
	for _, r := range []string{"EventSink", "EventSubscription"} {
		require.Contains(t, spec.Resources, "twilio:index:"+r)
	}
	for _, f := range []string{"getEventSink", "getEventSubscription"} {
		require.Contains(t, spec.Functions, "twilio:index:"+f)
	}
	sink := spec.Resources["twilio:index:EventSink"]
	for _, p := range []string{"sinkType", "webhookDestination", "webhookMethod", "webhookBatchEvents", "kinesisArn", "kinesisRoleArn", "kinesisExternalId", "segmentWriteKey"} {
		require.True(t, sink.InputProperties[p].ReplaceOnChanges, "EventSink.%s", p)
	}
	require.False(t, sink.InputProperties["description"].ReplaceOnChanges)
	for _, p := range []string{"segmentWriteKey", "kinesisExternalId"} {
		require.True(t, sink.InputProperties[p].Secret, "input %s is secret", p)
		require.True(t, sink.Properties[p].Secret, "output %s is secret", p)
	}
	sub := spec.Resources["twilio:index:EventSubscription"]
	require.True(t, sub.InputProperties["sinkSid"].ReplaceOnChanges)
	for _, p := range []string{"description", "types"} {
		require.False(t, sub.InputProperties[p].ReplaceOnChanges, "EventSubscription.%s", p)
	}
	require.ElementsMatch(t, []string{"description", "sinkSid", "types"}, sub.RequiredInputs)
}
