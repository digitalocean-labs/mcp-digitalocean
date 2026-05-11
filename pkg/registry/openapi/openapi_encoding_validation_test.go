package openapi

import (
	"context"
	"encoding/json"
	"net/http"
	"strings"
	"testing"

	"github.com/digitalocean/godo"
	"github.com/mark3labs/mcp-go/mcp"
	"github.com/stretchr/testify/require"
)

func TestExecute_parameterEncoding_table(t *testing.T) {
	t.Parallel()
	const spec = `openapi: 3.0.0
info:
  title: T
  version: "1"
servers:
  - url: https://api.example.com
paths:
  /v1/{region}/{name}:
    get:
      operationId: regions_get
      summary: Get
      parameters:
        - name: region
          in: path
          required: true
          schema:
            type: string
        - name: name
          in: path
          required: true
          schema:
            type: string
        - name: tags
          in: query
          schema:
            type: array
            items:
              type: string
        - name: X-Multi
          in: header
          required: false
          schema:
            type: string
      responses:
        "200":
          description: OK
`

	echo := newTestGodoClient(t, func(w http.ResponseWriter, r *http.Request) {
		type echoOut struct {
			Path    string              `json:"path"`
			Query   map[string][]string `json:"query"`
			Headers map[string][]string `json:"headers"`
		}
		hdr := make(map[string][]string)
		for k, v := range r.Header {
			hdr[k] = v
		}
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		_ = json.NewEncoder(w).Encode(echoOut{
			Path:    r.URL.Path,
			Query:   r.URL.Query(),
			Headers: hdr,
		})
	})

	tool := &OpenAPITool{
		getClient: func(context.Context) (*godo.Client, error) {
			return echo, nil
		},
		api: newTestOpenAPIClientYAML(t, spec),
	}

	req := mcp.CallToolRequest{Params: mcp.CallToolParams{Arguments: map[string]any{
		"OperationID": "regions_get",
		"Parameters": map[string]any{
			"region":  "nyc3",
			"name":    "api",
			"tags":    []any{"a", "b"},
			"X-Multi": []any{"one", "two"},
		},
	}}}
	res, err := tool.execute(context.Background(), req)
	require.NoError(t, err)
	require.False(t, res.IsError)
	txt := res.Content[0].(mcp.TextContent).Text
	require.Contains(t, txt, `"path":"/v1/nyc3/api"`)
	require.Contains(t, txt, `"tags":["a","b"]`)
	require.Contains(t, txt, `"X-Multi":["one","two"]`)
}

func TestExecute_validation_missingRequiredQuery_hint(t *testing.T) {
	t.Parallel()
	const spec = `openapi: 3.0.0
info:
  title: T
  version: "1"
servers:
  - url: https://api.example.com
paths:
  /v2/things/{thing_id}:
    get:
      operationId: things_get
      parameters:
        - name: thing_id
          in: path
          required: true
          schema:
            type: integer
        - name: force
          in: query
          required: true
          schema:
            type: boolean
      responses:
        "200":
          description: OK
`
	echo := newTestGodoClient(t, func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{}`))
	})

	tool := &OpenAPITool{
		getClient: func(context.Context) (*godo.Client, error) { return echo, nil },
		api:       newTestOpenAPIClientYAML(t, spec),
	}

	req := mcp.CallToolRequest{Params: mcp.CallToolParams{Arguments: map[string]any{
		"OperationID": "things_get",
		"Parameters": map[string]any{
			"thing_id": float64(1),
		},
	}}}
	res, err := tool.execute(context.Background(), req)
	require.NoError(t, err)
	require.True(t, res.IsError)
	txt := res.Content[0].(mcp.TextContent).Text
	require.Contains(t, strings.ToLower(txt), "openapi-get-operation")
	require.Contains(t, strings.ToLower(txt), "force")
}

func TestExecute_validation_wrongBodyType(t *testing.T) {
	t.Parallel()
	const spec = `openapi: 3.0.0
info:
  title: T
  version: "1"
servers:
  - url: https://api.example.com
paths:
  /v2/things:
    post:
      operationId: things_create
      requestBody:
        required: true
        content:
          application/json:
            schema:
              type: object
              required:
                - title
              properties:
                title:
                  type: string
      responses:
        "200":
          description: OK
`
	echo := newTestGodoClient(t, func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{}`))
	})

	tool := &OpenAPITool{
		getClient: func(context.Context) (*godo.Client, error) { return echo, nil },
		api:       newTestOpenAPIClientYAML(t, spec),
	}

	req := mcp.CallToolRequest{Params: mcp.CallToolParams{Arguments: map[string]any{
		"OperationID": "things_create",
		"Body": map[string]any{
			"title": float64(42),
		},
	}}}
	res, err := tool.execute(context.Background(), req)
	require.NoError(t, err)
	require.True(t, res.IsError)
	txt := res.Content[0].(mcp.TextContent).Text
	require.Contains(t, strings.ToLower(txt), "openapi-get-operation")
}
