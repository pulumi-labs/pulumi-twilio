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
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
)

var knowledgeFileMimes = map[string]string{"text/csv": "csv", "text/markdown": "md", "application/pdf": "pdf", "text/tab-separated-values": "tsv", "text/plain": "txt"}

// knowledgeUploadOmitURL, when set, makes the create answer leave importUrl out (the provider then reads the source).
var knowledgeUploadOmitURL bool

// knowledgeRejectUpload makes the presigned PUT answer 403, as an expired URL would.
var knowledgeRejectUpload bool

// handleKnowledgeUpload is the presigned PUT: it must carry the declared content type and size and no Twilio
// credentials, after which the source is queued for processing.
func (f *fakeTwilio) handleKnowledgeUpload(w http.ResponseWriter, r *http.Request, id string) {
	o, ok := f.objs[key("kitem", testAccount, id)]
	if !ok || r.Method != "PUT" {
		ferr(w, 404, "no such upload")
		return
	}
	if knowledgeRejectUpload {
		ferr(w, 403, "Request has expired")
		return
	}
	if r.Header.Get("Authorization") != "" {
		ferr(w, 400, "presigned uploads must not carry credentials")
		return
	}
	src := o["source"].(map[string]any)
	if r.Header.Get("Content-Type") != src["mimeType"] {
		ferr(w, 403, "SignatureDoesNotMatch: content type")
		return
	}
	data, _ := io.ReadAll(r.Body)
	if float64(len(data)) != src["fileSize"] {
		ferr(w, 400, "size does not match fileSize")
		return
	}
	sum := sha256.Sum256(data)
	o["uploadedSha256"] = hex.EncodeToString(sum[:])
	delete(src, "importUrl")
	delete(src, "uploadExpiration")
	o["status"] = "QUEUED"
	w.WriteHeader(200)
}

var _ = fmt.Sprint

// The Knowledge v2 API (knowledge.twilio.com), served for the Enterprise Knowledge tests:
//
//	/knowledge/v2/ControlPlane/KnowledgeBases[/{id}]    (create, update and delete answer 202 with a statusUrl)
//	/knowledge/v2/ControlPlane/Operations/{id}
//	/knowledge/v2/KnowledgeBases/{id}/Knowledge[/{kid}]
//
// The shapes follow twilio-oai spec/json/twilio_knowledge_v2.json. An operation reports PENDING on
// its first poll and COMPLETED on the second; a source reports QUEUED on create and COMPLETED on its
// next read, or FAILED when its URL or content contains "fail".

const knowledgeStamp = "2026-11-10T00:00:00Z"

// knowledgeFailOps, when set, makes every Knowledge Base operation created from then on end FAILED with this detail.
var knowledgeFailOps string

func (f *fakeTwilio) handleKnowledge(w http.ResponseWriter, r *http.Request, body []byte, rest string) {
	parts := strings.Split(strings.Trim(rest, "/"), "/")
	notFound := func() { ferr(w, 404, "The requested resource "+r.URL.Path+" was not found") }
	var in map[string]any
	if len(body) > 0 {
		if err := json.Unmarshal(body, &in); err != nil {
			ferr(w, 400, "body is not JSON")
			return
		}
	}
	switch {
	case len(parts) == 2 && parts[0] == "ControlPlane" && parts[1] == "KnowledgeBases" && r.Method == "POST":
		f.knowledgeCreateBase(w, in)
	case len(parts) == 3 && parts[0] == "ControlPlane" && parts[1] == "KnowledgeBases":
		f.knowledgeBase(w, r, in, parts[2], notFound)
	case len(parts) == 3 && parts[0] == "ControlPlane" && parts[1] == "Operations" && r.Method == "GET":
		f.knowledgeOperation(w, parts[2], notFound)
	case len(parts) == 3 && parts[0] == "KnowledgeBases" && parts[2] == "Knowledge" && r.Method == "POST":
		f.knowledgeCreateItem(w, in, parts[1], notFound)
	case len(parts) == 4 && parts[0] == "KnowledgeBases" && parts[2] == "Knowledge":
		f.knowledgeItem(w, r, in, parts[1], parts[3], notFound)
	default:
		notFound()
	}
}

func (f *fakeTwilio) knowledgeOp(kind, resource string, extra map[string]any) map[string]any {
	id := "op-" + f.sid("k0")
	o := map[string]any{"operationId": id, "kind": kind, "resource": resource, "polls": 0.0, "createdAt": knowledgeStamp}
	for k, v := range extra {
		o[k] = v
	}
	if knowledgeFailOps != "" {
		o["failWith"] = knowledgeFailOps
	}
	f.put("kop", testAccount, id, o)
	return map[string]any{"message": "accepted", "statusUrl": "https://knowledge.twilio.com/v2/ControlPlane/Operations/" + id}
}

func (f *fakeTwilio) knowledgeCreateBase(w http.ResponseWriter, in map[string]any) {
	name, _ := in["displayName"].(string)
	if name == "" {
		ferr(w, 400, "displayName is required")
		return
	}
	for _, o := range f.objs {
		if o["displayName"] == name && o["resource"] == nil && o["id"] != nil && strings.HasPrefix(o["id"].(string), "know_knowledgebase_") {
			ferr(w, 409, "displayName already exists")
			return
		}
	}
	id := "know_knowledgebase_" + f.sid("k0")
	o := map[string]any{"id": id, "displayName": name, "status": "PROVISIONING", "version": 1.0, "createdAt": knowledgeStamp, "updatedAt": knowledgeStamp}
	if d, ok := in["description"].(string); ok {
		o["description"] = d
	}
	f.put("kbase", testAccount, id, o)
	fwrite(w, 202, f.knowledgeOp("create", id, nil))
}

func (f *fakeTwilio) knowledgeBase(w http.ResponseWriter, r *http.Request, in map[string]any, id string, notFound func()) {
	o, ok := f.objs[key("kbase", testAccount, id)]
	if !ok {
		notFound()
		return
	}
	switch r.Method {
	case "GET":
		fwrite(w, 200, clone(o))
	case "PATCH":
		if v, ok := in["displayName"].(string); ok {
			o["displayName"] = v
		}
		if v, ok := in["description"].(string); ok {
			if v == "" {
				ferr(w, 400, "description must not be empty")
				return
			}
			o["description"] = v
		}
		o["status"] = "PROVISIONING"
		fwrite(w, 202, f.knowledgeOp("update", id, nil))
	case "DELETE":
		o["status"] = "DELETING"
		fwrite(w, 202, f.knowledgeOp("delete", id, nil))
	default:
		ferr(w, 405, "method not allowed")
	}
}

// knowledgeOperation advances an operation: PENDING first, then COMPLETED, applying its effect.
func (f *fakeTwilio) knowledgeOperation(w http.ResponseWriter, id string, notFound func()) {
	op, ok := f.objs[key("kop", testAccount, id)]
	if !ok {
		notFound()
		return
	}
	res, _ := op["resource"].(string)
	if op["polls"].(float64) < 1 {
		op["polls"] = op["polls"].(float64) + 1
		fwrite(w, 200, map[string]any{"operationId": id, "status": "PENDING", "createdAt": knowledgeStamp})
		return
	}
	if msg, ok := op["failWith"].(string); ok {
		fwrite(w, 200, map[string]any{"operationId": id, "status": "FAILED", "createdAt": knowledgeStamp, "error": map[string]any{"title": "Provisioning failed", "detail": msg}})
		return
	}
	if kb, ok := f.objs[key("kbase", testAccount, res)]; ok {
		if op["kind"] == "delete" {
			f.del("kbase", testAccount, res)
		} else {
			kb["status"] = "ACTIVE"
			kb["updatedAt"] = "2026-11-11T00:00:00Z"
			if op["kind"] == "update" {
				kb["version"] = kb["version"].(float64) + 1
			}
		}
	}
	fwrite(w, 200, map[string]any{"operationId": id, "status": "COMPLETED", "createdAt": knowledgeStamp, "result": map[string]any{"type": "RESOURCE_ID", "id": res}})
}

func (f *fakeTwilio) knowledgeCreateItem(w http.ResponseWriter, in map[string]any, kb string, notFound func()) {
	if _, ok := f.objs[key("kbase", testAccount, kb)]; !ok {
		notFound()
		return
	}
	name, _ := in["name"].(string)
	src, _ := in["source"].(map[string]any)
	if name == "" || len(name) > 30 || src == nil {
		ferr(w, 400, "name (max 30) and source are required")
		return
	}
	id := "know_knowledge_" + f.sid("k0")
	o := map[string]any{"id": id, "kb": kb, "name": name, "status": "QUEUED", "source": knowledgeSourceOut(src), "createdAt": knowledgeStamp, "updatedAt": knowledgeStamp}
	if src["type"] == "File" {
		// A File source waits (SCHEDULED) for its content to be PUT to the presigned importUrl.
		fn, _ := src["fileName"].(string)
		size, _ := src["fileSize"].(float64)
		if fn == "" || size < 1 || size > 16*1024*1024 || knowledgeFileMimes[fmt.Sprint(src["mimeType"])] == "" {
			ferr(w, 400, "fileName, fileSize (1 byte to 16 MiB) and a supported mimeType are required")
			return
		}
		o["status"] = "SCHEDULED"
		o["source"].(map[string]any)["importUrl"] = f.server.URL + "/knowledge-upload/" + id
		o["source"].(map[string]any)["uploadExpiration"] = "2026-11-10T01:00:00Z"
	}
	if d, ok := in["description"].(string); ok {
		o["description"] = d
	}
	f.put("kitem", testAccount, id, o)
	out := clone(o)
	if knowledgeUploadOmitURL && src["type"] == "File" {
		src2 := clone(out["source"].(map[string]any))
		delete(src2, "importUrl")
		out["source"] = src2
	}
	fwrite(w, 201, out)
}

// knowledgeSourceOut applies Twilio's defaults to a Web source and drops the text of a Text source.
func knowledgeSourceOut(src map[string]any) map[string]any {
	out := clone(src)
	if src["type"] == "Web" {
		if _, ok := out["crawlDepth"]; !ok {
			out["crawlDepth"] = 2.0
		}
		if _, ok := out["crawlPeriod"]; !ok {
			out["crawlPeriod"] = "NEVER"
		}
	}
	return out
}

func (f *fakeTwilio) knowledgeItem(w http.ResponseWriter, r *http.Request, in map[string]any, kb, id string, notFound func()) {
	o, ok := f.objs[key("kitem", testAccount, id)]
	if !ok || o["kb"] != kb {
		notFound()
		return
	}
	switch r.Method {
	case "GET":
		out := clone(o)
		if o["status"] == "QUEUED" {
			// The next read after a create or an update sees the finished (or failed) processing.
			if strings.Contains(strings.ToLower(jsonString(o["source"])), "fail") {
				o["status"] = "FAILED"
			} else {
				o["status"] = "COMPLETED"
			}
		}
		delete(out, "uploadedSha256")
		fwrite(w, 200, f.knowledgeShow(out))
	case "PATCH":
		if v, _ := in["name"].(string); v == "" {
			ferr(w, 400, "name is required")
			return
		} else {
			o["name"] = v
		}
		if v, ok := in["description"].(string); ok {
			o["description"] = v
		}
		if src, ok := in["source"].(map[string]any); ok {
			o["source"] = knowledgeSourceOut(src)
			o["status"] = "QUEUED"
		}
		o["updatedAt"] = "2026-11-11T00:00:00Z"
		fwrite(w, 200, clone(o))
	case "DELETE":
		f.del("kitem", testAccount, id)
		w.WriteHeader(204)
	default:
		ferr(w, 405, "method not allowed")
	}
}

// knowledgeShow hides the text of a Text source, which the list and get calls do not echo here.
func (f *fakeTwilio) knowledgeShow(o map[string]any) map[string]any {
	if src, ok := o["source"].(map[string]any); ok && src["type"] == "Text" {
		cp := clone(src)
		delete(cp, "content")
		o["source"] = cp
	}
	delete(o, "kb")
	return o
}

func jsonString(v any) string {
	b, _ := json.Marshal(v)
	return string(b)
}
