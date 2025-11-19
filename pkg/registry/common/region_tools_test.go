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

func setupRegionToolsWithMock(mockRegions *MockRegionsService) *RegionTools {
	client := func(ctx context.Context) (*godo.Client, error) {
		return &godo.Client{
			Regions: mockRegions,
		}, nil
	}

	return NewRegionTools(client)
}

func TestRegionTools_listRegions(t *testing.T) {
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
		page        float64
		perPage     float64
		mockSetup   func(*MockRegionsService)
		expectError bool
	}{
		{
			name:    "Successful list",
			page:    1,
			perPage: 2,
			mockSetup: func(m *MockRegionsService) {
				m.EXPECT().
					List(gomock.Any(), &godo.ListOptions{Page: 1, PerPage: 2}).
					Return(mockRegions, &godo.Response{}, nil).
					Times(1)
			},
		},
		{
			name:    "API error",
			page:    1,
			perPage: 2,
			mockSetup: func(m *MockRegionsService) {
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
			mockSetup: func(m *MockRegionsService) {
				m.EXPECT().
					List(gomock.Any(), &godo.ListOptions{Page: 1, PerPage: 50}).
					Return(mockRegions, &godo.Response{}, nil).
					Times(1)
			},
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			mockRegionsSvc := NewMockRegionsService(ctrl)
			if tc.mockSetup != nil {
				tc.mockSetup(mockRegionsSvc)
			}
			tool := setupRegionToolsWithMock(mockRegionsSvc)
			args := map[string]any{}
			if tc.page != 0 {
				args["Page"] = tc.page
			}
			if tc.perPage != 0 {
				args["PerPage"] = tc.perPage
			}
			req := mcp.CallToolRequest{Params: mcp.CallToolParams{Arguments: args}}
			resp, err := tool.listRegions(context.Background(), req)
			if tc.expectError {
				require.NotNil(t, resp)
				require.True(t, resp.IsError)
				return
			}
			require.NoError(t, err)
			require.NotNil(t, resp)
			require.False(t, resp.IsError)
			require.NotEmpty(t, resp.Content)
			content := resp.Content[0].(mcp.TextContent).Text
			var regionsOut []godo.Region
			require.NoError(t, json.Unmarshal([]byte(content), &regionsOut))
			require.Equal(t, mockRegions, regionsOut)
		})
	}
}
