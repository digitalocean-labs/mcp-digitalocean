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

func setupCertificateResourceWithMock(cert *MockCertificatesService) *CertificateMCPResource {
	client := &godo.Client{}
	client.Certificates = cert
	return NewCertificateMCPResource(client)
}

func TestCertificateMCPResource_handleGetCertificateResource(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	testCert := &godo.Certificate{
		ID:   "cert-123",
		Name: "my-cert",
	}
	tests := []struct {
		name        string
		uri         string
		mockSetup   func(*MockCertificatesService)
		expectError bool
	}{
		{
			name: "Successful get",
			uri:  "certificates://cert-123",
			mockSetup: func(m *MockCertificatesService) {
				m.EXPECT().
					Get(gomock.Any(), "cert-123").
					Return(testCert, nil, nil).
					Times(1)
			},
		},
		{
			name: "API error",
			uri:  "certificates://fail-456",
			mockSetup: func(m *MockCertificatesService) {
				m.EXPECT().
					Get(gomock.Any(), "fail-456").
					Return(nil, nil, errors.New("api error")).
					Times(1)
			},
			expectError: true,
		},
		{
			name:        "Invalid URI",
			uri:         "certificates123",
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
			resource := setupCertificateResourceWithMock(mockCert)
			req := mcp.ReadResourceRequest{
				Params: mcp.ReadResourceParams{
					URI: tc.uri,
				},
			}
			resp, err := resource.handleGetCertificateResource(context.Background(), req)
			if tc.expectError {
				require.Error(t, err)
				require.Nil(t, resp)
				return
			}
			require.NoError(t, err)
			require.NotNil(t, resp)
			require.Len(t, resp, 1)
			content, ok := resp[0].(mcp.TextResourceContents)
			require.True(t, ok)
			require.Equal(t, tc.uri, content.URI)
			require.Equal(t, "application/json", content.MIMEType)
			var outCert godo.Certificate
			require.NoError(t, json.Unmarshal([]byte(content.Text), &outCert))
			require.Equal(t, testCert.ID, outCert.ID)
		})
	}
}
