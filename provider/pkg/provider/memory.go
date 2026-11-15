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
	"regexp"
	"sort"
	"strings"
	"unicode/utf8"

	p "github.com/pulumi/pulumi-go-provider"
	"github.com/pulumi/pulumi-go-provider/infer"
)

// Twilio Conversation Memory (memory.twilio.com, API version v1) is new and reaches only accounts that
// have it enabled in the Console. The shapes below follow twilio-oai spec/json/twilio_memory_v1.json
// (release 2.11.0); nothing here has been exercised against a live account yet. Profiles, observations,
// summaries and imports hold customer data and are not managed.

const (
	memoryStoresPath = "/v1/ControlPlane/Stores"
	memoryOpsPath    = "/v1/ControlPlane/Operations/"
)

func memoryStorePath(id string) string { return memoryStoresPath + "/" + url.PathEscape(id) }

func memoryGroupsPath(store string) string { return memoryStorePath(store) + "/TraitGroups" }

func memoryGroupPath(store, name string) string {
	return memoryGroupsPath(store) + "/" + url.PathEscape(name)
}

// MemoryStore manages a Twilio Conversation Memory store: the container for customer profiles and the trait groups that describe them.
type MemoryStore struct{}

// MemoryStoreArgs are the inputs of a MemoryStore.
type MemoryStoreArgs struct {
	DisplayName string  `pulumi:"displayName"`
	Description *string `pulumi:"description,optional"`
}

func (a *MemoryStoreArgs) Annotate(an infer.Annotator) {
	an.Describe(&a.DisplayName, "Unique, human-readable name of the store within the account: letters, digits and hyphens, at most 64 characters. Updates in place.")
	an.Describe(&a.Description, "Description of the store, 1 to 128 characters. Twilio cannot clear a description, so removing it replaces the store, which destroys the profiles and memories in it.")
}

// MemoryStoreState is the persisted state of a MemoryStore.
type MemoryStoreState struct {
	MemoryStoreArgs

	Status                string  `pulumi:"status"`
	Revision              int     `pulumi:"revision"`
	IntelligenceServiceId *string `pulumi:"intelligenceServiceId,optional"`
	DataAsOf              *string `pulumi:"dataAsOf,optional"`
}

func (s *MemoryStoreState) Annotate(a infer.Annotator) {
	a.Describe(&s.Status, "Provisioning status reported by Twilio: QUEUED, PROVISIONING, ACTIVE or FAILED. The provider waits for ACTIVE.")
	a.Describe(&s.Revision, "Version number Twilio reports for the store; it increases on every successful update.")
	a.Describe(&s.IntelligenceServiceId, "ID of the Conversational Intelligence service Twilio provisioned for memory extraction. It is created and removed with the store and is not managed separately.")
	a.Describe(&s.DataAsOf, "Time up to which data sent to the store has been processed downstream (ISO 8601).")
}

func (r *MemoryStore) Annotate(a infer.Annotator) {
	a.Describe(&r, "A Twilio Conversation Memory store (Memory API v1). Twilio creates, updates and deletes it asynchronously and provisions a Conversational Intelligence service with it; the provider follows the operation and waits until the store is ACTIVE (or gone). "+
		"An account can hold at most 15 stores. Deleting a store deletes the profiles and memories in it. "+
		"Experimental: the API is new, needs Conversation Memory enabled on the account, and has not been verified against a live account. The resource ID is the store ID (mem_store_...).")
}

var _ infer.CustomResource[MemoryStoreArgs, MemoryStoreState] = (*MemoryStore)(nil)
var _ infer.CustomCheck[MemoryStoreArgs] = (*MemoryStore)(nil)
var _ infer.CustomDiff[MemoryStoreArgs, MemoryStoreState] = (*MemoryStore)(nil)
var _ infer.CustomRead[MemoryStoreArgs, MemoryStoreState] = (*MemoryStore)(nil)
var _ infer.CustomUpdate[MemoryStoreArgs, MemoryStoreState] = (*MemoryStore)(nil)
var _ infer.CustomDelete[MemoryStoreState] = (*MemoryStore)(nil)

func (r *MemoryStore) Check(ctx context.Context, req infer.CheckRequest) (infer.CheckResponse[MemoryStoreArgs], error) {
	args, failures, err := infer.DefaultCheck[MemoryStoreArgs](ctx, req.NewInputs)
	if err != nil || len(failures) > 0 {
		return infer.CheckResponse[MemoryStoreArgs]{Inputs: args, Failures: failures}, err
	}
	unk := unknowns(req.NewInputs)
	fail := func(prop, reason string) { failures = append(failures, p.CheckFailure{Property: prop, Reason: reason}) }
	if !unk("displayName") && (!knowledgeBaseName.MatchString(args.DisplayName) || len(args.DisplayName) > 64) {
		fail("displayName", "displayName must be 1 to 64 letters, digits or hyphens")
	}
	if !unk("description") && args.Description != nil {
		if n := utf8.RuneCountInString(*args.Description); n < 1 || n > 128 {
			fail("description", "description must be 1 to 128 characters; omit it for none")
		}
	}
	return infer.CheckResponse[MemoryStoreArgs]{Inputs: args, Failures: failures}, nil
}

func (r *MemoryStore) Diff(_ context.Context, req infer.DiffRequest[MemoryStoreArgs, MemoryStoreState]) (infer.DiffResponse, error) {
	o, n := req.State.MemoryStoreArgs, req.Inputs
	detail := map[string]p.PropertyDiff{}
	if o.DisplayName != n.DisplayName {
		detail["displayName"] = p.PropertyDiff{Kind: p.Update}
	}
	if deref(o.Description) != deref(n.Description) {
		kind := p.Update
		if n.Description == nil {
			kind = p.UpdateReplace
		}
		detail["description"] = p.PropertyDiff{Kind: kind}
	}
	return infer.DiffResponse{HasChanges: len(detail) > 0, DetailedDiff: detail}, nil
}

func (s *MemoryStoreState) fill(o obj) {
	s.Status = o.str("status")
	if v := o.optInt("version"); v != nil {
		s.Revision = *v
	}
	s.IntelligenceServiceId = nonEmpty(o.optStr("intelligenceServiceId"))
	s.DataAsOf = nonEmpty(o.optStr("dataAsOf"))
}

func waitMemoryStoreActive(ctx context.Context, c *Client, id string) (obj, error) {
	var out obj
	err := pollUntil(ctx, "memory store "+id, func(ctx context.Context) (bool, error) {
		o, err := c.do(ctx, "GET", "memory", memoryStorePath(id), nil, nil)
		if err != nil {
			return false, err
		}
		switch s := o.str("status"); s {
		case "ACTIVE":
			out = o
			return true, nil
		case "FAILED":
			return false, fmt.Errorf("memory store %s is FAILED", id)
		}
		return false, nil
	})
	return out, err
}

func (r *MemoryStore) Create(ctx context.Context, req infer.CreateRequest[MemoryStoreArgs]) (infer.CreateResponse[MemoryStoreState], error) {
	a := req.Inputs
	if req.DryRun {
		return infer.CreateResponse[MemoryStoreState]{Output: MemoryStoreState{MemoryStoreArgs: a}}, nil
	}
	c, err := clientFor(ctx)
	if err != nil {
		return infer.CreateResponse[MemoryStoreState]{}, err
	}
	body := map[string]any{"displayName": a.DisplayName}
	if a.Description != nil {
		body["description"] = *a.Description
	}
	acc, err := c.doJSON(ctx, "POST", "memory", memoryStoresPath, nil, body)
	if err != nil {
		return infer.CreateResponse[MemoryStoreState]{}, fmt.Errorf("create memory store: %w", err)
	}
	op, err := waitOperation(ctx, c, "memory", memoryOpsPath, acc, "create memory store "+a.DisplayName)
	if err != nil {
		return infer.CreateResponse[MemoryStoreState]{}, err
	}
	id := op.str("result.id")
	if id == "" {
		return infer.CreateResponse[MemoryStoreState]{}, fmt.Errorf("create memory store %s: the completed operation carries no resource ID", a.DisplayName)
	}
	o, err := waitMemoryStoreActive(ctx, c, id)
	if err != nil {
		_, _ = c.do(context.WithoutCancel(ctx), "DELETE", "memory", memoryStorePath(id), nil, nil)
		return infer.CreateResponse[MemoryStoreState]{}, fmt.Errorf("create memory store %s: %w", a.DisplayName, err)
	}
	st := MemoryStoreState{MemoryStoreArgs: a}
	st.fill(o)
	return infer.CreateResponse[MemoryStoreState]{ID: id, Output: st}, nil
}

func (r *MemoryStore) Read(ctx context.Context, req infer.ReadRequest[MemoryStoreArgs, MemoryStoreState]) (infer.ReadResponse[MemoryStoreArgs, MemoryStoreState], error) {
	type resp = infer.ReadResponse[MemoryStoreArgs, MemoryStoreState]
	c, err := clientFor(ctx)
	if err != nil {
		return resp{}, err
	}
	o, err := c.do(ctx, "GET", "memory", memoryStorePath(req.ID), nil, nil)
	if err != nil {
		if isNotFound(err) {
			if err := notFoundOnImport("memory store", req.ID, req.Inputs, req.State); err != nil {
				return resp{}, err
			}
			return resp{}, nil
		}
		return resp{}, fmt.Errorf("read memory store %s: %w", req.ID, err)
	}
	in := req.Inputs
	in.DisplayName = o.str("displayName")
	in.Description = nonEmpty(o.optStr("description"))
	st := MemoryStoreState{MemoryStoreArgs: in}
	st.fill(o)
	return resp{ID: req.ID, Inputs: in, State: st}, nil
}

func (r *MemoryStore) Update(ctx context.Context, req infer.UpdateRequest[MemoryStoreArgs, MemoryStoreState]) (infer.UpdateResponse[MemoryStoreState], error) {
	st := req.State
	st.MemoryStoreArgs = req.Inputs
	if req.DryRun {
		return infer.UpdateResponse[MemoryStoreState]{Output: st}, nil
	}
	c, err := clientFor(ctx)
	if err != nil {
		return infer.UpdateResponse[MemoryStoreState]{}, err
	}
	body := map[string]any{}
	if req.Inputs.DisplayName != req.State.DisplayName {
		body["displayName"] = req.Inputs.DisplayName
	}
	if deref(req.Inputs.Description) != deref(req.State.Description) {
		body["description"] = deref(req.Inputs.Description)
	}
	if len(body) > 0 {
		acc, err := c.doJSON(ctx, "PATCH", "memory", memoryStorePath(req.ID), nil, body)
		if err != nil {
			return infer.UpdateResponse[MemoryStoreState]{}, fmt.Errorf("update memory store %s: %w", req.ID, err)
		}
		if acc.str("statusUrl") != "" {
			if _, err := waitOperation(ctx, c, "memory", memoryOpsPath, acc, "update memory store "+req.ID); err != nil {
				return infer.UpdateResponse[MemoryStoreState]{}, err
			}
		}
	}
	o, err := waitMemoryStoreActive(ctx, c, req.ID)
	if err != nil {
		return infer.UpdateResponse[MemoryStoreState]{}, fmt.Errorf("update memory store %s: %w", req.ID, err)
	}
	st.fill(o)
	return infer.UpdateResponse[MemoryStoreState]{Output: st}, nil
}

func (r *MemoryStore) Delete(ctx context.Context, req infer.DeleteRequest[MemoryStoreState]) (infer.DeleteResponse, error) {
	c, err := clientFor(ctx)
	if err != nil {
		return infer.DeleteResponse{}, err
	}
	acc, err := c.do(ctx, "DELETE", "memory", memoryStorePath(req.ID), nil, nil)
	if err != nil {
		if isNotFound(err) {
			return infer.DeleteResponse{}, nil
		}
		return infer.DeleteResponse{}, fmt.Errorf("delete memory store %s: %w", req.ID, err)
	}
	if acc.str("statusUrl") != "" {
		if _, err := waitOperation(ctx, c, "memory", memoryOpsPath, acc, "delete memory store "+req.ID); err != nil {
			return infer.DeleteResponse{}, err
		}
	}
	if err := waitGone(ctx, c, "memory", memoryStorePath(req.ID), "deleting memory store "+req.ID); err != nil {
		return infer.DeleteResponse{}, fmt.Errorf("delete memory store %s: %w", req.ID, err)
	}
	return infer.DeleteResponse{}, nil
}

// waitGone polls a path until Twilio answers 404, so a dependent resource or a re-create does not race the delete.
func waitGone(ctx context.Context, c *Client, product, path, what string) error {
	return pollUntil(ctx, what, func(ctx context.Context) (bool, error) {
		_, err := c.do(ctx, "GET", product, path, nil, nil)
		if isNotFound(err) {
			return true, nil
		}
		return false, err
	})
}

// MemoryTrait is the definition of one trait of a MemoryTraitGroup.
type MemoryTrait struct {
	DataType        string  `pulumi:"dataType"`
	Description     *string `pulumi:"description,optional"`
	IdTypePromotion *string `pulumi:"idTypePromotion,optional"`
	ValidationRule  *string `pulumi:"validationRule,optional"`
}

func (t *MemoryTrait) Annotate(a infer.Annotator) {
	a.Describe(&t.DataType, "Data type of the trait: STRING, NUMBER, BOOLEAN or ARRAY.")
	a.Describe(&t.Description, "What the trait holds, up to 1024 characters.")
	a.Describe(&t.IdTypePromotion, "Name of the identifier type (for example email or phone) that the trait value is promoted to for Identity Resolution, up to 30 characters. The identifier type must be configured in the store's Identity Resolution settings.")
	a.Describe(&t.ValidationRule, "Validation rule as a JSON object string; its ruleType is STRING, NUMBER, BOOLEAN or ARRAY and the other keys follow Twilio's validation rule schemas for that type. Formatting and key order do not matter.")
}

var memoryDataTypes = map[string]bool{"STRING": true, "NUMBER": true, "BOOLEAN": true, "ARRAY": true}

// normalizeJSON re-encodes a JSON object compactly with sorted keys; ok is false when s is not a JSON object.
func normalizeJSON(s string) (string, bool) {
	var m map[string]any
	if err := json.Unmarshal([]byte(s), &m); err != nil || m == nil {
		return "", false
	}
	b, _ := json.Marshal(m)
	return string(b), true
}

func (t MemoryTrait) equal(u MemoryTrait) bool {
	rule := func(s *string) string {
		if s == nil {
			return ""
		}
		if n, ok := normalizeJSON(*s); ok {
			return n
		}
		return *s
	}
	return t.DataType == u.DataType && deref(t.Description) == deref(u.Description) &&
		deref(t.IdTypePromotion) == deref(u.IdTypePromotion) && rule(t.ValidationRule) == rule(u.ValidationRule)
}

func (t MemoryTrait) body() map[string]any {
	m := map[string]any{"dataType": t.DataType}
	if t.Description != nil {
		m["description"] = *t.Description
	}
	if t.IdTypePromotion != nil {
		m["idTypePromotion"] = *t.IdTypePromotion
	}
	if t.ValidationRule != nil {
		var rule map[string]any
		if json.Unmarshal([]byte(*t.ValidationRule), &rule) == nil {
			m["validationRule"] = rule
		}
	}
	return m
}

func traitsEqual(a, b map[string]MemoryTrait) bool {
	if len(a) != len(b) {
		return false
	}
	for k, v := range a {
		w, ok := b[k]
		if !ok || !v.equal(w) {
			return false
		}
	}
	return true
}

// MemoryTraitGroup manages a trait group of a MemoryStore: a named collection of lasting facts about a customer, with declared types.
type MemoryTraitGroup struct{}

// MemoryTraitGroupArgs are the inputs of a MemoryTraitGroup.
type MemoryTraitGroupArgs struct {
	StoreId     string                 `pulumi:"storeId" provider:"replaceOnChanges"`
	DisplayName string                 `pulumi:"displayName" provider:"replaceOnChanges"`
	Description *string                `pulumi:"description,optional"`
	Traits      map[string]MemoryTrait `pulumi:"traits,optional"`
}

func (a *MemoryTraitGroupArgs) Annotate(an infer.Annotator) {
	an.Describe(&a.StoreId, "ID (mem_store_...) of the memory store the trait group belongs to. Changing it replaces the trait group.")
	an.Describe(&a.DisplayName, "Name of the trait group: letters, digits, underscores, hyphens and dots, at most 64 characters. Twilio addresses the group by this name, so changing it replaces the trait group.")
	an.Describe(&a.Description, "What the trait group holds, up to 1024 characters. Updates in place.")
	an.Describe(&a.Traits, "The traits of the group, keyed by trait name, at most 95. Adding, changing and removing a trait updates the group in place; Twilio removes a trait that is left out. Whether Twilio accepts changing the data type of an existing trait is unverified.")
}

// MemoryTraitGroupState is the persisted state of a MemoryTraitGroup.
type MemoryTraitGroupState struct {
	MemoryTraitGroupArgs

	Revision int `pulumi:"revision"`
}

func (s *MemoryTraitGroupState) Annotate(a infer.Annotator) {
	a.Describe(&s.Revision, "Version number Twilio reports for the trait group; it increases on every successful update.")
}

func (r *MemoryTraitGroup) Annotate(a infer.Annotator) {
	a.Describe(&r, "A trait group of a Twilio Conversation Memory store (Memory API v1): the declared traits (lasting facts about a customer) that Twilio extracts into profiles. "+
		"Create, update and delete are asynchronous on Twilio; the provider follows the operation and waits until the change is visible. Deleting a trait group deletes its traits from the store, and Twilio refuses it while something still depends on it. "+
		"Experimental: the API is new, needs Conversation Memory on the account, and has not been verified against a live account. Import with the ID <storeId>/<displayName>.")
}

var _ infer.CustomResource[MemoryTraitGroupArgs, MemoryTraitGroupState] = (*MemoryTraitGroup)(nil)
var _ infer.CustomCheck[MemoryTraitGroupArgs] = (*MemoryTraitGroup)(nil)
var _ infer.CustomDiff[MemoryTraitGroupArgs, MemoryTraitGroupState] = (*MemoryTraitGroup)(nil)
var _ infer.CustomRead[MemoryTraitGroupArgs, MemoryTraitGroupState] = (*MemoryTraitGroup)(nil)
var _ infer.CustomUpdate[MemoryTraitGroupArgs, MemoryTraitGroupState] = (*MemoryTraitGroup)(nil)
var _ infer.CustomDelete[MemoryTraitGroupState] = (*MemoryTraitGroup)(nil)

var memoryGroupName = regexp.MustCompile(`^[a-zA-Z0-9_.\-]+$`)

func (r *MemoryTraitGroup) Check(ctx context.Context, req infer.CheckRequest) (infer.CheckResponse[MemoryTraitGroupArgs], error) {
	args, failures, err := infer.DefaultCheck[MemoryTraitGroupArgs](ctx, req.NewInputs)
	if err != nil || len(failures) > 0 {
		return infer.CheckResponse[MemoryTraitGroupArgs]{Inputs: args, Failures: failures}, err
	}
	unk := unknowns(req.NewInputs)
	fail := func(prop, reason string) { failures = append(failures, p.CheckFailure{Property: prop, Reason: reason}) }
	if !unk("displayName") && (!memoryGroupName.MatchString(args.DisplayName) || len(args.DisplayName) > 64) {
		fail("displayName", "displayName must be 1 to 64 letters, digits, underscores, hyphens or dots")
	}
	if !unk("description") && args.Description != nil && utf8.RuneCountInString(*args.Description) > 1024 {
		fail("description", "description must be at most 1024 characters")
	}
	if !unk("traits") {
		if len(args.Traits) > 95 {
			fail("traits", "a trait group holds at most 95 traits")
		}
		names := make([]string, 0, len(args.Traits))
		for n := range args.Traits {
			names = append(names, n)
		}
		sort.Strings(names)
		for _, n := range names {
			t := args.Traits[n]
			prop := "traits[" + n + "]"
			switch {
			case strings.TrimSpace(n) == "":
				fail("traits", "a trait name must not be empty")
			case !memoryDataTypes[t.DataType]:
				fail(prop+".dataType", "dataType must be one of STRING, NUMBER, BOOLEAN or ARRAY")
			}
			if t.Description != nil && utf8.RuneCountInString(*t.Description) > 1024 {
				fail(prop+".description", "description must be at most 1024 characters")
			}
			if t.IdTypePromotion != nil && utf8.RuneCountInString(*t.IdTypePromotion) > 30 {
				fail(prop+".idTypePromotion", "idTypePromotion must be at most 30 characters")
			}
			if t.ValidationRule != nil {
				if _, ok := normalizeJSON(*t.ValidationRule); !ok {
					fail(prop+".validationRule", "validationRule must be a JSON object")
				}
			}
		}
	}
	return infer.CheckResponse[MemoryTraitGroupArgs]{Inputs: args, Failures: failures}, nil
}

func (r *MemoryTraitGroup) Diff(_ context.Context, req infer.DiffRequest[MemoryTraitGroupArgs, MemoryTraitGroupState]) (infer.DiffResponse, error) {
	o, n := req.State.MemoryTraitGroupArgs, req.Inputs
	detail := map[string]p.PropertyDiff{}
	if o.StoreId != n.StoreId {
		detail["storeId"] = p.PropertyDiff{Kind: p.UpdateReplace}
	}
	if o.DisplayName != n.DisplayName {
		detail["displayName"] = p.PropertyDiff{Kind: p.UpdateReplace}
	}
	if deref(o.Description) != deref(n.Description) {
		detail["description"] = p.PropertyDiff{Kind: p.Update}
	}
	if !traitsEqual(o.Traits, n.Traits) {
		detail["traits"] = p.PropertyDiff{Kind: p.Update}
	}
	return infer.DiffResponse{HasChanges: len(detail) > 0, DetailedDiff: detail}, nil
}

func splitMemoryGroupID(id string) (store, name string, err error) {
	store, name, ok := strings.Cut(id, "/")
	if !ok || store == "" || name == "" || strings.Contains(name, "/") {
		return "", "", fmt.Errorf("invalid trait group ID %q: expected <storeId>/<displayName>", id)
	}
	return store, name, nil
}

// memoryGroupOf returns the trait group object of a GET response, which wraps it as {"traitGroup": {...}}.
func memoryGroupOf(o obj) obj {
	if m, ok := o["traitGroup"].(map[string]any); ok {
		return obj(m)
	}
	return o
}

func memoryTraitsOf(g obj) map[string]MemoryTrait {
	raw, _ := g["traits"].(map[string]any)
	if len(raw) == 0 {
		return nil
	}
	out := make(map[string]MemoryTrait, len(raw))
	for name, v := range raw {
		m, _ := v.(map[string]any)
		t := obj(m)
		mt := MemoryTrait{DataType: t.str("dataType"), Description: nonEmpty(t.optStr("description")), IdTypePromotion: nonEmpty(t.optStr("idTypePromotion"))}
		if rule, ok := t["validationRule"].(map[string]any); ok && len(rule) > 0 {
			b, _ := json.Marshal(rule)
			s := string(b)
			mt.ValidationRule = &s
		}
		out[name] = mt
	}
	return out
}

// waitGroupVisible polls until the trait group can be read and returns it.
func waitGroupVisible(ctx context.Context, c *Client, store, name string) (obj, error) {
	var out obj
	err := pollUntil(ctx, "trait group "+store+"/"+name, func(ctx context.Context) (bool, error) {
		o, err := c.do(ctx, "GET", "memory", memoryGroupPath(store, name), nil, nil)
		if isNotFound(err) {
			return false, nil
		}
		if err != nil {
			return false, err
		}
		out = memoryGroupOf(o)
		return true, nil
	})
	return out, err
}

func (r *MemoryTraitGroup) Create(ctx context.Context, req infer.CreateRequest[MemoryTraitGroupArgs]) (infer.CreateResponse[MemoryTraitGroupState], error) {
	a := req.Inputs
	if req.DryRun {
		return infer.CreateResponse[MemoryTraitGroupState]{Output: MemoryTraitGroupState{MemoryTraitGroupArgs: a}}, nil
	}
	c, err := clientFor(ctx)
	if err != nil {
		return infer.CreateResponse[MemoryTraitGroupState]{}, err
	}
	body := map[string]any{"displayName": a.DisplayName}
	if a.Description != nil {
		body["description"] = *a.Description
	}
	if len(a.Traits) > 0 {
		traits := make(map[string]any, len(a.Traits))
		for n, t := range a.Traits {
			traits[n] = t.body()
		}
		body["traits"] = traits
	}
	acc, err := c.doJSON(ctx, "POST", "memory", memoryGroupsPath(a.StoreId), nil, body)
	if err != nil {
		return infer.CreateResponse[MemoryTraitGroupState]{}, fmt.Errorf("create trait group %s: %w", a.DisplayName, err)
	}
	what := "create trait group " + a.DisplayName
	if _, err := waitOperation(ctx, c, "memory", memoryOpsPath, acc, what); err != nil {
		return infer.CreateResponse[MemoryTraitGroupState]{}, err
	}
	g, err := waitGroupVisible(ctx, c, a.StoreId, a.DisplayName)
	if err != nil {
		// Do not leave a trait group behind that no state tracks.
		_, _ = c.do(context.WithoutCancel(ctx), "DELETE", "memory", memoryGroupPath(a.StoreId, a.DisplayName), nil, nil)
		return infer.CreateResponse[MemoryTraitGroupState]{}, fmt.Errorf("%s: %w", what, err)
	}
	st := MemoryTraitGroupState{MemoryTraitGroupArgs: a}
	if v := g.optInt("version"); v != nil {
		st.Revision = *v
	}
	return infer.CreateResponse[MemoryTraitGroupState]{ID: a.StoreId + "/" + a.DisplayName, Output: st}, nil
}

func (r *MemoryTraitGroup) Read(ctx context.Context, req infer.ReadRequest[MemoryTraitGroupArgs, MemoryTraitGroupState]) (infer.ReadResponse[MemoryTraitGroupArgs, MemoryTraitGroupState], error) {
	type resp = infer.ReadResponse[MemoryTraitGroupArgs, MemoryTraitGroupState]
	store, name, err := splitMemoryGroupID(req.ID)
	if err != nil {
		return resp{}, err
	}
	c, err := clientFor(ctx)
	if err != nil {
		return resp{}, err
	}
	o, err := c.do(ctx, "GET", "memory", memoryGroupPath(store, name), nil, nil)
	if err != nil {
		if isNotFound(err) {
			if err := notFoundOnImport("trait group", req.ID, req.Inputs, req.State); err != nil {
				return resp{}, err
			}
			return resp{}, nil
		}
		return resp{}, fmt.Errorf("read trait group %s: %w", req.ID, err)
	}
	g := memoryGroupOf(o)
	in := req.Inputs
	in.StoreId, in.DisplayName = store, name
	in.Description = nonEmpty(g.optStr("description"))
	remote := memoryTraitsOf(g)
	// A rule that means the same keeps the program's spelling, so a refresh plans no change.
	for n, t := range remote {
		if old, ok := in.Traits[n]; ok && old.equal(t) {
			remote[n] = old
		}
	}
	in.Traits = remote
	st := MemoryTraitGroupState{MemoryTraitGroupArgs: in}
	if v := g.optInt("version"); v != nil {
		st.Revision = *v
	}
	return resp{ID: req.ID, Inputs: in, State: st}, nil
}

func (r *MemoryTraitGroup) Update(ctx context.Context, req infer.UpdateRequest[MemoryTraitGroupArgs, MemoryTraitGroupState]) (infer.UpdateResponse[MemoryTraitGroupState], error) {
	st := req.State
	st.MemoryTraitGroupArgs = req.Inputs
	if req.DryRun {
		return infer.UpdateResponse[MemoryTraitGroupState]{Output: st}, nil
	}
	store, name, err := splitMemoryGroupID(req.ID)
	if err != nil {
		return infer.UpdateResponse[MemoryTraitGroupState]{}, err
	}
	c, err := clientFor(ctx)
	if err != nil {
		return infer.UpdateResponse[MemoryTraitGroupState]{}, err
	}
	body := map[string]any{}
	if deref(req.Inputs.Description) != deref(req.State.Description) {
		body["description"] = deref(req.Inputs.Description)
	}
	traits := map[string]any{}
	for n, t := range req.Inputs.Traits {
		if old, ok := req.State.Traits[n]; !ok || !old.equal(t) {
			traits[n] = t.body()
		}
	}
	for n := range req.State.Traits {
		if _, ok := req.Inputs.Traits[n]; !ok {
			traits[n] = map[string]any{"dataType": ""} // Twilio's explicit deletion marker
		}
	}
	if len(traits) > 0 {
		body["traits"] = traits
	}
	if len(body) > 0 {
		acc, err := c.doJSON(ctx, "PATCH", "memory", memoryGroupPath(store, name), nil, body)
		if err != nil {
			return infer.UpdateResponse[MemoryTraitGroupState]{}, fmt.Errorf("update trait group %s: %w", req.ID, err)
		}
		if _, err := waitOperation(ctx, c, "memory", memoryOpsPath, acc, "update trait group "+req.ID); err != nil {
			return infer.UpdateResponse[MemoryTraitGroupState]{}, err
		}
	}
	g, err := waitGroupVisible(ctx, c, store, name)
	if err != nil {
		return infer.UpdateResponse[MemoryTraitGroupState]{}, fmt.Errorf("update trait group %s: %w", req.ID, err)
	}
	if v := g.optInt("version"); v != nil {
		st.Revision = *v
	}
	return infer.UpdateResponse[MemoryTraitGroupState]{Output: st}, nil
}

func (r *MemoryTraitGroup) Delete(ctx context.Context, req infer.DeleteRequest[MemoryTraitGroupState]) (infer.DeleteResponse, error) {
	store, name, err := splitMemoryGroupID(req.ID)
	if err != nil {
		return infer.DeleteResponse{}, err
	}
	c, err := clientFor(ctx)
	if err != nil {
		return infer.DeleteResponse{}, err
	}
	acc, err := c.do(ctx, "DELETE", "memory", memoryGroupPath(store, name), nil, nil)
	if err != nil {
		if isNotFound(err) {
			return infer.DeleteResponse{}, nil
		}
		return infer.DeleteResponse{}, fmt.Errorf("delete trait group %s: %w", req.ID, err)
	}
	if acc.str("statusUrl") != "" {
		if _, err := waitOperation(ctx, c, "memory", memoryOpsPath, acc, "delete trait group "+req.ID); err != nil {
			return infer.DeleteResponse{}, err
		}
	}
	if err := waitGone(ctx, c, "memory", memoryGroupPath(store, name), "deleting trait group "+req.ID); err != nil {
		return infer.DeleteResponse{}, fmt.Errorf("delete trait group %s: %w", req.ID, err)
	}
	return infer.DeleteResponse{}, nil
}
