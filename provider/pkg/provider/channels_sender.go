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

	"github.com/pulumi/pulumi-go-provider/infer"
)

// The Messaging v2 Channels Senders API (/v2/Channels/Senders on messaging.twilio.com) manages
// the WhatsApp and RCS senders of an account. It takes and returns JSON bodies. Update and delete
// are documented as WhatsApp only, so the resource is limited to WhatsApp senders (senderId
// whatsapp:+E.164); the lookups read both channels. This is not the messaging v1 sender pool
// (MessagingServiceChannelSender, CH... SIDs).
//
// The shapes follow twilio-oai spec/json/twilio_messaging_v2.json (version 1.0.0) read
// 2026-11-14 and have not been checked against a live account.

const channelsSenderProduct = "messaging"

var (
	channelsSenderSidRe      = regexp.MustCompile(`^XE[0-9a-fA-F]{32}$`)
	channelsSenderWhatsappRe = regexp.MustCompile(`^whatsapp:\+[0-9]{4,15}$`)
	channelsSenderMethods    = []string{"POST", "PUT"}
	// channelsSenderRetryNames are the pending display name states in which the name did not take
	// effect and re-submitting it is the documented retry.
	channelsSenderFailedNames = []string{"DECLINED", "EXPIRED", "PIN_MISMATCH", "REGISTRATION_FAILED"}
)

func channelsSenderPath(sid string) string {
	path := "/v2/Channels/Senders"
	if sid != "" {
		path += "/" + url.PathEscape(sid)
	}
	return path
}

// ------------------------------------------------------------------------ nested types

// ChannelsSenderConfiguration is the onboarding configuration of a sender.
type ChannelsSenderConfiguration struct {
	WabaId              *string `pulumi:"wabaId,optional"`
	VerificationMethod  *string `pulumi:"verificationMethod,optional"`
	VerificationCode    *string `pulumi:"verificationCode,optional" provider:"secret"`
	VoiceApplicationSid *string `pulumi:"voiceApplicationSid,optional"`
	AccountType         *string `pulumi:"accountType,optional"`
}

func (a *ChannelsSenderConfiguration) Annotate(an infer.Annotator) {
	an.Describe(&a.WabaId, "ID of the WhatsApp Business Account (WABA) the sender belongs to.")
	an.Describe(&a.VerificationMethod, "How Twilio delivers the one-time verification code to the phone number: sms or voice. Twilio uses sms when it is not set.")
	an.Describe(&a.VerificationCode, "The one-time code Twilio delivered, to finish a sender that is in the PENDING_VERIFICATION status. It is write-only: Twilio never returns it, so it is kept from your program and never shows as drift. It is sent on create, and again whenever you change it.")
	an.Describe(&a.VoiceApplicationSid, "SID (AP...) of the Twilio Voice application for the sender.")
	an.Describe(&a.AccountType, "The account type for an ISV account type migration: ISV or ISVSubAccount. Removing it leaves the value as it is on Twilio.")
}

// ChannelsSenderWebhook holds the webhooks of a sender.
type ChannelsSenderWebhook struct {
	CallbackUrl          *string `pulumi:"callbackUrl,optional"`
	CallbackMethod       *string `pulumi:"callbackMethod,optional"`
	FallbackUrl          *string `pulumi:"fallbackUrl,optional"`
	FallbackMethod       *string `pulumi:"fallbackMethod,optional"`
	StatusCallbackUrl    *string `pulumi:"statusCallbackUrl,optional"`
	StatusCallbackMethod *string `pulumi:"statusCallbackMethod,optional"`
}

func (a *ChannelsSenderWebhook) Annotate(an infer.Annotator) {
	an.Describe(&a.CallbackUrl, "URL Twilio sends incoming messages to.")
	an.Describe(&a.CallbackMethod, "HTTP method of the callback: POST or PUT.")
	an.Describe(&a.FallbackUrl, "URL Twilio sends to when the callback URL fails.")
	an.Describe(&a.FallbackMethod, "HTTP method of the fallback: POST or PUT.")
	an.Describe(&a.StatusCallbackUrl, "URL Twilio sends message status updates to.")
	an.Describe(&a.StatusCallbackMethod, "HTTP method of the status callback.")
}

// ChannelsSenderLabeledWebsite is one website of a sender profile.
type ChannelsSenderLabeledWebsite struct {
	Website string  `pulumi:"website"`
	Label   *string `pulumi:"label,optional"`
}

func (a *ChannelsSenderLabeledWebsite) Annotate(an infer.Annotator) {
	an.Describe(&a.Website, "The website URL.")
	an.Describe(&a.Label, "Label of the website.")
}

// ChannelsSenderLabeledEmail is one email address of a sender profile.
type ChannelsSenderLabeledEmail struct {
	Email string  `pulumi:"email"`
	Label *string `pulumi:"label,optional"`
}

func (a *ChannelsSenderLabeledEmail) Annotate(an infer.Annotator) {
	an.Describe(&a.Email, "The email address.")
	an.Describe(&a.Label, "Label of the email address.")
}

// ChannelsSenderLabeledPhone is one phone number of a sender profile.
type ChannelsSenderLabeledPhone struct {
	PhoneNumber string  `pulumi:"phoneNumber"`
	Label       *string `pulumi:"label,optional"`
}

func (a *ChannelsSenderLabeledPhone) Annotate(an infer.Annotator) {
	an.Describe(&a.PhoneNumber, "The phone number.")
	an.Describe(&a.Label, "Label of the phone number.")
}

// ChannelsSenderProfile is the business profile of a WhatsApp sender.
type ChannelsSenderProfile struct {
	Name              string                         `pulumi:"name"`
	About             *string                        `pulumi:"about,optional"`
	Address           *string                        `pulumi:"address,optional"`
	Description       *string                        `pulumi:"description,optional"`
	LogoUrl           *string                        `pulumi:"logoUrl,optional"`
	BannerUrl         *string                        `pulumi:"bannerUrl,optional"`
	PrivacyUrl        *string                        `pulumi:"privacyUrl,optional"`
	TermsOfServiceUrl *string                        `pulumi:"termsOfServiceUrl,optional"`
	AccentColor       *string                        `pulumi:"accentColor,optional"`
	Vertical          *string                        `pulumi:"vertical,optional"`
	Websites          []ChannelsSenderLabeledWebsite `pulumi:"websites,optional"`
	Emails            []ChannelsSenderLabeledEmail   `pulumi:"emails,optional"`
	PhoneNumbers      []ChannelsSenderLabeledPhone   `pulumi:"phoneNumbers,optional"`
}

func (a *ChannelsSenderProfile) Annotate(an infer.Annotator) {
	an.Describe(&a.Name, "The WhatsApp display name. It must follow Meta's display name guidelines. Twilio does not change the name straight away: a new name goes to Meta for review and the sender keeps its current name until Meta approves it and the sender is registered again. Watch pendingDisplayNameStatus. Meta limits the number of display name changes per 30 days.")
	an.Describe(&a.About, "The about text of the profile.")
	an.Describe(&a.Address, "The address of the business.")
	an.Describe(&a.Description, "The description of the business.")
	an.Describe(&a.LogoUrl, "URL of the logo.")
	an.Describe(&a.BannerUrl, "URL of the banner image.")
	an.Describe(&a.PrivacyUrl, "Publicly accessible HTTP or HTTPS URL of the privacy policy.")
	an.Describe(&a.TermsOfServiceUrl, "URL of the terms of service.")
	an.Describe(&a.AccentColor, "Color theme of the profile, in hex format.")
	an.Describe(&a.Vertical, "The industry vertical of the business, for example Retail.")
	an.Describe(&a.Websites, "Websites of the business.")
	an.Describe(&a.Emails, "Email addresses of the business.")
	an.Describe(&a.PhoneNumbers, "Phone numbers of the business.")
}

// ChannelsSenderProperties holds the quality figures WhatsApp reports for a sender.
type ChannelsSenderProperties struct {
	QualityRating  *string `pulumi:"qualityRating,optional"`
	MessagingLimit *string `pulumi:"messagingLimit,optional"`
}

func (a *ChannelsSenderProperties) Annotate(an infer.Annotator) {
	an.Describe(&a.QualityRating, "The quality rating of the sender.")
	an.Describe(&a.MessagingLimit, "The messaging limit of the sender.")
}

// ChannelsSenderOfflineReason is one reason a sender is offline.
type ChannelsSenderOfflineReason struct {
	Code     *string `pulumi:"code,optional"`
	Message  *string `pulumi:"message,optional"`
	MoreInfo *string `pulumi:"moreInfo,optional"`
}

func (a *ChannelsSenderOfflineReason) Annotate(an infer.Annotator) {
	an.Describe(&a.Code, "The error code.")
	an.Describe(&a.Message, "The error message.")
	an.Describe(&a.MoreInfo, "URL with more information about the error.")
}

// ------------------------------------------------------------------------- the resource

// ChannelsSender manages a WhatsApp sender of the account through the Messaging v2 Channels
// Senders API.
type ChannelsSender struct{}

// ChannelsSenderArgs are the inputs of a ChannelsSender. Only senderId replaces; the rest updates
// in place.
type ChannelsSenderArgs struct {
	SenderId      string                       `pulumi:"senderId" provider:"replaceOnChanges"`
	FriendlyName  *string                      `pulumi:"friendlyName,optional"`
	Configuration *ChannelsSenderConfiguration `pulumi:"configuration,optional"`
	Webhook       *ChannelsSenderWebhook       `pulumi:"webhook,optional"`
	Profile       ChannelsSenderProfile        `pulumi:"profile"`
}

func (a *ChannelsSenderArgs) Annotate(an infer.Annotator) {
	an.Describe(&a.SenderId, "The sender, as whatsapp:<phone number in E.164 format>, for example whatsapp:+15017122661. Changing it replaces the resource.")
	an.Describe(&a.FriendlyName, "Label of the sender in the Twilio Console. Removing it leaves the label as it is on Twilio.")
	an.Describe(&a.Configuration, "Onboarding configuration of the sender.")
	an.Describe(&a.Webhook, "Webhooks of the sender.")
	an.Describe(&a.Profile, "Business profile of the sender. profile.name is required.")
}

// ChannelsSenderState is the persisted state of a ChannelsSender.
type ChannelsSenderState struct {
	ChannelsSenderArgs

	Sid                          string                        `pulumi:"sid"`
	Status                       string                        `pulumi:"status"`
	PendingDisplayName           *string                       `pulumi:"pendingDisplayName,optional"`
	PendingDisplayNameStatus     *string                       `pulumi:"pendingDisplayNameStatus,optional"`
	PendingDisplayNameStatusDate *string                       `pulumi:"pendingDisplayNameStatusDate,optional"`
	Properties                   *ChannelsSenderProperties     `pulumi:"properties,optional"`
	OfflineReasons               []ChannelsSenderOfflineReason `pulumi:"offlineReasons,optional"`
	Url                          string                        `pulumi:"url"`
}

func (s *ChannelsSenderState) Annotate(a infer.Annotator) {
	a.Describe(&s.Sid, "The sender SID (XE...).")
	a.Describe(&s.Status, "Where the sender is in its life: CREATING, ONLINE, OFFLINE, PENDING_VERIFICATION, VERIFYING, ONLINE:UPDATING, TWILIO_REVIEW, DRAFT or STUBBED. Onboarding a WhatsApp sender can take from minutes to days, so the resource does not wait for ONLINE; it is refreshed on every read.")
	a.Describe(&s.PendingDisplayName, "The display name the latest name change applies to: under Meta review, approved and waiting to be applied, or (status COMPLETED) now in effect. Null when no name change was made, or once a completed change is no longer reported.")
	a.Describe(&s.PendingDisplayNameStatus, "Status of the latest display name change: PENDING_REVIEW, APPROVED, DECLINED, PIN_MISMATCH, REGISTRATION_FAILED, EXPIRED or COMPLETED. For PIN_MISMATCH, REGISTRATION_FAILED and EXPIRED, running the program again re-submits the name. A COMPLETED change is reported for 14 days.")
	a.Describe(&s.PendingDisplayNameStatusDate, "When pendingDisplayNameStatus last changed, as Twilio reports it.")
	a.Describe(&s.Properties, "Quality rating and messaging limit WhatsApp reports for the sender.")
	a.Describe(&s.OfflineReasons, "Why the sender is offline, when it is.")
	a.Describe(&s.Url, "The URL of the sender.")
}

func (r *ChannelsSender) Annotate(a infer.Annotator) {
	a.Describe(&r, "A Twilio WhatsApp sender (Messaging v2 Channels Senders; not a messaging service sender pool member). The resource ID is the sender SID (XE...). "+
		"Create registers the phone number and returns at once; the sender then goes through verification and Meta review, which can take days. Read `status`, `offlineReasons` and `pendingDisplayNameStatus` to follow it, and set configuration.verificationCode in a later update when Twilio asks for the one-time code (status PENDING_VERIFICATION). "+
		"A changed profile.name is submitted to Meta for review; until Meta approves it Twilio keeps reporting the old name, and this resource keeps your new name in state so there is no drift while the review is pending. If the review fails (DECLINED, EXPIRED, PIN_MISMATCH, REGISTRATION_FAILED) the active name is reported and the next update re-submits the name. "+
		"Optional settings that are removed leave the setting as it is on Twilio. RCS senders are not managed here because Twilio documents update and delete for WhatsApp only; read them with getChannelsSenders.")
}

var _ infer.CustomResource[ChannelsSenderArgs, ChannelsSenderState] = (*ChannelsSender)(nil)
var _ infer.CustomCheck[ChannelsSenderArgs] = (*ChannelsSender)(nil)
var _ infer.CustomRead[ChannelsSenderArgs, ChannelsSenderState] = (*ChannelsSender)(nil)
var _ infer.CustomUpdate[ChannelsSenderArgs, ChannelsSenderState] = (*ChannelsSender)(nil)
var _ infer.CustomDelete[ChannelsSenderState] = (*ChannelsSender)(nil)

func (r *ChannelsSender) Check(ctx context.Context, req infer.CheckRequest) (infer.CheckResponse[ChannelsSenderArgs], error) {
	args, failures, err := infer.DefaultCheck[ChannelsSenderArgs](ctx, req.NewInputs)
	if err != nil || len(failures) > 0 {
		return infer.CheckResponse[ChannelsSenderArgs]{Inputs: args, Failures: failures}, err
	}
	c := &checker{unk: unknowns(req.NewInputs)}
	if !c.unk("senderId") && !channelsSenderWhatsappRe.MatchString(args.SenderId) {
		c.fail("senderId", "senderId must be whatsapp:<phone number in E.164 format>, for example whatsapp:+15017122661 (RCS senders are not managed by this resource)")
	}
	if !c.unk("profile", "profile.name") {
		if args.Profile.Name == "" {
			c.fail("profile.name", "profile.name must not be empty")
		}
		c.maxLen("profile.name", args.Profile.Name, 255)
	}
	if args.FriendlyName != nil && !c.unk("friendlyName") && *args.FriendlyName == "" {
		c.fail("friendlyName", "friendlyName must not be empty")
	}
	enum := func(prop string, v *string, allowed []string) {
		if v != nil && !c.unk(prop) {
			c.failures = oneOf(c.failures, prop, v, allowed)
		}
	}
	if cf := args.Configuration; cf != nil {
		enum("configuration.verificationMethod", cf.VerificationMethod, []string{"sms", "voice"})
		enum("configuration.accountType", cf.AccountType, []string{"ISV", "ISVSubAccount"})
	}
	if w := args.Webhook; w != nil {
		enum("webhook.callbackMethod", w.CallbackMethod, channelsSenderMethods)
		enum("webhook.fallbackMethod", w.FallbackMethod, channelsSenderMethods)
	}
	return infer.CheckResponse[ChannelsSenderArgs]{Inputs: args, Failures: c.failures}, nil
}

// ------------------------------------------------------------------------- wire format

func put(m map[string]any, k string, v *string) {
	if v != nil {
		m[k] = *v
	}
}

// body is the JSON request body. sendCode says whether the one-time verification code goes along
// (always on create, on update only when it changed). create adds the senderId.
func (a ChannelsSenderArgs) body(create, sendCode bool) map[string]any {
	b := map[string]any{}
	if create {
		b["sender_id"] = a.SenderId
	}
	put(b, "friendly_name", a.FriendlyName)
	if cf := a.Configuration; cf != nil {
		m := map[string]any{}
		put(m, "waba_id", cf.WabaId)
		put(m, "verification_method", cf.VerificationMethod)
		put(m, "voice_application_sid", cf.VoiceApplicationSid)
		put(m, "account_type", cf.AccountType)
		if sendCode {
			put(m, "verification_code", cf.VerificationCode)
		}
		if len(m) > 0 {
			b["configuration"] = m
		}
	}
	if w := a.Webhook; w != nil {
		m := map[string]any{}
		put(m, "callback_url", w.CallbackUrl)
		put(m, "callback_method", w.CallbackMethod)
		put(m, "fallback_url", w.FallbackUrl)
		put(m, "fallback_method", w.FallbackMethod)
		put(m, "status_callback_url", w.StatusCallbackUrl)
		put(m, "status_callback_method", w.StatusCallbackMethod)
		if len(m) > 0 {
			b["webhook"] = m
		}
	}
	p := a.Profile
	m := map[string]any{"name": p.Name}
	put(m, "about", p.About)
	put(m, "address", p.Address)
	put(m, "description", p.Description)
	put(m, "logo_url", p.LogoUrl)
	put(m, "banner_url", p.BannerUrl)
	put(m, "privacy_url", p.PrivacyUrl)
	put(m, "terms_of_service_url", p.TermsOfServiceUrl)
	put(m, "accent_color", p.AccentColor)
	put(m, "vertical", p.Vertical)
	if len(p.Websites) > 0 {
		l := []any{}
		for _, w := range p.Websites {
			e := map[string]any{"website": w.Website}
			put(e, "label", w.Label)
			l = append(l, e)
		}
		m["websites"] = l
	}
	if len(p.Emails) > 0 {
		l := []any{}
		for _, w := range p.Emails {
			e := map[string]any{"email": w.Email}
			put(e, "label", w.Label)
			l = append(l, e)
		}
		m["emails"] = l
	}
	if len(p.PhoneNumbers) > 0 {
		l := []any{}
		for _, w := range p.PhoneNumbers {
			e := map[string]any{"phone_number": w.PhoneNumber}
			put(e, "label", w.Label)
			l = append(l, e)
		}
		m["phone_numbers"] = l
	}
	b["profile"] = m
	return b
}

// channelsSenderResponse is a decoded sender.
type channelsSenderResponse struct {
	Sid                          string
	Status                       string
	SenderId                     string
	FriendlyName                 *string
	Configuration                *ChannelsSenderConfiguration
	Webhook                      *ChannelsSenderWebhook
	Profile                      ChannelsSenderProfile
	ProfileUseCase               *string
	PendingDisplayName           *string
	PendingDisplayNameStatus     *string
	PendingDisplayNameStatusDate *string
	Properties                   *ChannelsSenderProperties
	OfflineReasons               []ChannelsSenderOfflineReason
	Compliance                   *ChannelsSenderCompliance
	Url                          string
}

func subObj(o obj, k string) (obj, bool) {
	m, ok := o[k].(map[string]any)
	return obj(m), ok && m != nil
}

func optNonEmpty(o obj, k string) *string { return nonEmpty(o.optStr(k)) }

func decodeLabeled(o obj, k string, each func(obj)) {
	items, _ := o[k].([]any)
	for _, it := range items {
		if m, ok := it.(map[string]any); ok {
			each(obj(m))
		}
	}
}

func decodeChannelsSender(o obj) channelsSenderResponse {
	r := channelsSenderResponse{Sid: o.str("sid"), Status: o.str("status"), SenderId: o.str("sender_id"),
		FriendlyName: optNonEmpty(o, "friendly_name"), PendingDisplayName: optNonEmpty(o, "pending_display_name"),
		PendingDisplayNameStatus:     optNonEmpty(o, "pending_display_name_status"),
		PendingDisplayNameStatusDate: optNonEmpty(o, "pending_display_name_status_date"), Url: o.str("url")}
	if c, ok := subObj(o, "configuration"); ok {
		r.Configuration = &ChannelsSenderConfiguration{WabaId: optNonEmpty(c, "waba_id"), VerificationMethod: optNonEmpty(c, "verification_method"),
			VoiceApplicationSid: optNonEmpty(c, "voice_application_sid"), AccountType: optNonEmpty(c, "account_type")}
	}
	if w, ok := subObj(o, "webhook"); ok {
		r.Webhook = &ChannelsSenderWebhook{CallbackUrl: optNonEmpty(w, "callback_url"), CallbackMethod: optNonEmpty(w, "callback_method"),
			FallbackUrl: optNonEmpty(w, "fallback_url"), FallbackMethod: optNonEmpty(w, "fallback_method"),
			StatusCallbackUrl: optNonEmpty(w, "status_callback_url"), StatusCallbackMethod: optNonEmpty(w, "status_callback_method")}
	}
	if p, ok := subObj(o, "profile"); ok {
		r.Profile = ChannelsSenderProfile{Name: p.str("name"), About: optNonEmpty(p, "about"), Address: optNonEmpty(p, "address"),
			Description: optNonEmpty(p, "description"), LogoUrl: optNonEmpty(p, "logo_url"), BannerUrl: optNonEmpty(p, "banner_url"),
			PrivacyUrl: optNonEmpty(p, "privacy_url"), TermsOfServiceUrl: optNonEmpty(p, "terms_of_service_url"),
			AccentColor: optNonEmpty(p, "accent_color"), Vertical: optNonEmpty(p, "vertical")}
		r.ProfileUseCase = optNonEmpty(p, "use_case")
		decodeLabeled(p, "websites", func(e obj) {
			r.Profile.Websites = append(r.Profile.Websites, ChannelsSenderLabeledWebsite{Website: e.str("website"), Label: optNonEmpty(e, "label")})
		})
		decodeLabeled(p, "emails", func(e obj) {
			r.Profile.Emails = append(r.Profile.Emails, ChannelsSenderLabeledEmail{Email: e.str("email"), Label: optNonEmpty(e, "label")})
		})
		decodeLabeled(p, "phone_numbers", func(e obj) {
			r.Profile.PhoneNumbers = append(r.Profile.PhoneNumbers, ChannelsSenderLabeledPhone{PhoneNumber: e.str("phone_number"), Label: optNonEmpty(e, "label")})
		})
	}
	if p, ok := subObj(o, "properties"); ok {
		r.Properties = &ChannelsSenderProperties{QualityRating: optNonEmpty(p, "quality_rating"), MessagingLimit: optNonEmpty(p, "messaging_limit")}
	}
	decodeLabeled(o, "offline_reasons", func(e obj) {
		r.OfflineReasons = append(r.OfflineReasons, ChannelsSenderOfflineReason{Code: optNonEmpty(e, "code"), Message: optNonEmpty(e, "message"), MoreInfo: optNonEmpty(e, "more_info")})
	})
	if c, ok := subObj(o, "compliance"); ok {
		r.Compliance = decodeChannelsSenderCompliance(c)
	}
	return r
}

// ------------------------------------------------------------------------ state building

// adoptStr takes Twilio's value for a setting the program manages (or on import); a setting
// Twilio does not report keeps the program's value, so a field the API omits never shows as drift.
func adoptStr(in, got *string, imported bool) *string {
	if got == nil {
		if imported {
			return nil
		}
		return in
	}
	if imported || in != nil {
		return got
	}
	return in
}

func adoptList[T any](in, got []T, imported bool) []T {
	if imported || len(in) > 0 {
		return got
	}
	return in
}

// mergeChannelsSenderInputs merges Twilio's view of a sender into the program's inputs. imported is
// true on an import, where the program has no inputs yet.
func mergeChannelsSenderInputs(in ChannelsSenderArgs, got channelsSenderResponse, imported bool) ChannelsSenderArgs {
	in.SenderId = got.SenderId
	in.FriendlyName = adoptStr(in.FriendlyName, got.FriendlyName, imported)
	if got.Configuration != nil && (imported || in.Configuration != nil) {
		if in.Configuration == nil {
			in.Configuration = &ChannelsSenderConfiguration{}
		}
		cf := *in.Configuration // VerificationCode stays as the program wrote it: Twilio never returns it.
		g := got.Configuration
		cf.WabaId = adoptStr(cf.WabaId, g.WabaId, imported)
		cf.VerificationMethod = adoptStr(cf.VerificationMethod, g.VerificationMethod, imported)
		cf.VoiceApplicationSid = adoptStr(cf.VoiceApplicationSid, g.VoiceApplicationSid, imported)
		cf.AccountType = adoptStr(cf.AccountType, g.AccountType, imported)
		in.Configuration = &cf
	}
	if got.Webhook != nil && (imported || in.Webhook != nil) {
		if in.Webhook == nil {
			in.Webhook = &ChannelsSenderWebhook{}
		}
		w, g := *in.Webhook, got.Webhook
		w.CallbackUrl = adoptStr(w.CallbackUrl, g.CallbackUrl, imported)
		w.CallbackMethod = adoptStr(w.CallbackMethod, g.CallbackMethod, imported)
		w.FallbackUrl = adoptStr(w.FallbackUrl, g.FallbackUrl, imported)
		w.FallbackMethod = adoptStr(w.FallbackMethod, g.FallbackMethod, imported)
		w.StatusCallbackUrl = adoptStr(w.StatusCallbackUrl, g.StatusCallbackUrl, imported)
		w.StatusCallbackMethod = adoptStr(w.StatusCallbackMethod, g.StatusCallbackMethod, imported)
		in.Webhook = &w
	}
	p, g := in.Profile, got.Profile
	// A new WhatsApp name stays pending at Meta while Twilio reports the old one. Keep the
	// program's name for as long as that review (or the re-registration after it) can still end
	// with the name applied; once it failed, report the active name so the next update re-submits.
	pendingKept := !imported && got.PendingDisplayName != nil && *got.PendingDisplayName == p.Name && g.Name != p.Name &&
		(got.PendingDisplayNameStatus == nil || !contains(channelsSenderFailedNames, *got.PendingDisplayNameStatus))
	if !pendingKept {
		p.Name = g.Name
	}
	p.About = adoptStr(p.About, g.About, imported)
	p.Address = adoptStr(p.Address, g.Address, imported)
	p.Description = adoptStr(p.Description, g.Description, imported)
	p.LogoUrl = adoptStr(p.LogoUrl, g.LogoUrl, imported)
	p.BannerUrl = adoptStr(p.BannerUrl, g.BannerUrl, imported)
	p.PrivacyUrl = adoptStr(p.PrivacyUrl, g.PrivacyUrl, imported)
	p.TermsOfServiceUrl = adoptStr(p.TermsOfServiceUrl, g.TermsOfServiceUrl, imported)
	p.AccentColor = adoptStr(p.AccentColor, g.AccentColor, imported)
	p.Vertical = adoptStr(p.Vertical, g.Vertical, imported)
	p.Websites = adoptList(p.Websites, g.Websites, imported)
	p.Emails = adoptList(p.Emails, g.Emails, imported)
	p.PhoneNumbers = adoptList(p.PhoneNumbers, g.PhoneNumbers, imported)
	in.Profile = p
	return in
}

func contains(l []string, s string) bool {
	for _, x := range l {
		if x == s {
			return true
		}
	}
	return false
}

func channelsSenderStateFrom(in ChannelsSenderArgs, r channelsSenderResponse) ChannelsSenderState {
	return ChannelsSenderState{ChannelsSenderArgs: in, Sid: r.Sid, Status: r.Status,
		PendingDisplayName: r.PendingDisplayName, PendingDisplayNameStatus: r.PendingDisplayNameStatus,
		PendingDisplayNameStatusDate: r.PendingDisplayNameStatusDate, Properties: r.Properties,
		OfflineReasons: r.OfflineReasons, Url: r.Url}
}

// ---------------------------------------------------------------------------- lifecycle

func (r *ChannelsSender) Create(ctx context.Context, req infer.CreateRequest[ChannelsSenderArgs]) (infer.CreateResponse[ChannelsSenderState], error) {
	type resp = infer.CreateResponse[ChannelsSenderState]
	a := req.Inputs
	if req.DryRun {
		return resp{Output: ChannelsSenderState{ChannelsSenderArgs: a}}, nil
	}
	c, err := clientFor(ctx)
	if err != nil {
		return resp{}, err
	}
	o, err := c.doJSON(ctx, "POST", channelsSenderProduct, channelsSenderPath(""), nil, a.body(true, true))
	if err != nil {
		return resp{}, fmt.Errorf("create channels sender: %w", err)
	}
	got := decodeChannelsSender(o)
	if got.Sid == "" {
		return resp{}, fmt.Errorf("create channels sender: Twilio returned no sid")
	}
	// Twilio answers 202: the sender is still being created. The answer's own status, name and
	// settings are recorded as reported; the program's inputs stay as written.
	return resp{ID: got.Sid, Output: channelsSenderStateFrom(a, got)}, nil
}

func (r *ChannelsSender) Read(ctx context.Context, req infer.ReadRequest[ChannelsSenderArgs, ChannelsSenderState]) (infer.ReadResponse[ChannelsSenderArgs, ChannelsSenderState], error) {
	type resp = infer.ReadResponse[ChannelsSenderArgs, ChannelsSenderState]
	c, err := clientFor(ctx)
	if err != nil {
		return resp{}, err
	}
	o, err := c.doJSON(ctx, "GET", channelsSenderProduct, channelsSenderPath(req.ID), nil, nil)
	if err != nil {
		if isNotFound(err) {
			if err := notFoundOnImport("channels sender", req.ID, req.Inputs, req.State); err != nil {
				return resp{}, err
			}
			return resp{}, nil
		}
		return resp{}, fmt.Errorf("read channels sender %s: %w", req.ID, err)
	}
	got := decodeChannelsSender(o)
	imported := req.State.Sid == ""
	in := mergeChannelsSenderInputs(req.Inputs, got, imported)
	return resp{ID: req.ID, Inputs: in, State: channelsSenderStateFrom(in, got)}, nil
}

func (r *ChannelsSender) Update(ctx context.Context, req infer.UpdateRequest[ChannelsSenderArgs, ChannelsSenderState]) (infer.UpdateResponse[ChannelsSenderState], error) {
	type resp = infer.UpdateResponse[ChannelsSenderState]
	st := req.State
	st.ChannelsSenderArgs = req.Inputs
	if req.DryRun {
		return resp{Output: st}, nil
	}
	c, err := clientFor(ctx)
	if err != nil {
		return resp{}, err
	}
	// The one-time code goes only when it is new: a spent code would be refused.
	sendCode := req.Inputs.Configuration != nil && req.Inputs.Configuration.VerificationCode != nil &&
		(req.State.Configuration == nil || req.State.Configuration.VerificationCode == nil ||
			*req.State.Configuration.VerificationCode != *req.Inputs.Configuration.VerificationCode)
	if _, err := c.doJSON(ctx, "POST", channelsSenderProduct, channelsSenderPath(req.ID), nil, req.Inputs.body(false, sendCode)); err != nil {
		return resp{}, fmt.Errorf("update channels sender %s: %w", req.ID, err)
	}
	// The update answer lacks the pending display name fields, so read the sender back.
	o, err := c.doJSON(ctx, "GET", channelsSenderProduct, channelsSenderPath(req.ID), nil, nil)
	if err != nil {
		return resp{}, fmt.Errorf("read channels sender %s after update: %w", req.ID, err)
	}
	st = channelsSenderStateFrom(req.Inputs, decodeChannelsSender(o))
	return resp{Output: st}, nil
}

func (r *ChannelsSender) Delete(ctx context.Context, req infer.DeleteRequest[ChannelsSenderState]) (infer.DeleteResponse, error) {
	c, err := clientFor(ctx)
	if err != nil {
		return infer.DeleteResponse{}, err
	}
	if _, err := c.doJSON(ctx, "DELETE", channelsSenderProduct, channelsSenderPath(req.ID), nil, nil); err != nil && !isNotFound(err) {
		return infer.DeleteResponse{}, fmt.Errorf("delete channels sender %s: %w", req.ID, err)
	}
	return infer.DeleteResponse{}, nil
}
