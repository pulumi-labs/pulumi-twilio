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

package provider

import (
	"reflect"
	"testing"

	pgp "github.com/pulumi/pulumi-go-provider"
	"github.com/pulumi/pulumi/sdk/v3/go/property"
	"github.com/stretchr/testify/require"
)

// #1573: with no WireDependencies infer marks every output of an Update preview computed when any
// input changed, so a preview that only renames a Subaccount made its sid unknown, and every
// account-scoped child that reads ${sub.sid} (accountSid replaces) was planned for replacement,
// although the real update is in place. These tests run the preview Update through the provider.

// previewUpdate runs a preview Update (DryRun) of one resource with changed inputs.
func previewUpdate(t *testing.T, l *tl, id string, state, oldInputs, newInputs property.Map) pgp.UpdateResponse {
	t.Helper()
	u, err := l.s.Update(pgp.UpdateRequest{ID: id, Urn: l.urn, State: state, OldInputs: oldInputs, Inputs: l.checked(newInputs), DryRun: true})
	require.NoError(t, err)
	return u
}

func requireKnown(t *testing.T, m property.Map, keys ...string) {
	t.Helper()
	for _, k := range keys {
		v, ok := m.GetOk(k)
		require.True(t, ok, "%s missing from %v", k, m)
		require.False(t, v.IsComputed(), "%s must stay known in an update preview: %v", k, m)
	}
}

func TestSubaccount_PreviewUpdateKeepsIdentityKnown(t *testing.T) {
	l := newTL(t, "Subaccount", "sub")
	in := pv(map[string]property.Value{"friendlyName": sv("a")})
	c := l.create(in)
	next := pv(map[string]property.Value{"friendlyName": sv("b")})
	u := previewUpdate(t, l, c.ID, c.Properties, in, next)
	requireKnown(t, u.Properties, "sid", "ownerAccountSid", "dateCreated", "authToken")
	require.Equal(t, "b", getS(t, u.Properties, "friendlyName"))
}

// requireComputed asserts the keys are unknown in the preview: they are values Twilio reports and
// can change with the update, so wiring must not have made them look fixed.
func requireComputed(t *testing.T, m property.Map, keys ...string) {
	t.Helper()
	for _, k := range keys {
		v, ok := m.GetOk(k)
		require.True(t, ok, "%s missing from %v", k, m)
		require.True(t, v.IsComputed(), "%s depends on the inputs and must be unknown in an update preview: %v", k, m)
	}
}

// Every resource: the identifiers fixed at creation stay known when an input changes in a
// preview; what Twilio reports (the effective configuration, counters, validation) stays unknown.
func TestWiring_PreviewUpdateKeepsIdentifiersKnown(t *testing.T) {
	cases := []struct {
		name, typ string
		in, next  property.Map
		known     []string
		computed  []string
		setup     func(l *tl) property.Map // optional: returns the inputs to use
	}{
		{name: "Subaccount", typ: "Subaccount", in: pv(map[string]property.Value{"friendlyName": sv("a")}),
			next:  pv(map[string]property.Value{"friendlyName": sv("b")}),
			known: []string{"sid", "ownerAccountSid", "dateCreated", "authToken"}},
		{name: "Address", typ: "Address", in: addrInputs(), next: with(addrInputs(), map[string]property.Value{"friendlyName": sv("hq2")}),
			known: []string{"sid", "ownerAccountSid", "dateCreated"}, computed: []string{"validated", "verified", "dateUpdated"}},
		{name: "ApiKey", typ: "ApiKey", in: pv(map[string]property.Value{"friendlyName": sv("k")}),
			next:  pv(map[string]property.Value{"friendlyName": sv("k2")}),
			known: []string{"sid", "secret", "dateCreated"}},
		{name: "SigningKey", typ: "SigningKey", in: pv(map[string]property.Value{"friendlyName": sv("k")}),
			next:  pv(map[string]property.Value{"friendlyName": sv("k2")}),
			known: []string{"sid", "ownerAccountSid", "secret", "dateCreated"}},
		{name: "Application", typ: "Application", in: appInputs(), next: with(appInputs(), map[string]property.Value{"friendlyName": sv("ivr2")}),
			known: []string{"sid", "ownerAccountSid"}, computed: []string{"effective"}},
		{name: "MessagingService", typ: "MessagingService", in: msgInputs(), next: with(msgInputs(), map[string]property.Value{"friendlyName": sv("notify2")}),
			known: []string{"sid"}, computed: []string{"effective"}},
		{name: "VerifyService", typ: "VerifyService", in: verifyInputs(), next: with(verifyInputs(), map[string]property.Value{"friendlyName": sv("login2")}),
			known: []string{"sid"}, computed: []string{"effective"}},
		{name: "UsageTrigger", typ: "UsageTrigger", in: triggerInputs(), next: with(triggerInputs(), map[string]property.Value{"friendlyName": sv("sms-cap2")}),
			known: []string{"sid", "ownerAccountSid", "usageRecordUri", "apiVersion"}, computed: []string{"currentValue", "dateFired"}},
		{name: "Queue", typ: "Queue", in: pv(map[string]property.Value{"friendlyName": sv("q"), "maxSize": nv(10)}),
			next:  pv(map[string]property.Value{"friendlyName": sv("q"), "maxSize": nv(20)}),
			known: []string{"sid", "ownerAccountSid", "dateCreated"}, computed: []string{"currentSize", "averageWaitTime", "dateUpdated"}},
		{name: "EventSink", typ: "EventSink", in: webhookSinkInputs(), next: with(webhookSinkInputs(), map[string]property.Value{"description": sv("renamed")}),
			known: []string{"sid", "dateCreated"}, computed: []string{"status", "dateUpdated"}},
		{name: "IncomingPhoneNumber", typ: "IncomingPhoneNumber", in: numberInputs(), next: with(numberInputs(), map[string]property.Value{"friendlyName": sv("support2")}),
			known: []string{"sid", "number", "ownerAccountSid"}, computed: []string{"effective"},
			setup: func(l *tl) property.Map { l.f.addNumber(testAccount, numSid, numE); return numberInputs() }},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			l := newTL(t, c.typ, "r")
			in := c.in
			if c.setup != nil {
				in = c.setup(l)
			}
			created := l.create(in)
			u := previewUpdate(t, l, created.ID, created.Properties, in, c.next)
			requireKnown(t, u.Properties, c.known...)
			requireComputed(t, u.Properties, c.computed...)
		})
	}
}

// wireEchoes lines the state up with the embedded args struct by position, and panics when a state
// struct does not embed its args first. This pins that for every resource, including
// MessagingServicePhoneNumber, whose inputs all replace and so has no update preview to cover it.
func TestWiring_EveryStateEmbedsItsArgsFirst(t *testing.T) {
	pairs := map[string][2]any{
		"Subaccount":                             {SubaccountArgs{}, SubaccountState{}},
		"Address":                                {AddressArgs{}, AddressState{}},
		"ApiKey":                                 {ApiKeyArgs{}, ApiKeyState{}},
		"Application":                            {ApplicationArgs{}, ApplicationState{}},
		"IncomingPhoneNumber":                    {IncomingPhoneNumberArgs{}, IncomingPhoneNumberState{}},
		"MessagingService":                       {MessagingServiceArgs{}, MessagingServiceState{}},
		"MessagingServicePhoneNumber":            {MessagingServicePhoneNumberArgs{}, MessagingServicePhoneNumberState{}},
		"OAuthApp":                               {OAuthAppArgs{}, OAuthAppState{}},
		"MessagingServiceShortCode":              {MessagingServiceShortCodeArgs{}, MessagingServiceShortCodeState{}},
		"MessagingServiceChannelSender":          {MessagingServiceChannelSenderArgs{}, MessagingServiceChannelSenderState{}},
		"MessagingServiceAlphaSender":            {MessagingServiceAlphaSenderArgs{}, MessagingServiceAlphaSenderState{}},
		"MessagingServiceDestinationAlphaSender": {MessagingServiceDestinationAlphaSenderArgs{}, MessagingServiceDestinationAlphaSenderState{}},
		"Queue":                                  {QueueArgs{}, QueueState{}},
		"SigningKey":                             {SigningKeyArgs{}, SigningKeyState{}},
		"UsageTrigger":                           {UsageTriggerArgs{}, UsageTriggerState{}},
		"SipCredentialList":                      {SipCredentialListArgs{}, SipCredentialListState{}},
		"SipCredential":                          {SipCredentialArgs{}, SipCredentialState{}},
		"SipIpAccessControlList":                 {SipIpAccessControlListArgs{}, SipIpAccessControlListState{}},
		"SipIpAddress":                           {SipIpAddressArgs{}, SipIpAddressState{}},
		"SipDomain":                              {SipDomainArgs{}, SipDomainState{}},
		"VerifyService":                          {VerifyServiceArgs{}, VerifyServiceState{}},
		"ShortCode":                              {ShortCodeArgs{}, ShortCodeState{}},
		"ConnectApp":                             {ConnectAppArgs{}, ConnectAppState{}},
		"OutgoingCallerId":                       {OutgoingCallerIdArgs{}, OutgoingCallerIdState{}},
		"EventSink":                              {EventSinkArgs{}, EventSinkState{}},
		"EventSubscription":                      {EventSubscriptionArgs{}, EventSubscriptionState{}},
		"ConnectionPolicy":                       {ConnectionPolicyArgs{}, ConnectionPolicyState{}},
		"ConnectionPolicyTarget":                 {ConnectionPolicyTargetArgs{}, ConnectionPolicyTargetState{}},
		"ByocTrunk":                              {ByocTrunkArgs{}, ByocTrunkState{}},
		"IpRecord":                               {IpRecordArgs{}, IpRecordState{}},
		"VoiceRecordingConfiguration":            {VoiceRecordingConfigurationArgs{}, VoiceRecordingConfigurationState{}},
		"VoiceTranscriptionConfiguration":        {VoiceTranscriptionConfigurationArgs{}, VoiceTranscriptionConfigurationState{}},
		"VoiceAccountDefaultRecordingConfiguration": {VoiceAccountDefaultRecordingConfigurationArgs{}, VoiceAccountDefaultRecordingConfigurationState{}},
		"VoiceDefaultRecordingConfiguration":        {VoiceDefaultRecordingConfigurationArgs{}, VoiceDefaultRecordingConfigurationState{}},
		"VoiceDefaultTranscriptionConfiguration":    {VoiceDefaultTranscriptionConfigurationArgs{}, VoiceDefaultTranscriptionConfigurationState{}},
		"RoutesSipDomain":                           {RoutesSipDomainArgs{}, RoutesSipDomainState{}},
		"RoutesTrunk":                               {RoutesTrunkArgs{}, RoutesTrunkState{}},
		"RoutesPhoneNumber":                         {RoutesPhoneNumberArgs{}, RoutesPhoneNumberState{}},
		"RoutesShortCode":                           {RoutesShortCodeArgs{}, RoutesShortCodeState{}},
		"SourceIpMapping":                           {SourceIpMappingArgs{}, SourceIpMappingState{}},
		"ContentTemplate":                           {ContentTemplateArgs{}, ContentTemplateState{}},
		"ChannelsSender":                            {ChannelsSenderArgs{}, ChannelsSenderState{}},
		"SuperSimNetworkAccessProfile":              {SuperSimNetworkAccessProfileArgs{}, SuperSimNetworkAccessProfileState{}},
		"FlexFlow":                                  {FlexFlowArgs{}, FlexFlowState{}},
		"FlexPlugin":                                {FlexPluginArgs{}, FlexPluginState{}},
		"FlexPluginVersion":                         {FlexPluginVersionArgs{}, FlexPluginVersionState{}},
		"FlexPluginConfiguration":                   {FlexPluginConfigurationArgs{}, FlexPluginConfigurationState{}},
		"FlexPluginRelease":                         {FlexPluginReleaseArgs{}, FlexPluginReleaseState{}},
		"LinkShorteningDomainConfig":                {LinkShorteningDomainConfigArgs{}, LinkShorteningDomainConfigState{}},
		"LinkShorteningDomainCertificate":           {LinkShorteningDomainCertificateArgs{}, LinkShorteningDomainCertificateState{}},
		"LinkShorteningMessagingServiceDomain":      {LinkShorteningMessagingServiceDomainArgs{}, LinkShorteningMessagingServiceDomainState{}},
		"TrustHubEndUser":                           {TrustHubEndUserArgs{}, TrustHubEndUserState{}},
		"TrustHubSupportingDocument":                {TrustHubSupportingDocumentArgs{}, TrustHubSupportingDocumentState{}},
		"TollfreeVerification":                      {TollfreeVerificationArgs{}, TollfreeVerificationState{}},
		"UsAppToPerson":                             {UsAppToPersonArgs{}, UsAppToPersonState{}},
		"BrandRegistration":                         {BrandRegistrationArgs{}, BrandRegistrationState{}},
		"SuperSimFleet":                             {SuperSimFleetArgs{}, SuperSimFleetState{}},
		"WirelessRatePlan":                          {WirelessRatePlanArgs{}, WirelessRatePlanState{}},
	}
	for name, p := range pairs {
		args, state := reflect.TypeOf(p[0]), reflect.TypeOf(p[1])
		first := state.Field(0)
		require.True(t, first.Anonymous && first.Type == args, "%sState must embed %sArgs as its first field for wireEchoes", name, name)
	}
}
