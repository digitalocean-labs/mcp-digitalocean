package genai

import (
	"bytes"
	"context"
	"encoding/csv"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"sync"
	"time"

	"github.com/digitalocean/godo"
	"github.com/mark3labs/mcp-go/mcp"
	"github.com/mark3labs/mcp-go/server"
)

// genAIAPIPath is the relative path prefix for GenAI endpoints (same style as godo's "v2/droplets").
// With the default API base (https://api.digitalocean.com/), this resolves to .../v2/gen-ai/...
const genAIAPIPath = "v2/gen-ai"

// presignedUploadHTTPClient performs PUTs to third-party presigned URLs (not the godo client).
var presignedUploadHTTPClient = &http.Client{Timeout: 15 * time.Minute}

// EvaluationService provides helpers for evaluation operations
type EvaluationService struct {
	metricsCache      map[string]*EvaluationMetric
	metricsCacheMutex sync.RWMutex
	metricsExpiry     time.Time
	cacheTTL          time.Duration
}

// NewEvaluationService creates a new EvaluationService instance
func NewEvaluationService() *EvaluationService {
	return &EvaluationService{
		metricsCache: make(map[string]*EvaluationMetric),
		cacheTTL:     5 * time.Minute,
	}
}

// ValidateEvaluationDataset validates a CSV file for evaluation
// Checks: extension, CSV format, 'query' column, JSON in query cells
func (es *EvaluationService) ValidateEvaluationDataset(filePath string) error {
	if !isCSVFile(filePath) {
		return fmt.Errorf("file must have .csv extension")
	}

	file, err := os.Open(filePath)
	if err != nil {
		return fmt.Errorf("failed to open file: %w", err)
	}
	defer file.Close()

	reader := csv.NewReader(file)
	header, err := reader.Read()
	if err != nil {
		return fmt.Errorf("failed to read CSV header: %w", err)
	}

	// Check for 'query' column
	queryColIndex := -1
	for i, col := range header {
		if col == "query" {
			queryColIndex = i
			break
		}
	}

	if queryColIndex == -1 {
		return fmt.Errorf("CSV must contain a 'query' column")
	}

	// Validate each query cell contains valid JSON
	rowNum := 1
	for {
		row, err := reader.Read()
		if err == io.EOF {
			break
		}
		if err != nil {
			return fmt.Errorf("failed to read CSV row %d: %w", rowNum, err)
		}
		rowNum++

		if queryColIndex < len(row) {
			queryValue := row[queryColIndex]
			if queryValue != "" {
				var jsonObj interface{}
				if err := json.Unmarshal([]byte(queryValue), &jsonObj); err != nil {
					return fmt.Errorf("row %d: 'query' column contains invalid JSON: %w", rowNum, err)
				}
			}
		}
	}

	return nil
}

// GetCachedMetrics returns cached metrics if valid, otherwise nil
func (es *EvaluationService) GetCachedMetrics() map[string]*EvaluationMetric {
	es.metricsCacheMutex.RLock()
	defer es.metricsCacheMutex.RUnlock()

	if time.Now().After(es.metricsExpiry) {
		return nil
	}
	return es.metricsCache
}

// SetCachedMetrics caches the metrics
func (es *EvaluationService) SetCachedMetrics(metrics map[string]*EvaluationMetric) {
	es.metricsCacheMutex.Lock()
	defer es.metricsCacheMutex.Unlock()

	es.metricsCache = metrics
	es.metricsExpiry = time.Now().Add(es.cacheTTL)
}

// FilterMetricsByCategory filters metrics by category names
func (es *EvaluationService) FilterMetricsByCategory(metrics []*EvaluationMetric, categories []string) ([]string, error) {
	if len(categories) == 0 {
		// No category filter, return all metric UUIDs
		var uuids []string
		for _, m := range metrics {
			uuids = append(uuids, m.MetricUUID)
		}
		return uuids, nil
	}

	categorySet := make(map[string]struct{})
	for _, cat := range categories {
		categorySet[cat] = struct{}{}
	}

	var selectedUUIDs []string
	for _, m := range metrics {
		if m.Category != nil {
			if _, ok := categorySet[string(*m.Category)]; ok {
				selectedUUIDs = append(selectedUUIDs, m.MetricUUID)
			}
		}
	}

	if len(selectedUUIDs) == 0 {
		return nil, fmt.Errorf("no metrics found matching categories: %v", categories)
	}

	return selectedUUIDs, nil
}

// EvaluationTool provides evaluation management tools
type EvaluationTool struct {
	client  func(ctx context.Context) (*godo.Client, error)
	service *EvaluationService
}

// NewEvaluationTool creates a new EvaluationTool instance
func NewEvaluationTool(client func(ctx context.Context) (*godo.Client, error)) *EvaluationTool {
	return &EvaluationTool{
		client:  client,
		service: NewEvaluationService(),
	}
}

// listEvaluationMetrics lists all available evaluation metrics
func (et *EvaluationTool) listEvaluationMetrics(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	client, err := et.client(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to get DigitalOcean client: %w", err)
	}

	// Make API call
	apiReq, err := client.NewRequest(ctx, http.MethodGet, genAIAPIPath+"/evaluation_metrics", nil)
	if err != nil {
		return mcp.NewToolResultErrorFromErr("failed to create request", err), nil
	}

	var output ListEvaluationMetricsOutput
	resp, err := client.Do(ctx, apiReq, &output)
	if err != nil || resp.StatusCode >= 400 {
		return mcp.NewToolResultErrorFromErr("failed to list evaluation metrics", err), nil
	}

	type MetricsResponse struct {
		Metrics []*EvaluationMetric `json:"metrics"`
		Count   int                 `json:"count"`
	}

	response := MetricsResponse{
		Metrics: output.Metrics,
		Count:   len(output.Metrics),
	}

	jsonData, err := json.MarshalIndent(response, "", "  ")
	if err != nil {
		return nil, fmt.Errorf("marshal error: %w", err)
	}

	// Cache the metrics
	metricsMap := make(map[string]*EvaluationMetric)
	for _, m := range output.Metrics {
		metricsMap[m.MetricUUID] = m
	}
	et.service.SetCachedMetrics(metricsMap)

	return mcp.NewToolResultText(string(jsonData)), nil
}

// listEvaluationTestCases lists evaluation test cases by workspace
func (et *EvaluationTool) listEvaluationTestCases(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	args := req.GetArguments()

	workspaceUUID, _ := args["workspace_uuid"].(string)
	workspaceName, _ := args["agent_workspace_name"].(string)

	if workspaceUUID == "" && workspaceName == "" {
		return mcp.NewToolResultError("Either workspace_uuid or agent_workspace_name is required"), nil
	}

	client, err := et.client(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to get DigitalOcean client: %w", err)
	}

	path := genAIAPIPath + "/evaluation_test_cases"
	q := url.Values{}
	if workspaceUUID != "" {
		q.Set("workspace_uuid", workspaceUUID)
	} else if workspaceName != "" {
		q.Set("agent_workspace_name", workspaceName)
	}
	if enc := q.Encode(); enc != "" {
		path += "?" + enc
	}

	apiReq, err := client.NewRequest(ctx, http.MethodGet, path, nil)
	if err != nil {
		return mcp.NewToolResultErrorFromErr("failed to create request", err), nil
	}

	var output ListEvaluationTestCasesByWorkspaceOutput
	resp, err := client.Do(ctx, apiReq, &output)
	if err != nil || resp.StatusCode >= 400 {
		return mcp.NewToolResultErrorFromErr("failed to list evaluation test cases", err), nil
	}

	type TestCasesResponse struct {
		TestCases []*EvaluationTestCase `json:"test_cases"`
		Count     int                   `json:"count"`
	}

	response := TestCasesResponse{
		TestCases: output.EvaluationTestCases,
		Count:     len(output.EvaluationTestCases),
	}

	jsonData, err := json.MarshalIndent(response, "", "  ")
	if err != nil {
		return nil, fmt.Errorf("marshal error: %w", err)
	}

	return mcp.NewToolResultText(string(jsonData)), nil
}

// createEvaluationDataset creates an evaluation dataset with file upload
func (et *EvaluationTool) createEvaluationDataset(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	args := req.GetArguments()

	name, ok := args["name"].(string)
	if !ok || name == "" {
		return mcp.NewToolResultError("name is required"), nil
	}

	filePath, ok := args["file_path"].(string)
	if !ok || filePath == "" {
		return mcp.NewToolResultError("file_path is required"), nil
	}

	// Validate the dataset
	if err := et.service.ValidateEvaluationDataset(filePath); err != nil {
		return mcp.NewToolResultErrorFromErr("dataset validation failed", err), nil
	}

	client, err := et.client(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to get DigitalOcean client: %w", err)
	}

	// Read file
	fileData, err := os.ReadFile(filePath)
	if err != nil {
		return mcp.NewToolResultErrorFromErr("failed to read file", err), nil
	}

	// Get presigned URL
	fileSize := int64(len(fileData))
	fileName := getFileName(filePath)

	presignedInput := &CreateEvaluationDatasetFileUploadPresignedUrlsInput{
		Files: []PresignedUrlFile{
			{
				FileName: fileName,
				FileSize: fileSize,
			},
		},
	}

	presignedReq, err := client.NewRequest(ctx, http.MethodPost, genAIAPIPath+"/evaluation_datasets/file_upload_presigned_urls", presignedInput)
	if err != nil {
		return mcp.NewToolResultErrorFromErr("failed to create presigned URL request", err), nil
	}

	var presignedOutput CreateEvaluationDatasetFileUploadPresignedUrlsOutput
	resp, err := client.Do(ctx, presignedReq, &presignedOutput)
	if err != nil || resp.StatusCode >= 400 {
		return mcp.NewToolResultErrorFromErr("failed to create presigned URL", err), nil
	}

	if len(presignedOutput.Uploads) == 0 {
		return mcp.NewToolResultError("no presigned URL returned"), nil
	}

	upload := presignedOutput.Uploads[0]

	// Upload file to presigned URL
	uploadReq, err := http.NewRequestWithContext(ctx, "PUT", upload.PresignedURL, bytes.NewReader(fileData))
	if err != nil {
		return mcp.NewToolResultErrorFromErr("failed to create upload request", err), nil
	}
	uploadReq.ContentLength = fileSize
	uploadReq.Header.Set("Content-Type", "text/csv")

	httpResp, err := presignedUploadHTTPClient.Do(uploadReq)
	if err != nil {
		return mcp.NewToolResultErrorFromErr("failed to upload file", err), nil
	}
	defer httpResp.Body.Close()

	if httpResp.StatusCode != http.StatusOK {
		bodyBytes, _ := io.ReadAll(httpResp.Body)
		return mcp.NewToolResultError(fmt.Sprintf("file upload failed with status %d: %s", httpResp.StatusCode, string(bodyBytes))), nil
	}

	// Create dataset
	datasetInput := &CreateEvaluationDatasetInput{
		Name: name,
		FileUploadDataSource: FileUploadDataSource{
			OriginalFileName: fileName,
			StoredObjectKey:  upload.ObjectKey,
			SizeInBytes:      fileSize,
		},
	}

	datasetReq, err := client.NewRequest(ctx, http.MethodPost, genAIAPIPath+"/evaluation_datasets", datasetInput)
	if err != nil {
		return mcp.NewToolResultErrorFromErr("failed to create dataset request", err), nil
	}

	var datasetOutput CreateEvaluationDatasetOutput
	resp, err = client.Do(ctx, datasetReq, &datasetOutput)
	if err != nil || resp.StatusCode >= 400 {
		return mcp.NewToolResultErrorFromErr("failed to create evaluation dataset", err), nil
	}

	type DatasetResponse struct {
		DatasetUUID string `json:"dataset_uuid"`
		Name        string `json:"name"`
		FileSize    int64  `json:"file_size"`
	}

	response := DatasetResponse{
		DatasetUUID: datasetOutput.EvaluationDatasetUUID,
		Name:        name,
		FileSize:    fileSize,
	}

	jsonData, err := json.MarshalIndent(response, "", "  ")
	if err != nil {
		return nil, fmt.Errorf("marshal error: %w", err)
	}

	return mcp.NewToolResultText(string(jsonData)), nil
}

// createEvaluationTestCase creates an evaluation test case
func (et *EvaluationTool) createEvaluationTestCase(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	args := req.GetArguments()

	name, _ := args["name"].(string)
	description, _ := args["description"].(string)
	datasetUUID, _ := args["dataset_uuid"].(string)
	workspaceUUID, _ := args["workspace_uuid"].(string)
	workspaceName, _ := args["agent_workspace_name"].(string)

	// Convert metrics to strings
	metricsRaw, _ := args["metrics"].([]interface{})
	var metrics []string
	for _, m := range metricsRaw {
		if metricStr, ok := m.(string); ok {
			metrics = append(metrics, metricStr)
		}
	}

	if name == "" || datasetUUID == "" {
		return mcp.NewToolResultError("name and dataset_uuid are required"), nil
	}

	if workspaceUUID == "" && workspaceName == "" {
		return mcp.NewToolResultError("Either workspace_uuid or agent_workspace_name is required"), nil
	}

	starMetric := parseStarMetricArg(args)
	if starMetric == nil {
		starMetric = defaultStarMetric(metrics)
	}
	if starMetric == nil {
		return mcp.NewToolResultError("At least one metric UUID is required in 'metrics' (used as the star metric), or provide 'star_metric' with metric_uuid"), nil
	}

	client, err := et.client(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to get DigitalOcean client: %w", err)
	}

	input := &CreateEvaluationTestCaseInput{
		Name:        name,
		Description: description,
		DatasetUUID: datasetUUID,
		Metrics:     metrics,
		StarMetric:  starMetric,
	}
	if workspaceUUID != "" {
		input.WorkspaceUUID = &workspaceUUID
	} else {
		input.AgentWorkspaceName = &workspaceName
	}

	apiReq, err := client.NewRequest(ctx, http.MethodPost, genAIAPIPath+"/evaluation_test_cases", input)
	if err != nil {
		return mcp.NewToolResultErrorFromErr("failed to create request", err), nil
	}

	var output CreateEvaluationTestCaseOutput
	resp, err := client.Do(ctx, apiReq, &output)
	if err != nil || resp.StatusCode >= 400 {
		return mcp.NewToolResultErrorFromErr("failed to create evaluation test case", err), nil
	}

	type TestCaseResponse struct {
		TestCaseUUID string `json:"test_case_uuid"`
		Name         string `json:"name"`
		DatasetUUID  string `json:"dataset_uuid"`
	}

	response := TestCaseResponse{
		TestCaseUUID: output.TestCaseUUID,
		Name:         name,
		DatasetUUID:  datasetUUID,
	}

	jsonData, err := json.MarshalIndent(response, "", "  ")
	if err != nil {
		return nil, fmt.Errorf("marshal error: %w", err)
	}

	return mcp.NewToolResultText(string(jsonData)), nil
}

// updateEvaluationTestCase updates an evaluation test case
func (et *EvaluationTool) updateEvaluationTestCase(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	args := req.GetArguments()

	testCaseUUID, _ := args["test_case_uuid"].(string)

	// Convert metrics to strings
	metricsRaw, _ := args["metrics"].([]interface{})
	var metrics []string
	for _, m := range metricsRaw {
		if metricStr, ok := m.(string); ok {
			metrics = append(metrics, metricStr)
		}
	}

	if testCaseUUID == "" {
		return mcp.NewToolResultError("test_case_uuid is required"), nil
	}

	starMetric := parseStarMetricArg(args)
	if starMetric == nil {
		starMetric = defaultStarMetric(metrics)
	}

	client, err := et.client(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to get DigitalOcean client: %w", err)
	}

	input := &UpdateEvaluationTestCaseInput{
		TestCaseUUID: testCaseUUID,
		Metrics:      metrics,
		StarMetric:   starMetric,
	}
	if v, ok := args["name"].(string); ok && v != "" {
		input.Name = &v
	}
	if v, ok := args["description"].(string); ok && v != "" {
		input.Description = &v
	}
	if v, ok := args["dataset_uuid"].(string); ok && v != "" {
		input.DatasetUUID = &v
	}

	path := genAIAPIPath + "/evaluation_test_cases/" + testCaseUUID
	apiReq, err := client.NewRequest(ctx, http.MethodPut, path, input)
	if err != nil {
		return mcp.NewToolResultErrorFromErr("failed to create request", err), nil
	}

	var output UpdateEvaluationTestCaseOutput
	resp, err := client.Do(ctx, apiReq, &output)
	if err != nil || resp.StatusCode >= 400 {
		return mcp.NewToolResultErrorFromErr("failed to update evaluation test case", err), nil
	}

	type UpdateResponse struct {
		TestCaseUUID string `json:"test_case_uuid"`
		Version      int    `json:"version"`
	}

	response := UpdateResponse{
		TestCaseUUID: output.TestCaseUUID,
		Version:      output.Version,
	}

	jsonData, err := json.MarshalIndent(response, "", "  ")
	if err != nil {
		return nil, fmt.Errorf("marshal error: %w", err)
	}

	return mcp.NewToolResultText(string(jsonData)), nil
}

// runEvaluationTestCase runs an evaluation test case
func (et *EvaluationTool) runEvaluationTestCase(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	args := req.GetArguments()

	testCaseUUID, _ := args["test_case_uuid"].(string)
	runName, _ := args["run_name"].(string)

	// Convert deployment names
	deploymentsRaw, _ := args["agent_deployment_names"].([]interface{})
	var deployments []string
	for _, d := range deploymentsRaw {
		if depStr, ok := d.(string); ok {
			deployments = append(deployments, depStr)
		}
	}

	if testCaseUUID == "" || runName == "" {
		return mcp.NewToolResultError("test_case_uuid and run_name are required"), nil
	}

	client, err := et.client(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to get DigitalOcean client: %w", err)
	}

	input := &RunEvaluationTestCaseInput{
		TestCaseUUID:         testCaseUUID,
		AgentDeploymentNames: deployments,
		RunName:              runName,
	}

	apiReq, err := client.NewRequest(ctx, http.MethodPost, genAIAPIPath+"/evaluation_runs", input)
	if err != nil {
		return mcp.NewToolResultErrorFromErr("failed to create request", err), nil
	}

	var output RunEvaluationTestCaseOutput
	resp, err := client.Do(ctx, apiReq, &output)
	if err != nil || resp.StatusCode >= 400 {
		return mcp.NewToolResultErrorFromErr("failed to run evaluation test case", err), nil
	}

	type RunResponse struct {
		EvaluationRunUUIDs []string `json:"evaluation_run_uuids"`
		Count              int      `json:"count"`
	}

	response := RunResponse{
		EvaluationRunUUIDs: output.EvaluationRunUUIDs,
		Count:              len(output.EvaluationRunUUIDs),
	}

	jsonData, err := json.MarshalIndent(response, "", "  ")
	if err != nil {
		return nil, fmt.Errorf("marshal error: %w", err)
	}

	return mcp.NewToolResultText(string(jsonData)), nil
}

// getEvaluationRun gets evaluation run details
func (et *EvaluationTool) getEvaluationRun(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	args := req.GetArguments()

	runUUID, ok := args["evaluation_run_uuid"].(string)
	if !ok || runUUID == "" {
		return mcp.NewToolResultError("evaluation_run_uuid is required"), nil
	}

	client, err := et.client(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to get DigitalOcean client: %w", err)
	}

	path := genAIAPIPath + "/evaluation_runs/" + runUUID
	apiReq, err := client.NewRequest(ctx, http.MethodGet, path, nil)
	if err != nil {
		return mcp.NewToolResultErrorFromErr("failed to create request", err), nil
	}

	var output GetEvaluationRunOutput
	resp, err := client.Do(ctx, apiReq, &output)
	if err != nil || resp.StatusCode >= 400 {
		return mcp.NewToolResultErrorFromErr("failed to get evaluation run", err), nil
	}

	jsonData, err := json.MarshalIndent(output, "", "  ")
	if err != nil {
		return nil, fmt.Errorf("marshal error: %w", err)
	}

	return mcp.NewToolResultText(string(jsonData)), nil
}

// runEvaluationWorkflow orchestrates the full evaluation workflow
func (et *EvaluationTool) runEvaluationWorkflow(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	args := req.GetArguments()

	datasetFilePath, _ := args["dataset_file_path"].(string)
	workspaceName, _ := args["workspace_name"].(string)
	runName, _ := args["run_name"].(string)
	testCaseName, _ := args["test_case_name"].(string)
	description, _ := args["description"].(string)

	// Parse timeout and poll interval
	timeoutSec := int64(300)
	if t, ok := args["timeout_seconds"].(float64); ok {
		timeoutSec = int64(t)
	}
	pollIntervalSec := int64(5)
	if p, ok := args["poll_interval_seconds"].(float64); ok {
		pollIntervalSec = int64(p)
	}

	// Convert metrics categories
	categoriesRaw, _ := args["metric_categories"].([]interface{})
	var categories []string
	for _, c := range categoriesRaw {
		if catStr, ok := c.(string); ok {
			categories = append(categories, catStr)
		}
	}

	// Convert deployment names
	deploymentsRaw, _ := args["agent_deployment_names"].([]interface{})
	var deployments []string
	for _, d := range deploymentsRaw {
		if depStr, ok := d.(string); ok {
			deployments = append(deployments, depStr)
		}
	}

	if datasetFilePath == "" || workspaceName == "" || runName == "" || testCaseName == "" {
		return mcp.NewToolResultError("dataset_file_path, workspace_name, run_name, and test_case_name are required"), nil
	}

	client, err := et.client(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to get DigitalOcean client: %w", err)
	}

	startTime := time.Now()

	// Step 1: Validate dataset
	if err := et.service.ValidateEvaluationDataset(datasetFilePath); err != nil {
		return mcp.NewToolResultErrorFromErr("step 1: dataset validation failed", err), nil
	}

	// Step 2: List metrics
	metricsReq, err := client.NewRequest(ctx, http.MethodGet, genAIAPIPath+"/evaluation_metrics", nil)
	if err != nil {
		return mcp.NewToolResultErrorFromErr("step 2: failed to create request", err), nil
	}

	var metricsOutput ListEvaluationMetricsOutput
	resp, err := client.Do(ctx, metricsReq, &metricsOutput)
	if err != nil || resp.StatusCode >= 400 {
		return mcp.NewToolResultErrorFromErr("step 2: failed to list metrics", err), nil
	}

	metrics := metricsOutput.Metrics

	// Step 3: Filter metrics by category
	var metricUUIDs []string
	if len(categories) > 0 {
		selectedUUIDs, err := et.service.FilterMetricsByCategory(metrics, categories)
		if err != nil {
			return mcp.NewToolResultErrorFromErr("step 3: metric filtering failed", err), nil
		}
		metricUUIDs = selectedUUIDs
	} else {
		for _, m := range metrics {
			metricUUIDs = append(metricUUIDs, m.MetricUUID)
		}
	}

	// Step 4: Upload dataset
	fileData, err := os.ReadFile(datasetFilePath)
	if err != nil {
		return mcp.NewToolResultErrorFromErr("step 4: failed to read dataset file", err), nil
	}

	fileSize := int64(len(fileData))
	fileName := getFileName(datasetFilePath)

	presignedInput := &CreateEvaluationDatasetFileUploadPresignedUrlsInput{
		Files: []PresignedUrlFile{
			{
				FileName: fileName,
				FileSize: fileSize,
			},
		},
	}

	presignedReq, err := client.NewRequest(ctx, http.MethodPost, genAIAPIPath+"/evaluation_datasets/file_upload_presigned_urls", presignedInput)
	if err != nil {
		return mcp.NewToolResultErrorFromErr("step 4: failed to create presigned URL request", err), nil
	}

	var presignedOutput CreateEvaluationDatasetFileUploadPresignedUrlsOutput
	resp, err = client.Do(ctx, presignedReq, &presignedOutput)
	if err != nil || resp.StatusCode >= 400 {
		return mcp.NewToolResultErrorFromErr("step 4: failed to create presigned URL", err), nil
	}

	if len(presignedOutput.Uploads) == 0 {
		return mcp.NewToolResultError("step 4: no presigned URL returned"), nil
	}

	upload := presignedOutput.Uploads[0]

	uploadReq2, err := http.NewRequestWithContext(ctx, "PUT", upload.PresignedURL, bytes.NewReader(fileData))
	if err != nil {
		return mcp.NewToolResultErrorFromErr("step 4: failed to create upload request", err), nil
	}
	uploadReq2.ContentLength = fileSize
	uploadReq2.Header.Set("Content-Type", "text/csv")

	httpResp, err := presignedUploadHTTPClient.Do(uploadReq2)
	if err != nil {
		return mcp.NewToolResultErrorFromErr("step 4: failed to upload file", err), nil
	}
	defer httpResp.Body.Close()

	if httpResp.StatusCode != http.StatusOK {
		bodyBytes, _ := io.ReadAll(httpResp.Body)
		return mcp.NewToolResultError(fmt.Sprintf("step 4: file upload failed with status %d: %s", httpResp.StatusCode, string(bodyBytes))), nil
	}

	datasetInput := &CreateEvaluationDatasetInput{
		Name: testCaseName + "_dataset",
		FileUploadDataSource: FileUploadDataSource{
			OriginalFileName: fileName,
			StoredObjectKey:  upload.ObjectKey,
			SizeInBytes:      fileSize,
		},
	}

	datasetReq, err := client.NewRequest(ctx, http.MethodPost, genAIAPIPath+"/evaluation_datasets", datasetInput)
	if err != nil {
		return mcp.NewToolResultErrorFromErr("step 4: failed to create dataset request", err), nil
	}

	var datasetOutput CreateEvaluationDatasetOutput
	resp, err = client.Do(ctx, datasetReq, &datasetOutput)
	if err != nil || resp.StatusCode >= 400 {
		return mcp.NewToolResultErrorFromErr("step 4: failed to create evaluation dataset", err), nil
	}

	datasetUUID := datasetOutput.EvaluationDatasetUUID

	// Step 5: Find or create test case
	testCasesQ := url.Values{}
	testCasesQ.Set("agent_workspace_name", workspaceName)
	testCasesReq, err := client.NewRequest(ctx, http.MethodGet, genAIAPIPath+"/evaluation_test_cases?"+testCasesQ.Encode(), nil)
	if err != nil {
		return mcp.NewToolResultErrorFromErr("step 5: failed to create request", err), nil
	}

	var testCasesOutput ListEvaluationTestCasesByWorkspaceOutput
	resp, err = client.Do(ctx, testCasesReq, &testCasesOutput)
	if err != nil || resp.StatusCode >= 400 {
		return mcp.NewToolResultErrorFromErr("step 5: failed to list test cases", err), nil
	}

	var testCaseUUID string
	for _, tc := range testCasesOutput.EvaluationTestCases {
		if tc.Name == testCaseName {
			testCaseUUID = tc.TestCaseUUID
			// Update the test case with new metrics and dataset
			updateInput := &UpdateEvaluationTestCaseInput{
				TestCaseUUID: testCaseUUID,
				DatasetUUID:  &datasetUUID,
				Metrics:      metricUUIDs,
				StarMetric:   defaultStarMetric(metricUUIDs),
			}

			updateReq, err := client.NewRequest(ctx, http.MethodPut, genAIAPIPath+"/evaluation_test_cases/"+testCaseUUID, updateInput)
			if err != nil {
				return mcp.NewToolResultErrorFromErr("step 5: failed to create update request", err), nil
			}

			_, err = client.Do(ctx, updateReq, nil)
			if err != nil {
				return mcp.NewToolResultErrorFromErr("step 5: failed to update test case", err), nil
			}
			break
		}
	}

	if testCaseUUID == "" {
		// Create new test case
		createInput := &CreateEvaluationTestCaseInput{
			Name:               testCaseName,
			Description:        description,
			DatasetUUID:        datasetUUID,
			Metrics:            metricUUIDs,
			StarMetric:         defaultStarMetric(metricUUIDs),
			AgentWorkspaceName: &workspaceName,
		}

		createReq, err := client.NewRequest(ctx, http.MethodPost, genAIAPIPath+"/evaluation_test_cases", createInput)
		if err != nil {
			return mcp.NewToolResultErrorFromErr("step 5: failed to create request", err), nil
		}

		var createOutput CreateEvaluationTestCaseOutput
		resp, err = client.Do(ctx, createReq, &createOutput)
		if err != nil || resp.StatusCode >= 400 {
			return mcp.NewToolResultErrorFromErr("step 5: failed to create test case", err), nil
		}

		testCaseUUID = createOutput.TestCaseUUID
	}

	// Step 6: Run evaluation
	runInput := &RunEvaluationTestCaseInput{
		TestCaseUUID:         testCaseUUID,
		AgentDeploymentNames: deployments,
		RunName:              runName,
	}

	runReq, err := client.NewRequest(ctx, http.MethodPost, genAIAPIPath+"/evaluation_runs", runInput)
	if err != nil {
		return mcp.NewToolResultErrorFromErr("step 6: failed to create request", err), nil
	}

	var runOutput RunEvaluationTestCaseOutput
	resp, err = client.Do(ctx, runReq, &runOutput)
	if err != nil || resp.StatusCode >= 400 {
		return mcp.NewToolResultErrorFromErr("step 6: failed to run evaluation", err), nil
	}

	if len(runOutput.EvaluationRunUUIDs) == 0 {
		return mcp.NewToolResultError("step 6: no evaluation run UUID returned"), nil
	}

	evaluationRunUUID := runOutput.EvaluationRunUUIDs[0]

	// Step 7: Poll for completion
	timeout := time.Duration(timeoutSec) * time.Second
	pollInterval := time.Duration(pollIntervalSec) * time.Second
	deadline := time.Now().Add(timeout)

	var finalRun *EvaluationRun
	for {
		if time.Now().After(deadline) {
			return mcp.NewToolResultError("step 7: evaluation polling timed out"), nil
		}

		getReq, err := client.NewRequest(ctx, http.MethodGet, genAIAPIPath+"/evaluation_runs/"+evaluationRunUUID, nil)
		if err != nil {
			return mcp.NewToolResultErrorFromErr("step 7: failed to create request", err), nil
		}

		var output GetEvaluationRunOutput
		resp, err = client.Do(ctx, getReq, &output)
		if err != nil || resp.StatusCode >= 400 {
			return mcp.NewToolResultErrorFromErr("step 7: failed to poll evaluation run", err), nil
		}

		finalRun = output.EvaluationRun
		if finalRun == nil {
			return mcp.NewToolResultError("step 7: evaluation run missing from API response"), nil
		}

		// Check for terminal status
		if isTerminalStatus(finalRun.Status) {
			break
		}

		select {
		case <-ctx.Done():
			return mcp.NewToolResultError("workflow cancelled"), nil
		case <-time.After(pollInterval):
			// Continue polling
		}
	}

	duration := time.Since(startTime).Seconds()

	// Build response
	type WorkflowResponse struct {
		DatasetUUID       string                   `json:"dataset_uuid"`
		TestCaseUUID      string                   `json:"test_case_uuid"`
		EvaluationRunUUID string                   `json:"evaluation_run_uuid"`
		Status            string                   `json:"status"`
		MetricResults     []map[string]interface{} `json:"metric_results"`
		DurationSeconds   float64                  `json:"duration_seconds"`
		ErrorMessage      string                   `json:"error_message,omitempty"`
	}

	metricResults := []map[string]interface{}{}
	if finalRun.StarMetricResult != nil {
		metricResults = append(metricResults, metricResultToMap(finalRun.StarMetricResult))
	}
	for _, mr := range finalRun.RunLevelMetricResults {
		metricResults = append(metricResults, metricResultToMap(&mr))
	}

	response := WorkflowResponse{
		DatasetUUID:       datasetUUID,
		TestCaseUUID:      testCaseUUID,
		EvaluationRunUUID: evaluationRunUUID,
		Status:            string(finalRun.Status),
		MetricResults:     metricResults,
		DurationSeconds:   duration,
		ErrorMessage:      derefString(finalRun.ErrorDescription),
	}

	jsonData, err := json.MarshalIndent(response, "", "  ")
	if err != nil {
		return nil, fmt.Errorf("marshal error: %w", err)
	}

	return mcp.NewToolResultText(string(jsonData)), nil
}

// Tools returns the list of server tools for evaluation management
func (et *EvaluationTool) Tools() []server.ServerTool {
	return []server.ServerTool{
		{
			Handler: et.listEvaluationMetrics,
			Tool: mcp.NewTool(
				"genai-list-evaluation-metrics",
				mcp.WithDescription("List all available evaluation metrics."),
			),
		},
		{
			Handler: et.listEvaluationTestCases,
			Tool: mcp.NewTool(
				"genai-list-evaluation-test-cases",
				mcp.WithDescription("List evaluation test cases for a workspace."),
				mcp.WithString("workspace_uuid", mcp.Description("Workspace UUID (optional if agent_workspace_name is provided)")),
				mcp.WithString("agent_workspace_name", mcp.Description("Workspace name (optional if workspace_uuid is provided)")),
			),
		},
		{
			Handler: et.createEvaluationDataset,
			Tool: mcp.NewTool(
				"genai-create-evaluation-dataset",
				mcp.WithDescription("Create an evaluation dataset by uploading a CSV file. The file must contain a 'query' column with JSON objects."),
				mcp.WithString("name", mcp.Required(), mcp.Description("Name for the dataset")),
				mcp.WithString("file_path", mcp.Required(), mcp.Description("Path to the CSV file to upload")),
			),
		},
		{
			Handler: et.createEvaluationTestCase,
			Tool: mcp.NewTool(
				"genai-create-evaluation-test-case",
				mcp.WithDescription("Create an evaluation test case."),
				mcp.WithString("name", mcp.Required(), mcp.Description("Name of the test case")),
				mcp.WithString("description", mcp.Description("Description of the test case")),
				mcp.WithString("dataset_uuid", mcp.Required(), mcp.Description("Dataset UUID")),
				mcp.WithArray("metrics", mcp.Description("List of metric UUIDs"), mcp.Items(map[string]any{"type": "string"})),
				mcp.WithObject("star_metric", mcp.Description("Optional primary metric: metric_uuid, optional success_threshold_pct (defaults to first metric at 80% if omitted)")),
				mcp.WithString("workspace_uuid", mcp.Description("Workspace UUID (optional if agent_workspace_name is provided)")),
				mcp.WithString("agent_workspace_name", mcp.Description("Workspace name (optional if workspace_uuid is provided)")),
			),
		},
		{
			Handler: et.updateEvaluationTestCase,
			Tool: mcp.NewTool(
				"genai-update-evaluation-test-case",
				mcp.WithDescription("Update an evaluation test case."),
				mcp.WithString("test_case_uuid", mcp.Required(), mcp.Description("Test case UUID to update")),
				mcp.WithString("name", mcp.Description("New name for the test case")),
				mcp.WithString("description", mcp.Description("New description for the test case")),
				mcp.WithString("dataset_uuid", mcp.Description("New dataset UUID")),
				mcp.WithArray("metrics", mcp.Description("List of metric UUIDs"), mcp.Items(map[string]any{"type": "string"})),
				mcp.WithObject("star_metric", mcp.Description("Optional primary metric object; defaults from metrics when omitted")),
			),
		},
		{
			Handler: et.runEvaluationTestCase,
			Tool: mcp.NewTool(
				"genai-run-evaluation-test-case",
				mcp.WithDescription("Run an evaluation test case."),
				mcp.WithString("test_case_uuid", mcp.Required(), mcp.Description("Test case UUID to run")),
				mcp.WithArray("agent_deployment_names", mcp.Description("List of agent deployment names"), mcp.Items(map[string]any{"type": "string"})),
				mcp.WithString("run_name", mcp.Required(), mcp.Description("Name for this evaluation run")),
			),
		},
		{
			Handler: et.getEvaluationRun,
			Tool: mcp.NewTool(
				"genai-get-evaluation-run",
				mcp.WithDescription("Get the status and results of an evaluation run."),
				mcp.WithString("evaluation_run_uuid", mcp.Required(), mcp.Description("Evaluation run UUID")),
			),
		},
		{
			Handler: et.runEvaluationWorkflow,
			Tool: mcp.NewTool(
				"genai-run-evaluation-workflow",
				mcp.WithDescription("Run a complete evaluation workflow: validate dataset, create/update test case, run evaluation, and poll for results. This is a convenience tool for users unfamiliar with the multi-step evaluation process."),
				mcp.WithString("dataset_file_path", mcp.Required(), mcp.Description("Path to the CSV evaluation dataset")),
				mcp.WithString("workspace_name", mcp.Required(), mcp.Description("Agent workspace name")),
				mcp.WithString("test_case_name", mcp.Required(), mcp.Description("Name for the evaluation test case")),
				mcp.WithArray("agent_deployment_names", mcp.Required(), mcp.Description("List of agent deployment names to evaluate"), mcp.Items(map[string]any{"type": "string"})),
				mcp.WithString("run_name", mcp.Required(), mcp.Description("Name for this evaluation run")),
				mcp.WithString("description", mcp.Description("Description for the test case")),
				mcp.WithArray("metric_categories", mcp.Description("List of metric categories to filter by (e.g., 'METRIC_CATEGORY_CORRECTNESS'). If empty, all metrics are used."), mcp.Items(map[string]any{"type": "string"})),
				mcp.WithNumber("timeout_seconds", mcp.Description("Timeout for polling evaluation results in seconds (default: 300)")),
				mcp.WithNumber("poll_interval_seconds", mcp.Description("Interval between polls in seconds (default: 5)")),
			),
		},
	}
}

// Helper functions

func defaultStarMetric(metricUUIDs []string) *StarMetric {
	if len(metricUUIDs) == 0 {
		return nil
	}
	pct := 80.0
	return &StarMetric{
		MetricUUID:          metricUUIDs[0],
		SuccessThresholdPct: &pct,
	}
}

func parseStarMetricArg(args map[string]interface{}) *StarMetric {
	raw, ok := args["star_metric"].(map[string]interface{})
	if !ok || raw == nil {
		return nil
	}
	sm := &StarMetric{}
	if u, ok := raw["metric_uuid"].(string); ok {
		sm.MetricUUID = u
	}
	if n, ok := raw["name"].(string); ok && n != "" {
		sm.Name = &n
	}
	if v, ok := raw["success_threshold"].(float64); ok {
		sm.SuccessThreshold = &v
	}
	if v, ok := raw["success_threshold_pct"].(float64); ok {
		sm.SuccessThresholdPct = &v
	}
	if sm.MetricUUID == "" {
		return nil
	}
	return sm
}

func isCSVFile(path string) bool {
	return len(path) > 4 && path[len(path)-4:] == ".csv"
}

func isJSONLFile(path string) bool {
	return len(path) > 6 && path[len(path)-6:] == ".jsonl"
}

func getFileName(path string) string {
	for i := len(path) - 1; i >= 0; i-- {
		if path[i] == '/' || path[i] == '\\' {
			return path[i+1:]
		}
	}
	return path
}

func isTerminalStatus(status EvaluationRunStatus) bool {
	switch status {
	case EvaluationRunStatusSuccessful,
		EvaluationRunStatusFailed,
		EvaluationRunStatusCancelled,
		EvaluationRunStatusPartiallySuccessful:
		return true
	default:
		return false
	}
}

func metricResultToMap(mr *EvaluationMetricResult) map[string]interface{} {
	result := map[string]interface{}{
		"metric_name": mr.MetricName,
	}

	if mr.NumberValue != nil {
		result["number_value"] = *mr.NumberValue
	}
	if mr.StringValue != nil {
		result["string_value"] = *mr.StringValue
	}
	if mr.Reasoning != nil {
		result["reasoning"] = *mr.Reasoning
	}
	if mr.ErrorDescription != nil {
		result["error_description"] = *mr.ErrorDescription
	}
	if mr.MetricValueType != nil {
		result["metric_value_type"] = *mr.MetricValueType
	}

	return result
}

func derefString(s *string) string {
	if s == nil {
		return ""
	}
	return *s
}
