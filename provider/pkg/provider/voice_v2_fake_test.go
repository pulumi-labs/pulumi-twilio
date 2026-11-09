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
	"encoding/json"
	"net/http"
	"slices"
	"strings"
)

// The Voice v2 configurations, served for the recording and transcription tests:
//
//	/voice/v2/Configurations/Recording[/{idOrUniqueName}]
//	/voice/v2/Configurations/Transcription[/{idOrUniqueName}]
//
// Bodies are JSON. The shapes follow twilio-oai spec/json/twilio_voice_v2.json (spec version 0.0.1):
// POST answers 201 with the configuration and a server-assigned id, GET fetches by id or unique
// name, PUT replaces the whole configuration (an omitted unique_name or description is cleared)
// and answers 200, DELETE answers 204. The fake applies the one default the tests rely on: a
// status callback with a url and no method reports POST. Transcription needs an engine, a
// language, participant defaults, and a status callback or a conversation configuration id, as
// the schema descriptions say.

type voiceV2Kind struct {
	kind, prefix, typ string
	validate          func(cfg map[string]any) string
}

var voiceV2Kinds = map[string]voiceV2Kind{
	"Recording":     {kind: "voicerec", prefix: "VJ", typ: "Recording", validate: validateRecordingCfg},
	"Transcription": {kind: "voicetx", prefix: "VX", typ: "Transcription", validate: validateTranscriptionCfg},
}

func validateRecordingCfg(cfg map[string]any) string {
	if cp, ok := cfg["compositionPolicy"].(map[string]any); ok {
		for k, allowed := range map[string][]string{"channels": recordingChannels, "trim": recordingTrims, "track": recordingTracks} {
			if v, ok := cp[k].(string); ok && !slices.Contains(allowed, v) {
				return "invalid compositionPolicy." + k + " " + v
			}
		}
	}
	for _, k := range []string{"callRecordingStatusCallback", "conferenceRecordingStatusCallback"} {
		if cb, ok := cfg[k].(map[string]any); ok {
			if v, ok := cb["method"].(string); ok && !slices.Contains(recordingMethods, v) {
				return "invalid " + k + ".method " + v
			}
			if _, has := cb["url"]; has && cb["method"] == nil {
				cb["method"] = "POST"
			}
		}
	}
	if fs, ok := cfg["features"].([]any); ok && len(fs) > 3 {
		return "features takes at most 3 entries"
	}
	return ""
}

func validateTranscriptionCfg(cfg map[string]any) string {
	for _, k := range []string{"transcriptionEngine", "language"} {
		if s, _ := cfg[k].(string); s == "" {
			return k + " is required"
		}
	}
	pd, ok := cfg["participantDefaults"].([]any)
	if !ok || len(pd) != 2 {
		return "participantDefaults must contain exactly 2 participants"
	}
	cb, hasCB := cfg["transcriptionStatusCallback"].(map[string]any)
	if conv, _ := cfg["conversationConfigurationId"].(string); !hasCB && conv == "" {
		return "either transcriptionStatusCallback or conversationConfigurationId must be provided"
	}
	if hasCB {
		if _, has := cb["url"]; has && cb["method"] == nil {
			cb["method"] = "POST"
		}
	}
	return ""
}

func (f *fakeTwilio) handleVoiceV2(w http.ResponseWriter, r *http.Request, body []byte, parts []string) {
	if len(parts) < 2 || len(parts) > 3 || parts[0] != "Configurations" {
		ferr(w, 404, "no route "+r.URL.Path)
		return
	}
	k, ok := voiceV2Kinds[parts[1]]
	if !ok {
		ferr(w, 404, "no route "+r.URL.Path)
		return
	}
	read := func() (map[string]any, string) {
		var req struct {
			UniqueName    *string        `json:"unique_name"`
			Description   *string        `json:"description"`
			Configuration map[string]any `json:"configuration"`
		}
		if err := json.Unmarshal(body, &req); err != nil || req.Configuration == nil {
			return nil, "configuration is required"
		}
		if t, _ := req.Configuration["configurationType"].(string); t != k.typ {
			return nil, "configurationType must be " + k.typ
		}
		if msg := k.validate(req.Configuration); msg != "" {
			return nil, msg
		}
		o := map[string]any{"account_sid": testAccount, "unique_name": nil, "description": nil, "configuration": req.Configuration}
		if req.UniqueName != nil {
			o["unique_name"] = *req.UniqueName
		}
		if req.Description != nil {
			o["description"] = *req.Description
		}
		return o, ""
	}
	if len(parts) == 2 {
		if r.Method != "POST" {
			ferr(w, 405, "method not allowed")
			return
		}
		o, msg := read()
		if msg != "" {
			ferr(w, 400, msg)
			return
		}
		if un, ok := o["unique_name"].(string); ok && f.findVoiceV2(k, un) != nil {
			ferr(w, 400, "unique_name already exists")
			return
		}
		id := f.sid(k.prefix)
		o["id"], o["date_created"], o["date_updated"] = id, voiceStamp, voiceStamp
		f.put(k.kind, testAccount, id, o)
		fwrite(w, 201, o)
		return
	}
	o := f.findVoiceV2(k, parts[2])
	if o == nil {
		ferr(w, 404, "The requested resource "+r.URL.Path+" was not found")
		return
	}
	switch r.Method {
	case "GET":
		fwrite(w, 200, o)
	case "PUT":
		n, msg := read()
		if msg != "" {
			ferr(w, 400, msg)
			return
		}
		for _, key := range []string{"unique_name", "description", "configuration"} {
			o[key] = n[key]
		}
		o["date_updated"] = voiceUpdatedStamp
		fwrite(w, 200, o)
	case "DELETE":
		f.del(k.kind, testAccount, o["id"].(string))
		w.WriteHeader(204)
	default:
		ferr(w, 405, "method not allowed")
	}
}

// findVoiceV2 returns the configuration with the given id or unique name.
func (f *fakeTwilio) findVoiceV2(k voiceV2Kind, idOrName string) map[string]any {
	if o, ok := f.objs[key(k.kind, testAccount, idOrName)]; ok {
		return o
	}
	for kk, o := range f.objs {
		if strings.HasPrefix(kk, k.kind+"/") && o["unique_name"] == idOrName {
			return o
		}
	}
	return nil
}
