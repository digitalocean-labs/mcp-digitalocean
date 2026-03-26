package functions

import (
	"bytes"
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
)

const owBasePath = "/api/v1"

type owClient struct {
	apiHost string
	authKey string
	http    *http.Client
}

type owError struct {
	Error string `json:"error"`
	Code  string `json:"code,omitempty"`
}

func newOWClient(apiHost, authKey string) *owClient {
	apiHost = strings.TrimRight(apiHost, "/")
	return &owClient{
		apiHost: apiHost,
		authKey: authKey,
		http:    http.DefaultClient,
	}
}

func (c *owClient) basicAuth() string {
	return "Basic " + base64.StdEncoding.EncodeToString([]byte(c.authKey))
}

func (c *owClient) do(ctx context.Context, method, path string, query url.Values, body any) ([]byte, error) {
	u := c.apiHost + owBasePath + path
	if len(query) > 0 {
		u += "?" + query.Encode()
	}

	var bodyReader io.Reader
	if body != nil {
		b, err := json.Marshal(body)
		if err != nil {
			return nil, fmt.Errorf("marshal request body: %w", err)
		}
		bodyReader = bytes.NewReader(b)
	}

	req, err := http.NewRequestWithContext(ctx, method, u, bodyReader)
	if err != nil {
		return nil, fmt.Errorf("create request: %w", err)
	}
	req.Header.Set("Authorization", c.basicAuth())
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Accept", "application/json")

	resp, err := c.http.Do(req)
	if err != nil {
		return nil, fmt.Errorf("execute request: %w", err)
	}
	defer resp.Body.Close()

	data, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("read response: %w", err)
	}

	if resp.StatusCode >= 400 {
		var oe owError
		if json.Unmarshal(data, &oe) == nil && oe.Error != "" {
			return nil, fmt.Errorf("openwhisk API error (HTTP %d): %s", resp.StatusCode, oe.Error)
		}
		return nil, fmt.Errorf("openwhisk API error (HTTP %d): %s", resp.StatusCode, string(data))
	}

	return data, nil
}

func (c *owClient) get(ctx context.Context, path string, query url.Values) ([]byte, error) {
	return c.do(ctx, http.MethodGet, path, query, nil)
}

func (c *owClient) put(ctx context.Context, path string, query url.Values, body any) ([]byte, error) {
	return c.do(ctx, http.MethodPut, path, query, body)
}

func (c *owClient) post(ctx context.Context, path string, query url.Values, body any) ([]byte, error) {
	return c.do(ctx, http.MethodPost, path, query, body)
}

func (c *owClient) del(ctx context.Context, path string, query url.Values) ([]byte, error) {
	return c.do(ctx, http.MethodDelete, path, query, nil)
}
