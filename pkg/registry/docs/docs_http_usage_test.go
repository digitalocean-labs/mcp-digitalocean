package docs

import (
	"bytes"
	"encoding/json"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	"go.uber.org/mock/gomock"
)

var testSecret = []byte(strings.Repeat("s", MinClientIDSecretBytes))

func mustTagger(t *testing.T, now time.Time) *ClientTagger {
	t.Helper()
	tg, err := NewClientTagger(testSecret, cfIP)
	require.NoError(t, err)
	tg.now = func() time.Time { return now }
	return tg
}

func reqFrom(ip, ua string) *http.Request {
	req := httptest.NewRequest(http.MethodGet, "/v1/search?q=x", nil)
	if ip != "" {
		req.Header.Set(cfIP, ip)
	}
	if ua != "" {
		req.Header.Set("User-Agent", ua)
	}
	return req
}

func TestNewClientTagger_RejectsWeakConfig(t *testing.T) {
	_, err := NewClientTagger([]byte("short"), cfIP)
	require.Error(t, err)
	_, err = NewClientTagger(testSecret, " ")
	require.Error(t, err, "without a trusted header every client shares the edge address")
}

func TestClientTagger_StableWithinWeekRotatesAcrossWeeks(t *testing.T) {
	mon := time.Date(2026, 11, 9, 0, 0, 1, 0, time.UTC)
	sun := time.Date(2026, 11, 15, 23, 59, 59, 0, time.UTC)
	next := time.Date(2026, 11, 16, 0, 0, 1, 0, time.UTC)

	a := mustTagger(t, mon).ID(reqFrom("203.0.113.7", ""))
	require.Len(t, a, clientIDHexLen)
	require.NotContains(t, a, "203.0.113.7")
	require.Equal(t, a, mustTagger(t, sun).ID(reqFrom("203.0.113.7", "")), "same ISO week, same id")
	require.NotEqual(t, a, mustTagger(t, next).ID(reqFrom("203.0.113.7", "")), "new week, new id")
	require.NotEqual(t, a, mustTagger(t, mon).ID(reqFrom("203.0.113.8", "")), "different clients differ")
}

func TestClientTagger_SecretChangesID(t *testing.T) {
	now := time.Date(2026, 11, 12, 0, 0, 0, 0, time.UTC)
	other, err := NewClientTagger([]byte(strings.Repeat("t", MinClientIDSecretBytes)), cfIP)
	require.NoError(t, err)
	other.now = func() time.Time { return now }
	require.NotEqual(t, mustTagger(t, now).ID(reqFrom("203.0.113.7", "")), other.ID(reqFrom("203.0.113.7", "")))
}

func TestClientTagger_GroupsIPv6By64AndFlagsMissingIP(t *testing.T) {
	tg := mustTagger(t, time.Now())
	require.Equal(t, tg.ID(reqFrom("2001:db8:1:2::1", "")), tg.ID(reqFrom("2001:db8:1:2::ffff", "")))
	require.Equal(t, unidentifiedClient, tg.ID(reqFrom("", "")))
}

func TestUserAgentFamily(t *testing.T) {
	for ua, want := range map[string]string{
		"":                       "none",
		"curl/8.7.1":             "curl",
		"python-requests/2.32.3": "python-requests",
		"python-httpx/0.27.0":    "python-httpx",
		"node":                   "node",
		"Go-http-client/2.0":     "go",
		"Mozilla/5.0 AppleWebKit/537.36 (KHTML, like Gecko; compatible; ChatGPT-User/1.0; +https://openai.com/bot)": "chatgpt-user",
		"Mozilla/5.0 AppleWebKit/537.36 (KHTML, like Gecko; compatible; ClaudeBot/1.0; +claudebot@anthropic.com)":   "claudebot",
		"Claude-User": "claude-user",
		"Mozilla/5.0 (Macintosh; Intel Mac OS X 10_15_7) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/130.0 Safari/537.36": "browser",
		"acme-agent/0.1": "other",
	} {
		require.Equal(t, want, userAgentFamily(ua), ua)
	}
}

func TestCleanUserAgent(t *testing.T) {
	require.Equal(t, "evilforged", cleanUserAgent("evil\nforged"))
	require.LessOrEqual(t, len(cleanUserAgent(strings.Repeat("a", 500))), maxUserAgentLen)
}

func usageLogs(t *testing.T, buf *bytes.Buffer) []map[string]any {
	t.Helper()
	var out []map[string]any
	for _, line := range strings.Split(strings.TrimSpace(buf.String()), "\n") {
		var m map[string]any
		require.NoError(t, json.Unmarshal([]byte(line), &m))
		if m["msg"] == "docs search api" {
			out = append(out, m)
		}
	}
	return out
}

func TestSearchHandler_UsageLogCarriesClientFields(t *testing.T) {
	ctrl := gomock.NewController(t)
	m := NewMockDocsService(ctrl)
	m.EXPECT().GetDocsIndex().Return(searchTestIndex(), nil)
	m.EXPECT().ContentExcerpts(gomock.Any(), gomock.Any(), gomock.Any()).Return(map[string]string{})

	var buf bytes.Buffer
	h := newSearchHandler(m, slog.New(slog.NewJSONHandler(&buf, nil)))
	h.clients = mustTagger(t, time.Now())

	req := httptest.NewRequest(http.MethodGet, "/v1/search?q=create+droplet+sk-secret", nil)
	req.Header.Set(cfIP, "203.0.113.7")
	req.Header.Set("User-Agent", "curl/8.7.1")
	h.ServeHTTP(httptest.NewRecorder(), req)

	logs := usageLogs(t, &buf)
	require.Len(t, logs, 1)
	l := logs[0]
	require.EqualValues(t, http.StatusOK, l["status"])
	require.Equal(t, "curl", l["client_type"])
	require.Equal(t, "curl/8.7.1", l["user_agent"])
	require.Len(t, l["client_id"], clientIDHexLen)
	require.Equal(t, false, l["zero_results"])
	require.NotContains(t, buf.String(), "203.0.113.7", "raw IPs stay out of usage logs")
	require.NotContains(t, buf.String(), "sk-secret", "query text stays out of logs")
}

func TestSearchHandler_UsageLogCoversRejections(t *testing.T) {
	ctrl := gomock.NewController(t)
	var buf bytes.Buffer
	h := newSearchHandler(NewMockDocsService(ctrl), slog.New(slog.NewJSONHandler(&buf, nil)))

	for _, tt := range []struct {
		method, target string
		status         int
	}{
		{http.MethodGet, "/v1/search", http.StatusBadRequest},
		{http.MethodGet, "/v1/search?q=x&junk=1", http.StatusBadRequest},
		{http.MethodPost, "/v1/search?q=x", http.StatusMethodNotAllowed},
	} {
		doSearch(t, h, tt.method, tt.target)
	}
	doSearch(t, h, http.MethodOptions, "/v1/search")

	logs := usageLogs(t, &buf)
	require.Len(t, logs, 3, "preflights are not usage")
	require.EqualValues(t, http.StatusBadRequest, logs[0]["status"])
	require.Equal(t, "bad_request", logs[0]["error_id"])
	require.EqualValues(t, http.StatusMethodNotAllowed, logs[2]["status"])
	require.NotContains(t, logs[0], "client_id", "no tagger, no client_id")
}
