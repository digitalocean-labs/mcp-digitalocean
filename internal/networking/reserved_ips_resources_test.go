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

func setupReservedIPResourceWithMocks(ipv4 *MockReservedIPsService, ipv6 *MockReservedIPV6sService) *ReservedIPMCPResource {
	client := &godo.Client{}
	client.ReservedIPs = ipv4
	client.ReservedIPV6s = ipv6
	return NewReservedIPMCPResource(client)
}

func TestReservedIPMCPResource_handleGetIPv4Resource(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	testIPv4 := &godo.ReservedIP{
		IP:     "192.0.2.1",
		Region: &godo.Region{Slug: "nyc3"},
	}
	tests := []struct {
		name        string
		uri         string
		mockSetup   func(*MockReservedIPsService)
		expectError bool
	}{
		{
			name: "Successful get IPv4",
			uri:  "reserved_ipv4://192.0.2.1",
			mockSetup: func(m *MockReservedIPsService) {
				m.EXPECT().
					Get(gomock.Any(), "192.0.2.1").
					Return(testIPv4, nil, nil).
					Times(1)
			},
		},
		{
			name:        "Invalid URI",
			uri:         "reserved_ipv4:192.0.2.1",
			mockSetup:   nil,
			expectError: true,
		},
		{
			name: "API error",
			uri:  "reserved_ipv4://203.0.113.1",
			mockSetup: func(m *MockReservedIPsService) {
				m.EXPECT().
					Get(gomock.Any(), "203.0.113.1").
					Return(nil, nil, errors.New("api error")).
					Times(1)
			},
			expectError: true,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			mockIPv4 := NewMockReservedIPsService(ctrl)
			mockIPv6 := NewMockReservedIPV6sService(ctrl)
			if tc.mockSetup != nil {
				tc.mockSetup(mockIPv4)
			}
			resource := setupReservedIPResourceWithMocks(mockIPv4, mockIPv6)
			req := mcp.ReadResourceRequest{
				Params: mcp.ReadResourceParams{
					URI: tc.uri,
				},
			}
			resp, err := resource.handleGetIPv4Resource(context.Background(), req)
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
			var outIP godo.ReservedIP
			require.NoError(t, json.Unmarshal([]byte(content.Text), &outIP))
			require.Equal(t, testIPv4.IP, outIP.IP)
		})
	}
}

func TestReservedIPMCPResource_handleGetIPv6Resource(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	testIPv6 := &godo.ReservedIPV6{
		IP:         "2001:db8::1",
		RegionSlug: "nyc3",
	}
	tests := []struct {
		name        string
		uri         string
		mockSetup   func(*MockReservedIPV6sService)
		expectError bool
	}{
		{
			name: "Successful get IPv6",
			uri:  "reserved_ipv6://2001:db8::1",
			mockSetup: func(m *MockReservedIPV6sService) {
				m.EXPECT().
					Get(gomock.Any(), "2001:db8::1").
					Return(testIPv6, nil, nil).
					Times(1)
			},
		},
		{
			name:        "Invalid URI",
			uri:         "reserved_ipv6:2001:db8::1",
			mockSetup:   nil,
			expectError: true,
		},
		{
			name: "API error",
			uri:  "reserved_ipv6://2001:db8::dead:beef",
			mockSetup: func(m *MockReservedIPV6sService) {
				m.EXPECT().
					Get(gomock.Any(), "2001:db8::dead:beef").
					Return(nil, nil, errors.New("api error")).
					Times(1)
			},
			expectError: true,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			mockIPv4 := NewMockReservedIPsService(ctrl)
			mockIPv6 := NewMockReservedIPV6sService(ctrl)
			if tc.mockSetup != nil {
				tc.mockSetup(mockIPv6)
			}
			resource := setupReservedIPResourceWithMocks(mockIPv4, mockIPv6)
			req := mcp.ReadResourceRequest{
				Params: mcp.ReadResourceParams{
					URI: tc.uri,
				},
			}
			resp, err := resource.handleGetIPv6Resource(context.Background(), req)
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
			var outIP godo.ReservedIPV6
			require.NoError(t, json.Unmarshal([]byte(content.Text), &outIP))
			require.Equal(t, testIPv6.IP, outIP.IP)
		})
	}
}
