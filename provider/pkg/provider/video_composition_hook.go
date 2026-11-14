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

// Twilio defaults of a composition hook that an import leaves unset.
const (
	videoDefaultResolution     = "640x480"
	videoDefaultCallbackMethod = "POST"
)

// VideoCompositionHook manages a Programmable Video composition hook: the standing rule that turns
// every completed group room into a composition.
type VideoCompositionHook struct{}

// VideoCompositionHookArgs are the inputs of a VideoCompositionHook.
type VideoCompositionHookArgs struct {
	FriendlyName         string    `pulumi:"friendlyName"`
	Enabled              *bool     `pulumi:"enabled,optional"`
	VideoLayout          *string   `pulumi:"videoLayout,optional"`
	AudioSources         *[]string `pulumi:"audioSources,optional"`
	AudioSourcesExcluded *[]string `pulumi:"audioSourcesExcluded,optional"`
	Resolution           *string   `pulumi:"resolution,optional"`
	Format               *string   `pulumi:"format,optional"`
	Trim                 *bool     `pulumi:"trim,optional"`
	StatusCallback       *string   `pulumi:"statusCallback,optional"`
	StatusCallbackMethod *string   `pulumi:"statusCallbackMethod,optional"`
}

func (a *VideoCompositionHookArgs) Annotate(an infer.Annotator) {
	an.Describe(&a.FriendlyName, "Name of the hook, up to 100 characters, unique within the account.")
	an.Describe(&a.Enabled, "Whether the hook is active: when true it is triggered for every completed group room in the account. Removing it leaves the setting unchanged.")
	an.Describe(&a.VideoLayout, "The video layout as a JSON object of regions (see Twilio's Specifying Video Layouts). Key order, formatting and the defaults Twilio fills in for a region (an empty cells_excluded or video_sources_excluded, reuse show_oldest, x_pos, y_pos and z_pos 0) are not significant. Removing it leaves the layout unchanged.")
	an.Describe(&a.AudioSources, "Track names (a * matches any characters) whose audio is merged into the compositions. Order is not significant. Removing it leaves the setting unchanged.")
	an.Describe(&a.AudioSourcesExcluded, "Track names to leave out of the merged audio. Order is not significant. Removing it leaves the setting unchanged.")
	an.Describe(&a.Resolution, "Size of the composed video as {width}x{height} pixels, each from 16 to 1280 and at most 921,600 pixels in all (Twilio default 640x480).")
	an.Describe(&a.Format, "Container format of the compositions, mp4 or webm. Removing it leaves the setting unchanged.")
	an.Describe(&a.Trim, "Whether to clip the stretches with no active media from the compositions (Twilio default true).")
	an.Describe(&a.StatusCallback, "Absolute URL Twilio calls on every composition event. Removing it leaves the setting unchanged.")
	an.Describe(&a.StatusCallbackMethod, "HTTP method of the status callback, GET or POST (Twilio default POST).")
}

// VideoCompositionHookState is the persisted state of a VideoCompositionHook.
type VideoCompositionHookState struct {
	VideoCompositionHookArgs

	Sid         string `pulumi:"sid"`
	DateCreated string `pulumi:"dateCreated"`
	DateUpdated string `pulumi:"dateUpdated"`
}

func (s *VideoCompositionHookState) Annotate(a infer.Annotator) {
	a.Describe(&s.Sid, "The hook SID (HK...).")
	a.Describe(&s.DateCreated, "When the hook was created (ISO 8601).")
	a.Describe(&s.DateUpdated, "When the hook was last updated (ISO 8601).")
}

func (r *VideoCompositionHook) Annotate(a infer.Annotator) {
	a.Describe(&r, "A Programmable Video composition hook. The resource ID is the hook SID. Optional settings that are set are sent and kept in step with Twilio; one that is removed leaves the setting as it is on Twilio.")
}

var _ infer.CustomResource[VideoCompositionHookArgs, VideoCompositionHookState] = (*VideoCompositionHook)(nil)
var _ infer.CustomCheck[VideoCompositionHookArgs] = (*VideoCompositionHook)(nil)
var _ infer.CustomDiff[VideoCompositionHookArgs, VideoCompositionHookState] = (*VideoCompositionHook)(nil)
var _ infer.CustomRead[VideoCompositionHookArgs, VideoCompositionHookState] = (*VideoCompositionHook)(nil)
var _ infer.CustomUpdate[VideoCompositionHookArgs, VideoCompositionHookState] = (*VideoCompositionHook)(nil)
var _ infer.CustomDelete[VideoCompositionHookState] = (*VideoCompositionHook)(nil)

func (r *VideoCompositionHook) Check(ctx context.Context, req infer.CheckRequest) (infer.CheckResponse[VideoCompositionHookArgs], error) {
	args, failures, err := infer.DefaultCheck[VideoCompositionHookArgs](ctx, req.NewInputs)
	if err != nil || len(failures) > 0 {
		return infer.CheckResponse[VideoCompositionHookArgs]{Inputs: args, Failures: failures}, err
	}
	c := &checker{unk: unknowns(req.NewInputs)}
	c.required("friendlyName", args.FriendlyName)
	c.maxLen("friendlyName", args.FriendlyName, 100)
	if v := args.VideoLayout; v != nil && !c.unk("videoLayout") && !jsonObject(*v) {
		c.fail("videoLayout", "videoLayout must be a JSON object")
	}
	if v := args.Resolution; v != nil && !c.unk("resolution") && !videoResolutionOK(*v) {
		c.fail("resolution", "resolution must be {width}x{height} with each side from 16 to 1280 and at most 921600 pixels in all")
	}
	if v := args.Format; v != nil {
		c.oneOf("format", *v, []string{"mp4", "webm"})
	}
	if v := args.StatusCallbackMethod; v != nil {
		c.oneOf("statusCallbackMethod", *v, []string{"GET", "POST"})
	}
	return infer.CheckResponse[VideoCompositionHookArgs]{Inputs: args, Failures: c.failures}, nil
}

func (r *VideoCompositionHook) Diff(_ context.Context, req infer.DiffRequest[VideoCompositionHookArgs, VideoCompositionHookState]) (infer.DiffResponse, error) {
	o, n := req.State.VideoCompositionHookArgs, req.Inputs
	d := trDiffSet{}
	d.requiredStr("friendlyName", o.FriendlyName, n.FriendlyName)
	d.boolean("enabled", o.Enabled, n.Enabled)
	d.videoLayoutDoc("videoLayout", o.VideoLayout, n.VideoLayout)
	d.videoListDiff("audioSources", o.AudioSources, n.AudioSources)
	d.videoListDiff("audioSourcesExcluded", o.AudioSourcesExcluded, n.AudioSourcesExcluded)
	d.strDef(videoCompositionHookDefaults, "resolution", o.Resolution, n.Resolution)
	d.str("format", o.Format, n.Format)
	d.boolDef(videoCompositionHookDefaults, "trim", o.Trim, n.Trim)
	d.strDef(videoCompositionHookDefaults, "statusCallback", o.StatusCallback, n.StatusCallback)
	d.strDef(videoCompositionHookDefaults, "statusCallbackMethod", o.StatusCallbackMethod, n.StatusCallbackMethod)
	return d.response(), nil
}

func (a VideoCompositionHookArgs) form() url.Values {
	form := url.Values{}
	form.Set("FriendlyName", a.FriendlyName)
	setB(form, "Enabled", a.Enabled)
	setS(form, "VideoLayout", a.VideoLayout)
	videoAddList(form, "AudioSources", a.AudioSources)
	videoAddList(form, "AudioSourcesExcluded", a.AudioSourcesExcluded)
	setS(form, "Resolution", a.Resolution)
	setS(form, "Format", a.Format)
	setB(form, "Trim", a.Trim)
	setS(form, "StatusCallback", a.StatusCallback)
	setS(form, "StatusCallbackMethod", a.StatusCallbackMethod)
	return form
}

func (r *VideoCompositionHook) Create(ctx context.Context, req infer.CreateRequest[VideoCompositionHookArgs]) (infer.CreateResponse[VideoCompositionHookState], error) {
	type resp = infer.CreateResponse[VideoCompositionHookState]
	a := req.Inputs
	if req.DryRun {
		return resp{Output: VideoCompositionHookState{VideoCompositionHookArgs: a}}, nil
	}
	c, err := clientFor(ctx)
	if err != nil {
		return resp{}, err
	}
	o, err := c.do(ctx, "POST", videoProduct, videoPath("CompositionHooks", ""), nil, a.form())
	if err != nil {
		return resp{}, fmt.Errorf("create Video composition hook: %w", err)
	}
	st := VideoCompositionHookState{VideoCompositionHookArgs: a}
	st.Sid, st.DateCreated, st.DateUpdated = trSid(o)
	return resp{ID: st.Sid, Output: st}, nil
}

// videoAdoptBool is the refreshed value of an optional flag whose Twilio default is not documented:
// a program's value follows Twilio, and an import adopts whatever Twilio reports.
func videoAdoptBool(in, api *bool, imported bool) *bool {
	switch {
	case imported:
		return api
	case in == nil, api == nil:
		return in
	}
	return api
}

func (r *VideoCompositionHook) Read(ctx context.Context, req infer.ReadRequest[VideoCompositionHookArgs, VideoCompositionHookState]) (infer.ReadResponse[VideoCompositionHookArgs, VideoCompositionHookState], error) {
	type resp = infer.ReadResponse[VideoCompositionHookArgs, VideoCompositionHookState]
	c, err := clientFor(ctx)
	if err != nil {
		return resp{}, err
	}
	o, err := c.do(ctx, "GET", videoProduct, videoPath("CompositionHooks", req.ID), nil, nil)
	if err != nil {
		if isNotFound(err) {
			if err := notFoundOnImport("Video composition hook", req.ID, req.Inputs, req.State); err != nil {
				return resp{}, err
			}
			return resp{}, nil
		}
		return resp{}, fmt.Errorf("read Video composition hook %s: %w", req.ID, err)
	}
	in := req.Inputs
	// An import arrives with no prior state; the program's own settings are the only record of what it manages.
	imported := req.State.Sid == ""
	in.FriendlyName = o.str("friendly_name")
	in.Enabled = videoAdoptBool(in.Enabled, o.optBool("enabled"), imported)
	in.VideoLayout = videoLayoutRead(in.VideoLayout, videoLayoutJSON(o, "video_layout"), imported)
	in.AudioSources = cvList(in.AudioSources, o.optList("audio_sources"), imported)
	in.AudioSourcesExcluded = cvList(in.AudioSourcesExcluded, o.optList("audio_sources_excluded"), imported)
	in.Resolution = trStr(in.Resolution, o.optStr("resolution"), videoCompositionHookDefaults.str("resolution"), imported)
	in.Format = videoAdoptStr(in.Format, o.optStr("format"), imported)
	in.Trim = trBool(in.Trim, o.optBool("trim"), videoCompositionHookDefaults.boolean("trim"), imported)
	in.StatusCallback = trStr(in.StatusCallback, o.optStr("status_callback"), videoCompositionHookDefaults.str("statusCallback"), imported)
	in.StatusCallbackMethod = trStr(in.StatusCallbackMethod, o.optStr("status_callback_method"), videoCompositionHookDefaults.str("statusCallbackMethod"), imported)
	st := VideoCompositionHookState{VideoCompositionHookArgs: in}
	st.Sid, st.DateCreated, st.DateUpdated = trSid(o)
	return resp{ID: req.ID, Inputs: in, State: st}, nil
}

// videoAdoptStr is videoAdoptBool for a string.
func videoAdoptStr(in, api *string, imported bool) *string {
	api = nonEmpty(api)
	switch {
	case imported:
		return api
	case in == nil, api == nil:
		return in
	}
	return api
}

func (r *VideoCompositionHook) Update(ctx context.Context, req infer.UpdateRequest[VideoCompositionHookArgs, VideoCompositionHookState]) (infer.UpdateResponse[VideoCompositionHookState], error) {
	type resp = infer.UpdateResponse[VideoCompositionHookState]
	st := req.State
	st.VideoCompositionHookArgs = req.Inputs
	if req.DryRun {
		return resp{Output: st}, nil
	}
	c, err := clientFor(ctx)
	if err != nil {
		return resp{}, err
	}
	o, err := c.do(ctx, "POST", videoProduct, videoPath("CompositionHooks", req.ID), nil, req.Inputs.form())
	if err != nil {
		return resp{}, fmt.Errorf("update Video composition hook %s: %w", req.ID, err)
	}
	st.DateUpdated = o.str("date_updated")
	return resp{Output: st}, nil
}

func (r *VideoCompositionHook) Delete(ctx context.Context, req infer.DeleteRequest[VideoCompositionHookState]) (infer.DeleteResponse, error) {
	c, err := clientFor(ctx)
	if err != nil {
		return infer.DeleteResponse{}, err
	}
	if _, err := c.do(ctx, "DELETE", videoProduct, videoPath("CompositionHooks", req.ID), nil, nil); err != nil && !isNotFound(err) {
		return infer.DeleteResponse{}, fmt.Errorf("delete Video composition hook %s: %w", req.ID, err)
	}
	return infer.DeleteResponse{}, nil
}

// videoCompositionHookDefaults are the Twilio defaults of this resource, shared by Read (import adoption) and Diff.
var videoCompositionHookDefaults = trDefaults{
	"resolution":           videoDefaultResolution,
	"trim":                 true,
	"statusCallback":       "",
	"statusCallbackMethod": videoDefaultCallbackMethod,
}
