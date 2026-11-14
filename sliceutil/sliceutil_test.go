package sliceutil_test

import (
	"fmt"
	"reflect"
	"strconv"
	"testing"

	"github.com/Jkenyut/nvx-go-helper/sliceutil"
	"github.com/stretchr/testify/assert"
)

func TestMap(t *testing.T) {
	t.Run("Int to String", func(t *testing.T) {
		got := sliceutil.Map([]int{1, 2, 3}, func(v int) string { return strconv.Itoa(v) })
		assert.Equal(t, []string{"1", "2", "3"}, got)
	})
	t.Run("Empty", func(t *testing.T) {
		got := sliceutil.Map([]int{}, func(v int) string { return strconv.Itoa(v) })
		assert.Equal(t, []string{}, got)
	})
	t.Run("Nil", func(t *testing.T) {
		got := sliceutil.Map(nil, func(v int) string { return strconv.Itoa(v) })
		assert.Equal(t, []string{}, got)
	})
}

func TestFilter(t *testing.T) {
	t.Run("Filter evens", func(t *testing.T) {
		got := sliceutil.Filter([]int{1, 2, 3, 4, 5, 6}, func(v int) bool { return v%2 == 0 })
		assert.Equal(t, []int{2, 4, 6}, got)
	})
	t.Run("None match", func(t *testing.T) {
		got := sliceutil.Filter([]int{1, 3, 5}, func(v int) bool { return v%2 == 0 })
		assert.Equal(t, []int{}, got)
	})
	t.Run("All match", func(t *testing.T) {
		got := sliceutil.Filter([]int{2, 4, 6}, func(v int) bool { return v%2 == 0 })
		assert.Equal(t, []int{2, 4, 6}, got)
	})
	t.Run("Empty", func(t *testing.T) {
		got := sliceutil.Filter([]int{}, func(v int) bool { return v > 0 })
		assert.Equal(t, []int{}, got)
	})
	t.Run("Nil", func(t *testing.T) {
		got := sliceutil.Filter(nil, func(v int) bool { return v > 0 })
		assert.Equal(t, []int{}, got)
	})
}

func TestUnique(t *testing.T) {
	t.Run("Ints", func(t *testing.T) {
		assert.Equal(t, []int{1, 2, 3, 4}, sliceutil.Unique([]int{1, 2, 2, 3, 1, 4, 2}))
	})
	t.Run("Strings", func(t *testing.T) {
		assert.Equal(t, []string{"apple", "banana", "orange"}, sliceutil.Unique([]string{"apple", "banana", "apple", "orange", "banana"}))
	})
	t.Run("Empty", func(t *testing.T) {
		assert.Equal(t, []int{}, sliceutil.Unique([]int{}))
	})
	t.Run("Nil", func(t *testing.T) {
		assert.Equal(t, []int{}, sliceutil.Unique[int](nil))
	})
	t.Run("All same", func(t *testing.T) {
		assert.Equal(t, []int{1}, sliceutil.Unique([]int{1, 1, 1}))
	})
}

func TestFind(t *testing.T) {
	t.Run("Found", func(t *testing.T) {
		val, ok := sliceutil.Find([]int{1, 2, 3, 4}, func(v int) bool { return v > 2 })
		assert.True(t, ok)
		assert.Equal(t, 3, val)
	})
	t.Run("Not found", func(t *testing.T) {
		val, ok := sliceutil.Find([]int{1, 2, 3}, func(v int) bool { return v > 10 })
		assert.False(t, ok)
		assert.Equal(t, 0, val)
	})
	t.Run("Empty", func(t *testing.T) {
		_, ok := sliceutil.Find([]int{}, func(v int) bool { return true })
		assert.False(t, ok)
	})
}

func TestToMap(t *testing.T) {
	type User struct {
		ID   int
		Name string
	}
	users := []User{{1, "Alice"}, {2, "Bob"}, {3, "Charlie"}}
	m := sliceutil.ToMap(users, func(u User) int { return u.ID })
	assert.Equal(t, "Alice", m[1].Name)
	assert.Equal(t, "Bob", m[2].Name)
	assert.Equal(t, "Charlie", m[3].Name)

	// Empty
	empty := sliceutil.ToMap([]User{}, func(u User) int { return u.ID })
	assert.Empty(t, empty)
}

func TestGroupBy(t *testing.T) {
	type Item struct {
		Category string
		Name     string
	}
	items := []Item{
		{"fruit", "apple"},
		{"veggie", "carrot"},
		{"fruit", "banana"},
	}
	groups := sliceutil.GroupBy(items, func(i Item) string { return i.Category })
	assert.Len(t, groups["fruit"], 2)
	assert.Len(t, groups["veggie"], 1)

	// Empty
	emptyGroups := sliceutil.GroupBy([]Item{}, func(i Item) string { return i.Category })
	assert.Empty(t, emptyGroups)
}

func TestIntersection(t *testing.T) {
	t.Run("Normal", func(t *testing.T) {
		result := sliceutil.Intersection([]int{1, 2, 3, 4}, []int{2, 4, 5})
		assert.Equal(t, []int{2, 4}, result)
	})
	t.Run("No overlap", func(t *testing.T) {
		result := sliceutil.Intersection([]int{1, 2}, []int{3, 4})
		assert.Equal(t, []int{}, result)
	})
	t.Run("Empty a", func(t *testing.T) {
		result := sliceutil.Intersection([]int{}, []int{1, 2})
		assert.Equal(t, []int{}, result)
	})
	t.Run("Empty b", func(t *testing.T) {
		result := sliceutil.Intersection([]int{1, 2}, []int{})
		assert.Equal(t, []int{}, result)
	})
}

func TestDifference(t *testing.T) {
	t.Run("Normal", func(t *testing.T) {
		result := sliceutil.Difference([]int{1, 2, 3, 4}, []int{2, 4})
		assert.Equal(t, []int{1, 3}, result)
	})
	t.Run("No difference", func(t *testing.T) {
		result := sliceutil.Difference([]int{1, 2}, []int{1, 2, 3})
		assert.Equal(t, []int{}, result)
	})
	t.Run("Empty a", func(t *testing.T) {
		result := sliceutil.Difference([]int{}, []int{1, 2})
		assert.Equal(t, []int{}, result)
	})
	t.Run("Empty b returns copy", func(t *testing.T) {
		result := sliceutil.Difference([]int{1, 2, 3}, []int{})
		assert.Equal(t, []int{1, 2, 3}, result)
	})
}

func TestFlatten(t *testing.T) {
	t.Run("Normal", func(t *testing.T) {
		result := sliceutil.Flatten([][]int{{1, 2}, {3}, {4, 5}})
		assert.Equal(t, []int{1, 2, 3, 4, 5}, result)
	})
	t.Run("Empty", func(t *testing.T) {
		assert.Equal(t, []int{}, sliceutil.Flatten([][]int{}))
	})
}

func TestReduce(t *testing.T) {
	t.Run("Sum", func(t *testing.T) {
		result := sliceutil.Reduce([]int{1, 2, 3, 4}, 0, func(acc, v int) int { return acc + v })
		assert.Equal(t, 10, result)
	})
	t.Run("String concat", func(t *testing.T) {
		result := sliceutil.Reduce([]string{"a", "b", "c"}, "", func(acc, v string) string { return acc + v })
		assert.Equal(t, "abc", result)
	})
	t.Run("Empty", func(t *testing.T) {
		result := sliceutil.Reduce([]int{}, 42, func(acc, v int) int { return acc + v })
		assert.Equal(t, 42, result)
	})
	t.Run("Different types", func(t *testing.T) {
		result := sliceutil.Reduce([]int{1, 2, 3}, "", func(acc string, v int) string {
			return acc + strconv.Itoa(v)
		})
		assert.Equal(t, "123", result)
	})
}

func TestFlatMap(t *testing.T) {
	t.Run("Normal", func(t *testing.T) {
		result := sliceutil.FlatMap([]string{"ab", "cd"}, func(s string) []byte { return []byte(s) })
		assert.Equal(t, []byte{'a', 'b', 'c', 'd'}, result)
	})
	t.Run("Empty input", func(t *testing.T) {
		result := sliceutil.FlatMap([]string{}, func(s string) []int { return []int{1} })
		assert.Equal(t, []int{}, result)
	})
	t.Run("Empty output per element", func(t *testing.T) {
		result := sliceutil.FlatMap([]int{1, 2, 3}, func(v int) []string { return nil })
		assert.Equal(t, []string{}, result)
	})
}

func TestLast(t *testing.T) {
	t.Run("Normal", func(t *testing.T) {
		val, ok := sliceutil.Last([]int{1, 2, 3})
		assert.True(t, ok)
		assert.Equal(t, 3, val)
	})
	t.Run("Single", func(t *testing.T) {
		val, ok := sliceutil.Last([]int{42})
		assert.True(t, ok)
		assert.Equal(t, 42, val)
	})
	t.Run("Empty", func(t *testing.T) {
		val, ok := sliceutil.Last([]int{})
		assert.False(t, ok)
		assert.Equal(t, 0, val)
	})
}

func TestCompact(t *testing.T) {
	t.Run("Strings", func(t *testing.T) {
		result := sliceutil.Compact([]string{"a", "", "b", "", "c"})
		assert.Equal(t, []string{"a", "b", "c"}, result)
	})
	t.Run("Ints", func(t *testing.T) {
		result := sliceutil.Compact([]int{0, 1, 2, 0, 3})
		assert.Equal(t, []int{1, 2, 3}, result)
	})
	t.Run("All zeros", func(t *testing.T) {
		result := sliceutil.Compact([]int{0, 0, 0})
		assert.Equal(t, []int{}, result)
	})
	t.Run("No zeros", func(t *testing.T) {
		result := sliceutil.Compact([]int{1, 2, 3})
		assert.Equal(t, []int{1, 2, 3}, result)
	})
	t.Run("Empty", func(t *testing.T) {
		result := sliceutil.Compact([]int{})
		assert.Equal(t, []int{}, result)
	})
}

// Example uses for godoc
func ExampleMap() {
	nums := []int{1, 2, 3}
	strs := sliceutil.Map(nums, func(v int) string {
		return fmt.Sprintf("ID-%d", v)
	})
	fmt.Println(strs)
	// Output: [ID-1 ID-2 ID-3]
}

func ExampleFilter() {
	nums := []int{1, 2, 3, 4, 5, 6}
	evens := sliceutil.Filter(nums, func(v int) bool {
		return v%2 == 0
	})
	fmt.Println(evens)
	// Output: [2 4 6]
}

func ExampleUnique() {
	fruits := []string{"apple", "banana", "apple", "orange"}
	uniqueFruits := sliceutil.Unique(fruits)
	fmt.Println(uniqueFruits)
	// Output: [apple banana orange]
}

// Keep reflect import for backward compat with ExampleChunk etc.
var _ = reflect.DeepEqual
