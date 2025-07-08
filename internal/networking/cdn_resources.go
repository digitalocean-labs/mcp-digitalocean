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

const CDNURI = "cdn://"

type CDNMCPResource struct {
	client *godo.Client
}

func NewCDNMCPResource(client *godo.Client) *CDNMCPResource {
	return &CDNMCPResource{
		client: client,
	}
}

func (c *CDNMCPResource) getCDNResourceTemplate() mcp.ResourceTemplate {
	return mcp.NewResourceTemplate(
		CDNURI+"{id}",
		"CDN",
		mcp.WithTemplateDescription("Returns CDN information"),
		mcp.WithTemplateMIMEType("application/json"),
	)
}

func (c *CDNMCPResource) handleGetCDNResource(ctx context.Context, request mcp.ReadResourceRequest) ([]mcp.ResourceContents, error) {
	cdnID, err := common.ExtractStringIDFromURI(request.Params.URI)
	if err != nil {
		return nil, fmt.Errorf("invalid CDN URI: %w", err)
	}

	cdn, _, err := c.client.CDNs.Get(ctx, cdnID)
	if err != nil {
		return nil, fmt.Errorf("error fetching CDN: %w", err)
	}

	jsonData, err := json.MarshalIndent(cdn, "", "  ")
	if err != nil {
		return nil, fmt.Errorf("error serializing CDN: %w", err)
	}

	return []mcp.ResourceContents{
		mcp.TextResourceContents{
			URI:      request.Params.URI,
			MIMEType: "application/json",
			Text:     string(jsonData),
		},
	}, nil
}

func (c *CDNMCPResource) ResourceTemplates() map[mcp.ResourceTemplate]server.ResourceTemplateHandlerFunc {
	return map[mcp.ResourceTemplate]server.ResourceTemplateHandlerFunc{
		c.getCDNResourceTemplate(): c.handleGetCDNResource,
	}
}
