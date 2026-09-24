package spaces

import (
	"github.com/digitalocean/godo"

	"mcp-digitalocean/pkg/registry/common"
)

// spacesKeyList is the shape spaces-key-list has always returned: the page of
// keys alongside the pagination metadata from the same response. It was an
// anonymous struct declared inside the handler; naming it lets one type drive
// both the declared schema and the emitted payload.
type spacesKeyList struct {
	Keys []*godo.SpacesKey `json:"keys"`
	Meta *godo.Meta        `json:"meta,omitempty"`
}

// Output contracts for this package's tools; see the marketplace package for
// the convention.
//
// keyOut and cdnOut are each shared by every tool returning that resource
// (create/update/get for keys, create/get for CDNs) because all of them emit
// the same bare godo resource, which an envelope lifts into an object.
// cdnListOut wraps an array, which MCP cannot accept at the root.
// spacesKeyListOut is published without an envelope: the handler already
// hand-wraps its payload in a "keys"/"meta" object, and enveloping it again
// would both nest redundantly and strip those keys from the text content.
//
// spaces-key-delete, spaces-cdn-delete and spaces-cdn-flush-cache stay
// text-only: each returns a fixed success message rather than a resource, so
// an output schema would describe nothing.
var (
	keyOut           = common.NewOutput[*godo.SpacesKey]("key")
	spacesKeyListOut = common.NewObjectOutput[spacesKeyList]()
	cdnOut           = common.NewOutput[*godo.CDN]("cdn")
	cdnListOut       = common.NewOutput[[]godo.CDN]("cdns")
)
