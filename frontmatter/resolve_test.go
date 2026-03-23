package frontmatter_test

import (
	"errors"
	"testing"

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
