# Twilio Pulumi Provider

> [!NOTE]
> **Experimental.** This provider is experimental, versions are `0.x`, and its API may change without notice.

A native Pulumi provider for [Twilio](https://www.twilio.com). There is no maintained Terraform provider for Twilio (the official `twilio/twilio` provider is labelled pilot and has not been released since June 2024), so this provider calls the Twilio REST API directly and manages the account-level building blocks of a Twilio estate as infrastructure: subaccounts, API keys, signing keys, phone-number routing, TwiML applications, addresses, call queues, usage triggers, SIP credential lists, IP access control lists and domains, messaging services and Verify services.

## Installation

```bash
# TypeScript / JavaScript
npm install @pulumi-labs/twilio

# Python
pip install pulumi-labs-twilio

# Go
go get github.com/pulumi-labs/pulumi-twilio/sdk/go/twilio

# .NET
dotnet add package Pulumi.Labs.Twilio
```

Java: `com.pulumi.labs:twilio` on Maven Central.

## Resources

| Token | What it manages |
| --- | --- |
| `twilio:index:Subaccount` | A subaccount of the authenticating account. **Destroy closes it**; set `status: suspended` to pause one reversibly. Exposes the subaccount `authToken` as a secret. |
| `twilio:index:ApiKey` | A standard or restricted API key (`keyType`, `policy`). The `secret` is returned by Twilio once and stored as a secret. |
| `twilio:index:IncomingPhoneNumber` | Voice and SMS routing for a number the account already owns, adopted by `phoneNumber` or `phoneNumberSid`. Never buys a number; destroy never releases it. |
| `twilio:index:Application` | A TwiML application: voice and SMS webhooks a number can point at. |
| `twilio:index:Address` | A customer postal address (regulatory and emergency calling). Twilio auto-corrects it unless `autoCorrectAddress` is false; `isoCountry` replaces. Delete fails while a number or bundle uses it. |
| `twilio:index:SigningKey` | A signing key for Access Tokens. The `secret` is returned once and stored as a secret. |
| `twilio:index:Queue` | A voice call queue (`friendlyName`, `maxSize`); the live `currentSize` and `averageWaitTime` are outputs. |
| `twilio:index:ShortCode` | SMS webhooks (`smsUrl`, `smsMethod`, `smsFallbackUrl`, `smsFallbackMethod`), `friendlyName` and `apiVersion` of a short code the account already leases. Identity: `code` or `shortCodeSid` (replaces). Adopt-only: Twilio provisions short codes through support, and destroy never releases one. |
| `twilio:index:ConnectApp` | Listing details of a Connect app created in the Console (`friendlyName`, `companyName`, `description`, `homepageUrl`, `authorizeRedirectUrl`, `deauthorizeCallbackUrl` and method, `permissions`). Identity: `connectAppSid` (replaces). Adopt-only: destroy never deletes the app. |
| `twilio:index:OutgoingCallerId` | The `friendlyName` of a number the account has already verified as a caller ID. Identity: `phoneNumber` or `outgoingCallerIdSid` (replaces). Adopt-only: Twilio verifies a number by phoning it, so the resource never starts a verification and destroy never removes the caller ID. |
| `twilio:index:EventSink` | An Event Streams sink (`webhook`, `kinesis` or `segment` destination). Only `description` updates in place; the type and connection settings replace. `segmentWriteKey` and `kinesisExternalId` are secrets. A sink must be active (validated through Twilio's sink test and validate calls) before a subscription can use it. |
| `twilio:index:Trunk` | An Elastic SIP trunk: `domainName` (`<name>.pstn.twilio.com`), `friendlyName`, disaster recovery URL and method, `transferMode`, `transferCallerId`, `secure`, `cnamLookupEnabled` and the call recording settings `recordingMode` and `recordingTrim` (set on Twilio's Recording sub-resource). Everything updates in place and the `effective` output reports what Twilio holds, including the assigned `domainName`. Phone numbers attach through `IncomingPhoneNumber.trunkSid`. |
| `twilio:index:TrunkOriginationUrl` | A SIP address (`sipUrl`, `sip:` only) Twilio routes the trunk's inbound calls to, with `weight`, `priority` and `enabled`. Updates in place; `trunkSid` replaces. ID `<trunkSid>/<sid>`. |
| `twilio:index:TrunkCredentialList` and `twilio:index:TrunkIpAccessControlList` | Attach a `SipCredentialList` or `SipIpAccessControlList` to a trunk. Twilio has no update call, so every input replaces; destroy detaches the list without deleting it. ID `<trunkSid>/<listSid>`. |
| `twilio:index:EventSubscription` | The event types (`types`, each with an optional `schemaVersion`) delivered to an `EventSink`. Types change in place; `sinkSid` replaces. |
| `twilio:index:SipCredentialList` | A SIP credential list (`friendlyName`). Add credentials with `SipCredential`. |
| `twilio:index:SipCredential` | A username and password in a credential list. The `password` is a write-only secret (Twilio never returns it) and updates in place; `username` and `credentialListSid` replace. ID `<credentialListSid>/<sid>`. |
| `twilio:index:SipIpAccessControlList` | A SIP IP access control list (`friendlyName`). Add addresses with `SipIpAddress`. |
| `twilio:index:SipIpAddress` | An IPv4 address or range (`ipAddress`, `cidrPrefixLength`, default 32) in an access control list. ID `<ipAccessControlListSid>/<sid>`. |
| `twilio:index:SipDomain` | A programmable SIP domain (`<name>.sip.twilio.com`) with voice webhooks, `sipRegistration`, `secure`, emergency calling and BYOC trunk settings. `authType` is read-only: it follows the mappings below. |
| `twilio:index:SipDomainCredentialListMapping` | Maps a credential list to a domain for `authScope` `calls` (default) or `registrations`. Twilio has no update call, so every input (`domainSid`, `credentialListSid`, `authScope`, `accountSid`) replaces. ID `<domainSid>/<authScope>/<sid>`; the `sid` is the credential list's SID. |
| `twilio:index:SipDomainIpAccessControlListMapping` | Maps an IP access control list to a domain for calls. Every input replaces. ID `<domainSid>/<sid>`; the `sid` is the list's SID. |
| `twilio:index:UsageTrigger` | A webhook fired when usage in a category reaches a value. Only `callbackUrl`, `callbackMethod` and `friendlyName` update in place. |
| `twilio:index:MessagingService` | A Messaging Service (sender pool) with inbound and status webhooks, sticky sender, smart encoding, validity period and more. |
| `twilio:index:MessagingServicePhoneNumber` | Membership of an owned number in a Messaging Service sender pool. |
| `twilio:index:VerifyService` | A Verify service for one-time passcodes (code length, lookup, PSD2, default template, and the push, TOTP, WhatsApp and passkeys settings as flat prefixed fields such as `totpIssuer` and `passkeysRelyingPartyId`). |
| `twilio:index:VerifyRateLimit` and `twilio:index:VerifyRateLimitBucket` | A rate limit of a Verify service (`uniqueName` replaces, `description` updates in place) and its buckets (`max` requests per `interval` seconds, both update in place). IDs `<serviceSid>/<sid>` (`RK...`) and `<serviceSid>/<rateLimitSid>/<sid>` (`BL...`). |
| `twilio:index:VerifyMessagingConfiguration` | The Messaging Service (`messagingServiceSid`, updates in place) a Verify service sends SMS through for one `country` (an ISO code, or `all`; replaces). ID `<serviceSid>/<country>`. |
| `twilio:index:VerifyWebhook` | A webhook of a Verify service: `friendlyName`, `eventTypes`, `webhookUrl`, `status` (`enabled` default) and `webhookVersion` (`v2` default; named so because `version` is reserved). Updates in place; `serviceSid` replaces. ID `<serviceSid>/<sid>` (`YW...`). |
| `twilio:index:ServerlessService` | A Twilio Serverless service (`uniqueName` replaces; `friendlyName`, `includeCredentials`, `uiEditable` update in place). Deleting it deletes everything inside it. ID is the SID (`ZS...`). |
| `twilio:index:ServerlessEnvironment` | An environment of a service (`uniqueName`, optional `domainSuffix`). Twilio has no update call, so every input replaces. ID `<serviceSid>/<sid>` (`ZE...`). |
| `twilio:index:ServerlessVariable` | An environment variable (`key` and `value` update in place; `value` is a Pulumi secret). ID `<serviceSid>/<environmentSid>/<sid>` (`ZV...`). |
| `twilio:index:ServerlessFunction` and `twilio:index:ServerlessAsset` | The container of a Function or Asset (`friendlyName`). The code or file is uploaded as a version through Twilio's upload host and is not managed here. ID `<serviceSid>/<sid>` (`ZH...`). |
| `twilio:index:ByocTrunk` | A BYOC (Bring Your Own Carrier) trunk on the Voice API: `friendlyName`, the voice, fallback and status callback URLs and methods, `cnamLookupEnabled`, `connectionPolicySid` and `fromDomainSid`. Everything updates in place. ID is the SID (`BY...`). |
| `twilio:index:ConnectionPolicy` and `twilio:index:ConnectionPolicyTarget` | The carrier targets a BYOC trunk sends outbound calls to. A policy has a `friendlyName`; a target has a `sip:` `target`, `priority` (default 10), `weight` (default 10) and `enabled` (default true), all updating in place, and `connectionPolicySid` replaces. IDs `NY...` and `<connectionPolicySid>/<sid>` (`NE...`). |
| `twilio:index:IpRecord` and `twilio:index:SourceIpMapping` | The carrier's IPv4 address or CIDR block (`ipAddress` and `cidrPrefixLength` replace, `friendlyName` updates in place) and its mapping to a `SipDomain` for inbound calls (`ipRecordSid` replaces, `sipDomainSid` updates in place). IDs `IL...` and `IB...`. |
| `twilio:index:IncomingPhoneNumberAssignedAddOn` | An installed Marketplace add-on assigned to an owned number (`phoneNumberSid`, `installedAddOnSid`, optional `accountSid`). Twilio has no update call, so every input replaces. ID `<phoneNumberSid>/<sid>` (`XE...`); Twilio never returns `installedAddOnSid`, so an import has none until the program sets it, which plans an in-place update that only records it (no Twilio call). Lookup `getIncomingPhoneNumberAssignedAddOn` (`phoneNumberSid`, `assignedAddOnSid`) also lists the add-on's read-only extensions. |
| `twilio:index:VerifySafeListNumber` | A phone number (`phoneNumber`, E.164) on the account's Verify SafeList. Twilio has no update call, so a change replaces it. The SafeList is account-wide, not per service. ID is the phone number. |

Data sources: `twilio:index:getAccount` (the provider account or any subaccount by `accountSid`; the auth token is deliberately not returned) and `twilio:index:getIncomingPhoneNumber` (by `phoneNumber` or `sid`).

Read-only catalogue and reporting lookups: `getAvailablePhoneNumbers` (search purchasable numbers by `countryCode`, `numberType` local, tollFree or mobile, and Twilio's filters; searching buys nothing), `getUsageRecords` (usage by `period`, `category` and date range), `getAuthorizedConnectApps` and `getAuthorizedConnectApp` (Connect Apps the account authorized), `getEventTypes`, `getEventType` and `getEventSchema` (the Event Streams catalogue; use a returned `type` in `EventSubscription.types`). List lookups follow Twilio's paging and fail with an error if a list runs past 100 pages instead of returning a truncated result.

Lookup functions, one per resource: `getSubaccount` (`subaccountSid`), `getApiKey` (`keySid`, optional `accountSid`; the secret is never returned), `getApplication` (`applicationSid`, optional `accountSid`), `getAddress` (`addressSid`), `getSigningKey` (`keySid`; no secret), `getQueue` (`queueSid`), `getUsageTrigger` (`triggerSid`), `getSipCredentialList` (`credentialListSid`), `getSipCredential` (`credentialListSid`, `credentialSid`; no password), `getSipIpAccessControlList` (`ipAccessControlListSid`), `getSipIpAddress` (`ipAccessControlListSid`, `ipAddressSid`) and `getSipDomainCredentialListMapping` (`domainSid`, `credentialListSid`, optional `authScope`), `getSipDomainIpAccessControlListMapping` (`domainSid`, `ipAccessControlListSid`), `getSipDomain` (`domainSid`), `getShortCode` (`code` or `sid`), `getConnectApp` (`sid`), `getOutgoingCallerId` (`phoneNumber` or `sid`), each with optional `accountSid`, `getEventSink` and `getEventSubscription` (`sid`; the sink lookup never returns the secret settings), `getTrunk` (`trunkSid`), `getTrunkOriginationUrl` (`trunkSid`, `originationUrlSid`), `getTrunkCredentialList` (`trunkSid`, `credentialListSid`), `getTrunkIpAccessControlList` (`trunkSid`, `ipAccessControlListSid`), `getMessagingService` and `getVerifyService` (`serviceSid`), `getVerifyRateLimit` (`serviceSid`, `sid`), `getVerifyRateLimitBucket` (`serviceSid`, `rateLimitSid`, `sid`), `getVerifyMessagingConfiguration` (`serviceSid`, `country`), `getVerifyWebhook` (`serviceSid`, `sid`), `getVerifySafeListNumber` (`phoneNumber`), `getVerifyTemplates` (lists the message templates, optional exact `friendlyName`; use a returned `sid` as `VerifyService.defaultTemplateSid`), `getMessagingServicePhoneNumber` (`serviceSid`, `phoneNumberSid`) and `getBalance` (optional `accountSid`). Each fails with "... was not found" for an object that does not exist. The full list of what Twilio's SDKs offer and what is deferred is in `research/coverage/twilio.md`.

`IncomingPhoneNumber`, `Application`, `ApiKey`, `Address`, `SigningKey`, `Queue`, `UsageTrigger` and the five SIP resources accept an optional `accountSid` to act inside a subaccount with the parent's credentials. Changing it replaces everything but an `IncomingPhoneNumber`; for an `IncomingPhoneNumber` it transfers the number to the other account in place (Twilio's `AccountSid` update parameter, posted to the account that currently owns the number).

EventSink (`DG...`) and EventSubscription (`DF...`) import by SID; the Events API is not account-scoped. ShortCode, ConnectApp and OutgoingCallerId import by bare SID (`SC...`, `CN...`, `PN...`), or `<accountSid>/<sid>` for a subaccount. Their `accountSid` replaces, because Twilio cannot move these objects between accounts.

Elastic SIP Trunking (`trunking.twilio.com`) is not account-scoped: trunks belong to the account of the provider credentials, so these four resources have no `accountSid`. `Trunk` imports by SID (`TK...`), `TrunkOriginationUrl` as `<trunkSid>/<sid>` (`OU...`), `TrunkCredentialList` and `TrunkIpAccessControlList` as `<trunkSid>/<listSid>`. An imported trunk adopts only settings that differ from Twilio's defaults (`disable-all`, `from-transferee`, `do-not-record`, `do-not-trim`, `POST`, `false`), so a program that omits a setting or states its default plans no change.

SIP import IDs: `SipCredentialList`, `SipIpAccessControlList` and `SipDomain` import by bare SID (`<accountSid>/<sid>` for a subaccount). The two domain mappings live under a domain: `SipDomainCredentialListMapping` imports as `<domainSid>/<calls|registrations>/<sid>` and `SipDomainIpAccessControlListMapping` as `<domainSid>/<sid>` (each with an `<accountSid>/` prefix in a subaccount). `SipCredential` and `SipIpAddress` live under a list, so their ID is `<listSid>/<sid>` (`<accountSid>/<listSid>/<sid>` for a subaccount). An imported `SipCredential` has no `password` (Twilio never returns it); the first `pulumi up` with the password in the program updates it in place.

## Configuration

| Property | Env var | Notes |
| --- | --- | --- |
| `accountSid` | `TWILIO_ACCOUNT_SID` | Required. The account SID (`AC...`). |
| `authToken` | `TWILIO_AUTH_TOKEN` | Secret. The account auth token. |
| `apiKey` | `TWILIO_API_KEY` | Optional API key SID (`SK...`) used instead of the auth token. |
| `apiSecret` | `TWILIO_API_SECRET` | Secret. Secret of `apiKey`. |
| `region` | `TWILIO_REGION` | Optional data-residency region such as `au1` or `ie1`. |
| `edge` | `TWILIO_EDGE` | Optional edge location such as `sydney`; requires `region`. |
| `baseUrl` | `TWILIO_BASE_URL` | Testing aid: send every request to `<baseUrl>/<product>/<path>`. Must be `https` (loopback `http` is accepted). |

Authenticate with either the auth token, or an API key and secret (the account SID is still required because it appears in request paths). When both are set the API key is used. A restricted key cannot manage `Subaccount` or `ApiKey` resources; those need the account auth token.

```bash
pulumi config set twilio:accountSid <ACxxxxxxxx>
pulumi config set --secret twilio:authToken <token>
# or an API key instead of the auth token:
pulumi config set twilio:apiKey <SKxxxxxxxx>
pulumi config set --secret twilio:apiSecret <secret>
```

Or set the environment variables instead (config takes precedence when both are set):

```bash
export TWILIO_ACCOUNT_SID=<ACxxxxxxxx>
export TWILIO_AUTH_TOKEN=<token>
```

## Validation status

This provider has not been validated against a live Twilio account. Endpoints, form fields and response shapes come from the Twilio OpenAPI definitions (twilio-oai v2.10.0) and the official `twilio-go` SDK (v1.32.0), and are exercised only against an in-process fake of the API in the unit tests. The fake follows the documented response shapes, enum rules and account scoping, but behaviour it cannot know is unverified: whether the IAM key route sees keys of a subaccount (the provider falls back to the account-explicit route when it does not), whether Twilio accepts every `AccountSid` number transfer. Try it on a trial or test account before relying on it.

## Outputs, refresh and import

Inputs hold only what the program wrote. `Application`, `IncomingPhoneNumber`, `MessagingService` and `VerifyService` also expose an `effective` output with the configuration Twilio currently reports, including fields the program does not set and changes made in the console, so server defaults never turn into a permanent diff. A field the program does set is refreshed into the input, so a change made outside Pulumi is detected.

Every resource can be imported. Objects in the provider's own account use the bare SID. Objects in a subaccount (every resource with an `accountSid`) use `<accountSid>/<sid>`, which also sets `accountSid` on import:

```bash
pulumi import twilio:index:Subaccount tenant ACxxxxxxxx
pulumi import twilio:index:ApiKey ci SKxxxxxxxx
pulumi import twilio:index:IncomingPhoneNumber main PNxxxxxxxx
pulumi import twilio:index:Application ivr ACsubxxxxxxxx/APxxxxxxxx   # an application in a subaccount
pulumi import twilio:index:MessagingService svc MGxxxxxxxx
pulumi import twilio:index:MessagingServicePhoneNumber sender MGxxxxxxxx/PNxxxxxxxx
pulumi import twilio:index:VerifyService verify VAxxxxxxxx
```

Import adopts every string, list and flag or number whose Twilio default the OpenAPI states, when the returned value differs from it, so a program that leaves the defaults out plans no change. For flags and numbers the OpenAPI states no default for, an import-only map of the values a fresh resource reports on the live API (for example `stickySender` true, `codeLength` 6, `psd2Enabled` false, `voiceCallerIdLookup` false) decides: a differing value is adopted, an equal one is not. A flag outside both maps (for example `lookupEnabled`) is not adopted; the `effective` output still reports it, and declaring it in the program afterwards is an ordinary in-place update. Resources created by earlier versions with a bare SID keep working.

## Destructive and billing semantics

* `Subaccount` destroy **closes** the subaccount. Twilio releases its phone numbers when it closes, and a closed subaccount cannot be reopened after 30 days. Use `status: suspended` to pause instead.
* `IncomingPhoneNumber` adopts a number the account already owns and fails if it is not owned. It does not purchase numbers (that is billed and out of scope for this release). Destroy only stops managing the number: it is not released and its current webhooks stay in place. Webhook fields you remove from the program are cleared on Twilio at the next update.
* A phone number belongs to at most one Messaging Service. `MessagingServicePhoneNumber` destroy removes the number from the pool without releasing it.
* `ApiKey.secret` is returned only by the create call, so it cannot be recovered or imported. Rotate by creating a new key, rolling consumers, then removing the old one. `keyType` and `accountSid` replace the key; `friendlyName` and `policy` update in place. The policy is read back on every refresh, so a restricted key widened in the console shows up as a diff and the next `pulumi up` restores it. A standard key is created without a `KeyType` parameter; `restricted` is sent only for restricted keys.
* Removing an optional input from the program resets that field on Twilio at the next update: URLs are cleared, HTTP methods go back to `POST`, and a boolean or number goes back to Twilio's default only where the Twilio OpenAPI definitions state one: `validityPeriod` (`36000`) and `doNotShareWarningEnabled` (`false`). Every other removed boolean or number (`stickySender`, `mmsConverter`, `smartEncoding`, `areaCodeGeomatch`, `useInboundWebhookOnNumber`, `voiceCallerIdLookup`, `codeLength`, `lookupEnabled`, `psd2Enabled` and so on) is left untouched on Twilio, because the API gives no documented value to reset it to; set it explicitly to change it. The friendly name of an `IncomingPhoneNumber` has no default and is left as it is. Import adopts every returned boolean and number without a documented default as an input. String fields that reference another object (`trunkSid`, `addressSid`, `identitySid`, `bundleSid`, `emergencyAddressSid`, the Verify `push*` credential SIDs and `whatsappMsgServiceSid`) are unassigned when removed, as are the Verify `totpIssuer`, `whatsappFrom` and `passkeys*` text fields and the origins list. Fields with no valid empty value and no documented default (`apiVersion`, `voiceReceiveMode`, `emergencyStatus`, `usecase`, `defaultTemplateSid`) are left as they are on Twilio when removed, and import does not adopt `apiVersion`. The Verify nested settings are read from Twilio's `push`, `totp`, `whatsapp` and `passkeys` response objects; the OpenAPI definitions leave those objects untyped, so the key names follow the request parameters.
* Destroying a `Subaccount` that is the provider's own account is refused, and the top-level account cannot be imported as a `Subaccount`.
* `Subaccount.authToken` is the primary auth token at the time of the last refresh and is stored as a secret.
* Webhook URLs are stored in state in plain text; if one embeds a token, treat it as sensitive and prefer request signature validation on the receiving side.
* Requests that Twilio throttles (HTTP 429) are retried with backoff; transient 5xx responses are retried for reads only. Creates are never retried after a transport error or 5xx because the first attempt may have been applied.

## Example

```yaml
resources:
  app:
    type: twilio:index:Application
    properties:
      friendlyName: support-ivr
      voiceUrl: https://example.com/webhooks/voice
      voiceMethod: POST
  svc:
    type: twilio:index:MessagingService
    properties:
      friendlyName: alerts
      inboundRequestUrl: https://example.com/webhooks/sms
      stickySender: true
  number:
    type: twilio:index:IncomingPhoneNumber
    properties:
      phoneNumber: "+14155550100"   # a number the account already owns
      voiceApplicationSid: ${app.sid}
  sender:
    type: twilio:index:MessagingServicePhoneNumber
    properties:
      serviceSid: ${svc.sid}
      phoneNumberSid: ${number.sid}
```

A Verify service's extras are scoped by the service: `VerifyRateLimit`, `VerifyMessagingConfiguration` and `VerifyWebhook` take `serviceSid` (`VA...`) and import as `<serviceSid>/<sid>`; `VerifyRateLimitBucket` as `<serviceSid>/<rateLimitSid>/<sid>`; `VerifyMessagingConfiguration` as `<serviceSid>/<country>`; `VerifySafeListNumber` by its E.164 number. An imported webhook adopts `status` and `webhookVersion` only when they differ from Twilio's defaults, and the order of `eventTypes` is not significant.

Serverless: `ServerlessService` imports by its `ZS...` SID, `ServerlessEnvironment`, `ServerlessFunction` and `ServerlessAsset` as `<serviceSid>/<sid>` and `ServerlessVariable` as `<serviceSid>/<environmentSid>/<sid>`. An imported service adopts the `includeCredentials` and `uiEditable` Twilio reports; a program that omits a flag keeps what Twilio holds. Lookups: `getServerlessService` (`serviceSid`), `getServerlessEnvironment` (`serviceSid`, `environmentSid`), `getServerlessVariable` (`serviceSid`, `environmentSid`, `variableSid`; the value is a secret), `getServerlessFunction` (`serviceSid`, `functionSid`) and `getServerlessAsset` (`serviceSid`, `assetSid`). Function and Asset versions, builds and deployments are not managed: deploy code with the Serverless Toolkit or the Twilio CLI.

BYOC (voice.twilio.com): `ByocTrunk`, `ConnectionPolicy` and `IpRecord` import by SID, `ConnectionPolicyTarget` as `<connectionPolicySid>/<sid>` and `SourceIpMapping` by SID. A target must be a `sip:` URI (`sips:` is not supported) and an IP record an IPv4 address; an imported target or record adopts `priority`, `weight`, `enabled` and `cidrPrefixLength` only when they differ from Twilio's defaults. Removing a trunk's callback URL, `connectionPolicySid` or `fromDomainSid` clears it on Twilio. Lookups: `getByocTrunk` (`byocTrunkSid`), `getConnectionPolicy` (`connectionPolicySid`), `getConnectionPolicyTarget` (`connectionPolicySid`, `targetSid`), `getIpRecord` (`ipRecordSid`) and `getSourceIpMapping` (`sourceIpMappingSid`). Dialing permissions and the recording and transcription configurations of the Voice API are not managed.

Examples in all six languages are in `examples/`, starter templates in `templates/`.

## Roadmap (not yet implemented)

* Buying numbers (purchase and release), which is billed and needs an explicit release policy. Searching is available through `getAvailablePhoneNumbers`.
* Studio flows, Flex, TaskRouter, Conversations and Messaging compliance (A2P 10DLC brands and campaigns).
* Account-level settings and usage triggers.

## Development

```bash
make tidy build test lint
make gen-sdk   # needs the pulumi CLI
```

The provider uses its own small REST client (form-encoded requests, JSON responses, HTTP Basic) instead of the `twilio-go` SDK, which keeps the module graph small and lets the tests point every product host at one fake server.
