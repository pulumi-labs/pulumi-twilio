---
title: Twilio Installation & Configuration
meta_desc: How to install the Pulumi Twilio provider and its language SDKs, and how to configure credentials, region and edge.
layout: package
---

## Installation

The Twilio provider is available as a package in all Pulumi-supported languages:

* JavaScript/TypeScript: [`@pulumi-labs/twilio`](https://www.npmjs.com/package/@pulumi-labs/twilio)
* Python: [`pulumi-labs-twilio`](https://pypi.org/project/pulumi-labs-twilio/)
* Go: [`github.com/pulumi-labs/pulumi-twilio/sdk/go/twilio`](https://github.com/pulumi-labs/pulumi-twilio)
* .NET: [`Pulumi.Labs.Twilio`](https://www.nuget.org/packages/Pulumi.Labs.Twilio)
* Java: [`com.pulumi.labs:twilio`](https://central.sonatype.com/artifact/com.pulumi.labs/twilio)

In normal usage, installing one of the language SDKs above is enough. The first `pulumi up` automatically downloads the matching provider binary from the URL embedded in the SDK, so no separate step is required.

### Provider Binary (optional)

If you need to install the provider binary explicitly (for example, in an air-gapped environment or to prewarm a CI cache), use the `pulumi plugin` command:

```bash
pulumi plugin install resource twilio <version> --server github://api.github.com/pulumi-labs/pulumi-twilio
```

Replace `<version>` with your desired release (see the [releases page](https://github.com/pulumi-labs/pulumi-twilio/releases)).

## Configuration

The provider authenticates with the account SID and either the account auth token, or an API key and its secret. It needs no explicit `Provider` resource and no stack config when the credentials are available as environment variables:

```bash
export TWILIO_ACCOUNT_SID=<account-sid>
export TWILIO_AUTH_TOKEN=<auth-token>
```

You can also set them as provider configuration:

```bash
pulumi config set twilio:accountSid <account-sid>
pulumi config set --secret twilio:authToken <auth-token>
```

Explicit Pulumi config wins over the environment variable. The following keys are available:

| Key | Env var | Default | Description |
|---|---|---|---|
| `accountSid` | `TWILIO_ACCOUNT_SID` | none | Required. The SID of the account that owns the resources. |
| `authToken` (secret) | `TWILIO_AUTH_TOKEN` | none | The account auth token. |
| `apiKey` | `TWILIO_API_KEY` | none | An API key SID, used instead of the auth token; requires `apiSecret`. A restricted key cannot manage `Subaccount` or `ApiKey` resources. |
| `apiSecret` (secret) | `TWILIO_API_SECRET` | none | The secret of `apiKey`. |
| `region` | `TWILIO_REGION` | none | Twilio region for data residency, such as `au1` or `ie1`. |
| `edge` | `TWILIO_EDGE` | none | Twilio edge location, such as `sydney`; requires `region`. |
| `baseUrl` | `TWILIO_BASE_URL` | none | Testing aid: send every request to `<baseUrl>/<product>/<path>`. Must be `https` (loopback `http` is accepted). |

When both an API key and an auth token are set, the API key is used. An explicit `twilio` provider resource is only needed to target several accounts in one program.
