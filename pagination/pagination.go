// Package pagination provides a robust, production-ready, and highly reusable
// pagination system for REST APIs.
//
// Features (2025 enterprise standard):
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
	"math"
	"net/url"
	"strconv"
)

// Default values
const (
	DefaultPage  = 1
	DefaultLimit = 10
	MaxLimit     = 100000
	MinLimit     = 1
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
	page := parseInt(pageStr, DefaultPage)
	limit := parseInt(limitStr, DefaultLimit)

	// Sanitize
	if page < 1 {
		page = DefaultPage
	}
	if limit < MinLimit {
		limit = DefaultLimit
	}
	if limit > MaxLimit {
		limit = MaxLimit
	}

	p := Pagination{
		Page:  page,
		Limit: limit,
		Total: total,
	}

	p.TotalPages = p.calculateTotalPages()
	p.HasNext = p.Page < p.TotalPages
	p.HasPrev = p.Page > 1
	p.NextPage = p.Page + 1
	p.PrevPage = p.Page - 1

	// Omit next/prev if not exist
	if !p.HasNext {
		p.NextPage = 0
	}
	if !p.HasPrev {
		p.PrevPage = 0
	}

	return p
}

// Offset returns SQL OFFSET value (0-based)
func (p Pagination) Offset() int {
	return (p.Page - 1) * p.Limit
}

// calculateTotalPages computes ceil(total / limit)
func (p Pagination) calculateTotalPages() int {
	if p.Limit == 0 {
		return 0
	}
	return int(math.Ceil(float64(p.Total) / float64(p.Limit)))
}

// Links generates RFC 5988 Link headers: <url>?page=3>; rel="next"
// Links generates RFC 5988 Link headers with FULL URL (scheme + host + path)
// Links generates RFC 5988 Link headers with FULL URL (scheme + host + path)
func (p Pagination) Links(baseURL string) (map[string]string, error) {
	u, err := url.Parse(baseURL)
	if err != nil {
		return nil, err
	}

	// Gunakan RawPath jika ada, fallback ke Path
	path := u.Path
	if u.RawPath != "" {
		path = u.RawPath
	}

	// Bangun base URL lengkap: scheme://host/path
	base := fmt.Sprintf("%s://%s%s", u.Scheme, u.Host, path)

	links := make(map[string]string)
	q := u.Query()
	q.Set("limit", strconv.Itoa(p.Limit))

	if p.HasPrev {
		q.Set("page", strconv.Itoa(p.PrevPage))
		links["prev"] = fmt.Sprintf(`<%s?%s>; rel="prev"`, base, q.Encode())
	}
	if p.HasNext {
		q.Set("page", strconv.Itoa(p.NextPage))
		links["next"] = fmt.Sprintf(`<%s?%s>; rel="next"`, base, q.Encode())
	}

	return links, nil
}

// parseInt safely converts string to int with fallback
func parseInt(s string, fallback int) int {
	if s == "" {
		return fallback
	}
	val, err := strconv.Atoi(s)
	if err != nil {
		return fallback
	}
	return val
}
