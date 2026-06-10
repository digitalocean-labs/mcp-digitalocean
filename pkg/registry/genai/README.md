# GenAI Tools

This package provides MCP tools for DigitalOcean's GenAI platform.

## Overview

The package contains two sets of evaluation tools:

**Agent Evaluation** (`genai-evaluation`) — evaluate deployed agents end-to-end:
- List available evaluation metrics
- Manage evaluation datasets (upload CSV files)
- Create and update evaluation test cases
- Run evaluations against agent deployments
- Monitor evaluation run status

**Model Evaluation** (under `genai-evaluation`) — evaluate raw models directly:
- List available model evaluation metrics
- Upload evaluation datasets
- Create and run model evaluation runs
- Download evaluation results
- Monitor model evaluation run status

## Tools

### Atomic Tools (One API Call Each)

#### `genai-list-evaluation-metrics`
Lists all available evaluation metrics that can be used in test cases.

**Arguments:** None

**Returns:** JSON object with array of metrics and metadata

```json
{
  "metrics": [
    {
      "metric_uuid": "...",
      "metric_name": "correctness",
      "metric_type": "...",
      "category": "METRIC_CATEGORY_CORRECTNESS",
      ...
    }
  ],
  "count": 5
}
```

#### `genai-list-evaluation-test-cases`
Lists evaluation test cases for a specific workspace.

**Arguments:**
- `workspace_uuid` (string, optional): Workspace UUID
- `agent_workspace_name` (string, optional): Workspace name

At least one of `workspace_uuid` or `agent_workspace_name` must be provided.

**Returns:** JSON object with array of test cases

```json
{
  "test_cases": [
    {
      "test_case_uuid": "...",
      "name": "my_test",
      "description": "...",
      ...
    }
  ],
  "count": 2
}
```

#### `genai-create-evaluation-dataset`
Creates an evaluation dataset by uploading a CSV file. The file is validated to ensure:
- File extension is `.csv`
- Contains a `query` column
- All query column values are valid JSON objects

**Arguments:**
- `name` (string, required): Name for the dataset
- `file_path` (string, required): Path to the CSV file to upload

**Returns:** JSON object with dataset UUID and metadata

```json
{
  "dataset_uuid": "...",
  "name": "my_dataset",
  "file_size": 1024
}
```

#### `genai-create-evaluation-test-case`
Creates a new evaluation test case.

**Arguments:**
- `name` (string, required): Name of the test case
- `description` (string, optional): Description
- `dataset_uuid` (string, required): Dataset UUID to use
- `metrics` (array of strings, optional): Metric UUIDs to include
- `workspace_uuid` (string, optional): Workspace UUID
- `agent_workspace_name` (string, optional): Workspace name

At least one of `workspace_uuid` or `agent_workspace_name` must be provided.

**Returns:** JSON object with test case UUID

```json
{
  "test_case_uuid": "...",
  "name": "my_test",
  "dataset_uuid": "..."
}
```

#### `genai-update-evaluation-test-case`
Updates an existing evaluation test case.

**Arguments:**
- `test_case_uuid` (string, required): Test case UUID to update
- `name` (string, optional): New name
- `description` (string, optional): New description
- `dataset_uuid` (string, optional): New dataset UUID
- `metrics` (array of strings, optional): New metric UUIDs

**Returns:** JSON object with test case UUID and new version

```json
{
  "test_case_uuid": "...",
  "version": 2
}
```

#### `genai-run-evaluation-test-case`
Runs an evaluation test case against specified agent deployments.

**Arguments:**
- `test_case_uuid` (string, required): Test case UUID to run
- `agent_deployment_names` (array of strings, required): Deployment names to evaluate
- `run_name` (string, required): Name for this evaluation run

**Returns:** JSON object with evaluation run UUIDs

```json
{
  "evaluation_run_uuids": ["uuid1", "uuid2"],
  "count": 2
}
```

#### `genai-get-evaluation-run`
Gets the status and results of an evaluation run.

**Arguments:**
- `evaluation_run_uuid` (string, required): Evaluation run UUID

**Returns:** JSON object with full evaluation run details

```json
{
  "evaluation_run": {
    "evaluation_run_uuid": "...",
    "status": "EVALUATION_RUN_SUCCESSFUL",
    "run_level_metric_results": [
      {
        "metric_name": "correctness",
        "number_value": 0.95,
        "reasoning": "..."
      }
    ],
    ...
  }
}
```

### High-Level Orchestrated Tool

#### `genai-run-evaluation-workflow`
Runs a complete end-to-end evaluation workflow. This tool orchestrates all the steps:
1. Validates the dataset CSV
2. Lists available metrics and filters by category
3. Uploads the dataset
4. Creates or updates the test case
5. Runs the evaluation
6. Polls for results until completion (with configurable timeout)

This tool is ideal for users unfamiliar with the multi-step evaluation process, as it handles all orchestration internally.

**Arguments:**
- `dataset_file_path` (string, required): Path to CSV evaluation dataset
- `workspace_name` (string, required): Agent workspace name
- `test_case_name` (string, required): Name for the test case
- `agent_deployment_names` (array of strings, required): Deployment names to evaluate
- `run_name` (string, required): Name for the evaluation run
- `description` (string, optional): Test case description
- `metric_categories` (array of strings, optional): Filter by metric categories (e.g., `"METRIC_CATEGORY_CORRECTNESS"`, `"METRIC_CATEGORY_SAFETY_AND_SECURITY"`). If empty, all metrics are used.
- `timeout_seconds` (number, optional): Timeout for polling results (default: 300 seconds)
- `poll_interval_seconds` (number, optional): Interval between status polls (default: 5 seconds)

**Returns:** JSON object with complete workflow results

```json
{
  "dataset_uuid": "...",
  "test_case_uuid": "...",
  "evaluation_run_uuid": "...",
  "status": "EVALUATION_RUN_SUCCESSFUL",
  "metric_results": [
    {
      "metric_name": "correctness",
      "number_value": 0.95,
      "reasoning": "..."
    }
  ],
  "duration_seconds": 45.3,
  "error_message": null
}
```

## Dataset CSV Format

Evaluation datasets must be CSV files with:
- A `query` column containing JSON objects as strings
- Additional columns for ground truth, expected outputs, etc. (optional)

Example:
```csv
query,expected_output
"{\"question\": \"What is 2+2?\"}",4
"{\"question\": \"What is the capital of France?\"}","Paris"
```

## Workflow Example

### Using Atomic Tools (Step-by-Step)

```
1. List metrics to see available options:
   genai-list-evaluation-metrics

2. Upload your dataset:
   genai-create-evaluation-dataset
     name: "my_dataset"
     file_path: "/path/to/queries.csv"

3. Create a test case:
   genai-create-evaluation-test-case
     name: "test_my_agent"
     description: "Testing agent correctness"
     dataset_uuid: "<uuid from step 2>"
     metrics: ["<metric_uuid_1>", "<metric_uuid_2>"]
     agent_workspace_name: "my_workspace"

4. Run the evaluation:
   genai-run-evaluation-test-case
     test_case_uuid: "<uuid from step 3>"
     agent_deployment_names: ["my_agent_deployment"]
     run_name: "run_1"

5. Poll for results:
   genai-get-evaluation-run
     evaluation_run_uuid: "<uuid from step 4>"
```

### Using the Orchestrated Workflow Tool (All-in-One)

```
genai-run-evaluation-workflow
  dataset_file_path: "/path/to/queries.csv"
  workspace_name: "my_workspace"
  test_case_name: "test_my_agent"
  agent_deployment_names: ["my_agent_deployment"]
  run_name: "run_1"
  description: "Testing agent correctness"
  metric_categories: ["METRIC_CATEGORY_CORRECTNESS"]
  timeout_seconds: 300
  poll_interval_seconds: 5
```

## CSV Validation

The CSV dataset is validated to ensure:
1. File has `.csv` extension
2. File contains a `query` column
3. All `query` column values are valid JSON
4. File is readable and not empty

If validation fails, a detailed error message is returned describing the issue.

## Error Handling

All tools return structured error messages. Errors from API calls are wrapped with context about which step failed:

```json
{
  "error": "failed to create evaluation dataset: service error"
}
```

For workflow tool, errors include the step number:
```
"step 4: failed to create presigned URL: ..."
"step 7: evaluation polling timed out"
```

## Metric Categories

Available metric categories (when filtering in workflow tool):
- `METRIC_CATEGORY_CORRECTNESS`: Correctness and accuracy metrics
- `METRIC_CATEGORY_USER_OUTCOMES`: User satisfaction and engagement metrics
- `METRIC_CATEGORY_SAFETY_AND_SECURITY`: Safety and security related metrics
- `METRIC_CATEGORY_CONTEXT_QUALITY`: Context and retrieval quality metrics
- `METRIC_CATEGORY_MODEL_FIT`: Model fit and performance metrics

## Evaluation Run Status Values

- `EVALUATION_RUN_QUEUED`: Run is waiting to start
- `EVALUATION_RUN_RUNNING`: Run is currently executing
- `EVALUATION_RUN_RUNNING_DATASET`: Processing dataset
- `EVALUATION_RUN_EVALUATING_RESULTS`: Evaluating metric results
- `EVALUATION_RUN_SUCCESSFUL`: Run completed successfully
- `EVALUATION_RUN_PARTIALLY_SUCCESSFUL`: Some metrics were evaluated, others failed
- `EVALUATION_RUN_FAILED`: Run failed completely
- `EVALUATION_RUN_CANCELLED`: Run was cancelled

Terminal statuses: `SUCCESSFUL`, `FAILED`, `CANCELLED`, `PARTIALLY_SUCCESSFUL`

---

# Model Evaluation Tools

These tools evaluate raw models directly (not full agent deployments). They use the `/v2/genai/model_evaluation*` API endpoints.

## Key Concepts

- **Candidate Model**: The model being evaluated.
- **Judge Model**: An LLM that scores the candidate model's responses.

## Tools

### Atomic Tools

#### `genai-model-eval-list-metrics`
List all available model evaluation metrics.

**Arguments:** None

**Returns:** JSON object with array of metrics and count

```json
{
  "metrics": [
    {
      "metric_uuid": "...",
      "metric_name": "correctness",
      "category": "METRIC_CATEGORY_CORRECTNESS"
    }
  ],
  "count": 5
}
```

#### `genai-model-eval-list-datasets`
List previously uploaded evaluation datasets so you can reuse an existing dataset's UUID in `genai-model-eval-create-run` (instead of uploading a new one). Defaults to model-evaluation datasets.

**Arguments:**
- `dataset_type` (string, optional): Filter by dataset type. Defaults to `EVALUATION_DATASET_TYPE_MODEL`. Other values: `EVALUATION_DATASET_TYPE_UNKNOWN`, `EVALUATION_DATASET_TYPE_ADK`, `EVALUATION_DATASET_TYPE_NON_ADK`.

**Returns:** JSON object with an array of datasets and a count. Each dataset includes `dataset_uuid`, `dataset_name`, `created_at`, `row_count`, `file_size`, and `has_ground_truth`.

```json
{
  "datasets": [
    {
      "dataset_uuid": "...",
      "dataset_name": "queries.csv",
      "row_count": 20,
      "has_ground_truth": true,
      "created_at": "2025-01-01T00:00:00Z"
    }
  ],
  "count": 1
}
```

> Backed by `GET /v2/gen-ai/evaluation_datasets`. Use the returned `dataset_uuid` directly as the `dataset_uuid` argument of `genai-model-eval-create-run`.

#### `genai-model-eval-create-dataset`
Upload and register a model evaluation dataset (presign → Spaces upload → database record).

**Arguments:**
- `name` (string, required): Name for the dataset
- `file_path` (string, required): Path to a `.csv` or `.jsonl` file to upload. CSV must include an `input` column; JSONL must be one JSON object per line with an `input` field. `ground_truth` is optional in both formats.

**Returns:** JSON object with the registered dataset UUID and upload metadata

```json
{
  "evaluation_dataset_uuid": "...",
  "dataset_uuid": "...",
  "object_key": "...",
  "name": "my_dataset",
  "file_name": "queries.csv",
  "file_size": 1024
}
```

#### `genai-model-eval-create-run`
Create a model evaluation run.

**User confirmation (chat, two steps):** (1) Call without `user_message` — returns a preview with `prompt_for_user`. Post that to the end user and wait for their chat reply. (2) Call again with `user_message` set to their verbatim reply (typically **yes**) and the same arguments. The run is not created until step 2.

**Arguments:**
- `name` (string, required): Name for the evaluation run
- `candidate_model_name` (string, required): Exact candidate model name (partial names return match list)
- `candidate_model_uuid` (string, optional): Exact full candidate UUID (optional when name is exact)
- `eval_preset_uuid` (string, optional): Preset UUID (dataset/judge/metrics from preset; judge name not required)
- `dataset_uuid` (string, required without preset): Dataset UUID. Get it from `genai-model-eval-create-dataset` (returns `evaluation_dataset_uuid`) or, for an already-uploaded dataset, from `genai-model-eval-list-datasets`.
- `judge_model_name` (string, required without preset): Exact judge model name
- `judge_model_uuid` (string, optional): Exact full judge UUID
- `metric_uuids` (array of strings, optional): Metric UUIDs to evaluate
- `star_metric` (object, optional): Primary success metric
- `candidate_inference_config` (object, optional): Inference params (max_tokens, temperature, top_p)
- `user_message` (string, optional): End user's verbatim chat reply after preview (second call; typically `yes`)

**Returns:** JSON object with the evaluation run UUID

```json
{
  "eval_run_uuid": "...",
  "name": "my_eval_run"
}
```

#### `genai-model-eval-list-runs`
List model evaluation runs with optional filters.

**Arguments:**
- `status` (string, optional): Filter by status (e.g., MODEL_EVALUATION_RUN_SUCCESSFUL, FAILED, QUEUED)
- `page` (number, optional): Page number
- `per_page` (number, optional): Results per page

**Returns:** JSON object with array of run summaries

#### `genai-model-eval-get-run`
Get a single model evaluation run with per-prompt results.

**Arguments:**
- `eval_run_uuid` (string, required): UUID of the evaluation run
- `page` (number, optional): Page for per-prompt results
- `per_page` (number, optional): Per-prompt results per page

**Returns:** JSON object with full run detail and per-prompt results

#### `genai-model-eval-get-results-download-url`
Get a presigned download URL for the full results of an evaluation run.

**Arguments:**
- `eval_run_uuid` (string, required): UUID of the evaluation run

**Returns:** JSON object with download URL and expiry

```json
{
  "download_url": "https://...",
  "expires_at": "2025-01-01T00:00:00Z"
}
```

#### `genai-model-eval-delete-run`
Delete a model evaluation run by UUID. Deletion is permanent: the run record and its results cannot be recovered.

**User consent:** `confirm_deletion` must be `true`. Present `eval_run_uuid` and that deletion is permanent, then ask for yes/no in chat. Only set `confirm_deletion: true` after the user explicitly agrees.

**Arguments:**
- `eval_run_uuid` (string, required): UUID of the run to delete
- `confirm_deletion` (boolean, required): Must be true; only after the user has agreed in chat

**Returns:** JSON status from the delete API.

#### `genai-model-eval-cancel-run`
Cancel an in-progress model evaluation run by UUID. The run transitions to `MODEL_EVALUATION_RUN_CANCELLING` and then `MODEL_EVALUATION_RUN_CANCELLED`. Any partial results may be lost.

**User consent:** `confirm_cancel` must be `true`. Present `eval_run_uuid` and that partial results may be lost, then ask for yes/no in chat. Only set `confirm_cancel: true` after the user explicitly agrees.

**Arguments:**
- `eval_run_uuid` (string, required): UUID of the run to cancel
- `confirm_cancel` (boolean, required): Must be true; only after the user has agreed in chat

**Returns:** JSON object with the cancelled run summary.

#### `genai-model-eval-delete-preset`
Delete a saved model evaluation preset by UUID. Existing runs that referenced the preset are not affected.

**User consent:** `confirm_deletion` must be `true`. Present `eval_preset_uuid` and that deletion is permanent, then ask for yes/no in chat. Only set `confirm_deletion: true` after the user explicitly agrees.

**Arguments:**
- `eval_preset_uuid` (string, required): UUID of the preset to delete
- `confirm_deletion` (boolean, required): Must be true; only after the user has agreed in chat

**Returns:** JSON object confirming the deletion.

### Orchestrated Workflow Tool

#### `genai-model-eval-run-workflow`
Run a complete model evaluation workflow: upload dataset, create run, and poll for results.

**User consent:** Same two-step chat confirmation as `genai-model-eval-create-run`.

**Arguments:**
- `dataset_file_path` (string, required): Path to the `.csv` or `.jsonl` evaluation dataset
- `name` (string, required): Name for the evaluation run
- `candidate_model_name` (string, required): Exact candidate model name
- `candidate_model_uuid` (string, optional): Exact full candidate UUID
- `judge_model_name` (string, required): Exact judge model name
- `judge_model_uuid` (string, optional): Exact full judge UUID
- `metric_uuids` (array of strings, optional): Metric UUIDs (if empty, all available metrics are used)
- `candidate_inference_config` (object, optional): Inference params
- `timeout_seconds` (number, optional): Polling timeout (default: 300)
- `poll_interval_seconds` (number, optional): Poll interval (default: 5)
- `user_message` (string, optional): End user's verbatim chat reply after preview (second call; typically `yes`)

**Returns:** JSON object with complete workflow results

```json
{
  "eval_run_uuid": "...",
  "status": "SUCCESSFUL",
  "metric_results": [
    {
      "metric_name": "correctness",
      "number_value": 0.92
    }
  ],
  "duration_seconds": 45.3,
  "error_message": ""
}
```

## Model Evaluation Dataset Format

Model evaluation datasets accept **CSV** or **JSONL**:

**CSV** — header row with `input` column (and optional `ground_truth`):

```csv
input,ground_truth
What is 2+2?,4
What is the capital of France?,Paris
```

**JSONL** — one JSON object per line with an `input` field (and optional `ground_truth`):

```jsonl
{"input":"What is 2+2?","ground_truth":"4"}
{"input":"What is the capital of France?","ground_truth":"Paris"}
```

## Model Evaluation Workflow Examples

### Using Atomic Tools (Step-by-Step)

```
1. Upload dataset:
   genai-model-eval-create-dataset
     name: "my_dataset"
     file_path: "/path/to/queries.csv"

2. List metrics:
   genai-model-eval-list-metrics

3. Create run (first call — preview; post prompt_for_user and wait for user to type yes):
   genai-model-eval-create-run
     name: "eval_run_1"
     candidate_model_name: "Llama 3.3 70B"
     judge_model_name: "GPT-4o"
     dataset_uuid: "<evaluation_dataset_uuid from step 1>"
     metric_uuids: ["<metric-uuid-1>", "<metric-uuid-2>"]

4. Create run (second call — after user types yes):
   genai-model-eval-create-run
     (same arguments as step 3)
     user_message: "yes"

5. Poll for results:
   genai-model-eval-get-run
     eval_run_uuid: "<uuid from step 3>"
```

### Using the Orchestrated Workflow (All-in-One)

```
# First call — preview; post prompt_for_user and wait for yes in chat
genai-model-eval-run-workflow
  dataset_file_path: "/path/to/queries.csv"
  name: "eval_llama_v1"
  candidate_model_name: "Llama 3.3 70B"
  judge_model_name: "GPT-4o"

# Second call — same args plus user_message (verbatim reply from end user)
genai-model-eval-run-workflow
  dataset_file_path: "/path/to/queries.csv"
  name: "eval_llama_v1"
  candidate_model_name: "Llama 3.3 70B"
  judge_model_name: "GPT-4o"
  user_message: "yes"
  timeout_seconds: 300
  poll_interval_seconds: 5
```

## Model Evaluation Run Status Values

- `QUEUED`: Run is waiting to start
- `RUNNING_DATASET`: Processing dataset queries through the candidate model
- `EVALUATING_RESULTS`: Judge model is scoring the responses
- `CANCELLING`: Run cancellation in progress
- `CANCELLED`: Run was cancelled
- `SUCCESSFUL`: Run completed successfully
- `PARTIALLY_SUCCESSFUL`: Some prompts were evaluated, others failed
- `FAILED`: Run failed completely

Terminal statuses: `SUCCESSFUL`, `FAILED`, `CANCELLED`, `PARTIALLY_SUCCESSFUL`

