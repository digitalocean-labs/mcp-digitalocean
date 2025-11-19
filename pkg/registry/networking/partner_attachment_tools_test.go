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

func setupPartnerAttachmentToolWithMock(pa *MockPartnerAttachmentService) *PartnerAttachmentTool {
	client := func(ctx context.Context) (*godo.Client, error) {
		return &godo.Client{PartnerAttachment: pa}, nil
	}

	return NewPartnerAttachmentTool(client)
}

func TestPartnerAttachmentTool_getPartnerAttachment(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	testAttachment := &godo.PartnerAttachment{
		ID:                        "pa-123",
		Name:                      "fast-connect",
		Region:                    "nyc",
		ConnectionBandwidthInMbps: 1000,
	}
	tests := []struct {
		name        string
		id          string
		mockSetup   func(*MockPartnerAttachmentService)
		expectError bool
	}{
		{
			name: "Successful get",
			id:   "pa-123",
			mockSetup: func(m *MockPartnerAttachmentService) {
				m.EXPECT().
					Get(gomock.Any(), "pa-123").
					Return(testAttachment, nil, nil).
					Times(1)
			},
		},
		{
			name: "API error",
			id:   "pa-456",
			mockSetup: func(m *MockPartnerAttachmentService) {
				m.EXPECT().
					Get(gomock.Any(), "pa-456").
					Return(nil, nil, errors.New("api error")).
					Times(1)
			},
			expectError: true,
		},
		{
			name:        "Missing ID argument",
			id:          "",
			mockSetup:   nil,
			expectError: true,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			mockPA := NewMockPartnerAttachmentService(ctrl)
			if tc.mockSetup != nil {
				tc.mockSetup(mockPA)
			}
			tool := setupPartnerAttachmentToolWithMock(mockPA)
			args := map[string]any{}
			if tc.name != "Missing ID argument" {
				args["ID"] = tc.id
			}
			req := mcp.CallToolRequest{Params: mcp.CallToolParams{Arguments: args}}
			resp, err := tool.getPartnerAttachment(context.Background(), req)
			if tc.expectError {
				require.NotNil(t, resp)
				require.True(t, resp.IsError)
				return
			}
			require.NoError(t, err)
			require.NotNil(t, resp)
			require.False(t, resp.IsError)
			var outAttachment godo.PartnerAttachment
			require.NoError(t, json.Unmarshal([]byte(resp.Content[0].(mcp.TextContent).Text), &outAttachment))
			require.Equal(t, testAttachment.ID, outAttachment.ID)
		})
	}
}

func TestPartnerAttachmentTool_listPartnerAttachments(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	testAttachments := []*godo.PartnerAttachment{
		{ID: "pa-1", Name: "pa1"},
		{ID: "pa-2", Name: "pa2"},
	}
	tests := []struct {
		name        string
		page        float64
		perPage     float64
		mockSetup   func(*MockPartnerAttachmentService)
		expectError bool
	}{
		{
			name:    "Successful list",
			page:    2,
			perPage: 1,
			mockSetup: func(m *MockPartnerAttachmentService) {
				m.EXPECT().
					List(gomock.Any(), &godo.ListOptions{Page: 2, PerPage: 1}).
					Return(testAttachments, nil, nil).
					Times(1)
			},
		},
		{
			name:    "API error",
			page:    1,
			perPage: 2,
			mockSetup: func(m *MockPartnerAttachmentService) {
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
			mockSetup: func(m *MockPartnerAttachmentService) {
				m.EXPECT().
					List(gomock.Any(), &godo.ListOptions{Page: 1, PerPage: 20}).
					Return(testAttachments, nil, nil).
					Times(1)
			},
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			mockPA := NewMockPartnerAttachmentService(ctrl)
			if tc.mockSetup != nil {
				tc.mockSetup(mockPA)
			}
			tool := setupPartnerAttachmentToolWithMock(mockPA)
			args := map[string]any{}
			if tc.page != 0 {
				args["Page"] = tc.page
			}
			if tc.perPage != 0 {
				args["PerPage"] = tc.perPage
			}
			req := mcp.CallToolRequest{Params: mcp.CallToolParams{Arguments: args}}
			resp, err := tool.listPartnerAttachments(context.Background(), req)
			if tc.expectError {
				require.NotNil(t, resp)
				require.True(t, resp.IsError)
				return
			}
			require.NoError(t, err)
			require.NotNil(t, resp)
			require.False(t, resp.IsError)
			require.NotEmpty(t, resp.Content)
			var outAttachments []godo.PartnerAttachment
			require.NoError(t, json.Unmarshal([]byte(resp.Content[0].(mcp.TextContent).Text), &outAttachments))
			require.GreaterOrEqual(t, len(outAttachments), 1)
		})
	}
}

func TestPartnerAttachmentTool_createPartnerAttachment(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	testAttachment := &godo.PartnerAttachment{
		ID:                        "pa-123",
		Name:                      "fast-connect",
		Region:                    "nyc",
		ConnectionBandwidthInMbps: 1000,
	}
	tests := []struct {
		name        string
		args        map[string]any
		mockSetup   func(*MockPartnerAttachmentService)
		expectError bool
	}{
		{
			name: "Successful create",
			args: map[string]any{
				"Name":      "fast-connect",
				"Region":    "nyc3",
				"Bandwidth": float64(1000),
			},
			mockSetup: func(m *MockPartnerAttachmentService) {
				m.EXPECT().
					Create(gomock.Any(), &godo.PartnerAttachmentCreateRequest{
						Name:                      "fast-connect",
						Region:                    "nyc3",
						ConnectionBandwidthInMbps: 1000,
					}).
					Return(testAttachment, nil, nil).
					Times(1)
			},
		},
		{
			name: "API error",
			args: map[string]any{
				"Name":      "fail-connect",
				"Region":    "sfo2",
				"Bandwidth": float64(500),
			},
			mockSetup: func(m *MockPartnerAttachmentService) {
				m.EXPECT().
					Create(gomock.Any(), &godo.PartnerAttachmentCreateRequest{
						Name:                      "fail-connect",
						Region:                    "sfo2",
						ConnectionBandwidthInMbps: 500,
					}).
					Return(nil, nil, errors.New("api error")).
					Times(1)
			},
			expectError: true,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			mockPA := NewMockPartnerAttachmentService(ctrl)
			if tc.mockSetup != nil {
				tc.mockSetup(mockPA)
			}
			tool := setupPartnerAttachmentToolWithMock(mockPA)
			req := mcp.CallToolRequest{Params: mcp.CallToolParams{Arguments: tc.args}}
			resp, err := tool.createPartnerAttachment(context.Background(), req)
			if tc.expectError {
				require.NotNil(t, resp)
				require.True(t, resp.IsError)
				return
			}
			require.NoError(t, err)
			require.NotNil(t, resp)
			require.False(t, resp.IsError)
			var outAttachment godo.PartnerAttachment
			require.NoError(t, json.Unmarshal([]byte(resp.Content[0].(mcp.TextContent).Text), &outAttachment))
			require.Equal(t, testAttachment.ID, outAttachment.ID)
		})
	}
}

func TestPartnerAttachmentTool_deletePartnerAttachment(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	tests := []struct {
		name        string
		args        map[string]any
		mockSetup   func(*MockPartnerAttachmentService)
		expectError bool
		expectText  string
	}{
		{
			name: "Successful delete",
			args: map[string]any{"ID": "pa-123"},
			mockSetup: func(m *MockPartnerAttachmentService) {
				m.EXPECT().
					Delete(gomock.Any(), "pa-123").
					Return(&godo.Response{}, nil).
					Times(1)
			},
			expectText: "Partner attachment deleted successfully",
		},
		{
			name: "API error",
			args: map[string]any{"ID": "pa-456"},
			mockSetup: func(m *MockPartnerAttachmentService) {
				m.EXPECT().
					Delete(gomock.Any(), "pa-456").
					Return(nil, errors.New("api error")).
					Times(1)
			},
			expectError: true,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			mockPA := NewMockPartnerAttachmentService(ctrl)
			if tc.mockSetup != nil {
				tc.mockSetup(mockPA)
			}
			tool := setupPartnerAttachmentToolWithMock(mockPA)
			req := mcp.CallToolRequest{Params: mcp.CallToolParams{Arguments: tc.args}}
			resp, err := tool.deletePartnerAttachment(context.Background(), req)
			if tc.expectError {
				require.NotNil(t, resp)
				require.True(t, resp.IsError)
				return
			}
			require.NoError(t, err)
			require.NotNil(t, resp)
			require.False(t, resp.IsError)
			require.Contains(t, resp.Content[0].(mcp.TextContent).Text, tc.expectText)
		})
	}
}

func TestPartnerAttachmentTool_getServiceKey(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	testServiceKey := &godo.ServiceKey{
		Value: "sk-123",
		State: "active",
	}
	tests := []struct {
		name        string
		args        map[string]any
		mockSetup   func(*MockPartnerAttachmentService)
		expectError bool
	}{
		{
			name: "Successful get service key",
			args: map[string]any{"ID": "pa-123"},
			mockSetup: func(m *MockPartnerAttachmentService) {
				m.EXPECT().
					GetServiceKey(gomock.Any(), "pa-123").
					Return(testServiceKey, nil, nil).
					Times(1)
			},
		},
		{
			name: "API error",
			args: map[string]any{"ID": "pa-456"},
			mockSetup: func(m *MockPartnerAttachmentService) {
				m.EXPECT().
					GetServiceKey(gomock.Any(), "pa-456").
					Return(nil, nil, errors.New("api error")).
					Times(1)
			},
			expectError: true,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			mockPA := NewMockPartnerAttachmentService(ctrl)
			if tc.mockSetup != nil {
				tc.mockSetup(mockPA)
			}
			tool := setupPartnerAttachmentToolWithMock(mockPA)
			req := mcp.CallToolRequest{Params: mcp.CallToolParams{Arguments: tc.args}}
			resp, err := tool.getServiceKey(context.Background(), req)
			if tc.expectError {
				require.NotNil(t, resp)
				require.True(t, resp.IsError)
				return
			}
			require.NoError(t, err)
			require.NotNil(t, resp)
			require.False(t, resp.IsError)
			var outKey godo.ServiceKey
			require.NoError(t, json.Unmarshal([]byte(resp.Content[0].(mcp.TextContent).Text), &outKey))
			require.Equal(t, testServiceKey.Value, outKey.Value)
		})
	}
}

func TestPartnerAttachmentTool_getBGPConfig(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	testBGP := &godo.BgpAuthKey{
		Value: "bgp-123",
	}
	tests := []struct {
		name        string
		args        map[string]any
		mockSetup   func(*MockPartnerAttachmentService)
		expectError bool
	}{
		{
			name: "Successful get BGP config",
			args: map[string]any{"ID": "pa-123"},
			mockSetup: func(m *MockPartnerAttachmentService) {
				m.EXPECT().
					GetBGPAuthKey(gomock.Any(), "pa-123").
					Return(testBGP, nil, nil).
					Times(1)
			},
		},
		{
			name: "API error",
			args: map[string]any{"ID": "pa-456"},
			mockSetup: func(m *MockPartnerAttachmentService) {
				m.EXPECT().
					GetBGPAuthKey(gomock.Any(), "pa-456").
					Return(nil, nil, errors.New("api error")).
					Times(1)
			},
			expectError: true,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			mockPA := NewMockPartnerAttachmentService(ctrl)
			if tc.mockSetup != nil {
				tc.mockSetup(mockPA)
			}
			tool := setupPartnerAttachmentToolWithMock(mockPA)
			req := mcp.CallToolRequest{Params: mcp.CallToolParams{Arguments: tc.args}}
			resp, err := tool.getBGPConfig(context.Background(), req)
			if tc.expectError {
				require.NotNil(t, resp)
				require.True(t, resp.IsError)
				return
			}
			require.NoError(t, err)
			require.NotNil(t, resp)
			require.False(t, resp.IsError)
			var outBGP godo.BgpAuthKey
			require.NoError(t, json.Unmarshal([]byte(resp.Content[0].(mcp.TextContent).Text), &outBGP))
			require.Equal(t, testBGP.Value, outBGP.Value)
		})
	}
}

func TestPartnerAttachmentTool_updatePartnerAttachment(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	testAttachment := &godo.PartnerAttachment{
		ID:   "pa-789",
		Name: "updated-connect",
	}
	tests := []struct {
		name        string
		args        map[string]any
		mockSetup   func(*MockPartnerAttachmentService)
		expectError bool
	}{
		{
			name: "Successful update",
			args: map[string]any{
				"ID":     "pa-789",
				"Name":   "updated-connect",
				"VPCIDs": []any{"vpc-1", "vpc-2"},
			},
			mockSetup: func(m *MockPartnerAttachmentService) {
				m.EXPECT().
					Update(gomock.Any(), "pa-789", &godo.PartnerAttachmentUpdateRequest{
						Name:   "updated-connect",
						VPCIDs: []string{"vpc-1", "vpc-2"},
					}).
					Return(testAttachment, nil, nil).
					Times(1)
			},
		},
		{
			name: "API error",
			args: map[string]any{
				"ID":     "pa-999",
				"Name":   "fail-update",
				"VPCIDs": []any{"vpc-x"},
			},
			mockSetup: func(m *MockPartnerAttachmentService) {
				m.EXPECT().
					Update(gomock.Any(), "pa-999", &godo.PartnerAttachmentUpdateRequest{
						Name:   "fail-update",
						VPCIDs: []string{"vpc-x"},
					}).
					Return(nil, nil, errors.New("api error")).
					Times(1)
			},
			expectError: true,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			mockPA := NewMockPartnerAttachmentService(ctrl)
			if tc.mockSetup != nil {
				tc.mockSetup(mockPA)
			}
			tool := setupPartnerAttachmentToolWithMock(mockPA)
			req := mcp.CallToolRequest{Params: mcp.CallToolParams{Arguments: tc.args}}
			resp, err := tool.updatePartnerAttachment(context.Background(), req)
			if tc.expectError {
				require.NotNil(t, resp)
				require.True(t, resp.IsError)
				return
			}
			require.NoError(t, err)
			require.NotNil(t, resp)
			require.False(t, resp.IsError)
			var outAttachment godo.PartnerAttachment
			require.NoError(t, json.Unmarshal([]byte(resp.Content[0].(mcp.TextContent).Text), &outAttachment))
			require.Equal(t, testAttachment.ID, outAttachment.ID)
		})
	}
}
