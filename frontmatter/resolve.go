package frontmatter

import (
	"crypto/rand"
	"fmt"
	"path/filepath"
	"regexp"
	"strings"
	"time"
)

type ValueSource string

const (
	SourceStatic    ValueSource = "static"
	SourceComputed  ValueSource = "computed"
	SourceTransform ValueSource = "transform"
	SourceLLM       ValueSource = "llm"
)

type ResolveContext struct {
	FilePath    string
	Content     string
	FrontMatter map[string]any
}

func dispatchComputed(policy PropertyPolicy, ctx ResolveContext) (any, error) {
	switch policy.Fn {
	case "today":
		return time.Now().Format("2006-01-02"), nil
	case "uuid":
		return generateUUID()
	case "path_segments":
		return pathSegmentTags(policy.Params, ctx)
	default:
		return nil, fmt.Errorf("%w: computed %q", ErrUnknownFunction, policy.Fn)
	}
}

func pathSegmentTags(params map[string]any, ctx ResolveContext) (any, error) {
	parts := strings.Split(filepath.ToSlash(ctx.FilePath), "/")

	// collect non-empty segments
	var segments []string
	for _, p := range parts {
		if p != "" {
			segments = append(segments, p)
		}
	}

	// drop first and last (root prefix and filename)
	if len(segments) <= 2 {
		return toStringSlice(ctx.FrontMatter["tags"]), nil
	}
	segments = segments[1 : len(segments)-1]

	// drop additional leading segments per "skip" param
	skip := 0
	if v, ok := params["skip"]; ok {
		if f, ok := v.(float64); ok { // JSON numbers decode as float64
			skip = int(f)
		}
	}
	if skip >= len(segments) {
		return toStringSlice(ctx.FrontMatter["tags"]), nil
	}
	segments = segments[skip:]

	// merge into existing tags, no duplicates
	existing := toStringSlice(ctx.FrontMatter["tags"])
	seen := make(map[string]bool, len(existing))
	for _, t := range existing {
		seen[t] = true
	}
	result := append([]string{}, existing...)
	for _, seg := range segments {
		if !seen[seg] {
			result = append(result, seg)
			seen[seg] = true
		}
	}
	return result, nil
}

func toStringSlice(v any) []string {
	if v == nil {
		return nil
	}
	switch val := v.(type) {
	case []any:
		out := make([]string, 0, len(val))
		for _, item := range val {
			if s, ok := item.(string); ok {
				out = append(out, s)
			}
		}
		return out
	case []string:
		return val
	case string:
		if val == "" {
			return nil
		}
		return []string{val}
	default:
		return nil
	}
}

func generateUUID() (string, error) {
	b := make([]byte, 16)
	if _, err := rand.Read(b); err != nil {
		return "", fmt.Errorf("failed to generate UUID: %w", err)
	}
	b[6] = (b[6] & 0x0f) | 0x40 // version 4
	b[8] = (b[8] & 0x3f) | 0x80 // variant bits
	return fmt.Sprintf("%08x-%04x-%04x-%04x-%012x",
		b[0:4], b[4:6], b[6:8], b[8:10], b[10:]), nil
}

var nonAlphanumDash = regexp.MustCompile(`[^a-z0-9-]+`)
var multipleDashes = regexp.MustCompile(`-{2,}`)

func toSlug(s string) string {
	s = strings.ToLower(strings.TrimSpace(s))
	s = strings.ReplaceAll(s, " ", "-")
	s = nonAlphanumDash.ReplaceAllString(s, "")
	s = multipleDashes.ReplaceAllString(s, "-")
	return strings.Trim(s, "-")
}

func dispatchTransform(fn, fromKey string, ctx ResolveContext) (any, error) {
	if fromKey == "" {
		return nil, fmt.Errorf("transform %q requires a \"from\" key", fn)
	}
	sourceVal, ok := nestedGet(ctx.FrontMatter, keyPath(fromKey))
	if !ok {
		return nil, fmt.Errorf("transform %q: source key %q not found in front matter", fn, fromKey)
	}

	switch fn {
	case "copy":
		return sourceVal, nil
	case "slug", "urlsafe":
		str, ok := sourceVal.(string)
		if !ok {
			return nil, fmt.Errorf("transform %q: source key %q is not a string", fn, fromKey)
		}
		return toSlug(str), nil
	default:
		return nil, fmt.Errorf("%w: transform %q", ErrUnknownFunction, fn)
	}
}

func ResolveValue(policy PropertyPolicy, ctx ResolveContext) (any, error) {
	switch policy.Source {
	case SourceStatic:
		return policy.StaticValue, nil

	case SourceComputed:
		return dispatchComputed(policy, ctx)

	case SourceTransform:
		return dispatchTransform(policy.Fn, policy.FromKey, ctx)

	case SourceLLM:
		return nil, ErrNotImplemented

	default:
		return nil, ErrInvalidSource
	}
}
