package genai

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"

	"github.com/digitalocean/godo"
	"github.com/mark3labs/mcp-go/mcp"
	"github.com/mark3labs/mcp-go/server"

	"mcp-digitalocean/pkg/registry/common"
)

const (
	simulationDeleteScenarioSetConsentMsg = "confirm_deletion must be true. Before deleting a scenario set, present the scenario_set_uuid and that deletion is permanent, then obtain explicit consent (yes) in this conversation. Consent is required for every delete."

	simulationDeleteConfirmDescription = "Must be true only after the end user has explicitly confirmed the deletion in conversation (yes/no in chat). Omitted or false is rejected."
)

// SimulationTool provides Agent Platform simulation (scenario set + run) management tools.
type SimulationTool struct {
	client func(ctx context.Context) (*godo.Client, error)
}

// NewSimulationTool creates a new SimulationTool instance.
func NewSimulationTool(client func(ctx context.Context) (*godo.Client, error)) *SimulationTool {
	return &SimulationTool{client: client}
}

func (st *SimulationTool) getClient(ctx context.Context) (*godo.Client, error) {
	return st.client(ctx)
}

// scenarioSetCreateResult serves the two branches of
// genai-simulation-create-scenario-set. They have always answered with
// different text — the bare set inline, the set plus upload details for a file
// — so text is marshalled from whatever that branch returned, while both
// publish the one payload shape that covers either case.
func scenarioSetCreateResult(text any, payload *ScenarioSetCreateResult) (*mcp.CallToolResult, error) {
	jsonData, err := json.MarshalIndent(text, "", "  ")
	if err != nil {
		return nil, fmt.Errorf("marshal error: %w", err)
	}
	return scenarioSetCreateOut.ResultWithText(string(jsonData), payload)
}

func listOptionsFromArgs(args map[string]any) godo.ListOptions {
	opt := godo.ListOptions{}
	if page, ok := args["page"].(float64); ok {
		opt.Page = int(page)
	}
	if perPage, ok := args["per_page"].(float64); ok {
		opt.PerPage = int(perPage)
	}
	return opt
}

// listScenarioSets lists team-owned scenario sets.
func (st *SimulationTool) listScenarioSets(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	args := req.GetArguments()

	client, err := st.getClient(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to get DigitalOcean client: %w", err)
	}

	opt := &godo.ScenarioSetListOptions{ListOptions: listOptionsFromArgs(args)}
	if search, ok := args["search"].(string); ok {
		opt.Search = strings.TrimSpace(search)
	}
	for _, status := range parseStringSlice(args["statuses"]) {
		opt.Statuses = append(opt.Statuses, godo.ScenarioSetStatus(status))
	}

	output, _, err := client.AgentPlatform.ListScenarioSets(ctx, opt)
	if err != nil {
		return mcp.NewToolResultErrorFromErr("failed to list scenario sets", err), nil
	}

	sets := []*godo.ScenarioSet{}
	if output != nil {
		sets = output.ScenarioSets
	}
	return scenarioSetListOut.Result(scenarioSetList{ScenarioSets: sets, Count: len(sets)})
}

// getScenarioSet retrieves a scenario set by UUID.
func (st *SimulationTool) getScenarioSet(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	uuid, _ := req.GetArguments()["scenario_set_uuid"].(string)
	uuid = strings.TrimSpace(uuid)
	if uuid == "" {
		return mcp.NewToolResultError("scenario_set_uuid is required"), nil
	}

	client, err := st.getClient(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to get DigitalOcean client: %w", err)
	}

	output, _, err := client.AgentPlatform.GetScenarioSet(ctx, uuid)
	if err != nil {
		return mcp.NewToolResultErrorFromErr("failed to get scenario set", err), nil
	}
	return scenarioSetOut.Result(output)
}

// createScenarioSet creates a scenario set from inline scenarios or a JSONL file upload.
func (st *SimulationTool) createScenarioSet(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	args := req.GetArguments()

	name, _ := args["name"].(string)
	name = strings.TrimSpace(name)
	if name == "" {
		return mcp.NewToolResultError("name is required"), nil
	}

	filePath, _ := args["file_path"].(string)
	filePath = strings.TrimSpace(filePath)
	_, hasScenarios := args["scenarios"]

	if (filePath != "") == hasScenarios {
		return mcp.NewToolResultError("provide exactly one of scenarios or file_path"), nil
	}

	client, err := st.getClient(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to get DigitalOcean client: %w", err)
	}

	if filePath != "" {
		result, err := uploadAndCreateScenarioSetFromFile(ctx, client, name, filePath)
		if err != nil {
			return mcp.NewToolResultErrorFromErr("failed to create scenario set from file", err), nil
		}
		return scenarioSetCreateResult(result, result)
	}

	scenarios, err := parseScenariosFromArgs(args["scenarios"])
	if err != nil {
		return mcp.NewToolResultError(err.Error()), nil
	}

	output, _, err := client.AgentPlatform.CreateScenarioSet(ctx, &godo.CreateScenarioSetRequest{
		Name:      name,
		Scenarios: scenarios,
	})
	if err != nil {
		return mcp.NewToolResultErrorFromErr("failed to create scenario set", err), nil
	}
	// The inline branch carries no upload, so those fields stay empty.
	return scenarioSetCreateResult(output, &ScenarioSetCreateResult{ScenarioSet: output})
}

// generateScenarioSet dispatches goal-driven scenario generation.
func (st *SimulationTool) generateScenarioSet(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	args := req.GetArguments()

	name, _ := args["name"].(string)
	name = strings.TrimSpace(name)
	goal, _ := args["goal_description"].(string)
	goal = strings.TrimSpace(goal)
	if name == "" || goal == "" {
		return mcp.NewToolResultError("name and goal_description are required"), nil
	}

	client, err := st.getClient(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to get DigitalOcean client: %w", err)
	}

	generateReq := &godo.GenerateScenarioSetRequest{
		Name:               name,
		GoalDescription:    goal,
		GeneratorModelUUID: strings.TrimSpace(stringArg(args, "generator_model_uuid")),
	}
	if n, ok := args["num_scenarios"].(float64); ok && n > 0 {
		generateReq.NumScenarios = uint32(n)
	}

	output, _, err := client.AgentPlatform.GenerateScenarioSet(ctx, generateReq)
	if err != nil {
		return mcp.NewToolResultErrorFromErr("failed to generate scenario set", err), nil
	}
	return scenarioSetOut.Result(output)
}

// listScenarios lists scenarios within a scenario set.
func (st *SimulationTool) listScenarios(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	args := req.GetArguments()

	uuid, _ := args["scenario_set_uuid"].(string)
	uuid = strings.TrimSpace(uuid)
	if uuid == "" {
		return mcp.NewToolResultError("scenario_set_uuid is required"), nil
	}

	client, err := st.getClient(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to get DigitalOcean client: %w", err)
	}

	opt := &godo.ScenarioListOptions{ListOptions: listOptionsFromArgs(args)}
	if search, ok := args["search"].(string); ok {
		opt.Search = strings.TrimSpace(search)
	}

	output, _, err := client.AgentPlatform.ListScenarios(ctx, uuid, opt)
	if err != nil {
		return mcp.NewToolResultErrorFromErr("failed to list scenarios", err), nil
	}

	scenarios := []*godo.Scenario{}
	if output != nil {
		scenarios = output.Scenarios
	}
	return scenarioListOut.Result(scenarioList{Scenarios: scenarios, Count: len(scenarios)})
}

// getScenarioSetDownloadURL returns a presigned download URL for a scenario set's JSONL.
func (st *SimulationTool) getScenarioSetDownloadURL(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	uuid, _ := req.GetArguments()["scenario_set_uuid"].(string)
	uuid = strings.TrimSpace(uuid)
	if uuid == "" {
		return mcp.NewToolResultError("scenario_set_uuid is required"), nil
	}

	client, err := st.getClient(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to get DigitalOcean client: %w", err)
	}

	output, _, err := client.AgentPlatform.GetScenarioSetDownloadURL(ctx, uuid)
	if err != nil {
		return mcp.NewToolResultErrorFromErr("failed to get scenario set download URL", err), nil
	}
	return scenarioSetDownloadURLOut.Result(output)
}

// updateScenarioSet updates a scenario set name and/or replaces its scenarios.
func (st *SimulationTool) updateScenarioSet(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	args := req.GetArguments()

	uuid, _ := args["scenario_set_uuid"].(string)
	uuid = strings.TrimSpace(uuid)
	if uuid == "" {
		return mcp.NewToolResultError("scenario_set_uuid is required"), nil
	}

	name, _ := args["name"].(string)
	name = strings.TrimSpace(name)
	_, hasScenarios := args["scenarios"]
	if name == "" && !hasScenarios {
		return mcp.NewToolResultError("at least one of name or scenarios must be set"), nil
	}

	updateReq := &godo.UpdateScenarioSetRequest{
		ScenarioSetUUID: uuid,
		Name:            name,
	}
	if hasScenarios {
		scenarios, err := parseScenariosFromArgs(args["scenarios"])
		if err != nil {
			return mcp.NewToolResultError(err.Error()), nil
		}
		updateReq.Scenarios = scenarios
	}

	client, err := st.getClient(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to get DigitalOcean client: %w", err)
	}

	output, _, err := client.AgentPlatform.UpdateScenarioSet(ctx, uuid, updateReq)
	if err != nil {
		return mcp.NewToolResultErrorFromErr("failed to update scenario set", err), nil
	}
	return scenarioSetOut.Result(output)
}

// deleteScenarioSet deletes a scenario set by UUID.
func (st *SimulationTool) deleteScenarioSet(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	args := req.GetArguments()

	uuid, _ := args["scenario_set_uuid"].(string)
	uuid = strings.TrimSpace(uuid)
	if uuid == "" {
		return mcp.NewToolResultError("scenario_set_uuid is required"), nil
	}
	if confirm, _ := args["confirm_deletion"].(bool); !confirm {
		return mcp.NewToolResultError(simulationDeleteScenarioSetConsentMsg), nil
	}

	client, err := st.getClient(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to get DigitalOcean client: %w", err)
	}

	output, resp, err := client.AgentPlatform.DeleteScenarioSet(ctx, uuid)
	if err != nil {
		return mcp.NewToolResultErrorFromErr("failed to delete scenario set", err), nil
	}
	if resp != nil && resp.StatusCode >= 400 {
		return mcp.NewToolResultError(fmt.Sprintf("failed to delete scenario set: status %d", resp.StatusCode)), nil
	}
	return scenarioSetDeletedOut.Result(output)
}

// listScenarioLibrary lists platform-curated scenario library entries.
func (st *SimulationTool) listScenarioLibrary(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	args := req.GetArguments()

	client, err := st.getClient(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to get DigitalOcean client: %w", err)
	}

	opt := &godo.ScenarioLibraryListOptions{ListOptions: listOptionsFromArgs(args)}
	if search, ok := args["search"].(string); ok {
		opt.Search = strings.TrimSpace(search)
	}
	if category, ok := args["category"].(string); ok {
		opt.Category = strings.TrimSpace(category)
	}

	output, _, err := client.AgentPlatform.ListScenarioLibrary(ctx, opt)
	if err != nil {
		return mcp.NewToolResultErrorFromErr("failed to list scenario library", err), nil
	}

	entries := []*godo.ScenarioLibraryEntry{}
	if output != nil {
		entries = output.Scenarios
	}
	return scenarioLibraryListOut.Result(scenarioLibraryList{Scenarios: entries, Count: len(entries)})
}

// listScenarioLibraryScenarios lists scenarios within a library entry.
func (st *SimulationTool) listScenarioLibraryScenarios(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	args := req.GetArguments()

	uuid, _ := args["library_scenario_uuid"].(string)
	uuid = strings.TrimSpace(uuid)
	if uuid == "" {
		return mcp.NewToolResultError("library_scenario_uuid is required"), nil
	}

	client, err := st.getClient(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to get DigitalOcean client: %w", err)
	}

	opt := &godo.ScenarioListOptions{ListOptions: listOptionsFromArgs(args)}
	if search, ok := args["search"].(string); ok {
		opt.Search = strings.TrimSpace(search)
	}

	output, _, err := client.AgentPlatform.ListScenarioLibraryScenarios(ctx, uuid, opt)
	if err != nil {
		return mcp.NewToolResultErrorFromErr("failed to list scenario library scenarios", err), nil
	}

	scenarios := []*godo.Scenario{}
	if output != nil {
		scenarios = output.Scenarios
	}
	return scenarioListOut.Result(scenarioList{Scenarios: scenarios, Count: len(scenarios)})
}

// createScenarioSetFromLibrary materializes a library entry into a team-owned scenario set.
func (st *SimulationTool) createScenarioSetFromLibrary(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	args := req.GetArguments()

	uuid, _ := args["library_scenario_uuid"].(string)
	uuid = strings.TrimSpace(uuid)
	if uuid == "" {
		return mcp.NewToolResultError("library_scenario_uuid is required"), nil
	}

	client, err := st.getClient(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to get DigitalOcean client: %w", err)
	}

	output, _, err := client.AgentPlatform.CreateScenarioSetFromLibrary(ctx, uuid, &godo.CreateScenarioSetFromLibraryRequest{
		LibraryScenarioUUID: uuid,
		Name:                strings.TrimSpace(stringArg(args, "name")),
	})
	if err != nil {
		return mcp.NewToolResultErrorFromErr("failed to create scenario set from library", err), nil
	}
	return scenarioSetOut.Result(output)
}

// createRun starts a simulation+evaluation run: a scenario set against a
// candidate agent with required evaluation metrics. There is no simulation-only
// run; metrics are mandatory. After create, run lifecycle (list/get/rename/
// cancel/delete) uses ModelEvaluationRun tools with evaluation_run_uuid.
func (st *SimulationTool) createRun(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	args := req.GetArguments()

	scenarioSetUUID, _ := args["scenario_set_uuid"].(string)
	scenarioSetUUID = strings.TrimSpace(scenarioSetUUID)
	agentUUID, _ := args["agent_uuid"].(string)
	agentUUID = strings.TrimSpace(agentUUID)
	if scenarioSetUUID == "" || agentUUID == "" {
		return mcp.NewToolResultError("scenario_set_uuid and agent_uuid are required"), nil
	}

	metricUUIDs := parseStringSlice(args["metric_uuids"])
	if len(metricUUIDs) == 0 {
		return mcp.NewToolResultError("metric_uuids is required: simulation always runs with evaluation metrics (get UUIDs from genai-model-eval-list-metrics)"), nil
	}
	starMetric := parseStarMetricFromArgs(args["star_metric"])

	createReq := &godo.CreateSimulationRunRequest{
		ScenarioSetUUID:        scenarioSetUUID,
		Name:                   strings.TrimSpace(stringArg(args, "name")),
		UserSimulatorModelUUID: strings.TrimSpace(stringArg(args, "user_simulator_model_uuid")),
		JudgeModelUUID:         strings.TrimSpace(stringArg(args, "judge_model_uuid")),
		AgentConfig: &godo.CandidateAgentConfig{
			AgentUUID:           agentUUID,
			AgentDeploymentUUID: strings.TrimSpace(stringArg(args, "agent_deployment_uuid")),
			Name:                strings.TrimSpace(stringArg(args, "agent_name")),
		},
		EvaluationConfig: &godo.SimulationEvaluationConfig{
			MetricUUIDs: metricUUIDs,
			StarMetric:  starMetric,
		},
	}
	if cfg, ok := args["user_simulator_config"].(map[string]any); ok {
		createReq.UserSimulatorConfig = cfg
	}
	if v, ok := args["exploration_budget"].(float64); ok && v > 0 {
		createReq.ExplorationBudget = uint32(v)
	}
	if v, ok := args["max_turns"].(float64); ok && v > 0 {
		createReq.MaxTurns = uint32(v)
	}

	client, err := st.getClient(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to get DigitalOcean client: %w", err)
	}

	output, _, err := client.AgentPlatform.CreateSimulationRun(ctx, createReq)
	if err != nil {
		return mcp.NewToolResultErrorFromErr("failed to create simulation run", err), nil
	}
	return simulationRunOut.Result(output)
}

// listEpisodes lists episodes for a simulation run.
func (st *SimulationTool) listEpisodes(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	args := req.GetArguments()

	runUUID, _ := args["run_uuid"].(string)
	runUUID = strings.TrimSpace(runUUID)
	if runUUID == "" {
		return mcp.NewToolResultError("run_uuid is required"), nil
	}

	client, err := st.getClient(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to get DigitalOcean client: %w", err)
	}

	opt := &godo.SimulationEpisodeListOptions{ListOptions: listOptionsFromArgs(args)}
	if scenarioUUID, ok := args["scenario_uuid"].(string); ok {
		opt.ScenarioUUID = strings.TrimSpace(scenarioUUID)
	}
	if search, ok := args["search"].(string); ok {
		opt.Search = strings.TrimSpace(search)
	}
	for _, status := range parseStringSlice(args["statuses"]) {
		opt.Statuses = append(opt.Statuses, godo.SimulationEpisodeStatus(status))
	}
	for _, verdict := range parseStringSlice(args["verdicts"]) {
		opt.Verdicts = append(opt.Verdicts, godo.SimulationEpisodeVerdict(verdict))
	}

	output, _, err := client.AgentPlatform.ListSimulationEpisodes(ctx, runUUID, opt)
	if err != nil {
		return mcp.NewToolResultErrorFromErr("failed to list simulation episodes", err), nil
	}

	episodes := []*godo.SimulationEpisode{}
	if output != nil {
		episodes = output.Episodes
	}
	return simulationEpisodeListOut.Result(simulationEpisodeList{Episodes: episodes, Count: len(episodes)})
}

// getEpisode retrieves a single episode within a simulation run.
func (st *SimulationTool) getEpisode(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	args := req.GetArguments()

	runUUID, _ := args["run_uuid"].(string)
	runUUID = strings.TrimSpace(runUUID)
	episodeUUID, _ := args["episode_uuid"].(string)
	episodeUUID = strings.TrimSpace(episodeUUID)
	if runUUID == "" || episodeUUID == "" {
		return mcp.NewToolResultError("run_uuid and episode_uuid are required"), nil
	}

	client, err := st.getClient(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to get DigitalOcean client: %w", err)
	}

	output, _, err := client.AgentPlatform.GetSimulationEpisode(ctx, runUUID, episodeUUID)
	if err != nil {
		return mcp.NewToolResultErrorFromErr("failed to get simulation episode", err), nil
	}
	return simulationEpisodeOut.Result(output)
}

// getEpisodeTrajectory retrieves the parsed trajectory JSON for an episode.
func (st *SimulationTool) getEpisodeTrajectory(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	args := req.GetArguments()

	runUUID, _ := args["run_uuid"].(string)
	runUUID = strings.TrimSpace(runUUID)
	episodeUUID, _ := args["episode_uuid"].(string)
	episodeUUID = strings.TrimSpace(episodeUUID)
	if runUUID == "" || episodeUUID == "" {
		return mcp.NewToolResultError("run_uuid and episode_uuid are required"), nil
	}

	client, err := st.getClient(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to get DigitalOcean client: %w", err)
	}

	output, _, err := client.AgentPlatform.GetSimulationEpisodeTrajectory(ctx, runUUID, episodeUUID)
	if err != nil {
		return mcp.NewToolResultErrorFromErr("failed to get simulation episode trajectory", err), nil
	}
	return simulationTrajectoryOut.Result(output)
}

// getEpisodeTrajectoryURL returns a presigned download URL for an episode's trajectory JSON.
func (st *SimulationTool) getEpisodeTrajectoryURL(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	args := req.GetArguments()

	runUUID, _ := args["run_uuid"].(string)
	runUUID = strings.TrimSpace(runUUID)
	episodeUUID, _ := args["episode_uuid"].(string)
	episodeUUID = strings.TrimSpace(episodeUUID)
	if runUUID == "" || episodeUUID == "" {
		return mcp.NewToolResultError("run_uuid and episode_uuid are required"), nil
	}

	client, err := st.getClient(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to get DigitalOcean client: %w", err)
	}

	output, _, err := client.AgentPlatform.GetSimulationEpisodeTrajectoryURL(ctx, runUUID, episodeUUID)
	if err != nil {
		return mcp.NewToolResultErrorFromErr("failed to get simulation episode trajectory URL", err), nil
	}
	return simulationTrajectoryURLOut.Result(output)
}

// Tools returns the list of server tools for simulation management.
func (st *SimulationTool) Tools() []server.ServerTool {
	return []server.ServerTool{
		{
			Handler: st.listScenarioSets,
			Tool: mcp.NewTool(
				"genai-simulation-list-scenario-sets",
				common.WithHints(common.HintsRead),
				common.WithRisk(common.RiskLow),
				scenarioSetListOut.Schema(),
				mcp.WithDescription("List team-owned simulation scenario sets. Use returned scenario_set_uuid with genai-simulation-create-run (sim+eval; metrics required). Run lifecycle uses genai-model-eval-list-runs / get-run / update-run / cancel-run / delete-run with evaluation_run_uuid."),
				mcp.WithString("search", mcp.Description("Optional search text filter")),
				mcp.WithArray("statuses", mcp.Description("Optional status filters, e.g. SCENARIO_SET_STATUS_READY"), mcp.Items(map[string]any{"type": "string"})),
				mcp.WithNumber("page", mcp.Description("Page number for pagination (default: 1)")),
				mcp.WithNumber("per_page", mcp.Description("Results per page (default: 20)")),
			),
		},
		{
			Handler: st.getScenarioSet,
			Tool: mcp.NewTool(
				"genai-simulation-get-scenario-set",
				common.WithHints(common.HintsRead),
				common.WithRisk(common.RiskLow),
				scenarioSetOut.Schema(),
				mcp.WithDescription("Get a single scenario set by UUID."),
				mcp.WithString("scenario_set_uuid", mcp.Required(), mcp.Description("UUID of the scenario set")),
			),
		},
		{
			Handler: st.createScenarioSet,
			Tool: mcp.NewTool(
				"genai-simulation-create-scenario-set",
				common.WithHints(common.HintsCreate),
				common.WithRisk(common.RiskLow),
				scenarioSetCreateOut.Schema(),
				mcp.WithDescription("Create a scenario set from either inline scenarios or a local JSONL file. Provide exactly one of scenarios or file_path. JSONL rows must include a non-empty name field."),
				mcp.WithString("name", mcp.Required(), mcp.Description("Name for the scenario set")),
				mcp.WithArray("scenarios", mcp.Description("Inline scenario objects. Each object should include name (required), and optional description, user_persona, stopping_criteria, max_turns, exploration_budget."), mcp.Items(map[string]any{"type": "object"})),
				mcp.WithString("file_path", mcp.Description("Path to a local .jsonl scenario set file to upload")),
			),
		},
		{
			Handler: st.generateScenarioSet,
			Tool: mcp.NewTool(
				"genai-simulation-generate-scenario-set",
				common.WithHints(common.HintsCreate),
				common.WithRisk(common.RiskMedium),
				scenarioSetOut.Schema(),
				mcp.WithDescription("Generate a scenario set from a natural-language goal description. Generation is asynchronous; poll genai-simulation-get-scenario-set until status is SCENARIO_SET_STATUS_READY."),
				mcp.WithString("name", mcp.Required(), mcp.Description("Name for the generated scenario set")),
				mcp.WithString("goal_description", mcp.Required(), mcp.Description("Natural-language goal used to generate scenarios")),
				mcp.WithNumber("num_scenarios", mcp.Description("Number of scenarios to generate")),
				mcp.WithString("generator_model_uuid", mcp.Description("Optional UUID of the model used for generation")),
			),
		},
		{
			Handler: st.listScenarios,
			Tool: mcp.NewTool(
				"genai-simulation-list-scenarios",
				common.WithHints(common.HintsRead),
				common.WithRisk(common.RiskLow),
				scenarioListOut.Schema(),
				mcp.WithDescription("List scenarios within a team-owned scenario set."),
				mcp.WithString("scenario_set_uuid", mcp.Required(), mcp.Description("UUID of the scenario set")),
				mcp.WithString("search", mcp.Description("Optional search text filter")),
				mcp.WithNumber("page", mcp.Description("Page number for pagination (default: 1)")),
				mcp.WithNumber("per_page", mcp.Description("Results per page (default: 20)")),
			),
		},
		{
			Handler: st.getScenarioSetDownloadURL,
			Tool: mcp.NewTool(
				"genai-simulation-get-scenario-set-download-url",
				common.WithHints(common.HintsRead),
				common.WithRisk(common.RiskLow),
				scenarioSetDownloadURLOut.Schema(),
				mcp.WithDescription("Get a short-lived presigned download URL for a scenario set's canonical JSONL file."),
				mcp.WithString("scenario_set_uuid", mcp.Required(), mcp.Description("UUID of the scenario set")),
			),
		},
		{
			Handler: st.updateScenarioSet,
			Tool: mcp.NewTool(
				"genai-simulation-update-scenario-set",
				common.WithHints(common.HintsToggle),
				common.WithRisk(common.RiskLow),
				scenarioSetOut.Schema(),
				mcp.WithDescription("Update a scenario set name and/or replace its scenarios. At least one of name or scenarios must be provided."),
				mcp.WithString("scenario_set_uuid", mcp.Required(), mcp.Description("UUID of the scenario set")),
				mcp.WithString("name", mcp.Description("New name for the scenario set")),
				mcp.WithArray("scenarios", mcp.Description("Replacement scenario objects (same shape as create)"), mcp.Items(map[string]any{"type": "object"})),
			),
		},
		{
			Handler: st.deleteScenarioSet,
			Tool: mcp.NewTool(
				"genai-simulation-delete-scenario-set",
				common.WithHints(common.HintsDelete),
				common.WithRisk(common.RiskMedium),
				scenarioSetDeletedOut.Schema(),
				mcp.WithDescription("Delete a scenario set by UUID. Deletion is permanent.\n\n"+
					"CONSENT REQUIRED (every delete): Do not call with confirm_deletion: true until the user has explicitly agreed in chat. "+
					"Present the scenario_set_uuid and that deletion is permanent; ask for yes/no."),
				mcp.WithString("scenario_set_uuid", mcp.Required(), mcp.Description("UUID of the scenario set to delete")),
				mcp.WithBoolean("confirm_deletion", mcp.Required(), mcp.Description(simulationDeleteConfirmDescription)),
			),
		},
		{
			Handler: st.listScenarioLibrary,
			Tool: mcp.NewTool(
				"genai-simulation-list-scenario-library",
				common.WithHints(common.HintsRead),
				common.WithRisk(common.RiskLow),
				scenarioLibraryListOut.Schema(),
				mcp.WithDescription("List platform-curated scenario library entries. Use library_scenario_uuid with genai-simulation-create-scenario-set-from-library."),
				mcp.WithString("category", mcp.Description("Optional category filter")),
				mcp.WithString("search", mcp.Description("Optional search text filter")),
				mcp.WithNumber("page", mcp.Description("Page number for pagination (default: 1)")),
				mcp.WithNumber("per_page", mcp.Description("Results per page (default: 20)")),
			),
		},
		{
			Handler: st.listScenarioLibraryScenarios,
			Tool: mcp.NewTool(
				"genai-simulation-list-scenario-library-scenarios",
				common.WithHints(common.HintsRead),
				common.WithRisk(common.RiskLow),
				scenarioListOut.Schema(),
				mcp.WithDescription("List scenarios within a platform-curated scenario library entry."),
				mcp.WithString("library_scenario_uuid", mcp.Required(), mcp.Description("UUID of the library scenario entry")),
				mcp.WithString("search", mcp.Description("Optional search text filter")),
				mcp.WithNumber("page", mcp.Description("Page number for pagination (default: 1)")),
				mcp.WithNumber("per_page", mcp.Description("Results per page (default: 20)")),
			),
		},
		{
			Handler: st.createScenarioSetFromLibrary,
			Tool: mcp.NewTool(
				"genai-simulation-create-scenario-set-from-library",
				common.WithHints(common.HintsCreate),
				common.WithRisk(common.RiskLow),
				scenarioSetOut.Schema(),
				mcp.WithDescription("Materialize a platform library entry into a team-owned scenario set. Returns a scenario_set_uuid usable with genai-simulation-create-run."),
				mcp.WithString("library_scenario_uuid", mcp.Required(), mcp.Description("UUID of the library scenario entry")),
				mcp.WithString("name", mcp.Description("Optional name for the created team scenario set")),
			),
		},
		{
			Handler: st.createRun,
			Tool: mcp.NewTool(
				"genai-simulation-create-run",
				common.WithHints(common.HintsCreate),
				common.WithRisk(common.RiskMedium),
				simulationRunOut.Schema(),
				mcp.WithDescription("Start a simulation+evaluation run: execute a scenario set against a candidate agent with required evaluation metrics. There is no simulation-only run. Requires scenario_set_uuid, agent_uuid, and metric_uuids. The response includes evaluation_run_uuid — use that with genai-model-eval-get-run / update-run / cancel-run / delete-run (and genai-model-eval-list-runs) for run lifecycle. Solo dataset evaluation uses genai-model-eval-create-run instead."),
				mcp.WithString("scenario_set_uuid", mcp.Required(), mcp.Description("UUID of the scenario set to run")),
				mcp.WithString("agent_uuid", mcp.Required(), mcp.Description("UUID of the candidate agent under test")),
				mcp.WithArray("metric_uuids", mcp.Required(), mcp.Description("Evaluation metric UUIDs (required). Simulation always runs with evaluation. Get UUIDs from genai-model-eval-list-metrics."), mcp.Items(map[string]any{"type": "string"})),
				mcp.WithString("name", mcp.Description("Optional name for this run")),
				mcp.WithString("agent_deployment_uuid", mcp.Description("Optional specific agent deployment UUID")),
				mcp.WithString("agent_name", mcp.Description("Optional display name for the candidate agent")),
				mcp.WithString("user_simulator_model_uuid", mcp.Description("Optional UUID of the user-simulator model")),
				mcp.WithString("judge_model_uuid", mcp.Description("Optional UUID of the judge model")),
				mcp.WithObject("user_simulator_config", mcp.Description("Optional user-simulator configuration object")),
				mcp.WithNumber("exploration_budget", mcp.Description("Optional exploration budget for episodes")),
				mcp.WithNumber("max_turns", mcp.Description("Optional max turns per episode")),
				mcp.WithObject("star_metric", mcp.Description("Optional primary success metric: metric_uuid and optional success_threshold")),
			),
		},
		{
			Handler: st.listEpisodes,
			Tool: mcp.NewTool(
				"genai-simulation-list-episodes",
				common.WithHints(common.HintsRead),
				common.WithRisk(common.RiskLow),
				simulationEpisodeListOut.Schema(),
				mcp.WithDescription("List episodes (individual scenario executions) for a simulation run. Filter by scenario, status, or verdict."),
				mcp.WithString("run_uuid", mcp.Required(), mcp.Description("UUID of the simulation run")),
				mcp.WithString("scenario_uuid", mcp.Description("Optional filter by scenario UUID")),
				mcp.WithArray("statuses", mcp.Description("Optional episode status filters"), mcp.Items(map[string]any{"type": "string"})),
				mcp.WithArray("verdicts", mcp.Description("Optional verdict filters, e.g. SIMULATION_EPISODE_VERDICT_SUCCESS"), mcp.Items(map[string]any{"type": "string"})),
				mcp.WithString("search", mcp.Description("Optional search text filter")),
				mcp.WithNumber("page", mcp.Description("Page number for pagination (default: 1)")),
				mcp.WithNumber("per_page", mcp.Description("Results per page (default: 20)")),
			),
		},
		{
			Handler: st.getEpisode,
			Tool: mcp.NewTool(
				"genai-simulation-get-episode",
				common.WithHints(common.HintsRead),
				common.WithRisk(common.RiskLow),
				simulationEpisodeOut.Schema(),
				mcp.WithDescription("Get a single simulation episode by run UUID and episode UUID."),
				mcp.WithString("run_uuid", mcp.Required(), mcp.Description("UUID of the simulation run")),
				mcp.WithString("episode_uuid", mcp.Required(), mcp.Description("UUID of the episode")),
			),
		},
		{
			Handler: st.getEpisodeTrajectory,
			Tool: mcp.NewTool(
				"genai-simulation-get-episode-trajectory",
				common.WithHints(common.HintsRead),
				common.WithRisk(common.RiskLow),
				simulationTrajectoryOut.Schema(),
				mcp.WithDescription("Get the parsed trajectory JSON for an episode, including messages, tool calls, judge result, and evaluation metrics."),
				mcp.WithString("run_uuid", mcp.Required(), mcp.Description("UUID of the simulation run")),
				mcp.WithString("episode_uuid", mcp.Required(), mcp.Description("UUID of the episode")),
			),
		},
		{
			Handler: st.getEpisodeTrajectoryURL,
			Tool: mcp.NewTool(
				"genai-simulation-get-episode-trajectory-url",
				common.WithHints(common.HintsRead),
				common.WithRisk(common.RiskLow),
				simulationTrajectoryURLOut.Schema(),
				mcp.WithDescription("Get a short-lived presigned download URL for an episode's trajectory JSON file."),
				mcp.WithString("run_uuid", mcp.Required(), mcp.Description("UUID of the simulation run")),
				mcp.WithString("episode_uuid", mcp.Required(), mcp.Description("UUID of the episode")),
			),
		},
	}
}
