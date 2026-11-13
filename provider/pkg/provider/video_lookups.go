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

// Lookups of the Video objects. Each reads one object and returns what Twilio reports, null where
// Twilio reports nothing.

// GetVideoCompositionHook looks up a Programmable Video composition hook by SID.
type GetVideoCompositionHook struct{}

// GetVideoCompositionHookArgs are the lookup arguments.
type GetVideoCompositionHookArgs struct {
	Sid string `pulumi:"sid"`
}

func (a *GetVideoCompositionHookArgs) Annotate(an infer.Annotator) {
	an.Describe(&a.Sid, "SID (HK...) of the composition hook.")
}

// GetVideoCompositionHookResult is the looked-up hook.
type GetVideoCompositionHookResult struct {
	Sid                  string    `pulumi:"sid"`
	FriendlyName         *string   `pulumi:"friendlyName,optional"`
	Enabled              *bool     `pulumi:"enabled,optional"`
	VideoLayout          *string   `pulumi:"videoLayout,optional"`
	AudioSources         *[]string `pulumi:"audioSources,optional"`
	AudioSourcesExcluded *[]string `pulumi:"audioSourcesExcluded,optional"`
	Resolution           *string   `pulumi:"resolution,optional"`
	Format               *string   `pulumi:"format,optional"`
	Trim                 *bool     `pulumi:"trim,optional"`
	StatusCallback       *string   `pulumi:"statusCallback,optional"`
	StatusCallbackMethod *string   `pulumi:"statusCallbackMethod,optional"`
	DateCreated          string    `pulumi:"dateCreated"`
	DateUpdated          string    `pulumi:"dateUpdated"`
}

func (f *GetVideoCompositionHook) Annotate(a infer.Annotator) {
	a.Describe(&f, "Look up a Programmable Video composition hook by SID. videoLayout is the layout as a JSON document.")
}

func (f *GetVideoCompositionHook) Invoke(ctx context.Context, req infer.FunctionRequest[GetVideoCompositionHookArgs]) (infer.FunctionResponse[GetVideoCompositionHookResult], error) {
	const fn = "getVideoCompositionHook"
	type resp = infer.FunctionResponse[GetVideoCompositionHookResult]
	if err := requireArg(fn, "sid", req.Input.Sid); err != nil {
		return resp{}, err
	}
	c, err := clientFor(ctx)
	if err != nil {
		return resp{}, err
	}
	o, err := c.do(ctx, "GET", videoProduct, videoPath("CompositionHooks", req.Input.Sid), nil, nil)
	if err != nil {
		return resp{}, lookupFailure(fn, "Video composition hook "+req.Input.Sid, err)
	}
	sid, created, updated := trSid(o)
	return resp{Output: GetVideoCompositionHookResult{
		Sid:                  sid,
		FriendlyName:         nonEmpty(o.optStr("friendly_name")),
		Enabled:              o.optBool("enabled"),
		VideoLayout:          videoLayoutJSON(o, "video_layout"),
		AudioSources:         o.optList("audio_sources"),
		AudioSourcesExcluded: o.optList("audio_sources_excluded"),
		Resolution:           nonEmpty(o.optStr("resolution")),
		Format:               nonEmpty(o.optStr("format")),
		Trim:                 o.optBool("trim"),
		StatusCallback:       nonEmpty(o.optStr("status_callback")),
		StatusCallbackMethod: nonEmpty(o.optStr("status_callback_method")),
		DateCreated:          created,
		DateUpdated:          updated,
	}}, nil
}
