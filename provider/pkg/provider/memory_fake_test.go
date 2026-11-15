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
	"net/http"
	"strings"
)

// The Memory v1 API (memory.twilio.com), served for the Conversation Memory tests:
//
//	/memory/v1/ControlPlane/Stores[/{id}]
//	/memory/v1/ControlPlane/Stores/{id}/TraitGroups[/{name}]
//	/memory/v1/ControlPlane/Operations/{id}
//
// The shapes follow twilio-oai spec/json/twilio_memory_v1.json. Every write answers 202 with a
// statusUrl. An operation reports PENDING on its first poll and COMPLETED on the second, and only then
// does its effect become visible: a created trait group answers 404 until then, a deleted one keeps
// answering 200, and a PATCH is applied when the operation completes. GET of a trait group wraps the
// group as {"traitGroup": {...}}.

// memoryFailOps, when set, ends every Memory operation created from then on FAILED with this detail.
var memoryFailOps string

// memoryHideGroups, when set, makes a created trait group never become readable although its operation completes.
var memoryHideGroups bool

func (f *fakeTwilio) handleMemory(w http.ResponseWriter, r *http.Request, body []byte, rest string) {
	parts := strings.Split(strings.Trim(rest, "/"), "/")
	notFound := func() { ferr(w, 404, "The requested resource "+r.URL.Path+" was not found") }
	var in map[string]any
	if len(body) > 0 {
		if err := json.Unmarshal(body, &in); err != nil {
			ferr(w, 400, "body is not JSON")
			return
		}
	}
	if len(parts) < 2 || parts[0] != "ControlPlane" {
		notFound()
		return
	}
	switch {
	case parts[1] == "Stores" && len(parts) == 2 && r.Method == "POST":
		f.memoryCreateStore(w, in)
	case parts[1] == "Stores" && len(parts) == 3:
		f.memoryStore(w, r, in, parts[2], notFound)
	case parts[1] == "Stores" && len(parts) == 4 && parts[3] == "TraitGroups" && r.Method == "POST":
		f.memoryCreateGroup(w, in, parts[2], notFound)
	case parts[1] == "Stores" && len(parts) == 5 && parts[3] == "TraitGroups":
		f.memoryGroup(w, r, in, parts[2], parts[4], notFound)
	case parts[1] == "Stores" && len(parts) == 4 && parts[3] == "DataMappings" && r.Method == "POST":
		f.memoryCreateMapping(w, in, parts[2], notFound)
	case parts[1] == "Stores" && len(parts) == 5 && parts[3] == "DataMappings":
		f.memoryMapping(w, r, in, parts[2], parts[4], notFound)
	case parts[1] == "Stores" && len(parts) == 4 && parts[3] == "IdentityResolutionSettings":
		f.memoryIdentity(w, r, in, parts[2], notFound)
	case parts[1] == "Operations" && len(parts) == 3 && r.Method == "GET":
		f.memoryOperation(w, parts[2], notFound)
	default:
		notFound()
	}
}

// memoryOp registers a pending operation; apply runs when it completes.
func (f *fakeTwilio) memoryOp(resource string, apply func()) map[string]any {
	id := "op-" + f.sid("m0")
	o := map[string]any{"operationId": id, "resource": resource, "polls": 0.0}
	if memoryFailOps != "" {
		o["failWith"] = memoryFailOps
	}
	f.put("mop", testAccount, id, o)
	f.memoryApply[id] = apply
	return map[string]any{"message": "accepted", "statusUrl": "https://memory.twilio.com/v1/ControlPlane/Operations/" + id}
}

func (f *fakeTwilio) memoryOperation(w http.ResponseWriter, id string, notFound func()) {
	op, ok := f.objs[key("mop", testAccount, id)]
	if !ok {
		notFound()
		return
	}
	if op["polls"].(float64) < 1 {
		op["polls"] = op["polls"].(float64) + 1
		fwrite(w, 200, map[string]any{"operationId": id, "status": "PENDING", "createdAt": knowledgeStamp})
		return
	}
	if msg, ok := op["failWith"].(string); ok {
		fwrite(w, 200, map[string]any{"operationId": id, "status": "FAILED", "createdAt": knowledgeStamp, "error": map[string]any{"title": "Operation failed", "detail": msg}})
		return
	}
	if apply := f.memoryApply[id]; apply != nil {
		apply()
		delete(f.memoryApply, id)
	}
	fwrite(w, 200, map[string]any{"operationId": id, "status": "COMPLETED", "createdAt": knowledgeStamp, "result": map[string]any{"type": "RESOURCE_ID", "id": op["resource"]}})
}

func (f *fakeTwilio) memoryCreateStore(w http.ResponseWriter, in map[string]any) {
	name, _ := in["displayName"].(string)
	if name == "" {
		ferr(w, 400, "displayName is required")
		return
	}
	id := "mem_store_" + f.sid("m1")
	o := map[string]any{"id": id, "displayName": name, "status": "PROVISIONING", "version": 1.0, "intelligenceServiceId": "intelligence_service_" + f.sid("m2"), "dataAsOf": knowledgeStamp}
	if d, ok := in["description"].(string); ok {
		o["description"] = d
	}
	f.put("mstore", testAccount, id, o)
	fwrite(w, 202, f.memoryOp(id, func() { o["status"] = "ACTIVE" }))
}

func (f *fakeTwilio) memoryStore(w http.ResponseWriter, r *http.Request, in map[string]any, id string, notFound func()) {
	o, ok := f.objs[key("mstore", testAccount, id)]
	if !ok {
		notFound()
		return
	}
	switch r.Method {
	case "GET":
		fwrite(w, 200, clone(o))
	case "PATCH":
		if len(in) == 0 {
			ferr(w, 400, "at least one property is required")
			return
		}
		if v, ok := in["description"].(string); ok && v == "" {
			ferr(w, 400, "description must not be empty")
			return
		}
		o["status"] = "PROVISIONING"
		fwrite(w, 202, f.memoryOp(id, func() {
			for _, k := range []string{"displayName", "description"} {
				if v, ok := in[k]; ok {
					o[k] = v
				}
			}
			o["status"] = "ACTIVE"
			o["version"] = o["version"].(float64) + 1
		}))
	case "DELETE":
		fwrite(w, 202, f.memoryOp(id, func() { f.del("mstore", testAccount, id) }))
	default:
		ferr(w, 405, "method not allowed")
	}
}

func memoryGroupKey(store, name string) string { return store + "~" + name }

func (f *fakeTwilio) memoryCreateGroup(w http.ResponseWriter, in map[string]any, store string, notFound func()) {
	if _, ok := f.objs[key("mstore", testAccount, store)]; !ok {
		notFound()
		return
	}
	name, _ := in["displayName"].(string)
	if name == "" {
		ferr(w, 400, "displayName is required")
		return
	}
	if _, dup := f.objs[key("mgroup", testAccount, memoryGroupKey(store, name))]; dup {
		ferr(w, 409, "trait group already exists")
		return
	}
	traits, _ := in["traits"].(map[string]any)
	for n, t := range traits {
		if m, _ := t.(map[string]any); m["dataType"] == "" || m["dataType"] == nil {
			ferr(w, 400, "trait "+n+" needs a dataType")
			return
		}
	}
	g := map[string]any{"displayName": name, "version": 1.0}
	if d, ok := in["description"]; ok {
		g["description"] = d
	}
	if traits != nil {
		g["traits"] = traits
	}
	k := memoryGroupKey(store, name)
	fwrite(w, 202, f.memoryOp(name, func() {
		if !memoryHideGroups {
			f.put("mgroup", testAccount, k, g)
		}
	}))
}

func (f *fakeTwilio) memoryGroup(w http.ResponseWriter, r *http.Request, in map[string]any, store, name string, notFound func()) {
	k := memoryGroupKey(store, name)
	g, ok := f.objs[key("mgroup", testAccount, k)]
	if !ok {
		notFound()
		return
	}
	switch r.Method {
	case "GET":
		fwrite(w, 200, map[string]any{"traitGroup": clone(g), "meta": map[string]any{}})
	case "PATCH":
		if len(in) == 0 {
			ferr(w, 400, "at least one property is required")
			return
		}
		fwrite(w, 202, f.memoryOp(name, func() {
			if v, ok := in["description"]; ok {
				g["description"] = v
			}
			if patch, ok := in["traits"].(map[string]any); ok {
				cur, _ := g["traits"].(map[string]any)
				if cur == nil {
					cur = map[string]any{}
				}
				for n, t := range patch {
					if m, _ := t.(map[string]any); m["dataType"] == "" {
						delete(cur, n)
					} else {
						cur[n] = t
					}
				}
				g["traits"] = cur
			}
			g["version"] = g["version"].(float64) + 1
		}))
	case "DELETE":
		if g["locked"] == true {
			ferr(w, 409, "trait group is in use")
			return
		}
		fwrite(w, 202, f.memoryOp(name, func() { f.del("mgroup", testAccount, k) }))
	default:
		ferr(w, 405, "method not allowed")
	}
}
