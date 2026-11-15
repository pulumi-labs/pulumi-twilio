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

// GetMemoryStore looks up a Twilio Conversation Memory store by ID.
type GetMemoryStore struct{}

// GetMemoryStoreArgs are the lookup arguments.
type GetMemoryStoreArgs struct {
	StoreId string `pulumi:"storeId"`
}

func (a *GetMemoryStoreArgs) Annotate(an infer.Annotator) {
	an.Describe(&a.StoreId, "ID (mem_store_...) of the memory store.")
}

// GetMemoryStoreResult is the looked-up store.
type GetMemoryStoreResult struct {
	StoreId               string  `pulumi:"storeId"`
	DisplayName           string  `pulumi:"displayName"`
	Description           *string `pulumi:"description,optional"`
	Status                string  `pulumi:"status"`
	Revision              int     `pulumi:"revision"`
	IntelligenceServiceId *string `pulumi:"intelligenceServiceId,optional"`
	DataAsOf              *string `pulumi:"dataAsOf,optional"`
}

func (a *GetMemoryStoreResult) Annotate(an infer.Annotator) {
	an.Describe(&a.StoreId, "The store ID.")
	an.Describe(&a.DisplayName, "Name of the store.")
	an.Describe(&a.Description, "Description of the store, if any.")
	an.Describe(&a.Status, "Provisioning status: QUEUED, PROVISIONING, ACTIVE or FAILED.")
	an.Describe(&a.Revision, "Version number Twilio reports for the store.")
	an.Describe(&a.IntelligenceServiceId, "ID of the Conversational Intelligence service provisioned with the store.")
	an.Describe(&a.DataAsOf, "Time up to which data sent to the store has been processed (ISO 8601).")
}

func (f *GetMemoryStore) Annotate(a infer.Annotator) {
	a.Describe(&f, "Look up a Twilio Conversation Memory store by ID. Experimental: needs Conversation Memory on the account and is unverified against a live account.")
}

func (f *GetMemoryStore) Invoke(ctx context.Context, req infer.FunctionRequest[GetMemoryStoreArgs]) (infer.FunctionResponse[GetMemoryStoreResult], error) {
	const fn = "getMemoryStore"
	if err := requireArg(fn, "storeId", req.Input.StoreId); err != nil {
		return infer.FunctionResponse[GetMemoryStoreResult]{}, err
	}
	c, err := clientFor(ctx)
	if err != nil {
		return infer.FunctionResponse[GetMemoryStoreResult]{}, err
	}
	o, err := c.do(ctx, "GET", "memory", memoryStorePath(req.Input.StoreId), nil, nil)
	if err != nil {
		return infer.FunctionResponse[GetMemoryStoreResult]{}, lookupFailure(fn, "memory store "+req.Input.StoreId, err)
	}
	st := MemoryStoreState{}
	st.fill(o)
	return infer.FunctionResponse[GetMemoryStoreResult]{Output: GetMemoryStoreResult{
		StoreId: req.Input.StoreId, DisplayName: o.str("displayName"), Description: nonEmpty(o.optStr("description")),
		Status: st.Status, Revision: st.Revision, IntelligenceServiceId: st.IntelligenceServiceId, DataAsOf: st.DataAsOf,
	}}, nil
}

// GetMemoryTraitGroup looks up one trait group of a Memory store.
type GetMemoryTraitGroup struct{}

// GetMemoryTraitGroupArgs are the lookup arguments.
type GetMemoryTraitGroupArgs struct {
	StoreId     string `pulumi:"storeId"`
	DisplayName string `pulumi:"displayName"`
}

func (a *GetMemoryTraitGroupArgs) Annotate(an infer.Annotator) {
	an.Describe(&a.StoreId, "ID (mem_store_...) of the memory store.")
	an.Describe(&a.DisplayName, "Name of the trait group.")
}

// GetMemoryTraitGroupResult is the looked-up trait group.
type GetMemoryTraitGroupResult struct {
	StoreId     string                 `pulumi:"storeId"`
	DisplayName string                 `pulumi:"displayName"`
	Description *string                `pulumi:"description,optional"`
	Revision    int                    `pulumi:"revision"`
	Traits      map[string]MemoryTrait `pulumi:"traits,optional"`
}

func (a *GetMemoryTraitGroupResult) Annotate(an infer.Annotator) {
	an.Describe(&a.StoreId, "The store ID.")
	an.Describe(&a.DisplayName, "Name of the trait group.")
	an.Describe(&a.Description, "Description of the trait group, if any.")
	an.Describe(&a.Revision, "Version number Twilio reports for the trait group.")
	an.Describe(&a.Traits, "The traits of the group, keyed by trait name.")
}

func (f *GetMemoryTraitGroup) Annotate(a infer.Annotator) {
	a.Describe(&f, "Look up a trait group of a Twilio Conversation Memory store. Experimental: needs Conversation Memory on the account and is unverified against a live account.")
}

func (f *GetMemoryTraitGroup) Invoke(ctx context.Context, req infer.FunctionRequest[GetMemoryTraitGroupArgs]) (infer.FunctionResponse[GetMemoryTraitGroupResult], error) {
	const fn = "getMemoryTraitGroup"
	in := req.Input
	if err := requireArgs(fn, [2]string{"storeId", in.StoreId}, [2]string{"displayName", in.DisplayName}); err != nil {
		return infer.FunctionResponse[GetMemoryTraitGroupResult]{}, err
	}
	c, err := clientFor(ctx)
	if err != nil {
		return infer.FunctionResponse[GetMemoryTraitGroupResult]{}, err
	}
	o, err := c.do(ctx, "GET", "memory", memoryGroupPath(in.StoreId, in.DisplayName), nil, nil)
	if err != nil {
		return infer.FunctionResponse[GetMemoryTraitGroupResult]{}, lookupFailure(fn, "trait group "+in.DisplayName, err)
	}
	g := memoryGroupOf(o)
	res := GetMemoryTraitGroupResult{StoreId: in.StoreId, DisplayName: in.DisplayName, Description: nonEmpty(g.optStr("description")), Traits: memoryTraitsOf(g)}
	if v := g.optInt("version"); v != nil {
		res.Revision = *v
	}
	return infer.FunctionResponse[GetMemoryTraitGroupResult]{Output: res}, nil
}
