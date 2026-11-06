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
	"github.com/stretchr/testify/require"
)

func TestSubaccount_Lifecycle(t *testing.T) {
	m, srv := newMock(t)
	s := newServerFor(t, srv.URL)
	urn := twilioURN("Subaccount", "sub")
	in := map[string]any{"friendlyName": "tenant-a"}

	// Check applies the active default.
	chk, err := s.Check(pgp.CheckRequest{Urn: urn, Inputs: pm(in)})
	require.NoError(t, err)
	require.Empty(t, chk.Failures)
	require.Equal(t, "active", str(chk.Inputs, "status"))
	in = map[string]any{"friendlyName": "tenant-a", "status": "active"}

	_, err = s.Create(pgp.CreateRequest{Urn: urn, Properties: pm(in), DryRun: true})
	require.NoError(t, err)
	require.Empty(t, m.callLog(), "preview must not call the API")

	cr, err := s.Create(pgp.CreateRequest{Urn: urn, Properties: pm(in)})
	require.NoError(t, err)
	require.Equal(t, "AC1", cr.ID)
	tok, ok := cr.Properties.GetOk("authToken")
	require.True(t, ok)
	require.Equal(t, "tok-1", tok.WithSecret(false).AsString())
	require.Equal(t, "tenant-a", m.forms["POST /api/2010-04-01/Accounts.json"].Get("FriendlyName"))

	// Suspend in place.
	in2 := map[string]any{"friendlyName": "tenant-a", "status": "suspended"}
	ur, err := s.Update(pgp.UpdateRequest{ID: cr.ID, Urn: urn, State: cr.Properties, Inputs: pm(in2), OldInputs: pm(in)})
	require.NoError(t, err)
	require.Equal(t, "suspended", str(ur.Properties, "status"))
	require.Equal(t, "suspended", m.get("/api/2010-04-01/Accounts/AC1")["status"])

	// Read reflects drift.
	m.get("/api/2010-04-01/Accounts/AC1")["friendly_name"] = "renamed-outside"
	rr, err := s.Read(pgp.ReadRequest{ID: cr.ID, Urn: urn, Properties: ur.Properties, Inputs: pm(in2)})
	require.NoError(t, err)
	require.Equal(t, "renamed-outside", str(rr.Inputs, "friendlyName"))

	// Delete closes (it cannot truly delete); Read then reports gone.
	require.NoError(t, s.Delete(pgp.DeleteRequest{ID: cr.ID, Urn: urn, Properties: ur.Properties}))
	require.Equal(t, "closed", m.get("/api/2010-04-01/Accounts/AC1")["status"])
	rr, err = s.Read(pgp.ReadRequest{ID: cr.ID, Urn: urn, Properties: ur.Properties, Inputs: pm(in2)})
	require.NoError(t, err)
	require.Empty(t, rr.ID)
	require.True(t, m.authSeen)
}

func TestSubaccount_CheckRejectsBadInputs(t *testing.T) {
	_, srv := newMock(t)
	s := newServerFor(t, srv.URL)
	for name, in := range map[string]map[string]any{
		"empty name": {"friendlyName": " "},
		"bad status": {"friendlyName": "x", "status": "closed"},
	} {
		t.Run(name, func(t *testing.T) {
			r, err := s.Check(pgp.CheckRequest{Urn: twilioURN("Subaccount", "x"), Inputs: pm(in)})
			require.NoError(t, err)
			require.NotEmpty(t, r.Failures)
		})
	}
}

func TestApiKey_RestrictedLifecycle(t *testing.T) {
	m, srv := newMock(t)
	s := newServerFor(t, srv.URL)
	urn := twilioURN("ApiKey", "k")
	in := map[string]any{"friendlyName": "ci", "keyType": "restricted", "policy": `{"allow":["/twilio/iam/api-keys/create"]}`}
	cr, err := s.Create(pgp.CreateRequest{Urn: urn, Properties: pm(in)})
	require.NoError(t, err)
	require.Equal(t, "SK1", cr.ID)
	sec, _ := cr.Properties.GetOk("secret")
	require.Equal(t, "secret-1", sec.WithSecret(false).AsString())
	f := m.forms["POST /iam/v1/Keys"]
	require.Equal(t, testAccount, f.Get("AccountSid"))
	require.Equal(t, "restricted", f.Get("KeyType"))
	require.JSONEq(t, in["policy"].(string), f.Get("Policy"))

	// Rename and change policy in place; the secret survives.
	in2 := map[string]any{"friendlyName": "ci2", "keyType": "restricted", "policy": `{"allow":["/twilio/iam/api-keys/update"]}`}
	ur, err := s.Update(pgp.UpdateRequest{ID: cr.ID, Urn: urn, State: cr.Properties, Inputs: pm(in2), OldInputs: pm(in)})
	require.NoError(t, err)
	sec, _ = ur.Properties.GetOk("secret")
	require.Equal(t, "secret-1", sec.WithSecret(false).AsString())
	require.Equal(t, "ci2", m.get("/iam/v1/Keys/SK1")["friendly_name"])

	// Read keeps the stored secret and tolerates the API hiding it.
	rr, err := s.Read(pgp.ReadRequest{ID: cr.ID, Urn: urn, Properties: ur.Properties, Inputs: pm(in2)})
	require.NoError(t, err)
	sec, _ = rr.Properties.GetOk("secret")
	require.Equal(t, "secret-1", sec.WithSecret(false).AsString())

	require.NoError(t, s.Delete(pgp.DeleteRequest{ID: cr.ID, Urn: urn, Properties: ur.Properties}))
	require.Nil(t, m.get("/iam/v1/Keys/SK1"))
	// Deleting again (gone out of band) is not an error.
	require.NoError(t, s.Delete(pgp.DeleteRequest{ID: cr.ID, Urn: urn, Properties: ur.Properties}))
}

func TestApiKey_CheckPolicyRules(t *testing.T) {
	_, srv := newMock(t)
	s := newServerFor(t, srv.URL)
	for name, in := range map[string]map[string]any{
		"restricted needs policy": {"friendlyName": "x", "keyType": "restricted"},
		"policy needs restricted": {"friendlyName": "x", "policy": "{}"},
		"policy must be json":     {"friendlyName": "x", "keyType": "restricted", "policy": "nope"},
		"unknown key type":        {"friendlyName": "x", "keyType": "master"},
	} {
		t.Run(name, func(t *testing.T) {
			r, err := s.Check(pgp.CheckRequest{Urn: twilioURN("ApiKey", "x"), Inputs: pm(in)})
			require.NoError(t, err)
			require.NotEmpty(t, r.Failures)
		})
	}
	r, err := s.Check(pgp.CheckRequest{Urn: twilioURN("ApiKey", "ok"), Inputs: pm(map[string]any{"friendlyName": "x"})})
	require.NoError(t, err)
	require.Empty(t, r.Failures)
	require.Equal(t, "standard", str(r.Inputs, "keyType"))
}

func seedNumber(m *mockTwilio) {
	m.seed("/api/2010-04-01/Accounts/"+testAccount+"/IncomingPhoneNumbers/PN900", map[string]any{
		"sid": "PN900", "phone_number": "+14155550100", "friendly_name": "main line",
		"voice_url": "https://old.example.com/voice", "sms_url": "https://old.example.com/sms",
		"capabilities": map[string]any{"voice": true, "sms": true, "mms": false},
	})
}

func TestIncomingPhoneNumber_AdoptConfigureAndKeepOnDestroy(t *testing.T) {
	m, srv := newMock(t)
	seedNumber(m)
	s := newServerFor(t, srv.URL)
	urn := twilioURN("IncomingPhoneNumber", "num")
	path := "/api/2010-04-01/Accounts/" + testAccount + "/IncomingPhoneNumbers/PN900"
	in := map[string]any{"phoneNumber": "+14155550100", "voiceUrl": "https://example.com/voice", "smsUrl": "https://example.com/sms", "smsMethod": "POST"}

	cr, err := s.Create(pgp.CreateRequest{Urn: urn, Properties: pm(in)})
	require.NoError(t, err)
	require.Equal(t, "PN900", cr.ID)
	require.Equal(t, "+14155550100", str(cr.Properties, "number"))
	require.Equal(t, "https://example.com/voice", m.get(path)["voice_url"])
	require.Equal(t, "POST", m.get(path)["sms_method"])

	// Dropping smsUrl from the program clears it on Twilio; the dropped method goes back to
	// Twilio's default POST (the real API rejects an empty method).
	in2 := map[string]any{"phoneNumber": "+14155550100", "voiceUrl": "https://example.com/voice"}
	ur, err := s.Update(pgp.UpdateRequest{ID: cr.ID, Urn: urn, State: cr.Properties, Inputs: pm(in2), OldInputs: pm(in)})
	require.NoError(t, err)
	require.Equal(t, "", m.get(path)["sms_url"])
	require.Equal(t, "POST", m.get(path)["sms_method"])

	// Drift on a managed field is surfaced; unmanaged fields are not adopted.
	m.get(path)["voice_url"] = "https://drift.example.com"
	rr, err := s.Read(pgp.ReadRequest{ID: cr.ID, Urn: urn, Properties: ur.Properties, Inputs: pm(in2)})
	require.NoError(t, err)
	require.Equal(t, "https://drift.example.com", str(rr.Inputs, "voiceUrl"))
	_, hasSms := rr.Inputs.GetOk("smsUrl")
	require.False(t, hasSms)

	// Destroy never releases the number.
	before := len(m.callLog())
	require.NoError(t, s.Delete(pgp.DeleteRequest{ID: cr.ID, Urn: urn, Properties: ur.Properties}))
	require.Equal(t, before, len(m.callLog()), "delete must make no API call")
	require.NotNil(t, m.get(path))
}

func TestIncomingPhoneNumber_ImportAdoptsEverything(t *testing.T) {
	m, srv := newMock(t)
	seedNumber(m)
	s := newServerFor(t, srv.URL)
	rr, err := s.Read(pgp.ReadRequest{ID: "PN900", Urn: twilioURN("IncomingPhoneNumber", "imp")})
	require.NoError(t, err)
	require.Equal(t, "PN900", rr.ID)
	require.Equal(t, "PN900", str(rr.Inputs, "phoneNumberSid"))
	require.Equal(t, "https://old.example.com/voice", str(rr.Inputs, "voiceUrl"))
	require.Equal(t, "main line", str(rr.Inputs, "friendlyName"))
}

func TestIncomingPhoneNumber_UnownedNumberFails(t *testing.T) {
	_, srv := newMock(t)
	s := newServerFor(t, srv.URL)
	_, err := s.Create(pgp.CreateRequest{Urn: twilioURN("IncomingPhoneNumber", "n"), Properties: pm(map[string]any{"phoneNumber": "+19998887777"})})
	require.ErrorContains(t, err, "does not own")
	_, err = s.Create(pgp.CreateRequest{Urn: twilioURN("IncomingPhoneNumber", "n"), Properties: pm(map[string]any{"phoneNumberSid": "PNnone"})})
	require.ErrorContains(t, err, "does not own")
}

func TestIncomingPhoneNumber_CheckNeedsExactlyOneIdentity(t *testing.T) {
	_, srv := newMock(t)
	s := newServerFor(t, srv.URL)
	for name, in := range map[string]map[string]any{
		"none":       {"friendlyName": "x"},
		"both":       {"phoneNumber": "+1415", "phoneNumberSid": "PN1"},
		"bad method": {"phoneNumber": "+1415", "voiceMethod": "PUT"},
	} {
		t.Run(name, func(t *testing.T) {
			r, err := s.Check(pgp.CheckRequest{Urn: twilioURN("IncomingPhoneNumber", "x"), Inputs: pm(in)})
			require.NoError(t, err)
			require.NotEmpty(t, r.Failures)
		})
	}
}

func TestApplication_LifecycleAndSubaccountPath(t *testing.T) {
	m, srv := newMock(t)
	s := newServerFor(t, srv.URL)
	urn := twilioURN("Application", "app")
	in := map[string]any{"friendlyName": "ivr", "voiceUrl": "https://example.com/voice", "voiceMethod": "GET", "smsUrl": "https://example.com/sms"}
	cr, err := s.Create(pgp.CreateRequest{Urn: urn, Properties: pm(in)})
	require.NoError(t, err)
	require.Equal(t, "AP1", cr.ID)
	require.Equal(t, testAccount, str(cr.Properties, "ownerAccountSid"))
	path := "/api/2010-04-01/Accounts/" + testAccount + "/Applications/AP1"
	require.Equal(t, "GET", m.get(path)["voice_method"])

	in2 := map[string]any{"friendlyName": "ivr-2", "voiceUrl": "https://example.com/voice"}
	ur, err := s.Update(pgp.UpdateRequest{ID: cr.ID, Urn: urn, State: cr.Properties, Inputs: pm(in2), OldInputs: pm(in)})
	require.NoError(t, err)
	require.Equal(t, "ivr-2", m.get(path)["friendly_name"])
	require.Equal(t, "", m.get(path)["sms_url"], "removed fields are cleared")

	require.NoError(t, s.Delete(pgp.DeleteRequest{ID: cr.ID, Urn: urn, Properties: ur.Properties}))
	require.Nil(t, m.get(path))
	rr, err := s.Read(pgp.ReadRequest{ID: cr.ID, Urn: urn, Properties: ur.Properties})
	require.NoError(t, err)
	require.Empty(t, rr.ID)

	// accountSid targets a subaccount path with the parent's credentials.
	in3 := map[string]any{"friendlyName": "sub-app", "accountSid": "ACsub"}
	cr, err = s.Create(pgp.CreateRequest{Urn: twilioURN("Application", "sub"), Properties: pm(in3)})
	require.NoError(t, err)
	require.Equal(t, "ACsub", str(cr.Properties, "ownerAccountSid"))
	_, subSid := splitScoped(cr.ID)
	require.Equal(t, "ACsub/"+subSid, cr.ID, "an object in a subaccount gets a scoped, importable ID")
	require.NotNil(t, m.get("/api/2010-04-01/Accounts/ACsub/Applications/"+subSid))
}

func TestMessagingServiceAndSenderPool(t *testing.T) {
	m, srv := newMock(t)
	s := newServerFor(t, srv.URL)
	svcURN, pnURN := twilioURN("MessagingService", "svc"), twilioURN("MessagingServicePhoneNumber", "sender")
	in := map[string]any{"friendlyName": "alerts", "inboundRequestUrl": "https://example.com/in", "stickySender": true, "validityPeriod": 600}
	cr, err := s.Create(pgp.CreateRequest{Urn: svcURN, Properties: pm(in)})
	require.NoError(t, err)
	f := m.forms["POST /messaging/v1/Services"]
	require.Equal(t, "true", f.Get("StickySender"))
	require.Equal(t, "600", f.Get("ValidityPeriod"))
	require.Equal(t, "https://example.com/in", f.Get("InboundRequestUrl"))

	in2 := map[string]any{"friendlyName": "alerts-2", "stickySender": true, "validityPeriod": 600}
	ur, err := s.Update(pgp.UpdateRequest{ID: cr.ID, Urn: svcURN, State: cr.Properties, Inputs: pm(in2), OldInputs: pm(in)})
	require.NoError(t, err)
	svcPath := "/messaging/v1/Services/" + cr.ID
	require.Equal(t, "alerts-2", m.get(svcPath)["friendly_name"])
	require.Equal(t, "", m.get(svcPath)["inbound_request_url"])

	pn, err := s.Create(pgp.CreateRequest{Urn: pnURN, Properties: pm(map[string]any{"serviceSid": cr.ID, "phoneNumberSid": "PN900"})})
	require.NoError(t, err)
	require.Equal(t, cr.ID+"/PN900", pn.ID)
	require.Equal(t, "+15550001111", str(pn.Properties, "phoneNumber"))
	rr, err := s.Read(pgp.ReadRequest{ID: pn.ID, Urn: pnURN, Properties: pn.Properties})
	require.NoError(t, err)
	require.Equal(t, "PN900", str(rr.Inputs, "phoneNumberSid"))

	require.NoError(t, s.Delete(pgp.DeleteRequest{ID: pn.ID, Urn: pnURN, Properties: pn.Properties}))
	require.Nil(t, m.get(svcPath+"/PhoneNumbers/PN900"))
	rr, err = s.Read(pgp.ReadRequest{ID: pn.ID, Urn: pnURN, Properties: pn.Properties})
	require.NoError(t, err)
	require.Empty(t, rr.ID)
	require.NoError(t, s.Delete(pgp.DeleteRequest{ID: cr.ID, Urn: svcURN, Properties: ur.Properties}))
	require.Nil(t, m.get(svcPath))
}

func TestMessagingService_CheckRejectsBadInputs(t *testing.T) {
	_, srv := newMock(t)
	s := newServerFor(t, srv.URL)
	for name, in := range map[string]map[string]any{
		"bad method":   {"friendlyName": "x", "inboundMethod": "DELETE"},
		"bad scan":     {"friendlyName": "x", "scanMessageContent": "maybe"},
		"bad validity": {"friendlyName": "x", "validityPeriod": 0},
	} {
		t.Run(name, func(t *testing.T) {
			r, err := s.Check(pgp.CheckRequest{Urn: twilioURN("MessagingService", "x"), Inputs: pm(in)})
			require.NoError(t, err)
			require.NotEmpty(t, r.Failures)
		})
	}
}

func TestVerifyService_Lifecycle(t *testing.T) {
	m, srv := newMock(t)
	s := newServerFor(t, srv.URL)
	urn := twilioURN("VerifyService", "v")
	in := map[string]any{"friendlyName": "Acme", "codeLength": 7, "lookupEnabled": true}
	cr, err := s.Create(pgp.CreateRequest{Urn: urn, Properties: pm(in)})
	require.NoError(t, err)
	f := m.forms["POST /verify/v2/Services"]
	require.Equal(t, "7", f.Get("CodeLength"))
	require.Equal(t, "true", f.Get("LookupEnabled"))
	in2 := map[string]any{"friendlyName": "Acme", "codeLength": 6}
	ur, err := s.Update(pgp.UpdateRequest{ID: cr.ID, Urn: urn, State: cr.Properties, Inputs: pm(in2), OldInputs: pm(in)})
	require.NoError(t, err)
	require.Equal(t, "6", m.get("/verify/v2/Services/" + cr.ID)["code_length"])
	require.NoError(t, s.Delete(pgp.DeleteRequest{ID: cr.ID, Urn: urn, Properties: ur.Properties}))
	require.Nil(t, m.get("/verify/v2/Services/"+cr.ID))

	r, err := s.Check(pgp.CheckRequest{Urn: urn, Inputs: pm(map[string]any{"friendlyName": "x", "codeLength": 3})})
	require.NoError(t, err)
	require.NotEmpty(t, r.Failures)
}

func TestFunctions(t *testing.T) {
	m, srv := newMock(t)
	seedNumber(m)
	m.seed("/api/2010-04-01/Accounts/"+testAccount, map[string]any{"sid": testAccount, "friendly_name": "Main", "status": "active", "type": "Full", "owner_account_sid": testAccount})
	s := newServerFor(t, srv.URL)

	r, err := s.Invoke(pgp.InvokeRequest{Token: "twilio:index:getAccount", Args: pm(map[string]any{})})
	require.NoError(t, err)
	require.Equal(t, "Main", str(r.Return, "friendlyName"))
	require.Equal(t, "active", str(r.Return, "status"))
	_, hasTok := r.Return.GetOk("authToken")
	require.False(t, hasTok)

	r, err = s.Invoke(pgp.InvokeRequest{Token: "twilio:index:getIncomingPhoneNumber", Args: pm(map[string]any{"phoneNumber": "+14155550100"})})
	require.NoError(t, err)
	require.Equal(t, "PN900", str(r.Return, "sid"))
	v, _ := r.Return.GetOk("voice")
	require.True(t, v.AsBool())
	mms, _ := r.Return.GetOk("mms")
	require.False(t, mms.AsBool())

	_, err = s.Invoke(pgp.InvokeRequest{Token: "twilio:index:getIncomingPhoneNumber", Args: pm(map[string]any{})})
	require.ErrorContains(t, err, "exactly one")
}

func TestClient_RetriesThrottling(t *testing.T) {
	m, srv := newMock(t)
	m.seed("/api/2010-04-01/Accounts/"+testAccount, map[string]any{"sid": testAccount, "friendly_name": "Main", "status": "active"})
	s := newServerFor(t, srv.URL)
	m.throttle = 2
	r, err := s.Invoke(pgp.InvokeRequest{Token: "twilio:index:getAccount", Args: pm(map[string]any{})})
	require.NoError(t, err)
	require.Equal(t, "Main", str(r.Return, "friendlyName"))
	m.throttle = 100
	_, err = s.Invoke(pgp.InvokeRequest{Token: "twilio:index:getAccount", Args: pm(map[string]any{})})
	require.ErrorContains(t, err, "429")
}

func TestClient_ApiErrorIsReadable(t *testing.T) {
	_, srv := newMock(t)
	s := newServerFor(t, srv.URL)
	_, err := s.Invoke(pgp.InvokeRequest{Token: "twilio:index:getAccount", Args: pm(map[string]any{"accountSid": "ACnope"})})
	require.ErrorContains(t, err, "HTTP 404")
	require.ErrorContains(t, err, "20404")
}
