package com.pulumi.example;

// Credentials come from config (`pulumi config set twilio:accountSid ...` and `pulumi config set --secret twilio:authToken ...`) or the TWILIO_ACCOUNT_SID / TWILIO_AUTH_TOKEN environment variables.
import com.pulumi.Context;
import com.pulumi.Pulumi;
import com.pulumi.twilio.ApiKey;
import com.pulumi.twilio.ApiKeyArgs;
import com.pulumi.twilio.Subaccount;
import com.pulumi.twilio.SubaccountArgs;
import com.pulumi.twilio.TwilioFunctions;
import com.pulumi.twilio.VerifyService;
import com.pulumi.twilio.VerifyServiceArgs;
import com.pulumi.twilio.inputs.GetAccountArgs;

public class App {
    public static void main(String[] args) {
        Pulumi.run(App::stack);
    }

    private static void stack(Context ctx) {
        var config = ctx.config();
        var tenantName = config.get("tenantName").orElse("pulumi-tenant");

        // Destroying the stack CLOSES the subaccount (permanent after 30 days); set status: suspended to pause it instead.
        var tenant = new Subaccount("tenant", SubaccountArgs.builder()
            .friendlyName(tenantName)
            .build());

        // A restricted API key scoped to the tenant that may only manage Verify services.
        var ci = new ApiKey("ci", ApiKeyArgs.builder()
            .friendlyName("pulumi-ci")
            .accountSid(tenant.sid())
            .keyType("restricted")
            .policy("{\"allow\":[\"/v2/Services/**\"]}")
            .build());

        var verify = new VerifyService("verify", VerifyServiceArgs.builder()
            .friendlyName("Pulumi Demo")
            .codeLength(6)
            .build());

        var account = tenant.sid().apply(sid ->
            TwilioFunctions.getAccount(GetAccountArgs.builder().accountSid(sid).build()));

        ctx.export("tenantSid", tenant.sid());
        ctx.export("keySid", ci.sid());
        ctx.export("keySecret", ci.secret());
        ctx.export("verifyServiceSid", verify.sid());
        ctx.export("tenantStatus", account.applyValue(r -> r.status()));
    }
}
