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

const (
	lsBase = "/messaging/v1/LinkShortening"
	lsPEM  = "-----BEGIN CERTIFICATE-----\nMIIB\n-----END CERTIFICATE-----\n-----BEGIN PRIVATE KEY-----\nMIIE\n-----END PRIVATE KEY-----\n"
)

func lsDomain(l *tl, c string) string {
	sid := "ZN" + strings.Repeat(c, 32)
	l.f.addLinkDomain(sid)
	return sid
}

func TestLinkShorteningDomainConfigLC_Lifecycle(t *testing.T) {
	l := newTL(t, "LinkShorteningDomainConfig", "cfg")
	dom := lsDomain(l, "1")
	in := pv(map[string]property.Value{"domainSid": sv(dom), "fallbackUrl": sv("https://acme.example.com/fallback"), "continueOnFailure": bv(true)})
	c := l.create(in)
	require.Equal(t, dom, c.ID)
	cf := l.f.callsMatching("POST", lsBase+"/Domains/"+dom+"/Config")[0].Form
	require.Equal(t, "https://acme.example.com/fallback", cf.Get("FallbackUrl"))
	require.Equal(t, "true", cf.Get("ContinueOnFailure"))
	require.Empty(t, cf["CallbackUrl"])
	require.Empty(t, cf["DisableHttps"])
	require.True(t, strings.HasPrefix(getS(t, c.Properties, "configSid"), "ZK"))
	requireNoChanges(t, l.diff(c.ID, c.Properties, in, in))

	// Drift shows for what the program states; what it leaves out is not adopted.
	l.f.mutate("lsdom", testAccount, dom, func(o map[string]any) {
		o["callback_url"] = "https://hooks.example.com/clicks"
		o["disable_https"] = true
		o["continue_on_failure"] = false
	})
	rd := l.read(c.ID, c.Properties, in)
	require.False(t, hasProp(rd.Inputs, "callbackUrl"))
	require.False(t, rd.Inputs.Get("continueOnFailure").AsBool())
	require.Equal(t, pgp.Update, l.diff(c.ID, rd.Properties, in, rd.Inputs).DetailedDiff["continueOnFailure"].Kind)

	// Everything but the domain updates in place.
	next := with(in, map[string]property.Value{"callbackUrl": sv("https://hooks.example.com/clicks"), "disableHttps": bv(true), "continueOnFailure": bv(false)})
	require.Equal(t, pgp.Update, l.diff(c.ID, rd.Properties, next, rd.Inputs).DetailedDiff["callbackUrl"].Kind)
	l.f.resetCalls()
	u := l.update(c.ID, rd.Properties, next, rd.Inputs)
	uf := l.f.callsMatching("POST", lsBase+"/Domains/"+dom+"/Config")[0].Form
	require.Equal(t, "false", uf.Get("ContinueOnFailure"))
	require.Equal(t, "true", uf.Get("DisableHttps"))
	rd = l.read(c.ID, u.Properties, next)
	requireNoChanges(t, l.diff(c.ID, rd.Properties, next, rd.Inputs))
	d := l.diff(c.ID, rd.Properties, with(next, map[string]property.Value{"domainSid": sv("ZN" + strings.Repeat("2", 32))}), rd.Inputs)
	require.Equal(t, pgp.UpdateReplace, d.DetailedDiff["domainSid"].Kind)

	// Import adopts what Twilio holds, booleans included, and the first preview plans no change.
	imp := l.read(c.ID, empty(), empty())
	require.Empty(t, l.check(imp.Inputs).Failures)
	require.Equal(t, dom, getS(t, imp.Inputs, "domainSid"))
	require.Equal(t, "https://acme.example.com/fallback", getS(t, imp.Inputs, "fallbackUrl"))
	require.True(t, imp.Inputs.Get("disableHttps").AsBool())
	requireNoChanges(t, l.diff(c.ID, imp.Properties, imp.Inputs, imp.Inputs))
	l.requireImportFails("ZN" + strings.Repeat("0", 32))

	// With nothing stated, create only reads; destroy forgets the configuration and sends nothing.
	d2 := lsDomain(l, "3")
	l.f.resetCalls()
	empt := pv(map[string]property.Value{"domainSid": sv(d2)})
	e := l.create(empt)
	require.Empty(t, l.f.callsMatching("POST", "/Config"))
	l.f.resetCalls()
	l.del(e.ID, e.Properties)
	require.Zero(t, l.f.callCount())
	require.Len(t, l.check(pv(map[string]property.Value{"domainSid": sv("nope")})).Failures, 1)
}

func TestLinkShorteningDomainCertificateLC_Lifecycle(t *testing.T) {
	l := newTL(t, "LinkShorteningDomainCertificate", "cert")
	dom := lsDomain(l, "4")
	in := pv(map[string]property.Value{"domainSid": sv(dom), "tlsCert": sv(lsPEM)})
	c := l.create(in)
	require.Equal(t, dom, c.ID)
	require.Equal(t, lsPEM, l.f.callsMatching("POST", lsBase+"/Domains/"+dom+"/Certificate")[0].Form.Get("TlsCert"))
	require.True(t, strings.HasPrefix(getS(t, c.Properties, "certificateSid"), "CR"))
	require.Equal(t, "2027-11-13T00:00:00Z", getS(t, c.Properties, "dateExpires"))
	requireNoChanges(t, l.diff(c.ID, c.Properties, in, in))

	// The certificate is never returned: a refresh keeps the one in state.
	rd := l.read(c.ID, c.Properties, in)
	require.Equal(t, lsPEM, getS(t, rd.Inputs, "tlsCert"))
	requireNoChanges(t, l.diff(c.ID, rd.Properties, in, rd.Inputs))

	// A new certificate updates in place and uploads it.
	next := with(in, map[string]property.Value{"tlsCert": sv(strings.ReplaceAll(lsPEM, "MIIB", "MIIC"))})
	require.Equal(t, pgp.Update, l.diff(c.ID, rd.Properties, next, rd.Inputs).DetailedDiff["tlsCert"].Kind)
	l.f.resetCalls()
	u := l.update(c.ID, rd.Properties, next, rd.Inputs)
	require.Contains(t, l.f.callsMatching("POST", lsBase+"/Domains/"+dom+"/Certificate")[0].Form.Get("TlsCert"), "MIIC")
	require.Contains(t, getS(t, u.Properties, "certInValidation"), "pending")
	d := l.diff(c.ID, u.Properties, with(next, map[string]property.Value{"domainSid": sv("ZN" + strings.Repeat("5", 32))}), next)
	require.Equal(t, pgp.UpdateReplace, d.DetailedDiff["domainSid"].Kind)

	// Import adopts the domain; the certificate is unknown, so the first update uploads it.
	imp := l.read(c.ID, empty(), empty())
	require.Equal(t, dom, getS(t, imp.Inputs, "domainSid"))
	require.True(t, !hasProp(imp.Inputs, "tlsCert") || getS(t, imp.Inputs, "tlsCert") == "")
	l.requireImportFails("ZN" + strings.Repeat("0", 32))

	// Destroy deletes; a certificate deleted outside Pulumi reads as gone.
	l.f.resetCalls()
	l.del(c.ID, u.Properties)
	require.Len(t, l.f.callsMatching("DELETE", lsBase+"/Domains/"+dom+"/Certificate"), 1)
	require.Empty(t, l.read(c.ID, u.Properties, next).ID)
	l.del(c.ID, u.Properties)
	require.Len(t, l.check(pv(map[string]property.Value{"domainSid": sv(dom), "tlsCert": sv("")})).Failures, 1)
}

func TestLinkShorteningMessagingServiceDomainLC_Lifecycle(t *testing.T) {
	l := newTL(t, "LinkShorteningMessagingServiceDomain", "assoc")
	dom, other := lsDomain(l, "6"), lsDomain(l, "7")
	svc := a2pMakeService(t, l, "ls")
	in := pv(map[string]property.Value{"domainSid": sv(dom), "messagingServiceSid": sv(svc)})
	c := l.create(in)
	require.Equal(t, dom+"/"+svc, c.ID)
	require.Len(t, l.f.callsMatching("POST", lsBase+"/Domains/"+dom+"/MessagingServices/"+svc), 1)
	requireNoChanges(t, l.diff(c.ID, c.Properties, in, in))
	rd := l.read(c.ID, c.Properties, in)
	requireNoChanges(t, l.diff(c.ID, rd.Properties, in, rd.Inputs))

	for k, v := range map[string]property.Value{"domainSid": sv(other), "messagingServiceSid": sv("MG" + strings.Repeat("9", 32))} {
		d := l.diff(c.ID, rd.Properties, with(in, map[string]property.Value{k: v}), rd.Inputs)
		require.Equal(t, pgp.UpdateReplace, d.DetailedDiff[k].Kind, k)
	}

	imp := l.read(c.ID, empty(), empty())
	require.Equal(t, dom, getS(t, imp.Inputs, "domainSid"))
	require.Equal(t, svc, getS(t, imp.Inputs, "messagingServiceSid"))
	l.requireImportFails(dom + "/MG" + strings.Repeat("0", 32))

	// The Messaging Service moved to another domain outside Pulumi: the association is gone.
	l.f.mutate("lsassoc", testAccount, svc, func(o map[string]any) { o["domain_sid"] = other })
	require.Empty(t, l.read(c.ID, rd.Properties, in).ID)
	l.f.mutate("lsassoc", testAccount, svc, func(o map[string]any) { o["domain_sid"] = dom })

	l.f.resetCalls()
	l.del(c.ID, rd.Properties)
	require.Len(t, l.f.callsMatching("DELETE", lsBase+"/Domains/"+dom+"/MessagingServices/"+svc), 1)
	require.Empty(t, l.read(c.ID, rd.Properties, in).ID)
	l.del(c.ID, rd.Properties)
	require.Len(t, l.check(pv(map[string]property.Value{"domainSid": sv("x"), "messagingServiceSid": sv("y")})).Failures, 2)
}

func TestLookup_LinkShortening(t *testing.T) {
	l := newTL(t, "LinkShorteningDomainConfig", "cfg")
	dom := lsDomain(l, "8")
	svc := a2pMakeService(t, l, "lk")
	l.other("LinkShorteningDomainConfig", "c").create(pv(map[string]property.Value{"domainSid": sv(dom), "callbackUrl": sv("https://hooks.example.com/c")}))
	l.other("LinkShorteningDomainCertificate", "k").create(pv(map[string]property.Value{"domainSid": sv(dom), "tlsCert": sv(lsPEM)}))
	l.other("LinkShorteningMessagingServiceDomain", "a").create(pv(map[string]property.Value{"domainSid": sv(dom), "messagingServiceSid": sv(svc)}))

	r, err := invoke(t, l, "getLinkShorteningDomainConfig", map[string]property.Value{"domainSid": sv(dom)})
	require.NoError(t, err)
	require.Equal(t, "https://hooks.example.com/c", getS(t, r, "callbackUrl"))
	require.False(t, r.Get("disableHttps").AsBool())
	_, err = invoke(t, l, "getLinkShorteningDomainConfig", map[string]property.Value{"domainSid": sv("ZNmissing")})
	require.ErrorContains(t, err, "was not found")
	_, err = invoke(t, l, "getLinkShorteningDomainConfig", map[string]property.Value{"domainSid": sv("")})
	require.ErrorContains(t, err, "must be set")

	r, err = invoke(t, l, "getLinkShorteningMessagingServiceDomainConfig", map[string]property.Value{"messagingServiceSid": sv(svc)})
	require.NoError(t, err)
	require.Equal(t, dom, getS(t, r, "domainSid"))
	require.Equal(t, svc, getS(t, r, "messagingServiceSid"))

	r, err = invoke(t, l, "getLinkShorteningDomainCertificate", map[string]property.Value{"domainSid": sv(dom)})
	require.NoError(t, err)
	require.Equal(t, "go.acme.example.com", getS(t, r, "domainName"))
	require.False(t, hasProp(r, "tlsCert"))

	r, err = invoke(t, l, "getLinkShorteningDomainDnsValidation", map[string]property.Value{"domainSid": sv(dom)})
	require.NoError(t, err)
	require.False(t, r.Get("isValid").AsBool())
	require.Equal(t, "CNAME record missing", getS(t, r, "reason"))

	r, err = invoke(t, l, "getLinkShorteningMessagingServiceDomain", map[string]property.Value{"messagingServiceSid": sv(svc)})
	require.NoError(t, err)
	require.Equal(t, dom, getS(t, r, "domainSid"))
	_, err = invoke(t, l, "getLinkShorteningMessagingServiceDomain", map[string]property.Value{"messagingServiceSid": sv("MG" + strings.Repeat("0", 32))})
	require.ErrorContains(t, err, "was not found")
}
