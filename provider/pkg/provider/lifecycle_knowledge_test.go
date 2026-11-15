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
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	pgp "github.com/pulumi/pulumi-go-provider"
	"github.com/pulumi/pulumi/sdk/v3/go/property"
	"github.com/stretchr/testify/require"
)

func fastKnowledgePolls(t *testing.T) {
	t.Helper()
	oi, ot := knowledgePollInterval, knowledgePollTimeout
	knowledgePollInterval, knowledgePollTimeout = time.Millisecond, 5*time.Second
	t.Cleanup(func() { knowledgePollInterval, knowledgePollTimeout, knowledgeFailOps = oi, ot, "" })
}

func kbInputs() property.Map {
	return pv(map[string]property.Value{"displayName": sv("product-docs"), "description": sv("Docs for support")})
}

func TestKnowledgeBaseLC_Lifecycle(t *testing.T) {
	fastKnowledgePolls(t)
	l := newTL(t, "KnowledgeBase", "kb")
	in := kbInputs()
	c := l.create(in)
	id := c.ID
	require.True(t, strings.HasPrefix(id, "know_knowledgebase_"), id)
	require.Equal(t, "ACTIVE", getS(t, c.Properties, "status"), "create waits for the operation and for ACTIVE")
	require.Len(t, l.f.callsMatching("GET", "/knowledge/v2/ControlPlane/Operations/"), 2, "the operation is polled until COMPLETED")
	require.JSONEq(t, `{"displayName":"product-docs","description":"Docs for support"}`, l.f.callsMatching("POST", "/knowledge/v2/ControlPlane/KnowledgeBases")[0].Body)
	require.Equal(t, 1.0, c.Properties.Get("revision").AsNumber())
	requireNoChanges(t, l.diff(id, c.Properties, in, in))

	// A refresh changes nothing; an out-of-band description change is drift on the input.
	rd := l.read(id, c.Properties, in)
	requireNoChanges(t, l.diff(id, rd.Properties, in, rd.Inputs))
	l.f.mutate("kbase", testAccount, id, func(o map[string]any) { o["description"] = "edited in console" })
	rd = l.read(id, c.Properties, in)
	require.Equal(t, "edited in console", getS(t, rd.Inputs, "description"))
	l.f.mutate("kbase", testAccount, id, func(o map[string]any) { o["description"] = "Docs for support" })

	// Name and description update in place, through one PATCH and a wait for the operation.
	next := with(in, map[string]property.Value{"displayName": sv("product-docs-v2"), "description": sv("Docs v2")})
	d := l.diff(id, c.Properties, next, in)
	require.Equal(t, pgp.Update, d.DetailedDiff["displayName"].Kind)
	require.Equal(t, pgp.Update, d.DetailedDiff["description"].Kind)
	l.f.resetCalls()
	u := l.update(id, c.Properties, next, in)
	require.Len(t, l.f.callsMatching("PATCH", "/knowledge/v2/ControlPlane/KnowledgeBases/"+id), 1)
	require.JSONEq(t, `{"displayName":"product-docs-v2","description":"Docs v2"}`, l.f.callsMatching("PATCH", "/knowledge/v2/ControlPlane/KnowledgeBases/"+id)[0].Body)
	require.Equal(t, "ACTIVE", getS(t, u.Properties, "status"))
	require.Equal(t, 2.0, u.Properties.Get("revision").AsNumber())
	requireNoChanges(t, l.diff(id, u.Properties, next, next))

	// Twilio cannot clear a description, so dropping it replaces.
	d = l.diff(id, u.Properties, without(next, "description"), next)
	require.Equal(t, pgp.UpdateReplace, d.DetailedDiff["description"].Kind)

	// Delete follows the operation and waits until the knowledge base is gone; deleting again is fine.
	l.f.resetCalls()
	l.del(id, u.Properties)
	require.Nil(t, l.f.get("kbase", testAccount, id))
	require.NotEmpty(t, l.f.callsMatching("DELETE", "/knowledge/v2/ControlPlane/KnowledgeBases/"+id))
	l.del(id, u.Properties)
	require.Empty(t, l.read(id, u.Properties, next).ID, "gone outside Pulumi")
}

func TestKnowledgeBaseLC_ImportAndChecks(t *testing.T) {
	fastKnowledgePolls(t)
	l := newTL(t, "KnowledgeBase", "kb")
	in := without(kbInputs(), "description")
	c := l.create(in)
	imp := l.read(c.ID, empty(), empty())
	require.Equal(t, c.ID, imp.ID)
	require.Equal(t, "product-docs", getS(t, imp.Inputs, "displayName"))
	require.False(t, imp.Inputs.Get("description").IsString(), "no description to adopt")
	requireNoChanges(t, l.diff(c.ID, imp.Properties, in, imp.Inputs))
	l.requireImportFails("know_knowledgebase_missing")

	for name, bad := range map[string]property.Map{
		"displayName": with(in, map[string]property.Value{"displayName": sv("has spaces")}),
		"description": with(in, map[string]property.Value{"description": sv("")}),
	} {
		f := l.check(bad).Failures
		require.NotEmpty(t, f, name)
		require.Equal(t, name, f[0].Property)
	}
	require.NotEmpty(t, l.check(with(in, map[string]property.Value{"displayName": sv(strings.Repeat("a", 65))})).Failures)
}

func TestKnowledgeBaseLC_FailedOperationLeavesNothing(t *testing.T) {
	fastKnowledgePolls(t)
	l := newTL(t, "KnowledgeBase", "kb")
	knowledgeFailOps = "quota exceeded"
	_, err := l.s.Create(pgp.CreateRequest{Urn: l.urn, Properties: l.checked(kbInputs())})
	require.ErrorContains(t, err, "quota exceeded")
}

func TestKnowledgeBaseLC_TimeoutIsReported(t *testing.T) {
	fastKnowledgePolls(t)
	knowledgePollTimeout = 20 * time.Millisecond
	knowledgePollInterval = 10 * time.Millisecond
	l := newTL(t, "KnowledgeBase", "kb")
	l.f.failNext("GET", "/knowledge/v2/ControlPlane/Operations/", 404)
	_, err := l.s.Create(pgp.CreateRequest{Urn: l.urn, Properties: l.checked(kbInputs())})
	require.Error(t, err)
}

const kbID = "know_knowledgebase_0001"

func textKnowledgeInputs() property.Map {
	return pv(map[string]property.Value{"knowledgeBaseId": sv(kbID), "name": sv("faq"), "description": sv("Common questions"),
		"sourceType": sv("Text"), "textContent": sv("Q: hours? A: 9 to 5.")})
}

func webKnowledgeInputs() property.Map {
	return pv(map[string]property.Value{"knowledgeBaseId": sv(kbID), "name": sv("docs-site"), "sourceType": sv("Web"), "webUrl": sv("https://docs.example.com")})
}

func seedKnowledgeBase(l *tl) {
	l.f.put("kbase", testAccount, kbID, map[string]any{"id": kbID, "displayName": "seed", "status": "ACTIVE", "version": 1.0})
}

func TestKnowledgeLC_TextLifecycle(t *testing.T) {
	fastKnowledgePolls(t)
	l := newTL(t, "Knowledge", "k")
	seedKnowledgeBase(l)
	in := textKnowledgeInputs()
	c := l.create(in)
	kid := getS(t, c.Properties, "knowledgeId")
	require.True(t, strings.HasPrefix(kid, "know_knowledge_"), kid)
	require.Equal(t, kbID+"/"+kid, c.ID)
	require.Equal(t, "COMPLETED", getS(t, c.Properties, "status"))
	require.True(t, c.Properties.Get("textContent").Secret(), "the text is a secret in the state")
	require.JSONEq(t, `{"name":"faq","description":"Common questions","source":{"type":"Text","content":"Q: hours? A: 9 to 5."}}`,
		l.f.callsMatching("POST", "/knowledge/v2/KnowledgeBases/"+kbID+"/Knowledge")[0].Body)
	requireNoChanges(t, l.diff(c.ID, c.Properties, in, in))

	// Twilio does not echo the text, so a refresh keeps the program's text and plans nothing.
	rd := l.read(c.ID, c.Properties, in)
	require.Equal(t, "Q: hours? A: 9 to 5.", getS(t, rd.Inputs, "textContent"))
	requireNoChanges(t, l.diff(c.ID, rd.Properties, in, rd.Inputs))
	l.f.mutate("kitem", testAccount, kid, func(o map[string]any) { o["name"] = "renamed" })
	require.Equal(t, "renamed", getS(t, l.read(c.ID, c.Properties, in).Inputs, "name"))
	l.f.mutate("kitem", testAccount, kid, func(o map[string]any) { o["name"] = "faq" })

	// A rename sends the name only (no reprocessing); new text sends the source too.
	next := with(in, map[string]property.Value{"name": sv("faq-2")})
	require.Equal(t, pgp.Update, l.diff(c.ID, c.Properties, next, in).DetailedDiff["name"].Kind)
	l.f.resetCalls()
	u := l.update(c.ID, c.Properties, next, in)
	require.JSONEq(t, `{"name":"faq-2"}`, l.f.callsMatching("PATCH", "/knowledge/v2/KnowledgeBases/"+kbID+"/Knowledge/"+kid)[0].Body)
	next2 := with(next, map[string]property.Value{"textContent": sv("new text"), "description": sv("")})
	require.Equal(t, pgp.Update, l.diff(c.ID, u.Properties, next2, next).DetailedDiff["textContent"].Kind)
	l.f.resetCalls()
	u = l.update(c.ID, u.Properties, next2, next)
	require.JSONEq(t, `{"name":"faq-2","description":"","source":{"type":"Text","content":"new text"}}`,
		l.f.callsMatching("PATCH", "/knowledge/v2/KnowledgeBases/"+kbID+"/Knowledge/"+kid)[0].Body)
	require.Equal(t, "COMPLETED", getS(t, u.Properties, "status"), "update waits for reprocessing")

	// The base, the type and (for Web) the URL replace the source.
	for k, v := range map[string]property.Value{"knowledgeBaseId": sv("know_knowledgebase_0002"), "sourceType": sv("Web")} {
		require.Equal(t, pgp.UpdateReplace, l.diff(c.ID, u.Properties, with(next2, map[string]property.Value{k: v}), next2).DetailedDiff[k].Kind, k)
	}

	l.del(c.ID, u.Properties)
	require.Nil(t, l.f.get("kitem", testAccount, kid))
	l.del(c.ID, u.Properties)
	require.Empty(t, l.read(c.ID, u.Properties, next2).ID)
}

func TestKnowledgeLC_WebDefaultsAndImport(t *testing.T) {
	fastKnowledgePolls(t)
	l := newTL(t, "Knowledge", "k")
	seedKnowledgeBase(l)
	in := webKnowledgeInputs()
	c := l.create(in)
	require.JSONEq(t, `{"name":"docs-site","source":{"type":"Web","url":"https://docs.example.com","crawlDepth":2,"crawlPeriod":"NEVER"}}`,
		l.f.callsMatching("POST", "/knowledge/v2/KnowledgeBases/"+kbID+"/Knowledge")[0].Body)
	requireNoChanges(t, l.diff(c.ID, c.Properties, in, in))
	// A program that writes the provider defaults plans no change, and neither does a refresh.
	explicit := with(in, map[string]property.Value{"crawlDepth": nv(2), "crawlPeriod": sv("NEVER")})
	requireNoChanges(t, l.diff(c.ID, c.Properties, explicit, in))
	rd := l.read(c.ID, c.Properties, in)
	requireNoChanges(t, l.diff(c.ID, rd.Properties, in, rd.Inputs))
	requireNoChanges(t, l.diff(c.ID, rd.Properties, explicit, rd.Inputs))

	custom := with(in, map[string]property.Value{"crawlDepth": nv(4), "crawlPeriod": sv("WEEKLY")})
	d := l.diff(c.ID, c.Properties, custom, in)
	require.Equal(t, pgp.Update, d.DetailedDiff["crawlDepth"].Kind)
	require.Equal(t, pgp.Update, d.DetailedDiff["crawlPeriod"].Kind)
	u := l.update(c.ID, c.Properties, custom, in)
	require.JSONEq(t, `{"name":"docs-site","source":{"type":"Web","url":"https://docs.example.com","crawlDepth":4,"crawlPeriod":"WEEKLY"}}`,
		l.f.callsMatching("PATCH", "/knowledge/v2/KnowledgeBases/"+kbID+"/Knowledge/")[0].Body)
	require.Equal(t, pgp.UpdateReplace, l.diff(c.ID, u.Properties, with(custom, map[string]property.Value{"webUrl": sv("https://other.example.com")}), custom).DetailedDiff["webUrl"].Kind)

	// Import adopts the crawl settings that differ from the defaults and the URL.
	imp := l.read(c.ID, empty(), empty())
	require.Equal(t, "Web", getS(t, imp.Inputs, "sourceType"))
	require.Equal(t, "https://docs.example.com", getS(t, imp.Inputs, "webUrl"))
	require.Equal(t, 4.0, imp.Inputs.Get("crawlDepth").AsNumber())
	requireNoChanges(t, l.diff(c.ID, imp.Properties, custom, imp.Inputs))
	l.requireImportFails(kbID + "/know_knowledge_missing")
	_, err := l.s.Read(pgp.ReadRequest{ID: "no-slash", Urn: l.urn, Properties: empty(), Inputs: empty()})
	require.ErrorContains(t, err, "expected <knowledgeBaseId>/<knowledgeId>")

	// Importing a default-settings source and then writing the defaults is not a change either.
	c2 := l.other("Knowledge", "k2").create(with(in, map[string]property.Value{"name": sv("other")}))
	imp2 := l.read(c2.ID, empty(), empty())
	require.False(t, imp2.Inputs.Get("crawlDepth").IsNumber())
	requireNoChanges(t, l.diff(c2.ID, imp2.Properties, with(explicit, map[string]property.Value{"name": sv("other")}), imp2.Inputs))
}

func TestKnowledgeLC_FailedProcessingRemovesSource(t *testing.T) {
	fastKnowledgePolls(t)
	l := newTL(t, "Knowledge", "k")
	seedKnowledgeBase(l)
	in := with(webKnowledgeInputs(), map[string]property.Value{"webUrl": sv("https://fail.example.com")})
	_, err := l.s.Create(pgp.CreateRequest{Urn: l.urn, Properties: l.checked(in)})
	require.ErrorContains(t, err, "FAILED")
	require.NotEmpty(t, l.f.callsMatching("DELETE", "/knowledge/v2/KnowledgeBases/"+kbID+"/Knowledge/"), "the half-created source is removed")
	for k := range l.f.objs {
		require.False(t, strings.HasPrefix(k, "kitem/"), "nothing is leaked: %s", k)
	}
}

func TestKnowledgeLC_Checks(t *testing.T) {
	l := newTL(t, "Knowledge", "k")
	text, web := textKnowledgeInputs(), webKnowledgeInputs()
	require.Empty(t, l.check(text).Failures)
	require.Empty(t, l.check(web).Failures)
	for name, c := range map[string]struct {
		in   property.Map
		prop string
	}{
		"empty name":        {with(text, map[string]property.Value{"name": sv("")}), "name"},
		"long name":         {with(text, map[string]property.Value{"name": sv(strings.Repeat("n", 31))}), "name"},
		"long description":  {with(text, map[string]property.Value{"description": sv(strings.Repeat("d", 1025))}), "description"},
		"file with text":    {with(text, map[string]property.Value{"sourceType": sv("File")}), "sourceType"},
		"unknown type":      {with(text, map[string]property.Value{"sourceType": sv("Video")}), "sourceType"},
		"web without url":   {without(web, "webUrl"), "webUrl"},
		"bad url":           {with(web, map[string]property.Value{"webUrl": sv("ftp://x")}), "webUrl"},
		"depth too big":     {with(web, map[string]property.Value{"crawlDepth": nv(11)}), "crawlDepth"},
		"bad period":        {with(web, map[string]property.Value{"crawlPeriod": sv("DAILY")}), "crawlPeriod"},
		"text without text": {without(text, "textContent"), "textContent"},
		"text with url":     {with(text, map[string]property.Value{"webUrl": sv("https://x.example.com")}), "webUrl"},
		"web with text":     {with(web, map[string]property.Value{"textContent": sv("x")}), "textContent"},
		"text with crawl":   {with(text, map[string]property.Value{"crawlDepth": nv(3)}), "crawlDepth"},
		"oversized text":    {with(text, map[string]property.Value{"textContent": sv(strings.Repeat("x", 185001))}), "textContent"},
	} {
		f := l.check(c.in).Failures
		require.NotEmpty(t, f, name)
		require.Equal(t, c.prop, f[0].Property, name)
	}
}

func TestLookup_KnowledgeBaseAndKnowledge(t *testing.T) {
	fastKnowledgePolls(t)
	l := newTL(t, "KnowledgeBase", "kb")
	kb := l.create(kbInputs())
	r, err := invoke(t, l, "getKnowledgeBase", map[string]property.Value{"knowledgeBaseId": sv(kb.ID)})
	require.NoError(t, err)
	require.Equal(t, "product-docs", getS(t, r, "displayName"))
	require.Equal(t, "Docs for support", getS(t, r, "description"))
	require.Equal(t, "ACTIVE", getS(t, r, "status"))
	_, err = invoke(t, l, "getKnowledgeBase", map[string]property.Value{"knowledgeBaseId": sv("know_knowledgebase_missing")})
	require.Error(t, err)
	_, err = invoke(t, l, "getKnowledgeBase", map[string]property.Value{"knowledgeBaseId": sv("")})
	require.Error(t, err)

	k := l.other("Knowledge", "k").create(with(webKnowledgeInputs(), map[string]property.Value{"knowledgeBaseId": sv(kb.ID)}))
	kid := getS(t, k.Properties, "knowledgeId")
	r, err = invoke(t, l, "getKnowledge", map[string]property.Value{"knowledgeBaseId": sv(kb.ID), "knowledgeId": sv(kid)})
	require.NoError(t, err)
	require.Equal(t, "Web", getS(t, r, "sourceType"))
	require.Equal(t, "https://docs.example.com", getS(t, r, "webUrl"))
	require.Equal(t, 2.0, r.Get("crawlDepth").AsNumber())
	require.Equal(t, "NEVER", getS(t, r, "crawlPeriod"))
	_, err = invoke(t, l, "getKnowledge", map[string]property.Value{"knowledgeBaseId": sv(kb.ID), "knowledgeId": sv("")})
	require.Error(t, err)
}

func writeKnowledgeFile(t *testing.T, name, content string) string {
	t.Helper()
	p := filepath.Join(t.TempDir(), name)
	require.NoError(t, os.WriteFile(p, []byte(content), 0o600))
	return p
}

func fileKnowledgeInputs(path string) property.Map {
	return pv(map[string]property.Value{"knowledgeBaseId": sv(kbID), "name": sv("handbook"), "sourceType": sv("File"), "filePath": sv(path)})
}

func sha256Hex(s string) string {
	sum := sha256.Sum256([]byte(s))
	return hex.EncodeToString(sum[:])
}

func TestKnowledgeLC_FileLifecycle(t *testing.T) {
	fastKnowledgePolls(t)
	l := newTL(t, "Knowledge", "k")
	seedKnowledgeBase(l)
	path := writeKnowledgeFile(t, "handbook.md", "# Handbook\nOpen 9 to 5.\n")
	in := fileKnowledgeInputs(path)
	require.Empty(t, l.check(in).Failures)
	c := l.create(in)
	kid := getS(t, c.Properties, "knowledgeId")
	require.Equal(t, "COMPLETED", getS(t, c.Properties, "status"))
	require.Equal(t, sha256Hex("# Handbook\nOpen 9 to 5.\n"), getS(t, c.Properties, "fileSha256"))
	require.JSONEq(t, `{"name":"handbook","source":{"type":"File","fileName":"handbook.md","fileSize":24,"mimeType":"text/markdown"}}`,
		l.f.callsMatching("POST", "/knowledge/v2/KnowledgeBases/"+kbID+"/Knowledge")[0].Body)
	// The upload reached the presigned URL with the file's exact content.
	require.Equal(t, sha256Hex("# Handbook\nOpen 9 to 5.\n"), l.f.get("kitem", testAccount, kid)["uploadedSha256"])
	requireNoChanges(t, l.diff(c.ID, c.Properties, in, in))
	rd := l.read(c.ID, c.Properties, in)
	requireNoChanges(t, l.diff(c.ID, rd.Properties, in, rd.Inputs))

	// A rename updates in place and sends no source (no re-upload).
	next := with(in, map[string]property.Value{"name": sv("handbook-2")})
	require.Equal(t, pgp.Update, l.diff(c.ID, c.Properties, next, in).DetailedDiff["name"].Kind)
	l.f.resetCalls()
	u := l.update(c.ID, c.Properties, next, in)
	require.JSONEq(t, `{"name":"handbook-2"}`, l.f.callsMatching("PATCH", "/knowledge/v2/KnowledgeBases/"+kbID+"/Knowledge/"+kid)[0].Body)

	// New content on disk replaces the source; the same content in another directory does not.
	require.NoError(t, os.WriteFile(path, []byte("# Handbook v2\n"), 0o600))
	require.Equal(t, pgp.UpdateReplace, l.diff(c.ID, u.Properties, next, next).DetailedDiff["filePath"].Kind)
	require.NoError(t, os.WriteFile(path, []byte("# Handbook\nOpen 9 to 5.\n"), 0o600))
	requireNoChanges(t, l.diff(c.ID, u.Properties, next, next))
	moved := writeKnowledgeFile(t, "handbook.md", "# Handbook\nOpen 9 to 5.\n")
	d := l.diff(c.ID, u.Properties, with(next, map[string]property.Value{"filePath": sv(moved)}), next)
	require.Equal(t, pgp.Update, d.DetailedDiff["filePath"].Kind)
	// A different file name replaces it, even with the same content.
	renamed := writeKnowledgeFile(t, "handbook-v2.md", "# Handbook\nOpen 9 to 5.\n")
	d = l.diff(c.ID, u.Properties, with(next, map[string]property.Value{"filePath": sv(renamed)}), next)
	require.Equal(t, pgp.UpdateReplace, d.DetailedDiff["filePath"].Kind)

	// Import: the file cannot be compared, so a program naming a file plans nothing.
	imp := l.read(c.ID, empty(), empty())
	require.Equal(t, "File", getS(t, imp.Inputs, "sourceType"))
	requireNoChanges(t, l.diff(c.ID, imp.Properties, next, imp.Inputs))

	l.del(c.ID, u.Properties)
	require.Nil(t, l.f.get("kitem", testAccount, kid))
}

func TestKnowledgeLC_FileCreateReadsImportUrlWhenOmitted(t *testing.T) {
	fastKnowledgePolls(t)
	knowledgeUploadOmitURL = true
	t.Cleanup(func() { knowledgeUploadOmitURL = false })
	l := newTL(t, "Knowledge", "k")
	seedKnowledgeBase(l)
	c := l.create(fileKnowledgeInputs(writeKnowledgeFile(t, "a.txt", "hello")))
	require.Equal(t, "COMPLETED", getS(t, c.Properties, "status"))
	require.Equal(t, sha256Hex("hello"), l.f.get("kitem", testAccount, getS(t, c.Properties, "knowledgeId"))["uploadedSha256"])
}

func TestKnowledgeLC_FileUploadFailureRemovesSource(t *testing.T) {
	fastKnowledgePolls(t)
	oldDelay := retryBaseDelay
	retryBaseDelay = time.Millisecond
	t.Cleanup(func() { retryBaseDelay = oldDelay })
	l := newTL(t, "Knowledge", "k")
	seedKnowledgeBase(l)
	in := fileKnowledgeInputs(writeKnowledgeFile(t, "a.txt", "hello"))
	knowledgeRejectUpload = true
	t.Cleanup(func() { knowledgeRejectUpload = false })
	_, err := l.s.Create(pgp.CreateRequest{Urn: l.urn, Properties: l.checked(in)})
	require.ErrorContains(t, err, "upload a.txt")
	require.NotEmpty(t, l.f.callsMatching("DELETE", "/knowledge/v2/KnowledgeBases/"+kbID+"/Knowledge/"), "the half-created source is removed")
	for k := range l.f.objs {
		require.False(t, strings.HasPrefix(k, "kitem/"), "nothing is leaked: %s", k)
	}
}

func TestKnowledgeLC_FileChecks(t *testing.T) {
	l := newTL(t, "Knowledge", "k")
	dir := t.TempDir()
	good := writeKnowledgeFile(t, "ok.pdf", "%PDF-1.4")
	require.Empty(t, l.check(fileKnowledgeInputs(good)).Failures)
	empty0 := writeKnowledgeFile(t, "empty.txt", "")
	for name, c := range map[string]struct {
		in   property.Map
		prop string
	}{
		"no path":         {without(fileKnowledgeInputs(good), "filePath"), "filePath"},
		"missing file":    {fileKnowledgeInputs(filepath.Join(dir, "nope.txt")), "filePath"},
		"directory":       {fileKnowledgeInputs(dir), "filePath"},
		"bad extension":   {fileKnowledgeInputs(writeKnowledgeFile(t, "x.docx", "x")), "filePath"},
		"empty file":      {fileKnowledgeInputs(empty0), "filePath"},
		"long name":       {fileKnowledgeInputs(filepath.Join(dir, strings.Repeat("n", 254)+".md")), "filePath"},
		"file with url":   {with(fileKnowledgeInputs(good), map[string]property.Value{"webUrl": sv("https://x.example.com")}), "sourceType"},
		"web with a path": {with(webKnowledgeInputs(), map[string]property.Value{"filePath": sv(good)}), "filePath"},
		"text with path":  {with(textKnowledgeInputs(), map[string]property.Value{"filePath": sv(good)}), "filePath"},
	} {
		f := l.check(c.in).Failures
		require.NotEmpty(t, f, name)
		require.Equal(t, c.prop, f[0].Property, name)
	}
}

func TestInspectKnowledgeFile_Limits(t *testing.T) {
	p := filepath.Join(t.TempDir(), "big.csv")
	f, err := os.Create(p)
	require.NoError(t, err)
	require.NoError(t, f.Truncate(knowledgeMaxFileSize+1))
	require.NoError(t, f.Close())
	_, err = inspectKnowledgeFile(p)
	require.ErrorContains(t, err, "16 MiB")
	require.NoError(t, os.Truncate(p, knowledgeMaxFileSize))
	fi, err := inspectKnowledgeFile(p)
	require.NoError(t, err)
	require.Equal(t, "text/csv", fi.mimeType)
}
