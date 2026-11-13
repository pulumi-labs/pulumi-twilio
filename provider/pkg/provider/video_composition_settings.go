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
	"reflect"

	"github.com/pulumi/pulumi-go-provider/infer"
)

// VideoCompositionSettings is the account-wide setting that decides where Programmable Video stores
// compositions: Twilio's cloud or an AWS S3 bucket of the account, optionally encrypted. It is a
// singleton Twilio never creates or deletes (POST /v1/CompositionSettings/Default replaces it), so
// Destroy forgets it and leaves the settings on Twilio as they are.

// videoCompositionSettingsID is the import ID of the singleton.
const videoCompositionSettingsID = "default"

// VideoCompositionSettings manages the account-wide Video compositions storage settings.
type VideoCompositionSettings struct{}

// VideoCompositionSettingsArgs are the inputs of VideoCompositionSettings.
type VideoCompositionSettingsArgs struct {
	FriendlyName      string  `pulumi:"friendlyName"`
	AwsCredentialsSid *string `pulumi:"awsCredentialsSid,optional"`
	AwsS3Url          *string `pulumi:"awsS3Url,optional"`
	AwsStorageEnabled *bool   `pulumi:"awsStorageEnabled,optional"`
	EncryptionKeySid  *string `pulumi:"encryptionKeySid,optional"`
	EncryptionEnabled *bool   `pulumi:"encryptionEnabled,optional"`
}

func (a *VideoCompositionSettingsArgs) Annotate(an infer.Annotator) {
	an.Describe(&a.FriendlyName, "Description of the settings shown in the Twilio console.")
	an.Describe(&a.AwsCredentialsSid, "SID (CR...) of the stored AWS credential Twilio writes to the bucket with. It is an identifier, not the credential. Removing it leaves the setting unchanged.")
	an.Describe(&a.AwsS3Url, "URL of the S3 bucket and path where compositions are stored, for example https://my-bucket.s3.amazonaws.com/compositions. Twilio only supports DNS-compliant bucket URLs. Removing it leaves the setting unchanged.")
	an.Describe(&a.AwsStorageEnabled, "Whether all compositions are written to awsS3Url; when false they are stored in Twilio's cloud (Twilio default false).")
	an.Describe(&a.EncryptionKeySid, "SID (CR...) of the public key used to encrypt compositions. It is an identifier, not the key. Removing it leaves the setting unchanged.")
	an.Describe(&a.EncryptionEnabled, "Whether all compositions are stored encrypted (Twilio default false).")
}

// VideoCompositionSettingsState is the persisted state of VideoCompositionSettings.
type VideoCompositionSettingsState struct {
	VideoCompositionSettingsArgs
}

func (r *VideoCompositionSettings) Annotate(a infer.Annotator) {
	a.Describe(&r, "The account-wide Video compositions storage settings. The resource ID is default. Twilio has no way to delete the settings, so destroying the resource forgets it and leaves them on Twilio as they are; set awsStorageEnabled and encryptionEnabled to false first to return to Twilio's own storage.")
}

var _ infer.CustomResource[VideoCompositionSettingsArgs, VideoCompositionSettingsState] = (*VideoCompositionSettings)(nil)
var _ infer.CustomCheck[VideoCompositionSettingsArgs] = (*VideoCompositionSettings)(nil)
var _ infer.CustomDiff[VideoCompositionSettingsArgs, VideoCompositionSettingsState] = (*VideoCompositionSettings)(nil)
var _ infer.CustomRead[VideoCompositionSettingsArgs, VideoCompositionSettingsState] = (*VideoCompositionSettings)(nil)
var _ infer.CustomUpdate[VideoCompositionSettingsArgs, VideoCompositionSettingsState] = (*VideoCompositionSettings)(nil)
var _ infer.CustomDelete[VideoCompositionSettingsState] = (*VideoCompositionSettings)(nil)

func (r *VideoCompositionSettings) Check(ctx context.Context, req infer.CheckRequest) (infer.CheckResponse[VideoCompositionSettingsArgs], error) {
	args, failures, err := infer.DefaultCheck[VideoCompositionSettingsArgs](ctx, req.NewInputs)
	if err != nil || len(failures) > 0 {
		return infer.CheckResponse[VideoCompositionSettingsArgs]{Inputs: args, Failures: failures}, err
	}
	c := &checker{unk: unknowns(req.NewInputs)}
	c.required("friendlyName", args.FriendlyName)
	videoCheckSettings(c, args.AwsCredentialsSid, args.AwsS3Url, args.EncryptionKeySid)
	return infer.CheckResponse[VideoCompositionSettingsArgs]{Inputs: args, Failures: c.failures}, nil
}

func (r *VideoCompositionSettings) Diff(_ context.Context, req infer.DiffRequest[VideoCompositionSettingsArgs, VideoCompositionSettingsState]) (infer.DiffResponse, error) {
	o, n := req.State.VideoCompositionSettingsArgs, req.Inputs
	d := trDiffSet{}
	d.requiredStr("friendlyName", o.FriendlyName, n.FriendlyName)
	d.strDef(videoCompositionSettingsDefaults, "awsCredentialsSid", o.AwsCredentialsSid, n.AwsCredentialsSid)
	d.strDef(videoCompositionSettingsDefaults, "awsS3Url", o.AwsS3Url, n.AwsS3Url)
	d.boolDef(videoCompositionSettingsDefaults, "awsStorageEnabled", o.AwsStorageEnabled, n.AwsStorageEnabled)
	d.strDef(videoCompositionSettingsDefaults, "encryptionKeySid", o.EncryptionKeySid, n.EncryptionKeySid)
	d.boolDef(videoCompositionSettingsDefaults, "encryptionEnabled", o.EncryptionEnabled, n.EncryptionEnabled)
	return d.response(), nil
}

func (a VideoCompositionSettingsArgs) form() url.Values {
	form := url.Values{}
	form.Set("FriendlyName", a.FriendlyName)
	setS(form, "AwsCredentialsSid", a.AwsCredentialsSid)
	setS(form, "AwsS3Url", a.AwsS3Url)
	setB(form, "AwsStorageEnabled", a.AwsStorageEnabled)
	setS(form, "EncryptionKeySid", a.EncryptionKeySid)
	setB(form, "EncryptionEnabled", a.EncryptionEnabled)
	return form
}

func (a VideoCompositionSettingsArgs) apply(ctx context.Context, verb string) error {
	c, err := clientFor(ctx)
	if err != nil {
		return err
	}
	if _, err := c.do(ctx, "POST", videoProduct, videoCompositionSettingsPath, nil, a.form()); err != nil {
		return fmt.Errorf("%s Video compositions settings: %w", verb, err)
	}
	return nil
}

func (r *VideoCompositionSettings) Create(ctx context.Context, req infer.CreateRequest[VideoCompositionSettingsArgs]) (infer.CreateResponse[VideoCompositionSettingsState], error) {
	type resp = infer.CreateResponse[VideoCompositionSettingsState]
	st := VideoCompositionSettingsState{VideoCompositionSettingsArgs: req.Inputs}
	if req.DryRun {
		return resp{Output: st}, nil
	}
	if err := req.Inputs.apply(ctx, "configure"); err != nil {
		return resp{}, err
	}
	return resp{ID: videoCompositionSettingsID, Output: st}, nil
}

func (r *VideoCompositionSettings) Read(ctx context.Context, req infer.ReadRequest[VideoCompositionSettingsArgs, VideoCompositionSettingsState]) (infer.ReadResponse[VideoCompositionSettingsArgs, VideoCompositionSettingsState], error) {
	type resp = infer.ReadResponse[VideoCompositionSettingsArgs, VideoCompositionSettingsState]
	if req.ID != videoCompositionSettingsID {
		return resp{}, fmt.Errorf("cannot import Video compositions settings %q: the import ID is %q", req.ID, videoCompositionSettingsID)
	}
	c, err := clientFor(ctx)
	if err != nil {
		return resp{}, err
	}
	o, err := c.do(ctx, "GET", videoProduct, videoCompositionSettingsPath, nil, nil)
	if err != nil {
		if isNotFound(err) {
			if err := notFoundOnImport("Video compositions settings", req.ID, req.Inputs, req.State); err != nil {
				return resp{}, err
			}
			return resp{}, nil
		}
		return resp{}, fmt.Errorf("read Video compositions settings: %w", err)
	}
	in := req.Inputs
	imported := reflect.ValueOf(req.State).IsZero()
	in.FriendlyName = o.str("friendly_name")
	in.AwsCredentialsSid = trStr(in.AwsCredentialsSid, o.optStr("aws_credentials_sid"), videoCompositionSettingsDefaults.str("awsCredentialsSid"), imported)
	in.AwsS3Url = trStr(in.AwsS3Url, o.optStr("aws_s3_url"), videoCompositionSettingsDefaults.str("awsS3Url"), imported)
	in.AwsStorageEnabled = trBool(in.AwsStorageEnabled, o.optBool("aws_storage_enabled"), videoCompositionSettingsDefaults.boolean("awsStorageEnabled"), imported)
	in.EncryptionKeySid = trStr(in.EncryptionKeySid, o.optStr("encryption_key_sid"), videoCompositionSettingsDefaults.str("encryptionKeySid"), imported)
	in.EncryptionEnabled = trBool(in.EncryptionEnabled, o.optBool("encryption_enabled"), videoCompositionSettingsDefaults.boolean("encryptionEnabled"), imported)
	return resp{ID: req.ID, Inputs: in, State: VideoCompositionSettingsState{VideoCompositionSettingsArgs: in}}, nil
}

func (r *VideoCompositionSettings) Update(ctx context.Context, req infer.UpdateRequest[VideoCompositionSettingsArgs, VideoCompositionSettingsState]) (infer.UpdateResponse[VideoCompositionSettingsState], error) {
	st := VideoCompositionSettingsState{VideoCompositionSettingsArgs: req.Inputs}
	if req.DryRun {
		return infer.UpdateResponse[VideoCompositionSettingsState]{Output: st}, nil
	}
	if err := req.Inputs.apply(ctx, "update"); err != nil {
		return infer.UpdateResponse[VideoCompositionSettingsState]{}, err
	}
	return infer.UpdateResponse[VideoCompositionSettingsState]{Output: st}, nil
}

func (r *VideoCompositionSettings) Delete(_ context.Context, _ infer.DeleteRequest[VideoCompositionSettingsState]) (infer.DeleteResponse, error) {
	return infer.DeleteResponse{}, nil
}

// ------------------------------------------------------------ getVideoCompositionSettings

// GetVideoCompositionSettings looks up the account-wide Video compositions storage settings.
type GetVideoCompositionSettings struct{}

// GetVideoCompositionSettingsArgs are the lookup arguments; there are none.
type GetVideoCompositionSettingsArgs struct{}

// GetVideoCompositionSettingsResult is the looked-up settings.
type GetVideoCompositionSettingsResult struct {
	FriendlyName      *string `pulumi:"friendlyName,optional"`
	AwsCredentialsSid *string `pulumi:"awsCredentialsSid,optional"`
	AwsS3Url          *string `pulumi:"awsS3Url,optional"`
	AwsStorageEnabled *bool   `pulumi:"awsStorageEnabled,optional"`
	EncryptionKeySid  *string `pulumi:"encryptionKeySid,optional"`
	EncryptionEnabled *bool   `pulumi:"encryptionEnabled,optional"`
}

func (f *GetVideoCompositionSettings) Annotate(a infer.Annotator) {
	a.Describe(&f, "Look up the account-wide Video compositions storage settings. The credential and key SIDs are identifiers, never the credential or key.")
}

func (f *GetVideoCompositionSettings) Invoke(ctx context.Context, _ infer.FunctionRequest[GetVideoCompositionSettingsArgs]) (infer.FunctionResponse[GetVideoCompositionSettingsResult], error) {
	const fn = "getVideoCompositionSettings"
	type resp = infer.FunctionResponse[GetVideoCompositionSettingsResult]
	c, err := clientFor(ctx)
	if err != nil {
		return resp{}, err
	}
	o, err := c.do(ctx, "GET", videoProduct, videoCompositionSettingsPath, nil, nil)
	if err != nil {
		return resp{}, lookupFailure(fn, "the Video compositions settings", err)
	}
	return resp{Output: GetVideoCompositionSettingsResult{
		FriendlyName:      nonEmpty(o.optStr("friendly_name")),
		AwsCredentialsSid: nonEmpty(o.optStr("aws_credentials_sid")),
		AwsS3Url:          nonEmpty(o.optStr("aws_s3_url")),
		AwsStorageEnabled: o.optBool("aws_storage_enabled"),
		EncryptionKeySid:  nonEmpty(o.optStr("encryption_key_sid")),
		EncryptionEnabled: o.optBool("encryption_enabled"),
	}}, nil
}

// videoCompositionSettingsDefaults are the Twilio defaults of this resource, shared by Read (import adoption) and Diff.
var videoCompositionSettingsDefaults = trDefaults{
	"awsCredentialsSid": "",
	"awsS3Url":          "",
	"awsStorageEnabled": false,
	"encryptionKeySid":  "",
	"encryptionEnabled": false,
}
