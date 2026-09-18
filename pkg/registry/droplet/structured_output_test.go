package droplet

import (
	"context"
	"encoding/json"
	"testing"
	"time"

	"github.com/digitalocean/godo"
	"github.com/mark3labs/mcp-go/mcp"
	"github.com/mark3labs/mcp-go/server"
	"github.com/stretchr/testify/require"
	"go.uber.org/mock/gomock"
)

// callTool drives one tools/call through a real MCPServer with
// WithOutputSchemaValidation enabled, so the server checks the emitted
// structuredContent against the outputSchema the tool advertises. Only the
// server (or a spec-compliant client) performs that check — handler unit tests
// cannot catch a schema the payload does not satisfy.
//
// Validation is enabled only in tests, not in cmd/mcp-digitalocean, so that a
// schema mismatch in production degrades to a wrong schema rather than a
// failed tool call.
func callTool(t *testing.T, tools []server.ServerTool, name string, args map[string]any) (json.RawMessage, string) {
	t.Helper()

	svr := server.NewMCPServer("test", "test", server.WithOutputSchemaValidation())
	svr.AddTools(tools...)

	ctx := context.Background()
	svr.HandleMessage(ctx, []byte(`{"jsonrpc":"2.0","id":1,"method":"initialize",`+
		`"params":{"protocolVersion":"2025-06-18","capabilities":{},"clientInfo":{"name":"t","version":"1"}}}`))

	params, err := json.Marshal(map[string]any{"name": name, "arguments": args})
	require.NoError(t, err)
	req, err := json.Marshal(map[string]any{
		"jsonrpc": "2.0", "id": 2, "method": "tools/call", "params": json.RawMessage(params),
	})
	require.NoError(t, err)

	raw, err := json.Marshal(svr.HandleMessage(ctx, req))
	require.NoError(t, err)

	var resp struct {
		Error  *struct{ Message string } `json:"error"`
		Result struct {
			IsError           bool              `json:"isError"`
			Content           []mcp.TextContent `json:"content"`
			StructuredContent json.RawMessage   `json:"structuredContent"`
		} `json:"result"`
	}
	require.NoError(t, json.Unmarshal(raw, &resp))
	require.Nil(t, resp.Error, "server rejected %s: %s", name, raw)
	require.False(t, resp.Result.IsError, "%s returned an error result: %s", name, raw)
	require.Len(t, resp.Result.Content, 1)

	return resp.Result.StructuredContent, resp.Result.Content[0].Text
}

// A full godo.Droplet exercises the reflection cases that made the SDK's own
// schema generator give up: nested region/image/size structs and the
// godo.Timestamp-style fields underneath them.
func TestStructuredOutputForFullDropletType(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockDroplets := NewMockDropletsService(ctrl)
	mockDroplets.EXPECT().
		Get(gomock.Any(), 123).
		Return(&godo.Droplet{
			ID:       123,
			Name:     "test-droplet",
			Status:   "active",
			SizeSlug: "s-1vcpu-1gb",
			Region:   &godo.Region{Slug: "nyc3", Name: "New York 3"},
			Image:    &godo.Image{ID: 456, Slug: "ubuntu-22-04-x64"},
			Networks: &godo.Networks{
				V4: []godo.NetworkV4{{IPAddress: "10.0.0.1", Type: "private"}},
			},
		}, nil, nil).
		Times(1)

	tool := setupDropletToolWithMocks(mockDroplets, NewMockDropletActionsService(ctrl))
	structured, text := callTool(t, tool.Tools(), "droplet-get", map[string]any{"ID": float64(123)})

	var envelope struct {
		Droplet *godo.Droplet `json:"droplet"`
	}
	require.NoError(t, json.Unmarshal(structured, &envelope))
	require.Equal(t, 123, envelope.Droplet.ID)
	require.Equal(t, "nyc3", envelope.Droplet.Region.Slug)

	// Text content is still the bare droplet, not the envelope.
	var fromText *godo.Droplet
	require.NoError(t, json.Unmarshal([]byte(text), &fromText))
	require.Equal(t, envelope.Droplet, fromText)
}

// droplet-list returns a curated subset of godo.Droplet. The summary type
// replaced a map[string]any literal, and a map always serialises every key it
// holds, so a zero-valued field must still appear rather than be dropped the
// way an omitempty godo field would be.
func TestStructuredOutputForDropletListKeepsEveryField(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockDroplets := NewMockDropletsService(ctrl)
	mockDroplets.EXPECT().
		List(gomock.Any(), &godo.ListOptions{Page: 1, PerPage: 50}).
		Return([]godo.Droplet{{ID: 1, Name: "only-these-two-fields-set"}}, nil, nil).
		Times(1)

	tool := setupDropletToolWithMocks(mockDroplets, NewMockDropletActionsService(ctrl))
	structured, text := callTool(t, tool.Tools(), "droplet-list", map[string]any{})

	var envelope struct {
		Droplets []map[string]any `json:"droplets"`
	}
	require.NoError(t, json.Unmarshal(structured, &envelope))
	require.Len(t, envelope.Droplets, 1)

	for _, field := range []string{
		"id", "name", "memory", "vcpus", "disk", "region", "image", "size",
		"size_slug", "backup_ids", "next_backup_window", "snapshot_ids",
		"features", "locked", "status", "networks", "created_at", "kernel",
		"tags", "volume_ids", "vpc_uuid",
	} {
		require.Contains(t, envelope.Droplets[0], field,
			"droplet-list dropped %q, which it has always returned", field)
	}

	// gpu_partition_mode is only populated on the create response, so the
	// list summary has never carried it.
	require.NotContains(t, envelope.Droplets[0], "gpu_partition_mode")

	var fromText []map[string]any
	require.NoError(t, json.Unmarshal([]byte(text), &fromText))
	require.Equal(t, envelope.Droplets, fromText)
}

// godo.Action carries godo.Timestamp fields, which marshal as RFC3339 strings
// rather than as the struct reflection sees. The 16 single-droplet actions and
// the 3 image actions all share this contract.
func TestStructuredOutputForActionTools(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockActions := NewMockDropletActionsService(ctrl)
	mockActions.EXPECT().
		Reboot(gomock.Any(), 123).
		Return(&godo.Action{
			ID:          2001,
			Status:      "in-progress",
			Type:        "reboot",
			StartedAt:   &godo.Timestamp{Time: time.Now().UTC()},
			ResourceID:  123,
			CompletedAt: nil,
		}, nil, nil).
		Times(1)

	tool := setupDropletActionsToolWithMocks(mockActions)
	structured, _ := callTool(t, tool.Tools(), "reboot-droplet", map[string]any{"ID": float64(123)})

	var envelope struct {
		Action *godo.Action `json:"action"`
	}
	require.NoError(t, json.Unmarshal(structured, &envelope))
	require.Equal(t, 2001, envelope.Action.ID)
	require.Equal(t, "reboot", envelope.Action.Type)
	require.NotNil(t, envelope.Action.StartedAt)
}

// The by-tag actions return a collection. A nil slice serialises to null,
// which an "array" schema would reject.
func TestStructuredOutputForByTagActionsAcceptsEmptyCollections(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockActions := NewMockDropletActionsService(ctrl)
	mockActions.EXPECT().PowerCycleByTag(gomock.Any(), "web").Return(nil, nil, nil).Times(1)

	tool := setupDropletActionsToolWithMocks(mockActions)
	structured, _ := callTool(t, tool.Tools(), "power-cycle-droplets-tag", map[string]any{"Tag": "web"})
	require.JSONEq(t, `{"actions":[]}`, string(structured))
}

// size-list and image-list are the other two curated summaries.
func TestStructuredOutputForSizeList(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockSizes := NewMockSizesService(ctrl)
	mockSizes.EXPECT().
		List(gomock.Any(), &godo.ListOptions{Page: 1, PerPage: 50}).
		Return([]godo.Size{{Slug: "s-1vcpu-1gb", Available: true, PriceMonthly: 5, Regions: []string{"nyc3"}}}, nil, nil).
		Times(1)

	structured, text := callTool(t, setupSizesToolWithMock(mockSizes).Tools(), "size-list", map[string]any{})

	var envelope struct {
		Sizes []sizeSummary `json:"sizes"`
	}
	require.NoError(t, json.Unmarshal(structured, &envelope))
	require.Len(t, envelope.Sizes, 1)
	require.Equal(t, "s-1vcpu-1gb", envelope.Sizes[0].Slug)
	require.Equal(t, float64(5), envelope.Sizes[0].PriceMonthly)

	var fromText []sizeSummary
	require.NoError(t, json.Unmarshal([]byte(text), &fromText))
	require.Equal(t, envelope.Sizes, fromText)
}

func TestStructuredOutputForImageList(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockImages := NewMockImagesService(ctrl)
	mockImages.EXPECT().
		List(gomock.Any(), &godo.ListOptions{Page: 1, PerPage: 50}).
		Return([]godo.Image{{ID: 1, Slug: "ubuntu-22-04-x64", Public: true}}, nil, nil).
		Times(1)

	tool := NewImageTool(func(context.Context) (*godo.Client, error) {
		return &godo.Client{Images: mockImages}, nil
	})
	structured, _ := callTool(t, tool.Tools(), "image-list", map[string]any{})

	var envelope struct {
		Images []imageSummary `json:"images"`
	}
	require.NoError(t, json.Unmarshal(structured, &envelope))
	require.Len(t, envelope.Images, 1)
	require.Equal(t, "ubuntu-22-04-x64", envelope.Images[0].Slug)
}
