package marketplace

import (
	"github.com/digitalocean/godo"

	"mcp-digitalocean/pkg/registry/common"
)

// oneClickList is the shape marketplace-1-click-list has always returned: the
// available apps, alongside the type they were requested for. It was an
// inline map literal; naming it lets one type drive both the declared schema
// and the emitted payload.
type oneClickList struct {
	Apps []*godo.OneClick `json:"apps"`
	Type string           `json:"type"`
}

// Output contracts for this package's tools; see the account package for the
// convention. Both payloads are already JSON objects, so neither is wrapped in
// an envelope.
var (
	oneClickListOut = common.NewObjectOutput[oneClickList]()
	installOut      = common.NewObjectOutput[*godo.InstallKubernetesAppsResponse]()
)
