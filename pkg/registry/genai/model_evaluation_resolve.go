package genai

import (
	"context"
	"encoding/json"
	"fmt"
	"regexp"
	"strings"
	"sync"

	"github.com/digitalocean/godo"
	"github.com/mark3labs/mcp-go/mcp"
)

var evalModelUUIDPattern = regexp.MustCompile(`^[0-9a-fA-F]{8}-[0-9a-fA-F]{4}-[0-9a-fA-F]{4}-[0-9a-fA-F]{4}-[0-9a-fA-F]{12}$`)

const (
	modelEvalCandidateNameRequiredMsg = "candidate_model_name is required: pass the candidate model in the 'candidate_model_name' field (it accepts the display name, the API/inference name, or the model UUID). If you don't know the exact name, ask the user before creating an evaluation run."

	modelEvalJudgeNameRequiredMsg = "judge_model_name is required when not using eval_preset_uuid (or pass eval_preset_uuid so the judge comes from the preset)."

	modelEvalConsentStatus        = "consent_required"
	modelEvalModelSelectionStatus = "model_selection_required"

	modelEvalUserMessageRequiredMsg = "user_message is required to start the run: pass the end user's verbatim chat reply (e.g. yes) from after they saw prompt_for_user. " +
		"First call: omit user_message — tool returns a preview. Post prompt_for_user to the end user and wait for their reply in chat. " +
		"Second call: pass user_message set to that exact user reply (not assistant text). Never pass user_message on the first call."

	modelEvalUserMessageNotYesMsg = "user_message is not yes — it must be the end user's verbatim chat reply (e.g. yes). Wait for the user to reply after prompt_for_user before retrying."

	modelEvalUUIDNameMismatchMsg = "candidate uuid and candidate_model_name refer to different models; provide the exact uuid and exact name for the same model, or pass only one identifier."

	modelEvalJudgeUUIDNameMismatchMsg = "judge_model_uuid and judge_model_name refer to different models; provide the exact uuid and exact name for the same model, or pass only one identifier."

	genaiModelEvalUserMessageDescription = "The end user's verbatim chat message after they reviewed prompt_for_user (typically yes). " +
		"Copy from the user's message only — never use assistant-generated text. Omit on the first call."

	genaiModelEvalCreateRunToolDescription = "Create a model evaluation run. Provide either an eval_preset_uuid to use a preset, or provide dataset_uuid, judge_model_name, and metric_uuids for inline configuration.\n\n" +
		"MODEL NAMES: candidate_model_name accepts display or API (inference) names. When candidate_model_uuid / judge_model_uuid are omitted, UUIDs are fetched from the catalog API and shown in the consent preview.\n\n" +
		"USER CONFIRMATION (chat): (1) Call without user_message — returns prompt_for_user. Post it to the end user and wait for their chat reply. " +
		"(2) Call again with user_message set to that exact reply and the same arguments. The run is not created until step 2."

	genaiModelEvalWorkflowToolDescription = "Run a complete model evaluation workflow: upload dataset, create evaluation run, and poll for results.\n\n" +
		"MODEL NAMES and USER CONSENT: same two-step chat confirmation as genai-model-eval-create-run."
)

// EvalCatalogModel is a catalog or custom model usable for model evaluation runs.
type EvalCatalogModel struct {
	UUID        string `json:"uuid"`
	APIName     string `json:"api_name"`     // sent as candidate_model_name / judge_model_name to the evaluation API
	DisplayName string `json:"display_name"` // human-readable catalog name
	Source      string `json:"source,omitempty"`
	Status      string `json:"status,omitempty"`
}

// ModelEvalMatchCandidate is a model returned when name/uuid resolution needs disambiguation or consent.
type ModelEvalMatchCandidate struct {
	UUID        string `json:"uuid"`
	DisplayName string `json:"display_name"`
	APIName     string `json:"api_name"`
	Source      string `json:"source,omitempty"`
	Status      string `json:"status,omitempty"`
}

// ModelEvalUnresolvedOutput is returned when a model identifier is not an exact match.
type ModelEvalUnresolvedOutput struct {
	Message                 string                    `json:"message"`
	Role                    string                    `json:"role"`
	Query                   string                    `json:"query"`
	QueryField              string                    `json:"query_field"`
	Matches                 []ModelEvalMatchCandidate `json:"matches"`
	RequiresExactMatch      bool                      `json:"requires_exact_match"`
	DoNotSubstituteFromList bool                      `json:"do_not_substitute_name_or_uuid_from_matches"`
}

// ModelEvalModelsResolveOutput is returned when candidate and/or judge models need disambiguation.
type ModelEvalModelsResolveOutput struct {
	Status                string                     `json:"status"`
	Message               string                     `json:"message"`
	Candidate             *ModelEvalUnresolvedOutput `json:"candidate,omitempty"`
	Judge                 *ModelEvalUnresolvedOutput `json:"judge,omitempty"`
	StopAndAskUser        bool                       `json:"stop_and_ask_user"`
	DoNotRetryUntilUserOK bool                       `json:"do_not_retry_until_user_confirms"`
}

// ModelEvalConsentPreviewOutput is returned before the user types yes in chat.
type ModelEvalConsentPreviewOutput struct {
	Status                     string                   `json:"status"`
	Message                    string                   `json:"message"`
	PromptForUser              string                   `json:"prompt_for_user"`
	RunName                    string                   `json:"run_name"`
	DatasetUUID                string                   `json:"dataset_uuid,omitempty"`
	DatasetFilePath            string                   `json:"dataset_file_path,omitempty"`
	EvalPresetUUID             string                   `json:"eval_preset_uuid,omitempty"`
	MetricUUIDs                []string                 `json:"metric_uuids,omitempty"`
	StarMetric                 *StarMetric              `json:"star_metric,omitempty"`
	CandidateModel             ModelEvalMatchCandidate  `json:"candidate_model"`
	JudgeModel                 *ModelEvalMatchCandidate `json:"judge_model,omitempty"`
	StopAndAskUser             bool                     `json:"stop_and_ask_user"`
	DoNotRetryUntilUserOK      bool                     `json:"do_not_retry_until_user_confirms"`
	RequireUserMessageFromChat bool                     `json:"require_user_message_from_chat"`
	InstructionForAgent        string                   `json:"instruction_for_agent"`
}

// modelEvalRunConfig holds run parameters shown in the consent prompt.
type modelEvalRunConfig struct {
	RunName         string
	DatasetUUID     string
	DatasetFilePath string
	EvalPresetUUID  string
	MetricUUIDs     []string
	StarMetric      *StarMetric
}

type modelEvalResolvedModel struct {
	UUID        string
	APIName     string
	DisplayName string
	Source      string
}

type modelEvalRunModels struct {
	Candidate modelEvalResolvedModel
	Judge     *modelEvalResolvedModel
}

// listAllEvalModels fetches catalog and custom models for evaluation run resolution.
func listAllEvalModels(ctx context.Context, client *godo.Client) ([]*EvalCatalogModel, error) {
	type catalogResult struct {
		models []*EvalCatalogModel
		err    error
	}
	type customResult struct {
		models []*EvalCatalogModel
		err    error
	}

	var wg sync.WaitGroup
	catalogCh := make(chan catalogResult, 1)
	customCh := make(chan customResult, 1)

	wg.Add(2)
	go func() {
		defer wg.Done()
		models, err := fetchEvalCatalogModels(ctx, client)
		catalogCh <- catalogResult{models: models, err: err}
	}()
	go func() {
		defer wg.Done()
		models, err := fetchEvalCustomModels(ctx, client)
		customCh <- customResult{models: models, err: err}
	}()

	wg.Wait()
	close(catalogCh)
	close(customCh)

	cr := <-catalogCh
	cu := <-customCh

	if cr.err != nil && cu.err != nil {
		return nil, fmt.Errorf("catalog: %w; custom: %w", cr.err, cu.err)
	}

	merged := make([]*EvalCatalogModel, 0, len(cr.models)+len(cu.models))
	merged = append(merged, cr.models...)
	merged = append(merged, cu.models...)
	return merged, nil
}

func fetchEvalCatalogModels(ctx context.Context, client *godo.Client) ([]*EvalCatalogModel, error) {
	uuids, _, err := client.GradientAI.SearchModels(ctx, "")
	if err != nil {
		return nil, fmt.Errorf("failed to search catalog: %w", err)
	}

	type result struct {
		model *EvalCatalogModel
		err   error
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
				results[idx] = result{err: getErr}
				return
			}
			apiName, displayName := catalogModelNames(model)
			results[idx] = result{model: &EvalCatalogModel{
				UUID:        model.Uuid,
				APIName:     apiName,
				DisplayName: displayName,
				Source:      "catalog",
			}}
		}(i, uuid)
	}
	wg.Wait()

	models := make([]*EvalCatalogModel, 0, len(results))
	for _, r := range results {
		if r.model != nil {
			models = append(models, r.model)
		}
	}
	return models, nil
}

func fetchEvalCustomModels(ctx context.Context, client *godo.Client) ([]*EvalCatalogModel, error) {
	const perPage = 100
	var all []*EvalCatalogModel
	page := 1

	for {
		out, resp, err := client.GradientAI.ListCustomModels(ctx, &godo.CustomModelListOptions{
			ListOptions: godo.ListOptions{Page: page, PerPage: perPage},
		})
		if err != nil {
			return nil, fmt.Errorf("failed to list custom models: %w", err)
		}
		if resp != nil && resp.StatusCode >= 400 {
			return nil, fmt.Errorf("failed to list custom models: status %d", resp.StatusCode)
		}

		var models []*godo.CustomModel
		var meta *godo.Meta
		if out != nil {
			models = out.Models
			meta = out.Meta
		}
		if meta == nil && resp != nil {
			meta = resp.Meta
		}

		for _, cm := range models {
			if cm == nil || cm.Status == godo.CustomModelStatusDeleted {
				continue
			}
			all = append(all, &EvalCatalogModel{
				UUID:        cm.Uuid,
				APIName:     cm.Name,
				DisplayName: cm.Name,
				Source:      "custom",
				Status:      string(cm.Status),
			})
		}

		if len(models) == 0 {
			break
		}
		if meta != nil && meta.Pages > 0 && page >= meta.Pages {
			break
		}
		if len(models) < perPage {
			break
		}
		page++
	}

	return all, nil
}

func resolveEvalModelsForRun(
	ctx context.Context,
	client *godo.Client,
	candidateUUID, candidateName string,
	judgeUUID, judgeName string,
	requireJudge bool,
	models []*EvalCatalogModel,
) (*modelEvalRunModels, *ModelEvalModelsResolveOutput, error) {
	candidate, candidateUnresolved, err := resolveEvalModel(ctx, client, "candidate", candidateUUID, candidateName, models)
	if err != nil {
		return nil, nil, err
	}

	var judge *modelEvalResolvedModel
	var judgeUnresolved *ModelEvalUnresolvedOutput

	if requireJudge {
		if strings.TrimSpace(judgeName) == "" && strings.TrimSpace(judgeUUID) == "" {
			return nil, nil, fmt.Errorf("%s", modelEvalJudgeNameRequiredMsg)
		}
		judge, judgeUnresolved, err = resolveEvalModel(ctx, client, "judge", judgeUUID, judgeName, models)
		if err != nil {
			return nil, nil, err
		}
	}

	if candidateUnresolved != nil || judgeUnresolved != nil {
		return nil, buildModelEvalModelsResolveOutput(candidateUnresolved, judgeUnresolved), nil
	}

	out := &modelEvalRunModels{Candidate: *candidate}
	if judge != nil {
		out.Judge = judge
	}
	return out, nil, nil
}

func resolveEvalModel(ctx context.Context, client *godo.Client, role, uuid, name string, models []*EvalCatalogModel) (*modelEvalResolvedModel, *ModelEvalUnresolvedOutput, error) {
	uuid = strings.TrimSpace(uuid)
	name = strings.TrimSpace(name)

	switch {
	case uuid != "" && name != "":
		uuidResolved, uuidUnresolved, err := resolveEvalModelByUUID(role, uuid, models)
		if err != nil {
			return nil, nil, err
		}
		if uuidUnresolved != nil {
			return nil, uuidUnresolved, nil
		}
		if !evalModelNameMatchesExact(findEvalModelByUUID(models, uuidResolved.UUID), name) {
			msg := modelEvalUUIDNameMismatchMsg
			if role == "judge" {
				msg = modelEvalJudgeUUIDNameMismatchMsg
			}
			return nil, nil, fmt.Errorf("%s", msg)
		}
		return uuidResolved, nil, nil

	case uuid != "":
		uuidResolved, uuidUnresolved, err := resolveEvalModelByUUID(role, uuid, models)
		if err != nil {
			return nil, nil, err
		}
		if uuidUnresolved != nil {
			return nil, uuidUnresolved, nil
		}
		return uuidResolved, nil, nil

	default:
		nameResolved, nameUnresolved, err := resolveEvalModelByName(ctx, client, role, name, models)
		if err != nil {
			return nil, nil, err
		}
		if nameUnresolved != nil {
			return nil, nameUnresolved, nil
		}
		return nameResolved, nil, nil
	}
}

// lookupEvalCatalogModelByName searches the catalog API and returns the model when display or API name matches exactly.
func lookupEvalCatalogModelByName(ctx context.Context, client *godo.Client, name string) (*modelEvalResolvedModel, error) {
	if client == nil {
		return nil, fmt.Errorf("catalog client required")
	}
	name = strings.TrimSpace(name)
	if name == "" {
		return nil, fmt.Errorf("name is required")
	}

	uuids, _, err := client.GradientAI.SearchModels(ctx, name)
	if err != nil {
		return nil, fmt.Errorf("failed to search catalog: %w", err)
	}

	var exact []*modelEvalResolvedModel
	for _, id := range uuids {
		model, _, getErr := client.GradientAI.GetModelByUUID(ctx, id)
		if getErr != nil || model == nil || model.Uuid == "" {
			continue
		}
		apiName, displayName := catalogModelNames(model)
		if !evalModelNameMatchesExact(&EvalCatalogModel{DisplayName: displayName, APIName: apiName}, name) {
			continue
		}
		exact = append(exact, &modelEvalResolvedModel{
			UUID:        model.Uuid,
			APIName:     apiName,
			DisplayName: displayName,
			Source:      "catalog",
		})
	}

	switch len(exact) {
	case 1:
		return exact[0], nil
	case 0:
		return nil, fmt.Errorf("no catalog model with exact name %q", name)
	default:
		return nil, fmt.Errorf("multiple catalog models named %q; provide uuid to select one", name)
	}
}

func catalogModelNames(model *godo.Model) (apiName, displayName string) {
	displayName = model.Name
	apiName = model.InferenceName
	if apiName == "" && displayName != "" {
		apiName = displayNameToInferenceSlug(displayName)
	}
	if apiName == "" {
		apiName = displayName
	}
	if displayName == "" {
		displayName = apiName
	}
	return apiName, displayName
}

// displayNameToInferenceSlug derives the evaluation API slug when InferenceName is unset.
func displayNameToInferenceSlug(displayName string) string {
	s := strings.ToLower(strings.TrimSpace(displayName))
	return strings.ReplaceAll(s, " ", "-")
}

func evalModelNameMatchesExact(m *EvalCatalogModel, name string) bool {
	if m == nil {
		return false
	}
	return m.DisplayName == name || m.APIName == name
}

func findEvalModelByUUID(models []*EvalCatalogModel, uuid string) *EvalCatalogModel {
	for _, m := range filterActiveEvalModels(models) {
		if strings.EqualFold(m.UUID, uuid) {
			return m
		}
	}
	return nil
}

func evalModelNameMatchesPartial(m *EvalCatalogModel, queryLower string) bool {
	return strings.Contains(strings.ToLower(m.DisplayName), queryLower) ||
		strings.Contains(strings.ToLower(m.APIName), queryLower)
}

func resolveEvalModelByName(ctx context.Context, client *godo.Client, role, name string, models []*EvalCatalogModel) (*modelEvalResolvedModel, *ModelEvalUnresolvedOutput, error) {
	active := filterActiveEvalModels(models)

	var exact []*EvalCatalogModel
	for _, m := range active {
		if evalModelNameMatchesExact(m, name) {
			exact = append(exact, m)
		}
	}

	switch len(exact) {
	case 1:
		return evalModelToResolved(exact[0]), nil, nil
	case 0:
		if client != nil {
			if catalogResolved, err := lookupEvalCatalogModelByName(ctx, client, name); err == nil {
				return catalogResolved, nil, nil
			}
		}
		queryLower := strings.ToLower(name)
		var partial []*EvalCatalogModel
		for _, m := range active {
			if evalModelNameMatchesPartial(m, queryLower) {
				partial = append(partial, m)
			}
		}
		return nil, buildEvalUnresolvedOutput(role, name, "name", partial), nil
	default:
		return nil, nil, fmt.Errorf("multiple %s models named %q; provide uuid to select one", role, name)
	}
}

func validateModelEvalResolvedUUIDs(resolved *modelEvalRunModels, requireJudge bool) error {
	if resolved == nil {
		return fmt.Errorf("no models resolved for evaluation run")
	}
	if strings.TrimSpace(resolved.Candidate.UUID) == "" {
		return fmt.Errorf("candidate model uuid could not be fetched from catalog; provide candidate_model_uuid or an exact catalog model name")
	}
	if requireJudge {
		if resolved.Judge == nil || strings.TrimSpace(resolved.Judge.UUID) == "" {
			return fmt.Errorf("judge model uuid could not be fetched from catalog; provide judge_model_uuid or an exact catalog model name")
		}
	}
	return nil
}

func resolveEvalModelByUUID(role, uuid string, models []*EvalCatalogModel) (*modelEvalResolvedModel, *ModelEvalUnresolvedOutput, error) {
	active := filterActiveEvalModels(models)

	if isExactEvalModelUUID(uuid) {
		var exact []*EvalCatalogModel
		for _, m := range active {
			if strings.EqualFold(m.UUID, uuid) {
				exact = append(exact, m)
			}
		}
		switch len(exact) {
		case 1:
			return evalModelToResolved(exact[0]), nil, nil
		case 0:
			return nil, buildEvalUnresolvedOutput(role, uuid, "uuid", nil), nil
		default:
			return nil, nil, fmt.Errorf("multiple %s models with uuid %q", role, uuid)
		}
	}

	queryLower := strings.ToLower(uuid)
	var partial []*EvalCatalogModel
	for _, m := range active {
		if strings.Contains(strings.ToLower(m.UUID), queryLower) {
			partial = append(partial, m)
		}
	}
	return nil, buildEvalUnresolvedOutput(role, uuid, "uuid", partial), nil
}

func filterActiveEvalModels(models []*EvalCatalogModel) []*EvalCatalogModel {
	out := make([]*EvalCatalogModel, 0, len(models))
	for _, m := range models {
		if m == nil || m.Status == "STATUS_DELETED" {
			continue
		}
		out = append(out, m)
	}
	return out
}

func evalModelToResolved(m *EvalCatalogModel) *modelEvalResolvedModel {
	return &modelEvalResolvedModel{
		UUID:        m.UUID,
		APIName:     m.APIName,
		DisplayName: m.DisplayName,
		Source:      m.Source,
	}
}

func evalModelToMatchCandidate(m *EvalCatalogModel) ModelEvalMatchCandidate {
	return ModelEvalMatchCandidate{
		UUID:        m.UUID,
		DisplayName: m.DisplayName,
		APIName:     m.APIName,
		Source:      m.Source,
		Status:      m.Status,
	}
}

func isModelEvalUserMessageYes(userMessage string) bool {
	return strings.EqualFold(strings.TrimSpace(userMessage), "yes")
}

// checkModelEvalUserMessage returns a tool result when the end user's confirmation is missing or invalid.
// ok is true only when user_message is the user's yes reply (case-insensitive).
func checkModelEvalUserMessage(args map[string]any, resolved *modelEvalRunModels, cfg *modelEvalRunConfig) (*mcp.CallToolResult, bool) {
	userMessage := strings.TrimSpace(stringArg(args, "user_message"))
	if isModelEvalUserMessageYes(userMessage) {
		return nil, true
	}
	if userMessage != "" {
		return mcp.NewToolResultError(modelEvalUserMessageNotYesMsg), false
	}
	result, err := modelEvalUserActionResult(buildModelEvalConsentPreview(resolved, cfg))
	if err != nil {
		return mcp.NewToolResultError(err.Error()), false
	}
	return result, false
}

func parseMetricUUIDsFromArgs(args map[string]any) []string {
	metricUUIDsRaw, ok := args["metric_uuids"].([]interface{})
	if !ok {
		return nil
	}
	out := make([]string, 0, len(metricUUIDsRaw))
	for _, m := range metricUUIDsRaw {
		if s, ok := m.(string); ok && s != "" {
			out = append(out, s)
		}
	}
	return out
}

func parseModelEvalRunConfig(args map[string]any, requireJudge bool) (*modelEvalRunConfig, error) {
	runName, _ := args["name"].(string)
	runName = strings.TrimSpace(runName)
	if runName == "" {
		return nil, fmt.Errorf("name is required")
	}

	cfg := &modelEvalRunConfig{
		RunName:        runName,
		EvalPresetUUID: strings.TrimSpace(stringArg(args, "eval_preset_uuid")),
		DatasetUUID:    strings.TrimSpace(stringArg(args, "dataset_uuid")),
		MetricUUIDs:    parseMetricUUIDsFromArgs(args),
		StarMetric:     parseStarMetricArg(args),
	}
	if cfg.StarMetric == nil && len(cfg.MetricUUIDs) > 0 {
		cfg.StarMetric = defaultStarMetric(cfg.MetricUUIDs)
	}

	if cfg.EvalPresetUUID == "" {
		if cfg.DatasetUUID == "" {
			return nil, fmt.Errorf("dataset_uuid is required when not using eval_preset_uuid")
		}
		if requireJudge && len(cfg.MetricUUIDs) == 0 {
			return nil, fmt.Errorf("metric_uuids is required when not using eval_preset_uuid")
		}
	}
	return cfg, nil
}

func stringArg(args map[string]any, key string) string {
	v, _ := args[key].(string)
	return v
}

func buildModelEvalPromptForUser(resolved *modelEvalRunModels, cfg *modelEvalRunConfig) string {
	var b strings.Builder
	b.WriteString("Please confirm this model evaluation run.\n")
	b.WriteString("Catalog UUIDs below were resolved from model names (use these exact values on the confirmed run).\n\n")
	fmt.Fprintf(&b, "Candidate model\n  display_name: %s\n  api_name: %s\n  uuid: %s\n\n",
		resolved.Candidate.DisplayName, resolved.Candidate.APIName, resolved.Candidate.UUID)
	if resolved.Judge != nil {
		fmt.Fprintf(&b, "Judge model\n  display_name: %s\n  api_name: %s\n  uuid: %s\n\n",
			resolved.Judge.DisplayName, resolved.Judge.APIName, resolved.Judge.UUID)
	}
	if cfg.EvalPresetUUID != "" {
		fmt.Fprintf(&b, "Eval preset UUID: %s\n", cfg.EvalPresetUUID)
	}
	if cfg.DatasetUUID != "" {
		fmt.Fprintf(&b, "Dataset UUID: %s\n", cfg.DatasetUUID)
	}
	if cfg.DatasetFilePath != "" {
		fmt.Fprintf(&b, "Dataset file: %s\n", cfg.DatasetFilePath)
	}
	if len(cfg.MetricUUIDs) > 0 {
		b.WriteString("Metric UUIDs:\n")
		for _, id := range cfg.MetricUUIDs {
			fmt.Fprintf(&b, "  - %s\n", id)
		}
	}
	if cfg.StarMetric != nil && cfg.StarMetric.MetricUUID != "" {
		fmt.Fprintf(&b, "Star metric UUID: %s\n", cfg.StarMetric.MetricUUID)
		if cfg.StarMetric.SuccessThresholdPct != nil {
			fmt.Fprintf(&b, "Star metric success_threshold_pct: %g\n", *cfg.StarMetric.SuccessThresholdPct)
		}
	}
	fmt.Fprintf(&b, "\nRun name: %s\n\nIf this looks correct, reply **yes** in this chat to start the evaluation run.", cfg.RunName)
	return b.String()
}

func buildModelEvalConsentPreview(resolved *modelEvalRunModels, cfg *modelEvalRunConfig) *ModelEvalConsentPreviewOutput {
	out := &ModelEvalConsentPreviewOutput{
		Status:                     modelEvalConsentStatus,
		Message:                    modelEvalUserMessageRequiredMsg,
		PromptForUser:              buildModelEvalPromptForUser(resolved, cfg),
		RunName:                    cfg.RunName,
		DatasetUUID:                cfg.DatasetUUID,
		DatasetFilePath:            cfg.DatasetFilePath,
		EvalPresetUUID:             cfg.EvalPresetUUID,
		MetricUUIDs:                append([]string(nil), cfg.MetricUUIDs...),
		StarMetric:                 cfg.StarMetric,
		CandidateModel:             evalModelToMatchCandidateFromResolved(resolved.Candidate),
		StopAndAskUser:             true,
		DoNotRetryUntilUserOK:      true,
		RequireUserMessageFromChat: true,
		InstructionForAgent: "Post prompt_for_user to the end user and wait for their chat reply. " +
			"On the next call, pass user_message set to that verbatim reply (typically yes). Do not pass user_message until the user has replied.",
	}
	if resolved.Judge != nil {
		judge := evalModelToMatchCandidateFromResolved(*resolved.Judge)
		out.JudgeModel = &judge
	}
	return out
}

func evalModelToMatchCandidateFromResolved(m modelEvalResolvedModel) ModelEvalMatchCandidate {
	return ModelEvalMatchCandidate{
		UUID:        m.UUID,
		DisplayName: m.DisplayName,
		APIName:     m.APIName,
		Source:      m.Source,
	}
}

// lookupEvalModelByUUID loads the canonical api_name (inference slug) and display_name for a model UUID.
func lookupEvalModelByUUID(ctx context.Context, client *godo.Client, uuid string) (*modelEvalResolvedModel, error) {
	model, _, err := client.GradientAI.GetModelByUUID(ctx, uuid)
	if err == nil && model != nil {
		apiName, displayName := catalogModelNames(model)
		return &modelEvalResolvedModel{
			UUID:        uuid,
			APIName:     apiName,
			DisplayName: displayName,
			Source:      "catalog",
		}, nil
	}

	customModel, _, err := client.GradientAI.GetCustomModel(ctx, uuid)
	if err != nil {
		return nil, fmt.Errorf("model %q not found in catalog or custom models: %w", uuid, err)
	}
	if customModel == nil {
		return nil, fmt.Errorf("model %q not found in catalog or custom models", uuid)
	}
	if customModel.Status == godo.CustomModelStatusDeleted {
		return nil, fmt.Errorf("model %q is deleted", uuid)
	}
	return &modelEvalResolvedModel{
		UUID:        uuid,
		APIName:     customModel.Name,
		DisplayName: customModel.Name,
		Source:      "custom",
	}, nil
}

// hydrateResolvedEvalModelsFromAPI re-fetches api_name values from the API so consent tokens cannot supply display names.
func hydrateResolvedEvalModelsFromAPI(ctx context.Context, client *godo.Client, resolved *modelEvalRunModels) (*modelEvalRunModels, error) {
	if resolved == nil {
		return nil, fmt.Errorf("no models to hydrate")
	}
	candidate, err := lookupEvalModelByUUID(ctx, client, resolved.Candidate.UUID)
	if err != nil {
		return nil, err
	}
	out := &modelEvalRunModels{Candidate: *candidate}
	if resolved.Judge != nil {
		judge, err := lookupEvalModelByUUID(ctx, client, resolved.Judge.UUID)
		if err != nil {
			return nil, err
		}
		out.Judge = judge
	}
	return out, nil
}

func isExactEvalModelUUID(uuid string) bool {
	return evalModelUUIDPattern.MatchString(uuid)
}

func buildEvalUnresolvedOutput(role, query, queryField string, matches []*EvalCatalogModel) *ModelEvalUnresolvedOutput {
	candidates := make([]ModelEvalMatchCandidate, 0, len(matches))
	for _, m := range matches {
		candidates = append(candidates, evalModelToMatchCandidate(m))
	}

	var msg string
	switch queryField {
	case "uuid":
		msg = fmt.Sprintf("no %s model with exact uuid %q; ask the user to provide the full uuid from the matches below — do not substitute a partial uuid", role, query)
		if len(candidates) == 0 {
			msg = fmt.Sprintf("no %s model with exact uuid %q and no models whose uuids contain that string", role, query)
		}
		if len(candidates) == 1 {
			msg = fmt.Sprintf("partial %s uuid %q — one model matched (%s / display %q, api %q) but the run requires the exact full uuid; ask the user to confirm the complete uuid", role, query, candidates[0].UUID, candidates[0].DisplayName, candidates[0].APIName)
		}
	default:
		msg = fmt.Sprintf("no %s model with exact name %q; ask the user to choose display_name or api_name from the matches below — do not substitute from this list", role, query)
		if len(candidates) == 0 {
			msg = fmt.Sprintf("no %s model with exact name %q and no models whose display or api names contain that string", role, query)
		}
		if len(candidates) == 1 {
			msg = fmt.Sprintf("no %s model with exact name %q; one similar model exists (display %q, api %q) — ask the user to confirm before proceeding", role, query, candidates[0].DisplayName, candidates[0].APIName)
		}
	}

	return &ModelEvalUnresolvedOutput{
		Message:                 msg,
		Role:                    role,
		Query:                   query,
		QueryField:              queryField,
		Matches:                 candidates,
		RequiresExactMatch:      true,
		DoNotSubstituteFromList: true,
	}
}

func buildModelEvalModelsResolveOutput(candidate, judge *ModelEvalUnresolvedOutput) *ModelEvalModelsResolveOutput {
	msg := "resolve candidate and judge models before creating the evaluation run"
	switch {
	case candidate != nil && judge != nil:
		msg = "candidate and judge model names or uuids are not exact matches; ask the user to pick exact identifiers from the match lists"
	case candidate != nil:
		msg = "candidate model name or uuid is not an exact match; ask the user to pick the exact candidate from the match list"
	case judge != nil:
		msg = "judge model name or uuid is not an exact match; ask the user to pick the exact judge from the match list"
	}
	return &ModelEvalModelsResolveOutput{
		Status:                modelEvalModelSelectionStatus,
		Message:               msg,
		Candidate:             candidate,
		Judge:                 judge,
		StopAndAskUser:        true,
		DoNotRetryUntilUserOK: true,
	}
}

func marshalModelEvalResolveResult(v any) (string, error) {
	jsonData, err := json.MarshalIndent(v, "", "  ")
	if err != nil {
		return "", fmt.Errorf("marshal error: %w", err)
	}
	return string(jsonData), nil
}

func modelEvalUserActionResult(v any) (*mcp.CallToolResult, error) {
	text, err := marshalModelEvalResolveResult(v)
	if err != nil {
		return nil, err
	}
	return mcp.NewToolResultText(text), nil
}
