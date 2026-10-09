package pagination

// ListResponse formats a standard paginated list response for offset pagination.
type ListResponse[T any] struct {
	Items      []T         `json:"items"`
	Pagination *Pagination `json:"pagination,omitempty"`
}

// NewListResponse creates a new ListResponse for offset pagination.
func NewListResponse[T any](items []T, p Pagination) ListResponse[T] {
	return ListResponse[T]{
		Items:      items,
		Pagination: &p,
	}
}

// CursorListResponse formats a paginated list response with bidirectional cursor metadata.
type CursorListResponse[T any] struct {
	Items      []T               `json:"items"`
	Pagination *CursorPagination `json:"pagination,omitempty"`
}

// NewCursorListResponse creates a new CursorListResponse for cursor pagination.
func NewCursorListResponse[T any](items []T, p *CursorPagination) CursorListResponse[T] {
	return CursorListResponse[T]{
		Items:      items,
		Pagination: p,
	}
}

// NewFinalizedCursorResponse reverses items if navigating backwards ("prev"), calculates cursor metadata,
// and returns a ready-to-send CursorListResponse[T].
func NewFinalizedCursorResponse[T any](items []T, req DynamicCursorRequest, extractFn func(T) []any) CursorListResponse[T] {
	finalItems, meta := FinalizeCursor(items, req, extractFn)
	return NewCursorListResponse(finalItems, meta)
}
