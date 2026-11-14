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
	"net/http"
	"net/url"
	"strings"
)

// The Flex v1 routes, served for the Flex tests:
//
//	/flex-api/v1/FlexFlows[/{sid}]
//	/flex-api/v1/PluginService/{Plugins,Configurations,Releases}[/{sid}[/Archive]]
//	/flex-api/v1/PluginService/Plugins/{sid}/Versions[/{sid}[/Archive]]
//	/flex-api/v1/PluginService/Configurations/{sid}/Plugins
//
// The shape follows twilio-oai spec/json/twilio_flex_v1.json. Objects are stored as kind "flex"
// keyed <collection>/<sid>.

var flexFlowFields = []fakeField{
	{"FriendlyName", "friendly_name", "s"}, {"ChatServiceSid", "chat_service_sid", "s"}, {"ChannelType", "channel_type", "s"},
	{"ContactIdentity", "contact_identity", "s"}, {"Enabled", "enabled", "b"}, {"IntegrationType", "integration_type", "s"},
	{"Integration.FlowSid", "integration.flow_sid", "s"}, {"Integration.Url", "integration.url", "s"},
	{"Integration.WorkspaceSid", "integration.workspace_sid", "s"}, {"Integration.WorkflowSid", "integration.workflow_sid", "s"},
	{"Integration.Channel", "integration.channel", "s"}, {"Integration.Timeout", "integration.timeout", "i"},
	{"Integration.Priority", "integration.priority", "i"}, {"Integration.CreationOnMessage", "integration.creation_on_message", "b"},
	{"Integration.RetryCount", "integration.retry_count", "i"}, {"LongLived", "long_lived", "b"}, {"JanitorEnabled", "janitor_enabled", "b"},
}

// flexList returns the stored objects of a collection in creation order.
func (f *fakeTwilio) flexList(coll string, keep func(map[string]any) bool) []map[string]any {
	out := []map[string]any{}
	prefix := "flex/" + testAccount + "/" + coll + "/"
	seen := map[string]bool{}
	for _, fk := range f.order {
		o, ok := f.objs[fk]
		if !ok || seen[fk] || !strings.HasPrefix(fk, prefix) {
			continue
		}
		seen[fk] = true
		if keep == nil || keep(o) {
			out = append(out, o)
		}
	}
	return out
}

func (f *fakeTwilio) handleFlex(w http.ResponseWriter, r *http.Request, form url.Values, rest string) {
	parts := strings.Split(rest, "/")
	if parts[0] == "PluginService" {
		f.flexPlugins(w, r, form, parts[1:])
		return
	}
	if parts[0] != "FlexFlows" || len(parts) > 2 {
		ferr(w, 404, "The requested resource "+r.URL.Path+" was not found")
		return
	}
	f.flexFlows(w, r, form, parts)
}

// flexApply applies the posted fields and the rules the real API enforces.
func flexApply(o map[string]any, form url.Values) string {
	if msg := applyForm(o, form, flexFlowFields); msg != "" {
		return msg
	}
	if v, _ := o["channel_type"].(string); v != "" {
		ok := false
		for _, t := range flexChannelTypes {
			ok = ok || t == v
		}
		if !ok {
			return "Invalid ChannelType " + v
		}
	}
	if n, ok := o["integration"].(map[string]any)["retry_count"].(float64); ok && (n < 0 || n > 3) {
		return "Integration.RetryCount must be between 0 and 3"
	}
	return ""
}

func (f *fakeTwilio) flexFlows(w http.ResponseWriter, r *http.Request, form url.Values, parts []string) {
	if len(parts) == 1 {
		switch r.Method {
		case "POST":
			for _, req := range []string{"FriendlyName", "ChatServiceSid", "ChannelType"} {
				if form.Get(req) == "" {
					ferr(w, 400, "Missing required parameter "+req)
					return
				}
			}
			id := f.sid("FO")
			o := map[string]any{"sid": id, "account_sid": testAccount, "date_created": cvStamp, "date_updated": cvStamp,
				"url": "https://flex-api.twilio.com/v1/FlexFlows/" + id, "enabled": true, "long_lived": false, "janitor_enabled": false,
				"contact_identity": nil, "integration_type": nil, "integration": map[string]any{}}
			if msg := flexApply(o, form); msg != "" {
				ferr(w, 400, msg)
				return
			}
			// Twilio fills the integration defaults of the type.
			in := o["integration"].(map[string]any)
			switch o["integration_type"] {
			case "task":
				for k, v := range map[string]any{"channel": "default", "timeout": float64(86400), "priority": float64(0)} {
					if in[k] == nil {
						in[k] = v
					}
				}
				if in["creation_on_message"] == nil {
					in["creation_on_message"] = false
				}
			case "studio", "external":
				if in["retry_count"] == nil {
					in["retry_count"] = float64(3)
				}
			}
			f.put("flex", testAccount, "FlexFlows/"+id, o)
			fwrite(w, 201, o)
		case "GET":
			name := r.URL.Query().Get("FriendlyName")
			metaPage(w, r, "flex_flows", f.flexList("FlexFlows", func(o map[string]any) bool { return name == "" || o["friendly_name"] == name }), 2)
		default:
			ferr(w, 405, "method not allowed")
		}
		return
	}
	o, ok := f.objs[key("flex", testAccount, "FlexFlows/"+parts[1])]
	if !ok {
		ferr(w, 404, "The requested resource "+r.URL.Path+" was not found")
		return
	}
	switch r.Method {
	case "GET":
		fwrite(w, 200, o)
	case "POST":
		if msg := flexApply(o, form); msg != "" {
			ferr(w, 400, msg)
			return
		}
		o["date_updated"] = cvUpdatedStamp
		fwrite(w, 200, o)
	case "DELETE":
		f.del("flex", testAccount, "FlexFlows/"+parts[1])
		w.WriteHeader(204)
	default:
		ferr(w, 405, "method not allowed")
	}
}
