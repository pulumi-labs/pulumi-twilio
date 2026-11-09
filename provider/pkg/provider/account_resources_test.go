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
	"testing"

	pgp "github.com/pulumi/pulumi-go-provider"
	"github.com/pulumi/pulumi/sdk/v3/go/property"
	"github.com/stretchr/testify/require"
)

func addrInputs() property.Map {
	return pv(map[string]property.Value{
		"customerName": sv("Acme"), "street": sv("1 Main St"), "city": sv("Portland"), "region": sv("OR"),
		"postalCode": sv("97201"), "isoCountry": sv("US"), "friendlyName": sv("hq"),
	})
}

// ------------------------------------------------------------------ Address

func TestAddressLC_Lifecycle(t *testing.T) {
	l := newTL(t, "Address", "addr")
	in := with(addrInputs(), map[string]property.Value{"autoCorrectAddress": bv(false)})
	c := l.create(in)
	sid := c.ID
	require.Equal(t, sid, getS(t, c.Properties, "sid"))
	require.Equal(t, testAccount, getS(t, c.Properties, "ownerAccountSid"))
	require.True(t, c.Properties.Get("validated").AsBool())
	stored := l.f.get("address", testAccount, sid)
	require.Equal(t, "1 Main St", stored["street"])
	require.Equal(t, "false", l.f.callsMatching("POST", "/Addresses.json")[0].Form.Get("AutoCorrectAddress"))
	requireNoChanges(t, l.diff(sid, c.Properties, in, in))
	rd := l.read(sid, c.Properties, in)
	requireNoChanges(t, l.diff(sid, rd.Properties, in, rd.Inputs))

	// Updatable fields reach the API; the update never sends IsoCountry.
	next := with(in, map[string]property.Value{"street": sv("2 Oak Ave"), "friendlyName": sv("hq-2"),
		"streetSecondary": sv("Suite 4"), "emergencyEnabled": bv(true)})
	d := l.diff(sid, rd.Properties, next, in)
	require.True(t, d.HasChanges)
	require.False(t, d.DeleteBeforeReplace)
	u := l.update(sid, rd.Properties, next, in)
	stored = l.f.get("address", testAccount, sid)
	require.Equal(t, "2 Oak Ave", stored["street"])
	require.Equal(t, "hq-2", stored["friendly_name"])
	require.Equal(t, "Suite 4", stored["street_secondary"])
	require.Equal(t, true, stored["emergency_enabled"])
	ups := l.f.callsMatching("POST", "/Addresses/")
	require.NotContains(t, ups[len(ups)-1].Form, "IsoCountry")
	requireNoChanges(t, l.diff(sid, u.Properties, next, next))

	// Removing the second line clears it; a removed name is left as it is (Twilio has no default for it).
	min := without(next, "friendlyName", "streetSecondary")
	u2 := l.update(sid, u.Properties, min, next)
	stored = l.f.get("address", testAccount, sid)
	require.Equal(t, "hq-2", stored["friendly_name"])
	require.Nil(t, stored["street_secondary"])

	// The country cannot change in place.
	d = l.diff(sid, u2.Properties, with(min, map[string]property.Value{"isoCountry": sv("CA")}), min)
	require.Equal(t, pgp.UpdateReplace, d.DetailedDiff["isoCountry"].Kind)
	d = l.diff(sid, u2.Properties, with(min, map[string]property.Value{"accountSid": sv(fakeSub)}), min)
	require.Equal(t, pgp.AddReplace, d.DetailedDiff["accountSid"].Kind)

	l.del(sid, u2.Properties)
	require.Nil(t, l.f.get("address", testAccount, sid))
	l.del(sid, u2.Properties) // 404 is success
}

func TestAddressLC_AutoCorrectDoesNotDrift(t *testing.T) {
	l := newTL(t, "Address", "addr")
	in := addrInputs() // autoCorrectAddress unset: Twilio upper-cases the stored street
	c := l.create(in)
	require.Equal(t, "1 MAIN ST", l.f.get("address", testAccount, c.ID)["street"])
	rd := l.read(c.ID, c.Properties, in)
	require.Equal(t, "1 Main St", getS(t, rd.Inputs, "street"), "auto-corrected lines are not refreshed")
	requireNoChanges(t, l.diff(c.ID, rd.Properties, in, rd.Inputs))
}

func TestAddressLC_DriftImportAndDeletedOutside(t *testing.T) {
	l := newTL(t, "Address", "addr")
	in := with(addrInputs(), map[string]property.Value{"autoCorrectAddress": bv(false)})
	c := l.create(in)
	l.f.mutate("address", testAccount, c.ID, func(o map[string]any) { o["city"] = "Seattle" })
	rd := l.read(c.ID, c.Properties, in)
	require.Equal(t, "Seattle", getS(t, rd.Inputs, "city"))
	require.Equal(t, pgp.Update, l.diff(c.ID, rd.Properties, in, rd.Inputs).DetailedDiff["city"].Kind)

	l.f.mutate("address", testAccount, c.ID, func(o map[string]any) { o["city"] = "Portland" })
	imp := l.read(c.ID, empty(), empty())
	require.Equal(t, c.ID, imp.ID)
	require.Equal(t, "Acme", getS(t, imp.Inputs, "customerName"))
	require.Equal(t, "US", getS(t, imp.Inputs, "isoCountry"))
	require.Equal(t, "hq", getS(t, imp.Inputs, "friendlyName"))
	require.False(t, hasProp(imp.Inputs, "accountSid"))
	requireNoChanges(t, l.diff(imp.ID, imp.Properties, l.checked(imp.Inputs), imp.Inputs))

	l.f.remove("address", testAccount, c.ID)
	require.Empty(t, l.read(c.ID, c.Properties, in).ID)
	l.del(c.ID, c.Properties)
}

func TestAddressLC_SubaccountAndInUse(t *testing.T) {
	l := newTL(t, "Address", "addr")
	l.f.addSubaccount(fakeSub)
	in := with(addrInputs(), map[string]property.Value{"accountSid": sv(fakeSub), "autoCorrectAddress": bv(false)})
	c := l.create(in)
	_, sid := splitScoped(c.ID)
	require.Equal(t, fakeSub+"/"+sid, c.ID)
	require.NotNil(t, l.f.get("address", fakeSub, sid))
	imp := l.read(c.ID, empty(), empty())
	require.Equal(t, fakeSub, getS(t, imp.Inputs, "accountSid"))
	requireNoChanges(t, l.diff(imp.ID, imp.Properties, l.checked(imp.Inputs), imp.Inputs))
	l.requireImportFails(sid)

	// An address that is still used cannot be deleted: the error reaches the user.
	l.f.mutate("address", fakeSub, sid, func(o map[string]any) { o["in_use"] = true })
	require.Error(t, l.s.Delete(pgp.DeleteRequest{ID: c.ID, Urn: l.urn, Properties: c.Properties}))
	l.f.mutate("address", fakeSub, sid, func(o map[string]any) { delete(o, "in_use") })
	l.del(c.ID, c.Properties)
	require.Nil(t, l.f.get("address", fakeSub, sid))
}

func TestAddressLC_PreviewAndUnknowns(t *testing.T) {
	l := newTL(t, "Address", "addr")
	_, err := l.s.Create(pgp.CreateRequest{Urn: l.urn, Properties: addrInputs(), DryRun: true})
	require.NoError(t, err)
	require.Zero(t, l.f.callCount())
	chk := l.check(with(addrInputs(), map[string]property.Value{"street": computed(), "isoCountry": computed()}))
	require.Empty(t, chk.Failures)
	require.NotEmpty(t, l.check(with(addrInputs(), map[string]property.Value{"street": sv("")})).Failures)
	require.NotEmpty(t, l.check(with(addrInputs(), map[string]property.Value{"isoCountry": sv("USA")})).Failures)
}

// --------------------------------------------------------------- SigningKey

func TestSigningKeyLC_Lifecycle(t *testing.T) {
	l := newTL(t, "SigningKey", "sk")
	in := pv(map[string]property.Value{"friendlyName": sv("tokens")})
	c := l.create(in)
	sid := c.ID
	require.Equal(t, "secret-"+sid, getS(t, c.Properties, "secret"))
	require.Equal(t, sid, getS(t, c.Properties, "sid"))
	requireNoChanges(t, l.diff(sid, c.Properties, in, in))

	// The secret survives a refresh (Twilio returns it only once).
	rd := l.read(sid, c.Properties, in)
	require.Equal(t, "secret-"+sid, getS(t, rd.Properties, "secret"))
	requireNoChanges(t, l.diff(sid, rd.Properties, in, rd.Inputs))

	next := pv(map[string]property.Value{"friendlyName": sv("tokens-2")})
	d := l.diff(sid, rd.Properties, next, in)
	require.True(t, d.HasChanges)
	require.False(t, d.DeleteBeforeReplace)
	u := l.update(sid, rd.Properties, next, in)
	require.Equal(t, "tokens-2", l.f.get("signingkey", testAccount, sid)["friendly_name"])
	require.Equal(t, "secret-"+sid, getS(t, u.Properties, "secret"))
	requireNoChanges(t, l.diff(sid, u.Properties, next, next))

	// Drift on the name is detected.
	l.f.mutate("signingkey", testAccount, sid, func(o map[string]any) { o["friendly_name"] = "renamed" })
	rd = l.read(sid, u.Properties, next)
	require.Equal(t, "renamed", getS(t, rd.Inputs, "friendlyName"))

	d = l.diff(sid, rd.Properties, with(next, map[string]property.Value{"accountSid": sv(fakeSub)}), next)
	require.Equal(t, pgp.AddReplace, d.DetailedDiff["accountSid"].Kind)

	l.del(sid, u.Properties)
	require.Nil(t, l.f.get("signingkey", testAccount, sid))
	l.del(sid, u.Properties)
}

func TestSigningKeyLC_ImportAndDeletedOutside(t *testing.T) {
	l := newTL(t, "SigningKey", "sk")
	l.f.addSubaccount(fakeSub)
	in := pv(map[string]property.Value{"friendlyName": sv("tokens"), "accountSid": sv(fakeSub)})
	c := l.create(in)
	_, sid := splitScoped(c.ID)
	require.Equal(t, fakeSub+"/"+sid, c.ID)

	imp := l.read(c.ID, empty(), empty())
	require.Equal(t, c.ID, imp.ID)
	require.Equal(t, "tokens", getS(t, imp.Inputs, "friendlyName"))
	require.Equal(t, fakeSub, getS(t, imp.Inputs, "accountSid"))
	require.Empty(t, getS(t, imp.Properties, "secret"), "the secret cannot be recovered on import")
	requireNoChanges(t, l.diff(imp.ID, imp.Properties, in, imp.Inputs))

	l.f.remove("signingkey", fakeSub, sid)
	require.Empty(t, l.read(c.ID, c.Properties, in).ID)
	l.del(c.ID, c.Properties)

	_, err := l.s.Create(pgp.CreateRequest{Urn: l.urn, Properties: pv(map[string]property.Value{"friendlyName": sv("x")}), DryRun: true})
	require.NoError(t, err)
	require.NotEmpty(t, l.check(pv(map[string]property.Value{"friendlyName": sv("")})).Failures)
	require.Empty(t, l.check(pv(map[string]property.Value{"friendlyName": computed()})).Failures)
}

// -------------------------------------------------------------------- Queue

func TestQueueLC_Lifecycle(t *testing.T) {
	l := newTL(t, "Queue", "q")
	in := pv(map[string]property.Value{"friendlyName": sv("support"), "maxSize": nv(50)})
	c := l.create(in)
	sid := c.ID
	require.Equal(t, sid, getS(t, c.Properties, "sid"))
	require.Equal(t, 0.0, c.Properties.Get("currentSize").AsNumber())
	require.Equal(t, 50.0, l.f.get("queue", testAccount, sid)["max_size"])
	requireNoChanges(t, l.diff(sid, c.Properties, in, in))

	rd := l.read(sid, c.Properties, in)
	requireNoChanges(t, l.diff(sid, rd.Properties, in, rd.Inputs))
	l.f.mutate("queue", testAccount, sid, func(o map[string]any) { o["current_size"] = float64(3) })
	rd = l.read(sid, c.Properties, in)
	require.Equal(t, 3.0, rd.Properties.Get("currentSize").AsNumber(), "live size is an output, not a diff")
	requireNoChanges(t, l.diff(sid, rd.Properties, in, rd.Inputs))

	next := pv(map[string]property.Value{"friendlyName": sv("support-2"), "maxSize": nv(200)})
	require.True(t, l.diff(sid, rd.Properties, next, in).HasChanges)
	u := l.update(sid, rd.Properties, next, in)
	stored := l.f.get("queue", testAccount, sid)
	require.Equal(t, "support-2", stored["friendly_name"])
	require.Equal(t, 200.0, stored["max_size"])
	requireNoChanges(t, l.diff(sid, u.Properties, next, next))

	// Removing maxSize resets it to Twilio's documented default.
	min := pv(map[string]property.Value{"friendlyName": sv("support-2")})
	u2 := l.update(sid, u.Properties, min, next)
	require.Equal(t, 1000.0, l.f.get("queue", testAccount, sid)["max_size"])
	requireNoChanges(t, l.diff(sid, u2.Properties, min, min))

	// Drift on maxSize when the program sets it.
	l.f.mutate("queue", testAccount, sid, func(o map[string]any) { o["max_size"] = float64(7) })
	rd = l.read(sid, u.Properties, next)
	require.Equal(t, 7.0, rd.Inputs.Get("maxSize").AsNumber())

	l.del(sid, u2.Properties)
	require.Nil(t, l.f.get("queue", testAccount, sid))
	l.del(sid, u2.Properties)
}

func TestQueueLC_ImportSubaccountDeletedAndChecks(t *testing.T) {
	l := newTL(t, "Queue", "q")
	l.f.addSubaccount(fakeSub)
	in := pv(map[string]property.Value{"friendlyName": sv("support"), "accountSid": sv(fakeSub), "maxSize": nv(10)})
	c := l.create(in)
	_, sid := splitScoped(c.ID)
	require.Equal(t, fakeSub+"/"+sid, c.ID)
	imp := l.read(c.ID, empty(), empty())
	require.Equal(t, fakeSub, getS(t, imp.Inputs, "accountSid"))
	require.Equal(t, 10.0, imp.Inputs.Get("maxSize").AsNumber())
	requireNoChanges(t, l.diff(imp.ID, imp.Properties, in, imp.Inputs))

	// A default-sized queue imports without a maxSize input.
	l.f.mutate("queue", fakeSub, sid, func(o map[string]any) { o["max_size"] = float64(1000) })
	imp = l.read(c.ID, empty(), empty())
	require.False(t, hasProp(imp.Inputs, "maxSize"))

	l.f.remove("queue", fakeSub, sid)
	require.Empty(t, l.read(c.ID, c.Properties, in).ID)
	l.del(c.ID, c.Properties)

	require.NotEmpty(t, l.check(pv(map[string]property.Value{"friendlyName": sv("")})).Failures)
	require.NotEmpty(t, l.check(pv(map[string]property.Value{"friendlyName": sv("q"), "maxSize": nv(5001)})).Failures)
	require.NotEmpty(t, l.check(pv(map[string]property.Value{"friendlyName": sv("q"), "maxSize": nv(0)})).Failures)
	require.Empty(t, l.check(pv(map[string]property.Value{"friendlyName": computed(), "maxSize": computed()})).Failures)
}

// ------------------------------------------------------------ UsageTrigger

func triggerInputs() property.Map {
	return pv(map[string]property.Value{
		"callbackUrl": sv("https://example.com/hook"), "triggerValue": sv("100"), "usageCategory": sv("sms"),
		"friendlyName": sv("sms-cap"), "recurring": sv("monthly"),
	})
}

func TestUsageTriggerLC_Lifecycle(t *testing.T) {
	l := newTL(t, "UsageTrigger", "ut")
	in := triggerInputs()
	c := l.create(in)
	sid := c.ID
	require.Equal(t, sid, getS(t, c.Properties, "sid"))
	require.Equal(t, "2010-04-01", getS(t, c.Properties, "apiVersion"))
	stored := l.f.get("trigger", testAccount, sid)
	require.Equal(t, "100.0", stored["trigger_value"], "Twilio reformats the value")
	require.Equal(t, "monthly", stored["recurring"])
	form := l.f.callsMatching("POST", "/Usage/Triggers.json")[0].Form
	require.Equal(t, "sms", form.Get("UsageCategory"))
	require.NotContains(t, form, "CallbackMethod")
	requireNoChanges(t, l.diff(sid, c.Properties, in, in))

	// A reformatted trigger value is not drift.
	rd := l.read(sid, c.Properties, in)
	require.Equal(t, "100", getS(t, rd.Inputs, "triggerValue"))
	requireNoChanges(t, l.diff(sid, rd.Properties, in, rd.Inputs))

	// Only the callback and the name change in place.
	next := with(in, map[string]property.Value{"callbackUrl": sv("https://example.com/v2"), "callbackMethod": sv("GET"), "friendlyName": sv("sms-cap-2")})
	d := l.diff(sid, rd.Properties, next, in)
	require.True(t, d.HasChanges)
	require.False(t, d.DeleteBeforeReplace)
	u := l.update(sid, rd.Properties, next, in)
	stored = l.f.get("trigger", testAccount, sid)
	require.Equal(t, "https://example.com/v2", stored["callback_url"])
	require.Equal(t, "GET", stored["callback_method"])
	require.Equal(t, "sms-cap-2", stored["friendly_name"])
	requireNoChanges(t, l.diff(sid, u.Properties, next, next))

	// Removing callbackMethod and friendlyName resets them.
	min := without(next, "callbackMethod", "friendlyName")
	u2 := l.update(sid, u.Properties, min, next)
	stored = l.f.get("trigger", testAccount, sid)
	require.Equal(t, "POST", stored["callback_method"])
	require.Equal(t, "", stored["friendly_name"])

	// The rest replaces the trigger.
	for k, v := range map[string]string{"triggerValue": "200", "usageCategory": "calls", "recurring": "daily", "triggerBy": "count", "accountSid": fakeSub} {
		d = l.diff(sid, u2.Properties, with(min, map[string]property.Value{k: sv(v)}), min)
		require.Contains(t, []pgp.DiffKind{pgp.UpdateReplace, pgp.AddReplace}, d.DetailedDiff[k].Kind, k)
	}

	l.del(sid, u2.Properties)
	require.Nil(t, l.f.get("trigger", testAccount, sid))
	l.del(sid, u2.Properties)
}

func TestUsageTriggerLC_OffsetImportAndDrift(t *testing.T) {
	l := newTL(t, "UsageTrigger", "ut")
	in := with(triggerInputs(), map[string]property.Value{"triggerValue": sv("+30")})
	c := l.create(in)
	sid := c.ID
	require.Equal(t, "30.0", l.f.get("trigger", testAccount, sid)["trigger_value"])
	rd := l.read(sid, c.Properties, in)
	require.Equal(t, "+30", getS(t, rd.Inputs, "triggerValue"), "an offset stays as the program wrote it")
	requireNoChanges(t, l.diff(sid, rd.Properties, in, rd.Inputs))

	// A trigger value changed outside Pulumi is drift (replacement).
	l.f.mutate("trigger", testAccount, sid, func(o map[string]any) { o["callback_url"] = "https://drift.example.com" })
	rd = l.read(sid, c.Properties, in)
	require.Equal(t, "https://drift.example.com", getS(t, rd.Inputs, "callbackUrl"))

	l.f.mutate("trigger", testAccount, sid, func(o map[string]any) { o["callback_url"] = "https://example.com/hook" })
	imp := l.read(sid, empty(), empty())
	require.Equal(t, sid, imp.ID)
	require.Equal(t, "sms", getS(t, imp.Inputs, "usageCategory"))
	require.Equal(t, "30.0", getS(t, imp.Inputs, "triggerValue"))
	require.Equal(t, "monthly", getS(t, imp.Inputs, "recurring"))
	require.False(t, hasProp(imp.Inputs, "triggerBy"), "Twilio's own pick is not adopted (#1679)")
	require.Equal(t, "usage", getS(t, imp.Properties, "effectiveTriggerBy"))
	require.False(t, hasProp(imp.Inputs, "callbackMethod"), "the default method is not adopted")
	requireNoChanges(t, l.diff(imp.ID, imp.Properties, l.checked(imp.Inputs), imp.Inputs))

	l.f.remove("trigger", testAccount, sid)
	require.Empty(t, l.read(sid, c.Properties, in).ID)
	l.del(sid, c.Properties)
}

// #1679: an import used to write Twilio's own triggerBy into the inputs, so the program that created the
// trigger (which never set it) planned a replacement. Whether the program omits triggerBy, states what Twilio
// chose, or states POST as the callback method, the preview is clean; a different triggerBy still replaces.
func TestUsageTriggerLC_ImportThenProgramPlansNoChange(t *testing.T) {
	l := newTL(t, "UsageTrigger", "ut")
	c := l.create(triggerInputs()) // the program never sets triggerBy; Twilio reports "usage"
	imp := l.read(c.ID, empty(), empty())
	require.False(t, hasProp(imp.Inputs, "triggerBy"))
	require.Equal(t, "usage", getS(t, imp.Properties, "effectiveTriggerBy"))

	for name, program := range map[string]property.Map{
		"omitted":              triggerInputs(),
		"states Twilio's pick": with(triggerInputs(), map[string]property.Value{"triggerBy": sv("usage")}),
		"states POST":          with(triggerInputs(), map[string]property.Value{"callbackMethod": sv("POST")}),
	} {
		requireNoChanges(t, l.diff(c.ID, imp.Properties, l.checked(program), imp.Inputs))
		_ = name
	}
	// A different triggerBy still replaces; a different callbackMethod updates in place.
	d := l.diff(c.ID, imp.Properties, l.checked(with(triggerInputs(), map[string]property.Value{"triggerBy": sv("count")})), imp.Inputs)
	require.Equal(t, pgp.UpdateReplace, d.DetailedDiff["triggerBy"].Kind)
	d = l.diff(c.ID, imp.Properties, l.checked(with(triggerInputs(), map[string]property.Value{"callbackMethod": sv("GET")})), imp.Inputs)
	require.Equal(t, pgp.Update, d.DetailedDiff["callbackMethod"].Kind)
	require.NotContains(t, d.DetailedDiff, "triggerBy")

	// A trigger created with an explicit triggerBy keeps comparing against it.
	c2 := l.create(with(triggerInputs(), map[string]property.Value{"triggerBy": sv("usage")}))
	d = l.diff(c2.ID, c2.Properties, with(triggerInputs(), map[string]property.Value{"triggerBy": sv("price")}), c2.Properties)
	require.Equal(t, pgp.UpdateReplace, d.DetailedDiff["triggerBy"].Kind)
}

func TestUsageTriggerLC_SubaccountAndChecks(t *testing.T) {
	l := newTL(t, "UsageTrigger", "ut")
	l.f.addSubaccount(fakeSub)
	in := with(triggerInputs(), map[string]property.Value{"accountSid": sv(fakeSub)})
	c := l.create(in)
	_, sid := splitScoped(c.ID)
	require.Equal(t, fakeSub+"/"+sid, c.ID)
	imp := l.read(c.ID, empty(), empty())
	require.Equal(t, fakeSub, getS(t, imp.Inputs, "accountSid"))
	l.del(c.ID, c.Properties)
	require.Nil(t, l.f.get("trigger", fakeSub, sid))

	bad := func(k, v string) {
		t.Helper()
		require.NotEmpty(t, l.check(with(triggerInputs(), map[string]property.Value{k: sv(v)})).Failures, k)
	}
	bad("callbackMethod", "PUT")
	bad("recurring", "weekly")
	bad("triggerBy", "volume")
	bad("callbackUrl", "")
	bad("usageCategory", "")
	require.Empty(t, l.check(with(triggerInputs(), map[string]property.Value{"callbackUrl": computed(), "recurring": computed()})).Failures)
	_, err := l.s.Create(pgp.CreateRequest{Urn: l.urn, Properties: triggerInputs(), DryRun: true})
	require.NoError(t, err)
}

// ------------------------------------------------------------------ lookups

func TestLookup_AccountResources(t *testing.T) {
	l := newTL(t, "Address", "a")
	ac := l.create(with(addrInputs(), map[string]property.Value{"autoCorrectAddress": bv(false)}))
	r, err := invoke(t, l, "getAddress", map[string]property.Value{"addressSid": sv(ac.ID)})
	require.NoError(t, err)
	require.Equal(t, ac.ID, getS(t, r, "sid"))
	require.Equal(t, "Portland", getS(t, r, "city"))
	require.Equal(t, "US", getS(t, r, "isoCountry"))
	require.True(t, r.Get("validated").AsBool())
	require.Equal(t, testAccount, getS(t, r, "ownerAccountSid"))
	l.del(ac.ID, ac.Properties)
	_, err = invoke(t, l, "getAddress", map[string]property.Value{"addressSid": sv(ac.ID)})
	require.ErrorContains(t, err, "was not found")
	_, err = invoke(t, l, "getAddress", map[string]property.Value{})
	require.Error(t, err)

	k := l.other("SigningKey", "k")
	kc := k.create(pv(map[string]property.Value{"friendlyName": sv("tokens")}))
	r, err = invoke(t, l, "getSigningKey", map[string]property.Value{"keySid": sv(kc.ID)})
	require.NoError(t, err)
	require.Equal(t, "tokens", getS(t, r, "friendlyName"))
	require.False(t, hasProp(r, "secret"), "the secret must not be returned")
	k.del(kc.ID, kc.Properties)
	_, err = invoke(t, l, "getSigningKey", map[string]property.Value{"keySid": sv(kc.ID)})
	require.ErrorContains(t, err, "was not found")

	q := l.other("Queue", "q")
	qc := q.create(pv(map[string]property.Value{"friendlyName": sv("support"), "maxSize": nv(25)}))
	r, err = invoke(t, l, "getQueue", map[string]property.Value{"queueSid": sv(qc.ID)})
	require.NoError(t, err)
	require.Equal(t, "support", getS(t, r, "friendlyName"))
	require.Equal(t, 25.0, r.Get("maxSize").AsNumber())
	q.del(qc.ID, qc.Properties)
	_, err = invoke(t, l, "getQueue", map[string]property.Value{"queueSid": sv(qc.ID)})
	require.ErrorContains(t, err, "was not found")

	u := l.other("UsageTrigger", "u")
	uc := u.create(triggerInputs())
	r, err = invoke(t, l, "getUsageTrigger", map[string]property.Value{"triggerSid": sv(uc.ID)})
	require.NoError(t, err)
	require.Equal(t, "sms", getS(t, r, "usageCategory"))
	require.Equal(t, "monthly", getS(t, r, "recurring"))
	require.Equal(t, "https://example.com/hook", getS(t, r, "callbackUrl"))
	u.del(uc.ID, uc.Properties)
	_, err = invoke(t, l, "getUsageTrigger", map[string]property.Value{"triggerSid": sv(uc.ID)})
	require.ErrorContains(t, err, "was not found")
}
