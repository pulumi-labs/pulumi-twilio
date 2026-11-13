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

import "net/url"

// The resources in this file family configure Sync (sync.twilio.com v1): a service. Documents,
// lists, maps, streams and permissions are runtime data and are not managed. The shapes follow
// twilio-oai spec/json/twilio_sync_v1.json.

const syncProduct = "sync"

// syncPath is /v1/Services[/sid].
func syncPath(sid string) string {
	path := "/v1/Services"
	if sid != "" {
		path += "/" + url.PathEscape(sid)
	}
	return path
}
