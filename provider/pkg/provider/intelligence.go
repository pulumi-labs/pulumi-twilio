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
)

// The resources in this file family configure Voice Intelligence (intelligence.twilio.com v2):
// services and custom operators. Transcripts, their sentences and operator results are runtime data
// and are not managed; prebuilt operators and operator types are a read-only catalog. The shapes
// follow twilio-oai spec/json/twilio_intelligence_v2.json.
//
// Optional settings follow the TaskRouter rule: an input that is set is sent and kept in step with
// Twilio (drift shows in a preview); an input that is removed leaves the setting as it is on
// Twilio, because the spec states no empty value to reset to. The optional If-Match ETag header of
// the update endpoints is not sent, so the last write wins.

const intelligenceProduct = "intelligence"

var (
	intelligenceServiceSidRE    = regexp.MustCompile(`^GA[0-9a-fA-F]{32}$`)
	intelligenceOperatorSidRE   = regexp.MustCompile(`^LY[0-9a-fA-F]{32}$`)
	intelligenceCredentialSidRE = regexp.MustCompile(`^CR[0-9a-fA-F]{32}$`)
)

// intelligencePath is /v2/<coll>[/sid].
func intelligencePath(coll, sid string) string {
	path := "/v2/" + coll
	if sid != "" {
		path += "/" + url.PathEscape(sid)
	}
	return path
}

// jsonObject reports whether s is a JSON object.
func jsonObject(s string) bool {
	var m map[string]any
	return json.Unmarshal([]byte(s), &m) == nil && m != nil
}

// objJSON renders the JSON value at key k of an API object as a compact document; nil when absent.
func objJSON(o obj, k string) *string {
	v := o.get(k)
	if v == nil {
		return nil
	}
	raw, err := json.Marshal(v)
	if err != nil {
		return nil
	}
	s := string(raw)
	return &s
}
