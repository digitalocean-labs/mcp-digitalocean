package harnessruntime

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"math"
	"net/http"
	"net/url"
	"strconv"
	"time"

	"github.com/google/uuid"
	"github.com/mark3labs/mcp-go/mcp"
	"github.com/mark3labs/mcp-go/server"

	"mcp-digitalocean/pkg/registry/common"
)

const (
	workspacesPath = "v2/harness-runtime/workspaces"
)

// Workspace is the user-facing record of a persistent workspace volume.
type Workspace struct {
	WorkspaceID       string     `json:"workspace_id"`
	Name              string     `json:"name,omitempty"`
	State             string     `json:"state"`
	AttachedSessionID string     `json:"attached_session_id,omitempty"`
	SizeGibibytes     int32      `json:"size_gibibytes"`
	BytesUsed         int64      `json:"bytes_used"`
	LastSavedAt       *time.Time `json:"last_saved_at,omitempty"`
	CreatedAt         time.Time  `json:"created_at"`
	UpdatedAt         time.Time  `json:"updated_at"`
	CreatedBy         string     `json:"created_by"`
	UpdatedBy         string     `json:"updated_by"`
}

// CreateWorkspaceResponse is the response of POST /v2/harness-runtime/workspaces.
type CreateWorkspaceResponse struct {
	Workspace Workspace `json:"workspace"`
}

// GetWorkspaceResponse is the response of GET /v2/harness-runtime/workspaces/{id}.
type GetWorkspaceResponse struct {
	Workspace Workspace `json:"workspace"`
}

// ListWorkspacesResponse is the response of GET /v2/harness-runtime/workspaces.
type ListWorkspacesResponse struct {
	Workspaces    []Workspace `json:"workspaces"`
	NextPageToken string      `json:"next_page_token,omitempty"`
}

// WorkspaceUploadResponse is the response of POST .../workspace/upload.
type WorkspaceUploadResponse struct {
	Path         string `json:"path,omitempty"`
	BytesWritten int64  `json:"bytes_written,omitempty"`
}

// WorkspaceDownloadResponse is synthesized from the download meta.
// The actual download is a binary stream; for the MCP tool we return the
// metadata that lets the caller fetch the file themselves.
type WorkspaceDownloadResponse struct {
	SessionID string `json:"session_id"`
	Path      string `json:"path"`
	URL       string `json:"url"`
}

// WorkspaceTransferCreated is the response of POST .../workspace/transfers.
type WorkspaceTransferCreated struct {
	TransferID string     `json:"transfer_id"`
	Direction  string     `json:"direction,omitempty"`
	Status     string     `json:"status,omitempty"`
	UploadID   string     `json:"upload_id,omitempty"`
	PartSize   int64      `json:"part_size,omitempty"`
	ExpiresAt  *time.Time `json:"expires_at,omitempty"`
}

// WorkspaceTransferStatus is the response of GET .../workspace/transfers/{id}.
type WorkspaceTransferStatus struct {
	TransferID   string     `json:"transfer_id"`
	Direction    string     `json:"direction,omitempty"`
	Status       string     `json:"status,omitempty"`
	BytesWritten int64      `json:"bytes_written,omitempty"`
	SHA256       string     `json:"sha256,omitempty"`
	DownloadURL  string     `json:"download_url,omitempty"`
	ExpiresAt    *time.Time `json:"expires_at,omitempty"`
	ErrorMessage string     `json:"error_message,omitempty"`
}

// WorkspaceTransferCancelled is the response of POST .../transfers/{id}/cancel.
type WorkspaceTransferCancelled struct {
	TransferID string `json:"transfer_id"`
	Aborted    bool   `json:"aborted"`
	Status     string `json:"status,omitempty"`
}

// WorkspaceUploadCommitted is the response of POST .../transfers/{id}/commit.
type WorkspaceUploadCommitted struct {
	TransferID string `json:"transfer_id"`
	Status     string `json:"status,omitempty"`
	SizeBytes  int64  `json:"size_bytes,omitempty"`
}

// WorkspacePartUploadURL is a presigned PUT URL for one part of a multipart upload.
type WorkspacePartUploadURL struct {
	PartNumber int32  `json:"part_number"`
	UploadURL  string `json:"upload_url"`
}

// WorkspacePartUploadURLs is the response of POST .../transfers/{id}/part-upload-urls.
type WorkspacePartUploadURLs struct {
	TransferID string                   `json:"transfer_id"`
	PartURLs   []WorkspacePartUploadURL `json:"part_urls"`
	ExpiresAt  *time.Time               `json:"expires_at,omitempty"`
}

// ---- Workspace CRUD ----

// createWorkspace provisions a new persistent workspace volume.
func (t *Tool) createWorkspace(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	client, err := t.client(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to get DigitalOcean client: %w", err)
	}

	args := req.GetArguments()
	sizeRaw, ok := args["SizeGibibytes"].(float64)
	if !ok || sizeRaw != math.Trunc(sizeRaw) || sizeRaw < 1 || sizeRaw > 100 {
		return mcp.NewToolResultError("SizeGibibytes is required and must be an integer between 1 and 100"), nil
	}

	body := map[string]any{
		"size_gibibytes": int32(sizeRaw),
	}
	if v, _ := args["Name"].(string); v != "" {
		body["name"] = v
	}

	encoded, err := json.Marshal(body)
	if err != nil {
		return mcp.NewToolResultErrorFromErr("Failed to encode request body", err), nil
	}

	apiReq, err := client.NewRequest(ctx, http.MethodPost, workspacesPath, nil)
	if err != nil {
		return mcp.NewToolResultErrorFromErr("Failed to create request", err), nil
	}
	apiReq.Header.Set("Content-Type", "application/json")
	apiReq.Body = io.NopCloser(bytes.NewReader(encoded))
	apiReq.ContentLength = int64(len(encoded))

	var out CreateWorkspaceResponse
	if _, err := client.Do(ctx, apiReq, &out); err != nil {
		return mcp.NewToolResultErrorFromErr("Failed to create workspace", err), nil
	}
	return workspaceCreateOut.Result(out)
}

// listWorkspaces enumerates the team's persistent workspaces.
func (t *Tool) listWorkspaces(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	client, err := t.client(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to get DigitalOcean client: %w", err)
	}

	args := req.GetArguments()
	query := url.Values{}

	if v, ok := args["PageSize"].(float64); ok {
		if v != math.Trunc(v) || v < 1 || v > maxPageSize {
			return mcp.NewToolResultError(fmt.Sprintf("PageSize must be an integer between 1 and %d", maxPageSize)), nil
		}
		query.Set("page_size", strconv.Itoa(int(v)))
	}
	if v, _ := args["PageToken"].(string); v != "" {
		query.Set("page_token", v)
	}
	if v, _ := args["State"].(string); v != "" {
		query.Set("state", v)
	}

	path := workspacesPath
	if len(query) > 0 {
		path += "?" + query.Encode()
	}

	apiReq, err := client.NewRequest(ctx, http.MethodGet, path, nil)
	if err != nil {
		return mcp.NewToolResultErrorFromErr("Failed to create request", err), nil
	}

	var out ListWorkspacesResponse
	if _, err := client.Do(ctx, apiReq, &out); err != nil {
		return mcp.NewToolResultErrorFromErr("Failed to list workspaces", err), nil
	}
	if out.Workspaces == nil {
		out.Workspaces = []Workspace{}
	}
	return workspaceListOut.Result(out)
}

// getWorkspace returns a single workspace by UUID.
func (t *Tool) getWorkspace(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	client, err := t.client(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to get DigitalOcean client: %w", err)
	}

	workspaceID, _ := req.GetArguments()["WorkspaceID"].(string)
	if workspaceID == "" {
		return mcp.NewToolResultError("WorkspaceID is required"), nil
	}
	if _, err := uuid.Parse(workspaceID); err != nil {
		return mcp.NewToolResultError("WorkspaceID must be a UUID"), nil
	}

	apiReq, err := client.NewRequest(ctx, http.MethodGet, workspacesPath+"/"+workspaceID, nil)
	if err != nil {
		return mcp.NewToolResultErrorFromErr("Failed to create request", err), nil
	}

	var out GetWorkspaceResponse
	if _, err := client.Do(ctx, apiReq, &out); err != nil {
		return mcp.NewToolResultErrorFromErr("Failed to get workspace", err), nil
	}
	return workspaceOut.Result(out)
}

// deleteWorkspace permanently removes a workspace.
func (t *Tool) deleteWorkspace(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	client, err := t.client(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to get DigitalOcean client: %w", err)
	}

	workspaceID, _ := req.GetArguments()["WorkspaceID"].(string)
	if workspaceID == "" {
		return mcp.NewToolResultError("WorkspaceID is required"), nil
	}
	if _, err := uuid.Parse(workspaceID); err != nil {
		return mcp.NewToolResultError("WorkspaceID must be a UUID"), nil
	}

	apiReq, err := client.NewRequest(ctx, http.MethodDelete, workspacesPath+"/"+workspaceID, nil)
	if err != nil {
		return mcp.NewToolResultErrorFromErr("Failed to create request", err), nil
	}

	if _, err := client.Do(ctx, apiReq, nil); err != nil {
		return mcp.NewToolResultErrorFromErr("Failed to delete workspace", err), nil
	}
	return mcp.NewToolResultText(fmt.Sprintf("Workspace %s deleted successfully.", workspaceID)), nil
}

// ---- Session-scoped workspace transfer tools ----

func workspaceTransfersPath(sessionID string) string {
	return sessionsPath + "/" + sessionID + "/workspace/transfers"
}

// createWorkspaceTransfer initiates a multipart upload or download transfer.
func (t *Tool) createWorkspaceTransfer(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	client, err := t.client(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to get DigitalOcean client: %w", err)
	}

	args := req.GetArguments()
	sessionID, _ := args["SessionID"].(string)
	if sessionID == "" {
		return mcp.NewToolResultError("SessionID is required"), nil
	}
	if _, err := uuid.Parse(sessionID); err != nil {
		return mcp.NewToolResultError("SessionID must be a UUID"), nil
	}

	direction, _ := args["Direction"].(string)
	if direction != "upload" && direction != "download" {
		return mcp.NewToolResultError("Direction is required and must be 'upload' or 'download'"), nil
	}

	wsPath, _ := args["Path"].(string)
	if wsPath == "" {
		return mcp.NewToolResultError("Path is required"), nil
	}

	body := map[string]any{
		"direction": direction,
		"path":      wsPath,
	}
	if v, ok := args["IsArchive"].(bool); ok {
		body["is_archive"] = v
	}
	if v, ok := args["AsArchive"].(bool); ok {
		body["as_archive"] = v
	}
	if v, ok := args["SizeBytes"].(float64); ok {
		body["size_bytes"] = int64(v)
	}
	if v, _ := args["SHA256"].(string); v != "" {
		body["sha256"] = v
	}

	encoded, err := json.Marshal(body)
	if err != nil {
		return mcp.NewToolResultErrorFromErr("Failed to encode request body", err), nil
	}

	apiReq, err := client.NewRequest(ctx, http.MethodPost, workspaceTransfersPath(sessionID), nil)
	if err != nil {
		return mcp.NewToolResultErrorFromErr("Failed to create request", err), nil
	}
	apiReq.Header.Set("Content-Type", "application/json")
	apiReq.Body = io.NopCloser(bytes.NewReader(encoded))
	apiReq.ContentLength = int64(len(encoded))

	var out WorkspaceTransferCreated
	if _, err := client.Do(ctx, apiReq, &out); err != nil {
		return mcp.NewToolResultErrorFromErr("Failed to create workspace transfer", err), nil
	}
	return wsTransferCreateOut.Result(out)
}

// getWorkspaceTransfer returns the status of a workspace transfer.
func (t *Tool) getWorkspaceTransfer(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	client, err := t.client(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to get DigitalOcean client: %w", err)
	}

	args := req.GetArguments()
	sessionID, _ := args["SessionID"].(string)
	if sessionID == "" {
		return mcp.NewToolResultError("SessionID is required"), nil
	}
	if _, err := uuid.Parse(sessionID); err != nil {
		return mcp.NewToolResultError("SessionID must be a UUID"), nil
	}
	transferID, _ := args["TransferID"].(string)
	if transferID == "" {
		return mcp.NewToolResultError("TransferID is required"), nil
	}

	path := workspaceTransfersPath(sessionID) + "/" + transferID
	apiReq, err := client.NewRequest(ctx, http.MethodGet, path, nil)
	if err != nil {
		return mcp.NewToolResultErrorFromErr("Failed to create request", err), nil
	}

	var out WorkspaceTransferStatus
	if _, err := client.Do(ctx, apiReq, &out); err != nil {
		return mcp.NewToolResultErrorFromErr("Failed to get workspace transfer", err), nil
	}
	return wsTransferStatusOut.Result(out)
}

// cancelWorkspaceTransfer aborts an active workspace transfer.
func (t *Tool) cancelWorkspaceTransfer(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	client, err := t.client(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to get DigitalOcean client: %w", err)
	}

	args := req.GetArguments()
	sessionID, _ := args["SessionID"].(string)
	if sessionID == "" {
		return mcp.NewToolResultError("SessionID is required"), nil
	}
	if _, err := uuid.Parse(sessionID); err != nil {
		return mcp.NewToolResultError("SessionID must be a UUID"), nil
	}
	transferID, _ := args["TransferID"].(string)
	if transferID == "" {
		return mcp.NewToolResultError("TransferID is required"), nil
	}

	path := workspaceTransfersPath(sessionID) + "/" + transferID + "/cancel"
	apiReq, err := client.NewRequest(ctx, http.MethodPost, path, nil)
	if err != nil {
		return mcp.NewToolResultErrorFromErr("Failed to create request", err), nil
	}

	var out WorkspaceTransferCancelled
	if _, err := client.Do(ctx, apiReq, &out); err != nil {
		return mcp.NewToolResultErrorFromErr("Failed to cancel workspace transfer", err), nil
	}
	return wsTransferCancelOut.Result(out)
}

// commitWorkspaceUpload finalizes a multipart upload.
func (t *Tool) commitWorkspaceUpload(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	client, err := t.client(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to get DigitalOcean client: %w", err)
	}

	args := req.GetArguments()
	sessionID, _ := args["SessionID"].(string)
	if sessionID == "" {
		return mcp.NewToolResultError("SessionID is required"), nil
	}
	if _, err := uuid.Parse(sessionID); err != nil {
		return mcp.NewToolResultError("SessionID must be a UUID"), nil
	}
	transferID, _ := args["TransferID"].(string)
	if transferID == "" {
		return mcp.NewToolResultError("TransferID is required"), nil
	}

	sha256, _ := args["SHA256"].(string)
	body := map[string]any{}
	if sha256 != "" {
		body["sha256"] = sha256
	}

	path := workspaceTransfersPath(sessionID) + "/" + transferID + "/commit"
	apiReq, err := client.NewRequest(ctx, http.MethodPost, path, body)
	if err != nil {
		return mcp.NewToolResultErrorFromErr("Failed to create request", err), nil
	}

	var out WorkspaceUploadCommitted
	if _, err := client.Do(ctx, apiReq, &out); err != nil {
		return mcp.NewToolResultErrorFromErr("Failed to commit workspace upload", err), nil
	}
	return wsUploadCommitOut.Result(out)
}

// createWorkspacePartUploadURLs requests presigned PUT URLs for multipart upload parts.
func (t *Tool) createWorkspacePartUploadURLs(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	client, err := t.client(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to get DigitalOcean client: %w", err)
	}

	args := req.GetArguments()
	sessionID, _ := args["SessionID"].(string)
	if sessionID == "" {
		return mcp.NewToolResultError("SessionID is required"), nil
	}
	if _, err := uuid.Parse(sessionID); err != nil {
		return mcp.NewToolResultError("SessionID must be a UUID"), nil
	}
	transferID, _ := args["TransferID"].(string)
	if transferID == "" {
		return mcp.NewToolResultError("TransferID is required"), nil
	}

	rawParts := args["PartNumbers"]
	if rawParts == nil {
		return mcp.NewToolResultError("PartNumbers is required"), nil
	}
	partsSlice, ok := rawParts.([]any)
	if !ok || len(partsSlice) == 0 {
		return mcp.NewToolResultError("PartNumbers must be a non-empty array of integers"), nil
	}
	parts := make([]int, 0, len(partsSlice))
	for _, p := range partsSlice {
		n, ok := p.(float64)
		if !ok || n != math.Trunc(n) || n < 1 {
			return mcp.NewToolResultError("Each element of PartNumbers must be a positive integer"), nil
		}
		parts = append(parts, int(n))
	}

	body := map[string]any{
		"part_numbers": parts,
	}

	encoded, err := json.Marshal(body)
	if err != nil {
		return mcp.NewToolResultErrorFromErr("Failed to encode request body", err), nil
	}

	path := workspaceTransfersPath(sessionID) + "/" + transferID + "/part-upload-urls"
	apiReq, err := client.NewRequest(ctx, http.MethodPost, path, nil)
	if err != nil {
		return mcp.NewToolResultErrorFromErr("Failed to create request", err), nil
	}
	apiReq.Header.Set("Content-Type", "application/json")
	apiReq.Body = io.NopCloser(bytes.NewReader(encoded))
	apiReq.ContentLength = int64(len(encoded))

	var out WorkspacePartUploadURLs
	if _, err := client.Do(ctx, apiReq, &out); err != nil {
		return mcp.NewToolResultErrorFromErr("Failed to create part upload URLs", err), nil
	}
	if out.PartURLs == nil {
		out.PartURLs = []WorkspacePartUploadURL{}
	}
	return wsPartURLsOut.Result(out)
}

// workspaceTools returns the workspace MCP tools to be merged into Tools().
func (t *Tool) workspaceTools() []server.ServerTool {
	return []server.ServerTool{
		// ---- Workspace CRUD ----
		{
			Handler: t.createWorkspace,
			Tool: mcp.NewTool("harness-runtime-create-workspace",
				common.WithHints(common.HintsCreate),
				common.WithRisk(common.RiskMedium),
				workspaceCreateOut.Schema(),
				mcp.WithDescription("Provision a new persistent workspace volume. The workspace is an empty virtual disk that can be attached to a session at create time (via WorkspaceID on create-session). Size is fixed at creation and counts against the team's storage quota."),
				mcp.WithNumber("SizeGibibytes", mcp.Required(), mcp.Description("Size in GiB (integer, 1-100). Fixed at creation.")),
				mcp.WithString("Name", mcp.Description("Optional human-readable label (not unique, not a substitute for the workspace_id)")),
			),
		},
		{
			Handler: t.listWorkspaces,
			Tool: mcp.NewTool("harness-runtime-list-workspaces",
				common.WithHints(common.HintsRead),
				common.WithRisk(common.RiskLow),
				workspaceListOut.Schema(),
				mcp.WithDescription("List the team's persistent workspaces. Returns workspace id, name, state, attached session, size, bytes used, and timestamps. Keyset-paginated: pass next_page_token back as PageToken for the next page."),
				mcp.WithNumber("PageSize", mcp.Description("Page size (1-200, default 50)")),
				mcp.WithString("PageToken", mcp.Description("Opaque cursor from the previous response's next_page_token")),
				mcp.WithString("State", mcp.Description("Optional state filter: AVAILABLE, ATTACHING, ATTACHED, RELEASING, FAILED, DELETING")),
			),
		},
		{
			Handler: t.getWorkspace,
			Tool: mcp.NewTool("harness-runtime-get-workspace",
				common.WithHints(common.HintsRead),
				common.WithRisk(common.RiskLow),
				workspaceOut.Schema(),
				mcp.WithDescription("Get a single persistent workspace by UUID, including its state, attached session, size, bytes used, and timestamps."),
				mcp.WithString("WorkspaceID", mcp.Required(), mcp.Description("The workspace UUID")),
			),
		},
		{
			Handler: t.deleteWorkspace,
			Tool: mcp.NewTool("harness-runtime-delete-workspace",
				common.WithHints(common.HintsDelete),
				common.WithRisk(common.RiskHigh),
				mcp.WithDescription("Permanently delete a persistent workspace by UUID. The workspace must not be attached to a session (409 if attached). Returns 204 on success. All data in the workspace is lost."),
				mcp.WithString("WorkspaceID", mcp.Required(), mcp.Description("The workspace UUID to delete")),
			),
		},
		// ---- Session-scoped workspace transfers ----
		{
			Handler: t.createWorkspaceTransfer,
			Tool: mcp.NewTool("harness-runtime-create-workspace-transfer",
				common.WithHints(common.HintsCreate),
				common.WithRisk(common.RiskMedium),
				wsTransferCreateOut.Schema(),
				mcp.WithDescription("Initiate a workspace bulk transfer (upload or download). For uploads: returns a transfer_id, upload_id, part_size, and expires_at — split the payload into parts of part_size bytes, then call create-workspace-part-upload-urls to get presigned PUT URLs, upload each part, and finally commit-workspace-upload. For downloads: starts an async export and returns transfer_id — poll get-workspace-transfer until status is 'completed' to get the download_url."),
				mcp.WithString("SessionID", mcp.Required(), mcp.Description("The session UUID whose workspace to transfer")),
				mcp.WithString("Direction", mcp.Required(), mcp.Description("Transfer direction: 'upload' or 'download'")),
				mcp.WithString("Path", mcp.Required(), mcp.Description("Workspace-relative path. Destination for upload, source for download. Paths that escape the workspace root are rejected.")),
				mcp.WithBoolean("IsArchive", mcp.Description("Upload only. When true, the payload is a tar the guest extracts at Path.")),
				mcp.WithBoolean("AsArchive", mcp.Description("Download only. When true, Path is exported as a tar stream.")),
				mcp.WithNumber("SizeBytes", mcp.Description("Optional total payload size in bytes. Validated against the max staged object size (50 GiB); a larger value returns 413.")),
				mcp.WithString("SHA256", mcp.Description("Optional lowercase hex SHA-256 of the full payload. If set on an upload, it must match on commit.")),
			),
		},
		{
			Handler: t.getWorkspaceTransfer,
			Tool: mcp.NewTool("harness-runtime-get-workspace-transfer",
				common.WithHints(common.HintsRead),
				common.WithRisk(common.RiskLow),
				wsTransferStatusOut.Schema(),
				mcp.WithDescription("Get the current status of a workspace transfer. For downloads, poll until status is 'completed' — the response then includes download_url (a presigned GET URL) and sha256 for verification. For uploads, check bytes_written progress."),
				mcp.WithString("SessionID", mcp.Required(), mcp.Description("The session UUID")),
				mcp.WithString("TransferID", mcp.Required(), mcp.Description("The transfer id from create-workspace-transfer")),
			),
		},
		{
			Handler: t.cancelWorkspaceTransfer,
			Tool: mcp.NewTool("harness-runtime-cancel-workspace-transfer",
				common.WithHints(common.HintsAction),
				common.WithRisk(common.RiskMedium),
				wsTransferCancelOut.Schema(),
				mcp.WithDescription("Abort an active workspace transfer. Idempotent: cancelling an already-terminal transfer returns success with aborted=false. Returns the transfer_id, aborted flag, and final status."),
				mcp.WithString("SessionID", mcp.Required(), mcp.Description("The session UUID")),
				mcp.WithString("TransferID", mcp.Required(), mcp.Description("The transfer id to cancel")),
			),
		},
		{
			Handler: t.commitWorkspaceUpload,
			Tool: mcp.NewTool("harness-runtime-commit-workspace-upload",
				common.WithHints(common.HintsAction),
				common.WithRisk(common.RiskMedium),
				wsUploadCommitOut.Schema(),
				mcp.WithDescription("Finalize a multipart workspace upload. Call after all parts have been PUT to their presigned URLs. The platform assembles the parts and writes the result into the workspace. Returns transfer_id, status, and size_bytes."),
				mcp.WithString("SessionID", mcp.Required(), mcp.Description("The session UUID")),
				mcp.WithString("TransferID", mcp.Required(), mcp.Description("The transfer id to commit")),
				mcp.WithString("SHA256", mcp.Description("Optional lowercase hex SHA-256 of the full payload. Must match the value supplied at create when both are set.")),
			),
		},
		{
			Handler: t.createWorkspacePartUploadURLs,
			Tool: mcp.NewTool("harness-runtime-create-workspace-part-urls",
				common.WithHints(common.HintsCreate),
				common.WithRisk(common.RiskLow),
				wsPartURLsOut.Schema(),
				mcp.WithDescription("Request presigned PUT URLs for a set of upload part numbers. Each returned URL accepts a single PUT of that part's bytes. URLs expire at the shared expires_at timestamp. Part numbers are 1-based and must match the part_size from create-workspace-transfer."),
				mcp.WithString("SessionID", mcp.Required(), mcp.Description("The session UUID")),
				mcp.WithString("TransferID", mcp.Required(), mcp.Description("The transfer id from create-workspace-transfer")),
				mcp.WithArray("PartNumbers", mcp.Required(), mcp.Description("Array of 1-based part numbers to get URLs for (e.g. [1, 2, 3])"), mcp.Items(map[string]any{"type": "integer"})),
			),
		},
	}
}
