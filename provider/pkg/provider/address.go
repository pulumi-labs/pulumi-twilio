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

// Address manages a Twilio address: the postal address of a customer, used for regulatory
// compliance and as an emergency-calling address for phone numbers.
type Address struct{}

// AddressArgs are the inputs of an Address.
type AddressArgs struct {
	CustomerName       string  `pulumi:"customerName"`
	Street             string  `pulumi:"street"`
	City               string  `pulumi:"city"`
	Region             string  `pulumi:"region"`
	PostalCode         string  `pulumi:"postalCode"`
	IsoCountry         string  `pulumi:"isoCountry" provider:"replaceOnChanges"`
	AccountSid         *string `pulumi:"accountSid,optional" provider:"replaceOnChanges"`
	FriendlyName       *string `pulumi:"friendlyName,optional"`
	StreetSecondary    *string `pulumi:"streetSecondary,optional"`
	EmergencyEnabled   *bool   `pulumi:"emergencyEnabled,optional"`
	AutoCorrectAddress *bool   `pulumi:"autoCorrectAddress,optional"`
}

func (a *AddressArgs) props() props {
	return props{
		strs:  []strProp{{"FriendlyName", "friendly_name", &a.FriendlyName}, {"StreetSecondary", "street_secondary", &a.StreetSecondary}},
		bools: []boolProp{{"EmergencyEnabled", "emergency_enabled", &a.EmergencyEnabled}},
	}
}

func (a *AddressArgs) Annotate(an infer.Annotator) {
	an.Describe(&a.CustomerName, "Name of the customer or business the address belongs to.")
	an.Describe(&a.Street, "Street number and name.")
	an.Describe(&a.City, "City.")
	an.Describe(&a.Region, "State or region.")
	an.Describe(&a.PostalCode, "Postal code.")
	an.Describe(&a.IsoCountry, "Two-letter ISO 3166-1 country code, for example US. Twilio cannot change the country of an address, so changing it replaces the resource.")
	an.Describe(&a.AccountSid, "SID of the account or subaccount that owns the address. Defaults to the provider account. Changing it replaces the resource.")
	an.Describe(&a.FriendlyName, "Human-readable name of the address (up to 64 characters for regulatory addresses, 32 otherwise). Removing it leaves the name unchanged on Twilio.")
	an.Describe(&a.StreetSecondary, "Second address line, for example a suite or floor.")
	an.Describe(&a.EmergencyEnabled, "Whether the address is enabled for emergency calling. Twilio only accepts true for a validated address.")
	an.Describe(&a.AutoCorrectAddress, "Whether Twilio corrects the address on write (Twilio default true). Twilio does not return this setting. "+
		"While it is not false, the address lines Twilio stores may differ from the inputs (normalized spelling and case), so they are not refreshed "+
		"from Twilio; set it to false to have drift in the address lines detected. Because Twilio never returns it, an imported address has no value for it.")
}

// AddressState is the persisted state of an Address.
type AddressState struct {
	AddressArgs

	// Sid is the address SID (AD...) and the resource ID.
	Sid string `pulumi:"sid"`
	// OwnerAccountSid is the account that owns the address.
	OwnerAccountSid string `pulumi:"ownerAccountSid"`
	// Validated reports whether Twilio validated the address.
	Validated bool `pulumi:"validated"`
	// Verified reports whether the address was verified by Twilio support.
	Verified bool `pulumi:"verified"`
	// DateCreated is the RFC 2822 creation time.
	DateCreated string `pulumi:"dateCreated"`
	// DateUpdated is the RFC 2822 time of the last update.
	DateUpdated string `pulumi:"dateUpdated"`
}

func (s *AddressState) Annotate(a infer.Annotator) {
	a.Describe(&s.Sid, "The address SID (AD...), usable as addressSid or emergencyAddressSid on a phone number.")
	a.Describe(&s.OwnerAccountSid, "SID of the account that owns the address.")
	a.Describe(&s.Validated, "Whether Twilio validated the address.")
	a.Describe(&s.Verified, "Whether the address was verified by Twilio support (required for some countries' numbers).")
	a.Describe(&s.DateCreated, "When the address was created (RFC 2822).")
	a.Describe(&s.DateUpdated, "When the address was last updated (RFC 2822).")
}

func (r *Address) Annotate(a infer.Annotator) {
	a.Describe(&r, "A Twilio address. Deleting an address that is still used by a phone number, bundle or regulatory "+
		"registration fails until it is released from them.")
}

var _ infer.CustomResource[AddressArgs, AddressState] = (*Address)(nil)
var _ infer.CustomCheck[AddressArgs] = (*Address)(nil)
var _ infer.CustomRead[AddressArgs, AddressState] = (*Address)(nil)
var _ infer.CustomUpdate[AddressArgs, AddressState] = (*Address)(nil)
var _ infer.CustomDelete[AddressState] = (*Address)(nil)

func (r *Address) Check(ctx context.Context, req infer.CheckRequest) (infer.CheckResponse[AddressArgs], error) {
	args, failures, err := infer.DefaultCheck[AddressArgs](ctx, req.NewInputs)
	if err != nil || len(failures) > 0 {
		return infer.CheckResponse[AddressArgs]{Inputs: args, Failures: failures}, err
	}
	unk := unknowns(req.NewInputs)
	for prop, v := range map[string]string{"customerName": args.CustomerName, "street": args.Street, "city": args.City,
		"region": args.Region, "postalCode": args.PostalCode} {
		if !unk(prop) && v == "" {
			failures = append(failures, p.CheckFailure{Property: prop, Reason: prop + " must not be empty"})
		}
	}
	if !unk("isoCountry") && len(args.IsoCountry) != 2 {
		failures = append(failures, p.CheckFailure{Property: "isoCountry", Reason: "isoCountry must be a two-letter ISO 3166-1 country code"})
	}
	return infer.CheckResponse[AddressArgs]{Inputs: args, Failures: failures}, nil
}

func (a AddressArgs) form() url.Values {
	return url.Values{"CustomerName": {a.CustomerName}, "Street": {a.Street}, "City": {a.City},
		"Region": {a.Region}, "PostalCode": {a.PostalCode}}
}

// fillLines copies the address lines Twilio stores into the inputs.
func (a *AddressArgs) fillLines(o obj) {
	a.CustomerName, a.Street, a.City = o.str("customer_name"), o.str("street"), o.str("city")
	a.Region, a.PostalCode, a.IsoCountry = o.str("region"), o.str("postal_code"), o.str("iso_country")
}

func addressOutputs(st *AddressState, o obj) {
	st.Validated, st.Verified = isTrue(o.optBool("validated")), isTrue(o.optBool("verified"))
	st.DateCreated, st.DateUpdated = o.str("date_created"), o.str("date_updated")
}

func (r *Address) Create(ctx context.Context, req infer.CreateRequest[AddressArgs]) (infer.CreateResponse[AddressState], error) {
	a := req.Inputs
	if req.DryRun {
		return infer.CreateResponse[AddressState]{ID: "", Output: AddressState{AddressArgs: a}}, nil
	}
	c, err := clientFor(ctx)
	if err != nil {
		return infer.CreateResponse[AddressState]{}, err
	}
	acct := c.acct(a.AccountSid)
	form := a.form()
	form.Set("IsoCountry", a.IsoCountry)
	a.props().encode(form, nil)
	setB(form, "AutoCorrectAddress", a.AutoCorrectAddress)
	o, err := c.do(ctx, "POST", "api", acctPath(acct, "Addresses", ""), nil, form)
	if err != nil {
		return infer.CreateResponse[AddressState]{}, fmt.Errorf("create address: %w", err)
	}
	sid := o.str("sid")
	st := AddressState{AddressArgs: a, Sid: sid, OwnerAccountSid: acct}
	addressOutputs(&st, o)
	return infer.CreateResponse[AddressState]{ID: scopedID(c, acct, sid), Output: st}, nil
}

func (r *Address) Read(ctx context.Context, req infer.ReadRequest[AddressArgs, AddressState]) (infer.ReadResponse[AddressArgs, AddressState], error) {
	c, err := clientFor(ctx)
	if err != nil {
		return infer.ReadResponse[AddressArgs, AddressState]{}, err
	}
	acct, sid := resolveScope(c, req.ID, req.State.OwnerAccountSid)
	o, err := c.do(ctx, "GET", "api", acctPath(acct, "Addresses", sid), nil, nil)
	if err != nil {
		if isNotFound(err) {
			return infer.ReadResponse[AddressArgs, AddressState]{}, nil
		}
		return infer.ReadResponse[AddressArgs, AddressState]{}, fmt.Errorf("read address %s: %w", req.ID, err)
	}
	in := req.Inputs
	imported := req.State.Sid == "" && in.CustomerName == ""
	switch {
	case imported:
		in.fillLines(o)
		if acct != c.accountSid {
			in.AccountSid = &acct
		}
		in.props().adopt(o)
	default:
		if in.AutoCorrectAddress != nil && !*in.AutoCorrectAddress {
			in.fillLines(o)
		}
		in.props().refreshSet(o)
	}
	st := AddressState{AddressArgs: in, Sid: sid, OwnerAccountSid: acct}
	addressOutputs(&st, o)
	return infer.ReadResponse[AddressArgs, AddressState]{ID: req.ID, Inputs: in, State: st}, nil
}

func (r *Address) Update(ctx context.Context, req infer.UpdateRequest[AddressArgs, AddressState]) (infer.UpdateResponse[AddressState], error) {
	a := req.Inputs
	st := req.State
	st.AddressArgs = a
	if req.DryRun {
		return infer.UpdateResponse[AddressState]{Output: st}, nil
	}
	c, err := clientFor(ctx)
	if err != nil {
		return infer.UpdateResponse[AddressState]{}, err
	}
	acct, sid := resolveScope(c, req.ID, req.State.OwnerAccountSid)
	form := a.form()
	prev := req.State.AddressArgs.props()
	a.props().encode(form, &prev)
	setB(form, "AutoCorrectAddress", a.AutoCorrectAddress)
	o, err := c.do(ctx, "POST", "api", acctPath(acct, "Addresses", sid), nil, form)
	if err != nil {
		return infer.UpdateResponse[AddressState]{}, fmt.Errorf("update address %s: %w", req.ID, err)
	}
	addressOutputs(&st, o)
	return infer.UpdateResponse[AddressState]{Output: st}, nil
}

func (r *Address) Delete(ctx context.Context, req infer.DeleteRequest[AddressState]) (infer.DeleteResponse, error) {
	c, err := clientFor(ctx)
	if err != nil {
		return infer.DeleteResponse{}, err
	}
	acct, sid := resolveScope(c, req.ID, req.State.OwnerAccountSid)
	if _, err := c.do(ctx, "DELETE", "api", acctPath(acct, "Addresses", sid), nil, nil); err != nil && !isNotFound(err) {
		return infer.DeleteResponse{}, fmt.Errorf("delete address %s: %w", req.ID, err)
	}
	return infer.DeleteResponse{}, nil
}
