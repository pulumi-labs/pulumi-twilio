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
	"net/url"

	"github.com/pulumi/pulumi-go-provider/infer"
)

// Lookups of link shortening (messaging.twilio.com v1 /LinkShortening).

// LinkShorteningDomainConfigInfo is the configuration of a link shortening domain.
type LinkShorteningDomainConfigInfo struct {
	DomainSid           string `pulumi:"domainSid"`
	ConfigSid           string `pulumi:"configSid"`
	MessagingServiceSid string `pulumi:"messagingServiceSid"`
	FallbackUrl         string `pulumi:"fallbackUrl"`
	CallbackUrl         string `pulumi:"callbackUrl"`
	ContinueOnFailure   bool   `pulumi:"continueOnFailure"`
	DisableHttps        bool   `pulumi:"disableHttps"`
	DateCreated         string `pulumi:"dateCreated"`
	DateUpdated         string `pulumi:"dateUpdated"`
}

func (a *LinkShorteningDomainConfigInfo) Annotate(an infer.Annotator) {
	an.Describe(&a.DomainSid, "SID (ZN...) of the link shortening domain.")
	an.Describe(&a.ConfigSid, "The SID (ZK...) of the domain configuration.")
	an.Describe(&a.MessagingServiceSid, "SID (MG...) of the Messaging Service the configuration was read for (empty when read by domain).")
	an.Describe(&a.FallbackUrl, "Requests that match no shortened message are redirected to this URL (empty when none).")
	an.Describe(&a.CallbackUrl, "URL that receives click events (empty when none).")
	an.Describe(&a.ContinueOnFailure, "Whether a message is delivered without a short link when link shortening fails.")
	an.Describe(&a.DisableHttps, "Whether shortened links are sent without the https:// prefix.")
	an.Describe(&a.DateCreated, "When the configuration was created (ISO 8601).")
	an.Describe(&a.DateUpdated, "When the configuration was last updated (ISO 8601).")
}

func lsConfigInfo(o obj) LinkShorteningDomainConfigInfo {
	return LinkShorteningDomainConfigInfo{DomainSid: o.str("domain_sid"), ConfigSid: o.str("config_sid"), MessagingServiceSid: o.str("messaging_service_sid"),
		FallbackUrl: o.str("fallback_url"), CallbackUrl: o.str("callback_url"), ContinueOnFailure: brandBool(o, "continue_on_failure"),
		DisableHttps: brandBool(o, "disable_https"), DateCreated: o.str("date_created"), DateUpdated: o.str("date_updated")}
}

// lsLookupByDomain is the shared body of the lookups that take a domain SID.
func lsLookupByDomain[T any](ctx context.Context, fn, domain string, sub string, conv func(obj) T) (T, error) {
	var zero T
	if err := requireArg(fn, "domainSid", domain); err != nil {
		return zero, err
	}
	c, err := clientFor(ctx)
	if err != nil {
		return zero, err
	}
	o, err := c.do(ctx, "GET", messagingProduct, lsDomainPath(domain, sub), nil, nil)
	if err != nil {
		return zero, lookupFailure(fn, "link shortening domain "+domain, err)
	}
	return conv(o), nil
}

// lsLookupByService is the shared body of the lookups that take a Messaging Service SID.
func lsLookupByService[T any](ctx context.Context, fn, svc, path string, conv func(obj) T) (T, error) {
	var zero T
	if err := requireArg(fn, "messagingServiceSid", svc); err != nil {
		return zero, err
	}
	c, err := clientFor(ctx)
	if err != nil {
		return zero, err
	}
	o, err := c.do(ctx, "GET", messagingProduct, path, url.Values{}, nil)
	if err != nil {
		return zero, lookupFailure(fn, "Messaging Service "+svc, err)
	}
	return conv(o), nil
}

// GetLinkShorteningDomainConfig looks up the configuration of a link shortening domain.
type GetLinkShorteningDomainConfig struct{}

// GetLinkShorteningDomainArgs are the lookup arguments of the lookups by domain.
type GetLinkShorteningDomainArgs struct {
	DomainSid string `pulumi:"domainSid"`
}

func (a *GetLinkShorteningDomainArgs) Annotate(an infer.Annotator) {
	an.Describe(&a.DomainSid, "SID (ZN...) of the link shortening domain.")
}

// GetLinkShorteningMessagingServiceArgs are the lookup arguments of the lookups by Messaging Service.
type GetLinkShorteningMessagingServiceArgs struct {
	MessagingServiceSid string `pulumi:"messagingServiceSid"`
}

func (a *GetLinkShorteningMessagingServiceArgs) Annotate(an infer.Annotator) {
	an.Describe(&a.MessagingServiceSid, "SID (MG...) of the Messaging Service.")
}

func (f *GetLinkShorteningDomainConfig) Annotate(a infer.Annotator) {
	a.Describe(&f, "Look up the configuration of a link shortening domain (GET /v1/LinkShortening/Domains/{DomainSid}/Config on messaging.twilio.com).")
}

func (f *GetLinkShorteningDomainConfig) Invoke(ctx context.Context, req infer.FunctionRequest[GetLinkShorteningDomainArgs]) (infer.FunctionResponse[LinkShorteningDomainConfigInfo], error) {
	out, err := lsLookupByDomain(ctx, "getLinkShorteningDomainConfig", req.Input.DomainSid, "Config", lsConfigInfo)
	return infer.FunctionResponse[LinkShorteningDomainConfigInfo]{Output: out}, err
}

// GetLinkShorteningMessagingServiceDomainConfig looks up the domain configuration a Messaging Service uses.
type GetLinkShorteningMessagingServiceDomainConfig struct{}

func (f *GetLinkShorteningMessagingServiceDomainConfig) Annotate(a infer.Annotator) {
	a.Describe(&f, "Look up the link shortening configuration that applies to a Messaging Service (GET /v1/LinkShortening/MessagingService/{MessagingServiceSid}/DomainConfig on messaging.twilio.com).")
}

func (f *GetLinkShorteningMessagingServiceDomainConfig) Invoke(ctx context.Context, req infer.FunctionRequest[GetLinkShorteningMessagingServiceArgs]) (infer.FunctionResponse[LinkShorteningDomainConfigInfo], error) {
	svc := req.Input.MessagingServiceSid
	out, err := lsLookupByService(ctx, "getLinkShorteningMessagingServiceDomainConfig", svc, "/v1/LinkShortening/MessagingService/"+url.PathEscape(svc)+"/DomainConfig", lsConfigInfo)
	return infer.FunctionResponse[LinkShorteningDomainConfigInfo]{Output: out}, err
}

// LinkShorteningDomainCertificateInfo is the certificate of a link shortening domain.
type LinkShorteningDomainCertificateInfo struct {
	DomainSid        string `pulumi:"domainSid"`
	CertificateSid   string `pulumi:"certificateSid"`
	DomainName       string `pulumi:"domainName"`
	DateExpires      string `pulumi:"dateExpires"`
	CertInValidation string `pulumi:"certInValidation"`
	DateCreated      string `pulumi:"dateCreated"`
	DateUpdated      string `pulumi:"dateUpdated"`
}

func (a *LinkShorteningDomainCertificateInfo) Annotate(an infer.Annotator) {
	an.Describe(&a.DomainSid, "SID (ZN...) of the link shortening domain.")
	an.Describe(&a.CertificateSid, "The certificate SID.")
	an.Describe(&a.DomainName, "The domain name of the certificate.")
	an.Describe(&a.DateExpires, "When the certificate expires (ISO 8601).")
	an.Describe(&a.CertInValidation, "A JSON document with the status of a new certificate that is being validated (empty when none).")
	an.Describe(&a.DateCreated, "When the domain was registered with Twilio (ISO 8601).")
	an.Describe(&a.DateUpdated, "When the certificate was last updated (ISO 8601).")
}

// GetLinkShorteningDomainCertificate looks up the certificate of a link shortening domain.
type GetLinkShorteningDomainCertificate struct{}

func (f *GetLinkShorteningDomainCertificate) Annotate(a infer.Annotator) {
	a.Describe(&f, "Look up the certificate of a link shortening domain, without the certificate itself (GET /v1/LinkShortening/Domains/{DomainSid}/Certificate on messaging.twilio.com).")
}

func (f *GetLinkShorteningDomainCertificate) Invoke(ctx context.Context, req infer.FunctionRequest[GetLinkShorteningDomainArgs]) (infer.FunctionResponse[LinkShorteningDomainCertificateInfo], error) {
	out, err := lsLookupByDomain(ctx, "getLinkShorteningDomainCertificate", req.Input.DomainSid, "Certificate", func(o obj) LinkShorteningDomainCertificateInfo {
		return LinkShorteningDomainCertificateInfo{DomainSid: o.str("domain_sid"), CertificateSid: o.str("certificate_sid"), DomainName: o.str("domain_name"),
			DateExpires: o.str("date_expires"), CertInValidation: a2pJSON(o.get("cert_in_validation")), DateCreated: o.str("date_created"), DateUpdated: o.str("date_updated")}
	})
	return infer.FunctionResponse[LinkShorteningDomainCertificateInfo]{Output: out}, err
}

// LinkShorteningDomainDnsValidationInfo is the DNS validation of a link shortening domain.
type LinkShorteningDomainDnsValidationInfo struct {
	DomainSid string `pulumi:"domainSid"`
	IsValid   bool   `pulumi:"isValid"`
	Reason    string `pulumi:"reason"`
}

func (a *LinkShorteningDomainDnsValidationInfo) Annotate(an infer.Annotator) {
	an.Describe(&a.DomainSid, "SID (ZN...) of the link shortening domain.")
	an.Describe(&a.IsValid, "Whether the DNS records of the domain are valid.")
	an.Describe(&a.Reason, "Why the DNS records are not valid (empty when they are).")
}

// GetLinkShorteningDomainDnsValidation checks the DNS records of a link shortening domain.
type GetLinkShorteningDomainDnsValidation struct{}

func (f *GetLinkShorteningDomainDnsValidation) Annotate(a infer.Annotator) {
	a.Describe(&f, "Validate the DNS records of a link shortening domain (GET /v1/LinkShortening/Domains/{DomainSid}/ValidateDns on messaging.twilio.com).")
}

func (f *GetLinkShorteningDomainDnsValidation) Invoke(ctx context.Context, req infer.FunctionRequest[GetLinkShorteningDomainArgs]) (infer.FunctionResponse[LinkShorteningDomainDnsValidationInfo], error) {
	out, err := lsLookupByDomain(ctx, "getLinkShorteningDomainDnsValidation", req.Input.DomainSid, "ValidateDns", func(o obj) LinkShorteningDomainDnsValidationInfo {
		return LinkShorteningDomainDnsValidationInfo{DomainSid: o.str("domain_sid"), IsValid: brandBool(o, "is_valid"), Reason: o.str("reason")}
	})
	return infer.FunctionResponse[LinkShorteningDomainDnsValidationInfo]{Output: out}, err
}

// LinkShorteningMessagingServiceDomainInfo is the domain associated with a Messaging Service.
type LinkShorteningMessagingServiceDomainInfo struct {
	DomainSid           string `pulumi:"domainSid"`
	MessagingServiceSid string `pulumi:"messagingServiceSid"`
}

func (a *LinkShorteningMessagingServiceDomainInfo) Annotate(an infer.Annotator) {
	an.Describe(&a.DomainSid, "SID (ZN...) of the link shortening domain of the Messaging Service.")
	an.Describe(&a.MessagingServiceSid, "SID (MG...) of the Messaging Service.")
}

// GetLinkShorteningMessagingServiceDomain looks up the link shortening domain of a Messaging Service.
type GetLinkShorteningMessagingServiceDomain struct{}

func (f *GetLinkShorteningMessagingServiceDomain) Annotate(a infer.Annotator) {
	a.Describe(&f, "Look up the link shortening domain associated with a Messaging Service (GET /v1/LinkShortening/MessagingServices/{MessagingServiceSid}/Domain on messaging.twilio.com).")
}

func (f *GetLinkShorteningMessagingServiceDomain) Invoke(ctx context.Context, req infer.FunctionRequest[GetLinkShorteningMessagingServiceArgs]) (infer.FunctionResponse[LinkShorteningMessagingServiceDomainInfo], error) {
	svc := req.Input.MessagingServiceSid
	out, err := lsLookupByService(ctx, "getLinkShorteningMessagingServiceDomain", svc, "/v1/LinkShortening/MessagingServices/"+url.PathEscape(svc)+"/Domain", func(o obj) LinkShorteningMessagingServiceDomainInfo {
		return LinkShorteningMessagingServiceDomainInfo{DomainSid: o.str("domain_sid"), MessagingServiceSid: o.str("messaging_service_sid")}
	})
	return infer.FunctionResponse[LinkShorteningMessagingServiceDomainInfo]{Output: out}, err
}
