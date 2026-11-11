package genaiinferencerouter

import (
	"github.com/digitalocean/godo"

	"mcp-digitalocean/pkg/registry/common"
)

// metaView is the trimmed pagination block these tools have always echoed from
// the API response: page/pages/total only, dropping the rest of godo.Meta.
type metaView struct {
	Page  int `json:"page"`
	Pages int `json:"pages"`
	Total int `json:"total"`
}

// routerEnvelope is the {"model_router": …} wrapper create/get/update have
// always emitted, mirroring the API response body. It was an inline
// map[string]any literal; naming it lets the schema describe the router
// inside instead of a bare "object".
type routerEnvelope struct {
	ModelRouter *godo.InferenceRouter `json:"model_router"`
}

// routerList is the shape genai-inference-router-list has always returned: a
// page of router summaries alongside its pagination metadata. It was an
// anonymous struct built inside the handler; naming it lets one type drive
// both the declared schema and the emitted payload.
type routerList struct {
	ModelRouters []*godo.InferenceRouterSummary `json:"model_routers"`
	Meta         *metaView                      `json:"meta,omitempty"`
	Links        *godo.Links                    `json:"links,omitempty"`
}

// taskPresetList is the same pairing for genai-inference-router-task-presets.
type taskPresetList struct {
	Tasks []*godo.InferenceRouterTaskPreset `json:"tasks"`
	Meta  *metaView                         `json:"meta,omitempty"`
	Links *godo.Links                       `json:"links,omitempty"`
}

// Output contracts for this package's tools; see the marketplace package for
// the convention. Every payload here is already a JSON object, so none of them
// take an Output envelope: create/get/update carry their own "model_router"
// wrapper in the payload type, which keeps the text content byte-for-byte what
// it was rather than moving the wrapper into structuredContent alone.
//
// Every tool in this package returns JSON, so none stay text-only. Delete has
// a defensive fallback for a nil godo response; it now emits the same
// godo.InferenceRouterDeleteResponse the success path does, so both branches
// share one schema instead of the fallback escaping as an unschema'd
// {"uuid": …} literal.
var (
	routerOut         = common.NewObjectOutput[routerEnvelope]()
	routerListOut     = common.NewObjectOutput[routerList]()
	taskPresetListOut = common.NewObjectOutput[taskPresetList]()
	routerDeleteOut   = common.NewObjectOutput[*godo.InferenceRouterDeleteResponse]()
)
