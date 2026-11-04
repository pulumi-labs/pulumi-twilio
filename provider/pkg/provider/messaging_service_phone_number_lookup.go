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

// GetMessagingServicePhoneNumber looks up a phone number in a Messaging Service sender pool.
type GetMessagingServicePhoneNumber struct{}

// GetMessagingServicePhoneNumberArgs are the lookup arguments.
type GetMessagingServicePhoneNumberArgs struct {
	ServiceSid     string `pulumi:"serviceSid"`
	PhoneNumberSid string `pulumi:"phoneNumberSid"`
}

func (a *GetMessagingServicePhoneNumberArgs) Annotate(an infer.Annotator) {
	an.Describe(&a.ServiceSid, "SID (MG...) of the Messaging Service.")
	an.Describe(&a.PhoneNumberSid, "SID (PN...) of the phone number in the sender pool.")
}

// GetMessagingServicePhoneNumberResult is the looked-up sender.
type GetMessagingServicePhoneNumberResult struct {
	ServiceSid     string   `pulumi:"serviceSid"`
	PhoneNumberSid string   `pulumi:"phoneNumberSid"`
	PhoneNumber    string   `pulumi:"phoneNumber"`
	Capabilities   []string `pulumi:"capabilities"`
}

func (a *GetMessagingServicePhoneNumberResult) Annotate(an infer.Annotator) {
	an.Describe(&a.ServiceSid, "SID of the Messaging Service.")
	an.Describe(&a.PhoneNumberSid, "SID of the phone number.")
	an.Describe(&a.PhoneNumber, "The sender in E.164 format.")
	an.Describe(&a.Capabilities, "Capabilities of the sender, for example SMS, MMS and Voice.")
}

func (f *GetMessagingServicePhoneNumber) Annotate(a infer.Annotator) {
	a.Describe(&f, "Look up a phone number in a Messaging Service sender pool.")
}

func (f *GetMessagingServicePhoneNumber) Invoke(ctx context.Context, req infer.FunctionRequest[GetMessagingServicePhoneNumberArgs]) (infer.FunctionResponse[GetMessagingServicePhoneNumberResult], error) {
	const fn = "getMessagingServicePhoneNumber"
	in := req.Input
	if err := requireArg(fn, "serviceSid", in.ServiceSid); err != nil {
		return infer.FunctionResponse[GetMessagingServicePhoneNumberResult]{}, err
	}
	if err := requireArg(fn, "phoneNumberSid", in.PhoneNumberSid); err != nil {
		return infer.FunctionResponse[GetMessagingServicePhoneNumberResult]{}, err
	}
	c, err := clientFor(ctx)
	if err != nil {
		return infer.FunctionResponse[GetMessagingServicePhoneNumberResult]{}, err
	}
	o, err := c.do(ctx, "GET", "messaging", senderPath(in.ServiceSid, in.PhoneNumberSid), nil, nil)
	if err != nil {
		return infer.FunctionResponse[GetMessagingServicePhoneNumberResult]{}, lookupFailure(fn, "phone number "+in.PhoneNumberSid+" in messaging service "+in.ServiceSid, err)
	}
	return infer.FunctionResponse[GetMessagingServicePhoneNumberResult]{Output: GetMessagingServicePhoneNumberResult{
		ServiceSid: in.ServiceSid, PhoneNumberSid: in.PhoneNumberSid, PhoneNumber: o.str("phone_number"), Capabilities: capsOf(o),
	}}, nil
}
