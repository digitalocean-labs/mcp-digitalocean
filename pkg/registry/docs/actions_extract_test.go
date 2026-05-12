package docs

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestExtractActions_curl(t *testing.T) {
	md := "```bash\ncurl -X GET https://api.digitalocean.com/v2/droplets\n```"
	acts := ExtractActions(md)
	require.NotEmpty(t, acts)
	require.Equal(t, "api", acts[0].Method)
	require.Contains(t, acts[0].Command, "api.digitalocean.com")
}
