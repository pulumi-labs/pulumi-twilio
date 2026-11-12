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
	"context"
	"encoding/json"
	"fmt"
	"net/url"
	"reflect"
	"regexp"

	p "github.com/pulumi/pulumi-go-provider"
	"github.com/pulumi/pulumi-go-provider/infer"
)

// The resources in this file configure TaskRouter (taskrouter.twilio.com v1): a workspace and the
// activities, task channels, task queues, workflows and workers inside it. Tasks, reservations,
// events and statistics are runtime data and are not managed. The shapes follow twilio-oai
// spec/json/twilio_taskrouter_v1.json (spec version 1.0.0).
//
// Optional settings follow one rule: an input that is set is sent and kept in step with Twilio
// (drift shows in a preview); an input that is removed leaves the setting as it is on Twilio,
// because the spec states no empty value to reset to. An import adopts a returned value only when
// it differs from the default Twilio documents.

const trProduct = "taskrouter"

var (
	trWorkspaceSidRE = regexp.MustCompile(`^WS[0-9a-fA-F]{32}$`)
	trActivitySidRE  = regexp.MustCompile(`^WA[0-9a-fA-F]{32}$`)
)

// trPath is the path of a TaskRouter collection or instance: /v1/Workspaces[/ws[/coll[/sid]]].
func trPath(ws, coll, sid string) string {
	path := "/v1/Workspaces"
	if ws == "" {
		return path
	}
	path += "/" + url.PathEscape(ws)
	if coll == "" {
		return path
	}
	path += "/" + coll
	if sid != "" {
		path += "/" + url.PathEscape(sid)
	}
	return path
}

// trSplitID splits a <workspaceSid>/<sid> import ID.
func trSplitID(id string) (ws, sid string, err error) {
	parts, err := splitVerifyID(id, 2, "<workspaceSid>/<sid>")
	if err != nil {
		return "", "", err
	}
	return parts[0], parts[1], nil
}

// jsonSame reports whether a and b are the same JSON document, ignoring formatting and key order.
func jsonSame(a, b string) bool {
	if a == b {
		return true
	}
	var x, y any
	if json.Unmarshal([]byte(a), &x) != nil || json.Unmarshal([]byte(b), &y) != nil {
		return false
	}
	return reflect.DeepEqual(x, y)
}

// trDiffSet collects the per-property diff of a TaskRouter resource.
type trDiffSet map[string]p.PropertyDiff

func (d trDiffSet) replace(name, o, n string) {
	if o != n {
		d[name] = p.PropertyDiff{Kind: p.UpdateReplace}
	}
}

func (d trDiffSet) requiredStr(name, o, n string) {
	if o != n {
		d[name] = p.PropertyDiff{Kind: p.Update}
	}
}

// str records an update of an optional string; removing the input is not a change.
func (d trDiffSet) str(name string, o, n *string) {
	if n != nil && (o == nil || *o != *n) {
		d[name] = p.PropertyDiff{Kind: p.Update}
	}
}

func (d trDiffSet) boolean(name string, o, n *bool) {
	if n != nil && (o == nil || *o != *n) {
		d[name] = p.PropertyDiff{Kind: p.Update}
	}
}

func (d trDiffSet) integer(name string, o, n *int) {
	if n != nil && (o == nil || *o != *n) {
		d[name] = p.PropertyDiff{Kind: p.Update}
	}
}

func (d trDiffSet) jsonDoc(name string, o, n *string) {
	if n != nil && (o == nil || !jsonSame(*o, *n)) {
		d[name] = p.PropertyDiff{Kind: p.Update}
	}
}

func (d trDiffSet) response() infer.DiffResponse {
	return infer.DiffResponse{HasChanges: len(d) > 0, DetailedDiff: d}
}

// trStr is the refreshed value of an optional string input: an input that is set follows Twilio, an
// unset one stays unset unless this is an import, which adopts a value that is not the default.
func trStr(in, api *string, def string, imported bool) *string {
	api = nonEmpty(api)
	switch {
	case imported:
		if api != nil && *api != def {
			return api
		}
		return nil
	case in == nil:
		return nil
	case api == nil:
		empty := ""
		return &empty
	}
	return api
}

func trBool(in, api *bool, def, imported bool) *bool {
	switch {
	case imported:
		if api != nil && *api != def {
			return api
		}
		return nil
	case in == nil:
		return nil
	case api == nil:
		return in
	}
	return api
}

func trInt(in, api *int, def int, imported bool) *int {
	switch {
	case imported:
		if api != nil && *api != def {
			return api
		}
		return nil
	case in == nil:
		return nil
	case api == nil:
		return in
	}
	return api
}

// trJSON is trStr for a JSON document: an input that means the same document as Twilio's is kept
// as written, so reformatting on Twilio's side is not drift.
func trJSON(in, api *string, imported bool) *string {
	if !imported && in != nil && api != nil && jsonSame(*in, *api) {
		return in
	}
	return trStr(in, api, "", imported)
}

// trSid returns the SID and timestamps every TaskRouter object shares.
func trSid(o obj) (sid, created, updated string) {
	return o.str("sid"), o.str("date_created"), o.str("date_updated")
}

func trCheckWorkspace(c *checker, ws string) {
	c.sid("workspaceSid", ws, trWorkspaceSidRE, "TaskRouter workspace (WS followed by 32 hex characters)")
}

func trCheckOptionalActivity(c *checker, prop string, v *string) {
	if v != nil && !c.unk(prop) && !trActivitySidRE.MatchString(*v) {
		c.fail(prop, prop+" must be the SID of a TaskRouter activity (WA followed by 32 hex characters)")
	}
}

// trDelete deletes a child of a workspace by its <workspaceSid>/<sid> ID; an object that is already gone is fine.
func trDelete(ctx context.Context, kind, coll, id string) error {
	ws, sid, err := trSplitID(id)
	if err != nil {
		return err
	}
	c, err := clientFor(ctx)
	if err != nil {
		return err
	}
	if _, err := c.do(ctx, "DELETE", trProduct, trPath(ws, coll, sid), nil, nil); err != nil && !isNotFound(err) {
		return fmt.Errorf("delete TaskRouter %s %s: %w", kind, id, err)
	}
	return nil
}
