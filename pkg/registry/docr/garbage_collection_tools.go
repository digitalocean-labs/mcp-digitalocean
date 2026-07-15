package docr

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/digitalocean/godo"
	"github.com/mark3labs/mcp-go/mcp"
	"github.com/mark3labs/mcp-go/server"

	"mcp-digitalocean/pkg/registry/common"
)

const (
	defaultGCPageSize = 20
	defaultGCPage     = 1
)

// GarbageCollectionTool provides garbage collection management tools for container registries
type GarbageCollectionTool struct {
	client func(ctx context.Context) (*godo.Client, error)
}

// NewGarbageCollectionTool creates a new GarbageCollectionTool
func NewGarbageCollectionTool(client func(ctx context.Context) (*godo.Client, error)) *GarbageCollectionTool {
	return &GarbageCollectionTool{
		client: client,
	}
}

// startGarbageCollection starts a garbage collection for a container registry
func (g *GarbageCollectionTool) startGarbageCollection(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	registryName, ok := req.GetArguments()["RegistryName"].(string)
	if !ok || registryName == "" {
		return mcp.NewToolResultError("RegistryName is required"), nil
	}

	var gcReq *godo.StartGarbageCollectionRequest
	if gcType, ok := req.GetArguments()["Type"].(string); ok && gcType != "" {
		gcReq = &godo.StartGarbageCollectionRequest{
			Type: godo.GarbageCollectionType(gcType),
		}
	}

	client, err := g.client(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to get DigitalOcean client: %w", err)
	}

	gc, _, err := client.Registries.StartGarbageCollection(ctx, registryName, gcReq)
	if err != nil {
		return mcp.NewToolResultErrorFromErr("api error", err), nil
	}

	jsonGC, err := json.MarshalIndent(gc, "", "  ")
	if err != nil {
		return nil, fmt.Errorf("marshal error: %w", err)
	}

	return mcp.NewToolResultText(string(jsonGC)), nil
}

// getGarbageCollection gets the active garbage collection for a container registry
func (g *GarbageCollectionTool) getGarbageCollection(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	registryName, ok := req.GetArguments()["RegistryName"].(string)
	if !ok || registryName == "" {
		return mcp.NewToolResultError("RegistryName is required"), nil
	}

	client, err := g.client(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to get DigitalOcean client: %w", err)
	}

	gc, _, err := client.Registries.GetGarbageCollection(ctx, registryName)
	if err != nil {
		return mcp.NewToolResultErrorFromErr("api error", err), nil
	}

	jsonGC, err := json.MarshalIndent(gc, "", "  ")
	if err != nil {
		return nil, fmt.Errorf("marshal error: %w", err)
	}

	return mcp.NewToolResultText(string(jsonGC)), nil
}

// listGarbageCollections lists garbage collections for a container registry
func (g *GarbageCollectionTool) listGarbageCollections(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	registryName, ok := req.GetArguments()["RegistryName"].(string)
	if !ok || registryName == "" {
		return mcp.NewToolResultError("RegistryName is required"), nil
	}

	page := defaultGCPage
	perPage := defaultGCPageSize
	if v, ok := req.GetArguments()["Page"].(float64); ok && int(v) > 0 {
		page = int(v)
	}
	if v, ok := req.GetArguments()["PerPage"].(float64); ok && int(v) > 0 {
		perPage = int(v)
	}

	client, err := g.client(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to get DigitalOcean client: %w", err)
	}

	gcs, resp, err := client.Registries.ListGarbageCollections(ctx, registryName, &godo.ListOptions{
		Page:    page,
		PerPage: perPage,
	})
	if err != nil {
		return mcp.NewToolResultErrorFromErr("api error", err), nil
	}

	result := struct {
		GarbageCollections []*godo.GarbageCollection `json:"garbage_collections"`
		Meta               *godo.Meta                `json:"meta,omitempty"`
	}{
		GarbageCollections: gcs,
		Meta:               resp.Meta,
	}

	jsonGCs, err := json.MarshalIndent(result, "", "  ")
	if err != nil {
		return nil, fmt.Errorf("marshal error: %w", err)
	}

	return mcp.NewToolResultText(string(jsonGCs)), nil
}

// updateGarbageCollection updates a garbage collection (e.g., to cancel it)
func (g *GarbageCollectionTool) updateGarbageCollection(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	registryName, ok := req.GetArguments()["RegistryName"].(string)
	if !ok || registryName == "" {
		return mcp.NewToolResultError("RegistryName is required"), nil
	}

	gcUUID, ok := req.GetArguments()["GarbageCollectionUUID"].(string)
	if !ok || gcUUID == "" {
		return mcp.NewToolResultError("GarbageCollectionUUID is required"), nil
	}

	cancel, _ := req.GetArguments()["Cancel"].(bool)

	client, err := g.client(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to get DigitalOcean client: %w", err)
	}

	gc, _, err := client.Registries.UpdateGarbageCollection(ctx, registryName, gcUUID, &godo.UpdateGarbageCollectionRequest{
		Cancel: cancel,
	})
	if err != nil {
		return mcp.NewToolResultErrorFromErr("api error", err), nil
	}

	jsonGC, err := json.MarshalIndent(gc, "", "  ")
	if err != nil {
		return nil, fmt.Errorf("marshal error: %w", err)
	}

	return mcp.NewToolResultText(string(jsonGC)), nil
}

// Tools returns a list of tool functions for garbage collection management
func (g *GarbageCollectionTool) Tools() []server.ServerTool {
	return []server.ServerTool{
		{
			Handler: g.startGarbageCollection,
			Tool: mcp.NewTool("docr-garbage-collection-start",
				common.WithHints(common.HintsReplace),
				common.WithRisk(common.RiskHigh),
				mcp.WithDescription("Start a garbage collection for a container registry to free up storage"),
				mcp.WithString("RegistryName", mcp.Required(), mcp.Description("Name of the container registry")),
				mcp.WithString("Type", mcp.Description("Type of garbage collection to perform (e.g., 'untagged manifests and unreferenced blobs' or 'unreferenced blobs only')")),
			),
		},
		{
			Handler: g.getGarbageCollection,
			Tool: mcp.NewTool("docr-garbage-collection-get",
				common.WithHints(common.HintsRead),
				common.WithRisk(common.RiskLow),
				mcp.WithDescription("Get the active garbage collection for a container registry"),
				mcp.WithString("RegistryName", mcp.Required(), mcp.Description("Name of the container registry")),
			),
		},
		{
			Handler: g.listGarbageCollections,
			Tool: mcp.NewTool("docr-garbage-collection-list",
				common.WithHints(common.HintsRead),
				common.WithRisk(common.RiskLow),
				mcp.WithDescription("List garbage collections for a container registry"),
				mcp.WithString("RegistryName", mcp.Required(), mcp.Description("Name of the container registry")),
				mcp.WithNumber("Page", mcp.DefaultNumber(defaultGCPage), mcp.Description("Page number")),
				mcp.WithNumber("PerPage", mcp.DefaultNumber(defaultGCPageSize), mcp.Description("Items per page")),
			),
		},
		{
			Handler: g.updateGarbageCollection,
			Tool: mcp.NewTool("docr-garbage-collection-update",
				common.WithHints(common.HintsToggle),
				common.WithRisk(common.RiskLow),
				mcp.WithDescription("Update a garbage collection for a container registry (e.g., to cancel it)"),
				mcp.WithString("RegistryName", mcp.Required(), mcp.Description("Name of the container registry")),
				mcp.WithString("GarbageCollectionUUID", mcp.Required(), mcp.Description("UUID of the garbage collection to update")),
				mcp.WithBoolean("Cancel", mcp.Required(), mcp.Description("Set to true to cancel the garbage collection")),
			),
		},
	}
}
