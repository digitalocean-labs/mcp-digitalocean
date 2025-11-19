package networking

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/netip"

	"github.com/digitalocean/godo"
	"github.com/mark3labs/mcp-go/mcp"
	"github.com/mark3labs/mcp-go/server"
)

// ReservedIPTool provides tools for managing reserved IPs
type ReservedIPTool struct {
	client func(ctx context.Context) (*godo.Client, error)
}

// NewReservedIPTool creates a new ReservedIPTool
func NewReservedIPTool(client func(ctx context.Context) (*godo.Client, error)) *ReservedIPTool {
	return &ReservedIPTool{
		client: client,
	}
}

// getReservedIP fetches reserved IPv4 or IPv6 information by IP
func (t *ReservedIPTool) getReservedIP(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	ip, ok := req.GetArguments()["IP"].(string)
	if !ok || ip == "" {
		return mcp.NewToolResultError("IPv4 address is required"), nil
	}

	netip, err := netip.ParseAddr(ip)
	if err != nil {
		return mcp.NewToolResultError("invalid IP address format"), nil
	}
	var reservedIP any

	client, err := t.client(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to get DigitalOcean client: %w", err)
	}

	if netip.Is4() {
		reservedIP, _, err = client.ReservedIPs.Get(ctx, ip)
	} else if netip.Is6() {
		reservedIP, _, err = client.ReservedIPV6s.Get(ctx, ip)
	} else {
		return mcp.NewToolResultError("unsupported IP address type"), nil
	}
	if err != nil {
		return mcp.NewToolResultErrorFromErr("api error", err), nil
	}
	jsonData, err := json.MarshalIndent(reservedIP, "", "  ")
	if err != nil {
		return nil, fmt.Errorf("marshal error: %w", err)
	}
	return mcp.NewToolResultText(string(jsonData)), nil
}

// listReservedIPs lists reserved IP addresses with pagination
func (t *ReservedIPTool) listReservedIPs(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	page := 1
	perPage := 20
	if v, ok := req.GetArguments()["Page"].(float64); ok && v > 0 {
		page = int(v)
	}
	if v, ok := req.GetArguments()["PerPage"].(float64); ok && v > 0 {
		perPage = int(v)
	}

	opts := &godo.ListOptions{Page: page, PerPage: perPage}
	var ips any
	var err error
	ipType := req.GetArguments()["Type"].(string) // "ipv4" or "ipv6"

	client, err := t.client(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to get DigitalOcean client: %w", err)
	}

	switch ipType {
	case "ipv4":
		ips, _, err = client.ReservedIPs.List(ctx, opts)
	case "ipv6":
		ips, _, err = client.ReservedIPV6s.List(ctx, opts)
	default:
		return mcp.NewToolResultErrorFromErr("invalid IP type. Use 'ipv4' or 'ipv6'", errors.New("invalid IP type")), nil
	}
	if err != nil {
		return mcp.NewToolResultErrorFromErr("api error", err), nil
	}
	jsonData, err := json.MarshalIndent(ips, "", "  ")
	if err != nil {
		return nil, fmt.Errorf("marshal error: %w", err)
	}
	return mcp.NewToolResultText(string(jsonData)), nil
}

// reserveIP reserves a new IPv4 or IPv6
func (t *ReservedIPTool) reserveIP(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	region := req.GetArguments()["Region"].(string)
	ipType := req.GetArguments()["Type"].(string) // "ipv4" or "ipv6"

	var reservedIP any
	var err error

	client, err := t.client(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to get DigitalOcean client: %w", err)
	}

	switch ipType {
	case "ipv4":
		reservedIP, _, err = client.ReservedIPs.Create(ctx, &godo.ReservedIPCreateRequest{Region: region})
	case "ipv6":
		reservedIP, _, err = client.ReservedIPV6s.Create(ctx, &godo.ReservedIPV6CreateRequest{Region: region})
	default:
		return mcp.NewToolResultErrorFromErr("invalid IP type. Use 'ipv4' or 'ipv6'", errors.New("invalid IP type")), nil
	}

	if err != nil {
		return mcp.NewToolResultErrorFromErr("api error", err), nil
	}

	jsonData, err := json.MarshalIndent(reservedIP, "", "  ")
	if err != nil {
		return nil, fmt.Errorf("marshal error: %w", err)
	}

	return mcp.NewToolResultText(string(jsonData)), nil
}

// releaseIP releases a reserved IPv4 or IPv6
func (t *ReservedIPTool) releaseIP(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	ip := req.GetArguments()["IP"].(string)
	ipType := req.GetArguments()["Type"].(string) // "ipv4" or "ipv6"

	var err error

	client, err := t.client(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to get DigitalOcean client: %w", err)
	}

	switch ipType {
	case "ipv4":
		_, err = client.ReservedIPs.Delete(ctx, ip)
	case "ipv6":
		_, err = client.ReservedIPV6s.Delete(ctx, ip)
	default:
		return mcp.NewToolResultErrorFromErr("invalid IP type. Use 'ipv4' or 'ipv6'", errors.New("invalid IP type")), nil
	}

	if err != nil {
		return mcp.NewToolResultErrorFromErr("api error", err), nil
	}

	return mcp.NewToolResultText("reserved IP released successfully"), nil
}

// assignIP assigns a reserved IP to a droplet
func (t *ReservedIPTool) assignIP(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	ip := req.GetArguments()["IP"].(string)
	dropletID := int(req.GetArguments()["DropletID"].(float64))
	ipType := req.GetArguments()["Type"].(string) // "ipv4" or "ipv6"

	var action *godo.Action
	var err error

	client, err := t.client(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to get DigitalOcean client: %w", err)
	}

	switch ipType {
	case "ipv4":
		action, _, err = client.ReservedIPActions.Assign(ctx, ip, dropletID)
	case "ipv6":
		action, _, err = client.ReservedIPV6Actions.Assign(ctx, ip, dropletID)
	default:
		return mcp.NewToolResultErrorFromErr("invalid IP type. Use 'ipv4' or 'ipv6'", errors.New("invalid IP type")), nil
	}

	if err != nil {
		return mcp.NewToolResultErrorFromErr("api error", err), nil
	}

	jsonData, err := json.MarshalIndent(action, "", "  ")
	if err != nil {
		return nil, fmt.Errorf("marshal error: %w", err)
	}

	return mcp.NewToolResultText(string(jsonData)), nil
}

// unassignIP unassigns a reserved IP from a droplet
func (t *ReservedIPTool) unassignIP(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	ip := req.GetArguments()["IP"].(string)
	ipType := req.GetArguments()["Type"].(string) // "ipv4" or "ipv6"

	var action *godo.Action
	var err error

	client, err := t.client(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to get DigitalOcean client: %w", err)
	}

	switch ipType {
	case "ipv4":
		action, _, err = client.ReservedIPActions.Unassign(ctx, ip)
	case "ipv6":
		action, _, err = client.ReservedIPV6Actions.Unassign(ctx, ip)
	default:
		return mcp.NewToolResultErrorFromErr("invalid IP type. Use 'ipv4' or 'ipv6'", errors.New("invalid IP type")), nil
	}

	if err != nil {
		return mcp.NewToolResultErrorFromErr("api error", err), nil
	}

	jsonData, err := json.MarshalIndent(action, "", "  ")
	if err != nil {
		return nil, fmt.Errorf("marshal error: %w", err)
	}

	return mcp.NewToolResultText(string(jsonData)), nil
}

// Tools returns a list of tools for managing reserved IPs
func (t *ReservedIPTool) Tools() []server.ServerTool {
	return []server.ServerTool{
		{
			Handler: t.getReservedIP,
			Tool: mcp.NewTool("reserved-ip-get",
				mcp.WithDescription("Get reserved IPv4 or IPv6 information by IP"),
				mcp.WithString("IP", mcp.Required(), mcp.Description("The reserved IPv4 or IPv6 address")),
			),
		},
		{
			Handler: t.listReservedIPs,
			Tool: mcp.NewTool("reserved-ip-list",
				mcp.WithDescription("List reserved IPv4 or IPv6 addresses with pagination"),
				mcp.WithString("Type", mcp.Required(), mcp.Description("Type of IP to list ('ipv4' or 'ipv6')")),
				mcp.WithNumber("Page", mcp.DefaultNumber(1), mcp.Description("Page number (default: 1)")),
				mcp.WithNumber("PerPage", mcp.DefaultNumber(20), mcp.Description("Items per page (default: 20)")),
			),
		},
		{
			Handler: t.reserveIP,
			Tool: mcp.NewTool("reserved-ip-reserve",
				mcp.WithDescription("Reserve a new IPv4 or IPv6"),
				mcp.WithString("Region", mcp.Required(), mcp.Description("Region to reserve the IP in")),
				mcp.WithString("Type", mcp.Required(), mcp.Description("Type of IP to reserve ('ipv4' or 'ipv6')")),
			),
		},
		{
			Handler: t.releaseIP,
			Tool: mcp.NewTool("reserved-ip-release",
				mcp.WithDescription("Release a reserved IPv4 or IPv6"),
				mcp.WithString("IP", mcp.Required(), mcp.Description("The reserved IP to release")),
				mcp.WithString("Type", mcp.Required(), mcp.Description("Type of IP to release ('ipv4' or 'ipv6')")),
			),
		},
		{
			Handler: t.assignIP,
			Tool: mcp.NewTool("reserved-ip-assign",
				mcp.WithDescription("Assign a reserved IP to a droplet"),
				mcp.WithString("IP", mcp.Required(), mcp.Description("The reserved IP to assign")),
				mcp.WithNumber("DropletID", mcp.Required(), mcp.Description("The ID of the droplet to assign the IP to")),
				mcp.WithString("Type", mcp.Required(), mcp.Description("Type of IP to assign ('ipv4' or 'ipv6')")),
			),
		},
		{
			Handler: t.unassignIP,
			Tool: mcp.NewTool("reserved-ip-unassign",
				mcp.WithDescription("Unassign a reserved IP from a droplet"),
				mcp.WithString("IP", mcp.Required(), mcp.Description("The reserved IP to unassign")),
				mcp.WithString("Type", mcp.Required(), mcp.Description("Type of IP to unassign ('ipv4' or 'ipv6')")),
			),
		},
	}
}
