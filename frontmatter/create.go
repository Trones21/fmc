package frontmatter

import (
	"fmt"
	"os"
	"strings"

	"gopkg.in/yaml.v3"
)

// FrontMatterCreationPlan describes the front matter to be prepended to a
// single file that currently has no front matter.
type FrontMatterCreationPlan struct {
	FilePath    string
	FrontMatter map[string]any // nil means this file should be skipped
	Preview     []string       // first N lines of the file body for display
}

func (p FrontMatterCreationPlan) ShouldCreate() bool {
	return p.FrontMatter != nil
}

// PlanFrontMatterCreation returns a plan for adding front matter to path if
// its content has PlacementMissing status. Files with any other placement
// status are skipped (ShouldCreate returns false). Only keys present in
// template are written; defaults supplies optional initial values.
func PlanFrontMatterCreation(path, content string, template map[string]any, defaults map[string]any, previewLines int) (FrontMatterCreationPlan, error) {
	plan := FrontMatterCreationPlan{FilePath: path}

	placement := AuditFrontMatterPlacement(content)
	if placement.Status != PlacementMissing {
		return plan, nil
	}

	fm := make(map[string]any, len(template))
	for key := range template {
		if val, ok := defaults[key]; ok {
			fm[key] = val
		} else {
			fm[key] = ""
		}
	}
	plan.FrontMatter = fm

	lines := strings.Split(content, "\n")
	if len(lines) > previewLines {
		lines = lines[:previewLines]
	}
	plan.Preview = lines

	return plan, nil
}

// ApplyFrontMatterCreation prepends the planned front matter block to the
// file, preserving the original body content.
func ApplyFrontMatterCreation(plan FrontMatterCreationPlan) error {
	content, err := os.ReadFile(plan.FilePath)
	if err != nil {
		return fmt.Errorf("failed to read file: %w", err)
	}

	fm, err := yaml.Marshal(plan.FrontMatter)
	if err != nil {
		return fmt.Errorf("failed to marshal front matter: %w", err)
	}

	raw := strings.ReplaceAll(string(content), "\r\n", "\n")
	raw = strings.ReplaceAll(raw, "\r", "\n")

	return os.WriteFile(plan.FilePath, []byte("---\n"+string(fm)+"---\n"+raw), 0644)
}
