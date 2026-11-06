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

// ---------------------------------------------------------------- getAddress

// GetAddress looks up an address by SID.
type GetAddress struct{}

// GetAddressArgs are the lookup arguments.
type GetAddressArgs struct {
	AddressSid string  `pulumi:"addressSid"`
	AccountSid *string `pulumi:"accountSid,optional"`
}

func (a *GetAddressArgs) Annotate(an infer.Annotator) {
	an.Describe(&a.AddressSid, "SID (AD...) of the address.")
	an.Describe(&a.AccountSid, "SID of the account that owns the address. Defaults to the provider account.")
}

// GetAddressResult is the looked-up address, as Twilio stores it (after any auto-correction).
type GetAddressResult struct {
	Sid              string `pulumi:"sid"`
	OwnerAccountSid  string `pulumi:"ownerAccountSid"`
	CustomerName     string `pulumi:"customerName"`
	Street           string `pulumi:"street"`
	StreetSecondary  string `pulumi:"streetSecondary"`
	City             string `pulumi:"city"`
	Region           string `pulumi:"region"`
	PostalCode       string `pulumi:"postalCode"`
	IsoCountry       string `pulumi:"isoCountry"`
	FriendlyName     string `pulumi:"friendlyName"`
	EmergencyEnabled bool   `pulumi:"emergencyEnabled"`
	Validated        bool   `pulumi:"validated"`
	Verified         bool   `pulumi:"verified"`
	DateCreated      string `pulumi:"dateCreated"`
	DateUpdated      string `pulumi:"dateUpdated"`
}

func (a *GetAddressResult) Annotate(an infer.Annotator) {
	an.Describe(&a.Sid, "The address SID (AD...).")
	an.Describe(&a.OwnerAccountSid, "SID of the account that owns the address.")
	an.Describe(&a.CustomerName, "Name of the customer or business.")
	an.Describe(&a.Street, "Street number and name as Twilio stores it.")
	an.Describe(&a.StreetSecondary, "Second address line (empty if none).")
	an.Describe(&a.City, "City.")
	an.Describe(&a.Region, "State or region.")
	an.Describe(&a.PostalCode, "Postal code.")
	an.Describe(&a.IsoCountry, "Two-letter ISO country code.")
	an.Describe(&a.FriendlyName, "Human-readable name (empty if none).")
	an.Describe(&a.EmergencyEnabled, "Whether the address is enabled for emergency calling.")
	an.Describe(&a.Validated, "Whether Twilio validated the address.")
	an.Describe(&a.Verified, "Whether the address was verified by Twilio support.")
	an.Describe(&a.DateCreated, "When the address was created (RFC 2822).")
	an.Describe(&a.DateUpdated, "When the address was last updated (RFC 2822).")
}

func (f *GetAddress) Annotate(a infer.Annotator) { a.Describe(&f, "Look up a Twilio address by SID.") }

func (f *GetAddress) Invoke(ctx context.Context, req infer.FunctionRequest[GetAddressArgs]) (infer.FunctionResponse[GetAddressResult], error) {
	const fn = "getAddress"
	if err := requireArg(fn, "addressSid", req.Input.AddressSid); err != nil {
		return infer.FunctionResponse[GetAddressResult]{}, err
	}
	c, err := clientFor(ctx)
	if err != nil {
		return infer.FunctionResponse[GetAddressResult]{}, err
	}
	acct := c.acct(req.Input.AccountSid)
	o, err := c.do(ctx, "GET", "api", acctPath(acct, "Addresses", req.Input.AddressSid), nil, nil)
	if err != nil {
		return infer.FunctionResponse[GetAddressResult]{}, lookupFailure(fn, "address "+req.Input.AddressSid, err)
	}
	return infer.FunctionResponse[GetAddressResult]{Output: GetAddressResult{
		Sid: o.str("sid"), OwnerAccountSid: acct, CustomerName: o.str("customer_name"), Street: o.str("street"),
		StreetSecondary: o.str("street_secondary"), City: o.str("city"), Region: o.str("region"),
		PostalCode: o.str("postal_code"), IsoCountry: o.str("iso_country"), FriendlyName: o.str("friendly_name"),
		EmergencyEnabled: isTrue(o.optBool("emergency_enabled")), Validated: isTrue(o.optBool("validated")),
		Verified: isTrue(o.optBool("verified")), DateCreated: o.str("date_created"), DateUpdated: o.str("date_updated"),
	}}, nil
}

// ------------------------------------------------------------- getSigningKey

// GetSigningKey looks up a signing key by SID. The secret is never returned.
type GetSigningKey struct{}

// GetSigningKeyArgs are the lookup arguments.
type GetSigningKeyArgs struct {
	KeySid     string  `pulumi:"keySid"`
	AccountSid *string `pulumi:"accountSid,optional"`
}

func (a *GetSigningKeyArgs) Annotate(an infer.Annotator) {
	an.Describe(&a.KeySid, "SID (SK...) of the signing key.")
	an.Describe(&a.AccountSid, "SID of the account that owns the key. Defaults to the provider account.")
}

// GetSigningKeyResult is the looked-up signing key.
type GetSigningKeyResult struct {
	Sid             string `pulumi:"sid"`
	OwnerAccountSid string `pulumi:"ownerAccountSid"`
	FriendlyName    string `pulumi:"friendlyName"`
	DateCreated     string `pulumi:"dateCreated"`
	DateUpdated     string `pulumi:"dateUpdated"`
}

func (a *GetSigningKeyResult) Annotate(an infer.Annotator) {
	an.Describe(&a.Sid, "The key SID (SK...).")
	an.Describe(&a.OwnerAccountSid, "SID of the account that owns the key.")
	an.Describe(&a.FriendlyName, "Name of the key.")
	an.Describe(&a.DateCreated, "When the key was created (RFC 2822).")
	an.Describe(&a.DateUpdated, "When the key was last updated (RFC 2822).")
}

func (f *GetSigningKey) Annotate(a infer.Annotator) {
	a.Describe(&f, "Look up a Twilio signing key by SID. The secret is never returned (Twilio shows it once, at creation).")
}

func (f *GetSigningKey) Invoke(ctx context.Context, req infer.FunctionRequest[GetSigningKeyArgs]) (infer.FunctionResponse[GetSigningKeyResult], error) {
	const fn = "getSigningKey"
	if err := requireArg(fn, "keySid", req.Input.KeySid); err != nil {
		return infer.FunctionResponse[GetSigningKeyResult]{}, err
	}
	c, err := clientFor(ctx)
	if err != nil {
		return infer.FunctionResponse[GetSigningKeyResult]{}, err
	}
	acct := c.acct(req.Input.AccountSid)
	o, err := c.do(ctx, "GET", "api", acctPath(acct, "SigningKeys", req.Input.KeySid), nil, nil)
	if err != nil {
		return infer.FunctionResponse[GetSigningKeyResult]{}, lookupFailure(fn, "signing key "+req.Input.KeySid, err)
	}
	return infer.FunctionResponse[GetSigningKeyResult]{Output: GetSigningKeyResult{
		Sid: o.str("sid"), OwnerAccountSid: acct, FriendlyName: o.str("friendly_name"),
		DateCreated: o.str("date_created"), DateUpdated: o.str("date_updated"),
	}}, nil
}

// ----------------------------------------------------------------- getQueue

// GetQueue looks up a call queue by SID.
type GetQueue struct{}

// GetQueueArgs are the lookup arguments.
type GetQueueArgs struct {
	QueueSid   string  `pulumi:"queueSid"`
	AccountSid *string `pulumi:"accountSid,optional"`
}

func (a *GetQueueArgs) Annotate(an infer.Annotator) {
	an.Describe(&a.QueueSid, "SID (QU...) of the queue.")
	an.Describe(&a.AccountSid, "SID of the account that owns the queue. Defaults to the provider account.")
}

// GetQueueResult is the looked-up queue.
type GetQueueResult struct {
	Sid             string `pulumi:"sid"`
	OwnerAccountSid string `pulumi:"ownerAccountSid"`
	FriendlyName    string `pulumi:"friendlyName"`
	MaxSize         int    `pulumi:"maxSize"`
	CurrentSize     int    `pulumi:"currentSize"`
	AverageWaitTime int    `pulumi:"averageWaitTime"`
	DateCreated     string `pulumi:"dateCreated"`
	DateUpdated     string `pulumi:"dateUpdated"`
}

func (a *GetQueueResult) Annotate(an infer.Annotator) {
	an.Describe(&a.Sid, "The queue SID (QU...).")
	an.Describe(&a.OwnerAccountSid, "SID of the account that owns the queue.")
	an.Describe(&a.FriendlyName, "Name of the queue.")
	an.Describe(&a.MaxSize, "Maximum number of calls the queue holds.")
	an.Describe(&a.CurrentSize, "Number of calls in the queue now.")
	an.Describe(&a.AverageWaitTime, "Average wait time of the queued calls, in seconds.")
	an.Describe(&a.DateCreated, "When the queue was created (RFC 2822).")
	an.Describe(&a.DateUpdated, "When the queue was last updated (RFC 2822).")
}

func (f *GetQueue) Annotate(a infer.Annotator) { a.Describe(&f, "Look up a Twilio call queue by SID.") }

func (f *GetQueue) Invoke(ctx context.Context, req infer.FunctionRequest[GetQueueArgs]) (infer.FunctionResponse[GetQueueResult], error) {
	const fn = "getQueue"
	if err := requireArg(fn, "queueSid", req.Input.QueueSid); err != nil {
		return infer.FunctionResponse[GetQueueResult]{}, err
	}
	c, err := clientFor(ctx)
	if err != nil {
		return infer.FunctionResponse[GetQueueResult]{}, err
	}
	acct := c.acct(req.Input.AccountSid)
	o, err := c.do(ctx, "GET", "api", acctPath(acct, "Queues", req.Input.QueueSid), nil, nil)
	if err != nil {
		return infer.FunctionResponse[GetQueueResult]{}, lookupFailure(fn, "queue "+req.Input.QueueSid, err)
	}
	return infer.FunctionResponse[GetQueueResult]{Output: GetQueueResult{
		Sid: o.str("sid"), OwnerAccountSid: acct, FriendlyName: o.str("friendly_name"), MaxSize: intOf(o.optInt("max_size")),
		CurrentSize: intOf(o.optInt("current_size")), AverageWaitTime: intOf(o.optInt("average_wait_time")),
		DateCreated: o.str("date_created"), DateUpdated: o.str("date_updated"),
	}}, nil
}

// ------------------------------------------------------------ getUsageTrigger

// GetUsageTrigger looks up a usage trigger by SID.
type GetUsageTrigger struct{}

// GetUsageTriggerArgs are the lookup arguments.
type GetUsageTriggerArgs struct {
	TriggerSid string  `pulumi:"triggerSid"`
	AccountSid *string `pulumi:"accountSid,optional"`
}

func (a *GetUsageTriggerArgs) Annotate(an infer.Annotator) {
	an.Describe(&a.TriggerSid, "SID (UT...) of the usage trigger.")
	an.Describe(&a.AccountSid, "SID of the account that owns the trigger. Defaults to the provider account.")
}

// GetUsageTriggerResult is the looked-up usage trigger.
type GetUsageTriggerResult struct {
	Sid             string `pulumi:"sid"`
	OwnerAccountSid string `pulumi:"ownerAccountSid"`
	FriendlyName    string `pulumi:"friendlyName"`
	UsageCategory   string `pulumi:"usageCategory"`
	TriggerValue    string `pulumi:"triggerValue"`
	TriggerBy       string `pulumi:"triggerBy"`
	Recurring       string `pulumi:"recurring"`
	CallbackUrl     string `pulumi:"callbackUrl"`
	CallbackMethod  string `pulumi:"callbackMethod"`
	CurrentValue    string `pulumi:"currentValue"`
	DateFired       string `pulumi:"dateFired"`
	UsageRecordUri  string `pulumi:"usageRecordUri"`
}

func (a *GetUsageTriggerResult) Annotate(an infer.Annotator) {
	an.Describe(&a.Sid, "The trigger SID (UT...).")
	an.Describe(&a.OwnerAccountSid, "SID of the account that owns the trigger.")
	an.Describe(&a.FriendlyName, "Name of the trigger (empty if none).")
	an.Describe(&a.UsageCategory, "Usage category the trigger watches.")
	an.Describe(&a.TriggerValue, "Value at which the trigger fires.")
	an.Describe(&a.TriggerBy, "What the value is compared with: count, usage or price.")
	an.Describe(&a.Recurring, "How often the trigger resets (empty for a one-shot trigger).")
	an.Describe(&a.CallbackUrl, "URL requested when the trigger fires.")
	an.Describe(&a.CallbackMethod, "HTTP method used for the callback.")
	an.Describe(&a.CurrentValue, "Current usage value.")
	an.Describe(&a.DateFired, "When the trigger last fired (RFC 2822), empty if never.")
	an.Describe(&a.UsageRecordUri, "Relative URI of the usage records the trigger watches.")
}

func (f *GetUsageTrigger) Annotate(a infer.Annotator) {
	a.Describe(&f, "Look up a Twilio usage trigger by SID.")
}

func (f *GetUsageTrigger) Invoke(ctx context.Context, req infer.FunctionRequest[GetUsageTriggerArgs]) (infer.FunctionResponse[GetUsageTriggerResult], error) {
	const fn = "getUsageTrigger"
	if err := requireArg(fn, "triggerSid", req.Input.TriggerSid); err != nil {
		return infer.FunctionResponse[GetUsageTriggerResult]{}, err
	}
	c, err := clientFor(ctx)
	if err != nil {
		return infer.FunctionResponse[GetUsageTriggerResult]{}, err
	}
	acct := c.acct(req.Input.AccountSid)
	o, err := c.do(ctx, "GET", "api", acctPath(acct, "Usage/Triggers", req.Input.TriggerSid), nil, nil)
	if err != nil {
		return infer.FunctionResponse[GetUsageTriggerResult]{}, lookupFailure(fn, "usage trigger "+req.Input.TriggerSid, err)
	}
	return infer.FunctionResponse[GetUsageTriggerResult]{Output: GetUsageTriggerResult{
		Sid: o.str("sid"), OwnerAccountSid: acct, FriendlyName: o.str("friendly_name"), UsageCategory: o.str("usage_category"),
		TriggerValue: o.str("trigger_value"), TriggerBy: o.str("trigger_by"), Recurring: o.str("recurring"),
		CallbackUrl: o.str("callback_url"), CallbackMethod: o.str("callback_method"), CurrentValue: o.str("current_value"),
		DateFired: o.str("date_fired"), UsageRecordUri: o.str("usage_record_uri"),
	}}, nil
}
