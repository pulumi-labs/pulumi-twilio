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

// GetMessagingService looks up a Messaging Service by SID.
type GetMessagingService struct{}

// GetMessagingServiceArgs are the lookup arguments.
type GetMessagingServiceArgs struct {
	ServiceSid string `pulumi:"serviceSid"`
}

func (a *GetMessagingServiceArgs) Annotate(an infer.Annotator) {
	an.Describe(&a.ServiceSid, "SID (MG...) of the Messaging Service. The Messaging API is scoped to the credentials' account.")
}

// GetMessagingServiceResult is the looked-up service: the configuration Twilio reports plus its SID.
type GetMessagingServiceResult struct {
	MessagingServiceEffective

	Sid string `pulumi:"sid"`
}

func (a *GetMessagingServiceResult) Annotate(an infer.Annotator) {
	an.Describe(&a.Sid, "The service SID (MG...).")
}

func (f *GetMessagingService) Annotate(a infer.Annotator) {
	a.Describe(&f, "Look up a Messaging Service by SID.")
}

func (f *GetMessagingService) Invoke(ctx context.Context, req infer.FunctionRequest[GetMessagingServiceArgs]) (infer.FunctionResponse[GetMessagingServiceResult], error) {
	const fn = "getMessagingService"
	if err := requireArg(fn, "serviceSid", req.Input.ServiceSid); err != nil {
		return infer.FunctionResponse[GetMessagingServiceResult]{}, err
	}
	c, err := clientFor(ctx)
	if err != nil {
		return infer.FunctionResponse[GetMessagingServiceResult]{}, err
	}
	o, err := c.do(ctx, "GET", "messaging", serviceBase(req.Input.ServiceSid), nil, nil)
	if err != nil {
		return infer.FunctionResponse[GetMessagingServiceResult]{}, lookupFailure(fn, "messaging service "+req.Input.ServiceSid, err)
	}
	return infer.FunctionResponse[GetMessagingServiceResult]{Output: GetMessagingServiceResult{
		MessagingServiceEffective: *messagingEffective(o), Sid: o.str("sid"),
	}}, nil
}
