package pagination

import (
	"encoding/base64"
	"fmt"
	"net/http"
	"strconv"
	"strings"

	"github.com/Jkenyut/nvx-go-helper/request"
	"github.com/bytedance/sonic"
)

// EncodeDynamicCursor takes arbitrary values (e.g., from the last row of a query)
// and encodes them into a base64 JSON array string to be used as a cursor.
// It uses URLEncoding to be safe for HTTP query parameters.
func EncodeDynamicCursor(values ...interface{}) (string, error) {
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
func DecodeDynamicCursor(cursor string) ([]interface{}, error) {
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

	var values []interface{}
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
func BuildDynamicKeyset(columns []string, operators []string, values []interface{}) (string, []interface{}) {
	n := min(len(columns), len(operators), len(values))
	if n == 0 {
		return "", nil
	}

	var orClauses []string
	var finalArgs []interface{}

	for i := 0; i < n; i++ {
		var andClauses []string

		// Add equals for all preceding columns
		for j := 0; j < i; j++ {
			andClauses = append(andClauses, fmt.Sprintf("%s = ?", columns[j]))
			finalArgs = append(finalArgs, values[j])
		}

		// Add operator for the current column
		andClauses = append(andClauses, fmt.Sprintf("%s %s ?", columns[i], operators[i]))
		finalArgs = append(finalArgs, values[i])

		// Join AND clauses for this block
		if len(andClauses) == 1 {
			orClauses = append(orClauses, fmt.Sprintf("(%s)", andClauses[0]))
		} else {
			orClauses = append(orClauses, fmt.Sprintf("(%s)", strings.Join(andClauses, " AND ")))
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
		if op == "<" {
			newOps[i] = ">"
		} else {
			newOps[i] = "<"
		}
	}
	for i, order := range orderStrs {
		up := strings.ToUpper(order)
		if strings.HasSuffix(up, " DESC") {
			newOrders[i] = order[:len(order)-5] + " ASC"
		} else if strings.HasSuffix(up, " ASC") {
			newOrders[i] = order[:len(order)-4] + " DESC"
		} else {
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
//   - extractFn: a callback to extract []interface{} keyset values from a single item
func GenerateBidirectionalCursor[T any](items []T, limit int, direction, currentCursor string, extractFn func(T) []interface{}) *CursorPagination {
	var nextCursor, prevCursor string
	hasNext := false

	if len(items) > 0 {
		showPrev := true
		if direction == "next" && currentCursor == "" {
			showPrev = false
		}
		if direction == "prev" && len(items) < limit {
			showPrev = false
		}

		if showPrev {
			prevVals := extractFn(items[0])
			prevCursor, _ = EncodeDynamicCursor(prevVals...)
		}

		if direction == "next" {
			hasNext = len(items) == limit
		} else {
			hasNext = true
		}

		if hasNext {
			nextVals := extractFn(items[len(items)-1])
			nextCursor, _ = EncodeDynamicCursor(nextVals...)
		}
	}

	limitStr := strconv.Itoa(limit)
	p := NewCursor(limitStr, nextCursor, prevCursor, hasNext)
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

// DynamicSortResult holds the resulting SQL columns, operators, and ORDER BY clauses.
type DynamicSortResult struct {
	Columns   []string
	Operators []string
	OrderStrs []string
}

// PrepareDynamicSort parses sort strings, validates them against allowed columns,
// appends the unique tie-breaker, and handles bidirectional inversion.
func PrepareDynamicSort(params DynamicSortParams) DynamicSortResult {
	var columns, operators, orderStrs []string

	sortBys := strings.Split(params.SortBy, ",")
	sortTypes := strings.Split(params.SortType, ",")

	for i, rawCol := range sortBys {
		rawCol = strings.TrimSpace(rawCol)
		if rawCol == "" {
			continue
		}

		dbCol, ok := params.AllowedColumns[strings.ToLower(rawCol)]
		if !ok {
			continue
		}

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
		orderStrs = append(orderStrs, fmt.Sprintf("%s %s", dbCol, strings.ToUpper(sortType)))
	}

	// Always append unique column at the end
	if params.UniqueColumn != "" {
		hasUnique := false
		for _, col := range columns {
			if col == params.UniqueColumn {
				hasUnique = true
				break
			}
		}

		if !hasUnique {
			columns = append(columns, params.UniqueColumn)
			st := strings.ToLower(params.UniqueSortType)
			if st == "" {
				st = "desc" // default to desc if not specified
			}
			op := ">"
			if st == "desc" {
				op = "<"
			}
			operators = append(operators, op)
			orderStrs = append(orderStrs, fmt.Sprintf("%s %s", params.UniqueColumn, strings.ToUpper(st)))
		}
	}

	if params.Direction == "prev" {
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
	SortBy         string `json:"sort_by" query:"sort_by"`
	// SortType supports multiple directions separated by comma (e.g. "asc,desc").
	SortType       string `json:"sort_type" query:"sort_type"`
	// Cursor expects a Base64 encoded string returned from the previous response. Leave empty for the first page.
	Cursor         string `json:"cursor" query:"cursor"`
	// Direction expects either "next" or "prev". Defaults to "next" if empty.
	Direction      string `json:"direction" query:"direction"`
	// Limit expects an integer to determine how many records to fetch (e.g. 10).
	Limit          int    `json:"limit" query:"limit"`
	// ShowPagination expects a boolean (true/false) to toggle pagination metadata in the response.
	ShowPagination bool   `json:"show_pagination" query:"show_pagination"`
}

// BindDynamicCursorRequest extracts standard pagination parameters from an HTTP request.
// It uses safe default values if the parameters are not provided in the query string.
func BindDynamicCursorRequest(r *http.Request) DynamicCursorRequest {
	return DynamicCursorRequest{
		SortBy:         request.GetQueryString(r, "sort_by", ""),
		SortType:       request.GetQueryString(r, "sort_type", ""),
		Cursor:         request.GetQueryString(r, "cursor", ""),
		Direction:      request.GetQueryString(r, "direction", "next"),
		Limit:          request.GetQueryInt(r, "limit", 10),
		ShowPagination: request.GetQueryBool(r, "show_pagination", true),
	}
}
