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

	if strings.TrimSpace(argString(args["name"])) == "" {
		return mcp.NewToolResultError("name is required"), nil
	}

	body, err := buildCreateBody(args)
	if err != nil {
		return mcp.NewToolResultError(err.Error()), nil
	}

	api, err := t.api(ctx)
	if err != nil {
		return nil, err
	}

	out, err := api.do(ctx, http.MethodPost, "", nil, body)
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

	out, err := api.do(ctx, http.MethodGet, "", query, nil)
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

	out, err := api.do(ctx, http.MethodGet, "/"+id, nil, nil)
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

	if err := api.doNoContent(ctx, http.MethodDelete, "/"+id); err != nil {
		return mcp.NewToolResultErrorFromErr("microdroplet delete", err), nil
	}
	return mcp.NewToolResultText("MicroVM deleted successfully"), nil
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

	out, err := api.do(ctx, http.MethodPost, "/"+id+"/pause", nil, nil)
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

	out, err := api.do(ctx, http.MethodPost, "/"+id+"/resume", nil, nil)
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

	out, err := api.do(ctx, http.MethodPost, "/"+id+"/checkpoints", nil, body)
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
	if v, _ := args["microvm_id"].(string); v != "" {
		query.Set("microvm_id", v)
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
	source, err := parseSource(args["source"])
	if err != nil {
		return nil, err
	}

	body := map[string]any{
		"name":   args["name"],
		"source": source,
	}

	copyOptionalString(body, args, "region")
	copyOptionalString(body, args, "networking")
	copyOptionalString(body, args, "vpc_uuid")
	copyOptionalBool(body, args, "auto_resume")
	copyOptionalNumber(body, args, "http_port")
	copyOptionalString(body, args, "http_protocol")

	size, hasSize, err := parseSize(args["size"])
	if err != nil {
		return nil, err
	}
	if hasSize {
		body["size"] = size
	}

	_, hasOCI := source["oci_ref"]
	if hasOCI {
		if _, ok := body["region"]; !ok {
			return nil, fmt.Errorf("region is required when source.oci_ref is set")
		}
		if !hasSize {
			return nil, fmt.Errorf("size is required when source.oci_ref is set")
		}
	}

	if v, ok := args["auto_pause"]; ok && v != nil {
		body["auto_pause"] = v
	}
	if v, ok := args["environment"]; ok && v != nil {
		body["environment"] = v
	}
	if ports := numberSliceArg(args["ports"]); len(ports) > 0 {
		body["ports"] = ports
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

func parseSource(raw any) (map[string]any, error) {
	source, ok := raw.(map[string]any)
	if !ok {
		return nil, fmt.Errorf("source is required")
	}
	ociRef := strings.TrimSpace(argString(source["oci_ref"]))
	checkpointID := strings.TrimSpace(argString(source["checkpoint_id"]))
	switch {
	case ociRef == "" && checkpointID == "":
		return nil, fmt.Errorf("source must have exactly one of oci_ref or checkpoint_id")
	case ociRef != "" && checkpointID != "":
		return nil, fmt.Errorf("source must have exactly one of oci_ref or checkpoint_id, not both")
	}
	if ociRef != "" {
		return map[string]any{"oci_ref": ociRef}, nil
	}
	return map[string]any{"checkpoint_id": checkpointID}, nil
}

func parseSize(raw any) (map[string]any, bool, error) {
	if raw == nil {
		return nil, false, nil
	}
	size, ok := raw.(map[string]any)
	if !ok {
		return nil, false, fmt.Errorf("size must be an object with cpu and memory")
	}
	cpu, hasCPU := intArg(size["cpu"])
	memory, hasMemory := intArg(size["memory"])
	if !hasCPU && !hasMemory {
		return nil, false, nil
	}
	if !hasCPU || !hasMemory {
		return nil, false, fmt.Errorf("size must include cpu and memory")
	}
	return map[string]any{"cpu": cpu, "memory": memory}, true, nil
}

func argString(v any) string {
	s, _ := v.(string)
	return s
}

func intArg(v any) (int, bool) {
	switch x := v.(type) {
	case float64:
		return int(x), true
	case int:
		return x, true
	default:
		return 0, false
	}
}

func numberSliceArg(v any) []int {
	switch x := v.(type) {
	case nil:
		return nil
	case []int:
		return x
	case []any:
		var out []int
		for _, el := range x {
			n, ok := intArg(el)
			if ok {
				out = append(out, n)
			}
		}
		return out
	default:
		return nil
	}
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

// Tools registers MCP tools for MicroVM lifecycle management.
func (t *MicroDropletTool) Tools() []server.ServerTool {
	return []server.ServerTool{
		{
			Handler: t.create,
			Tool: mcp.NewTool("microdroplet-create",
				common.WithHints(common.HintsCreate),
				common.WithRisk(common.RiskHigh),
				mcp.WithDescription("Create a MicroVM. source must have exactly one of oci_ref or checkpoint_id. region and size are required for oci_ref; both are inherited from the checkpoint when omitted."),
				mcp.WithString("name", mcp.Required(), mcp.Description("Human-readable name.")),
				mcp.WithObject("source", mcp.Required(), mcp.Description("Workload source. Exactly one of oci_ref or checkpoint_id."),
					mcp.Properties(map[string]any{
						"oci_ref": map[string]any{
							"type":        "string",
							"description": "OCI reference (public Hub or team DOCR), pulled at boot.",
						},
						"checkpoint_id": map[string]any{
							"type":        "string",
							"description": "Checkpoint UUID to restore. Mutually exclusive with oci_ref.",
						},
					})),
				mcp.WithString("region", mcp.Description("Region slug (e.g. nyc1). Required with oci_ref.")),
				mcp.WithObject("size", mcp.Description("Compute capacity. Required with oci_ref; must match the checkpoint if supplied on restore."),
					mcp.Properties(map[string]any{
						"cpu":    map[string]any{"type": "integer", "description": "vCPU count."},
						"memory": map[string]any{"type": "integer", "description": "Memory in MiB."},
					})),
				mcp.WithString("networking", mcp.Enum("public", "vpc"), mcp.Description("Network posture: public or vpc.")),
				mcp.WithString("vpc_uuid", mcp.Description("Required when networking is vpc.")),
				mcp.WithObject("auto_pause", mcp.Description("Auto-pause config: enabled (bool), idle_timeout (duration string).")),
				mcp.WithBoolean("auto_resume", mcp.Description("Whether to auto-resume on HTTP traffic (default true).")),
				mcp.WithNumber("http_port", mcp.Description("HTTP listen port for ingress and auto-resume probing.")),
				mcp.WithString("http_protocol", mcp.Enum("http", "http2"), mcp.Description("Application protocol on http_port: http or http2.")),
				mcp.WithArray("ports", mcp.Description("Guest ports to open for ingress (max 5). Must include http_port when set."), mcp.Items(map[string]any{"type": "integer"})),
				mcp.WithObject("environment", mcp.Description("Key-value environment variables injected at boot.")),
				mcp.WithArray("tags", mcp.Description("Resource tags."), mcp.Items(map[string]any{"type": "string"})),
			),
		},
		{
			Handler: t.list,
			Tool: mcp.NewTool("microdroplet-list",
				common.WithHints(common.HintsRead),
				common.WithRisk(common.RiskLow),
				mcp.WithDescription("List MicroVMs for the authenticated team."),
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
				mcp.WithDescription("Get a MicroVM by ID."),
				mcp.WithString("id", mcp.Required(), mcp.Description("MicroVM UUID.")),
			),
		},
		{
			Handler: t.delete,
			Tool: mcp.NewTool("microdroplet-delete",
				common.WithHints(common.HintsDelete),
				common.WithRisk(common.RiskHigh),
				mcp.WithDescription("Delete a MicroVM. Irreversible; checkpoints are retained until deleted separately."),
				mcp.WithString("id", mcp.Required(), mcp.Description("MicroVM UUID.")),
			),
		},
		{
			Handler: t.pause,
			Tool: mcp.NewTool("microdroplet-pause",
				common.WithHints(common.HintsToggle),
				common.WithRisk(common.RiskMedium),
				mcp.WithDescription("Pause a running MicroVM (synchronous, idempotent)."),
				mcp.WithString("id", mcp.Required(), mcp.Description("MicroVM UUID.")),
			),
		},
		{
			Handler: t.resume,
			Tool: mcp.NewTool("microdroplet-resume",
				common.WithHints(common.HintsToggle),
				common.WithRisk(common.RiskMedium),
				mcp.WithDescription("Resume a paused MicroVM (synchronous, idempotent)."),
				mcp.WithString("id", mcp.Required(), mcp.Description("MicroVM UUID.")),
			),
		},
		{
			Handler: t.checkpointCreate,
			Tool: mcp.NewTool("microdroplet-checkpoint-create",
				common.WithHints(common.HintsCreate),
				common.WithRisk(common.RiskMedium),
				mcp.WithDescription("Start async checkpoint of a running or paused MicroVM. Poll checkpoint-get until available or failed."),
				mcp.WithString("id", mcp.Required(), mcp.Description("MicroVM UUID.")),
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
				mcp.WithString("microvm_id", mcp.Description("Filter by source MicroVM UUID.")),
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
