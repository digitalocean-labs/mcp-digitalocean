package registry

import (
	"bytes"
	"encoding/json"
	"testing"

	"github.com/santhosh-tekuri/jsonschema/v6"
)

// TestDeclaredOutputSchemasAreUsable checks that any declared outputSchema
// compiles and is a usable object root. Rollout is incremental — tools with
// no schema are skipped. Catches reflection failures that Output swallows.
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

		// MCP requires an object root (envelope or inlined properties).
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

// compileSchema uses the same validator as mcp-go WithOutputSchemaValidation.
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

// TestOutputSchemaSizeBudget logs tools/list size impact (run with -v).
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
