package harnessruntime

import "mcp-digitalocean/pkg/registry/common"

var agentConfigListOut = common.NewObjectOutput[ListAgentConfigsResponse]()
var agentConfigOut = common.NewObjectOutput[AgentConfigResponse]()

var sessionCreateOut = common.NewObjectOutput[CreateSessionResponse]()
var sessionOut = common.NewObjectOutput[GetSessionResponse]()
var sessionListOut = common.NewObjectOutput[ListSessionsResponse]()
var sendInputOut = common.NewObjectOutput[SendInputResponse]()
var forkSessionOut = common.NewObjectOutput[ForkSessionResponse]()
var execInSandboxOut = common.NewObjectOutput[ExecInSandboxResponse]()
var ingressURLOut = common.NewObjectOutput[GetIngressURLResponse]()
var cancelTurnOut = common.NewObjectOutput[CancelTurnResponse]()

var sandboxSizesOut = common.NewObjectOutput[ListSandboxSizesResponse]()
var validatePolicyOut = common.NewObjectOutput[ValidatePolicyResponse]()

var checkpointCreateOut = common.NewObjectOutput[CreateCheckpointResponse]()
var checkpointOut = common.NewObjectOutput[GetCheckpointResponse]()
var checkpointListOut = common.NewObjectOutput[ListCheckpointsResponse]()
var checkpointDeleteOut = common.NewObjectOutput[DeleteCheckpointResponse]()
var rollbackOut = common.NewObjectOutput[RollbackToCheckpointResponse]()

var templateListOut = common.NewObjectOutput[ListTemplatesResponse]()
var templateOut = common.NewObjectOutput[GetTemplateResponse]()
var templateCreateOut = common.NewObjectOutput[CreateTemplateResponse]()
var templateDeleteOut = common.NewObjectOutput[DeleteTemplateResponse]()
var templateBuildListOut = common.NewObjectOutput[ListTemplateBuildsResponse]()
var templateBuildOut = common.NewObjectOutput[GetTemplateBuildResponse]()

var workspaceCreateOut = common.NewObjectOutput[CreateWorkspaceResponse]()
var workspaceOut = common.NewObjectOutput[GetWorkspaceResponse]()
var workspaceListOut = common.NewObjectOutput[ListWorkspacesResponse]()
var wsTransferCreateOut = common.NewObjectOutput[WorkspaceTransferCreated]()
var wsTransferStatusOut = common.NewObjectOutput[WorkspaceTransferStatus]()
var wsTransferCancelOut = common.NewObjectOutput[WorkspaceTransferCancelled]()
var wsUploadCommitOut = common.NewObjectOutput[WorkspaceUploadCommitted]()
var wsPartURLsOut = common.NewObjectOutput[WorkspacePartUploadURLs]()

var triggerOut = common.NewObjectOutput[TriggerResponse]()
var createTriggerOut = common.NewObjectOutput[CreateTriggerResponse]()
var triggerListOut = common.NewObjectOutput[ListTriggersResponse]()
var rotateTriggerSecretOut = common.NewObjectOutput[RotateTriggerSecretResponse]()
var triggerExecutionOut = common.NewObjectOutput[TriggerExecutionResponse]()
var triggerExecutionListOut = common.NewObjectOutput[ListTriggerExecutionsResponse]()
var cancelExecutionOut = common.NewObjectOutput[CancelExecutionResponse]()
var reusableSessionListOut = common.NewObjectOutput[ListReusableSessionsResponse]()

var connectionListOut = common.NewObjectOutput[ListConnectionsResponse]()
var connectionOut = common.NewObjectOutput[ConnectionEnvelope]()
var providerAuthStartOut = common.NewObjectOutput[ProviderAuthStartResponse]()
var providerAuthPollOut = common.NewObjectOutput[ProviderAuthPollResponse]()
var webhookProviderListOut = common.NewObjectOutput[ListWebhookProvidersResponse]()
