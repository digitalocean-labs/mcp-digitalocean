package sandbox

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strconv"
	"unicode/utf8"

	"github.com/mark3labs/mcp-go/mcp"
	"github.com/mark3labs/mcp-go/server"

	"mcp-digitalocean/pkg/registry/common"
)

// maxInlineFileBytes caps file content carried through MCP arguments and
// results. The sandbox API accepts up to 50 MiB, which is too large for JSON
// tool messages and for a model's context.
const maxInlineFileBytes = 1 << 20 // 1 MiB

// Download bodies end with a fixed 73-byte footer: the magic "DOWSSHA1" (8
// bytes), the lowercase hex SHA-256 of the payload (64 bytes), and a newline.
const (
	downloadFooterLen = 73
	downloadMagic     = "DOWSSHA1"
)

// errTooLarge is returned by capWriter when a download exceeds the inline cap.
var errTooLarge = errors.New("file exceeds the inline size limit")

// capWriter buffers a download body, failing once it passes max bytes.
type capWriter struct {
	buf bytes.Buffer
	max int
}

func (w *capWriter) Write(p []byte) (int, error) {
	if w.buf.Len()+len(p) > w.max {
		return 0, errTooLarge
	}
	return w.buf.Write(p)
}

// splitFooter removes the integrity footer from a download body and verifies
// the payload's SHA-256 against it.
func splitFooter(body []byte) ([]byte, error) {
	if len(body) < downloadFooterLen {
		return nil, errors.New("response is shorter than the integrity footer")
	}
	payload := body[:len(body)-downloadFooterLen]
	footer := body[len(body)-downloadFooterLen:]
	if string(footer[:8]) != downloadMagic || footer[downloadFooterLen-1] != '\n' {
		return nil, errors.New("missing integrity footer; the transfer may be truncated")
	}
	sum := sha256.Sum256(payload)
	if string(footer[8:72]) != hex.EncodeToString(sum[:]) {
		return nil, errors.New("SHA-256 mismatch; the transfer is corrupt")
	}
	return payload, nil
}

func (t *Tool) downloadFile(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	args := req.GetArguments()
	id, errRes := requireString(args, "SandboxID")
	if errRes != nil {
		return errRes, nil
	}
	path, errRes := requireString(args, "Path")
	if errRes != nil {
		return errRes, nil
	}

	q := url.Values{"path": {path}}
	if argBool(args, "AsArchive") {
		q.Set("as_archive", "true")
	}

	client, err := t.client(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to get DigitalOcean client: %w", err)
	}
	apiReq, err := client.NewRequest(ctx, http.MethodGet, withQuery(sandboxPath(id, "files", "download"), q), nil)
	if err != nil {
		return mcp.NewToolResultErrorFromErr("Failed to create request", err), nil
	}
	apiReq.Header.Set("Accept", "application/octet-stream")

	// Allow the payload plus its footer, and no more.
	w := &capWriter{max: maxInlineFileBytes + downloadFooterLen}
	if _, err := client.Do(ctx, apiReq, w); err != nil {
		if errors.Is(err, errTooLarge) {
			return mcp.NewToolResultError(fmt.Sprintf("%s is larger than the %d byte inline limit; process it in the sandbox with sandbox-exec-start instead", path, maxInlineFileBytes)), nil
		}
		return mcp.NewToolResultErrorFromErr("Failed to download file", err), nil
	}

	payload, err := splitFooter(w.buf.Bytes())
	if err != nil {
		return mcp.NewToolResultErrorFromErr("Download failed integrity check", err), nil
	}

	sum := sha256.Sum256(payload)
	res := FileDownloadResult{
		Path:      path,
		SizeBytes: len(payload),
		SHA256:    hex.EncodeToString(sum[:]),
	}
	if utf8.Valid(payload) {
		res.Encoding = "utf-8"
		res.Content = string(payload)
	} else {
		res.Encoding = "base64"
		res.Content = base64.StdEncoding.EncodeToString(payload)
	}
	return fileDownloadOut.Result(res)
}

func (t *Tool) uploadFile(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	args := req.GetArguments()
	id, errRes := requireString(args, "SandboxID")
	if errRes != nil {
		return errRes, nil
	}
	path, errRes := requireString(args, "Path")
	if errRes != nil {
		return errRes, nil
	}

	// Content is used as given: trimming would change the file.
	text, hasText := args["Content"].(string)
	encoded, hasBase64 := args["ContentBase64"].(string)
	if hasText == hasBase64 {
		return mcp.NewToolResultError("Provide exactly one of Content (text) or ContentBase64"), nil
	}

	var data []byte
	if hasText {
		data = []byte(text)
	} else {
		decoded, err := base64.StdEncoding.DecodeString(encoded)
		if err != nil {
			return mcp.NewToolResultError("ContentBase64 must be valid base64"), nil
		}
		data = decoded
	}
	if len(data) > maxInlineFileBytes {
		return mcp.NewToolResultError(fmt.Sprintf("File content is %d bytes; the inline limit is %d bytes", len(data), maxInlineFileBytes)), nil
	}

	q := url.Values{"path": {path}}
	if argBool(args, "IsArchive") {
		q.Set("is_archive", "true")
	}

	client, err := t.client(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to get DigitalOcean client: %w", err)
	}
	apiReq, err := client.NewRequest(ctx, http.MethodPost, withQuery(sandboxPath(id, "files", "upload"), q), nil)
	if err != nil {
		return mcp.NewToolResultErrorFromErr("Failed to create request", err), nil
	}
	sum := sha256.Sum256(data)
	apiReq.Header.Set("Content-Type", "application/octet-stream")
	apiReq.Header.Set("X-Content-Sha256", hex.EncodeToString(sum[:]))
	apiReq.Body = io.NopCloser(bytes.NewReader(data))
	apiReq.ContentLength = int64(len(data))

	var out fileUploadResponse
	if _, err := client.Do(ctx, apiReq, &out); err != nil {
		return mcp.NewToolResultErrorFromErr("Failed to upload file", err), nil
	}
	return fileUploadOut.Result(out)
}

func (t *Tool) listFiles(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	args := req.GetArguments()
	id, errRes := requireString(args, "SandboxID")
	if errRes != nil {
		return errRes, nil
	}
	path, errRes := requireString(args, "Path")
	if errRes != nil {
		return errRes, nil
	}

	q := url.Values{"path": {path}}
	if depth, ok := argInt(args, "Depth"); ok && depth > 0 {
		q.Set("depth", strconv.FormatInt(depth, 10))
	}

	var out fileListResponse
	if err := t.do(ctx, http.MethodGet, withQuery(sandboxPath(id, "files", "list"), q), nil, &out); err != nil {
		return mcp.NewToolResultErrorFromErr("Failed to list files", err), nil
	}
	if out.Entries == nil {
		out.Entries = []FileEntry{}
	}
	return fileListOut.Result(out)
}

func (t *Tool) fileInfo(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	args := req.GetArguments()
	id, errRes := requireString(args, "SandboxID")
	if errRes != nil {
		return errRes, nil
	}
	path, errRes := requireString(args, "Path")
	if errRes != nil {
		return errRes, nil
	}

	var out fileInfoResponse
	q := url.Values{"path": {path}}
	if err := t.do(ctx, http.MethodGet, withQuery(sandboxPath(id, "files", "info"), q), nil, &out); err != nil {
		return mcp.NewToolResultErrorFromErr("Failed to get file information", err), nil
	}
	return fileInfoOut.Result(out)
}

func (t *Tool) fileExists(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	args := req.GetArguments()
	id, errRes := requireString(args, "SandboxID")
	if errRes != nil {
		return errRes, nil
	}
	path, errRes := requireString(args, "Path")
	if errRes != nil {
		return errRes, nil
	}

	var out fileExistsResponse
	q := url.Values{"path": {path}}
	if err := t.do(ctx, http.MethodGet, withQuery(sandboxPath(id, "files", "exists"), q), nil, &out); err != nil {
		return mcp.NewToolResultErrorFromErr("Failed to check file", err), nil
	}
	return fileExistsOut.Result(out)
}

func (t *Tool) makeDirectory(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	args := req.GetArguments()
	id, errRes := requireString(args, "SandboxID")
	if errRes != nil {
		return errRes, nil
	}
	path, errRes := requireString(args, "Path")
	if errRes != nil {
		return errRes, nil
	}

	var out fileMakeDirectoryResponse
	if err := t.do(ctx, http.MethodPost, sandboxPath(id, "files", "make-directory"), fileMakeDirectoryRequest{Path: path}, &out); err != nil {
		return mcp.NewToolResultErrorFromErr("Failed to create directory", err), nil
	}
	return fileMkdirOut.Result(out)
}

func (t *Tool) moveFile(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	args := req.GetArguments()
	id, errRes := requireString(args, "SandboxID")
	if errRes != nil {
		return errRes, nil
	}
	source, errRes := requireString(args, "Source")
	if errRes != nil {
		return errRes, nil
	}
	destination, errRes := requireString(args, "Destination")
	if errRes != nil {
		return errRes, nil
	}

	var out fileMoveResponse
	body := fileMoveRequest{Source: source, Destination: destination}
	if err := t.do(ctx, http.MethodPost, sandboxPath(id, "files", "move"), body, &out); err != nil {
		return mcp.NewToolResultErrorFromErr("Failed to move file", err), nil
	}
	return fileMoveOut.Result(out)
}

func (t *Tool) removeFile(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	args := req.GetArguments()
	id, errRes := requireString(args, "SandboxID")
	if errRes != nil {
		return errRes, nil
	}
	path, errRes := requireString(args, "Path")
	if errRes != nil {
		return errRes, nil
	}

	if err := t.do(ctx, http.MethodPost, sandboxPath(id, "files", "remove"), fileRemoveRequest{Path: path}, nil); err != nil {
		return mcp.NewToolResultErrorFromErr("Failed to remove file", err), nil
	}
	return mcp.NewToolResultText(fmt.Sprintf("Removed %s from sandbox %s.", path, id)), nil
}

// fileTools returns the sandbox file tools.
func (t *Tool) fileTools() []server.ServerTool {
	return []server.ServerTool{
		{
			Handler: t.listFiles,
			Tool: mcp.NewTool("sandbox-file-list",
				common.WithHints(common.HintsRead),
				common.WithRisk(common.RiskLow),
				fileListOut.Schema(),
				mcp.WithDescription("List the files in a sandbox directory. Depth controls how many levels to descend. Requires the sandbox:read scope."),
				mcp.WithString("SandboxID", mcp.Required(), mcp.Description("The sandbox ID")),
				mcp.WithString("Path", mcp.Required(), mcp.Description("Directory to list, e.g. /workspace")),
				mcp.WithNumber("Depth", mcp.Description("How many levels to descend; omit for the API default")),
			),
		},
		{
			Handler: t.fileInfo,
			Tool: mcp.NewTool("sandbox-file-info",
				common.WithHints(common.HintsRead),
				common.WithRisk(common.RiskLow),
				fileInfoOut.Schema(),
				mcp.WithDescription("Get the name, type, size, mode, owner, and modification time of a file in a sandbox. Requires the sandbox:read scope."),
				mcp.WithString("SandboxID", mcp.Required(), mcp.Description("The sandbox ID")),
				mcp.WithString("Path", mcp.Required(), mcp.Description("File path inside the sandbox")),
			),
		},
		{
			Handler: t.fileExists,
			Tool: mcp.NewTool("sandbox-file-exists",
				common.WithHints(common.HintsRead),
				common.WithRisk(common.RiskLow),
				fileExistsOut.Schema(),
				mcp.WithDescription("Check whether a path exists in a sandbox. Requires the sandbox:read scope."),
				mcp.WithString("SandboxID", mcp.Required(), mcp.Description("The sandbox ID")),
				mcp.WithString("Path", mcp.Required(), mcp.Description("Path to check")),
			),
		},
		{
			Handler: t.downloadFile,
			Tool: mcp.NewTool("sandbox-file-download",
				common.WithHints(common.HintsRead),
				common.WithRisk(common.RiskLow),
				fileDownloadOut.Schema(),
				mcp.WithDescription("Read a file from a sandbox and return its contents. Text is returned as UTF-8; binary is returned base64-encoded (see encoding). The SHA-256 of the content is verified before it is returned. Files over 1 MiB are rejected. Set AsArchive to fetch a directory as a tar archive. Requires the sandbox:read scope."),
				mcp.WithString("SandboxID", mcp.Required(), mcp.Description("The sandbox ID")),
				mcp.WithString("Path", mcp.Required(), mcp.Description("File path inside the sandbox")),
				mcp.WithBoolean("AsArchive", mcp.Description("Return a directory as a tar archive")),
			),
		},
		{
			Handler: t.uploadFile,
			Tool: mcp.NewTool("sandbox-file-upload",
				common.WithHints(common.HintsOverwrite),
				common.WithRisk(common.RiskMedium),
				fileUploadOut.Schema(),
				mcp.WithDescription("Write a file into a sandbox, replacing any existing file at Path. Provide the content as Content (text) or ContentBase64 (binary), not both. Content is limited to 1 MiB. Set IsArchive to extract a tar archive at Path instead. A SHA-256 is sent so the sandbox verifies the write. Requires the sandbox:update scope."),
				mcp.WithString("SandboxID", mcp.Required(), mcp.Description("The sandbox ID")),
				mcp.WithString("Path", mcp.Required(), mcp.Description("Destination path inside the sandbox")),
				mcp.WithString("Content", mcp.Description("Text content to write, used exactly as given")),
				mcp.WithString("ContentBase64", mcp.Description("Base64-encoded binary content to write")),
				mcp.WithBoolean("IsArchive", mcp.Description("Treat the content as a tar archive and extract it at Path")),
			),
		},
		{
			Handler: t.makeDirectory,
			Tool: mcp.NewTool("sandbox-file-make-directory",
				common.WithHints(common.HintsAction),
				common.WithRisk(common.RiskLow),
				fileMkdirOut.Schema(),
				mcp.WithDescription("Create a directory in a sandbox. Requires the sandbox:update scope."),
				mcp.WithString("SandboxID", mcp.Required(), mcp.Description("The sandbox ID")),
				mcp.WithString("Path", mcp.Required(), mcp.Description("Directory path to create")),
			),
		},
		{
			Handler: t.moveFile,
			Tool: mcp.NewTool("sandbox-file-move",
				common.WithHints(common.HintsAction),
				common.WithRisk(common.RiskMedium),
				fileMoveOut.Schema(),
				mcp.WithDescription("Move or rename a file or directory in a sandbox. Requires the sandbox:update scope."),
				mcp.WithString("SandboxID", mcp.Required(), mcp.Description("The sandbox ID")),
				mcp.WithString("Source", mcp.Required(), mcp.Description("Current path")),
				mcp.WithString("Destination", mcp.Required(), mcp.Description("New path")),
			),
		},
		{
			Handler: t.removeFile,
			Tool: mcp.NewTool("sandbox-file-remove",
				common.WithHints(common.HintsDelete),
				common.WithRisk(common.RiskHigh),
				mcp.WithDescription("Delete a file or directory in a sandbox. Destructive. Requires the sandbox:update scope."),
				mcp.WithString("SandboxID", mcp.Required(), mcp.Description("The sandbox ID")),
				mcp.WithString("Path", mcp.Required(), mcp.Description("Path to delete")),
			),
		},
	}
}
