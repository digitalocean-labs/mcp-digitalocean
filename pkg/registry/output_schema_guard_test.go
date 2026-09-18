package registry

import (
	"bytes"
	"encoding/json"
	"testing"

	"github.com/santhosh-tekuri/jsonschema/v6"
)

// TestDeclaredOutputSchemasAreUsable is the backstop guard for structured
// output, and the counterpart to TestEveryRegisteredToolAnnotated.
//
// It does not require every tool to declare an output schema — the rollout is
// incremental. What it enforces is that any schema a tool does declare is one
// a spec-compliant client can actually use: the spec has clients validate
// structuredContent against outputSchema, so a schema that fails to compile
// or is not an object envelope would make the tool worse than having no
// schema at all.
//
// common.Output swallows reflection failures by design (a missing schema
// degrades gracefully, a panicking server does not), which is exactly why the
// empty-schema case has to be caught here instead.
func TestDeclaredOutputSchemasAreUsable(t *testing.T) {
	tools := registerAllTools(t)

	for name, st := range tools {
		raw := st.Tool.RawOutputSchema
		if len(raw) == 0 {
			continue
		}

		var schema struct {
			Type       string                     `json:"type"`
			Ref        string                     `json:"$ref"`
			Properties map[string]json.RawMessage `json:"properties"`
		}
		if err := json.Unmarshal(raw, &schema); err != nil {
			t.Errorf("tool %q: output schema is not valid JSON: %v", name, err)
			continue
		}

		// MCP requires the top-level output schema to be an object, which is
		// why common.Output publishes array payloads under a named field and
		// common.ObjectOutput inlines the payload's own properties instead of
		// emitting a top-level $ref.
		if schema.Type != "object" {
			t.Errorf("tool %q: output schema top-level type is %q, want \"object\"", name, schema.Type)
		}
		if schema.Ref != "" {
			t.Errorf("tool %q: output schema root is a $ref (%q), which is not an object", name, schema.Ref)
		}
		if len(schema.Properties) == 0 {
			t.Errorf("tool %q: output schema declares no properties, so it describes nothing", name)
		}

		if err := compileSchema(raw); err != nil {
			t.Errorf("tool %q: output schema does not compile, so clients cannot validate against it: %v",
				name, err)
		}
	}
}

// compileSchema runs the schema through the same validator mcp-go uses for
// WithOutputSchemaValidation. It catches dangling $ref pointers, which is the
// failure mode when $defs are not hoisted to the schema root.
func compileSchema(raw json.RawMessage) error {
	doc, err := jsonschema.UnmarshalJSON(bytes.NewReader(raw))
	if err != nil {
		return err
	}
	c := jsonschema.NewCompiler()
	if err := c.AddResource("output.json", doc); err != nil {
		return err
	}
	_, err = c.Compile("output.json")
	return err
}

// TestOutputSchemaSizeBudget reports what structured output costs in the
// tools/list response. Output schemas reflected from godo response types run
// from a few hundred bytes to ~24KB each, so a full 332-tool rollout is a
// material change to a payload every client fetches on connect. The test
// prints the numbers rather than asserting a ceiling; run it with -v when
// migrating a service.
func TestOutputSchemaSizeBudget(t *testing.T) {
	tools := registerAllTools(t)

	var totalWire, totalOutputSchema, withSchema int
	for _, st := range tools {
		encoded, err := json.Marshal(st.Tool)
		if err != nil {
			t.Fatalf("marshal tool %q: %v", st.Tool.Name, err)
		}
		totalWire += len(encoded)
		if n := len(st.Tool.RawOutputSchema); n > 0 {
			withSchema++
			totalOutputSchema += n
		}
	}

	t.Logf("tools registered:              %d", len(tools))
	t.Logf("tools declaring outputSchema:  %d", withSchema)
	t.Logf("outputSchema bytes:            %d (%.1f KB)", totalOutputSchema, float64(totalOutputSchema)/1024)
	t.Logf("total tools/list bytes:        %d (%.1f KB)", totalWire, float64(totalWire)/1024)
	if withSchema > 0 {
		t.Logf("mean outputSchema:             %d bytes", totalOutputSchema/withSchema)
		t.Logf("projected for all %d tools:   %.1f KB of outputSchema",
			len(tools), float64(totalOutputSchema)/float64(withSchema)*float64(len(tools))/1024)
	}
}
