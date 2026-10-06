package microdroplet

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/digitalocean/godo"
	"github.com/mark3labs/mcp-go/mcp"
	"github.com/stretchr/testify/require"
)

func testGodoClient(t *testing.T, h http.Handler) *godo.Client {
	t.Helper()
	ts := httptest.NewServer(h)
	t.Cleanup(ts.Close)
	c, err := godo.New(http.DefaultClient, godo.SetBaseURL(ts.URL+"/"))
	require.NoError(t, err)
	return c
}

func testTool(t *testing.T, c *godo.Client) *MicroDropletTool {
	t.Helper()
	return NewMicroDropletTool(func(ctx context.Context) (*godo.Client, error) { return c, nil })
}

func TestMicroDropletTool_create(t *testing.T) {
	var gotMethod, gotPath string
	var gotBody []byte
	srv := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotMethod = r.Method
		gotPath = r.URL.Path
		var err error
		gotBody, err = io.ReadAll(r.Body)
		require.NoError(t, err)
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusCreated)
		_, _ = w.Write([]byte(`{"microvm":{"id":"9f1c2d3e-4a5b-6c7d-8e9f-0a1b2c3d4e5f","name":"agent-sandbox-1","state":"creating"}}`))
	})

	tool := testTool(t, testGodoClient(t, srv))
	req := mcp.CallToolRequest{Params: mcp.CallToolParams{Arguments: map[string]any{
		"name":   "agent-sandbox-1",
		"region": "nyc1",
		"size":   map[string]any{"cpu": float64(2), "memory": float64(4096)},
		"source": map[string]any{"oci_ref": "docker.io/library/nginx:1.27"},
		"tags":   []any{"prod"},
	}}}

	resp, err := tool.create(context.Background(), req)
	require.NoError(t, err)
	require.NotNil(t, resp)
	require.False(t, resp.IsError)

	require.Equal(t, http.MethodPost, gotMethod)
	require.Equal(t, "/v2/microvms", gotPath)

	var decoded map[string]any
	require.NoError(t, json.Unmarshal(gotBody, &decoded))
	require.Equal(t, "agent-sandbox-1", decoded["name"])
	require.Equal(t, "nyc1", decoded["region"])
	size, ok := decoded["size"].(map[string]any)
	require.True(t, ok)
	require.Equal(t, float64(2), size["cpu"])
	require.Equal(t, float64(4096), size["memory"])
	source, ok := decoded["source"].(map[string]any)
	require.True(t, ok)
	require.Equal(t, "docker.io/library/nginx:1.27", source["oci_ref"])

	text := resp.Content[0].(mcp.TextContent).Text
	require.Contains(t, text, "9f1c2d3e-4a5b-6c7d-8e9f-0a1b2c3d4e5f")
}

func TestMicroDropletTool_create_validation(t *testing.T) {
	tool := testTool(t, testGodoClient(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		t.Fatal("server should not be called")
	})))

	for _, tc := range []struct {
		name string
		args map[string]any
		want string
	}{
		{
			name: "missing source",
			args: map[string]any{"name": "n", "region": "nyc1"},
			want: "source is required",
		},
		{
			name: "empty source",
			args: map[string]any{"name": "n", "source": map[string]any{}},
			want: "exactly one of oci_ref or checkpoint_id",
		},
		{
			name: "both sources",
			args: map[string]any{
				"name": "n",
				"source": map[string]any{
					"oci_ref":       "docker.io/library/nginx:1.27",
					"checkpoint_id": "0a1b2c3d-4e5f-6a7b-8c9d-0e1f2a3b4c5d",
				},
			},
			want: "not both",
		},
		{
			name: "oci_ref missing region",
			args: map[string]any{
				"name":   "n",
				"size":   map[string]any{"cpu": float64(2), "memory": float64(4096)},
				"source": map[string]any{"oci_ref": "docker.io/library/nginx:1.27"},
			},
			want: "region is required",
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			req := mcp.CallToolRequest{Params: mcp.CallToolParams{Arguments: tc.args}}
			resp, err := tool.create(context.Background(), req)
			require.NoError(t, err)
			require.True(t, resp.IsError)
			require.Contains(t, resp.Content[0].(mcp.TextContent).Text, tc.want)
		})
	}
}

func TestMicroDropletTool_create_checkpoint(t *testing.T) {
	var gotBody []byte
	srv := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		require.Equal(t, http.MethodPost, r.Method)
		require.Equal(t, "/v2/microvms", r.URL.Path)
		var err error
		gotBody, err = io.ReadAll(r.Body)
		require.NoError(t, err)
		w.WriteHeader(http.StatusCreated)
		_, _ = w.Write([]byte(`{"microvm":{"id":"9f1c2d3e-4a5b-6c7d-8e9f-0a1b2c3d4e5f","state":"running"}}`))
	})

	tool := testTool(t, testGodoClient(t, srv))
	req := mcp.CallToolRequest{Params: mcp.CallToolParams{Arguments: map[string]any{
		"name":   "from-chk",
		"source": map[string]any{"checkpoint_id": "0a1b2c3d-4e5f-6a7b-8c9d-0e1f2a3b4c5d"},
	}}}

	resp, err := tool.create(context.Background(), req)
	require.NoError(t, err)
	require.False(t, resp.IsError)

	var decoded map[string]any
	require.NoError(t, json.Unmarshal(gotBody, &decoded))
	source, ok := decoded["source"].(map[string]any)
	require.True(t, ok)
	require.Equal(t, "0a1b2c3d-4e5f-6a7b-8c9d-0e1f2a3b4c5d", source["checkpoint_id"])
	_, hasRegion := decoded["region"]
	require.False(t, hasRegion)
}

func TestMicroDropletTool_get_notFound(t *testing.T) {
	srv := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusNotFound)
		_, _ = w.Write([]byte(`{"id":"not_found","message":"The resource you requested could not be found."}`))
	})

	tool := testTool(t, testGodoClient(t, srv))
	req := mcp.CallToolRequest{Params: mcp.CallToolParams{Arguments: map[string]any{
		"id": "9f1c2d3e-4a5b-6c7d-8e9f-0a1b2c3d4e5f",
	}}}

	resp, err := tool.get(context.Background(), req)
	require.NoError(t, err)
	require.True(t, resp.IsError)
}

func TestMicroDropletTool_list(t *testing.T) {
	var gotQuery string
	srv := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		require.Equal(t, http.MethodGet, r.Method)
		require.Equal(t, "/v2/microvms", r.URL.Path)
		gotQuery = r.URL.RawQuery
		_, _ = w.Write([]byte(`{"microvms":[],"links":{},"meta":{"total":0}}`))
	})

	tool := testTool(t, testGodoClient(t, srv))
	req := mcp.CallToolRequest{Params: mcp.CallToolParams{Arguments: map[string]any{
		"page":     float64(2),
		"per_page": float64(50),
		"region":   "nyc1",
	}}}

	resp, err := tool.list(context.Background(), req)
	require.NoError(t, err)
	require.False(t, resp.IsError)
	require.Contains(t, gotQuery, "page=2")
	require.Contains(t, gotQuery, "per_page=50")
	require.Contains(t, gotQuery, "region=nyc1")
}

func TestMicroDropletTool_delete(t *testing.T) {
	var gotPath string
	srv := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		require.Equal(t, http.MethodDelete, r.Method)
		gotPath = r.URL.Path
		w.WriteHeader(http.StatusNoContent)
	})

	tool := testTool(t, testGodoClient(t, srv))
	id := "9f1c2d3e-4a5b-6c7d-8e9f-0a1b2c3d4e5f"
	req := mcp.CallToolRequest{Params: mcp.CallToolParams{Arguments: map[string]any{"id": id}}}

	resp, err := tool.delete(context.Background(), req)
	require.NoError(t, err)
	require.False(t, resp.IsError)
	require.Equal(t, "/v2/microvms/"+id, gotPath)
	require.Contains(t, resp.Content[0].(mcp.TextContent).Text, "deleted successfully")
}

func TestMicroDropletTool_checkpointCreate(t *testing.T) {
	var gotPath string
	var gotBody map[string]any
	srv := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		require.Equal(t, http.MethodPost, r.Method)
		gotPath = r.URL.Path
		require.NoError(t, json.NewDecoder(r.Body).Decode(&gotBody))
		_, _ = w.Write([]byte(`{"checkpoint":{"id":"0a1b2c3d-4e5f-6a7b-8c9d-0e1f2a3b4c5d","status":"CHECKPOINT_CREATING"}}`))
	})

	tool := testTool(t, testGodoClient(t, srv))
	mdID := "9f1c2d3e-4a5b-6c7d-8e9f-0a1b2c3d4e5f"
	req := mcp.CallToolRequest{Params: mcp.CallToolParams{Arguments: map[string]any{
		"id":   mdID,
		"name": "chk-1",
	}}}

	resp, err := tool.checkpointCreate(context.Background(), req)
	require.NoError(t, err)
	require.False(t, resp.IsError)
	require.Equal(t, "/v2/microvms/"+mdID+"/checkpoints", gotPath)
	require.Equal(t, "chk-1", gotBody["name"])
}

func TestMicroDropletTool_checkpointCreate_omittedName(t *testing.T) {
	var gotPath string
	var gotBody []byte
	srv := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		require.Equal(t, http.MethodPost, r.Method)
		gotPath = r.URL.Path
		var err error
		gotBody, err = io.ReadAll(r.Body)
		require.NoError(t, err)
		w.WriteHeader(http.StatusCreated)
		_, _ = w.Write([]byte(`{"checkpoint":{"id":"0a1b2c3d-4e5f-6a7b-8c9d-0e1f2a3b4c5d","status":"CHECKPOINT_CREATING"}}`))
	})

	tool := testTool(t, testGodoClient(t, srv))
	mdID := "9f1c2d3e-4a5b-6c7d-8e9f-0a1b2c3d4e5f"
	req := mcp.CallToolRequest{Params: mcp.CallToolParams{Arguments: map[string]any{
		"id": mdID,
	}}}

	resp, err := tool.checkpointCreate(context.Background(), req)
	require.NoError(t, err)
	require.False(t, resp.IsError)
	require.Equal(t, "/v2/microvms/"+mdID+"/checkpoints", gotPath)
	require.Empty(t, gotBody)
}

func TestMicroDropletTool_checkpointList_filter(t *testing.T) {
	var gotQuery string
	srv := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		require.Equal(t, "/v2/microvms/checkpoints", r.URL.Path)
		gotQuery = r.URL.RawQuery
		_, _ = w.Write([]byte(`{"checkpoints":[],"links":{},"meta":{"total":0}}`))
	})

	tool := testTool(t, testGodoClient(t, srv))
	mdID := "9f1c2d3e-4a5b-6c7d-8e9f-0a1b2c3d4e5f"
	req := mcp.CallToolRequest{Params: mcp.CallToolParams{Arguments: map[string]any{
		"microvm_id": mdID,
	}}}

	resp, err := tool.checkpointList(context.Background(), req)
	require.NoError(t, err)
	require.False(t, resp.IsError)
	require.Contains(t, gotQuery, "microvm_id="+mdID)
}
