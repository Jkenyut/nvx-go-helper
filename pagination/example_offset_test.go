package pagination_test

import (
	"fmt"
	"net/http"

	"github.com/Jkenyut/nvx-go-helper/pagination"
	"github.com/Jkenyut/nvx-go-helper/request"
)

// This is a standard DTO on the application side (e.g., in /internal/dto).
type ProductGetRequest struct {
	Search string `json:"search"`

	// 1. Simply embed this struct, and the DTO automatically supports Offset Pagination
	pagination.OffsetRequest
}

func Example_offsetPagination() {
	// =====================================================================
	// 2. INSIDE THE HANDLER:
	// =====================================================================
	req := &http.Request{} // Simulated HTTP Request

	// Automatically extract all query parameters (page, limit, sort_by, sort_type, etc.)
	dto := ProductGetRequest{
		Search:        request.GetQueryString(req, "search", ""),
		OffsetRequest: pagination.BindOffsetRequest(req),
	}

	fmt.Printf("Requested Page: %d, Limit: %d\n", dto.Page, dto.Limit)

	// =====================================================================
	// 3. INSIDE THE REPOSITORY (Find All Function):
	// =====================================================================
	// Suppose you queried the DB to get the total count first:
	totalCount := 150

	// Create a safe pagination object using the requested page and limit
	pageData := pagination.NewFromInt(dto.Page, dto.Limit, totalCount)

	// Extract the safe SQL OFFSET value
	sqlOffset := pageData.Offset()
	sqlLimit := pageData.Limit

	fmt.Printf("SQL LIMIT %d OFFSET %d\n", sqlLimit, sqlOffset)

	// Simulated Query Builder (Squirrel/GORM):
	// q = q.Limit(uint64(sqlLimit)).Offset(uint64(sqlOffset))
	// err := q.QueryRow().Scan(&products)

	// =====================================================================
	// 4. INSIDE THE SERVICE / RESPONSE:
	// =====================================================================

	// You can easily format this metadata into your JSON response
	fmt.Printf("Total Pages: %d\n", pageData.TotalPages)
	fmt.Printf("Has Next: %v\n", pageData.HasNext)

	// If you want to include standard RFC 5988 HTTP Link headers in your response
	// you can build it easily:
	baseURL := "https://api.example.com/products"
	linkHeaders, _ := pageData.Links(baseURL)
	_ = linkHeaders // iterate and set w.Header().Set("Link", ...)
}
