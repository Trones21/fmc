package main

import (
	"bytes"
	"encoding/json"
	"strings"
	"testing"
	"time"
)

func TestNewFindingEmitterRejectsUnknownMode(t *testing.T) {
	if _, err := NewFindingEmitter("yaml", &bytes.Buffer{}); err == nil {
		t.Fatal("expected an error for an unknown -json mode")
	}
}

func TestDisabledEmitterWritesNothing(t *testing.T) {
	var buf bytes.Buffer
	e, err := NewFindingEmitter(JSONOff, &buf)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if e.Enabled() {
		t.Fatal("empty mode should leave the emitter disabled")
	}
	e.Add(Finding{File: "a.md", Check: "checkFormat"})
	if err := e.Flush(); err != nil {
		t.Fatalf("flush: %v", err)
	}
	if buf.Len() != 0 {
		t.Fatalf("disabled emitter wrote %q", buf.String())
	}
	if e.Count() != 0 {
		t.Fatalf("disabled emitter counted %d findings", e.Count())
	}
}

// The streaming guarantee is the reason ndjson exists: a finding must be on the
// wire before the run ends, so an interrupted scan still yields everything it
// had found.
func TestNDJSONWritesOnAddNotOnFlush(t *testing.T) {
	var buf bytes.Buffer
	e, _ := NewFindingEmitter(JSONLines, &buf)

	e.Add(Finding{File: "a.md", Check: "checkFormat", Key: "id"})
	if buf.Len() == 0 {
		t.Fatal("ndjson buffered instead of streaming")
	}
	e.Add(Finding{File: "b.md", Check: "checkFormat", Key: "id"})

	lines := strings.Split(strings.TrimSpace(buf.String()), "\n")
	if len(lines) != 2 {
		t.Fatalf("expected 2 lines, got %d: %q", len(lines), buf.String())
	}
	for i, line := range lines {
		var f Finding
		if err := json.Unmarshal([]byte(line), &f); err != nil {
			t.Fatalf("line %d is not valid JSON on its own: %v", i, err)
		}
	}
}

// The array guarantee is the mirror image: one parseable document, nothing
// before it, so a consumer expecting exactly one JSON value is satisfied.
func TestArrayBuffersUntilFlush(t *testing.T) {
	var buf bytes.Buffer
	e, _ := NewFindingEmitter(JSONArray, &buf)

	e.Add(Finding{File: "a.md", Check: "checkType", Key: "draft"})
	if buf.Len() != 0 {
		t.Fatalf("array mode wrote before Flush: %q", buf.String())
	}
	if err := e.Flush(); err != nil {
		t.Fatalf("flush: %v", err)
	}

	var out []Finding
	if err := json.Unmarshal(buf.Bytes(), &out); err != nil {
		t.Fatalf("array output is not a single JSON document: %v", err)
	}
	if len(out) != 1 || out[0].File != "a.md" {
		t.Fatalf("unexpected findings: %+v", out)
	}
}

// An empty run must still be parseable, so consumers never special-case silence.
func TestArrayEmitsEmptyArrayWhenNothingFound(t *testing.T) {
	var buf bytes.Buffer
	e, _ := NewFindingEmitter(JSONArray, &buf)
	if err := e.Flush(); err != nil {
		t.Fatalf("flush: %v", err)
	}
	var out []Finding
	if err := json.Unmarshal(buf.Bytes(), &out); err != nil {
		t.Fatalf("empty run produced unparseable output %q: %v", buf.String(), err)
	}
	if len(out) != 0 {
		t.Fatalf("expected no findings, got %d", len(out))
	}
}

func TestSeverityDefaultsToErrorAndIsCounted(t *testing.T) {
	var buf bytes.Buffer
	e, _ := NewFindingEmitter(JSONLines, &buf)
	e.Add(Finding{File: "a.md", Check: "parse"})
	e.Add(Finding{File: "b.md", Check: "listMissingProps", Severity: "warn"})

	if e.Count() != 2 {
		t.Fatalf("Count() = %d, want 2", e.Count())
	}
	if e.ErrorCount() != 1 {
		t.Fatalf("ErrorCount() = %d, want 1", e.ErrorCount())
	}

	first := strings.Split(strings.TrimSpace(buf.String()), "\n")[0]
	var f Finding
	if err := json.Unmarshal([]byte(first), &f); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if f.Severity != "error" {
		t.Fatalf("severity = %q, want the default %q", f.Severity, "error")
	}
}

func TestOptionalFieldsAreOmitted(t *testing.T) {
	var buf bytes.Buffer
	e, _ := NewFindingEmitter(JSONLines, &buf)
	e.Add(Finding{File: "a.md", Check: "parse", Severity: "error", Detail: "no front matter"})

	var raw map[string]any
	if err := json.Unmarshal(buf.Bytes(), &raw); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	for _, k := range []string{"key", "value", "expected"} {
		if _, present := raw[k]; present {
			t.Errorf("empty field %q should have been omitted", k)
		}
	}
}

// Regression: YAML parses an unquoted ISO date into time.Time and an unquoted
// YYYYMMDD into an int, so a type assertion to string reported correctly
// formatted dates as violations.
func TestFormatCheckStringHandlesNonStringScalars(t *testing.T) {
	layout := "2006-01-02"

	cases := []struct {
		name string
		in   any
		want string
		ok   bool
	}{
		{"string passes through", "2026-08-07", "2026-08-07", true},
		{"unquoted date arrives as time.Time",
			time.Date(2026, 8, 7, 0, 0, 0, 0, time.UTC), "2026-08-07", true},
		{"unquoted YYYYMMDD arrives as int", 20250506, "20250506", true},
		{"list is not checkable", []any{"a"}, "", false},
		{"map is not checkable", map[string]any{"a": 1}, "", false},
		{"nil is not checkable", nil, "", false},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got, ok := formatCheckString(tc.in, layout)
			if ok != tc.ok {
				t.Fatalf("ok = %v, want %v", ok, tc.ok)
			}
			if got != tc.want {
				t.Fatalf("got %q, want %q", got, tc.want)
			}
		})
	}
}

func TestFormatCheckStringFeedsValidateFormat(t *testing.T) {
	layout := "2006-01-02"

	// A correctly written unquoted date must conform...
	s, ok := formatCheckString(time.Date(2026, 8, 7, 0, 0, 0, 0, time.UTC), layout)
	if !ok || !validateFormat("YYYY-MM-DD", layout, s) {
		t.Errorf("unquoted ISO date should conform to YYYY-MM-DD, got %q", s)
	}

	// ...and an unquoted YYYYMMDD must not.
	s, ok = formatCheckString(20250506, layout)
	if !ok {
		t.Fatal("int should be checkable")
	}
	if validateFormat("YYYY-MM-DD", layout, s) {
		t.Errorf("20250506 should not conform to YYYY-MM-DD")
	}
}
