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

// GetVoiceRecordingConfiguration looks up a Voice v2 recording configuration by ID or unique name.
type GetVoiceRecordingConfiguration struct{}

// GetVoiceRecordingConfigurationArgs are the lookup arguments.
type GetVoiceRecordingConfigurationArgs struct {
	IdOrUniqueName string `pulumi:"idOrUniqueName"`
}

func (a *GetVoiceRecordingConfigurationArgs) Annotate(an infer.Annotator) {
	an.Describe(&a.IdOrUniqueName, "The configuration ID or its unique name.")
}

// GetVoiceRecordingConfigurationResult is the looked-up configuration.
type GetVoiceRecordingConfigurationResult struct {
	ConfigurationId                   string                      `pulumi:"configurationId"`
	UniqueName                        string                      `pulumi:"uniqueName"`
	Description                       string                      `pulumi:"description"`
	CompositionPolicy                 *RecordingCompositionPolicy `pulumi:"compositionPolicy,optional"`
	CallRecordingStatusCallback       *RecordingStatusCallback    `pulumi:"callRecordingStatusCallback,optional"`
	ConferenceRecordingStatusCallback *RecordingStatusCallback    `pulumi:"conferenceRecordingStatusCallback,optional"`
	Features                          []RecordingFeature          `pulumi:"features"`
	AccountSid                        string                      `pulumi:"accountSid"`
	DateCreated                       string                      `pulumi:"dateCreated"`
	DateUpdated                       string                      `pulumi:"dateUpdated"`
}

func (a *GetVoiceRecordingConfigurationResult) Annotate(an infer.Annotator) {
	an.Describe(&a.ConfigurationId, "The configuration ID.")
	an.Describe(&a.UniqueName, "The unique name; empty when none is set.")
	an.Describe(&a.Description, "The description; empty when none is set.")
	an.Describe(&a.CompositionPolicy, "How recordings are composed; unset when Twilio reports none.")
	an.Describe(&a.CallRecordingStatusCallback, "Where call recording status events are reported; unset when none.")
	an.Describe(&a.ConferenceRecordingStatusCallback, "Where conference recording status events are reported; unset when none.")
	an.Describe(&a.Features, "Features applied to the recording; empty when none.")
	an.Describe(&a.AccountSid, "SID of the account that owns the configuration.")
	an.Describe(&a.DateCreated, "When the configuration was created (ISO 8601).")
	an.Describe(&a.DateUpdated, "When the configuration was last updated (ISO 8601).")
}

func (f *GetVoiceRecordingConfiguration) Annotate(a infer.Annotator) {
	a.Describe(&f, "Look up a Voice v2 recording configuration by ID or unique name.")
}

func (f *GetVoiceRecordingConfiguration) Invoke(ctx context.Context, req infer.FunctionRequest[GetVoiceRecordingConfigurationArgs]) (infer.FunctionResponse[GetVoiceRecordingConfigurationResult], error) {
	const fn = "getVoiceRecordingConfiguration"
	type out = infer.FunctionResponse[GetVoiceRecordingConfigurationResult]
	if err := requireArg(fn, "idOrUniqueName", req.Input.IdOrUniqueName); err != nil {
		return out{}, err
	}
	c, err := clientFor(ctx)
	if err != nil {
		return out{}, err
	}
	o, err := c.doJSON(ctx, "GET", voiceProduct, recordingConfigPath(req.Input.IdOrUniqueName), nil, nil)
	if err != nil {
		return out{}, lookupFailure(fn, "recording configuration "+req.Input.IdOrUniqueName, err)
	}
	got, err := decodeRecordingConfiguration(o)
	if err != nil {
		return out{}, err
	}
	cfg := got.Configuration
	res := GetVoiceRecordingConfigurationResult{ConfigurationId: got.ID, UniqueName: deref(got.UniqueName), Description: deref(got.Description),
		Features: cfg.Features, AccountSid: got.AccountSid, DateCreated: got.DateCreated, DateUpdated: got.DateUpdated}
	if res.Features == nil {
		res.Features = []RecordingFeature{}
	}
	if !cfg.CompositionPolicy.empty() {
		res.CompositionPolicy = cfg.CompositionPolicy
	}
	if !cfg.CallRecordingStatusCallback.empty() {
		res.CallRecordingStatusCallback = cfg.CallRecordingStatusCallback
	}
	if !cfg.ConferenceRecordingStatusCallback.empty() {
		res.ConferenceRecordingStatusCallback = cfg.ConferenceRecordingStatusCallback
	}
	return out{Output: res}, nil
}

// GetVoiceTranscriptionConfiguration looks up a Voice v2 transcription configuration by ID or unique name.
type GetVoiceTranscriptionConfiguration struct{}

// GetVoiceTranscriptionConfigurationArgs are the lookup arguments.
type GetVoiceTranscriptionConfigurationArgs struct {
	IdOrUniqueName string `pulumi:"idOrUniqueName"`
}

func (a *GetVoiceTranscriptionConfigurationArgs) Annotate(an infer.Annotator) {
	an.Describe(&a.IdOrUniqueName, "The configuration ID or its unique name.")
}

// GetVoiceTranscriptionConfigurationResult is the looked-up configuration.
type GetVoiceTranscriptionConfigurationResult struct {
	ConfigurationId             string                            `pulumi:"configurationId"`
	UniqueName                  string                            `pulumi:"uniqueName"`
	Description                 string                            `pulumi:"description"`
	TranscriptionEngine         string                            `pulumi:"transcriptionEngine"`
	SpeechModel                 string                            `pulumi:"speechModel"`
	Language                    string                            `pulumi:"language"`
	TranscriptionStatusCallback *TranscriptionStatusCallback      `pulumi:"transcriptionStatusCallback,optional"`
	ConversationConfigurationId string                            `pulumi:"conversationConfigurationId"`
	ParticipantDefaults         []TranscriptionParticipantDefault `pulumi:"participantDefaults"`
	AccountSid                  string                            `pulumi:"accountSid"`
	DateCreated                 string                            `pulumi:"dateCreated"`
	DateUpdated                 string                            `pulumi:"dateUpdated"`
}

func (a *GetVoiceTranscriptionConfigurationResult) Annotate(an infer.Annotator) {
	an.Describe(&a.ConfigurationId, "The configuration ID.")
	an.Describe(&a.UniqueName, "The unique name; empty when none is set.")
	an.Describe(&a.Description, "The description; empty when none is set.")
	an.Describe(&a.TranscriptionEngine, "The transcription engine provider.")
	an.Describe(&a.SpeechModel, "The speech model; empty when none is set.")
	an.Describe(&a.Language, "The language code.")
	an.Describe(&a.TranscriptionStatusCallback, "Webhook called when a transcription is complete; unset when none.")
	an.Describe(&a.ConversationConfigurationId, "ID of the Conversations configuration; empty when none is set.")
	an.Describe(&a.ParticipantDefaults, "Default participant roles per audio channel.")
	an.Describe(&a.AccountSid, "SID of the account that owns the configuration.")
	an.Describe(&a.DateCreated, "When the configuration was created (ISO 8601).")
	an.Describe(&a.DateUpdated, "When the configuration was last updated (ISO 8601).")
}

func (f *GetVoiceTranscriptionConfiguration) Annotate(a infer.Annotator) {
	a.Describe(&f, "Look up a Voice v2 transcription configuration by ID or unique name.")
}

func (f *GetVoiceTranscriptionConfiguration) Invoke(ctx context.Context, req infer.FunctionRequest[GetVoiceTranscriptionConfigurationArgs]) (infer.FunctionResponse[GetVoiceTranscriptionConfigurationResult], error) {
	const fn = "getVoiceTranscriptionConfiguration"
	type out = infer.FunctionResponse[GetVoiceTranscriptionConfigurationResult]
	if err := requireArg(fn, "idOrUniqueName", req.Input.IdOrUniqueName); err != nil {
		return out{}, err
	}
	c, err := clientFor(ctx)
	if err != nil {
		return out{}, err
	}
	o, err := c.doJSON(ctx, "GET", voiceProduct, transcriptionConfigPath(req.Input.IdOrUniqueName), nil, nil)
	if err != nil {
		return out{}, lookupFailure(fn, "transcription configuration "+req.Input.IdOrUniqueName, err)
	}
	got, err := decodeTranscriptionConfiguration(o)
	if err != nil {
		return out{}, err
	}
	cfg := got.Configuration
	res := GetVoiceTranscriptionConfigurationResult{ConfigurationId: got.ID, UniqueName: deref(got.UniqueName), Description: deref(got.Description),
		TranscriptionEngine: cfg.TranscriptionEngine, SpeechModel: deref(cfg.SpeechModel), Language: cfg.Language,
		ConversationConfigurationId: deref(cfg.ConversationConfigurationId), ParticipantDefaults: cfg.ParticipantDefaults,
		AccountSid: got.AccountSid, DateCreated: got.DateCreated, DateUpdated: got.DateUpdated}
	if res.ParticipantDefaults == nil {
		res.ParticipantDefaults = []TranscriptionParticipantDefault{}
	}
	if !cfg.TranscriptionStatusCallback.empty() {
		res.TranscriptionStatusCallback = cfg.TranscriptionStatusCallback
	}
	return out{Output: res}, nil
}
