package openapi

import (
	"context"
	"net/http"
	"net/http/httptest"
	"net/url"
	"testing"

	"github.com/digitalocean/godo"
	"github.com/stretchr/testify/require"
	"golang.org/x/oauth2"
)

func newTestGodoClient(t *testing.T, handler http.HandlerFunc) *godo.Client {
	t.Helper()
	srv := httptest.NewServer(handler)
	t.Cleanup(srv.Close)
	base, err := url.Parse(srv.URL)
	require.NoError(t, err)
	ts := oauth2.StaticTokenSource(&oauth2.Token{AccessToken: "test-token"})
	cli := oauth2.NewClient(context.Background(), ts)
	c, err := godo.New(cli, godo.SetBaseURL(base.String()))
	require.NoError(t, err)
	return c
}

func newTestOpenAPIClientYAML(t *testing.T, yaml string) *OpenAPIClient {
	t.Helper()
	return &OpenAPIClient{embedOverride: []byte(yaml)}
}
