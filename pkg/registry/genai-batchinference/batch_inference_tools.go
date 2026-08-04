package genaibatchinference

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"

	"github.com/digitalocean/godo"
	"github.com/mark3labs/mcp-go/mcp"
	"github.com/mark3labs/mcp-go/server"

	"mcp-digitalocean/pkg/registry/common"
)

// BatchInferenceTool provides GenAI Batch Inference lifecycle management tools.
type BatchInferenceTool struct {
	client func(ctx context.Context) (*godo.Client, error)
}

// NewBatchInferenceTool creates a new BatchInferenceTool instance.
func NewBatchInferenceTool(client func(ctx context.Context) (*godo.Client, error)) *BatchInferenceTool {
	return &BatchInferenceTool{client: client}
}

func (b *BatchInferenceTool) createFile(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	client, err := b.client(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to get DigitalOcean client: %w", err)
	}

	fileName, _ := req.GetArguments()["FileName"].(string)
	if fileName == "" {
		return mcp.NewToolResultError("FileName is required"), nil
	}

	upload, _, err := client.BatchInference.CreatePresignedUploadURL(ctx, &godo.CreateBatchFileRequest{
		FileName: fileName,
	})
	if err != nil {
		return mcp.NewToolResultErrorFromErr("Failed to create file upload presigned URL", err), nil
	}

	return marshalResult(upload)
}

func (b *BatchInferenceTool) uploadInputFile(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	client, err := b.client(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to get DigitalOcean client: %w", err)
	}

	args := req.GetArguments()

	uploadURL, _ := args["UploadURL"].(string)
	if uploadURL == "" {
		return mcp.NewToolResultError("UploadURL is required"), nil
	}

	content, _ := args["Content"].(string)
	if content == "" {
		return mcp.NewToolResultError("Content is required"), nil
	}

	_, err = client.BatchInference.UploadInputFile(ctx, uploadURL, strings.NewReader(content))
	if err != nil {
		return mcp.NewToolResultErrorFromErr("Failed to upload input file", err), nil
	}

	return mcp.NewToolResultText("File uploaded successfully"), nil
}

func (b *BatchInferenceTool) createJob(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	client, err := b.client(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to get DigitalOcean client: %w", err)
	}

	args := req.GetArguments()

	provider, _ := args["Provider"].(string)
	if provider == "" {
		return mcp.NewToolResultError("Provider is required"), nil
	}

	fileID, _ := args["FileID"].(string)
	if fileID == "" {
		return mcp.NewToolResultError("FileID is required"), nil
	}

	completionWindow, _ := args["CompletionWindow"].(string)
	if completionWindow == "" {
		return mcp.NewToolResultError("CompletionWindow is required"), nil
	}

	createReq := &godo.CreateBatchRequest{
		Provider:         provider,
		FileID:           fileID,
		CompletionWindow: completionWindow,
	}

	if v, _ := args["RequestID"].(string); v != "" {
		createReq.RequestID = v
	}
	if v, _ := args["Endpoint"].(string); v != "" {
		createReq.Endpoint = v
	}

	batch, _, err := client.BatchInference.CreateJob(ctx, createReq)
	if err != nil {
		return mcp.NewToolResultErrorFromErr("Failed to create batch inference job", err), nil
	}

	return marshalResult(batch)
}

func (b *BatchInferenceTool) getJob(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	client, err := b.client(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to get DigitalOcean client: %w", err)
	}

	batchID, _ := req.GetArguments()["BatchID"].(string)
	if batchID == "" {
		return mcp.NewToolResultError("BatchID is required"), nil
	}

	batch, _, err := client.BatchInference.GetJob(ctx, batchID)
	if err != nil {
		return mcp.NewToolResultErrorFromErr("Failed to get batch inference job", err), nil
	}

	return marshalResult(batch)
}

func (b *BatchInferenceTool) getJobResults(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	client, err := b.client(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to get DigitalOcean client: %w", err)
	}

	batchID, _ := req.GetArguments()["BatchID"].(string)
	if batchID == "" {
		return mcp.NewToolResultError("BatchID is required"), nil
	}

	results, _, err := client.BatchInference.GetJobResult(ctx, batchID)
	if err != nil {
		return mcp.NewToolResultErrorFromErr("Failed to get batch inference job results", err), nil
	}

	return marshalResult(results)
}

func (b *BatchInferenceTool) cancelJob(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	client, err := b.client(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to get DigitalOcean client: %w", err)
	}

	batchID, _ := req.GetArguments()["BatchID"].(string)
	if batchID == "" {
		return mcp.NewToolResultError("BatchID is required"), nil
	}

	batch, _, err := client.BatchInference.CancelJob(ctx, batchID)
	if err != nil {
		return mcp.NewToolResultErrorFromErr("Failed to cancel batch inference job", err), nil
	}

	return marshalResult(batch)
}

func (b *BatchInferenceTool) listJobs(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	client, err := b.client(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to get DigitalOcean client: %w", err)
	}

	args := req.GetArguments()

	opts := &godo.ListBatchesOptions{}
	if v, _ := args["Status"].(string); v != "" {
		opts.Status = v
	}
	if v, ok := args["Limit"].(float64); ok {
		opts.Limit = int(v)
	}
	if v, _ := args["After"].(string); v != "" {
		opts.After = v
	}

	list, _, err := client.BatchInference.ListJobs(ctx, opts)
	if err != nil {
		return mcp.NewToolResultErrorFromErr("Failed to list batch inference jobs", err), nil
	}

	return marshalResult(list)
}

// Tools returns the list of server tools for Batch Inference management.
func (b *BatchInferenceTool) Tools() []server.ServerTool {
	return []server.ServerTool{
		{
			Handler: b.createFile,
			Tool: mcp.NewTool(
				"genai-batch-inference-create-file",
				common.WithHints(common.HintsCreate),
				common.WithRisk(common.RiskLow),
				mcp.WithDescription("Create a presigned URL for uploading a batch inference JSONL input file. The file must have a .jsonl extension. Upload the file to the returned URL via HTTP PUT before creating a batch job."),
				mcp.WithString("FileName", mcp.Required(), mcp.Description("Name of the JSONL file to upload (must end in .jsonl)")),
			),
		},
		{
			Handler: b.uploadInputFile,
			Tool: mcp.NewTool(
				"genai-batch-inference-upload-file",
				common.WithHints(common.HintsCreate),
				common.WithRisk(common.RiskLow),
				mcp.WithDescription("Upload JSONL content to the presigned S3 URL returned by create-file. The content should be newline-delimited JSON (one request per line). Must be called after create-file and before create."),
				mcp.WithString("UploadURL", mcp.Required(), mcp.Description("Presigned upload URL from create-file response")),
				mcp.WithString("Content", mcp.Required(), mcp.Description("JSONL content to upload (newline-delimited JSON)")),
			),
		},
		{
			Handler: b.createJob,
			Tool: mcp.NewTool(
				"genai-batch-inference-create",
				common.WithHints(common.HintsCreate),
				common.WithRisk(common.RiskMedium),
				mcp.WithDescription("Create a new batch inference job. Requires a previously uploaded file (via create-file). For OpenAI provider, the Endpoint argument is also required."),
				mcp.WithString("Provider", mcp.Required(), mcp.Description("Batch provider: 'openai' or 'anthropic'")),
				mcp.WithString("FileID", mcp.Required(), mcp.Description("UUID of a previously uploaded .jsonl file")),
				mcp.WithString("CompletionWindow", mcp.Required(), mcp.Description("Completion window (e.g. '24h')")),
				mcp.WithString("RequestID", mcp.Description("Client-supplied idempotency key")),
				mcp.WithString("Endpoint", mcp.Description("OpenAI batch API target path (e.g. '/v1/chat/completions'). Required when provider is 'openai'.")),
			),
		},
		{
			Handler: b.getJob,
			Tool: mcp.NewTool(
				"genai-batch-inference-get",
				common.WithHints(common.HintsRead),
				common.WithRisk(common.RiskLow),
				mcp.WithDescription("Get the current status and metadata of a batch inference job by its ID."),
				mcp.WithString("BatchID", mcp.Required(), mcp.Description("UUID of the batch inference job")),
			),
		},
		{
			Handler: b.getJobResults,
			Tool: mcp.NewTool(
				"genai-batch-inference-get-results",
				common.WithHints(common.HintsRead),
				common.WithRisk(common.RiskLow),
				mcp.WithDescription("Get the results download URL for a completed batch inference job. Returns a presigned download URL and output file ID. Fails if the job has not completed."),
				mcp.WithString("BatchID", mcp.Required(), mcp.Description("UUID of the batch inference job")),
			),
		},
		{
			Handler: b.cancelJob,
			Tool: mcp.NewTool(
				"genai-batch-inference-cancel",
				common.WithHints(common.HintsToggle),
				common.WithRisk(common.RiskMedium),
				mcp.WithDescription("Request cancellation of a batch inference job. The job may not be cancelled immediately; poll with get to check status."),
				mcp.WithString("BatchID", mcp.Required(), mcp.Description("UUID of the batch inference job to cancel")),
			),
		},
		{
			Handler: b.listJobs,
			Tool: mcp.NewTool(
				"genai-batch-inference-list",
				common.WithHints(common.HintsRead),
				common.WithRisk(common.RiskLow),
				mcp.WithDescription("List batch inference jobs with optional status filter and cursor-based pagination. Returns Relay-style edges with per-row cursors and page_info."),
				mcp.WithString("Status", mcp.Description("Filter by job status (e.g. 'completed', 'in_progress', 'failed')")),
				mcp.WithNumber("Limit", mcp.Description("Maximum number of jobs to return per page")),
				mcp.WithString("After", mcp.Description("Cursor for pagination; pass endCursor from a previous response")),
			),
		},
	}
}

func marshalResult(v any) (*mcp.CallToolResult, error) {
	jsonData, err := json.MarshalIndent(v, "", "  ")
	if err != nil {
		return nil, fmt.Errorf("marshal error: %w", err)
	}
	return mcp.NewToolResultText(string(jsonData)), nil
}
