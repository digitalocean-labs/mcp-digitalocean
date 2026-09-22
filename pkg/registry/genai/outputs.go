package genai

import (
	"github.com/digitalocean/godo"

	"mcp-digitalocean/pkg/registry/common"
)

// --- agent evaluation (evaluation_tools.go) ---------------------------------

// evaluationMetricList is the shape genai-list-evaluation-metrics has always
// returned: the metrics alongside how many came back. It was declared inside
// the handler; naming it lets one type drive both the declared schema and the
// emitted payload.
type evaluationMetricList struct {
	Metrics []*EvaluationMetric `json:"metrics"`
	Count   int                 `json:"count"`
}

// evaluationTestCaseList is the same pairing for
// genai-list-evaluation-test-cases.
type evaluationTestCaseList struct {
	TestCases []*EvaluationTestCase `json:"test_cases"`
	Count     int                   `json:"count"`
}

// evaluationDatasetCreated is the receipt genai-create-evaluation-dataset has
// always returned after the CSV upload: the new dataset's UUID plus the name
// and size the caller supplied.
type evaluationDatasetCreated struct {
	DatasetUUID string `json:"dataset_uuid"`
	Name        string `json:"name"`
	FileSize    int64  `json:"file_size"`
}

// evaluationTestCaseCreated is the receipt genai-create-evaluation-test-case
// has always returned.
type evaluationTestCaseCreated struct {
	TestCaseUUID string `json:"test_case_uuid"`
	Name         string `json:"name"`
	DatasetUUID  string `json:"dataset_uuid"`
}

// evaluationTestCaseUpdated is the receipt genai-update-evaluation-test-case
// has always returned: the test case and the version the update produced.
type evaluationTestCaseUpdated struct {
	TestCaseUUID string `json:"test_case_uuid"`
	Version      int    `json:"version"`
}

// evaluationRunStarted is the receipt genai-run-evaluation-test-case has
// always returned: one run UUID per evaluated deployment.
type evaluationRunStarted struct {
	EvaluationRunUUIDs []string `json:"evaluation_run_uuids"`
	Count              int      `json:"count"`
}

// evaluationMetricResultView is the per-metric shape
// genai-run-evaluation-workflow has always reported. It was a map built key by
// key, with each optional key written only when the underlying pointer was
// set; the pointer fields with omitempty reproduce that exactly, so a metric
// without a reasoning string still omits "reasoning" rather than emitting null.
type evaluationMetricResultView struct {
	MetricName       string                     `json:"metric_name"`
	NumberValue      *float64                   `json:"number_value,omitempty"`
	StringValue      *string                    `json:"string_value,omitempty"`
	Reasoning        *string                    `json:"reasoning,omitempty"`
	ErrorDescription *string                    `json:"error_description,omitempty"`
	MetricValueType  *EvaluationMetricValueType `json:"metric_value_type,omitempty"`
}

// newEvaluationMetricResultView projects a metric result onto the workflow view.
func newEvaluationMetricResultView(mr *EvaluationMetricResult) evaluationMetricResultView {
	return evaluationMetricResultView{
		MetricName:       mr.MetricName,
		NumberValue:      mr.NumberValue,
		StringValue:      mr.StringValue,
		Reasoning:        mr.Reasoning,
		ErrorDescription: mr.ErrorDescription,
		MetricValueType:  mr.MetricValueType,
	}
}

// evaluationWorkflowResult is the end-to-end report
// genai-run-evaluation-workflow has always returned: the resources it created
// along the way, the terminal run status, and the metric results it polled for.
type evaluationWorkflowResult struct {
	DatasetUUID       string                       `json:"dataset_uuid"`
	TestCaseUUID      string                       `json:"test_case_uuid"`
	EvaluationRunUUID string                       `json:"evaluation_run_uuid"`
	Status            string                       `json:"status"`
	MetricResults     []evaluationMetricResultView `json:"metric_results"`
	DurationSeconds   float64                      `json:"duration_seconds"`
	ErrorMessage      string                       `json:"error_message,omitempty"`
}

// --- model evaluation (model_evaluation_tools.go) ---------------------------

// modelEvalMetricList is the shape genai-model-eval-list-metrics has always
// returned. Like its siblings below it was declared inside the handler.
type modelEvalMetricList struct {
	Metrics []*godo.EvaluationMetric `json:"metrics"`
	Count   int                      `json:"count"`
}

// modelEvalDatasetList is the same pairing for genai-model-eval-list-datasets.
type modelEvalDatasetList struct {
	Datasets []*ModelEvalDatasetListItem `json:"datasets"`
	Count    int                         `json:"count"`
}

// modelEvalPresetList is the same pairing for genai-model-eval-list-presets.
type modelEvalPresetList struct {
	Presets []*godo.ModelEvaluationPreset `json:"presets"`
	Count   int                           `json:"count"`
}

// modelEvalRunList is the same pairing for genai-model-eval-list-runs.
type modelEvalRunList struct {
	Runs  []*godo.ModelEvaluationRunSummary `json:"runs"`
	Count int                               `json:"count"`
}

// modelEvalRunCreated is the receipt genai-model-eval-create-run has always
// returned: the new run's UUID and the name it was created under.
type modelEvalRunCreated struct {
	EvalRunUUID string `json:"eval_run_uuid"`
	Name        string `json:"name"`
}

// modelEvalPresetDeleted, modelEvalDatasetDeleted and
// modelEvalCustomMetricDeleted are the fixed {uuid, status} acknowledgements
// the three delete tools synthesise locally; the API itself answers with an
// empty body, so these echo back what was deleted.
type modelEvalPresetDeleted struct {
	EvalPresetUUID string `json:"eval_preset_uuid"`
	Status         string `json:"status"`
}

type modelEvalDatasetDeleted struct {
	DatasetUUID string `json:"dataset_uuid"`
	Status      string `json:"status"`
}

type modelEvalCustomMetricDeleted struct {
	MetricUUID string `json:"metric_uuid"`
	Status     string `json:"status"`
}

// modelEvalWorkflowResult is the end-to-end report
// genai-model-eval-run-workflow has always returned after polling the run to a
// terminal status.
type modelEvalWorkflowResult struct {
	EvalRunUUID     string                                `json:"eval_run_uuid"`
	Status          string                                `json:"status"`
	ResultSummary   *godo.ModelEvaluationRunResultSummary `json:"result_summary,omitempty"`
	DurationSeconds float64                               `json:"duration_seconds"`
	ErrorMessage    string                                `json:"error_message,omitempty"`
}

// --- simulation (simulation_tools.go) ---------------------------------------

// scenarioSetList, scenarioList, scenarioLibraryList, simulationRunList and
// simulationJourneyList are the {items, count} shapes the simulation list
// tools have always returned. Each was an anonymous struct declared inside its
// handler. scenarioList is shared by genai-simulation-list-scenarios and
// genai-simulation-list-scenario-library-scenarios, which return the same
// scenario shape from team-owned and platform-curated sets respectively.
type scenarioSetList struct {
	ScenarioSets []*godo.ScenarioSet `json:"scenario_sets"`
	Count        int                 `json:"count"`
}

type scenarioList struct {
	Scenarios []*godo.Scenario `json:"scenarios"`
	Count     int              `json:"count"`
}

type scenarioLibraryList struct {
	Scenarios []*godo.ScenarioLibraryEntry `json:"scenarios"`
	Count     int                          `json:"count"`
}

type simulationRunList struct {
	SimulationRuns []*godo.SimulationRun `json:"simulation_runs"`
	Count          int                   `json:"count"`
}

type simulationJourneyList struct {
	Journeys []*godo.SimulationJourney `json:"journeys"`
	Count    int                       `json:"count"`
}

// Output contracts for this package's tools; see the marketplace package for
// the convention.
//
// Payloads that are already JSON objects — the {items, count} list shapes, the
// workflow reports, and the godo response types that wrap a resource in their
// own key — are published with NewObjectOutput, so the text content keeps the
// wrapper it has always had instead of the wrapper moving into
// structuredContent alone. Payloads that are a bare resource take an envelope
// under the key the API itself uses for that resource: "preset", "run",
// "metric", "scenario_set", "simulation_run", "journey", "trajectory".
//
// Shared vars mark tools that answer with the same shape: modelEvalCustomMetricOut
// by custom-metric create and update, scenarioSetOut by scenario-set get,
// generate, update and create-from-library, simulationRunOut by simulation run
// create, update and cancel, and scenarioListOut by the two scenario list tools.
//
// genai-simulation-create-scenario-set stays text-only: it answers with a bare
// godo.ScenarioSet for inline scenarios but with a ScenarioSetCreateResult
// (the set plus the uploaded object's key, name and size) for a JSONL file, and
// one tool cannot declare two payload shapes without changing what one of the
// branches returns.
//
// genai-model-eval-create-run and genai-model-eval-run-workflow declare the
// schema of the run they create. Both also have interstitial branches that ask
// the caller to disambiguate a model name or to collect the end user's consent;
// those keep returning their prompt as text only, since they describe work not
// yet done rather than a created run.
var (
	evaluationMetricListOut      = common.NewObjectOutput[evaluationMetricList]()
	evaluationTestCaseListOut    = common.NewObjectOutput[evaluationTestCaseList]()
	evaluationDatasetCreatedOut  = common.NewObjectOutput[evaluationDatasetCreated]()
	evaluationTestCaseCreatedOut = common.NewObjectOutput[evaluationTestCaseCreated]()
	evaluationTestCaseUpdatedOut = common.NewObjectOutput[evaluationTestCaseUpdated]()
	evaluationRunStartedOut      = common.NewObjectOutput[evaluationRunStarted]()
	evaluationRunOut             = common.NewObjectOutput[GetEvaluationRunOutput]()
	evaluationWorkflowOut        = common.NewObjectOutput[evaluationWorkflowResult]()

	modelEvalMetricListOut          = common.NewObjectOutput[modelEvalMetricList]()
	modelEvalDatasetListOut         = common.NewObjectOutput[modelEvalDatasetList]()
	modelEvalPresetListOut          = common.NewObjectOutput[modelEvalPresetList]()
	modelEvalPresetOut              = common.NewOutput[*godo.ModelEvaluationPreset]("preset")
	modelEvalDatasetOut             = common.NewObjectOutput[*ModelEvalDatasetResult]()
	modelEvalRunCreatedOut          = common.NewObjectOutput[modelEvalRunCreated]()
	modelEvalRunListOut             = common.NewObjectOutput[modelEvalRunList]()
	modelEvalRunOut                 = common.NewObjectOutput[*godo.ModelEvaluationRunGetResponse]()
	modelEvalRunUpdatedOut          = common.NewOutput[*godo.ModelEvaluationRunSummary]("run")
	modelEvalResultsURLOut          = common.NewObjectOutput[*godo.ModelEvaluationRunResultsDownloadURLResponse]()
	modelEvalRunDeletedOut          = common.NewObjectOutput[*godo.ModelEvaluationRunDeleteResponse]()
	modelEvalRunCancelledOut        = common.NewObjectOutput[*godo.ModelEvaluationRunCancelResponse]()
	modelEvalPresetDeletedOut       = common.NewObjectOutput[modelEvalPresetDeleted]()
	modelEvalDatasetDeletedOut      = common.NewObjectOutput[modelEvalDatasetDeleted]()
	modelEvalCustomMetricOut        = common.NewOutput[*godo.EvaluationMetric]("metric")
	modelEvalCustomMetricDeletedOut = common.NewObjectOutput[modelEvalCustomMetricDeleted]()
	modelEvalWorkflowOut            = common.NewObjectOutput[modelEvalWorkflowResult]()

	scenarioSetListOut         = common.NewObjectOutput[scenarioSetList]()
	scenarioSetOut             = common.NewOutput[*godo.ScenarioSet]("scenario_set")
	scenarioSetDeletedOut      = common.NewObjectOutput[*godo.ScenarioSetDeleteResponse]()
	scenarioSetDownloadURLOut  = common.NewObjectOutput[*godo.ScenarioSetDownloadURLResponse]()
	scenarioListOut            = common.NewObjectOutput[scenarioList]()
	scenarioLibraryListOut     = common.NewObjectOutput[scenarioLibraryList]()
	simulationRunOut           = common.NewOutput[*godo.SimulationRun]("simulation_run")
	simulationRunListOut       = common.NewObjectOutput[simulationRunList]()
	simulationRunGetOut        = common.NewObjectOutput[*godo.SimulationRunGetResponse]()
	simulationRunDeletedOut    = common.NewObjectOutput[*godo.SimulationRunDeleteResponse]()
	simulationJourneyListOut   = common.NewObjectOutput[simulationJourneyList]()
	simulationJourneyOut       = common.NewOutput[*godo.SimulationJourney]("journey")
	simulationTrajectoryOut    = common.NewOutput[*godo.SimulationTrajectory]("trajectory")
	simulationTrajectoryURLOut = common.NewObjectOutput[*godo.SimulationJourneyTrajectoryURLResponse]()
)
