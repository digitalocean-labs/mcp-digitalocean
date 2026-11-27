package sandbox

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"net/http"
	"testing"

	"github.com/mark3labs/mcp-go/server"
	"github.com/stretchr/testify/require"
)

// callValidated runs one tool through a real MCPServer with output validation
// on, so the server checks structuredContent against the tool's outputSchema.
// A mismatch between the emitted JSON and the schema fails the call.
func callValidated(t *testing.T, tool *Tool, name string, args map[string]any) {
	t.Helper()

	svr := server.NewMCPServer("test", "test", server.WithOutputSchemaValidation())
	svr.AddTools(tool.Tools()...)

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
			IsError bool `json:"isError"`
		} `json:"result"`
	}
	require.NoError(t, json.Unmarshal(raw, &resp))
	require.Nil(t, resp.Error, "server rejected %s: %s", name, raw)
	require.False(t, resp.Result.IsError, "%s returned an error result: %s", name, raw)
}

// TestStructuredOutputMatchesSchema checks that each output-bearing tool
// publishes a schema its real response satisfies. Fixtures use the API's
// string-encoded int64 fields (size, bytes_written, size_bytes) so the
// integer schema is tested against the wire shape.
func TestStructuredOutputMatchesSchema(t *testing.T) {
	const ts = `"2026-11-26T10:00:00Z"`
	sandboxObj := `{"sandbox_id":"sb1","template":"codex-base","state":"SANDBOX_STATE_RUNNING","size_slug":"s-1vcpu","created_at":` + ts + `}`
	snapshotObj := `{"snapshot_id":"snap1","sandbox_id":"sb1","name":"before","state":"SNAPSHOT_STATE_COMPLETE","size_bytes":"1048576","created_at":` + ts + `,"ttl":"24h"}`
	templateObj := `{"template_id":"t1","name":"py","spec":{"base_template":"codex-base"},"status":"TEMPLATE_STATUS_READY","created_at":` + ts + `,"updated_at":` + ts + `}`
	buildObj := `{"build_id":"b1","template_id":"t1","status":"TEMPLATE_BUILD_STATUS_SUCCEEDED","created_at":` + ts + `,"updated_at":` + ts + `}`
	fileEntryObj := `{"name":"a.txt","type":"FILE_ENTRY_TYPE_FILE","path":"/workspace/a.txt","size":"12","mode":420,"permissions":"-rw-r--r--","owner":"root","group":"root","modified_time":` + ts + `}`

	// Payload plus footer for the download fixture.
	payload := []byte("hello world\n")
	sum := sha256.Sum256(payload)
	downloadBody := string(payload) + "DOWSSHA1" + hex.EncodeToString(sum[:]) + "\n"

	const json1 = "application/json"
	cases := []struct {
		name     string
		args     map[string]any
		response string
		ctype    string
	}{
		{"sandbox-list", map[string]any{}, `{"sandboxes":[` + sandboxObj + `],"next_page_token":"n"}`, json1},
		{"sandbox-create", map[string]any{"Template": "codex-base", "SizeSlug": "s-1vcpu"}, `{"sandbox":` + sandboxObj + `}`, json1},
		{"sandbox-get", map[string]any{"SandboxID": "sb1"}, `{"sandbox":` + sandboxObj + `}`, json1},
		{"sandbox-pause", map[string]any{"SandboxID": "sb1"}, `{"sandbox":` + sandboxObj + `}`, json1},
		{"sandbox-resume", map[string]any{"SandboxID": "sb1"}, `{"sandbox":` + sandboxObj + `}`, json1},
		{"sandbox-snapshot-create", map[string]any{"SandboxID": "sb1"}, `{"snapshot":` + snapshotObj + `}`, json1},
		{"sandbox-snapshot-get", map[string]any{"SnapshotID": "snap1"}, `{"snapshot":` + snapshotObj + `}`, json1},
		{"sandbox-snapshot-list", map[string]any{}, `{"snapshots":[` + snapshotObj + `]}`, json1},
		{"sandbox-template-list", map[string]any{}, `{"templates":[` + templateObj + `]}`, json1},
		{"sandbox-template-get", map[string]any{"TemplateID": "t1"}, `{"template":` + templateObj + `}`, json1},
		{"sandbox-template-create", map[string]any{"Name": "py", "BaseTemplate": "codex-base", "SourceOCIRef": "reg/img:1"}, `{"template":` + templateObj + `}`, json1},
		{"sandbox-template-update", map[string]any{"TemplateID": "t1", "SourceOCIRef": "reg/img:2"}, `{"template":` + templateObj + `}`, json1},
		{"sandbox-template-build-list", map[string]any{"TemplateID": "t1"}, `{"builds":[` + buildObj + `]}`, json1},
		{"sandbox-template-build-get", map[string]any{"TemplateID": "t1", "BuildID": "b1"}, `{"build":` + buildObj + `}`, json1},
		{"sandbox-command-list", map[string]any{"SandboxID": "sb1"}, `{"commands":[{"pid":42,"cmd":"sleep","cwd":"/"}]}`, json1},
		{"sandbox-command-start", map[string]any{"SandboxID": "sb1", "Cmd": "sleep"}, `{"pid":42}`, json1},
		{"sandbox-file-list", map[string]any{"SandboxID": "sb1", "Path": "/workspace"}, `{"entries":[` + fileEntryObj + `]}`, json1},
		{"sandbox-file-info", map[string]any{"SandboxID": "sb1", "Path": "/workspace/a.txt"}, `{"entry":` + fileEntryObj + `}`, json1},
		{"sandbox-file-exists", map[string]any{"SandboxID": "sb1", "Path": "/workspace/a.txt"}, `{"exists":true}`, json1},
		{"sandbox-file-make-directory", map[string]any{"SandboxID": "sb1", "Path": "/workspace/d"}, `{"created":true,"entry":` + fileEntryObj + `}`, json1},
		{"sandbox-file-move", map[string]any{"SandboxID": "sb1", "Source": "/a", "Destination": "/b"}, `{"entry":` + fileEntryObj + `}`, json1},
		{"sandbox-file-upload", map[string]any{"SandboxID": "sb1", "Path": "/a", "Content": "x"}, `{"path":"/a","bytes_written":"1"}`, json1},
		{"sandbox-file-download", map[string]any{"SandboxID": "sb1", "Path": "/workspace/a.txt"}, downloadBody, "application/octet-stream"},
		{"sandbox-exec-start", map[string]any{"SandboxID": "sb1", "Argv": []any{"echo", "hi"}},
			`{"result":{"started":{"exec_id":"e1"}}}` + "\n" +
				`{"result":{"stdout":"aGkK"}}` + "\n" +
				`{"result":{"exit":{"exit_code":0,"duration_ms":"5","status":"exited"}}}` + "\n",
			json1},
	}

	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			tool := newTestTool(t, func(w http.ResponseWriter, _ *http.Request) {
				w.Header().Set("Content-Type", c.ctype)
				_, _ = w.Write([]byte(c.response))
			})
			callValidated(t, tool, c.name, c.args)
		})
	}
}
