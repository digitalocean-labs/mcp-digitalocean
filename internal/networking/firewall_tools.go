package networking

import (
	"context"
	"encoding/json"

	"github.com/digitalocean/godo"
	"github.com/mark3labs/mcp-go/mcp"
	"github.com/mark3labs/mcp-go/server"
)

// FirewallTool provides firewall management tools
type FirewallTool struct {
	client *godo.Client
}

// NewFirewallTool creates a new firewall tool
func NewFirewallTool(client *godo.Client) *FirewallTool {
	return &FirewallTool{
		client: client,
	}
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
	dropletIDs := req.GetArguments()["DropletIDs"].([]float64)
	tags := req.GetArguments()["Tags"].([]string)

	dIDs := make([]int, len(dropletIDs))
	for i, v := range dropletIDs {
		dIDs[i] = int(v)
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
		Tags:          tags,
	}

	firewall, _, err := f.client.Firewalls.Create(ctx, firewallRequest)
	if err != nil {
		return mcp.NewToolResultErrorFromErr("api error", err), nil
	}

	jsonFirewall, err := json.MarshalIndent(firewall, "", "  ")
	if err != nil {
		return mcp.NewToolResultErrorFromErr("marshal error", err), nil
	}

	return mcp.NewToolResultText(string(jsonFirewall)), nil
}

// deleteFirewall deletes a firewall
func (f *FirewallTool) deleteFirewall(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	firewallID := req.GetArguments()["ID"].(string)
	_, err := f.client.Firewalls.Delete(ctx, firewallID)
	if err != nil {
		return mcp.NewToolResultErrorFromErr("api error", err), nil
	}
	return mcp.NewToolResultText("Firewall deleted successfully"), nil
}

// addDroplets adds one or more droplet to a firewall
func (f *FirewallTool) addDroplets(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	firewallID := req.GetArguments()["ID"].(string)
	dropletIDs := req.GetArguments()["DropletIDs"].([]float64)
	dIDs := make([]int, len(dropletIDs))
	for i, id := range dropletIDs {
		dIDs[i] = int(id)
	}
	_, err := f.client.Firewalls.AddDroplets(ctx, firewallID, dIDs...)
	if err != nil {
		return mcp.NewToolResultErrorFromErr("api error", err), nil
	}
	return mcp.NewToolResultText("Droplet(s) added to firewall successfully"), nil
}

func (f *FirewallTool) removeDroplets(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	firewallID := req.GetArguments()["ID"].(string)
	dropletIDs := req.GetArguments()["DropletIDs"].([]float64)
	dIDs := make([]int, len(dropletIDs))
	for i, id := range dropletIDs {
		dIDs[i] = int(id)
	}
	_, err := f.client.Firewalls.RemoveDroplets(ctx, firewallID, dIDs...)
	if err != nil {
		return mcp.NewToolResultErrorFromErr("api error", err), nil
	}
	return mcp.NewToolResultText("Droplet(s) removed from firewall successfully"), nil
}

// addTags adds one or more tags to a firewall
func (f *FirewallTool) addTags(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	firewallID := req.GetArguments()["ID"].(string)
	tagNames := req.GetArguments()["Tags"].([]string)
	_, err := f.client.Firewalls.AddTags(ctx, firewallID, tagNames...)
	if err != nil {
		return mcp.NewToolResultErrorFromErr("api error", err), nil
	}
	return mcp.NewToolResultText("Tag(s) added to firewall successfully"), nil
}

// removeTags removes one or more tags from a firewall
func (f *FirewallTool) removeTags(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	firewallID := req.GetArguments()["ID"].(string)
	tagNames := req.GetArguments()["Tags"].([]string)
	_, err := f.client.Firewalls.RemoveTags(ctx, firewallID, tagNames...)
	if err != nil {
		return mcp.NewToolResultErrorFromErr("api error", err), nil
	}
	return mcp.NewToolResultText("Tag(s) removed from firewall successfully"), nil
}

// Tools returns a list of tool functions
func (f *FirewallTool) Tools() []server.ServerTool {
	return []server.ServerTool{
		{
			Handler: f.createFirewall,
			Tool: mcp.NewTool("digitalocean-firewall-create",
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
			Tool: mcp.NewTool("digitalocean-firewall-delete",
				mcp.WithDescription("Delete a firewall"),
				mcp.WithString("ID", mcp.Required(), mcp.Description("ID of the firewall to delete")),
			),
		},
		{
			Handler: f.addDroplets,
			Tool: mcp.NewTool("digitalocean-firewall-add-droplets",
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
			Tool: mcp.NewTool("digitalocean-firewall-add-tags",
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
			Tool: mcp.NewTool("digitalocean-firewall-remove-droplets",
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
			Tool: mcp.NewTool("digitalocean-firewall-remove-tags",
				mcp.WithDescription("Removes one or more tags from a firewall"),
				mcp.WithString("ID", mcp.Required(), mcp.Description("ID of the firewall to update tags")),
				mcp.WithArray("Tags", mcp.Required(), mcp.Description("Tags to remove from the firewall"), mcp.Items(map[string]any{
					"type":        "string",
					"description": "Tag to remove",
				})),
			),
		},
	}
}
