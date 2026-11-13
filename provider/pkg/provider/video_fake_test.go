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
	"net/url"
	"strings"
)

// The Video v1 routes, served for the Video tests:
//
//	/video/v1/CompositionHooks[/{sid}], /video/v1/CompositionSettings/Default, /video/v1/RecordingSettings/Default
//
// The shapes follow twilio-oai spec/json/twilio_video_v1.json. Objects are stored as kind "vd"
// keyed <collection>/<sid>. Lists arrive as repeated form keys; one empty value is an empty list.

var videoHookFakeFields = []fakeField{
	{"FriendlyName", "friendly_name", "s"}, {"Enabled", "enabled", "b"}, {"Resolution", "resolution", "s"}, {"Format", "format", "s"},
	{"Trim", "trim", "b"}, {"StatusCallback", "status_callback", "u"}, {"StatusCallbackMethod", "status_callback_method", "m"},
}

func videoFakeList(form url.Values, param string) []any {
	out := []any{}
	for _, v := range form[param] {
		if v != "" {
			out = append(out, v)
		}
	}
	return out
}

func (f *fakeTwilio) handleVideo(w http.ResponseWriter, r *http.Request, form url.Values, rest string) {
	parts := strings.Split(rest, "/")
	if len(parts) == 2 && parts[1] == "Default" && (parts[0] == "CompositionSettings" || parts[0] == "RecordingSettings") {
		f.videoSettings(w, r, form, parts[0])
		return
	}
	if parts[0] != "CompositionHooks" || len(parts) > 2 {
		ferr(w, 404, "The requested resource "+r.URL.Path+" was not found")
		return
	}
	apply := func(o map[string]any) string {
		if msg := applyForm(o, form, videoHookFakeFields); msg != "" {
			return msg
		}
		if v, ok := form["VideoLayout"]; ok {
			var layout map[string]any
			if json.Unmarshal([]byte(v[0]), &layout) != nil {
				return "VideoLayout must be a JSON object"
			}
			o["video_layout"] = layout
		}
		for _, l := range [][2]string{{"AudioSources", "audio_sources"}, {"AudioSourcesExcluded", "audio_sources_excluded"}} {
			if _, ok := form[l[0]]; ok {
				o[l[1]] = videoFakeList(form, l[0])
			}
		}
		return ""
	}
	if len(parts) == 1 {
		if r.Method != "POST" {
			ferr(w, 405, "method not allowed")
			return
		}
		if form.Get("FriendlyName") == "" {
			ferr(w, 400, "Missing required parameter FriendlyName")
			return
		}
		id := f.sid("HK")
		o := map[string]any{"sid": id, "account_sid": testAccount, "date_created": cvStamp, "date_updated": cvStamp,
			"enabled": true, "video_layout": map[string]any{}, "audio_sources": []any{}, "audio_sources_excluded": []any{},
			"resolution": "640x480", "format": "mp4", "trim": true, "status_callback": nil, "status_callback_method": "POST"}
		if msg := apply(o); msg != "" {
			ferr(w, 400, msg)
			return
		}
		f.put("vd", testAccount, "CompositionHooks/"+id, o)
		fwrite(w, 201, o)
		return
	}
	sid := parts[1]
	k := "CompositionHooks/" + sid
	o, ok := f.objs[key("vd", testAccount, k)]
	if !ok || !strings.HasPrefix(sid, "HK") {
		ferr(w, 404, "The requested resource "+r.URL.Path+" was not found")
		return
	}
	switch r.Method {
	case "GET":
		fwrite(w, 200, o)
	case "POST":
		if msg := apply(o); msg != "" {
			ferr(w, 400, msg)
			return
		}
		o["date_updated"] = cvUpdatedStamp
		fwrite(w, 200, o)
	case "DELETE":
		f.del("vd", testAccount, k)
		w.WriteHeader(204)
	default:
		ferr(w, 405, "method not allowed")
	}
}

var videoSettingsFakeFields = []fakeField{
	{"FriendlyName", "friendly_name", "s"}, {"AwsCredentialsSid", "aws_credentials_sid", "u"}, {"AwsS3Url", "aws_s3_url", "u"},
	{"AwsStorageEnabled", "aws_storage_enabled", "b"}, {"EncryptionKeySid", "encryption_key_sid", "u"}, {"EncryptionEnabled", "encryption_enabled", "b"},
}

// videoSettings serves a settings singleton: it always exists, GET reads it and POST replaces the
// fields it names. There is no DELETE.
func (f *fakeTwilio) videoSettings(w http.ResponseWriter, r *http.Request, form url.Values, name string) {
	k := key("vd", testAccount, name+"/Default")
	o, ok := f.objs[k]
	if !ok {
		o = map[string]any{"account_sid": testAccount, "friendly_name": nil, "aws_credentials_sid": nil, "aws_s3_url": nil,
			"aws_storage_enabled": false, "encryption_key_sid": nil, "encryption_enabled": false, "url": r.URL.Path}
		f.put("vd", testAccount, name+"/Default", o)
	}
	switch r.Method {
	case "GET":
		fwrite(w, 200, o)
	case "POST":
		if form.Get("FriendlyName") == "" {
			ferr(w, 400, "Missing required parameter FriendlyName")
			return
		}
		if msg := applyForm(o, form, videoSettingsFakeFields); msg != "" {
			ferr(w, 400, msg)
			return
		}
		fwrite(w, 200, o)
	default:
		ferr(w, 405, "method not allowed")
	}
}
