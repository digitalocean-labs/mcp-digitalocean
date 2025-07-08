package droplet

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"testing"

	"github.com/digitalocean/godo"
	"github.com/mark3labs/mcp-go/mcp"
	"github.com/stretchr/testify/require"
	"go.uber.org/mock/gomock"
)

func setupDropletResourceWithMocks(droplets *MockDropletsService, actions *MockDropletActionsService) *DropletMCPResource {
	client := &godo.Client{}
	client.Droplets = droplets
	client.DropletActions = actions
	return NewDropletMCPResource(client)
}

func TestDropletMCPResource_handleGetDropletResource(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	testDroplet := &godo.Droplet{
		ID:   123,
		Name: "test-droplet",
	}

	tests := []struct {
		name        string
		uri         string
		mockSetup   func(*MockDropletsService)
		expectError bool
	}{
		{
			name: "Successful get",
			uri:  "droplets://123",
			mockSetup: func(m *MockDropletsService) {
				m.EXPECT().
					Get(gomock.Any(), 123).
					Return(testDroplet, nil, nil).
					Times(1)
			},
		},
		{
			name: "API error",
			uri:  "droplets://456",
			mockSetup: func(m *MockDropletsService) {
				m.EXPECT().
					Get(gomock.Any(), 456).
					Return(nil, nil, errors.New("api error")).
					Times(1)
			},
			expectError: true,
		},
		{
			name:        "Invalid URI",
			uri:         "droplets123",
			mockSetup:   nil,
			expectError: true,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			mockDroplets := NewMockDropletsService(ctrl)
			mockActions := NewMockDropletActionsService(ctrl)
			if tc.mockSetup != nil {
				tc.mockSetup(mockDroplets)
			}
			resource := setupDropletResourceWithMocks(mockDroplets, mockActions)
			req := mcp.ReadResourceRequest{
				Params: mcp.ReadResourceParams{
					URI: tc.uri,
				},
			}
			resp, err := resource.handleGetDropletResource(context.Background(), req)
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
			var outDroplet godo.Droplet
			require.NoError(t, json.Unmarshal([]byte(content.Text), &outDroplet))
			require.Equal(t, testDroplet.ID, outDroplet.ID)
		})
	}
}

func TestDropletMCPResource_handleGetActionsResource(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	testAction := &godo.Action{
		ID:     789,
		Status: "completed",
	}

	tests := []struct {
		name        string
		uri         string
		mockSetup   func(*MockDropletActionsService)
		expectError bool
	}{
		{
			name: "Successful get action",
			uri:  "droplets://123/actions/789",
			mockSetup: func(m *MockDropletActionsService) {
				m.EXPECT().
					Get(gomock.Any(), 123, 789).
					Return(testAction, nil, nil).
					Times(1)
			},
		},
		{
			name: "API error",
			uri:  "droplets://456/actions/999",
			mockSetup: func(m *MockDropletActionsService) {
				m.EXPECT().
					Get(gomock.Any(), 456, 999).
					Return(nil, nil, errors.New("api error")).
					Times(1)
			},
			expectError: true,
		},
		{
			name:        "Invalid URI",
			uri:         "droplets://baduri",
			mockSetup:   nil,
			expectError: true,
		},
		{
			name:        "Non-numeric IDs",
			uri:         "droplets://abc/actions/def",
			mockSetup:   nil,
			expectError: true,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			mockDroplets := NewMockDropletsService(ctrl)
			mockActions := NewMockDropletActionsService(ctrl)
			if tc.mockSetup != nil {
				tc.mockSetup(mockActions)
			}
			resource := setupDropletResourceWithMocks(mockDroplets, mockActions)
			req := mcp.ReadResourceRequest{
				Params: mcp.ReadResourceParams{
					URI: tc.uri,
				},
			}
			resp, err := resource.handleGetActionsResource(context.Background(), req)
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
			var outAction godo.Action
			require.NoError(t, json.Unmarshal([]byte(content.Text), &outAction))
			require.Equal(t, testAction.ID, outAction.ID)
		})
	}
}

// Additional edge case: test extractDropletAndActionFromURI directly
func Test_extractDropletAndActionFromURI(t *testing.T) {
	tests := []struct {
		uri         string
		wantDroplet int
		wantAction  int
		expectError bool
	}{
		{"123/actions/456", 123, 456, false},
		{"bad/actions/456", 0, 0, true},
		{"123/actions/bad", 0, 0, true},
		{"123/act/456", 0, 0, true},
		{"", 0, 0, true},
	}

	for _, tc := range tests {
		t.Run(fmt.Sprintf("uri=%s", tc.uri), func(t *testing.T) {
			did, aid, err := extractDropletAndActionFromURI(tc.uri)
			if tc.expectError {
				require.Error(t, err)
			} else {
				require.NoError(t, err)
				require.Equal(t, tc.wantDroplet, did)
				require.Equal(t, tc.wantAction, aid)
			}
		})
	}
}
