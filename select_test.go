package main

import (
	"testing"
	"time"
)

func TestParseSelectBy(t *testing.T) {
	tests := []struct {
		name    string
		entry   string
		wantKey string
		wantOp  string
		wantVal string
		wantErr bool
	}{
		{name: "canonical op", entry: "date:gte:2024-01-01", wantKey: "date", wantOp: "gte", wantVal: "2024-01-01"},
		{name: "symbolic op", entry: "date:>=:2024-01-01", wantKey: "date", wantOp: "gte", wantVal: "2024-01-01"},
		{name: "equals alias", entry: "draft:=:false", wantKey: "draft", wantOp: "eq", wantVal: "false"},
		{name: "nested key", entry: "last_update.date:lt:2024-01-01", wantKey: "last_update.date", wantOp: "lt", wantVal: "2024-01-01"},
		{name: "value keeps colons", entry: "date:gt:2024-01-01T10:30:00Z", wantKey: "date", wantOp: "gt", wantVal: "2024-01-01T10:30:00Z"},
		{name: "exists takes no value", entry: "description:exists", wantKey: "description", wantOp: "exists"},
		{name: "missing takes no value", entry: "description:missing", wantKey: "description", wantOp: "missing"},
		{name: "unknown op", entry: "date:like:x", wantErr: true},
		{name: "value required", entry: "date:gte", wantErr: true},
		{name: "exists rejects value", entry: "date:exists:x", wantErr: true},
		{name: "empty key", entry: ":eq:x", wantErr: true},
		{name: "no op", entry: "date", wantErr: true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := parseSelectBy([]string{tt.entry})
			if tt.wantErr {
				if err == nil {
					t.Fatalf("expected error for %q, got %+v", tt.entry, got)
				}
				return
			}
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if len(got) != 1 {
				t.Fatalf("expected 1 clause, got %d", len(got))
			}
			c := got[0]
			if c.Key != tt.wantKey || c.Op != tt.wantOp || c.Value != tt.wantVal {
				t.Errorf("got {%q %q %q}, want {%q %q %q}",
					c.Key, c.Op, c.Value, tt.wantKey, tt.wantOp, tt.wantVal)
			}
		})
	}
}

func TestCompareSelectValues(t *testing.T) {
	tests := []struct {
		name      string
		got, want string
		expect    int
	}{
		{name: "dates less", got: "2023-01-05", want: "2024-01-01", expect: -1},
		{name: "dates greater", got: "2024-06-15", want: "2024-01-01", expect: 1},
		{name: "dates equal", got: "2024-01-01", want: "2024-01-01", expect: 0},
		{name: "mixed date formats", got: "20240615", want: "2024-01-01", expect: 1},
		{name: "rfc3339 vs date", got: "2025-03-01T10:30:00Z", want: "2024-01-01", expect: 1},
		{name: "numbers", got: "9", want: "10", expect: -1},
		{name: "numbers equal", got: "10.0", want: "10", expect: 0},
		{name: "strings", got: "apple", want: "banana", expect: -1},
		{name: "bools as strings", got: "false", want: "false", expect: 0},
		// A numeric-looking value against a date must not silently compare as
		// a number in one direction and a date in the other.
		{name: "date vs non-date falls back to string", got: "2024-01-01", want: "draft", expect: -1},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := compareSelectValues(tt.got, tt.want); got != tt.expect {
				t.Errorf("compare(%q, %q) = %d, want %d", tt.got, tt.want, got, tt.expect)
			}
		})
	}
}

func TestFmValueToString(t *testing.T) {
	tests := []struct {
		name   string
		val    any
		want   string
		wantOk bool
	}{
		{name: "string", val: "hello", want: "hello", wantOk: true},
		{name: "bool", val: false, want: "false", wantOk: true},
		{name: "int", val: 42, want: "42", wantOk: true},
		{name: "float", val: 3.5, want: "3.5", wantOk: true},
		{name: "yaml date", val: time.Date(2024, 6, 15, 0, 0, 0, 0, time.UTC), want: "2024-06-15", wantOk: true},
		{name: "yaml timestamp", val: time.Date(2024, 6, 15, 10, 30, 0, 0, time.UTC), want: "2024-06-15T10:30:00Z", wantOk: true},
		{name: "list is not scalar", val: []any{"a"}, wantOk: false},
		{name: "map is not scalar", val: map[string]any{"a": 1}, wantOk: false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, ok := fmValueToString(tt.val)
			if ok != tt.wantOk {
				t.Fatalf("ok = %v, want %v", ok, tt.wantOk)
			}
			if ok && got != tt.want {
				t.Errorf("got %q, want %q", got, tt.want)
			}
		})
	}
}

func TestEvalSelectClause(t *testing.T) {
	fm := map[string]any{
		"title": "A",
		"draft": false,
		"tags":  []any{"golang", "tutorial"},
		"last_update": map[string]any{
			"date": time.Date(2024, 6, 15, 0, 0, 0, 0, time.UTC),
		},
	}

	tests := []struct {
		name      string
		entry     string
		onMissing bool
		want      bool
	}{
		{name: "nested date gte pass", entry: "last_update.date:gte:2024-01-01", want: true},
		{name: "nested date gte fail", entry: "last_update.date:gte:2025-01-01", want: false},
		{name: "nested date lt pass", entry: "last_update.date:lt:2025-01-01", want: true},
		{name: "bool eq", entry: "draft:eq:false", want: true},
		{name: "bool ne", entry: "draft:ne:false", want: false},
		{name: "list contains", entry: "tags:contains:golang", want: true},
		{name: "list does not contain", entry: "tags:contains:python", want: false},
		{name: "substring contains", entry: "title:contains:A", want: true},
		{name: "exists", entry: "title:exists", want: true},
		{name: "exists on absent key", entry: "description:exists", want: false},
		{name: "missing on absent key", entry: "description:missing", want: true},
		{name: "missing on present key", entry: "title:missing", want: false},
		{name: "absent key excluded by default", entry: "description:eq:x", want: false},
		{name: "absent key included when configured", entry: "description:eq:x", onMissing: true, want: true},
		{name: "non-scalar cannot compare", entry: "tags:gte:a", want: false},
		{name: "onMissing does not affect exists", entry: "description:exists", onMissing: true, want: false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			clauses, err := parseSelectBy([]string{tt.entry})
			if err != nil {
				t.Fatalf("parse %q: %v", tt.entry, err)
			}
			got, reason := evalSelectClause(fm, clauses[0], tt.onMissing)
			if got != tt.want {
				t.Errorf("eval(%q) = %v (%s), want %v", tt.entry, got, reason, tt.want)
			}
			if !got && reason == "" {
				t.Errorf("eval(%q) failed without a reason", tt.entry)
			}
		})
	}
}
