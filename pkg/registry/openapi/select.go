package openapi

import (
	"fmt"
	"strconv"
	"strings"
)

// selectJSON returns the value at path within JSON-decoded data (map[string]any, []any, primitives).
// Path syntax:
//   - dot-separated keys: droplets.id (nested maps)
//   - bracket indices: droplets[0]
//   - wildcards over arrays: droplets[*].id
// Empty path returns data unchanged.
func selectJSON(path string, data any) (any, error) {
	path = strings.TrimSpace(path)
	if path == "" {
		return data, nil
	}
	v, err := selectAt(data, path)
	if err != nil {
		return nil, fmt.Errorf("select %q: %w", path, err)
	}
	return v, nil
}

func selectAt(cur any, path string) (any, error) {
	path = strings.TrimSpace(path)
	if path == "" {
		return cur, nil
	}

	path = strings.TrimLeft(path, ".")
	if path == "" {
		return cur, nil
	}

	// Index or wildcard directly on the current value (e.g. rest is "[*].id" after a key).
	if path[0] == '[' {
		seg, rest, err := parseBracketSegment(path)
		if err != nil {
			return nil, err
		}
		switch seg.kind {
		case segIndex:
			arr, ok := cur.([]any)
			if !ok {
				return nil, fmt.Errorf("expected array for index [%d]", seg.idx)
			}
			if seg.idx < 0 || seg.idx >= len(arr) {
				return nil, fmt.Errorf("index [%d] out of range (length %d)", seg.idx, len(arr))
			}
			return selectAt(arr[seg.idx], rest)
		case segWildcard:
			arr, ok := cur.([]any)
			if !ok {
				return nil, fmt.Errorf("expected array for [*]")
			}
			if strings.TrimSpace(rest) == "" {
				out := make([]any, len(arr))
				copy(out, arr)
				return out, nil
			}
			out := make([]any, 0, len(arr))
			for _, el := range arr {
				v, err := selectAt(el, rest)
				if err != nil {
					return nil, err
				}
				out = append(out, v)
			}
			return out, nil
		default:
			return nil, fmt.Errorf("internal error: bracket segment")
		}
	}

	seg, rest, err := nextSelectSegment(path)
	if err != nil {
		return nil, err
	}

	switch seg.kind {
	case segKey:
		m, ok := cur.(map[string]any)
		if !ok {
			return nil, fmt.Errorf("expected object for key %q", seg.key)
		}
		v, ok := m[seg.key]
		if !ok {
			return nil, fmt.Errorf("missing key %q", seg.key)
		}
		return selectAt(v, rest)

	case segIndex:
		arr, ok := cur.([]any)
		if !ok {
			return nil, fmt.Errorf("expected array for index [%d]", seg.idx)
		}
		if seg.idx < 0 || seg.idx >= len(arr) {
			return nil, fmt.Errorf("index [%d] out of range (length %d)", seg.idx, len(arr))
		}
		return selectAt(arr[seg.idx], rest)

	case segWildcard:
		arr, ok := cur.([]any)
		if !ok {
			return nil, fmt.Errorf("expected array for [*]")
		}
		if strings.TrimSpace(rest) == "" {
			out := make([]any, len(arr))
			copy(out, arr)
			return out, nil
		}
		out := make([]any, 0, len(arr))
		for _, el := range arr {
			v, err := selectAt(el, rest)
			if err != nil {
				return nil, err
			}
			out = append(out, v)
		}
		return out, nil

	default:
		return nil, fmt.Errorf("internal error: unknown segment kind")
	}
}

type segKind int

const (
	segKey segKind = iota
	segIndex
	segWildcard
)

type selectSegment struct {
	kind segKind
	key  string
	idx  int
}

func nextSelectSegment(path string) (selectSegment, string, error) {
	path = strings.TrimLeft(path, ".")
	if path == "" {
		return selectSegment{}, "", fmt.Errorf("empty path segment")
	}

	if path[0] == '[' {
		return parseBracketSegment(path)
	}

	i := 0
	for i < len(path) && path[i] != '.' && path[i] != '[' {
		i++
	}
	if i == 0 {
		return selectSegment{}, "", fmt.Errorf("invalid path near %q", path)
	}
	key := path[:i]
	rest := path[i:]
	return selectSegment{kind: segKey, key: key}, rest, nil
}

func parseBracketSegment(path string) (selectSegment, string, error) {
	if len(path) < 3 || path[0] != '[' {
		return selectSegment{}, "", fmt.Errorf("invalid bracket segment")
	}
	endIdx := strings.IndexByte(path, ']')
	if endIdx < 0 {
		return selectSegment{}, "", fmt.Errorf("unclosed '[' in path")
	}
	inner := strings.TrimSpace(path[1:endIdx])
	rest := path[endIdx+1:]
	switch inner {
	case "*":
		return selectSegment{kind: segWildcard}, rest, nil
	default:
		idx, err := strconv.Atoi(inner)
		if err != nil {
			return selectSegment{}, "", fmt.Errorf("invalid index %q: %w", inner, err)
		}
		return selectSegment{kind: segIndex, idx: idx}, rest, nil
	}
}
