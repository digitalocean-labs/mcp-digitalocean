package docs

import (
	"context"
	"testing"

	"mcp-digitalocean/internal/calltrace"

	"github.com/mark3labs/mcp-go/mcp"
	"github.com/stretchr/testify/require"
	"go.uber.org/mock/gomock"
)

func traceMap(tr *calltrace.Trace) map[string]string {
	out := map[string]string{}
	for _, a := range tr.Attrs() {
		out[a.Key] = a.Value.String()
	}
	return out
}

func TestSearchDocsRecordsResultsInTrace(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	index := &DocsIndex{Entries: []DocsEntry{
		{Title: "Droplet Quickstart", URL: "https://docs.digitalocean.com/products/droplets/getting-started/quickstart/index.html.md", Description: "Get started with Droplets."},
		{Title: "How to Create a Droplet", URL: "https://docs.digitalocean.com/products/droplets/how-to/create/", Description: "Create Droplets."},
	}}

	mock := NewMockDocsService(ctrl)
	mock.EXPECT().GetDocsIndex().Return(index, nil).Times(2)
	mock.EXPECT().ContentExcerpts(gomock.Any(), gomock.Any(), gomock.Any()).Return(nil)
	tool := setupDocsToolWithMock(mock)

	ctx, tr := calltrace.New(context.Background())
	req := mcp.CallToolRequest{Params: mcp.CallToolParams{Arguments: map[string]any{"Query": "droplet", "Limit": float64(1)}}}
	_, err := tool.searchDocs(ctx, req)
	require.NoError(t, err)
	got := traceMap(tr)
	require.Equal(t, "1", got["result_count"])
	require.NotContains(t, got["top_url"], "index.html.md")
	require.Contains(t, got["top_url"], "https://docs.digitalocean.com/products/droplets/")

	ctx, tr = calltrace.New(context.Background())
	req = mcp.CallToolRequest{Params: mcp.CallToolParams{Arguments: map[string]any{"Query": "nonexistent-term-xyz"}}}
	_, err = tool.searchDocs(ctx, req)
	require.NoError(t, err)
	require.Equal(t, map[string]string{"result_count": "0"}, traceMap(tr))
}
