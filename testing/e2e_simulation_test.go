//go:build integration

package testing

import (
	"strings"
	"testing"

	"github.com/mark3labs/mcp-go/mcp"
	"github.com/stretchr/testify/require"
)

// TestSimulationListScenarioSets calls genai-simulation-list-scenario-sets against the live API.
func TestSimulationListScenarioSets(t *testing.T) {
	t.Parallel()

	type scenarioSet struct {
		ScenarioSetUUID string `json:"scenario_set_uuid"`
		Name            string `json:"name"`
	}
	type listResponse struct {
		ScenarioSets []scenarioSet `json:"scenario_sets"`
		Count        int           `json:"count"`
	}

	out := callTool[listResponse](t, "genai-simulation-list-scenario-sets", map[string]any{})
	require.GreaterOrEqual(t, out.Count, 0)
	require.Len(t, out.ScenarioSets, out.Count)
	t.Logf("listed %d scenario set(s)", out.Count)
}

// TestSimulationRemovedRunToolsAbsent asserts SimulationRun list/get/update/cancel/delete
// tools are no longer registered, while create-run and model-eval lifecycle tools remain.
func TestSimulationRemovedRunToolsAbsent(t *testing.T) {
	t.Parallel()

	ctx, c := getTestClient(t)
	tools, err := c.ListTools(ctx, mcp.ListToolsRequest{})
	require.NoError(t, err)

	byName := make(map[string]bool, len(tools.Tools))
	for _, tool := range tools.Tools {
		byName[tool.Name] = true
	}

	removed := []string{
		"genai-simulation-list-runs",
		"genai-simulation-get-run",
		"genai-simulation-update-run",
		"genai-simulation-cancel-run",
		"genai-simulation-delete-run",
	}
	for _, name := range removed {
		require.False(t, byName[name], "removed tool still registered: %s", name)
		t.Logf("OK absent: %s", name)
	}

	kept := []string{
		"genai-simulation-create-run",
		"genai-simulation-list-scenario-sets",
		"genai-model-eval-list-runs",
		"genai-model-eval-get-run",
		"genai-model-eval-update-run",
		"genai-model-eval-cancel-run",
		"genai-model-eval-delete-run",
	}
	for _, name := range kept {
		require.True(t, byName[name], "expected tool missing: %s", name)
		t.Logf("OK present: %s", name)
	}
}

// TestSimulationCreateRunRequiresMetrics asserts create-run rejects calls without metric_uuids
// before hitting the API (sim+eval always requires metrics).
func TestSimulationCreateRunRequiresMetrics(t *testing.T) {
	t.Parallel()

	ctx, c := getTestClient(t)
	resp, err := c.CallTool(ctx, mcp.CallToolRequest{
		Params: mcp.CallToolParams{
			Name: "genai-simulation-create-run",
			Arguments: map[string]any{
				"scenario_set_uuid": "00000000-0000-0000-0000-000000000001",
				"agent_uuid":        "00000000-0000-0000-0000-000000000002",
			},
		},
	})
	require.NoError(t, err)
	require.True(t, resp.IsError, "expected error when metric_uuids is omitted")

	var errText string
	if len(resp.Content) > 0 {
		if tc, ok := resp.Content[0].(mcp.TextContent); ok {
			errText = tc.Text
		}
	}
	require.True(t, strings.Contains(errText, "metric_uuids"), "error should mention metric_uuids, got: %s", errText)
	t.Logf("OK create-run rejected missing metrics: %s", errText)
}
