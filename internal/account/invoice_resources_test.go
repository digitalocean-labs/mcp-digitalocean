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

func setupInvoiceResourceWithMock(mockInvoices *MockInvoicesService) *InvoicesMCPResource {
	client := &godo.Client{}
	client.Invoices = mockInvoices
	return NewInvoicesMCPResource(client)
}

func TestInvoicesMCPResource_handleGetInvoiceResourceTemplate(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	tests := []struct {
		name        string
		perPage     int
		mockSetup   func(*MockInvoicesService)
		expectError bool
	}{
		{
			name:    "Successful get",
			perPage: 2,
			mockSetup: func(m *MockInvoicesService) {
				m.EXPECT().
					List(gomock.Any(), &godo.ListOptions{Page: 1, PerPage: 2}).
					Return(&godo.InvoiceList{}, nil, nil).
					Times(1)
			},
		},
		{
			name:    "API error",
			perPage: 3,
			mockSetup: func(m *MockInvoicesService) {
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
			mockInvoices := NewMockInvoicesService(ctrl)
			if tc.mockSetup != nil {
				tc.mockSetup(mockInvoices)
			}
			resource := setupInvoiceResourceWithMock(mockInvoices)
			// Simulate the URI extraction logic
			req := mcp.ReadResourceRequest{
				Params: mcp.ReadResourceParams{
					URI: fmt.Sprintf("invoice://%d", tc.perPage),
				},
			}
			resp, err := resource.handleGetInvoiceResourceTemplate(context.Background(), req)
			if tc.expectError {
				require.Error(t, err)
				return
			}
			require.NoError(t, err)
			require.NotNil(t, resp)
			require.Len(t, resp, 1)
		})
	}
}
