package resources

import (
	"context"
	"encoding/json"
	"fmt"
	"strconv"
	"strings"

	"github.com/digitalocean/godo"
	"github.com/mark3labs/mcp-go/mcp"
)

// BillingMCPResource represents a handler for MCP Billing resources
type BillingMCPResource struct {
	client *godo.Client
}

// NewBillingMCPResource creates a new Billing MCP resource handler
func NewBillingMCPResource(client *godo.Client) *BillingMCPResource {
	return &BillingMCPResource{
		client: client,
	}
}

// GetResourceTemplate returns the template for the Billing MCP resource
func (b *BillingMCPResource) GetResourceTemplate() mcp.ResourceTemplate {
	return mcp.NewResourceTemplate(
		"billing://{last}",
		"Billing History",
		mcp.WithTemplateDescription("Returns billing history"),
		mcp.WithTemplateMIMEType("application/json"),
	)
}

// HandleGetResource handles the Billing MCP resource requests
func (b *BillingMCPResource) HandleGetResource(ctx context.Context, request mcp.ReadResourceRequest) ([]mcp.ResourceContents, error) {
	// Extract the last parameter from the URI
	lastParam := strings.TrimPrefix(request.Params.URI, "billing://")
	if lastParam == "" {
		return nil, fmt.Errorf("invalid billing URI: %s", request.Params.URI)
	}

	// Check if the last parameter is a valid number
	perpage, err := strconv.Atoi(lastParam)
	if err != nil {
		return nil, fmt.Errorf("invalid billing URI: %s", request.Params.URI)
	}

	// Get billing history from DigitalOcean API
	billingHistory, _, err := b.client.BillingHistory.List(ctx, &godo.ListOptions{
		Page:    1,
		PerPage: perpage,
	})
	if err != nil {
		return nil, fmt.Errorf("error fetching billing history: %s", err)
	}

	// Serialize to JSON
	jsonData, err := json.MarshalIndent(billingHistory, "", "  ")
	if err != nil {
		return nil, fmt.Errorf("error serializing billing history: %s", err)
	}

	return []mcp.ResourceContents{
		mcp.TextResourceContents{
			MIMEType: "application/json",
			Text:     string(jsonData),
		},
	}, nil
}

// ResourceTemplates returns the resource templates for the Billing MCP resource
func (b *BillingMCPResource) ResourceTemplates() map[mcp.ResourceTemplate]MCPResourceHandler {
	return map[mcp.ResourceTemplate]MCPResourceHandler{
		b.GetResourceTemplate(): b.HandleGetResource,
	}
}
