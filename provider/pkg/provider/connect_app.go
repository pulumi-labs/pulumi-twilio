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

	p "github.com/pulumi/pulumi-go-provider"
	"github.com/pulumi/pulumi-go-provider/infer"
)

// ConnectApp adopts a Twilio Connect app that was created in the Console and manages its
// listing details. It never creates or deletes a Connect app.
type ConnectApp struct{}

// ConnectAppArgs are the inputs of a ConnectApp.
type ConnectAppArgs struct {
	ConnectAppSid             string    `pulumi:"connectAppSid" provider:"replaceOnChanges"`
	AccountSid                *string   `pulumi:"accountSid,optional" provider:"replaceOnChanges"`
	FriendlyName              *string   `pulumi:"friendlyName,optional"`
	AuthorizeRedirectUrl      *string   `pulumi:"authorizeRedirectUrl,optional"`
	CompanyName               *string   `pulumi:"companyName,optional"`
	DeauthorizeCallbackUrl    *string   `pulumi:"deauthorizeCallbackUrl,optional"`
	DeauthorizeCallbackMethod *string   `pulumi:"deauthorizeCallbackMethod,optional"`
	Description               *string   `pulumi:"description,optional"`
	HomepageUrl               *string   `pulumi:"homepageUrl,optional"`
	Permissions               *[]string `pulumi:"permissions,optional"`
}

func (a *ConnectAppArgs) props() props {
	return props{
		strs: []strProp{
			{"FriendlyName", "friendly_name", &a.FriendlyName},
			{"AuthorizeRedirectUrl", "authorize_redirect_url", &a.AuthorizeRedirectUrl},
			{"CompanyName", "company_name", &a.CompanyName},
			{"DeauthorizeCallbackUrl", "deauthorize_callback_url", &a.DeauthorizeCallbackUrl},
			{"DeauthorizeCallbackMethod", "deauthorize_callback_method", &a.DeauthorizeCallbackMethod},
			{"Description", "description", &a.Description},
			{"HomepageUrl", "homepage_url", &a.HomepageUrl},
		},
		lists: []listProp{{"Permissions", "permissions", &a.Permissions}},
	}
}

func (a *ConnectAppArgs) Annotate(an infer.Annotator) {
	an.Describe(&a.ConnectAppSid, "SID (CN...) of the Connect app. Twilio creates Connect apps only in the Console, so this resource "+
		"adopts an existing one. Changing it replaces the resource (another app is adopted).")
	an.Describe(&a.AccountSid, "SID of the account or subaccount that owns the Connect app. Defaults to the provider account. "+
		"Changing it replaces the resource. To import an app owned by a subaccount use the ID <accountSid>/<connectAppSid>.")
	an.Describe(&a.FriendlyName, "Human-readable name of the Connect app. Removing it leaves the name unchanged.")
	an.Describe(&a.AuthorizeRedirectUrl, "URL Twilio redirects a user to after they authorize the app. Removing it leaves the URL unchanged.")
	an.Describe(&a.CompanyName, "Company name shown on the authorization page. Removing it leaves the name unchanged.")
	an.Describe(&a.DeauthorizeCallbackUrl, "URL Twilio requests when a user deauthorizes the app. Removing it clears the URL.")
	an.Describe(&a.DeauthorizeCallbackMethod, "HTTP method for deauthorizeCallbackUrl: GET or POST.")
	an.Describe(&a.Description, "Description shown on the authorization page. Removing it leaves the text unchanged.")
	an.Describe(&a.HomepageUrl, "Public URL of the company or app homepage. Removing it leaves the URL unchanged.")
	an.Describe(&a.Permissions, "Permissions the app requests from its users: get-all and post-all. Removing the list leaves the permissions unchanged.")
}

// ConnectAppState is the persisted state of a ConnectApp.
type ConnectAppState struct {
	ConnectAppArgs

	// Sid is the Connect app SID (CN...) and the resource ID.
	Sid string `pulumi:"sid"`
	// OwnerAccountSid is the account that owns the app.
	OwnerAccountSid string `pulumi:"ownerAccountSid"`
	// Effective is the configuration Twilio reports, including fields the program did not set.
	Effective *ConnectAppEffective `pulumi:"effective,optional"`
}

func (s *ConnectAppState) Annotate(a infer.Annotator) {
	a.Describe(&s.Sid, "The Connect app SID (CN...).")
	a.Describe(&s.OwnerAccountSid, "SID of the account that owns the Connect app.")
	a.Describe(&s.Effective, "The configuration Twilio currently reports for the Connect app, including values changed "+
		"outside Pulumi and fields the program does not set. Inputs the program leaves unset are not overwritten by these values.")
}

func (r *ConnectApp) Annotate(a infer.Annotator) {
	a.Describe(&r, "Listing details of a Twilio Connect app that already exists. The resource adopts the app; it never creates "+
		"one (Twilio offers that only in the Console), and destroying it does NOT delete the app (it only stops managing it).")
}

var _ infer.CustomResource[ConnectAppArgs, ConnectAppState] = (*ConnectApp)(nil)
var _ infer.CustomCheck[ConnectAppArgs] = (*ConnectApp)(nil)
var _ infer.CustomRead[ConnectAppArgs, ConnectAppState] = (*ConnectApp)(nil)
var _ infer.CustomUpdate[ConnectAppArgs, ConnectAppState] = (*ConnectApp)(nil)
var _ infer.CustomDelete[ConnectAppState] = (*ConnectApp)(nil)

func (r *ConnectApp) Check(ctx context.Context, req infer.CheckRequest) (infer.CheckResponse[ConnectAppArgs], error) {
	args, failures, err := infer.DefaultCheck[ConnectAppArgs](ctx, req.NewInputs)
	if err != nil || len(failures) > 0 {
		return infer.CheckResponse[ConnectAppArgs]{Inputs: args, Failures: failures}, err
	}
	unk := unknowns(req.NewInputs)
	if !unk("connectAppSid") && args.ConnectAppSid == "" {
		failures = append(failures, p.CheckFailure{Property: "connectAppSid", Reason: "connectAppSid must be set"})
	}
	failures = checkMethods(failures, unk, map[string]*string{"deauthorizeCallbackMethod": args.DeauthorizeCallbackMethod})
	if args.Permissions != nil && !unk("permissions") {
		for _, perm := range *args.Permissions {
			if perm != "get-all" && perm != "post-all" {
				failures = append(failures, p.CheckFailure{Property: "permissions", Reason: fmt.Sprintf("permission %q must be get-all or post-all", perm)})
			}
		}
	}
	return infer.CheckResponse[ConnectAppArgs]{Inputs: args, Failures: failures}, nil
}

// ConnectAppEffective is the configuration Twilio reports for a ConnectApp: every field of the inputs, filled from the API response.
type ConnectAppEffective ConnectAppArgs

// Annotate reuses the input descriptions.
func (e *ConnectAppEffective) Annotate(a infer.Annotator) { (*ConnectAppArgs)(e).Annotate(a) }

func connectAppEffective(o obj) *ConnectAppEffective {
	e := ConnectAppArgs{}
	e.props().fillAll(o)
	v := ConnectAppEffective(e)
	return &v
}

func (r *ConnectApp) Create(ctx context.Context, req infer.CreateRequest[ConnectAppArgs]) (infer.CreateResponse[ConnectAppState], error) {
	a := req.Inputs
	if req.DryRun {
		return infer.CreateResponse[ConnectAppState]{ID: "", Output: ConnectAppState{ConnectAppArgs: a}}, nil
	}
	c, err := clientFor(ctx)
	if err != nil {
		return infer.CreateResponse[ConnectAppState]{}, err
	}
	acct := c.acct(a.AccountSid)
	cur, err := fetchAdopted(ctx, c, acct, connectAppColl, a.ConnectAppSid)
	if err != nil {
		return infer.CreateResponse[ConnectAppState]{}, fmt.Errorf("adopt Connect app: %w", err)
	}
	sid := cur.str("sid")
	st := ConnectAppState{ConnectAppArgs: a, Sid: sid, OwnerAccountSid: acct, Effective: connectAppEffective(cur)}
	form := url.Values{}
	a.props().encode(form, nil)
	if len(form) > 0 {
		o, err := c.do(ctx, "POST", "api", acctPath(acct, connectAppColl.coll, sid), nil, form)
		if err != nil {
			// The app is adopted but not configured: save it so the next update retries.
			return infer.CreateResponse[ConnectAppState]{ID: scopedID(c, acct, sid), Output: st},
				infer.ResourceInitFailedError{Reasons: []string{fmt.Sprintf("configure Connect app %s: %v", sid, err)}}
		}
		st.Effective = connectAppEffective(o)
	}
	return infer.CreateResponse[ConnectAppState]{ID: scopedID(c, acct, sid), Output: st}, nil
}

func (r *ConnectApp) Read(ctx context.Context, req infer.ReadRequest[ConnectAppArgs, ConnectAppState]) (infer.ReadResponse[ConnectAppArgs, ConnectAppState], error) {
	c, err := clientFor(ctx)
	if err != nil {
		return infer.ReadResponse[ConnectAppArgs, ConnectAppState]{}, err
	}
	acct, sid := resolveScope(c, req.ID, req.State.OwnerAccountSid)
	o, err := c.do(ctx, "GET", "api", acctPath(acct, connectAppColl.coll, sid), nil, nil)
	if err != nil {
		if isNotFound(err) {
			if err := notFoundOnImport("connect app", req.ID, req.Inputs, req.State); err != nil {
				return infer.ReadResponse[ConnectAppArgs, ConnectAppState]{}, err
			}
			return infer.ReadResponse[ConnectAppArgs, ConnectAppState]{}, nil
		}
		return infer.ReadResponse[ConnectAppArgs, ConnectAppState]{}, fmt.Errorf("read Connect app %s: %w", req.ID, err)
	}
	in := req.Inputs
	if in.ConnectAppSid == "" { // an import
		in.ConnectAppSid = sid
		if acct != c.accountSid {
			in.AccountSid = &acct
		}
		in.props().adopt(o)
	} else {
		in.props().refreshSet(o)
	}
	st := ConnectAppState{ConnectAppArgs: in, Sid: sid, OwnerAccountSid: derefOr(nonEmpty(o.optStr("account_sid")), acct),
		Effective: connectAppEffective(o)}
	return infer.ReadResponse[ConnectAppArgs, ConnectAppState]{ID: req.ID, Inputs: in, State: st}, nil
}

func (r *ConnectApp) Update(ctx context.Context, req infer.UpdateRequest[ConnectAppArgs, ConnectAppState]) (infer.UpdateResponse[ConnectAppState], error) {
	a := req.Inputs
	st := req.State
	st.ConnectAppArgs = a
	if req.DryRun { // a preview needs no credentials
		return infer.UpdateResponse[ConnectAppState]{Output: st}, nil
	}
	c, err := clientFor(ctx)
	if err != nil {
		return infer.UpdateResponse[ConnectAppState]{}, err
	}
	acct, sid := resolveScope(c, req.ID, req.State.OwnerAccountSid)
	form := url.Values{}
	prev := req.State.ConnectAppArgs.props()
	a.props().encode(form, &prev)
	if len(form) > 0 {
		o, err := c.do(ctx, "POST", "api", acctPath(acct, connectAppColl.coll, sid), nil, form)
		if err != nil {
			return infer.UpdateResponse[ConnectAppState]{}, fmt.Errorf("update Connect app %s: %w", req.ID, err)
		}
		st.Effective = connectAppEffective(o)
	}
	return infer.UpdateResponse[ConnectAppState]{Output: st}, nil
}

// Delete only stops managing the Connect app: it is never deleted.
func (r *ConnectApp) Delete(ctx context.Context, req infer.DeleteRequest[ConnectAppState]) (infer.DeleteResponse, error) {
	return infer.DeleteResponse{}, nil
}
