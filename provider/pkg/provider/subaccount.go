package provider

import (
	"context"
	"fmt"
	"net/url"
	"strings"
	"unicode/utf8"

	p "github.com/pulumi/pulumi-go-provider"
	"github.com/pulumi/pulumi-go-provider/infer"
)

// Subaccount manages a Twilio subaccount under the account that authenticates
// the provider. Twilio cannot delete a subaccount: destroying the resource
// closes it, which is permanent.
type Subaccount struct{}

// SubaccountArgs are the inputs of a Subaccount.
type SubaccountArgs struct {
	FriendlyName string  `pulumi:"friendlyName"`
	Status       *string `pulumi:"status,optional"`
}

func (a *SubaccountArgs) Annotate(an infer.Annotator) {
	an.Describe(&a.FriendlyName, "Human-readable name of the subaccount (up to 64 characters).")
	an.Describe(&a.Status, "Desired status: active (default) or suspended. A suspended subaccount stops all activity "+
		"but can be reactivated. Closing is only done by destroying the resource.")
	an.SetDefault(&a.Status, "active")
}

// SubaccountState is the persisted state of a Subaccount.
type SubaccountState struct {
	SubaccountArgs

	// Sid is the subaccount SID (AC...) and the resource ID.
	Sid string `pulumi:"sid"`
	// AuthToken is the subaccount's primary auth token.
	AuthToken string `pulumi:"authToken" provider:"secret"`
	// OwnerAccountSid is the parent account that owns the subaccount.
	OwnerAccountSid string `pulumi:"ownerAccountSid"`
	// DateCreated is the RFC 2822 creation time.
	DateCreated string `pulumi:"dateCreated"`
}

func (s *SubaccountState) Annotate(a infer.Annotator) {
	a.Describe(&s.Sid, "The subaccount SID (AC...).")
	a.Describe(&s.AuthToken, "The subaccount's primary auth token. Treat it as a secret.")
	a.Describe(&s.OwnerAccountSid, "SID of the parent account that owns this subaccount.")
	a.Describe(&s.DateCreated, "When the subaccount was created (RFC 2822).")
}

func (r *Subaccount) Annotate(a infer.Annotator) {
	a.Describe(&r, "A Twilio subaccount. Destroying the resource CLOSES the subaccount, which releases its phone numbers "+
		"and cannot be undone after 30 days; set status to suspended to pause one reversibly.")
}

var _ infer.CustomResource[SubaccountArgs, SubaccountState] = (*Subaccount)(nil)
var _ infer.CustomCheck[SubaccountArgs] = (*Subaccount)(nil)
var _ infer.CustomRead[SubaccountArgs, SubaccountState] = (*Subaccount)(nil)
var _ infer.CustomUpdate[SubaccountArgs, SubaccountState] = (*Subaccount)(nil)
var _ infer.CustomDelete[SubaccountState] = (*Subaccount)(nil)

func (r *Subaccount) Check(ctx context.Context, req infer.CheckRequest) (infer.CheckResponse[SubaccountArgs], error) {
	args, failures, err := infer.DefaultCheck[SubaccountArgs](ctx, req.NewInputs)
	if err != nil || len(failures) > 0 {
		return infer.CheckResponse[SubaccountArgs]{Inputs: args, Failures: failures}, err
	}
	if n := utf8.RuneCountInString(strings.TrimSpace(args.FriendlyName)); n < 1 || utf8.RuneCountInString(args.FriendlyName) > 64 {
		failures = append(failures, p.CheckFailure{Property: "friendlyName", Reason: "friendlyName must be 1-64 characters"})
	}
	if s := deref(args.Status); s != "active" && s != "suspended" {
		failures = append(failures, p.CheckFailure{Property: "status", Reason: "status must be active or suspended"})
	}
	return infer.CheckResponse[SubaccountArgs]{Inputs: args, Failures: failures}, nil
}

func accountPath(sid string) string { return "/2010-04-01/Accounts/" + url.PathEscape(sid) + ".json" }

func subaccountState(args SubaccountArgs, o obj) SubaccountState {
	st := o.str("status")
	if st != "" {
		args.Status = &st
	}
	return SubaccountState{SubaccountArgs: args, Sid: o.str("sid"), AuthToken: o.str("auth_token"),
		OwnerAccountSid: o.str("owner_account_sid"), DateCreated: o.str("date_created")}
}

func (r *Subaccount) Create(ctx context.Context, req infer.CreateRequest[SubaccountArgs]) (infer.CreateResponse[SubaccountState], error) {
	a := req.Inputs
	if req.DryRun {
		return infer.CreateResponse[SubaccountState]{ID: req.Name, Output: SubaccountState{SubaccountArgs: a}}, nil
	}
	c, err := clientFor(ctx)
	if err != nil {
		return infer.CreateResponse[SubaccountState]{}, err
	}
	o, err := c.do(ctx, "POST", "api", "/2010-04-01/Accounts.json", nil, url.Values{"FriendlyName": {a.FriendlyName}})
	if err != nil {
		return infer.CreateResponse[SubaccountState]{}, fmt.Errorf("create subaccount: %w", err)
	}
	sid := o.str("sid")
	if deref(a.Status) == "suspended" {
		o, err = c.do(ctx, "POST", "api", accountPath(sid), nil, url.Values{"Status": {"suspended"}})
		if err != nil {
			return infer.CreateResponse[SubaccountState]{ID: sid, Output: subaccountState(a, obj{"sid": sid})},
				fmt.Errorf("suspend new subaccount %s: %w", sid, err)
		}
	}
	return infer.CreateResponse[SubaccountState]{ID: sid, Output: subaccountState(a, o)}, nil
}

func (r *Subaccount) Read(ctx context.Context, req infer.ReadRequest[SubaccountArgs, SubaccountState]) (infer.ReadResponse[SubaccountArgs, SubaccountState], error) {
	c, err := clientFor(ctx)
	if err != nil {
		return infer.ReadResponse[SubaccountArgs, SubaccountState]{}, err
	}
	o, err := c.do(ctx, "GET", "api", accountPath(req.ID), nil, nil)
	if err != nil {
		if isNotFound(err) {
			return infer.ReadResponse[SubaccountArgs, SubaccountState]{}, nil
		}
		return infer.ReadResponse[SubaccountArgs, SubaccountState]{}, fmt.Errorf("read subaccount %s: %w", req.ID, err)
	}
	if o.str("status") == "closed" {
		return infer.ReadResponse[SubaccountArgs, SubaccountState]{}, nil
	}
	in := req.Inputs
	in.FriendlyName = o.str("friendly_name")
	status := o.str("status")
	in.Status = &status
	st := subaccountState(in, o)
	return infer.ReadResponse[SubaccountArgs, SubaccountState]{ID: req.ID, Inputs: in, State: st}, nil
}

func (r *Subaccount) Update(ctx context.Context, req infer.UpdateRequest[SubaccountArgs, SubaccountState]) (infer.UpdateResponse[SubaccountState], error) {
	a := req.Inputs
	if req.DryRun {
		st := req.State
		st.SubaccountArgs = a
		return infer.UpdateResponse[SubaccountState]{Output: st}, nil
	}
	c, err := clientFor(ctx)
	if err != nil {
		return infer.UpdateResponse[SubaccountState]{}, err
	}
	o, err := c.do(ctx, "POST", "api", accountPath(req.ID), nil,
		url.Values{"FriendlyName": {a.FriendlyName}, "Status": {deref(a.Status)}})
	if err != nil {
		return infer.UpdateResponse[SubaccountState]{}, fmt.Errorf("update subaccount %s: %w", req.ID, err)
	}
	return infer.UpdateResponse[SubaccountState]{Output: subaccountState(a, o)}, nil
}

func (r *Subaccount) Delete(ctx context.Context, req infer.DeleteRequest[SubaccountState]) (infer.DeleteResponse, error) {
	c, err := clientFor(ctx)
	if err != nil {
		return infer.DeleteResponse{}, err
	}
	_, err = c.do(ctx, "POST", "api", accountPath(req.ID), nil, url.Values{"Status": {"closed"}})
	if err != nil && !isNotFound(err) {
		return infer.DeleteResponse{}, fmt.Errorf("close subaccount %s: %w", req.ID, err)
	}
	return infer.DeleteResponse{}, nil
}
