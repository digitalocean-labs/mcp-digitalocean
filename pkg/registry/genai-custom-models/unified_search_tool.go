package genaicustommodels

import (
	"context"
	"fmt"
	"strings"
	"sync"

	"github.com/digitalocean/godo"
	"github.com/mark3labs/mcp-go/mcp"
)

const (
	sourceCatalog = "catalog"
	sourceCustom  = "custom"
)

// unifiedSearch searches the model catalog and custom models in parallel, then returns
// two markdown tables (one row per model, never combined). Catalog matches use
// GradientAI.SearchModels; custom matches are ranked client-side by relevance.
func (cmt *CustomModelsTool) unifiedSearch(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	query, _ := req.GetArguments()["query"].(string)

	client, err := cmt.client(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to get DigitalOcean client: %w", err)
	}

	type catalogResult struct {
		models []CatalogSearchRow
		err    error
	}
	type customResult struct {
		models []CustomSearchRow
		err    error
	}

	var wg sync.WaitGroup
	catalogCh := make(chan catalogResult, 1)
	customCh := make(chan customResult, 1)

	wg.Add(2)
	go func() {
		defer wg.Done()
		models, err := fetchCatalogModels(ctx, client, query)
		catalogCh <- catalogResult{models: models, err: err}
	}()
	go func() {
		defer wg.Done()
		models, err := fetchCustomModels(ctx, client, query)
		customCh <- customResult{models: models, err: err}
	}()

	wg.Wait()
	close(catalogCh)
	close(customCh)

	cr := <-catalogCh
	cu := <-customCh

	var catalogModels []CatalogSearchRow
	var customModels []CustomSearchRow
	var errors []string

	if cr.err != nil {
		errors = append(errors, fmt.Sprintf("catalog: %s", cr.err))
	} else {
		catalogModels = cr.models
	}
	if cu.err != nil {
		errors = append(errors, fmt.Sprintf("custom: %s", cu.err))
	} else {
		customModels = cu.models
	}

	if len(errors) == 2 {
		return mcp.NewToolResultError(fmt.Sprintf("both sources failed: %s", strings.Join(errors, "; "))), nil
	}

	if catalogModels == nil {
		catalogModels = []CatalogSearchRow{}
	}
	if customModels == nil {
		customModels = []CustomSearchRow{}
	}

	text := formatUnifiedSearchTables(query, catalogModels, customModels)
	if len(errors) == 1 {
		text = fmt.Sprintf("partial results (one source failed: %s)\n\n%s", errors[0], text)
	}

	return mcp.NewToolResultText(text), nil
}

// fetchCatalogModels searches the catalog and returns one row per matching model UUID.
func fetchCatalogModels(ctx context.Context, client *godo.Client, query string) ([]CatalogSearchRow, error) {
	uuids, _, err := client.GradientAI.SearchModels(ctx, query)
	if err != nil {
		return nil, fmt.Errorf("failed to search catalog: %w", err)
	}

	type result struct {
		row CatalogSearchRow
		ok  bool
	}

	const maxConcurrent = 20
	sem := make(chan struct{}, maxConcurrent)
	results := make([]result, len(uuids))

	var wg sync.WaitGroup
	for i, uuid := range uuids {
		wg.Add(1)
		go func(idx int, id string) {
			defer wg.Done()
			sem <- struct{}{}
			defer func() { <-sem }()

			model, _, getErr := client.GradientAI.GetModelByUUID(ctx, id)
			if getErr != nil || model == nil {
				return
			}
			results[idx] = result{row: toCatalogSearchRow(model), ok: true}
		}(i, uuid)
	}
	wg.Wait()

	rows := make([]CatalogSearchRow, 0, len(uuids))
	for _, r := range results {
		if r.ok {
			rows = append(rows, r.row)
		}
	}
	return rows, nil
}

// fetchCustomModels lists custom models and returns one row per match, ranked by relevance.
func fetchCustomModels(ctx context.Context, client *godo.Client, query string) ([]CustomSearchRow, error) {
	models, _, err := listCustomModels(ctx, client, nil)
	if err != nil {
		return nil, err
	}

	matched := filterAndRankCustomModels(models, query)
	rows := make([]CustomSearchRow, 0, len(matched))
	for _, cm := range matched {
		rows = append(rows, toCustomSearchRow(cm))
	}
	return rows, nil
}
