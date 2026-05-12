package docs

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestOpenAPIOperationJSON_PutApps(t *testing.T) {
	b, err := OpenAPIOperationJSON("PUT", "/v2/apps/{id}")
	require.NoError(t, err)
	require.Contains(t, string(b), `"operationId"`)
	require.Contains(t, string(b), "apps")
}

func TestExtractActions_doctl(t *testing.T) {
	md := "Run this:\n\n```doctl\n doctl apps list \n doctl apps update abc --spec spec.yaml \n```\n"
	acts := ExtractActions(md)
	require.NotEmpty(t, acts)
	found := false
	for _, a := range acts {
		if a.Method == "doctl" && a.Command == "doctl apps update abc --spec spec.yaml" {
			found = true
			break
		}
	}
	require.True(t, found, "expected doctl apps update line")
}
