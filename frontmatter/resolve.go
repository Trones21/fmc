package frontmatter

type ValueSource string

const (
	SourceStatic   ValueSource = "static"
	SourceComputed ValueSource = "computed"
	SourceLLM      ValueSource = "llm"
)

type ResolveContext struct {
	FilePath    string
	Content     string
	FrontMatter map[string]any
}

func ResolveValue(policy PropertyPolicy, ctx ResolveContext) (any, error) {
	switch policy.Source {
	case SourceStatic:
		return policy.StaticValue, nil

	case SourceComputed:
		return nil, ErrNotImplemented

	case SourceLLM:
		return nil, ErrNotImplemented

	default:
		return nil, ErrInvalidSource
	}
}
