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

// Output publishes T under a named field in structuredContent.
// Schema() and Result() share that field so they cannot drift.
//
// Lists are wrapped ({"actions": [...]}) because MCP expects an object root.
//
//	var actionsOut = common.NewOutput[[]godo.Action]("actions")
//	mcp.NewTool("action-list", actionsOut.Schema(), ...)
//	return actionsOut.Result(actions)
type Output[T any] struct {
	field string
	once  sync.Once
	raw   json.RawMessage
}

// NewOutput publishes T under field ("droplet", "droplets", …).
func NewOutput[T any](field string) *Output[T] {
	return &Output[T]{field: field}
}

// Field is the structuredContent key.
func (o *Output[T]) Field() string { return o.field }

// Schema declares this Output's envelope as the tool's outputSchema.
func (o *Output[T]) Schema() mcp.ToolOption {
	return mcp.WithRawOutputSchema(o.RawSchema())
}

// RawSchema returns the generated envelope schema (lazily reflected once).
func (o *Output[T]) RawSchema() json.RawMessage {
	o.once.Do(func() {
		raw, err := buildEnvelopeSchema[T](o.field)
		if err != nil {
			// Skip rather than panic; CI guard catches empty schemas.
			return
		}
		o.raw = raw
	})
	return o.raw
}

// Result returns text (unchanged bare payload) plus enveloped structuredContent.
func (o *Output[T]) Result(payload T) (*mcp.CallToolResult, error) {
	text, err := json.MarshalIndent(payload, "", "  ")
	if err != nil {
		return nil, fmt.Errorf("marshal error: %w", err)
	}

	res := mcp.NewToolResultText(string(text))
	res.StructuredContent = map[string]json.RawMessage{o.field: normalizeNull[T](text)}
	return res, nil
}

// normalizeNull maps JSON null to [] / {} so empty collections satisfy array/object schemas.
func normalizeNull[T any](encoded json.RawMessage) json.RawMessage {
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

func buildEnvelopeSchema[T any](field string) (json.RawMessage, error) {
	payload := outputReflector().ReflectFromType(reflect.TypeFor[T]())
	if payload == nil {
		return nil, fmt.Errorf("output schema: reflecting %s produced no schema", reflect.TypeFor[T]())
	}

	// Lift $defs to the envelope root; $ref targets stay "#/$defs/...".
	defs := payload.Definitions
	payload.Definitions = nil
	payload.Version = ""
	payload.ID = ""

	payloadRaw, err := marshalRelaxed(payload)
	if err != nil {
		return nil, fmt.Errorf("output schema: payload schema for %s: %w", reflect.TypeFor[T](), err)
	}

	envelope := envelopeSchema{
		Type:       "object",
		Properties: map[string]json.RawMessage{field: payloadRaw},
		Required:   []string{field},
	}
	if len(defs) > 0 {
		defsRaw, err := marshalRelaxed(defs)
		if err != nil {
			return nil, fmt.Errorf("output schema: $defs for %s: %w", reflect.TypeFor[T](), err)
		}
		envelope.Definitions = defsRaw
	}

	raw, err := json.Marshal(envelope)
	if err != nil {
		return nil, fmt.Errorf("output schema: envelope for %s: %w", reflect.TypeFor[T](), err)
	}
	return raw, nil
}

type envelopeSchema struct {
	Type        string                     `json:"type"`
	Properties  map[string]json.RawMessage `json:"properties"`
	Required    []string                   `json:"required"`
	Definitions json.RawMessage            `json:"$defs,omitempty"`
}

// ObjectOutput publishes an already-object payload as-is (no envelope).
// Use for result structs that already pair data with meta, e.g. {repositories, meta}.
type ObjectOutput[T any] struct {
	once sync.Once
	raw  json.RawMessage
}

// NewObjectOutput publishes T directly as structuredContent.
func NewObjectOutput[T any]() *ObjectOutput[T] {
	return &ObjectOutput[T]{}
}

// Schema declares T as the tool's outputSchema.
func (o *ObjectOutput[T]) Schema() mcp.ToolOption {
	return mcp.WithRawOutputSchema(o.RawSchema())
}

// RawSchema returns the generated schema for T (lazily reflected once).
func (o *ObjectOutput[T]) RawSchema() json.RawMessage {
	o.once.Do(func() {
		raw, err := buildObjectSchema[T]()
		if err != nil {
			return
		}
		o.raw = raw
	})
	return o.raw
}

// Result returns text plus structuredContent set to the payload object.
func (o *ObjectOutput[T]) Result(payload T) (*mcp.CallToolResult, error) {
	text, err := json.MarshalIndent(payload, "", "  ")
	if err != nil {
		return nil, fmt.Errorf("marshal error: %w", err)
	}

	structured := json.RawMessage(text)
	if string(text) == "null" {
		structured = json.RawMessage("{}")
	}

	res := mcp.NewToolResultText(string(text))
	res.StructuredContent = structured
	return res, nil
}

// buildObjectSchema inlines T at the root (ExpandedStruct) so the schema is a
// real object, not a top-level $ref.
func buildObjectSchema[T any]() (json.RawMessage, error) {
	r := outputReflector()
	r.ExpandedStruct = true

	schema := r.ReflectFromType(reflect.TypeFor[T]())
	if schema == nil {
		return nil, fmt.Errorf("output schema: reflecting %s produced no schema", reflect.TypeFor[T]())
	}

	defs := schema.Definitions
	schema.Definitions = nil
	schema.Version = ""
	schema.ID = ""

	tree, err := relaxedTree(schema)
	if err != nil {
		return nil, fmt.Errorf("output schema: root schema for %s: %w", reflect.TypeFor[T](), err)
	}
	root, ok := tree.(map[string]any)
	if !ok {
		return nil, fmt.Errorf("output schema: %s did not reflect to an object schema", reflect.TypeFor[T]())
	}

	// Root must stay exactly "object"; do not null-widen it.
	root["type"] = "object"

	if len(defs) > 0 {
		defsTree, err := relaxedTree(defs)
		if err != nil {
			return nil, fmt.Errorf("output schema: $defs for %s: %w", reflect.TypeFor[T](), err)
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

// marshalRelaxed serialises a schema and widens each type to also admit null,
// matching nilable API fields that serialise as null.
func marshalRelaxed(v any) (json.RawMessage, error) {
	tree, err := relaxedTree(v)
	if err != nil {
		return nil, err
	}
	return json.Marshal(tree)
}

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
