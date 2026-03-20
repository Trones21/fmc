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

	trimmedPrefix := ""
	for i, line := range lines {
		if line == "---" {
			prefix := strings.Join(lines[:i], "\n")
			trimmedPrefix = strings.TrimSpace(prefix)

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
	}

	return PlacementResult{
		Status: PlacementMissing,
		Reason: "no front matter start boundary found",
	}
}
