package registry

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestIsOpenAPIOnlyServices(t *testing.T) {
	t.Parallel()
	cases := []struct {
		name     string
		services []string
		want     bool
	}{
		{name: "openapi only", services: []string{"openapi"}, want: true},
		{name: "openapi only case", services: []string{"OpenAPI"}, want: true},
		{name: "openapi trimmed", services: []string{" openapi "}, want: true},
		{name: "empty", services: nil, want: false},
		{name: "openapi and droplets", services: []string{"openapi", "droplets"}, want: false},
		{name: "droplets only", services: []string{"droplets"}, want: false},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			got := isOpenAPIOnlyServices(tc.services)
			require.Equal(t, tc.want, got)
		})
	}
}
