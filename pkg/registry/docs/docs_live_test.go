//go:build integration

package docs

import (
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

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
}
