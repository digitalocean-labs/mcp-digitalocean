package dedicatedinference

import (
	"github.com/digitalocean/godo"

	"mcp-digitalocean/pkg/registry/common"
)

// createResponse is the shape dedicated-inference-create has always returned:
// the new instance plus the initial auth token, which the API only ever hands
// back once. It was declared inside the handler; hoisting it to a package-level
// type lets one type drive both the declared schema and the emitted payload.
type createResponse struct {
	DedicatedInference *godo.DedicatedInference      `json:"dedicated_inference"`
	Token              *godo.DedicatedInferenceToken `json:"token,omitempty"`
}

// listResponse pairs a page of instances with the pagination metadata the API
// returned alongside it.
type listResponse struct {
	Items []godo.DedicatedInferenceListItem `json:"items"`
	Meta  *godo.Meta                        `json:"meta,omitempty"`
}

// Output contracts for this package's tools; see the account package for the
// convention.
//
// createOut and listOut publish payloads that are already JSON objects, so
// neither is wrapped in an envelope. dedicatedInferenceOut publishes a bare
// resource, shared by the get and update tools because they return the same
// shape, so it needs the "dedicated_inference" envelope to satisfy MCP's
// object-root requirement — the same key the API itself uses.
//
// dedicated-inference-delete stays text-only: its result is a fixed success
// message, not a resource, so an output schema would describe nothing.
var (
	createOut             = common.NewObjectOutput[createResponse]()
	listOut               = common.NewObjectOutput[listResponse]()
	dedicatedInferenceOut = common.NewOutput[*godo.DedicatedInference]("dedicated_inference")
)
