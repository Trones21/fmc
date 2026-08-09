package frontmatter

import (
	"crypto/rand"
	"fmt"
	"path/filepath"
	"regexp"
	"strconv"
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

// ExtractPathSegments returns the inner path segments of filePath: drops the
// first (root/prefix) and last (filename) segment, then drops an additional
// skip leading segments. Returns nil when there are no meaningful segments.
func ExtractPathSegments(filePath string, skip int) []string {
	parts := strings.Split(filepath.ToSlash(filePath), "/")
	var segments []string
	for _, p := range parts {
		if p != "" {
			segments = append(segments, p)
		}
	}
	if len(segments) <= 2 {
		return nil
	}
	segments = segments[1 : len(segments)-1]
	if skip >= len(segments) {
		return nil
	}
	return segments[skip:]
}

// ToStringSlice coerces a YAML-decoded value into a []string.
func ToStringSlice(v any) []string { return toStringSlice(v) }

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
	case "rfc3339":
		return toRFC3339(sourceVal, fromKey)
	default:
		return nil, fmt.Errorf("%w: transform %q", ErrUnknownFunction, fn)
	}
}

// dateInputLayouts are the serialisations seen in the wild, tried in order.
// RFC3339 is first so an already-correct value costs one parse and comes back
// unchanged — the transform has to be idempotent, because the normal way to run
// it is over every file including the ones already fine.
var dateInputLayouts = []string{
	time.RFC3339,
	"2006-01-02T15:04:05",
	"2006-01-02",
	"20060102",
	"2006/01/02",
	"02-01-2006",
	"02/01/2006",
}

// toRFC3339 normalises a date to RFC3339 in UTC.
//
// It has to cope with what YAML hands back rather than what was written. An
// unquoted 2026-08-08 arrives as a time.Time, a quoted "2026-08-08" as a string,
// and an unquoted 20250506 as an int — the last being the case this exists for,
// since a bare integer is not a date to anything downstream and sorts as a
// number.
//
// Dates without a time component become midnight UTC. That is an invention, but
// a harmless one: the source never carried a time, and every consumer of these
// fields works at day granularity.
func toRFC3339(v any, fromKey string) (string, error) {
	switch t := v.(type) {
	case time.Time:
		return t.UTC().Format(time.RFC3339), nil
	case int:
		return parseDateString(strconv.Itoa(t), fromKey)
	case int64:
		return parseDateString(strconv.FormatInt(t, 10), fromKey)
	case string:
		if strings.TrimSpace(t) == "" {
			return "", fmt.Errorf("transform \"rfc3339\": source key %q is empty", fromKey)
		}
		return parseDateString(t, fromKey)
	default:
		return "", fmt.Errorf("transform \"rfc3339\": source key %q is %T, not a date", fromKey, v)
	}
}

func parseDateString(s, fromKey string) (string, error) {
	s = strings.TrimSpace(s)
	for _, layout := range dateInputLayouts {
		if parsed, err := time.Parse(layout, s); err == nil {
			return parsed.UTC().Format(time.RFC3339), nil
		}
	}
	return "", fmt.Errorf("transform \"rfc3339\": source key %q value %q does not parse as a date", fromKey, s)
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
