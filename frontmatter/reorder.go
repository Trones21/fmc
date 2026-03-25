package frontmatter

import (
	"fmt"
	"os"
	"strings"

	"gopkg.in/yaml.v3"
)

type ReorderPlan struct {
	FilePath    string
	OldOrder    []string
	NewOrder    []string
	MissingKeys []string // listed in first/last but absent from the file
	HasChange   bool
}

// PlanReorder computes the new key order for a single file.
// firstKeys are moved to the front (in order); lastKeys are moved to the end
// (in order); all other keys keep their current relative positions in between.
func PlanReorder(path, content string, firstKeys, lastKeys []string) (ReorderPlan, error) {
	plan := ReorderPlan{FilePath: path}

	currentKeys, err := GetFrontMatterKeyOrder(content)
	if err != nil {
		return plan, err
	}
	plan.OldOrder = currentKeys

	currentSet := make(map[string]bool, len(currentKeys))
	for _, k := range currentKeys {
		currentSet[k] = true
	}
	firstSet := make(map[string]bool, len(firstKeys))
	for _, k := range firstKeys {
		firstSet[k] = true
	}
	lastSet := make(map[string]bool, len(lastKeys))
	for _, k := range lastKeys {
		lastSet[k] = true
	}

	for _, k := range firstKeys {
		if !currentSet[k] {
			plan.MissingKeys = append(plan.MissingKeys, k)
		}
	}
	for _, k := range lastKeys {
		if !currentSet[k] {
			plan.MissingKeys = append(plan.MissingKeys, k)
		}
	}

	var newOrder []string
	for _, k := range firstKeys {
		if currentSet[k] {
			newOrder = append(newOrder, k)
		}
	}
	for _, k := range currentKeys {
		if !firstSet[k] && !lastSet[k] {
			newOrder = append(newOrder, k)
		}
	}
	for _, k := range lastKeys {
		if currentSet[k] {
			newOrder = append(newOrder, k)
		}
	}
	plan.NewOrder = newOrder
	plan.HasChange = !stringSlicesEqual(plan.OldOrder, plan.NewOrder)

	return plan, nil
}

// ApplyReorder writes the reordered front matter back to disk.
func ApplyReorder(plan ReorderPlan) error {
	content, err := os.ReadFile(plan.FilePath)
	if err != nil {
		return fmt.Errorf("failed to read file: %w", err)
	}

	raw := strings.ReplaceAll(string(content), "\r\n", "\n")
	raw = strings.ReplaceAll(raw, "\r", "\n")

	newContent, err := reorderYAMLFrontMatter(raw, plan.NewOrder)
	if err != nil {
		return err
	}

	return os.WriteFile(plan.FilePath, []byte(newContent), 0644)
}

func reorderYAMLFrontMatter(content string, newOrder []string) (string, error) {
	fmRaw, err := ExtractFrontMatterBoundary(content)
	if err != nil {
		return "", err
	}

	var doc yaml.Node
	if err := yaml.Unmarshal([]byte(fmRaw), &doc); err != nil {
		return "", fmt.Errorf("failed to parse YAML: %w", err)
	}

	if doc.Kind != yaml.DocumentNode || len(doc.Content) == 0 {
		return content, nil
	}
	mapping := doc.Content[0]
	if mapping.Kind != yaml.MappingNode {
		return content, nil
	}

	type pair struct{ key, val *yaml.Node }
	pairs := make(map[string]pair, len(mapping.Content)/2)
	for i := 0; i+1 < len(mapping.Content); i += 2 {
		k := mapping.Content[i].Value
		pairs[k] = pair{mapping.Content[i], mapping.Content[i+1]}
	}

	rebuilt := make([]*yaml.Node, 0, len(mapping.Content))
	for _, k := range newOrder {
		if p, ok := pairs[k]; ok {
			rebuilt = append(rebuilt, p.key, p.val)
		}
	}
	mapping.Content = rebuilt

	updated, err := yaml.Marshal(&doc)
	if err != nil {
		return "", fmt.Errorf("failed to marshal YAML: %w", err)
	}

	lines := strings.Split(content, "\n")
	bodyStart := -1
	for i := 1; i < len(lines); i++ {
		if strings.TrimSpace(lines[i]) == "---" {
			bodyStart = i + 1
			break
		}
	}
	body := ""
	if bodyStart >= 0 && bodyStart < len(lines) {
		body = strings.Join(lines[bodyStart:], "\n")
	}

	return "---\n" + string(updated) + "---\n" + body, nil
}

func stringSlicesEqual(a, b []string) bool {
	if len(a) != len(b) {
		return false
	}
	for i := range a {
		if a[i] != b[i] {
			return false
		}
	}
	return true
}
