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

// Checkpoint is the public record of a captured sandbox-machine-state snapshot.
type Checkpoint struct {
	CheckpointID string     `json:"checkpoint_id"`
	SessionID    string     `json:"session_id,omitempty"`
	Status       string     `json:"status,omitempty"`
	Kind         string     `json:"kind,omitempty"`
	Label        string     `json:"label,omitempty"`
	EventID      string     `json:"event_id,omitempty"`
	SizeBytes    int64      `json:"size_bytes,omitempty"`
	CreatedAt    *time.Time `json:"created_at,omitempty"`
	ErrorMessage string     `json:"error_message,omitempty"`
}

// CreateCheckpointResponse is the response of POST .../checkpoints.
type CreateCheckpointResponse struct {
	Checkpoint Checkpoint `json:"checkpoint"`
}

// GetCheckpointResponse is the response of GET .../checkpoints/{id}.
type GetCheckpointResponse struct {
	Checkpoint Checkpoint `json:"checkpoint"`
}

// ListCheckpointsResponse is the response of GET .../checkpoints.
type ListCheckpointsResponse struct {
	Checkpoints   []Checkpoint `json:"checkpoints"`
	NextPageToken string       `json:"next_page_token,omitempty"`
}

// DeleteCheckpointResponse is the response of DELETE .../checkpoints/{id}.
type DeleteCheckpointResponse struct {
	CheckpointID string `json:"checkpoint_id"`
	Deleted      bool   `json:"deleted"`
}

// RollbackToCheckpointResponse is the response of POST .../checkpoints/{id}/rollback.
type RollbackToCheckpointResponse struct {
	Session Session `json:"session"`
}

// checkpointsPath returns the base path for a session's checkpoints.
func checkpointsPath(sessionID string) string {
	return sessionsPath + "/" + sessionID + "/checkpoints"
}

// createCheckpoint captures the session's live sandbox state as a durable checkpoint.
func (t *Tool) createCheckpoint(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
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

	body := map[string]any{}
	if v, _ := args["Label"].(string); v != "" {
		body["label"] = v
	}

	encoded, err := json.Marshal(body)
	if err != nil {
		return mcp.NewToolResultErrorFromErr("Failed to encode request body", err), nil
	}

	apiReq, err := client.NewRequest(ctx, http.MethodPost, checkpointsPath(sessionID), nil)
	if err != nil {
		return mcp.NewToolResultErrorFromErr("Failed to create request", err), nil
	}
	apiReq.Header.Set("Content-Type", "application/json")
	apiReq.Body = io.NopCloser(bytes.NewReader(encoded))
	apiReq.ContentLength = int64(len(encoded))

	var out CreateCheckpointResponse
	if _, err := client.Do(ctx, apiReq, &out); err != nil {
		return mcp.NewToolResultErrorFromErr("Failed to create checkpoint", err), nil
	}
	return checkpointCreateOut.Result(out)
}

// getCheckpoint returns the current snapshot of one checkpoint.
func (t *Tool) getCheckpoint(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
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
	checkpointID, _ := args["CheckpointID"].(string)
	if checkpointID == "" {
		return mcp.NewToolResultError("CheckpointID is required"), nil
	}

	path := checkpointsPath(sessionID) + "/" + checkpointID
	apiReq, err := client.NewRequest(ctx, http.MethodGet, path, nil)
	if err != nil {
		return mcp.NewToolResultErrorFromErr("Failed to create request", err), nil
	}

	var out GetCheckpointResponse
	if _, err := client.Do(ctx, apiReq, &out); err != nil {
		return mcp.NewToolResultErrorFromErr("Failed to get checkpoint", err), nil
	}
	return checkpointOut.Result(out)
}

// listCheckpoints enumerates a session's checkpoints, newest first.
func (t *Tool) listCheckpoints(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
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

	path := checkpointsPath(sessionID)
	if len(query) > 0 {
		path += "?" + query.Encode()
	}

	apiReq, err := client.NewRequest(ctx, http.MethodGet, path, nil)
	if err != nil {
		return mcp.NewToolResultErrorFromErr("Failed to create request", err), nil
	}

	var out ListCheckpointsResponse
	if _, err := client.Do(ctx, apiReq, &out); err != nil {
		return mcp.NewToolResultErrorFromErr("Failed to list checkpoints", err), nil
	}
	if out.Checkpoints == nil {
		out.Checkpoints = []Checkpoint{}
	}
	return checkpointListOut.Result(out)
}

// deleteCheckpoint removes a checkpoint and its underlying machine-state capture.
func (t *Tool) deleteCheckpoint(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
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
	checkpointID, _ := args["CheckpointID"].(string)
	if checkpointID == "" {
		return mcp.NewToolResultError("CheckpointID is required"), nil
	}

	path := checkpointsPath(sessionID) + "/" + checkpointID
	apiReq, err := client.NewRequest(ctx, http.MethodDelete, path, nil)
	if err != nil {
		return mcp.NewToolResultErrorFromErr("Failed to create request", err), nil
	}

	var out DeleteCheckpointResponse
	if _, err := client.Do(ctx, apiReq, &out); err != nil {
		return mcp.NewToolResultErrorFromErr("Failed to delete checkpoint", err), nil
	}
	return checkpointDeleteOut.Result(out)
}

// rollbackToCheckpoint rewinds a session to a previous checkpoint.
func (t *Tool) rollbackToCheckpoint(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
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
	checkpointID, _ := args["CheckpointID"].(string)
	if checkpointID == "" {
		return mcp.NewToolResultError("CheckpointID is required"), nil
	}

	path := checkpointsPath(sessionID) + "/" + checkpointID + "/rollback"
	apiReq, err := client.NewRequest(ctx, http.MethodPost, path, nil)
	if err != nil {
		return mcp.NewToolResultErrorFromErr("Failed to create request", err), nil
	}

	var out RollbackToCheckpointResponse
	if _, err := client.Do(ctx, apiReq, &out); err != nil {
		return mcp.NewToolResultErrorFromErr("Failed to rollback to checkpoint", err), nil
	}
	return rollbackOut.Result(out)
}

// checkpointTools returns the checkpoint MCP tools to be merged into Tools().
func (t *Tool) checkpointTools() []server.ServerTool {
	return []server.ServerTool{
		{
			Handler: t.listCheckpoints,
			Tool: mcp.NewTool("harness-runtime-list-checkpoints",
				common.WithHints(common.HintsRead),
				common.WithRisk(common.RiskLow),
				checkpointListOut.Schema(),
				mcp.WithDescription("List a session's checkpoints, newest first. Keyset-paginated; pass next_page_token back as PageToken for the next page. Only READY checkpoints can be used for forks or rollback."),
				mcp.WithString("SessionID", mcp.Required(), mcp.Description("The session UUID whose checkpoints to list")),
				mcp.WithNumber("PageSize", mcp.Description("Page size (1-200, default 50)")),
				mcp.WithString("PageToken", mcp.Description("Opaque cursor from the previous response's next_page_token")),
			),
		},
		{
			Handler: t.getCheckpoint,
			Tool: mcp.NewTool("harness-runtime-get-checkpoint",
				common.WithHints(common.HintsRead),
				common.WithRisk(common.RiskLow),
				checkpointOut.Schema(),
				mcp.WithDescription("Get the current snapshot of a checkpoint, including its status (PENDING → READY | FAILED), kind (explicit/implicit), label, and size. Poll until status is READY before forking or rolling back."),
				mcp.WithString("SessionID", mcp.Required(), mcp.Description("The session UUID that owns the checkpoint")),
				mcp.WithString("CheckpointID", mcp.Required(), mcp.Description("The checkpoint ID (e.g. cp_9f2c1a4b)")),
			),
		},
		{
			Handler: t.createCheckpoint,
			Tool: mcp.NewTool("harness-runtime-create-checkpoint",
				common.WithHints(common.HintsCreate),
				common.WithRisk(common.RiskMedium),
				checkpointCreateOut.Schema(),
				mcp.WithDescription("Capture the session's live sandbox state as a durable checkpoint. Synchronous: blocks until the checkpoint is READY and returns it directly (no poll needed). The session must not have an active turn in flight (409 if busy). Returns 501 for agents/substrates that do not support checkpoints."),
				mcp.WithString("SessionID", mcp.Required(), mcp.Description("The session UUID to checkpoint")),
				mcp.WithString("Label", mcp.Description("Optional human-readable label for the checkpoint (stored as-is)")),
			),
		},
		{
			Handler: t.deleteCheckpoint,
			Tool: mcp.NewTool("harness-runtime-delete-checkpoint",
				common.WithHints(common.HintsDelete),
				common.WithRisk(common.RiskHigh),
				checkpointDeleteOut.Schema(),
				mcp.WithDescription("Delete a checkpoint and its underlying machine-state capture. Idempotent: deleting an already-gone checkpoint still returns success. Frees the storage quota consumed by the capture."),
				mcp.WithString("SessionID", mcp.Required(), mcp.Description("The session UUID that owns the checkpoint")),
				mcp.WithString("CheckpointID", mcp.Required(), mcp.Description("The checkpoint ID to delete (e.g. cp_9f2c1a4b)")),
			),
		},
		{
			Handler: t.rollbackToCheckpoint,
			Tool: mcp.NewTool("harness-runtime-rollback-checkpoint",
				common.WithHints(common.HintsDelete),
				common.WithRisk(common.RiskHigh),
				rollbackOut.Schema(),
				mcp.WithDescription("Rewind a session to a READY checkpoint. In-place restore: boots a fresh sandbox from the checkpoint and atomically swaps the session's sandbox pointer, so the session keeps its same session_id. The old sandbox is torn down. The session must not have a workspace attached (409). Returns the session with its refreshed sandbox_id. Note: the ingress URL is revoked on rollback — re-fetch with harness-runtime-get-ingress-url after the session is READY."),
				mcp.WithString("SessionID", mcp.Required(), mcp.Description("The session UUID to roll back")),
				mcp.WithString("CheckpointID", mcp.Required(), mcp.Description("The checkpoint ID to restore from (must be READY)")),
			),
		},
	}
}
