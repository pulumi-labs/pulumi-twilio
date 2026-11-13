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
	"bytes"
	"encoding/json"
	"io"
	"mime"
	"mime/multipart"
	"net/http"
	"net/url"
	"strings"
)

// The Serverless deploy plane on the fake:
//
//	POST /serverless-upload/v1/Services/{svc}/{Functions|Assets}/{sid}/Versions   (multipart; not in the OpenAPI document)
//	GET  /serverless/v1/Services/{svc}/{Functions|Assets}/{sid}/Versions/{sid}[/Content]
//	POST|GET|DELETE /serverless/v1/Services/{svc}/Builds[/{sid}[/Status]]
//	POST|GET /serverless/v1/Services/{svc}/Environments/{env}/Deployments[/{sid}]

type fakeUpload struct {
	Filename, ContentType, Path, Visibility string
	Content                                 []byte
}

func (f *fakeTwilio) handleServerlessUpload(w http.ResponseWriter, r *http.Request, body []byte, parts []string) {
	if r.Method != "POST" || len(parts) != 4 || parts[3] != "Versions" || (parts[1] != "Functions" && parts[1] != "Assets") {
		ferr(w, 404, "no route "+r.URL.Path)
		return
	}
	svc, coll, parent := parts[0], parts[1], parts[2]
	ckind, vkind := map[string]string{"Functions": "zf", "Assets": "za"}[coll], map[string]string{"Functions": "zvf", "Assets": "zva"}[coll]
	if _, ok := f.objs[key("zs", testAccount, svc)]; !ok {
		ferr(w, 404, "The requested resource "+r.URL.Path+" was not found")
		return
	}
	if _, ok := f.objs[key(ckind, testAccount, svc+"/"+parent)]; !ok {
		ferr(w, 404, "The requested resource "+r.URL.Path+" was not found")
		return
	}
	mt, params, err := mime.ParseMediaType(r.Header.Get("Content-Type"))
	if err != nil || mt != "multipart/form-data" {
		ferr(w, 400, "expected multipart/form-data")
		return
	}
	up := fakeUpload{}
	mr := multipart.NewReader(bytes.NewReader(body), params["boundary"])
	for {
		part, err := mr.NextPart()
		if err != nil {
			break
		}
		data, _ := io.ReadAll(part)
		switch part.FormName() {
		case "Content":
			up.Filename, up.ContentType, up.Content = part.FileName(), part.Header.Get("Content-Type"), data
		case "Path":
			up.Path = string(data)
		case "Visibility":
			up.Visibility = string(data)
		}
	}
	if up.Path == "" || up.Visibility == "" || up.Content == nil {
		ferr(w, 400, "Missing required parameter Content, Path or Visibility")
		return
	}
	id := f.sid("ZN")
	if f.uploads == nil {
		f.uploads = map[string]fakeUpload{}
	}
	f.uploads[id] = up
	o := map[string]any{"sid": id, "service_sid": svc, "account_sid": testAccount, "path": up.Path, "visibility": up.Visibility,
		"url": "https://serverless.twilio.com/v1/Services/" + svc + "/" + coll + "/" + parent + "/Versions/" + id, "date_created": serverlessStamp}
	o[map[string]string{"Functions": "function_sid", "Assets": "asset_sid"}[coll]] = parent
	f.put(vkind, testAccount, svc+"/"+parent+"/"+id, o)
	fwrite(w, 201, o)
}

func (f *fakeTwilio) serverlessVersion(w http.ResponseWriter, r *http.Request, svc string, parts []string, notFound func()) {
	vkind := map[string]string{"Functions": "zvf", "Assets": "zva"}[parts[1]]
	o, ok := f.objs[key(vkind, testAccount, svc+"/"+parts[2]+"/"+parts[4])]
	if !ok || r.Method != "GET" || (len(parts) == 6 && (parts[5] != "Content" || parts[1] != "Functions")) {
		notFound()
		return
	}
	if len(parts) == 6 {
		fwrite(w, 200, map[string]any{"sid": parts[4], "content": string(f.uploads[parts[4]].Content)})
		return
	}
	fwrite(w, 200, o)
}

func (f *fakeTwilio) serverlessBuild(w http.ResponseWriter, r *http.Request, form url.Values, svc string, parts []string, notFound func()) {
	if len(parts) == 2 {
		if r.Method != "POST" {
			ferr(w, 405, "method not allowed")
			return
		}
		id := f.sid("ZB")
		deps := []any{}
		if raw := form.Get("Dependencies"); raw != "" {
			if err := json.Unmarshal([]byte(raw), &deps); err != nil {
				ferr(w, 400, "Dependencies must be a JSON array")
				return
			}
		}
		fv, av := []any{}, []any{}
		for _, v := range form["FunctionVersions"] {
			fv = append(fv, map[string]any{"sid": v, "service_sid": svc, "path": "/f"}) // real shape: objects, not SID strings
		}
		for _, v := range form["AssetVersions"] {
			av = append(av, map[string]any{"sid": v, "service_sid": svc, "path": "/a"})
		}
		runtime := "node18"
		if v := form.Get("Runtime"); v != "" {
			runtime = v
		}
		o := map[string]any{"sid": id, "service_sid": svc, "account_sid": testAccount, "status": "building", "function_versions": fv, "asset_versions": av,
			"dependencies": deps, "runtime": runtime, "url": "https://serverless.twilio.com/v1/Services/" + svc + "/Builds/" + id,
			"date_created": serverlessStamp, "date_updated": serverlessStamp}
		f.put("zb", testAccount, svc+"/"+id, o)
		fwrite(w, 201, o)
		return
	}
	k := key("zb", testAccount, svc+"/"+parts[2])
	o, ok := f.objs[k]
	if !ok {
		notFound()
		return
	}
	if len(parts) == 4 {
		if parts[3] != "Status" || r.Method != "GET" {
			notFound()
			return
		}
		if f.buildPolls == nil {
			f.buildPolls = map[string]int{}
		}
		f.buildPolls[parts[2]]++
		if f.buildPolls[parts[2]] > f.buildBuildingPolls {
			final := f.buildFinalStatus
			if final == "" {
				final = "completed"
			}
			o["status"] = final
		}
		fwrite(w, 200, map[string]any{"sid": parts[2], "service_sid": svc, "account_sid": testAccount, "status": o["status"]})
		return
	}
	switch r.Method {
	case "GET":
		if f.buildReadFails {
			ferr(w, 500, "build read failed")
			return
		}
		fwrite(w, 200, o)
	case "DELETE":
		if f.buildDeleteUnrelated {
			fwrite(w, 400, map[string]any{"code": 20001, "status": 400, "message": "Something else is wrong with this request."})
			return
		}
		// Twilio refuses to delete a Build a Deployment points at, and has no delete for deployments.
		for _, d := range f.objs {
			if d["service_sid"] == svc && d["build_sid"] == parts[2] && d["environment_sid"] != nil {
				fwrite(w, 400, map[string]any{"code": 20001, "status": 400, "message": "Cannot delete Build because it's part of an active Deployment."})
				return
			}
		}
		f.del("zb", testAccount, svc+"/"+parts[2])
		w.WriteHeader(204)
	default:
		ferr(w, 405, "method not allowed")
	}
}

func (f *fakeTwilio) serverlessDeployment(w http.ResponseWriter, r *http.Request, form url.Values, svc, env, sid string, notFound func()) {
	if sid == "" {
		if r.Method != "POST" {
			ferr(w, 405, "method not allowed")
			return
		}
		build := form.Get("BuildSid")
		if _, ok := f.objs[key("zb", testAccount, svc+"/"+build)]; !ok {
			ferr(w, 400, "Build "+build+" does not exist")
			return
		}
		id := f.sid("ZD")
		o := map[string]any{"sid": id, "service_sid": svc, "environment_sid": env, "account_sid": testAccount, "build_sid": build,
			"url":          "https://serverless.twilio.com/v1/Services/" + svc + "/Environments/" + env + "/Deployments/" + id,
			"date_created": serverlessStamp, "date_updated": serverlessStamp}
		f.put("zd", testAccount, svc+"/"+env+"/"+id, o)
		// The environment now serves the build.
		if eo, ok := f.objs[key("ze", testAccount, svc+"/"+env)]; ok {
			eo["build_sid"] = build
		}
		fwrite(w, 201, o)
		return
	}
	o, ok := f.objs[key("zd", testAccount, svc+"/"+env+"/"+sid)]
	if !ok || r.Method != "GET" {
		notFound()
		return
	}
	fwrite(w, 200, o)
}

// containerReferencedByBuild reports whether a Build of the service holds a version of the Function or Asset.
func (f *fakeTwilio) containerReferencedByBuild(svc, coll, sid string) bool {
	vkind, field := map[string]string{"Functions": "zvf", "Assets": "zva"}[coll], map[string]string{"Functions": "function_versions", "Assets": "asset_versions"}[coll]
	prefix := key(vkind, testAccount, svc+"/"+sid+"/")
	for k, v := range f.objs {
		if !strings.HasPrefix(k, prefix) {
			continue
		}
		for bk, b := range f.objs {
			if !strings.HasPrefix(bk, key("zb", testAccount, svc+"/")) {
				continue
			}
			list, _ := b[field].([]any)
			for _, e := range list {
				if e == v["sid"] {
					return true
				}
			}
		}
	}
	return false
}
