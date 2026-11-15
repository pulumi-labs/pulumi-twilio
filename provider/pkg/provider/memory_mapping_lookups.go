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

	"github.com/pulumi/pulumi-go-provider/infer"
)

// GetMemoryDataMapping looks up one data mapping of a Memory store.
type GetMemoryDataMapping struct{}

// GetMemoryDataMappingArgs are the lookup arguments.
type GetMemoryDataMappingArgs struct {
	StoreId       string `pulumi:"storeId"`
	DataMappingId string `pulumi:"dataMappingId"`
}

func (a *GetMemoryDataMappingArgs) Annotate(an infer.Annotator) {
	an.Describe(&a.StoreId, "ID (mem_store_...) of the memory store.")
	an.Describe(&a.DataMappingId, "ID (mem_datamapping_...) of the data mapping.")
}

// GetMemoryDataMappingResult is the looked-up data mapping.
type GetMemoryDataMappingResult struct {
	StoreId       string               `pulumi:"storeId"`
	DataMappingId string               `pulumi:"dataMappingId"`
	DisplayName   string               `pulumi:"displayName"`
	Description   *string              `pulumi:"description,optional"`
	IsEnabled     bool                 `pulumi:"isEnabled"`
	FromType      string               `pulumi:"fromType"`
	Columns       []string             `pulumi:"columns,optional"`
	DatasetId     *string              `pulumi:"datasetId,optional"`
	Mappings      []MemoryTraitMapping `pulumi:"mappings"`
	Revision      int                  `pulumi:"revision"`
	CreatedAt     string               `pulumi:"createdAt"`
	UpdatedAt     string               `pulumi:"updatedAt"`
}

func (a *GetMemoryDataMappingResult) Annotate(an infer.Annotator) {
	an.Describe(&a.StoreId, "The store ID.")
	an.Describe(&a.DataMappingId, "The data mapping ID.")
	an.Describe(&a.DisplayName, "Name of the data mapping.")
	an.Describe(&a.Description, "Description of the data mapping, if any.")
	an.Describe(&a.IsEnabled, "Whether data is ingested and mapped.")
	an.Describe(&a.FromType, "Where the data comes from: INGRESS, DATASET_CLOUDAPP or DATASET_WAREHOUSE.")
	an.Describe(&a.Columns, "INGRESS sources: the source column names.")
	an.Describe(&a.DatasetId, "Dataset sources: the ID of the connected dataset.")
	an.Describe(&a.Mappings, "The source field to trait mappings.")
	an.Describe(&a.Revision, "Version number Twilio reports for the data mapping.")
	an.Describe(&a.CreatedAt, "When the data mapping was created (ISO 8601).")
	an.Describe(&a.UpdatedAt, "When the data mapping was last updated (ISO 8601).")
}

func (f *GetMemoryDataMapping) Annotate(a infer.Annotator) {
	a.Describe(&f, "Look up a data mapping of a Twilio Conversation Memory store. Experimental: needs Conversation Memory on the account and is unverified against a live account.")
}

func (f *GetMemoryDataMapping) Invoke(ctx context.Context, req infer.FunctionRequest[GetMemoryDataMappingArgs]) (infer.FunctionResponse[GetMemoryDataMappingResult], error) {
	const fn = "getMemoryDataMapping"
	in := req.Input
	if err := requireArgs(fn, [2]string{"storeId", in.StoreId}, [2]string{"dataMappingId", in.DataMappingId}); err != nil {
		return infer.FunctionResponse[GetMemoryDataMappingResult]{}, err
	}
	c, err := clientFor(ctx)
	if err != nil {
		return infer.FunctionResponse[GetMemoryDataMappingResult]{}, err
	}
	o, err := c.do(ctx, "GET", "memory", memoryMappingPath(in.StoreId, in.DataMappingId), nil, nil)
	if err != nil {
		return infer.FunctionResponse[GetMemoryDataMappingResult]{}, lookupFailure(fn, "data mapping "+in.DataMappingId, err)
	}
	st := MemoryDataMappingState{}
	st.fill(o)
	res := GetMemoryDataMappingResult{StoreId: in.StoreId, DataMappingId: in.DataMappingId, DisplayName: o.str("displayName"),
		Description: nonEmpty(o.optStr("description")), IsEnabled: derefBool(o.optBool("isEnabled")) || o.optBool("isEnabled") == nil,
		FromType: o.str("mappingFrom.type"), DatasetId: nonEmpty(o.optStr("mappingFrom.datasetId")), Mappings: traitMappingsOf(o),
		Revision: st.Revision, CreatedAt: st.CreatedAt, UpdatedAt: st.UpdatedAt}
	if cols := o.optList("mappingFrom.columns"); cols != nil {
		res.Columns = *cols
	}
	return infer.FunctionResponse[GetMemoryDataMappingResult]{Output: res}, nil
}

// GetMemoryIdentityResolutionSettings looks up the identity resolution settings of a Memory store.
type GetMemoryIdentityResolutionSettings struct{}

// GetMemoryIdentityResolutionSettingsArgs are the lookup arguments.
type GetMemoryIdentityResolutionSettingsArgs struct {
	StoreId string `pulumi:"storeId"`
}

func (a *GetMemoryIdentityResolutionSettingsArgs) Annotate(an infer.Annotator) {
	an.Describe(&a.StoreId, "ID (mem_store_...) of the memory store.")
}

// GetMemoryIdentityResolutionSettingsResult is the looked-up settings.
type GetMemoryIdentityResolutionSettingsResult struct {
	StoreId           string             `pulumi:"storeId"`
	IdentifierConfigs []IdentifierConfig `pulumi:"identifierConfigs"`
	MatchingRules     []string           `pulumi:"matchingRules"`
	Revision          int                `pulumi:"revision"`
}

func (a *GetMemoryIdentityResolutionSettingsResult) Annotate(an infer.Annotator) {
	an.Describe(&a.StoreId, "The store ID.")
	an.Describe(&a.IdentifierConfigs, "The identifier types and their resolution settings, as Twilio reports them.")
	an.Describe(&a.MatchingRules, "Priority list of identifier types used to find or merge profiles.")
	an.Describe(&a.Revision, "Version number Twilio reports for the settings.")
}

func (f *GetMemoryIdentityResolutionSettings) Annotate(a infer.Annotator) {
	a.Describe(&f, "Look up the identity resolution settings of a Twilio Conversation Memory store. Experimental: needs Conversation Memory on the account and is unverified against a live account.")
}

func (f *GetMemoryIdentityResolutionSettings) Invoke(ctx context.Context, req infer.FunctionRequest[GetMemoryIdentityResolutionSettingsArgs]) (infer.FunctionResponse[GetMemoryIdentityResolutionSettingsResult], error) {
	const fn = "getMemoryIdentityResolutionSettings"
	if err := requireArg(fn, "storeId", req.Input.StoreId); err != nil {
		return infer.FunctionResponse[GetMemoryIdentityResolutionSettingsResult]{}, err
	}
	c, err := clientFor(ctx)
	if err != nil {
		return infer.FunctionResponse[GetMemoryIdentityResolutionSettingsResult]{}, err
	}
	o, err := c.do(ctx, "GET", "memory", memoryIdentityPath(req.Input.StoreId), nil, nil)
	if err != nil {
		return infer.FunctionResponse[GetMemoryIdentityResolutionSettingsResult]{}, lookupFailure(fn, "identity resolution settings of "+req.Input.StoreId, err)
	}
	st := MemoryIdentityResolutionSettingsState{}
	st.fill(o)
	res := GetMemoryIdentityResolutionSettingsResult{StoreId: req.Input.StoreId, IdentifierConfigs: identifierConfigsOf(o), Revision: st.Revision}
	if rules := o.optList("matchingRules"); rules != nil {
		res.MatchingRules = *rules
	}
	return infer.FunctionResponse[GetMemoryIdentityResolutionSettingsResult]{Output: res}, nil
}
