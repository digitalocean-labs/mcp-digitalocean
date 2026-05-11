package registry

import (
	"context"
	"log/slog"
	"testing"

	"github.com/digitalocean/godo"
	"github.com/mark3labs/mcp-go/server"
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

func TestRegister_openapi_disableDeletes_omitsTool(t *testing.T) {
	t.Parallel()
	logger := slog.New(slog.DiscardHandler)
	getClient := func(context.Context) (*godo.Client, error) { return nil, nil }

	svr := server.NewMCPServer("test", "1.0")
	err := Register(logger, svr, getClient, Options{OpenAPIDisableDeletes: true}, "openapi")
	require.NoError(t, err)
	tools := svr.ListTools()
	_, hasDelete := tools["openapi-execute-delete"]
	require.False(t, hasDelete)

	svr2 := server.NewMCPServer("test", "1.0")
	err = Register(logger, svr2, getClient, Options{OpenAPIDisableDeletes: false}, "openapi")
	require.NoError(t, err)
	tools2 := svr2.ListTools()
	_, hasDelete2 := tools2["openapi-execute-delete"]
	require.True(t, hasDelete2)
}
