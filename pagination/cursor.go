package pagination

import (
	"fmt"
	"net/http"
	"strconv"
	"strings"

	"github.com/Jkenyut/nvx-go-helper/sliceutil"
	"github.com/bytedance/sonic"
)

func formatScalarValue(v any) string {
	switch val := v.(type) {
	case string:
		return val
	case int:
		return strconv.Itoa(val)
	case int64:
		return strconv.FormatInt(val, 10)
	case int32:
		return strconv.FormatInt(int64(val), 10)
	case uint:
		return strconv.FormatUint(uint64(val), 10)
	case uint64:
		return strconv.FormatUint(val, 10)
	case float64:
		return strconv.FormatFloat(val, 'f', -1, 64)
	case float32:
		return strconv.FormatFloat(float64(val), 'f', -1, 32)
	case bool:
		if val {
			return "true"
		}
		return "false"
	case fmt.Stringer:
		return val.String()
	default:
		return fmt.Sprintf("%v", v)
	}
}

// EncodeDynamicCursor takes arbitrary keyset values (e.g. from the last row of a query)
// and encodes them into a human-readable comma-separated string to be used as a cursor.
// For single-column keysets, it outputs the single scalar value directly (e.g. "105" or "user-uuid").
// Values containing commas or quotes are safely escaped with quotes.
func EncodeDynamicCursor(values ...any) (string, error) {
	if len(values) == 0 {
		return "", nil
	}

	if len(values) == 1 {
		s := formatScalarValue(values[0])
		if strings.ContainsAny(s, ",\"\n\r") {
			return strconv.Quote(s), nil
		}
		return s, nil
	}

	parts := make([]string, len(values))
	for i, v := range values {
		s := formatScalarValue(v)
		if strings.ContainsAny(s, ",\"\n\r") {
			parts[i] = strconv.Quote(s)
		} else {
			parts[i] = s
		}
	}
	return strings.Join(parts, ","), nil
}

// DecodeDynamicCursor decodes a cursor string back into an array of values.
// It parses delimited scalar values (e.g. "105", "App A,105", "active,105")
// or JSON array strings (e.g. `["App A",105]`), with automatic type inference (number as float64, boolean, string).
func DecodeDynamicCursor(cursor string) ([]any, error) {
	trimmed := strings.TrimSpace(cursor)
	if trimmed == "" {
		return nil, nil
	}

	// 1. Direct Plain JSON array check (if client sends raw JSON)
	if strings.HasPrefix(trimmed, "[") && strings.HasSuffix(trimmed, "]") {
		var values []any
		if err := sonic.Unmarshal([]byte(trimmed), &values); err == nil {
			return values, nil
		}
	}

	// 2. Delimited / Scalar parsing
	tokens := parseDelimitedTokens(trimmed)
	if len(tokens) == 0 {
		return nil, nil
	}

	values := make([]any, len(tokens))
	for i, token := range tokens {
		values[i] = parseInferredValue(token)
	}
	return values, nil
}

// parseDelimitedTokens splits a comma-separated string while respecting quotes.
func parseDelimitedTokens(s string) []string {
	if !strings.ContainsAny(s, ",\"\\") {
		trimmed := strings.TrimSpace(s)
		if trimmed == "" {
			return nil
		}
		return []string{trimmed}
	}

	var tokens []string
	var cur strings.Builder
	inQuote := false
	escape := false

	for i := 0; i < len(s); i++ {
		ch := s[i]
		if escape {
			cur.WriteByte(ch)
			escape = false
			continue
		}
		if ch == '\\' {
			escape = true
			continue
		}
		if ch == '"' {
			inQuote = !inQuote
			continue
		}
		if ch == ',' && !inQuote {
			tokens = append(tokens, strings.TrimSpace(cur.String()))
			cur.Reset()
			continue
		}
		cur.WriteByte(ch)
	}
	if cur.Len() > 0 || len(tokens) > 0 {
		tokens = append(tokens, strings.TrimSpace(cur.String()))
	}
	return tokens
}

// parseInferredValue converts a string token into an appropriate scalar Go value.
// It infers numbers as float64 (matching standard json unmarshal behavior).
func parseInferredValue(token string) any {
	// 1. Boolean check
	if strings.EqualFold(token, "true") {
		return true
	}
	if strings.EqualFold(token, "false") {
		return false
	}

	// 2. Number check (integer or float) -> parse as float64
	if num, err := strconv.ParseFloat(token, 64); err == nil {
		return num
	}

	// 3. String literal
	return token
}

// BuildDynamicKeyset generates a nested OR/AND SQL condition for keyset pagination.
// It returns a raw SQL string and a slice of arguments ready to be passed to a WHERE clause.
//
// Example for columns=["name", "status"], operators=[">", "<"], values=["App A", 1]:
// Returns SQL: "(name > ?) OR (name = ? AND status < ?)"
// Returns Args: ["App A", "App A", 1]
func BuildDynamicKeyset(columns []string, operators []string, values []any) (string, []any) {
	n := min(len(columns), len(operators), len(values))
	if n == 0 {
		return "", nil
	}

	if n == 1 {
		return "(" + columns[0] + " " + operators[0] + " ?)", []any{values[0]}
	}

	totalArgs := n * (n + 1) / 2
	finalArgs := make([]any, 0, totalArgs)

	var b strings.Builder
	b.Grow(n * 35)

	for i := 0; i < n; i++ {
		if i > 0 {
			b.WriteString(" OR ")
		}
		b.WriteByte('(')
		for j := 0; j < i; j++ {
			b.WriteString(columns[j])
			b.WriteString(" = ? AND ")
			finalArgs = append(finalArgs, values[j])
		}
		b.WriteString(columns[i])
		b.WriteByte(' ')
		b.WriteString(operators[i])
		b.WriteString(" ?)")
		finalArgs = append(finalArgs, values[i])
	}

	return b.String(), finalArgs
}

// InvertSort inverts SQL operators and sort directions for keyset backward traversal.
// Example: "<" becomes ">", "DESC" becomes "ASC".
func InvertSort(operators, orderStrs []string) ([]string, []string) {
	newOps := make([]string, len(operators))
	newOrders := make([]string, len(orderStrs))
	for i, op := range operators {
		switch op {
		case "<":
			newOps[i] = ">"
		case "<=":
			newOps[i] = ">="
		case ">=":
			newOps[i] = "<="
		default:
			newOps[i] = "<"
		}
	}
	for i, order := range orderStrs {
		up := strings.ToUpper(order)
		switch {
		case strings.HasSuffix(up, " DESC"):
			newOrders[i] = order[:len(order)-5] + " ASC"
		case strings.HasSuffix(up, " ASC"):
			newOrders[i] = order[:len(order)-4] + " DESC"
		default:
			newOrders[i] = order
		}
	}
	return newOps, newOrders
}

// GenerateBidirectionalCursor abstracts all the complex logic to create a CursorPagination response.
//   - items: the array of structs returned by the DB
//   - limit: the limit specified in the request
//   - direction: "next" or "prev"
//   - currentCursor: the cursor passed in the request
//   - extractFn: a callback to extract []any keyset values from a single item
func GenerateBidirectionalCursor[T any](items []T, limit int, direction, currentCursor string, extractFn func(T) []any) *CursorPagination {
	var nextCursor, prevCursor string
	hasNext := false

	dir := strings.ToLower(strings.TrimSpace(direction))
	if dir != "prev" {
		dir = "next"
	}

	if len(items) > 0 && extractFn != nil {
		showPrev := true
		if dir == "next" && currentCursor == "" {
			showPrev = false
		}
		if dir == "prev" && limit > 0 && len(items) < limit {
			showPrev = false
		}

		if showPrev {
			prevVals := extractFn(items[0])
			prevCursor, _ = EncodeDynamicCursor(prevVals...)
		}

		if dir == "next" {
			hasNext = limit > 0 && len(items) == limit
		} else {
			hasNext = true
		}

		if hasNext {
			nextVals := extractFn(items[len(items)-1])
			nextCursor, _ = EncodeDynamicCursor(nextVals...)
		}
	}

	p := NewCursorFromInt(limit, nextCursor, prevCursor, hasNext)
	return &p
}

// prepareDynamicSort parses sort strings, validates them against allowed columns,
// appends the unique tie-breaker, and handles bidirectional inversion.
func prepareDynamicSort(sortBy, sortType, direction string, cfg KeysetConfig) (columns, operators, orderStrs []string) {
	trimmedSortBy := strings.TrimSpace(sortBy)
	if trimmedSortBy == "" {
		if cfg.UniqueColumn != "" {
			st := strings.ToLower(strings.TrimSpace(cfg.UniqueSortType))
			if st == "" {
				st = "desc"
			}
			op := ">"
			if st == "desc" {
				op = "<"
			}
			cols := []string{cfg.UniqueColumn}
			ops := []string{op}
			orders := []string{cfg.UniqueColumn + " " + strings.ToUpper(st)}
			if strings.EqualFold(strings.TrimSpace(direction), "prev") {
				ops, orders = InvertSort(ops, orders)
			}
			return cols, ops, orders
		}
		return nil, nil, nil
	}

	sortBys := strings.Split(trimmedSortBy, ",")
	sortTypes := strings.Split(sortType, ",")

	capHint := len(sortBys) + 1
	columns = make([]string, 0, capHint)
	operators = make([]string, 0, capHint)
	orderStrs = make([]string, 0, capHint)
	seenCols := make(map[string]bool, capHint)

	for i, rawCol := range sortBys {
		rawCol = strings.TrimSpace(rawCol)
		if rawCol == "" {
			continue
		}

		dbCol, ok := cfg.AllowedColumns[strings.ToLower(rawCol)]
		if !ok {
			continue
		}

		if seenCols[dbCol] {
			continue
		}
		seenCols[dbCol] = true

		sortT := "asc"
		if i < len(sortTypes) {
			st := strings.ToLower(strings.TrimSpace(sortTypes[i]))
			if st == "desc" {
				sortT = "desc"
			}
		}

		op := ">"
		if sortT == "desc" {
			op = "<"
		}

		columns = append(columns, dbCol)
		operators = append(operators, op)
		orderStrs = append(orderStrs, dbCol+" "+strings.ToUpper(sortT))
	}

	// Always append unique column at the end
	if cfg.UniqueColumn != "" && !seenCols[cfg.UniqueColumn] {
		seenCols[cfg.UniqueColumn] = true
		columns = append(columns, cfg.UniqueColumn)
		st := strings.ToLower(strings.TrimSpace(cfg.UniqueSortType))
		if st == "" {
			st = "desc"
		}
		op := ">"
		if st == "desc" {
			op = "<"
		}
		operators = append(operators, op)
		orderStrs = append(orderStrs, cfg.UniqueColumn+" "+strings.ToUpper(st))
	}

	if strings.EqualFold(strings.TrimSpace(direction), "prev") {
		operators, orderStrs = InvertSort(operators, orderStrs)
	}

	return columns, operators, orderStrs
}

// DynamicCursorRequest holds the standard fields required for dynamic keyset pagination in API requests.
// It can be embedded into any DTO to instantly support bidirectional cursor pagination.
type DynamicCursorRequest struct {
	// SortBy supports multiple columns separated by comma (e.g. "status,name").
	SortBy string `json:"sort_by" query:"sort_by"`
	// SortType supports multiple directions separated by comma (e.g. "asc,desc").
	SortType string `json:"sort_type" query:"sort_type"`
	// Cursor expects a Base64 encoded string returned from the previous response. Leave empty for the first page.
	Cursor string `json:"cursor" query:"cursor"`
	// Direction expects either "next" or "prev". Defaults to "next" if empty.
	Direction string `json:"direction" query:"direction"`
	// Limit expects an integer to determine how many records to fetch (e.g. 10).
	Limit int `json:"limit" query:"limit"`
	// ShowPagination expects a boolean (true/false) to toggle pagination metadata in the response.
	ShowPagination bool `json:"show_pagination" query:"show_pagination"`
}

// GetDirection returns the normalized direction ("next" or "prev"). Defaults to "next".
func (r DynamicCursorRequest) GetDirection() string {
	if strings.EqualFold(strings.TrimSpace(r.Direction), "prev") {
		return "prev"
	}
	return "next"
}

// BindDynamicCursorRequest extracts standard pagination parameters from an HTTP request.
// It uses safe default values if the parameters are not provided in the query string.
func BindDynamicCursorRequest(r *http.Request) DynamicCursorRequest {
	if r == nil || r.URL == nil {
		return DynamicCursorRequest{Direction: "next", ShowPagination: true}
	}
	return bindDynamicCursorRequestFromQuery(r.URL.Query())
}

// KeysetConfig holds the configuration for generating dynamic keyset SQL clauses.
type KeysetConfig struct {
	// AllowedColumns maps user-facing field names (case-insensitive) to database column names.
	AllowedColumns map[string]string
	// UniqueColumn is the mandatory tie-breaker database column (e.g. "id" or "user_id").
	UniqueColumn string
	// UniqueSortType defines the sort direction for the tie-breaker: "ASC" or "DESC" (defaults to "DESC").
	UniqueSortType string
	// Detector is an optional TypeDetector to normalize cursor values into native SQL types.
	// If nil and AutoDetectTypes is true, DefaultDetector is used.
	Detector *TypeDetector
	// AutoDetectTypes toggles automatic cursor value normalization against database column types.
	AutoDetectTypes bool
}

// KeysetQuery contains the generated WHERE condition, arguments, and ORDER BY clauses for SQL execution.
type KeysetQuery struct {
	// Where is the generated SQL WHERE expression (e.g. "(user_name > ?) OR (user_name = ? AND user_id < ?)"), or empty if no cursor.
	Where string
	// Args contains arguments corresponding to placeholders in the Where expression.
	Args []any
	// OrderStrs contains the SQL ORDER BY clauses (e.g. ["user_name ASC", "user_id DESC"]).
	OrderStrs []string
	// Columns contains the database columns included in the keyset condition.
	Columns []string
	// Operators contains the comparison operators applied for each column.
	Operators []string
}

// HasWhere reports whether a keyset WHERE filter was generated.
func (q KeysetQuery) HasWhere() bool {
	return q.Where != ""
}

// BuildKeysetQueryFromValues builds a complete KeysetQuery from explicit sort parameters and already-decoded cursor values.
func BuildKeysetQueryFromValues(sortBy, sortType, direction string, cursorValues []any, cfg KeysetConfig) KeysetQuery {
	cols, ops, orders := prepareDynamicSort(sortBy, sortType, direction, cfg)

	var sqlWhere string
	var sqlArgs []any
	if len(cursorValues) > 0 {
		vals := cursorValues
		if cfg.AutoDetectTypes || cfg.Detector != nil {
			d := DefaultDetector
			if cfg.Detector != nil {
				d = cfg.Detector
			}
			if normalized, err := d.NormalizeCursorValues(cols, cursorValues); err == nil {
				vals = normalized
			}
		}
		sqlWhere, sqlArgs = BuildDynamicKeyset(cols, ops, vals)
	}

	return KeysetQuery{
		Where:     sqlWhere,
		Args:      sqlArgs,
		OrderStrs: orders,
		Columns:   cols,
		Operators: ops,
	}
}

// BuildKeysetQuery generates a complete KeysetQuery directly from a DynamicCursorRequest and KeysetConfig.
// If req.Cursor is not empty, it automatically decodes the cursor into values before building the WHERE clause.
func BuildKeysetQuery(req DynamicCursorRequest, cfg KeysetConfig) (KeysetQuery, error) {
	var cursorVals []any
	if req.Cursor != "" {
		vals, err := DecodeDynamicCursor(req.Cursor)
		if err != nil {
			return KeysetQuery{}, fmt.Errorf("decoding dynamic cursor: %w", err)
		}
		cursorVals = vals
	}

	cols, ops, orders := prepareDynamicSort(req.SortBy, req.SortType, req.Direction, cfg)

	var sqlWhere string
	var sqlArgs []any
	if len(cursorVals) > 0 {
		vals := cursorVals
		if cfg.AutoDetectTypes || cfg.Detector != nil {
			d := DefaultDetector
			if cfg.Detector != nil {
				d = cfg.Detector
			}
			normVals, err := d.NormalizeCursorValues(cols, cursorVals)
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


// BuildKeysetQuery builds a KeysetQuery directly on DynamicCursorRequest.
func (r DynamicCursorRequest) BuildKeysetQuery(cfg KeysetConfig) (KeysetQuery, error) {
	return BuildKeysetQuery(r, cfg)
}

// FinalizeCursor automatically reverses items if navigating backwards ("prev") and generates bidirectional CursorPagination metadata.
// It returns the correctly-ordered items slice and the pagination metadata.
func FinalizeCursor[T any](items []T, req DynamicCursorRequest, extractFn func(T) []any) ([]T, *CursorPagination) {
	if req.GetDirection() == "prev" {
		items = sliceutil.Reverse(items)
	}
	meta := GenerateBidirectionalCursor(items, req.Limit, req.Direction, req.Cursor, extractFn)
	return items, meta
}

// FieldExtractor dynamically extracts keyset values from an item struct based on requested sort columns
// and a mandatory unique tie-breaker column.
type FieldExtractor[T any] struct {
	getters      map[string]func(T) any
	uniqueGetter func(T) any
}

// NewFieldExtractor creates a new FieldExtractor for type T.
// getters maps field names (case-insensitive) to getter functions.
// uniqueGetter extracts the mandatory tie-breaker value (e.g. ID).
func NewFieldExtractor[T any](getters map[string]func(T) any, uniqueGetter func(T) any) *FieldExtractor[T] {
	normalized := make(map[string]func(T) any, len(getters))
	for k, v := range getters {
		normalized[strings.ToLower(strings.TrimSpace(k))] = v
	}
	return &FieldExtractor[T]{
		getters:      normalized,
		uniqueGetter: uniqueGetter,
	}
}

func (e *FieldExtractor[T]) resolveGetters(sortBy string) []func(T) any {
	trimmed := strings.TrimSpace(sortBy)
	if trimmed == "" {
		if e.uniqueGetter != nil {
			return []func(T) any{e.uniqueGetter}
		}
		return nil
	}

	parts := strings.Split(trimmed, ",")
	resolved := make([]func(T) any, 0, len(parts)+1)
	for _, part := range parts {
		col := strings.ToLower(strings.TrimSpace(part))
		if getter, ok := e.getters[col]; ok && getter != nil {
			resolved = append(resolved, getter)
		}
	}
	if e.uniqueGetter != nil {
		resolved = append(resolved, e.uniqueGetter)
	}
	return resolved
}

// Extract extracts keyset values for item in the exact order specified by sortBy, ending with uniqueGetter.
func (e *FieldExtractor[T]) Extract(item T, sortBy string) []any {
	resolved := e.resolveGetters(sortBy)
	vals := make([]any, len(resolved))
	for i, getter := range resolved {
		vals[i] = getter(item)
	}
	return vals
}

// Fn returns an extractor callback optimized for batch execution with zero per-item string parsing or map lookups.
func (e *FieldExtractor[T]) Fn(sortBy string) func(T) []any {
	resolved := e.resolveGetters(sortBy)
	return func(item T) []any {
		vals := make([]any, len(resolved))
		for i, getter := range resolved {
			vals[i] = getter(item)
		}
		return vals
	}
}
