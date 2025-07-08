package account

import (
	"context"
	"encoding/json"
	"fmt"
	"mcp-digitalocean/internal/common"

	"github.com/digitalocean/godo"
	"github.com/mark3labs/mcp-go/mcp"
	"github.com/mark3labs/mcp-go/server"
)

const ActionURI = "actions://"

type ActionMCPResource struct {
	client *godo.Client
}

func NewActionMCPResource(client *godo.Client) *ActionMCPResource {
	return &ActionMCPResource{
		client: client,
	}
}

func (a *ActionMCPResource) getActionResourceTemplate() mcp.ResourceTemplate {
	return mcp.NewResourceTemplate(
		ActionURI+"{id}",
		"Action",
		mcp.WithTemplateDescription("Returns action information"),
		mcp.WithTemplateMIMEType("application/json"),
	)
}

func (a *ActionMCPResource) handleGetActionResourceTemplate(ctx context.Context, request mcp.ReadResourceRequest) ([]mcp.ResourceContents, error) {
	actionID, err := common.ExtractNumericIDFromURI(request.Params.URI)
	if err != nil {
		return nil, fmt.Errorf("invalid action URI: %w", err)
	}

	action, _, err := a.client.Actions.Get(ctx, int(actionID))
	if err != nil {
		return nil, fmt.Errorf("error fetching action: %w", err)
	}

	jsonData, err := json.MarshalIndent(action, "", "  ")
	if err != nil {
		return nil, fmt.Errorf("error serializing action: %w", err)
	}

	return []mcp.ResourceContents{
		mcp.TextResourceContents{
			MIMEType: "application/json",
			Text:     string(jsonData),
		},
	}, nil
}

func (a *ActionMCPResource) ResourcesTemplates() map[mcp.ResourceTemplate]server.ResourceTemplateHandlerFunc {
	return map[mcp.ResourceTemplate]server.ResourceTemplateHandlerFunc{
		a.getActionResourceTemplate(): a.handleGetActionResourceTemplate,
	}
}
