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

func TestMysqlTool_getMySQLConfig(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()
	mockDB := mocks.NewMockDatabasesService(ctrl)
	mockDB.EXPECT().GetMySQLConfig(gomock.Any(), "cid").Return(&godo.MySQLConfig{SQLMode: new(string)}, nil, nil)

	client := func(ctx context.Context) (*godo.Client, error) {
		return &godo.Client{
			Databases: mockDB,
		}, nil
	}

	mt := &MysqlTool{client: client}
	args := map[string]interface{}{"id": "cid"}
	req := mcp.CallToolRequest{Params: mcp.CallToolParams{Arguments: args}}
	res, err := mt.getMySQLConfig(context.Background(), req)
	assert.NoError(t, err)
	assert.Contains(t, res.Content[0].(mcp.TextContent).Text, "sql_mode")
	// Error case: missing id (should not expect a call to GetMySQLConfig)
	reqMissing := mcp.CallToolRequest{Params: mcp.CallToolParams{Arguments: map[string]interface{}{}}}
	res, err = mt.getMySQLConfig(context.Background(), reqMissing)
	assert.NoError(t, err)
	assert.Contains(t, res.Content[0].(mcp.TextContent).Text, "Cluster id is required")
}

func TestMysqlTool_updateMySQLConfig(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()
	mockDB := mocks.NewMockDatabasesService(ctrl)
	mockDB.EXPECT().UpdateMySQLConfig(gomock.Any(), "cid", gomock.Any()).Return(nil, nil)

	client := func(ctx context.Context) (*godo.Client, error) {
		return &godo.Client{
			Databases: mockDB,
		}, nil
	}

	mt := &MysqlTool{client: client}
	cfg := map[string]any{}
	args := map[string]interface{}{"id": "cid", "config": cfg}
	req := mcp.CallToolRequest{Params: mcp.CallToolParams{Arguments: args}}
	res, err := mt.updateMySQLConfig(context.Background(), req)
	assert.NoError(t, err)
	assert.Contains(t, res.Content[0].(mcp.TextContent).Text, "MySQL config updated successfully")
	// Error case: missing config
	args = map[string]interface{}{"id": "cid"}
	req = mcp.CallToolRequest{Params: mcp.CallToolParams{Arguments: args}}
	res, err = mt.updateMySQLConfig(context.Background(), req)
	assert.NoError(t, err)
	assert.Contains(t, res.Content[0].(mcp.TextContent).Text, "Invalid or missing 'config' object")
}

func TestMysqlTool_getSQLMode(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()
	mockDB := mocks.NewMockDatabasesService(ctrl)
	mockDB.EXPECT().GetSQLMode(gomock.Any(), "cid").Return("STRICT_TRANS_TABLES", nil, nil)

	client := func(ctx context.Context) (*godo.Client, error) {
		return &godo.Client{
			Databases: mockDB,
		}, nil
	}

	mt := &MysqlTool{client: client}
	args := map[string]interface{}{"id": "cid"}
	req := mcp.CallToolRequest{Params: mcp.CallToolParams{Arguments: args}}
	res, err := mt.getSQLMode(context.Background(), req)
	assert.NoError(t, err)
	assert.Contains(t, res.Content[0].(mcp.TextContent).Text, "STRICT_TRANS_TABLES")
	// Error case: missing id (should not expect a call to GetSQLMode)
	reqMissing := mcp.CallToolRequest{Params: mcp.CallToolParams{Arguments: map[string]interface{}{}}}
	res, err = mt.getSQLMode(context.Background(), reqMissing)
	assert.NoError(t, err)
	assert.Contains(t, res.Content[0].(mcp.TextContent).Text, "Cluster id is required")
}

func TestMysqlTool_setSQLMode(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()
	mockDB := mocks.NewMockDatabasesService(ctrl)
	mockDB.EXPECT().SetSQLMode(gomock.Any(), "cid", gomock.Any()).Return(nil, nil)

	client := func(ctx context.Context) (*godo.Client, error) {
		return &godo.Client{
			Databases: mockDB,
		}, nil
	}

	mt := &MysqlTool{client: client}
	args := map[string]interface{}{"id": "cid", "modes": "STRICT_TRANS_TABLES"}
	req := mcp.CallToolRequest{Params: mcp.CallToolParams{Arguments: args}}
	res, err := mt.setSQLMode(context.Background(), req)
	assert.NoError(t, err)
	assert.Contains(t, res.Content[0].(mcp.TextContent).Text, "SQL mode set successfully")
	// Error case: missing modes
	args = map[string]interface{}{"id": "cid"}
	req = mcp.CallToolRequest{Params: mcp.CallToolParams{Arguments: args}}
	res, err = mt.setSQLMode(context.Background(), req)
	assert.NoError(t, err)
	assert.Contains(t, res.Content[0].(mcp.TextContent).Text, "SQL modes are required")
}
