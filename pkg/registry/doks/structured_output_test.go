package doks

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/digitalocean/godo"
	"github.com/mark3labs/mcp-go/mcp"
	"github.com/mark3labs/mcp-go/server"
	"github.com/stretchr/testify/require"
)

// TestStructuredOutputSatisfiesDeclaredSchema drives a migrated tool through a
// real MCPServer with WithOutputSchemaValidation enabled, so the server itself
// checks the emitted structuredContent against the outputSchema the tool
// advertises. Unit tests on the handler cannot catch a schema that the payload
// does not satisfy; only the server (or a spec-compliant client) validates.
//
// Validation is enabled only here, not in cmd/mcp-digitalocean: in production
// a schema mismatch should degrade to a slightly-wrong schema, not a failed
// tool call. This test is what keeps the mismatch from reaching production.
//
// doks-get-credentials is the tool exercised because it is the one that
// projects the API resource onto a hand-written shape, so it is where the
// schema and the payload are most likely to drift apart.
func TestStructuredOutputSatisfiesDeclaredSchema(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		require.Equal(t, "/v2/kubernetes/clusters/cluster-1/credentials", r.URL.Path)
		w.Header().Set("Content-Type", "application/json")
		require.NoError(t, json.NewEncoder(w).Encode(map[string]any{
			"server":                     "https://cluster-1.k8s.ondigitalocean.com",
			"certificate_authority_data": base64.StdEncoding.EncodeToString([]byte("ca-data")),
			"client_certificate_data":    base64.StdEncoding.EncodeToString([]byte("cert-data")),
			"client_key_data":            base64.StdEncoding.EncodeToString([]byte("key-data")),
			"token":                      "dop_v1_token",
			"expires_at":                 "2024-01-01T00:00:00Z",
		}))
	}))
	defer ts.Close()

	clientFn := func(context.Context) (*godo.Client, error) {
		return godo.New(ts.Client(), godo.SetBaseURL(ts.URL))
	}

	svr := server.NewMCPServer("test", "test", server.WithOutputSchemaValidation())
	svr.AddTools(NewDoksTool(clientFn).Tools()...)

	ctx := context.Background()
	svr.HandleMessage(ctx, []byte(`{"jsonrpc":"2.0","id":1,"method":"initialize",`+
		`"params":{"protocolVersion":"2025-06-18","capabilities":{},"clientInfo":{"name":"t","version":"1"}}}`))

	raw, err := json.Marshal(svr.HandleMessage(ctx,
		[]byte(`{"jsonrpc":"2.0","id":2,"method":"tools/call","params":{"name":"doks-get-credentials",`+
			`"arguments":{"ClusterID":"cluster-1"}}}`)))
	require.NoError(t, err)

	var resp struct {
		Error  *struct{ Message string } `json:"error"`
		Result struct {
			IsError           bool               `json:"isError"`
			Content           []mcp.TextContent  `json:"content"`
			StructuredContent clusterCredentials `json:"structuredContent"`
		} `json:"result"`
	}
	require.NoError(t, json.Unmarshal(raw, &resp))

	require.Nil(t, resp.Error, "server rejected the call")
	require.False(t, resp.Result.IsError, "tool returned an error result: %s", raw)

	// The payload is already an object, so it is published without an
	// envelope: structuredContent is the payload itself.
	require.Equal(t, "https://cluster-1.k8s.ondigitalocean.com", resp.Result.StructuredContent.Server)
	require.Equal(t, "ca-data", resp.Result.StructuredContent.CertificateAuthorityData)
	require.Equal(t, "dop_v1_token", resp.Result.StructuredContent.Token)

	// The text block is retained for backwards compatibility and carries the
	// same payload.
	require.Len(t, resp.Result.Content, 1)
	var fromText clusterCredentials
	require.NoError(t, json.Unmarshal([]byte(resp.Result.Content[0].Text), &fromText))
	require.Equal(t, resp.Result.StructuredContent, fromText)
}
