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
	"regexp"
	"slices"

	p "github.com/pulumi/pulumi-go-provider"
	"github.com/pulumi/pulumi-go-provider/infer"
)

// The resources in this file family configure Super SIM (supersim.twilio.com v1): the Network
// Access Profile that decides which cellular networks a SIM may use, and the Fleet that groups SIMs
// under one data and commands policy. SIMs, eSIM profiles, commands, usage and billing are runtime
// or physical data and are not managed. The shapes follow twilio-oai spec/json/twilio_supersim_v1.json.
//
// Twilio has no DELETE for a Fleet or a Network Access Profile, so destroying either only forgets it.

const supersimProduct = "supersim"

var (
	supersimProfileSidRE = regexp.MustCompile(`^HA[0-9a-fA-F]{32}$`)
	supersimNetworkSidRE = regexp.MustCompile(`^HW[0-9a-fA-F]{32}$`)
)

// supersimPath is /v1/<coll>[/sid[/sub[/subSid]]].
func supersimPath(coll string, rest ...string) string {
	path := "/v1/" + coll
	for _, r := range rest {
		if r != "" {
			path += "/" + url.PathEscape(r)
		}
	}
	return path
}

// replaceStr, replaceBool, replaceInt and replaceList record a replacement of an optional input
// that Twilio cannot change after creation. The stored value is what Twilio reported, so setting
// the value Twilio already has is not a change; removing the input is not a change either.
func (d trDiffSet) replaceStr(name string, o, n *string) {
	if n != nil && (o == nil || *o != *n) {
		d[name] = p.PropertyDiff{Kind: p.UpdateReplace}
	}
}

func (d trDiffSet) replaceBool(name string, o, n *bool) {
	if n != nil && (o == nil || *o != *n) {
		d[name] = p.PropertyDiff{Kind: p.UpdateReplace}
	}
}

func (d trDiffSet) replaceInt(name string, o, n *int) {
	if n != nil && (o == nil || *o != *n) {
		d[name] = p.PropertyDiff{Kind: p.UpdateReplace}
	}
}

func (d trDiffSet) replaceList(name string, o, n *[]string) {
	if n == nil {
		return
	}
	if o == nil || !sameSet(*o, *n) {
		d[name] = p.PropertyDiff{Kind: p.UpdateReplace}
	}
}

// adoptAPI is the refreshed value of an input Twilio cannot change after creation: what Twilio
// reports when it reports anything, else what the program stated. Keeping Twilio's value even for
// an input the program leaves out is what lets a later program state it without a replacement
// when it names the value Twilio already has.
func adoptAPI[T any](in, api *T) *T {
	if api != nil {
		return api
	}
	return in
}

// dataLimit fails a data limit below min or above 2,000,000 megabytes (2 TB).
func (c *checker) dataLimit(prop string, v *int, min int) {
	if v != nil && !c.unk(prop) && (*v < min || *v > 2000000) {
		c.fail(prop, fmt.Sprintf("%s must be between %d and 2000000 megabytes", prop, min))
	}
}

// ------------------------------------------------------------ SuperSimNetworkAccessProfile

// SuperSimNetworkAccessProfile manages a Super SIM Network Access Profile: the set of cellular
// networks the SIMs of a Fleet may connect to.
type SuperSimNetworkAccessProfile struct{}

// SuperSimNetworkAccessProfileArgs are the inputs of a SuperSimNetworkAccessProfile.
type SuperSimNetworkAccessProfileArgs struct {
	UniqueName *string   `pulumi:"uniqueName,optional"`
	Networks   *[]string `pulumi:"networks,optional"`
}

func (a *SuperSimNetworkAccessProfileArgs) Annotate(an infer.Annotator) {
	an.Describe(&a.UniqueName, "An application-defined name that uniquely identifies the profile and can be used in place of its SID. Removing it leaves the name unchanged.")
	an.Describe(&a.Networks, "SIDs (HW...) of the networks the profile allows connections to; look them up with getSuperSimNetworks. When set, the profile holds exactly these networks (the provider adds and removes the difference); when omitted, the networks are not managed. Order is not significant.")
}

// SuperSimNetworkAccessProfileState is the persisted state of a SuperSimNetworkAccessProfile.
type SuperSimNetworkAccessProfileState struct {
	SuperSimNetworkAccessProfileArgs

	Sid         string `pulumi:"sid"`
	DateCreated string `pulumi:"dateCreated"`
	DateUpdated string `pulumi:"dateUpdated"`
}

func (s *SuperSimNetworkAccessProfileState) Annotate(a infer.Annotator) {
	a.Describe(&s.Sid, "The profile SID (HA...).")
	a.Describe(&s.DateCreated, "When the profile was created (ISO 8601).")
	a.Describe(&s.DateUpdated, "When the profile was last updated (ISO 8601).")
}

func (r *SuperSimNetworkAccessProfile) Annotate(a infer.Annotator) {
	a.Describe(&r, "A Super SIM Network Access Profile. The resource ID is the profile SID. Twilio has no way to delete a profile, so destroying the resource forgets it and leaves the profile (and its unique name, which stays taken) on Twilio.")
}

var _ infer.CustomResource[SuperSimNetworkAccessProfileArgs, SuperSimNetworkAccessProfileState] = (*SuperSimNetworkAccessProfile)(nil)
var _ infer.CustomCheck[SuperSimNetworkAccessProfileArgs] = (*SuperSimNetworkAccessProfile)(nil)
var _ infer.CustomDiff[SuperSimNetworkAccessProfileArgs, SuperSimNetworkAccessProfileState] = (*SuperSimNetworkAccessProfile)(nil)
var _ infer.CustomRead[SuperSimNetworkAccessProfileArgs, SuperSimNetworkAccessProfileState] = (*SuperSimNetworkAccessProfile)(nil)
var _ infer.CustomUpdate[SuperSimNetworkAccessProfileArgs, SuperSimNetworkAccessProfileState] = (*SuperSimNetworkAccessProfile)(nil)
var _ infer.CustomDelete[SuperSimNetworkAccessProfileState] = (*SuperSimNetworkAccessProfile)(nil)

func (r *SuperSimNetworkAccessProfile) Check(ctx context.Context, req infer.CheckRequest) (infer.CheckResponse[SuperSimNetworkAccessProfileArgs], error) {
	args, failures, err := infer.DefaultCheck[SuperSimNetworkAccessProfileArgs](ctx, req.NewInputs)
	if err != nil || len(failures) > 0 {
		return infer.CheckResponse[SuperSimNetworkAccessProfileArgs]{Inputs: args, Failures: failures}, err
	}
	c := &checker{unk: unknowns(req.NewInputs)}
	if args.Networks != nil && !c.unk("networks") {
		for _, n := range *args.Networks {
			if !supersimNetworkSidRE.MatchString(n) {
				c.fail("networks", "networks must hold SIDs of Super SIM networks (HW followed by 32 hex characters), got "+n)
				break
			}
		}
	}
	return infer.CheckResponse[SuperSimNetworkAccessProfileArgs]{Inputs: args, Failures: c.failures}, nil
}

func (r *SuperSimNetworkAccessProfile) Diff(_ context.Context, req infer.DiffRequest[SuperSimNetworkAccessProfileArgs, SuperSimNetworkAccessProfileState]) (infer.DiffResponse, error) {
	o, n := req.State.SuperSimNetworkAccessProfileArgs, req.Inputs
	d := trDiffSet{}
	d.str("uniqueName", o.UniqueName, n.UniqueName)
	d.videoListDiff("networks", o.Networks, n.Networks)
	return d.response(), nil
}

// supersimProfileNetworks lists the SIDs of the networks of a profile.
func supersimProfileNetworks(ctx context.Context, c *Client, fn, sid string) ([]string, error) {
	out := []string{}
	err := listPages(ctx, c, fn, supersimProduct, supersimPath("NetworkAccessProfiles", sid, "Networks"), url.Values{"PageSize": {"1000"}}, "networks", func(o obj) bool {
		out = append(out, o.str("sid"))
		return true
	})
	return out, err
}

func (r *SuperSimNetworkAccessProfile) Create(ctx context.Context, req infer.CreateRequest[SuperSimNetworkAccessProfileArgs]) (infer.CreateResponse[SuperSimNetworkAccessProfileState], error) {
	type resp = infer.CreateResponse[SuperSimNetworkAccessProfileState]
	a := req.Inputs
	if req.DryRun {
		return resp{Output: SuperSimNetworkAccessProfileState{SuperSimNetworkAccessProfileArgs: a}}, nil
	}
	c, err := clientFor(ctx)
	if err != nil {
		return resp{}, err
	}
	form := url.Values{}
	setS(form, "UniqueName", a.UniqueName)
	if a.Networks != nil {
		for _, n := range slices.Clone(*a.Networks) {
			form.Add("Networks", n)
		}
	}
	o, err := c.do(ctx, "POST", supersimProduct, supersimPath("NetworkAccessProfiles"), nil, form)
	if err != nil {
		return resp{}, fmt.Errorf("create Super SIM network access profile: %w", err)
	}
	st := SuperSimNetworkAccessProfileState{SuperSimNetworkAccessProfileArgs: a}
	st.Sid, st.DateCreated, st.DateUpdated = trSid(o)
	return resp{ID: st.Sid, Output: st}, nil
}

func (r *SuperSimNetworkAccessProfile) Read(ctx context.Context, req infer.ReadRequest[SuperSimNetworkAccessProfileArgs, SuperSimNetworkAccessProfileState]) (infer.ReadResponse[SuperSimNetworkAccessProfileArgs, SuperSimNetworkAccessProfileState], error) {
	type resp = infer.ReadResponse[SuperSimNetworkAccessProfileArgs, SuperSimNetworkAccessProfileState]
	c, err := clientFor(ctx)
	if err != nil {
		return resp{}, err
	}
	o, err := c.do(ctx, "GET", supersimProduct, supersimPath("NetworkAccessProfiles", req.ID), nil, nil)
	if err != nil {
		if isNotFound(err) {
			if err := notFoundOnImport("Super SIM network access profile", req.ID, req.Inputs, req.State); err != nil {
				return resp{}, err
			}
			return resp{}, nil
		}
		return resp{}, fmt.Errorf("read Super SIM network access profile %s: %w", req.ID, err)
	}
	nets, err := supersimProfileNetworks(ctx, c, "read", req.ID)
	if err != nil {
		return resp{}, fmt.Errorf("read the networks of Super SIM network access profile %s: %w", req.ID, err)
	}
	in := req.Inputs
	imported := req.State.Sid == ""
	in.UniqueName = trStr(in.UniqueName, o.optStr("unique_name"), "", imported)
	in.Networks = cvList(in.Networks, &nets, imported)
	st := SuperSimNetworkAccessProfileState{SuperSimNetworkAccessProfileArgs: in}
	st.Sid, st.DateCreated, st.DateUpdated = trSid(o)
	return resp{ID: req.ID, Inputs: in, State: st}, nil
}

func (r *SuperSimNetworkAccessProfile) Update(ctx context.Context, req infer.UpdateRequest[SuperSimNetworkAccessProfileArgs, SuperSimNetworkAccessProfileState]) (infer.UpdateResponse[SuperSimNetworkAccessProfileState], error) {
	type resp = infer.UpdateResponse[SuperSimNetworkAccessProfileState]
	st := req.State
	st.SuperSimNetworkAccessProfileArgs = req.Inputs
	if req.DryRun {
		return resp{Output: st}, nil
	}
	c, err := clientFor(ctx)
	if err != nil {
		return resp{}, err
	}
	if n := req.Inputs.UniqueName; n != nil && (req.State.UniqueName == nil || *req.State.UniqueName != *n) {
		o, err := c.do(ctx, "POST", supersimProduct, supersimPath("NetworkAccessProfiles", req.ID), nil, url.Values{"UniqueName": {*n}})
		if err != nil {
			return resp{}, fmt.Errorf("update Super SIM network access profile %s: %w", req.ID, err)
		}
		st.DateUpdated = o.str("date_updated")
	}
	if want := req.Inputs.Networks; want != nil {
		// Reconcile against what Twilio holds now, not against the stored state, so a change made
		// outside Pulumi is corrected too. Networks are added before any is removed, so the profile
		// never allows fewer networks than both the old and the new set.
		have, err := supersimProfileNetworks(ctx, c, "update", req.ID)
		if err != nil {
			return resp{}, fmt.Errorf("read the networks of Super SIM network access profile %s: %w", req.ID, err)
		}
		for _, n := range *want {
			if slices.Contains(have, n) {
				continue
			}
			if _, err := c.do(ctx, "POST", supersimProduct, supersimPath("NetworkAccessProfiles", req.ID, "Networks"), nil, url.Values{"Network": {n}}); err != nil {
				return resp{}, fmt.Errorf("add network %s to Super SIM network access profile %s: %w", n, req.ID, err)
			}
		}
		for _, n := range have {
			if slices.Contains(*want, n) {
				continue
			}
			if _, err := c.do(ctx, "DELETE", supersimProduct, supersimPath("NetworkAccessProfiles", req.ID, "Networks", n), nil, nil); err != nil && !isNotFound(err) {
				return resp{}, fmt.Errorf("remove network %s from Super SIM network access profile %s: %w", n, req.ID, err)
			}
		}
	}
	return resp{Output: st}, nil
}

// Delete forgets the profile: Twilio has no DELETE for a Network Access Profile.
func (r *SuperSimNetworkAccessProfile) Delete(_ context.Context, _ infer.DeleteRequest[SuperSimNetworkAccessProfileState]) (infer.DeleteResponse, error) {
	return infer.DeleteResponse{}, nil
}
