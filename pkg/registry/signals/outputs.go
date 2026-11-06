package signals

import (
	"encoding/json"
	"fmt"

	"github.com/digitalocean/godo"
	"github.com/mark3labs/mcp-go/mcp"

	"mcp-digitalocean/pkg/registry/common"
)

var (
	consentOut          = common.NewObjectOutput[*godo.SignalsAgentConsent]()
	consentSetOut       = common.NewObjectOutput[*godo.SignalsConsentRecord]()
	inferenceConsentOut = common.NewObjectOutput[*godo.SignalsConsentRecord]()
	consentsListOut     = common.NewObjectOutput[*godo.SignalsListConsentsResponse]()
	sessionsOut         = common.NewObjectOutput[*godo.SignalsListAgentSessionsResponse]()
	dialoguesOut        = common.NewObjectOutput[*godo.SignalsSessionDialoguesResponse]()
	exportsOut          = common.NewObjectOutput[*godo.SignalsListExportsResponse]()
	exportJobOut        = common.NewObjectOutput[*godo.SignalsExportJob]()
	exportDownloadOut   = common.NewObjectOutput[*godo.SignalsExportDownload]()
	exportOptionsOut    = common.NewObjectOutput[*godo.SignalsExportOptions]()
)

func resultDualEmit[T any](o *common.Output[T], payload T) (*mcp.CallToolResult, error) {
	raw, err := dualEmitEnvironmentIDJSON(payload)
	if err != nil {
		return nil, fmt.Errorf("marshal error: %w", err)
	}
	return o.ResultRaw(string(raw), raw), nil
}

func dualEmitEnvironmentIDJSON(v any) (json.RawMessage, error) {
	b, err := json.MarshalIndent(v, "", "  ")
	if err != nil {
		return nil, err
	}
	var tree any
	if err := json.Unmarshal(b, &tree); err != nil {
		return nil, err
	}
	injectEnvironmentID(tree)
	return json.MarshalIndent(tree, "", "  ")
}

func injectEnvironmentID(node any) {
	switch n := node.(type) {
	case map[string]any:
		if agentID, ok := n["agent_id"].(string); ok && agentID != "" {
			if _, has := n["environment_id"]; !has {
				n["environment_id"] = agentID
			}
		}
		for _, child := range n {
			injectEnvironmentID(child)
		}
	case []any:
		for _, child := range n {
			injectEnvironmentID(child)
		}
	}
}
