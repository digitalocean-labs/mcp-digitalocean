package genai

import (
	"bufio"
	"bytes"
	"context"
	"encoding/csv"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"strconv"
	"strings"

	"github.com/digitalocean/godo"
)

// ModelEvalDatasetResult is returned after uploading and registering a model evaluation dataset.
type ModelEvalDatasetResult struct {
	EvaluationDatasetUUID string `json:"evaluation_dataset_uuid"`
	DatasetUUID           string `json:"dataset_uuid"`
	ObjectKey             string `json:"object_key"`
	Name                  string `json:"name"`
	FileName              string `json:"file_name"`
	FileSize              int64  `json:"file_size"`
}

// validateModelEvaluationDataset checks that a CSV or JSONL file has the required input field for model evaluation.
func validateModelEvaluationDataset(filePath string) error {
	switch {
	case isCSVFile(filePath):
		return validateModelEvaluationDatasetCSV(filePath)
	case isJSONLFile(filePath):
		return validateModelEvaluationDatasetJSONL(filePath)
	default:
		return fmt.Errorf("file must have .csv or .jsonl extension")
	}
}

func validateModelEvaluationDatasetCSV(filePath string) error {
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

	hasInput := false
	for _, col := range header {
		if col == "input" {
			hasInput = true
			break
		}
	}
	if !hasInput {
		return fmt.Errorf("CSV must contain an 'input' column")
	}

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
		if len(row) == 0 {
			return fmt.Errorf("row %d: empty row", rowNum)
		}
	}

	return nil
}

func validateModelEvaluationDatasetJSONL(filePath string) error {
	file, err := os.Open(filePath)
	if err != nil {
		return fmt.Errorf("failed to open file: %w", err)
	}
	defer file.Close()

	scanner := bufio.NewScanner(file)
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
		if _, ok := record["input"]; !ok {
			return fmt.Errorf("line %d: JSON object must contain an 'input' field", lineNum)
		}
		recordCount++
	}

	if err := scanner.Err(); err != nil {
		return fmt.Errorf("failed to read JSONL file: %w", err)
	}
	if recordCount == 0 {
		return fmt.Errorf("JSONL file must contain at least one record with an 'input' field")
	}

	return nil
}

func modelEvaluationDatasetContentType(fileName string) string {
	if isJSONLFile(fileName) {
		return "application/jsonl"
	}
	return "text/csv"
}

// uploadAndRegisterModelEvaluationDataset presigns, uploads to Spaces, and registers the dataset record.
func uploadAndRegisterModelEvaluationDataset(
	ctx context.Context,
	client *godo.Client,
	name string,
	fileData []byte,
	fileName string,
) (*ModelEvalDatasetResult, error) {
	fileSize := int64(len(fileData))

	presignedInput := &godo.CreateModelEvalDatasetUploadPresignedURLsRequest{
		Files: []*godo.PresignedUrlFile{
			{
				FileName: fileName,
				FileSize: strconv.FormatInt(fileSize, 10),
			},
		},
	}

	presignedOutput, _, err := client.GradientAI.CreateModelEvalDatasetUploadPresignedURLs(ctx, presignedInput)
	if err != nil {
		return nil, fmt.Errorf("failed to create presigned URL: %w", err)
	}

	if presignedOutput == nil || len(presignedOutput.Uploads) == 0 {
		return nil, fmt.Errorf("no presigned URL returned")
	}

	upload := presignedOutput.Uploads[0]

	uploadReq, err := http.NewRequestWithContext(ctx, "PUT", upload.PresignedURL, bytes.NewReader(fileData))
	if err != nil {
		return nil, fmt.Errorf("failed to create upload request: %w", err)
	}
	uploadReq.ContentLength = fileSize
	uploadReq.Header.Set("Content-Type", modelEvaluationDatasetContentType(fileName))

	httpResp, err := presignedUploadHTTPClient.Do(uploadReq)
	if err != nil {
		return nil, fmt.Errorf("failed to upload file: %w", err)
	}
	defer httpResp.Body.Close()

	if httpResp.StatusCode != http.StatusOK {
		bodyBytes, _ := io.ReadAll(httpResp.Body)
		return nil, fmt.Errorf("file upload failed with status %d: %s", httpResp.StatusCode, string(bodyBytes))
	}

	datasetInput := &CreateEvaluationDatasetInput{
		Name:        name,
		DatasetType: EvaluationDatasetTypeModel,
		FileUploadDataSource: FileUploadDataSource{
			OriginalFileName: fileName,
			StoredObjectKey:  upload.ObjectKey,
			SizeInBytes:      fileSize,
		},
	}

	datasetReq, err := client.NewRequest(ctx, http.MethodPost, genAIAPIPath+"/evaluation_datasets", datasetInput)
	if err != nil {
		return nil, fmt.Errorf("failed to create dataset request: %w", err)
	}

	var datasetOutput CreateEvaluationDatasetOutput
	resp, err := client.Do(ctx, datasetReq, &datasetOutput)
	if err != nil {
		return nil, fmt.Errorf("failed to register evaluation dataset: %w", err)
	}
	if resp.StatusCode >= 400 {
		return nil, fmt.Errorf("failed to register evaluation dataset: status %d", resp.StatusCode)
	}

	if datasetOutput.EvaluationDatasetUUID == "" {
		return nil, fmt.Errorf("evaluation dataset registration returned empty UUID")
	}

	return &ModelEvalDatasetResult{
		EvaluationDatasetUUID: datasetOutput.EvaluationDatasetUUID,
		DatasetUUID:           datasetOutput.EvaluationDatasetUUID,
		ObjectKey:             upload.ObjectKey,
		Name:                  name,
		FileName:              fileName,
		FileSize:              fileSize,
	}, nil
}
