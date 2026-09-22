package genaicustommodels

import (
	"context"
	"encoding/json"
	"testing"

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
func TestStructuredOutputSatisfiesDeclaredSchema(t *testing.T) {
	const modelUUID = "123e4567-e89b-12d3-a456-426614174000"

	tool := setupCustomModelsToolWithTestServer(t, []*CustomModel{{
		UUID:            modelUUID,
		Name:            "my-model",
		Status:          CustomModelStatusReady,
		Architecture:    "LlamaForCausalLM",
		SourceType:      CustomModelSourceTypeHuggingFace,
		SourceRef:       &CustomModelSourceRef{RepoID: "meta-llama/Llama-3", AccessType: CustomModelAccessTypePublic},
		Tags:            &CustomModelTags{Tags: []string{"llm"}},
		FileCount:       "3",
		ContextLength:   "8192",
		InputModalities: []string{"text"},
	}})

	svr := server.NewMCPServer("test", "test", server.WithOutputSchemaValidation())
	svr.AddTools(tool.Tools()...)

	ctx := context.Background()
	svr.HandleMessage(ctx, []byte(`{"jsonrpc":"2.0","id":1,"method":"initialize",`+
		`"params":{"protocolVersion":"2025-06-18","capabilities":{},"clientInfo":{"name":"t","version":"1"}}}`))

	raw, err := json.Marshal(svr.HandleMessage(ctx,
		[]byte(`{"jsonrpc":"2.0","id":2,"method":"tools/call","params":{"name":"genai-custom-models-get",`+
			`"arguments":{"uuid":"`+modelUUID+`"}}}`)))
	require.NoError(t, err)

	var resp struct {
		Error  *struct{ Message string } `json:"error"`
		Result struct {
			IsError           bool              `json:"isError"`
			Content           []mcp.TextContent `json:"content"`
			StructuredContent struct {
				Model *CustomModel `json:"model"`
			} `json:"structuredContent"`
		} `json:"result"`
	}
	require.NoError(t, json.Unmarshal(raw, &resp))

	require.Nil(t, resp.Error, "server rejected the call")
	require.False(t, resp.Result.IsError, "tool returned an error result: %s", raw)

	// A bare resource cannot be an object root on its own, so it is published
	// under the "model" envelope.
	require.NotNil(t, resp.Result.StructuredContent.Model)
	require.Equal(t, modelUUID, resp.Result.StructuredContent.Model.UUID)
	require.Equal(t, "my-model", resp.Result.StructuredContent.Model.Name)

	// The text block is retained for backwards compatibility and carries the
	// same payload, unwrapped as it always was.
	require.Len(t, resp.Result.Content, 1)
	var fromText CustomModel
	require.NoError(t, json.Unmarshal([]byte(resp.Result.Content[0].Text), &fromText))
	require.Equal(t, *resp.Result.StructuredContent.Model, fromText)
}
