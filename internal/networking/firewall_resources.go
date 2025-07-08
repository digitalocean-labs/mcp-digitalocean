package networking

import (
	"context"
	"encoding/json"
	"fmt"
	"mcp-digitalocean/internal/common"

	"github.com/digitalocean/godo"
	"github.com/mark3labs/mcp-go/mcp"
	"github.com/mark3labs/mcp-go/server"
)

const FirewallURI = "firewalls://"

type FirewallMCPResource struct {
	client *godo.Client
}

func NewFirewallMCPResource(client *godo.Client) *FirewallMCPResource {
	return &FirewallMCPResource{
		client: client,
	}
}

func (f *FirewallMCPResource) getFirewallResourceTemplate() mcp.ResourceTemplate {
	return mcp.NewResourceTemplate(
		FirewallURI+"{id}",
		"Firewall",
		mcp.WithTemplateDescription("Returns firewall information"),
		mcp.WithTemplateMIMEType("application/json"),
	)
}

func (f *FirewallMCPResource) handleGetFirewallResource(ctx context.Context, request mcp.ReadResourceRequest) ([]mcp.ResourceContents, error) {
	firewallID, err := common.ExtractStringIDFromURI(request.Params.URI)
	if err != nil {
		return nil, fmt.Errorf("invalid firewall URI: %w", err)
	}

	firewall, _, err := f.client.Firewalls.Get(ctx, firewallID)
	if err != nil {
		return nil, fmt.Errorf("error fetching firewall: %w", err)
	}

	jsonData, err := json.MarshalIndent(firewall, "", "  ")
	if err != nil {
		return nil, fmt.Errorf("error serializing firewall: %w", err)
	}

	return []mcp.ResourceContents{
		mcp.TextResourceContents{
			URI:      request.Params.URI,
			MIMEType: "application/json",
			Text:     string(jsonData),
		},
	}, nil
}

func (f *FirewallMCPResource) ResourceTemplates() map[mcp.ResourceTemplate]server.ResourceTemplateHandlerFunc {
	return map[mcp.ResourceTemplate]server.ResourceTemplateHandlerFunc{
		f.getFirewallResourceTemplate(): f.handleGetFirewallResource,
	}
}
