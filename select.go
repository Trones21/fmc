package main

import (
	"fmt"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/Trones21/fmc/frontmatter"
)

// selectClause is one parsed -selectBy expression: a front matter key (dot
// notation for nested keys), a comparison operator, and the value to compare
// against. Value is unused for the exists/missing operators.
type selectClause struct {
	Key   string
	Op    string
	Value string
	Raw   string
}

// canonical operator names, plus the symbolic aliases users reach for first.
var selectOps = map[string]string{
	"eq":       "eq",
	"=":        "eq",
	"==":       "eq",
	"ne":       "ne",
	"!=":       "ne",
	"gt":       "gt",
	">":        "gt",
	"gte":      "gte",
	">=":       "gte",
	"lt":       "lt",
	"<":        "lt",
	"lte":      "lte",
	"<=":       "lte",
	"contains": "contains",
	"exists":   "exists",
	"missing":  "missing",
}

// opNeedsValue reports whether the operator compares against an operand.
func opNeedsValue(op string) bool { return op != "exists" && op != "missing" }

// parseSelectBy parses the repeatable -selectBy flag values into clauses.
// Format: key:op:value  (value omitted for exists/missing). The value is the
// remainder of the string, so values containing ':' (e.g. RFC3339 timestamps)
// need no escaping.
func parseSelectBy(entries []string) ([]selectClause, error) {
	var clauses []selectClause
	for _, entry := range entries {
		parts := strings.SplitN(entry, ":", 3)
		if len(parts) < 2 {
			return nil, fmt.Errorf("-selectBy %q: expected key:op:value (e.g. date:gte:2024-01-01)", entry)
		}
		key := strings.TrimSpace(parts[0])
		if key == "" {
			return nil, fmt.Errorf("-selectBy %q: empty key", entry)
		}
		op, ok := selectOps[strings.TrimSpace(parts[1])]
		if !ok {
			return nil, fmt.Errorf("-selectBy %q: unknown operator %q (valid: eq, ne, gt, gte, lt, lte, contains, exists, missing)", entry, parts[1])
		}
		clause := selectClause{Key: key, Op: op, Raw: entry}
		if opNeedsValue(op) {
			if len(parts) != 3 {
				return nil, fmt.Errorf("-selectBy %q: operator %q requires a value (e.g. %s:%s:2024-01-01)", entry, op, key, op)
			}
			clause.Value = parts[2]
		} else if len(parts) == 3 && strings.TrimSpace(parts[2]) != "" {
			return nil, fmt.Errorf("-selectBy %q: operator %q does not take a value", entry, op)
		}
		clauses = append(clauses, clause)
	}
	return clauses, nil
}

// selectByDateLayouts are the layouts tried when deciding whether both sides
// of a comparison are dates. Longest/most specific first.
var selectByDateLayouts = []string{
	time.RFC3339,
	"2006-01-02T15:04:05",
	"2006-01-02 15:04:05",
	"2006-01-02",
	"2006/01/02",
	"20060102",
	"02-01-2006",
	"02/01/2006",
	"01/02/2006",
}

// parseSelectDate parses s as a date using the known layouts.
func parseSelectDate(s string) (time.Time, bool) {
	s = strings.TrimSpace(s)
	if s == "" {
		return time.Time{}, false
	}
	for _, layout := range selectByDateLayouts {
		if t, err := time.Parse(layout, s); err == nil {
			return t, true
		}
	}
	return time.Time{}, false
}

// fmValueToString renders a front matter scalar for comparison and display.
// yaml.v3 decodes unquoted ISO dates into time.Time, so those are normalized
// back to a date string.
func fmValueToString(v any) (string, bool) {
	switch t := v.(type) {
	case nil:
		return "", true
	case string:
		return t, true
	case bool:
		return strconv.FormatBool(t), true
	case int:
		return strconv.Itoa(t), true
	case int64:
		return strconv.FormatInt(t, 10), true
	case float64:
		return strconv.FormatFloat(t, 'f', -1, 64), true
	case time.Time:
		if t.Hour() == 0 && t.Minute() == 0 && t.Second() == 0 {
			return t.Format("2006-01-02"), true
		}
		return t.Format(time.RFC3339), true
	default:
		return "", false // list, map, or other non-scalar
	}
}

// compareSelectValues compares got against want, choosing the comparison mode
// from the operands: dates if both parse as dates, numbers if both parse as
// numbers, otherwise a case-sensitive string compare. Returns -1, 0, or 1.
func compareSelectValues(got, want string) int {
	if gt, ok := parseSelectDate(got); ok {
		if wt, ok := parseSelectDate(want); ok {
			switch {
			case gt.Before(wt):
				return -1
			case gt.After(wt):
				return 1
			default:
				return 0
			}
		}
	}
	if gn, err := strconv.ParseFloat(strings.TrimSpace(got), 64); err == nil {
		if wn, err := strconv.ParseFloat(strings.TrimSpace(want), 64); err == nil {
			switch {
			case gn < wn:
				return -1
			case gn > wn:
				return 1
			default:
				return 0
			}
		}
	}
	return strings.Compare(got, want)
}

// selectContains reports whether the front matter value contains want: list
// membership for sequences, substring match for scalars.
func selectContains(val any, want string) bool {
	if items, ok := val.([]any); ok {
		for _, item := range items {
			if s, ok := fmValueToString(item); ok && s == want {
				return true
			}
		}
		return false
	}
	s, ok := fmValueToString(val)
	return ok && strings.Contains(s, want)
}

// evalSelectClause reports whether a file's front matter satisfies the clause.
// The second return value is a short reason, used for -verbose reporting.
func evalSelectClause(fm map[string]any, c selectClause, onMissingInclude bool) (bool, string) {
	val, found := frontmatter.NestedGet(fm, frontmatter.KeyPath(c.Key))

	switch c.Op {
	case "exists":
		if found {
			return true, ""
		}
		return false, fmt.Sprintf("%s missing", c.Key)
	case "missing":
		if !found {
			return true, ""
		}
		return false, fmt.Sprintf("%s present", c.Key)
	}

	if !found || val == nil {
		if onMissingInclude {
			return true, ""
		}
		return false, fmt.Sprintf("%s missing", c.Key)
	}

	if c.Op == "contains" {
		if selectContains(val, c.Value) {
			return true, ""
		}
		return false, fmt.Sprintf("%s does not contain %q", c.Key, c.Value)
	}

	got, ok := fmValueToString(val)
	if !ok {
		return false, fmt.Sprintf("%s is not a scalar — cannot compare", c.Key)
	}

	cmp := compareSelectValues(got, c.Value)
	var pass bool
	switch c.Op {
	case "eq":
		pass = cmp == 0
	case "ne":
		pass = cmp != 0
	case "gt":
		pass = cmp > 0
	case "gte":
		pass = cmp >= 0
	case "lt":
		pass = cmp < 0
	case "lte":
		pass = cmp <= 0
	}
	if pass {
		return true, ""
	}
	return false, fmt.Sprintf("%s=%s fails %s %s", c.Key, got, c.Op, c.Value)
}

// filterSelectedFiles keeps only the files whose front matter satisfies the
// -selectBy clauses. With -selectByMode all (default) every clause must match;
// with any, at least one must. A one-line summary is always printed; -verbose
// adds a per-file table of what was excluded and why.
func (fmc *FrontMatterChecker) filterSelectedFiles(files []string) ([]string, error) {
	clauses, err := parseSelectBy(fmc.SelectBy)
	if err != nil {
		return nil, err
	}

	mode := strings.ToLower(strings.TrimSpace(fmc.SelectByMode))
	if mode == "" {
		mode = "all"
	}
	if mode != "all" && mode != "any" {
		return nil, fmt.Errorf("-selectByMode %q: expected 'all' or 'any'", fmc.SelectByMode)
	}

	onMissing := strings.ToLower(strings.TrimSpace(fmc.SelectByOnMissing))
	if onMissing == "" {
		onMissing = "exclude"
	}
	if onMissing != "exclude" && onMissing != "include" {
		return nil, fmt.Errorf("-selectByOnMissing %q: expected 'exclude' or 'include'", fmc.SelectByOnMissing)
	}
	onMissingInclude := onMissing == "include"

	type excludedEntry struct{ path, reason string }

	var kept []string
	var excluded []excludedEntry

	for _, file := range files {
		content, err := os.ReadFile(file)
		if err != nil {
			fmt.Printf("warning: could not read %s: %v\n", file, err)
			continue
		}
		fm, err := frontmatter.GetFrontMatterMap(string(content))
		if err != nil {
			excluded = append(excluded, excludedEntry{displayPath(file, fmc.PathKeep), "unparseable front matter"})
			continue
		}

		var reasons []string
		matched := 0
		for _, c := range clauses {
			ok, reason := evalSelectClause(fm, c, onMissingInclude)
			if ok {
				matched++
			} else {
				reasons = append(reasons, reason)
			}
		}

		pass := matched == len(clauses)
		if mode == "any" {
			pass = matched > 0
		}

		if pass {
			kept = append(kept, file)
		} else {
			excluded = append(excluded, excludedEntry{displayPath(file, fmc.PathKeep), strings.Join(reasons, "; ")})
		}
	}

	fmt.Printf("selectBy (%s of %d condition(s)): %d of %d file(s) matched, %d excluded.\n",
		mode, len(clauses), len(kept), len(files), len(excluded))

	if fmc.Verbose && len(excluded) > 0 {
		fmt.Println()
		tbl := NewTable("Excluded File", "Reason")
		for _, e := range excluded {
			tbl.AddRow(e.path, e.reason)
		}
		tbl.Print()
		fmt.Println()
	} else if len(excluded) > 0 {
		fmt.Println("(run with -verbose to see which files were excluded and why)")
	}

	return kept, nil
}
