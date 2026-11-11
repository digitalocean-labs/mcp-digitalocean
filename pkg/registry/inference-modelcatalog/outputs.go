package inferencemodelcatalog

import (
	"mcp-digitalocean/pkg/registry/common"
)

// modelSearchResult is the shape inference-model-catalog-search has always
// returned: the matching model UUIDs alongside the query they were found with
// and their count. It was declared inside the handler; hoisting it to a
// package-level type lets one type drive both the declared schema and the
// emitted payload.
type modelSearchResult struct {
	ModelUUIDs  []string `json:"model_uuids"`
	SearchQuery string   `json:"search_query"`
	Count       int      `json:"count"`
}

// Output contracts for this package's tools; see the marketplace package for
// the convention.
//
// searchOut publishes a payload that already pairs a collection with its query
// metadata, so it needs no envelope. modelCardOut publishes a bare model card
// resource, so it takes the "model" envelope to satisfy MCP's object-root
// requirement.
//
// This package also registers MCP prompts, which have no output schema of
// their own; they are left untouched.
var (
	searchOut    = common.NewObjectOutput[modelSearchResult]()
	modelCardOut = common.NewOutput[*ModelMetadata]("model")
)
