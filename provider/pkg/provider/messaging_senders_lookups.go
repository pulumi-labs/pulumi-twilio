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

// Lookups of the Messaging Service sender pool members beside getMessagingServicePhoneNumber.

func msSenderLookup(ctx context.Context, fn, what, coll, svc, sid string) (obj, error) {
	if err := requireArgs(fn, [2]string{"serviceSid", svc}, [2]string{"sid", sid}); err != nil {
		return nil, err
	}
	c, err := clientFor(ctx)
	if err != nil {
		return nil, err
	}
	o, err := c.do(ctx, "GET", "messaging", msSenderPath(svc, coll, sid), nil, nil)
	if err != nil {
		return nil, lookupFailure(fn, what+" "+sid+" in messaging service "+svc, err)
	}
	return o, nil
}

// GetMessagingServiceShortCode looks up a short code in a Messaging Service sender pool.
type GetMessagingServiceShortCode struct{}

// GetMessagingServiceShortCodeArgs are the lookup arguments.
type GetMessagingServiceShortCodeArgs struct {
	ServiceSid string `pulumi:"serviceSid"`
	Sid        string `pulumi:"sid"`
}

func (a *GetMessagingServiceShortCodeArgs) Annotate(an infer.Annotator) {
	an.Describe(&a.ServiceSid, "SID (MG...) of the Messaging Service.")
	an.Describe(&a.Sid, "SID (SC...) of the short code in the sender pool.")
}

// GetMessagingServiceShortCodeResult is the looked-up short code.
type GetMessagingServiceShortCodeResult struct {
	Sid          string   `pulumi:"sid"`
	ServiceSid   string   `pulumi:"serviceSid"`
	ShortCode    string   `pulumi:"shortCode"`
	CountryCode  string   `pulumi:"countryCode"`
	Capabilities []string `pulumi:"capabilities"`
	DateCreated  string   `pulumi:"dateCreated"`
	DateUpdated  string   `pulumi:"dateUpdated"`
}

func (f *GetMessagingServiceShortCode) Annotate(a infer.Annotator) {
	a.Describe(&f, "Look up a short code in a Messaging Service sender pool.")
}

func (f *GetMessagingServiceShortCode) Invoke(ctx context.Context, req infer.FunctionRequest[GetMessagingServiceShortCodeArgs]) (infer.FunctionResponse[GetMessagingServiceShortCodeResult], error) {
	in := req.Input
	o, err := msSenderLookup(ctx, "getMessagingServiceShortCode", "short code", "ShortCodes", in.ServiceSid, in.Sid)
	if err != nil {
		return infer.FunctionResponse[GetMessagingServiceShortCodeResult]{}, err
	}
	sid, created, updated := trSid(o)
	return infer.FunctionResponse[GetMessagingServiceShortCodeResult]{Output: GetMessagingServiceShortCodeResult{Sid: sid, ServiceSid: in.ServiceSid,
		ShortCode: o.str("short_code"), CountryCode: o.str("country_code"), Capabilities: capsOf(o), DateCreated: created, DateUpdated: updated}}, nil
}

// GetMessagingServiceChannelSender looks up a channel sender in a Messaging Service sender pool.
type GetMessagingServiceChannelSender struct{}

// GetMessagingServiceChannelSenderArgs are the lookup arguments.
type GetMessagingServiceChannelSenderArgs struct {
	ServiceSid string `pulumi:"serviceSid"`
	Sid        string `pulumi:"sid"`
}

func (a *GetMessagingServiceChannelSenderArgs) Annotate(an infer.Annotator) {
	an.Describe(&a.ServiceSid, "SID (MG...) of the Messaging Service.")
	an.Describe(&a.Sid, "SID (XE...) of the channel sender in the sender pool.")
}

// GetMessagingServiceChannelSenderResult is the looked-up channel sender.
type GetMessagingServiceChannelSenderResult struct {
	Sid         string `pulumi:"sid"`
	ServiceSid  string `pulumi:"serviceSid"`
	Sender      string `pulumi:"sender"`
	SenderType  string `pulumi:"senderType"`
	CountryCode string `pulumi:"countryCode"`
	DateCreated string `pulumi:"dateCreated"`
	DateUpdated string `pulumi:"dateUpdated"`
}

func (f *GetMessagingServiceChannelSender) Annotate(a infer.Annotator) {
	a.Describe(&f, "Look up a channel sender in a Messaging Service sender pool.")
}

func (f *GetMessagingServiceChannelSender) Invoke(ctx context.Context, req infer.FunctionRequest[GetMessagingServiceChannelSenderArgs]) (infer.FunctionResponse[GetMessagingServiceChannelSenderResult], error) {
	in := req.Input
	o, err := msSenderLookup(ctx, "getMessagingServiceChannelSender", "channel sender", "ChannelSenders", in.ServiceSid, in.Sid)
	if err != nil {
		return infer.FunctionResponse[GetMessagingServiceChannelSenderResult]{}, err
	}
	sid, created, updated := trSid(o)
	return infer.FunctionResponse[GetMessagingServiceChannelSenderResult]{Output: GetMessagingServiceChannelSenderResult{Sid: sid, ServiceSid: in.ServiceSid,
		Sender: o.str("sender"), SenderType: o.str("sender_type"), CountryCode: o.str("country_code"), DateCreated: created, DateUpdated: updated}}, nil
}

// GetMessagingServiceAlphaSender looks up an alpha sender ID in a Messaging Service.
type GetMessagingServiceAlphaSender struct{}

// GetMessagingServiceAlphaSenderArgs are the lookup arguments.
type GetMessagingServiceAlphaSenderArgs struct {
	ServiceSid string `pulumi:"serviceSid"`
	Sid        string `pulumi:"sid"`
}

func (a *GetMessagingServiceAlphaSenderArgs) Annotate(an infer.Annotator) {
	an.Describe(&a.ServiceSid, "SID (MG...) of the Messaging Service.")
	an.Describe(&a.Sid, "SID (AI...) of the alpha sender.")
}

// GetMessagingServiceAlphaSenderResult is the looked-up alpha sender.
type GetMessagingServiceAlphaSenderResult struct {
	Sid          string   `pulumi:"sid"`
	ServiceSid   string   `pulumi:"serviceSid"`
	AlphaSender  string   `pulumi:"alphaSender"`
	Capabilities []string `pulumi:"capabilities"`
	DateCreated  string   `pulumi:"dateCreated"`
	DateUpdated  string   `pulumi:"dateUpdated"`
}

func (f *GetMessagingServiceAlphaSender) Annotate(a infer.Annotator) {
	a.Describe(&f, "Look up an alphanumeric sender ID in a Messaging Service.")
}

func (f *GetMessagingServiceAlphaSender) Invoke(ctx context.Context, req infer.FunctionRequest[GetMessagingServiceAlphaSenderArgs]) (infer.FunctionResponse[GetMessagingServiceAlphaSenderResult], error) {
	in := req.Input
	o, err := msSenderLookup(ctx, "getMessagingServiceAlphaSender", "alpha sender", "AlphaSenders", in.ServiceSid, in.Sid)
	if err != nil {
		return infer.FunctionResponse[GetMessagingServiceAlphaSenderResult]{}, err
	}
	sid, created, updated := trSid(o)
	return infer.FunctionResponse[GetMessagingServiceAlphaSenderResult]{Output: GetMessagingServiceAlphaSenderResult{Sid: sid, ServiceSid: in.ServiceSid,
		AlphaSender: o.str("alpha_sender"), Capabilities: capsOf(o), DateCreated: created, DateUpdated: updated}}, nil
}

// GetMessagingServiceDestinationAlphaSender looks up a destination alpha sender ID in a Messaging Service.
type GetMessagingServiceDestinationAlphaSender struct{}

// GetMessagingServiceDestinationAlphaSenderArgs are the lookup arguments.
type GetMessagingServiceDestinationAlphaSenderArgs struct {
	ServiceSid string `pulumi:"serviceSid"`
	Sid        string `pulumi:"sid"`
}

func (a *GetMessagingServiceDestinationAlphaSenderArgs) Annotate(an infer.Annotator) {
	an.Describe(&a.ServiceSid, "SID (MG...) of the Messaging Service.")
	an.Describe(&a.Sid, "SID (AI...) of the destination alpha sender.")
}

// GetMessagingServiceDestinationAlphaSenderResult is the looked-up destination alpha sender.
type GetMessagingServiceDestinationAlphaSenderResult struct {
	Sid            string   `pulumi:"sid"`
	ServiceSid     string   `pulumi:"serviceSid"`
	AlphaSender    string   `pulumi:"alphaSender"`
	IsoCountryCode *string  `pulumi:"isoCountryCode,optional"`
	Capabilities   []string `pulumi:"capabilities"`
	DateCreated    string   `pulumi:"dateCreated"`
	DateUpdated    string   `pulumi:"dateUpdated"`
}

func (f *GetMessagingServiceDestinationAlphaSender) Annotate(a infer.Annotator) {
	a.Describe(&f, "Look up a destination alphanumeric sender ID in a Messaging Service. isoCountryCode is null for a default sender ID.")
}

func (f *GetMessagingServiceDestinationAlphaSender) Invoke(ctx context.Context, req infer.FunctionRequest[GetMessagingServiceDestinationAlphaSenderArgs]) (infer.FunctionResponse[GetMessagingServiceDestinationAlphaSenderResult], error) {
	in := req.Input
	o, err := msSenderLookup(ctx, "getMessagingServiceDestinationAlphaSender", "destination alpha sender", "DestinationAlphaSenders", in.ServiceSid, in.Sid)
	if err != nil {
		return infer.FunctionResponse[GetMessagingServiceDestinationAlphaSenderResult]{}, err
	}
	sid, created, updated := trSid(o)
	return infer.FunctionResponse[GetMessagingServiceDestinationAlphaSenderResult]{Output: GetMessagingServiceDestinationAlphaSenderResult{Sid: sid, ServiceSid: in.ServiceSid,
		AlphaSender: o.str("alpha_sender"), IsoCountryCode: nonEmpty(o.optStr("iso_country_code")), Capabilities: capsOf(o), DateCreated: created, DateUpdated: updated}}, nil
}
