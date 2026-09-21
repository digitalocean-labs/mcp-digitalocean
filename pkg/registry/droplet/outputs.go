package droplet

import (
	"github.com/digitalocean/godo"

	"mcp-digitalocean/pkg/registry/common"
)

// Output contracts for this package's tools. Droplet and image actions all
// resolve to an action the caller polls, so the single-target tools share
// actionOut and the by-tag variants share actionsOut. droplet-delete and
// image-delete stay text-only: they return a confirmation, not a payload.
//
// droplet-list, image-list, and size-list return a curated map[string]any
// subset built in the handler. The schema is therefore an array of objects,
// not a field-level description of godo.Droplet / Image / Size.
var (
	dropletOut      = common.NewOutput[*godo.Droplet]("droplet")
	dropletListOut  = common.NewOutput[[]map[string]any]("droplets")
	neighborsOut    = common.NewOutput[[]godo.Droplet]("droplets")
	kernelsOut      = common.NewOutput[[]godo.Kernel]("kernels")
	backupPolicyOut = common.NewOutput[*godo.DropletBackupPolicy]("backup_policy")

	imageOut     = common.NewOutput[*godo.Image]("image")
	imageListOut = common.NewOutput[[]map[string]any]("images")
	sizeListOut  = common.NewOutput[[]map[string]any]("sizes")

	actionOut  = common.NewOutput[*godo.Action]("action")
	actionsOut = common.NewOutput[[]godo.Action]("actions")
)
