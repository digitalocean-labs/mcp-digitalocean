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
// unifiedSearchOut and modelRowsOut back the two table-rendering tools. Their
// text half stays the markdown the client is told to display verbatim, so both
// are emitted with ResultWithText: the rows behind the table are published as
// structuredContent so callers do not have to parse the table back apart.
// UnifiedSearchResponse is already an object, while the list tool publishes a
// bare array and so takes the "models" envelope.
var (
	customModelOut   = common.NewOutput[*CustomModel]("model")
	importOut        = common.NewObjectOutput[*ImportCustomModelOutput]()
	deleteOut        = common.NewObjectOutput[*DeleteCustomModelOutput]()
	unifiedSearchOut = common.NewObjectOutput[UnifiedSearchResponse]()
	modelRowsOut     = common.NewOutput[[]CustomSearchRow]("models")
)
