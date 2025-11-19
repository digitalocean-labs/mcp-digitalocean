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

func setupImagesToolWithMock(images *MockImagesService) *ImagesTool {
	client := func(ctx context.Context) (*godo.Client, error) {
		return &godo.Client{Images: images}, nil
	}

	return NewImagesTool(client)
}

func TestImagesTool_listImages(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	testImages := []godo.Image{
		{ID: 1, Name: "Ubuntu 22.04", Type: "distribution"},
		{ID: 2, Name: "Debian 11", Type: "distribution"},
	}
	tests := []struct {
		name        string
		page        float64
		perPage     float64
		mockSetup   func(*MockImagesService)
		expectError bool
	}{
		{
			name:    "Successful list (custom pagination)",
			page:    2,
			perPage: 10,
			mockSetup: func(m *MockImagesService) {
				m.EXPECT().
					ListDistribution(gomock.Any(), &godo.ListOptions{Page: 2, PerPage: 10}).
					Return(testImages, &godo.Response{}, nil).
					Times(1)
			},
		},
		{
			name:    "Successful list (default pagination)",
			page:    0,
			perPage: 0,
			mockSetup: func(m *MockImagesService) {
				m.EXPECT().
					ListDistribution(gomock.Any(), &godo.ListOptions{Page: 1, PerPage: 50}).
					Return(testImages, &godo.Response{}, nil).
					Times(1)
			},
		},
		{
			name:    "API error",
			page:    1,
			perPage: 5,
			mockSetup: func(m *MockImagesService) {
				m.EXPECT().
					ListDistribution(gomock.Any(), &godo.ListOptions{Page: 1, PerPage: 5}).
					Return(nil, nil, errors.New("api error")).
					Times(1)
			},
			expectError: true,
		},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			mockImages := NewMockImagesService(ctrl)
			if tc.mockSetup != nil {
				tc.mockSetup(mockImages)
			}
			tool := setupImagesToolWithMock(mockImages)
			args := map[string]any{}
			if tc.page != 0 {
				args["Page"] = tc.page
			}
			if tc.perPage != 0 {
				args["PerPage"] = tc.perPage
			}
			req := mcp.CallToolRequest{Params: mcp.CallToolParams{Arguments: args}}
			resp, err := tool.listImages(context.Background(), req)
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
			var outImages []map[string]any
			require.NoError(t, json.Unmarshal([]byte(content), &outImages))
			require.Len(t, outImages, len(testImages))
		})
	}
}

func TestImagesTool_getImageByID(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	testImage := &godo.Image{ID: 42, Name: "Fedora 38", Type: "distribution"}
	tests := []struct {
		name        string
		id          float64
		mockSetup   func(*MockImagesService)
		expectError bool
	}{
		{
			name: "Successful get by ID",
			id:   42,
			mockSetup: func(m *MockImagesService) {
				m.EXPECT().
					GetByID(gomock.Any(), 42).
					Return(testImage, &godo.Response{}, nil).
					Times(1)
			},
		},
		{
			name: "API error",
			id:   99,
			mockSetup: func(m *MockImagesService) {
				m.EXPECT().
					GetByID(gomock.Any(), 99).
					Return(nil, nil, errors.New("api error")).
					Times(1)
			},
			expectError: true,
		},
		{
			name:        "Missing ID argument",
			id:          0,
			mockSetup:   nil,
			expectError: true,
		},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			mockImages := NewMockImagesService(ctrl)
			if tc.mockSetup != nil {
				tc.mockSetup(mockImages)
			}
			tool := setupImagesToolWithMock(mockImages)
			args := map[string]any{}
			if tc.name != "Missing ID argument" {
				args["ID"] = tc.id
			}
			req := mcp.CallToolRequest{Params: mcp.CallToolParams{Arguments: args}}
			resp, err := tool.getImageByID(context.Background(), req)
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
			var outImage godo.Image
			require.NoError(t, json.Unmarshal([]byte(content), &outImage))
			require.Equal(t, int(tc.id), outImage.ID)
		})
	}
}

func TestImagesTool_deleteImage(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	tests := []struct {
		name        string
		imageID     float64
		mockSetup   func(*MockImagesService)
		expectError bool
	}{
		{
			name:    "Successful delete",
			imageID: 123,
			mockSetup: func(m *MockImagesService) {
				m.EXPECT().
					Delete(gomock.Any(), 123).
					Return(&godo.Response{}, nil).
					Times(1)
			},
		},
		{
			name:    "API error",
			imageID: 456,
			mockSetup: func(m *MockImagesService) {
				m.EXPECT().
					Delete(gomock.Any(), 456).
					Return(nil, errors.New("api error")).
					Times(1)
			},
			expectError: true,
		},
		{
			name:        "Missing ImageID argument",
			imageID:     0,
			mockSetup:   nil,
			expectError: true,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			mockImages := NewMockImagesService(ctrl)
			if tc.mockSetup != nil {
				tc.mockSetup(mockImages)
			}
			tool := setupImagesToolWithMock(mockImages)
			args := map[string]any{}
			if tc.imageID != 0 {
				args["ImageID"] = tc.imageID
			}
			req := mcp.CallToolRequest{Params: mcp.CallToolParams{Arguments: args}}
			resp, err := tool.deleteImage(context.Background(), req)
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
			require.Contains(t, content, "Image deleted successfully")
		})
	}
}
