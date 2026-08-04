//go:build integration

package testing

import (
	"testing"

	"github.com/digitalocean/godo"
	"github.com/stretchr/testify/require"
)

func TestVectorDBLifecycle(t *testing.T) {
	t.Parallel()

	newDB := CreateTestVectorDB(t, "vdb-e2e-lifecycle")

	activeDB := WaitForVectorDBActive(t, newDB.ID, defaultActionTimeout)
	require.Equal(t, "small", activeDB.Size)

	// Get
	getDB := callTool[godo.VectorDB](t, "vector-db-get", map[string]any{
		"ID": activeDB.ID,
	})
	require.Equal(t, activeDB.ID, getDB.ID)
	require.NotNil(t, getDB.Endpoints, "active vector database should expose connection endpoints")
	t.Logf("[Get] Successfully retrieved vector database:")
	t.Logf("      Name: %s", getDB.Name)
	t.Logf("      ID: %s", getDB.ID)
	t.Logf("      Size: %s", getDB.Size)
	t.Logf("      Status: %s", getDB.Status)
	if getDB.Endpoints != nil {
		t.Logf("      Endpoints: http=%s grpc=%s", getDB.Endpoints.HTTP, getDB.Endpoints.GRPC)
	}

	// Credentials
	creds := callTool[godo.VectorDBAdminCredentials](t, "vector-db-get-credentials", map[string]any{
		"ID": activeDB.ID,
	})
	require.NotEmpty(t, creds.APIToken, "expected an admin api token")
	t.Logf("[Credentials] Successfully retrieved admin credentials: UserID=%s", creds.UserID)

	// Resize small -> medium
	t.Logf("[Resize] Resizing vector database %s to medium...", activeDB.ID)
	_ = callTool[godo.VectorDB](t, "vector-db-resize", map[string]any{
		"ID":   activeDB.ID,
		"Size": "medium",
	})

	var resizedDB godo.VectorDB
	require.Eventually(t, func() bool {
		resizedDB = callTool[godo.VectorDB](t, "vector-db-get", map[string]any{
			"ID": activeDB.ID,
		})
		return resizedDB.Size == "medium" && resizedDB.Status == "active"
	}, defaultActionTimeout, defaultPollInterval, "vector database did not resize in time")

	t.Logf("[Resize] Successfully resized vector database: %s to %s", resizedDB.ID, resizedDB.Size)

	// Delete
	t.Logf("[Delete] Deleting vector database: %s...", activeDB.Name)
	DeleteResource(t, "vector-db", activeDB.ID)
}
