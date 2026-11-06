// Copyright 2026, Pulumi Corporation.
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

// Package provider implements the Twilio Pulumi provider.
package provider

import (
	"github.com/pulumi/pulumi-go-provider"
	"github.com/pulumi/pulumi-go-provider/infer"
	"github.com/pulumi/pulumi/sdk/v3/go/common/tokens"
)

// New builds the Twilio provider.
func newProvider() (provider.Provider, error) {
	return infer.NewProviderBuilder().
		WithDisplayName("Twilio").
		WithDescription(
			"Experimental: manage Twilio subaccounts, API keys, phone-number webhooks, TwiML applications, addresses, signing keys, call queues, usage triggers, SIP credential lists, IP access control lists and domains, short codes, Connect apps, verified caller IDs, Event Streams sinks and subscriptions, Elastic SIP trunks, messaging services and Verify services as infrastructure with Pulumi.",
		).
		WithHomepage("https://www.twilio.com").
		WithRepository("https://github.com/pulumi-labs/pulumi-twilio").
		WithPublisher("Pulumi Labs").
		WithLanguageMap(map[string]any{
			"go": map[string]any{"generateResourceContainerTypes": true, "respectSchemaVersion": true},
			"nodejs": map[string]any{
				"packageName":          "@pulumi-labs/twilio",
				"packageDescription":   "Experimental: manage Twilio subaccounts, API keys, phone-number webhooks, TwiML applications, addresses, signing keys, call queues, usage triggers, SIP credential lists, IP access control lists and domains, short codes, Connect apps, verified caller IDs, Event Streams sinks and subscriptions, Elastic SIP trunks, messaging services and Verify services as infrastructure with Pulumi.",
				"respectSchemaVersion": true,
			},
			"python": map[string]any{
				"packageName":          "pulumi_labs_twilio",
				"respectSchemaVersion": true,
				"pyproject":            map[string]any{"enabled": true},
			},
			"csharp": map[string]any{"rootNamespace": "Pulumi.Labs", "respectSchemaVersion": true},
			"java":   map[string]any{"basePackage": "com.pulumi.labs", "respectSchemaVersion": true},
		}).
		WithGoImportPath("github.com/pulumi-labs/pulumi-twilio/sdk/go/twilio").
		WithPluginDownloadURL("github://api.github.com/pulumi-labs/pulumi-twilio").
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
			infer.Resource[*Address, AddressArgs, AddressState](&Address{}),
			infer.Resource[*SigningKey, SigningKeyArgs, SigningKeyState](&SigningKey{}),
			infer.Resource[*Queue, QueueArgs, QueueState](&Queue{}),
			infer.Resource[*UsageTrigger, UsageTriggerArgs, UsageTriggerState](&UsageTrigger{}),
			infer.Resource[*SipCredentialList, SipCredentialListArgs, SipCredentialListState](&SipCredentialList{}),
			infer.Resource[*SipCredential, SipCredentialArgs, SipCredentialState](&SipCredential{}),
			infer.Resource[*SipIpAccessControlList, SipIpAccessControlListArgs, SipIpAccessControlListState](&SipIpAccessControlList{}),
			infer.Resource[*SipIpAddress, SipIpAddressArgs, SipIpAddressState](&SipIpAddress{}),
			infer.Resource[*SipDomain, SipDomainArgs, SipDomainState](&SipDomain{}),
			infer.Resource[*SipDomainCredentialListMapping, SipDomainCredentialListMappingArgs, SipDomainCredentialListMappingState](&SipDomainCredentialListMapping{}),
			infer.Resource[*SipDomainIpAccessControlListMapping, SipDomainIpAccessControlListMappingArgs, SipDomainIpAccessControlListMappingState](&SipDomainIpAccessControlListMapping{}),
			infer.Resource[*ShortCode, ShortCodeArgs, ShortCodeState](&ShortCode{}),
			infer.Resource[*ConnectApp, ConnectAppArgs, ConnectAppState](&ConnectApp{}),
			infer.Resource[*OutgoingCallerId, OutgoingCallerIdArgs, OutgoingCallerIdState](&OutgoingCallerId{}),
			infer.Resource[*EventSink, EventSinkArgs, EventSinkState](&EventSink{}),
			infer.Resource[*EventSubscription, EventSubscriptionArgs, EventSubscriptionState](&EventSubscription{}),
			infer.Resource[*Trunk, TrunkArgs, TrunkState](&Trunk{}),
			infer.Resource[*TrunkOriginationUrl, TrunkOriginationUrlArgs, TrunkOriginationUrlState](&TrunkOriginationUrl{}),
			infer.Resource[*TrunkCredentialList, TrunkCredentialListArgs, TrunkCredentialListState](&TrunkCredentialList{}),
			infer.Resource[*TrunkIpAccessControlList, TrunkIpAccessControlListArgs, TrunkIpAccessControlListState](&TrunkIpAccessControlList{}),
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
			infer.Function[*GetAddress, GetAddressArgs, GetAddressResult](&GetAddress{}),
			infer.Function[*GetSigningKey, GetSigningKeyArgs, GetSigningKeyResult](&GetSigningKey{}),
			infer.Function[*GetQueue, GetQueueArgs, GetQueueResult](&GetQueue{}),
			infer.Function[*GetUsageTrigger, GetUsageTriggerArgs, GetUsageTriggerResult](&GetUsageTrigger{}),
			infer.Function[*GetSipCredentialList, GetSipCredentialListArgs, GetSipCredentialListResult](&GetSipCredentialList{}),
			infer.Function[*GetSipCredential, GetSipCredentialArgs, GetSipCredentialResult](&GetSipCredential{}),
			infer.Function[*GetSipIpAccessControlList, GetSipIpAccessControlListArgs, GetSipIpAccessControlListResult](&GetSipIpAccessControlList{}),
			infer.Function[*GetSipIpAddress, GetSipIpAddressArgs, GetSipIpAddressResult](&GetSipIpAddress{}),
			infer.Function[*GetSipDomain, GetSipDomainArgs, GetSipDomainResult](&GetSipDomain{}),
			infer.Function[*GetSipDomainCredentialListMapping, GetSipDomainCredentialListMappingArgs, GetSipDomainCredentialListMappingResult](&GetSipDomainCredentialListMapping{}),
			infer.Function[*GetSipDomainIpAccessControlListMapping, GetSipDomainIpAccessControlListMappingArgs, GetSipDomainIpAccessControlListMappingResult](&GetSipDomainIpAccessControlListMapping{}),
			infer.Function[*GetShortCode, GetShortCodeArgs, GetShortCodeResult](&GetShortCode{}),
			infer.Function[*GetConnectApp, GetConnectAppArgs, GetConnectAppResult](&GetConnectApp{}),
			infer.Function[*GetOutgoingCallerId, GetOutgoingCallerIdArgs, GetOutgoingCallerIdResult](&GetOutgoingCallerId{}),
			infer.Function[*GetEventSink, GetEventSinkArgs, GetEventSinkResult](&GetEventSink{}),
			infer.Function[*GetEventSubscription, GetEventSubscriptionArgs, GetEventSubscriptionResult](&GetEventSubscription{}),
			infer.Function[*GetTrunk, GetTrunkArgs, GetTrunkResult](&GetTrunk{}),
			infer.Function[*GetTrunkOriginationUrl, GetTrunkOriginationUrlArgs, GetTrunkOriginationUrlResult](&GetTrunkOriginationUrl{}),
			infer.Function[*GetTrunkCredentialList, GetTrunkCredentialListArgs, GetTrunkCredentialListResult](&GetTrunkCredentialList{}),
			infer.Function[*GetTrunkIpAccessControlList, GetTrunkIpAccessControlListArgs, GetTrunkIpAccessControlListResult](&GetTrunkIpAccessControlList{}),
		).
		Build()
}
