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

func setupBillingToolsWithMock(mockBilling *MockBillingHistoryService) *BillingTools {
	client := func(ctx context.Context) (*godo.Client, error) {
		return &godo.Client{
			BillingHistory: mockBilling,
		}, nil
	}

	return &BillingTools{client: client}
}

func TestBillingTools_listBillingHistory(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	testBillingHistory := &godo.BillingHistory{
		BillingHistory: []godo.BillingHistoryEntry{
			{
				Description: "Droplet usage",
				Amount:      "10.00",
				Type:        "usage",
			},
			{
				Description: "App Platform usage",
				Amount:      "5.00",
				Type:        "usage",
			},
		},
	}

	tests := []struct {
		name        string
		page        float64
		perPage     float64
		mockSetup   func(*MockBillingHistoryService)
		expectError bool
	}{
		{
			name:    "Successful list",
			page:    1,
			perPage: 2,
			mockSetup: func(m *MockBillingHistoryService) {
				m.EXPECT().
					List(gomock.Any(), &godo.ListOptions{Page: 1, PerPage: 2}).
					Return(testBillingHistory, nil, nil).
					Times(1)
			},
		},
		{
			name:    "API error",
			page:    1,
			perPage: 3,
			mockSetup: func(m *MockBillingHistoryService) {
				m.EXPECT().
					List(gomock.Any(), &godo.ListOptions{Page: 1, PerPage: 3}).
					Return(nil, nil, errors.New("api error")).
					Times(1)
			},
			expectError: true,
		},
		{
			name:    "Default pagination",
			page:    0,
			perPage: 0,
			mockSetup: func(m *MockBillingHistoryService) {
				m.EXPECT().
					List(gomock.Any(), &godo.ListOptions{Page: 1, PerPage: 30}).
					Return(testBillingHistory, nil, nil).
					Times(1)
			},
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			mockBilling := NewMockBillingHistoryService(ctrl)
			if tc.mockSetup != nil {
				tc.mockSetup(mockBilling)
			}
			tool := setupBillingToolsWithMock(mockBilling)
			args := map[string]any{}
			if tc.page != 0 {
				args["Page"] = tc.page
			}
			if tc.perPage != 0 {
				args["PerPage"] = tc.perPage
			}
			req := mcp.CallToolRequest{Params: mcp.CallToolParams{Arguments: args}}
			resp, err := tool.listBillingHistory(context.Background(), req)
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
