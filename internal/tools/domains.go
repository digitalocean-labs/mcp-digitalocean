package tools

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

func (d *DomainsTool) CreateDomain(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	name, err := req.RequireString("Name")
	if err != nil {
		return nil, err
	}
	
	ipAddress, err := req.RequireString("IPAddress")
	if err != nil {
		return nil, err
	}

	createRequest := &godo.DomainCreateRequest{
		Name:      name,
		IPAddress: ipAddress,
	}

	domain, _, err := d.client.Domains.Create(ctx, createRequest)
	if err != nil {
		return nil, err
	}

	jsonDomain, err := json.MarshalIndent(domain, "", "  ")
	if err != nil {
		return nil, err
	}

	return mcp.NewToolResultText(string(jsonDomain)), nil
}

func (d *DomainsTool) DeleteDomain(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	name, err := req.RequireString("Name")
	if err != nil {
		return nil, err
	}

	_, err = d.client.Domains.Delete(ctx, name)
	if err != nil {
		return nil, err
	}

	return mcp.NewToolResultText("Domain deleted successfully"), nil
}

func (d *DomainsTool) CreateRecord(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	domain, err := req.RequireString("Domain")
	if err != nil {
		return nil, err
	}
	
	recordType, err := req.RequireString("Type")
	if err != nil {
		return nil, err
	}
	
	name, err := req.RequireString("Name")
	if err != nil {
		return nil, err
	}
	
	data, err := req.RequireString("Data")
	if err != nil {
		return nil, err
	}

	createRequest := &godo.DomainRecordEditRequest{
		Type: recordType,
		Name: name,
		Data: data,
	}

	record, _, err := d.client.Domains.CreateRecord(ctx, domain, createRequest)
	if err != nil {
		return nil, err
	}

	jsonRecord, err := json.MarshalIndent(record, "", "  ")
	if err != nil {
		return nil, err
	}

	return mcp.NewToolResultText(string(jsonRecord)), nil
}

func (d *DomainsTool) DeleteRecord(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	domain, err := req.RequireString("Domain")
	if err != nil {
		return nil, err
	}
	
	recordIDFloat, err := req.RequireFloat("RecordID")
	if err != nil {
		return nil, err
	}
	recordID := int(recordIDFloat)

	_, err = d.client.Domains.DeleteRecord(ctx, domain, recordID)
	if err != nil {
		return nil, err
	}

	return mcp.NewToolResultText("Record deleted successfully"), nil
}

func (d *DomainsTool) EditRecord(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	domain, err := req.RequireString("Domain")
	if err != nil {
		return nil, err
	}
	
	recordIDFloat, err := req.RequireFloat("RecordID")
	if err != nil {
		return nil, err
	}
	recordID := int(recordIDFloat)
	
	recordType, err := req.RequireString("Type")
	if err != nil {
		return nil, err
	}
	
	name, err := req.RequireString("Name")
	if err != nil {
		return nil, err
	}
	
	data, err := req.RequireString("Data")
	if err != nil {
		return nil, err
	}

	editRequest := &godo.DomainRecordEditRequest{
		Type: recordType,
		Name: name,
		Data: data,
	}

	record, _, err := d.client.Domains.EditRecord(ctx, domain, recordID, editRequest)
	if err != nil {
		return nil, err
	}

	jsonRecord, err := json.MarshalIndent(record, "", "  ")
	if err != nil {
		return nil, err
	}

	return mcp.NewToolResultText(string(jsonRecord)), nil
}

func (d *DomainsTool) Tools() []server.ServerTool {
	return []server.ServerTool{
		{
			Handler: d.CreateDomain,
			Tool: mcp.NewTool("digitalocean-domain-create",
				mcp.WithDescription("Create a new domain"),
				mcp.WithString("Name", mcp.Required(), mcp.Description("Name of the domain")),
				mcp.WithString("IPAddress", mcp.Required(), mcp.Description("IP address for the domain")),
			),
		},
		{
			Handler: d.DeleteDomain,
			Tool: mcp.NewTool("digitalocean-domain-delete",
				mcp.WithDescription("Delete a domain"),
				mcp.WithString("Name", mcp.Required(), mcp.Description("Name of the domain to delete")),
			),
		},
		{
			Handler: d.CreateRecord,
			Tool: mcp.NewTool("digitalocean-domain-record-create",
				mcp.WithDescription("Create a new domain record"),
				mcp.WithString("Domain", mcp.Required(), mcp.Description("Domain name")),
				mcp.WithString("Type", mcp.Required(), mcp.Description("Record type (e.g., A, CNAME, TXT)")),
				mcp.WithString("Name", mcp.Required(), mcp.Description("Record name")),
				mcp.WithString("Data", mcp.Required(), mcp.Description("Record data")),
			),
		},
		{
			Handler: d.DeleteRecord,
			Tool: mcp.NewTool("digitalocean-domain-record-delete",
				mcp.WithDescription("Delete a domain record"),
				mcp.WithString("Domain", mcp.Required(), mcp.Description("Domain name")),
				mcp.WithNumber("RecordID", mcp.Required(), mcp.Description("ID of the record to delete")),
			),
		},
		{
			Handler: d.EditRecord,
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
