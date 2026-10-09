package pagination

import (
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"testing"
)

func TestEncodeDecodeDynamicCursor(t *testing.T) {
	t.Run("success standard delimited encoding and decoding", func(t *testing.T) {
		values := []any{"App X", 1, 105}

		encoded, err := EncodeDynamicCursor(values...)
		require.NoError(t, err)
		assert.Equal(t, "App X,1,105", encoded)

		decoded, err := DecodeDynamicCursor(encoded)
		require.NoError(t, err)

		// Type inference parses numbers as float64
		assert.Equal(t, "App X", decoded[0])
		assert.Equal(t, float64(1), decoded[1])
		assert.Equal(t, float64(105), decoded[2])
	})

	t.Run("empty values", func(t *testing.T) {
		encoded, err := EncodeDynamicCursor()
		require.NoError(t, err)
		assert.Empty(t, encoded)

		decoded, err := DecodeDynamicCursor(encoded)
		require.NoError(t, err)
		assert.Nil(t, decoded)
	})

	t.Run("single scalar value cursor", func(t *testing.T) {
		// Single ID integer
		decoded, err := DecodeDynamicCursor("105")
		require.NoError(t, err)
		require.Len(t, decoded, 1)
		assert.Equal(t, float64(105), decoded[0])

		// Single string / UUID
		decoded, err = DecodeDynamicCursor("0192c84f-user-id")
		require.NoError(t, err)
		require.Len(t, decoded, 1)
		assert.Equal(t, "0192c84f-user-id", decoded[0])
	})

	t.Run("plain JSON array cursor", func(t *testing.T) {
		jsonCursor := `["App A",42,true]`

		decoded, err := DecodeDynamicCursor(jsonCursor)
		require.NoError(t, err)
		require.Len(t, decoded, 3)
		assert.Equal(t, "App A", decoded[0])
		assert.Equal(t, float64(42), decoded[1])
		assert.Equal(t, true, decoded[2])
	})

	t.Run("delimited string with special characters and commas", func(t *testing.T) {
		values := []any{`Special, "Name"`, 99}
		encoded, err := EncodeDynamicCursor(values...)
		require.NoError(t, err)
		assert.Equal(t, `"Special, \"Name\"",99`, encoded)

		decoded, err := DecodeDynamicCursor(encoded)
		require.NoError(t, err)
		require.Len(t, decoded, 2)
		assert.Equal(t, `Special, "Name"`, decoded[0])
		assert.Equal(t, float64(99), decoded[1])
	})

	t.Run("bidirectional cursor with delimited format", func(t *testing.T) {
		type Item struct {
			Name string
			ID   int
		}
		items := []Item{
			{Name: "Item 1", ID: 10},
			{Name: "Item 2", ID: 20},
		}

		res := GenerateBidirectionalCursor(
			items,
			2,
			"next",
			"Item 0,5",
			func(it Item) []any {
				return []any{it.Name, it.ID}
			},
		)

		assert.Equal(t, "Item 1,10", res.PrevCursor)
		assert.Equal(t, "Item 2,20", res.NextCursor)
		assert.True(t, res.HasNext)
	})
}

func TestBuildDynamicKeyset(t *testing.T) {
	t.Run("single column", func(t *testing.T) {
		cols := []string{"id"}
		ops := []string{">"}
		vals := []any{100}

		sqlStr, args := BuildDynamicKeyset(cols, ops, vals)
		assert.Equal(t, "(id > ?)", sqlStr)
		assert.Equal(t, []any{100}, args)
	})

	t.Run("two columns mixed directions", func(t *testing.T) {
		cols := []string{"name", "status"}
		ops := []string{">", "<"}
		vals := []any{"App A", 1}

		sqlStr, args := BuildDynamicKeyset(cols, ops, vals)
		assert.Equal(t, "(name > ?) OR (name = ? AND status < ?)", sqlStr)
		assert.Equal(t, []any{"App A", "App A", 1}, args)
	})

	t.Run("three columns all asc", func(t *testing.T) {
		cols := []string{"name", "status", "id"}
		ops := []string{">", ">", ">"}
		vals := []any{"App A", 1, 105}

		sqlStr, args := BuildDynamicKeyset(cols, ops, vals)
		expectedSQL := "(name > ?) OR (name = ? AND status > ?) OR (name = ? AND status = ? AND id > ?)"
		assert.Equal(t, expectedSQL, sqlStr)
		assert.Equal(t, []any{"App A", "App A", 1, "App A", 1, 105}, args)
	})

	t.Run("mismatched lengths", func(t *testing.T) {
		cols := []string{"name", "status"}
		ops := []string{">"} // missing operator
		vals := []any{"App A", 1}

		sqlStr, args := BuildDynamicKeyset(cols, ops, vals)
		// Should fallback to length 1 safely based on min calculation
		assert.Equal(t, "(name > ?)", sqlStr)
		assert.Equal(t, []any{"App A"}, args)
	})
}

func TestInvertSort(t *testing.T) {
	operators := []string{"<", ">", "<=", ">=", "<"}
	orderStrs := []string{"id DESC", "name ASC", "price DESC", "rating ASC", "status"}

	newOps, newOrders := InvertSort(operators, orderStrs)

	assert.Equal(t, []string{">", "<", ">=", "<=", ">"}, newOps)
	assert.Equal(t, []string{"id ASC", "name DESC", "price ASC", "rating DESC", "status"}, newOrders)
}

func TestGenerateBidirectionalCursor(t *testing.T) {
	type mockApp struct {
		ID   int
		Name string
	}

	items := []mockApp{
		{ID: 1, Name: "A"},
		{ID: 2, Name: "B"},
		{ID: 3, Name: "C"},
	}

	extractFn := func(a mockApp) []any {
		return []any{a.Name, a.ID}
	}

	t.Run("first_page_next", func(t *testing.T) {
		res := GenerateBidirectionalCursor(items, 3, "next", "", extractFn)
		assert.Equal(t, 3, res.Limit)
		assert.Empty(t, res.PrevCursor) // First page, no prev
		assert.NotEmpty(t, res.NextCursor)
		assert.True(t, res.HasNext)
	})

	t.Run("middle_page_next", func(t *testing.T) {
		res := GenerateBidirectionalCursor(items, 3, "next", "somecursor", extractFn)
		assert.NotEmpty(t, res.PrevCursor) // Middle page, should have prev
		assert.NotEmpty(t, res.NextCursor)
		assert.True(t, res.HasNext)
	})

	t.Run("last_page_next", func(t *testing.T) {
		itemsShort := items[:2] // less than limit
		res := GenerateBidirectionalCursor(itemsShort, 3, "next", "somecursor", extractFn)
		assert.NotEmpty(t, res.PrevCursor)
		assert.Empty(t, res.NextCursor)
		assert.False(t, res.HasNext) // Not equal to limit
	})

	t.Run("middle_page_prev", func(t *testing.T) {
		res := GenerateBidirectionalCursor(items, 3, "prev", "somecursor", extractFn)
		assert.NotEmpty(t, res.PrevCursor)
		assert.NotEmpty(t, res.NextCursor)
		assert.True(t, res.HasNext)
	})

	t.Run("first_page_prev", func(t *testing.T) {
		itemsShort := items[:2] // hit the beginning (less than limit)
		res := GenerateBidirectionalCursor(itemsShort, 3, "prev", "somecursor", extractFn)
		assert.Empty(t, res.PrevCursor) // should not have prev
		assert.NotEmpty(t, res.NextCursor)
		assert.True(t, res.HasNext)
	})

	t.Run("nil_extractFn_and_empty_items", func(t *testing.T) {
		res1 := GenerateBidirectionalCursor([]mockApp{}, 10, "next", "", nil)
		assert.Equal(t, 10, res1.Limit)
		assert.Empty(t, res1.NextCursor)
		assert.Empty(t, res1.PrevCursor)
		assert.False(t, res1.HasNext)

		res2 := GenerateBidirectionalCursor(items, 10, "next", "", nil)
		assert.Equal(t, 10, res2.Limit)
		assert.Empty(t, res2.NextCursor)
		assert.Empty(t, res2.PrevCursor)
	})
}

func TestPrepareDynamicSort(t *testing.T) {
	cfg := KeysetConfig{
		AllowedColumns: map[string]string{
			"code":   "ga_code",
			"status": "ga_status",
		},
		UniqueColumn:   "ga_id",
		UniqueSortType: "DESC",
	}

	t.Run("default_sort_when_empty", func(t *testing.T) {
		cols, ops, orders := prepareDynamicSort("", "", "next", cfg)
		assert.Equal(t, []string{"ga_id"}, cols)
		assert.Equal(t, []string{"<"}, ops)
		assert.Equal(t, []string{"ga_id DESC"}, orders)
	})

	t.Run("valid_sort_with_unique_appended", func(t *testing.T) {
		cols, ops, orders := prepareDynamicSort("code", "asc", "next", cfg)
		assert.Equal(t, []string{"ga_code", "ga_id"}, cols)
		assert.Equal(t, []string{">", "<"}, ops)
		assert.Equal(t, []string{"ga_code ASC", "ga_id DESC"}, orders)
	})

	t.Run("valid_sort_prev_direction", func(t *testing.T) {
		cols, ops, orders := prepareDynamicSort("code", "asc", "prev", cfg)
		assert.Equal(t, []string{"ga_code", "ga_id"}, cols)
		assert.Equal(t, []string{"<", ">"}, ops)
		assert.Equal(t, []string{"ga_code DESC", "ga_id ASC"}, orders)
	})

	t.Run("unique_column_already_in_sort_by_deduplicated", func(t *testing.T) {
		cfgWithID := KeysetConfig{
			AllowedColumns: map[string]string{
				"code": "ga_code",
				"id":   "ga_id",
			},
			UniqueColumn:   "ga_id",
			UniqueSortType: "DESC",
		}
		cols, ops, orders := prepareDynamicSort("code, id", "asc, desc", " PREV ", cfgWithID)
		assert.Equal(t, []string{"ga_code", "ga_id"}, cols)
		assert.Equal(t, []string{"<", ">"}, ops)
		assert.Equal(t, []string{"ga_code DESC", "ga_id ASC"}, orders)
	})
}

func TestGetDirection(t *testing.T) {
	tests := []struct {
		input string
		want  string
	}{
		{"", "next"},
		{"next", "next"},
		{"NEXT", "next"},
		{" prev ", "prev"},
		{"PREV", "prev"},
		{"unknown", "next"},
	}

	for _, tt := range tests {
		req := DynamicCursorRequest{Direction: tt.input}
		assert.Equal(t, tt.want, req.GetDirection())
	}
}

func TestBuildKeysetQuery(t *testing.T) {
	cfg := KeysetConfig{
		AllowedColumns: map[string]string{
			"name":       "users.name",
			"created_at": "users.created_at",
		},
		UniqueColumn:   "users.id",
		UniqueSortType: "DESC",
	}

	t.Run("empty cursor produces order without where", func(t *testing.T) {
		req := DynamicCursorRequest{
			SortBy:    "name",
			SortType:  "asc",
			Direction: "next",
		}
		q, err := BuildKeysetQuery(req, cfg)
		require.NoError(t, err)
		assert.False(t, q.HasWhere())
		assert.Empty(t, q.Where)
		assert.Empty(t, q.Args)
		assert.Equal(t, []string{"users.name ASC", "users.id DESC"}, q.OrderStrs)
	})

	t.Run("valid cursor produces keyset where clause and args", func(t *testing.T) {
		cursor, _ := EncodeDynamicCursor("Budi", 100)
		req := DynamicCursorRequest{
			SortBy:    "name",
			SortType:  "asc",
			Direction: "next",
			Cursor:    cursor,
		}
		q, err := req.BuildKeysetQuery(cfg)
		require.NoError(t, err)
		assert.True(t, q.HasWhere())
		assert.Equal(t, "(users.name > ?) OR (users.name = ? AND users.id < ?)", q.Where)
		assert.Equal(t, []any{"Budi", "Budi", float64(100)}, q.Args)
	})
}

func TestFieldExtractorAndFinalizeCursor(t *testing.T) {
	type TestUser struct {
		ID   int
		Name string
	}
	users := []TestUser{
		{ID: 1, Name: "Alice"},
		{ID: 2, Name: "Bob"},
	}

	extractor := NewFieldExtractor(map[string]func(TestUser) any{
		"name": func(u TestUser) any { return u.Name },
	}, func(u TestUser) any { return u.ID })

	t.Run("extracts fields in order", func(t *testing.T) {
		vals := extractor.Extract(users[0], "name")
		assert.Equal(t, []any{"Alice", 1}, vals)
	})

	t.Run("FinalizeCursor reverses array on prev direction", func(t *testing.T) {
		req := DynamicCursorRequest{
			Direction: "prev",
			Limit:     2,
			Cursor:    "dummy",
		}
		finalUsers, meta := FinalizeCursor(users, req, extractor.Fn("name"))
		assert.Equal(t, "Bob", finalUsers[0].Name)
		assert.Equal(t, "Alice", finalUsers[1].Name)
		assert.NotNil(t, meta)

		resp := NewFinalizedCursorResponse(users, req, extractor.Fn("name"))
		assert.Equal(t, "Bob", resp.Items[0].Name)
		assert.NotNil(t, resp.Pagination)
	})
}

func BenchmarkEncodeDynamicCursor_Single(b *testing.B) {
	for b.Loop() {
		_, _ = EncodeDynamicCursor(105)
	}
}

func BenchmarkEncodeDynamicCursor_Composite(b *testing.B) {
	for b.Loop() {
		_, _ = EncodeDynamicCursor("Alice", 105)
	}
}

func BenchmarkDecodeDynamicCursor(b *testing.B) {
	token := "Alice,105"
	for b.Loop() {
		_, _ = DecodeDynamicCursor(token)
	}
}

func BenchmarkBuildDynamicKeyset_Single(b *testing.B) {
	cols := []string{"id"}
	ops := []string{"<"}
	vals := []any{105}
	for b.Loop() {
		_, _ = BuildDynamicKeyset(cols, ops, vals)
	}
}

func BenchmarkBuildDynamicKeyset_Composite(b *testing.B) {
	cols := []string{"name", "id"}
	ops := []string{">", "<"}
	vals := []any{"Alice", 105}
	for b.Loop() {
		_, _ = BuildDynamicKeyset(cols, ops, vals)
	}
}

