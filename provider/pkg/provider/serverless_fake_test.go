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

// The Serverless v1 configuration plane, served for the service, environment, variable, function and
// asset tests:
//
//	/serverless/v1/Services[/{svc}]
//	/serverless/v1/Services/{svc}/Environments[/{env}]
//	/serverless/v1/Services/{svc}/Environments/{env}/Variables[/{var}]
//	/serverless/v1/Services/{svc}/Functions[/{sid}]
//	/serverless/v1/Services/{svc}/Assets[/{sid}]
//
// The shapes follow twilio-oai spec/json/twilio_serverless_v1.json (spec version 1.0.0). A service
// delete removes everything inside it, as Twilio does.

const (
	serverlessStamp        = "2026-11-06T00:00:00Z"
	serverlessUpdatedStamp = "2026-11-07T00:00:00Z"
)

func (f *fakeTwilio) handleServerless(w http.ResponseWriter, r *http.Request, form url.Values, parts []string) {
	notFound := func() { ferr(w, 404, "The requested resource "+r.URL.Path+" was not found") }
	if len(parts) == 0 {
		f.serverlessService(w, r, form, "", notFound)
		return
	}
	svc := parts[0]
	svcObj, ok := f.objs[key("zs", testAccount, svc)]
	if !ok {
		notFound()
		return
	}
	if len(parts) == 1 {
		f.serverlessService(w, r, form, svc, notFound)
		return
	}
	sid := ""
	if len(parts) >= 3 {
		sid = parts[2]
	}
	switch {
	case parts[1] == "Environments" && len(parts) <= 3:
		f.serverlessEnvironment(w, r, form, svcObj, sid, notFound)
	case parts[1] == "Environments" && len(parts) >= 4 && parts[3] == "Variables" && len(parts) <= 5:
		if _, ok := f.objs[key("ze", testAccount, svc+"/"+parts[2])]; !ok {
			notFound()
			return
		}
		vsid := ""
		if len(parts) == 5 {
			vsid = parts[4]
		}
		f.serverlessVariable(w, r, form, svc, parts[2], vsid, notFound)
	case (parts[1] == "Functions" || parts[1] == "Assets") && len(parts) <= 3:
		f.serverlessContainer(w, r, form, svc, parts[1], sid, notFound)
	case (parts[1] == "Functions" || parts[1] == "Assets") && len(parts) >= 5 && len(parts) <= 6 && parts[3] == "Versions":
		f.serverlessVersion(w, r, svc, parts, notFound)
	case parts[1] == "Builds" && len(parts) <= 4:
		f.serverlessBuild(w, r, form, svc, parts, notFound)
	case parts[1] == "Environments" && len(parts) >= 4 && parts[3] == "Deployments" && len(parts) <= 5:
		if _, ok := f.objs[key("ze", testAccount, svc+"/"+parts[2])]; !ok {
			notFound()
			return
		}
		dsid := ""
		if len(parts) == 5 {
			dsid = parts[4]
		}
		f.serverlessDeployment(w, r, form, svc, parts[2], dsid, notFound)
	default:
		notFound()
	}
}

func (f *fakeTwilio) serverlessService(w http.ResponseWriter, r *http.Request, form url.Values, svc string, notFound func()) {
	apply := func(o map[string]any) {
		if v, ok := form["FriendlyName"]; ok {
			o["friendly_name"] = v[0]
		}
		for param, k := range map[string]string{"IncludeCredentials": "include_credentials", "UiEditable": "ui_editable"} {
			if v, ok := form[param]; ok {
				o[k] = v[0] == "true"
			}
		}
	}
	if svc == "" {
		if r.Method != "POST" {
			ferr(w, 405, "method not allowed")
			return
		}
		for _, p := range []string{"UniqueName", "FriendlyName"} {
			if form.Get(p) == "" {
				ferr(w, 400, "Missing required parameter "+p)
				return
			}
		}
		for _, o := range f.objs {
			if o["unique_name"] == form.Get("UniqueName") && strings.HasPrefix(o["sid"].(string), "ZS") {
				ferr(w, 409, "Service with unique name already exists")
				return
			}
		}
		sid := f.sid("ZS")
		o := map[string]any{"sid": sid, "account_sid": testAccount, "unique_name": form.Get("UniqueName"), "friendly_name": "",
			"include_credentials": true, "ui_editable": false, "domain_base": "zoo-" + sid[2:6],
			"url": "https://serverless.twilio.com/v1/Services/" + sid, "date_created": serverlessStamp, "date_updated": serverlessStamp}
		apply(o)
		f.put("zs", testAccount, sid, o)
		fwrite(w, 201, o)
		return
	}
	o := f.objs[key("zs", testAccount, svc)]
	switch r.Method {
	case "GET":
		fwrite(w, 200, o)
	case "POST":
		apply(o)
		o["date_updated"] = serverlessUpdatedStamp
		fwrite(w, 200, o)
	case "DELETE":
		for k, c := range f.objs {
			if c["service_sid"] == svc {
				delete(f.objs, k)
			}
		}
		f.del("zs", testAccount, svc)
		w.WriteHeader(204)
	default:
		ferr(w, 405, "method not allowed")
	}
}

func (f *fakeTwilio) serverlessEnvironment(w http.ResponseWriter, r *http.Request, form url.Values, svcObj map[string]any, sid string, notFound func()) {
	svc := svcObj["sid"].(string)
	if sid == "" {
		if r.Method != "POST" {
			ferr(w, 405, "method not allowed")
			return
		}
		name := form.Get("UniqueName")
		if name == "" {
			ferr(w, 400, "Missing required parameter UniqueName")
			return
		}
		for _, o := range f.objs {
			if o["service_sid"] == svc && o["unique_name"] == name && strings.HasPrefix(o["sid"].(string), "ZE") {
				ferr(w, 409, "Environment with this unique name already exists")
				return
			}
		}
		suffix := form.Get("DomainSuffix")
		if suffix == "" {
			suffix = name
		}
		id := f.sid("ZE")
		o := map[string]any{"sid": id, "service_sid": svc, "account_sid": testAccount, "unique_name": name, "build_sid": nil,
			"domain_name": svcObj["domain_base"].(string) + "-" + suffix + ".twil.io", "url": "https://serverless.twilio.com/v1/Services/" + svc + "/Environments/" + id,
			"date_created": serverlessStamp, "date_updated": serverlessStamp}
		f.put("ze", testAccount, svc+"/"+id, o)
		fwrite(w, 201, o)
		return
	}
	o, ok := f.objs[key("ze", testAccount, svc+"/"+sid)]
	if !ok {
		notFound()
		return
	}
	switch r.Method {
	case "GET":
		fwrite(w, 200, o)
	case "DELETE":
		for k, c := range f.objs {
			if c["environment_sid"] == sid {
				delete(f.objs, k)
			}
		}
		f.del("ze", testAccount, svc+"/"+sid)
		w.WriteHeader(204)
	default:
		ferr(w, 405, "method not allowed") // an environment has no update
	}
}

func (f *fakeTwilio) serverlessVariable(w http.ResponseWriter, r *http.Request, form url.Values, svc, env, sid string, notFound func()) {
	if sid == "" {
		if r.Method != "POST" {
			ferr(w, 405, "method not allowed")
			return
		}
		for _, p := range []string{"Key", "Value"} {
			if _, ok := form[p]; !ok {
				ferr(w, 400, "Missing required parameter "+p)
				return
			}
		}
		for _, o := range f.objs {
			if o["environment_sid"] == env && o["key"] == form.Get("Key") {
				ferr(w, 409, "Variable with this key already exists")
				return
			}
		}
		id := f.sid("ZV")
		o := map[string]any{"sid": id, "service_sid": svc, "environment_sid": env, "account_sid": testAccount, "key": form.Get("Key"),
			"value": form.Get("Value"), "date_created": serverlessStamp, "date_updated": serverlessStamp}
		f.put("zv", testAccount, svc+"/"+env+"/"+id, o)
		fwrite(w, 201, o)
		return
	}
	o, ok := f.objs[key("zv", testAccount, svc+"/"+env+"/"+sid)]
	if !ok {
		notFound()
		return
	}
	switch r.Method {
	case "GET":
		fwrite(w, 200, o)
	case "POST":
		for param, k := range map[string]string{"Key": "key", "Value": "value"} {
			if v, ok := form[param]; ok {
				o[k] = v[0]
			}
		}
		o["date_updated"] = serverlessUpdatedStamp
		fwrite(w, 200, o)
	case "DELETE":
		f.del("zv", testAccount, svc+"/"+env+"/"+sid)
		w.WriteHeader(204)
	default:
		ferr(w, 405, "method not allowed")
	}
}

func (f *fakeTwilio) serverlessContainer(w http.ResponseWriter, r *http.Request, form url.Values, svc, coll, sid string, notFound func()) {
	kind := map[string]string{"Functions": "zf", "Assets": "za"}[coll]
	if sid == "" {
		if r.Method != "POST" {
			ferr(w, 405, "method not allowed")
			return
		}
		if form.Get("FriendlyName") == "" {
			ferr(w, 400, "Missing required parameter FriendlyName")
			return
		}
		id := f.sid("ZH")
		o := map[string]any{"sid": id, "service_sid": svc, "account_sid": testAccount, "friendly_name": form.Get("FriendlyName"),
			"url": "https://serverless.twilio.com/v1/Services/" + svc + "/" + coll + "/" + id, "date_created": serverlessStamp, "date_updated": serverlessStamp}
		f.put(kind, testAccount, svc+"/"+id, o)
		fwrite(w, 201, o)
		return
	}
	o, ok := f.objs[key(kind, testAccount, svc+"/"+sid)]
	if !ok {
		notFound()
		return
	}
	switch r.Method {
	case "GET":
		fwrite(w, 200, o)
	case "POST":
		if v, ok := form["FriendlyName"]; ok {
			o["friendly_name"] = v[0]
		}
		o["date_updated"] = serverlessUpdatedStamp
		fwrite(w, 200, o)
	case "DELETE":
		if f.containerReferencedByBuild(svc, coll, sid) {
			fwrite(w, 400, map[string]any{"code": 20001, "status": 400, "message": "Cannot delete " + strings.ToLower(strings.TrimSuffix(coll, "s")) + ". One or more of its versions is referenced in a Build."})
			return
		}
		f.del(kind, testAccount, svc+"/"+sid)
		w.WriteHeader(204)
	default:
		ferr(w, 405, "method not allowed")
	}
}
