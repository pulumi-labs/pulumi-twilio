import pulumi
import pulumi_labs_twilio as twilio

# Credentials come from `pulumi config set twilio:accountSid ...` / `twilio:authToken --secret`
# or the TWILIO_ACCOUNT_SID / TWILIO_AUTH_TOKEN environment variables.
config = pulumi.Config()
voice_url = config.get("voiceUrl") or "https://example.com/webhooks/voice"
sms_url = config.get("smsUrl") or "https://example.com/webhooks/sms"

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
