// Package maputil provides generic, production-ready utility functions
// for common map operations in Go.
// It leverages Go 1.18+ generics to provide type-safe manipulation without reflection.
// Zero external dependencies.
package maputil

// Keys returns a slice containing all the keys from a map.
// The order of the keys is not guaranteed.
//
// Example:
//
//	Keys(map[string]int{"a": 1, "b": 2}) // []string{"a", "b"} (order not guaranteed)
func Keys[K comparable, V any](m map[K]V) []K {
	keys := make([]K, 0, len(m))
	for k := range m {
		keys = append(keys, k)
	}
	return keys
}

// Values returns a slice containing all the values from a map.
// The order of the values is not guaranteed.
//
// Example:
//
//	Values(map[string]int{"a": 1, "b": 2}) // []int{1, 2} (order not guaranteed)
func Values[K comparable, V any](m map[K]V) []V {
	values := make([]V, 0, len(m))
	for _, v := range m {
		values = append(values, v)
	}
	return values
}

// Merge merges multiple maps into a new map.
// If there are duplicate keys, the value from the map provided later in the arguments will overwrite earlier ones.
//
// Example:
//
//	Merge(map[string]int{"a": 1}, map[string]int{"b": 2, "a": 3}) // map[string]int{"a": 3, "b": 2}
func Merge[K comparable, V any](maps ...map[K]V) map[K]V {
	result := make(map[K]V)
	for _, m := range maps {
		for k, v := range m {
			result[k] = v
		}
	}
	return result
}

// Pick returns a new map with only the specified keys.
// Keys that don't exist in the source map are silently ignored.
//
// Example:
//
//	Pick(map[string]int{"a": 1, "b": 2, "c": 3}, "a", "c") // map[string]int{"a": 1, "c": 3}
func Pick[K comparable, V any](m map[K]V, keys ...K) map[K]V {
	result := make(map[K]V, len(keys))
	for _, k := range keys {
		if v, ok := m[k]; ok {
			result[k] = v
		}
	}
	return result
}

// Omit returns a new map without the specified keys.
//
// Example:
//
//	Omit(map[string]int{"a": 1, "b": 2, "c": 3}, "b") // map[string]int{"a": 1, "c": 3}
func Omit[K comparable, V any](m map[K]V, keys ...K) map[K]V {
	excluded := make(map[K]struct{}, len(keys))
	for _, k := range keys {
		excluded[k] = struct{}{}
	}
	result := make(map[K]V, len(m))
	for k, v := range m {
		if _, skip := excluded[k]; !skip {
			result[k] = v
		}
	}
	return result
}

// Invert swaps keys and values in the map.
// If multiple keys have the same value, the last one wins (non-deterministic).
// Both K and V must be comparable.
//
// Example:
//
//	Invert(map[string]int{"a": 1, "b": 2}) // map[int]string{1: "a", 2: "b"}
func Invert[K, V comparable](m map[K]V) map[V]K {
	result := make(map[V]K, len(m))
	for k, v := range m {
		result[v] = k
	}
	return result
}

// Entry represents a key-value pair, used for constructing maps from slices of pairs.
type Entry[K comparable, V any] struct {
	Key   K
	Value V
}

// FromEntries creates a map from a slice of Entry key-value pairs.
// If duplicate keys exist, the last entry wins.
//
// Example:
//
//	FromEntries([]Entry[string, int]{{"a", 1}, {"b", 2}}) // map[string]int{"a": 1, "b": 2}
func FromEntries[K comparable, V any](entries []Entry[K, V]) map[K]V {
	result := make(map[K]V, len(entries))
	for _, e := range entries {
		result[e.Key] = e.Value
	}
	return result
}

// ContainsKey reports whether the map contains the given key.
//
// Example:
//
//	ContainsKey(map[string]int{"a": 1}, "a") // true
//	ContainsKey(map[string]int{"a": 1}, "b") // false
func ContainsKey[K comparable, V any](m map[K]V, key K) bool {
	_, ok := m[key]
	return ok
}

// ContainsValue reports whether the map contains the given value.
// Both K and V must be comparable.
//
// Example:
//
//	ContainsValue(map[string]int{"a": 1, "b": 2}, 2) // true
func ContainsValue[K, V comparable](m map[K]V, value V) bool {
	for _, v := range m {
		if v == value {
			return true
		}
	}
	return false
}

// Filter returns a new map containing only entries that satisfy the predicate.
//
// Example:
//
//	Filter(map[string]int{"a": 1, "b": 2, "c": 3}, func(k string, v int) bool { return v > 1 })
//	// map[string]int{"b": 2, "c": 3}
func Filter[K comparable, V any](m map[K]V, fn func(K, V) bool) map[K]V {
	result := make(map[K]V)
	for k, v := range m {
		if fn(k, v) {
			result[k] = v
		}
	}
	return result
}
