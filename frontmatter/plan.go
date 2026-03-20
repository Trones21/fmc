package frontmatter

import (
	"fmt"
	"os"
	"strings"

	"gopkg.in/yaml.v3"
)

type PropChange struct {
	Key         string
	OldValue    any
	NewValue    any
	RenamedFrom string // non-empty when this change is a key rename
}

type FileChangePlan struct {
	FilePath     string
	Changes      []PropChange
	KeysToDelete []string // old keys removed by renames
}

func (p FileChangePlan) HasChanges() bool {
	return len(p.Changes) > 0 || len(p.KeysToDelete) > 0
}

func PlanChanges(path string, content string, template map[string]any, policies []PropertyPolicy) (FileChangePlan, error) {
	plan := FileChangePlan{FilePath: path}

	fmRaw, err := ExtractFrontMatterBoundary(content)
	if err != nil {
		return plan, err
	}

	var current map[string]any
	if err := yaml.Unmarshal([]byte(fmRaw), &current); err != nil {
		return plan, fmt.Errorf("failed to parse YAML: %w", err)
	}
	if current == nil {
		current = make(map[string]any)
	}

	ctx := ResolveContext{FilePath: path, Content: content, FrontMatter: current}

	policyMap := make(map[string]PropertyPolicy, len(policies))
	for _, p := range policies {
		policyMap[p.Key] = p
	}

	for key := range template {
		policy, ok := policyMap[key]
		if !ok {
			policy = PropertyPolicy{Key: key, Action: ActionPreserve}
		}

		if policy.Action == ActionRenameFrom {
			sourceVal, exists := current[policy.FromKey]
			if !exists {
				continue
			}
			plan.Changes = append(plan.Changes, PropChange{
				Key:         key,
				OldValue:    current[key],
				NewValue:    sourceVal,
				RenamedFrom: policy.FromKey,
			})
			plan.KeysToDelete = append(plan.KeysToDelete, policy.FromKey)
			continue
		}

		newVal, changed, err := projectedValue(current, policy, ctx)
		if err != nil {
			return plan, fmt.Errorf("key %q: %w", key, err)
		}
		if changed {
			plan.Changes = append(plan.Changes, PropChange{
				Key:      key,
				OldValue: current[key],
				NewValue: newVal,
			})
		}
	}

	return plan, nil
}

// projectedValue returns what the value would become after applying the policy,
// and whether it differs from the current state.
func projectedValue(current map[string]any, policy PropertyPolicy, ctx ResolveContext) (any, bool, error) {
	switch policy.Action {
	case ActionPreserve:
		return nil, false, nil

	case ActionAddIfMissing:
		if _, exists := current[policy.Key]; exists {
			return nil, false, nil
		}
		val, err := ResolveValue(policy, ctx)
		if err != nil {
			return nil, false, err
		}
		return val, true, nil

	case ActionOverwriteAlways:
		val, err := ResolveValue(policy, ctx)
		if err != nil {
			return nil, false, err
		}
		return val, true, nil

	case ActionOverwriteIfEmpty:
		existing := current[policy.Key]
		if existing != nil && existing != "" {
			return nil, false, nil
		}
		val, err := ResolveValue(policy, ctx)
		if err != nil {
			return nil, false, err
		}
		return val, true, nil

	default:
		return nil, false, ErrInvalidAction
	}
}

func ApplyChangePlan(plan FileChangePlan) error {
	content, err := os.ReadFile(plan.FilePath)
	if err != nil {
		return fmt.Errorf("failed to read file: %w", err)
	}

	raw := strings.ReplaceAll(string(content), "\r\n", "\n")
	raw = strings.ReplaceAll(raw, "\r", "\n")

	fmRaw, err := ExtractFrontMatterBoundary(raw)
	if err != nil {
		return err
	}

	var current map[string]any
	if err := yaml.Unmarshal([]byte(fmRaw), &current); err != nil {
		return fmt.Errorf("failed to parse YAML: %w", err)
	}
	if current == nil {
		current = make(map[string]any)
	}

	for _, key := range plan.KeysToDelete {
		delete(current, key)
	}
	for _, change := range plan.Changes {
		current[change.Key] = change.NewValue
	}

	updated, err := yaml.Marshal(current)
	if err != nil {
		return fmt.Errorf("failed to marshal YAML: %w", err)
	}

	// Extract body (everything after the closing fence)
	lines := strings.Split(raw, "\n")
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

	return os.WriteFile(plan.FilePath, []byte("---\n"+string(updated)+"---\n"+body), 0644)
}
