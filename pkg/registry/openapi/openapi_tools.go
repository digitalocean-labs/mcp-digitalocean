package openapi

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"sort"
	"strings"

	"github.com/digitalocean/godo"
	"github.com/getkin/kin-openapi/openapi3"
	"github.com/getkin/kin-openapi/openapi3filter"
	"github.com/getkin/kin-openapi/routers"
	"github.com/mark3labs/mcp-go/mcp"
	"github.com/mark3labs/mcp-go/server"
)

const (
	defaultSearchLimit = 10

	// maxResponseBodyBytes caps bytes read from the API response body into memory for openapi execute tools.
	maxResponseBodyBytes = 1 << 20

	errDeleteUseOtherTool = "DELETE operations must use the openapi-execute-delete tool (destructive; host clients should require user approval)."
	errNonDeleteUseOther  = "openapi-execute-delete only accepts DELETE operations; use openapi-execute for %s."

	toolDescOpenAPISearch = "Search the embedded DigitalOcean public API OpenAPI document. Matches substrings in operationId, summary, path, HTTP method, and tags. Typical flow: openapi-search → openapi-get-operation (pick an operationId from the results) → openapi-execute or openapi-execute-delete. Raise Limit if you need more candidates."

	toolDescOpenAPIGetOperation = "Return full OpenAPI details for one operationId: method, path, parameters (names, location path/query/header, required flags, schemas), request body outline, and responses. Call this before openapi-execute or openapi-execute-delete so Parameters and Body match the spec; validation errors usually mean wrong names, types, or missing required parameters."

	toolDescOpenAPIExecute = "Validate the request against the embedded spec, then execute it via the configured DigitalOcean API client (a valid API token or equivalent credentials must be available where this MCP server expects them). Non-DELETE HTTP methods only—use openapi-execute-delete for DELETE. Workflow: openapi-get-operation for this operationId first. Request bodies are sent as application/json only—operations that require multipart or other content types are rejected with the MIME types the spec allows. Cookie parameters from the spec are not supported."

	toolDescOpenAPIExecuteDelete = "Same validation and execution path as openapi-execute but only for DELETE operations. Requires valid API credentials. Destructive — MCP hosts may require explicit approval (destructiveHint). Call openapi-get-operation first. Request bodies are application/json only when a body is sent. Cookie parameters are not supported."

	argDescSearchQuery = "Free-text keywords (e.g. API concepts like droplet, firewall, Kubernetes, or a path fragment). Matches substrings across operationId, summary, path, method, and tags."

	argDescSearchTag = "If non-empty, only operations whose OpenAPI tags contain this exact string (case-sensitive). Leave empty to search all tags."

	argDescSearchLimit = "Maximum operations to return (default 10). Increase if results are truncated or you need more choices."

	argDescOperationIDFromSearch = "The operationId string exactly as shown in openapi-search results (or from DigitalOcean API documentation)."

	argDescOperationIDForExecute = "The operationId you inspected with openapi-get-operation; must exist in the embedded spec."

	argDescExecuteParameters = "Maps OpenAPI parameter names to values from openapi-get-operation. Values may be string, boolean, number, or an array of those for repeated query parameters. Each path parameter must resolve to a single value. Header, query, and path parameters share this one object; placement follows the spec. Omit optional parameters you are not setting."

	argDescExecuteBody = "JSON object for the request body when the operation defines one; omit this argument entirely when there is no body. Shape must match the schema described by openapi-get-operation."

	argDescExecuteSelect = "Optional dotted path to narrow a JSON response body after the request succeeds (only when Content-Type is application/json). Examples: droplets[*].id for an array of ids, links.pages.next for a nested string. Leave empty to return the full body (still subject to the 1 MiB cap)."
)

// GodoClientFn resolves the DigitalOcean API client for the current MCP request
// (stdio: shared client; HTTP streamable: client built from the Authorization header).
type GodoClientFn = func(ctx context.Context) (*godo.Client, error)

// Options configures openapi tool registration and behavior.
type Options struct {
	// DisableDeletes, when true, skips registering openapi-execute-delete (OPENAPI_DISABLE_DELETES / --openapi-disable-deletes).
	DisableDeletes bool
}

// OpenAPITool registers the openapi-search, openapi-get-operation, openapi-execute,
// and optionally openapi-execute-delete MCP tools.
type OpenAPITool struct {
	getClient GodoClientFn
	api       *OpenAPIClient
	opts      Options
}

// NewOpenAPITool builds an OpenAPITool. getClient must be non-nil; the embedded
// spec loads lazily on the first search, get, or execute call.
func NewOpenAPITool(getClient GodoClientFn, opts Options) (*OpenAPITool, error) {
	if getClient == nil {
		return nil, fmt.Errorf("getClient is required")
	}
	return &OpenAPITool{
		getClient: getClient,
		api:       NewOpenAPIClient(),
		opts:      opts,
	}, nil
}

// search implements the openapi-search tool handler.
func (t *OpenAPITool) search(_ context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	args := req.GetArguments()

	query, ok := args["Query"].(string)
	if !ok || strings.TrimSpace(query) == "" {
		return mcp.NewToolResultError("Query is required and must be a non-empty string"), nil
	}

	limit := defaultSearchLimit
	if lim, ok := args["Limit"].(float64); ok && lim > 0 {
		limit = int(lim)
	}

	tag := ""
	if s, ok := args["Tag"].(string); ok {
		tag = s
	}

	results, err := t.api.SearchOperations(query, tag, limit)
	if err != nil {
		return mcp.NewToolResultErrorFromErr("openapi search failed", err), nil
	}

	if len(results) == 0 {
		fallback := fmt.Sprintf("No operations matched %q.", query)
		return mcp.NewToolResultStructured(SearchToolResult{Hits: []OperationSummary{}}, fallback), nil
	}

	var b strings.Builder
	fmt.Fprintf(&b, "Found %d operation(s) matching %q:\n\n", len(results), query)
	hits := make([]OperationSummary, 0, len(results))
	for i, r := range results {
		fmt.Fprintf(&b, "%d. %s %s  [%s]\n", i+1, r.Method, r.Path, r.OperationID)
		if r.Summary != "" {
			fmt.Fprintf(&b, "   %s\n", r.Summary)
		}
		if len(r.Tags) > 0 {
			fmt.Fprintf(&b, "   tags: %s\n", strings.Join(r.Tags, ", "))
		}
		b.WriteByte('\n')
		hits = append(hits, *r)
	}

	payload := SearchToolResult{Hits: hits}
	return mcp.NewToolResultStructured(payload, strings.TrimSpace(b.String())), nil
}

// getOperation implements the openapi-get-operation tool handler.
func (t *OpenAPITool) getOperation(_ context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	args := req.GetArguments()
	id, ok := args["OperationID"].(string)
	if !ok || strings.TrimSpace(id) == "" {
		return mcp.NewToolResultError("OperationID is required and must be a non-empty string"), nil
	}

	op, err := t.api.GetOperation(strings.TrimSpace(id))
	if err != nil {
		return mcp.NewToolResultErrorFromErr("failed to load operation", err), nil
	}

	fallback := formatOperation(op)
	payload := GetOperationToolResult{Operation: *op}
	return mcp.NewToolResultStructured(payload, fallback), nil
}

// formatOperation renders Operation as markdown-style text for the model.
func formatOperation(op *Operation) string {
	var b strings.Builder
	fmt.Fprintf(&b, "**%s** `%s`\n\n", op.Method, op.Path)
	if op.Summary != "" {
		fmt.Fprintf(&b, "%s\n\n", op.Summary)
	}
	if op.Description != "" {
		fmt.Fprintf(&b, "%s\n\n", op.Description)
	}
	if len(op.Tags) > 0 {
		fmt.Fprintf(&b, "**Tags:** %s\n\n", strings.Join(op.Tags, ", "))
	}

	if len(op.Parameters) > 0 {
		b.WriteString("### Parameters\n\n")
		for _, p := range op.Parameters {
			req := "optional"
			if p.Required {
				req = "required"
			}
			schema := p.Schema
			if schema == "" {
				schema = "unknown"
			}
			fmt.Fprintf(&b, "- %s (%s, %s, %s)", p.Name, p.In, schema, req)
			if p.Description != "" {
				fmt.Fprintf(&b, " — %s", p.Description)
			}
			b.WriteByte('\n')
		}
		b.WriteByte('\n')
	}

	if op.RequestBody != "" {
		b.WriteString("### Request body\n\n")
		b.WriteString(op.RequestBody)
		b.WriteString("\n\n")
	}

	if len(op.Responses) > 0 {
		b.WriteString("### Responses\n\n")
		codes := make([]string, 0, len(op.Responses))
		for code := range op.Responses {
			codes = append(codes, code)
		}
		sort.Strings(codes)
		for _, code := range codes {
			desc := op.Responses[code]
			line := desc
			if strings.TrimSpace(line) == "" {
				line = "(no description)"
			}
			fmt.Fprintf(&b, "- **%s:** %s\n", code, strings.TrimSpace(line))
		}
	}

	return strings.TrimSpace(b.String())
}

func parseExecuteArgs(args map[string]any) (id string, params map[string]any, body map[string]any, selectPath string, err error) {
	idRaw, ok := args["OperationID"].(string)
	if !ok || strings.TrimSpace(idRaw) == "" {
		return "", nil, nil, "", errors.New("OperationID is required and must be a non-empty string")
	}
	id = strings.TrimSpace(idRaw)

	if raw, ok := args["Parameters"].(map[string]any); ok && raw != nil {
		params = raw
	}
	if raw, ok := args["Body"].(map[string]any); ok && raw != nil {
		body = raw
	}
	if s, ok := args["Select"].(string); ok {
		selectPath = strings.TrimSpace(s)
	}
	return id, params, body, selectPath, nil
}

// execute implements openapi-execute: kin-openapi request validation against the spec's
// first server URL, then godo.Client.Do against the client's configured API base URL.
// DELETE operations are rejected; use openapi-execute-delete.
func (t *OpenAPITool) execute(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	id, params, body, selectPath, err := parseExecuteArgs(req.GetArguments())
	if err != nil {
		return mcp.NewToolResultError(err.Error()), nil
	}

	idx, err := t.api.getIndexed(id)
	if err != nil {
		return mcp.NewToolResultErrorFromErr("unknown operation", err), nil
	}

	method := strings.ToUpper(idx.method)
	if method == http.MethodDelete {
		return mcp.NewToolResultError(errDeleteUseOtherTool), nil
	}

	return t.runOperation(ctx, idx, params, body, selectPath)
}

// executeDelete implements openapi-execute-delete: same validation and execution as
// openapi-execute but only for DELETE operations (destructiveHint in tool metadata).
func (t *OpenAPITool) executeDelete(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	id, params, body, selectPath, err := parseExecuteArgs(req.GetArguments())
	if err != nil {
		return mcp.NewToolResultError(err.Error()), nil
	}

	idx, err := t.api.getIndexed(id)
	if err != nil {
		return mcp.NewToolResultErrorFromErr("unknown operation", err), nil
	}

	method := strings.ToUpper(idx.method)
	if method != http.MethodDelete {
		return mcp.NewToolResultError(fmt.Sprintf(errNonDeleteUseOther, method)), nil
	}

	return t.runOperation(ctx, idx, params, body, selectPath)
}

// errIfJSONBodyUnsupported returns an error when the caller supplied a JSON body but the
// operation's requestBody does not define application/json (execute tools only send JSON).
func errIfJSONBodyUnsupported(op *openapi3.Operation, hasJSONBody bool) error {
	if !hasJSONBody {
		return nil
	}
	if op.RequestBody == nil || op.RequestBody.Value == nil {
		return nil
	}
	content := op.RequestBody.Value.Content
	if len(content) == 0 {
		return nil
	}
	if _, ok := content["application/json"]; ok {
		return nil
	}
	types := make([]string, 0, len(content))
	for ct := range content {
		types = append(types, ct)
	}
	sort.Strings(types)
	return fmt.Errorf("this operation does not accept application/json request bodies (OpenAPI allows: %s); openapi execute tools only send JSON bodies", strings.Join(types, ", "))
}

// collectParams maps MCP Parameters to path, query, and header values per the OpenAPI operation.
func collectParams(idx *indexedOp, params map[string]any) (pathParams map[string]string, queryVals url.Values, headerVals http.Header, err error) {
	pathParams = make(map[string]string)
	queryVals = url.Values{}
	headerVals = http.Header{}

	allParams := mergeParameters(idx.pathItem, idx.op)
	for _, p := range allParams {
		if p == nil {
			continue
		}
		rawVal, has := params[p.Name]
		if !has || rawVal == nil {
			continue
		}
		strVals, err := paramToStrings(p.Name, rawVal)
		if err != nil {
			return nil, nil, nil, err
		}
		switch p.In {
		case openapi3.ParameterInPath:
			if len(strVals) != 1 {
				return nil, nil, nil, fmt.Errorf("path parameter %q expects a single value", p.Name)
			}
			pathParams[p.Name] = strVals[0]
		case openapi3.ParameterInQuery:
			for _, s := range strVals {
				queryVals.Add(p.Name, s)
			}
		case openapi3.ParameterInHeader:
			for _, s := range strVals {
				headerVals.Add(p.Name, s)
			}
		case openapi3.ParameterInCookie:
			return nil, nil, nil, fmt.Errorf("cookie parameter %q is not supported by openapi execute tools", p.Name)
		default:
			return nil, nil, nil, fmt.Errorf("unsupported parameter location %q for %q", p.In, p.Name)
		}
	}
	return pathParams, queryVals, headerVals, nil
}

// buildHTTPRequest builds an *http.Request against baseTrimmed+resolvedPath with query, optional JSON body, and headers.
func buildHTTPRequest(method, baseTrimmed, resolvedPath string, queryVals url.Values, jsonBody []byte, headerVals http.Header) (*http.Request, error) {
	fullURL := baseTrimmed + resolvedPath
	u, err := url.Parse(fullURL)
	if err != nil {
		return nil, err
	}
	u.RawQuery = queryVals.Encode()

	var bodyReader io.Reader = http.NoBody
	if len(jsonBody) > 0 {
		bodyReader = bytes.NewReader(jsonBody)
	}

	req, err := http.NewRequest(method, u.String(), bodyReader)
	if err != nil {
		return nil, err
	}
	if len(jsonBody) > 0 {
		req.Header.Set("Content-Type", "application/json")
		req.ContentLength = int64(len(jsonBody))
	}
	for k, vals := range headerVals {
		for _, v := range vals {
			req.Header.Add(k, v)
		}
	}
	return req, nil
}

func validateAgainstSpec(ctx context.Context, doc *openapi3.T, idx *indexedOp, method string, validationReq *http.Request, pathParams map[string]string, queryVals url.Values) error {
	route := &routers.Route{
		Spec:      doc,
		Server:    doc.Servers[0],
		Path:      idx.path,
		PathItem:  idx.pathItem,
		Method:    method,
		Operation: idx.op,
	}
	vinput := &openapi3filter.RequestValidationInput{
		Request:     validationReq,
		PathParams:  pathParams,
		QueryParams: queryVals,
		Route:       route,
		Options: &openapi3filter.Options{
			AuthenticationFunc: openapi3filter.NoopAuthenticationFunc,
		},
	}
	return openapi3filter.ValidateRequest(ctx, vinput)
}

// limitWriter writes at most remaining bytes to dst; further writes succeed without storing (truncated=true).
type limitWriter struct {
	dst         io.Writer
	remaining   int64
	truncated bool
}

func (w *limitWriter) Write(p []byte) (int, error) {
	if w.remaining <= 0 {
		w.truncated = true
		return len(p), nil
	}
	if int64(len(p)) <= w.remaining {
		n, err := w.dst.Write(p)
		w.remaining -= int64(n)
		return n, err
	}
	n, err := w.dst.Write(p[:w.remaining])
	w.remaining = 0
	w.truncated = true
	if err != nil {
		return n, err
	}
	return len(p), nil
}

var forwardResponseHeaderKeys = []string{
	"Content-Type",
	"Ratelimit-Limit",
	"Ratelimit-Remaining",
	"Ratelimit-Reset",
	"Retry-After",
	"Link",
}

func appendForwardedResponseHeaders(b *strings.Builder, h http.Header) {
	if h == nil {
		return
	}
	for _, key := range forwardResponseHeaderKeys {
		if v := h.Get(key); v != "" {
			fmt.Fprintf(b, "%s: %s\n", strings.ToLower(key), v)
		}
	}
}

func formatExecuteResponseBody(contentType string, raw []byte) string {
	ct := strings.ToLower(strings.TrimSpace(contentType))
	if strings.HasPrefix(ct, "application/json") || strings.HasPrefix(ct, "text/") {
		return strings.TrimSpace(string(raw))
	}
	return string(raw)
}

func (t *OpenAPITool) executeAndFormat(ctx context.Context, client *godo.Client, method, execBase, resolvedPath string, queryVals url.Values, jsonBody []byte, headerVals http.Header, selectPath string) (*mcp.CallToolResult, error) {
	execReq, err := buildHTTPRequest(method, execBase, resolvedPath, queryVals, jsonBody, headerVals)
	if err != nil {
		return mcp.NewToolResultErrorFromErr("failed to build API request", err), nil
	}

	var respBuf bytes.Buffer
	lw := &limitWriter{dst: &respBuf, remaining: maxResponseBodyBytes}
	resp, err := client.Do(ctx, execReq, lw)
	if err != nil {
		return mcp.NewToolResultErrorFromErr("API request failed", err), nil
	}
	if resp == nil || resp.Response == nil {
		return mcp.NewToolResultError("empty response from API client"), nil
	}

	ct := ""
	if resp.Response != nil {
		ct = resp.Header.Get("Content-Type")
	}

	bodyBytes := respBuf.Bytes()
	bodyStr := formatExecuteResponseBody(ct, bodyBytes)
	bodyAny := any(bodyStr)
	ctLower := strings.ToLower(strings.TrimSpace(ct))
	selectNA := strings.TrimSpace(selectPath) != "" && !strings.HasPrefix(ctLower, "application/json")

	if strings.HasPrefix(ctLower, "application/json") {
		var parsed any
		if err := json.Unmarshal(bodyBytes, &parsed); err != nil {
			if strings.TrimSpace(selectPath) != "" {
				return mcp.NewToolResultErrorFromErr("response body is not valid JSON for Select", err), nil
			}
		} else {
			bodyAny = parsed
			if strings.TrimSpace(selectPath) != "" {
				shaped, err := selectJSON(selectPath, parsed)
				if err != nil {
					return mcp.NewToolResultError(err.Error()), nil
				}
				bodyAny = shaped
				enc, err := json.MarshalIndent(shaped, "", "  ")
				if err != nil {
					return mcp.NewToolResultErrorFromErr("failed to encode selected JSON", err), nil
				}
				bodyStr = string(enc)
			}
		}
	}

	payload := ExecuteToolResult{
		Status:           resp.StatusCode,
		ResponseHeaders:  forwardedHeadersMap(resp.Header),
		Body:             bodyAny,
		Truncated:        lw.truncated,
		SelectNotApplied: selectNA,
	}

	var b strings.Builder
	fmt.Fprintf(&b, "status: %d\n", resp.StatusCode)
	appendForwardedResponseHeaders(&b, resp.Header)
	if selectNA {
		fmt.Fprintf(&b, "select-not-applied: response Content-Type is not JSON\n")
	}
	b.WriteByte('\n')
	b.WriteString(bodyStr)
	if lw.truncated {
		b.WriteString("\n\n[truncated: response exceeded 1 MiB; tighten parameters or use Select]")
	}

	fallback := strings.TrimRight(b.String(), "\n")
	return mcp.NewToolResultStructured(payload, fallback), nil
}

func (t *OpenAPITool) runOperation(ctx context.Context, idx *indexedOp, params map[string]any, body map[string]any, selectPath string) (*mcp.CallToolResult, error) {
	method := strings.ToUpper(idx.method)

	doc, err := t.api.document()
	if err != nil {
		return mcp.NewToolResultErrorFromErr("openapi document unavailable", err), nil
	}

	specBase, err := serverBase(doc)
	if err != nil {
		return mcp.NewToolResultErrorFromErr("invalid servers in openapi spec", err), nil
	}

	client, err := t.getClient(ctx)
	if err != nil {
		return mcp.NewToolResultErrorFromErr("failed to resolve API client", err), nil
	}

	pathParams, queryVals, headerVals, err := collectParams(idx, params)
	if err != nil {
		return mcp.NewToolResultError(err.Error()), nil
	}

	resolvedPath, err := substitutePath(idx.path, pathParams)
	if err != nil {
		return mcp.NewToolResultError(err.Error()), nil
	}

	var jsonBody []byte
	if len(body) > 0 {
		jsonBody, err = json.Marshal(body)
		if err != nil {
			return mcp.NewToolResultErrorFromErr("failed to encode JSON body", err), nil
		}
	}

	if err := errIfJSONBodyUnsupported(idx.op, len(jsonBody) > 0); err != nil {
		return mcp.NewToolResultError(err.Error()), nil
	}

	validationReq, err := buildHTTPRequest(method, specBase, resolvedPath, queryVals, jsonBody, headerVals)
	if err != nil {
		return mcp.NewToolResultErrorFromErr("invalid validation URL", err), nil
	}

	if err := validateAgainstSpec(ctx, doc, idx, method, validationReq, pathParams, queryVals); err != nil {
		return mcp.NewToolResultError(humanizeRequestValidationError(err)), nil
	}

	execBase := strings.TrimSuffix(strings.TrimSpace(client.BaseURL.String()), "/")
	return t.executeAndFormat(ctx, client, method, execBase, resolvedPath, queryVals, jsonBody, headerVals, selectPath)
}

// Tools returns openapi-search, openapi-get-operation, openapi-execute, and
// openapi-execute-delete (unless DisableDeletes is set) as server.ServerTool entries.
func (t *OpenAPITool) Tools() []server.ServerTool {
	readOnlyMeta := []mcp.ToolOption{
		mcp.WithReadOnlyHintAnnotation(true),
		mcp.WithDestructiveHintAnnotation(false),
		mcp.WithOpenWorldHintAnnotation(false),
	}
	executeMeta := []mcp.ToolOption{
		mcp.WithReadOnlyHintAnnotation(false),
		mcp.WithDestructiveHintAnnotation(false),
		mcp.WithIdempotentHintAnnotation(false),
		mcp.WithOpenWorldHintAnnotation(true),
	}
	deleteMeta := []mcp.ToolOption{
		mcp.WithReadOnlyHintAnnotation(false),
		mcp.WithDestructiveHintAnnotation(true),
		mcp.WithIdempotentHintAnnotation(true),
		mcp.WithOpenWorldHintAnnotation(true),
	}

	searchOpts := append([]mcp.ToolOption{
		mcp.WithTitleAnnotation("OpenAPI: Search"),
		mcp.WithDescription(toolDescOpenAPISearch),
	}, readOnlyMeta...)
	searchOpts = append(searchOpts,
		mcp.WithString("Query", mcp.Required(), mcp.Description(argDescSearchQuery)),
		mcp.WithString("Tag", mcp.Description(argDescSearchTag)),
		mcp.WithNumber("Limit", mcp.DefaultNumber(defaultSearchLimit), mcp.Description(argDescSearchLimit)),
		mcp.WithOutputSchema[SearchToolResult](),
	)

	getOpOpts := append([]mcp.ToolOption{
		mcp.WithTitleAnnotation("OpenAPI: Get operation"),
		mcp.WithDescription(toolDescOpenAPIGetOperation),
	}, readOnlyMeta...)
	getOpOpts = append(getOpOpts,
		mcp.WithString("OperationID", mcp.Required(), mcp.Description(argDescOperationIDFromSearch)),
		mcp.WithOutputSchema[GetOperationToolResult](),
	)

	execOpts := append([]mcp.ToolOption{
		mcp.WithTitleAnnotation("OpenAPI: Execute"),
		mcp.WithDescription(toolDescOpenAPIExecute),
	}, executeMeta...)
	execOpts = append(execOpts,
		mcp.WithString("OperationID", mcp.Required(), mcp.Description(argDescOperationIDForExecute)),
		mcp.WithObject("Parameters", mcp.Description(argDescExecuteParameters)),
		mcp.WithObject("Body", mcp.Description(argDescExecuteBody)),
		mcp.WithString("Select", mcp.Description(argDescExecuteSelect)),
		mcp.WithOutputSchema[ExecuteToolResult](),
	)

	out := []server.ServerTool{
		{Handler: t.search, Tool: mcp.NewTool("openapi-search", searchOpts...)},
		{Handler: t.getOperation, Tool: mcp.NewTool("openapi-get-operation", getOpOpts...)},
		{Handler: t.execute, Tool: mcp.NewTool("openapi-execute", execOpts...)},
	}

	if !t.opts.DisableDeletes {
		delOpts := append([]mcp.ToolOption{
			mcp.WithTitleAnnotation("OpenAPI: Execute DELETE"),
			mcp.WithDescription(toolDescOpenAPIExecuteDelete),
		}, deleteMeta...)
		delOpts = append(delOpts,
			mcp.WithString("OperationID", mcp.Required(), mcp.Description(argDescOperationIDForExecute)),
			mcp.WithObject("Parameters", mcp.Description(argDescExecuteParameters)),
			mcp.WithObject("Body", mcp.Description(argDescExecuteBody)),
			mcp.WithString("Select", mcp.Description(argDescExecuteSelect)),
			mcp.WithOutputSchema[ExecuteToolResult](),
		)
		out = append(out, server.ServerTool{
			Handler: t.executeDelete,
			Tool:    mcp.NewTool("openapi-execute-delete", delOpts...),
		})
	}

	return out
}
