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
	testTemplateID = "019fb39c-14d9-7080-933e-b9b90e25acdc"
	testBuildID    = "019fb39c-14d9-7080-933e-b9b90e25acdd"
)

func templateJSON(templateID, name, status string) string {
	return `{"template_id":"` + templateID + `","name":"` + name + `","status":"` + status + `","spec":{"base_template":"codex-base","image":{"registry":"registry.example.com","repository":"my-image","tag":"latest"}},"created_at":"2026-08-01T12:00:00Z","updated_at":"2026-08-01T12:00:00Z"}`
}

func buildJSON(buildID, templateID, status string) string {
	return `{"build_id":"` + buildID + `","template_id":"` + templateID + `","status":"` + status + `","spec":{"base_template":"codex-base"},"created_at":"2026-08-01T12:00:00Z","updated_at":"2026-08-01T12:00:00Z"}`
}

// ---- ListTemplates ----

func TestListTemplates(t *testing.T) {
	var gotPath string
	var gotQuery url.Values

	tool := newTestTool(t, func(w http.ResponseWriter, r *http.Request) {
		gotPath = r.URL.Path
		gotQuery = r.URL.Query()
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"templates":[` + templateJSON(testTemplateID, "my-template", "TEMPLATE_STATUS_READY") + `],"next_page_token":"tok2"}`))
	})

	res, err := tool.listTemplates(context.Background(), request(map[string]any{
		"PageSize":  float64(10),
		"PageToken": "tok1",
	}))
	require.NoError(t, err)
	require.False(t, res.IsError)
	require.Equal(t, "/v2/harness-runtime/templates", gotPath)
	require.Equal(t, "10", gotQuery.Get("page_size"))
	require.Equal(t, "tok1", gotQuery.Get("page_token"))

	text := callText(t, res)
	require.Contains(t, text, "my-template")
	require.Contains(t, text, testTemplateID)
	require.Contains(t, text, "tok2")
}

func TestListTemplatesNoArgsOmitsQuery(t *testing.T) {
	var gotRawQuery string
	tool := newTestTool(t, func(w http.ResponseWriter, r *http.Request) {
		gotRawQuery = r.URL.RawQuery
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{}`))
	})

	res, err := tool.listTemplates(context.Background(), request(nil))
	require.NoError(t, err)
	require.False(t, res.IsError)
	require.Empty(t, gotRawQuery)
	require.Regexp(t, `"templates":\s*\[\]`, callText(t, res))
}

func TestListTemplatesValidation(t *testing.T) {
	tool := newTestTool(t, func(http.ResponseWriter, *http.Request) {
		t.Fatal("API must not be called for invalid input")
	})

	for name, args := range map[string]map[string]any{
		"page size too small": {"PageSize": float64(0)},
		"page size too large": {"PageSize": float64(201)},
		"page size fraction":  {"PageSize": 1.5},
	} {
		t.Run(name, func(t *testing.T) {
			res, err := tool.listTemplates(context.Background(), request(args))
			require.NoError(t, err)
			require.True(t, res.IsError)
		})
	}
}

func TestListTemplatesAPIError(t *testing.T) {
	tool := newTestTool(t, func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusUnauthorized)
		_, _ = w.Write([]byte(`{"id":"unauthorized","message":"Unable to authenticate you"}`))
	})

	res, err := tool.listTemplates(context.Background(), request(nil))
	require.NoError(t, err)
	require.True(t, res.IsError)
	require.Contains(t, callText(t, res), "Failed to list templates")
}

// ---- GetTemplate ----

func TestGetTemplate(t *testing.T) {
	var gotPath string

	tool := newTestTool(t, func(w http.ResponseWriter, r *http.Request) {
		gotPath = r.URL.Path
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"template":` + templateJSON(testTemplateID, "my-template", "TEMPLATE_STATUS_READY") + `}`))
	})

	res, err := tool.getTemplate(context.Background(), request(map[string]any{"TemplateID": testTemplateID}))
	require.NoError(t, err)
	require.False(t, res.IsError)
	require.Equal(t, "/v2/harness-runtime/templates/"+testTemplateID, gotPath)

	text := callText(t, res)
	require.Contains(t, text, "my-template")
	require.Contains(t, text, "codex-base")
}

func TestGetTemplateValidation(t *testing.T) {
	tool := newTestTool(t, func(http.ResponseWriter, *http.Request) {
		t.Fatal("API must not be called for invalid input")
	})

	for name, args := range map[string]map[string]any{
		"missing":  {},
		"not uuid": {"TemplateID": "my-template"},
	} {
		t.Run(name, func(t *testing.T) {
			res, err := tool.getTemplate(context.Background(), request(args))
			require.NoError(t, err)
			require.True(t, res.IsError)
		})
	}
}

func TestGetTemplateNotFound(t *testing.T) {
	tool := newTestTool(t, func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusNotFound)
		_, _ = w.Write([]byte(`{"id":"not_found","message":"template not found"}`))
	})

	res, err := tool.getTemplate(context.Background(), request(map[string]any{"TemplateID": testTemplateID}))
	require.NoError(t, err)
	require.True(t, res.IsError)
	require.Contains(t, callText(t, res), "Failed to get template")
}

// ---- CreateTemplate ----

func TestCreateTemplate(t *testing.T) {
	var gotPath, gotMethod, gotContentType string
	var gotBody map[string]any

	tool := newTestTool(t, func(w http.ResponseWriter, r *http.Request) {
		gotPath = r.URL.Path
		gotMethod = r.Method
		gotContentType = r.Header.Get("Content-Type")
		_ = json.NewDecoder(r.Body).Decode(&gotBody)
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusCreated)
		_, _ = w.Write([]byte(`{"template":` + templateJSON(testTemplateID, "my-template", "TEMPLATE_STATUS_PENDING") + `}`))
	})

	res, err := tool.createTemplate(context.Background(), request(map[string]any{
		"Name":         "my-template",
		"BaseTemplate": "codex-base",
		"SourceOCIRef": "registry.example.com/my-image:latest",
	}))
	require.NoError(t, err)
	require.False(t, res.IsError)
	require.Equal(t, "/v2/harness-runtime/templates", gotPath)
	require.Equal(t, http.MethodPost, gotMethod)
	require.Contains(t, gotContentType, "application/json")
	require.Equal(t, "my-template", gotBody["name"])
	require.Equal(t, "codex-base", gotBody["base_template"])
	require.Equal(t, "registry.example.com/my-image:latest", gotBody["source_oci_ref"])

	text := callText(t, res)
	require.Contains(t, text, testTemplateID)
	require.Contains(t, text, "TEMPLATE_STATUS_PENDING")
}

func TestCreateTemplateValidation(t *testing.T) {
	tool := newTestTool(t, func(http.ResponseWriter, *http.Request) {
		t.Fatal("API must not be called for invalid input")
	})

	for name, args := range map[string]map[string]any{
		"missing all":          {},
		"missing name":         {"BaseTemplate": "codex-base", "SourceOCIRef": "reg/img:v1"},
		"missing base":         {"Name": "my-tpl", "SourceOCIRef": "reg/img:v1"},
		"missing source":       {"Name": "my-tpl", "BaseTemplate": "codex-base"},
		"empty name":           {"Name": "", "BaseTemplate": "codex-base", "SourceOCIRef": "reg/img:v1"},
		"empty base":           {"Name": "my-tpl", "BaseTemplate": "", "SourceOCIRef": "reg/img:v1"},
		"empty source oci ref": {"Name": "my-tpl", "BaseTemplate": "codex-base", "SourceOCIRef": ""},
	} {
		t.Run(name, func(t *testing.T) {
			res, err := tool.createTemplate(context.Background(), request(args))
			require.NoError(t, err)
			require.True(t, res.IsError)
		})
	}
}

func TestCreateTemplateAPIError(t *testing.T) {
	tool := newTestTool(t, func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusBadRequest)
		_, _ = w.Write([]byte(`{"id":"bad_request","message":"invalid base template"}`))
	})

	res, err := tool.createTemplate(context.Background(), request(map[string]any{
		"Name":         "my-tpl",
		"BaseTemplate": "invalid-base",
		"SourceOCIRef": "reg/img:v1",
	}))
	require.NoError(t, err)
	require.True(t, res.IsError)
	require.Contains(t, callText(t, res), "Failed to create template")
}

// ---- UpdateTemplate ----

func TestUpdateTemplate(t *testing.T) {
	var gotPath, gotMethod string
	var gotBody map[string]any

	tool := newTestTool(t, func(w http.ResponseWriter, r *http.Request) {
		gotPath = r.URL.Path
		gotMethod = r.Method
		_ = json.NewDecoder(r.Body).Decode(&gotBody)
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"template":` + templateJSON(testTemplateID, "my-template", "TEMPLATE_STATUS_BUILDING") + `}`))
	})

	res, err := tool.updateTemplate(context.Background(), request(map[string]any{
		"TemplateID":   testTemplateID,
		"SourceOCIRef": "registry.example.com/my-image:v2",
		"BaseTemplate": "opencode-base",
	}))
	require.NoError(t, err)
	require.False(t, res.IsError)
	require.Equal(t, "/v2/harness-runtime/templates/"+testTemplateID, gotPath)
	require.Equal(t, http.MethodPut, gotMethod)
	require.Equal(t, "registry.example.com/my-image:v2", gotBody["source_oci_ref"])
	require.Equal(t, "opencode-base", gotBody["base_template"])

	text := callText(t, res)
	require.Contains(t, text, "TEMPLATE_STATUS_BUILDING")
}

func TestUpdateTemplateSourceOnly(t *testing.T) {
	var gotBody map[string]any

	tool := newTestTool(t, func(w http.ResponseWriter, r *http.Request) {
		_ = json.NewDecoder(r.Body).Decode(&gotBody)
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"template":` + templateJSON(testTemplateID, "my-template", "TEMPLATE_STATUS_BUILDING") + `}`))
	})

	res, err := tool.updateTemplate(context.Background(), request(map[string]any{
		"TemplateID":   testTemplateID,
		"SourceOCIRef": "registry.example.com/my-image:v3",
	}))
	require.NoError(t, err)
	require.False(t, res.IsError)
	require.Equal(t, "registry.example.com/my-image:v3", gotBody["source_oci_ref"])
	_, hasBase := gotBody["base_template"]
	require.False(t, hasBase)
}

func TestUpdateTemplateValidation(t *testing.T) {
	tool := newTestTool(t, func(http.ResponseWriter, *http.Request) {
		t.Fatal("API must not be called for invalid input")
	})

	for name, args := range map[string]map[string]any{
		"missing template id":   {"SourceOCIRef": "reg/img:v1"},
		"invalid template id":   {"TemplateID": "not-a-uuid", "SourceOCIRef": "reg/img:v1"},
		"missing update fields": {"TemplateID": testTemplateID},
	} {
		t.Run(name, func(t *testing.T) {
			res, err := tool.updateTemplate(context.Background(), request(args))
			require.NoError(t, err)
			require.True(t, res.IsError)
		})
	}
}

func TestUpdateTemplateAPIError(t *testing.T) {
	tool := newTestTool(t, func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusNotFound)
		_, _ = w.Write([]byte(`{"id":"not_found","message":"template not found"}`))
	})

	res, err := tool.updateTemplate(context.Background(), request(map[string]any{
		"TemplateID":   testTemplateID,
		"SourceOCIRef": "reg/img:v1",
	}))
	require.NoError(t, err)
	require.True(t, res.IsError)
	require.Contains(t, callText(t, res), "Failed to update template")
}

// ---- DeleteTemplate ----

func TestDeleteTemplate(t *testing.T) {
	var gotPath, gotMethod string

	tool := newTestTool(t, func(w http.ResponseWriter, r *http.Request) {
		gotPath = r.URL.Path
		gotMethod = r.Method
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"template_id":"` + testTemplateID + `","deleted":true}`))
	})

	res, err := tool.deleteTemplate(context.Background(), request(map[string]any{"TemplateID": testTemplateID}))
	require.NoError(t, err)
	require.False(t, res.IsError)
	require.Equal(t, "/v2/harness-runtime/templates/"+testTemplateID, gotPath)
	require.Equal(t, http.MethodDelete, gotMethod)

	text := callText(t, res)
	require.Contains(t, text, testTemplateID)
	require.Contains(t, text, "true")
}

func TestDeleteTemplateValidation(t *testing.T) {
	tool := newTestTool(t, func(http.ResponseWriter, *http.Request) {
		t.Fatal("API must not be called for invalid input")
	})

	for name, args := range map[string]map[string]any{
		"missing":  {},
		"not uuid": {"TemplateID": "my-template"},
	} {
		t.Run(name, func(t *testing.T) {
			res, err := tool.deleteTemplate(context.Background(), request(args))
			require.NoError(t, err)
			require.True(t, res.IsError)
		})
	}
}

func TestDeleteTemplateAPIError(t *testing.T) {
	tool := newTestTool(t, func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusNotFound)
		_, _ = w.Write([]byte(`{"id":"not_found","message":"template not found"}`))
	})

	res, err := tool.deleteTemplate(context.Background(), request(map[string]any{"TemplateID": testTemplateID}))
	require.NoError(t, err)
	require.True(t, res.IsError)
	require.Contains(t, callText(t, res), "Failed to delete template")
}

// ---- ListTemplateBuilds ----

func TestListTemplateBuilds(t *testing.T) {
	var gotPath string
	var gotQuery url.Values

	tool := newTestTool(t, func(w http.ResponseWriter, r *http.Request) {
		gotPath = r.URL.Path
		gotQuery = r.URL.Query()
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"builds":[` + buildJSON(testBuildID, testTemplateID, "SUCCEEDED") + `],"next_page_token":"tok2"}`))
	})

	res, err := tool.listTemplateBuilds(context.Background(), request(map[string]any{
		"TemplateID": testTemplateID,
		"PageSize":   float64(5),
		"PageToken":  "tok1",
	}))
	require.NoError(t, err)
	require.False(t, res.IsError)
	require.Equal(t, "/v2/harness-runtime/templates/"+testTemplateID+"/builds", gotPath)
	require.Equal(t, "5", gotQuery.Get("page_size"))
	require.Equal(t, "tok1", gotQuery.Get("page_token"))

	text := callText(t, res)
	require.Contains(t, text, testBuildID)
	require.Contains(t, text, "SUCCEEDED")
	require.Contains(t, text, "tok2")
}

func TestListTemplateBuildsNoQuery(t *testing.T) {
	var gotRawQuery string

	tool := newTestTool(t, func(w http.ResponseWriter, r *http.Request) {
		gotRawQuery = r.URL.RawQuery
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{}`))
	})

	res, err := tool.listTemplateBuilds(context.Background(), request(map[string]any{
		"TemplateID": testTemplateID,
	}))
	require.NoError(t, err)
	require.False(t, res.IsError)
	require.Empty(t, gotRawQuery)
	require.Regexp(t, `"builds":\s*\[\]`, callText(t, res))
}

func TestListTemplateBuildsValidation(t *testing.T) {
	tool := newTestTool(t, func(http.ResponseWriter, *http.Request) {
		t.Fatal("API must not be called for invalid input")
	})

	for name, args := range map[string]map[string]any{
		"missing template id": {},
		"invalid template id": {"TemplateID": "not-a-uuid"},
		"page size too small": {"TemplateID": testTemplateID, "PageSize": float64(0)},
		"page size too large": {"TemplateID": testTemplateID, "PageSize": float64(201)},
	} {
		t.Run(name, func(t *testing.T) {
			res, err := tool.listTemplateBuilds(context.Background(), request(args))
			require.NoError(t, err)
			require.True(t, res.IsError)
		})
	}
}

func TestListTemplateBuildsAPIError(t *testing.T) {
	tool := newTestTool(t, func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusNotFound)
		_, _ = w.Write([]byte(`{"id":"not_found","message":"template not found"}`))
	})

	res, err := tool.listTemplateBuilds(context.Background(), request(map[string]any{"TemplateID": testTemplateID}))
	require.NoError(t, err)
	require.True(t, res.IsError)
	require.Contains(t, callText(t, res), "Failed to list template builds")
}

// ---- GetTemplateBuild ----

func TestGetTemplateBuild(t *testing.T) {
	var gotPath string

	tool := newTestTool(t, func(w http.ResponseWriter, r *http.Request) {
		gotPath = r.URL.Path
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"build":` + buildJSON(testBuildID, testTemplateID, "SUCCEEDED") + `}`))
	})

	res, err := tool.getTemplateBuild(context.Background(), request(map[string]any{
		"TemplateID": testTemplateID,
		"BuildID":    testBuildID,
	}))
	require.NoError(t, err)
	require.False(t, res.IsError)
	require.Equal(t, "/v2/harness-runtime/templates/"+testTemplateID+"/builds/"+testBuildID, gotPath)

	text := callText(t, res)
	require.Contains(t, text, testBuildID)
	require.Contains(t, text, "SUCCEEDED")
}

func TestGetTemplateBuildValidation(t *testing.T) {
	tool := newTestTool(t, func(http.ResponseWriter, *http.Request) {
		t.Fatal("API must not be called for invalid input")
	})

	for name, args := range map[string]map[string]any{
		"missing template id": {"BuildID": testBuildID},
		"invalid template id": {"TemplateID": "not-a-uuid", "BuildID": testBuildID},
		"missing build id":    {"TemplateID": testTemplateID},
		"invalid build id":    {"TemplateID": testTemplateID, "BuildID": "not-a-uuid"},
	} {
		t.Run(name, func(t *testing.T) {
			res, err := tool.getTemplateBuild(context.Background(), request(args))
			require.NoError(t, err)
			require.True(t, res.IsError)
		})
	}
}

func TestGetTemplateBuildAPIError(t *testing.T) {
	tool := newTestTool(t, func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusNotFound)
		_, _ = w.Write([]byte(`{"id":"not_found","message":"build not found"}`))
	})

	res, err := tool.getTemplateBuild(context.Background(), request(map[string]any{
		"TemplateID": testTemplateID,
		"BuildID":    testBuildID,
	}))
	require.NoError(t, err)
	require.True(t, res.IsError)
	require.Contains(t, callText(t, res), "Failed to get template build")
}
