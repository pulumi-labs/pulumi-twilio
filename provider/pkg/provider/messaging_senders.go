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
	"regexp"
	"strings"

	"github.com/pulumi/pulumi-go-provider/infer"
)

// Messaging Service sender pool members beside MessagingServicePhoneNumber: short codes, alpha
// sender IDs, destination alpha sender IDs and channel senders (messaging.twilio.com v1,
// /v1/Services/{ServiceSid}/<collection>). Twilio offers create, read and delete only, so every
// input identifies the member and any change replaces it; deleting detaches the sender without
// releasing it.

var (
	msShortCodeSidRE     = regexp.MustCompile(`^SC[0-9a-fA-F]{32}$`)
	msChannelSenderSidRE = regexp.MustCompile(`^XE[0-9a-fA-F]{32}$`)
	msAlphaSenderRE      = regexp.MustCompile(`^[A-Za-z0-9 +_&-]{1,11}$`)
	msAlphaOnlyDigitsRE  = regexp.MustCompile(`^[0-9]+$`)
	msCountryRE          = regexp.MustCompile(`^[A-Za-z]{2}$`)
)

// msSenderPath is /v1/Services/<service>/<collection>[/<sid>].
func msSenderPath(svc, coll, sid string) string {
	p := serviceBase(svc) + "/" + coll
	if sid != "" {
		p += "/" + url.PathEscape(sid)
	}
	return p
}

// msSenderIDShape is the import ID shape of every sender pool member.
const msSenderIDShape = "<serviceSid>/<sid>"

// msSenderGet reads one pool member. found is false when Twilio answers 404; the caller then decides
// between an import error and a deletion outside Pulumi.
func msSenderGet(ctx context.Context, what, coll, id string) (svc, sid string, o obj, found bool, err error) {
	ids, err := splitVerifyID(id, 2, msSenderIDShape)
	if err != nil {
		return "", "", nil, false, err
	}
	c, err := clientFor(ctx)
	if err != nil {
		return "", "", nil, false, err
	}
	o, err = c.do(ctx, "GET", "messaging", msSenderPath(ids[0], coll, ids[1]), nil, nil)
	if err != nil {
		if isNotFound(err) {
			return ids[0], ids[1], nil, false, nil
		}
		return "", "", nil, false, fmt.Errorf("read %s %s: %w", what, id, err)
	}
	return ids[0], ids[1], o, true, nil
}

func msSenderCreate(ctx context.Context, what, svc, coll string, form url.Values) (obj, error) {
	c, err := clientFor(ctx)
	if err != nil {
		return nil, err
	}
	o, err := c.do(ctx, "POST", "messaging", msSenderPath(svc, coll, ""), nil, form)
	if err != nil {
		return nil, fmt.Errorf("add %s to messaging service %s: %w", what, svc, err)
	}
	return o, nil
}

func msSenderDelete(ctx context.Context, what, svc, coll, sid string) error {
	c, err := clientFor(ctx)
	if err != nil {
		return err
	}
	if _, err := c.do(ctx, "DELETE", "messaging", msSenderPath(svc, coll, sid), nil, nil); err != nil && !isNotFound(err) {
		return fmt.Errorf("remove %s %s from messaging service %s: %w", what, sid, svc, err)
	}
	return nil
}

func msCheckService(c *checker, sid string) {
	c.sid("serviceSid", sid, verifyMessagingSidRE, "Messaging Service (MG followed by 32 hex characters)")
}

func msCheckAlpha(c *checker, v string) {
	if c.unk("alphaSender") {
		return
	}
	if !msAlphaSenderRE.MatchString(v) || msAlphaOnlyDigitsRE.MatchString(v) {
		c.fail("alphaSender", "alphaSender must be 1 to 11 characters of A-Z, a-z, 0-9, space, hyphen, plus, underscore or ampersand, and not only digits")
	}
}

// ------------------------------------------------- MessagingServiceShortCode

// MessagingServiceShortCode adds an owned short code to a Messaging Service's sender pool.
type MessagingServiceShortCode struct{}

// MessagingServiceShortCodeArgs are the inputs of a MessagingServiceShortCode.
type MessagingServiceShortCodeArgs struct {
	ServiceSid   string `pulumi:"serviceSid" provider:"replaceOnChanges"`
	ShortCodeSid string `pulumi:"shortCodeSid" provider:"replaceOnChanges"`
}

func (a *MessagingServiceShortCodeArgs) Annotate(an infer.Annotator) {
	an.Describe(&a.ServiceSid, "SID (MG...) of the Messaging Service. Changing it replaces the resource.")
	an.Describe(&a.ShortCodeSid, "SID (SC...) of the short code of the account to add as a sender (see ShortCode). Changing it replaces the resource.")
}

// MessagingServiceShortCodeState is the persisted state of a MessagingServiceShortCode.
type MessagingServiceShortCodeState struct {
	MessagingServiceShortCodeArgs

	ShortCode    string   `pulumi:"shortCode"`
	CountryCode  string   `pulumi:"countryCode"`
	Capabilities []string `pulumi:"capabilities"`
}

func (s *MessagingServiceShortCodeState) Annotate(a infer.Annotator) {
	a.Describe(&s.ShortCode, "The short code in E.164 format, as Twilio reports it.")
	a.Describe(&s.CountryCode, "The ISO country code of the short code.")
	a.Describe(&s.Capabilities, "Capabilities of the sender, for example SMS and MMS.")
}

func (r *MessagingServiceShortCode) Annotate(a infer.Annotator) {
	a.Describe(&r, "Membership of a short code in a Messaging Service sender pool. The resource ID is <serviceSid>/<shortCodeSid>. Every input replaces the resource; destroying it removes the short code from the pool and keeps the short code.")
}

var _ infer.CustomResource[MessagingServiceShortCodeArgs, MessagingServiceShortCodeState] = (*MessagingServiceShortCode)(nil)
var _ infer.CustomCheck[MessagingServiceShortCodeArgs] = (*MessagingServiceShortCode)(nil)
var _ infer.CustomRead[MessagingServiceShortCodeArgs, MessagingServiceShortCodeState] = (*MessagingServiceShortCode)(nil)
var _ infer.CustomDelete[MessagingServiceShortCodeState] = (*MessagingServiceShortCode)(nil)

func (r *MessagingServiceShortCode) Check(ctx context.Context, req infer.CheckRequest) (infer.CheckResponse[MessagingServiceShortCodeArgs], error) {
	args, failures, err := infer.DefaultCheck[MessagingServiceShortCodeArgs](ctx, req.NewInputs)
	if err != nil || len(failures) > 0 {
		return infer.CheckResponse[MessagingServiceShortCodeArgs]{Inputs: args, Failures: failures}, err
	}
	c := &checker{unk: unknowns(req.NewInputs)}
	msCheckService(c, args.ServiceSid)
	c.sid("shortCodeSid", args.ShortCodeSid, msShortCodeSidRE, "short code (SC followed by 32 hex characters)")
	return infer.CheckResponse[MessagingServiceShortCodeArgs]{Inputs: args, Failures: c.failures}, nil
}

func msShortCodeOutputs(st *MessagingServiceShortCodeState, o obj) {
	st.ShortCode, st.CountryCode, st.Capabilities = o.str("short_code"), o.str("country_code"), capsOf(o)
}

func (r *MessagingServiceShortCode) Create(ctx context.Context, req infer.CreateRequest[MessagingServiceShortCodeArgs]) (infer.CreateResponse[MessagingServiceShortCodeState], error) {
	type resp = infer.CreateResponse[MessagingServiceShortCodeState]
	a := req.Inputs
	st := MessagingServiceShortCodeState{MessagingServiceShortCodeArgs: a, Capabilities: []string{}}
	if req.DryRun {
		return resp{Output: st}, nil
	}
	o, err := msSenderCreate(ctx, "short code "+a.ShortCodeSid, a.ServiceSid, "ShortCodes", url.Values{"ShortCodeSid": {a.ShortCodeSid}})
	if err != nil {
		return resp{}, err
	}
	msShortCodeOutputs(&st, o)
	return resp{ID: a.ServiceSid + "/" + a.ShortCodeSid, Output: st}, nil
}

func (r *MessagingServiceShortCode) Read(ctx context.Context, req infer.ReadRequest[MessagingServiceShortCodeArgs, MessagingServiceShortCodeState]) (infer.ReadResponse[MessagingServiceShortCodeArgs, MessagingServiceShortCodeState], error) {
	type resp = infer.ReadResponse[MessagingServiceShortCodeArgs, MessagingServiceShortCodeState]
	svc, sid, o, found, err := msSenderGet(ctx, "messaging service short code", "ShortCodes", req.ID)
	if err != nil {
		return resp{}, err
	}
	if !found {
		if err := notFoundOnImport("messaging service short code", req.ID, req.Inputs, req.State); err != nil {
			return resp{}, err
		}
		return resp{}, nil
	}
	in := MessagingServiceShortCodeArgs{ServiceSid: svc, ShortCodeSid: sid}
	st := MessagingServiceShortCodeState{MessagingServiceShortCodeArgs: in}
	msShortCodeOutputs(&st, o)
	return resp{ID: req.ID, Inputs: in, State: st}, nil
}

func (r *MessagingServiceShortCode) Delete(ctx context.Context, req infer.DeleteRequest[MessagingServiceShortCodeState]) (infer.DeleteResponse, error) {
	s := req.State
	return infer.DeleteResponse{}, msSenderDelete(ctx, "short code", s.ServiceSid, "ShortCodes", s.ShortCodeSid)
}

// ------------------------------------------------- MessagingServiceChannelSender

// MessagingServiceChannelSender adds a channel sender (for example a WhatsApp sender) to a Messaging
// Service's sender pool.
type MessagingServiceChannelSender struct{}

// MessagingServiceChannelSenderArgs are the inputs of a MessagingServiceChannelSender.
type MessagingServiceChannelSenderArgs struct {
	ServiceSid       string `pulumi:"serviceSid" provider:"replaceOnChanges"`
	ChannelSenderSid string `pulumi:"channelSenderSid" provider:"replaceOnChanges"`
}

func (a *MessagingServiceChannelSenderArgs) Annotate(an infer.Annotator) {
	an.Describe(&a.ServiceSid, "SID (MG...) of the Messaging Service. Changing it replaces the resource.")
	an.Describe(&a.ChannelSenderSid, "SID (XE...) of the channel sender of the account to add, for example a registered WhatsApp sender. Changing it replaces the resource.")
}

// MessagingServiceChannelSenderState is the persisted state of a MessagingServiceChannelSender.
type MessagingServiceChannelSenderState struct {
	MessagingServiceChannelSenderArgs

	Sender      string `pulumi:"sender"`
	SenderType  string `pulumi:"senderType"`
	CountryCode string `pulumi:"countryCode"`
}

func (s *MessagingServiceChannelSenderState) Annotate(a infer.Annotator) {
	a.Describe(&s.Sender, "The sender, for example whatsapp:+14155550100, as Twilio reports it.")
	a.Describe(&s.SenderType, "The kind of sender, for example whatsapp or messenger.")
	a.Describe(&s.CountryCode, "The ISO country code of the sender.")
}

func (r *MessagingServiceChannelSender) Annotate(a infer.Annotator) {
	a.Describe(&r, "Membership of a channel sender in a Messaging Service sender pool. The resource ID is <serviceSid>/<channelSenderSid>. Every input replaces the resource; destroying it removes the sender from the pool and keeps the sender.")
}

var _ infer.CustomResource[MessagingServiceChannelSenderArgs, MessagingServiceChannelSenderState] = (*MessagingServiceChannelSender)(nil)
var _ infer.CustomCheck[MessagingServiceChannelSenderArgs] = (*MessagingServiceChannelSender)(nil)
var _ infer.CustomRead[MessagingServiceChannelSenderArgs, MessagingServiceChannelSenderState] = (*MessagingServiceChannelSender)(nil)
var _ infer.CustomDelete[MessagingServiceChannelSenderState] = (*MessagingServiceChannelSender)(nil)

func (r *MessagingServiceChannelSender) Check(ctx context.Context, req infer.CheckRequest) (infer.CheckResponse[MessagingServiceChannelSenderArgs], error) {
	args, failures, err := infer.DefaultCheck[MessagingServiceChannelSenderArgs](ctx, req.NewInputs)
	if err != nil || len(failures) > 0 {
		return infer.CheckResponse[MessagingServiceChannelSenderArgs]{Inputs: args, Failures: failures}, err
	}
	c := &checker{unk: unknowns(req.NewInputs)}
	msCheckService(c, args.ServiceSid)
	c.sid("channelSenderSid", args.ChannelSenderSid, msChannelSenderSidRE, "channel sender (XE followed by 32 hex characters)")
	return infer.CheckResponse[MessagingServiceChannelSenderArgs]{Inputs: args, Failures: c.failures}, nil
}

func msChannelSenderOutputs(st *MessagingServiceChannelSenderState, o obj) {
	st.Sender, st.SenderType, st.CountryCode = o.str("sender"), o.str("sender_type"), o.str("country_code")
}

func (r *MessagingServiceChannelSender) Create(ctx context.Context, req infer.CreateRequest[MessagingServiceChannelSenderArgs]) (infer.CreateResponse[MessagingServiceChannelSenderState], error) {
	type resp = infer.CreateResponse[MessagingServiceChannelSenderState]
	a := req.Inputs
	st := MessagingServiceChannelSenderState{MessagingServiceChannelSenderArgs: a}
	if req.DryRun {
		return resp{Output: st}, nil
	}
	o, err := msSenderCreate(ctx, "channel sender "+a.ChannelSenderSid, a.ServiceSid, "ChannelSenders", url.Values{"Sid": {a.ChannelSenderSid}})
	if err != nil {
		return resp{}, err
	}
	msChannelSenderOutputs(&st, o)
	return resp{ID: a.ServiceSid + "/" + a.ChannelSenderSid, Output: st}, nil
}

func (r *MessagingServiceChannelSender) Read(ctx context.Context, req infer.ReadRequest[MessagingServiceChannelSenderArgs, MessagingServiceChannelSenderState]) (infer.ReadResponse[MessagingServiceChannelSenderArgs, MessagingServiceChannelSenderState], error) {
	type resp = infer.ReadResponse[MessagingServiceChannelSenderArgs, MessagingServiceChannelSenderState]
	svc, sid, o, found, err := msSenderGet(ctx, "messaging service channel sender", "ChannelSenders", req.ID)
	if err != nil {
		return resp{}, err
	}
	if !found {
		if err := notFoundOnImport("messaging service channel sender", req.ID, req.Inputs, req.State); err != nil {
			return resp{}, err
		}
		return resp{}, nil
	}
	in := MessagingServiceChannelSenderArgs{ServiceSid: svc, ChannelSenderSid: sid}
	st := MessagingServiceChannelSenderState{MessagingServiceChannelSenderArgs: in}
	msChannelSenderOutputs(&st, o)
	return resp{ID: req.ID, Inputs: in, State: st}, nil
}

func (r *MessagingServiceChannelSender) Delete(ctx context.Context, req infer.DeleteRequest[MessagingServiceChannelSenderState]) (infer.DeleteResponse, error) {
	s := req.State
	return infer.DeleteResponse{}, msSenderDelete(ctx, "channel sender", s.ServiceSid, "ChannelSenders", s.ChannelSenderSid)
}

// ------------------------------------------------- MessagingServiceAlphaSender

// MessagingServiceAlphaSender adds an alphanumeric sender ID to a Messaging Service's sender pool.
type MessagingServiceAlphaSender struct{}

// MessagingServiceAlphaSenderArgs are the inputs of a MessagingServiceAlphaSender.
type MessagingServiceAlphaSenderArgs struct {
	ServiceSid  string `pulumi:"serviceSid" provider:"replaceOnChanges"`
	AlphaSender string `pulumi:"alphaSender" provider:"replaceOnChanges"`
}

func (a *MessagingServiceAlphaSenderArgs) Annotate(an infer.Annotator) {
	an.Describe(&a.ServiceSid, "SID (MG...) of the Messaging Service. Changing it replaces the resource.")
	an.Describe(&a.AlphaSender, "The alphanumeric sender ID, up to 11 characters of A-Z, a-z, 0-9, space, hyphen, plus, underscore and ampersand, and not only digits. Changing it replaces the resource.")
}

// MessagingServiceAlphaSenderState is the persisted state of a MessagingServiceAlphaSender.
type MessagingServiceAlphaSenderState struct {
	MessagingServiceAlphaSenderArgs

	Sid          string   `pulumi:"sid"`
	Capabilities []string `pulumi:"capabilities"`
}

func (s *MessagingServiceAlphaSenderState) Annotate(a infer.Annotator) {
	a.Describe(&s.Sid, "The alpha sender SID (AI...).")
	a.Describe(&s.Capabilities, "Capabilities of the sender, as Twilio reports them.")
}

func (r *MessagingServiceAlphaSender) Annotate(a infer.Annotator) {
	a.Describe(&r, "An alphanumeric sender ID in a Messaging Service sender pool. The resource ID is <serviceSid>/<sid>. Twilio has no update call, so every change replaces it.")
}

var _ infer.CustomResource[MessagingServiceAlphaSenderArgs, MessagingServiceAlphaSenderState] = (*MessagingServiceAlphaSender)(nil)
var _ infer.CustomCheck[MessagingServiceAlphaSenderArgs] = (*MessagingServiceAlphaSender)(nil)
var _ infer.CustomRead[MessagingServiceAlphaSenderArgs, MessagingServiceAlphaSenderState] = (*MessagingServiceAlphaSender)(nil)
var _ infer.CustomDelete[MessagingServiceAlphaSenderState] = (*MessagingServiceAlphaSender)(nil)

func (r *MessagingServiceAlphaSender) Check(ctx context.Context, req infer.CheckRequest) (infer.CheckResponse[MessagingServiceAlphaSenderArgs], error) {
	args, failures, err := infer.DefaultCheck[MessagingServiceAlphaSenderArgs](ctx, req.NewInputs)
	if err != nil || len(failures) > 0 {
		return infer.CheckResponse[MessagingServiceAlphaSenderArgs]{Inputs: args, Failures: failures}, err
	}
	c := &checker{unk: unknowns(req.NewInputs)}
	msCheckService(c, args.ServiceSid)
	msCheckAlpha(c, args.AlphaSender)
	return infer.CheckResponse[MessagingServiceAlphaSenderArgs]{Inputs: args, Failures: c.failures}, nil
}

func (r *MessagingServiceAlphaSender) Create(ctx context.Context, req infer.CreateRequest[MessagingServiceAlphaSenderArgs]) (infer.CreateResponse[MessagingServiceAlphaSenderState], error) {
	type resp = infer.CreateResponse[MessagingServiceAlphaSenderState]
	a := req.Inputs
	st := MessagingServiceAlphaSenderState{MessagingServiceAlphaSenderArgs: a, Capabilities: []string{}}
	if req.DryRun {
		return resp{Output: st}, nil
	}
	o, err := msSenderCreate(ctx, "alpha sender "+a.AlphaSender, a.ServiceSid, "AlphaSenders", url.Values{"AlphaSender": {a.AlphaSender}})
	if err != nil {
		return resp{}, err
	}
	st.Sid, st.Capabilities = o.str("sid"), capsOf(o)
	return resp{ID: a.ServiceSid + "/" + st.Sid, Output: st}, nil
}

func (r *MessagingServiceAlphaSender) Read(ctx context.Context, req infer.ReadRequest[MessagingServiceAlphaSenderArgs, MessagingServiceAlphaSenderState]) (infer.ReadResponse[MessagingServiceAlphaSenderArgs, MessagingServiceAlphaSenderState], error) {
	type resp = infer.ReadResponse[MessagingServiceAlphaSenderArgs, MessagingServiceAlphaSenderState]
	svc, sid, o, found, err := msSenderGet(ctx, "messaging service alpha sender", "AlphaSenders", req.ID)
	if err != nil {
		return resp{}, err
	}
	if !found {
		if err := notFoundOnImport("messaging service alpha sender", req.ID, req.Inputs, req.State); err != nil {
			return resp{}, err
		}
		return resp{}, nil
	}
	in := MessagingServiceAlphaSenderArgs{ServiceSid: svc, AlphaSender: o.str("alpha_sender")}
	return resp{ID: req.ID, Inputs: in, State: MessagingServiceAlphaSenderState{MessagingServiceAlphaSenderArgs: in, Sid: sid, Capabilities: capsOf(o)}}, nil
}

func (r *MessagingServiceAlphaSender) Delete(ctx context.Context, req infer.DeleteRequest[MessagingServiceAlphaSenderState]) (infer.DeleteResponse, error) {
	s := req.State
	return infer.DeleteResponse{}, msSenderDelete(ctx, "alpha sender", s.ServiceSid, "AlphaSenders", s.Sid)
}

// ------------------------------------------------- MessagingServiceDestinationAlphaSender

// MessagingServiceDestinationAlphaSender adds an alphanumeric sender ID that is used for one
// destination country (or, with no country, for all of them) to a Messaging Service.
type MessagingServiceDestinationAlphaSender struct{}

// MessagingServiceDestinationAlphaSenderArgs are the inputs of a MessagingServiceDestinationAlphaSender.
type MessagingServiceDestinationAlphaSenderArgs struct {
	ServiceSid     string  `pulumi:"serviceSid" provider:"replaceOnChanges"`
	AlphaSender    string  `pulumi:"alphaSender" provider:"replaceOnChanges"`
	IsoCountryCode *string `pulumi:"isoCountryCode,optional" provider:"replaceOnChanges"`
}

func (a *MessagingServiceDestinationAlphaSenderArgs) Annotate(an infer.Annotator) {
	an.Describe(&a.ServiceSid, "SID (MG...) of the Messaging Service. Changing it replaces the resource.")
	an.Describe(&a.AlphaSender, "The alphanumeric sender ID, up to 11 characters of A-Z, a-z, 0-9, space, hyphen, plus, underscore and ampersand, and not only digits. Changing it replaces the resource.")
	an.Describe(&a.IsoCountryCode, "Two letter ISO 3166-1 country code the sender ID is used for, stored in upper case. Omit it for a default sender ID that is used in every country. Changing it replaces the resource.")
}

// MessagingServiceDestinationAlphaSenderState is the persisted state of a MessagingServiceDestinationAlphaSender.
type MessagingServiceDestinationAlphaSenderState struct {
	MessagingServiceDestinationAlphaSenderArgs

	Sid          string   `pulumi:"sid"`
	Capabilities []string `pulumi:"capabilities"`
}

func (s *MessagingServiceDestinationAlphaSenderState) Annotate(a infer.Annotator) {
	a.Describe(&s.Sid, "The destination alpha sender SID (AI...).")
	a.Describe(&s.Capabilities, "Capabilities of the sender, as Twilio reports them.")
}

func (r *MessagingServiceDestinationAlphaSender) Annotate(a infer.Annotator) {
	a.Describe(&r, "A destination-specific alphanumeric sender ID in a Messaging Service. The resource ID is <serviceSid>/<sid>. Twilio has no update call, so every change replaces it.")
}

var _ infer.CustomResource[MessagingServiceDestinationAlphaSenderArgs, MessagingServiceDestinationAlphaSenderState] = (*MessagingServiceDestinationAlphaSender)(nil)
var _ infer.CustomCheck[MessagingServiceDestinationAlphaSenderArgs] = (*MessagingServiceDestinationAlphaSender)(nil)
var _ infer.CustomRead[MessagingServiceDestinationAlphaSenderArgs, MessagingServiceDestinationAlphaSenderState] = (*MessagingServiceDestinationAlphaSender)(nil)
var _ infer.CustomDelete[MessagingServiceDestinationAlphaSenderState] = (*MessagingServiceDestinationAlphaSender)(nil)

func (r *MessagingServiceDestinationAlphaSender) Check(ctx context.Context, req infer.CheckRequest) (infer.CheckResponse[MessagingServiceDestinationAlphaSenderArgs], error) {
	args, failures, err := infer.DefaultCheck[MessagingServiceDestinationAlphaSenderArgs](ctx, req.NewInputs)
	if err != nil || len(failures) > 0 {
		return infer.CheckResponse[MessagingServiceDestinationAlphaSenderArgs]{Inputs: args, Failures: failures}, err
	}
	c := &checker{unk: unknowns(req.NewInputs)}
	msCheckService(c, args.ServiceSid)
	msCheckAlpha(c, args.AlphaSender)
	if v := args.IsoCountryCode; v != nil && !c.unk("isoCountryCode") {
		if !msCountryRE.MatchString(*v) {
			c.fail("isoCountryCode", "isoCountryCode must be a two letter ISO 3166-1 country code")
		} else {
			// Twilio reports the code in upper case; storing it that way keeps a program that writes it in
			// lower case from replacing the sender on every refresh.
			up := strings.ToUpper(*v)
			args.IsoCountryCode = &up
		}
	}
	return infer.CheckResponse[MessagingServiceDestinationAlphaSenderArgs]{Inputs: args, Failures: c.failures}, nil
}

func (r *MessagingServiceDestinationAlphaSender) Create(ctx context.Context, req infer.CreateRequest[MessagingServiceDestinationAlphaSenderArgs]) (infer.CreateResponse[MessagingServiceDestinationAlphaSenderState], error) {
	type resp = infer.CreateResponse[MessagingServiceDestinationAlphaSenderState]
	a := req.Inputs
	st := MessagingServiceDestinationAlphaSenderState{MessagingServiceDestinationAlphaSenderArgs: a, Capabilities: []string{}}
	if req.DryRun {
		return resp{Output: st}, nil
	}
	form := url.Values{"AlphaSender": {a.AlphaSender}}
	setS(form, "IsoCountryCode", a.IsoCountryCode)
	o, err := msSenderCreate(ctx, "destination alpha sender "+a.AlphaSender, a.ServiceSid, "DestinationAlphaSenders", form)
	if err != nil {
		return resp{}, err
	}
	st.Sid, st.Capabilities = o.str("sid"), capsOf(o)
	return resp{ID: a.ServiceSid + "/" + st.Sid, Output: st}, nil
}

func (r *MessagingServiceDestinationAlphaSender) Read(ctx context.Context, req infer.ReadRequest[MessagingServiceDestinationAlphaSenderArgs, MessagingServiceDestinationAlphaSenderState]) (infer.ReadResponse[MessagingServiceDestinationAlphaSenderArgs, MessagingServiceDestinationAlphaSenderState], error) {
	type resp = infer.ReadResponse[MessagingServiceDestinationAlphaSenderArgs, MessagingServiceDestinationAlphaSenderState]
	svc, sid, o, found, err := msSenderGet(ctx, "messaging service destination alpha sender", "DestinationAlphaSenders", req.ID)
	if err != nil {
		return resp{}, err
	}
	if !found {
		if err := notFoundOnImport("messaging service destination alpha sender", req.ID, req.Inputs, req.State); err != nil {
			return resp{}, err
		}
		return resp{}, nil
	}
	in := MessagingServiceDestinationAlphaSenderArgs{ServiceSid: svc, AlphaSender: o.str("alpha_sender")}
	if cc := strings.ToUpper(strings.TrimSpace(o.str("iso_country_code"))); cc != "" {
		in.IsoCountryCode = &cc
	}
	return resp{ID: req.ID, Inputs: in, State: MessagingServiceDestinationAlphaSenderState{MessagingServiceDestinationAlphaSenderArgs: in, Sid: sid, Capabilities: capsOf(o)}}, nil
}

func (r *MessagingServiceDestinationAlphaSender) Delete(ctx context.Context, req infer.DeleteRequest[MessagingServiceDestinationAlphaSenderState]) (infer.DeleteResponse, error) {
	s := req.State
	return infer.DeleteResponse{}, msSenderDelete(ctx, "destination alpha sender", s.ServiceSid, "DestinationAlphaSenders", s.Sid)
}
