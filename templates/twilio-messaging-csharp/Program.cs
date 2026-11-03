using Pulumi;
using Pulumi.Twilio;

return await Deployment.RunAsync(() =>
{
    // Credentials come from `pulumi config set twilio:accountSid ...` / `twilio:authToken --secret`
    // or the TWILIO_ACCOUNT_SID / TWILIO_AUTH_TOKEN environment variables.
    var config = new Pulumi.Config();
    var voiceUrl = config.Get("voiceUrl") ?? "https://example.com/webhooks/voice";
    var smsUrl = config.Get("smsUrl") ?? "https://example.com/webhooks/sms";

    var app = new Pulumi.Twilio.Application("app", new ApplicationArgs
    {
        FriendlyName = "pulumi-messaging",
        VoiceUrl = voiceUrl,
        VoiceMethod = "POST",
        SmsUrl = smsUrl,
        SmsMethod = "POST",
    });

    var svc = new MessagingService("svc", new MessagingServiceArgs
    {
        FriendlyName = "pulumi-alerts",
        InboundRequestUrl = smsUrl,
        InboundMethod = "POST",
        StickySender = true,
    });

    return new Dictionary<string, object?>
    {
        ["applicationSid"] = app.Sid,
        ["messagingServiceSid"] = svc.Sid,
    };
});
