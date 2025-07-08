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

func setupSizesResourceWithMock(sizes *MockSizesService) *SizesMCPResource {
	client := &godo.Client{}
	client.Sizes = sizes
	return NewSizesMCPResource(client)
}

func TestSizesMCPResource_handleGetSizeResource(t *testing.T) {
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
		mockSetup   func(*MockSizesService)
		expectError bool
	}{
		{
			name: "Successful list",
			mockSetup: func(m *MockSizesService) {
				m.EXPECT().
					List(gomock.Any(), gomock.Any()).
					Return(testSizes, &godo.Response{}, nil).
					Times(1)
			},
		},
		{
			name: "API error",
			mockSetup: func(m *MockSizesService) {
				m.EXPECT().
					List(gomock.Any(), gomock.Any()).
					Return(nil, nil, errors.New("api error")).
					Times(1)
			},
			expectError: true,
		},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			mockSizes := NewMockSizesService(ctrl)
			if tc.mockSetup != nil {
				tc.mockSetup(mockSizes)
			}
			resource := setupSizesResourceWithMock(mockSizes)
			req := mcp.ReadResourceRequest{
				Params: mcp.ReadResourceParams{
					URI: SizesURI + "all",
				},
			}
			resp, err := resource.handleGetSizeResource(context.Background(), req)
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
			var outSizes []godo.Size
			require.NoError(t, json.Unmarshal([]byte(content.Text), &outSizes))
		})
	}
}
