package account

import (
	"context"
	"errors"
	"testing"

	"github.com/digitalocean/godo"
	"github.com/mark3labs/mcp-go/mcp"
	"github.com/stretchr/testify/require"
	"go.uber.org/mock/gomock"
)

func setupKeysToolWithMock(mockKeys *MockKeysService) *KeysTool {
	client := func(ctx context.Context) (*godo.Client, error) {
		return &godo.Client{
			Keys: mockKeys,
		}, nil
	}

	return &KeysTool{client: client}
}

func TestKeysTool_createKey(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	testKey := &godo.Key{
		ID:        123,
		Name:      "test-key",
		PublicKey: "ssh-rsa AAAAB3NzaC1yc2EAAAADAQABAAABAQC...",
	}
	tests := []struct {
		name        string
		args        map[string]any
		mockSetup   func(*MockKeysService)
		expectError bool
	}{
		{
			name: "Successful create",
			args: map[string]any{
				"Name":      "test-key",
				"PublicKey": "ssh-rsa AAAAB3NzaC1yc2EAAAADAQABAAABAQC...",
			},
			mockSetup: func(m *MockKeysService) {
				m.EXPECT().
					Create(gomock.Any(), &godo.KeyCreateRequest{
						Name:      "test-key",
						PublicKey: "ssh-rsa AAAAB3NzaC1yc2EAAAADAQABAAABAQC...",
					}).
					Return(testKey, nil, nil).
					Times(1)
			},
		},
		{
			name: "API error",
			args: map[string]any{
				"Name":      "fail-key",
				"PublicKey": "ssh-rsa BADKEY",
			},
			mockSetup: func(m *MockKeysService) {
				m.EXPECT().
					Create(gomock.Any(), &godo.KeyCreateRequest{
						Name:      "fail-key",
						PublicKey: "ssh-rsa BADKEY",
					}).
					Return(nil, nil, errors.New("api error")).
					Times(1)
			},
			expectError: true,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			mockKeys := NewMockKeysService(ctrl)
			if tc.mockSetup != nil {
				tc.mockSetup(mockKeys)
			}
			tool := setupKeysToolWithMock(mockKeys)
			req := mcp.CallToolRequest{Params: mcp.CallToolParams{Arguments: tc.args}}
			resp, err := tool.createKey(context.Background(), req)
			if tc.expectError {
				require.NotNil(t, resp)
				require.True(t, resp.IsError)
				return
			}
			require.NoError(t, err)
			require.NotNil(t, resp)
			require.False(t, resp.IsError)
		})
	}
}

func TestKeysTool_deleteKey(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	tests := []struct {
		name        string
		args        map[string]any
		mockSetup   func(*MockKeysService)
		expectError bool
		expectText  string
	}{
		{
			name: "Successful delete",
			args: map[string]any{"ID": float64(123)},
			mockSetup: func(m *MockKeysService) {
				m.EXPECT().
					DeleteByID(gomock.Any(), 123).
					Return(nil, nil).
					Times(1)
			},
			expectText: "SSH key deleted successfully",
		},
		{
			name: "API error",
			args: map[string]any{"ID": float64(456)},
			mockSetup: func(m *MockKeysService) {
				m.EXPECT().
					DeleteByID(gomock.Any(), 456).
					Return(nil, errors.New("api error")).
					Times(1)
			},
			expectError: true,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			mockKeys := NewMockKeysService(ctrl)
			if tc.mockSetup != nil {
				tc.mockSetup(mockKeys)
			}
			tool := setupKeysToolWithMock(mockKeys)
			req := mcp.CallToolRequest{Params: mcp.CallToolParams{Arguments: tc.args}}
			resp, err := tool.deleteKey(context.Background(), req)
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

func TestKeysTool_getKey(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	testKey := &godo.Key{
		ID:        123,
		Name:      "test-key",
		PublicKey: "ssh-rsa AAAAB3NzaC1yc2EAAAADAQABAAABAQC...",
	}
	tests := []struct {
		name        string
		id          float64
		mockSetup   func(*MockKeysService)
		expectError bool
	}{
		{
			name: "Successful get",
			id:   123,
			mockSetup: func(m *MockKeysService) {
				m.EXPECT().
					GetByID(gomock.Any(), 123).
					Return(testKey, nil, nil).
					Times(1)
			},
		},
		{
			name: "API error",
			id:   456,
			mockSetup: func(m *MockKeysService) {
				m.EXPECT().
					GetByID(gomock.Any(), 456).
					Return(nil, nil, errors.New("api error")).
					Times(1)
			},
			expectError: true,
		},
		{
			name: "Missing ID argument",
			id:   0,
			mockSetup: func(m *MockKeysService) {
				// No call expected
			},
			expectError: true,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			mockKeys := NewMockKeysService(ctrl)
			if tc.mockSetup != nil {
				tc.mockSetup(mockKeys)
			}
			tool := setupKeysToolWithMock(mockKeys)
			args := map[string]any{}
			if tc.name != "Missing ID argument" {
				args["ID"] = tc.id
			}
			req := mcp.CallToolRequest{Params: mcp.CallToolParams{Arguments: args}}
			resp, err := tool.getKey(context.Background(), req)
			if tc.expectError {
				require.NotNil(t, resp)
				require.True(t, resp.IsError)
				return
			}
			require.NoError(t, err)
			require.NotNil(t, resp)
			require.False(t, resp.IsError)
			require.NotEmpty(t, resp.Content)
		})
	}
}

func TestKeysTool_listKeys(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	testKeys := []godo.Key{
		{ID: 1, Name: "key1"},
		{ID: 2, Name: "key2"},
	}
	tests := []struct {
		name        string
		page        float64
		perPage     float64
		mockSetup   func(*MockKeysService)
		expectError bool
	}{
		{
			name:    "Successful list",
			page:    1,
			perPage: 2,
			mockSetup: func(m *MockKeysService) {
				m.EXPECT().
					List(gomock.Any(), &godo.ListOptions{Page: 1, PerPage: 2}).
					Return(testKeys, nil, nil).
					Times(1)
			},
		},
		{
			name:    "API error",
			page:    1,
			perPage: 2,
			mockSetup: func(m *MockKeysService) {
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
			mockSetup: func(m *MockKeysService) {
				m.EXPECT().
					List(gomock.Any(), &godo.ListOptions{Page: 1, PerPage: 30}).
					Return(testKeys, nil, nil).
					Times(1)
			},
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			mockKeys := NewMockKeysService(ctrl)
			if tc.mockSetup != nil {
				tc.mockSetup(mockKeys)
			}
			tool := setupKeysToolWithMock(mockKeys)
			args := map[string]any{}
			if tc.page != 0 {
				args["Page"] = tc.page
			}
			if tc.perPage != 0 {
				args["PerPage"] = tc.perPage
			}
			req := mcp.CallToolRequest{Params: mcp.CallToolParams{Arguments: args}}
			resp, err := tool.listKeys(context.Background(), req)
			if tc.expectError {
				require.NotNil(t, resp)
				require.True(t, resp.IsError)
				return
			}
			require.NoError(t, err)
			require.NotNil(t, resp)
			require.False(t, resp.IsError)
			require.NotEmpty(t, resp.Content)
		})
	}
}
