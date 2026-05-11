package openapi

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestSubstitutePath(t *testing.T) {
	out, err := substitutePath("/v2/droplets/{droplet_id}", map[string]string{"droplet_id": "123"})
	require.NoError(t, err)
	require.Equal(t, "/v2/droplets/123", out)

	_, err = substitutePath("/v2/droplets/{droplet_id}", map[string]string{})
	require.Error(t, err)
}

func TestParamToStrings(t *testing.T) {
	s, err := paramToStrings("x", "abc")
	require.NoError(t, err)
	require.Equal(t, []string{"abc"}, s)

	s, err = paramToStrings("x", float64(42))
	require.NoError(t, err)
	require.Equal(t, []string{"42"}, s)

	s, err = paramToStrings("x", []any{"a", float64(2)})
	require.NoError(t, err)
	require.Equal(t, []string{"a", "2"}, s)

	_, err = paramToStrings("x", nil)
	require.Error(t, err)

	_, err = paramToStrings("x", map[string]any{})
	require.Error(t, err)

	s, err = paramToStrings("x", []any{[]any{"nested"}})
	require.NoError(t, err)
	require.Equal(t, []string{"nested"}, s)
}

func TestEmbeddedSpecLoads(t *testing.T) {
	t.Parallel()
	c := NewOpenAPIClient()
	results, err := c.SearchOperations("droplet", "", 50)
	require.NoError(t, err)
	require.GreaterOrEqual(t, len(results), 5)
}

func TestGetOperation_droplets_list(t *testing.T) {
	t.Parallel()
	c := NewOpenAPIClient()
	op, err := c.GetOperation("droplets_list")
	require.NoError(t, err)
	require.Equal(t, "droplets_list", op.OperationID)
	require.Equal(t, "GET", op.Method)
	require.Contains(t, op.Path, "/droplets")
}
