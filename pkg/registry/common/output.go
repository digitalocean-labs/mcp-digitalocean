package common

import (
	"bytes"
	"encoding/json"
	"fmt"
	"reflect"
	"sync"
	"time"

	"github.com/invopop/jsonschema"
	"github.com/mark3labs/mcp-go/mcp"
)

// Output owns both halves of MCP's structured output contract for one tool:
// Schema() declares the outputSchema, Result() emits the matching
// structuredContent. Because one value drives both, they cannot drift.
//
// A field name wraps the payload, which is how array-returning tools satisfy
// MCP's object-root requirement:
//
//	var actionsOut = common.NewOutput[[]godo.Action]("actions")
//	mcp.NewTool("action-list", actionsOut.Schema(), ...)
//	return actionsOut.Result(actions) // {"actions": [...]}
type Output[T any] struct {
	field string // "" publishes T directly, with no envelope
	once  sync.Once
	raw   json.RawMessage
}

// NewOutput publishes T under field ("droplet", "droplets", …).
func NewOutput[T any](field string) *Output[T] {
	return &Output[T]{field: field}
}

// NewObjectOutput publishes T as structuredContent as-is. Use it for result
// structs that already pair a collection with its metadata, where an envelope
// would add a redundant second key. T must be a struct or a pointer to one.
func NewObjectOutput[T any]() *Output[T] {
	return &Output[T]{}
}

// Field is the structuredContent key, or "" when T is published directly.
func (o *Output[T]) Field() string { return o.field }

// Schema declares this Output as the tool's outputSchema.
func (o *Output[T]) Schema() mcp.ToolOption {
	return mcp.WithRawOutputSchema(o.RawSchema())
}

// RawSchema is the generated schema, reflected once on first use. Exported so
// tests can assert that the schema and the emitted payload agree.
func (o *Output[T]) RawSchema() json.RawMessage {
	o.once.Do(func() {
		raw, err := buildSchema[T](o.field)
		if err != nil {
			// A tool without an output schema still works, so degrade instead
			// of failing the server; the registry guard test catches this.
			return
		}
		o.raw = raw
	})
	return o.raw
}

// Result pairs the text content tools returned before structured output
// existed with the equivalent structuredContent.
func (o *Output[T]) Result(payload T) (*mcp.CallToolResult, error) {
	text, err := json.MarshalIndent(payload, "", "  ")
	if err != nil {
		return nil, fmt.Errorf("marshal error: %w", err)
	}

	res := mcp.NewToolResultText(string(text))
	res.StructuredContent = o.structured(text)
	return res, nil
}

// structured shapes the encoded payload for structuredContent, keeping it a
// JSON object even when the payload itself encodes to null.
func (o *Output[T]) structured(text json.RawMessage) any {
	if o.field != "" {
		return map[string]json.RawMessage{o.field: emptyForNull[T](text)}
	}
	if string(text) == "null" {
		return json.RawMessage("{}")
	}
	return text
}

// emptyForNull maps a null payload to [] or {} so that nil collections still
// satisfy an "array"/"object" schema.
func emptyForNull[T any](encoded json.RawMessage) json.RawMessage {
	if string(encoded) != "null" {
		return encoded
	}
	switch derefKind(reflect.TypeFor[T]()) {
	case reflect.Slice, reflect.Array:
		return json.RawMessage("[]")
	case reflect.Map:
		return json.RawMessage("{}")
	default:
		return encoded
	}
}

func derefKind(t reflect.Type) reflect.Kind {
	for t != nil && t.Kind() == reflect.Ptr {
		t = t.Elem()
	}
	if t == nil {
		return reflect.Invalid
	}
	return t.Kind()
}

// buildSchema reflects T, nesting it under field when one is given and
// inlining it at the root otherwise.
func buildSchema[T any](field string) (json.RawMessage, error) {
	t := reflect.TypeFor[T]()

	r := outputReflector()
	// Inlining keeps an envelope-less root an object rather than a bare $ref.
	r.ExpandedStruct = field == ""

	schema := r.ReflectFromType(t)
	if schema == nil {
		return nil, fmt.Errorf("output schema: reflecting %s produced no schema", t)
	}

	// $defs and $schema are only meaningful at the document root, and the
	// payload's $ref pointers are rooted at "#", so the definitions move up.
	defs := schema.Definitions
	schema.Definitions, schema.Version, schema.ID = nil, "", ""

	payload, err := relaxedTree(schema)
	if err != nil {
		return nil, fmt.Errorf("output schema: %s: %w", t, err)
	}

	root, inlined := payload.(map[string]any)
	switch {
	case field != "":
		root = map[string]any{
			"properties": map[string]any{field: payload},
			"required":   []string{field},
		}
	case !inlined:
		return nil, fmt.Errorf("output schema: %s did not reflect to an object schema", t)
	}
	// MCP requires an object root, which relaxedTree would otherwise widen to
	// admit null as well.
	root["type"] = "object"

	if len(defs) > 0 {
		defsTree, err := relaxedTree(defs)
		if err != nil {
			return nil, fmt.Errorf("output schema: $defs for %s: %w", t, err)
		}
		root["$defs"] = defsTree
	}

	return json.Marshal(root)
}

// outputReflector configures invopop for godo types:
//   - $defs on (handles recursion; smaller than full inline)
//   - required only from explicit jsonschema tags (not omitempty)
//   - additionalProperties allowed (API may add fields)
//
// Prefer this over mcp.WithOutputSchema, which fails on Timestamp/recursion.
func outputReflector() *jsonschema.Reflector {
	return &jsonschema.Reflector{
		RequiredFromJSONSchemaTags: true,
		AllowAdditionalProperties:  true,
		Mapper:                     schemaForCustomMarshaler,
	}
}

// schemaForCustomMarshaler probes json.Marshaler types for their real JSON
// kind (e.g. godo.Timestamp → date-time string). Returns nil for structural fallback.
func schemaForCustomMarshaler(t reflect.Type) *jsonschema.Schema {
	if t == reflect.TypeFor[time.Time]() {
		return nil // invopop already models time.Time correctly.
	}
	if !implementsJSONMarshaler(t) {
		return nil
	}

	encoded, err := json.Marshal(reflect.New(t).Interface())
	if err != nil {
		return nil
	}
	encoded = bytes.TrimSpace(encoded)
	if len(encoded) == 0 {
		return nil
	}

	switch encoded[0] {
	case '"':
		s := &jsonschema.Schema{Type: "string"}
		var probe string
		if json.Unmarshal(encoded, &probe) == nil {
			if _, err := time.Parse(time.RFC3339, probe); err == nil {
				s.Format = "date-time"
			}
		}
		return s
	case '[':
		return &jsonschema.Schema{Type: "array"}
	case 't', 'f':
		return &jsonschema.Schema{Type: "boolean"}
	case '-', '0', '1', '2', '3', '4', '5', '6', '7', '8', '9':
		return &jsonschema.Schema{Type: "number"}
	default:
		return nil
	}
}

func implementsJSONMarshaler(t reflect.Type) bool {
	marshaler := reflect.TypeFor[json.Marshaler]()
	return t.Implements(marshaler) || reflect.PointerTo(t).Implements(marshaler)
}

// relaxedTree decodes v into a schema tree and widens each declared type to
// also admit null, matching nilable API fields that serialise as null.
func relaxedTree(v any) (any, error) {
	raw, err := json.Marshal(v)
	if err != nil {
		return nil, err
	}
	var tree any
	if err := json.Unmarshal(raw, &tree); err != nil {
		return nil, err
	}
	return allowNull(tree), nil
}

// allowNull rewrites "type": "X" → "type": ["X", "null"].
func allowNull(node any) any {
	switch v := node.(type) {
	case map[string]any:
		for key, child := range v {
			// Property named "type" is a nested schema, not a type keyword.
			if key == "type" {
				if name, ok := child.(string); ok {
					if name != "null" {
						v[key] = []any{name, "null"}
					}
					continue
				}
			}
			v[key] = allowNull(child)
		}
		return v
	case []any:
		for i := range v {
			v[i] = allowNull(v[i])
		}
		return v
	default:
		return node
	}
}
