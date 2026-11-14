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

// The Twilio defaults of a Fleet that an import leaves unset.
const supersimDefaultMethod = "POST"

// SuperSimFleet manages a Super SIM Fleet: the group of SIMs that share one Network Access Profile,
// data limit and commands policy. Twilio has no DELETE for a Fleet, so destroying the resource only
// forgets it.
type SuperSimFleet struct{}

// SuperSimFleetArgs are the inputs of a SuperSimFleet.
type SuperSimFleetArgs struct {
	NetworkAccessProfile string  `pulumi:"networkAccessProfile"`
	UniqueName           *string `pulumi:"uniqueName,optional"`
	DataEnabled          *bool   `pulumi:"dataEnabled,optional"`
	DataLimit            *int    `pulumi:"dataLimit,optional"`
	IpCommandsUrl        *string `pulumi:"ipCommandsUrl,optional"`
	IpCommandsMethod     *string `pulumi:"ipCommandsMethod,optional"`
	SmsCommandsEnabled   *bool   `pulumi:"smsCommandsEnabled,optional"`
	SmsCommandsUrl       *string `pulumi:"smsCommandsUrl,optional"`
	SmsCommandsMethod    *string `pulumi:"smsCommandsMethod,optional"`
}

func (a *SuperSimFleetArgs) Annotate(an infer.Annotator) {
	an.Describe(&a.NetworkAccessProfile, "The SID (HA...) or unique name of the SuperSimNetworkAccessProfile that controls which cellular networks the SIMs of the Fleet can connect to.")
	an.Describe(&a.UniqueName, "An application-defined name that uniquely identifies the Fleet and can be used in place of its SID. Removing it leaves the name unchanged.")
	an.Describe(&a.DataEnabled, "Whether the SIMs of the Fleet can use 2G/3G/4G/LTE/CAT-M data connectivity (Twilio default true). Twilio cannot change it after creation, so changing it replaces the Fleet.")
	an.Describe(&a.DataLimit, "The data (download and upload combined) in megabytes each SIM of the Fleet may use in a billing period, from 1 to 2,000,000 (Twilio default 1,000).")
	an.Describe(&a.IpCommandsUrl, "The URL that receives a webhook when a SIM of the Fleet sends an IP Command to the special IP address. Removing it leaves the URL unchanged.")
	an.Describe(&a.IpCommandsMethod, "HTTP method of the IP Commands webhook, GET or POST (Twilio default POST).")
	an.Describe(&a.SmsCommandsEnabled, "Whether the SIMs of the Fleet can send and receive machine-to-machine SMS through Commands. Twilio cannot change it after creation, so changing it replaces the Fleet.")
	an.Describe(&a.SmsCommandsUrl, "The URL that receives a webhook when a SIM of the Fleet sends an SMS to the SMS Commands number. Removing it leaves the URL unchanged.")
	an.Describe(&a.SmsCommandsMethod, "HTTP method of the SMS Commands webhook, GET or POST (Twilio default POST).")
}

// SuperSimFleetState is the persisted state of a SuperSimFleet.
type SuperSimFleetState struct {
	SuperSimFleetArgs

	Sid                     string `pulumi:"sid"`
	NetworkAccessProfileSid string `pulumi:"networkAccessProfileSid"`
	DataMetering            string `pulumi:"dataMetering"`
	DateCreated             string `pulumi:"dateCreated"`
	DateUpdated             string `pulumi:"dateUpdated"`
}

func (s *SuperSimFleetState) Annotate(a infer.Annotator) {
	a.Describe(&s.Sid, "The Fleet SID (HF...).")
	a.Describe(&s.NetworkAccessProfileSid, "The SID (HA...) of the Network Access Profile Twilio reports for the Fleet.")
	a.Describe(&s.DataMetering, "The model by which the usage of the SIMs is metered and billed (payg); Twilio reports it and it cannot be set.")
	a.Describe(&s.DateCreated, "When the Fleet was created (ISO 8601).")
	a.Describe(&s.DateUpdated, "When the Fleet was last updated (ISO 8601).")
}

func (r *SuperSimFleet) Annotate(a infer.Annotator) {
	a.Describe(&r, "A Super SIM Fleet. The resource ID is the Fleet SID. Twilio has no way to delete a Fleet, so destroying the resource forgets it and leaves the Fleet on Twilio. Optional settings that are set are sent and kept in step with Twilio; one that is removed leaves the setting as it is on Twilio.")
}

var _ infer.CustomResource[SuperSimFleetArgs, SuperSimFleetState] = (*SuperSimFleet)(nil)
var _ infer.CustomCheck[SuperSimFleetArgs] = (*SuperSimFleet)(nil)
var _ infer.CustomDiff[SuperSimFleetArgs, SuperSimFleetState] = (*SuperSimFleet)(nil)
var _ infer.CustomRead[SuperSimFleetArgs, SuperSimFleetState] = (*SuperSimFleet)(nil)
var _ infer.CustomUpdate[SuperSimFleetArgs, SuperSimFleetState] = (*SuperSimFleet)(nil)
var _ infer.CustomDelete[SuperSimFleetState] = (*SuperSimFleet)(nil)

func (r *SuperSimFleet) Check(ctx context.Context, req infer.CheckRequest) (infer.CheckResponse[SuperSimFleetArgs], error) {
	args, failures, err := infer.DefaultCheck[SuperSimFleetArgs](ctx, req.NewInputs)
	if err != nil || len(failures) > 0 {
		return infer.CheckResponse[SuperSimFleetArgs]{Inputs: args, Failures: failures}, err
	}
	c := &checker{unk: unknowns(req.NewInputs)}
	c.required("networkAccessProfile", args.NetworkAccessProfile)
	c.dataLimit("dataLimit", args.DataLimit, 1)
	if v := args.IpCommandsMethod; v != nil {
		c.oneOf("ipCommandsMethod", *v, []string{"GET", "POST"})
	}
	if v := args.SmsCommandsMethod; v != nil {
		c.oneOf("smsCommandsMethod", *v, []string{"GET", "POST"})
	}
	return infer.CheckResponse[SuperSimFleetArgs]{Inputs: args, Failures: c.failures}, nil
}

func (r *SuperSimFleet) Diff(_ context.Context, req infer.DiffRequest[SuperSimFleetArgs, SuperSimFleetState]) (infer.DiffResponse, error) {
	o, n := req.State.SuperSimFleetArgs, req.Inputs
	d := trDiffSet{}
	d.requiredStr("networkAccessProfile", o.NetworkAccessProfile, n.NetworkAccessProfile)
	d.str("uniqueName", o.UniqueName, n.UniqueName)
	d.replaceBool("dataEnabled", o.DataEnabled, n.DataEnabled)
	d.intDef(supersimFleetDefaults, "dataLimit", o.DataLimit, n.DataLimit)
	d.strDef(supersimFleetDefaults, "ipCommandsUrl", o.IpCommandsUrl, n.IpCommandsUrl)
	d.strDef(supersimFleetDefaults, "ipCommandsMethod", o.IpCommandsMethod, n.IpCommandsMethod)
	d.replaceBool("smsCommandsEnabled", o.SmsCommandsEnabled, n.SmsCommandsEnabled)
	d.strDef(supersimFleetDefaults, "smsCommandsUrl", o.SmsCommandsUrl, n.SmsCommandsUrl)
	d.strDef(supersimFleetDefaults, "smsCommandsMethod", o.SmsCommandsMethod, n.SmsCommandsMethod)
	return d.response(), nil
}

// createForm is the body of the create call: every input.
func (a SuperSimFleetArgs) createForm() url.Values {
	form := url.Values{}
	form.Set("NetworkAccessProfile", a.NetworkAccessProfile)
	setS(form, "UniqueName", a.UniqueName)
	setB(form, "DataEnabled", a.DataEnabled)
	setI(form, "DataLimit", a.DataLimit)
	setS(form, "IpCommandsUrl", a.IpCommandsUrl)
	setS(form, "IpCommandsMethod", a.IpCommandsMethod)
	setB(form, "SmsCommandsEnabled", a.SmsCommandsEnabled)
	setS(form, "SmsCommandsUrl", a.SmsCommandsUrl)
	setS(form, "SmsCommandsMethod", a.SmsCommandsMethod)
	return form
}

// updateForm is the body of the update call: the inputs Twilio can change (not dataEnabled and
// smsCommandsEnabled, which replace the Fleet).
func (a SuperSimFleetArgs) updateForm() url.Values {
	form := a.createForm()
	form.Del("DataEnabled")
	form.Del("SmsCommandsEnabled")
	return form
}

func (r *SuperSimFleet) Create(ctx context.Context, req infer.CreateRequest[SuperSimFleetArgs]) (infer.CreateResponse[SuperSimFleetState], error) {
	type resp = infer.CreateResponse[SuperSimFleetState]
	a := req.Inputs
	if req.DryRun {
		return resp{Output: SuperSimFleetState{SuperSimFleetArgs: a}}, nil
	}
	c, err := clientFor(ctx)
	if err != nil {
		return resp{}, err
	}
	o, err := c.do(ctx, "POST", supersimProduct, supersimPath("Fleets"), nil, a.createForm())
	if err != nil {
		return resp{}, fmt.Errorf("create Super SIM fleet: %w", err)
	}
	// dataEnabled and smsCommandsEnabled cannot change later, so the state keeps what Twilio set,
	// which is what a later program that states them is compared with.
	a.DataEnabled = adoptAPI(a.DataEnabled, o.optBool("data_enabled"))
	a.SmsCommandsEnabled = adoptAPI(a.SmsCommandsEnabled, o.optBool("sms_commands_enabled"))
	st := SuperSimFleetState{SuperSimFleetArgs: a}
	st.fill(o)
	return resp{ID: st.Sid, Output: st}, nil
}

// fill sets the values Twilio reports.
func (s *SuperSimFleetState) fill(o obj) {
	s.Sid, s.DateCreated, s.DateUpdated = trSid(o)
	s.NetworkAccessProfileSid = o.str("network_access_profile_sid")
	s.DataMetering = o.str("data_metering")
}

func (r *SuperSimFleet) Read(ctx context.Context, req infer.ReadRequest[SuperSimFleetArgs, SuperSimFleetState]) (infer.ReadResponse[SuperSimFleetArgs, SuperSimFleetState], error) {
	type resp = infer.ReadResponse[SuperSimFleetArgs, SuperSimFleetState]
	c, err := clientFor(ctx)
	if err != nil {
		return resp{}, err
	}
	o, err := c.do(ctx, "GET", supersimProduct, supersimPath("Fleets", req.ID), nil, nil)
	if err != nil {
		if isNotFound(err) {
			if err := notFoundOnImport("Super SIM fleet", req.ID, req.Inputs, req.State); err != nil {
				return resp{}, err
			}
			return resp{}, nil
		}
		return resp{}, fmt.Errorf("read Super SIM fleet %s: %w", req.ID, err)
	}
	in := req.Inputs
	imported := req.State.Sid == ""
	apiProfile := o.str("network_access_profile_sid")
	switch {
	case apiProfile == "":
	case imported, supersimProfileSidRE.MatchString(in.NetworkAccessProfile):
		in.NetworkAccessProfile = apiProfile
	case req.State.NetworkAccessProfileSid != "" && req.State.NetworkAccessProfileSid != apiProfile:
		// The program names the profile by its unique name, which Twilio does not report: a profile
		// swapped outside Pulumi is noticed by its SID and shows as a change.
		in.NetworkAccessProfile = apiProfile
	}
	in.UniqueName = trStr(in.UniqueName, o.optStr("unique_name"), "", imported)
	in.DataEnabled = adoptAPI(in.DataEnabled, o.optBool("data_enabled"))
	in.DataLimit = trInt(in.DataLimit, o.optInt("data_limit"), supersimFleetDefaults.integer("dataLimit"), imported)
	in.IpCommandsUrl = trStr(in.IpCommandsUrl, o.optStr("ip_commands_url"), supersimFleetDefaults.str("ipCommandsUrl"), imported)
	in.IpCommandsMethod = trStr(in.IpCommandsMethod, o.optStr("ip_commands_method"), supersimFleetDefaults.str("ipCommandsMethod"), imported)
	in.SmsCommandsEnabled = adoptAPI(in.SmsCommandsEnabled, o.optBool("sms_commands_enabled"))
	in.SmsCommandsUrl = trStr(in.SmsCommandsUrl, o.optStr("sms_commands_url"), supersimFleetDefaults.str("smsCommandsUrl"), imported)
	in.SmsCommandsMethod = trStr(in.SmsCommandsMethod, o.optStr("sms_commands_method"), supersimFleetDefaults.str("smsCommandsMethod"), imported)
	st := SuperSimFleetState{SuperSimFleetArgs: in}
	st.fill(o)
	return resp{ID: req.ID, Inputs: in, State: st}, nil
}

func (r *SuperSimFleet) Update(ctx context.Context, req infer.UpdateRequest[SuperSimFleetArgs, SuperSimFleetState]) (infer.UpdateResponse[SuperSimFleetState], error) {
	type resp = infer.UpdateResponse[SuperSimFleetState]
	st := req.State
	st.SuperSimFleetArgs = req.Inputs
	// An input the program leaves out keeps the value Twilio reported for it.
	st.DataEnabled = adoptAPI(req.State.DataEnabled, req.Inputs.DataEnabled)
	st.SmsCommandsEnabled = adoptAPI(req.State.SmsCommandsEnabled, req.Inputs.SmsCommandsEnabled)
	if req.DryRun {
		return resp{Output: st}, nil
	}
	c, err := clientFor(ctx)
	if err != nil {
		return resp{}, err
	}
	o, err := c.do(ctx, "POST", supersimProduct, supersimPath("Fleets", req.ID), nil, req.Inputs.updateForm())
	if err != nil {
		return resp{}, fmt.Errorf("update Super SIM fleet %s: %w", req.ID, err)
	}
	st.fill(o)
	return resp{Output: st}, nil
}

// Delete forgets the Fleet: Twilio has no DELETE for a Fleet.
func (r *SuperSimFleet) Delete(_ context.Context, _ infer.DeleteRequest[SuperSimFleetState]) (infer.DeleteResponse, error) {
	return infer.DeleteResponse{}, nil
}

// supersimFleetDefaults are the Twilio defaults of this resource, shared by Read (import adoption) and Diff.
var supersimFleetDefaults = trDefaults{
	"dataLimit":         1000,
	"ipCommandsUrl":     "",
	"ipCommandsMethod":  supersimDefaultMethod,
	"smsCommandsUrl":    "",
	"smsCommandsMethod": supersimDefaultMethod,
}
