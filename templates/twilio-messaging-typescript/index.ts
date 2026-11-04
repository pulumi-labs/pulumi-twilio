import * as pulumi from "@pulumi/pulumi";
import * as twilio from "@pulumi-labs/twilio";

// Credentials come from `pulumi config set twilio:accountSid ...` / `twilio:authToken --secret`
// or the TWILIO_ACCOUNT_SID / TWILIO_AUTH_TOKEN environment variables.
const config = new pulumi.Config();
const voiceUrl = config.get("voiceUrl") ?? "https://example.com/webhooks/voice";
const smsUrl = config.get("smsUrl") ?? "https://example.com/webhooks/sms";

const app = new twilio.Application("app", {
    friendlyName: "pulumi-messaging",
    voiceUrl: voiceUrl,
    voiceMethod: "POST",
    smsUrl: smsUrl,
    smsMethod: "POST",
});

const svc = new twilio.MessagingService("svc", {
    friendlyName: "pulumi-alerts",
    inboundRequestUrl: smsUrl,
    inboundMethod: "POST",
    stickySender: true,
});

export const applicationSid = app.sid;
export const messagingServiceSid = svc.sid;
