package nfs

import (
	"context"

	"github.com/digitalocean/godo"
	"github.com/mark3labs/mcp-go/mcp"
	"github.com/mark3labs/mcp-go/server"

	"mcp-digitalocean/pkg/registry/common"
)

const (
	defaultFileShareListPage    = 1
	defaultFileShareListPerPage = 20
	maxFileShareListPerPage     = 50
)

type NfsTool struct {
	client func(ctx context.Context) (*godo.Client, error)
}

// NewNfsTool creates a new NfsTool instance
func NewNfsTool(client func(ctx context.Context) (*godo.Client, error)) *NfsTool {
	return &NfsTool{client: client}
}

func (n *NfsTool) createFileShare(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	args := req.GetArguments()
	// required arguments
	name, ok := args["Name"].(string)
	if !ok || name == "" {
		return mcp.NewToolResultError("Name is required"), nil
	}
	sizeGibibytes, ok := args["SizeGibibytes"].(float64)
	if !ok || sizeGibibytes < 50 {
		return mcp.NewToolResultError("SizeGibibytes is required and must be at least 50 GiB"), nil
	}
	region, ok := args["Region"].(string)
	if !ok || region == "" {
		return mcp.NewToolResultError("Region is required"), nil
	}

	vpcIdsArg, _ := args["VpcIds"].([]any)

	var vpcIds []string
	for _, vpcId := range vpcIdsArg {
		if s, ok := vpcId.(string); ok {
			vpcIds = append(vpcIds, s)
		}
	}

	// optional arguments
	performanceTier, _ := args["PerformanceTier"].(string)

	fileShareCreateRequest := &godo.NfsCreateRequest{
		Name:            name,
		SizeGib:         int(sizeGibibytes),
		Region:          region,
		VpcIDs:          vpcIds,
		PerformanceTier: performanceTier,
	}

	client, err := n.client(ctx)
	if err != nil {
		return mcp.NewToolResultErrorFromErr("Error getting DigitalOcean client", err), nil
	}

	fileShare, _, err := client.Nfs.Create(ctx, fileShareCreateRequest)
	if err != nil {
		return mcp.NewToolResultErrorFromErr("api error", err), nil
	}

	return shareOut.Result(fileShare)
}

func (n *NfsTool) listFileShares(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	args := req.GetArguments()

	region, _ := args["Region"].(string)

	page, ok := args["Page"].(float64)
	if !ok || page < 1 {
		page = defaultFileShareListPage
	}
	perPage, ok := args["PerPage"].(float64)
	if !ok || perPage < 1 {
		perPage = defaultFileShareListPerPage
	}
	if perPage > maxFileShareListPerPage {
		perPage = maxFileShareListPerPage
	}

	listOptions := &godo.ListOptions{
		Page:    int(page),
		PerPage: int(perPage),
	}

	client, err := n.client(ctx)
	if err != nil {
		return mcp.NewToolResultErrorFromErr("Error getting DigitalOcean client", err), nil
	}

	fileShares, _, err := client.Nfs.List(ctx, listOptions, region)
	if err != nil {
		return mcp.NewToolResultErrorFromErr("api error", err), nil
	}

	formattedFileShares := make([]fileShareSummary, len(fileShares))
	for i, fileShare := range fileShares {
		formattedFileShares[i] = newFileShareSummary(fileShare)
	}

	return shareListOut.Result(formattedFileShares)
}

func (n *NfsTool) getFileShareByID(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	args := req.GetArguments()
	id, ok := args["ID"].(string)
	if !ok || id == "" {
		return mcp.NewToolResultError("ID is required"), nil
	}

	client, err := n.client(ctx)
	if err != nil {
		return mcp.NewToolResultErrorFromErr("Error getting DigitalOcean client", err), nil
	}

	fileShare, _, err := client.Nfs.Get(ctx, id, "")
	if err != nil {
		return mcp.NewToolResultErrorFromErr("api error", err), nil
	}

	return shareOut.Result(fileShare)
}

func (n *NfsTool) deleteFileShare(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	args := req.GetArguments()
	id, ok := args["ID"].(string)
	if !ok || id == "" {
		return mcp.NewToolResultError("ID is required"), nil
	}

	client, err := n.client(ctx)
	if err != nil {
		return mcp.NewToolResultErrorFromErr("Error getting DigitalOcean client", err), nil
	}

	_, err = client.Nfs.Delete(ctx, id, "")
	if err != nil {
		return mcp.NewToolResultErrorFromErr("api error", err), nil
	}
	return mcp.NewToolResultText("File share deleted successfully"), nil
}

func (n *NfsTool) listNfsSnapshots(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	args := req.GetArguments()
	region, _ := args["Region"].(string)
	shareID, _ := args["ShareID"].(string)

	page, ok := args["Page"].(float64)
	if !ok || page < 1 {
		page = defaultFileShareListPage
	}
	perPage, ok := args["PerPage"].(float64)
	if !ok || perPage < 1 {
		perPage = defaultFileShareListPerPage
	}
	if perPage > maxFileShareListPerPage {
		perPage = maxFileShareListPerPage
	}
	listOptions := &godo.ListOptions{
		Page:    int(page),
		PerPage: int(perPage),
	}

	client, err := n.client(ctx)
	if err != nil {
		return mcp.NewToolResultErrorFromErr("Error getting DigitalOcean client", err), nil
	}

	snapshots, _, err := client.Nfs.ListSnapshots(ctx, listOptions, shareID, region)
	if err != nil {
		return mcp.NewToolResultErrorFromErr("api error", err), nil
	}

	formattedSnapshots := make([]snapshotSummary, len(snapshots))
	for i, snapshot := range snapshots {
		formattedSnapshots[i] = newSnapshotSummary(snapshot)
	}

	return snapshotListOut.Result(formattedSnapshots)
}

func (n *NfsTool) getNfsSnapshotByID(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	args := req.GetArguments()
	id, ok := args["ID"].(string)
	if !ok || id == "" {
		return mcp.NewToolResultError("ID is required"), nil
	}

	client, err := n.client(ctx)
	if err != nil {
		return mcp.NewToolResultErrorFromErr("Error getting DigitalOcean client", err), nil
	}

	snapshot, _, err := client.Nfs.GetSnapshot(ctx, id, "")
	if err != nil {
		return mcp.NewToolResultErrorFromErr("api error", err), nil
	}

	return snapshotOut.Result(snapshot)
}

func (n *NfsTool) deleteNfsSnapshot(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	args := req.GetArguments()
	id, ok := args["ID"].(string)
	if !ok || id == "" {
		return mcp.NewToolResultError("ID is required"), nil
	}

	client, err := n.client(ctx)
	if err != nil {
		return mcp.NewToolResultErrorFromErr("Error getting DigitalOcean client", err), nil
	}

	_, err = client.Nfs.DeleteSnapshot(ctx, id, "")
	if err != nil {
		return mcp.NewToolResultErrorFromErr("api error", err), nil
	}
	return mcp.NewToolResultText("NFS snapshot deleted successfully"), nil
}

func (n *NfsTool) Tools() []server.ServerTool {
	return []server.ServerTool{
		{
			Handler: n.createFileShare,
			Tool: mcp.NewTool(
				"nfs-file-share-create",
				common.WithHints(common.HintsCreate),
				common.WithRisk(common.RiskMedium),
				shareOut.Schema(),
				mcp.WithDescription("Create a new file share"),
				mcp.WithString("Name", mcp.Required(), mcp.Description("Name of the file share")),
				mcp.WithNumber("SizeGibibytes", mcp.Required(), mcp.Description("Size of the file share in GiB")),
				mcp.WithString("Region", mcp.Required(), mcp.Description("Region of the file share")),
				mcp.WithArray("VpcIds", mcp.Description("VPC IDs of the file share")),
				mcp.WithString("PerformanceTier", mcp.Description("Performance tier of the file share")),
			),
		},
		{
			Handler: n.listFileShares,
			Tool: mcp.NewTool(
				"nfs-file-share-list",
				common.WithHints(common.HintsRead),
				common.WithRisk(common.RiskLow),
				shareListOut.Schema(),
				mcp.WithDescription("List nfs file shares with optional Region filters. Supports pagination."),
				mcp.WithString("Region", mcp.Description("Optional region filtering parameter")),
				mcp.WithNumber("Page", mcp.DefaultNumber(1), mcp.Description("Page number of the results to fetch")),
				mcp.WithNumber("PerPage", mcp.DefaultNumber(20), mcp.Description("Number of items returned per page")),
			),
		},
		{
			Handler: n.getFileShareByID,
			Tool: mcp.NewTool(
				"nfs-file-share-get",
				common.WithHints(common.HintsRead),
				common.WithRisk(common.RiskLow),
				shareOut.Schema(),
				mcp.WithDescription("Get a file share by ID."),
				mcp.WithString("ID", mcp.Required(), mcp.Description("ID of the file share to get")),
			),
		},
		{
			Handler: n.deleteFileShare,
			Tool: mcp.NewTool(
				"nfs-file-share-delete",
				common.WithHints(common.HintsDelete),
				common.WithRisk(common.RiskHigh),
				mcp.WithDescription("Delete a file share by ID."),
				mcp.WithString("ID", mcp.Required(), mcp.Description("ID of the file share to delete")),
			),
		},
		{
			Handler: n.listNfsSnapshots,
			Tool: mcp.NewTool(
				"nfs-snapshot-list",
				common.WithHints(common.HintsRead),
				common.WithRisk(common.RiskLow),
				snapshotListOut.Schema(),
				mcp.WithDescription("List all NFS snapshots - supports pagination and filtering by region and share ID"),
				mcp.WithString("Region", mcp.Description("Optional region of the NFS snapshot")),
				mcp.WithString("ShareID", mcp.Description("Optional ID of the NFS share to list snapshots for")),
				mcp.WithNumber("Page", mcp.DefaultNumber(1), mcp.Description("Page number of the results to fetch")),
				mcp.WithNumber("PerPage", mcp.DefaultNumber(20), mcp.Description("Number of items returned per page")),
			),
		},
		{
			Handler: n.getNfsSnapshotByID,
			Tool: mcp.NewTool(
				"nfs-snapshot-get",
				common.WithHints(common.HintsRead),
				common.WithRisk(common.RiskLow),
				snapshotOut.Schema(),
				mcp.WithDescription("Get a NFS snapshot by ID."),
				mcp.WithString("ID", mcp.Required(), mcp.Description("ID of the NFS snapshot to get")),
			),
		},
		{
			Handler: n.deleteNfsSnapshot,
			Tool: mcp.NewTool(
				"nfs-snapshot-delete",
				common.WithHints(common.HintsDelete),
				common.WithRisk(common.RiskMedium),
				mcp.WithDescription("Delete a NFS snapshot by ID."),
				mcp.WithString("ID", mcp.Required(), mcp.Description("ID of the NFS snapshot to delete")),
			),
		},
	}
}
