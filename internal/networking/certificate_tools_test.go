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

func TestCertificateTool_createCertificate(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	testCert := &godo.Certificate{
		ID:   "cert-123",
		Name: "my-cert",
	}
	tests := []struct {
		name        string
		args        map[string]any
		mockSetup   func(*MockCertificatesService)
		expectError bool
	}{
		{
			name: "Successful create",
			args: map[string]any{
				"Name":             "my-cert",
				"PrivateKey":       "privkey",
				"LeafCertificate":  "leafcert",
				"CertificateChain": "chain",
			},
			mockSetup: func(m *MockCertificatesService) {
				m.EXPECT().
					Create(gomock.Any(), &godo.CertificateRequest{
						Name:             "my-cert",
						PrivateKey:       "privkey",
						LeafCertificate:  "leafcert",
						CertificateChain: "chain",
						Type:             "custom",
					}).
					Return(testCert, nil, nil).
					Times(1)
			},
		},
		{
			name: "API error",
			args: map[string]any{
				"Name":             "fail-cert",
				"PrivateKey":       "privkey",
				"LeafCertificate":  "leafcert",
				"CertificateChain": "chain",
			},
			mockSetup: func(m *MockCertificatesService) {
				m.EXPECT().
					Create(gomock.Any(), &godo.CertificateRequest{
						Name:             "fail-cert",
						PrivateKey:       "privkey",
						LeafCertificate:  "leafcert",
						CertificateChain: "chain",
						Type:             "custom",
					}).
					Return(nil, nil, errors.New("api error")).
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
			resp, err := tool.createCertificate(context.Background(), req)
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
