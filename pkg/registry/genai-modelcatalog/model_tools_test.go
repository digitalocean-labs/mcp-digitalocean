package genaimodelcatalog

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

func setupModelToolWithMock(mockGradientAI godo.GradientAIService) *ModelTool {
	client := func(ctx context.Context) (*godo.Client, error) {
		return &godo.Client{
			GradientAI: mockGradientAI,
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
		mockSetup     func(*MockGradientAIService)
		expectError   bool
		expectGoError bool
		checkUUIDs    bool
	}{
		{
			name: "missing SearchQuery returns all models",
			args: map[string]any{},
			mockSetup: func(m *MockGradientAIService) {
				// Missing parameter defaults to empty string, which matches all models
				m.EXPECT().SearchModels(gomock.Any(), "").Return(testUUIDs, nil, nil)
			},
			checkUUIDs: true,
		},
		{
			name: "empty SearchQuery returns all models",
			args: map[string]any{"SearchQuery": ""},
			mockSetup: func(m *MockGradientAIService) {
				// Empty string matches all models
				m.EXPECT().SearchModels(gomock.Any(), "").Return(testUUIDs, nil, nil)
			},
			checkUUIDs: true,
		},
		{
			name: "api error",
			args: map[string]any{"SearchQuery": "llama"},
			mockSetup: func(m *MockGradientAIService) {
				m.EXPECT().SearchModels(gomock.Any(), "llama").Return(nil, nil, errors.New("api error"))
			},
			expectError: true,
		},
		{
			name: "success with results",
			args: map[string]any{"SearchQuery": "llama"},
			mockSetup: func(m *MockGradientAIService) {
				m.EXPECT().SearchModels(gomock.Any(), "llama").Return(testUUIDs, nil, nil)
			},
			checkUUIDs: true,
		},
		{
			name: "success with no results",
			args: map[string]any{"SearchQuery": "nonexistent"},
			mockSetup: func(m *MockGradientAIService) {
				m.EXPECT().SearchModels(gomock.Any(), "nonexistent").Return([]string{}, nil, nil)
			},
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			mock := NewMockGradientAIService(ctrl)
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
		Provider:          "Meta",
		Url:               "https://example.com/model",
		Usecases:          []string{"chat", "completion"},
		ModelAvailability: "serverless",
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
		mockSetup     func(*MockGradientAIService)
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
			mockSetup: func(m *MockGradientAIService) {
				m.EXPECT().GetModelByUUID(gomock.Any(), "12345678-1234-1234-1234-123456789012").
					Return(nil, nil, errors.New("api error"))
			},
			expectError: true,
		},
		{
			name: "model not found",
			args: map[string]any{"ModelUUID": "99999999-9999-9999-9999-999999999999"},
			mockSetup: func(m *MockGradientAIService) {
				m.EXPECT().GetModelByUUID(gomock.Any(), "99999999-9999-9999-9999-999999999999").
					Return(nil, nil, nil)
			},
			expectError: true,
		},
		{
			name: "success",
			args: map[string]any{"ModelUUID": "12345678-1234-1234-1234-123456789012"},
			mockSetup: func(m *MockGradientAIService) {
				m.EXPECT().GetModelByUUID(gomock.Any(), "12345678-1234-1234-1234-123456789012").
					Return(testModel, nil, nil)
			},
			checkModel: true,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			mock := NewMockGradientAIService(ctrl)
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
					UUID              string          `json:"uuid"`
					Name              string          `json:"name"`
					Agreement         *godo.Agreement `json:"agreement,omitempty"`
					ModelAvailability string          `json:"model_availability,omitempty"`
				}
				err := json.Unmarshal([]byte(textContent.Text), &result)
				require.NoError(t, err)
				require.Equal(t, testModel.Uuid, result.UUID, "should return exact UUID")
				require.Equal(t, testModel.Name, result.Name, "should return exact name")
				require.Equal(t, testModel.ModelAvailability, result.ModelAvailability, "should return exact model availability")
				require.NotNil(t, result.Agreement, "should include agreement")
				require.Equal(t, testModel.Agreement.Name, result.Agreement.Name, "should return exact agreement name")
				require.Equal(t, testModel.Agreement.Description, result.Agreement.Description, "should return exact agreement description")
				require.Equal(t, testModel.Agreement.Url, result.Agreement.Url, "should return exact agreement URL")
				require.Equal(t, testModel.Agreement.Uuid, result.Agreement.Uuid, "should return exact agreement UUID")
			}
		})
	}
}

func TestModelTool_Tools(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mock := NewMockGradientAIService(ctrl)
	tool := setupModelToolWithMock(mock)

	tools := tool.Tools()
	require.Len(t, tools, 2)

	// Check that both tools are present
	toolNames := make(map[string]bool)
	for _, t := range tools {
		toolNames[t.Tool.Name] = true
	}

	require.True(t, toolNames["genai-model-catalog-search"], "should have genai-model-catalog-search tool")
	require.True(t, toolNames["genai-model-catalog-get-card"], "should have genai-model-catalog-get-card tool")
}
