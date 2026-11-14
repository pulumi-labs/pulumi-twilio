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
	"strings"
	"testing"

	pgp "github.com/pulumi/pulumi-go-provider"
	"github.com/pulumi/pulumi/sdk/v3/go/property"
	"github.com/stretchr/testify/require"
)

const a2pBase = "/messaging/v1/a2p"

func TestBrandRegistrationLC_Lifecycle(t *testing.T) {
	l := newTL(t, "BrandRegistration", "brand")
	bu1, bu2 := "BU"+strings.Repeat("1", 32), "BU"+strings.Repeat("2", 32)
	in := pv(map[string]property.Value{"customerProfileBundleSid": sv(bu1), "a2pProfileBundleSid": sv(bu2), "mock": bv(true)})
	c := l.create(in)
	require.True(t, strings.HasPrefix(c.ID, "BN"), c.ID)
	form := l.f.callsMatching("POST", a2pBase+"/BrandRegistrations")[0].Form
	require.Equal(t, bu1, form.Get("CustomerProfileBundleSid"))
	require.Equal(t, bu2, form.Get("A2PProfileBundleSid"))
	require.Equal(t, "true", form.Get("Mock"))
	require.Empty(t, form["BrandType"])
	require.Empty(t, form["SkipAutomaticSecVet"])
	require.Equal(t, "PENDING", getS(t, c.Properties, "status"))
	requireNoChanges(t, l.diff(c.ID, c.Properties, in, in))

	// A refresh follows the review: the status moves, the inputs stay, nothing shows as a change.
	l.f.mutate("a2p", testAccount, "BrandRegistrations/"+c.ID, func(o map[string]any) {
		o["status"], o["tcr_id"], o["brand_score"] = "APPROVED", "BXXXXXX", float64(42)
		o["errors"] = []any{map[string]any{"code": "E1"}}
	})
	rd := l.read(c.ID, c.Properties, in)
	require.Equal(t, "APPROVED", getS(t, rd.Properties, "status"))
	require.Equal(t, "BXXXXXX", getS(t, rd.Properties, "tcrId"))
	require.Equal(t, 42.0, rd.Properties.Get("brandScore").AsNumber())
	require.Equal(t, `{"code":"E1"}`, rd.Properties.Get("errors").AsArray().Get(0).AsString())
	require.False(t, hasProp(rd.Inputs, "brandType"))
	requireNoChanges(t, l.diff(c.ID, rd.Properties, in, rd.Inputs))

	// Stating the defaults is no change; every real change replaces.
	same := with(in, map[string]property.Value{"skipAutomaticSecVet": bv(false), "brandType": sv("STANDARD")})
	d := l.diff(c.ID, rd.Properties, with(same, map[string]property.Value{"brandType": sv("SOLE_PROPRIETOR")}), rd.Inputs)
	require.Equal(t, pgp.UpdateReplace, d.DetailedDiff["brandType"].Kind)
	d = l.diff(c.ID, rd.Properties, with(in, map[string]property.Value{"mock": bv(false)}), rd.Inputs)
	require.Equal(t, pgp.UpdateReplace, d.DetailedDiff["mock"].Kind)
	d = l.diff(c.ID, rd.Properties, with(in, map[string]property.Value{"a2pProfileBundleSid": sv(bu1)}), rd.Inputs)
	require.Equal(t, pgp.UpdateReplace, d.DetailedDiff["a2pProfileBundleSid"].Kind)

	// Import adopts what Twilio holds.
	imp := l.read(c.ID, empty(), empty())
	require.Empty(t, l.check(imp.Inputs).Failures)
	require.Equal(t, bu1, getS(t, imp.Inputs, "customerProfileBundleSid"))
	require.True(t, imp.Inputs.Get("mock").AsBool() || !hasProp(imp.Inputs, "mock"))
	l.requireImportFails("BN" + strings.Repeat("0", 32))

	// Destroy only forgets the brand: Twilio has no delete, so no request is sent.
	l.f.resetCalls()
	l.del(c.ID, rd.Properties)
	require.Zero(t, l.f.callCount())

	// Deleted outside Pulumi reads as gone.
	l.f.remove("a2p", testAccount, "BrandRegistrations/"+c.ID)
	require.Empty(t, l.read(c.ID, rd.Properties, in).ID)

	require.Len(t, l.check(pv(map[string]property.Value{"customerProfileBundleSid": sv("x"), "a2pProfileBundleSid": sv(bu2)})).Failures, 1)
}

func TestLookup_BrandRegistrations(t *testing.T) {
	l := newTL(t, "BrandRegistration", "brand")
	mk := func(i string) string {
		return l.other("BrandRegistration", "b"+i).create(pv(map[string]property.Value{
			"customerProfileBundleSid": sv("BU" + strings.Repeat(i, 32)), "a2pProfileBundleSid": sv("BU" + strings.Repeat("9", 32))})).ID
	}
	b1, _, _ := mk("1"), mk("2"), mk("3")
	r, err := invoke(t, l, "getBrandRegistration", map[string]property.Value{"sid": sv(b1)})
	require.NoError(t, err)
	require.Equal(t, "PENDING", getS(t, r, "status"))
	require.Equal(t, "STANDARD", getS(t, r, "brandType"))
	require.False(t, r.Get("mock").AsBool())
	_, err = invoke(t, l, "getBrandRegistration", map[string]property.Value{"sid": sv("BNmissing")})
	require.ErrorContains(t, err, "was not found")
	_, err = invoke(t, l, "getBrandRegistration", map[string]property.Value{"sid": sv("")})
	require.ErrorContains(t, err, "must be set")
	r, err = invoke(t, l, "getBrandRegistrations", map[string]property.Value{})
	require.NoError(t, err)
	require.Len(t, r.Get("brandRegistrations").AsArray().AsSlice(), 3)
}

func a2pMakeBrand(t *testing.T, l *tl, name string) string {
	return l.other("BrandRegistration", "brand-"+name).create(pv(map[string]property.Value{
		"customerProfileBundleSid": sv("BU" + strings.Repeat("1", 32)), "a2pProfileBundleSid": sv("BU" + strings.Repeat("2", 32)), "mock": bv(true)})).ID
}

func a2pMakeService(t *testing.T, l *tl, name string) string {
	return l.other("MessagingService", "svc-"+name).create(pv(map[string]property.Value{"friendlyName": sv(name)})).ID
}

const (
	a2pDesc = "Sends one time passcodes to customers who sign in to the web shop."
	a2pFlow = "Customers enter their phone number on the sign in page and tick the consent box."
)

func TestUsAppToPersonLC_Lifecycle(t *testing.T) {
	l := newTL(t, "UsAppToPerson", "camp")
	brand, svc := a2pMakeBrand(t, l, "c"), a2pMakeService(t, l, "c")
	in := pv(map[string]property.Value{"messagingServiceSid": sv(svc), "brandRegistrationSid": sv(brand), "usAppToPersonUsecase": sv("2FA"),
		"description": sv(a2pDesc), "messageFlow": sv(a2pFlow), "hasEmbeddedLinks": bv(false), "hasEmbeddedPhone": bv(false),
		"messageSamples": arrv(sv("Your code is 123456, valid for ten minutes."), sv("123456 is your verification code for the web shop.")),
		"optInKeywords":  arrv(sv("START"), sv("YES"))})
	c := l.create(in)
	require.True(t, strings.HasPrefix(c.ID, svc+"/QE"), c.ID)
	sid := strings.TrimPrefix(c.ID, svc+"/")
	cf := l.f.callsMatching("POST", "/messaging/v1/Services/"+svc+"/Compliance/Usa2p")[0].Form
	require.Equal(t, brand, cf.Get("BrandRegistrationSid"))
	require.Equal(t, "2FA", cf.Get("UsAppToPersonUsecase"))
	require.Len(t, cf["MessageSamples"], 2)
	require.Equal(t, []string{"START", "YES"}, cf["OptInKeywords"])
	require.Equal(t, "false", cf.Get("HasEmbeddedLinks"))
	require.Empty(t, cf["AgeGated"])
	require.Equal(t, "IN_PROGRESS", getS(t, c.Properties, "campaignStatus"))
	requireNoChanges(t, l.diff(c.ID, c.Properties, in, in))

	// A refresh follows the review and adopts what Twilio reports for inputs the program left out.
	l.f.mutate("usa2p", svc, sid, func(o map[string]any) {
		o["campaign_status"], o["campaign_id"] = "VERIFIED", "CXXXXXX"
		o["rate_limits"] = map[string]any{"att": map[string]any{"class": "A"}}
		o["opt_out_message"] = "Reply STOP to stop."
		o["opt_in_keywords"] = []any{"YES", "START"}
	})
	rd := l.read(c.ID, c.Properties, in)
	require.Equal(t, "VERIFIED", getS(t, rd.Properties, "campaignStatus"))
	require.Equal(t, `{"att":{"class":"A"}}`, getS(t, rd.Properties, "rateLimits"))
	require.Equal(t, "Reply STOP to stop.", getS(t, rd.Inputs, "optOutMessage"))
	requireNoChanges(t, l.diff(c.ID, rd.Properties, in, rd.Inputs))
	// Stating the value Twilio holds (or the age gate default) is no change.
	same := with(in, map[string]property.Value{"optOutMessage": sv("Reply STOP to stop."), "ageGated": bv(false), "directLending": bv(false)})
	requireNoChanges(t, l.diff(c.ID, rd.Properties, same, rd.Inputs))

	// The description, flow, samples, flags and policy URLs update in place; the update sends the age gate and direct lending flag.
	next := with(in, map[string]property.Value{"description": sv(a2pDesc + " Updated."), "hasEmbeddedLinks": bv(true),
		"privacyPolicyUrl": sv("https://example.com/privacy"), "ageGated": bv(true),
		"messageSamples": arrv(sv("Your code is 654321, valid for ten minutes."), sv("654321 is your verification code for the web shop."))})
	d := l.diff(c.ID, rd.Properties, next, rd.Inputs)
	for _, k := range []string{"description", "hasEmbeddedLinks", "privacyPolicyUrl", "ageGated", "messageSamples"} {
		require.Equal(t, pgp.Update, d.DetailedDiff[k].Kind, k)
	}
	l.f.resetCalls()
	u := l.update(c.ID, rd.Properties, next, rd.Inputs)
	uf := l.f.callsMatching("POST", "/messaging/v1/Services/"+svc+"/Compliance/Usa2p/"+sid)[0].Form
	require.Equal(t, "true", uf.Get("HasEmbeddedLinks"))
	require.Equal(t, "true", uf.Get("AgeGated"))
	require.Equal(t, "false", uf.Get("DirectLending"))
	require.Len(t, uf["MessageSamples"], 2)
	require.Empty(t, uf["BrandRegistrationSid"])
	require.Empty(t, uf["UsAppToPersonUsecase"])
	require.Empty(t, uf["OptInKeywords"])
	rd = l.read(c.ID, u.Properties, next)
	requireNoChanges(t, l.diff(c.ID, rd.Properties, next, rd.Inputs))

	// Everything fixed at registration replaces.
	for k, v := range map[string]property.Value{"usAppToPersonUsecase": sv("MARKETING"), "optInMessage": sv("Welcome"),
		"optInKeywords": arrv(sv("GO")), "subscriberOptIn": bv(true), "brandRegistrationSid": sv("BN" + strings.Repeat("7", 32))} {
		d = l.diff(c.ID, rd.Properties, with(next, map[string]property.Value{k: v}), rd.Inputs)
		require.Equal(t, pgp.UpdateReplace, d.DetailedDiff[k].Kind, k)
	}

	// Import adopts what Twilio holds and the first preview plans no change.
	imp := l.read(c.ID, empty(), empty())
	require.Empty(t, l.check(imp.Inputs).Failures)
	require.Equal(t, svc, getS(t, imp.Inputs, "messagingServiceSid"))
	require.Equal(t, "2FA", getS(t, imp.Inputs, "usAppToPersonUsecase"))
	requireNoChanges(t, l.diff(c.ID, imp.Properties, imp.Inputs, imp.Inputs))
	l.requireImportFails(svc + "/QE" + strings.Repeat("0", 32))

	// Destroy deletes; a campaign deleted outside Pulumi reads as gone and destroys cleanly.
	l.f.resetCalls()
	l.del(c.ID, u.Properties)
	require.Len(t, l.f.callsMatching("DELETE", "/messaging/v1/Services/"+svc+"/Compliance/Usa2p/"+sid), 1)
	require.Empty(t, l.read(c.ID, u.Properties, next).ID)
	l.del(c.ID, u.Properties)

	bad := l.check(pv(map[string]property.Value{"messagingServiceSid": sv("x"), "brandRegistrationSid": sv("y"), "usAppToPersonUsecase": sv(""),
		"description": sv(""), "messageFlow": sv(""), "hasEmbeddedLinks": bv(false), "hasEmbeddedPhone": bv(false), "messageSamples": arrv()}))
	require.Len(t, bad.Failures, 6)
}

func TestLookup_UsAppToPerson(t *testing.T) {
	l := newTL(t, "UsAppToPerson", "camp")
	brand, svc, other := a2pMakeBrand(t, l, "l"), a2pMakeService(t, l, "l1"), a2pMakeService(t, l, "l2")
	mk := func(s, desc string) string {
		return strings.TrimPrefix(l.other("UsAppToPerson", "c-"+desc).create(pv(map[string]property.Value{"messagingServiceSid": sv(s),
			"brandRegistrationSid": sv(brand), "usAppToPersonUsecase": sv("2FA"), "description": sv(a2pDesc + desc), "messageFlow": sv(a2pFlow),
			"hasEmbeddedLinks": bv(false), "hasEmbeddedPhone": bv(true), "messageSamples": arrv(sv("Your code is 123456, valid for ten minutes."))})).ID, s+"/")
	}
	c1 := mk(svc, "1")
	mk(svc, "2")
	mk(svc, "3")
	mk(other, "4")
	r, err := invoke(t, l, "getUsAppToPerson", map[string]property.Value{"messagingServiceSid": sv(svc), "sid": sv(c1)})
	require.NoError(t, err)
	require.Equal(t, "IN_PROGRESS", getS(t, r, "campaignStatus"))
	require.Equal(t, svc, getS(t, r, "messagingServiceSid"))
	require.True(t, r.Get("hasEmbeddedPhone").AsBool())
	require.Equal(t, 1, r.Get("messageSamples").AsArray().Len())
	_, err = invoke(t, l, "getUsAppToPerson", map[string]property.Value{"messagingServiceSid": sv(svc), "sid": sv("QEmissing")})
	require.ErrorContains(t, err, "was not found")
	_, err = invoke(t, l, "getUsAppToPerson", map[string]property.Value{"messagingServiceSid": sv(svc), "sid": sv("")})
	require.ErrorContains(t, err, "must be set")
	r, err = invoke(t, l, "getUsAppToPersons", map[string]property.Value{"messagingServiceSid": sv(svc)})
	require.NoError(t, err)
	require.Len(t, r.Get("usAppToPersons").AsArray().AsSlice(), 3)
	r, err = invoke(t, l, "getUsAppToPersonUsecases", map[string]property.Value{"messagingServiceSid": sv(svc), "brandRegistrationSid": sv(brand)})
	require.NoError(t, err)
	uc := r.Get("usecases").AsArray()
	require.Equal(t, 2, uc.Len())
	require.Equal(t, "2FA", uc.Get(0).AsMap().Get("code").AsString())
	require.True(t, uc.Get(1).AsMap().Get("postApprovalRequired").AsBool())
	require.NotEmpty(t, l.f.callsMatching("GET", "/Compliance/Usa2p/Usecases")[0].Query.Get("BrandRegistrationSid"))
}

func tollfreeInputs() property.Map {
	return pv(map[string]property.Value{"tollfreePhoneNumberSid": sv("PN" + strings.Repeat("a", 32)), "businessName": sv("Acme Corp"),
		"businessWebsite": sv("https://acme.example.com"), "notificationEmail": sv("ops@acme.example.com"),
		"useCaseCategories": arrv(sv("2FA")), "useCaseSummary": sv("One time passcodes for sign in."),
		"productionMessageSample": sv("Your Acme code is 123456."), "optInImageUrls": arrv(sv("https://acme.example.com/optin.png")),
		"optInType": sv("WEB_FORM"), "messageVolume": sv("1,000"), "businessCountry": sv("US"), "externalReferenceId": sv("acme-1")})
}

func TestTollfreeVerificationLC_Lifecycle(t *testing.T) {
	l := newTL(t, "TollfreeVerification", "tfv")
	in := tollfreeInputs()
	c := l.create(in)
	require.True(t, strings.HasPrefix(c.ID, "HH"), c.ID)
	cf := l.f.callsMatching("POST", "/messaging/v1/Tollfree/Verifications")[0].Form
	require.Equal(t, "PN"+strings.Repeat("a", 32), cf.Get("TollfreePhoneNumberSid"))
	require.Equal(t, []string{"2FA"}, cf["UseCaseCategories"])
	require.Equal(t, "acme-1", cf.Get("ExternalReferenceId"))
	require.Empty(t, cf["EditReason"])
	require.Empty(t, cf["AgeGatedContent"])
	require.Equal(t, "PENDING_REVIEW", getS(t, c.Properties, "status"))
	requireNoChanges(t, l.diff(c.ID, c.Properties, in, in))

	// A refresh follows the review: the status moves and the program's inputs stay.
	l.f.mutate("tfv", testAccount, c.ID, func(o map[string]any) {
		o["status"], o["rejection_reason"], o["edit_allowed"] = "TWILIO_REJECTED", "Website unreachable", true
		o["rejection_reasons"] = []any{map[string]any{"code": 1}}
		o["business_city"] = "Springfield"
	})
	rd := l.read(c.ID, c.Properties, in)
	require.Equal(t, "TWILIO_REJECTED", getS(t, rd.Properties, "status"))
	require.True(t, rd.Properties.Get("editAllowed").AsBool())
	require.Equal(t, `{"code":1}`, rd.Properties.Get("rejectionReasons").AsArray().Get(0).AsString())
	require.False(t, hasProp(rd.Inputs, "businessCity")) // an input the program leaves out is not adopted
	requireNoChanges(t, l.diff(c.ID, rd.Properties, in, rd.Inputs))
	stated := with(in, map[string]property.Value{"businessCity": sv("Shelbyville")})
	require.Equal(t, pgp.Update, l.diff(c.ID, rd.Properties, stated, rd.Inputs).DetailedDiff["businessCity"].Kind)
	rd = l.read(c.ID, c.Properties, with(in, map[string]property.Value{"businessCity": sv("Shelbyville")}))
	require.Equal(t, "Springfield", getS(t, rd.Inputs, "businessCity")) // an input the program states follows Twilio
	rd = l.read(c.ID, c.Properties, in)

	// An edit of a rejected verification updates in place, sends the edit reason and not the create-only fields.
	next := with(in, map[string]property.Value{"businessWebsite": sv("https://acme.example.org"), "useCaseCategories": arrv(sv("2FA"), sv("ACCOUNT_NOTIFICATIONS")),
		"ageGatedContent": bv(true), "editReason": sv("Website fixed"), "optInKeywords": arrv(sv("GO"))})
	d := l.diff(c.ID, rd.Properties, next, rd.Inputs)
	for _, k := range []string{"businessWebsite", "useCaseCategories", "ageGatedContent", "optInKeywords"} {
		require.Equal(t, pgp.Update, d.DetailedDiff[k].Kind, k)
	}
	require.NotContains(t, d.DetailedDiff, "editReason")
	l.f.resetCalls()
	u := l.update(c.ID, rd.Properties, next, rd.Inputs)
	uf := l.f.callsMatching("POST", "/messaging/v1/Tollfree/Verifications/"+c.ID)[0].Form
	require.Equal(t, "Website fixed", uf.Get("EditReason"))
	require.Equal(t, "https://acme.example.org", uf.Get("BusinessWebsite"))
	require.Equal(t, []string{"2FA", "ACCOUNT_NOTIFICATIONS"}, uf["UseCaseCategories"])
	require.Empty(t, uf["TollfreePhoneNumberSid"])
	require.Empty(t, uf["ExternalReferenceId"])
	require.Equal(t, "PENDING_REVIEW", getS(t, u.Properties, "status"))

	// The number and the external reference replace.
	d = l.diff(c.ID, u.Properties, with(next, map[string]property.Value{"tollfreePhoneNumberSid": sv("PN" + strings.Repeat("b", 32))}), next)
	require.Equal(t, pgp.UpdateReplace, d.DetailedDiff["tollfreePhoneNumberSid"].Kind)
	d = l.diff(c.ID, u.Properties, with(next, map[string]property.Value{"externalReferenceId": sv("other")}), next)
	require.Equal(t, pgp.UpdateReplace, d.DetailedDiff["externalReferenceId"].Kind)

	// An edit is refused by Twilio once the verification is no longer editable.
	l.f.mutate("tfv", testAccount, c.ID, func(o map[string]any) { o["edit_allowed"] = false })
	_, err := l.updateErr(c.ID, u.Properties, with(next, map[string]property.Value{"businessName": sv("Acme Inc")}), next)
	require.ErrorContains(t, err, "editAllowed")

	// Import adopts what Twilio holds and the first preview plans no change.
	imp := l.read(c.ID, empty(), empty())
	require.Empty(t, l.check(imp.Inputs).Failures)
	require.Equal(t, "Acme Corp", getS(t, imp.Inputs, "businessName"))
	require.Equal(t, "PN"+strings.Repeat("a", 32), getS(t, imp.Inputs, "tollfreePhoneNumberSid"))
	requireNoChanges(t, l.diff(c.ID, imp.Properties, imp.Inputs, imp.Inputs))
	l.requireImportFails("HH" + strings.Repeat("0", 32))

	// Destroy deletes; a verification deleted outside Pulumi reads as gone.
	l.f.resetCalls()
	l.del(c.ID, u.Properties)
	require.Len(t, l.f.callsMatching("DELETE", "/messaging/v1/Tollfree/Verifications/"+c.ID), 1)
	require.Empty(t, l.read(c.ID, u.Properties, next).ID)
	l.del(c.ID, u.Properties)

	bad := l.check(with(in, map[string]property.Value{"tollfreePhoneNumberSid": sv("x"), "businessName": sv(""), "useCaseCategories": arrv()}))
	require.Len(t, bad.Failures, 3)
}

func TestLookup_TollfreeVerifications(t *testing.T) {
	l := newTL(t, "TollfreeVerification", "tfv")
	mk := func(n string, c string) string {
		return l.other("TollfreeVerification", "v"+n).create(with(tollfreeInputs(), map[string]property.Value{
			"tollfreePhoneNumberSid": sv("PN" + strings.Repeat(c, 32)), "externalReferenceId": sv("ref-" + n)})).ID
	}
	v1 := mk("1", "1")
	mk("2", "2")
	mk("3", "3")
	l.f.mutate("tfv", testAccount, v1, func(o map[string]any) { o["status"] = "TWILIO_APPROVED" })
	r, err := invoke(t, l, "getTollfreeVerification", map[string]property.Value{"sid": sv(v1)})
	require.NoError(t, err)
	require.Equal(t, "TWILIO_APPROVED", getS(t, r, "status"))
	require.Equal(t, "Acme Corp", getS(t, r, "businessName"))
	require.Equal(t, "2FA", r.Get("useCaseCategories").AsArray().Get(0).AsString())
	_, err = invoke(t, l, "getTollfreeVerification", map[string]property.Value{"sid": sv("HHmissing")})
	require.ErrorContains(t, err, "was not found")
	_, err = invoke(t, l, "getTollfreeVerification", map[string]property.Value{"sid": sv("")})
	require.ErrorContains(t, err, "must be set")
	r, err = invoke(t, l, "getTollfreeVerifications", map[string]property.Value{})
	require.NoError(t, err)
	require.Len(t, r.Get("verifications").AsArray().AsSlice(), 3)
	r, err = invoke(t, l, "getTollfreeVerifications", map[string]property.Value{"status": sv("TWILIO_APPROVED"), "includeSubAccounts": bv(true)})
	require.NoError(t, err)
	require.Len(t, r.Get("verifications").AsArray().AsSlice(), 1)
	require.Equal(t, "true", l.f.callsMatching("GET", "/Tollfree/Verifications")[len(l.f.callsMatching("GET", "/Tollfree/Verifications"))-1].Query.Get("IncludeSubAccounts"))
}

func (l *tl) updateErr(id string, state, inputs, oldInputs property.Map) (pgp.UpdateResponse, error) {
	l.t.Helper()
	return l.s.Update(pgp.UpdateRequest{ID: id, Urn: l.urn, State: state, Inputs: l.checked(inputs), OldInputs: oldInputs})
}
