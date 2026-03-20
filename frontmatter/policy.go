package frontmatter

import (
	"encoding/json"
	"fmt"
	"os"
)

type PropertyAction string

const (
	ActionAddIfMissing     PropertyAction = "add_if_missing"
	ActionOverwriteAlways  PropertyAction = "overwrite_always"
	ActionOverwriteIfEmpty PropertyAction = "overwrite_if_empty"
	ActionPreserve         PropertyAction = "preserve"
	ActionRenameFrom       PropertyAction = "rename_from"
)

type PropertyPolicy struct {
	Key     string
	Action  PropertyAction
	Source  ValueSource
	Fn      string
	FromKey string

	StaticValue any
	Params      map[string]any
}

type policyFileEntry struct {
	Action string         `json:"action"`
	Source string         `json:"source"`
	Value  any            `json:"value,omitempty"`
	Fn     string         `json:"fn,omitempty"`
	From   string         `json:"from,omitempty"`
	Params map[string]any `json:"params,omitempty"`
}

func LoadPolicy(path string) ([]PropertyPolicy, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("failed to read policy file: %w", err)
	}

	var raw map[string]policyFileEntry
	if err := json.Unmarshal(data, &raw); err != nil {
		return nil, fmt.Errorf("failed to parse policy file: %w", err)
	}

	policies := make([]PropertyPolicy, 0, len(raw))
	for key, entry := range raw {
		policies = append(policies, PropertyPolicy{
			Key:         key,
			Action:      PropertyAction(entry.Action),
			Source:      ValueSource(entry.Source),
			Fn:          entry.Fn,
			FromKey:     entry.From,
			StaticValue: entry.Value,
			Params:      entry.Params,
		})
	}
	return policies, nil
}
