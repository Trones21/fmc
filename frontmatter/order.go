package frontmatter

import (
	"fmt"

	"gopkg.in/yaml.v3"
)

// GetFrontMatterKeyOrder returns the front matter keys in the order they
// appear in the YAML document.
func GetFrontMatterKeyOrder(content string) ([]string, error) {
	fmRaw, err := ExtractFrontMatterBoundary(content)
	if err != nil {
		return nil, err
	}

	var node yaml.Node
	if err := yaml.Unmarshal([]byte(fmRaw), &node); err != nil {
		return nil, fmt.Errorf("failed to parse YAML: %w", err)
	}

	if node.Kind == 0 || len(node.Content) == 0 {
		return nil, nil
	}

	mapping := node.Content[0] // document node → mapping node
	if mapping.Kind != yaml.MappingNode {
		return nil, nil
	}

	keys := make([]string, 0, len(mapping.Content)/2)
	for i := 0; i < len(mapping.Content); i += 2 {
		keys = append(keys, mapping.Content[i].Value)
	}
	return keys, nil
}

// IsOrderedByTemplate reports whether the template keys appear in the same
// relative order within fileKeys. Extra keys in fileKeys (not in template)
// are ignored.
func IsOrderedByTemplate(fileKeys, templateKeys []string) bool {
	templateSet := make(map[string]bool, len(templateKeys))
	for _, k := range templateKeys {
		templateSet[k] = true
	}

	// Extract the subsequence of fileKeys that are template keys
	var sub []string
	for _, k := range fileKeys {
		if templateSet[k] {
			sub = append(sub, k)
		}
	}

	if len(sub) != len(templateKeys) {
		return false // file is missing some template keys
	}
	for i, k := range templateKeys {
		if sub[i] != k {
			return false
		}
	}
	return true
}
