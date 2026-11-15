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
	"context"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"time"
	"unicode/utf8"

	p "github.com/pulumi/pulumi-go-provider"
	"github.com/pulumi/pulumi-go-provider/infer"
)

// Twilio Enterprise Knowledge (knowledge.twilio.com, API version v2) is new and reaches only accounts
// that have it enabled in the Console. The shapes below follow twilio-oai spec/json/twilio_knowledge_v2.json
// (release 2.11.0); nothing here has been exercised against a live account yet.

// knowledgePollInterval and knowledgePollTimeout pace the waits for an asynchronous Knowledge Base
// operation and for a Knowledge source to be processed; tests shrink both.
var (
	knowledgePollInterval = 3 * time.Second
	knowledgePollTimeout  = 30 * time.Minute
)

const (
	knowledgeBasesPath = "/v2/ControlPlane/KnowledgeBases"
	knowledgeOpsPath   = "/v2/ControlPlane/Operations/"
)

func knowledgeBasePath(id string) string { return knowledgeBasesPath + "/" + url.PathEscape(id) }

func knowledgeCollectionPath(kb string) string {
	return "/v2/KnowledgeBases/" + url.PathEscape(kb) + "/Knowledge"
}

func knowledgeItemPath(kb, id string) string {
	return knowledgeCollectionPath(kb) + "/" + url.PathEscape(id)
}

// operationID reads the operation ID from the last path segment of the statusUrl an asynchronous
// call returns.
func operationID(statusURL string) string {
	if i := strings.IndexAny(statusURL, "?#"); i >= 0 {
		statusURL = statusURL[:i]
	}
	statusURL = strings.TrimRight(statusURL, "/")
	return statusURL[strings.LastIndex(statusURL, "/")+1:]
}

// pollUntil calls check every knowledgePollInterval until it reports done, fails, or the deadline passes.
func pollUntil(ctx context.Context, what string, check func(ctx context.Context) (bool, error)) error {
	ctx, cancel := context.WithTimeout(ctx, knowledgePollTimeout)
	defer cancel()
	for {
		done, err := check(ctx)
		if err != nil {
			if ctx.Err() != nil {
				return fmt.Errorf("%s did not finish within %s: %w", what, knowledgePollTimeout, ctx.Err())
			}
			return err
		}
		if done {
			return nil
		}
		select {
		case <-ctx.Done():
			return fmt.Errorf("%s did not finish within %s: %w", what, knowledgePollTimeout, ctx.Err())
		case <-time.After(knowledgePollInterval):
		}
	}
}

// waitOperation follows an asynchronous control-plane operation (product knowledge or memory) to
// COMPLETED and returns it; FAILED and CANCELLED are errors carrying Twilio's explanation.
func waitOperation(ctx context.Context, c *Client, product, opsPath string, accepted obj, what string) (obj, error) {
	sURL := accepted.str("statusUrl")
	if sURL == "" {
		return nil, fmt.Errorf("%s: Twilio accepted the request without a statusUrl, so completion cannot be followed", what)
	}
	id := operationID(sURL)
	var out obj
	err := pollUntil(ctx, what, func(ctx context.Context) (bool, error) {
		o, err := c.do(ctx, "GET", product, opsPath+url.PathEscape(id), nil, nil)
		if err != nil {
			return false, err
		}
		switch s := o.str("status"); s {
		case "COMPLETED":
			out = o
			return true, nil
		case "FAILED", "CANCELLED":
			reason := strings.TrimSpace(o.str("error.title") + " " + o.str("error.detail"))
			if reason == "" {
				reason = "no reason given"
			}
			return false, fmt.Errorf("%s: operation %s %s: %s", what, id, strings.ToLower(s), reason)
		}
		return false, nil
	})
	return out, err
}

// KnowledgeBase manages a Twilio Enterprise Knowledge base: the container that Knowledge sources are added to.
type KnowledgeBase struct{}

// KnowledgeBaseArgs are the inputs of a KnowledgeBase.
type KnowledgeBaseArgs struct {
	DisplayName string  `pulumi:"displayName"`
	Description *string `pulumi:"description,optional"`
}

func (a *KnowledgeBaseArgs) Annotate(an infer.Annotator) {
	an.Describe(&a.DisplayName, "Unique, human-readable name of the knowledge base within the account: letters, digits and hyphens, at most 64 characters. Updates in place.")
	an.Describe(&a.Description, "Description of the knowledge base, 1 to 128 characters. Twilio cannot clear a description, so removing it replaces the knowledge base.")
}

// KnowledgeBaseState is the persisted state of a KnowledgeBase.
type KnowledgeBaseState struct {
	KnowledgeBaseArgs

	Status    string `pulumi:"status"`
	Revision  int    `pulumi:"revision"`
	CreatedAt string `pulumi:"createdAt"`
	UpdatedAt string `pulumi:"updatedAt"`
}

func (s *KnowledgeBaseState) Annotate(a infer.Annotator) {
	a.Describe(&s.Status, "Provisioning status reported by Twilio: QUEUED, PROVISIONING, ACTIVE, FAILED or DELETING. The provider waits for ACTIVE.")
	a.Describe(&s.Revision, "Version number Twilio reports for the knowledge base; it increases on every successful update.")
	a.Describe(&s.CreatedAt, "When the knowledge base was created (ISO 8601).")
	a.Describe(&s.UpdatedAt, "When the knowledge base was last updated (ISO 8601).")
}

func (r *KnowledgeBase) Annotate(a infer.Annotator) {
	a.Describe(&r, "A Twilio Enterprise Knowledge base (Knowledge API v2). Twilio creates, updates and deletes it asynchronously; the provider follows the operation and waits until the knowledge base is ACTIVE (or gone). "+
		"Experimental: the API is new, needs Enterprise Knowledge enabled on the account, and has not been verified against a live account. The resource ID is the knowledge base ID (know_knowledgebase_...).")
}

var _ infer.CustomResource[KnowledgeBaseArgs, KnowledgeBaseState] = (*KnowledgeBase)(nil)
var _ infer.CustomCheck[KnowledgeBaseArgs] = (*KnowledgeBase)(nil)
var _ infer.CustomDiff[KnowledgeBaseArgs, KnowledgeBaseState] = (*KnowledgeBase)(nil)
var _ infer.CustomRead[KnowledgeBaseArgs, KnowledgeBaseState] = (*KnowledgeBase)(nil)
var _ infer.CustomUpdate[KnowledgeBaseArgs, KnowledgeBaseState] = (*KnowledgeBase)(nil)
var _ infer.CustomDelete[KnowledgeBaseState] = (*KnowledgeBase)(nil)

var knowledgeBaseName = regexp.MustCompile(`^[a-zA-Z0-9-]+$`)

func (r *KnowledgeBase) Check(ctx context.Context, req infer.CheckRequest) (infer.CheckResponse[KnowledgeBaseArgs], error) {
	args, failures, err := infer.DefaultCheck[KnowledgeBaseArgs](ctx, req.NewInputs)
	if err != nil || len(failures) > 0 {
		return infer.CheckResponse[KnowledgeBaseArgs]{Inputs: args, Failures: failures}, err
	}
	unk := unknowns(req.NewInputs)
	fail := func(prop, reason string) { failures = append(failures, p.CheckFailure{Property: prop, Reason: reason}) }
	if !unk("displayName") {
		if !knowledgeBaseName.MatchString(args.DisplayName) || len(args.DisplayName) > 64 {
			fail("displayName", "displayName must be 1 to 64 letters, digits or hyphens")
		}
	}
	if !unk("description") && args.Description != nil {
		if n := utf8.RuneCountInString(*args.Description); n < 1 || n > 128 {
			fail("description", "description must be 1 to 128 characters; omit it for none")
		}
	}
	return infer.CheckResponse[KnowledgeBaseArgs]{Inputs: args, Failures: failures}, nil
}

func (r *KnowledgeBase) Diff(_ context.Context, req infer.DiffRequest[KnowledgeBaseArgs, KnowledgeBaseState]) (infer.DiffResponse, error) {
	o, n := req.State.KnowledgeBaseArgs, req.Inputs
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

func (s *KnowledgeBaseState) fill(o obj) {
	s.Status = o.str("status")
	if v := o.optInt("version"); v != nil {
		s.Revision = *v
	}
	s.CreatedAt = o.str("createdAt")
	s.UpdatedAt = o.str("updatedAt")
}

// waitKnowledgeBaseActive polls the knowledge base until it is ACTIVE and returns it.
func waitKnowledgeBaseActive(ctx context.Context, c *Client, id string) (obj, error) {
	var out obj
	err := pollUntil(ctx, "knowledge base "+id, func(ctx context.Context) (bool, error) {
		o, err := c.do(ctx, "GET", "knowledge", knowledgeBasePath(id), nil, nil)
		if err != nil {
			return false, err
		}
		switch s := o.str("status"); s {
		case "ACTIVE":
			out = o
			return true, nil
		case "FAILED":
			return false, fmt.Errorf("knowledge base %s is FAILED", id)
		}
		return false, nil
	})
	return out, err
}

func (r *KnowledgeBase) Create(ctx context.Context, req infer.CreateRequest[KnowledgeBaseArgs]) (infer.CreateResponse[KnowledgeBaseState], error) {
	a := req.Inputs
	if req.DryRun {
		return infer.CreateResponse[KnowledgeBaseState]{Output: KnowledgeBaseState{KnowledgeBaseArgs: a}}, nil
	}
	c, err := clientFor(ctx)
	if err != nil {
		return infer.CreateResponse[KnowledgeBaseState]{}, err
	}
	body := map[string]any{"displayName": a.DisplayName}
	if a.Description != nil {
		body["description"] = *a.Description
	}
	acc, err := c.doJSON(ctx, "POST", "knowledge", knowledgeBasesPath, nil, body)
	if err != nil {
		return infer.CreateResponse[KnowledgeBaseState]{}, fmt.Errorf("create knowledge base: %w", err)
	}
	op, err := waitOperation(ctx, c, "knowledge", knowledgeOpsPath, acc, "create knowledge base "+a.DisplayName)
	if err != nil {
		return infer.CreateResponse[KnowledgeBaseState]{}, err
	}
	id := op.str("result.id")
	if id == "" {
		return infer.CreateResponse[KnowledgeBaseState]{}, fmt.Errorf("create knowledge base %s: the completed operation carries no resource ID", a.DisplayName)
	}
	o, err := waitKnowledgeBaseActive(ctx, c, id)
	if err != nil {
		// Do not leave a half-provisioned knowledge base behind that no state tracks.
		_, _ = c.do(context.WithoutCancel(ctx), "DELETE", "knowledge", knowledgeBasePath(id), nil, nil)
		return infer.CreateResponse[KnowledgeBaseState]{}, fmt.Errorf("create knowledge base %s: %w", a.DisplayName, err)
	}
	st := KnowledgeBaseState{KnowledgeBaseArgs: a}
	st.fill(o)
	return infer.CreateResponse[KnowledgeBaseState]{ID: id, Output: st}, nil
}

func (r *KnowledgeBase) Read(ctx context.Context, req infer.ReadRequest[KnowledgeBaseArgs, KnowledgeBaseState]) (infer.ReadResponse[KnowledgeBaseArgs, KnowledgeBaseState], error) {
	type resp = infer.ReadResponse[KnowledgeBaseArgs, KnowledgeBaseState]
	c, err := clientFor(ctx)
	if err != nil {
		return resp{}, err
	}
	o, err := c.do(ctx, "GET", "knowledge", knowledgeBasePath(req.ID), nil, nil)
	if err != nil {
		if isNotFound(err) {
			if err := notFoundOnImport("knowledge base", req.ID, req.Inputs, req.State); err != nil {
				return resp{}, err
			}
			return resp{}, nil
		}
		return resp{}, fmt.Errorf("read knowledge base %s: %w", req.ID, err)
	}
	if o.str("status") == "DELETING" {
		// Being deleted counts as gone, like a 404.
		if err := notFoundOnImport("knowledge base", req.ID, req.Inputs, req.State); err != nil {
			return resp{}, err
		}
		return resp{}, nil
	}
	in := req.Inputs
	in.DisplayName = o.str("displayName")
	in.Description = nonEmpty(o.optStr("description"))
	st := KnowledgeBaseState{KnowledgeBaseArgs: in}
	st.fill(o)
	return resp{ID: req.ID, Inputs: in, State: st}, nil
}

func (r *KnowledgeBase) Update(ctx context.Context, req infer.UpdateRequest[KnowledgeBaseArgs, KnowledgeBaseState]) (infer.UpdateResponse[KnowledgeBaseState], error) {
	st := req.State
	st.KnowledgeBaseArgs = req.Inputs
	if req.DryRun {
		return infer.UpdateResponse[KnowledgeBaseState]{Output: st}, nil
	}
	c, err := clientFor(ctx)
	if err != nil {
		return infer.UpdateResponse[KnowledgeBaseState]{}, err
	}
	body := map[string]any{}
	if req.Inputs.DisplayName != req.State.DisplayName {
		body["displayName"] = req.Inputs.DisplayName
	}
	if deref(req.Inputs.Description) != deref(req.State.Description) {
		body["description"] = deref(req.Inputs.Description)
	}
	if len(body) > 0 {
		acc, err := c.doJSON(ctx, "PATCH", "knowledge", knowledgeBasePath(req.ID), nil, body)
		if err != nil {
			return infer.UpdateResponse[KnowledgeBaseState]{}, fmt.Errorf("update knowledge base %s: %w", req.ID, err)
		}
		if acc.str("statusUrl") != "" {
			if _, err := waitOperation(ctx, c, "knowledge", knowledgeOpsPath, acc, "update knowledge base "+req.ID); err != nil {
				return infer.UpdateResponse[KnowledgeBaseState]{}, err
			}
		}
	}
	o, err := waitKnowledgeBaseActive(ctx, c, req.ID)
	if err != nil {
		return infer.UpdateResponse[KnowledgeBaseState]{}, fmt.Errorf("update knowledge base %s: %w", req.ID, err)
	}
	st.fill(o)
	return infer.UpdateResponse[KnowledgeBaseState]{Output: st}, nil
}

func (r *KnowledgeBase) Delete(ctx context.Context, req infer.DeleteRequest[KnowledgeBaseState]) (infer.DeleteResponse, error) {
	c, err := clientFor(ctx)
	if err != nil {
		return infer.DeleteResponse{}, err
	}
	acc, err := c.do(ctx, "DELETE", "knowledge", knowledgeBasePath(req.ID), nil, nil)
	if err != nil {
		if isNotFound(err) {
			return infer.DeleteResponse{}, nil
		}
		return infer.DeleteResponse{}, fmt.Errorf("delete knowledge base %s: %w", req.ID, err)
	}
	if acc.str("statusUrl") != "" {
		if _, err := waitOperation(ctx, c, "knowledge", knowledgeOpsPath, acc, "delete knowledge base "+req.ID); err != nil && !isNotFound(err) {
			return infer.DeleteResponse{}, err
		}
	}
	// The operation can finish before the knowledge base is gone; wait for the 404 so a dependent
	// resource or a re-create under the same name does not race the delete.
	err = pollUntil(ctx, "deleting knowledge base "+req.ID, func(ctx context.Context) (bool, error) {
		_, err := c.do(ctx, "GET", "knowledge", knowledgeBasePath(req.ID), nil, nil)
		if isNotFound(err) {
			return true, nil
		}
		return false, err
	})
	if err != nil {
		return infer.DeleteResponse{}, fmt.Errorf("delete knowledge base %s: %w", req.ID, err)
	}
	return infer.DeleteResponse{}, nil
}

// Knowledge manages one Enterprise Knowledge source (a crawled website or a block of text) in a KnowledgeBase.
type Knowledge struct{}

// KnowledgeArgs are the inputs of a Knowledge source.
type KnowledgeArgs struct {
	KnowledgeBaseId string  `pulumi:"knowledgeBaseId" provider:"replaceOnChanges"`
	Name            string  `pulumi:"name"`
	Description     *string `pulumi:"description,optional"`
	SourceType      string  `pulumi:"sourceType" provider:"replaceOnChanges"`
	WebUrl          *string `pulumi:"webUrl,optional" provider:"replaceOnChanges"`
	CrawlDepth      *int    `pulumi:"crawlDepth,optional"`
	CrawlPeriod     *string `pulumi:"crawlPeriod,optional"`
	TextContent     *string `pulumi:"textContent,optional" provider:"secret"`
	FilePath        *string `pulumi:"filePath,optional"`
}

func (a *KnowledgeArgs) Annotate(an infer.Annotator) {
	an.Describe(&a.KnowledgeBaseId, "ID (know_knowledgebase_...) of the knowledge base the source belongs to. Changing it replaces the source.")
	an.Describe(&a.Name, "Name of the source, 1 to 30 characters. Updates in place.")
	an.Describe(&a.Description, "What the source holds and when to use it, up to 1024 characters. Updates in place.")
	an.Describe(&a.SourceType, "Kind of source: Web (a crawled website), Text (raw text) or File (an uploaded file). Changing it replaces the source.")
	an.Describe(&a.WebUrl, "Web sources: the URL to crawl. Required for Web sources. Changing it replaces the source, because Twilio lists the URL among the immutable fields.")
	an.Describe(&a.CrawlDepth, "Web sources: how many links deep to crawl, 1 to 10. Twilio's default is 2. Updates in place.")
	an.Describe(&a.CrawlPeriod, "Web sources: how often to re-crawl: WEEKLY, BIWEEKLY, MONTHLY or NEVER. Twilio's default is NEVER. Updates in place.")
	an.Describe(&a.FilePath, "File sources: path of the file to upload (.csv, .md, .pdf, .tsv or .txt, 1 byte to 16 MiB). Required for File sources; a relative path resolves against the working directory of the Pulumi program. The provider uploads the file to the presigned URL Twilio returns and waits for processing. Changing the file's content or name replaces the source, because Twilio gives a source no way to swap its file; moving the same file to another path does not. After an import the content cannot be compared, so the file is assumed to match until the source is replaced.")
	an.Describe(&a.TextContent, "Text sources: the text, up to 185000 characters. Required for Text sources. Treated as a secret, since it can hold private business content. Updates in place and reprocesses the source. Twilio does not necessarily return it, so drift in it is not detected.")
}

// KnowledgeState is the persisted state of a Knowledge source.
type KnowledgeState struct {
	KnowledgeArgs

	// KnowledgeId is the source ID within the knowledge base.
	KnowledgeId string `pulumi:"knowledgeId"`
	// FileSha256 is the SHA-256 of the uploaded file; the Diff compares the file on disk with it.
	FileSha256 *string `pulumi:"fileSha256,optional"`
	Status     string  `pulumi:"status"`
	CreatedAt  string  `pulumi:"createdAt"`
	UpdatedAt  string  `pulumi:"updatedAt"`
}

func (s *KnowledgeState) Annotate(a infer.Annotator) {
	a.Describe(&s.KnowledgeId, "The source ID (know_knowledge_...). The resource ID is the knowledge base ID and this ID joined by a slash.")
	a.Describe(&s.FileSha256, "File sources: hex SHA-256 of the file content that was uploaded. A different hash on disk replaces the source. Unset for imported sources.")
	a.Describe(&s.Status, "Processing status reported by Twilio: SCHEDULED, QUEUED, PROCESSING, COMPLETED or FAILED. The provider waits for COMPLETED.")
	a.Describe(&s.CreatedAt, "When the source was created (ISO 8601).")
	a.Describe(&s.UpdatedAt, "When the source was last updated (ISO 8601).")
}

func (r *Knowledge) Annotate(a infer.Annotator) {
	a.Describe(&r, "A source of knowledge (a crawled website, raw text or an uploaded file) in a Twilio Enterprise Knowledge base (Knowledge API v2). "+
		"Create waits until Twilio has processed the source (COMPLETED) and fails, removing the source, if processing FAILS. "+
		"Experimental: the API is new, needs Enterprise Knowledge enabled on the account, and has not been verified against a live account. "+
		"Import with the ID <knowledgeBaseId>/<knowledgeId>.")
}

var _ infer.CustomResource[KnowledgeArgs, KnowledgeState] = (*Knowledge)(nil)
var _ infer.CustomCheck[KnowledgeArgs] = (*Knowledge)(nil)
var _ infer.CustomDiff[KnowledgeArgs, KnowledgeState] = (*Knowledge)(nil)
var _ infer.CustomRead[KnowledgeArgs, KnowledgeState] = (*Knowledge)(nil)
var _ infer.CustomUpdate[KnowledgeArgs, KnowledgeState] = (*Knowledge)(nil)
var _ infer.CustomDelete[KnowledgeState] = (*Knowledge)(nil)

const (
	knowledgeDefaultCrawlDepth  = 2
	knowledgeDefaultCrawlPeriod = "NEVER"
)

var knowledgeCrawlPeriods = map[string]bool{"WEEKLY": true, "BIWEEKLY": true, "MONTHLY": true, "NEVER": true}

func (r *Knowledge) Check(ctx context.Context, req infer.CheckRequest) (infer.CheckResponse[KnowledgeArgs], error) {
	args, failures, err := infer.DefaultCheck[KnowledgeArgs](ctx, req.NewInputs)
	if err != nil || len(failures) > 0 {
		return infer.CheckResponse[KnowledgeArgs]{Inputs: args, Failures: failures}, err
	}
	unk := unknowns(req.NewInputs)
	fail := func(prop, reason string) { failures = append(failures, p.CheckFailure{Property: prop, Reason: reason}) }
	if !unk("name") {
		if n := utf8.RuneCountInString(args.Name); n < 1 || n > 30 {
			fail("name", "name must be 1 to 30 characters")
		}
	}
	if !unk("description") && args.Description != nil && utf8.RuneCountInString(*args.Description) > 1024 {
		fail("description", "description must be at most 1024 characters")
	}
	if unk("sourceType") {
		return infer.CheckResponse[KnowledgeArgs]{Inputs: args, Failures: failures}, nil
	}
	fields := []struct {
		name, sourceType string
		set, required    bool
	}{
		{"webUrl", "Web", args.WebUrl != nil, true},
		{"crawlDepth", "Web", args.CrawlDepth != nil, false},
		{"crawlPeriod", "Web", args.CrawlPeriod != nil, false},
		{"textContent", "Text", args.TextContent != nil, true},
		{"filePath", "File", args.FilePath != nil, true},
	}
	switch args.SourceType {
	case "Web", "Text":
		for _, f := range fields {
			switch {
			case f.sourceType != args.SourceType && f.set:
				fail(f.name, fmt.Sprintf("%s applies to %s sources, not %s", f.name, f.sourceType, args.SourceType))
			case f.sourceType == args.SourceType && f.required && !f.set && !unk(f.name):
				fail(f.name, fmt.Sprintf("%s is required for %s sources", f.name, f.sourceType))
			}
		}
	case "File":
		if args.WebUrl != nil || args.CrawlDepth != nil || args.CrawlPeriod != nil || args.TextContent != nil {
			fail("sourceType", "webUrl, crawlDepth, crawlPeriod and textContent do not apply to File sources")
		}
		switch {
		case args.FilePath == nil && !unk("filePath"):
			fail("filePath", "filePath is required for File sources")
		case args.FilePath != nil && !unk("filePath"):
			if _, err := inspectKnowledgeFile(*args.FilePath); err != nil {
				fail("filePath", err.Error())
			}
		}
		return infer.CheckResponse[KnowledgeArgs]{Inputs: args, Failures: failures}, nil
	default:
		fail("sourceType", "sourceType must be Web, Text or File")
		return infer.CheckResponse[KnowledgeArgs]{Inputs: args, Failures: failures}, nil
	}
	if !unk("webUrl") && args.WebUrl != nil {
		if u, err := url.Parse(*args.WebUrl); err != nil || (u.Scheme != "http" && u.Scheme != "https") || u.Host == "" {
			fail("webUrl", "webUrl must be an http or https URL")
		}
	}
	if !unk("crawlDepth") && args.CrawlDepth != nil && (*args.CrawlDepth < 1 || *args.CrawlDepth > 10) {
		fail("crawlDepth", "crawlDepth must be between 1 and 10")
	}
	if !unk("crawlPeriod") && args.CrawlPeriod != nil && !knowledgeCrawlPeriods[*args.CrawlPeriod] {
		fail("crawlPeriod", "crawlPeriod must be one of WEEKLY, BIWEEKLY, MONTHLY or NEVER")
	}
	if !unk("textContent") && args.TextContent != nil {
		if n := utf8.RuneCountInString(*args.TextContent); n == 0 || n > 185000 {
			fail("textContent", "textContent must be 1 to 185000 characters")
		}
	}
	return infer.CheckResponse[KnowledgeArgs]{Inputs: args, Failures: failures}, nil
}

func (a KnowledgeArgs) depth() int {
	if a.CrawlDepth == nil {
		return knowledgeDefaultCrawlDepth
	}
	return *a.CrawlDepth
}

func (a KnowledgeArgs) period() string { return derefOr(a.CrawlPeriod, knowledgeDefaultCrawlPeriod) }

// sourceChanged reports whether the source settings differ (the default crawl settings count as unset).
func (a KnowledgeArgs) sourceChanged(b KnowledgeArgs) bool {
	return a.depth() != b.depth() || a.period() != b.period() || deref(a.TextContent) != deref(b.TextContent)
}

func (r *Knowledge) Diff(_ context.Context, req infer.DiffRequest[KnowledgeArgs, KnowledgeState]) (infer.DiffResponse, error) {
	o, n := req.State.KnowledgeArgs, req.Inputs
	detail := map[string]p.PropertyDiff{}
	upd := func(prop string, changed bool) {
		if changed {
			detail[prop] = p.PropertyDiff{Kind: p.Update}
		}
	}
	rep := func(prop string, changed bool) {
		if changed {
			detail[prop] = p.PropertyDiff{Kind: p.UpdateReplace}
		}
	}
	rep("knowledgeBaseId", o.KnowledgeBaseId != n.KnowledgeBaseId)
	rep("sourceType", o.SourceType != n.SourceType)
	rep("webUrl", deref(o.WebUrl) != deref(n.WebUrl))
	upd("name", o.Name != n.Name)
	upd("description", deref(o.Description) != deref(n.Description))
	upd("crawlDepth", o.depth() != n.depth())
	upd("crawlPeriod", o.period() != n.period())
	// An import cannot read the text back, so a program that sets it is not a change.
	upd("textContent", o.TextContent != nil && n.TextContent != nil && *o.TextContent != *n.TextContent)
	if n.FilePath != nil && n.SourceType == "File" {
		replace := false
		if req.State.FileSha256 != nil {
			fi, err := inspectKnowledgeFile(*n.FilePath)
			if err != nil {
				return infer.DiffResponse{}, fmt.Errorf("filePath: %w", err)
			}
			replace = fi.sha256 != *req.State.FileSha256 || (o.FilePath != nil && filepath.Base(*o.FilePath) != fi.name)
		}
		switch {
		case replace:
			rep("filePath", true)
		default:
			// The same content under another directory only moves the pointer.
			upd("filePath", o.FilePath != nil && *o.FilePath != *n.FilePath)
		}
	}
	return infer.DiffResponse{HasChanges: len(detail) > 0, DetailedDiff: detail}, nil
}

// knowledgeMaxFileSize is Twilio's 16 MiB limit for a Knowledge file.
const knowledgeMaxFileSize = 16 * 1024 * 1024

var knowledgeFileTypes = map[string]string{
	".csv": "text/csv", ".md": "text/markdown", ".pdf": "application/pdf",
	".tsv": "text/tab-separated-values", ".txt": "text/plain",
}

// knowledgeFile describes a local file to upload as a Knowledge source.
type knowledgeFile struct {
	name, mimeType, sha256 string
	size                   int
}

// inspectKnowledgeFile validates a file against Twilio's limits (supported extension, 1 byte to 16 MiB,
// name at most 256 characters) and reads its size and SHA-256.
func inspectKnowledgeFile(path string) (knowledgeFile, error) {
	var fi knowledgeFile
	fi.name = filepath.Base(path)
	mt, ok := knowledgeFileTypes[strings.ToLower(filepath.Ext(path))]
	if !ok {
		return fi, fmt.Errorf("%q is not a supported file type: use .csv, .md, .pdf, .tsv or .txt", fi.name)
	}
	if utf8.RuneCountInString(fi.name) > 256 {
		return fi, fmt.Errorf("the file name %q is longer than 256 characters", fi.name)
	}
	st, err := os.Stat(path)
	if err != nil {
		return fi, fmt.Errorf("cannot read %q: %v", path, err)
	}
	if !st.Mode().IsRegular() {
		return fi, fmt.Errorf("%q is not a regular file", path)
	}
	if st.Size() < 1 || st.Size() > knowledgeMaxFileSize {
		return fi, fmt.Errorf("%q must be between 1 byte and 16 MiB, it is %d bytes", path, st.Size())
	}
	f, err := os.Open(path)
	if err != nil {
		return fi, fmt.Errorf("cannot read %q: %v", path, err)
	}
	defer f.Close()
	h := sha256.New()
	n, err := io.Copy(h, f)
	if err != nil {
		return fi, fmt.Errorf("cannot read %q: %v", path, err)
	}
	fi.mimeType, fi.size, fi.sha256 = mt, int(n), hex.EncodeToString(h.Sum(nil))
	return fi, nil
}

// putPresigned uploads data to the presigned URL Twilio handed out. The request deliberately carries no
// Twilio credentials: the URL is signed and belongs to another host. Network errors and 5xx answers
// are retried, since the PUT is idempotent.
func (c *Client) putPresigned(ctx context.Context, rawURL, contentType string, data []byte) error {
	u, err := url.Parse(rawURL)
	if err != nil || u.Host == "" {
		return fmt.Errorf("Twilio returned an unusable upload URL")
	}
	if u.Scheme != "https" && !(c.baseURL != "" && strings.HasPrefix(rawURL, c.baseURL+"/")) {
		return fmt.Errorf("refusing to upload to a non-HTTPS URL (%s)", u.Scheme)
	}
	var lastErr error
	for attempt := 0; attempt < 3; attempt++ {
		if attempt > 0 {
			select {
			case <-ctx.Done():
				return ctx.Err()
			case <-time.After(retryBaseDelay << (attempt - 1)):
			}
		}
		req, err := http.NewRequestWithContext(ctx, http.MethodPut, rawURL, bytes.NewReader(data))
		if err != nil {
			return err
		}
		req.Header.Set("Content-Type", contentType)
		resp, err := c.http.Do(req)
		if err != nil {
			lastErr = err
			continue
		}
		raw, _ := io.ReadAll(io.LimitReader(resp.Body, 4096))
		_ = resp.Body.Close()
		if resp.StatusCode >= 200 && resp.StatusCode < 300 {
			return nil
		}
		lastErr = fmt.Errorf("upload answered HTTP %d: %s", resp.StatusCode, strings.TrimSpace(string(raw)))
		if resp.StatusCode < 500 {
			return lastErr
		}
	}
	return lastErr
}

// source builds the request's source object.
func (a KnowledgeArgs) source() map[string]any {
	if a.SourceType == "Text" {
		return map[string]any{"type": "Text", "content": deref(a.TextContent)}
	}
	if a.SourceType == "File" {
		// Check has validated the file; a failure here would surface in Create.
		fi, _ := inspectKnowledgeFile(deref(a.FilePath))
		return map[string]any{"type": "File", "fileName": fi.name, "fileSize": fi.size, "mimeType": fi.mimeType}
	}
	return map[string]any{"type": "Web", "url": deref(a.WebUrl), "crawlDepth": a.depth(), "crawlPeriod": a.period()}
}

func (s *KnowledgeState) fill(o obj) {
	s.Status = o.str("status")
	s.CreatedAt = o.str("createdAt")
	s.UpdatedAt = o.str("updatedAt")
}

// waitKnowledgeCompleted polls the source until Twilio has processed it.
func waitKnowledgeCompleted(ctx context.Context, c *Client, kb, id string) (obj, error) {
	var out obj
	err := pollUntil(ctx, "knowledge source "+id, func(ctx context.Context) (bool, error) {
		o, err := c.do(ctx, "GET", "knowledge", knowledgeItemPath(kb, id), nil, nil)
		if err != nil {
			return false, err
		}
		switch s := o.str("status"); s {
		case "COMPLETED":
			out = o
			return true, nil
		case "FAILED":
			return false, fmt.Errorf("Twilio failed to process knowledge source %s (status FAILED); check the source's URL or content", id)
		}
		return false, nil
	})
	return out, err
}

func (r *Knowledge) Create(ctx context.Context, req infer.CreateRequest[KnowledgeArgs]) (infer.CreateResponse[KnowledgeState], error) {
	a := req.Inputs
	if req.DryRun {
		return infer.CreateResponse[KnowledgeState]{Output: KnowledgeState{KnowledgeArgs: a}}, nil
	}
	c, err := clientFor(ctx)
	if err != nil {
		return infer.CreateResponse[KnowledgeState]{}, err
	}
	var file knowledgeFile
	var content []byte
	if a.SourceType == "File" {
		if file, err = inspectKnowledgeFile(deref(a.FilePath)); err != nil {
			return infer.CreateResponse[KnowledgeState]{}, fmt.Errorf("create knowledge source %s: %w", a.Name, err)
		}
		if content, err = os.ReadFile(*a.FilePath); err != nil {
			return infer.CreateResponse[KnowledgeState]{}, fmt.Errorf("create knowledge source %s: %w", a.Name, err)
		}
		// The file may have changed between hashing and reading; upload what was hashed.
		if sum := sha256.Sum256(content); hex.EncodeToString(sum[:]) != file.sha256 {
			return infer.CreateResponse[KnowledgeState]{}, fmt.Errorf("create knowledge source %s: %q changed while it was being read; run again", a.Name, *a.FilePath)
		}
	}
	body := map[string]any{"name": a.Name, "source": a.source()}
	if a.Description != nil {
		body["description"] = *a.Description
	}
	o, err := c.doJSON(ctx, "POST", "knowledge", knowledgeCollectionPath(a.KnowledgeBaseId), nil, body)
	if err != nil {
		return infer.CreateResponse[KnowledgeState]{}, fmt.Errorf("create knowledge source %s: %w", a.Name, err)
	}
	kid := o.str("id")
	if kid == "" {
		return infer.CreateResponse[KnowledgeState]{}, fmt.Errorf("create knowledge source %s: Twilio returned no ID", a.Name)
	}
	if a.SourceType == "File" {
		if err := uploadKnowledgeFile(ctx, c, a.KnowledgeBaseId, kid, o, file, content); err != nil {
			_, _ = c.do(context.WithoutCancel(ctx), "DELETE", "knowledge", knowledgeItemPath(a.KnowledgeBaseId, kid), nil, nil)
			return infer.CreateResponse[KnowledgeState]{}, fmt.Errorf("create knowledge source %s: %w", a.Name, err)
		}
		o["status"] = "QUEUED"
	}
	if o.str("status") != "COMPLETED" {
		if o, err = waitKnowledgeCompleted(ctx, c, a.KnowledgeBaseId, kid); err != nil {
			_, _ = c.do(context.WithoutCancel(ctx), "DELETE", "knowledge", knowledgeItemPath(a.KnowledgeBaseId, kid), nil, nil)
			return infer.CreateResponse[KnowledgeState]{}, fmt.Errorf("create knowledge source %s: %w", a.Name, err)
		}
	}
	st := KnowledgeState{KnowledgeArgs: a, KnowledgeId: kid}
	if a.SourceType == "File" {
		st.FileSha256 = &file.sha256
	}
	st.fill(o)
	return infer.CreateResponse[KnowledgeState]{ID: a.KnowledgeBaseId + "/" + kid, Output: st}, nil
}

// uploadKnowledgeFile sends the file to the presigned importUrl of a new File source (read from the create
// answer, or from the source when the create answer omits it).
func uploadKnowledgeFile(ctx context.Context, c *Client, kb, kid string, created obj, file knowledgeFile, content []byte) error {
	importURL := created.str("source.importUrl")
	if importURL == "" {
		got, err := c.do(ctx, "GET", "knowledge", knowledgeItemPath(kb, kid), nil, nil)
		if err != nil {
			return err
		}
		if importURL = got.str("source.importUrl"); importURL == "" {
			return fmt.Errorf("Twilio returned no upload URL for the file source (status %s)", got.str("status"))
		}
	}
	if err := c.putPresigned(ctx, importURL, file.mimeType, content); err != nil {
		return fmt.Errorf("upload %s: %w", file.name, err)
	}
	return nil
}

func splitKnowledgeID(id string) (kb, kn string, err error) {
	kb, kn, ok := strings.Cut(id, "/")
	if !ok || kb == "" || kn == "" || strings.Contains(kn, "/") {
		return "", "", fmt.Errorf("invalid knowledge source ID %q: expected <knowledgeBaseId>/<knowledgeId>", id)
	}
	return kb, kn, nil
}

func (r *Knowledge) Read(ctx context.Context, req infer.ReadRequest[KnowledgeArgs, KnowledgeState]) (infer.ReadResponse[KnowledgeArgs, KnowledgeState], error) {
	type resp = infer.ReadResponse[KnowledgeArgs, KnowledgeState]
	kb, kn, err := splitKnowledgeID(req.ID)
	if err != nil {
		return resp{}, err
	}
	c, err := clientFor(ctx)
	if err != nil {
		return resp{}, err
	}
	o, err := c.do(ctx, "GET", "knowledge", knowledgeItemPath(kb, kn), nil, nil)
	if err != nil {
		if isNotFound(err) {
			if err := notFoundOnImport("knowledge source", req.ID, req.Inputs, req.State); err != nil {
				return resp{}, err
			}
			return resp{}, nil
		}
		return resp{}, fmt.Errorf("read knowledge source %s: %w", req.ID, err)
	}
	in := req.Inputs
	imported := in.SourceType == ""
	in.KnowledgeBaseId = kb
	in.Name = o.str("name")
	in.Description = nonEmpty(o.optStr("description"))
	if imported {
		in.SourceType = o.str("source.type")
	}
	switch o.str("source.type") {
	case "Web":
		if v := nonEmpty(o.optStr("source.url")); v != nil {
			in.WebUrl = v
		}
		if d := o.optInt("source.crawlDepth"); d != nil {
			if imported && *d != knowledgeDefaultCrawlDepth || !imported && in.CrawlDepth != nil {
				in.CrawlDepth = d
			}
		}
		if pd := nonEmpty(o.optStr("source.crawlPeriod")); pd != nil {
			if imported && *pd != knowledgeDefaultCrawlPeriod || !imported && in.CrawlPeriod != nil {
				in.CrawlPeriod = pd
			}
		}
	case "Text":
		if imported {
			in.TextContent = nonEmpty(o.optStr("source.content"))
		}
	}
	st := KnowledgeState{KnowledgeArgs: in, KnowledgeId: kn, FileSha256: req.State.FileSha256}
	st.fill(o)
	return resp{ID: req.ID, Inputs: in, State: st}, nil
}

func (r *Knowledge) Update(ctx context.Context, req infer.UpdateRequest[KnowledgeArgs, KnowledgeState]) (infer.UpdateResponse[KnowledgeState], error) {
	st := req.State
	st.KnowledgeArgs = req.Inputs
	if req.DryRun {
		return infer.UpdateResponse[KnowledgeState]{Output: st}, nil
	}
	if req.Inputs.SourceType == "File" && st.FileSha256 == nil && req.Inputs.FilePath != nil {
		// An imported source: take the file the program names as the baseline for later changes.
		if fi, err := inspectKnowledgeFile(*req.Inputs.FilePath); err == nil {
			st.FileSha256 = &fi.sha256
		}
	}
	kb, kn, err := splitKnowledgeID(req.ID)
	if err != nil {
		return infer.UpdateResponse[KnowledgeState]{}, err
	}
	c, err := clientFor(ctx)
	if err != nil {
		return infer.UpdateResponse[KnowledgeState]{}, err
	}
	// Twilio requires the name on every update.
	body := map[string]any{"name": req.Inputs.Name}
	if deref(req.Inputs.Description) != deref(req.State.Description) {
		body["description"] = deref(req.Inputs.Description)
	}
	reprocess := req.Inputs.sourceChanged(req.State.KnowledgeArgs)
	if reprocess {
		body["source"] = req.Inputs.source()
	}
	if _, err := c.doJSON(ctx, "PATCH", "knowledge", knowledgeItemPath(kb, kn), nil, body); err != nil {
		return infer.UpdateResponse[KnowledgeState]{}, fmt.Errorf("update knowledge source %s: %w", req.ID, err)
	}
	o, err := waitKnowledgeCompleted(ctx, c, kb, kn)
	if err != nil {
		return infer.UpdateResponse[KnowledgeState]{}, fmt.Errorf("update knowledge source %s: %w", req.ID, err)
	}
	st.fill(o)
	return infer.UpdateResponse[KnowledgeState]{Output: st}, nil
}

func (r *Knowledge) Delete(ctx context.Context, req infer.DeleteRequest[KnowledgeState]) (infer.DeleteResponse, error) {
	kb, kn, err := splitKnowledgeID(req.ID)
	if err != nil {
		return infer.DeleteResponse{}, err
	}
	c, err := clientFor(ctx)
	if err != nil {
		return infer.DeleteResponse{}, err
	}
	if _, err := c.do(ctx, "DELETE", "knowledge", knowledgeItemPath(kb, kn), nil, nil); err != nil && !isNotFound(err) {
		return infer.DeleteResponse{}, fmt.Errorf("delete knowledge source %s: %w", req.ID, err)
	}
	return infer.DeleteResponse{}, nil
}
