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

	"github.com/pulumi/pulumi-go-provider/infer"
)

// Lookups of the Intelligence objects. Each reads one object and returns what Twilio reports, null
// where Twilio reports nothing.

// intelligenceLookup reads one object for a lookup function.
func intelligenceLookup(ctx context.Context, fn, what, path string) (obj, error) {
	c, err := clientFor(ctx)
	if err != nil {
		return nil, err
	}
	o, err := c.do(ctx, "GET", intelligenceProduct, path, nil, nil)
	if err != nil {
		return nil, lookupFailure(fn, what, err)
	}
	return o, nil
}

// ------------------------------------------------------------ getIntelligenceService

// GetIntelligenceService looks up a Voice Intelligence service by SID.
type GetIntelligenceService struct{}

// GetIntelligenceServiceArgs are the lookup arguments.
type GetIntelligenceServiceArgs struct {
	Sid string `pulumi:"sid"`
}

func (a *GetIntelligenceServiceArgs) Annotate(an infer.Annotator) {
	an.Describe(&a.Sid, "SID (GA...) of the Intelligence service.")
}

// GetIntelligenceServiceResult is the looked-up service.
type GetIntelligenceServiceResult struct {
	Sid                     string  `pulumi:"sid"`
	UniqueName              *string `pulumi:"uniqueName,optional"`
	FriendlyName            *string `pulumi:"friendlyName,optional"`
	LanguageCode            *string `pulumi:"languageCode,optional"`
	AutoTranscribe          *bool   `pulumi:"autoTranscribe,optional"`
	AutoRedaction           *bool   `pulumi:"autoRedaction,optional"`
	MediaRedaction          *bool   `pulumi:"mediaRedaction,optional"`
	DataLogging             *bool   `pulumi:"dataLogging,optional"`
	WebhookUrl              *string `pulumi:"webhookUrl,optional"`
	WebhookHttpMethod       *string `pulumi:"webhookHttpMethod,optional"`
	EncryptionCredentialSid *string `pulumi:"encryptionCredentialSid,optional"`
	DateCreated             string  `pulumi:"dateCreated"`
	DateUpdated             string  `pulumi:"dateUpdated"`
}

func (f *GetIntelligenceService) Annotate(a infer.Annotator) {
	a.Describe(&f, "Look up a Voice Intelligence service by SID.")
}

func (f *GetIntelligenceService) Invoke(ctx context.Context, req infer.FunctionRequest[GetIntelligenceServiceArgs]) (infer.FunctionResponse[GetIntelligenceServiceResult], error) {
	const fn = "getIntelligenceService"
	type resp = infer.FunctionResponse[GetIntelligenceServiceResult]
	if err := requireArg(fn, "sid", req.Input.Sid); err != nil {
		return resp{}, err
	}
	o, err := intelligenceLookup(ctx, fn, "Intelligence service "+req.Input.Sid, intelligencePath("Services", req.Input.Sid))
	if err != nil {
		return resp{}, err
	}
	sid, created, updated := trSid(o)
	return resp{Output: GetIntelligenceServiceResult{
		Sid:                     sid,
		UniqueName:              nonEmpty(o.optStr("unique_name")),
		FriendlyName:            nonEmpty(o.optStr("friendly_name")),
		LanguageCode:            nonEmpty(o.optStr("language_code")),
		AutoTranscribe:          o.optBool("auto_transcribe"),
		AutoRedaction:           o.optBool("auto_redaction"),
		MediaRedaction:          o.optBool("media_redaction"),
		DataLogging:             o.optBool("data_logging"),
		WebhookUrl:              nonEmpty(o.optStr("webhook_url")),
		WebhookHttpMethod:       nonEmpty(o.optStr("webhook_http_method")),
		EncryptionCredentialSid: nonEmpty(o.optStr("encryption_credential_sid")),
		DateCreated:             created,
		DateUpdated:             updated,
	}}, nil
}

// ------------------------------------------------------------ getIntelligenceCustomOperator

// GetIntelligenceCustomOperator looks up a custom Voice Intelligence operator by SID.
type GetIntelligenceCustomOperator struct{}

// GetIntelligenceCustomOperatorArgs are the lookup arguments.
type GetIntelligenceCustomOperatorArgs struct {
	Sid string `pulumi:"sid"`
}

func (a *GetIntelligenceCustomOperatorArgs) Annotate(an infer.Annotator) {
	an.Describe(&a.Sid, "SID (LY...) of the custom operator.")
}

// GetIntelligenceCustomOperatorResult is the looked-up operator.
type GetIntelligenceCustomOperatorResult struct {
	Sid          string  `pulumi:"sid"`
	FriendlyName *string `pulumi:"friendlyName,optional"`
	Description  *string `pulumi:"description,optional"`
	Author       *string `pulumi:"author,optional"`
	OperatorType *string `pulumi:"operatorType,optional"`
	Availability *string `pulumi:"availability,optional"`
	Config       *string `pulumi:"config,optional"`
	DateCreated  string  `pulumi:"dateCreated"`
	DateUpdated  string  `pulumi:"dateUpdated"`
}

func (f *GetIntelligenceCustomOperator) Annotate(a infer.Annotator) {
	a.Describe(&f, "Look up a custom Voice Intelligence operator by SID. config is the configuration as a JSON document.")
}

func (f *GetIntelligenceCustomOperator) Invoke(ctx context.Context, req infer.FunctionRequest[GetIntelligenceCustomOperatorArgs]) (infer.FunctionResponse[GetIntelligenceCustomOperatorResult], error) {
	const fn = "getIntelligenceCustomOperator"
	type resp = infer.FunctionResponse[GetIntelligenceCustomOperatorResult]
	if err := requireArg(fn, "sid", req.Input.Sid); err != nil {
		return resp{}, err
	}
	o, err := intelligenceLookup(ctx, fn, "Intelligence custom operator "+req.Input.Sid, intelligencePath("Operators/Custom", req.Input.Sid))
	if err != nil {
		return resp{}, err
	}
	sid, created, updated := trSid(o)
	return resp{Output: GetIntelligenceCustomOperatorResult{
		Sid:          sid,
		FriendlyName: nonEmpty(o.optStr("friendly_name")),
		Description:  nonEmpty(o.optStr("description")),
		Author:       nonEmpty(o.optStr("author")),
		OperatorType: nonEmpty(o.optStr("operator_type")),
		Availability: nonEmpty(o.optStr("availability")),
		Config:       objJSON(o, "config"),
		DateCreated:  created,
		DateUpdated:  updated,
	}}, nil
}
