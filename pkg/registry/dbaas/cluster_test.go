package dbaas

import (
	"context"
	"mcp-digitalocean/pkg/registry/dbaas/mocks"
	"testing"
	"time"

	"github.com/digitalocean/godo"
	"github.com/mark3labs/mcp-go/mcp"
	"github.com/stretchr/testify/assert"
	"go.uber.org/mock/gomock"
)

func getText(res *mcp.CallToolResult) string {
	if len(res.Content) > 0 {
		if tc, ok := res.Content[0].(mcp.TextContent); ok {
			return tc.Text
		}
	}
	return ""
}

func TestClusterTool_listCluster(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()
	mockDB := mocks.NewMockDatabasesService(ctrl)
	mockDB.EXPECT().List(gomock.Any(), (*godo.ListOptions)(nil)).Return([]godo.Database{{Name: "test-db"}}, nil, nil)

	client := func(ctx context.Context) (*godo.Client, error) {
		return &godo.Client{
			Databases: mockDB,
		}, nil
	}

	ct := &ClusterTool{client: client}

	req := mcp.CallToolRequest{Params: mcp.CallToolParams{Arguments: map[string]interface{}{}}}
	res, err := ct.listCluster(context.Background(), req)
	assert.NoError(t, err)
	assert.Contains(t, getText(res), "test-db")
}

func TestClusterTool_getCluster(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()
	mockDB := mocks.NewMockDatabasesService(ctrl)
	cluster := &godo.Database{Name: "my-cluster"}
	mockDB.EXPECT().Get(gomock.Any(), "abc").Return(cluster, nil, nil)

	client := func(ctx context.Context) (*godo.Client, error) {
		return &godo.Client{
			Databases: mockDB,
		}, nil
	}

	ct := &ClusterTool{client: client}

	req := mcp.CallToolRequest{Params: mcp.CallToolParams{Arguments: map[string]interface{}{"id": "abc"}}}
	res, err := ct.getCluster(context.Background(), req)
	assert.NoError(t, err)
	assert.Contains(t, getText(res), "my-cluster")

	// Error case: missing id (should not expect a call to Get)
	reqMissing := mcp.CallToolRequest{Params: mcp.CallToolParams{Arguments: map[string]interface{}{}}}
	res, err = ct.getCluster(context.Background(), reqMissing)
	assert.NoError(t, err)
	assert.Contains(t, getText(res), "Cluster id is required")
}

func TestClusterTool_createCluster(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()
	mockDB := mocks.NewMockDatabasesService(ctrl)
	created := &godo.Database{Name: "new-cluster"}
	mockDB.EXPECT().Create(gomock.Any(), gomock.Any()).Return(created, nil, nil)

	client := func(ctx context.Context) (*godo.Client, error) {
		return &godo.Client{
			Databases: mockDB,
		}, nil
	}

	ct := &ClusterTool{client: client}

	args := map[string]interface{}{
		"name":      "new-cluster",
		"engine":    "pg",
		"version":   "13",
		"region":    "nyc1",
		"size":      "db-s-1vcpu-1gb",
		"num_nodes": float64(2),
	}
	req := mcp.CallToolRequest{Params: mcp.CallToolParams{Arguments: args}}
	res, err := ct.createCluster(context.Background(), req)
	assert.NoError(t, err)
	assert.Contains(t, getText(res), "new-cluster")

	// Error case: API error
	ctrl2 := gomock.NewController(t)
	defer ctrl2.Finish()
	mockDB2 := mocks.NewMockDatabasesService(ctrl2)
	mockDB2.EXPECT().Create(gomock.Any(), gomock.Any()).Return(nil, nil, assert.AnError)

	client2 := func(ctx context.Context) (*godo.Client, error) {
		return &godo.Client{
			Databases: mockDB2,
		}, nil
	}

	ct2 := &ClusterTool{client: client2}
	res, err = ct2.createCluster(context.Background(), req)
	assert.NoError(t, err)
	assert.Contains(t, getText(res), "api error")
}

func TestClusterTool_deleteCluster(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()
	mockDB := mocks.NewMockDatabasesService(ctrl)
	mockDB.EXPECT().Delete(gomock.Any(), "abc").Return(nil, nil)

	client := func(ctx context.Context) (*godo.Client, error) {
		return &godo.Client{
			Databases: mockDB,
		}, nil
	}

	ct := &ClusterTool{client: client}

	req := mcp.CallToolRequest{Params: mcp.CallToolParams{Arguments: map[string]interface{}{"id": "abc"}}}
	res, err := ct.deleteCluster(context.Background(), req)
	assert.NoError(t, err)
	assert.Contains(t, getText(res), "Cluster deleted successfully")

	// Error case: missing id
	reqMissing := mcp.CallToolRequest{Params: mcp.CallToolParams{Arguments: map[string]interface{}{}}}
	res, err = ct.deleteCluster(context.Background(), reqMissing)
	assert.NoError(t, err)
	assert.Contains(t, getText(res), "Cluster id is required")
}

func TestClusterTool_resizeCluster(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()
	mockDB := mocks.NewMockDatabasesService(ctrl)
	mockDB.EXPECT().Resize(gomock.Any(), "abc", gomock.Any()).Return(nil, nil)

	client := func(ctx context.Context) (*godo.Client, error) {
		return &godo.Client{
			Databases: mockDB,
		}, nil
	}

	ct := &ClusterTool{client: client}
	args := map[string]interface{}{"id": "abc", "size": "db-s-2vcpu-4gb", "num_nodes": float64(3)}
	req := mcp.CallToolRequest{Params: mcp.CallToolParams{Arguments: args}}
	res, err := ct.resizeCluster(context.Background(), req)
	assert.NoError(t, err)
	assert.Contains(t, getText(res), "Cluster resize initiated successfully")
	// Error case: missing id
	reqMissing := mcp.CallToolRequest{Params: mcp.CallToolParams{Arguments: map[string]interface{}{}}}
	res, err = ct.resizeCluster(context.Background(), reqMissing)
	assert.NoError(t, err)
	assert.Contains(t, getText(res), "Cluster id is required")
}

func TestClusterTool_getCA(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()
	mockDB := mocks.NewMockDatabasesService(ctrl)
	ca := &godo.DatabaseCA{Certificate: []byte("cert-data")}
	mockDB.EXPECT().GetCA(gomock.Any(), "abc").Return(ca, nil, nil)

	client := func(ctx context.Context) (*godo.Client, error) {
		return &godo.Client{
			Databases: mockDB,
		}, nil
	}

	ct := &ClusterTool{client: client}
	req := mcp.CallToolRequest{Params: mcp.CallToolParams{Arguments: map[string]interface{}{"id": "abc"}}}
	res, err := ct.getCA(context.Background(), req)
	assert.NoError(t, err)
	assert.Contains(t, getText(res), "Y2VydC1kYXRh")
	// Error case: missing id
	reqMissing := mcp.CallToolRequest{Params: mcp.CallToolParams{Arguments: map[string]interface{}{}}}
	res, err = ct.getCA(context.Background(), reqMissing)
	assert.NoError(t, err)
	assert.Contains(t, getText(res), "Cluster id is required")
}

func TestClusterTool_listBackups(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()
	mockDB := mocks.NewMockDatabasesService(ctrl)
	mockDB.EXPECT().ListBackups(gomock.Any(), "abc", gomock.Any()).Return([]godo.DatabaseBackup{{CreatedAt: time.Now()}}, nil, nil)

	client := func(ctx context.Context) (*godo.Client, error) {
		return &godo.Client{
			Databases: mockDB,
		}, nil
	}

	ct := &ClusterTool{client: client}
	req := mcp.CallToolRequest{Params: mcp.CallToolParams{Arguments: map[string]interface{}{"id": "abc"}}}
	res, err := ct.listBackups(context.Background(), req)
	assert.NoError(t, err)
	assert.Contains(t, getText(res), "created_at")
	// Error case: missing id
	reqMissing := mcp.CallToolRequest{Params: mcp.CallToolParams{Arguments: map[string]interface{}{}}}
	res, err = ct.listBackups(context.Background(), reqMissing)
	assert.NoError(t, err)
	assert.Contains(t, getText(res), "Cluster id is required")
}

func TestClusterTool_listOptions(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()
	mockDB := mocks.NewMockDatabasesService(ctrl)
	mockDB.EXPECT().ListOptions(gomock.Any()).Return(&godo.DatabaseOptions{}, nil, nil)

	client := func(ctx context.Context) (*godo.Client, error) {
		return &godo.Client{
			Databases: mockDB,
		}, nil
	}

	ct := &ClusterTool{client: client}
	req := mcp.CallToolRequest{Params: mcp.CallToolParams{Arguments: map[string]interface{}{}}}
	res, err := ct.listOptions(context.Background(), req)
	assert.NoError(t, err)
	assert.Contains(t, getText(res), "pg")
}

func TestClusterTool_upgradeMajorVersion(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()
	mockDB := mocks.NewMockDatabasesService(ctrl)
	mockDB.EXPECT().UpgradeMajorVersion(gomock.Any(), "abc", gomock.Any()).Return(nil, nil)

	client := func(ctx context.Context) (*godo.Client, error) {
		return &godo.Client{
			Databases: mockDB,
		}, nil
	}

	ct := &ClusterTool{client: client}
	args := map[string]interface{}{"id": "abc", "version": "15"}
	req := mcp.CallToolRequest{Params: mcp.CallToolParams{Arguments: args}}
	res, err := ct.upgradeMajorVersion(context.Background(), req)
	assert.NoError(t, err)
	assert.Contains(t, getText(res), "Major version upgrade initiated successfully")
	// Error case: missing version
	args = map[string]interface{}{"id": "abc"}
	req = mcp.CallToolRequest{Params: mcp.CallToolParams{Arguments: args}}
	res, err = ct.upgradeMajorVersion(context.Background(), req)
	assert.NoError(t, err)
	assert.Contains(t, getText(res), "Target version is required")
}
