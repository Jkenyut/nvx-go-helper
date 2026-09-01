package pagination

import (
	"net/http"
	"strings"

	"github.com/Jkenyut/nvx-go-helper/request"
	"github.com/Jkenyut/nvx-go-helper/sliceutil"
)

// reservedQueryParams lists all internal query parameter names used for pagination and search.
var reservedQueryParams = map[string]struct{}{
	"filter":          {},
	"search":          {},
	"sort_by":         {},
	"sort_type":       {},
	"cursor":          {},
	"direction":       {},
	"limit":           {},
	"show_pagination": {},
	"page":            {},
	"per_page":        {},
}

// ExtractFiltersAndSearch parses search and filter parameters from an HTTP request.
// It supports two query formats:
//  1. Grouped format: ?filter=col:val1,val2
//  2. Direct query parameters: ?col=val1,val2
//
// Only columns present in allowedFilters are accepted. Keys in allowedFilters are case-insensitive.
// Filter values are deduplicated per column.
func ExtractFiltersAndSearch(r *http.Request, allowedFilters map[string]string) (map[string][]string, string) {
	if r == nil {
		return make(map[string][]string), ""
	}

	normalizedAllowed := make(map[string]string, len(allowedFilters))
	for k, v := range allowedFilters {
		normalizedAllowed[strings.ToLower(strings.TrimSpace(k))] = v
	}

	filters := make(map[string][]string)

	// 1. Parse ?filter=col:val1,val2 format
	for k, v := range request.GetQueryMapSlice(r, "filter", ":") {
		col := strings.ToLower(strings.TrimSpace(k))
		if dbCol, ok := normalizedAllowed[col]; ok && len(v) > 0 {
			filters[dbCol] = append(filters[dbCol], v...)
		}
	}

	// 2. Parse direct query params matching allowed filter columns
	if r.URL != nil {
		for queryCol, vals := range r.URL.Query() {
			col := strings.ToLower(strings.TrimSpace(queryCol))
			if _, isReserved := reservedQueryParams[col]; isReserved {
				continue
			}

			if dbCol, ok := normalizedAllowed[col]; ok {
				for _, val := range vals {
					for _, part := range strings.Split(val, ",") {
						if trimmed := strings.TrimSpace(part); trimmed != "" {
							filters[dbCol] = append(filters[dbCol], trimmed)
						}
					}
				}
			}
		}
	}

	// 3. Deduplicate values per column
	for dbCol, vals := range filters {
		filters[dbCol] = sliceutil.Unique(vals)
	}

	search := request.GetQueryString(r, "search", "")
	return filters, search
}

// OffsetFilterRequest holds traditional offset pagination parameters, column filters, and search query.
type OffsetFilterRequest struct {
	OffsetRequest
	Filters map[string][]string `json:"filters"`
	Search  string              `json:"search"`
}

// HasFilter checks if a specific column filter exists and has at least one value.
func (r OffsetFilterRequest) HasFilter(col string) bool {
	vals, ok := r.Filters[col]
	return ok && len(vals) > 0
}

// GetFilter returns all filter values for a specific column.
func (r OffsetFilterRequest) GetFilter(col string) []string {
	return r.Filters[col]
}

// GetFirstFilter returns the first filter value for a specific column, or an empty string if not present.
func (r OffsetFilterRequest) GetFirstFilter(col string) string {
	if vals, ok := r.Filters[col]; ok && len(vals) > 0 {
		return vals[0]
	}
	return ""
}

// BindOffsetFilterRequest extracts offset pagination parameters, column filters, and search query from an HTTP request.
func BindOffsetFilterRequest(r *http.Request, allowedFilters map[string]string, maxLimit ...int) OffsetFilterRequest {
	offsetReq := BindOffsetRequest(r)

	// Apply max limit capping if specified
	if len(maxLimit) > 0 && maxLimit[0] > 0 {
		if offsetReq.Limit > maxLimit[0] {
			offsetReq.Limit = maxLimit[0]
		}
	}

	filters, search := ExtractFiltersAndSearch(r, allowedFilters)

	return OffsetFilterRequest{
		OffsetRequest: offsetReq,
		Filters:       filters,
		Search:        search,
	}
}

// CursorFilterRequest holds dynamic cursor pagination parameters, column filters, search query, and decoded cursor values.
type CursorFilterRequest struct {
	DynamicCursorRequest
	Filters      map[string][]string `json:"filters"`
	Search       string              `json:"search"`
	CursorValues []any               `json:"-"`
	CursorErr    error               `json:"-"`
}

// ListFilterRequest is an alias for CursorFilterRequest for flexible naming.
type ListFilterRequest = CursorFilterRequest

// HasFilter checks if a specific column filter exists and has at least one value.
func (r CursorFilterRequest) HasFilter(col string) bool {
	vals, ok := r.Filters[col]
	return ok && len(vals) > 0
}

// GetFilter returns all filter values for a specific column.
func (r CursorFilterRequest) GetFilter(col string) []string {
	return r.Filters[col]
}

// GetFirstFilter returns the first filter value for a specific column, or an empty string if not present.
func (r CursorFilterRequest) GetFirstFilter(col string) string {
	if vals, ok := r.Filters[col]; ok && len(vals) > 0 {
		return vals[0]
	}
	return ""
}

// BindCursorFilterRequest extracts dynamic cursor pagination parameters, column filters, search query,
// and automatically decodes the cursor values if present.
func BindCursorFilterRequest(r *http.Request, allowedFilters map[string]string, maxLimit ...int) CursorFilterRequest {
	cursorReq := BindDynamicCursorRequest(r)

	// Apply max limit capping if specified
	if len(maxLimit) > 0 && maxLimit[0] > 0 {
		if cursorReq.Limit > maxLimit[0] {
			cursorReq.Limit = maxLimit[0]
		}
	}

	filters, search := ExtractFiltersAndSearch(r, allowedFilters)

	var cursorVals []any
	var cursorErr error
	if cursorReq.Cursor != "" {
		cursorVals, cursorErr = DecodeDynamicCursor(cursorReq.Cursor)
	}

	return CursorFilterRequest{
		DynamicCursorRequest: cursorReq,
		Filters:              filters,
		Search:               search,
		CursorValues:         cursorVals,
		CursorErr:            cursorErr,
	}
}

// BindListFilterRequest is an alias for BindCursorFilterRequest.
func BindListFilterRequest(r *http.Request, allowedFilters map[string]string, maxLimit ...int) ListFilterRequest {
	return BindCursorFilterRequest(r, allowedFilters, maxLimit...)
}

// UnifiedFilterRequest can handle both traditional offset and dynamic cursor pagination in a single request struct.
type UnifiedFilterRequest struct {
	IsCursor             bool                 `json:"is_cursor"`
	OffsetRequest        OffsetRequest        `json:"offset_request"`
	DynamicCursorRequest DynamicCursorRequest `json:"dynamic_cursor_request"`
	Filters              map[string][]string  `json:"filters"`
	Search               string               `json:"search"`
	CursorValues         []any                `json:"-"`
	CursorErr            error                `json:"-"`
}

// HasFilter checks if a specific column filter exists and has at least one value.
func (r UnifiedFilterRequest) HasFilter(col string) bool {
	vals, ok := r.Filters[col]
	return ok && len(vals) > 0
}

// GetFilter returns all filter values for a specific column.
func (r UnifiedFilterRequest) GetFilter(col string) []string {
	return r.Filters[col]
}

// GetFirstFilter returns the first filter value for a specific column, or an empty string if not present.
func (r UnifiedFilterRequest) GetFirstFilter(col string) string {
	if vals, ok := r.Filters[col]; ok && len(vals) > 0 {
		return vals[0]
	}
	return ""
}

// BindUnifiedFilterRequest automatically detects whether the incoming request uses cursor pagination
// (if "cursor" or "direction" query parameters are provided) or falls back to traditional offset pagination.
func BindUnifiedFilterRequest(r *http.Request, allowedFilters map[string]string, maxLimit ...int) UnifiedFilterRequest {
	cursorReq := BindDynamicCursorRequest(r)
	offsetReq := BindOffsetRequest(r)

	// Apply max limit capping if specified
	if len(maxLimit) > 0 && maxLimit[0] > 0 {
		if cursorReq.Limit > maxLimit[0] {
			cursorReq.Limit = maxLimit[0]
		}
		if offsetReq.Limit > maxLimit[0] {
			offsetReq.Limit = maxLimit[0]
		}
	}

	filters, search := ExtractFiltersAndSearch(r, allowedFilters)

	isCursor := false
	if r != nil && r.URL != nil {
		q := r.URL.Query()
		if q.Has("cursor") || (q.Has("direction") && !q.Has("page")) {
			isCursor = true
		}
	}

	var cursorVals []any
	var cursorErr error
	if cursorReq.Cursor != "" {
		cursorVals, cursorErr = DecodeDynamicCursor(cursorReq.Cursor)
	}

	return UnifiedFilterRequest{
		IsCursor:             isCursor,
		OffsetRequest:        offsetReq,
		DynamicCursorRequest: cursorReq,
		Filters:              filters,
		Search:               search,
		CursorValues:         cursorVals,
		CursorErr:            cursorErr,
	}
}

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
