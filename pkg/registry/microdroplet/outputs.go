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

	// execResponse matches POST /v2/microvms/{id}/exec (flat body, not wrapped).
	execResponse struct {
		Stdout         string `json:"stdout"`
		Stderr         string `json:"stderr"`
		OutputEncoding string `json:"output_encoding,omitempty"`
		ExitCode       int32  `json:"exit_code"`
		Truncated      bool   `json:"truncated,omitempty"`
	}

	// fileReadResponse is this tool's own shape: GET …/files streams raw bytes
	// with the digest in X-Checksum-Sha256, so there is no JSON body to pass through.
	fileReadResponse struct {
		Path      string `json:"path"`
		SizeBytes int    `json:"size_bytes"`
		SHA256    string `json:"sha256"`
		Encoding  string `json:"encoding"`
		Content   string `json:"content"`
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
	execOut            = common.NewObjectOutput[execResponse]()
	fileInfoOut        = common.NewObjectOutput[*godo.MicroVMFileInfo]()
	fileReadOut        = common.NewObjectOutput[fileReadResponse]()
)
