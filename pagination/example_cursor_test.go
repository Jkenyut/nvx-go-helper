package pagination_test

import (
	"fmt"
	"net/http"

	"github.com/Jkenyut/nvx-go-helper/pagination"
	"github.com/Jkenyut/nvx-go-helper/request"
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
	req := &http.Request{} // Simulated HTTP Request
	dto := UserGetRequest{
		Search:               request.GetQueryString(req, "search", ""),
		DynamicCursorRequest: pagination.BindDynamicCursorRequest(req),
	}

	fmt.Println("Limit from handler:", dto.Limit)

	// =====================================================================
	// 3. INSIDE THE REPOSITORY (Find All Function):
	// =====================================================================
	// BuildKeysetQuery automatically parses SortBy, validates allowed columns,
	// handles Prev/Next operator inversion, and decodes the cursor into WHERE conditions.
	keyset, err := dto.BuildKeysetQuery(pagination.KeysetConfig{
		AllowedColumns: map[string]string{
			"name":       "user_name",
			"created_at": "user_created_at",
		},
		UniqueColumn:   "user_id", // Mandatory tie-breaker
		UniqueSortType: "DESC",
	})
	if err != nil {
		fmt.Println("Cursor decoding error:", err)
		return
	}

	// Simulated Query Builder (Squirrel/GORM):
	// if keyset.HasWhere() {
	//     q = q.Where(keyset.Where, keyset.Args...)
	// }
	// for _, order := range keyset.OrderStrs {
	//     q = q.OrderBy(order)
	// }
	// q = q.Limit(dto.Limit)
	_ = keyset

	// =====================================================================
	// 4. INSIDE THE SERVICE / HANDLER RESPONSE:
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

	// Create a reusable field extractor to extract keyset values dynamically
	extractor := pagination.NewFieldExtractor(map[string]func(u User) any{
		"name":       func(u User) any { return u.Name },
		"created_at": func(u User) any { return u.CreatedAt },
	}, func(u User) any { return u.ID })

	// NewFinalizedCursorResponse automatically:
	// 1. Reverses users slice if navigating backwards ("prev")
	// 2. Generates bidirectional cursor metadata (next_cursor, prev_cursor, has_next)
	// 3. Wraps into standardized CursorListResponse[T]
	resp := pagination.NewFinalizedCursorResponse(users, dto.DynamicCursorRequest, extractor.Fn(dto.SortBy))

	fmt.Printf("Has Next Page? %v\n", resp.Pagination.HasNext)
}
