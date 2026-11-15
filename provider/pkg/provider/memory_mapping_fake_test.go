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
	"regexp"
)

// Data mappings and identity resolution settings of the Memory v1 fake. Like trait groups, a write answers 202
// and takes effect when its operation completes: a created data mapping answers 404 until then and a deleted
// one keeps answering 200. Identity resolution settings answer 404 until the first PUT completes and report
// every setting with Twilio's defaults filled in.

var fakeMappingNameRe = regexp.MustCompile(`^[a-zA-Z][a-zA-Z0-9-.]*$`)

func (f *fakeTwilio) memoryCreateMapping(w http.ResponseWriter, in map[string]any, store string, notFound func()) {
	if _, ok := f.objs[key("mstore", testAccount, store)]; !ok {
		notFound()
		return
	}
	name, _ := in["displayName"].(string)
	from, _ := in["mappingFrom"].(map[string]any)
	to, _ := in["mappingTo"].(map[string]any)
	if !fakeMappingNameRe.MatchString(name) || from == nil || to == nil {
		ferr(w, 400, "displayName, mappingFrom and mappingTo are required")
		return
	}
	if ms, _ := to["mappings"].([]any); len(ms) == 0 || to["type"] != "TRAITS" {
		ferr(w, 400, "mappingTo needs type TRAITS and at least one mapping")
		return
	}
	switch from["type"] {
	case "INGRESS":
		if cols, _ := from["columns"].([]any); len(cols) == 0 {
			ferr(w, 400, "INGRESS needs columns")
			return
		}
	case "DATASET_CLOUDAPP", "DATASET_WAREHOUSE":
		if id, _ := from["datasetId"].(string); id == "" {
			ferr(w, 400, "dataset sources need a datasetId")
			return
		}
	default:
		ferr(w, 400, "unknown mappingFrom type")
		return
	}
	for _, o := range f.objs {
		if o["displayName"] == name && o["store"] == store && o["mappingFrom"] != nil {
			ferr(w, 409, "a data mapping with this name exists")
			return
		}
	}
	id := "mem_datamapping_" + f.sid("m3")
	m := map[string]any{"id": id, "store": store, "displayName": name, "isEnabled": true, "mappingFrom": from, "mappingTo": to,
		"version": 1.0, "createdAt": knowledgeStamp, "updatedAt": knowledgeStamp}
	if d, ok := in["description"]; ok {
		m["description"] = d
	}
	if v, ok := in["isEnabled"]; ok {
		m["isEnabled"] = v
	}
	fwrite(w, 202, f.memoryOp(id, func() { f.put("mmap", testAccount, id, m) }))
}

func (f *fakeTwilio) memoryMapping(w http.ResponseWriter, r *http.Request, in map[string]any, store, id string, notFound func()) {
	m, ok := f.objs[key("mmap", testAccount, id)]
	if !ok || m["store"] != store {
		notFound()
		return
	}
	switch r.Method {
	case "GET":
		out := clone(m)
		delete(out, "store")
		fwrite(w, 200, out)
	case "PATCH":
		if len(in) == 0 {
			ferr(w, 400, "at least one property is required")
			return
		}
		if _, bad := in["mappingFrom"]; bad {
			ferr(w, 400, "mappingFrom cannot be changed")
			return
		}
		if d, ok := in["description"].(string); ok && d == "" {
			ferr(w, 400, "description must not be empty")
			return
		}
		fwrite(w, 202, f.memoryOp(id, func() {
			for _, k := range []string{"displayName", "description", "isEnabled", "mappingTo"} {
				if v, ok := in[k]; ok {
					m[k] = v
				}
			}
			m["version"] = m["version"].(float64) + 1
			m["updatedAt"] = "2026-11-11T00:00:00Z"
		}))
	case "DELETE":
		fwrite(w, 202, f.memoryOp(id, func() { f.del("mmap", testAccount, id) }))
	default:
		ferr(w, 405, "method not allowed")
	}
}

func fakeIdentifierDefaults(c map[string]any) map[string]any {
	out := clone(c)
	for k, v := range map[string]any{"matchingAlgo": "exact", "matchingThreshold": 75.0, "limit": 100.0, "limitPolicy": "fifo", "enforceUnique": true, "normalization": "trim"} {
		if _, ok := out[k]; !ok {
			out[k] = v
		}
	}
	return out
}

func (f *fakeTwilio) memoryIdentity(w http.ResponseWriter, r *http.Request, in map[string]any, store string, notFound func()) {
	if _, ok := f.objs[key("mstore", testAccount, store)]; !ok {
		notFound()
		return
	}
	switch r.Method {
	case "GET":
		s, ok := f.objs[key("mident", testAccount, store)]
		if !ok {
			notFound()
			return
		}
		fwrite(w, 200, clone(s))
	case "PUT":
		cfgs, _ := in["identifierConfigs"].([]any)
		rules, _ := in["matchingRules"].([]any)
		if len(cfgs) == 0 || len(rules) == 0 {
			ferr(w, 400, "identifierConfigs and matchingRules are required")
			return
		}
		full := make([]any, 0, len(cfgs))
		for _, c := range cfgs {
			m, _ := c.(map[string]any)
			if id, _ := m["idType"].(string); len(id) < 2 {
				ferr(w, 400, "idType must be at least 2 characters")
				return
			}
			full = append(full, fakeIdentifierDefaults(m))
		}
		version := 1.0
		if cur, ok := f.objs[key("mident", testAccount, store)]; ok {
			version = cur["version"].(float64) + 1
		}
		fwrite(w, 202, f.memoryOp(store, func() {
			f.put("mident", testAccount, store, map[string]any{"identifierConfigs": full, "matchingRules": rules, "version": version})
		}))
	default:
		ferr(w, 405, "method not allowed")
	}
}
