package harnessruntime

import (
	"context"
	"fmt"
	"math"
	"net/http"
	"net/url"
	"strconv"

	"github.com/google/uuid"
	"github.com/mark3labs/mcp-go/mcp"
	"github.com/mark3labs/mcp-go/server"

	"mcp-digitalocean/pkg/registry/common"
)

const (
	imageMessagesPath = "v2/harness-runtime/image-messages"
)

// ImageMessage is one live notice on a sandbox image.
type ImageMessage struct {
	Type        string `json:"type"`
	Title       string `json:"title"`
	Message     string `json:"message"`
	MoreInfoURL string `json:"more_info_url"`
}

// ImageMessageResource is a session or checkpoint and the live notices on its
// sandbox image. Messages is empty when the image has no notice.
type ImageMessageResource struct {
	Kind               string         `json:"kind"`
	SessionID          string         `json:"session_id"`
	SessionName        string         `json:"session_name"`
	CheckpointID       string         `json:"checkpoint_id"`
	CheckpointName     string         `json:"checkpoint_name"`
	ImageID            string         `json:"image_id"`
	ResolutionComplete bool           `json:"resolution_complete"`
	Vulnerable         bool           `json:"vulnerable"`
	Messages           []ImageMessage `json:"messages"`
}

// ListImageMessagesResponse is the response of GET /v2/harness-runtime/image-messages.
type ListImageMessagesResponse struct {
	Resources     []ImageMessageResource `json:"resources"`
	NextPageToken string                 `json:"next_page_token,omitempty"`
}

// GetImageMessagesResponse is the response of the per-session and
// per-checkpoint image-messages reads.
type GetImageMessagesResponse struct {
	Resource ImageMessageResource `json:"resource"`
}

// imageMessageType reads the optional Type argument. The API documents one
// notice type; anything else is rejected before the call.
func imageMessageType(args map[string]any, query url.Values) *mcp.CallToolResult {
	v, _ := args["Type"].(string)
	if v == "" {
		return nil
	}
	if v != "vulnerability" {
		return mcp.NewToolResultError("Type must be 'vulnerability' or omitted")
	}
	query.Set("type", v)
	return nil
}

func withQuery(path string, query url.Values) string {
	if len(query) == 0 {
		return path
	}
	return path + "?" + query.Encode()
}

// listImageMessages lists the team's sessions and checkpoints whose sandbox image carries a live notice.
func (t *Tool) listImageMessages(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	client, err := t.client(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to get DigitalOcean client: %w", err)
	}

	args := req.GetArguments()
	query := url.Values{}
	if res := imageMessageType(args, query); res != nil {
		return res, nil
	}

	sessionName, _ := args["SessionName"].(string)
	checkpointName, _ := args["CheckpointName"].(string)
	if sessionName != "" && checkpointName != "" {
		return mcp.NewToolResultError("Set SessionName or CheckpointName, not both"), nil
	}
	if sessionName != "" {
		query.Set("session_name", sessionName)
	}
	if checkpointName != "" {
		query.Set("checkpoint_name", checkpointName)
	}

	if v, ok := args["PageSize"].(float64); ok {
		if v != math.Trunc(v) || v < 1 || v > maxPageSize {
			return mcp.NewToolResultError(fmt.Sprintf("PageSize must be an integer between 1 and %d", maxPageSize)), nil
		}
		query.Set("page_size", strconv.Itoa(int(v)))
	}
	if v, _ := args["PageToken"].(string); v != "" {
		query.Set("page_token", v)
	}

	apiReq, err := client.NewRequest(ctx, http.MethodGet, withQuery(imageMessagesPath, query), nil)
	if err != nil {
		return mcp.NewToolResultErrorFromErr("Failed to create request", err), nil
	}

	var out ListImageMessagesResponse
	if _, err := client.Do(ctx, apiReq, &out); err != nil {
		return mcp.NewToolResultErrorFromErr("Failed to list image messages", err), nil
	}
	if out.Resources == nil {
		out.Resources = []ImageMessageResource{}
	}
	for i := range out.Resources {
		if out.Resources[i].Messages == nil {
			out.Resources[i].Messages = []ImageMessage{}
		}
	}
	return imageMessageListOut.Result(out)
}

// getSessionImageMessages reads the live notices on one session's sandbox image.
func (t *Tool) getSessionImageMessages(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
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
	if res := imageMessageType(args, query); res != nil {
		return res, nil
	}

	path := sessionsPath + "/" + sessionID + "/image-messages"
	apiReq, err := client.NewRequest(ctx, http.MethodGet, withQuery(path, query), nil)
	if err != nil {
		return mcp.NewToolResultErrorFromErr("Failed to create request", err), nil
	}

	var out GetImageMessagesResponse
	if _, err := client.Do(ctx, apiReq, &out); err != nil {
		return mcp.NewToolResultErrorFromErr("Failed to get session image messages", err), nil
	}
	if out.Resource.Messages == nil {
		out.Resource.Messages = []ImageMessage{}
	}
	return imageMessageOut.Result(out)
}

// getCheckpointImageMessages reads the live notices on one checkpoint's sandbox image.
func (t *Tool) getCheckpointImageMessages(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
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

	query := url.Values{}
	if res := imageMessageType(args, query); res != nil {
		return res, nil
	}

	path := checkpointsPath(sessionID) + "/" + url.PathEscape(checkpointID) + "/image-messages"
	apiReq, err := client.NewRequest(ctx, http.MethodGet, withQuery(path, query), nil)
	if err != nil {
		return mcp.NewToolResultErrorFromErr("Failed to create request", err), nil
	}

	var out GetImageMessagesResponse
	if _, err := client.Do(ctx, apiReq, &out); err != nil {
		return mcp.NewToolResultErrorFromErr("Failed to get checkpoint image messages", err), nil
	}
	if out.Resource.Messages == nil {
		out.Resource.Messages = []ImageMessage{}
	}
	return imageMessageOut.Result(out)
}

// imageMessageTools returns the sandbox-image notice MCP tools to be merged into Tools().
func (t *Tool) imageMessageTools() []server.ServerTool {
	typeArg := mcp.WithString("Type", mcp.Description("Optional notice type filter. The only value is 'vulnerability'; omit it to return every type."))
	return []server.ServerTool{
		{
			Handler: t.listImageMessages,
			Tool: mcp.NewTool("harness-runtime-list-image-messages",
				common.WithHints(common.HintsRead),
				common.WithRisk(common.RiskLow),
				imageMessageListOut.Schema(),
				mcp.WithDescription("List the team's agent sessions and checkpoints whose sandbox image has a live notice, such as a known vulnerability with a published fix. Each resource has vulnerable, a messages list (title, message, more_info_url) and the session or checkpoint it belongs to. Set SessionName or CheckpointName (not both) to look up one resource by exact name; otherwise the list is keyset-paginated: pass next_page_token back as PageToken. Triggers are not listed; check the session a trigger runs on."),
				typeArg,
				mcp.WithString("SessionName", mcp.Description("Optional exact session name; returns that one session. Cannot be combined with CheckpointName")),
				mcp.WithString("CheckpointName", mcp.Description("Optional exact checkpoint name; returns that one checkpoint. Cannot be combined with SessionName")),
				mcp.WithNumber("PageSize", mcp.Description("Page size (1-200, default 50). Ignored when SessionName or CheckpointName is set")),
				mcp.WithString("PageToken", mcp.Description("Opaque cursor from the previous response's next_page_token. Ignored when SessionName or CheckpointName is set")),
			),
		},
		{
			Handler: t.getSessionImageMessages,
			Tool: mcp.NewTool("harness-runtime-get-session-image-messages",
				common.WithHints(common.HintsRead),
				common.WithRisk(common.RiskLow),
				imageMessageOut.Schema(),
				mcp.WithDescription("Get the live notices on one agent session's sandbox image, such as a known vulnerability with a published fix. messages is empty when the image has no notice."),
				mcp.WithString("SessionID", mcp.Required(), mcp.Description("The session UUID")),
				typeArg,
			),
		},
		{
			Handler: t.getCheckpointImageMessages,
			Tool: mcp.NewTool("harness-runtime-get-checkpoint-image-messages",
				common.WithHints(common.HintsRead),
				common.WithRisk(common.RiskLow),
				imageMessageOut.Schema(),
				mcp.WithDescription("Get the live notices on one checkpoint's sandbox image, such as a known vulnerability with a published fix. The checkpoint must belong to the session. messages is empty when the image has no notice."),
				mcp.WithString("SessionID", mcp.Required(), mcp.Description("The session UUID the checkpoint belongs to")),
				mcp.WithString("CheckpointID", mcp.Required(), mcp.Description("The checkpoint id, as returned by harness-runtime-list-checkpoints")),
				typeArg,
			),
		},
	}
}
