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

func setupBalanceToolsWithMock(mockBalance *MockBalanceService) *BalanceTools {
	client := func(ctx context.Context) (*godo.Client, error) {
		return &godo.Client{
			Balance: mockBalance,
		}, nil
	}

	return &BalanceTools{client: client}
}

func TestBalanceTools_getBalance(t *testing.T) {
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
			tool := setupBalanceToolsWithMock(mockBalance)
			req := mcp.CallToolRequest{Params: mcp.CallToolParams{Arguments: map[string]any{}}}
			resp, err := tool.getBalance(context.Background(), req)
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
