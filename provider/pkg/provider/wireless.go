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

// The resources in this file configure Programmable Wireless (wireless.twilio.com v1): the Rate
// Plan that sets the data and messaging allowances of a SIM. SIMs, Commands, data sessions and usage
// are runtime or physical data and are not managed. The shapes follow twilio-oai
// spec/json/twilio_wireless_v1.json.

const wirelessProduct = "wireless"

func wirelessPath(coll, sid string) string {
	path := "/v1/" + coll
	if sid != "" {
		path += "/" + url.PathEscape(sid)
	}
	return path
}

// WirelessRatePlan manages a Programmable Wireless Rate Plan. Twilio changes only the unique name
// and the friendly name in place; every other input replaces the plan.
type WirelessRatePlan struct{}

// WirelessRatePlanArgs are the inputs of a WirelessRatePlan.
type WirelessRatePlanArgs struct {
	UniqueName                    *string   `pulumi:"uniqueName,optional"`
	FriendlyName                  *string   `pulumi:"friendlyName,optional"`
	DataEnabled                   *bool     `pulumi:"dataEnabled,optional"`
	DataLimit                     *int      `pulumi:"dataLimit,optional"`
	DataMetering                  *string   `pulumi:"dataMetering,optional"`
	MessagingEnabled              *bool     `pulumi:"messagingEnabled,optional"`
	VoiceEnabled                  *bool     `pulumi:"voiceEnabled,optional"`
	NationalRoamingEnabled        *bool     `pulumi:"nationalRoamingEnabled,optional"`
	NationalRoamingDataLimit      *int      `pulumi:"nationalRoamingDataLimit,optional"`
	InternationalRoaming          *[]string `pulumi:"internationalRoaming,optional"`
	InternationalRoamingDataLimit *int      `pulumi:"internationalRoamingDataLimit,optional"`
	DataLimitStrategy             *string   `pulumi:"dataLimitStrategy,optional"`
}

func (a *WirelessRatePlanArgs) Annotate(an infer.Annotator) {
	an.Describe(&a.UniqueName, "An application-defined name that uniquely identifies the plan and can be used in place of its SID. Updates in place; removing it leaves the name unchanged.")
	an.Describe(&a.FriendlyName, "A descriptive name that does not have to be unique. Updates in place; removing it leaves the name unchanged.")
	an.Describe(&a.DataEnabled, "Whether SIMs can use GPRS/3G/4G/LTE data connectivity. Changing it replaces the plan.")
	an.Describe(&a.DataLimit, "The data (download and upload combined) in megabytes the home network (T-Mobile USA) allows per month, up to 2,000,000 (Twilio default 1,000). Changing it replaces the plan.")
	an.Describe(&a.DataMetering, "The model used to meter data usage: payg, quota-1, quota-10 or quota-50. Changing it replaces the plan.")
	an.Describe(&a.MessagingEnabled, "Whether SIMs can make, send and receive SMS using Commands. Changing it replaces the plan.")
	an.Describe(&a.VoiceEnabled, "Deprecated by Twilio: whether SIMs can make and receive voice calls. Changing it replaces the plan.")
	an.Describe(&a.NationalRoamingEnabled, "Whether SIMs can roam on networks other than the home network in the United States. Changing it replaces the plan.")
	an.Describe(&a.NationalRoamingDataLimit, "The data in megabytes allowed per month on non-home networks in the United States, up to 2,000,000. Changing it replaces the plan.")
	an.Describe(&a.InternationalRoaming, "The services SIMs can use outside the United States: data and messaging. Order is not significant. Changing it replaces the plan.")
	an.Describe(&a.InternationalRoamingDataLimit, "The data in megabytes allowed per month while roaming outside the United States, up to 2,000,000. Changing it replaces the plan.")
	an.Describe(&a.DataLimitStrategy, "What the home network does after a SIM exceeds dataLimit: block or throttle (Twilio default block). Twilio does not report it, so a change made outside Pulumi is not detected. Changing it replaces the plan.")
}

// WirelessRatePlanState is the persisted state of a WirelessRatePlan.
type WirelessRatePlanState struct {
	WirelessRatePlanArgs

	Sid         string `pulumi:"sid"`
	DateCreated string `pulumi:"dateCreated"`
	DateUpdated string `pulumi:"dateUpdated"`
}

func (s *WirelessRatePlanState) Annotate(a infer.Annotator) {
	a.Describe(&s.Sid, "The Rate Plan SID (WP...).")
	a.Describe(&s.DateCreated, "When the plan was created (ISO 8601).")
	a.Describe(&s.DateUpdated, "When the plan was last updated (ISO 8601).")
}

func (r *WirelessRatePlan) Annotate(a infer.Annotator) {
	a.Describe(&r, "A Programmable Wireless Rate Plan. The resource ID is the plan SID. Only uniqueName and friendlyName update in place; every other input replaces the plan, and Twilio refuses to delete a plan that SIMs still use. Inputs a program leaves out take Twilio's defaults and keep the value Twilio reports.")
}

var _ infer.CustomResource[WirelessRatePlanArgs, WirelessRatePlanState] = (*WirelessRatePlan)(nil)
var _ infer.CustomCheck[WirelessRatePlanArgs] = (*WirelessRatePlan)(nil)
var _ infer.CustomDiff[WirelessRatePlanArgs, WirelessRatePlanState] = (*WirelessRatePlan)(nil)
var _ infer.CustomRead[WirelessRatePlanArgs, WirelessRatePlanState] = (*WirelessRatePlan)(nil)
var _ infer.CustomUpdate[WirelessRatePlanArgs, WirelessRatePlanState] = (*WirelessRatePlan)(nil)
var _ infer.CustomDelete[WirelessRatePlanState] = (*WirelessRatePlan)(nil)

func (r *WirelessRatePlan) Check(ctx context.Context, req infer.CheckRequest) (infer.CheckResponse[WirelessRatePlanArgs], error) {
	args, failures, err := infer.DefaultCheck[WirelessRatePlanArgs](ctx, req.NewInputs)
	if err != nil || len(failures) > 0 {
		return infer.CheckResponse[WirelessRatePlanArgs]{Inputs: args, Failures: failures}, err
	}
	c := &checker{unk: unknowns(req.NewInputs)}
	c.dataLimit("dataLimit", args.DataLimit, 0)
	c.dataLimit("nationalRoamingDataLimit", args.NationalRoamingDataLimit, 0)
	c.dataLimit("internationalRoamingDataLimit", args.InternationalRoamingDataLimit, 0)
	if v := args.DataMetering; v != nil {
		c.oneOf("dataMetering", *v, []string{"payg", "quota-1", "quota-10", "quota-50"})
	}
	if v := args.DataLimitStrategy; v != nil {
		c.oneOf("dataLimitStrategy", *v, []string{"block", "throttle"})
	}
	if v := args.InternationalRoaming; v != nil && !c.unk("internationalRoaming") {
		for _, s := range *v {
			if s != "data" && s != "messaging" {
				c.fail("internationalRoaming", "internationalRoaming must hold only data and messaging, got "+s)
				break
			}
		}
	}
	return infer.CheckResponse[WirelessRatePlanArgs]{Inputs: args, Failures: c.failures}, nil
}

func (r *WirelessRatePlan) Diff(_ context.Context, req infer.DiffRequest[WirelessRatePlanArgs, WirelessRatePlanState]) (infer.DiffResponse, error) {
	o, n := req.State.WirelessRatePlanArgs, req.Inputs
	d := trDiffSet{}
	d.str("uniqueName", o.UniqueName, n.UniqueName)
	d.str("friendlyName", o.FriendlyName, n.FriendlyName)
	d.replaceBool("dataEnabled", o.DataEnabled, n.DataEnabled)
	d.replaceInt("dataLimit", o.DataLimit, n.DataLimit)
	d.replaceStr("dataMetering", o.DataMetering, n.DataMetering)
	d.replaceBool("messagingEnabled", o.MessagingEnabled, n.MessagingEnabled)
	d.replaceBool("voiceEnabled", o.VoiceEnabled, n.VoiceEnabled)
	d.replaceBool("nationalRoamingEnabled", o.NationalRoamingEnabled, n.NationalRoamingEnabled)
	d.replaceInt("nationalRoamingDataLimit", o.NationalRoamingDataLimit, n.NationalRoamingDataLimit)
	d.replaceList("internationalRoaming", o.InternationalRoaming, n.InternationalRoaming)
	d.replaceInt("internationalRoamingDataLimit", o.InternationalRoamingDataLimit, n.InternationalRoamingDataLimit)
	d.replaceStr("dataLimitStrategy", o.DataLimitStrategy, n.DataLimitStrategy)
	return d.response(), nil
}

func (a WirelessRatePlanArgs) createForm() url.Values {
	form := url.Values{}
	setS(form, "UniqueName", a.UniqueName)
	setS(form, "FriendlyName", a.FriendlyName)
	setB(form, "DataEnabled", a.DataEnabled)
	setI(form, "DataLimit", a.DataLimit)
	setS(form, "DataMetering", a.DataMetering)
	setB(form, "MessagingEnabled", a.MessagingEnabled)
	setB(form, "VoiceEnabled", a.VoiceEnabled)
	setB(form, "NationalRoamingEnabled", a.NationalRoamingEnabled)
	setI(form, "NationalRoamingDataLimit", a.NationalRoamingDataLimit)
	if a.InternationalRoaming != nil {
		for _, s := range *a.InternationalRoaming {
			form.Add("InternationalRoaming", s)
		}
	}
	setI(form, "InternationalRoamingDataLimit", a.InternationalRoamingDataLimit)
	setS(form, "DataLimitStrategy", a.DataLimitStrategy)
	return form
}

// adopt sets the inputs Twilio cannot change after creation to what it reports, so the state holds
// the plan as it is and a later program that names the same values is not a change. dataLimitStrategy
// is not reported and stays as stated.
func (a WirelessRatePlanArgs) adopt(o obj) WirelessRatePlanArgs {
	a.DataEnabled = adoptAPI(a.DataEnabled, o.optBool("data_enabled"))
	a.DataLimit = adoptAPI(a.DataLimit, o.optInt("data_limit"))
	a.DataMetering = adoptAPI(a.DataMetering, nonEmpty(o.optStr("data_metering")))
	a.MessagingEnabled = adoptAPI(a.MessagingEnabled, o.optBool("messaging_enabled"))
	a.VoiceEnabled = adoptAPI(a.VoiceEnabled, o.optBool("voice_enabled"))
	a.NationalRoamingEnabled = adoptAPI(a.NationalRoamingEnabled, o.optBool("national_roaming_enabled"))
	a.NationalRoamingDataLimit = adoptAPI(a.NationalRoamingDataLimit, o.optInt("national_roaming_data_limit"))
	a.InternationalRoaming = adoptAPI(a.InternationalRoaming, o.optList("international_roaming"))
	a.InternationalRoamingDataLimit = adoptAPI(a.InternationalRoamingDataLimit, o.optInt("international_roaming_data_limit"))
	return a
}

// keep fills the inputs Twilio cannot change from the stored state: an update never changes them, so
// a left-out input keeps the value Twilio reported.
func (a WirelessRatePlanArgs) keep(s WirelessRatePlanArgs) WirelessRatePlanArgs {
	a.DataEnabled = adoptAPI(s.DataEnabled, a.DataEnabled)
	a.DataLimit = adoptAPI(s.DataLimit, a.DataLimit)
	a.DataMetering = adoptAPI(s.DataMetering, a.DataMetering)
	a.MessagingEnabled = adoptAPI(s.MessagingEnabled, a.MessagingEnabled)
	a.VoiceEnabled = adoptAPI(s.VoiceEnabled, a.VoiceEnabled)
	a.NationalRoamingEnabled = adoptAPI(s.NationalRoamingEnabled, a.NationalRoamingEnabled)
	a.NationalRoamingDataLimit = adoptAPI(s.NationalRoamingDataLimit, a.NationalRoamingDataLimit)
	a.InternationalRoaming = adoptAPI(s.InternationalRoaming, a.InternationalRoaming)
	a.InternationalRoamingDataLimit = adoptAPI(s.InternationalRoamingDataLimit, a.InternationalRoamingDataLimit)
	a.DataLimitStrategy = adoptAPI(s.DataLimitStrategy, a.DataLimitStrategy)
	return a
}

func (r *WirelessRatePlan) Create(ctx context.Context, req infer.CreateRequest[WirelessRatePlanArgs]) (infer.CreateResponse[WirelessRatePlanState], error) {
	type resp = infer.CreateResponse[WirelessRatePlanState]
	a := req.Inputs
	if req.DryRun {
		return resp{Output: WirelessRatePlanState{WirelessRatePlanArgs: a}}, nil
	}
	c, err := clientFor(ctx)
	if err != nil {
		return resp{}, err
	}
	o, err := c.do(ctx, "POST", wirelessProduct, wirelessPath("RatePlans", ""), nil, a.createForm())
	if err != nil {
		return resp{}, fmt.Errorf("create Wireless rate plan: %w", err)
	}
	st := WirelessRatePlanState{WirelessRatePlanArgs: a.adopt(o)}
	st.Sid, st.DateCreated, st.DateUpdated = trSid(o)
	return resp{ID: st.Sid, Output: st}, nil
}

func (r *WirelessRatePlan) Read(ctx context.Context, req infer.ReadRequest[WirelessRatePlanArgs, WirelessRatePlanState]) (infer.ReadResponse[WirelessRatePlanArgs, WirelessRatePlanState], error) {
	type resp = infer.ReadResponse[WirelessRatePlanArgs, WirelessRatePlanState]
	c, err := clientFor(ctx)
	if err != nil {
		return resp{}, err
	}
	o, err := c.do(ctx, "GET", wirelessProduct, wirelessPath("RatePlans", req.ID), nil, nil)
	if err != nil {
		if isNotFound(err) {
			if err := notFoundOnImport("Wireless rate plan", req.ID, req.Inputs, req.State); err != nil {
				return resp{}, err
			}
			return resp{}, nil
		}
		return resp{}, fmt.Errorf("read Wireless rate plan %s: %w", req.ID, err)
	}
	in := req.Inputs.adopt(o)
	imported := req.State.Sid == ""
	in.UniqueName = trStr(req.Inputs.UniqueName, o.optStr("unique_name"), "", imported)
	in.FriendlyName = trStr(req.Inputs.FriendlyName, o.optStr("friendly_name"), "", imported)
	st := WirelessRatePlanState{WirelessRatePlanArgs: in}
	st.Sid, st.DateCreated, st.DateUpdated = trSid(o)
	return resp{ID: req.ID, Inputs: in, State: st}, nil
}

func (r *WirelessRatePlan) Update(ctx context.Context, req infer.UpdateRequest[WirelessRatePlanArgs, WirelessRatePlanState]) (infer.UpdateResponse[WirelessRatePlanState], error) {
	type resp = infer.UpdateResponse[WirelessRatePlanState]
	st := req.State
	st.WirelessRatePlanArgs = req.Inputs.keep(req.State.WirelessRatePlanArgs)
	if req.DryRun {
		return resp{Output: st}, nil
	}
	c, err := clientFor(ctx)
	if err != nil {
		return resp{}, err
	}
	form := url.Values{}
	setS(form, "UniqueName", req.Inputs.UniqueName)
	setS(form, "FriendlyName", req.Inputs.FriendlyName)
	o, err := c.do(ctx, "POST", wirelessProduct, wirelessPath("RatePlans", req.ID), nil, form)
	if err != nil {
		return resp{}, fmt.Errorf("update Wireless rate plan %s: %w", req.ID, err)
	}
	st.DateUpdated = o.str("date_updated")
	return resp{Output: st}, nil
}

func (r *WirelessRatePlan) Delete(ctx context.Context, req infer.DeleteRequest[WirelessRatePlanState]) (infer.DeleteResponse, error) {
	c, err := clientFor(ctx)
	if err != nil {
		return infer.DeleteResponse{}, err
	}
	if _, err := c.do(ctx, "DELETE", wirelessProduct, wirelessPath("RatePlans", req.ID), nil, nil); err != nil && !isNotFound(err) {
		return infer.DeleteResponse{}, fmt.Errorf("delete Wireless rate plan %s: %w", req.ID, err)
	}
	return infer.DeleteResponse{}, nil
}
