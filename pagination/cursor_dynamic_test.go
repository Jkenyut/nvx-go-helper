package pagination

import (
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"testing"
)

func TestEncodeDecodeDynamicCursor(t *testing.T) {
	t.Run("success standard encoding", func(t *testing.T) {
		values := []any{"App X", 1, 105}

		encoded, err := EncodeDynamicCursor(values...)
		require.NoError(t, err)
		assert.NotEmpty(t, encoded)

		decoded, err := DecodeDynamicCursor(encoded)
		require.NoError(t, err)

		// JSON unmarshals numbers as float64
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
	operators := []string{"<", ">", "<"}
	orderStrs := []string{"id DESC", "name ASC", "status"}

	newOps, newOrders := InvertSort(operators, orderStrs)

	assert.Equal(t, []string{">", "<", ">"}, newOps)
	assert.Equal(t, []string{"id ASC", "name DESC", "status"}, newOrders)
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
}

func TestPrepareDynamicSort(t *testing.T) {
	allowed := map[string]string{
		"code":   "ga_code",
		"status": "ga_status",
	}

	t.Run("default_sort_when_empty", func(t *testing.T) {
		res := PrepareDynamicSort(DynamicSortParams{
			SortBy:         "",
			SortType:       "",
			Direction:      "next",
			AllowedColumns: allowed,
			UniqueColumn:   "ga_id",
			UniqueSortType: "DESC",
		})
		assert.Equal(t, []string{"ga_id"}, res.Columns)
		assert.Equal(t, []string{"<"}, res.Operators)
		assert.Equal(t, []string{"ga_id DESC"}, res.OrderStrs)
	})

	t.Run("valid_sort_with_unique_appended", func(t *testing.T) {
		res := PrepareDynamicSort(DynamicSortParams{
			SortBy:         "code",
			SortType:       "asc",
			Direction:      "next",
			AllowedColumns: allowed,
			UniqueColumn:   "ga_id",
			UniqueSortType: "DESC",
		})
		assert.Equal(t, []string{"ga_code", "ga_id"}, res.Columns)
		assert.Equal(t, []string{">", "<"}, res.Operators)
		assert.Equal(t, []string{"ga_code ASC", "ga_id DESC"}, res.OrderStrs)
	})

	t.Run("valid_sort_prev_direction", func(t *testing.T) {
		res := PrepareDynamicSort(DynamicSortParams{
			SortBy:         "code",
			SortType:       "asc",
			Direction:      "prev",
			AllowedColumns: allowed,
			UniqueColumn:   "ga_id",
			UniqueSortType: "DESC",
		})
		// Inverted operators and orderstrs
		assert.Equal(t, []string{"ga_code", "ga_id"}, res.Columns)
		assert.Equal(t, []string{"<", ">"}, res.Operators)
		assert.Equal(t, []string{"ga_code DESC", "ga_id ASC"}, res.OrderStrs)
	})
}
