// Package format provides safe, reusable formatting utilities for time, currency, phone numbers, etc.
//
// Key principles:
//   - All dates in database → UTC
//   - All dates shown to users → WIB (UTC+7)
//   - Zero dependencies (standard library only)
package format

import (
	"fmt"
	"strconv"
	"strings"
	"time"
)

// =============================================================================
// TIMEZONE DEFINITIONS
// =============================================================================

var (
	// UTC is the standard time location for storage and internal logic.
	UTC = time.UTC

	// WIB (Waktu Indonesia Barat) is UTC+7. Used for display to users in Western Indonesia.
	// It is a fixed zone with no daylight saving time.
	WIB = time.FixedZone("Asia/Jakarta", 7*60*60)

	// Jakarta is an alias for WIB.
	Jakarta = WIB

	// Bangkok is UTC+7, same offset as WIB but distinct location.
	Bangkok = time.FixedZone("Asia/Bangkok", 7*60*60)
)

// =============================================================================
// COMMON DATE/TIME LAYOUTS
// =============================================================================

const (
	// LayoutISO is the standard RFC3339 format (e.g., "2006-01-02T15:04:05Z").
	LayoutISO = time.RFC3339

	// LayoutRFC3339WIB is RFC3339 with a fixed +07:00 offset.
	LayoutRFC3339WIB = "2006-01-02T15:04:05+07:00"

	// LayoutDateTimeSec is a SQL-friendly format "YYYY-MM-DD HH:MM:SS".
	LayoutDateTimeSec = "2006-01-02 15:04:05"

	// LayoutDate is just the date "YYYY-MM-DD".
	LayoutDate = "2006-01-02"
)

// =============================================================================
// NOW HELPERS
// =============================================================================

// NowUTC returns the current time in UTC.
// Use this for database timestamps and internal logic.
func NowUTC() time.Time { return time.Now().UTC() }

// NowWIB returns the current time in WIB (UTC+7).
// Use this for display purposes or business logic specific to Indonesia time.
func NowWIB() time.Time { return time.Now().In(WIB) }

// Now returns the current time in UTC (default safe choice).
func Now() time.Time { return NowUTC() }

// =============================================================================
// CONVERSIONS
// =============================================================================

// ToWIB converts a time to WIB (UTC+7).
func ToWIB(t time.Time) time.Time { return t.In(WIB) }

// ToUTC converts a time to UTC.
func ToUTC(t time.Time) time.Time { return t.UTC() }

// =============================================================================
// FORMATTERS
// =============================================================================

// FormatWIB formats a time in WIB using the specified layout.
// Returns an empty string if the time is zero.
func FormatWIB(t time.Time, layout string) string {
	if t.IsZero() {
		return ""
	}
	return t.In(WIB).Format(layout)
}

// FormatUTC formats a time in UTC using the specified layout.
// Returns an empty string if the time is zero.
func FormatUTC(t time.Time, layout string) string {
	if t.IsZero() {
		return ""
	}
	return t.UTC().Format(layout)
}

// =============================================================================
// PARSERS
// =============================================================================

// ParseRFC3339Safe parses RFC3339 safely.
// Empty or zero date returns zero time without error.
func ParseRFC3339Safe(s string) (time.Time, error) {
	s = strings.TrimSpace(s)
	if s == "" || strings.HasPrefix(s, "0001-01-01") {
		return time.Time{}, nil
	}
	return time.Parse(time.RFC3339, s)
}

// =============================================================================
// STRING → TIME (STRICT)
// =============================================================================

// StringToDateTimeSecWIB parses "YYYY-MM-DD HH:MM:SS" as WIB.
// Returns an error if parsing fails or input is empty.
func StringToDateTimeSecWIB(s string) (time.Time, error) {
	if s == "" {
		return time.Time{}, fmt.Errorf("empty datetime")
	}
	return time.ParseInLocation(LayoutDateTimeSec, s, WIB)
}

// StringToDateTimeSecUTC parses "YYYY-MM-DD HH:MM:SS" as UTC.
// Returns an error if parsing fails or input is empty.
func StringToDateTimeSecUTC(s string) (time.Time, error) {
	if s == "" {
		return time.Time{}, fmt.Errorf("empty datetime")
	}
	return time.ParseInLocation(LayoutDateTimeSec, s, UTC)
}

// StringToDateWIB parses "YYYY-MM-DD" as WIB.
// Returns an error if parsing fails or input is empty.
func StringToDateWIB(s string) (time.Time, error) {
	if s == "" {
		return time.Time{}, fmt.Errorf("empty date")
	}
	return time.ParseInLocation(LayoutDate, s, WIB)
}

// StringToDateUTC parses "YYYY-MM-DD" as UTC.
// Returns an error if parsing fails or input is empty.
func StringToDateUTC(s string) (time.Time, error) {
	if s == "" {
		return time.Time{}, fmt.Errorf("empty date")
	}
	return time.ParseInLocation(LayoutDate, s, UTC)
}

// =============================================================================
// STRING → TIME (FORGIVING)
// =============================================================================

// StringToDateTimeSecWIBOrZero parses "YYYY-MM-DD HH:MM:SS" as WIB.
// Returns zero time if parsing fails or input is empty.
func StringToDateTimeSecWIBOrZero(s string) time.Time {
	t, _ := StringToDateTimeSecWIB(s)
	return t
}

// StringToDateTimeSecUTCOrZero parses "YYYY-MM-DD HH:MM:SS" as UTC.
// Returns zero time if parsing fails or input is empty.
func StringToDateTimeSecUTCOrZero(s string) time.Time {
	t, _ := StringToDateTimeSecUTC(s)
	return t
}

// StringToDateWIBOrZero parses "YYYY-MM-DD" as WIB.
// Returns zero time if parsing fails or input is empty.
func StringToDateWIBOrZero(s string) time.Time {
	t, _ := StringToDateWIB(s)
	return t
}

// StringToDateUTCOrZero parses "YYYY-MM-DD" as UTC.
// Returns zero time if parsing fails or input is empty.
func StringToDateUTCOrZero(s string) time.Time {
	t, _ := StringToDateUTC(s)
	return t
}

// string to unix timestamp
func StringToUnix(tsString string) (time.Time, error) {
	sec, err := strconv.ParseInt(tsString, 10, 64)
	if err != nil {
		return time.Time{}, fmt.Errorf("invalid number: %w", err)
	}

	return time.Unix(sec, 0).UTC(), nil
}

// StringToUnixOrZero parses a string to a time.Time value.
// It expects the input string to be a Unix timestamp.
// It returns zero time if the input string is not a valid number.
func StringToUnixOrZero(tsString string) time.Time {
	t, _ := StringToUnix(tsString)
	return t
}

// =============================================================================
// TIME → STRING
// =============================================================================

// ToDateTimeSecString formats the time as "YYYY-MM-DD HH:MM:SS".
// Returns empty string if time is zero.
func ToDateTimeSecString(t time.Time) string {
	if t.IsZero() {
		return ""
	}
	return t.Format(LayoutDateTimeSec)
}

// ToDateString formats the time as "YYYY-MM-DD".
// Returns empty string if time is zero.
func ToDateString(t time.Time) string {
	if t.IsZero() {
		return ""
	}
	return t.Format(LayoutDate)
}

// Timestamp formats a time.Time value as a Unix timestamp string.
// Returns empty string if input is zero time.
//
// Example:
//
//	Timestamp(time.Now()) // "1633072800"
//	Timestamp(time.Time{}) // ""
func Timestamp(t time.Time) string {
	if t.IsZero() {
		return ""
	}
	return fmt.Sprintf("%d", t.Unix())
}
