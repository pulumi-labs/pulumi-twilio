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

// ConversationsServiceNotificationConfiguration manages the push notification settings of a
// Conversations service. A service has exactly one, so this adopts it; destroying the resource
// forgets it and leaves the settings as they are.
type ConversationsServiceNotificationConfiguration struct{}

// ConversationsNotification is the setting of one push notification event.
type ConversationsNotification struct {
	Enabled  *bool   `pulumi:"enabled,optional"`
	Template *string `pulumi:"template,optional"`
	Sound    *string `pulumi:"sound,optional"`
}

func (a *ConversationsNotification) Annotate(an infer.Annotator) {
	an.Describe(&a.Enabled, "Whether the notification is sent (Twilio default false).")
	an.Describe(&a.Template, "Template of the notification text.")
	an.Describe(&a.Sound, "Name of the sound played with the notification.")
}

// ConversationsMediaNotification is the setting of the notification for a message with media.
type ConversationsMediaNotification struct {
	Enabled  *bool   `pulumi:"enabled,optional"`
	Template *string `pulumi:"template,optional"`
}

func (a *ConversationsMediaNotification) Annotate(an infer.Annotator) {
	an.Describe(&a.Enabled, "Whether a notification is sent for a new message with media or file attachments (Twilio default false).")
	an.Describe(&a.Template, "Template of the notification text for a message with media.")
}

// ConversationsNewMessageNotification is the setting of the new message notification.
type ConversationsNewMessageNotification struct {
	Enabled           *bool                           `pulumi:"enabled,optional"`
	Template          *string                         `pulumi:"template,optional"`
	Sound             *string                         `pulumi:"sound,optional"`
	BadgeCountEnabled *bool                           `pulumi:"badgeCountEnabled,optional"`
	WithMedia         *ConversationsMediaNotification `pulumi:"withMedia,optional"`
}

func (a *ConversationsNewMessageNotification) Annotate(an infer.Annotator) {
	an.Describe(&a.Enabled, "Whether a notification is sent when a message is added to a conversation (Twilio default false).")
	an.Describe(&a.Template, "Template of the notification text.")
	an.Describe(&a.Sound, "Name of the sound played with the notification.")
	an.Describe(&a.BadgeCountEnabled, "Whether the badge count is enabled (Twilio default false).")
	an.Describe(&a.WithMedia, "The notification for a message with media or file attachments.")
}

// ConversationsServiceNotificationConfigurationArgs are the inputs of a ConversationsServiceNotificationConfiguration.
type ConversationsServiceNotificationConfigurationArgs struct {
	ChatServiceSid          string                               `pulumi:"chatServiceSid" provider:"replaceOnChanges"`
	LogEnabled              *bool                                `pulumi:"logEnabled,optional"`
	NewMessage              *ConversationsNewMessageNotification `pulumi:"newMessage,optional"`
	AddedToConversation     *ConversationsNotification           `pulumi:"addedToConversation,optional"`
	RemovedFromConversation *ConversationsNotification           `pulumi:"removedFromConversation,optional"`
}

func (a *ConversationsServiceNotificationConfigurationArgs) Annotate(an infer.Annotator) {
	an.Describe(&a.ChatServiceSid, "SID (IS...) of the Conversations service. Changing it replaces the resource.")
	an.Describe(&a.LogEnabled, "Whether notification logging is enabled. Removing it leaves the setting unchanged.")
	an.Describe(&a.NewMessage, "The notification sent when a message is added to a conversation. Fields that are removed are left unchanged.")
	an.Describe(&a.AddedToConversation, "The notification sent when a participant is added to a conversation. Fields that are removed are left unchanged.")
	an.Describe(&a.RemovedFromConversation, "The notification sent to a user removed from a conversation. Fields that are removed are left unchanged.")
}

// ConversationsServiceNotificationConfigurationState is the persisted state of a ConversationsServiceNotificationConfiguration.
type ConversationsServiceNotificationConfigurationState struct {
	ConversationsServiceNotificationConfigurationArgs
}

func (r *ConversationsServiceNotificationConfiguration) Annotate(a infer.Annotator) {
	a.Describe(&r, "The push notification settings of a Conversations service. The resource ID is the service SID. Destroying it forgets it and leaves the settings on Twilio as they are.")
}

var _ infer.CustomResource[ConversationsServiceNotificationConfigurationArgs, ConversationsServiceNotificationConfigurationState] = (*ConversationsServiceNotificationConfiguration)(nil)
var _ infer.CustomCheck[ConversationsServiceNotificationConfigurationArgs] = (*ConversationsServiceNotificationConfiguration)(nil)
var _ infer.CustomDiff[ConversationsServiceNotificationConfigurationArgs, ConversationsServiceNotificationConfigurationState] = (*ConversationsServiceNotificationConfiguration)(nil)
var _ infer.CustomRead[ConversationsServiceNotificationConfigurationArgs, ConversationsServiceNotificationConfigurationState] = (*ConversationsServiceNotificationConfiguration)(nil)
var _ infer.CustomUpdate[ConversationsServiceNotificationConfigurationArgs, ConversationsServiceNotificationConfigurationState] = (*ConversationsServiceNotificationConfiguration)(nil)
var _ infer.CustomDelete[ConversationsServiceNotificationConfigurationState] = (*ConversationsServiceNotificationConfiguration)(nil)

func (r *ConversationsServiceNotificationConfiguration) Check(ctx context.Context, req infer.CheckRequest) (infer.CheckResponse[ConversationsServiceNotificationConfigurationArgs], error) {
	args, failures, err := infer.DefaultCheck[ConversationsServiceNotificationConfigurationArgs](ctx, req.NewInputs)
	if err != nil || len(failures) > 0 {
		return infer.CheckResponse[ConversationsServiceNotificationConfigurationArgs]{Inputs: args, Failures: failures}, err
	}
	c := &checker{unk: unknowns(req.NewInputs)}
	c.sid("chatServiceSid", args.ChatServiceSid, cvServiceSidRE, "Conversations service (IS followed by 32 hex characters)")
	return infer.CheckResponse[ConversationsServiceNotificationConfigurationArgs]{Inputs: args, Failures: c.failures}, nil
}

// An import leaves a nested field unset when Twilio holds its default (false, empty), so an unset
// old value is that default and a program that writes it is not a change (#1888).
func sameFlag(o, n *bool) bool   { return n == nil || (o != nil && *o == *n) || (o == nil && !*n) }
func sameText(o, n *string) bool { return n == nil || (o != nil && *o == *n) || (o == nil && *n == "") }

// sameNotification reports whether every field the program sets in n is already set so in o.
func sameNotification(o, n *ConversationsNotification) bool {
	if n == nil {
		return true
	}
	if o == nil {
		o = &ConversationsNotification{}
	}
	return sameFlag(o.Enabled, n.Enabled) && sameText(o.Template, n.Template) && sameText(o.Sound, n.Sound)
}

func sameNewMessage(o, n *ConversationsNewMessageNotification) bool {
	if n == nil {
		return true
	}
	if o == nil {
		o = &ConversationsNewMessageNotification{}
	}
	if !sameNotification(&ConversationsNotification{o.Enabled, o.Template, o.Sound}, &ConversationsNotification{n.Enabled, n.Template, n.Sound}) || !sameFlag(o.BadgeCountEnabled, n.BadgeCountEnabled) {
		return false
	}
	if n.WithMedia == nil {
		return true
	}
	om := o.WithMedia
	if om == nil {
		om = &ConversationsMediaNotification{}
	}
	return sameFlag(om.Enabled, n.WithMedia.Enabled) && sameText(om.Template, n.WithMedia.Template)
}

func (r *ConversationsServiceNotificationConfiguration) Diff(_ context.Context, req infer.DiffRequest[ConversationsServiceNotificationConfigurationArgs, ConversationsServiceNotificationConfigurationState]) (infer.DiffResponse, error) {
	o, n := req.State.ConversationsServiceNotificationConfigurationArgs, req.Inputs
	d := trDiffSet{}
	d.replace("chatServiceSid", o.ChatServiceSid, n.ChatServiceSid)
	d.boolDef(conversationsNotificationConfigurationDefaults, "logEnabled", o.LogEnabled, n.LogEnabled)
	if !sameNewMessage(o.NewMessage, n.NewMessage) {
		d["newMessage"] = p.PropertyDiff{Kind: p.Update}
	}
	if !sameNotification(o.AddedToConversation, n.AddedToConversation) {
		d["addedToConversation"] = p.PropertyDiff{Kind: p.Update}
	}
	if !sameNotification(o.RemovedFromConversation, n.RemovedFromConversation) {
		d["removedFromConversation"] = p.PropertyDiff{Kind: p.Update}
	}
	return d.response(), nil
}

func (e *ConversationsNotification) set(form url.Values, prefix string) {
	if e == nil {
		return
	}
	setB(form, prefix+".Enabled", e.Enabled)
	setS(form, prefix+".Template", e.Template)
	setS(form, prefix+".Sound", e.Sound)
}

func (a ConversationsServiceNotificationConfigurationArgs) form() url.Values {
	form := url.Values{}
	setB(form, "LogEnabled", a.LogEnabled)
	if nm := a.NewMessage; nm != nil {
		(&ConversationsNotification{nm.Enabled, nm.Template, nm.Sound}).set(form, "NewMessage")
		setB(form, "NewMessage.BadgeCountEnabled", nm.BadgeCountEnabled)
		if wm := nm.WithMedia; wm != nil {
			setB(form, "NewMessage.WithMedia.Enabled", wm.Enabled)
			setS(form, "NewMessage.WithMedia.Template", wm.Template)
		}
	}
	a.AddedToConversation.set(form, "AddedToConversation")
	a.RemovedFromConversation.set(form, "RemovedFromConversation")
	return form
}

// apply writes the settings the program states; with none it only checks that the service exists.
func (a ConversationsServiceNotificationConfigurationArgs) apply(ctx context.Context, verb string) error {
	c, err := clientFor(ctx)
	if err != nil {
		return err
	}
	form, method := a.form(), "POST"
	if len(form) == 0 {
		method, form = "GET", nil
	}
	if _, err := c.do(ctx, method, cvProduct, cvPath(a.ChatServiceSid, "Configuration", "Notifications"), nil, form); err != nil {
		return fmt.Errorf("%s Conversations notification configuration %s: %w", verb, a.ChatServiceSid, err)
	}
	return nil
}

func (r *ConversationsServiceNotificationConfiguration) Create(ctx context.Context, req infer.CreateRequest[ConversationsServiceNotificationConfigurationArgs]) (infer.CreateResponse[ConversationsServiceNotificationConfigurationState], error) {
	type resp = infer.CreateResponse[ConversationsServiceNotificationConfigurationState]
	a := req.Inputs
	st := ConversationsServiceNotificationConfigurationState{ConversationsServiceNotificationConfigurationArgs: a}
	if req.DryRun {
		return resp{Output: st}, nil
	}
	if err := a.apply(ctx, "configure"); err != nil {
		return resp{}, err
	}
	return resp{ID: a.ChatServiceSid, Output: st}, nil
}

// cvNotification reads one notification back: the fields the program states follow Twilio, the
// others stay unset. On import it adopts the fields that are not at their default.
func cvNotification(in *ConversationsNotification, o obj, prefix string, imported bool) *ConversationsNotification {
	if imported {
		in = &ConversationsNotification{}
	}
	if in == nil {
		return nil
	}
	out := ConversationsNotification{
		Enabled:  trBool(in.Enabled, o.optBool(prefix+".enabled"), conversationsNotificationConfigurationDefaults.boolean("notification.enabled"), imported),
		Template: trStr(in.Template, o.optStr(prefix+".template"), conversationsNotificationConfigurationDefaults.str("notification.template"), imported),
		Sound:    trStr(in.Sound, o.optStr(prefix+".sound"), conversationsNotificationConfigurationDefaults.str("notification.sound"), imported),
	}
	if imported && out == (ConversationsNotification{}) {
		return nil
	}
	return &out
}

func cvNewMessage(in *ConversationsNewMessageNotification, o obj, imported bool) *ConversationsNewMessageNotification {
	if imported {
		in = &ConversationsNewMessageNotification{}
	}
	if in == nil {
		return nil
	}
	base := cvNotification(&ConversationsNotification{in.Enabled, in.Template, in.Sound}, o, "new_message", imported)
	out := ConversationsNewMessageNotification{BadgeCountEnabled: trBool(in.BadgeCountEnabled, o.optBool("new_message.badge_count_enabled"), conversationsNotificationConfigurationDefaults.boolean("newMessage.badgeCountEnabled"), imported)}
	if base != nil {
		out.Enabled, out.Template, out.Sound = base.Enabled, base.Template, base.Sound
	}
	if imported || in.WithMedia != nil {
		wm := in.WithMedia
		if wm == nil {
			wm = &ConversationsMediaNotification{}
		}
		m := ConversationsMediaNotification{
			Enabled:  trBool(wm.Enabled, o.optBool("new_message.with_media.enabled"), conversationsNotificationConfigurationDefaults.boolean("newMessage.withMedia.enabled"), imported),
			Template: trStr(wm.Template, o.optStr("new_message.with_media.template"), conversationsNotificationConfigurationDefaults.str("newMessage.withMedia.template"), imported),
		}
		if !imported || m != (ConversationsMediaNotification{}) {
			out.WithMedia = &m
		}
	}
	if imported && out.Enabled == nil && out.Template == nil && out.Sound == nil && out.BadgeCountEnabled == nil && out.WithMedia == nil {
		return nil
	}
	return &out
}

func (r *ConversationsServiceNotificationConfiguration) Read(ctx context.Context, req infer.ReadRequest[ConversationsServiceNotificationConfigurationArgs, ConversationsServiceNotificationConfigurationState]) (infer.ReadResponse[ConversationsServiceNotificationConfigurationArgs, ConversationsServiceNotificationConfigurationState], error) {
	type resp = infer.ReadResponse[ConversationsServiceNotificationConfigurationArgs, ConversationsServiceNotificationConfigurationState]
	c, err := clientFor(ctx)
	if err != nil {
		return resp{}, err
	}
	o, err := c.do(ctx, "GET", cvProduct, cvPath(req.ID, "Configuration", "Notifications"), nil, nil)
	if err != nil {
		if isNotFound(err) {
			if err := notFoundOnImport("Conversations notification configuration", req.ID, req.Inputs, req.State); err != nil {
				return resp{}, err
			}
			return resp{}, nil
		}
		return resp{}, fmt.Errorf("read Conversations notification configuration %s: %w", req.ID, err)
	}
	in := req.Inputs
	imported := in.ChatServiceSid == ""
	in.ChatServiceSid = req.ID
	in.LogEnabled = trBool(in.LogEnabled, o.optBool("log_enabled"), conversationsNotificationConfigurationDefaults.boolean("logEnabled"), imported)
	in.NewMessage = cvNewMessage(in.NewMessage, o, imported)
	in.AddedToConversation = cvNotification(in.AddedToConversation, o, "added_to_conversation", imported)
	in.RemovedFromConversation = cvNotification(in.RemovedFromConversation, o, "removed_from_conversation", imported)
	return resp{ID: req.ID, Inputs: in, State: ConversationsServiceNotificationConfigurationState{ConversationsServiceNotificationConfigurationArgs: in}}, nil
}

func (r *ConversationsServiceNotificationConfiguration) Update(ctx context.Context, req infer.UpdateRequest[ConversationsServiceNotificationConfigurationArgs, ConversationsServiceNotificationConfigurationState]) (infer.UpdateResponse[ConversationsServiceNotificationConfigurationState], error) {
	st := ConversationsServiceNotificationConfigurationState{ConversationsServiceNotificationConfigurationArgs: req.Inputs}
	if req.DryRun {
		return infer.UpdateResponse[ConversationsServiceNotificationConfigurationState]{Output: st}, nil
	}
	if err := req.Inputs.apply(ctx, "update"); err != nil {
		return infer.UpdateResponse[ConversationsServiceNotificationConfigurationState]{}, err
	}
	return infer.UpdateResponse[ConversationsServiceNotificationConfigurationState]{Output: st}, nil
}

// Delete forgets the settings: a service always has them and the spec offers no way to remove them.
func (r *ConversationsServiceNotificationConfiguration) Delete(_ context.Context, _ infer.DeleteRequest[ConversationsServiceNotificationConfigurationState]) (infer.DeleteResponse, error) {
	return infer.DeleteResponse{}, nil
}

// conversationsNotificationConfigurationDefaults are the Twilio defaults of this resource, shared by Read (import adoption) and Diff.
var conversationsNotificationConfigurationDefaults = trDefaults{
	"logEnabled":                    false,
	"notification.enabled":          false,
	"notification.template":         "",
	"notification.sound":            "",
	"newMessage.badgeCountEnabled":  false,
	"newMessage.withMedia.enabled":  false,
	"newMessage.withMedia.template": "",
}
