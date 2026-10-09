package microdroplet

import (
	"github.com/digitalocean/godo"

	"mcp-digitalocean/pkg/registry/common"
)

// Response envelopes match public REST /v2/microvms JSON (same keys as godo's
// unexported root types). Handlers pass API bytes through with ResultRaw so
// unknown fields are kept; schemas require nothing and allow additionalProperties.
type (
	microVMResponse struct {
		MicroVM *godo.MicroVM `json:"microvm"`
	}

	microVMsListResponse struct {
		MicroVMs []godo.MicroVM `json:"microvms"`
		Links    *godo.Links    `json:"links,omitempty"`
		Meta     *godo.Meta     `json:"meta,omitempty"`
	}

	checkpointResponse struct {
		Checkpoint *godo.MicroVMCheckpoint `json:"checkpoint"`
	}

	checkpointsListResponse struct {
		Checkpoints []godo.MicroVMCheckpoint `json:"checkpoints"`
		Links       *godo.Links              `json:"links,omitempty"`
		Meta        *godo.Meta               `json:"meta,omitempty"`
	}
)

// Output contracts for this package's tools; see the marketplace package for
// the convention.
//
// microVMOut is shared by create/get/pause/resume (same {"microvm":...} shape).
// checkpointOut is shared by checkpoint create/get.
// Delete tools stay text-only: each returns a fixed success message.
var (
	microVMOut         = common.NewObjectOutput[microVMResponse]()
	microVMsListOut    = common.NewObjectOutput[microVMsListResponse]()
	checkpointOut      = common.NewObjectOutput[checkpointResponse]()
	checkpointsListOut = common.NewObjectOutput[checkpointsListResponse]()
)
