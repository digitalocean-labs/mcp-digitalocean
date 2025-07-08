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

func setupFirewallResourceWithMock(firewalls *MockFirewallsService) *FirewallMCPResource {
	client := &godo.Client{}
	client.Firewalls = firewalls
	return NewFirewallMCPResource(client)
}

func TestFirewallMCPResource_handleGetFirewallResource(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	testFirewall := &godo.Firewall{
		ID:   "fw-123",
		Name: "test-fw",
	}

	tests := []struct {
		name        string
		uri         string
		mockSetup   func(*MockFirewallsService)
		expectError bool
	}{
		{
			name: "Successful get",
			uri:  "firewalls://fw-123",
			mockSetup: func(m *MockFirewallsService) {
				m.EXPECT().
					Get(gomock.Any(), "fw-123").
					Return(testFirewall, nil, nil).
					Times(1)
			},
		},
		{
			name: "API error",
			uri:  "firewalls://fw-456",
			mockSetup: func(m *MockFirewallsService) {
				m.EXPECT().
					Get(gomock.Any(), "fw-456").
					Return(nil, nil, errors.New("api error")).
					Times(1)
			},
			expectError: true,
		},
		{
			name:        "Invalid URI",
			uri:         "firewallsfw-123",
			mockSetup:   nil,
			expectError: true,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			mockFirewalls := NewMockFirewallsService(ctrl)
			if tc.mockSetup != nil {
				tc.mockSetup(mockFirewalls)
			}
			resource := setupFirewallResourceWithMock(mockFirewalls)
			req := mcp.ReadResourceRequest{
				Params: mcp.ReadResourceParams{
					URI: tc.uri,
				},
			}
			resp, err := resource.handleGetFirewallResource(context.Background(), req)
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
			var outFirewall godo.Firewall
			require.NoError(t, json.Unmarshal([]byte(content.Text), &outFirewall))
			require.Equal(t, testFirewall.ID, outFirewall.ID)
		})
	}
}
