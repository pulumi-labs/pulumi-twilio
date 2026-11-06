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
	"strconv"
	"strings"

	p "github.com/pulumi/pulumi-go-provider"
	"github.com/pulumi/pulumi-go-provider/infer"
)

// splitTrunkChildID splits <trunkSid>/<sid>.
func splitTrunkChildID(id string) (trunk, sid string, err error) {
	parts := strings.Split(id, "/")
	if len(parts) != 2 || parts[0] == "" || parts[1] == "" {
		return "", "", fmt.Errorf("invalid ID %q: expected <trunkSid>/<sid>", id)
	}
	return parts[0], parts[1], nil
}

// ------------------------------------------------------- TrunkOriginationUrl

// TrunkOriginationUrl manages an origination URL of a trunk: a SIP address Twilio routes the
// trunk's inbound calls to, with a priority and a weight that spread the load.
type TrunkOriginationUrl struct{}

const (
	originationDefaultWeight   = 10
	originationDefaultPriority = 10
)

// TrunkOriginationUrlArgs are the inputs of a TrunkOriginationUrl.
type TrunkOriginationUrlArgs struct {
	TrunkSid     string `pulumi:"trunkSid" provider:"replaceOnChanges"`
	FriendlyName string `pulumi:"friendlyName"`
	SipUrl       string `pulumi:"sipUrl"`
	Weight       *int   `pulumi:"weight,optional"`
	Priority     *int   `pulumi:"priority,optional"`
	Enabled      *bool  `pulumi:"enabled,optional"`
}

func (a *TrunkOriginationUrlArgs) Annotate(an infer.Annotator) {
	an.Describe(&a.TrunkSid, "SID (TK...) of the trunk. Changing it replaces the resource.")
	an.Describe(&a.FriendlyName, "Human-readable name of the origination URL (up to 64 characters).")
	an.Describe(&a.SipUrl, "The SIP address Twilio routes origination calls to. It must use the sip: scheme (sips: is not supported).")
	an.Describe(&a.Weight, "Relative share of the load a URL receives among URLs of the same priority, 1 to 65535 (default 10).")
	an.Describe(&a.Priority, "Relative importance of the URL, 0 to 65535 (default 10). The lowest number is the most important.")
	an.Describe(&a.Enabled, "Whether the URL is enabled (default true).")
}

// TrunkOriginationUrlState is the persisted state of a TrunkOriginationUrl.
type TrunkOriginationUrlState struct {
	TrunkOriginationUrlArgs

	// Sid is the origination URL SID (OU...).
	Sid string `pulumi:"sid"`
	// DateCreated and DateUpdated are RFC 2822 timestamps.
	DateCreated string `pulumi:"dateCreated"`
	DateUpdated string `pulumi:"dateUpdated"`
}

func (s *TrunkOriginationUrlState) Annotate(a infer.Annotator) {
	a.Describe(&s.Sid, "The origination URL SID (OU...).")
	a.Describe(&s.DateCreated, "When the URL was created (RFC 2822).")
	a.Describe(&s.DateUpdated, "When the URL was last updated (RFC 2822).")
}

func (r *TrunkOriginationUrl) Annotate(a infer.Annotator) {
	a.Describe(&r, "An origination URL of an Elastic SIP trunk. The resource ID is <trunkSid>/<sid>. "+
		"Twilio accepts only a sip: URL and rejects one whose host it cannot resolve.")
}

var _ infer.CustomResource[TrunkOriginationUrlArgs, TrunkOriginationUrlState] = (*TrunkOriginationUrl)(nil)
var _ infer.CustomCheck[TrunkOriginationUrlArgs] = (*TrunkOriginationUrl)(nil)
var _ infer.CustomDiff[TrunkOriginationUrlArgs, TrunkOriginationUrlState] = (*TrunkOriginationUrl)(nil)
var _ infer.CustomRead[TrunkOriginationUrlArgs, TrunkOriginationUrlState] = (*TrunkOriginationUrl)(nil)
var _ infer.CustomUpdate[TrunkOriginationUrlArgs, TrunkOriginationUrlState] = (*TrunkOriginationUrl)(nil)
var _ infer.CustomDelete[TrunkOriginationUrlState] = (*TrunkOriginationUrl)(nil)

func intOr(v *int, def int) int {
	if v == nil {
		return def
	}
	return *v
}

func (a *TrunkOriginationUrlArgs) form() url.Values {
	v := url.Values{"FriendlyName": {a.FriendlyName}, "SipUrl": {a.SipUrl}}
	// Twilio requires every field on create, so the defaults are sent explicitly.
	v.Set("Weight", strconv.Itoa(intOr(a.Weight, originationDefaultWeight)))
	v.Set("Priority", strconv.Itoa(intOr(a.Priority, originationDefaultPriority)))
	v.Set("Enabled", strconv.FormatBool(a.Enabled == nil || *a.Enabled))
	return v
}

func (r *TrunkOriginationUrl) Check(ctx context.Context, req infer.CheckRequest) (infer.CheckResponse[TrunkOriginationUrlArgs], error) {
	args, failures, err := infer.DefaultCheck[TrunkOriginationUrlArgs](ctx, req.NewInputs)
	if err != nil || len(failures) > 0 {
		return infer.CheckResponse[TrunkOriginationUrlArgs]{Inputs: args, Failures: failures}, err
	}
	unk := unknowns(req.NewInputs)
	fail := func(prop, reason string) { failures = append(failures, p.CheckFailure{Property: prop, Reason: reason}) }
	if !unk("friendlyName") && (args.FriendlyName == "" || len(args.FriendlyName) > 64) {
		fail("friendlyName", "friendlyName must be 1-64 characters")
	}
	if !unk("sipUrl") && !strings.HasPrefix(strings.ToLower(args.SipUrl), "sip:") {
		fail("sipUrl", "sipUrl must start with sip: (sips: is not supported)")
	}
	if args.Weight != nil && !unk("weight") && (*args.Weight < 1 || *args.Weight > 65535) {
		fail("weight", "weight must be between 1 and 65535")
	}
	if args.Priority != nil && !unk("priority") && (*args.Priority < 0 || *args.Priority > 65535) {
		fail("priority", "priority must be between 0 and 65535")
	}
	return infer.CheckResponse[TrunkOriginationUrlArgs]{Inputs: args, Failures: failures}, nil
}

// Diff treats an unset weight, priority or enabled flag as Twilio's default, so an imported URL
// (which adopts what Twilio returns) and a program that omits or restates a default agree.
func (r *TrunkOriginationUrl) Diff(_ context.Context, req infer.DiffRequest[TrunkOriginationUrlArgs, TrunkOriginationUrlState]) (infer.DiffResponse, error) {
	o, n := req.State.TrunkOriginationUrlArgs, req.Inputs
	detail := map[string]p.PropertyDiff{}
	if o.TrunkSid != n.TrunkSid {
		detail["trunkSid"] = p.PropertyDiff{Kind: p.UpdateReplace}
	}
	if o.FriendlyName != n.FriendlyName {
		detail["friendlyName"] = p.PropertyDiff{Kind: p.Update}
	}
	if o.SipUrl != n.SipUrl {
		detail["sipUrl"] = p.PropertyDiff{Kind: p.Update}
	}
	if intOr(o.Weight, originationDefaultWeight) != intOr(n.Weight, originationDefaultWeight) {
		detail["weight"] = p.PropertyDiff{Kind: p.Update}
	}
	if intOr(o.Priority, originationDefaultPriority) != intOr(n.Priority, originationDefaultPriority) {
		detail["priority"] = p.PropertyDiff{Kind: p.Update}
	}
	if (o.Enabled == nil || *o.Enabled) != (n.Enabled == nil || *n.Enabled) {
		detail["enabled"] = p.PropertyDiff{Kind: p.Update}
	}
	return infer.DiffResponse{HasChanges: len(detail) > 0, DetailedDiff: detail}, nil
}

func originationOutputs(st *TrunkOriginationUrlState, o obj) {
	st.Sid, st.DateCreated, st.DateUpdated = o.str("sid"), o.str("date_created"), o.str("date_updated")
}

func (r *TrunkOriginationUrl) Create(ctx context.Context, req infer.CreateRequest[TrunkOriginationUrlArgs]) (infer.CreateResponse[TrunkOriginationUrlState], error) {
	a := req.Inputs
	if req.DryRun {
		return infer.CreateResponse[TrunkOriginationUrlState]{Output: TrunkOriginationUrlState{TrunkOriginationUrlArgs: a}}, nil
	}
	c, err := clientFor(ctx)
	if err != nil {
		return infer.CreateResponse[TrunkOriginationUrlState]{}, err
	}
	o, err := c.do(ctx, "POST", trunkingProduct, trunkChildPath(a.TrunkSid, "OriginationUrls", ""), nil, a.form())
	if err != nil {
		return infer.CreateResponse[TrunkOriginationUrlState]{}, fmt.Errorf("create origination URL on trunk %s: %w", a.TrunkSid, err)
	}
	st := TrunkOriginationUrlState{TrunkOriginationUrlArgs: a}
	originationOutputs(&st, o)
	return infer.CreateResponse[TrunkOriginationUrlState]{ID: a.TrunkSid + "/" + st.Sid, Output: st}, nil
}

func (r *TrunkOriginationUrl) Read(ctx context.Context, req infer.ReadRequest[TrunkOriginationUrlArgs, TrunkOriginationUrlState]) (infer.ReadResponse[TrunkOriginationUrlArgs, TrunkOriginationUrlState], error) {
	type resp = infer.ReadResponse[TrunkOriginationUrlArgs, TrunkOriginationUrlState]
	trunk, sid, err := splitTrunkChildID(req.ID)
	if err != nil {
		return resp{}, err
	}
	c, err := clientFor(ctx)
	if err != nil {
		return resp{}, err
	}
	o, err := c.do(ctx, "GET", trunkingProduct, trunkChildPath(trunk, "OriginationUrls", sid), nil, nil)
	if err != nil {
		if isNotFound(err) {
			return resp{}, nil
		}
		return resp{}, fmt.Errorf("read origination URL %s: %w", req.ID, err)
	}
	in := req.Inputs
	if in.TrunkSid == "" {
		// An import: adopt everything Twilio reports.
		in = TrunkOriginationUrlArgs{TrunkSid: trunk, FriendlyName: o.str("friendly_name"), SipUrl: o.str("sip_url"),
			Weight: o.optInt("weight"), Priority: o.optInt("priority"), Enabled: o.optBool("enabled")}
	} else {
		in.FriendlyName, in.SipUrl = o.str("friendly_name"), o.str("sip_url")
		if in.Weight != nil {
			in.Weight = o.optInt("weight")
		}
		if in.Priority != nil {
			in.Priority = o.optInt("priority")
		}
		if in.Enabled != nil {
			in.Enabled = o.optBool("enabled")
		}
	}
	st := TrunkOriginationUrlState{TrunkOriginationUrlArgs: in}
	originationOutputs(&st, o)
	return resp{ID: req.ID, Inputs: in, State: st}, nil
}

func (r *TrunkOriginationUrl) Update(ctx context.Context, req infer.UpdateRequest[TrunkOriginationUrlArgs, TrunkOriginationUrlState]) (infer.UpdateResponse[TrunkOriginationUrlState], error) {
	a := req.Inputs
	st := req.State
	st.TrunkOriginationUrlArgs = a
	if req.DryRun {
		return infer.UpdateResponse[TrunkOriginationUrlState]{Output: st}, nil
	}
	trunk, sid, err := splitTrunkChildID(req.ID)
	if err != nil {
		return infer.UpdateResponse[TrunkOriginationUrlState]{}, err
	}
	c, err := clientFor(ctx)
	if err != nil {
		return infer.UpdateResponse[TrunkOriginationUrlState]{}, err
	}
	o, err := c.do(ctx, "POST", trunkingProduct, trunkChildPath(trunk, "OriginationUrls", sid), nil, a.form())
	if err != nil {
		return infer.UpdateResponse[TrunkOriginationUrlState]{}, fmt.Errorf("update origination URL %s: %w", req.ID, err)
	}
	originationOutputs(&st, o)
	return infer.UpdateResponse[TrunkOriginationUrlState]{Output: st}, nil
}

func (r *TrunkOriginationUrl) Delete(ctx context.Context, req infer.DeleteRequest[TrunkOriginationUrlState]) (infer.DeleteResponse, error) {
	trunk, sid, err := splitTrunkChildID(req.ID)
	if err != nil {
		return infer.DeleteResponse{}, err
	}
	c, err := clientFor(ctx)
	if err != nil {
		return infer.DeleteResponse{}, err
	}
	if _, err := c.do(ctx, "DELETE", trunkingProduct, trunkChildPath(trunk, "OriginationUrls", sid), nil, nil); err != nil && !isNotFound(err) {
		return infer.DeleteResponse{}, fmt.Errorf("delete origination URL %s: %w", req.ID, err)
	}
	return infer.DeleteResponse{}, nil
}

// ---------------------------------------------------------------- TrunkCredentialList

// TrunkCredentialList attaches a SIP credential list to a trunk. Every input identifies the attachment, so
// any change replaces it; deleting detaches the list without deleting it.
type TrunkCredentialList struct{}

// TrunkCredentialListArgs are the inputs of a TrunkCredentialList.
type TrunkCredentialListArgs struct {
	TrunkSid          string `pulumi:"trunkSid" provider:"replaceOnChanges"`
	CredentialListSid string `pulumi:"credentialListSid" provider:"replaceOnChanges"`
}

func (a *TrunkCredentialListArgs) Annotate(an infer.Annotator) {
	an.Describe(&a.TrunkSid, "SID (TK...) of the trunk. Changing it replaces the resource.")
	an.Describe(&a.CredentialListSid, "SID (CL...) of the SIP credential list (see SipCredentialList) to attach. Changing it replaces the resource.")
}

// TrunkCredentialListState is the persisted state of a TrunkCredentialList.
type TrunkCredentialListState struct {
	TrunkCredentialListArgs

	// Sid is the SID of the attached list.
	Sid string `pulumi:"sid"`
	// FriendlyName is the name of the attached list.
	FriendlyName string `pulumi:"friendlyName"`
	// DateCreated and DateUpdated are RFC 2822 timestamps of the attachment.
	DateCreated string `pulumi:"dateCreated"`
	DateUpdated string `pulumi:"dateUpdated"`
}

func (s *TrunkCredentialListState) Annotate(a infer.Annotator) {
	a.Describe(&s.Sid, "SID of the attached list (CL...).")
	a.Describe(&s.FriendlyName, "Name of the attached list.")
	a.Describe(&s.DateCreated, "When the list was attached (RFC 2822).")
	a.Describe(&s.DateUpdated, "When the list was last updated (RFC 2822).")
}

func (r *TrunkCredentialList) Annotate(a infer.Annotator) {
	a.Describe(&r, "Attachment of a SIP credential list to an Elastic SIP trunk, so the trunk authenticates outbound calls with the list's credentials. The resource ID is <trunkSid>/<credentialListSid>.")
}

var _ infer.CustomResource[TrunkCredentialListArgs, TrunkCredentialListState] = (*TrunkCredentialList)(nil)
var _ infer.CustomRead[TrunkCredentialListArgs, TrunkCredentialListState] = (*TrunkCredentialList)(nil)
var _ infer.CustomDelete[TrunkCredentialListState] = (*TrunkCredentialList)(nil)

func trunkCredentialListOutputs(st *TrunkCredentialListState, o obj) {
	st.Sid, st.FriendlyName = o.str("sid"), o.str("friendly_name")
	st.DateCreated, st.DateUpdated = o.str("date_created"), o.str("date_updated")
}

func (r *TrunkCredentialList) Create(ctx context.Context, req infer.CreateRequest[TrunkCredentialListArgs]) (infer.CreateResponse[TrunkCredentialListState], error) {
	a := req.Inputs
	id := a.TrunkSid + "/" + a.CredentialListSid
	if req.DryRun {
		return infer.CreateResponse[TrunkCredentialListState]{Output: TrunkCredentialListState{TrunkCredentialListArgs: a}}, nil
	}
	c, err := clientFor(ctx)
	if err != nil {
		return infer.CreateResponse[TrunkCredentialListState]{}, err
	}
	o, err := c.do(ctx, "POST", trunkingProduct, trunkChildPath(a.TrunkSid, "CredentialLists", ""), nil, url.Values{"CredentialListSid": {a.CredentialListSid}})
	if err != nil {
		return infer.CreateResponse[TrunkCredentialListState]{}, fmt.Errorf("attach %s to trunk %s: %w", a.CredentialListSid, a.TrunkSid, err)
	}
	st := TrunkCredentialListState{TrunkCredentialListArgs: a}
	trunkCredentialListOutputs(&st, o)
	return infer.CreateResponse[TrunkCredentialListState]{ID: id, Output: st}, nil
}

func (r *TrunkCredentialList) Read(ctx context.Context, req infer.ReadRequest[TrunkCredentialListArgs, TrunkCredentialListState]) (infer.ReadResponse[TrunkCredentialListArgs, TrunkCredentialListState], error) {
	type resp = infer.ReadResponse[TrunkCredentialListArgs, TrunkCredentialListState]
	trunk, sid, err := splitTrunkChildID(req.ID)
	if err != nil {
		return resp{}, err
	}
	c, err := clientFor(ctx)
	if err != nil {
		return resp{}, err
	}
	o, err := c.do(ctx, "GET", trunkingProduct, trunkChildPath(trunk, "CredentialLists", sid), nil, nil)
	if err != nil {
		if isNotFound(err) {
			return resp{}, nil
		}
		return resp{}, fmt.Errorf("read trunk attachment %s: %w", req.ID, err)
	}
	in := TrunkCredentialListArgs{TrunkSid: trunk, CredentialListSid: sid}
	st := TrunkCredentialListState{TrunkCredentialListArgs: in}
	trunkCredentialListOutputs(&st, o)
	return resp{ID: req.ID, Inputs: in, State: st}, nil
}

func (r *TrunkCredentialList) Delete(ctx context.Context, req infer.DeleteRequest[TrunkCredentialListState]) (infer.DeleteResponse, error) {
	c, err := clientFor(ctx)
	if err != nil {
		return infer.DeleteResponse{}, err
	}
	s := req.State
	if _, err := c.do(ctx, "DELETE", trunkingProduct, trunkChildPath(s.TrunkSid, "CredentialLists", s.CredentialListSid), nil, nil); err != nil && !isNotFound(err) {
		return infer.DeleteResponse{}, fmt.Errorf("detach %s from trunk %s: %w", s.CredentialListSid, s.TrunkSid, err)
	}
	return infer.DeleteResponse{}, nil
}

// ---------------------------------------------------------------- TrunkIpAccessControlList

// TrunkIpAccessControlList attaches an IP access control list to a trunk. Every input identifies the attachment, so
// any change replaces it; deleting detaches the list without deleting it.
type TrunkIpAccessControlList struct{}

// TrunkIpAccessControlListArgs are the inputs of a TrunkIpAccessControlList.
type TrunkIpAccessControlListArgs struct {
	TrunkSid               string `pulumi:"trunkSid" provider:"replaceOnChanges"`
	IpAccessControlListSid string `pulumi:"ipAccessControlListSid" provider:"replaceOnChanges"`
}

func (a *TrunkIpAccessControlListArgs) Annotate(an infer.Annotator) {
	an.Describe(&a.TrunkSid, "SID (TK...) of the trunk. Changing it replaces the resource.")
	an.Describe(&a.IpAccessControlListSid, "SID (AL...) of the IP access control list (see SipIpAccessControlList) to attach. Changing it replaces the resource.")
}

// TrunkIpAccessControlListState is the persisted state of a TrunkIpAccessControlList.
type TrunkIpAccessControlListState struct {
	TrunkIpAccessControlListArgs

	// Sid is the SID of the attached list.
	Sid string `pulumi:"sid"`
	// FriendlyName is the name of the attached list.
	FriendlyName string `pulumi:"friendlyName"`
	// DateCreated and DateUpdated are RFC 2822 timestamps of the attachment.
	DateCreated string `pulumi:"dateCreated"`
	DateUpdated string `pulumi:"dateUpdated"`
}

func (s *TrunkIpAccessControlListState) Annotate(a infer.Annotator) {
	a.Describe(&s.Sid, "SID of the attached list (AL...).")
	a.Describe(&s.FriendlyName, "Name of the attached list.")
	a.Describe(&s.DateCreated, "When the list was attached (RFC 2822).")
	a.Describe(&s.DateUpdated, "When the list was last updated (RFC 2822).")
}

func (r *TrunkIpAccessControlList) Annotate(a infer.Annotator) {
	a.Describe(&r, "Attachment of an IP access control list to an Elastic SIP trunk, so the trunk accepts outbound calls from the list's addresses. The resource ID is <trunkSid>/<ipAccessControlListSid>.")
}

var _ infer.CustomResource[TrunkIpAccessControlListArgs, TrunkIpAccessControlListState] = (*TrunkIpAccessControlList)(nil)
var _ infer.CustomRead[TrunkIpAccessControlListArgs, TrunkIpAccessControlListState] = (*TrunkIpAccessControlList)(nil)
var _ infer.CustomDelete[TrunkIpAccessControlListState] = (*TrunkIpAccessControlList)(nil)

func trunkIpAccessControlListOutputs(st *TrunkIpAccessControlListState, o obj) {
	st.Sid, st.FriendlyName = o.str("sid"), o.str("friendly_name")
	st.DateCreated, st.DateUpdated = o.str("date_created"), o.str("date_updated")
}

func (r *TrunkIpAccessControlList) Create(ctx context.Context, req infer.CreateRequest[TrunkIpAccessControlListArgs]) (infer.CreateResponse[TrunkIpAccessControlListState], error) {
	a := req.Inputs
	id := a.TrunkSid + "/" + a.IpAccessControlListSid
	if req.DryRun {
		return infer.CreateResponse[TrunkIpAccessControlListState]{Output: TrunkIpAccessControlListState{TrunkIpAccessControlListArgs: a}}, nil
	}
	c, err := clientFor(ctx)
	if err != nil {
		return infer.CreateResponse[TrunkIpAccessControlListState]{}, err
	}
	o, err := c.do(ctx, "POST", trunkingProduct, trunkChildPath(a.TrunkSid, "IpAccessControlLists", ""), nil, url.Values{"IpAccessControlListSid": {a.IpAccessControlListSid}})
	if err != nil {
		return infer.CreateResponse[TrunkIpAccessControlListState]{}, fmt.Errorf("attach %s to trunk %s: %w", a.IpAccessControlListSid, a.TrunkSid, err)
	}
	st := TrunkIpAccessControlListState{TrunkIpAccessControlListArgs: a}
	trunkIpAccessControlListOutputs(&st, o)
	return infer.CreateResponse[TrunkIpAccessControlListState]{ID: id, Output: st}, nil
}

func (r *TrunkIpAccessControlList) Read(ctx context.Context, req infer.ReadRequest[TrunkIpAccessControlListArgs, TrunkIpAccessControlListState]) (infer.ReadResponse[TrunkIpAccessControlListArgs, TrunkIpAccessControlListState], error) {
	type resp = infer.ReadResponse[TrunkIpAccessControlListArgs, TrunkIpAccessControlListState]
	trunk, sid, err := splitTrunkChildID(req.ID)
	if err != nil {
		return resp{}, err
	}
	c, err := clientFor(ctx)
	if err != nil {
		return resp{}, err
	}
	o, err := c.do(ctx, "GET", trunkingProduct, trunkChildPath(trunk, "IpAccessControlLists", sid), nil, nil)
	if err != nil {
		if isNotFound(err) {
			return resp{}, nil
		}
		return resp{}, fmt.Errorf("read trunk attachment %s: %w", req.ID, err)
	}
	in := TrunkIpAccessControlListArgs{TrunkSid: trunk, IpAccessControlListSid: sid}
	st := TrunkIpAccessControlListState{TrunkIpAccessControlListArgs: in}
	trunkIpAccessControlListOutputs(&st, o)
	return resp{ID: req.ID, Inputs: in, State: st}, nil
}

func (r *TrunkIpAccessControlList) Delete(ctx context.Context, req infer.DeleteRequest[TrunkIpAccessControlListState]) (infer.DeleteResponse, error) {
	c, err := clientFor(ctx)
	if err != nil {
		return infer.DeleteResponse{}, err
	}
	s := req.State
	if _, err := c.do(ctx, "DELETE", trunkingProduct, trunkChildPath(s.TrunkSid, "IpAccessControlLists", s.IpAccessControlListSid), nil, nil); err != nil && !isNotFound(err) {
		return infer.DeleteResponse{}, fmt.Errorf("detach %s from trunk %s: %w", s.IpAccessControlListSid, s.TrunkSid, err)
	}
	return infer.DeleteResponse{}, nil
}
