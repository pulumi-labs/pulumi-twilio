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
	"time"

	"github.com/pulumi/pulumi-go-provider/infer"
)

// Account-wide lookups of messaging.twilio.com v1 (twilio-oai twilio_messaging_v1 1.0.0) that are not
// tied to a Messaging Service resource.

// ------------------------------------------------------------ getMessagingUsecases

// GetMessagingUsecases lists the use cases a Messaging Service can declare.
type GetMessagingUsecases struct{}

// GetMessagingUsecasesArgs has no arguments.
type GetMessagingUsecasesArgs struct{}

// MessagingUsecase is one use case of a Messaging Service.
type MessagingUsecase struct {
	Usecase     *string `pulumi:"usecase,optional"`
	Description *string `pulumi:"description,optional"`
	Purpose     *string `pulumi:"purpose,optional"`
}

func (a *MessagingUsecase) Annotate(an infer.Annotator) {
	an.Describe(&a.Usecase, "The use case identifier, the value for the usecase of a Messaging Service.")
	an.Describe(&a.Description, "A human-readable description of the use case.")
	an.Describe(&a.Purpose, "The purpose of the use case.")
}

// GetMessagingUsecasesResult is the looked-up list.
type GetMessagingUsecasesResult struct {
	Usecases []MessagingUsecase `pulumi:"usecases"`
}

func (a *GetMessagingUsecasesResult) Annotate(an infer.Annotator) {
	an.Describe(&a.Usecases, "The use cases Twilio offers for a Messaging Service.")
}

func (f *GetMessagingUsecases) Annotate(a infer.Annotator) {
	a.Describe(&f, "List the use cases a Messaging Service can declare (messaging.twilio.com /v1/Services/Usecases).")
}

func (f *GetMessagingUsecases) Invoke(ctx context.Context, _ infer.FunctionRequest[GetMessagingUsecasesArgs]) (infer.FunctionResponse[GetMessagingUsecasesResult], error) {
	const fn = "getMessagingUsecases"
	type resp = infer.FunctionResponse[GetMessagingUsecasesResult]
	c, err := clientFor(ctx)
	if err != nil {
		return resp{}, err
	}
	o, err := c.do(ctx, "GET", "messaging", "/v1/Services/Usecases", nil, nil)
	if err != nil {
		return resp{}, lookupFailure(fn, "messaging use cases", err)
	}
	out := GetMessagingUsecasesResult{Usecases: []MessagingUsecase{}}
	if list, ok := o.get("usecases").([]any); ok {
		for _, e := range list {
			m, ok := e.(map[string]any)
			if !ok {
				continue
			}
			u := obj(m)
			out.Usecases = append(out.Usecases, MessagingUsecase{Usecase: nonEmpty(u.optStr("usecase")),
				Description: nonEmpty(u.optStr("description")), Purpose: nonEmpty(u.optStr("purpose"))})
		}
	}
	return resp{Output: out}, nil
}

// ------------------------------------------------------------ getMessagingDeactivations

// GetMessagingDeactivations returns the download URL of the US numbers deactivated on a day.
type GetMessagingDeactivations struct{}

// GetMessagingDeactivationsArgs are the lookup arguments.
type GetMessagingDeactivationsArgs struct {
	Date string `pulumi:"date"`
}

func (a *GetMessagingDeactivationsArgs) Annotate(an infer.Annotator) {
	an.Describe(&a.Date, "The day to report, as YYYY-MM-DD.")
}

// GetMessagingDeactivationsResult is the redirect Twilio answers with.
type GetMessagingDeactivationsResult struct {
	Date        string `pulumi:"date"`
	RedirectUrl string `pulumi:"redirectUrl" provider:"secret"`
}

func (a *GetMessagingDeactivationsResult) Annotate(an infer.Annotator) {
	an.Describe(&a.Date, "The day reported.")
	an.Describe(&a.RedirectUrl, "An authenticated URL of a file that lists the United States numbers deactivated on the day. Twilio says it is valid for about two minutes, so fetch it at once; it is not followed here and the file is not read. It is a Pulumi secret.")
}

func (f *GetMessagingDeactivations) Annotate(a infer.Annotator) {
	a.Describe(&f, "Get the URL of the list of United States numbers deactivated on a day (messaging.twilio.com /v1/Deactivations). Twilio answers with a redirect to the file; the lookup returns that URL without following it. The URL is short-lived and carries its own credentials, so the output is a Pulumi secret.")
}

func (f *GetMessagingDeactivations) Invoke(ctx context.Context, req infer.FunctionRequest[GetMessagingDeactivationsArgs]) (infer.FunctionResponse[GetMessagingDeactivationsResult], error) {
	const fn = "getMessagingDeactivations"
	type resp = infer.FunctionResponse[GetMessagingDeactivationsResult]
	date := req.Input.Date
	if err := requireArg(fn, "date", date); err != nil {
		return resp{}, err
	}
	if _, err := time.Parse("2006-01-02", date); err != nil {
		return resp{}, fmt.Errorf("%s: date must be a day as YYYY-MM-DD", fn)
	}
	c, err := clientFor(ctx)
	if err != nil {
		return resp{}, err
	}
	loc, err := c.redirectTarget(ctx, "messaging", "/v1/Deactivations", url.Values{"Date": {date}}, "redirect_to")
	if err != nil {
		return resp{}, lookupFailure(fn, "deactivations of "+date, err)
	}
	return resp{Output: GetMessagingDeactivationsResult{Date: date, RedirectUrl: loc}}, nil
}
