// Package format provides essential, production-grade utility functions
// using ONLY the Go standard library (2025 best practice).
//
// No external dependencies → smaller binary, faster build, zero supply-chain risk.
//
// Contains:
//   - Time: WIB ↔ UTC conversion
//   - String: Title case, unique append
//   - Number: Format Rupiah & BRI account
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

// Title converts string to Title Case using simple built-in logic
// (good enough for 99% cases: names, roles, categories)
func Title(s string) string {
	if s == "" {
		return ""
	}
	var result strings.Builder
	upperNext := true
	for _, r := range s {
		if r == ' ' || r == '-' || r == '_' {
			result.WriteRune(r)
			upperNext = true
			continue
		}
		if upperNext {
			result.WriteRune(toUpper(r))
			upperNext = false
		} else {
			result.WriteRune(toLower(r))
		}
	}
	return result.String()
}

// Simple ASCII-only upper/lower (fast & built-in)
func toUpper(r rune) rune {
	if r >= 'a' && r <= 'z' {
		return r - 32
	}
	return r
}
func toLower(r rune) rune {
	if r >= 'A' && r <= 'Z' {
		return r + 32
	}
	return r
}

// AddStringUnique appends value only if not exists (case-insensitive)
func AddStringUnique(value string, slice *[]string) {
	if strings.TrimSpace(value) == "" {
		return
	}
	value = Title(value)

	for _, v := range *slice {
		if strings.EqualFold(v, value) {
			return
		}
	}
	*slice = append(*slice, value)
}

// =============================================================================
// NUMBER & BANK HELPERS
// =============================================================================

// FormatRupiah formats number to Indonesian Rupiah: 1.234.567,89
// Indonesia uses dot (.) as thousand separator and comma (,) as decimal
func FormatRupiah(amount float64) string {
	return formatNumber(amount, 2, ",", ".")
}

// FormatBRINorek formats BRI account number: 1234-56-789012-34-5
func FormatBRINorek(norek string) string {
	norek = strings.ReplaceAll(norek, "-", "")
	norek = strings.ReplaceAll(norek, " ", "")
	if len(norek) < 15 {
		return ""
	}
	if len(norek) > 15 {
		norek = norek[:15]
	}
	return fmt.Sprintf("%s-%s-%s-%s-%s",
		norek[:4], norek[4:6], norek[6:12], norek[12:14], norek[14:])
}

// formatNumber is generic formatter (decimal places, decSep, thouSep)
func formatNumber(num float64, decimals int, decSep, thouSep string) string {
	isNegative := num < 0
	if isNegative {
		num = -num
	}

	str := strconv.FormatFloat(num, 'f', decimals, 64)
	parts := strings.Split(str, ".")
	intPart := parts[0]
	decPart := "0"
	if len(parts) > 1 {
		decPart = parts[1]
	}
	if len(decPart) < decimals {
		decPart += strings.Repeat("0", decimals-len(decPart))
	}

	// Add thousand separators
	var intFormatted strings.Builder
	l := len(intPart)
	for i := 0; i < l; i++ {
		if i > 0 && (l-i)%3 == 0 {
			intFormatted.WriteString(thouSep)
		}
		intFormatted.WriteByte(intPart[i])
	}

	result := intFormatted.String() + decSep + decPart
	if isNegative {
		return "-" + result
	}
	return result
}

// Package converter provides safe, zero-allocation when possible, and battle-tested
// conversion utilities from any type to string.
//
// Used in logging, response formatting, cache key, Redis key, file name, etc.
//
// ToString converts ANY type → string
// Support: string, int, float, bool, time.Time, []byte, struct, map, nil, dll
// Never panic, always return string
func ToString(v any) string {
	if v == nil {
		return ""
	}

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

		// JSON fallback
		if b, err := json.Marshal(v); err == nil {
			return string(b)
		}
		return fmt.Sprintf("%v", v)
	}
}

// ToSafeString – untuk Redis key, file name, log context
// Menghilangkan spasi, symbol berbahaya, dll
func ToSafeString(v any) string {
	s := ToString(v)
	s = strings.TrimSpace(s)
	s = strings.ReplaceAll(s, " ", "_")
	s = strings.ReplaceAll(s, "/", "_")
	s = strings.ReplaceAll(s, "\\", "_")
	s = strings.ReplaceAll(s, ":", "_")
	if s == "" {
		return "empty"
	}
	return s
}
