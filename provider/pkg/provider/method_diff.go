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
	"strings"

	p "github.com/pulumi/pulumi-go-provider"
	"github.com/pulumi/pulumi/sdk/v3/go/property"
)

// withDefaultMethodDiff treats an unset HTTP method input and an explicit "POST" as the same
// value on every resource (#1743).
//
// Twilio's default for every *Method field is POST. An import deliberately does not adopt a
// returned POST as an input, so a program that omits the method plans nothing (#1572). The
// other half is a program that states the default itself (voiceMethod: "POST"): infer compares
// the imported state (no method) with the program's (POST) and plans an add, even though
// nothing would change on Twilio. This wrapper keeps infer's diff and drops each top-level
// *Method entry whose old and new input are the same once an absent value counts as POST,
// comparing case-insensitively the way Twilio does. A method that really changes (GET to
// POST, GET to unset, unset to GET) still plans an update, and an unknown or non-string value
// is left to infer. Nested method fields are not touched.
func withDefaultMethodDiff(prov p.Provider) p.Provider {
	inner := prov.Diff
	if inner == nil {
		return prov
	}
	prov.Diff = func(ctx context.Context, req p.DiffRequest) (p.DiffResponse, error) {
		resp, err := inner(ctx, req)
		if err != nil || len(resp.DetailedDiff) == 0 {
			return resp, err
		}
		for k, d := range resp.DetailedDiff {
			if d.Kind != p.Add && d.Kind != p.Delete && d.Kind != p.Update {
				continue
			}
			if !strings.HasSuffix(k, "Method") || strings.ContainsAny(k, ".[") {
				continue
			}
			// Infer diffs the stored state against the new inputs; an import has no method in
			// either, so fall back to the old inputs and then to the default.
			old := req.State
			if _, ok := old.GetOk(k); !ok {
				old = req.OldInputs
			}
			oldM, okOld := methodValue(old, k)
			newM, okNew := methodValue(req.Inputs, k)
			if okOld && okNew && strings.EqualFold(oldM, newM) {
				delete(resp.DetailedDiff, k)
			}
		}
		if len(resp.DetailedDiff) == 0 {
			resp.HasChanges = false
		}
		return resp, nil
	}
	return prov
}

// methodValue returns the method in m, "POST" when it is unset, and false when the value is
// not a plain string (computed, or of another type), which this wrapper leaves alone.
func methodValue(m property.Map, key string) (string, bool) {
	v, ok := m.GetOk(key)
	if !ok || v.IsNull() {
		return strDefault(key), true
	}
	if !v.IsString() {
		return "", false
	}
	return v.AsString(), true
}
