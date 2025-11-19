// Package droplet provides tools for managing droplet actions
package droplet

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/digitalocean/godo"
	"github.com/mark3labs/mcp-go/mcp"
	"github.com/mark3labs/mcp-go/server"
)

// DropletActionsTool provides tools for droplet actions
type DropletActionsTool struct {
	client func(ctx context.Context) (*godo.Client, error)
}

// NewDropletActionsTool creates a new droplet actions tool
func NewDropletActionsTool(client func(ctx context.Context) (*godo.Client, error)) *DropletActionsTool {
	return &DropletActionsTool{
		client: client,
	}
}

// rebootDroplet reboots a droplet
func (da *DropletActionsTool) rebootDroplet(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	dropletID := req.GetArguments()["ID"].(float64)

	client, err := da.client(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to get DigitalOcean client: %w", err)
	}

	action, _, err := client.DropletActions.Reboot(ctx, int(dropletID))
	if err != nil {
		return mcp.NewToolResultErrorFromErr("api error", err), nil
	}

	jsonAction, err := json.MarshalIndent(action, "", "  ")
	if err != nil {
		return nil, fmt.Errorf("marshal error: %w", err)
	}

	return mcp.NewToolResultText(string(jsonAction)), nil
}

// passwordResetDroplet resets the password for a droplet
func (da *DropletActionsTool) passwordResetDroplet(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	dropletID := req.GetArguments()["ID"].(float64)

	client, err := da.client(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to get DigitalOcean client: %w", err)
	}

	action, _, err := client.DropletActions.PasswordReset(ctx, int(dropletID))
	if err != nil {
		return mcp.NewToolResultErrorFromErr("api error", err), nil
	}

	jsonAction, err := json.MarshalIndent(action, "", "  ")
	if err != nil {
		return nil, fmt.Errorf("marshal error: %w", err)
	}

	return mcp.NewToolResultText(string(jsonAction)), nil
}

// RebuildByImageSlugDroplet rebuilds a droplet using an image slug
func (da *DropletActionsTool) rebuildByImageSlugDroplet(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	dropletID := req.GetArguments()["ID"].(float64)
	imageSlug := req.GetArguments()["ImageSlug"].(string)

	client, err := da.client(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to get DigitalOcean client: %w", err)
	}

	action, _, err := client.DropletActions.RebuildByImageSlug(ctx, int(dropletID), imageSlug)
	if err != nil {
		return mcp.NewToolResultErrorFromErr("api error", err), nil
	}

	jsonAction, err := json.MarshalIndent(action, "", "  ")
	if err != nil {
		return nil, fmt.Errorf("marshal error: %w", err)
	}

	return mcp.NewToolResultText(string(jsonAction)), nil
}

// powerCycleByTag power cycles droplets by tag
func (da *DropletActionsTool) powerCycleByTag(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	tag := req.GetArguments()["Tag"].(string)

	client, err := da.client(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to get DigitalOcean client: %w", err)
	}

	actions, _, err := client.DropletActions.PowerCycleByTag(ctx, tag)
	if err != nil {
		return mcp.NewToolResultErrorFromErr("api error", err), nil
	}

	jsonActions, err := json.MarshalIndent(actions, "", "  ")
	if err != nil {
		return mcp.NewToolResultErrorFromErr("marshal error", err), nil
	}

	return mcp.NewToolResultText(string(jsonActions)), nil
}

// powerOnByTag powers on droplets by tag
func (da *DropletActionsTool) powerOnByTag(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	tag := req.GetArguments()["Tag"].(string)

	client, err := da.client(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to get DigitalOcean client: %w", err)
	}

	actions, _, err := client.DropletActions.PowerOnByTag(ctx, tag)
	if err != nil {
		return mcp.NewToolResultErrorFromErr("api error", err), nil
	}

	jsonActions, err := json.MarshalIndent(actions, "", "  ")
	if err != nil {
		return mcp.NewToolResultErrorFromErr("marshal error", err), nil
	}

	return mcp.NewToolResultText(string(jsonActions)), nil
}

// powerOffByTag powers off droplets by tag
func (da *DropletActionsTool) powerOffByTag(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	tag := req.GetArguments()["Tag"].(string)

	client, err := da.client(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to get DigitalOcean client: %w", err)
	}

	actions, _, err := client.DropletActions.PowerOffByTag(ctx, tag)
	if err != nil {
		return mcp.NewToolResultErrorFromErr("api error", err), nil
	}

	jsonActions, err := json.MarshalIndent(actions, "", "  ")
	if err != nil {
		return mcp.NewToolResultErrorFromErr("marshal error", err), nil
	}

	return mcp.NewToolResultText(string(jsonActions)), nil
}

// shutdownByTag shuts down droplets by tag
func (da *DropletActionsTool) shutdownByTag(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	tag := req.GetArguments()["Tag"].(string)

	client, err := da.client(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to get DigitalOcean client: %w", err)
	}

	actions, _, err := client.DropletActions.ShutdownByTag(ctx, tag)
	if err != nil {
		return mcp.NewToolResultErrorFromErr("api error", err), nil
	}

	jsonActions, err := json.MarshalIndent(actions, "", "  ")
	if err != nil {
		return mcp.NewToolResultErrorFromErr("marshal error", err), nil
	}

	return mcp.NewToolResultText(string(jsonActions)), nil
}

// enableBackupsByTag enables backups on droplets by tag
func (da *DropletActionsTool) enableBackupsByTag(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	tag := req.GetArguments()["Tag"].(string)

	client, err := da.client(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to get DigitalOcean client: %w", err)
	}

	actions, _, err := client.DropletActions.EnableBackupsByTag(ctx, tag)
	if err != nil {
		return mcp.NewToolResultErrorFromErr("api error", err), nil
	}

	jsonActions, err := json.MarshalIndent(actions, "", "  ")
	if err != nil {
		return mcp.NewToolResultErrorFromErr("marshal error", err), nil
	}

	return mcp.NewToolResultText(string(jsonActions)), nil
}

// disableBackupsByTag disables backups on droplets by tag
func (da *DropletActionsTool) disableBackupsByTag(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	tag := req.GetArguments()["Tag"].(string)

	client, err := da.client(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to get DigitalOcean client: %w", err)
	}

	actions, _, err := client.DropletActions.DisableBackupsByTag(ctx, tag)
	if err != nil {
		return mcp.NewToolResultErrorFromErr("api error", err), nil
	}

	jsonActions, err := json.MarshalIndent(actions, "", "  ")
	if err != nil {
		return mcp.NewToolResultErrorFromErr("marshal error", err), nil
	}

	return mcp.NewToolResultText(string(jsonActions)), nil
}

// snapshotByTag takes a snapshot of droplets by tag
func (da *DropletActionsTool) snapshotByTag(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	tag := req.GetArguments()["Tag"].(string)
	name := req.GetArguments()["Name"].(string)

	client, err := da.client(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to get DigitalOcean client: %w", err)
	}

	actions, _, err := client.DropletActions.SnapshotByTag(ctx, tag, name)
	if err != nil {
		return mcp.NewToolResultErrorFromErr("api error", err), nil
	}

	jsonActions, err := json.MarshalIndent(actions, "", "  ")
	if err != nil {
		return mcp.NewToolResultErrorFromErr("marshal error", err), nil
	}

	return mcp.NewToolResultText(string(jsonActions)), nil
}

// enableIPv6ByTag enables IPv6 on droplets by tag
func (da *DropletActionsTool) enableIPv6ByTag(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	tag := req.GetArguments()["Tag"].(string)

	client, err := da.client(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to get DigitalOcean client: %w", err)
	}

	actions, _, err := client.DropletActions.EnableIPv6ByTag(ctx, tag)
	if err != nil {
		return mcp.NewToolResultErrorFromErr("api error", err), nil
	}

	jsonActions, err := json.MarshalIndent(actions, "", "  ")
	if err != nil {
		return mcp.NewToolResultErrorFromErr("marshal error", err), nil
	}

	return mcp.NewToolResultText(string(jsonActions)), nil
}

// enablePrivateNetworkingByTag enables private networking on droplets by tag
func (da *DropletActionsTool) enablePrivateNetworkingByTag(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	tag := req.GetArguments()["Tag"].(string)

	client, err := da.client(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to get DigitalOcean client: %w", err)
	}

	actions, _, err := client.DropletActions.EnablePrivateNetworkingByTag(ctx, tag)
	if err != nil {
		return mcp.NewToolResultErrorFromErr("api error", err), nil
	}

	jsonActions, err := json.MarshalIndent(actions, "", "  ")
	if err != nil {
		return mcp.NewToolResultErrorFromErr("marshal error", err), nil
	}

	return mcp.NewToolResultText(string(jsonActions)), nil
}

// powerCycleDroplet power cycles a droplet
func (da *DropletActionsTool) powerCycleDroplet(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	dropletID := req.GetArguments()["ID"].(float64)

	client, err := da.client(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to get DigitalOcean client: %w", err)
	}

	action, _, err := client.DropletActions.PowerCycle(ctx, int(dropletID))
	if err != nil {
		return mcp.NewToolResultErrorFromErr("api error", err), nil
	}

	jsonAction, err := json.MarshalIndent(action, "", "  ")
	if err != nil {
		return nil, fmt.Errorf("marshal error: %w", err)
	}

	return mcp.NewToolResultText(string(jsonAction)), nil
}

// powerOnDroplet powers on a droplet
func (da *DropletActionsTool) powerOnDroplet(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	dropletID := req.GetArguments()["ID"].(float64)

	client, err := da.client(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to get DigitalOcean client: %w", err)
	}

	action, _, err := client.DropletActions.PowerOn(ctx, int(dropletID))
	if err != nil {
		return mcp.NewToolResultErrorFromErr("api error", err), nil
	}

	jsonAction, err := json.MarshalIndent(action, "", "  ")
	if err != nil {
		return nil, fmt.Errorf("marshal error: %w", err)
	}

	return mcp.NewToolResultText(string(jsonAction)), nil
}

// powerOffDroplet powers off a droplet
func (da *DropletActionsTool) powerOffDroplet(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	dropletID := req.GetArguments()["ID"].(float64)

	client, err := da.client(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to get DigitalOcean client: %w", err)
	}

	action, _, err := client.DropletActions.PowerOff(ctx, int(dropletID))
	if err != nil {
		return mcp.NewToolResultErrorFromErr("api error", err), nil
	}

	jsonAction, err := json.MarshalIndent(action, "", "  ")
	if err != nil {
		return nil, fmt.Errorf("marshal error: %w", err)
	}

	return mcp.NewToolResultText(string(jsonAction)), nil
}

// shutdownDroplet shuts down a droplet
func (da *DropletActionsTool) shutdownDroplet(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	dropletID := req.GetArguments()["ID"].(float64)

	client, err := da.client(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to get DigitalOcean client: %w", err)
	}

	action, _, err := client.DropletActions.Shutdown(ctx, int(dropletID))
	if err != nil {
		return mcp.NewToolResultErrorFromErr("api error", err), nil
	}

	jsonAction, err := json.MarshalIndent(action, "", "  ")
	if err != nil {
		return nil, fmt.Errorf("marshal error: %w", err)
	}

	return mcp.NewToolResultText(string(jsonAction)), nil
}

// restoreDroplet restores a droplet to a backup image
func (da *DropletActionsTool) restoreDroplet(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	dropletID := req.GetArguments()["ID"].(float64)
	imageID := req.GetArguments()["ImageID"].(float64)

	client, err := da.client(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to get DigitalOcean client: %w", err)
	}

	action, _, err := client.DropletActions.Restore(ctx, int(dropletID), int(imageID))
	if err != nil {
		return mcp.NewToolResultErrorFromErr("api error", err), nil
	}

	jsonAction, err := json.MarshalIndent(action, "", "  ")
	if err != nil {
		return nil, fmt.Errorf("marshal error: %w", err)
	}

	return mcp.NewToolResultText(string(jsonAction)), nil
}

// resizeDroplet resizes a droplet
func (da *DropletActionsTool) resizeDroplet(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	dropletID := req.GetArguments()["ID"].(float64)
	size := req.GetArguments()["Size"].(string)
	resizeDisk, _ := req.GetArguments()["ResizeDisk"].(bool) // Defaults to false

	client, err := da.client(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to get DigitalOcean client: %w", err)
	}

	action, _, err := client.DropletActions.Resize(ctx, int(dropletID), size, resizeDisk)
	if err != nil {
		return mcp.NewToolResultErrorFromErr("api error", err), nil
	}

	jsonAction, err := json.MarshalIndent(action, "", "  ")
	if err != nil {
		return nil, fmt.Errorf("marshal error: %w", err)
	}

	return mcp.NewToolResultText(string(jsonAction)), nil
}

// rebuildDroplet rebuilds a droplet using a provided image
func (da *DropletActionsTool) rebuildDroplet(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	dropletID := req.GetArguments()["ID"].(float64)
	imageID := req.GetArguments()["ImageID"].(float64)

	client, err := da.client(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to get DigitalOcean client: %w", err)
	}

	action, _, err := client.DropletActions.RebuildByImageID(ctx, int(dropletID), int(imageID))
	if err != nil {
		return mcp.NewToolResultErrorFromErr("api error", err), nil
	}

	jsonAction, err := json.MarshalIndent(action, "", "  ")
	if err != nil {
		return nil, fmt.Errorf("marshal error: %w", err)
	}

	return mcp.NewToolResultText(string(jsonAction)), nil
}

// renameDroplet renames a droplet
func (da *DropletActionsTool) renameDroplet(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	dropletID := req.GetArguments()["ID"].(float64)
	name := req.GetArguments()["Name"].(string)

	client, err := da.client(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to get DigitalOcean client: %w", err)
	}

	action, _, err := client.DropletActions.Rename(ctx, int(dropletID), name)
	if err != nil {
		return mcp.NewToolResultErrorFromErr("api error", err), nil
	}

	jsonAction, err := json.MarshalIndent(action, "", "  ")
	if err != nil {
		return nil, fmt.Errorf("marshal error: %w", err)
	}

	return mcp.NewToolResultText(string(jsonAction)), nil
}

// changeKernel changes a droplet's kernel
func (da *DropletActionsTool) changeKernel(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	dropletID := req.GetArguments()["ID"].(float64)
	kernelID := req.GetArguments()["KernelID"].(float64)

	client, err := da.client(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to get DigitalOcean client: %w", err)
	}

	action, _, err := client.DropletActions.ChangeKernel(ctx, int(dropletID), int(kernelID))
	if err != nil {
		return mcp.NewToolResultErrorFromErr("api error", err), nil
	}

	jsonAction, err := json.MarshalIndent(action, "", "  ")
	if err != nil {
		return nil, fmt.Errorf("marshal error: %w", err)
	}

	return mcp.NewToolResultText(string(jsonAction)), nil
}

// enableIPv6 enables IPv6 on a droplet
func (da *DropletActionsTool) enableIPv6(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	dropletID := req.GetArguments()["ID"].(float64)

	client, err := da.client(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to get DigitalOcean client: %w", err)
	}

	action, _, err := client.DropletActions.EnableIPv6(ctx, int(dropletID))
	if err != nil {
		return mcp.NewToolResultErrorFromErr("api error", err), nil
	}

	jsonAction, err := json.MarshalIndent(action, "", "  ")
	if err != nil {
		return nil, fmt.Errorf("marshal error: %w", err)
	}

	return mcp.NewToolResultText(string(jsonAction)), nil
}

// enableBackups enables backups on a droplet
func (da *DropletActionsTool) enableBackups(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	dropletID := req.GetArguments()["ID"].(float64)

	client, err := da.client(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to get DigitalOcean client: %w", err)
	}

	action, _, err := client.DropletActions.EnableBackups(ctx, int(dropletID))
	if err != nil {
		return mcp.NewToolResultErrorFromErr("api error", err), nil
	}

	jsonAction, err := json.MarshalIndent(action, "", "  ")
	if err != nil {
		return nil, fmt.Errorf("marshal error: %w", err)
	}

	return mcp.NewToolResultText(string(jsonAction)), nil
}

// disableBackups disables backups on a droplet
func (da *DropletActionsTool) disableBackups(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	dropletID := req.GetArguments()["ID"].(float64)

	client, err := da.client(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to get DigitalOcean client: %w", err)
	}

	action, _, err := client.DropletActions.DisableBackups(ctx, int(dropletID))
	if err != nil {
		return mcp.NewToolResultErrorFromErr("api error", err), nil
	}

	jsonAction, err := json.MarshalIndent(action, "", "  ")
	if err != nil {
		return nil, fmt.Errorf("marshal error: %w", err)
	}

	return mcp.NewToolResultText(string(jsonAction)), nil
}

// snapshotDroplet creates a snapshot of a droplet
func (da *DropletActionsTool) snapshotDroplet(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	dropletID := req.GetArguments()["ID"].(float64)
	name := req.GetArguments()["Name"].(string)

	client, err := da.client(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to get DigitalOcean client: %w", err)
	}

	action, _, err := client.DropletActions.Snapshot(ctx, int(dropletID), name)
	if err != nil {
		return mcp.NewToolResultErrorFromErr("api error", err), nil
	}

	jsonAction, err := json.MarshalIndent(action, "", "  ")
	if err != nil {
		return nil, fmt.Errorf("marshal error: %w", err)
	}

	return mcp.NewToolResultText(string(jsonAction)), nil
}

// Tools returns a list of tool functions
func (da *DropletActionsTool) Tools() []server.ServerTool {
	tools := []server.ServerTool{
		{
			Handler: da.rebootDroplet,
			Tool: mcp.NewTool("reboot-droplet",
				mcp.WithDescription("Reboot a droplet"),
				mcp.WithNumber("ID", mcp.Required(), mcp.Description("ID of the droplet to reboot")),
			),
		},
		{
			Handler: da.passwordResetDroplet,
			Tool: mcp.NewTool("reset-droplet-password",
				mcp.WithDescription("Reset password for a droplet"),
				mcp.WithNumber("ID", mcp.Required(), mcp.Description("ID of the droplet")),
			),
		},
		{
			Handler: da.rebuildByImageSlugDroplet,
			Tool: mcp.NewTool("rebuild-droplet-by-slug",
				mcp.WithDescription("Rebuild a droplet using an image slug"),
				mcp.WithNumber("ID", mcp.Required(), mcp.Description("ID of the droplet to rebuild")),
				mcp.WithString("ImageSlug", mcp.Required(), mcp.Description("Slug of the image to rebuild from")),
			),
		},
		{
			Handler: da.powerCycleByTag,
			Tool: mcp.NewTool("power-cycle-droplets-tag",
				mcp.WithDescription("Power cycle droplets by tag"),
				mcp.WithString("Tag", mcp.Required(), mcp.Description("Tag of the droplets to power cycle")),
			),
		},
		{
			Handler: da.powerOnByTag,
			Tool: mcp.NewTool("power-on-droplets-tag",
				mcp.WithDescription("Power on droplets by tag"),
				mcp.WithString("Tag", mcp.Required(), mcp.Description("Tag of the droplets to power on")),
			),
		},
		{
			Handler: da.powerOffByTag,
			Tool: mcp.NewTool("power-off-droplets-tag",
				mcp.WithDescription("Power off droplets by tag"),
				mcp.WithString("Tag", mcp.Required(), mcp.Description("Tag of the droplets to power off")),
			),
		},
		{
			Handler: da.shutdownByTag,
			Tool: mcp.NewTool("shutdown-droplets-tag",
				mcp.WithDescription("Shutdown droplets by tag"),
				mcp.WithString("Tag", mcp.Required(), mcp.Description("Tag of the droplets to shutdown")),
			),
		},
		{
			Handler: da.enableBackupsByTag,
			Tool: mcp.NewTool("enable-backups-droplets-tag",
				mcp.WithDescription("Enable backups on droplets by tag"),
				mcp.WithString("Tag", mcp.Required(), mcp.Description("Tag of the droplets")),
			),
		},
		{
			Handler: da.disableBackupsByTag,
			Tool: mcp.NewTool("disable-backups-droplets-tag",
				mcp.WithDescription("Disable backups on droplets by tag"),
				mcp.WithString("Tag", mcp.Required(), mcp.Description("Tag of the droplets")),
			),
		},
		{
			Handler: da.snapshotByTag,
			Tool: mcp.NewTool("snapshot-droplets-tag",
				mcp.WithDescription("Take a snapshot of droplets by tag"),
				mcp.WithString("Tag", mcp.Required(), mcp.Description("Tag of the droplets")),
				mcp.WithString("Name", mcp.Required(), mcp.Description("Name for the snapshot")),
			),
		},
		{
			Handler: da.enableIPv6ByTag,
			Tool: mcp.NewTool("enable-ipv6-droplets-tag",
				mcp.WithDescription("Enable IPv6 on droplets by tag"),
				mcp.WithString("Tag", mcp.Required(), mcp.Description("Tag of the droplets")),
			),
		},
		{
			Handler: da.enablePrivateNetworkingByTag,
			Tool: mcp.NewTool("enable-private-net-droplets-tag",
				mcp.WithDescription("Enable private networking on droplets by tag"),
				mcp.WithString("Tag", mcp.Required(), mcp.Description("Tag of the droplets")),
			),
		},
		{
			Handler: da.powerCycleDroplet,
			Tool: mcp.NewTool("power-cycle-droplet",
				mcp.WithDescription("Power cycle a droplet"),
				mcp.WithNumber("ID", mcp.Required(), mcp.Description("ID of the droplet to power cycle")),
			),
		},
		{
			Handler: da.powerOnDroplet,
			Tool: mcp.NewTool("power-on-droplet",
				mcp.WithDescription("Power on a droplet"),
				mcp.WithNumber("ID", mcp.Required(), mcp.Description("ID of the droplet to power on")),
			),
		},
		{
			Handler: da.powerOffDroplet,
			Tool: mcp.NewTool("power-off-droplet",
				mcp.WithDescription("Power off a droplet"),
				mcp.WithNumber("ID", mcp.Required(), mcp.Description("ID of the droplet to power off")),
			),
		},
		{
			Handler: da.shutdownDroplet,
			Tool: mcp.NewTool("shutdown-droplet",
				mcp.WithDescription("Shutdown a droplet"),
				mcp.WithNumber("ID", mcp.Required(), mcp.Description("ID of the droplet to shutdown")),
			),
		},
		{
			Handler: da.restoreDroplet,
			Tool: mcp.NewTool("restore-droplet",
				mcp.WithDescription("Restore a droplet from a backup/snapshot"),
				mcp.WithNumber("ID", mcp.Required(), mcp.Description("ID of the droplet to restore")),
				mcp.WithNumber("ImageID", mcp.Required(), mcp.Description("ID of the backup/snapshot image")),
			),
		},
		{
			Handler: da.resizeDroplet,
			Tool: mcp.NewTool("resize-droplet",
				mcp.WithDescription("Resize a droplet"),
				mcp.WithNumber("ID", mcp.Required(), mcp.Description("ID of the droplet to resize")),
				mcp.WithString("Size", mcp.Required(), mcp.Description("Slug of the new size (e.g., s-1vcpu-1gb)")),
				mcp.WithBoolean("ResizeDisk", mcp.DefaultBool(false), mcp.Description("Whether to resize the disk")),
			),
		},
		{
			Handler: da.rebuildDroplet,
			Tool: mcp.NewTool("rebuild-droplet",
				mcp.WithDescription("Rebuild a droplet from an image"),
				mcp.WithNumber("ID", mcp.Required(), mcp.Description("ID of the droplet to rebuild")),
				mcp.WithNumber("ImageID", mcp.Required(), mcp.Description("ID of the image to rebuild from")),
			),
		},
		{
			Handler: da.renameDroplet,
			Tool: mcp.NewTool("rename-droplet",
				mcp.WithDescription("Rename a droplet"),
				mcp.WithNumber("ID", mcp.Required(), mcp.Description("ID of the droplet to rename")),
				mcp.WithString("Name", mcp.Required(), mcp.Description("New name for the droplet")),
			),
		},
		{
			Handler: da.changeKernel,
			Tool: mcp.NewTool("change-kernel-droplet",
				mcp.WithDescription("Change a droplet's kernel"),
				mcp.WithNumber("ID", mcp.Required(), mcp.Description("ID of the droplet")),
				mcp.WithNumber("KernelID", mcp.Required(), mcp.Description("ID of the kernel to switch to")),
			),
		},
		{
			Handler: da.enableIPv6,
			Tool: mcp.NewTool("enable-ipv6-droplet",
				mcp.WithDescription("Enable IPv6 on a droplet"),
				mcp.WithNumber("ID", mcp.Required(), mcp.Description("ID of the droplet")),
			),
		},
		{
			Handler: da.enableBackups,
			Tool: mcp.NewTool("enable-backups-droplet",
				mcp.WithDescription("Enable backups on a droplet"),
				mcp.WithNumber("ID", mcp.Required(), mcp.Description("ID of the droplet")),
			),
		},
		{
			Handler: da.disableBackups,
			Tool: mcp.NewTool("disable-backups-droplet",
				mcp.WithDescription("Disable backups on a droplet"),
				mcp.WithNumber("ID", mcp.Required(), mcp.Description("ID of the droplet")),
			),
		},
		{
			Handler: da.snapshotDroplet,
			Tool: mcp.NewTool("snapshot-droplet",
				mcp.WithDescription("Take a snapshot of a droplet"),
				mcp.WithNumber("ID", mcp.Required(), mcp.Description("ID of the droplet")),
				mcp.WithString("Name", mcp.Required(), mcp.Description("Name for the snapshot")),
			),
		},
	}
	return tools
}
