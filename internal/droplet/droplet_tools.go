package droplet

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/digitalocean/godo"
	"github.com/mark3labs/mcp-go/mcp"
	"github.com/mark3labs/mcp-go/server"
)

// DropletTool provides droplet management tools
type DropletTool struct {
	client *godo.Client
}

// NewDropletTool creates a new droplet tool
func NewDropletTool(client *godo.Client) *DropletTool {
	return &DropletTool{
		client: client,
	}
}

// CreateDroplet creates a new droplet
func (d *DropletTool) createDroplet(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	args := req.GetArguments()
	dropletName := args["Name"].(string)
	size := args["Size"].(string)
	imageID := args["ImageID"].(float64)
	region := args["Region"].(string)
	backup, _ := args["Backup"].(bool)         // Defaults to false
	monitoring, _ := args["Monitoring"].(bool) // Defaults to false
	// Create the droplet
	dropletCreateRequest := &godo.DropletCreateRequest{
		Name:       dropletName,
		Size:       size,
		Image:      godo.DropletCreateImage{ID: int(imageID)},
		Region:     region,
		Backups:    backup,
		Monitoring: monitoring,
	}
	droplet, _, err := d.client.Droplets.Create(ctx, dropletCreateRequest)
	if err != nil {
		return mcp.NewToolResultErrorFromErr("droplet create", err), nil
	}
	jsonDroplet, err := json.MarshalIndent(droplet, "", "  ")
	if err != nil {
		return mcp.NewToolResultErrorFromErr("json marshal", err), nil
	}
	return mcp.NewToolResultText(string(jsonDroplet)), nil
}

// deleteDroplet deletes a droplet
func (d *DropletTool) deleteDroplet(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	dropletID := req.GetArguments()["ID"].(float64)
	_, err := d.client.Droplets.Delete(ctx, int(dropletID))
	if err != nil {
		return mcp.NewToolResultErrorFromErr("api error", err), nil
	}
	return mcp.NewToolResultText("Droplet deleted successfully"), nil
}

// powerCycleDroplet power cycles a droplet
func (d *DropletTool) powerCycleDroplet(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	dropletID := req.GetArguments()["ID"].(float64)
	action, _, err := d.client.DropletActions.PowerCycle(ctx, int(dropletID))
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
func (d *DropletTool) powerOnDroplet(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	dropletID := req.GetArguments()["ID"].(float64)
	action, _, err := d.client.DropletActions.PowerOn(ctx, int(dropletID))
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
func (d *DropletTool) powerOffDroplet(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	dropletID := req.GetArguments()["ID"].(float64)
	action, _, err := d.client.DropletActions.PowerOff(ctx, int(dropletID))
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
func (d *DropletTool) shutdownDroplet(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	dropletID := req.GetArguments()["ID"].(float64)
	action, _, err := d.client.DropletActions.Shutdown(ctx, int(dropletID))
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
func (d *DropletTool) restoreDroplet(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	dropletID := req.GetArguments()["ID"].(float64)
	imageID := req.GetArguments()["ImageID"].(float64)
	action, _, err := d.client.DropletActions.Restore(ctx, int(dropletID), int(imageID))
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
func (d *DropletTool) resizeDroplet(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	dropletID := req.GetArguments()["ID"].(float64)
	size := req.GetArguments()["Size"].(string)
	resizeDisk, _ := req.GetArguments()["ResizeDisk"].(bool) // Defaults to false
	action, _, err := d.client.DropletActions.Resize(ctx, int(dropletID), size, resizeDisk)
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
func (d *DropletTool) rebuildDroplet(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	dropletID := req.GetArguments()["ID"].(float64)
	imageID := req.GetArguments()["ImageID"].(float64)
	action, _, err := d.client.DropletActions.RebuildByImageID(ctx, int(dropletID), int(imageID))
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
func (d *DropletTool) renameDroplet(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	dropletID := req.GetArguments()["ID"].(float64)
	name := req.GetArguments()["Name"].(string)
	action, _, err := d.client.DropletActions.Rename(ctx, int(dropletID), name)
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
func (d *DropletTool) changeKernel(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	dropletID := req.GetArguments()["ID"].(float64)
	kernelID := req.GetArguments()["KernelID"].(float64)
	action, _, err := d.client.DropletActions.ChangeKernel(ctx, int(dropletID), int(kernelID))
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
func (d *DropletTool) enableIPv6(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	dropletID := req.GetArguments()["ID"].(float64)
	action, _, err := d.client.DropletActions.EnableIPv6(ctx, int(dropletID))
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
func (d *DropletTool) enableBackups(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	dropletID := req.GetArguments()["ID"].(float64)
	action, _, err := d.client.DropletActions.EnableBackups(ctx, int(dropletID))
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
func (d *DropletTool) disableBackups(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	dropletID := req.GetArguments()["ID"].(float64)
	action, _, err := d.client.DropletActions.DisableBackups(ctx, int(dropletID))
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
func (d *DropletTool) snapshotDroplet(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	dropletID := req.GetArguments()["ID"].(float64)
	name := req.GetArguments()["Name"].(string)
	action, _, err := d.client.DropletActions.Snapshot(ctx, int(dropletID), name)
	if err != nil {
		return mcp.NewToolResultErrorFromErr("api error", err), nil
	}

	jsonAction, err := json.MarshalIndent(action, "", "  ")
	if err != nil {
		return nil, fmt.Errorf("marshal error: %w", err)
	}

	return mcp.NewToolResultText(string(jsonAction)), nil
}

// getDropletNeighbors gets a droplet's neighbors
func (d *DropletTool) getDropletNeighbors(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	dropletID := req.GetArguments()["ID"].(float64)
	neighbors, _, err := d.client.Droplets.Neighbors(ctx, int(dropletID))
	if err != nil {
		return mcp.NewToolResultErrorFromErr("api error", err), nil
	}

	jsonNeighbors, err := json.MarshalIndent(neighbors, "", "  ")
	if err != nil {
		return nil, fmt.Errorf("marshal error: %w", err)
	}

	return mcp.NewToolResultText(string(jsonNeighbors)), nil
}

// enablePrivateNetworking enables private networking on a droplet
func (d *DropletTool) enablePrivateNetworking(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	dropletID := req.GetArguments()["ID"].(float64)
	action, _, err := d.client.DropletActions.EnablePrivateNetworking(ctx, int(dropletID))
	if err != nil {
		return mcp.NewToolResultErrorFromErr("api error", err), nil
	}

	jsonAction, err := json.MarshalIndent(action, "", "  ")
	if err != nil {
		return nil, fmt.Errorf("marshal error: %w", err)
	}

	return mcp.NewToolResultText(string(jsonAction)), nil
}

// getDropletKernels gets available kernels for a droplet
func (d *DropletTool) getDropletKernels(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	dropletID := req.GetArguments()["ID"].(float64)

	// Use list options to get all kernels
	opt := &godo.ListOptions{
		Page:    1,
		PerPage: 100,
	}

	kernels, _, err := d.client.Droplets.Kernels(ctx, int(dropletID), opt)
	if err != nil {
		return mcp.NewToolResultErrorFromErr("api error", err), nil
	}

	jsonKernels, err := json.MarshalIndent(kernels, "", "  ")
	if err != nil {
		return nil, fmt.Errorf("marshal error: %w", err)
	}

	return mcp.NewToolResultText(string(jsonKernels)), nil
}

// rebootDroplet reboots a droplet
func (d *DropletTool) rebootDroplet(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	dropletID := req.GetArguments()["ID"].(float64)
	action, _, err := d.client.DropletActions.Reboot(ctx, int(dropletID))
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
func (d *DropletTool) passwordResetDroplet(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	dropletID := req.GetArguments()["ID"].(float64)
	action, _, err := d.client.DropletActions.PasswordReset(ctx, int(dropletID))
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
func (d *DropletTool) rebuildByImageSlugDroplet(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	dropletID := req.GetArguments()["ID"].(float64)
	imageSlug := req.GetArguments()["ImageSlug"].(string)
	action, _, err := d.client.DropletActions.RebuildByImageSlug(ctx, int(dropletID), imageSlug)
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
func (d *DropletTool) powerCycleByTag(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	tag := req.GetArguments()["Tag"].(string)
	actions, _, err := d.client.DropletActions.PowerCycleByTag(ctx, tag)
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
func (d *DropletTool) powerOnByTag(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	tag := req.GetArguments()["Tag"].(string)
	actions, _, err := d.client.DropletActions.PowerOnByTag(ctx, tag)
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
func (d *DropletTool) powerOffByTag(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	tag := req.GetArguments()["Tag"].(string)
	actions, _, err := d.client.DropletActions.PowerOffByTag(ctx, tag)
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
func (d *DropletTool) shutdownByTag(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	tag := req.GetArguments()["Tag"].(string)
	actions, _, err := d.client.DropletActions.ShutdownByTag(ctx, tag)
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
func (d *DropletTool) enableBackupsByTag(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	tag := req.GetArguments()["Tag"].(string)
	actions, _, err := d.client.DropletActions.EnableBackupsByTag(ctx, tag)
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
func (d *DropletTool) disableBackupsByTag(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	tag := req.GetArguments()["Tag"].(string)
	actions, _, err := d.client.DropletActions.DisableBackupsByTag(ctx, tag)
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
func (d *DropletTool) snapshotByTag(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	tag := req.GetArguments()["Tag"].(string)
	name := req.GetArguments()["Name"].(string)
	actions, _, err := d.client.DropletActions.SnapshotByTag(ctx, tag, name)
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
func (d *DropletTool) enableIPv6ByTag(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	tag := req.GetArguments()["Tag"].(string)
	actions, _, err := d.client.DropletActions.EnableIPv6ByTag(ctx, tag)
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
func (d *DropletTool) enablePrivateNetworkingByTag(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	tag := req.GetArguments()["Tag"].(string)
	actions, _, err := d.client.DropletActions.EnablePrivateNetworkingByTag(ctx, tag)
	if err != nil {
		return mcp.NewToolResultErrorFromErr("api error", err), nil
	}

	jsonActions, err := json.MarshalIndent(actions, "", "  ")
	if err != nil {
		return mcp.NewToolResultErrorFromErr("marshal error", err), nil
	}

	return mcp.NewToolResultText(string(jsonActions)), nil
}

// Tools returns a list of tool functions
func (d *DropletTool) Tools() []server.ServerTool {
	return []server.ServerTool{
		{
			Handler: d.createDroplet,
			Tool: mcp.NewTool("digitalocean-droplet-create",
				mcp.WithDescription("Create a new droplet"),
				mcp.WithString("Name", mcp.Required(), mcp.Description("Name of the droplet")),
				mcp.WithString("Size", mcp.Required(), mcp.Description("Slug of the droplet size (e.g., s-1vcpu-1gb)")),
				mcp.WithNumber("ImageID", mcp.Required(), mcp.Description("ID of the image to use")),
				mcp.WithString("Region", mcp.Required(), mcp.Description("Slug of the region (e.g., nyc3)")),
				mcp.WithBoolean("Backup", mcp.DefaultBool(false), mcp.Description("Whether to enable backups")),
				mcp.WithBoolean("Monitoring", mcp.DefaultBool(false), mcp.Description("Whether to enable monitoring")),
			),
		},
		{
			Handler: d.deleteDroplet,
			Tool: mcp.NewTool("digitalocean-droplet-delete",
				mcp.WithDescription("Delete a droplet"),
				mcp.WithNumber("ID", mcp.Required(), mcp.Description("ID of the droplet to delete")),
			),
		},
		{
			Handler: d.powerCycleDroplet,
			Tool: mcp.NewTool("digitalocean-droplet-power-cycle",
				mcp.WithDescription("Power cycle a droplet"),
				mcp.WithNumber("ID", mcp.Required(), mcp.Description("ID of the droplet to power cycle")),
			),
		},
		{
			Handler: d.powerOnDroplet,
			Tool: mcp.NewTool("digitalocean-droplet-power-on",
				mcp.WithDescription("Power on a droplet"),
				mcp.WithNumber("ID", mcp.Required(), mcp.Description("ID of the droplet to power on")),
			),
		},
		{
			Handler: d.powerOffDroplet,
			Tool: mcp.NewTool("digitalocean-droplet-power-off",
				mcp.WithDescription("Power off a droplet"),
				mcp.WithNumber("ID", mcp.Required(), mcp.Description("ID of the droplet to power off")),
			),
		},
		{
			Handler: d.shutdownDroplet,
			Tool: mcp.NewTool("digitalocean-droplet-shutdown",
				mcp.WithDescription("Shutdown a droplet"),
				mcp.WithNumber("ID", mcp.Required(), mcp.Description("ID of the droplet to shutdown")),
			),
		},
		{
			Handler: d.restoreDroplet,
			Tool: mcp.NewTool("digitalocean-droplet-restore",
				mcp.WithDescription("Restore a droplet from a backup/snapshot"),
				mcp.WithNumber("ID", mcp.Required(), mcp.Description("ID of the droplet to restore")),
				mcp.WithNumber("ImageID", mcp.Required(), mcp.Description("ID of the backup/snapshot image")),
			),
		},
		{
			Handler: d.resizeDroplet,
			Tool: mcp.NewTool("digitalocean-droplet-resize",
				mcp.WithDescription("Resize a droplet"),
				mcp.WithNumber("ID", mcp.Required(), mcp.Description("ID of the droplet to resize")),
				mcp.WithString("Size", mcp.Required(), mcp.Description("Slug of the new size (e.g., s-1vcpu-1gb)")),
				mcp.WithBoolean("ResizeDisk", mcp.DefaultBool(false), mcp.Description("Whether to resize the disk")),
			),
		},
		{
			Handler: d.rebuildDroplet,
			Tool: mcp.NewTool("digitalocean-droplet-rebuild",
				mcp.WithDescription("Rebuild a droplet from an image"),
				mcp.WithNumber("ID", mcp.Required(), mcp.Description("ID of the droplet to rebuild")),
				mcp.WithNumber("ImageID", mcp.Required(), mcp.Description("ID of the image to rebuild from")),
			),
		},
		{
			Handler: d.renameDroplet,
			Tool: mcp.NewTool("digitalocean-droplet-rename",
				mcp.WithDescription("Rename a droplet"),
				mcp.WithNumber("ID", mcp.Required(), mcp.Description("ID of the droplet to rename")),
				mcp.WithString("Name", mcp.Required(), mcp.Description("New name for the droplet")),
			),
		},
		{
			Handler: d.changeKernel,
			Tool: mcp.NewTool("digitalocean-droplet-change-kernel",
				mcp.WithDescription("Change a droplet's kernel"),
				mcp.WithNumber("ID", mcp.Required(), mcp.Description("ID of the droplet")),
				mcp.WithNumber("KernelID", mcp.Required(), mcp.Description("ID of the kernel to switch to")),
			),
		},
		{
			Handler: d.enableIPv6,
			Tool: mcp.NewTool("digitalocean-droplet-enable-ipv6",
				mcp.WithDescription("Enable IPv6 on a droplet"),
				mcp.WithNumber("ID", mcp.Required(), mcp.Description("ID of the droplet")),
			),
		},
		{
			Handler: d.enableBackups,
			Tool: mcp.NewTool("digitalocean-droplet-enable-backups",
				mcp.WithDescription("Enable backups on a droplet"),
				mcp.WithNumber("ID", mcp.Required(), mcp.Description("ID of the droplet")),
			),
		},
		{
			Handler: d.disableBackups,
			Tool: mcp.NewTool("digitalocean-droplet-disable-backups",
				mcp.WithDescription("Disable backups on a droplet"),
				mcp.WithNumber("ID", mcp.Required(), mcp.Description("ID of the droplet")),
			),
		},
		{
			Handler: d.snapshotDroplet,
			Tool: mcp.NewTool("digitalocean-droplet-snapshot",
				mcp.WithDescription("Take a snapshot of a droplet"),
				mcp.WithNumber("ID", mcp.Required(), mcp.Description("ID of the droplet")),
				mcp.WithString("Name", mcp.Required(), mcp.Description("Name for the snapshot")),
			),
		},
		{
			Handler: d.getDropletNeighbors,
			Tool: mcp.NewTool("digitalocean-droplet-get-neighbors",
				mcp.WithDescription("Get neighbors of a droplet"),
				mcp.WithNumber("ID", mcp.Required(), mcp.Description("ID of the droplet")),
			),
		},
		{
			Handler: d.enablePrivateNetworking,
			Tool: mcp.NewTool("digitalocean-droplet-enable-private-net",
				mcp.WithDescription("Enable private networking on a droplet"),
				mcp.WithNumber("ID", mcp.Required(), mcp.Description("ID of the droplet")),
			),
		},
		{
			Handler: d.getDropletKernels,
			Tool: mcp.NewTool("digitalocean-droplet-get-kernels",
				mcp.WithDescription("Get available kernels for a droplet"),
				mcp.WithNumber("ID", mcp.Required(), mcp.Description("ID of the droplet")),
			),
		},
		{
			Handler: d.rebootDroplet,
			Tool: mcp.NewTool("digitalocean-droplet-reboot",
				mcp.WithDescription("Reboot a droplet"),
				mcp.WithNumber("ID", mcp.Required(), mcp.Description("ID of the droplet to reboot")),
			),
		},
		{
			Handler: d.passwordResetDroplet,
			Tool: mcp.NewTool("digitalocean-droplet-password-reset",
				mcp.WithDescription("Reset password for a droplet"),
				mcp.WithNumber("ID", mcp.Required(), mcp.Description("ID of the droplet")),
			),
		},
		{
			Handler: d.rebuildByImageSlugDroplet,
			Tool: mcp.NewTool("digitalocean-droplet-rebuild-by-slug",
				mcp.WithDescription("Rebuild a droplet using an image slug"),
				mcp.WithNumber("ID", mcp.Required(), mcp.Description("ID of the droplet to rebuild")),
				mcp.WithString("ImageSlug", mcp.Required(), mcp.Description("Slug of the image to rebuild from")),
			),
		},
		{
			Handler: d.powerCycleByTag,
			Tool: mcp.NewTool("digitalocean-droplet-power-cycle-by-tag",
				mcp.WithDescription("Power cycle droplets by tag"),
				mcp.WithString("Tag", mcp.Required(), mcp.Description("Tag of the droplets to power cycle")),
			),
		},
		{
			Handler: d.powerOnByTag,
			Tool: mcp.NewTool("digitalocean-droplet-power-on-by-tag",
				mcp.WithDescription("Power on droplets by tag"),
				mcp.WithString("Tag", mcp.Required(), mcp.Description("Tag of the droplets to power on")),
			),
		},
		{
			Handler: d.powerOffByTag,
			Tool: mcp.NewTool("digitalocean-droplet-power-off-by-tag",
				mcp.WithDescription("Power off droplets by tag"),
				mcp.WithString("Tag", mcp.Required(), mcp.Description("Tag of the droplets to power off")),
			),
		},
		{
			Handler: d.shutdownByTag,
			Tool: mcp.NewTool("digitalocean-droplet-shutdown-by-tag",
				mcp.WithDescription("Shutdown droplets by tag"),
				mcp.WithString("Tag", mcp.Required(), mcp.Description("Tag of the droplets to shutdown")),
			),
		},
		{
			Handler: d.enableBackupsByTag,
			Tool: mcp.NewTool("digitalocean-droplet-enable-backups-by-tag",
				mcp.WithDescription("Enable backups on droplets by tag"),
				mcp.WithString("Tag", mcp.Required(), mcp.Description("Tag of the droplets")),
			),
		},
		{
			Handler: d.disableBackupsByTag,
			Tool: mcp.NewTool("digitalocean-droplet-disable-backups-by-tag",
				mcp.WithDescription("Disable backups on droplets by tag"),
				mcp.WithString("Tag", mcp.Required(), mcp.Description("Tag of the droplets")),
			),
		},
		{
			Handler: d.snapshotByTag,
			Tool: mcp.NewTool("digitalocean-droplet-snapshot-by-tag",
				mcp.WithDescription("Take a snapshot of droplets by tag"),
				mcp.WithString("Tag", mcp.Required(), mcp.Description("Tag of the droplets")),
				mcp.WithString("Name", mcp.Required(), mcp.Description("Name for the snapshot")),
			),
		},
		{
			Handler: d.enableIPv6ByTag,
			Tool: mcp.NewTool("digitalocean-droplet-enable-ipv6-by-tag",
				mcp.WithDescription("Enable IPv6 on droplets by tag"),
				mcp.WithString("Tag", mcp.Required(), mcp.Description("Tag of the droplets")),
			),
		},
		{
			Handler: d.enablePrivateNetworkingByTag,
			Tool: mcp.NewTool("digitalocean-droplet-enable-private-net-by-tag",
				mcp.WithDescription("Enable private networking on droplets by tag"),
				mcp.WithString("Tag", mcp.Required(), mcp.Description("Tag of the droplets")),
			),
		},
	}
}
