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

const KeyURI = "keys://"

type KeyMCPResource struct {
	client *godo.Client
}

func NewKeyMCPResource(client *godo.Client) *KeyMCPResource {
	return &KeyMCPResource{
		client: client,
	}
}

func (k *KeyMCPResource) getKeyResourceTemplate() mcp.ResourceTemplate {
	return mcp.NewResourceTemplate(
		KeyURI+"{id}",
		"SSH Key",
		mcp.WithTemplateDescription("Returns SSH key information"),
		mcp.WithTemplateMIMEType("application/json"),
	)
}

func (k *KeyMCPResource) handleGetKeyResource(ctx context.Context, request mcp.ReadResourceRequest) ([]mcp.ResourceContents, error) {
	keyID, err := common.ExtractNumericIDFromURI(request.Params.URI)
	if err != nil {
		return nil, fmt.Errorf("invalid key URI: %w", err)
	}

	key, _, err := k.client.Keys.GetByID(ctx, int(keyID))
	if err != nil {
		return nil, fmt.Errorf("error fetching key: %w", err)
	}

	jsonData, err := json.MarshalIndent(key, "", "  ")
	if err != nil {
		return nil, fmt.Errorf("error serializing key: %w", err)
	}

	return []mcp.ResourceContents{
		mcp.TextResourceContents{
			URI:      request.Params.URI,
			MIMEType: "application/json",
			Text:     string(jsonData),
		},
	}, nil
}

func (k *KeyMCPResource) ResourceTemplates() map[mcp.ResourceTemplate]server.ResourceTemplateHandlerFunc {
	return map[mcp.ResourceTemplate]server.ResourceTemplateHandlerFunc{
		k.getKeyResourceTemplate(): k.handleGetKeyResource,
	}
}
