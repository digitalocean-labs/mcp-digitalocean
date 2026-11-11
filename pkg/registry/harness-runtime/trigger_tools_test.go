package harnessruntime

import (
	"context"
	"encoding/json"
	"net/http"
	"net/url"
	"testing"

	"github.com/stretchr/testify/require"
)

const (
	testTriggerID   = "049fb39c-14d9-7080-933e-b9b90e25acda"
	testExecutionID = "059fb39c-14d9-7080-933e-b9b90e25acdb"
)

func triggerJSON(id, name, kind, status string) string {
	return `{"trigger_id":"` + id + `","name":"` + name + `","kind":"` + kind + `","status":"` + status + `","config_id":"` + testConfigID + `","session_mode":"fresh","created_at":"2026-08-01T12:00:00Z","updated_at":"2026-08-01T12:00:00Z"}`
}

func executionJSON(id, triggerID, status string) string {
	return `{"execution_id":"` + id + `","trigger_id":"` + triggerID + `","session_id":"` + testSessionID + `","status":"` + status + `","created_at":"2026-08-01T12:00:00Z"}`
}

// ---- ListTriggers ----

func TestListTriggers(t *testing.T) {
	var gotPath string
	var gotQuery url.Values
	tool := newTestTool(t, func(w http.ResponseWriter, r *http.Request) {
		gotPath = r.URL.Path
		gotQuery = r.URL.Query()
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"triggers":[` + triggerJSON(testTriggerID, "nightly-eval", "cron", "active") + `],"next_page_token":"tok2"}`))
	})

	res, err := tool.listTriggers(context.Background(), request(map[string]any{
		"PageSize":  float64(10),
		"PageToken": "tok1",
		"ConfigID":  testConfigID,
	}))
	require.NoError(t, err)
	require.False(t, res.IsError)
	require.Equal(t, "/v2/harness-runtime/triggers", gotPath)
	require.Equal(t, "10", gotQuery.Get("page_size"))
	require.Equal(t, "tok1", gotQuery.Get("page_token"))
	require.Equal(t, testConfigID, gotQuery.Get("config_id"))
	text := callText(t, res)
	require.Contains(t, text, "nightly-eval")
	require.Contains(t, text, testTriggerID)
	require.Contains(t, text, "tok2")
}

func TestListTriggersNoArgsOmitsQuery(t *testing.T) {
	var gotRawQuery string
	tool := newTestTool(t, func(w http.ResponseWriter, r *http.Request) {
		gotRawQuery = r.URL.RawQuery
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{}`))
	})

	res, err := tool.listTriggers(context.Background(), request(nil))
	require.NoError(t, err)
	require.False(t, res.IsError)
	require.Empty(t, gotRawQuery)
	require.Regexp(t, `"triggers":\s*\[\]`, callText(t, res))
}

func TestListTriggersValidation(t *testing.T) {
	tool := newTestTool(t, func(http.ResponseWriter, *http.Request) {
		t.Fatal("API must not be called for invalid input")
	})

	for name, args := range map[string]map[string]any{
		"page size too small": {"PageSize": float64(0)},
		"page size too large": {"PageSize": float64(201)},
		"page size fraction":  {"PageSize": 1.9},
	} {
		t.Run(name, func(t *testing.T) {
			res, err := tool.listTriggers(context.Background(), request(args))
			require.NoError(t, err)
			require.True(t, res.IsError)
		})
	}
}

func TestListTriggersAPIError(t *testing.T) {
	tool := newTestTool(t, func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusUnauthorized)
		_, _ = w.Write([]byte(`{"id":"unauthorized","message":"Unable to authenticate you"}`))
	})

	res, err := tool.listTriggers(context.Background(), request(nil))
	require.NoError(t, err)
	require.True(t, res.IsError)
	require.Contains(t, callText(t, res), "Failed to list triggers")
}

// ---- CreateTrigger ----

func TestCreateTriggerWebhook(t *testing.T) {
	var gotPath, gotMethod, gotContentType string
	var gotBody map[string]any

	tool := newTestTool(t, func(w http.ResponseWriter, r *http.Request) {
		gotPath = r.URL.Path
		gotMethod = r.Method
		gotContentType = r.Header.Get("Content-Type")
		_ = json.NewDecoder(r.Body).Decode(&gotBody)
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"trigger":` + triggerJSON(testTriggerID, "my-webhook", "webhook", "active") + `,"webhook_secret":"secret123"}`))
	})

	res, err := tool.createTrigger(context.Background(), request(map[string]any{
		"Name":            "my-webhook",
		"Kind":            "webhook",
		"SessionMode":     "fresh",
		"SessionTemplate": "size: mars-2vcpu-4gb\nagent: codex\npermissions:\n  default: allow\n",
		"PromptTemplate":  "Run nightly evaluation",
	}))
	require.NoError(t, err)
	require.False(t, res.IsError)
	require.Equal(t, "/v2/harness-runtime/triggers", gotPath)
	require.Equal(t, http.MethodPost, gotMethod)
	require.Contains(t, gotContentType, "application/json")
	require.Nil(t, gotBody["config_id"])
	require.Equal(t, "my-webhook", gotBody["name"])
	require.Equal(t, "webhook", gotBody["kind"])
	require.Equal(t, "fresh", gotBody["session_mode"])
	require.Equal(t, "Run nightly evaluation", gotBody["prompt_template"])
	require.NotNil(t, gotBody["webhook"])

	output, ok := gotBody["output"].(map[string]any)
	require.True(t, ok)
	require.Equal(t, "none", output["mode"])

	text := callText(t, res)
	require.Contains(t, text, testTriggerID)
	require.Contains(t, text, "my-webhook")
	require.Contains(t, text, "secret123")
}

func TestCreateTriggerCron(t *testing.T) {
	var gotBody map[string]any

	tool := newTestTool(t, func(w http.ResponseWriter, r *http.Request) {
		_ = json.NewDecoder(r.Body).Decode(&gotBody)
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"trigger":` + triggerJSON(testTriggerID, "daily-cron", "cron", "active") + `}`))
	})

	res, err := tool.createTrigger(context.Background(), request(map[string]any{
		"Name":            "daily-cron",
		"Kind":            "cron",
		"SessionMode":     "fresh",
		"SessionTemplate": "size: mars-2vcpu-4gb\nagent: codex\npermissions:\n  default: allow\n",
		"PromptTemplate":  "Run daily evaluation",
		"CronExpr":        "0 0 * * *",
		"Timezone":        "America/New_York",
	}))
	require.NoError(t, err)
	require.False(t, res.IsError)
	require.Equal(t, "cron", gotBody["kind"])

	cronObj, ok := gotBody["cron"].(map[string]any)
	require.True(t, ok)
	require.Equal(t, "0 0 * * *", cronObj["cron_expr"])
	require.Equal(t, "America/New_York", cronObj["timezone"])
}

func TestCreateTriggerCronDefaultTimezone(t *testing.T) {
	var gotBody map[string]any

	tool := newTestTool(t, func(w http.ResponseWriter, r *http.Request) {
		_ = json.NewDecoder(r.Body).Decode(&gotBody)
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"trigger":` + triggerJSON(testTriggerID, "daily-utc", "cron", "active") + `}`))
	})

	res, err := tool.createTrigger(context.Background(), request(map[string]any{
		"Name":            "daily-utc",
		"Kind":            "cron",
		"SessionMode":     "fresh",
		"SessionTemplate": "size: mars-2vcpu-4gb\nagent: codex\npermissions:\n  default: allow\n",
		"PromptTemplate":  "Run daily evaluation",
		"CronExpr":        "0 0 * * *",
	}))
	require.NoError(t, err)
	require.False(t, res.IsError)

	cronObj, ok := gotBody["cron"].(map[string]any)
	require.True(t, ok)
	require.Equal(t, "UTC", cronObj["timezone"])
}

func TestCreateTriggerReuseMode(t *testing.T) {
	var gotBody map[string]any
	boundSessionID := "079fb39c-14d9-7080-933e-b9b90e25acdc"

	tool := newTestTool(t, func(w http.ResponseWriter, r *http.Request) {
		_ = json.NewDecoder(r.Body).Decode(&gotBody)
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"trigger":` + triggerJSON(testTriggerID, "reuse-trigger", "webhook", "active") + `}`))
	})

	res, err := tool.createTrigger(context.Background(), request(map[string]any{
		"Name":           "reuse-trigger",
		"Kind":           "webhook",
		"SessionMode":    "reuse",
		"BoundSessionID": boundSessionID,
		"PromptTemplate": "Run reuse evaluation",
		"OutputMode":     "email",
		"Email":          "test@example.com",
	}))
	require.NoError(t, err)
	require.False(t, res.IsError)
	require.Equal(t, "reuse", gotBody["session_mode"])
	require.Equal(t, boundSessionID, gotBody["bound_session_id"])

	output, ok := gotBody["output"].(map[string]any)
	require.True(t, ok)
	require.Equal(t, "email", output["mode"])
	require.Equal(t, "test@example.com", output["email"])
}

func TestCreateTriggerWithMaxConcurrency(t *testing.T) {
	var gotBody map[string]any

	tool := newTestTool(t, func(w http.ResponseWriter, r *http.Request) {
		_ = json.NewDecoder(r.Body).Decode(&gotBody)
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"trigger":` + triggerJSON(testTriggerID, "conc-trigger", "webhook", "active") + `}`))
	})

	res, err := tool.createTrigger(context.Background(), request(map[string]any{
		"Name":            "conc-trigger",
		"Kind":            "webhook",
		"SessionMode":     "fresh",
		"SessionTemplate": "size: mars-2vcpu-4gb\nagent: codex\npermissions:\n  default: allow\n",
		"PromptTemplate":  "Run concurrent evaluation",
		"MaxConcurrency":  float64(5),
	}))
	require.NoError(t, err)
	require.False(t, res.IsError)
	require.Equal(t, float64(5), gotBody["max_concurrency"])
}

func TestCreateTriggerValidation(t *testing.T) {
	tool := newTestTool(t, func(http.ResponseWriter, *http.Request) {
		t.Fatal("API must not be called for invalid input")
	})

	for name, args := range map[string]map[string]any{
		"missing name":                {"Kind": "cron", "SessionMode": "fresh", "PromptTemplate": "test", "SessionTemplate": "size: s\n", "CronExpr": "0 0 * * *"},
		"missing kind":                {"Name": "test", "SessionMode": "fresh", "PromptTemplate": "test", "SessionTemplate": "size: s\n"},
		"invalid kind":                {"Name": "test", "Kind": "manual", "SessionMode": "fresh", "PromptTemplate": "test", "SessionTemplate": "size: s\n"},
		"missing session mode":        {"Name": "test", "Kind": "webhook", "PromptTemplate": "test"},
		"invalid session mode":        {"Name": "test", "Kind": "webhook", "PromptTemplate": "test", "SessionMode": "something"},
		"missing prompt template":     {"Name": "test", "Kind": "webhook", "SessionMode": "fresh", "SessionTemplate": "size: s\n"},
		"fresh missing template":      {"Name": "test", "Kind": "webhook", "SessionMode": "fresh", "PromptTemplate": "test"},
		"reuse missing bound session": {"Name": "test", "Kind": "webhook", "SessionMode": "reuse", "PromptTemplate": "test"},
		"reuse invalid bound session": {"Name": "test", "Kind": "webhook", "SessionMode": "reuse", "PromptTemplate": "test", "BoundSessionID": "not-uuid"},
		"cron missing cron expr":      {"Name": "test", "Kind": "cron", "SessionMode": "fresh", "PromptTemplate": "test", "SessionTemplate": "size: s\n"},
		"invalid output mode":         {"Name": "test", "Kind": "webhook", "SessionMode": "fresh", "PromptTemplate": "test", "SessionTemplate": "size: s\n", "OutputMode": "invalid"},
		"email missing address":       {"Name": "test", "Kind": "webhook", "SessionMode": "fresh", "PromptTemplate": "test", "SessionTemplate": "size: s\n", "OutputMode": "email"},
		"slack missing webhook url":   {"Name": "test", "Kind": "webhook", "SessionMode": "fresh", "PromptTemplate": "test", "SessionTemplate": "size: s\n", "OutputMode": "slack"},
		"missing all":                 {},
	} {
		t.Run(name, func(t *testing.T) {
			res, err := tool.createTrigger(context.Background(), request(args))
			require.NoError(t, err)
			require.True(t, res.IsError, "expected error for %s, got: %s", name, callText(t, res))
		})
	}
}

func TestCreateTriggerAPIError(t *testing.T) {
	tool := newTestTool(t, func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusConflict)
		_, _ = w.Write([]byte(`{"id":"conflict","message":"trigger name already exists"}`))
	})

	res, err := tool.createTrigger(context.Background(), request(map[string]any{
		"Name":            "existing-trigger",
		"Kind":            "webhook",
		"SessionMode":     "fresh",
		"PromptTemplate":  "Run evaluation",
		"SessionTemplate": "size: mars-2vcpu-4gb\nagent: codex\npermissions:\n  default: allow\n",
	}))
	require.NoError(t, err)
	require.True(t, res.IsError)
	require.Contains(t, callText(t, res), "Failed to create trigger")
}

// ---- GetTrigger ----

func TestGetTrigger(t *testing.T) {
	var gotPath string
	tool := newTestTool(t, func(w http.ResponseWriter, r *http.Request) {
		gotPath = r.URL.Path
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"trigger":` + triggerJSON(testTriggerID, "my-trigger", "webhook", "active") + `}`))
	})

	res, err := tool.getTrigger(context.Background(), request(map[string]any{"TriggerID": testTriggerID}))
	require.NoError(t, err)
	require.False(t, res.IsError)
	require.Equal(t, "/v2/harness-runtime/triggers/"+testTriggerID, gotPath)
	text := callText(t, res)
	require.Contains(t, text, testTriggerID)
	require.Contains(t, text, "my-trigger")
}

func TestGetTriggerValidation(t *testing.T) {
	tool := newTestTool(t, func(http.ResponseWriter, *http.Request) {
		t.Fatal("API must not be called for invalid input")
	})

	for name, args := range map[string]map[string]any{
		"missing":   {},
		"not uuid":  {"TriggerID": "my-trigger"},
		"traversal": {"TriggerID": "../sessions"},
	} {
		t.Run(name, func(t *testing.T) {
			res, err := tool.getTrigger(context.Background(), request(args))
			require.NoError(t, err)
			require.True(t, res.IsError)
		})
	}
}

func TestGetTriggerNotFound(t *testing.T) {
	tool := newTestTool(t, func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusNotFound)
		_, _ = w.Write([]byte(`{"id":"not_found","message":"trigger not found"}`))
	})

	res, err := tool.getTrigger(context.Background(), request(map[string]any{"TriggerID": testTriggerID}))
	require.NoError(t, err)
	require.True(t, res.IsError)
	require.Contains(t, callText(t, res), "Failed to get trigger")
}

// ---- UpdateTrigger ----

func TestUpdateTrigger(t *testing.T) {
	var gotPath, gotMethod string
	var gotBody map[string]any

	tool := newTestTool(t, func(w http.ResponseWriter, r *http.Request) {
		gotPath = r.URL.Path
		gotMethod = r.Method
		_ = json.NewDecoder(r.Body).Decode(&gotBody)
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"trigger":` + triggerJSON(testTriggerID, "updated-trigger", "cron", "active") + `}`))
	})

	res, err := tool.updateTrigger(context.Background(), request(map[string]any{
		"TriggerID": testTriggerID,
		"Name":      "updated-trigger",
		"CronExpr":  "0 */12 * * *",
		"Timezone":  "UTC",
	}))
	require.NoError(t, err)
	require.False(t, res.IsError)
	require.Equal(t, "/v2/harness-runtime/triggers/"+testTriggerID, gotPath)
	require.Equal(t, http.MethodPatch, gotMethod)
	require.Equal(t, "updated-trigger", gotBody["name"])

	cronObj, ok := gotBody["cron"].(map[string]any)
	require.True(t, ok)
	require.Equal(t, "0 */12 * * *", cronObj["cron_expr"])

	text := callText(t, res)
	require.Contains(t, text, "updated-trigger")
}

func TestUpdateTriggerWithPromptTemplate(t *testing.T) {
	var gotBody map[string]any

	tool := newTestTool(t, func(w http.ResponseWriter, r *http.Request) {
		_ = json.NewDecoder(r.Body).Decode(&gotBody)
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"trigger":` + triggerJSON(testTriggerID, "my-trigger", "cron", "active") + `}`))
	})

	res, err := tool.updateTrigger(context.Background(), request(map[string]any{
		"TriggerID":      testTriggerID,
		"PromptTemplate": "Updated prompt template",
	}))
	require.NoError(t, err)
	require.False(t, res.IsError)
	require.Equal(t, "Updated prompt template", gotBody["prompt_template"])
}

func TestUpdateTriggerOutput(t *testing.T) {
	var gotBody map[string]any

	tool := newTestTool(t, func(w http.ResponseWriter, r *http.Request) {
		_ = json.NewDecoder(r.Body).Decode(&gotBody)
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"trigger":` + triggerJSON(testTriggerID, "my-trigger", "webhook", "active") + `}`))
	})

	res, err := tool.updateTrigger(context.Background(), request(map[string]any{
		"TriggerID":       testTriggerID,
		"OutputMode":      "slack",
		"SlackWebhookURL": "https://hooks.slack.com/services/T00/B00/xxx",
	}))
	require.NoError(t, err)
	require.False(t, res.IsError)

	output, ok := gotBody["output"].(map[string]any)
	require.True(t, ok)
	require.Equal(t, "slack", output["mode"])
	slack, ok := output["slack"].(map[string]any)
	require.True(t, ok)
	require.Equal(t, "https://hooks.slack.com/services/T00/B00/xxx", slack["webhook_url"])
}

func TestUpdateTriggerValidation(t *testing.T) {
	tool := newTestTool(t, func(http.ResponseWriter, *http.Request) {
		t.Fatal("API must not be called for invalid input")
	})

	for name, args := range map[string]map[string]any{
		"missing trigger id":    {"Name": "updated"},
		"invalid trigger id":    {"TriggerID": "not-a-uuid", "Name": "updated"},
		"no updatable fields":   {"TriggerID": testTriggerID},
		"invalid bound session": {"TriggerID": testTriggerID, "BoundSessionID": "not-a-uuid"},
	} {
		t.Run(name, func(t *testing.T) {
			res, err := tool.updateTrigger(context.Background(), request(args))
			require.NoError(t, err)
			require.True(t, res.IsError)
		})
	}
}

func TestUpdateTriggerAPIError(t *testing.T) {
	tool := newTestTool(t, func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusNotFound)
		_, _ = w.Write([]byte(`{"id":"not_found","message":"trigger not found"}`))
	})

	res, err := tool.updateTrigger(context.Background(), request(map[string]any{
		"TriggerID": testTriggerID,
		"Name":      "updated-trigger",
	}))
	require.NoError(t, err)
	require.True(t, res.IsError)
	require.Contains(t, callText(t, res), "Failed to update trigger")
}

// ---- DeleteTrigger ----

func TestDeleteTrigger(t *testing.T) {
	var gotPath, gotMethod string
	tool := newTestTool(t, func(w http.ResponseWriter, r *http.Request) {
		gotPath = r.URL.Path
		gotMethod = r.Method
		w.WriteHeader(http.StatusNoContent)
	})

	res, err := tool.deleteTrigger(context.Background(), request(map[string]any{"TriggerID": testTriggerID}))
	require.NoError(t, err)
	require.False(t, res.IsError)
	require.Equal(t, "/v2/harness-runtime/triggers/"+testTriggerID, gotPath)
	require.Equal(t, http.MethodDelete, gotMethod)
	require.Contains(t, callText(t, res), testTriggerID)
	require.Contains(t, callText(t, res), "deleted")
}

func TestDeleteTriggerValidation(t *testing.T) {
	tool := newTestTool(t, func(http.ResponseWriter, *http.Request) {
		t.Fatal("API must not be called for invalid input")
	})

	for name, args := range map[string]map[string]any{
		"missing":  {},
		"not uuid": {"TriggerID": "my-trigger"},
	} {
		t.Run(name, func(t *testing.T) {
			res, err := tool.deleteTrigger(context.Background(), request(args))
			require.NoError(t, err)
			require.True(t, res.IsError)
		})
	}
}

func TestDeleteTriggerAPIError(t *testing.T) {
	tool := newTestTool(t, func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusNotFound)
		_, _ = w.Write([]byte(`{"id":"not_found","message":"trigger not found"}`))
	})

	res, err := tool.deleteTrigger(context.Background(), request(map[string]any{"TriggerID": testTriggerID}))
	require.NoError(t, err)
	require.True(t, res.IsError)
	require.Contains(t, callText(t, res), "Failed to delete trigger")
}

// ---- PauseTrigger ----

func TestPauseTrigger(t *testing.T) {
	var gotPath, gotMethod string
	var gotBody map[string]any
	tool := newTestTool(t, func(w http.ResponseWriter, r *http.Request) {
		gotPath = r.URL.Path
		gotMethod = r.Method
		_ = json.NewDecoder(r.Body).Decode(&gotBody)
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"trigger":` + triggerJSON(testTriggerID, "my-trigger", "cron", "paused") + `}`))
	})

	res, err := tool.pauseTrigger(context.Background(), request(map[string]any{"TriggerID": testTriggerID}))
	require.NoError(t, err)
	require.False(t, res.IsError)
	require.Equal(t, "/v2/harness-runtime/triggers/"+testTriggerID, gotPath)
	require.Equal(t, http.MethodPatch, gotMethod)
	require.Equal(t, "paused", gotBody["status"])
	text := callText(t, res)
	require.Contains(t, text, testTriggerID)
	require.Contains(t, text, "paused")
}

func TestPauseTriggerValidation(t *testing.T) {
	tool := newTestTool(t, func(http.ResponseWriter, *http.Request) {
		t.Fatal("API must not be called for invalid input")
	})

	for name, args := range map[string]map[string]any{
		"missing":  {},
		"not uuid": {"TriggerID": "not-valid"},
	} {
		t.Run(name, func(t *testing.T) {
			res, err := tool.pauseTrigger(context.Background(), request(args))
			require.NoError(t, err)
			require.True(t, res.IsError)
		})
	}
}

func TestPauseTriggerAPIError(t *testing.T) {
	tool := newTestTool(t, func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusConflict)
		_, _ = w.Write([]byte(`{"id":"conflict","message":"trigger already paused"}`))
	})

	res, err := tool.pauseTrigger(context.Background(), request(map[string]any{"TriggerID": testTriggerID}))
	require.NoError(t, err)
	require.True(t, res.IsError)
	require.Contains(t, callText(t, res), "Failed to pause trigger")
}

// ---- ResumeTrigger ----

func TestResumeTrigger(t *testing.T) {
	var gotPath, gotMethod string
	var gotBody map[string]any
	tool := newTestTool(t, func(w http.ResponseWriter, r *http.Request) {
		gotPath = r.URL.Path
		gotMethod = r.Method
		_ = json.NewDecoder(r.Body).Decode(&gotBody)
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"trigger":` + triggerJSON(testTriggerID, "my-trigger", "cron", "active") + `}`))
	})

	res, err := tool.resumeTrigger(context.Background(), request(map[string]any{"TriggerID": testTriggerID}))
	require.NoError(t, err)
	require.False(t, res.IsError)
	require.Equal(t, "/v2/harness-runtime/triggers/"+testTriggerID, gotPath)
	require.Equal(t, http.MethodPatch, gotMethod)
	require.Equal(t, "active", gotBody["status"])
	text := callText(t, res)
	require.Contains(t, text, testTriggerID)
	require.Contains(t, text, "active")
}

func TestResumeTriggerValidation(t *testing.T) {
	tool := newTestTool(t, func(http.ResponseWriter, *http.Request) {
		t.Fatal("API must not be called for invalid input")
	})

	for name, args := range map[string]map[string]any{
		"missing":  {},
		"not uuid": {"TriggerID": "not-valid"},
	} {
		t.Run(name, func(t *testing.T) {
			res, err := tool.resumeTrigger(context.Background(), request(args))
			require.NoError(t, err)
			require.True(t, res.IsError)
		})
	}
}

func TestResumeTriggerAPIError(t *testing.T) {
	tool := newTestTool(t, func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusConflict)
		_, _ = w.Write([]byte(`{"id":"conflict","message":"trigger is not paused"}`))
	})

	res, err := tool.resumeTrigger(context.Background(), request(map[string]any{"TriggerID": testTriggerID}))
	require.NoError(t, err)
	require.True(t, res.IsError)
	require.Contains(t, callText(t, res), "Failed to resume trigger")
}

// ---- RotateTriggerSecret ----

func TestRotateTriggerSecret(t *testing.T) {
	var gotPath, gotMethod string
	tool := newTestTool(t, func(w http.ResponseWriter, r *http.Request) {
		gotPath = r.URL.Path
		gotMethod = r.Method
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"webhook_secret":"whsec_new_secret_value_123"}`))
	})

	res, err := tool.rotateTriggerSecret(context.Background(), request(map[string]any{"TriggerID": testTriggerID}))
	require.NoError(t, err)
	require.False(t, res.IsError)
	require.Equal(t, "/v2/harness-runtime/triggers/"+testTriggerID+"/rotate-secret", gotPath)
	require.Equal(t, http.MethodPost, gotMethod)
	text := callText(t, res)
	require.Contains(t, text, "whsec_new_secret_value_123")
}

func TestRotateTriggerSecretValidation(t *testing.T) {
	tool := newTestTool(t, func(http.ResponseWriter, *http.Request) {
		t.Fatal("API must not be called for invalid input")
	})

	for name, args := range map[string]map[string]any{
		"missing":  {},
		"not uuid": {"TriggerID": "not-valid"},
	} {
		t.Run(name, func(t *testing.T) {
			res, err := tool.rotateTriggerSecret(context.Background(), request(args))
			require.NoError(t, err)
			require.True(t, res.IsError)
		})
	}
}

func TestRotateTriggerSecretAPIError(t *testing.T) {
	tool := newTestTool(t, func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusNotFound)
		_, _ = w.Write([]byte(`{"id":"not_found","message":"trigger not found"}`))
	})

	res, err := tool.rotateTriggerSecret(context.Background(), request(map[string]any{"TriggerID": testTriggerID}))
	require.NoError(t, err)
	require.True(t, res.IsError)
	require.Contains(t, callText(t, res), "Failed to rotate trigger secret")
}

// ---- ListTriggerExecutions ----

func TestListTriggerExecutions(t *testing.T) {
	var gotPath string
	var gotQuery url.Values
	tool := newTestTool(t, func(w http.ResponseWriter, r *http.Request) {
		gotPath = r.URL.Path
		gotQuery = r.URL.Query()
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"executions":[` + executionJSON(testExecutionID, testTriggerID, "completed") + `],"next_page_token":"tok2"}`))
	})

	res, err := tool.listTriggerExecutions(context.Background(), request(map[string]any{
		"TriggerID": testTriggerID,
		"PageSize":  float64(5),
		"PageToken": "tok1",
	}))
	require.NoError(t, err)
	require.False(t, res.IsError)
	require.Equal(t, "/v2/harness-runtime/triggers/"+testTriggerID+"/executions", gotPath)
	require.Equal(t, "5", gotQuery.Get("page_size"))
	require.Equal(t, "tok1", gotQuery.Get("page_token"))
	text := callText(t, res)
	require.Contains(t, text, testExecutionID)
	require.Contains(t, text, "tok2")
}

func TestListTriggerExecutionsNoQueryParams(t *testing.T) {
	var gotRawQuery string
	tool := newTestTool(t, func(w http.ResponseWriter, r *http.Request) {
		gotRawQuery = r.URL.RawQuery
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{}`))
	})

	res, err := tool.listTriggerExecutions(context.Background(), request(map[string]any{
		"TriggerID": testTriggerID,
	}))
	require.NoError(t, err)
	require.False(t, res.IsError)
	require.Empty(t, gotRawQuery)
	require.Regexp(t, `"executions":\s*\[\]`, callText(t, res))
}

func TestListTriggerExecutionsValidation(t *testing.T) {
	tool := newTestTool(t, func(http.ResponseWriter, *http.Request) {
		t.Fatal("API must not be called for invalid input")
	})

	for name, args := range map[string]map[string]any{
		"missing trigger id":  {},
		"not uuid":            {"TriggerID": "my-trigger"},
		"page size too small": {"TriggerID": testTriggerID, "PageSize": float64(0)},
		"page size too large": {"TriggerID": testTriggerID, "PageSize": float64(201)},
		"page size fraction":  {"TriggerID": testTriggerID, "PageSize": 1.7},
	} {
		t.Run(name, func(t *testing.T) {
			res, err := tool.listTriggerExecutions(context.Background(), request(args))
			require.NoError(t, err)
			require.True(t, res.IsError)
		})
	}
}

func TestListTriggerExecutionsAPIError(t *testing.T) {
	tool := newTestTool(t, func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusNotFound)
		_, _ = w.Write([]byte(`{"id":"not_found","message":"trigger not found"}`))
	})

	res, err := tool.listTriggerExecutions(context.Background(), request(map[string]any{"TriggerID": testTriggerID}))
	require.NoError(t, err)
	require.True(t, res.IsError)
	require.Contains(t, callText(t, res), "Failed to list trigger executions")
}

// ---- GetTriggerExecution ----

func TestGetTriggerExecution(t *testing.T) {
	var gotPath string
	tool := newTestTool(t, func(w http.ResponseWriter, r *http.Request) {
		gotPath = r.URL.Path
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"execution":` + executionJSON(testExecutionID, testTriggerID, "running") + `}`))
	})

	res, err := tool.getTriggerExecution(context.Background(), request(map[string]any{
		"TriggerID":   testTriggerID,
		"ExecutionID": testExecutionID,
	}))
	require.NoError(t, err)
	require.False(t, res.IsError)
	require.Equal(t, "/v2/harness-runtime/triggers/"+testTriggerID+"/executions/"+testExecutionID, gotPath)
	text := callText(t, res)
	require.Contains(t, text, testExecutionID)
	require.Contains(t, text, "running")
}

func TestGetTriggerExecutionValidation(t *testing.T) {
	tool := newTestTool(t, func(http.ResponseWriter, *http.Request) {
		t.Fatal("API must not be called for invalid input")
	})

	for name, args := range map[string]map[string]any{
		"missing both":         {},
		"missing trigger id":   {"ExecutionID": testExecutionID},
		"missing execution id": {"TriggerID": testTriggerID},
		"invalid trigger id":   {"TriggerID": "not-uuid", "ExecutionID": testExecutionID},
		"invalid execution id": {"TriggerID": testTriggerID, "ExecutionID": "not-uuid"},
	} {
		t.Run(name, func(t *testing.T) {
			res, err := tool.getTriggerExecution(context.Background(), request(args))
			require.NoError(t, err)
			require.True(t, res.IsError)
		})
	}
}

func TestGetTriggerExecutionNotFound(t *testing.T) {
	tool := newTestTool(t, func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusNotFound)
		_, _ = w.Write([]byte(`{"id":"not_found","message":"execution not found"}`))
	})

	res, err := tool.getTriggerExecution(context.Background(), request(map[string]any{
		"TriggerID":   testTriggerID,
		"ExecutionID": testExecutionID,
	}))
	require.NoError(t, err)
	require.True(t, res.IsError)
	require.Contains(t, callText(t, res), "Failed to get trigger execution")
}

// ---- CancelExecution ----

func TestCancelExecution(t *testing.T) {
	var gotPath, gotMethod string
	tool := newTestTool(t, func(w http.ResponseWriter, r *http.Request) {
		gotPath = r.URL.Path
		gotMethod = r.Method
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"execution":` + executionJSON(testExecutionID, testTriggerID, "cancelled") + `}`))
	})

	res, err := tool.cancelExecution(context.Background(), request(map[string]any{
		"TriggerID":   testTriggerID,
		"ExecutionID": testExecutionID,
	}))
	require.NoError(t, err)
	require.False(t, res.IsError)
	require.Equal(t, "/v2/harness-runtime/triggers/"+testTriggerID+"/executions/"+testExecutionID+"/cancel", gotPath)
	require.Equal(t, http.MethodPost, gotMethod)
	text := callText(t, res)
	require.Contains(t, text, testExecutionID)
	require.Contains(t, text, "cancelled")
}

func TestCancelExecutionValidation(t *testing.T) {
	tool := newTestTool(t, func(http.ResponseWriter, *http.Request) {
		t.Fatal("API must not be called for invalid input")
	})

	for name, args := range map[string]map[string]any{
		"missing both":         {},
		"missing trigger id":   {"ExecutionID": testExecutionID},
		"missing execution id": {"TriggerID": testTriggerID},
		"invalid trigger id":   {"TriggerID": "not-uuid", "ExecutionID": testExecutionID},
		"invalid execution id": {"TriggerID": testTriggerID, "ExecutionID": "not-uuid"},
	} {
		t.Run(name, func(t *testing.T) {
			res, err := tool.cancelExecution(context.Background(), request(args))
			require.NoError(t, err)
			require.True(t, res.IsError)
		})
	}
}

func TestCancelExecutionAPIError(t *testing.T) {
	tool := newTestTool(t, func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusConflict)
		_, _ = w.Write([]byte(`{"id":"conflict","message":"execution is not running"}`))
	})

	res, err := tool.cancelExecution(context.Background(), request(map[string]any{
		"TriggerID":   testTriggerID,
		"ExecutionID": testExecutionID,
	}))
	require.NoError(t, err)
	require.True(t, res.IsError)
	require.Contains(t, callText(t, res), "Failed to cancel execution")
}

// ---- GetTriggerBySession ----

func TestGetTriggerBySession(t *testing.T) {
	var gotPath string
	tool := newTestTool(t, func(w http.ResponseWriter, r *http.Request) {
		gotPath = r.URL.Path
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"trigger":` + triggerJSON(testTriggerID, "session-trigger", "webhook", "active") + `}`))
	})

	res, err := tool.getTriggerBySession(context.Background(), request(map[string]any{"SessionID": testSessionID}))
	require.NoError(t, err)
	require.False(t, res.IsError)
	require.Equal(t, "/v2/harness-runtime/triggers/by-session/"+testSessionID, gotPath)
	text := callText(t, res)
	require.Contains(t, text, testTriggerID)
	require.Contains(t, text, "session-trigger")
}

func TestGetTriggerBySessionValidation(t *testing.T) {
	tool := newTestTool(t, func(http.ResponseWriter, *http.Request) {
		t.Fatal("API must not be called for invalid input")
	})

	for name, args := range map[string]map[string]any{
		"missing":   {},
		"not uuid":  {"SessionID": "my-session"},
		"traversal": {"SessionID": "../triggers"},
	} {
		t.Run(name, func(t *testing.T) {
			res, err := tool.getTriggerBySession(context.Background(), request(args))
			require.NoError(t, err)
			require.True(t, res.IsError)
		})
	}
}

func TestGetTriggerBySessionNotFound(t *testing.T) {
	tool := newTestTool(t, func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusNotFound)
		_, _ = w.Write([]byte(`{"id":"not_found","message":"no trigger for session"}`))
	})

	res, err := tool.getTriggerBySession(context.Background(), request(map[string]any{"SessionID": testSessionID}))
	require.NoError(t, err)
	require.True(t, res.IsError)
	require.Contains(t, callText(t, res), "Failed to get trigger by session")
}

// ---- ListReusableSessions ----

func TestListReusableSessions(t *testing.T) {
	var gotPath string
	var gotQuery url.Values
	tool := newTestTool(t, func(w http.ResponseWriter, r *http.Request) {
		gotPath = r.URL.Path
		gotQuery = r.URL.Query()
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"sessions":[{"session_id":"` + testSessionID + `","name":"idle-session","status":"SESSION_STATUS_READY"}],"next_page_token":"tok2"}`))
	})

	res, err := tool.listReusableSessions(context.Background(), request(map[string]any{
		"PageSize":  float64(5),
		"PageToken": "tok1",
	}))
	require.NoError(t, err)
	require.False(t, res.IsError)
	require.Equal(t, "/v2/harness-runtime/triggers/reusable-sessions", gotPath)
	require.Equal(t, "5", gotQuery.Get("page_size"))
	require.Equal(t, "tok1", gotQuery.Get("page_token"))
	text := callText(t, res)
	require.Contains(t, text, testSessionID)
	require.Contains(t, text, "idle-session")
	require.Contains(t, text, "tok2")
}

func TestListReusableSessionsNoQueryParams(t *testing.T) {
	var gotRawQuery string
	tool := newTestTool(t, func(w http.ResponseWriter, r *http.Request) {
		gotRawQuery = r.URL.RawQuery
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{}`))
	})

	res, err := tool.listReusableSessions(context.Background(), request(nil))
	require.NoError(t, err)
	require.False(t, res.IsError)
	require.Empty(t, gotRawQuery)
	require.Regexp(t, `"sessions":\s*\[\]`, callText(t, res))
}

func TestListReusableSessionsValidation(t *testing.T) {
	tool := newTestTool(t, func(http.ResponseWriter, *http.Request) {
		t.Fatal("API must not be called for invalid input")
	})

	for name, args := range map[string]map[string]any{
		"page size too small": {"PageSize": float64(0)},
		"page size too large": {"PageSize": float64(201)},
		"page size fraction":  {"PageSize": 1.7},
	} {
		t.Run(name, func(t *testing.T) {
			res, err := tool.listReusableSessions(context.Background(), request(args))
			require.NoError(t, err)
			require.True(t, res.IsError)
		})
	}
}

func TestListReusableSessionsAPIError(t *testing.T) {
	tool := newTestTool(t, func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusInternalServerError)
		_, _ = w.Write([]byte(`{"id":"server_error","message":"internal error"}`))
	})

	res, err := tool.listReusableSessions(context.Background(), request(nil))
	require.NoError(t, err)
	require.True(t, res.IsError)
	require.Contains(t, callText(t, res), "Failed to list reusable sessions")
}
