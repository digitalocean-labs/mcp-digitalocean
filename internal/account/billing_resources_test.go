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

func setupBillingResourceWithMock(mockBilling *MockBillingHistoryService) *BillingMCPResource {
	client := &godo.Client{}
	client.BillingHistory = mockBilling
	return NewBillingMCPResource(client)
}

func TestBillingMCPResource_handleGetBillingResourceTemplate(t *testing.T) {
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
		perPage     int
		mockSetup   func(*MockBillingHistoryService)
		expectError bool
	}{
		{
			name:    "Successful get",
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
			perPage: 3,
			mockSetup: func(m *MockBillingHistoryService) {
				m.EXPECT().
					List(gomock.Any(), &godo.ListOptions{Page: 1, PerPage: 3}).
					Return(nil, nil, errors.New("api error")).
					Times(1)
			},
			expectError: true,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			mockBilling := NewMockBillingHistoryService(ctrl)
			if tc.mockSetup != nil {
				tc.mockSetup(mockBilling)
			}
			resource := setupBillingResourceWithMock(mockBilling)
			// Simulate the URI extraction logic
			req := mcp.ReadResourceRequest{
				Params: mcp.ReadResourceParams{
					URI: fmt.Sprintf("billing://%d", tc.perPage),
				},
			}
			resp, err := resource.handleGetBillingResourceTemplate(context.Background(), req)
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
