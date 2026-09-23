package doks

import (
	"github.com/digitalocean/godo"

	"mcp-digitalocean/pkg/registry/common"
)

// clusterCredentials is the shape doks-get-credentials has always returned:
// the API server endpoint plus the client material needed to reach it, with
// the certificate blobs rendered as strings rather than godo's []byte. It was
// an anonymous struct declared inside the handler; naming it lets one type
// drive both the declared schema and the emitted payload.
type clusterCredentials struct {
	Server                   string `json:"server"`
	CertificateAuthorityData string `json:"certificate_authority_data"`
	ClientCertificateData    string `json:"client_certificate_data"`
	ClientKeyData            string `json:"client_key_data"`
	Token                    string `json:"token"`
	ExpiresAt                string `json:"expires_at"`
}

// newClusterCredentials projects godo's credentials onto the returned shape.
func newClusterCredentials(credentials *godo.KubernetesClusterCredentials) clusterCredentials {
	return clusterCredentials{
		Server:                   credentials.Server,
		CertificateAuthorityData: string(credentials.CertificateAuthorityData),
		ClientCertificateData:    string(credentials.ClientCertificateData),
		ClientKeyData:            string(credentials.ClientKeyData),
		Token:                    credentials.Token,
		ExpiresAt:                credentials.ExpiresAt.String(),
	}
}

// Output contracts for this package's tools; see the marketplace package for
// the convention.
//
// clusterOut, clusterListOut, upgradesOut, nodePoolOut and nodePoolListOut all
// publish a bare resource or a bare array, so each one carries an envelope to
// satisfy MCP's object-root requirement. clusterOut is shared by the get,
// create and update tools, and nodePoolOut by the node pool get, create and
// update tools, because each group returns the same resource shape.
//
// credentialsOut and optionsOut publish objects that are already keyed — the
// credential fields and the versions/regions/sizes lists respectively — so an
// envelope would only add a redundant second key.
//
// These tools stay text-only:
//   - doks-delete-cluster, doks-upgrade-cluster, doks-delete-nodepool,
//     doks-delete-node and doks-recycle-nodes each return a fixed confirmation
//     message rather than a resource, so an output schema would describe
//     nothing.
//   - doks-get-kubeconfig returns the kubeconfig document verbatim, which is
//     YAML, not JSON, so there is no JSON payload to describe.
var (
	clusterOut      = common.NewOutput[*godo.KubernetesCluster]("cluster")
	clusterListOut  = common.NewOutput[[]*godo.KubernetesCluster]("clusters")
	upgradesOut     = common.NewOutput[[]*godo.KubernetesVersion]("upgrades")
	credentialsOut  = common.NewObjectOutput[clusterCredentials]()
	nodePoolOut     = common.NewOutput[*godo.KubernetesNodePool]("node_pool")
	nodePoolListOut = common.NewOutput[[]*godo.KubernetesNodePool]("node_pools")
	optionsOut      = common.NewObjectOutput[*godo.KubernetesOptions]()
)
