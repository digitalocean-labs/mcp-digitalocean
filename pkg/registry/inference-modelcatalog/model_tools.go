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
	UUID              string                        `json:"uuid"`
	Name              string                        `json:"name"`
	Description       string                        `json:"description,omitempty"`
	Provider          string                        `json:"provider,omitempty"`
	Agreement         *godo.Agreement               `json:"agreement,omitempty"`
	ModelAvailability string                        `json:"model_availability,omitempty"`
	ContextWindow     string                        `json:"context_window,omitempty"`
	Capabilities      []string                      `json:"capabilities,omitempty"`
	Modalities        *godo.ModelModalities         `json:"modalities,omitempty"`
	ParameterCount    float64                       `json:"parameter_count,omitempty"`
	Type              string                        `json:"type,omitempty"`
	Pricing           *godo.ModelPricing            `json:"pricing,omitempty"`
	Performance       *godo.ModelCatalogPerformance `json:"performance,omitempty"`
	BenchmarkScore    json.RawMessage               `json:"benchmark_score,omitempty"`
}

// NewModelTool creates a new ModelTool instance
func NewModelTool(client func(ctx context.Context) (*godo.Client, error)) *ModelTool {
	return &ModelTool{
		client: client,
	}
}

// getModelMetadata fetches a V2 catalog card and returns it as ModelMetadata.
func (m *ModelTool) getModelMetadata(ctx context.Context, modelUUID string) (*ModelMetadata, error) {
	client, err := m.client(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to get DigitalOcean client: %w", err)
	}

	model, _, err := client.AgentPlatform.GetModelCatalogCardV2(ctx, modelUUID, "")
	if err != nil {
		return nil, fmt.Errorf("failed to get model: %w", err)
	}

	if model == nil {
		return nil, fmt.Errorf("model with UUID '%s' not found", modelUUID)
	}

	return metadataFromCatalogEntry(model), nil
}

func metadataFromCatalogEntry(model *godo.ModelCatalogEntryV2) *ModelMetadata {
	availability := ""
	if len(model.Serving) > 0 {
		availability = model.Serving[0]
	}
	var modalities *godo.ModelModalities
	if len(model.InputModality) > 0 || len(model.OutputModality) > 0 {
		modalities = &godo.ModelModalities{
			Input:  model.InputModality,
			Output: model.OutputModality,
		}
	}
	return &ModelMetadata{
		UUID:              model.ID,
		Name:              model.Name,
		Description:       model.Description,
		Provider:          model.Provider,
		ModelAvailability: availability,
		ContextWindow:     model.ContextWindow,
		Capabilities:      model.Capabilities,
		Modalities:        modalities,
		ParameterCount:    model.ParameterCountInBillions,
		Type:              model.Type,
		Pricing:           catalogPricing(model.Pricing),
		Performance:       model.Performance,
	}
}

func catalogPricing(pricing *godo.ModelCatalogPricing) *godo.ModelPricing {
	if pricing == nil {
		return nil
	}
	shared := pricing.ModelPricing
	return &shared
}

// searchModels searches for models in the catalog using a search string
func (m *ModelTool) searchModels(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	// Default to empty string if SearchQuery is not provided (returns all models)
	searchQuery, ok := req.GetArguments()["SearchQuery"].(string)
	if !ok {
		searchQuery = ""
	}

	entries, err := m.listCatalog(ctx, searchQuery)
	if err != nil {
		return mcp.NewToolResultErrorFromErr("Failed to search models", err), nil
	}

	uuids := make([]string, 0, len(entries))
	for _, entry := range entries {
		if entry == nil || entry.ID == "" {
			continue
		}
		uuids = append(uuids, entry.ID)
	}

	return searchOut.Result(modelSearchResult{
		ModelUUIDs:  uuids,
		SearchQuery: searchQuery,
		Count:       len(uuids),
	})
}

// listCatalog pages through List Model Catalog V2 until the API reports no next page.
func (m *ModelTool) listCatalog(ctx context.Context, searchQuery string) ([]*godo.ModelCatalogEntryV2, error) {
	client, err := m.client(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to get DigitalOcean client: %w", err)
	}

	var entries []*godo.ModelCatalogEntryV2
	for page := 1; page <= 1000; page++ {
		list, resp, err := client.AgentPlatform.ListModelCatalogV2(ctx, &godo.ModelCatalogV2ListOptions{
			Page:    page,
			PerPage: 200,
			Search:  searchQuery,
		})
		if err != nil {
			return nil, err
		}
		if list != nil {
			entries = append(entries, list.Data...)
		}
		if resp == nil || resp.Links == nil || resp.Links.Pages == nil || resp.Links.Pages.Next == "" {
			break
		}
	}
	return entries, nil
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

	return modelCardOut.Result(metadata)
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
				searchOut.Schema(),
				mcp.WithDescription("Search the V2 model catalog. Returns catalog IDs for models that match the search query. An empty or missing search query returns every catalog model visible to the authenticated account."),
				mcp.WithString("SearchQuery", mcp.Description("Search query string to find models (optional; empty or omitted returns all models)")),
			),
		},
		{
			Handler: m.getModelCard,
			Tool: mcp.NewTool(
				"inference-model-catalog-get-card",
				common.WithHints(common.HintsRead),
				common.WithRisk(common.RiskLow),
				modelCardOut.Schema(),
				mcp.WithDescription("Get the V2 model catalog card for a catalog ID."),
				mcp.WithString("ModelUUID", mcp.Required(), mcp.Description("Catalog ID of the model, as returned by inference-model-catalog-search")),
			),
		},
	}
}
