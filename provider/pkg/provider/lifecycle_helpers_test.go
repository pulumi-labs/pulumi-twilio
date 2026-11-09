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
	"testing"

	pgp "github.com/pulumi/pulumi-go-provider"
	"github.com/pulumi/pulumi-go-provider/integration"
	"github.com/pulumi/pulumi/sdk/v3/go/common/resource"
	"github.com/pulumi/pulumi/sdk/v3/go/property"
	"github.com/stretchr/testify/require"
)

// These helpers drive resources through the real provider RPC surface
// (integration.Server) against the in-memory fake in fake_twilio_test.go.

func pv(kv map[string]property.Value) property.Map { return property.NewMap(kv) }

func sv(s string) property.Value { return property.New(s) }

func bv(b bool) property.Value { return property.New(b) }

func nv(n int) property.Value { return property.New(float64(n)) }

func computed() property.Value { return property.New(property.Computed) }

func empty() property.Map { return property.NewMap(nil) }

func with(m property.Map, kv map[string]property.Value) property.Map {
	for k, v := range kv {
		m = m.Set(k, v)
	}
	return m
}

func without(m property.Map, keys ...string) property.Map {
	for _, k := range keys {
		m = m.Delete(k)
	}
	return m
}

func hasProp(m property.Map, key string) bool {
	_, ok := m.GetOk(key)
	return ok
}

func getS(t *testing.T, m property.Map, key string) string {
	t.Helper()
	v, ok := m.GetOk(key)
	require.True(t, ok, "property %q missing from %v", key, m)
	return v.AsString()
}

func requireNoChanges(t *testing.T, d pgp.DiffResponse) {
	t.Helper()
	require.False(t, d.HasChanges, "unexpected diff: %+v", d.DetailedDiff)
}

// effective returns the nested effective output as a map.
func effective(t *testing.T, state property.Map) property.Map {
	t.Helper()
	v, ok := state.GetOk("effective")
	require.True(t, ok, "effective output missing from %v", state)
	return v.AsMap()
}

// tl bundles the pieces every lifecycle step needs.
type tl struct {
	t   *testing.T
	s   integration.Server
	f   *fakeTwilio
	urn resource.URN
}

func newTL(t *testing.T, typ, name string) *tl {
	t.Helper()
	f := startFakeTwilio(t)
	return &tl{t: t, s: newServerFor(t, f.server.URL), f: f, urn: twilioURN(typ, name)}
}

// other returns a view of the same provider server and fake for another resource.
func (l *tl) other(typ, name string) *tl {
	return &tl{t: l.t, s: l.s, f: l.f, urn: twilioURN(typ, name)}
}

func (l *tl) check(inputs property.Map) pgp.CheckResponse {
	l.t.Helper()
	chk, err := l.s.Check(pgp.CheckRequest{Urn: l.urn, Inputs: inputs})
	require.NoError(l.t, err)
	return chk
}

// checked runs Check and returns the validated inputs (with defaults applied).
func (l *tl) checked(inputs property.Map) property.Map {
	l.t.Helper()
	chk := l.check(inputs)
	require.Empty(l.t, chk.Failures)
	return chk.Inputs
}

func (l *tl) create(inputs property.Map) pgp.CreateResponse {
	l.t.Helper()
	c, err := l.s.Create(pgp.CreateRequest{Urn: l.urn, Properties: l.checked(inputs)})
	require.NoError(l.t, err)
	require.NotEmpty(l.t, c.ID)
	return c
}

func (l *tl) diff(id string, state, inputs, oldInputs property.Map) pgp.DiffResponse {
	l.t.Helper()
	d, err := l.s.Diff(pgp.DiffRequest{ID: id, Urn: l.urn, State: state, Inputs: inputs, OldInputs: oldInputs})
	require.NoError(l.t, err)
	return d
}

func (l *tl) update(id string, state, inputs, oldInputs property.Map) pgp.UpdateResponse {
	l.t.Helper()
	u, err := l.s.Update(pgp.UpdateRequest{ID: id, Urn: l.urn, State: state, Inputs: l.checked(inputs), OldInputs: oldInputs})
	require.NoError(l.t, err)
	return u
}

func (l *tl) read(id string, state, inputs property.Map) pgp.ReadResponse {
	l.t.Helper()
	r, err := l.s.Read(pgp.ReadRequest{ID: id, Urn: l.urn, Properties: state, Inputs: inputs})
	require.NoError(l.t, err)
	return r
}

// requireImportFails asserts that an import (Read with neither inputs nor state) of an ID
// that does not exist is an error rather than an empty resource (#1715).
func (l *tl) requireImportFails(id string) {
	l.t.Helper()
	_, err := l.s.Read(pgp.ReadRequest{ID: id, Urn: l.urn, Properties: empty(), Inputs: empty()})
	require.ErrorContains(l.t, err, "does not exist")
}

func (l *tl) del(id string, state property.Map) {
	l.t.Helper()
	require.NoError(l.t, l.s.Delete(pgp.DeleteRequest{ID: id, Urn: l.urn, Properties: state}))
}
