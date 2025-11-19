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

func TestRedisTool_getRedisConfig(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()
	mockDB := mocks.NewMockDatabasesService(ctrl)
	val := "volatile-lru"
	mockDB.EXPECT().GetRedisConfig(gomock.Any(), "cid").Return(&godo.RedisConfig{RedisMaxmemoryPolicy: &val}, nil, nil)

	client := func(ctx context.Context) (*godo.Client, error) {
		return &godo.Client{
			Databases: mockDB,
		}, nil
	}

	rt := &RedisTool{client: client}
	args := map[string]interface{}{"id": "cid"}
	req := mcp.CallToolRequest{Params: mcp.CallToolParams{Arguments: args}}
	res, err := rt.getRedisConfig(context.Background(), req)
	assert.NoError(t, err)
	assert.Contains(t, res.Content[0].(mcp.TextContent).Text, "volatile-lru")
	// Error case: missing id (should not expect a call to GetRedisConfig)
	reqMissing := mcp.CallToolRequest{Params: mcp.CallToolParams{Arguments: map[string]interface{}{}}}
	res, err = rt.getRedisConfig(context.Background(), reqMissing)
	assert.NoError(t, err)
	assert.Contains(t, res.Content[0].(mcp.TextContent).Text, "Cluster id is required")
	// API error
	mockDB.EXPECT().GetRedisConfig(gomock.Any(), "badid").Return(nil, nil, assert.AnError)
	args = map[string]interface{}{"id": "badid"}
	req = mcp.CallToolRequest{Params: mcp.CallToolParams{Arguments: args}}
	res, err = rt.getRedisConfig(context.Background(), req)
	assert.NoError(t, err)
	assert.Contains(t, res.Content[0].(mcp.TextContent).Text, "api error")
}

func TestRedisTool_updateRedisConfig(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()
	mockDB := mocks.NewMockDatabasesService(ctrl)
	val := "allkeys-lru"
	mockDB.EXPECT().UpdateRedisConfig(gomock.Any(), "cid", gomock.Any()).Return(&godo.Response{}, nil)

	client := func(ctx context.Context) (*godo.Client, error) {
		return &godo.Client{
			Databases: mockDB,
		}, nil
	}

	rt := &RedisTool{client: client}
	config := map[string]any{"redis_maxmemory_policy": val}
	args := map[string]interface{}{"id": "cid", "config": config}
	req := mcp.CallToolRequest{Params: mcp.CallToolParams{Arguments: args}}
	res, err := rt.updateRedisConfig(context.Background(), req)
	assert.NoError(t, err)
	assert.Contains(t, res.Content[0].(mcp.TextContent).Text, "Redis config updated successfully")
	// Error case: missing id
	args = map[string]interface{}{"config": config}
	req = mcp.CallToolRequest{Params: mcp.CallToolParams{Arguments: args}}
	res, err = rt.updateRedisConfig(context.Background(), req)
	assert.NoError(t, err)
	assert.Contains(t, res.Content[0].(mcp.TextContent).Text, "Cluster id is required")
	// Error case: missing config
	args = map[string]interface{}{"id": "cid"}
	req = mcp.CallToolRequest{Params: mcp.CallToolParams{Arguments: args}}
	res, err = rt.updateRedisConfig(context.Background(), req)
	assert.NoError(t, err)
	assert.Contains(t, res.Content[0].(mcp.TextContent).Text, "Missing or invalid 'config' object")
	// Error case: invalid config (not a map)
	args = map[string]interface{}{"id": "cid", "config": "notmap"}
	req = mcp.CallToolRequest{Params: mcp.CallToolParams{Arguments: args}}
	res, err = rt.updateRedisConfig(context.Background(), req)
	assert.NoError(t, err)
	assert.Contains(t, res.Content[0].(mcp.TextContent).Text, "Missing or invalid 'config' object")
	// API error
	mockDB.EXPECT().UpdateRedisConfig(gomock.Any(), "badid", gomock.Any()).Return(nil, assert.AnError)
	args = map[string]interface{}{"id": "badid", "config": config}
	req = mcp.CallToolRequest{Params: mcp.CallToolParams{Arguments: args}}
	res, err = rt.updateRedisConfig(context.Background(), req)
	assert.NoError(t, err)
	assert.Contains(t, res.Content[0].(mcp.TextContent).Text, "api error")
}
