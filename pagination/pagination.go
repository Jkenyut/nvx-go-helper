// Package pagination provides a robust, production-ready, and highly reusable
// pagination system for REST APIs.
//
// Features (enterprise standard):
//   - Cursor-based & offset-based support
//   - Non-opinionated limit handling (no forced default limit)
//   - Safe page defaults (page=1)
//   - Helper methods: Offset, HasNext, Links (RFC 5988)
//   - Immutable design when possible
//   - Zero dependencies
//
// Example JSON response:
//
//	{
//	  "data": [ ... ],
//	  "pagination": {
//	    "page": 2,
//	    "limit": 20,
//	    "total": 1337,
//	    "total_pages": 67,
//	    "has_next": true,
//	    "has_prev": true,
//	    "next_page": 3,
//	    "prev_page": 1
//	  }
//	}
package pagination

import (
	"fmt"
	"net/http"
	"net/url"
	"strconv"

	"github.com/Jkenyut/nvx-go-helper/request"
)

// Default values for pagination
const (
	DefaultPage  = 1     // Default to first page
	DefaultLimit = 10    // Default limit per page
	MaxLimit     = 99999 // Maximum limit per page
)

// Pagination represents offset-based pagination metadata.
type Pagination struct {
	Page       int `json:"page"`        // Current page (1-based)
	Limit      int `json:"limit"`       // Items per page
	Total      int `json:"total"`       // Total items in database
	TotalPages int `json:"total_pages"` // Total number of pages

	// Navigation helpers
	HasNext  bool `json:"has_next"`
	HasPrev  bool `json:"has_prev"`
	NextPage int  `json:"next_page,omitempty"`
	PrevPage int  `json:"prev_page,omitempty"`
}

// New creates a new Pagination from request parameters.
// Automatically sanitizes and applies safe defaults.
// Used in Gin, Fiber, Echo, Chi handlers.
//
// Example:
//
//	p := pagination.New(c.Query("page"), c.Query("limit"), totalCount)
//	offset := p.Offset()
//	rows, _ := db.Limit(p.Limit).Offset(offset).Find(&users)
func New(pageStr, limitStr string, total int) Pagination {
	// Parse strings to integers with defaults
	page := parseInt(pageStr, DefaultPage)
	limit := parseInt(limitStr, 0)
	return NewFromInt(page, limit, total)
}

// NewFromInt creates a new Pagination from integer parameters.
// Automatically sanitizes and applies safe defaults without forcing an arbitrary limit.
func NewFromInt(page, limit, total int) Pagination {
	// Sanitize Inputs
	if page < 1 {
		page = DefaultPage
	}
	if limit < 0 {
		limit = 0
	}

	p := Pagination{
		Page:  page,
		Limit: limit,
		Total: total,
	}

	// Calculate derived fields
	p.TotalPages = p.calculateTotalPages()

	// Cap page to TotalPages to prevent ghost pages
	if p.TotalPages > 0 && p.Page > p.TotalPages {
		p.Page = p.TotalPages
	}

	p.HasNext = p.TotalPages > 0 && p.Page < p.TotalPages
	p.HasPrev = p.Page > 1
	p.NextPage = p.Page + 1
	p.PrevPage = p.Page - 1

	// Omit next/prev indices if they don't exist
	if !p.HasNext {
		p.NextPage = 0
	}
	if !p.HasPrev {
		p.PrevPage = 0
	}

	return p
}

// OffsetRequest holds the standard fields required for offset-based pagination in API requests.
// It can be embedded into any DTO to instantly support offset pagination.
type OffsetRequest struct {
	// SortBy supports multiple columns separated by comma (e.g. "status,name").
	SortBy string `json:"sort_by" query:"sort_by"`
	// SortType supports multiple directions separated by comma (e.g. "asc,desc").
	SortType string `json:"sort_type" query:"sort_type"`
	// Page expects an integer representing the page number. Starts from 1.
	Page int `json:"page" query:"page"`
	// Limit expects an integer to determine how many records to fetch per page (e.g. 10).
	Limit int `json:"limit" query:"limit"`
	// ShowPagination expects a boolean (true/false) to toggle pagination metadata in the response.
	ShowPagination bool `json:"show_pagination" query:"show_pagination"`
}

// BindOffsetRequest extracts standard offset pagination parameters from an HTTP request.
// If limit is not specified in the query, it defaults to 0 (no limit forced by helper).
func BindOffsetRequest(r *http.Request) OffsetRequest {
	return OffsetRequest{
		SortBy:         request.GetQueryString(r, "sort_by", ""),
		SortType:       request.GetQueryString(r, "sort_type", ""),
		Page:           request.GetQueryInt(r, "page", DefaultPage),
		Limit:          request.GetQueryInt(r, "limit", 0),
		ShowPagination: request.GetQueryBool(r, "show_pagination", true),
	}
}

// Offset returns SQL OFFSET value (0-based)
// Formula: (page - 1) * limit
func (p Pagination) Offset() int {
	if p.Limit <= 0 {
		return 0
	}
	return (p.Page - 1) * p.Limit
}

// calculateTotalPages computes ceil(total / limit)
func (p Pagination) calculateTotalPages() int {
	// Prevent division by zero
	if p.Limit <= 0 {
		return 0
	}
	// Use integer arithmetic to calculate total pages (ceil)
	return (p.Total + p.Limit - 1) / p.Limit
}

// Links generates RFC 5988 Link headers.
// Useful for HATEOAS compliance.
func (p Pagination) Links(baseURL string) (map[string]string, error) {
	u, err := url.Parse(baseURL)
	if err != nil {
		return nil, err
	}

	q := u.Query()
	q.Set("limit", strconv.Itoa(p.Limit))

	links := make(map[string]string)

	if p.HasPrev {
		q.Set("page", strconv.Itoa(p.PrevPage))
		u.RawQuery = q.Encode()
		links["prev"] = fmt.Sprintf(`<%s>; rel="prev"`, u.String())
	}

	if p.HasNext {
		q.Set("page", strconv.Itoa(p.NextPage))
		u.RawQuery = q.Encode()
		links["next"] = fmt.Sprintf(`<%s>; rel="next"`, u.String())
	}

	return links, nil
}

// parseInt safely converts string to int with fallback
func parseInt(s string, fallback int) int {
	if s == "" {
		return fallback
	}
	val, err := strconv.Atoi(s)
	// Return fallback on parsing error
	if err != nil {
		return fallback
	}
	return val
}

// CursorPagination represents cursor-based pagination metadata.
type CursorPagination struct {
	Limit      int    `json:"limit"`                 // Items per page
	NextCursor string `json:"next_cursor,omitempty"` // Opaque cursor for next page
	PrevCursor string `json:"prev_cursor,omitempty"` // Opaque cursor for previous page
	HasNext    bool   `json:"has_next"`              // Whether there is a next page
}

// NewCursor creates a new CursorPagination without forcing an arbitrary default limit.
func NewCursor(limitStr string, nextCursor string, prevCursor string, hasNext bool) CursorPagination {
	limit := parseInt(limitStr, 0)
	if limit < 0 {
		limit = 0
	}

	return CursorPagination{
		Limit:      limit,
		NextCursor: nextCursor,
		PrevCursor: prevCursor,
		HasNext:    hasNext,
	}
}
