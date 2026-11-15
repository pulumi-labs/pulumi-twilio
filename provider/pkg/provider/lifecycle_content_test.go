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
	"strings"
	"testing"

	pgp "github.com/pulumi/pulumi-go-provider"
	"github.com/pulumi/pulumi/sdk/v3/go/property"
	"github.com/stretchr/testify/require"
)

func fnv(f float64) property.Value { return property.New(f) }

func obj2(kv map[string]property.Value) property.Value { return property.New(pv(kv)) }

func contentInputs() property.Map {
	return pv(map[string]property.Value{
		"friendlyName": sv("welcome"),
		"language":     sv("en"),
		"variables":    obj2(map[string]property.Value{"1": sv("Ada")}),
		"types": obj2(map[string]property.Value{
			"text": obj2(map[string]property.Value{"body": sv("Hello {{1}}")}),
			"quickReply": obj2(map[string]property.Value{"body": sv("Pick one"), "actions": arrv(
				obj2(map[string]property.Value{"title": sv("Yes"), "id": sv("y")}),
				obj2(map[string]property.Value{"title": sv("No"), "id": sv("n")}))}),
			"whatsappCard": obj2(map[string]property.Value{"body": sv("Card {{1}}"), "headerText": sv("Hi"), "actions": arrv(
				obj2(map[string]property.Value{"type": sv("URL"), "title": sv("Open"), "url": sv("https://example.com/{{1}}"), "webviewSize": sv("FULL")}))}),
			"location": obj2(map[string]property.Value{"latitude": fnv(39.7392), "longitude": fnv(-104.9903), "label": sv("Denver")}),
		}),
	})
}

func TestContentTemplateLC_Lifecycle(t *testing.T) {
	l := newTL(t, "ContentTemplate", "tpl")
	in := contentInputs()
	c := l.create(in)
	require.True(t, strings.HasPrefix(c.ID, "HX"), c.ID)
	require.Equal(t, c.ID, getS(t, c.Properties, "sid"))
	require.Equal(t, testAccount, getS(t, c.Properties, "accountSid"))

	// The body is JSON and the types are keyed by the Twilio slash names.
	calls := l.f.callsMatching("POST", "/content/v1/Content")
	require.Len(t, calls, 1)
	var sent map[string]any
	require.NoError(t, json.Unmarshal([]byte(calls[0].Body), &sent))
	require.Equal(t, "en", sent["language"])
	require.Equal(t, "welcome", sent["friendly_name"])
	types := sent["types"].(map[string]any)
	require.Len(t, types, 4)
	for _, k := range []string{"twilio/text", "twilio/quick-reply", "whatsapp/card", "twilio/location"} {
		require.Contains(t, types, k)
	}
	card := types["whatsapp/card"].(map[string]any)
	require.Equal(t, "Hi", card["header_text"])
	require.Equal(t, "FULL", card["actions"].([]any)[0].(map[string]any)["webview_size"])
	require.InDelta(t, -104.9903, types["twilio/location"].(map[string]any)["longitude"], 1e-9)

	requireNoChanges(t, l.diff(c.ID, c.Properties, in, in))
	rd := l.read(c.ID, c.Properties, in)
	requireNoChanges(t, l.diff(c.ID, rd.Properties, in, rd.Inputs))

	// Drift in the text shows.
	l.f.mutate("ct", testAccount, c.ID, func(o map[string]any) {
		o["types"].(map[string]any)["twilio/text"] = map[string]any{"body": "Changed"}
	})
	rd = l.read(c.ID, c.Properties, in)
	require.Equal(t, pgp.Update, l.diff(c.ID, rd.Properties, in, rd.Inputs).DetailedDiff["types.text.body"].Kind)

	// Everything updates in place with a PUT that carries the whole template.
	next := with(in, map[string]property.Value{"language": sv("fr"), "friendlyName": sv("bienvenue")})
	d := l.diff(c.ID, c.Properties, next, in)
	require.Equal(t, pgp.Update, d.DetailedDiff["language"].Kind)
	require.Equal(t, pgp.Update, d.DetailedDiff["friendlyName"].Kind)
	l.f.resetCalls()
	u := l.update(c.ID, c.Properties, next, in)
	put := l.f.callsMatching("PUT", "/content/v1/Content/"+c.ID)
	require.Len(t, put, 1)
	require.Contains(t, put[0].Body, `"twilio/text"`)
	require.Equal(t, "fr", l.f.get("ct", testAccount, c.ID)["language"])
	require.Equal(t, "bienvenue", l.f.get("ct", testAccount, c.ID)["friendly_name"])
	requireNoChanges(t, l.diff(c.ID, u.Properties, next, next))

	// An import adopts what Twilio reports.
	l.f.mutate("ct", testAccount, c.ID, func(o map[string]any) {
		o["types"].(map[string]any)["twilio/text"] = map[string]any{"body": "Hello {{1}}"}
	})
	imp := l.read(c.ID, empty(), empty())
	require.Empty(t, l.check(imp.Inputs).Failures)
	require.Equal(t, "bienvenue", getS(t, imp.Inputs, "friendlyName"))
	require.Equal(t, "Ada", imp.Inputs.Get("variables").AsMap().Get("1").AsString())
	require.True(t, hasProp(imp.Inputs.Get("types").AsMap(), "whatsappCard"))
	require.False(t, hasProp(imp.Inputs.Get("types").AsMap(), "carousel"))
	l.requireImportFails("HX" + strings.Repeat("0", 32))

	// Deleted outside Pulumi: Read returns an empty ID, and Delete tolerates the absence.
	l.f.remove("ct", testAccount, c.ID)
	require.Empty(t, l.read(c.ID, u.Properties, next).ID)
	l.del(c.ID, u.Properties)
	c = l.create(in)
	l.del(c.ID, c.Properties)
	require.Nil(t, l.f.get("ct", testAccount, c.ID))
}

func TestContentTemplateLC_OptionalsAreLeftAloneWhenRemoved(t *testing.T) {
	l := newTL(t, "ContentTemplate", "tpl")
	in := contentInputs()
	c := l.create(in)
	min := pv(map[string]property.Value{"language": sv("en"), "types": in.Get("types")})
	u := l.update(c.ID, c.Properties, min, in)
	put := l.f.callsMatching("PUT", "/content/v1/Content/"+c.ID)[0].Body
	require.NotContains(t, put, "friendly_name")
	require.NotContains(t, put, "variables")
	// Twilio keeps the name, and a program that never states one sees no drift.
	rd := l.read(c.ID, u.Properties, min)
	require.False(t, hasProp(rd.Inputs, "friendlyName"))
	require.False(t, hasProp(rd.Inputs, "variables"))
	requireNoChanges(t, l.diff(c.ID, rd.Properties, min, rd.Inputs))
}

func TestContentTemplateLC_Check(t *testing.T) {
	l := newTL(t, "ContentTemplate", "tpl")
	text := obj2(map[string]property.Value{"text": obj2(map[string]property.Value{"body": sv("hi")})})
	require.Empty(t, l.check(pv(map[string]property.Value{"language": sv("en"), "types": text})).Failures)
	// No content type, empty language, empty name.
	require.Len(t, l.check(pv(map[string]property.Value{"language": sv(""), "types": obj2(nil), "friendlyName": sv("")})).Failures, 3)
	bad := obj2(map[string]property.Value{
		"callToAction": obj2(map[string]property.Value{"body": sv("b"), "actions": arrv(obj2(map[string]property.Value{"type": sv("EMAIL")}))}),
		"carousel": obj2(map[string]property.Value{"body": sv("b"), "cards": arrv(obj2(map[string]property.Value{
			"body": sv("c"), "media": sv("https://example.com/a.png"),
			"actions": arrv(obj2(map[string]property.Value{"type": sv("PIX"), "title": sv("t")}))}))}),
		"whatsappCard": obj2(map[string]property.Value{"body": sv("b"), "actions": arrv(obj2(map[string]property.Value{"type": sv("URL"), "webviewSize": sv("HUGE")}))}),
		"media":        obj2(map[string]property.Value{"media": arrv()}),
	})
	require.Len(t, l.check(pv(map[string]property.Value{"language": sv("en"), "types": bad})).Failures, 4)
	// An unknown types value is not judged.
	require.Empty(t, l.check(pv(map[string]property.Value{"language": sv("en"), "types": computed()})).Failures)
}

func TestContentTemplateLC_CreateReadBackFailureKeepsTheTemplate(t *testing.T) {
	l := newTL(t, "ContentTemplate", "tpl")
	// A response whose types cannot be decoded still names the template that was created.
	in := pv(map[string]property.Value{"language": sv("en"), "types": obj2(map[string]property.Value{"text": obj2(map[string]property.Value{"body": sv("hi")})})})
	contentMangleCreate = func(o map[string]any) { o["types"] = map[string]any{"twilio/text": []any{"not an object"}} }
	t.Cleanup(func() { contentMangleCreate = nil })
	_, err := l.s.Create(pgp.CreateRequest{Urn: l.urn, Properties: l.checked(in)})
	require.ErrorContains(t, err, "failed to initialize")
	n := 0
	for k := range l.f.objs {
		if strings.HasPrefix(k, "ct/") {
			n++
		}
	}
	require.Equal(t, 1, n)
}

func TestLookup_Content(t *testing.T) {
	l := newTL(t, "ContentTemplate", "tpl")
	c := l.create(contentInputs())
	r, err := invoke(t, l, "getContentTemplate", map[string]property.Value{"sid": sv(c.ID)})
	require.NoError(t, err)
	require.Equal(t, "welcome", getS(t, r, "friendlyName"))
	require.Equal(t, "en", getS(t, r, "language"))
	require.Equal(t, "Hello {{1}}", r.Get("types").AsMap().Get("text").AsMap().Get("body").AsString())
	require.Equal(t, "Denver", r.Get("types").AsMap().Get("location").AsMap().Get("label").AsString())
	require.Equal(t, "Ada", r.Get("variables").AsMap().Get("1").AsString())
	_, err = invoke(t, l, "getContentTemplate", map[string]property.Value{"sid": sv("HXmissing")})
	require.ErrorContains(t, err, "was not found")
	_, err = invoke(t, l, "getContentTemplate", map[string]property.Value{"sid": sv("")})
	require.ErrorContains(t, err, "must be set")

	// Never submitted: only the identity comes back.
	r, err = invoke(t, l, "getContentApprovalStatus", map[string]property.Value{"sid": sv(c.ID)})
	require.NoError(t, err)
	require.Equal(t, c.ID, getS(t, r, "sid"))
	require.False(t, hasProp(r, "whatsappStatus"))
	require.False(t, hasProp(r, "whatsappJson"))
	l.f.mutate("ct", testAccount, c.ID, func(o map[string]any) {
		o["approval"] = map[string]any{"status": "rejected", "rejection_reason": "INVALID_FORMAT", "category": "UTILITY", "name": "welcome"}
	})
	r, err = invoke(t, l, "getContentApprovalStatus", map[string]property.Value{"sid": sv(c.ID)})
	require.NoError(t, err)
	require.Equal(t, "rejected", getS(t, r, "whatsappStatus"))
	require.Equal(t, "INVALID_FORMAT", getS(t, r, "whatsappRejectionReason"))
	require.Equal(t, "UTILITY", getS(t, r, "whatsappCategory"))
	require.Contains(t, getS(t, r, "whatsappJson"), `"name":"welcome"`)
	_, err = invoke(t, l, "getContentApprovalStatus", map[string]property.Value{"sid": sv("HXmissing")})
	require.ErrorContains(t, err, "was not found")
	_, err = invoke(t, l, "getContentApprovalStatus", map[string]property.Value{"sid": sv("")})
	require.ErrorContains(t, err, "must be set")
}

func approvalInputs(sid string) property.Map {
	return pv(map[string]property.Value{"contentSid": sv(sid), "name": sv("welcome_msg"), "category": sv("UTILITY"), "sendTtlSeconds": fnv(3600)})
}

func TestContentApprovalRequestLC_Lifecycle(t *testing.T) {
	lt := newTL(t, "ContentTemplate", "tpl")
	tpl := lt.create(contentInputs())
	l := lt.other("ContentApprovalRequest", "appr")
	in := approvalInputs(tpl.ID)

	c := l.create(in)
	require.Equal(t, tpl.ID, c.ID)
	require.Equal(t, "received", getS(t, c.Properties, "status"))
	require.Equal(t, "UTILITY", getS(t, c.Properties, "reviewedCategory"))
	post := l.f.callsMatching("POST", "/content/v1/Content/"+tpl.ID+"/ApprovalRequests/whatsapp")
	require.Len(t, post, 1)
	var sent map[string]any
	require.NoError(t, json.Unmarshal([]byte(post[0].Body), &sent))
	require.Equal(t, map[string]any{"name": "welcome_msg", "category": "UTILITY", "send_ttl_seconds": float64(3600)}, sent)

	// No drift, also after a refresh; the program writing nothing for bidAmount sees none.
	requireNoChanges(t, l.diff(c.ID, c.Properties, in, in))
	rd := l.read(c.ID, c.Properties, in)
	requireNoChanges(t, l.diff(c.ID, rd.Properties, in, rd.Inputs))

	// The review runs later: a refresh shows it, and a recategorised template is not drift.
	l.f.mutate("ct", testAccount, tpl.ID, func(o map[string]any) {
		a := o["approval"].(map[string]any)
		a["status"] = "approved"
		a["category"] = "MARKETING"
	})
	rd = l.read(c.ID, c.Properties, in)
	require.Equal(t, "approved", getS(t, rd.Properties, "status"))
	require.Equal(t, "MARKETING", getS(t, rd.Properties, "reviewedCategory"))
	require.Equal(t, "UTILITY", getS(t, rd.Inputs, "category"))
	requireNoChanges(t, l.diff(c.ID, rd.Properties, in, rd.Inputs))

	// Name, category and the TTL replace; the content SID replaces.
	for k, v := range map[string]property.Value{"name": sv("other"), "category": sv("MARKETING"), "sendTtlSeconds": fnv(60), "contentSid": sv("HX" + strings.Repeat("1", 32))} {
		d := l.diff(c.ID, c.Properties, with(in, map[string]property.Value{k: v}), in)
		require.Equal(t, pgp.UpdateReplace, d.DetailedDiff[k].Kind, k)
	}

	// bidAmount updates in place with a PATCH.
	next := with(in, map[string]property.Value{"bidAmount": fnv(5)})
	d := l.diff(c.ID, c.Properties, next, in)
	require.Equal(t, pgp.Add, d.DetailedDiff["bidAmount"].Kind)
	l.f.resetCalls()
	u := l.update(c.ID, c.Properties, next, in)
	patch := l.f.callsMatching("PATCH", "/content/v1/Content/"+tpl.ID+"/ApprovalRequests/whatsapp")
	require.Len(t, patch, 1)
	require.JSONEq(t, `{"bid_amount":5}`, patch[0].Body)
	requireNoChanges(t, l.diff(c.ID, u.Properties, next, next))
	// Removing it sends nothing.
	l.f.resetCalls()
	l.update(c.ID, u.Properties, in, next)
	require.Empty(t, l.f.callsMatching("PATCH", "/content/v1/Content/"+tpl.ID+"/ApprovalRequests/whatsapp"))

	// An import adopts what Twilio reports.
	imp := l.read(tpl.ID, empty(), empty())
	require.Empty(t, l.check(imp.Inputs).Failures)
	require.Equal(t, "welcome_msg", getS(t, imp.Inputs, "name"))
	require.Equal(t, "MARKETING", getS(t, imp.Inputs, "category"))
	require.Equal(t, tpl.ID, getS(t, imp.Inputs, "contentSid"))

	// Delete forgets the submission: no call, the approval stays at Twilio.
	l.f.resetCalls()
	l.del(c.ID, c.Properties)
	for _, m := range []string{"DELETE", "POST", "PATCH", "PUT"} {
		require.Empty(t, l.f.callsMatching(m, "/content/v1/Content"))
	}
	require.NotNil(t, l.f.get("ct", testAccount, tpl.ID)["approval"])
}

func TestContentApprovalRequestLC_Gone(t *testing.T) {
	lt := newTL(t, "ContentTemplate", "tpl")
	tpl := lt.create(contentInputs())
	l := lt.other("ContentApprovalRequest", "appr")
	in := approvalInputs(tpl.ID)
	c := l.create(in)

	// Reported as unsubmitted: the request is gone, and importing it fails.
	l.f.mutate("ct", testAccount, tpl.ID, func(o map[string]any) { o["approval"] = map[string]any{"status": "unsubmitted"} })
	require.Empty(t, l.read(c.ID, c.Properties, in).ID)
	l.requireImportFails(tpl.ID)
	l.f.mutate("ct", testAccount, tpl.ID, func(o map[string]any) { delete(o, "approval") })
	require.Empty(t, l.read(c.ID, c.Properties, in).ID)
	l.requireImportFails(tpl.ID)

	// The template itself is gone.
	l.f.remove("ct", testAccount, tpl.ID)
	require.Empty(t, l.read(c.ID, c.Properties, in).ID)
	l.requireImportFails(tpl.ID)
	l.del(c.ID, c.Properties)
}

func TestContentApprovalRequestLC_Check(t *testing.T) {
	l := newTL(t, "ContentApprovalRequest", "appr")
	sid := "HX" + strings.Repeat("a", 32)
	require.Empty(t, l.check(approvalInputs(sid)).Failures)
	bad := pv(map[string]property.Value{"contentSid": sv("SM123"), "name": sv(""), "category": sv("SPAM"), "bidAmount": fnv(0), "sendTtlSeconds": fnv(-1)})
	require.Len(t, l.check(bad).Failures, 5)
	require.Empty(t, l.check(pv(map[string]property.Value{"contentSid": computed(), "name": computed(), "category": computed()})).Failures)
}

func TestLookup_ContentApprovalRequest(t *testing.T) {
	l := newTL(t, "ContentTemplate", "tpl")
	c := l.create(contentInputs())
	_, err := invoke(t, l, "getContentApprovalRequest", map[string]property.Value{"contentSid": sv(c.ID)})
	require.ErrorContains(t, err, "has no WhatsApp approval request")
	l.f.mutate("ct", testAccount, c.ID, func(o map[string]any) {
		o["approval"] = map[string]any{"status": "rejected", "rejection_reason": "INVALID_FORMAT", "category": "UTILITY", "name": "welcome", "bid_amount": float64(3)}
	})
	r, err := invoke(t, l, "getContentApprovalRequest", map[string]property.Value{"contentSid": sv(c.ID)})
	require.NoError(t, err)
	require.Equal(t, "welcome", getS(t, r, "name"))
	require.Equal(t, "rejected", getS(t, r, "status"))
	require.Equal(t, "INVALID_FORMAT", getS(t, r, "rejectionReason"))
	require.Equal(t, 3.0, r.Get("bidAmount").AsNumber())
	require.False(t, hasProp(r, "sendTtlSeconds"))
	_, err = invoke(t, l, "getContentApprovalRequest", map[string]property.Value{"contentSid": sv("HXmissing")})
	require.ErrorContains(t, err, "was not found")
	_, err = invoke(t, l, "getContentApprovalRequest", map[string]property.Value{"contentSid": sv("")})
	require.ErrorContains(t, err, "must be set")
}
