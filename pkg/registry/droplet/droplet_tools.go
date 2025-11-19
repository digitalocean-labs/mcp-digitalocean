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
	client func(ctx context.Context) (*godo.Client, error)
}

// NewDropletTool creates a new droplet tool
func NewDropletTool(client func(ctx context.Context) (*godo.Client, error)) *DropletTool {
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

	// Handle SSH keys if provided
	var sshKeys []godo.DropletCreateSSHKey
	if sshKeysRaw, ok := args["SSHKeys"]; ok && sshKeysRaw != nil {
		sshKeysList := sshKeysRaw.([]interface{})
		for _, key := range sshKeysList {
			switch v := key.(type) {
			case float64:
				sshKeys = append(sshKeys, godo.DropletCreateSSHKey{ID: int(v)})
			case string:
				sshKeys = append(sshKeys, godo.DropletCreateSSHKey{Fingerprint: v})
			}
		}
	}

	// Handle tags if provided
	var tags []string
	if tagsRaw, ok := args["Tags"]; ok && tagsRaw != nil {
		tagsList := tagsRaw.([]interface{})
		for _, tag := range tagsList {
			if tagStr, ok := tag.(string); ok {
				tags = append(tags, tagStr)
			}
		}
	}

	// Create the droplet
	dropletCreateRequest := &godo.DropletCreateRequest{
		Name:       dropletName,
		Size:       size,
		Image:      godo.DropletCreateImage{ID: int(imageID)},
		Region:     region,
		Backups:    backup,
		Monitoring: monitoring,
		SSHKeys:    sshKeys,
		Tags:       tags,
	}

	client, err := d.client(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to get DigitalOcean client: %w", err)
	}

	droplet, _, err := client.Droplets.Create(ctx, dropletCreateRequest)
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

	client, err := d.client(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to get DigitalOcean client: %w", err)
	}

	_, err = client.Droplets.Delete(ctx, int(dropletID))
	if err != nil {
		return mcp.NewToolResultErrorFromErr("api error", err), nil
	}
	return mcp.NewToolResultText("Droplet deleted successfully"), nil
}

// getDropletNeighbors gets a droplet's neighbors
func (d *DropletTool) getDropletNeighbors(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	dropletID := req.GetArguments()["ID"].(float64)

	client, err := d.client(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to get DigitalOcean client: %w", err)
	}

	neighbors, _, err := client.Droplets.Neighbors(ctx, int(dropletID))
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

	client, err := d.client(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to get DigitalOcean client: %w", err)
	}

	action, _, err := client.DropletActions.EnablePrivateNetworking(ctx, int(dropletID))
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

	client, err := d.client(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to get DigitalOcean client: %w", err)
	}

	kernels, _, err := client.Droplets.Kernels(ctx, int(dropletID), opt)
	if err != nil {
		return mcp.NewToolResultErrorFromErr("api error", err), nil
	}

	jsonKernels, err := json.MarshalIndent(kernels, "", "  ")
	if err != nil {
		return nil, fmt.Errorf("marshal error: %w", err)
	}

	return mcp.NewToolResultText(string(jsonKernels)), nil
}

// Tools returns a list of tool functions
func (d *DropletTool) getDropletByID(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	id, ok := req.GetArguments()["ID"].(float64)
	if !ok {
		return mcp.NewToolResultError("Droplet ID is required"), nil
	}

	client, err := d.client(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to get DigitalOcean client: %w", err)
	}

	droplet, _, err := client.Droplets.Get(ctx, int(id))
	if err != nil {
		return mcp.NewToolResultErrorFromErr("api error", err), nil
	}
	jsonData, err := json.MarshalIndent(droplet, "", "  ")
	if err != nil {
		return nil, fmt.Errorf("marshal error: %w", err)
	}
	return mcp.NewToolResultText(string(jsonData)), nil
}

func (d *DropletTool) getDropletActionByID(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	dropletID, ok := req.GetArguments()["DropletID"].(float64)
	if !ok {
		return mcp.NewToolResultError("DropletID is required"), nil
	}
	actionID, ok := req.GetArguments()["ActionID"].(float64)
	if !ok {
		return mcp.NewToolResultError("ActionID is required"), nil
	}

	client, err := d.client(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to get DigitalOcean client: %w", err)
	}

	action, _, err := client.DropletActions.Get(ctx, int(dropletID), int(actionID))
	if err != nil {
		return mcp.NewToolResultErrorFromErr("api error", err), nil
	}
	jsonData, err := json.MarshalIndent(action, "", "  ")
	if err != nil {
		return nil, fmt.Errorf("marshal error: %w", err)
	}
	return mcp.NewToolResultText(string(jsonData)), nil
}

// getDroplets lists all droplets for a user
func (d *DropletTool) getDroplets(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	page, ok := req.GetArguments()["Page"].(float64)
	if !ok {
		page = 1
	}
	perPage, ok := req.GetArguments()["PerPage"].(float64)
	if !ok {
		perPage = 50
	}

	opt := &godo.ListOptions{
		Page:    int(page),
		PerPage: int(perPage),
	}

	client, err := d.client(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to get DigitalOcean client: %w", err)
	}

	droplets, _, err := client.Droplets.List(ctx, opt)
	if err != nil {
		return mcp.NewToolResultErrorFromErr("api error", err), nil
	}

	filteredDroplets := make([]map[string]any, len(droplets))
	for i, droplet := range droplets {
		filteredDroplets[i] = map[string]any{
			"id":                 droplet.ID,
			"name":               droplet.Name,
			"memory":             droplet.Memory,
			"vcpus":              droplet.Vcpus,
			"disk":               droplet.Disk,
			"region":             droplet.Region,
			"image":              droplet.Image,
			"size":               droplet.Size,
			"size_slug":          droplet.SizeSlug,
			"backup_ids":         droplet.BackupIDs,
			"next_backup_window": droplet.NextBackupWindow,
			"snapshot_ids":       droplet.SnapshotIDs,
			"features":           droplet.Features,
			"locked":             droplet.Locked,
			"status":             droplet.Status,
			"networks":           droplet.Networks,
			"created_at":         droplet.Created,
			"kernel":             droplet.Kernel,
			"tags":               droplet.Tags,
			"volume_ids":         droplet.VolumeIDs,
			"vpc_uuid":           droplet.VPCUUID,
		}
	}

	jsonData, err := json.MarshalIndent(filteredDroplets, "", "  ")
	if err != nil {
		return nil, fmt.Errorf("marshal error: %w", err)
	}

	return mcp.NewToolResultText(string(jsonData)), nil
}

func (d *DropletTool) Tools() []server.ServerTool {
	tools := []server.ServerTool{
		{
			Handler: d.createDroplet,
			Tool: mcp.NewTool("droplet-create",
				mcp.WithDescription("Create a new droplet"),
				mcp.WithString("Name", mcp.Required(), mcp.Description("Name of the droplet")),
				mcp.WithString("Size", mcp.Required(), mcp.Description("Slug of the droplet size (e.g., s-1vcpu-1gb)")),
				mcp.WithNumber("ImageID", mcp.Required(), mcp.Description("ID of the image to use")),
				mcp.WithString("Region", mcp.Required(), mcp.Description("Slug of the region (e.g., nyc3)")),
				mcp.WithBoolean("Backup", mcp.DefaultBool(false), mcp.Description("Whether to enable backups")),
				mcp.WithBoolean("Monitoring", mcp.DefaultBool(false), mcp.Description("Whether to enable monitoring")),
				mcp.WithArray("SSHKeys", mcp.Description("Array of SSH key IDs (numbers) or fingerprints (strings) to add to the droplet")),
				mcp.WithArray("Tags", mcp.Description("Array of tag names to apply to the droplet")),
			),
		},
		{
			Handler: d.deleteDroplet,
			Tool: mcp.NewTool("droplet-delete",
				mcp.WithDescription("Delete a droplet"),
				mcp.WithNumber("ID", mcp.Required(), mcp.Description("ID of the droplet to delete")),
			),
		},
		{
			Handler: d.enablePrivateNetworking,
			Tool: mcp.NewTool("droplet-enable-private-net",
				mcp.WithDescription("Enable private networking on a droplet"),
				mcp.WithNumber("ID", mcp.Required(), mcp.Description("ID of the droplet")),
			),
		},
		{
			Handler: d.getDropletKernels,
			Tool: mcp.NewTool("droplet-kernels",
				mcp.WithDescription("Get available kernels for a droplet"),
				mcp.WithNumber("ID", mcp.Required(), mcp.Description("ID of the droplet")),
			),
		},
		{
			Handler: d.getDropletByID,
			Tool: mcp.NewTool("droplet-get",
				mcp.WithDescription("Get a droplet by its ID"),
				mcp.WithNumber("ID", mcp.Required(), mcp.Description("Droplet ID")),
			),
		},
		{
			Handler: d.getDropletActionByID,
			Tool: mcp.NewTool("droplet-action",
				mcp.WithDescription("Get a droplet action by droplet ID and action ID"),
				mcp.WithNumber("DropletID", mcp.Required(), mcp.Description("Droplet ID")),
				mcp.WithNumber("ActionID", mcp.Required(), mcp.Description("Action ID")),
			),
		},
		{
			Handler: d.getDroplets,
			Tool: mcp.NewTool("droplet-list",
				mcp.WithDescription("List all droplets for the user. Supports pagination."),
				mcp.WithNumber("Page", mcp.DefaultNumber(1), mcp.Description("Page number")),
				mcp.WithNumber("PerPage", mcp.DefaultNumber(50), mcp.Description("Items per page")),
			),
		},
	}
	return tools
}
