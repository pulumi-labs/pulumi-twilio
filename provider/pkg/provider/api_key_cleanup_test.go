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
	"os"
	"path/filepath"
	"strings"
	"testing"

	pgp "github.com/pulumi/pulumi-go-provider"
	"github.com/pulumi/pulumi/sdk/v3/go/property"
	"github.com/stretchr/testify/require"
)

// #1574: Twilio creates a restricted key and then fails the call with error 70002 when a policy
// assertion is not of the form /twilio/<product>/<resource>/<action>. The error carries no SID, so
// the provider recorded nothing and left a live credential behind. The fake now behaves the same way.

const (
	badPolicy  = `{"allow":["/v1/Services/**"]}`
	goodPolicy = `{"allow":["/twilio/iam/api-keys/create"]}`
)

func restrictedKey(name, policy string) property.Map {
	return pv(map[string]property.Value{"friendlyName": sv(name), "keyType": sv("restricted"), "policy": sv(policy)})
}

// keysNamed returns the SIDs of the keys of an account that carry a friendly name.
func keysNamed(l *tl, acct, name string) []string {
	l.f.mu.Lock()
	defer l.f.mu.Unlock()
	var sids []string
	for k, o := range l.f.objs {
		if strings.HasPrefix(k, "key/"+acct+"/") && o["friendly_name"] == name {
			sids = append(sids, o["sid"].(string))
		}
	}
	return sids
}

func createErr(l *tl, in property.Map) error {
	_, err := l.s.Create(pgp.CreateRequest{Urn: l.urn, Properties: l.checked(in)})
	return err
}

func TestApiKey_FailedCreateLeavesNoUntrackedKey(t *testing.T) {
	l := newTL(t, "ApiKey", "k")
	err := createErr(l, restrictedKey("arqa", badPolicy))
	require.Error(t, err)
	require.Contains(t, err.Error(), "70002", "the original Twilio error is kept")
	require.Contains(t, err.Error(), "was deleted", "the error says what became of the key Twilio created")
	require.Empty(t, keysNamed(l, testAccount, "arqa"), "no key Twilio created during the failed call may be left behind")

	// The documented policy works and is kept.
	c := l.create(restrictedKey("arqa", goodPolicy))
	require.Len(t, keysNamed(l, testAccount, "arqa"), 1)
	require.Equal(t, goodPolicy, getS(t, c.Properties, "policy"))
}

// A key that already carried the friendly name belongs to someone else and must survive.
func TestApiKey_FailedCreateKeepsAPreexistingKeyWithTheSameName(t *testing.T) {
	l := newTL(t, "ApiKey", "k")
	l.f.addKey(testAccount, "SKpre", "arqa", nil)
	err := createErr(l, restrictedKey("arqa", badPolicy))
	require.Error(t, err)
	require.Equal(t, []string{"SKpre"}, keysNamed(l, testAccount, "arqa"), "only the key created by the failed call is removed")
}

func TestApiKey_FailedCreateInASubaccountCleansUpThere(t *testing.T) {
	l := newTL(t, "ApiKey", "k")
	l.f.addSubaccount(fakeSub)
	in := with(restrictedKey("arqa", badPolicy), map[string]property.Value{"accountSid": sv(fakeSub)})
	require.Error(t, createErr(l, in))
	require.Empty(t, keysNamed(l, fakeSub, "arqa"))
}

// When the orphan cannot be deleted the error names it, because it is a live credential.
func TestApiKey_FailedCreateNamesAnOrphanItCannotDelete(t *testing.T) {
	l := newTL(t, "ApiKey", "k")
	l.f.failNext("DELETE", "/iam/v1/Keys/", 500)
	err := createErr(l, restrictedKey("arqa", badPolicy))
	require.Error(t, err)
	sids := keysNamed(l, testAccount, "arqa")
	require.Len(t, sids, 1)
	require.Contains(t, err.Error(), sids[0])
	require.Contains(t, err.Error(), "could not be deleted")
}

// When the keys cannot be listed the provider cannot tell which key is its own, so it deletes none.
func TestApiKey_FailedCreateWithoutAListDeletesNothingAndSaysSo(t *testing.T) {
	l := newTL(t, "ApiKey", "k")
	l.f.failNext("GET", "/Keys.json", 400)
	err := createErr(l, restrictedKey("arqa", badPolicy))
	require.Error(t, err)
	require.Contains(t, err.Error(), "may have created a key named \"arqa\"")
	require.Len(t, keysNamed(l, testAccount, "arqa"), 1, "nothing is deleted on a guess")
}

// A create that fails before Twilio made anything finds nothing to clean up and reports the bare error.
func TestApiKey_FailedCreateThatMadeNothingIsNotDressedUp(t *testing.T) {
	l := newTL(t, "ApiKey", "k")
	in := with(restrictedKey("arqa", goodPolicy), map[string]property.Value{"accountSid": sv("ACdoesnotexist")})
	err := createErr(l, in)
	require.Error(t, err)
	require.NotContains(t, err.Error(), "deleted")
	require.NotContains(t, err.Error(), "may have created")
}

// The documented example in every place a user reads it is an assertion Twilio accepts: none of the
// examples or docs carry a REST path.
func TestApiKey_DocumentedPoliciesAreAssertionsTwilioAccepts(t *testing.T) {
	require.Empty(t, unsupportedAssertions(goodPolicy))
	require.Equal(t, []string{"/v1/Services/**"}, unsupportedAssertions(badPolicy))

	dirs, err := filepath.Glob("../../../examples/twilio-subaccount-*")
	require.NoError(t, err)
	require.NotEmpty(t, dirs)
	// The agent guide is skills/SKILLS.md in the monorepo and CLAUDE.md at the root of the exported mirror.
	var files []string
	for _, g := range []string{"../../../skills/SKILLS.md", "../../../CLAUDE.md"} {
		if _, err := os.Stat(g); err == nil {
			files = append(files, g)
			break
		}
	}
	require.NotEmpty(t, files, "agent guide (skills/SKILLS.md or CLAUDE.md) not found")
	for _, d := range dirs {
		require.NoError(t, filepath.Walk(d, func(p string, info os.FileInfo, err error) error {
			if err == nil && !info.IsDir() && !strings.Contains(p, "node_modules") {
				files = append(files, p)
			}
			return err
		}))
	}
	for _, f := range files {
		b, err := os.ReadFile(f)
		require.NoError(t, err)
		require.NotRegexp(t, `/v[0-9]+/Services|"/Services/`, string(b), f)
		if strings.Contains(f, "twilio-subaccount-") && (strings.HasSuffix(f, "main.go") || strings.HasSuffix(f, "index.ts") ||
			strings.HasSuffix(f, "__main__.py") || strings.HasSuffix(f, "Program.cs") || strings.HasSuffix(f, "App.java") || strings.HasSuffix(f, "Pulumi.yaml")) {
			if strings.Contains(string(b), "allow") {
				require.Contains(t, string(b), "/twilio/iam/api-keys/create", f)
			}
		}
	}
}

// An account with more key pages than the walk reads: the listing refuses rather than let a
// pre-existing key look new, so a failed create falls into the "cannot list, delete nothing" branch.
func TestApiKey_FailedCreateOnAnEndlessKeyListDeletesNothing(t *testing.T) {
	l := newTL(t, "ApiKey", "k")
	l.f.endlessKeyPages = true
	err := createErr(l, restrictedKey("arqa", badPolicy))
	require.Error(t, err)
	require.Contains(t, err.Error(), "more than")
	require.Contains(t, err.Error(), "may have created a key named \"arqa\"")
	require.Len(t, keysNamed(l, testAccount, "arqa"), 1, "nothing is deleted on a guess")
}
