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
	"net/http"

	"github.com/pulumi/pulumi-go-provider/infer"
)

// The account-wide default Voice v2 recording configuration lives at
// /v2/AccountDefaultConfiguration/Recording on voice.twilio.com. There is at most one per
// account: POST creates it (409 when it already exists), GET reads it, PUT replaces it, DELETE
// removes it. Unlike a VoiceRecordingConfiguration it is not referenced by ID; the configuration is
// embedded in it and applies to every call that has no configuration of its own.
//
// The shapes follow twilio-oai spec/json/twilio_voice_v2.json (spec version 0.0.1, read
// 2026-11-10) and have not been checked against a live account. The spec lists a 412 "ETag
// mismatch" on PUT; no request header or response ETag is documented, so the provider sends none.

const accountDefaultRecordingPath = "/v2/AccountDefaultConfiguration/Recording"

func isConflict(err error) bool {
	ae, ok := err.(*apiError)
	return ok && ae.Status == http.StatusConflict
}

// VoiceAccountDefaultRecordingConfiguration manages the account-wide default Voice v2 recording
// configuration: how calls without a recording configuration of their own are recorded.
type VoiceAccountDefaultRecordingConfiguration struct{}

// VoiceAccountDefaultRecordingConfigurationArgs are the inputs. They are the inputs of a
// VoiceRecordingConfiguration without its unique name; every input updates in place and the whole
// configuration is sent on each update.
type VoiceAccountDefaultRecordingConfigurationArgs struct {
	Description                       *string                     `pulumi:"description,optional"`
	CompositionPolicy                 *RecordingCompositionPolicy `pulumi:"compositionPolicy,optional"`
	CallRecordingStatusCallback       *RecordingStatusCallback    `pulumi:"callRecordingStatusCallback,optional"`
	ConferenceRecordingStatusCallback *RecordingStatusCallback    `pulumi:"conferenceRecordingStatusCallback,optional"`
	Features                          []RecordingFeature          `pulumi:"features,optional"`
}

func (a *VoiceAccountDefaultRecordingConfigurationArgs) Annotate(an infer.Annotator) {
	an.Describe(&a.Description, "Free-text description of the default configuration.")
	an.Describe(&a.CompositionPolicy, "How recordings are composed: channels, silence trimming and track.")
	an.Describe(&a.CallRecordingStatusCallback, "Where call recording status events are reported.")
	an.Describe(&a.ConferenceRecordingStatusCallback, "Where conference recording status events are reported.")
	an.Describe(&a.Features, "Features applied to the recording (at most 3), for example a transcription configuration.")
}

// VoiceAccountDefaultRecordingConfigurationState is the persisted state.
type VoiceAccountDefaultRecordingConfigurationState struct {
	VoiceAccountDefaultRecordingConfigurationArgs

	// AccountSid is the account the default applies to and the resource ID.
	AccountSid string `pulumi:"accountSid"`
	// DateCreated and DateUpdated are ISO 8601 timestamps.
	DateCreated string `pulumi:"dateCreated"`
	DateUpdated string `pulumi:"dateUpdated"`
}

func (s *VoiceAccountDefaultRecordingConfigurationState) Annotate(a infer.Annotator) {
	a.Describe(&s.AccountSid, "SID of the account the default applies to; also the resource ID.")
	a.Describe(&s.DateCreated, "When the default was created (ISO 8601).")
	a.Describe(&s.DateUpdated, "When the default was last updated (ISO 8601).")
}

func (r *VoiceAccountDefaultRecordingConfiguration) Annotate(a infer.Annotator) {
	a.Describe(&r, "The account-wide default Twilio Voice v2 recording configuration: how calls that have no recording "+
		"configuration of their own are recorded. There is at most one per account, so create adopts an existing one "+
		"(it is replaced by this resource's inputs) and destroy removes the account default. Import with the account SID. "+
		"The Voice v2 API is a new, versioned 0.0.1 product, so expect it to change.")
}

var _ infer.CustomResource[VoiceAccountDefaultRecordingConfigurationArgs, VoiceAccountDefaultRecordingConfigurationState] = (*VoiceAccountDefaultRecordingConfiguration)(nil)
var _ infer.CustomCheck[VoiceAccountDefaultRecordingConfigurationArgs] = (*VoiceAccountDefaultRecordingConfiguration)(nil)
var _ infer.CustomRead[VoiceAccountDefaultRecordingConfigurationArgs, VoiceAccountDefaultRecordingConfigurationState] = (*VoiceAccountDefaultRecordingConfiguration)(nil)
var _ infer.CustomUpdate[VoiceAccountDefaultRecordingConfigurationArgs, VoiceAccountDefaultRecordingConfigurationState] = (*VoiceAccountDefaultRecordingConfiguration)(nil)
var _ infer.CustomDelete[VoiceAccountDefaultRecordingConfigurationState] = (*VoiceAccountDefaultRecordingConfiguration)(nil)

// asRecording maps the inputs onto those of a VoiceRecordingConfiguration, whose body, response
// handling and checks the default shares.
func (a VoiceAccountDefaultRecordingConfigurationArgs) asRecording() VoiceRecordingConfigurationArgs {
	return VoiceRecordingConfigurationArgs{Description: a.Description, CompositionPolicy: a.CompositionPolicy,
		CallRecordingStatusCallback: a.CallRecordingStatusCallback, ConferenceRecordingStatusCallback: a.ConferenceRecordingStatusCallback,
		Features: a.Features}
}

func accountDefaultFromRecording(a VoiceRecordingConfigurationArgs) VoiceAccountDefaultRecordingConfigurationArgs {
	return VoiceAccountDefaultRecordingConfigurationArgs{Description: a.Description, CompositionPolicy: a.CompositionPolicy,
		CallRecordingStatusCallback: a.CallRecordingStatusCallback, ConferenceRecordingStatusCallback: a.ConferenceRecordingStatusCallback,
		Features: a.Features}
}

// accountDefaultBody is the request body: the description and the embedded configuration. The
// default has no unique name.
func (a VoiceAccountDefaultRecordingConfigurationArgs) body() recordingConfigurationBody {
	b := a.asRecording().body()
	b.UniqueName = nil
	return b
}

// decodeAccountDefault decodes the response. The default has no id of its own, so the account
// SID stands in for it.
func decodeAccountDefault(o obj) (recordingConfigurationResponse, error) {
	var out recordingConfigurationResponse
	if err := decodeJSONObject(o, &out); err != nil {
		return out, fmt.Errorf("decode account default recording configuration: %w", err)
	}
	out.ID = out.AccountSid
	if out.ID == "" {
		return out, fmt.Errorf("decode account default recording configuration: no account_sid in the response")
	}
	return out, nil
}

func accountDefaultStateFrom(in VoiceAccountDefaultRecordingConfigurationArgs, resp recordingConfigurationResponse) VoiceAccountDefaultRecordingConfigurationState {
	return VoiceAccountDefaultRecordingConfigurationState{VoiceAccountDefaultRecordingConfigurationArgs: in,
		AccountSid: resp.AccountSid, DateCreated: resp.DateCreated, DateUpdated: resp.DateUpdated}
}

func (r *VoiceAccountDefaultRecordingConfiguration) Check(ctx context.Context, req infer.CheckRequest) (infer.CheckResponse[VoiceAccountDefaultRecordingConfigurationArgs], error) {
	// The checks are those of a recording configuration: the inputs are a subset of its inputs.
	chk, err := (&VoiceRecordingConfiguration{}).Check(ctx, req)
	if err != nil {
		return infer.CheckResponse[VoiceAccountDefaultRecordingConfigurationArgs]{}, err
	}
	return infer.CheckResponse[VoiceAccountDefaultRecordingConfigurationArgs]{
		Inputs: accountDefaultFromRecording(chk.Inputs), Failures: chk.Failures}, nil
}

func (r *VoiceAccountDefaultRecordingConfiguration) Create(ctx context.Context, req infer.CreateRequest[VoiceAccountDefaultRecordingConfigurationArgs]) (infer.CreateResponse[VoiceAccountDefaultRecordingConfigurationState], error) {
	type resp = infer.CreateResponse[VoiceAccountDefaultRecordingConfigurationState]
	a := req.Inputs
	if req.DryRun {
		return resp{Output: VoiceAccountDefaultRecordingConfigurationState{VoiceAccountDefaultRecordingConfigurationArgs: a}}, nil
	}
	c, err := clientFor(ctx)
	if err != nil {
		return resp{}, err
	}
	o, err := c.doJSON(ctx, "POST", voiceProduct, accountDefaultRecordingPath, nil, a.body())
	if isConflict(err) {
		// One already exists: adopt it by replacing it with the program's configuration.
		o, err = c.doJSON(ctx, "PUT", voiceProduct, accountDefaultRecordingPath, nil, a.body())
	}
	if err != nil {
		return resp{}, fmt.Errorf("create account default recording configuration: %w", err)
	}
	got, err := decodeAccountDefault(o)
	if err != nil {
		return resp{}, err
	}
	return resp{ID: got.ID, Output: accountDefaultStateFrom(a, got)}, nil
}

func (r *VoiceAccountDefaultRecordingConfiguration) Read(ctx context.Context, req infer.ReadRequest[VoiceAccountDefaultRecordingConfigurationArgs, VoiceAccountDefaultRecordingConfigurationState]) (infer.ReadResponse[VoiceAccountDefaultRecordingConfigurationArgs, VoiceAccountDefaultRecordingConfigurationState], error) {
	type resp = infer.ReadResponse[VoiceAccountDefaultRecordingConfigurationArgs, VoiceAccountDefaultRecordingConfigurationState]
	c, err := clientFor(ctx)
	if err != nil {
		return resp{}, err
	}
	if c.accountSid != "" && req.ID != c.accountSid {
		return resp{}, fmt.Errorf("invalid VoiceAccountDefaultRecordingConfiguration ID %q: it is the SID of the account the provider authenticates as (%q)", req.ID, c.accountSid)
	}
	o, err := c.doJSON(ctx, "GET", voiceProduct, accountDefaultRecordingPath, nil, nil)
	if err != nil {
		if isNotFound(err) {
			if err := notFoundOnImport("account default recording configuration", req.ID, req.Inputs, req.State); err != nil {
				return resp{}, err
			}
			return resp{}, nil
		}
		return resp{}, fmt.Errorf("read account default recording configuration: %w", err)
	}
	got, err := decodeAccountDefault(o)
	if err != nil {
		return resp{}, err
	}
	in := accountDefaultFromRecording(adoptRecording(req.Inputs.asRecording(), got))
	return resp{ID: req.ID, Inputs: in, State: accountDefaultStateFrom(in, got)}, nil
}

func (r *VoiceAccountDefaultRecordingConfiguration) Update(ctx context.Context, req infer.UpdateRequest[VoiceAccountDefaultRecordingConfigurationArgs, VoiceAccountDefaultRecordingConfigurationState]) (infer.UpdateResponse[VoiceAccountDefaultRecordingConfigurationState], error) {
	type resp = infer.UpdateResponse[VoiceAccountDefaultRecordingConfigurationState]
	a := req.Inputs
	st := req.State
	st.VoiceAccountDefaultRecordingConfigurationArgs = a
	if req.DryRun {
		return resp{Output: st}, nil
	}
	c, err := clientFor(ctx)
	if err != nil {
		return resp{}, err
	}
	o, err := c.doJSON(ctx, "PUT", voiceProduct, accountDefaultRecordingPath, nil, a.body())
	if err != nil {
		return resp{}, fmt.Errorf("update account default recording configuration: %w", err)
	}
	got, err := decodeAccountDefault(o)
	if err != nil {
		return resp{}, err
	}
	return resp{Output: accountDefaultStateFrom(a, got)}, nil
}

func (r *VoiceAccountDefaultRecordingConfiguration) Delete(ctx context.Context, req infer.DeleteRequest[VoiceAccountDefaultRecordingConfigurationState]) (infer.DeleteResponse, error) {
	c, err := clientFor(ctx)
	if err != nil {
		return infer.DeleteResponse{}, err
	}
	if _, err := c.doJSON(ctx, "DELETE", voiceProduct, accountDefaultRecordingPath, nil, nil); err != nil && !isNotFound(err) {
		return infer.DeleteResponse{}, fmt.Errorf("delete account default recording configuration: %w", err)
	}
	return infer.DeleteResponse{}, nil
}

// GetVoiceAccountDefaultRecordingConfiguration reads the account-wide default recording configuration.
type GetVoiceAccountDefaultRecordingConfiguration struct{}

// GetVoiceAccountDefaultRecordingConfigurationArgs has no arguments.
type GetVoiceAccountDefaultRecordingConfigurationArgs struct{}

// GetVoiceAccountDefaultRecordingConfigurationResult is the account default.
type GetVoiceAccountDefaultRecordingConfigurationResult struct {
	AccountSid                        string                      `pulumi:"accountSid"`
	Description                       string                      `pulumi:"description"`
	CompositionPolicy                 *RecordingCompositionPolicy `pulumi:"compositionPolicy,optional"`
	CallRecordingStatusCallback       *RecordingStatusCallback    `pulumi:"callRecordingStatusCallback,optional"`
	ConferenceRecordingStatusCallback *RecordingStatusCallback    `pulumi:"conferenceRecordingStatusCallback,optional"`
	Features                          []RecordingFeature          `pulumi:"features"`
	DateCreated                       string                      `pulumi:"dateCreated"`
	DateUpdated                       string                      `pulumi:"dateUpdated"`
}

func (a *GetVoiceAccountDefaultRecordingConfigurationResult) Annotate(an infer.Annotator) {
	an.Describe(&a.AccountSid, "SID of the account the default applies to.")
	an.Describe(&a.Description, "The description; empty when none is set.")
	an.Describe(&a.CompositionPolicy, "How recordings are composed; unset when Twilio reports none.")
	an.Describe(&a.CallRecordingStatusCallback, "Where call recording status events are reported; unset when none.")
	an.Describe(&a.ConferenceRecordingStatusCallback, "Where conference recording status events are reported; unset when none.")
	an.Describe(&a.Features, "Features applied to the recording; empty when none.")
	an.Describe(&a.DateCreated, "When the default was created (ISO 8601).")
	an.Describe(&a.DateUpdated, "When the default was last updated (ISO 8601).")
}

func (f *GetVoiceAccountDefaultRecordingConfiguration) Annotate(a infer.Annotator) {
	a.Describe(&f, "Read the account-wide default Twilio Voice v2 recording configuration. Fails when the account has none.")
}

func (f *GetVoiceAccountDefaultRecordingConfiguration) Invoke(ctx context.Context, _ infer.FunctionRequest[GetVoiceAccountDefaultRecordingConfigurationArgs]) (infer.FunctionResponse[GetVoiceAccountDefaultRecordingConfigurationResult], error) {
	type resp = infer.FunctionResponse[GetVoiceAccountDefaultRecordingConfigurationResult]
	c, err := clientFor(ctx)
	if err != nil {
		return resp{}, err
	}
	o, err := c.doJSON(ctx, "GET", voiceProduct, accountDefaultRecordingPath, nil, nil)
	if err != nil {
		if isNotFound(err) {
			return resp{}, fmt.Errorf("getVoiceAccountDefaultRecordingConfiguration: the account has no default recording configuration")
		}
		return resp{}, fmt.Errorf("getVoiceAccountDefaultRecordingConfiguration: %w", err)
	}
	got, err := decodeAccountDefault(o)
	if err != nil {
		return resp{}, err
	}
	out := GetVoiceAccountDefaultRecordingConfigurationResult{AccountSid: got.AccountSid, DateCreated: got.DateCreated,
		DateUpdated: got.DateUpdated, Features: got.Configuration.Features}
	if got.Description != nil {
		out.Description = *got.Description
	}
	if out.Features == nil {
		out.Features = []RecordingFeature{}
	}
	if !got.Configuration.CompositionPolicy.empty() {
		out.CompositionPolicy = got.Configuration.CompositionPolicy
	}
	if !got.Configuration.CallRecordingStatusCallback.empty() {
		out.CallRecordingStatusCallback = got.Configuration.CallRecordingStatusCallback
	}
	if !got.Configuration.ConferenceRecordingStatusCallback.empty() {
		out.ConferenceRecordingStatusCallback = got.Configuration.ConferenceRecordingStatusCallback
	}
	return resp{Output: out}, nil
}
