package docs

import (
	"bytes"
	"compress/gzip"
	_ "embed"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"sync"

	"github.com/getkin/kin-openapi/openapi3"
)

// Bundled public DigitalOcean API v2 OpenAPI (JSON), gzip-compressed.
// Regenerate from the openapi repo with:
//
//	npx @redocly/cli bundle specification/DigitalOcean-public.v2.yaml -o /tmp/do.json
//	gzip -9 -c /tmp/do.json > pkg/registry/docs/data/openapi_public_v2.json.gz
//
//go:embed data/openapi_public_v2.json.gz
var embeddedOpenAPIGzip []byte

var (
	openapiOnce sync.Once
	openapiDoc  *openapi3.T
	openapiErr  error
)

func decompressOpenAPISpec() ([]byte, error) {
	r, err := gzip.NewReader(bytes.NewReader(embeddedOpenAPIGzip))
	if err != nil {
		return nil, err
	}
	defer r.Close()
	return io.ReadAll(io.LimitReader(r, 25<<20))
}

func loadEmbeddedOpenAPI() (*openapi3.T, error) {
	openapiOnce.Do(func() {
		raw, err := decompressOpenAPISpec()
		if err != nil {
			openapiErr = err
			return
		}
		loader := openapi3.NewLoader()
		openapiDoc, openapiErr = loader.LoadFromData(raw)
	})
	if openapiErr != nil {
		return nil, openapiErr
	}
	return openapiDoc, nil
}

// OpenAPIOperationJSON returns the JSON encoding of one operation from the embedded public v2 spec.
func OpenAPIOperationJSON(method, path string) ([]byte, error) {
	doc, err := loadEmbeddedOpenAPI()
	if err != nil {
		return nil, err
	}
	if doc.Paths == nil {
		return nil, fmt.Errorf("OpenAPI document has no paths")
	}
	pi := doc.Paths.Value(path)
	if pi == nil {
		return nil, fmt.Errorf("path not found: %s", path)
	}

	var op *openapi3.Operation
	switch strings.ToUpper(strings.TrimSpace(method)) {
	case http.MethodGet:
		op = pi.Get
	case http.MethodPost:
		op = pi.Post
	case http.MethodPut:
		op = pi.Put
	case http.MethodPatch:
		op = pi.Patch
	case http.MethodDelete:
		op = pi.Delete
	default:
		return nil, fmt.Errorf("unsupported HTTP method: %s", method)
	}
	if op == nil {
		return nil, fmt.Errorf("operation %s %s is not defined in the spec", method, path)
	}
	return json.Marshal(op)
}
