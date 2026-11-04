# Credentials come from config (`pulumi config set twilio:accountSid ...` and `pulumi config set --secret twilio:authToken ...`) or the TWILIO_ACCOUNT_SID / TWILIO_AUTH_TOKEN environment variables.
import pulumi
import pulumi_labs_twilio as twilio

config = pulumi.Config()
sms_url = config.get("smsUrl") or "https://example.com/webhooks/sms"
voice_url = config.get("voiceUrl") or "https://example.com/webhooks/voice"
phone_number = config.get("phoneNumber")  # E.164 number the account already owns (optional)

app = twilio.Application(
    "app",
    friendly_name="pulumi-messaging",
    voice_url=voice_url,
    voice_method="POST",
    sms_url=sms_url,
    sms_method="POST",
)

svc = twilio.MessagingService(
    "svc",
    friendly_name="pulumi-alerts",
    inbound_request_url=sms_url,
    inbound_method="POST",
    sticky_sender=True,
)

pulumi.export("applicationSid", app.sid)
pulumi.export("messagingServiceSid", svc.sid)

if phone_number:
    # Adopts a number the account already owns: this never purchases a number, and destroying the stack does not release it.
    number = twilio.IncomingPhoneNumber(
        "number",
        phone_number=phone_number,
        voice_application_sid=app.sid,
    )
    twilio.MessagingServicePhoneNumber(
        "sender",
        service_sid=svc.sid,
        phone_number_sid=number.sid,
    )
    pulumi.export("numberSid", number.sid)
