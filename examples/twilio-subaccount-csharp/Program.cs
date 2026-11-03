// Credentials come from config (`pulumi config set twilio:accountSid ...` and `pulumi config set --secret twilio:authToken ...`) or the TWILIO_ACCOUNT_SID / TWILIO_AUTH_TOKEN environment variables.
using System.Text.Json;
using Pulumi;
using Pulumi.Twilio;

return await Deployment.RunAsync(() =>
{
    var config = new Pulumi.Config();
    var tenantName = config.Get("tenantName") ?? "pulumi-tenant";

    // Destroying the stack CLOSES the subaccount (permanent after 30 days); set status: suspended to pause it instead.
    var tenant = new Subaccount("tenant", new SubaccountArgs
    {
        FriendlyName = tenantName,
    });

    // A restricted API key scoped to the tenant that may only manage Verify services.
    var ci = new ApiKey("ci", new ApiKeyArgs
    {
        FriendlyName = "pulumi-ci",
        AccountSid = tenant.Sid,
        KeyType = "restricted",
        Policy = JsonSerializer.Serialize(new { allow = new[] { "/v2/Services/**" } }),
    });

    var verify = new VerifyService("verify", new VerifyServiceArgs
    {
        FriendlyName = "Pulumi Demo",
        CodeLength = 6,
    });

    var account = GetAccount.Invoke(new GetAccountInvokeArgs { AccountSid = tenant.Sid });

    return new Dictionary<string, object?>
    {
        ["tenantSid"] = tenant.Sid,
        ["keySid"] = ci.Sid,
        ["keySecret"] = ci.Secret,
        ["verifyServiceSid"] = verify.Sid,
        ["tenantStatus"] = account.Apply(a => a.Status),
    };
});
