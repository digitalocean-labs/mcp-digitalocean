package volumes

import (
	"context"
	"encoding/json"

	"github.com/digitalocean/godo"
	"github.com/mark3labs/mcp-go/mcp"
	"github.com/mark3labs/mcp-go/server"

	"mcp-digitalocean/pkg/registry/common"
)

type VolumeActionsTool struct {
	client func(ctx context.Context) (*godo.Client, error)
}

// NewVolumeActionsTool creates a new VolumeActionsTool instance
func NewVolumeActionsTool(client func(ctx context.Context) (*godo.Client, error)) *VolumeActionsTool {
	return &VolumeActionsTool{client: client}
}

func (v *VolumeActionsTool) attachVolume(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	args := req.GetArguments()
	volumeID, ok := args["VolumeID"].(string)
	if !ok || volumeID == "" {
		return mcp.NewToolResultError("Volume ID is required"), nil
	}
	dropletID, ok := args["DropletID"].(float64)
	if !ok || dropletID < 1 {
		return mcp.NewToolResultError("Droplet ID is required"), nil
	}

	client, err := v.client(ctx)
	if err != nil {
		return mcp.NewToolResultErrorFromErr("Error getting DigitalOcean client", err), nil
	}

	action, _, err := client.StorageActions.Attach(ctx, volumeID, int(dropletID))
	if err != nil {
		return mcp.NewToolResultErrorFromErr("api error", err), nil
	}

	jsonAction, err := json.MarshalIndent(action, "", "  ")
	if err != nil {
		return mcp.NewToolResultErrorFromErr("marshal error", err), nil
	}
	return mcp.NewToolResultText(string(jsonAction)), nil
}

func (v *VolumeActionsTool) detachVolume(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	args := req.GetArguments()
	volumeID, ok := args["VolumeID"].(string)
	if !ok || volumeID == "" {
		return mcp.NewToolResultError("Volume ID is required"), nil
	}
	dropletID, ok := args["DropletID"].(float64)
	if !ok || dropletID < 1 {
		return mcp.NewToolResultError("Droplet ID is required"), nil
	}

	client, err := v.client(ctx)
	if err != nil {
		return mcp.NewToolResultErrorFromErr("Error getting DigitalOcean client", err), nil
	}

	action, _, err := client.StorageActions.DetachByDropletID(ctx, volumeID, int(dropletID))
	if err != nil {
		return mcp.NewToolResultErrorFromErr("api error", err), nil
	}

	jsonAction, err := json.MarshalIndent(action, "", "  ")
	if err != nil {
		return mcp.NewToolResultErrorFromErr("marshal error", err), nil
	}
	return mcp.NewToolResultText(string(jsonAction)), nil
}

func (v *VolumeActionsTool) getVolumeAction(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	args := req.GetArguments()
	volumeID, ok := args["VolumeID"].(string)
	if !ok || volumeID == "" {
		return mcp.NewToolResultError("Volume ID is required"), nil
	}
	actionID, ok := args["ActionID"].(float64)
	if !ok || actionID < 1 {
		return mcp.NewToolResultError("Action ID is required"), nil
	}

	client, err := v.client(ctx)
	if err != nil {
		return mcp.NewToolResultErrorFromErr("Error getting DigitalOcean client", err), nil
	}

	action, _, err := client.StorageActions.Get(ctx, volumeID, int(actionID))
	if err != nil {
		return mcp.NewToolResultErrorFromErr("api error", err), nil
	}
	jsonAction, err := json.MarshalIndent(action, "", "  ")
	if err != nil {
		return mcp.NewToolResultErrorFromErr("marshal error", err), nil
	}
	return mcp.NewToolResultText(string(jsonAction)), nil
}

func (v *VolumeActionsTool) listVolumeActions(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	args := req.GetArguments()
	volumeID, ok := args["VolumeID"].(string)
	if !ok || volumeID == "" {
		return mcp.NewToolResultError("Volume ID is required"), nil
	}

	page, ok := args["Page"].(float64)
	if !ok || page < 1 {
		page = defaultVolumeListPage
	}
	perPage, ok := args["PerPage"].(float64)
	if !ok || perPage < 1 {
		perPage = defaultVolumeListPerPage
	}
	if perPage > maxVolumeListPerPage {
		perPage = maxVolumeListPerPage
	}

	client, err := v.client(ctx)
	if err != nil {
		return mcp.NewToolResultErrorFromErr("Error getting DigitalOcean client", err), nil
	}

	options := &godo.ListOptions{
		Page:    int(page),
		PerPage: int(perPage),
	}

	actions, _, err := client.StorageActions.List(ctx, volumeID, options)
	if err != nil {
		return mcp.NewToolResultErrorFromErr("api error", err), nil
	}
	jsonActions, err := json.MarshalIndent(actions, "", "  ")
	if err != nil {
		return mcp.NewToolResultErrorFromErr("marshal error", err), nil
	}
	return mcp.NewToolResultText(string(jsonActions)), nil
}

func (v *VolumeActionsTool) resizeVolume(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	args := req.GetArguments()
	volumeID, ok := args["VolumeID"].(string)
	if !ok || volumeID == "" {
		return mcp.NewToolResultError("Volume ID is required"), nil
	}
	sizeGigaBytes, ok := args["SizeGigaBytes"].(float64)
	if !ok || sizeGigaBytes < 1 {
		return mcp.NewToolResultError("SizeGigaBytes is required"), nil
	}
	region, ok := args["Region"].(string)
	if !ok || region == "" {
		return mcp.NewToolResultError("Region is required"), nil
	}
	client, err := v.client(ctx)
	if err != nil {
		return mcp.NewToolResultErrorFromErr("Error getting DigitalOcean client", err), nil
	}

	action, _, err := client.StorageActions.Resize(ctx, volumeID, int(sizeGigaBytes), region)
	if err != nil {
		return mcp.NewToolResultErrorFromErr("api error", err), nil
	}

	jsonAction, err := json.MarshalIndent(action, "", "  ")
	if err != nil {
		return mcp.NewToolResultErrorFromErr("marshal error", err), nil
	}
	return mcp.NewToolResultText(string(jsonAction)), nil
}

// Tools returns MCP server tools for volume lifecycle actions (attach, detach, resize, and action list/get).
func (v *VolumeActionsTool) Tools() []server.ServerTool {
	return []server.ServerTool{
		{
			Handler: v.attachVolume,
			Tool: mcp.NewTool("volume-attach",
				common.WithHints(common.HintsToggle),
				common.WithRisk(common.RiskMedium),
				mcp.WithDescription("Attach a volume to a droplet"),
				mcp.WithString("VolumeID", mcp.Required(), mcp.Description("The ID of the volume to attach")),
				mcp.WithNumber("DropletID", mcp.Required(), mcp.Description("The ID of the droplet to attach the volume to")),
			),
		},
		{
			Handler: v.detachVolume,
			Tool: mcp.NewTool("volume-detach",
				common.WithHints(common.HintsToggle),
				common.WithRisk(common.RiskMedium),
				mcp.WithDescription("Detach a volume from a droplet"),
				mcp.WithString("VolumeID", mcp.Required(), mcp.Description("The ID of the volume to detach")),
				mcp.WithNumber("DropletID", mcp.Required(), mcp.Description("The ID of the droplet to detach the volume from")),
			),
		},
		{
			Handler: v.getVolumeAction,
			Tool: mcp.NewTool("volume-action-get",
				common.WithHints(common.HintsRead),
				common.WithRisk(common.RiskLow),
				mcp.WithDescription("Get a volume action by ID"),
				mcp.WithString("VolumeID", mcp.Required(), mcp.Description("The ID of the volume")),
				mcp.WithNumber("ActionID", mcp.Required(), mcp.Description("The ID of the action")),
			),
		},
		{
			Handler: v.listVolumeActions,
			Tool: mcp.NewTool("volume-action-list",
				common.WithHints(common.HintsRead),
				common.WithRisk(common.RiskLow),
				mcp.WithDescription("List volume actions"),
				mcp.WithString("VolumeID", mcp.Required(), mcp.Description("The ID of the volume")),
				mcp.WithNumber("Page", mcp.DefaultNumber(defaultVolumeListPage), mcp.Description("Page number")),
				mcp.WithNumber("PerPage", mcp.DefaultNumber(defaultVolumeListPerPage), mcp.Description("Actions per page")),
			),
		},
		{
			Handler: v.resizeVolume,
			Tool: mcp.NewTool("volume-resize",
				common.WithHints(common.HintsToggle),
				common.WithRisk(common.RiskMedium),
				mcp.WithDescription("Resize a volume"),
				mcp.WithString("VolumeID", mcp.Required(), mcp.Description("The ID of the volume to resize")),
				mcp.WithNumber("SizeGigaBytes", mcp.Required(), mcp.Description("The size of the volume in GB")),
				mcp.WithString("Region", mcp.Required(), mcp.Description("The region slug where the volume will be resized")),
			),
		},
	}
}
