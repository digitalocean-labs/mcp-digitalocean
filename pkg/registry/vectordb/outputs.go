package vectordb

import (
	"time"

	"github.com/digitalocean/godo"

	"mcp-digitalocean/pkg/registry/common"
)

// vectorDBSummary is the per-cluster shape vector-db-list has always returned:
// the identifying and status fields, with the connection endpoints flattened
// out of the nested endpoints object. It was an inline map literal; naming it
// lets one type drive both the declared schema and the emitted payload. A map
// serialises every key it holds, so none of these fields are omitempty.
type vectorDBSummary struct {
	ID           string    `json:"id"`
	Name         string    `json:"name"`
	Region       string    `json:"region"`
	Size         string    `json:"size"`
	Status       string    `json:"status"`
	CreatedAt    time.Time `json:"created_at"`
	Tags         []string  `json:"tags"`
	EndpointHTTP string    `json:"endpoint_http"`
	EndpointGRPC string    `json:"endpoint_grpc"`
}

// newVectorDBSummary projects a cluster onto the list summary, tolerating the
// absent endpoints a cluster still being provisioned reports.
func newVectorDBSummary(vectorDB godo.VectorDB) vectorDBSummary {
	var httpEndpoint, grpcEndpoint string
	if vectorDB.Endpoints != nil {
		httpEndpoint = vectorDB.Endpoints.HTTP
		grpcEndpoint = vectorDB.Endpoints.GRPC
	}
	return vectorDBSummary{
		ID:           vectorDB.ID,
		Name:         vectorDB.Name,
		Region:       vectorDB.Region,
		Size:         vectorDB.Size,
		Status:       vectorDB.Status,
		CreatedAt:    vectorDB.CreatedAt,
		Tags:         vectorDB.Tags,
		EndpointHTTP: httpEndpoint,
		EndpointGRPC: grpcEndpoint,
	}
}

// Output contracts for this package's tools; see the marketplace package for
// the convention.
//
// Every payload here is either a bare resource or an array, so each one needs
// an envelope to satisfy MCP's object-root requirement. vectorDBOut is shared
// by the create, get and resize tools because all three return the same
// cluster shape, under the same "vector_db" key the API itself uses.
//
// vector-db-delete stays text-only: its result is a fixed success message, not
// a resource, so an output schema would describe nothing.
var (
	vectorDBOut     = common.NewOutput[*godo.VectorDB]("vector_db")
	vectorDBListOut = common.NewOutput[[]vectorDBSummary]("vector_dbs")
	credentialsOut  = common.NewOutput[*godo.VectorDBAdminCredentials]("credentials")
)
