package volumes

import (
	"context"
	"encoding/json"

	"github.com/digitalocean/godo"
	"github.com/mark3labs/mcp-go/mcp"
	"github.com/mark3labs/mcp-go/server"

	"mcp-digitalocean/pkg/registry/common"
)

type VolumeTool struct {
	client func(ctx context.Context) (*godo.Client, error)
}

const (
	defaultVolumeListPage    = 1
	defaultVolumeListPerPage = 50
	maxVolumeListPerPage     = 200
)

// NewVolumeTool creates a new VolumeTool instance
func NewVolumeTool(client func(ctx context.Context) (*godo.Client, error)) *VolumeTool {
	return &VolumeTool{client: client}
}

func (vt *VolumeTool) createVolume(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	args := req.GetArguments()
	// required arguments
	name, ok := args["Name"].(string)
	if !ok || name == "" {
		return mcp.NewToolResultError("Name is required"), nil
	}
	sizeGigaBytes, ok := args["SizeGigaBytes"].(float64)
	if !ok || sizeGigaBytes < 1 {
		return mcp.NewToolResultError("SizeGigaBytes is required"), nil
	}
	region, ok := args["Region"].(string)
	if !ok || region == "" {
		return mcp.NewToolResultError("Region is required"), nil
	}

	// optional arguments
	snapshotID, _ := args["SnapshotID"].(string)
	description, _ := args["Description"].(string)
	filesystemType, _ := args["FilesystemType"].(string)
	filesystemLabel, _ := args["FilesystemLabel"].(string)
	tagsArg, _ := args["Tags"].([]any)

	var tags []string
	for _, t := range tagsArg {
		if s, ok := t.(string); ok {
			tags = append(tags, s)
		}
	}

	volumeCreateRequest := &godo.VolumeCreateRequest{
		Name:            name,
		Description:     description,
		SizeGigaBytes:   int64(sizeGigaBytes),
		Region:          region,
		SnapshotID:      snapshotID,
		FilesystemType:  filesystemType,
		FilesystemLabel: filesystemLabel,
		Tags:            tags,
	}

	client, err := vt.client(ctx)
	if err != nil {
		return mcp.NewToolResultErrorFromErr("Error getting DigitalOcean client", err), nil
	}

	volume, _, err := client.Storage.CreateVolume(ctx, volumeCreateRequest)
	if err != nil {
		return mcp.NewToolResultErrorFromErr("api error", err), nil
	}

	jsonVolume, err := json.MarshalIndent(volume, "", "  ")
	if err != nil {
		return mcp.NewToolResultErrorFromErr("marshal error", err), nil
	}
	return mcp.NewToolResultText(string(jsonVolume)), nil
}

func (vt *VolumeTool) listVolumes(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	args := req.GetArguments()

	name, _ := args["Name"].(string)
	region, _ := args["Region"].(string)

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

	listRequest := &godo.ListVolumeParams{
		Name:   name,
		Region: region,
		ListOptions: &godo.ListOptions{
			Page:    int(page),
			PerPage: int(perPage),
		},
	}

	client, err := vt.client(ctx)
	if err != nil {
		return mcp.NewToolResultErrorFromErr("Error getting DigitalOcean client", err), nil
	}

	volumes, _, err := client.Storage.ListVolumes(ctx, listRequest)
	if err != nil {
		return mcp.NewToolResultErrorFromErr("api error", err), nil
	}

	filteredVolumes := make([]map[string]any, len(volumes))
	for i, volume := range volumes {
		filteredVolumes[i] = map[string]any{
			"id":               volume.ID,
			"name":             volume.Name,
			"size_gigabytes":   volume.SizeGigaBytes,
			"region":           volume.Region,
			"description":      volume.Description,
			"filesystem_type":  volume.FilesystemType,
			"filesystem_label": volume.FilesystemLabel,
			"tags":             volume.Tags,
			"created_at":       volume.CreatedAt,
			"droplet_ids":      volume.DropletIDs,
		}
	}

	jsonVolumes, err := json.MarshalIndent(filteredVolumes, "", "  ")
	if err != nil {
		return mcp.NewToolResultErrorFromErr("marshal error", err), nil
	}
	return mcp.NewToolResultText(string(jsonVolumes)), nil
}

func (vt *VolumeTool) getVolumeByID(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	args := req.GetArguments()
	volumeID, ok := args["ID"].(string)
	if !ok || volumeID == "" {
		return mcp.NewToolResultError("Volume ID is required"), nil
	}

	client, err := vt.client(ctx)
	if err != nil {
		return mcp.NewToolResultErrorFromErr("Error getting DigitalOcean client", err), nil
	}

	volume, _, err := client.Storage.GetVolume(ctx, volumeID)
	if err != nil {
		return mcp.NewToolResultErrorFromErr("api error", err), nil
	}

	jsonVolume, err := json.MarshalIndent(volume, "", "  ")
	if err != nil {
		return mcp.NewToolResultErrorFromErr("marshal error", err), nil
	}
	return mcp.NewToolResultText(string(jsonVolume)), nil
}

func (vt *VolumeTool) deleteVolume(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	args := req.GetArguments()
	volumeID, ok := args["ID"].(string)
	if !ok || volumeID == "" {
		return mcp.NewToolResultError("Volume ID is required"), nil
	}

	client, err := vt.client(ctx)
	if err != nil {
		return mcp.NewToolResultErrorFromErr("Error getting DigitalOcean client", err), nil
	}

	_, err = client.Storage.DeleteVolume(ctx, volumeID)
	if err != nil {
		return mcp.NewToolResultErrorFromErr("api error", err), nil
	}
	return mcp.NewToolResultText("Volume deleted successfully"), nil
}

func (vt *VolumeTool) createSnapshot(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	args := req.GetArguments()
	volumeID, ok := args["VolumeID"].(string)
	if !ok || volumeID == "" {
		return mcp.NewToolResultError("Volume ID is required"), nil
	}

	snapshotName, ok := args["Name"].(string)
	if !ok || snapshotName == "" {
		return mcp.NewToolResultError("Snapshot name is required"), nil
	}

	tagsArg, _ := args["Tags"].([]any)

	var tags []string
	for _, t := range tagsArg {
		if s, ok := t.(string); ok {
			tags = append(tags, s)
		}
	}

	request := &godo.SnapshotCreateRequest{
		VolumeID: volumeID,
		Name:     snapshotName,
		Tags:     tags,
	}

	client, err := vt.client(ctx)
	if err != nil {
		return mcp.NewToolResultErrorFromErr("Error getting DigitalOcean client", err), nil
	}

	snapshot, _, err := client.Storage.CreateSnapshot(ctx, request)
	if err != nil {
		return mcp.NewToolResultErrorFromErr("api error", err), nil
	}

	jsonSnapshot, err := json.MarshalIndent(snapshot, "", "  ")
	if err != nil {
		return mcp.NewToolResultErrorFromErr("marshal error", err), nil
	}
	return mcp.NewToolResultText(string(jsonSnapshot)), nil
}

func (vt *VolumeTool) listSnapshots(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
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

	client, err := vt.client(ctx)
	if err != nil {
		return mcp.NewToolResultErrorFromErr("Error getting DigitalOcean client", err), nil
	}

	options := &godo.ListOptions{
		Page:    int(page),
		PerPage: int(perPage),
	}

	snapshots, _, err := client.Storage.ListSnapshots(ctx, volumeID, options)
	if err != nil {
		return mcp.NewToolResultErrorFromErr("api error", err), nil
	}

	filteredSnapshots := make([]map[string]any, len(snapshots))
	for i, snapshot := range snapshots {
		filteredSnapshots[i] = map[string]any{
			"id":             snapshot.ID,
			"name":           snapshot.Name,
			"resource_id":    snapshot.ResourceID,
			"resource_type":  snapshot.ResourceType,
			"regions":        snapshot.Regions,
			"min_disk_size":  snapshot.MinDiskSize,
			"size_gigabytes": snapshot.SizeGigaBytes,
			"tags":           snapshot.Tags,
			"created_at":     snapshot.Created,
		}
	}

	jsonSnapshots, err := json.MarshalIndent(filteredSnapshots, "", "  ")
	if err != nil {
		return mcp.NewToolResultErrorFromErr("marshal error", err), nil
	}
	return mcp.NewToolResultText(string(jsonSnapshots)), nil
}

func (vt *VolumeTool) getSnapshotByID(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	args := req.GetArguments()
	snapshotID, ok := args["ID"].(string)
	if !ok || snapshotID == "" {
		return mcp.NewToolResultError("Snapshot ID is required"), nil
	}

	client, err := vt.client(ctx)
	if err != nil {
		return mcp.NewToolResultErrorFromErr("Error getting DigitalOcean client", err), nil
	}

	snapshot, _, err := client.Storage.GetSnapshot(ctx, snapshotID)
	if err != nil {
		return mcp.NewToolResultErrorFromErr("api error", err), nil
	}

	jsonSnapshot, err := json.MarshalIndent(snapshot, "", "  ")
	if err != nil {
		return mcp.NewToolResultErrorFromErr("marshal error", err), nil
	}
	return mcp.NewToolResultText(string(jsonSnapshot)), nil
}

func (vt *VolumeTool) deleteSnapshot(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	args := req.GetArguments()
	snapshotID, ok := args["ID"].(string)
	if !ok || snapshotID == "" {
		return mcp.NewToolResultError("Snapshot ID is required"), nil
	}

	client, err := vt.client(ctx)
	if err != nil {
		return mcp.NewToolResultErrorFromErr("Error getting DigitalOcean client", err), nil
	}

	_, err = client.Storage.DeleteSnapshot(ctx, snapshotID)
	if err != nil {
		return mcp.NewToolResultErrorFromErr("api error", err), nil
	}
	return mcp.NewToolResultText("Snapshot deleted successfully"), nil
}

// Tools returns MCP server tools for block storage volumes and their snapshots (create, list, get, delete).
func (vt *VolumeTool) Tools() []server.ServerTool {
	tools := []server.ServerTool{
		{
			Handler: vt.createVolume,
			Tool: mcp.NewTool(
				"volume-create",
				common.WithHints(common.HintsCreate),
				common.WithRisk(common.RiskMedium),
				mcp.WithDescription("Create a new block storage volume"),
				mcp.WithString("Name", mcp.Required(), mcp.Description("The name of the volume")),
				mcp.WithNumber("SizeGigaBytes", mcp.Required(), mcp.Description("The size of the volume in GB")),
				mcp.WithString("Region", mcp.Required(), mcp.Description("The region slug where the volume will be created")),
				mcp.WithString("Description", mcp.Description("A human-readable description of the volume (optional)")),
				mcp.WithString("SnapshotID", mcp.Description("The ID of a snapshot to create the volume from (optional)")),
				mcp.WithString("FilesystemType", mcp.Description("The filesystem type for the volume, e.g. ext4 or xfs (optional)")),
				mcp.WithString("FilesystemLabel", mcp.Description("The filesystem label for the volume (optional)")),
				mcp.WithArray("Tags", mcp.Description("Tags to apply"), mcp.Items(map[string]any{"type": "string"})),
			),
		},
		{
			Handler: vt.listVolumes,
			Tool: mcp.NewTool(
				"volume-list",
				common.WithHints(common.HintsRead),
				common.WithRisk(common.RiskLow),
				mcp.WithDescription("List block storage volumes with optional Name/Region filters. Supports pagination."),
				mcp.WithString("Name", mcp.Description("Name filtering parameter")),
				mcp.WithString("Region", mcp.Description("Region filtering parameter")),
				mcp.WithNumber("Page", mcp.DefaultNumber(defaultVolumeListPage), mcp.Description("Page number")),
				mcp.WithNumber("PerPage", mcp.DefaultNumber(defaultVolumeListPerPage), mcp.Description("Volumes per page")),
			),
		},
		{
			Handler: vt.getVolumeByID,
			Tool: mcp.NewTool(
				"volume-get",
				common.WithHints(common.HintsRead),
				common.WithRisk(common.RiskLow),
				mcp.WithDescription("Get a block storage volume by ID"),
				mcp.WithString("ID", mcp.Required(), mcp.Description("The ID of the volume to get")),
			),
		},
		{
			Handler: vt.deleteVolume,
			Tool: mcp.NewTool(
				"volume-delete",
				common.WithHints(common.HintsDelete),
				common.WithRisk(common.RiskHigh),
				mcp.WithDescription("Delete a block storage volume by ID"),
				mcp.WithString("ID", mcp.Required(), mcp.Description("The ID of the volume to delete")),
			),
		},
		{
			Handler: vt.createSnapshot,
			Tool: mcp.NewTool(
				"volume-snapshot-create",
				common.WithHints(common.HintsCreate),
				common.WithRisk(common.RiskLow),
				mcp.WithDescription("Create a new snapshot from a volume"),
				mcp.WithString("VolumeID", mcp.Required(), mcp.Description("The ID of the volume to create a snapshot from")),
				mcp.WithString("Name", mcp.Required(), mcp.Description("The name of the snapshot")),
				mcp.WithArray("Tags", mcp.Description("Tags to apply"), mcp.Items(map[string]any{"type": "string"})),
			),
		},
		{
			Handler: vt.listSnapshots,
			Tool: mcp.NewTool(
				"volume-snapshot-list",
				common.WithHints(common.HintsRead),
				common.WithRisk(common.RiskLow),
				mcp.WithDescription("List snapshots for a volume. Supports pagination."),
				mcp.WithString("VolumeID", mcp.Required(), mcp.Description("The ID of the volume to list snapshots for")),
				mcp.WithNumber("Page", mcp.DefaultNumber(defaultVolumeListPage), mcp.Description("Page number")),
				mcp.WithNumber("PerPage", mcp.DefaultNumber(defaultVolumeListPerPage), mcp.Description("Snapshots per page")),
			),
		},
		{
			Handler: vt.getSnapshotByID,
			Tool: mcp.NewTool(
				"volume-snapshot-get",
				common.WithHints(common.HintsRead),
				common.WithRisk(common.RiskLow),
				mcp.WithDescription("Get a snapshot by ID"),
				mcp.WithString("ID", mcp.Required(), mcp.Description("The ID of the snapshot to get")),
			),
		},
		{
			Handler: vt.deleteSnapshot,
			Tool: mcp.NewTool(
				"volume-snapshot-delete",
				common.WithHints(common.HintsDelete),
				common.WithRisk(common.RiskMedium),
				mcp.WithDescription("Delete a snapshot by ID"),
				mcp.WithString("ID", mcp.Required(), mcp.Description("The ID of the snapshot to delete")),
			),
		},
	}
	return tools
}
