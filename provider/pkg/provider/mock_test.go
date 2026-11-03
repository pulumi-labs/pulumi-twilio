package provider

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/blang/semver"
	pgp "github.com/pulumi/pulumi-go-provider"
	"github.com/pulumi/pulumi-go-provider/integration"
	"github.com/pulumi/pulumi/sdk/v3/go/common/resource"
	"github.com/pulumi/pulumi/sdk/v3/go/common/tokens"
	"github.com/pulumi/pulumi/sdk/v3/go/property"
	"github.com/stretchr/testify/require"
)

const (
	testAccount = "ACmain00000000000000000000000000"
	testToken   = "token-main"
)

// mockTwilio is a stateful in-memory fake of the Twilio endpoints the provider
// uses. Resources are stored under their normalized request path.
type mockTwilio struct {
	mu       sync.Mutex
	objs     map[string]map[string]any
	calls    []string
	forms    map[string]url.Values // last form sent per "METHOD path"
	next     int
	throttle int // number of upcoming requests to answer with 429
	authSeen bool
}

func newMock(t *testing.T) (*mockTwilio, *httptest.Server) {
	m := &mockTwilio{objs: map[string]map[string]any{}, forms: map[string]url.Values{}}
	srv := httptest.NewServer(m)
	t.Cleanup(srv.Close)
	return m, srv
}

func (m *mockTwilio) callLog() []string {
	m.mu.Lock()
	defer m.mu.Unlock()
	return append([]string(nil), m.calls...)
}

func (m *mockTwilio) get(path string) map[string]any {
	m.mu.Lock()
	defer m.mu.Unlock()
	return m.objs[path]
}

func (m *mockTwilio) seed(path string, o map[string]any) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.objs[path] = o
}

func writeJSON(w http.ResponseWriter, code int, v any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(code)
	_ = json.NewEncoder(w).Encode(v)
}

func snake(s string) string {
	var b strings.Builder
	for i, r := range s {
		if r >= 'A' && r <= 'Z' {
			if i > 0 {
				b.WriteByte('_')
			}
			r += 'a' - 'A'
		}
		b.WriteRune(r)
	}
	return b.String()
}

var collections = map[string]string{
	"Accounts": "AC", "Applications": "AP", "Keys": "SK", "Services": "SV", "IncomingPhoneNumbers": "PN", "PhoneNumbers": "PN",
}

func (m *mockTwilio) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	m.mu.Lock()
	defer m.mu.Unlock()
	u, pw, ok := r.BasicAuth()
	if !ok || (u != testAccount && u != "SKkey") || (pw != testToken && pw != "key-secret") {
		writeJSON(w, 401, map[string]any{"code": 20003, "message": "Authenticate", "status": 401})
		return
	}
	m.authSeen = true
	if m.throttle > 0 {
		m.throttle--
		writeJSON(w, 429, map[string]any{"code": 20429, "message": "Too Many Requests", "status": 429})
		return
	}
	body, _ := io.ReadAll(r.Body)
	form, _ := url.ParseQuery(string(body))
	norm := strings.TrimSuffix(r.URL.Path, ".json")
	m.calls = append(m.calls, r.Method+" "+r.URL.Path)
	m.forms[r.Method+" "+r.URL.Path] = form
	parts := strings.Split(strings.Trim(norm, "/"), "/")
	last := parts[len(parts)-1]
	prefix, isColl := collections[last]
	switch {
	case isColl && r.Method == "POST":
		m.next++
		sid := fmt.Sprintf("%s%d", prefix, m.next)
		o := map[string]any{"sid": sid, "date_created": "Mon, 02 Nov 2026 00:00:00 +0000"}
		for k, v := range form {
			o[snake(k)] = v[0]
		}
		switch last {
		case "Accounts":
			o["status"] = "active"
			o["auth_token"] = fmt.Sprintf("tok-%d", m.next)
			o["owner_account_sid"] = testAccount
			o["type"] = "Full"
		case "Keys":
			o["secret"] = fmt.Sprintf("secret-%d", m.next)
		case "PhoneNumbers":
			sid = form.Get("PhoneNumberSid")
			o["sid"] = sid
			o["phone_number"] = "+15550001111"
			o["capabilities"] = []any{"SMS", "MMS"}
		}
		m.objs[strings.TrimSuffix(norm, "/")+"/"+o["sid"].(string)] = o
		writeJSON(w, 201, o)
	case isColl && r.Method == "GET":
		want := r.URL.Query().Get("PhoneNumber")
		list := []any{}
		for k, v := range m.objs {
			if strings.HasPrefix(k, norm+"/") && (want == "" || v["phone_number"] == want) {
				list = append(list, v)
			}
		}
		writeJSON(w, 200, map[string]any{"incoming_phone_numbers": list})
	default:
		o, ok := m.objs[norm]
		if !ok {
			writeJSON(w, 404, map[string]any{"code": 20404, "message": "The requested resource " + r.URL.Path + " was not found", "status": 404})
			return
		}
		switch r.Method {
		case "GET":
			c := map[string]any{}
			for k, v := range o {
				if k != "secret" {
					c[k] = v
				}
			}
			writeJSON(w, 200, c)
		case "POST":
			for k, v := range form {
				o[snake(k)] = v[0]
			}
			writeJSON(w, 200, o)
		case "DELETE":
			delete(m.objs, norm)
			w.WriteHeader(204)
		}
	}
}

func init() { retryBaseDelay = time.Millisecond }

func twilioURN(typ, name string) resource.URN {
	return resource.NewURN("stack", "proj", "", tokens.Type("twilio:index:"+typ), name)
}

func newServerFor(t *testing.T, baseURL string) integration.Server {
	t.Helper()
	prov, err := New()
	require.NoError(t, err)
	s, err := integration.NewServer(t.Context(), "twilio", semver.MustParse("0.0.1"), integration.WithProvider(prov))
	require.NoError(t, err)
	require.NoError(t, s.Configure(pgp.ConfigureRequest{Args: property.NewMap(map[string]property.Value{
		"accountSid": property.New(testAccount),
		"authToken":  property.New(testToken),
		"baseUrl":    property.New(baseURL),
	})}))
	return s
}

func pm(kv map[string]any) property.Map {
	out := map[string]property.Value{}
	for k, v := range kv {
		out[k] = toValue(v)
	}
	return property.NewMap(out)
}

func toValue(v any) property.Value {
	switch x := v.(type) {
	case map[string]any:
		return property.New(pm(x))
	case int:
		return property.New(float64(x))
	case string:
		return property.New(x)
	case bool:
		return property.New(x)
	case float64:
		return property.New(x)
	default:
		panic(fmt.Sprintf("unsupported %T", v))
	}
}

func str(m property.Map, k string) string {
	v, ok := m.GetOk(k)
	if !ok || !v.IsString() {
		return ""
	}
	return v.AsString()
}
