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

	p "github.com/pulumi/pulumi-go-provider"
	"github.com/pulumi/pulumi-go-provider/infer"
)

// Queue manages a Twilio voice call queue (the target of the TwiML Enqueue verb).
type Queue struct{}

// QueueArgs are the inputs of a Queue.
type QueueArgs struct {
	FriendlyName string  `pulumi:"friendlyName"`
	AccountSid   *string `pulumi:"accountSid,optional" provider:"replaceOnChanges"`
	MaxSize      *int    `pulumi:"maxSize,optional"`
}

func (a *QueueArgs) props() props {
	return props{ints: []intProp{{"MaxSize", "max_size", &a.MaxSize}}}
}

func (a *QueueArgs) Annotate(an infer.Annotator) {
	an.Describe(&a.FriendlyName, "Human-readable name of the queue (up to 64 characters).")
	an.Describe(&a.AccountSid, "SID of the account or subaccount that owns the queue. Defaults to the provider account. Changing it replaces the resource.")
	an.Describe(&a.MaxSize, "Maximum number of calls allowed in the queue, 1 to 5000 (Twilio default 1000). Removing the input resets it to 1000.")
}

// QueueState is the persisted state of a Queue.
type QueueState struct {
	QueueArgs

	// Sid is the queue SID (QU...) and the resource ID.
	Sid string `pulumi:"sid"`
	// OwnerAccountSid is the account that owns the queue.
	OwnerAccountSid string `pulumi:"ownerAccountSid"`
	// CurrentSize is the number of calls in the queue right now.
	CurrentSize int `pulumi:"currentSize"`
	// AverageWaitTime is the average wait of the queue's calls, in seconds.
	AverageWaitTime int `pulumi:"averageWaitTime"`
	// DateCreated is the RFC 2822 creation time.
	DateCreated string `pulumi:"dateCreated"`
	// DateUpdated is the RFC 2822 time of the last update.
	DateUpdated string `pulumi:"dateUpdated"`
}

func (s *QueueState) Annotate(a infer.Annotator) {
	a.Describe(&s.Sid, "The queue SID (QU...).")
	a.Describe(&s.OwnerAccountSid, "SID of the account that owns the queue.")
	a.Describe(&s.CurrentSize, "Number of calls in the queue at the last refresh.")
	a.Describe(&s.AverageWaitTime, "Average wait time of the calls in the queue, in seconds, at the last refresh.")
	a.Describe(&s.DateCreated, "When the queue was created (RFC 2822).")
	a.Describe(&s.DateUpdated, "When the queue was last updated (RFC 2822).")
}

func (r *Queue) Annotate(a infer.Annotator) {
	a.Describe(&r, "A Twilio call queue. Calls waiting in a queue are dequeued by TwiML or the Calls API.")
}

var _ infer.CustomResource[QueueArgs, QueueState] = (*Queue)(nil)
var _ infer.CustomCheck[QueueArgs] = (*Queue)(nil)
var _ infer.CustomRead[QueueArgs, QueueState] = (*Queue)(nil)
var _ infer.CustomUpdate[QueueArgs, QueueState] = (*Queue)(nil)
var _ infer.CustomDelete[QueueState] = (*Queue)(nil)

func (r *Queue) Check(ctx context.Context, req infer.CheckRequest) (infer.CheckResponse[QueueArgs], error) {
	args, failures, err := infer.DefaultCheck[QueueArgs](ctx, req.NewInputs)
	if err != nil || len(failures) > 0 {
		return infer.CheckResponse[QueueArgs]{Inputs: args, Failures: failures}, err
	}
	unk := unknowns(req.NewInputs)
	if !unk("friendlyName") && (args.FriendlyName == "" || len(args.FriendlyName) > 64) {
		failures = append(failures, p.CheckFailure{Property: "friendlyName", Reason: "friendlyName must be 1-64 characters"})
	}
	if args.MaxSize != nil && !unk("maxSize") && (*args.MaxSize < 1 || *args.MaxSize > 5000) {
		failures = append(failures, p.CheckFailure{Property: "maxSize", Reason: "maxSize must be between 1 and 5000"})
	}
	return infer.CheckResponse[QueueArgs]{Inputs: args, Failures: failures}, nil
}

func queueOutputs(st *QueueState, o obj) {
	st.CurrentSize, st.AverageWaitTime = intOf(o.optInt("current_size")), intOf(o.optInt("average_wait_time"))
	st.DateCreated, st.DateUpdated = o.str("date_created"), o.str("date_updated")
}

func (r *Queue) Create(ctx context.Context, req infer.CreateRequest[QueueArgs]) (infer.CreateResponse[QueueState], error) {
	a := req.Inputs
	if req.DryRun {
		return infer.CreateResponse[QueueState]{ID: "", Output: QueueState{QueueArgs: a}}, nil
	}
	c, err := clientFor(ctx)
	if err != nil {
		return infer.CreateResponse[QueueState]{}, err
	}
	acct := c.acct(a.AccountSid)
	form := url.Values{"FriendlyName": {a.FriendlyName}}
	a.props().encode(form, nil)
	o, err := c.do(ctx, "POST", "api", acctPath(acct, "Queues", ""), nil, form)
	if err != nil {
		return infer.CreateResponse[QueueState]{}, fmt.Errorf("create queue: %w", err)
	}
	sid := o.str("sid")
	st := QueueState{QueueArgs: a, Sid: sid, OwnerAccountSid: acct}
	queueOutputs(&st, o)
	return infer.CreateResponse[QueueState]{ID: scopedID(c, acct, sid), Output: st}, nil
}

func (r *Queue) Read(ctx context.Context, req infer.ReadRequest[QueueArgs, QueueState]) (infer.ReadResponse[QueueArgs, QueueState], error) {
	c, err := clientFor(ctx)
	if err != nil {
		return infer.ReadResponse[QueueArgs, QueueState]{}, err
	}
	acct, sid := resolveScope(c, req.ID, req.State.OwnerAccountSid)
	o, err := c.do(ctx, "GET", "api", acctPath(acct, "Queues", sid), nil, nil)
	if err != nil {
		if isNotFound(err) {
			return infer.ReadResponse[QueueArgs, QueueState]{}, nil
		}
		return infer.ReadResponse[QueueArgs, QueueState]{}, fmt.Errorf("read queue %s: %w", req.ID, err)
	}
	in := req.Inputs
	imported := in.FriendlyName == ""
	in.FriendlyName = o.str("friendly_name")
	if imported {
		if acct != c.accountSid {
			in.AccountSid = &acct
		}
		in.props().adopt(o)
	} else {
		in.props().refreshSet(o)
	}
	st := QueueState{QueueArgs: in, Sid: sid, OwnerAccountSid: acct}
	queueOutputs(&st, o)
	return infer.ReadResponse[QueueArgs, QueueState]{ID: req.ID, Inputs: in, State: st}, nil
}

func (r *Queue) Update(ctx context.Context, req infer.UpdateRequest[QueueArgs, QueueState]) (infer.UpdateResponse[QueueState], error) {
	a := req.Inputs
	st := req.State
	st.QueueArgs = a
	if req.DryRun {
		return infer.UpdateResponse[QueueState]{Output: st}, nil
	}
	c, err := clientFor(ctx)
	if err != nil {
		return infer.UpdateResponse[QueueState]{}, err
	}
	acct, sid := resolveScope(c, req.ID, req.State.OwnerAccountSid)
	form := url.Values{"FriendlyName": {a.FriendlyName}}
	prev := req.State.QueueArgs.props()
	a.props().encode(form, &prev)
	o, err := c.do(ctx, "POST", "api", acctPath(acct, "Queues", sid), nil, form)
	if err != nil {
		return infer.UpdateResponse[QueueState]{}, fmt.Errorf("update queue %s: %w", req.ID, err)
	}
	queueOutputs(&st, o)
	return infer.UpdateResponse[QueueState]{Output: st}, nil
}

func (r *Queue) Delete(ctx context.Context, req infer.DeleteRequest[QueueState]) (infer.DeleteResponse, error) {
	c, err := clientFor(ctx)
	if err != nil {
		return infer.DeleteResponse{}, err
	}
	acct, sid := resolveScope(c, req.ID, req.State.OwnerAccountSid)
	if _, err := c.do(ctx, "DELETE", "api", acctPath(acct, "Queues", sid), nil, nil); err != nil && !isNotFound(err) {
		return infer.DeleteResponse{}, fmt.Errorf("delete queue %s: %w", req.ID, err)
	}
	return infer.DeleteResponse{}, nil
}
