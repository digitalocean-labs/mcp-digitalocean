//go:build integration

package docs

import (
	"context"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

// Live MRR is gated loosely so an upstream content edit does not fail the
// opt-in integration test. Measured 0.875 on 2026-10-20.
const minimumLiveMRR = 0.75

// TestLiveRegressionSet runs the judged regression set against the live
// docs.digitalocean.com index. It guards against drift in the published
// index (missing pages, schema changes) rather than ranking details, so the
// recall bar is slightly below the fixture test's 100%.
func TestLiveRegressionSet(t *testing.T) {
	client := NewDocsClient()

	coldStart := time.Now()
	index, err := client.GetDocsIndex()
	require.NoError(t, err)
	coldLatency := time.Since(coldStart)

	require.Equal(t, "llms-index.json", index.Source, "live index should come from llms-index.json")
	require.Greater(t, len(index.Entries), 4000, "live index should cover the full page universe")

	supportPages := 0
	for _, e := range index.Entries {
		if e.Section == "Support" {
			supportPages++
		}
	}
	require.Greater(t, supportPages, 200, "live index should include individual support articles")

	warmStart := time.Now()
	_, err = client.GetDocsIndex()
	require.NoError(t, err)
	warmLatency := time.Since(warmStart)

	queries := loadRegressionQueries(t)
	hits := 0
	reciprocal := 0.0
	var searchTotal time.Duration
	for _, q := range queries {
		start := time.Now()
		results := rankForTool(index, q)
		searchTotal += time.Since(start)

		require.NotEmpty(t, results, "zero results for %q", q.Query)
		rank := rankOf(results, q.Expected)
		if rank == 0 || rank > 5 {
			t.Logf("MISS %q rank=%d top-5=%v", q.Query, rank, topURLs(results, 5))
			continue
		}
		hits++
		reciprocal += 1 / float64(rank)
	}

	recall := float64(hits) / float64(len(queries))
	t.Logf("live: pages=%d support=%d Recall@5=%d/%d (%.2f) MRR=%.3f index cold=%s warm=%s search avg=%s",
		len(index.Entries), supportPages, hits, len(queries), recall, reciprocal/float64(len(queries)),
		coldLatency, warmLatency, searchTotal/time.Duration(len(queries)))
	require.GreaterOrEqual(t, recall, 0.9, "live Recall@5 regressed")
	require.GreaterOrEqual(t, reciprocal/float64(len(queries)), minimumLiveMRR, "live MRR regressed")
	require.Less(t, searchTotal/time.Duration(len(queries)), 50*time.Millisecond, "per-query ranking latency regressed")
}

// TestLiveContentExcerpts measures excerpt coverage and the cost of lazy
// shard fetches over the regression set: how many top-5 results get a body
// excerpt containing a query term, and the cold/warm time per query.
func TestLiveContentExcerpts(t *testing.T) {
	client := NewDocsClient()
	index, err := client.GetDocsIndex()
	require.NoError(t, err)

	queries := loadRegressionQueries(t)
	ctx := context.Background()
	withExcerpt, total, top1WithExcerpt := 0, 0, 0
	var coldTotal, warmTotal, coldMax time.Duration
	for _, q := range queries {
		// Each query starts with a fresh corpus cache. The second call uses
		// that same client, so cold/warm measure distinct cache states.
		excerptClient := NewDocsClient()
		results := rankForTool(index, q)
		if len(results) > 5 {
			results = results[:5]
		}

		start := time.Now()
		excerpts := excerptClient.ContentExcerpts(ctx, results, q.Query)
		cold := time.Since(start)
		coldTotal += cold
		coldMax = max(coldMax, cold)

		start = time.Now()
		excerptClient.ContentExcerpts(ctx, results, q.Query)
		warmTotal += time.Since(start)

		terms := parseQueryTerms(q.Query)
		for i, r := range results {
			total++
			text, ok := excerpts[canonicalDocURL(r.URL)]
			if !ok {
				continue
			}
			lower := strings.ToLower(text)
			var hasTerm bool
			for _, term := range terms {
				hasTerm = hasTerm || term.matches(lower)
			}
			require.True(t, hasTerm, "excerpt for %q lacks query terms: %q", r.URL, text)
			withExcerpt++
			if i == 0 {
				top1WithExcerpt++
			}
		}
	}

	n := time.Duration(len(queries))
	t.Logf("live excerpts: results with body excerpt=%d/%d top-1 with excerpt=%d/%d cold avg=%s max=%s warm avg=%s",
		withExcerpt, total, top1WithExcerpt, len(queries), coldTotal/n, coldMax, warmTotal/n)
	require.GreaterOrEqual(t, float64(top1WithExcerpt)/float64(len(queries)), 0.8, "too few top results have a body excerpt")
	// Latency is reported, not asserted against the exact context deadline:
	// supported timeout fallback necessarily includes scheduling/cleanup time.
	// Deterministic cancellation tests enforce the fallback behavior offline.
}
