package genaiinferencerouter

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
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

func TestRouterTool_create(t *testing.T) {
	var gotMethod, gotPath string
	var gotBody []byte
	srv := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotMethod = r.Method
		gotPath = r.URL.Path
		var err error
		gotBody, err = io.ReadAll(r.Body)
		require.NoError(t, err)
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"model_router":{"uuid":"11f117f7-d076-6272-b542-ca68c578b04b","name":"alex-test-2"}}`))
	})

	c := testGodoClient(t, srv)
	tool := NewRouterTool(func(ctx context.Context) (*godo.Client, error) { return c, nil })

	policies := `[{"model":"openai-gpt-5","usecase_class":"MODEL_ROUTER_USECASE_CLASS_CODE_GENERATION"},{"model":"llama3.3-70b-instruct","usecase_class":"MODEL_ROUTER_USECASE_CLASS_CHATBOT"}]`
	req := mcp.CallToolRequest{Params: mcp.CallToolParams{Arguments: map[string]any{
		"Name":           "alex-test-2",
		"PoliciesJson":   policies,
		"FallbackModels": []any{"llama3.3-70b-instruct"},
	}}}
	resp, err := tool.create(context.Background(), req)
	require.NoError(t, err)
	require.NotNil(t, resp)
	require.False(t, resp.IsError)

	require.Equal(t, http.MethodPost, gotMethod)
	require.Equal(t, "/v2/gen-ai/models/routers", gotPath)

	var decoded godo.InferenceRouterCreateRequest
	require.NoError(t, json.Unmarshal(gotBody, &decoded))
	require.Equal(t, "alex-test-2", decoded.Name)
	require.NotNil(t, decoded.Policies)
	var policyObjs []map[string]any
	require.NoError(t, json.Unmarshal(decoded.Policies, &policyObjs))
	require.Len(t, policyObjs, 2)
	require.Equal(t, "openai-gpt-5", policyObjs[0]["model"])
	require.Equal(t, "MODEL_ROUTER_USECASE_CLASS_CODE_GENERATION", policyObjs[0]["usecase_class"])
	require.Equal(t, []string{"llama3.3-70b-instruct"}, decoded.FallbackModels)

	text := resp.Content[0].(mcp.TextContent).Text
	require.Contains(t, text, "11f117f7-d076-6272-b542-ca68c578b04b")
}

func TestRouterTool_create_validation(t *testing.T) {
	c := testGodoClient(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		t.Fatal("server should not be called")
	}))
	tool := NewRouterTool(func(ctx context.Context) (*godo.Client, error) { return c, nil })

	req := mcp.CallToolRequest{Params: mcp.CallToolParams{Arguments: map[string]any{
		"Name": "n",
	}}}
	resp, err := tool.create(context.Background(), req)
	require.NoError(t, err)
	require.True(t, resp.IsError)
}

func TestRouterTool_create_policiesJson_validation(t *testing.T) {
	c := testGodoClient(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		t.Fatal("server should not be called")
	}))
	tool := NewRouterTool(func(ctx context.Context) (*godo.Client, error) { return c, nil })

	for _, tc := range []struct {
		name string
		args map[string]any
	}{
		{
			name: "not an array",
			args: map[string]any{"Name": "n", "PoliciesJson": `{"model":"x"}`, "FallbackModels": []any{"m"}},
		},
		{
			name: "invalid json",
			args: map[string]any{"Name": "n", "PoliciesJson": `[`, "FallbackModels": []any{"m"}},
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			req := mcp.CallToolRequest{Params: mcp.CallToolParams{Arguments: tc.args}}
			resp, err := tool.create(context.Background(), req)
			require.NoError(t, err)
			require.True(t, resp.IsError, "expected error for %s", tc.name)
		})
	}
}

func TestRouterTool_create_requiresFallbackModels(t *testing.T) {
	c := testGodoClient(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		t.Fatal("server should not be called")
	}))
	tool := NewRouterTool(func(ctx context.Context) (*godo.Client, error) { return c, nil })

	req := mcp.CallToolRequest{Params: mcp.CallToolParams{Arguments: map[string]any{
		"Name": "n", "FallbackModels": []any{},
	}}}
	resp, err := tool.create(context.Background(), req)
	require.NoError(t, err)
	require.True(t, resp.IsError)
}

func TestRouterTool_create_omitsPoliciesWhenEmpty(t *testing.T) {
	var gotBody []byte
	srv := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var err error
		gotBody, err = io.ReadAll(r.Body)
		require.NoError(t, err)
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"model_router":{"uuid":"u1","name":"no-policies"}}`))
	})
	c := testGodoClient(t, srv)
	tool := NewRouterTool(func(ctx context.Context) (*godo.Client, error) { return c, nil })

	req := mcp.CallToolRequest{Params: mcp.CallToolParams{Arguments: map[string]any{
		"Name": "no-policies", "FallbackModels": []any{"openai-gpt-oss-120b"},
	}}}
	resp, err := tool.create(context.Background(), req)
	require.NoError(t, err)
	require.False(t, resp.IsError)

	var decoded map[string]json.RawMessage
	require.NoError(t, json.Unmarshal(gotBody, &decoded))
	_, hasPolicies := decoded["policies"]
	require.False(t, hasPolicies)
	var fbs []string
	require.NoError(t, json.Unmarshal(decoded["fallback_models"], &fbs))
	require.Equal(t, []string{"openai-gpt-oss-120b"}, fbs)
}

func TestRouterTool_list(t *testing.T) {
	var gotQuery string
	srv := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			http.Error(w, "method", http.StatusMethodNotAllowed)
			return
		}
		gotQuery = r.URL.RawQuery
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"model_routers":[],"meta":{"total":0,"page":1,"pages":1}}`))
	})
	c := testGodoClient(t, srv)
	tool := NewRouterTool(func(ctx context.Context) (*godo.Client, error) { return c, nil })

	req := mcp.CallToolRequest{Params: mcp.CallToolParams{Arguments: map[string]any{
		"Page":    float64(2),
		"PerPage": float64(50),
	}}}
	resp, err := tool.list(context.Background(), req)
	require.NoError(t, err)
	require.False(t, resp.IsError)
	require.Contains(t, gotQuery, "page=2")
	require.Contains(t, gotQuery, "per_page=50")
	require.Contains(t, resp.Content[0].(mcp.TextContent).Text, "model_routers")
}

func TestRouterTool_get(t *testing.T) {
	var gotPath string
	srv := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotPath = r.URL.Path
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"model_router":{"uuid":"u1","name":"n"}}`))
	})
	c := testGodoClient(t, srv)
	tool := NewRouterTool(func(ctx context.Context) (*godo.Client, error) { return c, nil })

	req := mcp.CallToolRequest{Params: mcp.CallToolParams{Arguments: map[string]any{
		"UUID": "11f117f7-d076-6272-b542-ca68c578b04b",
	}}}
	resp, err := tool.get(context.Background(), req)
	require.NoError(t, err)
	require.False(t, resp.IsError)
	require.True(t, strings.HasSuffix(gotPath, "/11f117f7-d076-6272-b542-ca68c578b04b"))
	require.Contains(t, resp.Content[0].(mcp.TextContent).Text, "model_router")
}

func TestRouterTool_get_missingUUID(t *testing.T) {
	c := testGodoClient(t, http.NotFoundHandler())
	tool := NewRouterTool(func(ctx context.Context) (*godo.Client, error) { return c, nil })
	resp, err := tool.get(context.Background(), mcp.CallToolRequest{Params: mcp.CallToolParams{Arguments: map[string]any{}}})
	require.NoError(t, err)
	require.True(t, resp.IsError)
}

func TestRouterTool_delete(t *testing.T) {
	var gotMethod, gotPath string
	srv := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotMethod = r.Method
		gotPath = r.URL.Path
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"uuid":"11f117f7-d076-6272-b542-ca68c578b04b"}`))
	})
	c := testGodoClient(t, srv)
	tool := NewRouterTool(func(ctx context.Context) (*godo.Client, error) { return c, nil })

	req := mcp.CallToolRequest{Params: mcp.CallToolParams{Arguments: map[string]any{
		"UUID": "11f117f7-d076-6272-b542-ca68c578b04b",
	}}}
	resp, err := tool.delete(context.Background(), req)
	require.NoError(t, err)
	require.False(t, resp.IsError)
	require.Equal(t, http.MethodDelete, gotMethod)
	require.True(t, strings.HasSuffix(gotPath, "/11f117f7-d076-6272-b542-ca68c578b04b"))
	require.Contains(t, resp.Content[0].(mcp.TextContent).Text, "11f117f7-d076-6272-b542-ca68c578b04b")
}

// godo's JSON decoder requires a value on 2xx; an empty body fails decode. The API typically returns {} or {"uuid":"..."}.
func TestRouterTool_delete_minimalJSONBody(t *testing.T) {
	srv := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{}`))
	})
	c := testGodoClient(t, srv)
	tool := NewRouterTool(func(ctx context.Context) (*godo.Client, error) { return c, nil })

	req := mcp.CallToolRequest{Params: mcp.CallToolParams{Arguments: map[string]any{
		"UUID": "11f117f7-d076-6272-b542-ca68c578b04b",
	}}}
	resp, err := tool.delete(context.Background(), req)
	require.NoError(t, err)
	require.False(t, resp.IsError)
	require.Contains(t, resp.Content[0].(mcp.TextContent).Text, "11f117f7-d076-6272-b542-ca68c578b04b")
}

func TestRouterTool_delete_missingUUID(t *testing.T) {
	c := testGodoClient(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		t.Fatal("server should not be called")
	}))
	tool := NewRouterTool(func(ctx context.Context) (*godo.Client, error) { return c, nil })
	resp, err := tool.delete(context.Background(), mcp.CallToolRequest{Params: mcp.CallToolParams{Arguments: map[string]any{}}})
	require.NoError(t, err)
	require.True(t, resp.IsError)
}

func TestRouterTool_listTaskPresets(t *testing.T) {
	var gotPath, gotQuery string
	srv := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			http.Error(w, "method", http.StatusMethodNotAllowed)
			return
		}
		gotPath = r.URL.Path
		gotQuery = r.URL.RawQuery
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"tasks":[{"task_slug":"translation","name":"Translation"}],"meta":{"total":1,"page":1,"pages":1}}`))
	})
	c := testGodoClient(t, srv)
	tool := NewRouterTool(func(ctx context.Context) (*godo.Client, error) { return c, nil })

	req := mcp.CallToolRequest{Params: mcp.CallToolParams{Arguments: map[string]any{
		"Page": float64(1), "PerPage": float64(100),
	}}}
	resp, err := tool.listTaskPresets(context.Background(), req)
	require.NoError(t, err)
	require.False(t, resp.IsError)
	require.Equal(t, "/v2/gen-ai/models/routers/tasks/presets", gotPath)
	require.Contains(t, gotQuery, "page=1")
	require.Contains(t, gotQuery, "per_page=100")
	require.Contains(t, resp.Content[0].(mcp.TextContent).Text, `"tasks"`)
	require.Contains(t, resp.Content[0].(mcp.TextContent).Text, "translation")
}

func TestRouterTool_update(t *testing.T) {
	var gotMethod, gotPath string
	var gotBody []byte
	srv := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotMethod = r.Method
		gotPath = r.URL.Path
		var err error
		gotBody, err = io.ReadAll(r.Body)
		require.NoError(t, err)
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"model_router":{"uuid":"u1","name":"renamed"}}`))
	})
	c := testGodoClient(t, srv)
	tool := NewRouterTool(func(ctx context.Context) (*godo.Client, error) { return c, nil })

	req := mcp.CallToolRequest{Params: mcp.CallToolParams{Arguments: map[string]any{
		"UUID": "11f117f7-d076-6272-b542-ca68c578b04b",
		"Name": "renamed",
	}}}
	resp, err := tool.update(context.Background(), req)
	require.NoError(t, err)
	require.False(t, resp.IsError)
	require.Equal(t, http.MethodPut, gotMethod)
	require.True(t, strings.HasSuffix(gotPath, "/11f117f7-d076-6272-b542-ca68c578b04b"))
	var body map[string]any
	require.NoError(t, json.Unmarshal(gotBody, &body))
	require.Equal(t, "renamed", body["name"])
	require.Contains(t, resp.Content[0].(mcp.TextContent).Text, "renamed")
}

func TestRouterTool_update_validation(t *testing.T) {
	c := testGodoClient(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		t.Fatal("server should not be called")
	}))
	tool := NewRouterTool(func(ctx context.Context) (*godo.Client, error) { return c, nil })

	for _, tc := range []struct {
		name string
		args map[string]any
	}{
		{name: "missing uuid", args: map[string]any{"Name": "x"}},
		{name: "no fields", args: map[string]any{"UUID": "11f117f7-d076-6272-b542-ca68c578b04b"}},
		{name: "invalid policies", args: map[string]any{"UUID": "11f117f7-d076-6272-b542-ca68c578b04b", "PoliciesJson": `{`}},
	} {
		t.Run(tc.name, func(t *testing.T) {
			resp, err := tool.update(context.Background(), mcp.CallToolRequest{Params: mcp.CallToolParams{Arguments: tc.args}})
			require.NoError(t, err)
			require.True(t, resp.IsError, tc.name)
		})
	}
}

func TestRouterTool_Tools(t *testing.T) {
	tool := NewRouterTool(func(ctx context.Context) (*godo.Client, error) { return nil, nil })
	names := map[string]bool{}
	for _, st := range tool.Tools() {
		names[st.Tool.Name] = true
	}
	require.True(t, names["genai-inference-router-create"])
	require.True(t, names["genai-inference-router-list"])
	require.True(t, names["genai-inference-router-get"])
	require.True(t, names["genai-inference-router-delete"])
	require.True(t, names["genai-inference-router-task-presets"])
	require.True(t, names["genai-inference-router-update"])
}
