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
	"net/url"
	"regexp"
)

// The resources in this file family configure Proxy (proxy.twilio.com v1): services and the phone
// numbers in a service's pool. Sessions, participants and interactions are runtime data and are not
// managed. Twilio removed Proxy short codes from its API, so they are not modelled. Proxy is a
// Public Beta product. The shapes follow twilio-oai spec/json/twilio_proxy_v1.json.
//
// Optional settings follow the TaskRouter rule: an input that is set is sent and kept in step with
// Twilio (drift shows in a preview); an input that is removed leaves the setting as it is on
// Twilio, because the spec states no empty value to reset to.

const proxyProduct = "proxy"

var (
	proxyServiceSidRE = regexp.MustCompile(`^KS[0-9a-fA-F]{32}$`)
	proxyNumberSidRE  = regexp.MustCompile(`^PN[0-9a-fA-F]{32}$`)
	proxyChatSidRE    = regexp.MustCompile(`^IS[0-9a-fA-F]{32}$`)
	proxyE164RE       = regexp.MustCompile(`^\+[1-9][0-9]{1,14}$`)
)

// proxyPath is /v1/Services[/sid[/coll[/sid]]]; empty elements are left out.
func proxyPath(parts ...string) string {
	path := "/v1/Services"
	for _, p := range parts {
		if p == "" {
			break
		}
		path += "/" + url.PathEscape(p)
	}
	return path
}
