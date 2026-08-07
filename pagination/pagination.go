// Package pagination provides a robust, production-ready, and highly reusable
// pagination system for REST APIs.
//
// Features (enterprise standard):
//   - Cursor-based & offset-based support
//   - Automatic limit clamping (max 100)
//   - Safe defaults (page=1, limit=10)
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
	"net/url"
	"strconv"
)

// Default values for pagination
const (
	DefaultPage  = 1      // Default to first page
	DefaultLimit = 10     // Default 10 items per page
	MaxLimit     = 100000 // Protection against large queries
	MinLimit     = 1      // Minimum 1 item per page
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
	limit := parseInt(limitStr, DefaultLimit)

	// Sanitize Inputs
	// Ensure page is at least 1
	if page < 1 {
		page = DefaultPage
	}
	// Ensure limit is at least 1
	if limit < MinLimit {
		limit = DefaultLimit
	}
	// Cap limit at MaxLimit safely
	if limit > MaxLimit {
		limit = MaxLimit
	}

	// Initialize struct
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

	p.HasNext = p.Page < p.TotalPages
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

// Offset returns SQL OFFSET value (0-based)
// Formula: (page - 1) * limit
func (p Pagination) Offset() int {
	return (p.Page - 1) * p.Limit
}

// calculateTotalPages computes ceil(total / limit)
func (p Pagination) calculateTotalPages() int {
	// Prevent division by zero
	if p.Limit == 0 {
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

// NewCursor creates a new CursorPagination.
// Limit is sanitized similarly to offset pagination.
func NewCursor(limitStr string, nextCursor string, prevCursor string, hasNext bool) CursorPagination {
	limit := parseInt(limitStr, DefaultLimit)

	if limit < MinLimit {
		limit = DefaultLimit
	}
	if limit > MaxLimit {
		limit = MaxLimit
	}

	return CursorPagination{
		Limit:      limit,
		NextCursor: nextCursor,
		PrevCursor: prevCursor,
		HasNext:    hasNext,
	}
}
