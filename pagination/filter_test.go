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
		assert.True(t, uReq.HasFilter("status"))
		assert.Equal(t, []string{"active"}, uReq.GetFilter("status"))
		assert.Equal(t, "active", uReq.GetFirstFilter("status"))
		assert.Empty(t, uReq.GetFirstFilter("nonexistent"))
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

	t.Run("Request NormalizedFilterValue and NormalizedCursorValues", func(t *testing.T) {
		detector := pagination.NewTypeDetector().
			WithIntegerSuffixes("page_num").
			WithUUIDSuffixes("id")

		// OffsetFilterRequest
		offReq := pagination.OffsetFilterRequest{
			FilterBase: pagination.FilterBase{
				Filters: map[string][]string{
					"page_num": {"5"},
				},
			},
		}
		val, err := offReq.NormalizedFilterValue("page_num", detector)
		require.NoError(t, err)
		assert.Equal(t, 5, val)

		// CursorFilterRequest
		token, _ := pagination.EncodeDynamicCursor("99", "admin")
		curReq := pagination.CursorFilterRequest{
			FilterBase: pagination.FilterBase{
				Filters: map[string][]string{
					"page_num": {"10"},
				},
			},
			CursorValues: []any{"99", "admin"},
		}
		cVal, err := curReq.NormalizedFilterValue("page_num", detector)
		require.NoError(t, err)
		assert.Equal(t, 10, cVal)

		cCursorVals, err := curReq.NormalizedCursorValues([]string{"page_num", "role"}, detector)
		require.NoError(t, err)
		assert.Equal(t, int64(99), cCursorVals[0])
		assert.Equal(t, "admin", cCursorVals[1])

		// UnifiedFilterRequest
		uReq := pagination.UnifiedFilterRequest{
			FilterBase: pagination.FilterBase{
				Filters: map[string][]string{
					"page_num": {"15"},
				},
			},
			CursorValues: []any{"100"},
		}
		uVal, err := uReq.NormalizedFilterValue("page_num", detector)
		require.NoError(t, err)
		assert.Equal(t, 15, uVal)

		uCursorVals, err := uReq.NormalizedCursorValues([]string{"page_num"}, detector)
		require.NoError(t, err)
		assert.Equal(t, int64(100), uCursorVals[0])
		_ = token

		// Case-insensitive filter lookup
		assert.True(t, curReq.HasFilter("PAGE_NUM"))
		assert.Equal(t, "10", curReq.GetFirstFilter("PAGE_NUM"))
		assert.Equal(t, []string{"10"}, curReq.GetFilter("PAGE_NUM"))
		valCase, err := curReq.NormalizedFilterValue("PAGE_NUM", detector)
		require.NoError(t, err)
		assert.Equal(t, 10, valCase)

		// Interface compliance
		var _ pagination.FilterRequest = offReq
		var _ pagination.FilterRequest = curReq
		var _ pagination.FilterRequest = uReq
		var _ pagination.CursorFilterProvider = curReq
		var _ pagination.CursorFilterProvider = uReq
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
		cursorMeta := pagination.NewCursorFromInt(10, "next_xyz", "prev_abc", true)
		resp := pagination.NewCursorListResponse(products, &cursorMeta)

		assert.Len(t, resp.Items, 2)
		require.NotNil(t, resp.Pagination)
		assert.Equal(t, "next_xyz", resp.Pagination.NextCursor)
		assert.Equal(t, "prev_abc", resp.Pagination.PrevCursor)
		assert.True(t, resp.Pagination.HasNext)
	})
}

func TestUnifiedAndOffsetRequestHelpers(t *testing.T) {
	allowedFilters := map[string]string{
		"status": "users.status",
	}

	t.Run("Detects mode=cursor and pagination=cursor", func(t *testing.T) {
		reqURL1, _ := url.Parse("https://api.example.com/items?mode=cursor&limit=10")
		r1 := &http.Request{URL: reqURL1}
		uReq1 := pagination.BindUnifiedFilterRequest(r1, allowedFilters)
		assert.True(t, uReq1.IsCursor)

		reqURL2, _ := url.Parse("https://api.example.com/items?pagination=cursor&limit=10")
		r2 := &http.Request{URL: reqURL2}
		uReq2 := pagination.BindUnifiedFilterRequest(r2, allowedFilters)
		assert.True(t, uReq2.IsCursor)
	})

	t.Run("Unified conversions and getters", func(t *testing.T) {
		reqURL, _ := url.Parse("https://api.example.com/items?cursor=abc&limit=15&sort_by=name&sort_type=desc&status=active")
		r := &http.Request{URL: reqURL}
		uReq := pagination.BindUnifiedFilterRequest(r, allowedFilters)

		assert.Equal(t, 15, uReq.GetLimit())
		assert.Equal(t, "name", uReq.GetSortBy())
		assert.Equal(t, "desc", uReq.GetSortType())

		// Convert to CursorFilterRequest
		cReq := uReq.ToCursorFilterRequest()
		assert.Equal(t, "abc", cReq.Cursor)
		assert.Equal(t, 15, cReq.Limit)
		assert.True(t, cReq.HasFilter("users.status"))

		// Convert to OffsetFilterRequest
		oReq := uReq.ToOffsetFilterRequest()
		assert.Equal(t, 15, oReq.Limit)
		assert.True(t, oReq.HasFilter("users.status"))

		// Offset Pagination helper
		pageData := uReq.Pagination(50)
		assert.Equal(t, 50, pageData.Total)
		assert.Equal(t, 15, pageData.Limit)
	})

	t.Run("BuildKeysetQuery on CursorFilterRequest and UnifiedFilterRequest", func(t *testing.T) {
		token, _ := pagination.EncodeDynamicCursor("Budi", 100)
		reqURL, _ := url.Parse("https://api.example.com/items?cursor=" + url.QueryEscape(token) + "&sort_by=name&sort_type=asc")
		r := &http.Request{URL: reqURL}
		uReq := pagination.BindUnifiedFilterRequest(r, allowedFilters)

		cfg := pagination.KeysetConfig{
			AllowedColumns: map[string]string{
				"name": "users.name",
			},
			UniqueColumn:   "users.id",
			UniqueSortType: "DESC",
		}

		keyset, err := uReq.BuildKeysetQuery(cfg)
		require.NoError(t, err)
		assert.True(t, keyset.HasWhere())
		assert.Equal(t, "(users.name > ?) OR (users.name = ? AND users.id < ?)", keyset.Where)
		assert.Equal(t, []string{"users.name ASC", "users.id DESC"}, keyset.OrderStrs)

		cReq := uReq.ToCursorFilterRequest()
		keyset2, err := cReq.BuildKeysetQuery(cfg)
		require.NoError(t, err)
		assert.Equal(t, keyset.Where, keyset2.Where)
	})

	t.Run("OffsetRequest Offset and Pagination methods", func(t *testing.T) {
		offReq := pagination.OffsetRequest{
			Page:  3,
			Limit: 10,
		}
		assert.Equal(t, 20, offReq.Offset())
		pageData := offReq.Pagination(100)
		assert.Equal(t, 3, pageData.Page)
		assert.Equal(t, 10, pageData.Limit)
		assert.Equal(t, 100, pageData.Total)
	})
}
