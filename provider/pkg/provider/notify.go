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

// The resources in this file family configure Notify (notify.twilio.com v1): services and the push
// credentials they use. Bindings and notifications are runtime data and are not managed. The
// shapes follow twilio-oai spec/json/twilio_notify_v1.json.
//
// Optional settings follow the TaskRouter rule: an input that is set is sent and kept in step with
// Twilio (drift shows in a preview); an input that is removed leaves the setting as it is on
// Twilio, because the spec states no empty value to reset to.

const notifyProduct = "notify"

// notifyPath is /v1/<coll>[/sid].
func notifyPath(coll, sid string) string {
	path := "/v1/" + coll
	if sid != "" {
		path += "/" + url.PathEscape(sid)
	}
	return path
}
