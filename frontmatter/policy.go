package frontmatter

type PropertyAction string

const (
	ActionAddIfMissing     PropertyAction = "add_if_missing"
	ActionOverwriteAlways  PropertyAction = "overwrite_always"
	ActionOverwriteIfEmpty PropertyAction = "overwrite_if_empty"
	ActionPreserve         PropertyAction = "preserve"
)

type PropertyPolicy struct {
	Key    string
	Action PropertyAction
	Source ValueSource

	StaticValue any
	Params      map[string]any
}
