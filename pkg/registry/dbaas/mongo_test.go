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

func TestMongoTool_getMongoDBConfig(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()
	mockDB := mocks.NewMockDatabasesService(ctrl)
	mockDB.EXPECT().GetMongoDBConfig(gomock.Any(), "cid").Return(&godo.MongoDBConfig{Verbosity: new(int)}, nil, nil)

	client := func(ctx context.Context) (*godo.Client, error) {
		return &godo.Client{
			Databases: mockDB,
		}, nil
	}

	mt := &MongoTool{client: client}
	args := map[string]interface{}{"id": "cid"}
	req := mcp.CallToolRequest{Params: mcp.CallToolParams{Arguments: args}}
	res, err := mt.getMongoDBConfig(context.Background(), req)
	assert.NoError(t, err)
	assert.Contains(t, res.Content[0].(mcp.TextContent).Text, "verbosity")
	// Error case: missing id (should not expect a call to GetMongoDBConfig)
	reqMissing := mcp.CallToolRequest{Params: mcp.CallToolParams{Arguments: map[string]interface{}{}}}
	res, err = mt.getMongoDBConfig(context.Background(), reqMissing)
	assert.NoError(t, err)
	assert.Contains(t, res.Content[0].(mcp.TextContent).Text, "Cluster id is required")
}

func TestMongoTool_updateMongoDBConfig(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()
	mockDB := mocks.NewMockDatabasesService(ctrl)
	mockDB.EXPECT().UpdateMongoDBConfig(gomock.Any(), "cid", gomock.Any()).Return(nil, nil)

	client := func(ctx context.Context) (*godo.Client, error) {
		return &godo.Client{
			Databases: mockDB,
		}, nil
	}

	mt := &MongoTool{client: client}
	cfg := map[string]any{}
	args := map[string]interface{}{"id": "cid", "config": cfg}
	req := mcp.CallToolRequest{Params: mcp.CallToolParams{Arguments: args}}
	res, err := mt.updateMongoDBConfig(context.Background(), req)
	assert.NoError(t, err)
	assert.Contains(t, res.Content[0].(mcp.TextContent).Text, "MongoDB config updated successfully")
	// Error case: missing config
	args = map[string]interface{}{"id": "cid"}
	req = mcp.CallToolRequest{Params: mcp.CallToolParams{Arguments: args}}
	res, err = mt.updateMongoDBConfig(context.Background(), req)
	assert.NoError(t, err)
	assert.Contains(t, res.Content[0].(mcp.TextContent).Text, "Missing or invalid 'config' object")
}
