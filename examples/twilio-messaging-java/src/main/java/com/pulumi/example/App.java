package com.pulumi.example;

// Credentials come from config (`pulumi config set twilio:accountSid ...` and `pulumi config set --secret twilio:authToken ...`) or the TWILIO_ACCOUNT_SID / TWILIO_AUTH_TOKEN environment variables.
import com.pulumi.Context;
import com.pulumi.Pulumi;
import com.pulumi.labs.twilio.Application;
import com.pulumi.labs.twilio.ApplicationArgs;
import com.pulumi.labs.twilio.IncomingPhoneNumber;
import com.pulumi.labs.twilio.IncomingPhoneNumberArgs;
import com.pulumi.labs.twilio.MessagingService;
import com.pulumi.labs.twilio.MessagingServiceArgs;
import com.pulumi.labs.twilio.MessagingServicePhoneNumber;
import com.pulumi.labs.twilio.MessagingServicePhoneNumberArgs;

public class App {
    public static void main(String[] args) {
        Pulumi.run(App::stack);
    }

    private static void stack(Context ctx) {
        var config = ctx.config();
        var smsUrl = config.get("smsUrl").orElse("https://example.com/webhooks/sms");
        var voiceUrl = config.get("voiceUrl").orElse("https://example.com/webhooks/voice");
        var phoneNumber = config.get("phoneNumber"); // E.164 number the account already owns (optional)

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

        if (phoneNumber.isPresent()) {
            // Adopts a number the account already owns: this never purchases a number, and destroying the stack does not release it.
            var number = new IncomingPhoneNumber("number", IncomingPhoneNumberArgs.builder()
                .phoneNumber(phoneNumber.get())
                .voiceApplicationSid(app.sid())
                .build());
            new MessagingServicePhoneNumber("sender", MessagingServicePhoneNumberArgs.builder()
                .serviceSid(svc.sid())
                .phoneNumberSid(number.sid())
                .build());
            ctx.export("numberSid", number.sid());
        }
    }
}
