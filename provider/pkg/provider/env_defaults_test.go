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
	"testing"

	"github.com/stretchr/testify/require"
)

func envTestStr(s string) *string { return &s }

func envTestDeref(s *string) string {
	if s == nil {
		return ""
	}
	return *s
}

// Environment-only settings must reach Configure even when the engine skips
// CheckConfig, as it does for the provider of an imported resource (#1951).
func TestConfigure_EnvDefaultsWhenCheckConfigSkipped(t *testing.T) {
	t.Setenv("TWILIO_ACCOUNT_SID", "")
	t.Setenv("TWILIO_AUTH_TOKEN", "")
	t.Setenv("TWILIO_API_KEY", "")
	t.Setenv("TWILIO_API_SECRET", "")
	t.Setenv("TWILIO_REGION", "")
	t.Setenv("TWILIO_EDGE", "")
	t.Setenv("TWILIO_BASE_URL", "")
	t.Setenv("TWILIO_ACCOUNT_SID", "env-AccountSid")
	t.Setenv("TWILIO_AUTH_TOKEN", "env-AuthToken")
	t.Setenv("TWILIO_API_KEY", "env-APIKey")
	t.Setenv("TWILIO_API_SECRET", "env-APISecret")
	t.Setenv("TWILIO_REGION", "env-Region")
	t.Setenv("TWILIO_EDGE", "env-Edge")
	t.Setenv("TWILIO_BASE_URL", "env-BaseURL")
	c := Config{}
	require.NoError(t, c.Configure(context.Background()))
	require.Equal(t, "env-AccountSid", envTestDeref(c.AccountSid))
	require.Equal(t, "env-AuthToken", envTestDeref(c.AuthToken))
	require.Equal(t, "env-APIKey", envTestDeref(c.APIKey))
	require.Equal(t, "env-APISecret", envTestDeref(c.APISecret))
	require.Equal(t, "env-Region", envTestDeref(c.Region))
	require.Equal(t, "env-Edge", envTestDeref(c.Edge))
	require.Equal(t, "env-BaseURL", envTestDeref(c.BaseURL))
}

// A value from stack config wins over the environment, and an unset
// environment leaves the setting unset.
func TestConfigure_EnvDefaultsDoNotOverrideConfig(t *testing.T) {
	t.Setenv("TWILIO_ACCOUNT_SID", "env-AccountSid")
	t.Setenv("TWILIO_AUTH_TOKEN", "env-AuthToken")
	t.Setenv("TWILIO_API_KEY", "env-APIKey")
	t.Setenv("TWILIO_API_SECRET", "env-APISecret")
	t.Setenv("TWILIO_REGION", "env-Region")
	t.Setenv("TWILIO_EDGE", "env-Edge")
	t.Setenv("TWILIO_BASE_URL", "env-BaseURL")
	c := Config{AccountSid: envTestStr("from-config")}
	require.NoError(t, c.Configure(context.Background()))
	require.Equal(t, "from-config", envTestDeref(c.AccountSid))
	t.Setenv("TWILIO_ACCOUNT_SID", "")
	t.Setenv("TWILIO_AUTH_TOKEN", "")
	t.Setenv("TWILIO_API_KEY", "")
	t.Setenv("TWILIO_API_SECRET", "")
	t.Setenv("TWILIO_REGION", "")
	t.Setenv("TWILIO_EDGE", "")
	t.Setenv("TWILIO_BASE_URL", "")
	c = Config{}
	require.NoError(t, c.Configure(context.Background()))
	require.Nil(t, c.AccountSid)
}
