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

// UnifiedListResponse formats a standardized response for unified/hybrid pagination endpoints.
// The Pagination field can hold either *Pagination (offset mode) or *CursorPagination (cursor mode),
// serializing seamlessly into JSON while preserving strong type safety for Items.
type UnifiedListResponse[T any] struct {
	Items      []T `json:"items"`
	Pagination any `json:"pagination,omitempty"`
}

// NewUnifiedListResponse creates a new UnifiedListResponse for hybrid endpoints.
func NewUnifiedListResponse[T any](items []T, pagination any) UnifiedListResponse[T] {
	return UnifiedListResponse[T]{
		Items:      items,
		Pagination: pagination,
	}
}

// ToUnified converts a ListResponse[T] into a UnifiedListResponse[T].
func (r ListResponse[T]) ToUnified() UnifiedListResponse[T] {
	return UnifiedListResponse[T]{
		Items:      r.Items,
		Pagination: r.Pagination,
	}
}

// ToUnified converts a CursorListResponse[T] into a UnifiedListResponse[T].
func (r CursorListResponse[T]) ToUnified() UnifiedListResponse[T] {
	return UnifiedListResponse[T]{
		Items:      r.Items,
		Pagination: r.Pagination,
	}
}

