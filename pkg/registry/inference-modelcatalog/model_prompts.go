package inferencemodelcatalog

import (
	"context"
	"encoding/json"
	"fmt"
	"strconv"
	"strings"
	"sync"

	"github.com/digitalocean/godo"
	"github.com/mark3labs/mcp-go/mcp"
	"github.com/mark3labs/mcp-go/server"
)

const (
	modalityInput       = "input"
	modalityOutput      = "output"
	maxConcurrentFetch  = 20 // Number of parallel API calls
	targetMatchingCount = 10 // Stop after finding this many matches
)

// handleModelComparison compares two models side-by-side
func (m *ModelTool) handleModelComparison(ctx context.Context, req mcp.GetPromptRequest) (*mcp.GetPromptResult, error) {
	args := req.Params.Arguments
	if args == nil {
		return nil, fmt.Errorf("arguments are required")
	}

	modelUUID1, ok1 := args["ModelUUID1"]
	modelUUID2, ok2 := args["ModelUUID2"]

	if !ok1 || modelUUID1 == "" {
		return nil, fmt.Errorf("ModelUUID1 is required")
	}
	if !ok2 || modelUUID2 == "" {
		return nil, fmt.Errorf("ModelUUID2 is required")
	}

	model1, err := m.getModelMetadata(ctx, modelUUID1)
	if err != nil {
		return nil, fmt.Errorf("failed to get first model: %w", err)
	}

	model2, err := m.getModelMetadata(ctx, modelUUID2)
	if err != nil {
		return nil, fmt.Errorf("failed to get second model: %w", err)
	}

	comparisonText := formatModelComparison(model1, model2)

	return &mcp.GetPromptResult{
		Description: fmt.Sprintf("Model comparison between %s and %s", model1.Name, model2.Name),
		Messages: []mcp.PromptMessage{
			{
				Role: "user",
				Content: mcp.TextContent{
					Type: "text",
					Text: comparisonText,
				},
			},
		},
	}, nil
}

// formatModelComparison creates a structured comparison text
func formatModelComparison(model1, model2 *ModelMetadata) string {
	var sb strings.Builder

	sb.WriteString("# Model Comparison\n\n")

	// Single comprehensive comparison table
	sb.WriteString(fmt.Sprintf("| Model | %s | %s |\n", model1.Name, model2.Name))
	sb.WriteString("|---|---|---|\n")
	sb.WriteString(fmt.Sprintf("| **UUID** | %s | %s |\n", model1.UUID, model2.UUID))

	// Only show fields that have values in at least one model
	if model1.Provider != "" || model2.Provider != "" {
		sb.WriteString(fmt.Sprintf("| **Provider** | %s | %s |\n", formatOptionalString(model1.Provider), formatOptionalString(model2.Provider)))
	}
	if model1.Type != "" || model2.Type != "" {
		sb.WriteString(fmt.Sprintf("| **Type** | %s | %s |\n", formatOptionalString(model1.Type), formatOptionalString(model2.Type)))
	}
	if model1.ParameterCount > 0 || model2.ParameterCount > 0 {
		sb.WriteString(fmt.Sprintf("| **Parameter Count** | %s | %s |\n", formatParameterCount(model1.ParameterCount), formatParameterCount(model2.ParameterCount)))
	}
	if model1.ContextWindow != "" || model2.ContextWindow != "" {
		sb.WriteString(fmt.Sprintf("| **Context Window** | %s | %s |\n", formatContextWindow(model1.ContextWindow), formatContextWindow(model2.ContextWindow)))
	}
	if model1.Pricing != nil || model2.Pricing != nil {
		sb.WriteString(fmt.Sprintf("| **Input Price** | %s | %s |\n", formatPrice(model1.Pricing, true), formatPrice(model2.Pricing, true)))
		sb.WriteString(fmt.Sprintf("| **Output Price** | %s | %s |\n", formatPrice(model1.Pricing, false), formatPrice(model2.Pricing, false)))
	}
	if len(model1.BenchmarkScore) > 0 || len(model2.BenchmarkScore) > 0 {
		sb.WriteString(fmt.Sprintf("| **Benchmark Score** | %s | %s |\n", formatBenchmarkScore(model1.BenchmarkScore), formatBenchmarkScore(model2.BenchmarkScore)))
	}
	if len(model1.Capabilities) > 0 || len(model2.Capabilities) > 0 {
		sb.WriteString(fmt.Sprintf("| **Capabilities** | %s | %s |\n", formatCapabilities(model1.Capabilities), formatCapabilities(model2.Capabilities)))
	}
	if model1.Modalities != nil || model2.Modalities != nil {
		sb.WriteString(fmt.Sprintf("| **Input Modalities** | %s | %s |\n", formatModalities(model1.Modalities, modalityInput), formatModalities(model2.Modalities, modalityInput)))
		sb.WriteString(fmt.Sprintf("| **Output Modalities** | %s | %s |\n", formatModalities(model1.Modalities, modalityOutput), formatModalities(model2.Modalities, modalityOutput)))
	}
	if model1.ModelAvailability != "" || model2.ModelAvailability != "" {
		sb.WriteString(fmt.Sprintf("| **Availability** | %s | %s |\n", formatOptionalString(model1.ModelAvailability), formatOptionalString(model2.ModelAvailability)))
	}

	// Add descriptions if available
	if model1.Description != "" || model2.Description != "" {
		sb.WriteString("\n## Descriptions\n\n")
		if model1.Description != "" {
			sb.WriteString(fmt.Sprintf("**%s**: %s\n\n", model1.Name, model1.Description))
		}
		if model2.Description != "" {
			sb.WriteString(fmt.Sprintf("**%s**: %s\n\n", model2.Name, model2.Description))
		}
	}

	return sb.String()
}

// handleSearchByTask finds models matching task requirements with constraints
func (m *ModelTool) handleSearchByTask(ctx context.Context, req mcp.GetPromptRequest) (*mcp.GetPromptResult, error) {
	args := req.Params.Arguments
	if args == nil {
		return nil, fmt.Errorf("arguments are required")
	}

	task := args["Task"]

	var constraints taskConstraints

	if provider, ok := args["Provider"]; ok && provider != "" {
		constraints.provider = &provider
	}

	if deploymentType, ok := args["DeploymentType"]; ok && deploymentType != "" {
		constraints.deploymentType = &deploymentType
	}

	if minCtxWindow, ok := args["MinContextWindow"]; ok && minCtxWindow != "" {
		constraints.minContextWindow = &minCtxWindow
	}

	if maxInputPrice, ok := args["MaxInputPrice"]; ok && maxInputPrice != "" {
		constraints.maxInputPrice = &maxInputPrice
	}

	if maxOutputPrice, ok := args["MaxOutputPrice"]; ok && maxOutputPrice != "" {
		constraints.maxOutputPrice = &maxOutputPrice
	}

	client, err := m.client(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to get DigitalOcean client: %w", err)
	}

	uuids, _, err := client.GradientAI.SearchModels(ctx, "")
	if err != nil {
		return nil, fmt.Errorf("failed to search models: %w", err)
	}

	// Create a cancellable context to stop fetching once we have enough matches
	fetchCtx, cancel := context.WithCancel(ctx)
	defer cancel()

	// Fetch models in parallel with controlled concurrency
	type modelResult struct {
		model *ModelMetadata
		err   error
	}

	results := make(chan modelResult, targetMatchingCount)
	semaphore := make(chan struct{}, maxConcurrentFetch)

	var wg sync.WaitGroup
	for _, uuid := range uuids {
		wg.Add(1)
		go func(uuid string) {
			defer wg.Done()

			// Acquire semaphore or stop if cancelled
			select {
			case <-fetchCtx.Done():
				return
			case semaphore <- struct{}{}:
				defer func() { <-semaphore }()
			}

			// Fetch model metadata (will fail if context cancelled)
			model, err := m.getModelMetadata(fetchCtx, uuid)

			// Send result (non-blocking if context cancelled)
			select {
			case results <- modelResult{model: model, err: err}:
			case <-fetchCtx.Done():
			}
		}(uuid)
	}

	// Close results channel when all goroutines complete
	go func() {
		wg.Wait()
		close(results)
	}()

	// Collect and filter results
	var matchingModels []*ModelMetadata
	for result := range results {
		if result.err != nil {
			continue
		}

		// Filter by parsed constraints
		if !matchesConstraints(result.model, constraints) {
			continue
		}

		// Filter by task relevance if task is provided
		if task != "" && !matchesTask(result.model, task) {
			continue
		}

		matchingModels = append(matchingModels, result.model)

		// Stop fetching once we have enough good matches
		if len(matchingModels) >= targetMatchingCount {
			cancel() // Signal all goroutines to stop
			break    // Stop collecting results
		}
	}

	recommendationText := formatSearchResults(task, matchingModels, constraints)

	return &mcp.GetPromptResult{
		Description: fmt.Sprintf("Models for task: %s", task),
		Messages: []mcp.PromptMessage{
			{
				Role: "user",
				Content: mcp.TextContent{
					Type: "text",
					Text: recommendationText,
				},
			},
		},
	}, nil
}

// taskConstraints holds constraint filters for model search
type taskConstraints struct {
	provider         *string
	deploymentType   *string
	minContextWindow *string
	maxInputPrice    *string
	maxOutputPrice   *string
}

// matchesConstraints checks if a model matches the specified constraints
func matchesConstraints(model *ModelMetadata, constraints taskConstraints) bool {
	if constraints.provider != nil && !strings.Contains(strings.ToLower(model.Provider), strings.ToLower(*constraints.provider)) {
		return false
	}

	if constraints.deploymentType != nil && *constraints.deploymentType != "" {
		if !strings.Contains(strings.ToLower(model.ModelAvailability), strings.ToLower(*constraints.deploymentType)) {
			return false
		}
	}

	if constraints.minContextWindow != nil {
		// Parse the minimum context window from string
		if minCtxWindow, err := strconv.Atoi(*constraints.minContextWindow); err == nil {
			contextWindowInt := parseContextWindow(model.ContextWindow)
			if contextWindowInt > 0 && contextWindowInt < minCtxWindow {
				return false
			}
		}
	}

	// Check price constraints
	// If user specifies price constraints, only match models with actual (non-zero) pricing
	if constraints.maxInputPrice != nil || constraints.maxOutputPrice != nil {
		// Require pricing data to exist
		if model.Pricing == nil {
			return false
		}

		// Require non-zero pricing (exclude placeholder $0.00 models)
		if model.Pricing.InputPricePerMillion == 0 && model.Pricing.OutputPricePerMillion == 0 {
			return false
		}

		// Check input price constraint
		if constraints.maxInputPrice != nil {
			if maxPrice, err := strconv.ParseFloat(*constraints.maxInputPrice, 64); err == nil {
				if model.Pricing.InputPricePerMillion > maxPrice {
					return false
				}
			}
		}

		// Check output price constraint
		if constraints.maxOutputPrice != nil {
			if maxPrice, err := strconv.ParseFloat(*constraints.maxOutputPrice, 64); err == nil {
				if model.Pricing.OutputPricePerMillion > maxPrice {
					return false
				}
			}
		}
	}

	return true
}

// matchesTask checks if a model is relevant for the given task
func matchesTask(model *ModelMetadata, task string) bool {
	taskLower := strings.ToLower(task)

	// Collect important fields from the model
	fieldsToCheck := []string{
		model.Name,
		model.Provider,
		model.Type,
	}

	// Add capabilities to fields
	fieldsToCheck = append(fieldsToCheck, model.Capabilities...)

	// Check if task contains any of these fields
	for _, field := range fieldsToCheck {
		if field != "" && strings.Contains(taskLower, strings.ToLower(field)) {
			return true
		}
	}

	// No match - exclude this model
	return false
}

// formatSearchResults creates formatted output for search by task
func formatSearchResults(task string, models []*ModelMetadata, constraints taskConstraints) string {
	var sb strings.Builder

	sb.WriteString("# Model Search Results\n\n")
	sb.WriteString(fmt.Sprintf("**Task**: %s\n\n", task))

	sb.WriteString("## Applied Constraints\n\n")
	hasConstraints := false
	if constraints.provider != nil {
		sb.WriteString(fmt.Sprintf("- **Provider**: %s\n", *constraints.provider))
		hasConstraints = true
	}
	if constraints.deploymentType != nil {
		sb.WriteString(fmt.Sprintf("- **Deployment Type**: %s\n", *constraints.deploymentType))
		hasConstraints = true
	}
	if constraints.maxInputPrice != nil {
		sb.WriteString(fmt.Sprintf("- **Max Input Price**: $%s per million tokens\n", *constraints.maxInputPrice))
		hasConstraints = true
	}
	if constraints.maxOutputPrice != nil {
		sb.WriteString(fmt.Sprintf("- **Max Output Price**: $%s per million tokens\n", *constraints.maxOutputPrice))
		hasConstraints = true
	}
	if constraints.minContextWindow != nil {
		sb.WriteString(fmt.Sprintf("- **Min Context Window**: %s tokens\n", *constraints.minContextWindow))
		hasConstraints = true
	}
	if !hasConstraints {
		sb.WriteString("No constraints specified\n")
	}
	sb.WriteString("\n")

	if len(models) == 0 {
		sb.WriteString("## No Models Found\n\n")
		sb.WriteString("No models matched the search criteria and constraints.\n")
		return sb.String()
	}

	sb.WriteString(fmt.Sprintf("## Matching Models (%d found)\n\n", len(models)))

	for i, model := range models {
		sb.WriteString(fmt.Sprintf("### %d. %s\n\n", i+1, model.Name))
		sb.WriteString(fmt.Sprintf("- **UUID**: %s\n", model.UUID))
		if model.Provider != "" {
			sb.WriteString(fmt.Sprintf("- **Provider**: %s\n", model.Provider))
		}
		if model.Type != "" {
			sb.WriteString(fmt.Sprintf("- **Type**: %s\n", model.Type))
		}
		if model.ParameterCount > 0 {
			sb.WriteString(fmt.Sprintf("- **Parameter Count**: %s\n", formatParameterCount(model.ParameterCount)))
		}
		if model.ContextWindow != "" {
			sb.WriteString(fmt.Sprintf("- **Context Window**: %s\n", formatContextWindow(model.ContextWindow)))
		}
		if model.Pricing != nil {
			sb.WriteString(fmt.Sprintf("- **Input Price**: %s\n", formatPrice(model.Pricing, true)))
			sb.WriteString(fmt.Sprintf("- **Output Price**: %s\n", formatPrice(model.Pricing, false)))
		}
		if len(model.BenchmarkScore) > 0 && string(model.BenchmarkScore) != "null" {
			sb.WriteString(fmt.Sprintf("- **Benchmark Score**: %s\n", string(model.BenchmarkScore)))
		}
		if model.ModelAvailability != "" {
			sb.WriteString(fmt.Sprintf("- **Availability**: %s\n", model.ModelAvailability))
		}
		if len(model.Capabilities) > 0 {
			sb.WriteString(fmt.Sprintf("- **Capabilities**: %s\n", strings.Join(model.Capabilities, ", ")))
		}
		if model.Modalities != nil {
			if len(model.Modalities.Input) > 0 {
				sb.WriteString(fmt.Sprintf("- **Input Modalities**: %s\n", strings.Join(model.Modalities.Input, ", ")))
			}
			if len(model.Modalities.Output) > 0 {
				sb.WriteString(fmt.Sprintf("- **Output Modalities**: %s\n", strings.Join(model.Modalities.Output, ", ")))
			}
		}
		if model.Description != "" {
			sb.WriteString(fmt.Sprintf("- **Description**: %s\n", model.Description))
		}
		sb.WriteString("\n")
	}

	if len(models) > 0 {
		sb.WriteString("## Recommendation\n\n")
		sb.WriteString(fmt.Sprintf("**Best Match**: %s\n\n", models[0].Name))
		sb.WriteString(fmt.Sprintf("This model appears first in search results for '%s' and meets all specified constraints.\n", task))
	}

	return sb.String()
}

// Prompts returns the list of server prompts for model catalog assistance
func (m *ModelTool) Prompts() []server.ServerPrompt {
	return []server.ServerPrompt{
		{
			Handler: m.handleModelComparison,
			Prompt: mcp.NewPrompt(
				"model-comparison",
				mcp.WithPromptDescription("Compare two models side-by-side on parameters, pricing, capabilities, and availability. Provides a detailed comparison table for informed decision-making."),
				mcp.WithArgument("ModelUUID1",
					mcp.RequiredArgument(),
					mcp.ArgumentDescription("UUID of the first model to compare"),
				),
				mcp.WithArgument("ModelUUID2",
					mcp.RequiredArgument(),
					mcp.ArgumentDescription("UUID of the second model to compare"),
				),
			),
		},
		{
			Handler: m.handleSearchByTask,
			Prompt: mcp.NewPrompt(
				"search-by-task",
				mcp.WithPromptDescription("Find models matching a task description with optional constraints. Returns all matching models with recommendations."),
				mcp.WithArgument("Task",
					mcp.ArgumentDescription("Optional natural language task description (e.g., 'I need a reasoning model')"),
				),
				mcp.WithArgument("Provider",
					mcp.ArgumentDescription("Filter by provider (e.g., 'OpenAI', 'Anthropic')"),
				),
				mcp.WithArgument("DeploymentType",
					mcp.ArgumentDescription("Filter by deployment type (e.g., 'Serverless', 'Dedicated')"),
				),
				mcp.WithArgument("MinContextWindow",
					mcp.ArgumentDescription("Minimum context window size required (in tokens, e.g., '100000')"),
				),
				mcp.WithArgument("MaxInputPrice",
					mcp.ArgumentDescription("Maximum input token price per million tokens (e.g., '5.0')"),
				),
				mcp.WithArgument("MaxOutputPrice",
					mcp.ArgumentDescription("Maximum output token price per million tokens (e.g., '15.0')"),
				),
			),
		},
	}
}

// Helper functions

// formatOptionalString returns em-dash for empty strings, otherwise the value
func formatOptionalString(s string) string {
	if s == "" {
		return "—"
	}
	return s
}

func formatParameterCount(count float64) string {
	if count == 0 {
		return "—"
	}
	return fmt.Sprintf("%.1fB", count)
}

func formatCapabilities(caps []string) string {
	if len(caps) == 0 {
		return "—"
	}
	return strings.Join(caps, ", ")
}

func formatModalities(modalities *godo.ModelModalities, direction string) string {
	if modalities == nil {
		return "—"
	}
	var items []string
	switch direction {
	case modalityInput:
		items = modalities.Input
	case modalityOutput:
		items = modalities.Output
	}
	if len(items) == 0 {
		return "—"
	}
	return strings.Join(items, ", ")
}

// formatContextWindow formats context window in a consistent K format
func formatContextWindow(contextWindow string) string {
	if contextWindow == "" {
		return "—"
	}

	// Parse the value
	val := parseContextWindow(contextWindow)
	if val == 0 {
		return "—"
	}

	// Format consistently as K tokens
	if val >= 1000 {
		if val%1000 == 0 {
			return fmt.Sprintf("%dK tokens", val/1000)
		}
		return fmt.Sprintf("%.1fK tokens", float64(val)/1000.0)
	}
	return fmt.Sprintf("%d tokens", val)
}

func parseContextWindow(contextWindow string) int {
	if contextWindow == "" {
		return 0
	}

	// Use a replacer for efficient multiple string replacements
	replacer := strings.NewReplacer(
		",", "",
		" tokens", "",
		"tokens", "",
		"K", "000",
		"k", "000",
	)

	cleaned := strings.TrimSpace(replacer.Replace(contextWindow))
	val, err := strconv.Atoi(cleaned)
	if err != nil {
		return 0
	}
	return val
}

func formatPrice(pricing *godo.ModelPricing, isInput bool) string {
	if pricing == nil {
		return "—"
	}

	var price float64
	if isInput {
		price = pricing.InputPricePerMillion
	} else {
		price = pricing.OutputPricePerMillion
	}

	if price == 0 {
		return "—"
	}

	return fmt.Sprintf("$%.2f/M tokens", price)
}

func formatBenchmarkScore(benchmarkJSON json.RawMessage) string {
	if len(benchmarkJSON) == 0 || string(benchmarkJSON) == "null" {
		return "—"
	}
	return string(benchmarkJSON)
}
