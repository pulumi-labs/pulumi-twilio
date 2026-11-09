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
	"encoding/json"
	"fmt"
	"net/url"
	"regexp"

	p "github.com/pulumi/pulumi-go-provider"
	"github.com/pulumi/pulumi-go-provider/infer"
)

// An assigned add-on is an installed Marketplace add-on switched on for one incoming phone number.
// The 2010-04-01 API has create (InstalledAddOnSid), fetch, list and delete, and no update call,
// so every input replaces the resource. Fetch does not echo InstalledAddOnSid, so an imported
// resource has no installedAddOnSid until the program sets one (see Diff).

var installedAddOnSidRE = regexp.MustCompile(`^XE[0-9a-fA-F]{32}$`)

// assignedAddOnPath is the collection (sid empty) or instance path of an add-on assigned to a number.
func assignedAddOnPath(account, number, sid string) string {
	return acctPath(account, "IncomingPhoneNumbers/"+url.PathEscape(number)+"/AssignedAddOns", sid)
}

// assignedAddOnConfiguration renders the add-on's configuration object as a JSON string.
func assignedAddOnConfiguration(o obj) string {
	v, ok := o["configuration"]
	if !ok || v == nil {
		return ""
	}
	if s, ok := v.(string); ok {
		return s
	}
	b, err := json.Marshal(v)
	if err != nil {
		return ""
	}
	return string(b)
}

// IncomingPhoneNumberAssignedAddOn assigns an installed Marketplace add-on to an incoming phone number.
type IncomingPhoneNumberAssignedAddOn struct{}

// IncomingPhoneNumberAssignedAddOnArgs are the inputs of an IncomingPhoneNumberAssignedAddOn.
type IncomingPhoneNumberAssignedAddOnArgs struct {
	PhoneNumberSid    string  `pulumi:"phoneNumberSid" provider:"replaceOnChanges"`
	InstalledAddOnSid string  `pulumi:"installedAddOnSid" provider:"replaceOnChanges"`
	AccountSid        *string `pulumi:"accountSid,optional" provider:"replaceOnChanges"`
}

func (a *IncomingPhoneNumberAssignedAddOnArgs) Annotate(an infer.Annotator) {
	an.Describe(&a.PhoneNumberSid, "SID (PN...) of the incoming phone number the add-on is assigned to. Changing it replaces the resource.")
	an.Describe(&a.InstalledAddOnSid, "SID (XE...) of the installed add-on to assign. Twilio has no update call, so changing it replaces the resource. "+
		"Twilio does not return it, so an imported assignment has none until the program sets it, which then plans an update that only records it (no Twilio call).")
	an.Describe(&a.AccountSid, "SID of the account or subaccount that owns the phone number. Defaults to the provider account. "+
		"Changing it replaces the resource. To import an assignment on a subaccount's number use the ID <accountSid>/<phoneNumberSid>/<sid>.")
}

// IncomingPhoneNumberAssignedAddOnState is the persisted state of an IncomingPhoneNumberAssignedAddOn.
type IncomingPhoneNumberAssignedAddOnState struct {
	IncomingPhoneNumberAssignedAddOnArgs

	// Sid is the assignment SID (XE...).
	Sid string `pulumi:"sid"`
	// OwnerAccountSid is the account that owns the phone number.
	OwnerAccountSid string `pulumi:"ownerAccountSid"`
	// FriendlyName, Description and UniqueName describe the add-on.
	FriendlyName string `pulumi:"friendlyName"`
	Description  string `pulumi:"description"`
	UniqueName   string `pulumi:"uniqueName"`
	// Configuration is the add-on installation's configuration as a JSON string.
	Configuration string `pulumi:"configuration"`
	// DateCreated and DateUpdated are RFC 2822 timestamps.
	DateCreated string `pulumi:"dateCreated"`
	DateUpdated string `pulumi:"dateUpdated"`
}

func (s *IncomingPhoneNumberAssignedAddOnState) Annotate(a infer.Annotator) {
	a.Describe(&s.Sid, "The assignment SID (XE...).")
	a.Describe(&s.OwnerAccountSid, "SID of the account that owns the phone number.")
	a.Describe(&s.FriendlyName, "The add-on's friendly name.")
	a.Describe(&s.Description, "A short description of what the add-on provides.")
	a.Describe(&s.UniqueName, "The add-on's application-defined unique name.")
	a.Describe(&s.Configuration, "The current configuration of the add-on installation, as a JSON string.")
	a.Describe(&s.DateCreated, "When the assignment was created (RFC 2822).")
	a.Describe(&s.DateUpdated, "When the assignment was last updated (RFC 2822).")
}

func (r *IncomingPhoneNumberAssignedAddOn) Annotate(a infer.Annotator) {
	a.Describe(&r, "An installed Marketplace add-on assigned to a Twilio incoming phone number. Twilio has no update call, so every input replaces the assignment. "+
		"The resource ID is <phoneNumberSid>/<sid> (<accountSid>/<phoneNumberSid>/<sid> for a subaccount's number).")
}

var _ infer.CustomResource[IncomingPhoneNumberAssignedAddOnArgs, IncomingPhoneNumberAssignedAddOnState] = (*IncomingPhoneNumberAssignedAddOn)(nil)
var _ infer.CustomCheck[IncomingPhoneNumberAssignedAddOnArgs] = (*IncomingPhoneNumberAssignedAddOn)(nil)
var _ infer.CustomDiff[IncomingPhoneNumberAssignedAddOnArgs, IncomingPhoneNumberAssignedAddOnState] = (*IncomingPhoneNumberAssignedAddOn)(nil)
var _ infer.CustomUpdate[IncomingPhoneNumberAssignedAddOnArgs, IncomingPhoneNumberAssignedAddOnState] = (*IncomingPhoneNumberAssignedAddOn)(nil)
var _ infer.CustomRead[IncomingPhoneNumberAssignedAddOnArgs, IncomingPhoneNumberAssignedAddOnState] = (*IncomingPhoneNumberAssignedAddOn)(nil)
var _ infer.CustomDelete[IncomingPhoneNumberAssignedAddOnState] = (*IncomingPhoneNumberAssignedAddOn)(nil)

func (r *IncomingPhoneNumberAssignedAddOn) Check(ctx context.Context, req infer.CheckRequest) (infer.CheckResponse[IncomingPhoneNumberAssignedAddOnArgs], error) {
	args, failures, err := infer.DefaultCheck[IncomingPhoneNumberAssignedAddOnArgs](ctx, req.NewInputs)
	if err != nil || len(failures) > 0 {
		return infer.CheckResponse[IncomingPhoneNumberAssignedAddOnArgs]{Inputs: args, Failures: failures}, err
	}
	unk := unknowns(req.NewInputs)
	if !unk("phoneNumberSid") && args.PhoneNumberSid == "" {
		failures = append(failures, p.CheckFailure{Property: "phoneNumberSid", Reason: "phoneNumberSid must be set"})
	}
	// An imported assignment has an empty installedAddOnSid (Twilio never returns it) and the engine runs Check
	// on the imported inputs, so empty is not a failure here; Create refuses it.
	if !unk("installedAddOnSid") && args.InstalledAddOnSid != "" && !installedAddOnSidRE.MatchString(args.InstalledAddOnSid) {
		failures = append(failures, p.CheckFailure{Property: "installedAddOnSid", Reason: "installedAddOnSid must be an installed add-on SID (XE followed by 32 hex digits)"})
	}
	return infer.CheckResponse[IncomingPhoneNumberAssignedAddOnArgs]{Inputs: args, Failures: failures}, nil
}

// Diff replaces on any change, except that an imported assignment (no recorded installedAddOnSid)
// adopts the program's value without replacement: Twilio never says which installation it came from.
// The adoption is an Update that makes no API call and only records the value, so a later change to
// another installation is seen as a replacement even before a refresh.
func (r *IncomingPhoneNumberAssignedAddOn) Diff(_ context.Context, req infer.DiffRequest[IncomingPhoneNumberAssignedAddOnArgs, IncomingPhoneNumberAssignedAddOnState]) (infer.DiffResponse, error) {
	o, n := req.State.IncomingPhoneNumberAssignedAddOnArgs, req.Inputs
	detail := map[string]p.PropertyDiff{}
	if o.PhoneNumberSid != n.PhoneNumberSid {
		detail["phoneNumberSid"] = p.PropertyDiff{Kind: p.UpdateReplace}
	}
	switch {
	case o.InstalledAddOnSid == "" && n.InstalledAddOnSid != "":
		detail["installedAddOnSid"] = p.PropertyDiff{Kind: p.Update}
	case o.InstalledAddOnSid != n.InstalledAddOnSid:
		detail["installedAddOnSid"] = p.PropertyDiff{Kind: p.UpdateReplace}
	}
	if derefStr(o.AccountSid) != derefStr(n.AccountSid) {
		detail["accountSid"] = p.PropertyDiff{Kind: p.UpdateReplace}
	}
	return infer.DiffResponse{HasChanges: len(detail) > 0, DetailedDiff: detail}, nil
}

func assignedAddOnState(in IncomingPhoneNumberAssignedAddOnArgs, acct string, o obj) IncomingPhoneNumberAssignedAddOnState {
	return IncomingPhoneNumberAssignedAddOnState{
		IncomingPhoneNumberAssignedAddOnArgs: in, Sid: o.str("sid"), OwnerAccountSid: acct,
		FriendlyName: o.str("friendly_name"), Description: o.str("description"), UniqueName: o.str("unique_name"),
		Configuration: assignedAddOnConfiguration(o), DateCreated: o.str("date_created"), DateUpdated: o.str("date_updated"),
	}
}

func (r *IncomingPhoneNumberAssignedAddOn) Create(ctx context.Context, req infer.CreateRequest[IncomingPhoneNumberAssignedAddOnArgs]) (infer.CreateResponse[IncomingPhoneNumberAssignedAddOnState], error) {
	type resp = infer.CreateResponse[IncomingPhoneNumberAssignedAddOnState]
	a := req.Inputs
	if req.DryRun {
		return resp{Output: IncomingPhoneNumberAssignedAddOnState{IncomingPhoneNumberAssignedAddOnArgs: a}}, nil
	}
	if a.InstalledAddOnSid == "" {
		return resp{}, fmt.Errorf("installedAddOnSid must be set")
	}
	c, err := clientFor(ctx)
	if err != nil {
		return resp{}, err
	}
	acct := c.acct(a.AccountSid)
	o, err := c.do(ctx, "POST", "api", assignedAddOnPath(acct, a.PhoneNumberSid, ""), nil, url.Values{"InstalledAddOnSid": {a.InstalledAddOnSid}})
	if err != nil {
		return resp{}, fmt.Errorf("assign add-on %s to phone number %s: %w", a.InstalledAddOnSid, a.PhoneNumberSid, err)
	}
	st := assignedAddOnState(a, acct, o)
	return resp{ID: childID(c, acct, a.PhoneNumberSid, st.Sid), Output: st}, nil
}

func (r *IncomingPhoneNumberAssignedAddOn) Read(ctx context.Context, req infer.ReadRequest[IncomingPhoneNumberAssignedAddOnArgs, IncomingPhoneNumberAssignedAddOnState]) (infer.ReadResponse[IncomingPhoneNumberAssignedAddOnArgs, IncomingPhoneNumberAssignedAddOnState], error) {
	type resp = infer.ReadResponse[IncomingPhoneNumberAssignedAddOnArgs, IncomingPhoneNumberAssignedAddOnState]
	c, err := clientFor(ctx)
	if err != nil {
		return resp{}, err
	}
	acct, number, sid, err := resolveChildScope(c, req.ID, req.State.OwnerAccountSid)
	if err != nil {
		return resp{}, err
	}
	o, err := c.do(ctx, "GET", "api", assignedAddOnPath(acct, number, sid), nil, nil)
	if err != nil {
		if isNotFound(err) {
			return resp{}, nil
		}
		return resp{}, fmt.Errorf("read assigned add-on %s: %w", req.ID, err)
	}
	in := req.Inputs
	if in.PhoneNumberSid == "" && acct != c.accountSid { // import
		in.AccountSid = &acct
	}
	in.PhoneNumberSid = number
	if in.InstalledAddOnSid == "" {
		in.InstalledAddOnSid = req.State.InstalledAddOnSid
	}
	st := assignedAddOnState(in, acct, o)
	st.Sid = sid
	return resp{ID: req.ID, Inputs: in, State: st}, nil
}

// Update only records the installation SID adopted after an import: Twilio has no update call, and
// Diff reports every other change as a replacement.
func (r *IncomingPhoneNumberAssignedAddOn) Update(_ context.Context, req infer.UpdateRequest[IncomingPhoneNumberAssignedAddOnArgs, IncomingPhoneNumberAssignedAddOnState]) (infer.UpdateResponse[IncomingPhoneNumberAssignedAddOnState], error) {
	st := req.State
	st.IncomingPhoneNumberAssignedAddOnArgs = req.Inputs
	return infer.UpdateResponse[IncomingPhoneNumberAssignedAddOnState]{Output: st}, nil
}

func (r *IncomingPhoneNumberAssignedAddOn) Delete(ctx context.Context, req infer.DeleteRequest[IncomingPhoneNumberAssignedAddOnState]) (infer.DeleteResponse, error) {
	c, err := clientFor(ctx)
	if err != nil {
		return infer.DeleteResponse{}, err
	}
	acct, number, sid, err := resolveChildScope(c, req.ID, req.State.OwnerAccountSid)
	if err != nil {
		return infer.DeleteResponse{}, err
	}
	if _, err := c.do(ctx, "DELETE", "api", assignedAddOnPath(acct, number, sid), nil, nil); err != nil && !isNotFound(err) {
		return infer.DeleteResponse{}, fmt.Errorf("delete assigned add-on %s: %w", req.ID, err)
	}
	return infer.DeleteResponse{}, nil
}

// ------------------------------------- getIncomingPhoneNumberAssignedAddOn

// GetIncomingPhoneNumberAssignedAddOn looks up an add-on assigned to an incoming phone number, with its extensions.
type GetIncomingPhoneNumberAssignedAddOn struct{}

// GetIncomingPhoneNumberAssignedAddOnArgs are the lookup arguments.
type GetIncomingPhoneNumberAssignedAddOnArgs struct {
	PhoneNumberSid   string  `pulumi:"phoneNumberSid"`
	AssignedAddOnSid string  `pulumi:"assignedAddOnSid"`
	AccountSid       *string `pulumi:"accountSid,optional"`
}

func (a *GetIncomingPhoneNumberAssignedAddOnArgs) Annotate(an infer.Annotator) {
	an.Describe(&a.PhoneNumberSid, "SID (PN...) of the incoming phone number.")
	an.Describe(&a.AssignedAddOnSid, "SID (XE...) of the assignment.")
	an.Describe(&a.AccountSid, "SID of the account that owns the number. Defaults to the provider account.")
}

// AssignedAddOnExtension is one extension of an assigned add-on.
type AssignedAddOnExtension struct {
	Sid          string `pulumi:"sid"`
	FriendlyName string `pulumi:"friendlyName"`
	ProductName  string `pulumi:"productName"`
	UniqueName   string `pulumi:"uniqueName"`
	Enabled      bool   `pulumi:"enabled"`
}

func (a *AssignedAddOnExtension) Annotate(an infer.Annotator) {
	an.Describe(&a.Sid, "The extension SID (XF...).")
	an.Describe(&a.FriendlyName, "The extension's friendly name.")
	an.Describe(&a.ProductName, "The name of the add-on product the extension belongs to.")
	an.Describe(&a.UniqueName, "The extension's application-defined unique name.")
	an.Describe(&a.Enabled, "Whether the extension is enabled.")
}

// GetIncomingPhoneNumberAssignedAddOnResult is the looked-up assignment.
type GetIncomingPhoneNumberAssignedAddOnResult struct {
	Sid             string                   `pulumi:"sid"`
	OwnerAccountSid string                   `pulumi:"ownerAccountSid"`
	PhoneNumberSid  string                   `pulumi:"phoneNumberSid"`
	FriendlyName    string                   `pulumi:"friendlyName"`
	Description     string                   `pulumi:"description"`
	UniqueName      string                   `pulumi:"uniqueName"`
	Configuration   string                   `pulumi:"configuration"`
	DateCreated     string                   `pulumi:"dateCreated"`
	DateUpdated     string                   `pulumi:"dateUpdated"`
	Extensions      []AssignedAddOnExtension `pulumi:"extensions"`
}

func (a *GetIncomingPhoneNumberAssignedAddOnResult) Annotate(an infer.Annotator) {
	an.Describe(&a.Sid, "The assignment SID (XE...).")
	an.Describe(&a.OwnerAccountSid, "SID of the account that owns the phone number.")
	an.Describe(&a.PhoneNumberSid, "SID of the phone number the add-on is assigned to.")
	an.Describe(&a.FriendlyName, "The add-on's friendly name.")
	an.Describe(&a.Description, "A short description of what the add-on provides.")
	an.Describe(&a.UniqueName, "The add-on's application-defined unique name.")
	an.Describe(&a.Configuration, "The current configuration of the add-on installation, as a JSON string.")
	an.Describe(&a.DateCreated, "When the assignment was created (RFC 2822).")
	an.Describe(&a.DateUpdated, "When the assignment was last updated (RFC 2822).")
	an.Describe(&a.Extensions, "The extensions of the assigned add-on (read-only in the Twilio API).")
}

func (f *GetIncomingPhoneNumberAssignedAddOn) Annotate(a infer.Annotator) {
	a.Describe(&f, "Look up a Marketplace add-on assigned to a Twilio incoming phone number, with its extensions.")
}

func (f *GetIncomingPhoneNumberAssignedAddOn) Invoke(ctx context.Context, req infer.FunctionRequest[GetIncomingPhoneNumberAssignedAddOnArgs]) (infer.FunctionResponse[GetIncomingPhoneNumberAssignedAddOnResult], error) {
	const fn = "getIncomingPhoneNumberAssignedAddOn"
	type resp = infer.FunctionResponse[GetIncomingPhoneNumberAssignedAddOnResult]
	in := req.Input
	if err := requireArg(fn, "phoneNumberSid", in.PhoneNumberSid); err != nil {
		return resp{}, err
	}
	if err := requireArg(fn, "assignedAddOnSid", in.AssignedAddOnSid); err != nil {
		return resp{}, err
	}
	c, err := clientFor(ctx)
	if err != nil {
		return resp{}, err
	}
	acct := c.acct(in.AccountSid)
	o, err := c.do(ctx, "GET", "api", assignedAddOnPath(acct, in.PhoneNumberSid, in.AssignedAddOnSid), nil, nil)
	if err != nil {
		return resp{}, lookupFailure(fn, "assigned add-on "+in.AssignedAddOnSid, err)
	}
	exts := []AssignedAddOnExtension{}
	path := assignedAddOnPath(acct, in.PhoneNumberSid, in.AssignedAddOnSid)
	path = path[:len(path)-len(".json")] + "/Extensions.json"
	err = listPages(ctx, c, fn, "api", path, url.Values{"PageSize": {"1000"}}, "extensions", func(e obj) bool {
		enabled := e.optBool("enabled")
		exts = append(exts, AssignedAddOnExtension{Sid: e.str("sid"), FriendlyName: e.str("friendly_name"), ProductName: e.str("product_name"),
			UniqueName: e.str("unique_name"), Enabled: enabled != nil && *enabled})
		return true
	})
	if err != nil {
		return resp{}, lookupFailure(fn, "extensions of assigned add-on "+in.AssignedAddOnSid, err)
	}
	return resp{Output: GetIncomingPhoneNumberAssignedAddOnResult{
		Sid: o.str("sid"), OwnerAccountSid: acct, PhoneNumberSid: in.PhoneNumberSid, FriendlyName: o.str("friendly_name"),
		Description: o.str("description"), UniqueName: o.str("unique_name"), Configuration: assignedAddOnConfiguration(o),
		DateCreated: o.str("date_created"), DateUpdated: o.str("date_updated"), Extensions: exts,
	}}, nil
}
