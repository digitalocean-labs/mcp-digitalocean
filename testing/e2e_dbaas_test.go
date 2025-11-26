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
			defer deleteDbaasCluster(ctx, t, c, cluster.ID)

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

	topicName := "mcp-e2e-test-topic"
	// Create cluster with unique name
	clusterName := fmt.Sprintf("mcp-e2e-test-kafka-%s", uuid.New().String())

	// Create a Kafka cluster
	cluster := createDbaasCluster(ctx, t, c, clusterName, "kafka", "3.8", "nyc3", "db-s-2vcpu-4gb", 3)
	defer deleteDbaasCluster(ctx, t, c, cluster.ID)

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

	// List topics
	resp, err = c.CallTool(ctx, mcp.CallToolRequest{
		Params: mcp.CallToolParams{
			Name: "db-cluster-list-topics",
			Arguments: map[string]interface{}{
				"id":       cluster.ID,
				"page":     "1",
				"per_page": 10,
			},
		},
	})
	require.NoError(t, err)
	require.NotNil(t, resp)
	if resp.IsError {
		t.Fatalf("Tool call returned error: %v", resp.Content)
	}

	var topics []godo.DatabaseTopic
	topicsJSON := resp.Content[0].(mcp.TextContent).Text
	err = json.Unmarshal([]byte(topicsJSON), &topics)
	require.NoError(t, err)
	t.Logf("Found %d topics", len(topics))

	foundTargetTopic := false
	for _, targetTopic := range topics {
		t.Logf("%v", targetTopic)
		if targetTopic.Name == topicName {
			foundTargetTopic = true
			t.Logf("Kafka Topic with name %s found in the list", topicName)
			break
		}
	}
	require.Truef(t, foundTargetTopic, "Kafka Topic with name %s not found in list", topicName)

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

	userName := "mcp-e2e-test-user"
	// Create cluster with unique name
	clusterName := fmt.Sprintf("mcp-e2e-test-user-%s", uuid.New().String())

	// Create a cluster
	cluster := createDbaasCluster(ctx, t, c, clusterName, "pg", "14", "nyc3", "db-s-1vcpu-1gb", 1)
	defer deleteDbaasCluster(ctx, t, c, cluster.ID)

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

	// List users
	resp, err = c.CallTool(ctx, mcp.CallToolRequest{
		Params: mcp.CallToolParams{
			Name: "db-cluster-list-users",
			Arguments: map[string]interface{}{
				"id":       cluster.ID,
				"page":     "1",
				"per_page": 10,
			},
		},
	})

	require.NoError(t, err)
	require.NotNil(t, resp)
	if resp.IsError {
		t.Fatalf("Tool call returned error: %v", resp.Content)
	}

	var users []godo.DatabaseUser
	usersJSON := resp.Content[0].(mcp.TextContent).Text
	err = json.Unmarshal([]byte(usersJSON), &users)
	require.NoError(t, err)
	t.Logf("Found %d users", len(users))

	foundTargetUser := false
	for _, targetUser := range users {
		if targetUser.Name == userName {
			foundTargetUser = true
			t.Logf("User with name %s found in the list", userName)
			break
		}
	}
	require.Truef(t, foundTargetUser, "User with name %s not found in list", userName)

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
