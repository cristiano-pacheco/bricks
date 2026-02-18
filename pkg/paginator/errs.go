package paginator

import "errors"

var (
	ErrInvalidPage    = errors.New("invalid page")
	ErrInvalidPerPage = errors.New("invalid per_page")
)
