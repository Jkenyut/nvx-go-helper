package sliceutil_test

import (
	"fmt"
	"reflect"
	"sort"
	"strconv"
	"testing"

	"github.com/Jkenyut/nvx-go-helper/sliceutil"
	"github.com/stretchr/testify/assert"
)

func TestChunk(t *testing.T) {
	tests := []struct {
		name      string
		slice     []int
		size      int
		want      [][]int
		wantPanic bool
	}{
		{"Evenly divisible", []int{1, 2, 3, 4}, 2, [][]int{{1, 2}, {3, 4}}, false},
		{"Not evenly divisible", []int{1, 2, 3, 4, 5}, 2, [][]int{{1, 2}, {3, 4}, {5}}, false},
		{"Size larger than slice", []int{1, 2, 3}, 5, [][]int{{1, 2, 3}}, false},
		{"Empty slice", []int{}, 2, [][]int{}, false},
		{"Nil slice", nil, 2, [][]int{}, false},
		{"Zero size panic", []int{1, 2}, 0, nil, true},
		{"Negative size panic", []int{1, 2}, -1, nil, true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.wantPanic {
				defer func() {
					if r := recover(); r == nil {
						t.Errorf("Chunk() did not panic for invalid size %d", tt.size)
					}
				}()
				sliceutil.Chunk(tt.slice, tt.size)
				return
			}
			assert.Equal(t, tt.want, sliceutil.Chunk(tt.slice, tt.size))
		})
	}
}

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

func TestContains(t *testing.T) {
	assert.True(t, sliceutil.Contains([]int{1, 2, 3}, 2))
	assert.False(t, sliceutil.Contains([]int{1, 2, 3}, 4))
	assert.False(t, sliceutil.Contains([]int{}, 1))
	assert.False(t, sliceutil.Contains[int](nil, 1))
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

func TestReverse(t *testing.T) {
	t.Run("Normal", func(t *testing.T) {
		s := []int{1, 2, 3, 4, 5}
		result := sliceutil.Reverse(s)
		assert.Equal(t, []int{5, 4, 3, 2, 1}, result)
	})
	t.Run("Single element", func(t *testing.T) {
		s := []int{1}
		assert.Equal(t, []int{1}, sliceutil.Reverse(s))
	})
	t.Run("Empty", func(t *testing.T) {
		s := []int{}
		assert.Equal(t, []int{}, sliceutil.Reverse(s))
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

func TestMapKeys(t *testing.T) {
	m := map[string]int{"a": 1, "b": 2, "c": 3}
	keys := sliceutil.MapKeys(m)
	sort.Strings(keys)
	assert.Equal(t, []string{"a", "b", "c"}, keys)
}

func TestMapValues(t *testing.T) {
	m := map[string]int{"a": 1, "b": 2}
	values := sliceutil.MapValues(m)
	sort.Ints(values)
	assert.Equal(t, []int{1, 2}, values)
}

func TestMapMerge(t *testing.T) {
	m1 := map[string]int{"a": 1}
	m2 := map[string]int{"b": 2, "a": 3}
	result := sliceutil.MapMerge(m1, m2)
	assert.Equal(t, 3, result["a"])
	assert.Equal(t, 2, result["b"])
}

func TestMergeSlices(t *testing.T) {
	t.Run("Normal", func(t *testing.T) {
		got := sliceutil.MergeSlices([]int{1, 2}, []int{3, 4}, []int{5})
		assert.Equal(t, []int{1, 2, 3, 4, 5}, got)
	})
	t.Run("Empty", func(t *testing.T) {
		got := sliceutil.MergeSlices[int]()
		assert.Equal(t, []int{}, got)
	})
	t.Run("Nil slices", func(t *testing.T) {
		var s1 []int
		s2 := []int{}
		got := sliceutil.MergeSlices(s1, s2)
		assert.Equal(t, []int{}, got)
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

func TestIndexOf(t *testing.T) {
	assert.Equal(t, 1, sliceutil.IndexOf([]string{"a", "b", "c"}, "b"))
	assert.Equal(t, -1, sliceutil.IndexOf([]string{"a", "b", "c"}, "x"))
	assert.Equal(t, 0, sliceutil.IndexOf([]int{1, 2, 3}, 1))
	assert.Equal(t, -1, sliceutil.IndexOf([]int{}, 1))
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
func ExampleChunk() {
	nums := []int{1, 2, 3, 4, 5}
	chunks := sliceutil.Chunk(nums, 2)
	fmt.Println(chunks)
	// Output: [[1 2] [3 4] [5]]
}

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
