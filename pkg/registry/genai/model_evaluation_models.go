package genai

// ModelEvaluationRunStatus represents the status of a model evaluation run.
type ModelEvaluationRunStatus string

const (
	ModelEvalRunStatusUnspecified         ModelEvaluationRunStatus = "MODEL_EVALUATION_RUN_STATUS_UNSPECIFIED"
	ModelEvalRunStatusQueued              ModelEvaluationRunStatus = "QUEUED"
	ModelEvalRunStatusRunningDataset      ModelEvaluationRunStatus = "RUNNING_DATASET"
	ModelEvalRunStatusEvaluatingResults   ModelEvaluationRunStatus = "EVALUATING_RESULTS"
	ModelEvalRunStatusCancelling          ModelEvaluationRunStatus = "CANCELLING"
	ModelEvalRunStatusCancelled           ModelEvaluationRunStatus = "CANCELLED"
	ModelEvalRunStatusSuccessful          ModelEvaluationRunStatus = "SUCCESSFUL"
	ModelEvalRunStatusPartiallySuccessful ModelEvaluationRunStatus = "PARTIALLY_SUCCESSFUL"
	ModelEvalRunStatusFailed              ModelEvaluationRunStatus = "FAILED"
)
