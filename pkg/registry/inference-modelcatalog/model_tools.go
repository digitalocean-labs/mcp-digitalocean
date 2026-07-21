package inferencemodelcatalog

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/digitalocean/godo"
	"github.com/mark3labs/mcp-go/mcp"
	"github.com/mark3labs/mcp-go/server"

	"mcp-digitalocean/pkg/registry/common"
)

// ModelTool provides model catalog management tools
type ModelTool struct {
	client func(ctx context.Context) (*godo.Client, error)
}

// ModelMetadata represents model card metadata
type ModelMetadata struct {
	UUID              string                `json:"uuid"`
	Name              string                `json:"name"`
	Description       string                `json:"description,omitempty"`
	Provider          string                `json:"provider,omitempty"`
	Agreement         *godo.Agreement       `json:"agreement,omitempty"`
	ModelAvailability string                `json:"model_availability,omitempty"`
	ContextWindow     string                `json:"context_window,omitempty"`
	Capabilities      []string              `json:"capabilities,omitempty"`
	Modalities        *godo.ModelModalities `json:"modalities,omitempty"`
	ParameterCount    float64               `json:"parameter_count,omitempty"`
	Type              string                `json:"type,omitempty"`
	Pricing           *godo.ModelPricing    `json:"pricing,omitempty"`
	BenchmarkScore    json.RawMessage       `json:"benchmark_score,omitempty"`
}

// NewModelTool creates a new ModelTool instance
func NewModelTool(client func(ctx context.Context) (*godo.Client, error)) *ModelTool {
	return &ModelTool{
		client: client,
	}
}

// getModelMetadata fetches model card data and returns it as ModelMetadata
func (m *ModelTool) getModelMetadata(ctx context.Context, modelUUID string) (*ModelMetadata, error) {
	client, err := m.client(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to get DigitalOcean client: %w", err)
	}

	model, _, err := client.GradientAI.GetModelByUUID(ctx, modelUUID)
	if err != nil {
		return nil, fmt.Errorf("failed to get model: %w", err)
	}

	if model == nil {
		return nil, fmt.Errorf("model with UUID '%s' not found", modelUUID)
	}

	return &ModelMetadata{
		UUID:              model.Uuid,
		Name:              model.Name,
		Description:       model.Description,
		Provider:          model.Provider,
		Agreement:         model.Agreement,
		ModelAvailability: model.ModelAvailability,
		ContextWindow:     model.ContextWindow,
		Capabilities:      model.Capabilities,
		Modalities:        model.Modalities,
		ParameterCount:    model.ParameterCount,
		Type:              model.Type,
		Pricing:           model.Pricing,
		BenchmarkScore:    model.BenchmarkScore,
	}, nil
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

	metadata, err := m.getModelMetadata(ctx, modelUUID)
	if err != nil {
		return mcp.NewToolResultErrorFromErr("Failed to get model", err), nil
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
				"inference-model-catalog-search",
				common.WithHints(common.HintsRead),
				common.WithRisk(common.RiskLow),
				mcp.WithDescription("Search for models in the catalog using a search query. Returns a list of model UUIDs that match the search criteria. An empty or missing search query returns all available models."),
				mcp.WithString("SearchQuery", mcp.Description("Search query string to find models (optional; empty or omitted returns all models)")),
			),
		},
		{
			Handler: m.getModelCard,
			Tool: mcp.NewTool(
				"inference-model-catalog-get-card",
				common.WithHints(common.HintsRead),
				common.WithRisk(common.RiskLow),
				mcp.WithDescription("Get the model metadata for a specific model UUID."),
				mcp.WithString("ModelUUID", mcp.Required(), mcp.Description("The unique UUID identifier of the model")),
			),
		},
	}
}
