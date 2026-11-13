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
	"net/url"
	"regexp"
	"slices"
	"strconv"

	p "github.com/pulumi/pulumi-go-provider"
)

// The resources in this file family configure Programmable Video (video.twilio.com v1): composition
// hooks. Rooms, participants, recordings and compositions are runtime data and are not managed. The
// shapes follow twilio-oai spec/json/twilio_video_v1.json.
//
// Optional settings follow the TaskRouter rule: an input that is set is sent and kept in step with
// Twilio (drift shows in a preview); an input that is removed leaves the setting as it is on
// Twilio, because the spec states no empty value to reset to.

const videoProduct = "video"

var (
	videoHookSidRE    = regexp.MustCompile(`^HK[0-9a-fA-F]{32}$`)
	videoResolutionRE = regexp.MustCompile(`^([0-9]+)x([0-9]+)$`)
)

// videoPath is /v1/<coll>[/sid].
func videoPath(coll, sid string) string {
	path := "/v1/" + coll
	if sid != "" {
		path += "/" + url.PathEscape(sid)
	}
	return path
}

// videoResolutionOK reports whether r is {width}x{height} within the limits Twilio documents:
// 16 <= width, height <= 1280 and width * height <= 921600.
func videoResolutionOK(r string) bool {
	m := videoResolutionRE.FindStringSubmatch(r)
	if m == nil {
		return false
	}
	w, err1 := strconv.Atoi(m[1])
	h, err2 := strconv.Atoi(m[2])
	return err1 == nil && err2 == nil && w >= 16 && w <= 1280 && h >= 16 && h <= 1280 && w*h <= 921600
}

// videoLayoutJSON renders the video layout Twilio reports as a compact JSON document; nil when it
// reports none (absent, null or an empty object).
func videoLayoutJSON(o obj, k string) *string {
	v := o.get(k)
	if m, ok := v.(map[string]any); v == nil || (ok && len(m) == 0) {
		return nil
	}
	raw, err := json.Marshal(v)
	if err != nil {
		return nil
	}
	s := string(raw)
	return &s
}

// videoListDiff records an update of an optional list. An unset stored list is an empty one, so a
// program that writes an empty list is not a change; order is not significant.
func (d trDiffSet) videoListDiff(name string, o, n *[]string) {
	if n == nil {
		return
	}
	var old []string
	if o != nil {
		old = *o
	}
	if !sameSet(old, *n) {
		d[name] = p.PropertyDiff{Kind: p.Update}
	}
}

// videoAddList sets a list parameter as repeated form keys. An empty list is sent as one empty
// value, which is how Twilio takes an empty array.
func videoAddList(form url.Values, key string, l *[]string) {
	if l == nil {
		return
	}
	if len(*l) == 0 {
		form.Set(key, "")
		return
	}
	for _, v := range slices.Clone(*l) {
		form.Add(key, v)
	}
}
