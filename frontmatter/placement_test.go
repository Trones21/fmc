package frontmatter_test

import (
	"testing"

	"github.com/Trones21/fmc/frontmatter"
)

func TestAuditFrontMatterPlacement(t *testing.T) {
	tests := []struct {
		name           string
		content        string
		expectedStatus frontmatter.PlacementStatus
	}{
		{
			name:           "front matter on first line",
			content:        "---\ntitle: Test\n---\ncontent",
			expectedStatus: frontmatter.PlacementOK,
		},
		{
			name:           "whitespace before front matter",
			content:        "\n\n---\ntitle: Test\n---\ncontent",
			expectedStatus: frontmatter.PlacementWhitespaceOnly,
		},
		{
			name:           "non-whitespace content before front matter",
			content:        "some content\n---\ntitle: Test\n---",
			expectedStatus: frontmatter.PlacementManualReview,
		},
		{
			name:           "no front matter at all",
			content:        "just some markdown content",
			expectedStatus: frontmatter.PlacementMissing,
		},
		{
			name:           "markdown separator dashes not mistaken for front matter",
			content:        "# Heading\n\nSome content.\n\n---\n\n## Section Two\n",
			expectedStatus: frontmatter.PlacementMissing,
		},
		{
			name:           "multiple markdown separators before real front matter candidate",
			content:        "# Title\n\n---\n\n## Section\n\ncontent\n\n---\ntitle: Real FM\nid: abc\n---\nbody",
			expectedStatus: frontmatter.PlacementManualReview,
		},
		{
			name:           "separator with no closing fence is skipped",
			content:        "intro\n\n---\n\n## Just markdown, no second fence",
			expectedStatus: frontmatter.PlacementMissing,
		},
		{
			name:           "empty content",
			content:        "",
			expectedStatus: frontmatter.PlacementMissing,
		},
		{
			name:           "CRLF line endings normalized",
			content:        "---\r\ntitle: Test\r\n---\r\ncontent",
			expectedStatus: frontmatter.PlacementOK,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := frontmatter.AuditFrontMatterPlacement(tt.content)
			if result.Status != tt.expectedStatus {
				t.Errorf("expected status %q, got %q (reason: %s)", tt.expectedStatus, result.Status, result.Reason)
			}
		})
	}
}

func TestPlacementStatusMethods(t *testing.T) {
	tests := []struct {
		status               frontmatter.PlacementStatus
		isOK                 bool
		isProcessable        bool
		isFixable            bool
		requiresManualReview bool
	}{
		{frontmatter.PlacementOK, true, true, false, false},
		{frontmatter.PlacementWhitespaceOnly, false, true, true, false},
		{frontmatter.PlacementManualReview, false, false, false, true},
		{frontmatter.PlacementMissing, false, false, false, false},
	}

	for _, tt := range tests {
		t.Run(string(tt.status), func(t *testing.T) {
			if got := tt.status.IsOK(); got != tt.isOK {
				t.Errorf("IsOK(): expected %v, got %v", tt.isOK, got)
			}
			if got := tt.status.IsProcessable(); got != tt.isProcessable {
				t.Errorf("IsProcessable(): expected %v, got %v", tt.isProcessable, got)
			}
			if got := tt.status.IsFixable(); got != tt.isFixable {
				t.Errorf("IsFixable(): expected %v, got %v", tt.isFixable, got)
			}
			if got := tt.status.RequiresManualIntervention(); got != tt.requiresManualReview {
				t.Errorf("RequiresManualIntervention(): expected %v, got %v", tt.requiresManualReview, got)
			}
		})
	}
}
