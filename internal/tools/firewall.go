package tools

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

// CreateFirewall creates a new firewall
func (f *FirewallTool) CreateFirewall(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	name, err := req.RequireString("Name")
	if err != nil {
		return nil, err
	}

	inboundProtocol, err := req.RequireString("InboundProtocol")
	if err != nil {
		return nil, err
	}

	inboundPortRange, err := req.RequireString("InboundPortRange")
	if err != nil {
		return nil, err
	}

	inboundSource, err := req.RequireString("InboundSource")
	if err != nil {
		return nil, err
	}

	outboundProtocol, err := req.RequireString("OutboundProtocol")
	if err != nil {
		return nil, err
	}

	outboundPortRange, err := req.RequireString("OutboundPortRange")
	if err != nil {
		return nil, err
	}

	outboundDestination, err := req.RequireString("OutboundDestination")
	if err != nil {
		return nil, err
	}

	dIDs := req.GetIntSlice("DropletIDs", []int{})
	tags := req.GetStringSlice("Tags", []string{})

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
		return nil, err
	}

	jsonFirewall, err := json.MarshalIndent(firewall, "", "  ")
	if err != nil {
		return nil, err
	}

	return mcp.NewToolResultText(string(jsonFirewall)), nil
}

// DeleteFirewall deletes a firewall
func (f *FirewallTool) DeleteFirewall(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	firewallID, err := req.RequireString("ID")
	if err != nil {
		return nil, err
	}
	_, err = f.client.Firewalls.Delete(ctx, firewallID)
	if err != nil {
		return nil, err
	}
	return mcp.NewToolResultText("Firewall deleted successfully"), nil
}

// Tools returns a list of tool functions
func (f *FirewallTool) Tools() []server.ServerTool {
	return []server.ServerTool{
		{
			Handler: f.CreateFirewall,
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
			Handler: f.DeleteFirewall,
			Tool: mcp.NewTool("digitalocean-firewall-delete",
				mcp.WithDescription("Delete a firewall"),
				mcp.WithString("ID", mcp.Required(), mcp.Description("ID of the firewall to delete")),
			),
		},
	}
}
