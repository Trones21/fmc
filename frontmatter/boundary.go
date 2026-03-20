package frontmatter

import (
	"errors"
	"fmt"
	"strings"
)

type BoundaryStatus string

const (
	BoundaryValid      BoundaryStatus = "valid"
	BoundaryMalformed  BoundaryStatus = "malformed"
	BoundaryIncomplete BoundaryStatus = "incomplete"
)

type BoundaryCandidate struct {
	StartLine   int
	EndLine     int
	StartColumn int
	EndColumn   int

	OpeningFence string
	ClosingFence string

	Status BoundaryStatus
	Raw    string
}

func (s BoundaryStatus) IsValid() bool {
	return s == BoundaryValid
}

// extractFrontMatterBoundary extracts the front matter by reading up to the second ---.
func ExtractFrontMatterBoundary(content string) (string, error) {
	// Normalize line endings to \n to handle different platforms
	content = strings.ReplaceAll(content, "\r\n", "\n")
	content = strings.ReplaceAll(content, "\r", "\n")
	lines := strings.Split(content, "\n")

	if len(lines) < 2 || lines[0] != "---" {
		return "", fmt.Errorf("front matter start delimiter not found. First line: %s", lines[0])
	}

	var frontMatterLines []string
	for i := 1; i < len(lines); i++ {
		if strings.TrimSpace(lines[i]) == "---" {
			return strings.Join(frontMatterLines, "\n"), nil
		}
		frontMatterLines = append(frontMatterLines, lines[i])
	}

	return "", errors.New("front matter end delimiter not found")
}
