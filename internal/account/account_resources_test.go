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

func setupAccountResourceWithMock(mockAccount *MockAccountService) *AccountMCPResource {
	client := &godo.Client{}
	client.Account = mockAccount
	return NewAccountMCPResource(client)
}

func TestAccountMCPResource_handleGetAccountResource(t *testing.T) {
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
			resource := setupAccountResourceWithMock(mockAccount)
			req := mcp.ReadResourceRequest{}
			resp, err := resource.handleGetAccountResource(context.Background(), req)
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
