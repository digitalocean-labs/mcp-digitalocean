package docs

import (
	"bufio"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"regexp"
	"strings"
	"sync"
	"time"
	"unicode"
	"unicode/utf8"

	"golang.org/x/sync/errgroup"
)

// The docs site publishes its page bodies as docs-corpus.jsonl: a manifest
// that points at one JSONL shard per directory. Shards are small enough to
// fetch on demand for the handful of pages a query returns, which is how
// search results get a query-focused excerpt from the page body without the
// server ever loading the whole corpus (32 MB over 538 requests).
const (
	corpusManifestPath = "/docs-corpus.jsonl"

	// maxShardsPerQuery bounds the fan-out for one search; results in further
	// shards fall back to their description.
	maxShardsPerQuery = 8
	shardConcurrency  = 4

	// maxBodyBytes bounds the body kept per page; excerpts come from the
	// start of the page in practice and a shard can hold a 2.7 MB changelog.
	maxBodyBytes = 64 * 1024

	excerptLength = 300
	// excerptLead is how far before the first matched term a window starts.
	excerptLead = 40
	// maxHitsPerTerm bounds window candidates on long bodies.
	maxHitsPerTerm = 8
)

// excerptBudget caps enrichment time; late shards are skipped.
var excerptBudget = 5 * time.Second

// corpusManifest maps a directory URL (with trailing slash) to the shard that
// holds the bodies of pages under it.
type corpusManifest struct {
	shardByDir map[string]string
}

type corpusManifestRecord struct {
	Kind string `json:"kind"`
	URL  string `json:"url"`
}

// parseCorpusManifest reads the shard records of a docs-corpus.jsonl
// manifest. Lines that are not shard records (the corpus header, inline
// docs) are ignored, as are shards outside the docs site.
func parseCorpusManifest(data string) *corpusManifest {
	m := &corpusManifest{shardByDir: make(map[string]string)}
	for _, line := range strings.Split(data, "\n") {
		if !strings.Contains(line, `"shard"`) {
			continue
		}
		var rec corpusManifestRecord
		if err := json.Unmarshal([]byte(line), &rec); err != nil || rec.Kind != "shard" {
			continue
		}
		if !strings.HasPrefix(rec.URL, docsBase+"/") {
			continue
		}
		dir := rec.URL[:strings.LastIndex(rec.URL, "/")+1]
		m.shardByDir[dir] = rec.URL
	}
	return m
}

// shardFor returns the shard holding pageURL by walking up its path until a
// directory with a shard is found. The empty string means no shard covers it.
func (m *corpusManifest) shardFor(pageURL string) string {
	dir := canonicalDocURL(pageURL)
	if !strings.HasSuffix(dir, "/") {
		dir += "/"
	}
	for len(dir) > len(docsBase)+1 {
		if shard, ok := m.shardByDir[dir]; ok {
			return shard
		}
		// Drop the last path segment: ".../a/b/" -> ".../a/".
		dir = dir[:strings.LastIndex(dir[:len(dir)-1], "/")+1]
	}
	return m.shardByDir[dir]
}

// getCorpusManifest returns the cached manifest, or nil when it cannot be
// loaded. Failures are negative-cached so a broken feed costs one request
// per negativeCacheTTL, not one per query.
func (d *DocsClient) getCorpusManifest(ctx context.Context) *corpusManifest {
	value := d.corpusValue(ctx, "corpusManifest", indexCacheTTL, func(loadCtx context.Context) (any, error) {
		data, err := d.fetchUncachedContext(loadCtx, docsBase+corpusManifestPath)
		if err != nil {
			return nil, err
		}
		m := parseCorpusManifest(data)
		if len(m.shardByDir) == 0 {
			return nil, fmt.Errorf("corpus manifest contains no shards")
		}
		return m, nil
	})
	m, _ := value.(*corpusManifest)
	return m
}

type corpusDoc struct {
	CanonicalURL  string `json:"canonical_url"`
	Content       string `json:"content"`
	ContentFormat string `json:"content_format"`
}

// parseShard reads page bodies out of a JSONL shard, keyed by canonical URL.
// Malformed records are skipped. Oversized HTTP responses are rejected before
// parsing, so an oversized feed is never cached as a successful prefix.
func parseShard(data string) map[string]string {
	bodies := make(map[string]string)
	scanner := bufio.NewScanner(strings.NewReader(data))
	scanner.Buffer(make([]byte, 0, 64*1024), maxResponseSize+1)
	for scanner.Scan() {
		var doc corpusDoc
		if err := json.Unmarshal(scanner.Bytes(), &doc); err != nil || doc.CanonicalURL == "" {
			continue
		}
		content := truncateUTF8(doc.Content, maxBodyBytes)
		if doc.ContentFormat == "markdown" {
			content = stripMarkdown(content)
		}
		bodies[doc.CanonicalURL] = strings.Clone(content)
	}
	return bodies
}

var (
	markdownImageRe   = regexp.MustCompile(`!\[[^\]]*\]\([^)]*\)`)
	markdownLinkRe    = regexp.MustCompile(`\[([^\]]*)\]\([^)]*\)`)
	markdownHeadingRe = regexp.MustCompile(`(?m)^[\t ]*#+[\t ]*`)
	markdownMarksRe   = regexp.MustCompile("[*_`]+")
)

func stripMarkdown(content string) string {
	content = markdownImageRe.ReplaceAllString(content, "")
	content = markdownLinkRe.ReplaceAllString(content, "$1")
	content = markdownHeadingRe.ReplaceAllString(content, "")
	return strings.TrimSpace(markdownMarksRe.ReplaceAllString(content, ""))
}

// loadShard returns the parsed shard, or nil when it is unavailable.
func (d *DocsClient) loadShard(ctx context.Context, shardURL string) map[string]string {
	value := d.corpusValue(ctx, "shard:"+shardURL, pageCacheTTL, func(loadCtx context.Context) (any, error) {
		data, err := d.fetchUncachedContext(loadCtx, shardURL)
		if err != nil {
			return nil, err
		}
		return parseShard(data), nil
	})
	bodies, _ := value.(map[string]string)
	return bodies
}

// maxConsecutiveTimeouts is how many times in a row a shared load may hit its
// own deadline before the key is negative-cached. One stall is left for the
// next query to retry; a persistently slow origin must not add the full
// budget to every query.
const maxConsecutiveTimeouts = 3

// slots returns the client-wide semaphore for corpus loads, creating it on
// first use so a zero-value DocsClient does not block on a nil channel.
func (d *DocsClient) slots() chan struct{} {
	d.corpusSlotsOnce.Do(func() {
		if d.corpusSlots == nil {
			d.corpusSlots = make(chan struct{}, shardConcurrency)
		}
	})
	return d.corpusSlots
}

// noteTimeout records a deadline failure for key and reports whether the
// key has now failed enough times in a row to be negative-cached.
func (d *DocsClient) noteTimeout(key string) bool {
	d.timeoutMu.Lock()
	defer d.timeoutMu.Unlock()
	if d.timeouts == nil {
		d.timeouts = make(map[string]int)
	}
	d.timeouts[key]++
	if d.timeouts[key] < maxConsecutiveTimeouts {
		return false
	}
	delete(d.timeouts, key)
	return true
}

func (d *DocsClient) clearTimeouts(key string) {
	d.timeoutMu.Lock()
	delete(d.timeouts, key)
	d.timeoutMu.Unlock()
}

// corpusValue coalesces manifest and shard loads across queries. Each caller
// can stop waiting independently; the shared load has its own bounded lifetime
// and does not negative-cache a single timeout. Slots bound concurrent corpus
// loads across the entire client, including loads whose callers have stopped
// waiting.
func (d *DocsClient) corpusValue(ctx context.Context, key string, ttl time.Duration, load func(context.Context) (any, error)) any {
	if ctx.Err() != nil {
		return nil
	}
	if cached, ok := d.cache.get(key); ok {
		return cached
	}
	result := d.corpusGroup.DoChan(key, func() (any, error) {
		if cached, ok := d.cache.get(key); ok {
			return cached, nil
		}
		loadCtx, cancel := context.WithTimeout(context.Background(), excerptBudget)
		defer cancel()
		select {
		case d.slots() <- struct{}{}:
			defer func() { <-d.corpusSlots }()
		case <-loadCtx.Done():
			return nil, loadCtx.Err()
		}
		value, err := load(loadCtx)
		if err != nil {
			timedOut := loadCtx.Err() != nil || errors.Is(err, context.Canceled) || errors.Is(err, context.DeadlineExceeded)
			if !timedOut || d.noteTimeout(key) {
				d.cache.set(key, nil, negativeCacheTTL)
			}
			return nil, err
		}
		d.clearTimeouts(key)
		d.cache.set(key, value, ttl)
		return value, nil
	})
	select {
	case <-ctx.Done():
		return nil
	case loaded := <-result:
		return loaded.Val
	}
}

// ContentExcerpts returns a query-focused excerpt from the body of each
// entry it can resolve, keyed by canonical URL. It is best effort: entries
// whose shard is missing, oversized, slow, or whose body does not mention
// any query term are simply absent from the result.
func (d *DocsClient) ContentExcerpts(ctx context.Context, entries []DocsEntry, query string) map[string]string {
	terms := parseQueryTerms(query)
	if len(terms) == 0 || len(entries) == 0 {
		return nil
	}

	ctx, cancel := context.WithTimeout(ctx, excerptBudget)
	defer cancel()

	manifest := d.getCorpusManifest(ctx)
	if manifest == nil {
		return nil
	}

	shards := groupByShard(manifest, entries)

	var mu sync.Mutex
	excerpts := make(map[string]string)
	group, ctx := errgroup.WithContext(ctx)
	group.SetLimit(shardConcurrency)
	for _, shard := range shards {
		if ctx.Err() != nil {
			break
		}
		group.Go(func() error {
			bodies := d.loadShard(ctx, shard.url)
			if bodies == nil {
				return nil
			}
			for _, url := range shard.pages {
				if ctx.Err() != nil {
					break
				}
				if text := excerpt(bodies[url], terms); text != "" {
					mu.Lock()
					excerpts[url] = text
					mu.Unlock()
				}
			}
			return nil
		})
	}
	_ = group.Wait()
	return excerpts
}

type corpusShard struct {
	url   string
	pages []string
}

// groupByShard preserves first-result order when batching pages into the first
// maxShardsPerQuery shards, so the highest-ranked results are scheduled first.
func groupByShard(manifest *corpusManifest, entries []DocsEntry) []corpusShard {
	var shards []corpusShard
	positions := make(map[string]int)
	for _, entry := range entries {
		page := canonicalDocURL(entry.URL)
		shard := manifest.shardFor(page)
		if shard == "" {
			continue
		}
		pos, seen := positions[shard]
		if !seen {
			if len(shards) >= maxShardsPerQuery {
				continue
			}
			pos = len(shards)
			positions[shard] = pos
			shards = append(shards, corpusShard{url: shard})
		}
		shards[pos].pages = append(shards[pos].pages, page)
	}
	return shards
}

// excerpt returns a query-focused window of at most excerptLength UTF-8 bytes
// (plus ellipses), preserving source case. Only the first maxHitsPerTerm
// occurrences per term in the retained body are considered, bounding CPU work.
func excerpt(content string, terms []queryTerm) string {
	type hit struct{ start, end, term int }
	var hits []hit
	terms = terms[:min(len(terms), maxQueryTerms)]
	counts := make([]int, len(terms))
	// Token positions come from the original text. Lowercasing individual tokens
	// for comparison cannot move the byte offsets used to quote the source.
	for _, span := range wordTokenRe.FindAllStringIndex(content, -1) {
		text := strings.ToLower(content[span[0]:span[1]])
		token := queryTerm{text: text, stem: lightStem(text)}
		for t, term := range terms {
			if counts[t] < maxHitsPerTerm && term.matchesToken(token) {
				hits = append(hits, hit{span[0], span[1], t})
				counts[t]++
			}
		}
	}
	if len(hits) == 0 {
		return ""
	}
	bestStart, bestEnd, bestCount := 0, 0, 0
	for _, anchor := range hits {
		start, end := windowBounds(content, anchor.start, anchor.end)
		var covered uint32 // parseQueryTerms caps terms at 16; also enforced above.
		for _, other := range hits {
			if other.start >= start && other.end <= end {
				covered |= 1 << other.term
			}
		}
		if count := popcount(covered); count > bestCount {
			bestStart, bestEnd, bestCount = start, end, count
		}
	}
	if bestCount == 0 {
		return ""
	}
	text := strings.Join(strings.Fields(content[bestStart:bestEnd]), " ")
	if bestStart > 0 {
		text = "..." + text
	}
	if bestEnd < len(content) {
		text += "..."
	}
	return text
}

func popcount(bits uint32) int {
	count := 0
	for ; bits != 0; bits &= bits - 1 {
		count++
	}
	return count
}

// windowBounds prefers whitespace boundaries but never discards the anchor
// match. Long tokens at either edge may be cut, always at UTF-8 boundaries.
// An anchor too large to fit yields an empty window and description fallback.
func windowBounds(content string, anchorStart, anchorEnd int) (int, int) {
	start := max(anchorStart-excerptLead, 0)
	for start < anchorStart && !utf8.RuneStart(content[start]) {
		start++
	}
	if start > 0 {
		for offset, r := range content[start:anchorStart] {
			if unicode.IsSpace(r) {
				start += offset + utf8.RuneLen(r)
				break
			}
		}
	}
	end := start + min(excerptLength, len(content)-start)
	for end < len(content) && end > start && !utf8.RuneStart(content[end]) {
		end--
	}
	if end < anchorEnd {
		return 0, 0
	}
	if end < len(content) {
		last := -1
		for offset, r := range content[anchorEnd:end] {
			if unicode.IsSpace(r) {
				last = anchorEnd + offset
			}
		}
		if last >= anchorEnd {
			end = last
		}
	}
	return start, end
}

// truncateUTF8 cuts s to at most limit bytes without splitting a rune.
func truncateUTF8(s string, limit int) string {
	if len(s) <= limit {
		return s
	}
	for limit > 0 && !utf8.RuneStart(s[limit]) {
		limit--
	}
	return s[:limit]
}
