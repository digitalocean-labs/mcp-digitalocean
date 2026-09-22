package docr

import (
	"github.com/digitalocean/godo"

	"mcp-digitalocean/pkg/registry/common"
)

// Paginated result shapes for the list tools, previously anonymous structs
// inline in each handler. Naming them lets one type drive both the declared
// schema and the emitted payload.
type (
	repositoryList struct {
		Repositories []*godo.RepositoryV2 `json:"repositories"`
		Meta         *godo.Meta           `json:"meta,omitempty"`
	}

	repositoryTagList struct {
		Tags []*godo.RepositoryTag `json:"tags"`
		Meta *godo.Meta            `json:"meta,omitempty"`
	}

	repositoryManifestList struct {
		Manifests []*godo.RepositoryManifest `json:"manifests"`
		Meta      *godo.Meta                 `json:"meta,omitempty"`
	}

	garbageCollectionList struct {
		GarbageCollections []*godo.GarbageCollection `json:"garbage_collections"`
		Meta               *godo.Meta                `json:"meta,omitempty"`
	}
)

// Output contracts for this package's tools; see the account package for the
// convention. Tools that stay text-only: the delete and validate-name tools
// return a confirmation rather than a payload, and docr-docker-credentials
// returns a Docker-defined config blob with no godo type to describe it.
var (
	registryOut     = common.NewOutput[*godo.Registry]("registry")
	registriesOut   = common.NewOutput[[]*godo.Registry]("registries")
	registryOptsOut = common.NewOutput[*godo.RegistryOptions]("options")
	subscriptionOut = common.NewOutput[*godo.RegistrySubscription]("subscription")
	gcOut           = common.NewOutput[*godo.GarbageCollection]("garbage_collection")

	// These payloads are already objects that pair a collection with its
	// pagination metadata, so they are published without an extra envelope.
	repositoryListOut         = common.NewObjectOutput[repositoryList]()
	repositoryTagListOut      = common.NewObjectOutput[repositoryTagList]()
	repositoryManifestListOut = common.NewObjectOutput[repositoryManifestList]()
	garbageCollectionListOut  = common.NewObjectOutput[garbageCollectionList]()
)
