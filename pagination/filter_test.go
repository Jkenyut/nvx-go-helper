package pagination_test

import (
	"net/http"
	"net/url"
	"testing"

	"github.com/Jkenyut/nvx-go-helper/pagination"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestExtractFiltersAndSearch(t *testing.T) {
	allowedFilters := map[string]string{
		"status": "users.status",
		"role":   "roles.name",
		"code":   "product_code",
	}

	t.Run("Grouped and direct filters with search", func(t *testing.T) {
		reqURL, _ := url.Parse("https://api.example.com/items?search=john&filter=status:active,pending&filter=role:admin&code=ABC,XYZ&ignored=foo&sort_by=name&limit=20")
		r := &http.Request{URL: reqURL}

		filters, search := pagination.ExtractFiltersAndSearch(r, allowedFilters)

		assert.Equal(t, "john", search)
		assert.ElementsMatch(t, []string{"active", "pending"}, filters["users.status"])
		assert.Equal(t, []string{"admin"}, filters["roles.name"])
		assert.ElementsMatch(t, []string{"ABC", "XYZ"}, filters["product_code"])
		assert.NotContains(t, filters, "ignored")
		assert.NotContains(t, filters, "sort_by")
		assert.NotContains(t, filters, "limit")
	})

	t.Run("Duplicate values deduplication", func(t *testing.T) {
		reqURL, _ := url.Parse("https://api.example.com/items?filter=status:active,active&status=active,pending")
		r := &http.Request{URL: reqURL}

		filters, _ := pagination.ExtractFiltersAndSearch(r, allowedFilters)

		assert.ElementsMatch(t, []string{"active", "pending"}, filters["users.status"])
		assert.Len(t, filters["users.status"], 2)
	})

	t.Run("Nil request", func(t *testing.T) {
		filters, search := pagination.ExtractFiltersAndSearch(nil, allowedFilters)
		assert.Empty(t, filters)
		assert.Empty(t, search)
	})
}

func TestBindOffsetFilterRequest(t *testing.T) {
	allowedFilters := map[string]string{
		"status": "status",
	}

	reqURL, _ := url.Parse("https://api.example.com/items?page=3&limit=25&search=laptop&status=active")
	r := &http.Request{URL: reqURL}

	req := pagination.BindOffsetFilterRequest(r, allowedFilters, 50)

	assert.Equal(t, 3, req.Page)
	assert.Equal(t, 25, req.Limit)
	assert.Equal(t, "laptop", req.Search)
	assert.True(t, req.HasFilter("status"))
	assert.False(t, req.HasFilter("nonexistent"))
	assert.Equal(t, []string{"active"}, req.GetFilter("status"))
	assert.Equal(t, "active", req.GetFirstFilter("status"))
	assert.Empty(t, req.GetFirstFilter("nonexistent"))

	t.Run("Max limit capping", func(t *testing.T) {
		largeURL, _ := url.Parse("https://api.example.com/items?limit=500")
		rLarge := &http.Request{URL: largeURL}
		cappedReq := pagination.BindOffsetFilterRequest(rLarge, allowedFilters, 50)
		assert.Equal(t, 50, cappedReq.Limit)
	})
}

func TestBindCursorFilterRequest(t *testing.T) {
	allowedFilters := map[string]string{
		"role": "role_col",
	}

	// Create valid cursor
	encodedCursor, err := pagination.EncodeDynamicCursor("Andi", 42)
	require.NoError(t, err)

	reqURL, _ := url.Parse("https://api.example.com/items?cursor=" + url.QueryEscape(encodedCursor) + "&direction=next&limit=15&search=keyword&role=admin")
	r := &http.Request{URL: reqURL}

	req := pagination.BindCursorFilterRequest(r, allowedFilters)

	assert.Equal(t, encodedCursor, req.Cursor)
	assert.Equal(t, "next", req.Direction)
	assert.Equal(t, 15, req.Limit)
	assert.Equal(t, "keyword", req.Search)
	assert.True(t, req.HasFilter("role_col"))
	assert.Equal(t, []string{"admin"}, req.GetFilter("role_col"))
	assert.Equal(t, "admin", req.GetFirstFilter("role_col"))

	// Verify decoded cursor values
	require.Len(t, req.CursorValues, 2)
	assert.Equal(t, "Andi", req.CursorValues[0])
	assert.Equal(t, float64(42), req.CursorValues[1]) // JSON numbers decode to float64
	assert.NoError(t, req.CursorErr)

	t.Run("BindListFilterRequest alias works identically", func(t *testing.T) {
		listReq := pagination.BindListFilterRequest(r, allowedFilters)
		assert.Equal(t, req.Cursor, listReq.Cursor)
		assert.Equal(t, req.Search, listReq.Search)
	})
}

func TestBindUnifiedFilterRequest(t *testing.T) {
	allowedFilters := map[string]string{
		"status": "status",
	}

	t.Run("Detects cursor pagination", func(t *testing.T) {
		encodedCursor, _ := pagination.EncodeDynamicCursor("item_123")
		reqURL, _ := url.Parse("https://api.example.com/items?cursor=" + url.QueryEscape(encodedCursor) + "&direction=next&status=active")
		r := &http.Request{URL: reqURL}

		uReq := pagination.BindUnifiedFilterRequest(r, allowedFilters)

		assert.True(t, uReq.IsCursor)
		assert.Equal(t, encodedCursor, uReq.DynamicCursorRequest.Cursor)
		assert.Len(t, uReq.CursorValues, 1)
		assert.True(t, uReq.HasFilter("status"))
	})

	t.Run("Falls back to offset pagination", func(t *testing.T) {
		reqURL, _ := url.Parse("https://api.example.com/items?page=2&limit=20&search=test")
		r := &http.Request{URL: reqURL}

		uReq := pagination.BindUnifiedFilterRequest(r, allowedFilters)

		assert.False(t, uReq.IsCursor)
		assert.Equal(t, 2, uReq.OffsetRequest.Page)
		assert.Equal(t, 20, uReq.OffsetRequest.Limit)
		assert.Equal(t, "test", uReq.Search)
	})
}

func TestResponses(t *testing.T) {
	type Product struct {
		ID   int    `json:"id"`
		Name string `json:"name"`
	}

	products := []Product{
		{ID: 1, Name: "Item A"},
		{ID: 2, Name: "Item B"},
	}

	t.Run("NewListResponse", func(t *testing.T) {
		p := pagination.NewFromInt(1, 10, 2)
		resp := pagination.NewListResponse(products, p)

		assert.Len(t, resp.Items, 2)
		require.NotNil(t, resp.Pagination)
		assert.Equal(t, 1, resp.Pagination.Page)
		assert.Equal(t, 2, resp.Pagination.Total)
	})

	t.Run("NewCursorListResponse", func(t *testing.T) {
		cursorMeta := pagination.NewCursor("10", "next_xyz", "prev_abc", true)
		resp := pagination.NewCursorListResponse(products, &cursorMeta)

		assert.Len(t, resp.Items, 2)
		require.NotNil(t, resp.Pagination)
		assert.Equal(t, "next_xyz", resp.Pagination.NextCursor)
		assert.Equal(t, "prev_abc", resp.Pagination.PrevCursor)
		assert.True(t, resp.Pagination.HasNext)
	})
}
