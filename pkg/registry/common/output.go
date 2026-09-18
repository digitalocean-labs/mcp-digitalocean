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

// Output binds a tool's structured payload type T to the single field name it
// is published under inside structuredContent. One Output value is the source
// of truth for both halves of the MCP output contract — Schema() declares it
// at registration time and Result() emits it from the handler — so the field
// name and the payload type cannot drift apart.
//
// MCP requires structuredContent to be a JSON object, but most DigitalOcean
// list tools return a JSON array. Output is therefore always an envelope: the
// payload is nested under field rather than returned at the top level.
//
//	var actionsOut = common.NewOutput[[]godo.Action]("actions")
//
//	// registration
//	mcp.NewTool("action-list", actionsOut.Schema(), ...)
//
//	// handler
//	return actionsOut.Result(actions)
//
// Reflection runs at most once per Output, on first Schema() call.
type Output[T any] struct {
	field string
	once  sync.Once
	raw   json.RawMessage
}

// NewOutput returns an Output that publishes T under the given field name.
// field is the key clients read from structuredContent, so it should be the
// plural resource name for collections ("droplets") and the singular name for
// a single resource ("droplet").
func NewOutput[T any](field string) *Output[T] {
	return &Output[T]{field: field}
}

// Field is the structuredContent key this Output publishes under.
func (o *Output[T]) Field() string { return o.field }

// Schema returns the tool option that declares this Output's envelope as the
// tool's outputSchema.
func (o *Output[T]) Schema() mcp.ToolOption {
	return mcp.WithRawOutputSchema(o.RawSchema())
}

// RawSchema is the generated envelope schema. It is exported so tests can
// assert that the declared schema and the emitted structuredContent agree.
func (o *Output[T]) RawSchema() json.RawMessage {
	o.once.Do(func() {
		raw, err := buildEnvelopeSchema[T](o.field)
		if err != nil {
			// A tool with no output schema still works: mcp-go omits the
			// field and clients fall back to the text content. Failing the
			// whole server over one unreflectable payload type would be
			// worse, and TestOutputSchemasAreGenerated catches it in CI.
			return
		}
		o.raw = raw
	})
	return o.raw
}

// Result builds the tool result for payload. The text content is the
// pretty-printed payload exactly as tools returned it before structured
// output existed, which keeps existing consumers working; structuredContent
// carries the same data wrapped in the envelope.
func (o *Output[T]) Result(payload T) (*mcp.CallToolResult, error) {
	text, err := json.MarshalIndent(payload, "", "  ")
	if err != nil {
		return nil, fmt.Errorf("marshal error: %w", err)
	}

	res := mcp.NewToolResultText(string(text))
	res.StructuredContent = map[string]json.RawMessage{o.field: normalizeNull[T](text)}
	return res, nil
}

// normalizeNull replaces a JSON null payload with the empty value for T's
// kind. A nil Go slice marshals to null, and godo returns nil slices for
// empty collections, so without this an empty list would not satisfy the
// declared "array" schema.
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

// buildEnvelopeSchema reflects T and nests it under field in an object schema.
func buildEnvelopeSchema[T any](field string) (json.RawMessage, error) {
	payload := outputReflector().ReflectFromType(reflect.TypeFor[T]())
	if payload == nil {
		return nil, fmt.Errorf("output schema: reflecting %s produced no schema", reflect.TypeFor[T]())
	}

	// $defs and $schema are only meaningful at the document root, and the
	// payload's $ref pointers are rooted at "#", so the definitions have to
	// move up with them.
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

// envelopeSchema is the object wrapper placed around every reflected payload
// schema. Only one property is ever set, so an unordered map is fine.
type envelopeSchema struct {
	Type        string                     `json:"type"`
	Properties  map[string]json.RawMessage `json:"properties"`
	Required    []string                   `json:"required"`
	Definitions json.RawMessage            `json:"$defs,omitempty"`
}

// ObjectOutput is the counterpart to Output for tools whose payload is
// already a JSON object — typically a hand-written result struct that pairs a
// collection with its pagination metadata:
//
//	type repositoryList struct {
//	    Repositories []*godo.RepositoryV2 `json:"repositories"`
//	    Meta         *godo.Meta           `json:"meta,omitempty"`
//	}
//
// Such a payload already satisfies MCP's requirement that structuredContent
// be an object, so it is published as-is. Wrapping it in an Output envelope
// would nest it under a redundant second key.
//
// T must be a struct or a pointer to one.
type ObjectOutput[T any] struct {
	once sync.Once
	raw  json.RawMessage
}

// NewObjectOutput returns an ObjectOutput publishing T as structuredContent.
func NewObjectOutput[T any]() *ObjectOutput[T] {
	return &ObjectOutput[T]{}
}

// Schema returns the tool option that declares T as the tool's outputSchema.
func (o *ObjectOutput[T]) Schema() mcp.ToolOption {
	return mcp.WithRawOutputSchema(o.RawSchema())
}

// RawSchema is the generated schema for T. See Output.RawSchema.
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

// Result builds the tool result for payload, using it as structuredContent
// directly. As with Output.Result, the text content is unchanged from the
// pre-structured-output behaviour.
func (o *ObjectOutput[T]) Result(payload T) (*mcp.CallToolResult, error) {
	text, err := json.MarshalIndent(payload, "", "  ")
	if err != nil {
		return nil, fmt.Errorf("marshal error: %w", err)
	}

	structured := json.RawMessage(text)
	if string(text) == "null" {
		// structuredContent has to be an object even when the payload is a
		// nil pointer.
		structured = json.RawMessage("{}")
	}

	res := mcp.NewToolResultText(string(text))
	res.StructuredContent = structured
	return res, nil
}

// buildObjectSchema reflects T as the schema root.
//
// ExpandedStruct inlines the root type's properties instead of emitting a
// top-level $ref, because MCP requires the output schema root to be an object
// and a bare $ref is not one.
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

	// marshalRelaxed widens every declared type to admit null, which is
	// wrong for the root: MCP requires it to be exactly "object".
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

// outputReflector builds the reflector used for every output schema.
//
// $defs/$ref are left enabled (the default) because that is what lets
// reflection terminate on godo's self-referential types such as
// godo.DeploymentProgressStep, and it keeps types that recur across a
// response from being inlined once per occurrence.
//
// RequiredFromJSONSchemaTags is on so that "required" is driven by explicit
// jsonschema tags rather than inferred from the absence of `omitempty`. godo
// does not set those tags, and inferring required from omitempty produces
// claims the API does not honour — godo.Droplet.VolumeIDs has no omitempty,
// so it would be marked required while actually serialising to null.
//
// AllowAdditionalProperties is on so the schemas do not assert
// additionalProperties:false, which would make a client reject responses as
// soon as the DigitalOcean API grows a field ahead of the vendored godo.
//
// mcp.WithOutputSchema is deliberately unused: its reflector errors out on
// godo.Timestamp and on recursive types, and it reports failure by printing
// to stderr and leaving the schema unset.
func outputReflector() *jsonschema.Reflector {
	return &jsonschema.Reflector{
		RequiredFromJSONSchemaTags: true,
		AllowAdditionalProperties:  true,
		Mapper:                     schemaForCustomMarshaler,
	}
}

// schemaForCustomMarshaler describes types whose JSON shape is decided by a
// custom MarshalJSON rather than by their fields. Reflecting those types
// structurally yields a schema the payload can never satisfy — godo.Timestamp
// embeds time.Time and reflects to an empty object, but serialises to a
// string; godo.KubernetesMaintenancePolicyDay is an integer that serialises to
// a weekday name.
//
// The shape is probed rather than hard-coded per type so that godo gaining
// another custom marshaler does not silently reintroduce the mismatch.
// Returning nil falls back to normal structural reflection.
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
		// Objects and nulls are described well enough by structural
		// reflection.
		return nil
	}
}

func implementsJSONMarshaler(t reflect.Type) bool {
	marshaler := reflect.TypeFor[json.Marshaler]()
	return t.Implements(marshaler) || reflect.PointerTo(t).Implements(marshaler)
}

// marshalRelaxed serialises a reflected schema and then widens every declared
// type to also admit null.
//
// Reflection describes the Go type, not what the DigitalOcean API emits. Any
// nilable field without `omitempty` — a pointer, slice or map — serialises to
// null when unset, which a bare "type": "array" or a $ref to an "object"
// would reject. Widening the types keeps the schema something responses
// actually satisfy, which matters because the spec has clients validate
// structuredContent against it.
func marshalRelaxed(v any) (json.RawMessage, error) {
	tree, err := relaxedTree(v)
	if err != nil {
		return nil, err
	}
	return json.Marshal(tree)
}

// relaxedTree is marshalRelaxed's decoded form, for callers that need to
// patch the tree before serialising it.
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

// allowNull rewrites every `"type": "X"` in a schema tree into
// `"type": ["X", "null"]`.
func allowNull(node any) any {
	switch v := node.(type) {
	case map[string]any:
		for key, child := range v {
			// A property may itself be named "type" (godo.Action.Type), in
			// which case the value is a nested schema object, not a type
			// name; the string assertion is what tells the two apart.
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
