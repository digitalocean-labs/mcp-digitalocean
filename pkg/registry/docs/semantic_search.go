package docs

import (
	"context"
	"errors"
	"fmt"
	"math"
	"sort"
	"strings"
	"sync"

	"golang.org/x/sync/errgroup"
)

const (
	semanticStage1Max    = 40
	semanticBodyMaxRunes = 6000
	semanticFetchWorkers = 8
)

type docBody struct {
	title   string
	url     string
	textBM  string
	excerpt string
}

type bm25Doc struct {
	terms []string
	tf    map[string]float64
	dl    int
	title string
	url   string
	raw   string
}

func tokenize(s string) []string {
	return strings.Fields(strings.ToLower(s))
}

func termFreq(tokens []string) map[string]float64 {
	m := make(map[string]float64)
	for _, t := range tokens {
		m[t]++
	}
	return m
}

func buildBM25Docs(docs []docBody) []bm25Doc {
	out := make([]bm25Doc, 0, len(docs))
	for _, d := range docs {
		toks := tokenize(d.textBM)
		if len(toks) == 0 {
			continue
		}
		out = append(out, bm25Doc{
			terms: toks,
			tf:    termFreq(toks),
			dl:    len(toks),
			title: d.title,
			url:   d.url,
			raw:   d.excerpt,
		})
	}
	return out
}

func bm25ScoreAll(query string, corpus []bm25Doc) []float64 {
	qTerms := tokenize(query)
	N := len(corpus)
	if N == 0 || len(qTerms) == 0 {
		return nil
	}

	df := make(map[string]int)
	for _, d := range corpus {
		seen := make(map[string]struct{})
		for _, t := range d.terms {
			if _, ok := seen[t]; ok {
				continue
			}
			seen[t] = struct{}{}
			df[t]++
		}
	}

	avgdl := 0.0
	totalLen := 0
	for _, d := range corpus {
		totalLen += d.dl
	}
	if totalLen > 0 {
		avgdl = float64(totalLen) / float64(N)
	}
	if avgdl == 0 {
		avgdl = 1
	}

	const k1, b = 1.2, 0.75

	scores := make([]float64, N)
	for i, d := range corpus {
		score := 0.0
		for _, q := range qTerms {
			freq, ok := d.tf[q]
			if !ok || freq == 0 {
				continue
			}
			dfq := df[q]
			idf := math.Log((float64(N)-float64(dfq)+0.5)/(float64(dfq)+0.5) + 1.0)
			denom := freq + k1*(1-b+b*float64(d.dl)/avgdl)
			score += idf * (freq * (k1 + 1)) / denom
		}
		scores[i] = score
	}
	return scores
}

func snippetFromDoc(excerpt, query string, maxRunes int) string {
	runes := []rune(excerpt)
	runesLower := []rune(strings.ToLower(excerpt))
	for _, w := range tokenize(query) {
		if len([]rune(w)) < 3 {
			continue
		}
		wRunes := []rune(w)
		idx := runeIndex(runesLower, wRunes)
		if idx >= 0 {
			start := idx - 80
			if start < 0 {
				start = 0
			}
			end := start + maxRunes
			if end > len(runes) {
				end = len(runes)
			}
			return strings.TrimSpace(string(runes[start:end]))
		}
	}
	if len(runes) > maxRunes {
		return strings.TrimSpace(string(runes[:maxRunes])) + "..."
	}
	return strings.TrimSpace(string(runes))
}

func runeIndex(hay, needle []rune) int {
	if len(needle) > len(hay) {
		return -1
	}
outer:
	for i := 0; i <= len(hay)-len(needle); i++ {
		for j := range needle {
			if hay[i+j] != needle[j] {
				continue outer
			}
		}
		return i
	}
	return -1
}

// SemanticSearch ranks llms-index.json entries with a second stage BM25 pass over fetched markdown excerpts.
func (d *DocsClient) SemanticSearch(ctx context.Context, query string, resultLimit int) ([]SemanticSearchHit, error) {
	if resultLimit <= 0 {
		resultLimit = defaultSearchLimit
	}

	records, err := d.GetLLMSIndexRecords()
	if err != nil {
		return nil, err
	}

	pseudo := &DocsIndex{Entries: make([]DocsEntry, len(records))}
	urlToMarkdown := make(map[string]string, len(records))
	for i, r := range records {
		sec := r.Product
		if r.Section != "" {
			sec = r.Product + " / " + r.Section
		}
		pseudo.Entries[i] = DocsEntry{
			Title:       r.Title,
			URL:         r.URL,
			Description: r.Description,
			Section:     sec,
		}
		urlToMarkdown[r.URL] = r.MarkdownURL
	}

	stage1 := SearchIndex(pseudo, query)
	if len(stage1) == 0 {
		return nil, nil
	}
	if len(stage1) > semanticStage1Max {
		stage1 = stage1[:semanticStage1Max]
	}

	g, gctx := errgroup.WithContext(ctx)
	g.SetLimit(semanticFetchWorkers)

	var mu sync.Mutex
	loaded := make([]docBody, 0, len(stage1))

	for _, e := range stage1 {
		e := e
		mdURL := urlToMarkdown[e.URL]
		g.Go(func() error {
			if err := gctx.Err(); err != nil {
				return err
			}
			var cl string
			if mdURL != "" {
				raw, err := d.fetch(mdURL)
				if err != nil {
					raw = e.Description
				}
				cl = cleanMarkdown(raw)
			} else {
				cl = e.Description
			}
			rn := []rune(cl)
			if len(rn) > semanticBodyMaxRunes {
				cl = string(rn[:semanticBodyMaxRunes])
			}
			var bm strings.Builder
			bm.WriteString(strings.ToLower(e.Title))
			bm.WriteByte('\n')
			bm.WriteString(strings.ToLower(e.Description))
			bm.WriteByte('\n')
			bm.WriteString(strings.ToLower(cl))
			mu.Lock()
			loaded = append(loaded, docBody{title: e.Title, url: e.URL, textBM: bm.String(), excerpt: cl})
			mu.Unlock()
			return nil
		})
	}
	if err := g.Wait(); err != nil && !errors.Is(err, context.Canceled) {
		return nil, fmt.Errorf("fetch markdown for semantic search: %w", err)
	}

	corpus := buildBM25Docs(loaded)
	if len(corpus) == 0 {
		return nil, nil
	}
	scores := bm25ScoreAll(query, corpus)
	type idxScore struct {
		i int
		s float64
	}
	var ranked []idxScore
	for i, s := range scores {
		if s > 0 {
			ranked = append(ranked, idxScore{i, s})
		}
	}
	sort.Slice(ranked, func(i, j int) bool {
		return ranked[i].s > ranked[j].s
	})
	if len(ranked) > resultLimit {
		ranked = ranked[:resultLimit]
	}

	out := make([]SemanticSearchHit, 0, len(ranked))
	for _, r := range ranked {
		d := corpus[r.i]
		out = append(out, SemanticSearchHit{
			Title:   d.title,
			URL:     d.url,
			Score:   r.s,
			Snippet: snippetFromDoc(d.raw, query, 220),
		})
	}
	return out, nil
}
