package genaicustommodels

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"

	"github.com/digitalocean/godo"
	"github.com/mark3labs/mcp-go/mcp"
	"github.com/mark3labs/mcp-go/server"
)

const (
	importConsentRequiredMsg = "accept_terms_and_conditions must be true. Before importing, present the import terms to the user and obtain explicit consent (yes) in this conversation. Consent is required for every import, including re-imports of the same model."

	genaiCustomModelsImportToolDescription = "Import a custom model from an external source (e.g. HuggingFace). Starts an async import job.\n\n" +
		"CONSENT REQUIRED (every import): Do not call this tool until the user has explicitly agreed to the import terms in the current conversation. " +
		"Consent is checked before any import related API calls " +
		"Present the terms (storage cost, license, source) and ask for yes/no. This applies to every import request, even if the same model was imported before. " +
		"Only pass accept_terms_and_conditions: true after the user says yes."

	genaiCustomModelsImportAcceptTermsDescription = "Must be true only after the end user has explicitly agreed in conversation to the import terms (yes/no in chat). " +
		"Omitted or false is rejected. Required on every import, including re-imports."
)

// CustomModelsTool provides custom model management tools.
type CustomModelsTool struct {
	client func(ctx context.Context) (*godo.Client, error)
}

// NewCustomModelsTool creates a new CustomModelsTool instance.
func NewCustomModelsTool(client func(ctx context.Context) (*godo.Client, error)) *CustomModelsTool {
	return &CustomModelsTool{client: client}
}

// listModels lists custom models with optional filters. Returns one markdown table with
// every model on its own row (including STATUS_FAILED with UUID). For catalog + custom
// together, use genai-models-unified-search.
func (cmt *CustomModelsTool) listModels(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	args := req.GetArguments()

	client, err := cmt.client(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to get DigitalOcean client: %w", err)
	}

	if !listModelsHasFilters(args) {
		rows, err := fetchCustomModels(ctx, client, "")
		if err != nil {
			return mcp.NewToolResultErrorFromErr("failed to list custom models", err), nil
		}
		return mcp.NewToolResultText(formatCustomModelsList("", rows)), nil
	}

	statusFilter := ""
	opt := &godo.CustomModelListOptions{}
	if status, ok := args["status"].(string); ok && status != "" {
		statusFilter = status
		opt.Status = godo.CustomModelStatus(status)
	}
	if page, ok := args["page"].(float64); ok && int(page) > 0 {
		opt.Page = int(page)
	}
	if perPage, ok := args["per_page"].(float64); ok && int(perPage) > 0 {
		opt.PerPage = int(perPage)
	}

	models, _, err := listCustomModels(ctx, client, opt)
	if err != nil {
		return mcp.NewToolResultErrorFromErr("failed to list custom models", err), nil
	}

	rows := make([]CustomSearchRow, 0, len(models))
	for _, cm := range models {
		rows = append(rows, toCustomSearchRow(cm))
	}

	return mcp.NewToolResultText(formatCustomModelsList(statusFilter, rows)), nil
}

func listModelsHasFilters(args map[string]any) bool {
	if status, ok := args["status"].(string); ok && strings.TrimSpace(status) != "" {
		return true
	}
	if page, ok := args["page"].(float64); ok && int(page) > 0 {
		return true
	}
	if perPage, ok := args["per_page"].(float64); ok && int(perPage) > 0 {
		return true
	}
	return false
}

// importModel imports a custom model from an external source.
func (cmt *CustomModelsTool) importModel(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	args := req.GetArguments()

	acceptTerms, _ := args["accept_terms_and_conditions"].(bool)
	if !acceptTerms {
		return mcp.NewToolResultError(importConsentRequiredMsg), nil
	}

	var name string
	if nameRaw, ok := args["name"]; ok && nameRaw != nil {
		if s, ok := nameRaw.(string); ok {
			name = strings.TrimSpace(s)
		}
	}

	sourceType, _ := args["source_type"].(string)
	if sourceType == "" {
		return mcp.NewToolResultError("source_type is required"), nil
	}

	sourceRefRaw, ok := args["source_ref"].(map[string]interface{})
	if !ok || sourceRefRaw == nil {
		return mcp.NewToolResultError("source_ref is required"), nil
	}

	sourceRef := CustomModelSourceRef{}
	if v, ok := sourceRefRaw["repo_id"].(string); ok {
		sourceRef.RepoID = v
	}
	if v, ok := sourceRefRaw["commit_sha"].(string); ok {
		sourceRef.CommitSHA = v
	}
	if v, ok := sourceRefRaw["access_type"].(string); ok {
		sourceRef.AccessType = CustomModelAccessType(v)
	}
	if v, ok := sourceRefRaw["hf_token"].(string); ok {
		sourceRef.HFToken = v
	}
	if v, ok := sourceRefRaw["bucket"].(string); ok {
		sourceRef.Bucket = v
	}
	if v, ok := sourceRefRaw["region"].(string); ok {
		sourceRef.Region = v
	}
	if v, ok := sourceRefRaw["prefix"].(string); ok {
		sourceRef.Prefix = v
	}

	if CustomModelSourceType(sourceType) == CustomModelSourceTypeHuggingFace {
		if err := ensureHuggingFaceCommitSHA(ctx, &sourceRef); err != nil {
			return mcp.NewToolResultError(fmt.Sprintf("failed to resolve Hugging Face commit_sha: %v", err)), nil
		}
	}

	input := &ImportCustomModelInput{
		Name:                     name,
		SourceType:               CustomModelSourceType(sourceType),
		SourceRef:                sourceRef,
		AcceptTermsAndConditions: acceptTerms,
	}

	if desc, ok := args["description"].(string); ok && desc != "" {
		input.Description = desc
	}
	if region, ok := args["preferred_gpu_region"].(string); ok && region != "" {
		input.PreferredGPURegion = region
	}
	if tagsRaw, ok := args["tags"].(map[string]interface{}); ok {
		if tagsList, ok := tagsRaw["tags"].([]interface{}); ok {
			var tags []string
			for _, t := range tagsList {
				if s, ok := t.(string); ok {
					tags = append(tags, s)
				}
			}
			input.Tags = &CustomModelTags{Tags: tags}
		}
	}

	if input.Name == "" && input.SourceRef.RepoID != "" {
		input.Name = input.SourceRef.RepoID
	}

	client, err := cmt.client(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to get DigitalOcean client: %w", err)
	}

	out, _, err := client.GradientAI.ImportCustomModel(ctx, importRequestToGodo(input))
	if err != nil {
		return mcp.NewToolResultErrorFromErr("failed to import custom model", err), nil
	}

	jsonData, err := json.MarshalIndent(importResponseFromGodo(out), "", "  ")
	if err != nil {
		return nil, fmt.Errorf("marshal error: %w", err)
	}

	return mcp.NewToolResultText(string(jsonData)), nil
}

// updateMetadata updates the metadata of a custom model.
func (cmt *CustomModelsTool) updateMetadata(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	args := req.GetArguments()

	uuid, _ := args["uuid"].(string)
	if uuid == "" {
		return mcp.NewToolResultError("uuid is required"), nil
	}

	input := &UpdateCustomModelMetadataInput{}
	hasUpdate := false

	if name, ok := args["name"].(string); ok && name != "" {
		input.Name = &name
		hasUpdate = true
	}
	if desc, ok := args["description"].(string); ok && desc != "" {
		input.Description = &desc
		hasUpdate = true
	}
	if tagsRaw, ok := args["tags"].(map[string]interface{}); ok {
		if tagsList, ok := tagsRaw["tags"].([]interface{}); ok {
			var tags []string
			for _, t := range tagsList {
				if s, ok := t.(string); ok {
					tags = append(tags, s)
				}
			}
			input.Tags = &CustomModelTags{Tags: tags}
			hasUpdate = true
		}
	}

	if !hasUpdate {
		return mcp.NewToolResultError("at least one of name, description, or tags must be provided"), nil
	}

	client, err := cmt.client(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to get DigitalOcean client: %w", err)
	}

	model, _, err := client.GradientAI.UpdateCustomModelMetadata(ctx, uuid, metadataUpdateToGodo(input))
	if err != nil {
		return mcp.NewToolResultErrorFromErr("failed to update custom model metadata", err), nil
	}

	jsonData, err := json.MarshalIndent(customModelFromGodo(model), "", "  ")
	if err != nil {
		return nil, fmt.Errorf("marshal error: %w", err)
	}

	return mcp.NewToolResultText(string(jsonData)), nil
}

// getModel retrieves a single custom model by UUID (public endpoint).
func (cmt *CustomModelsTool) getModel(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	uuid, _ := req.GetArguments()["uuid"].(string)
	if uuid == "" {
		return mcp.NewToolResultError("uuid is required"), nil
	}

	client, err := cmt.client(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to get DigitalOcean client: %w", err)
	}

	model, err := getCustomModelByUUID(ctx, client, uuid)
	if err != nil {
		return mcp.NewToolResultErrorFromErr("failed to get custom model", err), nil
	}

	jsonData, err := json.MarshalIndent(model, "", "  ")
	if err != nil {
		return nil, fmt.Errorf("marshal error: %w", err)
	}

	return mcp.NewToolResultText(string(jsonData)), nil
}

// deleteModel deletes a custom model by exact uuid or exact name (partial identifiers return candidates only).
func (cmt *CustomModelsTool) deleteModel(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	args := req.GetArguments()
	uuid, _ := args["uuid"].(string)
	uuid = strings.TrimSpace(uuid)

	name := ""
	if nameRaw, hasName := args["name"]; hasName && nameRaw != nil {
		var ok bool
		name, ok = nameRaw.(string)
		if !ok {
			return mcp.NewToolResultError(deleteModelNameTypeMsg), nil
		}
		name = strings.TrimSpace(name)
	}

	if uuid == "" && name == "" {
		return mcp.NewToolResultError(deleteModelIdentifierRequiredMsg), nil
	}

	client, err := cmt.client(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to get DigitalOcean client: %w", err)
	}

	models, err := listAllCustomModels(ctx, client)
	if err != nil {
		return mcp.NewToolResultErrorFromErr("failed to list custom models for delete resolution", err), nil
	}

	target, unresolved, err := resolveDeleteTarget(uuid, name, models)
	if err != nil {
		return mcp.NewToolResultError(err.Error()), nil
	}
	if unresolved != nil {
		text, err := marshalDeleteUnresolvedResult(unresolved)
		if err != nil {
			return nil, err
		}
		return mcp.NewToolResultError(text), nil
	}

	confirmDeletion, _ := args["confirm_deletion"].(bool)
	if !confirmDeletion {
		return mcp.NewToolResultError(deleteConsentRequiredMsg), nil
	}

	model, err := getCustomModelByUUID(ctx, client, target.UUID)
	if err != nil {
		return mcp.NewToolResultErrorFromErr("failed to verify custom model before delete", err), nil
	}
	if model.Name != target.Name {
		return mcp.NewToolResultError(fmt.Sprintf("model name mismatch: resolved %q but API returned %q", target.Name, model.Name)), nil
	}

	return cmt.deleteModelByUUID(ctx, client, target.UUID, target.Name)
}

func (cmt *CustomModelsTool) deleteModelByUUID(ctx context.Context, client *godo.Client, uuid, name string) (*mcp.CallToolResult, error) {
	output, resp, err := deleteCustomModelByUUID(ctx, client, uuid)
	if err != nil {
		if isNotFoundResponse(resp) {
			return mcp.NewToolResultError(fmt.Sprintf(
				"DELETE returned 404 for model %q (%s) but the model still exists. Deletion may be blocked (for example active deployments) or the delete API may be unavailable in this environment — verify in the control panel or remove deployments first.",
				name, uuid)), nil
		}
		return mcp.NewToolResultErrorFromErr("failed to delete custom model", err), nil
	}

	jsonData, err := json.MarshalIndent(output, "", "  ")
	if err != nil {
		return nil, fmt.Errorf("marshal error: %w", err)
	}

	return mcp.NewToolResultText(string(jsonData)), nil
}

// Tools returns the list of server tools for custom model management.
func (cmt *CustomModelsTool) Tools() []server.ServerTool {
	return []server.ServerTool{
		{
			Handler: cmt.unifiedSearch,
			Tool: mcp.NewTool(
				"genai-models-unified-search",
				mcp.WithDescription("PRIMARY tool for listing or searching models. Use when the user asks to list all models, show available models, or search by partial name. Returns two markdown tables (Model Catalog and Custom Models) with one row per model: custom columns are UUID, Name, Source, Status, Architecture, Input Modalities, Output Modalities; catalog columns include Provider, Type, Context Window, Capabilities, and modalities. Empty query lists everything; partial query returns nearest matches."),
				mcp.WithString("query", mcp.Description("Partial model name or search string (optional). Empty returns all models in both tables.")),
			),
		},
		{
			Handler: cmt.listModels,
			Tool: mcp.NewTool(
				"genai-custom-models-list",
				mcp.WithDescription("List all custom models in one markdown table (one row per model, every UUID shown including STATUS_FAILED). Columns: UUID, Name, Source, Status, Architecture, Input Modalities, Output Modalities. Do not summarize the table in the response. For catalog + custom together use genai-models-unified-search. Optional status/page/per_page filters."),
				mcp.WithString("status", mcp.Description("Filter by status: STATUS_IMPORTING, STATUS_READY, STATUS_FAILED, STATUS_DELETED")),
				mcp.WithNumber("page", mcp.Description("Page number for pagination (default: 1)")),
				mcp.WithNumber("per_page", mcp.Description("Results per page (default: 20)")),
			),
		},
		{
			Handler: cmt.importModel,
			Tool: mcp.NewTool(
				"genai-custom-models-import",
				mcp.WithDescription(genaiCustomModelsImportToolDescription),
				mcp.WithString("name", mcp.Description("Optional display name for the custom model (leading/trailing whitespace is trimmed when provided).")),
				mcp.WithString("source_type", mcp.Required(), mcp.Description("Source type: SOURCE_TYPE_HUGGINGFACE, SOURCE_TYPE_SPACES_BUCKET, SOURCE_TYPE_SDK_UPLOAD, SOURCE_TYPE_FINE_TUNING")),
				mcp.WithObject("source_ref", mcp.Required(), mcp.Description("Source reference. For HuggingFace: repo_id (string, required), commit_sha (string, optional; if omitted, resolved from Hugging Face Hub before import), access_type (ACCESS_TYPE_PUBLIC, ACCESS_TYPE_PRIVATE, ACCESS_TYPE_GATED), hf_token (string, for private/gated models). For Spaces Bucket: bucket (string, required), region (string, optional), prefix (string, optional)")),
				mcp.WithBoolean("accept_terms_and_conditions", mcp.Required(), mcp.Description(genaiCustomModelsImportAcceptTermsDescription)),
				mcp.WithString("description", mcp.Description("Description of the model")),
				mcp.WithString("preferred_gpu_region", mcp.Description("Preferred GPU region for the model (e.g. nyc3)")),
				mcp.WithObject("tags", mcp.Description("Tags object with a 'tags' array of strings")),
			),
		},
		{
			Handler: cmt.updateMetadata,
			Tool: mcp.NewTool(
				"genai-custom-models-update-metadata",
				mcp.WithDescription("Update the name, description, or tags of an existing custom model."),
				mcp.WithString("uuid", mcp.Required(), mcp.Description("UUID of the custom model to update")),
				mcp.WithString("name", mcp.Description("New name for the model")),
				mcp.WithString("description", mcp.Description("New description for the model")),
				mcp.WithObject("tags", mcp.Description("New tags object with a 'tags' array of strings")),
			),
		},
		{
			Handler: cmt.getModel,
			Tool: mcp.NewTool(
				"genai-custom-models-get",
				mcp.WithDescription("Get the full catalog card for a custom model, including its status, architecture, source info, size, license, tags, active deployments, and cost estimate."),
				mcp.WithString("uuid", mcp.Required(), mcp.Description("UUID of the custom model to retrieve")),
			),
		},
		{
			Handler: cmt.deleteModel,
			Tool: mcp.NewTool(
				"genai-custom-models-delete",
				mcp.WithDescription(genaiCustomModelsDeleteToolDescription),
				mcp.WithString("name", mcp.Description("Exact custom model name the user provided or confirmed (character-for-character, whitespace trimmed). Use this OR uuid. Partial names return candidates only.")),
				mcp.WithString("uuid", mcp.Description("Exact full custom model UUID (8-4-4-4-12 hex). Use this OR name. Partial uuids return candidates only; never delete on a partial uuid even if one match.")),
				mcp.WithBoolean("confirm_deletion", mcp.Description(genaiCustomModelsDeleteConfirmDescription)),
			),
		},
	}
}
