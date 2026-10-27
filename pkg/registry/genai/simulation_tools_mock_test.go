package genai

import (
	"context"
	"errors"
	"net/http"
	"testing"

	"github.com/digitalocean/godo"
	"github.com/stretchr/testify/require"
	"go.uber.org/mock/gomock"
)

func setupSimulationToolWithAgentPlatformMock(m godo.AgentPlatformService) *SimulationTool {
	client := func(ctx context.Context) (*godo.Client, error) {
		return &godo.Client{AgentPlatform: m}, nil
	}
	return NewSimulationTool(client)
}

func TestSimulationTool_listScenarioSets_success(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	m := NewMockAgentPlatformService(ctrl)
	m.EXPECT().ListScenarioSets(gomock.Any(), gomock.Any()).Return(&godo.ScenarioSetListResponse{
		ScenarioSets: []*godo.ScenarioSet{
			{ScenarioSetUUID: "ss-1", Name: "checkout"},
		},
	}, okResponse(http.StatusOK), nil)

	resp := callTool(t, setupSimulationToolWithAgentPlatformMock(m).listScenarioSets, map[string]any{})
	require.False(t, resp.IsError)
	text := resultText(t, resp)
	require.Contains(t, text, "ss-1")
	require.Contains(t, text, "checkout")
	require.Contains(t, text, `"count": 1`)
}

func TestSimulationTool_listScenarioSets_apiError(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	m := NewMockAgentPlatformService(ctrl)
	m.EXPECT().ListScenarioSets(gomock.Any(), gomock.Any()).Return(nil, nil, errors.New("boom"))

	resp := callTool(t, setupSimulationToolWithAgentPlatformMock(m).listScenarioSets, map[string]any{})
	require.True(t, resp.IsError)
}

func TestSimulationTool_getScenarioSet_success(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	m := NewMockAgentPlatformService(ctrl)
	m.EXPECT().GetScenarioSet(gomock.Any(), "ss-1").Return(&godo.ScenarioSet{
		ScenarioSetUUID: "ss-1",
		Name:            "checkout",
	}, okResponse(http.StatusOK), nil)

	resp := callTool(t, setupSimulationToolWithAgentPlatformMock(m).getScenarioSet, map[string]any{
		"scenario_set_uuid": "ss-1",
	})
	require.False(t, resp.IsError)
	require.Contains(t, resultText(t, resp), "checkout")
}

func TestSimulationTool_createScenarioSet_inline_success(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	m := NewMockAgentPlatformService(ctrl)
	m.EXPECT().CreateScenarioSet(gomock.Any(), gomock.Any()).Return(&godo.ScenarioSet{
		ScenarioSetUUID: "ss-new",
		Name:            "inline-set",
	}, okResponse(http.StatusCreated), nil)

	resp := callTool(t, setupSimulationToolWithAgentPlatformMock(m).createScenarioSet, map[string]any{
		"name": "inline-set",
		"scenarios": []any{
			map[string]any{"name": "scenario-1", "description": "desc"},
		},
	})
	require.False(t, resp.IsError)
	require.Contains(t, resultText(t, resp), "ss-new")
}

func TestSimulationTool_createRun_success(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	m := NewMockAgentPlatformService(ctrl)
	m.EXPECT().CreateSimulationRun(gomock.Any(), gomock.Any()).DoAndReturn(
		func(ctx context.Context, req *godo.CreateSimulationRunRequest) (*godo.SimulationRun, *godo.Response, error) {
			require.Equal(t, "ss-1", req.ScenarioSetUUID)
			require.NotNil(t, req.AgentConfig)
			require.Equal(t, "agent-1", req.AgentConfig.AgentUUID)
			require.Equal(t, []string{"metric-1"}, req.EvaluationConfig.MetricUUIDs)
			return &godo.SimulationRun{RunUUID: "run-1", Name: "sim-run"}, okResponse(http.StatusCreated), nil
		},
	)

	resp := callTool(t, setupSimulationToolWithAgentPlatformMock(m).createRun, map[string]any{
		"scenario_set_uuid": "ss-1",
		"agent_uuid":        "agent-1",
		"name":              "sim-run",
		"metric_uuids":      []any{"metric-1"},
	})
	require.False(t, resp.IsError)
	require.Contains(t, resultText(t, resp), "run-1")
}

func TestSimulationTool_listEpisodes_success(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	m := NewMockAgentPlatformService(ctrl)
	m.EXPECT().ListSimulationEpisodes(gomock.Any(), "run-1", gomock.Any()).Return(&godo.SimulationEpisodeListResponse{
		Episodes: []*godo.SimulationEpisode{
			{EpisodeUUID: "j-1", Verdict: godo.SimulationEpisodeVerdictSuccess},
		},
	}, okResponse(http.StatusOK), nil)

	resp := callTool(t, setupSimulationToolWithAgentPlatformMock(m).listEpisodes, map[string]any{"run_uuid": "run-1"})
	require.False(t, resp.IsError)
	require.Contains(t, resultText(t, resp), "j-1")
}

func TestSimulationTool_getEpisodeTrajectory_success(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	m := NewMockAgentPlatformService(ctrl)
	m.EXPECT().GetSimulationEpisodeTrajectory(gomock.Any(), "run-1", "j-1").Return(&godo.SimulationTrajectory{
		EpisodeUUID: "j-1",
		TurnCount:   3,
	}, okResponse(http.StatusOK), nil)

	resp := callTool(t, setupSimulationToolWithAgentPlatformMock(m).getEpisodeTrajectory, map[string]any{
		"run_uuid":     "run-1",
		"episode_uuid": "j-1",
	})
	require.False(t, resp.IsError)
	require.Contains(t, resultText(t, resp), `"turn_count": 3`)
}

func TestSimulationTool_listScenarioLibrary_success(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	m := NewMockAgentPlatformService(ctrl)
	m.EXPECT().ListScenarioLibrary(gomock.Any(), gomock.Any()).Return(&godo.ScenarioLibraryListResponse{
		Scenarios: []*godo.ScenarioLibraryEntry{
			{LibraryScenarioUUID: "lib-1", Name: "support"},
		},
	}, okResponse(http.StatusOK), nil)

	resp := callTool(t, setupSimulationToolWithAgentPlatformMock(m).listScenarioLibrary, map[string]any{})
	require.False(t, resp.IsError)
	require.Contains(t, resultText(t, resp), "lib-1")
}
