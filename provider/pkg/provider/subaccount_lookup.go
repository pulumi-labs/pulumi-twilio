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

// GetSubaccount looks up a subaccount (or any account the credentials can see).
type GetSubaccount struct{}

// GetSubaccountArgs are the lookup arguments.
type GetSubaccountArgs struct {
	SubaccountSid string `pulumi:"subaccountSid"`
}

func (a *GetSubaccountArgs) Annotate(an infer.Annotator) {
	an.Describe(&a.SubaccountSid, "SID (AC...) of the subaccount to look up.")
}

// GetSubaccountResult is the looked-up subaccount. The auth token is deliberately not returned.
type GetSubaccountResult struct {
	Sid             string `pulumi:"sid"`
	FriendlyName    string `pulumi:"friendlyName"`
	Status          string `pulumi:"status"`
	Type            string `pulumi:"type"`
	OwnerAccountSid string `pulumi:"ownerAccountSid"`
	DateCreated     string `pulumi:"dateCreated"`
}

func (a *GetSubaccountResult) Annotate(an infer.Annotator) {
	an.Describe(&a.Sid, "Account SID (AC...).")
	an.Describe(&a.FriendlyName, "Account name.")
	an.Describe(&a.Status, "active, suspended or closed.")
	an.Describe(&a.Type, "Trial or Full.")
	an.Describe(&a.OwnerAccountSid, "SID of the parent account.")
	an.Describe(&a.DateCreated, "When the account was created (RFC 2822).")
}

func (f *GetSubaccount) Annotate(a infer.Annotator) {
	a.Describe(&f, "Look up a Twilio subaccount by SID. The auth token is deliberately not returned.")
}

func (f *GetSubaccount) Invoke(ctx context.Context, req infer.FunctionRequest[GetSubaccountArgs]) (infer.FunctionResponse[GetSubaccountResult], error) {
	const fn = "getSubaccount"
	if err := requireArg(fn, "subaccountSid", req.Input.SubaccountSid); err != nil {
		return infer.FunctionResponse[GetSubaccountResult]{}, err
	}
	c, err := clientFor(ctx)
	if err != nil {
		return infer.FunctionResponse[GetSubaccountResult]{}, err
	}
	sid := req.Input.SubaccountSid
	o, err := c.do(ctx, "GET", "api", accountPath(sid), nil, nil)
	if err != nil {
		return infer.FunctionResponse[GetSubaccountResult]{}, lookupFailure(fn, "subaccount "+sid, err)
	}
	return infer.FunctionResponse[GetSubaccountResult]{Output: GetSubaccountResult{
		Sid: o.str("sid"), FriendlyName: o.str("friendly_name"), Status: o.str("status"), Type: o.str("type"),
		OwnerAccountSid: o.str("owner_account_sid"), DateCreated: o.str("date_created"),
	}}, nil
}
