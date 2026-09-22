package common

import (
	"bytes"
	"encoding/json"
	"io"
	"testing"
	"time"

	"github.com/digitalocean/godo"
	"github.com/mark3labs/mcp-go/mcp"
	"github.com/santhosh-tekuri/jsonschema/v6"
)

func bytesReader(b []byte) io.Reader { return bytes.NewReader(b) }

func textOf(t *testing.T, res *mcp.CallToolResult) string {
	t.Helper()
	if len(res.Content) != 1 {
		t.Fatalf("expected exactly one content block, got %d", len(res.Content))
	}
	tc, ok := res.Content[0].(mcp.TextContent)
	if !ok {
		t.Fatalf("content block is %T, want mcp.TextContent", res.Content[0])
	}
	return tc.Text
}

// validateAgainst compiles raw and validates value, mirroring what mcp-go's
// WithOutputSchemaValidation (and a spec-compliant client) does to
// structuredContent.
func validateAgainst(t *testing.T, raw json.RawMessage, structured any) {
	t.Helper()

	if len(raw) == 0 {
		t.Fatal("no output schema was generated")
	}

	schemaDoc, err := jsonschema.UnmarshalJSON(bytesReader(raw))
	if err != nil {
		t.Fatalf("unmarshal schema: %v", err)
	}
	c := jsonschema.NewCompiler()
	if err := c.AddResource("schema.json", schemaDoc); err != nil {
		t.Fatalf("add resource: %v", err)
	}
	compiled, err := c.Compile("schema.json")
	if err != nil {
		t.Fatalf("compile schema: %v", err)
	}

	encoded, err := json.Marshal(structured)
	if err != nil {
		t.Fatalf("marshal structured content: %v", err)
	}
	doc, err := jsonschema.UnmarshalJSON(bytesReader(encoded))
	if err != nil {
		t.Fatalf("unmarshal structured content: %v", err)
	}

	if err := compiled.Validate(doc); err != nil {
		t.Errorf("structuredContent does not satisfy the declared outputSchema:\n%v\n\nschema: %s\n\ncontent: %s",
			err, raw, encoded)
	}
}

func TestOutputResultCarriesTextAndStructuredContent(t *testing.T) {
	out := NewOutput[[]godo.Region]("regions")

	regions := []godo.Region{
		{Slug: "blr1", Name: "Bangalore 1", Available: true, Sizes: []string{"s-1vcpu-1gb"}},
	}

	res, err := out.Result(regions)
	if err != nil {
		t.Fatalf("Result: %v", err)
	}

	// Text content must stay byte-identical to the pre-structured-output
	// behaviour so existing consumers are unaffected.
	want, err := json.MarshalIndent(regions, "", "  ")
	if err != nil {
		t.Fatal(err)
	}
	text := textOf(t, res)
	if text != string(want) {
		t.Errorf("text content changed:\n got: %s\nwant: %s", text, want)
	}

	if res.StructuredContent == nil {
		t.Fatal("StructuredContent is nil")
	}

	encoded, err := json.Marshal(res.StructuredContent)
	if err != nil {
		t.Fatal(err)
	}
	var envelope map[string]json.RawMessage
	if err := json.Unmarshal(encoded, &envelope); err != nil {
		t.Fatalf("structuredContent is not a JSON object: %v", err)
	}
	if _, ok := envelope["regions"]; !ok {
		t.Errorf("structuredContent missing %q key, got %s", "regions", encoded)
	}

	validateAgainst(t, out.RawSchema(), res.StructuredContent)
}

// A nil slice marshals to null, which would violate an "array" schema.
func TestOutputNormalizesNilSliceToEmptyArray(t *testing.T) {
	out := NewOutput[[]godo.Region]("regions")

	res, err := out.Result(nil)
	if err != nil {
		t.Fatalf("Result: %v", err)
	}

	encoded, err := json.Marshal(res.StructuredContent)
	if err != nil {
		t.Fatal(err)
	}
	if string(encoded) != `{"regions":[]}` {
		t.Errorf("nil slice: got %s, want {\"regions\":[]}", encoded)
	}

	validateAgainst(t, out.RawSchema(), res.StructuredContent)
}

// These are the godo types that break mcp-go's own WithOutputSchema reflector:
// godo.Action embeds godo.Timestamp (a struct that marshals as a string) and
// godo.App is self-referential through godo.DeploymentProgressStep.
func TestOutputHandlesGodoTypesThatBreakTheSDKReflector(t *testing.T) {
	t.Run("action embeds Timestamp", func(t *testing.T) {
		out := NewOutput[*godo.Action]("action")
		ts := &godo.Timestamp{Time: time.Now().UTC()}
		res, err := out.Result(&godo.Action{ID: 1, Status: "completed", Type: "create", StartedAt: ts, CompletedAt: ts})
		if err != nil {
			t.Fatal(err)
		}
		validateAgainst(t, out.RawSchema(), res.StructuredContent)
	})

	t.Run("recursive app spec", func(t *testing.T) {
		out := NewOutput[[]godo.App]("apps")
		res, err := out.Result([]godo.App{{
			ID:        "009ef225-576d-4920-88a4-8d4f4af7ba41",
			Spec:      &godo.AppSpec{Name: "logpoison-aspnetapp"},
			Region:    &godo.AppRegion{Slug: "blr", Label: "Bangalore", Continent: "Asia"},
			CreatedAt: time.Now().UTC(),
		}})
		if err != nil {
			t.Fatal(err)
		}
		validateAgainst(t, out.RawSchema(), res.StructuredContent)
	})

	t.Run("droplet", func(t *testing.T) {
		out := NewOutput[[]godo.Droplet]("droplets")
		res, err := out.Result([]godo.Droplet{{ID: 1, Name: "web-1", Status: "active"}})
		if err != nil {
			t.Fatal(err)
		}
		validateAgainst(t, out.RawSchema(), res.StructuredContent)
	})

	t.Run("kubernetes cluster", func(t *testing.T) {
		out := NewOutput[[]godo.KubernetesCluster]("clusters")
		res, err := out.Result([]godo.KubernetesCluster{{ID: "abc", Name: "k8s-1"}})
		if err != nil {
			t.Fatal(err)
		}
		validateAgainst(t, out.RawSchema(), res.StructuredContent)
	})
}

type paginatedRepos struct {
	Repositories []*godo.RepositoryV2 `json:"repositories"`
	Meta         *godo.Meta           `json:"meta,omitempty"`
}

func TestObjectOutputPublishesPayloadWithoutAnEnvelope(t *testing.T) {
	out := NewObjectOutput[paginatedRepos]()

	payload := paginatedRepos{
		Repositories: []*godo.RepositoryV2{{Name: "repo1", RegistryName: "reg"}},
		Meta:         &godo.Meta{Total: 1, Page: 1, Pages: 1},
	}

	res, err := out.Result(payload)
	if err != nil {
		t.Fatalf("Result: %v", err)
	}

	// The payload is already an object, so structuredContent and the text
	// block carry the same JSON.
	encoded, err := json.Marshal(res.StructuredContent)
	if err != nil {
		t.Fatal(err)
	}
	var structured, fromText paginatedRepos
	if err := json.Unmarshal(encoded, &structured); err != nil {
		t.Fatalf("structuredContent: %v", err)
	}
	if err := json.Unmarshal([]byte(textOf(t, res)), &fromText); err != nil {
		t.Fatalf("text content: %v", err)
	}
	if len(structured.Repositories) != 1 || structured.Repositories[0].Name != "repo1" {
		t.Errorf("unexpected structuredContent: %s", encoded)
	}
	if len(fromText.Repositories) != len(structured.Repositories) {
		t.Errorf("text and structured content disagree: %s", encoded)
	}

	validateAgainst(t, out.RawSchema(), res.StructuredContent)
}

// MCP requires the output schema root to be an object, so the root must be
// inlined rather than emitted as a top-level $ref.
func TestObjectOutputSchemaRootIsAnInlinedObject(t *testing.T) {
	out := NewObjectOutput[paginatedRepos]()

	var schema struct {
		Type  string                     `json:"type"`
		Ref   string                     `json:"$ref"`
		Props map[string]json.RawMessage `json:"properties"`
	}
	if err := json.Unmarshal(out.RawSchema(), &schema); err != nil {
		t.Fatalf("schema is not valid JSON: %v", err)
	}

	if schema.Type != "object" {
		t.Errorf("top-level type = %q, want \"object\"", schema.Type)
	}
	if schema.Ref != "" {
		t.Errorf("top-level schema is a $ref (%q), which is not an object", schema.Ref)
	}
	if _, ok := schema.Props["repositories"]; !ok {
		t.Errorf("schema does not describe the payload fields: %s", out.RawSchema())
	}
}

// A nil pointer payload would serialise to null, but structuredContent has to
// be an object.
func TestObjectOutputNormalizesNilPayloadToEmptyObject(t *testing.T) {
	out := NewObjectOutput[*paginatedRepos]()

	res, err := out.Result(nil)
	if err != nil {
		t.Fatalf("Result: %v", err)
	}

	encoded, err := json.Marshal(res.StructuredContent)
	if err != nil {
		t.Fatal(err)
	}
	if string(encoded) != "{}" {
		t.Errorf("nil payload: got %s, want {}", encoded)
	}

	validateAgainst(t, out.RawSchema(), res.StructuredContent)
}

func TestOutputSchemaIsAnObjectEnvelope(t *testing.T) {
	out := NewOutput[[]godo.Region]("regions")

	var schema struct {
		Type     string              `json:"type"`
		Props    map[string]any      `json:"properties"`
		Required []string            `json:"required"`
		Defs     map[string]struct{} `json:"$defs"`
	}
	if err := json.Unmarshal(out.RawSchema(), &schema); err != nil {
		t.Fatalf("schema is not valid JSON: %v", err)
	}

	// MCP requires the top-level output schema to be type "object".
	if schema.Type != "object" {
		t.Errorf("top-level type = %q, want \"object\"", schema.Type)
	}
	if _, ok := schema.Props["regions"]; !ok {
		t.Errorf("schema has no %q property: %s", "regions", out.RawSchema())
	}
	if len(schema.Required) != 1 || schema.Required[0] != "regions" {
		t.Errorf("required = %v, want [regions]", schema.Required)
	}
	// $defs must be hoisted to the root or the payload's $ref pointers dangle.
	if len(schema.Defs) == 0 {
		t.Errorf("$defs was not hoisted to the schema root: %s", out.RawSchema())
	}
}
