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
	"encoding/json"
	"fmt"
	"net/url"
	"slices"

	p "github.com/pulumi/pulumi-go-provider"
	"github.com/pulumi/pulumi-go-provider/infer"
)

// The Voice v2 recording and transcription configurations live on voice.twilio.com
// (/v2/Configurations/Recording and /v2/Configurations/Transcription). Unlike the rest of the
// provider they take and return JSON bodies. Nothing in their paths names an account.
//
// Both are named, reusable configurations (a collection, not a singleton): create one, then
// reference its ID from a call. The account-default configuration (/v2/AccountDefaultConfiguration)
// is a separate singleton and is not managed here.
//
// The Voice v2 definition is version 0.0.1 and the products are new: the shapes below follow
// twilio-oai spec/json/twilio_voice_v2.json read 2026-11-09 and have not been checked against a
// live account.

func recordingConfigPath(id string) string {
	path := "/v2/Configurations/Recording"
	if id != "" {
		path += "/" + url.PathEscape(id)
	}
	return path
}

func transcriptionConfigPath(id string) string {
	path := "/v2/Configurations/Transcription"
	if id != "" {
		path += "/" + url.PathEscape(id)
	}
	return path
}

// decodeJSONObject copies a decoded response object into v through JSON, so the nested
// configuration lands in the same typed structs the inputs use.
func decodeJSONObject(o obj, v any) error {
	raw, err := json.Marshal(o)
	if err != nil {
		return err
	}
	return json.Unmarshal(raw, v)
}

// ------------------------------------------------------------- shared nested types

// RecordingCompositionPolicy says how the recording is composed.
type RecordingCompositionPolicy struct {
	Channels *string `pulumi:"channels,optional" json:"channels,omitempty"`
	Trim     *string `pulumi:"trim,optional" json:"trim,omitempty"`
	Track    *string `pulumi:"track,optional" json:"track,omitempty"`
}

func (a *RecordingCompositionPolicy) Annotate(an infer.Annotator) {
	an.Describe(&a.Channels, "Number of channels to compose: dual or mono.")
	an.Describe(&a.Trim, "Whether silence is trimmed from the recording: trim-silence or do-not-trim.")
	an.Describe(&a.Track, "Which audio track to record: inbound, outbound or both.")
}

func (a *RecordingCompositionPolicy) empty() bool {
	return a == nil || (a.Channels == nil && a.Trim == nil && a.Track == nil)
}

// RecordingStatusCallback is where Twilio reports the recording status.
type RecordingStatusCallback struct {
	Url    *string  `pulumi:"url,optional" json:"url,omitempty"`
	Method *string  `pulumi:"method,optional" json:"method,omitempty"`
	Events []string `pulumi:"events,optional" json:"events,omitempty"`
}

func (a *RecordingStatusCallback) Annotate(an infer.Annotator) {
	an.Describe(&a.Url, "URL Twilio requests with the recording status.")
	an.Describe(&a.Method, "HTTP method used to request the URL: GET or POST.")
	an.Describe(&a.Events, "Status events to report: in-progress, completed, absent and failed.")
}

func (a *RecordingStatusCallback) empty() bool {
	return a == nil || (a.Url == nil && a.Method == nil && len(a.Events) == 0)
}

// RecordingFeature is a feature applied to the recording, today a transcription configuration.
type RecordingFeature struct {
	Type        string  `pulumi:"type" json:"type"`
	FeatureId   string  `pulumi:"featureId" json:"featureId"`
	Description *string `pulumi:"description,optional" json:"description,omitempty"`
}

func (a *RecordingFeature) Annotate(an infer.Annotator) {
	an.Describe(&a.Type, "The feature type. Only transcriptionConfiguration exists today.")
	an.Describe(&a.FeatureId, "The ID of the feature, for example the configurationId of a VoiceTranscriptionConfiguration.")
	an.Describe(&a.Description, "Free-text description of the feature.")
}

// TranscriptionStatusCallback is where Twilio reports that a transcription is complete.
type TranscriptionStatusCallback struct {
	Url    *string `pulumi:"url,optional" json:"url,omitempty"`
	Method *string `pulumi:"method,optional" json:"method,omitempty"`
}

func (a *TranscriptionStatusCallback) Annotate(an infer.Annotator) {
	an.Describe(&a.Url, "Webhook URL Twilio requests when the transcription is complete.")
	an.Describe(&a.Method, "HTTP method used to request the URL. Only POST is accepted.")
}

func (a *TranscriptionStatusCallback) empty() bool {
	return a == nil || (a.Url == nil && a.Method == nil)
}

// TranscriptionParticipantDefault maps an audio channel to a participant role.
type TranscriptionParticipantDefault struct {
	AudioChannelIndex int    `pulumi:"audioChannelIndex" json:"audioChannelIndex"`
	Type              string `pulumi:"type" json:"type"`
}

func (a *TranscriptionParticipantDefault) Annotate(an infer.Annotator) {
	an.Describe(&a.AudioChannelIndex, "One-based index of the audio channel of a multi-channel recording.")
	an.Describe(&a.Type, "The participant role of that channel: CUSTOMER, HUMAN_AGENT or AI_AGENT.")
}

// ------------------------------------------------------------------- input checks

var (
	recordingChannels = []string{"dual", "mono"}
	recordingTrims    = []string{"trim-silence", "do-not-trim"}
	recordingTracks   = []string{"inbound", "outbound", "both"}
	recordingMethods  = []string{"GET", "POST"}
	recordingEvents   = []string{"in-progress", "completed", "absent", "failed"}
	participantTypes  = []string{"CUSTOMER", "HUMAN_AGENT", "AI_AGENT"}
)

func oneOf(failures []p.CheckFailure, prop string, v *string, allowed []string) []p.CheckFailure {
	if v != nil && !slices.Contains(allowed, *v) {
		failures = append(failures, p.CheckFailure{Property: prop, Reason: fmt.Sprintf("%s %q must be one of %v", prop, *v, allowed)})
	}
	return failures
}

// checkRecordingCallback validates one recording status callback, unless its property is unknown.
func checkRecordingCallback(failures []p.CheckFailure, unk func(...string) bool, prop string, cb *RecordingStatusCallback) []p.CheckFailure {
	if cb == nil || unk(prop) {
		return failures
	}
	failures = oneOf(failures, prop+".method", cb.Method, recordingMethods)
	for i, e := range cb.Events {
		failures = oneOf(failures, fmt.Sprintf("%s.events[%d]", prop, i), &e, recordingEvents)
	}
	return failures
}

// ------------------------------------------------------- VoiceRecordingConfiguration

// VoiceRecordingConfiguration manages a Twilio Voice v2 recording configuration: how calls and
// conferences are recorded and where the recording status goes. It is a named, reusable
// configuration that calls refer to by its ID.
type VoiceRecordingConfiguration struct{}

// VoiceRecordingConfigurationArgs are the inputs of a VoiceRecordingConfiguration. Every input
// is optional and updates in place; the whole configuration is sent on each update.
type VoiceRecordingConfigurationArgs struct {
	UniqueName                        *string                     `pulumi:"uniqueName,optional"`
	Description                       *string                     `pulumi:"description,optional"`
	CompositionPolicy                 *RecordingCompositionPolicy `pulumi:"compositionPolicy,optional"`
	CallRecordingStatusCallback       *RecordingStatusCallback    `pulumi:"callRecordingStatusCallback,optional"`
	ConferenceRecordingStatusCallback *RecordingStatusCallback    `pulumi:"conferenceRecordingStatusCallback,optional"`
	Features                          []RecordingFeature          `pulumi:"features,optional"`
}

func (a *VoiceRecordingConfigurationArgs) Annotate(an infer.Annotator) {
	an.Describe(&a.UniqueName, "Unique name of the configuration; it can be used instead of the ID when referring to it.")
	an.Describe(&a.Description, "Free-text description of the configuration.")
	an.Describe(&a.CompositionPolicy, "How recordings are composed: channels, silence trimming and track.")
	an.Describe(&a.CallRecordingStatusCallback, "Where call recording status events are reported.")
	an.Describe(&a.ConferenceRecordingStatusCallback, "Where conference recording status events are reported.")
	an.Describe(&a.Features, "Features applied to the recording (at most 3), for example a transcription configuration.")
}

// VoiceRecordingConfigurationState is the persisted state of a VoiceRecordingConfiguration.
type VoiceRecordingConfigurationState struct {
	VoiceRecordingConfigurationArgs

	// ConfigurationId is the configuration ID and the resource ID.
	ConfigurationId string `pulumi:"configurationId"`
	// AccountSid is the account that owns the configuration.
	AccountSid string `pulumi:"accountSid"`
	// DateCreated and DateUpdated are ISO 8601 timestamps.
	DateCreated string `pulumi:"dateCreated"`
	DateUpdated string `pulumi:"dateUpdated"`
}

func (s *VoiceRecordingConfigurationState) Annotate(a infer.Annotator) {
	a.Describe(&s.ConfigurationId, "The configuration ID Twilio assigned.")
	a.Describe(&s.AccountSid, "SID of the account that owns the configuration.")
	a.Describe(&s.DateCreated, "When the configuration was created (ISO 8601).")
	a.Describe(&s.DateUpdated, "When the configuration was last updated (ISO 8601).")
}

func (r *VoiceRecordingConfiguration) Annotate(a infer.Annotator) {
	a.Describe(&r, "A Twilio Voice v2 recording configuration: how calls and conferences are recorded and where the recording status is reported. "+
		"Import with the configuration ID. The Voice v2 API is a new, versioned 0.0.1 product, so expect it to change.")
}

var _ infer.CustomResource[VoiceRecordingConfigurationArgs, VoiceRecordingConfigurationState] = (*VoiceRecordingConfiguration)(nil)
var _ infer.CustomCheck[VoiceRecordingConfigurationArgs] = (*VoiceRecordingConfiguration)(nil)
var _ infer.CustomRead[VoiceRecordingConfigurationArgs, VoiceRecordingConfigurationState] = (*VoiceRecordingConfiguration)(nil)
var _ infer.CustomUpdate[VoiceRecordingConfigurationArgs, VoiceRecordingConfigurationState] = (*VoiceRecordingConfiguration)(nil)
var _ infer.CustomDelete[VoiceRecordingConfigurationState] = (*VoiceRecordingConfiguration)(nil)

func (r *VoiceRecordingConfiguration) Check(ctx context.Context, req infer.CheckRequest) (infer.CheckResponse[VoiceRecordingConfigurationArgs], error) {
	args, failures, err := infer.DefaultCheck[VoiceRecordingConfigurationArgs](ctx, req.NewInputs)
	if err != nil || len(failures) > 0 {
		return infer.CheckResponse[VoiceRecordingConfigurationArgs]{Inputs: args, Failures: failures}, err
	}
	unk := unknowns(req.NewInputs)
	if args.UniqueName != nil && !unk("uniqueName") && *args.UniqueName == "" {
		failures = append(failures, p.CheckFailure{Property: "uniqueName", Reason: "uniqueName must not be empty"})
	}
	if cp := args.CompositionPolicy; cp != nil && !unk("compositionPolicy") {
		failures = oneOf(failures, "compositionPolicy.channels", cp.Channels, recordingChannels)
		failures = oneOf(failures, "compositionPolicy.trim", cp.Trim, recordingTrims)
		failures = oneOf(failures, "compositionPolicy.track", cp.Track, recordingTracks)
	}
	failures = checkRecordingCallback(failures, unk, "callRecordingStatusCallback", args.CallRecordingStatusCallback)
	failures = checkRecordingCallback(failures, unk, "conferenceRecordingStatusCallback", args.ConferenceRecordingStatusCallback)
	if !unk("features") {
		if len(args.Features) > 3 {
			failures = append(failures, p.CheckFailure{Property: "features", Reason: "features takes at most 3 entries"})
		}
		for i, f := range args.Features {
			// A feature wired to another resource (featureId: ${transcription.configurationId}) is
			// unknown at preview and decodes to "": skip it until it is known.
			if !unknownAt(req.NewInputs, "features", i, "type") {
				ft := f.Type
				failures = oneOf(failures, fmt.Sprintf("features[%d].type", i), &ft, []string{"transcriptionConfiguration"})
			}
			if f.FeatureId == "" && !unknownAt(req.NewInputs, "features", i, "featureId") {
				failures = append(failures, p.CheckFailure{Property: fmt.Sprintf("features[%d].featureId", i), Reason: "featureId must not be empty"})
			}
		}
	}
	return infer.CheckResponse[VoiceRecordingConfigurationArgs]{Inputs: args, Failures: failures}, nil
}

type recordingConfigurationBody struct {
	Configuration recordingConfigurationJSON `json:"configuration"`
	UniqueName    *string                    `json:"unique_name,omitempty"`
	Description   *string                    `json:"description,omitempty"`
}

type recordingConfigurationJSON struct {
	ConfigurationType                 string                      `json:"configurationType"`
	CompositionPolicy                 *RecordingCompositionPolicy `json:"compositionPolicy,omitempty"`
	CallRecordingStatusCallback       *RecordingStatusCallback    `json:"callRecordingStatusCallback,omitempty"`
	ConferenceRecordingStatusCallback *RecordingStatusCallback    `json:"conferenceRecordingStatusCallback,omitempty"`
	Features                          []RecordingFeature          `json:"features,omitempty"`
}

func (a VoiceRecordingConfigurationArgs) body() recordingConfigurationBody {
	return recordingConfigurationBody{UniqueName: a.UniqueName, Description: a.Description,
		Configuration: recordingConfigurationJSON{ConfigurationType: "Recording", CompositionPolicy: a.CompositionPolicy,
			CallRecordingStatusCallback: a.CallRecordingStatusCallback, ConferenceRecordingStatusCallback: a.ConferenceRecordingStatusCallback,
			Features: a.Features}}
}

// recordingConfigurationResponse is what Twilio answers for a recording configuration.
type recordingConfigurationResponse struct {
	ID            string                     `json:"id"`
	AccountSid    string                     `json:"account_sid"`
	UniqueName    *string                    `json:"unique_name"`
	Description   *string                    `json:"description"`
	DateCreated   string                     `json:"date_created"`
	DateUpdated   string                     `json:"date_updated"`
	Configuration recordingConfigurationJSON `json:"configuration"`
}

func decodeRecordingConfiguration(o obj) (recordingConfigurationResponse, error) {
	var out recordingConfigurationResponse
	if err := decodeJSONObject(o, &out); err != nil {
		return out, fmt.Errorf("decode recording configuration: %w", err)
	}
	if out.ID == "" {
		return out, fmt.Errorf("decode recording configuration: no id in the response")
	}
	return out, nil
}

func recordingStateFrom(in VoiceRecordingConfigurationArgs, resp recordingConfigurationResponse) VoiceRecordingConfigurationState {
	return VoiceRecordingConfigurationState{VoiceRecordingConfigurationArgs: in, ConfigurationId: resp.ID, AccountSid: resp.AccountSid,
		DateCreated: resp.DateCreated, DateUpdated: resp.DateUpdated}
}

// adoptCallback returns the callback to record for a refresh: what Twilio reports, unless that
// is empty and the program did not set one. A method Twilio defaults to POST is not adopted when
// the program left the method out.
func adoptCallback(in, got *RecordingStatusCallback) *RecordingStatusCallback {
	if got.empty() {
		return nil
	}
	out := *got
	if in != nil && in.Method == nil && out.Method != nil && *out.Method == "POST" {
		out.Method = nil
	}
	if in != nil && sameSet(in.Events, out.Events) {
		out.Events = in.Events
	}
	if out.empty() {
		return nil
	}
	return &out
}

func sameSet(a, b []string) bool {
	if len(a) != len(b) {
		return false
	}
	x, y := slices.Clone(a), slices.Clone(b)
	slices.Sort(x)
	slices.Sort(y)
	return slices.Equal(x, y)
}

func adoptRecording(in VoiceRecordingConfigurationArgs, resp recordingConfigurationResponse) VoiceRecordingConfigurationArgs {
	out := in
	out.UniqueName = adoptOptional(in.UniqueName, resp.UniqueName)
	out.Description = adoptOptional(in.Description, resp.Description)
	out.CompositionPolicy = nil
	if !resp.Configuration.CompositionPolicy.empty() {
		out.CompositionPolicy = resp.Configuration.CompositionPolicy
	}
	out.CallRecordingStatusCallback = adoptCallback(in.CallRecordingStatusCallback, resp.Configuration.CallRecordingStatusCallback)
	out.ConferenceRecordingStatusCallback = adoptCallback(in.ConferenceRecordingStatusCallback, resp.Configuration.ConferenceRecordingStatusCallback)
	out.Features = resp.Configuration.Features
	if len(out.Features) == 0 {
		out.Features = nil
	}
	return out
}

// adoptOptional returns what Twilio reports for an optional string, or nil when it reports none
// (or an empty string) and the program did not set the input.
func adoptOptional(in, got *string) *string {
	if got == nil || (*got == "" && in == nil) {
		return nil
	}
	return got
}

func (r *VoiceRecordingConfiguration) Create(ctx context.Context, req infer.CreateRequest[VoiceRecordingConfigurationArgs]) (infer.CreateResponse[VoiceRecordingConfigurationState], error) {
	a := req.Inputs
	if req.DryRun {
		return infer.CreateResponse[VoiceRecordingConfigurationState]{Output: VoiceRecordingConfigurationState{VoiceRecordingConfigurationArgs: a}}, nil
	}
	c, err := clientFor(ctx)
	if err != nil {
		return infer.CreateResponse[VoiceRecordingConfigurationState]{}, err
	}
	o, err := c.doJSON(ctx, "POST", voiceProduct, recordingConfigPath(""), nil, a.body())
	if err != nil {
		return infer.CreateResponse[VoiceRecordingConfigurationState]{}, fmt.Errorf("create recording configuration: %w", err)
	}
	resp, err := decodeRecordingConfiguration(o)
	if err != nil {
		return infer.CreateResponse[VoiceRecordingConfigurationState]{}, err
	}
	return infer.CreateResponse[VoiceRecordingConfigurationState]{ID: resp.ID, Output: recordingStateFrom(a, resp)}, nil
}

func (r *VoiceRecordingConfiguration) Read(ctx context.Context, req infer.ReadRequest[VoiceRecordingConfigurationArgs, VoiceRecordingConfigurationState]) (infer.ReadResponse[VoiceRecordingConfigurationArgs, VoiceRecordingConfigurationState], error) {
	type resp = infer.ReadResponse[VoiceRecordingConfigurationArgs, VoiceRecordingConfigurationState]
	c, err := clientFor(ctx)
	if err != nil {
		return resp{}, err
	}
	o, err := c.doJSON(ctx, "GET", voiceProduct, recordingConfigPath(req.ID), nil, nil)
	if err != nil {
		if isNotFound(err) {
			if err := notFoundOnImport("recording configuration", req.ID, req.Inputs, req.State); err != nil {
				return resp{}, err
			}
			return resp{}, nil
		}
		return resp{}, fmt.Errorf("read recording configuration %s: %w", req.ID, err)
	}
	got, err := decodeRecordingConfiguration(o)
	if err != nil {
		return resp{}, err
	}
	in := adoptRecording(req.Inputs, got)
	return resp{ID: req.ID, Inputs: in, State: recordingStateFrom(in, got)}, nil
}

func (r *VoiceRecordingConfiguration) Update(ctx context.Context, req infer.UpdateRequest[VoiceRecordingConfigurationArgs, VoiceRecordingConfigurationState]) (infer.UpdateResponse[VoiceRecordingConfigurationState], error) {
	a := req.Inputs
	st := req.State
	st.VoiceRecordingConfigurationArgs = a
	if req.DryRun {
		return infer.UpdateResponse[VoiceRecordingConfigurationState]{Output: st}, nil
	}
	c, err := clientFor(ctx)
	if err != nil {
		return infer.UpdateResponse[VoiceRecordingConfigurationState]{}, err
	}
	o, err := c.doJSON(ctx, "PUT", voiceProduct, recordingConfigPath(req.ID), nil, a.body())
	if err != nil {
		return infer.UpdateResponse[VoiceRecordingConfigurationState]{}, fmt.Errorf("update recording configuration %s: %w", req.ID, err)
	}
	resp, err := decodeRecordingConfiguration(o)
	if err != nil {
		return infer.UpdateResponse[VoiceRecordingConfigurationState]{}, err
	}
	return infer.UpdateResponse[VoiceRecordingConfigurationState]{Output: recordingStateFrom(a, resp)}, nil
}

func (r *VoiceRecordingConfiguration) Delete(ctx context.Context, req infer.DeleteRequest[VoiceRecordingConfigurationState]) (infer.DeleteResponse, error) {
	c, err := clientFor(ctx)
	if err != nil {
		return infer.DeleteResponse{}, err
	}
	if _, err := c.doJSON(ctx, "DELETE", voiceProduct, recordingConfigPath(req.ID), nil, nil); err != nil && !isNotFound(err) {
		return infer.DeleteResponse{}, fmt.Errorf("delete recording configuration %s: %w", req.ID, err)
	}
	return infer.DeleteResponse{}, nil
}

// ------------------------------------------------------ VoiceTranscriptionConfiguration

// VoiceTranscriptionConfiguration manages a Twilio Voice v2 transcription configuration: the
// speech engine, language and participants of a call transcription. It is a named, reusable
// configuration that calls and recording configurations refer to by its ID.
type VoiceTranscriptionConfiguration struct{}

// VoiceTranscriptionConfigurationArgs are the inputs of a VoiceTranscriptionConfiguration. The
// whole configuration is sent on each update, so every input updates in place.
type VoiceTranscriptionConfigurationArgs struct {
	UniqueName                  *string                           `pulumi:"uniqueName,optional"`
	Description                 *string                           `pulumi:"description,optional"`
	TranscriptionEngine         string                            `pulumi:"transcriptionEngine"`
	SpeechModel                 *string                           `pulumi:"speechModel,optional"`
	Language                    string                            `pulumi:"language"`
	TranscriptionStatusCallback *TranscriptionStatusCallback      `pulumi:"transcriptionStatusCallback,optional"`
	ConversationConfigurationId *string                           `pulumi:"conversationConfigurationId,optional"`
	ParticipantDefaults         []TranscriptionParticipantDefault `pulumi:"participantDefaults"`
}

func (a *VoiceTranscriptionConfigurationArgs) Annotate(an infer.Annotator) {
	an.Describe(&a.UniqueName, "Unique name of the configuration; it can be used instead of the ID when referring to it.")
	an.Describe(&a.Description, "Free-text description of the configuration.")
	an.Describe(&a.TranscriptionEngine, "The transcription engine provider: google, deepgram or twilio_managed. With twilio_managed do not set speechModel.")
	an.Describe(&a.SpeechModel, "The speech model. google: chirp_2; deepgram: nova-3 or nova-2; twilio_managed: twilio_managed (or leave it out).")
	an.Describe(&a.Language, "A BCP-47 language code such as en-US, or multi for multilingual code switching (Deepgram only).")
	an.Describe(&a.TranscriptionStatusCallback, "Webhook called when a transcription is complete. Twilio needs this or conversationConfigurationId (or both).")
	an.Describe(&a.ConversationConfigurationId, "ID of the Conversations configuration that stores the transcription and its add-ons.")
	an.Describe(&a.ParticipantDefaults, "Default participant roles per audio channel, for diarization. Twilio expects exactly two entries, for channels 1 and 2.")
}

// VoiceTranscriptionConfigurationState is the persisted state of a VoiceTranscriptionConfiguration.
type VoiceTranscriptionConfigurationState struct {
	VoiceTranscriptionConfigurationArgs

	// ConfigurationId is the configuration ID and the resource ID.
	ConfigurationId string `pulumi:"configurationId"`
	// AccountSid is the account that owns the configuration.
	AccountSid string `pulumi:"accountSid"`
	// DateCreated and DateUpdated are ISO 8601 timestamps.
	DateCreated string `pulumi:"dateCreated"`
	DateUpdated string `pulumi:"dateUpdated"`
}

func (s *VoiceTranscriptionConfigurationState) Annotate(a infer.Annotator) {
	a.Describe(&s.ConfigurationId, "The configuration ID Twilio assigned.")
	a.Describe(&s.AccountSid, "SID of the account that owns the configuration.")
	a.Describe(&s.DateCreated, "When the configuration was created (ISO 8601).")
	a.Describe(&s.DateUpdated, "When the configuration was last updated (ISO 8601).")
}

func (r *VoiceTranscriptionConfiguration) Annotate(a infer.Annotator) {
	a.Describe(&r, "A Twilio Voice v2 transcription configuration: the speech engine, language and participant roles of a call transcription. "+
		"Reference its configurationId from a VoiceRecordingConfiguration feature. Import with the configuration ID. "+
		"The Voice v2 API is a new, versioned 0.0.1 product, so expect it to change.")
}

var _ infer.CustomResource[VoiceTranscriptionConfigurationArgs, VoiceTranscriptionConfigurationState] = (*VoiceTranscriptionConfiguration)(nil)
var _ infer.CustomCheck[VoiceTranscriptionConfigurationArgs] = (*VoiceTranscriptionConfiguration)(nil)
var _ infer.CustomRead[VoiceTranscriptionConfigurationArgs, VoiceTranscriptionConfigurationState] = (*VoiceTranscriptionConfiguration)(nil)
var _ infer.CustomUpdate[VoiceTranscriptionConfigurationArgs, VoiceTranscriptionConfigurationState] = (*VoiceTranscriptionConfiguration)(nil)
var _ infer.CustomDelete[VoiceTranscriptionConfigurationState] = (*VoiceTranscriptionConfiguration)(nil)

func (r *VoiceTranscriptionConfiguration) Check(ctx context.Context, req infer.CheckRequest) (infer.CheckResponse[VoiceTranscriptionConfigurationArgs], error) {
	args, failures, err := infer.DefaultCheck[VoiceTranscriptionConfigurationArgs](ctx, req.NewInputs)
	if err != nil || len(failures) > 0 {
		return infer.CheckResponse[VoiceTranscriptionConfigurationArgs]{Inputs: args, Failures: failures}, err
	}
	unk := unknowns(req.NewInputs)
	fail := func(prop, reason string) { failures = append(failures, p.CheckFailure{Property: prop, Reason: reason}) }
	if args.UniqueName != nil && !unk("uniqueName") && *args.UniqueName == "" {
		fail("uniqueName", "uniqueName must not be empty")
	}
	if !unk("transcriptionEngine") && args.TranscriptionEngine == "" {
		fail("transcriptionEngine", "transcriptionEngine must not be empty")
	}
	if !unk("language") && args.Language == "" {
		fail("language", "language must not be empty")
	}
	if cb := args.TranscriptionStatusCallback; cb != nil && !unk("transcriptionStatusCallback") && cb.Method != nil && *cb.Method != "POST" {
		fail("transcriptionStatusCallback.method", fmt.Sprintf("method %q must be POST", *cb.Method))
	}
	if !unk("participantDefaults") {
		for i, d := range args.ParticipantDefaults {
			if !unknownAt(req.NewInputs, "participantDefaults", i, "type") {
				dt := d.Type
				failures = oneOf(failures, fmt.Sprintf("participantDefaults[%d].type", i), &dt, participantTypes)
			}
			if d.AudioChannelIndex < 1 && !unknownAt(req.NewInputs, "participantDefaults", i, "audioChannelIndex") {
				fail(fmt.Sprintf("participantDefaults[%d].audioChannelIndex", i), "audioChannelIndex is one-based and must be at least 1")
			}
		}
	}
	return infer.CheckResponse[VoiceTranscriptionConfigurationArgs]{Inputs: args, Failures: failures}, nil
}

type transcriptionConfigurationBody struct {
	Configuration transcriptionConfigurationJSON `json:"configuration"`
	UniqueName    *string                        `json:"unique_name,omitempty"`
	Description   *string                        `json:"description,omitempty"`
}

type transcriptionConfigurationJSON struct {
	ConfigurationType           string                            `json:"configurationType"`
	TranscriptionEngine         string                            `json:"transcriptionEngine"`
	SpeechModel                 *string                           `json:"speechModel,omitempty"`
	Language                    string                            `json:"language"`
	TranscriptionStatusCallback *TranscriptionStatusCallback      `json:"transcriptionStatusCallback,omitempty"`
	ConversationConfigurationId *string                           `json:"conversationConfigurationId,omitempty"`
	ParticipantDefaults         []TranscriptionParticipantDefault `json:"participantDefaults"`
}

func (a VoiceTranscriptionConfigurationArgs) body() transcriptionConfigurationBody {
	defaults := a.ParticipantDefaults
	if defaults == nil {
		defaults = []TranscriptionParticipantDefault{}
	}
	return transcriptionConfigurationBody{UniqueName: a.UniqueName, Description: a.Description,
		Configuration: transcriptionConfigurationJSON{ConfigurationType: "Transcription", TranscriptionEngine: a.TranscriptionEngine,
			SpeechModel: a.SpeechModel, Language: a.Language, TranscriptionStatusCallback: a.TranscriptionStatusCallback,
			ConversationConfigurationId: a.ConversationConfigurationId, ParticipantDefaults: defaults}}
}

type transcriptionConfigurationResponse struct {
	ID            string                         `json:"id"`
	AccountSid    string                         `json:"account_sid"`
	UniqueName    *string                        `json:"unique_name"`
	Description   *string                        `json:"description"`
	DateCreated   string                         `json:"date_created"`
	DateUpdated   string                         `json:"date_updated"`
	Configuration transcriptionConfigurationJSON `json:"configuration"`
}

func decodeTranscriptionConfiguration(o obj) (transcriptionConfigurationResponse, error) {
	var out transcriptionConfigurationResponse
	if err := decodeJSONObject(o, &out); err != nil {
		return out, fmt.Errorf("decode transcription configuration: %w", err)
	}
	if out.ID == "" {
		return out, fmt.Errorf("decode transcription configuration: no id in the response")
	}
	return out, nil
}

func transcriptionStateFrom(in VoiceTranscriptionConfigurationArgs, resp transcriptionConfigurationResponse) VoiceTranscriptionConfigurationState {
	return VoiceTranscriptionConfigurationState{VoiceTranscriptionConfigurationArgs: in, ConfigurationId: resp.ID, AccountSid: resp.AccountSid,
		DateCreated: resp.DateCreated, DateUpdated: resp.DateUpdated}
}

func adoptTranscription(in VoiceTranscriptionConfigurationArgs, resp transcriptionConfigurationResponse) VoiceTranscriptionConfigurationArgs {
	out := in
	cfg := resp.Configuration
	out.UniqueName = adoptOptional(in.UniqueName, resp.UniqueName)
	out.Description = adoptOptional(in.Description, resp.Description)
	out.TranscriptionEngine = cfg.TranscriptionEngine
	out.Language = cfg.Language
	out.SpeechModel = adoptOptional(in.SpeechModel, cfg.SpeechModel)
	out.ConversationConfigurationId = adoptOptional(in.ConversationConfigurationId, cfg.ConversationConfigurationId)
	out.TranscriptionStatusCallback = nil
	if !cfg.TranscriptionStatusCallback.empty() {
		cb := *cfg.TranscriptionStatusCallback
		// POST is the only method and the default, so it is not adopted unless the program set it.
		if in.TranscriptionStatusCallback != nil && in.TranscriptionStatusCallback.Method == nil && cb.Method != nil && *cb.Method == "POST" {
			cb.Method = nil
		}
		out.TranscriptionStatusCallback = &cb
	}
	out.ParticipantDefaults = cfg.ParticipantDefaults
	return out
}

func (r *VoiceTranscriptionConfiguration) Create(ctx context.Context, req infer.CreateRequest[VoiceTranscriptionConfigurationArgs]) (infer.CreateResponse[VoiceTranscriptionConfigurationState], error) {
	a := req.Inputs
	if req.DryRun {
		return infer.CreateResponse[VoiceTranscriptionConfigurationState]{Output: VoiceTranscriptionConfigurationState{VoiceTranscriptionConfigurationArgs: a}}, nil
	}
	c, err := clientFor(ctx)
	if err != nil {
		return infer.CreateResponse[VoiceTranscriptionConfigurationState]{}, err
	}
	o, err := c.doJSON(ctx, "POST", voiceProduct, transcriptionConfigPath(""), nil, a.body())
	if err != nil {
		return infer.CreateResponse[VoiceTranscriptionConfigurationState]{}, fmt.Errorf("create transcription configuration: %w", err)
	}
	resp, err := decodeTranscriptionConfiguration(o)
	if err != nil {
		return infer.CreateResponse[VoiceTranscriptionConfigurationState]{}, err
	}
	return infer.CreateResponse[VoiceTranscriptionConfigurationState]{ID: resp.ID, Output: transcriptionStateFrom(a, resp)}, nil
}

func (r *VoiceTranscriptionConfiguration) Read(ctx context.Context, req infer.ReadRequest[VoiceTranscriptionConfigurationArgs, VoiceTranscriptionConfigurationState]) (infer.ReadResponse[VoiceTranscriptionConfigurationArgs, VoiceTranscriptionConfigurationState], error) {
	type resp = infer.ReadResponse[VoiceTranscriptionConfigurationArgs, VoiceTranscriptionConfigurationState]
	c, err := clientFor(ctx)
	if err != nil {
		return resp{}, err
	}
	o, err := c.doJSON(ctx, "GET", voiceProduct, transcriptionConfigPath(req.ID), nil, nil)
	if err != nil {
		if isNotFound(err) {
			if err := notFoundOnImport("transcription configuration", req.ID, req.Inputs, req.State); err != nil {
				return resp{}, err
			}
			return resp{}, nil
		}
		return resp{}, fmt.Errorf("read transcription configuration %s: %w", req.ID, err)
	}
	got, err := decodeTranscriptionConfiguration(o)
	if err != nil {
		return resp{}, err
	}
	in := adoptTranscription(req.Inputs, got)
	return resp{ID: req.ID, Inputs: in, State: transcriptionStateFrom(in, got)}, nil
}

func (r *VoiceTranscriptionConfiguration) Update(ctx context.Context, req infer.UpdateRequest[VoiceTranscriptionConfigurationArgs, VoiceTranscriptionConfigurationState]) (infer.UpdateResponse[VoiceTranscriptionConfigurationState], error) {
	a := req.Inputs
	st := req.State
	st.VoiceTranscriptionConfigurationArgs = a
	if req.DryRun {
		return infer.UpdateResponse[VoiceTranscriptionConfigurationState]{Output: st}, nil
	}
	c, err := clientFor(ctx)
	if err != nil {
		return infer.UpdateResponse[VoiceTranscriptionConfigurationState]{}, err
	}
	o, err := c.doJSON(ctx, "PUT", voiceProduct, transcriptionConfigPath(req.ID), nil, a.body())
	if err != nil {
		return infer.UpdateResponse[VoiceTranscriptionConfigurationState]{}, fmt.Errorf("update transcription configuration %s: %w", req.ID, err)
	}
	resp, err := decodeTranscriptionConfiguration(o)
	if err != nil {
		return infer.UpdateResponse[VoiceTranscriptionConfigurationState]{}, err
	}
	return infer.UpdateResponse[VoiceTranscriptionConfigurationState]{Output: transcriptionStateFrom(a, resp)}, nil
}

func (r *VoiceTranscriptionConfiguration) Delete(ctx context.Context, req infer.DeleteRequest[VoiceTranscriptionConfigurationState]) (infer.DeleteResponse, error) {
	c, err := clientFor(ctx)
	if err != nil {
		return infer.DeleteResponse{}, err
	}
	if _, err := c.doJSON(ctx, "DELETE", voiceProduct, transcriptionConfigPath(req.ID), nil, nil); err != nil && !isNotFound(err) {
		return infer.DeleteResponse{}, fmt.Errorf("delete transcription configuration %s: %w", req.ID, err)
	}
	return infer.DeleteResponse{}, nil
}
