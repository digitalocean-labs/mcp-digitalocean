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

func setupImagesResourceWithMock(images *MockImagesService) *ImagesMCPResource {
	client := &godo.Client{}
	client.Images = images
	return NewImagesMCPResource(client)
}

func TestImagesMCPResource_handleGetImageResource(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	testImages := []godo.Image{
		{ID: 1, Name: "Ubuntu 22.04", Type: "distribution"},
		{ID: 2, Name: "Debian 11", Type: "distribution"},
	}
	tests := []struct {
		name        string
		mockSetup   func(*MockImagesService)
		expectError bool
	}{
		{
			name: "Successful list",
			mockSetup: func(m *MockImagesService) {
				m.EXPECT().
					ListDistribution(gomock.Any(), gomock.Any()).
					Return(testImages, &godo.Response{}, nil).
					Times(1)
			},
		},
		{
			name: "API error",
			mockSetup: func(m *MockImagesService) {
				m.EXPECT().
					ListDistribution(gomock.Any(), gomock.Any()).
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
			resource := setupImagesResourceWithMock(mockImages)
			req := mcp.ReadResourceRequest{
				Params: mcp.ReadResourceParams{
					URI: ImagesURI + "distribution",
				},
			}
			resp, err := resource.handleGetImageResource(context.Background(), req)
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
			require.Equal(t, "application/json", content.MIMEType)
			var outImages []godo.Image
			require.NoError(t, json.Unmarshal([]byte(content.Text), &outImages))
		})
	}
}

func TestImagesMCPResource_handleGetImageResourceTemplate(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	testImage := &godo.Image{ID: 42, Name: "Fedora 38", Type: "distribution"}
	tests := []struct {
		name        string
		uri         string
		mockSetup   func(*MockImagesService)
		expectError bool
	}{
		{
			name: "Successful get by ID",
			uri:  ImagesURI + "42",
			mockSetup: func(m *MockImagesService) {
				m.EXPECT().
					GetByID(gomock.Any(), 42).
					Return(testImage, &godo.Response{}, nil).
					Times(1)
			},
		},
		{
			name: "API error",
			uri:  ImagesURI + "99",
			mockSetup: func(m *MockImagesService) {
				m.EXPECT().
					GetByID(gomock.Any(), 99).
					Return(nil, nil, errors.New("api error")).
					Times(1)
			},
			expectError: true,
		},
		{
			name:        "Invalid URI",
			uri:         ImagesURI + "badid",
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
			resource := setupImagesResourceWithMock(mockImages)
			req := mcp.ReadResourceRequest{
				Params: mcp.ReadResourceParams{
					URI: tc.uri,
				},
			}
			resp, err := resource.handleGetImageResourceTemplate(context.Background(), req)
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
			var outImage godo.Image
			require.NoError(t, json.Unmarshal([]byte(content.Text), &outImage))
			require.Equal(t, testImage.ID, outImage.ID)
		})
	}
}
