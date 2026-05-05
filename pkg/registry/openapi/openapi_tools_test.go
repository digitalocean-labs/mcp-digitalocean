package openapi

import (
	"context"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"

	"github.com/digitalocean/godo"
	"github.com/mark3labs/mcp-go/mcp"
	"github.com/stretchr/testify/require"
	"golang.org/x/oauth2"
)

func TestExecute_DELETE_redirectsToDeleteTool(t *testing.T) {
	tool, err := NewOpenAPITool(func(context.Context) (*godo.Client, error) {
		t.Fatal("client must not be resolved when DELETE is routed to openapi-execute-delete")
		return nil, nil
	}, Options{})
	require.NoError(t, err)

	req := mcp.CallToolRequest{Params: mcp.CallToolParams{Arguments: map[string]interface{}{
		"OperationID": "droplets_destroy",
		"Parameters": map[string]interface{}{
			"droplet_id": float64(1),
		},
	}}}
	res, err := tool.execute(context.Background(), req)
	require.NoError(t, err)
	require.True(t, res.IsError)
	txt := res.Content[0].(mcp.TextContent).Text
	require.Contains(t, strings.ToLower(txt), "openapi-execute-delete")
}

func TestTools_excludesDeleteWhenDisabled(t *testing.T) {
	tool, err := NewOpenAPITool(func(context.Context) (*godo.Client, error) {
		return nil, nil
	}, Options{DisableDeletes: true})
	require.NoError(t, err)

	tools := tool.Tools()
	require.Len(t, tools, 3)
	for _, st := range tools {
		require.NotEqual(t, "openapi-execute-delete", st.Tool.Name)
	}
}

func TestTools_includesDeleteByDefault(t *testing.T) {
	tool, err := NewOpenAPITool(func(context.Context) (*godo.Client, error) {
		return nil, nil
	}, Options{})
	require.NoError(t, err)

	tools := tool.Tools()
	var deleteTool *mcp.Tool
	for i := range tools {
		if tools[i].Tool.Name == "openapi-execute-delete" {
			deleteTool = &tools[i].Tool
			break
		}
	}
	require.NotNil(t, deleteTool, "openapi-execute-delete should be registered")
	require.NotNil(t, deleteTool.Annotations.DestructiveHint)
	require.True(t, *deleteTool.Annotations.DestructiveHint)
}

func TestExecuteDelete_rejectsNonDelete(t *testing.T) {
	tool, err := NewOpenAPITool(func(context.Context) (*godo.Client, error) {
		t.Fatal("client must not be resolved for non-DELETE on openapi-execute-delete")
		return nil, nil
	}, Options{})
	require.NoError(t, err)

	req := mcp.CallToolRequest{Params: mcp.CallToolParams{Arguments: map[string]interface{}{
		"OperationID": "droplets_list",
	}}}
	res, err := tool.executeDelete(context.Background(), req)
	require.NoError(t, err)
	require.True(t, res.IsError)
	txt := res.Content[0].(mcp.TextContent).Text
	require.Contains(t, strings.ToLower(txt), "delete")
	require.Contains(t, txt, "openapi-execute")
}

func TestSearch_requiresQuery(t *testing.T) {
	tool, err := NewOpenAPITool(func(context.Context) (*godo.Client, error) {
		return nil, nil
	}, Options{})
	require.NoError(t, err)

	req := mcp.CallToolRequest{Params: mcp.CallToolParams{Arguments: map[string]interface{}{}}}
	res, err := tool.search(context.Background(), req)
	require.NoError(t, err)
	require.True(t, res.IsError)
}

func TestExecute_GET_droplets_list_smoke(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/v2/droplets" {
			http.NotFound(w, r)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{"droplets":[]}`))
	}))
	defer srv.Close()

	base, err := url.Parse(srv.URL)
	require.NoError(t, err)

	ts := oauth2.StaticTokenSource(&oauth2.Token{AccessToken: "test-token"})
	oauthCli := oauth2.NewClient(context.Background(), ts)
	client, err := godo.New(oauthCli, godo.SetBaseURL(base.String()))
	require.NoError(t, err)

	tool, err := NewOpenAPITool(func(context.Context) (*godo.Client, error) {
		return client, nil
	}, Options{})
	require.NoError(t, err)

	req := mcp.CallToolRequest{Params: mcp.CallToolParams{Arguments: map[string]interface{}{
		"OperationID": "droplets_list",
	}}}
	res, err := tool.execute(context.Background(), req)
	require.NoError(t, err)
	require.False(t, res.IsError)
	txt := res.Content[0].(mcp.TextContent).Text
	require.Contains(t, txt, "Status: 200")
	require.Contains(t, txt, `"droplets"`)
}

func TestExecuteDelete_DELETE_droplets_destroy_smoke(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodDelete || r.URL.Path != "/v2/droplets/42" {
			http.NotFound(w, r)
			return
		}
		w.WriteHeader(http.StatusNoContent)
	}))
	defer srv.Close()

	base, err := url.Parse(srv.URL)
	require.NoError(t, err)

	ts := oauth2.StaticTokenSource(&oauth2.Token{AccessToken: "test-token"})
	oauthCli := oauth2.NewClient(context.Background(), ts)
	client, err := godo.New(oauthCli, godo.SetBaseURL(base.String()))
	require.NoError(t, err)

	tool, err := NewOpenAPITool(func(context.Context) (*godo.Client, error) {
		return client, nil
	}, Options{})
	require.NoError(t, err)

	req := mcp.CallToolRequest{Params: mcp.CallToolParams{Arguments: map[string]interface{}{
		"OperationID": "droplets_destroy",
		"Parameters": map[string]interface{}{
			"droplet_id": float64(42),
		},
	}}}
	res, err := tool.executeDelete(context.Background(), req)
	require.NoError(t, err)
	require.False(t, res.IsError)
	txt := res.Content[0].(mcp.TextContent).Text
	require.Contains(t, txt, "Status: 204")
}
