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

func setupVPCMCPResourceWithMock(vpcs *MockVPCsService) *VPCMCPResource {
	client := &godo.Client{}
	client.VPCs = vpcs
	return NewVPCMCPResource(client)
}

func TestVPCMCPResource_handleGetVPCResource(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	testVPC := &godo.VPC{
		ID:         "vpc-123",
		Name:       "private-net",
		RegionSlug: "nyc3",
	}

	tests := []struct {
		name        string
		uri         string
		mockSetup   func(*MockVPCsService)
		expectError bool
	}{
		{
			name: "Successful get",
			uri:  "vpcs://vpc-123",
			mockSetup: func(m *MockVPCsService) {
				m.EXPECT().
					Get(gomock.Any(), "vpc-123").
					Return(testVPC, nil, nil).
					Times(1)
			},
		},
		{
			name: "API error",
			uri:  "vpcs://vpc-456",
			mockSetup: func(m *MockVPCsService) {
				m.EXPECT().
					Get(gomock.Any(), "vpc-456").
					Return(nil, nil, errors.New("api error")).
					Times(1)
			},
			expectError: true,
		},
		{
			name:        "Invalid URI",
			uri:         "vpcs123",
			mockSetup:   nil,
			expectError: true,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			mockVPCs := NewMockVPCsService(ctrl)
			if tc.mockSetup != nil {
				tc.mockSetup(mockVPCs)
			}
			resource := setupVPCMCPResourceWithMock(mockVPCs)
			req := mcp.ReadResourceRequest{
				Params: mcp.ReadResourceParams{
					URI: tc.uri,
				},
			}
			resp, err := resource.handleGetVPCResource(context.Background(), req)
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
			var outVPC godo.VPC
			require.NoError(t, json.Unmarshal([]byte(content.Text), &outVPC))
			require.Equal(t, testVPC.ID, outVPC.ID)
		})
	}
}
