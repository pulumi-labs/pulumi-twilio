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

// The Conversations v1 routes, served for the Conversations tests:
//
//	/conversations/v1/Services[/{svc}]
//	/conversations/v1/Services/{svc}/Configuration[/Webhooks]
//	/conversations/v1/Services/{svc}/Roles[/{sid}]
//	/conversations/v1/Roles[/{sid}], /Credentials[/{sid}], /Configuration/Addresses[/{sid}], /Configuration/Webhooks
//
// The shapes follow twilio-oai spec/json/twilio_conversations_v1.json. Objects are stored as kind
// "cv" keyed <collection>/<scope>/<sid>, singletons as kind "cvone" keyed <name>/<scope>.

const (
	cvStamp        = "2026-11-12T00:00:00Z"
	cvUpdatedStamp = "2026-11-12T01:00:00Z"
)

type cvSpec struct {
	prefix   string
	required []string
	create   []fakeField
	update   []fakeField
}

var cvSpecs = map[string]cvSpec{
	"Services": {"IS", []string{"FriendlyName"}, []fakeField{{"FriendlyName", "friendly_name", "s"}}, nil},
	"Roles": {"RL", []string{"FriendlyName", "Type", "Permission"},
		[]fakeField{{"FriendlyName", "friendly_name", "s"}, {"Type", "type", "s"}}, nil},
	"Credentials": {"CR", []string{"Type"},
		[]fakeField{{"Type", "type", "s"}, {"FriendlyName", "friendly_name", "s"}, {"Sandbox", "sandbox", "b"}},
		[]fakeField{{"FriendlyName", "friendly_name", "s"}, {"Sandbox", "sandbox", "b"}}},
	"Configuration/Addresses": {"IG", []string{"Type", "Address"},
		[]fakeField{{"Type", "type", "s"}, {"Address", "address", "s"}, {"FriendlyName", "friendly_name", "s"}, {"AddressCountry", "address_country", "s"},
			{"AutoCreation.Enabled", "auto_creation.enabled", "b"}, {"AutoCreation.Type", "auto_creation.type", "u"},
			{"AutoCreation.ConversationServiceSid", "auto_creation.conversation_service_sid", "u"}, {"AutoCreation.WebhookUrl", "auto_creation.webhook_url", "u"},
			{"AutoCreation.WebhookMethod", "auto_creation.webhook_method", "m"}, {"AutoCreation.StudioFlowSid", "auto_creation.studio_flow_sid", "u"},
			{"AutoCreation.StudioRetryCount", "auto_creation.studio_retry_count", "i"}},
		[]fakeField{{"FriendlyName", "friendly_name", "s"},
			{"AutoCreation.Enabled", "auto_creation.enabled", "b"}, {"AutoCreation.Type", "auto_creation.type", "u"},
			{"AutoCreation.ConversationServiceSid", "auto_creation.conversation_service_sid", "u"}, {"AutoCreation.WebhookUrl", "auto_creation.webhook_url", "u"},
			{"AutoCreation.WebhookMethod", "auto_creation.webhook_method", "m"}, {"AutoCreation.StudioFlowSid", "auto_creation.studio_flow_sid", "u"},
			{"AutoCreation.StudioRetryCount", "auto_creation.studio_retry_count", "i"}}},
}

var (
	cvServiceConfigFields = []fakeField{
		{"DefaultConversationCreatorRoleSid", "default_conversation_creator_role_sid", "u"},
		{"DefaultConversationRoleSid", "default_conversation_role_sid", "u"},
		{"DefaultChatServiceRoleSid", "default_chat_service_role_sid", "u"},
		{"ReachabilityEnabled", "reachability_enabled", "b"},
	}
	cvWebhookFields = []fakeField{
		{"Method", "method", "m"}, {"PreWebhookUrl", "pre_webhook_url", "u"}, {"PostWebhookUrl", "post_webhook_url", "u"},
		{"Target", "target", "u"},
	}
)

// cvList stores every value of a repeated form key, the way Twilio takes arrays.
func cvListParam(o map[string]any, form url.Values, param, key string) {
	vals, ok := form[param]
	if !ok {
		return
	}
	list := []any{}
	for _, v := range vals {
		list = append(list, v)
	}
	setPath(o, key, list)
}

func (f *fakeTwilio) handleConversations(w http.ResponseWriter, r *http.Request, form url.Values, rest string) {
	notFound := func() { ferr(w, 404, "The requested resource "+r.URL.Path+" was not found") }
	parts := strings.Split(rest, "/")
	switch {
	case parts[0] == "Services" && len(parts) >= 3 && parts[2] == "Configuration":
		if _, ok := f.objs[key("cv", testAccount, "Services//"+parts[1])]; !ok {
			notFound()
			return
		}
		switch {
		case len(parts) == 3:
			f.cvSingleton(w, r, form, "svccfg", parts[1], cvServiceConfigFields, notFound)
		case len(parts) == 4 && parts[3] == "Webhooks":
			f.cvSingleton(w, r, form, "svcwh", parts[1], cvWebhookFields, notFound)
		default:
			notFound()
		}
	case parts[0] == "Services" && len(parts) >= 3 && parts[2] == "Roles" && len(parts) <= 4:
		if _, ok := f.objs[key("cv", testAccount, "Services//"+parts[1])]; !ok {
			notFound()
			return
		}
		sid := ""
		if len(parts) == 4 {
			sid = parts[3]
		}
		f.cvCollection(w, r, form, "Roles", parts[1], sid, notFound)
	case parts[0] == "Configuration" && len(parts) >= 2 && parts[1] == "Webhooks" && len(parts) == 2:
		f.cvSingleton(w, r, form, "acctwh", "", cvWebhookFields, notFound)
	default:
		coll, sid := parts[0], ""
		if parts[0] == "Configuration" && len(parts) >= 2 {
			coll = "Configuration/Addresses"
			if parts[1] != "Addresses" {
				notFound()
				return
			}
			parts = parts[1:]
		}
		if len(parts) == 2 {
			sid = parts[1]
		} else if len(parts) > 2 {
			notFound()
			return
		}
		if _, ok := cvSpecs[coll]; !ok {
			notFound()
			return
		}
		f.cvCollection(w, r, form, coll, "", sid, notFound)
	}
}

func (f *fakeTwilio) cvCollection(w http.ResponseWriter, r *http.Request, form url.Values, coll, scope, sid string, notFound func()) {
	spec := cvSpecs[coll]
	if sid == "" {
		if r.Method != "POST" {
			ferr(w, 405, "method not allowed")
			return
		}
		for _, req := range spec.required {
			if len(form[req]) == 0 || form.Get(req) == "" {
				ferr(w, 400, "Missing required parameter "+req)
				return
			}
		}
		id := f.sid(spec.prefix)
		o := map[string]any{"sid": id, "account_sid": testAccount, "date_created": cvStamp, "date_updated": cvStamp}
		switch coll {
		case "Roles":
			o["chat_service_sid"] = nilIfEmpty(scope)
			o["permissions"] = []any{}
		case "Credentials":
			o["sandbox"] = "false"
		case "Configuration/Addresses":
			o["auto_creation"] = map[string]any{"enabled": false, "type": nil, "conversation_service_sid": nil, "webhook_url": nil,
				"webhook_method": nil, "webhook_filters": []any{}, "studio_flow_sid": nil, "studio_retry_count": nil}
			o["address_country"] = nil
		}
		if msg := applyForm(o, form, spec.create); msg != "" {
			ferr(w, 400, msg)
			return
		}
		cvListParam(o, form, "Permission", "permissions")
		cvListParam(o, form, "AutoCreation.WebhookFilters", "auto_creation.webhook_filters")
		f.put("cv", testAccount, coll+"/"+scope+"/"+id, o)
		fwrite(w, 201, o)
		return
	}
	k := coll + "/" + scope + "/" + sid
	o, ok := f.objs[key("cv", testAccount, k)]
	if !ok || !strings.HasPrefix(sid, spec.prefix) {
		notFound()
		return
	}
	switch r.Method {
	case "GET":
		fwrite(w, 200, o)
	case "POST":
		if coll == "Services" {
			ferr(w, 405, "method not allowed")
			return
		}
		fields := spec.update
		if fields == nil {
			fields = spec.create
		}
		if msg := applyForm(o, form, fields); msg != "" {
			ferr(w, 400, msg)
			return
		}
		cvListParam(o, form, "Permission", "permissions")
		cvListParam(o, form, "AutoCreation.WebhookFilters", "auto_creation.webhook_filters")
		o["date_updated"] = cvUpdatedStamp
		fwrite(w, 200, o)
	case "DELETE":
		f.del("cv", testAccount, k)
		w.WriteHeader(204)
	default:
		ferr(w, 405, "method not allowed")
	}
}

func nilIfEmpty(s string) any {
	if s == "" {
		return nil
	}
	return s
}

// cvSingleton serves a configuration that always exists once its service does.
func (f *fakeTwilio) cvSingleton(w http.ResponseWriter, r *http.Request, form url.Values, name, scope string, fields []fakeField, notFound func()) {
	k := key("cvone", testAccount, name+"/"+scope)
	o, ok := f.objs[k]
	if !ok {
		o = map[string]any{"account_sid": testAccount, "chat_service_sid": nilIfEmpty(scope), "url": r.URL.Path}
		switch name {
		case "svccfg":
			o["default_conversation_creator_role_sid"], o["default_conversation_role_sid"], o["default_chat_service_role_sid"] = nil, nil, nil
			o["reachability_enabled"] = false
		default:
			o["method"], o["pre_webhook_url"], o["post_webhook_url"], o["filters"] = "POST", nil, nil, []any{}
			if scope == "" {
				o["target"] = "webhook"
			}
		}
		f.put("cvone", testAccount, name+"/"+scope, o)
	}
	switch r.Method {
	case "GET":
		fwrite(w, 200, o)
	case "POST":
		if msg := applyForm(o, form, fields); msg != "" {
			ferr(w, 400, msg)
			return
		}
		cvListParam(o, form, "Filters", "filters")
		fwrite(w, 200, o)
	default:
		ferr(w, 405, "method not allowed")
	}
}
