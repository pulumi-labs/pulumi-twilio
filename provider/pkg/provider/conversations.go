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
	"strings"

	p "github.com/pulumi/pulumi-go-provider"
)

// The resources in this file family configure Conversations (conversations.twilio.com v1): a
// service, its configuration, roles, push credentials, address configurations and webhook
// configuration. Conversations, participants, messages, users and bindings are runtime data and
// are not managed. The shapes follow twilio-oai spec/json/twilio_conversations_v1.json.
//
// Optional settings follow the TaskRouter rule: an input that is set is sent and kept in step with
// Twilio (drift shows in a preview); an input that is removed leaves the setting as it is on
// Twilio, because the spec states no empty value to reset to.

const cvProduct = "conversations"

var (
	cvServiceSidRE    = regexp.MustCompile(`^IS[0-9a-fA-F]{32}$`)
	cvRoleSidRE       = regexp.MustCompile(`^RL[0-9a-fA-F]{32}$`)
	cvCredentialSidRE = regexp.MustCompile(`^CR[0-9a-fA-F]{32}$`)
	cvAddressSidRE    = regexp.MustCompile(`^IG[0-9a-fA-F]{32}$`)
)

// cvPath is /v1/Services[/service[/coll[/sid]]]; an empty service gives the account-level /v1/coll[/sid].
func cvPath(service, coll, sid string) string {
	path := "/v1"
	if service != "" {
		path += "/Services/" + url.PathEscape(service)
	}
	if coll != "" {
		path += "/" + coll
	}
	if sid != "" {
		path += "/" + url.PathEscape(sid)
	}
	return path
}

func cvCheckOptionalService(c *checker, prop string, v *string) {
	if v != nil {
		c.sid(prop, *v, cvServiceSidRE, "Conversations service (IS followed by 32 hex characters)")
	}
}

// cvForm sets a list parameter as repeated form keys, the way Twilio takes arrays.
func cvAddList(form url.Values, key string, l *[]string) {
	if l == nil {
		return
	}
	for _, v := range *l {
		form.Add(key, v)
	}
}

// strList records an update of an optional list; the order is not significant and removing the input is not a change.
func (d trDiffSet) strList(name string, o, n *[]string) {
	if n != nil && (o == nil || !sameSet(*o, *n)) {
		d[name] = p.PropertyDiff{Kind: p.Update}
	}
}

// cvList is trStr for a list: an input that is set follows Twilio (a different order is not drift),
// an unset one stays unset unless this is an import, which adopts a non-empty list.
func cvList(in, api *[]string, imported bool) *[]string {
	api = nonEmptyList(api)
	switch {
	case imported:
		return api
	case in == nil:
		return nil
	case api == nil:
		empty := []string{}
		return &empty
	case sameSet(*in, *api):
		return in
	}
	return api
}

// cvUpper normalizes an HTTP method in place; anything other than GET or POST is a failure.
func cvUpper(c *checker, prop string, v *string) {
	if v == nil || c.unk(prop) {
		return
	}
	up := strings.ToUpper(*v)
	if !httpMethods[up] {
		c.fail(prop, prop+" must be GET or POST (any case)")
		return
	}
	*v = up
}
