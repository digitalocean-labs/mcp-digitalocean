package sandbox

import (
	"context"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"

	"github.com/digitalocean/godo"
	"github.com/mark3labs/mcp-go/mcp"
	"github.com/mark3labs/mcp-go/server"
	"github.com/stretchr/testify/require"
)

// newTestTool points the tools at an httptest server running handler.
func newTestTool(t *testing.T, handler http.HandlerFunc) *Tool {
	t.Helper()
	srv := httptest.NewServer(handler)
	t.Cleanup(srv.Close)
	base, err := url.Parse(srv.URL)
	require.NoError(t, err)
	return NewTool(func(context.Context) (*godo.Client, error) {
		c := godo.NewFromToken("test")
		c.BaseURL = base
		return c, nil
	})
}

// findTool returns the registered tool with the given name.
func findTool(t *testing.T, tool *Tool, name string) server.ServerTool {
	t.Helper()
	for _, st := range tool.Tools() {
		if st.Tool.Name == name {
			return st
		}
	}
	t.Fatalf("tool %q not registered", name)
	return server.ServerTool{}
}

func call(t *testing.T, tool *Tool, name string, args map[string]any) *mcp.CallToolResult {
	t.Helper()
	res, err := findTool(t, tool, name).Handler(context.Background(), request(args))
	require.NoError(t, err)
	return res
}

func callText(t *testing.T, res *mcp.CallToolResult) string {
	t.Helper()
	require.NotEmpty(t, res.Content)
	text, ok := res.Content[0].(mcp.TextContent)
	require.True(t, ok)
	return text.Text
}

func request(args map[string]any) mcp.CallToolRequest {
	return mcp.CallToolRequest{Params: mcp.CallToolParams{Arguments: args}}
}

// okJSON answers every request with an empty JSON object.
func okJSON(w http.ResponseWriter, _ *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	_, _ = w.Write([]byte(`{}`))
}

func TestToolsRegistered(t *testing.T) {
	tool := NewTool(nil)
	seen := map[string]bool{}
	for _, st := range tool.Tools() {
		name := st.Tool.Name
		require.True(t, strings.HasPrefix(name, "sandbox-"), "tool %q must use the sandbox- prefix", name)
		require.False(t, seen[name], "duplicate tool %q", name)
		seen[name] = true
	}
	// 33 spec operations minus streaming attach, and minus exec resize and
	// stdin, which need a long-lived interactive exec session.
	require.Len(t, seen, 30)
}

// routeCase drives one tool and checks the HTTP request it produces.
type routeCase struct {
	tool     string
	args     map[string]any
	method   string
	uri      string // path plus query
	bodyPart string // substring the JSON body must contain, if non-empty
}

func TestToolRouting(t *testing.T) {
	cases := []routeCase{
		{tool: "sandbox-list", args: map[string]any{"PageSize": float64(10), "State": "SANDBOX_STATE_RUNNING"},
			method: http.MethodGet, uri: "/v2/sandboxes?page_size=10&state=SANDBOX_STATE_RUNNING"},
		{tool: "sandbox-create", args: map[string]any{"Template": "codex-base", "SizeSlug": "s-1vcpu", "AllowInternetAccess": false},
			method: http.MethodPost, uri: "/v2/sandboxes", bodyPart: `"allow_internet_access":false`},
		{tool: "sandbox-get", args: map[string]any{"SandboxID": "sb1"},
			method: http.MethodGet, uri: "/v2/sandboxes/sb1"},
		{tool: "sandbox-delete", args: map[string]any{"SandboxID": "sb1"},
			method: http.MethodDelete, uri: "/v2/sandboxes/sb1"},
		{tool: "sandbox-pause", args: map[string]any{"SandboxID": "sb1"},
			method: http.MethodPost, uri: "/v2/sandboxes/sb1/pause"},
		{tool: "sandbox-resume", args: map[string]any{"SandboxID": "sb1"},
			method: http.MethodPost, uri: "/v2/sandboxes/sb1/resume"},

		{tool: "sandbox-snapshot-create", args: map[string]any{"SandboxID": "sb1", "Name": "before"},
			method: http.MethodPost, uri: "/v2/sandboxes/sb1/snapshots", bodyPart: `"name":"before"`},
		{tool: "sandbox-snapshot-list", args: map[string]any{"SandboxID": "sb1"},
			method: http.MethodGet, uri: "/v2/sandboxes/snapshots?sandbox_id=sb1"},
		{tool: "sandbox-snapshot-get", args: map[string]any{"SnapshotID": "snap1"},
			method: http.MethodGet, uri: "/v2/sandboxes/snapshots/snap1"},
		{tool: "sandbox-snapshot-delete", args: map[string]any{"SnapshotID": "snap1"},
			method: http.MethodDelete, uri: "/v2/sandboxes/snapshots/snap1"},

		{tool: "sandbox-template-list", args: map[string]any{},
			method: http.MethodGet, uri: "/v2/sandboxes/templates"},
		{tool: "sandbox-template-get", args: map[string]any{"TemplateID": "t1"},
			method: http.MethodGet, uri: "/v2/sandboxes/templates/t1"},
		{tool: "sandbox-template-create", args: map[string]any{"Name": "py", "BaseTemplate": "codex-base", "SourceOCIRef": "reg/img:1"},
			method: http.MethodPost, uri: "/v2/sandboxes/templates", bodyPart: `"source_oci_ref":"reg/img:1"`},
		{tool: "sandbox-template-update", args: map[string]any{"TemplateID": "t1", "SourceOCIRef": "reg/img:2"},
			method: http.MethodPut, uri: "/v2/sandboxes/templates/t1", bodyPart: `"source_oci_ref":"reg/img:2"`},
		{tool: "sandbox-template-delete", args: map[string]any{"TemplateID": "t1"},
			method: http.MethodDelete, uri: "/v2/sandboxes/templates/t1"},
		{tool: "sandbox-template-build-list", args: map[string]any{"TemplateID": "t1"},
			method: http.MethodGet, uri: "/v2/sandboxes/templates/t1/builds"},
		{tool: "sandbox-template-build-get", args: map[string]any{"TemplateID": "t1", "BuildID": "b1"},
			method: http.MethodGet, uri: "/v2/sandboxes/templates/t1/builds/b1"},

		{tool: "sandbox-command-list", args: map[string]any{"SandboxID": "sb1"},
			method: http.MethodGet, uri: "/v2/sandboxes/sb1/commands"},
		{tool: "sandbox-command-start", args: map[string]any{"SandboxID": "sb1", "Cmd": "sleep", "Args": []any{"100"}, "Stdin": true},
			method: http.MethodPost, uri: "/v2/sandboxes/sb1/commands", bodyPart: `"stdin":true`},
		{tool: "sandbox-command-signal", args: map[string]any{"SandboxID": "sb1", "PID": float64(42), "Signal": "SIGTERM"},
			method: http.MethodPost, uri: "/v2/sandboxes/sb1/commands/42/signal", bodyPart: "SANDBOX_COMMAND_SIGNAL_SIGTERM"},
		{tool: "sandbox-command-stdin", args: map[string]any{"SandboxID": "sb1", "PID": float64(42), "Data": "aGkK", "EOF": true},
			method: http.MethodPost, uri: "/v2/sandboxes/sb1/commands/42/stdin", bodyPart: `"eof":true`},

		{tool: "sandbox-file-list", args: map[string]any{"SandboxID": "sb1", "Path": "/workspace", "Depth": float64(2)},
			method: http.MethodGet, uri: "/v2/sandboxes/sb1/files/list?depth=2&path=%2Fworkspace"},
		{tool: "sandbox-file-info", args: map[string]any{"SandboxID": "sb1", "Path": "/workspace/a.txt"},
			method: http.MethodGet, uri: "/v2/sandboxes/sb1/files/info?path=%2Fworkspace%2Fa.txt"},
		{tool: "sandbox-file-exists", args: map[string]any{"SandboxID": "sb1", "Path": "/workspace/a.txt"},
			method: http.MethodGet, uri: "/v2/sandboxes/sb1/files/exists?path=%2Fworkspace%2Fa.txt"},
		{tool: "sandbox-file-make-directory", args: map[string]any{"SandboxID": "sb1", "Path": "/workspace/d"},
			method: http.MethodPost, uri: "/v2/sandboxes/sb1/files/make-directory", bodyPart: `"path":"/workspace/d"`},
		{tool: "sandbox-file-move", args: map[string]any{"SandboxID": "sb1", "Source": "/a", "Destination": "/b"},
			method: http.MethodPost, uri: "/v2/sandboxes/sb1/files/move", bodyPart: `"destination":"/b"`},
		{tool: "sandbox-file-remove", args: map[string]any{"SandboxID": "sb1", "Path": "/workspace/a.txt"},
			method: http.MethodPost, uri: "/v2/sandboxes/sb1/files/remove", bodyPart: `"path":"/workspace/a.txt"`},
	}

	for _, c := range cases {
		t.Run(c.tool, func(t *testing.T) {
			var gotMethod, gotURI, gotBody string
			tool := newTestTool(t, func(w http.ResponseWriter, r *http.Request) {
				gotMethod = r.Method
				gotURI = r.URL.RequestURI()
				b, _ := io.ReadAll(r.Body)
				gotBody = string(b)
				okJSON(w, r)
			})

			res := call(t, tool, c.tool, c.args)
			require.False(t, res.IsError, callText(t, res))
			require.Equal(t, c.method, gotMethod)
			require.Equal(t, c.uri, gotURI)
			if c.bodyPart != "" {
				require.Contains(t, gotBody, c.bodyPart)
			}
		})
	}
}

func TestMissingRequiredArgumentsAreToolErrors(t *testing.T) {
	tool := newTestTool(t, okJSON)
	cases := map[string]map[string]any{
		"sandbox-get":             {},
		"sandbox-create":          {"SizeSlug": "s"},
		"sandbox-command-signal":  {"SandboxID": "sb1", "PID": float64(1), "Signal": "SIGHUP"},
		"sandbox-command-stdin":   {"SandboxID": "sb1", "PID": float64(0)},
		"sandbox-template-update": {"TemplateID": "t1"},
		"sandbox-file-move":       {"SandboxID": "sb1", "Source": "/a"},
	}
	for name, args := range cases {
		t.Run(name, func(t *testing.T) {
			res := call(t, tool, name, args)
			require.True(t, res.IsError)
		})
	}
}

func TestAPIErrorsBecomeToolErrors(t *testing.T) {
	tool := newTestTool(t, func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusNotFound)
		_, _ = w.Write([]byte(`{"id":"not_found","message":"sandbox not found"}`))
	})

	res := call(t, tool, "sandbox-get", map[string]any{"SandboxID": "missing"})
	require.True(t, res.IsError)
	require.Contains(t, callText(t, res), "Failed to get sandbox")
}

// decodeText unmarshals a tool's JSON text result into v.
func decodeText(t *testing.T, res *mcp.CallToolResult, v any) {
	t.Helper()
	require.NoError(t, json.Unmarshal([]byte(callText(t, res)), v))
}

func TestListSandboxesEmptyArrayNotNull(t *testing.T) {
	tool := newTestTool(t, okJSON)
	res := call(t, tool, "sandbox-list", map[string]any{})
	require.False(t, res.IsError)

	var got sandboxListResponse
	decodeText(t, res, &got)
	require.NotNil(t, got.Sandboxes, "empty list must serialize as [] rather than null")
}

// ---- Exec stream ----

const (
	b64Hello = "aGVsbG8K" // "hello\n"
	b64Err   = "ZXJy"     // "err"
)

func execStreamSSE() string {
	return "event: started\n" +
		`data: {"result":{"started":{"exec_id":"e1"}}}` + "\n\n" +
		"event: stdout\n" +
		`data: {"result":{"stdout":"` + b64Hello + `"}}` + "\n\n" +
		"event: stderr\n" +
		`data: {"result":{"stderr":"` + b64Err + `"}}` + "\n\n" +
		"event: exit\n" +
		`data: {"result":{"exit":{"exit_code":3,"duration_ms":"120","status":"exited"}}}` + "\n\n"
}

func execStreamNDJSON() string {
	return `{"result":{"started":{"exec_id":"e1"}}}` + "\n" +
		`{"result":{"stdout":"` + b64Hello + `"}}` + "\n" +
		`{"result":{"stderr":"` + b64Err + `"}}` + "\n" +
		`{"result":{"exit":{"exit_code":3,"duration_ms":"120","status":"exited"}}}` + "\n"
}

func TestExecStartParsesEventStream(t *testing.T) {
	for name, body := range map[string]string{"sse": execStreamSSE(), "ndjson": execStreamNDJSON()} {
		t.Run(name, func(t *testing.T) {
			var gotPath string
			tool := newTestTool(t, func(w http.ResponseWriter, r *http.Request) {
				gotPath = r.URL.Path
				w.Header().Set("Content-Type", "text/event-stream")
				_, _ = w.Write([]byte(body))
			})

			res := call(t, tool, "sandbox-exec-start", map[string]any{
				"SandboxID": "sb1",
				"Argv":      []any{"sh", "-c", "echo hello"},
			})
			require.False(t, res.IsError, callText(t, res))
			require.Equal(t, "/v2/sandboxes/sb1/exec", gotPath)

			var got ExecResult
			decodeText(t, res, &got)
			require.Equal(t, "e1", got.ExecID)
			require.Equal(t, int32(3), got.ExitCode)
			require.Equal(t, "hello\n", got.Stdout)
			require.Equal(t, "err", got.Stderr)
			require.Equal(t, int64(120), got.DurationMS)
			require.Equal(t, "exited", got.Status)
		})
	}
}

func TestExecStreamWithoutExitIsAnError(t *testing.T) {
	tool := newTestTool(t, func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte(`{"result":{"stdout":"` + b64Hello + `"}}` + "\n"))
	})

	res := call(t, tool, "sandbox-exec-start", map[string]any{"SandboxID": "sb1", "Argv": []any{"ls"}})
	require.True(t, res.IsError)
	require.Contains(t, callText(t, res), "ended before the process exited")
}

func TestExecStreamErrorEnvelopeIsAnError(t *testing.T) {
	tool := newTestTool(t, func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte(`{"error":{"message":"guest unreachable"}}` + "\n"))
	})

	res := call(t, tool, "sandbox-exec-start", map[string]any{"SandboxID": "sb1", "Argv": []any{"ls"}})
	require.True(t, res.IsError)
	require.Contains(t, callText(t, res), "guest unreachable")
}

func TestExecRequiresArgv(t *testing.T) {
	tool := newTestTool(t, okJSON)
	res := call(t, tool, "sandbox-exec-start", map[string]any{"SandboxID": "sb1"})
	require.True(t, res.IsError)
	require.Contains(t, callText(t, res), "Argv is required")
}

func TestExecOutputIsCapped(t *testing.T) {
	var b outputBuffer
	chunk := make([]byte, 600*1024)
	b.write(chunk)
	b.write(chunk) // second write exceeds the 1 MiB cap
	require.True(t, b.truncated)
	require.Equal(t, maxExecOutputBytes, b.buf.Len())
}

// ---- File download ----

// footerFor returns the 73-byte integrity footer for payload.
func footerFor(payload []byte) string {
	sum := sha256.Sum256(payload)
	return "DOWSSHA1" + hex.EncodeToString(sum[:]) + "\n"
}

func TestDownloadTextFile(t *testing.T) {
	payload := []byte("hello world")
	var gotQuery string
	var gotAccept string
	tool := newTestTool(t, func(w http.ResponseWriter, r *http.Request) {
		gotQuery = r.URL.RawQuery
		gotAccept = r.Header.Get("Accept")
		w.Header().Set("Content-Type", "application/octet-stream")
		_, _ = w.Write(append(payload, footerFor(payload)...))
	})

	res := call(t, tool, "sandbox-file-download", map[string]any{"SandboxID": "sb1", "Path": "/workspace/a.txt"})
	require.False(t, res.IsError, callText(t, res))
	require.Equal(t, "path=%2Fworkspace%2Fa.txt", gotQuery)
	require.Equal(t, "application/octet-stream", gotAccept)

	var got FileDownloadResult
	decodeText(t, res, &got)
	require.Equal(t, "hello world", got.Content)
	require.Equal(t, "utf-8", got.Encoding)
	require.Equal(t, len(payload), got.SizeBytes)
	sum := sha256.Sum256(payload)
	require.Equal(t, hex.EncodeToString(sum[:]), got.SHA256)
}

func TestDownloadBinaryFileIsBase64(t *testing.T) {
	payload := []byte{0xff, 0x00, 0x10}
	tool := newTestTool(t, func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write(append(payload, footerFor(payload)...))
	})

	res := call(t, tool, "sandbox-file-download", map[string]any{"SandboxID": "sb1", "Path": "/bin/x"})
	require.False(t, res.IsError, callText(t, res))
	var got FileDownloadResult
	decodeText(t, res, &got)
	require.Equal(t, "base64", got.Encoding)
	require.Equal(t, base64.StdEncoding.EncodeToString(payload), got.Content)
}

func TestDownloadRejectsCorruptPayload(t *testing.T) {
	payload := []byte("hello")
	wrong := footerFor([]byte("other"))
	tool := newTestTool(t, func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write(append(payload, wrong...))
	})

	res := call(t, tool, "sandbox-file-download", map[string]any{"SandboxID": "sb1", "Path": "/a"})
	require.True(t, res.IsError)
	require.Contains(t, callText(t, res), "integrity check")
}

func TestDownloadRejectsMissingFooter(t *testing.T) {
	tool := newTestTool(t, func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte("no footer here, just plain bytes that are long enough to pass the length check ....."))
	})

	res := call(t, tool, "sandbox-file-download", map[string]any{"SandboxID": "sb1", "Path": "/a"})
	require.True(t, res.IsError)
}

func TestDownloadRejectsOversizedFile(t *testing.T) {
	tool := newTestTool(t, func(w http.ResponseWriter, r *http.Request) {
		big := make([]byte, maxInlineFileBytes+downloadFooterLen+10)
		for i := range big {
			big[i] = 'a'
		}
		_, _ = w.Write(big)
	})

	res := call(t, tool, "sandbox-file-download", map[string]any{"SandboxID": "sb1", "Path": "/big"})
	require.True(t, res.IsError)
	require.Contains(t, callText(t, res), "inline limit")
}

// ---- File upload ----

func TestUploadTextSendsRawBodyAndChecksum(t *testing.T) {
	content := "hi there " // trailing space must survive
	var gotBody, gotSum, gotType, gotQuery string
	tool := newTestTool(t, func(w http.ResponseWriter, r *http.Request) {
		b, _ := io.ReadAll(r.Body)
		gotBody = string(b)
		gotSum = r.Header.Get("X-Content-Sha256")
		gotType = r.Header.Get("Content-Type")
		gotQuery = r.URL.RawQuery
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"path":"/workspace/a.txt","bytes_written":"9"}`))
	})

	res := call(t, tool, "sandbox-file-upload", map[string]any{
		"SandboxID": "sb1",
		"Path":      "/workspace/a.txt",
		"Content":   content,
	})
	require.False(t, res.IsError, callText(t, res))
	require.Equal(t, content, gotBody)
	sum := sha256.Sum256([]byte(content))
	require.Equal(t, hex.EncodeToString(sum[:]), gotSum)
	require.Equal(t, "application/octet-stream", gotType)
	require.Equal(t, "path=%2Fworkspace%2Fa.txt", gotQuery)

	var got fileUploadResponse
	decodeText(t, res, &got)
	require.Equal(t, int64(9), int64(got.BytesWritten))
}

func TestUploadBase64Content(t *testing.T) {
	raw := []byte{0x00, 0xff, 0x7f}
	var gotBody []byte
	tool := newTestTool(t, func(w http.ResponseWriter, r *http.Request) {
		gotBody, _ = io.ReadAll(r.Body)
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"path":"/bin/x","bytes_written":"3"}`))
	})

	res := call(t, tool, "sandbox-file-upload", map[string]any{
		"SandboxID":     "sb1",
		"Path":          "/bin/x",
		"ContentBase64": base64.StdEncoding.EncodeToString(raw),
	})
	require.False(t, res.IsError, callText(t, res))
	require.Equal(t, raw, gotBody)
}

func TestUploadArgumentValidation(t *testing.T) {
	tool := newTestTool(t, okJSON)
	base := map[string]any{"SandboxID": "sb1", "Path": "/a"}

	both := map[string]any{"SandboxID": "sb1", "Path": "/a", "Content": "x", "ContentBase64": "eA=="}
	require.True(t, call(t, tool, "sandbox-file-upload", both).IsError)

	require.True(t, call(t, tool, "sandbox-file-upload", base).IsError)

	bad := map[string]any{"SandboxID": "sb1", "Path": "/a", "ContentBase64": "!!not-base64!!"}
	res := call(t, tool, "sandbox-file-upload", bad)
	require.True(t, res.IsError)
	require.Contains(t, callText(t, res), "valid base64")
}

func TestUploadRejectsOversizedContentBeforeSending(t *testing.T) {
	sent := false
	tool := newTestTool(t, func(w http.ResponseWriter, r *http.Request) {
		sent = true
		okJSON(w, r)
	})

	res := call(t, tool, "sandbox-file-upload", map[string]any{
		"SandboxID": "sb1",
		"Path":      "/big",
		"Content":   strings.Repeat("a", maxInlineFileBytes+1),
	})
	require.True(t, res.IsError)
	require.Contains(t, callText(t, res), "inline limit")
	require.False(t, sent, "oversized content must not reach the API")
}
