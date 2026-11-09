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

func requireArgs(fn string, pairs ...[2]string) error {
	for _, a := range pairs {
		if err := requireArg(fn, a[0], a[1]); err != nil {
			return err
		}
	}
	return nil
}

// ------------------------------------------------------------ getVerifyRateLimit

// GetVerifyRateLimit looks up a rate limit of a Verify service.
type GetVerifyRateLimit struct{}

// GetVerifyRateLimitArgs are the lookup arguments.
type GetVerifyRateLimitArgs struct {
	ServiceSid string `pulumi:"serviceSid"`
	Sid        string `pulumi:"sid"`
}

func (a *GetVerifyRateLimitArgs) Annotate(an infer.Annotator) {
	an.Describe(&a.ServiceSid, "SID (VA...) of the Verify service.")
	an.Describe(&a.Sid, "SID (RK...) of the rate limit.")
}

// GetVerifyRateLimitResult is the looked-up rate limit.
type GetVerifyRateLimitResult struct {
	ServiceSid  string  `pulumi:"serviceSid"`
	Sid         string  `pulumi:"sid"`
	UniqueName  string  `pulumi:"uniqueName"`
	Description *string `pulumi:"description,optional"`
	DateCreated string  `pulumi:"dateCreated"`
	DateUpdated string  `pulumi:"dateUpdated"`
}

func (a *GetVerifyRateLimitResult) Annotate(an infer.Annotator) {
	an.Describe(&a.ServiceSid, "SID of the Verify service.")
	an.Describe(&a.Sid, "The rate limit SID (RK...).")
	an.Describe(&a.UniqueName, "Unique name of the rate limit.")
	an.Describe(&a.Description, "Description of the rate limit, when it has one.")
	an.Describe(&a.DateCreated, "When the rate limit was created (ISO 8601).")
	an.Describe(&a.DateUpdated, "When the rate limit was last updated (ISO 8601).")
}

func (f *GetVerifyRateLimit) Annotate(a infer.Annotator) {
	a.Describe(&f, "Look up a rate limit of a Verify service by SID.")
}

func (f *GetVerifyRateLimit) Invoke(ctx context.Context, req infer.FunctionRequest[GetVerifyRateLimitArgs]) (infer.FunctionResponse[GetVerifyRateLimitResult], error) {
	const fn = "getVerifyRateLimit"
	in := req.Input
	if err := requireArgs(fn, [2]string{"serviceSid", in.ServiceSid}, [2]string{"sid", in.Sid}); err != nil {
		return infer.FunctionResponse[GetVerifyRateLimitResult]{}, err
	}
	c, err := clientFor(ctx)
	if err != nil {
		return infer.FunctionResponse[GetVerifyRateLimitResult]{}, err
	}
	o, err := c.do(ctx, "GET", "verify", verifyChildPath(in.ServiceSid, "RateLimits", in.Sid), nil, nil)
	if err != nil {
		return infer.FunctionResponse[GetVerifyRateLimitResult]{}, lookupFailure(fn, "rate limit "+in.Sid+" of Verify service "+in.ServiceSid, err)
	}
	return infer.FunctionResponse[GetVerifyRateLimitResult]{Output: GetVerifyRateLimitResult{
		ServiceSid: in.ServiceSid, Sid: o.str("sid"), UniqueName: o.str("unique_name"), Description: nonEmpty(o.optStr("description")),
		DateCreated: o.str("date_created"), DateUpdated: o.str("date_updated"),
	}}, nil
}

// ------------------------------------------------------ getVerifyRateLimitBucket

// GetVerifyRateLimitBucket looks up a bucket of a Verify rate limit.
type GetVerifyRateLimitBucket struct{}

// GetVerifyRateLimitBucketArgs are the lookup arguments.
type GetVerifyRateLimitBucketArgs struct {
	ServiceSid   string `pulumi:"serviceSid"`
	RateLimitSid string `pulumi:"rateLimitSid"`
	Sid          string `pulumi:"sid"`
}

func (a *GetVerifyRateLimitBucketArgs) Annotate(an infer.Annotator) {
	an.Describe(&a.ServiceSid, "SID (VA...) of the Verify service.")
	an.Describe(&a.RateLimitSid, "SID (RK...) of the rate limit.")
	an.Describe(&a.Sid, "SID (BL...) of the bucket.")
}

// GetVerifyRateLimitBucketResult is the looked-up bucket.
type GetVerifyRateLimitBucketResult struct {
	ServiceSid   string `pulumi:"serviceSid"`
	RateLimitSid string `pulumi:"rateLimitSid"`
	Sid          string `pulumi:"sid"`
	Max          int    `pulumi:"max"`
	Interval     int    `pulumi:"interval"`
	DateCreated  string `pulumi:"dateCreated"`
	DateUpdated  string `pulumi:"dateUpdated"`
}

func (a *GetVerifyRateLimitBucketResult) Annotate(an infer.Annotator) {
	an.Describe(&a.ServiceSid, "SID of the Verify service.")
	an.Describe(&a.RateLimitSid, "SID of the rate limit.")
	an.Describe(&a.Sid, "The bucket SID (BL...).")
	an.Describe(&a.Max, "Maximum number of requests permitted during the interval.")
	an.Describe(&a.Interval, "Number of seconds the rate limit is enforced over.")
	an.Describe(&a.DateCreated, "When the bucket was created (ISO 8601).")
	an.Describe(&a.DateUpdated, "When the bucket was last updated (ISO 8601).")
}

func (f *GetVerifyRateLimitBucket) Annotate(a infer.Annotator) {
	a.Describe(&f, "Look up a bucket of a Verify rate limit by SID.")
}

func (f *GetVerifyRateLimitBucket) Invoke(ctx context.Context, req infer.FunctionRequest[GetVerifyRateLimitBucketArgs]) (infer.FunctionResponse[GetVerifyRateLimitBucketResult], error) {
	const fn = "getVerifyRateLimitBucket"
	in := req.Input
	if err := requireArgs(fn, [2]string{"serviceSid", in.ServiceSid}, [2]string{"rateLimitSid", in.RateLimitSid}, [2]string{"sid", in.Sid}); err != nil {
		return infer.FunctionResponse[GetVerifyRateLimitBucketResult]{}, err
	}
	c, err := clientFor(ctx)
	if err != nil {
		return infer.FunctionResponse[GetVerifyRateLimitBucketResult]{}, err
	}
	o, err := c.do(ctx, "GET", "verify", verifyBucketPath(in.ServiceSid, in.RateLimitSid, in.Sid), nil, nil)
	if err != nil {
		return infer.FunctionResponse[GetVerifyRateLimitBucketResult]{}, lookupFailure(fn, "bucket "+in.Sid+" of rate limit "+in.RateLimitSid, err)
	}
	res := GetVerifyRateLimitBucketResult{ServiceSid: in.ServiceSid, RateLimitSid: in.RateLimitSid, Sid: o.str("sid"),
		DateCreated: o.str("date_created"), DateUpdated: o.str("date_updated")}
	if v := o.optInt("max"); v != nil {
		res.Max = *v
	}
	if v := o.optInt("interval"); v != nil {
		res.Interval = *v
	}
	return infer.FunctionResponse[GetVerifyRateLimitBucketResult]{Output: res}, nil
}

// ------------------------------------------------ getVerifyMessagingConfiguration

// GetVerifyMessagingConfiguration looks up the per-country Messaging Service of a Verify service.
type GetVerifyMessagingConfiguration struct{}

// GetVerifyMessagingConfigurationArgs are the lookup arguments.
type GetVerifyMessagingConfigurationArgs struct {
	ServiceSid string `pulumi:"serviceSid"`
	Country    string `pulumi:"country"`
}

func (a *GetVerifyMessagingConfigurationArgs) Annotate(an infer.Annotator) {
	an.Describe(&a.ServiceSid, "SID (VA...) of the Verify service.")
	an.Describe(&a.Country, "ISO 3166-1 alpha-2 country code, or all for the global configuration.")
}

// GetVerifyMessagingConfigurationResult is the looked-up configuration.
type GetVerifyMessagingConfigurationResult struct {
	ServiceSid          string `pulumi:"serviceSid"`
	Country             string `pulumi:"country"`
	MessagingServiceSid string `pulumi:"messagingServiceSid"`
	DateCreated         string `pulumi:"dateCreated"`
	DateUpdated         string `pulumi:"dateUpdated"`
}

func (a *GetVerifyMessagingConfigurationResult) Annotate(an infer.Annotator) {
	an.Describe(&a.ServiceSid, "SID of the Verify service.")
	an.Describe(&a.Country, "The country the configuration applies to, as Twilio reports it.")
	an.Describe(&a.MessagingServiceSid, "SID (MG...) of the Messaging Service used to send SMS to the country.")
	an.Describe(&a.DateCreated, "When the configuration was created (ISO 8601).")
	an.Describe(&a.DateUpdated, "When the configuration was last updated (ISO 8601).")
}

func (f *GetVerifyMessagingConfiguration) Annotate(a infer.Annotator) {
	a.Describe(&f, "Look up the Messaging Service a Verify service uses for one country.")
}

func (f *GetVerifyMessagingConfiguration) Invoke(ctx context.Context, req infer.FunctionRequest[GetVerifyMessagingConfigurationArgs]) (infer.FunctionResponse[GetVerifyMessagingConfigurationResult], error) {
	const fn = "getVerifyMessagingConfiguration"
	in := req.Input
	if err := requireArgs(fn, [2]string{"serviceSid", in.ServiceSid}, [2]string{"country", in.Country}); err != nil {
		return infer.FunctionResponse[GetVerifyMessagingConfigurationResult]{}, err
	}
	c, err := clientFor(ctx)
	if err != nil {
		return infer.FunctionResponse[GetVerifyMessagingConfigurationResult]{}, err
	}
	o, err := c.do(ctx, "GET", "verify", verifyChildPath(in.ServiceSid, "MessagingConfigurations", in.Country), nil, nil)
	if err != nil {
		return infer.FunctionResponse[GetVerifyMessagingConfigurationResult]{}, lookupFailure(fn, "messaging configuration for "+in.Country+" of Verify service "+in.ServiceSid, err)
	}
	country := o.str("country")
	if country == "" {
		country = in.Country
	}
	return infer.FunctionResponse[GetVerifyMessagingConfigurationResult]{Output: GetVerifyMessagingConfigurationResult{
		ServiceSid: in.ServiceSid, Country: country, MessagingServiceSid: o.str("messaging_service_sid"),
		DateCreated: o.str("date_created"), DateUpdated: o.str("date_updated"),
	}}, nil
}

// ------------------------------------------------------------- getVerifyWebhook

// GetVerifyWebhook looks up a webhook of a Verify service.
type GetVerifyWebhook struct{}

// GetVerifyWebhookArgs are the lookup arguments.
type GetVerifyWebhookArgs struct {
	ServiceSid string `pulumi:"serviceSid"`
	Sid        string `pulumi:"sid"`
}

func (a *GetVerifyWebhookArgs) Annotate(an infer.Annotator) {
	an.Describe(&a.ServiceSid, "SID (VA...) of the Verify service.")
	an.Describe(&a.Sid, "SID (YW...) of the webhook.")
}

// GetVerifyWebhookResult is the looked-up webhook.
type GetVerifyWebhookResult struct {
	ServiceSid     string   `pulumi:"serviceSid"`
	Sid            string   `pulumi:"sid"`
	FriendlyName   string   `pulumi:"friendlyName"`
	EventTypes     []string `pulumi:"eventTypes"`
	WebhookUrl     string   `pulumi:"webhookUrl"`
	WebhookMethod  string   `pulumi:"webhookMethod"`
	Status         string   `pulumi:"status"`
	WebhookVersion string   `pulumi:"webhookVersion"`
	DateCreated    string   `pulumi:"dateCreated"`
	DateUpdated    string   `pulumi:"dateUpdated"`
}

func (a *GetVerifyWebhookResult) Annotate(an infer.Annotator) {
	an.Describe(&a.ServiceSid, "SID of the Verify service.")
	an.Describe(&a.Sid, "The webhook SID (YW...).")
	an.Describe(&a.FriendlyName, "Name of the webhook.")
	an.Describe(&a.EventTypes, "Events the webhook is subscribed to.")
	an.Describe(&a.WebhookUrl, "URL Twilio posts the events to.")
	an.Describe(&a.WebhookMethod, "HTTP method Twilio uses to call the URL (GET or POST).")
	an.Describe(&a.Status, "Webhook status: enabled or disabled.")
	an.Describe(&a.WebhookVersion, "Webhook payload version: v1 or v2.")
	an.Describe(&a.DateCreated, "When the webhook was created (ISO 8601).")
	an.Describe(&a.DateUpdated, "When the webhook was last updated (ISO 8601).")
}

func (f *GetVerifyWebhook) Annotate(a infer.Annotator) {
	a.Describe(&f, "Look up a webhook of a Verify service by SID.")
}

func (f *GetVerifyWebhook) Invoke(ctx context.Context, req infer.FunctionRequest[GetVerifyWebhookArgs]) (infer.FunctionResponse[GetVerifyWebhookResult], error) {
	const fn = "getVerifyWebhook"
	in := req.Input
	if err := requireArgs(fn, [2]string{"serviceSid", in.ServiceSid}, [2]string{"sid", in.Sid}); err != nil {
		return infer.FunctionResponse[GetVerifyWebhookResult]{}, err
	}
	c, err := clientFor(ctx)
	if err != nil {
		return infer.FunctionResponse[GetVerifyWebhookResult]{}, err
	}
	o, err := c.do(ctx, "GET", "verify", verifyChildPath(in.ServiceSid, "Webhooks", in.Sid), nil, nil)
	if err != nil {
		return infer.FunctionResponse[GetVerifyWebhookResult]{}, lookupFailure(fn, "webhook "+in.Sid+" of Verify service "+in.ServiceSid, err)
	}
	types := []string{}
	if l := o.optList("event_types"); l != nil {
		types = *l
	}
	return infer.FunctionResponse[GetVerifyWebhookResult]{Output: GetVerifyWebhookResult{
		ServiceSid: in.ServiceSid, Sid: o.str("sid"), FriendlyName: o.str("friendly_name"), EventTypes: types,
		WebhookUrl: o.str("webhook_url"), WebhookMethod: o.str("webhook_method"), Status: o.str("status"), WebhookVersion: o.str("version"),
		DateCreated: o.str("date_created"), DateUpdated: o.str("date_updated"),
	}}, nil
}

// ----------------------------------------------------- getVerifySafeListNumber

// GetVerifySafeListNumber looks up a phone number on the Verify SafeList.
type GetVerifySafeListNumber struct{}

// GetVerifySafeListNumberArgs are the lookup arguments.
type GetVerifySafeListNumberArgs struct {
	PhoneNumber string `pulumi:"phoneNumber"`
}

func (a *GetVerifySafeListNumberArgs) Annotate(an infer.Annotator) {
	an.Describe(&a.PhoneNumber, "The phone number to look up, in E.164 format.")
}

// GetVerifySafeListNumberResult is the looked-up SafeList entry.
type GetVerifySafeListNumberResult struct {
	PhoneNumber string `pulumi:"phoneNumber"`
	Sid         string `pulumi:"sid"`
}

func (a *GetVerifySafeListNumberResult) Annotate(an infer.Annotator) {
	an.Describe(&a.PhoneNumber, "The phone number on the SafeList.")
	an.Describe(&a.Sid, "The SafeList entry SID, as Twilio reports it.")
}

func (f *GetVerifySafeListNumber) Annotate(a infer.Annotator) {
	a.Describe(&f, "Look up a phone number on the account's Verify SafeList. Fails when the number is not on the list.")
}

func (f *GetVerifySafeListNumber) Invoke(ctx context.Context, req infer.FunctionRequest[GetVerifySafeListNumberArgs]) (infer.FunctionResponse[GetVerifySafeListNumberResult], error) {
	const fn = "getVerifySafeListNumber"
	in := req.Input
	if err := requireArg(fn, "phoneNumber", in.PhoneNumber); err != nil {
		return infer.FunctionResponse[GetVerifySafeListNumberResult]{}, err
	}
	c, err := clientFor(ctx)
	if err != nil {
		return infer.FunctionResponse[GetVerifySafeListNumberResult]{}, err
	}
	o, err := c.do(ctx, "GET", "verify", safeListPath(in.PhoneNumber), nil, nil)
	if err != nil {
		return infer.FunctionResponse[GetVerifySafeListNumberResult]{}, lookupFailure(fn, "SafeList number "+in.PhoneNumber, err)
	}
	number := o.str("phone_number")
	if number == "" {
		number = in.PhoneNumber
	}
	return infer.FunctionResponse[GetVerifySafeListNumberResult]{Output: GetVerifySafeListNumberResult{PhoneNumber: number, Sid: o.str("sid")}}, nil
}

// ---------------------------------------------------------- getVerifyTemplates

// GetVerifyTemplates lists the Verify message templates Twilio provides.
type GetVerifyTemplates struct{}

// GetVerifyTemplatesArgs are the lookup arguments.
type GetVerifyTemplatesArgs struct {
	FriendlyName *string `pulumi:"friendlyName,optional"`
}

func (a *GetVerifyTemplatesArgs) Annotate(an infer.Annotator) {
	an.Describe(&a.FriendlyName, "Only return templates with this exact name.")
}

// VerifyTemplate is one Verify message template.
type VerifyTemplate struct {
	Sid          string   `pulumi:"sid"`
	FriendlyName string   `pulumi:"friendlyName"`
	Channels     []string `pulumi:"channels"`
}

func (a *VerifyTemplate) Annotate(an infer.Annotator) {
	an.Describe(&a.Sid, "The template SID (HJ...). Use it as VerifyService.defaultTemplateSid.")
	an.Describe(&a.FriendlyName, "Name of the template.")
	an.Describe(&a.Channels, "Channels that support the template (sms, voice).")
}

// GetVerifyTemplatesResult is the list of templates.
type GetVerifyTemplatesResult struct {
	Templates []VerifyTemplate `pulumi:"templates"`
}

func (a *GetVerifyTemplatesResult) Annotate(an infer.Annotator) {
	an.Describe(&a.Templates, "The templates, in the order Twilio returns them.")
}

func (f *GetVerifyTemplates) Annotate(a infer.Annotator) {
	a.Describe(&f, "List the Verify message templates available to the account, optionally filtered by exact friendlyName.")
}

func (f *GetVerifyTemplates) Invoke(ctx context.Context, req infer.FunctionRequest[GetVerifyTemplatesArgs]) (infer.FunctionResponse[GetVerifyTemplatesResult], error) {
	const fn = "getVerifyTemplates"
	c, err := clientFor(ctx)
	if err != nil {
		return infer.FunctionResponse[GetVerifyTemplatesResult]{}, err
	}
	out := []VerifyTemplate{}
	query := url.Values{"PageSize": {"100"}}
	if n := req.Input.FriendlyName; n != nil && *n != "" {
		query.Set("FriendlyName", *n)
	}
	for page := 0; page < 100; page++ {
		o, err := c.do(ctx, "GET", "verify", "/v2/Templates", query, nil)
		if err != nil {
			return infer.FunctionResponse[GetVerifyTemplatesResult]{}, fmt.Errorf("%s: %w", fn, err)
		}
		items, _ := o["templates"].([]any)
		for _, it := range items {
			m, ok := it.(map[string]any)
			if !ok {
				continue
			}
			t := obj(m)
			ch := []string{}
			if l := t.optList("channels"); l != nil {
				ch = *l
			}
			out = append(out, VerifyTemplate{Sid: t.str("sid"), FriendlyName: t.str("friendly_name"), Channels: ch})
		}
		next := o.str("meta.next_page_url")
		if next == "" {
			return infer.FunctionResponse[GetVerifyTemplatesResult]{Output: GetVerifyTemplatesResult{Templates: out}}, nil
		}
		u, err := url.Parse(next)
		if err != nil {
			return infer.FunctionResponse[GetVerifyTemplatesResult]{}, fmt.Errorf("%s: parse next page URL: %w", fn, err)
		}
		query = u.Query()
	}
	return infer.FunctionResponse[GetVerifyTemplatesResult]{}, fmt.Errorf("%s: more than 100 pages of templates", fn)
}
