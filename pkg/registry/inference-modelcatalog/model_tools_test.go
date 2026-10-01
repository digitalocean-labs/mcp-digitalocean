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

func setupModelToolWithMock(mockAgentPlatform godo.AgentPlatformService) *ModelTool {
	client := func(ctx context.Context) (*godo.Client, error) {
		return &godo.Client{
			AgentPlatform: mockAgentPlatform,
		}, nil
	}
	return NewModelTool(client)
}

func TestModelTool_searchModels(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	testUUIDs := []string{
		"12345678-1234-1234-1234-123456789012",
		"87654321-4321-4321-4321-210987654321",
	}

	tests := []struct {
		name          string
		args          map[string]any
		mockSetup     func(*MockAgentPlatformService)
		expectError   bool
		expectGoError bool
		checkUUIDs    bool
	}{
		{
			name: "missing SearchQuery returns all models",
			args: map[string]any{},
			mockSetup: func(m *MockAgentPlatformService) {
				// Missing parameter defaults to empty string, which matches all models
				m.EXPECT().SearchModels(gomock.Any(), "").Return(testUUIDs, nil, nil)
			},
			checkUUIDs: true,
		},
		{
			name: "empty SearchQuery returns all models",
			args: map[string]any{"SearchQuery": ""},
			mockSetup: func(m *MockAgentPlatformService) {
				// Empty string matches all models
				m.EXPECT().SearchModels(gomock.Any(), "").Return(testUUIDs, nil, nil)
			},
			checkUUIDs: true,
		},
		{
			name: "api error",
			args: map[string]any{"SearchQuery": "llama"},
			mockSetup: func(m *MockAgentPlatformService) {
				m.EXPECT().SearchModels(gomock.Any(), "llama").Return(nil, nil, errors.New("api error"))
			},
			expectError: true,
		},
		{
			name: "success with results",
			args: map[string]any{"SearchQuery": "llama"},
			mockSetup: func(m *MockAgentPlatformService) {
				m.EXPECT().SearchModels(gomock.Any(), "llama").Return(testUUIDs, nil, nil)
			},
			checkUUIDs: true,
		},
		{
			name: "success with no results",
			args: map[string]any{"SearchQuery": "nonexistent"},
			mockSetup: func(m *MockAgentPlatformService) {
				m.EXPECT().SearchModels(gomock.Any(), "nonexistent").Return([]string{}, nil, nil)
			},
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			mock := NewMockAgentPlatformService(ctrl)
			if tc.mockSetup != nil {
				tc.mockSetup(mock)
			}
			tool := setupModelToolWithMock(mock)
			req := mcp.CallToolRequest{Params: mcp.CallToolParams{Arguments: tc.args}}
			resp, err := tool.searchModels(context.Background(), req)
			if tc.expectGoError {
				require.Error(t, err)
				return
			}
			if tc.expectError {
				require.NoError(t, err)
				require.NotNil(t, resp)
				require.True(t, resp.IsError)
				return
			}
			require.NoError(t, err)
			require.NotNil(t, resp)
			require.False(t, resp.IsError)
			require.NotEmpty(t, resp.Content)
			textContent, ok := resp.Content[0].(mcp.TextContent)
			require.True(t, ok)
			require.Contains(t, textContent.Text, "model_uuids")

			if tc.checkUUIDs {
				var result struct {
					ModelUUIDs  []string `json:"model_uuids"`
					SearchQuery string   `json:"search_query"`
					Count       int      `json:"count"`
				}
				err := json.Unmarshal([]byte(textContent.Text), &result)
				require.NoError(t, err)
				require.Equal(t, testUUIDs, result.ModelUUIDs, "should return exact UUIDs from mock")
				require.Equal(t, len(testUUIDs), result.Count, "count should match number of UUIDs")
			}
		})
	}
}

func TestModelTool_getModelCard(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	testModel := &godo.Model{
		Uuid:              "12345678-1234-1234-1234-123456789012",
		InferenceName:     "llama3.3-70b-instruct",
		Name:              "Llama 3.3 Instruct (70B)",
		Description:       "An advanced language model with greater capabilities due to its larger size.",
		Provider:          "Meta",
		Url:               "https://example.com/model",
		Usecases:          []string{"chat", "completion"},
		ModelAvailability: "serverless",
		ContextWindow:     "131072",
		Capabilities:      []string{"inference", "chat"},
		Modalities: &godo.ModelModalities{
			Input:  []string{"text"},
			Output: []string{"text"},
		},
		ParameterCount: 70.0,
		Type:           "chat",
		Pricing: &godo.ModelPricing{
			InputPricePerMillion:  3.0,
			OutputPricePerMillion: 15.0,
		},
		BenchmarkScore: json.RawMessage(`{"mmlu": 0.85, "hellaswag": 0.79}`),
		Agreement: &godo.Agreement{
			Name:        "Meta Llama 3.3 License",
			Description: "License for Llama 3.3",
			Url:         "https://example.com/license",
			Uuid:        "license-uuid",
		},
	}

	tests := []struct {
		name          string
		args          map[string]any
		mockSetup     func(*MockAgentPlatformService)
		expectError   bool
		expectGoError bool
		checkModel    bool
	}{
		{
			name:        "missing ModelUUID",
			args:        map[string]any{},
			expectError: true,
		},
		{
			name:        "empty ModelUUID",
			args:        map[string]any{"ModelUUID": ""},
			expectError: true,
		},
		{
			name: "api error",
			args: map[string]any{"ModelUUID": "12345678-1234-1234-1234-123456789012"},
			mockSetup: func(m *MockAgentPlatformService) {
				m.EXPECT().GetModelByUUID(gomock.Any(), "12345678-1234-1234-1234-123456789012").
					Return(nil, nil, errors.New("api error"))
			},
			expectError: true,
		},
		{
			name: "model not found",
			args: map[string]any{"ModelUUID": "99999999-9999-9999-9999-999999999999"},
			mockSetup: func(m *MockAgentPlatformService) {
				m.EXPECT().GetModelByUUID(gomock.Any(), "99999999-9999-9999-9999-999999999999").
					Return(nil, nil, nil)
			},
			expectError: true,
		},
		{
			name: "success",
			args: map[string]any{"ModelUUID": "12345678-1234-1234-1234-123456789012"},
			mockSetup: func(m *MockAgentPlatformService) {
				m.EXPECT().GetModelByUUID(gomock.Any(), "12345678-1234-1234-1234-123456789012").
					Return(testModel, nil, nil)
			},
			checkModel: true,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			mock := NewMockAgentPlatformService(ctrl)
			if tc.mockSetup != nil {
				tc.mockSetup(mock)
			}
			tool := setupModelToolWithMock(mock)
			req := mcp.CallToolRequest{Params: mcp.CallToolParams{Arguments: tc.args}}
			resp, err := tool.getModelCard(context.Background(), req)
			if tc.expectGoError {
				require.Error(t, err)
				return
			}
			if tc.expectError {
				require.NoError(t, err)
				require.NotNil(t, resp)
				require.True(t, resp.IsError)
				return
			}
			require.NoError(t, err)
			require.NotNil(t, resp)
			require.False(t, resp.IsError)
			require.NotEmpty(t, resp.Content)
			textContent, ok := resp.Content[0].(mcp.TextContent)
			require.True(t, ok)
			require.Contains(t, textContent.Text, testModel.Uuid)
			require.Contains(t, textContent.Text, testModel.Name)

			if tc.checkModel {
				var result struct {
					UUID              string                `json:"uuid"`
					Name              string                `json:"name"`
					Description       string                `json:"description,omitempty"`
					Provider          string                `json:"provider,omitempty"`
					Agreement         *godo.Agreement       `json:"agreement,omitempty"`
					ModelAvailability string                `json:"model_availability,omitempty"`
					ContextWindow     string                `json:"context_window,omitempty"`
					Capabilities      []string              `json:"capabilities,omitempty"`
					Modalities        *godo.ModelModalities `json:"modalities,omitempty"`
					ParameterCount    float64               `json:"parameter_count,omitempty"`
					Type              string                `json:"type,omitempty"`
					Pricing           *godo.ModelPricing    `json:"pricing,omitempty"`
					BenchmarkScore    json.RawMessage       `json:"benchmark_score,omitempty"`
				}
				err := json.Unmarshal([]byte(textContent.Text), &result)
				require.NoError(t, err)
				require.Equal(t, testModel.Uuid, result.UUID, "should return exact UUID")
				require.Equal(t, testModel.Name, result.Name, "should return exact name")
				require.Equal(t, testModel.Description, result.Description, "should return exact description")
				require.Equal(t, testModel.Provider, result.Provider, "should return exact provider")
				require.Equal(t, testModel.ModelAvailability, result.ModelAvailability, "should return exact model availability")
				require.Equal(t, testModel.ContextWindow, result.ContextWindow, "should return exact context window")
				require.Equal(t, testModel.Capabilities, result.Capabilities, "should return exact capabilities")
				require.Equal(t, testModel.ParameterCount, result.ParameterCount, "should return exact parameter count")
				require.Equal(t, testModel.Type, result.Type, "should return exact type")
				require.NotNil(t, result.Pricing, "should include pricing")
				require.Equal(t, testModel.Pricing.InputPricePerMillion, result.Pricing.InputPricePerMillion, "should return exact input price")
				require.Equal(t, testModel.Pricing.OutputPricePerMillion, result.Pricing.OutputPricePerMillion, "should return exact output price")

				// Compare benchmark score as JSON (ignore formatting differences)
				if len(testModel.BenchmarkScore) > 0 {
					var expectedBenchmark, actualBenchmark map[string]interface{}
					require.NoError(t, json.Unmarshal(testModel.BenchmarkScore, &expectedBenchmark), "should unmarshal expected benchmark")
					require.NoError(t, json.Unmarshal(result.BenchmarkScore, &actualBenchmark), "should unmarshal actual benchmark")
					require.Equal(t, expectedBenchmark, actualBenchmark, "should return same benchmark data")
				}

				require.NotNil(t, result.Agreement, "should include agreement")
				require.Equal(t, testModel.Agreement.Name, result.Agreement.Name, "should return exact agreement name")
				require.Equal(t, testModel.Agreement.Description, result.Agreement.Description, "should return exact agreement description")
				require.Equal(t, testModel.Agreement.Url, result.Agreement.Url, "should return exact agreement URL")
				require.Equal(t, testModel.Agreement.Uuid, result.Agreement.Uuid, "should return exact agreement UUID")
				require.NotNil(t, result.Modalities, "should include modalities")
				require.Equal(t, testModel.Modalities.Input, result.Modalities.Input, "should return exact modalities input")
				require.Equal(t, testModel.Modalities.Output, result.Modalities.Output, "should return exact modalities output")
			}
		})
	}
}

func TestModelTool_Tools(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mock := NewMockAgentPlatformService(ctrl)
	tool := setupModelToolWithMock(mock)

	tools := tool.Tools()
	require.Len(t, tools, 2)

	// Check that both tools are present
	toolNames := make(map[string]bool)
	for _, t := range tools {
		toolNames[t.Tool.Name] = true
	}

	require.True(t, toolNames["inference-model-catalog-search"], "should have inference-model-catalog-search tool")
	require.True(t, toolNames["inference-model-catalog-get-card"], "should have inference-model-catalog-get-card tool")
}
