package docs

import (
	neturl "net/url"
	"regexp"
	"strings"
)

// Console, pricing and signup links in served pages carry agent UTM tags so a
// person who follows a link an agent gave them is attributed to it at signup.
// The site's markdown mirrors already tag these links with utm_medium=md; pages
// served through this server are retagged as mcp so the two surfaces stay
// distinguishable. Links tagged by anyone else are left alone.
const (
	agentUTMSource = "docs-agent"
	agentUTMMedium = "mcp"
)

var (
	agentCTALink   = regexp.MustCompile(`\]\((https?://(?:cloud\.digitalocean\.com|(?:www\.)?digitalocean\.com/(?:pricing|try|sign-?up))[^)\s]*)`)
	mirrorUTMParam = regexp.MustCompile(`([?&])utm_source=docs-agent&utm_medium=md\b`)
)

// tagAgentLinks retags or tags console and signup link targets in markdown
// prose. Callers pass text outside code, so endpoint URLs in examples are never
// touched.
func tagAgentLinks(text string) string {
	if !strings.Contains(text, "digitalocean.com") {
		return text
	}
	return agentCTALink.ReplaceAllStringFunc(text, func(match string) string {
		return "](" + tagAgentLink(match[2:])
	})
}

// tagAgentLink sets utm_source and utm_medium on one URL, keeping any query
// string and fragment. Console API, OAuth and SSO callback paths, and URLs
// tagged by another source, are returned unchanged.
func tagAgentLink(raw string) string {
	u, err := neturl.Parse(raw)
	if err != nil || u.Host == "" {
		return raw
	}
	for _, p := range []string{"/v1/", "/v2/", "/api/", "/sessions/"} {
		if strings.HasPrefix(u.Path, p) {
			return raw
		}
	}
	if mirrorUTMParam.MatchString(raw) {
		return mirrorUTMParam.ReplaceAllString(raw, "${1}utm_source="+agentUTMSource+"&utm_medium="+agentUTMMedium)
	}
	if strings.Contains(u.RawQuery, "utm_source=") {
		return raw
	}
	base, frag := raw, ""
	if i := strings.IndexByte(raw, '#'); i >= 0 {
		base, frag = raw[:i], raw[i:]
	}
	sep := "?"
	if strings.Contains(base, "?") {
		sep = "&"
		if strings.HasSuffix(base, "?") || strings.HasSuffix(base, "&") {
			sep = ""
		}
	}
	return base + sep + "utm_source=" + agentUTMSource + "&utm_medium=" + agentUTMMedium + frag
}
