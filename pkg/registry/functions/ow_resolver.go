package functions

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"net/http"
	"strings"
	"sync"
	"time"

	middleware "mcp-digitalocean/internal"

	"github.com/digitalocean/godo"
)

const (
	mcpKeyPrefix = "mcp-do-"
	keyTTL       = "24h"
	// Refresh the cached key when it has less than this duration remaining,
	// so we never hand out a key that expires mid-request.
	keyRefreshBuffer = time.Hour
)

// cachedAuth holds a resolved OW client and the metadata needed to clean up
// the access key later.
type cachedAuth struct {
	ow         *owClient
	nsName     string
	nsID       string
	keyID      string
	validUntil time.Time
}

// OWResolver lazily provisions short-lived access keys for the OpenWhisk data
// plane and caches them in memory. On first use for a given namespace it:
//  1. Fetches namespace metadata (api_host, namespace name).
//  2. Lists existing access keys and deletes any with the "mcp-do-" prefix
//     (orphans from previous sessions).
//  3. Creates a new 24h access key.
//  4. Caches the result for the process lifetime (or until near-expiry).
type OWResolver struct {
	client func(ctx context.Context) (*godo.Client, error)
	mu     sync.Mutex
	cache  map[string]*cachedAuth
}

func NewOWResolver(client func(ctx context.Context) (*godo.Client, error)) *OWResolver {
	return &OWResolver{
		client: client,
		cache:  make(map[string]*cachedAuth),
	}
}

// cacheKey derives a cache key that is scoped to both the caller's identity
// and the namespace. In HTTP transport mode each user has a distinct auth token
// on the context, so different users get separate cache entries. In stdio mode
// (no per-request auth) we fall back to namespace-only keying.
func cacheKey(ctx context.Context, namespaceID string) string {
	auth, _ := ctx.Value(middleware.AuthKey{}).(string)
	if auth == "" {
		return namespaceID
	}
	h := sha256.Sum256([]byte(auth))
	return hex.EncodeToString(h[:8]) + ":" + namespaceID
}

// Resolve returns an authenticated OW client and the OW namespace name for the
// given DO namespace UUID. It creates or reuses a cached access key scoped to
// the caller's identity.
func (r *OWResolver) Resolve(ctx context.Context, namespaceID string) (*owClient, string, error) {
	ck := cacheKey(ctx, namespaceID)

	r.mu.Lock()
	if cached, ok := r.cache[ck]; ok && time.Now().Before(cached.validUntil) {
		r.mu.Unlock()
		return cached.ow, cached.nsName, nil
	}
	r.mu.Unlock()

	gc, err := r.client(ctx)
	if err != nil {
		return nil, "", fmt.Errorf("failed to get DigitalOcean client: %w", err)
	}

	// This call acts as the authorization check: if the caller's token cannot
	// access this namespace, godo returns an error and we never reach key creation.
	ns, _, err := gc.Functions.GetNamespace(ctx, namespaceID)
	if err != nil {
		return nil, "", fmt.Errorf("get namespace: %w", err)
	}
	if ns.ApiHost == "" {
		return nil, "", fmt.Errorf("namespace %s has no api_host", namespaceID)
	}

	r.cleanupOrphanedKeys(ctx, gc, namespaceID)

	key, err := r.createKey(ctx, gc, namespaceID)
	if err != nil {
		return nil, "", fmt.Errorf("create access key for namespace %s: %w (function:admin permission required)", namespaceID, err)
	}

	authKey := key.ID + ":" + key.Secret
	ow := newOWClient(ns.ApiHost, authKey)

	entry := &cachedAuth{
		ow:         ow,
		nsName:     ns.Namespace,
		nsID:       namespaceID,
		keyID:      key.ID,
		validUntil: time.Now().Add(24*time.Hour - keyRefreshBuffer),
	}

	r.mu.Lock()
	r.cache[ck] = entry
	r.mu.Unlock()

	return ow, ns.Namespace, nil
}

// Cleanup makes a best-effort attempt to delete all access keys created by
// this session. Call during process shutdown.
func (r *OWResolver) Cleanup(ctx context.Context) {
	r.mu.Lock()
	entries := make([]*cachedAuth, 0, len(r.cache))
	for _, e := range r.cache {
		entries = append(entries, e)
	}
	r.cache = make(map[string]*cachedAuth)
	r.mu.Unlock()

	for _, entry := range entries {
		gc, err := r.client(ctx)
		if err != nil {
			continue
		}
		path := fmt.Sprintf("/v2/functions/namespaces/%s/keys/%s", entry.nsID, entry.keyID)
		req, err := gc.NewRequest(ctx, http.MethodDelete, path, nil)
		if err != nil {
			continue
		}
		gc.Do(ctx, req, nil) //nolint:errcheck
	}
}

func (r *OWResolver) cleanupOrphanedKeys(ctx context.Context, gc *godo.Client, namespaceID string) {
	path := fmt.Sprintf("/v2/functions/namespaces/%s/keys", namespaceID)
	req, err := gc.NewRequest(ctx, http.MethodGet, path, nil)
	if err != nil {
		return
	}

	root := new(accessKeysRoot)
	_, err = gc.Do(ctx, req, root)
	if err != nil {
		return
	}

	for _, k := range root.AccessKeys {
		if strings.HasPrefix(k.Name, mcpKeyPrefix) {
			delPath := fmt.Sprintf("/v2/functions/namespaces/%s/keys/%s", namespaceID, k.ID)
			delReq, err := gc.NewRequest(ctx, http.MethodDelete, delPath, nil)
			if err != nil {
				continue
			}
			gc.Do(ctx, delReq, nil) //nolint:errcheck
		}
	}
}

func (r *OWResolver) createKey(ctx context.Context, gc *godo.Client, namespaceID string) (*AccessKey, error) {
	name := mcpKeyPrefix + fmt.Sprintf("%d", time.Now().UnixMilli())
	body := &accessKeyCreateRequest{
		Name:      name,
		ExpiresIn: keyTTL,
	}

	path := fmt.Sprintf("/v2/functions/namespaces/%s/keys", namespaceID)
	req, err := gc.NewRequest(ctx, http.MethodPost, path, body)
	if err != nil {
		return nil, fmt.Errorf("create request: %w", err)
	}

	root := new(accessKeyRoot)
	_, err = gc.Do(ctx, req, root)
	if err != nil {
		return nil, fmt.Errorf("create key API call: %w", err)
	}
	if root.AccessKey == nil {
		return nil, fmt.Errorf("empty response from create key")
	}

	return root.AccessKey, nil
}
