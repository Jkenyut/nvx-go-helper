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
	"fmt"
	"reflect"
	"regexp"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/bytedance/sonic"
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
	result.Grow(len(s))
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
	case int:
		return strconv.FormatInt(int64(value), 10)
	case int8:
		return strconv.FormatInt(int64(value), 10)
	case int16:
		return strconv.FormatInt(int64(value), 10)
	case int32:
		return strconv.FormatInt(int64(value), 10)
	case int64:
		return strconv.FormatInt(value, 10)
	case uint:
		return strconv.FormatUint(uint64(value), 10)
	case uint8:
		return strconv.FormatUint(uint64(value), 10)
	case uint16:
		return strconv.FormatUint(uint64(value), 10)
	case uint32:
		return strconv.FormatUint(uint64(value), 10)
	case uint64:
		return strconv.FormatUint(value, 10)
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
		// Handle nil pointer/interface safely.
		// reflect.ValueOf(v).IsNil() panics on non-nillable kinds, so we must check first.
		rv := reflect.ValueOf(v)
		switch rv.Kind() {
		case reflect.Pointer, reflect.Interface, reflect.Map, reflect.Slice, reflect.Chan, reflect.Func:
			if rv.IsNil() {
				return ""
			}
		case reflect.Invalid:
			return ""
		}

		// JSON fallback for complex types
		if b, err := sonic.Marshal(v); err == nil {
			return string(b)
		}
		// Ultimate fallback
		return fmt.Sprintf("%v", v)
	}
}

// ToInt64 attempts to convert any compatible value to int64.
// Supports all integer types, floats (truncated), and strings (parsed).
// Returns 0 if conversion fails or type is unsupported.
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
	// Handle empty result
	if s == "" {
		return "empty"
	}
	// Replace unsafe characters
	return strings.Map(func(r rune) rune {
		// Remove null-byte entirely (prevents path traversal vulnerability)
		if r == 0 {
			return -1
		}

		// Replace spaces, tabs, newlines, and other dangerous characters with '_'
		if r == ' ' || r == '\n' || r == '\r' || r == '\t' || r == '/' || r == '\\' || r == ':' {
			return '_'
		}

		return r
	}, s)
}

// =============================================================================
// STRING MASKING & FORMATTING
// =============================================================================

// MaskEmail masks an email address for display, preserving only the first
// character of the local part and the full domain.
// Returns empty string if input is empty or invalid.
//
// Example:
//
//	MaskEmail("john@example.com")      // "j***@example.com"
//	MaskEmail("ab@example.com")        // "a***@example.com"
func MaskEmail(email string) string {
	if email == "" {
		return ""
	}
	parts := strings.SplitN(email, "@", 2)
	if len(parts) != 2 || parts[0] == "" || parts[1] == "" {
		return ""
	}
	local := parts[0]
	// Show first character, mask the rest
	masked := string(local[0]) + "***"
	return masked + "@" + parts[1]
}

// MaskPhone masks a phone number for display, showing the prefix and last 4 digits.
// Designed for Indonesian phone numbers but works with any format.
// Returns empty string if input is empty or too short.
//
// Example:
//
//	MaskPhone("+6281234567890")  // "+62812****7890"
//	MaskPhone("081234567890")   // "08123****7890"
func MaskPhone(phone string) string {
	if len(phone) < 8 {
		return ""
	}
	// Show first (len-4)/2+1 chars, mask middle, show last 4
	visiblePrefix := len(phone) - 4
	if visiblePrefix > 5 {
		visiblePrefix = 5
	}
	masked := phone[:visiblePrefix] + strings.Repeat("*", len(phone)-visiblePrefix-4) + phone[len(phone)-4:]
	return masked
}

// Truncate truncates a string to maxLen characters and appends "..." if truncated.
// If the string is shorter than or equal to maxLen, it is returned unchanged.
// maxLen must be >= 3 (to fit the ellipsis); otherwise returns original string.
//
// Example:
//
//	Truncate("Hello, World!", 10) // "Hello, ..."
//	Truncate("Hi", 10)           // "Hi"
func Truncate(s string, maxLen int) string {
	if maxLen < 3 || len(s) <= maxLen {
		return s
	}
	return s[:maxLen-3] + "..."
}

// PadLeft pads a string on the left side to reach the desired length.
// If the string is already longer than or equal to length, it is returned unchanged.
//
// Example:
//
//	PadLeft("42", 5, '0')   // "00042"
//	PadLeft("hello", 3, ' ') // "hello"
func PadLeft(s string, length int, pad rune) string {
	if len(s) >= length {
		return s
	}
	return strings.Repeat(string(pad), length-len(s)) + s
}

// PadRight pads a string on the right side to reach the desired length.
// If the string is already longer than or equal to length, it is returned unchanged.
//
// Example:
//
//	PadRight("42", 5, '0')   // "42000"
//	PadRight("hello", 3, ' ') // "hello"
func PadRight(s string, length int, pad rune) string {
	if len(s) >= length {
		return s
	}
	return s + strings.Repeat(string(pad), length-len(s))
}

// Dollar formats a float64 amount as a USD currency string (e.g. 1,234,567.89).
// Uses comma (,) as thousand separator and dot (.) as decimal separator.
// Always shows exactly 2 decimal places.
//
// Example:
//
//	Dollar(1234567.89) // "1,234,567.89"
//	Dollar(-5000)      // "-5,000.00"
func Dollar(amount float64) string {
	return formatNumber(amount, 2, ".", ",")
}

var maskRegexCache sync.Map

// MaskAfterKeywords masks values that appear immediately after specific keywords.
// It is safe for Plain Text, JSON, and template formats (preserves surrounding quotes).
// Supports double-quoted, single-quoted, backtick-quoted, and unquoted values.
// Handles escaped quotes inside quoted strings (e.g. `\"`).
// Uses sync.Map to cache compiled regex for high-performance logging.
//
// Example:
//
//	MaskAfterKeywords(`{"password": "secret"}`, []string{"password"}, "*") // `{"password": "******"}`
//	MaskAfterKeywords("token=abc123", []string{"token"}, "*")               // "token=******"
//	MaskAfterKeywords("key=`mysecret`", []string{"key"}, "*")               // "key=`********`"
func MaskAfterKeywords(text string, keywords []string, maskChar string) string {
	if len(keywords) == 0 {
		return text
	}

	// Use joined keywords as cache key to prevent recompiling regex on every call
	cacheKey := strings.Join(keywords, "|")

	var re *regexp.Regexp
	if cached, ok := maskRegexCache.Load(cacheKey); ok {
		re = cached.(*regexp.Regexp)
	} else {
		var escaped []string
		for _, k := range keywords {
			escaped = append(escaped, regexp.QuoteMeta(k))
		}
		// Regex Pattern:
		// Group 1: Keyword (including optional surrounding quotes)
		// Group 2: Separator (spaces, =, or :)
		// Group 3: Value in double quotes — supports escaped \" inside
		// Group 4: Value in single quotes — supports escaped \' inside
		// Group 5: Value in backticks
		// Group 6: Unquoted value (plain text, JSON numbers/booleans)
		// Unquoted exclusion includes & to correctly split key=val&key=val pairs.
		pattern := fmt.Sprintf(`(?i)(["'`+"`"+`]?\b(?:%s)\b["'`+"`"+`]?)([\s=:]+)(?:"((?:[^"\\]|\\.)*)"|'((?:[^'\\]|\\.)*)'|`+"`"+`([^`+"`"+`]*)`+"`"+`|([^\s,;{}&]+))`, strings.Join(escaped, "|"))
		re = regexp.MustCompile(pattern)
		maskRegexCache.Store(cacheKey, re)
	}

	return re.ReplaceAllStringFunc(text, func(match string) string {
		parts := re.FindStringSubmatch(match)
		idx := re.FindStringSubmatchIndex(match)

		kw := parts[1]  // Keyword
		sep := parts[2] // Separator

		// Use submatch indices (not empty-string check) to detect which group matched.
		// parts[n] != "" fails for empty quoted values like ""; idx[n*2] >= 0 is correct.
		switch {
		case idx[6] >= 0: // Double-quoted (JSON / standard string) — group 3
			return kw + sep + `"` + strings.Repeat(maskChar, len(parts[3])) + `"`
		case idx[8] >= 0: // Single-quoted — group 4
			return kw + sep + `'` + strings.Repeat(maskChar, len(parts[4])) + `'`
		case idx[10] >= 0: // Backtick-quoted — group 5
			return kw + sep + "`" + strings.Repeat(maskChar, len(parts[5])) + "`"
		default: // Unquoted (plain text, JSON number/boolean, etc) — group 6
			return kw + sep + strings.Repeat(maskChar, len(parts[6]))
		}
	})
}
