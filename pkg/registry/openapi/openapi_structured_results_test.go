package openapi

import (
	"context"
	"fmt"
	"net/http"
	"strings"
	"testing"

	"github.com/digitalocean/godo"
	"github.com/mark3labs/mcp-go/mcp"
	"github.com/stretchr/testify/require"
)

func requireExecuteStructured(t *testing.T, res *mcp.CallToolResult) ExecuteToolResult {
	t.Helper()
	require.False(t, res.IsError)
	require.NotNil(t, res.StructuredContent)
	got, ok := res.StructuredContent.(ExecuteToolResult)
	require.True(t, ok, "expected ExecuteToolResult, got %T", res.StructuredContent)
	return got
}

func TestExecute_structuredContent_jsonBodyAndHeaders(t *testing.T) {
	client := newTestGodoClient(t, func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.Header().Set("Ratelimit-Remaining", "4999")
		w.Header().Set("Link", `<https://api.example.com/v2/droplets?page=2>; rel="next"`)
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{"droplets":[],"meta":{"total":0}}`))
	})

	tool := &OpenAPITool{
		getClient: func(context.Context) (*godo.Client, error) { return client, nil },
		api:       NewOpenAPIClient(),
	}

	req := mcp.CallToolRequest{Params: mcp.CallToolParams{Arguments: map[string]any{
		"OperationID": "droplets_list",
	}}}
	res, err := tool.execute(context.Background(), req)
	require.NoError(t, err)

	got := requireExecuteStructured(t, res)
	require.Equal(t, 200, got.Status)
	require.False(t, got.Truncated)
	require.False(t, got.SelectNotApplied)
	require.Equal(t, "4999", got.ResponseHeaders["ratelimit-remaining"])
	require.Equal(t, `<https://api.example.com/v2/droplets?page=2>; rel="next"`, got.ResponseHeaders["link"])

	body, ok := got.Body.(map[string]any)
	require.True(t, ok, "body type %T", got.Body)
	require.Contains(t, body, "droplets")
	require.Contains(t, body, "meta")
}

func TestExecute_structuredContent_selectProjectsBody(t *testing.T) {
	client := newTestGodoClient(t, func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{"droplets":[{"id":42,"name":"x"},{"id":7,"name":"y"}]}`))
	})

	tool := &OpenAPITool{
		getClient: func(context.Context) (*godo.Client, error) { return client, nil },
		api:       NewOpenAPIClient(),
	}

	req := mcp.CallToolRequest{Params: mcp.CallToolParams{Arguments: map[string]any{
		"OperationID": "droplets_list",
		"Select":      "droplets[*].id",
	}}}
	res, err := tool.execute(context.Background(), req)
	require.NoError(t, err)

	got := requireExecuteStructured(t, res)
	require.False(t, got.SelectNotApplied)

	arr, ok := got.Body.([]any)
	require.True(t, ok, "body type %T", got.Body)
	require.Len(t, arr, 2)
	require.Equal(t, float64(42), arr[0])
	require.Equal(t, float64(7), arr[1])

	txt := res.Content[0].(mcp.TextContent).Text
	require.Contains(t, txt, "42")
	require.Contains(t, txt, "7")
}

func TestExecute_invalidJSON_withSelect_errors(t *testing.T) {
	client := newTestGodoClient(t, func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{"broken":`))
	})

	tool := &OpenAPITool{
		getClient: func(context.Context) (*godo.Client, error) { return client, nil },
		api:       NewOpenAPIClient(),
	}

	req := mcp.CallToolRequest{Params: mcp.CallToolParams{Arguments: map[string]any{
		"OperationID": "droplets_list",
		"Select":      "droplets",
	}}}
	res, err := tool.execute(context.Background(), req)
	require.NoError(t, err)
	require.True(t, res.IsError)
	txt := res.Content[0].(mcp.TextContent).Text
	require.Contains(t, strings.ToLower(txt), "json")
}

func TestSearch_structuredContent_limitClamped(t *testing.T) {
	const nOps = 55
	var b strings.Builder
	b.WriteString(`openapi: 3.0.0
info:
  title: T
  version: "1"
servers:
  - url: https://api.example.com
paths:
`)
	for i := 0; i < nOps; i++ {
		fmt.Fprintf(&b, `  /p%d:
    get:
      operationId: limitprobe_%d
      summary: probe
      responses:
        "200":
          description: OK
`, i, i)
	}

	api := newTestOpenAPIClientYAML(t, b.String())
	tool := &OpenAPITool{
		getClient: func(context.Context) (*godo.Client, error) { return nil, nil },
		api:       api,
	}

	high := mcp.CallToolRequest{Params: mcp.CallToolParams{Arguments: map[string]any{
		"Query": "limitprobe",
		"Limit": float64(999),
	}}}
	resHigh, err := tool.search(context.Background(), high)
	require.NoError(t, err)
	require.False(t, resHigh.IsError)
	payloadHigh, ok := resHigh.StructuredContent.(SearchToolResult)
	require.True(t, ok, "got %T", resHigh.StructuredContent)
	require.Len(t, payloadHigh.Hits, maxSearchLimit)

	low := mcp.CallToolRequest{Params: mcp.CallToolParams{Arguments: map[string]any{
		"Query": "limitprobe",
		"Limit": float64(4),
	}}}
	resLow, err := tool.search(context.Background(), low)
	require.NoError(t, err)
	payloadLow, ok := resLow.StructuredContent.(SearchToolResult)
	require.True(t, ok)
	require.Len(t, payloadLow.Hits, 4)

	// Non-positive Limit in args falls through to defaultSearchLimit in handler.
	defaultLim := mcp.CallToolRequest{Params: mcp.CallToolParams{Arguments: map[string]any{
		"Query": "limitprobe",
		"Limit": float64(0),
	}}}
	resDef, err := tool.search(context.Background(), defaultLim)
	require.NoError(t, err)
	payloadDef, ok := resDef.StructuredContent.(SearchToolResult)
	require.True(t, ok)
	require.Len(t, payloadDef.Hits, defaultSearchLimit)
}

func TestSearch_structuredContent_tagFilter(t *testing.T) {
	const yaml = `openapi: 3.0.0
info:
  title: T
  version: "1"
servers:
  - url: https://api.example.com
paths:
  /a:
    get:
      operationId: tagprobe_alpha
      summary: alpha thing
      tags:
        - Alpha
      responses:
        "200":
          description: OK
  /b:
    get:
      operationId: tagprobe_beta
      summary: beta thing
      tags:
        - Beta
      responses:
        "200":
          description: OK
`
	api := newTestOpenAPIClientYAML(t, yaml)
	tool := &OpenAPITool{
		getClient: func(context.Context) (*godo.Client, error) { return nil, nil },
		api:       api,
	}

	req := mcp.CallToolRequest{Params: mcp.CallToolParams{Arguments: map[string]any{
		"Query": "tagprobe",
		"Tag":   "Alpha",
	}}}
	res, err := tool.search(context.Background(), req)
	require.NoError(t, err)
	payload, ok := res.StructuredContent.(SearchToolResult)
	require.True(t, ok)
	require.Len(t, payload.Hits, 1)
	require.Equal(t, "tagprobe_alpha", payload.Hits[0].OperationID)
}

func TestGetOperation_structuredContent(t *testing.T) {
	const yaml = `openapi: 3.0.0
info:
  title: T
  version: "1"
servers:
  - url: https://api.example.com
paths:
  /v1/x:
    get:
      operationId: thin_op
      summary: Thin
      responses:
        "200":
          description: OK
`
	api := newTestOpenAPIClientYAML(t, yaml)
	tool := &OpenAPITool{
		getClient: func(context.Context) (*godo.Client, error) { return nil, nil },
		api:       api,
	}

	req := mcp.CallToolRequest{Params: mcp.CallToolParams{Arguments: map[string]any{
		"OperationID": "thin_op",
	}}}
	res, err := tool.getOperation(context.Background(), req)
	require.NoError(t, err)
	require.False(t, res.IsError)

	payload, ok := res.StructuredContent.(GetOperationToolResult)
	require.True(t, ok, "got %T", res.StructuredContent)
	require.Equal(t, "thin_op", payload.Operation.OperationID)
	require.Equal(t, "GET", payload.Operation.Method)
	require.Contains(t, res.Content[0].(mcp.TextContent).Text, "Thin")
}
