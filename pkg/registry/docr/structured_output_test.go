package docr

import (
	"context"
	"encoding/json"
	"testing"
	"time"

	"github.com/digitalocean/godo"
	"github.com/mark3labs/mcp-go/mcp"
	"github.com/mark3labs/mcp-go/server"
	"github.com/stretchr/testify/require"
	"go.uber.org/mock/gomock"
)

// callTool drives one tools/call through a real MCPServer with
// WithOutputSchemaValidation enabled, so the server checks the emitted
// structuredContent against the outputSchema the tool advertises. Only the
// server (or a spec-compliant client) performs that check — handler unit tests
// cannot catch a schema the payload does not satisfy.
//
// Validation is enabled only in tests, not in cmd/mcp-digitalocean, so that a
// schema mismatch in production degrades to a wrong schema rather than a
// failed tool call.
func callTool(t *testing.T, tools []server.ServerTool, name string, args map[string]any) (json.RawMessage, string) {
	t.Helper()

	svr := server.NewMCPServer("test", "test", server.WithOutputSchemaValidation())
	svr.AddTools(tools...)

	ctx := context.Background()
	svr.HandleMessage(ctx, []byte(`{"jsonrpc":"2.0","id":1,"method":"initialize",`+
		`"params":{"protocolVersion":"2025-06-18","capabilities":{},"clientInfo":{"name":"t","version":"1"}}}`))

	params, err := json.Marshal(map[string]any{"name": name, "arguments": args})
	require.NoError(t, err)
	req, err := json.Marshal(map[string]any{
		"jsonrpc": "2.0", "id": 2, "method": "tools/call", "params": json.RawMessage(params),
	})
	require.NoError(t, err)

	raw, err := json.Marshal(svr.HandleMessage(ctx, req))
	require.NoError(t, err)

	var resp struct {
		Error  *struct{ Message string } `json:"error"`
		Result struct {
			IsError           bool              `json:"isError"`
			Content           []mcp.TextContent `json:"content"`
			StructuredContent json.RawMessage   `json:"structuredContent"`
		} `json:"result"`
	}
	require.NoError(t, json.Unmarshal(raw, &resp))
	require.Nil(t, resp.Error, "server rejected %s: %s", name, raw)
	require.False(t, resp.Result.IsError, "%s returned an error result: %s", name, raw)
	require.Len(t, resp.Result.Content, 1)

	return resp.Result.StructuredContent, resp.Result.Content[0].Text
}

// Output wraps the payload under a named field, so structuredContent is an
// envelope while the text block keeps the bare payload it always had.
func TestStructuredOutputEnvelopeTools(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mock := NewMockRegistriesService(ctrl)
	mock.EXPECT().
		List(gomock.Any()).
		Return([]*godo.Registry{{Name: "my-registry", Region: "blr1", CreatedAt: time.Now().UTC()}}, nil, nil).
		Times(1)

	structured, text := callTool(t, setupRegistryToolWithMock(mock).Tools(), "docr-list", map[string]any{})

	var envelope struct {
		Registries []*godo.Registry `json:"registries"`
	}
	require.NoError(t, json.Unmarshal(structured, &envelope))
	require.Len(t, envelope.Registries, 1)
	require.Equal(t, "my-registry", envelope.Registries[0].Name)

	// Text content is still the bare array, not the envelope.
	var fromText []*godo.Registry
	require.NoError(t, json.Unmarshal([]byte(text), &fromText))
	require.Equal(t, envelope.Registries, fromText)
}

// NewObjectOutput publishes an already-object payload as-is, so
// structuredContent and the text block carry the same JSON.
func TestStructuredOutputObjectTools(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mock := NewMockRegistriesService(ctrl)
	mock.EXPECT().
		ListRepositoriesV2(gomock.Any(), "my-registry", gomock.Any()).
		Return([]*godo.RepositoryV2{{Name: "repo1"}}, &godo.Response{Meta: &godo.Meta{Total: 1, Page: 1, Pages: 1}}, nil).
		Times(1)

	structured, text := callTool(t, setupRepositoryToolWithMock(mock).Tools(),
		"docr-repository-list", map[string]any{"RegistryName": "my-registry"})

	var payload repositoryList
	require.NoError(t, json.Unmarshal(structured, &payload))
	require.Len(t, payload.Repositories, 1)
	require.Equal(t, "repo1", payload.Repositories[0].Name)
	require.Equal(t, 1, payload.Meta.Total)

	require.JSONEq(t, text, string(structured))
}

// A nil collection serialises to null, which an "array" schema would reject.
func TestStructuredOutputAcceptsEmptyCollections(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mock := NewMockRegistriesService(ctrl)
	mock.EXPECT().List(gomock.Any()).Return(nil, nil, nil).Times(1)

	structured, _ := callTool(t, setupRegistryToolWithMock(mock).Tools(), "docr-list", map[string]any{})
	require.JSONEq(t, `{"registries":[]}`, string(structured))
}
