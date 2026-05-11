package openapi

import (
	"sync"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestOpenAPIClient_parallelEnsure(t *testing.T) {
	const workers = 32
	var wg sync.WaitGroup
	c := NewOpenAPIClient()
	for i := 0; i < workers; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			_, err := c.SearchOperations("droplet", "", 3)
			require.NoError(t, err)
			_, err = c.GetOperation("droplets_list")
			require.NoError(t, err)
		}()
	}
	wg.Wait()
}

func TestLoad_duplicateOperationID(t *testing.T) {
	t.Parallel()
	yaml := `openapi: 3.0.0
info:
  title: T
  version: "1"
servers:
  - url: https://api.example.com
paths:
  /a:
    get:
      operationId: dup
      responses:
        "200":
          description: ok
  /b:
    get:
      operationId: dup
      responses:
        "200":
          description: ok
`
	c := newTestOpenAPIClientYAML(t, yaml)
	_, err := c.SearchOperations("dup", "", 10)
	require.Error(t, err)
	require.Contains(t, err.Error(), "duplicate operationId")
}

func TestLoad_invalidYAML(t *testing.T) {
	t.Parallel()
	c := newTestOpenAPIClientYAML(t, "openapi: [\n")
	_, err := c.SearchOperations("x", "", 10)
	require.Error(t, err)
}
