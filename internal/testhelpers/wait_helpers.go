// Package testhelpers provides utilities for polling and waiting on DigitalOcean resources during integration tests.
package testhelpers

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"os"
	"time"

	"github.com/digitalocean/godo"
	"golang.org/x/oauth2"
)

// Constants for configuration and magic values
const (
	// Polling defaults
	defaultInterval = 2 * time.Second
	defaultTimeout  = 10 * time.Minute

	// Client configuration
	clientTimeout      = 30 * time.Second
	clientRetryMax     = 4
	clientRetryWaitMin = 1.0
	clientRetryWaitMax = 30.0

	// Environment variables
	envAPIToken = "DIGITALOCEAN_API_TOKEN"

	// Resource statuses
	actionStatusCompleted = "completed"
	actionStatusErrored   = "errored"
	dropletStatusActive   = "active"
	imageStatusAvailable  = "available"
	vectorDBStatusActive  = "active"
)

// WaitForAction polls for a droplet action to complete or error.
func WaitForAction(ctx context.Context, client *godo.Client, dropletID, actionID int, interval, timeout time.Duration) (*godo.Action, error) {
	return waitForActionGeneric(ctx, interval, timeout, func() (*godo.Action, *godo.Response, error) {
		return client.DropletActions.Get(ctx, dropletID, actionID)
	})
}

// WaitForImageAction polls for an image action to complete or error.
func WaitForImageAction(ctx context.Context, client *godo.Client, imageID, actionID int, interval, timeout time.Duration) (*godo.Action, error) {
	return waitForActionGeneric(ctx, interval, timeout, func() (*godo.Action, *godo.Response, error) {
		return client.ImageActions.Get(ctx, imageID, actionID)
	})
}

// WaitForStorageAction polls for a block storage (volume) action to complete or error.
func WaitForStorageAction(ctx context.Context, client *godo.Client, volumeID string, actionID int, interval, timeout time.Duration) (*godo.Action, error) {
	return waitForActionGeneric(ctx, interval, timeout, func() (*godo.Action, *godo.Response, error) {
		return client.StorageActions.Get(ctx, volumeID, actionID)
	})
}

// WaitForActions waits for multiple actions sequentially.
func WaitForActions(ctx context.Context, client *godo.Client, dropletID int, actionIDs []int, interval, timeout time.Duration) ([]*godo.Action, error) {
	results := make([]*godo.Action, 0, len(actionIDs))
	for _, aid := range actionIDs {
		act, err := WaitForAction(ctx, client, dropletID, aid, interval, timeout)
		if err != nil {
			return nil, err
		}
		results = append(results, act)
	}
	return results, nil
}

// WaitForDroplet polls until the predicate returns true.
// If predicate is nil, it returns (nil, nil) immediately upon a 404 (used for deletion checks).
func WaitForDroplet(ctx context.Context, client *godo.Client, dropletID int, predicate func(*godo.Droplet) bool, interval, timeout time.Duration) (*godo.Droplet, error) {
	return waitForResource(ctx, interval, timeout,
		func() (*godo.Droplet, *godo.Response, error) {
			return client.Droplets.Get(ctx, dropletID)
		},
		predicate,
	)
}

// WaitForImage polls until the predicate returns true.
// If predicate is nil, it returns (nil, nil) immediately upon a 404 (used for deletion checks).
func WaitForImage(ctx context.Context, client *godo.Client, imageID int, predicate func(*godo.Image) bool, interval, timeout time.Duration) (*godo.Image, error) {
	return waitForResource(ctx, interval, timeout,
		func() (*godo.Image, *godo.Response, error) {
			return client.Images.GetByID(ctx, imageID)
		},
		predicate,
	)
}

// WaitForNfsShare polls until the predicate returns true.
func WaitForNfsShare(ctx context.Context, client *godo.Client, shareID string, predicate func(*godo.Nfs) bool, interval, timeout time.Duration) (*godo.Nfs, error) {
	return waitForResource(ctx, interval, timeout,
		func() (*godo.Nfs, *godo.Response, error) {
			return client.Nfs.Get(ctx, shareID, "")
		},
		predicate,
	)
}

// WaitForVectorDB polls until the predicate returns true.
func WaitForVectorDB(ctx context.Context, client *godo.Client, id string, predicate func(*godo.VectorDB) bool, interval, timeout time.Duration) (*godo.VectorDB, error) {
	return waitForResource(ctx, interval, timeout,
		func() (*godo.VectorDB, *godo.Response, error) {
			return client.VectorDBs.Get(ctx, id)
		},
		predicate,
	)
}

// WaitForDropletDeleted checks for 404 status.
func WaitForDropletDeleted(ctx context.Context, client *godo.Client, dropletID int, interval, timeout time.Duration) error {
	_, err := WaitForDroplet(ctx, client, dropletID, nil, interval, timeout)
	return err
}

// WaitForImageDeleted checks for 404 status.
func WaitForImageDeleted(ctx context.Context, client *godo.Client, imageID int, interval, timeout time.Duration) error {
	_, err := WaitForImage(ctx, client, imageID, nil, interval, timeout)
	return err
}

// IsDropletActive checks if status is active and has an IPv4.
func IsDropletActive(d *godo.Droplet) bool {
	return d != nil && d.Status == dropletStatusActive && d.Networks != nil && len(d.Networks.V4) > 0
}

// IsImageAvailable checks if the image status is available.
func IsImageAvailable(i *godo.Image) bool {
	return i != nil && i.Status == imageStatusAvailable
}

// IsNfsShareActive checks if the NFS share status is active.
func IsNfsShareActive(n *godo.Nfs) bool {
	return n != nil && n.Status == godo.NfsShareActive
}

// IsVectorDBActive checks if the vector database status is active.
func IsVectorDBActive(v *godo.VectorDB) bool {
	return v != nil && v.Status == vectorDBStatusActive
}

// MustGodoClient returns a client or error if the token is missing.
func MustGodoClient(ctx context.Context, testName string) (*godo.Client, error) {
	token := os.Getenv(envAPIToken)
	if token == "" {
		return nil, fmt.Errorf("%s environment variable must be set to run E2E tests", envAPIToken)
	}

	tokenSource := oauth2.StaticTokenSource(&oauth2.Token{AccessToken: token})
	oauthClient := oauth2.NewClient(ctx, tokenSource)

	// 1. Timeout: Prevent indefinite hangs (Client-side resiliency)
	oauthClient.Timeout = clientTimeout

	// 2. Retries: Handle API flakes and Rate Limits (Server-side resiliency)
	retryConfig := godo.RetryConfig{
		RetryMax:     clientRetryMax,
		RetryWaitMin: godo.PtrTo(clientRetryWaitMin),
		RetryWaitMax: godo.PtrTo(clientRetryWaitMax),
	}

	return godo.New(
		oauthClient,
		godo.WithRetryAndBackoffs(retryConfig),
		godo.SetUserAgent(fmt.Sprintf("mcp-e2e-tests-%s", testName)),
	)
}

// --- Internal Helpers ---

// waitForActionGeneric handles the common logic for polling Action status (Completed/Errored).
func waitForActionGeneric(ctx context.Context, interval, timeout time.Duration, fetch func() (*godo.Action, *godo.Response, error)) (*godo.Action, error) {
	var action *godo.Action
	err := poll(ctx, interval, timeout, func() (bool, error) {
		a, resp, err := fetch()

		// Resiliency: If the request timed out locally (client-side), just retry
		if err != nil && os.IsTimeout(err) {
			return false, nil
		}

		if resp != nil && resp.StatusCode == http.StatusNotFound {
			return false, errors.New("action not found")
		}
		if err != nil {
			return false, err
		}
		action = a
		switch a.Status {
		case actionStatusCompleted:
			return true, nil
		case actionStatusErrored:
			return false, errors.New("action errored")
		default:
			return false, nil // in-progress
		}
	})
	return action, err
}

// waitForResource is a generic helper to poll for a resource [T] until it satisfies a predicate.
// If predicate is nil, it waits for the resource to NOT be found (404), used for deletion checks.
func waitForResource[T any](ctx context.Context, interval, timeout time.Duration, fetch func() (*T, *godo.Response, error), predicate func(*T) bool) (*T, error) {
	var last *T
	err := poll(ctx, interval, timeout, func() (bool, error) {
		resource, resp, err := fetch()
		if err != nil {
			if os.IsTimeout(err) {
				return false, nil
			}

			if resp != nil && resp.StatusCode == http.StatusNotFound {
				if predicate == nil {
					last = nil
					return true, nil // Deletion confirmed
				}
				return false, fmt.Errorf("resource not found")
			}
			return false, err
		}
		last = resource
		if predicate != nil && predicate(resource) {
			return true, nil
		}
		return false, nil
	})
	return last, err
}

// poll is a generic loop that runs 'check' every 'interval' until 'timeout'.
// check() returns (done, error). If done=true, poll returns nil. If error!=nil, poll returns error.
func poll(ctx context.Context, interval, timeout time.Duration, check func() (bool, error)) error {
	if interval == 0 {
		interval = defaultInterval
	}
	if timeout == 0 {
		timeout = defaultTimeout
	}

	ticker := time.NewTicker(interval)
	defer ticker.Stop()
	timer := time.NewTimer(timeout)
	defer timer.Stop()

	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-timer.C:
			return fmt.Errorf("timed out after %s", timeout)
		case <-ticker.C:
			// fallthrough to check
		default:
			// Check immediately on first run
		}

		done, err := check()
		if err != nil {
			return err
		}
		if done {
			return nil
		}

		// Ensure we wait for the ticker if we just ran a check to avoid hot looping
		if interval > 0 {
			select {
			case <-ctx.Done():
				return ctx.Err()
			case <-timer.C:
				return fmt.Errorf("timed out after %s", timeout)
			case <-ticker.C:
				continue
			}
		}
	}
}
