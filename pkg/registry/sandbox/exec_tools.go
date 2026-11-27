package sandbox

import (
	"bufio"
	"bytes"
	"context"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"strconv"
	"strings"

	"github.com/mark3labs/mcp-go/mcp"
	"github.com/mark3labs/mcp-go/server"

	"mcp-digitalocean/pkg/registry/common"
)

// maxExecOutputBytes caps stdout and stderr returned to the caller, so a noisy
// command cannot flood the model's context. Output past the cap is dropped and
// the matching *_truncated flag is set.
const maxExecOutputBytes = 1 << 20 // 1 MiB

// maxExecEventBytes bounds one stream line. Each line carries a base64 chunk
// of output, so it can be large.
const maxExecEventBytes = 4 << 20 // 4 MiB

// execEnvelope is one line of the exec stream: an event or a transport error.
type execEnvelope struct {
	Result *execEvent `json:"result,omitempty"`
	Error  *struct {
		Message string `json:"message"`
	} `json:"error,omitempty"`
}

// execEvent holds the fields of a sandbox exec event; exactly one is set.
type execEvent struct {
	Started *struct {
		ExecID string `json:"exec_id"`
	} `json:"started,omitempty"`
	Stdout string `json:"stdout,omitempty"` // base64
	Stderr string `json:"stderr,omitempty"` // base64
	Exit   *struct {
		ExitCode   int32  `json:"exit_code"`
		DurationMS string `json:"duration_ms"`
		Status     string `json:"status"`
		Error      string `json:"error"`
	} `json:"exit,omitempty"`
}

// outputBuffer collects process output up to maxExecOutputBytes.
type outputBuffer struct {
	buf       bytes.Buffer
	truncated bool
}

func (b *outputBuffer) write(p []byte) {
	room := maxExecOutputBytes - b.buf.Len()
	if room <= 0 {
		b.truncated = true
		return
	}
	if len(p) > room {
		p = p[:room]
		b.truncated = true
	}
	b.buf.Write(p)
}

// text returns the collected output as valid UTF-8. Invalid bytes become U+FFFD.
func (b *outputBuffer) text() string {
	return strings.ToValidUTF8(b.buf.String(), "\uFFFD")
}

// collectExec reads an exec stream until the exit event. The endpoint is
// documented as server-sent events but its schema describes one JSON envelope
// per line, so both framings are accepted.
func collectExec(r io.Reader) (ExecResult, error) {
	var (
		res            ExecResult
		stdout, stderr outputBuffer
		exited         bool
	)

	scanner := bufio.NewScanner(r)
	scanner.Buffer(make([]byte, 64*1024), maxExecEventBytes)
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		line = strings.TrimSpace(strings.TrimPrefix(line, "data:"))
		if line == "" || strings.HasPrefix(line, ":") || strings.HasPrefix(line, "event:") || strings.HasPrefix(line, "id:") {
			continue
		}

		var env execEnvelope
		if err := json.Unmarshal([]byte(line), &env); err != nil {
			return res, fmt.Errorf("unexpected exec stream line: %w", err)
		}
		if env.Error != nil {
			return res, fmt.Errorf("exec stream error: %s", env.Error.Message)
		}
		ev := env.Result
		if ev == nil {
			continue
		}
		if ev.Started != nil {
			res.ExecID = ev.Started.ExecID
		}
		if ev.Stdout != "" {
			chunk, err := base64.StdEncoding.DecodeString(ev.Stdout)
			if err != nil {
				return res, fmt.Errorf("invalid stdout chunk: %w", err)
			}
			stdout.write(chunk)
		}
		if ev.Stderr != "" {
			chunk, err := base64.StdEncoding.DecodeString(ev.Stderr)
			if err != nil {
				return res, fmt.Errorf("invalid stderr chunk: %w", err)
			}
			stderr.write(chunk)
		}
		if ev.Exit != nil {
			exited = true
			res.ExitCode = ev.Exit.ExitCode
			res.Status = ev.Exit.Status
			res.Error = ev.Exit.Error
			// duration_ms is a protobuf int64, so it arrives as a string.
			if ms, err := strconv.ParseInt(ev.Exit.DurationMS, 10, 64); err == nil {
				res.DurationMS = ms
			}
		}
	}
	if err := scanner.Err(); err != nil {
		return res, err
	}
	if !exited {
		return res, errors.New("exec stream ended before the process exited")
	}

	res.Stdout = stdout.text()
	res.StdoutTruncated = stdout.truncated
	res.Stderr = stderr.text()
	res.StderrTruncated = stderr.truncated
	return res, nil
}

// runExec posts to the exec endpoint and collects the stream into one result.
func (t *Tool) runExec(ctx context.Context, path string, body any) (ExecResult, error) {
	client, err := t.client(ctx)
	if err != nil {
		return ExecResult{}, fmt.Errorf("failed to get DigitalOcean client: %w", err)
	}
	req, err := client.NewRequest(ctx, http.MethodPost, path, body)
	if err != nil {
		return ExecResult{}, err
	}
	req.Header.Set("Accept", "text/event-stream")

	resp, err := client.DoStream(ctx, req)
	if err != nil {
		return ExecResult{}, err
	}
	defer resp.Body.Close()
	return collectExec(resp.Body)
}

func (t *Tool) startExec(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	args := req.GetArguments()
	id, errRes := requireString(args, "SandboxID")
	if errRes != nil {
		return errRes, nil
	}
	argv := argStringSlice(args, "Argv")
	if len(argv) == 0 {
		return mcp.NewToolResultError("Argv is required and must be a non-empty array of strings"), nil
	}

	body := execRequest{
		Argv:    argv,
		Workdir: argString(args, "Workdir"),
		EnvVars: argStringMap(args, "EnvVars"),
	}
	if n, ok := argInt(args, "TimeoutSeconds"); ok && n > 0 {
		body.TimeoutSeconds = n
	}
	if stdin := argString(args, "Stdin"); stdin != "" {
		if _, err := base64.StdEncoding.DecodeString(stdin); err != nil {
			return mcp.NewToolResultError("Stdin must be base64-encoded"), nil
		}
		body.Stdin = stdin
	}

	res, err := t.runExec(ctx, sandboxPath(id, "exec"), body)
	if err != nil {
		return mcp.NewToolResultErrorFromErr("Failed to run command in sandbox", err), nil
	}
	return execOut.Result(res)
}

// execTools returns the one-shot exec tool.
func (t *Tool) execTools() []server.ServerTool {
	return []server.ServerTool{
		{
			Handler: t.startExec,
			Tool: mcp.NewTool("sandbox-exec-start",
				common.WithHints(common.HintsCreate),
				common.WithRisk(common.RiskHigh),
				execOut.Schema(),
				mcp.WithDescription("Run a command in a sandbox to completion and return its exit code, stdout, and stderr. The program runs directly with no shell, so use argv ['sh', '-c', '...'] for pipes or redirection. Output is capped at 1 MiB per stream. Use sandbox-command-start for long-running processes. Requires the sandbox:update scope."),
				mcp.WithString("SandboxID", mcp.Required(), mcp.Description("The sandbox ID")),
				mcp.WithArray("Argv", mcp.Required(), mcp.Description("Program and arguments, e.g. [\"ls\", \"-la\", \"/tmp\"]"), mcp.Items(map[string]any{"type": "string"})),
				mcp.WithString("Workdir", mcp.Description("Working directory inside the sandbox")),
				mcp.WithObject("EnvVars", mcp.Description("Environment variables, as a string-to-string object")),
				mcp.WithNumber("TimeoutSeconds", mcp.Description("Kill the command after this many seconds")),
				mcp.WithString("Stdin", mcp.Description("Base64-encoded bytes delivered to stdin, then stdin is closed")),
			),
		},
	}
}
