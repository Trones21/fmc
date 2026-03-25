package frontmatter

import "strings"

// KeyPath splits a dot-notation key into path segments.
// "last_update.date" → ["last_update", "date"]
func KeyPath(key string) []string { return strings.Split(key, ".") }

// keyPath is the unexported alias used within the package.
func keyPath(key string) []string { return KeyPath(key) }

// NestedGet traverses a dot-split key path into m, returning the value and
// whether it was found.
func NestedGet(m map[string]any, path []string) (any, bool) {
	return nestedGet(m, path)
}

// nestedGet is the unexported implementation.
func nestedGet(m map[string]any, path []string) (any, bool) {
	if len(path) == 0 {
		return nil, false
	}
	val, ok := m[path[0]]
	if !ok || len(path) == 1 {
		return val, ok
	}
	child, ok := val.(map[string]any)
	if !ok {
		return nil, false
	}
	return nestedGet(child, path[1:])
}

// nestedSet writes val at the dot-split key path, creating intermediate maps
// as needed. If an intermediate value is a scalar it is replaced by a map.
func nestedSet(m map[string]any, path []string, val any) {
	if len(path) == 1 {
		m[path[0]] = val
		return
	}
	child, _ := m[path[0]].(map[string]any)
	if child == nil {
		child = make(map[string]any)
	}
	nestedSet(child, path[1:], val)
	m[path[0]] = child
}

// nestedDelete removes the value at the dot-split key path.
func nestedDelete(m map[string]any, path []string) {
	if len(path) == 1 {
		delete(m, path[0])
		return
	}
	child, ok := m[path[0]].(map[string]any)
	if !ok {
		return
	}
	nestedDelete(child, path[1:])
	m[path[0]] = child
}
