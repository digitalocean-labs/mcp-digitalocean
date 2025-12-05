//go:build integration

package testing

import (
	"context"
	"encoding/json"
	"fmt"
	"testing"

	"github.com/digitalocean/godo"
	"github.com/google/uuid"
	"github.com/mark3labs/mcp-go/client"
	"github.com/mark3labs/mcp-go/mcp"
	"github.com/stretchr/testify/require"
)

const (
	e2eSpaceKeyNamePrefix = "mcp-e2e-space-key"
)

type keysResponse struct {
	Keys []godo.SpacesKey `json:"keys"`
}

func TestSpacesKeyLifecycle(t *testing.T) {
	ctx := context.Background()
	c := initializeClient(ctx, t)
	t.Cleanup(func() {
		flushKeys(t, c)
		c.Close()
	})

	// first we create a key
	postfix, err := uuid.NewUUID()
	require.NoError(t, err)

	resp, err := c.CallTool(context.Background(), mcp.CallToolRequest{
		Params: mcp.CallToolParams{
			Name: "spaces-key-create",
			Arguments: map[string]interface{}{
				"Name": fmt.Sprintf("%s-%s", e2eSpaceKeyNamePrefix, postfix.String()),
			},
		},
	})
	requireBasicResponse(t, err, resp)

	var createdKey godo.SpacesKey
	createdKeyJSON := resp.Content[0].(mcp.TextContent).Text
	err = json.Unmarshal([]byte(createdKeyJSON), &createdKey)

	// now we get the key we created
	resp, err = c.CallTool(context.Background(), mcp.CallToolRequest{
		Params: mcp.CallToolParams{
			Name: "spaces-key-get",
			Arguments: map[string]interface{}{
				"AccessKey": createdKey.AccessKey,
			},
		},
	})
	requireBasicResponse(t, err, resp)

	var key godo.SpacesKey
	keyJSON := resp.Content[0].(mcp.TextContent).Text
	err = json.Unmarshal([]byte(keyJSON), &key)
	require.NoError(t, err)

	// the key from create and get should be the same
	require.Equal(t, createdKey.Name, key.Name)
	require.Equal(t, createdKey.AccessKey, key.AccessKey)

	// the key should be present when listed.
	resp, err = c.CallTool(context.Background(), mcp.CallToolRequest{
		Params: mcp.CallToolParams{
			Name: "spaces-key-list",
		},
	})

	requireBasicResponse(t, err, resp)

	var listKeysResponse keysResponse
	listedSpaceKeys := resp.Content[0].(mcp.TextContent).Text
	err = json.Unmarshal([]byte(listedSpaceKeys), &listKeysResponse)
	require.NoError(t, err)

	requireKeyPresent(t, key, listKeysResponse.Keys)

	// finally we delete the key
	resp, err = c.CallTool(context.Background(), mcp.CallToolRequest{
		Params: mcp.CallToolParams{
			Name: "spaces-key-delete",
			Arguments: map[string]interface{}{
				"AccessKey": key.AccessKey,
			},
		},
	})

	requireBasicResponse(t, err, resp)

	// validate that the key is no longer present
	resp, err = c.CallTool(context.Background(), mcp.CallToolRequest{
		Params: mcp.CallToolParams{
			Name: "spaces-key-list",
		},
	})

	requireBasicResponse(t, err, resp)
	require.True(t, resp.IsError)

	// check that the content of the response contains Not Found
	require.Contains(t, resp.Content[0].(mcp.TextContent).Text, "Not Found")
}

// cleanupKeys removes any keys created during testing.
func flushKeys(t *testing.T, c *client.Client) {
	ctx := context.Background()
	resp, err := c.CallTool(ctx, mcp.CallToolRequest{
		Params: mcp.CallToolParams{
			Name: "spaces-key-list",
		},
	})
	if err != nil || resp == nil || len(resp.Content) == 0 {
		t.Logf("Error listing spaces keys for cleanup: %v", err)
		return // nothing to clean up or error listing keys
	}

	var listKeysResponse keysResponse
	keyListJSON := resp.Content[0].(mcp.TextContent).Text
	err = json.Unmarshal([]byte(keyListJSON), &listKeysResponse)
	if err != nil {
		return
	}

	for _, key := range listKeysResponse.Keys {
		resp, err = c.CallTool(ctx, mcp.CallToolRequest{
			Params: mcp.CallToolParams{
				Name: "spaces-key-delete",
				Arguments: map[string]interface{}{
					"AccessKey": key.AccessKey,
				},
			},
		})
		if err != nil {
			t.Logf("Error deleting spaces key with AccessKey %s: %v", key.AccessKey, err)
		}
		if resp.IsError {
			t.Logf("Error response deleting spaces key with AccessKey %s: %v", key.AccessKey, resp.Content)
		}
	}
}

func requireKeyPresent(t *testing.T, key godo.SpacesKey, keys []godo.SpacesKey) {
	found := false
	for _, k := range keys {
		if k.AccessKey == key.AccessKey && k.Name == key.Name {
			found = true
			break
		}
	}

	require.True(t, found, "Expected key with AccessKey %s to be present in list", key.AccessKey)
}

func requireBasicResponse(t *testing.T, err error, resp *mcp.CallToolResult) {
	require.NoError(t, err)
	require.NotNil(t, resp)
	require.NotEmpty(t, resp.Content, "No content returned from tool")
}
