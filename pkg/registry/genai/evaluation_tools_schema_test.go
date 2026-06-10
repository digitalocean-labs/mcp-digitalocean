package genai

import (
	"context"
	"encoding/json"
	"testing"

	"github.com/digitalocean/godo"
	"github.com/stretchr/testify/require"
)

func TestAgentEvaluationToolListParamSchemas(t *testing.T) {
	et := NewEvaluationTool(func(ctx context.Context) (*godo.Client, error) {
		return nil, nil
	})

	cases := map[string][]string{
		"genai-create-evaluation-test-case": {"metrics"},
		"genai-update-evaluation-test-case": {"metrics"},
		"genai-run-evaluation-test-case":    {"agent_deployment_names"},
		"genai-run-evaluation-workflow":     {"agent_deployment_names", "metric_categories"},
	}

	byName := make(map[string]map[string]any)
	for _, st := range et.Tools() {
		raw, err := json.Marshal(st.Tool.InputSchema)
		require.NoError(t, err)
		var schema map[string]any
		require.NoError(t, json.Unmarshal(raw, &schema))
		props, ok := schema["properties"].(map[string]any)
		require.True(t, ok, "tool %s missing properties", st.Tool.Name)
		byName[st.Tool.Name] = props
	}

	for toolName, fields := range cases {
		props, ok := byName[toolName]
		require.True(t, ok, "tool %s not registered", toolName)
		for _, field := range fields {
			prop, ok := props[field].(map[string]any)
			require.True(t, ok, "tool %s missing property %s", toolName, field)
			require.Equal(t, "array", prop["type"], "tool %s field %s should be array, got %#v", toolName, field, prop["type"])
			items, ok := prop["items"].(map[string]any)
			require.True(t, ok, "tool %s field %s missing items", toolName, field)
			require.Equal(t, "string", items["type"], "tool %s field %s items should be string", toolName, field)
		}
	}
}

func TestModelEvaluationToolListParamSchemas(t *testing.T) {
	met := NewModelEvaluationTool(func(ctx context.Context) (*godo.Client, error) {
		return nil, nil
	})

	byName := make(map[string]map[string]any)
	for _, st := range met.Tools() {
		raw, err := json.Marshal(st.Tool.InputSchema)
		require.NoError(t, err)
		var schema map[string]any
		require.NoError(t, json.Unmarshal(raw, &schema))
		props, _ := schema["properties"].(map[string]any)
		byName[st.Tool.Name] = props
	}

	// The new dataset-discovery tool must be registered so an existing dataset_uuid
	// can be found without uploading a new dataset.
	_, ok := byName["genai-model-eval-list-datasets"]
	require.True(t, ok, "genai-model-eval-list-datasets must be registered")

	// metric_uuids must be advertised as an array of strings (not an object) so
	// schema-driven callers send ["uuid", ...] rather than guessing.
	for _, toolName := range []string{"genai-model-eval-create-run", "genai-model-eval-run-workflow"} {
		props, ok := byName[toolName]
		require.True(t, ok, "tool %s not registered", toolName)
		prop, ok := props["metric_uuids"].(map[string]any)
		require.True(t, ok, "tool %s missing property metric_uuids", toolName)
		require.Equal(t, "array", prop["type"], "tool %s metric_uuids should be array, got %#v", toolName, prop["type"])
		items, ok := prop["items"].(map[string]any)
		require.True(t, ok, "tool %s metric_uuids missing items", toolName)
		require.Equal(t, "string", items["type"], "tool %s metric_uuids items should be string", toolName)
	}
}
