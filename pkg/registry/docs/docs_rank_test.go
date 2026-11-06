package docs

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestProfileQuery(t *testing.T) {
	tests := []struct {
		query string
		want  queryProfile
	}{
		{"create a VPC network", queryProfile{action: true, terms: 4}},
		{"doctl compute droplet create flags", queryProfile{interfaces: true, action: true, terms: 5}},
		{"520 status code app platform", queryProfile{trouble: true, terms: 5}},
		{"how much does Spaces cost", queryProfile{pricing: true, terms: 5}},
	}
	for _, tc := range tests {
		t.Run(tc.query, func(t *testing.T) {
			require.Equal(t, tc.want, profileQuery(tc.query))
		})
	}
}

func TestPriorFactor(t *testing.T) {
	guide := queryProfile{action: true}
	cli := queryProfile{interfaces: true}

	require.Less(t, priorFactor("/reference/doctl/reference/compute/droplet/create/", guide, false), 1.0)
	require.Equal(t, 1.0, priorFactor("/reference/doctl/reference/compute/droplet/create/", cli, false))
	require.Less(t, priorFactor("/support/why-is-smtp-blocked/", guide, false), 1.0)
	require.Equal(t, 1.0, priorFactor("/support/why-is-smtp-blocked/", queryProfile{trouble: true}, false))
	require.Less(t, priorFactor("/products/spaces/", guide, false), 1.0)
	require.Equal(t, 1.0, priorFactor("/products/spaces/", queryProfile{}, true))
	require.Greater(t, priorFactor("/products/vpc/how-to/create/", guide, false), 1.0)
	require.Greater(t, priorFactor("/products/spaces/details/pricing/", queryProfile{pricing: true}, false), 1.0)
}

func TestCoverageFactorPrefersExactSlug(t *testing.T) {
	terms := parseQueryTerms("create a VPC network")
	create := coverageFactor(terms, fieldTerms("how to create vpc networks"), fieldTerms("create"))
	peering := coverageFactor(terms, fieldTerms("how to create vpc peering"), fieldTerms("create-peering"))
	require.Greater(t, create, peering)
}

func TestSynonymsMatchAlternateWords(t *testing.T) {
	terms := parseQueryTerms("remove an organization")
	require.True(t, terms[0].matchesTokens(fieldTerms("how to delete organizations")))
	require.False(t, parseQueryTerms("rename")[0].matchesTokens(fieldTerms("how to delete organizations")))
}

func TestSearchIndexDemotesGeneratedReference(t *testing.T) {
	index := newDocsIndex([]DocsEntry{
		{Title: "doctl compute vpc create", URL: docsBase + "/reference/doctl/reference/compute/vpc/create/"},
		{Title: "How to Create VPC Networks", URL: docsBase + "/products/networking/vpc/how-to/create/"},
	}, nil, "test")

	results := SearchIndex(index, "create a VPC network")
	require.Equal(t, docsBase+"/products/networking/vpc/how-to/create/", results[0].URL)

	results = SearchIndex(index, "doctl create vpc")
	require.Equal(t, docsBase+"/reference/doctl/reference/compute/vpc/create/", results[0].URL)
}

func TestCoverageFactorIgnoresStopWords(t *testing.T) {
	terms := parseQueryTerms("create a serverless function")
	guide := coverageFactor(terms, fieldTerms("how to create functions"), fieldTerms("create-functions"))
	require.Equal(t, 1.5, guide)
}

func TestTermCoverageFactorFavorsFullerMatches(t *testing.T) {
	require.Equal(t, 1.0, termCoverageFactor(7, 7))
	require.Less(t, termCoverageFactor(2, 7), termCoverageFactor(6, 7))
	require.Equal(t, 1.0, termCoverageFactor(0, 0))
}
