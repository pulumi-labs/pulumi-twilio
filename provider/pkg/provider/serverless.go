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
	"strconv"
	"strings"

	p "github.com/pulumi/pulumi-go-provider"
	"github.com/pulumi/pulumi-go-provider/infer"
)

// The resources in this file and its siblings (serverless_variable.go, serverless_container.go) manage
// the configuration plane of Twilio Serverless (serverless.twilio.com v1): services, environments,
// environment variables, and the Function and Asset containers. The shapes follow twilio-oai
// spec/json/twilio_serverless_v1.json (spec version 1.0.0). Versions (the uploaded code), builds and
// deployments are not managed: see research/coverage/twilio.md.

var (
	serverlessServiceSidRE     = regexp.MustCompile(`^ZS[0-9a-fA-F]{32}$`)
	serverlessEnvironmentSidRE = regexp.MustCompile(`^ZE[0-9a-fA-F]{32}$`)
	serverlessVariableSidRE    = regexp.MustCompile(`^ZV[0-9a-fA-F]{32}$`)
	serverlessContainerSidRE   = regexp.MustCompile(`^ZH[0-9a-fA-F]{32}$`)
)

// serverlessPath joins already-valid path segments below /v1/Services, escaping each one.
func serverlessPath(segments ...string) string {
	path := "/v1/Services"
	for _, s := range segments {
		if s != "" {
			path += "/" + url.PathEscape(s)
		}
	}
	return path
}

// serverlessSvcPath is the path of a collection (sid empty) or instance below a service.
func serverlessSvcPath(service, coll, sid string) string {
	return serverlessPath(service, coll, sid)
}

func serverlessEnvPath(service, env, sid string) string {
	return serverlessPath(service, "Environments", env, "Variables", sid)
}

// boolForm sends a bool only when the program set it.
func boolForm(form url.Values, key string, b *bool) {
	if b != nil {
		form.Set(key, strconv.FormatBool(*b))
	}
}

// ------------------------------------------------------------ ServerlessService

// ServerlessService manages a Twilio Serverless service: the container for Functions, Assets, builds
// and environments.
type ServerlessService struct{}

// ServerlessServiceArgs are the inputs of a ServerlessService.
type ServerlessServiceArgs struct {
	UniqueName         string `pulumi:"uniqueName" provider:"replaceOnChanges"`
	FriendlyName       string `pulumi:"friendlyName"`
	IncludeCredentials *bool  `pulumi:"includeCredentials,optional"`
	UiEditable         *bool  `pulumi:"uiEditable,optional"`
}

func (a *ServerlessServiceArgs) Annotate(an infer.Annotator) {
	an.Describe(&a.UniqueName, "Unique name of the service, at most 50 characters. It can be used instead of the SID to address the service. Twilio cannot change it after creation, so changing it replaces the resource.")
	an.Describe(&a.FriendlyName, "Human-readable name of the service, at most 255 characters.")
	an.Describe(&a.IncludeCredentials, "Whether Twilio injects the account credentials into the Functions of the service. Left unset, the value Twilio already has is kept.")
	an.Describe(&a.UiEditable, "Whether the Functions and Assets of the service may be edited in the Twilio Console. Left unset, the value Twilio already has is kept.")
}

// ServerlessServiceState is the persisted state of a ServerlessService.
type ServerlessServiceState struct {
	ServerlessServiceArgs

	// Sid is the service SID (ZS...).
	Sid         string `pulumi:"sid"`
	AccountSid  string `pulumi:"accountSid"`
	DomainBase  string `pulumi:"domainBase"`
	Url         string `pulumi:"url"`
	DateCreated string `pulumi:"dateCreated"`
	DateUpdated string `pulumi:"dateUpdated"`
}

func (s *ServerlessServiceState) Annotate(a infer.Annotator) {
	a.Describe(&s.Sid, "The service SID (ZS...).")
	a.Describe(&s.AccountSid, "SID of the account that owns the service.")
	a.Describe(&s.DomainBase, "The base domain name of the service's environments, for example zoo-1234.")
	a.Describe(&s.Url, "The absolute URL of the service resource.")
	a.Describe(&s.DateCreated, "When the service was created (ISO 8601).")
	a.Describe(&s.DateUpdated, "When the service was last updated (ISO 8601).")
}

func (r *ServerlessService) Annotate(a infer.Annotator) {
	a.Describe(&r, "A Twilio Serverless service. Add ServerlessEnvironment, ServerlessFunction and ServerlessAsset resources to it. The resource ID is the service SID (ZS...). Deleting a service deletes everything inside it.")
}

var _ infer.CustomResource[ServerlessServiceArgs, ServerlessServiceState] = (*ServerlessService)(nil)
var _ infer.CustomCheck[ServerlessServiceArgs] = (*ServerlessService)(nil)
var _ infer.CustomDiff[ServerlessServiceArgs, ServerlessServiceState] = (*ServerlessService)(nil)
var _ infer.CustomRead[ServerlessServiceArgs, ServerlessServiceState] = (*ServerlessService)(nil)
var _ infer.CustomUpdate[ServerlessServiceArgs, ServerlessServiceState] = (*ServerlessService)(nil)
var _ infer.CustomDelete[ServerlessServiceState] = (*ServerlessService)(nil)

func (c *checker) maxLen(prop, v string, max int) {
	if !c.unk(prop) && len(v) > max {
		c.fail(prop, fmt.Sprintf("%s must be at most %d characters", prop, max))
	}
}

func (c *checker) required(prop, v string) {
	if !c.unk(prop) && v == "" {
		c.fail(prop, prop+" must not be empty")
	}
}

func (r *ServerlessService) Check(ctx context.Context, req infer.CheckRequest) (infer.CheckResponse[ServerlessServiceArgs], error) {
	args, failures, err := infer.DefaultCheck[ServerlessServiceArgs](ctx, req.NewInputs)
	if err != nil || len(failures) > 0 {
		return infer.CheckResponse[ServerlessServiceArgs]{Inputs: args, Failures: failures}, err
	}
	c := &checker{unk: unknowns(req.NewInputs)}
	c.required("uniqueName", args.UniqueName)
	c.maxLen("uniqueName", args.UniqueName, 50)
	c.required("friendlyName", args.FriendlyName)
	c.maxLen("friendlyName", args.FriendlyName, 255)
	return infer.CheckResponse[ServerlessServiceArgs]{Inputs: args, Failures: c.failures}, nil
}

// boolChanged reports a difference only when the program sets the bool: an unset bool keeps whatever
// Twilio holds, so an import followed by a program that omits it plans nothing.
func boolChanged(old, next *bool) bool {
	return next != nil && (old == nil || *old != *next)
}

func (r *ServerlessService) Diff(_ context.Context, req infer.DiffRequest[ServerlessServiceArgs, ServerlessServiceState]) (infer.DiffResponse, error) {
	o, n := req.State.ServerlessServiceArgs, req.Inputs
	detail := map[string]p.PropertyDiff{}
	if o.UniqueName != n.UniqueName {
		detail["uniqueName"] = p.PropertyDiff{Kind: p.UpdateReplace}
	}
	if o.FriendlyName != n.FriendlyName {
		detail["friendlyName"] = p.PropertyDiff{Kind: p.Update}
	}
	if boolChanged(o.IncludeCredentials, n.IncludeCredentials) {
		detail["includeCredentials"] = p.PropertyDiff{Kind: p.Update}
	}
	if boolChanged(o.UiEditable, n.UiEditable) {
		detail["uiEditable"] = p.PropertyDiff{Kind: p.Update}
	}
	return infer.DiffResponse{HasChanges: len(detail) > 0, DetailedDiff: detail}, nil
}

func serverlessServiceOutputs(st *ServerlessServiceState, o obj) {
	st.Sid, st.AccountSid, st.DomainBase, st.Url = o.str("sid"), o.str("account_sid"), o.str("domain_base"), o.str("url")
	st.DateCreated, st.DateUpdated = o.str("date_created"), o.str("date_updated")
}

func (r *ServerlessService) Create(ctx context.Context, req infer.CreateRequest[ServerlessServiceArgs]) (infer.CreateResponse[ServerlessServiceState], error) {
	a := req.Inputs
	if req.DryRun {
		return infer.CreateResponse[ServerlessServiceState]{Output: ServerlessServiceState{ServerlessServiceArgs: a}}, nil
	}
	c, err := clientFor(ctx)
	if err != nil {
		return infer.CreateResponse[ServerlessServiceState]{}, err
	}
	form := url.Values{"UniqueName": {a.UniqueName}, "FriendlyName": {a.FriendlyName}}
	boolForm(form, "IncludeCredentials", a.IncludeCredentials)
	boolForm(form, "UiEditable", a.UiEditable)
	o, err := c.do(ctx, "POST", "serverless", serverlessPath(), nil, form)
	if err != nil {
		return infer.CreateResponse[ServerlessServiceState]{}, fmt.Errorf("create serverless service %q: %w", a.UniqueName, err)
	}
	st := ServerlessServiceState{ServerlessServiceArgs: a}
	serverlessServiceOutputs(&st, o)
	return infer.CreateResponse[ServerlessServiceState]{ID: st.Sid, Output: st}, nil
}

func (r *ServerlessService) Read(ctx context.Context, req infer.ReadRequest[ServerlessServiceArgs, ServerlessServiceState]) (infer.ReadResponse[ServerlessServiceArgs, ServerlessServiceState], error) {
	type resp = infer.ReadResponse[ServerlessServiceArgs, ServerlessServiceState]
	c, err := clientFor(ctx)
	if err != nil {
		return resp{}, err
	}
	o, err := c.do(ctx, "GET", "serverless", serverlessPath(req.ID), nil, nil)
	if err != nil {
		if isNotFound(err) {
			if err := notFoundOnImport("serverless service", req.ID, req.Inputs, req.State); err != nil {
				return resp{}, err
			}
			return resp{}, nil
		}
		return resp{}, fmt.Errorf("read serverless service %s: %w", req.ID, err)
	}
	in := req.Inputs
	if in.UniqueName == "" {
		// An import adopts what Twilio reports, the two flags included.
		in = ServerlessServiceArgs{IncludeCredentials: o.optBool("include_credentials"), UiEditable: o.optBool("ui_editable")}
	} else {
		if in.IncludeCredentials != nil {
			in.IncludeCredentials = o.optBool("include_credentials")
		}
		if in.UiEditable != nil {
			in.UiEditable = o.optBool("ui_editable")
		}
	}
	if v := o.str("unique_name"); v != "" {
		in.UniqueName = v
	}
	if v := o.str("friendly_name"); v != "" {
		in.FriendlyName = v
	}
	st := ServerlessServiceState{ServerlessServiceArgs: in}
	serverlessServiceOutputs(&st, o)
	return resp{ID: req.ID, Inputs: in, State: st}, nil
}

func (r *ServerlessService) Update(ctx context.Context, req infer.UpdateRequest[ServerlessServiceArgs, ServerlessServiceState]) (infer.UpdateResponse[ServerlessServiceState], error) {
	a := req.Inputs
	st := req.State
	st.ServerlessServiceArgs = a
	if req.DryRun {
		return infer.UpdateResponse[ServerlessServiceState]{Output: st}, nil
	}
	c, err := clientFor(ctx)
	if err != nil {
		return infer.UpdateResponse[ServerlessServiceState]{}, err
	}
	form := url.Values{"FriendlyName": {a.FriendlyName}}
	boolForm(form, "IncludeCredentials", a.IncludeCredentials)
	boolForm(form, "UiEditable", a.UiEditable)
	o, err := c.do(ctx, "POST", "serverless", serverlessPath(req.ID), nil, form)
	if err != nil {
		return infer.UpdateResponse[ServerlessServiceState]{}, fmt.Errorf("update serverless service %s: %w", req.ID, err)
	}
	serverlessServiceOutputs(&st, o)
	return infer.UpdateResponse[ServerlessServiceState]{Output: st}, nil
}

func (r *ServerlessService) Delete(ctx context.Context, req infer.DeleteRequest[ServerlessServiceState]) (infer.DeleteResponse, error) {
	c, err := clientFor(ctx)
	if err != nil {
		return infer.DeleteResponse{}, err
	}
	if _, err := c.do(ctx, "DELETE", "serverless", serverlessPath(req.ID), nil, nil); err != nil && !isNotFound(err) {
		return infer.DeleteResponse{}, fmt.Errorf("delete serverless service %s: %w", req.ID, err)
	}
	return infer.DeleteResponse{}, nil
}

// ------------------------------------------------------- ServerlessEnvironment

// ServerlessEnvironment manages an environment of a Serverless service: a stage (dev, stage, prod)
// with its own domain, variables and deployed build.
type ServerlessEnvironment struct{}

// ServerlessEnvironmentArgs are the inputs of a ServerlessEnvironment. Twilio offers no update for an
// environment, so every input replaces it.
type ServerlessEnvironmentArgs struct {
	ServiceSid   string  `pulumi:"serviceSid" provider:"replaceOnChanges"`
	UniqueName   string  `pulumi:"uniqueName" provider:"replaceOnChanges"`
	DomainSuffix *string `pulumi:"domainSuffix,optional" provider:"replaceOnChanges"`
}

func (a *ServerlessEnvironmentArgs) Annotate(an infer.Annotator) {
	an.Describe(&a.ServiceSid, "SID (ZS...) of the Serverless service. Changing it replaces the resource.")
	an.Describe(&a.UniqueName, "Unique name of the environment, at most 100 characters, for example stage. Twilio cannot change it, so changing it replaces the resource.")
	an.Describe(&a.DomainSuffix, "URL-friendly suffix that becomes part of the environment's domain name, at most 16 characters. Twilio cannot change it, so changing it replaces the resource.")
}

// ServerlessEnvironmentState is the persisted state of a ServerlessEnvironment.
type ServerlessEnvironmentState struct {
	ServerlessEnvironmentArgs

	// Sid is the environment SID (ZE...).
	Sid         string  `pulumi:"sid"`
	AccountSid  string  `pulumi:"accountSid"`
	BuildSid    *string `pulumi:"buildSid,optional"`
	DomainName  string  `pulumi:"domainName"`
	Url         string  `pulumi:"url"`
	DateCreated string  `pulumi:"dateCreated"`
	DateUpdated string  `pulumi:"dateUpdated"`
}

func (s *ServerlessEnvironmentState) Annotate(a infer.Annotator) {
	a.Describe(&s.Sid, "The environment SID (ZE...).")
	a.Describe(&s.AccountSid, "SID of the account that owns the environment.")
	a.Describe(&s.BuildSid, "SID (ZB...) of the build currently deployed to the environment, when there is one.")
	a.Describe(&s.DomainName, "The domain name of the environment, for example zoo-1234-stage.twil.io.")
	a.Describe(&s.Url, "The absolute URL of the environment resource.")
	a.Describe(&s.DateCreated, "When the environment was created (ISO 8601).")
	a.Describe(&s.DateUpdated, "When the environment was last updated (ISO 8601).")
}

func (r *ServerlessEnvironment) Annotate(a infer.Annotator) {
	a.Describe(&r, "An environment of a Twilio Serverless service. Add ServerlessVariable resources to configure it. The resource ID is <serviceSid>/<sid>.")
}

var _ infer.CustomResource[ServerlessEnvironmentArgs, ServerlessEnvironmentState] = (*ServerlessEnvironment)(nil)
var _ infer.CustomCheck[ServerlessEnvironmentArgs] = (*ServerlessEnvironment)(nil)
var _ infer.CustomDiff[ServerlessEnvironmentArgs, ServerlessEnvironmentState] = (*ServerlessEnvironment)(nil)
var _ infer.CustomRead[ServerlessEnvironmentArgs, ServerlessEnvironmentState] = (*ServerlessEnvironment)(nil)
var _ infer.CustomDelete[ServerlessEnvironmentState] = (*ServerlessEnvironment)(nil)

func (r *ServerlessEnvironment) Check(ctx context.Context, req infer.CheckRequest) (infer.CheckResponse[ServerlessEnvironmentArgs], error) {
	args, failures, err := infer.DefaultCheck[ServerlessEnvironmentArgs](ctx, req.NewInputs)
	if err != nil || len(failures) > 0 {
		return infer.CheckResponse[ServerlessEnvironmentArgs]{Inputs: args, Failures: failures}, err
	}
	c := &checker{unk: unknowns(req.NewInputs)}
	c.sid("serviceSid", args.ServiceSid, serverlessServiceSidRE, "Serverless service (ZS followed by 32 hex characters)")
	c.required("uniqueName", args.UniqueName)
	c.maxLen("uniqueName", args.UniqueName, 100)
	c.maxLen("domainSuffix", derefStr(args.DomainSuffix), 16)
	return infer.CheckResponse[ServerlessEnvironmentArgs]{Inputs: args, Failures: c.failures}, nil
}

// Diff treats an unset domain suffix as an empty one. Twilio's response does not carry the suffix as a
// field, so an import derives it from the environment's domain name (see environmentDomainSuffix). When
// that is not possible the recorded suffix stays empty; a program suffix that the recorded domain name
// already ends with ("-<suffix>.twil.io") is then not a change, so an import never plans a replacement
// (and with it a replacement of the environment's variables) for the program that created the environment.
func (r *ServerlessEnvironment) Diff(_ context.Context, req infer.DiffRequest[ServerlessEnvironmentArgs, ServerlessEnvironmentState]) (infer.DiffResponse, error) {
	o, n := req.State.ServerlessEnvironmentArgs, req.Inputs
	detail := map[string]p.PropertyDiff{}
	if o.ServiceSid != n.ServiceSid {
		detail["serviceSid"] = p.PropertyDiff{Kind: p.UpdateReplace}
	}
	if o.UniqueName != n.UniqueName {
		detail["uniqueName"] = p.PropertyDiff{Kind: p.UpdateReplace}
	}
	if n.DomainSuffix != nil && derefStr(o.DomainSuffix) != *n.DomainSuffix &&
		!(o.DomainSuffix == nil && *n.DomainSuffix != "" && strings.HasSuffix(req.State.DomainName, "-"+*n.DomainSuffix+".twil.io")) {
		detail["domainSuffix"] = p.PropertyDiff{Kind: p.UpdateReplace}
	}
	return infer.DiffResponse{HasChanges: len(detail) > 0, DetailedDiff: detail}, nil
}

func serverlessEnvironmentOutputs(st *ServerlessEnvironmentState, o obj) {
	st.Sid, st.AccountSid, st.BuildSid = o.str("sid"), o.str("account_sid"), nonEmpty(o.optStr("build_sid"))
	st.DomainName, st.Url = o.str("domain_name"), o.str("url")
	st.DateCreated, st.DateUpdated = o.str("date_created"), o.str("date_updated")
}

func (r *ServerlessEnvironment) Create(ctx context.Context, req infer.CreateRequest[ServerlessEnvironmentArgs]) (infer.CreateResponse[ServerlessEnvironmentState], error) {
	a := req.Inputs
	if req.DryRun {
		return infer.CreateResponse[ServerlessEnvironmentState]{Output: ServerlessEnvironmentState{ServerlessEnvironmentArgs: a}}, nil
	}
	c, err := clientFor(ctx)
	if err != nil {
		return infer.CreateResponse[ServerlessEnvironmentState]{}, err
	}
	form := url.Values{"UniqueName": {a.UniqueName}}
	setS(form, "DomainSuffix", a.DomainSuffix)
	o, err := c.do(ctx, "POST", "serverless", serverlessSvcPath(a.ServiceSid, "Environments", ""), nil, form)
	if err != nil {
		return infer.CreateResponse[ServerlessEnvironmentState]{}, fmt.Errorf("create environment %q on serverless service %s: %w", a.UniqueName, a.ServiceSid, err)
	}
	st := ServerlessEnvironmentState{ServerlessEnvironmentArgs: a}
	serverlessEnvironmentOutputs(&st, o)
	return infer.CreateResponse[ServerlessEnvironmentState]{ID: a.ServiceSid + "/" + st.Sid, Output: st}, nil
}

func (r *ServerlessEnvironment) Read(ctx context.Context, req infer.ReadRequest[ServerlessEnvironmentArgs, ServerlessEnvironmentState]) (infer.ReadResponse[ServerlessEnvironmentArgs, ServerlessEnvironmentState], error) {
	type resp = infer.ReadResponse[ServerlessEnvironmentArgs, ServerlessEnvironmentState]
	ids, err := splitVerifyID(req.ID, 2, "<serviceSid>/<sid>")
	if err != nil {
		return resp{}, err
	}
	c, err := clientFor(ctx)
	if err != nil {
		return resp{}, err
	}
	o, err := c.do(ctx, "GET", "serverless", serverlessSvcPath(ids[0], "Environments", ids[1]), nil, nil)
	if err != nil {
		if isNotFound(err) {
			if err := notFoundOnImport("serverless environment", req.ID, req.Inputs, req.State); err != nil {
				return resp{}, err
			}
			return resp{}, nil
		}
		return resp{}, fmt.Errorf("read serverless environment %s: %w", req.ID, err)
	}
	in := req.Inputs
	if in.ServiceSid == "" {
		// Twilio does not return the suffix as a field: derive it from the domain name and the parent
		// service's domain base. Any failure leaves it empty, which Diff tolerates.
		in = ServerlessEnvironmentArgs{ServiceSid: ids[0]}
		in.DomainSuffix = environmentDomainSuffix(ctx, c, ids[0], o)
		if in.DomainSuffix != nil && *in.DomainSuffix == o.str("unique_name") {
			in.DomainSuffix = nil // Twilio's own default suffix is the unique name
		}
	}
	if v := o.str("unique_name"); v != "" {
		in.UniqueName = v
	}
	st := ServerlessEnvironmentState{ServerlessEnvironmentArgs: in}
	serverlessEnvironmentOutputs(&st, o)
	return resp{ID: req.ID, Inputs: in, State: st}, nil
}

// environmentDomainSuffix derives the domain suffix of an environment from its domain name, which is
// "<domainBase>-<suffix>.twil.io" with a suffix and "<domainBase>.twil.io" without, where domainBase
// belongs to the parent service. It returns nil when the suffix cannot be derived with confidence.
func environmentDomainSuffix(ctx context.Context, c *Client, service string, env obj) *string {
	svc, err := c.do(ctx, "GET", "serverless", serverlessSvcPath(service, "", ""), nil, nil)
	if err != nil {
		return nil
	}
	base := svc.str("domain_base")
	name := strings.TrimSuffix(env.str("domain_name"), ".twil.io")
	if base == "" || name == env.str("domain_name") || !strings.HasPrefix(name, base+"-") {
		return nil
	}
	if suffix := strings.TrimPrefix(name, base+"-"); suffix != "" {
		return &suffix
	}
	return nil
}

func (r *ServerlessEnvironment) Delete(ctx context.Context, req infer.DeleteRequest[ServerlessEnvironmentState]) (infer.DeleteResponse, error) {
	ids, err := splitVerifyID(req.ID, 2, "<serviceSid>/<sid>")
	if err != nil {
		return infer.DeleteResponse{}, err
	}
	c, err := clientFor(ctx)
	if err != nil {
		return infer.DeleteResponse{}, err
	}
	if _, err := c.do(ctx, "DELETE", "serverless", serverlessSvcPath(ids[0], "Environments", ids[1]), nil, nil); err != nil && !isNotFound(err) {
		return infer.DeleteResponse{}, fmt.Errorf("delete serverless environment %s: %w", req.ID, err)
	}
	return infer.DeleteResponse{}, nil
}
