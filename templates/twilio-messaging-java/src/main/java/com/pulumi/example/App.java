package com.pulumi.example;

import com.pulumi.Context;
import com.pulumi.Pulumi;
import com.pulumi.twilio.Application;
import com.pulumi.twilio.ApplicationArgs;
import com.pulumi.twilio.MessagingService;
import com.pulumi.twilio.MessagingServiceArgs;

public class App {
    public static void main(String[] args) {
        Pulumi.run(App::stack);
    }

    private static void stack(Context ctx) {
        // Credentials come from `pulumi config set twilio:accountSid ...` / `twilio:authToken --secret`
        // or the TWILIO_ACCOUNT_SID / TWILIO_AUTH_TOKEN environment variables.
        var config = ctx.config();
        var voiceUrl = config.get("voiceUrl").orElse("https://example.com/webhooks/voice");
        var smsUrl = config.get("smsUrl").orElse("https://example.com/webhooks/sms");

        var app = new Application("app", ApplicationArgs.builder()
            .friendlyName("pulumi-messaging")
            .voiceUrl(voiceUrl)
            .voiceMethod("POST")
            .smsUrl(smsUrl)
            .smsMethod("POST")
            .build());

        var svc = new MessagingService("svc", MessagingServiceArgs.builder()
            .friendlyName("pulumi-alerts")
            .inboundRequestUrl(smsUrl)
            .inboundMethod("POST")
            .stickySender(true)
            .build());

        ctx.export("applicationSid", app.sid());
        ctx.export("messagingServiceSid", svc.sid());
    }
}
