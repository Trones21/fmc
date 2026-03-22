package frontmatter

import (
	"fmt"
	"os"
	"sort"

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

// PropNode is a single key found while walking a property's YAML value.
type PropNode struct {
	Key   string
	Depth int // 1 = direct child of the property, 2 = grandchild, etc.
}

// PropertyInspection is the result of inspecting one property in one file.
type PropertyInspection struct {
	PropKey  string
	Present  bool
	IsScalar bool // true when the value is not a map or slice (depth 0)
	MaxDepth int
	Nodes    []PropNode
}

// InspectProperty walks the YAML value of propKey and returns its structure.
func InspectProperty(content string, propKey string) (PropertyInspection, error) {
	result := PropertyInspection{PropKey: propKey}

	fmRaw, err := ExtractFrontMatterBoundary(content)
	if err != nil {
		return result, err
	}

	var fm map[string]any
	if err := yaml.Unmarshal([]byte(fmRaw), &fm); err != nil {
		return result, fmt.Errorf("failed to parse YAML: %w", err)
	}

	val, exists := fm[propKey]
	if !exists {
		return result, nil
	}
	result.Present = true
	result.Nodes = collectNodes(val, 1)
	if len(result.Nodes) == 0 {
		result.IsScalar = true
	}
	for _, n := range result.Nodes {
		if n.Depth > result.MaxDepth {
			result.MaxDepth = n.Depth
		}
	}
	return result, nil
}

func collectNodes(v any, depth int) []PropNode {
	var nodes []PropNode
	switch val := v.(type) {
	case map[string]any:
		for k, child := range val {
			nodes = append(nodes, PropNode{Key: k, Depth: depth})
			nodes = append(nodes, collectNodes(child, depth+1)...)
		}
	case []any:
		for _, item := range val {
			nodes = append(nodes, collectNodes(item, depth)...)
		}
	}
	return nodes
}

func FindMissingProps(content string, template map[string]any) ([]string, error) {
	fmRaw, err := ExtractFrontMatterBoundary(content)
	if err != nil {
		return nil, err
	}

	var current map[string]any
	if err := yaml.Unmarshal([]byte(fmRaw), &current); err != nil {
		return nil, fmt.Errorf("failed to parse YAML: %w", err)
	}

	var missing []string
	for key := range template {
		if _, exists := current[key]; !exists {
			missing = append(missing, key)
		}
	}
	sort.Strings(missing)
	return missing, nil
}

func FindExtraProps(content string, template map[string]any) ([]string, error) {
	fmRaw, err := ExtractFrontMatterBoundary(content)
	if err != nil {
		return nil, err
	}

	var current map[string]any
	if err := yaml.Unmarshal([]byte(fmRaw), &current); err != nil {
		return nil, fmt.Errorf("failed to parse YAML: %w", err)
	}

	var extras []string
	for key := range current {
		if _, inTemplate := template[key]; !inTemplate {
			extras = append(extras, key)
		}
	}
	sort.Strings(extras)
	return extras, nil
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
