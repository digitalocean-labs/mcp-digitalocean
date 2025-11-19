package networking

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/digitalocean/godo"
	"github.com/mark3labs/mcp-go/mcp"
	"github.com/mark3labs/mcp-go/server"
)

// FirewallTool provides firewall management tools
type FirewallTool struct {
	client func(ctx context.Context) (*godo.Client, error)
}

// NewFirewallTool creates a new firewall tool
func NewFirewallTool(client func(ctx context.Context) (*godo.Client, error)) *FirewallTool {
	return &FirewallTool{
		client: client,
	}
}

// getFirewall fetches firewall information by ID
func (f *FirewallTool) getFirewall(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	id, ok := req.GetArguments()["ID"].(string)
	if !ok || id == "" {
		return mcp.NewToolResultError("Firewall ID is required"), nil
	}

	client, err := f.client(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to get DigitalOcean client: %w", err)
	}

	firewall, _, err := client.Firewalls.Get(ctx, id)
	if err != nil {
		return mcp.NewToolResultErrorFromErr("api error", err), nil
	}
	jsonFirewall, err := json.MarshalIndent(firewall, "", "  ")
	if err != nil {
		return nil, fmt.Errorf("marshal error: %w", err)
	}
	return mcp.NewToolResultText(string(jsonFirewall)), nil
}

// listFirewalls lists firewalls with pagination support
func (f *FirewallTool) listFirewalls(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	page := 1
	perPage := 20
	if v, ok := req.GetArguments()["Page"].(float64); ok && int(v) > 0 {
		page = int(v)
	}
	if v, ok := req.GetArguments()["PerPage"].(float64); ok && int(v) > 0 {
		perPage = int(v)
	}

	client, err := f.client(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to get DigitalOcean client: %w", err)
	}

	firewalls, _, err := client.Firewalls.List(ctx, &godo.ListOptions{Page: page, PerPage: perPage})
	if err != nil {
		return mcp.NewToolResultErrorFromErr("api error", err), nil
	}
	jsonFirewalls, err := json.MarshalIndent(firewalls, "", "  ")
	if err != nil {
		return nil, fmt.Errorf("marshal error: %w", err)
	}
	return mcp.NewToolResultText(string(jsonFirewalls)), nil
}

// createFirewall creates a new firewall
func (f *FirewallTool) createFirewall(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	name := req.GetArguments()["Name"].(string)
	inboundProtocol := req.GetArguments()["InboundProtocol"].(string)
	inboundPortRange := req.GetArguments()["InboundPortRange"].(string)
	inboundSource := req.GetArguments()["InboundSource"].(string)
	outboundProtocol := req.GetArguments()["OutboundProtocol"].(string)
	outboundPortRange := req.GetArguments()["OutboundPortRange"].(string)
	outboundDestination := req.GetArguments()["OutboundDestination"].(string)

	dropletIDs := make([]any, 0)
	if v, ok := req.GetArguments()["DropletIDs"].([]any); ok {
		dropletIDs = v
	}
	tags := make([]any, 0)
	if v, ok := req.GetArguments()["Tags"].([]any); ok {
		tags = v
	}

	dIDs := make([]int, len(dropletIDs))
	for i, v := range dropletIDs {
		if stringID, ok := v.(float64); ok {
			dIDs[i] = int(stringID)
		}
	}

	tagsStr := make([]string, len(tags))
	for i, v := range tags {
		if tag, ok := v.(string); ok {
			tagsStr[i] = tag
		}
	}

	inboundRule := godo.InboundRule{
		Protocol:  inboundProtocol,
		PortRange: inboundPortRange,
		Sources:   &godo.Sources{Addresses: []string{inboundSource}},
	}

	outboundRule := godo.OutboundRule{
		Protocol:     outboundProtocol,
		PortRange:    outboundPortRange,
		Destinations: &godo.Destinations{Addresses: []string{outboundDestination}},
	}

	firewallRequest := &godo.FirewallRequest{
		Name:          name,
		InboundRules:  []godo.InboundRule{inboundRule},
		OutboundRules: []godo.OutboundRule{outboundRule},
		DropletIDs:    dIDs,
		Tags:          tagsStr,
	}

	client, err := f.client(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to get DigitalOcean client: %w", err)
	}

	firewall, _, err := client.Firewalls.Create(ctx, firewallRequest)
	if err != nil {
		return mcp.NewToolResultErrorFromErr("api error", err), nil
	}

	jsonFirewall, err := json.MarshalIndent(firewall, "", "  ")
	if err != nil {
		return nil, fmt.Errorf("marshal error: %w", err)
	}

	return mcp.NewToolResultText(string(jsonFirewall)), nil
}

// deleteFirewall deletes a firewall
func (f *FirewallTool) deleteFirewall(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	firewallID := req.GetArguments()["ID"].(string)

	client, err := f.client(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to get DigitalOcean client: %w", err)
	}

	_, err = client.Firewalls.Delete(ctx, firewallID)
	if err != nil {
		return mcp.NewToolResultErrorFromErr("api error", err), nil
	}
	return mcp.NewToolResultText("Firewall deleted successfully"), nil
}

// addDroplets adds one or more droplet to a firewall
func (f *FirewallTool) addDroplets(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	firewallID := req.GetArguments()["ID"].(string)
	dropletIDs := req.GetArguments()["DropletIDs"].([]any)
	dIDs := make([]int, len(dropletIDs))
	for i, id := range dropletIDs {
		if did, ok := id.(float64); ok {
			dIDs[i] = int(did)
		}
	}

	client, err := f.client(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to get DigitalOcean client: %w", err)
	}

	_, err = client.Firewalls.AddDroplets(ctx, firewallID, dIDs...)
	if err != nil {
		return mcp.NewToolResultErrorFromErr("api error", err), nil
	}
	return mcp.NewToolResultText("Droplet(s) added to firewall successfully"), nil
}

func (f *FirewallTool) removeDroplets(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	firewallID := req.GetArguments()["ID"].(string)
	dropletIDs := req.GetArguments()["DropletIDs"].([]any)
	dIDs := make([]int, len(dropletIDs))
	for i, id := range dropletIDs {
		if did, ok := id.(float64); ok {
			dIDs[i] = int(did)
		}
	}

	client, err := f.client(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to get DigitalOcean client: %w", err)
	}

	_, err = client.Firewalls.RemoveDroplets(ctx, firewallID, dIDs...)
	if err != nil {
		return mcp.NewToolResultErrorFromErr("api error", err), nil
	}
	return mcp.NewToolResultText("Droplet(s) removed from firewall successfully"), nil
}

// addTags adds one or more tags to a firewall
func (f *FirewallTool) addTags(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	firewallID := req.GetArguments()["ID"].(string)
	tagNames := req.GetArguments()["Tags"].([]any)
	tagNamesStr := make([]string, len(tagNames))
	for i, v := range tagNames {
		if tag, ok := v.(string); ok {
			tagNamesStr[i] = tag
		}
	}

	client, err := f.client(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to get DigitalOcean client: %w", err)
	}

	_, err = client.Firewalls.AddTags(ctx, firewallID, tagNamesStr...)
	if err != nil {
		return mcp.NewToolResultErrorFromErr("api error", err), nil
	}
	return mcp.NewToolResultText("Tag(s) added to firewall successfully"), nil
}

// removeTags removes one or more tags from a firewall
func (f *FirewallTool) removeTags(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	firewallID := req.GetArguments()["ID"].(string)
	tagNames := req.GetArguments()["Tags"].([]any)
	tagNamesStr := make([]string, len(tagNames))
	for i, v := range tagNames {
		if tag, ok := v.(string); ok {
			tagNamesStr[i] = tag
		}
	}

	client, err := f.client(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to get DigitalOcean client: %w", err)
	}

	_, err = client.Firewalls.RemoveTags(ctx, firewallID, tagNamesStr...)
	if err != nil {
		return mcp.NewToolResultErrorFromErr("api error", err), nil
	}
	return mcp.NewToolResultText("Tag(s) removed from firewall successfully"), nil
}

// addRules adds one or more rules to a firewall
func (f *FirewallTool) addRules(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	firewallID := req.GetArguments()["ID"].(string)

	var inboundRules []godo.InboundRule
	var outboundRules []godo.OutboundRule

	// Process inbound rules if provided
	if inboundData, ok := req.GetArguments()["InboundRules"]; ok && inboundData != nil {
		inboundRulesList := inboundData.([]any)
		for _, ruleData := range inboundRulesList {
			rule := ruleData.(map[string]any)

			protocol := rule["Protocol"].(string)
			portRange := rule["PortRange"].(string)
			sources := rule["Sources"].([]any)

			sourceAddresses := make([]string, len(sources))
			for i, source := range sources {
				sourceAddresses[i] = source.(string)
			}

			inboundRule := godo.InboundRule{
				Protocol:  protocol,
				PortRange: portRange,
				Sources:   &godo.Sources{Addresses: sourceAddresses},
			}
			inboundRules = append(inboundRules, inboundRule)
		}
	}

	// Process outbound rules if provided
	if outboundData, ok := req.GetArguments()["OutboundRules"]; ok && outboundData != nil {
		outboundRulesList := outboundData.([]any)
		for _, ruleData := range outboundRulesList {
			rule := ruleData.(map[string]any)

			protocol := rule["Protocol"].(string)
			portRange := rule["PortRange"].(string)
			destinations := rule["Destinations"].([]any)

			destAddresses := make([]string, len(destinations))
			for i, dest := range destinations {
				destAddresses[i] = dest.(string)
			}

			outboundRule := godo.OutboundRule{
				Protocol:     protocol,
				PortRange:    portRange,
				Destinations: &godo.Destinations{Addresses: destAddresses},
			}
			outboundRules = append(outboundRules, outboundRule)
		}
	}

	if len(inboundRules) == 0 && len(outboundRules) == 0 {
		return mcp.NewToolResultError("At least one inbound or outbound rule must be provided"), nil
	}

	rulesRequest := &godo.FirewallRulesRequest{
		InboundRules:  inboundRules,
		OutboundRules: outboundRules,
	}

	client, err := f.client(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to get DigitalOcean client: %w", err)
	}

	_, err = client.Firewalls.AddRules(ctx, firewallID, rulesRequest)
	if err != nil {
		return mcp.NewToolResultErrorFromErr("api error", err), nil
	}

	return mcp.NewToolResultText("Rule(s) added to firewall successfully"), nil
}

// removeRules removes one or more rules from a firewall
func (f *FirewallTool) removeRules(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	firewallID := req.GetArguments()["ID"].(string)

	var inboundRules []godo.InboundRule
	var outboundRules []godo.OutboundRule

	// Process inbound rules if provided
	if inboundData, ok := req.GetArguments()["InboundRules"]; ok && inboundData != nil {
		inboundRulesList := inboundData.([]any)
		for _, ruleData := range inboundRulesList {
			rule := ruleData.(map[string]any)

			protocol := rule["Protocol"].(string)
			portRange := rule["PortRange"].(string)
			sources := rule["Sources"].([]any)

			sourceAddresses := make([]string, len(sources))
			for i, source := range sources {
				sourceAddresses[i] = source.(string)
			}

			inboundRule := godo.InboundRule{
				Protocol:  protocol,
				PortRange: portRange,
				Sources:   &godo.Sources{Addresses: sourceAddresses},
			}
			inboundRules = append(inboundRules, inboundRule)
		}
	}

	// Process outbound rules if provided
	if outboundData, ok := req.GetArguments()["OutboundRules"]; ok && outboundData != nil {
		outboundRulesList := outboundData.([]any)
		for _, ruleData := range outboundRulesList {
			rule := ruleData.(map[string]any)

			protocol := rule["Protocol"].(string)
			portRange := rule["PortRange"].(string)
			destinations := rule["Destinations"].([]any)

			destAddresses := make([]string, len(destinations))
			for i, dest := range destinations {
				destAddresses[i] = dest.(string)
			}

			outboundRule := godo.OutboundRule{
				Protocol:     protocol,
				PortRange:    portRange,
				Destinations: &godo.Destinations{Addresses: destAddresses},
			}
			outboundRules = append(outboundRules, outboundRule)
		}
	}

	if len(inboundRules) == 0 && len(outboundRules) == 0 {
		return mcp.NewToolResultError("At least one inbound or outbound rule must be provided"), nil
	}

	rulesRequest := &godo.FirewallRulesRequest{
		InboundRules:  inboundRules,
		OutboundRules: outboundRules,
	}

	client, err := f.client(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to get DigitalOcean client: %w", err)
	}

	_, err = client.Firewalls.RemoveRules(ctx, firewallID, rulesRequest)
	if err != nil {
		return mcp.NewToolResultErrorFromErr("api error", err), nil
	}

	return mcp.NewToolResultText("Rule(s) removed from firewall successfully"), nil
}

// Tools returns a list of tool functions
func (f *FirewallTool) Tools() []server.ServerTool {
	return []server.ServerTool{
		{
			Handler: f.getFirewall,
			Tool: mcp.NewTool("firewall-get",
				mcp.WithDescription("Get firewall information by ID"),
				mcp.WithString("ID", mcp.Required(), mcp.Description("ID of the firewall")),
			),
		},
		{
			Handler: f.listFirewalls,
			Tool: mcp.NewTool("firewall-list",
				mcp.WithDescription("List firewalls with pagination"),
				mcp.WithNumber("Page", mcp.DefaultNumber(1), mcp.Description("Page number")),
				mcp.WithNumber("PerPage", mcp.DefaultNumber(20), mcp.Description("Items per page")),
			),
		},
		{
			Handler: f.createFirewall,
			Tool: mcp.NewTool("firewall-create",
				mcp.WithDescription("Create a new firewall"),
				mcp.WithString("Name", mcp.Required(), mcp.Description("Name of the firewall")),
				mcp.WithString("InboundProtocol", mcp.Required(), mcp.Description("Protocol for inbound rule")),
				mcp.WithString("InboundPortRange", mcp.Required(), mcp.Description("Port range for inbound rule")),
				mcp.WithString("InboundSource", mcp.Required(), mcp.Description("Source address for inbound rule")),
				mcp.WithString("OutboundProtocol", mcp.Required(), mcp.Description("Protocol for outbound rule")),
				mcp.WithString("OutboundPortRange", mcp.Required(), mcp.Description("Port range for outbound rule")),
				mcp.WithString("OutboundDestination", mcp.Required(), mcp.Description("Destination address for outbound rule")),
				mcp.WithArray("DropletIDs", mcp.Description("Droplet IDs to apply the firewall to"), mcp.Items(map[string]any{
					"type":        "number",
					"description": "droplet ID to apply the firewall to",
				})),
				mcp.WithArray("Tags", mcp.Description("Tags to apply the firewall to"), mcp.Items(map[string]any{
					"type":        "string",
					"description": "Tag to apply",
				})),
			),
		},
		{
			Handler: f.deleteFirewall,
			Tool: mcp.NewTool("firewall-delete",
				mcp.WithDescription("Delete a firewall"),
				mcp.WithString("ID", mcp.Required(), mcp.Description("ID of the firewall to delete")),
			),
		},
		{
			Handler: f.addDroplets,
			Tool: mcp.NewTool("firewall-add-droplets",
				mcp.WithDescription("Adds one or more droplets to a firewall"),
				mcp.WithString("ID", mcp.Required(), mcp.Description("ID of the firewall to apply to droplets")),
				mcp.WithArray("DropletIDs", mcp.Required(), mcp.Description("Droplet IDs to apply the firewall to"), mcp.Items(map[string]any{
					"type":        "number",
					"description": "droplet ID to apply the firewall to",
				})),
			),
		},
		{
			Handler: f.addTags,
			Tool: mcp.NewTool("firewall-add-tags",
				mcp.WithDescription("Adds one or more tags to a firewall"),
				mcp.WithString("ID", mcp.Required(), mcp.Description("ID of the firewall to update tags")),
				mcp.WithArray("Tags", mcp.Required(), mcp.Description("Tags to apply the firewall to"), mcp.Items(map[string]any{
					"type":        "string",
					"description": "Tag to apply",
				})),
			),
		},

		{
			Handler: f.removeDroplets,
			Tool: mcp.NewTool("firewall-remove-droplets",
				mcp.WithDescription("Removes one or more droplets from a firewall"),
				mcp.WithString("ID", mcp.Required(), mcp.Description("ID of the firewall to remove droplets from")),
				mcp.WithArray("DropletIDs", mcp.Required(), mcp.Description("Droplet IDs to remove from the firewall"), mcp.Items(map[string]any{
					"type":        "number",
					"description": "droplet ID to remove from the firewall",
				})),
			),
		},
		{
			Handler: f.removeTags,
			Tool: mcp.NewTool("firewall-remove-tags",
				mcp.WithDescription("Removes one or more tags from a firewall"),
				mcp.WithString("ID", mcp.Required(), mcp.Description("ID of the firewall to update tags")),
				mcp.WithArray("Tags", mcp.Required(), mcp.Description("Tags to remove from the firewall"), mcp.Items(map[string]any{
					"type":        "string",
					"description": "Tag to remove",
				})),
			),
		},
		{
			Handler: f.addRules,
			Tool: mcp.NewTool("firewall-add-rules",
				mcp.WithDescription("Add one or more rules to a firewall"),
				mcp.WithString("ID", mcp.Required(), mcp.Description("ID of the firewall to add rules to")),
				mcp.WithArray("InboundRules", mcp.Description("Inbound rules to add"), mcp.Items(map[string]any{
					"type": "object",
					"properties": map[string]any{
						"Protocol": map[string]any{
							"type":        "string",
							"description": "Protocol (tcp, udp, icmp)",
						},
						"PortRange": map[string]any{
							"type":        "string",
							"description": "Port range (e.g., '80', '443', '8000-8080')",
						},
						"Sources": map[string]any{
							"type": "array",
							"items": map[string]any{
								"type":        "string",
								"description": "Source IP address or CIDR block",
							},
							"description": "List of source addresses",
						},
					},
					"required":    []string{"Protocol", "PortRange", "Sources"},
					"description": "Inbound firewall rule",
				})),
				mcp.WithArray("OutboundRules", mcp.Description("Outbound rules to add"), mcp.Items(map[string]any{
					"type": "object",
					"properties": map[string]any{
						"Protocol": map[string]any{
							"type":        "string",
							"description": "Protocol (tcp, udp, icmp)",
						},
						"PortRange": map[string]any{
							"type":        "string",
							"description": "Port range (e.g., '80', '443', '8000-8080')",
						},
						"Destinations": map[string]any{
							"type": "array",
							"items": map[string]any{
								"type":        "string",
								"description": "Destination IP address or CIDR block",
							},
							"description": "List of destination addresses",
						},
					},
					"required":    []string{"Protocol", "PortRange", "Destinations"},
					"description": "Outbound firewall rule",
				})),
			),
		},
		{
			Handler: f.removeRules,
			Tool: mcp.NewTool("firewall-remove-rules",
				mcp.WithDescription("Remove one or more rules from a firewall"),
				mcp.WithString("ID", mcp.Required(), mcp.Description("ID of the firewall to remove rules from")),
				mcp.WithArray("InboundRules", mcp.Description("Inbound rules to remove"), mcp.Items(map[string]any{
					"type": "object",
					"properties": map[string]any{
						"Protocol": map[string]any{
							"type":        "string",
							"description": "Protocol (tcp, udp, icmp)",
						},
						"PortRange": map[string]any{
							"type":        "string",
							"description": "Port range (e.g., '80', '443', '8000-8080')",
						},
						"Sources": map[string]any{
							"type": "array",
							"items": map[string]any{
								"type":        "string",
								"description": "Source IP address or CIDR block",
							},
							"description": "List of source addresses",
						},
					},
					"required":    []string{"Protocol", "PortRange", "Sources"},
					"description": "Inbound firewall rule",
				})),
				mcp.WithArray("OutboundRules", mcp.Description("Outbound rules to remove"), mcp.Items(map[string]any{
					"type": "object",
					"properties": map[string]any{
						"Protocol": map[string]any{
							"type":        "string",
							"description": "Protocol (tcp, udp, icmp)",
						},
						"PortRange": map[string]any{
							"type":        "string",
							"description": "Port range (e.g., '80', '443', '8000-8080')",
						},
						"Destinations": map[string]any{
							"type": "array",
							"items": map[string]any{
								"type":        "string",
								"description": "Destination IP address or CIDR block",
							},
							"description": "List of destination addresses",
						},
					},
					"required":    []string{"Protocol", "PortRange", "Destinations"},
					"description": "Outbound firewall rule",
				})),
			),
		},
	}
}
