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

func TestKafkaTool_getKafkaConfig(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()
	mockDB := mocks.NewMockDatabasesService(ctrl)
	mockDB.EXPECT().GetKafkaConfig(gomock.Any(), "cid").Return(&godo.KafkaConfig{}, nil, nil)

	client := func(ctx context.Context) (*godo.Client, error) {
		return &godo.Client{
			Databases: mockDB,
		}, nil
	}

	kt := &KafkaTool{client: client}
	args := map[string]interface{}{"id": "cid"}
	req := mcp.CallToolRequest{Params: mcp.CallToolParams{Arguments: args}}
	res, err := kt.getKafkaConfig(context.Background(), req)
	assert.NoError(t, err)
	assert.Contains(t, res.Content[0].(mcp.TextContent).Text, "{")
	// Error case: missing id (should not expect a call to GetKafkaConfig)
	reqMissing := mcp.CallToolRequest{Params: mcp.CallToolParams{Arguments: map[string]interface{}{}}}
	res, err = kt.getKafkaConfig(context.Background(), reqMissing)
	assert.NoError(t, err)
	assert.Contains(t, res.Content[0].(mcp.TextContent).Text, "Cluster id is required")
}

func TestKafkaTool_updateKafkaConfig(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()
	mockDB := mocks.NewMockDatabasesService(ctrl)
	mockDB.EXPECT().UpdateKafkaConfig(gomock.Any(), "cid", gomock.Any()).Return(nil, nil)

	client := func(ctx context.Context) (*godo.Client, error) {
		return &godo.Client{
			Databases: mockDB,
		}, nil
	}

	kt := &KafkaTool{client: client}
	cfg := map[string]any{}
	args := map[string]interface{}{"id": "cid", "config": cfg}
	req := mcp.CallToolRequest{Params: mcp.CallToolParams{Arguments: args}}
	res, err := kt.updateKafkaConfig(context.Background(), req)
	assert.NoError(t, err)
	assert.Contains(t, res.Content[0].(mcp.TextContent).Text, "Kafka config updated successfully")
	// Error case: missing config
	args = map[string]interface{}{"id": "cid"}
	req = mcp.CallToolRequest{Params: mcp.CallToolParams{Arguments: args}}
	res, err = kt.updateKafkaConfig(context.Background(), req)
	assert.NoError(t, err)
	assert.Contains(t, res.Content[0].(mcp.TextContent).Text, "Missing or invalid 'config' object")
}

func TestKafkaTool_listTopics(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()
	mockDB := mocks.NewMockDatabasesService(ctrl)
	mockDB.EXPECT().ListTopics(gomock.Any(), "cid", gomock.Any()).Return([]godo.DatabaseTopic{{Name: "topic1"}}, nil, nil)

	client := func(ctx context.Context) (*godo.Client, error) {
		return &godo.Client{
			Databases: mockDB,
		}, nil
	}

	kt := &KafkaTool{client: client}
	args := map[string]interface{}{"id": "cid"}
	req := mcp.CallToolRequest{Params: mcp.CallToolParams{Arguments: args}}
	res, err := kt.listTopics(context.Background(), req)
	assert.NoError(t, err)
	assert.Contains(t, res.Content[0].(mcp.TextContent).Text, "topic1")
	// Error case: missing id
	reqMissing := mcp.CallToolRequest{Params: mcp.CallToolParams{Arguments: map[string]interface{}{}}}
	res, err = kt.listTopics(context.Background(), reqMissing)
	assert.NoError(t, err)
	assert.Contains(t, res.Content[0].(mcp.TextContent).Text, "Cluster id is required")
}

func TestKafkaTool_createTopic(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()
	mockDB := mocks.NewMockDatabasesService(ctrl)
	mockDB.EXPECT().CreateTopic(gomock.Any(), "cid", gomock.Any()).Return(&godo.DatabaseTopic{Name: "topic2"}, nil, nil)

	client := func(ctx context.Context) (*godo.Client, error) {
		return &godo.Client{
			Databases: mockDB,
		}, nil
	}

	kt := &KafkaTool{client: client}
	args := map[string]interface{}{"id": "cid", "name": "topic2"}
	req := mcp.CallToolRequest{Params: mcp.CallToolParams{Arguments: args}}
	res, err := kt.createTopic(context.Background(), req)
	assert.NoError(t, err)
	assert.Contains(t, res.Content[0].(mcp.TextContent).Text, "topic2")
	// Error case: missing name
	args = map[string]interface{}{"id": "cid"}
	req = mcp.CallToolRequest{Params: mcp.CallToolParams{Arguments: args}}
	res, err = kt.createTopic(context.Background(), req)
	assert.NoError(t, err)
	assert.Contains(t, res.Content[0].(mcp.TextContent).Text, "Topic name is required")
}

func TestKafkaTool_getTopic(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()
	mockDB := mocks.NewMockDatabasesService(ctrl)
	mockDB.EXPECT().GetTopic(gomock.Any(), "cid", "topic3").Return(&godo.DatabaseTopic{Name: "topic3"}, nil, nil)

	client := func(ctx context.Context) (*godo.Client, error) {
		return &godo.Client{
			Databases: mockDB,
		}, nil
	}

	kt := &KafkaTool{client: client}
	args := map[string]interface{}{"id": "cid", "name": "topic3"}
	req := mcp.CallToolRequest{Params: mcp.CallToolParams{Arguments: args}}
	res, err := kt.getTopic(context.Background(), req)
	assert.NoError(t, err)
	assert.Contains(t, res.Content[0].(mcp.TextContent).Text, "topic3")
	// Error case: missing name
	args = map[string]interface{}{"id": "cid"}
	req = mcp.CallToolRequest{Params: mcp.CallToolParams{Arguments: args}}
	res, err = kt.getTopic(context.Background(), req)
	assert.NoError(t, err)
	assert.Contains(t, res.Content[0].(mcp.TextContent).Text, "Topic name is required")
}

func TestKafkaTool_deleteTopic(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()
	mockDB := mocks.NewMockDatabasesService(ctrl)
	mockDB.EXPECT().DeleteTopic(gomock.Any(), "cid", "topic4").Return(nil, nil)

	client := func(ctx context.Context) (*godo.Client, error) {
		return &godo.Client{
			Databases: mockDB,
		}, nil
	}

	kt := &KafkaTool{client: client}
	args := map[string]interface{}{"id": "cid", "name": "topic4"}
	req := mcp.CallToolRequest{Params: mcp.CallToolParams{Arguments: args}}
	res, err := kt.deleteTopic(context.Background(), req)
	assert.NoError(t, err)
	assert.Contains(t, res.Content[0].(mcp.TextContent).Text, "Topic deleted successfully")
	// Error case: missing name
	args = map[string]interface{}{"id": "cid"}
	req = mcp.CallToolRequest{Params: mcp.CallToolParams{Arguments: args}}
	res, err = kt.deleteTopic(context.Background(), req)
	assert.NoError(t, err)
	assert.Contains(t, res.Content[0].(mcp.TextContent).Text, "Topic name is required")
}

func TestKafkaTool_updateTopic(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()
	mockDB := mocks.NewMockDatabasesService(ctrl)
	mockDB.EXPECT().UpdateTopic(gomock.Any(), "cid", "topic5", gomock.Any()).Return(nil, nil)

	client := func(ctx context.Context) (*godo.Client, error) {
		return &godo.Client{
			Databases: mockDB,
		}, nil
	}

	kt := &KafkaTool{client: client}
	args := map[string]interface{}{"id": "cid", "name": "topic5"}
	req := mcp.CallToolRequest{Params: mcp.CallToolParams{Arguments: args}}
	res, err := kt.updateTopic(context.Background(), req)
	assert.NoError(t, err)
	assert.Contains(t, res.Content[0].(mcp.TextContent).Text, "Topic updated successfully")
	// Error case: missing name
	args = map[string]interface{}{"id": "cid"}
	req = mcp.CallToolRequest{Params: mcp.CallToolParams{Arguments: args}}
	res, err = kt.updateTopic(context.Background(), req)
	assert.NoError(t, err)
	assert.Contains(t, res.Content[0].(mcp.TextContent).Text, "Topic name is required")
}
