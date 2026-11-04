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

// GetApplication looks up a TwiML application by SID.
type GetApplication struct{}

// GetApplicationArgs are the lookup arguments.
type GetApplicationArgs struct {
	ApplicationSid string  `pulumi:"applicationSid"`
	AccountSid     *string `pulumi:"accountSid,optional"`
}

func (a *GetApplicationArgs) Annotate(an infer.Annotator) {
	an.Describe(&a.ApplicationSid, "SID (AP...) of the application.")
	an.Describe(&a.AccountSid, "SID of the account that owns the application. Defaults to the provider account.")
}

// GetApplicationResult is the looked-up application: the configuration Twilio reports plus its identity.
type GetApplicationResult struct {
	ApplicationEffective

	Sid             string `pulumi:"sid"`
	OwnerAccountSid string `pulumi:"ownerAccountSid"`
}

func (a *GetApplicationResult) Annotate(an infer.Annotator) {
	an.Describe(&a.Sid, "The application SID (AP...).")
	an.Describe(&a.OwnerAccountSid, "SID of the account that owns the application.")
}

func (f *GetApplication) Annotate(a infer.Annotator) {
	a.Describe(&f, "Look up a TwiML application by SID.")
}

func (f *GetApplication) Invoke(ctx context.Context, req infer.FunctionRequest[GetApplicationArgs]) (infer.FunctionResponse[GetApplicationResult], error) {
	const fn = "getApplication"
	if err := requireArg(fn, "applicationSid", req.Input.ApplicationSid); err != nil {
		return infer.FunctionResponse[GetApplicationResult]{}, err
	}
	c, err := clientFor(ctx)
	if err != nil {
		return infer.FunctionResponse[GetApplicationResult]{}, err
	}
	acct := c.acct(req.Input.AccountSid)
	o, err := c.do(ctx, "GET", "api", appPath(acct, req.Input.ApplicationSid), nil, nil)
	if err != nil {
		return infer.FunctionResponse[GetApplicationResult]{}, lookupFailure(fn, "application "+req.Input.ApplicationSid, err)
	}
	return infer.FunctionResponse[GetApplicationResult]{Output: GetApplicationResult{
		ApplicationEffective: *applicationEffective(o), Sid: o.str("sid"), OwnerAccountSid: acct,
	}}, nil
}
