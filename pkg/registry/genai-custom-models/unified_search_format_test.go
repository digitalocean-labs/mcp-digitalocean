package genaicustommodels

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestFormatUnifiedSearchTables_OneRowPerModel(t *testing.T) {
	catalog := []CatalogSearchRow{
		{UUID: "uuid-a", Name: "Model A", Source: sourceCatalog, Type: "chat"},
		{UUID: "uuid-b", Name: "Model B", Source: sourceCatalog, Type: "chat"},
	}
	custom := []CustomSearchRow{
		{UUID: "uuid-c", Name: "Custom C", Source: sourceCustom, Status: "STATUS_READY", Architecture: "LlamaForCausalLM"},
	}

	out := formatUnifiedSearchTables("llama", catalog, custom)

	require.Contains(t, out, "## Model Catalog (2 models)")
	require.Contains(t, out, "## Custom Models (1 models)")
	require.Equal(t, 1, strings.Count(out, "uuid-a"))
	require.Equal(t, 1, strings.Count(out, "uuid-b"))
	require.Equal(t, 1, strings.Count(out, "uuid-c"))
	require.Equal(t, 3, countTableDataRows(out), "one markdown row per model across both tables")
}

func TestFormatUnifiedSearchTables_EmptySections(t *testing.T) {
	out := formatUnifiedSearchTables("", nil, nil)
	require.Contains(t, out, "## Model Catalog (0 models)")
	require.Contains(t, out, "## Custom Models (0 models)")
	require.Contains(t, out, "| UUID | Name | Source | Status |")
}

func countTableDataRows(markdown string) int {
	count := 0
	for _, line := range strings.Split(markdown, "\n") {
		if strings.HasPrefix(line, "| ") && !strings.HasPrefix(line, "| ---") && !strings.Contains(line, "| UUID |") {
			count++
		}
	}
	return count
}

func TestFormatCustomModelsList_IncludesFailedUUIDRow(t *testing.T) {
	rows := []CustomSearchRow{
		{UUID: "ready-uuid", Name: "ready", Source: sourceCustom, Status: "STATUS_READY"},
		{UUID: "failed-uuid", Name: "failed-one", Source: sourceCustom, Status: "STATUS_FAILED", Architecture: "MistralForCausalLM"},
	}
	out := formatCustomModelsList("", rows)
	require.Contains(t, out, modelListDisplayInstruction)
	require.Contains(t, out, "| failed-uuid | failed-one | custom | STATUS_FAILED | MistralForCausalLM |")
	require.Equal(t, 2, countTableDataRows(out))
}

func TestValidateOneRowPerModel_RejectsDuplicateUUID(t *testing.T) {
	catalog := []CatalogSearchRow{
		{UUID: "same", Name: "One", Source: sourceCatalog},
		{UUID: "same", Name: "Two", Source: sourceCatalog},
	}
	require.Panics(t, func() {
		formatUnifiedSearchTables("", catalog, nil)
	})
}
