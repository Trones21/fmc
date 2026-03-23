package frontmatter

import (
	"os"
	"strings"
)

type PlacementStatus string

const (
	PlacementOK             PlacementStatus = "ok"
	PlacementWhitespaceOnly PlacementStatus = "misplaced_whitespace_only"
	PlacementManualReview   PlacementStatus = "manual_review"
	PlacementMissing        PlacementStatus = "missing"
)

type PlacementResult struct {
	FilePath  string
	Status    PlacementStatus
	Candidate *BoundaryCandidate
	Reason    string
}

func (s PlacementStatus) IsOK() bool {
	return s == PlacementOK
}

func (s PlacementStatus) IsProcessable() bool {
	return s == PlacementOK || s == PlacementWhitespaceOnly
}

func (s PlacementStatus) IsFixable() bool {
	return s == PlacementWhitespaceOnly
}

func (s PlacementStatus) RequiresManualIntervention() bool {
	return s == PlacementManualReview
}

func AuditPlacementFiles(files []string) ([]PlacementResult, error) {
	results := make([]PlacementResult, 0, len(files))

	for _, file := range files {
		content, err := os.ReadFile(file)
		if err != nil {
			results = append(results, PlacementResult{
				FilePath: file,
				Reason:   err.Error(),
			})
			continue
		}

		result := AuditFrontMatterPlacement(string(content))
		result.FilePath = file
		results = append(results, result)
	}

	return results, nil
}

func AuditFrontMatterPlacement(content string) PlacementResult {
	content = strings.ReplaceAll(content, "\r\n", "\n")
	content = strings.ReplaceAll(content, "\r", "\n")

	lines := strings.Split(content, "\n")
	if len(lines) == 0 {
		return PlacementResult{
			Status: PlacementMissing,
			Reason: "empty file",
		}
	}

	if lines[0] == "---" {
		return PlacementResult{
			Status: PlacementOK,
			Reason: "front matter starts on line 1",
		}
	}

	for i, line := range lines {
		if line != "---" {
			continue
		}

		// Find the closing fence.
		closingIdx := -1
		for j := i + 1; j < len(lines); j++ {
			if lines[j] == "---" {
				closingIdx = j
				break
			}
		}

		// No closing fence, or the block between the fences contains no YAML
		// key:value lines — this "---" is a markdown separator, not front matter.
		if closingIdx == -1 || !blockContainsYAMLKeys(lines[i+1:closingIdx]) {
			continue
		}

		prefix := strings.Join(lines[:i], "\n")
		trimmedPrefix := strings.TrimSpace(prefix)

		candidate := &BoundaryCandidate{
			StartLine:    i + 1,
			StartColumn:  1,
			OpeningFence: "---",
			Status:       BoundaryValid,
		}

		if trimmedPrefix == "" {
			return PlacementResult{
				Status:    PlacementWhitespaceOnly,
				Candidate: candidate,
				Reason:    "only whitespace precedes candidate front matter",
			}
		}

		return PlacementResult{
			Status:    PlacementManualReview,
			Candidate: candidate,
			Reason:    "non-whitespace content precedes candidate front matter",
		}
	}

	return PlacementResult{
		Status: PlacementMissing,
		Reason: "no front matter start boundary found",
	}
}

// blockContainsYAMLKeys reports whether any line in the block looks like a
// YAML key (starts with a letter or underscore and contains a colon after
// a run of valid identifier characters).
func blockContainsYAMLKeys(lines []string) bool {
	for _, line := range lines {
		if isYAMLKeyLine(line) {
			return true
		}
	}
	return false
}

func isYAMLKeyLine(line string) bool {
	if len(line) == 0 {
		return false
	}
	ch := line[0]
	if !((ch >= 'a' && ch <= 'z') || (ch >= 'A' && ch <= 'Z') || ch == '_') {
		return false
	}
	colonIdx := strings.Index(line, ":")
	if colonIdx < 1 {
		return false
	}
	for _, c := range line[:colonIdx] {
		if !((c >= 'a' && c <= 'z') || (c >= 'A' && c <= 'Z') || (c >= '0' && c <= '9') || c == '_' || c == '-') {
			return false
		}
	}
	return true
}
