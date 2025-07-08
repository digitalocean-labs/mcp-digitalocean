package droplet

import (
	"context"
	"encoding/json"
	"fmt"
	"mcp-digitalocean/internal/common"
	"strconv"
	"strings"

	"github.com/digitalocean/godo"
	"github.com/mark3labs/mcp-go/mcp"
	"github.com/mark3labs/mcp-go/server"
)

const DropletURI = "droplets://"

type DropletMCPResource struct {
	client *godo.Client
}

func NewDropletMCPResource(client *godo.Client) *DropletMCPResource {
	return &DropletMCPResource{
		client: client,
	}
}

func (d *DropletMCPResource) getDropletResourceTemplate() mcp.ResourceTemplate {
	return mcp.NewResourceTemplate(
		DropletURI+"{id}",
		"Droplet",
		mcp.WithTemplateDescription("Returns droplet information"),
		mcp.WithTemplateMIMEType("application/json"),
	)
}

func (d *DropletMCPResource) handleGetDropletResource(ctx context.Context, request mcp.ReadResourceRequest) ([]mcp.ResourceContents, error) {
	dropletID, err := common.ExtractNumericIDFromURI(request.Params.URI)
	if err != nil {
		return nil, fmt.Errorf("invalid droplet URI: %w", err)
	}

	droplet, _, err := d.client.Droplets.Get(ctx, int(dropletID))
	if err != nil {
		return nil, fmt.Errorf("error fetching droplet: %w", err)
	}

	jsonData, err := json.MarshalIndent(droplet, "", "  ")
	if err != nil {
		return nil, fmt.Errorf("error serializing droplet: %w", err)
	}

	return []mcp.ResourceContents{
		mcp.TextResourceContents{
			URI:      request.Params.URI,
			MIMEType: "application/json",
			Text:     string(jsonData),
		},
	}, nil
}

func (d *DropletMCPResource) getActionsResourceTemplate() mcp.ResourceTemplate {
	return mcp.NewResourceTemplate(
		fmt.Sprintf("%s{id}/actions/{action_id}", DropletURI),
		"Droplet Action",
		mcp.WithTemplateDescription("Returns information about a droplet action"),
		mcp.WithTemplateMIMEType("application/json"),
	)
}

func (d *DropletMCPResource) handleGetActionsResource(ctx context.Context, request mcp.ReadResourceRequest) ([]mcp.ResourceContents, error) {
	uri := request.Params.URI
	dropletID, actionID, err := extractDropletAndActionFromURI(uri)
	if err != nil {
		return nil, err
	}

	action, _, err := d.client.DropletActions.Get(ctx, dropletID, actionID)
	if err != nil {
		return nil, fmt.Errorf("error fetching action: %w", err)
	}

	jsonData, err := json.MarshalIndent(action, "", "  ")
	if err != nil {
		return nil, fmt.Errorf("error serializing action: %w", err)
	}

	return []mcp.ResourceContents{
		mcp.TextResourceContents{
			URI:      request.Params.URI,
			MIMEType: "application/json",
			Text:     string(jsonData),
		},
	}, nil
}

func extractDropletAndActionFromURI(uri string) (int, int, error) {
	uri = strings.TrimPrefix(uri, DropletURI) //  Now: {}/actions/{}
	parts := strings.Split(uri, "/")
	if len(parts) != 3 {
		return 0, 0, fmt.Errorf("invalid URI format")
	}

	dropletID, err := strconv.Atoi(parts[0])
	if err != nil {
		return 0, 0, fmt.Errorf("invalid droplet ID")
	}

	actionID, err := strconv.Atoi(parts[2])
	if err != nil {
		return 0, 0, fmt.Errorf("invalid action ID")
	}

	return dropletID, actionID, nil
}

func (d *DropletMCPResource) ResourceTemplates() map[mcp.ResourceTemplate]server.ResourceTemplateHandlerFunc {
	return map[mcp.ResourceTemplate]server.ResourceTemplateHandlerFunc{
		d.getDropletResourceTemplate(): d.handleGetDropletResource,
		d.getActionsResourceTemplate(): d.handleGetActionsResource,
	}
}
