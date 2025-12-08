//go:build integration

package testing

import (
	"context"
	"encoding/json"
	"fmt"
	"testing"
	"time"

	"github.com/digitalocean/godo"
	"github.com/google/uuid"
	"github.com/mark3labs/mcp-go/mcp"
	"github.com/stretchr/testify/require"
)

var dbaasEngines = []struct {
	name      string
	engine    string
	version   string
	region    string
	size      string
	nodeCount int
}{
	{"postgres", "pg", "14", "nyc3", "db-s-1vcpu-1gb", 1},
	{"mysql", "mysql", "8", "nyc3", "db-s-1vcpu-1gb", 1},
	{"mongodb", "mongodb", "8", "nyc3", "db-s-1vcpu-1gb", 1},
	{"valkey", "valkey", "8", "nyc3", "db-s-1vcpu-1gb", 1},
	{"kafka", "kafka", "3.8", "nyc3", "db-s-2vcpu-4gb", 3},
	{"opensearch", "opensearch", "2", "nyc3", "db-s-1vcpu-2gb", 1},
}

func TestDbaasClusterLifecycle(t *testing.T) {
	registerClusterCleanup(t)

	for _, tc := range dbaasEngines {
		tc := tc // capture the range variable

		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			// Each test needs its own ctx and client
			ctx := context.Background()
			c := initializeClient(ctx, t)
			defer c.Close()

			// Create cluster with unique name
			clusterName := fmt.Sprintf("mcp-e2e-test-%s", tc.name)

			cluster := createDbaasCluster(ctx, t, c,
				clusterName,
				tc.engine, tc.version, tc.region, tc.size, tc.nodeCount,
			)

			// Validate cluster appears in list
			dbaasAssertClusterExists(ctx, t, c, cluster.ID)
		})
	}
}

func TestDbaasKafkaLifecycle(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	c := initializeClient(ctx, t)
	defer c.Close()

	registerClusterCleanup(t)

	topicName := "mcp-e2e-test-topic"
	// Create cluster with unique name
	clusterName := fmt.Sprintf("mcp-e2e-test-kafka-%s", uuid.New().String())

	// Create a Kafka cluster
	cluster := createDbaasCluster(ctx, t, c, clusterName, "kafka", "3.8", "nyc3", "db-s-2vcpu-4gb", 3)

	// Wait for kafka Cluster to become online
	waitForDbaasClusterActive(ctx, c, t, cluster.ID, 15*time.Minute)

	// Create a topic
	resp, err := c.CallTool(ctx, mcp.CallToolRequest{
		Params: mcp.CallToolParams{
			Name: "db-cluster-create-topic",
			Arguments: map[string]interface{}{
				"id":   cluster.ID,
				"name": topicName,
			},
		},
	})

	require.NoError(t, err)
	require.NotNil(t, resp)
	if resp.IsError {
		t.Fatalf("Tool call returned error: %v", resp.Content)
	}

	var topic godo.DatabaseTopic
	topicJSON := resp.Content[0].(mcp.TextContent).Text
	err = json.Unmarshal([]byte(topicJSON), &topic)
	require.NoError(t, err)
	t.Logf("Created Kafka Topic with name: %s", topicName)

	require.Eventually(t, func() bool {
		resp, err := c.CallTool(ctx, mcp.CallToolRequest{
			Params: mcp.CallToolParams{
				Name: "db-cluster-list-topics",
				Arguments: map[string]interface{}{
					"id":       cluster.ID,
					"page":     "1",
					"per_page": 10,
				},
			},
		})

		if err != nil || resp == nil || (resp != nil && resp.IsError) {
			return false
		}

		var topics []godo.DatabaseTopic
		topicsJSON := resp.Content[0].(mcp.TextContent).Text
		if err := json.Unmarshal([]byte(topicsJSON), &topics); err != nil {
			return false
		}

		for _, tp := range topics {
			if tp.Name == topicName {
				t.Logf("Kafka Topic with name %s found in the list", topicName)
				return true
			}
		}
		return false
	}, 60*time.Second, 3*time.Second, "Kafka topic did not appear in list in time")

	// Delete the topic
	resp, err = c.CallTool(ctx, mcp.CallToolRequest{
		Params: mcp.CallToolParams{
			Name: "db-cluster-delete-topic",
			Arguments: map[string]interface{}{
				"id":   cluster.ID,
				"name": topicName,
			},
		},
	})
	require.NoError(t, err)
	require.NotNil(t, resp)
	if resp.IsError {
		t.Fatalf("Tool call returned error: %v", resp.Content)
	}

	t.Logf("Deleted topic with name: %s", topicName)
}

func TestDbaasUserLifecycle(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	c := initializeClient(ctx, t)
	defer c.Close()

	registerClusterCleanup(t)

	userName := "mcp-e2e-test-user"
	// Create cluster with unique name
	clusterName := fmt.Sprintf("mcp-e2e-test-user-%s", uuid.New().String())

	// Create a cluster
	cluster := createDbaasCluster(ctx, t, c, clusterName, "pg", "14", "nyc3", "db-s-1vcpu-1gb", 1)

	// Wait for Db Cluster to become online
	waitForDbaasClusterActive(ctx, c, t, cluster.ID, 15*time.Minute)

	// Create a user
	resp, err := c.CallTool(ctx, mcp.CallToolRequest{
		Params: mcp.CallToolParams{
			Name: "db-cluster-create-user",
			Arguments: map[string]interface{}{
				"id":   cluster.ID,
				"name": userName,
			},
		},
	})
	require.NoError(t, err)
	require.NotNil(t, resp)
	if resp.IsError {
		t.Fatalf("Tool call returned error: %v", resp.Content)
	}

	var user godo.DatabaseUser
	userJSON := resp.Content[0].(mcp.TextContent).Text
	err = json.Unmarshal([]byte(userJSON), &user)
	require.NoError(t, err)
	t.Logf("Created user with username: %s", userName)

	require.Eventually(t, func() bool {
		resp, err := c.CallTool(ctx, mcp.CallToolRequest{
			Params: mcp.CallToolParams{
				Name: "db-cluster-list-users",
				Arguments: map[string]interface{}{
					"id":       cluster.ID,
					"page":     "1",
					"per_page": 100,
				},
			},
		})
		if err != nil || resp == nil || (resp != nil && resp.IsError) {
			return false
		}

		var users []godo.DatabaseUser
		usersJSON := resp.Content[0].(mcp.TextContent).Text
		json.Unmarshal([]byte(usersJSON), &users)

		for _, u := range users {
			if u.Name == userName {
				t.Logf("User with name %s found in the list", userName)
				return true
			}
		}

		return false
	}, 30*time.Second, 2*time.Second, "user did not appear in time")

	// Delete the user
	resp, err = c.CallTool(ctx, mcp.CallToolRequest{
		Params: mcp.CallToolParams{
			Name: "db-cluster-delete-user",
			Arguments: map[string]interface{}{
				"id":   cluster.ID,
				"user": userName,
			},
		},
	})
	require.NoError(t, err)
	require.NotNil(t, resp)
	if resp.IsError {
		t.Fatalf("Tool call returned error: %v", resp.Content)
	}

	t.Logf("Deleted user with name: %s", userName)
}
