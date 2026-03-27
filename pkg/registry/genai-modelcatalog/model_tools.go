package genaimodelcatalog

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/digitalocean/godo"
	"github.com/mark3labs/mcp-go/mcp"
	"github.com/mark3labs/mcp-go/server"
)

// ModelTool provides model catalog management tools
type ModelTool struct {
	client func(ctx context.Context) (*godo.Client, error)
}

// NewModelTool creates a new ModelTool instance
func NewModelTool(client func(ctx context.Context) (*godo.Client, error)) *ModelTool {
	return &ModelTool{
		client: client,
	}
}

// searchModels searches for models in the catalog using a search string
func (m *ModelTool) searchModels(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	// Default to empty string if SearchQuery is not provided (returns all models)
	searchQuery, ok := req.GetArguments()["SearchQuery"].(string)
	if !ok {
		searchQuery = ""
	}

	client, err := m.client(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to get DigitalOcean client: %w", err)
	}

	uuids, _, err := client.GradientAI.SearchModels(ctx, searchQuery)
	if err != nil {
		return mcp.NewToolResultErrorFromErr("Failed to search models", err), nil
	}

	type ModelSearchResult struct {
		ModelUUIDs  []string `json:"model_uuids"`
		SearchQuery string   `json:"search_query"`
		Count       int      `json:"count"`
	}

	result := ModelSearchResult{
		ModelUUIDs:  uuids,
		SearchQuery: searchQuery,
		Count:       len(uuids),
	}

	jsonData, err := json.MarshalIndent(result, "", "  ")
	if err != nil {
		return nil, fmt.Errorf("marshal error: %w", err)
	}

	return mcp.NewToolResultText(string(jsonData)), nil
}

// getModelCard retrieves metadata for a specific model
func (m *ModelTool) getModelCard(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	modelUUID, ok := req.GetArguments()["ModelUUID"].(string)
	if !ok || modelUUID == "" {
		return mcp.NewToolResultError("ModelUUID is required"), nil
	}

	client, err := m.client(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to get DigitalOcean client: %w", err)
	}

	model, _, err := client.GradientAI.GetModelByUUID(ctx, modelUUID)
	if err != nil {
		return mcp.NewToolResultErrorFromErr("Failed to get model", err), nil
	}

	if model == nil {
		return mcp.NewToolResultError(fmt.Sprintf("Model with UUID '%s' not found", modelUUID)), nil
	}

	type ModelMetadata struct {
		UUID              string          `json:"uuid"`
		Name              string          `json:"name"`
		Agreement         *godo.Agreement `json:"agreement,omitempty"`
		ModelAvailability string          `json:"model_availability,omitempty"`
	}

	metadata := ModelMetadata{
		UUID:              model.Uuid,
		Name:              model.Name,
		Agreement:         model.Agreement,
		ModelAvailability: model.ModelAvailability,
	}

	jsonData, err := json.MarshalIndent(metadata, "", "  ")
	if err != nil {
		return nil, fmt.Errorf("marshal error: %w", err)
	}

	return mcp.NewToolResultText(string(jsonData)), nil
}

// Tools returns the list of server tools for model catalog management
func (m *ModelTool) Tools() []server.ServerTool {
	return []server.ServerTool{
		{
			Handler: m.searchModels,
			Tool: mcp.NewTool(
				"genai-model-catalog-search",
				mcp.WithDescription("Search for models in the catalog using a search query. Returns a list of model UUIDs that match the search criteria. An empty or missing search query returns all available models."),
				mcp.WithString("SearchQuery", mcp.Description("Search query string to find models (optional; empty or omitted returns all models)")),
			),
		},
		{
			Handler: m.getModelCard,
			Tool: mcp.NewTool(
				"genai-model-catalog-get-card",
				mcp.WithDescription("Get the model metadata for a specific model UUID."),
				mcp.WithString("ModelUUID", mcp.Required(), mcp.Description("The unique UUID identifier of the model")),
			),
		},
	}
}
