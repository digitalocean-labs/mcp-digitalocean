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

func setupVPCPeeringResourceWithMock(vpcs *MockVPCsService) *VPCPeeringMCPResource {
	client := &godo.Client{}
	client.VPCs = vpcs
	return NewVPCPeeringMCPResource(client)
}

func TestVPCPeeringMCPResource_handleGetVPCPeering(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	testPeering := &godo.VPCPeering{
		ID:   "peer-123",
		Name: "test-peering",
	}
	tests := []struct {
		name        string
		uri         string
		mockSetup   func(*MockVPCsService)
		expectError bool
	}{
		{
			name: "Successful get",
			uri:  "vpc_peering://peer-123",
			mockSetup: func(m *MockVPCsService) {
				m.EXPECT().
					GetVPCPeering(gomock.Any(), "peer-123").
					Return(testPeering, nil, nil).
					Times(1)
			},
		},
		{
			name: "API error",
			uri:  "vpc_peering://peer-456",
			mockSetup: func(m *MockVPCsService) {
				m.EXPECT().
					GetVPCPeering(gomock.Any(), "peer-456").
					Return(nil, nil, errors.New("api error")).
					Times(1)
			},
			expectError: true,
		},
		{
			name:        "Invalid URI",
			uri:         "vpc_peering-peer-789",
			mockSetup:   nil,
			expectError: true,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			mockVPC := NewMockVPCsService(ctrl)
			if tc.mockSetup != nil {
				tc.mockSetup(mockVPC)
			}
			resource := setupVPCPeeringResourceWithMock(mockVPC)
			req := mcp.ReadResourceRequest{
				Params: mcp.ReadResourceParams{
					URI: tc.uri,
				},
			}
			resp, err := resource.handleGetVPCPeering(context.Background(), req)
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
			var outPeering godo.VPCPeering
			require.NoError(t, json.Unmarshal([]byte(content.Text), &outPeering))
			require.Equal(t, testPeering.ID, outPeering.ID)
		})
	}
}
