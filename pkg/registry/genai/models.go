package genai

import "time"

// EvaluationMetricCategory represents a category of evaluation metrics
type EvaluationMetricCategory string

const (
	MetricCategoryUnspecified       EvaluationMetricCategory = "METRIC_CATEGORY_UNSPECIFIED"
	MetricCategoryCorrectness       EvaluationMetricCategory = "METRIC_CATEGORY_CORRECTNESS"
	MetricCategoryUserOutcomes      EvaluationMetricCategory = "METRIC_CATEGORY_USER_OUTCOMES"
	MetricCategorySafetyAndSecurity EvaluationMetricCategory = "METRIC_CATEGORY_SAFETY_AND_SECURITY"
	MetricCategoryContextQuality    EvaluationMetricCategory = "METRIC_CATEGORY_CONTEXT_QUALITY"
	MetricCategoryModelFit          EvaluationMetricCategory = "METRIC_CATEGORY_MODEL_FIT"
)

// EvaluationMetricValueType represents the type of value a metric returns
type EvaluationMetricValueType string

const (
	MetricValueTypeUnspecified EvaluationMetricValueType = "METRIC_VALUE_TYPE_UNSPECIFIED"
	MetricValueTypeNumber      EvaluationMetricValueType = "METRIC_VALUE_TYPE_NUMBER"
	MetricValueTypeString      EvaluationMetricValueType = "METRIC_VALUE_TYPE_STRING"
	MetricValueTypePercentage  EvaluationMetricValueType = "METRIC_VALUE_TYPE_PERCENTAGE"
)

// EvaluationMetricType represents the type of metric
type EvaluationMetricType string

const (
	MetricTypeUnspecified    EvaluationMetricType = "METRIC_TYPE_UNSPECIFIED"
	MetricTypeGeneralQuality EvaluationMetricType = "METRIC_TYPE_GENERAL_QUALITY"
	MetricTypeRAGAndTool     EvaluationMetricType = "METRIC_TYPE_RAG_AND_TOOL"
)

// EvaluationMetric represents an evaluation metric
type EvaluationMetric struct {
	MetricUUID      string                     `json:"metric_uuid"`
	MetricName      string                     `json:"metric_name"`
	Description     *string                    `json:"description,omitempty"`
	MetricType      *EvaluationMetricType      `json:"metric_type,omitempty"`
	MetricValueType *EvaluationMetricValueType `json:"metric_value_type,omitempty"`
	RangeMin        *float64                   `json:"range_min,omitempty"`
	RangeMax        *float64                   `json:"range_max,omitempty"`
	Inverted        *bool                      `json:"inverted,omitempty"`
	Category        *EvaluationMetricCategory  `json:"category,omitempty"`
	IsMetricGoal    *bool                      `json:"is_metric_goal,omitempty"`
	MetricRank      *int                       `json:"metric_rank,omitempty"`
}

// EvaluationDataset represents an evaluation dataset
type EvaluationDataset struct {
	DatasetUUID    string     `json:"dataset_uuid"`
	DatasetName    string     `json:"dataset_name"`
	RowCount       *int       `json:"row_count,omitempty"`
	HasGroundTruth *bool      `json:"has_ground_truth,omitempty"`
	FileSize       *int64     `json:"file_size,omitempty"`
	CreatedAt      *time.Time `json:"created_at,omitempty"`
}

// EvaluationTestCase represents an evaluation test case
type EvaluationTestCase struct {
	TestCaseUUID    string              `json:"test_case_uuid"`
	Name            string              `json:"name"`
	Description     *string             `json:"description,omitempty"`
	Version         int                 `json:"version"`
	Metrics         []*EvaluationMetric `json:"metrics,omitempty"`
	TotalRuns       *int                `json:"total_runs,omitempty"`
	UpdatedByUserID *string             `json:"updated_by_user_id,omitempty"`
	CreatedByUserID *string             `json:"created_by_user_id,omitempty"`
	CreatedAt       *time.Time          `json:"created_at,omitempty"`
	UpdatedAt       *time.Time          `json:"updated_at,omitempty"`
	ArchivedAt      *time.Time          `json:"archived_at,omitempty"`
	Dataset         *EvaluationDataset  `json:"dataset,omitempty"`
}

// CreateEvaluationDatasetFileUploadPresignedUrlsInput input for creating presigned URLs
type CreateEvaluationDatasetFileUploadPresignedUrlsInput struct {
	Files []PresignedUrlFile `json:"files"`
}

// PresignedUrlFile represents a file for presigned URL generation
type PresignedUrlFile struct {
	FileName string `json:"file_name"`
	FileSize int64  `json:"file_size"`
}

// FilePresignedUrlResponse represents a presigned URL response
type FilePresignedUrlResponse struct {
	ObjectKey    string     `json:"object_key"`
	PresignedURL string     `json:"presigned_url"`
	ExpiresAt    *time.Time `json:"expires_at,omitempty"`
}

// CreateEvaluationDatasetFileUploadPresignedUrlsOutput output from presigned URL creation
type CreateEvaluationDatasetFileUploadPresignedUrlsOutput struct {
	RequestID string                     `json:"request_id"`
	Uploads   []FilePresignedUrlResponse `json:"uploads"`
}

// FileUploadDataSource represents a file upload data source
type FileUploadDataSource struct {
	OriginalFileName string `json:"original_file_name"`
	StoredObjectKey  string `json:"stored_object_key"`
	SizeInBytes      int64  `json:"size_in_bytes"`
}

// Evaluation dataset types for POST /evaluation_datasets and the dataset_type
// filter on GET /evaluation_datasets.
const (
	EvaluationDatasetTypeUnknown = "EVALUATION_DATASET_TYPE_UNKNOWN"
	EvaluationDatasetTypeADK     = "EVALUATION_DATASET_TYPE_ADK"
	EvaluationDatasetTypeNonADK  = "EVALUATION_DATASET_TYPE_NON_ADK"
	EvaluationDatasetTypeModel   = "EVALUATION_DATASET_TYPE_MODEL"
)

// ModelEvalDatasetListItem is a single dataset returned by GET /evaluation_datasets.
// Matches the apiEvaluationDataset schema (file_size is a string per the API spec).
type ModelEvalDatasetListItem struct {
	DatasetUUID    string `json:"dataset_uuid"`
	DatasetName    string `json:"dataset_name"`
	DatasetType    string `json:"dataset_type,omitempty"`
	CreatedAt      string `json:"created_at,omitempty"`
	FileSize       string `json:"file_size,omitempty"`
	RowCount       int64  `json:"row_count,omitempty"`
	HasGroundTruth bool   `json:"has_ground_truth,omitempty"`
}

// ListEvaluationDatasetsOutput is the response from GET /evaluation_datasets.
type ListEvaluationDatasetsOutput struct {
	EvaluationDatasets []*ModelEvalDatasetListItem `json:"evaluation_datasets"`
}

// CreateEvaluationDatasetInput input for creating a dataset
type CreateEvaluationDatasetInput struct {
	Name                 string               `json:"name"`
	DatasetType          string               `json:"dataset_type,omitempty"`
	FileUploadDataSource FileUploadDataSource `json:"file_upload_dataset"`
}

// CreateEvaluationDatasetOutput output from dataset creation
type CreateEvaluationDatasetOutput struct {
	EvaluationDatasetUUID string `json:"evaluation_dataset_uuid"`
}

// StarMetric is the primary success metric for an evaluation test case (required by the GenAI API).
type StarMetric struct {
	MetricUUID          string   `json:"metric_uuid"`
	Name                *string  `json:"name,omitempty"`
	SuccessThreshold    *float64 `json:"success_threshold,omitempty"`
	SuccessThresholdPct *float64 `json:"success_threshold_pct,omitempty"`
}

// CreateEvaluationTestCaseInput input for creating a test case
type CreateEvaluationTestCaseInput struct {
	Name               string      `json:"name"`
	Description        string      `json:"description"`
	DatasetUUID        string      `json:"dataset_uuid"`
	Metrics            []string    `json:"metrics,omitempty"`
	StarMetric         *StarMetric `json:"star_metric,omitempty"`
	WorkspaceUUID      *string     `json:"workspace_uuid,omitempty"`
	AgentWorkspaceName *string     `json:"agent_workspace_name,omitempty"`
}

// CreateEvaluationTestCaseOutput output from test case creation
type CreateEvaluationTestCaseOutput struct {
	TestCaseUUID string `json:"test_case_uuid"`
}

// UpdateEvaluationTestCaseInput input for updating a test case
type UpdateEvaluationTestCaseInput struct {
	TestCaseUUID string      `json:"test_case_uuid"`
	Name         *string     `json:"name,omitempty"`
	Description  *string     `json:"description,omitempty"`
	DatasetUUID  *string     `json:"dataset_uuid,omitempty"`
	Metrics      []string    `json:"metrics,omitempty"`
	StarMetric   *StarMetric `json:"star_metric,omitempty"`
}

// UpdateEvaluationTestCaseOutput output from test case update
type UpdateEvaluationTestCaseOutput struct {
	TestCaseUUID string `json:"test_case_uuid"`
	Version      int    `json:"version"`
}

// RunEvaluationTestCaseInput input for running a test case
type RunEvaluationTestCaseInput struct {
	TestCaseUUID         string   `json:"test_case_uuid"`
	AgentUUIDs           []string `json:"agent_uuids,omitempty"`
	AgentDeploymentNames []string `json:"agent_deployment_names,omitempty"`
	RunName              string   `json:"run_name"`
}

// RunEvaluationTestCaseOutput output from test case run
type RunEvaluationTestCaseOutput struct {
	EvaluationRunUUIDs []string `json:"evaluation_run_uuids"`
}

// EvaluationRunStatus represents the status of an evaluation run
type EvaluationRunStatus string

const (
	EvaluationRunStatusUnknown             EvaluationRunStatus = "EVALUATION_RUN_STATUS_UNKNOWN"
	EvaluationRunStatusQueued              EvaluationRunStatus = "EVALUATION_RUN_QUEUED"
	EvaluationRunStatusRunning             EvaluationRunStatus = "EVALUATION_RUN_RUNNING"
	EvaluationRunStatusCompleted           EvaluationRunStatus = "EVALUATION_RUN_COMPLETED"
	EvaluationRunStatusFailed              EvaluationRunStatus = "EVALUATION_RUN_FAILED"
	EvaluationRunStatusCancelled           EvaluationRunStatus = "EVALUATION_RUN_CANCELLED"
	EvaluationRunStatusRunningDataset      EvaluationRunStatus = "EVALUATION_RUN_RUNNING_DATASET"
	EvaluationRunStatusEvaluatingResults   EvaluationRunStatus = "EVALUATION_RUN_EVALUATING_RESULTS"
	EvaluationRunStatusPartiallySuccessful EvaluationRunStatus = "EVALUATION_RUN_PARTIALLY_SUCCESSFUL"
	EvaluationRunStatusSuccessful          EvaluationRunStatus = "EVALUATION_RUN_SUCCESSFUL"
)

// EvaluationMetricResult represents the result of an evaluation metric
type EvaluationMetricResult struct {
	MetricName       string                     `json:"metric_name"`
	NumberValue      *float64                   `json:"number_value,omitempty"`
	StringValue      *string                    `json:"string_value,omitempty"`
	Reasoning        *string                    `json:"reasoning,omitempty"`
	ErrorDescription *string                    `json:"error_description,omitempty"`
	MetricValueType  *EvaluationMetricValueType `json:"metric_value_type,omitempty"`
}

// EvaluationRun represents an evaluation run
type EvaluationRun struct {
	EvaluationRunUUID               string                   `json:"evaluation_run_uuid"`
	TestCaseUUID                    string                   `json:"test_case_uuid"`
	TestCaseVersion                 int                      `json:"test_case_version"`
	TestCaseName                    string                   `json:"test_case_name"`
	TestCaseDescription             *string                  `json:"test_case_description,omitempty"`
	AgentUUID                       string                   `json:"agent_uuid"`
	AgentVersionHash                *string                  `json:"agent_version_hash,omitempty"`
	RunName                         string                   `json:"run_name"`
	Status                          EvaluationRunStatus      `json:"status"`
	StartedAt                       *time.Time               `json:"started_at,omitempty"`
	FinishedAt                      *time.Time               `json:"finished_at,omitempty"`
	PassStatus                      *bool                    `json:"pass_status,omitempty"`
	StarMetricResult                *EvaluationMetricResult  `json:"star_metric_result,omitempty"`
	RunLevelMetricResults           []EvaluationMetricResult `json:"run_level_metric_results,omitempty"`
	AgentName                       *string                  `json:"agent_name,omitempty"`
	AgentWorkspaceUUID              *string                  `json:"agent_workspace_uuid,omitempty"`
	EvaluationTestCaseWorkspaceUUID *string                  `json:"evaluation_test_case_workspace_uuid,omitempty"`
	AgentDeleted                    *bool                    `json:"agent_deleted,omitempty"`
	ErrorDescription                *string                  `json:"error_description,omitempty"`
	CreatedByUserEmail              *string                  `json:"created_by_user_email,omitempty"`
	CreatedByUserID                 *string                  `json:"created_by_user_id,omitempty"`
	QueuedAt                        *time.Time               `json:"queued_at,omitempty"`
	AgentDeploymentName             *string                  `json:"agent_deployment_name,omitempty"`
}

// GetEvaluationRunOutput output from getting an evaluation run
type GetEvaluationRunOutput struct {
	EvaluationRun *EvaluationRun `json:"evaluation_run"`
}

// ListEvaluationTestCasesByWorkspaceOutput output from listing test cases
type ListEvaluationTestCasesByWorkspaceOutput struct {
	EvaluationTestCases []*EvaluationTestCase `json:"evaluation_test_cases,omitempty"`
}

// ListEvaluationMetricsOutput output from listing metrics
type ListEvaluationMetricsOutput struct {
	Metrics []*EvaluationMetric `json:"metrics,omitempty"`
}
