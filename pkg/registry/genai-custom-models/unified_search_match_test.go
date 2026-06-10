package genaicustommodels

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestCustomModelMatchesQuery(t *testing.T) {
	tests := []struct {
		name     string
		model    *CustomModel
		query    string
		expected bool
	}{
		{
			name:     "match by name",
			model:    &CustomModel{Name: "my-llama-model"},
			query:    "llama",
			expected: true,
		},
		{
			name:     "match by description",
			model:    &CustomModel{Name: "some-model", Description: "A fine-tuned Llama variant"},
			query:    "llama",
			expected: true,
		},
		{
			name:     "match by tag",
			model:    &CustomModel{Name: "some-model", Tags: &CustomModelTags{Tags: []string{"llm", "llama"}}},
			query:    "llama",
			expected: true,
		},
		{
			name:     "match by architecture",
			model:    &CustomModel{Name: "some-model", Architecture: "LlamaForCausalLM"},
			query:    "llama",
			expected: true,
		},
		{
			name:     "no match",
			model:    &CustomModel{Name: "my-gpt-model", Description: "A GPT variant"},
			query:    "llama",
			expected: false,
		},
		{
			name:     "case insensitive",
			model:    &CustomModel{Name: "My-LLAMA-Model"},
			query:    "llama",
			expected: true,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			result := customModelMatchesQuery(tc.model, tc.query)
			require.Equal(t, tc.expected, result)
		})
	}
}

func TestFilterAndRankCustomModels_PartialQueryOrdersByRelevance(t *testing.T) {
	models := []*CustomModel{
		{UUID: "1", Name: "alpha-llama-helper"},
		{UUID: "2", Name: "llama"},
		{UUID: "3", Name: "other"},
	}

	ranked := filterAndRankCustomModels(models, "llama")
	require.Len(t, ranked, 2)
	require.Equal(t, "2", ranked[0].UUID, "exact name match should rank first")
	require.Equal(t, "1", ranked[1].UUID)
}

func TestFilterAndRankCustomModels_EmptyQueryReturnsAll(t *testing.T) {
	models := []*CustomModel{
		{UUID: "b", Name: "b-model"},
		{UUID: "a", Name: "a-model"},
	}
	ranked := filterAndRankCustomModels(models, "")
	require.Len(t, ranked, 2)
	require.Equal(t, "a", ranked[0].UUID)
	require.Equal(t, "b", ranked[1].UUID)
}
