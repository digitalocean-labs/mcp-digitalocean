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

func setupCDNMCPResourceWithMock(cdns *MockCDNService) *CDNMCPResource {
	client := &godo.Client{}
	client.CDNs = cdns
	return NewCDNMCPResource(client)
}

func TestCDNMCPResource_handleGetCDNResource(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	testCDN := &godo.CDN{
		ID:     "cdn-123",
		Origin: "origin.example.com",
		TTL:    3600,
	}
	tests := []struct {
		name        string
		uri         string
		mockSetup   func(*MockCDNService)
		expectError bool
	}{
		{
			name: "Successful get",
			uri:  "cdn://cdn-123",
			mockSetup: func(m *MockCDNService) {
				m.EXPECT().
					Get(gomock.Any(), "cdn-123").
					Return(testCDN, nil, nil).
					Times(1)
			},
		},
		{
			name: "API error",
			uri:  "cdn://cdn-456",
			mockSetup: func(m *MockCDNService) {
				m.EXPECT().
					Get(gomock.Any(), "cdn-456").
					Return(nil, nil, errors.New("api error")).
					Times(1)
			},
			expectError: true,
		},
		{
			name:        "Invalid URI",
			uri:         "cdn123",
			mockSetup:   nil,
			expectError: true,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			mockCDN := NewMockCDNService(ctrl)
			if tc.mockSetup != nil {
				tc.mockSetup(mockCDN)
			}
			resource := setupCDNMCPResourceWithMock(mockCDN)
			req := mcp.ReadResourceRequest{
				Params: mcp.ReadResourceParams{
					URI: tc.uri,
				},
			}
			resp, err := resource.handleGetCDNResource(context.Background(), req)
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
			var outCDN godo.CDN
			require.NoError(t, json.Unmarshal([]byte(content.Text), &outCDN))
			require.Equal(t, testCDN.ID, outCDN.ID)
		})
	}
}
