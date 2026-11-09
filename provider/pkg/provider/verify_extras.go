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
	"sort"
	"strconv"
	"strings"

	p "github.com/pulumi/pulumi-go-provider"
	"github.com/pulumi/pulumi-go-provider/infer"
)

// The resources in this file extend a Verify service (verify.twilio.com v2): rate limits and their
// buckets, per-country messaging configurations, webhooks, and the SafeList. The shapes follow
// twilio-oai spec/json/twilio_verify_v2.json.

const (
	verifyWebhookDefaultStatus  = "enabled"
	verifyWebhookDefaultVersion = "v2"
)

var (
	verifyServiceSidRE   = regexp.MustCompile(`^VA[0-9a-fA-F]{32}$`)
	verifyRateLimitSidRE = regexp.MustCompile(`^RK[0-9a-fA-F]{32}$`)
	verifyMessagingSidRE = regexp.MustCompile(`^MG[0-9a-fA-F]{32}$`)
	e164RE               = regexp.MustCompile(`^\+[1-9][0-9]{1,14}$`)
)

// verifyChildPath is the path of a collection (sid empty) or instance below a Verify service, for
// example RateLimits. coll may itself be nested (RateLimits/RK.../Buckets) and must already be escaped.
func verifyChildPath(service, coll, sid string) string {
	path := verifyPath(service) + "/" + coll
	if sid != "" {
		path += "/" + url.PathEscape(sid)
	}
	return path
}

func verifyBucketPath(service, rateLimit, sid string) string {
	return verifyChildPath(service, "RateLimits/"+url.PathEscape(rateLimit)+"/Buckets", sid)
}

// safeListPath is the path of the SafeList collection (number empty) or of one number. The plus sign
// of an E.164 number is escaped, as Twilio documents it.
func safeListPath(number string) string {
	if number == "" {
		return "/v2/SafeList/Numbers"
	}
	return "/v2/SafeList/Numbers/" + strings.ReplaceAll(url.PathEscape(number), "+", "%2B")
}

// splitVerifyID splits an ID of n slash-separated parts, none empty.
func splitVerifyID(id string, n int, shape string) ([]string, error) {
	parts := strings.Split(id, "/")
	if len(parts) != n {
		return nil, fmt.Errorf("invalid ID %q: expected %s", id, shape)
	}
	for _, s := range parts {
		if s == "" {
			return nil, fmt.Errorf("invalid ID %q: expected %s", id, shape)
		}
	}
	return parts, nil
}

func derefStr(s *string) string {
	if s == nil {
		return ""
	}
	return *s
}

// strOrDefault treats an unset value as Twilio's default.
func strOrDefault(s *string, def string) string {
	if s == nil || *s == "" {
		return def
	}
	return *s
}

// sameStrings reports whether a and b hold the same strings, ignoring order and duplicates.
func sameStrings(a, b []string) bool {
	norm := func(l []string) []string {
		seen := map[string]bool{}
		var out []string
		for _, s := range l {
			if !seen[s] {
				seen[s] = true
				out = append(out, s)
			}
		}
		sort.Strings(out)
		return out
	}
	x, y := norm(a), norm(b)
	if len(x) != len(y) {
		return false
	}
	for i := range x {
		if x[i] != y[i] {
			return false
		}
	}
	return true
}

type checker struct {
	unk      func(keys ...string) bool
	failures []p.CheckFailure
}

func (c *checker) fail(prop, reason string) {
	c.failures = append(c.failures, p.CheckFailure{Property: prop, Reason: reason})
}

func (c *checker) sid(prop, v string, re *regexp.Regexp, what string) {
	if !c.unk(prop) && !re.MatchString(v) {
		c.fail(prop, prop+" must be the SID of a "+what)
	}
}

// ------------------------------------------------------------ VerifyRateLimit

// VerifyRateLimit manages a rate limit of a Verify service: a named bucket-holder that caps how often
// one value (a phone number, an IP address) may trigger a verification.
type VerifyRateLimit struct{}

// VerifyRateLimitArgs are the inputs of a VerifyRateLimit.
type VerifyRateLimitArgs struct {
	ServiceSid  string  `pulumi:"serviceSid" provider:"replaceOnChanges"`
	UniqueName  string  `pulumi:"uniqueName" provider:"replaceOnChanges"`
	Description *string `pulumi:"description,optional"`
}

func (a *VerifyRateLimitArgs) Annotate(an infer.Annotator) {
	an.Describe(&a.ServiceSid, "SID (VA...) of the Verify service. Changing it replaces the resource.")
	an.Describe(&a.UniqueName, "Unique, developer-assigned name of the rate limit, used by verification requests to refer to it. It cannot be changed after creation, so changing it replaces the resource. It must not contain PII.")
	an.Describe(&a.Description, "Description of the rate limit. Removing it clears the description.")
}

// VerifyRateLimitState is the persisted state of a VerifyRateLimit.
type VerifyRateLimitState struct {
	VerifyRateLimitArgs

	// Sid is the rate limit SID (RK...).
	Sid         string `pulumi:"sid"`
	DateCreated string `pulumi:"dateCreated"`
	DateUpdated string `pulumi:"dateUpdated"`
}

func (s *VerifyRateLimitState) Annotate(a infer.Annotator) {
	a.Describe(&s.Sid, "The rate limit SID (RK...).")
	a.Describe(&s.DateCreated, "When the rate limit was created (ISO 8601).")
	a.Describe(&s.DateUpdated, "When the rate limit was last updated (ISO 8601).")
}

func (r *VerifyRateLimit) Annotate(a infer.Annotator) {
	a.Describe(&r, "A rate limit of a Verify service. Add VerifyRateLimitBucket resources to say how many requests are allowed per interval. The resource ID is <serviceSid>/<sid>.")
}

var _ infer.CustomResource[VerifyRateLimitArgs, VerifyRateLimitState] = (*VerifyRateLimit)(nil)
var _ infer.CustomCheck[VerifyRateLimitArgs] = (*VerifyRateLimit)(nil)
var _ infer.CustomDiff[VerifyRateLimitArgs, VerifyRateLimitState] = (*VerifyRateLimit)(nil)
var _ infer.CustomRead[VerifyRateLimitArgs, VerifyRateLimitState] = (*VerifyRateLimit)(nil)
var _ infer.CustomUpdate[VerifyRateLimitArgs, VerifyRateLimitState] = (*VerifyRateLimit)(nil)
var _ infer.CustomDelete[VerifyRateLimitState] = (*VerifyRateLimit)(nil)

func (r *VerifyRateLimit) Check(ctx context.Context, req infer.CheckRequest) (infer.CheckResponse[VerifyRateLimitArgs], error) {
	args, failures, err := infer.DefaultCheck[VerifyRateLimitArgs](ctx, req.NewInputs)
	if err != nil || len(failures) > 0 {
		return infer.CheckResponse[VerifyRateLimitArgs]{Inputs: args, Failures: failures}, err
	}
	c := &checker{unk: unknowns(req.NewInputs)}
	c.sid("serviceSid", args.ServiceSid, verifyServiceSidRE, "Verify service (VA followed by 32 hex characters)")
	if !c.unk("uniqueName") && args.UniqueName == "" {
		c.fail("uniqueName", "uniqueName must not be empty")
	}
	return infer.CheckResponse[VerifyRateLimitArgs]{Inputs: args, Failures: c.failures}, nil
}

// Diff treats an unset description as an empty one, so an import (which adopts only a description
// Twilio reports) and a program that restates an empty description agree.
func (r *VerifyRateLimit) Diff(_ context.Context, req infer.DiffRequest[VerifyRateLimitArgs, VerifyRateLimitState]) (infer.DiffResponse, error) {
	o, n := req.State.VerifyRateLimitArgs, req.Inputs
	detail := map[string]p.PropertyDiff{}
	if o.ServiceSid != n.ServiceSid {
		detail["serviceSid"] = p.PropertyDiff{Kind: p.UpdateReplace}
	}
	if o.UniqueName != n.UniqueName {
		detail["uniqueName"] = p.PropertyDiff{Kind: p.UpdateReplace}
	}
	if derefStr(o.Description) != derefStr(n.Description) {
		detail["description"] = p.PropertyDiff{Kind: p.Update}
	}
	return infer.DiffResponse{HasChanges: len(detail) > 0, DetailedDiff: detail}, nil
}

func verifyRateLimitOutputs(st *VerifyRateLimitState, o obj) {
	st.Sid, st.DateCreated, st.DateUpdated = o.str("sid"), o.str("date_created"), o.str("date_updated")
}

func (r *VerifyRateLimit) Create(ctx context.Context, req infer.CreateRequest[VerifyRateLimitArgs]) (infer.CreateResponse[VerifyRateLimitState], error) {
	a := req.Inputs
	if req.DryRun {
		return infer.CreateResponse[VerifyRateLimitState]{Output: VerifyRateLimitState{VerifyRateLimitArgs: a}}, nil
	}
	c, err := clientFor(ctx)
	if err != nil {
		return infer.CreateResponse[VerifyRateLimitState]{}, err
	}
	form := url.Values{"UniqueName": {a.UniqueName}}
	setS(form, "Description", a.Description)
	o, err := c.do(ctx, "POST", "verify", verifyChildPath(a.ServiceSid, "RateLimits", ""), nil, form)
	if err != nil {
		return infer.CreateResponse[VerifyRateLimitState]{}, fmt.Errorf("create rate limit on Verify service %s: %w", a.ServiceSid, err)
	}
	st := VerifyRateLimitState{VerifyRateLimitArgs: a}
	verifyRateLimitOutputs(&st, o)
	return infer.CreateResponse[VerifyRateLimitState]{ID: a.ServiceSid + "/" + st.Sid, Output: st}, nil
}

func (r *VerifyRateLimit) Read(ctx context.Context, req infer.ReadRequest[VerifyRateLimitArgs, VerifyRateLimitState]) (infer.ReadResponse[VerifyRateLimitArgs, VerifyRateLimitState], error) {
	type resp = infer.ReadResponse[VerifyRateLimitArgs, VerifyRateLimitState]
	ids, err := splitVerifyID(req.ID, 2, "<serviceSid>/<sid>")
	if err != nil {
		return resp{}, err
	}
	c, err := clientFor(ctx)
	if err != nil {
		return resp{}, err
	}
	o, err := c.do(ctx, "GET", "verify", verifyChildPath(ids[0], "RateLimits", ids[1]), nil, nil)
	if err != nil {
		if isNotFound(err) {
			if err := notFoundOnImport("verify rate limit", req.ID, req.Inputs, req.State); err != nil {
				return resp{}, err
			}
			return resp{}, nil
		}
		return resp{}, fmt.Errorf("read rate limit %s: %w", req.ID, err)
	}
	in := req.Inputs
	if in.ServiceSid == "" {
		// An import: adopt a description only when Twilio reports one.
		in = VerifyRateLimitArgs{ServiceSid: ids[0], UniqueName: o.str("unique_name"), Description: nonEmpty(o.optStr("description"))}
	} else {
		if n := o.str("unique_name"); n != "" {
			in.UniqueName = n
		}
		in.Description = nonEmpty(o.optStr("description"))
	}
	st := VerifyRateLimitState{VerifyRateLimitArgs: in}
	verifyRateLimitOutputs(&st, o)
	return resp{ID: req.ID, Inputs: in, State: st}, nil
}

func (r *VerifyRateLimit) Update(ctx context.Context, req infer.UpdateRequest[VerifyRateLimitArgs, VerifyRateLimitState]) (infer.UpdateResponse[VerifyRateLimitState], error) {
	a := req.Inputs
	st := req.State
	st.VerifyRateLimitArgs = a
	if req.DryRun {
		return infer.UpdateResponse[VerifyRateLimitState]{Output: st}, nil
	}
	ids, err := splitVerifyID(req.ID, 2, "<serviceSid>/<sid>")
	if err != nil {
		return infer.UpdateResponse[VerifyRateLimitState]{}, err
	}
	c, err := clientFor(ctx)
	if err != nil {
		return infer.UpdateResponse[VerifyRateLimitState]{}, err
	}
	// Description is the only updatable field; sending it always makes removal clear it.
	form := url.Values{"Description": {derefStr(a.Description)}}
	o, err := c.do(ctx, "POST", "verify", verifyChildPath(ids[0], "RateLimits", ids[1]), nil, form)
	if err != nil {
		return infer.UpdateResponse[VerifyRateLimitState]{}, fmt.Errorf("update rate limit %s: %w", req.ID, err)
	}
	verifyRateLimitOutputs(&st, o)
	return infer.UpdateResponse[VerifyRateLimitState]{Output: st}, nil
}

func (r *VerifyRateLimit) Delete(ctx context.Context, req infer.DeleteRequest[VerifyRateLimitState]) (infer.DeleteResponse, error) {
	ids, err := splitVerifyID(req.ID, 2, "<serviceSid>/<sid>")
	if err != nil {
		return infer.DeleteResponse{}, err
	}
	c, err := clientFor(ctx)
	if err != nil {
		return infer.DeleteResponse{}, err
	}
	if _, err := c.do(ctx, "DELETE", "verify", verifyChildPath(ids[0], "RateLimits", ids[1]), nil, nil); err != nil && !isNotFound(err) {
		return infer.DeleteResponse{}, fmt.Errorf("delete rate limit %s: %w", req.ID, err)
	}
	return infer.DeleteResponse{}, nil
}

// ------------------------------------------------------- VerifyRateLimitBucket

// VerifyRateLimitBucket manages a bucket of a rate limit: at most max requests per interval seconds.
type VerifyRateLimitBucket struct{}

// VerifyRateLimitBucketArgs are the inputs of a VerifyRateLimitBucket.
type VerifyRateLimitBucketArgs struct {
	ServiceSid   string `pulumi:"serviceSid" provider:"replaceOnChanges"`
	RateLimitSid string `pulumi:"rateLimitSid" provider:"replaceOnChanges"`
	Max          int    `pulumi:"max"`
	Interval     int    `pulumi:"interval"`
}

func (a *VerifyRateLimitBucketArgs) Annotate(an infer.Annotator) {
	an.Describe(&a.ServiceSid, "SID (VA...) of the Verify service. Changing it replaces the resource.")
	an.Describe(&a.RateLimitSid, "SID (RK...) of the rate limit (see VerifyRateLimit). Changing it replaces the resource.")
	an.Describe(&a.Max, "Maximum number of requests permitted during the interval.")
	an.Describe(&a.Interval, "Number of seconds the rate limit is enforced over.")
}

// VerifyRateLimitBucketState is the persisted state of a VerifyRateLimitBucket.
type VerifyRateLimitBucketState struct {
	VerifyRateLimitBucketArgs

	// Sid is the bucket SID (BL...).
	Sid         string `pulumi:"sid"`
	DateCreated string `pulumi:"dateCreated"`
	DateUpdated string `pulumi:"dateUpdated"`
}

func (s *VerifyRateLimitBucketState) Annotate(a infer.Annotator) {
	a.Describe(&s.Sid, "The bucket SID (BL...).")
	a.Describe(&s.DateCreated, "When the bucket was created (ISO 8601).")
	a.Describe(&s.DateUpdated, "When the bucket was last updated (ISO 8601).")
}

func (r *VerifyRateLimitBucket) Annotate(a infer.Annotator) {
	a.Describe(&r, "A bucket of a Verify rate limit: at most max requests per interval seconds. The resource ID is <serviceSid>/<rateLimitSid>/<sid>.")
}

var _ infer.CustomResource[VerifyRateLimitBucketArgs, VerifyRateLimitBucketState] = (*VerifyRateLimitBucket)(nil)
var _ infer.CustomCheck[VerifyRateLimitBucketArgs] = (*VerifyRateLimitBucket)(nil)
var _ infer.CustomRead[VerifyRateLimitBucketArgs, VerifyRateLimitBucketState] = (*VerifyRateLimitBucket)(nil)
var _ infer.CustomUpdate[VerifyRateLimitBucketArgs, VerifyRateLimitBucketState] = (*VerifyRateLimitBucket)(nil)
var _ infer.CustomDelete[VerifyRateLimitBucketState] = (*VerifyRateLimitBucket)(nil)

func (r *VerifyRateLimitBucket) Check(ctx context.Context, req infer.CheckRequest) (infer.CheckResponse[VerifyRateLimitBucketArgs], error) {
	args, failures, err := infer.DefaultCheck[VerifyRateLimitBucketArgs](ctx, req.NewInputs)
	if err != nil || len(failures) > 0 {
		return infer.CheckResponse[VerifyRateLimitBucketArgs]{Inputs: args, Failures: failures}, err
	}
	c := &checker{unk: unknowns(req.NewInputs)}
	c.sid("serviceSid", args.ServiceSid, verifyServiceSidRE, "Verify service (VA followed by 32 hex characters)")
	c.sid("rateLimitSid", args.RateLimitSid, verifyRateLimitSidRE, "rate limit (RK followed by 32 hex characters)")
	// The spec states no upper bound, but a bucket that allows no requests, or over no time, cannot work.
	if !c.unk("max") && args.Max <= 0 {
		c.fail("max", "max must be greater than 0")
	}
	if !c.unk("interval") && args.Interval <= 0 {
		c.fail("interval", "interval must be greater than 0 (seconds)")
	}
	return infer.CheckResponse[VerifyRateLimitBucketArgs]{Inputs: args, Failures: c.failures}, nil
}

func (a *VerifyRateLimitBucketArgs) form() url.Values {
	return url.Values{"Max": {strconv.Itoa(a.Max)}, "Interval": {strconv.Itoa(a.Interval)}}
}

func verifyBucketOutputs(st *VerifyRateLimitBucketState, o obj) {
	st.Sid, st.DateCreated, st.DateUpdated = o.str("sid"), o.str("date_created"), o.str("date_updated")
}

func (r *VerifyRateLimitBucket) Create(ctx context.Context, req infer.CreateRequest[VerifyRateLimitBucketArgs]) (infer.CreateResponse[VerifyRateLimitBucketState], error) {
	a := req.Inputs
	if req.DryRun {
		return infer.CreateResponse[VerifyRateLimitBucketState]{Output: VerifyRateLimitBucketState{VerifyRateLimitBucketArgs: a}}, nil
	}
	c, err := clientFor(ctx)
	if err != nil {
		return infer.CreateResponse[VerifyRateLimitBucketState]{}, err
	}
	o, err := c.do(ctx, "POST", "verify", verifyBucketPath(a.ServiceSid, a.RateLimitSid, ""), nil, a.form())
	if err != nil {
		return infer.CreateResponse[VerifyRateLimitBucketState]{}, fmt.Errorf("create bucket on rate limit %s: %w", a.RateLimitSid, err)
	}
	st := VerifyRateLimitBucketState{VerifyRateLimitBucketArgs: a}
	verifyBucketOutputs(&st, o)
	return infer.CreateResponse[VerifyRateLimitBucketState]{ID: a.ServiceSid + "/" + a.RateLimitSid + "/" + st.Sid, Output: st}, nil
}

func (r *VerifyRateLimitBucket) Read(ctx context.Context, req infer.ReadRequest[VerifyRateLimitBucketArgs, VerifyRateLimitBucketState]) (infer.ReadResponse[VerifyRateLimitBucketArgs, VerifyRateLimitBucketState], error) {
	type resp = infer.ReadResponse[VerifyRateLimitBucketArgs, VerifyRateLimitBucketState]
	ids, err := splitVerifyID(req.ID, 3, "<serviceSid>/<rateLimitSid>/<sid>")
	if err != nil {
		return resp{}, err
	}
	c, err := clientFor(ctx)
	if err != nil {
		return resp{}, err
	}
	o, err := c.do(ctx, "GET", "verify", verifyBucketPath(ids[0], ids[1], ids[2]), nil, nil)
	if err != nil {
		if isNotFound(err) {
			if err := notFoundOnImport("verify rate limit bucket", req.ID, req.Inputs, req.State); err != nil {
				return resp{}, err
			}
			return resp{}, nil
		}
		return resp{}, fmt.Errorf("read bucket %s: %w", req.ID, err)
	}
	in := req.Inputs
	if in.ServiceSid == "" {
		in = VerifyRateLimitBucketArgs{ServiceSid: ids[0], RateLimitSid: ids[1]}
	}
	if v := o.optInt("max"); v != nil {
		in.Max = *v
	}
	if v := o.optInt("interval"); v != nil {
		in.Interval = *v
	}
	st := VerifyRateLimitBucketState{VerifyRateLimitBucketArgs: in}
	verifyBucketOutputs(&st, o)
	return resp{ID: req.ID, Inputs: in, State: st}, nil
}

func (r *VerifyRateLimitBucket) Update(ctx context.Context, req infer.UpdateRequest[VerifyRateLimitBucketArgs, VerifyRateLimitBucketState]) (infer.UpdateResponse[VerifyRateLimitBucketState], error) {
	a := req.Inputs
	st := req.State
	st.VerifyRateLimitBucketArgs = a
	if req.DryRun {
		return infer.UpdateResponse[VerifyRateLimitBucketState]{Output: st}, nil
	}
	ids, err := splitVerifyID(req.ID, 3, "<serviceSid>/<rateLimitSid>/<sid>")
	if err != nil {
		return infer.UpdateResponse[VerifyRateLimitBucketState]{}, err
	}
	c, err := clientFor(ctx)
	if err != nil {
		return infer.UpdateResponse[VerifyRateLimitBucketState]{}, err
	}
	o, err := c.do(ctx, "POST", "verify", verifyBucketPath(ids[0], ids[1], ids[2]), nil, a.form())
	if err != nil {
		return infer.UpdateResponse[VerifyRateLimitBucketState]{}, fmt.Errorf("update bucket %s: %w", req.ID, err)
	}
	verifyBucketOutputs(&st, o)
	return infer.UpdateResponse[VerifyRateLimitBucketState]{Output: st}, nil
}

func (r *VerifyRateLimitBucket) Delete(ctx context.Context, req infer.DeleteRequest[VerifyRateLimitBucketState]) (infer.DeleteResponse, error) {
	ids, err := splitVerifyID(req.ID, 3, "<serviceSid>/<rateLimitSid>/<sid>")
	if err != nil {
		return infer.DeleteResponse{}, err
	}
	c, err := clientFor(ctx)
	if err != nil {
		return infer.DeleteResponse{}, err
	}
	if _, err := c.do(ctx, "DELETE", "verify", verifyBucketPath(ids[0], ids[1], ids[2]), nil, nil); err != nil && !isNotFound(err) {
		return infer.DeleteResponse{}, fmt.Errorf("delete bucket %s: %w", req.ID, err)
	}
	return infer.DeleteResponse{}, nil
}
