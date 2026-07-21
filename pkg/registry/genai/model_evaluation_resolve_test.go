package genai

import (
	"context"
	"testing"

	"github.com/digitalocean/godo"
	"github.com/stretchr/testify/require"
)

func TestResolveEvalModelByName(t *testing.T) {
	models := []*EvalCatalogModel{
		{UUID: "uuid-1", DisplayName: "DeepSeek R1 Distill Llama 70B", APIName: "deepseek-r1-distill-llama-70b", Source: "catalog"},
		{UUID: "uuid-2", DisplayName: "OpenAI GPT-4o", APIName: "openai-gpt-4o", Source: "catalog"},
	}

	resolved, unresolved, err := resolveEvalModelByName(context.Background(), nil, "candidate", "DeepSeek R1 Distill Llama 70B", models)
	require.NoError(t, err)
	require.Nil(t, unresolved)
	require.Equal(t, "deepseek-r1-distill-llama-70b", resolved.APIName)
}

func TestCatalogModelNames(t *testing.T) {
	api, display := catalogModelNames(&godo.Model{
		Name:          "DeepSeek R1 Distill Llama 70B",
		InferenceName: "deepseek-r1-distill-llama-70b",
	})
	require.Equal(t, "deepseek-r1-distill-llama-70b", api)
	require.Equal(t, "DeepSeek R1 Distill Llama 70B", display)
}

func TestIsModelEvalUserMessageYes(t *testing.T) {
	require.True(t, isModelEvalUserMessageYes("yes"))
	require.True(t, isModelEvalUserMessageYes(" YES "))
	require.False(t, isModelEvalUserMessageYes(""))
	require.False(t, isModelEvalUserMessageYes("no"))
}

func TestBuildModelEvalConsentPreview(t *testing.T) {
	resolved := &modelEvalRunModels{
		Candidate: modelEvalResolvedModel{
			UUID: "c-uuid", DisplayName: "Candidate", APIName: "candidate-slug", Source: "catalog",
		},
		Judge: &modelEvalResolvedModel{
			UUID: "j-uuid", DisplayName: "Judge", APIName: "judge-slug", Source: "catalog",
		},
	}
	cfg := &modelEvalRunConfig{
		RunName:     "my-run",
		DatasetUUID: "ds-uuid",
		MetricUUIDs: []string{"m1"},
	}
	preview := buildModelEvalConsentPreview(resolved, cfg)
	require.Equal(t, modelEvalConsentStatus, preview.Status)
	require.True(t, preview.StopAndAskUser)
	require.Contains(t, preview.PromptForUser, "reply **yes**")
	require.Contains(t, preview.InstructionForAgent, "user_message")
	require.Equal(t, "candidate-slug", preview.CandidateModel.APIName)
	require.NotNil(t, preview.JudgeModel)
}

func TestValidateModelEvalResolvedUUIDs(t *testing.T) {
	resolved := &modelEvalRunModels{
		Candidate: modelEvalResolvedModel{UUID: "c-uuid", APIName: "c"},
		Judge:     &modelEvalResolvedModel{UUID: "j-uuid", APIName: "j"},
	}
	require.NoError(t, validateModelEvalResolvedUUIDs(resolved, true))

	require.Error(t, validateModelEvalResolvedUUIDs(&modelEvalRunModels{
		Candidate: modelEvalResolvedModel{UUID: ""},
		Judge:     &modelEvalResolvedModel{UUID: "j"},
	}, true))
}

func TestCheckModelEvalUserMessage(t *testing.T) {
	resolved := &modelEvalRunModels{
		Candidate: modelEvalResolvedModel{UUID: "c", APIName: "c-slug", DisplayName: "C"},
	}
	cfg := &modelEvalRunConfig{RunName: "run"}

	result, ok := checkModelEvalUserMessage(map[string]any{"user_message": "yes"}, resolved, cfg)
	require.True(t, ok)
	require.Nil(t, result)

	result, ok = checkModelEvalUserMessage(map[string]any{}, resolved, cfg)
	require.False(t, ok)
	require.NotNil(t, result)
	require.False(t, result.IsError)
	require.True(t, buildModelEvalConsentPreview(resolved, cfg).RequireUserMessageFromChat)

	result, ok = checkModelEvalUserMessage(map[string]any{"user_message": "no"}, resolved, cfg)
	require.False(t, ok)
	require.True(t, result.IsError)
}
