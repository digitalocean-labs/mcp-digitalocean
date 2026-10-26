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
