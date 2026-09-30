package genai

import (
	"bufio"
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"strconv"
	"strings"

	"github.com/digitalocean/godo"
)

// ScenarioSetCreateResult is returned after creating a scenario set from a file upload.
type ScenarioSetCreateResult struct {
	ScenarioSet *godo.ScenarioSet `json:"scenario_set"`
	ObjectKey   string            `json:"object_key,omitempty"`
	FileName    string            `json:"file_name,omitempty"`
	FileSize    int64             `json:"file_size,omitempty"`
}

// validateScenarioSetJSONL checks that a JSONL file has at least one scenario object with a name.
func validateScenarioSetJSONL(filePath string) error {
	if !isJSONLFile(filePath) {
		return fmt.Errorf("file must have .jsonl extension")
	}

	file, err := os.Open(filePath)
	if err != nil {
		return fmt.Errorf("failed to open file: %w", err)
	}
	defer file.Close()

	scanner := bufio.NewScanner(file)
	// Scenario lines can be large; allow up to 1 MiB per line.
	scanner.Buffer(make([]byte, 0, 64*1024), 1024*1024)

	lineNum := 0
	recordCount := 0
	for scanner.Scan() {
		lineNum++
		line := strings.TrimSpace(scanner.Text())
		if line == "" {
			continue
		}

		var record map[string]json.RawMessage
		if err := json.Unmarshal([]byte(line), &record); err != nil {
			return fmt.Errorf("line %d: invalid JSON: %w", lineNum, err)
		}
		nameRaw, ok := record["name"]
		if !ok {
			return fmt.Errorf("line %d: JSON object must contain a 'name' field", lineNum)
		}
		var name string
		if err := json.Unmarshal(nameRaw, &name); err != nil || strings.TrimSpace(name) == "" {
			return fmt.Errorf("line %d: 'name' must be a non-empty string", lineNum)
		}
		recordCount++
	}

	if err := scanner.Err(); err != nil {
		return fmt.Errorf("failed to read JSONL file: %w", err)
	}
	if recordCount == 0 {
		return fmt.Errorf("JSONL file must contain at least one scenario with a 'name' field")
	}
	return nil
}

// uploadAndCreateScenarioSetFromFile presigns, uploads to Spaces, and creates the scenario set record.
func uploadAndCreateScenarioSetFromFile(
	ctx context.Context,
	client *godo.Client,
	name string,
	filePath string,
) (*ScenarioSetCreateResult, error) {
	if err := validateScenarioSetJSONL(filePath); err != nil {
		return nil, err
	}

	fileData, err := os.ReadFile(filePath)
	if err != nil {
		return nil, fmt.Errorf("failed to read file: %w", err)
	}

	fileName := getFileName(filePath)
	fileSize := int64(len(fileData))

	presignedOutput, _, err := client.AgentPlatform.CreateScenarioSetUploadPresignedURLs(ctx, &godo.CreateScenarioSetUploadPresignedURLsRequest{
		Files: []*godo.PresignedUrlFile{
			{
				FileName: fileName,
				FileSize: strconv.FormatInt(fileSize, 10),
			},
		},
	})
	if err != nil {
		return nil, fmt.Errorf("failed to create presigned URL: %w", err)
	}
	if presignedOutput == nil || len(presignedOutput.Uploads) == 0 {
		return nil, fmt.Errorf("no presigned URL returned")
	}

	upload := presignedOutput.Uploads[0]
	uploadReq, err := http.NewRequestWithContext(ctx, http.MethodPut, upload.PresignedURL, bytes.NewReader(fileData))
	if err != nil {
		return nil, fmt.Errorf("failed to create upload request: %w", err)
	}
	uploadReq.ContentLength = fileSize
	uploadReq.Header.Set("Content-Type", "application/jsonl")

	httpResp, err := presignedUploadHTTPClient.Do(uploadReq)
	if err != nil {
		return nil, fmt.Errorf("failed to upload file: %w", err)
	}
	defer httpResp.Body.Close()

	if httpResp.StatusCode != http.StatusOK {
		bodyBytes, _ := io.ReadAll(httpResp.Body)
		return nil, fmt.Errorf("file upload failed with status %d: %s", httpResp.StatusCode, string(bodyBytes))
	}

	scenarioSet, _, err := client.AgentPlatform.CreateScenarioSet(ctx, &godo.CreateScenarioSetRequest{
		Name: name,
		FileUploadScenarioSet: &godo.FileUploadDataSource{
			OriginalFileName: fileName,
			StoredObjectKey:  upload.ObjectKey,
			Size:             strconv.FormatInt(fileSize, 10),
		},
	})
	if err != nil {
		return nil, fmt.Errorf("failed to create scenario set: %w", err)
	}

	return &ScenarioSetCreateResult{
		ScenarioSet: scenarioSet,
		ObjectKey:   upload.ObjectKey,
		FileName:    fileName,
		FileSize:    fileSize,
	}, nil
}

// parseScenariosFromArgs converts a tool scenarios array into godo.Scenario values.
func parseScenariosFromArgs(raw any) ([]*godo.Scenario, error) {
	items, ok := raw.([]any)
	if !ok {
		return nil, fmt.Errorf("scenarios must be an array of objects")
	}
	if len(items) == 0 {
		return nil, fmt.Errorf("scenarios must contain at least one scenario")
	}

	out := make([]*godo.Scenario, 0, len(items))
	for i, item := range items {
		m, ok := item.(map[string]any)
		if !ok {
			return nil, fmt.Errorf("scenarios[%d] must be an object", i)
		}
		name, _ := m["name"].(string)
		name = strings.TrimSpace(name)
		if name == "" {
			return nil, fmt.Errorf("scenarios[%d].name is required", i)
		}
		s := &godo.Scenario{
			Name:        name,
			Description: strings.TrimSpace(stringArg(m, "description")),
			UserPersona: strings.TrimSpace(stringArg(m, "user_persona")),
		}
		if criteria, ok := m["stopping_criteria"].([]any); ok {
			for _, c := range criteria {
				if cs, ok := c.(string); ok && strings.TrimSpace(cs) != "" {
					s.StoppingCriteria = append(s.StoppingCriteria, cs)
				}
			}
		}
		if v, ok := m["max_turns"].(float64); ok && v > 0 {
			s.MaxTurns = uint32(v)
		}
		if v, ok := m["exploration_budget"].(float64); ok && v > 0 {
			s.ExplorationBudget = uint32(v)
		}
		out = append(out, s)
	}
	return out, nil
}

// parseStringSlice converts a tool array argument into []string.
func parseStringSlice(raw any) []string {
	items, ok := raw.([]any)
	if !ok {
		return nil
	}
	out := make([]string, 0, len(items))
	for _, item := range items {
		if s, ok := item.(string); ok && strings.TrimSpace(s) != "" {
			out = append(out, strings.TrimSpace(s))
		}
	}
	return out
}

// parseStarMetricFromArgs builds a godo.StarMetric from a tool object argument.
func parseStarMetricFromArgs(raw any) *godo.StarMetric {
	m, ok := raw.(map[string]any)
	if !ok || m == nil {
		return nil
	}
	metricUUID, _ := m["metric_uuid"].(string)
	metricUUID = strings.TrimSpace(metricUUID)
	if metricUUID == "" {
		return nil
	}
	sm := &godo.StarMetric{MetricUUID: metricUUID}
	if name, ok := m["name"].(string); ok {
		sm.Name = strings.TrimSpace(name)
	}
	if v, ok := m["success_threshold"].(float64); ok {
		f := float32(v)
		sm.SuccessThreshold = &f
	}
	if v, ok := m["success_threshold_pct"].(float64); ok {
		i := int32(v)
		sm.SuccessThresholdPct = &i
	}
	return sm
}
