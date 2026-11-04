// Credentials come from config (`pulumi config set twilio:accountSid ...` and `pulumi config set --secret twilio:authToken ...`) or the TWILIO_ACCOUNT_SID / TWILIO_AUTH_TOKEN environment variables.
package main

import (
	"encoding/json"

	"github.com/pulumi-labs/pulumi-twilio/sdk/go/twilio"
	"github.com/pulumi/pulumi/sdk/v3/go/pulumi"
	"github.com/pulumi/pulumi/sdk/v3/go/pulumi/config"
)

func main() {
	pulumi.Run(func(ctx *pulumi.Context) error {
		cfg := config.New(ctx, "")
		tenantName := cfg.Get("tenantName")
		if tenantName == "" {
			tenantName = "pulumi-tenant"
		}

		// Destroying the stack CLOSES the subaccount (permanent after 30 days); set status: suspended to pause it instead.
		tenant, err := twilio.NewSubaccount(ctx, "tenant", &twilio.SubaccountArgs{
			FriendlyName: pulumi.String(tenantName),
		})
		if err != nil {
			return err
		}

		// A restricted API key scoped to the tenant that may only manage Verify services.
		policy, err := json.Marshal(map[string]interface{}{
			"allow": []string{"/v2/Services/**"},
		})
		if err != nil {
			return err
		}
		ci, err := twilio.NewApiKey(ctx, "ci", &twilio.ApiKeyArgs{
			FriendlyName: pulumi.String("pulumi-ci"),
			AccountSid:   tenant.Sid,
			KeyType:      pulumi.String("restricted"),
			Policy:       pulumi.String(string(policy)),
		})
		if err != nil {
			return err
		}

		verify, err := twilio.NewVerifyService(ctx, "verify", &twilio.VerifyServiceArgs{
			FriendlyName: pulumi.String("Pulumi Demo"),
			CodeLength:   pulumi.Int(6),
		})
		if err != nil {
			return err
		}

		account := twilio.GetAccountOutput(ctx, twilio.GetAccountOutputArgs{
			AccountSid: tenant.Sid,
		})

		ctx.Export("tenantSid", tenant.Sid)
		ctx.Export("keySid", ci.Sid)
		ctx.Export("keySecret", ci.Secret)
		ctx.Export("verifyServiceSid", verify.Sid)
		ctx.Export("tenantStatus", account.Status())
		return nil
	})
}
