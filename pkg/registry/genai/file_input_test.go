package genai

import (
	"context"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestResolveFileInput_path(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "data.csv")
	require.NoError(t, os.WriteFile(path, []byte("input\nhello\n"), 0o600))

	resolved, err := resolveFileInput(context.Background(), map[string]any{
		"file_path": path,
	}, createDatasetFileKeys)
	require.NoError(t, err)
	require.Equal(t, "data.csv", resolved.FileName)
	require.Equal(t, []byte("input\nhello\n"), resolved.Data)
	require.Equal(t, path, resolved.Display)
}

func TestResolveFileInput_content(t *testing.T) {
	resolved, err := resolveFileInput(context.Background(), map[string]any{
		"file_content": "input\nhello\n",
		"file_name":    "inline.csv",
	}, createDatasetFileKeys)
	require.NoError(t, err)
	require.Equal(t, "inline.csv", resolved.FileName)
	require.Equal(t, []byte("input\nhello\n"), resolved.Data)
}

func TestResolveFileInput_contentRequiresFileName(t *testing.T) {
	_, err := resolveFileInput(context.Background(), map[string]any{
		"file_content": "input\nhello\n",
	}, createDatasetFileKeys)
	require.Error(t, err)
	require.Contains(t, err.Error(), "file_name")
}

func TestResolveFileInput_url(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte("input\nfrom-url\n"))
	}))
	t.Cleanup(srv.Close)

	resolved, err := resolveFileInput(context.Background(), map[string]any{
		"file_url":  srv.URL + "/datasets/queries.csv",
		"file_name": "",
	}, createDatasetFileKeys)
	require.NoError(t, err)
	require.Equal(t, "queries.csv", resolved.FileName)
	require.Equal(t, []byte("input\nfrom-url\n"), resolved.Data)
}

func TestResolveFileInput_exactlyOneSource(t *testing.T) {
	_, err := resolveFileInput(context.Background(), map[string]any{}, createDatasetFileKeys)
	require.Error(t, err)
	require.Contains(t, err.Error(), "missing dataset file source")

	_, err = resolveFileInput(context.Background(), map[string]any{
		"file_path":    "/tmp/a.csv",
		"file_content": "x",
		"file_name":    "a.csv",
	}, createDatasetFileKeys)
	require.Error(t, err)
	require.Contains(t, err.Error(), "exactly one")
}

func TestResolveFileInput_rejectsNonHTTPURL(t *testing.T) {
	_, err := resolveFileInput(context.Background(), map[string]any{
		"file_url": "file:///tmp/data.csv",
	}, createDatasetFileKeys)
	require.Error(t, err)
	require.Contains(t, err.Error(), "http")
}

func TestFileSourceDisplay(t *testing.T) {
	require.Equal(t, "/tmp/a.csv", fileSourceDisplay(map[string]any{"dataset_file_path": "/tmp/a.csv"}, workflowDatasetFileKeys))
	require.Equal(t, "https://example.com/a.csv", fileSourceDisplay(map[string]any{"dataset_file_url": "https://example.com/a.csv"}, workflowDatasetFileKeys))
	require.Equal(t, "inline.csv", fileSourceDisplay(map[string]any{
		"dataset_file_content": "x",
		"dataset_file_name":    "inline.csv",
	}, workflowDatasetFileKeys))
}
