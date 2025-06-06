package tools

import (
	"context"
	"encoding/json"

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

// Create Droplet creates a new droplet
func (d *DropletTool) CreateDroplet(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	dropletName, err := req.RequireString("Name")
	if err != nil {
		return nil, err
	}
	
	size, err := req.RequireString("Size")
	if err != nil {
		return nil, err
	}
	
	imageIDFloat, err := req.RequireFloat("ImageID")
	if err != nil {
		return nil, err
	}
	imageID := int(imageIDFloat)
	
	region, err := req.RequireString("Region")
	if err != nil {
		return nil, err
	}
	
	backup := req.GetBool("Backup", false)
	monitoring := req.GetBool("Monitoring", false)
	
	// Create the droplet
	dropletCreateRequest := &godo.DropletCreateRequest{
		Name:       dropletName,
		Size:       size,
		Image:      godo.DropletCreateImage{ID: imageID},
		Region:     region,
		Backups:    backup,
		Monitoring: monitoring,
	}
	droplet, _, err := d.client.Droplets.Create(ctx, dropletCreateRequest)
	if err != nil {
		return nil, err
	}
	jsonDroplet, err := json.MarshalIndent(droplet, "", "  ")
	if err != nil {
		return nil, err
	}
	return mcp.NewToolResultText(string(jsonDroplet)), nil
}

// DeleteDroplet deletes a droplet
func (d *DropletTool) DeleteDroplet(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	dropletIDFloat, err := req.RequireFloat("ID")
	if err != nil {
		return nil, err
	}
	dropletID := int(dropletIDFloat)
	
	_, err = d.client.Droplets.Delete(ctx, dropletID)
	if err != nil {
		return nil, err
	}
	return mcp.NewToolResultText("Droplet deleted successfully"), nil
}

// PowerCycleDroplet power cycles a droplet
func (d *DropletTool) PowerCycleDroplet(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	dropletIDFloat, err := req.RequireFloat("ID")
	if err != nil {
		return nil, err
	}
	dropletID := int(dropletIDFloat)
	
	action, _, err := d.client.DropletActions.PowerCycle(ctx, dropletID)
	if err != nil {
		return nil, err
	}

	jsonAction, err := json.MarshalIndent(action, "", "  ")
	if err != nil {
		return nil, err
	}

	return mcp.NewToolResultText(string(jsonAction)), nil
}

// PowerOnDroplet powers on a droplet
func (d *DropletTool) PowerOnDroplet(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	dropletIDFloat, err := req.RequireFloat("ID")
	if err != nil {
		return nil, err
	}
	dropletID := int(dropletIDFloat)
	
	action, _, err := d.client.DropletActions.PowerOn(ctx, dropletID)
	if err != nil {
		return nil, err
	}

	jsonAction, err := json.MarshalIndent(action, "", "  ")
	if err != nil {
		return nil, err
	}

	return mcp.NewToolResultText(string(jsonAction)), nil
}

// PowerOffDroplet powers off a droplet
func (d *DropletTool) PowerOffDroplet(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	dropletIDFloat, err := req.RequireFloat("ID")
	if err != nil {
		return nil, err
	}
	dropletID := int(dropletIDFloat)
	
	action, _, err := d.client.DropletActions.PowerOff(ctx, dropletID)
	if err != nil {
		return nil, err
	}

	jsonAction, err := json.MarshalIndent(action, "", "  ")
	if err != nil {
		return nil, err
	}

	return mcp.NewToolResultText(string(jsonAction)), nil
}

// ShutdownDroplet shuts down a droplet
func (d *DropletTool) ShutdownDroplet(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	dropletIDFloat, err := req.RequireFloat("ID")
	if err != nil {
		return nil, err
	}
	dropletID := int(dropletIDFloat)
	
	action, _, err := d.client.DropletActions.Shutdown(ctx, dropletID)
	if err != nil {
		return nil, err
	}

	jsonAction, err := json.MarshalIndent(action, "", "  ")
	if err != nil {
		return nil, err
	}

	return mcp.NewToolResultText(string(jsonAction)), nil
}

// RestoreDroplet restores a droplet to a backup image
func (d *DropletTool) RestoreDroplet(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	dropletIDFloat, err := req.RequireFloat("ID")
	if err != nil {
		return nil, err
	}
	dropletID := int(dropletIDFloat)
	
	imageIDFloat, err := req.RequireFloat("ImageID")
	if err != nil {
		return nil, err
	}
	imageID := int(imageIDFloat)
	
	action, _, err := d.client.DropletActions.Restore(ctx, dropletID, imageID)
	if err != nil {
		return nil, err
	}

	jsonAction, err := json.MarshalIndent(action, "", "  ")
	if err != nil {
		return nil, err
	}

	return mcp.NewToolResultText(string(jsonAction)), nil
}

// ResizeDroplet resizes a droplet
func (d *DropletTool) ResizeDroplet(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	dropletIDFloat, err := req.RequireFloat("ID")
	if err != nil {
		return nil, err
	}
	dropletID := int(dropletIDFloat)
	
	size, err := req.RequireString("Size")
	if err != nil {
		return nil, err
	}
	
	resizeDisk := req.GetBool("ResizeDisk", false)
	
	action, _, err := d.client.DropletActions.Resize(ctx, dropletID, size, resizeDisk)
	if err != nil {
		return nil, err
	}

	jsonAction, err := json.MarshalIndent(action, "", "  ")
	if err != nil {
		return nil, err
	}

	return mcp.NewToolResultText(string(jsonAction)), nil
}

// RebuildDroplet rebuilds a droplet using a provided image
func (d *DropletTool) RebuildDroplet(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	dropletIDFloat, err := req.RequireFloat("ID")
	if err != nil {
		return nil, err
	}
	dropletID := int(dropletIDFloat)
	
	imageIDFloat, err := req.RequireFloat("ImageID") 
	if err != nil {
		return nil, err
	}
	imageID := int(imageIDFloat)
	
	action, _, err := d.client.DropletActions.RebuildByImageID(ctx, dropletID, imageID)
	if err != nil {
		return nil, err
	}

	jsonAction, err := json.MarshalIndent(action, "", "  ")
	if err != nil {
		return nil, err
	}

	return mcp.NewToolResultText(string(jsonAction)), nil
}

// RenameDroplet renames a droplet
func (d *DropletTool) RenameDroplet(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	dropletIDFloat, err := req.RequireFloat("ID")
	if err != nil {
		return nil, err
	}
	dropletID := int(dropletIDFloat)
	
	name, err := req.RequireString("Name")
	if err != nil {
		return nil, err
	}
	
	action, _, err := d.client.DropletActions.Rename(ctx, dropletID, name)
	if err != nil {
		return nil, err
	}

	jsonAction, err := json.MarshalIndent(action, "", "  ")
	if err != nil {
		return nil, err
	}

	return mcp.NewToolResultText(string(jsonAction)), nil
}

// ChangeKernel changes a droplet's kernel
func (d *DropletTool) ChangeKernel(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	dropletIDFloat, err := req.RequireFloat("ID")
	if err != nil {
		return nil, err
	}
	dropletID := int(dropletIDFloat)
	
	kernelIDFloat, err := req.RequireFloat("KernelID")
	if err != nil {
		return nil, err
	}
	kernelID := int(kernelIDFloat)
	
	action, _, err := d.client.DropletActions.ChangeKernel(ctx, dropletID, kernelID)
	if err != nil {
		return nil, err
	}

	jsonAction, err := json.MarshalIndent(action, "", "  ")
	if err != nil {
		return nil, err
	}

	return mcp.NewToolResultText(string(jsonAction)), nil
}

// EnableIPv6 enables IPv6 on a droplet
func (d *DropletTool) EnableIPv6(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	dropletIDFloat, err := req.RequireFloat("ID")
	if err != nil {
		return nil, err
	}
	dropletID := int(dropletIDFloat)
	
	action, _, err := d.client.DropletActions.EnableIPv6(ctx, dropletID)
	if err != nil {
		return nil, err
	}

	jsonAction, err := json.MarshalIndent(action, "", "  ")
	if err != nil {
		return nil, err
	}

	return mcp.NewToolResultText(string(jsonAction)), nil
}

// EnableBackups enables backups on a droplet
func (d *DropletTool) EnableBackups(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	dropletIDFloat, err := req.RequireFloat("ID")
	if err != nil {
		return nil, err
	}
	dropletID := int(dropletIDFloat)
	
	action, _, err := d.client.DropletActions.EnableBackups(ctx, dropletID)
	if err != nil {
		return nil, err
	}

	jsonAction, err := json.MarshalIndent(action, "", "  ")
	if err != nil {
		return nil, err
	}

	return mcp.NewToolResultText(string(jsonAction)), nil
}

// DisableBackups disables backups on a droplet
func (d *DropletTool) DisableBackups(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	dropletIDFloat, err := req.RequireFloat("ID")
	if err != nil {
		return nil, err
	}
	dropletID := int(dropletIDFloat)
	
	action, _, err := d.client.DropletActions.DisableBackups(ctx, dropletID)
	if err != nil {
		return nil, err
	}

	jsonAction, err := json.MarshalIndent(action, "", "  ")
	if err != nil {
		return nil, err
	}

	return mcp.NewToolResultText(string(jsonAction)), nil
}

// SnapshotDroplet creates a snapshot of a droplet
func (d *DropletTool) SnapshotDroplet(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	dropletIDFloat, err := req.RequireFloat("ID")
	if err != nil {
		return nil, err
	}
	dropletID := int(dropletIDFloat)
	
	name, err := req.RequireString("Name")
	if err != nil {
		return nil, err
	}
	
	action, _, err := d.client.DropletActions.Snapshot(ctx, dropletID, name)
	if err != nil {
		return nil, err
	}

	jsonAction, err := json.MarshalIndent(action, "", "  ")
	if err != nil {
		return nil, err
	}

	return mcp.NewToolResultText(string(jsonAction)), nil
}

// GetDropletNeighbors gets a droplet's neighbors
func (d *DropletTool) GetDropletNeighbors(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	dropletIDFloat, err := req.RequireFloat("ID")
	if err != nil {
		return nil, err
	}
	dropletID := int(dropletIDFloat)
	
	neighbors, _, err := d.client.Droplets.Neighbors(ctx, dropletID)
	if err != nil {
		return nil, err
	}

	jsonNeighbors, err := json.MarshalIndent(neighbors, "", "  ")
	if err != nil {
		return nil, err
	}

	return mcp.NewToolResultText(string(jsonNeighbors)), nil
}

// EnablePrivateNetworking enables private networking on a droplet
func (d *DropletTool) EnablePrivateNetworking(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	dropletIDFloat, err := req.RequireFloat("ID")
	if err != nil {
		return nil, err
	}
	dropletID := int(dropletIDFloat)
	
	action, _, err := d.client.DropletActions.EnablePrivateNetworking(ctx, dropletID)
	if err != nil {
		return nil, err
	}

	jsonAction, err := json.MarshalIndent(action, "", "  ")
	if err != nil {
		return nil, err
	}

	return mcp.NewToolResultText(string(jsonAction)), nil
}

// GetDropletKernels gets available kernels for a droplet
func (d *DropletTool) GetDropletKernels(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	dropletIDFloat, err := req.RequireFloat("ID")
	if err != nil {
		return nil, err
	}
	dropletID := int(dropletIDFloat)

	// Use list options to get all kernels
	opt := &godo.ListOptions{
		Page:    1,
		PerPage: 100,
	}

	kernels, _, err := d.client.Droplets.Kernels(ctx, dropletID, opt)
	if err != nil {
		return nil, err
	}

	jsonKernels, err := json.MarshalIndent(kernels, "", "  ")
	if err != nil {
		return nil, err
	}

	return mcp.NewToolResultText(string(jsonKernels)), nil
}

// RebootDroplet reboots a droplet
func (d *DropletTool) RebootDroplet(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	dropletIDFloat, err := req.RequireFloat("ID")
	if err != nil {
		return nil, err
	}
	dropletID := int(dropletIDFloat)
	
	action, _, err := d.client.DropletActions.Reboot(ctx, dropletID)
	if err != nil {
		return nil, err
	}

	jsonAction, err := json.MarshalIndent(action, "", "  ")
	if err != nil {
		return nil, err
	}

	return mcp.NewToolResultText(string(jsonAction)), nil
}

// PasswordResetDroplet resets the password for a droplet
func (d *DropletTool) PasswordResetDroplet(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	dropletIDFloat, err := req.RequireFloat("ID")
	if err != nil {
		return nil, err
	}
	dropletID := int(dropletIDFloat)
	
	action, _, err := d.client.DropletActions.PasswordReset(ctx, dropletID)
	if err != nil {
		return nil, err
	}

	jsonAction, err := json.MarshalIndent(action, "", "  ")
	if err != nil {
		return nil, err
	}

	return mcp.NewToolResultText(string(jsonAction)), nil
}

// RebuildByImageSlugDroplet rebuilds a droplet using an image slug
func (d *DropletTool) RebuildByImageSlugDroplet(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	dropletIDFloat, err := req.RequireFloat("ID")
	if err != nil {
		return nil, err
	}
	dropletID := int(dropletIDFloat)
	
	imageSlug, err := req.RequireString("ImageSlug")
	if err != nil {
		return nil, err
	}
	
	action, _, err := d.client.DropletActions.RebuildByImageSlug(ctx, dropletID, imageSlug)
	if err != nil {
		return nil, err
	}

	jsonAction, err := json.MarshalIndent(action, "", "  ")
	if err != nil {
		return nil, err
	}

	return mcp.NewToolResultText(string(jsonAction)), nil
}

// PowerCycleByTag power cycles droplets by tag
func (d *DropletTool) PowerCycleByTag(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	tag, err := req.RequireString("Tag")
	if err != nil {
		return nil, err
	}
	
	actions, _, err := d.client.DropletActions.PowerCycleByTag(ctx, tag)
	if err != nil {
		return nil, err
	}

	jsonActions, err := json.MarshalIndent(actions, "", "  ")
	if err != nil {
		return nil, err
	}

	return mcp.NewToolResultText(string(jsonActions)), nil
}

// PowerOnByTag powers on droplets by tag
func (d *DropletTool) PowerOnByTag(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	tag, err := req.RequireString("Tag")
	if err != nil {
		return nil, err
	}
	
	actions, _, err := d.client.DropletActions.PowerOnByTag(ctx, tag)
	if err != nil {
		return nil, err
	}

	jsonActions, err := json.MarshalIndent(actions, "", "  ")
	if err != nil {
		return nil, err
	}

	return mcp.NewToolResultText(string(jsonActions)), nil
}

// PowerOffByTag powers off droplets by tag
func (d *DropletTool) PowerOffByTag(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	tag, err := req.RequireString("Tag")
	if err != nil {
		return nil, err
	}
	
	actions, _, err := d.client.DropletActions.PowerOffByTag(ctx, tag)
	if err != nil {
		return nil, err
	}

	jsonActions, err := json.MarshalIndent(actions, "", "  ")
	if err != nil {
		return nil, err
	}

	return mcp.NewToolResultText(string(jsonActions)), nil
}

// ShutdownByTag shuts down droplets by tag
func (d *DropletTool) ShutdownByTag(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	tag, err := req.RequireString("Tag")
	if err != nil {
		return nil, err
	}
	
	actions, _, err := d.client.DropletActions.ShutdownByTag(ctx, tag)
	if err != nil {
		return nil, err
	}

	jsonActions, err := json.MarshalIndent(actions, "", "  ")
	if err != nil {
		return nil, err
	}

	return mcp.NewToolResultText(string(jsonActions)), nil
}

// EnableBackupsByTag enables backups on droplets by tag
func (d *DropletTool) EnableBackupsByTag(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	tag, err := req.RequireString("Tag")
	if err != nil {
		return nil, err
	}
	
	actions, _, err := d.client.DropletActions.EnableBackupsByTag(ctx, tag)
	if err != nil {
		return nil, err
	}

	jsonActions, err := json.MarshalIndent(actions, "", "  ")
	if err != nil {
		return nil, err
	}

	return mcp.NewToolResultText(string(jsonActions)), nil
}

// DisableBackupsByTag disables backups on droplets by tag
func (d *DropletTool) DisableBackupsByTag(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	tag, err := req.RequireString("Tag")
	if err != nil {
		return nil, err
	}
	
	actions, _, err := d.client.DropletActions.DisableBackupsByTag(ctx, tag)
	if err != nil {
		return nil, err
	}

	jsonActions, err := json.MarshalIndent(actions, "", "  ")
	if err != nil {
		return nil, err
	}

	return mcp.NewToolResultText(string(jsonActions)), nil
}

// SnapshotByTag takes a snapshot of droplets by tag
func (d *DropletTool) SnapshotByTag(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	tag, err := req.RequireString("Tag")
	if err != nil {
		return nil, err
	}
	
	name, err := req.RequireString("Name")
	if err != nil {
		return nil, err
	}
	
	actions, _, err := d.client.DropletActions.SnapshotByTag(ctx, tag, name)
	if err != nil {
		return nil, err
	}

	jsonActions, err := json.MarshalIndent(actions, "", "  ")
	if err != nil {
		return nil, err
	}

	return mcp.NewToolResultText(string(jsonActions)), nil
}

// EnableIPv6ByTag enables IPv6 on droplets by tag
func (d *DropletTool) EnableIPv6ByTag(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	tag, err := req.RequireString("Tag")
	if err != nil {
		return nil, err
	}
	
	actions, _, err := d.client.DropletActions.EnableIPv6ByTag(ctx, tag)
	if err != nil {
		return nil, err
	}

	jsonActions, err := json.MarshalIndent(actions, "", "  ")
	if err != nil {
		return nil, err
	}

	return mcp.NewToolResultText(string(jsonActions)), nil
}

// EnablePrivateNetworkingByTag enables private networking on droplets by tag
func (d *DropletTool) EnablePrivateNetworkingByTag(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	tag, err := req.RequireString("Tag")
	if err != nil {
		return nil, err
	}
	
	actions, _, err := d.client.DropletActions.EnablePrivateNetworkingByTag(ctx, tag)
	if err != nil {
		return nil, err
	}

	jsonActions, err := json.MarshalIndent(actions, "", "  ")
	if err != nil {
		return nil, err
	}

	return mcp.NewToolResultText(string(jsonActions)), nil
}

// Tools returns a list of tool functions
func (d *DropletTool) Tools() []server.ServerTool {
	return []server.ServerTool{
		{
			Handler: d.CreateDroplet,
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
			Handler: d.DeleteDroplet,
			Tool: mcp.NewTool("digitalocean-droplet-delete",
				mcp.WithDescription("Delete a droplet"),
				mcp.WithNumber("ID", mcp.Required(), mcp.Description("ID of the droplet to delete")),
			),
		},
		{
			Handler: d.PowerCycleDroplet,
			Tool: mcp.NewTool("digitalocean-droplet-power-cycle",
				mcp.WithDescription("Power cycle a droplet"),
				mcp.WithNumber("ID", mcp.Required(), mcp.Description("ID of the droplet to power cycle")),
			),
		},
		{
			Handler: d.PowerOnDroplet,
			Tool: mcp.NewTool("digitalocean-droplet-power-on",
				mcp.WithDescription("Power on a droplet"),
				mcp.WithNumber("ID", mcp.Required(), mcp.Description("ID of the droplet to power on")),
			),
		},
		{
			Handler: d.PowerOffDroplet,
			Tool: mcp.NewTool("digitalocean-droplet-power-off",
				mcp.WithDescription("Power off a droplet"),
				mcp.WithNumber("ID", mcp.Required(), mcp.Description("ID of the droplet to power off")),
			),
		},
		{
			Handler: d.ShutdownDroplet,
			Tool: mcp.NewTool("digitalocean-droplet-shutdown",
				mcp.WithDescription("Shutdown a droplet"),
				mcp.WithNumber("ID", mcp.Required(), mcp.Description("ID of the droplet to shutdown")),
			),
		},
		{
			Handler: d.RestoreDroplet,
			Tool: mcp.NewTool("digitalocean-droplet-restore",
				mcp.WithDescription("Restore a droplet from a backup/snapshot"),
				mcp.WithNumber("ID", mcp.Required(), mcp.Description("ID of the droplet to restore")),
				mcp.WithNumber("ImageID", mcp.Required(), mcp.Description("ID of the backup/snapshot image")),
			),
		},
		{
			Handler: d.ResizeDroplet,
			Tool: mcp.NewTool("digitalocean-droplet-resize",
				mcp.WithDescription("Resize a droplet"),
				mcp.WithNumber("ID", mcp.Required(), mcp.Description("ID of the droplet to resize")),
				mcp.WithString("Size", mcp.Required(), mcp.Description("Slug of the new size (e.g., s-1vcpu-1gb)")),
				mcp.WithBoolean("ResizeDisk", mcp.DefaultBool(false), mcp.Description("Whether to resize the disk")),
			),
		},
		{
			Handler: d.RebuildDroplet,
			Tool: mcp.NewTool("digitalocean-droplet-rebuild",
				mcp.WithDescription("Rebuild a droplet from an image"),
				mcp.WithNumber("ID", mcp.Required(), mcp.Description("ID of the droplet to rebuild")),
				mcp.WithNumber("ImageID", mcp.Required(), mcp.Description("ID of the image to rebuild from")),
			),
		},
		{
			Handler: d.RenameDroplet,
			Tool: mcp.NewTool("digitalocean-droplet-rename",
				mcp.WithDescription("Rename a droplet"),
				mcp.WithNumber("ID", mcp.Required(), mcp.Description("ID of the droplet to rename")),
				mcp.WithString("Name", mcp.Required(), mcp.Description("New name for the droplet")),
			),
		},
		{
			Handler: d.ChangeKernel,
			Tool: mcp.NewTool("digitalocean-droplet-change-kernel",
				mcp.WithDescription("Change a droplet's kernel"),
				mcp.WithNumber("ID", mcp.Required(), mcp.Description("ID of the droplet")),
				mcp.WithNumber("KernelID", mcp.Required(), mcp.Description("ID of the kernel to switch to")),
			),
		},
		{
			Handler: d.EnableIPv6,
			Tool: mcp.NewTool("digitalocean-droplet-enable-ipv6",
				mcp.WithDescription("Enable IPv6 on a droplet"),
				mcp.WithNumber("ID", mcp.Required(), mcp.Description("ID of the droplet")),
			),
		},
		{
			Handler: d.EnableBackups,
			Tool: mcp.NewTool("digitalocean-droplet-enable-backups",
				mcp.WithDescription("Enable backups on a droplet"),
				mcp.WithNumber("ID", mcp.Required(), mcp.Description("ID of the droplet")),
			),
		},
		{
			Handler: d.DisableBackups,
			Tool: mcp.NewTool("digitalocean-droplet-disable-backups",
				mcp.WithDescription("Disable backups on a droplet"),
				mcp.WithNumber("ID", mcp.Required(), mcp.Description("ID of the droplet")),
			),
		},
		{
			Handler: d.SnapshotDroplet,
			Tool: mcp.NewTool("digitalocean-droplet-snapshot",
				mcp.WithDescription("Take a snapshot of a droplet"),
				mcp.WithNumber("ID", mcp.Required(), mcp.Description("ID of the droplet")),
				mcp.WithString("Name", mcp.Required(), mcp.Description("Name for the snapshot")),
			),
		},
		{
			Handler: d.GetDropletNeighbors,
			Tool: mcp.NewTool("digitalocean-droplet-get-neighbors",
				mcp.WithDescription("Get neighbors of a droplet"),
				mcp.WithNumber("ID", mcp.Required(), mcp.Description("ID of the droplet")),
			),
		},
		{
			Handler: d.EnablePrivateNetworking,
			Tool: mcp.NewTool("digitalocean-droplet-enable-private-net",
				mcp.WithDescription("Enable private networking on a droplet"),
				mcp.WithNumber("ID", mcp.Required(), mcp.Description("ID of the droplet")),
			),
		},
		{
			Handler: d.GetDropletKernels,
			Tool: mcp.NewTool("digitalocean-droplet-get-kernels",
				mcp.WithDescription("Get available kernels for a droplet"),
				mcp.WithNumber("ID", mcp.Required(), mcp.Description("ID of the droplet")),
			),
		},
		{
			Handler: d.RebootDroplet,
			Tool: mcp.NewTool("digitalocean-droplet-reboot",
				mcp.WithDescription("Reboot a droplet"),
				mcp.WithNumber("ID", mcp.Required(), mcp.Description("ID of the droplet to reboot")),
			),
		},
		{
			Handler: d.PasswordResetDroplet,
			Tool: mcp.NewTool("digitalocean-droplet-password-reset",
				mcp.WithDescription("Reset password for a droplet"),
				mcp.WithNumber("ID", mcp.Required(), mcp.Description("ID of the droplet")),
			),
		},
		{
			Handler: d.RebuildByImageSlugDroplet,
			Tool: mcp.NewTool("digitalocean-droplet-rebuild-by-slug",
				mcp.WithDescription("Rebuild a droplet using an image slug"),
				mcp.WithNumber("ID", mcp.Required(), mcp.Description("ID of the droplet to rebuild")),
				mcp.WithString("ImageSlug", mcp.Required(), mcp.Description("Slug of the image to rebuild from")),
			),
		},
		{
			Handler: d.PowerCycleByTag,
			Tool: mcp.NewTool("digitalocean-droplet-power-cycle-by-tag",
				mcp.WithDescription("Power cycle droplets by tag"),
				mcp.WithString("Tag", mcp.Required(), mcp.Description("Tag of the droplets to power cycle")),
			),
		},
		{
			Handler: d.PowerOnByTag,
			Tool: mcp.NewTool("digitalocean-droplet-power-on-by-tag",
				mcp.WithDescription("Power on droplets by tag"),
				mcp.WithString("Tag", mcp.Required(), mcp.Description("Tag of the droplets to power on")),
			),
		},
		{
			Handler: d.PowerOffByTag,
			Tool: mcp.NewTool("digitalocean-droplet-power-off-by-tag",
				mcp.WithDescription("Power off droplets by tag"),
				mcp.WithString("Tag", mcp.Required(), mcp.Description("Tag of the droplets to power off")),
			),
		},
		{
			Handler: d.ShutdownByTag,
			Tool: mcp.NewTool("digitalocean-droplet-shutdown-by-tag",
				mcp.WithDescription("Shutdown droplets by tag"),
				mcp.WithString("Tag", mcp.Required(), mcp.Description("Tag of the droplets to shutdown")),
			),
		},
		{
			Handler: d.EnableBackupsByTag,
			Tool: mcp.NewTool("digitalocean-droplet-enable-backups-by-tag",
				mcp.WithDescription("Enable backups on droplets by tag"),
				mcp.WithString("Tag", mcp.Required(), mcp.Description("Tag of the droplets")),
			),
		},
		{
			Handler: d.DisableBackupsByTag,
			Tool: mcp.NewTool("digitalocean-droplet-disable-backups-by-tag",
				mcp.WithDescription("Disable backups on droplets by tag"),
				mcp.WithString("Tag", mcp.Required(), mcp.Description("Tag of the droplets")),
			),
		},
		{
			Handler: d.SnapshotByTag,
			Tool: mcp.NewTool("digitalocean-droplet-snapshot-by-tag",
				mcp.WithDescription("Take a snapshot of droplets by tag"),
				mcp.WithString("Tag", mcp.Required(), mcp.Description("Tag of the droplets")),
				mcp.WithString("Name", mcp.Required(), mcp.Description("Name for the snapshot")),
			),
		},
		{
			Handler: d.EnableIPv6ByTag,
			Tool: mcp.NewTool("digitalocean-droplet-enable-ipv6-by-tag",
				mcp.WithDescription("Enable IPv6 on droplets by tag"),
				mcp.WithString("Tag", mcp.Required(), mcp.Description("Tag of the droplets")),
			),
		},
		{
			Handler: d.EnablePrivateNetworkingByTag,
			Tool: mcp.NewTool("digitalocean-droplet-enable-private-net-by-tag",
				mcp.WithDescription("Enable private networking on droplets by tag"),
				mcp.WithString("Tag", mcp.Required(), mcp.Description("Tag of the droplets")),
			),
		},
	}
}
