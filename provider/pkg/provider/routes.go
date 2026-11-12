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
	"strings"

	"github.com/pulumi/pulumi-go-provider/infer"
)

// Twilio Routes (routes.twilio.com) pins the region that processes inbound voice or messaging
// traffic of one SIP domain, SIP trunk domain, phone number or short code. It is a settings API:
// POST updates the region of an object that already exists, GET reads it, and no call removes
// the setting, so destroy only forgets it. The v2 endpoints (SIP domains and trunks) take
// PascalCase form fields and answer snake_case JSON; the v3 endpoints (phone numbers, short
// codes) take camelCase form fields and answer camelCase JSON. Shapes follow twilio-oai
// spec/json/twilio_routes_v2.json and twilio_routes_v3.json (spec version 1.0.0, read
// 2026-11-11) and have not been checked against a live account. The v2 /PhoneNumbers endpoint
// is superseded by v3 and is not modelled.

const routesProduct = "routes"

// routesSettings are the settings the Routes API writes; a nil field is not sent.
type routesSettings struct {
	VoiceRegion, MessagingRegion, FriendlyName *string
}

// routesRemote is a Routes object as Twilio reports it.
type routesRemote struct {
	routesSettings
	sid, accountSid, url, dateCreated, dateUpdated string
}

func (s routesSettings) empty() bool {
	return s.VoiceRegion == nil && s.MessagingRegion == nil && s.FriendlyName == nil
}

// routesForm encodes the set settings. v2 field names are PascalCase, v3 camelCase.
func routesForm(s routesSettings, v2 bool) url.Values {
	form := url.Values{}
	name := func(camel string) string {
		if v2 {
			return strings.ToUpper(camel[:1]) + camel[1:]
		}
		return camel
	}
	if s.VoiceRegion != nil {
		form.Set(name("voiceRegion"), *s.VoiceRegion)
	}
	if s.MessagingRegion != nil {
		form.Set(name("messagingRegion"), *s.MessagingRegion)
	}
	if s.FriendlyName != nil {
		form.Set(name("friendlyName"), *s.FriendlyName)
	}
	return form
}

func routesPath(version, collection string, keys ...string) string {
	p := "/" + version + "/" + collection
	for _, k := range keys {
		p += "/" + url.PathEscape(k)
	}
	return p
}

func routesParse(o obj, v2 bool) routesRemote {
	key := func(camel string) string {
		if !v2 {
			return camel
		}
		var b strings.Builder
		for _, r := range camel {
			if r >= 'A' && r <= 'Z' {
				b.WriteByte('_')
				r += 'a' - 'A'
			}
			b.WriteRune(r)
		}
		return b.String()
	}
	return routesRemote{
		routesSettings: routesSettings{VoiceRegion: o.optStr(key("voiceRegion")), MessagingRegion: o.optStr(key("messagingRegion")), FriendlyName: o.optStr(key("friendlyName"))},
		sid:            o.str("sid"), accountSid: o.str(key("accountSid")), url: o.str("url"),
		dateCreated: o.str(key("dateCreated")), dateUpdated: o.str(key("dateUpdated")),
	}
}

func routesRead(ctx context.Context, c *Client, v2 bool, path, what string) (routesRemote, error) {
	o, err := c.do(ctx, "GET", routesProduct, path, nil, nil)
	if err != nil {
		if isNotFound(err) {
			return routesRemote{}, err
		}
		return routesRemote{}, fmt.Errorf("read routes of %s: %w", what, err)
	}
	return routesParse(o, v2), nil
}

// routesWrite POSTs the set settings and returns the object Twilio answers with.
func routesWrite(ctx context.Context, c *Client, v2 bool, path, what string, s routesSettings) (routesRemote, error) {
	if s.empty() {
		return routesRemote{}, fmt.Errorf("set routes of %s: set at least one of the region or friendlyName inputs", what)
	}
	o, err := c.do(ctx, "POST", routesProduct, path, nil, routesForm(s, v2))
	if err != nil {
		return routesRemote{}, fmt.Errorf("set routes of %s: %w", what, err)
	}
	return routesParse(o, v2), nil
}

// routesAdopt folds a remote reading into the inputs. An import (no prior inputs) adopts every
// setting Twilio reports; a refresh only reads back the settings the program set, so a setting
// the program leaves to Twilio does not become a diff.
func routesAdopt(cur, remote routesSettings, imported bool) routesSettings {
	pick := func(cur, remote *string) *string {
		if imported {
			if remote != nil && *remote != "" {
				return remote
			}
			return nil
		}
		if cur == nil {
			return nil
		}
		return remote
	}
	return routesSettings{VoiceRegion: pick(cur.VoiceRegion, remote.VoiceRegion), MessagingRegion: pick(cur.MessagingRegion, remote.MessagingRegion), FriendlyName: pick(cur.FriendlyName, remote.FriendlyName)}
}

func routesKeys(kind, id string, n int) ([]string, error) {
	keys := strings.Split(id, "/")
	if len(keys) != n || strings.Contains(id, "//") || keys[0] == "" || keys[n-1] == "" {
		want := "<value>"
		if n == 2 {
			want = "<isoCountryCode>/<shortCode>"
		}
		return nil, fmt.Errorf("invalid %s ID %q: want %s", kind, id, want)
	}
	return keys, nil
}

// RoutesSipDomain pins the inbound voice processing region of an existing SIP domain.
type RoutesSipDomain struct{}

// RoutesSipDomainArgs are the inputs. Twilio cannot unset a setting, so removing one from the program leaves the current value in place.
type RoutesSipDomainArgs struct {
	SipDomain    string  `pulumi:"sipDomain" provider:"replaceOnChanges"`
	VoiceRegion  *string `pulumi:"voiceRegion,optional"`
	FriendlyName *string `pulumi:"friendlyName,optional"`
}

func (a *RoutesSipDomainArgs) Annotate(an infer.Annotator) {
	an.Describe(&a.SipDomain, "The SIP domain name (for example `example.sip.twilio.com`) whose inbound voice processing region is set. The domain must already exist; changing it replaces the resource.")
	an.Describe(&a.VoiceRegion, "Region that processes inbound voice traffic for the object (for example `us1`, `ie1`, `au1`). Twilio keeps the current region when unset.")
	an.Describe(&a.FriendlyName, "Friendly name of the routing setting. Twilio keeps the current name when unset.")
}

// RoutesSipDomainState is the persisted state.
type RoutesSipDomainState struct {
	RoutesSipDomainArgs

	// Sid is the SID of the routing setting.
	Sid string `pulumi:"sid"`

	// AccountSid is the SID of the account that owns the object.
	AccountSid string `pulumi:"accountSid"`

	// Url absolute URL of the Routes resource.
	Url string `pulumi:"url"`

	// DateCreated when the routing setting was created (ISO 8601).
	DateCreated string `pulumi:"dateCreated"`

	// DateUpdated when the routing setting was last updated (ISO 8601).
	DateUpdated string `pulumi:"dateUpdated"`
}

func (s *RoutesSipDomainState) Annotate(a infer.Annotator) {
	a.Describe(&s.Sid, "SID of the routing setting.")
	a.Describe(&s.AccountSid, "SID of the account that owns the object.")
	a.Describe(&s.Url, "Absolute URL of the Routes resource.")
	a.Describe(&s.DateCreated, "When the routing setting was created (ISO 8601).")
	a.Describe(&s.DateUpdated, "When the routing setting was last updated (ISO 8601).")
}

func (r *RoutesSipDomain) Annotate(a infer.Annotator) {
	a.Describe(&r, "Sets the inbound processing region of an existing Twilio SIP domain (POST /v2/SipDomains/{SipDomain}). "+
		"Twilio documents no call that removes the setting, so destroy only forgets it and the object keeps its current region. "+
		"The object must already exist and belong to the account. Import with the SIP domain name. "+
		"The Routes API is unverified against a live account.")
}

var _ infer.CustomResource[RoutesSipDomainArgs, RoutesSipDomainState] = (*RoutesSipDomain)(nil)
var _ infer.CustomRead[RoutesSipDomainArgs, RoutesSipDomainState] = (*RoutesSipDomain)(nil)
var _ infer.CustomUpdate[RoutesSipDomainArgs, RoutesSipDomainState] = (*RoutesSipDomain)(nil)
var _ infer.CustomDelete[RoutesSipDomainState] = (*RoutesSipDomain)(nil)

func (a RoutesSipDomainArgs) settings() routesSettings {
	return routesSettings{VoiceRegion: a.VoiceRegion, FriendlyName: a.FriendlyName}
}

func (a RoutesSipDomainArgs) keys() []string { return []string{a.SipDomain} }

func (a RoutesSipDomainArgs) path() string { return routesPath("v2", "SipDomains", a.keys()...) }

func (a RoutesSipDomainArgs) what() string { return "SIP domain " + strings.Join(a.keys(), "/") }

func (a RoutesSipDomainArgs) validate() error {
	for _, k := range a.keys() {
		if k == "" {
			return fmt.Errorf("RoutesSipDomain: sipDomain must not be empty")
		}
	}
	return nil
}

func (a RoutesSipDomainArgs) withSettings(s routesSettings) RoutesSipDomainArgs {
	a.VoiceRegion = s.VoiceRegion
	a.FriendlyName = s.FriendlyName
	return a
}

func (a RoutesSipDomainArgs) state(rm routesRemote) RoutesSipDomainState {
	return RoutesSipDomainState{RoutesSipDomainArgs: a, Sid: rm.sid, AccountSid: rm.accountSid, Url: rm.url, DateCreated: rm.dateCreated, DateUpdated: rm.dateUpdated}
}

func (r *RoutesSipDomain) Create(ctx context.Context, req infer.CreateRequest[RoutesSipDomainArgs]) (infer.CreateResponse[RoutesSipDomainState], error) {
	type resp = infer.CreateResponse[RoutesSipDomainState]
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
	rm, err := routesWrite(ctx, c, true, a.path(), a.what(), a.settings())
	if err != nil {
		return resp{}, err
	}
	return resp{ID: strings.Join(a.keys(), "/"), Output: a.state(rm)}, nil
}

func (r *RoutesSipDomain) Read(ctx context.Context, req infer.ReadRequest[RoutesSipDomainArgs, RoutesSipDomainState]) (infer.ReadResponse[RoutesSipDomainArgs, RoutesSipDomainState], error) {
	type resp = infer.ReadResponse[RoutesSipDomainArgs, RoutesSipDomainState]
	keys, err := routesKeys("RoutesSipDomain", req.ID, 1)
	if err != nil {
		return resp{}, err
	}
	c, err := clientFor(ctx)
	if err != nil {
		return resp{}, err
	}
	in := req.Inputs
	in.SipDomain = keys[0]
	rm, err := routesRead(ctx, c, true, in.path(), in.what())
	if err != nil {
		if isNotFound(err) {
			if err := notFoundOnImport("routing setting of SIP domain", req.ID, req.Inputs, req.State); err != nil {
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

func (r *RoutesSipDomain) Update(ctx context.Context, req infer.UpdateRequest[RoutesSipDomainArgs, RoutesSipDomainState]) (infer.UpdateResponse[RoutesSipDomainState], error) {
	type resp = infer.UpdateResponse[RoutesSipDomainState]
	st := req.State
	st.RoutesSipDomainArgs = req.Inputs
	if req.DryRun {
		return resp{Output: st}, nil
	}
	c, err := clientFor(ctx)
	if err != nil {
		return resp{}, err
	}
	rm, err := routesWrite(ctx, c, true, req.Inputs.path(), req.Inputs.what(), req.Inputs.settings())
	if err != nil {
		return resp{}, err
	}
	return resp{Output: req.Inputs.state(rm)}, nil
}

// Delete only forgets the setting: Twilio documents no call that removes it.
func (r *RoutesSipDomain) Delete(ctx context.Context, req infer.DeleteRequest[RoutesSipDomainState]) (infer.DeleteResponse, error) {
	return infer.DeleteResponse{}, nil
}

// GetRoutesSipDomain reads the routing settings of a SIP domain.
type GetRoutesSipDomain struct{}

// GetRoutesSipDomainArgs are the arguments.
type GetRoutesSipDomainArgs struct {
	SipDomain string `pulumi:"sipDomain"`
}

func (a *GetRoutesSipDomainArgs) Annotate(an infer.Annotator) {
	an.Describe(&a.SipDomain, "The SIP domain name (for example `example.sip.twilio.com`) whose inbound voice processing region is set. The domain must already exist; changing it replaces the resource.")
}

// GetRoutesSipDomainResult is the object's routing settings.
type GetRoutesSipDomainResult struct {
	SipDomain    string `pulumi:"sipDomain"`
	VoiceRegion  string `pulumi:"voiceRegion"`
	FriendlyName string `pulumi:"friendlyName"`
	Sid          string `pulumi:"sid"`
	AccountSid   string `pulumi:"accountSid"`
	Url          string `pulumi:"url"`
	DateCreated  string `pulumi:"dateCreated"`
	DateUpdated  string `pulumi:"dateUpdated"`
}

func (a *GetRoutesSipDomainResult) Annotate(an infer.Annotator) {
	an.Describe(&a.SipDomain, "As passed in.")
	an.Describe(&a.VoiceRegion, "Region that processes inbound voice traffic for the object (for example `us1`, `ie1`, `au1`); empty when Twilio reports none.")
	an.Describe(&a.FriendlyName, "Friendly name of the routing setting; empty when Twilio reports none.")
	an.Describe(&a.Sid, "SID of the routing setting.")
	an.Describe(&a.AccountSid, "SID of the account that owns the object.")
	an.Describe(&a.Url, "Absolute URL of the Routes resource.")
	an.Describe(&a.DateCreated, "When the routing setting was created (ISO 8601).")
	an.Describe(&a.DateUpdated, "When the routing setting was last updated (ISO 8601).")
}

func (f *GetRoutesSipDomain) Annotate(a infer.Annotator) {
	a.Describe(&f, "Read the routing settings of a SIP domain (GET /v2/SipDomains/{SipDomain}). Fails when the object does not exist.")
}

func (f *GetRoutesSipDomain) Invoke(ctx context.Context, req infer.FunctionRequest[GetRoutesSipDomainArgs]) (infer.FunctionResponse[GetRoutesSipDomainResult], error) {
	type resp = infer.FunctionResponse[GetRoutesSipDomainResult]
	in := RoutesSipDomainArgs{SipDomain: req.Input.SipDomain}
	for i, name := range []string{"sipDomain"} {
		if err := requireArg("getRoutesSipDomain", name, in.keys()[i]); err != nil {
			return resp{}, err
		}
	}
	c, err := clientFor(ctx)
	if err != nil {
		return resp{}, err
	}
	rm, err := routesRead(ctx, c, true, in.path(), in.what())
	if err != nil {
		return resp{}, lookupFailure("getRoutesSipDomain", in.what(), err)
	}
	return resp{Output: GetRoutesSipDomainResult{SipDomain: in.SipDomain, VoiceRegion: deref(rm.VoiceRegion), FriendlyName: deref(rm.FriendlyName), Sid: rm.sid, AccountSid: rm.accountSid, Url: rm.url, DateCreated: rm.dateCreated, DateUpdated: rm.dateUpdated}}, nil
}

// RoutesTrunk pins the inbound voice processing region of an existing SIP trunk domain.
type RoutesTrunk struct{}

// RoutesTrunkArgs are the inputs. Twilio cannot unset a setting, so removing one from the program leaves the current value in place.
type RoutesTrunkArgs struct {
	SipTrunkDomain string  `pulumi:"sipTrunkDomain" provider:"replaceOnChanges"`
	VoiceRegion    *string `pulumi:"voiceRegion,optional"`
	FriendlyName   *string `pulumi:"friendlyName,optional"`
}

func (a *RoutesTrunkArgs) Annotate(an infer.Annotator) {
	an.Describe(&a.SipTrunkDomain, "The SIP trunk domain name (for example `example.pstn.twilio.com`) whose inbound voice processing region is set. The trunk must already exist; changing it replaces the resource.")
	an.Describe(&a.VoiceRegion, "Region that processes inbound voice traffic for the object (for example `us1`, `ie1`, `au1`). Twilio keeps the current region when unset.")
	an.Describe(&a.FriendlyName, "Friendly name of the routing setting. Twilio keeps the current name when unset.")
}

// RoutesTrunkState is the persisted state.
type RoutesTrunkState struct {
	RoutesTrunkArgs

	// Sid is the SID of the routing setting.
	Sid string `pulumi:"sid"`

	// AccountSid is the SID of the account that owns the object.
	AccountSid string `pulumi:"accountSid"`

	// Url absolute URL of the Routes resource.
	Url string `pulumi:"url"`

	// DateCreated when the routing setting was created (ISO 8601).
	DateCreated string `pulumi:"dateCreated"`

	// DateUpdated when the routing setting was last updated (ISO 8601).
	DateUpdated string `pulumi:"dateUpdated"`
}

func (s *RoutesTrunkState) Annotate(a infer.Annotator) {
	a.Describe(&s.Sid, "SID of the routing setting.")
	a.Describe(&s.AccountSid, "SID of the account that owns the object.")
	a.Describe(&s.Url, "Absolute URL of the Routes resource.")
	a.Describe(&s.DateCreated, "When the routing setting was created (ISO 8601).")
	a.Describe(&s.DateUpdated, "When the routing setting was last updated (ISO 8601).")
}

func (r *RoutesTrunk) Annotate(a infer.Annotator) {
	a.Describe(&r, "Sets the inbound processing region of an existing Twilio SIP trunk domain (POST /v2/Trunks/{SipTrunkDomain}). "+
		"Twilio documents no call that removes the setting, so destroy only forgets it and the object keeps its current region. "+
		"The object must already exist and belong to the account. Import with the SIP trunk domain name. "+
		"The Routes API is unverified against a live account.")
}

var _ infer.CustomResource[RoutesTrunkArgs, RoutesTrunkState] = (*RoutesTrunk)(nil)
var _ infer.CustomRead[RoutesTrunkArgs, RoutesTrunkState] = (*RoutesTrunk)(nil)
var _ infer.CustomUpdate[RoutesTrunkArgs, RoutesTrunkState] = (*RoutesTrunk)(nil)
var _ infer.CustomDelete[RoutesTrunkState] = (*RoutesTrunk)(nil)

func (a RoutesTrunkArgs) settings() routesSettings {
	return routesSettings{VoiceRegion: a.VoiceRegion, FriendlyName: a.FriendlyName}
}

func (a RoutesTrunkArgs) keys() []string { return []string{a.SipTrunkDomain} }

func (a RoutesTrunkArgs) path() string { return routesPath("v2", "Trunks", a.keys()...) }

func (a RoutesTrunkArgs) what() string { return "SIP trunk domain " + strings.Join(a.keys(), "/") }

func (a RoutesTrunkArgs) validate() error {
	for _, k := range a.keys() {
		if k == "" {
			return fmt.Errorf("RoutesTrunk: sipTrunkDomain must not be empty")
		}
	}
	return nil
}

func (a RoutesTrunkArgs) withSettings(s routesSettings) RoutesTrunkArgs {
	a.VoiceRegion = s.VoiceRegion
	a.FriendlyName = s.FriendlyName
	return a
}

func (a RoutesTrunkArgs) state(rm routesRemote) RoutesTrunkState {
	return RoutesTrunkState{RoutesTrunkArgs: a, Sid: rm.sid, AccountSid: rm.accountSid, Url: rm.url, DateCreated: rm.dateCreated, DateUpdated: rm.dateUpdated}
}

func (r *RoutesTrunk) Create(ctx context.Context, req infer.CreateRequest[RoutesTrunkArgs]) (infer.CreateResponse[RoutesTrunkState], error) {
	type resp = infer.CreateResponse[RoutesTrunkState]
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
	rm, err := routesWrite(ctx, c, true, a.path(), a.what(), a.settings())
	if err != nil {
		return resp{}, err
	}
	return resp{ID: strings.Join(a.keys(), "/"), Output: a.state(rm)}, nil
}

func (r *RoutesTrunk) Read(ctx context.Context, req infer.ReadRequest[RoutesTrunkArgs, RoutesTrunkState]) (infer.ReadResponse[RoutesTrunkArgs, RoutesTrunkState], error) {
	type resp = infer.ReadResponse[RoutesTrunkArgs, RoutesTrunkState]
	keys, err := routesKeys("RoutesTrunk", req.ID, 1)
	if err != nil {
		return resp{}, err
	}
	c, err := clientFor(ctx)
	if err != nil {
		return resp{}, err
	}
	in := req.Inputs
	in.SipTrunkDomain = keys[0]
	rm, err := routesRead(ctx, c, true, in.path(), in.what())
	if err != nil {
		if isNotFound(err) {
			if err := notFoundOnImport("routing setting of SIP trunk domain", req.ID, req.Inputs, req.State); err != nil {
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

func (r *RoutesTrunk) Update(ctx context.Context, req infer.UpdateRequest[RoutesTrunkArgs, RoutesTrunkState]) (infer.UpdateResponse[RoutesTrunkState], error) {
	type resp = infer.UpdateResponse[RoutesTrunkState]
	st := req.State
	st.RoutesTrunkArgs = req.Inputs
	if req.DryRun {
		return resp{Output: st}, nil
	}
	c, err := clientFor(ctx)
	if err != nil {
		return resp{}, err
	}
	rm, err := routesWrite(ctx, c, true, req.Inputs.path(), req.Inputs.what(), req.Inputs.settings())
	if err != nil {
		return resp{}, err
	}
	return resp{Output: req.Inputs.state(rm)}, nil
}

// Delete only forgets the setting: Twilio documents no call that removes it.
func (r *RoutesTrunk) Delete(ctx context.Context, req infer.DeleteRequest[RoutesTrunkState]) (infer.DeleteResponse, error) {
	return infer.DeleteResponse{}, nil
}

// GetRoutesTrunk reads the routing settings of a SIP trunk domain.
type GetRoutesTrunk struct{}

// GetRoutesTrunkArgs are the arguments.
type GetRoutesTrunkArgs struct {
	SipTrunkDomain string `pulumi:"sipTrunkDomain"`
}

func (a *GetRoutesTrunkArgs) Annotate(an infer.Annotator) {
	an.Describe(&a.SipTrunkDomain, "The SIP trunk domain name (for example `example.pstn.twilio.com`) whose inbound voice processing region is set. The trunk must already exist; changing it replaces the resource.")
}

// GetRoutesTrunkResult is the object's routing settings.
type GetRoutesTrunkResult struct {
	SipTrunkDomain string `pulumi:"sipTrunkDomain"`
	VoiceRegion    string `pulumi:"voiceRegion"`
	FriendlyName   string `pulumi:"friendlyName"`
	Sid            string `pulumi:"sid"`
	AccountSid     string `pulumi:"accountSid"`
	Url            string `pulumi:"url"`
	DateCreated    string `pulumi:"dateCreated"`
	DateUpdated    string `pulumi:"dateUpdated"`
}

func (a *GetRoutesTrunkResult) Annotate(an infer.Annotator) {
	an.Describe(&a.SipTrunkDomain, "As passed in.")
	an.Describe(&a.VoiceRegion, "Region that processes inbound voice traffic for the object (for example `us1`, `ie1`, `au1`); empty when Twilio reports none.")
	an.Describe(&a.FriendlyName, "Friendly name of the routing setting; empty when Twilio reports none.")
	an.Describe(&a.Sid, "SID of the routing setting.")
	an.Describe(&a.AccountSid, "SID of the account that owns the object.")
	an.Describe(&a.Url, "Absolute URL of the Routes resource.")
	an.Describe(&a.DateCreated, "When the routing setting was created (ISO 8601).")
	an.Describe(&a.DateUpdated, "When the routing setting was last updated (ISO 8601).")
}

func (f *GetRoutesTrunk) Annotate(a infer.Annotator) {
	a.Describe(&f, "Read the routing settings of a SIP trunk domain (GET /v2/Trunks/{SipTrunkDomain}). Fails when the object does not exist.")
}

func (f *GetRoutesTrunk) Invoke(ctx context.Context, req infer.FunctionRequest[GetRoutesTrunkArgs]) (infer.FunctionResponse[GetRoutesTrunkResult], error) {
	type resp = infer.FunctionResponse[GetRoutesTrunkResult]
	in := RoutesTrunkArgs{SipTrunkDomain: req.Input.SipTrunkDomain}
	for i, name := range []string{"sipTrunkDomain"} {
		if err := requireArg("getRoutesTrunk", name, in.keys()[i]); err != nil {
			return resp{}, err
		}
	}
	c, err := clientFor(ctx)
	if err != nil {
		return resp{}, err
	}
	rm, err := routesRead(ctx, c, true, in.path(), in.what())
	if err != nil {
		return resp{}, lookupFailure("getRoutesTrunk", in.what(), err)
	}
	return resp{Output: GetRoutesTrunkResult{SipTrunkDomain: in.SipTrunkDomain, VoiceRegion: deref(rm.VoiceRegion), FriendlyName: deref(rm.FriendlyName), Sid: rm.sid, AccountSid: rm.accountSid, Url: rm.url, DateCreated: rm.dateCreated, DateUpdated: rm.dateUpdated}}, nil
}
