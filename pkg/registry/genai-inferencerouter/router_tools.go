package genaiinferencerouter

import (
	"context"
	"encoding/json"
	"fmt"
	"strconv"
	"strings"

	"github.com/digitalocean/godo"
	"github.com/mark3labs/mcp-go/mcp"
	"github.com/mark3labs/mcp-go/server"

	"mcp-digitalocean/pkg/registry/common"
)

// RouterTool exposes GenAI inference router operations via godo.GradientAI.
type RouterTool struct {
	client func(ctx context.Context) (*godo.Client, error)
}

// NewRouterTool builds a RouterTool that uses the shared godo client.
func NewRouterTool(client func(ctx context.Context) (*godo.Client, error)) *RouterTool {
	return &RouterTool{client: client}
}

// validatePoliciesJSON ensures PoliciesJson decodes to a JSON array (empty allowed; API permits no policies).
func validatePoliciesJSON(raw json.RawMessage) error {
	var policies []json.RawMessage
	if err := json.Unmarshal(raw, &policies); err != nil {
		return err
	}
	return nil
}

func (t *RouterTool) create(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	args := req.GetArguments()
	name, _ := args["Name"].(string)
	name = strings.TrimSpace(name)
	if name == "" {
		return mcp.NewToolResultError("Name is required"), nil
	}
	policiesJSON, _ := args["PoliciesJson"].(string)
	policiesJSON = strings.TrimSpace(policiesJSON)

	fallbacks := stringSliceArg(args["FallbackModels"])
	if len(fallbacks) == 0 {
		return mcp.NewToolResultError("FallbackModels is required: provide at least one fallback model id (the API requires fallbacks for inference routers)"), nil
	}

	c, err := t.client(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to get DigitalOcean client: %w", err)
	}

	create := &godo.InferenceRouterCreateRequest{
		Name:           name,
		FallbackModels: fallbacks,
	}
	if policiesJSON != "" {
		raw := json.RawMessage(policiesJSON)
		if err := validatePoliciesJSON(raw); err != nil {
			return mcp.NewToolResultError(fmt.Sprintf("invalid PoliciesJson: %v", err)), nil
		}
		create.Policies = raw
	}

	router, _, err := c.GradientAI.CreateInferenceRouter(ctx, create)
	if err != nil {
		return mcp.NewToolResultErrorFromErr("CreateInferenceRouter failed", err), nil
	}

	return encodeJSON(map[string]any{"model_router": router})
}

func (t *RouterTool) list(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	page := intFromArg(req.GetArguments()["Page"], 1)
	perPage := intFromArg(req.GetArguments()["PerPage"], 1000)
	if page < 1 {
		page = 1
	}
	if perPage < 1 {
		perPage = 20
	}
	if perPage > 1000 {
		perPage = 1000
	}

	c, err := t.client(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to get DigitalOcean client: %w", err)
	}

	routers, apiResp, err := c.GradientAI.ListInferenceRouters(ctx, &godo.ListOptions{Page: page, PerPage: perPage})
	if err != nil {
		return mcp.NewToolResultErrorFromErr("ListInferenceRouters failed", err), nil
	}
	if routers == nil {
		routers = []*godo.InferenceRouterSummary{}
	}

	type metaView struct {
		Page  int `json:"page"`
		Pages int `json:"pages"`
		Total int `json:"total"`
	}
	payload := struct {
		ModelRouters []*godo.InferenceRouterSummary `json:"model_routers"`
		Meta         *metaView                      `json:"meta,omitempty"`
		Links        *godo.Links                    `json:"links,omitempty"`
	}{
		ModelRouters: routers,
	}
	if apiResp != nil {
		if apiResp.Meta != nil {
			payload.Meta = &metaView{
				Page:  apiResp.Meta.Page,
				Pages: apiResp.Meta.Pages,
				Total: apiResp.Meta.Total,
			}
		}
		if apiResp.Links != nil {
			payload.Links = apiResp.Links
		}
	}

	return encodeJSON(payload)
}

func (t *RouterTool) listTaskPresets(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	page := intFromArg(req.GetArguments()["Page"], 1)
	perPage := intFromArg(req.GetArguments()["PerPage"], 1000)
	if page < 1 {
		page = 1
	}
	if perPage < 1 {
		perPage = 20
	}
	if perPage > 1000 {
		perPage = 1000
	}

	c, err := t.client(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to get DigitalOcean client: %w", err)
	}

	tasks, apiResp, err := c.GradientAI.ListInferenceRouterTaskPresets(ctx, &godo.ListOptions{Page: page, PerPage: perPage})
	if err != nil {
		return mcp.NewToolResultErrorFromErr("ListInferenceRouterTaskPresets failed", err), nil
	}
	if tasks == nil {
		tasks = []*godo.InferenceRouterTaskPreset{}
	}

	type metaView struct {
		Page  int `json:"page"`
		Pages int `json:"pages"`
		Total int `json:"total"`
	}
	payload := struct {
		Tasks []*godo.InferenceRouterTaskPreset `json:"tasks"`
		Meta  *metaView                         `json:"meta,omitempty"`
		Links *godo.Links                       `json:"links,omitempty"`
	}{
		Tasks: tasks,
	}
	if apiResp != nil {
		if apiResp.Meta != nil {
			payload.Meta = &metaView{
				Page:  apiResp.Meta.Page,
				Pages: apiResp.Meta.Pages,
				Total: apiResp.Meta.Total,
			}
		}
		if apiResp.Links != nil {
			payload.Links = apiResp.Links
		}
	}

	return encodeJSON(payload)
}

func (t *RouterTool) update(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	args := req.GetArguments()
	uuid, _ := args["UUID"].(string)
	uuid = strings.TrimSpace(uuid)
	if uuid == "" {
		return mcp.NewToolResultError("UUID is required"), nil
	}

	update := &godo.InferenceRouterUpdateRequest{}
	if name, _ := args["Name"].(string); strings.TrimSpace(name) != "" {
		update.Name = strings.TrimSpace(name)
	}
	if desc, _ := args["Description"].(string); strings.TrimSpace(desc) != "" {
		update.Description = strings.TrimSpace(desc)
	}

	if policiesJSON, ok := args["PoliciesJson"].(string); ok {
		policiesJSON = strings.TrimSpace(policiesJSON)
		if policiesJSON != "" {
			raw := json.RawMessage(policiesJSON)
			if err := validatePoliciesJSON(raw); err != nil {
				return mcp.NewToolResultError(fmt.Sprintf("invalid PoliciesJson: %v", err)), nil
			}
			update.Policies = &raw
		}
	}

	if _, hasFallbacks := args["FallbackModels"]; hasFallbacks {
		update.FallbackModels = stringSliceArg(args["FallbackModels"])
	}

	if update.Name == "" && update.Description == "" && update.Policies == nil && len(update.FallbackModels) == 0 {
		return mcp.NewToolResultError("at least one of Name, Description, PoliciesJson (non-empty JSON array), or FallbackModels must be provided (matches godo UpdateInferenceRouter validation)"), nil
	}

	c, err := t.client(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to get DigitalOcean client: %w", err)
	}

	router, _, err := c.GradientAI.UpdateInferenceRouter(ctx, uuid, update)
	if err != nil {
		return mcp.NewToolResultErrorFromErr("UpdateInferenceRouter failed", err), nil
	}

	return encodeJSON(map[string]any{"model_router": router})
}

func (t *RouterTool) get(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	uuid, _ := req.GetArguments()["UUID"].(string)
	uuid = strings.TrimSpace(uuid)
	if uuid == "" {
		return mcp.NewToolResultError("UUID is required"), nil
	}

	c, err := t.client(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to get DigitalOcean client: %w", err)
	}

	router, _, err := c.GradientAI.GetInferenceRouter(ctx, uuid)
	if err != nil {
		return mcp.NewToolResultErrorFromErr("GetInferenceRouter failed", err), nil
	}

	return encodeJSON(map[string]any{"model_router": router})
}

func (t *RouterTool) delete(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	uuid, _ := req.GetArguments()["UUID"].(string)
	uuid = strings.TrimSpace(uuid)
	if uuid == "" {
		return mcp.NewToolResultError("UUID is required"), nil
	}

	c, err := t.client(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to get DigitalOcean client: %w", err)
	}

	out, _, err := c.GradientAI.DeleteInferenceRouter(ctx, uuid)
	if err != nil {
		return mcp.NewToolResultErrorFromErr("DeleteInferenceRouter failed", err), nil
	}

	if out == nil {
		b, _ := json.MarshalIndent(map[string]string{"uuid": uuid}, "", "  ")
		return mcp.NewToolResultText(string(b)), nil
	}

	return encodeJSON(out)
}

func encodeJSON(v any) (*mcp.CallToolResult, error) {
	b, err := json.MarshalIndent(v, "", "  ")
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
	case string:
		if n, err := strconv.Atoi(x); err == nil && n >= 1 {
			return n
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

// Tools registers MCP tools for model routers.
func (t *RouterTool) Tools() []server.ServerTool {
	return []server.ServerTool{
		{
			Handler: t.create,
			Tool: mcp.NewTool(
				"genai-inference-router-create",
				common.WithHints(common.HintsCreate),
				common.WithRisk(common.RiskLow),
				mcp.WithDescription(`Create a GenAI model router (godo.GradientAI.CreateInferenceRouter). JSON body: "name", optional "policies" array, and required "fallback_models" (at least one). Each policy needs a task: either "task_slug" plus "models" and "selection_policy":{"prefer":"fastest"|"cheapest"}, or "custom_task":{"name","description"} with "models" and selection_policy. Flat {"model","usecase_class"} policies fail with "task is required". List/get return the same policy shape under model_router.config.`),
				mcp.WithString("Name", mcp.Required(), mcp.Description("Router name")),
				mcp.WithString("PoliciesJson", mcp.Description(`JSON array for "policies". Example: [{"task_slug":"code-generation","models":["openai-gpt-5"],"selection_policy":{"prefer":"fastest"}}]. Custom task: use custom_task with name+description instead of task_slug. Omit or "[]" if allowed.`)),
				mcp.WithArray("FallbackModels", mcp.Required(), mcp.MinItems(1), mcp.Description("At least one fallback model id, in order, sent as fallback_models (required by the API).")),
			),
		},
		{
			Handler: t.list,
			Tool: mcp.NewTool(
				"genai-inference-router-list",
				common.WithHints(common.HintsRead),
				common.WithRisk(common.RiskLow),
				mcp.WithDescription("List GenAI model routers (godo.GradientAI.ListInferenceRouters) with pagination."),
				mcp.WithNumber("Page", mcp.DefaultNumber(1), mcp.Description("Page number (default 1)")),
				mcp.WithNumber("PerPage", mcp.DefaultNumber(1000), mcp.Description("Items per page (default 1000, max 1000)")),
			),
		},
		{
			Handler: t.get,
			Tool: mcp.NewTool(
				"genai-inference-router-get",
				common.WithHints(common.HintsRead),
				common.WithRisk(common.RiskLow),
				mcp.WithDescription("Get a GenAI model router by UUID (godo.GradientAI.GetInferenceRouter)."),
				mcp.WithString("UUID", mcp.Required(), mcp.Description("Model router UUID")),
			),
		},
		{
			Handler: t.delete,
			Tool: mcp.NewTool(
				"genai-inference-router-delete",
				common.WithHints(common.HintsDelete),
				common.WithRisk(common.RiskHigh),
				mcp.WithDescription("Delete a GenAI model router by UUID (godo.GradientAI.DeleteInferenceRouter)."),
				mcp.WithString("UUID", mcp.Required(), mcp.Description("Model router UUID")),
			),
		},
		{
			Handler: t.listTaskPresets,
			Tool: mcp.NewTool(
				"genai-inference-router-task-presets",
				common.WithHints(common.HintsRead),
				common.WithRisk(common.RiskLow),
				mcp.WithDescription("List preset inference-router tasks (task_slug, name, models, etc.) from GET /v2/gen-ai/models/routers/tasks/presets via godo.GradientAI.ListInferenceRouterTaskPresets. Use task_slug values when building PoliciesJson for create/update."),
				mcp.WithNumber("Page", mcp.DefaultNumber(1), mcp.Description("Page number (default 1)")),
				mcp.WithNumber("PerPage", mcp.DefaultNumber(1000), mcp.Description("Items per page (default 1000, max 1000)")),
			),
		},
		{
			Handler: t.update,
			Tool: mcp.NewTool(
				"genai-inference-router-update",
				common.WithHints(common.HintsToggle),
				// Reconfiguring a router that is actively serving traffic can
				// incur cost / affect live routing, so risk is medium.
				common.WithRisk(common.RiskMedium),
				mcp.WithDescription(`Update a GenAI model router (godo.GradientAI.UpdateInferenceRouter, PUT). At least one of Name, Description, PoliciesJson (non-empty), or FallbackModels must be supplied. PoliciesJson must be a JSON array (same rules as create). Omit fields you do not want to change.`),
				mcp.WithString("UUID", mcp.Required(), mcp.Description("Model router UUID")),
				mcp.WithString("Name", mcp.Description("New router name (optional)")),
				mcp.WithString("Description", mcp.Description("New description (optional)")),
				mcp.WithString("PoliciesJson", mcp.Description(`JSON array for "policies" (optional). Same shape as create; set to "[]" only if the API should receive an empty policies list.`)),
				mcp.WithArray("FallbackModels", mcp.Description("Optional ordered fallback model ids; include this argument only when updating fallbacks.")),
			),
		},
	}
}
