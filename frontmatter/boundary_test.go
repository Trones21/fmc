package frontmatter_test

import (
	"testing"

	"github.com/Trones21/fmc/frontmatter"
)

func TestExtractFrontMatterBoundary(t *testing.T) {
	tests := []struct {
		name      string
		content   string
		expected  string
		expectErr bool
	}{
		{
			name: "valid front matter",
			content: "---\ntitle: Test\ndate: 2024-12-21\n---\n<the contents of the article>",
			expected:  "title: Test\ndate: 2024-12-21",
			expectErr: false,
		},
		{
			name: "valid front matter with trailing whitespace on closing fence",
			content: "---\ntitle: Test\ndate: 2024-12-21\n---   \n<the contents of the article>",
			expected:  "title: Test\ndate: 2024-12-21",
			expectErr: false,
		},
		{
			name:      "empty front matter",
			content:   "---\n---\ncontent here",
			expected:  "",
			expectErr: false,
		},
		{
			name:      "missing end delimiter",
			content:   "---\ntitle: Missing End",
			expected:  "",
			expectErr: true,
		},
		{
			name:      "no start delimiter",
			content:   "No Delimiters Here",
			expected:  "",
			expectErr: true,
		},
		{
			name:      "CRLF line endings",
			content:   "---\r\ntitle: Test\r\ndate: 2024-12-21\r\n---\r\ncontent",
			expected:  "title: Test\ndate: 2024-12-21",
			expectErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := frontmatter.ExtractFrontMatterBoundary(tt.content)
			if tt.expectErr {
				if err == nil {
					t.Errorf("expected error but got none")
				}
				return
			}
			if err != nil {
				t.Errorf("unexpected error: %v", err)
				return
			}
			if result != tt.expected {
				t.Errorf("expected:\n%q\ngot:\n%q", tt.expected, result)
			}
		})
	}
}
