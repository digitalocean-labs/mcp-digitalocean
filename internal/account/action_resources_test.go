package account

import (
	"context"
	"errors"
	"fmt"
	"testing"

	"github.com/digitalocean/godo"
	"github.com/mark3labs/mcp-go/mcp"
	"github.com/stretchr/testify/require"
	"go.uber.org/mock/gomock"
)

func setupActionResourceWithMock(mockActions *MockActionsService) *ActionMCPResource {
	client := &godo.Client{}
	client.Actions = mockActions
	return NewActionMCPResource(client)
}

func TestActionMCPResource_handleGetActionResourceTemplate(t *testing.T) {
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
		actionID    int
		mockSetup   func(*MockActionsService)
		expectError bool
	}{
		{
			name:     "Successful get",
			actionID: 123456,
			mockSetup: func(m *MockActionsService) {
				m.EXPECT().
					Get(gomock.Any(), 123456).
					Return(testAction, nil, nil).
					Times(1)
			},
		},
		{
			name:     "API error",
			actionID: 654321,
			mockSetup: func(m *MockActionsService) {
				m.EXPECT().
					Get(gomock.Any(), 654321).
					Return(nil, nil, errors.New("api error")).
					Times(1)
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
			resource := setupActionResourceWithMock(mockActions)
			// Simulate the URI extraction logic
			req := mcp.ReadResourceRequest{
				Params: mcp.ReadResourceParams{
					URI: fmt.Sprintf("actions://%d", tc.actionID),
				},
			}
			resp, err := resource.handleGetActionResourceTemplate(context.Background(), req)
			if tc.expectError {
				require.Error(t, err)
				return
			}
			require.NoError(t, err)
			require.NotNil(t, resp)
			require.Len(t, resp, 1)
			content, ok := resp[0].(mcp.TextResourceContents)
			require.True(t, ok)
			require.Equal(t, "application/json", content.MIMEType)
		})
	}
}
