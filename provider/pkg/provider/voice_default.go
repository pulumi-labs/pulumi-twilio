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

	"github.com/pulumi/pulumi-go-provider/infer"
)

// The default Voice v2 configuration of a type is selected with POST
// /v2/Configurations/{Type}/Default and a configuration_id (204, no body) and read with GET on the
// same path, which answers the selected configuration (404 when none is set). The spec documents
// no way to unset it. Shapes follow twilio-oai spec/json/twilio_voice_v2.json (spec version 0.0.1,
// read 2026-11-11) and have not been checked against a live account.

type voiceDefault struct {
	configurationID, uniqueName, accountSid, dateUpdated string
}

func readVoiceDefault(ctx context.Context, c *Client, path, what string) (voiceDefault, error) {
	o, err := c.doJSON(ctx, "GET", voiceProduct, path, nil, nil)
	if err != nil {
		if isNotFound(err) {
			return voiceDefault{}, err
		}
		return voiceDefault{}, fmt.Errorf("read default %s configuration: %w", what, err)
	}
	var out struct {
		ID         string  `json:"id"`
		AccountSid string  `json:"account_sid"`
		UniqueName *string `json:"unique_name"`
		Updated    string  `json:"date_updated"`
	}
	if err := decodeJSONObject(o, &out); err != nil {
		return voiceDefault{}, fmt.Errorf("decode default %s configuration: %w", what, err)
	}
	if out.ID == "" {
		return voiceDefault{}, fmt.Errorf("decode default %s configuration: no id in the response", what)
	}
	if out.AccountSid == "" {
		out.AccountSid = c.accountSid
	}
	if out.AccountSid == "" {
		return voiceDefault{}, fmt.Errorf("decode default %s configuration: no account_sid in the response", what)
	}
	return voiceDefault{configurationID: out.ID, uniqueName: deref(out.UniqueName), accountSid: out.AccountSid, dateUpdated: out.Updated}, nil
}

// setVoiceDefault selects the configuration and reads the default back, which confirms the
// selection and supplies the account SID the resource is identified by.
func setVoiceDefault(ctx context.Context, c *Client, path, what, id string) (voiceDefault, error) {
	if id == "" {
		return voiceDefault{}, fmt.Errorf("set default %s configuration: configurationId must not be empty", what)
	}
	if _, err := c.doJSON(ctx, "POST", voiceProduct, path, nil, map[string]string{"configuration_id": id}); err != nil {
		return voiceDefault{}, fmt.Errorf("set default %s configuration: %w", what, err)
	}
	got, err := readVoiceDefault(ctx, c, path, what)
	if err != nil {
		return voiceDefault{}, err
	}
	if got.configurationID != id {
		return voiceDefault{}, fmt.Errorf("set default %s configuration: Twilio reports %q as the default after selecting %q", what, got.configurationID, id)
	}
	return got, nil
}

// VoiceDefaultRecordingConfiguration selects which existing Voice v2 recording configuration is the account default.
type VoiceDefaultRecordingConfiguration struct{}

// VoiceDefaultRecordingConfigurationArgs are the inputs.
type VoiceDefaultRecordingConfigurationArgs struct {
	ConfigurationId string `pulumi:"configurationId"`
}

func (a *VoiceDefaultRecordingConfigurationArgs) Annotate(an infer.Annotator) {
	an.Describe(&a.ConfigurationId, "ID of the existing recording configuration to make the default (a VoiceRecordingConfiguration's configurationId). Changing it selects another configuration.")
}

// VoiceDefaultRecordingConfigurationState is the persisted state.
type VoiceDefaultRecordingConfigurationState struct {
	VoiceDefaultRecordingConfigurationArgs

	// AccountSid is the account the default applies to and the resource ID.
	AccountSid string `pulumi:"accountSid"`
	// DateUpdated is when the selected configuration was last updated (ISO 8601).
	DateUpdated string `pulumi:"dateUpdated"`
}

func (s *VoiceDefaultRecordingConfigurationState) Annotate(a infer.Annotator) {
	a.Describe(&s.AccountSid, "SID of the account the default applies to; also the resource ID.")
	a.Describe(&s.DateUpdated, "When the selected configuration was last updated (ISO 8601).")
}

func (r *VoiceDefaultRecordingConfiguration) Annotate(a infer.Annotator) {
	a.Describe(&r, "Marks an existing Twilio Voice v2 recording configuration as the account default (POST /v2/Configurations/Recording/Default). "+
		"There is at most one default per account. Twilio documents no way to unset it, so destroy only forgets the selection: "+
		"the configuration stays the default until another is chosen. Import with the account SID. "+
		"The Voice v2 API is a new, versioned 0.0.1 product, so expect it to change.")
}

var _ infer.CustomResource[VoiceDefaultRecordingConfigurationArgs, VoiceDefaultRecordingConfigurationState] = (*VoiceDefaultRecordingConfiguration)(nil)
var _ infer.CustomRead[VoiceDefaultRecordingConfigurationArgs, VoiceDefaultRecordingConfigurationState] = (*VoiceDefaultRecordingConfiguration)(nil)
var _ infer.CustomUpdate[VoiceDefaultRecordingConfigurationArgs, VoiceDefaultRecordingConfigurationState] = (*VoiceDefaultRecordingConfiguration)(nil)
var _ infer.CustomDelete[VoiceDefaultRecordingConfigurationState] = (*VoiceDefaultRecordingConfiguration)(nil)

const defaultRecordingConfigPath = "/v2/Configurations/Recording/Default"

func (r *VoiceDefaultRecordingConfiguration) Create(ctx context.Context, req infer.CreateRequest[VoiceDefaultRecordingConfigurationArgs]) (infer.CreateResponse[VoiceDefaultRecordingConfigurationState], error) {
	type resp = infer.CreateResponse[VoiceDefaultRecordingConfigurationState]
	a := req.Inputs
	if req.DryRun {
		return resp{Output: VoiceDefaultRecordingConfigurationState{VoiceDefaultRecordingConfigurationArgs: a}}, nil
	}
	c, err := clientFor(ctx)
	if err != nil {
		return resp{}, err
	}
	st, err := setVoiceDefault(ctx, c, defaultRecordingConfigPath, "recording", a.ConfigurationId)
	if err != nil {
		return resp{}, err
	}
	return resp{ID: st.accountSid, Output: VoiceDefaultRecordingConfigurationState{VoiceDefaultRecordingConfigurationArgs: a, AccountSid: st.accountSid, DateUpdated: st.dateUpdated}}, nil
}

func (r *VoiceDefaultRecordingConfiguration) Read(ctx context.Context, req infer.ReadRequest[VoiceDefaultRecordingConfigurationArgs, VoiceDefaultRecordingConfigurationState]) (infer.ReadResponse[VoiceDefaultRecordingConfigurationArgs, VoiceDefaultRecordingConfigurationState], error) {
	type resp = infer.ReadResponse[VoiceDefaultRecordingConfigurationArgs, VoiceDefaultRecordingConfigurationState]
	c, err := clientFor(ctx)
	if err != nil {
		return resp{}, err
	}
	if c.accountSid != "" && req.ID != c.accountSid {
		return resp{}, fmt.Errorf("invalid VoiceDefaultRecordingConfiguration ID %q: it is the SID of the account the provider authenticates as (%q)", req.ID, c.accountSid)
	}
	got, err := readVoiceDefault(ctx, c, defaultRecordingConfigPath, "recording")
	if err != nil {
		if isNotFound(err) {
			if err := notFoundOnImport("default recording configuration", req.ID, req.Inputs, req.State); err != nil {
				return resp{}, err
			}
			return resp{}, nil
		}
		return resp{}, err
	}
	in := VoiceDefaultRecordingConfigurationArgs{ConfigurationId: got.configurationID}
	return resp{ID: req.ID, Inputs: in, State: VoiceDefaultRecordingConfigurationState{VoiceDefaultRecordingConfigurationArgs: in, AccountSid: got.accountSid, DateUpdated: got.dateUpdated}}, nil
}

func (r *VoiceDefaultRecordingConfiguration) Update(ctx context.Context, req infer.UpdateRequest[VoiceDefaultRecordingConfigurationArgs, VoiceDefaultRecordingConfigurationState]) (infer.UpdateResponse[VoiceDefaultRecordingConfigurationState], error) {
	type resp = infer.UpdateResponse[VoiceDefaultRecordingConfigurationState]
	st := req.State
	st.VoiceDefaultRecordingConfigurationArgs = req.Inputs
	if req.DryRun {
		return resp{Output: st}, nil
	}
	c, err := clientFor(ctx)
	if err != nil {
		return resp{}, err
	}
	got, err := setVoiceDefault(ctx, c, defaultRecordingConfigPath, "recording", req.Inputs.ConfigurationId)
	if err != nil {
		return resp{}, err
	}
	st.AccountSid, st.DateUpdated = got.accountSid, got.dateUpdated
	return resp{Output: st}, nil
}

// Delete only forgets the selection: Twilio documents no call that unsets the default.
func (r *VoiceDefaultRecordingConfiguration) Delete(ctx context.Context, req infer.DeleteRequest[VoiceDefaultRecordingConfigurationState]) (infer.DeleteResponse, error) {
	return infer.DeleteResponse{}, nil
}

// GetVoiceDefaultRecordingConfiguration reads which recording configuration is the account default.
type GetVoiceDefaultRecordingConfiguration struct{}

// GetVoiceDefaultRecordingConfigurationArgs has no arguments.
type GetVoiceDefaultRecordingConfigurationArgs struct{}

// GetVoiceDefaultRecordingConfigurationResult is the account default.
type GetVoiceDefaultRecordingConfigurationResult struct {
	ConfigurationId string `pulumi:"configurationId"`
	UniqueName      string `pulumi:"uniqueName"`
	AccountSid      string `pulumi:"accountSid"`
	DateUpdated     string `pulumi:"dateUpdated"`
}

func (a *GetVoiceDefaultRecordingConfigurationResult) Annotate(an infer.Annotator) {
	an.Describe(&a.ConfigurationId, "ID of the default recording configuration; pass it to getVoiceRecordingConfiguration for its settings.")
	an.Describe(&a.UniqueName, "Unique name of the default configuration; empty when none is set.")
	an.Describe(&a.AccountSid, "SID of the account that owns the configuration.")
	an.Describe(&a.DateUpdated, "When the default configuration was last updated (ISO 8601).")
}

func (f *GetVoiceDefaultRecordingConfiguration) Annotate(a infer.Annotator) {
	a.Describe(&f, "Read which Twilio Voice v2 recording configuration is the account default. Fails when the account has none.")
}

func (f *GetVoiceDefaultRecordingConfiguration) Invoke(ctx context.Context, _ infer.FunctionRequest[GetVoiceDefaultRecordingConfigurationArgs]) (infer.FunctionResponse[GetVoiceDefaultRecordingConfigurationResult], error) {
	type resp = infer.FunctionResponse[GetVoiceDefaultRecordingConfigurationResult]
	c, err := clientFor(ctx)
	if err != nil {
		return resp{}, err
	}
	got, err := readVoiceDefault(ctx, c, defaultRecordingConfigPath, "recording")
	if err != nil {
		if isNotFound(err) {
			return resp{}, fmt.Errorf("getVoiceDefaultRecordingConfiguration: the account has no default recording configuration")
		}
		return resp{}, fmt.Errorf("getVoiceDefaultRecordingConfiguration: %w", err)
	}
	return resp{Output: GetVoiceDefaultRecordingConfigurationResult{ConfigurationId: got.configurationID, UniqueName: got.uniqueName, AccountSid: got.accountSid, DateUpdated: got.dateUpdated}}, nil
}

// VoiceDefaultTranscriptionConfiguration selects which existing Voice v2 transcription configuration is the account default.
type VoiceDefaultTranscriptionConfiguration struct{}

// VoiceDefaultTranscriptionConfigurationArgs are the inputs.
type VoiceDefaultTranscriptionConfigurationArgs struct {
	ConfigurationId string `pulumi:"configurationId"`
}

func (a *VoiceDefaultTranscriptionConfigurationArgs) Annotate(an infer.Annotator) {
	an.Describe(&a.ConfigurationId, "ID of the existing transcription configuration to make the default (a VoiceTranscriptionConfiguration's configurationId). Changing it selects another configuration.")
}

// VoiceDefaultTranscriptionConfigurationState is the persisted state.
type VoiceDefaultTranscriptionConfigurationState struct {
	VoiceDefaultTranscriptionConfigurationArgs

	// AccountSid is the account the default applies to and the resource ID.
	AccountSid string `pulumi:"accountSid"`
	// DateUpdated is when the selected configuration was last updated (ISO 8601).
	DateUpdated string `pulumi:"dateUpdated"`
}

func (s *VoiceDefaultTranscriptionConfigurationState) Annotate(a infer.Annotator) {
	a.Describe(&s.AccountSid, "SID of the account the default applies to; also the resource ID.")
	a.Describe(&s.DateUpdated, "When the selected configuration was last updated (ISO 8601).")
}

func (r *VoiceDefaultTranscriptionConfiguration) Annotate(a infer.Annotator) {
	a.Describe(&r, "Marks an existing Twilio Voice v2 transcription configuration as the account default (POST /v2/Configurations/Transcription/Default). "+
		"There is at most one default per account. Twilio documents no way to unset it, so destroy only forgets the selection: "+
		"the configuration stays the default until another is chosen. Import with the account SID. "+
		"The Voice v2 API is a new, versioned 0.0.1 product, so expect it to change.")
}

var _ infer.CustomResource[VoiceDefaultTranscriptionConfigurationArgs, VoiceDefaultTranscriptionConfigurationState] = (*VoiceDefaultTranscriptionConfiguration)(nil)
var _ infer.CustomRead[VoiceDefaultTranscriptionConfigurationArgs, VoiceDefaultTranscriptionConfigurationState] = (*VoiceDefaultTranscriptionConfiguration)(nil)
var _ infer.CustomUpdate[VoiceDefaultTranscriptionConfigurationArgs, VoiceDefaultTranscriptionConfigurationState] = (*VoiceDefaultTranscriptionConfiguration)(nil)
var _ infer.CustomDelete[VoiceDefaultTranscriptionConfigurationState] = (*VoiceDefaultTranscriptionConfiguration)(nil)

const defaultTranscriptionConfigPath = "/v2/Configurations/Transcription/Default"

func (r *VoiceDefaultTranscriptionConfiguration) Create(ctx context.Context, req infer.CreateRequest[VoiceDefaultTranscriptionConfigurationArgs]) (infer.CreateResponse[VoiceDefaultTranscriptionConfigurationState], error) {
	type resp = infer.CreateResponse[VoiceDefaultTranscriptionConfigurationState]
	a := req.Inputs
	if req.DryRun {
		return resp{Output: VoiceDefaultTranscriptionConfigurationState{VoiceDefaultTranscriptionConfigurationArgs: a}}, nil
	}
	c, err := clientFor(ctx)
	if err != nil {
		return resp{}, err
	}
	st, err := setVoiceDefault(ctx, c, defaultTranscriptionConfigPath, "transcription", a.ConfigurationId)
	if err != nil {
		return resp{}, err
	}
	return resp{ID: st.accountSid, Output: VoiceDefaultTranscriptionConfigurationState{VoiceDefaultTranscriptionConfigurationArgs: a, AccountSid: st.accountSid, DateUpdated: st.dateUpdated}}, nil
}

func (r *VoiceDefaultTranscriptionConfiguration) Read(ctx context.Context, req infer.ReadRequest[VoiceDefaultTranscriptionConfigurationArgs, VoiceDefaultTranscriptionConfigurationState]) (infer.ReadResponse[VoiceDefaultTranscriptionConfigurationArgs, VoiceDefaultTranscriptionConfigurationState], error) {
	type resp = infer.ReadResponse[VoiceDefaultTranscriptionConfigurationArgs, VoiceDefaultTranscriptionConfigurationState]
	c, err := clientFor(ctx)
	if err != nil {
		return resp{}, err
	}
	if c.accountSid != "" && req.ID != c.accountSid {
		return resp{}, fmt.Errorf("invalid VoiceDefaultTranscriptionConfiguration ID %q: it is the SID of the account the provider authenticates as (%q)", req.ID, c.accountSid)
	}
	got, err := readVoiceDefault(ctx, c, defaultTranscriptionConfigPath, "transcription")
	if err != nil {
		if isNotFound(err) {
			if err := notFoundOnImport("default transcription configuration", req.ID, req.Inputs, req.State); err != nil {
				return resp{}, err
			}
			return resp{}, nil
		}
		return resp{}, err
	}
	in := VoiceDefaultTranscriptionConfigurationArgs{ConfigurationId: got.configurationID}
	return resp{ID: req.ID, Inputs: in, State: VoiceDefaultTranscriptionConfigurationState{VoiceDefaultTranscriptionConfigurationArgs: in, AccountSid: got.accountSid, DateUpdated: got.dateUpdated}}, nil
}

func (r *VoiceDefaultTranscriptionConfiguration) Update(ctx context.Context, req infer.UpdateRequest[VoiceDefaultTranscriptionConfigurationArgs, VoiceDefaultTranscriptionConfigurationState]) (infer.UpdateResponse[VoiceDefaultTranscriptionConfigurationState], error) {
	type resp = infer.UpdateResponse[VoiceDefaultTranscriptionConfigurationState]
	st := req.State
	st.VoiceDefaultTranscriptionConfigurationArgs = req.Inputs
	if req.DryRun {
		return resp{Output: st}, nil
	}
	c, err := clientFor(ctx)
	if err != nil {
		return resp{}, err
	}
	got, err := setVoiceDefault(ctx, c, defaultTranscriptionConfigPath, "transcription", req.Inputs.ConfigurationId)
	if err != nil {
		return resp{}, err
	}
	st.AccountSid, st.DateUpdated = got.accountSid, got.dateUpdated
	return resp{Output: st}, nil
}

// Delete only forgets the selection: Twilio documents no call that unsets the default.
func (r *VoiceDefaultTranscriptionConfiguration) Delete(ctx context.Context, req infer.DeleteRequest[VoiceDefaultTranscriptionConfigurationState]) (infer.DeleteResponse, error) {
	return infer.DeleteResponse{}, nil
}

// GetVoiceDefaultTranscriptionConfiguration reads which transcription configuration is the account default.
type GetVoiceDefaultTranscriptionConfiguration struct{}

// GetVoiceDefaultTranscriptionConfigurationArgs has no arguments.
type GetVoiceDefaultTranscriptionConfigurationArgs struct{}

// GetVoiceDefaultTranscriptionConfigurationResult is the account default.
type GetVoiceDefaultTranscriptionConfigurationResult struct {
	ConfigurationId string `pulumi:"configurationId"`
	UniqueName      string `pulumi:"uniqueName"`
	AccountSid      string `pulumi:"accountSid"`
	DateUpdated     string `pulumi:"dateUpdated"`
}

func (a *GetVoiceDefaultTranscriptionConfigurationResult) Annotate(an infer.Annotator) {
	an.Describe(&a.ConfigurationId, "ID of the default transcription configuration; pass it to getVoiceTranscriptionConfiguration for its settings.")
	an.Describe(&a.UniqueName, "Unique name of the default configuration; empty when none is set.")
	an.Describe(&a.AccountSid, "SID of the account that owns the configuration.")
	an.Describe(&a.DateUpdated, "When the default configuration was last updated (ISO 8601).")
}

func (f *GetVoiceDefaultTranscriptionConfiguration) Annotate(a infer.Annotator) {
	a.Describe(&f, "Read which Twilio Voice v2 transcription configuration is the account default. Fails when the account has none.")
}

func (f *GetVoiceDefaultTranscriptionConfiguration) Invoke(ctx context.Context, _ infer.FunctionRequest[GetVoiceDefaultTranscriptionConfigurationArgs]) (infer.FunctionResponse[GetVoiceDefaultTranscriptionConfigurationResult], error) {
	type resp = infer.FunctionResponse[GetVoiceDefaultTranscriptionConfigurationResult]
	c, err := clientFor(ctx)
	if err != nil {
		return resp{}, err
	}
	got, err := readVoiceDefault(ctx, c, defaultTranscriptionConfigPath, "transcription")
	if err != nil {
		if isNotFound(err) {
			return resp{}, fmt.Errorf("getVoiceDefaultTranscriptionConfiguration: the account has no default transcription configuration")
		}
		return resp{}, fmt.Errorf("getVoiceDefaultTranscriptionConfiguration: %w", err)
	}
	return resp{Output: GetVoiceDefaultTranscriptionConfigurationResult{ConfigurationId: got.configurationID, UniqueName: got.uniqueName, AccountSid: got.accountSid, DateUpdated: got.dateUpdated}}, nil
}
