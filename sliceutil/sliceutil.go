// Package sliceutil provides generic, production-ready utility functions
// for common slice and array operations.
// It leverages Go 1.18+ generics to provide type-safe manipulation without reflection.
// Zero external dependencies.
package sliceutil

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

// Find returns the first element that satisfies the condition f, and true.
// If no element satisfies the condition, it returns the zero value of T and false.
func Find[T any](slice []T, f func(T) bool) (T, bool) {
	for i := range slice {
		if f(slice[i]) {
			return slice[i], true
		}
	}
	var zero T
	return zero, false
}

// ToMap converts a slice into a map, using the keyFn to generate keys for each element.
// If multiple elements resolve to the same key, the last one wins.
func ToMap[K comparable, V any](slice []V, keyFn func(V) K) map[K]V {
	if len(slice) == 0 {
		return map[K]V{}
	}
	result := make(map[K]V, len(slice))
	for i := range slice {
		result[keyFn(slice[i])] = slice[i]
	}
	return result
}

// GroupBy groups the elements of the slice into a map, using the keyFn to determine the group key.
func GroupBy[K comparable, V any](slice []V, keyFn func(V) K) map[K][]V {
	if len(slice) == 0 {
		return map[K][]V{}
	}
	result := make(map[K][]V)
	for i := range slice {
		key := keyFn(slice[i])
		result[key] = append(result[key], slice[i])
	}
	return result
}

// Intersection returns a new slice containing elements that are present in both slice a and slice b.
func Intersection[T comparable](a, b []T) []T {
	if len(a) == 0 || len(b) == 0 {
		return []T{}
	}
	seen := make(map[T]struct{}, len(b))
	for i := range b {
		seen[b[i]] = struct{}{}
	}

	var result []T
	for i := range a {
		if _, ok := seen[a[i]]; ok {
			result = append(result, a[i])
		}
	}
	if result == nil {
		return []T{}
	}
	return result
}

// Difference returns a new slice containing elements that are in slice a but not in slice b.
func Difference[T comparable](a, b []T) []T {
	if len(a) == 0 {
		return []T{}
	}
	if len(b) == 0 {
		result := make([]T, len(a))
		copy(result, a)
		return result
	}

	seen := make(map[T]struct{}, len(b))
	for i := range b {
		seen[b[i]] = struct{}{}
	}

	var result []T
	for i := range a {
		if _, ok := seen[a[i]]; !ok {
			result = append(result, a[i])
		}
	}
	if result == nil {
		return []T{}
	}
	return result
}

// Flatten converts a two-dimensional slice (slice of slices) into a single one-dimensional slice.
func Flatten[T any](slices [][]T) []T {
	if len(slices) == 0 {
		return []T{}
	}

	var totalLen int
	for i := range slices {
		totalLen += len(slices[i])
	}

	result := make([]T, 0, totalLen)
	for i := range slices {
		result = append(result, slices[i]...)
	}
	return result
}

// Reduce reduces a slice to a single value by applying the given function
// to each element, accumulating the result starting from the initial value.
//
// Example:
//
//	sum := Reduce([]int{1, 2, 3}, 0, func(acc, v int) int { return acc + v }) // 6
func Reduce[T any, U any](slice []T, initial U, fn func(U, T) U) U {
	result := initial
	for i := range slice {
		result = fn(result, slice[i])
	}
	return result
}

// FlatMap maps each element to a slice and flattens the result into a single slice.
//
// Example:
//
//	FlatMap([]string{"hello", "world"}, func(s string) []byte { return []byte(s) })
func FlatMap[T, U any](slice []T, fn func(T) []U) []U {
	if len(slice) == 0 {
		return []U{}
	}
	var result []U
	for i := range slice {
		result = append(result, fn(slice[i])...)
	}
	if result == nil {
		return []U{}
	}
	return result
}

// Last returns the last element of the slice and true, or zero value and false if the slice is empty.
//
// Example:
//
//	Last([]int{1, 2, 3}) // (3, true)
//	Last([]int{})        // (0, false)
func Last[T any](slice []T) (T, bool) {
	if len(slice) == 0 {
		var zero T
		return zero, false
	}
	return slice[len(slice)-1], true
}

// Compact removes zero-value elements from a slice.
// Returns a new slice without any elements that are equal to the zero value of type T.
//
// Example:
//
//	Compact([]string{"a", "", "b", "", "c"}) // ["a", "b", "c"]
//	Compact([]int{0, 1, 2, 0, 3})            // [1, 2, 3]
func Compact[T comparable](slice []T) []T {
	if len(slice) == 0 {
		return []T{}
	}
	var zero T
	result := make([]T, 0, len(slice))
	for i := range slice {
		if slice[i] != zero {
			result = append(result, slice[i])
		}
	}
	if len(result) == 0 {
		return []T{}
	}
	return result
}
