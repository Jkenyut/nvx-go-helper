package pagination

import (
	"testing"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestEncodeDecodeDynamicCursor(t *testing.T) {
	t.Run("success standard encoding", func(t *testing.T) {
		values := []interface{}{"App X", 1, 105}
		
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
		vals := []interface{}{100}
		
		sqlStr, args := BuildDynamicKeyset(cols, ops, vals)
		assert.Equal(t, "(id > ?)", sqlStr)
		assert.Equal(t, []interface{}{100}, args)
	})
	
	t.Run("two columns mixed directions", func(t *testing.T) {
		cols := []string{"name", "status"}
		ops := []string{">", "<"}
		vals := []interface{}{"App A", 1}
		
		sqlStr, args := BuildDynamicKeyset(cols, ops, vals)
		assert.Equal(t, "(name > ?) OR (name = ? AND status < ?)", sqlStr)
		assert.Equal(t, []interface{}{"App A", "App A", 1}, args)
	})
	
	t.Run("three columns all asc", func(t *testing.T) {
		cols := []string{"name", "status", "id"}
		ops := []string{">", ">", ">"}
		vals := []interface{}{"App A", 1, 105}
		
		sqlStr, args := BuildDynamicKeyset(cols, ops, vals)
		expectedSQL := "(name > ?) OR (name = ? AND status > ?) OR (name = ? AND status = ? AND id > ?)"
		assert.Equal(t, expectedSQL, sqlStr)
		assert.Equal(t, []interface{}{"App A", "App A", 1, "App A", 1, 105}, args)
	})
	
	t.Run("mismatched lengths", func(t *testing.T) {
		cols := []string{"name", "status"}
		ops := []string{">"} // missing operator
		vals := []interface{}{"App A", 1}
		
		sqlStr, args := BuildDynamicKeyset(cols, ops, vals)
		// Should fallback to length 1 safely based on min calculation
		assert.Equal(t, "(name > ?)", sqlStr)
		assert.Equal(t, []interface{}{"App A"}, args)
	})
}
