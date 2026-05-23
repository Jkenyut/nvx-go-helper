package sliceutil_test

import (
	"fmt"
	"reflect"
	"strconv"
	"testing"

	"github.com/Jkenyut/nvx-go-helper/sliceutil"
)

func TestChunk(t *testing.T) {
	tests := []struct {
		name      string
		slice     []int
		size      int
		want      [][]int
		wantPanic bool
	}{
		{
			name:  "Evenly divisible",
			slice: []int{1, 2, 3, 4},
			size:  2,
			want:  [][]int{{1, 2}, {3, 4}},
		},
		{
			name:  "Not evenly divisible",
			slice: []int{1, 2, 3, 4, 5},
			size:  2,
			want:  [][]int{{1, 2}, {3, 4}, {5}},
		},
		{
			name:  "Size larger than slice",
			slice: []int{1, 2, 3},
			size:  5,
			want:  [][]int{{1, 2, 3}},
		},
		{
			name:  "Empty slice",
			slice: []int{},
			size:  2,
			want:  [][]int{},
		},
		{
			name:  "Nil slice",
			slice: nil,
			size:  2,
			want:  [][]int{},
		},
		{
			name:      "Zero size panic",
			slice:     []int{1, 2},
			size:      0,
			wantPanic: true,
		},
		{
			name:      "Negative size panic",
			slice:     []int{1, 2},
			size:      -1,
			wantPanic: true,
		},
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

			if got := sliceutil.Chunk(tt.slice, tt.size); !reflect.DeepEqual(got, tt.want) {
				t.Errorf("Chunk() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestMap(t *testing.T) {
	t.Run("Int to String", func(t *testing.T) {
		input := []int{1, 2, 3}
		want := []string{"1", "2", "3"}
		got := sliceutil.Map(input, func(v int) string {
			return strconv.Itoa(v)
		})
		if !reflect.DeepEqual(got, want) {
			t.Errorf("Map() = %v, want %v", got, want)
		}
	})

	t.Run("String to Int", func(t *testing.T) {
		input := []string{"1", "2", "3"}
		want := []int{1, 2, 3}
		got := sliceutil.Map(input, func(v string) int {
			i, _ := strconv.Atoi(v)
			return i
		})
		if !reflect.DeepEqual(got, want) {
			t.Errorf("Map() = %v, want %v", got, want)
		}
	})

	t.Run("Empty slice", func(t *testing.T) {
		input := []int{}
		want := []string{}
		got := sliceutil.Map(input, func(v int) string {
			return strconv.Itoa(v)
		})
		if !reflect.DeepEqual(got, want) {
			t.Errorf("Map() = %v, want %v", got, want)
		}
	})

	t.Run("Nil slice", func(t *testing.T) {
		var input []int
		want := []string{}
		got := sliceutil.Map(input, func(v int) string {
			return strconv.Itoa(v)
		})
		if !reflect.DeepEqual(got, want) {
			t.Errorf("Map() = %v, want %v", got, want)
		}
	})
}

func TestFilter(t *testing.T) {
	tests := []struct {
		name  string
		slice []int
		f     func(int) bool
		want  []int
	}{
		{
			name:  "Filter evens",
			slice: []int{1, 2, 3, 4, 5, 6},
			f:     func(v int) bool { return v%2 == 0 },
			want:  []int{2, 4, 6},
		},
		{
			name:  "Filter all",
			slice: []int{1, 3, 5},
			f:     func(v int) bool { return v%2 == 0 },
			want:  []int{}, // Empty slice, not nil
		},
		{
			name:  "Filter none",
			slice: []int{2, 4, 6},
			f:     func(v int) bool { return v%2 == 0 },
			want:  []int{2, 4, 6},
		},
		{
			name:  "Empty slice",
			slice: []int{},
			f:     func(v int) bool { return v > 0 },
			want:  []int{},
		},
		{
			name:  "Nil slice",
			slice: nil,
			f:     func(v int) bool { return v > 0 },
			want:  []int{},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := sliceutil.Filter(tt.slice, tt.f); !reflect.DeepEqual(got, tt.want) {
				t.Errorf("Filter() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestUnique(t *testing.T) {
	t.Run("Ints", func(t *testing.T) {
		input := []int{1, 2, 2, 3, 1, 4, 2}
		want := []int{1, 2, 3, 4}
		got := sliceutil.Unique(input)
		if !reflect.DeepEqual(got, want) {
			t.Errorf("Unique() = %v, want %v", got, want)
		}
	})

	t.Run("Strings", func(t *testing.T) {
		input := []string{"apple", "banana", "apple", "orange", "banana"}
		want := []string{"apple", "banana", "orange"}
		got := sliceutil.Unique(input)
		if !reflect.DeepEqual(got, want) {
			t.Errorf("Unique() = %v, want %v", got, want)
		}
	})

	t.Run("Structs", func(t *testing.T) {
		type user struct {
			ID int
		}
		input := []user{{1}, {2}, {1}, {3}}
		want := []user{{1}, {2}, {3}}
		got := sliceutil.Unique(input)
		if !reflect.DeepEqual(got, want) {
			t.Errorf("Unique() = %v, want %v", got, want)
		}
	})

	t.Run("All unique", func(t *testing.T) {
		input := []int{1, 2, 3}
		want := []int{1, 2, 3}
		got := sliceutil.Unique(input)
		if !reflect.DeepEqual(got, want) {
			t.Errorf("Unique() = %v, want %v", got, want)
		}
	})

	t.Run("Empty slice", func(t *testing.T) {
		input := []int{}
		want := []int{}
		got := sliceutil.Unique(input)
		if !reflect.DeepEqual(got, want) {
			t.Errorf("Unique() = %v, want %v", got, want)
		}
	})

	t.Run("Nil slice", func(t *testing.T) {
		var input []int
		want := []int{}
		got := sliceutil.Unique(input)
		if !reflect.DeepEqual(got, want) {
			t.Errorf("Unique() = %v, want %v", got, want)
		}
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

func TestMergeSlices(t *testing.T) {
	t.Run("Merge multiple integer slices", func(t *testing.T) {
		s1 := []int{1, 2}
		s2 := []int{3, 4}
		s3 := []int{5}
		got := sliceutil.MergeSlices(s1, s2, s3)
		want := []int{1, 2, 3, 4, 5}
		if !reflect.DeepEqual(got, want) {
			t.Errorf("MergeSlices() = %v, want %v", got, want)
		}
	})

	t.Run("Merge empty and nil slices", func(t *testing.T) {
		var s1 []int
		s2 := []int{}
		got := sliceutil.MergeSlices(s1, s2)
		want := []int{}
		if !reflect.DeepEqual(got, want) {
			t.Errorf("MergeSlices() = %v, want %v", got, want)
		}
	})

	t.Run("Merge strings", func(t *testing.T) {
		s1 := []string{"a"}
		s2 := []string{"b", "c"}
		got := sliceutil.MergeSlices(s1, s2)
		want := []string{"a", "b", "c"}
		if !reflect.DeepEqual(got, want) {
			t.Errorf("MergeSlices() = %v, want %v", got, want)
		}
	})
}
