package account

import (
	"context"
	"errors"
	"testing"

	"github.com/digitalocean/godo"
	"github.com/mark3labs/mcp-go/mcp"
	"github.com/stretchr/testify/require"
	"go.uber.org/mock/gomock"
)

func setupActionToolsWithMock(mockActions *MockActionsService) *ActionTools {
	client := func(ctx context.Context) (*godo.Client, error) {
		return &godo.Client{
			Actions: mockActions,
		}, nil
	}

	return &ActionTools{client: client}
}

func TestActionTools_getAction(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	testAction := &godo.Action{
		ID:           123456,
		Status:       "completed",
		Type:         "create",
		ResourceType: "droplet",
		RegionSlug:   "nyc3",
	}
	tests := []struct {
		name        string
		id          float64
		mockSetup   func(*MockActionsService)
		expectError bool
	}{
		{
			name: "Successful get",
			id:   123456,
			mockSetup: func(m *MockActionsService) {
				m.EXPECT().
					Get(gomock.Any(), 123456).
					Return(testAction, nil, nil).
					Times(1)
			},
		},
		{
			name: "API error",
			id:   654321,
			mockSetup: func(m *MockActionsService) {
				m.EXPECT().
					Get(gomock.Any(), 654321).
					Return(nil, nil, errors.New("api error")).
					Times(1)
			},
			expectError: true,
		},
		{
			name: "Missing ID argument",
			id:   0,
			mockSetup: func(m *MockActionsService) {
				// No call expected
			},
			expectError: true,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			mockActions := NewMockActionsService(ctrl)
			if tc.mockSetup != nil {
				tc.mockSetup(mockActions)
			}
			tool := setupActionToolsWithMock(mockActions)
			args := map[string]any{}
			if tc.name != "Missing ID argument" {
				args["ID"] = tc.id
			}
			req := mcp.CallToolRequest{Params: mcp.CallToolParams{Arguments: args}}
			resp, err := tool.getAction(context.Background(), req)
			if tc.expectError {
				require.NotNil(t, resp)
				require.True(t, resp.IsError)
				return
			}
			require.NoError(t, err)
			require.NotNil(t, resp)
			require.False(t, resp.IsError)
			require.NotEmpty(t, resp.Content)
		})
	}
}

func TestActionTools_listActions(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	testActions := []godo.Action{
		{ID: 1, Status: "completed"},
		{ID: 2, Status: "in-progress"},
	}
	tests := []struct {
		name        string
		page        float64
		perPage     float64
		mockSetup   func(*MockActionsService)
		expectError bool
	}{
		{
			name:    "Successful list",
			page:    1,
			perPage: 2,
			mockSetup: func(m *MockActionsService) {
				m.EXPECT().
					List(gomock.Any(), &godo.ListOptions{Page: 1, PerPage: 2}).
					Return(testActions, nil, nil).
					Times(1)
			},
		},
		{
			name:    "API error",
			page:    1,
			perPage: 2,
			mockSetup: func(m *MockActionsService) {
				m.EXPECT().
					List(gomock.Any(), &godo.ListOptions{Page: 1, PerPage: 2}).
					Return(nil, nil, errors.New("api error")).
					Times(1)
			},
			expectError: true,
		},
		{
			name:    "Default pagination",
			page:    0,
			perPage: 0,
			mockSetup: func(m *MockActionsService) {
				m.EXPECT().
					List(gomock.Any(), &godo.ListOptions{Page: 1, PerPage: 30}).
					Return(testActions, nil, nil).
					Times(1)
			},
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			mockActions := NewMockActionsService(ctrl)
			if tc.mockSetup != nil {
				tc.mockSetup(mockActions)
			}
			tool := setupActionToolsWithMock(mockActions)
			args := map[string]any{}
			if tc.page != 0 {
				args["Page"] = tc.page
			}
			if tc.perPage != 0 {
				args["PerPage"] = tc.perPage
			}
			req := mcp.CallToolRequest{Params: mcp.CallToolParams{Arguments: args}}
			resp, err := tool.listActions(context.Background(), req)
			if tc.expectError {
				require.NotNil(t, resp)
				require.True(t, resp.IsError)
				return
			}
			require.NoError(t, err)
			require.NotNil(t, resp)
			require.False(t, resp.IsError)
			require.NotEmpty(t, resp.Content)
		})
	}
}
