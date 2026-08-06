package dbaas

import (
	"context"
	"encoding/json"
	"fmt"
	"mcp-digitalocean/pkg/response"

	"github.com/digitalocean/godo"
	"github.com/mark3labs/mcp-go/mcp"
	"github.com/mark3labs/mcp-go/server"
)

type FirewallTool struct {
	client func(ctx context.Context) (*godo.Client, error)
}

func NewFirewallTool(client func(ctx context.Context) (*godo.Client, error)) *FirewallTool {
	return &FirewallTool{
		client: client,
	}
}

func (s *FirewallTool) getFirewallRules(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	args := req.GetArguments()
	id, ok := args["id"].(string)
	if !ok || id == "" {
		return mcp.NewToolResultError("Cluster id is required"), nil
	}

	client, err := s.client(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to get DigitalOcean client: %w", err)
	}

	rules, _, err := client.Databases.GetFirewallRules(ctx, id)
	if err != nil {
		return mcp.NewToolResultErrorFromErr("api error", err), nil
	}

	jsonRules, err := response.CompactJSON(rules)
	if err != nil {
		return nil, fmt.Errorf("marshal error: %w", err)
	}

	return mcp.NewToolResultText(jsonRules), nil
}

func (s *FirewallTool) updateFirewallRules(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	args := req.GetArguments()
	id, ok := args["id"].(string)
	if !ok || id == "" {
		return mcp.NewToolResultError("Cluster id is required"), nil
	}

	rawRules, ok := args["rules"].([]any)
	if !ok {
		return mcp.NewToolResultError("Missing or invalid 'rules' array object"), nil
	}

	var rules []*godo.DatabaseFirewallRule
	for _, r := range rawRules {
		ruleMap, ok := r.(map[string]any)
		if !ok {
			return mcp.NewToolResultError("Each rule must be an object"), nil
		}

		ruleBytes, err := json.Marshal(ruleMap)
		if err != nil {
			return nil, fmt.Errorf("marshal error: %w", err)
		}

		var rule godo.DatabaseFirewallRule
		if err := json.Unmarshal(ruleBytes, &rule); err != nil {
			return mcp.NewToolResultError("Invalid rule: " + err.Error()), nil
		}

		rules = append(rules, &rule)
	}

	updateReq := &godo.DatabaseUpdateFirewallRulesRequest{Rules: rules}
	client, err := s.client(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to get DigitalOcean client: %w", err)
	}
	_, err = client.Databases.UpdateFirewallRules(ctx, id, updateReq)
	if err != nil {
		return mcp.NewToolResultErrorFromErr("api error", err), nil
	}

	return mcp.NewToolResultText("Firewall rules updated successfully"), nil
}

func (s *FirewallTool) Tools() []server.ServerTool {
	return []server.ServerTool{
		{
			Handler: s.getFirewallRules,
			Tool: mcp.NewTool("db-cluster-get-firewall-rules",
				mcp.WithDescription("Get firewall rules for a database cluster."),
				mcp.WithString("id", mcp.Required(), mcp.Description("The cluster UUID")),
			),
		},
		{
			Handler: s.updateFirewallRules,
			Tool: mcp.NewTool("db-cluster-update-firewall-rules",
				mcp.WithDescription("Update firewall rules for a cluster using a structured list of rules."),
				mcp.WithString("id", mcp.Required(), mcp.Description("The cluster UUID")),
				mcp.WithArray("rules",
					mcp.Items(map[string]any{
						"type": "object",
						"properties": map[string]any{
							"uuid": map[string]any{
								"type":        "string",
								"description": "Rule UUID (optional when creating new rules)",
							},
							"cluster_uuid": map[string]any{
								"type":        "string",
								"description": "UUID of the cluster the rule belongs to",
							},
							"type": map[string]any{
								"type":        "string",
								"description": "Type of the rule (e.g., ip_addr, droplet, tag, app)",
							},
							"value": map[string]any{
								"type":        "string",
								"description": "Value for the rule (e.g., IP address or tag name)",
							},
						},
					}),
				),
			),
		},
	}
}
