package maputil

import (
	"sort"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestKeys(t *testing.T) {
	t.Run("Normal", func(t *testing.T) {
		m := map[string]int{"a": 1, "b": 2, "c": 3}
		keys := Keys(m)
		sort.Strings(keys)
		assert.Equal(t, []string{"a", "b", "c"}, keys)
	})

	t.Run("Empty", func(t *testing.T) {
		m := map[string]int{}
		keys := Keys(m)
		assert.Empty(t, keys)
	})
}

func TestValues(t *testing.T) {
	t.Run("Normal", func(t *testing.T) {
		m := map[string]int{"a": 1, "b": 2}
		values := Values(m)
		sort.Ints(values)
		assert.Equal(t, []int{1, 2}, values)
	})

	t.Run("Empty", func(t *testing.T) {
		m := map[string]int{}
		values := Values(m)
		assert.Empty(t, values)
	})
}

func TestMerge(t *testing.T) {
	t.Run("Two maps", func(t *testing.T) {
		m1 := map[string]int{"a": 1}
		m2 := map[string]int{"b": 2, "a": 3}
		result := Merge(m1, m2)
		assert.Equal(t, 3, result["a"]) // last wins
		assert.Equal(t, 2, result["b"])
	})

	t.Run("Empty maps", func(t *testing.T) {
		result := Merge[string, int]()
		assert.Empty(t, result)
	})

	t.Run("Three maps", func(t *testing.T) {
		m1 := map[string]int{"a": 1}
		m2 := map[string]int{"b": 2}
		m3 := map[string]int{"c": 3}
		result := Merge(m1, m2, m3)
		assert.Len(t, result, 3)
	})
}

func TestPick(t *testing.T) {
	m := map[string]int{"a": 1, "b": 2, "c": 3}

	t.Run("Pick existing keys", func(t *testing.T) {
		result := Pick(m, "a", "c")
		assert.Equal(t, map[string]int{"a": 1, "c": 3}, result)
	})

	t.Run("Pick non-existing key", func(t *testing.T) {
		result := Pick(m, "x", "y")
		assert.Empty(t, result)
	})

	t.Run("Pick mixed", func(t *testing.T) {
		result := Pick(m, "a", "x")
		assert.Equal(t, map[string]int{"a": 1}, result)
	})
}

func TestOmit(t *testing.T) {
	m := map[string]int{"a": 1, "b": 2, "c": 3}

	t.Run("Omit existing key", func(t *testing.T) {
		result := Omit(m, "b")
		assert.Equal(t, map[string]int{"a": 1, "c": 3}, result)
	})

	t.Run("Omit non-existing key", func(t *testing.T) {
		result := Omit(m, "x")
		assert.Len(t, result, 3)
	})

	t.Run("Omit all", func(t *testing.T) {
		result := Omit(m, "a", "b", "c")
		assert.Empty(t, result)
	})
}

func TestInvert(t *testing.T) {
	t.Run("Normal", func(t *testing.T) {
		m := map[string]int{"a": 1, "b": 2}
		result := Invert(m)
		assert.Equal(t, "a", result[1])
		assert.Equal(t, "b", result[2])
	})

	t.Run("Empty", func(t *testing.T) {
		m := map[string]int{}
		result := Invert(m)
		assert.Empty(t, result)
	})
}

func TestFromEntries(t *testing.T) {
	t.Run("Normal", func(t *testing.T) {
		entries := []Entry[string, int]{
			{Key: "a", Value: 1},
			{Key: "b", Value: 2},
		}
		result := FromEntries(entries)
		assert.Equal(t, 1, result["a"])
		assert.Equal(t, 2, result["b"])
	})

	t.Run("Empty", func(t *testing.T) {
		result := FromEntries[string, int](nil)
		assert.Empty(t, result)
	})

	t.Run("Duplicate keys", func(t *testing.T) {
		entries := []Entry[string, int]{
			{Key: "a", Value: 1},
			{Key: "a", Value: 2},
		}
		result := FromEntries(entries)
		assert.Equal(t, 2, result["a"]) // last wins
	})
}

func TestContainsKey(t *testing.T) {
	m := map[string]int{"a": 1, "b": 2}

	assert.True(t, ContainsKey(m, "a"))
	assert.False(t, ContainsKey(m, "c"))
}

func TestContainsValue(t *testing.T) {
	m := map[string]int{"a": 1, "b": 2}

	assert.True(t, ContainsValue(m, 1))
	assert.False(t, ContainsValue(m, 99))
}

func TestFilter(t *testing.T) {
	m := map[string]int{"a": 1, "b": 2, "c": 3}

	t.Run("Filter by value", func(t *testing.T) {
		result := Filter(m, func(k string, v int) bool { return v > 1 })
		assert.Len(t, result, 2)
		assert.Equal(t, 2, result["b"])
		assert.Equal(t, 3, result["c"])
	})

	t.Run("Filter none match", func(t *testing.T) {
		result := Filter(m, func(k string, v int) bool { return v > 100 })
		assert.Empty(t, result)
	})

	t.Run("Filter all match", func(t *testing.T) {
		result := Filter(m, func(k string, v int) bool { return true })
		assert.Len(t, result, 3)
	})
}
