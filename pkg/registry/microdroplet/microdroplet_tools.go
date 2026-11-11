package microdroplet

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"unicode/utf8"

	"github.com/digitalocean/godo"
	"github.com/mark3labs/mcp-go/mcp"
	"github.com/mark3labs/mcp-go/server"

	"mcp-digitalocean/pkg/registry/common"
)

// MicroDropletTool exposes MicroDroplet lifecycle tools over the public REST API.
type MicroDropletTool struct {
	client func(ctx context.Context) (*godo.Client, error)
}

// NewMicroDropletTool creates a MicroDropletTool backed by the shared godo client.
func NewMicroDropletTool(client func(ctx context.Context) (*godo.Client, error)) *MicroDropletTool {
	return &MicroDropletTool{client: client}
}

func (t *MicroDropletTool) api(ctx context.Context) (*apiClient, error) {
	c, err := t.client(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to get DigitalOcean client: %w", err)
	}
	return newAPIClient(c), nil
}

func (t *MicroDropletTool) create(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	args := req.GetArguments()

	if strings.TrimSpace(argString(args["name"])) == "" {
		return mcp.NewToolResultError("name is required"), nil
	}

	body, err := buildCreateBody(args)
	if err != nil {
		return mcp.NewToolResultError(err.Error()), nil
	}

	api, err := t.api(ctx)
	if err != nil {
		return nil, err
	}

	out, err := api.do(ctx, http.MethodPost, "", nil, body)
	if err != nil {
		return mcp.NewToolResultErrorFromErr("microvm create", err), nil
	}
	return toolResultJSON(microVMOut, out)
}

func (t *MicroDropletTool) list(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	args := req.GetArguments()
	query := url.Values{}
	query.Set("page", fmt.Sprintf("%d", intFromArg(args["page"], 1)))
	query.Set("per_page", fmt.Sprintf("%d", intFromArg(args["per_page"], 25)))
	if v, _ := args["region"].(string); v != "" {
		query.Set("region", v)
	}
	if v, _ := args["name"].(string); v != "" {
		query.Set("name", v)
	}
	if v, _ := args["tag_name"].(string); v != "" {
		query.Set("tag_name", v)
	}

	api, err := t.api(ctx)
	if err != nil {
		return nil, err
	}

	out, err := api.do(ctx, http.MethodGet, "", query, nil)
	if err != nil {
		return mcp.NewToolResultErrorFromErr("microvm list", err), nil
	}
	return toolResultJSON(microVMsListOut, out)
}

func (t *MicroDropletTool) get(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	id, errMsg := requiredUUID(req.GetArguments(), "id")
	if errMsg != "" {
		return mcp.NewToolResultError(errMsg), nil
	}

	api, err := t.api(ctx)
	if err != nil {
		return nil, err
	}

	out, err := api.do(ctx, http.MethodGet, "/"+id, nil, nil)
	if err != nil {
		return mcp.NewToolResultErrorFromErr("microvm get", err), nil
	}
	return toolResultJSON(microVMOut, out)
}

func (t *MicroDropletTool) delete(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	id, errMsg := requiredUUID(req.GetArguments(), "id")
	if errMsg != "" {
		return mcp.NewToolResultError(errMsg), nil
	}

	api, err := t.api(ctx)
	if err != nil {
		return nil, err
	}

	if err := api.doNoContent(ctx, http.MethodDelete, "/"+id); err != nil {
		return mcp.NewToolResultErrorFromErr("microvm delete", err), nil
	}
	return mcp.NewToolResultText("MicroVM deleted successfully"), nil
}

func (t *MicroDropletTool) pause(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	id, errMsg := requiredUUID(req.GetArguments(), "id")
	if errMsg != "" {
		return mcp.NewToolResultError(errMsg), nil
	}

	api, err := t.api(ctx)
	if err != nil {
		return nil, err
	}

	out, err := api.do(ctx, http.MethodPost, "/"+id+"/pause", nil, nil)
	if err != nil {
		return mcp.NewToolResultErrorFromErr("microvm pause", err), nil
	}
	return toolResultJSON(microVMOut, out)
}

func (t *MicroDropletTool) resume(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	id, errMsg := requiredUUID(req.GetArguments(), "id")
	if errMsg != "" {
		return mcp.NewToolResultError(errMsg), nil
	}

	api, err := t.api(ctx)
	if err != nil {
		return nil, err
	}

	out, err := api.do(ctx, http.MethodPost, "/"+id+"/resume", nil, nil)
	if err != nil {
		return mcp.NewToolResultErrorFromErr("microvm resume", err), nil
	}
	return toolResultJSON(microVMOut, out)
}

func (t *MicroDropletTool) exec(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	args := req.GetArguments()
	id, errMsg := requiredUUID(args, "id")
	if errMsg != "" {
		return mcp.NewToolResultError(errMsg), nil
	}

	argv := stringSliceArg(args["argv"])
	if len(argv) == 0 {
		return mcp.NewToolResultError("argv is required and must be a non-empty array of strings"), nil
	}
	if strings.TrimSpace(argv[0]) == "" {
		return mcp.NewToolResultError("argv[0] (the command) must be non-empty"), nil
	}

	body := map[string]any{"argv": argv}
	if cwd, ok := args["cwd"].(string); ok && strings.TrimSpace(cwd) != "" {
		body["cwd"] = cwd
	}
	if enc, ok := args["output_encoding"].(string); ok && strings.TrimSpace(enc) != "" {
		switch enc {
		case "utf-8", "base64":
			body["output_encoding"] = enc
		default:
			return mcp.NewToolResultError("output_encoding must be utf-8 or base64"), nil
		}
	}

	api, err := t.api(ctx)
	if err != nil {
		return nil, err
	}

	out, err := api.do(ctx, http.MethodPost, "/"+id+"/exec", nil, body)
	if err != nil {
		return mcp.NewToolResultErrorFromErr("microvm exec", err), nil
	}
	return toolResultJSON(execOut, out)
}

// fileReadMaxBytes caps microvm-file-read so a large file cannot be pulled into
// the model's context. The API itself allows 64 MiB; doctl covers that range.
const fileReadMaxBytes = 4 << 20

func (t *MicroDropletTool) fileWrite(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	args := req.GetArguments()
	id, errMsg := requiredUUID(args, "id")
	if errMsg != "" {
		return mcp.NewToolResultError(errMsg), nil
	}
	path := strings.TrimSpace(argString(args["path"]))
	if path == "" {
		return mcp.NewToolResultError("path is required"), nil
	}

	content, _ := args["content"].(string)
	data := []byte(content)
	switch enc := argString(args["encoding"]); enc {
	case "", "utf-8":
	case "base64":
		var err error
		if data, err = base64.StdEncoding.DecodeString(content); err != nil {
			return mcp.NewToolResultError("content is not valid base64: " + err.Error()), nil
		}
	default:
		return mcp.NewToolResultError("encoding must be utf-8 or base64"), nil
	}

	uid, hasUID := intArg(args["uid"])
	gid, hasGID := intArg(args["gid"])
	if hasUID != hasGID {
		return mcp.NewToolResultError("uid and gid must be set together"), nil
	}
	if hasUID && (uid < 0 || gid < 0) {
		return mcp.NewToolResultError("uid and gid must not be negative"), nil
	}

	sum := sha256.Sum256(data)
	upload := &godo.MicroVMFileUploadRequest{
		Path:   path,
		Body:   bytes.NewReader(data),
		Size:   int64(len(data)),
		Mode:   argString(args["mode"]),
		SHA256: hex.EncodeToString(sum[:]),
	}
	if hasUID {
		u, g := uint32(uid), uint32(gid)
		upload.UID, upload.GID = &u, &g
	}

	c, err := t.client(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to get DigitalOcean client: %w", err)
	}
	info, _, err := c.MicroVMs.UploadFile(ctx, id, upload)
	if err != nil {
		return mcp.NewToolResultErrorFromErr("microvm file write", err), nil
	}
	return fileInfoOut.Result(info)
}

func (t *MicroDropletTool) fileRead(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	args := req.GetArguments()
	id, errMsg := requiredUUID(args, "id")
	if errMsg != "" {
		return mcp.NewToolResultError(errMsg), nil
	}
	path := strings.TrimSpace(argString(args["path"]))
	if path == "" {
		return mcp.NewToolResultError("path is required"), nil
	}
	enc := argString(args["encoding"])
	if enc != "" && enc != "utf-8" && enc != "base64" {
		return mcp.NewToolResultError("encoding must be utf-8 or base64"), nil
	}

	c, err := t.client(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to get DigitalOcean client: %w", err)
	}
	dl, _, err := c.MicroVMs.DownloadFile(ctx, id, path)
	if err != nil {
		return mcp.NewToolResultErrorFromErr("microvm file read", err), nil
	}
	defer dl.Body.Close()

	if dl.SizeBytes > fileReadMaxBytes {
		return mcp.NewToolResultError(fmt.Sprintf("%s is %d bytes; microvm-file-read returns at most %d bytes inline. Use `doctl compute microvm download` for larger files.", path, dl.SizeBytes, fileReadMaxBytes)), nil
	}
	// dl.Body verifies the announced SHA-256 at EOF, so a short or corrupted
	// transfer surfaces here as an error rather than as wrong content.
	data, err := io.ReadAll(io.LimitReader(dl.Body, fileReadMaxBytes+1))
	if err != nil {
		return mcp.NewToolResultErrorFromErr("microvm file read", err), nil
	}
	if len(data) > fileReadMaxBytes {
		return mcp.NewToolResultError(fmt.Sprintf("%s exceeds the %d byte inline limit. Use `doctl compute microvm download` for larger files.", path, fileReadMaxBytes)), nil
	}

	out := fileReadResponse{Path: path, SizeBytes: len(data), SHA256: dl.SHA256, Encoding: "utf-8", Content: string(data)}
	if enc == "base64" || !utf8.Valid(data) {
		out.Encoding, out.Content = "base64", base64.StdEncoding.EncodeToString(data)
	}
	return fileReadOut.Result(out)
}

func (t *MicroDropletTool) checkpointCreate(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	args := req.GetArguments()
	id, errMsg := requiredUUID(args, "id")
	if errMsg != "" {
		return mcp.NewToolResultError(errMsg), nil
	}

	var body any
	if name, ok := args["name"].(string); ok && strings.TrimSpace(name) != "" {
		body = map[string]string{"name": name}
	}

	api, err := t.api(ctx)
	if err != nil {
		return nil, err
	}

	out, err := api.do(ctx, http.MethodPost, "/"+id+"/checkpoints", nil, body)
	if err != nil {
		return mcp.NewToolResultErrorFromErr("checkpoint create", err), nil
	}
	return toolResultJSON(checkpointOut, out)
}

func (t *MicroDropletTool) checkpointList(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	args := req.GetArguments()
	query := url.Values{}
	query.Set("page", fmt.Sprintf("%d", intFromArg(args["page"], 1)))
	query.Set("per_page", fmt.Sprintf("%d", intFromArg(args["per_page"], 25)))
	if v, _ := args["microvm_id"].(string); v != "" {
		query.Set("microvm_id", v)
	}

	api, err := t.api(ctx)
	if err != nil {
		return nil, err
	}

	out, err := api.do(ctx, http.MethodGet, "/checkpoints", query, nil)
	if err != nil {
		return mcp.NewToolResultErrorFromErr("checkpoint list", err), nil
	}
	return toolResultJSON(checkpointsListOut, out)
}

func (t *MicroDropletTool) checkpointGet(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	id, errMsg := requiredUUID(req.GetArguments(), "id")
	if errMsg != "" {
		return mcp.NewToolResultError(errMsg), nil
	}

	api, err := t.api(ctx)
	if err != nil {
		return nil, err
	}

	out, err := api.do(ctx, http.MethodGet, "/checkpoints/"+id, nil, nil)
	if err != nil {
		return mcp.NewToolResultErrorFromErr("checkpoint get", err), nil
	}
	return toolResultJSON(checkpointOut, out)
}

func (t *MicroDropletTool) checkpointDelete(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	id, errMsg := requiredUUID(req.GetArguments(), "id")
	if errMsg != "" {
		return mcp.NewToolResultError(errMsg), nil
	}

	api, err := t.api(ctx)
	if err != nil {
		return nil, err
	}

	if err := api.doNoContent(ctx, http.MethodDelete, "/checkpoints/"+id); err != nil {
		return mcp.NewToolResultErrorFromErr("checkpoint delete", err), nil
	}
	return mcp.NewToolResultText("Checkpoint deleted successfully"), nil
}

func buildCreateBody(args map[string]any) (map[string]any, error) {
	source, err := parseSource(args["source"])
	if err != nil {
		return nil, err
	}

	body := map[string]any{
		"name":   args["name"],
		"source": source,
	}

	copyOptionalString(body, args, "region")
	copyOptionalString(body, args, "networking")
	copyOptionalString(body, args, "vpc_uuid")
	copyOptionalBool(body, args, "auto_resume")
	copyOptionalNumber(body, args, "http_port")
	copyOptionalString(body, args, "http_protocol")

	size, hasSize, err := parseSize(args["size"])
	if err != nil {
		return nil, err
	}
	if hasSize {
		body["size"] = size
	}

	_, hasOCI := source["oci_ref"]
	if hasOCI {
		if _, ok := body["region"]; !ok {
			return nil, fmt.Errorf("region is required when source.oci_ref is set")
		}
		if !hasSize {
			return nil, fmt.Errorf("size is required when source.oci_ref is set")
		}
	}

	if v, ok := args["auto_pause"]; ok && v != nil {
		body["auto_pause"] = v
	}
	if v, ok := args["environment"]; ok && v != nil {
		body["environment"] = v
	}
	if ports := numberSliceArg(args["ports"]); len(ports) > 0 {
		body["ports"] = ports
	}
	if tags := stringSliceArg(args["tags"]); len(tags) > 0 {
		body["tags"] = tags
	}

	return body, nil
}

func copyOptionalString(body, args map[string]any, key string) {
	if v, ok := args[key].(string); ok && v != "" {
		body[key] = v
	}
}

func copyOptionalBool(body, args map[string]any, key string) {
	if v, ok := args[key].(bool); ok {
		body[key] = v
	}
}

func copyOptionalNumber(body, args map[string]any, key string) {
	if v, ok := args[key].(float64); ok {
		body[key] = int(v)
	}
}

func parseSource(raw any) (map[string]any, error) {
	source, ok := raw.(map[string]any)
	if !ok {
		return nil, fmt.Errorf("source is required")
	}
	ociRef := strings.TrimSpace(argString(source["oci_ref"]))
	checkpointID := strings.TrimSpace(argString(source["checkpoint_id"]))
	switch {
	case ociRef == "" && checkpointID == "":
		return nil, fmt.Errorf("source must have exactly one of oci_ref or checkpoint_id")
	case ociRef != "" && checkpointID != "":
		return nil, fmt.Errorf("source must have exactly one of oci_ref or checkpoint_id, not both")
	}
	if ociRef != "" {
		return map[string]any{"oci_ref": ociRef}, nil
	}
	return map[string]any{"checkpoint_id": checkpointID}, nil
}

func parseSize(raw any) (map[string]any, bool, error) {
	if raw == nil {
		return nil, false, nil
	}
	size, ok := raw.(map[string]any)
	if !ok {
		return nil, false, fmt.Errorf("size must be an object with cpu and memory")
	}
	cpu, hasCPU := intArg(size["cpu"])
	memory, hasMemory := intArg(size["memory"])
	if !hasCPU && !hasMemory {
		return nil, false, nil
	}
	if !hasCPU || !hasMemory {
		return nil, false, fmt.Errorf("size must include cpu and memory")
	}
	return map[string]any{"cpu": cpu, "memory": memory}, true, nil
}

func argString(v any) string {
	s, _ := v.(string)
	return s
}

func intArg(v any) (int, bool) {
	switch x := v.(type) {
	case float64:
		return int(x), true
	case int:
		return x, true
	default:
		return 0, false
	}
}

func numberSliceArg(v any) []int {
	switch x := v.(type) {
	case nil:
		return nil
	case []int:
		return x
	case []any:
		var out []int
		for _, el := range x {
			n, ok := intArg(el)
			if ok {
				out = append(out, n)
			}
		}
		return out
	default:
		return nil
	}
}

func requiredUUID(args map[string]any, key string) (string, string) {
	id, _ := args[key].(string)
	id = strings.TrimSpace(id)
	if id == "" {
		return "", key + " is required"
	}
	return id, ""
}

func toolResultJSON[T any](out *common.Output[T], raw json.RawMessage) (*mcp.CallToolResult, error) {
	if len(raw) == 0 {
		raw = json.RawMessage("{}")
	}
	text := string(raw)
	var pretty json.RawMessage
	if err := json.Unmarshal(raw, &pretty); err == nil {
		if b, err := json.MarshalIndent(pretty, "", "  "); err == nil {
			text = string(b)
		}
	}
	// ResultRaw keeps the upstream document byte-for-byte in structuredContent
	// while text stays the indented form clients already consumed.
	return out.ResultRaw(text, raw), nil
}

func intFromArg(v any, def int) int {
	switch x := v.(type) {
	case nil:
		return def
	case float64:
		if x >= 1 {
			return int(x)
		}
		return def
	case int:
		if x >= 1 {
			return x
		}
		return def
	default:
		return def
	}
}

func stringSliceArg(v any) []string {
	switch x := v.(type) {
	case nil:
		return nil
	case []string:
		return x
	case []any:
		var out []string
		for _, el := range x {
			if s, ok := el.(string); ok {
				out = append(out, s)
			}
		}
		return out
	default:
		return nil
	}
}

// Tools registers MCP tools for MicroVM lifecycle management.
func (t *MicroDropletTool) Tools() []server.ServerTool {
	return []server.ServerTool{
		{
			Handler: t.create,
			Tool: mcp.NewTool("microvm-create",
				common.WithHints(common.HintsCreate),
				common.WithRisk(common.RiskHigh),
				microVMOut.Schema(),
				mcp.WithDescription("Create a MicroVM. source must have exactly one of oci_ref or checkpoint_id. region and size are required for oci_ref; both are inherited from the checkpoint when omitted."),
				mcp.WithString("name", mcp.Required(), mcp.Description("Human-readable name.")),
				mcp.WithObject("source", mcp.Required(), mcp.Description("Workload source. Exactly one of oci_ref or checkpoint_id."),
					mcp.Properties(map[string]any{
						"oci_ref": map[string]any{
							"type":        "string",
							"description": "OCI reference (public Hub or team DOCR), pulled at boot.",
						},
						"checkpoint_id": map[string]any{
							"type":        "string",
							"description": "Checkpoint UUID to restore. Mutually exclusive with oci_ref.",
						},
					})),
				mcp.WithString("region", mcp.Description("Region slug (e.g. nyc1). Required with oci_ref.")),
				mcp.WithObject("size", mcp.Description("Compute capacity. Required with oci_ref; must match the checkpoint if supplied on restore."),
					mcp.Properties(map[string]any{
						"cpu":    map[string]any{"type": "integer", "description": "vCPU count."},
						"memory": map[string]any{"type": "integer", "description": "Memory in MiB."},
					})),
				mcp.WithString("networking", mcp.Enum("public", "vpc"), mcp.Description("Network posture: public or vpc.")),
				mcp.WithString("vpc_uuid", mcp.Description("Required when networking is vpc.")),
				mcp.WithObject("auto_pause", mcp.Description("Auto-pause config: enabled (bool), idle_timeout (duration string).")),
				mcp.WithBoolean("auto_resume", mcp.Description("Whether to auto-resume on HTTP traffic (default true).")),
				mcp.WithNumber("http_port", mcp.Description("HTTP listen port for ingress and auto-resume probing.")),
				mcp.WithString("http_protocol", mcp.Enum("http", "http2"), mcp.Description("Application protocol on http_port: http or http2.")),
				mcp.WithArray("ports", mcp.Description("Guest ports to open for ingress (max 5). Must include http_port when set."), mcp.Items(map[string]any{"type": "integer"})),
				mcp.WithObject("environment", mcp.Description("Key-value environment variables injected at boot.")),
				mcp.WithArray("tags", mcp.Description("Resource tags."), mcp.Items(map[string]any{"type": "string"})),
			),
		},
		{
			Handler: t.list,
			Tool: mcp.NewTool("microvm-list",
				common.WithHints(common.HintsRead),
				common.WithRisk(common.RiskLow),
				microVMsListOut.Schema(),
				mcp.WithDescription("List MicroVMs for the authenticated team."),
				mcp.WithNumber("page", mcp.DefaultNumber(1), mcp.Description("Page number (default 1).")),
				mcp.WithNumber("per_page", mcp.DefaultNumber(25), mcp.Description("Items per page (default 25, max 200).")),
				mcp.WithString("region", mcp.Description("Filter by region slug.")),
				mcp.WithString("name", mcp.Description("Filter by name.")),
				mcp.WithString("tag_name", mcp.Description("Filter by resource tag name.")),
			),
		},
		{
			Handler: t.get,
			Tool: mcp.NewTool("microvm-get",
				common.WithHints(common.HintsRead),
				common.WithRisk(common.RiskLow),
				microVMOut.Schema(),
				mcp.WithDescription("Get a MicroVM by ID."),
				mcp.WithString("id", mcp.Required(), mcp.Description("MicroVM UUID.")),
			),
		},
		{
			Handler: t.delete,
			Tool: mcp.NewTool("microvm-delete",
				common.WithHints(common.HintsDelete),
				common.WithRisk(common.RiskHigh),
				mcp.WithDescription("Delete a MicroVM. Irreversible; checkpoints are retained until deleted separately."),
				mcp.WithString("id", mcp.Required(), mcp.Description("MicroVM UUID.")),
			),
		},
		{
			Handler: t.pause,
			Tool: mcp.NewTool("microvm-pause",
				common.WithHints(common.HintsToggle),
				common.WithRisk(common.RiskMedium),
				microVMOut.Schema(),
				mcp.WithDescription("Pause a running MicroVM (synchronous, idempotent)."),
				mcp.WithString("id", mcp.Required(), mcp.Description("MicroVM UUID.")),
			),
		},
		{
			Handler: t.resume,
			Tool: mcp.NewTool("microvm-resume",
				common.WithHints(common.HintsToggle),
				common.WithRisk(common.RiskMedium),
				microVMOut.Schema(),
				mcp.WithDescription("Resume a paused MicroVM (synchronous, idempotent)."),
				mcp.WithString("id", mcp.Required(), mcp.Description("MicroVM UUID.")),
			),
		},
		{
			Handler: t.exec,
			Tool: mcp.NewTool("microvm-exec",
				common.WithHints(common.HintsAction),
				common.WithRisk(common.RiskHigh),
				execOut.Schema(),
				mcp.WithDescription("Run a one-shot non-interactive command in a MicroVM. Requires the exec_pty feature gate. Returns stdout, stderr, and exit_code; non-zero exit is still a successful tool call. Commands over about 50 seconds may time out. Interactive console sessions are not available as an MCP tool."),
				mcp.WithString("id", mcp.Required(), mcp.Description("MicroVM UUID.")),
				mcp.WithArray("argv", mcp.Required(), mcp.Description("Command and arguments (e.g. [\"echo\", \"hi\"]). Must be non-empty."), mcp.Items(map[string]any{"type": "string"})),
				mcp.WithString("cwd", mcp.Description("Optional working directory.")),
				mcp.WithString("output_encoding", mcp.Enum("utf-8", "base64"), mcp.Description("How stdout/stderr are encoded (default utf-8). Use base64 for binary output.")),
			),
		},
		{
			Handler: t.fileWrite,
			Tool: mcp.NewTool("microvm-file-write",
				common.WithHints(common.HintsReplace),
				common.WithRisk(common.RiskHigh),
				fileInfoOut.Schema(),
				mcp.WithDescription("Write a single file into a MicroVM's workload container (PUT /v2/microvms/{id}/files). Replaces any existing file at path; the parent directory must exist. Requires the file_transfer feature gate. A paused MicroVM is auto-resumed. Content is sent with its SHA-256 so the API verifies it before replacing the target. Max 64 MiB; for large or binary files prefer `doctl compute microvm upload`."),
				mcp.WithString("id", mcp.Required(), mcp.Description("MicroVM UUID.")),
				mcp.WithString("path", mcp.Required(), mcp.Description("Absolute destination path inside the workload container, e.g. /app/config.json.")),
				mcp.WithString("content", mcp.Required(), mcp.Description("File content. Text by default; set encoding to base64 for binary.")),
				mcp.WithString("encoding", mcp.Enum("utf-8", "base64"), mcp.Description("How content is encoded (default utf-8).")),
				mcp.WithString("mode", mcp.Description("Octal permission mode such as 0600 (default 0644).")),
				mcp.WithNumber("uid", mcp.Description("Owner user ID. Set together with gid; defaults to the container's user.")),
				mcp.WithNumber("gid", mcp.Description("Owner group ID. Set together with uid; defaults to the container's group.")),
			),
		},
		{
			Handler: t.fileRead,
			Tool: mcp.NewTool("microvm-file-read",
				common.WithHints(common.HintsRead),
				common.WithRisk(common.RiskLow),
				fileReadOut.Schema(),
				mcp.WithDescription("Read a single regular file from a MicroVM's workload container (GET /v2/microvms/{id}/files). Requires the file_transfer feature gate. A paused MicroVM is auto-resumed. Content is verified against the server's SHA-256. Returns at most 4 MiB inline; larger files error and should be fetched with `doctl compute microvm download`. Non-UTF-8 content is returned as base64 with encoding set accordingly."),
				mcp.WithString("id", mcp.Required(), mcp.Description("MicroVM UUID.")),
				mcp.WithString("path", mcp.Required(), mcp.Description("Absolute path of the file inside the workload container.")),
				mcp.WithString("encoding", mcp.Enum("utf-8", "base64"), mcp.Description("Requested content encoding (default utf-8; falls back to base64 for binary content).")),
			),
		},
		{
			Handler: t.checkpointCreate,
			Tool: mcp.NewTool("microvm-checkpoint-create",
				common.WithHints(common.HintsCreate),
				common.WithRisk(common.RiskMedium),
				checkpointOut.Schema(),
				mcp.WithDescription("Start async checkpoint of a running or paused MicroVM. Poll checkpoint-get until available or failed."),
				mcp.WithString("id", mcp.Required(), mcp.Description("MicroVM UUID.")),
				mcp.WithString("name", mcp.Description("Optional checkpoint name.")),
			),
		},
		{
			Handler: t.checkpointList,
			Tool: mcp.NewTool("microvm-checkpoint-list",
				common.WithHints(common.HintsRead),
				common.WithRisk(common.RiskLow),
				checkpointsListOut.Schema(),
				mcp.WithDescription("List team checkpoints, newest first."),
				mcp.WithNumber("page", mcp.DefaultNumber(1), mcp.Description("Page number (default 1).")),
				mcp.WithNumber("per_page", mcp.DefaultNumber(25), mcp.Description("Items per page (default 25, max 200).")),
				mcp.WithString("microvm_id", mcp.Description("Filter by source MicroVM UUID.")),
			),
		},
		{
			Handler: t.checkpointGet,
			Tool: mcp.NewTool("microvm-checkpoint-get",
				common.WithHints(common.HintsRead),
				common.WithRisk(common.RiskLow),
				checkpointOut.Schema(),
				mcp.WithDescription("Get a checkpoint by ID."),
				mcp.WithString("id", mcp.Required(), mcp.Description("Checkpoint UUID.")),
			),
		},
		{
			Handler: t.checkpointDelete,
			Tool: mcp.NewTool("microvm-checkpoint-delete",
				common.WithHints(common.HintsDelete),
				common.WithRisk(common.RiskHigh),
				mcp.WithDescription("Delete a checkpoint. Irreversible."),
				mcp.WithString("id", mcp.Required(), mcp.Description("Checkpoint UUID.")),
			),
		},
	}
}
