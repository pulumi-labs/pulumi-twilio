// Credentials come from config (`pulumi config set twilio:accountSid ...` and `pulumi config set --secret twilio:authToken ...`) or the TWILIO_ACCOUNT_SID / TWILIO_AUTH_TOKEN environment variables.
import * as pulumi from "@pulumi/pulumi";
import * as twilio from "@pulumi/twilio";

const config = new pulumi.Config();
const smsUrl = config.get("smsUrl") ?? "https://example.com/webhooks/sms";
const voiceUrl = config.get("voiceUrl") ?? "https://example.com/webhooks/voice";
const phoneNumber = config.get("phoneNumber"); // E.164 number the account already owns (optional)

const app = new twilio.Application("app", {
    friendlyName: "pulumi-messaging",
    voiceUrl,
    voiceMethod: "POST",
    smsUrl,
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

let numberSid: pulumi.Output<string> | undefined;
if (phoneNumber) {
    // Adopts a number the account already owns: this never purchases a number, and destroying the stack does not release it.
    const number = new twilio.IncomingPhoneNumber("number", {
        phoneNumber,
        voiceApplicationSid: app.sid,
    });
    new twilio.MessagingServicePhoneNumber("sender", {
        serviceSid: svc.sid,
        phoneNumberSid: number.sid,
    });
    numberSid = number.sid;
}
export { numberSid };
