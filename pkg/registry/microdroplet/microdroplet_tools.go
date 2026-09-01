package microdroplet

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"strings"

	"github.com/digitalocean/godo"
	"github.com/mark3labs/mcp-go/mcp"
	"github.com/mark3labs/mcp-go/server"

	"mcp-digitalocean/pkg/registry/common"
)

// MicroDropletTool exposes MicroDroplet lifecycle tools over the public REST API.
type MicroDropletTool struct {
	client func(ctx context.Context) (*godo.Client, error)
}

// NewMicroDropletTool creates a MicroDropletTool backed by the shared godo client.
func NewMicroDropletTool(client func(ctx context.Context) (*godo.Client, error)) *MicroDropletTool {
	return &MicroDropletTool{client: client}
}

func (t *MicroDropletTool) api(ctx context.Context) (*apiClient, error) {
	c, err := t.client(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to get DigitalOcean client: %w", err)
	}
	return newAPIClient(c), nil
}

func (t *MicroDropletTool) create(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	args := req.GetArguments()

	name, _ := args["name"].(string)
	region, _ := args["region"].(string)
	size, _ := args["size"].(string)
	if strings.TrimSpace(name) == "" {
		return mcp.NewToolResultError("name is required"), nil
	}
	if strings.TrimSpace(region) == "" {
		return mcp.NewToolResultError("region is required"), nil
	}
	if strings.TrimSpace(size) == "" {
		return mcp.NewToolResultError("size is required"), nil
	}

	body, err := buildCreateBody(args)
	if err != nil {
		return mcp.NewToolResultError(err.Error()), nil
	}

	api, err := t.api(ctx)
	if err != nil {
		return nil, err
	}

	out, err := api.do(ctx, http.MethodPost, "/instances", nil, body)
	if err != nil {
		return mcp.NewToolResultErrorFromErr("microdroplet create", err), nil
	}
	return toolResultJSON(out)
}

func (t *MicroDropletTool) list(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	args := req.GetArguments()
	query := url.Values{}
	query.Set("page", fmt.Sprintf("%d", intFromArg(args["page"], 1)))
	query.Set("per_page", fmt.Sprintf("%d", intFromArg(args["per_page"], 25)))
	if v, _ := args["region"].(string); v != "" {
		query.Set("region", v)
	}
	if v, _ := args["name"].(string); v != "" {
		query.Set("name", v)
	}
	if v, _ := args["tag_name"].(string); v != "" {
		query.Set("tag_name", v)
	}

	api, err := t.api(ctx)
	if err != nil {
		return nil, err
	}

	out, err := api.do(ctx, http.MethodGet, "/instances", query, nil)
	if err != nil {
		return mcp.NewToolResultErrorFromErr("microdroplet list", err), nil
	}
	return toolResultJSON(out)
}

func (t *MicroDropletTool) get(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	id, errMsg := requiredUUID(req.GetArguments(), "id")
	if errMsg != "" {
		return mcp.NewToolResultError(errMsg), nil
	}

	api, err := t.api(ctx)
	if err != nil {
		return nil, err
	}

	out, err := api.do(ctx, http.MethodGet, "/instances/"+id, nil, nil)
	if err != nil {
		return mcp.NewToolResultErrorFromErr("microdroplet get", err), nil
	}
	return toolResultJSON(out)
}

func (t *MicroDropletTool) delete(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	id, errMsg := requiredUUID(req.GetArguments(), "id")
	if errMsg != "" {
		return mcp.NewToolResultError(errMsg), nil
	}

	api, err := t.api(ctx)
	if err != nil {
		return nil, err
	}

	if err := api.doNoContent(ctx, http.MethodDelete, "/instances/"+id); err != nil {
		return mcp.NewToolResultErrorFromErr("microdroplet delete", err), nil
	}
	return mcp.NewToolResultText("MicroDroplet deleted successfully"), nil
}

func (t *MicroDropletTool) pause(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	id, errMsg := requiredUUID(req.GetArguments(), "id")
	if errMsg != "" {
		return mcp.NewToolResultError(errMsg), nil
	}

	api, err := t.api(ctx)
	if err != nil {
		return nil, err
	}

	out, err := api.do(ctx, http.MethodPost, "/instances/"+id+"/pause", nil, nil)
	if err != nil {
		return mcp.NewToolResultErrorFromErr("microdroplet pause", err), nil
	}
	return toolResultJSON(out)
}

func (t *MicroDropletTool) resume(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	id, errMsg := requiredUUID(req.GetArguments(), "id")
	if errMsg != "" {
		return mcp.NewToolResultError(errMsg), nil
	}

	api, err := t.api(ctx)
	if err != nil {
		return nil, err
	}

	out, err := api.do(ctx, http.MethodPost, "/instances/"+id+"/resume", nil, nil)
	if err != nil {
		return mcp.NewToolResultErrorFromErr("microdroplet resume", err), nil
	}
	return toolResultJSON(out)
}

func (t *MicroDropletTool) checkpointCreate(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	args := req.GetArguments()
	id, errMsg := requiredUUID(args, "id")
	if errMsg != "" {
		return mcp.NewToolResultError(errMsg), nil
	}

	var body any
	if name, ok := args["name"].(string); ok && strings.TrimSpace(name) != "" {
		body = map[string]string{"name": name}
	}

	api, err := t.api(ctx)
	if err != nil {
		return nil, err
	}

	out, err := api.do(ctx, http.MethodPost, "/instances/"+id+"/checkpoints", nil, body)
	if err != nil {
		return mcp.NewToolResultErrorFromErr("checkpoint create", err), nil
	}
	return toolResultJSON(out)
}

func (t *MicroDropletTool) checkpointList(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	args := req.GetArguments()
	query := url.Values{}
	query.Set("page", fmt.Sprintf("%d", intFromArg(args["page"], 1)))
	query.Set("per_page", fmt.Sprintf("%d", intFromArg(args["per_page"], 25)))
	if v, _ := args["micro_droplet_id"].(string); v != "" {
		query.Set("micro_droplet_id", v)
	}

	api, err := t.api(ctx)
	if err != nil {
		return nil, err
	}

	out, err := api.do(ctx, http.MethodGet, "/checkpoints", query, nil)
	if err != nil {
		return mcp.NewToolResultErrorFromErr("checkpoint list", err), nil
	}
	return toolResultJSON(out)
}

func (t *MicroDropletTool) checkpointGet(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	id, errMsg := requiredUUID(req.GetArguments(), "id")
	if errMsg != "" {
		return mcp.NewToolResultError(errMsg), nil
	}

	api, err := t.api(ctx)
	if err != nil {
		return nil, err
	}

	out, err := api.do(ctx, http.MethodGet, "/checkpoints/"+id, nil, nil)
	if err != nil {
		return mcp.NewToolResultErrorFromErr("checkpoint get", err), nil
	}
	return toolResultJSON(out)
}

func (t *MicroDropletTool) checkpointDelete(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	id, errMsg := requiredUUID(req.GetArguments(), "id")
	if errMsg != "" {
		return mcp.NewToolResultError(errMsg), nil
	}

	api, err := t.api(ctx)
	if err != nil {
		return nil, err
	}

	if err := api.doNoContent(ctx, http.MethodDelete, "/checkpoints/"+id); err != nil {
		return mcp.NewToolResultErrorFromErr("checkpoint delete", err), nil
	}
	return mcp.NewToolResultText("Checkpoint deleted successfully"), nil
}

func buildCreateBody(args map[string]any) (map[string]any, error) {
	image, hasImage := args["image"].(string)
	checkpoint, hasCheckpoint := args["checkpoint"].(string)
	container, hasContainer, err := parseContainer(args["container"])
	if err != nil {
		return nil, err
	}

	hasImage = hasImage && strings.TrimSpace(image) != ""
	hasCheckpoint = hasCheckpoint && strings.TrimSpace(checkpoint) != ""

	switch {
	case !hasImage && !hasCheckpoint && !hasContainer:
		return nil, fmt.Errorf("exactly one of image, checkpoint, or container must be provided")
	case hasImage && hasCheckpoint, hasImage && hasContainer, hasCheckpoint && hasContainer:
		return nil, fmt.Errorf("exactly one of image, checkpoint, or container must be provided, not more than one")
	}

	body := map[string]any{
		"name":   args["name"],
		"region": args["region"],
		"size":   args["size"],
	}

	if hasImage {
		body["image"] = image
	}
	if hasCheckpoint {
		body["checkpoint"] = checkpoint
	}
	if hasContainer {
		body["container"] = container
	}

	copyOptionalString(body, args, "networking")
	copyOptionalString(body, args, "vpc_uuid")
	copyOptionalBool(body, args, "auto_resume")
	copyOptionalNumber(body, args, "http_port")
	copyOptionalString(body, args, "http_protocol")

	if v, ok := args["auto_pause"]; ok && v != nil {
		body["auto_pause"] = v
	}
	if v, ok := args["environment"]; ok && v != nil {
		body["environment"] = v
	}
	if tags := stringSliceArg(args["tags"]); len(tags) > 0 {
		body["tags"] = tags
	}

	return body, nil
}

func copyOptionalString(body, args map[string]any, key string) {
	if v, ok := args[key].(string); ok && v != "" {
		body[key] = v
	}
}

func copyOptionalBool(body, args map[string]any, key string) {
	if v, ok := args[key].(bool); ok {
		body[key] = v
	}
}

func copyOptionalNumber(body, args map[string]any, key string) {
	if v, ok := args[key].(float64); ok {
		body[key] = int(v)
	}
}

// parseContainer treats nil, {}, and missing/blank image as omitted so create
// fails with the same "exactly one of ..." message as a missing boot source.
func parseContainer(raw any) (map[string]any, bool, error) {
	if raw == nil {
		return nil, false, nil
	}
	container, ok := raw.(map[string]any)
	if !ok {
		return nil, false, fmt.Errorf("container must be an object with an image field")
	}
	image, _ := container["image"].(string)
	if strings.TrimSpace(image) == "" {
		return nil, false, nil
	}
	return container, true, nil
}

func requiredUUID(args map[string]any, key string) (string, string) {
	id, _ := args[key].(string)
	id = strings.TrimSpace(id)
	if id == "" {
		return "", key + " is required"
	}
	return id, ""
}

func toolResultJSON(raw json.RawMessage) (*mcp.CallToolResult, error) {
	if len(raw) == 0 {
		return mcp.NewToolResultText("{}"), nil
	}
	var pretty json.RawMessage
	if err := json.Unmarshal(raw, &pretty); err != nil {
		return mcp.NewToolResultText(string(raw)), nil
	}
	b, err := json.MarshalIndent(pretty, "", "  ")
	if err != nil {
		return nil, fmt.Errorf("marshal response: %w", err)
	}
	return mcp.NewToolResultText(string(b)), nil
}

func intFromArg(v any, def int) int {
	switch x := v.(type) {
	case nil:
		return def
	case float64:
		if x >= 1 {
			return int(x)
		}
		return def
	case int:
		if x >= 1 {
			return x
		}
		return def
	default:
		return def
	}
}

func stringSliceArg(v any) []string {
	switch x := v.(type) {
	case nil:
		return nil
	case []string:
		return x
	case []any:
		var out []string
		for _, el := range x {
			if s, ok := el.(string); ok {
				out = append(out, s)
			}
		}
		return out
	default:
		return nil
	}
}

// Tools registers MCP tools for MicroDroplet lifecycle management.
func (t *MicroDropletTool) Tools() []server.ServerTool {
	return []server.ServerTool{
		{
			Handler: t.create,
			Tool: mcp.NewTool("microdroplet-create",
				common.WithHints(common.HintsCreate),
				common.WithRisk(common.RiskHigh),
				mcp.WithDescription("Create a MicroDroplet. Provide exactly one of image, checkpoint, or container as the boot source."),
				mcp.WithString("name", mcp.Required(), mcp.Description("Human-readable name.")),
				mcp.WithString("region", mcp.Required(), mcp.Description("Region slug (e.g. nyc1).")),
				mcp.WithString("size", mcp.Required(), mcp.Description("Size slug (e.g. mv-2vcpu-4gb).")),
				mcp.WithString("image", mcp.Description("Imported image UUID or do:microdroplet_image URN. Mutually exclusive with checkpoint and container.")),
				mcp.WithString("checkpoint", mcp.Description("Checkpoint UUID to restore from. Mutually exclusive with image and container.")),
				mcp.WithObject("container", mcp.Description("Golden-mode OCI workload ref."),
					mcp.Properties(map[string]any{
						"image": map[string]any{
							"type":        "string",
							"description": "OCI reference (public Hub or team DOCR).",
						},
					})),
				mcp.WithString("networking", mcp.Enum("public", "vpc"), mcp.Description("Network posture: public or vpc.")),
				mcp.WithString("vpc_uuid", mcp.Description("Required when networking is vpc.")),
				mcp.WithObject("auto_pause", mcp.Description("Auto-pause config: enabled (bool), idle_timeout (duration string).")),
				mcp.WithBoolean("auto_resume", mcp.Description("Whether to auto-resume on HTTP traffic (default true).")),
				mcp.WithNumber("http_port", mcp.Description("HTTP listen port for ingress and auto-resume probing.")),
				mcp.WithString("http_protocol", mcp.Enum("http", "http2"), mcp.Description("Application protocol on http_port: http or http2.")),
				mcp.WithObject("environment", mcp.Description("Key-value environment variables injected at boot.")),
				mcp.WithArray("tags", mcp.Description("Resource tags."), mcp.Items(map[string]any{"type": "string"})),
			),
		},
		{
			Handler: t.list,
			Tool: mcp.NewTool("microdroplet-list",
				common.WithHints(common.HintsRead),
				common.WithRisk(common.RiskLow),
				mcp.WithDescription("List MicroDroplets for the authenticated team."),
				mcp.WithNumber("page", mcp.DefaultNumber(1), mcp.Description("Page number (default 1).")),
				mcp.WithNumber("per_page", mcp.DefaultNumber(25), mcp.Description("Items per page (default 25, max 200).")),
				mcp.WithString("region", mcp.Description("Filter by region slug.")),
				mcp.WithString("name", mcp.Description("Filter by name.")),
				mcp.WithString("tag_name", mcp.Description("Filter by resource tag name.")),
			),
		},
		{
			Handler: t.get,
			Tool: mcp.NewTool("microdroplet-get",
				common.WithHints(common.HintsRead),
				common.WithRisk(common.RiskLow),
				mcp.WithDescription("Get a MicroDroplet by ID."),
				mcp.WithString("id", mcp.Required(), mcp.Description("MicroDroplet UUID.")),
			),
		},
		{
			Handler: t.delete,
			Tool: mcp.NewTool("microdroplet-delete",
				common.WithHints(common.HintsDelete),
				common.WithRisk(common.RiskHigh),
				mcp.WithDescription("Delete a MicroDroplet. Irreversible; checkpoints are retained until deleted separately."),
				mcp.WithString("id", mcp.Required(), mcp.Description("MicroDroplet UUID.")),
			),
		},
		{
			Handler: t.pause,
			Tool: mcp.NewTool("microdroplet-pause",
				common.WithHints(common.HintsToggle),
				common.WithRisk(common.RiskMedium),
				mcp.WithDescription("Pause a running MicroDroplet (synchronous, idempotent)."),
				mcp.WithString("id", mcp.Required(), mcp.Description("MicroDroplet UUID.")),
			),
		},
		{
			Handler: t.resume,
			Tool: mcp.NewTool("microdroplet-resume",
				common.WithHints(common.HintsToggle),
				common.WithRisk(common.RiskMedium),
				mcp.WithDescription("Resume a paused MicroDroplet (synchronous, idempotent)."),
				mcp.WithString("id", mcp.Required(), mcp.Description("MicroDroplet UUID.")),
			),
		},
		{
			Handler: t.checkpointCreate,
			Tool: mcp.NewTool("microdroplet-checkpoint-create",
				common.WithHints(common.HintsCreate),
				common.WithRisk(common.RiskMedium),
				mcp.WithDescription("Start async checkpoint of a running MicroDroplet without pausing. Poll checkpoint-get until available or failed."),
				mcp.WithString("id", mcp.Required(), mcp.Description("MicroDroplet UUID.")),
				mcp.WithString("name", mcp.Description("Optional checkpoint name.")),
			),
		},
		{
			Handler: t.checkpointList,
			Tool: mcp.NewTool("microdroplet-checkpoint-list",
				common.WithHints(common.HintsRead),
				common.WithRisk(common.RiskLow),
				mcp.WithDescription("List team checkpoints, newest first."),
				mcp.WithNumber("page", mcp.DefaultNumber(1), mcp.Description("Page number (default 1).")),
				mcp.WithNumber("per_page", mcp.DefaultNumber(25), mcp.Description("Items per page (default 25, max 200).")),
				mcp.WithString("micro_droplet_id", mcp.Description("Filter by source MicroDroplet UUID.")),
			),
		},
		{
			Handler: t.checkpointGet,
			Tool: mcp.NewTool("microdroplet-checkpoint-get",
				common.WithHints(common.HintsRead),
				common.WithRisk(common.RiskLow),
				mcp.WithDescription("Get a checkpoint by ID."),
				mcp.WithString("id", mcp.Required(), mcp.Description("Checkpoint UUID.")),
			),
		},
		{
			Handler: t.checkpointDelete,
			Tool: mcp.NewTool("microdroplet-checkpoint-delete",
				common.WithHints(common.HintsDelete),
				common.WithRisk(common.RiskHigh),
				mcp.WithDescription("Delete a checkpoint. Irreversible."),
				mcp.WithString("id", mcp.Required(), mcp.Description("Checkpoint UUID.")),
			),
		},
	}
}
