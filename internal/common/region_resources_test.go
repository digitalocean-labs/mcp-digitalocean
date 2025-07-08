package common

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

func setupRegionResourceWithMock(mockRegions *MockRegionsService) *RegionMCPResource {
	client := &godo.Client{}
	client.Regions = mockRegions
	return NewRegionMCPResource(client)
}

func TestRegionMCPResource_handleGetRegionsResource(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockRegions := []godo.Region{
		{
			Slug:      "nyc1",
			Name:      "New York 1",
			Available: true,
			Features:  []string{"ipv6", "backups"},
		},
		{
			Slug:      "sfo2",
			Name:      "San Francisco 2",
			Available: false,
			Features:  []string{"private_networking"},
		},
	}

	tests := []struct {
		name        string
		mockSetup   func(*MockRegionsService)
		expectError bool
	}{
		{
			name: "Successful get",
			mockSetup: func(m *MockRegionsService) {
				m.EXPECT().
					List(gomock.Any(), gomock.Any()).
					Return(mockRegions, &godo.Response{}, nil).
					Times(1)
			},
		},
		{
			name: "API error",
			mockSetup: func(m *MockRegionsService) {
				m.EXPECT().
					List(gomock.Any(), gomock.Any()).
					Return(nil, nil, errors.New("api error")).
					Times(1)
			},
			expectError: true,
		},
		// Note: JSON serialization error is not practical with godo.Region, so not included.
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			mockRegionsSvc := NewMockRegionsService(ctrl)
			if tc.mockSetup != nil {
				tc.mockSetup(mockRegionsSvc)
			}
			resource := setupRegionResourceWithMock(mockRegionsSvc)
			req := mcp.ReadResourceRequest{
				Params: mcp.ReadResourceParams{
					URI: RegionsURI + "all",
				},
			}
			resp, err := resource.handleGetRegionsResource(context.Background(), req)
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
			require.Equal(t, req.Params.URI, content.URI)
			require.Equal(t, "application/json", content.MIMEType)
			var regionsOut []godo.Region
			require.NoError(t, json.Unmarshal([]byte(content.Text), &regionsOut))
			require.Equal(t, mockRegions, regionsOut)
		})
	}
}
