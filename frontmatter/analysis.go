package frontmatter

import (
	"fmt"
	"os"

	"gopkg.in/yaml.v3"
)

type FileAnalysis struct {
	Path string

	Placement PlacementResult

	MissingProps     []string
	ExtraProps       []string
	OutOfOrder       bool
	MissingOrEmptyID bool
	HasFrontMatter   bool
}

func AnalyzeFile(path string, template map[string]interface{}) (FileAnalysis, error) {
	return FileAnalysis{}, ErrNotImplemented
}

func (fa FileAnalysis) HasIssues() bool {
	return !fa.Placement.Status.IsOK() ||
		len(fa.MissingProps) > 0 ||
		len(fa.ExtraProps) > 0 ||
		fa.OutOfOrder ||
		fa.MissingOrEmptyID
}

func processFile(path string, template map[string]interface{}) (FileAnalysis, error) {
	analysis := FileAnalysis{
		Path: path,
	}

	content, err := os.ReadFile(path)
	if err != nil {
		return analysis, fmt.Errorf("failed to read file: %w", err)
	}

	raw := string(content)

	placement := AuditFrontMatterPlacement(raw)
	analysis.Placement = placement
	analysis.HasFrontMatter = placement.Status.IsOK()

	if !placement.Status.IsOK() {
		return analysis, nil
	}

	frontMatter, err := ExtractFrontMatterBoundary(raw)
	if err != nil {
		return analysis, fmt.Errorf("failed to extract front matter: %w", err)
	}

	var frontMatterKVs map[string]interface{}
	err = yaml.Unmarshal([]byte(frontMatter), &frontMatterKVs)
	if err != nil {
		return analysis, fmt.Errorf("failed to parse YAML front matter: %w", err)
	}

	for key := range template {
		if _, exists := frontMatterKVs[key]; !exists {
			analysis.MissingProps = append(analysis.MissingProps, key)
		}
	}

	return analysis, nil
}
