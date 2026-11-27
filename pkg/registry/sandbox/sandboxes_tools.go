package sandbox

import (
	"context"
	"fmt"
	"net/http"

	"github.com/mark3labs/mcp-go/mcp"
	"github.com/mark3labs/mcp-go/server"

	"mcp-digitalocean/pkg/registry/common"
)

// ---- Sandboxes ----

func (t *Tool) listSandboxes(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	args := req.GetArguments()
	q := pageQuery(args)
	if state := argString(args, "State"); state != "" {
		q.Set("state", state)
	}

	var out sandboxListResponse
	if err := t.do(ctx, http.MethodGet, withQuery(sandboxesPath, q), nil, &out); err != nil {
		return mcp.NewToolResultErrorFromErr("Failed to list sandboxes", err), nil
	}
	if out.Sandboxes == nil {
		out.Sandboxes = []Sandbox{}
	}
	return sandboxListOut.Result(out)
}

func (t *Tool) createSandbox(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	args := req.GetArguments()
	template, errRes := requireString(args, "Template")
	if errRes != nil {
		return errRes, nil
	}
	sizeSlug, errRes := requireString(args, "SizeSlug")
	if errRes != nil {
		return errRes, nil
	}

	body := createSandboxRequest{
		Template: template,
		SizeSlug: sizeSlug,
		EnvVars:  argStringMap(args, "EnvVars"),
	}
	if v, ok := args["AllowInternetAccess"].(bool); ok {
		body.AllowInternetAccess = &v
	}

	var out sandboxResponse
	if err := t.do(ctx, http.MethodPost, sandboxesPath, body, &out); err != nil {
		return mcp.NewToolResultErrorFromErr("Failed to create sandbox", err), nil
	}
	return sandboxOut.Result(out)
}

func (t *Tool) getSandbox(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	id, errRes := requireString(req.GetArguments(), "SandboxID")
	if errRes != nil {
		return errRes, nil
	}

	var out sandboxResponse
	if err := t.do(ctx, http.MethodGet, sandboxPath(id), nil, &out); err != nil {
		return mcp.NewToolResultErrorFromErr("Failed to get sandbox", err), nil
	}
	return sandboxOut.Result(out)
}

func (t *Tool) deleteSandbox(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	id, errRes := requireString(req.GetArguments(), "SandboxID")
	if errRes != nil {
		return errRes, nil
	}

	if err := t.do(ctx, http.MethodDelete, sandboxPath(id), nil, nil); err != nil {
		return mcp.NewToolResultErrorFromErr("Failed to delete sandbox", err), nil
	}
	return mcp.NewToolResultText(fmt.Sprintf("Sandbox %s deleted.", id)), nil
}

func (t *Tool) pauseSandbox(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	return t.sandboxStateChange(ctx, req, "pause", "Failed to pause sandbox")
}

func (t *Tool) resumeSandbox(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	return t.sandboxStateChange(ctx, req, "resume", "Failed to resume sandbox")
}

// sandboxStateChange posts to /v2/sandboxes/{id}/{action} and returns the
// sandbox in its new state.
func (t *Tool) sandboxStateChange(ctx context.Context, req mcp.CallToolRequest, action, failure string) (*mcp.CallToolResult, error) {
	id, errRes := requireString(req.GetArguments(), "SandboxID")
	if errRes != nil {
		return errRes, nil
	}

	var out sandboxResponse
	if err := t.do(ctx, http.MethodPost, sandboxPath(id, action), nil, &out); err != nil {
		return mcp.NewToolResultErrorFromErr(failure, err), nil
	}
	return sandboxOut.Result(out)
}

// sandboxTools returns the sandbox lifecycle tools.
func (t *Tool) sandboxTools() []server.ServerTool {
	return []server.ServerTool{
		{
			Handler: t.listSandboxes,
			Tool: mcp.NewTool("sandbox-list",
				common.WithHints(common.HintsRead),
				common.WithRisk(common.RiskLow),
				sandboxListOut.Schema(),
				mcp.WithDescription("List sandboxes for the team. Keyset-paginated: pass next_page_token back as PageToken for the next page. Requires the sandbox:read scope."),
				mcp.WithNumber("PageSize", mcp.Description("Page size")),
				mcp.WithString("PageToken", mcp.Description("Opaque cursor from the previous response's next_page_token")),
				mcp.WithString("State", mcp.Description("Optional state filter: SANDBOX_STATE_RUNNING, SANDBOX_STATE_PAUSED, SANDBOX_STATE_CREATING")),
			),
		},
		{
			Handler: t.createSandbox,
			Tool: mcp.NewTool("sandbox-create",
				common.WithHints(common.HintsCreate),
				common.WithRisk(common.RiskMedium),
				sandboxOut.Schema(),
				mcp.WithDescription("Create a sandbox from a template. Creation is asynchronous: the sandbox may be in SANDBOX_STATE_CREATING until it is ready; poll sandbox-get. Requires the sandbox:create scope."),
				mcp.WithString("Template", mcp.Required(), mcp.Description("Template to boot the sandbox from")),
				mcp.WithString("SizeSlug", mcp.Required(), mcp.Description("Compute size slug")),
				mcp.WithBoolean("AllowInternetAccess", mcp.Description("Whether the sandbox may reach the internet. Omit to use the API default.")),
				mcp.WithObject("EnvVars", mcp.Description("Environment variables for the sandbox, as a string-to-string object")),
			),
		},
		{
			Handler: t.getSandbox,
			Tool: mcp.NewTool("sandbox-get",
				common.WithHints(common.HintsRead),
				common.WithRisk(common.RiskLow),
				sandboxOut.Schema(),
				mcp.WithDescription("Get one sandbox by ID, including its state and size. Requires the sandbox:read scope."),
				mcp.WithString("SandboxID", mcp.Required(), mcp.Description("The sandbox ID")),
			),
		},
		{
			Handler: t.deleteSandbox,
			Tool: mcp.NewTool("sandbox-delete",
				common.WithHints(common.HintsDelete),
				common.WithRisk(common.RiskHigh),
				mcp.WithDescription("Permanently delete a sandbox. Destructive. Requires the sandbox:delete scope."),
				mcp.WithString("SandboxID", mcp.Required(), mcp.Description("The sandbox ID to delete")),
			),
		},
		{
			Handler: t.pauseSandbox,
			Tool: mcp.NewTool("sandbox-pause",
				common.WithHints(common.HintsToggle),
				common.WithRisk(common.RiskLow),
				sandboxOut.Schema(),
				mcp.WithDescription("Pause a running sandbox. Requires the sandbox:update scope."),
				mcp.WithString("SandboxID", mcp.Required(), mcp.Description("The sandbox ID")),
			),
		},
		{
			Handler: t.resumeSandbox,
			Tool: mcp.NewTool("sandbox-resume",
				common.WithHints(common.HintsToggle),
				common.WithRisk(common.RiskLow),
				sandboxOut.Schema(),
				mcp.WithDescription("Resume a paused sandbox. Requires the sandbox:update scope."),
				mcp.WithString("SandboxID", mcp.Required(), mcp.Description("The sandbox ID")),
			),
		},
	}
}

// ---- Snapshots ----

func (t *Tool) createSnapshot(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	args := req.GetArguments()
	id, errRes := requireString(args, "SandboxID")
	if errRes != nil {
		return errRes, nil
	}

	body := createSnapshotRequest{
		Name: argString(args, "Name"),
		TTL:  argString(args, "TTL"),
	}
	var out snapshotResponse
	if err := t.do(ctx, http.MethodPost, sandboxPath(id, "snapshots"), body, &out); err != nil {
		return mcp.NewToolResultErrorFromErr("Failed to create snapshot", err), nil
	}
	return snapshotOut.Result(out)
}

func (t *Tool) listSnapshots(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	args := req.GetArguments()
	q := pageQuery(args)
	if id := argString(args, "SandboxID"); id != "" {
		q.Set("sandbox_id", id)
	}

	var out snapshotListResponse
	if err := t.do(ctx, http.MethodGet, withQuery(sandboxPath("snapshots"), q), nil, &out); err != nil {
		return mcp.NewToolResultErrorFromErr("Failed to list snapshots", err), nil
	}
	if out.Snapshots == nil {
		out.Snapshots = []Snapshot{}
	}
	return snapshotListOut.Result(out)
}

func (t *Tool) getSnapshot(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	id, errRes := requireString(req.GetArguments(), "SnapshotID")
	if errRes != nil {
		return errRes, nil
	}

	var out snapshotResponse
	if err := t.do(ctx, http.MethodGet, sandboxPath("snapshots", id), nil, &out); err != nil {
		return mcp.NewToolResultErrorFromErr("Failed to get snapshot", err), nil
	}
	return snapshotOut.Result(out)
}

func (t *Tool) deleteSnapshot(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	id, errRes := requireString(req.GetArguments(), "SnapshotID")
	if errRes != nil {
		return errRes, nil
	}

	if err := t.do(ctx, http.MethodDelete, sandboxPath("snapshots", id), nil, nil); err != nil {
		return mcp.NewToolResultErrorFromErr("Failed to delete snapshot", err), nil
	}
	return mcp.NewToolResultText(fmt.Sprintf("Snapshot %s deleted.", id)), nil
}

// snapshotTools returns the snapshot tools.
func (t *Tool) snapshotTools() []server.ServerTool {
	return []server.ServerTool{
		{
			Handler: t.createSnapshot,
			Tool: mcp.NewTool("sandbox-snapshot-create",
				common.WithHints(common.HintsCreate),
				common.WithRisk(common.RiskMedium),
				snapshotOut.Schema(),
				mcp.WithDescription("Snapshot a sandbox. Returns 202 with the snapshot in SNAPSHOT_STATE_CREATING; poll sandbox-snapshot-get until it is SNAPSHOT_STATE_COMPLETE. Requires the sandbox:update scope."),
				mcp.WithString("SandboxID", mcp.Required(), mcp.Description("The sandbox to snapshot")),
				mcp.WithString("Name", mcp.Description("Optional snapshot name")),
				mcp.WithString("TTL", mcp.Description("Optional time to live for the snapshot, e.g. 24h")),
			),
		},
		{
			Handler: t.listSnapshots,
			Tool: mcp.NewTool("sandbox-snapshot-list",
				common.WithHints(common.HintsRead),
				common.WithRisk(common.RiskLow),
				snapshotListOut.Schema(),
				mcp.WithDescription("List sandbox snapshots, optionally filtered to one sandbox. Keyset-paginated. Requires the sandbox:read scope."),
				mcp.WithString("SandboxID", mcp.Description("Only list snapshots of this sandbox")),
				mcp.WithNumber("PageSize", mcp.Description("Page size")),
				mcp.WithString("PageToken", mcp.Description("Opaque cursor from the previous response's next_page_token")),
			),
		},
		{
			Handler: t.getSnapshot,
			Tool: mcp.NewTool("sandbox-snapshot-get",
				common.WithHints(common.HintsRead),
				common.WithRisk(common.RiskLow),
				snapshotOut.Schema(),
				mcp.WithDescription("Get one sandbox snapshot by ID, including its state and size. Requires the sandbox:read scope."),
				mcp.WithString("SnapshotID", mcp.Required(), mcp.Description("The snapshot ID")),
			),
		},
		{
			Handler: t.deleteSnapshot,
			Tool: mcp.NewTool("sandbox-snapshot-delete",
				common.WithHints(common.HintsDelete),
				common.WithRisk(common.RiskMedium),
				mcp.WithDescription("Permanently delete a sandbox snapshot. Destructive. Requires the sandbox:update scope."),
				mcp.WithString("SnapshotID", mcp.Required(), mcp.Description("The snapshot ID to delete")),
			),
		},
	}
}
