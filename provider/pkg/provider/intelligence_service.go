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

	p "github.com/pulumi/pulumi-go-provider"
	"github.com/pulumi/pulumi-go-provider/infer"
)

// intelligenceDefaultLanguage is the language Twilio uses when a service is created without one.
const intelligenceDefaultLanguage = "en-US"

// IntelligenceService manages a Voice Intelligence service: the transcription settings that
// transcripts are processed with.
type IntelligenceService struct{}

// IntelligenceServiceArgs are the inputs of an IntelligenceService.
type IntelligenceServiceArgs struct {
	UniqueName              string  `pulumi:"uniqueName"`
	FriendlyName            *string `pulumi:"friendlyName,optional"`
	LanguageCode            *string `pulumi:"languageCode,optional"`
	AutoTranscribe          *bool   `pulumi:"autoTranscribe,optional"`
	AutoRedaction           *bool   `pulumi:"autoRedaction,optional"`
	MediaRedaction          *bool   `pulumi:"mediaRedaction,optional"`
	WebhookUrl              *string `pulumi:"webhookUrl,optional"`
	WebhookHttpMethod       *string `pulumi:"webhookHttpMethod,optional"`
	EncryptionCredentialSid *string `pulumi:"encryptionCredentialSid,optional"`
}

func (a *IntelligenceServiceArgs) Annotate(an infer.Annotator) {
	an.Describe(&a.UniqueName, "Unique, addressable name of the service, usable in place of the SID.")
	an.Describe(&a.FriendlyName, "Human-readable name of the service, up to 64 characters. Removing it leaves the name unchanged.")
	an.Describe(&a.LanguageCode, "Transcription language of every recording the service processes (Twilio default en-US). Twilio does not allow changing it once set, so changing it replaces the service.")
	an.Describe(&a.AutoTranscribe, "Whether Twilio transcribes every recording made on the account automatically (Twilio default false).")
	an.Describe(&a.AutoRedaction, "Whether Twilio redacts PII from transcripts automatically (Twilio default false).")
	an.Describe(&a.MediaRedaction, "Whether Twilio redacts PII from the media as well as the transcript (Twilio default false).")
	an.Describe(&a.WebhookUrl, "Absolute URL Twilio calls when a transcript completes. Removing it leaves the setting unchanged.")
	an.Describe(&a.WebhookHttpMethod, "HTTP method of the webhook call, GET or POST. Removing it leaves the setting unchanged.")
	an.Describe(&a.EncryptionCredentialSid, "SID (CR...) of the public key credential used to encrypt sentences and operator results. Removing it leaves the setting unchanged.")
}

// IntelligenceServiceState is the persisted state of an IntelligenceService.
type IntelligenceServiceState struct {
	IntelligenceServiceArgs

	Sid         string `pulumi:"sid"`
	DataLogging *bool  `pulumi:"dataLogging,optional"`
	DateCreated string `pulumi:"dateCreated"`
	DateUpdated string `pulumi:"dateUpdated"`
}

func (s *IntelligenceServiceState) Annotate(a infer.Annotator) {
	a.Describe(&s.Sid, "The service SID (GA...).")
	a.Describe(&s.DataLogging, "Whether Twilio may use the service's data to improve its speech models. Twilio only allows turning it on from the console, so it is reported and never set.")
	a.Describe(&s.DateCreated, "When the service was created (ISO 8601).")
	a.Describe(&s.DateUpdated, "When the service was last updated (ISO 8601).")
}

func (r *IntelligenceService) Annotate(a infer.Annotator) {
	a.Describe(&r, "A Voice Intelligence service. The resource ID is the service SID. Optional settings that are set are sent and kept in step with Twilio; one that is removed leaves the setting as it is on Twilio.")
}

var _ infer.CustomResource[IntelligenceServiceArgs, IntelligenceServiceState] = (*IntelligenceService)(nil)
var _ infer.CustomCheck[IntelligenceServiceArgs] = (*IntelligenceService)(nil)
var _ infer.CustomDiff[IntelligenceServiceArgs, IntelligenceServiceState] = (*IntelligenceService)(nil)
var _ infer.CustomRead[IntelligenceServiceArgs, IntelligenceServiceState] = (*IntelligenceService)(nil)
var _ infer.CustomUpdate[IntelligenceServiceArgs, IntelligenceServiceState] = (*IntelligenceService)(nil)
var _ infer.CustomDelete[IntelligenceServiceState] = (*IntelligenceService)(nil)

func (r *IntelligenceService) Check(ctx context.Context, req infer.CheckRequest) (infer.CheckResponse[IntelligenceServiceArgs], error) {
	args, failures, err := infer.DefaultCheck[IntelligenceServiceArgs](ctx, req.NewInputs)
	if err != nil || len(failures) > 0 {
		return infer.CheckResponse[IntelligenceServiceArgs]{Inputs: args, Failures: failures}, err
	}
	c := &checker{unk: unknowns(req.NewInputs)}
	c.required("uniqueName", args.UniqueName)
	if v := args.FriendlyName; v != nil {
		c.maxLen("friendlyName", *v, 64)
	}
	if v := args.WebhookHttpMethod; v != nil {
		c.oneOf("webhookHttpMethod", *v, []string{"GET", "POST"})
	}
	if v := args.EncryptionCredentialSid; v != nil {
		c.sid("encryptionCredentialSid", *v, intelligenceCredentialSidRE, "public key credential (CR followed by 32 hex characters)")
	}
	return infer.CheckResponse[IntelligenceServiceArgs]{Inputs: args, Failures: c.failures}, nil
}

func (r *IntelligenceService) Diff(_ context.Context, req infer.DiffRequest[IntelligenceServiceArgs, IntelligenceServiceState]) (infer.DiffResponse, error) {
	o, n := req.State.IntelligenceServiceArgs, req.Inputs
	d := trDiffSet{}
	d.requiredStr("uniqueName", o.UniqueName, n.UniqueName)
	d.strDef(intelligenceServiceDefaults, "friendlyName", o.FriendlyName, n.FriendlyName)
	if n.LanguageCode != nil && derefOr(o.LanguageCode, intelligenceServiceDefaults.str("languageCode")) != *n.LanguageCode {
		d["languageCode"] = p.PropertyDiff{Kind: p.UpdateReplace}
	}
	d.boolDef(intelligenceServiceDefaults, "autoTranscribe", o.AutoTranscribe, n.AutoTranscribe)
	d.boolDef(intelligenceServiceDefaults, "autoRedaction", o.AutoRedaction, n.AutoRedaction)
	d.boolDef(intelligenceServiceDefaults, "mediaRedaction", o.MediaRedaction, n.MediaRedaction)
	d.strDef(intelligenceServiceDefaults, "webhookUrl", o.WebhookUrl, n.WebhookUrl)
	d.strDef(intelligenceServiceDefaults, "webhookHttpMethod", o.WebhookHttpMethod, n.WebhookHttpMethod)
	d.strDef(intelligenceServiceDefaults, "encryptionCredentialSid", o.EncryptionCredentialSid, n.EncryptionCredentialSid)
	return d.response(), nil
}

// form is the request body; Twilio accepts the language only when a service is created.
func (a IntelligenceServiceArgs) form(create bool) url.Values {
	form := url.Values{}
	form.Set("UniqueName", a.UniqueName)
	setS(form, "FriendlyName", a.FriendlyName)
	if create {
		setS(form, "LanguageCode", a.LanguageCode)
	}
	setB(form, "AutoTranscribe", a.AutoTranscribe)
	setB(form, "AutoRedaction", a.AutoRedaction)
	setB(form, "MediaRedaction", a.MediaRedaction)
	setS(form, "WebhookUrl", a.WebhookUrl)
	setS(form, "WebhookHttpMethod", a.WebhookHttpMethod)
	setS(form, "EncryptionCredentialSid", a.EncryptionCredentialSid)
	return form
}

func (r *IntelligenceService) Create(ctx context.Context, req infer.CreateRequest[IntelligenceServiceArgs]) (infer.CreateResponse[IntelligenceServiceState], error) {
	type resp = infer.CreateResponse[IntelligenceServiceState]
	a := req.Inputs
	if req.DryRun {
		return resp{Output: IntelligenceServiceState{IntelligenceServiceArgs: a}}, nil
	}
	c, err := clientFor(ctx)
	if err != nil {
		return resp{}, err
	}
	o, err := c.do(ctx, "POST", intelligenceProduct, intelligencePath("Services", ""), nil, a.form(true))
	if err != nil {
		return resp{}, fmt.Errorf("create Intelligence service: %w", err)
	}
	st := IntelligenceServiceState{IntelligenceServiceArgs: a, DataLogging: o.optBool("data_logging")}
	st.Sid, st.DateCreated, st.DateUpdated = trSid(o)
	return resp{ID: st.Sid, Output: st}, nil
}

func (r *IntelligenceService) Read(ctx context.Context, req infer.ReadRequest[IntelligenceServiceArgs, IntelligenceServiceState]) (infer.ReadResponse[IntelligenceServiceArgs, IntelligenceServiceState], error) {
	type resp = infer.ReadResponse[IntelligenceServiceArgs, IntelligenceServiceState]
	c, err := clientFor(ctx)
	if err != nil {
		return resp{}, err
	}
	o, err := c.do(ctx, "GET", intelligenceProduct, intelligencePath("Services", req.ID), nil, nil)
	if err != nil {
		if isNotFound(err) {
			if err := notFoundOnImport("Intelligence service", req.ID, req.Inputs, req.State); err != nil {
				return resp{}, err
			}
			return resp{}, nil
		}
		return resp{}, fmt.Errorf("read Intelligence service %s: %w", req.ID, err)
	}
	in := req.Inputs
	// An import arrives with no prior state; the program's own settings are the only record of what it manages.
	imported := req.State.Sid == ""
	in.UniqueName = o.str("unique_name")
	in.FriendlyName = trStr(in.FriendlyName, o.optStr("friendly_name"), intelligenceServiceDefaults.str("friendlyName"), imported)
	in.LanguageCode = trStr(in.LanguageCode, o.optStr("language_code"), intelligenceServiceDefaults.str("languageCode"), imported)
	in.AutoTranscribe = trBool(in.AutoTranscribe, o.optBool("auto_transcribe"), intelligenceServiceDefaults.boolean("autoTranscribe"), imported)
	in.AutoRedaction = trBool(in.AutoRedaction, o.optBool("auto_redaction"), intelligenceServiceDefaults.boolean("autoRedaction"), imported)
	in.MediaRedaction = trBool(in.MediaRedaction, o.optBool("media_redaction"), intelligenceServiceDefaults.boolean("mediaRedaction"), imported)
	in.WebhookUrl = trStr(in.WebhookUrl, o.optStr("webhook_url"), intelligenceServiceDefaults.str("webhookUrl"), imported)
	in.WebhookHttpMethod = trStr(in.WebhookHttpMethod, o.optStr("webhook_http_method"), intelligenceServiceDefaults.str("webhookHttpMethod"), imported)
	in.EncryptionCredentialSid = trStr(in.EncryptionCredentialSid, o.optStr("encryption_credential_sid"), intelligenceServiceDefaults.str("encryptionCredentialSid"), imported)
	st := IntelligenceServiceState{IntelligenceServiceArgs: in, DataLogging: o.optBool("data_logging")}
	st.Sid, st.DateCreated, st.DateUpdated = trSid(o)
	return resp{ID: req.ID, Inputs: in, State: st}, nil
}

func (r *IntelligenceService) Update(ctx context.Context, req infer.UpdateRequest[IntelligenceServiceArgs, IntelligenceServiceState]) (infer.UpdateResponse[IntelligenceServiceState], error) {
	type resp = infer.UpdateResponse[IntelligenceServiceState]
	st := req.State
	st.IntelligenceServiceArgs = req.Inputs
	if req.DryRun {
		return resp{Output: st}, nil
	}
	c, err := clientFor(ctx)
	if err != nil {
		return resp{}, err
	}
	o, err := c.do(ctx, "POST", intelligenceProduct, intelligencePath("Services", req.ID), nil, req.Inputs.form(false))
	if err != nil {
		return resp{}, fmt.Errorf("update Intelligence service %s: %w", req.ID, err)
	}
	st.DateUpdated = o.str("date_updated")
	st.DataLogging = o.optBool("data_logging")
	return resp{Output: st}, nil
}

func (r *IntelligenceService) Delete(ctx context.Context, req infer.DeleteRequest[IntelligenceServiceState]) (infer.DeleteResponse, error) {
	c, err := clientFor(ctx)
	if err != nil {
		return infer.DeleteResponse{}, err
	}
	if _, err := c.do(ctx, "DELETE", intelligenceProduct, intelligencePath("Services", req.ID), nil, nil); err != nil && !isNotFound(err) {
		return infer.DeleteResponse{}, fmt.Errorf("delete Intelligence service %s: %w", req.ID, err)
	}
	return infer.DeleteResponse{}, nil
}

// intelligenceServiceDefaults are the Twilio defaults of this resource, shared by Read (import adoption) and Diff.
var intelligenceServiceDefaults = trDefaults{
	"friendlyName":            "",
	"autoTranscribe":          false,
	"autoRedaction":           false,
	"mediaRedaction":          false,
	"webhookUrl":              "",
	"webhookHttpMethod":       "",
	"encryptionCredentialSid": "",
	"languageCode":            intelligenceDefaultLanguage,
}
