package sandbox

import (
	"context"
	"encoding/base64"
	"fmt"
	"net/http"

	"github.com/mark3labs/mcp-go/mcp"
	"github.com/mark3labs/mcp-go/server"

	"mcp-digitalocean/pkg/registry/common"
)

// signalValues maps the short signal names accepted by sandbox-command-signal
// to the enum values the API expects.
var signalValues = map[string]string{
	"SIGTERM": "SANDBOX_COMMAND_SIGNAL_SIGTERM",
	"SIGKILL": "SANDBOX_COMMAND_SIGNAL_SIGKILL",
}

// requirePID reads the PID argument, which identifies a command started with
// sandbox-command-start.
func requirePID(args map[string]any) (int64, *mcp.CallToolResult) {
	pid, ok := argInt(args, "PID")
	if !ok || pid <= 0 {
		return 0, mcp.NewToolResultError("PID is required and must be a positive integer")
	}
	return pid, nil
}

func (t *Tool) listCommands(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	id, errRes := requireString(req.GetArguments(), "SandboxID")
	if errRes != nil {
		return errRes, nil
	}

	var out commandListResponse
	if err := t.do(ctx, http.MethodGet, sandboxPath(id, "commands"), nil, &out); err != nil {
		return mcp.NewToolResultErrorFromErr("Failed to list sandbox commands", err), nil
	}
	if out.Commands == nil {
		out.Commands = []Command{}
	}
	return commandListOut.Result(out)
}

func (t *Tool) startCommand(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	args := req.GetArguments()
	id, errRes := requireString(args, "SandboxID")
	if errRes != nil {
		return errRes, nil
	}
	cmd, errRes := requireString(args, "Cmd")
	if errRes != nil {
		return errRes, nil
	}

	body := commandStartRequest{
		Cmd:     cmd,
		Args:    argStringSlice(args, "Args"),
		Cwd:     argString(args, "Cwd"),
		EnvVars: argStringMap(args, "EnvVars"),
		Stdin:   argBool(args, "Stdin"),
	}
	if n, ok := argInt(args, "TimeoutSeconds"); ok && n > 0 {
		body.TimeoutSeconds = n
	}

	var out commandStartResponse
	if err := t.do(ctx, http.MethodPost, sandboxPath(id, "commands"), body, &out); err != nil {
		return mcp.NewToolResultErrorFromErr("Failed to start sandbox command", err), nil
	}
	return commandStartOut.Result(out)
}

func (t *Tool) signalCommand(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	args := req.GetArguments()
	id, errRes := requireString(args, "SandboxID")
	if errRes != nil {
		return errRes, nil
	}
	pid, errRes := requirePID(args)
	if errRes != nil {
		return errRes, nil
	}
	signal, ok := signalValues[argString(args, "Signal")]
	if !ok {
		return mcp.NewToolResultError("Signal must be SIGTERM or SIGKILL"), nil
	}

	body := commandSignalRequest{Signal: signal}
	if err := t.do(ctx, http.MethodPost, sandboxPath(id, "commands", fmt.Sprint(pid), "signal"), body, nil); err != nil {
		return mcp.NewToolResultErrorFromErr("Failed to signal sandbox command", err), nil
	}
	return mcp.NewToolResultText(fmt.Sprintf("Sent %s to command %d.", argString(args, "Signal"), pid)), nil
}

func (t *Tool) writeCommandStdin(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	args := req.GetArguments()
	id, errRes := requireString(args, "SandboxID")
	if errRes != nil {
		return errRes, nil
	}
	pid, errRes := requirePID(args)
	if errRes != nil {
		return errRes, nil
	}
	data := argString(args, "Data")
	if data != "" {
		if _, err := base64.StdEncoding.DecodeString(data); err != nil {
			return mcp.NewToolResultError("Data must be base64-encoded"), nil
		}
	}

	body := commandStdinRequest{Data: data, EOF: argBool(args, "EOF")}
	if err := t.do(ctx, http.MethodPost, sandboxPath(id, "commands", fmt.Sprint(pid), "stdin"), body, nil); err != nil {
		return mcp.NewToolResultErrorFromErr("Failed to write command stdin", err), nil
	}
	return mcp.NewToolResultText(fmt.Sprintf("Wrote stdin to command %d.", pid)), nil
}

// commandTools returns the long-running command tools.
func (t *Tool) commandTools() []server.ServerTool {
	return []server.ServerTool{
		{
			Handler: t.listCommands,
			Tool: mcp.NewTool("sandbox-command-list",
				common.WithHints(common.HintsRead),
				common.WithRisk(common.RiskLow),
				commandListOut.Schema(),
				mcp.WithDescription("List the commands running in a sandbox, with their PIDs. Requires the sandbox:read scope."),
				mcp.WithString("SandboxID", mcp.Required(), mcp.Description("The sandbox ID")),
			),
		},
		{
			Handler: t.startCommand,
			Tool: mcp.NewTool("sandbox-command-start",
				common.WithHints(common.HintsCreate),
				common.WithRisk(common.RiskHigh),
				commandStartOut.Schema(),
				mcp.WithDescription("Start a long-running command in a sandbox and return its PID. The process keeps running after this call returns. Use sandbox-command-list to check it, sandbox-command-signal to stop it, and sandbox-command-stdin to write to it when Stdin is true. For a command you want to run to completion and read the output of, use sandbox-exec-start instead. Requires the sandbox:update scope."),
				mcp.WithString("SandboxID", mcp.Required(), mcp.Description("The sandbox ID")),
				mcp.WithString("Cmd", mcp.Required(), mcp.Description("Program to run (executed directly, no shell)")),
				mcp.WithArray("Args", mcp.Description("Arguments for the program"), mcp.Items(map[string]any{"type": "string"})),
				mcp.WithString("Cwd", mcp.Description("Working directory inside the sandbox")),
				mcp.WithObject("EnvVars", mcp.Description("Environment variables, as a string-to-string object")),
				mcp.WithBoolean("Stdin", mcp.Description("Keep stdin open so sandbox-command-stdin can write to the process")),
				mcp.WithNumber("TimeoutSeconds", mcp.Description("Kill the process after this many seconds")),
			),
		},
		{
			Handler: t.signalCommand,
			Tool: mcp.NewTool("sandbox-command-signal",
				common.WithHints(common.HintsAction),
				common.WithRisk(common.RiskMedium),
				mcp.WithDescription("Send SIGTERM or SIGKILL to a command started in a sandbox. Requires the sandbox:update scope."),
				mcp.WithString("SandboxID", mcp.Required(), mcp.Description("The sandbox ID")),
				mcp.WithNumber("PID", mcp.Required(), mcp.Description("PID of the command")),
				mcp.WithString("Signal", mcp.Required(), mcp.Description("SIGTERM or SIGKILL")),
			),
		},
		{
			Handler: t.writeCommandStdin,
			Tool: mcp.NewTool("sandbox-command-stdin",
				common.WithHints(common.HintsAction),
				common.WithRisk(common.RiskMedium),
				mcp.WithDescription("Write bytes to the stdin of a command started with Stdin=true. Data is base64-encoded. Set EOF to close stdin after writing. Requires the sandbox:update scope."),
				mcp.WithString("SandboxID", mcp.Required(), mcp.Description("The sandbox ID")),
				mcp.WithNumber("PID", mcp.Required(), mcp.Description("PID of the command")),
				mcp.WithString("Data", mcp.Description("Base64-encoded bytes to write")),
				mcp.WithBoolean("EOF", mcp.Description("Close stdin after writing")),
			),
		},
	}
}
