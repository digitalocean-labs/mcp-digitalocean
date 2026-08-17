package droplet

import (
	"context"
	"testing"

	"github.com/digitalocean/godo"
	"github.com/mark3labs/mcp-go/mcp"
	"github.com/stretchr/testify/require"
	"go.uber.org/mock/gomock"
)

func TestDropletTool_createDropletWithUserData(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockDroplets := NewMockDropletsService(ctrl)
	mockActions := NewMockDropletActionsService(ctrl)
	tool := setupDropletToolWithMocks(mockDroplets, mockActions)

	userData := "#cloud-config\nruncmd:\n  - echo ready\n"

	mockDroplets.EXPECT().
		Create(gomock.Any(), &godo.DropletCreateRequest{
			Name:       "cloud-init-droplet",
			Region:     "nyc3",
			Size:       "s-1vcpu-1gb",
			Image:      godo.DropletCreateImage{ID: 456},
			Backups:    false,
			Monitoring: false,
			UserData:   userData,
		}).
		Return(&godo.Droplet{ID: 123, Name: "cloud-init-droplet"}, nil, nil).
		Times(1)

	req := mcp.CallToolRequest{Params: mcp.CallToolParams{Arguments: map[string]any{
		"Name":       "cloud-init-droplet",
		"Size":       "s-1vcpu-1gb",
		"ImageID":    float64(456),
		"Region":     "nyc3",
		"Backup":     false,
		"Monitoring": false,
		"UserData":   userData,
	}}}

	resp, err := tool.createDroplet(context.Background(), req)
	require.NoError(t, err)
	require.NotNil(t, resp)
	require.False(t, resp.IsError)
}
