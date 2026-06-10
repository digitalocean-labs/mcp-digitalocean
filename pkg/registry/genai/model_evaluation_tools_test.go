package genai

import (
	"context"
	"testing"

	"github.com/digitalocean/godo"
	"github.com/mark3labs/mcp-go/mcp"
	"github.com/stretchr/testify/require"
)

func setupModelEvalToolWithFailingClient() *ModelEvaluationTool {
	client := func(ctx context.Context) (*godo.Client, error) {
		return nil, context.Canceled
	}
	return NewModelEvaluationTool(client)
}

func TestModelEvaluationTool_Tools(t *testing.T) {
	tool := NewModelEvaluationTool(func(ctx context.Context) (*godo.Client, error) {
		return &godo.Client{}, nil
	})

	tools := tool.Tools()
	require.Len(t, tools, 13, "should have 13 model evaluation tools")

	expectedTools := map[string]bool{
		"genai-model-eval-list-metrics":             false,
		"genai-model-eval-list-datasets":            false,
		"genai-model-eval-list-presets":             false,
		"genai-model-eval-get-preset":               false,
		"genai-model-eval-create-dataset":           false,
		"genai-model-eval-create-run":               false,
		"genai-model-eval-list-runs":                false,
		"genai-model-eval-get-run":                  false,
		"genai-model-eval-get-results-download-url": false,
		"genai-model-eval-delete-run":               false,
		"genai-model-eval-cancel-run":               false,
		"genai-model-eval-delete-preset":            false,
		"genai-model-eval-run-workflow":             false,
	}

	for _, st := range tools {
		name := st.Tool.Name
		_, ok := expectedTools[name]
		require.True(t, ok, "unexpected tool name: %s", name)
		expectedTools[name] = true
	}

	for name, found := range expectedTools {
		require.True(t, found, "missing expected tool: %s", name)
	}
}

func TestModelEvaluationTool_listMetrics_clientError(t *testing.T) {
	tool := setupModelEvalToolWithFailingClient()
	req := mcp.CallToolRequest{Params: mcp.CallToolParams{Arguments: map[string]any{}}}
	_, err := tool.listMetrics(context.Background(), req)
	require.Error(t, err)
}

func TestModelEvaluationTool_listPresets_clientError(t *testing.T) {
	tool := setupModelEvalToolWithFailingClient()
	req := mcp.CallToolRequest{Params: mcp.CallToolParams{Arguments: map[string]any{}}}
	_, err := tool.listPresets(context.Background(), req)
	require.Error(t, err)
}

func TestModelEvaluationTool_getPreset_missingUUID(t *testing.T) {
	tool := setupModelEvalToolWithFailingClient()

	tests := []struct {
		name string
		args map[string]any
	}{
		{name: "missing eval_preset_uuid", args: map[string]any{}},
		{name: "empty eval_preset_uuid", args: map[string]any{"eval_preset_uuid": ""}},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			req := mcp.CallToolRequest{Params: mcp.CallToolParams{Arguments: tc.args}}
			resp, err := tool.getPreset(context.Background(), req)
			require.NoError(t, err)
			require.NotNil(t, resp)
			require.True(t, resp.IsError)
		})
	}
}

func TestModelEvaluationTool_getPreset_clientError(t *testing.T) {
	tool := setupModelEvalToolWithFailingClient()
	req := mcp.CallToolRequest{Params: mcp.CallToolParams{Arguments: map[string]any{
		"eval_preset_uuid": "test-uuid",
	}}}
	_, err := tool.getPreset(context.Background(), req)
	require.Error(t, err)
}

func TestModelEvaluationTool_createDataset_validation(t *testing.T) {
	tool := setupModelEvalToolWithFailingClient()

	tests := []struct {
		name string
		args map[string]any
	}{
		{name: "missing name", args: map[string]any{"file_path": "/tmp/test.csv"}},
		{name: "empty name", args: map[string]any{"name": "", "file_path": "/tmp/test.csv"}},
		{name: "missing file_path", args: map[string]any{"name": "test"}},
		{name: "empty file_path", args: map[string]any{"name": "test", "file_path": ""}},
		{name: "unsupported file format", args: map[string]any{"name": "test", "file_path": "/tmp/test.json"}},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			req := mcp.CallToolRequest{Params: mcp.CallToolParams{Arguments: tc.args}}
			resp, err := tool.createDataset(context.Background(), req)
			require.NoError(t, err)
			require.NotNil(t, resp)
			require.True(t, resp.IsError)
		})
	}
}

func TestModelEvaluationTool_createRun_validation(t *testing.T) {
	tool := setupModelEvalToolWithFailingClient()

	tests := []struct {
		name string
		args map[string]any
	}{
		{name: "missing name", args: map[string]any{
			"candidate_model_name": "model",
		}},
		{name: "missing candidate_model_name", args: map[string]any{
			"name": "run1",
		}},
		{name: "empty candidate_model_name", args: map[string]any{
			"name":                 "run1",
			"candidate_model_name": "",
		}},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			req := mcp.CallToolRequest{Params: mcp.CallToolParams{Arguments: tc.args}}
			resp, err := tool.createRun(context.Background(), req)
			require.NoError(t, err)
			require.NotNil(t, resp)
			require.True(t, resp.IsError)
		})
	}
}

func TestModelEvaluationTool_listRuns_clientError(t *testing.T) {
	tool := setupModelEvalToolWithFailingClient()
	req := mcp.CallToolRequest{Params: mcp.CallToolParams{Arguments: map[string]any{}}}
	_, err := tool.listRuns(context.Background(), req)
	require.Error(t, err)
}

func TestModelEvaluationTool_getRun_missingUUID(t *testing.T) {
	tool := setupModelEvalToolWithFailingClient()

	tests := []struct {
		name string
		args map[string]any
	}{
		{name: "missing eval_run_uuid", args: map[string]any{}},
		{name: "empty eval_run_uuid", args: map[string]any{"eval_run_uuid": ""}},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			req := mcp.CallToolRequest{Params: mcp.CallToolParams{Arguments: tc.args}}
			resp, err := tool.getRun(context.Background(), req)
			require.NoError(t, err)
			require.NotNil(t, resp)
			require.True(t, resp.IsError)
		})
	}
}

func TestModelEvaluationTool_getRun_clientError(t *testing.T) {
	tool := setupModelEvalToolWithFailingClient()
	req := mcp.CallToolRequest{Params: mcp.CallToolParams{Arguments: map[string]any{
		"eval_run_uuid": "test-uuid",
	}}}
	_, err := tool.getRun(context.Background(), req)
	require.Error(t, err)
}

func TestModelEvaluationTool_getResultsDownloadURL_missingUUID(t *testing.T) {
	tool := setupModelEvalToolWithFailingClient()

	tests := []struct {
		name string
		args map[string]any
	}{
		{name: "missing eval_run_uuid", args: map[string]any{}},
		{name: "empty eval_run_uuid", args: map[string]any{"eval_run_uuid": ""}},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			req := mcp.CallToolRequest{Params: mcp.CallToolParams{Arguments: tc.args}}
			resp, err := tool.getResultsDownloadURL(context.Background(), req)
			require.NoError(t, err)
			require.NotNil(t, resp)
			require.True(t, resp.IsError)
		})
	}
}

func TestModelEvaluationTool_getResultsDownloadURL_clientError(t *testing.T) {
	tool := setupModelEvalToolWithFailingClient()
	req := mcp.CallToolRequest{Params: mcp.CallToolParams{Arguments: map[string]any{
		"eval_run_uuid": "test-uuid",
	}}}
	_, err := tool.getResultsDownloadURL(context.Background(), req)
	require.Error(t, err)
}

func TestModelEvaluationTool_runWorkflow_validation(t *testing.T) {
	tool := setupModelEvalToolWithFailingClient()

	tests := []struct {
		name string
		args map[string]any
	}{
		{name: "missing all required", args: map[string]any{}},
		{name: "missing dataset_file_path", args: map[string]any{
			"name":                 "run1",
			"candidate_model_name": "model", "judge_model_name": "judge",
		}},
		{name: "missing name", args: map[string]any{
			"dataset_file_path":    "/tmp/test.csv",
			"candidate_model_name": "model", "judge_model_name": "judge",
		}},
		{name: "missing candidate_model_name", args: map[string]any{
			"dataset_file_path": "/tmp/test.csv", "name": "run1",
			"judge_model_name": "judge",
		}},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			req := mcp.CallToolRequest{Params: mcp.CallToolParams{Arguments: tc.args}}
			resp, err := tool.runWorkflow(context.Background(), req)
			require.NoError(t, err)
			require.NotNil(t, resp)
			require.True(t, resp.IsError)
		})
	}
}

func TestModelEvaluationTool_deleteRun_validation(t *testing.T) {
	tool := setupModelEvalToolWithFailingClient()

	tests := []struct {
		name string
		args map[string]any
	}{
		{name: "missing eval_run_uuid", args: map[string]any{"confirm_deletion": true}},
		{name: "empty eval_run_uuid", args: map[string]any{"eval_run_uuid": "", "confirm_deletion": true}},
		{name: "missing confirm_deletion", args: map[string]any{"eval_run_uuid": "test-uuid"}},
		{name: "false confirm_deletion", args: map[string]any{"eval_run_uuid": "test-uuid", "confirm_deletion": false}},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			req := mcp.CallToolRequest{Params: mcp.CallToolParams{Arguments: tc.args}}
			resp, err := tool.deleteRun(context.Background(), req)
			require.NoError(t, err)
			require.NotNil(t, resp)
			require.True(t, resp.IsError)
		})
	}
}

func TestModelEvaluationTool_cancelRun_validation(t *testing.T) {
	tool := setupModelEvalToolWithFailingClient()

	tests := []struct {
		name string
		args map[string]any
	}{
		{name: "missing eval_run_uuid", args: map[string]any{"confirm_cancel": true}},
		{name: "empty eval_run_uuid", args: map[string]any{"eval_run_uuid": "", "confirm_cancel": true}},
		{name: "missing confirm_cancel", args: map[string]any{"eval_run_uuid": "test-uuid"}},
		{name: "false confirm_cancel", args: map[string]any{"eval_run_uuid": "test-uuid", "confirm_cancel": false}},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			req := mcp.CallToolRequest{Params: mcp.CallToolParams{Arguments: tc.args}}
			resp, err := tool.cancelRun(context.Background(), req)
			require.NoError(t, err)
			require.NotNil(t, resp)
			require.True(t, resp.IsError)
		})
	}
}

func TestModelEvaluationTool_deletePreset_validation(t *testing.T) {
	tool := setupModelEvalToolWithFailingClient()

	tests := []struct {
		name string
		args map[string]any
	}{
		{name: "missing eval_preset_uuid", args: map[string]any{"confirm_deletion": true}},
		{name: "empty eval_preset_uuid", args: map[string]any{"eval_preset_uuid": "", "confirm_deletion": true}},
		{name: "missing confirm_deletion", args: map[string]any{"eval_preset_uuid": "test-uuid"}},
		{name: "false confirm_deletion", args: map[string]any{"eval_preset_uuid": "test-uuid", "confirm_deletion": false}},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			req := mcp.CallToolRequest{Params: mcp.CallToolParams{Arguments: tc.args}}
			resp, err := tool.deletePreset(context.Background(), req)
			require.NoError(t, err)
			require.NotNil(t, resp)
			require.True(t, resp.IsError)
		})
	}
}

func TestModelEvaluationTool_deleteRun_clientError(t *testing.T) {
	tool := setupModelEvalToolWithFailingClient()
	req := mcp.CallToolRequest{Params: mcp.CallToolParams{Arguments: map[string]any{
		"eval_run_uuid":    "test-uuid",
		"confirm_deletion": true,
	}}}
	_, err := tool.deleteRun(context.Background(), req)
	require.Error(t, err)
}

func TestModelEvaluationTool_cancelRun_clientError(t *testing.T) {
	tool := setupModelEvalToolWithFailingClient()
	req := mcp.CallToolRequest{Params: mcp.CallToolParams{Arguments: map[string]any{
		"eval_run_uuid":  "test-uuid",
		"confirm_cancel": true,
	}}}
	_, err := tool.cancelRun(context.Background(), req)
	require.Error(t, err)
}

func TestModelEvaluationTool_deletePreset_clientError(t *testing.T) {
	tool := setupModelEvalToolWithFailingClient()
	req := mcp.CallToolRequest{Params: mcp.CallToolParams{Arguments: map[string]any{
		"eval_preset_uuid": "test-uuid",
		"confirm_deletion": true,
	}}}
	_, err := tool.deletePreset(context.Background(), req)
	require.Error(t, err)
}

func TestIsModelEvalTerminalStatus(t *testing.T) {
	terminal := []ModelEvaluationRunStatus{
		ModelEvalRunStatusSuccessful,
		ModelEvalRunStatusFailed,
		ModelEvalRunStatusCancelled,
		ModelEvalRunStatusPartiallySuccessful,
	}
	for _, s := range terminal {
		require.True(t, isModelEvalTerminalStatus(s), "expected %s to be terminal", s)
	}

	nonTerminal := []ModelEvaluationRunStatus{
		ModelEvalRunStatusUnspecified,
		ModelEvalRunStatusQueued,
		ModelEvalRunStatusRunningDataset,
		ModelEvalRunStatusEvaluatingResults,
		ModelEvalRunStatusCancelling,
	}
	for _, s := range nonTerminal {
		require.False(t, isModelEvalTerminalStatus(s), "expected %s to be non-terminal", s)
	}
}
