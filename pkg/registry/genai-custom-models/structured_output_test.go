package genaicustommodels

import (
	"context"
	"encoding/json"
	"strings"
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

// TestTableToolsStructuredOutput covers the two tools whose text half is a
// markdown table rather than the payload's JSON. They are the only tools here
// that do not derive their text from the payload, so the table and
// structuredContent can drift apart; this asserts they agree, and runs through
// a validating server so the rows still satisfy the declared schema.
func TestTableToolsStructuredOutput(t *testing.T) {
	models := []*CustomModel{
		{UUID: "uuid-ready", Name: "ready-model", Status: CustomModelStatusReady, Architecture: "LlamaForCausalLM"},
		{UUID: "uuid-failed", Name: "failed-model", Status: CustomModelStatusFailed, ErrorMessage: "import validation failed"},
	}

	svr := server.NewMCPServer("test", "test", server.WithOutputSchemaValidation())
	svr.AddTools(setupCustomModelsToolWithTestServer(t, models).Tools()...)

	ctx := context.Background()
	svr.HandleMessage(ctx, []byte(`{"jsonrpc":"2.0","id":1,"method":"initialize",`+
		`"params":{"protocolVersion":"2025-06-18","capabilities":{},"clientInfo":{"name":"t","version":"1"}}}`))

	call := func(id, name, args string) (string, json.RawMessage) {
		t.Helper()
		raw, err := json.Marshal(svr.HandleMessage(ctx,
			[]byte(`{"jsonrpc":"2.0","id":`+id+`,"method":"tools/call","params":{"name":"`+name+`","arguments":`+args+`}}`)))
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
		require.Nil(t, resp.Error, "server rejected the call")
		require.False(t, resp.Result.IsError, "tool returned an error result: %s", raw)
		require.Len(t, resp.Result.Content, 1)
		return resp.Result.Content[0].Text, resp.Result.StructuredContent
	}

	t.Run("list", func(t *testing.T) {
		text, structured := call("2", "genai-custom-models-list", `{}`)

		// The table is what the client is told to display, so it must survive
		// unchanged rather than being replaced by the payload's JSON.
		require.Contains(t, text, "## Custom Models (2 models)")
		require.Contains(t, text, "| uuid-failed | failed-model |")

		var payload struct {
			Models []CustomSearchRow `json:"models"`
		}
		require.NoError(t, json.Unmarshal(structured, &payload))
		requireRowsMatchTable(t, text, payload.Models)
	})

	t.Run("unified search", func(t *testing.T) {
		text, structured := call("3", "genai-models-unified-search", `{"query":"model"}`)

		require.Contains(t, text, "# Unified Model Search")
		require.Contains(t, text, "**Query:** model")

		var payload UnifiedSearchResponse
		require.NoError(t, json.Unmarshal(structured, &payload))
		require.Equal(t, "model", payload.Query)
		require.Equal(t, len(payload.CatalogModels), payload.Counts.Catalog)
		require.Equal(t, len(payload.CustomModels), payload.Counts.Custom)
		require.Equal(t, payload.Counts.Catalog+payload.Counts.Custom, payload.Counts.Total)
		requireRowsMatchTable(t, text, payload.CustomModels)
	})
}

// requireRowsMatchTable checks that the payload lists the same models, in the
// same order, as the rendered table.
func requireRowsMatchTable(t *testing.T, text string, rows []CustomSearchRow) {
	t.Helper()

	var fromTable []string
	for _, line := range strings.Split(text, "\n") {
		cells := strings.Split(strings.Trim(strings.TrimSpace(line), "|"), "|")
		if len(cells) != 8 {
			continue
		}
		if uuid := strings.TrimSpace(cells[0]); strings.HasPrefix(uuid, "uuid-") {
			fromTable = append(fromTable, uuid)
		}
	}

	fromPayload := make([]string, 0, len(rows))
	for _, r := range rows {
		fromPayload = append(fromPayload, r.UUID)
	}

	require.NotEmpty(t, fromTable, "no model rows found in table:\n%s", text)
	require.Equal(t, fromTable, fromPayload, "table and structuredContent disagree")
}
