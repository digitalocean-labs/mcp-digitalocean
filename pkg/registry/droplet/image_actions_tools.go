package droplet

import (
	"context"
	"fmt"
	"mcp-digitalocean/pkg/response"

	"github.com/digitalocean/godo"
	"github.com/mark3labs/mcp-go/mcp"
	"github.com/mark3labs/mcp-go/server"
	"mcp-digitalocean/pkg/registry/common"
)

// ImageActionsTool provides tool-based handlers for DigitalOcean image actions.
type ImageActionsTool struct {
	client func(ctx context.Context) (*godo.Client, error)
}

// NewImageActionsTool creates a new ImageActionsTool instance.
func NewImageActionsTool(client func(ctx context.Context) (*godo.Client, error)) *ImageActionsTool {
	return &ImageActionsTool{client: client}
}

// transferImage triggers a transfer action for an image to a new region.
func (ia *ImageActionsTool) transferImage(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	imageID, ok := req.GetArguments()["ID"].(float64)
	if !ok {
		return mcp.NewToolResultError("ID is required"), nil
	}
	region, ok := req.GetArguments()["Region"].(string)
	if !ok || region == "" {
		return mcp.NewToolResultError("Region is required"), nil
	}

	client, err := ia.client(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to get DigitalOcean client: %w", err)
	}

	transferRequest := &godo.ActionRequest{
		"type":   "transfer",
		"region": region,
	}

	action, _, err := client.ImageActions.Transfer(ctx, int(imageID), transferRequest)
	if err != nil {
		return mcp.NewToolResultErrorFromErr("api error", err), nil
	}

	jsonAction, err := response.CompactJSON(action)
	if err != nil {
		return nil, fmt.Errorf("marshal error: %w", err)
	}

	return mcp.NewToolResultText(jsonAction), nil
}

// convertImageToSnapshot converts a backup into a snapshot.
func (ia *ImageActionsTool) convertImageToSnapshot(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	imageID, ok := req.GetArguments()["ID"].(float64)
	if !ok {
		return mcp.NewToolResultError("ID is required"), nil
	}

	client, err := ia.client(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to get DigitalOcean client: %w", err)
	}

	action, _, err := client.ImageActions.Convert(ctx, int(imageID))
	if err != nil {
		return mcp.NewToolResultErrorFromErr("api error", err), nil
	}

	jsonAction, err := response.CompactJSON(action)
	if err != nil {
		return nil, fmt.Errorf("marshal error: %w", err)
	}

	return mcp.NewToolResultText(jsonAction), nil
}

// getImageAction retrieves the status of an image action.
func (ia *ImageActionsTool) getImageAction(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	imageID, ok := req.GetArguments()["ImageID"].(float64)
	if !ok {
		return mcp.NewToolResultError("ImageID is required"), nil
	}
	actionID, ok := req.GetArguments()["ActionID"].(float64)
	if !ok {
		return mcp.NewToolResultError("ActionID is required"), nil
	}

	client, err := ia.client(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to get DigitalOcean client: %w", err)
	}

	action, _, err := client.ImageActions.Get(ctx, int(imageID), int(actionID))
	if err != nil {
		return mcp.NewToolResultErrorFromErr("api error", err), nil
	}

	jsonAction, err := response.CompactJSON(action)
	if err != nil {
		return nil, fmt.Errorf("marshal error: %w", err)
	}

	return mcp.NewToolResultText(jsonAction), nil
}

// Tools returns the list of server tools for image actions.
func (ia *ImageActionsTool) Tools() []server.ServerTool {
	return []server.ServerTool{
		{
			Handler: ia.transferImage,
			Tool: mcp.NewTool(
				"image-action-transfer",
				common.WithHints(common.HintsToggle),
				common.WithRisk(common.RiskLow),
				mcp.WithDescription("Transfer an image to another region."),
				mcp.WithNumber("ID", mcp.Required(), mcp.Description("ID of the image to transfer")),
				mcp.WithString("Region", mcp.Required(), mcp.Description("Region slug to transfer to (e.g., nyc3)")),
			),
		},
		{
			Handler: ia.convertImageToSnapshot,
			Tool: mcp.NewTool(
				"image-action-convert",
				common.WithHints(common.HintsToggle),
				common.WithRisk(common.RiskLow),
				mcp.WithDescription("Convert an image (backup) to a snapshot."),
				mcp.WithNumber("ID", mcp.Required(), mcp.Description("ID of the image to convert")),
			),
		},
		{
			Handler: ia.getImageAction,
			Tool: mcp.NewTool(
				"image-action-get",
				common.WithHints(common.HintsRead),
				common.WithRisk(common.RiskLow),
				mcp.WithDescription("Retrieve the status of an image action."),
				mcp.WithNumber("ImageID", mcp.Required(), mcp.Description("ID of the image")),
				mcp.WithNumber("ActionID", mcp.Required(), mcp.Description("ID of the action")),
			),
		},
	}
}
