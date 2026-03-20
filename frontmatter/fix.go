package frontmatter

func ApplyPropertyPolicy(current map[string]any, policy PropertyPolicy, ctx ResolveContext) error {
	switch policy.Action {
	case ActionAddIfMissing:
		return ErrNotImplemented

	case ActionOverwriteAlways:
		return ErrNotImplemented

	case ActionOverwriteIfEmpty:
		return ErrNotImplemented

	case ActionPreserve:
		return nil

	default:
		return ErrInvalidAction
	}
}
