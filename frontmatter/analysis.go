package frontmatter

import (
	"fmt"
	"os"
	"sort"
	"strings"

	"gopkg.in/yaml.v3"
)

type FileAnalysis struct {
	Path string

	Placement PlacementResult

	MissingProps     []string
	ExtraProps       []string
	EmptyProps       []string
	OutOfOrder       bool
	MissingOrEmptyID bool
	HasFrontMatter   bool
}

// AnalyzeFile runs all checks against a single file. templateKeys may be nil
// to skip the order check.
func AnalyzeFile(path string, template map[string]any, templateKeys []string) (FileAnalysis, error) {
	analysis := FileAnalysis{Path: path}

	content, err := os.ReadFile(path)
	if err != nil {
		return analysis, fmt.Errorf("failed to read file: %w", err)
	}
	raw := string(content)

	placement := AuditFrontMatterPlacement(raw)
	analysis.Placement = placement
	analysis.HasFrontMatter = placement.Status.IsProcessable()

	if !placement.Status.IsProcessable() {
		return analysis, nil
	}

	if template != nil {
		if analysis.MissingProps, err = FindMissingProps(raw, template); err != nil {
			return analysis, err
		}
		if analysis.ExtraProps, err = FindExtraProps(raw, template); err != nil {
			return analysis, err
		}
		templateKeys2 := make([]string, 0, len(template))
		for k := range template {
			templateKeys2 = append(templateKeys2, k)
		}
		if analysis.EmptyProps, err = FindEmptyProps(raw, templateKeys2); err != nil {
			return analysis, err
		}
	}

	if len(templateKeys) > 0 && len(analysis.MissingProps) == 0 {
		fileKeys, err := GetFrontMatterKeyOrder(raw)
		if err == nil {
			analysis.OutOfOrder = !IsOrderedByTemplate(fileKeys, templateKeys)
		}
	}

	return analysis, nil
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

// FindEmptyProps returns the subset of keys that are present in the front
// matter but have a nil, empty-string, or whitespace-only value.
func FindEmptyProps(content string, keys []string) ([]string, error) {
	fmRaw, err := ExtractFrontMatterBoundary(content)
	if err != nil {
		return nil, err
	}

	var current map[string]any
	if err := yaml.Unmarshal([]byte(fmRaw), &current); err != nil {
		return nil, fmt.Errorf("failed to parse YAML: %w", err)
	}

	var empty []string
	for _, key := range keys {
		val, exists := current[key]
		if !exists {
			continue
		}
		switch v := val.(type) {
		case nil:
			empty = append(empty, key)
		case string:
			if strings.TrimSpace(v) == "" {
				empty = append(empty, key)
			}
		}
	}
	return empty, nil
}

// GetFrontMatterMap parses the front matter of content into a map.
// Returns an empty map (not an error) when the file has no front matter.
func GetFrontMatterMap(content string) (map[string]any, error) {
	fmRaw, err := ExtractFrontMatterBoundary(content)
	if err != nil {
		return map[string]any{}, nil // no front matter
	}
	var fm map[string]any
	if err := yaml.Unmarshal([]byte(fmRaw), &fm); err != nil {
		return nil, fmt.Errorf("failed to parse YAML: %w", err)
	}
	if fm == nil {
		return map[string]any{}, nil
	}
	return fm, nil
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
		len(fa.EmptyProps) > 0 ||
		fa.OutOfOrder ||
		fa.MissingOrEmptyID
}

