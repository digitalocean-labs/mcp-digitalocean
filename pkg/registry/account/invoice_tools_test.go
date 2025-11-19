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

func setupInvoiceToolsWithMock(mockInvoices *MockInvoicesService) *InvoiceTools {
	client := func(ctx context.Context) (*godo.Client, error) {
		return &godo.Client{
			Invoices: mockInvoices,
		}, nil
	}

	return &InvoiceTools{client: client}
}

func TestInvoiceTools_listInvoices(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	testInvoices := &godo.InvoiceList{
		Invoices: []godo.InvoiceListItem{
			{InvoiceUUID: "inv-1", Amount: "10.00"},
			{InvoiceUUID: "inv-2", Amount: "20.00"},
		},
	}
	tests := []struct {
		name        string
		page        float64
		perPage     float64
		mockSetup   func(*MockInvoicesService)
		expectError bool
	}{
		{
			name:    "Successful list",
			page:    1,
			perPage: 2,
			mockSetup: func(m *MockInvoicesService) {
				m.EXPECT().
					List(gomock.Any(), &godo.ListOptions{Page: 1, PerPage: 2}).
					Return(testInvoices, nil, nil).
					Times(1)
			},
		},
		{
			name:    "API error",
			page:    1,
			perPage: 2,
			mockSetup: func(m *MockInvoicesService) {
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
			mockSetup: func(m *MockInvoicesService) {
				m.EXPECT().
					List(gomock.Any(), &godo.ListOptions{Page: 1, PerPage: 30}).
					Return(testInvoices, nil, nil).
					Times(1)
			},
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			mockInvoices := NewMockInvoicesService(ctrl)
			if tc.mockSetup != nil {
				tc.mockSetup(mockInvoices)
			}
			tool := setupInvoiceToolsWithMock(mockInvoices)
			args := map[string]any{}
			if tc.page != 0 {
				args["Page"] = tc.page
			}
			if tc.perPage != 0 {
				args["PerPage"] = tc.perPage
			}
			req := mcp.CallToolRequest{Params: mcp.CallToolParams{Arguments: args}}
			resp, err := tool.listInvoices(context.Background(), req)
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

func TestInvoiceTools_getInvoice(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	testInvoice := &godo.Invoice{
		InvoiceItems: []godo.InvoiceItem{
			{ResourceID: "item-1", Amount: "10.00"},
			{ResourceID: "item-2", Amount: "20.00"},
		},
	}
	tests := []struct {
		invoiceUUID string
		name        string
		page        float64
		perPage     float64
		mockSetup   func(*MockInvoicesService)
		expectError bool
	}{
		{
			name:        "Successful list",
			invoiceUUID: "inv-123",
			page:        1,
			perPage:     2,
			mockSetup: func(m *MockInvoicesService) {
				m.EXPECT().
					Get(gomock.Any(), "inv-123", &godo.ListOptions{Page: 1, PerPage: 2}).
					Return(testInvoice, nil, nil).
					Times(1)
			},
		},
		{
			name:        "API error",
			invoiceUUID: "inv-123",
			page:        1,
			perPage:     2,
			mockSetup: func(m *MockInvoicesService) {
				m.EXPECT().
					Get(gomock.Any(), "inv-123", &godo.ListOptions{Page: 1, PerPage: 2}).
					Return(nil, nil, errors.New("api error")).
					Times(1)
			},
			expectError: true,
		},
		{
			name:        "Default pagination",
			invoiceUUID: "inv-123",
			page:        0,
			perPage:     0,
			mockSetup: func(m *MockInvoicesService) {
				m.EXPECT().
					Get(gomock.Any(), "inv-123", &godo.ListOptions{Page: 1, PerPage: 30}).
					Return(testInvoice, nil, nil).
					Times(1)
			},
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			mockInvoices := NewMockInvoicesService(ctrl)
			if tc.mockSetup != nil {
				tc.mockSetup(mockInvoices)
			}
			tool := setupInvoiceToolsWithMock(mockInvoices)
			args := map[string]any{}
			if tc.invoiceUUID != "" {
				args["InvoiceUUID"] = tc.invoiceUUID
			}
			if tc.page != 0 {
				args["Page"] = tc.page
			}
			if tc.perPage != 0 {
				args["PerPage"] = tc.perPage
			}

			req := mcp.CallToolRequest{Params: mcp.CallToolParams{Arguments: args}}
			resp, err := tool.getInvoice(context.Background(), req)
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
