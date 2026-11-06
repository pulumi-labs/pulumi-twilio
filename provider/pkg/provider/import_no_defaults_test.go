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

	"github.com/pulumi/pulumi/sdk/v3/go/property"
	"github.com/stretchr/testify/require"
)

// #1572: an import adopted every returned bool and int that had no verified default, so Twilio's
// own defaults (stickySender, codeLength, psd2Enabled, ...) became inputs and a program that set
// only the fields it cares about planned to remove them on the first preview. Each case creates
// a resource from a minimal program, lets the server report its defaults, imports by ID alone and
// diffs the program against the imported state.
func TestImport_ServerDefaultsAreNotAdoptedAsInputs(t *testing.T) {
	cases := []struct {
		name, typ, kind string
		program         property.Map
		serverDefaults  map[string]any
		effective       map[string]any // camelCase effective keys and the values the server reports
		setup           func(l *tl)
	}{
		{
			name: "Application", typ: "Application", kind: "app",
			program:        pv(map[string]property.Value{"friendlyName": sv("arqa")}),
			serverDefaults: map[string]any{"voice_caller_id_lookup": false},
			effective:      map[string]any{"voiceCallerIdLookup": false},
		},
		{
			name: "MessagingService", typ: "MessagingService", kind: "msg",
			program: pv(map[string]property.Value{"friendlyName": sv("arqa")}),
			serverDefaults: map[string]any{
				"area_code_geomatch": true, "fallback_to_long_code": true, "mms_converter": true, "smart_encoding": true,
				"sticky_sender": true, "synchronous_validation": false, "use_inbound_webhook_on_number": false,
			},
			effective: map[string]any{"stickySender": true, "areaCodeGeomatch": true, "synchronousValidation": false},
		},
		{
			name: "VerifyService", typ: "VerifyService", kind: "verify",
			program: pv(map[string]property.Value{"friendlyName": sv("arqa")}),
			serverDefaults: map[string]any{
				"code_length": float64(6), "custom_code_enabled": false, "dtmf_input_required": true,
				"psd2_enabled": false, "skip_sms_to_landlines": false,
			},
			effective: map[string]any{"codeLength": float64(6), "psd2Enabled": false, "dtmfInputRequired": true},
		},
		{
			name: "IncomingPhoneNumber", typ: "IncomingPhoneNumber", kind: "number",
			program: pv(map[string]property.Value{"phoneNumberSid": sv(numSid), "friendlyName": sv("support"),
				// The fake gives a number Twilio's demo voice URL, which an import adopts like the real API's.
				"voiceUrl": sv("https://demo.twilio.com/welcome/voice/")}),
			serverDefaults: map[string]any{"voice_caller_id_lookup": false},
			effective:      map[string]any{"voiceCallerIdLookup": false},
			setup:          func(l *tl) { l.f.addNumber(testAccount, numSid, numE) },
		},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			l := newTL(t, c.typ, "r")
			if c.setup != nil {
				c.setup(l)
			}
			created := l.create(c.program)
			l.f.mutate(c.kind, testAccount, created.ID, func(o map[string]any) {
				for k, v := range c.serverDefaults {
					o[k] = v
				}
			})
			imp := l.read(created.ID, empty(), empty())
			d := l.diff(created.ID, imp.Properties, l.checked(c.program), imp.Inputs)
			require.False(t, d.HasChanges, "importing %s must not plan a change against the program that created it: %+v (imported inputs %v)", c.typ, d.DetailedDiff, imp.Inputs)
			// The effective output still shows what the server holds, key by key.
			eff := effective(t, imp.Properties)
			for k, want := range c.effective {
				got, ok := eff.GetOk(k)
				require.True(t, ok, "effective must report %s: %v", k, eff)
				switch w := want.(type) {
				case bool:
					require.Equal(t, w, got.AsBool(), k)
				case float64:
					require.Equal(t, w, got.AsNumber(), k)
				}
			}
		})
	}
}

// #1595: a flag or number that differs from the import-only default is a genuine setting and an
// import adopts it, so the program that declares it plans nothing; one that equals the default
// stays out of the inputs. A field with no known default (lookupEnabled) is still not adopted.
func TestImport_NonDefaultSettingsAreAdopted(t *testing.T) {
	cases := []struct {
		name, typ, kind string
		program         property.Map
		live            map[string]any // what a freshly created resource reports on the live API
		server          map[string]any // out-of-band settings, non-default and default mixed
		adopted         map[string]property.Value
		notAdopted      []string
		setup           func(l *tl)
	}{
		{
			name: "Application", typ: "Application", kind: "app",
			program: pv(map[string]property.Value{"friendlyName": sv("arqa")}),
			live:    map[string]any{"voice_caller_id_lookup": false},
			server:  map[string]any{"voice_caller_id_lookup": true},
			adopted: map[string]property.Value{"voiceCallerIdLookup": bv(true)},
		},
		{
			name: "MessagingService", typ: "MessagingService", kind: "msg",
			program: pv(map[string]property.Value{"friendlyName": sv("arqa")}),
			live: map[string]any{"area_code_geomatch": true, "mms_converter": true, "smart_encoding": true,
				"sticky_sender": true, "synchronous_validation": false, "use_inbound_webhook_on_number": false},
			server:     map[string]any{"sticky_sender": false, "mms_converter": true, "smart_encoding": false, "area_code_geomatch": true},
			adopted:    map[string]property.Value{"stickySender": bv(false), "smartEncoding": bv(false)},
			notAdopted: []string{"mmsConverter", "areaCodeGeomatch"},
		},
		{
			name: "VerifyService", typ: "VerifyService", kind: "verify",
			program: pv(map[string]property.Value{"friendlyName": sv("arqa")}),
			live: map[string]any{"code_length": float64(6), "custom_code_enabled": false, "dtmf_input_required": true,
				"psd2_enabled": false, "skip_sms_to_landlines": false},
			server:     map[string]any{"code_length": float64(4), "psd2_enabled": true, "dtmf_input_required": true, "lookup_enabled": true},
			adopted:    map[string]property.Value{"codeLength": nv(4), "psd2Enabled": bv(true)},
			notAdopted: []string{"dtmfInputRequired", "lookupEnabled"},
		},
		{
			name: "IncomingPhoneNumber", typ: "IncomingPhoneNumber", kind: "number",
			program: pv(map[string]property.Value{"phoneNumberSid": sv(numSid), "friendlyName": sv("support"),
				"voiceUrl": sv("https://demo.twilio.com/welcome/voice/")}),
			live:    map[string]any{"voice_caller_id_lookup": false},
			server:  map[string]any{"voice_caller_id_lookup": true},
			adopted: map[string]property.Value{"voiceCallerIdLookup": bv(true)},
			setup:   func(l *tl) { l.f.addNumber(testAccount, numSid, numE) },
		},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			l := newTL(t, c.typ, "r")
			if c.setup != nil {
				c.setup(l)
			}
			created := l.create(c.program)
			l.f.mutate(c.kind, testAccount, created.ID, func(o map[string]any) {
				for k, v := range c.live {
					o[k] = v
				}
			})
			// Create from the minimal program, import: nothing planned (#1572 still holds).
			base := l.read(created.ID, empty(), empty())
			requireNoDiff := l.diff(created.ID, base.Properties, l.checked(c.program), base.Inputs)
			require.False(t, requireNoDiff.HasChanges, "%+v", requireNoDiff.DetailedDiff)
			// Set non-default values out of band, import: they appear in the inputs.
			l.f.mutate(c.kind, testAccount, created.ID, func(o map[string]any) {
				for k, v := range c.server {
					o[k] = v
				}
			})
			imp := l.read(created.ID, empty(), empty())
			for k, want := range c.adopted {
				got, ok := imp.Inputs.GetOk(k)
				require.True(t, ok, "%s must be adopted: %v", k, imp.Inputs)
				require.True(t, want.Equals(got), k)
			}
			for _, k := range c.notAdopted {
				require.False(t, hasProp(imp.Inputs, k), "%s equals the default or has none and must stay out", k)
			}
			// The program that declares the adopted values plans nothing against the import.
			declared := with(c.program, c.adopted)
			require.False(t, l.diff(created.ID, imp.Properties, l.checked(declared), imp.Inputs).HasChanges)
		})
	}
}
