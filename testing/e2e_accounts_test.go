//go:build integration

package testing

import (
	"context"
	"crypto/rand"
	"crypto/rsa"
	"encoding/json"
	"golang.org/x/crypto/ssh"
	"testing"

	"github.com/digitalocean/godo"
	"github.com/mark3labs/mcp-go/mcp"
	"github.com/stretchr/testify/require"
)

func TestAccountGetInformation(t *testing.T) {
	ctx := context.Background()
	c := initializeClient(ctx, t)
	defer c.Close()

	resp, err := c.CallTool(context.Background(), mcp.CallToolRequest{
		Params: mcp.CallToolParams{
			Name: "account-get-information",
			Arguments: map[string]interface{}{
				"Page":    1,
				"PerPage": 10,
			},
		},
	})

	require.NoError(t, err)
	require.NotNil(t, resp)
	if resp.IsError {
		t.Fatalf("Tool call returned error: %v", resp.Content)
	}

	var account godo.Account
	accountsJSON := resp.Content[0].(mcp.TextContent).Text
	err = json.Unmarshal([]byte(accountsJSON), &account)
	require.NoError(t, err)
	require.NotEmpty(t, account.UUID, "Account UUID should not be empty")
	require.NotEmpty(t, account.Email, "Account Email should not be empty")

	t.Logf("Account Information %v", account)
}

func TestActionGet(t *testing.T) {
	ctx := context.Background()
	c := initializeClient(ctx, t)
	defer c.Close()

	resp, err := c.CallTool(context.Background(), mcp.CallToolRequest{
		Params: mcp.CallToolParams{
			Name: "action-list",
			Arguments: map[string]interface{}{
				"Page":    1,
				"PerPage": 1,
			},
		},
	})

	require.NoError(t, err)
	require.NotNil(t, resp)
	if resp.IsError {
		t.Fatalf("Tool call returned error: %v", resp.Content)
	}
	var action []godo.Action
	actionJSON := resp.Content[0].(mcp.TextContent).Text
	err = json.Unmarshal([]byte(actionJSON), &action)
	require.NoError(t, err)
	require.NotEmpty(t, action, "Action list should not be empty")

	actionID := action[0].ID
	require.NotZero(t, actionID, "Action ID should not be zero")

	t.Logf("First Action ID: %v", actionID)

	respActionID, err := c.CallTool(context.Background(), mcp.CallToolRequest{
		Params: mcp.CallToolParams{
			Name: "action-get",
			Arguments: map[string]interface{}{
				"ID": actionID,
			},
		},
	})

	require.NoError(t, err)
	require.NotNil(t, respActionID)
	if respActionID.IsError {
		t.Fatalf("Tool call returned error: %v", respActionID.Content)
	}
	var actionGet []godo.Action
	actionGETJSON := resp.Content[0].(mcp.TextContent).Text
	err = json.Unmarshal([]byte(actionGETJSON), &actionGet)
	require.NoError(t, err)
	require.NotEmpty(t, actionGet, "Action list should not be empty")
	t.Logf("Found Action with ID: %v", actionGet[0].ID)

}

func TestGetBalance(t *testing.T) {
	ctx := context.Background()
	c := initializeClient(ctx, t)
	defer c.Close()

	resp, err := c.CallTool(context.Background(), mcp.CallToolRequest{
		Params: mcp.CallToolParams{
			Name: "balance-get",
			Arguments: map[string]interface{}{
				"Page":    1,
				"PerPage": 10,
			},
		},
	})

	require.NoError(t, err)
	require.NotNil(t, resp)
	if resp.IsError {
		t.Fatalf("Tool call returned error: %v", resp.Content)
	}

	var balance godo.Balance
	balanceJSON := resp.Content[0].(mcp.TextContent).Text
	err = json.Unmarshal([]byte(balanceJSON), &balance)
	require.NoError(t, err)

	t.Logf("Balance Due %v", balance)
}

func TestBillingHistory(t *testing.T) {
	ctx := context.Background()
	c := initializeClient(ctx, t)
	defer c.Close()

	resp, err := c.CallTool(context.Background(), mcp.CallToolRequest{
		Params: mcp.CallToolParams{
			Name: "billing-history-list",
			Arguments: map[string]interface{}{
				"Page":    1,
				"PerPage": 10,
			},
		},
	})

	require.NoError(t, err)
	require.NotNil(t, resp)
	if resp.IsError {
		t.Fatalf("Tool call returned error: %v", resp.Content)
	}

	var balance godo.Balance
	balanceJSON := resp.Content[0].(mcp.TextContent).Text
	err = json.Unmarshal([]byte(balanceJSON), &balance)
	require.NoError(t, err)
	t.Logf("Balance History %v", balance)
}

func TestInvoice(t *testing.T) {
	ctx := context.Background()
	c := initializeClient(ctx, t)
	defer c.Close()

	resp, err := c.CallTool(context.Background(), mcp.CallToolRequest{
		Params: mcp.CallToolParams{
			Name: "invoice-list",
			Arguments: map[string]interface{}{
				"Page":    1,
				"PerPage": 10,
			},
		},
	})

	require.NoError(t, err)
	require.NotNil(t, resp)
	if resp.IsError {
		t.Fatalf("Tool call returned error: %v", resp.Content)
	}

	var invoice godo.Invoice
	invoiceJSON := resp.Content[0].(mcp.TextContent).Text
	err = json.Unmarshal([]byte(invoiceJSON), &invoice)
	require.NoError(t, err)

	t.Logf("invoice List %v", invoice)
}

func TestKeys(t *testing.T) {
	const TestKeyName = "test-key-mcp"
	publicKey, err := generateSSHKeyPair()
	require.NoError(t, err, "Error generating SSH key pair")
	require.NotEmpty(t, publicKey, "Generated public key should not be empty")
	ctx := context.Background()
	c := initializeClient(ctx, t)
	defer c.Close()

	// Create a new SSH key
	createResp, err := c.CallTool(context.Background(), mcp.CallToolRequest{
		Params: mcp.CallToolParams{
			Name: "key-create",
			Arguments: map[string]interface{}{
				"Name":      TestKeyName,
				"PublicKey": publicKey,
			},
		},
	})
	require.NoError(t, err)
	require.NotNil(t, createResp)
	if createResp.IsError {
		t.Fatalf("Key create returned error: %v", createResp.Content)
	}

	// List keys
	respGetList, err := c.CallTool(context.Background(), mcp.CallToolRequest{
		Params: mcp.CallToolParams{
			Name: "key-list",
			Arguments: map[string]interface{}{
				"Page":    1,
				"PerPage": 10,
			},
		},
	})

	require.NoError(t, err)
	require.NotNil(t, respGetList)
	if respGetList.IsError {
		t.Fatalf("Tool call returned error: %v", respGetList.Content)
	}

	var keyList []godo.Key
	keyListJSON := respGetList.Content[0].(mcp.TextContent).Text
	err = json.Unmarshal([]byte(keyListJSON), &keyList)
	require.NoError(t, err)
	require.NotEmpty(t, keyList, "key list should not be empty")

	t.Logf("Found Keys %v", keyList)

	// Find the created key by name
	var createdKeyID int
	for _, k := range keyList {
		if k.Name == "test-key-mcp" {
			createdKeyID = k.ID
			break
		}
	}
	require.NotZero(t, createdKeyID, "Created key not found in key list")
	t.Logf("Deleting key with ID: %v", createdKeyID)

	// Call key-delete tool
	respDelete, err := c.CallTool(context.Background(), mcp.CallToolRequest{
		Params: mcp.CallToolParams{
			Name: "key-delete",
			Arguments: map[string]interface{}{
				"ID": createdKeyID,
			},
		},
	})
	require.NoError(t, err)
	require.NotNil(t, respDelete)
	if respDelete.IsError {
		t.Fatalf("Key delete returned error: %v", respDelete.Content)
	}
	t.Logf("Key deleted: %v", respDelete.Content)
}

func generateSSHKeyPair() (string, error) {
	privateKey, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		return "", err
	}

	pub, err := ssh.NewPublicKey(&privateKey.PublicKey)
	if err != nil {
		return "", err
	}

	return string(ssh.MarshalAuthorizedKey(pub)), nil
}
