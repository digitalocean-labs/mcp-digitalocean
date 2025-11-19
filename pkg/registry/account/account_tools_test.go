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

func setupAccountToolsWithMock(mockAccount *MockAccountService) *AccountTools {
	client := func(ctx context.Context) (*godo.Client, error) {
		return &godo.Client{
			Account: mockAccount,
		}, nil
	}
	return NewAccountTools(client)
}

func TestAccountTools_handleGetAccountInformation(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	testAccount := &godo.Account{
		UUID:   "abc-123",
		Email:  "test@example.com",
		Status: "active",
	}
	tests := []struct {
		name        string
		mockSetup   func(*MockAccountService)
		expectError bool
	}{
		{
			name: "Successful get",
			mockSetup: func(m *MockAccountService) {
				m.EXPECT().
					Get(gomock.Any()).
					Return(testAccount, nil, nil).
					Times(1)
			},
		},
		{
			name: "API error",
			mockSetup: func(m *MockAccountService) {
				m.EXPECT().
					Get(gomock.Any()).
					Return(nil, nil, errors.New("api error")).
					Times(1)
			},
			expectError: true,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			mockAccount := NewMockAccountService(ctrl)
			if tc.mockSetup != nil {
				tc.mockSetup(mockAccount)
			}
			tool := setupAccountToolsWithMock(mockAccount)
			req := mcp.CallToolRequest{}
			resp, err := tool.getAccountInformation(context.Background(), req)
			if tc.expectError {
				require.NotNil(t, resp)
				require.True(t, resp.IsError)
				return
			}
			require.NoError(t, err)
			require.NotNil(t, resp)
			require.False(t, resp.IsError)
			content := resp.Content[0].(mcp.TextContent).Text
			require.NotEmpty(t, content)
		})
	}
}
