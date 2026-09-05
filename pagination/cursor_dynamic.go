package pagination

import (
	"encoding/base64"
	"fmt"
	"net/http"
	"strings"

	"github.com/Jkenyut/nvx-go-helper/request"
	"github.com/bytedance/sonic"
)

// EncodeDynamicCursor takes arbitrary values (e.g., from the last row of a query)
// and encodes them into a base64 JSON array string to be used as a cursor.
// It uses URLEncoding to be safe for HTTP query parameters.
func EncodeDynamicCursor(values ...any) (string, error) {
	if len(values) == 0 {
		return "", nil
	}
	b, err := sonic.Marshal(values)
	if err != nil {
		return "", fmt.Errorf("failed to encode cursor: %w", err)
	}
	return base64.URLEncoding.EncodeToString(b), nil
}

// DecodeDynamicCursor decodes a base64 JSON array string back into an array of values.
func DecodeDynamicCursor(cursor string) ([]any, error) {
	if cursor == "" {
		return nil, nil
	}

	// First try URLEncoding
	b, err := base64.URLEncoding.DecodeString(cursor)
	if err != nil {
		// Fallback to StdEncoding for backwards compatibility
		b, err = base64.StdEncoding.DecodeString(cursor)
		if err != nil {
			return nil, fmt.Errorf("invalid base64 cursor: %w", err)
		}
	}

	var values []any
	if err := sonic.Unmarshal(b, &values); err != nil {
		return nil, fmt.Errorf("failed to decode cursor values: %w", err)
	}

	return values, nil
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

	totalArgs := n * (n + 1) / 2
	orClauses := make([]string, 0, n)
	finalArgs := make([]any, 0, totalArgs)

	for i := 0; i < n; i++ {
		andClauses := make([]string, 0, i+1)

		// Add equals for all preceding columns
		for j := 0; j < i; j++ {
			andClauses = append(andClauses, columns[j]+" = ?")
			finalArgs = append(finalArgs, values[j])
		}

		// Add operator for the current column
		andClauses = append(andClauses, columns[i]+" "+operators[i]+" ?")
		finalArgs = append(finalArgs, values[i])

		// Join AND clauses for this block
		if len(andClauses) == 1 {
			orClauses = append(orClauses, "("+andClauses[0]+")")
		} else {
			orClauses = append(orClauses, "("+strings.Join(andClauses, " AND ")+")")
		}
	}

	sqlStr := strings.Join(orClauses, " OR ")
	return sqlStr, finalArgs
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

// DynamicSortParams holds configuration for generating dynamic keyset sorting.
type DynamicSortParams struct {
	SortBy         string            // User input e.g. "code,name"
	SortType       string            // User input e.g. "asc,desc"
	Direction      string            // User input "next" or "prev"
	AllowedColumns map[string]string // Map of allowed fields to DB columns
	UniqueColumn   string            // Tie-breaker DB column (e.g. "id")
	UniqueSortType string            // "ASC" or "DESC" for tie-breaker
}

// GetDirection returns the normalized direction ("next" or "prev"). Defaults to "next".
func (p DynamicSortParams) GetDirection() string {
	if strings.EqualFold(strings.TrimSpace(p.Direction), "prev") {
		return "prev"
	}
	return "next"
}

// DynamicSortResult holds the resulting SQL columns, operators, and ORDER BY clauses.
type DynamicSortResult struct {
	Columns   []string
	Operators []string
	OrderStrs []string
}

// PrepareDynamicSort parses sort strings, validates them against allowed columns,
// appends the unique tie-breaker, and handles bidirectional inversion.
func PrepareDynamicSort(params DynamicSortParams) DynamicSortResult {
	sortBys := strings.Split(params.SortBy, ",")
	sortTypes := strings.Split(params.SortType, ",")

	capHint := len(sortBys) + 1
	columns := make([]string, 0, capHint)
	operators := make([]string, 0, capHint)
	orderStrs := make([]string, 0, capHint)
	seenCols := make(map[string]bool, capHint)

	for i, rawCol := range sortBys {
		rawCol = strings.TrimSpace(rawCol)
		if rawCol == "" {
			continue
		}

		dbCol, ok := params.AllowedColumns[strings.ToLower(rawCol)]
		if !ok {
			continue
		}

		if seenCols[dbCol] {
			continue
		}
		seenCols[dbCol] = true

		sortType := "asc"
		if i < len(sortTypes) {
			st := strings.ToLower(strings.TrimSpace(sortTypes[i]))
			if st == "desc" {
				sortType = "desc"
			}
		}

		op := ">"
		if sortType == "desc" {
			op = "<"
		}

		columns = append(columns, dbCol)
		operators = append(operators, op)
		orderStrs = append(orderStrs, dbCol+" "+strings.ToUpper(sortType))
	}

	// Always append unique column at the end
	if params.UniqueColumn != "" && !seenCols[params.UniqueColumn] {
		seenCols[params.UniqueColumn] = true
		columns = append(columns, params.UniqueColumn)
		st := strings.ToLower(strings.TrimSpace(params.UniqueSortType))
		if st == "" {
			st = "desc" // default to desc if not specified
		}
		op := ">"
		if st == "desc" {
			op = "<"
		}
		operators = append(operators, op)
		orderStrs = append(orderStrs, params.UniqueColumn+" "+strings.ToUpper(st))
	}

	if params.GetDirection() == "prev" {
		operators, orderStrs = InvertSort(operators, orderStrs)
	}

	return DynamicSortResult{
		Columns:   columns,
		Operators: operators,
		OrderStrs: orderStrs,
	}
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
	return DynamicCursorRequest{
		SortBy:         request.GetQueryString(r, "sort_by", ""),
		SortType:       request.GetQueryString(r, "sort_type", ""),
		Cursor:         request.GetQueryString(r, "cursor", ""),
		Direction:      request.GetQueryString(r, "direction", "next"),
		Limit:          request.GetQueryInt(r, "limit", 0),
		ShowPagination: request.GetQueryBool(r, "show_pagination", true),
	}
}
