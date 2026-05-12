package docs

import (
	"encoding/json"
	"regexp"
	"strings"
)

var (
	fenceDoctl = regexp.MustCompile("(?is)```(?:doctl)?\\s*\\n([^`]+)```")
	fenceShell = regexp.MustCompile("(?is)```(?:bash|sh|shell)?\\s*\\n([^`]+)```")
	curlAPI    = regexp.MustCompile(`(?m)curl\s[^\\\n]*api\.digitalocean\.com[^\\\n]*`)
)

// Action is a machine-oriented step extracted from documentation markdown.
type Action struct {
	Method       string          `json:"method"`
	Command      string          `json:"command,omitempty"`
	Endpoint     string          `json:"endpoint,omitempty"`
	BodyTemplate json.RawMessage `json:"body_template,omitempty"`
	Path         string          `json:"path,omitempty"`
}

// ExtractActions pulls doctl commands, curl calls to the public API, and simple control panel paths from markdown.
func ExtractActions(md string) []Action {
	var actions []Action
	seen := make(map[string]struct{})

	for _, m := range fenceDoctl.FindAllStringSubmatch(md, -1) {
		if len(m) < 2 {
			continue
		}
		lines := strings.Split(strings.TrimSpace(m[1]), "\n")
		for _, line := range lines {
			line = strings.TrimSpace(line)
			if strings.HasPrefix(line, "doctl ") {
				key := "doctl:" + line
				if _, ok := seen[key]; ok {
					continue
				}
				seen[key] = struct{}{}
				actions = append(actions, Action{Method: "doctl", Command: line})
			}
		}
	}

	for _, m := range fenceShell.FindAllStringSubmatch(md, -1) {
		if len(m) < 2 {
			continue
		}
		// Join backslash-continued lines before splitting
		block := joinContinuationLines(m[1])
		for _, line := range strings.Split(block, "\n") {
			line = strings.TrimSpace(line)
			if strings.HasPrefix(strings.ToLower(line), "curl ") && strings.Contains(line, "api.digitalocean.com") {
				key := "curl:" + line
				if _, ok := seen[key]; ok {
					continue
				}
				seen[key] = struct{}{}
				ep := extractAPIEndpoint(line)
				actions = append(actions, Action{Method: "api", Command: line, Endpoint: ep})
			}
		}
	}

	for _, m := range curlAPI.FindAllString(md, -1) {
		line := strings.TrimSpace(m)
		key := "curl:" + line
		if _, ok := seen[key]; ok {
			continue
		}
		seen[key] = struct{}{}
		ep := extractAPIEndpoint(line)
		actions = append(actions, Action{Method: "api", Command: line, Endpoint: ep})
	}

	return actions
}

// joinContinuationLines merges lines ending with \ into a single line.
func joinContinuationLines(block string) string {
	lines := strings.Split(block, "\n")
	var merged []string
	var current strings.Builder
	for _, line := range lines {
		trimmed := strings.TrimRight(line, " \t")
		if strings.HasSuffix(trimmed, `\`) {
			current.WriteString(strings.TrimSuffix(trimmed, `\`))
			current.WriteByte(' ')
		} else {
			current.WriteString(line)
			merged = append(merged, current.String())
			current.Reset()
		}
	}
	if current.Len() > 0 {
		merged = append(merged, current.String())
	}
	return strings.Join(merged, "\n")
}

func extractAPIEndpoint(curlLine string) string {
	lower := strings.ToLower(curlLine)
	idx := strings.Index(lower, "api.digitalocean.com")
	if idx < 0 {
		return ""
	}
	rest := curlLine[idx:]
	rest = strings.TrimPrefix(rest, "api.digitalocean.com")
	rest = strings.TrimSpace(rest)
	if i := strings.IndexAny(rest, " ?"); i > 0 {
		rest = rest[:i]
	}
	rest = strings.Trim(rest, `"'`+"`")
	rest = strings.TrimPrefix(rest, "/")
	if !strings.HasPrefix(rest, "v2/") {
		return ""
	}
	return "/" + strings.TrimRight(strings.TrimSpace(rest), `"'`+"`")
}
