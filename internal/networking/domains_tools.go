package networking

import (
	"context"
	"encoding/json"

	"github.com/digitalocean/godo"
	"github.com/mark3labs/mcp-go/mcp"
	"github.com/mark3labs/mcp-go/server"
)

type DomainsTool struct {
	client *godo.Client
}

func NewDomainsTool(client *godo.Client) *DomainsTool {
	return &DomainsTool{
		client: client,
	}
}

func (d *DomainsTool) createDomain(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	name := req.GetArguments()["Name"].(string)
	ipAddress := req.GetArguments()["IPAddress"].(string)

	createRequest := &godo.DomainCreateRequest{
		Name:      name,
		IPAddress: ipAddress,
	}

	domain, _, err := d.client.Domains.Create(ctx, createRequest)
	if err != nil {
		return mcp.NewToolResultErrorFromErr("api error", err), nil
	}

	jsonDomain, err := json.MarshalIndent(domain, "", "  ")
	if err != nil {
		return mcp.NewToolResultErrorFromErr("marshal error", err), nil
	}

	return mcp.NewToolResultText(string(jsonDomain)), nil
}

func (d *DomainsTool) deleteDomain(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	name := req.GetArguments()["Name"].(string)

	_, err := d.client.Domains.Delete(ctx, name)
	if err != nil {
		return mcp.NewToolResultErrorFromErr("api error", err), nil
	}

	return mcp.NewToolResultText("Domain deleted successfully"), nil
}

func (d *DomainsTool) createRecord(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	domain := req.GetArguments()["Domain"].(string)
	recordType := req.GetArguments()["Type"].(string)
	name := req.GetArguments()["Name"].(string)
	data := req.GetArguments()["Data"].(string)

	createRequest := &godo.DomainRecordEditRequest{
		Type: recordType,
		Name: name,
		Data: data,
	}

	record, _, err := d.client.Domains.CreateRecord(ctx, domain, createRequest)
	if err != nil {
		return mcp.NewToolResultErrorFromErr("api error", err), nil
	}

	jsonRecord, err := json.MarshalIndent(record, "", "  ")
	if err != nil {
		return mcp.NewToolResultErrorFromErr("marshal error", err), nil
	}

	return mcp.NewToolResultText(string(jsonRecord)), nil
}

func (d *DomainsTool) deleteRecord(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	domain := req.GetArguments()["Domain"].(string)
	recordID := int(req.GetArguments()["RecordID"].(float64))

	_, err := d.client.Domains.DeleteRecord(ctx, domain, recordID)
	if err != nil {
		return mcp.NewToolResultErrorFromErr("api error", err), nil
	}

	return mcp.NewToolResultText("Record deleted successfully"), nil
}

func (d *DomainsTool) editRecord(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	domain := req.GetArguments()["Domain"].(string)
	recordID := int(req.GetArguments()["RecordID"].(float64))
	recordType := req.GetArguments()["Type"].(string)
	name := req.GetArguments()["Name"].(string)
	data := req.GetArguments()["Data"].(string)

	editRequest := &godo.DomainRecordEditRequest{
		Type: recordType,
		Name: name,
		Data: data,
	}

	record, _, err := d.client.Domains.EditRecord(ctx, domain, recordID, editRequest)
	if err != nil {
		return mcp.NewToolResultErrorFromErr("api error", err), nil
	}

	jsonRecord, err := json.MarshalIndent(record, "", "  ")
	if err != nil {
		return mcp.NewToolResultErrorFromErr("marshal error", err), nil
	}

	return mcp.NewToolResultText(string(jsonRecord)), nil
}

func (d *DomainsTool) Tools() []server.ServerTool {
	return []server.ServerTool{
		{
			Handler: d.createDomain,
			Tool: mcp.NewTool("digitalocean-domain-create",
				mcp.WithDescription("Create a new domain"),
				mcp.WithString("Name", mcp.Required(), mcp.Description("Name of the domain")),
				mcp.WithString("IPAddress", mcp.Required(), mcp.Description("IP address for the domain")),
			),
		},
		{
			Handler: d.deleteDomain,
			Tool: mcp.NewTool("digitalocean-domain-delete",
				mcp.WithDescription("Delete a domain"),
				mcp.WithString("Name", mcp.Required(), mcp.Description("Name of the domain to delete")),
			),
		},
		{
			Handler: d.createRecord,
			Tool: mcp.NewTool("digitalocean-domain-record-create",
				mcp.WithDescription("Create a new domain record"),
				mcp.WithString("Domain", mcp.Required(), mcp.Description("Domain name")),
				mcp.WithString("Type", mcp.Required(), mcp.Description("Record type (e.g., A, CNAME, TXT)")),
				mcp.WithString("Name", mcp.Required(), mcp.Description("Record name")),
				mcp.WithString("Data", mcp.Required(), mcp.Description("Record data")),
			),
		},
		{
			Handler: d.deleteRecord,
			Tool: mcp.NewTool("digitalocean-domain-record-delete",
				mcp.WithDescription("Delete a domain record"),
				mcp.WithString("Domain", mcp.Required(), mcp.Description("Domain name")),
				mcp.WithNumber("RecordID", mcp.Required(), mcp.Description("ID of the record to delete")),
			),
		},
		{
			Handler: d.editRecord,
			Tool: mcp.NewTool("digitalocean-domain-record-edit",
				mcp.WithDescription("Edit a domain record"),
				mcp.WithString("Domain", mcp.Required(), mcp.Description("Domain name")),
				mcp.WithNumber("RecordID", mcp.Required(), mcp.Description("ID of the record to edit")),
				mcp.WithString("Type", mcp.Required(), mcp.Description("Record type (e.g., A, CNAME, TXT)")),
				mcp.WithString("Name", mcp.Required(), mcp.Description("Record name")),
				mcp.WithString("Data", mcp.Required(), mcp.Description("Record data")),
			),
		},
	}
}
