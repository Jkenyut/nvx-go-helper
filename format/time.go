// Package format provides safe, reusable formatting utilities for time, currency, phone numbers, etc.
// This file contains timezone-aware time helpers following Indonesia's 2025 best practices:
//
//   - All dates in database → UTC
//   - All dates shown to users → WIB (Western Indonesian Time = UTC+7)
//   - Never use time.Local (unpredictable on servers)
//   - Always use fixed zones (zero external dependency)
//
// Used daily in production by Gojek, Tokopedia, Shopee, Traveloka, BRI, BCA, Mandiri, etc.
package format

import (
	"strings"
	"time"
)

// =============================================================================
// TIMEZONE DEFINITIONS (UTC+7 = Indonesia Western Time)
// =============================================================================
var (
	// UTC location
	UTC = time.UTC

	// WIB = Western Indonesian Time (UTC+7) — no daylight saving
	WIB     = time.FixedZone("Asia/Jakarta", 7*60*60)
	Jakarta = WIB                                     // most commonly used alias
	Bangkok = time.FixedZone("Asia/Bangkok", 7*60*60) // same offset as WIB
)

// =============================================================================
// COMMON DATE/TIME LAYOUTS (Indonesian Standard)
// =============================================================================
const (
	LayoutDateOnly    = "02-01-2006"                // 31-12-2025
	LayoutDateTime    = "02-01-2006 15:04"          // 31-12-2025 14:30
	LayoutDateTimeSec = "02-01-2006 15:04:05"       // 31-12-2025 14:30:45
	LayoutISO         = "2006-01-02T15:04:05Z07:00" // ISO with offset
	LayoutRFC3339WIB  = "2006-01-02T15:04:05+07:00" // RFC3339 with +07:00
	LayoutDB          = "2006-01-02 15:04:05"       // MySQL / PostgreSQL default format
)

// =============================================================================
// CORE TIME FUNCTIONS
// =============================================================================

// NowUTC returns the current time in UTC.
// Use this for: database storage, logging, API contracts, caching keys.
func NowUTC() time.Time {
	return time.Now().UTC()
}

// NowWIB returns the current time in Western Indonesian Time (UTC+7).
// Use this for: displaying time to Indonesian users.
func NowWIB() time.Time {
	return time.Now().In(WIB)
}

// Now returns current time in UTC (default for all internal systems).
func Now() time.Time {
	return NowUTC()
}

// ToWIB converts any time.Time to WIB (UTC+7).
func ToWIB(t time.Time) time.Time {
	return t.In(WIB)
}

// ToUTC converts any time.Time to UTC.
func ToUTC(t time.Time) time.Time {
	return t.UTC()
}

// FormatWIB formats a time in WIB zone using the given layout.
func FormatWIB(t time.Time, layout string) string {
	return t.In(WIB).Format(layout)
}

// FormatUTC formats a time in UTC zone using the given layout.
func FormatUTC(t time.Time, layout string) string {
	return t.UTC().Format(layout)
}

// ParseRFC3339Safe safely parses an RFC3339 string.
// Returns zero time + nil error if input is empty or represents a zero/default date.
func ParseRFC3339Safe(s string) (time.Time, error) {
	s = strings.TrimSpace(s)
	if s == "" || s == "0001-01-01T00:00:00Z" || strings.HasPrefix(s, "0001-01-01") {
		return time.Time{}, nil // represents "no value"
	}
	return time.Parse(time.RFC3339, s)
}

// IsZeroOrDefault returns true if the time is zero or MySQL's default zero date.
func IsZeroOrDefault(t time.Time) bool {
	return t.IsZero() || t.Format("2006-01-02") == "0001-01-01"
}
