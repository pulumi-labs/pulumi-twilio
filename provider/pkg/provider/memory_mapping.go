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
	"fmt"
	"net/url"
	"regexp"
	"slices"
	"strings"
	"unicode/utf8"

	p "github.com/pulumi/pulumi-go-provider"
	"github.com/pulumi/pulumi-go-provider/infer"
)

// Data mappings and identity resolution settings of a Twilio Conversation Memory store (memory.twilio.com, v1).
// The shapes follow twilio-oai spec/json/twilio_memory_v1.json (release 2.11.0); nothing here has been
// exercised against a live account yet.

func memoryMappingsPath(store string) string { return memoryStorePath(store) + "/DataMappings" }

func memoryMappingPath(store, id string) string {
	return memoryMappingsPath(store) + "/" + url.PathEscape(id)
}

func memoryIdentityPath(store string) string {
	return memoryStorePath(store) + "/IdentityResolutionSettings"
}

// waitMemoryVisible polls a path until it can be read; a 404 means the operation's effect is not visible yet.
func waitMemoryVisible(ctx context.Context, c *Client, path, what string) (obj, error) {
	var out obj
	err := pollUntil(ctx, what, func(ctx context.Context) (bool, error) {
		o, err := c.do(ctx, "GET", "memory", path, nil, nil)
		if isNotFound(err) {
			return false, nil
		}
		if err != nil {
			return false, err
		}
		out = o
		return true, nil
	})
	return out, err
}

// waitMemoryAccepted follows the operation an accepted write returned, when it carries a statusUrl.
func waitMemoryAccepted(ctx context.Context, c *Client, acc obj, what string) (obj, error) {
	if acc.str("statusUrl") == "" {
		return nil, nil
	}
	return waitOperation(ctx, c, "memory", memoryOpsPath, acc, what)
}

// MemoryTraitMapping maps one source field to a trait of a trait group.
type MemoryTraitMapping struct {
	Expression string `pulumi:"expression"`
	TraitGroup string `pulumi:"traitGroup"`
	TraitName  string `pulumi:"traitName"`
}

func (m *MemoryTraitMapping) Annotate(a infer.Annotator) {
	a.Describe(&m.Expression, "The field or column in the source (an ingress column name, or a field of the dataset), 1 to 256 characters.")
	a.Describe(&m.TraitGroup, "Name of the trait group to write to (letters, digits, underscores, hyphens and dots, at most 64 characters).")
	a.Describe(&m.TraitName, "Name of the trait within the group: starts with a lowercase letter, then letters, digits, underscores, hyphens or dots, at most 128 characters.")
}

var (
	memoryMappingName  = regexp.MustCompile(`^[a-zA-Z][a-zA-Z0-9-.]*$`)
	memoryTraitGroupID = regexp.MustCompile(`^[a-zA-Z0-9_\-.]+$`)
	memoryTraitNameRe  = regexp.MustCompile(`^[a-z][a-zA-Z0-9_\-.]*$`)
	memoryDatasetID    = regexp.MustCompile(`^tdi_dataset_[0-7][0-9a-z]{25}$`)
	memoryFromTypes    = map[string]bool{"INGRESS": true, "DATASET_CLOUDAPP": true, "DATASET_WAREHOUSE": true}
)

func traitMappingsBody(ms []MemoryTraitMapping) map[string]any {
	items := make([]map[string]any, 0, len(ms))
	for _, m := range ms {
		items = append(items, map[string]any{"expression": m.Expression, "traitGroup": m.TraitGroup, "traitName": m.TraitName})
	}
	return map[string]any{"type": "TRAITS", "mappings": items}
}

func traitMappingsOf(o obj) []MemoryTraitMapping {
	raw, _ := o.get("mappingTo.mappings").([]any)
	out := make([]MemoryTraitMapping, 0, len(raw))
	for _, it := range raw {
		m, ok := it.(map[string]any)
		if !ok {
			continue
		}
		s := func(k string) string { v, _ := m[k].(string); return v }
		out = append(out, MemoryTraitMapping{Expression: s("expression"), TraitGroup: s("traitGroup"), TraitName: s("traitName")})
	}
	return out
}

// MemoryDataMapping connects a data source to the traits of a Conversation Memory store, so that data is ingested into profiles automatically.
type MemoryDataMapping struct{}

// MemoryDataMappingArgs are the inputs of a MemoryDataMapping.
type MemoryDataMappingArgs struct {
	StoreId     string               `pulumi:"storeId" provider:"replaceOnChanges"`
	DisplayName string               `pulumi:"displayName"`
	Description *string              `pulumi:"description,optional"`
	IsEnabled   *bool                `pulumi:"isEnabled,optional"`
	FromType    string               `pulumi:"fromType" provider:"replaceOnChanges"`
	Columns     []string             `pulumi:"columns,optional" provider:"replaceOnChanges"`
	DatasetId   *string              `pulumi:"datasetId,optional" provider:"replaceOnChanges"`
	Mappings    []MemoryTraitMapping `pulumi:"mappings"`
}

func (a *MemoryDataMappingArgs) Annotate(an infer.Annotator) {
	an.Describe(&a.StoreId, "ID (mem_store_...) of the memory store. Changing it replaces the data mapping.")
	an.Describe(&a.DisplayName, "Name of the data mapping: starts with a letter, then letters, digits, hyphens or dots, at most 64 characters. Updates in place.")
	an.Describe(&a.Description, "Description of the data mapping, 1 to 512 characters. Twilio cannot clear a description, so removing it replaces the data mapping.")
	an.Describe(&a.IsEnabled, "Whether data is ingested and mapped. Twilio's default is true. Updates in place.")
	an.Describe(&a.FromType, "Where the data comes from: INGRESS (columns sent in ingest requests), DATASET_CLOUDAPP (a dataset backed by a cloud app such as a CRM) or DATASET_WAREHOUSE (a dataset backed by a warehouse). Twilio's update call cannot change the source, so changing it replaces the data mapping.")
	an.Describe(&a.Columns, "INGRESS sources: the names of the ingress columns that serve as source fields, 1 to 100 names of 1 to 256 characters. Required for INGRESS. Changing it replaces the data mapping.")
	an.Describe(&a.DatasetId, "DATASET_CLOUDAPP and DATASET_WAREHOUSE sources: the ID (tdi_dataset_...) of the dataset to connect. Required for those types. Changing it replaces the data mapping.")
	an.Describe(&a.Mappings, "The source field to trait mappings, 1 to 100. Writing the list replaces the whole list on Twilio. For dataset sources Twilio validates that the trait groups and traits exist and that the data types match.")
}

// MemoryDataMappingState is the persisted state of a MemoryDataMapping.
type MemoryDataMappingState struct {
	MemoryDataMappingArgs

	DataMappingId string `pulumi:"dataMappingId"`
	Revision      int    `pulumi:"revision"`
	CreatedAt     string `pulumi:"createdAt"`
	UpdatedAt     string `pulumi:"updatedAt"`
}

func (s *MemoryDataMappingState) Annotate(a infer.Annotator) {
	a.Describe(&s.DataMappingId, "The data mapping ID (mem_datamapping_...). The resource ID is the store ID and this ID joined by a slash.")
	a.Describe(&s.Revision, "Version number Twilio reports for the data mapping; it increases on every successful update.")
	a.Describe(&s.CreatedAt, "When the data mapping was created (ISO 8601).")
	a.Describe(&s.UpdatedAt, "When the data mapping was last updated (ISO 8601).")
}

func (r *MemoryDataMapping) Annotate(a infer.Annotator) {
	a.Describe(&r, "A data mapping of a Twilio Conversation Memory store: it connects a source (ingress columns or a dataset) to the traits of the store's profiles. "+
		"Twilio creates, updates and deletes it asynchronously; the provider follows the operation. "+
		"Experimental: needs Conversation Memory on the account and has not been verified against a live account. Import with the ID <storeId>/<dataMappingId>.")
}

var _ infer.CustomResource[MemoryDataMappingArgs, MemoryDataMappingState] = (*MemoryDataMapping)(nil)
var _ infer.CustomCheck[MemoryDataMappingArgs] = (*MemoryDataMapping)(nil)
var _ infer.CustomDiff[MemoryDataMappingArgs, MemoryDataMappingState] = (*MemoryDataMapping)(nil)
var _ infer.CustomRead[MemoryDataMappingArgs, MemoryDataMappingState] = (*MemoryDataMapping)(nil)
var _ infer.CustomUpdate[MemoryDataMappingArgs, MemoryDataMappingState] = (*MemoryDataMapping)(nil)
var _ infer.CustomDelete[MemoryDataMappingState] = (*MemoryDataMapping)(nil)

func (r *MemoryDataMapping) Check(ctx context.Context, req infer.CheckRequest) (infer.CheckResponse[MemoryDataMappingArgs], error) {
	args, failures, err := infer.DefaultCheck[MemoryDataMappingArgs](ctx, req.NewInputs)
	if err != nil || len(failures) > 0 {
		return infer.CheckResponse[MemoryDataMappingArgs]{Inputs: args, Failures: failures}, err
	}
	unk := unknowns(req.NewInputs)
	fail := func(prop, reason string) { failures = append(failures, p.CheckFailure{Property: prop, Reason: reason}) }
	if !unk("displayName") && (!memoryMappingName.MatchString(args.DisplayName) || len(args.DisplayName) > 64) {
		fail("displayName", "displayName must start with a letter and hold at most 64 letters, digits, hyphens or dots")
	}
	if !unk("description") && args.Description != nil {
		if n := utf8.RuneCountInString(*args.Description); n < 1 || n > 512 {
			fail("description", "description must be 1 to 512 characters; omit it for none")
		}
	}
	if !unk("fromType") {
		switch {
		case !memoryFromTypes[args.FromType]:
			fail("fromType", "fromType must be INGRESS, DATASET_CLOUDAPP or DATASET_WAREHOUSE")
		case args.FromType == "INGRESS":
			if args.DatasetId != nil {
				fail("datasetId", "datasetId applies to dataset sources, not INGRESS")
			}
			if !unk("columns") {
				if len(args.Columns) < 1 || len(args.Columns) > 100 {
					fail("columns", "an INGRESS source needs 1 to 100 columns")
				}
				for i, c := range args.Columns {
					if unknownAt(req.NewInputs, "columns", i) {
						continue
					}
					if n := utf8.RuneCountInString(c); n < 1 || n > 256 {
						fail("columns", "each column name must be 1 to 256 characters")
						break
					}
				}
			}
		default:
			if args.Columns != nil {
				fail("columns", "columns apply to INGRESS sources, not "+args.FromType)
			}
			if !unk("datasetId") {
				switch {
				case args.DatasetId == nil:
					fail("datasetId", "datasetId is required for "+args.FromType+" sources")
				case !memoryDatasetID.MatchString(*args.DatasetId):
					fail("datasetId", "datasetId must look like tdi_dataset_ followed by 26 characters")
				}
			}
		}
	}
	if !unk("mappings") {
		if len(args.Mappings) < 1 || len(args.Mappings) > 100 {
			fail("mappings", "a data mapping needs 1 to 100 mappings")
		}
		for i, m := range args.Mappings {
			prop := fmt.Sprintf("mappings[%d]", i)
			if unknownAt(req.NewInputs, "mappings", i) {
				continue
			}
			if n := utf8.RuneCountInString(m.Expression); !unknownAt(req.NewInputs, "mappings", i, "expression") && (n < 1 || n > 256) {
				fail(prop+".expression", "expression must be 1 to 256 characters")
			}
			if !unknownAt(req.NewInputs, "mappings", i, "traitGroup") && (!memoryTraitGroupID.MatchString(m.TraitGroup) || len(m.TraitGroup) > 64) {
				fail(prop+".traitGroup", "traitGroup must be 1 to 64 letters, digits, underscores, hyphens or dots")
			}
			if !unknownAt(req.NewInputs, "mappings", i, "traitName") && (!memoryTraitNameRe.MatchString(m.TraitName) || len(m.TraitName) > 128) {
				fail(prop+".traitName", "traitName must start with a lowercase letter and hold at most 128 letters, digits, underscores, hyphens or dots")
			}
		}
	}
	return infer.CheckResponse[MemoryDataMappingArgs]{Inputs: args, Failures: failures}, nil
}

// enabled is the effective value: Twilio's default is true.
func (a MemoryDataMappingArgs) enabled() bool { return a.IsEnabled == nil || *a.IsEnabled }

func (r *MemoryDataMapping) Diff(_ context.Context, req infer.DiffRequest[MemoryDataMappingArgs, MemoryDataMappingState]) (infer.DiffResponse, error) {
	o, n := req.State.MemoryDataMappingArgs, req.Inputs
	detail := map[string]p.PropertyDiff{}
	set := func(prop string, changed bool, kind p.DiffKind) {
		if changed {
			detail[prop] = p.PropertyDiff{Kind: kind}
		}
	}
	set("storeId", o.StoreId != n.StoreId, p.UpdateReplace)
	set("fromType", o.FromType != n.FromType, p.UpdateReplace)
	set("columns", !slices.Equal(o.Columns, n.Columns), p.UpdateReplace)
	set("datasetId", deref(o.DatasetId) != deref(n.DatasetId), p.UpdateReplace)
	set("displayName", o.DisplayName != n.DisplayName, p.Update)
	if deref(o.Description) != deref(n.Description) {
		kind := p.Update
		if n.Description == nil {
			kind = p.UpdateReplace
		}
		detail["description"] = p.PropertyDiff{Kind: kind}
	}
	set("isEnabled", o.enabled() != n.enabled(), p.Update)
	set("mappings", !slices.Equal(o.Mappings, n.Mappings), p.Update)
	return infer.DiffResponse{HasChanges: len(detail) > 0, DetailedDiff: detail}, nil
}

func (s *MemoryDataMappingState) fill(o obj) {
	if id := o.str("id"); id != "" {
		s.DataMappingId = id
	}
	if v := o.optInt("version"); v != nil {
		s.Revision = *v
	}
	s.CreatedAt = o.str("createdAt")
	s.UpdatedAt = o.str("updatedAt")
}

func (a MemoryDataMappingArgs) from() map[string]any {
	if a.FromType == "INGRESS" {
		return map[string]any{"type": "INGRESS", "columns": a.Columns}
	}
	return map[string]any{"type": a.FromType, "datasetId": deref(a.DatasetId)}
}

func splitMemoryMappingID(id string) (store, mapping string, err error) {
	store, mapping, ok := strings.Cut(id, "/")
	if !ok || store == "" || mapping == "" || strings.Contains(mapping, "/") {
		return "", "", fmt.Errorf("invalid data mapping ID %q: expected <storeId>/<dataMappingId>", id)
	}
	return store, mapping, nil
}

func (r *MemoryDataMapping) Create(ctx context.Context, req infer.CreateRequest[MemoryDataMappingArgs]) (infer.CreateResponse[MemoryDataMappingState], error) {
	a := req.Inputs
	type resp = infer.CreateResponse[MemoryDataMappingState]
	if req.DryRun {
		return resp{Output: MemoryDataMappingState{MemoryDataMappingArgs: a}}, nil
	}
	c, err := clientFor(ctx)
	if err != nil {
		return resp{}, err
	}
	body := map[string]any{"displayName": a.DisplayName, "mappingFrom": a.from(), "mappingTo": traitMappingsBody(a.Mappings)}
	if a.Description != nil {
		body["description"] = *a.Description
	}
	if a.IsEnabled != nil {
		body["isEnabled"] = *a.IsEnabled
	}
	what := "create data mapping " + a.DisplayName
	acc, err := c.doJSON(ctx, "POST", "memory", memoryMappingsPath(a.StoreId), nil, body)
	if err != nil {
		return resp{}, fmt.Errorf("%s: %w", what, err)
	}
	op, err := waitMemoryAccepted(ctx, c, acc, what)
	if err != nil {
		return resp{}, err
	}
	id := op.str("result.id")
	if id == "" {
		return resp{}, fmt.Errorf("%s: Twilio completed the operation without a data mapping ID", what)
	}
	o, err := waitMemoryVisible(ctx, c, memoryMappingPath(a.StoreId, id), "data mapping "+id)
	if err != nil {
		// Do not leave a data mapping behind that no state tracks.
		_, _ = c.do(context.WithoutCancel(ctx), "DELETE", "memory", memoryMappingPath(a.StoreId, id), nil, nil)
		return resp{}, fmt.Errorf("%s: %w", what, err)
	}
	st := MemoryDataMappingState{MemoryDataMappingArgs: a, DataMappingId: id}
	st.fill(o)
	return resp{ID: a.StoreId + "/" + id, Output: st}, nil
}

func (r *MemoryDataMapping) Read(ctx context.Context, req infer.ReadRequest[MemoryDataMappingArgs, MemoryDataMappingState]) (infer.ReadResponse[MemoryDataMappingArgs, MemoryDataMappingState], error) {
	type resp = infer.ReadResponse[MemoryDataMappingArgs, MemoryDataMappingState]
	store, id, err := splitMemoryMappingID(req.ID)
	if err != nil {
		return resp{}, err
	}
	c, err := clientFor(ctx)
	if err != nil {
		return resp{}, err
	}
	o, err := c.do(ctx, "GET", "memory", memoryMappingPath(store, id), nil, nil)
	if err != nil {
		if isNotFound(err) {
			if err := notFoundOnImport("data mapping", req.ID, req.Inputs, req.State); err != nil {
				return resp{}, err
			}
			return resp{}, nil
		}
		return resp{}, fmt.Errorf("read data mapping %s: %w", req.ID, err)
	}
	in := req.Inputs
	imported := in.FromType == ""
	in.StoreId = store
	in.DisplayName = o.str("displayName")
	in.Description = nonEmpty(o.optStr("description"))
	if en := o.optBool("isEnabled"); en != nil && (*en != true || !imported && in.IsEnabled != nil) {
		in.IsEnabled = en
	}
	in.FromType = o.str("mappingFrom.type")
	in.Columns, in.DatasetId = nil, nil
	if cols := o.optList("mappingFrom.columns"); cols != nil {
		in.Columns = *cols
	}
	in.DatasetId = nonEmpty(o.optStr("mappingFrom.datasetId"))
	in.Mappings = traitMappingsOf(o)
	st := MemoryDataMappingState{MemoryDataMappingArgs: in, DataMappingId: id}
	st.fill(o)
	return resp{ID: req.ID, Inputs: in, State: st}, nil
}

func (r *MemoryDataMapping) Update(ctx context.Context, req infer.UpdateRequest[MemoryDataMappingArgs, MemoryDataMappingState]) (infer.UpdateResponse[MemoryDataMappingState], error) {
	type resp = infer.UpdateResponse[MemoryDataMappingState]
	st := req.State
	st.MemoryDataMappingArgs = req.Inputs
	if req.DryRun {
		return resp{Output: st}, nil
	}
	store, id, err := splitMemoryMappingID(req.ID)
	if err != nil {
		return resp{}, err
	}
	c, err := clientFor(ctx)
	if err != nil {
		return resp{}, err
	}
	n, o := req.Inputs, req.State.MemoryDataMappingArgs
	body := map[string]any{}
	if n.DisplayName != o.DisplayName {
		body["displayName"] = n.DisplayName
	}
	if deref(n.Description) != deref(o.Description) {
		body["description"] = deref(n.Description)
	}
	if n.enabled() != o.enabled() {
		body["isEnabled"] = n.enabled()
	}
	if !slices.Equal(n.Mappings, o.Mappings) {
		body["mappingTo"] = traitMappingsBody(n.Mappings)
	}
	what := "update data mapping " + req.ID
	if len(body) > 0 {
		acc, err := c.doJSON(ctx, "PATCH", "memory", memoryMappingPath(store, id), nil, body)
		if err != nil {
			return resp{}, fmt.Errorf("%s: %w", what, err)
		}
		if _, err := waitMemoryAccepted(ctx, c, acc, what); err != nil {
			return resp{}, err
		}
	}
	got, err := waitMemoryVisible(ctx, c, memoryMappingPath(store, id), "data mapping "+id)
	if err != nil {
		return resp{}, fmt.Errorf("%s: %w", what, err)
	}
	st.fill(got)
	return resp{Output: st}, nil
}

func (r *MemoryDataMapping) Delete(ctx context.Context, req infer.DeleteRequest[MemoryDataMappingState]) (infer.DeleteResponse, error) {
	store, id, err := splitMemoryMappingID(req.ID)
	if err != nil {
		return infer.DeleteResponse{}, err
	}
	c, err := clientFor(ctx)
	if err != nil {
		return infer.DeleteResponse{}, err
	}
	what := "delete data mapping " + req.ID
	acc, err := c.do(ctx, "DELETE", "memory", memoryMappingPath(store, id), nil, nil)
	if err != nil {
		if isNotFound(err) {
			return infer.DeleteResponse{}, nil
		}
		return infer.DeleteResponse{}, fmt.Errorf("%s: %w", what, err)
	}
	if _, err := waitMemoryAccepted(ctx, c, acc, what); err != nil {
		return infer.DeleteResponse{}, err
	}
	if err := waitGone(ctx, c, "memory", memoryMappingPath(store, id), "deleting data mapping "+req.ID); err != nil {
		return infer.DeleteResponse{}, fmt.Errorf("%s: %w", what, err)
	}
	return infer.DeleteResponse{}, nil
}

// IdentifierConfig is the resolution setting of one identifier type.
type IdentifierConfig struct {
	IdType            string  `pulumi:"idType"`
	MatchingAlgo      *string `pulumi:"matchingAlgo,optional"`
	MatchingThreshold *int    `pulumi:"matchingThreshold,optional"`
	Limit             *int    `pulumi:"limit,optional"`
	LimitPolicy       *string `pulumi:"limitPolicy,optional"`
	EnforceUnique     *bool   `pulumi:"enforceUnique,optional"`
	Normalization     *string `pulumi:"normalization,optional"`
}

func (c *IdentifierConfig) Annotate(a infer.Annotator) {
	a.Describe(&c.IdType, "Name of the identifier type, 2 to 30 characters: usually email, phone or external_id.")
	a.Describe(&c.MatchingAlgo, "How identifier values are matched: exact or fuzzy. Twilio's default is exact.")
	a.Describe(&c.MatchingThreshold, "The fuzzy matching threshold percentage, 50 to 99. Twilio's default is 75.")
	a.Describe(&c.Limit, "Maximum number of historical values to keep, 1 to 100. Twilio's default is 100.")
	a.Describe(&c.LimitPolicy, "What is removed when the limit is exceeded: fifo (oldest first) or lifo (newest first). Twilio's default is fifo.")
	a.Describe(&c.EnforceUnique, "When true, no two profiles share an identifier value and adding a shared value to a second profile may merge them. Twilio's default is true.")
	a.Describe(&c.Normalization, "Normalization applied before storing and matching: phone (E.164), email, trim or none. Twilio's default is trim.")
}

var (
	identifierAlgos         = map[string]bool{"exact": true, "fuzzy": true}
	identifierLimitPolicies = map[string]bool{"fifo": true, "lifo": true}
	identifierNormalization = map[string]bool{"phone": true, "email": true, "trim": true, "none": true}
)

// effectiveIdentifier is an IdentifierConfig with Twilio's defaults filled in, the form two configurations are compared in.
type effectiveIdentifier struct {
	idType, algo, limitPolicy, normalization string
	threshold, limit                         int
	enforceUnique                            bool
}

func (c IdentifierConfig) effective() effectiveIdentifier {
	e := effectiveIdentifier{idType: c.IdType, algo: derefOr(c.MatchingAlgo, "exact"), limitPolicy: derefOr(c.LimitPolicy, "fifo"),
		normalization: derefOr(c.Normalization, "trim"), threshold: intOr(c.MatchingThreshold, 75), limit: intOr(c.Limit, 100), enforceUnique: true}
	if c.EnforceUnique != nil {
		e.enforceUnique = *c.EnforceUnique
	}
	return e
}

func (c IdentifierConfig) equal(d IdentifierConfig) bool { return c.effective() == d.effective() }

// body sends only the settings the program stated.
func (c IdentifierConfig) body() map[string]any {
	m := map[string]any{"idType": c.IdType}
	if c.MatchingAlgo != nil {
		m["matchingAlgo"] = *c.MatchingAlgo
	}
	if c.MatchingThreshold != nil {
		m["matchingThreshold"] = *c.MatchingThreshold
	}
	if c.Limit != nil {
		m["limit"] = *c.Limit
	}
	if c.LimitPolicy != nil {
		m["limitPolicy"] = *c.LimitPolicy
	}
	if c.EnforceUnique != nil {
		m["enforceUnique"] = *c.EnforceUnique
	}
	if c.Normalization != nil {
		m["normalization"] = *c.Normalization
	}
	return m
}

func identifierConfigsOf(o obj) []IdentifierConfig {
	raw, _ := o.get("identifierConfigs").([]any)
	out := make([]IdentifierConfig, 0, len(raw))
	for _, it := range raw {
		m, ok := it.(map[string]any)
		if !ok {
			continue
		}
		eo := obj(m)
		out = append(out, IdentifierConfig{IdType: eo.str("idType"), MatchingAlgo: nonEmpty(eo.optStr("matchingAlgo")),
			MatchingThreshold: eo.optInt("matchingThreshold"), Limit: eo.optInt("limit"), LimitPolicy: nonEmpty(eo.optStr("limitPolicy")),
			EnforceUnique: eo.optBool("enforceUnique"), Normalization: nonEmpty(eo.optStr("normalization"))})
	}
	return out
}

// MemoryIdentityResolutionSettings manages how a Conversation Memory store resolves identifiers into profiles. It is a singleton of the store: create writes the settings, destroy only forgets them.
type MemoryIdentityResolutionSettings struct{}

// MemoryIdentityResolutionSettingsArgs are the inputs of MemoryIdentityResolutionSettings.
type MemoryIdentityResolutionSettingsArgs struct {
	StoreId           string             `pulumi:"storeId" provider:"replaceOnChanges"`
	IdentifierConfigs []IdentifierConfig `pulumi:"identifierConfigs"`
	MatchingRules     []string           `pulumi:"matchingRules"`
}

func (a *MemoryIdentityResolutionSettingsArgs) Annotate(an infer.Annotator) {
	an.Describe(&a.StoreId, "ID (mem_store_...) of the memory store. Changing it replaces the resource, which then writes the other store's settings.")
	an.Describe(&a.IdentifierConfigs, "The identifier types and their resolution settings, 1 to 15. Twilio documents that some settings can only take their defaults for now. Updates in place; the whole list is written (Twilio's update is a full replacement).")
	an.Describe(&a.MatchingRules, "Priority list of identifier types used to find the profile new data belongs to, or to decide whether two profiles merge: 1 to 15 names of at most 30 characters, evaluated in order, any single match counts. Compound rules are not supported. Updates in place.")
}

// MemoryIdentityResolutionSettingsState is the persisted state.
type MemoryIdentityResolutionSettingsState struct {
	MemoryIdentityResolutionSettingsArgs

	Revision int `pulumi:"revision"`
}

func (s *MemoryIdentityResolutionSettingsState) Annotate(a infer.Annotator) {
	a.Describe(&s.Revision, "Version number Twilio reports for the settings; it increases on every successful update.")
}

func (r *MemoryIdentityResolutionSettings) Annotate(a infer.Annotator) {
	a.Describe(&r, "The identity resolution settings of a Twilio Conversation Memory store: which identifiers locate and merge customer profiles. "+
		"A store has exactly one set of settings, so create writes them (replacing what is there) and destroy only removes the resource from state, leaving the settings on Twilio. "+
		"Twilio applies the change asynchronously; the provider follows the operation. The resource ID is the store ID. "+
		"Experimental: needs Conversation Memory on the account and has not been verified against a live account.")
}

var _ infer.CustomResource[MemoryIdentityResolutionSettingsArgs, MemoryIdentityResolutionSettingsState] = (*MemoryIdentityResolutionSettings)(nil)
var _ infer.CustomCheck[MemoryIdentityResolutionSettingsArgs] = (*MemoryIdentityResolutionSettings)(nil)
var _ infer.CustomDiff[MemoryIdentityResolutionSettingsArgs, MemoryIdentityResolutionSettingsState] = (*MemoryIdentityResolutionSettings)(nil)
var _ infer.CustomRead[MemoryIdentityResolutionSettingsArgs, MemoryIdentityResolutionSettingsState] = (*MemoryIdentityResolutionSettings)(nil)
var _ infer.CustomUpdate[MemoryIdentityResolutionSettingsArgs, MemoryIdentityResolutionSettingsState] = (*MemoryIdentityResolutionSettings)(nil)
var _ infer.CustomDelete[MemoryIdentityResolutionSettingsState] = (*MemoryIdentityResolutionSettings)(nil)

func (r *MemoryIdentityResolutionSettings) Check(ctx context.Context, req infer.CheckRequest) (infer.CheckResponse[MemoryIdentityResolutionSettingsArgs], error) {
	type resp = infer.CheckResponse[MemoryIdentityResolutionSettingsArgs]
	args, failures, err := infer.DefaultCheck[MemoryIdentityResolutionSettingsArgs](ctx, req.NewInputs)
	if err != nil || len(failures) > 0 {
		return resp{Inputs: args, Failures: failures}, err
	}
	unk := unknowns(req.NewInputs)
	fail := func(prop, reason string) { failures = append(failures, p.CheckFailure{Property: prop, Reason: reason}) }
	if !unk("identifierConfigs") {
		if len(args.IdentifierConfigs) < 1 || len(args.IdentifierConfigs) > 15 {
			fail("identifierConfigs", "identifierConfigs needs 1 to 15 entries")
		}
		seen := map[string]bool{}
		for i, c := range args.IdentifierConfigs {
			prop := fmt.Sprintf("identifierConfigs[%d]", i)
			if unknownAt(req.NewInputs, "identifierConfigs", i) {
				continue
			}
			if !unknownAt(req.NewInputs, "identifierConfigs", i, "idType") {
				if n := utf8.RuneCountInString(c.IdType); n < 2 || n > 30 {
					fail(prop+".idType", "idType must be 2 to 30 characters")
				}
				if seen[c.IdType] {
					fail(prop+".idType", "idType "+c.IdType+" is listed twice")
				}
				seen[c.IdType] = true
			}
			if !unknownAt(req.NewInputs, "identifierConfigs", i, "matchingAlgo") && (c.MatchingAlgo != nil && !identifierAlgos[*c.MatchingAlgo]) {
				fail(prop+".matchingAlgo", "matchingAlgo must be exact or fuzzy")
			}
			if !unknownAt(req.NewInputs, "identifierConfigs", i, "matchingThreshold") && (c.MatchingThreshold != nil && (*c.MatchingThreshold < 50 || *c.MatchingThreshold > 99)) {
				fail(prop+".matchingThreshold", "matchingThreshold must be between 50 and 99")
			}
			if !unknownAt(req.NewInputs, "identifierConfigs", i, "limit") && (c.Limit != nil && (*c.Limit < 1 || *c.Limit > 100)) {
				fail(prop+".limit", "limit must be between 1 and 100")
			}
			if !unknownAt(req.NewInputs, "identifierConfigs", i, "limitPolicy") && (c.LimitPolicy != nil && !identifierLimitPolicies[*c.LimitPolicy]) {
				fail(prop+".limitPolicy", "limitPolicy must be fifo or lifo")
			}
			if !unknownAt(req.NewInputs, "identifierConfigs", i, "normalization") && (c.Normalization != nil && !identifierNormalization[*c.Normalization]) {
				fail(prop+".normalization", "normalization must be phone, email, trim or none")
			}
		}
	}
	if !unk("matchingRules") {
		if len(args.MatchingRules) < 1 || len(args.MatchingRules) > 15 {
			fail("matchingRules", "matchingRules needs 1 to 15 entries")
		}
		for i, m := range args.MatchingRules {
			if unknownAt(req.NewInputs, "matchingRules", i) {
				continue
			}
			if n := utf8.RuneCountInString(m); n < 1 || n > 30 {
				fail("matchingRules", "each matching rule is one identifier type of 1 to 30 characters")
				break
			}
		}
	}
	return resp{Inputs: args, Failures: failures}, nil
}

func (r *MemoryIdentityResolutionSettings) Diff(_ context.Context, req infer.DiffRequest[MemoryIdentityResolutionSettingsArgs, MemoryIdentityResolutionSettingsState]) (infer.DiffResponse, error) {
	o, n := req.State.MemoryIdentityResolutionSettingsArgs, req.Inputs
	detail := map[string]p.PropertyDiff{}
	if o.StoreId != n.StoreId {
		detail["storeId"] = p.PropertyDiff{Kind: p.UpdateReplace}
	}
	// A setting left out and Twilio's default are the same value.
	if !slices.EqualFunc(o.IdentifierConfigs, n.IdentifierConfigs, func(a, b IdentifierConfig) bool { return a.equal(b) }) {
		detail["identifierConfigs"] = p.PropertyDiff{Kind: p.Update}
	}
	if !slices.Equal(o.MatchingRules, n.MatchingRules) {
		detail["matchingRules"] = p.PropertyDiff{Kind: p.Update}
	}
	return infer.DiffResponse{HasChanges: len(detail) > 0, DetailedDiff: detail}, nil
}

// put writes the whole settings document and returns what Twilio then reports.
func (r *MemoryIdentityResolutionSettings) put(ctx context.Context, a MemoryIdentityResolutionSettingsArgs, what string) (obj, error) {
	c, err := clientFor(ctx)
	if err != nil {
		return nil, err
	}
	cfgs := make([]map[string]any, 0, len(a.IdentifierConfigs))
	for _, ic := range a.IdentifierConfigs {
		cfgs = append(cfgs, ic.body())
	}
	acc, err := c.doJSON(ctx, "PUT", "memory", memoryIdentityPath(a.StoreId), nil, map[string]any{"identifierConfigs": cfgs, "matchingRules": a.MatchingRules})
	if err != nil {
		return nil, fmt.Errorf("%s: %w", what, err)
	}
	if _, err := waitMemoryAccepted(ctx, c, acc, what); err != nil {
		return nil, err
	}
	o, err := waitMemoryVisible(ctx, c, memoryIdentityPath(a.StoreId), "identity resolution settings of "+a.StoreId)
	if err != nil {
		return nil, fmt.Errorf("%s: %w", what, err)
	}
	return o, nil
}

func (s *MemoryIdentityResolutionSettingsState) fill(o obj) {
	if v := o.optInt("version"); v != nil {
		s.Revision = *v
	}
}

func (r *MemoryIdentityResolutionSettings) Create(ctx context.Context, req infer.CreateRequest[MemoryIdentityResolutionSettingsArgs]) (infer.CreateResponse[MemoryIdentityResolutionSettingsState], error) {
	a := req.Inputs
	st := MemoryIdentityResolutionSettingsState{MemoryIdentityResolutionSettingsArgs: a}
	if req.DryRun {
		return infer.CreateResponse[MemoryIdentityResolutionSettingsState]{Output: st}, nil
	}
	o, err := r.put(ctx, a, "write identity resolution settings of "+a.StoreId)
	if err != nil {
		return infer.CreateResponse[MemoryIdentityResolutionSettingsState]{}, err
	}
	st.fill(o)
	return infer.CreateResponse[MemoryIdentityResolutionSettingsState]{ID: a.StoreId, Output: st}, nil
}

func (r *MemoryIdentityResolutionSettings) Update(ctx context.Context, req infer.UpdateRequest[MemoryIdentityResolutionSettingsArgs, MemoryIdentityResolutionSettingsState]) (infer.UpdateResponse[MemoryIdentityResolutionSettingsState], error) {
	st := req.State
	st.MemoryIdentityResolutionSettingsArgs = req.Inputs
	if req.DryRun {
		return infer.UpdateResponse[MemoryIdentityResolutionSettingsState]{Output: st}, nil
	}
	o, err := r.put(ctx, req.Inputs, "update identity resolution settings of "+req.ID)
	if err != nil {
		return infer.UpdateResponse[MemoryIdentityResolutionSettingsState]{}, err
	}
	st.fill(o)
	return infer.UpdateResponse[MemoryIdentityResolutionSettingsState]{Output: st}, nil
}

func (r *MemoryIdentityResolutionSettings) Read(ctx context.Context, req infer.ReadRequest[MemoryIdentityResolutionSettingsArgs, MemoryIdentityResolutionSettingsState]) (infer.ReadResponse[MemoryIdentityResolutionSettingsArgs, MemoryIdentityResolutionSettingsState], error) {
	type resp = infer.ReadResponse[MemoryIdentityResolutionSettingsArgs, MemoryIdentityResolutionSettingsState]
	c, err := clientFor(ctx)
	if err != nil {
		return resp{}, err
	}
	o, err := c.do(ctx, "GET", "memory", memoryIdentityPath(req.ID), nil, nil)
	if err != nil {
		if isNotFound(err) {
			if err := notFoundOnImport("identity resolution settings", req.ID, req.Inputs, req.State); err != nil {
				return resp{}, err
			}
			return resp{}, nil
		}
		return resp{}, fmt.Errorf("read identity resolution settings %s: %w", req.ID, err)
	}
	in := req.Inputs
	in.StoreId = req.ID
	in.MatchingRules = nil
	if rules := o.optList("matchingRules"); rules != nil {
		in.MatchingRules = *rules
	}
	// Twilio reports every setting. A refresh keeps the program's own spelling where it already means the
	// same value, so a setting the program leaves out is not turned into the default.
	got := identifierConfigsOf(o)
	if len(req.Inputs.IdentifierConfigs) > 0 && slices.EqualFunc(req.Inputs.IdentifierConfigs, got, func(a, b IdentifierConfig) bool { return a.equal(b) }) {
		in.IdentifierConfigs = req.Inputs.IdentifierConfigs
	} else {
		in.IdentifierConfigs = got
	}
	st := MemoryIdentityResolutionSettingsState{MemoryIdentityResolutionSettingsArgs: in}
	st.fill(o)
	return resp{ID: req.ID, Inputs: in, State: st}, nil
}

// Delete only forgets: Twilio has no call that removes the settings of a store.
func (r *MemoryIdentityResolutionSettings) Delete(_ context.Context, _ infer.DeleteRequest[MemoryIdentityResolutionSettingsState]) (infer.DeleteResponse, error) {
	return infer.DeleteResponse{}, nil
}
