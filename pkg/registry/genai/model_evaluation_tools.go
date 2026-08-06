package genai

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"os"
	"strings"
	"time"

	"github.com/digitalocean/godo"
	"github.com/mark3labs/mcp-go/mcp"
	"github.com/mark3labs/mcp-go/server"

	"mcp-digitalocean/pkg/registry/common"
)

// toGodoStarMetric converts the locally parsed star metric into the godo SDK type.
func toGodoStarMetric(sm *StarMetric) *godo.StarMetric {
	if sm == nil {
		return nil
	}
	g := &godo.StarMetric{MetricUUID: sm.MetricUUID}
	if sm.Name != nil {
		g.Name = *sm.Name
	}
	if sm.SuccessThreshold != nil {
		v := float32(*sm.SuccessThreshold)
		g.SuccessThreshold = &v
	}
	if sm.SuccessThresholdPct != nil {
		v := int32(*sm.SuccessThresholdPct)
		g.SuccessThresholdPct = &v
	}
	return g
}

// godoInferenceConfigFromArgs builds a godo CandidateInferenceConfig from raw tool args.
func godoInferenceConfigFromArgs(args map[string]interface{}) *godo.CandidateInferenceConfig {
	raw, ok := args["candidate_inference_config"].(map[string]interface{})
	if !ok || raw == nil {
		return nil
	}
	config := &godo.CandidateInferenceConfig{}
	if maxTokens, ok := raw["max_tokens"].(float64); ok {
		config.MaxTokens = int64(maxTokens)
	}
	if temp, ok := raw["temperature"].(float64); ok {
		config.Temperature = float32(temp)
	}
	return config
}

// isGodoModelEvalRunTerminal reports whether a godo run status is terminal.
func isGodoModelEvalRunTerminal(status godo.ModelEvaluationRunStatus) bool {
	switch status {
	case godo.ModelEvaluationRunSuccessful,
		godo.ModelEvaluationRunFailed,
		godo.ModelEvaluationRunCancelled,
		godo.ModelEvaluationRunPartiallySuccessful:
		return true
	default:
		return false
	}
}

// ModelEvaluationTool provides model evaluation management tools.
type ModelEvaluationTool struct {
	client func(ctx context.Context) (*godo.Client, error)
}

// NewModelEvaluationTool creates a new ModelEvaluationTool instance.
func NewModelEvaluationTool(client func(ctx context.Context) (*godo.Client, error)) *ModelEvaluationTool {
	return &ModelEvaluationTool{
		client: client,
	}
}

// listMetrics lists all available model evaluation metrics.
func (met *ModelEvaluationTool) listMetrics(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	client, err := met.client(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to get DigitalOcean client: %w", err)
	}

	output, _, err := client.GradientAI.ListModelEvaluationMetrics(ctx)
	if err != nil {
		return mcp.NewToolResultErrorFromErr("failed to list model evaluation metrics", err), nil
	}

	type MetricsResponse struct {
		Metrics []*godo.EvaluationMetric `json:"metrics"`
		Count   int                      `json:"count"`
	}

	response := MetricsResponse{
		Metrics: output.Metrics,
		Count:   len(output.Metrics),
	}

	jsonData, err := json.MarshalIndent(response, "", "  ")
	if err != nil {
		return nil, fmt.Errorf("marshal error: %w", err)
	}

	return mcp.NewToolResultText(string(jsonData)), nil
}

// listDatasets lists evaluation datasets (defaults to model-evaluation datasets) so a
// previously uploaded dataset's UUID can be reused in a model evaluation run.
func (met *ModelEvaluationTool) listDatasets(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	args := req.GetArguments()

	datasetType := EvaluationDatasetTypeModel
	if v, ok := args["dataset_type"].(string); ok && strings.TrimSpace(v) != "" {
		datasetType = strings.TrimSpace(v)
	}

	client, err := met.client(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to get DigitalOcean client: %w", err)
	}

	path := genAIAPIPath + "/evaluation_datasets"
	q := url.Values{}
	if datasetType != "" {
		q.Set("dataset_type", datasetType)
	}
	if enc := q.Encode(); enc != "" {
		path += "?" + enc
	}

	apiReq, err := client.NewRequest(ctx, http.MethodGet, path, nil)
	if err != nil {
		return mcp.NewToolResultErrorFromErr("failed to create request", err), nil
	}

	var output ListEvaluationDatasetsOutput
	resp, err := client.Do(ctx, apiReq, &output)
	if err != nil || (resp != nil && resp.StatusCode >= 400) {
		return mcp.NewToolResultErrorFromErr("failed to list evaluation datasets", err), nil
	}

	type DatasetsResponse struct {
		Datasets []*ModelEvalDatasetListItem `json:"datasets"`
		Count    int                         `json:"count"`
	}

	response := DatasetsResponse{
		Datasets: output.EvaluationDatasets,
		Count:    len(output.EvaluationDatasets),
	}

	jsonData, err := json.MarshalIndent(response, "", "  ")
	if err != nil {
		return nil, fmt.Errorf("marshal error: %w", err)
	}

	return mcp.NewToolResultText(string(jsonData)), nil
}

// listPresets lists all model evaluation presets.
func (met *ModelEvaluationTool) listPresets(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	client, err := met.client(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to get DigitalOcean client: %w", err)
	}

	output, _, err := client.GradientAI.ListModelEvaluationPresets(ctx)
	if err != nil {
		return mcp.NewToolResultErrorFromErr("failed to list model evaluation presets", err), nil
	}

	type PresetsResponse struct {
		Presets []*godo.ModelEvaluationPreset `json:"presets"`
		Count   int                           `json:"count"`
	}

	response := PresetsResponse{
		Presets: output.Presets,
		Count:   len(output.Presets),
	}

	jsonData, err := json.MarshalIndent(response, "", "  ")
	if err != nil {
		return nil, fmt.Errorf("marshal error: %w", err)
	}

	return mcp.NewToolResultText(string(jsonData)), nil
}

// getPreset gets a single model evaluation preset by UUID.
func (met *ModelEvaluationTool) getPreset(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	presetUUID, ok := req.GetArguments()["eval_preset_uuid"].(string)
	if !ok || presetUUID == "" {
		return mcp.NewToolResultError("eval_preset_uuid is required"), nil
	}

	client, err := met.client(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to get DigitalOcean client: %w", err)
	}

	output, _, err := client.GradientAI.GetModelEvaluationPreset(ctx, presetUUID)
	if err != nil {
		return mcp.NewToolResultErrorFromErr("failed to get model evaluation preset", err), nil
	}

	jsonData, err := json.MarshalIndent(output.Preset, "", "  ")
	if err != nil {
		return nil, fmt.Errorf("marshal error: %w", err)
	}

	return mcp.NewToolResultText(string(jsonData)), nil
}

// createDataset uploads a CSV or JSONL file to Spaces and registers it as a model evaluation dataset.
func (met *ModelEvaluationTool) createDataset(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	args := req.GetArguments()

	name, ok := args["name"].(string)
	if !ok || name == "" {
		return mcp.NewToolResultError("name is required"), nil
	}

	filePath, ok := args["file_path"].(string)
	if !ok || filePath == "" {
		return mcp.NewToolResultError("file_path is required"), nil
	}

	if err := validateModelEvaluationDataset(filePath); err != nil {
		return mcp.NewToolResultErrorFromErr("dataset validation failed", err), nil
	}

	fileData, err := os.ReadFile(filePath)
	if err != nil {
		return mcp.NewToolResultErrorFromErr("failed to read file", err), nil
	}

	client, err := met.client(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to get DigitalOcean client: %w", err)
	}

	result, err := uploadAndRegisterModelEvaluationDataset(ctx, client, name, fileData, getFileName(filePath))
	if err != nil {
		return mcp.NewToolResultErrorFromErr("failed to create model evaluation dataset", err), nil
	}

	jsonData, err := json.MarshalIndent(result, "", "  ")
	if err != nil {
		return nil, fmt.Errorf("marshal error: %w", err)
	}

	return mcp.NewToolResultText(string(jsonData)), nil
}

// resolveEvalModelsOnly resolves candidate and judge models before chat-based user consent.
func (met *ModelEvaluationTool) resolveEvalModelsOnly(
	ctx context.Context,
	args map[string]any,
	requireJudge bool,
) (*modelEvalRunModels, *mcp.CallToolResult, error) {
	candidateModelName := strings.TrimSpace(stringArg(args, "candidate_model_name"))
	if candidateModelName == "" {
		return nil, mcp.NewToolResultError(modelEvalCandidateNameRequiredMsg), nil
	}

	client, err := met.client(ctx)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to get DigitalOcean client: %w", err)
	}

	models, err := listAllEvalModels(ctx, client)
	if err != nil {
		return nil, mcp.NewToolResultErrorFromErr("failed to list models for evaluation run resolution", err), nil
	}

	resolved, unresolved, err := resolveEvalModelsForRun(
		ctx,
		client,
		strings.TrimSpace(stringArg(args, "candidate_model_uuid")),
		candidateModelName,
		strings.TrimSpace(stringArg(args, "judge_model_uuid")),
		strings.TrimSpace(stringArg(args, "judge_model_name")),
		requireJudge,
		models,
	)
	if err != nil {
		return nil, mcp.NewToolResultError(err.Error()), nil
	}
	if unresolved != nil {
		result, err := modelEvalUserActionResult(unresolved)
		return nil, result, err
	}

	hydrated, err := hydrateResolvedEvalModelsFromAPI(ctx, client, resolved)
	if err != nil {
		return nil, mcp.NewToolResultErrorFromErr("failed to fetch catalog UUIDs for candidate/judge models", err), nil
	}
	if err := validateModelEvalResolvedUUIDs(hydrated, requireJudge); err != nil {
		return nil, mcp.NewToolResultError(err.Error()), nil
	}

	return hydrated, nil, nil
}

// createRun creates a new model evaluation run.
func (met *ModelEvaluationTool) createRun(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	args := req.GetArguments()

	name, _ := args["name"].(string)
	if name == "" {
		return mcp.NewToolResultError("name is required"), nil
	}
	presetUUID, _ := args["eval_preset_uuid"].(string)
	requireJudge := presetUUID == ""

	runCfg, err := parseModelEvalRunConfig(args, requireJudge)
	if err != nil {
		return mcp.NewToolResultError(err.Error()), nil
	}

	resolved, selectionResult, err := met.resolveEvalModelsOnly(ctx, args, requireJudge)
	if selectionResult != nil || err != nil {
		return selectionResult, err
	}

	if consentResult, ok := checkModelEvalUserMessage(args, resolved, runCfg); !ok {
		return consentResult, nil
	}

	client, err := met.client(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to get DigitalOcean client: %w", err)
	}

	createReq := &godo.CreateModelEvaluationRunRequest{
		Name:               name,
		CandidateModelUUID: resolved.Candidate.UUID,
		CandidateModelName: resolved.Candidate.APIName,
	}

	if presetUUID != "" {
		createReq.EvalPresetUUID = presetUUID
	}

	if datasetUUID, ok := args["dataset_uuid"].(string); ok && datasetUUID != "" {
		createReq.DatasetUUID = datasetUUID
	}

	if resolved.Judge != nil {
		createReq.JudgeModelUUID = resolved.Judge.UUID
	}

	if metricUUIDsRaw, ok := args["metric_uuids"].([]interface{}); ok {
		for _, m := range metricUUIDsRaw {
			if s, ok := m.(string); ok {
				createReq.MetricUUIDs = append(createReq.MetricUUIDs, s)
			}
		}
	}

	createReq.StarMetric = toGodoStarMetric(parseStarMetricArg(args))

	if source, ok := args["source"].(string); ok && source != "" {
		createReq.Source = source
	}

	if presetName, ok := args["preset_name"].(string); ok && presetName != "" {
		createReq.PresetName = presetName
	}

	if candidateModelSource, ok := args["candidate_model_source"].(string); ok && candidateModelSource != "" {
		createReq.CandidateModelSource = godo.CandidateModelSource(candidateModelSource)
	}

	createReq.CandidateInferenceConfig = godoInferenceConfigFromArgs(args)

	output, _, err := client.GradientAI.CreateModelEvaluationRun(ctx, createReq)
	if err != nil {
		return mcp.NewToolResultErrorFromErr("failed to create model evaluation run", err), nil
	}

	type RunCreatedResponse struct {
		EvalRunUUID string `json:"eval_run_uuid"`
		Name        string `json:"name"`
	}

	response := RunCreatedResponse{
		EvalRunUUID: output.EvalRunUuid,
		Name:        name,
	}

	jsonData, err := json.MarshalIndent(response, "", "  ")
	if err != nil {
		return nil, fmt.Errorf("marshal error: %w", err)
	}

	return mcp.NewToolResultText(string(jsonData)), nil
}

// listRuns lists model evaluation runs with optional filters.
func (met *ModelEvaluationTool) listRuns(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	args := req.GetArguments()

	client, err := met.client(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to get DigitalOcean client: %w", err)
	}

	opt := &godo.ModelEvaluationRunListOptions{}
	if presetUUID, ok := args["eval_preset_uuid"].(string); ok && presetUUID != "" {
		opt.EvalPresetUUID = presetUUID
	}
	if status, ok := args["status"].(string); ok && status != "" {
		opt.Status = godo.ModelEvaluationRunStatus(status)
	}
	if page, ok := args["page"].(float64); ok {
		opt.Page = int(page)
	}
	if perPage, ok := args["per_page"].(float64); ok {
		opt.PerPage = int(perPage)
	}

	output, _, err := client.GradientAI.ListModelEvaluationRuns(ctx, opt)
	if err != nil {
		return mcp.NewToolResultErrorFromErr("failed to list model evaluation runs", err), nil
	}

	type RunsListResponse struct {
		Runs  []*godo.ModelEvaluationRunSummary `json:"runs"`
		Count int                               `json:"count"`
	}

	response := RunsListResponse{
		Runs:  output.Runs,
		Count: len(output.Runs),
	}

	jsonData, err := json.MarshalIndent(response, "", "  ")
	if err != nil {
		return nil, fmt.Errorf("marshal error: %w", err)
	}

	return mcp.NewToolResultText(string(jsonData)), nil
}

// getRun gets a single model evaluation run with per-prompt results.
func (met *ModelEvaluationTool) getRun(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	args := req.GetArguments()

	runUUID, ok := args["eval_run_uuid"].(string)
	if !ok || runUUID == "" {
		return mcp.NewToolResultError("eval_run_uuid is required"), nil
	}

	client, err := met.client(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to get DigitalOcean client: %w", err)
	}

	opt := &godo.ModelEvaluationRunGetOptions{}
	if page, ok := args["page"].(float64); ok {
		opt.Page = int(page)
	}
	if perPage, ok := args["per_page"].(float64); ok {
		opt.PerPage = int(perPage)
	}

	output, _, err := client.GradientAI.GetModelEvaluationRun(ctx, runUUID, opt)
	if err != nil {
		return mcp.NewToolResultErrorFromErr("failed to get model evaluation run", err), nil
	}

	jsonData, err := json.MarshalIndent(output, "", "  ")
	if err != nil {
		return nil, fmt.Errorf("marshal error: %w", err)
	}

	return mcp.NewToolResultText(string(jsonData)), nil
}

// updateRun updates mutable fields on a model evaluation run (currently name only).
func (met *ModelEvaluationTool) updateRun(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	args := req.GetArguments()

	runUUID, ok := args["eval_run_uuid"].(string)
	if !ok || runUUID == "" {
		return mcp.NewToolResultError("eval_run_uuid is required"), nil
	}

	name, ok := args["name"].(string)
	if !ok || name == "" {
		return mcp.NewToolResultError("name is required"), nil
	}

	client, err := met.client(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to get DigitalOcean client: %w", err)
	}

	output, _, err := client.GradientAI.UpdateModelEvaluationRun(ctx, runUUID, &godo.UpdateModelEvaluationRunRequest{
		Name: name,
	})
	if err != nil {
		return mcp.NewToolResultErrorFromErr("failed to update model evaluation run", err), nil
	}

	if output == nil || output.Run == nil {
		return mcp.NewToolResultError("empty response from update model evaluation run"), nil
	}

	jsonData, err := json.MarshalIndent(output.Run, "", "  ")
	if err != nil {
		return nil, fmt.Errorf("marshal error: %w", err)
	}

	return mcp.NewToolResultText(string(jsonData)), nil
}

// getResultsDownloadURL gets a presigned download URL for run results.
func (met *ModelEvaluationTool) getResultsDownloadURL(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	runUUID, ok := req.GetArguments()["eval_run_uuid"].(string)
	if !ok || runUUID == "" {
		return mcp.NewToolResultError("eval_run_uuid is required"), nil
	}

	client, err := met.client(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to get DigitalOcean client: %w", err)
	}

	output, _, err := client.GradientAI.GetModelEvaluationRunResultsDownloadURL(ctx, runUUID)
	if err != nil {
		return mcp.NewToolResultErrorFromErr("failed to get results download URL", err), nil
	}

	jsonData, err := json.MarshalIndent(output, "", "  ")
	if err != nil {
		return nil, fmt.Errorf("marshal error: %w", err)
	}

	return mcp.NewToolResultText(string(jsonData)), nil
}

const (
	modelEvalDeleteRunConsentMsg = "confirm_deletion must be true. Before deleting an evaluation run, present the run UUID and that deletion is permanent (results cannot be recovered) and obtain explicit consent (yes) in this conversation. Consent is required for every delete."

	modelEvalDeletePresetConsentMsg = "confirm_deletion must be true. Before deleting an evaluation preset, present the preset UUID and that deletion is permanent and obtain explicit consent (yes) in this conversation. Consent is required for every delete."

	modelEvalDeleteDatasetConsentMsg = "confirm_deletion must be true. Before deleting an evaluation dataset, present the dataset UUID and that deletion is permanent and obtain explicit consent (yes) in this conversation. Consent is required for every delete."

	modelEvalDeleteCustomMetricConsentMsg = "confirm_deletion must be true. Before deleting a custom evaluation metric, present the metric UUID and that the metric will no longer be available for new evaluation runs, and obtain explicit consent (yes) in this conversation. Consent is required for every delete."

	modelEvalCancelRunConsentMsg = "confirm_cancel must be true. Before cancelling an evaluation run, present the run UUID and explain that any partial results may be lost, then obtain explicit consent (yes) in this conversation."

	modelEvalDeleteRunConfirmDescription = "Must be true only after the end user has explicitly confirmed the deletion in conversation (yes/no in chat). Omitted or false is rejected."

	modelEvalDeletePresetConfirmDescription = "Must be true only after the end user has explicitly confirmed the preset deletion in conversation (yes/no in chat). Omitted or false is rejected."

	modelEvalDeleteDatasetConfirmDescription = "Must be true only after the end user has explicitly confirmed the dataset deletion in conversation (yes/no in chat). Omitted or false is rejected."

	modelEvalDeleteCustomMetricConfirmDescription = "Must be true only after the end user has explicitly confirmed the custom metric deletion in conversation (yes/no in chat). Omitted or false is rejected."

	modelEvalCancelRunConfirmDescription = "Must be true only after the end user has explicitly confirmed the cancellation in conversation (yes/no in chat). Omitted or false is rejected."
)

// deleteRun deletes a model evaluation run by UUID.
func (met *ModelEvaluationTool) deleteRun(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	args := req.GetArguments()

	runUUID, _ := args["eval_run_uuid"].(string)
	runUUID = strings.TrimSpace(runUUID)
	if runUUID == "" {
		return mcp.NewToolResultError("eval_run_uuid is required"), nil
	}

	if confirm, _ := args["confirm_deletion"].(bool); !confirm {
		return mcp.NewToolResultError(modelEvalDeleteRunConsentMsg), nil
	}

	client, err := met.client(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to get DigitalOcean client: %w", err)
	}

	output, resp, err := client.GradientAI.DeleteModelEvaluationRun(ctx, runUUID)
	if err != nil {
		return mcp.NewToolResultErrorFromErr("failed to delete model evaluation run", err), nil
	}
	if resp != nil && resp.StatusCode >= 400 {
		return mcp.NewToolResultError(fmt.Sprintf("failed to delete model evaluation run: status %d", resp.StatusCode)), nil
	}

	jsonData, err := json.MarshalIndent(output, "", "  ")
	if err != nil {
		return nil, fmt.Errorf("marshal error: %w", err)
	}
	return mcp.NewToolResultText(string(jsonData)), nil
}

// cancelRun cancels an in-progress model evaluation run by UUID.
func (met *ModelEvaluationTool) cancelRun(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	args := req.GetArguments()

	runUUID, _ := args["eval_run_uuid"].(string)
	runUUID = strings.TrimSpace(runUUID)
	if runUUID == "" {
		return mcp.NewToolResultError("eval_run_uuid is required"), nil
	}

	if confirm, _ := args["confirm_cancel"].(bool); !confirm {
		return mcp.NewToolResultError(modelEvalCancelRunConsentMsg), nil
	}

	client, err := met.client(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to get DigitalOcean client: %w", err)
	}

	output, resp, err := client.GradientAI.CancelModelEvaluationRun(ctx, runUUID)
	if err != nil {
		return mcp.NewToolResultErrorFromErr("failed to cancel model evaluation run", err), nil
	}
	if resp != nil && resp.StatusCode >= 400 {
		return mcp.NewToolResultError(fmt.Sprintf("failed to cancel model evaluation run: status %d", resp.StatusCode)), nil
	}

	jsonData, err := json.MarshalIndent(output, "", "  ")
	if err != nil {
		return nil, fmt.Errorf("marshal error: %w", err)
	}
	return mcp.NewToolResultText(string(jsonData)), nil
}

// deletePreset deletes a saved model evaluation preset by UUID.
func (met *ModelEvaluationTool) deletePreset(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	args := req.GetArguments()

	presetUUID, _ := args["eval_preset_uuid"].(string)
	presetUUID = strings.TrimSpace(presetUUID)
	if presetUUID == "" {
		return mcp.NewToolResultError("eval_preset_uuid is required"), nil
	}

	if confirm, _ := args["confirm_deletion"].(bool); !confirm {
		return mcp.NewToolResultError(modelEvalDeletePresetConsentMsg), nil
	}

	client, err := met.client(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to get DigitalOcean client: %w", err)
	}

	output, resp, err := client.GradientAI.DeleteModelEvaluationPreset(ctx, presetUUID)
	if err != nil {
		return mcp.NewToolResultErrorFromErr("failed to delete model evaluation preset", err), nil
	}
	if resp != nil && resp.StatusCode >= 400 {
		return mcp.NewToolResultError(fmt.Sprintf("failed to delete model evaluation preset: status %d", resp.StatusCode)), nil
	}

	type DeletePresetResponse struct {
		EvalPresetUUID string `json:"eval_preset_uuid"`
		Status         string `json:"status"`
	}
	response := DeletePresetResponse{
		EvalPresetUUID: presetUUID,
		Status:         "deleted",
	}
	_ = output
	jsonData, err := json.MarshalIndent(response, "", "  ")
	if err != nil {
		return nil, fmt.Errorf("marshal error: %w", err)
	}
	return mcp.NewToolResultText(string(jsonData)), nil
}

// deleteDataset deletes an evaluation dataset by UUID. This works for both model
// and agent evaluation datasets.
func (met *ModelEvaluationTool) deleteDataset(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	args := req.GetArguments()

	datasetUUID, _ := args["dataset_uuid"].(string)
	datasetUUID = strings.TrimSpace(datasetUUID)
	if datasetUUID == "" {
		return mcp.NewToolResultError("dataset_uuid is required"), nil
	}

	if confirm, _ := args["confirm_deletion"].(bool); !confirm {
		return mcp.NewToolResultError(modelEvalDeleteDatasetConsentMsg), nil
	}

	client, err := met.client(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to get DigitalOcean client: %w", err)
	}

	output, resp, err := client.GradientAI.DeleteEvaluationDataset(ctx, datasetUUID)
	if err != nil {
		return mcp.NewToolResultErrorFromErr("failed to delete evaluation dataset", err), nil
	}
	if resp != nil && resp.StatusCode >= 400 {
		return mcp.NewToolResultError(fmt.Sprintf("failed to delete evaluation dataset: status %d", resp.StatusCode)), nil
	}

	type DeleteDatasetResponse struct {
		DatasetUUID string `json:"dataset_uuid"`
		Status      string `json:"status"`
	}
	response := DeleteDatasetResponse{
		DatasetUUID: datasetUUID,
		Status:      "deleted",
	}
	_ = output
	jsonData, err := json.MarshalIndent(response, "", "  ")
	if err != nil {
		return nil, fmt.Errorf("marshal error: %w", err)
	}
	return mcp.NewToolResultText(string(jsonData)), nil
}

// createCustomMetric creates a custom (LLM-as-judge) model evaluation metric.
func (met *ModelEvaluationTool) createCustomMetric(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	args := req.GetArguments()

	metricName := strings.TrimSpace(stringArg(args, "metric_name"))
	if metricName == "" {
		return mcp.NewToolResultError("metric_name is required"), nil
	}

	scoringPrompt := strings.TrimSpace(stringArg(args, "scoring_prompt"))
	if scoringPrompt == "" {
		return mcp.NewToolResultError("scoring_prompt is required"), nil
	}

	requiresGroundTruth, _ := args["requires_ground_truth"].(bool)

	client, err := met.client(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to get DigitalOcean client: %w", err)
	}

	metric, resp, err := client.GradientAI.CreateCustomEvaluationMetric(ctx, &godo.CreateCustomEvaluationMetricRequest{
		MetricName:  metricName,
		Description: strings.TrimSpace(stringArg(args, "description")),
		Config: &godo.CustomEvaluationMetricConfig{
			ScoringPrompt:       scoringPrompt,
			RequiresGroundTruth: requiresGroundTruth,
		},
	})
	if err != nil {
		return mcp.NewToolResultErrorFromErr("failed to create custom evaluation metric", err), nil
	}
	if resp != nil && resp.StatusCode >= 400 {
		return mcp.NewToolResultError(fmt.Sprintf("failed to create custom evaluation metric: status %d", resp.StatusCode)), nil
	}
	if metric == nil {
		return mcp.NewToolResultError("empty response from create custom evaluation metric"), nil
	}

	jsonData, err := json.MarshalIndent(metric, "", "  ")
	if err != nil {
		return nil, fmt.Errorf("marshal error: %w", err)
	}
	return mcp.NewToolResultText(string(jsonData)), nil
}

// updateCustomMetric updates an existing custom model evaluation metric.
func (met *ModelEvaluationTool) updateCustomMetric(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	args := req.GetArguments()

	metricUUID := strings.TrimSpace(stringArg(args, "metric_uuid"))
	if metricUUID == "" {
		return mcp.NewToolResultError("metric_uuid is required"), nil
	}

	metricName := strings.TrimSpace(stringArg(args, "metric_name"))
	description := strings.TrimSpace(stringArg(args, "description"))
	scoringPrompt := strings.TrimSpace(stringArg(args, "scoring_prompt"))
	requiresGroundTruth, hasRequiresGroundTruth := args["requires_ground_truth"].(bool)

	if metricName == "" && description == "" && scoringPrompt == "" && !hasRequiresGroundTruth {
		return mcp.NewToolResultError("at least one of metric_name, description, scoring_prompt, or requires_ground_truth must be provided"), nil
	}

	updateReq := &godo.UpdateCustomEvaluationMetricRequest{
		MetricUUID:  metricUUID,
		MetricName:  metricName,
		Description: description,
	}
	if scoringPrompt != "" || hasRequiresGroundTruth {
		updateReq.Config = &godo.CustomEvaluationMetricConfig{
			ScoringPrompt:       scoringPrompt,
			RequiresGroundTruth: requiresGroundTruth,
		}
	}

	client, err := met.client(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to get DigitalOcean client: %w", err)
	}

	metric, resp, err := client.GradientAI.UpdateCustomEvaluationMetric(ctx, metricUUID, updateReq)
	if err != nil {
		return mcp.NewToolResultErrorFromErr("failed to update custom evaluation metric", err), nil
	}
	if resp != nil && resp.StatusCode >= 400 {
		return mcp.NewToolResultError(fmt.Sprintf("failed to update custom evaluation metric: status %d", resp.StatusCode)), nil
	}
	if metric == nil {
		return mcp.NewToolResultError("empty response from update custom evaluation metric"), nil
	}

	jsonData, err := json.MarshalIndent(metric, "", "  ")
	if err != nil {
		return nil, fmt.Errorf("marshal error: %w", err)
	}
	return mcp.NewToolResultText(string(jsonData)), nil
}

// deleteCustomMetric deletes a custom model evaluation metric by UUID.
func (met *ModelEvaluationTool) deleteCustomMetric(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	args := req.GetArguments()

	metricUUID := strings.TrimSpace(stringArg(args, "metric_uuid"))
	if metricUUID == "" {
		return mcp.NewToolResultError("metric_uuid is required"), nil
	}

	if confirm, _ := args["confirm_deletion"].(bool); !confirm {
		return mcp.NewToolResultError(modelEvalDeleteCustomMetricConsentMsg), nil
	}

	client, err := met.client(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to get DigitalOcean client: %w", err)
	}

	resp, err := client.GradientAI.DeleteCustomEvaluationMetric(ctx, metricUUID)
	if err != nil {
		return mcp.NewToolResultErrorFromErr("failed to delete custom evaluation metric", err), nil
	}
	if resp != nil && resp.StatusCode >= 400 {
		return mcp.NewToolResultError(fmt.Sprintf("failed to delete custom evaluation metric: status %d", resp.StatusCode)), nil
	}

	type DeleteCustomMetricResponse struct {
		MetricUUID string `json:"metric_uuid"`
		Status     string `json:"status"`
	}
	response := DeleteCustomMetricResponse{
		MetricUUID: metricUUID,
		Status:     "deleted",
	}
	jsonData, err := json.MarshalIndent(response, "", "  ")
	if err != nil {
		return nil, fmt.Errorf("marshal error: %w", err)
	}
	return mcp.NewToolResultText(string(jsonData)), nil
}

// isModelEvalTerminalStatus checks if a model evaluation run status is terminal.
func isModelEvalTerminalStatus(status ModelEvaluationRunStatus) bool {
	switch status {
	case ModelEvalRunStatusSuccessful,
		ModelEvalRunStatusFailed,
		ModelEvalRunStatusCancelled,
		ModelEvalRunStatusPartiallySuccessful:
		return true
	default:
		return false
	}
}

// runWorkflow orchestrates a complete model evaluation: upload dataset, create run, poll for results.
func (met *ModelEvaluationTool) runWorkflow(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	args := req.GetArguments()

	datasetFilePath, _ := args["dataset_file_path"].(string)
	runName, _ := args["name"].(string)

	if datasetFilePath == "" || runName == "" {
		return mcp.NewToolResultError("dataset_file_path and name are required"), nil
	}
	if strings.TrimSpace(stringArg(args, "candidate_model_name")) == "" {
		return mcp.NewToolResultError(modelEvalCandidateNameRequiredMsg), nil
	}
	if strings.TrimSpace(stringArg(args, "judge_model_name")) == "" {
		return mcp.NewToolResultError(modelEvalJudgeNameRequiredMsg), nil
	}

	timeoutSec := int64(300)
	if t, ok := args["timeout_seconds"].(float64); ok {
		timeoutSec = int64(t)
	}
	pollIntervalSec := int64(5)
	if p, ok := args["poll_interval_seconds"].(float64); ok {
		pollIntervalSec = int64(p)
	}

	// Convert metric UUIDs
	var metricUUIDs []string
	if metricUUIDsRaw, ok := args["metric_uuids"].([]interface{}); ok {
		for _, m := range metricUUIDsRaw {
			if s, ok := m.(string); ok {
				metricUUIDs = append(metricUUIDs, s)
			}
		}
	}

	resolved, selectionResult, err := met.resolveEvalModelsOnly(ctx, args, true)
	if selectionResult != nil || err != nil {
		return selectionResult, err
	}

	workflowCfg := &modelEvalRunConfig{
		RunName:         runName,
		DatasetFilePath: datasetFilePath,
		MetricUUIDs:     append([]string(nil), metricUUIDs...),
		StarMetric:      parseStarMetricArg(args),
	}
	if workflowCfg.StarMetric == nil && len(metricUUIDs) > 0 {
		workflowCfg.StarMetric = defaultStarMetric(metricUUIDs)
	}

	if consentResult, ok := checkModelEvalUserMessage(args, resolved, workflowCfg); !ok {
		return consentResult, nil
	}

	client, err := met.client(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to get DigitalOcean client: %w", err)
	}

	startTime := time.Now()

	if err := validateModelEvaluationDataset(datasetFilePath); err != nil {
		return mcp.NewToolResultErrorFromErr("step 1: dataset validation failed", err), nil
	}

	fileData, err := os.ReadFile(datasetFilePath)
	if err != nil {
		return mcp.NewToolResultErrorFromErr("step 1: failed to read dataset file", err), nil
	}

	fileName := getFileName(datasetFilePath)
	datasetName := runName + "-dataset"

	// Steps 2–4: presign, upload to Spaces, register dataset record.
	datasetResult, err := uploadAndRegisterModelEvaluationDataset(ctx, client, datasetName, fileData, fileName)
	if err != nil {
		return mcp.NewToolResultErrorFromErr("step 2: failed to upload and register dataset", err), nil
	}

	// Step 5: List metrics if none provided
	if len(metricUUIDs) == 0 {
		metricsOutput, _, err := client.GradientAI.ListModelEvaluationMetrics(ctx)
		if err != nil {
			return mcp.NewToolResultErrorFromErr("step 5: failed to list metrics", err), nil
		}

		for _, m := range metricsOutput.Metrics {
			metricUUIDs = append(metricUUIDs, m.MetricUUID)
		}
	}

	// Step 6: Create evaluation run
	createReq := &godo.CreateModelEvaluationRunRequest{
		Name:               runName,
		CandidateModelUUID: resolved.Candidate.UUID,
		CandidateModelName: resolved.Candidate.APIName,
		DatasetUUID:        datasetResult.EvaluationDatasetUUID,
		JudgeModelUUID:     resolved.Judge.UUID,
		MetricUUIDs:        metricUUIDs,
		StarMetric:         toGodoStarMetric(defaultStarMetric(metricUUIDs)),
		Source:             "mcp",
	}

	createReq.CandidateInferenceConfig = godoInferenceConfigFromArgs(args)

	runOutput, _, err := client.GradientAI.CreateModelEvaluationRun(ctx, createReq)
	if err != nil {
		return mcp.NewToolResultErrorFromErr("step 6: failed to create model evaluation run", err), nil
	}

	evalRunUUID := runOutput.EvalRunUuid

	// Step 7: Poll for completion
	timeout := time.Duration(timeoutSec) * time.Second
	pollInterval := time.Duration(pollIntervalSec) * time.Second
	deadline := time.Now().Add(timeout)

	var finalRun *godo.ModelEvaluationRunDetail
	for {
		if time.Now().After(deadline) {
			return mcp.NewToolResultError("step 7: evaluation polling timed out"), nil
		}

		output, _, err := client.GradientAI.GetModelEvaluationRun(ctx, evalRunUUID, nil)
		if err != nil {
			return mcp.NewToolResultErrorFromErr("step 7: failed to poll evaluation run", err), nil
		}

		finalRun = output.Run
		if finalRun == nil {
			return mcp.NewToolResultError("step 7: evaluation run missing from API response"), nil
		}

		if isGodoModelEvalRunTerminal(finalRun.Status) {
			break
		}

		select {
		case <-ctx.Done():
			return mcp.NewToolResultError("workflow cancelled"), nil
		case <-time.After(pollInterval):
		}
	}

	duration := time.Since(startTime).Seconds()

	type WorkflowResponse struct {
		EvalRunUUID     string                                `json:"eval_run_uuid"`
		Status          string                                `json:"status"`
		ResultSummary   *godo.ModelEvaluationRunResultSummary `json:"result_summary,omitempty"`
		DurationSeconds float64                               `json:"duration_seconds"`
		ErrorMessage    string                                `json:"error_message,omitempty"`
	}

	response := WorkflowResponse{
		EvalRunUUID:     evalRunUUID,
		Status:          string(finalRun.Status),
		ResultSummary:   finalRun.ResultSummary,
		DurationSeconds: duration,
		ErrorMessage:    finalRun.ErrorDescription,
	}

	jsonData, err := json.MarshalIndent(response, "", "  ")
	if err != nil {
		return nil, fmt.Errorf("marshal error: %w", err)
	}

	return mcp.NewToolResultText(string(jsonData)), nil
}

// Tools returns the list of server tools for model evaluation management.
func (met *ModelEvaluationTool) Tools() []server.ServerTool {
	return []server.ServerTool{
		{
			Handler: met.listMetrics,
			Tool: mcp.NewTool(
				"genai-model-eval-list-metrics",
				common.WithHints(common.HintsRead),
				common.WithRisk(common.RiskLow),
				mcp.WithDescription("List all available model evaluation metrics."),
			),
		},
		{
			Handler: met.listDatasets,
			Tool: mcp.NewTool(
				"genai-model-eval-list-datasets",
				common.WithHints(common.HintsRead),
				common.WithRisk(common.RiskLow),
				mcp.WithDescription("List previously uploaded evaluation datasets so you can reuse an existing dataset's UUID in genai-model-eval-create-run. Defaults to model-evaluation datasets. Each item includes dataset_uuid, dataset_name, created_at, row_count, file_size, and has_ground_truth. Use this to find the dataset_uuid for a dataset the user already uploaded (instead of uploading a new one)."),
				mcp.WithString("dataset_type", mcp.Description("Filter by dataset type. Defaults to EVALUATION_DATASET_TYPE_MODEL (datasets usable for model evaluation). Other values: EVALUATION_DATASET_TYPE_UNKNOWN, EVALUATION_DATASET_TYPE_ADK, EVALUATION_DATASET_TYPE_NON_ADK.")),
			),
		},
		{
			Handler: met.listPresets,
			Tool: mcp.NewTool(
				"genai-model-eval-list-presets",
				common.WithHints(common.HintsRead),
				common.WithRisk(common.RiskLow),
				mcp.WithDescription("List all model evaluation presets. Presets are reusable evaluation configurations containing a dataset, judge model, and metrics."),
			),
		},
		{
			Handler: met.getPreset,
			Tool: mcp.NewTool(
				"genai-model-eval-get-preset",
				common.WithHints(common.HintsRead),
				common.WithRisk(common.RiskLow),
				mcp.WithDescription("Get a single model evaluation preset by UUID."),
				mcp.WithString("eval_preset_uuid", mcp.Required(), mcp.Description("UUID of the evaluation preset")),
			),
		},
		{
			Handler: met.createDataset,
			Tool: mcp.NewTool(
				"genai-model-eval-create-dataset",
				common.WithHints(common.HintsCreate),
				common.WithRisk(common.RiskLow),
				mcp.WithDescription("Upload and register a model evaluation dataset (presign → Spaces upload → database record). Accepts .csv (with 'input' column) or .jsonl (one JSON object per line with 'input' field); 'ground_truth' is optional. Returns evaluation_dataset_uuid for use with genai-model-eval-create-run."),
				mcp.WithString("name", mcp.Required(), mcp.Description("Name for the dataset")),
				mcp.WithString("file_path", mcp.Required(), mcp.Description("Path to the .csv or .jsonl dataset file to upload")),
			),
		},
		{
			Handler: met.createRun,
			Tool: mcp.NewTool(
				"genai-model-eval-create-run",
				common.WithHints(common.HintsCreate),
				common.WithRisk(common.RiskMedium),
				mcp.WithDescription(genaiModelEvalCreateRunToolDescription),
				mcp.WithString("name", mcp.Required(), mcp.Description("Name for this evaluation run")),
				mcp.WithString("candidate_model_name", mcp.Required(), mcp.Description("Exact candidate model name the user provided or confirmed (character-for-character, whitespace trimmed). Partial names return nearest matches only.")),
				mcp.WithString("candidate_model_uuid", mcp.Description("Exact full candidate model UUID (8-4-4-4-12 hex). Optional if name is exact; partial uuids return matches only.")),
				mcp.WithString("eval_preset_uuid", mcp.Description("UUID of a preset to use (optional; if provided, dataset/judge/metrics come from the preset)")),
				mcp.WithString("dataset_uuid", mcp.Description("Dataset UUID to evaluate against (required if not using a preset). Get it from genai-model-eval-create-dataset (returns evaluation_dataset_uuid) or, for an already-uploaded dataset, from genai-model-eval-list-datasets.")),
				mcp.WithString("judge_model_name", mcp.Description("Exact judge model name (required for inline configuration without a preset). Partial names return nearest matches only.")),
				mcp.WithString("judge_model_uuid", mcp.Description("Exact full judge model UUID. Optional if judge_model_name is exact; partial uuids return matches only.")),
				mcp.WithArray("metric_uuids", mcp.Description("Array of metric UUID strings to evaluate (required if not using a preset). Get UUIDs from genai-model-eval-list-metrics. Pass plain UUID strings, e.g. [\"<metric-uuid-1>\", \"<metric-uuid-2>\"], not metric objects."), mcp.Items(map[string]any{"type": "string"})),
				mcp.WithObject("star_metric", mcp.Description("Primary success metric: metric_uuid and optional success_threshold_pct")),
				mcp.WithObject("candidate_inference_config", mcp.Description("Inference parameters: max_tokens (int), temperature (float), top_p (float)")),
				mcp.WithString("source", mcp.Description("Source identifier for this run (e.g., 'mcp')")),
				mcp.WithString("preset_name", mcp.Description("Name for a new preset; providing this saves the run's configuration as a reusable preset")),
				mcp.WithString("candidate_model_source", mcp.Description("Source of the candidate model")),
				mcp.WithString("user_message", mcp.Description(genaiModelEvalUserMessageDescription)),
			),
		},
		{
			Handler: met.listRuns,
			Tool: mcp.NewTool(
				"genai-model-eval-list-runs",
				common.WithHints(common.HintsRead),
				common.WithRisk(common.RiskLow),
				mcp.WithDescription("List model evaluation runs with optional filters. Each run includes its eval_run_uuid (use it with genai-model-eval-get-run / cancel-run / delete-run), name, status, and the candidate/judge model and dataset it used."),
				mcp.WithString("eval_preset_uuid", mcp.Description("Filter by preset UUID")),
				mcp.WithString("status", mcp.Description("Filter by run status. Accepts the full enum values: MODEL_EVALUATION_RUN_QUEUED, MODEL_EVALUATION_RUN_RUNNING_DATASET, MODEL_EVALUATION_RUN_EVALUATING_RESULTS, MODEL_EVALUATION_RUN_CANCELLING, MODEL_EVALUATION_RUN_CANCELLED, MODEL_EVALUATION_RUN_SUCCESSFUL, MODEL_EVALUATION_RUN_PARTIALLY_SUCCESSFUL (some rows scored, others failed), MODEL_EVALUATION_RUN_FAILED.")),
				mcp.WithNumber("page", mcp.Description("Page number for pagination (default: 1)")),
				mcp.WithNumber("per_page", mcp.Description("Results per page for pagination (default: 20)")),
			),
		},
		{
			Handler: met.getRun,
			Tool: mcp.NewTool(
				"genai-model-eval-get-run",
				common.WithHints(common.HintsRead),
				common.WithRisk(common.RiskLow),
				mcp.WithDescription("Get the status, details, and per-prompt results of a model evaluation run."),
				mcp.WithString("eval_run_uuid", mcp.Required(), mcp.Description("UUID of the evaluation run (the eval_run_uuid returned by genai-model-eval-list-runs or genai-model-eval-create-run)")),
				mcp.WithNumber("page", mcp.Description("Page number for per-prompt results pagination")),
				mcp.WithNumber("per_page", mcp.Description("Results per page for per-prompt results pagination")),
			),
		},
		{
			Handler: met.updateRun,
			Tool: mcp.NewTool(
				"genai-model-eval-update-run",
				common.WithHints(common.HintsToggle),
				common.WithRisk(common.RiskLow),
				mcp.WithDescription("Update a model evaluation run. Currently only the run name can be changed."),
				mcp.WithString("eval_run_uuid", mcp.Required(), mcp.Description("UUID of the evaluation run")),
				mcp.WithString("name", mcp.Required(), mcp.Description("New name for the evaluation run")),
			),
		},
		{
			Handler: met.getResultsDownloadURL,
			Tool: mcp.NewTool(
				"genai-model-eval-get-results-download-url",
				common.WithHints(common.HintsRead),
				common.WithRisk(common.RiskLow),
				mcp.WithDescription("Get a presigned download URL for the full results of a model evaluation run. The returned URL is short-lived (expires in ~15 minutes) and points to a gzip-compressed JSON (.json.gz) file, so use it promptly."),
				mcp.WithString("eval_run_uuid", mcp.Required(), mcp.Description("UUID of the evaluation run (the eval_run_uuid returned by genai-model-eval-list-runs)")),
			),
		},
		{
			Handler: met.deleteRun,
			Tool: mcp.NewTool(
				"genai-model-eval-delete-run",
				common.WithHints(common.HintsDelete),
				common.WithRisk(common.RiskMedium),
				mcp.WithDescription("Delete a model evaluation run by UUID. Deletion is permanent: the run record and its results cannot be recovered.\n\n"+
					"CONSENT REQUIRED (every delete): Do not call with confirm_deletion: true until the user has explicitly agreed in chat. "+
					"Present the eval_run_uuid and that deletion is permanent; ask for yes/no."),
				mcp.WithString("eval_run_uuid", mcp.Required(), mcp.Description("UUID of the evaluation run to delete (the eval_run_uuid returned by genai-model-eval-list-runs)")),
				mcp.WithBoolean("confirm_deletion", mcp.Required(), mcp.Description(modelEvalDeleteRunConfirmDescription)),
			),
		},
		{
			Handler: met.cancelRun,
			Tool: mcp.NewTool(
				"genai-model-eval-cancel-run",
				common.WithHints(common.HintsToggle),
				common.WithRisk(common.RiskMedium),
				mcp.WithDescription("Cancel an in-progress model evaluation run by UUID. The run transitions to MODEL_EVALUATION_RUN_CANCELLING and then MODEL_EVALUATION_RUN_CANCELLED. Any partial results may be lost.\n\n"+
					"CONSENT REQUIRED (every cancel): Do not call with confirm_cancel: true until the user has explicitly agreed in chat. "+
					"Present the eval_run_uuid and that any partial results may be lost; ask for yes/no."),
				mcp.WithString("eval_run_uuid", mcp.Required(), mcp.Description("UUID of the evaluation run to cancel (the eval_run_uuid returned by genai-model-eval-list-runs)")),
				mcp.WithBoolean("confirm_cancel", mcp.Required(), mcp.Description(modelEvalCancelRunConfirmDescription)),
			),
		},
		{
			Handler: met.deletePreset,
			Tool: mcp.NewTool(
				"genai-model-eval-delete-preset",
				common.WithHints(common.HintsDelete),
				common.WithRisk(common.RiskLow),
				mcp.WithDescription("Delete a saved model evaluation preset by UUID. Deletion is permanent and existing runs that referenced the preset are not affected.\n\n"+
					"CONSENT REQUIRED (every delete): Do not call with confirm_deletion: true until the user has explicitly agreed in chat. "+
					"Present the eval_preset_uuid and that deletion is permanent; ask for yes/no."),
				mcp.WithString("eval_preset_uuid", mcp.Required(), mcp.Description("UUID of the evaluation preset to delete")),
				mcp.WithBoolean("confirm_deletion", mcp.Required(), mcp.Description(modelEvalDeletePresetConfirmDescription)),
			),
		},
		{
			Handler: met.deleteDataset,
			Tool: mcp.NewTool(
				"genai-model-eval-delete-dataset",
				common.WithHints(common.HintsDelete),
				common.WithRisk(common.RiskMedium),
				mcp.WithDescription("Delete an evaluation dataset by UUID. Works for both model and agent evaluation datasets. Deletion is permanent: the dataset record cannot be recovered.\n\n"+
					"CONSENT REQUIRED (every delete): Do not call with confirm_deletion: true until the user has explicitly agreed in chat. "+
					"Present the dataset_uuid and that deletion is permanent; ask for yes/no."),
				mcp.WithString("dataset_uuid", mcp.Required(), mcp.Description("UUID of the evaluation dataset to delete (the dataset_uuid returned by genai-model-eval-list-datasets, or evaluation_dataset_uuid from genai-model-eval-create-dataset)")),
				mcp.WithBoolean("confirm_deletion", mcp.Required(), mcp.Description(modelEvalDeleteDatasetConfirmDescription)),
			),
		},
		{
			Handler: met.createCustomMetric,
			Tool: mcp.NewTool(
				"genai-model-eval-create-custom-metric",
				common.WithHints(common.HintsCreate),
				common.WithRisk(common.RiskLow),
				mcp.WithDescription("Create a custom (LLM-as-judge) model evaluation metric. The judge model scores each response against the scoring_prompt. The created metric appears in genai-model-eval-list-metrics (source EVALUATION_METRIC_SOURCE_CUSTOM) and its metric_uuid can be used in evaluation runs and presets."),
				mcp.WithString("metric_name", mcp.Required(), mcp.Description("Display name for the custom metric")),
				mcp.WithString("scoring_prompt", mcp.Required(), mcp.Description("LLM-as-judge scoring prompt describing how the judge model should score a response")),
				mcp.WithString("description", mcp.Description("Human-readable description of what the metric measures")),
				mcp.WithBoolean("requires_ground_truth", mcp.Description("Whether scoring this metric requires a ground_truth column in the evaluation dataset (default: false)")),
			),
		},
		{
			Handler: met.updateCustomMetric,
			Tool: mcp.NewTool(
				"genai-model-eval-update-custom-metric",
				common.WithHints(common.HintsToggle),
				common.WithRisk(common.RiskLow),
				mcp.WithDescription("Update an existing custom model evaluation metric. Only custom metrics (source EVALUATION_METRIC_SOURCE_CUSTOM) can be updated; built-in catalog metrics cannot. Provide at least one field to change."),
				mcp.WithString("metric_uuid", mcp.Required(), mcp.Description("UUID of the custom metric to update (get it from genai-model-eval-list-metrics or genai-model-eval-create-custom-metric)")),
				mcp.WithString("metric_name", mcp.Description("New display name for the metric")),
				mcp.WithString("description", mcp.Description("New description of what the metric measures")),
				mcp.WithString("scoring_prompt", mcp.Description("New LLM-as-judge scoring prompt")),
				mcp.WithBoolean("requires_ground_truth", mcp.Description("Whether scoring this metric requires a ground_truth column in the evaluation dataset")),
			),
		},
		{
			Handler: met.deleteCustomMetric,
			Tool: mcp.NewTool(
				"genai-model-eval-delete-custom-metric",
				common.WithHints(common.HintsDelete),
				common.WithRisk(common.RiskLow),
				mcp.WithDescription("Delete a custom model evaluation metric by UUID. Only custom metrics (source EVALUATION_METRIC_SOURCE_CUSTOM) can be deleted; built-in catalog metrics cannot. After deletion the metric is no longer available for new evaluation runs; completed runs keep their results.\n\n"+
					"CONSENT REQUIRED (every delete): Do not call with confirm_deletion: true until the user has explicitly agreed in chat. "+
					"Present the metric_uuid and that the metric will no longer be usable in new runs; ask for yes/no."),
				mcp.WithString("metric_uuid", mcp.Required(), mcp.Description("UUID of the custom metric to delete (get it from genai-model-eval-list-metrics)")),
				mcp.WithBoolean("confirm_deletion", mcp.Required(), mcp.Description(modelEvalDeleteCustomMetricConfirmDescription)),
			),
		},
		{
			Handler: met.runWorkflow,
			Tool: mcp.NewTool(
				"genai-model-eval-run-workflow",
				common.WithHints(common.HintsAction),
				common.WithRisk(common.RiskMedium),
				mcp.WithDescription(genaiModelEvalWorkflowToolDescription),
				mcp.WithString("dataset_file_path", mcp.Required(), mcp.Description("Path to the .csv or .jsonl evaluation dataset")),
				mcp.WithString("name", mcp.Required(), mcp.Description("Name for the evaluation run")),
				mcp.WithString("candidate_model_name", mcp.Required(), mcp.Description("Exact candidate model name. Partial names return nearest matches only.")),
				mcp.WithString("candidate_model_uuid", mcp.Description("Exact full candidate model UUID. Optional if name is exact.")),
				mcp.WithString("judge_model_name", mcp.Required(), mcp.Description("Exact judge model name. Partial names return nearest matches only.")),
				mcp.WithString("judge_model_uuid", mcp.Description("Exact full judge model UUID. Optional if judge_model_name is exact.")),
				mcp.WithArray("metric_uuids", mcp.Description("Array of metric UUID strings to evaluate (if empty, all available metrics are used). Get UUIDs from genai-model-eval-list-metrics. Pass plain UUID strings, not metric objects."), mcp.Items(map[string]any{"type": "string"})),
				mcp.WithObject("candidate_inference_config", mcp.Description("Inference parameters: max_tokens (int), temperature (float), top_p (float)")),
				mcp.WithNumber("timeout_seconds", mcp.Description("Timeout for polling evaluation results in seconds (default: 300)")),
				mcp.WithNumber("poll_interval_seconds", mcp.Description("Interval between status polls in seconds (default: 5)")),
				mcp.WithString("user_message", mcp.Description(genaiModelEvalUserMessageDescription)),
			),
		},
	}
}
