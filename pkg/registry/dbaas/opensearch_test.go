package dbaas

import (
	"context"
	"mcp-digitalocean/pkg/registry/dbaas/mocks"
	"testing"

	"github.com/digitalocean/godo"
	"github.com/mark3labs/mcp-go/mcp"
	"github.com/stretchr/testify/assert"
	"go.uber.org/mock/gomock"
)

func TestOpenSearchTool_getOpensearchConfig(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()
	mockDB := mocks.NewMockDatabasesService(ctrl)
	val := 12345
	mockDB.EXPECT().GetOpensearchConfig(gomock.Any(), "cid").Return(&godo.OpensearchConfig{HttpMaxContentLengthBytes: &val}, nil, nil)

	client := func(ctx context.Context) (*godo.Client, error) {
		return &godo.Client{
			Databases: mockDB,
		}, nil
	}

	ot := &OpenSearchTool{client: client}
	args := map[string]interface{}{"id": "cid"}
	req := mcp.CallToolRequest{Params: mcp.CallToolParams{Arguments: args}}
	res, err := ot.getOpensearchConfig(context.Background(), req)
	assert.NoError(t, err)
	assert.Contains(t, res.Content[0].(mcp.TextContent).Text, "12345")
	// Error case: missing id (should not expect a call to GetOpensearchConfig)
	reqMissing := mcp.CallToolRequest{Params: mcp.CallToolParams{Arguments: map[string]interface{}{}}}
	res, err = ot.getOpensearchConfig(context.Background(), reqMissing)
	assert.NoError(t, err)
	assert.Contains(t, res.Content[0].(mcp.TextContent).Text, "Cluster id is required")
	// API error
	mockDB.EXPECT().GetOpensearchConfig(gomock.Any(), "badid").Return(nil, nil, assert.AnError)
	args = map[string]interface{}{"id": "badid"}
	req = mcp.CallToolRequest{Params: mcp.CallToolParams{Arguments: args}}
	res, err = ot.getOpensearchConfig(context.Background(), req)
	assert.NoError(t, err)
	assert.Contains(t, res.Content[0].(mcp.TextContent).Text, "api error")
}

func TestOpenSearchTool_updateOpensearchConfig(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()
	mockDB := mocks.NewMockDatabasesService(ctrl)
	val := 54321
	mockDB.EXPECT().UpdateOpensearchConfig(gomock.Any(), "cid", gomock.Any()).Return(&godo.Response{}, nil)

	client := func(ctx context.Context) (*godo.Client, error) {
		return &godo.Client{
			Databases: mockDB,
		}, nil
	}

	ot := &OpenSearchTool{client: client}
	config := map[string]any{"http_max_content_length_bytes": val}
	args := map[string]interface{}{"id": "cid", "config": config}
	req := mcp.CallToolRequest{Params: mcp.CallToolParams{Arguments: args}}
	res, err := ot.updateOpensearchConfig(context.Background(), req)
	assert.NoError(t, err)
	assert.Contains(t, res.Content[0].(mcp.TextContent).Text, "Opensearch config updated successfully")
	// Error case: missing id
	args = map[string]interface{}{"config": config}
	req = mcp.CallToolRequest{Params: mcp.CallToolParams{Arguments: args}}
	res, err = ot.updateOpensearchConfig(context.Background(), req)
	assert.NoError(t, err)
	assert.Contains(t, res.Content[0].(mcp.TextContent).Text, "Cluster id is required")
	// Error case: missing config
	args = map[string]interface{}{"id": "cid"}
	req = mcp.CallToolRequest{Params: mcp.CallToolParams{Arguments: args}}
	res, err = ot.updateOpensearchConfig(context.Background(), req)
	assert.NoError(t, err)
	assert.Contains(t, res.Content[0].(mcp.TextContent).Text, "Missing or invalid 'config' object")
	// Error case: invalid config (not a map)
	args = map[string]interface{}{"id": "cid", "config": "notmap"}
	req = mcp.CallToolRequest{Params: mcp.CallToolParams{Arguments: args}}
	res, err = ot.updateOpensearchConfig(context.Background(), req)
	assert.NoError(t, err)
	assert.Contains(t, res.Content[0].(mcp.TextContent).Text, "Missing or invalid 'config' object")
	// API error
	mockDB.EXPECT().UpdateOpensearchConfig(gomock.Any(), "badid", gomock.Any()).Return(nil, assert.AnError)
	args = map[string]interface{}{"id": "badid", "config": config}
	req = mcp.CallToolRequest{Params: mcp.CallToolParams{Arguments: args}}
	res, err = ot.updateOpensearchConfig(context.Background(), req)
	assert.NoError(t, err)
	assert.Contains(t, res.Content[0].(mcp.TextContent).Text, "api error")
}
