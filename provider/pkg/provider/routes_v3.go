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
	"reflect"
	"strings"

	"github.com/pulumi/pulumi-go-provider/infer"
)

// RoutesPhoneNumber pins the inbound voice and messaging processing regions of an existing phone number.
type RoutesPhoneNumber struct{}

// RoutesPhoneNumberArgs are the inputs. Twilio cannot unset a setting, so removing one from the program leaves the current value in place.
type RoutesPhoneNumberArgs struct {
	PhoneNumber     string  `pulumi:"phoneNumber" provider:"replaceOnChanges"`
	VoiceRegion     *string `pulumi:"voiceRegion,optional"`
	MessagingRegion *string `pulumi:"messagingRegion,optional"`
	FriendlyName    *string `pulumi:"friendlyName,optional"`
}

func (a *RoutesPhoneNumberArgs) Annotate(an infer.Annotator) {
	an.Describe(&a.PhoneNumber, "The phone number in E.164 format whose inbound processing regions are set. The number must already exist in Twilio; changing it replaces the resource.")
	an.Describe(&a.VoiceRegion, "Region that processes inbound voice traffic for the object (for example `us1`, `ie1`, `au1`). Twilio keeps the current region when unset.")
	an.Describe(&a.MessagingRegion, "Region that processes inbound messaging traffic for the object (for example `us1`, `ie1`, `au1`). Twilio keeps the current region when unset.")
	an.Describe(&a.FriendlyName, "Friendly name of the routing setting. Twilio keeps the current name when unset.")
}

// RoutesPhoneNumberState is the persisted state.
type RoutesPhoneNumberState struct {
	RoutesPhoneNumberArgs

	// AccountSid is the SID of the account that owns the object.
	AccountSid string `pulumi:"accountSid"`

	// Url absolute URL of the Routes resource.
	Url string `pulumi:"url"`

	// DateCreated when the routing setting was created (ISO 8601).
	DateCreated string `pulumi:"dateCreated"`

	// DateUpdated when the routing setting was last updated (ISO 8601).
	DateUpdated string `pulumi:"dateUpdated"`
}

func (s *RoutesPhoneNumberState) Annotate(a infer.Annotator) {
	a.Describe(&s.AccountSid, "SID of the account that owns the object.")
	a.Describe(&s.Url, "Absolute URL of the Routes resource.")
	a.Describe(&s.DateCreated, "When the routing setting was created (ISO 8601).")
	a.Describe(&s.DateUpdated, "When the routing setting was last updated (ISO 8601).")
}

func (r *RoutesPhoneNumber) Annotate(a infer.Annotator) {
	a.Describe(&r, "Sets the inbound voice and messaging processing regions of an existing Twilio phone number (POST /v3/PhoneNumbers/{phoneNumber}). "+
		"Twilio documents no call that removes the setting, so destroy only forgets it and the object keeps its current region. "+
		"The object must already exist and belong to the account. Import with the phone number in E.164 format. "+
		"The Routes API is unverified against a live account.")
}

var _ infer.CustomResource[RoutesPhoneNumberArgs, RoutesPhoneNumberState] = (*RoutesPhoneNumber)(nil)
var _ infer.CustomRead[RoutesPhoneNumberArgs, RoutesPhoneNumberState] = (*RoutesPhoneNumber)(nil)
var _ infer.CustomUpdate[RoutesPhoneNumberArgs, RoutesPhoneNumberState] = (*RoutesPhoneNumber)(nil)
var _ infer.CustomDelete[RoutesPhoneNumberState] = (*RoutesPhoneNumber)(nil)

func (a RoutesPhoneNumberArgs) settings() routesSettings {
	return routesSettings{VoiceRegion: a.VoiceRegion, MessagingRegion: a.MessagingRegion, FriendlyName: a.FriendlyName}
}

func (a RoutesPhoneNumberArgs) keys() []string { return []string{a.PhoneNumber} }

func (a RoutesPhoneNumberArgs) path() string { return routesPath("v3", "PhoneNumbers", a.keys()...) }

func (a RoutesPhoneNumberArgs) what() string { return "phone number " + strings.Join(a.keys(), "/") }

func (a RoutesPhoneNumberArgs) validate() error {
	for _, k := range a.keys() {
		if k == "" {
			return fmt.Errorf("RoutesPhoneNumber: phoneNumber must not be empty")
		}
	}
	return nil
}

func (a RoutesPhoneNumberArgs) withSettings(s routesSettings) RoutesPhoneNumberArgs {
	a.VoiceRegion = s.VoiceRegion
	a.MessagingRegion = s.MessagingRegion
	a.FriendlyName = s.FriendlyName
	return a
}

func (a RoutesPhoneNumberArgs) state(rm routesRemote) RoutesPhoneNumberState {
	return RoutesPhoneNumberState{RoutesPhoneNumberArgs: a, AccountSid: rm.accountSid, Url: rm.url, DateCreated: rm.dateCreated, DateUpdated: rm.dateUpdated}
}

func (r *RoutesPhoneNumber) Create(ctx context.Context, req infer.CreateRequest[RoutesPhoneNumberArgs]) (infer.CreateResponse[RoutesPhoneNumberState], error) {
	type resp = infer.CreateResponse[RoutesPhoneNumberState]
	a := req.Inputs
	if req.DryRun {
		return resp{Output: a.state(routesRemote{})}, nil
	}
	if err := a.validate(); err != nil {
		return resp{}, err
	}
	c, err := clientFor(ctx)
	if err != nil {
		return resp{}, err
	}
	rm, err := routesWrite(ctx, c, false, a.path(), a.what(), a.settings())
	if err != nil {
		return resp{}, err
	}
	return resp{ID: strings.Join(a.keys(), "/"), Output: a.state(rm)}, nil
}

func (r *RoutesPhoneNumber) Read(ctx context.Context, req infer.ReadRequest[RoutesPhoneNumberArgs, RoutesPhoneNumberState]) (infer.ReadResponse[RoutesPhoneNumberArgs, RoutesPhoneNumberState], error) {
	type resp = infer.ReadResponse[RoutesPhoneNumberArgs, RoutesPhoneNumberState]
	keys, err := routesKeys("RoutesPhoneNumber", req.ID, 1)
	if err != nil {
		return resp{}, err
	}
	c, err := clientFor(ctx)
	if err != nil {
		return resp{}, err
	}
	in := req.Inputs
	in.PhoneNumber = keys[0]
	rm, err := routesRead(ctx, c, false, in.path(), in.what())
	if err != nil {
		if isNotFound(err) {
			if err := notFoundOnImport("routing setting of phone number", req.ID, req.Inputs, req.State); err != nil {
				return resp{}, err
			}
			return resp{}, nil
		}
		return resp{}, err
	}
	imported := reflect.ValueOf(req.Inputs).IsZero() && reflect.ValueOf(req.State).IsZero()
	in = in.withSettings(routesAdopt(in.settings(), rm.routesSettings, imported))
	return resp{ID: req.ID, Inputs: in, State: in.state(rm)}, nil
}

func (r *RoutesPhoneNumber) Update(ctx context.Context, req infer.UpdateRequest[RoutesPhoneNumberArgs, RoutesPhoneNumberState]) (infer.UpdateResponse[RoutesPhoneNumberState], error) {
	type resp = infer.UpdateResponse[RoutesPhoneNumberState]
	st := req.State
	st.RoutesPhoneNumberArgs = req.Inputs
	if req.DryRun {
		return resp{Output: st}, nil
	}
	c, err := clientFor(ctx)
	if err != nil {
		return resp{}, err
	}
	rm, err := routesWrite(ctx, c, false, req.Inputs.path(), req.Inputs.what(), req.Inputs.settings())
	if err != nil {
		return resp{}, err
	}
	return resp{Output: req.Inputs.state(rm)}, nil
}

// Delete only forgets the setting: Twilio documents no call that removes it.
func (r *RoutesPhoneNumber) Delete(ctx context.Context, req infer.DeleteRequest[RoutesPhoneNumberState]) (infer.DeleteResponse, error) {
	return infer.DeleteResponse{}, nil
}

// GetRoutesPhoneNumber reads the routing settings of a phone number.
type GetRoutesPhoneNumber struct{}

// GetRoutesPhoneNumberArgs are the arguments.
type GetRoutesPhoneNumberArgs struct {
	PhoneNumber string `pulumi:"phoneNumber"`
}

func (a *GetRoutesPhoneNumberArgs) Annotate(an infer.Annotator) {
	an.Describe(&a.PhoneNumber, "The phone number in E.164 format whose inbound processing regions are set. The number must already exist in Twilio; changing it replaces the resource.")
}

// GetRoutesPhoneNumberResult is the object's routing settings.
type GetRoutesPhoneNumberResult struct {
	PhoneNumber     string `pulumi:"phoneNumber"`
	VoiceRegion     string `pulumi:"voiceRegion"`
	MessagingRegion string `pulumi:"messagingRegion"`
	FriendlyName    string `pulumi:"friendlyName"`
	AccountSid      string `pulumi:"accountSid"`
	Url             string `pulumi:"url"`
	DateCreated     string `pulumi:"dateCreated"`
	DateUpdated     string `pulumi:"dateUpdated"`
}

func (a *GetRoutesPhoneNumberResult) Annotate(an infer.Annotator) {
	an.Describe(&a.PhoneNumber, "As passed in.")
	an.Describe(&a.VoiceRegion, "Region that processes inbound voice traffic for the object (for example `us1`, `ie1`, `au1`); empty when Twilio reports none.")
	an.Describe(&a.MessagingRegion, "Region that processes inbound messaging traffic for the object (for example `us1`, `ie1`, `au1`); empty when Twilio reports none.")
	an.Describe(&a.FriendlyName, "Friendly name of the routing setting; empty when Twilio reports none.")
	an.Describe(&a.AccountSid, "SID of the account that owns the object.")
	an.Describe(&a.Url, "Absolute URL of the Routes resource.")
	an.Describe(&a.DateCreated, "When the routing setting was created (ISO 8601).")
	an.Describe(&a.DateUpdated, "When the routing setting was last updated (ISO 8601).")
}

func (f *GetRoutesPhoneNumber) Annotate(a infer.Annotator) {
	a.Describe(&f, "Read the routing settings of a phone number (GET /v3/PhoneNumbers/{phoneNumber}). Fails when the object does not exist.")
}

func (f *GetRoutesPhoneNumber) Invoke(ctx context.Context, req infer.FunctionRequest[GetRoutesPhoneNumberArgs]) (infer.FunctionResponse[GetRoutesPhoneNumberResult], error) {
	type resp = infer.FunctionResponse[GetRoutesPhoneNumberResult]
	in := RoutesPhoneNumberArgs{PhoneNumber: req.Input.PhoneNumber}
	for i, name := range []string{"phoneNumber"} {
		if err := requireArg("getRoutesPhoneNumber", name, in.keys()[i]); err != nil {
			return resp{}, err
		}
	}
	c, err := clientFor(ctx)
	if err != nil {
		return resp{}, err
	}
	rm, err := routesRead(ctx, c, false, in.path(), in.what())
	if err != nil {
		return resp{}, lookupFailure("getRoutesPhoneNumber", in.what(), err)
	}
	return resp{Output: GetRoutesPhoneNumberResult{PhoneNumber: in.PhoneNumber, VoiceRegion: deref(rm.VoiceRegion), MessagingRegion: deref(rm.MessagingRegion), FriendlyName: deref(rm.FriendlyName), AccountSid: rm.accountSid, Url: rm.url, DateCreated: rm.dateCreated, DateUpdated: rm.dateUpdated}}, nil
}

// RoutesShortCode pins the inbound messaging processing region of an existing short code.
type RoutesShortCode struct{}

// RoutesShortCodeArgs are the inputs. Twilio cannot unset a setting, so removing one from the program leaves the current value in place.
type RoutesShortCodeArgs struct {
	IsoCountryCode  string  `pulumi:"isoCountryCode" provider:"replaceOnChanges"`
	ShortCode       string  `pulumi:"shortCode" provider:"replaceOnChanges"`
	MessagingRegion *string `pulumi:"messagingRegion,optional"`
	FriendlyName    *string `pulumi:"friendlyName,optional"`
}

func (a *RoutesShortCodeArgs) Annotate(an infer.Annotator) {
	an.Describe(&a.IsoCountryCode, "ISO 3166-1 alpha-2 country code of the short code (for example `US`). Changing it replaces the resource.")
	an.Describe(&a.ShortCode, "The short code itself (for example `12345`). The short code must already exist in Twilio; changing it replaces the resource.")
	an.Describe(&a.MessagingRegion, "Region that processes inbound messaging traffic for the object (for example `us1`, `ie1`, `au1`). Twilio keeps the current region when unset.")
	an.Describe(&a.FriendlyName, "Friendly name of the routing setting. Twilio keeps the current name when unset.")
}

// RoutesShortCodeState is the persisted state.
type RoutesShortCodeState struct {
	RoutesShortCodeArgs

	// AccountSid is the SID of the account that owns the object.
	AccountSid string `pulumi:"accountSid"`

	// Url absolute URL of the Routes resource.
	Url string `pulumi:"url"`

	// DateCreated when the routing setting was created (ISO 8601).
	DateCreated string `pulumi:"dateCreated"`

	// DateUpdated when the routing setting was last updated (ISO 8601).
	DateUpdated string `pulumi:"dateUpdated"`
}

func (s *RoutesShortCodeState) Annotate(a infer.Annotator) {
	a.Describe(&s.AccountSid, "SID of the account that owns the object.")
	a.Describe(&s.Url, "Absolute URL of the Routes resource.")
	a.Describe(&s.DateCreated, "When the routing setting was created (ISO 8601).")
	a.Describe(&s.DateUpdated, "When the routing setting was last updated (ISO 8601).")
}

func (r *RoutesShortCode) Annotate(a infer.Annotator) {
	a.Describe(&r, "Sets the inbound messaging processing region of an existing Twilio short code (POST /v3/ShortCodes/{isoCountryCode}/{shortCode}). "+
		"Twilio documents no call that removes the setting, so destroy only forgets it and the object keeps its current region. "+
		"The object must already exist and belong to the account. Import with `<isoCountryCode>/<shortCode>`. "+
		"The Routes API is unverified against a live account.")
}

var _ infer.CustomResource[RoutesShortCodeArgs, RoutesShortCodeState] = (*RoutesShortCode)(nil)
var _ infer.CustomRead[RoutesShortCodeArgs, RoutesShortCodeState] = (*RoutesShortCode)(nil)
var _ infer.CustomUpdate[RoutesShortCodeArgs, RoutesShortCodeState] = (*RoutesShortCode)(nil)
var _ infer.CustomDelete[RoutesShortCodeState] = (*RoutesShortCode)(nil)

func (a RoutesShortCodeArgs) settings() routesSettings {
	return routesSettings{MessagingRegion: a.MessagingRegion, FriendlyName: a.FriendlyName}
}

func (a RoutesShortCodeArgs) keys() []string { return []string{a.IsoCountryCode, a.ShortCode} }

func (a RoutesShortCodeArgs) path() string { return routesPath("v3", "ShortCodes", a.keys()...) }

func (a RoutesShortCodeArgs) what() string { return "short code " + strings.Join(a.keys(), "/") }

func (a RoutesShortCodeArgs) validate() error {
	for _, k := range a.keys() {
		if k == "" {
			return fmt.Errorf("RoutesShortCode: isoCountryCode and shortCode must not be empty")
		}
	}
	return nil
}

func (a RoutesShortCodeArgs) withSettings(s routesSettings) RoutesShortCodeArgs {
	a.MessagingRegion = s.MessagingRegion
	a.FriendlyName = s.FriendlyName
	return a
}

func (a RoutesShortCodeArgs) state(rm routesRemote) RoutesShortCodeState {
	return RoutesShortCodeState{RoutesShortCodeArgs: a, AccountSid: rm.accountSid, Url: rm.url, DateCreated: rm.dateCreated, DateUpdated: rm.dateUpdated}
}

func (r *RoutesShortCode) Create(ctx context.Context, req infer.CreateRequest[RoutesShortCodeArgs]) (infer.CreateResponse[RoutesShortCodeState], error) {
	type resp = infer.CreateResponse[RoutesShortCodeState]
	a := req.Inputs
	if req.DryRun {
		return resp{Output: a.state(routesRemote{})}, nil
	}
	if err := a.validate(); err != nil {
		return resp{}, err
	}
	c, err := clientFor(ctx)
	if err != nil {
		return resp{}, err
	}
	rm, err := routesWrite(ctx, c, false, a.path(), a.what(), a.settings())
	if err != nil {
		return resp{}, err
	}
	return resp{ID: strings.Join(a.keys(), "/"), Output: a.state(rm)}, nil
}

func (r *RoutesShortCode) Read(ctx context.Context, req infer.ReadRequest[RoutesShortCodeArgs, RoutesShortCodeState]) (infer.ReadResponse[RoutesShortCodeArgs, RoutesShortCodeState], error) {
	type resp = infer.ReadResponse[RoutesShortCodeArgs, RoutesShortCodeState]
	keys, err := routesKeys("RoutesShortCode", req.ID, 2)
	if err != nil {
		return resp{}, err
	}
	c, err := clientFor(ctx)
	if err != nil {
		return resp{}, err
	}
	in := req.Inputs
	in.IsoCountryCode = keys[0]
	in.ShortCode = keys[1]
	rm, err := routesRead(ctx, c, false, in.path(), in.what())
	if err != nil {
		if isNotFound(err) {
			if err := notFoundOnImport("routing setting of short code", req.ID, req.Inputs, req.State); err != nil {
				return resp{}, err
			}
			return resp{}, nil
		}
		return resp{}, err
	}
	imported := reflect.ValueOf(req.Inputs).IsZero() && reflect.ValueOf(req.State).IsZero()
	in = in.withSettings(routesAdopt(in.settings(), rm.routesSettings, imported))
	return resp{ID: req.ID, Inputs: in, State: in.state(rm)}, nil
}

func (r *RoutesShortCode) Update(ctx context.Context, req infer.UpdateRequest[RoutesShortCodeArgs, RoutesShortCodeState]) (infer.UpdateResponse[RoutesShortCodeState], error) {
	type resp = infer.UpdateResponse[RoutesShortCodeState]
	st := req.State
	st.RoutesShortCodeArgs = req.Inputs
	if req.DryRun {
		return resp{Output: st}, nil
	}
	c, err := clientFor(ctx)
	if err != nil {
		return resp{}, err
	}
	rm, err := routesWrite(ctx, c, false, req.Inputs.path(), req.Inputs.what(), req.Inputs.settings())
	if err != nil {
		return resp{}, err
	}
	return resp{Output: req.Inputs.state(rm)}, nil
}

// Delete only forgets the setting: Twilio documents no call that removes it.
func (r *RoutesShortCode) Delete(ctx context.Context, req infer.DeleteRequest[RoutesShortCodeState]) (infer.DeleteResponse, error) {
	return infer.DeleteResponse{}, nil
}

// GetRoutesShortCode reads the routing settings of a short code.
type GetRoutesShortCode struct{}

// GetRoutesShortCodeArgs are the arguments.
type GetRoutesShortCodeArgs struct {
	IsoCountryCode string `pulumi:"isoCountryCode"`
	ShortCode      string `pulumi:"shortCode"`
}

func (a *GetRoutesShortCodeArgs) Annotate(an infer.Annotator) {
	an.Describe(&a.IsoCountryCode, "ISO 3166-1 alpha-2 country code of the short code (for example `US`).")
	an.Describe(&a.ShortCode, "The short code itself (for example `12345`). The short code must already exist in Twilio; changing it replaces the resource.")
}

// GetRoutesShortCodeResult is the object's routing settings.
type GetRoutesShortCodeResult struct {
	IsoCountryCode  string `pulumi:"isoCountryCode"`
	ShortCode       string `pulumi:"shortCode"`
	MessagingRegion string `pulumi:"messagingRegion"`
	FriendlyName    string `pulumi:"friendlyName"`
	AccountSid      string `pulumi:"accountSid"`
	Url             string `pulumi:"url"`
	DateCreated     string `pulumi:"dateCreated"`
	DateUpdated     string `pulumi:"dateUpdated"`
}

func (a *GetRoutesShortCodeResult) Annotate(an infer.Annotator) {
	an.Describe(&a.IsoCountryCode, "As passed in.")
	an.Describe(&a.ShortCode, "As passed in.")
	an.Describe(&a.MessagingRegion, "Region that processes inbound messaging traffic for the object (for example `us1`, `ie1`, `au1`); empty when Twilio reports none.")
	an.Describe(&a.FriendlyName, "Friendly name of the routing setting; empty when Twilio reports none.")
	an.Describe(&a.AccountSid, "SID of the account that owns the object.")
	an.Describe(&a.Url, "Absolute URL of the Routes resource.")
	an.Describe(&a.DateCreated, "When the routing setting was created (ISO 8601).")
	an.Describe(&a.DateUpdated, "When the routing setting was last updated (ISO 8601).")
}

func (f *GetRoutesShortCode) Annotate(a infer.Annotator) {
	a.Describe(&f, "Read the routing settings of a short code (GET /v3/ShortCodes/{isoCountryCode}/{shortCode}). Fails when the object does not exist.")
}

func (f *GetRoutesShortCode) Invoke(ctx context.Context, req infer.FunctionRequest[GetRoutesShortCodeArgs]) (infer.FunctionResponse[GetRoutesShortCodeResult], error) {
	type resp = infer.FunctionResponse[GetRoutesShortCodeResult]
	in := RoutesShortCodeArgs{IsoCountryCode: req.Input.IsoCountryCode, ShortCode: req.Input.ShortCode}
	for i, name := range []string{"isoCountryCode", "shortCode"} {
		if err := requireArg("getRoutesShortCode", name, in.keys()[i]); err != nil {
			return resp{}, err
		}
	}
	c, err := clientFor(ctx)
	if err != nil {
		return resp{}, err
	}
	rm, err := routesRead(ctx, c, false, in.path(), in.what())
	if err != nil {
		return resp{}, lookupFailure("getRoutesShortCode", in.what(), err)
	}
	return resp{Output: GetRoutesShortCodeResult{IsoCountryCode: in.IsoCountryCode, ShortCode: in.ShortCode, MessagingRegion: deref(rm.MessagingRegion), FriendlyName: deref(rm.FriendlyName), AccountSid: rm.accountSid, Url: rm.url, DateCreated: rm.dateCreated, DateUpdated: rm.dateUpdated}}, nil
}
