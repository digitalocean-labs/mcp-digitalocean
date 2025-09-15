package networking

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/digitalocean/godo"
	"github.com/mark3labs/mcp-go/mcp"
	"github.com/mark3labs/mcp-go/server"
)

// NATGatewayTool provides VPC NAT Gateway management tools
type NATGatewayTool struct {
	client *godo.Client
}

// NewNATGatewayTool creates a new NATGateway tool
func NewNATGatewayTool(client *godo.Client) *NATGatewayTool {
	return &NATGatewayTool{
		client: client,
	}
}

// createNATGateway creates a new VPC NAT Gateway
func (n *NATGatewayTool) createNATGateway(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	name := req.GetArguments()["Name"].(string)
	region := req.GetArguments()["Region"].(string)
	size := req.GetArguments()["Size"].(uint32)
	vpcIDs := req.GetArguments()["VPCs"].([]string)
	if name == "" || region == "" || size == 0 || len(vpcIDs) == 0 {
		return mcp.NewToolResultError("missing required argument"), nil
	}
	var vpcUUIDs []*godo.IngressVPC
	for _, vpcID := range vpcIDs {
		vpcUUIDs = append(vpcUUIDs, &godo.IngressVPC{VpcUUID: vpcID})
	}

	createRequest := &godo.VPCNATGatewayRequest{
		Name:   name,
		Type:   "PUBLIC",
		Region: region,
		Size:   size,
		VPCs:   vpcUUIDs,
	}

	// Optional timeouts
	if tpcTimeout, ok := req.GetArguments()["TCP Timeout (Seconds)"].(uint32); ok && tpcTimeout != 0 {
		createRequest.TCPTimeoutSeconds = tpcTimeout
	}
	if udpTimeout, ok := req.GetArguments()["UDP Timeout (Seconds)"].(uint32); ok && udpTimeout != 0 {
		createRequest.UDPTimeoutSeconds = udpTimeout
	}
	if icmpTimeout, ok := req.GetArguments()["ICMP Timeout (Seconds)"].(uint32); ok && icmpTimeout != 0 {
		createRequest.ICMPTimeoutSeconds = icmpTimeout
	}

	ng, _, err := n.client.VPCNATGateways.Create(ctx, createRequest)
	if err != nil {
		return mcp.NewToolResultErrorFromErr("api error", err), nil
	}

	jsonNATGateway, err := json.MarshalIndent(ng, "", "  ")
	if err != nil {
		return nil, fmt.Errorf("marshal error: %w", err)
	}

	return mcp.NewToolResultText(string(jsonNATGateway)), nil
}

// getNATGateway fetches VPC NAT Gateway information by ID
func (n *NATGatewayTool) getNATGateway(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	id, ok := req.GetArguments()["ID"].(string)
	if !ok || id == "" {
		return mcp.NewToolResultError("NAT Gateway ID is required"), nil
	}
	ng, _, err := n.client.VPCNATGateways.Get(ctx, id)
	if err != nil {
		return mcp.NewToolResultErrorFromErr("api error", err), nil
	}
	jsonNATGateway, err := json.MarshalIndent(ng, "", "  ")
	if err != nil {
		return nil, fmt.Errorf("marshal error: %w", err)
	}
	return mcp.NewToolResultText(string(jsonNATGateway)), nil
}

// listNATGateways lists VPC NAT Gateways with pagination support
func (n *NATGatewayTool) listNATGateways(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	page := 1
	perPage := 20
	if vArg, ok := req.GetArguments()["Page"].(float64); ok && int(vArg) > 0 {
		page = int(vArg)
	}
	if vArg, ok := req.GetArguments()["PerPage"].(float64); ok && int(vArg) > 0 {
		perPage = int(vArg)
	}

	ngs, _, err := n.client.VPCNATGateways.List(ctx, &godo.VPCNATGatewaysListOptions{ListOptions: godo.ListOptions{Page: page, PerPage: perPage}})
	if err != nil {
		return mcp.NewToolResultErrorFromErr("api error", err), nil
	}
	jsonNGs, err := json.MarshalIndent(ngs, "", "  ")
	if err != nil {
		return nil, fmt.Errorf("marshal error: %w", err)
	}
	return mcp.NewToolResultText(string(jsonNGs)), nil
}

// updateNATGateway updates an existing VPC NAT Gateway by ID
func (n *NATGatewayTool) updateNATGateway(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	id := req.GetArguments()["ID"].(string)
	name := req.GetArguments()["Name"].(string)

	ng, _, err := n.client.VPCNATGateways.Get(ctx, id)
	if err != nil {
		return mcp.NewToolResultErrorFromErr("api error", err), nil
	}

	if ng.Name != name {
		return mcp.NewToolResultError("NAT Gateway Name does not ID, and cannot be updated"), nil
	}

	updateRequest := &godo.VPCNATGatewayRequest{
		Name:               ng.Name,
		Type:               ng.Type,
		Region:             ng.Region,
		Size:               ng.Size,
		VPCs:               ng.VPCs,
		TCPTimeoutSeconds:  ng.TCPTimeoutSeconds,
		UDPTimeoutSeconds:  ng.UDPTimeoutSeconds,
		ICMPTimeoutSeconds: ng.ICMPTimeoutSeconds,
	}

	if size, ok := req.GetArguments()["Size"].(uint32); ok && size != 0 {
		updateRequest.Size = size
	}

	// Optional timeouts
	if tpcTimeout, ok := req.GetArguments()["TCP Timeout (Seconds)"].(uint32); ok && tpcTimeout != 0 {
		updateRequest.TCPTimeoutSeconds = tpcTimeout
	}
	if udpTimeout, ok := req.GetArguments()["UDP Timeout (Seconds)"].(uint32); ok && udpTimeout != 0 {
		updateRequest.UDPTimeoutSeconds = udpTimeout
	}
	if icmpTimeout, ok := req.GetArguments()["ICMP Timeout (Seconds)"].(uint32); ok && icmpTimeout != 0 {
		updateRequest.ICMPTimeoutSeconds = icmpTimeout
	}

	updatedNG, _, err := n.client.VPCNATGateways.Update(ctx, id, updateRequest)

	if err != nil {
		return mcp.NewToolResultErrorFromErr("api error", err), nil
	}

	jsonNATGateway, err := json.MarshalIndent(updatedNG, "", "  ")
	if err != nil {
		return nil, fmt.Errorf("marshal error: %w", err)
	}

	return mcp.NewToolResultText(string(jsonNATGateway)), nil
}

// deleteNATGateway deletes a VPC NAT Gateway by ID
func (n *NATGatewayTool) deleteNATGateway(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	ngID := req.GetArguments()["ID"].(string)

	_, err := n.client.VPCNATGateways.Delete(ctx, ngID)
	if err != nil {
		return mcp.NewToolResultErrorFromErr("api error", err), nil
	}

	return mcp.NewToolResultText("VPC NAT Gateway deleted successfully"), nil
}

// Tools returns a list of tool functions
func (n *NATGatewayTool) Tools() []server.ServerTool {
	return []server.ServerTool{
		{
			Handler: n.createNATGateway,
			Tool: mcp.NewTool("nat-gateway-create",
				mcp.WithDescription("Create a new VPC NAT Gateway"),
				mcp.WithString("Name", mcp.Required(), mcp.Description("Name of the VPC NAT Gateway")),
				mcp.WithString("Region", mcp.Required(), mcp.Description("Region (e.g., nyc3) in which the VPC NAT Gateway will be created")),
				mcp.WithNumber("Size", mcp.DefaultNumber(1), mcp.Required(), mcp.Description("The size of the VPC NAT Gateway")),
				mcp.WithString("VPCs", mcp.Required(), mcp.Description("An array of VPC IDs to attach to the VPC NAT Gateway")),
				mcp.WithNumber("TCP Timeout (Seconds)", mcp.Description("Set how long a connection can remain idle")),
				mcp.WithNumber("UDP Timeout (Seconds)", mcp.Description("Set how long a connection can remain idle")),
				mcp.WithNumber("ICMP Timeout (Seconds)", mcp.Description("Set how long a connection can remain idle")),
			),
		},
		{
			Handler: n.getNATGateway,
			Tool: mcp.NewTool("nat-gateway-get",
				mcp.WithDescription("Get VPC NAT Gateway information by ID"),
				mcp.WithString("ID", mcp.Required(), mcp.Description("ID of the VPC NAT Gateway")),
			),
		},
		{
			Handler: n.listNATGateways,
			Tool: mcp.NewTool("nat-gateway-list",
				mcp.WithDescription("List VPC NAT Gateways with pagination"),
				mcp.WithNumber("Page", mcp.DefaultNumber(1), mcp.Description("Page number")),
				mcp.WithNumber("PerPage", mcp.DefaultNumber(20), mcp.Description("Items per page")),
			),
		},
		{
			Handler: n.updateNATGateway,
			Tool: mcp.NewTool("nat-gateway-update",
				mcp.WithDescription("Update an existing VPC NAT Gateway by ID"),
				mcp.WithString("ID", mcp.Required(), mcp.Description("ID of the VPC NAT Gateway")),
				mcp.WithString("Name", mcp.Required(), mcp.Description("Name of the VPC NAT Gateway")),
				mcp.WithNumber("Size", mcp.Description("The size of the VPC NAT Gateway")),
				mcp.WithNumber("TCP Timeout (Seconds)", mcp.Description("Set how long a connection can remain idle")),
				mcp.WithNumber("UDP Timeout (Seconds)", mcp.Description("Set how long a connection can remain idle")),
				mcp.WithNumber("ICMP Timeout (Seconds)", mcp.Description("Set how long a connection can remain idle")),
			),
		},
		{
			Handler: n.deleteNATGateway,
			Tool: mcp.NewTool("nat-gateway-delete",
				mcp.WithDescription("Delete a VPC NAT Gateway by ID"),
				mcp.WithString("ID", mcp.Required(), mcp.Description("ID of the VPC NAT Gateway to delete")),
			),
		},
	}
}
