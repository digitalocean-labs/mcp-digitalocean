package inferencemodelcatalog

import (
	"context"
	"encoding/json"
	"errors"
	"testing"

	"github.com/digitalocean/godo"
	"github.com/mark3labs/mcp-go/mcp"
	"github.com/stretchr/testify/require"
	"go.uber.org/mock/gomock"
)

func TestModelTool_Prompts(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mock := NewMockAgentPlatformService(ctrl)
	tool := setupModelToolWithMock(mock)

	prompts := tool.Prompts()
	require.Len(t, prompts, 2, "should have 2 prompts")

	// Check that both prompts are present
	promptNames := make(map[string]bool)
	for _, p := range prompts {
		promptNames[p.Prompt.Name] = true
	}

	require.True(t, promptNames["model-comparison"], "should have model-comparison prompt")
	require.True(t, promptNames["search-by-task"], "should have search-by-task prompt")
}

func TestMatchesConstraints(t *testing.T) {
	tests := []struct {
		name        string
		model       *ModelMetadata
		constraints taskConstraints
		expected    bool
	}{
		{
			name: "no constraints - always matches",
			model: &ModelMetadata{
				Provider: "OpenAI",
			},
			constraints: taskConstraints{},
			expected:    true,
		},
		{
			name: "provider matches",
			model: &ModelMetadata{
				Provider: "OpenAI",
			},
			constraints: taskConstraints{
				provider: stringPtr("openai"),
			},
			expected: true,
		},
		{
			name: "provider doesn't match",
			model: &ModelMetadata{
				Provider: "Meta",
			},
			constraints: taskConstraints{
				provider: stringPtr("openai"),
			},
			expected: false,
		},
		{
			name: "deployment type matches",
			model: &ModelMetadata{
				ModelAvailability: "Serverless, Dedicated",
			},
			constraints: taskConstraints{
				deploymentType: stringPtr("serverless"),
			},
			expected: true,
		},
		{
			name: "deployment type doesn't match",
			model: &ModelMetadata{
				ModelAvailability: "Dedicated",
			},
			constraints: taskConstraints{
				deploymentType: stringPtr("serverless"),
			},
			expected: false,
		},
		{
			name: "context window meets minimum",
			model: &ModelMetadata{
				ContextWindow: "128000 tokens",
			},
			constraints: taskConstraints{
				minContextWindow: stringPtr("100000"),
			},
			expected: true,
		},
		{
			name: "context window below minimum",
			model: &ModelMetadata{
				ContextWindow: "8000 tokens",
			},
			constraints: taskConstraints{
				minContextWindow: stringPtr("100000"),
			},
			expected: false,
		},
		{
			name: "input price within budget",
			model: &ModelMetadata{
				Pricing: &godo.ModelPricing{
					InputPricePerMillion: 3.0,
				},
			},
			constraints: taskConstraints{
				maxInputPrice: stringPtr("5.0"),
			},
			expected: true,
		},
		{
			name: "input price exceeds budget",
			model: &ModelMetadata{
				Pricing: &godo.ModelPricing{
					InputPricePerMillion: 10.0,
				},
			},
			constraints: taskConstraints{
				maxInputPrice: stringPtr("5.0"),
			},
			expected: false,
		},
		{
			name: "output price within budget",
			model: &ModelMetadata{
				Pricing: &godo.ModelPricing{
					OutputPricePerMillion: 12.0,
				},
			},
			constraints: taskConstraints{
				maxOutputPrice: stringPtr("15.0"),
			},
			expected: true,
		},
		{
			name: "output price exceeds budget",
			model: &ModelMetadata{
				Pricing: &godo.ModelPricing{
					OutputPricePerMillion: 20.0,
				},
			},
			constraints: taskConstraints{
				maxOutputPrice: stringPtr("15.0"),
			},
			expected: false,
		},
		{
			name: "multiple constraints all match",
			model: &ModelMetadata{
				Provider:          "Meta",
				ModelAvailability: "Serverless",
				ContextWindow:     "128K tokens",
				Pricing: &godo.ModelPricing{
					InputPricePerMillion:  3.0,
					OutputPricePerMillion: 15.0,
				},
			},
			constraints: taskConstraints{
				provider:         stringPtr("meta"),
				deploymentType:   stringPtr("serverless"),
				minContextWindow: stringPtr("100000"),
				maxInputPrice:    stringPtr("5.0"),
				maxOutputPrice:   stringPtr("20.0"),
			},
			expected: true,
		},
		{
			name: "multiple constraints - one fails",
			model: &ModelMetadata{
				Provider:          "Meta",
				ModelAvailability: "Serverless",
				ContextWindow:     "128K tokens",
				Pricing: &godo.ModelPricing{
					InputPricePerMillion:  10.0, // Exceeds budget
					OutputPricePerMillion: 15.0,
				},
			},
			constraints: taskConstraints{
				provider:         stringPtr("meta"),
				deploymentType:   stringPtr("serverless"),
				minContextWindow: stringPtr("100000"),
				maxInputPrice:    stringPtr("5.0"),
				maxOutputPrice:   stringPtr("20.0"),
			},
			expected: false,
		},
		{
			name: "price constraint with nil pricing - excluded",
			model: &ModelMetadata{
				Provider: "OpenAI",
				Pricing:  nil,
			},
			constraints: taskConstraints{
				maxInputPrice: stringPtr("5.0"),
			},
			expected: false,
		},
		{
			name: "price constraint with zero pricing - excluded",
			model: &ModelMetadata{
				Provider: "OpenAI",
				Pricing: &godo.ModelPricing{
					InputPricePerMillion:  0.0,
					OutputPricePerMillion: 0.0,
				},
			},
			constraints: taskConstraints{
				maxInputPrice: stringPtr("5.0"),
			},
			expected: false,
		},
		{
			name: "price constraint with partial zero pricing (input non-zero) - included",
			model: &ModelMetadata{
				Provider: "OpenAI",
				Pricing: &godo.ModelPricing{
					InputPricePerMillion:  3.0,
					OutputPricePerMillion: 0.0,
				},
			},
			constraints: taskConstraints{
				maxInputPrice: stringPtr("5.0"),
			},
			expected: true,
		},
		{
			name: "price constraint with partial zero pricing (output non-zero) - included",
			model: &ModelMetadata{
				Provider: "OpenAI",
				Pricing: &godo.ModelPricing{
					InputPricePerMillion:  0.0,
					OutputPricePerMillion: 10.0,
				},
			},
			constraints: taskConstraints{
				maxOutputPrice: stringPtr("15.0"),
			},
			expected: true,
		},
		{
			name: "no price constraint with zero pricing - included",
			model: &ModelMetadata{
				Provider: "OpenAI",
				Pricing: &godo.ModelPricing{
					InputPricePerMillion:  0.0,
					OutputPricePerMillion: 0.0,
				},
			},
			constraints: taskConstraints{
				provider: stringPtr("openai"),
			},
			expected: true,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			result := matchesConstraints(tc.model, tc.constraints)
			require.Equal(t, tc.expected, result)
		})
	}
}

func TestMatchesTask(t *testing.T) {
	tests := []struct {
		name     string
		model    *ModelMetadata
		task     string
		expected bool
	}{
		{
			name: "task contains model name",
			model: &ModelMetadata{
				Name: "llama",
			},
			task:     "I need llama for chat",
			expected: true,
		},
		{
			name: "task contains provider",
			model: &ModelMetadata{
				Name:     "gpt-4o",
				Provider: "OpenAI",
			},
			task:     "I want an OpenAI model",
			expected: true,
		},
		{
			name: "task contains model type",
			model: &ModelMetadata{
				Name: "claude-3-5-sonnet",
				Type: "chat",
			},
			task:     "chat model for conversations",
			expected: true,
		},
		{
			name: "task contains capability",
			model: &ModelMetadata{
				Name:         "gpt-4o",
				Capabilities: []string{"vision", "reasoning"},
			},
			task:     "I need vision capabilities",
			expected: true,
		},
		{
			name: "task doesn't match anything",
			model: &ModelMetadata{
				Name:         "llama-3.3-70b",
				Provider:     "Meta",
				Type:         "chat",
				Capabilities: []string{"inference"},
			},
			task:     "image generation model",
			expected: false,
		},
		{
			name: "case insensitive matching",
			model: &ModelMetadata{
				Name:     "gpt",
				Provider: "OpenAI",
			},
			task:     "GPT model",
			expected: true,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			result := matchesTask(tc.model, tc.task)
			require.Equal(t, tc.expected, result)
		})
	}
}

func TestParseContextWindow(t *testing.T) {
	tests := []struct {
		input    string
		expected int
	}{
		{"", 0},
		{"128000 tokens", 128000},
		{"128K tokens", 128000},
		{"128k", 128000},
		{"8,192 tokens", 8192},
		{"200000", 200000},
		{"invalid", 0},
	}

	for _, tc := range tests {
		t.Run(tc.input, func(t *testing.T) {
			result := parseContextWindow(tc.input)
			require.Equal(t, tc.expected, result)
		})
	}
}

func TestFormatContextWindow(t *testing.T) {
	tests := []struct {
		input    string
		expected string
	}{
		{"", "—"},
		{"128000 tokens", "128K tokens"},
		{"128K tokens", "128K tokens"},
		{"8192", "8.2K tokens"},
		{"8000", "8K tokens"},
		{"1024", "1.0K tokens"},
		{"500", "500 tokens"},
	}

	for _, tc := range tests {
		t.Run(tc.input, func(t *testing.T) {
			result := formatContextWindow(tc.input)
			require.Equal(t, tc.expected, result)
		})
	}
}

func TestFormatPrice(t *testing.T) {
	tests := []struct {
		name     string
		pricing  *godo.ModelPricing
		isInput  bool
		expected string
	}{
		{
			name:     "nil pricing",
			pricing:  nil,
			isInput:  true,
			expected: "—",
		},
		{
			name: "zero input price",
			pricing: &godo.ModelPricing{
				InputPricePerMillion: 0,
			},
			isInput:  true,
			expected: "—",
		},
		{
			name: "input price",
			pricing: &godo.ModelPricing{
				InputPricePerMillion: 3.0,
			},
			isInput:  true,
			expected: "$3.00/M tokens",
		},
		{
			name: "output price",
			pricing: &godo.ModelPricing{
				OutputPricePerMillion: 15.0,
			},
			isInput:  false,
			expected: "$15.00/M tokens",
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			result := formatPrice(tc.pricing, tc.isInput)
			require.Equal(t, tc.expected, result)
		})
	}
}

func TestFormatBenchmarkScore(t *testing.T) {
	tests := []struct {
		name     string
		input    json.RawMessage
		expected string
	}{
		{
			name:     "empty",
			input:    json.RawMessage{},
			expected: "—",
		},
		{
			name:     "null",
			input:    json.RawMessage("null"),
			expected: "—",
		},
		{
			name:     "valid json",
			input:    json.RawMessage(`{"mmlu": 0.85, "hellaswag": 0.79}`),
			expected: `{"mmlu": 0.85, "hellaswag": 0.79}`,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			result := formatBenchmarkScore(tc.input)
			require.Equal(t, tc.expected, result)
		})
	}
}

func TestFormatSearchResults_NoModels(t *testing.T) {
	constraints := taskConstraints{
		maxInputPrice:  stringPtr("5.0"),
		maxOutputPrice: stringPtr("15.0"),
	}

	result := formatSearchResults("low-latency chat", []*ModelMetadata{}, constraints)

	// Strict assertions on structure
	require.Contains(t, result, "# Model Search Results\n\n")
	require.Contains(t, result, "**Task**: low-latency chat\n\n")
	require.Contains(t, result, "## Applied Constraints\n\n")
	require.Contains(t, result, "- **Max Input Price**: $5.0 per million tokens\n")
	require.Contains(t, result, "- **Max Output Price**: $15.0 per million tokens\n")
	require.Contains(t, result, "## No Models Found\n\n")
	require.Contains(t, result, "No models matched the search criteria and constraints.\n")
	require.NotContains(t, result, "Recommendation")
	require.NotContains(t, result, "Matching Models")
}

func TestFormatSearchResults_WithModels(t *testing.T) {
	models := []*ModelMetadata{
		{
			UUID:     "uuid-1",
			Name:     "llama-3.3-70b-instruct",
			Provider: "Meta",
			Type:     "chat",
		},
		{
			UUID:     "uuid-2",
			Name:     "gpt-4o",
			Provider: "OpenAI",
			Type:     "chat",
		},
	}

	result := formatSearchResults("chat model", models, taskConstraints{})

	// Strict assertions on structure
	require.Contains(t, result, "# Model Search Results\n\n")
	require.Contains(t, result, "**Task**: chat model\n\n")
	require.Contains(t, result, "## Matching Models (2 found)\n\n")
	require.Contains(t, result, "### 1. llama-3.3-70b-instruct\n\n")
	require.Contains(t, result, "- **UUID**: uuid-1\n")
	require.Contains(t, result, "- **Provider**: Meta\n")
	require.Contains(t, result, "### 2. gpt-4o\n\n")
	require.Contains(t, result, "- **UUID**: uuid-2\n")
	require.Contains(t, result, "- **Provider**: OpenAI\n")
	require.Contains(t, result, "## Recommendation\n\n")
	require.Contains(t, result, "**Best Match**: llama-3.3-70b-instruct\n\n")
	require.NotContains(t, result, "No Models Found")
}

func TestFormatModelComparison(t *testing.T) {
	model1 := &ModelMetadata{
		UUID:              "uuid-1",
		Name:              "llama-3.3-70b-instruct",
		Description:       "Meta's flagship model",
		Provider:          "Meta",
		Type:              "chat",
		ParameterCount:    70.0,
		ContextWindow:     "128K tokens",
		ModelAvailability: "Serverless",
		Capabilities:      []string{"chat", "inference"},
		Pricing: &godo.ModelPricing{
			InputPricePerMillion:  3.0,
			OutputPricePerMillion: 15.0,
		},
		BenchmarkScore: json.RawMessage(`{"mmlu": 0.85}`),
	}

	model2 := &ModelMetadata{
		UUID:              "uuid-2",
		Name:              "gpt-4o",
		Description:       "OpenAI's multimodal model",
		Provider:          "OpenAI",
		Type:              "chat",
		ParameterCount:    0, // Not disclosed
		ContextWindow:     "128K tokens",
		ModelAvailability: "Dedicated",
		Capabilities:      []string{"chat", "vision"},
		Pricing: &godo.ModelPricing{
			InputPricePerMillion:  5.0,
			OutputPricePerMillion: 15.0,
		},
	}

	result := formatModelComparison(model1, model2)

	// Strict assertions on structure and content
	require.Contains(t, result, "# Model Comparison\n\n")
	require.Contains(t, result, "| Model | llama-3.3-70b-instruct | gpt-4o |\n")
	require.Contains(t, result, "| **UUID** | uuid-1 | uuid-2 |\n")
	require.Contains(t, result, "| **Provider** | Meta | OpenAI |\n")
	require.Contains(t, result, "| **Type** | chat | chat |\n")
	require.Contains(t, result, "| **Parameter Count** | 70.0B | — |\n")
	require.Contains(t, result, "| **Context Window** | 128K tokens | 128K tokens |\n")
	require.Contains(t, result, "| **Input Price** | $3.00/M tokens | $5.00/M tokens |\n")
	require.Contains(t, result, "| **Output Price** | $15.00/M tokens | $15.00/M tokens |\n")
	require.Contains(t, result, "| **Availability** | Serverless | Dedicated |\n")
	require.Contains(t, result, "## Descriptions\n\n")
	require.Contains(t, result, "**llama-3.3-70b-instruct**: Meta's flagship model\n\n")
	require.Contains(t, result, "**gpt-4o**: OpenAI's multimodal model\n\n")
}

func TestHandleModelComparison_Success(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	model1 := &godo.Model{
		Uuid:              "uuid-1",
		Name:              "llama-3.3-70b-instruct",
		Provider:          "Meta",
		Type:              "chat",
		ContextWindow:     "128K tokens",
		ModelAvailability: "Serverless",
	}

	model2 := &godo.Model{
		Uuid:              "uuid-2",
		Name:              "gpt-4o",
		Provider:          "OpenAI",
		Type:              "chat",
		ContextWindow:     "128K tokens",
		ModelAvailability: "Dedicated",
	}

	mock := NewMockAgentPlatformService(ctrl)
	mock.EXPECT().GetModelByUUID(gomock.Any(), "uuid-1").Return(model1, nil, nil)
	mock.EXPECT().GetModelByUUID(gomock.Any(), "uuid-2").Return(model2, nil, nil)

	tool := setupModelToolWithMock(mock)

	req := mcp.GetPromptRequest{
		Params: mcp.GetPromptParams{
			Arguments: map[string]string{
				"ModelUUID1": "uuid-1",
				"ModelUUID2": "uuid-2",
			},
		},
	}

	result, err := tool.handleModelComparison(context.Background(), req)

	require.NoError(t, err)
	require.NotNil(t, result)
	require.NotEmpty(t, result.Messages)
	require.Equal(t, "user", string(result.Messages[0].Role))

	textContent := result.Messages[0].Content.(mcp.TextContent)
	require.Contains(t, textContent.Text, "Model Comparison")
	require.Contains(t, textContent.Text, "llama-3.3-70b-instruct")
	require.Contains(t, textContent.Text, "gpt-4o")
}

func TestHandleModelComparison_MissingArguments(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	tool := setupModelToolWithMock(NewMockAgentPlatformService(ctrl))

	tests := []struct {
		name string
		args map[string]string
	}{
		{
			name: "missing ModelUUID1",
			args: map[string]string{
				"ModelUUID2": "uuid-2",
			},
		},
		{
			name: "missing ModelUUID2",
			args: map[string]string{
				"ModelUUID1": "uuid-1",
			},
		},
		{
			name: "missing both",
			args: map[string]string{},
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			req := mcp.GetPromptRequest{
				Params: mcp.GetPromptParams{
					Arguments: tc.args,
				},
			}

			result, err := tool.handleModelComparison(context.Background(), req)
			require.Error(t, err)
			require.Nil(t, result)
		})
	}
}

// Helper function for tests
func stringPtr(s string) *string {
	return &s
}

func TestHandleSearchByTask_Success(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mock := NewMockAgentPlatformService(ctrl)
	mock.EXPECT().SearchModels(gomock.Any(), "").Return([]string{"uuid-1", "uuid-2"}, nil, nil)

	model1 := &godo.Model{
		Uuid:     "uuid-1",
		Name:     "llama-3.3-70b-instruct",
		Provider: "Meta",
		Type:     "chat",
		Pricing: &godo.ModelPricing{
			InputPricePerMillion:  3.0,
			OutputPricePerMillion: 15.0,
		},
	}
	model2 := &godo.Model{
		Uuid:     "uuid-2",
		Name:     "gpt-4o",
		Provider: "OpenAI",
		Type:     "chat",
		Pricing: &godo.ModelPricing{
			InputPricePerMillion:  5.0,
			OutputPricePerMillion: 15.0,
		},
	}

	mock.EXPECT().GetModelByUUID(gomock.Any(), "uuid-1").Return(model1, nil, nil)
	mock.EXPECT().GetModelByUUID(gomock.Any(), "uuid-2").Return(model2, nil, nil)

	tool := setupModelToolWithMock(mock)

	req := mcp.GetPromptRequest{
		Params: mcp.GetPromptParams{
			Arguments: map[string]string{
				"Task":          "chat model",
				"MaxInputPrice": "10.0",
			},
		},
	}

	result, err := tool.handleSearchByTask(context.Background(), req)

	require.NoError(t, err)
	require.NotNil(t, result)
	require.NotEmpty(t, result.Messages)
	require.Equal(t, "user", string(result.Messages[0].Role))

	textContent := result.Messages[0].Content.(mcp.TextContent)
	require.Contains(t, textContent.Text, "Model Search Results")
	require.Contains(t, textContent.Text, "Matching Models")
}

func TestHandleSearchByTask_MissingArguments(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	tool := setupModelToolWithMock(NewMockAgentPlatformService(ctrl))

	req := mcp.GetPromptRequest{
		Params: mcp.GetPromptParams{
			Arguments: nil,
		},
	}

	result, err := tool.handleSearchByTask(context.Background(), req)
	require.Error(t, err)
	require.Nil(t, result)
}

func TestHandleSearchByTask_SearchError(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mock := NewMockAgentPlatformService(ctrl)
	mock.EXPECT().SearchModels(gomock.Any(), "").Return(nil, nil, errors.New("API unavailable"))

	tool := setupModelToolWithMock(mock)

	req := mcp.GetPromptRequest{
		Params: mcp.GetPromptParams{
			Arguments: map[string]string{
				"Task": "chat model",
			},
		},
	}

	result, err := tool.handleSearchByTask(context.Background(), req)
	require.Error(t, err)
	require.Nil(t, result)
	require.Contains(t, err.Error(), "failed to search models")
}

func TestHandleSearchByTask_ModelFetchError(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mock := NewMockAgentPlatformService(ctrl)
	mock.EXPECT().SearchModels(gomock.Any(), "").Return([]string{"uuid-1", "uuid-2"}, nil, nil)
	mock.EXPECT().GetModelByUUID(gomock.Any(), "uuid-1").Return(nil, nil, errors.New("model not found"))
	// Should continue with uuid-2
	mock.EXPECT().GetModelByUUID(gomock.Any(), "uuid-2").Return(&godo.Model{
		Uuid:     "uuid-2",
		Name:     "gpt-4o",
		Provider: "OpenAI",
		Type:     "chat",
	}, nil, nil)

	tool := setupModelToolWithMock(mock)

	req := mcp.GetPromptRequest{
		Params: mcp.GetPromptParams{
			Arguments: map[string]string{
				"Task": "chat",
			},
		},
	}

	result, err := tool.handleSearchByTask(context.Background(), req)
	require.NoError(t, err, "should continue despite one model fetch failure")
	require.NotNil(t, result)

	textContent := result.Messages[0].Content.(mcp.TextContent)
	require.Contains(t, textContent.Text, "gpt-4o")
	require.NotContains(t, textContent.Text, "uuid-1")
}

func TestHandleSearchByTask_WithInvalidPriceConstraints(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mock := NewMockAgentPlatformService(ctrl)
	mock.EXPECT().SearchModels(gomock.Any(), "").Return([]string{"uuid-1"}, nil, nil)
	mock.EXPECT().GetModelByUUID(gomock.Any(), "uuid-1").Return(&godo.Model{
		Uuid:     "uuid-1",
		Name:     "test-model",
		Provider: "OpenAI",
		Type:     "chat",
		Pricing: &godo.ModelPricing{
			InputPricePerMillion:  5.0,
			OutputPricePerMillion: 10.0,
		},
	}, nil, nil)

	tool := setupModelToolWithMock(mock)

	req := mcp.GetPromptRequest{
		Params: mcp.GetPromptParams{
			Arguments: map[string]string{
				"Task":           "chat",
				"MaxInputPrice":  "invalid",  // Invalid price format
				"MaxOutputPrice": "also bad", // Invalid price format
			},
		},
	}

	result, err := tool.handleSearchByTask(context.Background(), req)
	require.NoError(t, err, "invalid price strings should be ignored gracefully")
	require.NotNil(t, result)

	textContent := result.Messages[0].Content.(mcp.TextContent)
	// Model should be included since invalid constraints are ignored
	require.Contains(t, textContent.Text, "test-model")
}

func TestMatchesConstraints_PricingNilButConstraintExists(t *testing.T) {
	model := &ModelMetadata{
		Provider: "OpenAI",
		Pricing:  nil, // No pricing data
	}

	constraints := taskConstraints{
		maxInputPrice: stringPtr("5.0"),
	}

	// Should filter out - if user specifies price constraint, require actual pricing
	result := matchesConstraints(model, constraints)
	require.False(t, result, "models without pricing should be excluded when price constraint exists")
}

func TestMatchesConstraints_InvalidPriceStrings(t *testing.T) {
	model := &ModelMetadata{
		Provider: "OpenAI",
		Pricing: &godo.ModelPricing{
			InputPricePerMillion:  5.0,
			OutputPricePerMillion: 10.0,
		},
	}

	tests := []struct {
		name        string
		constraints taskConstraints
		shouldMatch bool
	}{
		{
			name: "invalid input price format",
			constraints: taskConstraints{
				maxInputPrice: stringPtr("abc"),
			},
			shouldMatch: true, // Invalid format ignored
		},
		{
			name: "invalid output price format",
			constraints: taskConstraints{
				maxOutputPrice: stringPtr("xyz123"),
			},
			shouldMatch: true, // Invalid format ignored
		},
		{
			name: "empty price string",
			constraints: taskConstraints{
				maxInputPrice: stringPtr(""),
			},
			shouldMatch: true, // Empty ignored
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			result := matchesConstraints(model, tc.constraints)
			require.Equal(t, tc.shouldMatch, result)
		})
	}
}

func TestFormatModalities(t *testing.T) {
	tests := []struct {
		name       string
		modalities *godo.ModelModalities
		direction  string
		expected   string
	}{
		{
			name:       "nil modalities",
			modalities: nil,
			direction:  "input",
			expected:   "—",
		},
		{
			name: "empty input modalities",
			modalities: &godo.ModelModalities{
				Input:  []string{},
				Output: []string{"text"},
			},
			direction: "input",
			expected:  "—",
		},
		{
			name: "single input modality",
			modalities: &godo.ModelModalities{
				Input:  []string{"text"},
				Output: []string{"text"},
			},
			direction: "input",
			expected:  "text",
		},
		{
			name: "multiple input modalities",
			modalities: &godo.ModelModalities{
				Input:  []string{"text", "image"},
				Output: []string{"text"},
			},
			direction: "input",
			expected:  "text, image",
		},
		{
			name: "output modalities",
			modalities: &godo.ModelModalities{
				Input:  []string{"text"},
				Output: []string{"text", "image", "audio"},
			},
			direction: "output",
			expected:  "text, image, audio",
		},
		{
			name: "empty output modalities",
			modalities: &godo.ModelModalities{
				Input:  []string{"text"},
				Output: []string{},
			},
			direction: "output",
			expected:  "—",
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			result := formatModalities(tc.modalities, tc.direction)
			require.Equal(t, tc.expected, result)
		})
	}
}

func TestFormatOptionalString(t *testing.T) {
	tests := []struct {
		input    string
		expected string
	}{
		{"", "—"},
		{"OpenAI", "OpenAI"},
		{"chat", "chat"},
	}

	for _, tc := range tests {
		t.Run(tc.input, func(t *testing.T) {
			result := formatOptionalString(tc.input)
			require.Equal(t, tc.expected, result)
		})
	}
}

func TestFormatCapabilities(t *testing.T) {
	tests := []struct {
		name     string
		input    []string
		expected string
	}{
		{
			name:     "empty capabilities",
			input:    []string{},
			expected: "—",
		},
		{
			name:     "nil capabilities",
			input:    nil,
			expected: "—",
		},
		{
			name:     "single capability",
			input:    []string{"chat"},
			expected: "chat",
		},
		{
			name:     "multiple capabilities",
			input:    []string{"chat", "vision", "reasoning"},
			expected: "chat, vision, reasoning",
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			result := formatCapabilities(tc.input)
			require.Equal(t, tc.expected, result)
		})
	}
}
