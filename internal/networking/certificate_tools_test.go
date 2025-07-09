package networking

import (
	"context"
	"encoding/json"
	"errors"
	"testing"

	"github.com/digitalocean/godo"
	"github.com/mark3labs/mcp-go/mcp"
	"github.com/stretchr/testify/require"
	"go.uber.org/mock/gomock"
)

func setupCertificateToolWithMock(cert *MockCertificatesService) *CertificateTool {
	client := &godo.Client{}
	client.Certificates = cert
	return NewCertificateTool(client)
}

func TestCertificateTool_getCertificate(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	testCert := &godo.Certificate{
		ID:   "cert-123",
		Name: "my-cert",
	}
	tests := []struct {
		name        string
		id          string
		mockSetup   func(*MockCertificatesService)
		expectError bool
	}{
		{
			name: "Successful get",
			id:   "cert-123",
			mockSetup: func(m *MockCertificatesService) {
				m.EXPECT().
					Get(gomock.Any(), "cert-123").
					Return(testCert, nil, nil).
					Times(1)
			},
		},
		{
			name: "API error",
			id:   "fail-456",
			mockSetup: func(m *MockCertificatesService) {
				m.EXPECT().
					Get(gomock.Any(), "fail-456").
					Return(nil, nil, errors.New("api error")).
					Times(1)
			},
			expectError: true,
		},
		{
			name:        "Missing ID argument",
			id:          "",
			mockSetup:   nil,
			expectError: true,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			mockCert := NewMockCertificatesService(ctrl)
			if tc.mockSetup != nil {
				tc.mockSetup(mockCert)
			}
			tool := setupCertificateToolWithMock(mockCert)
			args := map[string]any{}
			if tc.name != "Missing ID argument" {
				args["ID"] = tc.id
			}
			req := mcp.CallToolRequest{Params: mcp.CallToolParams{Arguments: args}}
			resp, err := tool.getCertificate(context.Background(), req)
			if tc.expectError {
				require.NotNil(t, resp)
				require.True(t, resp.IsError)
				return
			}
			require.NoError(t, err)
			require.NotNil(t, resp)
			require.False(t, resp.IsError)
			var outCert godo.Certificate
			require.NoError(t, json.Unmarshal([]byte(resp.Content[0].(mcp.TextContent).Text), &outCert))
			require.Equal(t, testCert.ID, outCert.ID)
		})
	}
}

func TestCertificateTool_listCertificates(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	testCerts := []godo.Certificate{
		{ID: "cert-1", Name: "cert1"},
		{ID: "cert-2", Name: "cert2"},
	}
	tests := []struct {
		name        string
		page        float64
		perPage     float64
		mockSetup   func(*MockCertificatesService)
		expectError bool
	}{
		{
			name:    "Successful list",
			page:    2,
			perPage: 1,
			mockSetup: func(m *MockCertificatesService) {
				m.EXPECT().
					List(gomock.Any(), &godo.ListOptions{Page: 2, PerPage: 1}).
					Return(testCerts, nil, nil).
					Times(1)
			},
		},
		{
			name:    "API error",
			page:    1,
			perPage: 2,
			mockSetup: func(m *MockCertificatesService) {
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
			mockSetup: func(m *MockCertificatesService) {
				m.EXPECT().
					List(gomock.Any(), &godo.ListOptions{Page: 1, PerPage: 20}).
					Return(testCerts, nil, nil).
					Times(1)
			},
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			mockCert := NewMockCertificatesService(ctrl)
			if tc.mockSetup != nil {
				tc.mockSetup(mockCert)
			}
			tool := setupCertificateToolWithMock(mockCert)
			args := map[string]any{}
			if tc.page != 0 {
				args["Page"] = tc.page
			}
			if tc.perPage != 0 {
				args["PerPage"] = tc.perPage
			}
			req := mcp.CallToolRequest{Params: mcp.CallToolParams{Arguments: args}}
			resp, err := tool.listCertificates(context.Background(), req)
			if tc.expectError {
				require.NotNil(t, resp)
				require.True(t, resp.IsError)
				return
			}
			require.NoError(t, err)
			require.NotNil(t, resp)
			require.False(t, resp.IsError)
			require.NotEmpty(t, resp.Content)
			var outCerts []godo.Certificate
			require.NoError(t, json.Unmarshal([]byte(resp.Content[0].(mcp.TextContent).Text), &outCerts))
			require.GreaterOrEqual(t, len(outCerts), 1)
		})
	}
}

func TestCertificateTool_deleteCertificate(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	tests := []struct {
		name        string
		args        map[string]any
		mockSetup   func(*MockCertificatesService)
		expectError bool
		expectText  string
	}{
		{
			name: "Successful delete",
			args: map[string]any{"ID": "cert-123"},
			mockSetup: func(m *MockCertificatesService) {
				m.EXPECT().
					Delete(gomock.Any(), "cert-123").
					Return(&godo.Response{}, nil).
					Times(1)
			},
			expectText: "Certificate deleted successfully",
		},
		{
			name: "API error",
			args: map[string]any{"ID": "cert-456"},
			mockSetup: func(m *MockCertificatesService) {
				m.EXPECT().
					Delete(gomock.Any(), "cert-456").
					Return(nil, errors.New("api error")).
					Times(1)
			},
			expectError: true,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			mockCert := NewMockCertificatesService(ctrl)
			if tc.mockSetup != nil {
				tc.mockSetup(mockCert)
			}
			tool := setupCertificateToolWithMock(mockCert)
			req := mcp.CallToolRequest{Params: mcp.CallToolParams{Arguments: tc.args}}
			resp, err := tool.deleteCertificate(context.Background(), req)
			if tc.expectError {
				require.NotNil(t, resp)
				require.True(t, resp.IsError)
				return
			}
			require.NoError(t, err)
			require.NotNil(t, resp)
			require.False(t, resp.IsError)
			require.Contains(t, resp.Content[0].(mcp.TextContent).Text, tc.expectText)
		})
	}
}
