// Package sliceutil provides generic, production-ready utility functions
// for common slice and array operations.
// It leverages Go 1.18+ generics to provide type-safe manipulation without reflection.
// Zero external dependencies.
package sliceutil

// Chunk splits a single slice into multiple smaller slices, each of a specified maximum size.
// If the input slice is empty, it returns an empty slice of slices.
// If size is <= 0, it panics as a zero or negative chunk size is invalid.
//
// Example:
//
//	Chunk([]int{1, 2, 3, 4, 5}, 2) // returns [][]int{{1, 2}, {3, 4}, {5}}
func Chunk[T any](slice []T, size int) [][]T {
	if size <= 0 {
		panic("sliceutil: chunk size must be greater than 0")
	}

	if len(slice) == 0 {
		return [][]T{}
	}

	// Pre-calculate the exact capacity needed to avoid slice re-allocations
	capacity := (len(slice) + size - 1) / size
	chunks := make([][]T, 0, capacity)
	for i := 0; i < len(slice); i += size {
		end := min(i + size, len(slice))
		chunks = append(chunks, slice[i:end])
	}
	return chunks
}

// Map transforms a slice of type T into a slice of type U by applying the provided
// function f to each element.
// If the input slice is nil or empty, it returns an empty slice of type U.
//
// Example:
//
//	Map([]int{1, 2, 3}, func(v int) string { return strconv.Itoa(v) }) // returns []string{"1", "2", "3"}
func Map[T, U any](slice []T, f func(T) U) []U {
	if len(slice) == 0 {
		return []U{}
	}

	result := make([]U, len(slice))
	for i := range slice {
		result[i] = f(slice[i])
	}
	return result
}

// Filter returns a new slice containing only the elements of the input slice
// for which the provided condition function f returns true.
//
// Example:
//
//	Filter([]int{1, 2, 3, 4}, func(v int) bool { return v%2 == 0 }) // returns []int{2, 4}
func Filter[T any](slice []T, f func(T) bool) []T {
	if len(slice) == 0 {
		return []T{}
	}

	var result []T
	for i := range slice {
		if f(slice[i]) {
			result = append(result, slice[i])
		}
	}
	// Return empty slice instead of nil if no elements pass the filter
	if result == nil {
		return []T{}
	}
	return result
}

// Unique returns a new slice with all duplicate elements removed.
// It preserves the original order of the first occurrences of each element.
// The type T must be comparable (can be compared using == and !=).
//
// Example:
//
//	Unique([]int{1, 2, 2, 3, 1}) // returns []int{1, 2, 3}
func Unique[T comparable](slice []T) []T {
	if len(slice) == 0 {
		return []T{}
	}

	// Pre-allocate map capacity to avoid expensive map resizing
	seen := make(map[T]struct{}, len(slice))
	
	// Pre-allocate slice capacity to avoid re-allocations (tradeoff: slightly higher memory if many duplicates)
	result := make([]T, 0, len(slice))

	for i := range slice {
		if _, exists := seen[slice[i]]; !exists {
			seen[slice[i]] = struct{}{}
			result = append(result, slice[i])
		}
	}
	return result
}
