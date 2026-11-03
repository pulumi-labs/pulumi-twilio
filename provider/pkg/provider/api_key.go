package provider

import (
	"context"
	"encoding/json"
	"fmt"
	"net/url"

	p "github.com/pulumi/pulumi-go-provider"
	"github.com/pulumi/pulumi-go-provider/infer"
)

// ApiKey manages a Twilio API key (standard or restricted).
type ApiKey struct{}

// ApiKeyArgs are the inputs of an ApiKey.
type ApiKeyArgs struct {
	FriendlyName string  `pulumi:"friendlyName"`
	AccountSid   *string `pulumi:"accountSid,optional" provider:"replaceOnChanges"`
	KeyType      *string `pulumi:"keyType,optional" provider:"replaceOnChanges"`
	Policy       *string `pulumi:"policy,optional"`
}

func (a *ApiKeyArgs) Annotate(an infer.Annotator) {
	an.Describe(&a.FriendlyName, "Human-readable name of the key (up to 64 characters).")
	an.Describe(&a.AccountSid, "SID of the account or subaccount the key belongs to. Defaults to the provider account. "+
		"Changing it replaces the resource.")
	an.Describe(&a.KeyType, "standard (default; full access to the account's REST API except account and key management) "+
		"or restricted (limited to the permissions in policy). Changing it replaces the resource.")
	an.Describe(&a.Policy, "JSON policy document for a restricted key, for example "+
		"{\"allow\":[\"/v1/Services/**\"]}. Required when keyType is restricted and not allowed otherwise. "+
		"Updated in place; the stored policy is not read back, so out-of-band policy edits are not detected.")
	an.SetDefault(&a.KeyType, "standard")
}

// ApiKeyState is the persisted state of an ApiKey.
type ApiKeyState struct {
	ApiKeyArgs

	// Sid is the key SID (SK...) and the resource ID.
	Sid string `pulumi:"sid"`
	// Secret is returned by Twilio exactly once, at creation.
	Secret string `pulumi:"secret" provider:"secret"`
	// DateCreated is the RFC 2822 creation time.
	DateCreated string `pulumi:"dateCreated"`
}

func (s *ApiKeyState) Annotate(a infer.Annotator) {
	a.Describe(&s.Sid, "The API key SID (SK...); use it as the username with the secret.")
	a.Describe(&s.Secret, "The API key secret. Twilio returns it only once, at creation, so it cannot be recovered "+
		"later or after import.")
	a.Describe(&s.DateCreated, "When the key was created (RFC 2822).")
}

func (r *ApiKey) Annotate(a infer.Annotator) {
	a.Describe(&r, "A Twilio API key. The secret is available only in the creation response and is stored as a "+
		"Pulumi secret. Creating and deleting keys requires the account auth token (an API key cannot manage keys).")
}

var _ infer.CustomResource[ApiKeyArgs, ApiKeyState] = (*ApiKey)(nil)
var _ infer.CustomCheck[ApiKeyArgs] = (*ApiKey)(nil)
var _ infer.CustomRead[ApiKeyArgs, ApiKeyState] = (*ApiKey)(nil)
var _ infer.CustomUpdate[ApiKeyArgs, ApiKeyState] = (*ApiKey)(nil)
var _ infer.CustomDelete[ApiKeyState] = (*ApiKey)(nil)

func (r *ApiKey) Check(ctx context.Context, req infer.CheckRequest) (infer.CheckResponse[ApiKeyArgs], error) {
	args, failures, err := infer.DefaultCheck[ApiKeyArgs](ctx, req.NewInputs)
	if err != nil || len(failures) > 0 {
		return infer.CheckResponse[ApiKeyArgs]{Inputs: args, Failures: failures}, err
	}
	switch deref(args.KeyType) {
	case "standard":
		if args.Policy != nil {
			failures = append(failures, p.CheckFailure{Property: "policy", Reason: "policy is only valid when keyType is restricted"})
		}
	case "restricted":
		if args.Policy == nil {
			failures = append(failures, p.CheckFailure{Property: "policy", Reason: "policy is required when keyType is restricted"})
		} else if !json.Valid([]byte(*args.Policy)) {
			failures = append(failures, p.CheckFailure{Property: "policy", Reason: "policy must be valid JSON"})
		}
	default:
		failures = append(failures, p.CheckFailure{Property: "keyType", Reason: "keyType must be standard or restricted"})
	}
	if args.FriendlyName == "" || len(args.FriendlyName) > 64 {
		failures = append(failures, p.CheckFailure{Property: "friendlyName", Reason: "friendlyName must be 1-64 characters"})
	}
	return infer.CheckResponse[ApiKeyArgs]{Inputs: args, Failures: failures}, nil
}

func keyPath(sid string) string { return "/v1/Keys/" + url.PathEscape(sid) }

func (r *ApiKey) Create(ctx context.Context, req infer.CreateRequest[ApiKeyArgs]) (infer.CreateResponse[ApiKeyState], error) {
	a := req.Inputs
	if req.DryRun {
		return infer.CreateResponse[ApiKeyState]{ID: req.Name, Output: ApiKeyState{ApiKeyArgs: a}}, nil
	}
	c, err := clientFor(ctx)
	if err != nil {
		return infer.CreateResponse[ApiKeyState]{}, err
	}
	form := url.Values{"AccountSid": {c.acct(a.AccountSid)}, "FriendlyName": {a.FriendlyName}, "KeyType": {deref(a.KeyType)}}
	setS(form, "Policy", a.Policy)
	o, err := c.do(ctx, "POST", "iam", "/v1/Keys", nil, form)
	if err != nil {
		return infer.CreateResponse[ApiKeyState]{}, fmt.Errorf("create API key: %w", err)
	}
	return infer.CreateResponse[ApiKeyState]{ID: o.str("sid"), Output: ApiKeyState{
		ApiKeyArgs: a, Sid: o.str("sid"), Secret: o.str("secret"), DateCreated: o.str("date_created"),
	}}, nil
}

func (r *ApiKey) Read(ctx context.Context, req infer.ReadRequest[ApiKeyArgs, ApiKeyState]) (infer.ReadResponse[ApiKeyArgs, ApiKeyState], error) {
	c, err := clientFor(ctx)
	if err != nil {
		return infer.ReadResponse[ApiKeyArgs, ApiKeyState]{}, err
	}
	o, err := c.do(ctx, "GET", "iam", keyPath(req.ID), nil, nil)
	if err != nil {
		if isNotFound(err) {
			return infer.ReadResponse[ApiKeyArgs, ApiKeyState]{}, nil
		}
		return infer.ReadResponse[ApiKeyArgs, ApiKeyState]{}, fmt.Errorf("read API key %s: %w", req.ID, err)
	}
	in := req.Inputs
	st := req.State
	in.FriendlyName = o.str("friendly_name")
	if req.Inputs.FriendlyName == "" { // import: infer the key type from the presence of a policy
		if pol, ok := o["policy"].(map[string]any); ok && len(pol) > 0 {
			b, _ := json.Marshal(pol)
			s, kt := string(b), "restricted"
			in.Policy, in.KeyType = &s, &kt
		} else {
			kt := "standard"
			in.KeyType = &kt
		}
	}
	st.ApiKeyArgs = in
	st.Sid = o.str("sid")
	st.DateCreated = o.str("date_created")
	return infer.ReadResponse[ApiKeyArgs, ApiKeyState]{ID: req.ID, Inputs: in, State: st}, nil
}

func (r *ApiKey) Update(ctx context.Context, req infer.UpdateRequest[ApiKeyArgs, ApiKeyState]) (infer.UpdateResponse[ApiKeyState], error) {
	a := req.Inputs
	st := req.State
	st.ApiKeyArgs = a
	if req.DryRun {
		return infer.UpdateResponse[ApiKeyState]{Output: st}, nil
	}
	c, err := clientFor(ctx)
	if err != nil {
		return infer.UpdateResponse[ApiKeyState]{}, err
	}
	form := url.Values{"FriendlyName": {a.FriendlyName}}
	setS(form, "Policy", a.Policy)
	if _, err := c.do(ctx, "POST", "iam", keyPath(req.ID), nil, form); err != nil {
		return infer.UpdateResponse[ApiKeyState]{}, fmt.Errorf("update API key %s: %w", req.ID, err)
	}
	return infer.UpdateResponse[ApiKeyState]{Output: st}, nil
}

func (r *ApiKey) Delete(ctx context.Context, req infer.DeleteRequest[ApiKeyState]) (infer.DeleteResponse, error) {
	c, err := clientFor(ctx)
	if err != nil {
		return infer.DeleteResponse{}, err
	}
	if _, err := c.do(ctx, "DELETE", "iam", keyPath(req.ID), nil, nil); err != nil && !isNotFound(err) {
		return infer.DeleteResponse{}, fmt.Errorf("delete API key %s: %w", req.ID, err)
	}
	return infer.DeleteResponse{}, nil
}
