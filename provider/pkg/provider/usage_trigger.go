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
	"strings"

	p "github.com/pulumi/pulumi-go-provider"
	"github.com/pulumi/pulumi-go-provider/infer"
)

// UsageTrigger manages a Twilio usage trigger: a webhook that fires when a usage category
// crosses a threshold.
type UsageTrigger struct{}

// UsageTriggerArgs are the inputs of a UsageTrigger.
type UsageTriggerArgs struct {
	CallbackUrl    string  `pulumi:"callbackUrl"`
	TriggerValue   string  `pulumi:"triggerValue" provider:"replaceOnChanges"`
	UsageCategory  string  `pulumi:"usageCategory" provider:"replaceOnChanges"`
	AccountSid     *string `pulumi:"accountSid,optional" provider:"replaceOnChanges"`
	CallbackMethod *string `pulumi:"callbackMethod,optional"`
	FriendlyName   *string `pulumi:"friendlyName,optional"`
	Recurring      *string `pulumi:"recurring,optional" provider:"replaceOnChanges"`
	TriggerBy      *string `pulumi:"triggerBy,optional" provider:"replaceOnChanges"`
}

func (a *UsageTriggerArgs) props() props {
	return props{strs: []strProp{
		{"CallbackMethod", "callback_method", &a.CallbackMethod}, {"FriendlyName", "friendly_name", &a.FriendlyName},
		{"Recurring", "recurring", &a.Recurring}, {"TriggerBy", "trigger_by", &a.TriggerBy},
	}}
}

func (a *UsageTriggerArgs) Annotate(an infer.Annotator) {
	an.Describe(&a.CallbackUrl, "URL Twilio requests when the trigger fires.")
	an.Describe(&a.TriggerValue, "Usage value at which the trigger fires, for example 100 (a count or amount in the category's unit), "+
		"or an offset such as +30 from the current value. Twilio cannot change it, so changing it replaces the trigger.")
	an.Describe(&a.UsageCategory, "Usage category to watch, for example sms, calls or totalprice (see the Twilio usage categories). "+
		"Changing it replaces the trigger.")
	an.Describe(&a.AccountSid, "SID of the account or subaccount that owns the trigger. Defaults to the provider account. Changing it replaces the resource.")
	an.Describe(&a.CallbackMethod, "HTTP method for callbackUrl: GET or POST (Twilio default POST).")
	an.Describe(&a.FriendlyName, "Human-readable name of the trigger (up to 64 characters).")
	an.Describe(&a.Recurring, "How often the trigger resets: daily, monthly, yearly or alltime. Omit for a trigger that fires once. Changing it replaces the trigger.")
	an.Describe(&a.TriggerBy, "What triggerValue is compared with: count, usage or price. Twilio picks one from the category when it is omitted and always reports one, so after importing a trigger set this input to the imported value to avoid a replacement. Changing it replaces the trigger.")
}

// UsageTriggerState is the persisted state of a UsageTrigger.
type UsageTriggerState struct {
	UsageTriggerArgs

	// Sid is the trigger SID (UT...) and the resource ID.
	Sid string `pulumi:"sid"`
	// OwnerAccountSid is the account that owns the trigger.
	OwnerAccountSid string `pulumi:"ownerAccountSid"`
	// CurrentValue is the current usage value Twilio compares against the trigger.
	CurrentValue string `pulumi:"currentValue"`
	// DateFired is when the trigger last fired (empty if it never has).
	DateFired string `pulumi:"dateFired"`
	// UsageRecordUri is the URI of the usage records the trigger watches.
	UsageRecordUri string `pulumi:"usageRecordUri"`
	// ApiVersion is the API version used for the callback.
	ApiVersion string `pulumi:"apiVersion"`
}

func (s *UsageTriggerState) Annotate(a infer.Annotator) {
	a.Describe(&s.Sid, "The usage trigger SID (UT...).")
	a.Describe(&s.OwnerAccountSid, "SID of the account that owns the trigger.")
	a.Describe(&s.CurrentValue, "Current usage value at the last refresh.")
	a.Describe(&s.DateFired, "When the trigger last fired (RFC 2822), empty if it never has.")
	a.Describe(&s.UsageRecordUri, "Relative URI of the usage records the trigger watches.")
	a.Describe(&s.ApiVersion, "API version Twilio uses for the callback request.")
}

func (r *UsageTrigger) Annotate(a infer.Annotator) {
	a.Describe(&r, "A Twilio usage trigger: calls a webhook when usage in a category reaches a value. Only the callback "+
		"and name can be changed in place.")
}

var _ infer.CustomResource[UsageTriggerArgs, UsageTriggerState] = (*UsageTrigger)(nil)
var _ infer.CustomCheck[UsageTriggerArgs] = (*UsageTrigger)(nil)
var _ infer.CustomRead[UsageTriggerArgs, UsageTriggerState] = (*UsageTrigger)(nil)
var _ infer.CustomUpdate[UsageTriggerArgs, UsageTriggerState] = (*UsageTrigger)(nil)
var _ infer.CustomDelete[UsageTriggerState] = (*UsageTrigger)(nil)

var triggerRecurring = map[string]bool{"daily": true, "monthly": true, "yearly": true, "alltime": true}
var triggerBy = map[string]bool{"count": true, "usage": true, "price": true}

func (r *UsageTrigger) Check(ctx context.Context, req infer.CheckRequest) (infer.CheckResponse[UsageTriggerArgs], error) {
	args, failures, err := infer.DefaultCheck[UsageTriggerArgs](ctx, req.NewInputs)
	if err != nil || len(failures) > 0 {
		return infer.CheckResponse[UsageTriggerArgs]{Inputs: args, Failures: failures}, err
	}
	unk := unknowns(req.NewInputs)
	for prop, v := range map[string]string{"callbackUrl": args.CallbackUrl, "triggerValue": args.TriggerValue, "usageCategory": args.UsageCategory} {
		if !unk(prop) && v == "" {
			failures = append(failures, p.CheckFailure{Property: prop, Reason: prop + " must not be empty"})
		}
	}
	failures = checkMethods(failures, unk, map[string]*string{"callbackMethod": args.CallbackMethod})
	if args.Recurring != nil && !unk("recurring") && !triggerRecurring[*args.Recurring] {
		failures = append(failures, p.CheckFailure{Property: "recurring", Reason: "recurring must be daily, monthly, yearly or alltime"})
	}
	if args.TriggerBy != nil && !unk("triggerBy") && !triggerBy[*args.TriggerBy] {
		failures = append(failures, p.CheckFailure{Property: "triggerBy", Reason: "triggerBy must be count, usage or price"})
	}
	return infer.CheckResponse[UsageTriggerArgs]{Inputs: args, Failures: failures}, nil
}

func triggerOutputs(st *UsageTriggerState, o obj) {
	st.CurrentValue, st.DateFired = o.str("current_value"), o.str("date_fired")
	st.UsageRecordUri, st.ApiVersion = o.str("usage_record_uri"), o.str("api_version")
}

func (r *UsageTrigger) Create(ctx context.Context, req infer.CreateRequest[UsageTriggerArgs]) (infer.CreateResponse[UsageTriggerState], error) {
	a := req.Inputs
	if req.DryRun {
		return infer.CreateResponse[UsageTriggerState]{ID: "", Output: UsageTriggerState{UsageTriggerArgs: a}}, nil
	}
	c, err := clientFor(ctx)
	if err != nil {
		return infer.CreateResponse[UsageTriggerState]{}, err
	}
	acct := c.acct(a.AccountSid)
	form := url.Values{"CallbackUrl": {a.CallbackUrl}, "TriggerValue": {a.TriggerValue}, "UsageCategory": {a.UsageCategory}}
	a.props().encode(form, nil)
	o, err := c.do(ctx, "POST", "api", acctPath(acct, "Usage/Triggers", ""), nil, form)
	if err != nil {
		return infer.CreateResponse[UsageTriggerState]{}, fmt.Errorf("create usage trigger: %w", err)
	}
	sid := o.str("sid")
	st := UsageTriggerState{UsageTriggerArgs: a, Sid: sid, OwnerAccountSid: acct}
	triggerOutputs(&st, o)
	return infer.CreateResponse[UsageTriggerState]{ID: scopedID(c, acct, sid), Output: st}, nil
}

func (r *UsageTrigger) Read(ctx context.Context, req infer.ReadRequest[UsageTriggerArgs, UsageTriggerState]) (infer.ReadResponse[UsageTriggerArgs, UsageTriggerState], error) {
	c, err := clientFor(ctx)
	if err != nil {
		return infer.ReadResponse[UsageTriggerArgs, UsageTriggerState]{}, err
	}
	acct, sid := resolveScope(c, req.ID, req.State.OwnerAccountSid)
	o, err := c.do(ctx, "GET", "api", acctPath(acct, "Usage/Triggers", sid), nil, nil)
	if err != nil {
		if isNotFound(err) {
			return infer.ReadResponse[UsageTriggerArgs, UsageTriggerState]{}, nil
		}
		return infer.ReadResponse[UsageTriggerArgs, UsageTriggerState]{}, fmt.Errorf("read usage trigger %s: %w", req.ID, err)
	}
	in := req.Inputs
	imported := in.CallbackUrl == ""
	in.CallbackUrl, in.UsageCategory = o.str("callback_url"), o.str("usage_category")
	// Twilio turns an offset (+30) into the absolute value and may reformat a number, so keep
	// the program's value while it still means the same thing.
	if imported || (!strings.HasPrefix(in.TriggerValue, "+") && !sameNumber(in.TriggerValue, o.str("trigger_value"))) {
		in.TriggerValue = o.str("trigger_value")
	}
	if imported {
		if acct != c.accountSid {
			in.AccountSid = &acct
		}
		in.props().adopt(o)
	} else {
		in.props().refreshSet(o)
	}
	st := UsageTriggerState{UsageTriggerArgs: in, Sid: sid, OwnerAccountSid: acct}
	triggerOutputs(&st, o)
	return infer.ReadResponse[UsageTriggerArgs, UsageTriggerState]{ID: req.ID, Inputs: in, State: st}, nil
}

func (r *UsageTrigger) Update(ctx context.Context, req infer.UpdateRequest[UsageTriggerArgs, UsageTriggerState]) (infer.UpdateResponse[UsageTriggerState], error) {
	a := req.Inputs
	st := req.State
	st.UsageTriggerArgs = a
	if req.DryRun {
		return infer.UpdateResponse[UsageTriggerState]{Output: st}, nil
	}
	c, err := clientFor(ctx)
	if err != nil {
		return infer.UpdateResponse[UsageTriggerState]{}, err
	}
	acct, sid := resolveScope(c, req.ID, req.State.OwnerAccountSid)
	// Only the callback and the name are updatable; the rest replaces the trigger.
	form := url.Values{"CallbackUrl": {a.CallbackUrl}}
	if a.CallbackMethod != nil {
		form.Set("CallbackMethod", *a.CallbackMethod)
	} else if req.State.CallbackMethod != nil && *req.State.CallbackMethod != "POST" {
		form.Set("CallbackMethod", "POST")
	}
	if a.FriendlyName != nil {
		form.Set("FriendlyName", *a.FriendlyName)
	} else if req.State.FriendlyName != nil {
		form.Set("FriendlyName", "")
	}
	o, err := c.do(ctx, "POST", "api", acctPath(acct, "Usage/Triggers", sid), nil, form)
	if err != nil {
		return infer.UpdateResponse[UsageTriggerState]{}, fmt.Errorf("update usage trigger %s: %w", req.ID, err)
	}
	triggerOutputs(&st, o)
	return infer.UpdateResponse[UsageTriggerState]{Output: st}, nil
}

func (r *UsageTrigger) Delete(ctx context.Context, req infer.DeleteRequest[UsageTriggerState]) (infer.DeleteResponse, error) {
	c, err := clientFor(ctx)
	if err != nil {
		return infer.DeleteResponse{}, err
	}
	acct, sid := resolveScope(c, req.ID, req.State.OwnerAccountSid)
	if _, err := c.do(ctx, "DELETE", "api", acctPath(acct, "Usage/Triggers", sid), nil, nil); err != nil && !isNotFound(err) {
		return infer.DeleteResponse{}, fmt.Errorf("delete usage trigger %s: %w", req.ID, err)
	}
	return infer.DeleteResponse{}, nil
}
