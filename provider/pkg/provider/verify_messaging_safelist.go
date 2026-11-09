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
	"strings"

	"github.com/pulumi/pulumi-go-provider/infer"
)

// VerifyMessagingConfiguration and VerifySafeListNumber (verify.twilio.com v2).

// ------------------------------------------------- VerifyMessagingConfiguration

// VerifyMessagingConfiguration selects the Messaging Service a Verify service sends SMS through for one country.
type VerifyMessagingConfiguration struct{}

// VerifyMessagingConfigurationArgs are the inputs of a VerifyMessagingConfiguration.
type VerifyMessagingConfigurationArgs struct {
	ServiceSid          string `pulumi:"serviceSid" provider:"replaceOnChanges"`
	Country             string `pulumi:"country" provider:"replaceOnChanges"`
	MessagingServiceSid string `pulumi:"messagingServiceSid"`
}

func (a *VerifyMessagingConfigurationArgs) Annotate(an infer.Annotator) {
	an.Describe(&a.ServiceSid, "SID (VA...) of the Verify service. Changing it replaces the resource.")
	an.Describe(&a.Country, "ISO 3166-1 alpha-2 country code the configuration applies to, or all for a global configuration. Changing it replaces the resource.")
	an.Describe(&a.MessagingServiceSid, "SID (MG...) of the Messaging Service (see MessagingService) used to send SMS to the country.")
}

// VerifyMessagingConfigurationState is the persisted state of a VerifyMessagingConfiguration.
type VerifyMessagingConfigurationState struct {
	VerifyMessagingConfigurationArgs

	DateCreated string `pulumi:"dateCreated"`
	DateUpdated string `pulumi:"dateUpdated"`
}

func (s *VerifyMessagingConfigurationState) Annotate(a infer.Annotator) {
	a.Describe(&s.DateCreated, "When the configuration was created (ISO 8601).")
	a.Describe(&s.DateUpdated, "When the configuration was last updated (ISO 8601).")
}

func (r *VerifyMessagingConfiguration) Annotate(a infer.Annotator) {
	a.Describe(&r, "A per-country Messaging Service override of a Verify service. The resource ID is <serviceSid>/<country>.")
}

var _ infer.CustomResource[VerifyMessagingConfigurationArgs, VerifyMessagingConfigurationState] = (*VerifyMessagingConfiguration)(nil)
var _ infer.CustomCheck[VerifyMessagingConfigurationArgs] = (*VerifyMessagingConfiguration)(nil)
var _ infer.CustomRead[VerifyMessagingConfigurationArgs, VerifyMessagingConfigurationState] = (*VerifyMessagingConfiguration)(nil)
var _ infer.CustomUpdate[VerifyMessagingConfigurationArgs, VerifyMessagingConfigurationState] = (*VerifyMessagingConfiguration)(nil)
var _ infer.CustomDelete[VerifyMessagingConfigurationState] = (*VerifyMessagingConfiguration)(nil)

func (r *VerifyMessagingConfiguration) Check(ctx context.Context, req infer.CheckRequest) (infer.CheckResponse[VerifyMessagingConfigurationArgs], error) {
	args, failures, err := infer.DefaultCheck[VerifyMessagingConfigurationArgs](ctx, req.NewInputs)
	if err != nil || len(failures) > 0 {
		return infer.CheckResponse[VerifyMessagingConfigurationArgs]{Inputs: args, Failures: failures}, err
	}
	c := &checker{unk: unknowns(req.NewInputs)}
	c.sid("serviceSid", args.ServiceSid, verifyServiceSidRE, "Verify service (VA followed by 32 hex characters)")
	c.sid("messagingServiceSid", args.MessagingServiceSid, verifyMessagingSidRE, "Messaging Service (MG followed by 32 hex characters)")
	if !c.unk("country") && (args.Country == "" || strings.Contains(args.Country, "/")) {
		c.fail("country", "country must be an ISO 3166-1 alpha-2 code or all")
	}
	return infer.CheckResponse[VerifyMessagingConfigurationArgs]{Inputs: args, Failures: c.failures}, nil
}

// verifyMessagingConfigFound reports whether a messaging configuration response describes the country
// that was asked for. Twilio answers a GET for a country that has no configuration with HTTP 200 and an
// empty or foreign configuration (country "" and a made-up MG SID) instead of 404, so a 200 alone does
// not prove the configuration exists.
func verifyMessagingConfigFound(o obj, country string) bool {
	got := o.str("country")
	return got != "" && strings.EqualFold(got, country)
}

func verifyMessagingConfigOutputs(st *VerifyMessagingConfigurationState, o obj) {
	st.DateCreated, st.DateUpdated = o.str("date_created"), o.str("date_updated")
}

func (r *VerifyMessagingConfiguration) Create(ctx context.Context, req infer.CreateRequest[VerifyMessagingConfigurationArgs]) (infer.CreateResponse[VerifyMessagingConfigurationState], error) {
	a := req.Inputs
	if req.DryRun {
		return infer.CreateResponse[VerifyMessagingConfigurationState]{Output: VerifyMessagingConfigurationState{VerifyMessagingConfigurationArgs: a}}, nil
	}
	c, err := clientFor(ctx)
	if err != nil {
		return infer.CreateResponse[VerifyMessagingConfigurationState]{}, err
	}
	form := url.Values{"Country": {a.Country}, "MessagingServiceSid": {a.MessagingServiceSid}}
	o, err := c.do(ctx, "POST", "verify", verifyChildPath(a.ServiceSid, "MessagingConfigurations", ""), nil, form)
	if err != nil {
		return infer.CreateResponse[VerifyMessagingConfigurationState]{}, fmt.Errorf("create messaging configuration for %s on Verify service %s: %w", a.Country, a.ServiceSid, err)
	}
	st := VerifyMessagingConfigurationState{VerifyMessagingConfigurationArgs: a}
	verifyMessagingConfigOutputs(&st, o)
	return infer.CreateResponse[VerifyMessagingConfigurationState]{ID: a.ServiceSid + "/" + a.Country, Output: st}, nil
}

func (r *VerifyMessagingConfiguration) Read(ctx context.Context, req infer.ReadRequest[VerifyMessagingConfigurationArgs, VerifyMessagingConfigurationState]) (infer.ReadResponse[VerifyMessagingConfigurationArgs, VerifyMessagingConfigurationState], error) {
	type resp = infer.ReadResponse[VerifyMessagingConfigurationArgs, VerifyMessagingConfigurationState]
	ids, err := splitVerifyID(req.ID, 2, "<serviceSid>/<country>")
	if err != nil {
		return resp{}, err
	}
	c, err := clientFor(ctx)
	if err != nil {
		return resp{}, err
	}
	o, err := c.do(ctx, "GET", "verify", verifyChildPath(ids[0], "MessagingConfigurations", ids[1]), nil, nil)
	if err != nil {
		if isNotFound(err) {
			if err := notFoundOnImport("verify messaging configuration", req.ID, req.Inputs, req.State); err != nil {
				return resp{}, err
			}
			return resp{}, nil
		}
		return resp{}, fmt.Errorf("read messaging configuration %s: %w", req.ID, err)
	}
	if !verifyMessagingConfigFound(o, ids[1]) {
		if req.Inputs.ServiceSid == "" {
			return resp{}, fmt.Errorf("messaging configuration %s does not exist: Twilio returned no configuration for country %q", req.ID, ids[1])
		}
		if err := notFoundOnImport("verify messaging configuration", req.ID, req.Inputs, req.State); err != nil {
			return resp{}, err
		}
		return resp{}, nil
	}
	in := req.Inputs
	if in.ServiceSid == "" {
		in = VerifyMessagingConfigurationArgs{ServiceSid: ids[0], Country: ids[1]}
	}
	if v := o.str("messaging_service_sid"); v != "" {
		in.MessagingServiceSid = v
	}
	st := VerifyMessagingConfigurationState{VerifyMessagingConfigurationArgs: in}
	verifyMessagingConfigOutputs(&st, o)
	return resp{ID: req.ID, Inputs: in, State: st}, nil
}

func (r *VerifyMessagingConfiguration) Update(ctx context.Context, req infer.UpdateRequest[VerifyMessagingConfigurationArgs, VerifyMessagingConfigurationState]) (infer.UpdateResponse[VerifyMessagingConfigurationState], error) {
	a := req.Inputs
	st := req.State
	st.VerifyMessagingConfigurationArgs = a
	if req.DryRun {
		return infer.UpdateResponse[VerifyMessagingConfigurationState]{Output: st}, nil
	}
	ids, err := splitVerifyID(req.ID, 2, "<serviceSid>/<country>")
	if err != nil {
		return infer.UpdateResponse[VerifyMessagingConfigurationState]{}, err
	}
	c, err := clientFor(ctx)
	if err != nil {
		return infer.UpdateResponse[VerifyMessagingConfigurationState]{}, err
	}
	o, err := c.do(ctx, "POST", "verify", verifyChildPath(ids[0], "MessagingConfigurations", ids[1]), nil, url.Values{"MessagingServiceSid": {a.MessagingServiceSid}})
	if err != nil {
		return infer.UpdateResponse[VerifyMessagingConfigurationState]{}, fmt.Errorf("update messaging configuration %s: %w", req.ID, err)
	}
	verifyMessagingConfigOutputs(&st, o)
	return infer.UpdateResponse[VerifyMessagingConfigurationState]{Output: st}, nil
}

func (r *VerifyMessagingConfiguration) Delete(ctx context.Context, req infer.DeleteRequest[VerifyMessagingConfigurationState]) (infer.DeleteResponse, error) {
	ids, err := splitVerifyID(req.ID, 2, "<serviceSid>/<country>")
	if err != nil {
		return infer.DeleteResponse{}, err
	}
	c, err := clientFor(ctx)
	if err != nil {
		return infer.DeleteResponse{}, err
	}
	if _, err := c.do(ctx, "DELETE", "verify", verifyChildPath(ids[0], "MessagingConfigurations", ids[1]), nil, nil); err != nil && !isNotFound(err) {
		return infer.DeleteResponse{}, fmt.Errorf("delete messaging configuration %s: %w", req.ID, err)
	}
	return infer.DeleteResponse{}, nil
}

// ------------------------------------------------------------ VerifySafeListNumber

// VerifySafeListNumber puts a phone number on the Verify SafeList, which exempts it from Twilio's
// fraud guard blocks. The SafeList belongs to the account, not to a Verify service.
type VerifySafeListNumber struct{}

// VerifySafeListNumberArgs are the inputs of a VerifySafeListNumber.
type VerifySafeListNumberArgs struct {
	PhoneNumber string `pulumi:"phoneNumber" provider:"replaceOnChanges"`
}

func (a *VerifySafeListNumberArgs) Annotate(an infer.Annotator) {
	an.Describe(&a.PhoneNumber, "The phone number to add to the SafeList, in E.164 format (+14155550100). Changing it replaces the resource.")
}

// VerifySafeListNumberState is the persisted state of a VerifySafeListNumber.
type VerifySafeListNumberState struct {
	VerifySafeListNumberArgs

	// Sid is the SafeList entry SID (GN...).
	Sid string `pulumi:"sid"`
}

func (s *VerifySafeListNumberState) Annotate(a infer.Annotator) {
	a.Describe(&s.Sid, "The SafeList entry SID, as Twilio reports it.")
}

func (r *VerifySafeListNumber) Annotate(a infer.Annotator) {
	a.Describe(&r, "A phone number on the account's Verify SafeList. The resource ID is the phone number in E.164 format. Twilio has no update call, so changing the number replaces the entry.")
}

var _ infer.CustomResource[VerifySafeListNumberArgs, VerifySafeListNumberState] = (*VerifySafeListNumber)(nil)
var _ infer.CustomCheck[VerifySafeListNumberArgs] = (*VerifySafeListNumber)(nil)
var _ infer.CustomRead[VerifySafeListNumberArgs, VerifySafeListNumberState] = (*VerifySafeListNumber)(nil)
var _ infer.CustomDelete[VerifySafeListNumberState] = (*VerifySafeListNumber)(nil)

func (r *VerifySafeListNumber) Check(ctx context.Context, req infer.CheckRequest) (infer.CheckResponse[VerifySafeListNumberArgs], error) {
	args, failures, err := infer.DefaultCheck[VerifySafeListNumberArgs](ctx, req.NewInputs)
	if err != nil || len(failures) > 0 {
		return infer.CheckResponse[VerifySafeListNumberArgs]{Inputs: args, Failures: failures}, err
	}
	c := &checker{unk: unknowns(req.NewInputs)}
	if !c.unk("phoneNumber") && !e164RE.MatchString(args.PhoneNumber) {
		c.fail("phoneNumber", "phoneNumber must be in E.164 format, for example +14155550100")
	}
	return infer.CheckResponse[VerifySafeListNumberArgs]{Inputs: args, Failures: c.failures}, nil
}

func (r *VerifySafeListNumber) Create(ctx context.Context, req infer.CreateRequest[VerifySafeListNumberArgs]) (infer.CreateResponse[VerifySafeListNumberState], error) {
	a := req.Inputs
	if req.DryRun {
		return infer.CreateResponse[VerifySafeListNumberState]{Output: VerifySafeListNumberState{VerifySafeListNumberArgs: a}}, nil
	}
	c, err := clientFor(ctx)
	if err != nil {
		return infer.CreateResponse[VerifySafeListNumberState]{}, err
	}
	o, err := c.do(ctx, "POST", "verify", safeListPath(""), nil, url.Values{"PhoneNumber": {a.PhoneNumber}})
	if err != nil {
		return infer.CreateResponse[VerifySafeListNumberState]{}, fmt.Errorf("add %s to the Verify SafeList: %w", a.PhoneNumber, err)
	}
	return infer.CreateResponse[VerifySafeListNumberState]{ID: a.PhoneNumber,
		Output: VerifySafeListNumberState{VerifySafeListNumberArgs: a, Sid: o.str("sid")}}, nil
}

func (r *VerifySafeListNumber) Read(ctx context.Context, req infer.ReadRequest[VerifySafeListNumberArgs, VerifySafeListNumberState]) (infer.ReadResponse[VerifySafeListNumberArgs, VerifySafeListNumberState], error) {
	type resp = infer.ReadResponse[VerifySafeListNumberArgs, VerifySafeListNumberState]
	c, err := clientFor(ctx)
	if err != nil {
		return resp{}, err
	}
	o, err := c.do(ctx, "GET", "verify", safeListPath(req.ID), nil, nil)
	if err != nil {
		if isNotFound(err) {
			if err := notFoundOnImport("verify safe list number", req.ID, req.Inputs, req.State); err != nil {
				return resp{}, err
			}
			return resp{}, nil
		}
		return resp{}, fmt.Errorf("read SafeList number %s: %w", req.ID, err)
	}
	in := req.Inputs
	if in.PhoneNumber == "" {
		in.PhoneNumber = req.ID
	}
	return resp{ID: req.ID, Inputs: in, State: VerifySafeListNumberState{VerifySafeListNumberArgs: in, Sid: o.str("sid")}}, nil
}

func (r *VerifySafeListNumber) Delete(ctx context.Context, req infer.DeleteRequest[VerifySafeListNumberState]) (infer.DeleteResponse, error) {
	c, err := clientFor(ctx)
	if err != nil {
		return infer.DeleteResponse{}, err
	}
	if _, err := c.do(ctx, "DELETE", "verify", safeListPath(req.ID), nil, nil); err != nil && !isNotFound(err) {
		return infer.DeleteResponse{}, fmt.Errorf("remove %s from the Verify SafeList: %w", req.ID, err)
	}
	return infer.DeleteResponse{}, nil
}
