package droplet

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/digitalocean/godo"
	"github.com/mark3labs/mcp-go/mcp"
	"github.com/mark3labs/mcp-go/server"
)

const (
	defaultImagesPageSize = 50
	defaultImagesPage     = 1
)

// ImagesTool provides tool-based handlers for DigitalOcean images.
type ImagesTool struct {
	client *godo.Client
}

// NewImagesTool creates a new ImagesTool instance.
func NewImagesTool(client *godo.Client) *ImagesTool {
	return &ImagesTool{client: client}
}

// listImages lists all distribution images with pagination support.
func (i *ImagesTool) listImages(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	page, ok := req.GetArguments()["Page"].(float64)
	if !ok {
		page = defaultImagesPage
	}
	perPage, ok := req.GetArguments()["PerPage"].(float64)
	if !ok {
		perPage = defaultImagesPageSize
	}

	opt := &godo.ListOptions{
		Page:    int(page),
		PerPage: int(perPage),
	}

	images, _, err := i.client.Images.ListDistribution(ctx, opt)
	if err != nil {
		return mcp.NewToolResultErrorFromErr("api error", err), nil
	}

	filteredImages := make([]map[string]any, len(images))
	for idx, image := range images {
		filteredImages[idx] = map[string]any{
			"id":           image.ID,
			"name":         image.Name,
			"distribution": image.Distribution,
			"type":         image.Type,
		}
	}

	jsonData, err := json.MarshalIndent(filteredImages, "", "  ")
	if err != nil {
		return nil, fmt.Errorf("marshal error: %w", err)
	}

	return mcp.NewToolResultText(string(jsonData)), nil
}

// getImageByID retrieves a specific image by its numeric ID.
func (i *ImagesTool) getImageByID(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	id, ok := req.GetArguments()["ID"].(float64)
	if !ok {
		return mcp.NewToolResultError("Image ID is required"), nil
	}

	image, _, err := i.client.Images.GetByID(ctx, int(id))
	if err != nil {
		return mcp.NewToolResultErrorFromErr("api error", err), nil
	}

	jsonData, err := json.MarshalIndent(image, "", "  ")
	if err != nil {
		return nil, fmt.Errorf("marshal error: %w", err)
	}

	return mcp.NewToolResultText(string(jsonData)), nil
}

// Tools returns the list of server tools for images.
func (i *ImagesTool) Tools() []server.ServerTool {
	return []server.ServerTool{
		{
			Handler: i.listImages,
			Tool: mcp.NewTool(
				"digitalocean-image-list",
				mcp.WithDescription("List all available distribution images. Supports pagination."),
				mcp.WithNumber("Page", mcp.DefaultNumber(defaultImagesPage), mcp.Description("Page number")),
				mcp.WithNumber("PerPage", mcp.DefaultNumber(defaultImagesPageSize), mcp.Description("Items per page")),
			),
		},
		{
			Handler: i.getImageByID,
			Tool: mcp.NewTool(
				"digitalocean-image-get",
				mcp.WithDescription("Get a specific image by its numeric ID."),
				mcp.WithNumber("ID", mcp.Required(), mcp.Description("Image ID")),
			),
		},
	}
}
