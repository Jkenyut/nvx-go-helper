package pagination_test

import (
	"fmt"
	"net/http"

	"github.com/Jkenyut/nvx-go-helper/pagination"
	"github.com/Jkenyut/nvx-go-helper/request"
	"github.com/Jkenyut/nvx-go-helper/sliceutil"
	"strings"
)

// This is a standard DTO on the application side (e.g., in /internal/dto).
type UserGetRequest struct {
	Search string `json:"search"`

	// 1. Simply embed this struct, and the DTO automatically supports Dynamic Cursor
	pagination.DynamicCursorRequest
}

func Example_dynamicCursor() {
	// =====================================================================
	// 2. INSIDE THE HANDLER:
	// =====================================================================
	// You only need to retrieve specific data (e.g., Search), the rest is handled by the helper
	req := &http.Request{} // Simulated HTTP Request
	dto := UserGetRequest{
		Search:               request.GetQueryString(req, "search", ""),
		DynamicCursorRequest: pagination.BindDynamicCursorRequest(req),
	}

	fmt.Println("Limit from handler:", dto.Limit)

	// =====================================================================
	// 3. INSIDE THE REPOSITORY (Find All Function):
	// =====================================================================
	// Configure allowed columns for sorting (Prevents SQL Injection)
	allowedSort := map[string]string{
		"name":       "user_name",
		"created_at": "user_created_at",
	}

	// a. Prepare Dynamic Sort SQL & Reversal Logic (Prev/Next)
	sortRes := pagination.PrepareDynamicSort(pagination.DynamicSortParams{
		SortBy:         dto.SortBy,
		SortType:       dto.SortType,
		Direction:      dto.Direction,
		AllowedColumns: allowedSort,
		UniqueColumn:   "user_id", // Mandatory tie-breaker
		UniqueSortType: "DESC",
	})

	var sqlWhere string
	var sqlArgs []interface{}

	// b. Extract Cursor Values if a cursor is sent by the user
	var cursorVals []interface{}
	if dto.Cursor != "" {
		cursorVals, _ = pagination.DecodeDynamicCursor(dto.Cursor)
	}

	// c. Build Keyset Pagination Condition (WHERE clause)
	if len(cursorVals) > 0 {
		sqlWhere, sqlArgs = pagination.BuildDynamicKeyset(sortRes.Columns, sortRes.Operators, cursorVals)
	}

	// Simulated Query Builder (Squirrel/GORM):
	// q = q.Where(sqlWhere, sqlArgs...)
	// for _, order := range sortRes.OrderStrs { q = q.OrderBy(order) }
	// q = q.Limit(dto.Limit)
	_ = sqlWhere
	_ = sqlArgs

	// =====================================================================
	// 4. INSIDE THE SERVICE (After Repository returns data array):
	// =====================================================================
	type User struct {
		ID        int
		Name      string
		CreatedAt string
	}

	// Simulated query results from DB
	users := []User{
		{ID: 10, Name: "Andi"},
		{ID: 9, Name: "Budi"},
	}

	// a. You must reverse the array if navigating backwards (Prev)
	if dto.Direction == "prev" {
		users = sliceutil.Reverse(users)
	}

	// b. Automatically Generate Bidirectional Cursor!
	pageRes := pagination.GenerateBidirectionalCursor(
		users,
		dto.Limit,
		dto.Direction,
		dto.Cursor,
		func(u User) []interface{} {
			// You only need to define how to extract values from the struct.
			// The order MUST EXACTLY MATCH the sort configuration.
			var vals []interface{}
			
			sortBys := strings.Split(dto.SortBy, ",")
			for _, col := range sortBys {
				col = strings.ToLower(strings.TrimSpace(col))
				switch col {
				case "name":
					vals = append(vals, u.Name)
				case "created_at":
					vals = append(vals, u.CreatedAt)
				}
			}
			
			vals = append(vals, u.ID) // UniqueColumn (tie-breaker) must be placed at the end
			return vals
		},
	)

	fmt.Printf("Has Next Page? %v\n", pageRes.HasNext)
}
