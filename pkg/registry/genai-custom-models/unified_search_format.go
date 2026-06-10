package genaicustommodels

import (
	"fmt"
	"sort"
	"strings"

	"github.com/digitalocean/godo"
)

// modelListDisplayInstruction tells clients to show every table row (including STATUS_FAILED).
const modelListDisplayInstruction = "**Display requirement:** Show the complete markdown table below exactly as returned. " +
	"Every model must appear on its own row with UUID. Do not group by status, omit failed models, or replace rows with a summary.\n\n"

// formatUnifiedSearchTables renders catalog and custom models as two markdown tables.
// Each model is exactly one table row; rows are never combined.
func formatUnifiedSearchTables(query string, catalog []CatalogSearchRow, custom []CustomSearchRow) string {
	if err := validateOneRowPerModel(catalog, custom); err != nil {
		panic(fmt.Sprintf("unified search invariant violated: %v", err))
	}

	var sb strings.Builder
	sb.WriteString("# Unified Model Search\n\n")
	if strings.TrimSpace(query) == "" {
		sb.WriteString("**Query:** (all models)\n\n")
	} else {
		sb.WriteString(fmt.Sprintf("**Query:** %s\n\n", query))
	}

	sb.WriteString(fmt.Sprintf("## Model Catalog (%d models)\n\n", len(catalog)))
	sb.WriteString(renderCatalogTable(catalog))
	sb.WriteString("\n")

	sb.WriteString(fmt.Sprintf("## Custom Models (%d models)\n\n", len(custom)))
	sb.WriteString(modelListDisplayInstruction)
	sb.WriteString(renderCustomTable(sortCustomSearchRows(custom)))

	return sb.String()
}

// formatCustomModelsList renders all custom models in a single table (one row per model).
func formatCustomModelsList(statusFilter string, rows []CustomSearchRow) string {
	if err := validateUniqueRows(rows, func(r CustomSearchRow) string { return r.UUID }); err != nil {
		panic(fmt.Sprintf("custom models list invariant violated: %v", err))
	}

	rows = sortCustomSearchRows(rows)

	var sb strings.Builder
	sb.WriteString("# Custom Models\n\n")
	if strings.TrimSpace(statusFilter) != "" {
		sb.WriteString(fmt.Sprintf("**Status filter:** %s\n\n", statusFilter))
	}
	sb.WriteString(fmt.Sprintf("## Custom Models (%d models)\n\n", len(rows)))
	sb.WriteString(modelListDisplayInstruction)
	sb.WriteString(renderCustomTable(rows))
	return sb.String()
}

func sortCustomSearchRows(rows []CustomSearchRow) []CustomSearchRow {
	sorted := append([]CustomSearchRow(nil), rows...)
	sort.Slice(sorted, func(i, j int) bool {
		si, sj := customStatusSortKey(sorted[i].Status), customStatusSortKey(sorted[j].Status)
		if si != sj {
			return si < sj
		}
		return strings.ToLower(sorted[i].Name) < strings.ToLower(sorted[j].Name)
	})
	return sorted
}

func customStatusSortKey(status string) int {
	switch status {
	case string(CustomModelStatusReady):
		return 0
	case string(CustomModelStatusImporting):
		return 1
	case string(CustomModelStatusFailed):
		return 2
	case string(CustomModelStatusDeleted):
		return 3
	default:
		return 4
	}
}

func validateOneRowPerModel(catalog []CatalogSearchRow, custom []CustomSearchRow) error {
	if err := validateUniqueRows(catalog, func(r CatalogSearchRow) string { return r.UUID }); err != nil {
		return fmt.Errorf("catalog: %w", err)
	}
	if err := validateUniqueRows(custom, func(r CustomSearchRow) string { return r.UUID }); err != nil {
		return fmt.Errorf("custom: %w", err)
	}
	return nil
}

func validateUniqueRows[T any](rows []T, uuid func(T) string) error {
	seen := make(map[string]struct{}, len(rows))
	for i, row := range rows {
		id := strings.TrimSpace(uuid(row))
		if id == "" {
			return fmt.Errorf("row %d has empty uuid", i)
		}
		if _, ok := seen[id]; ok {
			return fmt.Errorf("duplicate uuid %q", id)
		}
		seen[id] = struct{}{}
	}
	return nil
}

func renderCatalogTable(rows []CatalogSearchRow) string {
	headers := []string{
		"UUID", "Name", "Source", "Provider", "Type",
		"Context Window", "Capabilities", "Input Modalities", "Output Modalities",
	}
	var sb strings.Builder
	sb.WriteString(markdownTableHeader(headers))
	if len(rows) == 0 {
		return sb.String()
	}
	for _, r := range rows {
		sb.WriteString(markdownTableRow([]string{
			r.UUID,
			r.Name,
			r.Source,
			r.Provider,
			r.Type,
			r.ContextWindow,
			joinCSV(r.Capabilities),
			joinCSV(r.InputModalities),
			joinCSV(r.OutputModalities),
		}))
	}
	return sb.String()
}

func renderCustomTable(rows []CustomSearchRow) string {
	headers := []string{
		"UUID", "Name", "Source", "Status", "Architecture",
		"Input Modalities", "Output Modalities",
	}
	var sb strings.Builder
	sb.WriteString(markdownTableHeader(headers))
	if len(rows) == 0 {
		return sb.String()
	}
	for _, r := range rows {
		sb.WriteString(markdownTableRow([]string{
			r.UUID,
			r.Name,
			r.Source,
			r.Status,
			r.Architecture,
			joinCSV(r.InputModalities),
			joinCSV(r.OutputModalities),
		}))
	}
	return sb.String()
}

func markdownTableHeader(cols []string) string {
	var sb strings.Builder
	sb.WriteString("| ")
	sb.WriteString(strings.Join(cols, " | "))
	sb.WriteString(" |\n| ")
	seps := make([]string, len(cols))
	for i := range seps {
		seps[i] = "---"
	}
	sb.WriteString(strings.Join(seps, " | "))
	sb.WriteString(" |\n")
	return sb.String()
}

func markdownTableRow(cells []string) string {
	escaped := make([]string, len(cells))
	for i, c := range cells {
		escaped[i] = escapeMarkdownCell(c)
	}
	return "| " + strings.Join(escaped, " | ") + " |\n"
}

func escapeMarkdownCell(s string) string {
	s = strings.ReplaceAll(s, "\n", " ")
	s = strings.ReplaceAll(s, "|", "\\|")
	return strings.TrimSpace(s)
}

func joinCSV(items []string) string {
	if len(items) == 0 {
		return ""
	}
	return strings.Join(items, ", ")
}

func toCustomSearchRow(cm *CustomModel) CustomSearchRow {
	return CustomSearchRow{
		UUID:             cm.UUID,
		Name:             cm.Name,
		Source:           sourceCustom,
		Status:           string(cm.Status),
		Architecture:     cm.Architecture,
		InputModalities:  append([]string(nil), cm.InputModalities...),
		OutputModalities: append([]string(nil), cm.OutputModalities...),
	}
}

func toCatalogSearchRow(model *godo.Model) CatalogSearchRow {
	var inputMod, outputMod []string
	if model.Modalities != nil {
		inputMod = append([]string(nil), model.Modalities.Input...)
		outputMod = append([]string(nil), model.Modalities.Output...)
	}
	return CatalogSearchRow{
		UUID:             model.Uuid,
		Name:             model.Name,
		Source:           sourceCatalog,
		Provider:         model.Provider,
		Type:             model.Type,
		ContextWindow:    model.ContextWindow,
		Capabilities:     append([]string(nil), model.Capabilities...),
		InputModalities:  inputMod,
		OutputModalities: outputMod,
	}
}
