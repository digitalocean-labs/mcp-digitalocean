package microdroplet

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"

	"github.com/digitalocean/godo"
)

const apiBasePath = "v2/microdroplets"

type apiClient struct {
	client *godo.Client
}

func newAPIClient(client *godo.Client) *apiClient {
	return &apiClient{client: client}
}

func (a *apiClient) do(ctx context.Context, method, subPath string, query url.Values, body any) (json.RawMessage, error) {
	path := apiBasePath + subPath
	if len(query) > 0 {
		path += "?" + query.Encode()
	}

	req, err := a.client.NewRequest(ctx, method, path, body)
	if err != nil {
		return nil, fmt.Errorf("build request: %w", err)
	}

	var out json.RawMessage
	_, err = a.client.Do(ctx, req, &out)
	if err != nil {
		return nil, err
	}
	return out, nil
}

func (a *apiClient) doNoContent(ctx context.Context, method, subPath string) error {
	req, err := a.client.NewRequest(ctx, method, apiBasePath+subPath, nil)
	if err != nil {
		return fmt.Errorf("build request: %w", err)
	}

	resp, err := a.client.Do(ctx, req, nil)
	if err != nil {
		return err
	}
	if resp != nil && resp.StatusCode != http.StatusNoContent && resp.StatusCode != http.StatusOK {
		return fmt.Errorf("unexpected status %d", resp.StatusCode)
	}
	return nil
}
