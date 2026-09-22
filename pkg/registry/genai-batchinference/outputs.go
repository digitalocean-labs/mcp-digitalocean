package genaibatchinference

import (
	"github.com/digitalocean/godo"

	"mcp-digitalocean/pkg/registry/common"
)

// Output contracts for this package's tools; see the marketplace package for
// the convention. Every batch inference payload is already a JSON object, so
// none of them need an envelope: the list response pairs its edges with
// page_info, and the rest are single resources.
//
// genai-batch-inference-upload-file stays text-only: it PUTs content to a
// presigned URL and reports success as a sentence, with no JSON body to
// describe.
var (
	fileUploadOut = common.NewObjectOutput[*godo.CreateBatchFileResponse]()
	batchOut      = common.NewObjectOutput[*godo.Batch]()
	resultsOut    = common.NewObjectOutput[*godo.BatchResultsResponse]()
	batchListOut  = common.NewObjectOutput[*godo.ListBatchesResponse]()
)
