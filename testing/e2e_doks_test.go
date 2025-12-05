//go:build integration

package testing

import (
	"context"
	"encoding/json"
	"fmt"
	"testing"

	"github.com/digitalocean/godo"
	"github.com/google/uuid"
	"github.com/mark3labs/mcp-go/mcp"
	"github.com/stretchr/testify/require"
)

func TestCreateCluster(t *testing.T) {
	ctx := context.Background()
	c := initializeClient(ctx, t)
	defer c.Close()

	create := godo.KubernetesClusterCreateRequest{
		Name:        fmt.Sprintf("doks-%s", uuid.New().String()),
		RegionSlug:  "sfo3",
		VersionSlug: "latest",
		NodePools: []*godo.KubernetesNodePoolCreateRequest{
			{
				Name:     "mcp-node-pool-1",
				Size:     "s-1vcpu-2gb",
				Count:    1,
				MinNodes: 1,
				MaxNodes: 1,
			},
		},
	}

	resp, err := c.CallTool(context.Background(), mcp.CallToolRequest{
		Params: mcp.CallToolParams{
			Name:      "doks-create-cluster",
			Arguments: create,
		},
	})

	require.NoError(t, err)
	require.NotNil(t, resp)

	if resp.IsError {
		fmt.Printf("response is error: %s", resp.Content[0].(mcp.TextContent).Text)
		t.Fail()
	}

	var doksCluster godo.KubernetesCluster
	doksClusterJSON := resp.Content[0].(mcp.TextContent).Text
	err = json.Unmarshal([]byte(doksClusterJSON), &doksCluster)
	require.NoError(t, err)
	require.NotEmpty(t, doksCluster.ID)

	fmt.Printf("created doks cluster: %v+", doksCluster)

	resp, err = c.CallTool(ctx, mcp.CallToolRequest{
		Params: mcp.CallToolParams{
			Name: "doks-get-cluster",
			Arguments: map[string]interface{}{
				"ClusterID": doksCluster.ID,
			},
		},
	})
	require.NoError(t, err)
	require.False(t, resp.IsError)
	doksClusterJSON = resp.Content[0].(mcp.TextContent).Text
	err = json.Unmarshal([]byte(doksClusterJSON), &doksCluster)
	require.NoError(t, err)
	require.NotEmpty(t, doksCluster.ID)

	resp, err = c.CallTool(ctx, mcp.CallToolRequest{
		Params: mcp.CallToolParams{
			Name: "doks-delete-cluster",
			Arguments: map[string]interface{}{
				"ClusterID": doksCluster.ID,
			},
		},
	})

	require.NoError(t, err)
	require.False(t, resp.IsError)

	fmt.Printf("doks cluster %s deleted", doksCluster.ID)
}
