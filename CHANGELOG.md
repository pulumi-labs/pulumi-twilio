# Changelog

## [Unreleased]

- Initial native provider for Twilio (requested in #1412; Twilio has no maintained Terraform provider, so this is not a bridge).
- Resources: `Subaccount` (inputs `friendlyName`, `status`; outputs `sid`, secret `authToken`, `ownerAccountSid`, `dateCreated`; destroy closes the subaccount), `ApiKey` (inputs `friendlyName`, `accountSid`, `keyType`, `policy`; outputs `sid`, secret `secret`, `dateCreated`), `IncomingPhoneNumber` (adopts an owned number; inputs `phoneNumber` or `phoneNumberSid`, `accountSid`, `friendlyName`, voice and SMS webhooks and methods, fallbacks, status callback, `voiceApplicationSid`, `smsApplicationSid`, `voiceCallerIdLookup`; outputs `sid`, `number`, `ownerAccountSid`; never purchases or releases), `Application` (TwiML application webhooks; outputs `sid`, `ownerAccountSid`), `MessagingService` (webhooks, `stickySender`, `mmsConverter`, `smartEncoding`, `fallbackToLongCode`, `areaCodeGeomatch`, `synchronousValidation`, `useInboundWebhookOnNumber`, `scanMessageContent`, `validityPeriod`; output `sid`), `MessagingServicePhoneNumber` (inputs `serviceSid`, `phoneNumberSid`; outputs `phoneNumber`, `capabilities`) and `VerifyService` (inputs `friendlyName`, `codeLength`, `lookupEnabled`, `skipSmsToLandlines`, `dtmfInputRequired`, `ttsName`, `psd2Enabled`, `doNotShareWarningEnabled`, `customCodeEnabled`; output `sid`).
- Functions: `getAccount` (output `sid`, `friendlyName`, `status`, `type`, `ownerAccountSid`, `dateCreated`) and `getIncomingPhoneNumber` (by `phoneNumber` or `sid`).
- Provider config: `accountSid`, secret `authToken`, `apiKey`, secret `apiSecret`, `region`, `edge` and a testing-only `baseUrl`, each with a `TWILIO_*` environment fallback.
- Examples (messaging and subaccount scenarios in six languages) and `twilio-messaging` templates.
- Not validated against a live Twilio account; see README.
