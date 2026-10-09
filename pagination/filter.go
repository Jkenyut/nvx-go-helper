package pagination

import (
	"fmt"
	"net/http"
	"net/url"
	"strconv"
	"strings"

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
	"mode":            {},
	"pagination":      {},
}

func parseQueryInt(q url.Values, key string, defaultValue int) int {
	val := strings.TrimSpace(q.Get(key))
	if val == "" {
		return defaultValue
	}
	parsed, err := strconv.Atoi(val)
	if err != nil {
		return defaultValue
	}
	return parsed
}

func parseQueryBool(q url.Values, key string, defaultValue bool) bool {
	val := strings.TrimSpace(q.Get(key))
	if val == "" {
		return defaultValue
	}
	parsed, err := strconv.ParseBool(val)
	if err != nil {
		return defaultValue
	}
	return parsed
}

func bindOffsetRequestFromQuery(q url.Values) OffsetRequest {
	return OffsetRequest{
		SortBy:         strings.TrimSpace(q.Get("sort_by")),
		SortType:       strings.TrimSpace(q.Get("sort_type")),
		Page:           parseQueryInt(q, "page", DefaultPage),
		Limit:          parseQueryInt(q, "limit", 0),
		ShowPagination: parseQueryBool(q, "show_pagination", true),
	}
}

func bindDynamicCursorRequestFromQuery(q url.Values) DynamicCursorRequest {
	dir := strings.TrimSpace(q.Get("direction"))
	if dir == "" {
		dir = "next"
	}
	return DynamicCursorRequest{
		SortBy:         strings.TrimSpace(q.Get("sort_by")),
		SortType:       strings.TrimSpace(q.Get("sort_type")),
		Cursor:         strings.TrimSpace(q.Get("cursor")),
		Direction:      dir,
		Limit:          parseQueryInt(q, "limit", 0),
		ShowPagination: parseQueryBool(q, "show_pagination", true),
	}
}

func extractFiltersAndSearchFromQuery(q url.Values, allowedFilters map[string]string) (map[string][]string, string) {
	search := strings.TrimSpace(q.Get("search"))
	if len(allowedFilters) == 0 {
		return make(map[string][]string), search
	}

	normalizedAllowed := make(map[string]string, len(allowedFilters))
	for k, v := range allowedFilters {
		normalizedAllowed[strings.ToLower(strings.TrimSpace(k))] = v
	}

	filters := make(map[string][]string)

	// 1. Parse ?filter=col:val1,val2 format from q["filter"]
	if filterValues, ok := q["filter"]; ok {
		for _, val := range filterValues {
			rawCol, rawVals, found := strings.Cut(val, ":")
			if !found {
				continue
			}
			col := strings.ToLower(strings.TrimSpace(rawCol))
			if dbCol, ok := normalizedAllowed[col]; ok {
				if strings.ContainsRune(rawVals, ',') {
					for _, part := range strings.Split(rawVals, ",") {
						if trimmed := strings.TrimSpace(part); trimmed != "" {
							filters[dbCol] = append(filters[dbCol], trimmed)
						}
					}
				} else {
					if trimmed := strings.TrimSpace(rawVals); trimmed != "" {
						filters[dbCol] = append(filters[dbCol], trimmed)
					}
				}
			}
		}
	}

	// 2. Parse direct query params matching allowed filter columns
	for queryCol, vals := range q {
		col := strings.ToLower(strings.TrimSpace(queryCol))
		if _, isReserved := reservedQueryParams[col]; isReserved {
			continue
		}

		if dbCol, ok := normalizedAllowed[col]; ok {
			for _, val := range vals {
				if strings.ContainsRune(val, ',') {
					for _, part := range strings.Split(val, ",") {
						if trimmed := strings.TrimSpace(part); trimmed != "" {
							filters[dbCol] = append(filters[dbCol], trimmed)
						}
					}
				} else {
					if trimmed := strings.TrimSpace(val); trimmed != "" {
						filters[dbCol] = append(filters[dbCol], trimmed)
					}
				}
			}
		}
	}

	// 3. Deduplicate values per column
	for dbCol, vals := range filters {
		if len(vals) > 1 {
			filters[dbCol] = sliceutil.Unique(vals)
		}
	}

	return filters, search
}

// ExtractFiltersAndSearch parses search and filter parameters from an HTTP request.
// It supports two query formats:
//  1. Grouped format: ?filter=col:val1,val2
//  2. Direct query parameters: ?col=val1,val2
//
// Only columns present in allowedFilters are accepted. Keys in allowedFilters are case-insensitive.
// Filter values are deduplicated per column.
func ExtractFiltersAndSearch(r *http.Request, allowedFilters map[string]string) (map[string][]string, string) {
	if r == nil || r.URL == nil {
		return make(map[string][]string), ""
	}
	return extractFiltersAndSearchFromQuery(r.URL.Query(), allowedFilters)
}


// FilterRequest represents any filter-enabled pagination request (Offset, Cursor, or Unified).
type FilterRequest interface {
	HasFilter(col string) bool
	GetFilter(col string) []string
	GetFirstFilter(col string) string
	NormalizedFilterValue(col string, detector ...*TypeDetector) (any, error)
}

// CursorFilterProvider represents requests supporting cursor decoding and normalization.
type CursorFilterProvider interface {
	FilterRequest
	NormalizedCursorValues(sortCols []string, detector ...*TypeDetector) ([]any, error)
}

func lookupFilter(filters map[string][]string, col string) []string {
	if len(filters) == 0 {
		return nil
	}
	if vals, ok := filters[col]; ok && len(vals) > 0 {
		return vals
	}
	norm := strings.ToLower(strings.TrimSpace(col))
	if vals, ok := filters[norm]; ok && len(vals) > 0 {
		return vals
	}
	for k, vals := range filters {
		if strings.ToLower(strings.TrimSpace(k)) == norm && len(vals) > 0 {
			return vals
		}
	}
	return nil
}

// FilterBase holds extracted column filters and search query, providing query evaluation methods.
type FilterBase struct {
	Filters map[string][]string `json:"filters"`
	Search  string              `json:"search"`
}

// HasFilter checks if a specific column filter exists and has at least one value.
func (f FilterBase) HasFilter(col string) bool {
	return len(lookupFilter(f.Filters, col)) > 0
}

// GetFilter returns all filter values for a specific column.
func (f FilterBase) GetFilter(col string) []string {
	return lookupFilter(f.Filters, col)
}

// GetFirstFilter returns the first filter value for a specific column, or an empty string if not present.
func (f FilterBase) GetFirstFilter(col string) string {
	if vals := lookupFilter(f.Filters, col); len(vals) > 0 {
		return vals[0]
	}
	return ""
}

// NormalizedFilterValue converts filter values for a column using DefaultDetector or a custom detector.
func (f FilterBase) NormalizedFilterValue(col string, detector ...*TypeDetector) (any, error) {
	d := DefaultDetector
	if len(detector) > 0 && detector[0] != nil {
		d = detector[0]
	}
	return d.NormalizeFilterValue(col, lookupFilter(f.Filters, col))
}

// OffsetFilterRequest holds traditional offset pagination parameters, column filters, and search query.
type OffsetFilterRequest struct {
	OffsetRequest
	FilterBase
}

// BindOffsetFilterRequest extracts offset pagination parameters, column filters, and search query from an HTTP request.
func BindOffsetFilterRequest(r *http.Request, allowedFilters map[string]string, maxLimit ...int) OffsetFilterRequest {
	if r == nil || r.URL == nil {
		return OffsetFilterRequest{
			OffsetRequest: OffsetRequest{Page: DefaultPage, ShowPagination: true},
			FilterBase:    FilterBase{Filters: make(map[string][]string)},
		}
	}

	q := r.URL.Query()
	offsetReq := bindOffsetRequestFromQuery(q)

	// Apply max limit capping if specified
	if len(maxLimit) > 0 && maxLimit[0] > 0 {
		if offsetReq.Limit > maxLimit[0] {
			offsetReq.Limit = maxLimit[0]
		}
	}

	filters, search := extractFiltersAndSearchFromQuery(q, allowedFilters)

	return OffsetFilterRequest{
		OffsetRequest: offsetReq,
		FilterBase: FilterBase{
			Filters: filters,
			Search:  search,
		},
	}
}

// CursorFilterRequest holds dynamic cursor pagination parameters, column filters, search query, and decoded cursor values.
type CursorFilterRequest struct {
	DynamicCursorRequest
	FilterBase
	CursorValues []any `json:"-"`
	CursorErr    error `json:"-"`
}

// NormalizedCursorValues normalizes the decoded cursor values against sort columns using DefaultDetector or a custom detector.
func (r CursorFilterRequest) NormalizedCursorValues(sortCols []string, detector ...*TypeDetector) ([]any, error) {
	if r.CursorErr != nil {
		return nil, r.CursorErr
	}
	d := DefaultDetector
	if len(detector) > 0 && detector[0] != nil {
		d = detector[0]
	}
	return d.NormalizeCursorValues(sortCols, r.CursorValues)
}

// BuildKeysetQuery builds a KeysetQuery using this cursor filter request.
func (r CursorFilterRequest) BuildKeysetQuery(cfg KeysetConfig) (KeysetQuery, error) {
	if r.CursorErr != nil {
		return KeysetQuery{}, r.CursorErr
	}

	cols, ops, orders := prepareDynamicSort(r.SortBy, r.SortType, r.Direction, cfg)

	var sqlWhere string
	var sqlArgs []any
	if len(r.CursorValues) > 0 {
		vals := r.CursorValues
		if cfg.AutoDetectTypes || cfg.Detector != nil {
			d := DefaultDetector
			if cfg.Detector != nil {
				d = cfg.Detector
			}
			normVals, err := d.NormalizeCursorValues(cols, r.CursorValues)
			if err != nil {
				return KeysetQuery{}, fmt.Errorf("normalizing cursor values: %w", err)
			}
			vals = normVals
		}
		sqlWhere, sqlArgs = BuildDynamicKeyset(cols, ops, vals)
	}

	return KeysetQuery{
		Where:     sqlWhere,
		Args:      sqlArgs,
		OrderStrs: orders,
		Columns:   cols,
		Operators: ops,
	}, nil
}

// BindCursorFilterRequest extracts dynamic cursor pagination parameters, column filters, search query,
// and automatically decodes the cursor values if present.
func BindCursorFilterRequest(r *http.Request, allowedFilters map[string]string, maxLimit ...int) CursorFilterRequest {
	if r == nil || r.URL == nil {
		return CursorFilterRequest{
			DynamicCursorRequest: DynamicCursorRequest{Direction: "next", ShowPagination: true},
			FilterBase:           FilterBase{Filters: make(map[string][]string)},
		}
	}

	q := r.URL.Query()
	cursorReq := bindDynamicCursorRequestFromQuery(q)

	// Apply max limit capping if specified
	if len(maxLimit) > 0 && maxLimit[0] > 0 {
		if cursorReq.Limit > maxLimit[0] {
			cursorReq.Limit = maxLimit[0]
		}
	}

	filters, search := extractFiltersAndSearchFromQuery(q, allowedFilters)

	var cursorVals []any
	var cursorErr error
	if cursorReq.Cursor != "" {
		cursorVals, cursorErr = DecodeDynamicCursor(cursorReq.Cursor)
	}

	return CursorFilterRequest{
		DynamicCursorRequest: cursorReq,
		FilterBase: FilterBase{
			Filters: filters,
			Search:  search,
		},
		CursorValues: cursorVals,
		CursorErr:    cursorErr,
	}
}

// UnifiedFilterRequest can handle both traditional offset and dynamic cursor pagination in a single request struct.
type UnifiedFilterRequest struct {
	IsCursor             bool                 `json:"is_cursor"`
	OffsetRequest        OffsetRequest        `json:"offset_request"`
	DynamicCursorRequest DynamicCursorRequest `json:"dynamic_cursor_request"`
	FilterBase
	CursorValues []any `json:"-"`
	CursorErr    error `json:"-"`
}

// ToOffsetFilterRequest converts the unified request into a dedicated OffsetFilterRequest.
func (r UnifiedFilterRequest) ToOffsetFilterRequest() OffsetFilterRequest {
	return OffsetFilterRequest{
		OffsetRequest: r.OffsetRequest,
		FilterBase:    r.FilterBase,
	}
}

// ToCursorFilterRequest converts the unified request into a dedicated CursorFilterRequest.
func (r UnifiedFilterRequest) ToCursorFilterRequest() CursorFilterRequest {
	return CursorFilterRequest{
		DynamicCursorRequest: r.DynamicCursorRequest,
		FilterBase:           r.FilterBase,
		CursorValues:         r.CursorValues,
		CursorErr:            r.CursorErr,
	}
}

// GetLimit returns the pagination limit.
func (r UnifiedFilterRequest) GetLimit() int {
	if r.IsCursor {
		return r.DynamicCursorRequest.Limit
	}
	return r.OffsetRequest.Limit
}

// GetSortBy returns the requested sort_by column string.
func (r UnifiedFilterRequest) GetSortBy() string {
	if r.IsCursor {
		return r.DynamicCursorRequest.SortBy
	}
	return r.OffsetRequest.SortBy
}

// GetSortType returns the requested sort_type direction string.
func (r UnifiedFilterRequest) GetSortType() string {
	if r.IsCursor {
		return r.DynamicCursorRequest.SortType
	}
	return r.OffsetRequest.SortType
}

// GetPage returns the requested page number (starts from 1, defaults to 1).
func (r UnifiedFilterRequest) GetPage() int {
	if r.OffsetRequest.Page < 1 {
		return DefaultPage
	}
	return r.OffsetRequest.Page
}

// Offset returns the SQL OFFSET value (0-based) computed from Page and Limit for offset mode.
func (r UnifiedFilterRequest) Offset() int {
	return r.OffsetRequest.Offset()
}

// Pagination builds a safe Pagination metadata object given the total record count (for offset mode).
func (r UnifiedFilterRequest) Pagination(totalCount int) Pagination {
	return r.OffsetRequest.Pagination(totalCount)
}

// BuildKeysetQuery builds a KeysetQuery using this unified request's cursor parameters.
func (r UnifiedFilterRequest) BuildKeysetQuery(cfg KeysetConfig) (KeysetQuery, error) {
	return r.ToCursorFilterRequest().BuildKeysetQuery(cfg)
}

// NormalizedCursorValues normalizes the decoded cursor values against sort columns using DefaultDetector or a custom detector.
func (r UnifiedFilterRequest) NormalizedCursorValues(sortCols []string, detector ...*TypeDetector) ([]any, error) {
	if r.CursorErr != nil {
		return nil, r.CursorErr
	}
	d := DefaultDetector
	if len(detector) > 0 && detector[0] != nil {
		d = detector[0]
	}
	return d.NormalizeCursorValues(sortCols, r.CursorValues)
}

// NewUnifiedResponse creates a UnifiedListResponse[T] using either offset or cursor metadata based on req.IsCursor.
func NewUnifiedResponse[T any](req UnifiedFilterRequest, items []T, pageData *Pagination, cursorMeta *CursorPagination) UnifiedListResponse[T] {
	if req.IsCursor {
		return NewUnifiedListResponse(items, cursorMeta)
	}
	return NewUnifiedListResponse(items, pageData)
}

// NewUnifiedFinalizedCursorResponse reverses items if navigating backwards ("prev"), calculates cursor metadata,
// and returns a ready-to-send UnifiedListResponse[T].
func NewUnifiedFinalizedCursorResponse[T any](items []T, req UnifiedFilterRequest, extractFn func(T) []any) UnifiedListResponse[T] {
	finalItems, meta := FinalizeCursor(items, req.DynamicCursorRequest, extractFn)
	return NewUnifiedListResponse(finalItems, meta)
}

// NewUnifiedOffsetResponse wraps items and offset pageData into a ready-to-send UnifiedListResponse[T].
func NewUnifiedOffsetResponse[T any](items []T, pageData Pagination) UnifiedListResponse[T] {
	return NewUnifiedListResponse(items, &pageData)
}

// BindUnifiedFilterRequest automatically detects whether the incoming request uses cursor pagination
// (if "cursor", "direction", "mode=cursor", or "pagination=cursor" parameters are provided) or falls back to traditional offset pagination.
func BindUnifiedFilterRequest(r *http.Request, allowedFilters map[string]string, maxLimit ...int) UnifiedFilterRequest {
	if r == nil || r.URL == nil {
		return UnifiedFilterRequest{
			OffsetRequest:        OffsetRequest{Page: DefaultPage, ShowPagination: true},
			DynamicCursorRequest: DynamicCursorRequest{Direction: "next", ShowPagination: true},
			FilterBase:           FilterBase{Filters: make(map[string][]string)},
		}
	}

	// Parse query parameters once for maximum zero-allocation performance
	q := r.URL.Query()

	sortBy := strings.TrimSpace(q.Get("sort_by"))
	sortType := strings.TrimSpace(q.Get("sort_type"))
	limit := parseQueryInt(q, "limit", 0)
	showPagination := parseQueryBool(q, "show_pagination", true)
	if len(maxLimit) > 0 && maxLimit[0] > 0 && limit > maxLimit[0] {
		limit = maxLimit[0]
	}

	page := parseQueryInt(q, "page", DefaultPage)
	cursor := strings.TrimSpace(q.Get("cursor"))
	direction := strings.TrimSpace(q.Get("direction"))
	if direction == "" {
		direction = "next"
	}

	offsetReq := OffsetRequest{
		SortBy:         sortBy,
		SortType:       sortType,
		Page:           page,
		Limit:          limit,
		ShowPagination: showPagination,
	}
	cursorReq := DynamicCursorRequest{
		SortBy:         sortBy,
		SortType:       sortType,
		Cursor:         cursor,
		Direction:      direction,
		Limit:          limit,
		ShowPagination: showPagination,
	}

	filters, search := extractFiltersAndSearchFromQuery(q, allowedFilters)

	mode := strings.ToLower(strings.TrimSpace(q.Get("mode")))
	pag := strings.ToLower(strings.TrimSpace(q.Get("pagination")))
	isCursor := cursor != "" || (q.Has("direction") && !q.Has("page")) || mode == "cursor" || pag == "cursor"

	var cursorVals []any
	var cursorErr error
	if cursor != "" {
		cursorVals, cursorErr = DecodeDynamicCursor(cursor)
	}

	return UnifiedFilterRequest{
		IsCursor:             isCursor,
		OffsetRequest:        offsetReq,
		DynamicCursorRequest: cursorReq,
		FilterBase: FilterBase{
			Filters: filters,
			Search:  search,
		},
		CursorValues: cursorVals,
		CursorErr:    cursorErr,
	}
}

