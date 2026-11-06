// Credentials come from config (`pulumi config set twilio:accountSid ...` and `pulumi config set --secret twilio:authToken ...`) or the TWILIO_ACCOUNT_SID / TWILIO_AUTH_TOKEN environment variables.
import * as pulumi from "@pulumi/pulumi";
import * as twilio from "@pulumi-labs/twilio";

const config = new pulumi.Config();
const tenantName = config.get("tenantName") ?? "pulumi-tenant";

// Destroying the stack CLOSES the subaccount (permanent after 30 days); set status: suspended to pause it instead.
const tenant = new twilio.Subaccount("tenant", {
    friendlyName: tenantName,
});

// A restricted API key scoped to the tenant that may only manage Verify services.
const ci = new twilio.ApiKey("ci", {
    friendlyName: "pulumi-ci",
    accountSid: tenant.sid,
    keyType: "restricted",
    policy: JSON.stringify({ allow: ["/twilio/iam/api-keys/create"] }),
});

const verify = new twilio.VerifyService("verify", {
    friendlyName: "Pulumi Demo",
    codeLength: 6,
});

const account = twilio.getAccountOutput({ accountSid: tenant.sid });

export const tenantSid = tenant.sid;
export const keySid = ci.sid;
export const keySecret = ci.secret;
export const verifyServiceSid = verify.sid;
export const tenantStatus = account.status;
