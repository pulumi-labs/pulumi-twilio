package provider

import (
	"context"
	"fmt"

	"github.com/pulumi/pulumi-go-provider/infer"
)

// GetAccount looks up a Twilio account (default: the provider account).
type GetAccount struct{}

// GetAccountArgs are the lookup arguments.
type GetAccountArgs struct {
	AccountSid *string `pulumi:"accountSid,optional"`
}

func (a *GetAccountArgs) Annotate(an infer.Annotator) {
	an.Describe(&a.AccountSid, "SID of the account or subaccount to look up. Defaults to the provider account.")
}

// GetAccountResult is the looked-up account.
type GetAccountResult struct {
	Sid             string `pulumi:"sid"`
	FriendlyName    string `pulumi:"friendlyName"`
	Status          string `pulumi:"status"`
	Type            string `pulumi:"type"`
	OwnerAccountSid string `pulumi:"ownerAccountSid"`
	DateCreated     string `pulumi:"dateCreated"`
}

func (a *GetAccountResult) Annotate(an infer.Annotator) {
	an.Describe(&a.Sid, "Account SID (AC...).")
	an.Describe(&a.FriendlyName, "Account name.")
	an.Describe(&a.Status, "active, suspended or closed.")
	an.Describe(&a.Type, "Trial or Full.")
	an.Describe(&a.OwnerAccountSid, "SID of the parent account (the account itself for a top-level account).")
	an.Describe(&a.DateCreated, "When the account was created (RFC 2822).")
}

func (f *GetAccount) Annotate(a infer.Annotator) {
	a.Describe(&f, "Look up a Twilio account or subaccount by SID. The auth token is deliberately not returned.")
}

func (f *GetAccount) Invoke(ctx context.Context, req infer.FunctionRequest[GetAccountArgs]) (infer.FunctionResponse[GetAccountResult], error) {
	c, err := clientFor(ctx)
	if err != nil {
		return infer.FunctionResponse[GetAccountResult]{}, err
	}
	sid := c.acct(req.Input.AccountSid)
	o, err := c.do(ctx, "GET", "api", accountPath(sid), nil, nil)
	if err != nil {
		return infer.FunctionResponse[GetAccountResult]{}, fmt.Errorf("getAccount %s: %w", sid, err)
	}
	return infer.FunctionResponse[GetAccountResult]{Output: GetAccountResult{
		Sid: o.str("sid"), FriendlyName: o.str("friendly_name"), Status: o.str("status"), Type: o.str("type"),
		OwnerAccountSid: o.str("owner_account_sid"), DateCreated: o.str("date_created"),
	}}, nil
}

// GetIncomingPhoneNumber looks up an owned phone number.
type GetIncomingPhoneNumber struct{}

// GetIncomingPhoneNumberArgs are the lookup arguments; set exactly one of phoneNumber or sid.
type GetIncomingPhoneNumberArgs struct {
	PhoneNumber *string `pulumi:"phoneNumber,optional"`
	Sid         *string `pulumi:"sid,optional"`
	AccountSid  *string `pulumi:"accountSid,optional"`
}

func (a *GetIncomingPhoneNumberArgs) Annotate(an infer.Annotator) {
	an.Describe(&a.PhoneNumber, "The owned number in E.164 format. Exactly one of phoneNumber or sid must be set.")
	an.Describe(&a.Sid, "The number SID (PN...). Exactly one of phoneNumber or sid must be set.")
	an.Describe(&a.AccountSid, "SID of the account that owns the number. Defaults to the provider account.")
}

// GetIncomingPhoneNumberResult is the looked-up number.
type GetIncomingPhoneNumberResult struct {
	Sid                 string `pulumi:"sid"`
	PhoneNumber         string `pulumi:"phoneNumber"`
	FriendlyName        string `pulumi:"friendlyName"`
	VoiceUrl            string `pulumi:"voiceUrl"`
	SmsUrl              string `pulumi:"smsUrl"`
	VoiceApplicationSid string `pulumi:"voiceApplicationSid"`
	SmsApplicationSid   string `pulumi:"smsApplicationSid"`
	Voice               bool   `pulumi:"voice"`
	Sms                 bool   `pulumi:"sms"`
	Mms                 bool   `pulumi:"mms"`
}

func (a *GetIncomingPhoneNumberResult) Annotate(an infer.Annotator) {
	an.Describe(&a.Sid, "The number SID (PN...).")
	an.Describe(&a.PhoneNumber, "The number in E.164 format.")
	an.Describe(&a.FriendlyName, "Label of the number.")
	an.Describe(&a.VoiceUrl, "Current voice webhook URL (empty when none).")
	an.Describe(&a.SmsUrl, "Current SMS webhook URL (empty when none).")
	an.Describe(&a.VoiceApplicationSid, "Application handling voice, if any.")
	an.Describe(&a.SmsApplicationSid, "Application handling SMS, if any.")
	an.Describe(&a.Voice, "Whether the number can handle voice calls.")
	an.Describe(&a.Sms, "Whether the number can send and receive SMS.")
	an.Describe(&a.Mms, "Whether the number can send and receive MMS.")
}

func (f *GetIncomingPhoneNumber) Annotate(a infer.Annotator) {
	a.Describe(&f, "Look up a phone number owned by the account by E.164 number or SID.")
}

func (f *GetIncomingPhoneNumber) Invoke(ctx context.Context, req infer.FunctionRequest[GetIncomingPhoneNumberArgs]) (infer.FunctionResponse[GetIncomingPhoneNumberResult], error) {
	in := req.Input
	hasNum, hasSid := deref(in.PhoneNumber) != "", deref(in.Sid) != ""
	if hasNum == hasSid {
		return infer.FunctionResponse[GetIncomingPhoneNumberResult]{}, fmt.Errorf("getIncomingPhoneNumber: set exactly one of phoneNumber or sid")
	}
	c, err := clientFor(ctx)
	if err != nil {
		return infer.FunctionResponse[GetIncomingPhoneNumberResult]{}, err
	}
	acct := c.acct(in.AccountSid)
	o, err := findNumber(ctx, c, acct, IncomingPhoneNumberArgs{PhoneNumber: in.PhoneNumber, PhoneNumberSid: in.Sid})
	if err != nil {
		return infer.FunctionResponse[GetIncomingPhoneNumberResult]{}, fmt.Errorf("getIncomingPhoneNumber: %w", err)
	}
	caps, _ := o["capabilities"].(map[string]any)
	flag := func(k string) bool { b, _ := caps[k].(bool); return b }
	return infer.FunctionResponse[GetIncomingPhoneNumberResult]{Output: GetIncomingPhoneNumberResult{
		Sid: o.str("sid"), PhoneNumber: o.str("phone_number"), FriendlyName: o.str("friendly_name"),
		VoiceUrl: o.str("voice_url"), SmsUrl: o.str("sms_url"),
		VoiceApplicationSid: o.str("voice_application_sid"), SmsApplicationSid: o.str("sms_application_sid"),
		Voice: flag("voice"), Sms: flag("sms"), Mms: flag("mms"),
	}}, nil
}
