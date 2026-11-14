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

	"github.com/pulumi/pulumi-go-provider/infer"
)

// By default infer marks every output computed during an update preview when any input changed.
// Most Twilio outputs are identifiers fixed at creation (sid, ownerAccountSid, dateCreated, the
// auth token), so previewing a rename of a Subaccount made its sid unknown, and every resource
// that reads ${sub.sid} into its replaceOnChanges accountSid was planned for replacement, although
// the real update is in place and replacing a subaccount ApiKey would lose its one-time secret
// (#1573). Each resource below declares what its outputs really derive from. Declaring any
// dependency turns the defaults off for all of the resource's outputs, so everything is wired:
//
//   - an input echoed into the state depends on that input;
//   - an identifier fixed at creation depends on nothing, so it stays known on update;
//   - a value Twilio reports and can change (the effective configuration, counters, timestamps of
//     the last update, validation flags) depends on every input.
//
// Create previews are unaffected: nothing is known before the resource exists.

// wireEchoes makes each input echoed into the state (the embedded args struct) depend on itself.
// state embeds A as its first field, so the fields line up one to one.
func wireEchoes[A, S any](f infer.FieldSelector, args *A, state *S) {
	av := reflect.ValueOf(args).Elem()
	sv := reflect.ValueOf(state).Elem().Field(0)
	for i := 0; i < av.NumField(); i++ {
		f.OutputField(sv.Field(i).Addr().Interface()).DependsOn(f.InputField(av.Field(i).Addr().Interface()).Computed())
	}
}

// WireDependencies: the SID, owner, creation time and auth token are fixed when the subaccount is created.
func (r *Subaccount) WireDependencies(f infer.FieldSelector, args *SubaccountArgs, state *SubaccountState) {
	wireEchoes(f, args, state)
}

// WireDependencies: the SID, owner and creation time are fixed at creation; validation can change with the address.
func (r *Address) WireDependencies(f infer.FieldSelector, args *AddressArgs, state *AddressState) {
	wireEchoes(f, args, state)
	for _, o := range []any{&state.Validated, &state.Verified, &state.DateUpdated} {
		f.OutputField(o).DependsOn(f.InputField(args).Computed())
	}
}

// WireDependencies: the SID, secret and creation time are fixed at creation.
func (r *ApiKey) WireDependencies(f infer.FieldSelector, args *ApiKeyArgs, state *ApiKeyState) {
	wireEchoes(f, args, state)
}

// WireDependencies: the SID and owner are fixed at creation; the effective configuration follows every input.
func (r *Application) WireDependencies(f infer.FieldSelector, args *ApplicationArgs, state *ApplicationState) {
	wireEchoes(f, args, state)
	f.OutputField(&state.Effective).DependsOn(f.InputField(args).Computed())
}

// WireDependencies: the SID, number and owner are fixed once the number is adopted; the effective
// configuration follows every input.
func (r *IncomingPhoneNumber) WireDependencies(f infer.FieldSelector, args *IncomingPhoneNumberArgs, state *IncomingPhoneNumberState) {
	wireEchoes(f, args, state)
	f.OutputField(&state.Effective).DependsOn(f.InputField(args).Computed())
}

// WireDependencies: the SID is fixed at creation; the effective configuration follows every input.
func (r *MessagingService) WireDependencies(f infer.FieldSelector, args *MessagingServiceArgs, state *MessagingServiceState) {
	wireEchoes(f, args, state)
	f.OutputField(&state.Effective).DependsOn(f.InputField(args).Computed())
}

// WireDependencies: both inputs replace; the sender's number and capabilities are what Twilio reports for it.
func (r *MessagingServicePhoneNumber) WireDependencies(f infer.FieldSelector, args *MessagingServicePhoneNumberArgs, state *MessagingServicePhoneNumberState) {
	wireEchoes(f, args, state)
	f.OutputField(&state.PhoneNumber).DependsOn(f.InputField(args).Computed())
	f.OutputField(&state.Capabilities).DependsOn(f.InputField(args).Computed())
}

// WireDependencies: the SID, owner and creation time are fixed at creation; the counters and the
// update time are what Twilio reports.
func (r *Queue) WireDependencies(f infer.FieldSelector, args *QueueArgs, state *QueueState) {
	wireEchoes(f, args, state)
	for _, o := range []any{&state.CurrentSize, &state.AverageWaitTime, &state.DateUpdated} {
		f.OutputField(o).DependsOn(f.InputField(args).Computed())
	}
}

// WireDependencies: the SID, owner, secret and creation time are fixed at creation.
func (r *SigningKey) WireDependencies(f infer.FieldSelector, args *SigningKeyArgs, state *SigningKeyState) {
	wireEchoes(f, args, state)
}

// WireDependencies: the SID, owner, record URI and API version are fixed at creation; the usage
// value and the time the trigger fired are what Twilio reports.
func (r *UsageTrigger) WireDependencies(f infer.FieldSelector, args *UsageTriggerArgs, state *UsageTriggerState) {
	wireEchoes(f, args, state)
	f.OutputField(&state.CurrentValue).DependsOn(f.InputField(args).Computed())
	f.OutputField(&state.DateFired).DependsOn(f.InputField(args).Computed())
}

// WireDependencies: the SID is fixed at creation; the effective configuration follows every input.
func (r *VerifyService) WireDependencies(f infer.FieldSelector, args *VerifyServiceArgs, state *VerifyServiceState) {
	wireEchoes(f, args, state)
	f.OutputField(&state.Effective).DependsOn(f.InputField(args).Computed())
}

// WireDependencies: the SID, owner and creation time are fixed at creation; the update time is what Twilio reports.
func (r *SipCredentialList) WireDependencies(f infer.FieldSelector, args *SipCredentialListArgs, state *SipCredentialListState) {
	wireEchoes(f, args, state)
	f.OutputField(&state.DateUpdated).DependsOn(f.InputField(args).Computed())
}

// WireDependencies: the SID, owner and creation time are fixed at creation; the update time is what Twilio reports.
func (r *SipCredential) WireDependencies(f infer.FieldSelector, args *SipCredentialArgs, state *SipCredentialState) {
	wireEchoes(f, args, state)
	f.OutputField(&state.DateUpdated).DependsOn(f.InputField(args).Computed())
}

// WireDependencies: the SID, owner and creation time are fixed at creation; the update time is what Twilio reports.
func (r *SipIpAccessControlList) WireDependencies(f infer.FieldSelector, args *SipIpAccessControlListArgs, state *SipIpAccessControlListState) {
	wireEchoes(f, args, state)
	f.OutputField(&state.DateUpdated).DependsOn(f.InputField(args).Computed())
}

// WireDependencies: the SID, owner and creation time are fixed at creation; the update time is what Twilio reports.
func (r *SipIpAddress) WireDependencies(f infer.FieldSelector, args *SipIpAddressArgs, state *SipIpAddressState) {
	wireEchoes(f, args, state)
	f.OutputField(&state.DateUpdated).DependsOn(f.InputField(args).Computed())
}

// WireDependencies: the SID, owner, API version and creation time are fixed at creation; the update time is what Twilio reports.
// The auth type is set by domain mappings Twilio manages outside this resource, so it follows every input.
func (r *SipDomain) WireDependencies(f infer.FieldSelector, args *SipDomainArgs, state *SipDomainState) {
	wireEchoes(f, args, state)
	for _, o := range []any{&state.AuthType, &state.DateUpdated} {
		f.OutputField(o).DependsOn(f.InputField(args).Computed())
	}
}

func (r *ShortCode) WireDependencies(f infer.FieldSelector, args *ShortCodeArgs, state *ShortCodeState) {
	wireEchoes(f, args, state)
	f.OutputField(&state.Effective).DependsOn(f.InputField(args).Computed())
}

func (r *ConnectApp) WireDependencies(f infer.FieldSelector, args *ConnectAppArgs, state *ConnectAppState) {
	wireEchoes(f, args, state)
	f.OutputField(&state.Effective).DependsOn(f.InputField(args).Computed())
}

func (r *OutgoingCallerId) WireDependencies(f infer.FieldSelector, args *OutgoingCallerIdArgs, state *OutgoingCallerIdState) {
	wireEchoes(f, args, state)
	f.OutputField(&state.Effective).DependsOn(f.InputField(args).Computed())
}

// WireDependencies: the SID and creation time are fixed at creation; the status follows validation and the update time what Twilio reports.
func (r *EventSink) WireDependencies(f infer.FieldSelector, args *EventSinkArgs, state *EventSinkState) {
	wireEchoes(f, args, state)
	for _, o := range []any{&state.Status, &state.DateUpdated} {
		f.OutputField(o).DependsOn(f.InputField(args).Computed())
	}
}

// WireDependencies: the SID, owner and creation time are fixed at creation; the update time is what Twilio reports.
func (r *EventSubscription) WireDependencies(f infer.FieldSelector, args *EventSubscriptionArgs, state *EventSubscriptionState) {
	wireEchoes(f, args, state)
	f.OutputField(&state.DateUpdated).DependsOn(f.InputField(args).Computed())
}

// WireDependencies: the SID and creation time are fixed at creation; the update time is what Twilio reports.
func (r *VerifyRateLimit) WireDependencies(f infer.FieldSelector, args *VerifyRateLimitArgs, state *VerifyRateLimitState) {
	wireEchoes(f, args, state)
	f.OutputField(&state.DateUpdated).DependsOn(f.InputField(args).Computed())
}

// WireDependencies: the SID and creation time are fixed at creation; the update time is what Twilio reports.
func (r *VerifyRateLimitBucket) WireDependencies(f infer.FieldSelector, args *VerifyRateLimitBucketArgs, state *VerifyRateLimitBucketState) {
	wireEchoes(f, args, state)
	f.OutputField(&state.DateUpdated).DependsOn(f.InputField(args).Computed())
}

// WireDependencies: the creation time is fixed at creation; the update time is what Twilio reports.
func (r *VerifyMessagingConfiguration) WireDependencies(f infer.FieldSelector, args *VerifyMessagingConfigurationArgs, state *VerifyMessagingConfigurationState) {
	wireEchoes(f, args, state)
	f.OutputField(&state.DateUpdated).DependsOn(f.InputField(args).Computed())
}

// WireDependencies: the SID and creation time are fixed at creation; the method and update time are what Twilio reports.
func (r *VerifyWebhook) WireDependencies(f infer.FieldSelector, args *VerifyWebhookArgs, state *VerifyWebhookState) {
	wireEchoes(f, args, state)
	for _, o := range []any{&state.WebhookMethod, &state.DateUpdated} {
		f.OutputField(o).DependsOn(f.InputField(args).Computed())
	}
}

// WireDependencies: the SID, domain and creation time are fixed at creation; the update time is what Twilio reports.
func (r *ServerlessService) WireDependencies(f infer.FieldSelector, args *ServerlessServiceArgs, state *ServerlessServiceState) {
	wireEchoes(f, args, state)
	f.OutputField(&state.DateUpdated).DependsOn(f.InputField(args).Computed())
}

// WireDependencies: an environment cannot be updated, so every output but the deployed build is fixed at creation.
func (r *ServerlessEnvironment) WireDependencies(f infer.FieldSelector, args *ServerlessEnvironmentArgs, state *ServerlessEnvironmentState) {
	wireEchoes(f, args, state)
	for _, o := range []any{&state.BuildSid, &state.DateUpdated} {
		f.OutputField(o).DependsOn(f.InputField(args).Computed())
	}
}

// WireDependencies: the SID and creation time are fixed at creation; the update time is what Twilio reports.
func (r *ServerlessVariable) WireDependencies(f infer.FieldSelector, args *ServerlessVariableArgs, state *ServerlessVariableState) {
	wireEchoes(f, args, state)
	f.OutputField(&state.DateUpdated).DependsOn(f.InputField(args).Computed())
}

// WireDependencies: the SID and creation time are fixed at creation; the update time is what Twilio reports.
func (r *ServerlessFunction) WireDependencies(f infer.FieldSelector, args *ServerlessFunctionArgs, state *ServerlessFunctionState) {
	wireEchoes(f, args, state)
	f.OutputField(&state.DateUpdated).DependsOn(f.InputField(args).Computed())
}

// WireDependencies: the SID and creation time are fixed at creation; the update time is what Twilio reports.
func (r *ServerlessAsset) WireDependencies(f infer.FieldSelector, args *ServerlessAssetArgs, state *ServerlessAssetState) {
	wireEchoes(f, args, state)
	f.OutputField(&state.DateUpdated).DependsOn(f.InputField(args).Computed())
}

// WireDependencies: versions are immutable, so every output is fixed at creation.
func (r *ServerlessFunctionVersion) WireDependencies(f infer.FieldSelector, args *ServerlessFunctionVersionArgs, state *ServerlessFunctionVersionState) {
	wireEchoes(f, args, state)
}

// WireDependencies: versions are immutable, so every output is fixed at creation.
func (r *ServerlessAssetVersion) WireDependencies(f infer.FieldSelector, args *ServerlessAssetVersionArgs, state *ServerlessAssetVersionState) {
	wireEchoes(f, args, state)
}

// WireDependencies: the SID and creation time are fixed at creation; the status and update time are what Twilio reports.
func (r *ServerlessBuild) WireDependencies(f infer.FieldSelector, args *ServerlessBuildArgs, state *ServerlessBuildState) {
	wireEchoes(f, args, state)
	for _, o := range []any{&state.Status, &state.DateUpdated} {
		f.OutputField(o).DependsOn(f.InputField(args).Computed())
	}
}

// WireDependencies: the SID and creation time are fixed at creation; the update time is what Twilio reports.
func (r *ServerlessDeployment) WireDependencies(f infer.FieldSelector, args *ServerlessDeploymentArgs, state *ServerlessDeploymentState) {
	wireEchoes(f, args, state)
	f.OutputField(&state.DateUpdated).DependsOn(f.InputField(args).Computed())
}

// WireDependencies: the SID, owner and creation time are fixed at creation; the update time is what Twilio reports.
func (r *ConnectionPolicy) WireDependencies(f infer.FieldSelector, args *ConnectionPolicyArgs, state *ConnectionPolicyState) {
	wireEchoes(f, args, state)
	f.OutputField(&state.DateUpdated).DependsOn(f.InputField(args).Computed())
}

// WireDependencies: the SID, owner and creation time are fixed at creation; the update time is what Twilio reports.
func (r *ConnectionPolicyTarget) WireDependencies(f infer.FieldSelector, args *ConnectionPolicyTargetArgs, state *ConnectionPolicyTargetState) {
	wireEchoes(f, args, state)
	f.OutputField(&state.DateUpdated).DependsOn(f.InputField(args).Computed())
}

// WireDependencies: the SID, owner and creation time are fixed at creation; the update time is what Twilio reports.
func (r *ByocTrunk) WireDependencies(f infer.FieldSelector, args *ByocTrunkArgs, state *ByocTrunkState) {
	wireEchoes(f, args, state)
	f.OutputField(&state.DateUpdated).DependsOn(f.InputField(args).Computed())
}

// WireDependencies: the SID, owner and creation time are fixed at creation; the update time is what Twilio reports.
func (r *IpRecord) WireDependencies(f infer.FieldSelector, args *IpRecordArgs, state *IpRecordState) {
	wireEchoes(f, args, state)
	f.OutputField(&state.DateUpdated).DependsOn(f.InputField(args).Computed())
}

// WireDependencies: the SID and creation time are fixed at creation; the update time is what Twilio reports.
func (r *SourceIpMapping) WireDependencies(f infer.FieldSelector, args *SourceIpMappingArgs, state *SourceIpMappingState) {
	wireEchoes(f, args, state)
	f.OutputField(&state.DateUpdated).DependsOn(f.InputField(args).Computed())
}

// WireDependencies: the ID, owner and creation time are fixed at creation; the update time is what Twilio reports.
func (r *VoiceRecordingConfiguration) WireDependencies(f infer.FieldSelector, args *VoiceRecordingConfigurationArgs, state *VoiceRecordingConfigurationState) {
	wireEchoes(f, args, state)
	f.OutputField(&state.DateUpdated).DependsOn(f.InputField(args).Computed())
}

// WireDependencies: the ID, owner and creation time are fixed at creation; the update time is what Twilio reports.
func (r *VoiceTranscriptionConfiguration) WireDependencies(f infer.FieldSelector, args *VoiceTranscriptionConfigurationArgs, state *VoiceTranscriptionConfigurationState) {
	wireEchoes(f, args, state)
	f.OutputField(&state.DateUpdated).DependsOn(f.InputField(args).Computed())
}

// WireDependencies: the account SID is fixed; the update time is what Twilio reports.
func (r *VoiceDefaultRecordingConfiguration) WireDependencies(f infer.FieldSelector, args *VoiceDefaultRecordingConfigurationArgs, state *VoiceDefaultRecordingConfigurationState) {
	wireEchoes(f, args, state)
	f.OutputField(&state.DateUpdated).DependsOn(f.InputField(args).Computed())
}

// WireDependencies: the account SID is fixed; the update time is what Twilio reports.
func (r *VoiceDefaultTranscriptionConfiguration) WireDependencies(f infer.FieldSelector, args *VoiceDefaultTranscriptionConfigurationArgs, state *VoiceDefaultTranscriptionConfigurationState) {
	wireEchoes(f, args, state)
	f.OutputField(&state.DateUpdated).DependsOn(f.InputField(args).Computed())
}

// WireDependencies: the SID and creation time are fixed at creation; the update time and the default activities are what Twilio reports.
func (r *TaskRouterWorkspace) WireDependencies(f infer.FieldSelector, args *TaskRouterWorkspaceArgs, state *TaskRouterWorkspaceState) {
	wireEchoes(f, args, state)
	for _, o := range []any{&state.DateUpdated, &state.DefaultActivitySid, &state.TimeoutActivitySid} {
		f.OutputField(o).DependsOn(f.InputField(args).Computed())
	}
}

// WireDependencies: the SID and creation time are fixed at creation; the update time is what Twilio reports.
func (r *TaskRouterActivity) WireDependencies(f infer.FieldSelector, args *TaskRouterActivityArgs, state *TaskRouterActivityState) {
	wireEchoes(f, args, state)
	f.OutputField(&state.DateUpdated).DependsOn(f.InputField(args).Computed())
}

// WireDependencies: the SID and creation time are fixed at creation; the update time is what Twilio reports.
func (r *TaskRouterTaskChannel) WireDependencies(f infer.FieldSelector, args *TaskRouterTaskChannelArgs, state *TaskRouterTaskChannelState) {
	wireEchoes(f, args, state)
	f.OutputField(&state.DateUpdated).DependsOn(f.InputField(args).Computed())
}

// WireDependencies: the SID and creation time are fixed at creation; the update time is what Twilio reports.
func (r *TaskRouterTaskQueue) WireDependencies(f infer.FieldSelector, args *TaskRouterTaskQueueArgs, state *TaskRouterTaskQueueState) {
	wireEchoes(f, args, state)
	f.OutputField(&state.DateUpdated).DependsOn(f.InputField(args).Computed())
}

// WireDependencies: the SID and creation time are fixed at creation; the update time is what Twilio reports.
func (r *TaskRouterWorkflow) WireDependencies(f infer.FieldSelector, args *TaskRouterWorkflowArgs, state *TaskRouterWorkflowState) {
	wireEchoes(f, args, state)
	f.OutputField(&state.DateUpdated).DependsOn(f.InputField(args).Computed())
}

// WireDependencies: the SID and creation time are fixed at creation; the update time is what Twilio reports.
func (r *TaskRouterWorker) WireDependencies(f infer.FieldSelector, args *TaskRouterWorkerArgs, state *TaskRouterWorkerState) {
	wireEchoes(f, args, state)
	f.OutputField(&state.DateUpdated).DependsOn(f.InputField(args).Computed())
}

// WireDependencies: a service cannot be updated, so every output is fixed at creation.
func (r *ConversationsService) WireDependencies(f infer.FieldSelector, args *ConversationsServiceArgs, state *ConversationsServiceState) {
	wireEchoes(f, args, state)
}

// WireDependencies: the SID and creation time are fixed at creation; the update time is what Twilio reports.
func (r *ConversationsRole) WireDependencies(f infer.FieldSelector, args *ConversationsRoleArgs, state *ConversationsRoleState) {
	wireEchoes(f, args, state)
	f.OutputField(&state.DateUpdated).DependsOn(f.InputField(args).Computed())
}

// WireDependencies: the SID and creation time are fixed at creation; the update time is what Twilio reports.
func (r *ConversationsCredential) WireDependencies(f infer.FieldSelector, args *ConversationsCredentialArgs, state *ConversationsCredentialState) {
	wireEchoes(f, args, state)
	f.OutputField(&state.DateUpdated).DependsOn(f.InputField(args).Computed())
}

// WireDependencies: the SID and creation time are fixed at creation; the update time is what Twilio reports.
func (r *ConversationsAddressConfiguration) WireDependencies(f infer.FieldSelector, args *ConversationsAddressConfigurationArgs, state *ConversationsAddressConfigurationState) {
	wireEchoes(f, args, state)
	f.OutputField(&state.DateUpdated).DependsOn(f.InputField(args).Computed())
}

// WireDependencies: the state holds only what the program set.
func (r *ConversationsServiceConfiguration) WireDependencies(f infer.FieldSelector, args *ConversationsServiceConfigurationArgs, state *ConversationsServiceConfigurationState) {
	wireEchoes(f, args, state)
}

// WireDependencies: the state holds only what the program set.
func (r *ConversationsWebhookConfiguration) WireDependencies(f infer.FieldSelector, args *ConversationsWebhookConfigurationArgs, state *ConversationsWebhookConfigurationState) {
	wireEchoes(f, args, state)
}

// WireDependencies: the state holds only what the program set.
func (r *ConversationsServiceNotificationConfiguration) WireDependencies(f infer.FieldSelector, args *ConversationsServiceNotificationConfigurationArgs, state *ConversationsServiceNotificationConfigurationState) {
	wireEchoes(f, args, state)
}

// WireDependencies: the state holds only what the program set.
func (r *ConversationsConfiguration) WireDependencies(f infer.FieldSelector, args *ConversationsConfigurationArgs, state *ConversationsConfigurationState) {
	wireEchoes(f, args, state)
}

// WireDependencies: the SID and creation time are fixed at creation; the update time is what Twilio reports.
func (r *SyncService) WireDependencies(f infer.FieldSelector, args *SyncServiceArgs, state *SyncServiceState) {
	wireEchoes(f, args, state)
	f.OutputField(&state.DateUpdated).DependsOn(f.InputField(args).Computed())
}

// WireDependencies: the SID and creation time are fixed at creation; the update time is what Twilio reports.
func (r *NotifyService) WireDependencies(f infer.FieldSelector, args *NotifyServiceArgs, state *NotifyServiceState) {
	wireEchoes(f, args, state)
	f.OutputField(&state.DateUpdated).DependsOn(f.InputField(args).Computed())
}

// WireDependencies: the SID and creation time are fixed at creation; the update time is what Twilio reports.
func (r *NotifyCredential) WireDependencies(f infer.FieldSelector, args *NotifyCredentialArgs, state *NotifyCredentialState) {
	wireEchoes(f, args, state)
	f.OutputField(&state.DateUpdated).DependsOn(f.InputField(args).Computed())
}

// WireDependencies: the SID and creation time are fixed at creation; the update time and the data logging flag are what Twilio reports.
func (r *IntelligenceService) WireDependencies(f infer.FieldSelector, args *IntelligenceServiceArgs, state *IntelligenceServiceState) {
	wireEchoes(f, args, state)
	f.OutputField(&state.DateUpdated).DependsOn(f.InputField(args).Computed())
	f.OutputField(&state.DataLogging).DependsOn(f.InputField(args).Computed())
}

// WireDependencies: the SID and creation time are fixed at creation; the update time and what Twilio reports about the operator follow its configuration.
func (r *IntelligenceCustomOperator) WireDependencies(f infer.FieldSelector, args *IntelligenceCustomOperatorArgs, state *IntelligenceCustomOperatorState) {
	wireEchoes(f, args, state)
	f.OutputField(&state.DateUpdated).DependsOn(f.InputField(args).Computed())
	f.OutputField(&state.Description).DependsOn(f.InputField(args).Computed())
	f.OutputField(&state.Availability).DependsOn(f.InputField(args).Computed())
}

// WireDependencies: the SID and creation time are fixed at creation; the update time is what Twilio reports.
func (r *VideoCompositionHook) WireDependencies(f infer.FieldSelector, args *VideoCompositionHookArgs, state *VideoCompositionHookState) {
	wireEchoes(f, args, state)
	f.OutputField(&state.DateUpdated).DependsOn(f.InputField(args).Computed())
}

// WireDependencies: the SID and creation time are fixed at creation; the update time is what Twilio reports.
func (r *ProxyService) WireDependencies(f infer.FieldSelector, args *ProxyServiceArgs, state *ProxyServiceState) {
	wireEchoes(f, args, state)
	f.OutputField(&state.DateUpdated).DependsOn(f.InputField(args).Computed())
}

// WireDependencies: the number's identifiers, name and country are fixed once it is in the pool;
// the reservation, use count and update time follow the settings.
func (r *ProxyPhoneNumber) WireDependencies(f infer.FieldSelector, args *ProxyPhoneNumberArgs, state *ProxyPhoneNumberState) {
	wireEchoes(f, args, state)
	f.OutputField(&state.DateUpdated).DependsOn(f.InputField(args).Computed())
	f.OutputField(&state.InUse).DependsOn(f.InputField(args).Computed())
}

// WireDependencies: the settings are a singleton with no identifiers or timestamps of its own; every output echoes an input.
func (r *VideoCompositionSettings) WireDependencies(f infer.FieldSelector, args *VideoCompositionSettingsArgs, state *VideoCompositionSettingsState) {
	wireEchoes(f, args, state)
}

// WireDependencies: the settings are a singleton with no identifiers or timestamps of its own; every output echoes an input.
func (r *VideoRecordingSettings) WireDependencies(f infer.FieldSelector, args *VideoRecordingSettingsArgs, state *VideoRecordingSettingsState) {
	wireEchoes(f, args, state)
}

// WireDependencies: the SID and creation time are fixed at creation; the update time is what Twilio reports.
func (r *AwsCredential) WireDependencies(f infer.FieldSelector, args *AwsCredentialArgs, state *AwsCredentialState) {
	wireEchoes(f, args, state)
	f.OutputField(&state.DateUpdated).DependsOn(f.InputField(args).Computed())
}

// WireDependencies: the SID and creation time are fixed at creation; the update time is what Twilio reports.
func (r *PublicKeyCredential) WireDependencies(f infer.FieldSelector, args *PublicKeyCredentialArgs, state *PublicKeyCredentialState) {
	wireEchoes(f, args, state)
	f.OutputField(&state.DateUpdated).DependsOn(f.InputField(args).Computed())
}

// WireDependencies: the entry SID is fixed when the number is added.
func (r *AccountSafeListNumber) WireDependencies(f infer.FieldSelector, args *AccountSafeListNumberArgs, state *AccountSafeListNumberState) {
	wireEchoes(f, args, state)
}

// WireDependencies: every input replaces the app and every identifier and the secret are fixed at creation.
func (r *OAuthApp) WireDependencies(f infer.FieldSelector, args *OAuthAppArgs, state *OAuthAppState) {
	wireEchoes(f, args, state)
}

// WireDependencies: every input identifies the pool member, so the outputs are fixed once it is added.
func (r *MessagingServiceShortCode) WireDependencies(f infer.FieldSelector, args *MessagingServiceShortCodeArgs, state *MessagingServiceShortCodeState) {
	wireEchoes(f, args, state)
}

// WireDependencies: every input identifies the pool member, so the outputs are fixed once it is added.
func (r *MessagingServiceChannelSender) WireDependencies(f infer.FieldSelector, args *MessagingServiceChannelSenderArgs, state *MessagingServiceChannelSenderState) {
	wireEchoes(f, args, state)
}

// WireDependencies: every input identifies the pool member, so the outputs are fixed once it is added.
func (r *MessagingServiceAlphaSender) WireDependencies(f infer.FieldSelector, args *MessagingServiceAlphaSenderArgs, state *MessagingServiceAlphaSenderState) {
	wireEchoes(f, args, state)
}

// WireDependencies: every input identifies the pool member, so the outputs are fixed once it is added.
func (r *MessagingServiceDestinationAlphaSender) WireDependencies(f infer.FieldSelector, args *MessagingServiceDestinationAlphaSenderArgs, state *MessagingServiceDestinationAlphaSenderState) {
	wireEchoes(f, args, state)
}

// WireDependencies: the SID, owner and creation time are fixed at creation; the update time is what Twilio reports.
func (r *ContentTemplate) WireDependencies(f infer.FieldSelector, args *ContentTemplateArgs, state *ContentTemplateState) {
	wireEchoes(f, args, state)
	f.OutputField(&state.DateUpdated).DependsOn(f.InputField(args).Computed())
}

// WireDependencies: the SID and URL are fixed at creation; status, the pending display name, quality figures and offline reasons are what Twilio reports.
func (r *ChannelsSender) WireDependencies(f infer.FieldSelector, args *ChannelsSenderArgs, state *ChannelsSenderState) {
	wireEchoes(f, args, state)
	for _, o := range []any{&state.Status, &state.PendingDisplayName, &state.PendingDisplayNameStatus, &state.PendingDisplayNameStatusDate, &state.Properties, &state.OfflineReasons} {
		f.OutputField(o).DependsOn(f.InputField(args).Computed())
	}
}

// WireDependencies: the SID and creation time are fixed at creation; the update time is what Twilio reports.
func (r *SuperSimNetworkAccessProfile) WireDependencies(f infer.FieldSelector, args *SuperSimNetworkAccessProfileArgs, state *SuperSimNetworkAccessProfileState) {
	wireEchoes(f, args, state)
	f.OutputField(&state.DateUpdated).DependsOn(f.InputField(args).Computed())
}

// WireDependencies: the SID and creation time are fixed at creation; the profile SID, metering and
// update time are what Twilio reports for the current inputs.
func (r *SuperSimFleet) WireDependencies(f infer.FieldSelector, args *SuperSimFleetArgs, state *SuperSimFleetState) {
	wireEchoes(f, args, state)
	f.OutputField(&state.NetworkAccessProfileSid).DependsOn(f.InputField(&args.NetworkAccessProfile).Computed())
	f.OutputField(&state.DateUpdated).DependsOn(f.InputField(args).Computed())
}

// WireDependencies: the SID and creation time are fixed at creation; the update time is what Twilio reports.
func (r *WirelessRatePlan) WireDependencies(f infer.FieldSelector, args *WirelessRatePlanArgs, state *WirelessRatePlanState) {
	wireEchoes(f, args, state)
	f.OutputField(&state.DateUpdated).DependsOn(f.InputField(args).Computed())
}

// WireDependencies: the SID and creation time are fixed at creation; the update time is what Twilio reports.
func (r *FlexFlow) WireDependencies(f infer.FieldSelector, args *FlexFlowArgs, state *FlexFlowState) {
	wireEchoes(f, args, state)
	f.OutputField(&state.DateUpdated).DependsOn(f.InputField(args).Computed())
}

// WireDependencies: the SID and creation time are fixed at creation; the update time is what Twilio reports.
func (r *FlexPlugin) WireDependencies(f infer.FieldSelector, args *FlexPluginArgs, state *FlexPluginState) {
	wireEchoes(f, args, state)
	f.OutputField(&state.DateUpdated).DependsOn(f.InputField(args).Computed())
}

// WireDependencies: a Plugin Version never changes, so everything Twilio reports is fixed at creation.
func (r *FlexPluginVersion) WireDependencies(f infer.FieldSelector, args *FlexPluginVersionArgs, state *FlexPluginVersionState) {
	wireEchoes(f, args, state)
	f.OutputField(&state.Validated).DependsOn(f.InputField(args).Computed())
}

// WireDependencies: a Configuration never changes, so everything Twilio reports is fixed at creation.
func (r *FlexPluginConfiguration) WireDependencies(f infer.FieldSelector, args *FlexPluginConfigurationArgs, state *FlexPluginConfigurationState) {
	wireEchoes(f, args, state)
}

// WireDependencies: the Configuration SID Twilio reports follows from the configurationId input.
func (r *FlexPluginRelease) WireDependencies(f infer.FieldSelector, args *FlexPluginReleaseArgs, state *FlexPluginReleaseState) {
	wireEchoes(f, args, state)
	f.OutputField(&state.ConfigurationSid).DependsOn(f.InputField(&args.ConfigurationId).Computed())
}

// WireDependencies: the registration status, scores and timestamps are what Twilio and The Campaign Registry report.
func (r *BrandRegistration) WireDependencies(f infer.FieldSelector, args *BrandRegistrationArgs, state *BrandRegistrationState) {
	wireEchoes(f, args, state)
	f.OutputField(&state.Status).DependsOn(f.InputField(args).Computed())
}

// WireDependencies: the campaign status, rate limits and timestamps are what Twilio and The Campaign Registry report.
func (r *UsAppToPerson) WireDependencies(f infer.FieldSelector, args *UsAppToPersonArgs, state *UsAppToPersonState) {
	wireEchoes(f, args, state)
	f.OutputField(&state.CampaignStatus).DependsOn(f.InputField(args).Computed())
	f.OutputField(&state.DateUpdated).DependsOn(f.InputField(args).Computed())
}

// WireDependencies: the status, the rejection details and the timestamps are what Twilio reports during the review.
func (r *TollfreeVerification) WireDependencies(f infer.FieldSelector, args *TollfreeVerificationArgs, state *TollfreeVerificationState) {
	wireEchoes(f, args, state)
	f.OutputField(&state.Status).DependsOn(f.InputField(args).Computed())
	f.OutputField(&state.DateUpdated).DependsOn(f.InputField(args).Computed())
}

// WireDependencies: the configuration SID and creation time are fixed; the update time is what Twilio reports.
func (r *LinkShorteningDomainConfig) WireDependencies(f infer.FieldSelector, args *LinkShorteningDomainConfigArgs, state *LinkShorteningDomainConfigState) {
	wireEchoes(f, args, state)
	f.OutputField(&state.DateUpdated).DependsOn(f.InputField(args).Computed())
}

// WireDependencies: the expiry, the validation status and the update time follow the uploaded certificate.
func (r *LinkShorteningDomainCertificate) WireDependencies(f infer.FieldSelector, args *LinkShorteningDomainCertificateArgs, state *LinkShorteningDomainCertificateState) {
	wireEchoes(f, args, state)
	f.OutputField(&state.CertificateSid).DependsOn(f.InputField(args).Computed())
	f.OutputField(&state.DateExpires).DependsOn(f.InputField(args).Computed())
	f.OutputField(&state.CertInValidation).DependsOn(f.InputField(args).Computed())
	f.OutputField(&state.DateUpdated).DependsOn(f.InputField(args).Computed())
}

// WireDependencies: an association never changes, so everything Twilio reports is fixed at creation.
func (r *LinkShorteningMessagingServiceDomain) WireDependencies(f infer.FieldSelector, args *LinkShorteningMessagingServiceDomainArgs, state *LinkShorteningMessagingServiceDomainState) {
	wireEchoes(f, args, state)
}

// WireDependencies: the SID, account and creation time are fixed; the update time is what Twilio reports.
func (r *TrustHubEndUser) WireDependencies(f infer.FieldSelector, args *TrustHubEndUserArgs, state *TrustHubEndUserState) {
	wireEchoes(f, args, state)
	f.OutputField(&state.DateUpdated).DependsOn(f.InputField(args).Computed())
}

// WireDependencies: the status, MIME type and update time are what Twilio reports.
func (r *TrustHubSupportingDocument) WireDependencies(f infer.FieldSelector, args *TrustHubSupportingDocumentArgs, state *TrustHubSupportingDocumentState) {
	wireEchoes(f, args, state)
	f.OutputField(&state.MimeType).DependsOn(f.InputField(args).Computed())
	f.OutputField(&state.Status).DependsOn(f.InputField(args).Computed())
	f.OutputField(&state.DateUpdated).DependsOn(f.InputField(args).Computed())
}
