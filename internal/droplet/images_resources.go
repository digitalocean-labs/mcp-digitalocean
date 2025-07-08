package droplet

import (
	"context"
	"encoding/json"
	"fmt"
	"mcp-digitalocean/internal/common"

	"github.com/digitalocean/godo"
	"github.com/mark3labs/mcp-go/mcp"
	"github.com/mark3labs/mcp-go/server"
)

const ImagesURI = "images://"

type ImagesMCPResource struct {
	client *godo.Client
}

func NewImagesMCPResource(client *godo.Client) *ImagesMCPResource {
	return &ImagesMCPResource{
		client: client,
	}
}

// GetResource returns the resource for all distribution images
func (i *ImagesMCPResource) getImageResource() mcp.Resource {
	return mcp.NewResource(
		ImagesURI+"distribution",
		"Distribution Images",
		mcp.WithResourceDescription("Returns all available distribution images"),
		mcp.WithMIMEType("application/json"),
	)
}

// HandleGetResource handles the Images MCP resource requests for all images
func (i *ImagesMCPResource) handleGetImageResource(ctx context.Context, request mcp.ReadResourceRequest) ([]mcp.ResourceContents, error) {
	opt := &godo.ListOptions{
		Page:    1,
		PerPage: 200,
	}

	images, _, err := i.client.Images.ListDistribution(ctx, opt)
	if err != nil {
		return nil, fmt.Errorf("error fetching images: %w", err)
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
		return nil, fmt.Errorf("error serializing images: %w", err)
	}

	return []mcp.ResourceContents{
		mcp.TextResourceContents{
			MIMEType: "application/json",
			Text:     string(jsonData),
		},
	}, nil
}

// getImageResourceTemplate returns the template for a single image
func (i *ImagesMCPResource) getImageResourceTemplate() mcp.ResourceTemplate {
	return mcp.NewResourceTemplate(
		ImagesURI+"{id}",
		"Image",
		mcp.WithTemplateDescription("Returns image information"),
		mcp.WithTemplateMIMEType("application/json"),
	)
}

// handleGetImageResource handles the Images MCP resource requests for a single image
func (i *ImagesMCPResource) handleGetImageResourceTemplate(ctx context.Context, request mcp.ReadResourceRequest) ([]mcp.ResourceContents, error) {
	imageID, err := common.ExtractNumericIDFromURI(request.Params.URI)
	if err != nil {
		return nil, fmt.Errorf("invalid image URI: %w", err)
	}

	image, _, err := i.client.Images.GetByID(ctx, int(imageID))
	if err != nil {
		return nil, fmt.Errorf("error fetching image: %w", err)
	}

	jsonData, err := json.MarshalIndent(image, "", "  ")
	if err != nil {
		return nil, fmt.Errorf("error serializing image: %w", err)
	}

	return []mcp.ResourceContents{
		mcp.TextResourceContents{
			URI:      request.Params.URI,
			MIMEType: "application/json",
			Text:     string(jsonData),
		},
	}, nil
}

func (i *ImagesMCPResource) Resources() map[mcp.Resource]server.ResourceHandlerFunc {
	return map[mcp.Resource]server.ResourceHandlerFunc{
		i.getImageResource(): i.handleGetImageResource,
	}
}

func (i *ImagesMCPResource) ResourceTemplates() map[mcp.ResourceTemplate]server.ResourceTemplateHandlerFunc {
	return map[mcp.ResourceTemplate]server.ResourceTemplateHandlerFunc{
		i.getImageResourceTemplate(): i.handleGetImageResourceTemplate,
	}
}
