package openapi

import (
	"encoding/json"
	"testing"

	"github.com/stretchr/testify/require"
)

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
