package dedicatedinference

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/digitalocean/godo"
	"github.com/mark3labs/mcp-go/mcp"
	"github.com/mark3labs/mcp-go/server"

	"mcp-digitalocean/pkg/registry/common"
)

// DedicatedInferenceTool provides Dedicated Inference lifecycle management tools.
type DedicatedInferenceTool struct {
	client func(ctx context.Context) (*godo.Client, error)
}

type listResponse struct {
	Items []godo.DedicatedInferenceListItem `json:"items"`
	Meta  *godo.Meta                        `json:"meta,omitempty"`
}

// NewDedicatedInferenceTool creates a new DedicatedInferenceTool instance.
func NewDedicatedInferenceTool(client func(ctx context.Context) (*godo.Client, error)) *DedicatedInferenceTool {
	return &DedicatedInferenceTool{
		client: client,
	}
}

func (d *DedicatedInferenceTool) createDedicatedInference(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	client, err := d.client(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to get DigitalOcean client: %w", err)
	}

	args := req.GetArguments()

	name, _ := args["Name"].(string)
	if name == "" {
		return mcp.NewToolResultError("Name is required"), nil
	}

	region, _ := args["Region"].(string)
	if region == "" {
		return mcp.NewToolResultError("Region is required"), nil
	}

	spec := &godo.DedicatedInferenceSpecRequest{
		Name:   name,
		Region: region,
	}

	if v, ok := args["EnablePublicEndpoint"].(bool); ok {
		spec.EnablePublicEndpoint = v
	}

	if v, _ := args["VPCUUID"].(string); v != "" {
		spec.VPC = &godo.DedicatedInferenceVPCRequest{UUID: v}
	}

	spec.ModelDeployments = parseModelDeployments(args)
	if len(spec.ModelDeployments) == 0 {
		return mcp.NewToolResultError("ModelDeployments is required and must not be empty"), nil
	}

	spec.Version = 1

	createReq := &godo.DedicatedInferenceCreateRequest{
		Spec: spec,
	}

	if token, _ := args["HuggingFaceToken"].(string); token != "" {
		createReq.Secrets = &godo.DedicatedInferenceSecrets{
			HuggingFaceToken: token,
		}
	}

	di, authToken, _, err := client.DedicatedInference.Create(ctx, createReq)
	if err != nil {
		return mcp.NewToolResultErrorFromErr("Failed to create dedicated inference", err), nil
	}

	type createResponse struct {
		DedicatedInference *godo.DedicatedInference      `json:"dedicated_inference"`
		Token              *godo.DedicatedInferenceToken `json:"token,omitempty"`
	}

	return marshalResult(createResponse{
		DedicatedInference: di,
		Token:              authToken,
	})
}

func (d *DedicatedInferenceTool) getDedicatedInference(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	client, err := d.client(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to get DigitalOcean client: %w", err)
	}

	id, _ := req.GetArguments()["DedicatedInferenceID"].(string)
	if id == "" {
		return mcp.NewToolResultError("DedicatedInferenceID is required"), nil
	}

	di, _, err := client.DedicatedInference.Get(ctx, id)
	if err != nil {
		return mcp.NewToolResultErrorFromErr("Failed to get dedicated inference", err), nil
	}

	return marshalResult(di)
}

func (d *DedicatedInferenceTool) listDedicatedInferences(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	client, err := d.client(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to get DigitalOcean client: %w", err)
	}

	args := req.GetArguments()

	opts := &godo.DedicatedInferenceListOptions{}
	if v, _ := args["Region"].(string); v != "" {
		opts.Region = v
	}
	if v, _ := args["Name"].(string); v != "" {
		opts.Name = v
	}
	if v, ok := args["Page"].(float64); ok {
		opts.Page = int(v)
	}
	if v, ok := args["PerPage"].(float64); ok {
		opts.PerPage = int(v)
	}

	items, resp, err := client.DedicatedInference.List(ctx, opts)
	if err != nil {
		return mcp.NewToolResultErrorFromErr("Failed to list dedicated inferences", err), nil
	}

	result := listResponse{
		Items: items,
	}
	if resp != nil && resp.Meta != nil {
		result.Meta = resp.Meta
	}

	return marshalResult(result)
}

func (d *DedicatedInferenceTool) updateDedicatedInference(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	client, err := d.client(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to get DigitalOcean client: %w", err)
	}

	args := req.GetArguments()

	id, _ := args["DedicatedInferenceID"].(string)
	if id == "" {
		return mcp.NewToolResultError("DedicatedInferenceID is required"), nil
	}

	spec := &godo.DedicatedInferenceSpecRequest{}
	if v, _ := args["Name"].(string); v != "" {
		spec.Name = v
	}
	if v, _ := args["Region"].(string); v != "" {
		spec.Region = v
	}
	if v, ok := args["EnablePublicEndpoint"].(bool); ok {
		spec.EnablePublicEndpoint = v
	}
	if v, _ := args["VPCUUID"].(string); v != "" {
		spec.VPC = &godo.DedicatedInferenceVPCRequest{UUID: v}
	}

	spec.ModelDeployments = parseModelDeployments(args)
	if len(spec.ModelDeployments) == 0 {
		return mcp.NewToolResultError("ModelDeployments is required and must not be empty"), nil
	}

	updateReq := &godo.DedicatedInferenceUpdateRequest{
		Spec: spec,
	}

	if token, _ := args["HuggingFaceToken"].(string); token != "" {
		updateReq.Secrets = &godo.DedicatedInferenceSecrets{
			HuggingFaceToken: token,
		}
	}

	di, _, err := client.DedicatedInference.Update(ctx, id, updateReq)
	if err != nil {
		return mcp.NewToolResultErrorFromErr("Failed to update dedicated inference", err), nil
	}

	return marshalResult(di)
}

func (d *DedicatedInferenceTool) deleteDedicatedInference(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	client, err := d.client(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to get DigitalOcean client: %w", err)
	}

	id, _ := req.GetArguments()["DedicatedInferenceID"].(string)
	if id == "" {
		return mcp.NewToolResultError("DedicatedInferenceID is required"), nil
	}

	_, err = client.DedicatedInference.Delete(ctx, id)
	if err != nil {
		return mcp.NewToolResultErrorFromErr("Failed to delete dedicated inference", err), nil
	}

	return mcp.NewToolResultText(`{"status": "success", "message": "Dedicated inference instance deleted"}`), nil
}

// Tools returns the list of server tools for Dedicated Inference management.
func (d *DedicatedInferenceTool) Tools() []server.ServerTool {
	return []server.ServerTool{
		{
			Handler: d.createDedicatedInference,
			Tool: mcp.NewTool(
				"dedicated-inference-create",
				common.WithHints(common.HintsCreate),
				common.WithRisk(common.RiskHigh),
				mcp.WithDescription("Create a new Dedicated Inference instance (CreateDedicatedInferenceV2). See spec/dedicated-inference-create-schema.json for the HTTP/API-aligned request shape. Tool arguments use UpperCamelCase; returns instance and optional initial auth token."),
				mcp.WithString("Name", mcp.Required(), mcp.Description("Name of the dedicated inference instance")),
				mcp.WithString("Region", mcp.Required(), mcp.Description("Region slug for deployment (e.g. nyc2, tor1, atl1)")),
				mcp.WithBoolean("EnablePublicEndpoint", mcp.Description("Whether to enable a public endpoint for the instance")),
				mcp.WithString("VPCUUID", mcp.Description("UUID of the VPC to deploy into")),
				mcp.WithArray("ModelDeployments", mcp.Required(), mcp.Description("Model deployments to configure"),
					mcp.Items(modelDeploymentSchema([]string{"ModelSlug", "ModelProvider", "Accelerators"}))),
				mcp.WithString("HuggingFaceToken", mcp.Description("HuggingFace API token for gated models (write-only, never returned in responses)")),
			),
		},
		{
			Handler: d.getDedicatedInference,
			Tool: mcp.NewTool(
				"dedicated-inference-get",
				common.WithHints(common.HintsRead),
				common.WithRisk(common.RiskLow),
				mcp.WithDescription("Get details of a Dedicated Inference instance (GetDedicatedInferenceV2) by ID."),
				mcp.WithString("DedicatedInferenceID", mcp.Required(), mcp.Description("UUID of the dedicated inference instance")),
			),
		},
		{
			Handler: d.listDedicatedInferences,
			Tool: mcp.NewTool(
				"dedicated-inference-list",
				common.WithHints(common.HintsRead),
				common.WithRisk(common.RiskLow),
				mcp.WithDescription("List Dedicated Inference instances (ListDedicatedInferenceV2) with optional filters and pagination."),
				mcp.WithString("Region", mcp.Description("Filter by region slug (e.g. nyc2)")),
				mcp.WithString("Name", mcp.Description("Filter by instance name")),
				mcp.WithNumber("Page", mcp.Description("Page number for pagination")),
				mcp.WithNumber("PerPage", mcp.Description("Number of items per page")),
			),
		},
		{
			Handler: d.updateDedicatedInference,
			Tool: mcp.NewTool(
				"dedicated-inference-update",
				common.WithHints(common.HintsToggle),
				common.WithRisk(common.RiskMedium),
				mcp.WithDescription("Update a Dedicated Inference instance (UpdateDedicatedInferenceV2). See spec/dedicated-inference-update-schema.json for the HTTP/API-aligned body shape."),
				mcp.WithString("DedicatedInferenceID", mcp.Required(), mcp.Description("UUID of the dedicated inference instance to update")),
				mcp.WithString("Name", mcp.Description("New name for the instance")),
				mcp.WithString("Region", mcp.Description("New region slug")),
				mcp.WithBoolean("EnablePublicEndpoint", mcp.Description("Whether to enable a public endpoint")),
				mcp.WithString("VPCUUID", mcp.Description("UUID of the VPC to deploy into")),
				mcp.WithArray("ModelDeployments", mcp.Description("Updated model deployments"),
					mcp.Items(modelDeploymentSchema([]string{"ModelSlug", "ModelProvider", "ModelID", "Accelerators"}))),
				mcp.WithString("HuggingFaceToken", mcp.Description("HuggingFace API token for gated models (replaces existing if provided, preserved if omitted)")),
			),
		},
		{
			Handler: d.deleteDedicatedInference,
			Tool: mcp.NewTool(
				"dedicated-inference-delete",
				common.WithHints(common.HintsDelete),
				common.WithRisk(common.RiskHigh),
				mcp.WithDescription("Delete a Dedicated Inference instance (DeleteDedicatedInferenceV2)."),
				mcp.WithString("DedicatedInferenceID", mcp.Required(), mcp.Description("UUID of the dedicated inference instance to delete")),
			),
		},
	}
}

func parseModelDeployments(args map[string]any) []*godo.DedicatedInferenceModelRequest {
	deploymentsRaw, ok := args["ModelDeployments"].([]any)
	if !ok {
		return nil
	}

	var deployments []*godo.DedicatedInferenceModelRequest
	for _, depRaw := range deploymentsRaw {
		dep, ok := depRaw.(map[string]any)
		if !ok {
			continue
		}
		modelReq := &godo.DedicatedInferenceModelRequest{
			ModelSlug:     stringFromMap(dep, "ModelSlug"),
			ModelProvider: stringFromMap(dep, "ModelProvider"),
		}
		if v := stringFromMap(dep, "ModelID"); v != "" {
			modelReq.ModelID = v
		}

		if accsRaw, ok := dep["Accelerators"].([]any); ok {
			for _, accRaw := range accsRaw {
				acc, ok := accRaw.(map[string]any)
				if !ok {
					continue
				}
				scale, err := uint64FromMap(acc, "Scale")
				if err != nil {
					continue
				}
				modelReq.Accelerators = append(modelReq.Accelerators, &godo.DedicatedInferenceAcceleratorRequest{
					AcceleratorSlug: stringFromMap(acc, "AcceleratorSlug"),
					Scale:           scale,
					Type:            stringFromMap(acc, "Type"),
				})
			}
		}
		deployments = append(deployments, modelReq)
	}
	return deployments
}

func marshalResult(v any) (*mcp.CallToolResult, error) {
	jsonData, err := json.MarshalIndent(v, "", "  ")
	if err != nil {
		return nil, fmt.Errorf("marshal error: %w", err)
	}
	return mcp.NewToolResultText(string(jsonData)), nil
}

var acceleratorItemsSchema = map[string]any{
	"type": "object",
	"properties": map[string]any{
		"AcceleratorSlug": map[string]any{"type": "string", "description": "GPU accelerator slug"},
		"Scale":           map[string]any{"type": "number", "description": "Number of accelerator instances"},
		"Type":            map[string]any{"type": "string", "description": "Accelerator type"},
	},
	"required": []string{"AcceleratorSlug", "Scale", "Type"},
}

func modelDeploymentSchema(requiredFields []string) map[string]any {
	return map[string]any{
		"type": "object",
		"properties": map[string]any{
			"ModelSlug":     map[string]any{"type": "string", "description": "Slug identifier of the model"},
			"ModelProvider": map[string]any{"type": "string", "description": "Model provider"},
			"ModelID":       map[string]any{"type": "string", "description": "Model deployment ID"},
			"Accelerators": map[string]any{
				"type":        "array",
				"items":       acceleratorItemsSchema,
				"description": "GPU accelerators for this model deployment",
			},
		},
		"required": requiredFields,
	}
}

func stringFromMap(m map[string]any, key string) string {
	v, _ := m[key].(string)
	return v
}

func uint64FromMap(m map[string]any, key string) (uint64, error) {
	v, ok := m[key].(float64)
	if !ok {
		return 0, fmt.Errorf("%s must be a number", key)
	}
	if v < 1 {
		return 0, fmt.Errorf("%s must be greater than 0, got %v", key, v)
	}
	return uint64(v), nil
}
