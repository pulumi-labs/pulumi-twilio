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

// MessagingServicePhoneNumber attaches an owned phone number to a Messaging
// Service's sender pool. Every input identifies the attachment, so any change
// replaces it; deleting detaches the number without releasing it.
type MessagingServicePhoneNumber struct{}

// MessagingServicePhoneNumberArgs are the inputs of a MessagingServicePhoneNumber.
type MessagingServicePhoneNumberArgs struct {
	ServiceSid     string `pulumi:"serviceSid" provider:"replaceOnChanges"`
	PhoneNumberSid string `pulumi:"phoneNumberSid" provider:"replaceOnChanges"`
}

func (a *MessagingServicePhoneNumberArgs) Annotate(an infer.Annotator) {
	an.Describe(&a.ServiceSid, "SID (MG...) of the Messaging Service. Changing it replaces the resource.")
	an.Describe(&a.PhoneNumberSid, "SID (PN...) of the owned phone number to add as a sender. A number can belong to "+
		"only one Messaging Service. Changing it replaces the resource.")
}

// MessagingServicePhoneNumberState is the persisted state.
type MessagingServicePhoneNumberState struct {
	MessagingServicePhoneNumberArgs

	// PhoneNumber is the sender in E.164 format.
	PhoneNumber string `pulumi:"phoneNumber"`
	// Capabilities lists what the sender supports (for example SMS, MMS, Voice).
	Capabilities []string `pulumi:"capabilities"`
}

func (s *MessagingServicePhoneNumberState) Annotate(a infer.Annotator) {
	a.Describe(&s.PhoneNumber, "The sender phone number in E.164 format.")
	a.Describe(&s.Capabilities, "Capabilities of the sender, for example SMS, MMS and Voice.")
}

func (r *MessagingServicePhoneNumber) Annotate(a infer.Annotator) {
	a.Describe(&r, "Membership of a phone number in a Messaging Service sender pool. The resource ID is <serviceSid>/<phoneNumberSid>.")
}

var _ infer.CustomResource[MessagingServicePhoneNumberArgs, MessagingServicePhoneNumberState] = (*MessagingServicePhoneNumber)(nil)
var _ infer.CustomRead[MessagingServicePhoneNumberArgs, MessagingServicePhoneNumberState] = (*MessagingServicePhoneNumber)(nil)
var _ infer.CustomDelete[MessagingServicePhoneNumberState] = (*MessagingServicePhoneNumber)(nil)

func senderPath(svc, pn string) string {
	return serviceBase(svc) + "/PhoneNumbers/" + url.PathEscape(pn)
}

func capsOf(o obj) []string {
	out := []string{}
	if l, ok := o["capabilities"].([]any); ok {
		for _, v := range l {
			if s, ok := v.(string); ok {
				out = append(out, s)
			}
		}
	}
	return out
}

func (r *MessagingServicePhoneNumber) Create(ctx context.Context, req infer.CreateRequest[MessagingServicePhoneNumberArgs]) (infer.CreateResponse[MessagingServicePhoneNumberState], error) {
	a := req.Inputs
	id := a.ServiceSid + "/" + a.PhoneNumberSid
	if req.DryRun {
		return infer.CreateResponse[MessagingServicePhoneNumberState]{ID: id, Output: MessagingServicePhoneNumberState{MessagingServicePhoneNumberArgs: a, Capabilities: []string{}}}, nil
	}
	c, err := clientFor(ctx)
	if err != nil {
		return infer.CreateResponse[MessagingServicePhoneNumberState]{}, err
	}
	o, err := c.do(ctx, "POST", "messaging", serviceBase(a.ServiceSid)+"/PhoneNumbers", nil, url.Values{"PhoneNumberSid": {a.PhoneNumberSid}})
	if err != nil {
		return infer.CreateResponse[MessagingServicePhoneNumberState]{}, fmt.Errorf("add %s to messaging service %s: %w", a.PhoneNumberSid, a.ServiceSid, err)
	}
	return infer.CreateResponse[MessagingServicePhoneNumberState]{ID: id, Output: MessagingServicePhoneNumberState{
		MessagingServicePhoneNumberArgs: a, PhoneNumber: o.str("phone_number"), Capabilities: capsOf(o)}}, nil
}

func splitSenderID(id string) (string, string) {
	for i := 0; i < len(id); i++ {
		if id[i] == '/' {
			return id[:i], id[i+1:]
		}
	}
	return "", id
}

func (r *MessagingServicePhoneNumber) Read(ctx context.Context, req infer.ReadRequest[MessagingServicePhoneNumberArgs, MessagingServicePhoneNumberState]) (infer.ReadResponse[MessagingServicePhoneNumberArgs, MessagingServicePhoneNumberState], error) {
	svc, pn := splitSenderID(req.ID)
	if svc == "" {
		return infer.ReadResponse[MessagingServicePhoneNumberArgs, MessagingServicePhoneNumberState]{},
			fmt.Errorf("invalid ID %q: expected <serviceSid>/<phoneNumberSid>", req.ID)
	}
	c, err := clientFor(ctx)
	if err != nil {
		return infer.ReadResponse[MessagingServicePhoneNumberArgs, MessagingServicePhoneNumberState]{}, err
	}
	o, err := c.do(ctx, "GET", "messaging", senderPath(svc, pn), nil, nil)
	if err != nil {
		if isNotFound(err) {
			return infer.ReadResponse[MessagingServicePhoneNumberArgs, MessagingServicePhoneNumberState]{}, nil
		}
		return infer.ReadResponse[MessagingServicePhoneNumberArgs, MessagingServicePhoneNumberState]{}, fmt.Errorf("read messaging service sender %s: %w", req.ID, err)
	}
	in := MessagingServicePhoneNumberArgs{ServiceSid: svc, PhoneNumberSid: pn}
	return infer.ReadResponse[MessagingServicePhoneNumberArgs, MessagingServicePhoneNumberState]{ID: req.ID, Inputs: in,
		State: MessagingServicePhoneNumberState{MessagingServicePhoneNumberArgs: in, PhoneNumber: o.str("phone_number"), Capabilities: capsOf(o)}}, nil
}

func (r *MessagingServicePhoneNumber) Delete(ctx context.Context, req infer.DeleteRequest[MessagingServicePhoneNumberState]) (infer.DeleteResponse, error) {
	c, err := clientFor(ctx)
	if err != nil {
		return infer.DeleteResponse{}, err
	}
	s := req.State
	if _, err := c.do(ctx, "DELETE", "messaging", senderPath(s.ServiceSid, s.PhoneNumberSid), nil, nil); err != nil && !isNotFound(err) {
		return infer.DeleteResponse{}, fmt.Errorf("remove %s from messaging service %s: %w", s.PhoneNumberSid, s.ServiceSid, err)
	}
	return infer.DeleteResponse{}, nil
}
