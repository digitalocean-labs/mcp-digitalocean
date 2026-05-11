package openapi

import (
	_ "embed"
	"fmt"
	"regexp"
	"sort"
	"strconv"
	"strings"
	"sync"

	"github.com/getkin/kin-openapi/openapi3"
)

//go:embed spec/DigitalOcean-public.v2.yaml
var embeddedSpec []byte

// OpenAPIService is the read-only surface of the embedded spec used by tools
// and tests (search and operation lookup).
type OpenAPIService interface {
	GetOperation(id string) (*Operation, error)
	SearchOperations(query, tag string, limit int) ([]*OperationSummary, error)
}

// OperationSummary is one ranked hit from SearchOperations (openapi-search).
type OperationSummary struct {
	OperationID string   `json:"operationId"`
	Method      string   `json:"method"`
	Path        string   `json:"path"`
	Summary     string   `json:"summary,omitempty"`
	Tags        []string `json:"tags,omitempty"`
}

// ParameterDetail is a flattened parameter row for display in openapi-get-operation.
type ParameterDetail struct {
	Name        string `json:"name"`
	In          string `json:"in"` // path, query, header, or cookie (cookie unsupported in execute)
	Required    bool   `json:"required"`
	Description string `json:"description,omitempty"`
	Schema      string `json:"schema,omitempty"` // short type/format summary for the agent
}

// Operation is a stable, tool-friendly view of one OpenAPI operation. The
// kin-openapi types are kept unexported; same-package code uses them for
// validation in openapi-execute.
type Operation struct {
	OperationID string            `json:"operationId"`
	Method      string            `json:"method"`
	Path        string            `json:"path"`
	Summary     string            `json:"summary,omitempty"`
	Description string            `json:"description,omitempty"`
	Tags        []string          `json:"tags,omitempty"`
	Parameters  []ParameterDetail `json:"parameters,omitempty"`
	// RequestBody is a rendered text summary (content types and schema outline), not raw YAML.
	RequestBody string            `json:"requestBody,omitempty"`
	Responses   map[string]string `json:"responses,omitempty"` // HTTP status or name (e.g. default) → description
}

// indexedOp holds resolved path/method plus pointers needed for validation and routing.
type indexedOp struct {
	method   string
	path     string
	op       *openapi3.Operation
	pathItem *openapi3.PathItem
}

// OpenAPIClient parses the embedded YAML once (sync.Once), resolves references
// via kin-openapi, and builds indexes for lookup and search.
type OpenAPIClient struct {
	once sync.Once
	doc  *openapi3.T
	byID map[string]*indexedOp // operationId → operation
	idx  []*OperationSummary   // sorted by operationId for stable search
	err  error
}

// NewOpenAPIClient returns a client; the spec is not loaded until the first method call.
func NewOpenAPIClient() *OpenAPIClient {
	return &OpenAPIClient{}
}

// ensure runs a one-time load of embeddedSpec into doc, byID, and idx.
func (c *OpenAPIClient) ensure() error {
	c.once.Do(func() {
		c.err = c.load()
	})
	return c.err
}

// load parses embeddedSpec with openapi3.Loader (ResolveRefsIn) and builds indexes.
func (c *OpenAPIClient) load() error {
	loader := openapi3.NewLoader()
	doc, err := loader.LoadFromData(embeddedSpec)
	if err != nil {
		return fmt.Errorf("load openapi spec: %w", err)
	}
	if len(doc.Servers) == 0 {
		return fmt.Errorf("openapi spec has no servers")
	}

	c.doc = doc
	c.byID = make(map[string]*indexedOp)
	c.idx = nil

	paths := doc.Paths
	if paths == nil {
		return fmt.Errorf("openapi spec has no paths")
	}

	pathMap := paths.Map()
	pathKeys := make([]string, 0, len(pathMap))
	for p := range pathMap {
		pathKeys = append(pathKeys, p)
	}
	sort.Strings(pathKeys)

	for _, path := range pathKeys {
		pathItem := pathMap[path]
		if pathItem == nil {
			continue
		}
		for method, op := range pathItem.Operations() {
			if op == nil {
				continue
			}
			method = strings.ToUpper(method)
			id := op.OperationID
			if id == "" {
				continue
			}
			idx := &indexedOp{
				method:   method,
				path:     path,
				op:       op,
				pathItem: pathItem,
			}
			if _, exists := c.byID[id]; exists {
				return fmt.Errorf("duplicate operationId %q", id)
			}
			c.byID[id] = idx

			tags := append([]string(nil), op.Tags...)
			sort.Strings(tags)
			c.idx = append(c.idx, &OperationSummary{
				OperationID: id,
				Method:      method,
				Path:        path,
				Summary:     op.Summary,
				Tags:        tags,
			})
		}
	}

	sort.Slice(c.idx, func(i, j int) bool {
		return c.idx[i].OperationID < c.idx[j].OperationID
	})

	return nil
}

// GetOperation returns a text-oriented projection for the given OpenAPI operationId.
func (c *OpenAPIClient) GetOperation(id string) (*Operation, error) {
	if err := c.ensure(); err != nil {
		return nil, err
	}
	idx, ok := c.byID[id]
	if !ok {
		return nil, fmt.Errorf("unknown operationId %q", id)
	}
	return projectionFromIndexed(idx), nil
}

// projectionFromIndexed builds an Operation for MCP text output.
func projectionFromIndexed(idx *indexedOp) *Operation {
	op := idx.op
	resp := make(map[string]string)
	if op.Responses != nil {
		for status, ref := range op.Responses.Map() {
			if ref != nil && ref.Value != nil && ref.Value.Description != nil {
				resp[status] = *ref.Value.Description
			} else if ref != nil && ref.Ref != "" {
				resp[status] = ref.Ref
			} else {
				resp[status] = ""
			}
		}
	}

	params := mergeParameters(idx.pathItem, op)
	var details []ParameterDetail
	for _, p := range params {
		if p == nil {
			continue
		}
		details = append(details, ParameterDetail{
			Name:        p.Name,
			In:          string(p.In),
			Required:    p.Required,
			Description: p.Description,
			Schema:      briefSchemaRef(p.Schema),
		})
	}

	bodyDesc := ""
	if op.RequestBody != nil && op.RequestBody.Value != nil {
		bodyDesc = describeRequestBody(op.RequestBody.Value)
	}

	return &Operation{
		OperationID: op.OperationID,
		Method:      idx.method,
		Path:        idx.path,
		Summary:     op.Summary,
		Description: op.Description,
		Tags:        append([]string(nil), op.Tags...),
		Parameters:  details,
		RequestBody: bodyDesc,
		Responses:   resp,
	}
}

// mergeParameters merges path-item and operation parameters; later entries override
// same (in, name) per OpenAPI rules.
func mergeParameters(pathItem *openapi3.PathItem, op *openapi3.Operation) []*openapi3.Parameter {
	byKey := make(map[string]*openapi3.Parameter)
	var order []string
	add := func(pref *openapi3.ParameterRef) {
		if pref == nil || pref.Value == nil {
			return
		}
		p := pref.Value
		k := string(p.In) + ":" + p.Name
		if _, ok := byKey[k]; !ok {
			order = append(order, k)
		}
		byKey[k] = p
	}
	for _, pref := range pathItem.Parameters {
		add(pref)
	}
	for _, pref := range op.Parameters {
		add(pref)
	}
	out := make([]*openapi3.Parameter, 0, len(order))
	for _, k := range order {
		out = append(out, byKey[k])
	}
	return out
}

func briefSchemaRef(sr *openapi3.SchemaRef) string {
	if sr == nil || sr.Value == nil {
		return ""
	}
	return briefSchema(sr.Value)
}

func briefSchema(s *openapi3.Schema) string {
	if s == nil {
		return ""
	}
	if len(s.Type.Slice()) > 0 {
		types := append([]string(nil), s.Type.Slice()...)
		sort.Strings(types)
		t := strings.Join(types, "|")
		if s.Format != "" {
			t += " (" + s.Format + ")"
		}
		return t
	}
	return "object"
}

func describeRequestBody(rb *openapi3.RequestBody) string {
	var b strings.Builder
	if rb.Description != "" {
		b.WriteString(rb.Description)
		b.WriteByte('\n')
	}
	for ct, media := range rb.Content {
		if media == nil {
			continue
		}
		fmt.Fprintf(&b, "Content-Type: %s\n", ct)
		if media.Schema != nil {
			b.WriteString(schemaRefToText(media.Schema, 0))
		}
		b.WriteByte('\n')
	}
	return strings.TrimSpace(b.String())
}

func schemaRefToText(sr *openapi3.SchemaRef, indent int) string {
	if sr == nil {
		return ""
	}
	if sr.Ref != "" {
		return strings.Repeat("  ", indent) + "$ref: " + sr.Ref + "\n"
	}
	return schemaToText(sr.Value, indent)
}

func schemaToText(s *openapi3.Schema, indent int) string {
	if s == nil {
		return ""
	}
	pad := strings.Repeat("  ", indent)
	var b strings.Builder
	if len(s.Type.Slice()) > 0 {
		fmt.Fprintf(&b, "%stype: %s\n", pad, strings.Join(s.Type.Slice(), "|"))
	}
	if s.Description != "" {
		fmt.Fprintf(&b, "%sdescription: %s\n", pad, s.Description)
	}
	if s.Properties != nil {
		for _, name := range sortedKeys(s.Properties) {
			prop := s.Properties[name]
			fmt.Fprintf(&b, "%s- %s:\n", pad, name)
			b.WriteString(schemaRefToText(prop, indent+1))
		}
	}
	return b.String()
}

func sortedKeys(m openapi3.Schemas) []string {
	keys := make([]string, 0, len(m))
	for k := range m {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	return keys
}

// SearchOperations scores each operation by query terms against operationId,
// summary, path, method, and tags. If tag is non-empty, only operations
// listing that exact tag name are considered.
func (c *OpenAPIClient) SearchOperations(query, tag string, limit int) ([]*OperationSummary, error) {
	if err := c.ensure(); err != nil {
		return nil, err
	}
	if limit <= 0 {
		limit = 10
	}

	tag = strings.TrimSpace(tag)
	q := strings.TrimSpace(query)
	if q == "" {
		return nil, fmt.Errorf("query is required")
	}

	terms := strings.Fields(strings.ToLower(q))
	if len(terms) == 0 {
		return nil, fmt.Errorf("query is required")
	}

	type scored struct {
		entry *OperationSummary
		score int
	}

	wordRegexes := make([]*regexp.Regexp, len(terms))
	for i, term := range terms {
		wordRegexes[i] = regexp.MustCompile(`\b` + regexp.QuoteMeta(term) + `\b`)
	}

	var results []scored
	for _, entry := range c.idx {
		if tag != "" {
			found := false
			for _, t := range entry.Tags {
				if strings.EqualFold(t, tag) {
					found = true
					break
				}
			}
			if !found {
				continue
			}
		}

		opIDLower := strings.ToLower(entry.OperationID)
		sumLower := strings.ToLower(entry.Summary)
		pathLower := strings.ToLower(entry.Path)
		methodLower := strings.ToLower(entry.Method)
		var tagsLower strings.Builder
		for _, t := range entry.Tags {
			tagsLower.WriteString(strings.ToLower(t))
			tagsLower.WriteByte(' ')
		}
		tagsJoined := tagsLower.String()

		score := 0
		for i, term := range terms {
			if strings.Contains(opIDLower, term) {
				score += 10
			}
			if wordRegexes[i].MatchString(opIDLower) {
				score += 5
			}
			if strings.Contains(sumLower, term) {
				score += 6
			}
			if strings.Contains(pathLower, term) {
				score += 4
			}
			if strings.Contains(methodLower, term) {
				score += 3
			}
			if strings.Contains(tagsJoined, term) {
				score += 2
			}
		}
		if score > 0 {
			results = append(results, scored{entry: entry, score: score})
		}
	}

	sort.Slice(results, func(i, j int) bool {
		return results[i].score > results[j].score
	})

	out := make([]*OperationSummary, 0, limit)
	for i := 0; i < len(results) && len(out) < limit; i++ {
		out = append(out, results[i].entry)
	}
	return out, nil
}

// getIndexed returns the internal index entry for openapi execute tools.
func (c *OpenAPIClient) getIndexed(operationID string) (*indexedOp, error) {
	if err := c.ensure(); err != nil {
		return nil, err
	}
	idx, ok := c.byID[operationID]
	if !ok {
		return nil, fmt.Errorf("unknown operationId %q", operationID)
	}
	return idx, nil
}

// document returns the loaded *openapi3.T after ensure.
func (c *OpenAPIClient) document() (*openapi3.T, error) {
	if err := c.ensure(); err != nil {
		return nil, err
	}
	return c.doc, nil
}

// serverBase returns the first server URL from the spec, trimmed, used only
// for request validation (must match the document’s servers entry).
func serverBase(doc *openapi3.T) (string, error) {
	if doc == nil || len(doc.Servers) == 0 {
		return "", fmt.Errorf("spec has no servers")
	}
	return strings.TrimRight(strings.TrimSpace(doc.Servers[0].URL), "/"), nil
}

var pathParamPlaceholder = regexp.MustCompile(`\{([^}]+)}`)

// substitutePath replaces {name} placeholders with values from pathValues.
func substitutePath(pathTemplate string, pathValues map[string]string) (string, error) {
	var missing []string
	out := pathParamPlaceholder.ReplaceAllStringFunc(pathTemplate, func(match string) string {
		name := strings.TrimSuffix(strings.TrimPrefix(match, "{"), "}")
		v, ok := pathValues[name]
		if !ok || v == "" {
			missing = append(missing, name)
			return match
		}
		return v
	})
	if len(missing) > 0 {
		return "", fmt.Errorf("missing path parameter(s): %s", strings.Join(missing, ", "))
	}
	return out, nil
}

// paramToStrings turns JSON-decoded MCP argument values into strings for path,
// query, and header parameters. Supports string, bool, float64 (including
// integers from JSON), and []interface{} for repeated query values.
func paramToStrings(name string, v any) ([]string, error) {
	if v == nil {
		return nil, fmt.Errorf("parameter %q is null", name)
	}
	switch t := v.(type) {
	case string:
		return []string{t}, nil
	case bool:
		return []string{strconv.FormatBool(t)}, nil
	case float64:
		if t == float64(int64(t)) {
			return []string{strconv.FormatInt(int64(t), 10)}, nil
		}
		return []string{strconv.FormatFloat(t, 'f', -1, 64)}, nil
	case []any:
		var out []string
		for _, item := range t {
			subs, err := paramToStrings(name, item)
			if err != nil {
				return nil, err
			}
			out = append(out, subs...)
		}
		return out, nil
	default:
		return nil, fmt.Errorf("parameter %q has unsupported type %T", name, v)
	}
}
