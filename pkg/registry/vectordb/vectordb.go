package vectordb

import (
	"context"
	"encoding/json"

	"github.com/digitalocean/godo"
	"github.com/mark3labs/mcp-go/mcp"
	"github.com/mark3labs/mcp-go/server"

	"mcp-digitalocean/pkg/registry/common"
)

const (
	defaultVectorDBListPage    = 1
	defaultVectorDBListPerPage = 20
	maxVectorDBListPerPage     = 50
)

// VectorDBTool exposes DigitalOcean Managed Weaviate (vector database) control
// plane operations as MCP tools.
type VectorDBTool struct {
	client func(ctx context.Context) (*godo.Client, error)
}

// NewVectorDBTool creates a new VectorDBTool instance.
func NewVectorDBTool(client func(ctx context.Context) (*godo.Client, error)) *VectorDBTool {
	return &VectorDBTool{client: client}
}

func (v *VectorDBTool) createVectorDB(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	args := req.GetArguments()
	// required arguments
	name, ok := args["Name"].(string)
	if !ok || name == "" {
		return mcp.NewToolResultError("Name is required"), nil
	}
	region, ok := args["Region"].(string)
	if !ok || region == "" {
		return mcp.NewToolResultError("Region is required"), nil
	}
	size, ok := args["Size"].(string)
	if !ok || size == "" {
		return mcp.NewToolResultError("Size is required (one of: small, medium, large)"), nil
	}
	projectID, ok := args["ProjectId"].(string)
	if !ok || projectID == "" {
		return mcp.NewToolResultError("ProjectId is required"), nil
	}

	// optional arguments
	tagsArg, _ := args["Tags"].([]any)
	var tags []string
	for _, tag := range tagsArg {
		if s, ok := tag.(string); ok {
			tags = append(tags, s)
		}
	}

	createRequest := &godo.VectorDBCreateRequest{
		Name:      name,
		Region:    region,
		Size:      size,
		ProjectID: projectID,
		Tags:      tags,
	}

	client, err := v.client(ctx)
	if err != nil {
		return mcp.NewToolResultErrorFromErr("Error getting DigitalOcean client", err), nil
	}

	vectorDB, _, err := client.VectorDBs.Create(ctx, createRequest)
	if err != nil {
		return mcp.NewToolResultErrorFromErr("api error", err), nil
	}

	jsonVectorDB, err := json.MarshalIndent(vectorDB, "", "  ")
	if err != nil {
		return mcp.NewToolResultErrorFromErr("marshal error", err), nil
	}
	return mcp.NewToolResultText(string(jsonVectorDB)), nil
}

func (v *VectorDBTool) listVectorDBs(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	args := req.GetArguments()

	page, ok := args["Page"].(float64)
	if !ok || page < 1 {
		page = defaultVectorDBListPage
	}
	perPage, ok := args["PerPage"].(float64)
	if !ok || perPage < 1 {
		perPage = defaultVectorDBListPerPage
	}
	if perPage > maxVectorDBListPerPage {
		perPage = maxVectorDBListPerPage
	}

	listOptions := &godo.ListOptions{
		Page:    int(page),
		PerPage: int(perPage),
	}

	client, err := v.client(ctx)
	if err != nil {
		return mcp.NewToolResultErrorFromErr("Error getting DigitalOcean client", err), nil
	}

	vectorDBs, _, err := client.VectorDBs.List(ctx, listOptions)
	if err != nil {
		return mcp.NewToolResultErrorFromErr("api error", err), nil
	}

	formattedVectorDBs := make([]map[string]any, len(vectorDBs))
	for i, vectorDB := range vectorDBs {
		var httpEndpoint, grpcEndpoint string
		if vectorDB.Endpoints != nil {
			httpEndpoint = vectorDB.Endpoints.HTTP
			grpcEndpoint = vectorDB.Endpoints.GRPC
		}
		formattedVectorDBs[i] = map[string]any{
			"id":            vectorDB.ID,
			"name":          vectorDB.Name,
			"region":        vectorDB.Region,
			"size":          vectorDB.Size,
			"status":        vectorDB.Status,
			"created_at":    vectorDB.CreatedAt,
			"tags":          vectorDB.Tags,
			"endpoint_http": httpEndpoint,
			"endpoint_grpc": grpcEndpoint,
		}
	}

	jsonVectorDBs, err := json.MarshalIndent(formattedVectorDBs, "", "  ")
	if err != nil {
		return mcp.NewToolResultErrorFromErr("marshal error", err), nil
	}
	return mcp.NewToolResultText(string(jsonVectorDBs)), nil
}

func (v *VectorDBTool) getVectorDBByID(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	args := req.GetArguments()
	id, ok := args["ID"].(string)
	if !ok || id == "" {
		return mcp.NewToolResultError("ID is required"), nil
	}

	client, err := v.client(ctx)
	if err != nil {
		return mcp.NewToolResultErrorFromErr("Error getting DigitalOcean client", err), nil
	}

	vectorDB, _, err := client.VectorDBs.Get(ctx, id)
	if err != nil {
		return mcp.NewToolResultErrorFromErr("api error", err), nil
	}

	jsonVectorDB, err := json.MarshalIndent(vectorDB, "", "  ")
	if err != nil {
		return mcp.NewToolResultErrorFromErr("marshal error", err), nil
	}
	return mcp.NewToolResultText(string(jsonVectorDB)), nil
}

func (v *VectorDBTool) resizeVectorDB(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	args := req.GetArguments()
	id, ok := args["ID"].(string)
	if !ok || id == "" {
		return mcp.NewToolResultError("ID is required"), nil
	}
	size, ok := args["Size"].(string)
	if !ok || size == "" {
		return mcp.NewToolResultError("Size is required (one of: small, medium, large)"), nil
	}

	resizeRequest := &godo.VectorDBResizeRequest{
		Size: size,
	}

	client, err := v.client(ctx)
	if err != nil {
		return mcp.NewToolResultErrorFromErr("Error getting DigitalOcean client", err), nil
	}

	vectorDB, _, err := client.VectorDBs.Resize(ctx, id, resizeRequest)
	if err != nil {
		return mcp.NewToolResultErrorFromErr("api error", err), nil
	}

	jsonVectorDB, err := json.MarshalIndent(vectorDB, "", "  ")
	if err != nil {
		return mcp.NewToolResultErrorFromErr("marshal error", err), nil
	}
	return mcp.NewToolResultText(string(jsonVectorDB)), nil
}

func (v *VectorDBTool) deleteVectorDB(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	args := req.GetArguments()
	id, ok := args["ID"].(string)
	if !ok || id == "" {
		return mcp.NewToolResultError("ID is required"), nil
	}

	client, err := v.client(ctx)
	if err != nil {
		return mcp.NewToolResultErrorFromErr("Error getting DigitalOcean client", err), nil
	}

	_, err = client.VectorDBs.Delete(ctx, id)
	if err != nil {
		return mcp.NewToolResultErrorFromErr("api error", err), nil
	}
	return mcp.NewToolResultText("Vector database deleted successfully"), nil
}

func (v *VectorDBTool) getVectorDBCredentials(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	args := req.GetArguments()
	id, ok := args["ID"].(string)
	if !ok || id == "" {
		return mcp.NewToolResultError("ID is required"), nil
	}

	client, err := v.client(ctx)
	if err != nil {
		return mcp.NewToolResultErrorFromErr("Error getting DigitalOcean client", err), nil
	}

	credentials, _, err := client.VectorDBs.GetCredentials(ctx, id)
	if err != nil {
		return mcp.NewToolResultErrorFromErr("api error", err), nil
	}

	jsonCredentials, err := json.MarshalIndent(credentials, "", "  ")
	if err != nil {
		return mcp.NewToolResultErrorFromErr("marshal error", err), nil
	}
	return mcp.NewToolResultText(string(jsonCredentials)), nil
}

func (v *VectorDBTool) Tools() []server.ServerTool {
	return []server.ServerTool{
		{
			Handler: v.createVectorDB,
			Tool: mcp.NewTool(
				"vector-db-create",
				common.WithHints(common.HintsCreate),
				common.WithRisk(common.RiskMedium),
				mcp.WithDescription("Create a new managed Weaviate vector database cluster. Returns the cluster including its ID, status, and connection endpoints (endpoints.http, endpoints.grpc)."),
				mcp.WithString("Name", mcp.Required(), mcp.Description("Name of the vector database cluster")),
				mcp.WithString("Region", mcp.Required(), mcp.Description("Region slug where the cluster will be created. Currently only 'tor1' is available.")),
				mcp.WithString("Size", mcp.Required(), mcp.Description("Resource tier of the cluster. One of: small, medium, large")),
				mcp.WithString("ProjectId", mcp.Required(), mcp.Description("ID of the project to assign the cluster to. Write-only: it is not returned in the response.")),
				mcp.WithArray("Tags", mcp.Description("Optional tags to apply to the cluster")),
			),
		},
		{
			Handler: v.listVectorDBs,
			Tool: mcp.NewTool(
				"vector-db-list",
				common.WithHints(common.HintsRead),
				common.WithRisk(common.RiskLow),
				mcp.WithDescription("List vector database clusters. Supports pagination. Returns a summary of each cluster including its connection endpoints."),
				mcp.WithNumber("Page", mcp.DefaultNumber(1), mcp.Description("Page number of the results to fetch")),
				mcp.WithNumber("PerPage", mcp.DefaultNumber(20), mcp.Description("Number of items returned per page")),
			),
		},
		{
			Handler: v.getVectorDBByID,
			Tool: mcp.NewTool(
				"vector-db-get",
				common.WithHints(common.HintsRead),
				common.WithRisk(common.RiskLow),
				mcp.WithDescription("Get a vector database cluster by ID. Returns the full cluster including status, config, and connection endpoints."),
				mcp.WithString("ID", mcp.Required(), mcp.Description("ID of the vector database cluster to get")),
			),
		},
		{
			Handler: v.resizeVectorDB,
			Tool: mcp.NewTool(
				"vector-db-resize",
				common.WithHints(common.HintsToggle),
				common.WithRisk(common.RiskMedium),
				mcp.WithDescription("Resize a vector database cluster to a new resource tier."),
				mcp.WithString("ID", mcp.Required(), mcp.Description("ID of the vector database cluster to resize")),
				mcp.WithString("Size", mcp.Required(), mcp.Description("New resource tier. One of: small, medium, large")),
			),
		},
		{
			Handler: v.deleteVectorDB,
			Tool: mcp.NewTool(
				"vector-db-delete",
				common.WithHints(common.HintsDelete),
				common.WithRisk(common.RiskHigh),
				mcp.WithDescription("Delete a vector database cluster by ID. This is irreversible."),
				mcp.WithString("ID", mcp.Required(), mcp.Description("ID of the vector database cluster to delete")),
			),
		},
		{
			Handler: v.getVectorDBCredentials,
			Tool: mcp.NewTool(
				"vector-db-get-credentials",
				common.WithHints(common.HintsRead),
				common.WithRisk(common.RiskMedium),
				mcp.WithDescription("Get the admin credentials (user_id and api_token) for a vector database cluster. Use these together with the cluster's endpoints to connect a Weaviate client."),
				mcp.WithString("ID", mcp.Required(), mcp.Description("ID of the vector database cluster to get credentials for")),
			),
		},
	}
}
