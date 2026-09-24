package genaicustommodels

import (
	"mcp-digitalocean/pkg/registry/common"
)

// Output contracts for this package's tools; see the marketplace package for
// the convention.
//
// customModelOut publishes a bare resource, shared by the get and
// update-metadata tools because both return the same custom model shape, so it
// needs the "model" envelope to satisfy MCP's object-root requirement — the
// same key the API itself uses. importOut and deleteOut publish payloads that
// are already JSON objects, so neither is wrapped in an envelope.
//
// genai-models-unified-search and genai-custom-models-list stay text-only:
// both return markdown tables rather than JSON, so there is no payload for a
// schema to describe.
var (
	customModelOut = common.NewOutput[*CustomModel]("model")
	importOut      = common.NewObjectOutput[*ImportCustomModelOutput]()
	deleteOut      = common.NewObjectOutput[*DeleteCustomModelOutput]()
)
