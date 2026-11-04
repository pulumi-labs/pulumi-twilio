// Credentials come from config (`pulumi config set twilio:accountSid ...` and `pulumi config set --secret twilio:authToken ...`) or the TWILIO_ACCOUNT_SID / TWILIO_AUTH_TOKEN environment variables.
using Pulumi;
using Pulumi.Labs.Twilio;

return await Deployment.RunAsync(() =>
{
    var config = new Pulumi.Config();
    var smsUrl = config.Get("smsUrl") ?? "https://example.com/webhooks/sms";
    var voiceUrl = config.Get("voiceUrl") ?? "https://example.com/webhooks/voice";
    var phoneNumber = config.Get("phoneNumber"); // E.164 number the account already owns (optional)

    var app = new Application("app", new ApplicationArgs
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

    var outputs = new Dictionary<string, object?>
    {
        ["applicationSid"] = app.Sid,
        ["messagingServiceSid"] = svc.Sid,
    };

    if (!string.IsNullOrEmpty(phoneNumber))
    {
        // Adopts a number the account already owns: this never purchases a number, and destroying the stack does not release it.
        var number = new IncomingPhoneNumber("number", new IncomingPhoneNumberArgs
        {
            PhoneNumber = phoneNumber,
            VoiceApplicationSid = app.Sid,
        });
        new MessagingServicePhoneNumber("sender", new MessagingServicePhoneNumberArgs
        {
            ServiceSid = svc.Sid,
            PhoneNumberSid = number.Sid,
        });
        outputs["numberSid"] = number.Sid;
    }

    return outputs;
});
