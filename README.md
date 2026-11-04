# Twilio Pulumi Provider

A native Pulumi provider for [Twilio](https://www.twilio.com). There is no maintained Terraform provider for Twilio (the official `twilio/twilio` provider is labelled pilot and has not been released since June 2024), so this provider calls the Twilio REST API directly and manages the account-level building blocks of a Twilio estate as infrastructure: subaccounts, API keys, phone-number routing, TwiML applications, messaging services and Verify services.

## Resources

| Token | What it manages |
| --- | --- |
| `twilio:index:Subaccount` | A subaccount of the authenticating account. **Destroy closes it**; set `status: suspended` to pause one reversibly. Exposes the subaccount `authToken` as a secret. |
| `twilio:index:ApiKey` | A standard or restricted API key (`keyType`, `policy`). The `secret` is returned by Twilio once and stored as a secret. |
| `twilio:index:IncomingPhoneNumber` | Voice and SMS routing for a number the account already owns, adopted by `phoneNumber` or `phoneNumberSid`. Never buys a number; destroy never releases it. |
| `twilio:index:Application` | A TwiML application: voice and SMS webhooks a number can point at. |
| `twilio:index:MessagingService` | A Messaging Service (sender pool) with inbound and status webhooks, sticky sender, smart encoding, validity period and more. |
| `twilio:index:MessagingServicePhoneNumber` | Membership of an owned number in a Messaging Service sender pool. |
| `twilio:index:VerifyService` | A Verify service for one-time passcodes (code length, lookup, PSD2, and so on). |

Data sources: `twilio:index:getAccount` (the provider account or any subaccount by `accountSid`; the auth token is deliberately not returned) and `twilio:index:getIncomingPhoneNumber` (by `phoneNumber` or `sid`).

`IncomingPhoneNumber`, `Application` and `ApiKey` accept an optional `accountSid` to act inside a subaccount with the parent's credentials. Changing it replaces an `Application` or `ApiKey`; for an `IncomingPhoneNumber` it transfers the number to the other account in place (Twilio's `AccountSid` update parameter, posted to the account that currently owns the number).

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

Every resource can be imported. Objects in the provider's own account use the bare SID. Objects in a subaccount (`Application`, `IncomingPhoneNumber`, `ApiKey`) use `<accountSid>/<sid>`, which also sets `accountSid` on import:

```bash
pulumi import twilio:index:Subaccount tenant ACxxxxxxxx
pulumi import twilio:index:ApiKey ci SKxxxxxxxx
pulumi import twilio:index:IncomingPhoneNumber main PNxxxxxxxx
pulumi import twilio:index:Application ivr ACsubxxxxxxxx/APxxxxxxxx   # an application in a subaccount
pulumi import twilio:index:MessagingService svc MGxxxxxxxx
pulumi import twilio:index:MessagingServicePhoneNumber sender MGxxxxxxxx/PNxxxxxxxx
pulumi import twilio:index:VerifyService verify VAxxxxxxxx
```

Import adopts every setting that differs from Twilio's default, so a program that leaves the defaults out plans no change. Resources created by earlier versions with a bare SID keep working.

## Destructive and billing semantics

* `Subaccount` destroy **closes** the subaccount. Twilio releases its phone numbers when it closes, and a closed subaccount cannot be reopened after 30 days. Use `status: suspended` to pause instead.
* `IncomingPhoneNumber` adopts a number the account already owns and fails if it is not owned. It does not purchase numbers (that is billed and out of scope for this release). Destroy only stops managing the number: it is not released and its current webhooks stay in place. Webhook fields you remove from the program are cleared on Twilio at the next update.
* A phone number belongs to at most one Messaging Service. `MessagingServicePhoneNumber` destroy removes the number from the pool without releasing it.
* `ApiKey.secret` is returned only by the create call, so it cannot be recovered or imported. Rotate by creating a new key, rolling consumers, then removing the old one. `keyType` and `accountSid` replace the key; `friendlyName` and `policy` update in place. The policy is read back on every refresh, so a restricted key widened in the console shows up as a diff and the next `pulumi up` restores it. A standard key is created without a `KeyType` parameter; `restricted` is sent only for restricted keys.
* Removing an optional input from the program resets that field on Twilio at the next update: URLs are cleared, HTTP methods go back to `POST`, and a boolean or number goes back to Twilio's default only where the Twilio OpenAPI definitions state one: `validityPeriod` (`36000`) and `doNotShareWarningEnabled` (`false`). Every other removed boolean or number (`stickySender`, `mmsConverter`, `smartEncoding`, `areaCodeGeomatch`, `useInboundWebhookOnNumber`, `voiceCallerIdLookup`, `codeLength`, `lookupEnabled`, `psd2Enabled` and so on) is left untouched on Twilio, because the API gives no documented value to reset it to; set it explicitly to change it. The friendly name of an `IncomingPhoneNumber` has no default and is left as it is. Import adopts every returned boolean and number without a documented default as an input.
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

Examples in all six languages are in `examples/`, starter templates in `templates/`.

## Roadmap (not yet implemented)

* Buying numbers (search and purchase), which is billed and needs an explicit release policy.
* Studio flows, Flex, TaskRouter, Conversations, SIP trunking, Verify templates and Messaging compliance (A2P 10DLC brands and campaigns).
* Account-level settings and usage triggers.

## Development

```bash
make tidy build test lint
make gen-sdk   # needs the pulumi CLI
```

The provider uses its own small REST client (form-encoded requests, JSON responses, HTTP Basic) instead of the `twilio-go` SDK, which keeps the module graph small and lets the tests point every product host at one fake server.
