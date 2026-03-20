package frontmatter

func ApplyPropertyPolicy(current map[string]any, policy PropertyPolicy, ctx ResolveContext) error {
	switch policy.Action {
	case ActionAddIfMissing:
		if _, exists := current[policy.Key]; !exists {
			val, err := ResolveValue(policy, ctx)
			if err != nil {
				return err
			}
			current[policy.Key] = val
		}
		return nil

	case ActionOverwriteAlways:
		val, err := ResolveValue(policy, ctx)
		if err != nil {
			return err
		}
		current[policy.Key] = val
		return nil

	case ActionOverwriteIfEmpty:
		existing := current[policy.Key]
		if existing == nil || existing == "" {
			val, err := ResolveValue(policy, ctx)
			if err != nil {
				return err
			}
			current[policy.Key] = val
		}
		return nil

	case ActionPreserve:
		return nil

	default:
		return ErrInvalidAction
	}
}
