//go:build integration

package testing

import (
	"context"
	"encoding/json"
	"testing"

	"github.com/digitalocean/godo"
	"github.com/mark3labs/mcp-go/mcp"
	"github.com/stretchr/testify/require"
)

func TestListDomains(t *testing.T) {
	ctx := context.Background()
	c := initializeClient(ctx, t)
	defer c.Close()

	resp, err := c.CallTool(context.Background(), mcp.CallToolRequest{
		Params: mcp.CallToolParams{
			Name: "domain-list",
			Arguments: map[string]interface{}{
				"Page":    1,
				"PerPage": 10,
			},
		},
	})

	require.NoError(t, err)
	require.NotNil(t, resp)
	if resp.IsError {
		t.Fatalf("Tool call returned error: %v", resp.Content)
	}

	var domains []godo.Domain
	domainsJSON := resp.Content[0].(mcp.TextContent).Text
	err = json.Unmarshal([]byte(domainsJSON), &domains)
	require.NoError(t, err)

	t.Logf("Found %d domains", len(domains))
}

func TestListFirewalls(t *testing.T) {
	ctx := context.Background()
	c := initializeClient(ctx, t)
	defer c.Close()

	resp, err := c.CallTool(context.Background(), mcp.CallToolRequest{
		Params: mcp.CallToolParams{
			Name: "firewall-list",
			Arguments: map[string]interface{}{
				"Page":    1,
				"PerPage": 10,
			},
		},
	})

	require.NoError(t, err)
	require.NotNil(t, resp)
	require.False(t, resp.IsError)

	var firewalls []godo.Firewall
	firewallsJSON := resp.Content[0].(mcp.TextContent).Text
	err = json.Unmarshal([]byte(firewallsJSON), &firewalls)
	require.NoError(t, err)

	t.Logf("Found %d firewalls", len(firewalls))
}

func TestListLoadBalancers(t *testing.T) {
	ctx := context.Background()
	c := initializeClient(ctx, t)
	defer c.Close()

	resp, err := c.CallTool(context.Background(), mcp.CallToolRequest{
		Params: mcp.CallToolParams{
			Name: "load-balancer-list",
			Arguments: map[string]interface{}{
				"Page":    1,
				"PerPage": 10,
			},
		},
	})

	require.NoError(t, err)
	require.NotNil(t, resp)
	require.False(t, resp.IsError)

	var loadBalancers []godo.LoadBalancer
	loadBalancersJSON := resp.Content[0].(mcp.TextContent).Text
	err = json.Unmarshal([]byte(loadBalancersJSON), &loadBalancers)
	require.NoError(t, err)

	t.Logf("Found %d load balancers", len(loadBalancers))
}

func TestListReservedIPs(t *testing.T) {
	ctx := context.Background()
	c := initializeClient(ctx, t)
	defer c.Close()

	resp, err := c.CallTool(context.Background(), mcp.CallToolRequest{
		Params: mcp.CallToolParams{
			Name: "reserved-ip-list",
			Arguments: map[string]interface{}{
				"Type":    "ipv4",
				"Page":    1,
				"PerPage": 10,
			},
		},
	})

	require.NoError(t, err)
	require.NotNil(t, resp)
	if resp.IsError {
		t.Fatalf("Tool call returned error: %v", resp.Content)
	}

	var reservedIPs []godo.ReservedIP
	reservedIPsJSON := resp.Content[0].(mcp.TextContent).Text
	err = json.Unmarshal([]byte(reservedIPsJSON), &reservedIPs)
	require.NoError(t, err)

	t.Logf("Found %d reserved IPs", len(reservedIPs))
}

func TestListVPCs(t *testing.T) {
	ctx := context.Background()
	c := initializeClient(ctx, t)
	defer c.Close()

	resp, err := c.CallTool(context.Background(), mcp.CallToolRequest{
		Params: mcp.CallToolParams{
			Name: "vpc-list",
			Arguments: map[string]interface{}{
				"Page":    1,
				"PerPage": 10,
			},
		},
	})

	require.NoError(t, err)
	require.NotNil(t, resp)
	require.False(t, resp.IsError)

	var vpcs []godo.VPC
	vpcsJSON := resp.Content[0].(mcp.TextContent).Text
	err = json.Unmarshal([]byte(vpcsJSON), &vpcs)
	require.NoError(t, err)

	t.Logf("Found %d VPCs", len(vpcs))
}

func TestListCertificates(t *testing.T) {
	ctx := context.Background()
	c := initializeClient(ctx, t)
	defer c.Close()

	resp, err := c.CallTool(context.Background(), mcp.CallToolRequest{
		Params: mcp.CallToolParams{
			Name: "certificate-list",
			Arguments: map[string]interface{}{
				"Page":    1,
				"PerPage": 10,
			},
		},
	})

	require.NoError(t, err)
	require.NotNil(t, resp)
	require.False(t, resp.IsError)

	var certificates []godo.Certificate
	certificatesJSON := resp.Content[0].(mcp.TextContent).Text
	err = json.Unmarshal([]byte(certificatesJSON), &certificates)
	require.NoError(t, err)

	t.Logf("Found %d certificates", len(certificates))
}

func TestListVPCPeerings(t *testing.T) {
	ctx := context.Background()
	c := initializeClient(ctx, t)
	defer c.Close()

	resp, err := c.CallTool(context.Background(), mcp.CallToolRequest{
		Params: mcp.CallToolParams{
			Name: "vpc-peering-list",
			Arguments: map[string]interface{}{
				"Page":    1,
				"PerPage": 10,
			},
		},
	})

	require.NoError(t, err)
	require.NotNil(t, resp)
	if resp.IsError {
		t.Fatalf("Tool call returned error: %v", resp.Content)
	}

	var vpcPeerings []godo.VPCPeering
	vpcPeeringsJSON := resp.Content[0].(mcp.TextContent).Text
	err = json.Unmarshal([]byte(vpcPeeringsJSON), &vpcPeerings)
	require.NoError(t, err)

	t.Logf("Found %d VPC peerings", len(vpcPeerings))
}

func TestListBYOIPPrefixes(t *testing.T) {
	ctx := context.Background()
	c := initializeClient(ctx, t)
	defer c.Close()

	resp, err := c.CallTool(context.Background(), mcp.CallToolRequest{
		Params: mcp.CallToolParams{
			Name: "byoip-prefix-list",
			Arguments: map[string]interface{}{
				"Page":    1,
				"PerPage": 10,
			},
		},
	})

	require.NoError(t, err)
	require.NotNil(t, resp)
	if resp.IsError {
		t.Fatalf("Tool call returned error: %v", resp.Content)
	}

	var byoipPrefixes []godo.BYOIPPrefix
	byoipPrefixesJSON := resp.Content[0].(mcp.TextContent).Text
	err = json.Unmarshal([]byte(byoipPrefixesJSON), &byoipPrefixes)
	require.NoError(t, err)

	t.Logf("Found %d BYOIP prefixes", len(byoipPrefixes))
}
