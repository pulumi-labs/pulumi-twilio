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
	"fmt"
	"net/url"
	"regexp"

	"github.com/pulumi/pulumi-go-provider/infer"
)

// Link shortening (messaging.twilio.com v1 /LinkShortening): a domain of the account turns the links
// of a Messaging Service's messages into short links. The domain itself is set up with Twilio, so its
// SID (ZN...) is an input here. The shapes follow twilio-oai spec/json/twilio_messaging_v1.json (1.0.0).

var lsDomainSidRE = regexp.MustCompile(`^ZN[0-9a-fA-F]{32}$`)

func lsDomainPath(domain string, rest ...string) string {
	path := "/v1/LinkShortening/Domains/" + url.PathEscape(domain)
	for _, r := range rest {
		path += "/" + url.PathEscape(r)
	}
	return path
}

func lsCheckDomain(c *checker, sid string) {
	c.sid("domainSid", sid, lsDomainSidRE, "link shortening domain (ZN followed by 32 hex characters)")
}

// lsBool is the refreshed value of an optional bool Twilio always returns: an input that is set
// follows Twilio, an unset one stays unset, and an import adopts the value Twilio holds (Twilio
// documents no default for these settings, so none is assumed).
func lsBool(in, api *bool, imported bool) *bool {
	switch {
	case imported:
		return api
	case in == nil:
		return nil
	case api == nil:
		return in
	}
	return api
}

// ------------------------------------------------------------ LinkShorteningDomainConfig

// LinkShorteningDomainConfig manages the configuration of a link shortening domain: where unknown
// short links go, where click events are sent and how failures are handled. A domain always has
// one, so this adopts it; destroying the resource forgets it and leaves the settings as they are.
type LinkShorteningDomainConfig struct{}

// LinkShorteningDomainConfigArgs are the inputs of a LinkShorteningDomainConfig.
type LinkShorteningDomainConfigArgs struct {
	DomainSid         string  `pulumi:"domainSid" provider:"replaceOnChanges"`
	FallbackUrl       *string `pulumi:"fallbackUrl,optional"`
	CallbackUrl       *string `pulumi:"callbackUrl,optional"`
	ContinueOnFailure *bool   `pulumi:"continueOnFailure,optional"`
	DisableHttps      *bool   `pulumi:"disableHttps,optional"`
}

func (a *LinkShorteningDomainConfigArgs) Annotate(an infer.Annotator) {
	an.Describe(&a.DomainSid, "SID (ZN...) of the link shortening domain. Changing it replaces the resource.")
	an.Describe(&a.FallbackUrl, "Requests to the domain that match no shortened message are redirected to this URL. Removing it leaves the setting unchanged.")
	an.Describe(&a.CallbackUrl, "URL that receives a click event whenever a recipient clicks a shortened link. Removing it leaves the setting unchanged.")
	an.Describe(&a.ContinueOnFailure, "Whether a message is still delivered, without a short link, when link shortening fails. Removing it leaves the setting unchanged.")
	an.Describe(&a.DisableHttps, "Whether shortened links are sent without the https:// prefix. Removing it leaves the setting unchanged.")
}

// LinkShorteningDomainConfigState is the persisted state of a LinkShorteningDomainConfig.
type LinkShorteningDomainConfigState struct {
	LinkShorteningDomainConfigArgs

	ConfigSid   string `pulumi:"configSid"`
	Url         string `pulumi:"url"`
	DateCreated string `pulumi:"dateCreated"`
	DateUpdated string `pulumi:"dateUpdated"`
}

func (s *LinkShorteningDomainConfigState) Annotate(a infer.Annotator) {
	a.Describe(&s.ConfigSid, "The SID (ZK...) of the domain configuration.")
	a.Describe(&s.Url, "The absolute URL of the domain configuration.")
	a.Describe(&s.DateCreated, "When the domain configuration was created (ISO 8601).")
	a.Describe(&s.DateUpdated, "When the domain configuration was last updated (ISO 8601).")
}

func (r *LinkShorteningDomainConfig) Annotate(a infer.Annotator) {
	a.Describe(&r, "The configuration of a link shortening domain. The resource ID is the domain SID. A domain always has one configuration, so this adopts it; destroying the resource forgets it and leaves the settings on Twilio as they are.")
}

var _ infer.CustomResource[LinkShorteningDomainConfigArgs, LinkShorteningDomainConfigState] = (*LinkShorteningDomainConfig)(nil)
var _ infer.CustomCheck[LinkShorteningDomainConfigArgs] = (*LinkShorteningDomainConfig)(nil)
var _ infer.CustomDiff[LinkShorteningDomainConfigArgs, LinkShorteningDomainConfigState] = (*LinkShorteningDomainConfig)(nil)
var _ infer.CustomRead[LinkShorteningDomainConfigArgs, LinkShorteningDomainConfigState] = (*LinkShorteningDomainConfig)(nil)
var _ infer.CustomUpdate[LinkShorteningDomainConfigArgs, LinkShorteningDomainConfigState] = (*LinkShorteningDomainConfig)(nil)
var _ infer.CustomDelete[LinkShorteningDomainConfigState] = (*LinkShorteningDomainConfig)(nil)

func (r *LinkShorteningDomainConfig) Check(ctx context.Context, req infer.CheckRequest) (infer.CheckResponse[LinkShorteningDomainConfigArgs], error) {
	args, failures, err := infer.DefaultCheck[LinkShorteningDomainConfigArgs](ctx, req.NewInputs)
	if err != nil || len(failures) > 0 {
		return infer.CheckResponse[LinkShorteningDomainConfigArgs]{Inputs: args, Failures: failures}, err
	}
	c := &checker{unk: unknowns(req.NewInputs)}
	lsCheckDomain(c, args.DomainSid)
	return infer.CheckResponse[LinkShorteningDomainConfigArgs]{Inputs: args, Failures: c.failures}, nil
}

func (r *LinkShorteningDomainConfig) Diff(_ context.Context, req infer.DiffRequest[LinkShorteningDomainConfigArgs, LinkShorteningDomainConfigState]) (infer.DiffResponse, error) {
	o, n := req.State.LinkShorteningDomainConfigArgs, req.Inputs
	d := trDiffSet{}
	d.replace("domainSid", o.DomainSid, n.DomainSid)
	d.str("fallbackUrl", o.FallbackUrl, n.FallbackUrl)
	d.str("callbackUrl", o.CallbackUrl, n.CallbackUrl)
	d.boolean("continueOnFailure", o.ContinueOnFailure, n.ContinueOnFailure)
	d.boolean("disableHttps", o.DisableHttps, n.DisableHttps)
	return d.response(), nil
}

func (a LinkShorteningDomainConfigArgs) form() url.Values {
	form := url.Values{}
	setS(form, "FallbackUrl", a.FallbackUrl)
	setS(form, "CallbackUrl", a.CallbackUrl)
	setB(form, "ContinueOnFailure", a.ContinueOnFailure)
	setB(form, "DisableHttps", a.DisableHttps)
	return form
}

func (s *LinkShorteningDomainConfigState) fill(o obj) {
	s.ConfigSid, s.Url = o.str("config_sid"), o.str("url")
	s.DateCreated, s.DateUpdated = o.str("date_created"), o.str("date_updated")
}

// apply writes the settings the program states; with none it only reads the configuration.
func (a LinkShorteningDomainConfigArgs) apply(ctx context.Context, verb string) (obj, error) {
	c, err := clientFor(ctx)
	if err != nil {
		return nil, err
	}
	form, method := a.form(), "POST"
	if len(form) == 0 {
		method, form = "GET", nil
	}
	o, err := c.do(ctx, method, messagingProduct, lsDomainPath(a.DomainSid, "Config"), nil, form)
	if err != nil {
		return nil, fmt.Errorf("%s link shortening domain configuration %s: %w", verb, a.DomainSid, err)
	}
	return o, nil
}

func (r *LinkShorteningDomainConfig) Create(ctx context.Context, req infer.CreateRequest[LinkShorteningDomainConfigArgs]) (infer.CreateResponse[LinkShorteningDomainConfigState], error) {
	type resp = infer.CreateResponse[LinkShorteningDomainConfigState]
	a := req.Inputs
	st := LinkShorteningDomainConfigState{LinkShorteningDomainConfigArgs: a}
	if req.DryRun {
		return resp{Output: st}, nil
	}
	o, err := a.apply(ctx, "configure")
	if err != nil {
		return resp{}, err
	}
	st.fill(o)
	return resp{ID: a.DomainSid, Output: st}, nil
}

func (r *LinkShorteningDomainConfig) Read(ctx context.Context, req infer.ReadRequest[LinkShorteningDomainConfigArgs, LinkShorteningDomainConfigState]) (infer.ReadResponse[LinkShorteningDomainConfigArgs, LinkShorteningDomainConfigState], error) {
	type resp = infer.ReadResponse[LinkShorteningDomainConfigArgs, LinkShorteningDomainConfigState]
	c, err := clientFor(ctx)
	if err != nil {
		return resp{}, err
	}
	o, err := c.do(ctx, "GET", messagingProduct, lsDomainPath(req.ID, "Config"), nil, nil)
	if err != nil {
		if isNotFound(err) {
			if err := notFoundOnImport("link shortening domain configuration", req.ID, req.Inputs, req.State); err != nil {
				return resp{}, err
			}
			return resp{}, nil
		}
		return resp{}, fmt.Errorf("read link shortening domain configuration %s: %w", req.ID, err)
	}
	in := req.Inputs
	imported := in.DomainSid == ""
	in.DomainSid = req.ID
	in.FallbackUrl = trStr(in.FallbackUrl, o.optStr("fallback_url"), "", imported)
	in.CallbackUrl = trStr(in.CallbackUrl, o.optStr("callback_url"), "", imported)
	in.ContinueOnFailure = lsBool(in.ContinueOnFailure, o.optBool("continue_on_failure"), imported)
	in.DisableHttps = lsBool(in.DisableHttps, o.optBool("disable_https"), imported)
	st := LinkShorteningDomainConfigState{LinkShorteningDomainConfigArgs: in}
	st.fill(o)
	return resp{ID: req.ID, Inputs: in, State: st}, nil
}

func (r *LinkShorteningDomainConfig) Update(ctx context.Context, req infer.UpdateRequest[LinkShorteningDomainConfigArgs, LinkShorteningDomainConfigState]) (infer.UpdateResponse[LinkShorteningDomainConfigState], error) {
	type resp = infer.UpdateResponse[LinkShorteningDomainConfigState]
	st := req.State
	st.LinkShorteningDomainConfigArgs = req.Inputs
	if req.DryRun {
		return resp{Output: st}, nil
	}
	o, err := req.Inputs.apply(ctx, "update")
	if err != nil {
		return resp{}, err
	}
	st.fill(o)
	return resp{Output: st}, nil
}

// Delete forgets the configuration: a domain always has one and the spec offers no way to remove it.
func (r *LinkShorteningDomainConfig) Delete(_ context.Context, _ infer.DeleteRequest[LinkShorteningDomainConfigState]) (infer.DeleteResponse, error) {
	return infer.DeleteResponse{}, nil
}

// ------------------------------------------------------------ LinkShorteningDomainCertificate

// LinkShorteningDomainCertificate manages the private TLS certificate of a link shortening domain.
type LinkShorteningDomainCertificate struct{}

// LinkShorteningDomainCertificateArgs are the inputs of a LinkShorteningDomainCertificate.
type LinkShorteningDomainCertificateArgs struct {
	DomainSid string `pulumi:"domainSid" provider:"replaceOnChanges"`
	TlsCert   string `pulumi:"tlsCert" provider:"secret"`
}

func (a *LinkShorteningDomainCertificateArgs) Annotate(an infer.Annotator) {
	an.Describe(&a.DomainSid, "SID (ZN...) of the link shortening domain. Changing it replaces the resource.")
	an.Describe(&a.TlsCert, "The full TLS certificate and its private key for the domain in PEM format. It holds a private key, so it is a secret. Twilio never returns it: it is sent on create and whenever it changes, and a refresh keeps the value in state. After an import it is unknown, so the first update uploads it again.")
}

// LinkShorteningDomainCertificateState is the persisted state of a LinkShorteningDomainCertificate.
type LinkShorteningDomainCertificateState struct {
	LinkShorteningDomainCertificateArgs

	CertificateSid   string `pulumi:"certificateSid"`
	DomainName       string `pulumi:"domainName"`
	DateExpires      string `pulumi:"dateExpires"`
	CertInValidation string `pulumi:"certInValidation"`
	Url              string `pulumi:"url"`
	DateCreated      string `pulumi:"dateCreated"`
	DateUpdated      string `pulumi:"dateUpdated"`
}

func (s *LinkShorteningDomainCertificateState) Annotate(a infer.Annotator) {
	a.Describe(&s.CertificateSid, "The certificate SID.")
	a.Describe(&s.DomainName, "The domain name of the certificate.")
	a.Describe(&s.DateExpires, "When the certificate expires (ISO 8601); upload a new one before then.")
	a.Describe(&s.CertInValidation, "A JSON document with the status and upload date of a new certificate that is still being validated (empty when none).")
	a.Describe(&s.Url, "The absolute URL of the certificate.")
	a.Describe(&s.DateCreated, "When the domain was registered with Twilio (ISO 8601).")
	a.Describe(&s.DateUpdated, "When the certificate was last updated (ISO 8601).")
}

func (r *LinkShorteningDomainCertificate) Annotate(a infer.Annotator) {
	a.Describe(&r, "The private TLS certificate of a link shortening domain. The resource ID is the domain SID. Destroying the resource deletes the certificate. Requesting a certificate that Twilio manages is not part of this resource.")
}

var _ infer.CustomResource[LinkShorteningDomainCertificateArgs, LinkShorteningDomainCertificateState] = (*LinkShorteningDomainCertificate)(nil)
var _ infer.CustomCheck[LinkShorteningDomainCertificateArgs] = (*LinkShorteningDomainCertificate)(nil)
var _ infer.CustomDiff[LinkShorteningDomainCertificateArgs, LinkShorteningDomainCertificateState] = (*LinkShorteningDomainCertificate)(nil)
var _ infer.CustomRead[LinkShorteningDomainCertificateArgs, LinkShorteningDomainCertificateState] = (*LinkShorteningDomainCertificate)(nil)
var _ infer.CustomUpdate[LinkShorteningDomainCertificateArgs, LinkShorteningDomainCertificateState] = (*LinkShorteningDomainCertificate)(nil)
var _ infer.CustomDelete[LinkShorteningDomainCertificateState] = (*LinkShorteningDomainCertificate)(nil)

func (r *LinkShorteningDomainCertificate) Check(ctx context.Context, req infer.CheckRequest) (infer.CheckResponse[LinkShorteningDomainCertificateArgs], error) {
	args, failures, err := infer.DefaultCheck[LinkShorteningDomainCertificateArgs](ctx, req.NewInputs)
	if err != nil || len(failures) > 0 {
		return infer.CheckResponse[LinkShorteningDomainCertificateArgs]{Inputs: args, Failures: failures}, err
	}
	c := &checker{unk: unknowns(req.NewInputs)}
	lsCheckDomain(c, args.DomainSid)
	c.required("tlsCert", args.TlsCert)
	return infer.CheckResponse[LinkShorteningDomainCertificateArgs]{Inputs: args, Failures: c.failures}, nil
}

func (r *LinkShorteningDomainCertificate) Diff(_ context.Context, req infer.DiffRequest[LinkShorteningDomainCertificateArgs, LinkShorteningDomainCertificateState]) (infer.DiffResponse, error) {
	o, n := req.State.LinkShorteningDomainCertificateArgs, req.Inputs
	d := trDiffSet{}
	d.replace("domainSid", o.DomainSid, n.DomainSid)
	d.requiredStr("tlsCert", o.TlsCert, n.TlsCert)
	return d.response(), nil
}

func (s *LinkShorteningDomainCertificateState) fill(o obj) {
	s.CertificateSid, s.DomainName, s.DateExpires = o.str("certificate_sid"), o.str("domain_name"), o.str("date_expires")
	s.CertInValidation, s.Url = a2pJSON(o.get("cert_in_validation")), o.str("url")
	s.DateCreated, s.DateUpdated = o.str("date_created"), o.str("date_updated")
}

func (r *LinkShorteningDomainCertificate) upload(ctx context.Context, a LinkShorteningDomainCertificateArgs, verb string) (obj, error) {
	c, err := clientFor(ctx)
	if err != nil {
		return nil, err
	}
	o, err := c.do(ctx, "POST", messagingProduct, lsDomainPath(a.DomainSid, "Certificate"), nil, url.Values{"TlsCert": {a.TlsCert}})
	if err != nil {
		return nil, fmt.Errorf("%s link shortening certificate of domain %s: %w", verb, a.DomainSid, err)
	}
	return o, nil
}

func (r *LinkShorteningDomainCertificate) Create(ctx context.Context, req infer.CreateRequest[LinkShorteningDomainCertificateArgs]) (infer.CreateResponse[LinkShorteningDomainCertificateState], error) {
	type resp = infer.CreateResponse[LinkShorteningDomainCertificateState]
	a := req.Inputs
	st := LinkShorteningDomainCertificateState{LinkShorteningDomainCertificateArgs: a}
	if req.DryRun {
		return resp{Output: st}, nil
	}
	o, err := r.upload(ctx, a, "upload")
	if err != nil {
		return resp{}, err
	}
	st.fill(o)
	return resp{ID: a.DomainSid, Output: st}, nil
}

func (r *LinkShorteningDomainCertificate) Read(ctx context.Context, req infer.ReadRequest[LinkShorteningDomainCertificateArgs, LinkShorteningDomainCertificateState]) (infer.ReadResponse[LinkShorteningDomainCertificateArgs, LinkShorteningDomainCertificateState], error) {
	type resp = infer.ReadResponse[LinkShorteningDomainCertificateArgs, LinkShorteningDomainCertificateState]
	c, err := clientFor(ctx)
	if err != nil {
		return resp{}, err
	}
	o, err := c.do(ctx, "GET", messagingProduct, lsDomainPath(req.ID, "Certificate"), nil, nil)
	if err != nil {
		if isNotFound(err) {
			if err := notFoundOnImport("link shortening certificate", req.ID, req.Inputs, req.State); err != nil {
				return resp{}, err
			}
			return resp{}, nil
		}
		return resp{}, fmt.Errorf("read link shortening certificate of domain %s: %w", req.ID, err)
	}
	in := req.Inputs
	in.DomainSid = req.ID
	st := LinkShorteningDomainCertificateState{LinkShorteningDomainCertificateArgs: in}
	st.fill(o)
	return resp{ID: req.ID, Inputs: in, State: st}, nil
}

func (r *LinkShorteningDomainCertificate) Update(ctx context.Context, req infer.UpdateRequest[LinkShorteningDomainCertificateArgs, LinkShorteningDomainCertificateState]) (infer.UpdateResponse[LinkShorteningDomainCertificateState], error) {
	type resp = infer.UpdateResponse[LinkShorteningDomainCertificateState]
	st := req.State
	st.LinkShorteningDomainCertificateArgs = req.Inputs
	if req.DryRun {
		return resp{Output: st}, nil
	}
	o, err := r.upload(ctx, req.Inputs, "update")
	if err != nil {
		return resp{}, err
	}
	st.fill(o)
	return resp{Output: st}, nil
}

func (r *LinkShorteningDomainCertificate) Delete(ctx context.Context, req infer.DeleteRequest[LinkShorteningDomainCertificateState]) (infer.DeleteResponse, error) {
	c, err := clientFor(ctx)
	if err != nil {
		return infer.DeleteResponse{}, err
	}
	if _, err := c.do(ctx, "DELETE", messagingProduct, lsDomainPath(req.ID, "Certificate"), nil, nil); err != nil && !isNotFound(err) {
		return infer.DeleteResponse{}, fmt.Errorf("delete link shortening certificate of domain %s: %w", req.ID, err)
	}
	return infer.DeleteResponse{}, nil
}

// ------------------------------------------------------------ LinkShorteningMessagingServiceDomain

// LinkShorteningMessagingServiceDomain associates a link shortening domain with a Messaging Service.
type LinkShorteningMessagingServiceDomain struct{}

// LinkShorteningMessagingServiceDomainArgs are the inputs of a LinkShorteningMessagingServiceDomain.
// Twilio cannot change an association, so every input replaces it.
type LinkShorteningMessagingServiceDomainArgs struct {
	DomainSid           string `pulumi:"domainSid" provider:"replaceOnChanges"`
	MessagingServiceSid string `pulumi:"messagingServiceSid" provider:"replaceOnChanges"`
}

func (a *LinkShorteningMessagingServiceDomainArgs) Annotate(an infer.Annotator) {
	an.Describe(&a.DomainSid, "SID (ZN...) of the link shortening domain. Changing it replaces the association.")
	an.Describe(&a.MessagingServiceSid, "SID (MG...) of the Messaging Service whose links the domain shortens. Changing it replaces the association.")
}

// LinkShorteningMessagingServiceDomainState is the persisted state of a LinkShorteningMessagingServiceDomain.
type LinkShorteningMessagingServiceDomainState struct {
	LinkShorteningMessagingServiceDomainArgs

	Url string `pulumi:"url"`
}

func (s *LinkShorteningMessagingServiceDomainState) Annotate(a infer.Annotator) {
	a.Describe(&s.Url, "The absolute URL of the association.")
}

func (r *LinkShorteningMessagingServiceDomain) Annotate(a infer.Annotator) {
	a.Describe(&r, "The association of a link shortening domain with a Messaging Service. The resource ID is <domainSid>/<messagingServiceSid>. A Messaging Service has at most one domain. Destroying the resource removes the association.")
}

var _ infer.CustomResource[LinkShorteningMessagingServiceDomainArgs, LinkShorteningMessagingServiceDomainState] = (*LinkShorteningMessagingServiceDomain)(nil)
var _ infer.CustomCheck[LinkShorteningMessagingServiceDomainArgs] = (*LinkShorteningMessagingServiceDomain)(nil)
var _ infer.CustomDiff[LinkShorteningMessagingServiceDomainArgs, LinkShorteningMessagingServiceDomainState] = (*LinkShorteningMessagingServiceDomain)(nil)
var _ infer.CustomRead[LinkShorteningMessagingServiceDomainArgs, LinkShorteningMessagingServiceDomainState] = (*LinkShorteningMessagingServiceDomain)(nil)
var _ infer.CustomDelete[LinkShorteningMessagingServiceDomainState] = (*LinkShorteningMessagingServiceDomain)(nil)

func lsAssociationPath(domain, svc string) string {
	return lsDomainPath(domain, "MessagingServices", svc)
}

func (r *LinkShorteningMessagingServiceDomain) Check(ctx context.Context, req infer.CheckRequest) (infer.CheckResponse[LinkShorteningMessagingServiceDomainArgs], error) {
	args, failures, err := infer.DefaultCheck[LinkShorteningMessagingServiceDomainArgs](ctx, req.NewInputs)
	if err != nil || len(failures) > 0 {
		return infer.CheckResponse[LinkShorteningMessagingServiceDomainArgs]{Inputs: args, Failures: failures}, err
	}
	c := &checker{unk: unknowns(req.NewInputs)}
	lsCheckDomain(c, args.DomainSid)
	c.sid("messagingServiceSid", args.MessagingServiceSid, verifyMessagingSidRE, "Messaging Service (MG followed by 32 hex characters)")
	return infer.CheckResponse[LinkShorteningMessagingServiceDomainArgs]{Inputs: args, Failures: c.failures}, nil
}

func (r *LinkShorteningMessagingServiceDomain) Diff(_ context.Context, req infer.DiffRequest[LinkShorteningMessagingServiceDomainArgs, LinkShorteningMessagingServiceDomainState]) (infer.DiffResponse, error) {
	o, n := req.State.LinkShorteningMessagingServiceDomainArgs, req.Inputs
	d := trDiffSet{}
	d.replace("domainSid", o.DomainSid, n.DomainSid)
	d.replace("messagingServiceSid", o.MessagingServiceSid, n.MessagingServiceSid)
	return d.response(), nil
}

func (r *LinkShorteningMessagingServiceDomain) Create(ctx context.Context, req infer.CreateRequest[LinkShorteningMessagingServiceDomainArgs]) (infer.CreateResponse[LinkShorteningMessagingServiceDomainState], error) {
	type resp = infer.CreateResponse[LinkShorteningMessagingServiceDomainState]
	a := req.Inputs
	st := LinkShorteningMessagingServiceDomainState{LinkShorteningMessagingServiceDomainArgs: a}
	if req.DryRun {
		return resp{Output: st}, nil
	}
	c, err := clientFor(ctx)
	if err != nil {
		return resp{}, err
	}
	o, err := c.do(ctx, "POST", messagingProduct, lsAssociationPath(a.DomainSid, a.MessagingServiceSid), nil, url.Values{})
	if err != nil {
		return resp{}, fmt.Errorf("associate link shortening domain %s with Messaging Service %s: %w", a.DomainSid, a.MessagingServiceSid, err)
	}
	st.Url = o.str("url")
	return resp{ID: a.DomainSid + "/" + a.MessagingServiceSid, Output: st}, nil
}

func (r *LinkShorteningMessagingServiceDomain) Read(ctx context.Context, req infer.ReadRequest[LinkShorteningMessagingServiceDomainArgs, LinkShorteningMessagingServiceDomainState]) (infer.ReadResponse[LinkShorteningMessagingServiceDomainArgs, LinkShorteningMessagingServiceDomainState], error) {
	type resp = infer.ReadResponse[LinkShorteningMessagingServiceDomainArgs, LinkShorteningMessagingServiceDomainState]
	parts, err := splitVerifyID(req.ID, 2, "<domainSid>/<messagingServiceSid>")
	if err != nil {
		return resp{}, err
	}
	c, err := clientFor(ctx)
	if err != nil {
		return resp{}, err
	}
	o, err := c.do(ctx, "GET", messagingProduct, "/v1/LinkShortening/MessagingServices/"+url.PathEscape(parts[1])+"/Domain", nil, nil)
	gone := false
	switch {
	case err != nil && isNotFound(err):
		gone = true
	case err != nil:
		return resp{}, fmt.Errorf("read link shortening association %s: %w", req.ID, err)
	case o.str("domain_sid") != parts[0]:
		gone = true // the Messaging Service uses another domain now
	}
	if gone {
		if err := notFoundOnImport("link shortening association", req.ID, req.Inputs, req.State); err != nil {
			return resp{}, err
		}
		return resp{}, nil
	}
	in := LinkShorteningMessagingServiceDomainArgs{DomainSid: parts[0], MessagingServiceSid: parts[1]}
	return resp{ID: req.ID, Inputs: in, State: LinkShorteningMessagingServiceDomainState{LinkShorteningMessagingServiceDomainArgs: in, Url: o.str("url")}}, nil
}

func (r *LinkShorteningMessagingServiceDomain) Delete(ctx context.Context, req infer.DeleteRequest[LinkShorteningMessagingServiceDomainState]) (infer.DeleteResponse, error) {
	parts, err := splitVerifyID(req.ID, 2, "<domainSid>/<messagingServiceSid>")
	if err != nil {
		return infer.DeleteResponse{}, err
	}
	c, err := clientFor(ctx)
	if err != nil {
		return infer.DeleteResponse{}, err
	}
	if _, err := c.do(ctx, "DELETE", messagingProduct, lsAssociationPath(parts[0], parts[1]), nil, nil); err != nil && !isNotFound(err) {
		return infer.DeleteResponse{}, fmt.Errorf("remove link shortening association %s: %w", req.ID, err)
	}
	return infer.DeleteResponse{}, nil
}
