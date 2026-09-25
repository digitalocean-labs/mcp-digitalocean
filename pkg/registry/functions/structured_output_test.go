package functions

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/digitalocean/godo"
	"github.com/mark3labs/mcp-go/mcp"
	"github.com/mark3labs/mcp-go/server"
	"github.com/stretchr/testify/require"
	"go.uber.org/mock/gomock"
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
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mock := NewMockFunctionsService(ctrl)
	mock.EXPECT().
		ListNamespaces(gomock.Any()).
		Return([]godo.FunctionsNamespace{
			{Namespace: "ns-1", Label: "test-1", Region: "nyc1", ApiHost: "https://faas.example.com"},
		}, nil, nil).
		Times(1)

	svr := server.NewMCPServer("test", "test", server.WithOutputSchemaValidation())
	svr.AddTools(setupNamespaceTool(mock).Tools()...)

	ctx := context.Background()
	svr.HandleMessage(ctx, []byte(`{"jsonrpc":"2.0","id":1,"method":"initialize",`+
		`"params":{"protocolVersion":"2025-06-18","capabilities":{},"clientInfo":{"name":"t","version":"1"}}}`))

	raw, err := json.Marshal(svr.HandleMessage(ctx,
		[]byte(`{"jsonrpc":"2.0","id":2,"method":"tools/call","params":{"name":"functions-list-namespaces","arguments":{}}}`)))
	require.NoError(t, err)

	var resp struct {
		Error  *struct{ Message string } `json:"error"`
		Result struct {
			IsError           bool              `json:"isError"`
			Content           []mcp.TextContent `json:"content"`
			StructuredContent struct {
				Namespaces []godo.FunctionsNamespace `json:"namespaces"`
			} `json:"structuredContent"`
		} `json:"result"`
	}
	require.NoError(t, json.Unmarshal(raw, &resp))

	require.Nil(t, resp.Error, "server rejected the call")
	require.False(t, resp.Result.IsError, "tool returned an error result: %s", raw)

	// The payload is an array, so it is published under an envelope to keep
	// structuredContent a JSON object.
	require.Len(t, resp.Result.StructuredContent.Namespaces, 1)
	require.Equal(t, "ns-1", resp.Result.StructuredContent.Namespaces[0].Namespace)

	// The text block is retained for backwards compatibility and carries the
	// bare payload, without the envelope.
	require.Len(t, resp.Result.Content, 1)
	var fromText []godo.FunctionsNamespace
	require.NoError(t, json.Unmarshal([]byte(resp.Result.Content[0].Text), &fromText))
	require.Equal(t, resp.Result.StructuredContent.Namespaces, fromText)
}

// TestDataPlaneStructuredOutputIsLossless covers the OpenWhisk tools, which
// differ from every other migrated tool here: they never decode the response,
// so their structuredContent is the upstream bytes rather than something this
// package re-encoded. Two properties follow from that and neither holds
// automatically.
//
// First, the payload has to keep fields the models do not declare, since
// dropping them would change what the tools return. Every fixture below
// therefore carries an undeclared key, and the test compares the whole
// document rather than picking out known fields.
//
// Second, those extra fields still have to satisfy the declared schema. That
// is only true because the generated schema requires nothing and permits
// additional properties, which is a property of the schema generator rather
// than of these models — so it is worth pinning here, with validation on.
func TestDataPlaneStructuredOutputIsLossless(t *testing.T) {
	tests := []struct {
		name      string
		tool      string
		arguments string
		// envelope is the structuredContent key a bare array or bare resource
		// is published under, or "" when the payload is already an object.
		envelope string
		response string
	}{
		{
			name:      "list actions",
			tool:      "functions-list-actions",
			arguments: `{"NamespaceID":"ns-1"}`,
			envelope:  "actions",
			response: `[{"name":"hello","namespace":"test-ns","version":"0.0.1","publish":false,
				"exec":{"kind":"nodejs:20","binary":false},
				"limits":{"timeout":60000,"memory":256,"logs":10,"concurrency":1},
				"annotations":[{"key":"provide-api-key","value":false}],
				"updated":1700000000000,"unmodelled":{"nested":true}}]`,
		},
		{
			name:      "get action",
			tool:      "functions-get-action",
			arguments: `{"NamespaceID":"ns-1","ActionName":"hello"}`,
			envelope:  "action",
			response: `{"name":"hello","namespace":"test-ns","version":"0.0.1","publish":false,
				"exec":{"kind":"nodejs:20","code":"function main(){}","main":"main","binary":false},
				"parameters":[{"key":"greeting","value":"hi"}],
				"limits":{"timeout":60000,"memory":256,"logs":10},
				"unmodelled":"kept"}`,
		},
		{
			name:      "create or update action",
			tool:      "functions-create-or-update-action",
			arguments: `{"NamespaceID":"ns-1","ActionName":"hello","Kind":"nodejs:20"}`,
			envelope:  "action",
			response:  `{"name":"hello","namespace":"test-ns","version":"0.0.2","exec":{"kind":"nodejs:20"},"unmodelled":1}`,
		},
		{
			name:      "list packages",
			tool:      "functions-list-packages",
			arguments: `{"NamespaceID":"ns-1"}`,
			envelope:  "packages",
			response: `[{"name":"utils","namespace":"test-ns","version":"0.0.1","publish":true,
				"binding":{},"actions":[{"name":"hello","version":"0.0.1"}],
				"feeds":[],"updated":1700000000000,"unmodelled":["a"]}]`,
		},
		{
			name:      "get package",
			tool:      "functions-get-package",
			arguments: `{"NamespaceID":"ns-1","PackageName":"utils"}`,
			envelope:  "package",
			response: `{"name":"utils","namespace":"test-ns","version":"0.0.1","publish":false,
				"binding":{"namespace":"other","name":"shared"},
				"parameters":[{"key":"k","value":{"deep":[1,2]}}],"unmodelled":null}`,
		},
		{
			name:      "create or update package",
			tool:      "functions-create-or-update-package",
			arguments: `{"NamespaceID":"ns-1","PackageName":"utils"}`,
			envelope:  "package",
			response:  `{"name":"utils","namespace":"test-ns","version":"0.0.2","unmodelled":true}`,
		},
		{
			name:      "list activations",
			tool:      "functions-list-activations",
			arguments: `{"NamespaceID":"ns-1"}`,
			envelope:  "activations",
			response: `[{"activationId":"act-1","name":"hello","namespace":"test-ns","version":"0.0.1",
				"start":1700000000000,"end":1700000000100,"duration":100,"statusCode":0,
				"annotations":[{"key":"path","value":"test-ns/hello"}],"unmodelled":"kept"}]`,
		},
		{
			name:      "get activation",
			tool:      "functions-get-activation",
			arguments: `{"NamespaceID":"ns-1","ActivationID":"act-1"}`,
			envelope:  "activation",
			response: `{"activationId":"act-1","name":"hello","namespace":"test-ns","subject":"user@example.com",
				"start":1700000000000,"end":1700000000100,"duration":100,
				"response":{"status":"success","success":true,"size":42,"result":{"body":"hello"}},
				"logs":["2024-01-01T00:00:00Z stdout: hi"],"unmodelled":{"a":1}}`,
		},
		{
			name:      "get activation logs",
			tool:      "functions-get-activation-logs",
			arguments: `{"NamespaceID":"ns-1","ActivationID":"act-1"}`,
			response:  `{"logs":["2024-01-01T00:00:00Z stdout: hi","2024-01-01T00:00:01Z stderr: oops"]}`,
		},
		{
			// The result is the invoked function's own output, so it is
			// deliberately untyped in the model. A value that would clash with
			// a declared field if it were at the root sits safely nested here.
			name:      "get activation result",
			tool:      "functions-get-activation-result",
			arguments: `{"NamespaceID":"ns-1","ActivationID":"act-1"}`,
			response:  `{"status":"success","success":true,"size":7,"result":{"name":12345,"logs":"not-an-array"}}`,
		},
		{
			name:      "invoke blocking",
			tool:      "functions-invoke-action",
			arguments: `{"NamespaceID":"ns-1","ActionName":"hello","Blocking":true}`,
			envelope:  "activation",
			response: `{"activationId":"act-1","name":"hello","namespace":"test-ns","version":"0.0.1",
				"publish":false,"subject":"user","start":1700000000000,"end":1700000000002,"duration":2,
				"logs":[],"annotations":[{"key":"waitTime","value":808},{"key":"timeout","value":false},
				{"key":"limits","value":{"logs":16,"memory":256,"timeout":3000}}],
				"response":{"status":"success","success":true,"size":19,"result":{"message":"hello"}},
				"unmodelled":"kept"}`,
		},
		{
			name:      "invoke non-blocking",
			tool:      "functions-invoke-action",
			arguments: `{"NamespaceID":"ns-1","ActionName":"hello","Blocking":false}`,
			envelope:  "activation",
			response:  `{"activationId":"act-2"}`,
		},
		{
			// name and duration would fail owActivation if this document were
			// published as an activation. Under result the schema accepts any
			// JSON, which is what a function is allowed to return.
			name:      "invoke result",
			tool:      "functions-invoke-action",
			arguments: `{"NamespaceID":"ns-1","ActionName":"hello","Result":true}`,
			envelope:  "result",
			response:  `{"message":"hello","name":12345,"duration":"not-a-number"}`,
		},
		{
			name:      "invoke result not an object",
			tool:      "functions-invoke-action",
			arguments: `{"NamespaceID":"ns-1","ActionName":"hello","Result":true}`,
			envelope:  "result",
			response:  `["not","an","object"]`,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
				w.Header().Set("Content-Type", "application/json")
				_, _ = w.Write([]byte(tc.response))
			}))
			defer ts.Close()

			resolver := newTestResolver(t, ts, "ns-1", "test-ns")

			svr := server.NewMCPServer("test", "test", server.WithOutputSchemaValidation())
			svr.AddTools(NewActionTool(resolver).Tools()...)
			svr.AddTools(NewPackageTool(resolver).Tools()...)
			svr.AddTools(NewActivationTool(resolver).Tools()...)

			ctx := context.Background()
			svr.HandleMessage(ctx, []byte(`{"jsonrpc":"2.0","id":1,"method":"initialize",`+
				`"params":{"protocolVersion":"2025-06-18","capabilities":{},"clientInfo":{"name":"t","version":"1"}}}`))

			raw, err := json.Marshal(svr.HandleMessage(ctx,
				[]byte(`{"jsonrpc":"2.0","id":2,"method":"tools/call","params":{"name":"`+tc.tool+
					`","arguments":`+tc.arguments+`}}`)))
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

			want := tc.response
			if tc.envelope != "" {
				want = `{"` + tc.envelope + `":` + tc.response + `}`
			}
			require.JSONEq(t, want, string(resp.Result.StructuredContent),
				"structuredContent should be the upstream response verbatim")

			// The text half carries the same document, re-indented.
			require.Len(t, resp.Result.Content, 1)
			require.JSONEq(t, tc.response, resp.Result.Content[0].Text)
		})
	}
}

// TestDataPlaneSchemaRejectsWrongType is the negative half of the test above.
// Publishing bytes the compiler never checked is only safe while the schema
// stays permissive, so this confirms validation is actually running and shows
// where the permissiveness stops: an undeclared field passes, but a declared
// one arriving with the wrong type does not.
//
// functions-invoke-action avoids this for its Result form by nesting that
// document under result, whose schema accepts any JSON, rather than describing
// it as an activation.
func TestDataPlaneSchemaRejectsWrongType(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		// "name" is declared as a string by owAction.
		_, _ = w.Write([]byte(`{"name":12345}`))
	}))
	defer ts.Close()

	svr := server.NewMCPServer("test", "test", server.WithOutputSchemaValidation())
	svr.AddTools(NewActionTool(newTestResolver(t, ts, "ns-1", "test-ns")).Tools()...)

	ctx := context.Background()
	svr.HandleMessage(ctx, []byte(`{"jsonrpc":"2.0","id":1,"method":"initialize",`+
		`"params":{"protocolVersion":"2025-06-18","capabilities":{},"clientInfo":{"name":"t","version":"1"}}}`))

	raw, err := json.Marshal(svr.HandleMessage(ctx,
		[]byte(`{"jsonrpc":"2.0","id":2,"method":"tools/call","params":{"name":"functions-get-action",`+
			`"arguments":{"NamespaceID":"ns-1","ActionName":"hello"}}}`)))
	require.NoError(t, err)

	var resp struct {
		Error  *struct{ Message string } `json:"error"`
		Result struct {
			IsError bool `json:"isError"`
		} `json:"result"`
	}
	require.NoError(t, json.Unmarshal(raw, &resp))
	require.True(t, resp.Error != nil || resp.Result.IsError,
		"a declared field with the wrong type should not validate: %s", raw)
}
