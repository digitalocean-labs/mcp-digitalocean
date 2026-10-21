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
