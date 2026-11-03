package provider

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func str1(s string) *string { return &s }

func TestClientFromConfig(t *testing.T) {
	cases := map[string]struct {
		cfg     Config
		wantErr string
		user    string
		pass    string
	}{
		"missing account":     {cfg: Config{AuthToken: str1("t")}, wantErr: "accountSid is required"},
		"missing credential":  {cfg: Config{AccountSid: str1("AC1")}, wantErr: "authToken (TWILIO_AUTH_TOKEN) or apiKey and apiSecret"},
		"auth token":          {cfg: Config{AccountSid: str1("AC1"), AuthToken: str1("tok")}, user: "AC1", pass: "tok"},
		"api key":             {cfg: Config{AccountSid: str1("AC1"), APIKey: str1("SK1"), APISecret: str1("sec")}, user: "SK1", pass: "sec"},
		"api key wins":        {cfg: Config{AccountSid: str1("AC1"), AuthToken: str1("tok"), APIKey: str1("SK1"), APISecret: str1("sec")}, user: "SK1", pass: "sec"},
		"key without secret":  {cfg: Config{AccountSid: str1("AC1"), APIKey: str1("SK1")}, wantErr: "set together"},
		"edge without region": {cfg: Config{AccountSid: str1("AC1"), AuthToken: str1("t"), Edge: str1("sydney")}, wantErr: "edge requires region"},
		"plain http base":     {cfg: Config{AccountSid: str1("AC1"), AuthToken: str1("t"), BaseURL: str1("http://example.com")}, wantErr: "must use https"},
		"loopback http base":  {cfg: Config{AccountSid: str1("AC1"), AuthToken: str1("t"), BaseURL: str1("http://127.0.0.1:1234")}, user: "AC1", pass: "t"},
	}
	for name, tc := range cases {
		t.Run(name, func(t *testing.T) {
			c, err := clientFromConfig(&tc.cfg)
			if tc.wantErr != "" {
				require.ErrorContains(t, err, tc.wantErr)
				return
			}
			require.NoError(t, err)
			require.Equal(t, tc.user, c.user)
			require.Equal(t, tc.pass, c.pass)
		})
	}
}

func TestHostSelection(t *testing.T) {
	require.Equal(t, "https://api.twilio.com", newClient("AC1", "u", "p", "", "", "").host("api"))
	require.Equal(t, "https://verify.ie1.twilio.com", newClient("AC1", "u", "p", "", "ie1", "").host("verify"))
	require.Equal(t, "https://api.sydney.au1.twilio.com", newClient("AC1", "u", "p", "sydney", "au1", "").host("api"))
	require.Equal(t, "http://127.0.0.1:9/iam", newClient("AC1", "u", "p", "", "", "http://127.0.0.1:9/").host("iam"))
}
