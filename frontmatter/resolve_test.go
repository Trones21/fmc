package frontmatter_test

import (
	"errors"
	"testing"
	"time"

	"github.com/Trones21/fmc/frontmatter"
)

func TestResolveValue(t *testing.T) {
	ctx := frontmatter.ResolveContext{
		FilePath:    "test.md",
		Content:     "some content",
		FrontMatter: map[string]any{"title": "Test"},
	}

	tests := []struct {
		name        string
		policy      frontmatter.PropertyPolicy
		expectedVal any
		expectedErr error
	}{
		{
			name: "static string value",
			policy: frontmatter.PropertyPolicy{
				Key:         "title",
				Source:      frontmatter.SourceStatic,
				StaticValue: "My Blog",
			},
			expectedVal: "My Blog",
			expectedErr: nil,
		},
		{
			name: "static nil value",
			policy: frontmatter.PropertyPolicy{
				Key:         "draft",
				Source:      frontmatter.SourceStatic,
				StaticValue: nil,
			},
			expectedVal: nil,
			expectedErr: nil,
		},
		{
			name: "computed unknown function returns error",
			policy: frontmatter.PropertyPolicy{
				Key:    "slug",
				Source: frontmatter.SourceComputed,
				Fn:     "notafunction",
			},
			expectedVal: nil,
			expectedErr: frontmatter.ErrUnknownFunction,
		},
		{
			name: "LLM source not implemented",
			policy: frontmatter.PropertyPolicy{
				Key:    "summary",
				Source: frontmatter.SourceLLM,
			},
			expectedVal: nil,
			expectedErr: frontmatter.ErrNotImplemented,
		},
		{
			name: "invalid source",
			policy: frontmatter.PropertyPolicy{
				Key:    "title",
				Source: frontmatter.ValueSource("bogus"),
			},
			expectedVal: nil,
			expectedErr: frontmatter.ErrInvalidSource,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			val, err := frontmatter.ResolveValue(tt.policy, ctx)

			if tt.expectedErr != nil {
				if !errors.Is(err, tt.expectedErr) {
					t.Errorf("expected error %v, got %v", tt.expectedErr, err)
				}
				return
			}
			if err != nil {
				t.Errorf("unexpected error: %v", err)
				return
			}
			if val != tt.expectedVal {
				t.Errorf("expected value %v, got %v", tt.expectedVal, val)
			}
		})
	}
}

// The rfc3339 transform exists because YAML hands back three different Go types
// for what a human wrote as a date, and one of them (int) is not a date to
// anything downstream. Exercised through ResolveValue — the path fmc actually
// takes for -createFrom ...:transform:rfc3339.
func resolveRFC3339(t *testing.T, in any) (any, error) {
	t.Helper()
	return frontmatter.ResolveValue(
		frontmatter.PropertyPolicy{
			Key:     "last_update.date",
			Source:  frontmatter.SourceTransform,
			Fn:      "rfc3339",
			FromKey: "last_update.date",
		},
		frontmatter.ResolveContext{
			FrontMatter: map[string]any{"last_update": map[string]any{"date": in}},
		},
	)
}

func TestResolveRFC3339Transform(t *testing.T) {
	cases := []struct {
		name string
		in   any
		want string
		err  bool
	}{
		{"unquoted YYYYMMDD arrives as int", 20250506, "2025-05-06T00:00:00Z", false},
		{"int64 too", int64(20250506), "2025-05-06T00:00:00Z", false},
		{"quoted YYYYMMDD string", "20250506", "2025-05-06T00:00:00Z", false},
		{"quoted ISO string", "2026-08-08", "2026-08-08T00:00:00Z", false},
		{"already RFC3339 is unchanged", "2026-08-08T00:00:00Z", "2026-08-08T00:00:00Z", false},
		{"offset normalises to UTC", "2026-08-08T02:00:00+02:00", "2026-08-08T00:00:00Z", false},
		{"slashed date", "2026/08/08", "2026-08-08T00:00:00Z", false},
		{"unquoted date arrives as time.Time",
			time.Date(2026, 8, 8, 0, 0, 0, 0, time.UTC), "2026-08-08T00:00:00Z", false},
		{"whitespace is tolerated", "  2026-08-08  ", "2026-08-08T00:00:00Z", false},
		{"empty string is an error", "", "", true},
		{"nonsense is an error", "last tuesday", "", true},
		{"a list is not a date", []any{"2026-08-08"}, "", true},
		{"nil is not a date", nil, "", true},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got, err := resolveRFC3339(t, tc.in)
			if tc.err {
				if err == nil {
					t.Fatalf("expected an error, got %v", got)
				}
				return
			}
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if got != tc.want {
				t.Fatalf("got %v, want %q", got, tc.want)
			}
		})
	}
}

// Running the transform over every file, including the ones already correct, is
// the normal way to use it — so applying it twice must not drift.
func TestResolveRFC3339IsIdempotent(t *testing.T) {
	for _, in := range []any{20250506, "2026-08-08", "2026-08-08T00:00:00Z"} {
		once, err := resolveRFC3339(t, in)
		if err != nil {
			t.Fatalf("first pass on %v: %v", in, err)
		}
		twice, err := resolveRFC3339(t, once)
		if err != nil {
			t.Fatalf("second pass on %v: %v", once, err)
		}
		if once != twice {
			t.Errorf("not idempotent for %v: %v then %v", in, once, twice)
		}
	}
}

func TestResolveRejectsUnknownTransform(t *testing.T) {
	_, err := frontmatter.ResolveValue(
		frontmatter.PropertyPolicy{Source: frontmatter.SourceTransform, Fn: "nope", FromKey: "a"},
		frontmatter.ResolveContext{FrontMatter: map[string]any{"a": "b"}},
	)
	if err == nil {
		t.Fatal("expected an error for an unknown transform")
	}
}
