# Credentials come from config (`pulumi config set twilio:accountSid ...` and `pulumi config set --secret twilio:authToken ...`) or the TWILIO_ACCOUNT_SID / TWILIO_AUTH_TOKEN environment variables.
import json

import pulumi
import pulumi_twilio as twilio

config = pulumi.Config()
tenant_name = config.get("tenantName") or "pulumi-tenant"

# Destroying the stack CLOSES the subaccount (permanent after 30 days); set status: suspended to pause it instead.
tenant = twilio.Subaccount("tenant", friendly_name=tenant_name)

# A restricted API key scoped to the tenant that may only manage Verify services.
ci = twilio.ApiKey(
    "ci",
    friendly_name="pulumi-ci",
    account_sid=tenant.sid,
    key_type="restricted",
    policy=json.dumps({"allow": ["/v2/Services/**"]}),
)

verify = twilio.VerifyService(
    "verify",
    friendly_name="Pulumi Demo",
    code_length=6,
)

account = twilio.get_account_output(account_sid=tenant.sid)

pulumi.export("tenantSid", tenant.sid)
pulumi.export("keySid", ci.sid)
pulumi.export("keySecret", ci.secret)
pulumi.export("verifyServiceSid", verify.sid)
pulumi.export("tenantStatus", account.status)
