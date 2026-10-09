package harnessruntime

import "mcp-digitalocean/pkg/registry/common"

var agentConfigListOut = common.NewObjectOutput[ListAgentConfigsResponse]()
var agentConfigOut = common.NewObjectOutput[AgentConfigResponse]()
