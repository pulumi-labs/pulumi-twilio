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

// GetKnowledgeBase looks up a Twilio Enterprise Knowledge base by ID.
type GetKnowledgeBase struct{}

// GetKnowledgeBaseArgs are the lookup arguments.
type GetKnowledgeBaseArgs struct {
	KnowledgeBaseId string `pulumi:"knowledgeBaseId"`
}

func (a *GetKnowledgeBaseArgs) Annotate(an infer.Annotator) {
	an.Describe(&a.KnowledgeBaseId, "ID (know_knowledgebase_...) of the knowledge base.")
}

// GetKnowledgeBaseResult is the looked-up knowledge base.
type GetKnowledgeBaseResult struct {
	KnowledgeBaseId string  `pulumi:"knowledgeBaseId"`
	DisplayName     string  `pulumi:"displayName"`
	Description     *string `pulumi:"description,optional"`
	Status          string  `pulumi:"status"`
	Revision        int     `pulumi:"revision"`
	CreatedAt       string  `pulumi:"createdAt"`
	UpdatedAt       string  `pulumi:"updatedAt"`
}

func (a *GetKnowledgeBaseResult) Annotate(an infer.Annotator) {
	an.Describe(&a.KnowledgeBaseId, "The knowledge base ID.")
	an.Describe(&a.DisplayName, "Name of the knowledge base.")
	an.Describe(&a.Description, "Description of the knowledge base, if any.")
	an.Describe(&a.Status, "Provisioning status: QUEUED, PROVISIONING, ACTIVE, FAILED or DELETING.")
	an.Describe(&a.Revision, "Version number Twilio reports for the knowledge base.")
	an.Describe(&a.CreatedAt, "When the knowledge base was created (ISO 8601).")
	an.Describe(&a.UpdatedAt, "When the knowledge base was last updated (ISO 8601).")
}

func (f *GetKnowledgeBase) Annotate(a infer.Annotator) {
	a.Describe(&f, "Look up a Twilio Enterprise Knowledge base by ID. Experimental: needs Enterprise Knowledge on the account and is unverified against a live account.")
}

func (f *GetKnowledgeBase) Invoke(ctx context.Context, req infer.FunctionRequest[GetKnowledgeBaseArgs]) (infer.FunctionResponse[GetKnowledgeBaseResult], error) {
	const fn = "getKnowledgeBase"
	if err := requireArg(fn, "knowledgeBaseId", req.Input.KnowledgeBaseId); err != nil {
		return infer.FunctionResponse[GetKnowledgeBaseResult]{}, err
	}
	c, err := clientFor(ctx)
	if err != nil {
		return infer.FunctionResponse[GetKnowledgeBaseResult]{}, err
	}
	o, err := c.do(ctx, "GET", "knowledge", knowledgeBasePath(req.Input.KnowledgeBaseId), nil, nil)
	if err != nil {
		return infer.FunctionResponse[GetKnowledgeBaseResult]{}, lookupFailure(fn, "knowledge base "+req.Input.KnowledgeBaseId, err)
	}
	st := KnowledgeBaseState{}
	st.fill(o)
	return infer.FunctionResponse[GetKnowledgeBaseResult]{Output: GetKnowledgeBaseResult{
		KnowledgeBaseId: req.Input.KnowledgeBaseId, DisplayName: o.str("displayName"), Description: nonEmpty(o.optStr("description")),
		Status: st.Status, Revision: st.Revision, CreatedAt: st.CreatedAt, UpdatedAt: st.UpdatedAt,
	}}, nil
}

// GetKnowledge looks up one Enterprise Knowledge source.
type GetKnowledge struct{}

// GetKnowledgeArgs are the lookup arguments.
type GetKnowledgeArgs struct {
	KnowledgeBaseId string `pulumi:"knowledgeBaseId"`
	KnowledgeId     string `pulumi:"knowledgeId"`
}

func (a *GetKnowledgeArgs) Annotate(an infer.Annotator) {
	an.Describe(&a.KnowledgeBaseId, "ID (know_knowledgebase_...) of the knowledge base.")
	an.Describe(&a.KnowledgeId, "ID (know_knowledge_...) of the source.")
}

// GetKnowledgeResult is the looked-up source. The text of a Text source is not returned.
type GetKnowledgeResult struct {
	KnowledgeBaseId string  `pulumi:"knowledgeBaseId"`
	KnowledgeId     string  `pulumi:"knowledgeId"`
	Name            string  `pulumi:"name"`
	Description     *string `pulumi:"description,optional"`
	Status          string  `pulumi:"status"`
	SourceType      string  `pulumi:"sourceType"`
	WebUrl          *string `pulumi:"webUrl,optional"`
	CrawlDepth      *int    `pulumi:"crawlDepth,optional"`
	CrawlPeriod     *string `pulumi:"crawlPeriod,optional"`
	CreatedAt       string  `pulumi:"createdAt"`
	UpdatedAt       string  `pulumi:"updatedAt"`
}

func (a *GetKnowledgeResult) Annotate(an infer.Annotator) {
	an.Describe(&a.KnowledgeBaseId, "The knowledge base ID.")
	an.Describe(&a.KnowledgeId, "The source ID.")
	an.Describe(&a.Name, "Name of the source.")
	an.Describe(&a.Description, "Description of the source, if any.")
	an.Describe(&a.Status, "Processing status: SCHEDULED, QUEUED, PROCESSING, COMPLETED or FAILED.")
	an.Describe(&a.SourceType, "Kind of source: Web, Text or File.")
	an.Describe(&a.WebUrl, "Web sources: the URL crawled.")
	an.Describe(&a.CrawlDepth, "Web sources: how many links deep the crawl goes.")
	an.Describe(&a.CrawlPeriod, "Web sources: how often the site is re-crawled.")
	an.Describe(&a.CreatedAt, "When the source was created (ISO 8601).")
	an.Describe(&a.UpdatedAt, "When the source was last updated (ISO 8601).")
}

func (f *GetKnowledge) Annotate(a infer.Annotator) {
	a.Describe(&f, "Look up a source in a Twilio Enterprise Knowledge base. Experimental: needs Enterprise Knowledge on the account and is unverified against a live account.")
}

func (f *GetKnowledge) Invoke(ctx context.Context, req infer.FunctionRequest[GetKnowledgeArgs]) (infer.FunctionResponse[GetKnowledgeResult], error) {
	const fn = "getKnowledge"
	if err := requireArgs(fn, [2]string{"knowledgeBaseId", req.Input.KnowledgeBaseId}, [2]string{"knowledgeId", req.Input.KnowledgeId}); err != nil {
		return infer.FunctionResponse[GetKnowledgeResult]{}, err
	}
	c, err := clientFor(ctx)
	if err != nil {
		return infer.FunctionResponse[GetKnowledgeResult]{}, err
	}
	in := req.Input
	o, err := c.do(ctx, "GET", "knowledge", knowledgeItemPath(in.KnowledgeBaseId, in.KnowledgeId), nil, nil)
	if err != nil {
		return infer.FunctionResponse[GetKnowledgeResult]{}, lookupFailure(fn, "knowledge source "+in.KnowledgeId, err)
	}
	return infer.FunctionResponse[GetKnowledgeResult]{Output: GetKnowledgeResult{
		KnowledgeBaseId: in.KnowledgeBaseId, KnowledgeId: in.KnowledgeId, Name: o.str("name"),
		Description: nonEmpty(o.optStr("description")), Status: o.str("status"), SourceType: o.str("source.type"),
		WebUrl: nonEmpty(o.optStr("source.url")), CrawlDepth: o.optInt("source.crawlDepth"), CrawlPeriod: nonEmpty(o.optStr("source.crawlPeriod")),
		CreatedAt: o.str("createdAt"), UpdatedAt: o.str("updatedAt"),
	}}, nil
}
