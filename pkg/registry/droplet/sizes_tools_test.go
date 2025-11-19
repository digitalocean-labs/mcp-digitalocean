package droplet

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

func setupSizesToolWithMock(sizes *MockSizesService) *SizesTool {
	client := func(ctx context.Context) (*godo.Client, error) {
		return &godo.Client{Sizes: sizes}, nil
	}
	return NewSizesTool(client)
}

func TestSizesTool_listSizes(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	testSizes := []godo.Size{
		{
			Slug:         "s-1vcpu-1gb",
			Memory:       1024,
			Vcpus:        1,
			Disk:         25,
			Transfer:     1.0,
			PriceMonthly: 5.0,
			PriceHourly:  0.00744,
			Available:    true,
			Regions:      []string{"nyc1", "sfo2"},
		},
		{
			Slug:         "s-2vcpu-2gb",
			Memory:       2048,
			Vcpus:        2,
			Disk:         50,
			Transfer:     2.0,
			PriceMonthly: 10.0,
			PriceHourly:  0.01488,
			Available:    true,
			Regions:      []string{"nyc3", "ams3"},
		},
	}
	tests := []struct {
		name        string
		page        float64
		perPage     float64
		mockSetup   func(*MockSizesService)
		expectError bool
	}{
		{
			name:    "Successful list with pagination",
			page:    2,
			perPage: 1,
			mockSetup: func(m *MockSizesService) {
				m.EXPECT().
					List(gomock.Any(), &godo.ListOptions{Page: 2, PerPage: 1}).
					Return(testSizes, &godo.Response{}, nil).
					Times(1)
			},
		},
		{
			name:    "API error",
			page:    1,
			perPage: 2,
			mockSetup: func(m *MockSizesService) {
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
			mockSetup: func(m *MockSizesService) {
				m.EXPECT().
					List(gomock.Any(), &godo.ListOptions{Page: 1, PerPage: 50}).
					Return(testSizes, &godo.Response{}, nil).
					Times(1)
			},
		},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			mockSizes := NewMockSizesService(ctrl)
			if tc.mockSetup != nil {
				tc.mockSetup(mockSizes)
			}
			tool := setupSizesToolWithMock(mockSizes)
			args := map[string]any{}
			if tc.page != 0 {
				args["Page"] = tc.page
			}
			if tc.perPage != 0 {
				args["PerPage"] = tc.perPage
			}
			req := mcp.CallToolRequest{Params: mcp.CallToolParams{Arguments: args}}
			resp, err := tool.listSizes(context.Background(), req)
			if tc.expectError {
				require.NotNil(t, resp)
				require.True(t, resp.IsError)
				return
			}
			require.NoError(t, err)
			require.NotNil(t, resp)
			require.False(t, resp.IsError)
			require.NotEmpty(t, resp.Content)
			// Check that the returned JSON can be unmarshaled into a slice of maps
			var out []map[string]any
			require.NoError(t, json.Unmarshal([]byte(resp.Content[0].(mcp.TextContent).Text), &out))
			require.GreaterOrEqual(t, len(out), 1)
			require.Contains(t, out[0], "slug")
			require.Contains(t, out[0], "available")
			require.Contains(t, out[0], "price_monthly")
			require.Contains(t, out[0], "price_hourly")
			require.Contains(t, out[0], "memory")
			require.Contains(t, out[0], "vcpus")
			require.Contains(t, out[0], "disk")
			require.Contains(t, out[0], "transfer")
			require.Contains(t, out[0], "regions")
		})
	}
}
