package openapi

import (
	"encoding/json"
	"testing"

	"github.com/stretchr/testify/require"
)

// dropletsListFixtureJSON is a valid JSON reduction of the DigitalOcean API list-droplets
// response (see components/examples/droplets_all and responses/all_droplets in
// spec/DigitalOcean-public.v2.yaml), plus documented meta and links.pages pagination objects.
const dropletsListFixtureJSON = `{
  "droplets": [
    {
      "id": 3164444,
      "name": "example.com",
      "features": ["backups", "ipv6"],
      "networks": {
        "v4": [
          {"ip_address": "10.128.192.124", "type": "private"},
          {"ip_address": "192.241.165.154", "type": "public"}
        ]
      },
      "disk_info": [{"type": "local", "size": {"amount": 25, "unit": "gib"}}],
      "size": {"slug": "s-1vcpu-1gb", "memory": 1024}
    },
    {
      "id": 3164459,
      "name": "assets.example.com",
      "features": ["private_networking"],
      "networks": {"v4": []},
      "disk_info": [{"type": "local"}],
      "size": {"slug": "s-1vcpu-1gb"}
    }
  ],
  "meta": {"total": 43},
  "links": {
    "pages": {
      "next": "https://api.digitalocean.com/v2/droplets?page=2",
      "last": "https://api.digitalocean.com/v2/droplets?page=3"
    }
  }
}`

func TestSelectJSON(t *testing.T) {
	t.Parallel()
	raw := `{"droplets":[{"id":1,"name":"a"},{"id":2}]}`
	var data any
	require.NoError(t, json.Unmarshal([]byte(raw), &data))

	out, err := selectJSON("droplets[*].id", data)
	require.NoError(t, err)
	arr, ok := out.([]any)
	require.True(t, ok)
	require.Len(t, arr, 2)
	require.Equal(t, float64(1), arr[0])
	require.Equal(t, float64(2), arr[1])

	single, err := selectJSON("droplets[0].name", data)
	require.NoError(t, err)
	require.Equal(t, "a", single)
}

func TestSelectJSON_digitalOceanListResponse(t *testing.T) {
	t.Parallel()
	var root any
	require.NoError(t, json.Unmarshal([]byte(dropletsListFixtureJSON), &root))

	t.Run("meta_total", func(t *testing.T) {
		t.Parallel()
		got, err := selectJSON("meta.total", root)
		require.NoError(t, err)
		require.Equal(t, float64(43), got)
	})

	t.Run("links_pages_next", func(t *testing.T) {
		t.Parallel()
		got, err := selectJSON("links.pages.next", root)
		require.NoError(t, err)
		require.Equal(t, "https://api.digitalocean.com/v2/droplets?page=2", got)
	})

	t.Run("droplet_ids", func(t *testing.T) {
		t.Parallel()
		got, err := selectJSON("droplets[*].id", root)
		require.NoError(t, err)
		arr := got.([]any)
		require.Len(t, arr, 2)
		require.Equal(t, float64(3164444), arr[0])
		require.Equal(t, float64(3164459), arr[1])
	})

	t.Run("droplet_names", func(t *testing.T) {
		t.Parallel()
		got, err := selectJSON("droplets[*].name", root)
		require.NoError(t, err)
		arr := got.([]any)
		require.Equal(t, "example.com", arr[0])
		require.Equal(t, "assets.example.com", arr[1])
	})

	t.Run("nested_wildcard_network_v4_ip", func(t *testing.T) {
		t.Parallel()
		got, err := selectJSON("droplets[*].networks.v4[*].ip_address", root)
		require.NoError(t, err)
		outer := got.([]any)
		require.Len(t, outer, 2)
		first := outer[0].([]any)
		require.Len(t, first, 2)
		require.Equal(t, "10.128.192.124", first[0])
		require.Equal(t, "192.241.165.154", first[1])
		second := outer[1].([]any)
		require.Len(t, second, 0)
	})

	t.Run("nested_wildcard_network_v4_type", func(t *testing.T) {
		t.Parallel()
		got, err := selectJSON("droplets[*].networks.v4[*].type", root)
		require.NoError(t, err)
		outer := got.([]any)
		first := outer[0].([]any)
		require.Equal(t, "private", first[0])
		require.Equal(t, "public", first[1])
	})

	t.Run("disk_info_types_per_droplet", func(t *testing.T) {
		t.Parallel()
		got, err := selectJSON("droplets[*].disk_info[*].type", root)
		require.NoError(t, err)
		outer := got.([]any)
		require.Len(t, outer, 2)
		require.Equal(t, []any{"local"}, outer[0])
		require.Equal(t, []any{"local"}, outer[1])
	})

	t.Run("indexed_first_size_slug", func(t *testing.T) {
		t.Parallel()
		got, err := selectJSON("droplets[0].size.slug", root)
		require.NoError(t, err)
		require.Equal(t, "s-1vcpu-1gb", got)
	})

	t.Run("indexed_second_features", func(t *testing.T) {
		t.Parallel()
		got, err := selectJSON("droplets[1].features", root)
		require.NoError(t, err)
		feat := got.([]any)
		require.Len(t, feat, 1)
		require.Equal(t, "private_networking", feat[0])
	})
}

func TestSelectJSON_digitalOceanSingleResource(t *testing.T) {
	t.Parallel()
	// Mirrors singular resource wrapper described in the API introduction (resource key is singular).
	raw := `{
  "droplet": {
    "id": 3164444,
    "name": "example.com",
    "image": {
      "slug": "ubuntu-20-04-x64",
      "distribution": "Ubuntu"
    },
    "region": {
      "slug": "nyc3",
      "name": "New York 3"
    },
    "tags": ["web", "env:prod"]
  }
}`
	var root any
	require.NoError(t, json.Unmarshal([]byte(raw), &root))

	got, err := selectJSON("droplet.image.slug", root)
	require.NoError(t, err)
	require.Equal(t, "ubuntu-20-04-x64", got)

	got, err = selectJSON("droplet.region.slug", root)
	require.NoError(t, err)
	require.Equal(t, "nyc3", got)

	got, err = selectJSON("droplet.tags", root)
	require.NoError(t, err)
	tags := got.([]any)
	require.Len(t, tags, 2)
}

func TestSelectJSON_errors(t *testing.T) {
	t.Parallel()
	var root any
	require.NoError(t, json.Unmarshal([]byte(dropletsListFixtureJSON), &root))

	_, err := selectJSON("droplets[*].missing_key", root)
	require.Error(t, err)

	_, err = selectJSON("droplets[99].id", root)
	require.Error(t, err)
}
