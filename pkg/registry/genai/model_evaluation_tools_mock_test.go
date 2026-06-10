package genai

import (
	"context"
	"errors"
	"net/http"
	"testing"

	"github.com/digitalocean/godo"
	"github.com/mark3labs/mcp-go/mcp"
	"github.com/stretchr/testify/require"
	"go.uber.org/mock/gomock"
)

// setupModelEvalToolWithGradientMock wires a ModelEvaluationTool to a mocked
// godo.GradientAIService so tests can exercise the success paths without hitting
// the real DigitalOcean API.
func setupModelEvalToolWithGradientMock(m godo.GradientAIService) *ModelEvaluationTool {
	client := func(ctx context.Context) (*godo.Client, error) {
		return &godo.Client{GradientAI: m}, nil
	}
	return NewModelEvaluationTool(client)
}

func okResponse(statusCode int) *godo.Response {
	return &godo.Response{Response: &http.Response{StatusCode: statusCode}}
}

func callTool(
	t *testing.T,
	handler func(context.Context, mcp.CallToolRequest) (*mcp.CallToolResult, error),
	args map[string]any,
) *mcp.CallToolResult {
	t.Helper()
	req := mcp.CallToolRequest{Params: mcp.CallToolParams{Arguments: args}}
	resp, err := handler(context.Background(), req)
	require.NoError(t, err)
	require.NotNil(t, resp)
	return resp
}

func resultText(t *testing.T, resp *mcp.CallToolResult) string {
	t.Helper()
	require.NotEmpty(t, resp.Content)
	tc, ok := resp.Content[0].(mcp.TextContent)
	require.True(t, ok, "expected text content")
	return tc.Text
}

func TestModelEvaluationTool_listMetrics_success(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	m := NewMockGradientAIService(ctrl)
	m.EXPECT().ListModelEvaluationMetrics(gomock.Any()).Return(&godo.ModelEvaluationMetricListResponse{
		Metrics: []*godo.EvaluationMetric{
			{MetricUUID: "metric-1", MetricName: "accuracy"},
			{MetricUUID: "metric-2", MetricName: "faithfulness"},
		},
	}, okResponse(http.StatusOK), nil)

	resp := callTool(t, setupModelEvalToolWithGradientMock(m).listMetrics, map[string]any{})

	require.False(t, resp.IsError)
	text := resultText(t, resp)
	require.Contains(t, text, "accuracy")
	require.Contains(t, text, "faithfulness")
	require.Contains(t, text, `"count": 2`)
}

func TestModelEvaluationTool_listMetrics_apiError(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	m := NewMockGradientAIService(ctrl)
	m.EXPECT().ListModelEvaluationMetrics(gomock.Any()).Return(nil, nil, errors.New("boom"))

	resp := callTool(t, setupModelEvalToolWithGradientMock(m).listMetrics, map[string]any{})
	require.True(t, resp.IsError)
}

func TestModelEvaluationTool_listPresets_success(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	m := NewMockGradientAIService(ctrl)
	m.EXPECT().ListModelEvaluationPresets(gomock.Any()).Return(&godo.ModelEvaluationPresetListResponse{
		Presets: []*godo.ModelEvaluationPreset{
			{EvalPresetUuid: "preset-1", DatasetName: "qa-dataset"},
		},
	}, okResponse(http.StatusOK), nil)

	resp := callTool(t, setupModelEvalToolWithGradientMock(m).listPresets, map[string]any{})

	require.False(t, resp.IsError)
	text := resultText(t, resp)
	require.Contains(t, text, "preset-1")
	require.Contains(t, text, "qa-dataset")
	require.Contains(t, text, `"count": 1`)
}

func TestModelEvaluationTool_getPreset_success(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	m := NewMockGradientAIService(ctrl)
	m.EXPECT().GetModelEvaluationPreset(gomock.Any(), "preset-1").Return(&godo.ModelEvaluationPresetGetResponse{
		Preset: &godo.ModelEvaluationPreset{
			EvalPresetUuid: "preset-1",
			JudgeModelName: "gpt-judge",
		},
	}, okResponse(http.StatusOK), nil)

	resp := callTool(t, setupModelEvalToolWithGradientMock(m).getPreset, map[string]any{
		"eval_preset_uuid": "preset-1",
	})

	require.False(t, resp.IsError)
	text := resultText(t, resp)
	require.Contains(t, text, "preset-1")
	require.Contains(t, text, "gpt-judge")
}

func TestModelEvaluationTool_listRuns_success(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	m := NewMockGradientAIService(ctrl)
	m.EXPECT().ListModelEvaluationRuns(gomock.Any(), gomock.Any()).Return(&godo.ModelEvaluationRunListResponse{
		Runs: []*godo.ModelEvaluationRunSummary{
			{EvalRunUuid: "run-1", Name: "nightly-eval", Status: godo.ModelEvaluationRunSuccessful},
		},
	}, okResponse(http.StatusOK), nil)

	resp := callTool(t, setupModelEvalToolWithGradientMock(m).listRuns, map[string]any{
		"status":   "SUCCESSFUL",
		"page":     float64(1),
		"per_page": float64(20),
	})

	require.False(t, resp.IsError)
	text := resultText(t, resp)
	require.Contains(t, text, "run-1")
	require.Contains(t, text, "nightly-eval")
	require.Contains(t, text, `"count": 1`)
}

func TestModelEvaluationTool_getRun_success(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	m := NewMockGradientAIService(ctrl)
	m.EXPECT().GetModelEvaluationRun(gomock.Any(), "run-1", gomock.Any()).Return(&godo.ModelEvaluationRunGetResponse{
		Run: &godo.ModelEvaluationRunDetail{
			CandidateModelName: "candidate-model",
			Status:             godo.ModelEvaluationRunSuccessful,
		},
	}, okResponse(http.StatusOK), nil)

	resp := callTool(t, setupModelEvalToolWithGradientMock(m).getRun, map[string]any{
		"eval_run_uuid": "run-1",
	})

	require.False(t, resp.IsError)
	require.Contains(t, resultText(t, resp), "candidate-model")
}

func TestModelEvaluationTool_getResultsDownloadURL_success(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	m := NewMockGradientAIService(ctrl)
	m.EXPECT().GetModelEvaluationRunResultsDownloadURL(gomock.Any(), "run-1").Return(
		&godo.ModelEvaluationRunResultsDownloadURLResponse{
			DownloadURL: "https://example.com/results.json.gz",
		}, okResponse(http.StatusOK), nil)

	resp := callTool(t, setupModelEvalToolWithGradientMock(m).getResultsDownloadURL, map[string]any{
		"eval_run_uuid": "run-1",
	})

	require.False(t, resp.IsError)
	require.Contains(t, resultText(t, resp), "https://example.com/results.json.gz")
}

func TestModelEvaluationTool_deleteRun_success(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	m := NewMockGradientAIService(ctrl)
	m.EXPECT().DeleteModelEvaluationRun(gomock.Any(), "run-1").Return(
		&godo.ModelEvaluationRunDeleteResponse{Status: godo.DeleteModelEvaluationRunStatusSuccess},
		okResponse(http.StatusOK), nil)

	resp := callTool(t, setupModelEvalToolWithGradientMock(m).deleteRun, map[string]any{
		"eval_run_uuid":    "run-1",
		"confirm_deletion": true,
	})

	require.False(t, resp.IsError)
}

func TestModelEvaluationTool_deleteRun_apiNon2xx(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	m := NewMockGradientAIService(ctrl)
	m.EXPECT().DeleteModelEvaluationRun(gomock.Any(), "run-1").Return(
		&godo.ModelEvaluationRunDeleteResponse{}, okResponse(http.StatusInternalServerError), nil)

	resp := callTool(t, setupModelEvalToolWithGradientMock(m).deleteRun, map[string]any{
		"eval_run_uuid":    "run-1",
		"confirm_deletion": true,
	})

	require.True(t, resp.IsError)
	require.Contains(t, resultText(t, resp), "500")
}

func TestModelEvaluationTool_cancelRun_success(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	m := NewMockGradientAIService(ctrl)
	m.EXPECT().CancelModelEvaluationRun(gomock.Any(), "run-1").Return(
		&godo.ModelEvaluationRunCancelResponse{
			Run: &godo.ModelEvaluationRunSummary{EvalRunUuid: "run-1", Status: godo.ModelEvaluationRunCancelled},
		}, okResponse(http.StatusOK), nil)

	resp := callTool(t, setupModelEvalToolWithGradientMock(m).cancelRun, map[string]any{
		"eval_run_uuid":  "run-1",
		"confirm_cancel": true,
	})

	require.False(t, resp.IsError)
	require.Contains(t, resultText(t, resp), "run-1")
}

func TestModelEvaluationTool_deletePreset_success(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	m := NewMockGradientAIService(ctrl)
	m.EXPECT().DeleteModelEvaluationPreset(gomock.Any(), "preset-1").Return(
		&godo.ModelEvaluationPresetDeleteResponse{}, okResponse(http.StatusOK), nil)

	resp := callTool(t, setupModelEvalToolWithGradientMock(m).deletePreset, map[string]any{
		"eval_preset_uuid": "preset-1",
		"confirm_deletion": true,
	})

	require.False(t, resp.IsError)
	text := resultText(t, resp)
	require.Contains(t, text, "preset-1")
	require.Contains(t, text, "deleted")
}
