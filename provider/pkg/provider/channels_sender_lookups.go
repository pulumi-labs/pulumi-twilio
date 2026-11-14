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

	"github.com/pulumi/pulumi-go-provider/infer"
)

// Lookups of the Messaging v2 Channels Senders API. They read WhatsApp and RCS senders and return
// what Twilio reports, null where Twilio reports nothing.

// ChannelsSenderCarrier is the status of an RCS sender at one carrier.
type ChannelsSenderCarrier struct {
	Name   *string `pulumi:"name,optional"`
	Status *string `pulumi:"status,optional"`
}

func (a *ChannelsSenderCarrier) Annotate(an infer.Annotator) {
	an.Describe(&a.Name, "Name of the carrier, for example Verizon.")
	an.Describe(&a.Status, "Carrier status: UNKNOWN, UNLAUNCHED, CARRIER_REVIEW, APPROVED, REJECTED or SUSPENDED.")
}

// ChannelsSenderComplianceCountry is the compliance state of an RCS sender in one country.
type ChannelsSenderComplianceCountry struct {
	Country         string                  `pulumi:"country"`
	RegistrationSid *string                 `pulumi:"registrationSid,optional"`
	Status          *string                 `pulumi:"status,optional"`
	Carriers        []ChannelsSenderCarrier `pulumi:"carriers,optional"`
}

func (a *ChannelsSenderComplianceCountry) Annotate(an infer.Annotator) {
	an.Describe(&a.Country, "ISO 3166-1 alpha-2 country code.")
	an.Describe(&a.RegistrationSid, "Compliance registration SID for the country.")
	an.Describe(&a.Status, "Country status, aggregated from the carriers: ONLINE, OFFLINE, TWILIO_REVIEW or PENDING_VERIFICATION.")
	an.Describe(&a.Carriers, "Status per carrier.")
}

// ChannelsSenderCompliance is the compliance information of an RCS sender.
type ChannelsSenderCompliance struct {
	RegistrationSid string                            `pulumi:"registrationSid"`
	Countries       []ChannelsSenderComplianceCountry `pulumi:"countries,optional"`
}

func (a *ChannelsSenderCompliance) Annotate(an infer.Annotator) {
	an.Describe(&a.RegistrationSid, "Default compliance registration SID, which applies to every country without its own.")
	an.Describe(&a.Countries, "Country-specific compliance details.")
}

func decodeChannelsSenderCompliance(o obj) *ChannelsSenderCompliance {
	out := &ChannelsSenderCompliance{RegistrationSid: o.str("registration_sid")}
	decodeLabeled(o, "countries", func(e obj) {
		cc := ChannelsSenderComplianceCountry{Country: e.str("country"), RegistrationSid: optNonEmpty(e, "registration_sid"), Status: optNonEmpty(e, "status")}
		decodeLabeled(e, "carriers", func(k obj) {
			cc.Carriers = append(cc.Carriers, ChannelsSenderCarrier{Name: optNonEmpty(k, "name"), Status: optNonEmpty(k, "status")})
		})
		out.Countries = append(out.Countries, cc)
	})
	return out
}

// ChannelsSenderInfo is one looked-up sender.
type ChannelsSenderInfo struct {
	Sid                          string                        `pulumi:"sid"`
	Status                       string                        `pulumi:"status"`
	SenderId                     string                        `pulumi:"senderId"`
	FriendlyName                 *string                       `pulumi:"friendlyName,optional"`
	Configuration                *ChannelsSenderConfiguration  `pulumi:"configuration,optional"`
	Webhook                      *ChannelsSenderWebhook        `pulumi:"webhook,optional"`
	Profile                      *ChannelsSenderProfile        `pulumi:"profile,optional"`
	ProfileUseCase               *string                       `pulumi:"profileUseCase,optional"`
	PendingDisplayName           *string                       `pulumi:"pendingDisplayName,optional"`
	PendingDisplayNameStatus     *string                       `pulumi:"pendingDisplayNameStatus,optional"`
	PendingDisplayNameStatusDate *string                       `pulumi:"pendingDisplayNameStatusDate,optional"`
	Properties                   *ChannelsSenderProperties     `pulumi:"properties,optional"`
	OfflineReasons               []ChannelsSenderOfflineReason `pulumi:"offlineReasons,optional"`
	Compliance                   *ChannelsSenderCompliance     `pulumi:"compliance,optional"`
	Url                          string                        `pulumi:"url"`
}

func (a *ChannelsSenderInfo) Annotate(an infer.Annotator) {
	an.Describe(&a.Sid, "The sender SID (XE...).")
	an.Describe(&a.Status, "Status of the sender: CREATING, ONLINE, OFFLINE, PENDING_VERIFICATION, VERIFYING, ONLINE:UPDATING, TWILIO_REVIEW, DRAFT or STUBBED.")
	an.Describe(&a.SenderId, "The sender ID, as whatsapp:<E.164 number> or the RCS sender ID.")
	an.Describe(&a.FriendlyName, "Label of the sender in the Twilio Console.")
	an.Describe(&a.Configuration, "Configuration of the sender. The verification code is never returned.")
	an.Describe(&a.Webhook, "Webhooks of the sender.")
	an.Describe(&a.Profile, "Business profile of the sender.")
	an.Describe(&a.ProfileUseCase, "The RCS messaging use case: PROMOTIONAL, TRANSACTIONAL, OTP or MULTI_USE. Null for WhatsApp.")
	an.Describe(&a.PendingDisplayName, "WhatsApp only: the display name the latest name change applies to. Null when there was no name change.")
	an.Describe(&a.PendingDisplayNameStatus, "WhatsApp only: status of the latest display name change.")
	an.Describe(&a.PendingDisplayNameStatusDate, "WhatsApp only: when that status last changed.")
	an.Describe(&a.Properties, "Quality rating and messaging limit of the sender.")
	an.Describe(&a.OfflineReasons, "Why the sender is offline, when it is.")
	an.Describe(&a.Compliance, "RCS only: compliance registration and carrier status per country.")
	an.Describe(&a.Url, "The URL of the sender.")
}

func channelsSenderInfo(g channelsSenderResponse) ChannelsSenderInfo {
	p := g.Profile
	return ChannelsSenderInfo{Sid: g.Sid, Status: g.Status, SenderId: g.SenderId, FriendlyName: g.FriendlyName,
		Configuration: g.Configuration, Webhook: g.Webhook, Profile: &p, ProfileUseCase: g.ProfileUseCase,
		PendingDisplayName: g.PendingDisplayName, PendingDisplayNameStatus: g.PendingDisplayNameStatus,
		PendingDisplayNameStatusDate: g.PendingDisplayNameStatusDate, Properties: g.Properties,
		OfflineReasons: g.OfflineReasons, Compliance: g.Compliance, Url: g.Url}
}

// ------------------------------------------------------------------- getChannelsSender

// GetChannelsSender looks up a WhatsApp or RCS sender by SID.
type GetChannelsSender struct{}

// GetChannelsSenderArgs are the lookup arguments.
type GetChannelsSenderArgs struct {
	Sid string `pulumi:"sid"`
}

func (a *GetChannelsSenderArgs) Annotate(an infer.Annotator) {
	an.Describe(&a.Sid, "SID (XE...) of the sender.")
}

// GetChannelsSenderResult is the looked-up sender.
type GetChannelsSenderResult struct {
	ChannelsSenderInfo
}

func (f *GetChannelsSender) Annotate(a infer.Annotator) {
	a.Describe(&f, "Look up a WhatsApp or RCS sender by SID (Messaging v2 Channels Senders).")
}

func (f *GetChannelsSender) Invoke(ctx context.Context, req infer.FunctionRequest[GetChannelsSenderArgs]) (infer.FunctionResponse[GetChannelsSenderResult], error) {
	const fn = "getChannelsSender"
	type resp = infer.FunctionResponse[GetChannelsSenderResult]
	if err := requireArg(fn, "sid", req.Input.Sid); err != nil {
		return resp{}, err
	}
	if !channelsSenderSidRe.MatchString(req.Input.Sid) {
		return resp{}, fmt.Errorf("%s: sid must be the SID of a sender (XE followed by 32 hex characters)", fn)
	}
	c, err := clientFor(ctx)
	if err != nil {
		return resp{}, err
	}
	o, err := c.doJSON(ctx, "GET", channelsSenderProduct, channelsSenderPath(req.Input.Sid), nil, nil)
	if err != nil {
		return resp{}, lookupFailure(fn, "channels sender "+req.Input.Sid, err)
	}
	return resp{Output: GetChannelsSenderResult{ChannelsSenderInfo: channelsSenderInfo(decodeChannelsSender(o))}}, nil
}

// ------------------------------------------------------------------ getChannelsSenders

// GetChannelsSenders lists the senders of one channel.
type GetChannelsSenders struct{}

// GetChannelsSendersArgs are the lookup arguments.
type GetChannelsSendersArgs struct {
	Channel string `pulumi:"channel"`
}

func (a *GetChannelsSendersArgs) Annotate(an infer.Annotator) {
	an.Describe(&a.Channel, "The channel to list senders of: whatsapp or rcs.")
}

// GetChannelsSendersResult is the list of senders.
type GetChannelsSendersResult struct {
	Senders []ChannelsSenderInfo `pulumi:"senders"`
}

func (a *GetChannelsSendersResult) Annotate(an infer.Annotator) {
	an.Describe(&a.Senders, "The senders of the channel, empty when there are none.")
}

func (f *GetChannelsSenders) Annotate(a infer.Annotator) {
	a.Describe(&f, "List the WhatsApp or RCS senders of the account (Messaging v2 Channels Senders). Every page is read.")
}

func (f *GetChannelsSenders) Invoke(ctx context.Context, req infer.FunctionRequest[GetChannelsSendersArgs]) (infer.FunctionResponse[GetChannelsSendersResult], error) {
	const fn = "getChannelsSenders"
	type resp = infer.FunctionResponse[GetChannelsSendersResult]
	if err := requireArg(fn, "channel", req.Input.Channel); err != nil {
		return resp{}, err
	}
	if ch := req.Input.Channel; ch != "whatsapp" && ch != "rcs" {
		return resp{}, fmt.Errorf("%s: channel must be whatsapp or rcs", fn)
	}
	c, err := clientFor(ctx)
	if err != nil {
		return resp{}, err
	}
	out := []ChannelsSenderInfo{}
	// No PageSize is sent: the spec's cap is not trusted, the next page URL is followed instead.
	err = listPages(ctx, c, fn, channelsSenderProduct, channelsSenderPath(""), url.Values{"Channel": {req.Input.Channel}}, "senders", func(o obj) bool {
		out = append(out, channelsSenderInfo(decodeChannelsSender(o)))
		return true
	})
	if err != nil {
		return resp{}, lookupFailure(fn, "channels senders", err)
	}
	return resp{Output: GetChannelsSendersResult{Senders: out}}, nil
}
