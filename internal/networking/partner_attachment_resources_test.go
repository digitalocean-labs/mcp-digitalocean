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

func setupPartnerAttachmentResourceWithMock(pa *MockPartnerAttachmentService) *PartnerAttachmentMCPResource {
	client := &godo.Client{}
	client.PartnerAttachment = pa
	return NewPartnerAttachmentMCPResource(client)
}

func TestPartnerAttachmentMCPResource_handleGetPartnerAttachmentResource(t *testing.T) {
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
		uri         string
		mockSetup   func(*MockPartnerAttachmentService)
		expectError bool
	}{
		{
			name: "Successful get",
			uri:  "partner_attachment://pa-123",
			mockSetup: func(m *MockPartnerAttachmentService) {
				m.EXPECT().
					Get(gomock.Any(), "pa-123").
					Return(testAttachment, nil, nil).
					Times(1)
			},
		},
		{
			name: "API error",
			uri:  "partner_attachment://pa-456",
			mockSetup: func(m *MockPartnerAttachmentService) {
				m.EXPECT().
					Get(gomock.Any(), "pa-456").
					Return(nil, nil, errors.New("api error")).
					Times(1)
			},
			expectError: true,
		},
		{
			name:        "Invalid URI",
			uri:         "partner_attachment-pa-789",
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
			resource := setupPartnerAttachmentResourceWithMock(mockPA)
			req := mcp.ReadResourceRequest{
				Params: mcp.ReadResourceParams{
					URI: tc.uri,
				},
			}
			resp, err := resource.handleGetPartnerAttachmentResource(context.Background(), req)
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
			var outAttachment godo.PartnerAttachment
			require.NoError(t, json.Unmarshal([]byte(content.Text), &outAttachment))
			require.Equal(t, testAttachment.ID, outAttachment.ID)
		})
	}
}
