// Package format provides essential, production-grade utility functions
// using ONLY the Go standard library (best practice).
//
// No external dependencies → smaller binary, faster build, zero supply-chain risk.
//
// Contains:
//   - String helpers: Title case, unique append
//   - Number formatting: Currency
//   - Bank formatting: Account number (specific format)
//   - Safe type-to-string conversion for logging, cache keys, filenames, etc.
package format

import (
	"encoding/json"
	"fmt"
	"reflect"
	"strconv"
	"strings"
	"time"
)

// =============================================================================
// STRING HELPERS
// =============================================================================

// Title converts a string to Title Case using simple ASCII-based logic.
// It uppercases the letter following spaces, hyphens, or underscores.
// Suitable for names, roles, categories, and most UI text (99% of cases).
// Returns empty string if input is empty.
//
// Example:
//
//	Title("john doe-jr") // "John Doe-Jr"
func Title(s string) string {
	if s == "" {
		return ""
	}
	var result strings.Builder
	// Flag to track if the next character should be uppercased
	upperNext := true
	for _, r := range s {
		// Check for delimiters
		if r == ' ' || r == '-' || r == '_' {
			result.WriteRune(r)
			upperNext = true
			continue
		}
		// Uppercase if flag is set
		if upperNext {
			result.WriteRune(toUpper(r))
			upperNext = false
		} else {
			// Otherwise lowercase
			result.WriteRune(toLower(r))
		}
	}
	return result.String()
}

// toUpper converts an ASCII lowercase letter to uppercase.
// Non-letter runes are returned unchanged. Fast and zero-allocation.
func toUpper(r rune) rune {
	if r >= 'a' && r <= 'z' {
		return r - 32 // ASCII logic
	}
	return r
}

// toLower converts an ASCII uppercase letter to lowercase.
// Non-letter runes are returned unchanged. Fast and zero-allocation.
func toLower(r rune) rune {
	if r >= 'A' && r <= 'Z' {
		return r + 32 // ASCII logic
	}
	return r
}

// AddStringUnique appends a value to a string slice only if it does not already exist
// (case-insensitive comparison). The value is normalized with Title() before insertion.
// Empty (after trim) values are ignored. The slice is modified in place.
//
// Example:
//
//	items := []string{"Admin", "User"}
//	AddStringUnique("admin", &items)      // no change
//	AddStringUnique("moderator", &items) // items becomes ["Admin", "User", "Moderator"]
func AddStringUnique(value string, slice *[]string) {
	// Guard clause for empty input
	if strings.TrimSpace(value) == "" {
		return
	}
	// Normalize value
	value = Title(value)

	// Check for duplicates
	for _, v := range *slice {
		if strings.EqualFold(v, value) {
			return // Already exists
		}
	}
	// Append if unique
	*slice = append(*slice, value)
}

// =============================================================================
// NUMBER & BANK HELPERS
// =============================================================================

// Rupiah formats a float64 amount as a currency string (e.g. 150.000,00).
// Uses dot (.) as thousand separator and comma (,) as decimal separator.
// Always shows exactly 2 decimal places.
//
// Example:
//
//	Rupiah(1234567.89) // "1.234.567,89"
//	Rupiah(-5000)      // "-5.000,00"
func Rupiah(amount float64) string {
	return formatNumber(amount, 2, ",", ".")
}

// BRINorek formats an account number into the standard pattern: XXXX-XX-XXXXXX-XX-X
// All existing hyphens and spaces are removed first.
// If input is shorter than 15 digits, returns empty string.
// If longer, only the first 15 digits are used.
//
// Example:
//
//	BRINorek("123456789012345") // "1234-56-789012-34-5"
func BRINorek(norek string) string {
	// Clean input
	norek = strings.ReplaceAll(norek, "-", "")
	norek = strings.ReplaceAll(norek, " ", "")
	// Validate length
	if len(norek) < 15 {
		return ""
	}
	// Truncate if too long
	if len(norek) > 15 {
		norek = norek[:15]
	}
	// Format with hyphens
	return fmt.Sprintf("%s-%s-%s-%s-%s",
		norek[:4], norek[4:6], norek[6:12], norek[12:14], norek[14:])
}

// formatNumber is a generic number formatter used internally by Rupiah.
// Formats num with given decimal places, decimal separator, and thousand separator.
func formatNumber(num float64, decimals int, decSep, thouSep string) string {
	// Handle negative numbers
	isNegative := num < 0
	if isNegative {
		num = -num
	}

	// Convert float to string with fixed precision
	str := strconv.FormatFloat(num, 'f', decimals, 64)
	parts := strings.Split(str, ".")
	intPart := parts[0]
	decPart := "0"
	if len(parts) > 1 {
		decPart = parts[1]
	}
	// Pad decimal part if needed
	if len(decPart) < decimals {
		decPart += strings.Repeat("0", decimals-len(decPart))
	}

	// Add thousand separators
	var intFormatted strings.Builder
	l := len(intPart)
	for i := 0; i < l; i++ {
		// Add separator every 3 digits (except at start)
		if i > 0 && (l-i)%3 == 0 {
			intFormatted.WriteString(thouSep)
		}
		intFormatted.WriteByte(intPart[i])
	}

	// Assemble final string
	result := intFormatted.String() + decSep + decPart
	if isNegative {
		return "-" + result
	}
	return result
}

// =============================================================================
// TYPE CONVERSION UTILITIES
// =============================================================================

// ToString safely converts any value to its string representation.
// Never panics. Supports built-in types, time.Time, fmt.Stringer, slices, maps, structs, etc.
// Used for logging, JSON responses, cache keys, Redis keys, filenames, etc.
//
// Priority order:
//   - string / []byte → direct conversion
//   - numeric types → decimal formatting
//   - bool → "true"/"false"
//   - time.Time → RFC3339 (empty if zero)
//   - fmt.Stringer → .String()
//   - nil / nil pointer → ""
//   - other → JSON marshal → fallback to fmt.Sprintf("%v")
//
// Example:
//
//	ToString(123)                     // "123"
//	ToString(time.Now())              // "2006-01-02T15:04:05Z07:00"
//	ToString(map[string]int{"a":1})   // `{"a":1}`
func ToString(v any) string {
	if v == nil {
		return ""
	}

	// Type switch for optimal performance on common types
	switch value := v.(type) {
	case string:
		return value
	case []byte:
		return string(value)
	case int, int8, int16, int32, int64:
		return fmt.Sprintf("%d", value)
	case uint, uint8, uint16, uint32, uint64:
		return fmt.Sprintf("%d", value)
	case float32:
		return strconv.FormatFloat(float64(value), 'f', -1, 32)
	case float64:
		return strconv.FormatFloat(value, 'f', -1, 64)
	case bool:
		return strconv.FormatBool(value)
	case time.Time:
		if value.IsZero() {
			return ""
		}
		return value.Format(time.RFC3339)
	case fmt.Stringer:
		return value.String()
	default:
		// Handle nil pointer/interface
		if reflect.ValueOf(v).Kind() == reflect.Ptr && reflect.ValueOf(v).IsNil() {
			return ""
		}

		// JSON fallback for complex types
		if b, err := json.Marshal(v); err == nil {
			return string(b)
		}
		// Ultimate fallback
		return fmt.Sprintf("%v", v)
	}
}

func ToInt64(v any) int64 {
	switch value := v.(type) {
	case int:
		return int64(value)
	case int8:
		return int64(value)
	case int16:
		return int64(value)
	case int32:
		return int64(value)
	case int64:
		return value
	case uint:
		return int64(value)
	case uint8:
		return int64(value)
	case uint16:
		return int64(value)
	case uint32:
		return int64(value)
	case uint64:
		return int64(value)
	case float32:
		return int64(value)
	case float64:
		return int64(value)
	case string:
		val, _ := strconv.ParseInt(value, 10, 64)
		return val
	default:
		return 0
	}
}

// ToSafeString converts any value to string and sanitizes it for use in
// filenames, Redis keys, log context, URLs, etc.
// Replaces spaces and dangerous characters (/ \ :) with underscores.
// Returns "empty" if the result is blank after sanitization.
//
// Example:
//
//	ToSafeString("user/name:123") // "user_name_123"
//	ToSafeString("")              // "empty"
func ToSafeString(v any) string {
	// Convert to string first
	s := ToString(v)
	// Trim whitespace
	s = strings.TrimSpace(s)
	// Replace unsafe characters
	s = strings.ReplaceAll(s, " ", "_")
	s = strings.ReplaceAll(s, "/", "_")
	s = strings.ReplaceAll(s, "\\", "_")
	s = strings.ReplaceAll(s, ":", "_")
	// Handle empty result
	if s == "" {
		return "empty"
	}
	return s
}
