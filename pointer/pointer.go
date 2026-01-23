// Package pointer provides generic helpers to easily get pointers to literal values.
// This is frequently needed for initializing structs with pointer fields (e.g. SQL nullables, JSON optionals).
package pointer

import "time"

// Of returns a pointer to the given value.
// It uses Go 1.18+ generics [T any] to work with any type (int, string, struct, etc.)
//
// Example:
//
//	active := pointer.Of(true)
//	count := pointer.Of(10)
func Of[T any](v T) *T {
	// Taking the address of 'v' works because 'v' is a function argument,
	// so it's a local variable. Go's escape analysis will move it to the heap.
	return &v
}

// String returns a pointer to the given string.
// (Legacy helper, prefer Of)
func String(s string) *string {
	return &s
}

// Int returns a pointer to the given int.
// (Legacy helper, prefer Of)
func Int(i int) *int {
	return &i
}

// Bool returns a pointer to the given bool.
// (Legacy helper, prefer Of)
func Bool(b bool) *bool {
	return &b
}

// Time returns a pointer to the given time.Time.
// (Legacy helper, prefer Of)
func Time(t time.Time) *time.Time {
	return &t
}
