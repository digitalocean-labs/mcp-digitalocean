package vectordb

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

func setupVectorDBToolWithMocks(vectorSvc *MockVectorDBsService) *VectorDBTool {
	client := func(ctx context.Context) (*godo.Client, error) {
		return &godo.Client{VectorDBs: vectorSvc}, nil
	}
	return NewVectorDBTool(client)
}

func TestVectorDBTool_createVectorDB(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	testDB := &godo.VectorDB{
		ID:     "vdb-abc",
		Name:   "my-weaviate",
		Region: "tor1",
		Size:   "small",
		Status: "creating",
		Endpoints: &godo.VectorDBEndpoints{
			HTTP: "https://vdb-abc.weaviate.tor1.do.com",
			GRPC: "grpc://vdb-abc.weaviate.tor1.do.com:50051",
		},
	}

	tests := []struct {
		name        string
		args        map[string]any
		mockSetup   func(*MockVectorDBsService)
		expectError bool
	}{
		{
			name: "Successful create",
			args: map[string]any{
				"Name":      "my-weaviate",
				"Region":    "tor1",
				"Size":      "small",
				"ProjectId": "proj-1",
			},
			mockSetup: func(m *MockVectorDBsService) {
				m.EXPECT().
					Create(gomock.Any(), &godo.VectorDBCreateRequest{
						Name:      "my-weaviate",
						Region:    "tor1",
						Size:      "small",
						ProjectID: "proj-1",
					}).
					Return(testDB, nil, nil).
					Times(1)
			},
		},
		{
			name: "Successful create with tags",
			args: map[string]any{
				"Name":      "my-weaviate",
				"Region":    "tor1",
				"Size":      "medium",
				"ProjectId": "proj-1",
				"Tags":      []any{"prod", "ai"},
			},
			mockSetup: func(m *MockVectorDBsService) {
				m.EXPECT().
					Create(gomock.Any(), &godo.VectorDBCreateRequest{
						Name:      "my-weaviate",
						Region:    "tor1",
						Size:      "medium",
						ProjectID: "proj-1",
						Tags:      []string{"prod", "ai"},
					}).
					Return(testDB, nil, nil).
					Times(1)
			},
		},
		{
			name: "Missing Name",
			args: map[string]any{
				"Region":    "tor1",
				"Size":      "small",
				"ProjectId": "proj-1",
			},
			expectError: true,
		},
		{
			name: "Missing Region",
			args: map[string]any{
				"Name":      "my-weaviate",
				"Size":      "small",
				"ProjectId": "proj-1",
			},
			expectError: true,
		},
		{
			name: "Missing Size",
			args: map[string]any{
				"Name":      "my-weaviate",
				"Region":    "tor1",
				"ProjectId": "proj-1",
			},
			expectError: true,
		},
		{
			name: "Missing ProjectId",
			args: map[string]any{
				"Name":   "my-weaviate",
				"Region": "tor1",
				"Size":   "small",
			},
			expectError: true,
		},
		{
			name: "API error",
			args: map[string]any{
				"Name":      "my-weaviate",
				"Region":    "tor1",
				"Size":      "small",
				"ProjectId": "proj-1",
			},
			mockSetup: func(m *MockVectorDBsService) {
				m.EXPECT().
					Create(gomock.Any(), &godo.VectorDBCreateRequest{
						Name:      "my-weaviate",
						Region:    "tor1",
						Size:      "small",
						ProjectID: "proj-1",
					}).
					Return(nil, nil, errors.New("api error")).
					Times(1)
			},
			expectError: true,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			mockSvc := NewMockVectorDBsService(ctrl)
			if tc.mockSetup != nil {
				tc.mockSetup(mockSvc)
			}
			tool := setupVectorDBToolWithMocks(mockSvc)
			req := mcp.CallToolRequest{Params: mcp.CallToolParams{Arguments: tc.args}}
			resp, err := tool.createVectorDB(context.Background(), req)

			if tc.expectError {
				require.NotNil(t, resp)
				require.True(t, resp.IsError)
				return
			}

			require.NoError(t, err)
			require.NotNil(t, resp)
			require.False(t, resp.IsError)

			var out godo.VectorDB
			require.NoError(t, json.Unmarshal([]byte(resp.Content[0].(mcp.TextContent).Text), &out))
			require.Equal(t, testDB.ID, out.ID)
			require.Equal(t, testDB.Name, out.Name)
			require.NotNil(t, out.Endpoints)
			require.Equal(t, testDB.Endpoints.HTTP, out.Endpoints.HTTP)
		})
	}
}

func TestVectorDBTool_listVectorDBs(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	dbs := []godo.VectorDB{
		{
			ID:     "vdb-1",
			Name:   "db-a",
			Region: "tor1",
			Size:   "small",
			Status: "active",
			Endpoints: &godo.VectorDBEndpoints{
				HTTP: "https://vdb-1.weaviate.tor1.do.com",
				GRPC: "grpc://vdb-1.weaviate.tor1.do.com:50051",
			},
		},
		{ID: "vdb-2", Name: "db-b", Region: "tor1", Size: "medium", Status: "active"},
	}

	tests := []struct {
		name        string
		args        map[string]any
		mockSetup   func(*MockVectorDBsService)
		expectError bool
		wantIDs     []string
	}{
		{
			name: "Successful list with defaults",
			args: map[string]any{},
			mockSetup: func(m *MockVectorDBsService) {
				m.EXPECT().
					List(gomock.Any(), &godo.ListOptions{Page: 1, PerPage: 20}).
					Return(dbs, nil, nil).
					Times(1)
			},
			wantIDs: []string{"vdb-1", "vdb-2"},
		},
		{
			name: "Successful list with pagination",
			args: map[string]any{
				"Page":    float64(2),
				"PerPage": float64(5),
			},
			mockSetup: func(m *MockVectorDBsService) {
				m.EXPECT().
					List(gomock.Any(), &godo.ListOptions{Page: 2, PerPage: 5}).
					Return(dbs, nil, nil).
					Times(1)
			},
			wantIDs: []string{"vdb-1", "vdb-2"},
		},
		{
			name: "PerPage capped at max",
			args: map[string]any{
				"PerPage": float64(999),
			},
			mockSetup: func(m *MockVectorDBsService) {
				m.EXPECT().
					List(gomock.Any(), &godo.ListOptions{Page: 1, PerPage: 50}).
					Return([]godo.VectorDB{}, nil, nil).
					Times(1)
			},
			wantIDs: nil,
		},
		{
			name: "API error",
			args: map[string]any{},
			mockSetup: func(m *MockVectorDBsService) {
				m.EXPECT().
					List(gomock.Any(), &godo.ListOptions{Page: 1, PerPage: 20}).
					Return(nil, nil, errors.New("api error")).
					Times(1)
			},
			expectError: true,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			mockSvc := NewMockVectorDBsService(ctrl)
			if tc.mockSetup != nil {
				tc.mockSetup(mockSvc)
			}
			tool := setupVectorDBToolWithMocks(mockSvc)
			req := mcp.CallToolRequest{Params: mcp.CallToolParams{Arguments: tc.args}}
			resp, err := tool.listVectorDBs(context.Background(), req)

			if tc.expectError {
				require.NotNil(t, resp)
				require.True(t, resp.IsError)
				return
			}

			require.NoError(t, err)
			require.NotNil(t, resp)
			require.False(t, resp.IsError)

			var out []map[string]any
			require.NoError(t, json.Unmarshal([]byte(resp.Content[0].(mcp.TextContent).Text), &out))
			if len(tc.wantIDs) == 0 {
				require.Empty(t, out)
				return
			}
			require.Len(t, out, len(tc.wantIDs))
			for i, id := range tc.wantIDs {
				require.Equal(t, id, out[i]["id"])
			}
			// The list summary must surface the connection endpoints.
			require.Equal(t, "https://vdb-1.weaviate.tor1.do.com", out[0]["endpoint_http"])
			require.Equal(t, "grpc://vdb-1.weaviate.tor1.do.com:50051", out[0]["endpoint_grpc"])
		})
	}
}

func TestVectorDBTool_getVectorDBByID(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	testDB := &godo.VectorDB{
		ID:     "vdb-abc",
		Name:   "my-weaviate",
		Region: "tor1",
		Size:   "small",
		Status: "active",
	}

	tests := []struct {
		name        string
		args        map[string]any
		mockSetup   func(*MockVectorDBsService)
		expectError bool
	}{
		{
			name: "Successful get",
			args: map[string]any{"ID": "vdb-abc"},
			mockSetup: func(m *MockVectorDBsService) {
				m.EXPECT().
					Get(gomock.Any(), "vdb-abc").
					Return(testDB, nil, nil).
					Times(1)
			},
		},
		{
			name:        "Missing ID",
			args:        map[string]any{},
			expectError: true,
		},
		{
			name:        "Empty ID",
			args:        map[string]any{"ID": ""},
			expectError: true,
		},
		{
			name: "API error",
			args: map[string]any{"ID": "vdb-abc"},
			mockSetup: func(m *MockVectorDBsService) {
				m.EXPECT().
					Get(gomock.Any(), "vdb-abc").
					Return(nil, nil, errors.New("api error")).
					Times(1)
			},
			expectError: true,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			mockSvc := NewMockVectorDBsService(ctrl)
			if tc.mockSetup != nil {
				tc.mockSetup(mockSvc)
			}
			tool := setupVectorDBToolWithMocks(mockSvc)
			req := mcp.CallToolRequest{Params: mcp.CallToolParams{Arguments: tc.args}}
			resp, err := tool.getVectorDBByID(context.Background(), req)

			if tc.expectError {
				require.NotNil(t, resp)
				require.True(t, resp.IsError)
				return
			}

			require.NoError(t, err)
			require.NotNil(t, resp)
			require.False(t, resp.IsError)

			var out godo.VectorDB
			require.NoError(t, json.Unmarshal([]byte(resp.Content[0].(mcp.TextContent).Text), &out))
			require.Equal(t, testDB.ID, out.ID)
			require.Equal(t, testDB.Name, out.Name)
			require.Equal(t, testDB.Status, out.Status)
		})
	}
}

func TestVectorDBTool_resizeVectorDB(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	testDB := &godo.VectorDB{
		ID:     "vdb-abc",
		Name:   "my-weaviate",
		Region: "tor1",
		Size:   "large",
		Status: "resizing",
	}

	tests := []struct {
		name        string
		args        map[string]any
		mockSetup   func(*MockVectorDBsService)
		expectError bool
	}{
		{
			name: "Successful resize",
			args: map[string]any{"ID": "vdb-abc", "Size": "large"},
			mockSetup: func(m *MockVectorDBsService) {
				m.EXPECT().
					Resize(gomock.Any(), "vdb-abc", &godo.VectorDBResizeRequest{Size: "large"}).
					Return(testDB, nil, nil).
					Times(1)
			},
		},
		{
			name:        "Missing ID",
			args:        map[string]any{"Size": "large"},
			expectError: true,
		},
		{
			name:        "Missing Size",
			args:        map[string]any{"ID": "vdb-abc"},
			expectError: true,
		},
		{
			name: "API error",
			args: map[string]any{"ID": "vdb-abc", "Size": "large"},
			mockSetup: func(m *MockVectorDBsService) {
				m.EXPECT().
					Resize(gomock.Any(), "vdb-abc", &godo.VectorDBResizeRequest{Size: "large"}).
					Return(nil, nil, errors.New("api error")).
					Times(1)
			},
			expectError: true,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			mockSvc := NewMockVectorDBsService(ctrl)
			if tc.mockSetup != nil {
				tc.mockSetup(mockSvc)
			}
			tool := setupVectorDBToolWithMocks(mockSvc)
			req := mcp.CallToolRequest{Params: mcp.CallToolParams{Arguments: tc.args}}
			resp, err := tool.resizeVectorDB(context.Background(), req)

			if tc.expectError {
				require.NotNil(t, resp)
				require.True(t, resp.IsError)
				return
			}

			require.NoError(t, err)
			require.NotNil(t, resp)
			require.False(t, resp.IsError)

			var out godo.VectorDB
			require.NoError(t, json.Unmarshal([]byte(resp.Content[0].(mcp.TextContent).Text), &out))
			require.Equal(t, testDB.ID, out.ID)
			require.Equal(t, testDB.Size, out.Size)
		})
	}
}

func TestVectorDBTool_deleteVectorDB(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	tests := []struct {
		name        string
		args        map[string]any
		mockSetup   func(*MockVectorDBsService)
		expectError bool
		expectText  string
	}{
		{
			name: "Successful delete",
			args: map[string]any{"ID": "vdb-abc"},
			mockSetup: func(m *MockVectorDBsService) {
				m.EXPECT().
					Delete(gomock.Any(), "vdb-abc").
					Return(&godo.Response{}, nil).
					Times(1)
			},
			expectText: "Vector database deleted successfully",
		},
		{
			name:        "Missing ID",
			args:        map[string]any{},
			expectError: true,
		},
		{
			name:        "Empty ID",
			args:        map[string]any{"ID": ""},
			expectError: true,
		},
		{
			name: "API error",
			args: map[string]any{"ID": "vdb-abc"},
			mockSetup: func(m *MockVectorDBsService) {
				m.EXPECT().
					Delete(gomock.Any(), "vdb-abc").
					Return(nil, errors.New("api error")).
					Times(1)
			},
			expectError: true,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			mockSvc := NewMockVectorDBsService(ctrl)
			if tc.mockSetup != nil {
				tc.mockSetup(mockSvc)
			}
			tool := setupVectorDBToolWithMocks(mockSvc)
			req := mcp.CallToolRequest{Params: mcp.CallToolParams{Arguments: tc.args}}
			resp, err := tool.deleteVectorDB(context.Background(), req)

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

func TestVectorDBTool_getVectorDBCredentials(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	testCreds := &godo.VectorDBAdminCredentials{
		UserID:   "user-1",
		APIToken: "secret-token",
	}

	tests := []struct {
		name        string
		args        map[string]any
		mockSetup   func(*MockVectorDBsService)
		expectError bool
	}{
		{
			name: "Successful get credentials",
			args: map[string]any{"ID": "vdb-abc"},
			mockSetup: func(m *MockVectorDBsService) {
				m.EXPECT().
					GetCredentials(gomock.Any(), "vdb-abc").
					Return(testCreds, nil, nil).
					Times(1)
			},
		},
		{
			name:        "Missing ID",
			args:        map[string]any{},
			expectError: true,
		},
		{
			name:        "Empty ID",
			args:        map[string]any{"ID": ""},
			expectError: true,
		},
		{
			name: "API error",
			args: map[string]any{"ID": "vdb-abc"},
			mockSetup: func(m *MockVectorDBsService) {
				m.EXPECT().
					GetCredentials(gomock.Any(), "vdb-abc").
					Return(nil, nil, errors.New("api error")).
					Times(1)
			},
			expectError: true,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			mockSvc := NewMockVectorDBsService(ctrl)
			if tc.mockSetup != nil {
				tc.mockSetup(mockSvc)
			}
			tool := setupVectorDBToolWithMocks(mockSvc)
			req := mcp.CallToolRequest{Params: mcp.CallToolParams{Arguments: tc.args}}
			resp, err := tool.getVectorDBCredentials(context.Background(), req)

			if tc.expectError {
				require.NotNil(t, resp)
				require.True(t, resp.IsError)
				return
			}

			require.NoError(t, err)
			require.NotNil(t, resp)
			require.False(t, resp.IsError)

			var out godo.VectorDBAdminCredentials
			require.NoError(t, json.Unmarshal([]byte(resp.Content[0].(mcp.TextContent).Text), &out))
			require.Equal(t, testCreds.UserID, out.UserID)
			require.Equal(t, testCreds.APIToken, out.APIToken)
		})
	}
}
