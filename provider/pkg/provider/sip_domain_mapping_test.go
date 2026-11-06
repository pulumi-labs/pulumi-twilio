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

// mappingFixture creates a domain, a credential list and an IP access control list.
type mappingFixture struct {
	l                  *tl
	domain, cl, acl    string
	domainP, clP, aclP property.Map
}

func newMappingFixture(t *testing.T, typ string) *mappingFixture {
	l := newTL(t, typ, "m")
	d := l.other("SipDomain", "d").create(domainInputs())
	cl := l.other("SipCredentialList", "cl").create(pv(map[string]property.Value{"friendlyName": sv("users")}))
	acl := l.other("SipIpAccessControlList", "acl").create(pv(map[string]property.Value{"friendlyName": sv("offices")}))
	return &mappingFixture{l: l, domain: d.ID, cl: cl.ID, acl: acl.ID, domainP: d.Properties, clP: cl.Properties, aclP: acl.Properties}
}

func (m *mappingFixture) authType(t *testing.T) string {
	o := m.l.f.get("sipdomain", testAccount, m.domain)
	require.NotNil(t, o)
	return o["auth_type"].(string)
}

func TestSipDomainCredentialListMappingLC_Lifecycle(t *testing.T) {
	m := newMappingFixture(t, "SipDomainCredentialListMapping")
	l := m.l
	in := pv(map[string]property.Value{"domainSid": sv(m.domain), "credentialListSid": sv(m.cl)})
	chk := l.checked(in)
	require.Equal(t, "calls", getS(t, chk, "authScope"), "the scope defaults to calls")

	c := l.create(chk)
	require.Equal(t, m.domain+"/calls/"+m.cl, c.ID)
	require.Equal(t, m.cl, getS(t, c.Properties, "sid"))
	require.Equal(t, "users", getS(t, c.Properties, "friendlyName"))
	require.Equal(t, testAccount, getS(t, c.Properties, "ownerAccountSid"))
	require.NotNil(t, l.f.get("sipmapcl", testAccount, sipMapKey(m.domain, "Calls", m.cl)))
	require.Equal(t, "CREDENTIAL_LIST", m.authType(t))
	requireNoChanges(t, l.diff(c.ID, c.Properties, chk, chk))

	// A refresh changes nothing and the list name follows the list.
	rd := l.read(c.ID, c.Properties, chk)
	require.Equal(t, c.ID, rd.ID)
	requireNoChanges(t, l.diff(c.ID, rd.Properties, chk, rd.Inputs))

	// Every input replaces: there is no update call.
	for name, v := range map[string]property.Value{"domainSid": sv("SDother"), "credentialListSid": sv("CLother"), "authScope": sv("registrations"), "accountSid": sv(fakeSub)} {
		d := l.diff(c.ID, rd.Properties, with(chk, map[string]property.Value{name: v}), chk)
		require.True(t, d.HasChanges, name)
		require.Contains(t, []pgp.DiffKind{pgp.UpdateReplace, pgp.AddReplace}, d.DetailedDiff[name].Kind, name)
	}

	// Gone out of band: a refresh drops it, and deleting twice is fine.
	l.f.remove("sipmapcl", testAccount, sipMapKey(m.domain, "Calls", m.cl))
	require.Empty(t, l.read(c.ID, c.Properties, chk).ID)
	l.del(c.ID, c.Properties)

	c = l.create(chk)
	l.del(c.ID, c.Properties)
	require.Nil(t, l.f.get("sipmapcl", testAccount, sipMapKey(m.domain, "Calls", m.cl)))
	require.Empty(t, m.authType(t))
	require.Len(t, l.f.callsMatching("DELETE", "/Auth/Calls/CredentialListMappings/"+m.cl+".json"), 2)
}

func TestSipDomainCredentialListMappingLC_RegistrationsImportAndChecks(t *testing.T) {
	m := newMappingFixture(t, "SipDomainCredentialListMapping")
	l := m.l
	in := l.checked(pv(map[string]property.Value{"domainSid": sv(m.domain), "credentialListSid": sv(m.cl), "authScope": sv("registrations")}))
	c := l.create(in)
	require.Equal(t, m.domain+"/registrations/"+m.cl, c.ID)
	require.NotNil(t, l.f.get("sipmapcl", testAccount, sipMapKey(m.domain, "Registrations", m.cl)))
	require.Empty(t, m.authType(t), "registrations do not set the call auth type")

	// Both scopes can hold the same list.
	both := l.create(l.checked(pv(map[string]property.Value{"domainSid": sv(m.domain), "credentialListSid": sv(m.cl)})))
	require.Equal(t, "CREDENTIAL_LIST", m.authType(t))

	// Import by ID alone adopts the ID's parts.
	imp := l.read(c.ID, empty(), empty())
	require.Equal(t, m.domain, getS(t, imp.Inputs, "domainSid"))
	require.Equal(t, m.cl, getS(t, imp.Inputs, "credentialListSid"))
	require.Equal(t, "registrations", getS(t, imp.Inputs, "authScope"))
	require.False(t, hasProp(imp.Inputs, "accountSid"))
	requireNoChanges(t, l.diff(imp.ID, imp.Properties, in, imp.Inputs))
	imp = l.read(both.ID, empty(), empty())
	require.Equal(t, "calls", getS(t, imp.Inputs, "authScope"))
	requireNoChanges(t, l.diff(imp.ID, imp.Properties, l.checked(pv(map[string]property.Value{"domainSid": sv(m.domain), "credentialListSid": sv(m.cl)})), imp.Inputs))

	// A preview creates nothing and invents no ID.
	before := l.f.sipCount("sipmapcl")
	pr, err := l.s.Create(pgp.CreateRequest{Urn: l.urn, Properties: in, DryRun: true})
	require.NoError(t, err)
	require.Empty(t, pr.ID)
	require.Equal(t, before, l.f.sipCount("sipmapcl"))

	// A second mapping of the same list and scope is Twilio's error, surfaced.
	_, err = l.s.Create(pgp.CreateRequest{Urn: l.urn, Properties: in})
	require.ErrorContains(t, err, "already mapped")

	// Bad IDs and inputs.
	for _, bad := range []string{"SD1", "SD1/calls", "SD1/nope/CL1", "a/b/c/d/e"} {
		_, err = l.s.Read(pgp.ReadRequest{Urn: l.urn, ID: bad, Properties: empty(), Inputs: empty()})
		require.Error(t, err, bad)
	}
	require.NotEmpty(t, l.check(pv(map[string]property.Value{"domainSid": sv(""), "credentialListSid": sv(m.cl)})).Failures)
	require.NotEmpty(t, l.check(pv(map[string]property.Value{"domainSid": sv(m.domain), "credentialListSid": sv("")})).Failures)
	require.NotEmpty(t, l.check(pv(map[string]property.Value{"domainSid": sv(m.domain), "credentialListSid": sv(m.cl), "authScope": sv("all")})).Failures)
	require.Empty(t, l.check(pv(map[string]property.Value{"domainSid": computed(), "credentialListSid": computed(), "authScope": computed(), "accountSid": computed()})).Failures)

	l.del(c.ID, c.Properties)
	l.del(both.ID, both.Properties)
}

func TestSipDomainCredentialListMappingLC_Subaccount(t *testing.T) {
	l := newTL(t, "SipDomainCredentialListMapping", "m")
	l.f.addSubaccount(fakeSub)
	sub := func(extra map[string]property.Value) property.Map {
		return with(pv(extra), map[string]property.Value{"accountSid": sv(fakeSub)})
	}
	d := l.other("SipDomain", "d").create(with(domainInputs(), map[string]property.Value{"accountSid": sv(fakeSub)}))
	_, dsid := splitScoped(d.ID)
	cl := l.other("SipCredentialList", "cl").create(sub(map[string]property.Value{"friendlyName": sv("users")}))
	_, clsid := splitScoped(cl.ID)

	in := l.checked(sub(map[string]property.Value{"domainSid": sv(dsid), "credentialListSid": sv(clsid)}))
	c := l.create(in)
	require.Equal(t, fakeSub+"/"+dsid+"/calls/"+clsid, c.ID)
	require.NotNil(t, l.f.get("sipmapcl", fakeSub, sipMapKey(dsid, "Calls", clsid)))
	imp := l.read(c.ID, empty(), empty())
	require.Equal(t, fakeSub, getS(t, imp.Inputs, "accountSid"))
	requireNoChanges(t, l.diff(imp.ID, imp.Properties, in, imp.Inputs))
	l.del(c.ID, c.Properties)
	require.Nil(t, l.f.get("sipmapcl", fakeSub, sipMapKey(dsid, "Calls", clsid)))
}

func TestSipDomainIpAccessControlListMappingLC_Lifecycle(t *testing.T) {
	m := newMappingFixture(t, "SipDomainIpAccessControlListMapping")
	l := m.l
	in := pv(map[string]property.Value{"domainSid": sv(m.domain), "ipAccessControlListSid": sv(m.acl)})
	c := l.create(in)
	require.Equal(t, m.domain+"/"+m.acl, c.ID)
	require.Equal(t, m.acl, getS(t, c.Properties, "sid"))
	require.Equal(t, "offices", getS(t, c.Properties, "friendlyName"))
	require.Equal(t, "IP_ACL", m.authType(t))
	requireNoChanges(t, l.diff(c.ID, c.Properties, in, in))

	// Both kinds of mapping show in the domain's auth type.
	cm := m.l.other("SipDomainCredentialListMapping", "cm").create(l.other("SipDomainCredentialListMapping", "cm").checked(
		pv(map[string]property.Value{"domainSid": sv(m.domain), "credentialListSid": sv(m.cl)})))
	require.Equal(t, "IP_ACL,CREDENTIAL_LIST", m.authType(t))
	m.l.other("SipDomainCredentialListMapping", "cm").del(cm.ID, cm.Properties)

	rd := l.read(c.ID, c.Properties, in)
	requireNoChanges(t, l.diff(c.ID, rd.Properties, in, rd.Inputs))
	for name, v := range map[string]property.Value{"domainSid": sv("SDother"), "ipAccessControlListSid": sv("ALother"), "accountSid": sv(fakeSub)} {
		d := l.diff(c.ID, rd.Properties, with(in, map[string]property.Value{name: v}), in)
		require.True(t, d.HasChanges, name)
		require.Contains(t, []pgp.DiffKind{pgp.UpdateReplace, pgp.AddReplace}, d.DetailedDiff[name].Kind, name)
	}

	// Import, preview, checks.
	imp := l.read(c.ID, empty(), empty())
	require.Equal(t, m.domain, getS(t, imp.Inputs, "domainSid"))
	require.Equal(t, m.acl, getS(t, imp.Inputs, "ipAccessControlListSid"))
	requireNoChanges(t, l.diff(imp.ID, imp.Properties, in, imp.Inputs))
	before := l.f.sipCount("sipmapip")
	pr, err := l.s.Create(pgp.CreateRequest{Urn: l.urn, Properties: in, DryRun: true})
	require.NoError(t, err)
	require.Empty(t, pr.ID)
	require.Equal(t, before, l.f.sipCount("sipmapip"))
	require.NotEmpty(t, l.check(pv(map[string]property.Value{"domainSid": sv(""), "ipAccessControlListSid": sv(m.acl)})).Failures)
	require.NotEmpty(t, l.check(pv(map[string]property.Value{"domainSid": sv(m.domain), "ipAccessControlListSid": sv("")})).Failures)
	require.Empty(t, l.check(pv(map[string]property.Value{"domainSid": computed(), "ipAccessControlListSid": computed()})).Failures)
	_, err = l.s.Read(pgp.ReadRequest{Urn: l.urn, ID: "SD1", Properties: empty(), Inputs: empty()})
	require.Error(t, err)

	// Gone out of band, and deleting twice is fine.
	l.f.remove("sipmapip", testAccount, sipMapKey(m.domain, "Calls", m.acl))
	require.Empty(t, l.read(c.ID, c.Properties, in).ID)
	l.del(c.ID, c.Properties)
	c = l.create(in)
	l.del(c.ID, c.Properties)
	require.Empty(t, m.authType(t))
}

func TestLookup_SipDomainMappings(t *testing.T) {
	m := newMappingFixture(t, "SipDomainCredentialListMapping")
	l := m.l
	cm := l.create(l.checked(pv(map[string]property.Value{"domainSid": sv(m.domain), "credentialListSid": sv(m.cl), "authScope": sv("registrations")})))
	r, err := invoke(t, l, "getSipDomainCredentialListMapping", map[string]property.Value{"domainSid": sv(m.domain), "credentialListSid": sv(m.cl), "authScope": sv("registrations")})
	require.NoError(t, err)
	require.Equal(t, m.cl, getS(t, r, "sid"))
	require.Equal(t, "users", getS(t, r, "friendlyName"))
	require.Equal(t, "registrations", getS(t, r, "authScope"))
	require.Equal(t, testAccount, getS(t, r, "ownerAccountSid"))

	// The calls scope has no such mapping.
	_, err = invoke(t, l, "getSipDomainCredentialListMapping", map[string]property.Value{"domainSid": sv(m.domain), "credentialListSid": sv(m.cl)})
	require.ErrorContains(t, err, "was not found")
	_, err = invoke(t, l, "getSipDomainCredentialListMapping", map[string]property.Value{"domainSid": sv(m.domain), "credentialListSid": sv(m.cl), "authScope": sv("x")})
	require.ErrorContains(t, err, "authScope")
	l.del(cm.ID, cm.Properties)

	im := l.other("SipDomainIpAccessControlListMapping", "im")
	ic := im.create(pv(map[string]property.Value{"domainSid": sv(m.domain), "ipAccessControlListSid": sv(m.acl)}))
	r, err = invoke(t, l, "getSipDomainIpAccessControlListMapping", map[string]property.Value{"domainSid": sv(m.domain), "ipAccessControlListSid": sv(m.acl)})
	require.NoError(t, err)
	require.Equal(t, "offices", getS(t, r, "friendlyName"))
	im.del(ic.ID, ic.Properties)
	_, err = invoke(t, l, "getSipDomainIpAccessControlListMapping", map[string]property.Value{"domainSid": sv(m.domain), "ipAccessControlListSid": sv(m.acl)})
	require.ErrorContains(t, err, "was not found")

	for tok, args := range map[string]map[string]property.Value{
		"getSipDomainCredentialListMapping":      {"domainSid": sv("SD1")},
		"getSipDomainIpAccessControlListMapping": {"domainSid": sv("SD1")},
	} {
		_, err = invoke(t, l, tok, args)
		require.Error(t, err, tok)
	}
}

func TestSipCredMappingIDs(t *testing.T) {
	a, d, s, id, err := splitCredMappingID("SD1/registrations/CL2")
	require.NoError(t, err)
	require.Equal(t, []string{"", "SD1", "registrations", "CL2"}, []string{a, d, s, id})
	a, d, s, id, err = splitCredMappingID("AC9/SD1/calls/CL2")
	require.NoError(t, err)
	require.Equal(t, []string{"AC9", "SD1", "calls", "CL2"}, []string{a, d, s, id})
}
