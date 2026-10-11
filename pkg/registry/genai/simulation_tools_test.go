package genai

import (
	"context"
	"os"
	"path/filepath"
	"testing"

	"github.com/digitalocean/godo"
	"github.com/mark3labs/mcp-go/mcp"
	"github.com/stretchr/testify/require"
)

func setupSimulationToolWithFailingClient() *SimulationTool {
	client := func(ctx context.Context) (*godo.Client, error) {
		return nil, context.Canceled
	}
	return NewSimulationTool(client)
}

func TestSimulationTool_Tools(t *testing.T) {
	tool := NewSimulationTool(func(ctx context.Context) (*godo.Client, error) {
		return &godo.Client{}, nil
	})

	tools := tool.Tools()
	require.Len(t, tools, 21, "should have 21 simulation tools")

	expectedTools := map[string]bool{
		"genai-simulation-list-scenario-sets":               false,
		"genai-simulation-get-scenario-set":                 false,
		"genai-simulation-create-scenario-set":              false,
		"genai-simulation-generate-scenario-set":            false,
		"genai-simulation-list-scenarios":                   false,
		"genai-simulation-get-scenario-set-download-url":    false,
		"genai-simulation-update-scenario-set":              false,
		"genai-simulation-delete-scenario-set":              false,
		"genai-simulation-list-scenario-library":            false,
		"genai-simulation-list-scenario-library-scenarios":  false,
		"genai-simulation-create-scenario-set-from-library": false,
		"genai-simulation-create-run":                       false,
		"genai-simulation-list-runs":                        false,
		"genai-simulation-get-run":                          false,
		"genai-simulation-update-run":                       false,
		"genai-simulation-cancel-run":                       false,
		"genai-simulation-delete-run":                       false,
		"genai-simulation-list-episodes":                    false,
		"genai-simulation-get-episode":                      false,
		"genai-simulation-get-episode-trajectory":           false,
		"genai-simulation-get-episode-trajectory-url":       false,
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

func TestSimulationTool_listScenarioSets_clientError(t *testing.T) {
	tool := setupSimulationToolWithFailingClient()
	req := mcp.CallToolRequest{Params: mcp.CallToolParams{Arguments: map[string]any{}}}
	_, err := tool.listScenarioSets(context.Background(), req)
	require.Error(t, err)
}

func TestSimulationTool_getScenarioSet_missingUUID(t *testing.T) {
	tool := setupSimulationToolWithFailingClient()
	req := mcp.CallToolRequest{Params: mcp.CallToolParams{Arguments: map[string]any{}}}
	resp, err := tool.getScenarioSet(context.Background(), req)
	require.NoError(t, err)
	require.True(t, resp.IsError)
}

func TestSimulationTool_createScenarioSet_validation(t *testing.T) {
	tool := setupSimulationToolWithFailingClient()

	tests := []struct {
		name string
		args map[string]any
	}{
		{name: "missing name", args: map[string]any{"scenarios": []any{map[string]any{"name": "s1"}}}},
		{name: "neither source", args: map[string]any{"name": "set"}},
		{name: "both sources", args: map[string]any{"name": "set", "file_path": "x.jsonl", "scenarios": []any{map[string]any{"name": "s1"}}}},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := mcp.CallToolRequest{Params: mcp.CallToolParams{Arguments: tt.args}}
			resp, err := tool.createScenarioSet(context.Background(), req)
			require.NoError(t, err)
			require.True(t, resp.IsError)
		})
	}
}

func TestSimulationTool_generateScenarioSet_missingArgs(t *testing.T) {
	tool := setupSimulationToolWithFailingClient()
	req := mcp.CallToolRequest{Params: mcp.CallToolParams{Arguments: map[string]any{"name": "set"}}}
	resp, err := tool.generateScenarioSet(context.Background(), req)
	require.NoError(t, err)
	require.True(t, resp.IsError)
}

func TestSimulationTool_createRun_missingArgs(t *testing.T) {
	tool := setupSimulationToolWithFailingClient()
	req := mcp.CallToolRequest{Params: mcp.CallToolParams{Arguments: map[string]any{"scenario_set_uuid": "ss"}}}
	resp, err := tool.createRun(context.Background(), req)
	require.NoError(t, err)
	require.True(t, resp.IsError)
}

func TestSimulationTool_deleteScenarioSet_consent(t *testing.T) {
	tool := setupSimulationToolWithFailingClient()

	tests := []struct {
		name string
		args map[string]any
	}{
		{name: "missing uuid", args: map[string]any{"confirm_deletion": true}},
		{name: "missing confirm", args: map[string]any{"scenario_set_uuid": "ss"}},
		{name: "false confirm", args: map[string]any{"scenario_set_uuid": "ss", "confirm_deletion": false}},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := mcp.CallToolRequest{Params: mcp.CallToolParams{Arguments: tt.args}}
			resp, err := tool.deleteScenarioSet(context.Background(), req)
			require.NoError(t, err)
			require.True(t, resp.IsError)
		})
	}
}

func TestSimulationTool_deleteRun_consent(t *testing.T) {
	tool := setupSimulationToolWithFailingClient()
	req := mcp.CallToolRequest{Params: mcp.CallToolParams{Arguments: map[string]any{
		"run_uuid": "run-1",
	}}}
	resp, err := tool.deleteRun(context.Background(), req)
	require.NoError(t, err)
	require.True(t, resp.IsError)
}

func TestSimulationTool_cancelRun_consent(t *testing.T) {
	tool := setupSimulationToolWithFailingClient()
	req := mcp.CallToolRequest{Params: mcp.CallToolParams{Arguments: map[string]any{
		"run_uuid": "run-1",
	}}}
	resp, err := tool.cancelRun(context.Background(), req)
	require.NoError(t, err)
	require.True(t, resp.IsError)
}

func TestSimulationTool_getEpisode_missingArgs(t *testing.T) {
	tool := setupSimulationToolWithFailingClient()
	req := mcp.CallToolRequest{Params: mcp.CallToolParams{Arguments: map[string]any{"run_uuid": "run-1"}}}
	resp, err := tool.getEpisode(context.Background(), req)
	require.NoError(t, err)
	require.True(t, resp.IsError)
}

func TestValidateScenarioSetJSONL(t *testing.T) {
	dir := t.TempDir()

	valid := filepath.Join(dir, "ok.jsonl")
	require.NoError(t, os.WriteFile(valid, []byte("{\"name\":\"checkout\"}\n"), 0o600))
	require.NoError(t, validateScenarioSetJSONL(valid))

	missingName := filepath.Join(dir, "bad.jsonl")
	require.NoError(t, os.WriteFile(missingName, []byte("{\"description\":\"x\"}\n"), 0o600))
	require.Error(t, validateScenarioSetJSONL(missingName))

	wrongExt := filepath.Join(dir, "bad.csv")
	require.NoError(t, os.WriteFile(wrongExt, []byte("name\n"), 0o600))
	require.Error(t, validateScenarioSetJSONL(wrongExt))
}

func TestParseScenariosFromArgs(t *testing.T) {
	scenarios, err := parseScenariosFromArgs([]any{
		map[string]any{
			"name":               "s1",
			"description":        "d",
			"user_persona":       "p",
			"stopping_criteria":  []any{"done", ""},
			"max_turns":          float64(5),
			"exploration_budget": float64(2),
		},
	})
	require.NoError(t, err)
	require.Len(t, scenarios, 1)
	require.Equal(t, "s1", scenarios[0].Name)
	require.Equal(t, []string{"done"}, scenarios[0].StoppingCriteria)
	require.Equal(t, uint32(5), scenarios[0].MaxTurns)

	_, err = parseScenariosFromArgs([]any{map[string]any{"description": "no name"}})
	require.Error(t, err)
}
