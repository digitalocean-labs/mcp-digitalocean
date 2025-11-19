package networking

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/digitalocean/godo"
	"github.com/mark3labs/mcp-go/mcp"
	"github.com/mark3labs/mcp-go/server"
)

type DomainsTool struct {
	client func(ctx context.Context) (*godo.Client, error)
}

func NewDomainsTool(client func(ctx context.Context) (*godo.Client, error)) *DomainsTool {
	return &DomainsTool{
		client: client,
	}
}

// getDomain fetches domain information by name
func (d *DomainsTool) getDomain(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	name, ok := req.GetArguments()["Name"].(string)
	if !ok || name == "" {
		return mcp.NewToolResultError("Domain name is required"), nil
	}

	client, err := d.client(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to get DigitalOcean client: %w", err)
	}

	domain, _, err := client.Domains.Get(ctx, name)
	if err != nil {
		return mcp.NewToolResultErrorFromErr("api error", err), nil
	}
	jsonDomain, err := json.MarshalIndent(domain, "", "  ")
	if err != nil {
		return nil, fmt.Errorf("marshal error: %w", err)
	}
	return mcp.NewToolResultText(string(jsonDomain)), nil
}

// listDomains lists domains with pagination support
func (d *DomainsTool) listDomains(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	page := 1
	perPage := 20
	if v, ok := req.GetArguments()["Page"].(float64); ok && int(v) > 0 {
		page = int(v)
	}
	if v, ok := req.GetArguments()["PerPage"].(float64); ok && int(v) > 0 {
		perPage = int(v)
	}

	client, err := d.client(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to get DigitalOcean client: %w", err)
	}

	domains, _, err := client.Domains.List(ctx, &godo.ListOptions{Page: page, PerPage: perPage})
	if err != nil {
		return mcp.NewToolResultErrorFromErr("api error", err), nil
	}
	jsonDomains, err := json.MarshalIndent(domains, "", "  ")
	if err != nil {
		return nil, fmt.Errorf("marshal error: %w", err)
	}
	return mcp.NewToolResultText(string(jsonDomains)), nil
}

// getDomainRecord fetches a domain record by domain name and record ID
func (d *DomainsTool) getDomainRecord(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	domain, ok := req.GetArguments()["Domain"].(string)
	if !ok || domain == "" {
		return mcp.NewToolResultError("Domain name is required"), nil
	}
	recordIDf, ok := req.GetArguments()["RecordID"].(float64)
	if !ok {
		return mcp.NewToolResultError("RecordID is required"), nil
	}
	recordID := int(recordIDf)

	client, err := d.client(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to get DigitalOcean client: %w", err)
	}

	record, _, err := client.Domains.Record(ctx, domain, recordID)
	if err != nil {
		return mcp.NewToolResultErrorFromErr("api error", err), nil
	}
	jsonRecord, err := json.MarshalIndent(record, "", "  ")
	if err != nil {
		return nil, fmt.Errorf("marshal error: %w", err)
	}
	return mcp.NewToolResultText(string(jsonRecord)), nil
}

// listDomainRecords lists domain records for a domain with pagination support
func (d *DomainsTool) listDomainRecords(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	domain, ok := req.GetArguments()["Domain"].(string)
	if !ok || domain == "" {
		return mcp.NewToolResultError("Domain name is required"), nil
	}
	page := 1
	perPage := 20
	if v, ok := req.GetArguments()["Page"].(float64); ok && int(v) > 0 {
		page = int(v)
	}
	if v, ok := req.GetArguments()["PerPage"].(float64); ok && int(v) > 0 {
		perPage = int(v)
	}

	client, err := d.client(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to get DigitalOcean client: %w", err)
	}

	records, _, err := client.Domains.Records(ctx, domain, &godo.ListOptions{Page: page, PerPage: perPage})
	if err != nil {
		return mcp.NewToolResultErrorFromErr("api error", err), nil
	}
	jsonRecords, err := json.MarshalIndent(records, "", "  ")
	if err != nil {
		return nil, fmt.Errorf("marshal error: %w", err)
	}
	return mcp.NewToolResultText(string(jsonRecords)), nil
}

func (d *DomainsTool) createDomain(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	name := req.GetArguments()["Name"].(string)
	ipAddress := req.GetArguments()["IPAddress"].(string)

	createRequest := &godo.DomainCreateRequest{
		Name:      name,
		IPAddress: ipAddress,
	}

	client, err := d.client(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to get DigitalOcean client: %w", err)
	}

	domain, _, err := client.Domains.Create(ctx, createRequest)
	if err != nil {
		return mcp.NewToolResultErrorFromErr("api error", err), nil
	}

	jsonDomain, err := json.MarshalIndent(domain, "", "  ")
	if err != nil {
		return nil, fmt.Errorf("marshal error: %w", err)
	}

	return mcp.NewToolResultText(string(jsonDomain)), nil
}

func (d *DomainsTool) deleteDomain(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	name := req.GetArguments()["Name"].(string)

	client, err := d.client(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to get DigitalOcean client: %w", err)
	}

	_, err = client.Domains.Delete(ctx, name)
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

	client, err := d.client(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to get DigitalOcean client: %w", err)
	}

	record, _, err := client.Domains.CreateRecord(ctx, domain, createRequest)
	if err != nil {
		return mcp.NewToolResultErrorFromErr("api error", err), nil
	}

	jsonRecord, err := json.MarshalIndent(record, "", "  ")
	if err != nil {
		return nil, fmt.Errorf("marshal error: %w", err)
	}

	return mcp.NewToolResultText(string(jsonRecord)), nil
}

func (d *DomainsTool) deleteRecord(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	domain := req.GetArguments()["Domain"].(string)
	recordID := int(req.GetArguments()["RecordID"].(float64))

	client, err := d.client(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to get DigitalOcean client: %w", err)
	}

	_, err = client.Domains.DeleteRecord(ctx, domain, recordID)
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

	client, err := d.client(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to get DigitalOcean client: %w", err)
	}

	record, _, err := client.Domains.EditRecord(ctx, domain, recordID, editRequest)
	if err != nil {
		return mcp.NewToolResultErrorFromErr("api error", err), nil
	}

	jsonRecord, err := json.MarshalIndent(record, "", "  ")
	if err != nil {
		return nil, fmt.Errorf("marshal error: %w", err)
	}

	return mcp.NewToolResultText(string(jsonRecord)), nil
}

func (d *DomainsTool) Tools() []server.ServerTool {
	return []server.ServerTool{
		{
			Handler: d.getDomain,
			Tool: mcp.NewTool("domain-get",
				mcp.WithDescription("Get domain information by name"),
				mcp.WithString("Name", mcp.Required(), mcp.Description("Name of the domain")),
			),
		},
		{
			Handler: d.listDomains,
			Tool: mcp.NewTool("domain-list",
				mcp.WithDescription("List domains with pagination"),
				mcp.WithNumber("Page", mcp.DefaultNumber(1), mcp.Description("Page number")),
				mcp.WithNumber("PerPage", mcp.DefaultNumber(20), mcp.Description("Items per page")),
			),
		},
		{
			Handler: d.getDomainRecord,
			Tool: mcp.NewTool("domain-record-get",
				mcp.WithDescription("Get a domain record by domain name and record ID"),
				mcp.WithString("Domain", mcp.Required(), mcp.Description("Domain name")),
				mcp.WithNumber("RecordID", mcp.Required(), mcp.Description("ID of the domain record")),
			),
		},
		{
			Handler: d.listDomainRecords,
			Tool: mcp.NewTool("domain-record-list",
				mcp.WithDescription("List domain records for a domain with pagination"),
				mcp.WithString("Domain", mcp.Required(), mcp.Description("Domain name")),
				mcp.WithNumber("Page", mcp.DefaultNumber(1), mcp.Description("Page number")),
				mcp.WithNumber("PerPage", mcp.DefaultNumber(20), mcp.Description("Items per page")),
			),
		},
		{
			Handler: d.createDomain,
			Tool: mcp.NewTool("domain-create",
				mcp.WithDescription("Create a new domain"),
				mcp.WithString("Name", mcp.Required(), mcp.Description("Name of the domain")),
				mcp.WithString("IPAddress", mcp.Required(), mcp.Description("IP address for the domain")),
			),
		},
		{
			Handler: d.deleteDomain,
			Tool: mcp.NewTool("domain-delete",
				mcp.WithDescription("Delete a domain"),
				mcp.WithString("Name", mcp.Required(), mcp.Description("Name of the domain to delete")),
			),
		},
		{
			Handler: d.createRecord,
			Tool: mcp.NewTool("domain-record-create",
				mcp.WithDescription("Create a new domain record"),
				mcp.WithString("Domain", mcp.Required(), mcp.Description("Domain name")),
				mcp.WithString("Type", mcp.Required(), mcp.Description("Record type (e.g., A, CNAME, TXT)")),
				mcp.WithString("Name", mcp.Required(), mcp.Description("Record name")),
				mcp.WithString("Data", mcp.Required(), mcp.Description("Record data")),
			),
		},
		{
			Handler: d.deleteRecord,
			Tool: mcp.NewTool("domain-record-delete",
				mcp.WithDescription("Delete a domain record"),
				mcp.WithString("Domain", mcp.Required(), mcp.Description("Domain name")),
				mcp.WithNumber("RecordID", mcp.Required(), mcp.Description("ID of the record to delete")),
			),
		},
		{
			Handler: d.editRecord,
			Tool: mcp.NewTool("domain-record-edit",
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
