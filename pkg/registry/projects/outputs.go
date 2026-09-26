package projects

import (
	"github.com/digitalocean/godo"

	"mcp-digitalocean/pkg/registry/common"
)

// Output contracts for this package's tools. Each var is used twice — Schema()
// at registration, Result() in the handler — so the declared outputSchema and
// the emitted structuredContent cannot disagree. Collections go under the
// plural resource name, single resources under the singular one.
var (
	projectOut  = common.NewOutput[*godo.Project]("project")
	projectsOut = common.NewOutput[[]godo.Project]("projects")
)
