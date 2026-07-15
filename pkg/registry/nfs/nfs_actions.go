package nfs

import (
	"context"
	"encoding/json"

	"github.com/digitalocean/godo"
	"github.com/mark3labs/mcp-go/mcp"
	"github.com/mark3labs/mcp-go/server"

	"mcp-digitalocean/pkg/registry/common"
)

type NfsActionsTool struct {
	client func(ctx context.Context) (*godo.Client, error)
}

// NewNfsActionsTool creates a new NfsActionsTool instance
func NewNfsActionsTool(client func(ctx context.Context) (*godo.Client, error)) *NfsActionsTool {
	return &NfsActionsTool{client: client}
}

func (n *NfsActionsTool) resizeFileShare(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	args := req.GetArguments()
	shareID, ok := args["ShareID"].(string)
	if !ok || shareID == "" {
		return mcp.NewToolResultError("Share ID is required"), nil
	}
	sizeGibibytes, ok := args["SizeGibibytes"].(float64)
	if !ok || sizeGibibytes < 50 {
		return mcp.NewToolResultError("SizeGibibytes is required and must be at least 50 GiB"), nil
	}

	client, err := n.client(ctx)
	if err != nil {
		return mcp.NewToolResultErrorFromErr("Error getting DigitalOcean client", err), nil
	}

	action, _, err := client.NfsActions.Resize(ctx, shareID, uint64(sizeGibibytes), "")
	if err != nil {
		return mcp.NewToolResultErrorFromErr("api error", err), nil
	}

	jsonAction, err := json.MarshalIndent(action, "", "  ")
	if err != nil {
		return mcp.NewToolResultErrorFromErr("marshal error", err), nil
	}
	return mcp.NewToolResultText(string(jsonAction)), nil
}

func (n *NfsActionsTool) snapshotFileShare(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	args := req.GetArguments()
	shareID, ok := args["ShareID"].(string)
	if !ok || shareID == "" {
		return mcp.NewToolResultError("Share ID is required"), nil
	}
	snapshotName, ok := args["SnapshotName"].(string)
	if !ok || snapshotName == "" {
		return mcp.NewToolResultError("Snapshot name is required"), nil
	}

	client, err := n.client(ctx)
	if err != nil {
		return mcp.NewToolResultErrorFromErr("Error getting DigitalOcean client", err), nil
	}

	action, _, err := client.NfsActions.Snapshot(ctx, shareID, snapshotName, "")
	if err != nil {
		return mcp.NewToolResultErrorFromErr("api error", err), nil
	}

	jsonAction, err := json.MarshalIndent(action, "", "  ")
	if err != nil {
		return mcp.NewToolResultErrorFromErr("marshal error", err), nil
	}
	return mcp.NewToolResultText(string(jsonAction)), nil
}

func (n *NfsActionsTool) attachFileShare(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	args := req.GetArguments()
	shareID, ok := args["ShareID"].(string)
	if !ok || shareID == "" {
		return mcp.NewToolResultError("Share ID is required"), nil
	}
	vpcID, ok := args["VpcID"].(string)
	if !ok || vpcID == "" {
		return mcp.NewToolResultError("VPC ID is required"), nil
	}

	client, err := n.client(ctx)
	if err != nil {
		return mcp.NewToolResultErrorFromErr("Error getting DigitalOcean client", err), nil
	}

	action, _, err := client.NfsActions.Attach(ctx, shareID, vpcID, "")
	if err != nil {
		return mcp.NewToolResultErrorFromErr("api error", err), nil
	}

	jsonAction, err := json.MarshalIndent(action, "", "  ")
	if err != nil {
		return mcp.NewToolResultErrorFromErr("marshal error", err), nil
	}
	return mcp.NewToolResultText(string(jsonAction)), nil
}

func (n *NfsActionsTool) detachFileShare(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	args := req.GetArguments()
	shareID, ok := args["ShareID"].(string)
	if !ok || shareID == "" {
		return mcp.NewToolResultError("Share ID is required"), nil
	}
	vpcID, ok := args["VpcID"].(string)
	if !ok || vpcID == "" {
		return mcp.NewToolResultError("VPC ID is required"), nil
	}

	client, err := n.client(ctx)
	if err != nil {
		return mcp.NewToolResultErrorFromErr("Error getting DigitalOcean client", err), nil
	}

	action, _, err := client.NfsActions.Detach(ctx, shareID, vpcID, "")
	if err != nil {
		return mcp.NewToolResultErrorFromErr("api error", err), nil
	}

	jsonAction, err := json.MarshalIndent(action, "", "  ")
	if err != nil {
		return mcp.NewToolResultErrorFromErr("marshal error", err), nil
	}
	return mcp.NewToolResultText(string(jsonAction)), nil
}

func (n *NfsActionsTool) reassignFileShare(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	args := req.GetArguments()
	shareID, ok := args["ShareID"].(string)
	if !ok || shareID == "" {
		return mcp.NewToolResultError("Share ID is required"), nil
	}
	oldVpcID, ok := args["OldVpcID"].(string)
	if !ok || oldVpcID == "" {
		return mcp.NewToolResultError("Old VPC ID is required"), nil
	}
	newVpcID, ok := args["NewVpcID"].(string)
	if !ok || newVpcID == "" {
		return mcp.NewToolResultError("New VPC ID is required"), nil
	}

	client, err := n.client(ctx)
	if err != nil {
		return mcp.NewToolResultErrorFromErr("Error getting DigitalOcean client", err), nil
	}

	action, _, err := client.NfsActions.Reassign(ctx, shareID, oldVpcID, newVpcID)
	if err != nil {
		return mcp.NewToolResultErrorFromErr("api error", err), nil
	}

	jsonAction, err := json.MarshalIndent(action, "", "  ")
	if err != nil {
		return mcp.NewToolResultErrorFromErr("marshal error", err), nil
	}
	return mcp.NewToolResultText(string(jsonAction)), nil
}

func (n *NfsActionsTool) switchPerformanceTier(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	args := req.GetArguments()
	shareID, ok := args["ShareID"].(string)
	if !ok || shareID == "" {
		return mcp.NewToolResultError("Share ID is required"), nil
	}
	performanceTier, ok := args["PerformanceTier"].(string)
	if !ok || performanceTier == "" {
		return mcp.NewToolResultError("Performance tier is required"), nil
	}

	client, err := n.client(ctx)
	if err != nil {
		return mcp.NewToolResultErrorFromErr("Error getting DigitalOcean client", err), nil
	}

	action, _, err := client.NfsActions.SwitchPerformanceTier(ctx, shareID, performanceTier)
	if err != nil {
		return mcp.NewToolResultErrorFromErr("api error", err), nil
	}

	jsonAction, err := json.MarshalIndent(action, "", "  ")
	if err != nil {
		return mcp.NewToolResultErrorFromErr("marshal error", err), nil
	}
	return mcp.NewToolResultText(string(jsonAction)), nil
}

func (n *NfsActionsTool) Tools() []server.ServerTool {
	return []server.ServerTool{
		{
			Handler: n.resizeFileShare,
			Tool: mcp.NewTool("nfs-resize",
				common.WithHints(common.HintsToggle),
				common.WithRisk(common.RiskMedium),
				mcp.WithDescription("Resize a NFS file share"),
				mcp.WithString("ShareID", mcp.Required(), mcp.Description("ID of the NFS file share to resize")),
				mcp.WithNumber("SizeGibibytes", mcp.Required(), mcp.Description("Size of the file share in GiB")),
			),
		},
		{
			Handler: n.snapshotFileShare,
			Tool: mcp.NewTool("nfs-snapshot",
				common.WithHints(common.HintsCreate),
				common.WithRisk(common.RiskLow),
				mcp.WithDescription("Create a snapshot of a NFS file share"),
				mcp.WithString("ShareID", mcp.Required(), mcp.Description("ID of the NFS file share to snapshot")),
				mcp.WithString("SnapshotName", mcp.Required(), mcp.Description("Name of the snapshot")),
			),
		},
		{
			Handler: n.attachFileShare,
			Tool: mcp.NewTool("nfs-attach",
				common.WithHints(common.HintsToggle),
				common.WithRisk(common.RiskMedium),
				mcp.WithDescription("Attach a NFS file share to a VPC"),
				mcp.WithString("ShareID", mcp.Required(), mcp.Description("ID of the NFS file share to attach")),
				mcp.WithString("VpcID", mcp.Required(), mcp.Description("ID of the VPC to attach the file share to")),
			),
		},
		{
			Handler: n.detachFileShare,
			Tool: mcp.NewTool("nfs-detach",
				common.WithHints(common.HintsToggle),
				common.WithRisk(common.RiskMedium),
				mcp.WithDescription("Detach a NFS file share from a VPC"),
				mcp.WithString("ShareID", mcp.Required(), mcp.Description("ID of the NFS file share to detach")),
				mcp.WithString("VpcID", mcp.Required(), mcp.Description("ID of the VPC to detach the file share from")),
			),
		},
		{
			Handler: n.reassignFileShare,
			Tool: mcp.NewTool("nfs-reassign",
				common.WithHints(common.HintsAction),
				common.WithRisk(common.RiskMedium),
				mcp.WithDescription("Reassign a NFS file share from one VPC to another"),
				mcp.WithString("ShareID", mcp.Required(), mcp.Description("ID of the NFS file share to reassign")),
				mcp.WithString("OldVpcID", mcp.Required(), mcp.Description("ID of the VPC to reassign the file share from")),
				mcp.WithString("NewVpcID", mcp.Required(), mcp.Description("ID of the VPC to reassign the file share to")),
			),
		},
		{
			Handler: n.switchPerformanceTier,
			Tool: mcp.NewTool("nfs-switch-performance-tier",
				common.WithHints(common.HintsToggle),
				common.WithRisk(common.RiskMedium),
				mcp.WithDescription("Switch the performance tier of a NFS file share"),
				mcp.WithString("ShareID", mcp.Required(), mcp.Description("ID of the NFS file share to switch the performance tier of")),
				mcp.WithString("PerformanceTier", mcp.Required(), mcp.Description("Performance tier to switch the file share to")),
			),
		},
	}
}
