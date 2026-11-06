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

// GetTrunk looks up an Elastic SIP trunk by SID.
type GetTrunk struct{}

// GetTrunkArgs are the lookup arguments.
type GetTrunkArgs struct {
	TrunkSid string `pulumi:"trunkSid"`
}

func (a *GetTrunkArgs) Annotate(an infer.Annotator) {
	an.Describe(&a.TrunkSid, "SID (TK...) of the trunk.")
}

// GetTrunkResult is the looked-up trunk: the configuration Twilio reports plus its identity.
type GetTrunkResult struct {
	TrunkEffective

	Sid         string `pulumi:"sid"`
	AccountSid  string `pulumi:"accountSid"`
	AuthType    string `pulumi:"authType"`
	DateCreated string `pulumi:"dateCreated"`
	DateUpdated string `pulumi:"dateUpdated"`
}

func (a *GetTrunkResult) Annotate(an infer.Annotator) {
	an.Describe(&a.Sid, "The trunk SID (TK...).")
	an.Describe(&a.AccountSid, "SID of the account that owns the trunk.")
	an.Describe(&a.AuthType, "The authentication types of the trunk, IP_ACL and CREDENTIAL_LIST (comma-delimited when both).")
	an.Describe(&a.DateCreated, "When the trunk was created (RFC 2822).")
	an.Describe(&a.DateUpdated, "When the trunk was last updated (RFC 2822).")
}

func (f *GetTrunk) Annotate(a infer.Annotator) {
	a.Describe(&f, "Look up an Elastic SIP trunk by SID, including its termination domain and recording settings.")
}

func (f *GetTrunk) Invoke(ctx context.Context, req infer.FunctionRequest[GetTrunkArgs]) (infer.FunctionResponse[GetTrunkResult], error) {
	const fn = "getTrunk"
	if err := requireArg(fn, "trunkSid", req.Input.TrunkSid); err != nil {
		return infer.FunctionResponse[GetTrunkResult]{}, err
	}
	c, err := clientFor(ctx)
	if err != nil {
		return infer.FunctionResponse[GetTrunkResult]{}, err
	}
	o, err := c.do(ctx, "GET", trunkingProduct, trunkPath(req.Input.TrunkSid), nil, nil)
	if err != nil {
		return infer.FunctionResponse[GetTrunkResult]{}, lookupFailure(fn, "trunk "+req.Input.TrunkSid, err)
	}
	rec, err := c.do(ctx, "GET", trunkingProduct, trunkChildPath(req.Input.TrunkSid, "Recording", ""), nil, nil)
	if err != nil {
		return infer.FunctionResponse[GetTrunkResult]{}, lookupFailure(fn, "recording of trunk "+req.Input.TrunkSid, err)
	}
	st := TrunkState{}
	trunkOutputs(&st, o, rec)
	return infer.FunctionResponse[GetTrunkResult]{Output: GetTrunkResult{TrunkEffective: *st.Effective, Sid: st.Sid,
		AccountSid: st.AccountSid, AuthType: st.AuthType, DateCreated: st.DateCreated, DateUpdated: st.DateUpdated}}, nil
}

// GetTrunkOriginationUrl looks up an origination URL of a trunk.
type GetTrunkOriginationUrl struct{}

// GetTrunkOriginationUrlArgs are the lookup arguments.
type GetTrunkOriginationUrlArgs struct {
	TrunkSid          string `pulumi:"trunkSid"`
	OriginationUrlSid string `pulumi:"originationUrlSid"`
}

func (a *GetTrunkOriginationUrlArgs) Annotate(an infer.Annotator) {
	an.Describe(&a.TrunkSid, "SID (TK...) of the trunk.")
	an.Describe(&a.OriginationUrlSid, "SID (OU...) of the origination URL.")
}

// GetTrunkOriginationUrlResult is the looked-up origination URL.
type GetTrunkOriginationUrlResult struct {
	TrunkSid          string `pulumi:"trunkSid"`
	OriginationUrlSid string `pulumi:"originationUrlSid"`
	FriendlyName      string `pulumi:"friendlyName"`
	SipUrl            string `pulumi:"sipUrl"`
	Weight            int    `pulumi:"weight"`
	Priority          int    `pulumi:"priority"`
	Enabled           bool   `pulumi:"enabled"`
	DateCreated       string `pulumi:"dateCreated"`
	DateUpdated       string `pulumi:"dateUpdated"`
}

func (a *GetTrunkOriginationUrlResult) Annotate(an infer.Annotator) {
	an.Describe(&a.TrunkSid, "SID of the trunk.")
	an.Describe(&a.OriginationUrlSid, "SID of the origination URL.")
	an.Describe(&a.FriendlyName, "Name of the origination URL.")
	an.Describe(&a.SipUrl, "The SIP address Twilio routes origination calls to.")
	an.Describe(&a.Weight, "Relative share of the load among URLs of the same priority.")
	an.Describe(&a.Priority, "Relative importance of the URL; the lowest number is the most important.")
	an.Describe(&a.Enabled, "Whether the URL is enabled.")
	an.Describe(&a.DateCreated, "When the URL was created (RFC 2822).")
	an.Describe(&a.DateUpdated, "When the URL was last updated (RFC 2822).")
}

func (f *GetTrunkOriginationUrl) Annotate(a infer.Annotator) {
	a.Describe(&f, "Look up an origination URL of an Elastic SIP trunk.")
}

func (f *GetTrunkOriginationUrl) Invoke(ctx context.Context, req infer.FunctionRequest[GetTrunkOriginationUrlArgs]) (infer.FunctionResponse[GetTrunkOriginationUrlResult], error) {
	const fn = "getTrunkOriginationUrl"
	in := req.Input
	for _, a := range [][2]string{{"trunkSid", in.TrunkSid}, {"originationUrlSid", in.OriginationUrlSid}} {
		if err := requireArg(fn, a[0], a[1]); err != nil {
			return infer.FunctionResponse[GetTrunkOriginationUrlResult]{}, err
		}
	}
	c, err := clientFor(ctx)
	if err != nil {
		return infer.FunctionResponse[GetTrunkOriginationUrlResult]{}, err
	}
	o, err := c.do(ctx, "GET", trunkingProduct, trunkChildPath(in.TrunkSid, "OriginationUrls", in.OriginationUrlSid), nil, nil)
	if err != nil {
		return infer.FunctionResponse[GetTrunkOriginationUrlResult]{}, lookupFailure(fn, "origination URL "+in.OriginationUrlSid+" on trunk "+in.TrunkSid, err)
	}
	res := GetTrunkOriginationUrlResult{TrunkSid: in.TrunkSid, OriginationUrlSid: in.OriginationUrlSid, FriendlyName: o.str("friendly_name"),
		SipUrl: o.str("sip_url"), Weight: intOr(o.optInt("weight"), 0), Priority: intOr(o.optInt("priority"), 0),
		DateCreated: o.str("date_created"), DateUpdated: o.str("date_updated")}
	if b := o.optBool("enabled"); b != nil {
		res.Enabled = *b
	}
	return infer.FunctionResponse[GetTrunkOriginationUrlResult]{Output: res}, nil
}

// GetTrunkCredentialList looks up a SIP credential list attached to a trunk.
type GetTrunkCredentialList struct{}

// GetTrunkCredentialListArgs are the lookup arguments.
type GetTrunkCredentialListArgs struct {
	TrunkSid          string `pulumi:"trunkSid"`
	CredentialListSid string `pulumi:"credentialListSid"`
}

func (a *GetTrunkCredentialListArgs) Annotate(an infer.Annotator) {
	an.Describe(&a.TrunkSid, "SID (TK...) of the trunk.")
	an.Describe(&a.CredentialListSid, "SID (CL...) of the attached list.")
}

// GetTrunkCredentialListResult is the looked-up attachment.
type GetTrunkCredentialListResult struct {
	TrunkSid          string `pulumi:"trunkSid"`
	CredentialListSid string `pulumi:"credentialListSid"`
	FriendlyName      string `pulumi:"friendlyName"`
	DateCreated       string `pulumi:"dateCreated"`
	DateUpdated       string `pulumi:"dateUpdated"`
}

func (a *GetTrunkCredentialListResult) Annotate(an infer.Annotator) {
	an.Describe(&a.TrunkSid, "SID of the trunk.")
	an.Describe(&a.CredentialListSid, "SID of the attached list.")
	an.Describe(&a.FriendlyName, "Name of the attached list.")
	an.Describe(&a.DateCreated, "When the list was attached (RFC 2822).")
	an.Describe(&a.DateUpdated, "When the list was last updated (RFC 2822).")
}

func (f *GetTrunkCredentialList) Annotate(a infer.Annotator) {
	a.Describe(&f, "Look up a SIP credential list attached to an Elastic SIP trunk.")
}

func (f *GetTrunkCredentialList) Invoke(ctx context.Context, req infer.FunctionRequest[GetTrunkCredentialListArgs]) (infer.FunctionResponse[GetTrunkCredentialListResult], error) {
	const fn = "getTrunkCredentialList"
	in := req.Input
	for _, a := range [][2]string{{"trunkSid", in.TrunkSid}, {"credentialListSid", in.CredentialListSid}} {
		if err := requireArg(fn, a[0], a[1]); err != nil {
			return infer.FunctionResponse[GetTrunkCredentialListResult]{}, err
		}
	}
	c, err := clientFor(ctx)
	if err != nil {
		return infer.FunctionResponse[GetTrunkCredentialListResult]{}, err
	}
	o, err := c.do(ctx, "GET", trunkingProduct, trunkChildPath(in.TrunkSid, "CredentialLists", in.CredentialListSid), nil, nil)
	if err != nil {
		return infer.FunctionResponse[GetTrunkCredentialListResult]{}, lookupFailure(fn, in.CredentialListSid+" on trunk "+in.TrunkSid, err)
	}
	return infer.FunctionResponse[GetTrunkCredentialListResult]{Output: GetTrunkCredentialListResult{TrunkSid: in.TrunkSid, CredentialListSid: in.CredentialListSid,
		FriendlyName: o.str("friendly_name"), DateCreated: o.str("date_created"), DateUpdated: o.str("date_updated")}}, nil
}

// GetTrunkIpAccessControlList looks up an IP access control list attached to a trunk.
type GetTrunkIpAccessControlList struct{}

// GetTrunkIpAccessControlListArgs are the lookup arguments.
type GetTrunkIpAccessControlListArgs struct {
	TrunkSid               string `pulumi:"trunkSid"`
	IpAccessControlListSid string `pulumi:"ipAccessControlListSid"`
}

func (a *GetTrunkIpAccessControlListArgs) Annotate(an infer.Annotator) {
	an.Describe(&a.TrunkSid, "SID (TK...) of the trunk.")
	an.Describe(&a.IpAccessControlListSid, "SID (AL...) of the attached list.")
}

// GetTrunkIpAccessControlListResult is the looked-up attachment.
type GetTrunkIpAccessControlListResult struct {
	TrunkSid               string `pulumi:"trunkSid"`
	IpAccessControlListSid string `pulumi:"ipAccessControlListSid"`
	FriendlyName           string `pulumi:"friendlyName"`
	DateCreated            string `pulumi:"dateCreated"`
	DateUpdated            string `pulumi:"dateUpdated"`
}

func (a *GetTrunkIpAccessControlListResult) Annotate(an infer.Annotator) {
	an.Describe(&a.TrunkSid, "SID of the trunk.")
	an.Describe(&a.IpAccessControlListSid, "SID of the attached list.")
	an.Describe(&a.FriendlyName, "Name of the attached list.")
	an.Describe(&a.DateCreated, "When the list was attached (RFC 2822).")
	an.Describe(&a.DateUpdated, "When the list was last updated (RFC 2822).")
}

func (f *GetTrunkIpAccessControlList) Annotate(a infer.Annotator) {
	a.Describe(&f, "Look up an IP access control list attached to an Elastic SIP trunk.")
}

func (f *GetTrunkIpAccessControlList) Invoke(ctx context.Context, req infer.FunctionRequest[GetTrunkIpAccessControlListArgs]) (infer.FunctionResponse[GetTrunkIpAccessControlListResult], error) {
	const fn = "getTrunkIpAccessControlList"
	in := req.Input
	for _, a := range [][2]string{{"trunkSid", in.TrunkSid}, {"ipAccessControlListSid", in.IpAccessControlListSid}} {
		if err := requireArg(fn, a[0], a[1]); err != nil {
			return infer.FunctionResponse[GetTrunkIpAccessControlListResult]{}, err
		}
	}
	c, err := clientFor(ctx)
	if err != nil {
		return infer.FunctionResponse[GetTrunkIpAccessControlListResult]{}, err
	}
	o, err := c.do(ctx, "GET", trunkingProduct, trunkChildPath(in.TrunkSid, "IpAccessControlLists", in.IpAccessControlListSid), nil, nil)
	if err != nil {
		return infer.FunctionResponse[GetTrunkIpAccessControlListResult]{}, lookupFailure(fn, in.IpAccessControlListSid+" on trunk "+in.TrunkSid, err)
	}
	return infer.FunctionResponse[GetTrunkIpAccessControlListResult]{Output: GetTrunkIpAccessControlListResult{TrunkSid: in.TrunkSid, IpAccessControlListSid: in.IpAccessControlListSid,
		FriendlyName: o.str("friendly_name"), DateCreated: o.str("date_created"), DateUpdated: o.str("date_updated")}}, nil
}
