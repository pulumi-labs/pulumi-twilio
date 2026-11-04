// Package provider implements the Twilio Pulumi provider.
package provider

import (
	"github.com/pulumi/pulumi-go-provider"
	"github.com/pulumi/pulumi-go-provider/infer"
	"github.com/pulumi/pulumi/sdk/v3/go/common/tokens"
)

// New builds the Twilio provider.
func New() (provider.Provider, error) {
	return infer.NewProviderBuilder().
		WithDisplayName("Twilio").
		WithDescription(
			"Manage Twilio subaccounts, API keys, phone-number webhooks, TwiML applications, messaging services and Verify services as infrastructure with Pulumi.",
		).
		WithHomepage("https://www.twilio.com").
		WithRepository("https://github.com/pulumi/autoregistry").
		WithGoImportPath("github.com/pulumi/pulumi-twilio/sdk/go/twilio").
		WithPublisher("Pulumi").
		WithLicense("Apache-2.0").
		WithKeywords("pulumi", "twilio", "communications", "sms", "voice", "category/network").
		WithModuleMap(map[tokens.ModuleName]tokens.ModuleName{
			"provider": "index",
		}).
		WithConfig(infer.Config(Config{})).
		WithResources(
			infer.Resource[*Subaccount, SubaccountArgs, SubaccountState](&Subaccount{}),
			infer.Resource[*ApiKey, ApiKeyArgs, ApiKeyState](&ApiKey{}),
			infer.Resource[*IncomingPhoneNumber, IncomingPhoneNumberArgs, IncomingPhoneNumberState](&IncomingPhoneNumber{}),
			infer.Resource[*Application, ApplicationArgs, ApplicationState](&Application{}),
			infer.Resource[*MessagingService, MessagingServiceArgs, MessagingServiceState](&MessagingService{}),
			infer.Resource[*MessagingServicePhoneNumber, MessagingServicePhoneNumberArgs, MessagingServicePhoneNumberState](&MessagingServicePhoneNumber{}),
			infer.Resource[*VerifyService, VerifyServiceArgs, VerifyServiceState](&VerifyService{}),
		).
		WithFunctions(
			infer.Function[*GetAccount, GetAccountArgs, GetAccountResult](&GetAccount{}),
			infer.Function[*GetIncomingPhoneNumber, GetIncomingPhoneNumberArgs, GetIncomingPhoneNumberResult](&GetIncomingPhoneNumber{}),
			infer.Function[*GetSubaccount, GetSubaccountArgs, GetSubaccountResult](&GetSubaccount{}),
			infer.Function[*GetApiKey, GetApiKeyArgs, GetApiKeyResult](&GetApiKey{}),
			infer.Function[*GetApplication, GetApplicationArgs, GetApplicationResult](&GetApplication{}),
			infer.Function[*GetMessagingService, GetMessagingServiceArgs, GetMessagingServiceResult](&GetMessagingService{}),
			infer.Function[*GetMessagingServicePhoneNumber, GetMessagingServicePhoneNumberArgs, GetMessagingServicePhoneNumberResult](&GetMessagingServicePhoneNumber{}),
			infer.Function[*GetVerifyService, GetVerifyServiceArgs, GetVerifyServiceResult](&GetVerifyService{}),
			infer.Function[*GetBalance, GetBalanceArgs, GetBalanceResult](&GetBalance{}),
		).
		Build()
}
