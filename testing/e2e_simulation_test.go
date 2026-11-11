//go:build integration

package testing

import (
	"testing"

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

// TestSimulationListRuns calls genai-simulation-list-runs against the live API.
func TestSimulationListRuns(t *testing.T) {
	t.Parallel()

	type simulationRun struct {
		RunUUID string `json:"run_uuid"`
		Name    string `json:"name"`
		Status  string `json:"status"`
	}
	type listResponse struct {
		SimulationRuns []simulationRun `json:"simulation_runs"`
		Count          int             `json:"count"`
	}

	out := callTool[listResponse](t, "genai-simulation-list-runs", map[string]any{})
	require.GreaterOrEqual(t, out.Count, 0)
	require.Len(t, out.SimulationRuns, out.Count)
	t.Logf("listed %d simulation run(s)", out.Count)
}
