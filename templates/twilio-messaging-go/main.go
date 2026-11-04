package main

import (
	"github.com/pulumi-labs/pulumi-twilio/sdk/go/twilio"
	"github.com/pulumi/pulumi/sdk/v3/go/pulumi"
	"github.com/pulumi/pulumi/sdk/v3/go/pulumi/config"
)

func main() {
	pulumi.Run(func(ctx *pulumi.Context) error {
		// Credentials come from `pulumi config set twilio:accountSid ...` / `twilio:authToken --secret`
		// or the TWILIO_ACCOUNT_SID / TWILIO_AUTH_TOKEN environment variables.
		cfg := config.New(ctx, "")
		voiceURL := cfg.Get("voiceUrl")
		if voiceURL == "" {
			voiceURL = "https://example.com/webhooks/voice"
		}
		smsURL := cfg.Get("smsUrl")
		if smsURL == "" {
			smsURL = "https://example.com/webhooks/sms"
		}

		app, err := twilio.NewApplication(ctx, "app", &twilio.ApplicationArgs{
			FriendlyName: pulumi.String("pulumi-messaging"),
			VoiceUrl:     pulumi.String(voiceURL),
			VoiceMethod:  pulumi.String("POST"),
			SmsUrl:       pulumi.String(smsURL),
			SmsMethod:    pulumi.String("POST"),
		})
		if err != nil {
			return err
		}

		svc, err := twilio.NewMessagingService(ctx, "svc", &twilio.MessagingServiceArgs{
			FriendlyName:      pulumi.String("pulumi-alerts"),
			InboundRequestUrl: pulumi.String(smsURL),
			InboundMethod:     pulumi.String("POST"),
			StickySender:      pulumi.Bool(true),
		})
		if err != nil {
			return err
		}

		ctx.Export("applicationSid", app.Sid)
		ctx.Export("messagingServiceSid", svc.Sid)
		return nil
	})
}
