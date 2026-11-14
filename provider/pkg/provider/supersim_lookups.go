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
	"net/url"

	"github.com/pulumi/pulumi-go-provider/infer"
)

// Lookups of the Super SIM objects and of the Wireless rate plan. Each reads what Twilio reports,
// null where Twilio reports nothing.

// SuperSimNetworkIdentifier is one MCC-MNC pair of a network.
type SuperSimNetworkIdentifier struct {
	Mcc string `pulumi:"mcc"`
	Mnc string `pulumi:"mnc"`
}

func (a *SuperSimNetworkIdentifier) Annotate(an infer.Annotator) {
	an.Describe(&a.Mcc, "The mobile country code.")
	an.Describe(&a.Mnc, "The mobile network code of the operator network.")
}

// SuperSimNetwork is a cellular network a Network Access Profile can allow.
type SuperSimNetwork struct {
	Sid          string                      `pulumi:"sid"`
	FriendlyName *string                     `pulumi:"friendlyName,optional"`
	IsoCountry   *string                     `pulumi:"isoCountry,optional"`
	Identifiers  []SuperSimNetworkIdentifier `pulumi:"identifiers"`
}

func (a *SuperSimNetwork) Annotate(an infer.Annotator) {
	an.Describe(&a.Sid, "The network SID (HW...), the value networks of a SuperSimNetworkAccessProfile takes.")
	an.Describe(&a.FriendlyName, "A human readable name of the network.")
	an.Describe(&a.IsoCountry, "The ISO 3166-1 alpha-2 country code of the network.")
	an.Describe(&a.Identifiers, "The MCC-MNC pairs the network covers; empty when Twilio reports none.")
}

func supersimNetwork(o obj) SuperSimNetwork {
	ids := []SuperSimNetworkIdentifier{}
	if l, ok := o.get("identifiers").([]any); ok {
		for _, it := range l {
			if m, ok := it.(map[string]any); ok {
				id := obj(m)
				ids = append(ids, SuperSimNetworkIdentifier{Mcc: id.str("mcc"), Mnc: id.str("mnc")})
			}
		}
	}
	return SuperSimNetwork{
		Sid:          o.str("sid"),
		FriendlyName: nonEmpty(o.optStr("friendly_name")),
		IsoCountry:   nonEmpty(o.optStr("iso_country")),
		Identifiers:  ids,
	}
}

// supersimLookup reads one object for a lookup function.
func supersimLookup(ctx context.Context, fn, what, product, path string) (obj, error) {
	c, err := clientFor(ctx)
	if err != nil {
		return nil, err
	}
	o, err := c.do(ctx, "GET", product, path, nil, nil)
	if err != nil {
		return nil, lookupFailure(fn, what, err)
	}
	return o, nil
}

// ------------------------------------------------------------ getSuperSimFleet

// GetSuperSimFleet looks up a Super SIM Fleet by SID or unique name.
type GetSuperSimFleet struct{}

// GetSuperSimFleetArgs are the lookup arguments.
type GetSuperSimFleetArgs struct {
	Sid string `pulumi:"sid"`
}

func (a *GetSuperSimFleetArgs) Annotate(an infer.Annotator) {
	an.Describe(&a.Sid, "SID (HF...) or unique name of the Fleet.")
}

// GetSuperSimFleetResult is the looked-up Fleet.
type GetSuperSimFleetResult struct {
	Sid                     string  `pulumi:"sid"`
	UniqueName              *string `pulumi:"uniqueName,optional"`
	NetworkAccessProfileSid *string `pulumi:"networkAccessProfileSid,optional"`
	DataEnabled             *bool   `pulumi:"dataEnabled,optional"`
	DataLimit               *int    `pulumi:"dataLimit,optional"`
	DataMetering            *string `pulumi:"dataMetering,optional"`
	IpCommandsUrl           *string `pulumi:"ipCommandsUrl,optional"`
	IpCommandsMethod        *string `pulumi:"ipCommandsMethod,optional"`
	SmsCommandsEnabled      *bool   `pulumi:"smsCommandsEnabled,optional"`
	SmsCommandsUrl          *string `pulumi:"smsCommandsUrl,optional"`
	SmsCommandsMethod       *string `pulumi:"smsCommandsMethod,optional"`
	DateCreated             string  `pulumi:"dateCreated"`
	DateUpdated             string  `pulumi:"dateUpdated"`
}

func (f *GetSuperSimFleet) Annotate(a infer.Annotator) {
	a.Describe(&f, "Look up a Super SIM Fleet by SID or unique name.")
}

func (f *GetSuperSimFleet) Invoke(ctx context.Context, req infer.FunctionRequest[GetSuperSimFleetArgs]) (infer.FunctionResponse[GetSuperSimFleetResult], error) {
	const fn = "getSuperSimFleet"
	type resp = infer.FunctionResponse[GetSuperSimFleetResult]
	if err := requireArg(fn, "sid", req.Input.Sid); err != nil {
		return resp{}, err
	}
	o, err := supersimLookup(ctx, fn, "Super SIM fleet "+req.Input.Sid, supersimProduct, supersimPath("Fleets", req.Input.Sid))
	if err != nil {
		return resp{}, err
	}
	sid, created, updated := trSid(o)
	return resp{Output: GetSuperSimFleetResult{
		Sid:                     sid,
		UniqueName:              nonEmpty(o.optStr("unique_name")),
		NetworkAccessProfileSid: nonEmpty(o.optStr("network_access_profile_sid")),
		DataEnabled:             o.optBool("data_enabled"),
		DataLimit:               o.optInt("data_limit"),
		DataMetering:            nonEmpty(o.optStr("data_metering")),
		IpCommandsUrl:           nonEmpty(o.optStr("ip_commands_url")),
		IpCommandsMethod:        nonEmpty(o.optStr("ip_commands_method")),
		SmsCommandsEnabled:      o.optBool("sms_commands_enabled"),
		SmsCommandsUrl:          nonEmpty(o.optStr("sms_commands_url")),
		SmsCommandsMethod:       nonEmpty(o.optStr("sms_commands_method")),
		DateCreated:             created,
		DateUpdated:             updated,
	}}, nil
}

// ------------------------------------------------------------ getSuperSimNetworkAccessProfile

// GetSuperSimNetworkAccessProfile looks up a Super SIM Network Access Profile and its networks.
type GetSuperSimNetworkAccessProfile struct{}

// GetSuperSimNetworkAccessProfileArgs are the lookup arguments.
type GetSuperSimNetworkAccessProfileArgs struct {
	Sid string `pulumi:"sid"`
}

func (a *GetSuperSimNetworkAccessProfileArgs) Annotate(an infer.Annotator) {
	an.Describe(&a.Sid, "SID (HA...) or unique name of the Network Access Profile.")
}

// GetSuperSimNetworkAccessProfileResult is the looked-up profile.
type GetSuperSimNetworkAccessProfileResult struct {
	Sid         string            `pulumi:"sid"`
	UniqueName  *string           `pulumi:"uniqueName,optional"`
	Networks    []SuperSimNetwork `pulumi:"networks"`
	DateCreated string            `pulumi:"dateCreated"`
	DateUpdated string            `pulumi:"dateUpdated"`
}

func (a *GetSuperSimNetworkAccessProfileResult) Annotate(an infer.Annotator) {
	an.Describe(&a.Networks, "The networks the profile allows, in the order Twilio returns them; empty when it allows none.")
}

func (f *GetSuperSimNetworkAccessProfile) Annotate(a infer.Annotator) {
	a.Describe(&f, "Look up a Super SIM Network Access Profile by SID or unique name, with the networks it allows.")
}

func (f *GetSuperSimNetworkAccessProfile) Invoke(ctx context.Context, req infer.FunctionRequest[GetSuperSimNetworkAccessProfileArgs]) (infer.FunctionResponse[GetSuperSimNetworkAccessProfileResult], error) {
	const fn = "getSuperSimNetworkAccessProfile"
	type resp = infer.FunctionResponse[GetSuperSimNetworkAccessProfileResult]
	if err := requireArg(fn, "sid", req.Input.Sid); err != nil {
		return resp{}, err
	}
	what := "Super SIM network access profile " + req.Input.Sid
	c, err := clientFor(ctx)
	if err != nil {
		return resp{}, err
	}
	o, err := c.do(ctx, "GET", supersimProduct, supersimPath("NetworkAccessProfiles", req.Input.Sid), nil, nil)
	if err != nil {
		return resp{}, lookupFailure(fn, what, err)
	}
	nets := []SuperSimNetwork{}
	err = listPages(ctx, c, fn, supersimProduct, supersimPath("NetworkAccessProfiles", req.Input.Sid, "Networks"), url.Values{"PageSize": {"1000"}}, "networks", func(n obj) bool {
		nets = append(nets, supersimNetwork(n))
		return true
	})
	if err != nil {
		return resp{}, lookupFailure(fn, "the networks of "+what, err)
	}
	sid, created, updated := trSid(o)
	return resp{Output: GetSuperSimNetworkAccessProfileResult{
		Sid:         sid,
		UniqueName:  nonEmpty(o.optStr("unique_name")),
		Networks:    nets,
		DateCreated: created,
		DateUpdated: updated,
	}}, nil
}

// ------------------------------------------------------------ getSuperSimNetwork

// GetSuperSimNetwork looks up one network of the Super SIM catalog.
type GetSuperSimNetwork struct{}

// GetSuperSimNetworkArgs are the lookup arguments.
type GetSuperSimNetworkArgs struct {
	Sid string `pulumi:"sid"`
}

func (a *GetSuperSimNetworkArgs) Annotate(an infer.Annotator) {
	an.Describe(&a.Sid, "SID (HW...) of the network.")
}

func (f *GetSuperSimNetwork) Annotate(a infer.Annotator) {
	a.Describe(&f, "Look up one cellular network of the Super SIM catalog by SID.")
}

func (f *GetSuperSimNetwork) Invoke(ctx context.Context, req infer.FunctionRequest[GetSuperSimNetworkArgs]) (infer.FunctionResponse[SuperSimNetwork], error) {
	const fn = "getSuperSimNetwork"
	type resp = infer.FunctionResponse[SuperSimNetwork]
	if err := requireArg(fn, "sid", req.Input.Sid); err != nil {
		return resp{}, err
	}
	o, err := supersimLookup(ctx, fn, "Super SIM network "+req.Input.Sid, supersimProduct, supersimPath("Networks", req.Input.Sid))
	if err != nil {
		return resp{}, err
	}
	return resp{Output: supersimNetwork(o)}, nil
}

// ------------------------------------------------------------ getSuperSimNetworks

// GetSuperSimNetworks lists the networks of the Super SIM catalog.
type GetSuperSimNetworks struct{}

// GetSuperSimNetworksArgs are the lookup arguments; every one narrows the list.
type GetSuperSimNetworksArgs struct {
	IsoCountry *string `pulumi:"isoCountry,optional"`
	Mcc        *string `pulumi:"mcc,optional"`
	Mnc        *string `pulumi:"mnc,optional"`
}

func (a *GetSuperSimNetworksArgs) Annotate(an infer.Annotator) {
	an.Describe(&a.IsoCountry, "Only networks in this ISO 3166-1 alpha-2 country, for example `US`.")
	an.Describe(&a.Mcc, "Only networks with this mobile country code in their identifiers.")
	an.Describe(&a.Mnc, "Only networks with this mobile network code in their identifiers.")
}

// GetSuperSimNetworksResult is the list.
type GetSuperSimNetworksResult struct {
	Networks []SuperSimNetwork `pulumi:"networks"`
}

func (a *GetSuperSimNetworksResult) Annotate(an infer.Annotator) {
	an.Describe(&a.Networks, "The matching networks, in the order Twilio returns them.")
}

func (f *GetSuperSimNetworks) Annotate(a infer.Annotator) {
	a.Describe(&f, "List the cellular networks of the Super SIM catalog, optionally narrowed by country, MCC or MNC (GET /v1/Networks on supersim.twilio.com). The SIDs are what a SuperSimNetworkAccessProfile takes in networks.")
}

func (f *GetSuperSimNetworks) Invoke(ctx context.Context, req infer.FunctionRequest[GetSuperSimNetworksArgs]) (infer.FunctionResponse[GetSuperSimNetworksResult], error) {
	const fn = "getSuperSimNetworks"
	none := infer.FunctionResponse[GetSuperSimNetworksResult]{}
	c, err := clientFor(ctx)
	if err != nil {
		return none, err
	}
	in := req.Input
	q := url.Values{"PageSize": {"1000"}}
	for k, v := range map[string]*string{"IsoCountry": in.IsoCountry, "Mcc": in.Mcc, "Mnc": in.Mnc} {
		if v != nil && *v != "" {
			q.Set(k, *v)
		}
	}
	out := []SuperSimNetwork{}
	err = listPages(ctx, c, fn, supersimProduct, supersimPath("Networks"), q, "networks", func(o obj) bool {
		out = append(out, supersimNetwork(o))
		return true
	})
	if err != nil {
		return none, lookupFailure(fn, "Super SIM networks", err)
	}
	return infer.FunctionResponse[GetSuperSimNetworksResult]{Output: GetSuperSimNetworksResult{Networks: out}}, nil
}

// ------------------------------------------------------------ getWirelessRatePlan

// GetWirelessRatePlan looks up a Programmable Wireless Rate Plan.
type GetWirelessRatePlan struct{}

// GetWirelessRatePlanArgs are the lookup arguments.
type GetWirelessRatePlanArgs struct {
	Sid string `pulumi:"sid"`
}

func (a *GetWirelessRatePlanArgs) Annotate(an infer.Annotator) {
	an.Describe(&a.Sid, "SID (WP...) or unique name of the Rate Plan.")
}

// GetWirelessRatePlanResult is the looked-up plan.
type GetWirelessRatePlanResult struct {
	Sid                           string    `pulumi:"sid"`
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
	DateCreated                   string    `pulumi:"dateCreated"`
	DateUpdated                   string    `pulumi:"dateUpdated"`
}

func (f *GetWirelessRatePlan) Annotate(a infer.Annotator) {
	a.Describe(&f, "Look up a Programmable Wireless Rate Plan by SID or unique name.")
}

func (f *GetWirelessRatePlan) Invoke(ctx context.Context, req infer.FunctionRequest[GetWirelessRatePlanArgs]) (infer.FunctionResponse[GetWirelessRatePlanResult], error) {
	const fn = "getWirelessRatePlan"
	type resp = infer.FunctionResponse[GetWirelessRatePlanResult]
	if err := requireArg(fn, "sid", req.Input.Sid); err != nil {
		return resp{}, err
	}
	o, err := supersimLookup(ctx, fn, "Wireless rate plan "+req.Input.Sid, wirelessProduct, wirelessPath("RatePlans", req.Input.Sid))
	if err != nil {
		return resp{}, err
	}
	sid, created, updated := trSid(o)
	return resp{Output: GetWirelessRatePlanResult{
		Sid:                           sid,
		UniqueName:                    nonEmpty(o.optStr("unique_name")),
		FriendlyName:                  nonEmpty(o.optStr("friendly_name")),
		DataEnabled:                   o.optBool("data_enabled"),
		DataLimit:                     o.optInt("data_limit"),
		DataMetering:                  nonEmpty(o.optStr("data_metering")),
		MessagingEnabled:              o.optBool("messaging_enabled"),
		VoiceEnabled:                  o.optBool("voice_enabled"),
		NationalRoamingEnabled:        o.optBool("national_roaming_enabled"),
		NationalRoamingDataLimit:      o.optInt("national_roaming_data_limit"),
		InternationalRoaming:          o.optList("international_roaming"),
		InternationalRoamingDataLimit: o.optInt("international_roaming_data_limit"),
		DateCreated:                   created,
		DateUpdated:                   updated,
	}}, nil
}
