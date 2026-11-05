# Changelog

## [Unreleased]

## [0.1.0] - 2026-11-04

### Added

- Add `qa.yaml`, the runtime test manifest QA reads: credentials, every resource and function with its test plan and cost, and the regressions to re-check. It is copied to the mirror root and changes no provider behavior.
- Lookup functions for every resource: `getSubaccount` (arg `subaccountSid`), `getApiKey` (`keySid`, optional `accountSid`; never returns the secret), `getApplication` (`applicationSid`, optional `accountSid`), `getMessagingService` and `getVerifyService` (`serviceSid`; results carry the configuration Twilio reports) and `getMessagingServicePhoneNumber` (`serviceSid`, `phoneNumberSid`). A missing object fails with "<object> was not found".
- `getBalance` reads the prepaid balance and currency of the provider account or a subaccount.
- Coverage inventory of Twilio's SDKs and OpenAPI definitions in `research/coverage/twilio.md`.

### Fixed

- `Subaccount` destroy of the provider's own top-level account is refused with a hint to run `pulumi state delete` to drop it from state.
- Preview no longer fails when a validated input is computed (for example a `friendlyName` built from another resource, or the `policy` of a restricted `ApiKey` that references a service created in the same stack): `Check` skips unknown values on `Subaccount`, `ApiKey`, `IncomingPhoneNumber`, `Application`, `MessagingService` and `VerifyService`.
- Removing a boolean or number input (`MessagingService` flags and `validityPeriod`, `VerifyService` flags and `codeLength`, `voiceCallerIdLookup`) is no longer silently mishandled: only inputs whose default is stated in Twilio's OpenAPI definitions are reset (`validityPeriod` to `36000`, `doNotShareWarningEnabled` to `false`); every other removed flag or number (`stickySender`, `useInboundWebhookOnNumber`, `codeLength`, `voiceCallerIdLookup` and so on) is left untouched on Twilio, because a reset to an unverified guess is worse than doing nothing. Import adopts every returned flag or number that has no documented default as an input.
- BREAKING: removing an HTTP method input (`voiceMethod`, `smsMethod`, `inboundMethod`, and the other `*Method` inputs) now sends `POST` instead of an empty string, which Twilio can reject.
- Refresh no longer echoes inputs: `Application`, `IncomingPhoneNumber`, `MessagingService` and `VerifyService` gain an `effective` output with the configuration Twilio reports (including console changes to fields the program does not set). Inputs the program leaves unset are never overwritten by server defaults.
- BREAKING: import now works for objects in a subaccount. `Application`, `IncomingPhoneNumber` and `ApiKey` created in a subaccount get the ID `<accountSid>/<sid>` (objects in the provider's own account keep the bare SID, and existing state keeps working). Importing an `ApiKey` from a subaccount sets `accountSid`, so a program that sets it no longer plans a replace.
- `ApiKey` in a subaccount: refresh, update and delete fall back to the account-explicit `/Accounts/{AccountSid}/Keys/{Sid}.json` route when the IAM route cannot see the key, so a subaccount key is no longer dropped from state on refresh and a delete no longer leaves a live credential behind. (Whether the IAM route actually scopes by account is not verified against a live account.)
- `IncomingPhoneNumber.accountSid` is no longer replace-on-change: a number is moved between accounts in place with Twilio's `AccountSid` update parameter (removing `accountSid` sends the number back to the provider account, so `protect` the resource). Previewing an update needs no credentials. BREAKING: changing `accountSid` used to plan a replace that always failed.
- `ApiKey` policy drift is detected: the policy is read back on refresh and compared as JSON on the keys the program supplied (extra fields Twilio echoes are not drift, and key order never matters, including in the subaccount-policy update check), so a restricted key widened in the console shows up as a diff and is restored by the next update. Import infers `keyType` and `policy`.
- `ApiKey` is created without a `KeyType` parameter for standard keys (`restricted` is sent only for restricted keys).
- `Subaccount` can no longer adopt or close the provider's own top-level account: import of it fails and destroy of it is refused.
- A failure after the object exists keeps it in state instead of leaking it: `Subaccount` created with `status: suspended` whose suspend call fails, and an `IncomingPhoneNumber` whose webhook configuration call fails, now return an initialization failure with the ID and partial state.
- README: the stored `ApiKey` policy is now read back, and import, subaccount IDs and removal semantics are documented.

- Initial native provider for Twilio (requested in #1412; Twilio has no maintained Terraform provider, so this is not a bridge).
- Resources: `Subaccount` (inputs `friendlyName`, `status`; outputs `sid`, secret `authToken`, `ownerAccountSid`, `dateCreated`; destroy closes the subaccount), `ApiKey` (inputs `friendlyName`, `accountSid`, `keyType`, `policy`; outputs `sid`, secret `secret`, `dateCreated`), `IncomingPhoneNumber` (adopts an owned number; inputs `phoneNumber` or `phoneNumberSid`, `accountSid`, `friendlyName`, voice and SMS webhooks and methods, fallbacks, status callback, `voiceApplicationSid`, `smsApplicationSid`, `voiceCallerIdLookup`; outputs `sid`, `number`, `ownerAccountSid`; never purchases or releases), `Application` (TwiML application webhooks; outputs `sid`, `ownerAccountSid`), `MessagingService` (webhooks, `stickySender`, `mmsConverter`, `smartEncoding`, `fallbackToLongCode`, `areaCodeGeomatch`, `synchronousValidation`, `useInboundWebhookOnNumber`, `scanMessageContent`, `validityPeriod`; output `sid`), `MessagingServicePhoneNumber` (inputs `serviceSid`, `phoneNumberSid`; outputs `phoneNumber`, `capabilities`) and `VerifyService` (inputs `friendlyName`, `codeLength`, `lookupEnabled`, `skipSmsToLandlines`, `dtmfInputRequired`, `ttsName`, `psd2Enabled`, `doNotShareWarningEnabled`, `customCodeEnabled`; output `sid`).
- Functions: `getAccount` (output `sid`, `friendlyName`, `status`, `type`, `ownerAccountSid`, `dateCreated`) and `getIncomingPhoneNumber` (by `phoneNumber` or `sid`).
- Provider config: `accountSid`, secret `authToken`, `apiKey`, secret `apiSecret`, `region`, `edge` and a testing-only `baseUrl`, each with a `TWILIO_*` environment fallback.
- Examples (messaging and subaccount scenarios in six languages) and `twilio-messaging` templates.
- Not validated against a live Twilio account; see README.
