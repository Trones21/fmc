package frontmatter

import "errors"

var ErrNotImplemented = errors.New("not implemented")
var ErrInvalidSource = errors.New("invalid value source")
var ErrInvalidAction = errors.New("invalid property action")
var ErrUnknownFunction = errors.New("unknown function")
