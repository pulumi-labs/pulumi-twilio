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

// VideoRecordingSettings is the account-wide setting that decides where Programmable Video stores
// recordings: Twilio's cloud or an AWS S3 bucket of the account, optionally encrypted. It is a
// singleton Twilio never creates or deletes (POST /v1/RecordingSettings/Default replaces it), so
// Destroy forgets it and leaves the settings on Twilio as they are.

// videoRecordingSettingsID is the import ID of the singleton.
const videoRecordingSettingsID = "default"

// VideoRecordingSettings manages the account-wide Video recordings storage settings.
type VideoRecordingSettings struct{}

// VideoRecordingSettingsArgs are the inputs of VideoRecordingSettings.
type VideoRecordingSettingsArgs struct {
	FriendlyName      string  `pulumi:"friendlyName"`
	AwsCredentialsSid *string `pulumi:"awsCredentialsSid,optional"`
	AwsS3Url          *string `pulumi:"awsS3Url,optional"`
	AwsStorageEnabled *bool   `pulumi:"awsStorageEnabled,optional"`
	EncryptionKeySid  *string `pulumi:"encryptionKeySid,optional"`
	EncryptionEnabled *bool   `pulumi:"encryptionEnabled,optional"`
}

func (a *VideoRecordingSettingsArgs) Annotate(an infer.Annotator) {
	an.Describe(&a.FriendlyName, "Description of the settings shown in the Twilio console.")
	an.Describe(&a.AwsCredentialsSid, "SID (CR...) of the stored AWS credential Twilio writes to the bucket with. It is an identifier, not the credential. Removing it leaves the setting unchanged.")
	an.Describe(&a.AwsS3Url, "URL of the S3 bucket and path where recordings are stored, for example https://my-bucket.s3.amazonaws.com/recordings. Twilio only supports DNS-compliant bucket URLs. Removing it leaves the setting unchanged.")
	an.Describe(&a.AwsStorageEnabled, "Whether all recordings are written to awsS3Url; when false they are stored in Twilio's cloud (Twilio default false).")
	an.Describe(&a.EncryptionKeySid, "SID (CR...) of the public key used to encrypt recordings. It is an identifier, not the key. Removing it leaves the setting unchanged.")
	an.Describe(&a.EncryptionEnabled, "Whether all recordings are stored encrypted (Twilio default false).")
}

// VideoRecordingSettingsState is the persisted state of VideoRecordingSettings.
type VideoRecordingSettingsState struct {
	VideoRecordingSettingsArgs
}

func (r *VideoRecordingSettings) Annotate(a infer.Annotator) {
	a.Describe(&r, "The account-wide Video recordings storage settings. The resource ID is default. Twilio has no way to delete the settings, so destroying the resource forgets it and leaves them on Twilio as they are; set awsStorageEnabled and encryptionEnabled to false first to return to Twilio's own storage.")
}

var _ infer.CustomResource[VideoRecordingSettingsArgs, VideoRecordingSettingsState] = (*VideoRecordingSettings)(nil)
var _ infer.CustomCheck[VideoRecordingSettingsArgs] = (*VideoRecordingSettings)(nil)
var _ infer.CustomDiff[VideoRecordingSettingsArgs, VideoRecordingSettingsState] = (*VideoRecordingSettings)(nil)
var _ infer.CustomRead[VideoRecordingSettingsArgs, VideoRecordingSettingsState] = (*VideoRecordingSettings)(nil)
var _ infer.CustomUpdate[VideoRecordingSettingsArgs, VideoRecordingSettingsState] = (*VideoRecordingSettings)(nil)
var _ infer.CustomDelete[VideoRecordingSettingsState] = (*VideoRecordingSettings)(nil)

func (r *VideoRecordingSettings) Check(ctx context.Context, req infer.CheckRequest) (infer.CheckResponse[VideoRecordingSettingsArgs], error) {
	args, failures, err := infer.DefaultCheck[VideoRecordingSettingsArgs](ctx, req.NewInputs)
	if err != nil || len(failures) > 0 {
		return infer.CheckResponse[VideoRecordingSettingsArgs]{Inputs: args, Failures: failures}, err
	}
	c := &checker{unk: unknowns(req.NewInputs)}
	c.required("friendlyName", args.FriendlyName)
	videoCheckSettings(c, args.AwsCredentialsSid, args.AwsS3Url, args.EncryptionKeySid)
	return infer.CheckResponse[VideoRecordingSettingsArgs]{Inputs: args, Failures: c.failures}, nil
}

func (r *VideoRecordingSettings) Diff(_ context.Context, req infer.DiffRequest[VideoRecordingSettingsArgs, VideoRecordingSettingsState]) (infer.DiffResponse, error) {
	o, n := req.State.VideoRecordingSettingsArgs, req.Inputs
	d := trDiffSet{}
	d.requiredStr("friendlyName", o.FriendlyName, n.FriendlyName)
	d.strDef(videoRecordingSettingsDefaults, "awsCredentialsSid", o.AwsCredentialsSid, n.AwsCredentialsSid)
	d.strDef(videoRecordingSettingsDefaults, "awsS3Url", o.AwsS3Url, n.AwsS3Url)
	d.boolDef(videoRecordingSettingsDefaults, "awsStorageEnabled", o.AwsStorageEnabled, n.AwsStorageEnabled)
	d.strDef(videoRecordingSettingsDefaults, "encryptionKeySid", o.EncryptionKeySid, n.EncryptionKeySid)
	d.boolDef(videoRecordingSettingsDefaults, "encryptionEnabled", o.EncryptionEnabled, n.EncryptionEnabled)
	return d.response(), nil
}

func (a VideoRecordingSettingsArgs) form() url.Values {
	form := url.Values{}
	form.Set("FriendlyName", a.FriendlyName)
	setS(form, "AwsCredentialsSid", a.AwsCredentialsSid)
	setS(form, "AwsS3Url", a.AwsS3Url)
	setB(form, "AwsStorageEnabled", a.AwsStorageEnabled)
	setS(form, "EncryptionKeySid", a.EncryptionKeySid)
	setB(form, "EncryptionEnabled", a.EncryptionEnabled)
	return form
}

func (a VideoRecordingSettingsArgs) apply(ctx context.Context, verb string) error {
	c, err := clientFor(ctx)
	if err != nil {
		return err
	}
	if _, err := c.do(ctx, "POST", videoProduct, videoRecordingSettingsPath, nil, a.form()); err != nil {
		return fmt.Errorf("%s Video recordings settings: %w", verb, err)
	}
	return nil
}

func (r *VideoRecordingSettings) Create(ctx context.Context, req infer.CreateRequest[VideoRecordingSettingsArgs]) (infer.CreateResponse[VideoRecordingSettingsState], error) {
	type resp = infer.CreateResponse[VideoRecordingSettingsState]
	st := VideoRecordingSettingsState{VideoRecordingSettingsArgs: req.Inputs}
	if req.DryRun {
		return resp{Output: st}, nil
	}
	if err := req.Inputs.apply(ctx, "configure"); err != nil {
		return resp{}, err
	}
	return resp{ID: videoRecordingSettingsID, Output: st}, nil
}

func (r *VideoRecordingSettings) Read(ctx context.Context, req infer.ReadRequest[VideoRecordingSettingsArgs, VideoRecordingSettingsState]) (infer.ReadResponse[VideoRecordingSettingsArgs, VideoRecordingSettingsState], error) {
	type resp = infer.ReadResponse[VideoRecordingSettingsArgs, VideoRecordingSettingsState]
	if req.ID != videoRecordingSettingsID {
		return resp{}, fmt.Errorf("cannot import Video recordings settings %q: the import ID is %q", req.ID, videoRecordingSettingsID)
	}
	c, err := clientFor(ctx)
	if err != nil {
		return resp{}, err
	}
	o, err := c.do(ctx, "GET", videoProduct, videoRecordingSettingsPath, nil, nil)
	if err != nil {
		if isNotFound(err) {
			if err := notFoundOnImport("Video recordings settings", req.ID, req.Inputs, req.State); err != nil {
				return resp{}, err
			}
			return resp{}, nil
		}
		return resp{}, fmt.Errorf("read Video recordings settings: %w", err)
	}
	in := req.Inputs
	imported := reflect.ValueOf(req.State).IsZero()
	in.FriendlyName = o.str("friendly_name")
	in.AwsCredentialsSid = trStr(in.AwsCredentialsSid, o.optStr("aws_credentials_sid"), videoRecordingSettingsDefaults.str("awsCredentialsSid"), imported)
	in.AwsS3Url = trStr(in.AwsS3Url, o.optStr("aws_s3_url"), videoRecordingSettingsDefaults.str("awsS3Url"), imported)
	in.AwsStorageEnabled = trBool(in.AwsStorageEnabled, o.optBool("aws_storage_enabled"), videoRecordingSettingsDefaults.boolean("awsStorageEnabled"), imported)
	in.EncryptionKeySid = trStr(in.EncryptionKeySid, o.optStr("encryption_key_sid"), videoRecordingSettingsDefaults.str("encryptionKeySid"), imported)
	in.EncryptionEnabled = trBool(in.EncryptionEnabled, o.optBool("encryption_enabled"), videoRecordingSettingsDefaults.boolean("encryptionEnabled"), imported)
	return resp{ID: req.ID, Inputs: in, State: VideoRecordingSettingsState{VideoRecordingSettingsArgs: in}}, nil
}

func (r *VideoRecordingSettings) Update(ctx context.Context, req infer.UpdateRequest[VideoRecordingSettingsArgs, VideoRecordingSettingsState]) (infer.UpdateResponse[VideoRecordingSettingsState], error) {
	st := VideoRecordingSettingsState{VideoRecordingSettingsArgs: req.Inputs}
	if req.DryRun {
		return infer.UpdateResponse[VideoRecordingSettingsState]{Output: st}, nil
	}
	if err := req.Inputs.apply(ctx, "update"); err != nil {
		return infer.UpdateResponse[VideoRecordingSettingsState]{}, err
	}
	return infer.UpdateResponse[VideoRecordingSettingsState]{Output: st}, nil
}

func (r *VideoRecordingSettings) Delete(_ context.Context, _ infer.DeleteRequest[VideoRecordingSettingsState]) (infer.DeleteResponse, error) {
	return infer.DeleteResponse{}, nil
}

// ------------------------------------------------------------ getVideoRecordingSettings

// GetVideoRecordingSettings looks up the account-wide Video recordings storage settings.
type GetVideoRecordingSettings struct{}

// GetVideoRecordingSettingsArgs are the lookup arguments; there are none.
type GetVideoRecordingSettingsArgs struct{}

// GetVideoRecordingSettingsResult is the looked-up settings.
type GetVideoRecordingSettingsResult struct {
	FriendlyName      *string `pulumi:"friendlyName,optional"`
	AwsCredentialsSid *string `pulumi:"awsCredentialsSid,optional"`
	AwsS3Url          *string `pulumi:"awsS3Url,optional"`
	AwsStorageEnabled *bool   `pulumi:"awsStorageEnabled,optional"`
	EncryptionKeySid  *string `pulumi:"encryptionKeySid,optional"`
	EncryptionEnabled *bool   `pulumi:"encryptionEnabled,optional"`
}

func (f *GetVideoRecordingSettings) Annotate(a infer.Annotator) {
	a.Describe(&f, "Look up the account-wide Video recordings storage settings. The credential and key SIDs are identifiers, never the credential or key.")
}

func (f *GetVideoRecordingSettings) Invoke(ctx context.Context, _ infer.FunctionRequest[GetVideoRecordingSettingsArgs]) (infer.FunctionResponse[GetVideoRecordingSettingsResult], error) {
	const fn = "getVideoRecordingSettings"
	type resp = infer.FunctionResponse[GetVideoRecordingSettingsResult]
	c, err := clientFor(ctx)
	if err != nil {
		return resp{}, err
	}
	o, err := c.do(ctx, "GET", videoProduct, videoRecordingSettingsPath, nil, nil)
	if err != nil {
		return resp{}, lookupFailure(fn, "the Video recordings settings", err)
	}
	return resp{Output: GetVideoRecordingSettingsResult{
		FriendlyName:      nonEmpty(o.optStr("friendly_name")),
		AwsCredentialsSid: nonEmpty(o.optStr("aws_credentials_sid")),
		AwsS3Url:          nonEmpty(o.optStr("aws_s3_url")),
		AwsStorageEnabled: o.optBool("aws_storage_enabled"),
		EncryptionKeySid:  nonEmpty(o.optStr("encryption_key_sid")),
		EncryptionEnabled: o.optBool("encryption_enabled"),
	}}, nil
}

// videoRecordingSettingsDefaults are the Twilio defaults of this resource, shared by Read (import adoption) and Diff.
var videoRecordingSettingsDefaults = trDefaults{
	"awsCredentialsSid": "",
	"awsS3Url":          "",
	"awsStorageEnabled": false,
	"encryptionKeySid":  "",
	"encryptionEnabled": false,
}
