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

	"github.com/stretchr/testify/require"
)

// Twilio accepts batch_events=true on create and then reports false. A refresh must keep the recorded
// input, otherwise the next preview plans a replacement of the sink on every run (pulumi-labs/pulumi-twilio#11).
func TestEventSinkLC_BatchEventsReportedFalseDoesNotReplace(t *testing.T) {
	l := newTL(t, "EventSink", "sink")
	in := webhookSinkInputs()
	c := l.create(in)
	l.f.mutate("evsink", testAccount, c.ID, func(o map[string]any) {
		o["sink_configuration"].(map[string]any)["batch_events"] = false
	})

	rd := l.read(c.ID, c.Properties, in)
	require.True(t, rd.Inputs.Get("webhookBatchEvents").AsBool(), "refresh must keep the recorded batch flag")
	require.True(t, rd.Properties.Get("webhookBatchEvents").AsBool())
	requireNoChanges(t, l.diff(c.ID, rd.Properties, in, rd.Inputs))
	// Repeating the refresh stays stable.
	rd = l.read(c.ID, rd.Properties, rd.Inputs)
	requireNoChanges(t, l.diff(c.ID, rd.Properties, in, rd.Inputs))
}

// An import of a sink Twilio reports as unbatched adopts no flag, so a program that omits it is not a change.
func TestEventSinkLC_ImportOfUnbatchedSinkAdoptsNoBatchFlag(t *testing.T) {
	l := newTL(t, "EventSink", "sink")
	in := webhookSinkInputs()
	c := l.create(in)
	l.f.mutate("evsink", testAccount, c.ID, func(o map[string]any) {
		o["sink_configuration"].(map[string]any)["batch_events"] = false
	})
	imp := l.read(c.ID, empty(), empty())
	require.False(t, hasProp(imp.Inputs, "webhookBatchEvents"))
	requireNoChanges(t, l.diff(c.ID, imp.Properties, without(in, "webhookBatchEvents"), imp.Inputs))
}
