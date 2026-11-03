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

	"github.com/pulumi/autoregistry/lib/providers/twilio/provider/pkg/provider"
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
		"twilio:index:IncomingPhoneNumber":         {"phoneNumber", "phoneNumberSid", "accountSid"},
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
	require.Equal(t, "https://github.com/pulumi/autoregistry", spec.Repository)
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
