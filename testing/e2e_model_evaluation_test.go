//go:build integration

package testing

import (
	"fmt"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

// TestModelEvalListMetrics calls genai-model-eval-list-metrics against the live GenAI API.
func TestModelEvalListMetrics(t *testing.T) {
	t.Parallel()

	type metric struct {
		MetricUUID string `json:"metric_uuid"`
		MetricName string `json:"metric_name"`
	}
	type listMetricsResponse struct {
		Metrics []metric `json:"metrics"`
		Count   int      `json:"count"`
	}

	out := callTool[listMetricsResponse](t, "genai-model-eval-list-metrics", map[string]any{})

	require.Greater(t, out.Count, 0, "expected at least one model evaluation metric from API")
	require.Len(t, out.Metrics, out.Count)
	require.NotEmpty(t, out.Metrics[0].MetricUUID)
	require.NotEmpty(t, out.Metrics[0].MetricName)
	t.Logf("listed %d model evaluation metric(s)", out.Count)
}

// TestModelEvalListPresets calls genai-model-eval-list-presets against the live GenAI API.
func TestModelEvalListPresets(t *testing.T) {
	t.Parallel()

	type presetSummary struct {
		EvalPresetUUID string `json:"eval_preset_uuid"`
		Name           string `json:"name"`
	}
	type listPresetsResponse struct {
		Presets []presetSummary `json:"presets"`
		Count   int             `json:"count"`
	}

	out := callTool[listPresetsResponse](t, "genai-model-eval-list-presets", map[string]any{})

	require.GreaterOrEqual(t, out.Count, 0)
	require.Len(t, out.Presets, out.Count)
	t.Logf("listed %d model evaluation preset(s)", out.Count)
}

// TestModelEvalGetPreset gets a single preset using a UUID from list-presets.
// Skipped if no presets exist.
func TestModelEvalGetPreset(t *testing.T) {
	t.Parallel()

	type presetSummary struct {
		EvalPresetUUID string `json:"eval_preset_uuid"`
		Name           string `json:"name"`
	}
	type listPresetsResponse struct {
		Presets []presetSummary `json:"presets"`
		Count   int             `json:"count"`
	}

	listOut := callTool[listPresetsResponse](t, "genai-model-eval-list-presets", map[string]any{})
	if listOut.Count == 0 {
		t.Skip("no model evaluation presets available to test get-preset")
	}

	presetUUID := listOut.Presets[0].EvalPresetUUID
	require.NotEmpty(t, presetUUID)

	type presetDetail struct {
		EvalPresetUUID string `json:"eval_preset_uuid"`
		Name           string `json:"name"`
	}

	out := callTool[presetDetail](t, "genai-model-eval-get-preset", map[string]any{
		"eval_preset_uuid": presetUUID,
	})

	require.Equal(t, presetUUID, out.EvalPresetUUID)
	require.NotEmpty(t, out.Name)
	t.Logf("got preset %s (%s)", out.Name, out.EvalPresetUUID)
}

// TestModelEvalListRuns calls genai-model-eval-list-runs against the live GenAI API.
func TestModelEvalListRuns(t *testing.T) {
	t.Parallel()

	type runSummary struct {
		EvalRunUUID string `json:"eval_run_uuid"`
		Name        string `json:"name"`
		Status      string `json:"status"`
	}
	type listRunsResponse struct {
		Runs  []runSummary `json:"runs"`
		Count int          `json:"count"`
	}

	out := callTool[listRunsResponse](t, "genai-model-eval-list-runs", map[string]any{})

	require.GreaterOrEqual(t, out.Count, 0)
	require.Len(t, out.Runs, out.Count)
	t.Logf("listed %d model evaluation run(s)", out.Count)
}

// TestModelEvalListRunsWithPagination tests pagination parameters on list-runs.
func TestModelEvalListRunsWithPagination(t *testing.T) {
	t.Parallel()

	type runSummary struct {
		EvalRunUUID string `json:"eval_run_uuid"`
		Name        string `json:"name"`
		Status      string `json:"status"`
	}
	type listRunsResponse struct {
		Runs  []runSummary `json:"runs"`
		Count int          `json:"count"`
	}

	out := callTool[listRunsResponse](t, "genai-model-eval-list-runs", map[string]any{
		"page":     float64(1),
		"per_page": float64(5),
	})

	require.GreaterOrEqual(t, out.Count, 0)
	require.LessOrEqual(t, out.Count, 5, "per_page=5 should return at most 5 runs")
	require.Len(t, out.Runs, out.Count)
	t.Logf("listed %d model evaluation run(s) with pagination", out.Count)
}

// TestModelEvalGetRun gets a single run using a UUID from list-runs.
// Skipped if no runs exist.
func TestModelEvalGetRun(t *testing.T) {
	t.Parallel()

	type runSummary struct {
		EvalRunUUID string `json:"eval_run_uuid"`
		Name        string `json:"name"`
		Status      string `json:"status"`
	}
	type listRunsResponse struct {
		Runs  []runSummary `json:"runs"`
		Count int          `json:"count"`
	}

	listOut := callTool[listRunsResponse](t, "genai-model-eval-list-runs", map[string]any{
		"page":     float64(1),
		"per_page": float64(1),
	})

	if listOut.Count == 0 {
		t.Skip("no model evaluation runs exist, skipping get-run test")
	}

	runUUID := listOut.Runs[0].EvalRunUUID
	require.NotEmpty(t, runUUID)

	type runDetail struct {
		EvalRunUUID        string `json:"eval_run_uuid"`
		Name               string `json:"name"`
		Status             string `json:"status"`
		CandidateModelUUID string `json:"candidate_model_uuid"`
		CandidateModelName string `json:"candidate_model_name"`
	}
	type getRunResponse struct {
		Run *runDetail `json:"run"`
	}

	out := callTool[getRunResponse](t, "genai-model-eval-get-run", map[string]any{
		"eval_run_uuid": runUUID,
	})

	require.NotNil(t, out.Run, "run detail should not be nil")
	require.Equal(t, runUUID, out.Run.EvalRunUUID)
	require.NotEmpty(t, out.Run.Name)
	require.NotEmpty(t, out.Run.Status)
	t.Logf("got run %s: name=%s status=%s", out.Run.EvalRunUUID, out.Run.Name, out.Run.Status)
}

// TestModelEvalUpdateRun renames an existing run via genai-model-eval-update-run and
// restores the original name afterwards to keep the test non-destructive.
// Skipped if no runs exist.
func TestModelEvalUpdateRun(t *testing.T) {
	t.Parallel()

	type runSummary struct {
		EvalRunUUID string `json:"eval_run_uuid"`
		Name        string `json:"name"`
		Status      string `json:"status"`
	}
	type listRunsResponse struct {
		Runs  []runSummary `json:"runs"`
		Count int          `json:"count"`
	}

	listOut := callTool[listRunsResponse](t, "genai-model-eval-list-runs", map[string]any{
		"page":     float64(1),
		"per_page": float64(1),
	})

	if listOut.Count == 0 {
		t.Skip("no model evaluation runs exist, skipping update-run test")
	}

	runUUID := listOut.Runs[0].EvalRunUUID
	originalName := listOut.Runs[0].Name
	require.NotEmpty(t, runUUID)
	require.NotEmpty(t, originalName)

	newName := fmt.Sprintf("%s-e2e-%d", originalName, time.Now().Unix())

	// The update-run handler returns the run summary directly.
	updated := callTool[runSummary](t, "genai-model-eval-update-run", map[string]any{
		"eval_run_uuid": runUUID,
		"name":          newName,
	})

	require.Equal(t, runUUID, updated.EvalRunUUID)
	require.Equal(t, newName, updated.Name, "run name should be updated")
	t.Logf("updated run %s name: %q -> %q", runUUID, originalName, newName)

	// Restore the original name so repeated runs stay idempotent.
	restored := callTool[runSummary](t, "genai-model-eval-update-run", map[string]any{
		"eval_run_uuid": runUUID,
		"name":          originalName,
	})
	require.Equal(t, originalName, restored.Name, "run name should be restored to original")
	t.Logf("restored run %s name to %q", runUUID, originalName)
}

// TestModelEvalGetResultsDownloadURL gets a download URL for a completed run.
// Skipped if no successful runs exist.
func TestModelEvalGetResultsDownloadURL(t *testing.T) {
	t.Parallel()

	type runSummary struct {
		EvalRunUUID string `json:"eval_run_uuid"`
		Name        string `json:"name"`
		Status      string `json:"status"`
	}
	type listRunsResponse struct {
		Runs  []runSummary `json:"runs"`
		Count int          `json:"count"`
	}

	listOut := callTool[listRunsResponse](t, "genai-model-eval-list-runs", map[string]any{
		"status": "MODEL_EVALUATION_RUN_SUCCESSFUL",
	})

	if listOut.Count == 0 {
		t.Skip("no successful model evaluation runs exist, skipping download URL test")
	}

	runUUID := listOut.Runs[0].EvalRunUUID
	require.NotEmpty(t, runUUID)

	type downloadURLResponse struct {
		DownloadURL string `json:"download_url"`
	}

	out := callTool[downloadURLResponse](t, "genai-model-eval-get-results-download-url", map[string]any{
		"eval_run_uuid": runUUID,
	})

	require.NotEmpty(t, out.DownloadURL, "download URL should not be empty")
	t.Logf("got download URL for run %s", runUUID)
}
