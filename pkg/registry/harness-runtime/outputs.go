package harnessruntime

import "mcp-digitalocean/pkg/registry/common"

var agentConfigListOut = common.NewObjectOutput[ListAgentConfigsResponse]()
var agentConfigOut = common.NewObjectOutput[AgentConfigResponse]()

var sessionCreateOut = common.NewObjectOutput[CreateSessionResponse]()
var sessionOut = common.NewObjectOutput[GetSessionResponse]()
var sessionListOut = common.NewObjectOutput[ListSessionsResponse]()
var sendInputOut = common.NewObjectOutput[SendInputResponse]()

var sandboxSizesOut = common.NewObjectOutput[ListSandboxSizesResponse]()
var validatePolicyOut = common.NewObjectOutput[ValidatePolicyResponse]()
