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

// Twilio defaults of a Proxy service that an import leaves unset.
const (
	proxyDefaultGeoMatch  = "country"
	proxyDefaultSelection = "prefer-sticky"
)

// ProxyService manages a Proxy service: the top-level scope of sessions and proxy numbers.
type ProxyService struct{}

// ProxyServiceArgs are the inputs of a ProxyService.
type ProxyServiceArgs struct {
	UniqueName              string  `pulumi:"uniqueName"`
	DefaultTtl              *int    `pulumi:"defaultTtl,optional"`
	CallbackUrl             *string `pulumi:"callbackUrl,optional"`
	GeoMatchLevel           *string `pulumi:"geoMatchLevel,optional"`
	NumberSelectionBehavior *string `pulumi:"numberSelectionBehavior,optional"`
	InterceptCallbackUrl    *string `pulumi:"interceptCallbackUrl,optional"`
	OutOfSessionCallbackUrl *string `pulumi:"outOfSessionCallbackUrl,optional"`
	ChatInstanceSid         *string `pulumi:"chatInstanceSid,optional"`
}

func (a *ProxyServiceArgs) Annotate(an infer.Annotator) {
	an.Describe(&a.UniqueName, "Application-defined name of the service, up to 191 characters and unique in the account. Do not put personal data in it.")
	an.Describe(&a.DefaultTtl, "Default time to live in seconds of the sessions created in the service, measured from the session's last create or interaction; 0 is unlimited (Twilio default 0).")
	an.Describe(&a.CallbackUrl, "Absolute URL Twilio calls when an interaction status changes. Removing it leaves the setting unchanged.")
	an.Describe(&a.GeoMatchLevel, "Where a proxy number must be located relative to the participant: country, area-code, overlay, radius or extended-area-code (Twilio default country; more specific levels are only available in North America).")
	an.Describe(&a.NumberSelectionBehavior, "How proxy numbers are chosen: prefer-sticky tries to reuse a participant's earlier number, avoid-sticky tries to use a different one (Twilio default prefer-sticky).")
	an.Describe(&a.InterceptCallbackUrl, "Absolute URL Twilio calls on every interaction; a 403 response blocks the interaction. Removing it leaves the setting unchanged.")
	an.Describe(&a.OutOfSessionCallbackUrl, "Absolute URL Twilio calls when a call or SMS arrives for a closed or missing session. Removing it leaves the setting unchanged.")
	an.Describe(&a.ChatInstanceSid, "SID (IS...) of the Chat service instance Proxy forwards SMS and channel messages to. Removing it leaves the setting unchanged.")
}

// ProxyServiceState is the persisted state of a ProxyService.
type ProxyServiceState struct {
	ProxyServiceArgs

	Sid         string `pulumi:"sid"`
	DateCreated string `pulumi:"dateCreated"`
	DateUpdated string `pulumi:"dateUpdated"`
}

func (s *ProxyServiceState) Annotate(a infer.Annotator) {
	a.Describe(&s.Sid, "The service SID (KS...).")
	a.Describe(&s.DateCreated, "When the service was created (ISO 8601).")
	a.Describe(&s.DateUpdated, "When the service was last updated (ISO 8601).")
}

func (r *ProxyService) Annotate(a infer.Annotator) {
	a.Describe(&r, "A Proxy service (Public Beta). The resource ID is the service SID. Optional settings that are set are sent and kept in step with Twilio; one that is removed leaves the setting as it is on Twilio.")
}

var _ infer.CustomResource[ProxyServiceArgs, ProxyServiceState] = (*ProxyService)(nil)
var _ infer.CustomCheck[ProxyServiceArgs] = (*ProxyService)(nil)
var _ infer.CustomDiff[ProxyServiceArgs, ProxyServiceState] = (*ProxyService)(nil)
var _ infer.CustomRead[ProxyServiceArgs, ProxyServiceState] = (*ProxyService)(nil)
var _ infer.CustomUpdate[ProxyServiceArgs, ProxyServiceState] = (*ProxyService)(nil)
var _ infer.CustomDelete[ProxyServiceState] = (*ProxyService)(nil)

func (r *ProxyService) Check(ctx context.Context, req infer.CheckRequest) (infer.CheckResponse[ProxyServiceArgs], error) {
	args, failures, err := infer.DefaultCheck[ProxyServiceArgs](ctx, req.NewInputs)
	if err != nil || len(failures) > 0 {
		return infer.CheckResponse[ProxyServiceArgs]{Inputs: args, Failures: failures}, err
	}
	c := &checker{unk: unknowns(req.NewInputs)}
	c.required("uniqueName", args.UniqueName)
	c.maxLen("uniqueName", args.UniqueName, 191)
	if v := args.DefaultTtl; v != nil && !c.unk("defaultTtl") && *v < 0 {
		c.fail("defaultTtl", "defaultTtl must not be negative")
	}
	if v := args.GeoMatchLevel; v != nil {
		c.oneOf("geoMatchLevel", *v, []string{"country", "area-code", "extended-area-code", "overlay", "radius"})
	}
	if v := args.NumberSelectionBehavior; v != nil {
		c.oneOf("numberSelectionBehavior", *v, []string{"prefer-sticky", "avoid-sticky"})
	}
	if v := args.ChatInstanceSid; v != nil {
		c.sid("chatInstanceSid", *v, proxyChatSidRE, "Chat service (IS followed by 32 hex characters)")
	}
	return infer.CheckResponse[ProxyServiceArgs]{Inputs: args, Failures: c.failures}, nil
}

func (r *ProxyService) Diff(_ context.Context, req infer.DiffRequest[ProxyServiceArgs, ProxyServiceState]) (infer.DiffResponse, error) {
	o, n := req.State.ProxyServiceArgs, req.Inputs
	d := trDiffSet{}
	d.requiredStr("uniqueName", o.UniqueName, n.UniqueName)
	d.intDef(proxyServiceDefaults, "defaultTtl", o.DefaultTtl, n.DefaultTtl)
	d.strDef(proxyServiceDefaults, "callbackUrl", o.CallbackUrl, n.CallbackUrl)
	d.strDef(proxyServiceDefaults, "geoMatchLevel", o.GeoMatchLevel, n.GeoMatchLevel)
	d.strDef(proxyServiceDefaults, "numberSelectionBehavior", o.NumberSelectionBehavior, n.NumberSelectionBehavior)
	d.strDef(proxyServiceDefaults, "interceptCallbackUrl", o.InterceptCallbackUrl, n.InterceptCallbackUrl)
	d.strDef(proxyServiceDefaults, "outOfSessionCallbackUrl", o.OutOfSessionCallbackUrl, n.OutOfSessionCallbackUrl)
	d.strDef(proxyServiceDefaults, "chatInstanceSid", o.ChatInstanceSid, n.ChatInstanceSid)
	return d.response(), nil
}

func (a ProxyServiceArgs) form() url.Values {
	form := url.Values{}
	form.Set("UniqueName", a.UniqueName)
	setI(form, "DefaultTtl", a.DefaultTtl)
	setS(form, "CallbackUrl", a.CallbackUrl)
	setS(form, "GeoMatchLevel", a.GeoMatchLevel)
	setS(form, "NumberSelectionBehavior", a.NumberSelectionBehavior)
	setS(form, "InterceptCallbackUrl", a.InterceptCallbackUrl)
	setS(form, "OutOfSessionCallbackUrl", a.OutOfSessionCallbackUrl)
	setS(form, "ChatInstanceSid", a.ChatInstanceSid)
	return form
}

func (r *ProxyService) Create(ctx context.Context, req infer.CreateRequest[ProxyServiceArgs]) (infer.CreateResponse[ProxyServiceState], error) {
	type resp = infer.CreateResponse[ProxyServiceState]
	a := req.Inputs
	if req.DryRun {
		return resp{Output: ProxyServiceState{ProxyServiceArgs: a}}, nil
	}
	c, err := clientFor(ctx)
	if err != nil {
		return resp{}, err
	}
	o, err := c.do(ctx, "POST", proxyProduct, proxyPath(), nil, a.form())
	if err != nil {
		return resp{}, fmt.Errorf("create Proxy service: %w", err)
	}
	st := ProxyServiceState{ProxyServiceArgs: a}
	st.Sid, st.DateCreated, st.DateUpdated = trSid(o)
	return resp{ID: st.Sid, Output: st}, nil
}

func (r *ProxyService) Read(ctx context.Context, req infer.ReadRequest[ProxyServiceArgs, ProxyServiceState]) (infer.ReadResponse[ProxyServiceArgs, ProxyServiceState], error) {
	type resp = infer.ReadResponse[ProxyServiceArgs, ProxyServiceState]
	c, err := clientFor(ctx)
	if err != nil {
		return resp{}, err
	}
	o, err := c.do(ctx, "GET", proxyProduct, proxyPath(req.ID), nil, nil)
	if err != nil {
		if isNotFound(err) {
			if err := notFoundOnImport("Proxy service", req.ID, req.Inputs, req.State); err != nil {
				return resp{}, err
			}
			return resp{}, nil
		}
		return resp{}, fmt.Errorf("read Proxy service %s: %w", req.ID, err)
	}
	in := req.Inputs
	// An import arrives with no prior state; the program's own settings are the only record of what it manages.
	imported := req.State.Sid == ""
	in.UniqueName = o.str("unique_name")
	in.DefaultTtl = trInt(in.DefaultTtl, o.optInt("default_ttl"), proxyServiceDefaults.integer("defaultTtl"), imported)
	in.CallbackUrl = trStr(in.CallbackUrl, o.optStr("callback_url"), proxyServiceDefaults.str("callbackUrl"), imported)
	in.GeoMatchLevel = trStr(in.GeoMatchLevel, o.optStr("geo_match_level"), proxyServiceDefaults.str("geoMatchLevel"), imported)
	in.NumberSelectionBehavior = trStr(in.NumberSelectionBehavior, o.optStr("number_selection_behavior"), proxyServiceDefaults.str("numberSelectionBehavior"), imported)
	in.InterceptCallbackUrl = trStr(in.InterceptCallbackUrl, o.optStr("intercept_callback_url"), proxyServiceDefaults.str("interceptCallbackUrl"), imported)
	in.OutOfSessionCallbackUrl = trStr(in.OutOfSessionCallbackUrl, o.optStr("out_of_session_callback_url"), proxyServiceDefaults.str("outOfSessionCallbackUrl"), imported)
	in.ChatInstanceSid = trStr(in.ChatInstanceSid, o.optStr("chat_instance_sid"), proxyServiceDefaults.str("chatInstanceSid"), imported)
	st := ProxyServiceState{ProxyServiceArgs: in}
	st.Sid, st.DateCreated, st.DateUpdated = trSid(o)
	return resp{ID: req.ID, Inputs: in, State: st}, nil
}

func (r *ProxyService) Update(ctx context.Context, req infer.UpdateRequest[ProxyServiceArgs, ProxyServiceState]) (infer.UpdateResponse[ProxyServiceState], error) {
	type resp = infer.UpdateResponse[ProxyServiceState]
	st := req.State
	st.ProxyServiceArgs = req.Inputs
	if req.DryRun {
		return resp{Output: st}, nil
	}
	c, err := clientFor(ctx)
	if err != nil {
		return resp{}, err
	}
	o, err := c.do(ctx, "POST", proxyProduct, proxyPath(req.ID), nil, req.Inputs.form())
	if err != nil {
		return resp{}, fmt.Errorf("update Proxy service %s: %w", req.ID, err)
	}
	st.DateUpdated = o.str("date_updated")
	return resp{Output: st}, nil
}

func (r *ProxyService) Delete(ctx context.Context, req infer.DeleteRequest[ProxyServiceState]) (infer.DeleteResponse, error) {
	c, err := clientFor(ctx)
	if err != nil {
		return infer.DeleteResponse{}, err
	}
	if _, err := c.do(ctx, "DELETE", proxyProduct, proxyPath(req.ID), nil, nil); err != nil && !isNotFound(err) {
		return infer.DeleteResponse{}, fmt.Errorf("delete Proxy service %s: %w", req.ID, err)
	}
	return infer.DeleteResponse{}, nil
}

// proxyServiceDefaults are the Twilio defaults of this resource, shared by Read (import adoption) and Diff.
var proxyServiceDefaults = trDefaults{
	"defaultTtl":              0,
	"callbackUrl":             "",
	"geoMatchLevel":           proxyDefaultGeoMatch,
	"numberSelectionBehavior": proxyDefaultSelection,
	"interceptCallbackUrl":    "",
	"outOfSessionCallbackUrl": "",
	"chatInstanceSid":         "",
}
