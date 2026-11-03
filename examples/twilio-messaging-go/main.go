// Credentials come from config (`pulumi config set twilio:accountSid ...` and `pulumi config set --secret twilio:authToken ...`) or the TWILIO_ACCOUNT_SID / TWILIO_AUTH_TOKEN environment variables.
package main

import (
	"github.com/pulumi/pulumi-twilio/sdk/go/twilio"
	"github.com/pulumi/pulumi/sdk/v3/go/pulumi"
	"github.com/pulumi/pulumi/sdk/v3/go/pulumi/config"
)

func main() {
	pulumi.Run(func(ctx *pulumi.Context) error {
		cfg := config.New(ctx, "")
		smsURL := cfg.Get("smsUrl")
		if smsURL == "" {
			smsURL = "https://example.com/webhooks/sms"
		}
		voiceURL := cfg.Get("voiceUrl")
		if voiceURL == "" {
			voiceURL = "https://example.com/webhooks/voice"
		}
		phoneNumber := cfg.Get("phoneNumber") // E.164 number the account already owns (optional)

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

		if phoneNumber != "" {
			// Adopts a number the account already owns: this never purchases a number, and destroying the stack does not release it.
			number, err := twilio.NewIncomingPhoneNumber(ctx, "number", &twilio.IncomingPhoneNumberArgs{
				PhoneNumber:         pulumi.String(phoneNumber),
				VoiceApplicationSid: app.Sid,
			})
			if err != nil {
				return err
			}
			_, err = twilio.NewMessagingServicePhoneNumber(ctx, "sender", &twilio.MessagingServicePhoneNumberArgs{
				ServiceSid:     svc.Sid,
				PhoneNumberSid: number.Sid,
			})
			if err != nil {
				return err
			}
			ctx.Export("numberSid", number.Sid)
		}
		return nil
	})
}
