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

func setupBalanceResourceWithMock(mockBalance *MockBalanceService) *BalanceMCPResource {
	client := &godo.Client{}
	client.Balance = mockBalance
	return NewBalanceMCPResource(client)
}

func TestBalanceMCPResource_handleGetBalanceResource(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	testBalance := &godo.Balance{
		MonthToDateBalance: "10.00",
		AccountBalance:     "5.00",
		MonthToDateUsage:   "15.00",
	}

	tests := []struct {
		name        string
		mockSetup   func(*MockBalanceService)
		expectError bool
	}{
		{
			name: "Successful get",
			mockSetup: func(m *MockBalanceService) {
				m.EXPECT().
					Get(gomock.Any()).
					Return(testBalance, nil, nil).
					Times(1)
			},
		},
		{
			name: "API error",
			mockSetup: func(m *MockBalanceService) {
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
			mockBalance := NewMockBalanceService(ctrl)
			if tc.mockSetup != nil {
				tc.mockSetup(mockBalance)
			}
			resource := setupBalanceResourceWithMock(mockBalance)
			req := mcp.ReadResourceRequest{}
			resp, err := resource.handleGetBalanceResource(context.Background(), req)
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
